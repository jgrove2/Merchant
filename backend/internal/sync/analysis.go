package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"backend/internal/db"
)

// AnalyzeRelatedMarkets finds related markets for upcoming events
func (s *Syncer) AnalyzeRelatedMarkets() {
	log.Println("Starting related markets analysis...")

	// 1. Fetch upcoming events (closing within 14 days)
	now := time.Now()
	nextTwoWeeks := now.AddDate(0, 0, 14)

	var upcomingEvents []db.Event
	err := s.DB.Where("closest_market_close_time BETWEEN ? AND ?", now, nextTwoWeeks).
		Find(&upcomingEvents).Error
	if err != nil {
		log.Printf("Failed to fetch upcoming events for analysis: %v", err)
		return
	}

	log.Printf("Found %d upcoming events to analyze.", len(upcomingEvents))

	for _, event := range upcomingEvents {
		// 2. Fetch live markets for this event
		// Using the new method we added to the client
		liveMarkets, err := s.KClient.GetMarketsForEventNextMonth(event.ExternalID)
		if err != nil {
			log.Printf("Failed to fetch live markets for event %s: %v", event.ExternalID, err)
			continue
		}

		// 3. Loop through markets and find related ones
		for _, m := range liveMarkets {
			// Construct query from title and subtitle
			queryText := fmt.Sprintf("%s %s", m.Title, m.Subtitle)

			// Find top 10 related markets
			related, err := s.findRelatedMarkets(queryText, 10)
			if err != nil {
				log.Printf("Failed to find related markets for %s: %v", m.Ticker, err)
				continue
			}

			// Process related markets with SLM
			if len(related) > 0 {
				// log.Printf("Analyzing %d related markets for %s...", len(related), m.Ticker)

				// Convert simplified market to db.Market style for SLM helper
				sourceMarket := db.Market{
					ExternalID:  m.Ticker,
					EventTicker: m.EventTicker,
					Title:       m.Title,
					Description: m.Subtitle,
					YesSubTitle: m.YesSubTitle,
					NoSubTitle:  m.NoSubTitle,
				}

				for _, r := range related {
					if r.Market.ExternalID == m.Ticker {
						continue
					}
					var targetEvent db.Event
					// Try to fetch event to get close time, fallback to current market update time if fails
					targetCloseTime := r.Market.LastDataUpdate
					if err := s.DB.Where("external_id = ?", r.EventTicker).First(&targetEvent).Error; err == nil {
						targetCloseTime = targetEvent.ClosestMarketCloseTime
					}

					s.processComparison(sourceMarket, r.Market, m.CloseTime, targetCloseTime)
				}
			}
		}

	}
}

func (s *Syncer) processComparison(source, target db.Market, sourceTime, targetTime time.Time) {
	// 1. Date Check (within 1 month)
	diff := sourceTime.Sub(targetTime)
	daysDiff := math.Abs(diff.Hours() / 24.0)

	if daysDiff > 30 {
		// log.Printf("Skipping comparison %s vs %s: Date diff %.1f days > 30", source.ExternalID, target.ExternalID, daysDiff)
		return
	}

	// 2. Redis Check
	cacheKey := fmt.Sprintf("rel:%s:%s", source.ExternalID, target.ExternalID)

	if s.Redis != nil {
		_, err := s.Redis.Get(cacheKey)
		if err == nil {
			// Found in Redis. Extend TTL to 3 hours and skip SLM.
			// Re-save (or just set expire) - Set is easiest with existing helper if we had the value.
			// Since we just want to extend, Expire is better but our wrapper might not expose it.
			// Let's just skip for now as requested "if its in redis and not expired... it won't do the comparison".
			// The user said "lets set the expiration to 3 hours".
			// We need a way to extend. If wrapper doesn't have it, we assume we skip.
			// User: "If its in redis and not expired lets set the expiration to 3 hours"
			// I'll add Expire to Redis wrapper if needed or just re-set if I had the value.
			// Wrapper only has Get/Add. I will use Get, then AddWithTTL.
			// Wait, I need the value to re-set it. Get returns string.
			val, _ := s.Redis.Get(cacheKey)
			s.Redis.AddWithTTL(cacheKey, val, 3*time.Hour)
			return
		}
	}

	// 3. SLM Call
	if s.SLMService == nil {
		return
	}

	result, err := s.SLMService.CompareMarkets(source, target)
	if err != nil {
		log.Printf("SLM Comparison failed for %s vs %s: %v", source.ExternalID, target.ExternalID, err)
		return
	}

	if result.SourceYes == nil && result.SourceNo == nil {
		log.Printf("SLM Analysis [%s vs %s]: No logical necessity found (both null), skipping cache", source.ExternalID, target.ExternalID)
		return
	}

	yesStr := "null"
	if result.SourceYes != nil {
		yesStr = *result.SourceYes
	}
	noStr := "null"
	if result.SourceNo != nil {
		noStr = *result.SourceNo
	}

	fmtMarket := func(m db.Market) string {
		if m.YesSubTitle != "" || m.NoSubTitle != "" {
			return fmt.Sprintf("%s [Yes: %s | No: %s]", m.Title, m.YesSubTitle, m.NoSubTitle)
		}
		return fmt.Sprintf("%s [%s]", m.Title, m.Description)
	}

	// 4. Save to Redis
	if s.Redis != nil {
		jsonBytes, _ := json.Marshal(result)
		err := s.Redis.AddWithTTL(cacheKey, string(jsonBytes), 3*time.Hour)
		if err != nil {
			log.Printf("Failed to cache comparison for %s vs %s: %v", source.ExternalID, target.ExternalID, err)
		} else {
			log.Printf("SLM Analysis [%s vs %s]: SourceYes->%s, SourceNo->%s | Saved to Redis", fmtMarket(source), fmtMarket(target), yesStr, noStr)
		}
	} else {
		log.Printf("SLM Analysis [%s vs %s]: SourceYes->%s, SourceNo->%s | Redis not available", fmtMarket(source), fmtMarket(target), yesStr, noStr)
	}
}

type MarketWithScore struct {
	db.Market
	Score float32
}

func (s *Syncer) findRelatedMarkets(query string, limit int) ([]MarketWithScore, error) {
	// 1. Generate embedding
	vec, err := s.EmbeddingService.Generate(query)
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", err)
	}

	vecBytes, _ := json.Marshal(vec)
	vecString := string(vecBytes)

	// 2. Perform vector search
	type SearchResult struct {
		ID       uint
		Distance float64
	}
	var results []SearchResult

	err = s.DB.Raw(`
		SELECT rowid as id, distance 
		FROM vec_markets 
		WHERE embedding MATCH ? 
		AND k = ?
		ORDER BY distance
	`, vecString, limit).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []MarketWithScore{}, nil
	}

	// 3. Fetch full market details
	var marketIDs []uint
	scoreMap := make(map[uint]float64)
	for _, r := range results {
		marketIDs = append(marketIDs, r.ID)
		scoreMap[r.ID] = r.Distance
	}

	var dbMarkets []db.Market
	if err := s.DB.Where("id IN ?", marketIDs).Find(&dbMarkets).Error; err != nil {
		return nil, err
	}

	// 4. Reconstruct order
	var response []MarketWithScore
	marketMap := make(map[uint]db.Market)
	for _, m := range dbMarkets {
		marketMap[m.ID] = m
	}

	for _, r := range results {
		if m, exists := marketMap[r.ID]; exists {
			response = append(response, MarketWithScore{
				Market: m,
				Score:  float32(r.Distance),
			})
		}
	}

	return response, nil
}
