package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"backend/internal/db"
	"backend/internal/llm"
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

			// Process related markets with LLM
			if len(related) > 0 {
				// log.Printf("Analyzing %d related markets for %s...", len(related), m.Ticker)

				// Convert simplified market to db.Market style for LLM helper
				sourceMarket := db.Market{
					ExternalID:  m.Ticker,
					EventTicker: m.EventTicker,
					Title:       m.Title,
					Description: m.Subtitle,
					YesSubTitle: m.YesSubTitle,
					NoSubTitle:  m.NoSubTitle,
				}

				// We need the ID for foreign key. Try to fetch from DB using ExternalID
				var sourceDB db.Market
				if err := s.DB.Where("external_id = ?", m.Ticker).First(&sourceDB).Error; err == nil {
					sourceMarket.ID = sourceDB.ID
				} else {
					// If we can't find the source market in DB, we can't save a mapping for it.
					// Skip this market for now.
					continue
				}

				for _, r := range related {
					if r.Market.ExternalID == m.Ticker {
						continue
					}

					// Skip if markets are in different categories
					if m.Category != r.Market.Category {
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
	// Safety check for Foreign Keys
	if source.ID == 0 || target.ID == 0 {
		return
	}

	// 1. Date Check (within 1 month)
	diff := sourceTime.Sub(targetTime)
	daysDiff := math.Abs(diff.Hours() / 24.0)

	if daysDiff > 30 {
		// log.Printf("Skipping comparison %s vs %s: Date diff %.1f days > 30", source.ExternalID, target.ExternalID, daysDiff)
		return
	}

	// 2. DB Check (Existing Mapping)
	// We check if a mapping already exists between these two markets
	var existing db.MarketMapping
	if err := s.DB.Where("source_market_id = ? AND target_market_id = ?", source.ID, target.ID).First(&existing).Error; err == nil {
		// Found existing mapping, skip LLM
		return
	}

	// 3. LLM Call
	if s.LLMManager == nil {
		return
	}

	// Map db.Market to llm.Market
	llmSource := llm.Market{
		Title:       source.Title,
		YesSubTitle: source.YesSubTitle,
		NoSubTitle:  source.NoSubTitle,
		EventId:     source.EventTicker,
		ID:          source.ExternalID,
	}
	llmTarget := llm.Market{
		Title:       target.Title,
		YesSubTitle: target.YesSubTitle,
		NoSubTitle:  target.NoSubTitle,
		EventId:     target.EventTicker,
		ID:          target.ExternalID,
	}

	result, err := s.LLMManager.CompareMarkets(context.Background(), llmSource, llmTarget)
	if err != nil {
		log.Printf("LLM Comparison failed for %s vs %s: %v", source.ExternalID, target.ExternalID, err)
		return
	}

	if result.Mapping.PrimaryYes == nil && result.Mapping.PrimaryNo == nil {
		log.Printf("LLM Analysis [%s vs %s]: No logical necessity found (both null), skipping save", source.ExternalID, target.ExternalID)
		return
	}

	yesStr := "null"
	if result.Mapping.PrimaryYes != nil {
		yesStr = *result.Mapping.PrimaryYes
	}
	noStr := "null"
	if result.Mapping.PrimaryNo != nil {
		noStr = *result.Mapping.PrimaryNo
	}

	fmtMarket := func(m db.Market) string {
		if m.YesSubTitle != "" || m.NoSubTitle != "" {
			return fmt.Sprintf("%s [Yes: %s | No: %s]", m.Title, m.YesSubTitle, m.NoSubTitle)
		}
		return fmt.Sprintf("%s [%s]", m.Title, m.Description)
	}

	// 4. Save to DB
	newMapping := db.MarketMapping{
		SourceMarketID: source.ID,
		TargetMarketID: target.ID,
		YesMapping:     result.Mapping.PrimaryYes,
		NoMapping:      result.Mapping.PrimaryNo,
		Analysis:       result.RawResponse, // Save raw response as analysis for now, or just empty if not needed
		CreatedAt:      time.Now(),
	}

	if err := s.DB.Create(&newMapping).Error; err != nil {
		log.Printf("Failed to save comparison for %s vs %s: %v", source.ExternalID, target.ExternalID, err)
	} else {
		log.Printf("LLM Analysis [%s vs %s]: PrimaryYes->%s, PrimaryNo->%s | Saved to DB", fmtMarket(source), fmtMarket(target), yesStr, noStr)
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

	// Format vector string for pgvector: '[1.0, 2.0, ...]'
	vecString = pgVectorString(vec)

	err = s.DB.Raw(`
		SELECT rowid as id, embedding <-> ? as distance
		FROM vec_markets
		ORDER BY distance
		LIMIT ?
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
