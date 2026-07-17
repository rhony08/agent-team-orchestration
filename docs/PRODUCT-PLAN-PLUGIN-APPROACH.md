# Product Plan: Agent Team Orchestration — Go Wrapper + OpenCode Plugin

**Document Version:** 2.0
**Last Updated:** 2026-07-17
**Status:** Draft
**Owner:** Product Engineering Team
**Approach:** Go CLI Wrapper + OpenCode Plugin (Hybrid)
**Supersedes:** v1.0 (pure plugin approach)

---

## 1. Executive Summary

### Overview
The **Agent Team Orchestration** system is a hybrid architecture combining a **Go CLI wrapper** for process management and state ownership with an **OpenCode TypeScript plugin** for orchestration logic. The Go wrapper spawns and monitors OpenCode instances across multiple repositories, while the TS plugin hooks into OpenCode's event system to coordinate agent behavior.

### Why Hybrid

| Concern | Go Wrapper | TS Plugin |
|---------|------------|-----------|
| Process lifecycle | ✅ Spawn/kill OpenCode processes | ❌ |
| State ownership | ✅ Canonical state (single writer) | ❌ Read via HTTP |
| CLI distribution | ✅ Single binary | ❌ Requires Node/Bun |
| Memory efficiency | ✅ ~10-20MB | Inside OpenCode runtime |
| Agent hooks | ❌ | ✅ Event hooks, custom tools |
| Agent configs | ✅ Generate | ✅ Use |

### Key Advantages Over Pure Plugin

1. **Single writer for state** — No concurrent write corruption
2. **Process management** — Go monitors and restarts crashed OpenCode instances
3. **Single binary distribution** — No Node/Bun dependency for CLI
4. **Memory efficient** — Go wrapper is lightweight
5. **Clear ownership boundary** — Go owns state, TS owns hooks

### Key Advantages Over Standalone

1. **3-5x faster to MVP** — Leverage OpenCode's AI infrastructure
2. **~1500 lines Go + ~1000 lines TS** vs ~5000+ lines Go
3. **Zero learning curve** — Developers already know OpenCode
4. **Automatic improvements** — OpenCode updates benefit the system

---

## 2. Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      GO WRAPPER (CLI)                            │
│  crush-orchestrator init / start / stop / status                 │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Process      │  │ State        │  │ HTTP API     │          │
│  │ Manager      │  │ Manager      │  │ (localhost)  │          │
│  │ (spawn/kill) │  │ (canonical)  │  │ Port 9800    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│  ┌──────────────┐  ┌──────────────┐                             │
│  │ Checkpoint   │  │ Health       │                             │
│  │ Handler      │  │ Monitor      │                             │
│  └──────────────┘  └──────────────┘                             │
└─────────────────────────┬───────────────────────────────────────┘
                          │ HTTP + Bearer Token
                          │
┌─────────────────────────┴───────────────────────────────────────┐
│                    OPENCODE INSTANCES                            │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           ORCHESTRATION PLUGIN (TS)                       │   │
│  │  - Hooks into OpenCode events                             │   │
│  │  - Defines custom tools (team_message, task_create, etc)  │   │
│  │  - Calls Go HTTP API for state mutations                  │   │
│  │  - Injects orchestration context on compaction            │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           AGENT CONFIGS (Markdown)                        │   │
│  │  tech-lead.md  │  backend-dev.md  │  frontend-dev.md      │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  Repo A (port 9801)  Repo B (port 9802)  Repo C (port 9803)    │
└─────────────────────────────────────────────────────────────────┘
```

### Responsibility Split

| Component | Responsibility |
|-----------|----------------|
| **Go CLI** | Process lifecycle, port allocation, state management, checkpoint UI, health monitoring |
| **Go HTTP API** | State mutations, agent registration, message routing, checkpoint resolution |
| **TS Plugin** | Event hooks, custom tools, context injection, audit logging |
| **Agent Configs** | Role definitions, prompts, permissions |
| **Shared Workspace** | Per-entity files in `.orchestrator/` directory |

### IPC Protocol

**Go → TS:** Not needed (Go owns state, TS reads via HTTP)
**TS → Go:** HTTP POST to `http://localhost:9800/api/v1/`

