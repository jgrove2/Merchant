package db

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect now reads directly from the environment and uses Postgres
func Connect() (*gorm.DB, error) {
	// 1. Get the connection string from the .env (via os.Getenv)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback for local development
		dsn = "host=localhost user=vector_admin password=password dbname=merchant port=5432 sslmode=disable"
	}

	// 2. Open the connection using postgres driver
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 3. Create the vector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Printf("Warning: Failed to create vector extension: %v", err)
		return nil, err
	}

	// 4. Create the vec_markets table if it doesn't exist
	// In pgvector, we use a standard table with a vector column type.
	// We map it 1:1 with markets via id.
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS vec_markets (
			rowid BIGINT PRIMARY KEY,
			embedding vector(384)
		);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create table vec_markets: %v", err)
	}

	// 5. Auto-migrate the schemas
	err = db.AutoMigrate(&Provider{}, &Market{}, &Event{}, &ArbitrageOpportunity{}, &MarketMapping{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
