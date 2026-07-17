package api

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhony08/agent-team-orchestration/pkg/state"
)

// Server is the HTTP API server
type Server struct {
	router     *gin.Engine
	state      *state.Manager
	port       int
	authSecret string
	startTime  time.Time
	httpServer *http.Server
	cpHandlers map[string]chan bool // checkpoint ID -> resolution channel
	cpMu       sync.Mutex
}

// NewServer creates a new API server
func NewServer(stateManager *state.Manager, port int, authSecret string) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		router:     router,
		state:      stateManager,
		port:       port,
		authSecret: authSecret,
		startTime:  time.Now(),
		cpHandlers: make(map[string]chan bool),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health endpoint (no auth)
	s.router.GET("/health", s.healthHandler)

	// API routes with auth
	api := s.router.Group("/api/v1", s.authMiddleware())
	{
		// Status
		api.GET("/status", s.statusHandler)

		// Tasks
		api.POST("/tasks", s.createTask)
		api.GET("/tasks", s.listTasks)
		api.GET("/tasks/:id", s.getTask)
		api.PATCH("/tasks/:id", s.updateTask)

		// Messages
		api.POST("/messages", s.sendMessage)
		api.GET("/messages/:agent_id", s.getMessages)

		// Checkpoints
		api.POST("/checkpoints", s.createCheckpoint)
		api.GET("/checkpoints", s.listCheckpoints)
		api.POST("/checkpoints/:id/approve", s.approveCheckpoint)
		api.POST("/checkpoints/:id/deny", s.denyCheckpoint)

		// Agents
		api.POST("/agents", s.registerAgent)
		api.GET("/agents", s.listAgents)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.port
}

// WaitForCheckpoint blocks until a checkpoint is resolved
func (s *Server) WaitForCheckpoint(cpID string, timeout time.Duration) (bool, string) {
	ch := make(chan bool, 1)
	reasonCh := make(chan string, 1)

	s.cpMu.Lock()
	s.cpHandlers[cpID] = ch
	s.cpMu.Unlock()

	defer func() {
		s.cpMu.Lock()
		delete(s.cpHandlers, cpID)
		s.cpMu.Unlock()
	}()

	select {
	case approved := <-ch:
		if !approved {
			return false, <-reasonCh
		}
		return true, ""
	case <-time.After(timeout):
		return false, "timeout"
	}
}

// --- Handlers ---

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"uptime":  time.Since(s.startTime).String(),
		"version": "1.0.0",
	})
}

func (s *Server) statusHandler(c *gin.Context) {
	summary, _ := s.state.GetSummary()
	agents, _ := s.state.ListAgents()

	c.JSON(http.StatusOK, gin.H{
		"running": true,
		"agents":  agents,
		"stats":   summary.Stats,
	})
}

func (s *Server) createTask(c *gin.Context) {
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

	if err := s.state.CreateTask(task); err != nil {
		if err.Error() == "circular dependency detected" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (s *Server) listTasks(c *gin.Context) {
	status := state.TaskStatus(c.Query("status"))
	assignee := c.Query("assignee")

	tasks, err := s.state.ListTasks(status, assignee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (s *Server) getTask(c *gin.Context) {
	id := c.Param("id")
	task, err := s.state.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) updateTask(c *gin.Context) {
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

	task, err := s.state.UpdateTask(id, update)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) sendMessage(c *gin.Context) {
	var req struct {
		From    string          `json:"from" binding:"required"`
		To      string          `json:"to" binding:"required"`
		Type    state.MessageType `json:"type" binding:"required"`
		Content string          `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Content) > 10240 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content too large (max 10KB)"})
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

	if err := s.state.SendMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

func (s *Server) getMessages(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	ack := c.Query("ack") == "true"

	messages, err := s.state.GetMessages(agentID, limit, ack)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (s *Server) createCheckpoint(c *gin.Context) {
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

	if err := s.state.CreateCheckpoint(cp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cp)
}

func (s *Server) listCheckpoints(c *gin.Context) {
	checkpoints, err := s.state.ListCheckpoints()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, checkpoints)
}

func (s *Server) approveCheckpoint(c *gin.Context) {
	id := c.Param("id")

	cp, err := s.state.ResolveCheckpoint(id, true, "")
	if err != nil {
		if err.Error() == "checkpoint already resolved" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Notify waiting handler
	s.cpMu.Lock()
	if ch, ok := s.cpHandlers[id]; ok {
		ch <- true
	}
	s.cpMu.Unlock()

	c.JSON(http.StatusOK, cp)
}

func (s *Server) denyCheckpoint(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	cp, err := s.state.ResolveCheckpoint(id, false, req.Reason)
	if err != nil {
		if err.Error() == "checkpoint already resolved" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Notify waiting handler
	s.cpMu.Lock()
	if ch, ok := s.cpHandlers[id]; ok {
		ch <- false
		reasonCh := make(chan string, 1)
		reasonCh <- req.Reason
	}
	s.cpMu.Unlock()

	c.JSON(http.StatusOK, cp)
}

func (s *Server) registerAgent(c *gin.Context) {
	var req struct {
		ID   string `json:"id" binding:"required"`
		Type string `json:"type" binding:"required"`
		Repo string `json:"repo" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent := &state.Agent{
		ID:     req.ID,
		Type:   req.Type,
		Repo:   req.Repo,
		Status: state.AgentStatusActive,
	}

	if err := s.state.RegisterAgent(agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, agent)
}

func (s *Server) listAgents(c *gin.Context) {
	agents, err := s.state.ListAgents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// --- Middleware ---

func (s *Server) authMiddleware() gin.HandlerFunc {
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
		if token != s.authSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
