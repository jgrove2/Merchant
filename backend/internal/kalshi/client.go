package kalshi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	// "log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/kalshi/types"
)

// Client handles communication with the Kalshi API
type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	Credentials AuthCredentials
}

// NewClient initializes a new Kalshi API client
func NewClient(baseURL, accessKey, keyPath string) (*Client, error) {
	// Validate required parameters
	if baseURL == "" {
		return nil, errors.New("baseURL is required")
	}
	if accessKey == "" {
		return nil, errors.New("accessKey is required")
	}
	if keyPath == "" {
		return nil, errors.New("keyPath is required")
	}

	// Validate URL has a scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, fmt.Errorf("baseURL must start with http:// or https://, got: %s", baseURL)
	}

	privKey, err := LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Credentials: AuthCredentials{
			PrivateKey: privKey,
			AccessKey:  accessKey,
		},
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// DoRequest performs an authenticated request to Kalshi
func (c *Client) DoRequest(method, path string, body io.Reader) ([]byte, error) {
	// 1. Prepare Timestamp
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// 2. Generate Signature (Path must be stripped of query params for signing)
	pathWithoutQuery := strings.Split(path, "?")[0]

	// log.Println(pathWithoutQuery)

	sig, err := c.Credentials.SignMessage(method, pathWithoutQuery, timestamp)
	if err != nil {
		return nil, fmt.Errorf("signing error: %w", err)
	}

	// 3. Create Request
	url := c.BaseURL + path
	// log.Println(url)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// 4. Set Headers
	req.Header.Set("KALSHI-ACCESS-KEY", c.Credentials.AccessKey)
	req.Header.Set("KALSHI-ACCESS-SIGNATURE", sig)
	req.Header.Set("KALSHI-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("Content-Type", "application/json")

	// 5. Execute
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kalshi api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

type APIResponse struct {
	Markets []types.MarketData `json:"markets"`
	Cursor  string             `json:"cursor"`
}

type MarketsResponse struct {
	Markets []types.SimplifiedMarket `json:"markets"`
	Cursor  string                   `json:"cursor"`
}

type EventsAPIResponse struct {
	Events []types.EventData `json:"events"`
	Cursor string            `json:"cursor"`
}

type EventsResponse struct {
	Events     []types.SimplifiedEvent `json:"events"`
	Cursor     string                  `json:"cursor"`
	Milestones []interface{}           `json:"milestones"`
}

func (c *Client) GetBalance() (int64, error) {
	data, err := c.DoRequest("GET", "/trade-api/v2/portfolio/balance", nil)
	if err != nil {
		return 0, err
	}

	var res BalanceResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, err
	}

	return res.Balance, nil
}

func (c *Client) GetMarkets(limit int, cursor string, mveFilter string, minCloseTs int64, maxCloseTs int64) (*MarketsResponse, error) {
	// Build query parameters
	path := "/trade-api/v2/markets"
	params := []string{}

	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}

	if cursor != "" {
		params = append(params, fmt.Sprintf("cursor=%s", url.QueryEscape(cursor)))
	}

	if mveFilter != "" {
		params = append(params, fmt.Sprintf("mve_filter=%s", url.QueryEscape(mveFilter)))
	}

	// Set status to open to ensure markets are currently open for betting
	params = append(params, "status=open")

	// Use provided timestamps or calculate defaults
	var minCloseTime, maxCloseTime int64
	if minCloseTs > 0 && maxCloseTs > 0 {
		minCloseTime = minCloseTs
		maxCloseTime = maxCloseTs
	} else if minCloseTs > 0 {
		// Only min provided
		minCloseTime = minCloseTs
		maxCloseTime = time.Now().Add(5 * 365 * 24 * time.Hour).Unix() // 5 years default cap
	} else {
		// Default: markets should settle between 12 hours and 7 days from now
		now := time.Now()
		minCloseTime = now.Add(12 * time.Hour).Unix()
		maxCloseTime = now.Add(7 * 24 * time.Hour).Unix()
	}

	// Format as Unix epoch seconds
	params = append(params, fmt.Sprintf("min_close_ts=%d", minCloseTime))
	params = append(params, fmt.Sprintf("max_close_ts=%d", maxCloseTime))

	if len(params) > 0 {
		path = path + "?" + strings.Join(params, "&")
	}

	// log.Printf("[Kalshi] Fetching markets with path: %s", path)
	data, err := c.DoRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var fullResponse APIResponse
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return nil, err
	}

	simplified := make([]types.SimplifiedMarket, len(fullResponse.Markets))

	for i, m := range fullResponse.Markets {
		simplified[i] = types.SimplifiedMarket{
			Ticker:        m.Ticker,
			EventTicker:   m.EventTicker,
			Title:         m.Title,
			Subtitle:      m.Subtitle,
			NoBidDollars:  m.YesBidDollars,
			YesBidDollars: m.NoBidDollars,
			YesAsk:        m.YesAsk,
			NoAsk:         m.NoAsk,
			YesSubTitle:   m.YesSubTitle,
			NoSubTitle:    m.NoSubTitle,
			Status:        m.Status,
			CloseTime:     m.CloseTime,
			YesAskDollars: m.YesAskDollars,
			NoAskDollars:  m.NoAskDollars,
		}
	}

	return &MarketsResponse{
		Markets: simplified,
		Cursor:  fullResponse.Cursor,
	}, nil
}

