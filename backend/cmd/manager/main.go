package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/db"
	"backend/internal/embeddings"
	"backend/internal/kalshi"
	"backend/internal/manager"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-sqlite3"
)

func main() {
	// Register sqlite-vec extension
	sqlite_vec.Auto()
	// Force registration of sqlite3 driver to ensure extensions are loaded
	_ = sqlite3.SQLITE_DELETE

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

	// 3. Initialize Embedding Service
	embService, err := embeddings.NewService()
	if err != nil {
		log.Printf("Warning: Failed to init embedding service: %v. Vector search will be disabled.", err)
		// We can proceed without it, just pass nil
	} else {
		defer embService.Close()
	}

	// 4. Initialize Handler
	h := manager.NewHandler(database, kClient, embService)

	// 5. Start Manager API
	go func() {
		r := gin.Default()
		r.GET("/providers/:name/balance", h.GetProviderBalance)
		r.GET("/markets", h.GetMarkets)
		r.GET("/markets/by-event", h.GetMarketsByEvent)
		r.GET("/events", h.GetEvents)
		r.GET("/events/:event_id", h.GetEvent)
		r.POST("/markets/search", h.SearchMarkets)

		log.Println("Manager API running on :8081")
		if err := r.Run(":8081"); err != nil {
			log.Fatalf("Manager API failed: %v", err)
		}
	}()

	// 6. Setup Context for graceful shutdown
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

	// 7. Execution Loop
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run initial sync on startup
	go h.RunSyncCycle()

	for {
		select {
		case <-ctx.Done():
			log.Println("Manager stopped.")
			return
		case <-ticker.C:
			go h.RunSyncCycle()
		}
	}
}
