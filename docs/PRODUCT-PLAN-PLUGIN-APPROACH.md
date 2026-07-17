# Product Plan: Agent Team Orchestration — OpenCode Plugin Approach

**Document Version:** 1.0
**Last Updated:** 2026-07-17
**Status:** Draft
**Owner:** Product Engineering Team
**Approach:** OpenCode Plugin + Custom Agents (v2)

---

## 1. Executive Summary

### Overview
The **Agent Team Orchestration** system is an OpenCode plugin that enables multiple coordinated AI agents to collaborate on multi-repository projects. Instead of building a standalone orchestration binary, we leverage OpenCode's native extension system — plugins, custom tools, agent configs, and the SDK — to add orchestration capabilities directly into the workflow developers already use.

### Why Plugin Instead of Standalone

| Dimension | Standalone (v1) | Plugin (v2) |
|-----------|----------------|-------------|
| Time to MVP | 8-12 weeks | 3-5 weeks |
| Lines of code | ~5000+ Go | ~1000-1500 TS |
| AI infrastructure | Build from scratch | Reuse OpenCode |
| TUI/Dashboard | Build from scratch | Already exists |
| Agent runtime | Custom | OpenCode handles it |
| LLM integration | Custom | OpenCode handles it |
| Git operations | Custom | OpenCode handles it |
| Installation | Separate binary | `npm install` plugin |
| Maintenance | High (own stack) | Low (leverage OpenCode) |

### The Problem (Unchanged)
Current AI coding agents operate in isolation:
- Changes in one repository break dependencies in others
- No coordination between agents working on interconnected services
- Context fragmentation across project boundaries
- Human overhead required to synchronize cross-repo work

### The Solution
An OpenCode plugin that adds orchestration capabilities:

- **Orchestration Plugin** — JS/TS module that hooks into OpenCode events
- **Custom Tools** — `team_message`, `request_checkpoint`, `sync_workspace`, `task_create`
- **Specialized Agents** — Markdown configs for Tech Lead, Backend, Frontend roles
- **Shared Workspace** — File-based state in `.orchestrator/` directory
- **SDK Integration** — Programmatic control of multiple OpenCode sessions

### Key Value Propositions
| Value Proposition | Description |
|------------------|-------------|
| **Zero Friction** | Works inside OpenCode, no new tools to learn |
| **Leverage** | Reuses OpenCode's LLM, git, file, and bash capabilities |
| **Fast MVP** | 3-5 weeks with 2-3 developers |
| **Portable** | Plugin works on any OpenCode installation |
| **Composable** | Agents are just config files, easy to customize |

---

## 2. Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     OPENCODE INSTANCE                            │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              ORCHESTRATION PLUGIN                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │ Task         │  │ Checkpoint   │  │ State        │   │   │
│  │  │ Manager      │  │ Coordinator  │  │ Manager      │   │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │   │
│  │  ┌──────────────┐  ┌──────────────┐                     │   │
│  │  │ Dependency   │  │ Message      │                     │   │
│  │  │ Tracker      │  │ Router       │                     │   │
│  │  └──────────────┘  └──────────────┘                     │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    CUSTOM TOOLS                           │   │
│  │  team_message  │  request_checkpoint  │  sync_workspace  │   │
│  │  task_create   │  task_assign         │  dependency_add  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │               SPECIALIZED AGENTS (Config)                │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐           │   │
│  │  │ tech-lead  │ │ backend-dev│ │ frontend-dev│           │   │
│  │  │ (primary)  │ │ (subagent) │ │ (subagent)  │           │   │
│  │  └────────────┘ └────────────┘ └────────────┘           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 SHARED WORKSPACE (Files)                  │   │
│  │  .orchestrator/                                          │   │
│  │  ├── state.json         # Master orchestration state     │   │
│  │  ├── tasks/             # Task definitions               │   │
│  │  ├── messages/          # Inter-agent messages           │   │
│  │  ├── checkpoints/       # Pending approvals              │   │
│  │  └── context/           # Cross-repo shared context      │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ SDK (optional)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              OTHER OPENCODE INSTANCES (Other Repos)              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ OpenCode     │  │ OpenCode     │  │ OpenCode     │          │
│  │ (Repo A)     │  │ (Repo B)     │  │ (Repo C)     │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

