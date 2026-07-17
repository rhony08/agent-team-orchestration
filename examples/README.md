# Examples

This directory contains example configurations and templates for the Agent Team Orchestration system.

## Directory Structure

```
examples/
├── workspace.yaml              # Example workspace configuration
├── templates/
│   ├── tech-lead.yaml         # Tech Lead agent template
│   └── backend-dev.yaml       # Backend Developer agent template
└── tasks/
    └── TASK-001.yaml          # Example task definition
```

## Usage

### 1. Workspace Configuration

Copy and customize `workspace.yaml` for your project:

```bash
cp examples/workspace.yaml my-project/workspace.yaml
# Edit my-project/workspace.yaml
```

### 2. Agent Templates

Create custom agent templates in `~/.crush/orchestrator/templates/`:

```bash
mkdir -p ~/.crush/orchestrator/templates
cp examples/templates/tech-lead.yaml ~/.crush/orchestrator/templates/
# Customize as needed
```

### 3. Task Definitions

Tasks are automatically created by the orchestrator, but you can view `TASK-001.yaml` for reference.

## Creating Custom Templates

To create a new agent template:

1. Create a YAML file in `templates/`
2. Define the role, capabilities, and system prompt
3. Add custom commands specific to the role
4. Specify required permissions

Example:

```yaml
name: "DevOps Engineer"
role: "devops"
system_prompt: |
  You are a DevOps Engineer responsible for infrastructure...

capabilities:
  - id: "terraform"
    description: "Manage Terraform configurations"
  - id: "kubernetes"
    description: "Deploy to Kubernetes"

custom_commands:
  - name: "deploy-service"
    description: "Deploy a service to Kubernetes"
    prompt: |
      Deploy {{service}} to {{environment}}...
```

## More Examples

See the main documentation for more complex scenarios:
- Multi-service feature development
- Emergency production fixes
- Large-scale refactoring
- Cross-team coordination
