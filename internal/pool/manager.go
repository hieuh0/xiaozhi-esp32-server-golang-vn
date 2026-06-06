package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/asr"
	"xiaozhi-esp32-server-golang/internal/domain/llm"
	"xiaozhi-esp32-server-golang/internal/domain/tts"
	"xiaozhi-esp32-server-golang/internal/domain/vad"
	vad_inter "xiaozhi-esp32-server-golang/internal/domain/vad/inter"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/spf13/viper"
)

var (
	globalManager *UniversalResourcePoolManager
	once          sync.Once
)

// UniversalResourcePoolManager manages shared resource pools.
type UniversalResourcePoolManager struct {
	pools        map[string]*util.ResourcePool // Key format: "resourceType:provider".
	creators     map[string]interface{}        // Registered creator functions.
	closeFuncs   map[string]func(interface{}) error
	isValidFuncs map[string]func(interface{}) bool
	resetFuncs   map[string]func(interface{}) error
	mu           sync.RWMutex
}

// GetGlobalResourcePoolManager returns the global resource pool manager.
func GetGlobalResourcePoolManager() *UniversalResourcePoolManager {
	once.Do(func() {
		globalManager = &UniversalResourcePoolManager{
			pools:        make(map[string]*util.ResourcePool),
			creators:     make(map[string]interface{}),
			closeFuncs:   make(map[string]func(interface{}) error),
			isValidFuncs: make(map[string]func(interface{}) bool),
			resetFuncs:   make(map[string]func(interface{}) error),
		}
		log.Info("universal resource pool manager initialized")
	})
	return globalManager
}

// ResourceTypeOption configures resource type registration.
type ResourceTypeOption func(*ResourceTypeConfig)

// ResourceTypeConfig contains resource type settings.
type ResourceTypeConfig struct {
	CloseFunc   func(interface{}) error
	IsValidFunc func(interface{}) bool
	ResetFunc   func(interface{}) error
}

// WithCloseFunc sets the close function.
func WithCloseFunc(fn func(interface{}) error) ResourceTypeOption {
	return func(c *ResourceTypeConfig) {
		c.CloseFunc = fn
	}
}

// WithIsValidFunc sets the validation function.
func WithIsValidFunc(fn func(interface{}) bool) ResourceTypeOption {
	return func(c *ResourceTypeConfig) {
		c.IsValidFunc = fn
	}
}

// WithResetFunc sets the reset function.
func WithResetFunc(fn func(interface{}) error) ResourceTypeOption {
	return func(c *ResourceTypeConfig) {
		c.ResetFunc = fn
	}
}

// RegisterResourceType registers a resource type.
// resourceType is a name such as "vad", "asr", or "custom_type".
// creator creates the resource.
// opts contains optional close, validation, and reset functions.
func RegisterResourceType[T any](
	resourceType string,
	creator CreatorFunc[T],
	opts ...ResourceTypeOption,
) error {
	mgr := GetGlobalResourcePoolManager()
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// Check whether the type is already registered.
	if _, exists := mgr.creators[resourceType]; exists {
		return fmt.Errorf("resource type %s is already registered", resourceType)
	}

	// Register the creator.
	mgr.creators[resourceType] = creator

	// Apply options.
	config := &ResourceTypeConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.CloseFunc != nil {
		mgr.closeFuncs[resourceType] = config.CloseFunc
	}
	if config.IsValidFunc != nil {
		mgr.isValidFuncs[resourceType] = config.IsValidFunc
	}
	if config.ResetFunc != nil {
		mgr.resetFuncs[resourceType] = config.ResetFunc
	}

	log.Infof("registered resource type: %s", resourceType)
	return nil
}

// GenerateConfigKey generates a key that distinguishes resource pool configurations.
// hashstructure makes the fingerprint independent of map key order.
func GenerateConfigKey(provider string, config map[string]interface{}) string {
	input := map[string]interface{}{"provider": provider, "config": config}
	h, err := hashstructure.Hash(input, hashstructure.FormatV2, nil)
	if err != nil {
		log.Warnf("failed to calculate config fingerprint; using provider as key: %v", err)
		return provider
	}
	return fmt.Sprintf("%016x", h)
}

