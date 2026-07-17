# Product Plan: Agent Team Orchestration for OpenCode

**Document Version:** 1.0  
**Last Updated:** 2026-04-01  
**Status:** Draft  
**Owner:** Product Engineering Team  

---

## 1. Executive Summary

### Overview
The **Agent Team Orchestration** system is a distributed coordination layer that enables multiple OpenCode (now Crush) instances to collaborate on complex, multi-repository projects. While OpenCode excels at single-repository coding tasks, modern software development often spans multiple microservices, APIs, and shared libraries that require coordinated changes across repositories.

### The Problem
Current AI coding agents operate in isolation:
- Changes in one repository break dependencies in others
- No coordination between agents working on interconnected services
- Context fragmentation across project boundaries
- Human overhead required to synchronize cross-repo work

### The Solution
A hub-and-spoke orchestration architecture where:
- An **Orchestrator Agent** manages coordination and planning
- **Specialized Worker Agents** execute tasks in specific repositories
- A **Shared Workspace** maintains cross-repository context
- Humans interact through a single entry-point with full visibility

### Key Value Propositions
| Value Proposition | Description |
|------------------|-------------|
| **Scale** | Handle multi-repo architectures without losing coherence |
| **Specialization** | Deploy purpose-built agents (Tech Lead, Backend, Frontend, etc.) |
| **Coordination** | Automatic dependency management across services |
| **Transparency** | Single pane of glass for human oversight |

### Target Outcomes
- Reduce cross-repo coordination overhead by 70%
- Enable true parallel development on microservice architectures
- Maintain architectural coherence across distributed changes
- Preserve human agency with intervention capabilities

---

## 2. Core Features

### P0 Features (MVP - Must Have)

| Feature | Description | Success Criteria |
|---------|-------------|------------------|
| **Multi-Project Workspace** | Support 2-5 concurrent repositories with shared context | Successfully coordinate changes across 3 repos |
| **Orchestrator Agent** | Central planning agent that decomposes tasks and assigns work | 90% of tasks correctly decomposed and assigned |
| **Inter-Agent Communication** | Message passing between agents via shared workspace | <100ms latency for agent-to-agent messages |
| **Human Checkpoint System** | Required approval gates for critical decisions | All destructive operations require explicit approval |
| **Basic Agent Templates** | Tech Lead, Backend Developer, Frontend Developer | 3 core templates functional with distinct behaviors |
| **File-Based State** | Persistent state storage in JSON/YAML format | State survives restarts; human-readable format |

### P1 Features (v1 - Important)

| Feature | Description | Success Criteria |
|---------|-------------|------------------|
| **Dependency Graph Analysis** | Automatic mapping of cross-repo dependencies | 80% of dependencies auto-detected |
| **Conflict Detection** | Identify and flag potential merge conflicts across repos | Conflicts detected before execution |
| **Agent Lifecycle Management** | Start, pause, resume, terminate agent instances | Zero-downtime agent restarts |
| **Progress Dashboard** | Terminal UI showing all active agents and status | Real-time status for 10+ concurrent agents |
| **Custom Agent Builder** | YAML/JSON template system for custom agent types | New agent type defined in <10 minutes |
| **Rollback Coordination** | Synchronized rollback across multiple repositories | Rollback completes in <2 minutes |

### P2 Features (v2 - Nice to Have)

| Feature | Description | Success Criteria |
|---------|-------------|------------------|
| **Auto-Scaling** | Dynamic agent creation based on workload | Scale 1-10 agents based on queue depth |
| **Learning System** | Agent behavior improvement from past runs | 10% efficiency gain per month |
| **Advanced Visualizations** | Mermaid/Graphviz diagrams of system state | Generate architecture diagrams on-demand |
| **Integration Tests** | Cross-repo test orchestration | Run tests spanning 5 repos in parallel |
| **Cost Optimization** | Token usage tracking and optimization alerts | 20% token usage reduction |
| **Multi-Modal Support** | Image/design file coordination | Sync design changes across repos |

---

## 3. Feature Roadmap

### Phase 1: MVP (Months 1-3)
**Goal:** Prove core orchestration concept with minimal complexity

**Key Deliverables:**
- Week 1-2: Architecture design and shared workspace specification
- Week 3-4: Basic orchestrator agent with task decomposition
- Week 5-6: Agent communication protocol implementation
- Week 7-8: Three core agent templates (Tech Lead, Backend, Frontend)
- Week 9-10: Human checkpoint system and approval flows
- Week 11-12: End-to-end testing and documentation

**Success Gate:** Successfully coordinate a feature requiring changes across 3 repositories without human intervention beyond checkpoints.

### Phase 2: v1 Production Ready (Months 4-6)
**Goal:** Production-grade reliability and developer experience

**Key Deliverables:**
- Month 4: Dependency graph analysis and conflict detection
- Month 5: Agent lifecycle management and dashboard
- Month 6: Custom agent builder and comprehensive testing

**Success Gate:** 5+ teams using system weekly with <5% coordination failures.

### Phase 3: v2 Scale & Intelligence (Months 7-12)
**Goal:** Enterprise-scale operation with intelligent optimization

**Key Deliverables:**
- Month 7-8: Auto-scaling and load balancing
- Month 9-10: Learning system and pattern recognition
- Month 11-12: Advanced visualizations and cost optimization

**Success Gate:** Handle 50+ concurrent agents with automatic optimization.

### Timeline Visualization

```
Month:  1   2   3   4   5   6   7   8   9   10  11  12
        |---MVP---|
                    |----v1----|
                                |-------v2-------|
        
MVP:     [██████████]
v1:                  [██████████]
v2:                               [████████████████]
        
Key: █ = Active Development
```

---

## 4. User Stories

### Story 1: Sarah - Tech Lead Starting a New Feature

