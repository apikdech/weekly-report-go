# Generic LLM Provider Interface Design

**Date:** 2026-04-04  
**Scope:** Refactor Hacker News Gemini analysis to support multiple LLM providers via generic interface

## Overview

Currently, the Hacker News source has hardcoded Gemini integration for analyzing and summarizing articles. When rate limiting occurs, users have no easy way to switch to alternative providers (OpenAI, OpenRouter, Fireworks.ai, etc.).

This design introduces a generic LLM provider interface that abstracts the differences between various AI providers, making it trivial to switch providers via configuration and add new providers in the future.

## Goals

1. Easy provider switching via environment config (no code changes)
2. Support both native APIs (Gemini) and OpenAI-compatible APIs (OpenRouter, Fireworks.ai, Kimi, OpenAI)
3. Easy extensibility - adding a new provider should require minimal code
4. Maintain existing behavior (log warnings on errors, don't fail the report)

## Non-Goals

- Provider fallback / failover (not needed per user requirements)
- Streaming responses (not needed for this use case)
- Advanced retry logic with exponential backoff (can be added later if needed)

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────┐
│         internal/llm/provider.go            │
│  ┌─────────────────────────────────────────┐│
│  │        interface LLMProvider            ││
│  │  Generate(ctx, prompt, articles)        ││
│  │         ([]TechHighlight, error)        ││
│  └─────────────────────────────────────────┘│
│                    ▲                        │
│          ┌────────┴────────┐                 │
│          │                 │                 │
│  ┌───────┴─────┐   ┌───────┴─────────┐      │
│  │gemini.go    │   │openai.go        │      │
│  │GeminiProvider│   │OpenAIProvider   │      │
│  └─────────────┘   └─────────────────┘      │
└─────────────────────────────────────────────┘
                      ▲
┌─────────────────────┴───────────────────────┐
│    internal/sources/hackernews/hackernews.go │
│  - Accepts LLMProvider in NewSource()        │
│  - Calls provider.Generate()                 │
└─────────────────────────────────────────────┘
```

### Package Structure

```
internal/
├── llm/
│   ├── provider.go      # Interface definition + shared types
│   ├── gemini.go        # Gemini native implementation
│   ├── openai.go        # OpenAI-compatible implementation
│   └── factory.go       # Provider factory function
└── sources/
    └── hackernews/
        ├── hackernews.go  # Updated to use generic provider
        ├── gemini.go      # DEPRECATED - move logic to internal/llm/
        └── types.go       # May need updates
```

## Interface Design

### Core Provider Interface

```go
// internal/llm/provider.go

type LLMProvider interface {
    // Generate analyzes articles using the provider's LLM and returns technical highlights.
    // The prompt includes the system instructions and articles as JSON.
    Generate(ctx context.Context, articles []hackernews.HNArticle) ([]pipeline.TechHighlight, error)
}
```

### Provider Implementations

#### 1. Gemini Provider (Native)

```go
// internal/llm/gemini.go

type GeminiProvider struct {
    apiKey string
    model  string
    client *http.Client
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
    if model == "" {
        model = "gemini-3-flash"
    }
    return &GeminiProvider{
        apiKey: apiKey,
        model:  model,
        client: &http.Client{Timeout: 120 * time.Second},
    }
}

func (p *GeminiProvider) Generate(ctx context.Context, articles []hackernews.HNArticle) ([]pipeline.TechHighlight, error) {
    // Uses Gemini API: POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent
    // Returns TechHighlight slice
}
```

#### 2. OpenAI-Compatible Provider

```go
// internal/llm/openai.go

type OpenAIProvider struct {
    baseURL string
    apiKey  string
    model   string
    client  *http.Client
}

func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIProvider {
    if baseURL == "" {
        baseURL = "https://api.openai.com/v1"
    }
    if model == "" {
        model = "gpt-4"
    }
    return &OpenAIProvider{
        baseURL: strings.TrimSuffix(baseURL, "/"),
        apiKey:  apiKey,
        model:   model,
        client:  &http.Client{Timeout: 120 * time.Second},
    }
}

func (p *OpenAIProvider) Generate(ctx context.Context, articles []hackernews.HNArticle) ([]pipeline.TechHighlight, error) {
    // Uses OpenAI chat completions: POST {baseURL}/chat/completions
    // Returns TechHighlight slice
}
```

### Factory Function

```go
// internal/llm/factory.go

func NewProvider(cfg *config.Config) (LLMProvider, error) {
    switch strings.ToLower(cfg.LLMProvider) {
    case "gemini", "":
        if cfg.LLMAPIKey == "" {
            return nil, fmt.Errorf("LLM_API_KEY required for Gemini provider")
        }
        return NewGeminiProvider(cfg.LLMAPIKey, cfg.LLMModel), nil
    case "openai":
        if cfg.LLMAPIKey == "" {
            return nil, fmt.Errorf("LLM_API_KEY required for OpenAI provider")
        }
        return NewOpenAIProvider(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel), nil
    default:
        return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLMProvider)
    }
}
```

## Configuration

### Environment Variables

```bash
# Provider selection (gemini | openai)
# Default: gemini
LLM_PROVIDER=gemini

