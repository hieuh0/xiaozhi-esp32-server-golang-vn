# Project Overview & Product Development Requirements (PDR)

**Project**: xiaozhi-esp32-server-golang-vn  
**Version**: 0.7.0  
**Last Updated**: 2026-06-20  
**License**: MIT

---

## 1. Executive Summary

**xiaozhi-esp32-server-golang** is a high-performance, fully-streaming Go backend for ESP32 IoT voice AI devices. The project provides a complete AI voice pipeline (ASR → LLM → TTS) with multi-protocol support (WebSocket, MQTT, UDP) and a management dashboard for configuration, device activation, knowledge management, and system monitoring.

**Core Purpose**: Enable ESP32 and other edge devices to perform real-time voice AI interactions through a unified backend with pluggable AI provider support.

**Target Users**:
- IoT device manufacturers and integrators
- Voice assistant application developers
- Researchers in edge AI and conversational systems
- Self-hosted AI infrastructure operators

---

## 2. Key Features & Capabilities

### Voice Pipeline
- **End-to-end streaming**: Audio from device → ASR (speech-to-text) → LLM (language model) → TTS (text-to-speech) → audio response back to device
- **Low latency**: Designed for real-time interaction; metrics tracked per-stage
- **Configurable interrupt modes**: VAD-based, ASR-based, voice identity, or streaming ASR interruption

### Provider Flexibility
- **Multiple AI providers per domain**: Swap providers via configuration without code changes
- **Supported domains**:
  - ASR: FunASR, Aliyun Qwen3, Doubao, Xunfei
  - TTS: EdgeTTS, OpenAI, Doubao, Xunfei, Qwen, CosyVoice, Supertonic, Minimax, Xiaozhi
  - LLM: OpenAI, Azure, Ollama, Coze, Dify, Doubao, DeepSeek (via Cloudwego Eino)
  - VAD: Silero VAD, Tencent TEN VAD, WebRTC VAD
  - Memory: Memobase, Mem0, MemOS

### Distributed Architecture
- **Multi-protocol**: WebSocket (TCP), MQTT (with internal server), UDP for low-latency scenarios
- **Concurrent connections**: Resource pools with configurable limits and idle timeouts
- **Transport abstraction**: Single `IConn` interface for uniform message/audio handling
- **Embedded manager backend**: Optional REST API + JWT auth for device management and configuration

### Advanced Capabilities
- **Speaker identification**: Automatic voice recognition via sherpa-onnx + vector DB; enables per-speaker TTS voice selection
- **MCP integration**: 3-tier architecture (local process, global server, per-device) for tool/plugin extensions
- **Knowledge base**: Integration with Dify, RAGFlow, Weknora for RAG (retrieval-augmented generation)
- **Voice cloning**: Support for Minimax, CosyVoice voice synthesis
- **OpenClaw agent integration**: Keyword-based mode switching (e.g., "activate agent mode", "exit agent mode")
- **Configuration hot-reload**: Update MQTT, UDP, MCP settings at runtime via HTTP without restart
- **OTA updates**: Over-the-air firmware update support via MQTT
- **i18n support**: Dashboard in English, Vietnamese, Chinese

### Management Dashboard
- **Web-based admin UI**: React SPA for device activation, configuration, monitoring
- **Health checks**: Real-time latency and availability checks for VAD/ASR/LLM/TTS per config
- **Device pool monitoring**: Connection count, idle sessions, resource utilization
- **Chat history**: View device-agent interactions
- **User & permission management**: Admin, user roles with multi-tenancy support (planned)

---

## 3. System Requirements

### Runtime Environment
- **OS**: Linux (preferred), macOS, Windows (dev/test)
- **Go**: 1.24.2+ (builds with 1.24.11)
- **Memory**: Minimum 1 GB (typical 2-4 GB with AI models loaded)
- **CPU**: 2+ cores (multicore for concurrent connections)
- **Network**: Stable TCP/UDP connectivity for external AI providers

### Build Dependencies
- **Opus codec**: libopus (for audio compression)
- **ONNX Runtime**: 1.21.0+ (for VAD models)
- **C++ stdlib**: libc++1, libc++abi1 (for TEN VAD)

