# Codebase Summary

**xiaozhi-esp32-server-golang**  
High-performance Go backend for ESP32 voice AI pipeline. Modular architecture with pluggable AI providers.

---

## Directory Structure

```
xiaozhi-esp32-server-golang-vn/
├── cmd/
│   ├── server/
│   │   └── main.go                  # Main server entry point; flags: -c, -manager-enable, -asr-enable
│   ├── mqtt/
│   │   └── *.go                     # MQTT client CLI utility
│   └── mock_ai_server/
│       └── *.go                     # Mock AI provider for testing
│
├── internal/
│   ├── app/
│   │   ├── server/
│   │   │   ├── app.go               # App orchestrator; wires server + MQTT + chat managers
│   │   │   ├── conn.go              # IConn abstraction; WebSocket + MQTT+UDP adapters
│   │   │   ├── websocket_server.go  # WebSocket handler (port 8989)
│   │   │   └── ...                  # Config hot-reload, system config management
│   │   └── mqtt_server/
│   │       ├── mqtt_server.go       # Embedded MQTT broker (mochi-mqtt)
│   │       └── ...
│   │
│   ├── config/
│   │   └── config.go                # Legacy config struct (replaced by Viper + domain/config)
│   │
│   ├── data/
│   │   ├── client/
│   │   │   └── client.go            # ClientState: per-device session state machine
│   │   ├── message/
│   │   │   └── message_types.go     # Protocol messages: hello, listen, stt, tts, llm, goodbye, etc.
│   │   ├── dialogue.go              # Dialogue: thread-safe conversation history
│   │   └── chat_session.go          # ChatSession: orchestrates ASR/LLM/TTS/Speaker managers
│   │
│   ├── domain/                      # 16+ pluggable modules (see next section)
│   │   ├── asr/                     # Speech-to-text providers
│   │   ├── tts/                     # Text-to-speech providers
│   │   ├── llm/                     # Large language model providers
│   │   ├── vad/                     # Voice activity detection
│   │   ├── memory/                  # Conversation memory providers
│   │   ├── mcp/                     # Model Context Protocol (3-tier)
│   │   ├── speaker/                 # Speaker identification & voice selection
│   │   ├── rag/                     # Knowledge base / RAG searchers
│   │   ├── chat/                    # Chat hooks & event handlers
│   │   ├── openclaw/                # OpenClaw agent endpoint integration
│   │   ├── config/                  # Configuration management (manager, redis, local, http)
│   │   ├── audio/                   # Opus codec wrapper
│   │   ├── eventbus/                # Pub/sub event bus
│   │   ├── play_music/              # Audio playback control
│   │   └── doubaoapi/               # ByteDance proprietary integration
│   │
│   ├── pkg/
│   │   └── *.go                     # Shared utilities: chat hooks, transforms
│   │
│   └── util/
│       └── *.go                     # Helpers: queue, pool, sentence boundary, crypto
│
├── manager/
│   ├── backend/                     # REST API server (Go + Gin + GORM)
│   │   ├── main.go                  # Entry point (port 8080)
│   │   ├── controllers/             # HTTP handlers (devices, configs, auth, pool stats)
│   │   ├── models/                  # GORM data models
│   │   ├── services/                # Business logic (config sync, auth, device activation)
│   │   ├── middleware/              # JWT auth, internal service token
│   │   ├── router/                  # Route registration
│   │   ├── storage/                 # Repository pattern (GORM)
│   │   ├── database/                # DB schema, migrations
│   │   └── config/
│   │       └── config.json          # Backend config (DB, port, JWT secret, etc.)
│   │
│   └── frontend/                    # React SPA (TypeScript + TanStack Router)
│       └── src/
│           ├── routes/              # TanStack file-based routes (dashboard, admin, user)
│           ├── components/          # React components (forms, charts, layout)
│           ├── i18n/                # Translations (EN, VI, ZH)
│           ├── styles/              # Tailwind CSS globals
│           └── ...
│
├── config/
│   ├── config.yaml                  # Main server config (Viper YAML)
│   └── config.personal.yaml         # Local dev overrides (git-ignored)
│
├── docker/
│   ├── docker-compose.yml           # Production compose
│   ├── docker-compose.local.yml     # Local dev compose
│   ├── Dockerfile.funasr            # FunASR server image
│   └── ...
│
├── asr_server/                      # FunASR integration (Git submodule)
│   └── ...
│
├── docs/
│   ├── project-overview-pdr.md      # This document structure
│   ├── codebase-summary.md          # You are here
│   ├── code-standards.md            # Code style, patterns, error handling
│   ├── system-architecture.md       # Architecture diagrams, data flow
│   ├── project-roadmap.md           # Current state, planned features
│   ├── deployment-guide.md          # Docker, env setup, scaling
│   ├── local-dev-guide-vi.md        # Vietnamese local dev guide (Windows)
│   └── mqtt-broker-setup-guide.md   # MQTT setup instructions
│
├── go.mod / go.sum                  # Go module dependencies
├── README.md                        # Vietnamese project README
├── LICENSE                          # MIT
└── docker-local.sh / docker-local.ps1 # Local dev Docker helpers
```

