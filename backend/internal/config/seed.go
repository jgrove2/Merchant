package config

import (
	"log"

	"backend/internal/db"
	"gorm.io/gorm"
)

// SeedProviders ensures default providers exist in the database
func SeedProviders(database *gorm.DB) {
	providers := []db.Provider{
		{
			Name:     "kalshi",
			IsActive: true,
		},
	}

	for _, p := range providers {
		var count int64
		database.Model(&db.Provider{}).Where("name = ?", p.Name).Count(&count)
		if count == 0 {
			if err := database.Create(&p).Error; err != nil {
				log.Printf("Failed to seed provider %s: %v", p.Name, err)
			} else {
				log.Printf("Seeded provider: %s", p.Name)
			}
		}
	}
}