# API key (shared between providers)
# Required if using LLM features
LLM_API_KEY=your-api-key-here

# Model name (provider-specific)
# Gemini: gemini-3-flash, gemini-2.5-pro, etc.
# OpenAI: gpt-4, gpt-3.5-turbo, etc.
LLM_MODEL=gemini-3-flash

# Base URL for OpenAI-compatible providers (optional)
# If not set, defaults to https://api.openai.com/v1
LLM_BASE_URL=https://api.openai.com/v1
# Or for OpenRouter: https://openrouter.ai/api/v1
# Or for Fireworks: https://api.fireworks.ai/inference/v1
```

### Config Structure

```go
// internal/config/config.go

type Config struct {
    // ... existing fields ...
    
    // LLM configuration
    LLMProvider string // "gemini" or "openai" (default: "gemini")
    LLMBaseURL  string // For OpenAI-compatible providers (optional)
    LLMAPIKey   string // API key for the selected provider
    LLMModel    string // Model name (provider-specific)
}
```

## Usage Examples

### Using Gemini (Default)

```bash
LLM_PROVIDER=gemini
LLM_API_KEY=your-gemini-key
LLM_MODEL=gemini-3-flash
```

### Using OpenAI

```bash
LLM_PROVIDER=openai
LLM_API_KEY=your-openai-key
LLM_MODEL=gpt-4
# LLM_BASE_URL defaults to https://api.openai.com/v1
```

### Using OpenRouter

```bash
LLM_PROVIDER=openai
LLM_BASE_URL=https://openrouter.ai/api/v1
LLM_API_KEY=your-openrouter-key
LLM_MODEL=anthropic/claude-3.5-sonnet
```

### Using Fireworks.ai

```bash
LLM_PROVIDER=openai
LLM_BASE_URL=https://api.fireworks.ai/inference/v1
LLM_API_KEY=your-fireworks-key
LLM_MODEL=accounts/fireworks/models/llama-v3p1-70b-instruct
```

### Using Kimi

```bash
LLM_PROVIDER=openai
LLM_BASE_URL=https://api.moonshot.cn/v1
LLM_API_KEY=your-kimi-key
LLM_MODEL=kimi-k2-5-turbo
```

## Migration: Adding New Native Provider

To add a new native provider (e.g., Anthropic Claude):

1. **Create implementation file** (`internal/llm/anthropic.go`):

```go
package llm

type AnthropicProvider struct {
    apiKey string
    model  string
    client *http.Client
}

func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
    if model == "" {
        model = "claude-3-5-sonnet-20241022"
    }
    return &AnthropicProvider{
        apiKey: apiKey,
        model:  model,
        client: &http.Client{Timeout: 120 * time.Second},
    }
}

func (p *AnthropicProvider) Generate(ctx context.Context, articles []hackernews.HNArticle) ([]pipeline.TechHighlight, error) {
    // Implement Anthropic's Messages API
    // Use shared extractJSONFromMarkdown utility
    // Return TechHighlight slice
}
```

2. **Update factory** (`internal/llm/factory.go`):

```go
func NewProvider(cfg *config.Config) (LLMProvider, error) {
    switch strings.ToLower(cfg.LLMProvider) {
    case "gemini", "":
        return NewGeminiProvider(cfg.LLMAPIKey, cfg.LLMModel), nil
    case "openai":
        return NewOpenAIProvider(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel), nil
    case "anthropic":  // <-- Add this
        return NewAnthropicProvider(cfg.LLMAPIKey, cfg.LLMModel), nil
    default:
        return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLMProvider)
    }
}
```

3. **Update config** (optional - for validation):

Add "anthropic" to any provider validation lists.

4. **Test**:

Add unit tests for the new provider.

**Total changes: ~3 files, ~100 lines of code.**

## Shared Utilities

The `internal/llm` package will include shared utilities used by all providers:

```go
// internal/llm/util.go

// PromptTemplate is the system prompt for article analysis
const PromptTemplate = `Act as a technical research assistant for a Principal Software Development Engineer. I am providing a JSON list of technical articles from Hacker News with their full URLs. Navigate to the URLs, read the content, and summarize what matters for engineering readers. Skip lifestyle, political, or generic news; include only articles with real technical substance.

For each included article, write "highlights" as plain text with line breaks:
- First line(s): a tight summary (2-4 sentences max) of what the piece is about and why it matters technically.
- Then 2-5 short bullet lines (each line starting with "• " or "- ") with the most interesting or useful details—e.g. approach, trade-offs, tools, numbers, or a takeaway. No filler; prefer specifics over generic praise.

Keep the whole "highlights" field scannable: avoid long paragraphs and numbered section headers like "Core Concept".

Return the technical articles in this exact JSON format:
{
  "articles": [
    {
      "title": "string",
      "url": "string",
      "highlights": "multi-line string: brief summary, then bullet points"
    }
  ]
}`

