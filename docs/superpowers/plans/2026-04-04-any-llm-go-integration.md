# any-llm-go Library Integration - Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate mozilla-ai/any-llm-go library to support multiple LLM providers with minimal code changes.

**Architecture:** Add any-llm-go as dependency, create simple factory to instantiate providers, update hackernews source to use library's Provider interface. Library handles all provider abstractions, rate limiting, and error handling.

**Tech Stack:** Go, github.com/mozilla-ai/any-llm-go library

---

## File Structure

```
internal/
├── llm/
│   ├── factory.go       # Provider factory using any-llm-go
│   ├── util.go          # Shared utilities (prompt, JSON extraction)
│   └── factory_test.go  # Factory tests
└── sources/
    └── hackernews/
        ├── hackernews.go  # Updated to use anyllm.Provider
        ├── gemini.go      # Will be removed
        └── types.go       # Keep HNArticle type
```

---

## Task 1: Add any-llm-go Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add the library to go.mod**

Run:
```bash
cd /home/ricky-setiawan/Other/ok/gws-weekly-report
go get github.com/mozilla-ai/any-llm-go
```

Expected: Library added to go.mod and go.sum

- [ ] **Step 2: Verify the module is available**

Run: `go list -m github.com/mozilla-ai/any-llm-go`
Expected: Shows module version

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add any-llm-go library for multi-provider support"
```

---

## Task 2: Create Shared Utilities

**Files:**
- Create: `internal/llm/util.go`

- [ ] **Step 1: Create util.go with prompt template and JSON extraction**

```go
package llm

import "regexp"

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

- [ ] **Step 2: Verify the file compiles**

Run: `go build ./internal/llm/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/llm/util.go
git commit -m "feat(llm): add shared prompt template and JSON extraction utility"
```

---

## Task 3: Create Provider Factory

**Files:**
- Create: `internal/llm/factory.go`

- [ ] **Step 1: Create factory function using any-llm-go**

```go
package llm

import (
	"fmt"
	"strings"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
	"github.com/apikdech/gws-weekly-report/internal/config"
)

// NewProvider creates an any-llm-go Provider based on configuration
func NewProvider(cfg *config.Config) (anyllm.Provider, error) {
	provider := strings.ToLower(cfg.LLMProvider)

	switch provider {
	case "gemini", "":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY required for Gemini provider")
		}
		return gemini.New(anyllm.WithAPIKey(cfg.LLMAPIKey)), nil

	case "openai":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY required for OpenAI provider")
		}
		return openai.New(anyllm.WithAPIKey(cfg.LLMAPIKey)), nil

	case "anthropic":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("LLM_API_KEY required for Anthropic provider")
		}
		return anthropic.New(anyllm.WithAPIKey(cfg.LLMAPIKey)), nil

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s (supported: gemini, openai, anthropic)", cfg.LLMProvider)
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/llm/...`
Expected: No errors (this will download the provider packages)

- [ ] **Step 3: Commit**

```bash
git add internal/llm/factory.go
git commit -m "feat(llm): add provider factory using any-llm-go library"
```

---

## Task 4: Update Configuration

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Read current config.go**

First, let's see the current config structure:
```bash
cat internal/config/config.go
```

- [ ] **Step 2: Add new LLM configuration fields**

Replace the Gemini-specific fields with unified LLM fields:

```go
// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// ... existing fields remain unchanged ...
	GitHubToken        string
	GitHubUsername     string
	GWSEmailSender     string
	ReportName         string
	GWSCredentialsFile string
	GWSChatSpacesID    string
	GWSChatSenderName  string
	ReportTimezone     string
	TempDir            string
	NextActions        []string

	// LLM configuration (new unified fields)
	LLMProvider string // "gemini" | "openai" | "anthropic" (default: "gemini")
	LLMAPIKey   string // API key for the selected provider
	LLMModel    string // Model name (provider-specific)

	// Legacy fields for backward compatibility (deprecated, but still supported)
	GeminiAPIKey string // Deprecated: use LLMAPIKey
	GeminiModel  string // Deprecated: use LLMModel
}
```

- [ ] **Step 3: Update Load() function**

Add the new env var reading with backward compatibility:

