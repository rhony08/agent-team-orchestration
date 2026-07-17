package daemon

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhony08/agent-team-orchestration/pkg/headless"
	"github.com/rhony08/agent-team-orchestration/pkg/state"
)

// Daemon is the background orchestration server
type Daemon struct {
	router     *gin.Engine
	state      *state.Manager
	port       int
	authSecret string
	startTime  time.Time
	httpServer *http.Server
	headless   *headless.Manager

	// Checkpoint resolution channels
	cpHandlers map[string]chan CheckpointResult
	cpMu       sync.Mutex
}

// CheckpointResult holds the user's decision
type CheckpointResult struct {
	Approved bool
	Reason   string
}

// New creates a new daemon
func New(stateManager *state.Manager, port int, authSecret string, headlessMgr *headless.Manager) *Daemon {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	d := &Daemon{
		router:     router,
		state:      stateManager,
		port:       port,
		authSecret: authSecret,
		startTime:  time.Now(),
		headless:   headlessMgr,
		cpHandlers: make(map[string]chan CheckpointResult),
	}

	d.setupRoutes()
	return d
}

func (d *Daemon) setupRoutes() {
	// Health (no auth)
	d.router.GET("/health", d.healthHandler)

	// Sessions
	d.router.POST("/api/v1/sessions", d.authMiddleware(), d.createSession)
	d.router.GET("/api/v1/sessions", d.authMiddleware(), d.listSessions)

	// Send message to session
	d.router.POST("/api/v1/send", d.authMiddleware(), d.sendToSession)

	// Tasks
	d.router.POST("/api/v1/tasks", d.authMiddleware(), d.createTask)
	d.router.GET("/api/v1/tasks", d.authMiddleware(), d.listTasks)
	d.router.GET("/api/v1/tasks/:id", d.authMiddleware(), d.getTask)
	d.router.PATCH("/api/v1/tasks/:id", d.authMiddleware(), d.updateTask)

	// Messages
	d.router.POST("/api/v1/messages", d.authMiddleware(), d.sendMessage)
	d.router.GET("/api/v1/messages/:agent", d.authMiddleware(), d.getMessages)

	// Checkpoints
	d.router.POST("/api/v1/checkpoints", d.authMiddleware(), d.createCheckpoint)
	d.router.GET("/api/v1/checkpoints", d.authMiddleware(), d.listCheckpoints)
	d.router.POST("/api/v1/checkpoints/:id/approve", d.authMiddleware(), d.approveCheckpoint)
	d.router.POST("/api/v1/checkpoints/:id/deny", d.authMiddleware(), d.denyCheckpoint)

	// Status
	d.router.GET("/api/v1/status", d.authMiddleware(), d.statusHandler)
}

// Start starts the daemon
func (d *Daemon) Start() error {
	d.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.port),
		Handler: d.router,
	}

	go func() {
		if err := d.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Daemon error: %v", err)
		}
	}()

	return nil
}

// Stop stops the daemon
func (d *Daemon) Stop() {
	if d.httpServer != nil {
		d.httpServer.Close()
	}
}

// --- Handlers ---

func (d *Daemon) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"uptime":   time.Since(d.startTime).String(),
		"version":  "1.0.0",
		"sessions": len(d.headless.ListSessions()),
	})
}

