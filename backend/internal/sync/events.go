package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"backend/internal/db"
	"backend/internal/kalshi"
	kalshiTypes "backend/internal/kalshi/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Syncer) SyncEvents() {
	if s.KClient == nil {
		log.Println("Skipping event sync: Kalshi client not configured")
		return
	}

	log.Println("Starting daily event sync...")

	// 1. Get/Create Kalshi Provider
	var provider db.Provider
	if err := s.DB.Where("name = ?", "kalshi").FirstOrCreate(&provider, db.Provider{Name: "kalshi"}).Error; err != nil {
		log.Printf("Failed to get/create provider: %v", err)
		return
	}

	// Check if we synced recently (within 24 hours)
	if time.Since(provider.LastEventSync) < 24*time.Hour {
		log.Printf("Skipping sync: Last sync was %v ago", time.Since(provider.LastEventSync))
		// For development/debugging, you might want to comment this out to force sync
		// return
		// Keeping return for production behavior, but ensure logic allows forcing if needed.
		// Given the user wants to test the "next part", we might want to bypass this check if we haven't synced *this run*.
		// However, s.LastEventSync on the struct is ephemeral. provider.LastEventSync is persistent.
		// If the user wants to run the analysis, we should let it proceed even if sync is skipped.
	} else {
		// Only run the heavy sync if needed
		s.performEventSync(provider)
	}

	// Update in-memory state
	s.LastEventSync = time.Now()
}

func (s *Syncer) performEventSync(provider db.Provider) {
	totalFetched := 0
	eventCursor := ""
	const batchSize = 100
	const rateLimitDelay = 100 * time.Millisecond

	for {
		time.Sleep(rateLimitDelay)

		// 2. Fetch from API
		resp, err := s.fetchEventsWithRetry(batchSize, eventCursor)
		if err != nil {
			log.Printf("Failed to fetch events: %v", err)
			break
		}
		eventCursor = resp.Cursor
		if resp == nil || len(resp.Events) == 0 {
			break
		}

		// 3. Process Data into Structs
		eventsToUpsert, marketsToUpsert := s.processEventBatch(resp.Events, provider.ID)

		// 4. DB Operations (Upsert Events & Markets)
		// We do this in a transaction to ensure consistency
		if len(eventsToUpsert) > 0 {
			err := s.upsertEventsData(eventsToUpsert)
			if err != nil {
				log.Printf("Failed to upsert events batch: %v", err)
				continue
			}
		}

		if len(marketsToUpsert) > 0 {
			err := s.upsertMarketData(marketsToUpsert)
			if err != nil {
				log.Printf("Failed to upsert markets batch: %v", err)
				continue
			}
		}

		// 5. Update Embeddings (Outside Transaction)
		if s.EmbeddingService != nil {
			if len(marketsToUpsert) > 0 {
				s.updateMarketEmbeddings(marketsToUpsert)
			}
		}

		totalFetched += len(resp.Events)
		if resp.Cursor == "" {
			log.Println("Reached end of events list.")
			break
		}
	}

	// 6. Cleanup
	s.pruneStaleEmbeddings()

	// Update provider last sync time
	now := time.Now()
	if err := s.DB.Model(&provider).Update("last_event_sync", now).Error; err != nil {
		log.Printf("Failed to update provider last sync time: %v", err)
	}

	log.Printf("Event sync complete. Total processed: %d", totalFetched)
}

// --- Helper Functions ---

func (s *Syncer) fetchEventsWithRetry(limit int, cursor string) (*kalshi.EventsResponse, error) {
	var resp *kalshi.EventsResponse
	var err error

	log.Printf("Fetching event batch (cursor: %s)...", cursor)

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(1 * time.Second)
		}
		resp, err = s.KClient.GetEvents(limit, cursor)
		if err == nil {
			return resp, nil
		}
	}
	return nil, err
}