func (c *Client) GetEvent(eventTicker string) (*types.SimplifiedEvent, error) {
	path := fmt.Sprintf("/trade-api/v2/events/%s?with_nested_markets=true", url.PathEscape(eventTicker))

	data, err := c.DoRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var fullResponse struct {
		Event types.EventData `json:"event"`
	}
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return nil, err
	}

	e := fullResponse.Event

	// Simplify nested markets
	simplifiedMarkets := make([]types.SimplifiedMarket, len(e.Markets))
	for j, m := range e.Markets {
		simplifiedMarkets[j] = types.SimplifiedMarket{
			Ticker:        m.Ticker,
			EventTicker:   m.EventTicker,
			Title:         m.Title,
			Subtitle:      m.Subtitle,
			NoBidDollars:  m.YesBidDollars,
			YesBidDollars: m.NoBidDollars,
			YesAsk:        m.YesAsk,
			NoAsk:         m.NoAsk,
			YesSubTitle:   m.YesSubTitle,
			NoSubTitle:    m.NoSubTitle,
			Status:        m.Status,
			CloseTime:     m.CloseTime,
			YesAskDollars: m.YesAskDollars,
			NoAskDollars:  m.NoAskDollars,
		}
	}

	simplifiedEvent := &types.SimplifiedEvent{
		AvailableOnBrokers:   e.AvailableOnBrokers,
		Category:             e.Category,
		CollateralReturnType: e.CollateralReturnType,
		EventTicker:          e.EventTicker,
		ExpirationTime:       e.ExpirationTime,
		MutuallyExclusive:    e.MutuallyExclusive,
		SeriesTicker:         e.SeriesTicker,
		StrikePeriod:         e.StrikePeriod,
		SubTitle:             e.SubTitle,
		Title:                e.Title,
		Markets:              simplifiedMarkets,
	}

	return simplifiedEvent, nil
}

func (c *Client) GetEvents(limit int, cursor string) (*EventsResponse, error) {
	// Limit validation - max 200
	if limit > 200 {
		limit = 200
	}
	if limit <= 0 {
		limit = 100 // default
	}

	// Build query parameters
	path := "/trade-api/v2/events"
	params := []string{}
	// log.Println("test")

	params = append(params, fmt.Sprintf("limit=%d", limit))

	// Add with_nested_markets=true to get markets inline and avoid N+1 queries
	params = append(params, "with_nested_markets=true")

	// log.Printf("Cursor: %s", cursor)
	if cursor != "" {
		params = append(params, fmt.Sprintf("cursor=%s", url.QueryEscape(cursor)))
	}

	// Set min_close_ts to current timestamp
	minCloseTs := time.Now().Unix()
	params = append(params, fmt.Sprintf("min_close_ts=%d", minCloseTs))

	if len(params) > 0 {
		path = path + "?" + strings.Join(params, "&")
	}

	// log.Println(path)
	data, err := c.DoRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var fullResponse EventsAPIResponse
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return nil, err
	}

	// Simplify the events to only return required fields
	simplified := make([]types.SimplifiedEvent, len(fullResponse.Events))
	for i, e := range fullResponse.Events {
		// Simplify nested markets
		simplifiedMarkets := make([]types.SimplifiedMarket, len(e.Markets))
		for j, m := range e.Markets {
			simplifiedMarkets[j] = types.SimplifiedMarket{
				Ticker:        m.Ticker,
				EventTicker:   m.EventTicker,
				Title:         m.Title,
				NoBidDollars:  m.YesBidDollars,
				YesBidDollars: m.NoBidDollars,
				YesAsk:        m.YesAsk,
				NoAsk:         m.NoAsk,
				YesSubTitle:   m.YesSubTitle,
				NoSubTitle:    m.NoSubTitle,
				Status:        m.Status,
				CloseTime:     m.CloseTime,
				YesAskDollars: m.YesAskDollars,
				NoAskDollars:  m.NoAskDollars,
			}
		}

		simplified[i] = types.SimplifiedEvent{
			AvailableOnBrokers:   e.AvailableOnBrokers,
			Category:             e.Category,
			CollateralReturnType: e.CollateralReturnType,
			EventTicker:          e.EventTicker,
			ExpirationTime:       e.ExpirationTime,
			MutuallyExclusive:    e.MutuallyExclusive,
			SeriesTicker:         e.SeriesTicker,
			StrikePeriod:         e.StrikePeriod,
			SubTitle:             e.SubTitle,
			Title:                e.Title,
			Markets:              simplifiedMarkets,
		}
	}

	return &EventsResponse{
		Events:     simplified,
		Cursor:     fullResponse.Cursor,
		Milestones: []any{},
	}, nil
}

