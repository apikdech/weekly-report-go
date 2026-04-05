package notifier

import (
	"errors"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestStartEvent_ImplementsInterface(t *testing.T) {
	event := &StartEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != "start" {
		t.Errorf("expected Type() to return 'start', got %q", event.Type())
	}

	if !event.Timestamp().Equal(time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC)) {
		t.Errorf("Timestamp() returned unexpected value")
	}
}

func TestFailedEvent_ImplementsInterface(t *testing.T) {
	event := &FailedEvent{
		WeekRange: "22 March 2026 - 28 March 2026",
		Error:     errTest,
		EventTime: time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != "failed" {
		t.Errorf("expected Type() to return 'failed', got %q", event.Type())
	}
}

func TestFinishedEvent_ImplementsInterface(t *testing.T) {
	event := &FinishedEvent{
		WeekRange:  "22 March 2026 - 28 March 2026",
		DocID:      "abc123",
		DocURL:     "https://docs.google.com/document/d/abc123/edit",
		ReportPath: "/tmp/report.md",
		EventTime:  time.Date(2026, 3, 28, 9, 15, 0, 0, time.UTC),
	}

	var _ NotificationEvent = event

	if event.Type() != "finished" {
		t.Errorf("expected Type() to return 'finished', got %q", event.Type())
	}
}
