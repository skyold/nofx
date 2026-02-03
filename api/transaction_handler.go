package api

import (
	"net/http"
	"nofx/store"
	"strconv"

	"github.com/gin-gonic/gin"
)

// handleGetTransactions Get transaction list
func (s *Server) handleGetTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	exchangeID := c.Query("exchange_id")
	traderID := c.Query("trader_id")
	sortOrder := c.DefaultQuery("sort", "desc")

	transactions, total, err := s.store.Order().GetTransactions(page, pageSize, exchangeID, traderID, sortOrder)
	if err != nil {
		SafeInternalError(c, "Get transactions", err)
		return
	}

	// Ensure transactions is not nil (return empty array instead of null)
	if transactions == nil {
		transactions = []*store.TraderFill{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     transactions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// handleAssignTransaction Assign transaction to trader
func (s *Server) handleAssignTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		SafeBadRequest(c, "Invalid transaction ID")
		return
	}

	var req struct {
		TraderID string `json:"trader_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	err = s.store.Order().UpdateTransactionTrader(id, req.TraderID)
	if err != nil {
		SafeInternalError(c, "Assign transaction", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction assigned successfully"})
}

// handleClearHistory Clear trader history (Reset)
func (s *Server) handleClearHistory(c *gin.Context) {
	traderID := c.Param("id")
	userID := c.GetString("user_id")

	// Verify ownership
	if _, err := s.store.Trader().GetFullConfig(userID, traderID); err != nil {
		SafeNotFound(c, "Trader")
		return
	}

	if err := s.store.Order().ClearHistory(traderID); err != nil {
		SafeInternalError(c, "Clear history", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "History cleared successfully"})
}
