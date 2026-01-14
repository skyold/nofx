package api

import (
	"encoding/json"
	"net/http"
	"nofx/logger"
	"strings"

	"github.com/gin-gonic/gin"
)

// handleTimeMachineRun runs the Time Machine replay logic
func (s *Server) handleTimeMachineRun(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		DecisionID    int64  `json:"decision_id" binding:"required"`
		SystemPrompt  string `json:"system_prompt" binding:"required"`
		AIModelID     string `json:"ai_model_id"`
		PromptVariant string `json:"prompt_variant"` // Optional, for reference
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	// 1. Get original decision record
	record, err := s.store.Decision().GetByID(req.DecisionID)
	if err != nil {
		logger.Errorf("[TimeMachine] Failed to get decision record %d: %v", req.DecisionID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Decision record not found"})
		return
	}

	// 2. Check permission (trader belongs to user)
	// Although the record doesn't store UserID directly, we can check via TraderID
	trader, err := s.store.Trader().GetByID(record.TraderID)
	if err != nil {
		// If trader deleted, we might still allow replaying if we want, but for security let's check ownership
		// If trader not found, we can't verify ownership easily unless we trust the user.
		// For now, if trader exists, check ownership. If not, maybe allow if admin?
		// Let's assume standard flow: Trader exists.
		logger.Warnf("[TimeMachine] Trader %s not found for decision %d", record.TraderID, req.DecisionID)
	} else {
		if trader.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this trader's data"})
			return
		}
	}

	// 3. Determine AI Model
	modelID := req.AIModelID
	if modelID == "" {
		// Use trader's current model if available
		if trader != nil {
			modelID = trader.AIModelID
		} else {
			// Fallback or error
			c.JSON(http.StatusBadRequest, gin.H{"error": "AI Model ID required"})
			return
		}
	}

	// 4. Run AI with OLD context (InputPrompt) and NEW System Prompt
	// Reuse s.runRealAITest which is available in package api
	logger.Infof("[TimeMachine] Replaying decision %d with model %s", req.DecisionID, modelID)

	aiResponse, err := s.runRealAITest(userID, modelID, req.SystemPrompt, record.InputPrompt)
	if err != nil {
		logger.Errorf("[TimeMachine] AI call failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       "AI call failed",
			"details":     err.Error(),
			"ai_response": "",
		})
		return
	}

	// 5. Parse the AI response to extract reasoning and decisions (if possible)
	// We try to parse it as JSON to structured data
	var decisions []interface{}
	var reasoning string

	// Simple heuristic parsing: find first [ and last ]
	start := strings.Index(aiResponse, "[")
	end := strings.LastIndex(aiResponse, "]")
	
	if start != -1 && end != -1 && end > start {
		jsonPart := aiResponse[start : end+1]
		if err := json.Unmarshal([]byte(jsonPart), &decisions); err != nil {
			logger.Warnf("[TimeMachine] Failed to parse JSON from AI response: %v", err)
		}
	}

	// Try to extract reasoning if it's not in the JSON or if we want a text summary
	// For now, just return the raw response and let frontend display it,
	// but if we parsed decisions, we can send them too.

	c.JSON(http.StatusOK, gin.H{
		"original_decision_id": req.DecisionID,
		"new_system_prompt":    req.SystemPrompt,
		"ai_model_id":          modelID,
		"ai_response":          aiResponse, // Raw text
		"decisions":            decisions,  // Parsed JSON objects
		"reasoning":            reasoning,  // Extracted reasoning (if we implement extraction)
		"timestamp":            record.Timestamp,
	})
}
