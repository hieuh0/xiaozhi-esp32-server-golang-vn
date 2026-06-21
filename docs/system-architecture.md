# System Architecture

**xiaozhi-esp32-server-golang**  
A modular, streaming voice AI backend with multi-provider support and pluggable components.

---

## 1. System Context Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         External Services                        │
├─────────────────────────────────────────────────────────────────┤
│ ASR Providers: FunASR, Doubao, Xunfei                           │
│ LLM Providers: OpenAI, Azure, Ollama, Doubao, Coze, Dify        │
│ TTS Providers: EdgeTTS, OpenAI, Doubao, Xunfei, Minimax, etc.   │
│ VAD Models: Silero ONNX, Tencent TEN                            │
│ Vector DB: Qdrant (speaker embedding)                           │
│ Knowledge Base: Dify, RAGFlow, Weknora                          │
│ MCP Servers: File, Memory, Custom Tools                         │
└─────────────────────────────────────────────────────────────────┘
                             ▲ API calls
                             │
                             │ (REST, WebSocket, gRPC)
                             │
┌─────────────────────────────────────────────────────────────────┐
│             xiaozhi-esp32-server-golang (Main Server)           │
├─────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────────────────────────────┐   │
│ │            Transport Layer (Multi-Protocol)              │   │
│ │  ┌────────────────┬──────────────────┬────────────────┐  │   │
│ │  │  WebSocket     │  MQTT + UDP      │  Reserved      │  │   │
│ │  │  (port 8989)   │  (2883, 8990)    │  (future)      │  │   │
│ │  └────────────────┴──────────────────┴────────────────┘  │   │
│ │                         ↑                                  │   │
│ │                  IConn Abstraction                        │   │
│ └──────────────────────────────────────────────────────────┘   │
│                             ▲                                    │
│                             │                                    │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │             Device Manager & Chat Sessions              │   │
│ │  One ChatSession per connected device (per-device state) │   │
│ │  ├─ ClientState (FSM: init→listening→llm→tts→idle)      │   │
│ │  └─ Dialogue (conversation history + context)           │   │
│ └──────────────────────────────────────────────────────────┘   │
│                             ▲                                    │
│                             │                                    │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │         AI Pipeline (Streaming, per-device)             │   │
│ │  ┌────────────┬──────────┬──────────┬─────────────┐    │   │
│ │  │  VADMgr    │ ASRMgr   │ LLMMgr   │  TTSMgr     │    │   │
│ │  │ (activity) │ (speech) │ (reason) │  (response) │    │   │
│ │  └────────────┴──────────┴──────────┴─────────────┘    │   │
│ │         ↓          ↓         ↓            ↓             │   │
│ │  ┌──────────────────────────────────────────────────┐   │   │
│ │  │   Event Hooks (7 stages, 12 metrics pipeline)   │   │   │
│ │  │   - ASR output, LLM input/output, TTS in/out    │   │   │
│ │  │   - For metrics, logging, RAG, speaker ID       │   │   │
│ │  └──────────────────────────────────────────────────┘   │   │
│ │                                                          │   │
│ │  ┌──────────────────────────────────────────────────┐   │   │
│ │  │   Supporting Managers                            │   │   │
│ │  │   - SpeakerManager (voiceprint → voice clone)   │   │   │
│ │  │   - MemoryManager (context retention)           │   │   │
│ │  │   - RAGManager (knowledge base search)          │   │   │
│ │  │   - MCPManager (3-tier tool servers)            │   │   │
│ │  │   - OpenClawManager (agent endpoint)            │   │   │
│ │  └──────────────────────────────────────────────────┘   │   │
│ └──────────────────────────────────────────────────────────┘   │
│                             ▲                                    │
│                             │                                    │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │       Configuration & Lifecycle Management              │   │
│ │  - Hot-reload handler (semantic diff)                   │   │
│ │  - Provider initialization & cleanup                    │   │
│ │  - Graceful shutdown (close all connections)           │   │
│ │  - Profiling (pprof on configurable port)             │   │
│ └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
         │                              ▲
         │                              │
         │ (Internal HTTP Token Auth)  │
         │                              │
         ▼                              │
