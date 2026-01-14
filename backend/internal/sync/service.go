package sync

import (
	"fmt"
	"log"
	"time"

	"backend/internal/db"
	"backend/internal/embeddings"
	"backend/internal/kalshi"
	"backend/internal/llm"

	"gorm.io/gorm"
)

type Syncer struct {
	DB               *gorm.DB
	KClient          *kalshi.Client
	EmbeddingService embeddings.Service
	LLMManager       *llm.Manager // Changed from SLMService to LLMManager
	LastEventSync    time.Time
	TargetCategories []string // Categories to sync and analyze
}

func NewSyncer(database *gorm.DB, kClient *kalshi.Client, embeddingService embeddings.Service, llmManager *llm.Manager, targetCategories []string) *Syncer {
	return &Syncer{
		DB:               database,
		KClient:          kClient,
		EmbeddingService: embeddingService,
		LLMManager:       llmManager,
		TargetCategories: targetCategories,
	}
}

// SyncEmbeddings generates and saves embeddings for new active markets
func (s *Syncer) SyncEmbeddings() {
	if s.EmbeddingService == nil {
		log.Println("Skipping embedding sync: Embedding service not available")
		return
	}
	log.Println("Starting embedding sync...")

	// 1. Find missing embeddings for active markets that close within 14 days
	var missingMarkets []db.Market
	now := time.Now()
	twoWeeks := now.AddDate(0, 0, 14)

	// We join with events to filter by close time, and with vec_markets to find missing ones
	err := s.DB.Table("markets").
		Select("markets.*").
		Joins("JOIN events ON markets.event_ticker = events.external_id").
		Joins("LEFT JOIN vec_markets ON markets.id = vec_markets.rowid").
		Where("markets.status = ? AND events.closest_market_close_time BETWEEN ? AND ? AND vec_markets.rowid IS NULL", "active", now, twoWeeks).
		Find(&missingMarkets).Error

	if err != nil {
		log.Printf("Failed to fetch markets missing embeddings: %v", err)
		return
	}

	if len(missingMarkets) == 0 {
		log.Println("No missing embeddings found.")
		return
	}

	log.Printf("Generating embeddings for %d new markets...", len(missingMarkets))

	for _, m := range missingMarkets {
		// Construct text representation
		embeddingText := fmt.Sprintf("%s %s %s", m.Title, m.Description, m.Category)
		vec, err := s.EmbeddingService.Generate(embeddingText)
		if err != nil {
			log.Printf("Embedding generation failed for market %d: %v", m.ID, err)
			continue
		}

		vecString := pgVectorString(vec)

		// Insert directly into virtual table
		query := `INSERT INTO vec_markets(rowid, embedding) VALUES (?, ?)`
		if err := s.DB.Exec(query, m.ID, vecString).Error; err != nil {
			log.Printf("Failed to save embedding for market %d: %v", m.ID, err)
		}
	}

	// Prune stale embeddings for inactive markets
	s.pruneStaleEmbeddings()

	log.Println("Embedding sync complete.")
}

func (s *Syncer) pruneStaleEmbeddings() {
	// Remove embeddings where the market is no longer active
	// Since vec_markets is virtual, we delete by rowid
	query := `DELETE FROM vec_markets WHERE rowid IN (SELECT id FROM markets WHERE status != 'active')`
	if err := s.DB.Exec(query).Error; err != nil {
		log.Printf("Failed to prune stale embeddings: %v", err)
	}
}

// PruneExpiredEmbeddings removes embeddings for markets whose event close time has passed
func (s *Syncer) PruneExpiredEmbeddings() {
	log.Println("Pruning expired embeddings...")

	query := `DELETE FROM vec_markets WHERE rowid IN (
		SELECT markets.id FROM markets 
		JOIN events ON markets.event_ticker = events.external_id 
		WHERE events.closest_market_close_time < NOW()
	)`

	result := s.DB.Exec(query)
	if result.Error != nil {
		log.Printf("Failed to prune expired embeddings: %v", result.Error)
		return
	}

	log.Printf("Pruned %d expired embeddings.", result.RowsAffected)
}
