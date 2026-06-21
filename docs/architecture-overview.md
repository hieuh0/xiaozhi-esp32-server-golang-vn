# Architecture Overview

**xiaozhi-esp32-server-golang** — Core system design and components.

---

## High-Level Architecture

```
ESP32 Devices (WebSocket/MQTT/UDP)
    │
    ▼
┌─────────────────────────────────────────┐
│  Main Server (Go, Streaming Pipeline)  │
│  ├─ Transport: IConn abstraction        │
│  ├─ Per-device: ClientState + ChatSession
│  ├─ AI Pipeline: ASR → LLM → TTS       │
│  ├─ Managers: Speaker, Memory, RAG, MCP│
│  └─ Config: Hot-reload from Manager    │
└────────┬────────────────────────────────┘
         │ (Internal HTTP, bearer token)
         ▼
┌─────────────────────────────────────────┐
│  Manager Backend (REST API + WebSocket)│
│  ├─ Device activation & management      │
│  ├─ Config CRUD (ASR, TTS, LLM, etc.)  │
│  ├─ User auth (JWT), roles, quota      │
│  └─ Storage: GORM (SQLite/MySQL)       │
└────────┬────────────────────────────────┘
         │ (React SPA served by backend)
         ▼
┌─────────────────────────────────────────┐
│  Manager Frontend (React + TypeScript)  │
│  ├─ Dashboard (pool stats, device list) │
│  ├─ Config forms (ASR, TTS, LLM, etc.)  │
│  ├─ User management, chat history       │
│  └─ i18n (EN, VI, ZH)                   │
└─────────────────────────────────────────┘
```

---

## Core Components

### 1. Transport Abstraction (`IConn`)

Single interface for all protocols:
- **WebSocket** (port 8989): TCP-based, low-latency
- **MQTT** (port 2883): Pub/sub, reliable delivery
- **UDP** (port 8990): Ultra-low latency (optional)

All three map to uniform: `SendCmd()`, `RecvCmd()`, `SendAudio()`, `RecvAudio()`.

### 2. Per-Device State Machine (ClientState)

```
init → listening → listenStop → llmStart → ttsStart → idle ↻
```

States:
- **listening**: Waiting for user input (VAD active)
- **llmStart**: LLM processing user request
- **ttsStart**: TTS generating response
- **idle**: Waiting for next utterance

Interrupt modes (configurable):
1. VAD-based (fast, may cut off speech)
2. ASR-based (wait for speech detection)
3. Speaker ID-based (personalized)
4. Streaming ASR result (default)

### 3. AI Pipeline (per-device)

```
Audio Input
    ↓ [VAD: Voice Activity Detection]
Silence Detection
    ↓ [ASR: Speech-to-Text]
User Text
    ↓ [Speaker ID + RAG Context]
LLM Input + Context
    ↓ [LLM: Inference]
Response Text
    ↓ [TTS: Text-to-Speech]
Audio Response Output
```

**Key managers**:
- `ASRManager`: Handles multiple ASR providers (FunASR, Doubao, Xunfei)
- `LLMManager`: Handles multiple LLM providers (OpenAI, Ollama, etc.) via Eino framework
- `TTSManager`: Handles multiple TTS providers
- `VADManager`: Voice activity detection
- `SpeakerManager`: Voiceprint recognition + voice selection
- `MemoryManager`: Conversation context retention
- `RAGManager`: Knowledge base retrieval
- `MCPManager`: Tool/plugin management

### 4. Event Hook System

7 stages with observer pattern for metrics, logging, RAG augmentation:

```
ASR Output → LLM Input Preparation → LLM Output → TTS Input Prep → TTS Output
```

Each stage fires events that hooks can intercept to: collect metrics, log, filter, augment context, etc.

### 5. Configuration Hot-Reload

```
Admin updates config (UI)
    ↓
Manager Backend (database)
    ↓
Main Server (periodic poll or HTTP PUT)
    ↓
Semantic Diff (only reload if changed)
    ↓
Affected subsystem reloaded (MQTT, UDP, MCP, providers)
    ↓
Zero downtime (existing chats continue)
```

### 6. MCP 3-Tier Architecture

- **Tier 1 (Local)**: Tools in main server process (filesystem, arithmetic)
- **Tier 2 (Global)**: Separate MCP servers (file, memory, custom)
- **Tier 3 (Device)**: Per-device custom tools

LLM can call any tool during response generation; result integrated into next response.

### 7. Authentication & Authorization

**Device**:
- One-time activation via Manager dashboard
- Device ID + token in connection hello message
- Per-device rate limits and quotas

**User**:
- JWT token (HS256, 24h expiry)
- Attached to every API request
- Role-based access (admin, user, guest)

