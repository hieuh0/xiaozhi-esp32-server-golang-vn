package hooks

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Context struct {
	Ctx  context.Context
	Meta any
}

type SyncHandler func(Context, any) (any, bool, error)
type AsyncHandler func(Context, any)

type PluginKind string

const (
	PluginKindInterceptor PluginKind = "interceptor"
	PluginKindObserver    PluginKind = "observer"
)

type PluginMeta struct {
	Name     string
	Priority int
	Kind     PluginKind
	Stage    string
	Enabled  bool
}

type AsyncConfig struct {
	QueueSize    int
	WorkerCount  int
	DropWhenFull bool
	Timeout      time.Duration
}

type HubOption func(*Hub)

type namedSync struct {
	meta    PluginMeta
	enabled bool
	handler SyncHandler
}

type namedAsync struct {
	meta    PluginMeta
	enabled bool
	handler AsyncHandler
}

type PluginStats struct {
	Invocations     int64
	Errors          int64
	Stops           int64
	Timeouts        int64
	DroppedAsync    int64
	TotalDurationMs int64
}

type Stats struct {
	AsyncQueueLength int
	DroppedAsync     int64
	Plugins          map[string]PluginStats
}

type AsyncExecutor struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg AsyncConfig

	mu    sync.Mutex
	cond  *sync.Cond
	queue []func()

	workerWG sync.WaitGroup
}

func NewAsyncExecutor(parent context.Context, cfg AsyncConfig) *AsyncExecutor {
	if parent == nil {
		parent = context.Background()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1024
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	ctx, cancel := context.WithCancel(parent)
	exec := &AsyncExecutor{ctx: ctx, cancel: cancel, cfg: cfg}
	exec.cond = sync.NewCond(&exec.mu)
	for i := 0; i < exec.cfg.WorkerCount; i++ {
		exec.workerWG.Add(1)
		go exec.run()
	}
	return exec
}

func (e *AsyncExecutor) run() {
	defer e.workerWG.Done()
	for {
		task := e.pop()
		if task == nil {
			return
		}
		task()
	}
}

func (e *AsyncExecutor) pop() func() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for len(e.queue) == 0 && e.ctx.Err() == nil {
		e.cond.Wait()
	}
	if len(e.queue) == 0 {
		return nil
	}
	task := e.queue[0]
	e.queue[0] = nil
	e.queue = e.queue[1:]
	return task
}

func (e *AsyncExecutor) Submit(task func()) bool {
	if e == nil || task == nil {
		return true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.ctx.Err() != nil {
		return false
	}

	// Handle a full queue according to configuration.
	if e.cfg.QueueSize > 0 && len(e.queue) >= e.cfg.QueueSize {
		if e.cfg.DropWhenFull {
			return false
		}
		// When DropWhenFull is false, wait for the queue to consume an item.
		// Use a timeout to avoid blocking forever.
		waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		for len(e.queue) >= e.cfg.QueueSize && waitCtx.Err() == nil {
			e.cond.Wait()
		}
		if waitCtx.Err() != nil {
			// Drop on timeout.
			return false
		}
	}

	e.queue = append(e.queue, task)
	e.cond.Signal()
	return true
}

func (e *AsyncExecutor) QueueLength() int {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.queue)
}

func (e *AsyncExecutor) Close() {
	if e == nil || e.cancel == nil {
		return
	}
	e.cancel()
	e.cond.Broadcast()
	e.workerWG.Wait()
}

type Hub struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu sync.RWMutex

	syncHandlers  map[string][]namedSync
	asyncHandlers map[string][]namedAsync

	asyncConfig AsyncConfig
	asyncExec   *AsyncExecutor
	ownsAsync   bool

	droppedAsync atomic.Int64

	statsMu      sync.Mutex
	pluginStats  map[string]*PluginStats
	pluginMetaMu sync.RWMutex
	pluginMeta   map[string]PluginMeta
}

func defaultAsyncConfig() AsyncConfig {
	return AsyncConfig{QueueSize: 1024, WorkerCount: 1, DropWhenFull: true, Timeout: 200 * time.Millisecond}
}

func WithAsyncConfig(cfg AsyncConfig) HubOption {
	return func(h *Hub) {
		if cfg.QueueSize > 0 {
			h.asyncConfig.QueueSize = cfg.QueueSize
		}
		if cfg.WorkerCount > 0 {
			h.asyncConfig.WorkerCount = cfg.WorkerCount
		}
		if cfg.Timeout > 0 {
			h.asyncConfig.Timeout = cfg.Timeout
		}
		h.asyncConfig.DropWhenFull = cfg.DropWhenFull
	}
}

