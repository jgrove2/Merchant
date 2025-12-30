package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"backend/internal/db"
	"backend/internal/embeddings"
	"backend/internal/kalshi"
	kalshiTypes "backend/internal/kalshi/types"
	"bytes"
	"encoding/binary"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

type MarketWithScore struct {
	db.Market
	Score float32 `json:"score"`
}

type Handler struct {
	DB               *gorm.DB
	KClient          *kalshi.Client
	EmbeddingService embeddings.Service
	LastEventSync    time.Time
}

// NewHandler creates a new manager Handler instance
func NewHandler(database *gorm.DB, kClient *kalshi.Client, embeddingService embeddings.Service) *Handler {
	return &Handler{
		DB:               database,
		KClient:          kClient,
		EmbeddingService: embeddingService,
	}
}

// GetProviderBalance returns the balance for a specific provider
func (h *Handler) GetProviderBalance(c *gin.Context) {
	name := c.Param("name")
	if name == "kalshi" {
		if h.KClient == nil {
			c.JSON(503, gin.H{"error": "Kalshi client not configured"})
			return
		}
		bal, err := h.KClient.GetBalance()
		if err != nil {
			log.Println(err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"balance": bal})
		return
	}
	c.JSON(404, gin.H{"error": "Provider not supported"})
}