**Endpoints:**
```
POST /api/v1/tasks              # Create task
GET  /api/v1/tasks              # List tasks
POST /api/v1/messages           # Send message
GET  /api/v1/messages/{agent}   # Get agent inbox
POST /api/v1/checkpoints        # Create checkpoint
POST /api/v1/checkpoints/{id}   # Resolve checkpoint
GET  /api/v1/status             # Get orchestration status
POST /api/v1/agents             # Register agent
GET  /api/v1/health             # Health check
```

**Authentication:** Bearer token (shared secret from `.orchestrator/auth.key`)

---

## 3. Core Features

### P0 Features (MVP — Must Have)

| Feature | Owner | Description | Effort |
|---------|-------|-------------|--------|
| **Go CLI** | Go | `init`, `start`, `stop`, `status` commands | 3 days |
| **Go HTTP API** | Go | State mutation endpoints | 3 days |
| **Go Process Manager** | Go | Spawn/monitor/restart OpenCode instances | 2 days |
| **Go State Manager** | Go | Per-entity files, atomic writes, single writer | 2 days |
| **TS Plugin** | TS | Event hooks, initialization | 2 days |
| **TS Custom Tools** | TS | team_message, task_create, request_checkpoint, sync_workspace | 3 days |
| **Agent Configs** | Config | tech-lead.md, backend-dev.md, frontend-dev.md | 1 day |
| **Checkpoint System** | Go+TS | Terminal prompts, approval/denial flow | 2 days |
| **Shared Secret Auth** | Go+TS | Bearer token for Go ↔ TS communication | 0.5 day |

**MVP Total: ~3.5 weeks (2 developers: 1 Go, 1 TS)**

### P1 Features (v1 — Important)

| Feature | Owner | Effort |
|---------|-------|--------|
| **Dependency Graph** | Go | 3 days |
| **Multi-Repo Spawning** | Go | 2 days |
| **Health Monitoring** | Go | 1 day |
| **Agent Lifecycle** | Go+TS | 2 days |
| **Message Archival** | Go | 1 day |
| **State Migration** | Go | 2 days |
| **Audit Logging** | Go+TS | 1 day |

### P2 Features (v2 — Nice to Have)

| Feature | Owner | Effort |
|---------|-------|--------|
| **TUI Dashboard** | Go (Bubble Tea) | 5 days |
| **Rollback Coordination** | Go+TS | 3 days |
| **Cost Tracking** | TS | 2 days |
| **Agent Templates** | Go | 2 days |
| **Learning System** | TS | 5 days |

---

## 4. Implementation Details

### 4.1 Go Wrapper Structure

```
cmd/orchestrator/
├── main.go              # Entry point, CLI commands
├── init.go              # Workspace initialization
├── start.go             # Start orchestration
├── stop.go              # Stop orchestration
├── status.go            # Status reporting
└── dashboard.go         # TUI dashboard (v2)

pkg/
├── api/
│   ├── server.go        # HTTP API server
│   ├── handlers.go      # Request handlers
│   ├── middleware.go     # Auth middleware
│   └── types.go         # Request/response types
├── process/
│   ├── manager.go       # Process lifecycle
│   ├── health.go        # Health monitoring
│   └── port.go          # Port allocation
├── state/
│   ├── manager.go       # State persistence
│   ├── atomic.go        # Atomic file writes
│   ├── migration.go     # Schema migration
│   └── types.go         # State types
├── checkpoint/
│   ├── handler.go       # Terminal checkpoint UI
│   └── timeout.go       # Checkpoint timeout
├── workspace/
│   ├── workspace.go     # Workspace management
│   └── sync.go          # Plugin file sync
└── types/
    └── types.go         # Shared types
```

### 4.2 TS Plugin Structure

```
.opencode/
├── plugins/
│   └── orchestration.ts     # Main plugin entry point
├── tools/
│   ├── team_message.ts      # Agent messaging
│   ├── task_create.ts       # Task creation
│   ├── task_assign.ts       # Task assignment
│   ├── request_checkpoint.ts # Human approval
│   └── sync_workspace.ts    # State query
└── agents/
    ├── tech-lead.md         # Tech Lead config
    ├── backend-dev.md       # Backend Developer config
    └── frontend-dev.md      # Frontend Developer config
```

