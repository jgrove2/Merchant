package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"backend/internal/db"
	"backend/internal/embeddings"
	"backend/internal/kalshi"
	"backend/internal/llm"
	"backend/internal/sync"

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

	// 3. Initialize Embedding Service
	embService, err := embeddings.NewService()
	if err != nil {
		log.Printf("Warning: Failed to init embedding service: %v. Vector search will be disabled.", err)
		// We can proceed without it, just pass nil
	} else {
		defer embService.Close()
	}

	// 4. Initialize Redis
	// Redis is deprecated and removed.
	// _, err = db.NewRedis(os.Getenv("REDIS_URL"))
	// if err != nil {
	// 	log.Printf("Warning: Failed to init Redis: %v. Caching will be disabled.", err)
	// }

	// 5. Initialize LLM Service
	// Use SLM_MODEL env var, default to qwen3:14b if not set
	modelName := os.Getenv("SLM_MODEL")
	if modelName == "" {
		modelName = "qwen3:14b"
	}
	llmCfg := llm.Config{
		Provider: llm.ProviderGroq,
		Model:    modelName,
		APIKey:   os.Getenv("GROQ_API_KEY"),
	}
	llmManager := llm.NewManager(llmCfg)
	// llmManager can now act as the service provider if we adjust how it's used or just pass it where needed
	// The original code tried to get a Service interface from Connect().
	// Let's assume for now we want the manager itself if it has the CompareMarkets method,
	// OR we need to update the syncer to accept the manager.
	// However, looking at the previous code, LLM manager has Connect() returning Service.
	// But CompareMarkets is defined on *Manager in prompt.go.
	// This suggests *Manager IS the service we want to pass around if we want CompareMarkets.
	// Let's check prompt.go again. Yes, func (m *Manager) CompareMarkets.

	// So we pass llmManager to the syncer.

	// 6. Initialize Syncer
	// Parse SYNC_CATEGORIES from environment (comma-separated list)
	categoriesEnv := os.Getenv("SYNC_CATEGORIES")
	if categoriesEnv == "" {
		log.Fatal("SYNC_CATEGORIES environment variable is required. Set it to a comma-separated list of categories (e.g., SOCIAL,POLITICS,SPORTS)")
	}
	var categories []string
	for _, cat := range strings.Split(categoriesEnv, ",") {
		trimmed := strings.TrimSpace(cat)
		if trimmed != "" {
			categories = append(categories, trimmed)
		}
	}
	if len(categories) == 0 {
		log.Fatal("SYNC_CATEGORIES must contain at least one valid category")
	}
	log.Printf("Configured categories: %v", categories)

	syncer := sync.NewSyncer(database, kClient, embService, llmManager, categories)

	// 7. Initialize Handler
	h := NewHandler(database, kClient, embService, syncer)

	// 8. Start Manager API
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

	// 9. Setup Context for graceful shutdown
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

	// 10. Execution Loop
	// Wait for interrupt signal to gracefully shutdown the server
	<-ctx.Done()
	log.Println("Manager stopped.")
}
