package db

import (
	"log"
	"os"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Connect now reads directly from the environment and loads the local vector extension
func Connect() (*gorm.DB, error) {
	// 1. Get the connection string from the .env (via os.Getenv)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback for local development if variable isn't set
		dsn = "../data/merchant.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	} else if strings.HasPrefix(dsn, "file:") {
		// Clean up if needed, though usually fine
	}

	// 2. Open the connection using standard gorm sqlite driver (CGO)
	// The vector extension is already registered globally via sqlite_vec.Auto() in main.go
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 3. Create the virtual table for embeddings if it doesn't exist
	// We use `vec0` from sqlite-vec.
	// `id` is an INTEGER PRIMARY KEY to map 1:1 with our `markets` table.
	// `embedding` is a float array of size 384 (for all-MiniLM-L6-v2).
	if err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS vec_markets USING vec0(
			embedding FLOAT[384]
		);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create virtual table vec_markets: %v", err)
	}

	// 4. Auto-migrate the schemas
	err = db.AutoMigrate(&Provider{}, &Market{}, &Event{}, &ArbitrageOpportunity{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
