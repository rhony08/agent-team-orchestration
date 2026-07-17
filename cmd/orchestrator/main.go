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
	Version       string `json:"version"`
	Project       string `json:"project"`
	Repos         []Repo `json:"repos"`
	AuthSecret    string `json:"-"`
	Port          int    `json:"port"`
	OpenCodePort  int    `json:"opencode_port"`
	CreatedAt     string `json:"created_at"`
}

type Repo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsGit bool   `json:"is_git"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "crush-orchestrator",
		Short: "Orchestrate multiple OpenCode agents across projects",
		Long: `Agent Team Orchestrator for OpenCode

Runs ONE headless OpenCode instance and creates multiple sessions
for each project. Users interact via CLI commands.

Workflow:
  crush-orchestrator init my-project --repos ./repo1,./repo2
  crush-orchestrator start
  crush-orchestrator send repo1 "implement JWT authentication"`,
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(sendCmd())
	rootCmd.AddCommand(taskCmd())
	rootCmd.AddCommand(checkpointCmd())
	rootCmd.AddCommand(projectCmd())
	rootCmd.AddCommand(sessionsCmd())
	rootCmd.AddCommand(messagesCmd())

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
		Args:  cobra.ExactArgs(1),
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
					IsGit: isGit,
				})
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

			registerProject(projectName, stateDir)

			fmt.Printf("✓ Initialized: %s\n", projectName)
			for _, r := range repoConfigs {
				tag := ""
				if !r.IsGit {
					tag = " (no git)"
				}
				fmt.Printf("  • %s%s\n", r.Name, tag)
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
	var projectName string
	var opencodePort int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start orchestration daemon",
		Long: `Starts the orchestration daemon and connects to existing OpenCode server.
Sessions are created for each project repository.

