# HackerNews Technology Section Design

**Date:** 2026-04-04  
**Status:** Approved  
**Approach:** A - New Data Source

## Overview

Add an automated Technology section to the weekly report that:
1. Fetches top 15 technical articles from Hacker News API for the report week
2. Analyzes them using Gemini LLM to identify the top 3 most technically valuable articles
3. Renders formatted highlights in the "Technology, Business, Communication, Leadership, Management & Marketing" section

## Architecture

### New Components

1. **`internal/sources/hackernews/`** - New source package
   - `hackernews.go` - Fetches articles from HN Algolia API
   - `gemini.go` - Analyzes articles with Gemini LLM
   - `types.go` - Data structures for HN articles and tech highlights

2. **Updated Components**
   - `internal/pipeline/types.go` - Add `TechnologyHighlights` field to `ReportData`
   - `internal/config/config.go` - Add `GeminiAPIKey` and `GeminiModel` config
   - `internal/report/report.tmpl` - Update template to render technology section
   - `internal/report/render.go` - Pass technology highlights to template
   - `cmd/reporter/main.go` - Wire up new data source

## Data Flow

```
1. main.go creates HackerNewsSource with config.GeminiAPIKey, config.GeminiModel
2. Runner calls source.Fetch(week):
   a. Calculate Unix timestamps from week.Start and week.End
   b. Call HN API: https://hn.algolia.com/api/v1/search?tags=story&...
   c. Parse top 15 articles (title, url, points)
   d. Build JSON list and send to Gemini with analysis prompt
   e. Parse Gemini response for top 3 articles with highlights
3. Runner calls source.Contribute(report):
   a. Set report.TechnologyHighlights
4. Render passes TechnologyHighlights to template
5. Template renders formatted list
```

## API Details

### HN Algolia API

```
GET https://hn.algolia.com/api/v1/search?tags=story&numericFilters=created_at_i>{start},created_at_i<{end}

Parameters:
- start: Unix timestamp of week.Start
- end: Unix timestamp of week.End

Response structure (hits array, we need first 15):
- hits[].title: Article title
- hits[].url: Article URL
- hits[].points: Vote count
```

### Gemini API

```
Model: Configurable via GEMINI_MODEL env (default: gemini-3-flash)
Endpoint: https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent

Prompt:
"Act as a technical research assistant for a Principal Software Development Engineer. 
I am providing a JSON list of technical articles from Hacker News with their full URLs. 
Please navigate to the URLs, read the content, and extract the hardcore engineering value. 
Skip the lifestyle, political, or generic news articles and focus strictly on the technical ones.

For each technical article, provide:
1. Core Concept: What is the primary problem being solved or concept being introduced?
2. Architecture & Trade-offs: Are there any specific system design patterns, architectural decisions, scaling implications, or performance trade-offs discussed?
3. Tech Stack Relevance: Highlight any mentions of specific languages, infrastructure, or tools (paying special attention to things like Java, Go, Docker, or Cloud platforms).
4. Strategic Takeaway: One actionable insight or strategic consideration a Principal Engineer should take away from this.

Return ONLY the top 3 technical articles in this exact JSON format:
{
  \"articles\": [
    {
      \"title\": \"string\",
      \"url\": \"string\",
      \"highlights\": \"string with line breaks between the 4 analysis points\"
    }
  ]
}"
```

## Error Handling

If any step fails (HN API, Gemini API, JSON parsing):
1. Log warning with detailed error message
2. Set `TechnologyHighlights` to nil (section will be empty in report)
3. Continue report generation
4. The template will simply not render any technology content if data is unavailable

## Data Structures

```go
// internal/sources/hackernews/types.go

type HNArticle struct {
    Title  string `json:"title"`
    URL    string `json:"url"`
    Points int    `json:"points"`
}

type TechHighlight struct {
    Title     string `json:"title"`
    URL       string `json:"url"`
    Highlights string `json:"highlights"`
}

// internal/pipeline/types.go additions

type ReportData struct {
    // ... existing fields ...
    TechnologyHighlights []TechHighlight
}
```

## Configuration

New environment variables (optional):
- `GEMINI_API_KEY` - Required for technology section
- `GEMINI_MODEL` - Optional, defaults to "gemini-3-flash"

If `GEMINI_API_KEY` is not set, the technology section will be skipped with a warning.

## Template Changes

In `internal/report/report.tmpl`, replace line 31:

```
## **Technology, Business, Communication, Leadership, Management & Marketing**
```

With:

```
## **Technology, Business, Communication, Leadership, Management & Marketing**
{{ if .TechnologyHighlights -}}
### Technology Highlights (Top HN Articles)
{{ range .TechnologyHighlights -}}
1. [{{ .Title }}]({{ .URL }})
{{ .Highlights }}
{{ end }}
{{ end }}

## Testing Strategy

1. Unit tests for HN API client (mock HTTP server)
2. Unit tests for Gemini client (mock HTTP server)
3. Integration test for full source.Fetch() flow
4. Template rendering test with TechnologyHighlights data

## Security Considerations

- Gemini API key stored in env var, never logged
- Article URLs passed to Gemini are from HN API (trusted source)
- No caching of article content or LLM responses

## Future Enhancements (Out of Scope)

- Discord webhook notification on errors
- Cache LLM responses to reduce costs
- Support for multiple LLM providers
- Configurable number of articles to fetch/analyze
