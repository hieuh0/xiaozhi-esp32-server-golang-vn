# System Architecture — Operations & Performance

Supplement to [system-architecture.md](system-architecture.md).

---

## 1. Deployment Topology

### Single Server (Development)

```
┌──────────────────────────────────┐
│ Docker Compose                   │
├──────────────────────────────────┤
│ Service            Port           │
├──────────────────────────────────┤
│ main-server        8989 (WebSocket)
│                    2883 (MQTT)    │
│                    8990 (UDP)     │
├──────────────────────────────────┤
│ backend            8080 (HTTP REST)
│                    + WebSocket    │
├──────────────────────────────────┤
│ frontend           80 (React SPA) │
├──────────────────────────────────┤
│ mysql              3306 (internal)│
├──────────────────────────────────┤
│ redis              6379 (internal)│
├──────────────────────────────────┤
│ qdrant             6333 (vector DB)
├──────────────────────────────────┤
│ funasr-server      10095 (gRPC)  │
│ (optional)         10096 (REST)  │
├──────────────────────────────────┤
│ mcp-filesystem     3001           │
│ mcp-memory         3002           │
└──────────────────────────────────┘
```

### Multi-Server (Production)

```
┌────────────────────────────────────────┐
│ Load Balancer (HAProxy / NGINX)        │
│ ├─ 8989 (WebSocket) → main-servers    │
│ ├─ 2883 (MQTT) → mqtt-broker          │
│ └─ 8080 (API) → backend-servers       │
└────────────────────────────────────────┘
           │
     ┌─────┼─────┬─────────────┐
     │     │     │             │
     ▼     ▼     ▼             ▼
┌─────────────────┐      ┌─────────────────┐
│ main-server-1   │      │ main-server-2   │
│ (WebSocket)     │      │ (WebSocket)     │
│ (MQTT client)   │      │ (MQTT client)   │
└─────────────────┘      └─────────────────┘
     │                        │
     └────────────┬───────────┘
                  │
                  ▼
         ┌─────────────────┐
         │ Central Broker  │
         │ (MQTT/Redis)    │
         │ - Message sync  │
         │ - Config sync   │
         │ - Session cache │
         └─────────────────┘
                  │
     ┌────────────┼────────────┐
     │            │            │
     ▼            ▼            ▼
┌─────────────────────────────────────┐
│ Shared Services                     │
│ ├─ MySQL (primary + replica)       │
│ ├─ Redis (cluster)                  │
│ ├─ Qdrant (vector DB)              │
│ ├─ MCP Global Servers              │
│ └─ AI Provider APIs (external)     │
└─────────────────────────────────────┘
```

**Key Points**:
- Main servers are stateless (no session affinity needed)
- All config + history centralized (MySQL, Redis)
- MQTT broker routes messages between servers
- Each main server independently connects to AI providers
- Horizontal scaling: add more main-servers behind load balancer

---

## 2. Error Handling & Recovery

**Connection failures**: `IConn.OnClose()` triggers ChatSession cleanup (cancel ASR/LLM/TTS, close connections). Device can reconnect and resume from chat history (Redis/DB).

**Provider failures**: Caught by manager (ASRManager, LLMManager). Error logged, fallback to secondary provider if configured, otherwise send error to device.

**Resource exhaustion**: Message queue overflow (configurable drop/block). Idle sessions auto-close after timeout (`chat.max_idle_duration`, default 30s).

---

## 3. Performance Characteristics

### Memory

```
Per-device (idle):
  - ClientState struct: ~1 KB
  - Dialogue (100 messages): ~50 KB
  - WebSocket connection buffer: ~10 KB
  - Total: ~100 KB

Per-device (active ASR):
  - Audio buffer (5s at 16kHz): ~400 KB
  - Total: ~500 KB

Per-device (active LLM):
  - Token buffer: ~10 KB
  - Total: ~100 KB

Total for 1000 idle devices: ~100 MB
Total for 100 active devices: ~50 MB
```

### CPU

```
Per-device (idle):
  - Goroutines: 3 (reader, writer, state machine)
  - CPU: < 0.01%

Per-device (active):
  - ASR: varies (depends on provider; mostly I/O wait)
  - LLM: varies (mostly I/O wait for streaming)
  - TTS: varies (mostly I/O wait for streaming)
  - Total: ~0.5-1% per device with streaming

1000 idle devices on 8-core machine: ~5% CPU
100 active devices on 8-core machine: ~50% CPU
```

### Network

```
Audio stream (Opus):
  - Bitrate: 8-16 Kbps (configurable)
  - 5-second utterance: ~5-10 KB

ASR result: ~100-500 bytes (JSON text)
LLM response (streaming): ~1-10 KB (JSON chunks)
TTS audio (Opus): ~10-50 KB per response

Per-device bandwidth:
  - ~30 KB per utterance (up + down)
  - Rate: 1 utterance per 10-30 seconds
  - Average: ~1-3 Kbps per device
```

### End-to-End Latency

| Stage | Typical |
|-------|---------|
| VAD silence detection | ~400 ms |
| ASR transcription | ~500 ms |
| LLM first token | ~800 ms |
| TTS first chunk | ~400 ms |
| **Total (first audio back)** | **~1.7 s** |

---

## 4. Security Architecture

**Data Protection**:
- TLS optional on MQTT (port 8883) and via reverse proxy HTTPS
- Passwords bcrypt-hashed in DB
- API keys encrypted at rest in `configs.data`
- JWT HS256-signed (24h expiry)

**Access Control**:
- Device must be activated before connecting
- Per-device rate limits + quotas (configurable)
- JWT for user/admin APIs; bearer token for internal service calls
- CORS restricted to configured origins

**Audit**:
- Logged: device connect/disconnect, config changes, auth failures
- Not logged: chat content (privacy), raw audio, API keys

---

## 5. Future Architecture Improvements

- **Distributed session state**: Move from per-server to Redis for true multi-server failover
- **Event sourcing**: Store all config changes + chat events for audit + replay
- **Streaming LLM inference**: Embed smaller LLMs on server instead of external APIs
- **Advanced RAG**: Vector store integration for knowledge base semantic search
- **A/B testing**: Support multiple provider configs per device
- **Usage analytics**: Detailed metrics per device, user, provider for billing/optimization