**User:** Sarah, Senior Engineer at a 20-person startup  
**Context:** Needs to implement user authentication that spans 3 microservices (API Gateway, User Service, Notification Service)  

**Scenario:**
1. Sarah opens her terminal and runs `crush-orchestrate init auth-rewrite`
2. She defines the project scope: "Implement JWT authentication across gateway, user-service, and notification-service"
3. The Orchestrator Agent spawns:
   - Tech Lead Agent (planning and coordination)
   - Backend Agent #1 (API Gateway changes)
   - Backend Agent #2 (User Service changes)
   - Backend Agent #3 (Notification Service changes)
4. Tech Lead Agent analyzes dependencies and creates execution plan:
   - User Service must be updated first (schema changes)
   - API Gateway depends on User Service
   - Notification Service can be done in parallel
5. Agents begin working, reporting progress to Sarah via terminal UI
6. At each checkpoint (before commits, before PRs), Sarah reviews and approves
7. All 3 PRs are ready simultaneously with coordinated changes

**Acceptance Criteria:**
- [ ] Multi-repo project initialized in <2 minutes
- [ ] Execution plan generated with correct dependency ordering
- [ ] All 3 agents work in parallel where possible
- [ ] Human checkpoints function at key decision points
- [ ] Total time: 30% faster than sequential approach

---

### Story 2: Marcus - Platform Engineer Maintaining Dependencies

**User:** Marcus, Platform Engineer at a mid-size company  
**Context:** Need to update a shared library used by 8 different services  

**Scenario:**
1. Marcus runs `crush-orchestrate plan shared-library-update`
2. System analyzes all 8 repos and identifies breaking vs non-breaking changes
3. Orchestrator creates execution groups:
   - Group A: 5 services (non-breaking updates, parallel)
   - Group B: 3 services (breaking changes, requires migration code)
4. Agents generate migration scripts where needed
5. Marcus reviews the plan, approves Group A for auto-execution
6. Group B requires individual approval for each service
7. All services updated with consistent library version
8. Integration tests run across all services to verify compatibility

**Acceptance Criteria:**
- [ ] Dependency analysis identifies all 8 affected repos
- [ ] Breaking changes correctly flagged and prioritized
- [ ] Migration scripts generated for breaking changes
- [ ] Selective approval workflow functions correctly
- [ ] Integration test coordination across repos works

---

### Story 3: Emily - CTO Reviewing Multiple Projects

**User:** Emily, CTO at a growing startup  
**Context:** Needs visibility into 5 active projects across engineering team  

**Scenario:**
1. Emily runs `crush-orchestrate dashboard` to see all active orchestrations
2. Dashboard shows:
   - Project A: 3 agents, 80% complete, 2 checkpoints pending
   - Project B: 5 agents, waiting for human input (database migration)
   - Project C: 1 agent, completed, ready for review
   - Project D: 2 agents, blocked (dependency conflict detected)
   - Project E: 4 agents, just started
3. Emily drills into Project B to approve the database migration
4. She reviews Project C's completed work and approves merge
5. She intervenes in Project D to resolve the conflict manually
6. She sets Project A to auto-approve remaining checkpoints (team is trusted)

**Acceptance Criteria:**
- [ ] Dashboard loads with all 5 projects in <3 seconds
- [ ] Project status accurately reflects agent states
- [ ] Drill-down provides detailed progress view
- [ ] Intervention commands execute correctly
- [ ] Auto-approval settings persist across sessions

---

### Story 4: James - Junior Developer Learning the System

**User:** James, Junior Backend Developer  
**Context:** First time using the orchestration system for a bug fix  

**Scenario:**
1. James is assigned a bug that requires changes in 2 repos
2. He runs `crush-orchestrate start --template backend-dev --repos repo-a,repo-b`
3. The system guides him through:
   - Describing the bug in natural language
   - Confirming the affected repositories
   - Understanding the proposed fix
4. A single Backend Developer agent is spawned with explicit instructions
5. The agent works on both repos, explaining each step
6. James learns from the agent's approach (educational mode enabled)
7. At each checkpoint, James reviews the changes and asks questions
8. The agent explains its reasoning and suggests alternatives

**Acceptance Criteria:**
- [ ] Onboarding flow guides new users in <5 minutes
- [ ] Educational mode provides explanations for each action
- [ ] Agent adapts to user's experience level
- [ ] User can ask questions at any checkpoint
- [ ] Tutorial completion increases user confidence

---

### Story 5: Priya - DevOps Engineer Customizing Agent Behavior

**User:** Priya, DevOps Lead  
**Context:** Needs specialized agents for infrastructure-as-code work  

**Scenario:**
1. Priya creates a custom agent template for Terraform work:
   ```yaml
   agent_type: terraform-specialist
   capabilities:
     - terraform_plan_review
     - cost_estimation
     - security_scanning
   constraints:
     - requires_approval: [apply,destroy]
     - max_resources_per_run: 10
   ```
2. She defines infrastructure change workflow:
   - Security Agent scans for vulnerabilities
   - Cost Agent estimates cloud spend changes
   - Terraform Agent generates and applies plans
3. Priya shares the template with her team via git
4. Team uses `crush-orchestrate start --template terraform-specialist`
5. The custom agent follows all defined constraints
6. Audit log tracks all actions for compliance

**Acceptance Criteria:**
- [ ] Custom template created in <15 minutes
- [ ] Template validation catches configuration errors
- [ ] Custom capabilities execute correctly
- [ ] Constraints enforced automatically
- [ ] Audit logging captures all required fields

---

### Story 6: Alex - Handling Emergency Production Fix

**User:** Alex, On-Call Engineer  
**Context:** Critical bug affecting production, needs fast coordinated fix  

**Scenario:**
1. Alert fires: "Payment processing down, affects 3 services"
2. Alex runs `crush-orchestrate emergency --severity critical`
3. System automatically:
   - Skips non-essential checkpoints
   - Notifies team leads via configured channels
   - Creates rollback plan before any changes
