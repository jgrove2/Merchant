package manager

import (
	"log"
	"strconv"

	"backend/internal/kalshi"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB      *gorm.DB
	KClient *kalshi.Client
}

// NewHandler creates a new manager Handler instance
func NewHandler(database *gorm.DB, kClient *kalshi.Client) *Handler {
	return &Handler{
		DB:      database,
		KClient: kClient,
	}
}

// GetProviderBalance returns the balance for a specific provider
func (h *Handler) GetProviderBalance(c *gin.Context) {
	name := c.Param("name")
	if name == "kalshi" {
		if h.KClient == nil {
			c.JSON(503, gin.H{"error": "Kalshi client not configured"})
			return
		}
		bal, err := h.KClient.GetBalance()
		if err != nil {
			log.Println(err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"balance": bal})
		return
	}
	c.JSON(404, gin.H{"error": "Provider not supported"})
}

func (h *Handler) GetMarkets(c *gin.Context) {
	if h.KClient == nil {
		c.JSON(503, gin.H{"error": "Kalshi client not configured"})
		return
	}

	// Parse query parameters
	limit := 100 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Status defaults to "open" and only accepts "open"
	status := c.Query("status")
	if status == "" {
		status = "open"
	} else if status != "open" {
		c.JSON(400, gin.H{"error": "Only 'open' status is currently supported"})
		return
	}

	cursor := c.Query("cursor")

	response, err := h.KClient.GetMarkets(limit, cursor, status)
	if err != nil {
		log.Println("Error fetching markets:", err.Error())
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if response == nil || response.Markets == nil || len(response.Markets) == 0 {
		c.JSON(404, gin.H{"error": "No markets found"})
		return
	}

	c.JSON(200, gin.H{
		"markets": response.Markets,
		"cursor":  response.Cursor,
	})
}

// RunSyncCycle performs the market sync and arbitrage calculation
func (h *Handler) RunSyncCycle() {
	log.Println("Syncing markets and calculating arbitrage...")
	// TODO: Implement Kalshi fetch and Arb logic here
}
