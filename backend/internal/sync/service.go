package sync

import (
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
	Redis            *db.Redis
	LastEventSync    time.Time
	TargetCategory   string // Optional filter for sync
}

func NewSyncer(database *gorm.DB, kClient *kalshi.Client, embeddingService embeddings.Service, llmManager *llm.Manager, rdb *db.Redis, targetCategory string) *Syncer {
	return &Syncer{
		DB:               database,
		KClient:          kClient,
		EmbeddingService: embeddingService,
		LLMManager:       llmManager,
		Redis:            rdb,
		TargetCategory:   targetCategory,
	}
}

// RunCycle performs the market sync and analysis
func (s *Syncer) RunCycle() {
	// 1. Sync events (daily check inside)
	s.SyncEvents()

	// 2. Analyze related markets for upcoming events
	if s.EmbeddingService != nil {
		// Check global analysis cooldown
		runAnalysis := true
		if s.Redis != nil {
			if _, err := s.Redis.Get("analysis:global_cooldown"); err == nil {
				runAnalysis = false
				log.Println("Skipping analysis: Global cooldown active")
			}
		}

		if runAnalysis {
			s.AnalyzeRelatedMarkets()
			// Set global cooldown
			if s.Redis != nil {
				s.Redis.AddWithTTL("analysis:global_cooldown", "1", 3*time.Hour)
			}
		}
	} else {
		log.Println("Skipping related markets analysis: Embedding service not available")
	}
}