4. Orchestrator spawns specialized agents:
   - Incident Commander Agent (coordination)
   - Payment Service Agent (root cause)
   - Database Agent (schema check)
5. Agents work with elevated permissions but full audit logging
6. Fix deployed with automatic rollback timer (30 min)
7. Post-incident report auto-generated with timeline

**Acceptance Criteria:**
- [ ] Emergency mode activated in <1 minute
- [ ] Rollback plan created before any changes
- [ ] Team notifications sent automatically
- [ ] Full audit trail maintained despite urgency
- [ ] Post-incident report generated automatically

---

## 5. Competitive Differentiation

### Market Landscape

| Competitor | Approach | Limitation |
|------------|----------|------------|
| **AutoGPT** | Single autonomous agent with tool use | No multi-agent coordination; often loses context |
| **CrewAI** | Role-based multi-agent collaboration | Requires Python coding; no terminal integration |
| **LangChain** | Framework for agent composition | Low-level; requires significant setup |
| **Devin (Cognition)** | End-to-end autonomous engineer | Closed system; no multi-repo coordination |
| **GitHub Copilot Workspace** | AI-assisted code generation | Single repo focus; no agent autonomy |
| **Amazon CodeWhisperer** | AI pair programming | No project-level orchestration |

### Our Differentiation

#### 1. Terminal-Native Architecture
- **Others:** Web-based or IDE plugins
- **Us:** Native terminal experience matching OpenCode/Crush philosophy
- **Advantage:** Fits existing developer workflows; no context switching

#### 2. Repository-First Design
- **Others:** Treat code as generic text
- **Us:** Deep understanding of git workflows, PR processes, repo boundaries
- **Advantage:** Natural fit for real software development practices

#### 3. Human-in-the-Loop Philosophy
- **Others:** Full autonomy (AutoGPT) or full manual (Copilot)
- **Us:** Configurable autonomy with mandatory checkpoints for critical actions
- **Advantage:** Balance of efficiency and safety

#### 4. File-Based Simplicity
- **Others:** Complex database schemas, cloud dependencies
- **Us:** Plain text files (YAML/JSON) for state, configuration, and logs
- **Advantage:** Debuggable, portable, version-controllable

#### 5. Microservice-Aware Coordination
- **Others:** Single codebase focus
- **Us:** Built for multi-repo architectures from day one
- **Advantage:** Handles real-world distributed systems complexity

### Unique Value Proposition Matrix

| Capability | OpenCode Orchestration | Best Alternative | Gap |
|------------|------------------------|------------------|-----|
| Terminal-native | ✅ Core design | ⚠️ IDE plugins | First-class terminal |
| Multi-repo coordination | ✅ P0 feature | ❌ Not supported | Sole focus area |
| Human checkpoints | ✅ Configurable | ⚠️ Limited | Fine-grained control |
| File-based state | ✅ JSON/YAML | ❌ Database/cloud | Portable/debuggable |
| Agent templates | ✅ Customizable | ⚠️ Pre-defined | Fully extensible |
| Open source | ✅ Planned | ❌ Mostly closed | Community extensible |

### Positioning Statement

> For engineering teams managing microservice architectures, the OpenCode Agent Team Orchestration system provides coordinated multi-repository development where other AI tools only handle single repos. Unlike autonomous agents that operate in black boxes, our system maintains human oversight through configurable checkpoints while enabling true parallel development across service boundaries.

---

## 6. Integration Strategy

### OpenCode/Crush Architecture Compatibility

```
┌─────────────────────────────────────────────────────────────────┐
│                    HUMAN INTERFACE LAYER                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Terminal   │  │   Config     │  │   Logs       │          │
│  │   UI         │  │   Files      │  │   Viewer     │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
└─────────┼─────────────────┼─────────────────┼───────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                   ORCHESTRATION LAYER (New)                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              ORCHESTRATOR AGENT                         │   │
│  │  • Task decomposition    • State management            │   │
│  │  • Conflict resolution   • Checkpoint coordination     │   │
│  └──────────────────┬──────────────────────────────────────┘   │
│                     │                                          │
│         ┌───────────┼───────────┐                             │
│         ▼           ▼           ▼                             │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐                 │
│  │ Worker     │ │ Worker     │ │ Worker     │                 │
│  │ Agent #1   │ │ Agent #2   │ │ Agent #N   │                 │
│  │ (Repo A)   │ │ (Repo B)   │ │ (Repo N)   │                 │
│  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘                 │
│        │              │              │                         │
└────────┼──────────────┼──────────────┼─────────────────────────┘
         │              │              │
         ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    OPENCODE/CRUSH LAYER (Existing)              │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐                  │
│  │ OpenCode   │ │ OpenCode   │ │ OpenCode   │                  │
│  │ Instance   │ │ Instance   │ │ Instance   │                  │
│  │ (Repo A)   │ │ (Repo B)   │ │ (Repo N)   │                  │
│  └────────────┘ └────────────┘ └────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
```

### Integration Points

#### 1. Configuration Layer
**Location:** `~/.crush/orchestrator/config.yaml`

```yaml
orchestrator:
  version: "1.0.0"
  default_workspace: "~/.crush/orchestrator/workspaces"
  checkpoint_policy: "explicit"  # explicit, implicit, none
  
agents:
  max_concurrent: 10
  default_timeout: "30m"
  lifecycle_policy: "manual"  # manual, auto-suspend, auto-terminate
  
communication:
  protocol: "file_based"  # file_based, socket, http
  message_format: "json"
  retention_hours: 72
```

#### 2. Workspace Structure
**Location:** `~/.crush/orchestrator/workspaces/{project-id}/`