```go
func Load() (*Config, error) {
	cfg := &Config{
		// ... existing fields ...
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		GitHubUsername:     os.Getenv("GITHUB_USERNAME"),
		GWSEmailSender:     os.Getenv("GWS_EMAIL_SENDER"),
		ReportName:         os.Getenv("REPORT_NAME"),
		GWSCredentialsFile: os.Getenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE"),
		GWSChatSpacesID:    os.Getenv("GWS_CHAT_SPACES_ID"),
		GWSChatSenderName:  os.Getenv("GWS_CHAT_SENDER_NAME"),
		ReportTimezone:     os.Getenv("REPORT_TIMEZONE"),
		TempDir:            os.Getenv("TEMP_DIR"),
	}

	// Apply defaults
	if cfg.ReportTimezone == "" {
		cfg.ReportTimezone = "UTC"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "/tmp"
	}
	cfg.NextActions = parseCommaSeparated(os.Getenv("REPORT_NEXT_ACTIONS"))

	// Read new LLM configuration
	cfg.LLMProvider = os.Getenv("LLM_PROVIDER")
	cfg.LLMAPIKey = os.Getenv("LLM_API_KEY")
	cfg.LLMModel = os.Getenv("LLM_MODEL")

	// Read legacy config for backward compatibility
	cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	cfg.GeminiModel = os.Getenv("GEMINI_MODEL")

	// Apply backward compatibility: if LLM_* not set but GEMINI_* is, use GEMINI_*
	if cfg.LLMProvider == "" && cfg.GeminiAPIKey != "" {
		cfg.LLMProvider = "gemini"
	}
	if cfg.LLMAPIKey == "" && cfg.GeminiAPIKey != "" {
		cfg.LLMAPIKey = cfg.GeminiAPIKey
	}
	if cfg.LLMModel == "" && cfg.GeminiModel != "" {
		cfg.LLMModel = cfg.GeminiModel
	}

	// Validate required fields (existing validation)
	type requiredVar struct {
		name string
		val  string
	}
	required := []requiredVar{
		{"GITHUB_TOKEN", cfg.GitHubToken},
		{"GITHUB_USERNAME", cfg.GitHubUsername},
		{"GWS_EMAIL_SENDER", cfg.GWSEmailSender},
		{"REPORT_NAME", cfg.ReportName},
		{"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", cfg.GWSCredentialsFile},
		{"GWS_CHAT_SPACES_ID", cfg.GWSChatSpacesID},
		{"GWS_CHAT_SENDER_NAME", cfg.GWSChatSenderName},
	}
	var missing []string
	for _, r := range required {
		if r.val == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/config/...`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add unified LLM configuration with backward compatibility"
```

---

## Task 5: Update HackerNews Source

**Files:**
- Modify: `internal/sources/hackernews/hackernews.go`
- Modify: `internal/sources/hackernews/types.go` (add articleResult type)

- [ ] **Step 1: Update types.go to add articleResult**

Add to `internal/sources/hackernews/types.go`:

```go
// articleResult represents the expected JSON structure from LLM responses
type articleResult struct {
	Articles []struct {
		Title      string `json:"title"`
		URL        string `json:"url"`
		Highlights string `json:"highlights"`
	} `json:"articles"`
}
```

- [ ] **Step 2: Update hackernews.go**

Replace the entire content:

```go
package hackernews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/apikdech/gws-weekly-report/internal/llm"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

// Source fetches top technical articles from Hacker News for the week.
type Source struct {
	provider   anyllm.Provider
	model      string
	highlights []pipeline.TechHighlight
}

// NewSource creates a HackerNewsSource.
// provider can be nil (section will be skipped).
func NewSource(provider anyllm.Provider, model string) *Source {
	return &Source{provider: provider, model: model}
}

// Name implements DataSource.
func (s *Source) Name() string { return "hackernews" }

// Fetch retrieves top 15 HN articles and analyzes them with LLM.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	if s.provider == nil {
		log.Printf("[hackernews] LLM provider not configured, skipping technology section")
		return nil
	}

	// Fetch articles from HN API
	articles, err := s.fetchHNArticles(ctx, week)
	if err != nil {
		log.Printf("[hackernews] WARNING: Failed to fetch HN articles: %v", err)
		return nil // Don't fail the whole report
	}

	if len(articles) == 0 {
		log.Printf("[hackernews] No articles found for week %s", week.HeaderLabel())
		return nil
	}

	// Analyze with LLM to get technical articles
	highlights, err := s.analyzeWithLLM(ctx, articles)
	if err != nil {
		// Check for specific error types from any-llm-go
		if errors.Is(err, anyllm.ErrRateLimit) {
			log.Printf("[hackernews] WARNING: Rate limited by LLM provider: %v", err)
		} else if errors.Is(err, anyllm.ErrAuthentication) {
			log.Printf("[hackernews] WARNING: LLM authentication failed: %v", err)
		} else {
			log.Printf("[hackernews] WARNING: Failed to analyze with LLM: %v", err)
		}
		return nil // Don't fail the whole report
	}

	s.highlights = highlights
	log.Printf("[hackernews] Successfully analyzed %d articles", len(highlights))
	return nil
}