func (h *Handler) GetMarkets(c *gin.Context) {
	if h.KClient == nil {
		c.JSON(503, gin.H{"error": "Kalshi client not configured"})
		return
	}

	// Parse query parameters
	limit := 100 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	cursor := c.Query("cursor")

	// Default mve_filter to "exclude"
	mveFilter := c.Query("mve_filter")
	if mveFilter == "" {
		mveFilter = "exclude"
	}

	// Parse optional date parameters (Unix timestamps)
	var minCloseTs, maxCloseTs int64
	if minCloseTsStr := c.Query("min_close_ts"); minCloseTsStr != "" {
		if parsedTs, err := strconv.ParseInt(minCloseTsStr, 10, 64); err == nil {
			minCloseTs = parsedTs
		}
	}
	if maxCloseTsStr := c.Query("max_close_ts"); maxCloseTsStr != "" {
		if parsedTs, err := strconv.ParseInt(maxCloseTsStr, 10, 64); err == nil {
			maxCloseTs = parsedTs
		}
	}

	response, err := h.KClient.GetMarkets(limit, cursor, mveFilter, minCloseTs, maxCloseTs)
	if err != nil {
		log.Println("Error fetching markets:", err.Error())
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if response == nil || response.Markets == nil || len(response.Markets) == 0 {
		c.JSON(404, gin.H{"error": "No markets found"})
		return
	}

	c.JSON(200, gin.H{
		"markets": response.Markets,
		"cursor":  response.Cursor,
	})
}

func (h *Handler) GetEvents(c *gin.Context) {
	if h.KClient == nil {
		c.JSON(503, gin.H{"error": "Kalshi client not configured"})
		return
	}

	// Parse query parameters
	limit := 100 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	cursor := c.Query("cursor")

	response, err := h.KClient.GetEvents(limit, cursor)
	if err != nil {
		log.Println("Error fetching events:", err.Error())
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if response == nil || response.Events == nil || len(response.Events) == 0 {
		c.JSON(404, gin.H{"error": "No events found"})
		return
	}

	c.JSON(200, gin.H{
		"events":     response.Events,
		"cursor":     response.Cursor,
		"milestones": response.Milestones,
	})
}

func (h *Handler) GetEvent(c *gin.Context) {
	if h.KClient == nil {
		c.JSON(503, gin.H{"error": "Kalshi client not configured"})
		return
	}

	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(400, gin.H{"error": "event_id is required"})
		return
	}

	event, err := h.KClient.GetEvent(eventID)
	if err != nil {
		log.Println("Error fetching event:", err.Error())
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if event == nil {
		c.JSON(404, gin.H{"error": "Event not found"})
		return
	}

	c.JSON(200, gin.H{"event": event})
}

func (h *Handler) GetMarketsByEvent(c *gin.Context) {
	if h.KClient == nil {
		c.JSON(503, gin.H{"error": "Kalshi client not configured"})
		return
	}

	// Get event_ticker from query parameter (required)
	eventTicker := c.Query("event_ticker")
	if eventTicker == "" {
		c.JSON(400, gin.H{"error": "event_ticker query parameter is required"})
		return
	}

	// Parse query parameters
	limit := 100 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	cursor := c.Query("cursor")

	// Parse optional date parameters (Unix timestamps)
	var minCloseTs, maxCloseTs int64
	if minCloseTsStr := c.Query("min_close_ts"); minCloseTsStr != "" {
		if parsedTs, err := strconv.ParseInt(minCloseTsStr, 10, 64); err == nil {
			minCloseTs = parsedTs
		}
	}
	if maxCloseTsStr := c.Query("max_close_ts"); maxCloseTsStr != "" {
		if parsedTs, err := strconv.ParseInt(maxCloseTsStr, 10, 64); err == nil {
			maxCloseTs = parsedTs
		}
	}

	response, err := h.KClient.GetMarketsByEvent(eventTicker,
		limit,
		cursor,
		eventTicker, minCloseTs, maxCloseTs)
	if err != nil {
		log.Println("Error fetching markets by event:", err.Error())
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if response == nil || response.Markets == nil || len(response.Markets) == 0 {
		c.JSON(404, gin.H{"error": "No markets found for this event"})
		return
	}

	c.JSON(200, gin.H{
		"markets": response.Markets,
		"cursor":  response.Cursor,
	})
}

// SearchMarkets performs a vector similarity search
func (h *Handler) SearchMarkets(c *gin.Context) {
	if h.EmbeddingService == nil {
		c.JSON(503, gin.H{"error": "Embedding service not available"})
		return
	}

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10 // Default limit
	}
	if req.Limit > 50 {
		req.Limit = 50 // Max limit
	}

	// 1. Generate embedding for query
	queryVec, err := h.EmbeddingService.Generate(req.Query)
	if err != nil {
		log.Println("Error generating query embedding:", err)
		c.JSON(500, gin.H{"error": "Failed to process query"})
		return
	}

	// Marshal query vector to JSON string
	queryVecBytes, _ := json.Marshal(queryVec)
	queryVecString := string(queryVecBytes)

	// 2. Perform vector search
	// We select rowid (which maps to market ID) and distance
	type SearchResult struct {
		ID       uint
		Distance float64
	}
	var results []SearchResult

	// Note: sqlite-vec uses 'distance' as the column for the score in `vec0` queries.
	// We scan directly into our struct.
	// We select 'rowid' which is the primary key in vec0 tables and map it to our ID.
	err = h.DB.Raw(`
		SELECT rowid as id, distance 
		FROM vec_markets 
		WHERE embedding MATCH ? 
		AND k = ?
		ORDER BY distance
	`, queryVecString, req.Limit).Scan(&results).Error

	if err != nil {
		log.Println("Error performing vector search:", err)
		c.JSON(500, gin.H{"error": "Search failed"})
		return
	}

	if len(results) == 0 {
		c.JSON(200, gin.H{"markets": []MarketWithScore{}})
		return
	}

	// 3. Fetch full market details and combine
	var marketIDs []uint
	scoreMap := make(map[uint]float64)
	for _, r := range results {
		marketIDs = append(marketIDs, r.ID)
		scoreMap[r.ID] = r.Distance
	}

	var dbMarkets []db.Market
	if err := h.DB.Where("id IN ?", marketIDs).Find(&dbMarkets).Error; err != nil {
		log.Println("Error fetching market details:", err)
		c.JSON(500, gin.H{"error": "Failed to fetch market details"})
		return
	}

	// 4. Sort and format response
	// The DB query might not return them in the same order, so we reconstruct the list
	// based on the original search results order to maintain relevance ranking.
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

	c.JSON(200, gin.H{"markets": response})
}

// RunSyncCycle performs the market sync and arbitrage calculation
func (h *Handler) RunSyncCycle() {
	// Sync events every 24 hours
	h.SyncEvents()
}

func (h *Handler) SyncEvents() {
	if h.KClient == nil {
		log.Println("Skipping event sync: Kalshi client not configured")
		return
	}

	log.Println("Starting daily event sync...")

	// 1. Get/Create Kalshi Provider
	var provider db.Provider
	if err := h.DB.Where("name = ?", "kalshi").FirstOrCreate(&provider, db.Provider{Name: "kalshi"}).Error; err != nil {
		log.Printf("Failed to get/create provider: %v", err)
		return
	}

	// Check if we synced recently (within 24 hours)
	if time.Since(provider.LastEventSync) < 24*time.Hour {
		log.Printf("Skipping sync: Last sync was %v ago", time.Since(provider.LastEventSync))
		return
	}

	totalFetched := 0
	eventCursor := ""
	const batchSize = 100
	const rateLimitDelay = 100 * time.Millisecond

	for {
		time.Sleep(rateLimitDelay)

		// 2. Fetch from API
		resp, err := h.fetchEventsWithRetry(batchSize, eventCursor)
		if err != nil {
			log.Printf("Failed to fetch events: %v", err)
			break
		}
		eventCursor = resp.Cursor
		if resp == nil || len(resp.Events) == 0 {
			break
		}

		// 3. Process Data into Structs
		eventsToUpsert, marketsToUpsert := h.processEventBatch(resp.Events, provider.ID)

		// 4. DB Operations (Upsert Events & Markets)
		// We do this in a transaction to ensure consistency
		if len(eventsToUpsert) > 0 {
			err := h.upsertEventsData(eventsToUpsert)
			if err != nil {
				log.Printf("Failed to upsert events batch: %v", err)
				continue
			}
		}

		if len(marketsToUpsert) > 0 {
			err := h.upsertMarketData(marketsToUpsert)
			if err != nil {
				log.Printf("Failed to upsert markets batch: %v", err)
				continue
			}
		}

		// 5. Update Embeddings (Outside Transaction)
		if h.EmbeddingService != nil {
			if len(marketsToUpsert) > 0 {
				h.updateMarketEmbeddings(marketsToUpsert)
			}
		}

		totalFetched += len(resp.Events)
		if resp.Cursor == "" {
			log.Println("Reached end of events list.")
			break
		}
	}

	// 6. Cleanup
	h.pruneStaleEmbeddings()

	// Update provider last sync time
	now := time.Now()
	if err := h.DB.Model(&provider).Update("last_event_sync", now).Error; err != nil {
		log.Printf("Failed to update provider last sync time: %v", err)
	}

	h.LastEventSync = now
	log.Printf("Event sync complete. Total processed: %d", totalFetched)
}

// --- Helper Functions ---

func (h *Handler) fetchEventsWithRetry(limit int, cursor string) (*kalshi.EventsResponse, error) {
	var resp *kalshi.EventsResponse
	var err error

	log.Printf("Fetching event batch (cursor: %s)...", cursor)

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(1 * time.Second)
		}
		resp, err = h.KClient.GetEvents(limit, cursor)
		if err == nil {
			return resp, nil
		}
	}
	return nil, err
}

