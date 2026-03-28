package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

var docIDRegexp = regexp.MustCompile(`docs\.google\.com/document/d/([a-zA-Z0-9_-]+)`)

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
func ExtractDocID(body []byte) (string, error) {
	matches := docIDRegexp.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("no Google Docs URL found in email body")
	}
	return string(matches[1]), nil
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

	readOut, err := s.executor.Run(ctx, "gmail", "+read", "--id", msgID)
	if err != nil {
		return fmt.Errorf("gmail read message %s: %w", msgID, err)
	}

	docID, err := ExtractDocID(readOut)
	if err != nil {
		return fmt.Errorf("extract doc ID from email: %w", err)
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
