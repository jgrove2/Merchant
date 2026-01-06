package llm

import (
	"context"
	"errors"
	"os"

	"backend/internal/groq"
)

// ProviderType defines the supported LLM providers
type ProviderType string

const (
	ProviderGroq ProviderType = "groq"
)

// Config holds the configuration for the LLM manager
type Config struct {
	Provider        ProviderType
	Model           string
	APIKey          string
	ReasoningEffort string
	Temperature     float64
	TopP            *float64
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
	if cfg.Provider == "" {
		cfg.Provider = ProviderGroq
	}
	if cfg.Model == "" {
		cfg.Model = "openai/gpt-oss-20b"
	}
	return &Manager{config: cfg}
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
		return &groqService{client: client, model: m.config.Model, reasoningEffort: m.config.ReasoningEffort, temperature: m.config.Temperature, topP: m.config.TopP}, nil
	default:
		return nil, errors.New("unsupported provider")
	}
}

// groqService implements the Service interface for Groq
type groqService struct {
	client          *groq.Client
	model           string
	reasoningEffort string
	temperature     float64
	topP            *float64
}

func (s *groqService) Generate(ctx context.Context, prompt string) (*GenerateResponse, error) {
	content, usage, err := s.client.SimplePrompt(ctx, s.model, prompt, s.reasoningEffort, s.temperature, s.topP)
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
