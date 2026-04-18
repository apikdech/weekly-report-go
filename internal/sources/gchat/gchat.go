package gchat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

type chatMessage struct {
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
	Sender     struct {
		Name string `json:"name"`
	} `json:"sender"`
	Text string `json:"text"`
}

type messagesResponse struct {
	Messages []chatMessage `json:"messages"`
}

// PickLatestBySender parses a gws chat spaces messages list JSON response,
// filters by senderName, and returns the text of the message with the latest
// createTime. Returns an empty string (no error) when no message matches.
func PickLatestBySender(data []byte, senderName string) (string, error) {
	var resp messagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse chat messages response: %w", err)
	}

	var latestTime time.Time
	var latestText string
	for _, msg := range resp.Messages {
		if msg.Sender.Name != senderName {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, msg.CreateTime)
		if err != nil {
			// Fall back to RFC3339 without nanoseconds
			t, err = time.Parse(time.RFC3339, msg.CreateTime)
			if err != nil {
				log.Printf("[gchat] skipping message %q: unparseable createTime %q", msg.Name, msg.CreateTime)
				continue
			}
		}
		if t.After(latestTime) {
			latestTime = t
			latestText = msg.Text
		}
	}
	return latestText, nil
}

// Source fetches the Key Metrics / OMTM text from a Google Chat space.
type Source struct {
	executor   *gws.Executor
	spacesID   string
	senderName string
	keyMetrics string
	fetched    bool
}

// NewSource creates a GChat Source.
func NewSource(executor *gws.Executor, spacesID, senderName string) *Source {
	return &Source{
		executor:   executor,
		spacesID:   spacesID,
		senderName: senderName,
	}
}

// Name implements DataSource.
func (s *Source) Name() string { return "gchat" }

// Fetch lists messages from the configured space starting from week.Start and
// picks the latest message matching the configured sender.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	parent := "spaces/" + s.spacesID
	filter := fmt.Sprintf(`createTime > "%s"`, week.Start.UTC().Format("2006-01-02T15:04:05Z"))
	params := fmt.Sprintf(`{"parent":%q,"filter":%q}`, parent, filter)

	out, err := s.executor.Run(ctx, "chat", "spaces", "messages", "list", "--params", params)
	if err != nil {
		log.Printf("[gchat] error listing messages: %v", err)
		return nil
	}

	text, err := PickLatestBySender(out, s.senderName)
	if err != nil {
		return err
	}
	s.keyMetrics = text
	s.fetched = true
	return nil
}

// Contribute sets KeyMetrics on the report. A missing message is not an error.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	if !s.fetched {
		log.Printf("[gchat] error contributing: Contribute called before Fetch")
		return nil
	}
	report.KeyMetrics = s.keyMetrics
	return nil
}
