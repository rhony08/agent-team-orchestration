// cmd/orchestrator/main.go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/rhony08/agent-team-orchestration/pkg/api"
	"github.com/rhony08/agent-team-orchestration/pkg/process"
	"github.com/rhony08/agent-team-orchestration/pkg/state"
)

var (
	version = "0.1.0"
	commit  = "dev"
)

// Config represents the orchestration config
type Config struct {
	Version    string   `json:"version"`
	Project    string   `json:"project"`
	Repos      []Repo   `json:"repos"`
	AuthSecret string   `json:"-"`
	CreatedAt  string   `json:"created_at"`
}

// Repo represents a repository configuration
type Repo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Port   int    `json:"port"`
	IsGit  bool   `json:"is_git"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "crush-orchestrator",
		Short: "Orchestrate multiple OpenCode agents across projects",
		Long: `Agent Team Orchestrator for OpenCode

Coordinate multiple AI coding agents across different projects
through a central hub with shared workspace.

Works with git repositories and non-git projects.`,
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(statusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	var repos []string
	var force bool
	var cloneDir string

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new orchestration workspace",
		Long: `Initialize a new orchestration workspace.

Projects can be specified as:
  - Local paths: /path/to/project, ~/code/project
  - Remote URLs: git@github.com:org/repo.git, https://github.com/org/repo.git

Remote repositories will be cloned to the specified clone directory.
Git is optional - works with any project directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			stateDir := ".orchestrator"

			// Check if already exists
			if _, err := os.Stat(stateDir); err == nil && !force {
				return fmt.Errorf("workspace already exists. Use --force to overwrite")
			}

			// Validate project name
			if strings.ContainsAny(projectName, " \t\n") {
				return fmt.Errorf("project name must not contain spaces")
			}

			// Validate repos
			if len(repos) == 0 {
				return fmt.Errorf("at least one project required (--repos)")
			}

			// Create clone directory if needed
			if cloneDir == "" {
				cloneDir = filepath.Join(stateDir, "repos")
			}

			var repoConfigs []Repo
			for i, repoRef := range repos {
				var absPath string
				var repoName string
				var isGit bool

				if isRemoteURL(repoRef) {
					// Remote URLs must be git
					repoName = extractRepoName(repoRef)
					clonePath := filepath.Join(cloneDir, repoName)

					fmt.Printf("  Cloning %s...\n", repoRef)
					if err := cloneRepo(repoRef, clonePath); err != nil {
						return fmt.Errorf("failed to clone %s: %w", repoRef, err)
					}

					absPath = clonePath
					isGit = true
				} else {
					// Local path
					var err error
					absPath, err = filepath.Abs(repoRef)
					if err != nil {
						return fmt.Errorf("invalid path %s: %w", repoRef, err)
					}

					// Check directory exists
					info, err := os.Stat(absPath)
					if err != nil {
						return fmt.Errorf("project not found: %s", absPath)
					}
					if !info.IsDir() {
						return fmt.Errorf("not a directory: %s", absPath)
					}

					repoName = filepath.Base(absPath)

					// Check if git repo (optional)
					gitDir := filepath.Join(absPath, ".git")
					if _, err := os.Stat(gitDir); err == nil {
						isGit = true
					} else {
						fmt.Printf("  Note: %s is not a git repository (git features disabled)\n", repoName)
					}
				}

				repoConfigs = append(repoConfigs, Repo{
					Name:  repoName,
					Path:  absPath,
					Port:  9801 + i,
					IsGit: isGit,
				})
			}

			// Generate auth secret
			secret := generateSecret()

			// Create config
			config := Config{
				Version:    "1.0.0",
				Project:    projectName,
				Repos:      repoConfigs,
				AuthSecret: secret,
				CreatedAt:  time.Now().Format(time.RFC3339),
			}

			// Create state directory structure
			dirs := []string{
				"tasks/active",
				"tasks/completed",
				"messages/inbox",
				"messages/archive",
				"checkpoints/pending",
				"checkpoints/resolved",
				"agents",
			}

			if err := os.RemoveAll(stateDir); err != nil && !force {
				return err
			}

			for _, dir := range dirs {
				path := filepath.Join(stateDir, dir)
				if err := os.MkdirAll(path, 0755); err != nil {
					return fmt.Errorf("failed to create %s: %w", dir, err)
				}
			}

			// Write config
			configData, _ := json.MarshalIndent(config, "", "  ")
			if err := os.WriteFile(filepath.Join(stateDir, "config.json"), configData, 0644); err != nil {
				return err
			}

			// Write auth secret
			if err := os.WriteFile(filepath.Join(stateDir, "auth.key"), []byte(secret), 0600); err != nil {
				return err
			}

			// Copy plugin files to each repo
			for _, repo := range repoConfigs {
				if err := setupRepoPlugins(repo.Path, secret); err != nil {
					fmt.Printf("Warning: Failed to setup plugins in %s: %v\n", repo.Name, err)
				}
			}

			fmt.Printf("✓ Workspace initialized: %s\n", projectName)
			fmt.Printf("  Projects:\n")
			for _, repo := range repoConfigs {
				gitStatus := ""
				if !repo.IsGit {
					gitStatus = " (no git)"
				}
				fmt.Printf("    - %s (port %d)%s\n", repo.Name, repo.Port, gitStatus)
			}
			fmt.Printf("  State: %s\n", stateDir)
			fmt.Printf("\nRun 'crush-orchestrator start' to begin.\n")

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repos, "repos", nil, "Project paths or git URLs (required)")
	cmd.MarkFlagRequired("repos")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing workspace")
	cmd.Flags().StringVar(&cloneDir, "clone-dir", "", "Directory to clone remote repos (default: .orchestrator/repos)")

	return cmd
}

