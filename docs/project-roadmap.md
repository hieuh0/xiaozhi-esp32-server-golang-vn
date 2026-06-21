# Project Roadmap

**xiaozhi-esp32-server-golang**  
Development status, planned features, and technical debt management.

---

## Current State: v0.7.0

**Release Date**: June 2026  
**Commit**: Latest from `develop` branch

### Stable Features

✅ **Core Voice Pipeline**
- End-to-end ASR → LLM → TTS streaming
- Multi-protocol transport (WebSocket, MQTT, UDP)
- Per-device state machine and session management
- Configuration hot-reload (MQTT, UDP, MCP settings)

✅ **Multi-Provider Support**
- ASR: FunASR, Aliyun Qwen3, Doubao, Xunfei
- TTS: EdgeTTS, OpenAI, Doubao, Xunfei, Qwen, Minimax, CosyVoice, Supertonic, Xiaozhi, IndexTTS
- LLM: OpenAI, Azure, Ollama, Coze, Dify, Doubao, DeepSeek (via Eino)
- VAD: Silero, Tencent TEN, WebRTC
- Memory: Memobase, Mem0, MemOS

✅ **Advanced Capabilities**
- Speaker identification and per-speaker voice selection
- MCP 3-tier architecture (local, global, device)
- Knowledge base integration (Dify, RAGFlow, Weknora)
- Voice cloning support (Minimax, CosyVoice)
- OpenClaw agent integration with keyword routing
- Event hooks (7 stages, 12 metrics pipeline)
- OTA update support via MQTT

✅ **Management Dashboard**
- React SPA with TanStack Router
- Device activation and management
- Provider configuration UI (ASR, TTS, LLM, VAD, Memory, etc.)
- Real-time pool statistics and latency monitoring
- Chat history viewer
- User authentication (JWT)
- i18n support (EN, VI, ZH)

✅ **Infrastructure**
- Docker Compose (production + local dev)
- SQLite/MySQL database support
- Redis caching and session management
- Embedded MQTT broker (mochi-mqtt)
- Structured logging with rotation
- pprof profiling support

✅ **Documentation**
- Architecture documentation
- Code standards and conventions
- Provider integration guides
- Local development setup (Windows, Linux)
- Deployment guides

---

## Near-Term Roadmap (v0.8.0 - Q3 2026)

### 1. Enhanced Observability

**Goal**: Better monitoring and debugging for production deployments.

**Features**:
- [ ] Prometheus metrics export (/metrics endpoint)
- [ ] Distributed tracing (OpenTelemetry integration)
- [ ] Health check dashboard improvements
- [ ] Provider latency breakdown per request
- [ ] Database query performance monitoring
- [ ] WebSocket/MQTT connection analytics

**Effort**: Medium (2-3 weeks)  
**Owner**: TBD

---

### 2. Advanced RBAC & Multi-Tenancy

**Goal**: Enterprise-ready permission and isolation system.

**Features**:
- [ ] Fine-grained role-based access control (RBAC)
  - Admin, operator, agent_owner, user roles
  - Granular permissions (create_device, edit_config, view_analytics, etc.)
- [ ] Organization/team support
  - Devices and agents scoped to organization
  - Multi-user collaboration
- [ ] Audit logging
  - All config changes logged with user/timestamp
  - API access logs (who accessed what, when)
  - Chat message retention policies
- [ ] Usage quotas and rate limiting per device/user
  - API rate limits (requests/sec)
  - Token usage limits (for LLM)
  - Storage quotas (chat history, vector embeddings)

**Effort**: Large (4-6 weeks)  
**Owner**: TBD  
**Dependencies**: No blockers; can be independent feature

---

### 3. Advanced Caching & Performance

**Goal**: Sub-1-second latency for common operations.

**Features**:
- [ ] Response caching layer
  - Cache ASR results for duplicate audio
  - Cache LLM responses for common prompts
  - TTL-based expiration
- [ ] Vector embedding cache (speaker ID)
  - Pre-compute embeddings at device activation
  - Incremental updates
- [ ] Provider response pooling
  - Reuse streaming connections
  - Connection keep-alive tuning

**Effort**: Medium (2-3 weeks)  
**Owner**: TBD  
**Impact**: ~30% latency reduction

---

### 4. Streaming LLM Inference

**Goal**: Reduce external API dependency; support on-device inference.

**Features**:
- [ ] Support for open-source LLMs via Ollama
  - Pre-packaged models (Llama 2, Mistral)
  - Easy Docker setup
- [ ] Quantized models for lower latency
  - GGUF format support
  - Dynamic model loading based on device config
- [ ] Fallback chain
  - If local model fails/slow, fallback to cloud API

**Effort**: Large (3-4 weeks)  
**Owner**: TBD  
**Risk**: Requires testing on target hardware; latency trade-offs

---

### 5. Knowledge Base Enhancements

