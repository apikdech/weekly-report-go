package llm

import (
	"fmt"
	"strings"

	"github.com/apikdech/gws-weekly-report/internal/config"
	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
)

// NewProvider creates an any-llm-go Provider based on configuration
func NewProvider(cfg *config.Config) (anyllm.Provider, error) {
	provider := strings.ToLower(cfg.LLMProvider)

	switch provider {
	case "gemini", "":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY required for Gemini provider")
		}
		return gemini.New(anyllm.WithAPIKey(cfg.LLMAPIKey))

	case "openai":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY required for OpenAI provider")
		}
		opts := []anyllm.Option{anyllm.WithAPIKey(cfg.LLMAPIKey)}
		if cfg.LLMBaseURL != "" {
			opts = append(opts, anyllm.WithBaseURL(cfg.LLMBaseURL))
		}
		return openai.New(opts...)

	case "anthropic":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY required for Anthropic provider")
		}
		return anthropic.New(anyllm.WithAPIKey(cfg.LLMAPIKey))

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s (supported: gemini, openai, anthropic)", cfg.LLMProvider)
	}
}
