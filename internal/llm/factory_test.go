package llm

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/config"
)

func TestNewProvider_Gemini(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "gemini",
		LLMAPIKey:   "test-key",
		LLMModel:    "gemini-pro",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}
}

func TestNewProvider_Gemini_EmptyProvider(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "", // empty defaults to gemini
		LLMAPIKey:   "test-key",
		LLMModel:    "gemini-pro",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider for empty provider string")
	}
}

func TestNewProvider_Gemini_MissingKey(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "gemini",
		LLMAPIKey:   "",
		LLMModel:    "gemini-pro",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestNewProvider_OpenAI(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "openai",
		LLMAPIKey:   "test-key",
		LLMModel:    "gpt-4",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}
}

func TestNewProvider_OpenAI_MissingKey(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "openai",
		LLMAPIKey:   "",
		LLMModel:    "gpt-4",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestNewProvider_OpenAI_WithBaseURL(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "openai",
		LLMAPIKey:   "test-key",
		LLMModel:    "gpt-4",
		LLMBaseURL:  "https://api.openrouter.ai/api/v1",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}
	// Provider is created successfully with custom base URL
}

func TestNewProvider_Anthropic(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "anthropic",
		LLMAPIKey:   "test-key",
		LLMModel:    "claude-3-5-sonnet",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}
}

func TestNewProvider_CaseInsensitive(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "OPENAI", // uppercase
		LLMAPIKey:   "test-key",
		LLMModel:    "gpt-4",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider for uppercase provider name")
	}
}

func TestNewProvider_UnknownDefaultsToGemini(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "unknown-provider",
		LLMAPIKey:   "test-key",
		LLMModel:    "model",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Error("expected non-nil provider when LLM_PROVIDER is unrecognized (defaults to Gemini)")
	}
}
