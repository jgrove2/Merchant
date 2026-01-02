package manager

import (
	"encoding/json"
	"log"
	"strconv"

	"backend/internal/db"
	"backend/internal/embeddings"
	"backend/internal/kalshi"
	"backend/internal/sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	SyncService      *sync.Syncer
}

// NewHandler creates a new manager Handler instance
func NewHandler(database *gorm.DB, kClient *kalshi.Client, embeddingService embeddings.Service, syncer *sync.Syncer) *Handler {
	return &Handler{
		DB:               database,
		KClient:          kClient,
		EmbeddingService: embeddingService,
		SyncService:      syncer,
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
	if h.SyncService != nil {
		h.SyncService.RunCycle()
	} else {
		log.Println("Sync service not available, skipping cycle.")
	}
}
