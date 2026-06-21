# Code Standards & Conventions

**Project**: xiaozhi-esp32-server-golang  
**Effective**: 2026-06-20  
**Scope**: Go backend, TypeScript/React frontend, shell scripts

---

## 1. Go Code Standards

### 1.1 Naming Conventions

**Packages**
- Single word lowercase: `asr`, `tts`, `llm`, `vad`
- No underscores or hyphens
- Use `internal/domain/{domain_name}` for core modules
- Use `internal/app` for application wiring
- Use `internal/util` for helpers

**Files**
- Lowercase with underscores: `provider.go`, `websocket_server.go`, `chat_session.go`
- Provider implementations: `{provider_name}.go` (e.g., `funasr.go`, `openai.go`)
- Tests: `{file}_test.go` suffix

**Functions & Methods**
- Exported: PascalCase (e.g., `NewChatSession()`, `StreamChat()`)
- Unexported: camelCase (e.g., `processAudio()`, `validateConfig()`)
- Constructor pattern: `New{Type}` (e.g., `NewASRManager`)
- Boolean functions: `Is{Action}`, `Has{Property}` (e.g., `IsValid()`, `HasError()`)

**Variables & Constants**
- Local: camelCase (e.g., `deviceID`, `audioBuffer`)
- Package-level constants: UPPER_SNAKE_CASE (e.g., `MAX_QUEUE_SIZE`)
- Interface definitions: `I{Name}` prefix (e.g., `IConn`, `IProvider`)

**Types**
- Structs: PascalCase (e.g., `ChatSession`, `ASRProvider`)
- Interfaces: PascalCase with `I` prefix (e.g., `IConn`)
- Enums: PascalCase constants (e.g., `StateListening`, `StateTTSStart`)

### 1.2 Project Structure

```
cmd/
  ├── server/main.go              # Main entry point
  └── {utility_name}/main.go      # CLI utilities

internal/
  ├── app/
  │   ├── server/                 # Main server logic
  │   └── mqtt_server/            # MQTT broker
  ├── data/
  │   ├── client/client.go        # Per-device state
  │   ├── message/message_types.go # Protocol definitions
  │   └── dialogue.go             # Chat history
  ├── domain/
  │   ├── {domain_name}/
  │   │   ├── provider.go         # Interface
  │   │   ├── {provider_name}.go  # Implementation
  │   │   └── manager.go          # Provider lifecycle
  │   └── config/provider.go      # Config abstractions
  ├── pkg/                        # Shared utilities (hooks, transforms)
  └── util/                       # Helpers (queue, pool, crypto)

manager/
  ├── backend/
  │   ├── main.go
  │   ├── controllers/
  │   ├── models/
  │   ├── services/
  │   ├── middleware/
  │   └── router/
  └── frontend/
      └── src/

docs/                            # Documentation
```

### 1.3 Interface Design

All AI providers follow a common interface pattern:

```go
// provider.go
type Provider interface {
    // Initialize sets up the provider with given config
    Initialize(ctx context.Context) error
    
    // Close releases resources (models, connections)
    Close() error
    
    // Config returns the provider configuration structure
    // Returns a pointer to config struct (e.g., *OpenAIConfig)
    Config() interface{}
}
```

**Transport abstraction** (`IConn`):

```go
type IConn interface {
    // SendCmd sends JSON command message
    SendCmd(msg []byte) error
    
    // RecvCmd receives JSON command message with timeout (ms)
    RecvCmd(ctx context.Context, timeout int) ([]byte, error)
    
    // SendAudio sends Opus audio chunk
    SendAudio(audio []byte) error
    
    // RecvAudio receives audio chunk with timeout (ms)
    RecvAudio(ctx context.Context, timeout int) ([]byte, error)
    
    GetDeviceID() string
    GetTransportType() string // "websocket", "mqtt", "udp"
    Close() error
    OnClose(callback func(deviceId string))
    CloseAudioChannel() error
}
```