Make sure OpenCode is running before starting the daemon:
  opencode serve --port 4096`,
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

			config, err := loadConfig(stateDir)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			secretBytes, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))
			secret := string(secretBytes)

			if port == 9800 && config.Port != 0 {
				port = config.Port
			}

			if !headless.IsPortAvailable(port) {
				return fmt.Errorf("port %d is already in use. Is the daemon already running?", port)
			}

			stateManager, err := state.NewManager(stateDir)
			if err != nil {
				return err
			}

			// Connect to existing OpenCode server
			hm := headless.NewManager()

			// Get credentials from environment
			username := os.Getenv("OPENCODE_SERVER_USERNAME")
			password := os.Getenv("OPENCODE_SERVER_PASSWORD")

			if username != "" && password != "" {
				err = hm.ConnectWithCredentials(opencodePort, username, password)
			} else {
				err = hm.Connect(opencodePort)
			}

			if err != nil {
				return fmt.Errorf("failed to connect to OpenCode: %w\n\nMake sure OpenCode is running:\n  opencode serve --port %d", err, opencodePort)
			}

			// Create sessions for each repo
			fmt.Println("Creating sessions for each project...")
			for _, repo := range config.Repos {
				session, err := hm.CreateSession(repo.Name, repo.Path, "build")
				if err != nil {
					fmt.Printf("  ✗ %s: %v\n", repo.Name, err)
				} else {
					fmt.Printf("  ✓ %s (session: %s)\n", repo.Name, session.ID[:8])
				}
			}

			// Create daemon
			d := daemon.New(stateManager, port, secret, hm)
			if err := d.Start(); err != nil {
				return err
			}

			sessions := hm.ListSessions()

			fmt.Printf("\n╔══════════════════════════════════════════════════╗\n")
			fmt.Printf("║  ORCHESTRATION DAEMON                           \n")
			fmt.Printf("╠══════════════════════════════════════════════════╣\n")
			fmt.Printf("║  Daemon:   http://localhost:%d\n", port)
			fmt.Printf("║  OpenCode: http://localhost:%d (existing)\n", opencodePort)
			fmt.Printf("║  Project:  %s\n", config.Project)
			fmt.Printf("║                                                  \n")
			fmt.Printf("║  Sessions:\n")
			for _, s := range sessions {
				fmt.Printf("║    • %s (%s)\n", s.Name, s.ID[:8])
			}
			fmt.Printf("║                                                  \n")
			fmt.Printf("║  Commands:\n")
			fmt.Printf("║    crush-orchestrator send <repo> <message>\n")
			fmt.Printf("║    crush-orchestrator sessions\n")
			fmt.Printf("║    crush-orchestrator messages <repo>\n")
			fmt.Printf("║    crush-orchestrator task create ...\n")
			fmt.Printf("╚══════════════════════════════════════════════════╝\n")

			pidFile := filepath.Join(stateDir, "daemon.pid")
			os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

			fmt.Println("\nPress Ctrl+C to stop.")
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nShutting down...")
			d.Stop()
			os.Remove(pidFile)
			fmt.Println("✓ Stopped.")

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 9800, "Daemon port")
	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	cmd.Flags().IntVar(&opencodePort, "opencode-port", 4096, "OpenCode server port")

	return cmd
}

func stopCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon and OpenCode",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

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

	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func statusCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon and session status",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

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
				Running  bool `json:"running"`
				OpenCode bool `json:"opencode"`
				Sessions []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Agent  string `json:"agent"`
					Status string `json:"status"`
				} `json:"sessions"`
				Stats struct {
					ActiveTasks        int `json:"active_tasks"`
					CompletedTasks     int `json:"completed_tasks"`
					PendingCheckpoints int `json:"pending_checkpoints"`
				} `json:"stats"`
			}
			json.NewDecoder(resp.Body).Decode(&status)

			fmt.Printf("Project: %s\n", config.Project)
			fmt.Printf("Daemon: running\n")
			fmt.Printf("OpenCode: %v\n", map[bool]string{true: "running", false: "stopped"}[status.OpenCode])
			fmt.Printf("\nSessions:\n")
			for _, s := range status.Sessions {
				fmt.Printf("  • %s [%s] - %s\n", s.Name, s.Agent, s.ID[:8])
			}
			fmt.Printf("\nTasks: %d active, %d completed\n", status.Stats.ActiveTasks, status.Stats.CompletedTasks)
			fmt.Printf("Checkpoints: %d pending\n", status.Stats.PendingCheckpoints)
			return nil
		},
	}

	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func sendCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "send [session] [message]",
		Short: "Send a message to a session",
		Long: `Send a message to a project session.