// ExtractJSONFromMarkdown extracts JSON from markdown code blocks or returns raw string
func ExtractJSONFromMarkdown(text string) string {
    // Try to extract JSON from ```json ... ``` block
    re := regexp.MustCompile("(?s)```(?:json)?\\s*({.+?})\\s*```")
    matches := re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return matches[1]
    }

    // Try to find JSON object directly
    re = regexp.MustCompile("(?s)({[\\s\\S]+})")
    matches = re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return matches[1]
    }

    return text
}
```

## Error Handling

All providers follow the same error handling strategy:

1. **API errors** (non-200 status, invalid JSON) → Return wrapped error
2. **Empty responses** → Return wrapped error
3. **Invalid response format** → Return wrapped error with response preview

The Hacker News source handles errors by logging a warning and returning `nil` (skipping the section):

```go
highlights, err := s.provider.Generate(ctx, articles)
if err != nil {
    log.Printf("[hackernews] WARNING: Failed to analyze articles: %v", err)
    return nil // Don't fail the whole report
}
```

## Testing Strategy

### Unit Tests per Provider

Each provider implementation gets unit tests with mocked HTTP responses:

```go
// internal/llm/gemini_test.go
func TestGeminiProvider_Generate(t *testing.T) {
    // Mock Gemini API response
    // Test successful response parsing
    // Test error handling
}

// internal/llm/openai_test.go
func TestOpenAIProvider_Generate(t *testing.T) {
    // Mock OpenAI API response
    // Test successful response parsing
    // Test error handling
}
```

### Factory Tests

```go
// internal/llm/factory_test.go
func TestNewProvider(t *testing.T) {
    // Test "gemini" returns GeminiProvider
    // Test "openai" returns OpenAIProvider
    // Test unknown provider returns error
    // Test case insensitivity
}
```

### Integration Tests

Existing hackernews tests continue to work by injecting a mock provider:

```go
// internal/sources/hackernews/hackernews_test.go
type mockLLMProvider struct {
    highlights []pipeline.TechHighlight
    err        error
}

func (m *mockLLMProvider) Generate(ctx context.Context, articles []HNArticle) ([]pipeline.TechHighlight, error) {
    return m.highlights, m.err
}
```

## Backward Compatibility

- Existing `GEMINI_API_KEY` and `GEMINI_MODEL` env vars continue to work
- If `LLM_PROVIDER` is not set, defaults to Gemini provider
- If `LLM_MODEL` is not set, uses provider-specific defaults
- Graceful degradation: if no API key is set, the technology section is skipped (existing behavior)

## Migration Plan

1. Create `internal/llm/` package with interface, utilities, and factory
2. Move Gemini logic from `hackernews/gemini.go` to `llm/gemini.go`
3. Create OpenAI provider implementation
4. Update hackernews source to accept LLMProvider interface
5. Update config to read new env vars (with backward compatibility)
6. Update main.go to use factory
7. Add tests for new providers
8. Deprecate old hackernews/gemini.go (can be removed after migration)

## Files to Create/Modify

### Create
- `internal/llm/provider.go` - Interface and shared types
- `internal/llm/gemini.go` - Gemini implementation
- `internal/llm/openai.go` - OpenAI implementation
- `internal/llm/factory.go` - Provider factory
- `internal/llm/util.go` - Shared utilities (prompt, JSON extraction)
- `internal/llm/*_test.go` - Unit tests

### Modify
- `internal/config/config.go` - Add LLM configuration fields
- `internal/sources/hackernews/hackernews.go` - Accept LLMProvider interface
- `internal/sources/hackernews/types.go` - May need minor updates
- `cmd/reporter/main.go` - Use factory to create provider

### Deprecate (can remove later)
- `internal/sources/hackernews/gemini.go` - Logic moved to `llm/gemini.go`

## Success Criteria

- [x] User can switch providers by changing `LLM_PROVIDER` env var
- [x] Gemini provider works identically to current implementation
- [x] OpenAI provider works with OpenAI API
- [x] OpenAI provider works with OpenRouter
- [x] OpenAI provider works with Fireworks.ai
- [x] OpenAI provider works with Kimi
- [x] Adding a new provider requires changes to only 2-3 files
- [x] All existing tests pass
- [x] New providers have unit tests
- [x] Error handling maintains current behavior (log warning, skip section)

## Open Questions

None - design is finalized.
