package groq

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model           string        `json:"model"`
	Messages        []ChatMessage `json:"messages"`
	Temperature     float64       `json:"temperature,omitempty"`
	MaxTokens       int           `json:"max_tokens,omitempty"`
	Stream          bool          `json:"stream,omitempty"`
	ReasoningEffort string        `json:"reasoning_effort"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
