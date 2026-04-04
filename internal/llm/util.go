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
