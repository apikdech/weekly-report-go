package calendar_test

import (
	"testing"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
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
	events, ooo, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ooo) != 0 {
		t.Fatalf("expected no OOO dates, got %v", ooo)
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
	events, ooo, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ooo) != 0 {
		t.Fatalf("expected no OOO dates, got %v", ooo)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Date != "24 March 2026" {
		t.Errorf("expected '24 March 2026', got %q", events[0].Date)
	}
}

func TestParseEvents_SkipsOutOfOfficeAndWorkingLocation(t *testing.T) {
	input := []byte(`{
	  "items": [
	    {
	      "summary": "Team sync",
	      "eventType": "default",
	      "start": {"dateTime": "2026-03-23T10:00:00+07:00"}
	    },
	    {
	      "summary": "Out of office",
	      "eventType": "outOfOffice",
	      "start": {"dateTime": "2026-03-23T00:00:00+07:00"}
	    },
	    {
	      "summary": "Working elsewhere",
	      "eventType": "workingLocation",
	      "start": {"dateTime": "2026-03-24T09:00:00+07:00"}
	    }
	  ]
	}`)
	events, ooo, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := calendar.FormatOOODates(ooo)
	if len(got) != 1 || got[0] != "23 March 2026" {
		t.Fatalf("expected OOO [23 March 2026], got %v", got)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Team sync" {
		t.Errorf("expected Team sync, got %q", events[0].Title)
	}
}

func TestParseEvents_SkipsSelfDeclined(t *testing.T) {
	input := []byte(`{
	  "items": [
	    {
	      "summary": "Optional sync",
	      "start": {"dateTime": "2026-03-23T10:00:00+07:00"},
	      "attendees": [
	        {
	          "email": "other@example.com",
	          "responseStatus": "accepted",
	          "self": false
	        },
	        {
	          "comment": "Declined because I am out of office",
	          "email": "ricky.setiawan@gdplabs.id",
	          "responseStatus": "declined",
	          "self": true
	        }
	      ]
	    },
	    {
	      "summary": "Stand-up",
	      "start": {"dateTime": "2026-03-24T09:00:00+07:00"},
	      "attendees": [
	        {"email": "a@example.com", "responseStatus": "accepted", "self": true}
	      ]
	    }
	  ]
	}`)
	events, ooo, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ooo) != 0 {
		t.Fatalf("expected no OOO dates, got %v", ooo)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Stand-up" {
		t.Errorf("expected Stand-up, got %q", events[0].Title)
	}
}

func TestParseEvents_OutOfOffice_AllDayRange(t *testing.T) {
	input := []byte(`{
	  "items": [
	    {
	      "summary": "Vacation",
	      "eventType": "outOfOffice",
	      "start": {"date": "2026-03-23"},
	      "end": {"date": "2026-03-25"}
	    }
	  ]
	}`)
	events, ooo, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no regular events, got %d", len(events))
	}
	want := []string{"23 March 2026", "24 March 2026"}
	got := calendar.FormatOOODates(ooo)
	if len(got) != len(want) {
		t.Fatalf("expected OOO %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("OOO[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestParseEvents_OutOfOffice_DateTimeExclusiveEnd(t *testing.T) {
	input := []byte(`{
	  "items": [
	    {
	      "summary": "Vacation",
	      "eventType": "outOfOffice",
	      "start": {"dateTime": "2026-03-18T00:00:00+07:00"},
	      "end": {"dateTime": "2026-03-25T00:00:00+07:00"}
	    }
	  ]
	}`)
	events, ooo, err := calendar.ParseEvents(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no regular events, got %d", len(events))
	}
	got := calendar.FormatOOODates(ooo)
	if len(got) != 7 {
		t.Fatalf("expected 7 OOO days (18–24), got %d: %v", len(got), got)
	}
	if got[0] != "18 March 2026" || got[len(got)-1] != "24 March 2026" {
		t.Fatalf("expected range 18–24 Mar, got %v", got)
	}
}

func TestFilterOOOForReportWeek_WeekdaysOnly(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	// Report week Sun 22 – Sat 28 March 2026
	week := pipeline.WeekRange{
		Start: time.Date(2026, 3, 22, 0, 0, 0, 0, loc),
		End:   time.Date(2026, 3, 28, 23, 59, 59, 0, loc),
	}
	// Raw OOO days 18–24 Mar (from exclusive end at 25th 00:00)
	var raw []time.Time
	for d := 18; d <= 24; d++ {
		raw = append(raw, time.Date(2026, 3, d, 0, 0, 0, 0, loc))
	}
	filtered := calendar.FilterOOOForReportWeek(raw, week)
	got := calendar.FormatOOODates(filtered)
	want := []string{"23 March 2026", "24 March 2026"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}