┌─────────────────────────────────────────────────────────────────┐
│         Manager Backend REST API & WebSocket                    │
│         (Go + Gin + GORM + SQLite/MySQL)                        │
├─────────────────────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────────────────┐     │
│ │ Routes:                                                 │     │
│ │  - POST   /api/devices/{id}/activate       (device reg)│     │
│ │  - GET    /api/devices                     (list)      │     │
│ │  - GET    /admin/asr-config                (read)      │     │
│ │  - POST   /admin/asr-config                (write)     │     │
│ │  - GET    /admin/pool-stats                (metrics)   │     │
│ │  - WS     /ws                              (broadcast) │     │
│ │  - POST   /login, /register                (auth)      │     │
│ │  - POST   /open/v1/chat (OpenAPI)          (public)    │     │
│ └────────────────────────────────────────────────────────┘     │
│                                                                  │
│ ┌────────────────────────────────────────────────────────┐     │
│ │ Storage Layer (GORM Repository Pattern)                │     │
│ │  - Device, User, Config, ChatMessage models           │     │
│ │  - Config sync service (periodic pull from DB)        │     │
│ │  - Auth service (JWT, bcrypt)                         │     │
│ └────────────────────────────────────────────────────────┘     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                             ▲
                             │
                      (SQLite/MySQL)
                             │
┌─────────────────────────────────────────────────────────────────┐
│  Database (Persistent State)                                    │
│  - Device registry, user accounts, roles                        │
│  - Config snapshots (provider settings)                         │
│  - Chat message history (optional)                              │
│  - Speaker metadata + voice clones                              │
│  - Knowledge base references                                    │
│  - API tokens, OTA update state                                 │
└─────────────────────────────────────────────────────────────────┘

         │
         │ (Optional: Redis cache layer)
         │
         ▼
    ┌─────────────────┐
    │ Redis           │
    │ - Session cache │
    │ - Rate limits   │
    │ - Chat history  │
    └─────────────────┘
         │
         │ (Optional: Embedded in main server)
         │
         ▼
    ┌─────────────────┐
    │ MQTT Broker     │
    │ (mochi-mqtt)    │
    │ Device-to-server│
    │ pub/sub channel │
    └─────────────────┘
```

---

## 2. Data Flow: Audio to Response

```
Device (ESP32)
    ↓
    │ [1] Opus Audio Stream (port 8989 WebSocket OR 2883 MQTT)
    ▼
┌──────────────────────────────────────────────────────┐
│         IConn (Transport Abstraction)                │
│  - Buffers audio chunks                             │
│  - Validates device authentication                  │
│  - Keeps connection alive (heartbeat)               │
└──────────────────────────────────────────────────────┘
    ↓
    │ [2] Audio Decoded (Opus → PCM float32)
    ▼
┌──────────────────────────────────────────────────────┐
│         VADManager                                   │
│  - Voice activity detection (Silero ONNX)          │
│  - Detects speech start/end                        │
│  - Configurable silence threshold (default 400ms)   │
└──────────────────────────────────────────────────────┘
    ↓ (When VAD detects speech end)
    │ [3] Audio Buffer Accumulated
    ▼
┌──────────────────────────────────────────────────────┐
│         ASRManager                                   │
│  - Speech-to-text transcription                     │
│  - Provider: FunASR, Doubao, Xunfei, etc.          │
│  - May support streaming (2-pass) or offline mode   │
│  - Latency: ~500ms                                  │
│  Event: EventChatASROutput (observed by hooks)      │
└──────────────────────────────────────────────────────┘
    ↓ (ASR result: "你好")
    │ [4] ASR Text + Metadata
    ▼
┌──────────────────────────────────────────────────────┐
│         SpeakerManager (Optional)                    │
│  - Generate speaker embedding (sherpa-onnx)        │
│  - Match against vector DB (Qdrant)                │
│  - Lookup custom TTS voice for speaker             │
│  - If found: speaker_id, voice_name                │
└──────────────────────────────────────────────────────┘
    ↓ (Speaker ID optional; may skip)
    │ [5] ASR Text + Optional Speaker Context
    ▼
