package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

type eventAttendee struct {
	Self           bool   `json:"self"`
	ResponseStatus string `json:"responseStatus"`
}

type calendarListItem struct {
	Summary   string          `json:"summary"`
	EventType string          `json:"eventType"`
	Attendees []eventAttendee `json:"attendees"`
	Start     struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"end"`
}

type eventsResponse struct {
	Items []calendarListItem `json:"items"`
}

// ParseEvents parses the gws calendar events list JSON into regular events and
// raw out-of-office calendar days (eventType outOfOffice). workingLocation is omitted from both.
// Call FilterOOOForReportWeek then formatSortedUniqueDates when building the report.
func ParseEvents(data []byte) ([]pipeline.CalendarEvent, []time.Time, error) {
	var resp eventsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse calendar events: %w", err)
	}

	var events []pipeline.CalendarEvent
	var oooDays []time.Time
	for _, item := range resp.Items {
		if item.EventType == "outOfOffice" {
			if selfAttendeeDeclined(item.Attendees) {
				continue
			}
			days, err := oooDaysFromItem(item)
			if err != nil {
				continue
			}
			oooDays = append(oooDays, days...)
			continue
		}
		if item.EventType == "workingLocation" {
			continue
		}
		if selfAttendeeDeclined(item.Attendees) {
			continue
		}
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
	return events, oooDays, nil
}

// FilterOOOForReportWeek keeps OOO days that fall within the report week (Sunday–Saturday
// inclusive, using week.Start's location) and are weekdays (Monday–Friday).
func FilterOOOForReportWeek(days []time.Time, week pipeline.WeekRange) []time.Time {
	loc := week.Start.Location()
	ws := week.Start.In(loc)
	weekStart := time.Date(ws.Year(), ws.Month(), ws.Day(), 0, 0, 0, 0, loc)
	we := week.End.In(loc)
	weekEnd := time.Date(we.Year(), we.Month(), we.Day(), 0, 0, 0, 0, loc)

	var out []time.Time
	for _, d := range days {
		dloc := d.In(loc)
		cd := time.Date(dloc.Year(), dloc.Month(), dloc.Day(), 0, 0, 0, 0, loc)
		if cd.Before(weekStart) || cd.After(weekEnd) {
			continue
		}
		switch cd.Weekday() {
		case time.Saturday, time.Sunday:
			continue
		}
		out = append(out, cd)
	}
	return out
}

// oooDaysFromItem expands an out-of-office event into calendar dates in the event timezone.
// All-day events use end.date as exclusive (Google Calendar convention).
// Timed blocks that end at local midnight on a later day use the same exclusive end rule.
func oooDaysFromItem(item calendarListItem) ([]time.Time, error) {
	if item.Start.Date != "" {
		start, err := time.Parse("2006-01-02", item.Start.Date)
		if err != nil {
			return nil, err
		}
		if item.End.Date != "" {
			end, err := time.Parse("2006-01-02", item.End.Date)
			if err != nil {
				return nil, err
			}
			var out []time.Time
			for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
				out = append(out, d)
			}
			return out, nil
		}
		return []time.Time{start}, nil
	}
	if item.Start.DateTime == "" {
		return nil, fmt.Errorf("ooo: no start time")
	}
	startT, err := time.Parse(time.RFC3339, item.Start.DateTime)
	if err != nil {
		return nil, err
	}
	loc := startT.Location()
	sy, sm, sd := startT.Date()
	day := time.Date(sy, sm, sd, 0, 0, 0, 0, loc)
	if item.End.DateTime == "" {
		return []time.Time{day}, nil
	}
	endT, err := time.Parse(time.RFC3339, item.End.DateTime)
	if err != nil {
		return []time.Time{day}, nil
	}
	ey, em, ed := endT.Date()
	endDay := time.Date(ey, em, ed, 0, 0, 0, 0, endT.Location())
	endIsMidnight := endT.Hour() == 0 && endT.Minute() == 0 && endT.Second() == 0 && endT.Nanosecond() == 0

	var out []time.Time
	if endIsMidnight && endDay.After(day) {
		// Multi-day OOO: end instant is start of the first day *after* the block (e.g. 25th 00:00 → last OOO day is 24th).
		for t := day; t.Before(endDay); t = t.AddDate(0, 0, 1) {
			out = append(out, t)
		}
		return out, nil
	}
	for t := day; !t.After(endDay); t = t.AddDate(0, 0, 1) {
		out = append(out, t)
	}
	return out, nil
}

// FormatOOODates returns sorted unique display strings ("2 January 2006") for each civil day.
func FormatOOODates(days []time.Time) []string {
	return formatSortedUniqueDates(days)
}

func formatSortedUniqueDates(days []time.Time) []string {
	if len(days) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(days))
	for _, d := range days {
		seen[d.Format("2006-01-02")] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		t, err := time.Parse("2006-01-02", k)
		if err != nil {
			continue
		}
		out = append(out, t.Format("2 January 2006"))
	}
	return out
}

// selfAttendeeDeclined is true when the authenticated user's attendee entry
// (Calendar API marks it with self: true) has responseStatus declined.
func selfAttendeeDeclined(attendees []eventAttendee) bool {
	for _, a := range attendees {
		if a.Self && strings.EqualFold(a.ResponseStatus, "declined") {
			return true
		}
	}
	return false
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
	oooDates []string
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

	events, oooRaw, err := ParseEvents(out)
	if err != nil {
		return err
	}
	s.events = events
	s.oooDates = formatSortedUniqueDates(FilterOOOForReportWeek(oooRaw, week))
	return nil
}

// Contribute sets Events and OutOfOfficeDates on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	report.Events = s.events
	report.OutOfOfficeDates = s.oooDates
	return nil
}