### 4.3 State Schema

```json
{
  "version": "1.0.0",
  "project": "auth-rewrite",
  "created_at": "2026-07-17T10:00:00Z",
  "repos": [
    {
      "name": "api-gateway",
      "path": "/path/to/api-gateway",
      "port": 9801,
      "status": "running",
      "pid": 12345
    }
  ],
  "agents": [
    {
      "id": "tech-lead",
      "type": "tech-lead",
      "repo": "all",
      "status": "active"
    },
    {
      "id": "backend-dev-api-gateway",
      "type": "backend-dev",
      "repo": "api-gateway",
      "status": "active"
    }
  ],
  "tasks": {
    "active": ["task-001", "task-002"],
    "completed": ["task-000"]
  },
  "checkpoints": {
    "pending": ["cp-001"],
    "resolved": ["cp-000"]
  },
  "stats": {
    "total_tasks": 3,
    "total_messages": 15,
    "total_checkpoints": 2
  }
}
```

### 4.4 Checkpoint Flow

```
Agent                    TS Plugin                  Go API              User Terminal
  │                         │                         │                      │
  │ request_checkpoint()    │                         │                      │
  │────────────────────────>│                         │                      │
  │                         │ POST /checkpoints       │                      │
  │                         │────────────────────────>│                      │
  │                         │                         │ Display prompt       │
  │                         │                         │─────────────────────>│
  │                         │                         │                      │
  │                         │                         │  User types y/n      │
  │                         │                         │<─────────────────────│
  │                         │                         │                      │
  │                         │  SSE: checkpoint_resolved                      │
  │                         │<────────────────────────│                      │
  │  Checkpoint approved    │                         │                      │
  │<────────────────────────│                         │                      │
  │                         │                         │                      │
  │ Continue working...     │                         │                      │
```

### 4.5 Message Delivery Flow

```
Agent A                   TS Plugin A               Go API              TS Plugin B            Agent B
  │                         │                         │                      │                    │
  │ team_message(to: B)     │                         │                      │                    │
  │────────────────────────>│                         │                      │                    │
  │                         │ POST /messages          │                      │                    │
  │                         │────────────────────────>│                      │                    │
  │                         │                         │ Write to inbox       │                    │
  │                         │                         │  B/msg-001.json      │                    │
  │                         │                         │                      │                    │
  │                         │                         │                      │                    │
  │                         │                         │  (next prompt)       │                    │
  │                         │                         │  GET /messages/B     │                    │
  │                         │                         │<─────────────────────│                    │
  │                         │                         │                      │                    │
  │                         │                         │  Return messages     │                    │
  │                         │                         │─────────────────────>│                    │
  │                         │                         │                      │ Inject into        │
  │                         │                         │                      │ session context    │
  │                         │                         │                      │───────────────────>│
  │                         │                         │                      │                    │
  │                         │                         │                      │                    │
  │                         │                         │                      │   Agent reads      │
  │                         │                         │                      │   message and      │
  │                         │                         │                      │   responds...      │
```

---

## 5. User Workflow

### Setup

```bash
# Install Go wrapper
go install github.com/rhony08/agent-team-orchestration/cmd/orchestrator@latest

# Initialize workspace
crush-orchestrator init my-project \
  --repos ~/code/api-gateway,~/code/user-service,~/code/notification-service

# Start orchestration
crush-orchestrator start

# OpenCode opens in each repo directory
# Use tech-lead agent to coordinate work
```

### Usage

```bash
# In any repo directory, OpenCode is running with orchestration plugin
# Switch to tech-lead agent (Tab key)
# Describe the task:

> Implement JWT authentication across api-gateway, user-service, and notification-service

# Tech Lead decomposes into tasks
# Backend agents are invoked via @backend-dev
# Checkpoints appear in terminal where crush-orchestrator is running
# Approve/deny checkpoints as they appear
```

---

## 6. Security Model

### Authentication
- Shared secret generated on `init`
- Stored in `.orchestrator/auth.key` (mode 0600)
- Bearer token in all Go ↔ TS HTTP requests
- Secret not committed to git (`.gitignore`)

