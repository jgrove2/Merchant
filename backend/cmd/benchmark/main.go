package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"backend/internal/llm"
)

// BenchmarkCase represents a single test case for benchmarking
type BenchmarkCase struct {
	SourceTitle       string  `json:"source_title"`
	SourceYesSub      string  `json:"source_yes_sub"`
	SourceCategory    string  `json:"source_category"`
	TargetTitle       string  `json:"target_title"`
	TargetYesSub      string  `json:"target_yes_sub"`
	TargetCategory    string  `json:"target_category"`
	ExpectedSourceYes *string `json:"expected_source_yes"`
	ExpectedSourceNo  *string `json:"expected_source_no"`
}

// Result represents the outcome of a benchmark run
type Result struct {
	Case             BenchmarkCase
	Matches          bool
	Error            string
	ActualYesMapping *string
	ActualNoMapping  *string
	Duration         time.Duration
	TokenUsage       llm.TokenUsage
}

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
)

func main() {
	modelName := flag.String("model", "openai/gpt-oss-20b", "Model name to use for benchmarking")
	casesFile := flag.String("cases", "benchmark_cases.json", "Path to the JSON file containing test cases")
	reasoningEffort := flag.String("reasoning-effort", "", "Reasoning effort (e.g., 'low', 'medium', 'high') - defaults to empty/none")
	temperature := flag.Float64("temperature", 1.0, "Sampling temperature to use (default: 1.0)")
	topP := flag.Float64("top-p", -1.0, "Top-p sampling (nucleus sampling) - set to -1 to disable (default: -1)")
	flag.Parse()

	fmt.Printf("Running benchmark on model: %s\n", *modelName)
	if *reasoningEffort != "" {
		fmt.Printf("Reasoning effort: %s\n", *reasoningEffort)
	}
	fmt.Printf("Temperature: %.2f\n", *temperature)
	if *topP != -1.0 {
		fmt.Printf("Top-P: %.2f\n", *topP)
	}
	fmt.Printf("Loading cases from: %s\n", *casesFile)

	// Load test cases
	casesData, err := os.ReadFile(*casesFile)
	if err != nil {
		log.Fatalf("Failed to read cases file: %v", err)
	}

	var cases []BenchmarkCase
	if err := json.Unmarshal(casesData, &cases); err != nil {
		log.Fatalf("Failed to parse cases file: %v", err)
	}

	fmt.Printf("Loaded %d test cases.\n\n", len(cases))

	// Run benchmarks
	var results []Result
	passCount := 0

	llmCfg := llm.Config{
		Provider:        llm.ProviderGroq,
		Model:           *modelName,
		ReasoningEffort: *reasoningEffort,
		Temperature:     *temperature,
	}
	if *topP != -1.0 {
		llmCfg.TopP = topP
	}
	llmManager := llm.NewManager(llmCfg)

	for i, c := range cases {
		source := llm.Market{
			Title:       c.SourceTitle,
			YesSubTitle: c.SourceYesSub,
			// Category not in llm.Market struct yet, ignoring
			ID:      fmt.Sprintf("mkt-%s", generateRandomID()),
			EventId: fmt.Sprintf("evt-%s", generateRandomID()),
		}
		target := llm.Market{
			Title:       c.TargetTitle,
			YesSubTitle: c.TargetYesSub,
			// Category not in llm.Market struct yet, ignoring
			ID:      fmt.Sprintf("mkt-%s", generateRandomID()),
			EventId: fmt.Sprintf("evt-%s", generateRandomID()),
		}

		//fmt.Printf("[%d/%d] Comparing: %s vs %s... ", i+1, len(cases), c.SourceTitle, c.TargetTitle)

		start := time.Now()
		res, err := llmManager.CompareMarkets(context.Background(), source, target)
		duration := time.Since(start)

		r := Result{
			Case:     c,
			Duration: duration,
		}

		if err != nil {
			r.Error = err.Error()
			//fmt.Printf("ERROR (%v)\n", duration)
			fmt.Printf("%s[%d/%d] ERROR: %s vs %s (%v)%s\n", ColorRed, i+1, len(cases), c.SourceTitle, c.TargetTitle, duration, ColorReset)
			fmt.Printf("  Error details: %v\n", err)

			fmt.Printf("\n  %sPrompt Used:%s\n", ColorYellow, ColorReset)
			fmt.Println(llm.ComparisonPrompt(source, target))
			fmt.Println("--------------------------------------------------")
		} else {
			r.ActualYesMapping = res.Mapping.PrimaryYes
			r.ActualNoMapping = res.Mapping.PrimaryNo
			r.TokenUsage = res.Usage

			// Compare results
			match := true
			if !ptrEqual(c.ExpectedSourceYes, res.Mapping.PrimaryYes) {
				match = false
			}
			if !ptrEqual(c.ExpectedSourceNo, res.Mapping.PrimaryNo) {
				match = false
			}
			r.Matches = match
			if match {
				passCount++
				//fmt.Printf("PASSED (%v)\n", duration)
			} else {
				//fmt.Printf("FAILED (%v)\n", duration)
				fmt.Printf("%s[%d/%d] FAILED: %s vs %s (%v)%s\n", ColorRed, i+1, len(cases), c.SourceTitle, c.TargetTitle, duration, ColorReset)
				fmt.Printf("  Mismatch:\n")
				if !ptrEqual(c.ExpectedSourceYes, res.Mapping.PrimaryYes) {
					fmt.Printf("    SourceYes: Expected %v, Got %v\n", ptrStr(c.ExpectedSourceYes), ptrStr(res.Mapping.PrimaryYes))
				}
				if !ptrEqual(c.ExpectedSourceNo, res.Mapping.PrimaryNo) {
					fmt.Printf("    SourceNo:  Expected %v, Got %v\n", ptrStr(c.ExpectedSourceNo), ptrStr(res.Mapping.PrimaryNo))
				}

				fmt.Printf("\n  %sPrompt Used:%s\n", ColorYellow, ColorReset)
				fmt.Println(llm.ComparisonPrompt(source, target))
				fmt.Println("--------------------------------------------------")
			}
		}
		results = append(results, r)
	}

	// Calculate statistics
	var durations []time.Duration
	var totalPromptTokens, totalCompletionTokens, totalTokens, successfulRequests int

	for _, r := range results {
		if r.Error == "" { // Only count successful requests for timing
			durations = append(durations, r.Duration)
			totalPromptTokens += r.TokenUsage.PromptTokens
			totalCompletionTokens += r.TokenUsage.CompletionTokens
			totalTokens += r.TokenUsage.TotalTokens
			successfulRequests++
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var avgDuration, medianDuration, p90Duration time.Duration
	if len(durations) > 0 {
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration = totalDuration / time.Duration(len(durations))
		medianDuration = durations[len(durations)/2]
		p90Index := int(math.Ceil(float64(len(durations))*0.9)) - 1
		if p90Index < 0 {
			p90Index = 0
		}
		p90Duration = durations[p90Index]
	}

	passRate := 0.0
	if len(cases) > 0 {
		passRate = float64(passCount) / float64(len(cases)) * 100.0
	}

	fmt.Printf("\n=== Benchmark Summary ===\n")
	fmt.Printf("Model: %s\n", *modelName)
	fmt.Printf("Total Cases: %d\n", len(cases))
	fmt.Printf("%sPassed: %d (%.2f%%)%s\n", ColorGreen, passCount, passRate, ColorReset)
	fmt.Printf("%sFailed: %d%s\n", ColorRed, len(cases)-passCount, ColorReset)
	fmt.Printf("\n--- Timing Metrics (Successful Calls) ---\n")
	if len(durations) > 0 {
		fmt.Printf("Average: %v\n", avgDuration)
		fmt.Printf("Median:  %v\n", medianDuration)
		fmt.Printf("P90:     %v\n", p90Duration)
	} else {
		fmt.Printf("No successful calls to measure timing.\n")
	}

	fmt.Printf("\n--- Token Usage ---\n")
	if successfulRequests > 0 {
		fmt.Printf("Total Input:  %d\n", totalPromptTokens)
		fmt.Printf("Total Output: %d\n", totalCompletionTokens)
		fmt.Printf("Total:        %d\n", totalTokens)
		fmt.Printf("Avg per Req:  %d\n", totalTokens/successfulRequests)
	} else {
		fmt.Printf("No successful calls to measure token usage.\n")
	}
}

func generateRandomID() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(bytes)
}

func ptr(s string) *string {
	return &s
}

func ptrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrStr(s *string) string {
	if s == nil {
		return "null"
	}
	return fmt.Sprintf("\"%s\"", *s)
}

// Temporary unused variable to keep imports happy if we strip logic later
var _ = context.Background
var _ = json.Unmarshal
var _ = log.Println
