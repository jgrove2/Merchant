package slm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"backend/internal/db"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Service defines the interface for the SLM service
type Service interface {
	CompareMarkets(source, target db.Market) (*ComparisonResult, error)
}

// ComparisonResult represents the JSON output from the SLM
type ComparisonResult struct {
	MarketID         string  `json:"market_id"`
	EventID          string  `json:"event_id"`
	ComparedMarketID string  `json:"compared_market_id"`
	ComparedEventID  string  `json:"compared_event_id"`
	Reason           string  `json:"reason"`
	SourceYes        *string `json:"source_yes"` // "target_yes", "target_no", or null
	SourceNo         *string `json:"source_no"`  // "target_yes", "target_no", or null
}

type slmService struct {
	llm llms.Model
}

// NewService initializes a new SLM service using the OpenAI adapter
// This is compatible with local runners like llama.cpp server
func NewService(modelName string) (Service, error) {
	// 1. Determine Base URL
	baseURL := os.Getenv("SLM_URL")
	if baseURL == "" {
		// Default to local dev docker-compose setup
		baseURL = "http://localhost:8088/v1"
	}

	log.Printf("Initializing SLM service with model: %s at %s", modelName, baseURL)

	// 2. Validate URL
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("invalid SLM_URL: %w", err)
	}

	// 3. Initialize OpenAI Client (points to local llama.cpp)
	// We need a dummy token because the client requires it, even if the server doesn't.
	llm, err := openai.New(
		openai.WithModel(modelName),
		openai.WithBaseURL(baseURL),
		openai.WithToken("dummy-token"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai/slm client: %w", err)
	}

	return &slmService{llm: llm}, nil
}

func (s *slmService) CompareMarkets(source, target db.Market) (*ComparisonResult, error) {
	reasoningSystemPrompt, reasoningUserPrompt := s.buildReasoningPrompts(source, target)

	ctx := context.Background()
	log.Printf("[SLM] Reasoning System Prompt:\n%s\n", reasoningSystemPrompt)
	log.Printf("[SLM] Reasoning User Prompt:\n%s\n", reasoningUserPrompt)

	// Turn 1: Reasoning
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, reasoningSystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, reasoningUserPrompt),
	}

	reasoningResp, err := s.llm.GenerateContent(ctx, messages, llms.WithTemperature(0.0))
	if err != nil {
		return nil, fmt.Errorf("SLM reasoning generation failed: %w", err)
	}
	reasoningCompletion := reasoningResp.Choices[0].Content
	log.Printf("[SLM] Reasoning Output:\n%s\n", reasoningCompletion)

	// Turn 2: JSON Extraction
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, reasoningCompletion))
	jsonUserPrompt := s.buildJSONPrompt()
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, jsonUserPrompt))

	jsonResp, err := s.llm.GenerateContent(ctx, messages, llms.WithTemperature(0.0))
	if err != nil {
		return nil, fmt.Errorf("SLM JSON generation failed: %w", err)
	}
	jsonCompletion := jsonResp.Choices[0].Content
	log.Printf("[SLM] JSON Output:\n%s\n", jsonCompletion)

	// Clean up response to ensure valid JSON
	// Sometimes models add markdown blocks ```json ... ```
	cleaned := strings.TrimSpace(jsonCompletion)
	if idx := strings.Index(cleaned, "{"); idx != -1 {
		cleaned = cleaned[idx:]
	}
	if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
		cleaned = cleaned[:idx+1]
	}

	var result ComparisonResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse SLM JSON output: %w. Output was: %s", err, jsonCompletion)
	}

	// Ensure IDs are set correctly (in case model hallucinated them)
	result.MarketID = target.ExternalID
	result.EventID = target.EventTicker
	result.ComparedMarketID = source.ExternalID
	result.ComparedEventID = source.EventTicker

	// Validate the result format (SourceYes/SourceNo fields)
	if err := validateResult(&result); err != nil {
		// Log but continue (validateResult now only nullifies invalid fields and shouldn't return error in practice for this usecase, but we keep the error return in signature just in case)
		// Actually, per instructions: "effectively always return nil error".
		// But if we did return an error, we should log it.
		// However, validateResult below is being modified to not return error for values.
		// Let's just log if err happens (it won't).
		log.Printf("SLM output validation warning: %v. Output was: %s", err, jsonCompletion)
	}

	return &result, nil
}