### How It Works

1. **User** runs `opencode` in their project with the orchestration plugin installed
2. **Plugin** initializes, creates `.orchestrator/` directory, loads agent configs
3. **Tech Lead Agent** (primary) decomposes tasks and creates execution plan
4. **Backend/Frontend Agents** (subagents) are invoked via `@backend-dev` mentions or Task tool
5. **Custom Tools** enable agents to communicate, create checkpoints, and sync state
6. **Shared Workspace** (`.orchestrator/`) persists state across sessions and repos
7. **Multi-Repo** coordination via OpenCode SDK — spawn sessions in other repo directories

---

## 3. Core Features

### P0 Features (MVP — Must Have)

| Feature | Description | Implementation | Effort |
|---------|-------------|----------------|--------|
| **Orchestration Plugin** | JS/TS plugin hooking into OpenCode events | `.opencode/plugins/orchestration.ts` | 3 days |
| **Custom Tools** | team_message, task_create, request_checkpoint, sync_workspace | `.opencode/tools/` | 3 days |
| **Agent Configs** | tech-lead.md, backend-dev.md, frontend-dev.md | `.opencode/agents/` | 1 day |
| **Task Manager** | Create, assign, track tasks with dependencies | Plugin state module | 3 days |
| **Checkpoint System** | Human approval gates for critical operations | Plugin + custom tool | 2 days |
| **File-Based State** | JSON persistence in `.orchestrator/` | Plugin state module | 1 day |
| **Message Passing** | Agents communicate via shared files | Custom tool + file watcher | 2 days |

**MVP Total: ~2.5 weeks (1 developer), ~1.5 weeks (2 developers)**

### P1 Features (v1 — Important)

| Feature | Description | Effort |
|---------|-------------|--------|
| **Dependency Graph** | Auto-detect cross-repo dependencies | 3 days |
| **Multi-Repo Sessions** | Spawn OpenCode in other repo dirs via SDK | 3 days |
| **Conflict Detection** | Identify potential merge conflicts early | 2 days |
| **Progress Tracking** | Task completion metrics and reporting | 2 days |
| **Agent Lifecycle** | Start, pause, resume agents | 2 days |
| **Template System** | YAML-based custom agent templates | 2 days |

### P2 Features (v2 — Nice to Have)

| Feature | Description | Effort |
|---------|-------------|--------|
| **TUI Dashboard** | Terminal UI for orchestration status | 5 days |
| **Rollback Coordination** | Synchronized rollback across repos | 3 days |
| **Learning System** | Improve agent behavior from past runs | 5 days |
| **Cost Tracking** | Token usage monitoring | 2 days |

---

## 4. Implementation Details

### 4.1 Plugin Structure

```
.opencode/
├── plugins/
│   └── orchestration.ts          # Main plugin
├── tools/
│   ├── team_message.ts           # Send messages between agents
│   ├── task_create.ts            # Create and manage tasks
│   ├── task_assign.ts            # Assign tasks to agents
│   ├── request_checkpoint.ts     # Request human approval
│   ├── sync_workspace.ts         # Sync state across repos
│   └── dependency_add.ts         # Track cross-repo dependencies
├── agents/
│   ├── tech-lead.md              # Tech Lead primary agent
│   ├── backend-dev.md            # Backend Developer subagent
│   └── frontend-dev.md           # Frontend Developer subagent
└── .orchestrator/                # Runtime state (gitignored)
    ├── state.json
    ├── tasks/
    ├── messages/
    ├── checkpoints/
    └── context/
```

### 4.2 Plugin Implementation

