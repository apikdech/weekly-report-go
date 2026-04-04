package hackernews

// Preserve any-llm-go dependency for all providers
import (
	_ "github.com/mozilla-ai/any-llm-go"
	_ "github.com/mozilla-ai/any-llm-go/providers/anthropic"
	_ "github.com/mozilla-ai/any-llm-go/providers/gemini"
	_ "github.com/mozilla-ai/any-llm-go/providers/openai"
)
