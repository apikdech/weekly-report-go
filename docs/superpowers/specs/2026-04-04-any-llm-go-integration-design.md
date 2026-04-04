# Generic LLM Provider with any-llm-go Library - Design

**Date:** 2026-04-04  
**Scope:** Refactor Hacker News LLM integration to use mozilla-ai/any-llm-go library for multi-provider support

## Overview

Instead of building and maintaining our own provider abstraction layer, we will leverage Mozilla's `any-llm-go` library which provides a unified interface for 10+ LLM providers including Gemini, OpenAI, Anthropic, DeepSeek, Mistral, Ollama, Groq, and more.

## Goals

1. Support multiple LLM providers with minimal code changes
2. Leverage battle-tested library for provider abstractions
3. Better error handling including automatic rate limit detection
4. Maintain existing behavior (log warnings on errors, skip section if no provider)
5. Simple configuration via environment variables

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│            internal/sources/hackernews/hackernews.go          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Uses any-llm-go Library                    ││
│  │  import "github.com/mozilla-ai/any-llm-go"                ││
│  │  import "github.com/mozilla-ai/any-llm-go/providers/..."││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              any-llm-go Library (External)                  │
│         ┌───────────┐  ┌───────────┐  ┌───────────┐         │
│         │  Gemini   │  │  OpenAI   │  │ Anthropic │         │
│         │  Provider │  │  Provider │  │  Provider │         │
│         └───────────┘  └───────────┘  └───────────┘         │
│         ┌───────────┐  ┌───────────┐  ┌───────────┐         │
│         │ DeepSeek  │  │  Mistral  │  │   Groq    │         │
│         │  Provider │  │  Provider │  │  Provider │         │
│         └───────────┘  └───────────┘  └───────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## Interface Design

### Provider Factory Function

```go
// internal/llm/factory.go

package llm

import (
    "fmt"
    "strings"

    "github.com/mozilla-ai/any-llm-go"
    "github.com/apikdech/gws-weekly-report/internal/config"
    
    // Import providers we want to support
    "github.com/mozilla-ai/any-llm-go/providers/anthropic"
    "github.com/mozilla-ai/any-llm-go/providers/gemini"
    "github.com/mozilla-ai/any-llm-go/providers/openai"
)

// NewProvider creates an any-llm-go Provider based on configuration
func NewProvider(cfg *config.Config) (anyllm.Provider, error) {
    provider := strings.ToLower(cfg.LLMProvider)

    switch provider {
    case "gemini", "":
        return gemini.New(anyllm.WithAPIKey(cfg.LLMAPIKey))
    case "openai":
        return openai.New(anyllm.WithAPIKey(cfg.LLMAPIKey))
    case "anthropic":
        return anthropic.New(anyllm.WithAPIKey(cfg.LLMAPIKey))
    default:
        return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLMProvider)
    }
}
```

### HackerNews Source Integration

```go
// internal/sources/hackernews/hackernews.go

type Source struct {
    provider   anyllm.Provider  // From any-llm-go library
    model      string
    highlights []pipeline.TechHighlight
}

func NewSource(provider anyllm.Provider, model string) *Source {
    return &Source{provider: provider, model: model}
}

func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
    if s.provider == nil {
        log.Printf("[hackernews] LLM provider not configured, skipping technology section")
        return nil
    }

    // Fetch articles...
    articles, err := s.fetchHNArticles(ctx, week)
    if err != nil {
        log.Printf("[hackernews] WARNING: Failed to fetch HN articles: %v", err)
        return nil
    }

    // Use any-llm-go for analysis
    highlights, err := s.analyzeWithLLM(ctx, articles)
    if err != nil {
        log.Printf("[hackernews] WARNING: Failed to analyze with LLM: %v", err)
        return nil
    }

    s.highlights = highlights
    return nil
}

func (s *Source) analyzeWithLLM(ctx context.Context, articles []HNArticle) ([]TechHighlight, error) {
    // Build prompt with articles
    articlesJSON, _ := json.Marshal(articles)
    fullPrompt := PromptTemplate + "\n\nArticles JSON:\n" + string(articlesJSON)

    // Use any-llm-go for completion
    response, err := s.provider.Completion(ctx, anyllm.CompletionParams{
        Model: s.model,
        Messages: []anyllm.Message{
            {Role: anyllm.RoleUser, Content: fullPrompt},
        },
    })
    if err != nil {
        return nil, err
    }

    // Parse JSON from response
    responseText := response.Choices[0].Message.Content
    jsonStr := ExtractJSONFromMarkdown(responseText)
    
    // Parse result...
}
```

## Configuration

