package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
)

// Connect now reads directly from the environment
func Connect() (*gorm.DB, error) {
	// 1. Get the connection string from the .env (via os.Getenv)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback for local development if variable isn't set
		dsn = "host=localhost user=postgres password=password dbname=arb_db port=5432 sslmode=disable"
	}

	// 2. Open the connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 3. Auto-migrate the schemas
	err = db.AutoMigrate(&Provider{}, &Market{}, &ArbitrageOpportunity{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