┌──────────────────────────────────────────────────────┐
│         Chat Hooks: EventChatLLMInput               │
│  - Observers can augment input (e.g., RAG context)  │
│  - RAGManager searches knowledge base               │
│  - Inject relevant documents into LLM prompt        │
└──────────────────────────────────────────────────────┘
    ↓ (Enriched input ready)
    │ [6] Input Text + Context + Chat History
    ▼
┌──────────────────────────────────────────────────────┐
│         LLMManager                                   │
│  - Language model inference (streaming)             │
│  - Provider: OpenAI, Ollama, Doubao, Coze, etc.     │
│  - Accesses conversation history (Dialogue)         │
│  - May use tools (MCP)                              │
│  - Latency: ~800ms (model + network)                │
│  Events:                                            │
│   - EventChatLLMOutputRaw (raw tokens)              │
│   - EventChatLLMOutput (final response)             │
└──────────────────────────────────────────────────────┘
    ↓ (LLM response: "你好！我是小智。")
    │ [7] LLM Text Response (may be chunked)
    ▼
┌──────────────────────────────────────────────────────┐
│         Chat Hooks: EventChatTTSInput               │
│  - Transform text (punctuation, emphasis)           │
│  - Select TTS voice (default or speaker-matched)    │
│  - Prepare for synthesis                            │
└──────────────────────────────────────────────────────┘
    ↓ (Text + Voice Selected)
    │ [8] Text + Voice Configuration
    ▼
┌──────────────────────────────────────────────────────┐
│         TTSManager                                   │
│  - Text-to-speech synthesis                        │
│  - Provider: EdgeTTS, OpenAI, Doubao, etc.          │
│  - Streaming or chunked delivery                    │
│  - Latency: ~400ms                                  │
│  Events:                                            │
│   - EventChatTTSOutputStart (first chunk)           │
│   - EventChatTTSOutputStop (complete)               │
└──────────────────────────────────────────────────────┘
    ↓ (TTS response: Opus audio chunks)
    │ [9] Opus Audio Response Stream
    ▼
┌──────────────────────────────────────────────────────┐
│         IConn.SendAudio()                            │
│  - Encode response as Opus                          │
│  - Send over WebSocket/MQTT                         │
│  - Handle backpressure & client buffer              │
└──────────────────────────────────────────────────────┘
    ↓ (Sent back to device)
    │ [10] Audio Received by Device
    ▼
Device (ESP32) — Plays Audio
    ↓
    │ (Device may send new audio, loop repeats)
    ▼
[Back to Step 1: VAD Detection]
```

**Latency Summary**:
- ASR: ~500ms
- LLM: ~800ms
- TTS: ~400ms
- **Total: ~1.7s** (excluding network & provider latency variance)

---

## 3. Multi-Protocol Transport Abstraction

### IConn Interface

All device connections implement a single abstraction:

```
Device connects via:
├── WebSocket (TCP, port 8989)
│   └── WebSocketConn{} ← implements IConn
├── MQTT (TCP, port 2883)
│   └── MqttConn{} ← implements IConn
└── UDP (port 8990)
    └── UdpConn{} ← implements IConn

All three map to:
  SendCmd(msg []byte) → device receives JSON command
  RecvCmd(timeout int) → server polls device JSON
  SendAudio(audio []byte) → device receives audio
  RecvAudio(timeout int) → server polls audio
```

**Benefits**:
- ChatSession code is transport-agnostic
- Add new protocol → only implement IConn once
- Message routing determined by transport type at connection time

---

## 4. Device State Machine (ClientState)

```
                     ┌─────────────────┐
                     │     init        │
                     │ (connected,     │
                     │  hello handshake)
                     └────────┬────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   listening      │
                    │ (waiting for user)
                    └────────┬─────────┘
                             │
        (VAD detects speech) │ (if realtime_mode=4)
        (ASR result arrived) │ (interrupt on ASR)
        (abort command)      │ (user cancellation)
                             │
                    ┌────────▼────────┐
                    │  listenStop     │
                    │ (speech ended)   │
                    └────────┬────────┘
                             │
         (VAD silence ok)     │
         (ASR done)           │
                             │
                    ┌────────▼────────┐
                    │   llmStart      │
                    │ (LLM inferring) │
                    └────────┬────────┘
                             │
        (LLM output ready)   │
                             │
                    ┌────────▼────────┐
                    │   ttsStart      │
                    │ (TTS streaming) │
                    └────────┬────────┘
                             │
         (TTS complete)      │
         (user abort)        │
                             │
                    ┌────────▼────────┐
                    │     idle        │
                    │ (waiting again) │
                    └────────┬────────┘
                             │
                             │ (back to listening)
                             ▼
                    (loop repeats)