```
workspaces/
└── auth-rewrite-2026-04-01/
    ├── orchestrator-state.json      # Master state
    ├── execution-plan.yaml          # Generated plan
    ├── checkpoint-log.jsonl         # Decision history
    ├── agents/
    │   ├── tech-lead-agent/
    │   │   ├── config.yaml
    │   │   ├── state.json
    │   │   └── messages/
    │   ├── backend-api-gateway/
    │   └── backend-user-service/
    └── shared/
        ├── dependency-graph.json
        ├── conflict-report.yaml
        └── context-cache/
```

#### 3. OpenCode Hook Points

| Hook Point | Integration Method | Purpose |
|------------|-------------------|---------|
| CLI Extension | Subcommand (`crush orchestrate`) | Entry point |
| Config Loader | Additional config paths | Agent templates |
| Message Bus | File-based queue | Inter-agent communication |
| State Hook | State file watcher | Orchestrator awareness |
| Checkpoint | Pre-commit hook | Human approval gate |

#### 4. API Surface

**Orchestrator Commands:**
```bash
# Initialize new orchestrated project
crush-orchestrate init <project-name> --repos <repo-list>

# Start orchestration with specific template
crush-orchestrate start --template <template-name> --task <description>

# View dashboard
crush-orchestrate dashboard [--project <id>]

# Intervene at checkpoint
crush-orchestrate approve <checkpoint-id> --project <id>
crush-orchestrate reject <checkpoint-id> --reason <explanation>

# Manage agents
crush-orchestrator agent list --project <id>
crush-orchestrator agent pause <agent-id>
crush-orchestrator agent resume <agent-id>
crush-orchestrator agent logs <agent-id>

# Template management
crush-orchestrate template list
crush-orchestrate template create <name> --from <base>
crush-orchestrate template edit <name>
```

### Backward Compatibility

| OpenCode Version | Orchestration Support | Notes |
|------------------|----------------------|-------|
| Current (pre-Crush) | ⚠️ Limited | Requires adapter layer |
| Crush v1.0+ | ✅ Full | Native integration |
| Crush v2.0+ | ✅ Enhanced | Optimized for multi-agent |

---

## 7. Success Metrics

### Primary KPIs

| Metric | Target | Measurement Method | Timeline |
|--------|--------|-------------------|----------|
| **Coordination Success Rate** | >90% | % of multi-repo tasks completed without manual conflict resolution | Month 3 |
| **Developer Time Saved** | 30% | Time comparison: orchestrated vs manual multi-repo work | Month 3 |
| **Checkpoint Approval Rate** | >85% | % of agent-proposed actions approved without modification | Month 4 |
| **Adoption Rate** | 50% of teams | % of OpenCode users using orchestration weekly | Month 6 |
| **Mean Time to Resolution** | <2 hours | Average time from task start to PR ready | Month 6 |
| **System Uptime** | 99.5% | Availability of orchestration layer | Month 6 |

### Secondary KPIs

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| **Agent Spawn Time** | <30s | Time from request to agent ready |
| **Message Latency** | <100ms | P95 latency for agent-to-agent messages |
| **Memory Efficiency** | <500MB | Max memory per agent instance |
| **Token Efficiency** | Baseline -20% | LLM token usage vs single-agent approach |
| **User Satisfaction** | >4.0/5.0 | Post-task survey scores |

### Success Measurement Framework

