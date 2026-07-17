# TODO List: Agent Team Orchestration

**Last Updated:** 2026-07-17
**Total Estimated Effort:** 5 weeks (2 developers)

---

## Phase 0: Spike & Validation (Week 0)

### TODO-001: Validate OpenCode SDK Multi-Instance Feasibility
**Priority:** 🔴 Critical (Blocking)
**Effort:** 2 days
**Owner:** TS Developer
**Status:** Not Started

**Description:**
Verify that OpenCode's SDK can create and manage sessions programmatically. This is the foundation for multi-repo coordination.

**Tasks:**
- [ ] Install `@opencode-ai/sdk` in a test project
- [ ] Call `createOpencode()` to start an instance
- [ ] Call `session.create()` to create a session
- [ ] Call `session.prompt()` to send a message
- [ ] Verify response is received
- [ ] Test starting multiple instances on different ports
- [ ] Document SDK capabilities and limitations

**DoD:**
- Can create 2+ OpenCode instances on different ports
- Can send prompts and receive responses programmatically
- Documented any limitations or blockers

**Edge Cases:**
- What if SDK doesn't support session creation?
- What if port conflicts occur?
- What if instances can't communicate?

**Fallback:**
If SDK doesn't support multi-instance, pivot to:
- User manually starts OpenCode in each repo
- Go wrapper connects to existing instances via ports
- Reduced automation, but still functional

---

### TODO-002: Define OpenAPI Contract Between Go and TS
**Priority:** 🔴 Critical (Blocking)
**Effort:** 1 day
**Owner:** Both
**Status:** Not Started

**Description:**
Define the HTTP API contract between Go wrapper and TS plugin.

**Tasks:**
- [ ] Define all endpoints (tasks, messages, checkpoints, status)
- [ ] Define request/response schemas
- [ ] Define error responses
- [ ] Define authentication header format
- [ ] Generate OpenAPI spec file
- [ ] Review with team

**DoD:**
- OpenAPI spec file at `docs/api-spec.yaml`
- Both Go and TS developers agree on contract
- All edge cases documented

**Deliverable:**
```yaml
# docs/api-spec.yaml
openapi: 3.0.0
info:
  title: Orchestration API
  version: 1.0.0
paths:
  /api/v1/tasks:
    post:
      summary: Create task
      ...
```

---

## Phase 1: Foundation (Week 1)

### TODO-101: Set Up Go HTTP API Server
**Priority:** 🔴 High
**Effort:** 2 days
**Owner:** Go Developer
**Dependencies:** TODO-002

**Description:**
Create the HTTP API server using Gin (already in go.mod).

**Tasks:**
- [ ] Create `pkg/api/server.go` with Gin router
- [ ] Create `pkg/api/middleware.go` with auth middleware
- [ ] Create `pkg/api/types.go` with request/response types
- [ ] Implement `/health` endpoint
- [ ] Implement `/api/v1/status` endpoint
- [ ] Add CORS for localhost only
- [ ] Add request logging middleware
- [ ] Write unit tests for middleware

**DoD:**
- Server starts on configurable port (default 9800)
- Auth middleware validates Bearer token
- Health endpoint returns 200
- Unit tests pass

**Code Structure:**
```go
// pkg/api/server.go
type Server struct {
    router     *gin.Engine
    state      *state.Manager
    port       int
    authSecret string
}

func NewServer(state *state.Manager, port int, secret string) *Server {
    s := &Server{...}
    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    s.router.GET("/health", s.healthHandler)
    api := s.router.Group("/api/v1", s.authMiddleware())
    {
        api.POST("/tasks", s.createTask)
        api.GET("/tasks", s.listTasks)
        // ... more routes
    }
}
```

**Edge Cases:**
- Port already in use → try next port
- Invalid auth token → 401
- Malformed JSON body → 400
- Server already running → error

---

### TODO-102: Implement Go State Manager
**Priority:** 🔴 High
**Effort:** 2 days
**Owner:** Go Developer
**Dependencies:** None