**Goal**: Smarter RAG with semantic search.

**Features**:
- [ ] Chunking strategy improvements
  - Hierarchical chunking (section → paragraph → sentence)
  - Overlap handling
- [ ] Embedding cache & incremental updates
  - Only re-embed changed documents
  - Vector DB optimization
- [ ] Multi-knowledge-base support per device
  - Priority-based search (public KB, personal KB, org KB)
  - Conflict resolution
- [ ] Retrieval evaluation
  - Track which KB articles are used
  - Feedback loop for relevance

**Effort**: Medium (2-3 weeks)  
**Owner**: TBD  
**Depends on**: Vector DB setup (Qdrant, Milvus)

---

### 6. Webhook & Event Stream

**Goal**: Enable external integrations.

**Features**:
- [ ] Webhook notifications
  - Device connect/disconnect
  - Config change events
  - Error events
  - Configurable delivery (HTTP POST, MQTT, etc.)
- [ ] Server-sent events (SSE) stream
  - Real-time event feed for admin dashboard
  - Reduced polling overhead
- [ ] Event replay
  - Recover events from message queue if client temporarily offline

**Effort**: Small (1-2 weeks)  
**Owner**: TBD

---

## Medium-Term Roadmap (v0.9.0 - Q4 2026)

### 1. Proactive AI (Mentioned in README)

**Goal**: Enable AI to initiate conversation without user input.

**Features**:
- [ ] Time-based triggers
  - "At 9 AM, remind user to check calendar"
  - "Every 30 minutes, ask if help is needed"
- [ ] Context-based triggers
  - "When outdoor temperature drops below 5°C, suggest heating"
  - "When battery level < 10%, warn about low power"
- [ ] Interrupt management
  - Don't interrupt during important user conversations
  - Respect user availability (silent hours, busy mode)
- [ ] Conversation initiation prompts
  - Dynamic system prompts based on context
  - "Warmth" tuning (friendly vs. professional)

**Effort**: Large (4-5 weeks)  
**Owner**: TBD  
**Risk**: UX challenge (avoid annoying users)

---

### 2. Agent Swarm & Multi-Agent Orchestration

**Goal**: Support multiple AI agents in conversation trees.

**Features**:
- [ ] Agent composition
  - Route to different agents based on intent
  - "Transfer to support agent", "consult finance bot"
- [ ] Multi-agent conversation
  - Agents can call other agents as tools
  - Hierarchical reasoning
- [ ] Agent specialization
  - Per-agent knowledge base
  - Per-agent system prompt
  - Per-agent provider configs (e.g., specialized LLM for math)

**Effort**: Large (3-4 weeks)  
**Owner**: TBD  
**Depends on**: MCP improvements

---

### 3. Vision Capabilities Expansion

**Goal**: Improve image/video understanding.

**Features**:
- [ ] Multi-modal LLM support
  - GPT-4 Vision, Claude Vision, Doubao Vision
  - Image description, scene understanding, OCR
- [ ] Video processing
  - Frame extraction at intervals
  - Change detection (summarize changes between frames)
- [ ] On-device vision models
  - Real-time object detection (YOLO, etc.)
  - Edge processing before sending to cloud

**Effort**: Medium (2-3 weeks)  
**Owner**: TBD

---

### 4. Voice Clone Improvements

**Goal**: Higher-quality voice synthesis from user samples.

**Features**:
- [ ] Custom voice training
  - User uploads voice samples
  - Fine-tune TTS model (if provider supports)
- [ ] Voice adaptation
  - Adjust voice characteristics (age, accent, emotion)
  - Style transfer (formal vs. casual)
- [ ] Multi-voice support per user
  - Different voices for different personas/agents
  - Voice mixing/blending

**Effort**: Large (3-4 weeks)  
**Owner**: TBD  
**Risk**: Provider API limitations

---

### 5. Distributed Deployment & Failover

**Goal**: True high-availability multi-region setup.

**Features**:
- [ ] Multi-region main servers
  - Session affinity / sticky sessions
  - Device reconnection to nearest region
- [ ] Database replication
  - Primary-replica MySQL setup
  - Read-replica for analytics
- [ ] Failover automation
  - Automatic detection of server failure
  - Transparent reconnection for devices
- [ ] Metrics aggregation across regions
  - Central analytics dashboard

**Effort**: Large (4-5 weeks)  
**Owner**: TBD  
**Risk**: Complexity; requires careful testing

---

## Long-Term Vision (v1.0+ - 2026-2027)

### 1. Embedded ML Runtime

Support for running neural networks directly on the server:
- TensorFlow Lite, ONNX Runtime
- Custom voice models (ASR, TTS fine-tuning)
- Fast inference without cloud API calls
- Cost savings for high-volume deployments

---

### 2. Blockchain Integration (Optional)

For decentralized deployments:
- Device identity verification via blockchain
- Tamper-proof audit logs
- Peer-to-peer device discovery