```
┌─────────────────────────────────────────────────────────────┐
│                   SUCCESS PYRAMID                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                    [BUSINESS IMPACT]                        │
│               Developer Productivity +30%                   │
│                   Time-to-Market -25%                       │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                 [USER OUTCOMES]                     │  │
│  │      Multi-repo tasks painless and fast             │  │
│  │      Confidence in AI coordination                  │  │
│  │      Human oversight without micromanagement        │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │               [PRODUCT METRICS]                     │  │
│  │    Coordination success rate, Adoption, Engagement  │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              [SYSTEM HEALTH]                        │  │
│  │     Latency, Reliability, Resource efficiency       │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Data Collection Strategy

**Automatic Telemetry (Opt-in):**
- Task completion rates and timing
- Agent communication patterns
- Error rates and types
- Resource utilization

**User Feedback:**
- Post-task satisfaction surveys (1-5 scale)
- Qualitative feedback at checkpoints
- Feature request tracking
- Support ticket categorization

**A/B Testing Opportunities:**
- Different checkpoint frequencies
- Various agent template defaults
- Communication protocol options
- UI layout variations

### Reporting Dashboard

**Weekly Metrics Report:**
- Active orchestrations
- Success/failure rates
- Time savings achieved
- Top blockers and issues

**Monthly Business Review:**
- Adoption trends
- Productivity impact
- User satisfaction scores
- Competitive win/loss analysis

---

## 8. Technical Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              USER INTERFACE                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   CLI       │  │  Dashboard  │  │   Config    │  │   Human Interface   │ │
│  │   Parser    │  │   (TUI)     │  │   Manager   │  │   (Checkpoints)     │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
│         │                │                │                    │            │
│         └────────────────┴────────────────┴────────────────────┘            │
│                                    │                                        │
│                                    ▼                                        │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                      ORCHESTRATION ENGINE                              │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                 │ │
│  │  │   Task       │  │   Agent      │  │   State      │                 │ │
│  │  │   Planner    │  │   Scheduler  │  │   Manager    │                 │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                 │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                 │ │
│  │  │ Dependency   │  │ Checkpoint   │  │ Conflict     │                 │ │
│  │  │ Analyzer     │  │ Coordinator  │  │ Resolver     │                 │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                 │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                     SHARED WORKSPACE (File-Based)                      │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  │ │
│  │  │   State     │  │   Message   │  │   Context   │  │   Audit      │  │ │
│  │  │   Files     │  │   Queue     │  │   Cache     │  │   Log        │  │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └──────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    ▼                                        │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                      AGENT EXECUTION LAYER                             │ │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐           │ │
│  │  │  Tech Lead     │  │  Backend Dev   │  │  Frontend Dev  │           │ │
│  │  │  Agent         │  │  Agent         │  │  Agent         │           │ │
│  │  │                │  │                │  │                │           │ │
│  │  │ • Planning     │  │ • API Design   │  │ • UI/UX        │           │ │
│  │  │ • Architecture │  │ • DB Schema    │  │ • Components   │           │ │
│  │  │ • Review       │  │ • Testing      │  │ • Styling      │           │ │
│  │  └────────┬───────┘  └────────┬───────┘  └────────┬───────┘           │ │
│  │           │                   │                   │                    │ │
│  │           └───────────────────┴───────────────────┘                    │ │
│  │                               │                                        │ │
│  │                               ▼                                        │ │
│  │  ┌────────────────────────────────────────────────────────────────┐   │ │
│  │  │              OPENCODE/CRUSH AGENT INSTANCES                     │   │ │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │   │ │
│  │  │  │  Crush   │  │  Crush   │  │  Crush   │  │  Crush   │        │   │ │
│  │  │  │  Repo A  │  │  Repo B  │  │  Repo C  │  │  Repo N  │        │   │ │
│  │  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │   │ │
│  │  └────────────────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. Orchestration Engine
| Component | Technology | Responsibility |
|-----------|------------|----------------|
| Task Planner | Python/Go | Decompose high-level tasks into agent-assignable subtasks |
| Agent Scheduler | Rust/Go | Manage agent lifecycle and resource allocation |
| State Manager | File-based JSON | Persist and synchronize system state |
| Dependency Analyzer | Tree-sitter + Graph | Build and maintain cross-repo dependency graphs |
| Checkpoint Coordinator | Event-driven | Manage human approval workflow |
| Conflict Resolver | Rule-based + LLM | Detect and suggest resolutions for conflicts |

#### 2. Shared Workspace

**State File Schema (`orchestrator-state.json`):**
```json
{
  "version": "1.0.0",
  "project_id": "auth-rewrite-2026-04-01",
  "status": "in_progress",
  "created_at": "2026-04-01T10:00:00Z",
  "updated_at": "2026-04-01T14:30:00Z",
  "agents": [
    {
      "id": "tech-lead-001",
      "type": "tech-lead",
      "status": "active",
      "assigned_repos": ["all"],
      "current_task": "review-architecture"
    },
    {
      "id": "backend-001",
      "type": "backend-dev",
      "status": "working",
      "assigned_repos": ["user-service"],
      "current_task": "implement-jwt"
    }
  ],
  "tasks": [
    {
      "id": "task-001",
      "description": "Implement JWT authentication",
      "status": "in_progress",
      "dependencies": [],
      "assigned_agents": ["backend-001"],
      "checkpoints": [
        {
          "id": "cp-001",
          "type": "pre_commit",
          "status": "pending_approval",
          "created_at": "2026-04-01T14:25:00Z"
        }
      ]
    }
  ],
  "metadata": {
    "total_commits": 5,
    "files_modified": 12,
    "estimated_completion": "2026-04-01T16:00:00Z"
  }
}
```

#### 3. Agent Execution Layer

**Agent Template Schema (`templates/backend-dev.yaml`):**
```yaml
api_version: "v1"
template:
  name: "backend-dev"
  description: "Specialized agent for backend development tasks"
  version: "1.0.0"
  
  capabilities:
    - api_design
    - database_schema
    - business_logic
    - unit_testing
    - code_review
    
  constraints:
    max_file_size: "100KB"
    allowed_languages: ["python", "go", "rust", "java"]
    forbidden_operations: ["force_push", "delete_branch"]
    
  prompts:
    system: |
      You are a senior backend developer specializing in API design
      and database optimization. Always consider:
      1. API versioning and backward compatibility
      2. Database query efficiency
      3. Error handling patterns
      4. Security best practices
      
    task_prefix: "[Backend Task] "
    
  checkpoint_rules:
    pre_commit: "required"
    pre_pr: "required"
    destructive_operation: "required"
    test_failure: "notify"
