package manager

import (
	"log"

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

// RunSyncCycle performs the market sync and arbitrage calculation
func (h *Handler) RunSyncCycle() {
	log.Println("Syncing markets and calculating arbitrage...")
	// TODO: Implement Kalshi fetch and Arb logic here
}
