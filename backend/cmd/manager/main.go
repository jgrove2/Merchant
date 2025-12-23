package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/db"
	"backend/internal/kalshi"
	"backend/internal/manager"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting Manager Service...")

	// 1. Initialize DB
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Could not connect to DB: %v", err)
	}

	// 2. Initialize Kalshi Client
	kClient, err := kalshi.NewClient(
		os.Getenv("KALSHI_BASE_URL"),
		os.Getenv("KALSHI_API_KEY"),
		os.Getenv("KALSHI_KEY_PATH"),
	)
	if err != nil {
		log.Printf("Warning: Failed to init Kalshi client: %v", err)
	}

	// 3. Initialize Handler
	h := manager.NewHandler(database, kClient)

	// 4. Start Manager API
	go func() {
		r := gin.Default()
		r.GET("/providers/:name/balance", h.GetProviderBalance)
		r.GET("/markets", h.GetMarkets)

		log.Println("Manager API running on :8081")
		if err := r.Run(":8081"); err != nil {
			log.Fatalf("Manager API failed: %v", err)
		}
	}()

	// 5. Setup Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down Manager...")
		cancel()
	}()

	// 6. Execution Loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Manager stopped.")
			return
		case <-ticker.C:
			h.RunSyncCycle()
		}
	}
}
