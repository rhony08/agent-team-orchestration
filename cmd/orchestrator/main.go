// cmd/orchestrator/main.go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/rhony08/agent-team-orchestration/pkg/daemon"
	"github.com/rhony08/agent-team-orchestration/pkg/headless"
	"github.com/rhony08/agent-team-orchestration/pkg/state"
)

var (
	version = "0.1.0"
	commit  = "dev"
)

type Config struct {
	Version    string `json:"version"`
	Project    string `json:"project"`
	Repos      []Repo `json:"repos"`
	AuthSecret string `json:"-"`
	Port       int    `json:"port"`
	CreatedAt  string `json:"created_at"`
}

type Repo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Port  int    `json:"port"`
	IsGit bool   `json:"is_git"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "crush-orchestrator",
		Short: "Orchestrate multiple OpenCode agents across projects",
		Long: `Agent Team Orchestrator for OpenCode

Runs OpenCode instances in headless mode and coordinates them through
a central daemon. Users interact via CLI commands.

Workflow:
  1. crush-orchestrator init my-project --repos ./repo1,./repo2
  2. crush-orchestrator start
  3. crush-orchestrator task create --title "Implement auth" --repo repo1
  4. crush-orchestrator send repo1 "implement JWT authentication"`,
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(sendCmd())
	rootCmd.AddCommand(taskCmd())
	rootCmd.AddCommand(checkpointCmd())

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
		Short: "Initialize orchestration workspace",
		Long: `Initialize a new orchestration workspace.

Projects can be local paths or remote git URLs:
  --repos ./repo1,./repo2
  --repos git@github.com:org/repo.git,./local-project`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			stateDir := ".orchestrator"

			if _, err := os.Stat(stateDir); err == nil && !force {
				return fmt.Errorf("workspace already exists. Use --force to overwrite")
			}

			if len(repos) == 0 {
				return fmt.Errorf("at least one project required (--repos)")
			}

			if cloneDir == "" {
				cloneDir = filepath.Join(stateDir, "repos")
			}

			var repoConfigs []Repo
			portOffset := 0
			for _, repoRef := range repos {
				var absPath, repoName string
				var isGit bool

				if isRemoteURL(repoRef) {
					repoName = extractRepoName(repoRef)
					clonePath := filepath.Join(cloneDir, repoName)
					fmt.Printf("  Cloning %s...\n", repoRef)
					if err := cloneRepo(repoRef, clonePath); err != nil {
						return fmt.Errorf("failed to clone: %w", err)
					}
					absPath = clonePath
					isGit = true
				} else {
					var err error
					absPath, err = filepath.Abs(repoRef)
					if err != nil {
						return fmt.Errorf("invalid path: %w", err)
					}
					info, err := os.Stat(absPath)
					if err != nil || !info.IsDir() {
						return fmt.Errorf("not a directory: %s", absPath)
					}
					repoName = filepath.Base(absPath)
					if _, err := os.Stat(filepath.Join(absPath, ".git")); err == nil {
						isGit = true
					}
				}

				repoConfigs = append(repoConfigs, Repo{
					Name:  repoName,
					Path:  absPath,
					Port:  9801 + portOffset,
					IsGit: isGit,
				})
				portOffset++
			}

			secret := generateSecret()

			config := Config{
				Version:    "1.0.0",
				Project:    projectName,
				Repos:      repoConfigs,
				AuthSecret: secret,
				Port:       9800,
				CreatedAt:  time.Now().Format(time.RFC3339),
			}

			// Create state directory
			dirs := []string{
				"tasks/active", "tasks/completed",
				"messages/inbox", "messages/archive",
				"checkpoints/pending", "checkpoints/resolved",
				"agents",
			}
			os.RemoveAll(stateDir)
			for _, dir := range dirs {
				os.MkdirAll(filepath.Join(stateDir, dir), 0755)
			}

			configData, _ := json.MarshalIndent(config, "", "  ")
			os.WriteFile(filepath.Join(stateDir, "config.json"), configData, 0644)
			os.WriteFile(filepath.Join(stateDir, "auth.key"), []byte(secret), 0600)

			// Setup .opencode in each repo
			for _, repo := range repoConfigs {
				setupRepoConfig(repo.Path, secret, config.Port)
			}

			fmt.Printf("✓ Initialized: %s\n", projectName)
			for _, r := range repoConfigs {
				tag := ""
				if !r.IsGit {
					tag = " (no git)"
				}
				fmt.Printf("  • %s (port %d)%s\n", r.Name, r.Port, tag)
			}
			fmt.Printf("\nRun: crush-orchestrator start\n")

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repos, "repos", nil, "Project paths or git URLs (required)")
	cmd.MarkFlagRequired("repos")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing workspace")
	cmd.Flags().StringVar(&cloneDir, "clone-dir", "", "Directory for cloned repos")

	return cmd
}

func startCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start orchestration daemon and headless OpenCode instances",
		Long: `Starts the orchestration daemon and spawns OpenCode instances
in headless mode for each configured project.