```

**Key State Transitions**:
- `listening → listenStop`: VAD silence or explicit abort
- `listenStop → llmStart`: ASR completes, LLM begins
- `llmStart → ttsStart`: LLM response ready
- `ttsStart → idle`: TTS completes
- `idle → listening`: Repeat for next utterance

**Interrupt Modes** (configurable `realtime_mode`):
- Mode 1: VAD interruption (fast but may cut off speech)
- Mode 2: ASR interruption (wait for ASR to confirm speech end)
- Mode 3: Interrupt when ASR identifies voiceprint (speaker-aware)
- Mode 4: Interrupt on ASR result (default; stream processing)

---

## 5. Configuration Hot-Reload Flow

```
┌─────────────────────────────────┐
│ Admin updates config in UI       │
│ (e.g., change TTS provider)      │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│ Manager Backend                  │
│ POST /api/admin/tts-config       │
│ ├─ Validate config (schema)     │
│ ├─ Store in database (Config)   │
│ └─ Notify main server (HTTP PUT) │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│ Main Server                      │
│ ConfigUpdateHandler (HTTP PUT)   │
│ ├─ Parse new config             │
│ ├─ Semantic diff (compare old)   │
│ │  if same → ignore              │
│ │  if different → reload          │
│ └─ Apply changes                 │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│ Affected Subsystems             │
│ ├─ TTS: Close old provider       │
│ │       Initialize new provider  │
│ │       Existing chats continue  │
│ │       New chats use new config │
│ ├─ MQTT: Reconnect to new broker│
│ └─ UDP: Rebind to new listen    │
│       address if changed         │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│ Result: Config updated at runtime│
│ No restart needed                │
│ Active connections unaffected    │
└─────────────────────────────────┘
```

**Periodic Sync**:
- Every 5 minutes (configurable), main server polls manager:
  - `GET /internal/system-config`
  - Compare with current config
  - If changed, apply changes

---

## 6. Authentication & Authorization

### Device Activation Flow

```
Device (ESP32)
    │ [1] Device sends hello with activation token
    │     (or needs activation first)
    ▼
┌──────────────────────────────────┐
│ Main Server WebSocket Handler    │
│ - Extract device ID from hello   │
│ - Verify against known devices   │
└────────────┬───────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│ Manager Backend                  │
│ GET /internal/device/{id}/status │
│ (uses internal auth token)        │
│ Returns: DeviceConfig + status   │
└────────────┬───────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│ Database Lookup                  │
│ - Device model (id, user_id,     │
│   agent_id, config)              │
│ - User model (id, role, status) │
└────────────┬───────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│ Decision:                        │
│ ├─ Device unknown → reject       │
│ ├─ Device disabled → reject      │
│ ├─ User inactive → reject        │
│ └─ All OK → accept + load config│
└────────────┬───────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│ Device Connected                 │
│ - Create ChatSession             │
│ - Load device config (ASR, LLM,  │
│   TTS, etc.)                     │
│ - Ready for commands             │
└──────────────────────────────────┘
```

### API Authentication

**JWT Token Flow**:

```
User
│ POST /login (email, password)
└─→ Manager Backend
    │ [1] Verify password (bcrypt)
    │ [2] Generate JWT (HS256, 24h expiry)
    │ [3] Return token + refresh_token
    │
    └─→ Client stores in localStorage
        │ X-Authorization: Bearer {token}
        │ Attach to every request
        │
        └─→ Manager Backend (JWT Middleware)
            │ [1] Extract token from header
            │ [2] Verify signature (secret from config)
            │ [3] Check expiry
            │ [4] Extract user_id + role
            │ [5] Attach to request context
            │
            └─→ Route handler (authorized)
                │ Access req.Context().UserID
                │ Check req.Context().Role for permission