```

### Communication Protocol

**Message Format (`shared/messages/agent-001.jsonl`):**
```jsonl
{"timestamp":"2026-04-01T14:30:00Z","type":"task_started","from":"orchestrator","to":"backend-001","payload":{"task_id":"task-001"}}
{"timestamp":"2026-04-01T14:35:00Z","type":"status_update","from":"backend-001","to":"orchestrator","payload":{"status":"in_progress","progress":25,"files_touched":["auth.py","models.py"]}}
{"timestamp":"2026-04-01T14:45:00Z","type":"checkpoint_reached","from":"backend-001","to":"orchestrator","payload":{"checkpoint_id":"cp-001","type":"pre_commit","description":"Ready to commit JWT implementation"}}
```

### Technology Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| CLI/TUI | Go (Charm ecosystem) | Consistent with Crush, excellent TUI libraries |
| Orchestration Engine | Rust or Go | Performance, reliability, concurrency |
| Agent Runtime | Python | LLM integration, OpenCode compatibility |
| State Storage | JSON/YAML files | Portability, version control, debuggability |
| Communication | File-based queue + Optional sockets | Simplicity first, performance second |
| Configuration | YAML | Human-readable, widely understood |

### Scalability Considerations

| Metric | MVP Target | v1 Target | v2 Target |
|--------|------------|-----------|-----------|
| Concurrent Agents | 5 | 20 | 100+ |
| Repositories per Project | 5 | 20 | 50+ |
| Active Projects | 10 | 50 | 200+ |
| Message Throughput | 10/sec | 100/sec | 1000/sec |
| State File Size | <10MB | <50MB | <100MB (sharded) |

---

## 9. Risk Analysis

### Risk Matrix

| Risk | Probability | Impact | Mitigation Strategy | Owner |
|------|-------------|--------|---------------------|-------|
| **Agent Coordination Failures** | Medium | High | - Implement transaction-like semantics<br>- Automatic rollback on failure<br>- State snapshots before operations | Engineering |
| **LLM Context Overflow** | High | Medium | - Aggressive context pruning<br>- File-based context cache<br>- Hierarchical summarization | AI/ML Team |
| **State File Corruption** | Low | Critical | - Atomic writes with temp files<br>- Automatic backup generation<br>- Validation schemas | Engineering |
| **Cross-Repo Merge Conflicts** | High | Medium | - Dependency-aware ordering<br>- Early conflict detection<br>- Staging area for coordination | Engineering |
| **Human Oversight Overload** | Medium | High | - Smart checkpoint batching<br>- ML-based urgency scoring<br>- Auto-approval for low-risk changes | Product |
| **Performance at Scale** | Medium | High | - Lazy loading of state<br>- Incremental sync protocols<br>- Optional database backend | Engineering |
| **Security (Agent Permissions)** | Medium | Critical | - Principle of least privilege<br>- Audit logging for all actions<br>- Sandboxed execution | Security |
| **Integration Complexity** | High | Medium | - Adapter pattern for OpenCode<br>- Gradual rollout strategy<br>- Fallback to single-agent mode | Engineering |
| **User Adoption Resistance** | Medium | High | - Extensive onboarding flow<br>- Educational mode<br>- Gradual autonomy settings | Product |
| **Vendor Lock-in (LLM)** | Low | Medium | - Abstract LLM interface<br>- Support multiple providers<br>- Local model compatibility | Engineering |

### Detailed Risk Mitigation Plans

#### Risk 1: Agent Coordination Failures
**Scenario:** Multiple agents attempt conflicting operations, causing system-wide inconsistency.

**Prevention:**
- Implement distributed locking mechanism
- Two-phase commit protocol for cross-repo operations
- Mandatory conflict detection before execution

**Detection:**
- Real-time state validation
- Anomaly detection in agent behavior
- Health check endpoints

**Response:**
- Automatic pause of affected agents
- Rollback to last known good state
- Human notification with suggested resolution

**Monitoring:**
- Coordination success rate dashboards
- Mean time between failures tracking
- Escalation alerts for repeated issues

---

#### Risk 2: LLM Context Overflow
**Scenario:** Multi-agent coordination generates too much context for LLM to process effectively.

**Prevention:**
- Token budget management per agent
- Automatic context summarization
- Focused information retrieval

**Detection:**
- Token usage monitoring
- Response quality scoring
- Context size alerts

**Response:**
- Automatic context compression
- Handoff to specialized agents with focused context
- Human intervention for complex cases

**Monitoring:**
- Token efficiency metrics
- Context pruning effectiveness
- User satisfaction scores

---

#### Risk 3: Security - Agent Permissions
**Scenario:** Compromised agent gains excessive permissions and causes damage.

**Prevention:**
- Sandboxed execution environments
- Capability-based security model
- All destructive operations require explicit approval

**Detection:**
- Behavior anomaly detection
- Permission usage monitoring
- Audit log analysis

**Response:**
- Immediate agent termination
- Automatic state snapshot preservation
- Security incident response procedure

**Monitoring:**
- Security audit dashboards
- Permission escalation alerts
- Compliance reporting

---

### Contingency Planning

**Plan A (File-Based State):**
- Primary architecture using JSON/YAML files
- Simple, debuggable, version-controllable
- Works for MVP through v1

**Plan B (Database Backend):**
- If file-based state hits scalability limits
- SQLite for local, PostgreSQL for team server
- Transparent migration path

**Plan C (Cloud Coordination):**
- If on-premise coordination insufficient
- Optional cloud-based orchestration service
- Hybrid architecture supported

**Fallback Mode:**
- If orchestration fails, agents fall back to single-agent mode
- No data loss, graceful degradation
- Clear user notification of mode switch

---

## 10. Appendix

### A. Glossary

| Term | Definition |
|------|------------|
| **Agent** | An AI-powered worker with specific capabilities and constraints |
| **Agent Template** | Configuration defining an agent's role, skills, and behavior |
| **Checkpoint** | Required human approval gate before critical operations |
| **Cross-Repo Dependency** | Code or API contract between different repositories |
| **Execution Plan** | Ordered list of tasks with dependencies and assignments |
| **Hub-and-Spoke** | Architecture where a central orchestrator coordinates multiple workers |
| **Human-in-the-Loop** | Design pattern requiring human approval for key decisions |
| **Message Queue** | Asynchronous communication channel between agents |
| **Orchestrator** | Central agent responsible for planning and coordination |
| **Shared Workspace** | File-based directory where agents share state and context |
| **Specialized Agent** | Agent configured for a specific domain (backend, frontend, etc.) |
| **State File** | JSON/YAML file persisting the system's current condition |
| **Worker Agent** | Agent spawned to execute tasks in a specific repository |

### B. References

#### Related Projects
- [OpenCode](https://github.com/opencode-ai/opencode) - Original terminal-based AI coding agent
- [Crush](https://github.com/charmbracelet/crush) - Successor to OpenCode under Charm team
- [CrewAI](https://github.com/joaomdmoura/crewAI) - Framework for orchestrating role-playing AI agents
- [AutoGPT](https://github.com/Significant-Gravitas/AutoGPT) - Autonomous GPT-4 experiment
- [LangChain](https://github.com/langchain-ai/langchain) - Framework for LLM applications

#### Technical Resources
- [Multi-Agent Systems Research](https://arxiv.org/list/multi-agent-systems/recent) - Academic papers
- [Charm Ecosystem](https://charm.sh/) - Terminal UI libraries for Go
- [Temporal.io](https://temporal.io/) - Durable execution patterns

#### Internal Documents
- OpenCode Architecture Overview (internal)
- Crush Migration Plan (internal)
- Agent Safety Guidelines (internal)

### C. Template Reference

#### Complete Agent Template Example

```yaml
# agent-template.yaml
api_version: "v1"
metadata:
  name: "senior-backend-engineer"
  description: "Senior-level backend developer with security focus"
  version: "2.1.0"
  author: "platform-team@company.com"
  tags: ["backend", "senior", "security", "api-design"]

