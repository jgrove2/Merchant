package llm

import (
	"context"
	"errors"
	"os"

	"backend/internal/groq"
	"backend/internal/openrouter"
)

// ProviderType defines the supported LLM providers
type ProviderType string

const (
	ProviderGroq       ProviderType = "groq"
	ProviderOpenRouter ProviderType = "openrouter"
)

var (
	GroqModels       = []string{"openai/gpt-oss-20b"}
	OpenRouterModels = []string{
		"xiaomi/mimo-v2-flash:free",
		"tngtech/deepseek-r1t2-chimera:free",
		"x-ai/grok-4-fast",
		"qwen/qwen3-235b-a22b-2507",
		"openai/gpt-oss-safeguard-20b",
		"google/gemini-2.5-flash-lite",
	}
)

// Config holds the configuration for the LLM manager
type Config struct {
	Provider            ProviderType
	Model               string
	APIKey              string
	ReasoningEffort     string
	Temperature         float64
	TopP                *float64
	MaxCompletionTokens int
}

// Service defines the interface for LLM interactions
type Service interface {
	Generate(ctx context.Context, prompt string) (*GenerateResponse, error)
}

// Manager handles the creation of LLM connections
type Manager struct {
	config Config
}

// NewManager creates a new LLM manager with the given config
func NewManager(cfg Config) *Manager {
	if cfg.Model == "" {
		cfg.Model = "openai/gpt-oss-20b"
	}

	if cfg.Provider == "" {
		cfg.Provider = inferProvider(cfg.Model)
	}

	return &Manager{config: cfg}
}

func inferProvider(model string) ProviderType {
	for _, m := range OpenRouterModels {
		if m == model {
			return ProviderOpenRouter
		}
	}
	return ProviderGroq
}

// Connect establishes a connection to the selected provider and returns a Service
func (m *Manager) Connect() (Service, error) {
	switch m.config.Provider {
	case ProviderGroq:
		apiKey := m.config.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("GROQ_API_KEY")
		}
		if apiKey == "" {
			return nil, errors.New("GROQ_API_KEY is required")
		}

		client, err := groq.NewClient(apiKey)
		if err != nil {
			return nil, err
		}
		return &groqService{client: client, model: m.config.Model}, nil
	case ProviderOpenRouter:
		apiKey := m.config.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		}
		if apiKey == "" {
			return nil, errors.New("OPENROUTER_API_KEY is required")
		}

		client, err := openrouter.NewClient(apiKey)
		if err != nil {
			return nil, err
		}
		return &openRouterService{client: client, model: m.config.Model}, nil
	default:
		return nil, errors.New("unsupported provider")
	}
}

// groqService implements the Service interface for Groq
type groqService struct {
	client *groq.Client
	model  string
}

func (s *groqService) Generate(ctx context.Context, prompt string) (*GenerateResponse, error) {
	content, usage, err := s.client.SimplePrompt(ctx, s.model, prompt, "", 0, nil, 0)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{
		Content: content,
		Usage: TokenUsage{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
		},
	}, nil
}

// openRouterService implements the Service interface for OpenRouter
type openRouterService struct {
	client *openrouter.Client
	model  string
}

func (s *openRouterService) Generate(ctx context.Context, prompt string) (*GenerateResponse, error) {
	content, usage, err := s.client.SimplePrompt(ctx, s.model, prompt, "", 0, nil, 0)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{
		Content: content,
		Usage: TokenUsage{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
		},
	}, nil
}