### Authorization
- Agents can only modify files in their assigned repo
- Agents cannot modify `.orchestrator/` or `.opencode/`
- Checkpoints require user approval for destructive operations
- No self-approval (agent cannot approve its own checkpoint)

### Prompt Injection Prevention
- Inter-agent messages wrapped in `<message>` XML tags
- Instruction-like patterns stripped
- Messages cannot override system prompts

### Sandboxing
- File operations validated against repo path
- Bash commands restricted via OpenCode permission system
- Destructive operations require checkpoints

---

## 7. Error Handling

### Graceful Degradation

| Failure | Behavior |
|---------|----------|
| Go wrapper crashes | OpenCode instances continue independently (no coordination) |
| TS plugin fails to load | OpenCode works without orchestration |
| OpenCode instance crashes | Go wrapper restarts (up to 3 times) |
| Go HTTP API unavailable | TS plugin returns error, agent can retry |
| State file corrupted | Restore from backup (`state.json.bak`) |
| Checkpoint timeout | Auto-denied, agent notified |

### Recovery

1. **State corruption:** Restore from `.orchestrator/state.json.bak`
2. **Process crash:** Go wrapper restarts OpenCode instance
3. **Plugin error:** OpenCode continues without orchestration
4. **Network partition:** Agents work independently, sync when reconnected

---

## 8. Testing Strategy

### Unit Tests (Week 1+)
- State manager: atomic writes, concurrent access, corruption recovery
- Process manager: spawn, health check, restart, port allocation
- API handlers: auth, validation, error responses

### Integration Tests (Week 2+)
- Plugin loads in OpenCode
- Custom tools work in agent context
- Checkpoint flow end-to-end
- Message delivery between agents

### Contract Tests (Week 3+)
- Go API ↔ TS plugin request/response contracts
- State schema compatibility

### End-to-End Tests (Week 4+)
- JWT auth across 3 repos (full scenario)
- Agent crash recovery
- Concurrent task creation
- Checkpoint denial and retry

---

## 9. Development Roadmap

### Week 1: Foundation (Go + TS)

**Go:**
- [ ] Set up HTTP API server (Gin)
- [ ] Implement state manager with atomic writes
- [ ] Implement process manager (spawn/kill)
- [ ] Implement `init` command
- [ ] Implement `start` command

**TS:**
- [ ] Set up plugin skeleton
- [ ] Implement `sync_workspace` tool (read state)
- [ ] Spike on OpenCode SDK for multi-repo feasibility

**Shared:**
- [ ] Define API contract (OpenAPI spec)
- [ ] Define shared types
- [ ] Set up shared secret auth

### Week 2: Core Tools + State

**Go:**
- [ ] Implement `stop` and `status` commands
- [ ] Implement task CRUD endpoints
- [ ] Implement message endpoints
- [ ] Implement checkpoint endpoints
- [ ] Implement health monitoring

**TS:**
- [ ] Implement `team_message` tool
- [ ] Implement `task_create` tool
- [ ] Implement `request_checkpoint` tool
- [ ] Test tools in OpenCode agent context

**Shared:**
- [ ] End-to-end: create task → assign → complete
- [ ] End-to-end: send message → receive → respond

### Week 3: Agents + Checkpoints

**Go:**
- [ ] Implement checkpoint terminal UI
- [ ] Implement checkpoint timeout
- [ ] Implement agent registration
- [ ] Implement status summary

**TS:**
- [ ] Create tech-lead.md agent config
- [ ] Create backend-dev.md agent config
- [ ] Create frontend-dev.md agent config
- [ ] Implement context injection on compaction

**Shared:**
- [ ] End-to-end: checkpoint flow (request → approve → continue)
- [ ] End-to-end: agent switching (tech-lead → backend-dev)
- [ ] End-to-end: dependency tracking

### Week 4: Multi-Repo + Polish

**Go:**
- [ ] Implement multi-repo spawning
- [ ] Implement cross-repo dependency detection
- [ ] Create example project with 3 repos
- [ ] Write installation script

