package llm

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type GenerateResponse struct {
	Content string     `json:"content"`
	Usage   TokenUsage `json:"usage"`
}

type ComparisonResponse struct {
	IsRelated bool `json:"is_related"`
	Mapping   struct {
		PrimaryYes *string `json:"primary_yes"` // "comparison_yes", "comparison_no", or null
		PrimaryNo  *string `json:"primary_no"`  // "comparison_yes", "comparison_no", or null
	} `json:"mapping"`

	// Metadata populated after LLM response
	PrimaryMarketID    string     `json:"primary_market_id,omitempty"`
	ComparisonMarketID string     `json:"comparison_market_id,omitempty"`
	Usage              TokenUsage `json:"usage,omitempty"`
}

type Market struct {
	Title       string
	YesSubTitle string
	NoSubTitle  string
	EventId     string
	ID          string
}