// isRemoteURL checks if a string looks like a remote git URL
func isRemoteURL(s string) bool {
	return strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "ssh://")
}

// extractRepoName extracts the repository name from a URL
// e.g., git@github.com:org/repo.git -> repo
// e.g., https://github.com/org/repo.git -> repo
func extractRepoName(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Get last path component
	parts := strings.Split(url, "/")
	name := parts[len(parts)-1]

	// Handle SSH format (git@github.com:org/repo)
	if strings.Contains(name, ":") {
		parts = strings.Split(name, ":")
		name = parts[len(parts)-1]
	}

	// If still has org/repo format, take just the repo
	if strings.Contains(name, "/") {
		parts = strings.Split(name, "/")
		name = parts[len(parts)-1]
	}

	return name
}

// cloneRepo clones a remote repository to a local directory
func cloneRepo(url, dest string) error {
	// Check if already cloned
	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		fmt.Printf("    Already cloned: %s\n", dest)
		return nil
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	// Run git clone
	cmd := exec.Command("git", "clone", "--depth=1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func startCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the orchestration server and agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"

			// Load config
			config, err := loadConfig(stateDir)
			if err != nil {
				return fmt.Errorf("failed to load config. Run 'crush-orchestrator init' first: %w", err)
			}

			// Read auth secret
			secretBytes, err := os.ReadFile(filepath.Join(stateDir, "auth.key"))
			if err != nil {
				return fmt.Errorf("failed to read auth key: %w", err)
			}
			secret := string(secretBytes)

			// Initialize state manager
			stateManager, err := state.NewManager(stateDir)
			if err != nil {
				return fmt.Errorf("failed to initialize state: %w", err)
			}

			// Start HTTP API server
			apiServer := api.NewServer(stateManager, port, secret)
			if err := apiServer.Start(); err != nil {
				return fmt.Errorf("failed to start API server: %w", err)
			}
			fmt.Printf("✓ API server started on port %d\n", port)

			// Start process manager
			procManager := process.NewManager(9801)

			// Start OpenCode instances
			for _, repo := range config.Repos {
				fmt.Printf("  Starting OpenCode in %s (port %d)...\n", repo.Name, repo.Port)
				if err := procManager.Spawn(repo.Name, repo.Path, repo.Port); err != nil {
					fmt.Printf("  ⚠ Failed to start %s: %v\n", repo.Name, err)
				}
			}

			// Wait for processes to be ready
			fmt.Println("\nWaiting for agents to be ready...")
			time.Sleep(5 * time.Second)

			// Show status
			health := procManager.HealthCheck()
			for name, status := range health {
				symbol := "✓"
				if status != "healthy" {
					symbol = "✗"
				}
				fmt.Printf("  %s %s: %s\n", symbol, name, status)
			}

			// Write PID files
			procManager.WritePIDFiles(stateDir)

			fmt.Printf("\n✓ Orchestration started. API at http://localhost:%d\n", port)
			fmt.Println("Press Ctrl+C to stop.")

			// Wait for shutdown signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nShutting down...")
			procManager.StopAll()
			apiServer.Stop()
			procManager.CleanupPIDFiles(stateDir)
			fmt.Println("✓ Stopped.")

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 9800, "API server port")

	return cmd
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the orchestration server and agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"
			pidDir := filepath.Join(stateDir, "pids")

			entries, err := os.ReadDir(pidDir)
			if err != nil {
				return fmt.Errorf("not running (no PID files found)")
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				name := strings.TrimSuffix(entry.Name(), ".pid")
				pidData, err := os.ReadFile(filepath.Join(pidDir, entry.Name()))
				if err != nil {
					continue
				}

				var pid int
				fmt.Sscanf(string(pidData), "%d", &pid)

				if process, err := os.FindProcess(pid); err == nil {
					process.Signal(syscall.SIGTERM)
					fmt.Printf("  Sent SIGTERM to %s (PID %d)\n", name, pid)
				}
			}

			// Cleanup PID files
			os.RemoveAll(pidDir)
			fmt.Println("✓ Stopped.")

			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show orchestration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"

			// Check if running
			pidDir := filepath.Join(stateDir, "pids")
			if _, err := os.Stat(pidDir); os.IsNotExist(err) {
				fmt.Println("Not running. Run 'crush-orchestrator start' to begin.")
				return nil
			}

			// Load config
			config, err := loadConfig(stateDir)
			if err != nil {
				return err
			}

			// Load state summary
			summaryData, err := os.ReadFile(filepath.Join(stateDir, "state.json"))
			if err == nil {
				var summary state.Summary
				json.Unmarshal(summaryData, &summary)

				fmt.Printf("Project: %s\n", config.Project)
				fmt.Printf("Status: Running\n")
				fmt.Printf("\nRepositories:\n")
				for _, repo := range config.Repos {
					fmt.Printf("  - %s (port %d)\n", repo.Name, repo.Port)
				}
				fmt.Printf("\nStatistics:\n")
				fmt.Printf("  Active tasks: %d\n", summary.Stats.ActiveTasks)
				fmt.Printf("  Completed tasks: %d\n", summary.Stats.CompletedTasks)
				fmt.Printf("  Pending checkpoints: %d\n", summary.Stats.PendingCheckpoints)
				fmt.Printf("  Registered agents: %d\n", summary.Stats.TotalAgents)
			} else {
				fmt.Printf("Project: %s\n", config.Project)
				fmt.Printf("Status: Running (no state yet)\n")
			}

			return nil
		},
	}
}

