package util

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Resource interface; all resources managed by the pool must implement this interface
type Resource interface {
	// Close closes the resource
	Close() error
	// IsValid checks whether the resource is valid
	IsValid() bool
}

// ResourceFactory interface for creating and validating resources
type ResourceFactory interface {
	// Create creates a new resource instance
	Create() (Resource, error)
	// Validate checks whether the resource is valid (optional; if false, resource will be destroyed)
	Validate(resource Resource) bool
	// Reset resets the resource state (optional, for cleanup before reuse)
	Reset(resource Resource) error
}

// PoolConfig resource pool configuration
type PoolConfig struct {
	// MaxSize maximum number of resources
	MaxSize int
	// MinSize minimum number of resources (pre-created)
	MinSize int
	// MaxIdle maximum number of idle resources
	MaxIdle int
	// AcquireTimeout timeout for acquiring a resource
	AcquireTimeout time.Duration
	// IdleTimeout resource idle timeout duration
	IdleTimeout time.Duration
	// ValidateOnBorrow whether to validate the resource on borrow
	ValidateOnBorrow bool
	// ValidateOnReturn whether to validate the resource on return
	ValidateOnReturn bool
}

// DefaultConfig returns the default pool configuration
func DefaultConfig() *PoolConfig {
	return &PoolConfig{
		MaxSize:          1000,
		MinSize:          1,
		MaxIdle:          5,
		AcquireTimeout:   30 * time.Second,
		IdleTimeout:      5 * time.Minute,
		ValidateOnBorrow: true,
		ValidateOnReturn: false,
	}
}

// pooledResource wraps a pool-managed resource
type pooledResource struct {
	resource   Resource
	createTime time.Time
	lastUsed   time.Time
	inUse      bool
}

// ResourcePool generic resource pool
type ResourcePool struct {
	config  *PoolConfig
	factory ResourceFactory

	// available resource queue
	available chan *pooledResource
	// map of all resources (both in-use and available)
	resources map[Resource]*pooledResource
	// read-write lock
	mu sync.RWMutex
	// closed flag
	closed bool
	// cancellation context
	ctx    context.Context
	cancel context.CancelFunc
	// cleanup goroutine wait group
	cleanupWg sync.WaitGroup
}