func (c *Client) GetMarketsByEvent(eventTicker string, limit int, cursor string, mveFilter string, minCloseTs int64, maxCloseTs int64) (*MarketsResponse, error) {
	// Build query parameters
	path := "/trade-api/v2/markets"
	params := []string{}

	// Add event_ticker as required parameter
	params = append(params, fmt.Sprintf("event_ticker=%s", url.QueryEscape(eventTicker)))

	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}

	if cursor != "" {
		params = append(params, fmt.Sprintf("cursor=%s", url.QueryEscape(cursor)))
	}

	params = append(params, "mve_filter=exclude")

	// Use provided timestamps or calculate defaults
	var minCloseTime, maxCloseTime int64
	if minCloseTs > 0 && maxCloseTs > 0 {
		minCloseTime = minCloseTs
		maxCloseTime = maxCloseTs
	} else if minCloseTs > 0 {
		// Only min provided
		minCloseTime = minCloseTs
		maxCloseTime = time.Now().Add(5 * 365 * 24 * time.Hour).Unix() // 5 years default cap
	} else {
		// Default: markets should settle between 12 hours and 7 days from now
		now := time.Now()
		minCloseTime = now.Add(12 * time.Hour).Unix()
		maxCloseTime = now.Add(7 * 24 * time.Hour).Unix()
	}

	// Format as Unix epoch seconds
	params = append(params, fmt.Sprintf("min_close_ts=%d", minCloseTime))
	params = append(params, fmt.Sprintf("max_close_ts=%d", maxCloseTime))

	if len(params) > 0 {
		path = path + "?" + strings.Join(params, "&")
	}

	// log.Printf("[Kalshi] Fetching markets by event with path: %s", path)
	data, err := c.DoRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var fullResponse APIResponse
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return nil, err
	}

	simplified := make([]types.SimplifiedMarket, len(fullResponse.Markets))

	for i, m := range fullResponse.Markets {
		simplified[i] = types.SimplifiedMarket{
			Ticker:        m.Ticker,
			EventTicker:   m.EventTicker,
			Title:         m.Title,
			Subtitle:      m.Subtitle,
			NoBidDollars:  m.YesBidDollars,
			YesBidDollars: m.NoBidDollars,
			YesAsk:        m.YesAsk,
			NoAsk:         m.NoAsk,
			YesSubTitle:   m.YesSubTitle,
			NoSubTitle:    m.NoSubTitle,
			Status:        m.Status,
			CloseTime:     m.CloseTime,
			YesAskDollars: m.YesAskDollars,
			NoAskDollars:  m.NoAskDollars,
		}
	}

	// log.Printf("Cursor 123: %s", fullResponse.Cursor)
	return &MarketsResponse{
		Markets: simplified,
		Cursor:  fullResponse.Cursor,
	}, nil
}

// GetMarketsForEventNextMonth retrieves all markets for a specific event that close within the next month.
// It handles pagination automatically to return the complete list.
func (c *Client) GetMarketsForEventNextMonth(eventTicker string) ([]types.SimplifiedMarket, error) {
	var allMarkets []types.SimplifiedMarket
	cursor := ""
	limit := 100 // Maximize batch size for efficiency

	// Time window: Now until 1 month from now
	now := time.Now()
	minCloseTs := now.Unix()
	maxCloseTs := now.AddDate(0, 1, 0).Unix() // Add 1 month

	for {
		// We use GetMarketsByEvent which already handles the API call structure
		// Passing "" for mveFilter to use the default behavior or the function's internal hardcoding
		// Note: GetMarketsByEvent currently forces "mve_filter=exclude" internally.
		resp, err := c.GetMarketsByEvent(eventTicker, limit, cursor, "", minCloseTs, maxCloseTs)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch markets page: %w", err)
		}

		allMarkets = append(allMarkets, resp.Markets...)

		if resp.Cursor == "" {
			break
		}
		cursor = resp.Cursor
	}

	return allMarkets, nil
}
