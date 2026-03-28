package pipeline_test

import (
	"testing"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

func TestWeekRange_CurrentWeek(t *testing.T) {
	// 2026-03-28 is a Saturday; week should be 2026-03-22 (Sun) to 2026-03-28 (Sat)
	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)

	wr := pipeline.WeekRangeFor(now, loc)

	wantStart := time.Date(2026, 3, 22, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 3, 28, 23, 59, 59, 0, loc)

	if !wr.Start.Equal(wantStart) {
		t.Errorf("Start: want %v, got %v", wantStart, wr.Start)
	}
	if !wr.End.Equal(wantEnd) {
		t.Errorf("End: want %v, got %v", wantEnd, wr.End)
	}
}

func TestWeekRange_OnSunday(t *testing.T) {
	// 2026-03-22 is a Sunday; start should be same day
	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 22, 8, 0, 0, 0, loc)

	wr := pipeline.WeekRangeFor(now, loc)

	wantStart := time.Date(2026, 3, 22, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 3, 28, 23, 59, 59, 0, loc)

	if !wr.Start.Equal(wantStart) {
		t.Errorf("Start: want %v, got %v", wantStart, wr.Start)
	}
	if !wr.End.Equal(wantEnd) {
		t.Errorf("End: want %v, got %v", wantEnd, wr.End)
	}
}

func TestWeekRange_EmailDateLabel(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)
	wr := pipeline.WeekRangeFor(now, loc)

	// Email search uses "22 March 2026"
	if got := wr.EmailDateLabel(); got != "22 March 2026" {
		t.Errorf("EmailDateLabel: want %q, got %q", "22 March 2026", got)
	}
}

func TestWeekRange_HeaderLabel(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)
	wr := pipeline.WeekRangeFor(now, loc)

	// Header uses "22 March 2026 - 28 March 2026"
	want := "22 March 2026 - 28 March 2026"
	if got := wr.HeaderLabel(); got != want {
		t.Errorf("HeaderLabel: want %q, got %q", want, got)
	}
}
