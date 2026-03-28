package pipeline

import (
	"context"
	"fmt"
	"time"
)

// WeekRange represents an inclusive Sunday-to-Saturday date range.
type WeekRange struct {
	Start time.Time // Sunday 00:00:00
	End   time.Time // Saturday 23:59:59
}

// WeekRangeFor calculates the Sunday–Saturday week that contains t.
func WeekRangeFor(t time.Time, loc *time.Location) WeekRange {
	t = t.In(loc)
	weekday := int(t.Weekday()) // Sunday=0, Monday=1, ..., Saturday=6
	daysToSunday := weekday
	sunday := time.Date(t.Year(), t.Month(), t.Day()-daysToSunday, 0, 0, 0, 0, loc)
	saturday := time.Date(sunday.Year(), sunday.Month(), sunday.Day()+6, 23, 59, 59, 0, loc)
	return WeekRange{Start: sunday, End: saturday}
}

// EmailDateLabel returns the start date formatted for Gmail search, e.g. "22 March 2026".
func (w WeekRange) EmailDateLabel() string {
	return w.Start.Format("2 January 2006")
}

// HeaderLabel returns the date range for the report header, e.g. "22 March 2026 - 28 March 2026".
func (w WeekRange) HeaderLabel() string {
	return fmt.Sprintf("%s - %s", w.Start.Format("2 January 2006"), w.End.Format("2 January 2006"))
}

// ReportData holds all collected data used to render the weekly report.
type ReportData struct {
	ReportName string
	Week       WeekRange
	DocID      string
	PRsByRepo  map[string]*RepoPRs // keyed by repo NameWithOwner
	Events     []CalendarEvent
}

// RepoPRs holds authored and reviewed PRs for a single repository.
type RepoPRs struct {
	RepoName    string
	Implemented []PR
	Reviewed    []PR
}

// PR represents a single pull request.
type PR struct {
	Title string
	URL   string
}

// CalendarEvent represents a single calendar event.
type CalendarEvent struct {
	Title string
	Date  string // formatted as "2 January 2006"
}

// DataSource is implemented by each data fetcher.
type DataSource interface {
	Name() string
	Fetch(ctx context.Context, week WeekRange) error
	Contribute(report *ReportData) error
}