```typescript
// .opencode/plugins/orchestration.ts
import type { Plugin } from "@opencode-ai/plugin"

export const OrchestrationPlugin: Plugin = async ({ project, client, $, directory }) => {
  const stateManager = new StateManager(directory)

  return {
    // Initialize orchestration on session start
    "session.created": async (input) => {
      await stateManager.init()
      await client.app.log({
        body: {
          service: "orchestration",
          level: "info",
          message: `Orchestration initialized for ${project.name}`,
        },
      })
    },

    // Track tool usage for audit log
    "tool.execute.after": async (input, output) => {
      if (input.tool === "team_message" || input.tool === "task_create") {
        await stateManager.logAction(input.tool, input.args, output.result)
      }
    },

    // Handle checkpoint responses
    "permission.replied": async (input) => {
      if (input.tool === "request_checkpoint") {
        await stateManager.resolveCheckpoint(input.args.checkpointId, input.allowed)
      }
    },

    // Inject orchestration context on compaction
    "experimental.session.compacting": async (input, output) => {
      const state = await stateManager.getState()
      output.context.push(`## Orchestration State
Active tasks: ${state.tasks.filter(t => t.status === "in_progress").length}
Pending checkpoints: ${state.checkpoints.filter(c => c.status === "pending").length}
Recent messages: ${state.messages.slice(-5).map(m => `- ${m.from} → ${m.to}: ${m.type}`).join("\n")}`)
    },

    // Custom tools
    tool: {
      team_message: tool({
        description: "Send a message to another agent in the team",
        args: {
          to: tool.schema.string().describe("Recipient agent ID or 'all' for broadcast"),
          type: tool.schema.enum(["status_update", "question", "dependency_alert", "checkpoint_request"]).describe("Message type"),
          content: tool.schema.string().describe("Message content"),
        },
        async execute(args, context) {
          const message = {
            id: crypto.randomUUID(),
            from: context.agent,
            to: args.to,
            type: args.type,
            content: args.content,
            timestamp: new Date().toISOString(),
          }
          await stateManager.addMessage(message)
          return `Message sent to ${args.to}: ${args.content}`
        },
      }),

      task_create: tool({
        description: "Create a new task for the team",
        args: {
          title: tool.schema.string().describe("Task title"),
          description: tool.schema.string().describe("Task description"),
          assignee: tool.schema.string().optional().describe("Agent ID to assign"),
          dependencies: tool.schema.array(tool.schema.string()).optional().describe("Task IDs this depends on"),
          repo: tool.schema.string().optional().describe("Repository path"),
        },
        async execute(args) {
          const task = {
            id: `task-${Date.now()}`,
            title: args.title,
            description: args.description,
            status: "pending",
            assignee: args.assignee,
            dependencies: args.dependencies || [],
            repo: args.repo,
            created_at: new Date().toISOString(),
          }
          await stateManager.addTask(task)
          return `Task created: ${task.id} — ${task.title}`
        },
      }),

      request_checkpoint: tool({
        description: "Request human approval before proceeding",
        args: {
          type: tool.schema.enum(["pre_commit", "pre_push", "schema_change", "breaking_change", "deploy"]).describe("Checkpoint type"),
          description: tool.schema.string().describe("What needs approval"),
          affected_repos: tool.schema.array(tool.schema.string()).optional().describe("Affected repositories"),
        },
        async execute(args) {
          const checkpoint = {
            id: `cp-${Date.now()}`,
            type: args.type,
            description: args.description,
            affected_repos: args.affected_repos || [],
            status: "pending",
            created_at: new Date().toISOString(),
          }
          await stateManager.addCheckpoint(checkpoint)
          // Permission hook will handle the actual approval flow
          return `Checkpoint requested: ${checkpoint.id} — ${checkpoint.description}`
        },
      }),

      sync_workspace: tool({
        description: "Sync orchestration state with other repositories",
        args: {
          action: tool.schema.enum(["push", "pull", "status"]).describe("Sync action"),
          repo: tool.schema.string().optional().describe("Specific repo to sync"),
        },
        async execute(args) {
          const state = await stateManager.getState()
          if (args.action === "status") {
            return JSON.stringify(state, null, 2)
          }
          // Sync logic via file system
          return `Workspace ${args.action} completed`
        },
      }),
    },
  }
}
```