---

## Core Modules (internal/domain)

| Module | Purpose | Key Providers | Files |
|--------|---------|---|---|
| **asr** | Speech-to-text | FunASR, Aliyun Qwen3, Doubao, Xunfei | provider.go, funasr.go, aliyun.go, etc. |
| **tts** | Text-to-speech | EdgeTTS, EdgeOffline, OpenAI, Doubao, Xunfei, Qwen, Minimax, CosyVoice, Supertonic, Xiaozhi, IndexTTS | provider.go, edge_tts.go, openai.go, doubao.go, etc. |
| **llm** | Language model inference | OpenAI, Azure, Ollama, Anthropic, Coze, Dify, Zhipu, Doubao, DeepSeek (Cloudwego Eino) | provider.go, eino_*.go, response_transformer.go |
| **vad** | Voice activity detection | Silero VAD, Tencent TEN VAD, WebRTC VAD | provider.go, silero.go, ten_vad.go, webrtc_vad.go |
| **memory** | Conversation memory | None (stateless), Memobase, Mem0, MemOS | provider.go, memobase.go, mem0.go, memos.go |
| **mcp** | Model Context Protocol | Local (process), Global (server), Device (per-client) | manager.go, sse_handler.go, ws_handler.go, protocol.go |
| **speaker** | Speaker ID + voice selection | sherpa-onnx + vector DB (Qdrant/Milvus) | manager.go, speaker_manager.go, embedding.go |
| **rag** | Knowledge base search | Dify, RAGFlow, Weknora | searcher.go, dify.go, ragflow.go, weknora.go |
| **chat** | Event hooks & transforms | 7 interceptors (ASR/LLM/TTS stages) | hooks.go, transforms.go, metrics.go |
| **openclaw** | Agent endpoint | ByteDance proprietary | manager.go, endpoint.go, router.go |
| **config** | Config management | Manager (HTTP), Redis, Local, HTTP | provider.go, manager.go, redis.go, local.go |
| **audio** | Audio codec | Opus wrapper | encoder.go, decoder.go |
| **eventbus** | Pub/sub events | Internal memory-based bus | bus.go, subscription.go |
| **play_music** | Audio playback control | MCP resource handler | manager.go |
| **doubaoapi** | ByteDance proprietary | Doubao LLM, TTS, ASR | doubaoapi.go |

---

## Key Design Patterns

### Provider Pattern
All AI providers follow a common interface:

```go
type Provider interface {
    // Config returns provider-specific configuration structure
    Config() interface{}
    // Initialize loads model and establishes connections
    Initialize(ctx context.Context) error
    // Inference / Synthesis runs the provider
    // (method name varies: StreamChat, Synthesize, Recognize, etc.)
    // Close releases resources
    Close() error
}
```

Example: `asr/provider.go` defines `Provider` interface; `asr/funasr.go` implements it.

### Transport Abstraction (IConn)
All device connections implement `IConn` interface defined in `app/server/conn.go`:

```go
type IConn interface {
    SendCmd(msg []byte) error                               // Send JSON command
    RecvCmd(ctx context.Context, timeout int) ([]byte, error) // Receive JSON command
    SendAudio(audio []byte) error                           // Send Opus audio
    RecvAudio(ctx context.Context, timeout int) ([]byte, error) // Receive audio
    GetDeviceID() string
    GetTransportType() string                               // "websocket", "mqtt", "udp"
    Close() error
    OnClose(func(deviceId string))
}
```

Adapters: `WebSocketConn`, `MqttUdpConn` both implement `IConn`.

### State Machine (ClientState)
Per-device state defined in `data/client/client.go`:

```
init → listening → listenStop → llmStart → ttsStart → idle
       (↻ realtime mode: VAD/ASR can interrupt listening)
```

Transitions based on events: ASR result, VAD silence, TTS completion, user abort.

### Event Hooks
7 chat stages emit events; observers (hooks) can intercept:

- `EventChatASROutput`: ASR result ready
- `EventChatLLMInput`: LLM receiving input
- `EventChatLLMOutputRaw`: LLM raw output
- `EventChatLLMOutput`: Processed LLM output
- `EventChatTTSInput`: TTS receiving text
- `EventChatTTSOutputStart`: TTS response streaming
- `EventChatTTSOutputStop`: TTS complete