// NewResourcePool creates a new resource pool
func NewResourcePool(config *PoolConfig, factory ResourceFactory) (*ResourcePool, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if factory == nil {
		return nil, errors.New("factory cannot be nil")
	}
	if config.MaxSize <= 0 {
		return nil, errors.New("max size must be positive")
	}
	if config.MinSize < 0 {
		return nil, errors.New("min size cannot be negative")
	}
	if config.MinSize > config.MaxSize {
		return nil, errors.New("min size cannot be greater than max size")
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ResourcePool{
		config:    config,
		factory:   factory,
		available: make(chan *pooledResource, config.MaxSize),
		resources: make(map[Resource]*pooledResource),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Pre-create the minimum number of resources
	if err := pool.preCreateResources(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to pre-create resources: %w", err)
	}

	// Start the cleanup goroutine
	pool.startCleanupRoutine()

	return pool, nil
}

// preCreateResources pre-creates resources up to MinSize
func (p *ResourcePool) preCreateResources() error {
	for i := 0; i < p.config.MinSize; i++ {
		resource, err := p.factory.Create()
		if err != nil {
			return fmt.Errorf("failed to create resource %d: %w", i, err)
		}

		pooled := &pooledResource{
			resource:   resource,
			createTime: time.Now(),
			lastUsed:   time.Now(),
			inUse:      false,
		}

		p.resources[resource] = pooled
		p.available <- pooled
	}
	return nil
}

// Acquire acquires a resource from the pool
func (p *ResourcePool) Acquire() (Resource, error) {
	return p.AcquireWithTimeout(p.config.AcquireTimeout)
}

// AcquireWithTimeout acquires a resource within the specified timeout duration
func (p *ResourcePool) AcquireWithTimeout(timeout time.Duration) (Resource, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, errors.New("pool is closed")
	}
	p.mu.RUnlock()

	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire timeout after %v", timeout)
		case pooled := <-p.available:
			// Validate the resource
			if p.config.ValidateOnBorrow && pooled.resource != nil {
				if !pooled.resource.IsValid() || !p.factory.Validate(pooled.resource) {
					// Resource is invalid; destroy it and try to create a new one
					p.destroyResource(pooled)
					if newResource, err := p.tryCreateResource(); err == nil {
						return newResource, nil
					}
					continue
				}
			}

			// Reset the resource state
			if err := p.factory.Reset(pooled.resource); err != nil {
				p.destroyResource(pooled)
				continue
			}

			// Mark as in-use
			p.mu.Lock()
			pooled.inUse = true
			pooled.lastUsed = time.Now()
			p.mu.Unlock()

			return pooled.resource, nil
		default:
			// No available resources; try to create a new one
			if resource, err := p.tryCreateResource(); err == nil {
				return resource, nil
			}
			// Creation failed; wait for a resource to be released
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// tryCreateResource attempts to create a new resource
func (p *ResourcePool) tryCreateResource() (Resource, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.resources) >= p.config.MaxSize {
		return nil, errors.New("pool is full")
	}

	resource, err := p.factory.Create()
	if err != nil {
		return nil, err
	}

	pooled := &pooledResource{
		resource:   resource,
		createTime: time.Now(),
		lastUsed:   time.Now(),
		inUse:      true,
	}

	p.resources[resource] = pooled
	return resource, nil
}

// Release returns a resource back to the pool
func (p *ResourcePool) Release(resource Resource) error {
	if resource == nil {
		return errors.New("resource cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("pool is closed")
	}

	pooled, exists := p.resources[resource]
	if !exists {
		return errors.New("resource not managed by this pool")
	}

	if !pooled.inUse {
		return errors.New("resource is not in use")
	}

	// Validate the resource
	if p.config.ValidateOnReturn {
		if !resource.IsValid() || !p.factory.Validate(resource) {
			p.destroyResourceUnsafe(pooled)
			return nil
		}
	}

	// Check whether the max idle count is exceeded
	if len(p.available) >= p.config.MaxIdle {
		p.destroyResourceUnsafe(pooled)
		return nil
	}

	// Mark as available
	pooled.inUse = false
	pooled.lastUsed = time.Now()

	// Try to put back into the available queue
	select {
	case p.available <- pooled:
		return nil
	default:
		// Queue is full; destroy the resource
		p.destroyResourceUnsafe(pooled)
		return nil
	}
}

// destroyResource destroys a resource (with lock)
func (p *ResourcePool) destroyResource(pooled *pooledResource) {
	p.mu.Lock()
	p.destroyResourceUnsafe(pooled)
	p.mu.Unlock()
}

// destroyResourceUnsafe destroys a resource (without lock)
func (p *ResourcePool) destroyResourceUnsafe(pooled *pooledResource) {
	if pooled.resource != nil {
		pooled.resource.Close()
		delete(p.resources, pooled.resource)
	}
}

// Stats returns pool statistics
func (p *ResourcePool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	inUseCount := 0
	for _, pooled := range p.resources {
		if pooled.inUse {
			inUseCount++
		}
	}

	return map[string]interface{}{
		"total_resources":     len(p.resources),
		"available_resources": len(p.available),
		"in_use_resources":    inUseCount,
		"max_size":            p.config.MaxSize,
		"min_size":            p.config.MinSize,
		"max_idle":            p.config.MaxIdle,
		"is_closed":           p.closed,
	}
}

// Resize adjusts the pool size
func (p *ResourcePool) Resize(newMaxSize int) error {
	if newMaxSize <= 0 {
		return errors.New("new max size must be positive")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("pool is closed")
	}

	oldMaxSize := p.config.MaxSize
	p.config.MaxSize = newMaxSize

	// If shrinking the pool, remove excess resources
	if newMaxSize < oldMaxSize {
		excess := len(p.resources) - newMaxSize
		for excess > 0 {
			select {
			case pooled := <-p.available:
				p.destroyResourceUnsafe(pooled)
				excess--
			default:
				// No more available resources to remove
				break
			}
		}
	}

	return nil
}

// startCleanupRoutine starts the cleanup goroutine
func (p *ResourcePool) startCleanupRoutine() {
	if p.config.IdleTimeout <= 0 {
		return
	}

	p.cleanupWg.Add(1)
	go func() {
		defer p.cleanupWg.Done()
		ticker := time.NewTicker(p.config.IdleTimeout / 2)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.cleanupIdleResources()
			}
		}
	}()
}

// cleanupIdleResources cleans up resources that have been idle past the timeout
func (p *ResourcePool) cleanupIdleResources() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	now := time.Now()
	var toRemove []*pooledResource

	// Check idle resources in the available queue
	for {
		select {
		case pooled := <-p.available:
			if now.Sub(pooled.lastUsed) > p.config.IdleTimeout {
				toRemove = append(toRemove, pooled)
			} else {
				// Put back into the queue
				p.available <- pooled
				goto cleanup
			}
		default:
			goto cleanup
		}
	}

cleanup:
	// Destroy timed-out resources
	for _, pooled := range toRemove {
		p.destroyResourceUnsafe(pooled)
	}
}

// Close closes the resource pool
func (p *ResourcePool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Cancel the context
	p.cancel()

	// Wait for the cleanup goroutine to finish
	p.cleanupWg.Wait()

	// Close all resources
	p.mu.Lock()
	defer p.mu.Unlock()

	// Drain the available queue
	close(p.available)
	for pooled := range p.available {
		p.destroyResourceUnsafe(pooled)
	}

	// Close all resources
	for _, pooled := range p.resources {
		p.destroyResourceUnsafe(pooled)
	}

	return nil
}
