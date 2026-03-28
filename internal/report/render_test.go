package report_test

import (
	"strings"
	"testing"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	"github.com/apikdech/gws-weekly-report/internal/report"
)

func testReportData() *pipeline.ReportData {
	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)
	week := pipeline.WeekRangeFor(now, loc)

	data := &pipeline.ReportData{
		ReportName: "Ricky Setiawan",
		Week:       week,
		DocID:      "abc123",
		PRsByRepo:  map[string]*pipeline.RepoPRs{},
		Events:     []pipeline.CalendarEvent{},
	}
	data.PRsByRepo["org/repo-a"] = &pipeline.RepoPRs{
		RepoName: "org/repo-a",
		Implemented: []pipeline.PR{
			{Title: "Add feature X", URL: "https://github.com/org/repo-a/pull/1"},
		},
		Reviewed: []pipeline.PR{
			{Title: "Fix bug Y", URL: "https://github.com/org/repo-a/pull/2"},
		},
	}
	data.Events = []pipeline.CalendarEvent{
		{Title: "Sprint Planning", Date: "23 March 2026"},
	}
	return data
}

func TestRender_ContainsHeader(t *testing.T) {
	data := testReportData()
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "# [Weekly Report: Ricky Setiawan] 22 March 2026 - 28 March 2026"
	if !strings.Contains(out, want) {
		t.Errorf("output missing header %q\ngot:\n%s", want, out)
	}
}

func TestRender_ContainsPR(t *testing.T) {
	data := testReportData()
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[Add feature X](https://github.com/org/repo-a/pull/1)") {
		t.Errorf("output missing implemented PR link\ngot:\n%s", out)
	}
	if !strings.Contains(out, "[Fix bug Y](https://github.com/org/repo-a/pull/2)") {
		t.Errorf("output missing reviewed PR link\ngot:\n%s", out)
	}
}

func TestRender_ContainsCalendarEvent(t *testing.T) {
	data := testReportData()
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Sprint Planning (23 March 2026)") {
		t.Errorf("output missing calendar event\ngot:\n%s", out)
	}
}

func TestRender_OutOfOfficeNumberedDates(t *testing.T) {
	data := testReportData()
	data.OutOfOfficeDates = []string{"23 March 2026", "24 March 2026"}
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1. 23 March 2026") {
		t.Errorf("missing first OOO line\ngot:\n%s", out)
	}
	if !strings.Contains(out, "2. 24 March 2026") {
		t.Errorf("missing second OOO line\ngot:\n%s", out)
	}
	if strings.Contains(out, "March 2026\n\n2.") {
		t.Errorf("unexpected blank line between OOO items\ngot:\n%s", out)
	}
}

func TestRender_KeyMetricsPresent(t *testing.T) {
	data := testReportData()
	data.KeyMetrics = "DAU: 1000"
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "DAU: 1000") {
		t.Errorf("output missing KeyMetrics text\ngot:\n%s", out)
	}
}

func TestRender_KeyMetricsEmpty(t *testing.T) {
	data := testReportData()
	data.KeyMetrics = ""
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "## **Key Metrics / OMTM**") {
		t.Errorf("output missing Key Metrics heading\ngot:\n%s", out)
	}
	if strings.Contains(out, "DAU:") {
		t.Errorf("output should not contain placeholder KeyMetrics text\ngot:\n%s", out)
	}
}

func TestRender_ContainsEmptySections(t *testing.T) {
	data := testReportData()
	out, err := report.Render(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, section := range []string{
		"## **Issues**",
		"## **Key Metrics / OMTM**",
		"## **Next Actions**",
		"## **Technology, Business, Communication, Leadership, Management & Marketing**",
		"## **Out of Office**",
	} {
		if !strings.Contains(out, section) {
			t.Errorf("output missing section %q\ngot:\n%s", section, out)
		}
	}
}