capabilities:
  primary:
    - rest_api_design
    - graphql_schema_design
    - database_optimization
    - performance_tuning
    - security_audit
  secondary:
    - code_review
    - mentoring
    - architecture_review
    
constraints:
  operations:
    allowed:
      - branch_create
      - commit
      - push
      - pr_create
      - test_run
    forbidden:
      - force_push
      - branch_delete
      - production_deploy
      - secret_modification
    requires_approval:
      - database_migration
      - api_version_change
      - auth_modification
      
  resources:
    max_file_size: "100KB"
    max_files_per_commit: 20
    max_lines_changed: 500
    
  languages:
    primary: ["python", "go", "rust"]
    can_read: ["javascript", "typescript", "sql"]
    
  repositories:
    max_concurrent: 3
    allowed_domains: ["github.com/company"]

behavior:
  autonomy_level: "high"  # low, medium, high
  communication_frequency: "checkpoint_only"  # continuous, hourly, checkpoint_only
  error_handling: "escalate"  # retry, skip, escalate, halt
  
  style:
    code_style: "pep8"  # configurable
    documentation: "required"  # required, optional, none
    testing: "mandatory"  # mandatory, recommended, optional
    
  preferences:
    prefers_rebase: false
    commit_message_style: "conventional"  # conventional, detailed, minimal
    pr_description_detail: "high"

prompts:
  system: |
    You are a senior backend engineer with 10+ years of experience.
    Your expertise includes:
    - Designing scalable RESTful and GraphQL APIs
    - Database optimization and query tuning
    - Security best practices and vulnerability assessment
    - Performance profiling and optimization
    
    When working:
    1. Always consider backward compatibility
    2. Document security considerations
    3. Include performance impact analysis
    4. Write comprehensive tests
    5. Consider observability (logging, metrics, tracing)
    
    You are working alongside other specialized agents. Coordinate
    through the orchestrator for cross-cutting concerns.
    
  task_prefix: "[Backend Engineering Task] "
  
  examples:
    - input: "Add user authentication"
      output: |
        I'll implement secure user authentication with:
        1. JWT token generation with proper expiration
        2. Secure password hashing (Argon2)
        3. Rate limiting for login attempts
        4. Refresh token rotation
        5. Comprehensive audit logging
        
        Security considerations:
        - Tokens will expire after 1 hour
        - Failed login attempts will be rate-limited
        - All auth events will be logged
        
    - input: "Optimize slow query"
      output: |
        I'll analyze and optimize the query performance:
        1. Review current query execution plan
        2. Identify missing indexes
        3. Consider query restructuring
        4. Evaluate caching strategies
        5. Benchmark before/after performance
        
checkpoints:
  default_policy: "required"
  
  rules:
    - trigger: "pre_commit"
      policy: "required"
      description: "Review all changes before committing"
      
    - trigger: "pre_push"
      policy: "required"
      description: "Final review before pushing to remote"
      
    - trigger: "test_failure"
      policy: "notify"
      description: "Alert on test failures, suggest fixes"
      
    - trigger: "security_scan_failure"
      policy: "required"
      description: "Mandatory review for security issues"
      
    - trigger: "performance_regression"
      policy: "notify"
      description: "Alert on performance degradation"
      
integrations:
  tools:
    - name: "pytest"
      command: "pytest -v --tb=short"
      required: true
      
    - name: "black"
      command: "black --check --diff ."
      required: true
      
    - name: "bandit"
      command: "bandit -r . -f json"
      required: true
      
    - name: "mypy"
      command: "mypy --strict ."
      required: false
      
  notifications:
    channels:
      - type: "slack"
        webhook: "${SLACK_WEBHOOK_URL}"
        events: ["checkpoint", "error", "completion"]
        
      - type: "email"
        recipients: ["${USER_EMAIL}"]
        events: ["completion", "failure"]

logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  destination: "file"  # file, stdout, both
  retention_days: 30
  
  sensitive_patterns:
    - "password"
    - "token"
    - "secret"
    - "key"
```

### D. Command Reference

#### Full CLI Specification

```bash
# PROJECT MANAGEMENT

crush-orchestrate init <name> [options]
  --repos <repo-list>          # Comma-separated list of repos
  --template <template>        # Default agent template
  --description <desc>         # Project description
  --from <config-file>         # Initialize from existing config

crush-orchestrate list
  --status <status>            # Filter by status
  --created-after <date>       # Filter by creation date
  
crush-orchestrate delete <project-id>
  --force                      # Skip confirmation
  --archive                    # Archive instead of delete

# EXECUTION

crush-orchestrate start <project-id> [options]
  --task <description>         # Task description
  --template <template>        # Override default template
  --checkpoint-policy <policy> # Override checkpoint settings
  --dry-run                    # Plan only, don't execute

crush-orchestrate pause <project-id>
crush-orchestrate resume <project-id>
crush-orchestrate cancel <project-id>
  --graceful                   # Wait for current tasks
  --force                      # Immediate termination

crush-orchestrate status <project-id>
  --watch                      # Live updates
  --format <format>            # json, yaml, table

# CHECKPOINT MANAGEMENT

crush-orchestrate checkpoints <project-id>
  --pending                    # Show only pending
  --approved                   # Show approved
  --rejected                   # Show rejected

crush-orchestrate approve <checkpoint-id>
  --project <project-id>       # Required if multiple projects
  --with-comment <comment>     # Approval comment
  --auto-approve-similar       # Auto-approve similar future checkpoints