Each OpenCode instance runs as a background server (no TUI).
The daemon coordinates all instances and handles checkpoints.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"

			config, err := loadConfig(stateDir)
			if err != nil {
				return fmt.Errorf("run 'crush-orchestrator init' first: %w", err)
			}

			secretBytes, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))
			secret := string(secretBytes)

			// Start state manager
			stateManager, err := state.NewManager(stateDir)
			if err != nil {
				return err
			}

			// Start daemon
			d := daemon.New(stateManager, port, secret)
			if err := d.Start(); err != nil {
				return err
			}

			// Start headless OpenCode instances
			hm := headless.NewManager(9801)

			fmt.Printf("╔══════════════════════════════════════════════════╗\n")
			fmt.Printf("║  ORCHESTRATION DAEMON                           \n")
			fmt.Printf("╠══════════════════════════════════════════════════╣\n")
			fmt.Printf("║  Daemon: http://localhost:%d\n", port)
			fmt.Printf("║  Project: %s\n", config.Project)
			fmt.Printf("║                                                  \n")
			fmt.Printf("║  Starting headless instances...\n")

			for _, repo := range config.Repos {
				fmt.Printf("║    Starting %s on port %d...\n", repo.Name, repo.Port)
				if err := hm.Spawn(repo.Name, repo.Path, repo.Port); err != nil {
					fmt.Printf("║    ✗ Failed: %v\n", err)
				}
			}

			// Wait for instances to be ready
			fmt.Printf("║  Waiting for instances...\n")
			time.Sleep(5 * time.Second)

			instances := hm.ListInstances()
			for _, inst := range instances {
				symbol := "✓"
				if inst.Status != "running" {
					symbol = "✗"
				}
				fmt.Printf("║    %s %s: %s\n", symbol, inst.Name, inst.Status)
			}

			fmt.Printf("║                                                  \n")
			fmt.Printf("║  Ready! Use CLI commands to interact:            \n")
			fmt.Printf("║    crush-orchestrator send <repo> <message>      \n")
			fmt.Printf("║    crush-orchestrator task create ...            \n")
			fmt.Printf("║    crush-orchestrator checkpoint list            \n")
			fmt.Printf("║                                                  \n")
			fmt.Printf("║  Checkpoints will appear below.                  \n")
			fmt.Printf("╚══════════════════════════════════════════════════╝\n")

			// Write PID
			pidFile := filepath.Join(stateDir, "daemon.pid")
			os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

			// Wait for signal
			fmt.Println("\nPress Ctrl+C to stop.")
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nShutting down...")
			hm.StopAll()
			d.Stop()
			os.Remove(pidFile)
			fmt.Println("✓ Stopped.")

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 9800, "Daemon port")
	return cmd
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon and all instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"
			pidFile := filepath.Join(stateDir, "daemon.pid")

			data, err := os.ReadFile(pidFile)
			if err != nil {
				return fmt.Errorf("not running")
			}

			var pid int
			fmt.Sscanf(string(data), "%d", &pid)

			process, _ := os.FindProcess(pid)
			process.Signal(syscall.SIGTERM)
			os.Remove(pidFile)

			fmt.Printf("✓ Stopped (PID %d)\n", pid)
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon and instance status",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"
			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/status", config.Port)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+string(secret))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Daemon not running.")
				return nil
			}
			defer resp.Body.Close()

			var status struct {
				Running   bool `json:"running"`
				Instances []struct {
					Name   string `json:"name"`
					Status string `json:"status"`
				} `json:"instances"`
				Stats struct {
					ActiveTasks        int `json:"active_tasks"`
					CompletedTasks     int `json:"completed_tasks"`
					PendingCheckpoints int `json:"pending_checkpoints"`
				} `json:"stats"`
			}
			json.NewDecoder(resp.Body).Decode(&status)

			fmt.Println("Daemon: running")
			fmt.Printf("Tasks: %d active, %d completed\n", status.Stats.ActiveTasks, status.Stats.CompletedTasks)
			fmt.Printf("Checkpoints: %d pending\n", status.Stats.PendingCheckpoints)
			fmt.Println("\nInstances:")
			for _, inst := range status.Instances {
				fmt.Printf("  • %s: %s\n", inst.Name, inst.Status)
			}
			return nil
		},
	}
}

func sendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send [instance] [message]",
		Short: "Send a message to an OpenCode instance",
		Long: `Send a task or message to a headless OpenCode instance.

Example:
  crush-orchestrator send api-gateway "implement JWT authentication"
  crush-orchestrator send user-service "add email verification endpoint"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			message := args[1]

			stateDir := ".orchestrator"
			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			// Find instance port
			var port int
			for _, repo := range config.Repos {
				if repo.Name == instanceName {
					port = repo.Port
					break
				}
			}
			if port == 0 {
				return fmt.Errorf("instance not found: %s", instanceName)
			}

			// Send via daemon
			url := fmt.Sprintf("http://localhost:%d/api/v1/send", config.Port)
			body, _ := json.Marshal(map[string]string{
				"instance": instanceName,
				"message":  message,
			})

			req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
			req.Header.Set("Authorization", "Bearer "+string(secret))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			defer resp.Body.Close()

			var result struct {
				Response string `json:"response"`
				Error    string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&result)

			if result.Error != "" {
				return fmt.Errorf("%s", result.Error)
			}

			fmt.Printf("Response from %s:\n%s\n", instanceName, result.Response)
			return nil
		},
	}
}

func taskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			title, _ := cmd.Flags().GetString("title")
			desc, _ := cmd.Flags().GetString("description")
			assignee, _ := cmd.Flags().GetString("assignee")
			priority, _ := cmd.Flags().GetString("priority")

			stateDir := ".orchestrator"
			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/tasks", config.Port)
			body, _ := json.Marshal(map[string]string{
				"title":       title,
				"description": desc,
				"assignee":    assignee,
				"priority":    priority,
				"creator":     "user",
			})

			req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
			req.Header.Set("Authorization", "Bearer "+string(secret))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			defer resp.Body.Close()

			var task struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			}
			json.NewDecoder(resp.Body).Decode(&task)

			fmt.Printf("✓ Task created: %s - %s\n", task.ID, task.Title)
			return nil
		},
	}

	createCmd.Flags().String("title", "", "Task title")
	createCmd.Flags().String("description", "", "Task description")
	createCmd.Flags().String("assignee", "", "Assign to instance")
	createCmd.Flags().String("priority", "medium", "Priority: critical, high, medium, low")
	createCmd.MarkFlagRequired("title")
	createCmd.MarkFlagRequired("description")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"
			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/tasks", config.Port)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+string(secret))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			defer resp.Body.Close()

			var tasks []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Status   string `json:"status"`
				Assignee string `json:"assignee"`
			}
			json.NewDecoder(resp.Body).Decode(&tasks)

			if len(tasks) == 0 {
				fmt.Println("No tasks.")
				return nil
			}

			for _, t := range tasks {
				fmt.Printf("[%s] %s (%s) - %s\n", t.ID[:8], t.Title, t.Status, t.Assignee)
			}
			return nil
		},
	}

	cmd.AddCommand(createCmd, listCmd)
	return cmd
}

func checkpointCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Manage checkpoints",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pending checkpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := ".orchestrator"
			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/checkpoints", config.Port)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+string(secret))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			defer resp.Body.Close()

			var cps []struct {
				ID          string `json:"id"`
				Type        string `json:"type"`
				Description string `json:"description"`
				Requester   string `json:"requester"`
			}
			json.NewDecoder(resp.Body).Decode(&cps)

			if len(cps) == 0 {
				fmt.Println("No pending checkpoints.")
				return nil
			}

			fmt.Println("Pending checkpoints:")
			for _, cp := range cps {
				fmt.Printf("  [%s] %s - %s (from: %s)\n", cp.ID[:8], cp.Type, cp.Description, cp.Requester)
			}
			return nil
		},
	}

	approveCmd := &cobra.Command{
		Use:   "approve [id]",
		Short: "Approve a checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return resolveCheckpoint(args[0], "approve", "")
		},
	}

	denyCmd := &cobra.Command{
		Use:   "deny [id] [reason]",
		Short: "Deny a checkpoint",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reason := ""
			if len(args) > 1 {
				reason = args[1]
			}
			return resolveCheckpoint(args[0], "deny", reason)
		},
	}

	cmd.AddCommand(listCmd, approveCmd, denyCmd)
	return cmd
}

func resolveCheckpoint(id, action, reason string) error {
	stateDir := ".orchestrator"
	config, _ := loadConfig(stateDir)
	secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

	url := fmt.Sprintf("http://localhost:%d/api/v1/checkpoints/%s/%s", config.Port, id, action)

	var body string
	if reason != "" {
		body = fmt.Sprintf(`{"reason":"%s"}`, reason)
	}

	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+string(secret))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("daemon not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("✓ Checkpoint %s\n", action+"d")
	} else {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("%s", errResp.Error)
	}
	return nil
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
	json.Unmarshal(data, &config)
	return &config, nil
}

func isRemoteURL(s string) bool {
	return strings.HasPrefix(s, "git@") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}

func extractRepoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	name := parts[len(parts)-1]
	if strings.Contains(name, ":") {
		parts = strings.Split(name, ":")
		name = parts[len(parts)-1]
	}
	return name
}

func cloneRepo(url, dest string) error {
	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		return nil
	}
	os.MkdirAll(filepath.Dir(dest), 0755)
	cmd := exec.Command("git", "clone", "--depth=1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func setupRepoConfig(repoPath, secret string, daemonPort int) error {
	opencodeDir := filepath.Join(repoPath, ".opencode")
	for _, dir := range []string{"plugins", "tools", "agents"} {
		os.MkdirAll(filepath.Join(opencodeDir, dir), 0755)
	}

	// Plugin
	plugin := fmt.Sprintf(`// Orchestration Plugin
import type { Plugin } from "@opencode-ai/plugin"
export const OrchestrationPlugin: Plugin = async () => ({})
`)
	os.WriteFile(filepath.Join(opencodeDir, "plugins", "orchestration.ts"), []byte(plugin), 0644)

	return nil
}
