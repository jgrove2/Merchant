package main

import (
	"log"
	"strconv"

	"backend/internal/db"
	"backend/internal/embeddings"
)

func main() {
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

		vecString := pgVectorString(vec)

		// Postgres uses ON CONFLICT for replace behavior if primary key matches
		// Here we assume rowid is the primary key
		query := `
			INSERT INTO vec_markets(rowid, embedding) 
			VALUES (?, ?)
			ON CONFLICT(rowid) DO UPDATE SET embedding = EXCLUDED.embedding
		`
		if err := database.Exec(query, m.ID, vecString).Error; err != nil {
			log.Printf("Failed to insert embedding for %s: %v", m.Ticker, err)
		}
	}
	log.Println("Finished backfilling embeddings.")
}

// pgVectorString helper to convert []float32 to string format '[1.0, 2.0]'
func pgVectorString(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	var b []byte
	b = append(b, '[')
	for i, v := range vec {
		if i > 0 {
			b = append(b, ',')
		}
		b = strconv.AppendFloat(b, float64(v), 'f', -1, 32)
	}
	b = append(b, ']')
	return string(b)
}