### 1.4 Error Handling

**Patterns**:

```go
// Explicit error returns (no exceptions)
func ProcessAudio(audio []byte) error {
    if len(audio) == 0 {
        return fmt.Errorf("audio buffer empty")
    }
    // ... processing
    return nil
}

// Error wrapping (Go 1.13+)
if err != nil {
    return fmt.Errorf("failed to transcribe: %w", err)
}

// Silent failures for non-critical operations (e.g., logging, analytics)
if err := updateMetrics(metric); err != nil {
    log.WithError(err).Warn("failed to update metrics")
    // Continue execution
}

// Panic only in initialization or truly fatal scenarios (rare)
if err := os.ReadFile(configPath); err != nil {
    panic(fmt.Sprintf("failed to read config: %v", err))
}
```

**Error context**: Always include why the error happened, not just what:

```go
// Bad:
return err

// Good:
return fmt.Errorf("failed to connect to MQTT broker %s:%d: %w", host, port, err)
```

### 1.5 Configuration Pattern

Configs are YAML-driven via Viper:

```go
// In config.yaml
asr:
  provider: "funasr"
  funasr:
    ws_url: "ws://127.0.0.1:10095"
    language: "zh"

// In code (internal/domain/asr/provider.go)
type Config struct {
    Provider string `json:"provider"`
    FunASR  FunASRConfig `json:"funasr"`
    // ... other providers
}

type FunASRConfig struct {
    WsURL    string `json:"ws_url"`
    Language string `json:"language"`
}

// Manager initializes provider based on config
func (m *Manager) Initialize(cfg *Config) error {
    switch cfg.Provider {
    case "funasr":
        p := NewFunASRProvider(cfg.FunASR)
        return p.Initialize(ctx)
    case "doubao":
        p := NewDouBaoProvider(cfg.DoubaoASR)
        return p.Initialize(ctx)
    default:
        return fmt.Errorf("unsupported ASR provider: %s", cfg.Provider)
    }
}
```

**Hot-reload**: Config changes trigger semantic diff check:

```go
// In cmd/server/main.go
user_config.RegisterManagerSystemConfigHandler(func(data map[string]interface{}) {
    if data["mqtt"] != nil {
        if !SystemConfigEqual(data["mqtt"], oldMqtt) {
            appInstance.ReloadMqtt() // Only reload if changed
        }
    }
})
```

### 1.6 Logging

Use `github.com/sirupsen/logrus` for structured logging:

```go
import log "xiaozhi-esp32-server-golang/logger"

// Log levels: debug, info, warn, error
log.Infof("server started on port %d", 8989)
log.WithFields(logrus.Fields{
    "device_id": deviceID,
    "provider": "openai",
}).Warn("provider timeout")

log.WithError(err).Error("failed to process audio")

// Never log sensitive data (API keys, passwords)
// Redact in middleware
```

**Log rotation**: Configured in `config.yaml`:

```yaml
log:
  path: "logs/"
  level: "info"
  max_age: 3           # days
  rotation_time: 10    # hours
  stdout: true
```

### 1.7 Concurrency

Use channels and mutexes appropriately:

```go
// For device connections (many goroutines, few write ops):
type ClientState struct {
    lock sync.RWMutex  // RWMutex for read-heavy workloads
    state State
    dialogue *Dialogue
}

func (cs *ClientState) GetState() State {
    cs.lock.RLock()
    defer cs.lock.RUnlock()
    return cs.state
}

// For message queues:
type MessageQueue struct {
    ch   chan Message
    done <-chan struct{}
}

func (q *MessageQueue) Send(msg Message) error {
    select {
    case q.ch <- msg:
        return nil
    case <-q.done:
        return fmt.Errorf("queue closed")
    }
}

// Avoid: global mutexes, callback hell, unbuffered channels in hot paths
```

### 1.8 Testing