**Description:**
Implement file-based state management with atomic writes.

**Tasks:**
- [ ] Create `pkg/state/manager.go` with state CRUD
- [ ] Create `pkg/state/atomic.go` with atomic write utilities
- [ ] Create `pkg/state/types.go` with state types
- [ ] Implement per-entity file storage (tasks/*.json, messages/*.json)
- [ ] Implement atomic writes (write-to-temp, rename)
- [ ] Implement state summary (`state.json`)
- [ ] Implement backup (`state.json.bak`)
- [ ] Write unit tests for concurrent access
- [ ] Write unit tests for corruption recovery

**DoD:**
- Tasks stored as individual files
- Atomic writes (no partial files on crash)
- Concurrent writes serialized with mutex
- Backup created before each write
- Unit tests pass with 100 concurrent goroutines

**Code Structure:**
```go
// pkg/state/manager.go
type Manager struct {
    basePath string
    mu       sync.RWMutex
}

func (m *Manager) CreateTask(task Task) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }
    
    path := filepath.Join(m.basePath, "tasks", "active", task.ID+".json")
    return atomicWrite(path, data)
}

// pkg/state/atomic.go
func atomicWrite(path string, data []byte) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0644); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
```

**Edge Cases:**
- Disk full during write → temp file remains, error returned
- Process killed during rename → OS guarantees atomicity
- Corrupted JSON file → skip, log error
- Directory doesn't exist → create it

---

### TODO-103: Implement Go Process Manager
**Priority:** 🔴 High
**Effort:** 2 days
**Owner:** Go Developer
**Dependencies:** None

**Description:**
Manage OpenCode process lifecycle.

**Tasks:**
- [ ] Create `pkg/process/manager.go` with spawn/kill
- [ ] Create `pkg/process/health.go` with health checks
- [ ] Create `pkg/process/port.go` with port allocation
- [ ] Implement process spawning with `exec.Command`
- [ ] Implement PID file management
- [ ] Implement health check (HTTP GET to OpenCode)
- [ ] Implement auto-restart (up to 3 attempts)
- [ ] Implement graceful shutdown (SIGTERM → SIGKILL)
- [ ] Write unit tests

**DoD:**
- Can spawn OpenCode on specific port
- Health check detects running/crashed instances
- Auto-restart works
- Graceful shutdown kills all processes

**Code Structure:**
```go
// pkg/process/manager.go
type Manager struct {
    processes map[string]*Process
    mu        sync.Mutex
}

type Process struct {
    Name    string
    Path    string
    Port    int
    PID     int
    Cmd     *exec.Cmd
    Status  string
}

func (m *Manager) Spawn(name, path string, port int) error {
    cmd := exec.Command("opencode")
    cmd.Dir = path
    cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
    // ...
}

func (m *Manager) HealthCheck() map[string]string {
    // HTTP GET to each instance
}
```

**Edge Cases:**
- OpenCode not in PATH → error with install instructions
- Port already in use → try next port
- Process crashes immediately → retry 3 times, then fail
- Zombie processes → kill by PID

---

### TODO-104: Implement Go CLI Commands
**Priority:** 🔴 High
**Effort:** 2 days
**Owner:** Go Developer
**Dependencies:** TODO-101, TODO-102, TODO-103

**Description:**
Implement CLI commands using Cobra (already in go.mod).

**Tasks:**
- [ ] Implement `init` command
  - [ ] Create `.orchestrator/` directory structure
  - [ ] Generate shared secret
  - [ ] Create `config.json`
  - [ ] Copy plugin files to repos
  - [ ] Update `.gitignore`
- [ ] Implement `start` command
  - [ ] Start HTTP API server
  - [ ] Spawn OpenCode instances
  - [ ] Wait for health checks
- [ ] Implement `stop` command
  - [ ] Graceful shutdown
  - [ ] Cleanup PID files
- [ ] Implement `status` command
  - [ ] Show running state
  - [ ] Show task/checkpoint counts
- [ ] Write integration tests

**DoD:**
- `init` creates workspace in <2 seconds
- `start` launches all instances in <10 seconds
- `stop` terminates all processes
- `status` shows current state

**Edge Cases:**
- Workspace already exists → error or --force
- Not in a git repo → error
- Multiple repos on same filesystem → valid
- Single repo → valid (single-repo mode)

---

### TODO-105: Set Up TS Plugin Skeleton
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** TS Developer
**Dependencies:** TODO-001

**Description:**
Create the OpenCode plugin skeleton.

**Tasks:**
- [ ] Create `.opencode/plugins/orchestration.ts`
- [ ] Implement plugin function with context
- [ ] Implement `session.created` hook (connect to Go API)
- [ ] Implement `session.idle` hook (check messages)
- [ ] Implement error handling
- [ ] Test plugin loads in OpenCode

**DoD:**
- Plugin loads without errors
- Connects to Go API on session start
- Logs initialization status

**Code Structure:**
```typescript
// .opencode/plugins/orchestration.ts
import type { Plugin } from "@opencode-ai/plugin"

export const OrchestrationPlugin: Plugin = async ({ project, client, $, directory }) => {
  const apiClient = new OrchestratorAPI(process.env.ORCHESTRATOR_SECRET!)
  
  return {
    "session.created": async () => {
      try {
        await apiClient.connect()
        await client.app.log({
          body: { service: "orchestration", level: "info", message: "Connected" },
        })
      } catch (e) {
        // Graceful degradation
      }
    },
    "session.idle": async () => {
      // Check for pending messages
    },
  }
}
```

**Edge Cases:**
- Go API not running → graceful degradation
- Secret mismatch → refuse to load
- Plugin syntax error → OpenCode reports error

---

### TODO-106: Create sync_workspace Tool
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** TS Developer
**Dependencies:** TODO-105

**Description:**
Create tool for querying orchestration state.

**Tasks:**
- [ ] Create `.opencode/tools/sync_workspace.ts`
- [ ] Implement `status` action (summary)
- [ ] Implement `get_tasks` action (with filters)
- [ ] Implement `get_messages` action (agent inbox)
- [ ] Implement `get_checkpoints` action
- [ ] Handle Go API errors gracefully
- [ ] Test tool in agent context

**DoD:**
- Tool appears in OpenCode tool list
- Can query tasks, messages, checkpoints
- Error messages are actionable

**Edge Cases:**
- Go API unavailable → "Orchestrator not running"
- Large result set → paginate (max 50)
- Filter matches nothing → empty array

---

## Phase 2: Core Tools + State (Week 2)

### TODO-201: Implement Task CRUD Endpoints (Go)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** Go Developer
**Dependencies:** TODO-101, TODO-102

**Description:**
Implement task creation, listing, updating endpoints.

**Tasks:**
- [ ] `POST /api/v1/tasks` — Create task
- [ ] `GET /api/v1/tasks` — List tasks (with filters)
- [ ] `GET /api/v1/tasks/:id` — Get task
- [ ] `PATCH /api/v1/tasks/:id` — Update task
- [ ] `POST /api/v1/tasks/:id/complete` — Mark complete
- [ ] Validate dependencies exist
- [ ] Detect circular dependencies
- [ ] Write unit tests

**DoD:**
- All CRUD operations work
- Dependencies validated
- Circular dependencies detected

**Edge Cases:**
- Duplicate task ID → 409 Conflict
- Invalid dependency → 400 Bad Request
- Task not found → 404

---

### TODO-202: Implement Message Endpoints (Go)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** Go Developer
**Dependencies:** TODO-101, TODO-102

**Description:**
Implement message sending and inbox retrieval.

**Tasks:**
- [ ] `POST /api/v1/messages` — Send message
- [ ] `GET /api/v1/messages/:agent` — Get agent inbox
- [ ] `POST /api/v1/messages/:agent/ack` — Acknowledge messages
- [ ] Implement broadcast (to: "all")
- [ ] Implement message archival (7-day TTL)
- [ ] Write unit tests

**DoD:**
- Messages delivered to inboxes
- Inbox returns messages ordered by timestamp
- Old messages archived

**Edge Cases:**
- Recipient doesn't exist → 404
- Content too large (>10KB) → 400
- Concurrent writes → UUID prevents collision

---

### TODO-203: Implement Checkpoint Endpoints (Go)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** Go Developer
**Dependencies:** TODO-101, TODO-102

**Description:**
Implement checkpoint creation and resolution.

**Tasks:**
- [ ] `POST /api/v1/checkpoints` — Create checkpoint
- [ ] `GET /api/v1/checkpoints` — List pending
- [ ] `POST /api/v1/checkpoints/:id/approve` — Approve
- [ ] `POST /api/v1/checkpoints/:id/deny` — Deny with reason
- [ ] Implement timeout (5 minutes default)
- [ ] Write unit tests

**DoD:**
- Checkpoints created with "pending" status
- Approval/denial updates status
- Timeout auto-denies

**Edge Cases:**
- Checkpoint already resolved → 409
- Invalid checkpoint ID → 404
- Timeout while user is typing → respect user input

---

### TODO-204: Implement Checkpoint Terminal UI (Go)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** Go Developer
**Dependencies:** TODO-203

**Description:**
Display checkpoint prompts in terminal for user approval.

**Tasks:**
- [ ] Display checkpoint description in terminal
- [ ] Show affected repos
- [ ] Accept y/n input
- [ ] Accept denial reason on 'n'
- [ ] Show queue if multiple pending
- [ ] Implement timeout countdown

**DoD:**
- Checkpoint appears clearly in terminal
- User can approve (y) or deny (n + reason)
- Queue displays when multiple pending

**Code Structure:**
```go
func (h *CheckpointHandler) Display(cp Checkpoint) bool {
    fmt.Printf("\n╔══════════════════════════════════════╗\n")
    fmt.Printf("║  CHECKPOINT: %s\n", cp.Type)
    fmt.Printf("╠══════════════════════════════════════╣\n")
    fmt.Printf("║  %s\n", cp.Description)
    fmt.Printf("║  Affected: %s\n", strings.Join(cp.AffectedRepos, ", "))
    fmt.Printf("╚══════════════════════════════════════╝\n")
    fmt.Printf("  Approve? (y/n): ")
    // ...
}
```

---

### TODO-205: Implement team_message Tool (TS)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** TS Developer
**Dependencies:** TODO-105, TODO-202

**Description:**
Create tool for agent-to-agent messaging.

**Tasks:**
- [ ] Create `.opencode/tools/team_message.ts`
- [ ] Implement message sending via Go API
- [ ] Implement broadcast (to: "all")
- [ ] Sanitize message content
- [ ] Test tool in agent context

**DoD:**
- Tool sends messages via Go API
- Messages appear in recipient's inbox
- Content sanitized against injection

**Edge Cases:**
- Recipient doesn't exist → error
- Content empty → error
- Content has prompt injection → sanitize

---

### TODO-206: Implement task_create Tool (TS)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** TS Developer
**Dependencies:** TODO-105, TODO-201

**Description:**
Create tool for task creation.

**Tasks:**
- [ ] Create `.opencode/tools/task_create.ts`
- [ ] Implement task creation via Go API
- [ ] Validate dependencies
- [ ] Test tool in agent context

**DoD:**
- Tool creates tasks via Go API
- Dependencies validated
- Task ID returned

---

### TODO-207: Implement request_checkpoint Tool (TS)
**Priority:** 🔴 High
**Effort:** 1 day
**Owner:** TS Developer
**Dependencies:** TODO-105, TODO-203

**Description:**
Create tool for requesting human approval.

**Tasks:**
- [ ] Create `.opencode/tools/request_checkpoint.ts`
- [ ] Implement checkpoint creation via Go API
- [ ] Implement waiting for resolution (polling or SSE)
- [ ] Return approval/denial to agent
- [ ] Test tool in agent context

**DoD:**
- Tool creates checkpoint
- Agent waits for user decision
- Approval/denial returned to agent

**Edge Cases:**
- Timeout → auto-deny
- User denies → agent receives reason
- Go API unavailable → error

---

### TODO-208: End-to-End Test: Task + Message Flow
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Both
**Dependencies:** TODO-201 through TODO-207

**Description:**
Test complete task and message flow.

**Test Scenario:**
1. Start orchestration
2. Create task via task_create
3. Send message via team_message
4. Query state via sync_workspace
5. Verify all data consistent

**DoD:**
- All tools work end-to-end
- No data loss
- Latency <500ms

---

## Phase 3: Agents + Checkpoints (Week 3)

### TODO-301: Create Tech Lead Agent Config
**Priority:** 🔴 High
**Effort:** 0.5 day
**Owner:** TS Developer
**Dependencies:** None

**Description:**
Create tech-lead.md agent configuration.

**Tasks:**
- [ ] Create `.opencode/agents/tech-lead.md`
- [ ] Define capabilities and responsibilities
- [ ] Define permissions (edit: ask, bash: restricted)
- [ ] Define tool access
- [ ] Define system prompt
- [ ] Test agent switching

**DoD:**
- Agent appears in OpenCode agent list
- Permissions enforced
- System prompt effective

---

### TODO-302: Create Backend Developer Agent Config
**Priority:** 🔴 High
**Effort:** 0.5 day
**Owner:** TS Developer
**Dependencies:** None

**Description:**
Create backend-dev.md agent configuration.

**Tasks:**
- [ ] Create `.opencode/agents/backend-dev.md`
- [ ] Define capabilities
- [ ] Define permissions (edit: allow, git commit: ask)
- [ ] Define system prompt
- [ ] Test agent switching

---

### TODO-303: Create Frontend Developer Agent Config
**Priority:** 🟡 Medium
**Effort:** 0.5 day
**Owner:** TS Developer
**Dependencies:** None

**Description:**
Create frontend-dev.md agent configuration.

**Tasks:**
- [ ] Create `.opencode/agents/frontend-dev.md`
- [ ] Define capabilities
- [ ] Define permissions
- [ ] Define system prompt

---

### TODO-304: Implement Agent Registration (Go)
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Go Developer
**Dependencies:** TODO-101, TODO-102

**Description:**
Track registered agents in state.

**Tasks:**
- [ ] `POST /api/v1/agents` — Register agent
- [ ] `GET /api/v1/agents` — List agents
- [ ] Track agent status (active/idle/failed)
- [ ] Write unit tests

---

### TODO-305: Implement Context Injection on Compaction (TS)
**Priority:** 🟡 Medium
**Effort:** 0.5 day
**Owner:** TS Developer
**Dependencies:** TODO-105

**Description:**
Inject orchestration state when OpenCode compacts context.

**Tasks:**
- [ ] Implement `experimental.session.compacting` hook
- [ ] Query Go API for current state
- [ ] Format state summary
- [ ] Inject into compaction context
- [ ] Add fallback if hook removed

**DoD:**
- Orchestration state preserved across compaction
- Fallback works if hook removed

---

### TODO-306: End-to-End Test: Agent + Checkpoint Flow
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Both
**Dependencies:** TODO-301 through TODO-305

**Description:**
Test complete agent and checkpoint flow.

**Test Scenario:**
1. Start orchestration
2. Switch to tech-lead agent
3. Create tasks
4. Switch to backend-dev agent
5. Work on task
6. Request checkpoint
7. Approve in terminal
8. Verify task progress

**DoD:**
- Agent switching works
- Checkpoint flow works
- State consistent

---

## Phase 4: Multi-Repo + Polish (Week 4)

### TODO-401: Implement Multi-Repo Spawning (Go)
**Priority:** 🟡 Medium
**Effort:** 2 days
**Owner:** Go Developer
**Dependencies:** TODO-103, TODO-001

**Description:**
Spawn OpenCode instances in each repo directory.

**Tasks:**
- [ ] Implement per-repo process spawning
- [ ] Implement port allocation (9801, 9802, ...)
- [ ] Implement plugin file copying to each repo
- [ ] Implement config merging
- [ ] Test with 3 repos
- [ ] Write integration tests

**DoD:**
- 3 repos → 3 OpenCode instances
- Each has plugin loaded
- Each on unique port

**Edge Cases:**
- Repo has no `.opencode/` → create it
- Repo has existing `opencode.json` → merge
- Instance crashes → restart

---

### TODO-402: Implement Cross-Repo Dependency Detection (Go)
**Priority:** 🟡 Medium
**Effort:** 2 days
**Owner:** Go Developer
**Dependencies:** TODO-201

**Description:**
Detect dependencies between tasks in different repos.

**Tasks:**
- [ ] Implement dependency graph data structure
- [ ] Implement cycle detection
- [ ] Implement topological sort for execution order
- [ ] Block tasks with unmet dependencies
- [ ] Unblock when dependency met
- [ ] Write unit tests

**DoD:**
- Dependencies tracked correctly
- Cycles detected at creation
- Tasks execute in order

---

### TODO-403: Implement Audit Logging (Go + TS)
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Both
**Dependencies:** TODO-101, TODO-105

**Description:**
Log all orchestration actions.

**Tasks:**
- [ ] Implement append-only log in Go
- [ ] Log all API requests
- [ ] Log all tool executions in TS
- [ ] Implement log rotation (10MB, 5 files)
- [ ] JSON lines format

**DoD:**
- All actions logged
- Log rotates automatically
- JSON lines format

---

### TODO-404: Create Example Project
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Both
**Dependencies:** TODO-401

**Description:**
Create example project with 3 repos for demo.

**Tasks:**
- [ ] Create `examples/` directory
- [ ] Create 3 sample repos (api-gateway, user-service, notification-service)
- [ ] Create example orchestration scenario
- [ ] Write README with instructions
- [ ] Test full flow

**DoD:**
- Example works end-to-end
- README is clear

---

### TODO-405: Bug Fixes and Edge Cases
**Priority:** 🟡 Medium
**Effort:** 2 days
**Owner:** Both
**Dependencies:** All previous

**Description:**
Fix bugs found during testing.

**Tasks:**
- [ ] Fix all known issues
- [ ] Handle edge cases
- [ ] Improve error messages
- [ ] Performance optimization

---

## Phase 5: Testing + Release (Week 5)

### TODO-501: Full End-to-End Testing
**Priority:** 🔴 High
**Effort:** 2 days
**Owner:** Both

**Test Scenarios:**
- [ ] JWT auth across 3 repos
- [ ] Emergency hotfix flow
- [ ] Agent crash recovery
- [ ] Concurrent task creation
- [ ] Checkpoint denial and retry
- [ ] Multi-repo dependency ordering
- [ ] State corruption recovery
- [ ] Plugin failure graceful degradation

---

### TODO-502: Documentation
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Both

**Tasks:**
- [ ] Write README.md
- [ ] Write installation guide
- [ ] Write usage guide
- [ ] Write architecture overview
- [ ] Write troubleshooting guide

---

### TODO-503: Release
**Priority:** 🟡 Medium
**Effort:** 1 day
**Owner:** Both

**Tasks:**
- [ ] Build Go binary for multiple platforms
- [ ] Publish TS plugin to npm
- [ ] Create GitHub release
- [ ] Write release notes
- [ ] Announce to community

---

## Summary

| Phase | Weeks | Tasks | Effort |
|-------|-------|-------|--------|
| Phase 0: Spike | 0 | 2 | 3 days |
| Phase 1: Foundation | 1 | 6 | 10 days |
| Phase 2: Core Tools | 2 | 8 | 8 days |
| Phase 3: Agents | 3 | 6 | 5 days |
| Phase 4: Multi-Repo | 4 | 5 | 8 days |
| Phase 5: Release | 5 | 3 | 4 days |
| **Total** | **5 weeks** | **30 tasks** | **~38 days** |

With 2 developers working in parallel: **~5 weeks**
