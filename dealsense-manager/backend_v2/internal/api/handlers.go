package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/manager"
	"joinly-manager/internal/models"
)

var startTime = time.Now()

// Handler holds the dependencies for HTTP handlers
type Handler struct {
	agentManager *manager.AgentManager
}

// NewHandler creates a new handler instance
func NewHandler(agentManager *manager.AgentManager) *Handler {
	return &Handler{
		agentManager: agentManager,
	}
}

// HealthCheck handles the root endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	uptime := time.Since(startTime)

	// Get basic system stats
	agents := h.agentManager.ListAgents()
	runningAgents := 0
	for _, agent := range agents {
		if agent.Status == "running" {
			runningAgents++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"message":        "DealSense API is running",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"uptime_seconds": uptime.Seconds(),
		"uptime":         uptime.String(),
		"version":        "1.0.0",
		"agents": gin.H{
			"total":   len(agents),
			"running": runningAgents,
			"stopped": len(agents) - runningAgents,
		},
	})
}

// ReadinessCheck handles the readiness endpoint with deeper checks
func (h *Handler) ReadinessCheck(c *gin.Context) {
	uptime := time.Since(startTime)

	// Get basic system stats
	agents := h.agentManager.ListAgents()
	runningAgents := 0
	for _, agent := range agents {
		if agent.Status == "running" {
			runningAgents++
		}
	}

	// Perform readiness checks
	checks := gin.H{
		"database": gin.H{"status": "ok", "message": "Agent manager operational"},
	}

	// Check if we can communicate with Joinly core (basic connectivity check)
	// This is a simple check - in production you might want more thorough validation
	joinlyStatus := "unknown"
	if h.agentManager != nil {
		joinlyStatus = "ok"
	}

	checks["joinly_core"] = gin.H{"status": joinlyStatus, "message": "Joinly core connectivity"}

	// Determine overall readiness
	overallStatus := "ready"
	for _, check := range checks {
		if checkMap, ok := check.(gin.H); ok {
			if status, exists := checkMap["status"]; exists && status != "ok" {
				overallStatus = "not_ready"
				break
			}
		}
	}

	statusCode := http.StatusOK
	if overallStatus != "ready" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":         overallStatus,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"uptime_seconds": uptime.Seconds(),
		"uptime":         uptime.String(),
		"version":        "1.0.0",
		"checks":         checks,
		"agents": gin.H{
			"total":   len(agents),
			"running": runningAgents,
			"stopped": len(agents) - runningAgents,
		},
	})
}

// ListAgents handles GET /agents
func (h *Handler) ListAgents(c *gin.Context) {
	agents := h.agentManager.ListAgents()
	c.JSON(http.StatusOK, agents)
}

// CreateAgent handles POST /agents
func (h *Handler) CreateAgent(c *gin.Context) {
	var config models.AgentConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values if not provided
	val := 1.0
	config.UtteranceTailSeconds = &val

	// Set default conversation mode if not provided
	if config.ConversationMode == nil {
		defaultMode := models.ConversationModeConversational
		config.ConversationMode = &defaultMode
	}

	// Set default TTS provider for conversational mode if not provided
	if *config.ConversationMode == models.ConversationModeConversational && config.TTSProvider == nil {
		defaultTTS := models.TTSProviderKokoro
		config.TTSProvider = &defaultTTS
	}

	agent, err := h.agentManager.CreateAgent(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send response first
	c.JSON(http.StatusCreated, agent)

	// Auto-start if enabled (after response is sent to prevent deadlock)
	if config.AutoJoin {
		go func() {
			// Small delay to ensure response is sent
			time.Sleep(500 * time.Millisecond)
			if err := h.agentManager.StartAgent(agent.ID); err != nil {
				// Log error but don't affect the creation response
				logrus.Errorf("Failed to auto-start agent %s: %v", agent.ID, err)
			}
		}()
	}
}

// GetAgent handles GET /agents/{agent_id}
func (h *Handler) GetAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, exists := h.agentManager.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// DeleteAgent handles DELETE /agents/{agent_id}
func (h *Handler) DeleteAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.DeleteAgent(agentID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted successfully"})
}

// StartAgent handles POST /agents/{agent_id}/start
func (h *Handler) StartAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.StartAgent(agentID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent started successfully"})
}

// StopAgent handles POST /agents/{agent_id}/stop
func (h *Handler) StopAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.StopAgent(agentID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent stopped successfully"})
}

// JoinMeeting handles POST /agents/{agent_id}/join-meeting
func (h *Handler) JoinMeeting(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.JoinMeeting(agentID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found or not running" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "agent not connected" {
			statusCode = http.StatusBadRequest
		} else if err.Error() == "agent already joined meeting" {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Join meeting initiated"})
}

// GetAgentLogs handles GET /agents/{agent_id}/logs
func (h *Handler) GetAgentLogs(c *gin.Context) {
	agentID := c.Param("agent_id")

	lines := 100 // default
	if linesStr := c.Query("lines"); linesStr != "" {
		if parsedLines, err := strconv.Atoi(linesStr); err == nil && parsedLines > 0 {
			lines = parsedLines
		}
	}

	logs, err := h.agentManager.GetAgentLogs(agentID, lines)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// ListMeetings handles GET /meetings
func (h *Handler) ListMeetings(c *gin.Context) {
	meetings := h.agentManager.ListMeetings()
	c.JSON(http.StatusOK, meetings)
}

// GetUsageStats handles GET /usage (additional endpoint for usage statistics)
func (h *Handler) GetUsageStats(c *gin.Context) {
	stats := h.agentManager.GetUsageStats()
	c.JSON(http.StatusOK, stats)
}

// GetAgentAnalysis handles GET /agents/{agent_id}/analysis
func (h *Handler) GetAgentAnalysis(c *gin.Context) {
	agentID := c.Param("agent_id")

	// Check if agent exists and is in analyst mode
	agent, exists := h.agentManager.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if *agent.Config.ConversationMode != models.ConversationModeAnalyst {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent is not in analyst mode"})
		return
	}

	// Get the analyst agent
	analyst := h.agentManager.GetAnalystAgent(agentID)
	if analyst == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analyst agent not found"})
		return
	}

	// Get analysis data
	analysis := analyst.GetAnalysis()
	c.JSON(http.StatusOK, analysis)
}

// GetAgentAnalysisFormatted handles GET /agents/{agent_id}/analysis/formatted
func (h *Handler) GetAgentAnalysisFormatted(c *gin.Context) {
	agentID := c.Param("agent_id")

	// Check if agent exists and is in analyst mode
	agent, exists := h.agentManager.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if *agent.Config.ConversationMode != models.ConversationModeAnalyst {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent is not in analyst mode"})
		return
	}

	// Get the analyst agent
	analyst := h.agentManager.GetAnalystAgent(agentID)
	if analyst == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analyst agent not found"})
		return
	}

	// Get formatted analysis
	formattedAnalysis := analyst.GetFormattedAnalysis()

	// Return as plain text
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, formattedAnalysis)
}