// --- Helpers ---

func generateSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func loadConfig(stateDir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(stateDir, "config.json"))
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setupRepoPlugins(repoPath, secret string) error {
	opencodeDir := filepath.Join(repoPath, ".opencode")
	pluginsDir := filepath.Join(opencodeDir, "plugins")
	toolsDir := filepath.Join(opencodeDir, "tools")
	agentsDir := filepath.Join(opencodeDir, "agents")

	// Create directories
	for _, dir := range []string{pluginsDir, toolsDir, agentsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Write plugin file
	pluginContent := getPluginContent(secret)
	if err := os.WriteFile(filepath.Join(pluginsDir, "orchestration.ts"), []byte(pluginContent), 0644); err != nil {
		return err
	}

	// Write agent configs
	if err := writeAgentConfigs(agentsDir); err != nil {
		return err
	}

	return nil
}

func getPluginContent(secret string) string {
	return `// Orchestration Plugin
// Auto-generated by crush-orchestrator
import type { Plugin } from "@opencode-ai/plugin"

const API_URL = "http://localhost:9800"
const SECRET = "` + secret + `"

async function apiCall(path: string, method: string = "GET", body?: any) {
  const url = API_URL + path
  const res = await fetch(url, {
    method,
    headers: {
      "Authorization": "Bearer " + SECRET,
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) throw new Error("API error: " + res.status)
  return res.json()
}

export const OrchestrationPlugin: Plugin = async ({ project, client, $, directory }) => {
  return {
    "session.created": async () => {
      try {
        await apiCall("/health")
        await client.app.log({
          body: { service: "orchestration", level: "info", message: "Connected to orchestrator" },
        })
      } catch (e) {
        console.warn("Orchestrator not available, running in standalone mode")
      }
    },

    "session.idle": async () => {
      try {
        // Check for pending messages
        // This will be handled by custom tools
      } catch (e) {
        // Ignore errors
      }
    },
  }
}
`
}

func writeAgentConfigs(agentsDir string) error {
	configs := map[string]string{
		"tech-lead.md": `---
description: Coordinates multi-repo development, decomposes tasks, manages dependencies
mode: primary
permission:
  edit: ask
  bash:
    "*": ask
    "git status*": allow
    "git log*": allow
    "ls*": allow
---

You are a Tech Lead coordinating a multi-repository development effort.

## Responsibilities
1. Analyze the project scope and break it into tasks
2. Identify dependencies between repositories
3. Assign tasks to specialized agents (backend-dev, frontend-dev)
4. Monitor progress and resolve blockers
5. Ensure architectural consistency across repos

## Rules
- Never commit without a checkpoint
- Always check dependencies before starting work
- Document architectural decisions
`,
		"backend-dev.md": `---
description: Specialized backend development across microservices
mode: subagent
permission:
  edit: allow
  bash:
    "*": ask
    "git add*": allow
    "git commit*": ask
    "go test*": allow
    "npm test*": allow
---

You are a Backend Developer working on microservices.

## Capabilities
- API design and implementation
- Database schema changes
- Business logic implementation
- Unit and integration testing

## Rules
- Always check API contracts before changing interfaces
- Coordinate database changes with other services
- Include test coverage for new functionality
`,
		"frontend-dev.md": `---
description: Specialized frontend/UI development
mode: subagent
permission:
  edit: allow
  bash:
    "*": ask
    "npm test*": allow
    "npm run build*": allow
---

You are a Frontend Developer working on UI components.

## Rules
- Follow existing design system patterns
- Ensure accessibility (WCAG 2.1 AA)
- Coordinate API contracts with backend agents
`,
	}

	for name, content := range configs {
		path := filepath.Join(agentsDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}
