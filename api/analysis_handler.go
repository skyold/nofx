package api

import (
	"net/http"
	"nofx/logger"
	"nofx/store"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalystProfileRequest Request structure for creating/updating analyst profile
type AnalystProfileRequest struct {
	Name         string `json:"name" binding:"required"`
	SystemPrompt string `json:"system_prompt" binding:"required"`
	ModelID      string `json:"model_id" binding:"required"`
	CronSchedule string `json:"cron_schedule"`
	IsEnabled    bool   `json:"is_enabled"`
}

// handleListAnalystProfiles Get all analyst profiles
func (s *Server) handleListAnalystProfiles(c *gin.Context) {
	profiles, err := s.store.AnalystProfile().GetAll()
	if err != nil {
		SafeInternalError(c, "Failed to list analyst profiles", err)
		return
	}
	c.JSON(http.StatusOK, profiles)
}

// handleGetAnalystProfile Get specific analyst profile
func (s *Server) handleGetAnalystProfile(c *gin.Context) {
	id := c.Param("id")
	profile, err := s.store.AnalystProfile().GetByID(id)
	if err != nil {
		SafeNotFound(c, "Analyst Profile")
		return
	}
	c.JSON(http.StatusOK, profile)
}

// handleCreateAnalystProfile Create new analyst profile
func (s *Server) handleCreateAnalystProfile(c *gin.Context) {
	var req AnalystProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	profile := &store.AnalystProfile{
		Name:         req.Name,
		SystemPrompt: req.SystemPrompt,
		ModelID:      req.ModelID,
		CronSchedule: req.CronSchedule,
		IsEnabled:    req.IsEnabled,
	}

	if err := s.store.AnalystProfile().Create(profile); err != nil {
		SafeInternalError(c, "Failed to create analyst profile", err)
		return
	}

	// Sync scheduler
	if s.scheduler != nil {
		s.scheduler.SyncJobs()
	}

	logger.Infof("✓ Analyst profile created: %s (%s)", profile.Name, profile.ID)
	c.JSON(http.StatusCreated, profile)
}

// handleUpdateAnalystProfile Update analyst profile
func (s *Server) handleUpdateAnalystProfile(c *gin.Context) {
	id := c.Param("id")
	var req AnalystProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	// Check if exists
	_, err := s.store.AnalystProfile().GetByID(id)
	if err != nil {
		SafeNotFound(c, "Analyst Profile")
		return
	}

	profile := &store.AnalystProfile{
		ID:           id,
		Name:         req.Name,
		SystemPrompt: req.SystemPrompt,
		ModelID:      req.ModelID,
		CronSchedule: req.CronSchedule,
		IsEnabled:    req.IsEnabled,
		UpdatedAt:    time.Now(),
	}

	if err := s.store.AnalystProfile().Update(profile); err != nil {
		SafeInternalError(c, "Failed to update analyst profile", err)
		return
	}

	// Sync scheduler
	if s.scheduler != nil {
		s.scheduler.SyncJobs()
	}

	logger.Infof("✓ Analyst profile updated: %s", profile.Name)
	c.JSON(http.StatusOK, profile)
}

// handleDeleteAnalystProfile Delete analyst profile
func (s *Server) handleDeleteAnalystProfile(c *gin.Context) {
	id := c.Param("id")
	if err := s.store.AnalystProfile().Delete(id); err != nil {
		SafeInternalError(c, "Failed to delete analyst profile", err)
		return
	}

	// Sync scheduler
	if s.scheduler != nil {
		s.scheduler.SyncJobs()
	}

	logger.Infof("✓ Analyst profile deleted: %s", id)
	c.JSON(http.StatusOK, gin.H{"message": "Analyst profile deleted"})
}

// handleTriggerAnalysis Manually trigger an analysis for a specific profile
func (s *Server) handleTriggerAnalysis(c *gin.Context) {
	id := c.Param("id")
	if s.scheduler == nil {
		SafeInternalError(c, "Scheduler not initialized", nil)
		return
	}

	if err := s.scheduler.TriggerManually(id); err != nil {
		if err.Error() == "record not found" {
			SafeNotFound(c, "Analyst Profile")
		} else {
			SafeInternalError(c, "Failed to trigger analysis", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Analysis triggered successfully"})
}

// handleListAnalysisSessions List analysis sessions
func (s *Server) handleListAnalysisSessions(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	sessions, err := s.store.Analysis().GetLatestSessions(limit)
	if err != nil {
		SafeInternalError(c, "Failed to list analysis sessions", err)
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// handleGetAnalysisSession Get detailed analysis session with records
func (s *Server) handleGetAnalysisSession(c *gin.Context) {
	id := c.Param("id")
	// ID is string (UUID)
	session, err := s.store.Analysis().GetSession(id)
	if err != nil {
		SafeNotFound(c, "Analysis Session")
		return
	}

	// Session already contains records
	c.JSON(http.StatusOK, session)
}
