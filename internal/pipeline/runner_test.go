package pipeline_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

// mockSource is a test DataSource that records calls and optionally returns errors.
type mockSource struct {
	name             string
	fetchErr         error
	fetchCalled      bool
	contributeErr    error
	contributeCalled bool
}

func (m *mockSource) Name() string { return m.name }
func (m *mockSource) Fetch(_ context.Context, _ pipeline.WeekRange) error {
	m.fetchCalled = true
	return m.fetchErr
}
func (m *mockSource) Contribute(r *pipeline.ReportData) error {
	m.contributeCalled = true
	return m.contributeErr
}

func TestRunner_RunsAllSources(t *testing.T) {
	s1 := &mockSource{name: "s1"}
	s2 := &mockSource{name: "s2"}

	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)
	week := pipeline.WeekRangeFor(now, loc)

	r := pipeline.NewRunner([]pipeline.DataSource{s1, s2})
	report := &pipeline.ReportData{Week: week}
	err := r.Run(context.Background(), report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s1.fetchCalled || !s2.fetchCalled {
		t.Error("expected both sources to be fetched")
	}
	if !s1.contributeCalled || !s2.contributeCalled {
		t.Error("expected both sources to contribute")
	}
}

func TestRunner_StopsOnFetchError(t *testing.T) {
	s1 := &mockSource{name: "s1", fetchErr: errors.New("fetch failed")}
	s2 := &mockSource{name: "s2"}

	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, loc)
	week := pipeline.WeekRangeFor(now, loc)

	r := pipeline.NewRunner([]pipeline.DataSource{s1, s2})
	report := &pipeline.ReportData{Week: week}
	err := r.Run(context.Background(), report)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s2.fetchCalled {
		t.Error("s2 should not have been fetched after s1 error")
	}
}
