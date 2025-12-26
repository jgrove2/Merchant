package types

// EventData represents the full event data from Kalshi API
type EventData struct {
	AvailableOnBrokers   bool         `json:"available_on_brokers"`
	BannerURL            string       `json:"banner_url"`
	Category             string       `json:"category"`
	CollateralReturnType string       `json:"collateral_return_type"`
	EndDateIso           string       `json:"end_date_iso"`
	EventTicker          string       `json:"event_ticker"`
	ExpirationTime       string       `json:"expiration_time"`
	MarketCount          int          `json:"market_count"`
	MutuallyExclusive    bool         `json:"mutually_exclusive"`
	SeriesTicker         string       `json:"series_ticker"`
	StartDateIso         string       `json:"start_date_iso"`
	StrikePeriod         string       `json:"strike_period"`
	SubTitle             string       `json:"sub_title"`
	Title                string       `json:"title"`
	Markets              []MarketData `json:"markets"`
}

// SimplifiedEvent represents a simplified version of event data for BFF
type SimplifiedEvent struct {
	AvailableOnBrokers   bool               `json:"available_on_brokers"`
	Category             string             `json:"category"`
	CollateralReturnType string             `json:"collateral_return_type"`
	EventTicker          string             `json:"event_ticker"`
	ExpirationTime       string             `json:"expiration_time"`
	MutuallyExclusive    bool               `json:"mutually_exclusive"`
	SeriesTicker         string             `json:"series_ticker"`
	StrikePeriod         string             `json:"strike_period"`
	SubTitle             string             `json:"sub_title"`
	Title                string             `json:"title"`
	Markets              []SimplifiedMarket `json:"markets,omitempty"`
}
