# Documentation Index

**xiaozhi-esp32-server-golang** — Complete documentation for the voice AI backend system.

---

## Quick Navigation

### For New Users

**Start here**:
1. **[Project Overview & PDR](./project-overview-pdr.md)** — What the project is, key features, requirements
2. **[Architecture Overview](./architecture-overview.md)** — High-level system design, components, data flow
3. **[Deployment Guide](./deployment-guide.md)** — How to deploy locally or in production

### For Developers

**Get started**:
1. **[Codebase Summary](./codebase-summary.md)** — Repository structure, module layout, dependencies
2. **[Code Standards](./code-standards.md)** — Go conventions, patterns, error handling (backend)
3. **[Frontend Code Standards](./frontend-code-standards.md)** — React/TypeScript conventions, component patterns
4. **[Local Dev Guide (Vietnamese)](./local-dev-guide-vi.md)** — Windows/WSL setup instructions

### For Operators & DevOps

**Production deployment**:
1. **[Deployment Guide](./deployment-guide.md)** — Docker Compose setup, configuration, health checks
2. **[Project Roadmap](./project-roadmap.md)** — Current state (v0.7.0), planned features, technical debt

### For System Architects

**Deep dives**:
1. **[System Architecture](./system-architecture.md)** — Detailed diagrams, data flows, concurrency model, security
2. **[MQTT Broker Setup](./mqtt-broker-setup-guide.md)** — MQTT configuration and integration

---

## Document Overview

| Document | Lines | Purpose |
|----------|-------|---------|
| **project-overview-pdr.md** | 374 | Project vision, functional/non-functional requirements, acceptance criteria |
| **codebase-summary.md** | 455 | Repository structure, module descriptions, dependencies, patterns |
| **architecture-overview.md** | 293 | High-level system design, components, data flow, performance |
| **system-architecture.md** | 955 | Detailed architecture, diagrams, flows, concurrency, security, scalability |
| **code-standards.md** | 614 | Go conventions, interfaces, error handling, logging, testing |
| **frontend-code-standards.md** | 482 | React/TypeScript patterns, forms, API clients, styling, i18n |
| **deployment-guide.md** | 711 | Docker setup, configuration, health checks, scaling, troubleshooting |
| **project-roadmap.md** | 504 | Version history, planned features, roadmap, technical debt |
| **local-dev-guide-vi.md** | 358 | Vietnamese local development setup guide |
| **mqtt-broker-setup-guide.md** | 200 | MQTT configuration reference |

**Total**: ~4,950 lines of documentation (concise, focused)

---

## How to Use This Documentation

### Setting Up Locally

1. Clone: `git clone https://github.com/hackers365/xiaozhi-esp32-server-golang.git`
2. Read: **[Local Dev Guide](./local-dev-guide-vi.md)** (or Docker Compose in **[Deployment Guide](./deployment-guide.md)**)
3. Configure: Update `config/config.personal.yaml` with your AI provider API keys
4. Start: `docker-compose -f docker/docker-compose.local.yml up -d`
5. Access: http://localhost:3000 (dashboard)

### Building a Feature

1. Understand: Read **[Codebase Summary](./codebase-summary.md)** for module structure
2. Design: Check **[Architecture Overview](./architecture-overview.md)** for data flow
3. Code: Follow **[Code Standards](./code-standards.md)** (Go) or **[Frontend Code Standards](./frontend-code-standards.md)** (React)
4. Test: Run tests; check against acceptance criteria in **[PDR](./project-overview-pdr.md)**

### Deploying to Production

1. Review: **[Deployment Guide](./deployment-guide.md)** Quick Start section
2. Configure: Set environment variables, secrets, database
3. Deploy: `docker-compose -f docker/docker-compose.yml up -d`
4. Monitor: Use health endpoints and dashboard
5. Scale: Refer to Scaling Considerations in **[Deployment Guide](./deployment-guide.md)**

### Understanding Architecture

1. Overview: **[Architecture Overview](./architecture-overview.md)** (5 min read)
2. Deep dive: **[System Architecture](./system-architecture.md)** (detailed diagrams, flows)
3. Specific topic: Jump to section in **[System Architecture](./system-architecture.md)**
   - Audio pipeline: Section 2 (Data Flow)
   - State machine: Section 4 (Device State Machine)
   - MCP: Section 7 (3-Tier Architecture)
   - Security: Section 14 (Security Architecture)

---

