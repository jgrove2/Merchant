package llm

import (
	"context"
	"encoding/json"
	"fmt"
	//"log"
	"strings"
)

// ComparisonPrompt generates a logic analysis prompt to determine if a Primary
// market outcome guarantees a specific Comparison market outcome.
func ComparisonPrompt(primaryInput, comparisonInput Market) string {
	var primaryDesc string
	if primaryInput.NoSubTitle != "" {
		primaryDesc = fmt.Sprintf("%s (YES: %s, NO: %s)", primaryInput.Title, primaryInput.YesSubTitle, primaryInput.NoSubTitle)
	} else {
		primaryDesc = fmt.Sprintf("%s (YES: %s)", primaryInput.Title, primaryInput.YesSubTitle)
	}

	var comparisonDesc string
	if comparisonInput.NoSubTitle != "" {
		comparisonDesc = fmt.Sprintf("%s (YES: %s, NO: %s)", comparisonInput.Title, comparisonInput.YesSubTitle, comparisonInput.NoSubTitle)
	} else {
		comparisonDesc = fmt.Sprintf("%s (YES: %s)", comparisonInput.Title, comparisonInput.YesSubTitle)
	}
	return fmt.Sprintf(`Act as a logic analyst. Evaluate two markets: Primary and Comparison.

	Determine if the Primary outcome 100%% guarantees a specific Comparison outcome. Use null if the result is not guaranteed or logically uncertain.

	### Logic Rules:
	1. Prerequisite: Primary YES requires Comparison YES (e.g., win requires enter).
	2. Exclusion: Primary YES makes Comparison NO impossible (e.g., win requires not losing).
	3. Inverse: Primary NO guarantees a Comparison result (e.g., if not entered, cannot win).

	### Schema:
	{
		"is_related": boolean,
		"mapping": {
			"primary_yes": "comparison_yes" | "comparison_no" | null,
			"primary_no": "comparison_yes" | "comparison_no" | null
		}
	}
    
    IMPORTANT: Return ONLY the JSON object. Do not include any explanations, reasoning, or markdown formatting (no `+"```"+`json blocks).

	### Input:
	Primary: %s
	Comparison: %s`, primaryDesc, comparisonDesc)
}

func (m *Manager) CompareMarkets(ctx context.Context, primaryMarket, comparisonMarket Market) (*ComparisonResponse, error) {

	comparePrompt := ComparisonPrompt(primaryMarket, comparisonMarket)

	// Call the Generate method using the service (which wraps the client)
	service, err := m.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LLM provider: %w", err)
	}

	genResp, err := service.Generate(ctx, comparePrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	jsonCompletion := genResp.Content
	// log.Printf("[LLM] JSON Output:\n%s\n", jsonCompletion)

	// Clean up response to ensure valid JSON
	// Sometimes models add markdown blocks ```json ... ```
	cleaned := strings.TrimSpace(jsonCompletion)
	if idx := strings.Index(cleaned, "{"); idx != -1 {
		cleaned = cleaned[idx:]
	}
	if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
		cleaned = cleaned[:idx+1]
	}

	var result ComparisonResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON output: %w\n\n=== RAW LLM OUTPUT ===\n%s\n======================\n", err, jsonCompletion)
	}

	// Normalize "null" strings to nil pointers
	if result.Mapping.PrimaryYes != nil && *result.Mapping.PrimaryYes == "null" {
		result.Mapping.PrimaryYes = nil
	}
	if result.Mapping.PrimaryNo != nil && *result.Mapping.PrimaryNo == "null" {
		result.Mapping.PrimaryNo = nil
	}

	// Ensure IDs are set correctly (in case model hallucinated them - usually IDs are not in prompt output schema but good to set on struct)
	result.PrimaryMarketID = primaryMarket.ID
	result.ComparisonMarketID = comparisonMarket.ID
	result.Usage = genResp.Usage

	// Log validation warning if needed, but we don't return error on validation failure per previous pattern if we wanted to be lenient
	// For now, we assume the unmarshal works. Add specific validation logic here if needed.

	return &result, nil
}