```go
// Test file naming: {module}_test.go
// Test function naming: TestFunctionName

func TestASRManagerInitialize(t *testing.T) {
    m := NewASRManager()
    err := m.Initialize(context.Background(), &Config{Provider: "funasr"})
    if err != nil {
        t.Errorf("Initialize failed: %v", err)
    }
    // Clean up
    m.Close()
}

// Use subtests for multiple scenarios
func TestStreamChat(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantError bool
    }{
        {"valid input", "hello", false},
        {"empty input", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := llm.StreamChat(ctx, tt.input)
            if (err != nil) != tt.wantError {
                t.Errorf("got error %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

// Avoid: testing implementation details, brittle mocks, hardcoded timeouts
```

### 1.9 Comments & Documentation

```go
// Package asr provides speech-to-text recognition with multiple provider support.
package asr

// Manager coordinates ASR provider lifecycle and transcription requests.
type Manager struct {
    provider Provider // Current active provider
}

// Recognize transcribes audio to text using the configured provider.
// The audio buffer should contain Opus-encoded data.
// If the context is cancelled, Recognize returns the context error.
func (m *Manager) Recognize(ctx context.Context, audio []byte) (string, error) {
    // ...
}

// Unexported helper with clear intent
func (m *Manager) selectProvider(cfg *Config) (Provider, error) {
    // ...
}
```

**Avoid**:
- Over-commenting obvious code: `i++  // increment i`
- TODO comments without context: use GitHub issues instead
- Stale comments (update when refactoring)

---

## 2. TypeScript/React Frontend Standards

**Frontend standards are in a separate document** for maintainability. See **docs/frontend-code-standards.md** for:
- File organization & directory structure
- Component patterns and best practices
- Form handling (React Hook Form + Zod)
- API client patterns (Axios + React Query)
- Styling conventions (Tailwind CSS)
- i18n and type safety
- Testing patterns (Vitest)

---

## 3. Shell Script Standards

### 3.1 Bash/Shell

```bash
#!/bin/bash
# docker-local.sh — Local Docker development environment helper
# Usage: ./docker-local.sh {up|down|logs|rebuild}

set -euo pipefail  # Exit on error, undefined var, pipe failure

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'  # No Color

# Defaults
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.local.yml}"
SERVICE="${SERVICE:-}"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Main logic
if [[ "${1:-}" == "up" ]]; then
    log_info "Starting services..."
    docker-compose -f "$COMPOSE_FILE" up -d
elif [[ "${1:-}" == "down" ]]; then
    log_info "Stopping services..."
    docker-compose -f "$COMPOSE_FILE" down
else
    log_error "Unknown command: ${1:-}"
    exit 1
fi
```

---

## 4. Common Patterns & Idioms

### 4.1 Provider Registration

```go
// All providers register themselves in init()
func init() {
    RegisterProvider("funasr", func(cfg interface{}) (Provider, error) {
        return NewFunASRProvider(cfg.(*FunASRConfig))
    })
}
```

### 4.2 Manager Pattern (Lifecycle)

```go
type Manager struct {
    provider Provider
    mu       sync.RWMutex
}

func (m *Manager) Initialize(ctx context.Context, cfg *Config) error {
    provider, err := createProvider(cfg)
    if err != nil {
        return err
    }
    if err := provider.Initialize(ctx); err != nil {
        return err
    }
    m.mu.Lock()
    m.provider = provider
    m.mu.Unlock()
    return nil
}

func (m *Manager) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.provider != nil {
        return m.provider.Close()
    }
    return nil
}

func (m *Manager) InUse() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.provider != nil
}
```

### 4.3 Hot-Reload Pattern