### External Services (optional but typical)
- **MySQL/SQLite**: Configuration and device metadata storage
- **Redis**: Session caching, rate limiting, chat history
- **External AI APIs**: OpenAI, Doubao, Xunfei, Aliyun, etc.
- **Qdrant/Milvus**: Vector database for speaker identification and RAG

---

## 4. Functional Requirements

### 4.1 Device Connection & Transport

**Requirement**: Support multiple transport protocols for device communication.

**Acceptance Criteria**:
- Devices can connect via WebSocket on port 8989
- Devices can connect via MQTT on port 2883 (with optional TLS 8883)
- UDP mode (port 8990) supports low-latency audio streaming
- Message protocol includes device identification, audio format negotiation, and state machine
- Graceful disconnect handling with cleanup of per-device state

**Key Components**:
- `IConn` interface: Abstracts transport; implementations in WebSocket and MQTT+UDP adapters
- `ClientState`: Per-device session state machine (init → listening → llmStart → ttsStart → idle)
- Message types: `hello`, `listen`, `abort`, `stt`, `tts`, `llm`, `goodbye` + audio chunks

### 4.2 Audio Processing Pipeline

**Requirement**: Process audio from device through ASR, LLM, and TTS in streaming fashion.

**Acceptance Criteria**:
- Audio arrives in Opus format (configurable)
- VAD detects speech activity; silence triggers end-of-utterance
- ASR transcribes audio to text; supports streaming and offline modes
- LLM generates response based on transcription and conversation history
- TTS converts response to audio; supports streaming
- Total pipeline latency < 3s for typical utterances (ASR 500ms, LLM 800ms, TTS 400ms)
- Audio output is Opus; client handles playback

**Key Components**:
- `ChatSession`: Orchestrates ASR, LLM, TTS, and memory managers
- `ASRManager`: Handles speech-to-text with provider abstraction
- `LLMManager`: Handles text-to-response with provider abstraction
- `TTSManager`: Handles response-to-speech with provider abstraction
- `VADManager`: Detects speech activity; configurable models

### 4.3 Provider Configuration

**Requirement**: Allow runtime configuration of AI providers without code recompilation.

**Acceptance Criteria**:
- Dashboard UI exposes forms for each provider domain (ASR, TTS, LLM, VAD, Memory, etc.)
- Configuration stored in backend database and hot-reloaded to main server via HTTP
- Each provider type has documented required parameters (API keys, model names, endpoints)
- Configuration validation detects invalid settings before deploy
- Fallback support: if primary provider fails, optionally use secondary

**Key Components**:
- `config_provider`: Interface for fetching configs (manager, redis, local, http)
- Domain-specific config structs (e.g., `ASRConfig`, `TTSConfig`)
- Manager backend `/api/admin/asr-config`, `/api/admin/tts-config` endpoints

### 4.4 Speaker Identification & Per-Speaker TTS

**Requirement**: Identify speaker and optionally use custom TTS voice for each speaker.

**Acceptance Criteria**:
- Speaker embedding model (sherpa-onnx) processes audio to generate speaker vectors
- Vectors stored in vector DB (Qdrant) with speaker name association
- If speaker identified, lookup custom TTS voice and apply to response
- Admin can link speakers to voice groups
- Support up to 100+ distinct speakers in production

**Key Components**:
- `SpeakerManager`: Orchestrates embedding, matching, voice selection
- `speaker_identification` module with sherpa-onnx integration
- Qdrant client for vector search

### 4.5 Message Queue & Async Processing

**Requirement**: Decouple LLM and TTS from client connection to improve responsiveness.

**Acceptance Criteria**:
- ASR result queued internally; LLM processing starts async
- LLM output queued for TTS; response streams back to client as ready
- Queue overflow behavior: configurable (drop oldest, drop newest, or block)
- Timeout handling: if LLM takes > configured limit, send error response

**Key Components**:
- `chat_hooks`: Event-driven hooks at ASR/LLM/TTS stages
- Priority queue implementation for message ordering
- Timeout monitors and error recovery