---

### 3. Edge Device Support

Expand beyond ESP32:
- iOS/Android native clients
- Web browser clients (browser speech API)
- Smart speaker integration (Alexa, Google Assistant)
- Car infotainment systems

---

### 4. Advanced Personalization

Machine learning-based user modeling:
- Conversation style adaptation
- Preference learning (voice, response length, topics)
- Proactive recommendations

---

## Technical Debt & Improvements

### High Priority

- [ ] **Refactor config hot-reload**
  - Current semantic diffing is brittle
  - Use JSON schema validation + change detection library
  - Estimated effort: 1 week

- [ ] **Improve error messages**
  - Current error messages lack context
  - Add user-friendly error codes + troubleshooting links
  - Estimated effort: 2-3 days

- [ ] **Simplify provider registration**
  - Too much boilerplate per provider
  - Auto-register via reflection or plugin system
  - Estimated effort: 1 week

### Medium Priority

- [ ] **Database migration tooling**
  - Currently manual; prone to errors
  - Use GORM auto-migrations with version tracking
  - Estimated effort: 3-4 days

- [ ] **Frontend code splitting**
  - Currently single bundle; slow on slow networks
  - Lazy-load routes, components
  - Estimated effort: 2-3 days

- [ ] **WebSocket reconnection logic**
  - Currently basic; no exponential backoff
  - Add smart reconnection with jitter
  - Estimated effort: 2-3 days

### Low Priority (Nice to Have)

- [ ] Code generation for provider templates
- [ ] GraphQL API (alongside REST)
- [ ] Admin CLI tool (instead of dashboard)
- [ ] SDK for custom device clients

---

## Known Limitations & Constraints

### Architectural

- **Single-server sessions**: Device affinity required for state; multi-server requires Redis
- **Synchronous pipeline stages**: ASR must complete before LLM; no parallel processing
- **Provider-specific**: Each provider has different latency/quality trade-offs; no universal optimization

### Provider-Related

- **API rate limits**: External AI providers have rate limits; may throttle on high traffic
- **Model availability**: Not all providers available in all regions
- **Cost**: Heavier usage leads to higher API bills

### Platform-Related

- **Audio codecs**: Only Opus supported; MP3, WAV require conversion
- **Device compatibility**: Primary focus on ESP32; other platforms supported but less tested
- **Bandwidth**: Streaming requires stable network; poor connectivity degrades experience

---

## Success Metrics

### Adoption

- [ ] 500+ active devices in production (by v0.8.0)
- [ ] 10+ community-contributed provider integrations
- [ ] 5+ organizations using multi-tenant setup (by v0.9.0)

### Performance

- [ ] P95 latency < 2s for full pipeline
- [ ] Support 5000+ concurrent devices on single server (optimized)
- [ ] < 100ms per-request overhead (excluding provider latency)

### Quality

- [ ] 95%+ message delivery success rate
- [ ] 99.5% availability (excluding provider outages)
- [ ] > 4.0/5 stars on user satisfaction surveys

### Community

- [ ] 1000+ GitHub stars
- [ ] 50+ code contributions from community
- [ ] 10+ blog posts / case studies

---

## Contributing & Roadmap Feedback

This roadmap is community-driven. To suggest features or report priorities:

1. **Open GitHub Issue** with feature request (use `enhancement` label)
2. **Discuss on WeChat**: hackers365 (community group)
3. **Vote on planned features**: React with 👍 on GitHub issues to show interest

Prioritization factors:
- User demand (votes + comments)
- Implementation complexity
- Strategic importance (roadmap alignment)
- Maintenance burden

---

## Appendix: Feature Request Template

```markdown
## Feature: [Clear, concise title]

### Problem
What problem does this solve? Who needs this?

### Proposed Solution
How would this feature work?

### Alternatives Considered
What other approaches were considered?

### Additional Context
Links, examples, related features.

### Effort Estimate
Small (< 1 week), Medium (1-3 weeks), Large (3+ weeks)?

### Priority
Critical, High, Medium, Low?
```

---

## Release Schedule

| Version | Target | Key Features | Status |
|---------|--------|--------------|--------|
| v0.7.0 | June 2026 | Core voice pipeline, multi-provider, dashboard | ✅ Released |
| v0.8.0 | Sep 2026 | Observability, RBAC, advanced caching | 📋 Planning |
| v0.9.0 | Dec 2026 | Proactive AI, agent swarms, vision expansion | 📋 Planning |
| v1.0.0 | Q2 2027 | Stable API, embedded ML, production hardening | 🔮 Vision |

---

## Contact & Governance

**Project Lead**: hackers365 (GitHub)  
**Community**: WeChat group (add to join)  
**License**: MIT (open source)

For questions, reach out via:
- GitHub Issues (bugs, feature requests)
- GitHub Discussions (questions, ideas)
- WeChat (real-time chat, community support)
