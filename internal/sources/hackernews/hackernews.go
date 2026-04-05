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

	"github.com/apikdech/gws-weekly-report/internal/llm"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers"
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

	// Prepare prompt - no need to ask for JSON format explicitly as ResponseFormat enforces it
	fullPrompt := llm.PromptTemplate + "\n\nArticles JSON:\n" + string(inputJSON)

	// Define the JSON schema for structured output
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"articles": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title": map[string]any{
							"type":        "string",
							"description": "The original article title",
						},
						"url": map[string]any{
							"type":        "string",
							"description": "The original article URL",
						},
						"highlights": map[string]any{
							"type":        "string",
							"description": "2-3 sentence summary with bullet points of technical details",
						},
					},
					"required": []string{"title", "url", "highlights"},
				},
			},
		},
		"required": []string{"articles"},
	}

	strict := true
	log.Printf("[hackernews] Sending request to LLM provider %s with model %s (structured JSON output)", s.provider.Name(), s.model)

	// Use any-llm-go for completion with structured JSON output
	response, err := s.provider.Completion(ctx, anyllm.CompletionParams{
		Model: s.model,
		Messages: []anyllm.Message{
			{Role: anyllm.RoleUser, Content: fullPrompt},
		},
		// Use structured JSON output - this enforces the LLM to return valid JSON matching our schema
		ResponseFormat: &providers.ResponseFormat{
			Type: "json_schema",
			JSONSchema: &providers.JSONSchema{
				Name:        "article_analysis",
				Description: "Analysis of technical Hacker News articles",
				Schema:      schema,
				Strict:      &strict,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// With structured output, the response is guaranteed to be valid JSON matching our schema
	responseText := response.Choices[0].Message.ContentString()

	// Parse the result directly - no need for manual JSON extraction from markdown
	var result articleResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
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
