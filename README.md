# Agent Team Orchestration

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

> Orchestrate multiple OpenCode/Crush agents across microservices repositories

## 🚧 Status: Planning Phase

This project is in the planning and design phase. We are defining the architecture and gathering feedback before implementation begins.

## 📁 Project Structure

```
agent-team-orchestration/
├── cmd/
│   └── orchestrator/          # Main orchestrator CLI
├── pkg/
│   ├── types/                 # Core data types
│   └── workspace/             # Workspace management
├── examples/                  # Example configurations
│   ├── workspace.yaml
│   └── templates/
├── docs/
│   ├── PRODUCT-PLAN.md        # Product roadmap and features
│   ├── ARCHITECTURE-BACKEND.md # Backend system design
│   └── ARCHITECTURE-FRONTEND.md # TUI design
├── README.md
├── LICENSE
└── go.mod
```

## 📚 Documentation

- **[Product Plan](docs/PRODUCT-PLAN.md)** - Features, roadmap, user stories
- **[Backend Architecture](docs/ARCHITECTURE-BACKEND.md)** - System design, APIs, data models
- **[Frontend Architecture](docs/ARCHITECTURE-FRONTEND.md)** - TUI screens and components

## 🎯 Vision

Enable multiple OpenCode/Crush instances to coordinate and collaborate across different repositories through a central orchestrator with shared workspace.

## 🔑 Key Features

- **Multi-Project Workspace**: Coordinate 2-5+ concurrent repositories
- **Agent Templates**: Pre-built and custom agent types (Tech Lead, Backend Dev, etc.)
- **Shared Workspace**: File-based persistent state and context
- **Message Bus**: Real-time agent communication via hub
- **Human Checkpoints**: Approval gates for critical actions
- **TUI Dashboard**: Real-time visualization of all agents

## 🚀 Quick Start (Future)

```bash
# Install
go install github.com/yourusername/agent-team-orchestration/cmd/orchestrator@latest

# Initialize workspace
crush-orchestrator init-workspace my-project

# Add agents
crush-orchestrator add-agent --role=tech-lead --repo=./api-gateway
crush-orchestrator add-agent --role=backend-dev --repo=./user-service

# Start orchestration
crush-orchestrator start --workspace=my-project
```

## 🤝 Contributing

This project is in the early planning phase. We welcome:

- **Feedback** on the architecture and features
- **Use cases** from your microservices projects
- **Contributions** once implementation begins

See our [Product Plan](docs/PRODUCT-PLAN.md) for current priorities.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- [OpenCode](https://github.com/opencode-ai/opencode) / [Crush](https://github.com/charmbracelet/crush) for the terminal AI agent
- [Charm.sh](https://charm.sh) for the beautiful TUI libraries

---

**Questions or Ideas?** Open an issue or start a discussion!
