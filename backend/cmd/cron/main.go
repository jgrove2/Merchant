package main

import (
	"log"
	"os"
	"strings"
	"time"

	"backend/internal/db"
	"backend/internal/embeddings"
	"backend/internal/kalshi"
	"backend/internal/llm"
	"backend/internal/sync"
)

func main() {
	log.Println("Starting Cron Service...")

	// Init DB
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("DB Connect error: %v", err)
	}

	// Init Kalshi
	kClient, err := kalshi.NewClient(
		os.Getenv("KALSHI_BASE_URL"),
		os.Getenv("KALSHI_API_KEY"),
		os.Getenv("KALSHI_KEY_PATH"),
	)
	if err != nil {
		log.Printf("Kalshi init warning: %v", err)
	}

	// Init Embeddings
	embService, err := embeddings.NewService()
	if err != nil {
		log.Printf("Embedding init warning: %v", err)
	} else {
		defer embService.Close()
	}

	// Init LLM
	modelName := os.Getenv("SLM_MODEL")
	if modelName == "" {
		modelName = "qwen2.5:14b"
	}
	llmCfg := llm.Config{
		Provider: llm.ProviderGroq,
		Model:    modelName,
		APIKey:   os.Getenv("GROQ_API_KEY"),
	}
	llmManager := llm.NewManager(llmCfg)

	// Init Syncer
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
	log.Printf("Syncing categories: %v", categories)

	syncer := sync.NewSyncer(database, kClient, embService, llmManager, categories)

	// Define Tickers
	eventTicker := time.NewTicker(12 * time.Hour)
	embeddingTicker := time.NewTicker(6 * time.Hour)
	analysisTicker := time.NewTicker(6 * time.Hour)

	// Initial Run
	go func() {
		log.Println("Running initial Expired Embedding Prune...")
		syncer.PruneExpiredEmbeddings()
		log.Println("Running initial Event Sync...")
		syncer.SyncKalshiEvents()
	}()
	go func() {
		// Sleep a bit to let events sync potentially start
		time.Sleep(1 * time.Minute)
		log.Println("Running initial Embedding Sync...")
		syncer.SyncEmbeddings()
	}()
	go func() {
		time.Sleep(2 * time.Minute)
		log.Println("Running initial Analysis...")
		syncer.AnalyzeRelatedMarkets()
	}()

	// Loop
	for {
		select {
		case <-eventTicker.C:
			go func() {
				syncer.PruneExpiredEmbeddings()
				syncer.SyncKalshiEvents()
			}()
		case <-embeddingTicker.C:
			go syncer.SyncEmbeddings()
		case <-analysisTicker.C:
			go syncer.AnalyzeRelatedMarkets()
		}
	}
}