// getOrCreatePool gets or creates a generic resource pool.
// The config fingerprint ensures changed settings use a new pool instance.
func getOrCreatePool[T any](
	resourceType, provider string,
	config map[string]interface{},
) (*util.ResourcePool, error) {
	mgr := GetGlobalResourcePoolManager()
	// Pool key format: type:configuration fingerprint.
	configKey := GenerateConfigKey(provider, config)
	poolKey := fmt.Sprintf("%s:%s", resourceType, configKey)

	mgr.mu.RLock()
	pool, exists := mgr.pools[poolKey]
	mgr.mu.RUnlock()

	if exists {
		return pool, nil
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// Double-check after acquiring the lock.
	if pool, exists := mgr.pools[poolKey]; exists {
		return pool, nil
	}

	// Get the registered creator.
	creatorInterface, exists := mgr.creators[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type is not registered: %s (call RegisterResourceType first)", resourceType)
	}

	// Assert the generic creator type.
	creator, ok := creatorInterface.(CreatorFunc[T])
	if !ok {
		return nil, fmt.Errorf("creator type mismatch for resource type %s", resourceType)
	}

	// Create the generic resource factory.
	factory := &ResourceFactory[T]{
		resourceType: resourceType,
		provider:     provider,
		config:       config,
		configKey:    configKey,
		creator:      creator,
		closeFunc: func(p T) error {
			if closeFunc := mgr.closeFuncs[resourceType]; closeFunc != nil {
				return closeFunc(any(p))
			}
			return nil
		},
		isValidFunc: func(p T) bool {
			if isValidFunc := mgr.isValidFuncs[resourceType]; isValidFunc != nil {
				return isValidFunc(any(p))
			}
			return true
		},
		resetFunc: func(p T) error {
			if resetFunc := mgr.resetFuncs[resourceType]; resetFunc != nil {
				return resetFunc(any(p))
			}
			return nil
		},
	}

	// Get pool settings shared by all resource types.
	poolConfig := getPoolConfig()

	// Create the resource pool.
	pool, err := util.NewResourcePool(poolConfig, factory)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource pool [%s:%s]: %w", resourceType, configKey, err)
	}

	mgr.pools[poolKey] = pool
	fpShort := configKey
	if len(configKey) > 8 {
		fpShort = configKey[:8] + "..."
	}
	log.Infof("created resource pool: type=%s, provider=%s, fingerprint=%s", resourceType, provider, fpShort)

	return pool, nil
}

// Acquire gets a type-safe resource and supports lazy initialization.
// T is the resource type.
// resourceType identifies vad/asr/llm/tts or another registered type.
// provider is the provider name and config contains its settings.
func Acquire[T any](
	resourceType, provider string,
	config map[string]interface{},
) (*ResourceWrapper[T], error) {
	pool, err := getOrCreatePool[T](resourceType, provider, config)
	if err != nil {
		return nil, err
	}

	resource, err := pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire resource [%s:%s]: %w", resourceType, provider, err)
	}

	wrapper, ok := resource.(*ResourceWrapper[T])
	if !ok {
		pool.Release(resource)
		return nil, fmt.Errorf("invalid resource type: expected ResourceWrapper[%T]", *new(T))
	}

	return wrapper, nil
}

// Release returns a type-safe resource to its pool.
func Release[T any](wrapper *ResourceWrapper[T]) error {
	if wrapper == nil {
		return nil
	}

	mgr := GetGlobalResourcePoolManager()
	// All resource pool keys use the type:provider format.
	poolKey := fmt.Sprintf("%s:%s", wrapper.resourceType, wrapper.configKey)

	mgr.mu.RLock()
	pool, exists := mgr.pools[poolKey]
	mgr.mu.RUnlock()

	if !exists {
		log.Warnf("resource pool does not exist: %s", poolKey)
		return nil
	}

	return pool.Release(wrapper)
}

// GetStats returns statistics for all resource pools.
func GetStats() map[string]interface{} {
	mgr := GetGlobalResourcePoolManager()
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	stats := make(map[string]interface{})

	for poolKey, pool := range mgr.pools {
		stats[poolKey] = pool.Stats()
	}

	return stats
}

// StartStatsMonitor logs resource pool statistics at each interval.
func StartStatsMonitor(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Debugf("resource pool stats monitor stopped")
				return
			case <-ticker.C:
				stats := GetStats()
				if len(stats) > 0 {
					statsJSON, err := json.MarshalIndent(stats, "", "  ")
					if err != nil {
						log.Errorf("failed to marshal resource pool statistics: %v", err)
						continue
					}
					log.Infof("========== Global Resource Pool Statistics ==========")
					log.Infof("timestamp: %s", time.Now().Format("2006-01-02 15:04:05"))
					log.Infof("pool count: %d", len(stats))
					log.Infof("details:\n%s", string(statsJSON))
					log.Infof("========================================")
				} else {
					log.Infof("========== Global Resource Pool Statistics ==========")
					log.Infof("timestamp: %s", time.Now().Format("2006-01-02 15:04:05"))
					log.Infof("no active resource pools")
					log.Infof("========================================")
				}
			}
		}
	}()
	log.Infof("resource pool stats monitor started, interval: %v", interval)
}