func (s *Source) fetchHNArticles(ctx context.Context, week pipeline.WeekRange) ([]HNArticle, error) {
	startUnix := week.Start.Unix()
	endUnix := week.End.Unix()

	u, err := url.Parse("https://hn.algolia.com/api/v1/search")
	if err != nil {
		return nil, fmt.Errorf("parse hn api base url: %w", err)
	}
	q := u.Query()
	q.Set("tags", "story")
	q.Set("numericFilters", fmt.Sprintf("created_at_i>%d,created_at_i<%d", startUnix, endUnix))
	q.Set("hitsPerPage", "15")
	u.RawQuery = q.Encode()
	apiURL := u.String()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d", resp.StatusCode)
	}

	var apiResp hnAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	articles := make([]HNArticle, 0, len(apiResp.Hits))
	for _, hit := range apiResp.Hits {
		articles = append(articles, HNArticle{
			Title:  hit.Title,
			URL:    hit.URL,
			Points: hit.Points,
		})
	}

	return articles, nil
}

func (s *Source) analyzeWithLLM(ctx context.Context, articles []HNArticle) ([]pipeline.TechHighlight, error) {
	// Build JSON input
	inputJSON, err := json.Marshal(articles)
	if err != nil {
		return nil, fmt.Errorf("marshal articles: %w", err)
	}

	// Prepare prompt
	fullPrompt := llm.PromptTemplate + "\n\nArticles JSON:\n" + string(inputJSON)

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

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract JSON from response text
	responseText := response.Choices[0].Message.Content
	jsonStr := llm.ExtractJSONFromMarkdown(responseText)

	// Parse the result
	var result articleResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse LLM result: %w", err)
	}

	// Convert to TechHighlight slice
	highlights := make([]pipeline.TechHighlight, 0, len(result.Articles))
	for _, article := range result.Articles {
		highlights = append(highlights, pipeline.TechHighlight{
			Title:      article.Title,
			URL:        article.URL,
			Highlights: article.Highlights,
		})
	}

	return highlights, nil
}

// Contribute sets TechnologyHighlights on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	report.TechnologyHighlights = s.highlights
	return nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/sources/hackernews/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/sources/hackernews/hackernews.go internal/sources/hackernews/types.go
git commit -m "refactor(hackernews): use any-llm-go library for LLM providers"
```

---

## Task 6: Remove Old Gemini Implementation

**Files:**
- Delete: `internal/sources/hackernews/gemini.go`

- [ ] **Step 1: Remove the old file**

```bash
rm internal/sources/hackernews/gemini.go
```

- [ ] **Step 2: Verify compilation still works**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git rm internal/sources/hackernews/gemini.go
git commit -m "refactor(hackernews): remove old Gemini implementation (now handled by any-llm-go)"
```

---

## Task 7: Update Main Entry Point

**Files:**
- Modify: `cmd/reporter/main.go`

- [ ] **Step 1: Read current main.go**

```bash
cat cmd/reporter/main.go
```

- [ ] **Step 2: Find where hackernews.NewSource is called and update it**

Find the existing hackernews source initialization and replace it:

```go
// OLD CODE (to be replaced):
// hnSource := hackernews.NewSource(cfg.GeminiAPIKey, cfg.GeminiModel)

// NEW CODE:
// Create LLM provider if configured
var llmProvider anyllm.Provider
if cfg.LLMAPIKey != "" {
	var err error
	llmProvider, err = llm.NewProvider(cfg)
	if err != nil {
		log.Printf("[main] WARNING: Failed to create LLM provider: %v", err)
		// Continue without LLM provider - section will be skipped
	}
}

hnSource := hackernews.NewSource(llmProvider, cfg.LLMModel)
```

Make sure to add imports:

```go
import (
	// ... existing imports ...
	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/apikdech/gws-weekly-report/internal/llm"
	"github.com/apikdech/gws-weekly-report/internal/sources/hackernews"
)
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./cmd/reporter/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add cmd/reporter/main.go
git commit -m "feat(main): integrate any-llm-go provider factory"
```

---

## Task 8: Add Unit Tests for Factory

**Files:**
- Create: `internal/llm/factory_test.go`

- [ ] **Step 1: Create factory tests**

```go
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

func TestNewProvider_Unknown(t *testing.T) {
	cfg := &config.Config{
		LLMProvider: "unknown-provider",
		LLMAPIKey:   "test-key",
		LLMModel:    "model",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("Expected error for unknown provider")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/llm/... -v`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/llm/factory_test.go