```go
// Config change handler
type ConfigHandler func(data map[string]interface{})

var handlers []ConfigHandler

func RegisterConfigHandler(h ConfigHandler) {
    handlers = append(handlers, h)
}

func ApplyConfigChange(data map[string]interface{}) {
    for _, h := range handlers {
        h(data)
    }
}

// Semantic diff (only reload if values change)
func SystemConfigEqual(a, b interface{}) bool {
    aJSON, _ := json.Marshal(a)
    bJSON, _ := json.Marshal(b)
    return bytes.Equal(aJSON, bJSON)
}
```

---

## 5. Code Review Checklist

Before submitting a PR:

- [ ] Error handling: All non-trivial function calls check errors
- [ ] No hardcoded secrets (API keys, passwords) in code
- [ ] Logging includes context (device ID, provider, etc.)
- [ ] Configuration changes are hot-reload compatible
- [ ] Comments on exported functions explain intent
- [ ] Tests pass (run `go test ./...`)
- [ ] TypeScript compiles without errors (run `npm run build`)
- [ ] No console warnings in React builds

---

## 6. Performance Considerations

- **Go**: Avoid allocations in hot paths (use object pools, buffer reuse)
- **React**: Use `React.memo()` for expensive components; lazy load routes
- **Concurrency**: Limit goroutines per connection; use buffered channels
- **Database**: Index frequently queried fields; use connection pooling

---

## 7. Backward Compatibility

- **Config format**: New fields are optional; old fields remain supported
- **API responses**: Extend, don't remove or rename fields; use `omitempty` for optional
- **Provider interface**: Add new methods as separate version (e.g., `Provider` → `ProviderV2`)

---

## 8. Security Guidelines

- **Input validation**: All user inputs validated before use
- **API auth**: JWT for REST; bearer tokens for internal services
- **Secret management**: API keys stored in environment variables or encrypted DB
- **Logging**: Never log passwords, tokens, or sensitive PII
- **CORS**: Backend restricts origins to known frontend URLs
- **Rate limiting**: Per-device message limits; configurable

---

## 9. Documentation Standards

Every module's `provider.go` should include:

```go
// Package asr provides speech-to-text providers with streaming support.
//
// Supported providers:
// - FunASR: WebSocket-based streaming ASR with 2-pass support
// - Doubao: Alibaba Cloud ASR with dialect support
// - Xunfei: Xunfei ASR with real-time transcription
//
// Example:
//     cfg := &FunASRConfig{WsURL: "ws://localhost:10095"}
//     provider := NewFunASRProvider(cfg)
//     if err := provider.Initialize(ctx); err != nil {
//         log.Fatal(err)
//     }
//     text, err := provider.Recognize(ctx, audioData)
package asr
```

---

## 10. Tools & Automation

- **Linting**: `golangci-lint` (optional; follow conventions above instead)
- **Formatting**: `go fmt` (automatic via most IDEs)
- **Testing**: `go test ./...`
- **Build**: `go build -o xiaozhi_server ./cmd/server/`
- **Frontend**: `npm run dev` (dev server), `npm run build` (prod), `npm run lint`

---

## Appendix: Useful Patterns Reference

### Go Module Initialization

```go
// provider.go (interface definition)
type Provider interface {
    Initialize(ctx context.Context) error
    Close() error
}

// manager.go (lifecycle management)
type Manager struct {
    provider Provider
    mu       sync.Mutex
}

// funasr.go (implementation)
type FunASRProvider struct {
    cfg Config
    conn *websocket.Conn
}

func NewFunASRProvider(cfg Config) *FunASRProvider {
    return &FunASRProvider{cfg: cfg}
}

// init.go (registration)
func init() {
    Register("funasr", func(cfg interface{}) (Provider, error) {
        return NewFunASRProvider(cfg.(Config)), nil
    })
}
```

### React Query + Axios

```tsx
// Mutation example
const { mutate: saveConfig } = useMutation({
  mutationFn: (config: ASRConfig) => saveASRConfig(config),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["asr-config"] });
    toast.success("Config saved");
  },
  onError: (error) => {
    toast.error(`Failed: ${error.message}`);
  },
});
```