```

**Internal Service Authentication**:

```
Main Server → Manager Backend (HTTP calls)
  X-Authorization: Bearer {internal_auth_token}

Middleware verifies:
  - Token matches config.manager.auth_token
  - Request from known IP (optional)
```

---

## 7. MCP 3-Tier Architecture

```
┌──────────────────────────────────────────────────────┐
│ Tier 1: Local MCP (within main server process)      │
├──────────────────────────────────────────────────────┤
│ Tools:                                               │
│ - File system access (read, write, search)          │
│ - Arithmetic / calculator                           │
│ - Web search (if configured)                        │
│                                                      │
│ Protocol: Direct function calls (no network)        │
│ Scope: Available to all devices on this server      │
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│ Tier 2: Global MCP (separate server pool)           │
├──────────────────────────────────────────────────────┤
│ Servers:                                             │
│ - mcp-filesystem (port 3001)                        │
│ - mcp-memory (port 3002)                            │
│ - Custom tools (e.g., email, database)             │
│                                                      │
│ Protocol: SSE or WebSocket (over network)           │
│ Scope: Shared across all main servers + devices    │
│ Auto-discovery: Via /internal/mcp/tools             │
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│ Tier 3: Device MCP (per-device custom endpoint)     │
├──────────────────────────────────────────────────────┤
│ Services:                                            │
│ - Device-specific tools (e.g., GPIO, sensor)       │
│ - User-uploaded scripts                             │
│ - Voice clone tools                                 │
│                                                      │
│ Protocol: Device-specific (REST, WebSocket, etc.)   │
│ Scope: Only for that device                         │
│ Lookup: Via device config (mcp.device_endpoint)     │
└──────────────────────────────────────────────────────┘

                         │
                         │ Unified Tool Registry
                         │
                    ┌────▼─────┐
                    │ MCPManager│
                    │ - discover
                    │ - cache
                    │ - call tools
                    └────┬─────┘
                         │
                    ┌────▼─────────┐
                    │ LLMManager    │
                    │ - use tools   │
                    │ - update      │
                    │ - response    │
                    └───────────────┘
```

**Tool Invocation Flow**:

```
LLM generates response with tool call:
  {
    "type": "tool_call",
    "name": "file_read",
    "params": {"path": "/etc/hostname"}
  }

MCPManager receives call:
  ├─ [1] Look up tool in registry (all tiers)
  ├─ [2] Route to appropriate tier
  │  ├─ Tier 1: Direct call
  │  ├─ Tier 2: HTTP POST to MCP server
  │  └─ Tier 3: Forward to device endpoint
  ├─ [3] Get result
  ├─ [4] Return to LLM for next response
  │
  └─ [5] LLM continues with tool result in context

Result: LLM response integrates tool output seamlessly
```

---

## 8. Database Schema (Simplified)

### Core Tables

```sql
-- Users
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR UNIQUE,
  password_hash VARCHAR,
  role ENUM('admin', 'user'),
  status ENUM('active', 'disabled'),
  created_at TIMESTAMP
);

-- Devices
CREATE TABLE devices (
  id VARCHAR PRIMARY KEY,          -- e.g., "esp32_12345"
  user_id UUID FOREIGN KEY,        -- owner
  agent_id VARCHAR,                -- default agent for this device
  name VARCHAR,
  transport_type VARCHAR,          -- "websocket", "mqtt", "udp"
  last_connected TIMESTAMP,
  status ENUM('online', 'offline', 'idle'),
  created_at TIMESTAMP
);