Used for metrics, logging, RAG augmentation, etc.

### Configuration Hot-Reload
Viper-based config with semantic diffing:

1. Manager backend writes config change to database
2. Main server polls periodically (default 5m)
3. Semantic diff compares old/new values
4. Only changed subsystems reinitialized (MQTT, UDP, MCP, etc.)

### MCP 3-Tier Architecture
- **Tier 1 (Local)**: Tools running in main server process
- **Tier 2 (Global)**: MCP servers in separate processes; shared across all devices
- **Tier 3 (Device)**: Per-client MCP endpoints; device-specific tools

Unified protocol: SSE or WebSocket for communication.

---

## Key Files Reference

### Main Server

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Entry point; CLI flags; graceful shutdown |
| `internal/app/server/app.go` | App struct; orchestrates server, MQTT, chat managers |
| `internal/app/server/websocket_server.go` | WebSocket listener (port 8989) |
| `internal/app/mqtt_server/mqtt_server.go` | Embedded MQTT broker (mochi-mqtt) |
| `internal/data/client/client.go` | ClientState: per-device session state |
| `internal/data/chat_session.go` | ChatSession: coordinates ASR/LLM/TTS |
| `internal/data/message/message_types.go` | Protocol message definitions |

### Domain Modules (Example: ASR)

| File | Purpose |
|------|---------|
| `internal/domain/asr/provider.go` | ASR provider interface |
| `internal/domain/asr/funasr.go` | FunASR WebSocket client |
| `internal/domain/asr/manager.go` | ASRManager: maintains provider instance |

(Pattern repeats for TTS, LLM, VAD, Memory, etc.)

### Manager Backend

| File | Purpose |
|------|---------|
| `manager/backend/main.go` | Gin router, GORM init, server start |
| `manager/backend/controllers/device_activation.go` | Device registration & activation |
| `manager/backend/controllers/config.go` | Config CRUD (ASR, TTS, LLM, VAD, etc.) |
| `manager/backend/services/config_service.go` | Config sync to main server |
| `manager/backend/models/device.go` | GORM Device model |
| `manager/backend/router/router.go` | Route registration |

### Frontend

| File | Purpose |
|------|---------|
| `manager/frontend/src/routes/_auth/_layout/dashboard.tsx` | Main dashboard page |
| `manager/frontend/src/components/admin/asr-config-form.tsx` | ASR config editor |
| `manager/frontend/src/components/admin/tts-config-form.tsx` | TTS config editor |
| `manager/frontend/src/i18n/en.ts` | English translations |
| `manager/frontend/src/i18n/vi.ts` | Vietnamese translations |

---

## External Dependencies (Go)

### Core Frameworks
- `github.com/gin-gonic/gin` - Web framework
- `gorm.io/gorm` - ORM
- `github.com/spf13/viper` - Configuration management
- `github.com/sirupsen/logrus` - Structured logging

### AI & ML
- `github.com/cloudwego/eino` - ByteDance LLM framework (OpenAI, Ollama, Doubao, etc.)
- `github.com/cloudwego/eino-ext/components/model/openai` - OpenAI integration
- `github.com/cloudwego/eino-ext/components/model/ollama` - Ollama integration
- `github.com/ThinkInAIXYZ/go-mcp` - MCP protocol
- `github.com/mark3labs/mcp-go` - MCP implementation

### Audio
- `github.com/hraban/opus` - Opus codec
- `github.com/go-audio/audio` - Audio utilities
- `github.com/hackers365/silero-vad-go` - Silero VAD bindings
- `github.com/hackers365/go-webrtcvad` - WebRTC VAD bindings
- `github.com/hackers365/silero-vad-go` - TEN VAD bindings

### Communication
- `github.com/gorilla/websocket` - WebSocket server
- `github.com/eclipse/paho.mqtt.golang` - MQTT client
- `github.com/mochi-mqtt/server/v2` - Embedded MQTT broker
- `github.com/tmaxmax/go-sse` - Server-Sent Events

### Storage & Caching
- `github.com/redis/go-redis/v9` - Redis client
- Various GORM database drivers

### Utilities
- `github.com/google/uuid` - UUID generation
- `github.com/golang-jwt/jwt/v4` - JWT
- `github.com/bytedance/sonic` - Fast JSON (replacing stdlib)
- `github.com/lestrrat-go/file-rotatelogs` - Log rotation

---

## Frontend Dependencies (npm)

### Core
- `react@19` - UI library
- `typescript@5` - Type system
- `@tanstack/react-router@1` - File-based routing
- `@tanstack/react-query@5` - Data fetching & caching