func WithAsyncExecutor(exec *AsyncExecutor) HubOption {
	return func(h *Hub) {
		h.asyncExec = exec
		h.ownsAsync = false
	}
}

func NewHub(parent context.Context, opts ...HubOption) *Hub {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	h := &Hub{
		ctx:           ctx,
		cancel:        cancel,
		syncHandlers:  make(map[string][]namedSync),
		asyncHandlers: make(map[string][]namedAsync),
		asyncConfig:   defaultAsyncConfig(),
		pluginStats:   make(map[string]*PluginStats),
		pluginMeta:    make(map[string]PluginMeta),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	if h.asyncExec == nil {
		h.asyncExec = NewAsyncExecutor(ctx, h.asyncConfig)
		h.ownsAsync = true
	}
	return h
}

func (h *Hub) Close() {
	if h == nil || h.cancel == nil {
		return
	}
	h.cancel()
	if h.ownsAsync && h.asyncExec != nil {
		h.asyncExec.Close()
	}
}

// EnablePlugin enables a registered plugin by name
func (h *Hub) EnablePlugin(name string) {
	h.setPluginEnabled(name, true)
}

// DisablePlugin disables a registered plugin by name
func (h *Hub) DisablePlugin(name string) {
	h.setPluginEnabled(name, false)
}

func (h *Hub) setPluginEnabled(name string, enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find and update the synchronous handler.
	for event, handlers := range h.syncHandlers {
		for i := range handlers {
			if handlers[i].meta.Name == name {
				handlers[i].enabled = enabled
				h.syncHandlers[event] = handlers
				return
			}
		}
	}

	// Find and update the asynchronous handler.
	for event, handlers := range h.asyncHandlers {
		for i := range handlers {
			if handlers[i].meta.Name == name {
				handlers[i].enabled = enabled
				h.asyncHandlers[event] = handlers
				return
			}
		}
	}
}

func (h *Hub) Stats() Stats {
	if h == nil {
		return Stats{}
	}
	queueLen := 0
	if h.asyncExec != nil {
		queueLen = h.asyncExec.QueueLength()
	}
	h.statsMu.Lock()
	plugins := make(map[string]PluginStats, len(h.pluginStats))
	for name, st := range h.pluginStats {
		plugins[name] = *st
	}
	h.statsMu.Unlock()
	return Stats{AsyncQueueLength: queueLen, DroppedAsync: h.droppedAsync.Load(), Plugins: plugins}
}

func (h *Hub) PluginMetas() []PluginMeta {
	if h == nil {
		return nil
	}
	h.pluginMetaMu.RLock()
	defer h.pluginMetaMu.RUnlock()
	out := make([]PluginMeta, 0, len(h.pluginMeta))
	for _, meta := range h.pluginMeta {
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Stage == out[j].Stage {
			return out[i].Priority < out[j].Priority
		}
		if out[i].Kind == out[j].Kind {
			return out[i].Stage < out[j].Stage
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func (h *Hub) emitAsync(ctx Context, hooks []namedAsync, payload any) {
	for _, hk := range hooks {
		if !hk.enabled {
			continue
		}
		hook := hk
		if h.ctx.Err() != nil {
			return
		}
		queued := false
		if h.asyncExec != nil {
			queued = h.asyncExec.Submit(func() { h.runAsyncHandler(hook, ctx, payload) })
		}
		if !queued {
			h.droppedAsync.Add(1)
			h.recordDropped(hook.meta.Name)
		}
	}
}

func (h *Hub) runAsyncHandler(hk namedAsync, ctx Context, payload any) {
	start := time.Now()
	h.recordInvocation(hk.meta.Name)
	if h.asyncConfig.Timeout <= 0 {
		hk.handler(ctx, payload)
		h.recordDuration(hk.meta.Name, time.Since(start))
		return
	}
	baseCtx := ctx.Ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	deadlineCtx, cancel := context.WithTimeout(baseCtx, h.asyncConfig.Timeout)
	defer cancel()
	wrappedCtx := ctx
	wrappedCtx.Ctx = deadlineCtx
	done := make(chan struct{})
	go func() {
		defer close(done)
		hk.handler(wrappedCtx, payload)
	}()
	select {
	case <-done:
		h.recordDuration(hk.meta.Name, time.Since(start))
	case <-deadlineCtx.Done():
		h.recordTimeout(hk.meta.Name)
		h.recordDuration(hk.meta.Name, time.Since(start))
	}
}

func (h *Hub) recordPluginMeta(meta PluginMeta) {
	h.pluginMetaMu.Lock()
	defer h.pluginMetaMu.Unlock()
	h.pluginMeta[meta.Name] = meta
}

func (h *Hub) ensurePluginStatsLocked(name string) *PluginStats {
	if h.pluginStats[name] == nil {
		h.pluginStats[name] = &PluginStats{}
	}
	return h.pluginStats[name]
}

func (h *Hub) recordInvocation(name string) {
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(name).Invocations++
	h.statsMu.Unlock()
}
func (h *Hub) recordError(name string) {
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(name).Errors++
	h.statsMu.Unlock()
}
func (h *Hub) recordStop(name string) {
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(name).Stops++
	h.statsMu.Unlock()
}
func (h *Hub) recordTimeout(name string) {
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(name).Timeouts++
	h.statsMu.Unlock()
}
func (h *Hub) recordDropped(name string) {
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(name).DroppedAsync++
	h.statsMu.Unlock()
}
func (h *Hub) recordDuration(name string, duration time.Duration) {
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(name).TotalDurationMs += duration.Milliseconds()
	h.statsMu.Unlock()
}

func (h *Hub) RegisterSync(event, name string, priority int, handler SyncHandler) {
	h.RegisterSyncMeta(event, PluginMeta{Name: name, Priority: priority, Kind: PluginKindInterceptor, Stage: event}, handler)
}

func (h *Hub) RegisterSyncMeta(event string, meta PluginMeta, handler SyncHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if meta.Name == "" {
		meta.Name = event
	}
	meta.Kind = PluginKindInterceptor
	meta.Stage = event
	// Enabled by default.
	enabled := meta.Enabled
	if !meta.Enabled {
		// Controlled by Enabled, which defaults to true.
		enabled = true
	}
	h.recordPluginMeta(meta)
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(meta.Name)
	h.statsMu.Unlock()
	cur := h.syncHandlers[event]
	next := make([]namedSync, 0, len(cur)+1)
	next = append(next, cur...)
	next = append(next, namedSync{meta: meta, enabled: enabled, handler: handler})
	sort.SliceStable(next, func(i, j int) bool { return next[i].meta.Priority < next[j].meta.Priority })
	h.syncHandlers[event] = next
}

func (h *Hub) RegisterAsync(event, name string, priority int, handler AsyncHandler) {
	h.RegisterAsyncMeta(event, PluginMeta{Name: name, Priority: priority, Kind: PluginKindObserver, Stage: event}, handler)
}

func (h *Hub) RegisterAsyncMeta(event string, meta PluginMeta, handler AsyncHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if meta.Name == "" {
		meta.Name = event
	}
	meta.Kind = PluginKindObserver
	meta.Stage = event
	// Enabled by default.
	enabled := meta.Enabled
	if !meta.Enabled {
		enabled = true
	}
	h.recordPluginMeta(meta)
	h.statsMu.Lock()
	h.ensurePluginStatsLocked(meta.Name)
	h.statsMu.Unlock()
	cur := h.asyncHandlers[event]
	next := make([]namedAsync, 0, len(cur)+1)
	next = append(next, cur...)
	next = append(next, namedAsync{meta: meta, enabled: enabled, handler: handler})
	sort.SliceStable(next, func(i, j int) bool { return next[i].meta.Priority < next[j].meta.Priority })
	h.asyncHandlers[event] = next
}

func (h *Hub) Emit(event string, ctx Context, payload any) (any, bool, error) {
	h.mu.RLock()
	syncs := h.syncHandlers[event]
	asyncs := h.asyncHandlers[event]
	h.mu.RUnlock()
	out := payload
	for _, hk := range syncs {
		if !hk.enabled {
			continue
		}
		start := time.Now()
		h.recordInvocation(hk.meta.Name)
		next, stop, err := hk.handler(ctx, out)
		h.recordDuration(hk.meta.Name, time.Since(start))
		if err != nil {
			h.recordError(hk.meta.Name)
			return out, stop, fmt.Errorf("hook %s failed: %w", hk.meta.Name, err)
		}
		out = next
		if stop {
			h.recordStop(hk.meta.Name)
			h.emitAsync(ctx, asyncs, out)
			return out, true, nil
		}
	}
	h.emitAsync(ctx, asyncs, out)
	return out, false, nil
}
