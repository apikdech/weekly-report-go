package pipeline

import (
	"context"
	"fmt"
	"time"
)

// dateFormatLayout is the Go time layout for labels like "02 March 2026" (day with leading zero).
const dateFormatLayout = "02 January 2006"

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

// EmailDateLabel returns the start date formatted for Gmail search, e.g. "02 March 2026".
func (w WeekRange) EmailDateLabel() string {
	return w.Start.Format(dateFormatLayout)
}

// HeaderLabel returns the date range for the report header, e.g. "02 March 2026 - 08 March 2026".
func (w WeekRange) HeaderLabel() string {
	return fmt.Sprintf("%s - %s", w.Start.Format(dateFormatLayout), w.End.Format(dateFormatLayout))
}

// ReportData holds all collected data used to render the weekly report.
type ReportData struct {
	ReportName           string
	Week                 WeekRange
	DocID                string
	PRsByRepo            map[string]*RepoPRs // keyed by repo NameWithOwner
	Events               []CalendarEvent
	OutOfOfficeDates     []string // sorted unique, formatted as "2 January 2006"
	KeyMetrics           string   // raw text from Google Chat spaces bot message
	NextActions          []string // from REPORT_NEXT_ACTIONS (comma-separated), rendered as numbered list
	TechnologyHighlights []TechHighlight
}

// TechHighlight represents a single analyzed technical article.
type TechHighlight struct {
	Title      string
	URL        string
	Highlights string
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