**TS:**
- [ ] Implement audit logging
- [ ] Implement message archival
- [ ] Handle edge cases (timeouts, errors)

**Shared:**
- [ ] Integration testing with real microservice project
- [ ] Performance testing with 5+ concurrent tasks
- [ ] Bug fixes

### Week 5: Testing + Release

- [ ] Full end-to-end testing
- [ ] Documentation and README
- [ ] Publish Go binary
- [ ] Publish TS plugin to npm
- [ ] Write blog post / demo

---

## 10. Risk Analysis

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **OpenCode SDK can't spawn sessions** | Medium | High | Spike in Week 1; fallback to manual repo switching |
| **Plugin API changes** | Low | Medium | Pin SDK version, follow changelog |
| **State corruption** | Low | Critical | Atomic writes, backups, single writer |
| **Go ↔ TS latency** | Low | Low | localhost HTTP is sub-ms |
| **Checkpoint timeout** | Medium | Medium | Configurable timeout, auto-deny |
| **Agent prompt injection** | Low | High | Message sanitization, XML wrapping |
| **OpenCode instance crash** | Medium | Medium | Auto-restart (3 attempts) |
| **Context overflow** | Medium | Medium | Task decomposition, message archival |

### Contingency Plans

**If SDK can't spawn sessions:**
- Fallback: User manually starts OpenCode in each repo
- Go wrapper connects to existing instances via ports

**If plugin API changes:**
- Pin `@opencode-ai/plugin` version
- Maintain compatibility layer

**If state corruption occurs:**
- Restore from `state.json.bak`
- Audit log for recovery context

---

## 11. Success Metrics

### Primary KPIs

| Metric | Target | Timeline |
|--------|--------|----------|
| **Time to MVP** | <5 weeks | Week 5 |
| **Setup Time** | <2 minutes | Week 3 |
| **Task Coordination Success** | >80% | Week 6 |
| **Checkpoint Approval Rate** | >90% | Week 6 |
| **Multi-Repo Support** | 3 repos | Week 4 |
| **Memory Usage (Go wrapper)** | <30MB | Week 2 |

### Secondary KPIs

| Metric | Target |
|--------|--------|
| **Go binary size** | <20MB |
| **State file size** | <100KB per 100 tasks |
| **Message latency** | <500ms |
| **Agent spawn time** | <5 seconds |
| **Checkpoint UI response** | <1 second |

---

## 12. Appendix

### A. Files to Create

**Go:**
```
cmd/orchestrator/main.go          # CLI entry point
pkg/api/server.go                  # HTTP API server
pkg/api/handlers.go                # API handlers
pkg/api/middleware.go              # Auth middleware
pkg/process/manager.go             # Process lifecycle
pkg/process/health.go              # Health monitoring
pkg/state/manager.go               # State persistence
pkg/state/atomic.go                # Atomic writes
pkg/checkpoint/handler.go          # Checkpoint UI
pkg/workspace/workspace.go         # Workspace management
pkg/types/types.go                 # Shared types
```

**TS:**
```
.opencode/plugins/orchestration.ts  # Main plugin
.opencode/tools/team_message.ts     # Messaging tool
.opencode/tools/task_create.ts      # Task tool
.opencode/tools/request_checkpoint.ts # Checkpoint tool
.opencode/tools/sync_workspace.ts   # State query tool
.opencode/agents/tech-lead.md       # Tech Lead agent
.opencode/agents/backend-dev.md     # Backend agent
.opencode/agents/frontend-dev.md    # Frontend agent
```

### B. Related Documents

- [ADR-001: Plugin-first architecture with Go CLI wrapper](./ADR-001-plugin-first-architecture.md)
- [Requirements Document](./REQUIREMENTS.md)
- [Original Product Plan](./PRODUCT-PLAN.md)
- [Backend Architecture](./ARCHITECTURE-BACKEND.md)

### C. OpenCode Extension Points Used

| Extension Point | How We Use It |
|-----------------|---------------|
| **Plugins** | Event hooks, context injection |
| **Custom Tools** | Agent communication, task management |
| **Agents** | Specialized roles |
| **Permissions** | Sandbox, checkpoint flow |
| **SDK** | Status queries (optional) |