### UI & Styling
- `tailwindcss@4` - Utility CSS
- `@radix-ui/*` - Headless components
- `shadcn/ui` - Pre-built Radix components
- `recharts` - Charts library

### Forms & Validation
- `react-hook-form` - Form state management
- `zod` - Schema validation

### Localization
- `i18next` - i18n framework
- `react-i18next` - React bindings

### HTTP
- `axios` - HTTP client

---

## Module Dependency Graph

```
ChatSession (orchestrator)
    ├── ASRManager
    │   └── ASR Provider (FunASR, Doubao, etc.)
    ├── LLMManager
    │   └── LLM Provider (OpenAI, Ollama, etc.)
    │       ├── RAG Searcher (optional; augments context)
    │       └── Memory Manager (optional; stores/retrieves context)
    ├── TTSManager
    │   └── TTS Provider (EdgeTTS, OpenAI, etc.)
    ├── VADManager
    │   └── VAD Provider (Silero, TEN, WebRTC)
    ├── SpeakerManager
    │   ├── Speaker Embedding (sherpa-onnx)
    │   └── Vector DB (Qdrant/Milvus)
    └── Chat Hooks (metrics, logging, RAG augmentation)

App (server)
    ├── WebSocket Server (IConn)
    ├── MQTT Server (embedded) + MQTT UDP Client (IConn)
    ├── Device Manager (ChatSession per device)
    ├── Config Manager (hot-reload)
    ├── MCP Manager (3-tier)
    └── OpenClaw Endpoint

Manager Backend (REST API)
    ├── Device Controller
    ├── Config Controller
    ├── Auth Middleware (JWT)
    └── Storage (GORM)

Manager Frontend (React SPA)
    ├── Dashboard (device status, pool stats)
    ├── Config Forms (ASR, TTS, LLM, VAD, etc.)
    ├── Device Management
    └── Admin Panel
```

---

## Key Configuration Sections

### Server Config (config.yaml)

| Section | Purpose | Example |
|---------|---------|---------|
| `server.pprof` | CPU/memory profiling | `enable: true, port: 6060` |
| `auth` | Enable/disable auth | `enable: true` |
| `chat` | Pipeline behavior | `max_idle_duration: 30000ms`, realtime_mode |
| `config_provider` | Config source | `type: manager`, `update_interval: 5m` |
| `manager` | Backend API | `backend_url, auth_token` |
| `websocket` | WebSocket listener | `port: 8989` |
| `mqtt` | MQTT client (external broker) | `broker, port, auth` |
| `mqtt_server` | Embedded MQTT broker | `listen_port: 2883` |
| `udp` | UDP server | `listen_port: 8990` |
| `vad/asr/tts/llm` | Provider configs | Provider-specific settings (API keys, models) |
| `mcp/local_mcp` | MCP servers | Endpoint URLs, ports |
| `memory` | Memory provider | `provider: memobase`, config |
| `rag` | Knowledge base | `provider: dify`, config |

---

## Build & Runtime

### Compile Main Server
```bash
go build -o xiaozhi_server ./cmd/server/
```

### Compile Manager Backend
```bash
cd manager/backend
go build -o backend main.go
```

### Compile Manager Frontend
```bash
cd manager/frontend
npm install
npm run build  # Outputs to dist/
```

### Run with Docker
```bash
docker-compose -f docker/docker-compose.yml up -d
```

---

## Testing Approach

- Unit tests: `*_test.go` files alongside implementation
- Integration tests: Mock AI providers in `cmd/mock_ai_server/`
- Health checks: `/api/health` endpoint in manager backend
- Load testing: Metrics dashboard shows active connections, latency P50/P95/P99

---

## Code Quality

- **Linting**: No explicit linter config; follows Go conventions
- **Type safety**: Full type system; no dynamic types except JSON unmarshaling
- **Error handling**: Explicit error returns; no panics in critical paths
- **Logging**: Structured logs via logrus; sensitive data filtered
- **Documentation**: Comments on exported functions; architecture docs in `docs/`

---

## Performance Characteristics

- **Per-device memory**: ~100-500 KB (idle), ~1-2 MB (active chat)
- **CPU per device**: Negligible (< 1% on 2-core at 100 devices)
- **Database writes**: Config changes, device state, chat history (optional)
- **Network I/O**: Opus audio (8-16 Kbps typical), JSON commands (< 1 KB)

---

## Known Limitations

- **Single-server**: Horizontal scaling requires MQTT broker for state sync (session affinity)
- **Audio formats**: Opus primary; PCM fallback; no MP3, WAV, etc.
- **LLM providers**: Depends on external APIs (OpenAI, Doubao, etc.); no on-device inference by default
- **Speaker ID**: Requires vector DB setup; not included in minimal Docker Compose