Example:
  crush-orchestrator send todo-api "implement JWT authentication"
  crush-orchestrator send todo-db "create Prisma schema for users"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]
			message := args[1]

			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/send", config.Port)
			body, _ := json.Marshal(map[string]string{
				"session": sessionName,
				"message": message,
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

			fmt.Printf("Response from %s:\n%s\n", sessionName, result.Response)
			return nil
		},
	}

	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func sessionsCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/sessions", config.Port)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+string(secret))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			defer resp.Body.Close()

			var sessions []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Path   string `json:"path"`
				Agent  string `json:"agent"`
				Status string `json:"status"`
			}
			json.NewDecoder(resp.Body).Decode(&sessions)

			if len(sessions) == 0 {
				fmt.Println("No active sessions.")
				return nil
			}

			fmt.Printf("%-20s %-10s %-40s %s\n", "SESSION", "AGENT", "ID", "PATH")
			fmt.Printf("%-20s %-10s %-40s %s\n", "───────", "─────", "──", "────")
			for _, s := range sessions {
				fmt.Printf("%-20s %-10s %-40s %s\n", s.Name, s.Agent, s.ID, s.Path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func messagesCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "messages [session-name]",
		Short: "Show messages from a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]

			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

			config, _ := loadConfig(stateDir)
			secret, _ := os.ReadFile(filepath.Join(stateDir, "auth.key"))

			url := fmt.Sprintf("http://localhost:%d/api/v1/messages/%s", config.Port, sessionName)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+string(secret))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			defer resp.Body.Close()

			var messages []struct {
				ID      string    `json:"id"`
				Role    string    `json:"role"`
				Content string    `json:"content"`
				Time    time.Time `json:"time"`
				Error   string    `json:"error,omitempty"`
			}
			json.NewDecoder(resp.Body).Decode(&messages)

			if len(messages) == 0 {
				fmt.Println("No messages yet.")
				return nil
			}

			fmt.Printf("Messages for %s:\n\n", sessionName)
			for _, msg := range messages {
				if msg.Error != "" {
					fmt.Printf("[%s] ERROR: %s\n", msg.Role, msg.Error)
				} else if msg.Content != "" {
					fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func taskCmd() *cobra.Command {
	var projectName string

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

			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

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
	createCmd.Flags().String("assignee", "", "Assign to session")
	createCmd.Flags().String("priority", "medium", "Priority: critical, high, medium, low")
	createCmd.MarkFlagRequired("title")
	createCmd.MarkFlagRequired("description")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

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
	cmd.PersistentFlags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func checkpointCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Manage checkpoints",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pending checkpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := findProjectDir(projectName)
			if err != nil {
				return err
			}

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
			return resolveCheckpoint(args[0], "approve", "", projectName)
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
			return resolveCheckpoint(args[0], "deny", reason, projectName)
		},
	}

	cmd.AddCommand(listCmd, approveCmd, denyCmd)
	cmd.PersistentFlags().StringVar(&projectName, "project", "", "Project name")
	return cmd
}

func projectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage orchestration projects",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects := listProjects()

			if len(projects) == 0 {
				fmt.Println("No projects found.")
				fmt.Println("Run: crush-orchestrator init <name> --repos <paths>")
				return nil
			}

			fmt.Println("Projects:")
			for name, path := range projects {
				pidFile := filepath.Join(path, "daemon.pid")
				status := "stopped"
				if data, err := os.ReadFile(pidFile); err == nil {
					var pid int
					fmt.Sscanf(string(data), "%d", &pid)
					if process, err := os.FindProcess(pid); err == nil {
						if err := process.Signal(syscall.Signal(0)); err == nil {
							status = "running"
						}
					}
				}
				fmt.Printf("  • %s (%s) - %s\n", name, path, status)
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

func resolveCheckpoint(id, action, reason, projectName string) error {
	stateDir, err := findProjectDir(projectName)
	if err != nil {
		return err
	}

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

// --- Project Management ---

func getProjectsFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".crush-orchestrator", "projects.json")
}

func registerProject(name, path string) error {
	projectsFile := getProjectsFile()
	os.MkdirAll(filepath.Dir(projectsFile), 0755)

	projects := make(map[string]string)
	if data, err := os.ReadFile(projectsFile); err == nil {
		json.Unmarshal(data, &projects)
	}

	absPath, _ := filepath.Abs(path)
	projects[name] = absPath

	data, _ := json.MarshalIndent(projects, "", "  ")
	return os.WriteFile(projectsFile, data, 0644)
}

func listProjects() map[string]string {
	projectsFile := getProjectsFile()

	projects := make(map[string]string)
	if data, err := os.ReadFile(projectsFile); err == nil {
		json.Unmarshal(data, &projects)
	}

	return projects
}

func findProjectDir(projectName string) (string, error) {
	if projectName != "" {
		projects := listProjects()
		path, ok := projects[projectName]
		if !ok {
			return "", fmt.Errorf("project not found: %s", projectName)
		}
		return path, nil
	}

	if _, err := os.Stat(".orchestrator/config.json"); err == nil {
		return ".orchestrator", nil
	}

	return "", fmt.Errorf("no project found. Specify with --project or run from a project directory")
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