### 4.3 Agent Configurations

#### Tech Lead Agent (Primary)

```markdown
<!-- .opencode/agents/tech-lead.md -->
---
description: Coordinates multi-repo development, decomposes tasks, manages dependencies
mode: primary
model: anthropic/claude-sonnet-4-20250514
permission:
  edit: ask
  bash:
    "*": ask
    "git status*": allow
    "git log*": allow
    "ls*": allow
    "cat*": allow
---

You are a Tech Lead coordinating a multi-repository development effort.

## Your Responsibilities
1. Analyze the project scope and break it into tasks
2. Identify dependencies between repositories
3. Assign tasks to specialized agents (backend-dev, frontend-dev)
4. Monitor progress and resolve blockers
5. Ensure architectural consistency across repos

## How to Use Tools
- Use `task_create` to create tasks for the team
- Use `team_message` to communicate with other agents
- Use `request_checkpoint` before any destructive operations
- Use `sync_workspace` to check cross-repo status

## Coordination Flow
1. Analyze the request
2. Create tasks with `task_create`
3. Identify dependencies between tasks
4. Assign to appropriate agents via `@backend-dev` or `@frontend-dev`
5. Monitor progress via `sync_workspace`

## Rules
- Never commit without a checkpoint
- Always check dependencies before starting work
- Communicate blockers immediately via `team_message`
- Document architectural decisions in shared context
```

#### Backend Developer Agent (Subagent)

```markdown
<!-- .opencode/agents/backend-dev.md -->
---
description: Specialized backend development across microservices
mode: subagent
model: anthropic/claude-sonnet-4-20250514
permission:
  edit: allow
  bash:
    "*": ask
    "git add*": allow
    "git commit*": ask
    "go test*": allow
    "npm test*": allow
    "pytest*": allow
---

You are a Backend Developer working on microservices.

## Your Capabilities
- API design and implementation
- Database schema changes
- Business logic implementation
- Unit and integration testing

## When Working
1. Check for task assignment via `sync_workspace`
2. Understand the full context before making changes
3. Consider impact on other services
4. Write tests for all changes
5. Use `request_checkpoint` before committing
6. Report progress via `team_message`

## Rules
- Always check API contracts before changing interfaces
- Coordinate database changes with other services
- Never break backward compatibility without explicit approval
- Include test coverage for new functionality
```

#### Frontend Developer Agent (Subagent)

```markdown
<!-- .opencode/agents/frontend-dev.md -->
---
description: Specialized frontend/UI development
mode: subagent
model: anthropic/claude-sonnet-4-20250514
permission:
  edit: allow
  bash:
    "*": ask
    "git add*": allow
    "git commit*": ask
    "npm test*": allow
    "npm run build*": allow
---

You are a Frontend Developer working on UI components.

## Your Capabilities
- Component design and implementation
- UI/UX improvements
- Responsive design
- Accessibility compliance

## Rules
- Follow existing design system patterns
- Test across viewport sizes
- Ensure accessibility (WCAG 2.1 AA)
- Coordinate API contracts with backend agents
```

### 4.4 Multi-Repo Coordination

For multi-repo orchestration, the plugin uses OpenCode's SDK to spawn sessions in other repositories:

```typescript
// Multi-repo session management
import { createOpencodeClient } from "@opencode-ai/sdk"

class MultiRepoManager {
  private clients: Map<string, ReturnType<typeof createOpencodeClient>> = new Map()

  async connectToRepo(repoPath: string, port: number) {
    const client = createOpencodeClient({
      baseUrl: `http://localhost:${port}`,
    })
    this.clients.set(repoPath, client)
    return client
  }

  async spawnAgentInRepo(repoPath: string, agentType: string, task: string) {
    const client = this.clients.get(repoPath)
    if (!client) throw new Error(`Not connected to ${repoPath}`)

    // Create session in the target repo
    const session = await client.session.create({
      body: { title: `${agentType}: ${task}` },
    })

    // Send the task to the agent
    await client.session.prompt({
      path: { id: session.id },
      body: {
        parts: [{ type: "text", text: task }],
      },
    })

    return session
  }
}
```

### 4.5 State Schema

```json
{
  "version": "1.0.0",
  "project": "auth-rewrite",
  "created_at": "2026-07-17T10:00:00Z",
  "repos": [
    {
      "name": "api-gateway",
      "path": "/path/to/api-gateway",
      "agent_session": "session-abc123"
    },
    {
      "name": "user-service",
      "path": "/path/to/user-service",
      "agent_session": "session-def456"
    }
  ],
  "tasks": [
    {
      "id": "task-001",
      "title": "Implement JWT authentication",
      "description": "Add JWT auth to API gateway",
      "status": "in_progress",
      "assignee": "backend-dev-001",
      "repo": "api-gateway",
      "dependencies": [],
      "checkpoints": ["cp-001"],
      "created_at": "2026-07-17T10:30:00Z"
    }
  ],
  "messages": [
    {
      "id": "msg-001",
      "from": "tech-lead",
      "to": "backend-dev-001",
      "type": "task_assignment",
      "content": "Implement JWT auth in api-gateway",
      "timestamp": "2026-07-17T10:35:00Z"
    }
  ],
  "checkpoints": [
    {
      "id": "cp-001",
      "type": "pre_commit",
      "description": "Approve JWT implementation in api-gateway",
      "status": "pending",
      "affected_repos": ["api-gateway"],
      "created_at": "2026-07-17T11:00:00Z"
    }
  ],
  "dependencies": [
    {
      "from_repo": "api-gateway",
      "to_repo": "user-service",
      "type": "api_contract",
      "description": "Auth endpoints depend on user-service"
    }
  ]
}
```

---

## 5. User Workflow

### Setup (One-time)

```bash
# 1. Install the plugin
cd your-project
mkdir -p .opencode/plugins .opencode/tools .opencode/agents