### 4.6 Device Activation & Authentication

**Requirement**: Manage device lifecycle and enforce access control.

**Acceptance Criteria**:
- Devices register with backend (one-time activation via manager UI or API)
- Device gets unique ID and authentication token
- Device includes token in WebSocket/MQTT hello message
- Manager validates token and associates device with user/agent
- Token refresh / rotation support (optional)
- Per-device rate limiting and quota enforcement

**Key Components**:
- Manager backend `Device` model and activation controller
- JWT token validation middleware
- Internal auth token for server-to-manager communication

### 4.7 MCP (Model Context Protocol) Support

**Requirement**: Enable tools and plugins via MCP servers at three tiers.

**Acceptance Criteria**:
- Local MCP: Process-hosted tools (filesystem, memory, arithmetic, etc.)
- Global MCP: Server-level tool registry shared across all devices
- Device MCP: Per-device custom tools or endpoints
- Tools discoverable at runtime; LLM can invoke tools as part of response generation
- Tool output integrated into LLM context without separate turn
- Remote call debugging: trace MCP invocation per device/agent

**Key Components**:
- `mcp` module with 3-tier manager
- SSE/WebSocket protocol handlers for MCP communication
- Tool registry and schema validation

### 4.8 Knowledge Base & RAG

**Requirement**: Enable retrieval-augmented generation (RAG) for grounded responses.

**Acceptance Criteria**:
- Integration with Dify, RAGFlow, or Weknora for knowledge search
- Admin can upload documents and sync embeddings
- LLM receives relevant context (from knowledge base) before generating response
- Retrieval metrics tracked: # of chunks, relevance score
- Support for multiple knowledge bases per device/agent

**Key Components**:
- `rag` module with Dify, RAGFlow, Weknora searchers
- Knowledge sync API to external RAG systems
- LLM context augmentation in `LLMManager`

### 4.9 Configuration Hot-Reload

**Requirement**: Update MQTT, UDP, and MCP settings without restarting server.

**Acceptance Criteria**:
- Manager backend writes config changes to database
- Main server polls for updates (default 5m, configurable)
- Changes applied only if semantically different (e.g., different port or broker)
- MQTT/UDP connections gracefully closed and re-established
- MCP managers reinitialized with new settings
- No message loss during reload (queued until reconnect)

**Key Components**:
- Viper config hot-reload mechanism
- Semantic diff function to avoid unnecessary restarts
- Periodic config sync from manager via HTTP

### 4.10 Monitoring & Observability

**Requirement**: Track performance metrics and health status.

**Acceptance Criteria**:
- Per-request latency: ASR, LLM, TTS stages recorded
- Server metrics: active connections, message throughput, queue depth
- Health endpoints: `/metrics`, provider health checks
- pprof profiling available on configurable port (default disabled)
- Dashboard shows real-time pool stats, active devices, latencies
- Log rotation and retention configured (default 3 days, 10-hour rotation)

**Key Components**:
- `chat_hooks` metrics pipeline
- Prometheus-compatible `/metrics` endpoint (optional)
- Health check routes in manager backend

---

## 5. Non-Functional Requirements

### 5.1 Performance

- **Device latency**: < 100ms for message round-trip (excluding AI provider latency)
- **Concurrent connections**: Support 1000+ devices on single server (depends on hardware)
- **Memory per connection**: < 1 MB per idle device
- **Message throughput**: 10,000+ events/sec at server level
- **CPU efficiency**: < 50% CPU for 100 concurrent devices with 2 cores

### 5.2 Reliability

- **Availability**: 99.5% uptime in production (excluding AI provider outages)
- **Graceful degradation**: If one provider fails, fallback to secondary or error response
- **Connection recovery**: Device auto-reconnect with exponential backoff
- **Data integrity**: No message loss during normal operation; queue persistence (optional Redis)

### 5.3 Scalability

