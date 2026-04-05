package notifier

import (
	"testing"
	"time"
)

// customError is a custom error type for testing
type customError struct {
	msg string
}

func (e *customError) Error() string { return e.msg }

var errTestDiscord = &customError{msg: "test error"}

func TestBuildStartEmbed(t *testing.T) {
	event := &StartEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	embed := buildStartEmbed(event)

	if embed.Title != "📊 Weekly Report Generation Started" {
		t.Errorf("Title: expected start message, got %q", embed.Title)
	}

	if embed.Color != 0x3498db {
		t.Errorf("Color: expected blue (0x3498db), got 0x%x", embed.Color)
	}

	if len(embed.Fields) != 2 {
		t.Errorf("Fields: expected 2 fields, got %d", len(embed.Fields))
	}
}

func TestBuildFailedEmbed(t *testing.T) {
	event := &FailedEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		Error:     errTestDiscord,
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	embed := buildFailedEmbed(event)

	if embed.Title != "❌ Weekly Report Generation Failed" {
		t.Errorf("Title: expected failure message, got %q", embed.Title)
	}

	if embed.Color != 0xe74c3c {
		t.Errorf("Color: expected red (0xe74c3c), got 0x%x", embed.Color)
	}

	if len(embed.Fields) != 3 {
		t.Errorf("Fields: expected 3 fields, got %d", len(embed.Fields))
	}
}

func TestBuildFailedEmbed_TruncatesLongErrors(t *testing.T) {
	longError := ""
	for i := 0; i < 1100; i++ {
		longError += "a"
	}

	event := &FailedEvent{
		WeekRange: "Week",
		Error:     &customError{msg: longError},
		EventTime: time.Now(),
	}

	embed := buildFailedEmbed(event)

	errorField := embed.Fields[1]
	if len(errorField.Value) != 1024 {
		t.Errorf("Error field should be truncated to 1024 chars, got %d", len(errorField.Value))
	}
	if errorField.Value[len(errorField.Value)-3:] != "..." {
		t.Errorf("Error field should end with '...'")
	}
}

func TestBuildFinishedEmbed(t *testing.T) {
	event := &FinishedEvent{
		WeekRange:  "22 March 2026 - 28 March 2026",
		DocID:      "abc123",
		DocURL:     "https://docs.google.com/document/d/abc123/edit",
		ReportPath: "/tmp/report.md",
		EventTime:  time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	embed := buildFinishedEmbed(event)

	if embed.Title != "✅ Weekly Report Generation Complete" {
		t.Errorf("Title: expected success message, got %q", embed.Title)
	}

	if embed.Color != 0x2ecc71 {
		t.Errorf("Color: expected green (0x2ecc71), got 0x%x", embed.Color)
	}

	if len(embed.Fields) != 3 {
		t.Errorf("Fields: expected 3 fields, got %d", len(embed.Fields))
	}
}