### Environment Variables

```bash
# Provider selection: gemini | openai | anthropic (default: gemini)
LLM_PROVIDER=gemini

# API key for the selected provider
LLM_API_KEY=your-api-key-here

# Model name (provider-specific)
# Gemini: gemini-3-flash, gemini-2.5-pro, gemini-2.0-flash, etc.
# OpenAI: gpt-4, gpt-3.5-turbo, gpt-4o, etc.
# Anthropic: claude-3-5-sonnet-20241022, claude-3-opus, etc.
LLM_MODEL=gemini-3-flash
```

### Backward Compatibility

Legacy `GEMINI_API_KEY` and `GEMINI_MODEL` will be mapped to new unified config:
- If `LLM_API_KEY` not set but `GEMINI_API_KEY` is → use `GEMINI_API_KEY`
- If `LLM_MODEL` not set but `GEMINI_MODEL` is → use `GEMINI_MODEL`
- If `LLM_PROVIDER` not set but `GEMINI_API_KEY` is → default to `gemini`

## Supported Providers

With any-llm-go, we get instant support for:

| Provider | Import Path | Notes |
|----------|--------------|-------|
| Gemini | `providers/gemini` | Native Google Gemini |
| OpenAI | `providers/openai` | OpenAI + compatible APIs (OpenRouter, etc.) |
| Anthropic | `providers/anthropic` | Claude models |
| DeepSeek | `providers/deepseek` | DeepSeek models |
| Mistral | `providers/mistral` | Mistral AI |
| Groq | `providers/groq` | Fast inference |
| Ollama | `providers/ollama` | Local models |
| llama.cpp | `providers/llamacpp` | Local inference |
| z.ai | `providers/zai` | Z.ai platform |

**Note:** For initial implementation, we'll support Gemini, OpenAI, and Anthropic. Adding more is trivial (just add the import and case to factory).

## Error Handling

The any-llm-go library provides standardized error types:

```go
response, err := provider.Completion(ctx, params)
if err != nil {
    switch {
    case errors.Is(err, anyllm.ErrRateLimit):
        // Rate limited - could suggest retry with backoff
        return fmt.Errorf("rate limited: %w", err)
    case errors.Is(err, anyllm.ErrAuthentication):
        // Auth error - invalid API key
        return fmt.Errorf("authentication failed: %w", err)
    case errors.Is(err, anyllm.ErrContextLength):
        // Input too long
        return fmt.Errorf("input too long: %w", err)
    default:
        return fmt.Errorf("LLM request failed: %w", err)
    }
}
```

This gives us **automatic rate limit detection** - exactly what you need!

## Benefits Over Custom Implementation

| Aspect | Custom Implementation | any-llm-go Library |
|--------|---------------------|-------------------|
| Code to maintain | ~500+ lines (providers + types + tests) | ~50 lines (factory only) |
| Providers supported | 2 (Gemini + OpenAI-compatible) | 10+ (Gemini, OpenAI, Anthropic, etc.) |
| Rate limit handling | Manual | Automatic with typed errors |
| Testing | Extensive unit tests needed | Library already tested |
| Updates | We maintain all providers | Community maintains |
| Features | Basic only | Streaming, tools, reasoning, etc. |

## Migration: Adding New Provider

To add a new provider (e.g., DeepSeek):

1. **Import the provider**:
```go
import "github.com/mozilla-ai/any-llm-go/providers/deepseek"
```

2. **Add to factory**:
```go
case "deepseek":
    return deepseek.New(anyllm.WithAPIKey(cfg.LLMAPIKey))
```

**Done!** No other changes needed.

## Package Structure

```
internal/
├── llm/
│   ├── factory.go       # Creates any-llm-go provider instances
│   ├── util.go          # Shared prompt template + JSON extraction
│   └── factory_test.go  # Tests for provider factory
└── sources/
    └── hackernews/
        ├── hackernews.go  # Uses anyllm.Provider interface
        └── types.go       # Keep HNArticle type
```

## Success Criteria

- [x] Library integrated as dependency
- [x] Gemini provider works (same as current implementation)
- [x] OpenAI provider works via library
- [x] Anthropic provider works via library
- [x] Rate limit errors are properly detected and logged
- [x] Backward compatibility maintained
- [x] Adding new provider requires only 2 lines of code
- [x] All existing tests pass
- [x] Library error handling works correctly

## Non-Goals

- Streaming support (can be added later)
- Tool/Function calling (not needed for this use case)
- Extended thinking/reasoning display (not needed)
- All 10+ providers enabled initially (start with 3 main ones)

## Open Questions

None - this is a straightforward library integration.