// Close closes all resource pools.
func Close() error {
	mgr := GetGlobalResourcePoolManager()
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	var errs []error

	for poolKey, pool := range mgr.pools {
		if err := pool.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close resource pool %s: %w", poolKey, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while closing resource pools: %v", errs)
	}

	return nil
}

// getPoolConfig returns shared resource pool settings from configuration.
func getPoolConfig() *util.PoolConfig {
	// Use default settings.
	config := util.DefaultConfig()

	// Override defaults when resource_pools is configured.
	if viper.IsSet("resource_pools.max_size") {
		config.MaxSize = viper.GetInt("resource_pools.max_size")
	}
	if viper.IsSet("resource_pools.min_size") {
		config.MinSize = viper.GetInt("resource_pools.min_size")
	}
	if viper.IsSet("resource_pools.max_idle") {
		config.MaxIdle = viper.GetInt("resource_pools.max_idle")
	}
	if viper.IsSet("resource_pools.acquire_timeout") {
		config.AcquireTimeout = viper.GetDuration("resource_pools.acquire_timeout")
	}
	if viper.IsSet("resource_pools.idle_timeout") {
		config.IdleTimeout = viper.GetDuration("resource_pools.idle_timeout")
	}
	if viper.IsSet("resource_pools.validate_on_borrow") {
		config.ValidateOnBorrow = viper.GetBool("resource_pools.validate_on_borrow")
	}
	if viper.IsSet("resource_pools.validate_on_return") {
		config.ValidateOnReturn = viper.GetBool("resource_pools.validate_on_return")
	}

	return config
}

// init registers built-in resource types.
func init() {
	// Register the VAD resource type.
	RegisterResourceType[vad_inter.VAD](
		"vad",
		func(rt, p string, cfg map[string]interface{}) (vad_inter.VAD, error) {
			vadProvider, err := vad.AcquireVAD(p, cfg)
			if err != nil {
				return nil, err
			}
			if vadProvider != nil {
				vadProvider.Reset()
			}
			return vadProvider, nil
		},
		WithCloseFunc(func(p interface{}) error {
			if vadProvider, ok := p.(vad_inter.VAD); ok && vadProvider != nil {
				return vadProvider.Close()
			}
			return nil
		}),
		WithIsValidFunc(func(p interface{}) bool {
			if vadProvider, ok := p.(vad_inter.VAD); ok && vadProvider != nil {
				return vadProvider.IsValid()
			}
			return false
		}),
		WithResetFunc(func(p interface{}) error {
			if vadProvider, ok := p.(vad_inter.VAD); ok && vadProvider != nil {
				return vadProvider.Reset()
			}
			return nil
		}),
	)

	// Register the ASR resource type.
	RegisterResourceType[asr.AsrProvider](
		"asr",
		func(rt, p string, cfg map[string]interface{}) (asr.AsrProvider, error) {
			return asr.NewAsrProvider(p, cfg)
		},
		WithIsValidFunc(func(p interface{}) bool {
			if asrProvider, ok := p.(asr.AsrProvider); ok && asrProvider != nil {
				return asrProvider.IsValid()
			}
			return false
		}),
		WithCloseFunc(func(p interface{}) error {
			if asrProvider, ok := p.(asr.AsrProvider); ok && asrProvider != nil {
				return asrProvider.Close()
			}
			return nil
		}),
	)

	// Register the LLM resource type.
	RegisterResourceType[llm.LLMProvider](
		"llm",
		func(rt, p string, cfg map[string]interface{}) (llm.LLMProvider, error) {
			providerName, ok := cfg["provider"].(string)
			if !ok || providerName == "" {
				providerName = p
			}
			return llm.GetLLMProvider(providerName, cfg)
		},
		WithIsValidFunc(func(p interface{}) bool {
			if llmProvider, ok := p.(llm.LLMProvider); ok && llmProvider != nil {
				return llmProvider.IsValid()
			}
			return false
		}),
		WithCloseFunc(func(p interface{}) error {
			if llmProvider, ok := p.(llm.LLMProvider); ok && llmProvider != nil {
				return llmProvider.Close()
			}
			return nil
		}),
	)

	// Register the TTS resource type.
	RegisterResourceType[tts.TTSProvider](
		"tts",
		func(rt, p string, cfg map[string]interface{}) (tts.TTSProvider, error) {
			return tts.GetTTSProvider(p, cfg)
		},
		WithIsValidFunc(func(p interface{}) bool {
			if ttsProvider, ok := p.(tts.TTSProvider); ok && ttsProvider != nil {
				return ttsProvider.IsValid()
			}
			return false
		}),
		WithCloseFunc(func(p interface{}) error {
			if ttsProvider, ok := p.(tts.TTSProvider); ok && ttsProvider != nil {
				return ttsProvider.Close()
			}
			return nil
		}),
	)
}