## Key Concepts Quick Reference

### Core Components

| Component | Purpose | File | Section |
|-----------|---------|------|---------|
| **IConn** | Transport abstraction (WebSocket/MQTT/UDP) | codebase-summary.md | Key Design Patterns |
| **ClientState** | Per-device state machine | codebase-summary.md | Core Modules |
| **ChatSession** | Orchestrates ASR/LLM/TTS pipeline | architecture-overview.md | Core Components |
| **Managers** | ASR, TTS, LLM, VAD, Speaker, Memory, RAG, MCP | codebase-summary.md | Core Modules (16+) |
| **Config Provider** | Hot-reload configuration | system-architecture.md | Section 5 |
| **Event Hooks** | Observable stages in AI pipeline | system-architecture.md | Section 9 |

### Key Design Patterns

| Pattern | File | Section |
|---------|------|---------|
| Provider Pattern | codebase-summary.md | Key Design Patterns |
| Transport Abstraction | codebase-summary.md | Key Design Patterns |
| State Machine | codebase-summary.md | Key Design Patterns |
| Configuration Hot-Reload | codebase-summary.md | Key Design Patterns |
| MCP 3-Tier | codebase-summary.md | Key Design Patterns |

### Provider Ecosystem

- **ASR**: FunASR, Aliyun Qwen3, Doubao, Xunfei
- **TTS**: EdgeTTS, OpenAI, Doubao, Xunfei, Qwen, Minimax, CosyVoice, Supertonic, Xiaozhi, IndexTTS
- **LLM**: OpenAI, Azure, Ollama, Coze, Dify, Doubao, DeepSeek (via Eino)
- **VAD**: Silero, Tencent TEN, WebRTC
- **Memory**: Memobase, Mem0, MemOS
- **RAG**: Dify, RAGFlow, Weknora

See **[Project Overview](./project-overview-pdr.md)** Section 2.3 for details.

---

## Frequently Asked Questions

**Q: Where do I start?**  
A: Read **[Project Overview](./project-overview-pdr.md)** for context, then **[Architecture Overview](./architecture-overview.md)** for system design.

**Q: How do I add a new provider?**  
A: See **[Codebase Summary](./codebase-summary.md)** → "Provider Pattern" section. Follow the existing pattern (implement Provider interface, register in init()).

**Q: How do I deploy to production?**  
A: Follow **[Deployment Guide](./deployment-guide.md)** → "Quick Start: Docker Compose (Production)" section.

**Q: What's the latency breakdown?**  
A: See **[Architecture Overview](./architecture-overview.md)** → "Data Flow" section (~1.7s typical).

**Q: How do I scale to multiple servers?**  
A: See **[Deployment Guide](./deployment-guide.md)** → "Scaling Considerations" or **[System Architecture](./system-architecture.md)** → Section 11.

**Q: Where's the Vietnamese dev guide?**  
A: **[Local Dev Guide (Vietnamese)](./local-dev-guide-vi.md)** covers Windows/WSL setup.

---

## Documentation Maintenance

### When to Update

- Code changes that affect public contracts (APIs, configs, message formats)
- New features or modules
- Architecture changes or refactoring
- Build/deployment process changes
- Security or performance improvements

### When NOT to Update

- Internal refactoring (no behavior change)
- Bug fixes to private functions
- Dependency updates (unless API changed)
- Internal variable renames

### How to Update

1. Read the existing doc section
2. Verify your changes against the actual codebase
3. Update the doc to match reality
4. Keep concise (trim examples, move details to code comments)
5. Update the Document Overview table above (line count)

---

## Related Resources

- **GitHub**: https://github.com/hackers365/xiaozhi-esp32-server-golang
- **License**: MIT
- **Community**: WeChat group (add hackers365 to join)
- **Contact**: hackers365 (GitHub maintainer)

---

## Document Versions

| Date | Version | Changes |
|------|---------|---------|
| 2026-06-20 | 1.0 | Initial documentation set (10 files, ~5000 lines) |

---

## Feedback & Contributions

Found an error or want to improve these docs?

1. **GitHub Issues**: Open an issue with `documentation` label
2. **Pull Requests**: Submit fixes directly (PRs welcome!)
3. **WeChat**: Discuss in community group

All contributions are appreciated!

---

**Last Updated**: 2026-06-20  
**Status**: Complete (v0.7.0)  
**LOC Target**: Keep all files < 800 lines (maintainability)
