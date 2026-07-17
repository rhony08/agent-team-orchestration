# Requirements Document: Agent Team Orchestration

**Version:** 1.0
**Last Updated:** 2026-07-17
**Status:** Draft

---

## Table of Contents

1. [Go Wrapper CLI](#1-go-wrapper-cli)
2. [State Management](#2-state-management)
3. [OpenCode Plugin](#3-opencode-plugin)
4. [Custom Tools](#4-custom-tools)
5. [Agent Configurations](#5-agent-configurations)
6. [Multi-Repo Coordination](#6-multi-repo-coordination)
7. [Checkpoint System](#7-checkpoint-system)
8. [Message Passing](#8-message-passing)
9. [Security](#9-security)
10. [Observability](#10-observability)

---

## 1. Go Wrapper CLI

### REQ-CLI-001: Workspace Initialization

**Description:** `crush-orchestrator init <project-name> --repos <repo1,repo2,...>` initializes a new orchestration workspace.

**Acceptance Criteria:**
- [ ] Creates `.orchestrator/` directory structure
- [ ] Generates `config.json` with project metadata
- [ ] Generates shared secret for Go ↔ TS authentication
- [ ] Creates `.opencode/` directories in each repo (plugins, tools, agents)
- [ ] Copies plugin/tool/agent files to each repo
- [ ] Validates all repo paths exist and are git repositories
- [ ] Creates `.gitignore` entry for `.orchestrator/`

**Definition of Done:**
- Command completes in <2 seconds
- All directories created with correct permissions
- Shared secret is 32-byte hex string
- Error messages are actionable (e.g., "Repo not found: /path/to/repo")

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Project name contains spaces | Reject with error: "Project name must be alphanumeric" |
| Repo path doesn't exist | Error: "Repository not found: {path}" |
| Repo path exists but not a git repo | Error: "Not a git repository: {path}" |
| `.orchestrator/` already exists | Error: "Workspace already exists. Use --force to overwrite." |
| `--force` flag used | Back up existing `.orchestrator/` to `.orchestrator.bak/` |
| No repos specified | Error: "At least one repository required" |
| Single repo specified | Valid (single-repo orchestration) |
| Repo is a git worktree | Valid (worktree is a valid git context) |

---

### REQ-CLI-002: Start Orchestration

**Description:** `crush-orchestrator start` launches OpenCode instances for each repo and starts the orchestration server.

**Acceptance Criteria:**
- [ ] Starts Go HTTP API server on configurable port (default: 9800)
- [ ] Spawns one OpenCode process per repo
- [ ] Each OpenCode instance gets a unique port (9801, 9802, ...)
- [ ] Waits for all OpenCode instances to be healthy before reporting "started"
- [ ] Writes PID files for each OpenCode process
- [ ] Registers signal handlers for graceful shutdown (SIGINT, SIGTERM)

**Definition of Done:**
- All OpenCode instances report healthy within 10 seconds
- `crush-orchestrator status` shows all instances as "running"
- Ctrl+C triggers graceful shutdown of all instances

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Port 9800 already in use | Try next port, or error with "Port {port} in use" |
| OpenCode binary not found | Error: "opencode not found in PATH" |
| OpenCode fails to start | Log error, continue with remaining repos, report partial start |
| OpenCode crashes after start | Restart up to 3 times, then mark as "failed" and continue |
| System has <500MB free RAM | Warning: "Low memory. Consider reducing concurrent agents." |
| `start` called when already running | Error: "Already running. Use `stop` first." |

---

### REQ-CLI-003: Stop Orchestration

**Description:** `crush-orchestrator stop` gracefully shuts down all OpenCode instances and the API server.

**Acceptance Criteria:**
- [ ] Sends SIGTERM to all OpenCode processes
- [ ] Waits up to 10 seconds for graceful shutdown
- [ ] Sends SIGKILL if process doesn't exit within timeout
- [ ] Closes HTTP API server
- [ ] Cleans up PID files
- [ ] Preserves state files (does not delete `.orchestrator/`)

**Definition of Done:**
- All processes terminated
- No orphaned OpenCode processes
- State preserved for next `start`

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| OpenCode process already dead | Skip, clean up PID file |
| `stop` called when not running | Error: "Not running" |
| Partial start (some repos failed) | Stop only running instances |
| User sends SIGKILL to wrapper | Orphaned OpenCode processes remain; `start` should detect and clean up |

---

### REQ-CLI-004: Status Reporting

**Description:** `crush-orchestrator status` shows current orchestration state.

**Acceptance Criteria:**
- [ ] Shows running/stopped state
- [ ] Lists all repos and their OpenCode instance status
- [ ] Shows active tasks, pending checkpoints, recent messages
- [ ] Shows port numbers for each instance
- [ ] Output is human-readable (table format)

**Definition of Done:**
- Status command completes in <1 second
- Output is clear and actionable

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Not running | Show "Not running" with hint to `start` |
| Partial state (some repos crashed) | Show per-repo status with "failed" indicator |
| State file corrupted | Show "State corrupted" with recovery hint |

---

## 2. State Management

### REQ-STATE-001: State Ownership

**Description:** Go wrapper is the single canonical owner of orchestration state.

**Acceptance Criteria:**
- [ ] Go wrapper serves state via HTTP API (localhost only)
- [ ] TS plugin reads state via HTTP GET
- [ ] TS plugin requests state mutations via HTTP POST
- [ ] No direct file writes from TS plugin to `.orchestrator/`
- [ ] Go wrapper validates all mutation requests

**Definition of Done:**
- Single writer (Go) for all state files
- TS plugin is read-only on filesystem
- All mutations go through Go HTTP API

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Go HTTP server not available | TS plugin returns error to agent, suggests checking orchestrator |
| Concurrent mutation requests | Go serializes requests with mutex |
| Invalid mutation data | Go returns 400 with validation error |
| TS plugin sends malformed request | Go rejects with 400, logs error |

---

### REQ-STATE-002: Atomic Writes

**Description:** All state file writes are atomic (write-to-temp, then rename).

**Acceptance Criteria:**
- [ ] All writes use `write to tmpfile → rename` pattern
- [ ] Temp files are in same directory as target (for atomic rename)
- [ ] Failed writes leave no partial files
- [ ] State files are always valid JSON (never partial/corrupt)

**Definition of Done:**
- Pull plug during write → state file is either old or new, never corrupt
- 1000 concurrent writes → no corruption

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Disk full during write | Temp file remains, error returned, old state preserved |
| Process killed during rename | OS guarantees rename is atomic on POSIX |
| Windows filesystem | Use `os.Rename` which handles Windows semantics |

---

### REQ-STATE-003: Per-Entity State Files

**Description:** State is stored as individual files per entity, not a single monolithic file.

**Directory Structure:**
```
.orchestrator/
├── config.json                    # Project config (read-only after init)
├── state.json                     # Summary state (lightweight, for display)
├── auth.key                       # Shared secret
├── tasks/
│   ├── active/
│   │   ├── task-001.json
│   │   └── task-002.json
│   └── completed/
│       └── task-000.json
├── messages/
│   ├── inbox/
│   │   ├── tech-lead/
│   │   │   └── msg-001.json
│   │   └── backend-dev-api-gateway/
│   │       └── msg-002.json
│   └── archive/                   # Messages older than 7 days
├── checkpoints/
│   ├── pending/
│   │   └── cp-001.json
│   └── resolved/
│       └── cp-000.json
├── agents/
│   ├── tech-lead.json
│   ├── backend-dev-api-gateway.json
│   └── frontend-dev-ui.json
└── audit.log                      # Append-only audit trail
```

**Acceptance Criteria:**
- [ ] Each task is a separate file in `tasks/active/` or `tasks/completed/`
- [ ] Each message is a separate file in `messages/inbox/{agent-id}/`
- [ ] Each checkpoint is a separate file in `checkpoints/pending/` or `checkpoints/resolved/`
- [ ] `state.json` is a lightweight summary (updated on every mutation)
- [ ] Completed items are moved, not deleted (audit trail)

**Definition of Done:**
- File count scales linearly with entity count
- No single file exceeds 100KB
- Individual entity reads are O(1)

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| 1000+ tasks | Directory listing is fast (<100ms) |
| Entity file corrupted | Skip entity, log error, continue |
| Agent ID contains path separators | Sanitize agent ID (replace `/` with `-`) |
| Concurrent file creation | Use unique IDs (UUID) to avoid collisions |

---

### REQ-STATE-004: State Schema Migration

**Description:** State schema includes version and migration support.

**Acceptance Criteria:**
- [ ] `config.json` contains `version` field
- [ ] On load, Go wrapper checks version compatibility
- [ ] Migration functions exist for each version bump
- [ ] Migrations are idempotent (safe to run multiple times)
- [ ] Backup created before migration

**Definition of Done:**
- v1.0 → v1.1 migration works without data loss
- Corrupted migration can be rolled back from backup

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Unknown version | Error: "Unsupported state version: {version}" |
| Downgrade (newer → older) | Error: "Cannot downgrade from {new} to {old}" |
| Migration fails midway | Rollback from backup, error with details |
| Empty `.orchestrator/` directory | Initialize fresh state at current version |

---

## 3. OpenCode Plugin

### REQ-PLUGIN-001: Plugin Initialization

**Description:** Plugin initializes on session start, connects to Go wrapper.

**Acceptance Criteria:**
- [ ] Plugin loads from `.opencode/plugins/orchestration.ts`
- [ ] On `session.created`, plugin connects to Go HTTP API
- [ ] Plugin validates shared secret
- [ ] Plugin logs initialization status
- [ ] Plugin fails gracefully if Go wrapper is not running

**Definition of Done:**
- Plugin loads in <1 second
- Connection to Go API established
- Shared secret validated

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Go wrapper not running | Plugin logs warning, operates in degraded mode |
| Shared secret mismatch | Plugin refuses to load, error message |
| Plugin file has syntax error | OpenCode reports error, continues without plugin |
| Multiple OpenCode instances | Each plugin instance connects independently |

---

### REQ-PLUGIN-002: Event Hooks

**Description:** Plugin hooks into OpenCode events for orchestration.

**Events Used:**
| Event | Purpose |
|-------|---------|
| `session.created` | Initialize plugin, connect to Go API |
| `session.idle` | Check for pending messages, dependent tasks |
| `tool.execute.after` | Log tool usage for audit |
| `permission.replied` | Resolve checkpoint approvals |
| `experimental.session.compacting` | Inject orchestration context |

**Acceptance Criteria:**
- [ ] All event handlers are async and non-blocking
- [ ] Event handler failures don't crash the session
- [ ] Events are logged to audit trail

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Event handler throws | Log error, continue session |
| Event fires before plugin init | Skip handler (guard clause) |
| Experimental hook removed | Fallback: inject context via system message |

---

## 4. Custom Tools

### REQ-TOOL-001: team_message Tool

**Description:** Send messages between agents.

**Arguments:**
| Arg | Type | Required | Description |
|-----|------|----------|-------------|
| `to` | string | Yes | Agent ID or "all" |
| `type` | enum | Yes | status_update, question, dependency_alert, task_assignment |
| `content` | string | Yes | Message content |

**Acceptance Criteria:**
- [ ] Message is written to recipient's inbox via Go API
- [ ] Message has unique ID (UUID)
- [ ] Message includes sender, timestamp, type
- [ ] Broadcast (`to: "all"`) writes to all agent inboxes
- [ ] Message content is sanitized (strip instruction-like patterns)

**Definition of Done:**
- Message appears in recipient's inbox within 500ms
- Sender receives confirmation with message ID

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Recipient agent doesn't exist | Error: "Agent not found: {id}" |
| Content is empty | Error: "Message content required" |
| Content contains prompt injection | Sanitize: wrap in `<message>` tags, strip `ignore previous` patterns |
| Content >10KB | Error: "Message too large (max 10KB)" |
| `to: "all"` with no agents | Warning: "No agents registered" |

---

### REQ-TOOL-002: task_create Tool

**Description:** Create a new task.

**Arguments:**
| Arg | Type | Required | Description |
|-----|------|----------|-------------|
| `title` | string | Yes | Task title |
| `description` | string | Yes | Task description |
| `assignee` | string | No | Agent ID |
| `dependencies` | string[] | No | Task IDs |
| `repo` | string | No | Repository name |

**Acceptance Criteria:**
- [ ] Task is created via Go API
- [ ] Task ID is UUID
- [ ] Task status defaults to "pending"
- [ ] Dependencies are validated (must exist)
- [ ] No circular dependencies

**Definition of Done:**
- Task created and returned with ID
- Dependencies validated

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Dependency doesn't exist | Error: "Task not found: {id}" |
| Circular dependency detected | Error: "Circular dependency: task-A → task-B → task-A" |
| Title is empty | Error: "Title required" |
| Assignee doesn't exist | Warning: "Agent not found: {id}. Task created unassigned." |

---

### REQ-TOOL-003: request_checkpoint Tool

**Description:** Request human approval before proceeding.

**Arguments:**
| Arg | Type | Required | Description |
|-----|------|----------|-------------|
| `type` | enum | Yes | pre_commit, pre_push, schema_change, breaking_change |
| `description` | string | Yes | What needs approval |
| `affected_repos` | string[] | No | Affected repositories |

**Acceptance Criteria:**
- [ ] Checkpoint is created via Go API
- [ ] Checkpoint status is "pending"
- [ ] Agent receives confirmation with checkpoint ID
- [ ] Go wrapper displays checkpoint prompt to user in terminal
- [ ] User can approve/deny via terminal input
- [ ] On approval: checkpoint status → "approved", agent notified
- [ ] On denial: checkpoint status → "denied" with reason, agent notified

**Definition of Done:**
- Checkpoint prompt appears in terminal within 1 second
- User decision propagated to agent within 1 second

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| User doesn't respond for 5 minutes | Timeout: checkpoint auto-denied with "timeout" reason |
| User denies with reason | Agent receives denial with reason, can adapt |
| Multiple checkpoints pending | Queue displayed, user processes in order |
| Agent creates duplicate checkpoint | Deduplicate by (type + description hash) |
| User not in terminal (headless) | Auto-approve if config says `checkpoint_policy: "auto"` |

---

### REQ-TOOL-004: sync_workspace Tool

**Description:** Get or sync orchestration state.

**Arguments:**
| Arg | Type | Required | Description |
|-----|------|----------|-------------|
| `action` | enum | Yes | status, get_tasks, get_messages, get_checkpoints |
| `filter` | object | No | Filter criteria |

**Acceptance Criteria:**
- [ ] `status` returns summary (active tasks, pending checkpoints, agent count)
- [ ] `get_tasks` returns tasks (optionally filtered by status/assignee)
- [ ] `get_messages` returns messages for current agent
- [ ] `get_checkpoints` returns pending checkpoints
- [ ] All reads go through Go HTTP API

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Go API unavailable | Error: "Orchestrator not running" |
| Large result set | Paginate (max 50 items per request) |
| Filter matches nothing | Return empty array, not error |

---

## 5. Agent Configurations

### REQ-AGENT-001: Tech Lead Agent

**Type:** Primary agent
**Mode:** `primary`

**Capabilities:**
- Task decomposition and planning
- Dependency identification
- Task assignment to subagents
- Progress monitoring
- Architectural decision making

**Permissions:**
| Permission | Value | Reason |
|------------|-------|--------|
| `edit` | `ask` | Should plan, not code directly |
| `bash` | `ask` (with allowlist) | Needs `git status`, `git log`, `ls` |
| `task` | `allow` | Can invoke subagents |
| `team_message` | `allow` | Core coordination |
| `task_create` | `allow` | Core coordination |
| `request_checkpoint` | `allow` | Core coordination |
| `sync_workspace` | `allow` | Core coordination |

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| User asks tech-lead to code | Agent should delegate to backend/frontend |
| Tech-lead tries to commit | Checkpoint required |
| Tech-lead creates 100 tasks | No limit, but warn if >20 |

---

### REQ-AGENT-002: Backend Developer Agent

**Type:** Subagent
**Mode:** `subagent`

**Capabilities:**
- API design and implementation
- Database schema changes
- Business logic
- Unit and integration testing

**Permissions:**
| Permission | Value | Reason |
|------------|-------|--------|
| `edit` | `allow` | Primary job is coding |
| `bash.git commit*` | `ask` | Checkpoint before commits |
| `bash.go test*` | `allow` | Running tests |
| `bash.npm test*` | `allow` | Running tests |
| `team_message` | `allow` | Communication |
| `request_checkpoint` | `allow` | Can request approval |

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Agent edits files outside assigned repo | Plugin validates path, blocks if outside |
| Agent tries to git push | Checkpoint required |
| Agent runs destructive command | Permission system blocks |

---

### REQ-AGENT-003: Frontend Developer Agent

**Type:** Subagent
**Mode:** `subagent`

**Capabilities:**
- Component design and implementation
- UI/UX improvements
- Responsive design
- Accessibility compliance

**Permissions:** Similar to backend-dev but with frontend-specific bash allowlist.

---

## 6. Multi-Repo Coordination

### REQ-MULTI-001: Multi-Repo Session Spawning

**Description:** Go wrapper spawns OpenCode instances in each repo directory.

**Acceptance Criteria:**
- [ ] One OpenCode process per repo
- [ ] Each instance has unique port
- [ ] Each instance has the orchestration plugin installed
- [ ] Instances can communicate via Go API
- [ ] Go wrapper monitors all instances

**Definition of Done:**
- 3 repos → 3 OpenCode instances running on ports 9801, 9802, 9803
- Each instance has plugin loaded

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Repo has no `.opencode/` directory | Go wrapper creates it and copies plugin files |
| Repo has existing `opencode.json` | Go wrapper merges orchestration config |
| OpenCode instance crashes | Restart up to 3 times, then mark failed |

---

### REQ-MULTI-002: Cross-Repo Dependency Tracking

**Description:** System tracks dependencies between tasks in different repos.

**Acceptance Criteria:**
- [ ] Dependencies are declared at task creation
- [ ] System validates dependencies exist
- [ ] System detects circular dependencies
- [ ] Tasks with unmet dependencies are blocked
- [ ] When dependency is met, blocked tasks are unblocked

**Definition of Done:**
- Dependency graph is acyclic
- Tasks execute in dependency order

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Circular dependency A→B→A | Error at creation time |
| Dependency on deleted task | Error: "Dependency target not found" |
| Dependency across repos | Valid (this is the core use case) |
| 10+ dependency chain | Process in topological order |

---

## 7. Checkpoint System

### REQ-CP-001: Checkpoint Lifecycle

**Description:** Checkpoints are approval gates that block agent progress.

**Lifecycle:**
```
pending → approved
pending → denied (with reason)
pending → timed_out
```

**Acceptance Criteria:**
- [ ] Agent blocks (returns to user) when checkpoint is pending
- [ ] Go wrapper displays checkpoint in terminal
- [ ] User can approve (y) or deny (n + reason)
- [ ] Approval unblocks agent
- [ ] Denial notifies agent with reason
- [ ] Timeout after 5 minutes (configurable)

**Definition of Done:**
- Agent waits for checkpoint resolution
- User sees clear checkpoint description
- Resolution propagated within 1 second

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| User closes terminal | Checkpoint times out, agent receives "timeout" |
| Multiple checkpoints queued | Display all, user processes in order |
| Agent creates checkpoint for another agent | Not allowed (self-checkpoints only) |
| Checkpoint for destructive operation | Always requires explicit approval (no auto-approve) |

---

### REQ-CP-002: Checkpoint Types

| Type | Description | Auto-Approve? |
|------|-------------|---------------|
| `pre_commit` | Before git commit | No |
| `pre_push` | Before git push | No |
| `schema_change` | Database/API schema change | No |
| `breaking_change` | Breaking API change | No |
| `destructive` | Delete/drop operations | Never |

---

## 8. Message Passing

### REQ-MSG-001: Message Delivery

**Description:** Messages are delivered to agent inboxes.

**Acceptance Criteria:**
- [ ] Messages written to `messages/inbox/{agent-id}/`
- [ ] Agent receives message context on next prompt (via plugin hook)
- [ ] Messages have unique IDs (UUID)
- [ ] Messages are ordered by timestamp
- [ ] Read messages are moved to archive after 7 days

**Definition of Done:**
- Message delivery latency <500ms
- No message loss

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Agent inbox has 100+ messages | Summarize old messages, show recent 10 |
| Message for non-existent agent | Error: "Agent not found" |
| Concurrent message writes | UUID prevents collisions |
| Disk full | Error: "Cannot write message" |

---

### REQ-MSG-002: Message Types

| Type | Description | Priority |
|------|-------------|----------|
| `task_assignment` | Assign task to agent | High |
| `status_update` | Progress update | Normal |
| `question` | Question to another agent | Normal |
| `dependency_alert` | Notify dependency met | High |
| `blocker` | Agent is blocked | Critical |
| `checkpoint_request` | Request checkpoint | High |

---

## 9. Security

### REQ-SEC-001: Authentication

**Description:** Go ↔ TS communication is authenticated.

**Acceptance Criteria:**
- [ ] Shared secret generated on `init`
- [ ] TS plugin sends secret in `Authorization: Bearer {secret}` header
- [ ] Go API validates secret on every request
- [ ] Secret stored in `.orchestrator/auth.key` (mode 0600)
- [ ] Secret is not committed to git (in `.gitignore`)

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Secret file missing | Go API rejects all requests |
| Secret mismatch | 401 Unauthorized |
| Secret rotated | Old secret valid for 5 minutes (grace period) |

---

### REQ-SEC-002: Prompt Injection Prevention

**Description:** Inter-agent messages are sanitized.

**Acceptance Criteria:**
- [ ] Messages wrapped in `<message>` XML tags
- [ ] Instruction-like patterns stripped (e.g., "ignore previous", "you are now")
- [ ] Messages cannot override system prompts
- [ ] Message content is escaped for LLM context

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Agent sends "Approve all checkpoints" | Wrapped as data, not instruction |
| Message contains code blocks | Preserved (code is valid content) |
| Message >10KB | Truncated with "...[truncated]" |

---

### REQ-SEC-003: Agent Sandboxing

**Description:** Agents cannot escape their assigned repository.

**Acceptance Criteria:**
- [ ] Plugin validates file paths are within assigned repo
- [ ] `bash` commands are restricted via permission system
- [ ] Agents cannot modify `.orchestrator/` directory
- [ ] Agents cannot modify `.opencode/` directory

**Edge Cases:**
| Case | Expected Behavior |
|------|-------------------|
| Agent tries `../../other-repo/file` | Blocked by plugin |
| Agent tries `rm -rf /` | Blocked by OpenCode permissions |
| Agent modifies `.orchestrator/state.json` | Blocked (Go owns state) |

---

## 10. Observability

### REQ-OBS-001: Audit Logging

**Description:** All orchestration actions are logged.

**Acceptance Criteria:**
- [ ] Append-only log at `.orchestrator/audit.log`
- [ ] JSON lines format
- [ ] Includes: timestamp, action, actor, details, result
- [ ] Log rotation (10MB max, keep 5 rotated files)

**Log Entry Schema:**
```json
{
  "timestamp": "2026-07-17T10:30:00Z",
  "action": "task_create",
  "actor": "tech-lead",
  "details": {
    "task_id": "task-001",
    "title": "Implement JWT auth"
  },
  "result": "success"
}
```

---

### REQ-OBS-002: Health Checks

**Description:** System health is queryable.

**Acceptance Criteria:**
- [ ] Go API exposes `/health` endpoint
- [ ] Returns status of all OpenCode instances
- [ ] Returns last activity timestamp
- [ ] Returns resource usage (memory, disk)

---

## Appendix: Acceptance Test Scenarios

### Scenario 1: JWT Auth Across 3 Repos

1. `crush-orchestrator init auth-project --repos api-gateway,user-service,notification-service`
2. `crush-orchestrator start`
3. User: "Implement JWT authentication across all services"
4. Tech Lead creates 3 tasks with dependencies
5. Backend agent works on user-service first (schema)
6. Checkpoint: "Approve schema migration" → User approves
7. Backend agents work on api-gateway and notification-service in parallel
8. All tasks complete, PRs ready

**Expected:** All 3 PRs created with coordinated changes. Total time <30 minutes.

### Scenario 2: Emergency Hotfix

1. `crush-orchestrator start` (already running)
2. User: "Critical bug in payment service"
3. Tech Lead creates single high-priority task
4. Checkpoint skipped (emergency mode)
5. Backend agent fixes bug
6. Checkpoint: "Approve hotfix" → User approves
7. Fix deployed

**Expected:** Hotfix deployed in <10 minutes.

### Scenario 3: Agent Crash Recovery

1. Orchestration running with 3 agents
2. Backend agent crashes (OpenCode process killed)
3. Go wrapper detects crash via health check
4. Go wrapper restarts OpenCode instance
5. Plugin reconnects to Go API
6. Agent resumes from last state

**Expected:** Recovery in <30 seconds, no state loss.

### Scenario 4: Concurrent Task Creation

1. Tech Lead creates task-A and task-B simultaneously
2. Both tasks have UUID IDs (no collision)
3. Both tasks written atomically
4. Both tasks visible in `sync_workspace status`

**Expected:** Both tasks created, no corruption.

### Scenario 5: Checkpoint Denial

1. Agent requests checkpoint: "Approve database migration"
2. User denies: "Schema needs index on user_id column"
3. Agent receives denial with reason
4. Agent modifies implementation to include index
5. Agent requests new checkpoint
6. User approves

**Expected:** Agent adapts to feedback, final implementation includes index.