**Internal**:
- Bearer token for Main Server ↔ Manager Backend communication
- Validates with `config.manager.auth_token`

---

## Data Flow: Audio to Response

```
1. Device sends Opus audio (WebSocket/MQTT)
2. IConn buffers audio, validates device
3. VAD detects speech (Silero ONNX)
4. ASR transcribes (FunASR, Doubao, etc.) → "你好"
5. Speaker ID matches voice → custom TTS voice (optional)
6. RAG searches knowledge base → context (optional)
7. LLM generates response → "你好！我是小智。"
   - Uses conversation history
   - May call MCP tools
   - Returns streaming tokens
8. TTS synthesizes response → Opus audio
9. IConn sends audio back to device
10. Device plays audio
```

**Latency breakdown** (typical):
- VAD: 50-100ms
- ASR: 500ms
- LLM: 800ms
- TTS: 400ms
- **Total: ~1.7s** (excluding network jitter + provider variance)

---

## Database Schema (Simplified)

**Core Tables**:
- `users`: Admin, user accounts, roles
- `devices`: Device registry, activation, ownership
- `configs`: Generic provider config store (ASR, TTS, LLM, VAD, etc.)
- `chat_messages`: Optional conversation history
- `speaker_groups`: Voice clone management
- `knowledge_bases`: RAG integration metadata

**Database types**: SQLite (dev), MySQL (production).

---

## Concurrency Model

**Per-device** (3 goroutines):
- Reader: buffers incoming messages/audio
- Writer: sends queued responses
- State machine: orchestrates ASR/LLM/TTS

**Server-level**:
- One goroutine per device connection (scalable to 1000s)
- Hook worker pool (async event handling)
- Config reload handler (periodic or on-demand)
- Resource cleanup (idle session eviction)

**Synchronization**:
- RWMutex for state machine (per-device)
- Channels for message queues (async hand-off)
- No global locks → excellent scalability

---

## Deployment Topology

**Single server** (docker-compose):
- Main server (8989 WebSocket, 2883 MQTT, 8990 UDP)
- Backend API (8080)
- Frontend (React SPA)
- MySQL / SQLite
- Redis (optional caching)
- Qdrant (optional speaker ID)

**Multi-server** (production):
- Load balancer (HAProxy/NGINX)
- Multiple main-servers (stateless)
- Shared MySQL + Redis
- Central MQTT broker (message bus)
- Shared Qdrant for speaker ID

---

## Provider Ecosystem

| Domain | Providers | Purpose |
|--------|-----------|---------|
| ASR | FunASR, Aliyun Qwen3, Doubao, Xunfei | Speech → Text |
| TTS | EdgeTTS, OpenAI, Doubao, Xunfei, Qwen, Minimax, CosyVoice, Supertonic, Xiaozhi, IndexTTS | Text → Speech |
| LLM | OpenAI, Azure, Ollama, Coze, Dify, Doubao, DeepSeek | Inference (via Eino) |
| VAD | Silero, Tencent TEN, WebRTC | Voice detection |
| Memory | Memobase, Mem0, MemOS | Context retention |
| RAG | Dify, RAGFlow, Weknora | Knowledge search |

---

## Performance Characteristics

**Memory**:
- Per-device idle: ~100 KB
- Per-device active: ~500 KB–1 MB
- 1000 devices: ~100 MB

**CPU**:
- Per-device idle: < 0.01%
- Per-device active: ~0.5-1% (mostly I/O wait)
- 1000 idle on 8-core: ~5% CPU

**Network**:
- Audio (Opus): 8-16 Kbps
- Per-utterance: ~30 KB
- Average per device: ~1-3 Kbps

---

## Security Architecture

**Data Protection**:
- TLS optional (MQTT 8883, reverse proxy HTTPS)
- API keys encrypted in database
- User passwords: bcrypt (salt + hash)
- JWT signed with HS256

**Access Control**:
- Device activation required
- Per-device rate limiting
- User authentication + role-based permissions
- Internal service authentication (bearer token)

**Audit**:
- Device connect/disconnect logged
- Config changes logged (user, timestamp)
- API errors logged (not personal data)
- Chat content not logged (privacy-first)

---

## Future Improvements

- **Proactive AI**: Server-initiated conversations
- **Advanced RBAC**: Fine-grained permissions per device/agent
- **Embedded ML**: On-device LLM inference (Ollama, GGUF)
- **Distributed sessions**: True multi-region failover
- **Advanced caching**: Provider response caching, embeddings cache
- **A/B testing**: Multiple provider configs per device

For detailed diagrams and extended architecture, see **docs/system-architecture.md** (reference).