func (d *Daemon) createSession(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Path  string `json:"path" binding:"required"`
		Agent string `json:"agent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Agent == "" {
		req.Agent = "build"
	}

	session, err := d.headless.CreateSession(req.Name, req.Path, req.Agent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (d *Daemon) listSessions(c *gin.Context) {
	sessions := d.headless.ListSessions()
	c.JSON(http.StatusOK, sessions)
}

func (d *Daemon) sendToSession(c *gin.Context) {
	var req struct {
		Session string `json:"session" binding:"required"`
		Message string `json:"message" binding:"required"`
		Model   string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := d.headless.SendPrompt(req.Session, req.Message, req.Model)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":  req.Session,
		"response": response,
	})
}

func (d *Daemon) statusHandler(c *gin.Context) {
	summary, _ := d.state.GetSummary()

	sessions := d.headless.ListSessions()
	sessionStatuses := make([]gin.H, 0)
	for _, s := range sessions {
		sessionStatuses = append(sessionStatuses, gin.H{
			"id":     s.ID,
			"name":   s.Name,
			"agent":  s.Agent,
			"status": s.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"running":   true,
		"uptime":    time.Since(d.startTime).String(),
		"opencode":  d.headless.IsRunning(),
		"sessions":  sessionStatuses,
		"stats":     summary.Stats,
	})
}

func (d *Daemon) createTask(c *gin.Context) {
	var req struct {
		Title        string             `json:"title" binding:"required"`
		Description  string             `json:"description" binding:"required"`
		Type         state.TaskType     `json:"type"`
		Priority     state.TaskPriority `json:"priority"`
		Assignee     string             `json:"assignee"`
		Creator      string             `json:"creator" binding:"required"`
		Dependencies []state.Dependency `json:"dependencies"`
		Repo         string             `json:"repo"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := &state.Task{
		ID:           uuid.New().String(),
		Title:        req.Title,
		Description:  req.Description,
		Type:         req.Type,
		Priority:     req.Priority,
		Assignee:     req.Assignee,
		Creator:      req.Creator,
		Dependencies: req.Dependencies,
		Repo:         req.Repo,
		Status:       state.TaskStatusPending,
		CreatedAt:    time.Now(),
	}

	if err := d.state.CreateTask(task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (d *Daemon) listTasks(c *gin.Context) {
	status := state.TaskStatus(c.Query("status"))
	assignee := c.Query("assignee")

	tasks, err := d.state.ListTasks(status, assignee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (d *Daemon) getTask(c *gin.Context) {
	id := c.Param("id")
	task, err := d.state.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (d *Daemon) updateTask(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status   *state.TaskStatus   `json:"status"`
		Assignee *string             `json:"assignee"`
		Priority *state.TaskPriority `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := state.TaskUpdate{
		Status:   req.Status,
		Assignee: req.Assignee,
		Priority: req.Priority,
	}

	task, err := d.state.UpdateTask(id, update)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (d *Daemon) sendMessage(c *gin.Context) {
	var req struct {
		From    string            `json:"from" binding:"required"`
		To      string            `json:"to" binding:"required"`
		Type    state.MessageType `json:"type" binding:"required"`
		Content string            `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := &state.Message{
		ID:        uuid.New().String(),
		From:      req.From,
		To:        req.To,
		Type:      req.Type,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	if err := d.state.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

func (d *Daemon) getMessages(c *gin.Context) {
	agentID := c.Param("agent")
	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	ack := c.Query("ack") == "true"

	messages, err := d.state.GetMessages(agentID, limit, ack)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (d *Daemon) createCheckpoint(c *gin.Context) {
	var req struct {
		Type          state.CheckpointType `json:"type" binding:"required"`
		Description   string               `json:"description" binding:"required"`
		Requester     string               `json:"requester" binding:"required"`
		AffectedRepos []string             `json:"affected_repos"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cp := &state.Checkpoint{
		ID:            uuid.New().String(),
		Type:          req.Type,
		Description:   req.Description,
		Requester:     req.Requester,
		AffectedRepos: req.AffectedRepos,
		Status:        state.CheckpointStatusPending,
		CreatedAt:     time.Now(),
	}

	if err := d.state.CreateCheckpoint(cp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Display checkpoint in terminal
	fmt.Printf("\n╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║  CHECKPOINT REQUEST                             \n")
	fmt.Printf("╠══════════════════════════════════════════════════╣\n")
	fmt.Printf("║  From: %-42s\n", req.Requester)
	fmt.Printf("║  Type: %-42s\n", req.Type)
	fmt.Printf("║  %s\n", req.Description)
	fmt.Printf("║                                                  \n")
	fmt.Printf("║  Approve: crush-orchestrator checkpoint approve %s\n", cp.ID[:8])
	fmt.Printf("║  Deny:    crush-orchestrator checkpoint deny %s [reason]\n", cp.ID[:8])
	fmt.Printf("╚══════════════════════════════════════════════════╝\n")

	c.JSON(http.StatusCreated, cp)
}

func (d *Daemon) listCheckpoints(c *gin.Context) {
	checkpoints, err := d.state.ListCheckpoints()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, checkpoints)
}

func (d *Daemon) approveCheckpoint(c *gin.Context) {
	id := c.Param("id")

	fullID := d.findCheckpointID(id)
	if fullID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Checkpoint not found"})
		return
	}

	cp, err := d.state.ResolveCheckpoint(fullID, true, "")
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	d.cpMu.Lock()
	if ch, ok := d.cpHandlers[fullID]; ok {
		ch <- CheckpointResult{Approved: true}
	}
	d.cpMu.Unlock()

	fmt.Printf("✓ Checkpoint approved: %s\n", cp.Description)

	c.JSON(http.StatusOK, cp)
}

func (d *Daemon) denyCheckpoint(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	fullID := d.findCheckpointID(id)
	if fullID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Checkpoint not found"})
		return
	}

	cp, err := d.state.ResolveCheckpoint(fullID, false, req.Reason)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	d.cpMu.Lock()
	if ch, ok := d.cpHandlers[fullID]; ok {
		ch <- CheckpointResult{Approved: false, Reason: req.Reason}
	}
	d.cpMu.Unlock()

	fmt.Printf("✗ Checkpoint denied: %s\n", cp.Description)

	c.JSON(http.StatusOK, cp)
}

func (d *Daemon) findCheckpointID(prefix string) string {
	checkpoints, _ := d.state.ListCheckpoints()
	for _, cp := range checkpoints {
		if len(cp.ID) >= 8 && cp.ID[:8] == prefix {
			return cp.ID
		}
	}
	return prefix
}

// --- Middleware ---

func (d *Daemon) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		if len(auth) < 8 || auth[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token := auth[7:]
		if token != d.authSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
