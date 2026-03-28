package calendar_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/sources/calendar"
)

func TestParseEvents(t *testing.T) {
	input := []byte(`{
	  "items": [
	    {
	      "summary": "Sprint Planning",
	      "start": {"dateTime": "2026-03-23T10:00:00+07:00"}
	    },
	    {
	      "summary": "All Hands",
	      "start": {"dateTime": "2026-03-25T14:00:00+07:00"}
	    }
	  ]
	}`)
	events, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Title != "Sprint Planning" {
		t.Errorf("expected Sprint Planning, got %q", events[0].Title)
	}
	if events[0].Date != "23 March 2026" {
		t.Errorf("expected '23 March 2026', got %q", events[0].Date)
	}
}

func TestParseEvents_AllDayEvent(t *testing.T) {
	// All-day events use "date" instead of "dateTime"
	input := []byte(`{
	  "items": [
	    {
	      "summary": "Public Holiday",
	      "start": {"date": "2026-03-24"}
	    }
	  ]
	}`)
	events, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Date != "24 March 2026" {
		t.Errorf("expected '24 March 2026', got %q", events[0].Date)
	}
}
