package hackernews

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const geminiPrompt = `Act as a technical research assistant for a Principal Software Development Engineer. I am providing a JSON list of technical articles from Hacker News with their full URLs. Navigate to the URLs, read the content, and summarize what matters for engineering readers. Skip lifestyle, political, or generic news; include only articles with real technical substance.

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

func (s *Source) analyzeWithGemini(ctx context.Context, articles []HNArticle) ([]TechHighlight, error) {
	// Build JSON input for Gemini
	inputJSON, err := json.Marshal(articles)
	if err != nil {
		return nil, fmt.Errorf("marshal articles: %w", err)
	}

	// Prepare request body
	fullPrompt := geminiPrompt + "\n\nArticles JSON:\n" + string(inputJSON)
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: fullPrompt}}},
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Build API URL
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		s.model, s.apiKey,
	)

	log.Printf("[hackernews] Sending request to Gemini API with model %s", s.model)
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		body := strings.TrimSpace(string(b))
		if body != "" {
			return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, body)
		}
		return nil, fmt.Errorf("Gemini API returned status %d", resp.StatusCode)
	}

	// Parse response
	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Extract JSON from response text (it may be wrapped in markdown code blocks)
	responseText := geminiResp.Candidates[0].Content.Parts[0].Text
	jsonStr := extractJSONFromMarkdown(responseText)

	// Parse the result
	var result geminiArticleResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse Gemini result: %w", err)
	}

	log.Printf("[hackernews] Found %d articles", len(result.Articles))

	// Convert to TechHighlight slice
	highlights := make([]TechHighlight, 0, len(result.Articles))
	for _, article := range result.Articles {
		highlights = append(highlights, TechHighlight{
			Title:      article.Title,
			URL:        article.URL,
			Highlights: article.Highlights,
		})
	}

	return highlights, nil
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks or returns raw string
func extractJSONFromMarkdown(text string) string {
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
