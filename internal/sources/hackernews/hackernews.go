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
	responseText, ok := response.Choices[0].Message.Content.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected response content type")
	}
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