- **Horizontal scaling**: Multiple server instances behind load balancer (via MQTT broker)
- **Vertical scaling**: Support for larger machines (16+ cores, 32+ GB RAM)
- **Database**: SQLite for dev, MySQL for production multi-instance
- **Session state**: Redis for distributed session caching

### 5.4 Security

- **Authentication**: JWT for API access (24h default expiry)
- **Internal auth**: HTTP bearer token for manager-to-server calls
- **MQTT TLS**: Optional MQTT over TLS (8883)
- **Input validation**: All user inputs validated and sanitized
- **Rate limiting**: Per-device message rate limit (configurable)
- **Secret management**: API keys stored encrypted in database; not in logs

### 5.5 Maintainability

- **Code modularization**: 16+ independent domain modules with clear interfaces
- **Provider pattern**: All AI providers follow same registration pattern
- **Configuration-driven**: Behavior controlled via YAML config, not code
- **Logging**: Structured logs with levels (debug, info, warn, error); rotation enabled
- **Documentation**: Code comments, architecture diagrams, provider setup guides

### 5.6 Compatibility

- **Devices**: ESP32 (primary), any device with WebSocket or MQTT support
- **Frameworks**: Eino-based LLMs (OpenAI API-compatible, Ollama, etc.)
- **Operating systems**: Linux, macOS, Windows (for development)

---

## 6. Out of Scope

- **Proactive AI**: AI initiates conversation without user input (mentioned as future roadmap)
- **Advanced security & permission system**: Comprehensive RBAC, multi-tenant isolation (placeholder in code)
- **Audio processing**: Noise cancellation, enhancement (assumed handled by client or external service)
- **Distributed training**: AI model fine-tuning (uses pre-trained models only)
- **Mobile-first UI**: Dashboard optimized for desktop; mobile support secondary
- **Real-time collaboration**: No live co-editing or shared sessions

---

## 7. Success Metrics

- **Adoption**: 100+ active device connections in pilot phase
- **Performance**: P95 latency < 2s for full pipeline on typical hardware
- **Reliability**: 99%+ message delivery success rate
- **User satisfaction**: > 4/5 stars on configuration experience (dashboard)
- **Community**: 10+ provider integrations from community contributions

---

## 8. Future Roadmap

- **Proactive AI**: Server initiates conversation based on context or time
- **Advanced RBAC**: Fine-grained permission system for multi-tenant deployments
- **Voice cloning improvements**: Custom voice training from user samples
- **Vision capabilities**: Expand vision module to support more providers
- **Embedded ML models**: Run small LLMs on server without external APIs
- **AI agent swarms**: Coordinate multiple agents in conversation tree
- **Enterprise features**: Single sign-on, audit logging, compliance reporting

---

## 9. Acceptance Criteria Summary

The project is considered complete when:

1. ✅ Core voice pipeline (ASR → LLM → TTS) works end-to-end with latency < 3s
2. ✅ Dashboard allows configuration of all major providers without code changes
3. ✅ Support for 1000+ concurrent device connections
4. ✅ Device activation and JWT authentication working
5. ✅ MCP 3-tier architecture implemented and tools callable from LLM
6. ✅ Knowledge base / RAG integration functional
7. ✅ Configuration hot-reload working for MQTT, UDP, MCP settings
8. ✅ Monitoring dashboard showing real-time metrics and health
9. ✅ Docker Compose deployment single-command start
10. ✅ Documentation covers setup, configuration, integration, troubleshooting

---

## 10. Constraints & Risks

### Constraints
- AI provider availability: System depends on external provider uptime
- Audio codec support: Limited to Opus for compression; PCM fallback for legacy devices
- Model availability: Some providers (e.g., Doubao) region-locked or API-limited

### Risks
- **Provider API changes**: Provider SDKs may break with updates; requires maintenance
- **Latency variance**: Network jitter affects perceived responsiveness
- **Model cost**: Heavy usage can lead to unexpected bills from AI providers
- **Scaling bottlenecks**: Vector DB or database may become bottleneck at high concurrency

### Mitigation
- Periodic provider integration tests in CI/CD
- Caching strategy for provider responses
- Cost monitoring and alert system
- Load testing and capacity planning before production deployment
