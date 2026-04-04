package hackernews

import "github.com/apikdech/gws-weekly-report/internal/pipeline"

// HNArticle represents a single Hacker News story from the Algolia API
type HNArticle struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Points int    `json:"points"`
}

// TechHighlight represents a single analyzed technical article
type TechHighlight = pipeline.TechHighlight

// hnAPIResponse represents the Algolia API response structure
type hnAPIResponse struct {
	Hits []struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		Points    int    `json:"points"`
		CreatedAt int64  `json:"created_at_i"`
	} `json:"hits"`
}

// articleResult represents the expected JSON structure from LLM responses
type articleResult struct {
	Articles []struct {
		Title      string `json:"title"`
		URL        string `json:"url"`
		Highlights string `json:"highlights"`
	} `json:"articles"`
}
