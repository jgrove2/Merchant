package types

import (
	"time"
)

type MarketData struct {
	Ticker                  string       `json:"ticker"`
	EventTicker             string       `json:"event_ticker"`
	MarketType              string       `json:"market_type"`
	Title                   string       `json:"title"`
	Subtitle                string       `json:"subtitle"`
	YesSubTitle             string       `json:"yes_sub_title"`
	NoSubTitle              string       `json:"no_sub_title"`
	CreatedTime             time.Time    `json:"created_time"`
	OpenTime                time.Time    `json:"open_time"`
	CloseTime               time.Time    `json:"close_time"`
	ExpirationTime          time.Time    `json:"expiration_time"`
	LatestExpirationTime    time.Time    `json:"latest_expiration_time"`
	SettlementTimerSeconds  int          `json:"settlement_timer_seconds"`
	Status                  string       `json:"status"`
	ResponsePriceUnits      string       `json:"response_price_units"`
	YesBid                  int          `json:"yes_bid"`
	YesBidDollars           string       `json:"yes_bid_dollars"`
	YesAsk                  int          `json:"yes_ask"`
	YesAskDollars           string       `json:"yes_ask_dollars"`
	NoBid                   int          `json:"no_bid"`
	NoBidDollars            string       `json:"no_bid_dollars"`
	NoAsk                   int          `json:"no_ask"`
	NoAskDollars            string       `json:"no_ask_dollars"`
	LastPrice               int          `json:"last_price"`
	LastPriceDollars        string       `json:"last_price_dollars"`
	Volume                  int          `json:"volume"`
	Volume24h               int          `json:"volume_24h"`
	Result                  string       `json:"result"`
	CanCloseEarly           bool         `json:"can_close_early"`
	OpenInterest            int          `json:"open_interest"`
	NotionalValue           int          `json:"notional_value"`
	NotionalValueDollars    string       `json:"notional_value_dollars"`
	PreviousYesBid          int          `json:"previous_yes_bid"`
	PreviousYesBidDollars   string       `json:"previous_yes_bid_dollars"`
	PreviousYesAsk          int          `json:"previous_yes_ask"`
	PreviousYesAskDollars   string       `json:"previous_yes_ask_dollars"`
	PreviousPrice           int          `json:"previous_price"`
	PreviousPriceDollars    string       `json:"previous_price_dollars"`
	Liquidity               int          `json:"liquidity"`
	LiquidityDollars        string       `json:"liquidity_dollars"`
	ExpirationValue         string       `json:"expiration_value"`
	Category                string       `json:"category"`
	RiskLimitCents          int          `json:"risk_limit_cents"`
	TickSize                int          `json:"tick_size"`
	RulesPrimary            string       `json:"rules_primary"`
	RulesSecondary          string       `json:"rules_secondary"`
	PriceLevelStructure     string       `json:"price_level_structure"`
	PriceRanges             []PriceRange `json:"price_ranges"`
	ExpectedExpirationTime  time.Time    `json:"expected_expiration_time"`
	SettlementValue         int          `json:"settlement_value"`
	SettlementValueDollars  string       `json:"settlement_value_dollars"`
	SettlementTs            time.Time    `json:"settlement_ts"`
	FeeWaiverExpirationTime time.Time    `json:"fee_waiver_expiration_time"`
	EarlyCloseCondition     string       `json:"early_close_condition"`
	StrikeType              string       `json:"strike_type"`
	FloorStrike             float64      `json:"floor_strike"`
	CapStrike               float64      `json:"cap_strike"`
	FunctionalStrike        string       `json:"functional_strike"`
	CustomStrike            interface{}  `json:"custom_strike"`
	MveCollectionTicker     string       `json:"mve_collection_ticker"`
	MveSelectedLegs         []MveLeg     `json:"mve_selected_legs"`
	PrimaryParticipantKey   string       `json:"primary_participant_key"`
}

type PriceRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Step  string `json:"step"`
}

type MveLeg struct {
	EventTicker  string `json:"event_ticker"`
	MarketTicker string `json:"market_ticker"`
	Side         string `json:"side"`
}

type SimplifiedMarket struct {
	Ticker        string    `json:"ticker"`
	EventTicker   string    `json:"event_ticker"`
	Title         string    `json:"title"`
	Subtitle      string    `json:"subtitle"`
	YesSubTitle   string    `json:"yes_sub_title"`
	NoSubTitle    string    `json:"no_sub_title"`
	YesBidDollars string    `json:"yes_bid_dollars"`
	NoBidDollars  string    `json:"no_bid_dollars"`
	YesAsk        int       `json:"yes_ask"`
	YesAskDollars string    `json:"yes_ask_dollars"`
	NoAsk         int       `json:"no_ask"`
	NoAskDollars  string    `json:"no_ask_dollars"`
	Status        string    `json:"status"`
	Category      string    `json:"category"`
	CloseTime     time.Time `json:"close_time"`
}
