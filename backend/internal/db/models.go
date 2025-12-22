package db

import (
	"time"
)

// Provider represents an exchange like Kalshi, Polymarket, etc.
type Provider struct {
	ID        uint     `gorm:"primaryKey"`
	Name      string   `gorm:"uniqueIndex;not null"` // e.g., "kalshi"
	IsActive  bool     `gorm:"default:true"`
	Markets   []Market `gorm:"foreignKey:ProviderID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Market represents a specific betting contract or event
type Market struct {
	ID             uint   `gorm:"primaryKey"`
	ProviderID     uint   `gorm:"not null"`
	ExternalID     string `gorm:"uniqueIndex:idx_provider_market"` // The ID from Kalshi
	Ticker         string `gorm:"index"`                           // e.g., "FED-24DEC-T25"
	Title          string
	Description    string
	Status         string    `gorm:"default:'active'"` // active, closed, settled
	Category       string    // e.g., "Economics", "Politics"
	LastDataUpdate time.Time // Last time we pulled orderbook data
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ArbitrageOpportunity represents a detected trade
type ArbitrageOpportunity struct {
	ID              uint    `gorm:"primaryKey"`
	MarketID        uint    `gorm:"index"`
	Market          Market  `gorm:"foreignKey:MarketID"`
	StrategyType    string  `gorm:"index"` // e.g., "cross_exchange", "binary_hedge"
	BuyPrice        float64 // The price to enter
	SellPrice       float64 // The price to exit/offset
	ExpectedYield   float64 `gorm:"index"` // Calculated ROI
	PotentialProfit float64
	RequiredCapital float64
	Status          string    `gorm:"default:'detected'"` // detected, pending, executed, ignored
	DetectedAt      time.Time `gorm:"autoCreateTime"`
	ExpiresAt       time.Time
}
