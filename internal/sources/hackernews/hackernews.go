package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

// Source fetches top technical articles from Hacker News for the week.
type Source struct {
	apiKey     string
	model      string
	highlights []pipeline.TechHighlight
}

// NewSource creates a HackerNewsSource. apiKey can be empty (section will be skipped).
func NewSource(apiKey, model string) *Source {
	if model == "" {
		model = "gemini-3-flash"
	}
	return &Source{apiKey: apiKey, model: model}
}

// Name implements DataSource.
func (s *Source) Name() string { return "hackernews" }

// Fetch retrieves top 15 HN articles and analyzes them with Gemini.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	if s.apiKey == "" {
		log.Printf("[hackernews] GEMINI_API_KEY not set, skipping technology section")
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

	// Analyze with Gemini to get top 3 technical articles
	highlights, err := s.analyzeWithGemini(ctx, articles)
	if err != nil {
		log.Printf("[hackernews] WARNING: Failed to analyze with Gemini: %v", err)
		return nil // Don't fail the whole report
	}

	s.highlights = highlights
	log.Printf("[hackernews] Successfully analyzed %d articles", len(highlights))
	return nil
}

func (s *Source) fetchHNArticles(ctx context.Context, week pipeline.WeekRange) ([]HNArticle, error) {
	startUnix := week.Start.Unix()
	endUnix := week.End.Unix()

	url := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?tags=story&numericFilters=created_at_i>%d,created_at_i<%d&hitsPerPage=15",
		startUnix, endUnix,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

// Contribute sets TechnologyHighlights on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	report.TechnologyHighlights = s.highlights
	return nil
}
