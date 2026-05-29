package hackernews

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maxContentLength = 4000

var (
	scriptStyleRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
	tagRe         = regexp.MustCompile(`<[^>]*>`)
	entityRe      = regexp.MustCompile(`&[a-zA-Z]{2,6};|&#\d{2,4};`)
	wsRe          = regexp.MustCompile(`\s+`)
)

func fetchContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	text := stripHTML(body)
	text = strings.TrimSpace(text)

	if len(text) > maxContentLength {
		text = text[:maxContentLength]
	}

	return text, nil
}

func stripHTML(body []byte) string {
	s := scriptStyleRe.ReplaceAllLiteral(body, nil)
	s = tagRe.ReplaceAll(s, []byte(" "))
	s = entityRe.ReplaceAll(s, []byte(" "))
	s = bytes.ReplaceAll(s, []byte("\n"), []byte(" "))
	s = bytes.ReplaceAll(s, []byte("\r"), []byte(" "))
	s = bytes.ReplaceAll(s, []byte("\t"), []byte(" "))
	s = wsRe.ReplaceAll(s, []byte(" "))
	return string(s)
}