crush-orchestrate reject <checkpoint-id>
  --project <project-id>
  --reason <reason>            # Required rejection reason
  --suggest-alternative <alt>  # Suggest different approach

# AGENT MANAGEMENT

crush-orchestrate agent list <project-id>
  --active                     # Only active agents
  --type <type>                # Filter by agent type

crush-orchestrate agent spawn <project-id>
  --template <template>        # Agent template to use
  --repo <repo>                # Target repository
  --task <task>                # Initial task assignment

crush-orchestrate agent pause <agent-id>
crush-orchestrate agent resume <agent-id>
crush-orchestrate agent terminate <agent-id>
  --graceful                   # Allow cleanup

crush-orchestrate agent logs <agent-id>
  --follow                     # Tail logs
  --since <duration>           # Logs from last N minutes
  --level <level>              # Filter by log level

crush-orchestrate agent message <agent-id> <message>
  --priority <priority>        # normal, high, urgent
  --require-response           # Wait for agent response

# TEMPLATE MANAGEMENT

crush-orchestrate template list
  --category <category>        # Filter by category
  --builtin                    # Show only built-in
  --custom                     # Show only custom

crush-orchestrate template show <name>
  --format <format>            # yaml, json

crush-orchestrate template create <name>
  --from <base-template>       # Inherit from existing
  --edit                       # Open in editor

crush-orchestrate template edit <name>
crush-orchestrate template delete <name>
crush-orchestrate template validate <file>
  --strict                     # Strict validation

# DASHBOARD & VISUALIZATION

crush-orchestrate dashboard [project-id]
  --refresh <seconds>          # Auto-refresh interval
  --compact                    # Compact view

crush-orchestrate visualize <project-id>
  --type <type>                # dependency, timeline, agent-network
  --output <file>              # Save to file
  --format <format>            # mermaid, dot, svg

# WORKSPACE MANAGEMENT

crush-orchestrate workspace list
crush-orchestrate workspace create <path>
crush-orchestrate workspace set-default <path>
crush-orchestrate workspace cleanup
  --older-than <days>          # Remove old workspaces
  --dry-run                    # Preview only

# CONFIGURATION

crush-orchestrate config get <key>
crush-orchestrate config set <key> <value>
crush-orchestrate config list
crush-orchestrate config reset <key>

# DEBUGGING

crush-orchestrate debug state <project-id>
  --validate                   # Validate state integrity
  --repair                     # Attempt automatic repair

crush-orchestrate debug simulate <project-id>
  --scenario <scenario>        # Simulate specific scenario
  --steps <steps>              # Number of steps to simulate

crush-orchestrate export <project-id>
  --format <format>            # json, yaml, html-report
  --include-logs               # Include full logs
  --include-state-history      # Include state snapshots

# HELP

crush-orchestrate help <command>
crush-orchestrate examples                    # Show usage examples
crush-orchestrate version
crush-orchestrate doctor                      # Diagnostic checks
```

### E. State Transition Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PROJECT LIFECYCLE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────┐     init      ┌──────────┐     start      ┌──────────┐       │
│   │  NEW    │──────────────▶│ PLANNING │───────────────▶│ RUNNING  │       │
│   └─────────┘               └──────────┘                └────┬─────┘       │
│                                                               │             │
│         ▲                                                    │              │
│         │                                            ┌────────┴────────┐    │
│         │                                            ▼                 ▼    │
│   ┌─────┴─────┐                              ┌──────────┐        ┌────────┐ │
│   │ COMPLETED │◀─────────────────────────────│ CHECKPOINT │◀─────│ WORKING│ │
│   └───────────┘                              └────┬─────┘        └────────┘ │
│        ▲                                          │                         │
│        │                                          │ reject                  │
│        │                                     ┌────┴────┐                    │
│        │                                     │ BLOCKED │                    │
│        │                                     └────┬────┘                    │
│        │                                          │                         │
│        │            resolve/cancel                │                         │
│        └──────────────────────────────────────────┘                         │
│                                                                             │
│   Additional Transitions:                                                   │
│   - Any state ──cancel──▶ CANCELLED                                         │
│   - RUNNING ──pause──▶ PAUSED ──resume──▶ RUNNING                          │
│   - Any state ──error──▶ ERROR ──recover──▶ PLANNING (retry)               │
│   - ERROR ──irrecoverable──▶ FAILED                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### F. FAQ

**Q: Does this replace OpenCode/Crush?**  
A: No, this is an orchestration layer that coordinates multiple OpenCode/Crush instances. Single-agent mode continues to work as before.

**Q: Can I use this with non-OpenCode agents?**  
A: The architecture is extensible. While initially designed for OpenCode/Crush, the agent interface can be adapted for other tools.

**Q: What happens if the orchestrator crashes?**  
A: State is persisted to files. On restart, the orchestrator recovers from the last known state. Running agents may need to be reconnected.

**Q: How does billing work with multiple agents?**  
A: Each agent consumes tokens independently. The system tracks per-agent and total token usage with configurable limits.

**Q: Can agents from different LLM providers work together?**  
A: Yes, the orchestration layer is LLM-agnostic. Different agents can use different providers (OpenAI, Anthropic, local models, etc.).

**Q: Is there a cloud-hosted version?**  
A: The initial release is self-hosted. A cloud version may be offered based on demand.

**Q: How do I debug when things go wrong?**  
A: All state, messages, and decisions are stored in human-readable files. The `crush-orchestrate debug` command provides diagnostic tools.

**Q: Can I use this with private repositories?**  
A: Yes, the system works with any git repository you have access to. No code leaves your environment unless configured.

---

## Document Control

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-04-01 | Product Team | Initial version |

---

*This document is a living specification. As the product evolves, sections will be updated to reflect implementation realities and user feedback.*
