package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestBuildProcessingEmbed(t *testing.T) {
	event := &ProcessingEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		Stage:     "Rendering markdown report",
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	embed := buildProcessingEmbed(event)

	if embed.Title != "⏳ Weekly Report — In Progress" {
		t.Errorf("Title: expected processing message, got %q", embed.Title)
	}

	if embed.Color != 0xf39c12 {
		t.Errorf("Color: expected orange (0xf39c12), got 0x%x", embed.Color)
	}

	if len(embed.Fields) != 3 {
		t.Errorf("Fields: expected 3 fields, got %d", len(embed.Fields))
	}
}

func TestBuildProcessingEmbed_OmitsEmptyStage(t *testing.T) {
	event := &ProcessingEvent{
		WeekRange: "Week",
		Stage:     "",
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	embed := buildProcessingEmbed(event)

	if len(embed.Fields) != 2 {
		t.Errorf("Fields: expected 2 fields when Stage empty, got %d", len(embed.Fields))
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

func TestDiscordHandler_Supports(t *testing.T) {
	handler := NewDiscordHandler("https://discord.com/api/webhooks/123/abc", 30, 1)

	if !handler.Supports(EventTypeStart) {
		t.Error("should support start events")
	}
	if !handler.Supports(EventTypeProcessing) {
		t.Error("should support processing events")
	}
	if !handler.Supports(EventTypeFailed) {
		t.Error("should support failed events")
	}
	if !handler.Supports(EventTypeFinished) {
		t.Error("should support finished events")
	}
	if handler.Supports(EventType("unknown")) {
		t.Error("should not support unknown events")
	}
}

func TestDiscordHandler_Handle_StartEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected application/json, got %s", contentType)
		}

		var payload DiscordWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		if len(payload.Embeds) != 1 {
			t.Errorf("expected 1 embed, got %d", len(payload.Embeds))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	handler := NewDiscordHandler(server.URL, 30, 0)
	event := &StartEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		EventTime: time.Now(),
	}

	handler.Handle(event)
}

func TestDiscordHandler_Handle_ProcessingEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	handler := NewDiscordHandler(server.URL, 30, 0)
	event := &ProcessingEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		Stage:     "Rendering",
		EventTime: time.Now(),
	}

	handler.Handle(event)
}

func TestDiscordHandler_Handle_FinishedEvent_WithAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", contentType)
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		payloadJSON := r.FormValue("payload_json")
		if payloadJSON == "" {
			t.Error("expected payload_json field")
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("expected file field: %v", err)
		} else {
			file.Close()
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "report-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("# Test Report")
	tmpFile.Close()

	handler := NewDiscordHandler(server.URL, 30, 0)
	event := &FinishedEvent{
		WeekRange:  "22 March 2026 - 28 March 2026",
		DocID:      "abc123",
		DocURL:     "https://docs.google.com/document/d/abc123/edit",
		ReportPath: tmpFile.Name(),
		EventTime:  time.Now(),
	}

	handler.Handle(event)
}
