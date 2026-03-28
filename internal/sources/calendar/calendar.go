package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

type eventsResponse struct {
	Items []struct {
		Summary string `json:"summary"`
		Start   struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
	} `json:"items"`
}

// ParseEvents parses the gws calendar events list JSON into CalendarEvents.
func ParseEvents(data []byte) ([]pipeline.CalendarEvent, error) {
	var resp eventsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse calendar events: %w", err)
	}

	var events []pipeline.CalendarEvent
	for _, item := range resp.Items {
		if item.Summary == "" {
			continue
		}
		dateStr := item.Start.DateTime
		if dateStr == "" {
			dateStr = item.Start.Date
		}
		formatted, err := formatDate(dateStr)
		if err != nil {
			continue // skip unparseable dates
		}
		events = append(events, pipeline.CalendarEvent{
			Title: item.Summary,
			Date:  formatted,
		})
	}
	return events, nil
}

func formatDate(s string) (string, error) {
	// Try RFC3339 (dateTime) first, then date-only
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2 January 2006"), nil
		}
	}
	return "", fmt.Errorf("unrecognized date format: %q", s)
}

// Source fetches calendar events for the week via gws CLI.
type Source struct {
	executor *gws.Executor
	events   []pipeline.CalendarEvent
}

// NewSource creates a CalendarSource.
func NewSource(executor *gws.Executor) *Source {
	return &Source{executor: executor}
}

// Name implements DataSource.
func (s *Source) Name() string { return "calendar" }

// Fetch retrieves calendar events for the week range.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	params := fmt.Sprintf(
		`{"calendarId":"primary","timeMin":%q,"timeMax":%q,"singleEvents":true,"orderBy":"startTime"}`,
		week.Start.Format(time.RFC3339),
		week.End.Format(time.RFC3339),
	)
	out, err := s.executor.Run(ctx, "calendar", "events", "list", "--params", params)
	if err != nil {
		return fmt.Errorf("calendar events list: %w", err)
	}

	events, err := ParseEvents(out)
	if err != nil {
		return err
	}
	s.events = events
	return nil
}

// Contribute sets Events on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	report.Events = s.events
	return nil
}
