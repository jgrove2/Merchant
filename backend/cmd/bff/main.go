package main

import (
	"log"
	"os"

	"backend/internal/bff"
	"backend/internal/config"
	"backend/internal/db"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Initialize Database Connection
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Seed database with providers
	config.SeedProviders(database)

	// 2. Initialize Handler
	managerURL := os.Getenv("MANAGER_URL")
	h := bff.NewHandler(database, managerURL)

	// 3. Setup Router
	r := gin.Default()

	// Enable CORS for frontend development
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 4. Routes
	api := r.Group("/api/v1")
	{
		api.GET("/balance", h.GetTotalBalance)
		api.GET("/markets", h.GetMarkets)
		api.GET("/markets/by-event", h.GetMarketsByEvent)
		api.GET("/events", h.GetEvents)
	}

	// 5. Start Server
	log.Println("BFF running on :8080")
	r.Run(":8080")
}
