package bff

import (
	"encoding/json"
	"log"
	"net/http"

	"backend/internal/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB         *gorm.DB
	ManagerURL string
}

// NewHandler creates a new Handler instance
func NewHandler(database *gorm.DB, managerURL string) *Handler {
	return &Handler{
		DB:         database,
		ManagerURL: managerURL,
	}
}

// GetOpportunities returns all active arb trades sorted by yield
func (h *Handler) GetOpportunities(c *gin.Context) {
	var opportunities []db.ArbitrageOpportunity

	// Preload the Market relationship to show Tickers/Titles in the UI
	result := h.DB.Preload("Market").
		Order("expected_yield DESC").
		Find(&opportunities)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, opportunities)
}

// GetProviders shows which exchanges we are currently monitoring
func (h *Handler) GetProviders(c *gin.Context) {
	var providers []db.Provider
	h.DB.Find(&providers)
	c.JSON(http.StatusOK, providers)
}

// GetTotalBalance aggregates balances from all providers
func (h *Handler) GetTotalBalance(c *gin.Context) {
	// For now, we only query Kalshi from the manager
	// In the future, this would aggregate from multiple provider endpoints

	log.Println("Fetching total balance from manager:", h.ManagerURL)
	resp, err := http.Get(h.ManagerURL + "/providers/kalshi/balance")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to contact manager"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to get balance from manager"})
		return
	}

	var data struct {
		Balance int64 `json:"balance"`
	}
	// Use json decoder
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_balance": data.Balance, // Returns in cents
		"currency":      "USD",
		"breakdown": gin.H{
			"kalshi": data.Balance,
		},
	})
}

// ToggleTrading is a placeholder for your moratorium call
func (h *Handler) ToggleTrading(c *gin.Context) {
	var input struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Logic to store a global "Pause" flag in DB or Redis
	// The Manager and Trader services should respect this flag
	c.JSON(http.StatusOK, gin.H{"trading_active": input.Active})
}