func validateResult(r *ComparisonResult) error {
	validValues := map[string]bool{
		"target_yes": true,
		"target_no":  true,
	}

	if r.SourceYes != nil {
		if !validValues[*r.SourceYes] {
			// Instead of erroring, set to nil
			r.SourceYes = nil
		}
	}
	if r.SourceNo != nil {
		if !validValues[*r.SourceNo] {
			// Instead of erroring, set to nil
			r.SourceNo = nil
		}
	}
	return nil
}

func (s *slmService) buildReasoningPrompts(source, target db.Market) (string, string) {
	sourceDesc := source.Description
	if source.YesSubTitle != "" {
		sourceDesc = fmt.Sprintf("%s (%s)", source.Title, source.YesSubTitle)
	} else {
		sourceDesc = source.Title
	}

	targetDesc := target.Description
	if target.YesSubTitle != "" {
		targetDesc = fmt.Sprintf("%s (%s)", target.Title, target.YesSubTitle)
	} else {
		targetDesc = target.Title
	}

	systemPrompt := `You are a logical reasoning engine specialized in prediction market implications.
Task: Determine if the outcome of a "Source" market logically necessitates a specific outcome in a "Target" market.

CRITICAL DISTINCTION:
- You must distinguish between CORRELATION (likely to happen) and LOGICAL NECESSITY (must happen).
- Only conclude a definite outcome if the outcome is a LOGICAL NECESSITY based on the definitions of the events.
- If the relationship is merely correlational (e.g. "Stock A going up usually means Stock B goes up"), you MUST conclude no necessity.

Rules:
1. Analyze if Source=YES implies Target=YES (NECESSITY).
2. Analyze if Source=YES implies Target=NO (NECESSITY).
3. Analyze if Source=NO implies Target=YES (NECESSITY).
4. Analyze if Source=NO implies Target=NO (NECESSITY).
5. If the outcome is uncertain, not guaranteed, or merely correlated, state that there is no logical necessity.
6. Check for MUTUAL EXCLUSIVITY: If Source and Target describe different outcomes for the same unique position (e.g. Winner, Top Rank, Next CEO), then Source=YES implies Target=NO.

Constraint Examples:
- "Total > 10" implies "Total > 5" (NECESSITY).
- "A wins" implies "B loses" (if mutually exclusive) (NECESSITY).
- "Inflation goes up" implies "Rates go up" (CORRELATION - No necessity).
- 'Song A is #1' implies 'Song B is NOT #1' (NECESSITY - Mutually Exclusive).
- Specific dates/values must be strictly compared.

DO NOT output JSON. Provide a step-by-step logical analysis.`

	userPrompt := fmt.Sprintf(`Source Bet:
Category: %s
Market: %s

Target Bet:
Category: %s
Market: %s

Please provide a step-by-step logical analysis of whether the Source outcome necessitates the Target outcome.`,
		source.Category, sourceDesc,
		target.Category, targetDesc)

	return systemPrompt, userPrompt
}

func (s *slmService) buildJSONPrompt() string {
	return `Based on the above analysis, map the logical implications to strict JSON.

JSON Schema:
{
  "reason": "A summary of the logic",
  "source_yes": "target_yes" | "target_no" | null,
  "source_no": "target_yes" | "target_no" | null
}

Constraints:
- "source_yes": The necessary outcome of the Target market if the Source market resolves to YES. Null if no necessity.
- "source_no": The necessary outcome of the Target market if the Source market resolves to NO. Null if no necessity.
- "reason": A brief summary string explaining the logic.`
}