-- Configs (generic key-value store)
CREATE TABLE configs (
  id UUID PRIMARY KEY,
  type VARCHAR,                    -- "asr", "tts", "llm", "vad", etc.
  provider VARCHAR,               -- "funasr", "openai", etc.
  data JSON,                      -- provider-specific config
  created_by UUID FOREIGN KEY,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- Chat Messages (optional; can be in Redis instead)
CREATE TABLE chat_messages (
  id UUID PRIMARY KEY,
  device_id VARCHAR FOREIGN KEY,
  role VARCHAR,                   -- "user", "assistant"
  content TEXT,
  created_at TIMESTAMP,
  metadata JSON                   -- latency, provider, etc.
);

-- Speaker Groups (voice clone management)
CREATE TABLE speaker_groups (
  id UUID PRIMARY KEY,
  user_id UUID FOREIGN KEY,
  name VARCHAR,
  speaker_embeddings JSON,        -- stored embeddings
  created_at TIMESTAMP
);

-- Knowledge Base References
CREATE TABLE knowledge_bases (
  id UUID PRIMARY KEY,
  user_id UUID FOREIGN KEY,
  provider VARCHAR,               -- "dify", "ragflow", "weknora"
  endpoint VARCHAR,
  api_key VARCHAR,                -- encrypted
  synced_at TIMESTAMP
);
```

---

## 9. Event Hook System

Seven interceptors for observing/modifying chat stages:

```
Device Audio Input
    ↓
VAD [no hooks]
    ↓
ASR
    ├─ EventChatASROutput (observers see: asr_result, confidence, duration)
    └─ Hook can: log, update metrics, filter results
         ↓
Speaker ID [no hooks]
         ↓
LLM Input Preparation
    ├─ EventChatLLMInput (observers see: user_text, conversation_history)
    └─ Hook can: augment with RAG context, apply filters
         ↓
LLM
    ├─ EventChatLLMOutputRaw (observers see: token stream)
    └─ Hook can: real-time metrics, token counting
         ↓
    ├─ EventChatLLMOutput (observers see: final_response, tokens_used)
    └─ Hook can: cache response, log quality metrics
         ↓
TTS Input Preparation
    ├─ EventChatTTSInput (observers see: text, voice_id)
    └─ Hook can: text normalization, SSML injection
         ↓
TTS
    ├─ EventChatTTSOutputStart (observers see: first_chunk)
    └─ Hook can: start timing, update UI
         ↓
    ├─ EventChatTTSOutputStop (observers see: duration, size)
    └─ Hook can: finalize metrics, log quality
         ↓
Device Audio Output
```

**Metrics Pipeline** (built-in hook):

```
12-stage metrics collector:
  1. VAD start time
  2. ASR start/end time + confidence
  3. Speaker ID start/end time
  4. LLM start/end time + tokens
  5. TTS start/end time + duration
  6. Network latency (device RTT)
  7. Total latency (device input → device output)
  + Custom metrics (business logic)

Exported via:
  - /admin/metrics (Prometheus format)
  - Dashboard (real-time charts)
  - Analytics (historical analysis)
```

---

## 10. Concurrency Model

### Per-Device

```
Device Connection (IConn)
    │
    ├─ Reader goroutine (RecvCmd, RecvAudio)
    │  └─ Buffers incoming messages/audio
    │
    └─ Writer goroutine (SendCmd, SendAudio)
       └─ Sends queued responses

ChatSession (per-device state machine)
    │
    ├─ ASR handler (receives audio chunks)
    │  └─ Calls ASRManager.Recognize()
    │
    ├─ LLM handler (receives ASR result)
    │  └─ Calls LLMManager.StreamChat()
    │
    └─ TTS handler (receives LLM response)
       └─ Calls TTSManager.Synthesize()

Synchronization:
  - ClientState uses RWMutex (state machine)
  - Message queue uses channels (async hand-off)
  - Dialogue history uses mutex (concurrent reads)
  - No deadlocks: lock hierarchy enforced
```

### Server-Level

```
Main Server (cmd/server)
    │
    ├─ WebSocket Listener (port 8989)
    │  └─ One goroutine per client connection
    │
    ├─ MQTT Broker (mochi-mqtt)
    │  └─ One goroutine per client connection
    │
    ├─ Config Reload Handler
    │  └─ Periodic (every 5m) or on-demand
    │
    ├─ Hook Worker Pool (async events)
    │  └─ N workers (configurable; default 1)
    │
    └─ Resource Cleanup
       └─ Idle session eviction (configurable timeout)

No global locks → excellent scalability
Limited only by CPU cores + network bandwidth
```


> For deployment topologies, performance characteristics, security, and error handling see [system-architecture-ops.md](system-architecture-ops.md).