# 2. Copy plugin files (or npm install)
cp -r orchestration-plugin/.opencode/* .opencode/

# 3. Configure in opencode.json
cat > opencode.json << 'EOF'
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["./.opencode/plugins/orchestration.ts"],
  "agent": {
    "tech-lead": {
      "mode": "primary",
      "prompt": "{file:./.opencode/agents/tech-lead.md}"
    },
    "backend-dev": {
      "mode": "subagent",
      "prompt": "{file:./.opencode/agents/backend-dev.md}"
    },
    "frontend-dev": {
      "mode": "subagent",
      "prompt": "{file:./.opencode/agents/frontend-dev.md}"
    }
  }
}
EOF
```

### Usage Flow

```
User: Implement JWT authentication across api-gateway, user-service, and notification-service

[Tech Lead Agent analyzes request]

Tech Lead: I'll break this into tasks:

1. task_create: "Update user-service schema for JWT"
   - Assign: backend-dev-001
   - Repo: user-service
   - Dependencies: none

2. task_create: "Implement JWT middleware in api-gateway"
   - Assign: backend-dev-002
   - Repo: api-gateway
   - Dependencies: task-001 (schema must exist first)

3. task_create: "Add JWT token refresh to notification-service"
   - Assign: backend-dev-003
   - Repo: notification-service
   - Dependencies: task-001

[Agent switches to backend-dev via @backend-dev or Tab]

Backend Dev (user-service): Working on JWT schema...
- Creating migration files
- Updating user model
- Writing tests

Backend Dev: request_checkpoint: "Approve schema migration"
[User approves]

Backend Dev: task-001 complete. Schema updated.

[Other agents can now start their dependent tasks]

Backend Dev (api-gateway): Starting JWT middleware...
- Depends on user-service schema (task-001 ✓)
- Implementing auth middleware
- Writing tests

Backend Dev: request_checkpoint: "Approve JWT middleware"
[User approves]
```

---

## 6. File Structure

```
agent-team-orchestration-plugin/
├── .opencode/
│   ├── plugins/
│   │   └── orchestration.ts        # Main plugin entry point
│   ├── tools/
│   │   ├── team_message.ts         # Agent-to-agent messaging
│   │   ├── task_create.ts          # Task creation and management
│   │   ├── task_assign.ts          # Task assignment
│   │   ├── request_checkpoint.ts   # Human approval gates
│   │   ├── sync_workspace.ts       # State synchronization
│   │   └── dependency_add.ts       # Cross-repo dependency tracking
│   └── agents/
│       ├── tech-lead.md            # Tech Lead agent config
│       ├── backend-dev.md          # Backend Developer agent config
│       └── frontend-dev.md         # Frontend Developer agent config
├── src/
│   ├── state/
│   │   ├── manager.ts              # State management
│   │   └── types.ts                # TypeScript types
│   ├── messaging/
│   │   └── router.ts               # Message routing
│   └── coordination/
│       ├── task-tracker.ts         # Task lifecycle
│       └── dependency-graph.ts     # Dependency management
├── package.json
├── tsconfig.json
├── opencode.json                   # Example config
└── README.md
```

---

## 7. Technology Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Plugin | TypeScript (Bun runtime) | OpenCode's native plugin system |
| Tools | TypeScript + Zod | OpenCode's custom tool API |
| Agents | Markdown configs | OpenCode's agent system |
| State | JSON files | Simple, debuggable, portable |
| SDK | @opencode-ai/sdk | Multi-repo session control |
| Messaging | File-based (.orchestrator/messages/) | Simple, no external deps |
| Dependencies | Graph algorithms in TS | No external graph library needed |

### Dependencies

```json
{
  "dependencies": {
    "@opencode-ai/plugin": "latest",
    "@opencode-ai/sdk": "latest"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "zod": "^3.23.0"
  }
}
```

---

## 8. Success Metrics

### Primary KPIs

| Metric | Target | Measurement | Timeline |
|--------|--------|-------------|----------|
| **Time to MVP** | <5 weeks | Weeks from start to working demo | Week 5 |
| **Setup Time** | <5 minutes | Time from clone to first orchestrated task | Week 3 |
| **Task Coordination** | >80% success | Tasks completed without manual intervention | Week 6 |
| **Checkpoint Approval** | >90% | Agent-proposed actions approved as-is | Week 6 |
| **Multi-Repo Support** | 3 repos | Successfully coordinate across 3 repositories | Week 5 |

### Secondary KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Plugin Size** | <50KB | Bundle size |
| **State File Size** | <1MB per project | File size after 100 tasks |
| **Message Latency** | <500ms | File write to read |
| **Agent Spawn Time** | <2 seconds | Session creation to ready |

---

## 9. Development Roadmap

### Week 1: Foundation
- [ ] Set up plugin skeleton with OpenCode plugin API
- [ ] Implement basic state manager (create, read, update)
- [ ] Create team_message custom tool
- [ ] Create task_create custom tool
- [ ] Test plugin loads in OpenCode

### Week 2: Core Tools + Agents
- [ ] Implement request_checkpoint tool with permission flow
- [ ] Implement sync_workspace tool
- [ ] Create tech-lead.md agent config
- [ ] Create backend-dev.md agent config
- [ ] Create frontend-dev.md agent config
- [ ] Test agent switching and tool invocation

### Week 3: Coordination Logic
- [ ] Implement dependency tracking between tasks
- [ ] Add message routing between agents
- [ ] Implement checkpoint approval workflow
- [ ] Add state persistence and recovery
- [ ] End-to-end test: 2 tasks with dependencies

### Week 4: Multi-Repo + Polish
- [ ] Implement multi-repo session management via SDK
- [ ] Add cross-repo dependency detection
- [ ] Create example project with 3 repos
- [ ] Write documentation and README
- [ ] Create installation script

### Week 5: Testing + Release
- [ ] Integration testing with real microservice project
- [ ] Performance testing with 5+ concurrent tasks
- [ ] Bug fixes and edge cases
- [ ] Publish to npm / GitHub
- [ ] Write blog post / demo

---

## 10. Risk Analysis

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **OpenCode Plugin API Changes** | Low | Medium | Pin SDK version, follow changelog |
| **Multi-Repo Session Complexity** | Medium | High | Start with 2 repos, expand gradually |
| **State File Conflicts** | Low | Low | Use timestamps, atomic writes |
| **Agent Coordination Failures** | Medium | Medium | Checkpoint system catches issues |
| **Performance at Scale** | Low | Medium | File-based state works for MVP |
| **LLM Context Overflow** | Medium | Medium | Task decomposition keeps context focused |

### Contingency Plans

**If plugin API doesn't support needed hooks:**
- Fall back to custom tools only (reduced functionality)
- Use file watchers as alternative to event hooks

**If multi-repo sessions too complex:**
- Start with single-repo, add multi-repo in v1.1
- Use manual repo switching as interim solution

---

## 11. Comparison: Standalone vs Plugin

### What We Lose (Standalone → Plugin)

| Feature | Impact | Alternative |
|---------|--------|-------------|
| Custom TUI dashboard | Medium | OpenCode's existing UI + status commands |
| Custom CLI binary | Low | Plugin installed via config |
| Go performance | Low | TypeScript is sufficient for orchestration |
| Custom agent runtime | None | Reuse OpenCode's runtime |

### What We Gain (Standalone → Plugin)

| Benefit | Impact | Value |
|---------|--------|-------|
| 3-5x faster to MVP | High | Ship in weeks, not months |
| Leverage OpenCode ecosystem | High | Plugins, tools, agents, SDK |
| Zero learning curve | High | Developers already know OpenCode |
| Automatic updates | Medium | OpenCode improvements = our improvements |
| Community adoption | Medium | Plugin format is familiar |
| Less maintenance | High | No custom AI infrastructure |

### Recommendation

**Plugin approach is strongly recommended.** The standalone approach builds infrastructure that OpenCode already provides. By building as a plugin, we:
1. Ship faster (3-5 weeks vs 8-12 weeks)
2. Maintain less code (~1500 lines TS vs ~5000 lines Go)
3. Get better user experience (integrated vs separate tool)
4. Benefit from OpenCode's ongoing improvements
5. Can still go standalone later if needed

---

## 12. Appendix

### A. OpenCode Extension Points Used

| Extension Point | How We Use It |
|-----------------|---------------|
| **Plugins** | Main orchestration logic, event hooks |
| **Custom Tools** | Agent communication, task management |
| **Agents** | Specialized roles (tech-lead, backend, frontend) |
| **SDK** | Multi-repo session control |
| **Permissions** | Checkpoint approval flow |
| **File System** | Shared state in .orchestrator/ |

### B. References

- [OpenCode Plugin Docs](https://opencode.ai/docs/plugins/)
- [OpenCode Custom Tools](https://opencode.ai/docs/custom-tools/)
- [OpenCode Agents](https://opencode.ai/docs/agents/)
- [OpenCode SDK](https://opencode.ai/docs/sdk/)
- [OpenCode MCP Servers](https://opencode.ai/docs/mcp-servers/)

### C. Example Plugin Events

| Event | When It Fires | How We Use It |
|-------|---------------|---------------|
| `session.created` | New session starts | Initialize orchestration state |
| `session.idle` | Agent completes work | Check for dependent tasks |
| `tool.execute.after` | Tool finishes | Log actions for audit |
| `permission.replied` | User approves/denies | Resolve checkpoints |
| `message.updated` | New message | Route to other agents |
| `experimental.session.compacting` | Context compression | Inject orchestration state |
