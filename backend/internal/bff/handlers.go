package bff

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB         *gorm.DB
	ManagerURL string
}

// NewHandler creates a new Handler instance
func NewHandler(database *gorm.DB, managerURL string) *Handler {
	return &Handler{
		DB:         database,
		ManagerURL: managerURL,
	}
}

// GetTotalBalance aggregates balances from all providers
func (h *Handler) GetTotalBalance(c *gin.Context) {
	// For now, we only query Kalshi from the manager
	// In the future, this would aggregate from multiple provider endpoints

	log.Println("Fetching total balance from manager:", h.ManagerURL)
	resp, err := http.Get(h.ManagerURL + "/providers/kalshi/balance")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to contact manager"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to get balance from manager"})
		return
	}

	var data struct {
		Balance int64 `json:"balance"`
	}
	// Use json decoder
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_balance": data.Balance, // Returns in cents
		"currency":      "USD",
		"breakdown": gin.H{
			"kalshi": data.Balance,
		},
	})
}

func (h *Handler) ToggleTrading(c *gin.Context) {
	var input struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trading_active": input.Active})
}

func (h *Handler) GetMarkets(c *gin.Context) {
	url := h.ManagerURL + "/markets"
	params := []string{}
	if limit := c.Query("limit"); limit != "" {
		params = append(params, "limit="+limit)
	}
	if cursor := c.Query("cursor"); cursor != "" {
		params = append(params, "cursor="+cursor)
	}
	if mveFilter := c.Query("mve_filter"); mveFilter != "" {
		params = append(params, "mve_filter="+mveFilter)
	}
	if minCloseTs := c.Query("min_close_ts"); minCloseTs != "" {
		params = append(params, "min_close_ts="+minCloseTs)
	}
	if maxCloseTs := c.Query("max_close_ts"); maxCloseTs != "" {
		params = append(params, "max_close_ts="+maxCloseTs)
	}

	if len(params) > 0 {
		url = url + "?"
		for i, param := range params {
			if i > 0 {
				url = url + "&"
			}
			url = url + param
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to contact manager"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to get markets from manager"})
		return
	}

	var data struct {
		Markets []map[string]interface{} `json:"markets"`
		Cursor  string                   `json:"cursor"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"markets": data.Markets,
		"cursor":  data.Cursor,
	})
}

func (h *Handler) GetEvents(c *gin.Context) {
	url := h.ManagerURL + "/events"
	params := []string{}

	// Only allow limit and cursor parameters from the client
	// Do NOT allow min_close_ts to be set by the client
	if limit := c.Query("limit"); limit != "" {
		params = append(params, "limit="+limit)
	}
	if cursor := c.Query("cursor"); cursor != "" {
		params = append(params, "cursor="+cursor)
	}

	if len(params) > 0 {
		url = url + "?"
		for i, param := range params {
			if i > 0 {
				url = url + "&"
			}
			url = url + param
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to contact manager"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to get events from manager"})
		return
	}

	var data struct {
		Events     []map[string]interface{} `json:"events"`
		Cursor     string                   `json:"cursor"`
		Milestones []interface{}            `json:"milestones"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":     data.Events,
		"cursor":     data.Cursor,
		"milestones": data.Milestones,
	})
}

func (h *Handler) GetMarketsByEvent(c *gin.Context) {
	url := h.ManagerURL + "/markets/by-event"
	params := []string{}

	// event_ticker is required
	if eventTicker := c.Query("event_ticker"); eventTicker != "" {
		params = append(params, "event_ticker="+eventTicker)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_ticker query parameter is required"})
		return
	}

	if limit := c.Query("limit"); limit != "" {
		params = append(params, "limit="+limit)
	}
	if cursor := c.Query("cursor"); cursor != "" {
		params = append(params, "cursor="+cursor)
	}
	if mveFilter := c.Query("mve_filter"); mveFilter != "" {
		params = append(params, "mve_filter="+mveFilter)
	}
	if minCloseTs := c.Query("min_close_ts"); minCloseTs != "" {
		params = append(params, "min_close_ts="+minCloseTs)
	}
	if maxCloseTs := c.Query("max_close_ts"); maxCloseTs != "" {
		params = append(params, "max_close_ts="+maxCloseTs)
	}

	if len(params) > 0 {
		url = url + "?"
		for i, param := range params {
			if i > 0 {
				url = url + "&"
			}
			url = url + param
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to contact manager"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to get markets from manager"})
		return
	}

	var data struct {
		Markets []map[string]interface{} `json:"markets"`
		Cursor  string                   `json:"cursor"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"markets": data.Markets,
		"cursor":  data.Cursor,
	})
}
