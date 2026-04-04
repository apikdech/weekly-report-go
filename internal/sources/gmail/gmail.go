package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

var docIDRegexp = regexp.MustCompile(`docs\.google\.com/document/d/([a-zA-Z0-9_-]+)`)

func qpHexDigit(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'A' && b <= 'F':
		return int(b - 'A' + 10)
	case b >= 'a' && b <= 'f':
		return int(b - 'a' + 10)
	default:
		return -1
	}
}

// normalizeQuotedPrintableForScan applies RFC 2045 rules enough for URL extraction:
// soft line breaks ("=" then CRLF or LF) are removed, then "=HH" hex bytes are decoded.
func normalizeQuotedPrintableForScan(b []byte) []byte {
	s := bytes.ReplaceAll(b, []byte("=\r\n"), nil)
	s = bytes.ReplaceAll(s, []byte("=\n"), nil)
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '=' && i+2 < len(s) {
			hi, lo := qpHexDigit(s[i+1]), qpHexDigit(s[i+2])
			if hi >= 0 && lo >= 0 {
				out = append(out, byte(hi<<4|lo))
				i += 2
				continue
			}
		}
		out = append(out, s[i])
	}
	return out
}

type messagesResponse struct {
	Messages []struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"messages"`
	ResultSizeEstimate int `json:"resultSizeEstimate"`
}

// ExtractMessageID parses the gws messages list JSON and returns the first message ID.
func ExtractMessageID(data []byte) (string, error) {
	var resp messagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse messages response: %w", err)
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no messages found in response")
	}
	return resp.Messages[0].ID, nil
}

// ExtractDocID scans email body bytes for the first Google Docs URL and returns the file ID.
// Bodies often use quoted-printable (soft breaks as "=\n", "=" as "=3D"); those are normalized first.
func ExtractDocID(body []byte) (string, error) {
	normalized := normalizeQuotedPrintableForScan(body)
	matches := docIDRegexp.FindSubmatch(normalized)
	if len(matches) < 2 {
		return "", fmt.Errorf("no Google Docs URL found in email body")
	}
	return string(matches[1]), nil
}

// decodeGmailRawMessage decodes the Gmail API "raw" field (RFC 822), trying common base64 variants.
func decodeGmailRawMessage(b64 string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, enc := range encodings {
		b, err := enc.DecodeString(b64)
		if err == nil {
			return b, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("decode gmail raw: %w", lastErr)
}

func (s *Source) fetchMessageRawRFC822(ctx context.Context, msgID string) ([]byte, error) {
	params := fmt.Sprintf(`{"userId":"me","id":%q,"format":"raw"}`, msgID)
	out, err := s.executor.Run(ctx, "gmail", "users", "messages", "get", "--params", params)
	if err != nil {
		return nil, fmt.Errorf("gmail messages get raw: %w", err)
	}
	var resp struct {
		Raw string `json:"raw"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse messages get: %w", err)
	}
	if resp.Raw == "" {
		return nil, fmt.Errorf("messages get: empty raw field")
	}
	return decodeGmailRawMessage(resp.Raw)
}

// Source fetches the Google Doc ID from the weekly report email.
type Source struct {
	executor    *gws.Executor
	emailSender string
	reportName  string
	docID       string
}

// NewSource creates a GmailSource.
func NewSource(executor *gws.Executor, emailSender, reportName string) *Source {
	return &Source{
		executor:    executor,
		emailSender: emailSender,
		reportName:  reportName,
	}
}

// Name implements DataSource.
func (s *Source) Name() string { return "gmail" }

// Fetch searches Gmail for the weekly report email and extracts the Google Doc ID.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	query := fmt.Sprintf(`from:(%s) [Fill Weekly Report: %s] %s`,
		s.emailSender, s.reportName, week.EmailDateLabel())
	params := fmt.Sprintf(`{"userId":"me","q":%q}`, query)

	listOut, err := s.executor.Run(ctx, "gmail", "users", "messages", "list", "--params", params)
	if err != nil {
		return fmt.Errorf("gmail list messages: %w", err)
	}

	msgID, err := ExtractMessageID(listOut)
	if err != nil {
		return fmt.Errorf("extract message ID: %w", err)
	}

	readOut, readErr := s.executor.Run(ctx, "gmail", "+read", "--id", msgID)
	var docID string
	var docErr error
	if readErr == nil {
		docID, docErr = ExtractDocID(readOut)
	}
	if readErr != nil || docErr != nil {
		rawBody, rawErr := s.fetchMessageRawRFC822(ctx, msgID)
		if rawErr != nil {
			if readErr != nil {
				return fmt.Errorf("gmail read message %s: %w; fallback raw: %v", msgID, readErr, rawErr)
			}
			return fmt.Errorf("extract doc ID from email: %w; fallback raw: %v", docErr, rawErr)
		}
		docID, docErr = ExtractDocID(rawBody)
		if docErr != nil {
			return fmt.Errorf("extract doc ID from raw RFC822: %w", docErr)
		}
	}

	s.docID = docID
	return nil
}

// Contribute sets the DocID on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	if s.docID == "" {
		return fmt.Errorf("gmail source has no doc ID; was Fetch called?")
	}
	report.DocID = s.docID
	return nil
}