func (s *Syncer) processEventBatch(apiEvents []kalshiTypes.SimplifiedEvent, providerID uint) ([]db.Event, []db.Market) {
	var dbEvents []db.Event
	var dbMarkets []db.Market

	now := time.Now()

	for _, e := range apiEvents {
		closestCloseTime := time.Time{}

		// Logic to extract markets
		var eventMarkets []kalshiTypes.SimplifiedMarket
		if len(e.Markets) > 0 {
			eventMarkets = e.Markets
		} else {
			// Fallback: Fetch markets individually if not nested
			time.Sleep(50 * time.Millisecond) // Rate limit protection
			fullEvent, err := s.KClient.GetEvent(e.EventTicker)
			if err != nil {
				log.Printf("Failed to fetch fallback markets for event %s: %v", e.EventTicker, err)
				continue
			}
			if fullEvent != nil {
				eventMarkets = fullEvent.Markets
			}
		}

		// Calculate closest time
		for _, m := range eventMarkets {
			if m.CloseTime.After(now) {
				if closestCloseTime.IsZero() || m.CloseTime.Before(closestCloseTime) {
					closestCloseTime = m.CloseTime
				}
			}
		}

		// If no future close time found, fallback to expiration time
		expTime, _ := time.Parse(time.RFC3339, e.ExpirationTime)

		dbEvents = append(dbEvents, db.Event{
			ProviderID:             providerID,
			ExternalID:             e.EventTicker,
			Title:                  e.Title,
			Subtitle:               e.SubTitle,
			Category:               e.Category,
			MutuallyExclusive:      e.MutuallyExclusive,
			SeriesTicker:           e.SeriesTicker,
			StrikePeriod:           e.StrikePeriod,
			ExpirationTime:         expTime,
			ClosestMarketCloseTime: closestCloseTime,
		})

		for _, m := range eventMarkets {
			fullTitle := m.Title
			if m.YesSubTitle != "" || m.NoSubTitle != "" {
				fullTitle = fullTitle + " " + m.YesSubTitle
			} else if m.Subtitle != "" {
				fullTitle = fullTitle + " " + m.Subtitle
			}

			cat := m.Category
			if cat == "" {
				cat = e.Category
			}

			dbMarkets = append(dbMarkets, db.Market{
				ProviderID:     providerID,
				ExternalID:     m.Ticker,
				Ticker:         m.Ticker,
				EventTicker:    e.EventTicker,
				Title:          fullTitle,
				Description:    m.Subtitle,
				YesSubTitle:    m.YesSubTitle,
				NoSubTitle:     m.NoSubTitle,
				Status:         m.Status,
				Category:       cat,
				LastDataUpdate: time.Now(),
			})
		}
	}
	return dbEvents, dbMarkets
}

func (s *Syncer) upsertMarketData(markets []db.Market) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// Upsert Markets
		if len(markets) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "provider_id"}, {Name: "external_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"title", "description", "yes_sub_title", "no_sub_title", "status", "category", "last_data_update", "updated_at", "event_ticker",
				}),
			}).Create(&markets).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Syncer) upsertEventsData(events []db.Event) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// Upsert Events
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "provider_id"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"title", "subtitle", "category", "expiration_time", "closest_market_close_time", "updated_at",
			}),
		}).Create(&events).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *Syncer) updateMarketEmbeddings(markets []db.Market) {
	// Re-fetch the markets to ensure we have the IDs populated from the upsert
	var freshMarkets []db.Market
	tickers := make([]string, len(markets))
	for i, m := range markets {
		tickers[i] = m.Ticker
	}

	if err := s.DB.Where("ticker IN ?", tickers).Find(&freshMarkets).Error; err != nil {
		log.Printf("Failed to fetch fresh markets for embeddings: %v", err)
		return
	}

	log.Printf("Updating embeddings for %d markets...", len(freshMarkets))

	for _, m := range freshMarkets {
		if m.Status != "active" {
			continue
		}

		embeddingText := fmt.Sprintf("%s %s %s", m.Title, m.Description, m.Category)
		vec, err := s.EmbeddingService.Generate(embeddingText)
		if err != nil {
			log.Printf("Gen failed: %v", err)
			continue
		}

		vecBytes, err := json.Marshal(vec)
		if err != nil {
			log.Printf("Marshal failed: %v", err)
			continue
		}
		vecString := string(vecBytes)

		// Delete existing
		_ = s.DB.Exec("DELETE FROM vec_markets WHERE rowid = ?", m.ID)

		query := `
			INSERT INTO vec_markets(rowid, embedding) 
			VALUES (?, ?)
		`
		if err := s.DB.Exec(query, m.ID, vecString).Error; err != nil {
			log.Printf("Failed to save embedding for market %d: %v", m.ID, err)
		}
	}
}

func (s *Syncer) pruneStaleEmbeddings() {
	if s.EmbeddingService == nil {
		return
	}
	log.Println("Pruning stale embeddings...")

	err := s.DB.Exec(`
		DELETE FROM vec_markets 
		WHERE rowid IN (
			SELECT id FROM markets WHERE status != 'active'
		)
	`).Error

	if err != nil {
		log.Printf("Failed to prune stale embeddings: %v", err)
	} else {
		log.Println("Pruning complete.")
	}
}
