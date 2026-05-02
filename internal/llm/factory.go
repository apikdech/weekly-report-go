package llm

import (
	"fmt"

	"github.com/apikdech/gws-weekly-report/internal/config"
	anyllm "github.com/mozilla-ai/any-llm-go"
)

// NewProvider creates an any-llm-go Provider based on configuration
func NewProvider(cfg *config.Config) (anyllm.Provider, error) {
	if cfg.LLMAPIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required")
	}

	pType := ParseProvider(cfg.LLMProvider)

	makeFn := providerRegistry[pType]
	if makeFn == nil {
		// Defensive: registry should cover every parsed kind
		return nil, fmt.Errorf("no constructor registered for provider %v", pType)
	}

	opts := []anyllm.Option{anyllm.WithAPIKey(cfg.LLMAPIKey)}
	if (pType == OpenAI || pType == Ollama) && cfg.LLMBaseURL != "" {
		opts = append(opts, anyllm.WithBaseURL(cfg.LLMBaseURL))
	}

	return makeFn(opts...)
}