func (h *Handler) processEventBatch(apiEvents []kalshiTypes.SimplifiedEvent, providerID uint) ([]db.Event, []db.Market) {
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
			// This handles cases where the bulk API didn't return nested markets
			time.Sleep(50 * time.Millisecond) // Rate limit protection
			fullEvent, err := h.KClient.GetEvent(e.EventTicker)
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

		// If no future close time found, fallback to expiration time or now (though 0 time is fine too)
		// But let's keep it zero if none found to indicate no active markets closing soon.

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
				Status:         m.Status,
				Category:       cat,
				LastDataUpdate: time.Now(),
			})
		}
	}
	return dbEvents, dbMarkets
}

func (h *Handler) upsertMarketData(markets []db.Market) error {
	return h.DB.Transaction(func(tx *gorm.DB) error {
		// Upsert Markets
		if len(markets) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "provider_id"}, {Name: "external_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"title", "description", "status", "category", "last_data_update", "updated_at", "event_ticker",
				}),
			}).Create(&markets).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *Handler) upsertEventsData(events []db.Event) error {
	return h.DB.Transaction(func(tx *gorm.DB) error {
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

func (h *Handler) updateMarketEmbeddings(markets []db.Market) {
	// Re-fetch the markets to ensure we have the IDs populated from the upsert
	var freshMarkets []db.Market
	tickers := make([]string, len(markets))
	for i, m := range markets {
		tickers[i] = m.Ticker
	}

	// Fetch markets regardless of status to debug why they aren't being embedded
	// We'll log the count to be sure.
	if err := h.DB.Where("ticker IN ?", tickers).Find(&freshMarkets).Error; err != nil {
		log.Printf("Failed to fetch fresh markets for embeddings: %v", err)
		return
	}

	log.Printf("Updating embeddings for %d markets...", len(freshMarkets))

	for _, m := range freshMarkets {
		// Only embed if active, OR if we want to support searching non-active markets.
		// For now, let's keep the active check but LOG if we skip one.
		if m.Status != "active" {
			// log.Printf("Skipping embedding for non-active market %s (status: %s)", m.Ticker, m.Status)
			continue
		}

		// 1. Generate Embedding
		// Include title, subtitle (description), and category.
		// Note: Many descriptions are empty, so Title carries the weight.
		embeddingText := fmt.Sprintf("%s %s %s", m.Title, m.Description, m.Category)

		// Optional: Normalize text (lowercase, remove special chars) if needed,
		// but the embedding model usually handles raw text fine.

		vec, err := h.EmbeddingService.Generate(embeddingText)
		if err != nil {
			log.Printf("Gen failed: %v", err)
			continue
		}

		// 2. Marshal to JSON
		vecBytes, err := json.Marshal(vec)
		if err != nil {
			log.Printf("Marshal failed: %v", err)
			continue
		}
		vecString := string(vecBytes) // This looks like "[0.123, -0.456, ...]"

		// 3. Construct the Query
		// We pass the JSON string directly. sqlite-vec (vec0) supports parsing JSON arrays.
		// We use 'rowid' explicitly as vec0 uses it for the primary key.
		// Note: INSERT OR REPLACE INTO vec_markets(rowid, embedding) VALUES (?, ?)
		// often fails with UNIQUE constraint on rowid despite the REPLACE keyword
		// when using virtual tables.
		//
		// Strategy: Always delete first, then insert. This is slower but safe.
		_ = h.DB.Exec("DELETE FROM vec_markets WHERE rowid = ?", m.ID)

		query := `
			INSERT INTO vec_markets(rowid, embedding) 
			VALUES (?, ?)
		`

		// 4. Execute
		// NOTE: sqlite-vec is VERY picky about types.
		// We found that for insertion to work reliably with the vec0 virtual table:
		// 1. We must pass the ID as an explicit argument if we want to set the rowid.
		// 2. We must pass the vector as a raw JSON string (e.g. "[0.1, 0.2]")
		if err := h.DB.Exec(query, m.ID, vecString).Error; err != nil {
			log.Printf("Failed to save embedding for market %d: %v", m.ID, err)
		}
	}
}

func (h *Handler) pruneStaleEmbeddings() {
	if h.EmbeddingService == nil {
		return
	}
	log.Println("Pruning stale embeddings...")

	// Note: We use 'rowid' for the virtual table delete
	err := h.DB.Exec(`
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

func float32ToBytes(floats []float32) ([]byte, error) {
	buf := new(bytes.Buffer)
	// Write the entire slice into the buffer as Little Endian binary
	for _, f := range floats {
		err := binary.Write(buf, binary.LittleEndian, f)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
