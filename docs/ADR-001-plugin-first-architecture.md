# ADR-001: Plugin-First Architecture with Go CLI Wrapper

**Status:** Accepted
**Date:** 2026-07-17
**Deciders:** Product Engineering Team
**Supersedes:** Original standalone Go architecture

---

## Context

The Agent Team Orchestration system was initially planned as a standalone Go binary with its own CLI, agent runtime, LLM integration, TUI dashboard, and message bus. This approach would require:

- ~5000+ lines of Go code
- 8-12 weeks for MVP with 2-3 developers
- Building AI infrastructure from scratch (LLM calls, session management, git ops)
- Maintaining a separate binary alongside OpenCode/Crush

OpenCode already provides:
- Plugin system (JS/TS) with event hooks
- Custom Tools API (JS/TS with Zod schemas)
- Agent configuration system (Markdown/JSON)
- SDK for programmatic session control
- TUI, git integration, LLM integration, file operations

The user wants to minimize memory usage by using Go for the wrapper/CLI while leveraging OpenCode's existing extension system for orchestration logic.

---

## Decision

**Build the system as a hybrid: Go CLI wrapper + OpenCode TypeScript plugin.**

### Architecture Split

```
┌─────────────────────────────────────────────────────────┐
│                    GO WRAPPER (CLI)                      │
│  crush-orchestrator init / start / stop / status         │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Process      │  │ State        │  │ HTTP API     │  │
│  │ Manager      │  │ Manager      │  │ (localhost)  │  │
│  │ (spawn/kill) │  │ (canonical)  │  │ (for TS)     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────┬───────────────────────────────┘
                          │ HTTP (localhost:9800)
                          │
┌─────────────────────────┴───────────────────────────────┐
│                 OPENCODE INSTANCES                       │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │           ORCHESTRATION PLUGIN (TS)               │   │
│  │  - Hooks into OpenCode events                     │   │
│  │  - Defines custom tools                           │   │
│  │  - Calls Go HTTP API for state mutations          │   │
│  │  - Reads state directly for display               │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │           AGENT CONFIGS (Markdown)                │   │
│  │  tech-lead.md  │  backend-dev.md  │  frontend.md  │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Responsibility Split

| Responsibility | Go Wrapper | TS Plugin |
|---|---|---|
| Process lifecycle | ✅ Start/stop/monitor OpenCode instances | ❌ |
| Port allocation | ✅ Manage ports (9800+) | ❌ |
| State ownership | ✅ Canonical state (single writer) | ❌ Read via HTTP |
| State mutations | ✅ HTTP API for mutations | ✅ Calls Go API |
| State display | ❌ | ✅ Reads state for agent context |
| CLI UX | ✅ `init/start/stop/status/dashboard` | ❌ |
| Agent hooks | ❌ | ✅ Event hooks |
| Custom tools | ❌ | ✅ Tool definitions |
| Agent configs | ✅ Generate/write | ✅ Read/use |
| Multi-repo spawn | ✅ Spawn `opencode` per repo | ✅ SDK connections |
| Health monitoring | ✅ Heartbeat checks | ✅ Report via events |
| Checkpoint UI | ✅ Terminal prompts | ✅ Trigger via tool |
| Memory footprint | ~10-20MB Go binary | Inside OpenCode runtime |

---

## Alternatives Considered

### Alternative 1: Pure TypeScript Plugin (No Go Wrapper)

**Pros:**
- Single language
- Simpler architecture
- No IPC needed

**Cons:**
- No process management (can't spawn OpenCode instances)
- Requires Node/Bun installed
- Higher memory (Bun runtime)
- No standalone CLI distribution

**Rejected because:** User explicitly wants Go wrapper for memory efficiency and process management.

### Alternative 2: Pure Go Standalone (Original Plan)

**Pros:**
- Single binary
- Full control
- No dependencies

**Cons:**
- 8-12 weeks to MVP
- ~5000+ lines of code
- Must build AI infrastructure from scratch
- Separate tool from OpenCode

**Rejected because:** Too much effort, duplicates OpenCode's capabilities.

### Alternative 3: MCP Server (Go) + OpenCode

**Pros:**
- Standard protocol
- Go handles heavy lifting
- OpenCode connects via MCP

**Cons:**
- MCP tools are stateless (no session state)
- Can't hook into OpenCode events
- Can't define agents
- More complex than plugin

**Rejected because:** MCP doesn't support event hooks or agent definitions.

---

## Consequences

### Positive
- 3-5x faster to MVP than standalone
- Leverages OpenCode's mature AI infrastructure
- Go wrapper provides efficient process management
- Single Go binary for distribution
- TypeScript plugin is easy to modify/extend

### Negative
- Two codebases to maintain (Go + TS)
- IPC complexity between Go and TS
- Dependency on OpenCode's plugin API stability
- Need to keep Go and TS types in sync

### Risks
- OpenCode plugin API changes → Mitigated by pinning SDK version
- IPC latency → Mitigated by localhost HTTP (sub-ms)
- State corruption → Mitigated by Go owning state (single writer)

---

## Implementation Notes

1. **Go wrapper uses existing codebase** — `cmd/orchestrator/`, `pkg/workspace/`, `pkg/types/`
2. **Gin HTTP API** (already in go.mod) serves state to TS plugin
3. **TS plugin** goes in `.opencode/plugins/orchestration.ts`
4. **Agent configs** go in `.opencode/agents/`
5. **Shared secret** for Go ↔ TS authentication
6. **Per-entity state files** (tasks/*.json, messages/*.json) instead of single state.json

---

## References

- [OpenCode Plugin Docs](https://opencode.ai/docs/plugins/)
- [OpenCode Custom Tools](https://opencode.ai/docs/custom-tools/)
- [OpenCode Agents](https://opencode.ai/docs/agents/)
- [OpenCode SDK](https://opencode.ai/docs/sdk/)
- Original plan: `docs/PRODUCT-PLAN.md`
- Plugin plan: `docs/PRODUCT-PLAN-PLUGIN-APPROACH.md`
