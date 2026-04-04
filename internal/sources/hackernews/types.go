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

// geminiRequest represents the request body for Gemini API
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse represents the response from Gemini API
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// geminiArticleResult represents the expected JSON structure from Gemini
type geminiArticleResult struct {
	Articles []struct {
		Title      string `json:"title"`
		URL        string `json:"url"`
		Highlights string `json:"highlights"`
	} `json:"articles"`
}