git commit -m "test(llm): add unit tests for provider factory"
```

---

## Task 9: Update Config Tests

**Files:**
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Add tests for LLM configuration**

Add these test functions to the existing test file:

```go
func TestLoad_LLMConfiguration(t *testing.T) {
	// Save and restore original env vars
	origVars := map[string]string{
		"GITHUB_TOKEN":                            os.Getenv("GITHUB_TOKEN"),
		"GITHUB_USERNAME":                         os.Getenv("GITHUB_USERNAME"),
		"GWS_EMAIL_SENDER":                        os.Getenv("GWS_EMAIL_SENDER"),
		"REPORT_NAME":                             os.Getenv("REPORT_NAME"),
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE":   os.Getenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE"),
		"GWS_CHAT_SPACES_ID":                      os.Getenv("GWS_CHAT_SPACES_ID"),
		"GWS_CHAT_SENDER_NAME":                    os.Getenv("GWS_CHAT_SENDER_NAME"),
		"LLM_PROVIDER":                            os.Getenv("LLM_PROVIDER"),
		"LLM_API_KEY":                             os.Getenv("LLM_API_KEY"),
		"LLM_MODEL":                               os.Getenv("LLM_MODEL"),
	}
	defer func() {
		for k, v := range origVars {
			os.Setenv(k, v)
		}
	}()

	// Set required env vars
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_USERNAME", "test-user")
	os.Setenv("GWS_EMAIL_SENDER", "test@example.com")
	os.Setenv("REPORT_NAME", "Test Report")
	os.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	os.Setenv("GWS_CHAT_SPACES_ID", "SPACE123")
	os.Setenv("GWS_CHAT_SENDER_NAME", "users/123")
	
	// Set LLM configuration
	os.Setenv("LLM_PROVIDER", "openai")
	os.Setenv("LLM_API_KEY", "test-llm-key")
	os.Setenv("LLM_MODEL", "gpt-4")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %v, want openai", cfg.LLMProvider)
	}
	if cfg.LLMAPIKey != "test-llm-key" {
		t.Errorf("LLMAPIKey = %v, want test-llm-key", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "gpt-4" {
		t.Errorf("LLMModel = %v, want gpt-4", cfg.LLMModel)
	}
}

func TestLoad_LLMBackwardCompatibility(t *testing.T) {
	// Save and restore original env vars
	origVars := map[string]string{
		"GITHUB_TOKEN":                          os.Getenv("GITHUB_TOKEN"),
		"GITHUB_USERNAME":                     os.Getenv("GITHUB_USERNAME"),
		"GWS_EMAIL_SENDER":                      os.Getenv("GWS_EMAIL_SENDER"),
		"REPORT_NAME":                           os.Getenv("REPORT_NAME"),
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE": os.Getenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE"),
		"GWS_CHAT_SPACES_ID":                    os.Getenv("GWS_CHAT_SPACES_ID"),
		"GWS_CHAT_SENDER_NAME":                  os.Getenv("GWS_CHAT_SENDER_NAME"),
		"LLM_PROVIDER":                          os.Getenv("LLM_PROVIDER"),
		"LLM_API_KEY":                           os.Getenv("LLM_API_KEY"),
		"LLM_MODEL":                             os.Getenv("LLM_MODEL"),
		"GEMINI_API_KEY":                        os.Getenv("GEMINI_API_KEY"),
		"GEMINI_MODEL":                          os.Getenv("GEMINI_MODEL"),
	}
	defer func() {
		for k, v := range origVars {
			os.Setenv(k, v)
		}
	}()

	// Set required env vars
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_USERNAME", "test-user")
	os.Setenv("GWS_EMAIL_SENDER", "test@example.com")
	os.Setenv("REPORT_NAME", "Test Report")
	os.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	os.Setenv("GWS_CHAT_SPACES_ID", "SPACE123")
	os.Setenv("GWS_CHAT_SENDER_NAME", "users/123")
	
	// Clear new LLM vars
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")
	os.Unsetenv("LLM_MODEL")
	
	// Set legacy Gemini vars
	os.Setenv("GEMINI_API_KEY", "legacy-gemini-key")
	os.Setenv("GEMINI_MODEL", "gemini-pro")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should use legacy values
	if cfg.LLMProvider != "gemini" {
		t.Errorf("LLMProvider should default to gemini when GEMINI_API_KEY is set, got %v", cfg.LLMProvider)
	}
	if cfg.LLMAPIKey != "legacy-gemini-key" {
		t.Errorf("LLMAPIKey should be set from GEMINI_API_KEY, got %v", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "gemini-pro" {
		t.Errorf("LLMModel should be set from GEMINI_MODEL, got %v", cfg.LLMModel)
	}
}
```

- [ ] **Step 2: Run config tests**

Run: `go test ./internal/config/... -v`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test(config): add tests for LLM configuration and backward compatibility"
```

---

## Task 10: Run Full Test Suite

**Files:**
- All packages

- [ ] **Step 1: Run all tests**

Run: `go test ./...`
Expected: All tests pass

- [ ] **Step 2: Build the entire project**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 3: Verify the binary builds**

Run: `go build -o reporter ./cmd/reporter`
Expected: Binary created successfully

- [ ] **Step 4: Run the reporter binary to verify it starts**

Run: `./reporter --help` (or just `./reporter` to see usage)
Expected: Shows usage or help message without crashing

- [ ] **Step 5: Commit any final fixes**

If any fixes were needed:

```bash
git add .
git commit -m "test: fix any test issues from any-llm-go integration"
```

---

## Task 11: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `.env.example`

- [ ] **Step 1: Update README.md**

Find the Gemini configuration section and replace with:

```markdown
## LLM Configuration (for Technology Highlights)

The application supports multiple LLM providers via the [any-llm-go](https://github.com/mozilla-ai/any-llm-go) library. Configure via environment variables:

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

### Provider Examples

**Google Gemini (default):**
```bash
LLM_PROVIDER=gemini
LLM_API_KEY=your-gemini-key
LLM_MODEL=gemini-3-flash
```

**OpenAI:**
```bash
LLM_PROVIDER=openai
LLM_API_KEY=your-openai-key
LLM_MODEL=gpt-4
```

**Anthropic Claude:**
```bash
LLM_PROVIDER=anthropic
LLM_API_KEY=your-anthropic-key
LLM_MODEL=claude-3-5-sonnet-20241022
```

**OpenRouter (via OpenAI-compatible API):**
```bash
LLM_PROVIDER=openai
OPENAI_BASE_URL=https://openrouter.ai/api/v1  # Set this env var for the library
LLM_API_KEY=your-openrouter-key
LLM_MODEL=anthropic/claude-3.5-sonnet
```

### Backward Compatibility

Legacy `GEMINI_API_KEY` and `GEMINI_MODEL` environment variables are still supported and will be used if `LLM_*` variables are not set.
```

- [ ] **Step 2: Update .env.example**

Replace the Gemini section with:

```bash
# LLM Provider Configuration (for Technology Highlights)
# Provider: gemini | openai | anthropic (default: gemini)
LLM_PROVIDER=gemini

# API key for the selected provider
LLM_API_KEY=your-api-key-here

# Model name (provider-specific)
# Gemini: gemini-3-flash, gemini-2.5-pro, gemini-2.0-flash
# OpenAI: gpt-4, gpt-3.5-turbo, gpt-4o
# Anthropic: claude-3-5-sonnet-20241022, claude-3-opus
LLM_MODEL=gemini-3-flash

# Legacy variables (backward compatible, will be used if LLM_* not set)
# GEMINI_API_KEY=your-gemini-key
# GEMINI_MODEL=gemini-3-flash
```

- [ ] **Step 3: Commit documentation**

```bash
git add README.md .env.example
git commit -m "docs: update README and .env.example with any-llm-go configuration"
```

---

## Summary

After completing all tasks, you will have:

1. ✅ Integrated mozilla-ai/any-llm-go library as dependency
2. ✅ Created simple factory for provider instantiation (3 providers supported)
3. ✅ Updated hackernews source to use library's Provider interface
4. ✅ Removed old custom Gemini implementation (~100 lines → 0 lines)
5. ✅ Updated configuration with backward compatibility
6. ✅ Added unit tests for factory
7. ✅ Updated documentation

**Files Created:**
- `internal/llm/util.go`
- `internal/llm/factory.go`
- `internal/llm/factory_test.go`

**Files Modified:**
- `go.mod` (added dependency)
- `go.sum` (added dependency)
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/sources/hackernews/hackernews.go`
- `internal/sources/hackernews/types.go`
- `cmd/reporter/main.go`
- `README.md`
- `.env.example`

**Files Deleted:**
- `internal/sources/hackernews/gemini.go`

**Code Reduction:** ~100 lines removed, ~50 lines added (net reduction)

**Provider Support:** Gemini ✅, OpenAI ✅, Anthropic ✅, Easy to add more
