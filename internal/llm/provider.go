package llm

import (
	"strings"
)

// ProviderType identifies a built-in any-llm-go backend. Config still stores the
// wire value as a string; use ParseProviderKind when resolving.
type ProviderType int

const (
	Gemini ProviderType = iota
	OpenAI
	Anthropic
	Groq
	DeepSeek
	Ollama
)

// ParseProvider normalizes an env-style value. Empty string defaults to Gemini.
func ParseProvider(s string) ProviderType {
	switch strings.ToLower(s) {
	case "openai":
		return OpenAI
	case "anthropic":
		return Anthropic
	case "groq":
		return Groq
	case "deepseek":
		return DeepSeek
	case "ollama":
		return Ollama
	default:
		return Gemini
	}
}
