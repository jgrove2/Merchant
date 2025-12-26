package main

import (
	"log"

	"backend/internal/db"
	"backend/internal/embeddings"
	"encoding/json"
	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/mattn/go-sqlite3"
)

func main() {
	sqlite_vec.Auto()
	_ = sqlite3.SQLITE_DELETE

	log.Println("Connecting to DB...")
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	embService, err := embeddings.NewService()
	if err != nil {
		log.Fatalf("Failed to init embedding service: %v", err)
	}
	defer embService.Close()

	// 1. Manually embed all active markets
	var markets []db.Market
	database.Where("status = ?", "active").Find(&markets)
	log.Printf("Found %d active markets", len(markets))

	for i, m := range markets {
		// Log progress every 10 items
		if i%10 == 0 {
			log.Printf("Processing market %d/%d: %s", i+1, len(markets), m.Ticker)
		}

		embeddingText := m.Title + " " + m.Description + " " + m.Category
		vec, err := embService.Generate(embeddingText)
		if err != nil {
			log.Printf("Failed to generate embedding: %v", err)
			continue
		}

		vecBytes, _ := json.Marshal(vec)
		vecString := string(vecBytes)

		query := `INSERT OR REPLACE INTO vec_markets(id, embedding) VALUES (?, ?)`
		if err := database.Exec(query, m.ID, vecString).Error; err != nil {
			log.Printf("Failed to insert embedding for %s: %v", m.Ticker, err)
		}
	}
	log.Println("Finished backfilling embeddings.")
}
