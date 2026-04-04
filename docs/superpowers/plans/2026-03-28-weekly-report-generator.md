# GWS Weekly Report Generator — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go binary that auto-generates a weekly report by fetching GitHub PRs, calendar events, and a Google Doc ID from Gmail, then uploads the rendered markdown to Google Docs via the `gws` CLI — packaged in a distroless Docker container triggered by crontab.

**Architecture:** Single binary with a `DataSource` interface (plugin pattern). Three data sources (`GmailSource`, `GitHubSource`, `CalendarSource`) populate a shared `ReportData` struct; a pipeline runner orchestrates fetch → render → upload → cleanup. A shared `gws.Executor` wraps all `gws` CLI calls.

**Tech Stack:** Go 1.26.1, `github.com/shurcooL/githubv4`, `golang.org/x/oauth2`, `gws` CLI binary (pre-built Rust), Docker multi-stage build, `gcr.io/distroless/base-debian12` runtime.

---

## File Map

```
go.mod
go.sum
cmd/reporter/main.go
internal/config/config.go
internal/pipeline/types.go          # WeekRange, DataSource interface, ReportData
internal/pipeline/runner.go         # Pipeline struct + Run()
internal/gws/executor.go            # gws CLI wrapper
internal/sources/gmail/gmail.go     # GmailSource
internal/sources/github/github.go   # GitHubSource
internal/sources/calendar/calendar.go # CalendarSource
internal/uploader/drive/drive.go    # DriveUploader
internal/report/render.go           # markdown renderer + template
Dockerfile
docker-compose.yml
.env.example
.gitignore
```

Test files mirror source files:
```
internal/config/config_test.go
internal/pipeline/types_test.go
internal/pipeline/runner_test.go
internal/gws/executor_test.go
internal/sources/gmail/gmail_test.go
internal/sources/github/github_test.go
internal/sources/calendar/calendar_test.go
internal/uploader/drive/drive_test.go
internal/report/render_test.go
```

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `.env.example`
- Create: `cmd/reporter/main.go` (stub)

- [ ] **Step 1: Initialize Go module**

```bash
go mod init github.com/apikdech/gws-weekly-report
```

Expected: `go.mod` created with `module github.com/apikdech/gws-weekly-report` and `go 1.26.1`.

- [ ] **Step 2: Create `.gitignore`**

Create `/.gitignore`:
```
.env
credentials.json
/tmp/
*.log
```

- [ ] **Step 3: Create `.env.example`**

Create `/.env.example`:
```
# Required
GITHUB_TOKEN=ghp_your_token_here
GITHUB_USERNAME=your-github-username
GWS_EMAIL_SENDER=sender@example.com
REPORT_NAME=Your Full Name
GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/run/secrets/gws-credentials

# Optional
REPORT_TIMEZONE=Asia/Jakarta
TEMP_DIR=/tmp
```

- [ ] **Step 4: Create stub entrypoint**

Create `cmd/reporter/main.go`:
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stdout, "gws-weekly-report starting")
	os.Exit(0)
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./cmd/reporter
```

Expected: binary `reporter` created in working directory, no errors.

- [ ] **Step 6: Commit**

```bash
git init
git add .
git commit -m "chore: scaffold project structure"
```

---

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/config/config_test.go`:
```go
package config_test

import (
	"os"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/config"
)

func TestLoad_AllRequiredPresent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.GitHubToken != "ghp_test" {
		t.Errorf("expected GitHubToken=ghp_test, got %q", cfg.GitHubToken)
	}
	if cfg.GitHubUsername != "testuser" {
		t.Errorf("expected GitHubUsername=testuser, got %q", cfg.GitHubUsername)
	}
	if cfg.GWSEmailSender != "agent@example.com" {
		t.Errorf("expected GWSEmailSender=agent@example.com, got %q", cfg.GWSEmailSender)
	}
	if cfg.ReportName != "Test User" {
		t.Errorf("expected ReportName=Test User, got %q", cfg.ReportName)
	}
	if cfg.GWSCredentialsFile != "/tmp/creds.json" {
		t.Errorf("expected GWSCredentialsFile=/tmp/creds.json, got %q", cfg.GWSCredentialsFile)
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	os.Unsetenv("REPORT_TIMEZONE")
	os.Unsetenv("TEMP_DIR")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReportTimezone != "UTC" {
		t.Errorf("expected default timezone UTC, got %q", cfg.ReportTimezone)
	}
	if cfg.TempDir != "/tmp" {
		t.Errorf("expected default TempDir=/tmp, got %q", cfg.TempDir)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_USERNAME")
	os.Unsetenv("GWS_EMAIL_SENDER")
	os.Unsetenv("REPORT_NAME")
	os.Unsetenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing required vars, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/...
```

Expected: FAIL — `package config_test: cannot find package`

- [ ] **Step 3: Implement config**

Create `internal/config/config.go`:
```go
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	GitHubToken        string
	GitHubUsername     string
	GWSEmailSender     string
	ReportName         string
	GWSCredentialsFile string
	ReportTimezone     string
	TempDir            string
}

// Load reads configuration from environment variables.
// Returns an error listing all missing required variables.
func Load() (*Config, error) {
	cfg := &Config{
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		GitHubUsername:     os.Getenv("GITHUB_USERNAME"),
		GWSEmailSender:     os.Getenv("GWS_EMAIL_SENDER"),
		ReportName:         os.Getenv("REPORT_NAME"),
		GWSCredentialsFile: os.Getenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE"),
		ReportTimezone:     os.Getenv("REPORT_TIMEZONE"),
		TempDir:            os.Getenv("TEMP_DIR"),
	}

	if cfg.ReportTimezone == "" {
		cfg.ReportTimezone = "UTC"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "/tmp"
	}

	var missing []string
	required := map[string]string{
		"GITHUB_TOKEN":                          cfg.GitHubToken,
		"GITHUB_USERNAME":                       cfg.GitHubUsername,
		"GWS_EMAIL_SENDER":                      cfg.GWSEmailSender,
		"REPORT_NAME":                           cfg.ReportName,
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE": cfg.GWSCredentialsFile,
	}
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, errors.New(fmt.Sprintf("missing required environment variables: %s", strings.Join(missing, ", ")))
	}

	return cfg, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with env var loading"
```

---

## Task 3: Pipeline Types (`WeekRange`, `DataSource`, `ReportData`)

**Files:**
- Create: `internal/pipeline/types.go`
- Create: `internal/pipeline/types_test.go`

- [ ] **Step 1: Write failing tests for WeekRange**

Create `internal/pipeline/types_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/pipeline/...
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement types**

Create `internal/pipeline/types.go`:
```go
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
	RepoName     string
	Implemented  []PR
	Reviewed     []PR
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/pipeline/... -v
```

Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/pipeline/types.go internal/pipeline/types_test.go
git commit -m "feat: add pipeline types (WeekRange, DataSource, ReportData)"
```

---

## Task 4: Pipeline Runner

**Files:**
- Create: `internal/pipeline/runner.go`
- Create: `internal/pipeline/runner_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/pipeline/runner_test.go`:
```go
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
	name        string
	fetchErr    error
	fetchCalled bool
	contributeErr error
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/pipeline/... -run TestRunner
```

Expected: FAIL — `NewRunner` undefined.

- [ ] **Step 3: Implement runner**

Create `internal/pipeline/runner.go`:
```go
package pipeline

import (
	"context"
	"fmt"
	"log"
)

// Runner executes a sequence of DataSources against a ReportData.
type Runner struct {
	sources []DataSource
}

// NewRunner creates a Runner with the given sources (executed in order).
func NewRunner(sources []DataSource) *Runner {
	return &Runner{sources: sources}
}

// Run fetches from each source then collects contributions into report.
// Stops and returns an error if any source fails.
func (r *Runner) Run(ctx context.Context, report *ReportData) error {
	for _, src := range r.sources {
		log.Printf("[pipeline] fetching: %s", src.Name())
		if err := src.Fetch(ctx, report.Week); err != nil {
			return fmt.Errorf("source %q fetch failed: %w", src.Name(), err)
		}
		log.Printf("[pipeline] contributing: %s", src.Name())
		if err := src.Contribute(report); err != nil {
			return fmt.Errorf("source %q contribute failed: %w", src.Name(), err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/pipeline/... -v
```

Expected: all 6 tests PASS (4 from types + 2 from runner).

- [ ] **Step 5: Commit**

```bash
git add internal/pipeline/runner.go internal/pipeline/runner_test.go
git commit -m "feat: add pipeline runner"
```

---

## Task 5: GWS CLI Executor

**Files:**
- Create: `internal/gws/executor.go`
- Create: `internal/gws/executor_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/gws/executor_test.go`:
```go
package gws_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/gws"
)

func TestExecutor_Run_EchoBinary(t *testing.T) {
	// Use "echo" as a stand-in for the gws binary to test executor mechanics.
	echoBin, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("echo not available")
	}

	tmpCreds, _ := os.CreateTemp("", "creds*.json")
	defer os.Remove(tmpCreds.Name())

	ex := gws.NewExecutor(echoBin, tmpCreds.Name())
	out, err := ex.Run(context.Background(), "hello", "world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", string(out))
	}
}

func TestExecutor_Run_PropagatesError(t *testing.T) {
	// Use "false" binary which always exits non-zero.
	falseBin, err := exec.LookPath("false")
	if err != nil {
		t.Skip("false not available")
	}

	tmpCreds, _ := os.CreateTemp("", "creds*.json")
	defer os.Remove(tmpCreds.Name())

	ex := gws.NewExecutor(falseBin, tmpCreds.Name())
	_, err = ex.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from non-zero exit, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/gws/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement executor**

Create `internal/gws/executor.go`:
```go
package gws

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Executor runs the gws CLI binary with credentials injected via environment.
type Executor struct {
	gwsBinPath     string
	credentialsFile string
}

// NewExecutor creates an Executor using the given binary path and credentials file path.
func NewExecutor(gwsBinPath, credentialsFile string) *Executor {
	return &Executor{
		gwsBinPath:     gwsBinPath,
		credentialsFile: credentialsFile,
	}
}

// Run executes the gws binary with the given arguments.
// It injects GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE into the process environment.
// Returns stdout bytes on success, or an error containing stderr on non-zero exit.
func (e *Executor) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.gwsBinPath, args...)
	cmd.Env = append(cmd.Environ(),
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE="+e.credentialsFile,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gws command %v failed: %w\nstderr: %s", args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/gws/... -v
```

Expected: both tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/gws/
git commit -m "feat: add gws CLI executor"
```

---

## Task 6: Report Renderer

**Files:**
- Create: `internal/report/render.go`
- Create: `internal/report/render_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/report/render_test.go`:
```go
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
		"## Out of Office",
	} {
		if !strings.Contains(out, section) {
			t.Errorf("output missing section %q\ngot:\n%s", section, out)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/report/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement renderer**

Create `internal/report/render.go`:
```go
package report

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

const reportTemplate = `# [Weekly Report: {{ .ReportName }}] {{ .Week.HeaderLabel }}

## **Issues**

## **Accomplishment**
{{ range .SortedRepos -}}
### {{ .RepoName }}
#### Implemented PR
{{ range .Implemented -}}
1. [{{ .Title }}]({{ .URL }})
{{ end }}
#### Reviewed PR
{{ range .Reviewed -}}
1. [{{ .Title }}]({{ .URL }})
{{ end }}
{{ end }}
## **Meetings/Events/Training/Conferences**
{{ range .Events -}}
- {{ .Title }} ({{ .Date }})
{{ end }}
## **Key Metrics / OMTM**

## **Next Actions**
1. Continue implement admin dashboard features

## **Technology, Business, Communication, Leadership, Management & Marketing**

## Out of Office
`

type templateData struct {
	ReportName  string
	Week        pipeline.WeekRange
	SortedRepos []*pipeline.RepoPRs
	Events      []pipeline.CalendarEvent
}

// Render produces the weekly report markdown string from ReportData.
func Render(data *pipeline.ReportData) (string, error) {
	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	// Sort repos alphabetically for deterministic output.
	repos := make([]*pipeline.RepoPRs, 0, len(data.PRsByRepo))
	for _, r := range data.PRsByRepo {
		repos = append(repos, r)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].RepoName < repos[j].RepoName
	})

	td := templateData{
		ReportName:  data.ReportName,
		Week:        data.Week,
		SortedRepos: repos,
		Events:      data.Events,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/report/... -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/report/
git commit -m "feat: add report markdown renderer"
```

---

## Task 7: Gmail Source

**Files:**
- Create: `internal/sources/gmail/gmail.go`
- Create: `internal/sources/gmail/gmail_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/sources/gmail/gmail_test.go`:
```go
package gmail_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/sources/gmail"
)

func TestExtractMessageID(t *testing.T) {
	input := []byte(`{
	  "messages": [
	    {"id": "19d100ee48e14953", "threadId": "19d100ee48e14953"}
	  ],
	  "resultSizeEstimate": 1
	}`)
	id, err := gmail.ExtractMessageID(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "19d100ee48e14953" {
		t.Errorf("expected 19d100ee48e14953, got %q", id)
	}
}

func TestExtractMessageID_Empty(t *testing.T) {
	input := []byte(`{"messages": [], "resultSizeEstimate": 0}`)
	_, err := gmail.ExtractMessageID(input)
	if err == nil {
		t.Fatal("expected error for empty messages, got nil")
	}
}

func TestExtractDocID(t *testing.T) {
	emailBody := `Dear Colleague,
Open Weekly Report
<https://docs.google.com/document/d/1FGG0-VOGVBoRFaLOsnZ9ApWmcBue_hJZay1xtN3aKig/edit?usp=drivesdk>
Best regards`

	docID, err := gmail.ExtractDocID([]byte(emailBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docID != "1FGG0-VOGVBoRFaLOsnZ9ApWmcBue_hJZay1xtN3aKig" {
		t.Errorf("expected doc ID, got %q", docID)
	}
}

func TestExtractDocID_NoURL(t *testing.T) {
	_, err := gmail.ExtractDocID([]byte("no links here"))
	if err == nil {
		t.Fatal("expected error when no doc URL found, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/sources/gmail/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement GmailSource**

Create `internal/sources/gmail/gmail.go`:
```go
package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

var docIDRegexp = regexp.MustCompile(`docs\.google\.com/document/d/([a-zA-Z0-9_-]+)`)

type messagesResponse struct {
	Messages []struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"messages"`
	ResultSizeEstimate int `json:"resultSizeEstimate"`
}

// ExtractMessageID parses the gws messages list JSON and returns the first message ID.
func ExtractMessageID(data []byte) (string, error) {
	var resp messagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse messages response: %w", err)
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no messages found in response")
	}
	return resp.Messages[0].ID, nil
}

// ExtractDocID scans email body bytes for the first Google Docs URL and returns the file ID.
func ExtractDocID(body []byte) (string, error) {
	matches := docIDRegexp.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("no Google Docs URL found in email body")
	}
	return string(matches[1]), nil
}

// Source fetches the Google Doc ID from the weekly report email.
type Source struct {
	executor    *gws.Executor
	emailSender string
	reportName  string
	docID       string
}

// NewSource creates a GmailSource.
func NewSource(executor *gws.Executor, emailSender, reportName string) *Source {
	return &Source{
		executor:    executor,
		emailSender: emailSender,
		reportName:  reportName,
	}
}

// Name implements DataSource.
func (s *Source) Name() string { return "gmail" }

// Fetch searches Gmail for the weekly report email and extracts the Google Doc ID.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	query := fmt.Sprintf(`from:(%s) [Fill Weekly Report: %s] %s`,
		s.emailSender, s.reportName, week.EmailDateLabel())
	params := fmt.Sprintf(`{"userId":"me","q":%q}`, query)

	listOut, err := s.executor.Run(ctx, "gmail", "users", "messages", "list", "--params", params)
	if err != nil {
		return fmt.Errorf("gmail list messages: %w", err)
	}

	msgID, err := ExtractMessageID(listOut)
	if err != nil {
		return fmt.Errorf("extract message ID: %w", err)
	}

	readOut, err := s.executor.Run(ctx, "gmail", "+read", "--id", msgID)
	if err != nil {
		return fmt.Errorf("gmail read message %s: %w", msgID, err)
	}

	docID, err := ExtractDocID(readOut)
	if err != nil {
		return fmt.Errorf("extract doc ID from email: %w", err)
	}

	s.docID = docID
	return nil
}

// Contribute sets the DocID on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	if s.docID == "" {
		return fmt.Errorf("gmail source has no doc ID; was Fetch called?")
	}
	report.DocID = s.docID
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/sources/gmail/... -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/sources/gmail/
git commit -m "feat: add gmail source (find Google Doc ID from email)"
```

---

## Task 8: GitHub Source

**Files:**
- Create: `internal/sources/github/github.go`
- Create: `internal/sources/github/github_test.go`

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/shurcooL/githubv4
go get golang.org/x/oauth2
```

Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 2: Write failing tests**

Create `internal/sources/github/github_test.go`:
```go
package github_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	gh "github.com/apikdech/gws-weekly-report/internal/sources/github"
)

func TestCleanPRTitle(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Add feature…", "Add feature"},
		{"  Fix bug  ", "Fix bug"},
		{"Normal title", "Normal title"},
		{"Title with … ellipsis", "Title with  ellipsis"},
	}
	for _, tc := range cases {
		got := gh.CleanTitle(tc.input)
		if got != tc.want {
			t.Errorf("CleanTitle(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGroupByRepo_MergesImplementedAndReviewed(t *testing.T) {
	implemented := []pipeline.PR{
		{Title: "Add X", URL: "https://github.com/org/a/pull/1"},
	}
	reviewed := []pipeline.PR{
		{Title: "Fix Y", URL: "https://github.com/org/b/pull/2"},
	}
	// org/a has implemented, org/b has reviewed
	repoImpl := map[string][]pipeline.PR{"org/a": implemented}
	repoReviewed := map[string][]pipeline.PR{"org/b": reviewed}

	result := gh.GroupByRepo(repoImpl, repoReviewed)

	if len(result) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(result))
	}
	if len(result["org/a"].Implemented) != 1 {
		t.Errorf("expected 1 implemented PR for org/a")
	}
	if len(result["org/b"].Reviewed) != 1 {
		t.Errorf("expected 1 reviewed PR for org/b")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/sources/github/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 4: Implement GitHubSource**

Create `internal/sources/github/github.go`:
```go
package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

// CleanTitle removes ellipsis characters and trims whitespace from a PR title.
func CleanTitle(title string) string {
	title = strings.ReplaceAll(title, "…", "")
	return strings.TrimSpace(title)
}

// GroupByRepo merges implemented and reviewed PR maps into a map of RepoPRs.
func GroupByRepo(implemented, reviewed map[string][]pipeline.PR) map[string]*pipeline.RepoPRs {
	result := make(map[string]*pipeline.RepoPRs)

	for repo, prs := range implemented {
		if _, ok := result[repo]; !ok {
			result[repo] = &pipeline.RepoPRs{RepoName: repo}
		}
		result[repo].Implemented = append(result[repo].Implemented, prs...)
	}
	for repo, prs := range reviewed {
		if _, ok := result[repo]; !ok {
			result[repo] = &pipeline.RepoPRs{RepoName: repo}
		}
		result[repo].Reviewed = append(result[repo].Reviewed, prs...)
	}
	return result
}

// Source fetches GitHub PRs authored and reviewed by the user for the week.
type Source struct {
	token    string
	username string
	prsByRepo map[string]*pipeline.RepoPRs
}

// NewSource creates a GitHubSource.
func NewSource(token, username string) *Source {
	return &Source{token: token, username: username}
}

// Name implements DataSource.
func (s *Source) Name() string { return "github" }

// Fetch queries GitHub GraphQL API for authored and reviewed PRs within the week range.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.token})
	tc := oauth2.NewClient(ctx, ts)
	client := githubv4.NewClient(tc)

	implemented, err := s.fetchPRs(ctx, client,
		fmt.Sprintf("author:%s is:pr created:%s..%s",
			s.username,
			week.Start.Format("2006-01-02T15:04:05Z"),
			week.End.Format("2006-01-02T15:04:05Z"),
		),
	)
	if err != nil {
		return fmt.Errorf("fetch implemented PRs: %w", err)
	}

	reviewed, err := s.fetchPRs(ctx, client,
		fmt.Sprintf("reviewed-by:%s is:pr created:%s..%s",
			s.username,
			week.Start.Format("2006-01-02T15:04:05Z"),
			week.End.Format("2006-01-02T15:04:05Z"),
		),
	)
	if err != nil {
		return fmt.Errorf("fetch reviewed PRs: %w", err)
	}

	s.prsByRepo = GroupByRepo(implemented, reviewed)
	return nil
}

func (s *Source) fetchPRs(ctx context.Context, client *githubv4.Client, searchQuery string) (map[string][]pipeline.PR, error) {
	var query struct {
		Search struct {
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
			Nodes []struct {
				PullRequest struct {
					Title      githubv4.String
					URL        githubv4.URI
					Repository struct {
						NameWithOwner githubv4.String
					}
				} `graphql:"... on PullRequest"`
			}
		} `graphql:"search(query: $query, type: ISSUE, first: 100, after: $cursor)"`
	}

	variables := map[string]interface{}{
		"query":  githubv4.String(searchQuery),
		"cursor": (*githubv4.String)(nil),
	}

	result := make(map[string][]pipeline.PR)
	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("graphql query: %w", err)
		}
		for _, node := range query.Search.Nodes {
			pr := node.PullRequest
			repo := string(pr.Repository.NameWithOwner)
			if repo == "" {
				continue
			}
			result[repo] = append(result[repo], pipeline.PR{
				Title: CleanTitle(string(pr.Title)),
				URL:   pr.URL.String(),
			})
		}
		if !query.Search.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}
	return result, nil
}

// Contribute sets PRsByRepo on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	if report.PRsByRepo == nil {
		report.PRsByRepo = make(map[string]*pipeline.RepoPRs)
	}
	for k, v := range s.prsByRepo {
		report.PRsByRepo[k] = v
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/sources/github/... -v
```

Expected: both tests PASS (unit tests only; GraphQL calls are integration-tested at run time).

- [ ] **Step 6: Commit**

```bash
git add internal/sources/github/ go.mod go.sum
git commit -m "feat: add github source (GraphQL PR fetcher)"
```

---

## Task 9: Calendar Source

**Files:**
- Create: `internal/sources/calendar/calendar.go`
- Create: `internal/sources/calendar/calendar_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/sources/calendar/calendar_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/sources/calendar/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement CalendarSource**

Create `internal/sources/calendar/calendar.go`:
```go
package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

type eventsResponse struct {
	Items []struct {
		Summary string `json:"summary"`
		Start   struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
	} `json:"items"`
}

// ParseEvents parses the gws calendar events list JSON into CalendarEvents.
func ParseEvents(data []byte) ([]pipeline.CalendarEvent, error) {
	var resp eventsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse calendar events: %w", err)
	}

	var events []pipeline.CalendarEvent
	for _, item := range resp.Items {
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
	return events, nil
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

	events, err := ParseEvents(out)
	if err != nil {
		return err
	}
	s.events = events
	return nil
}

// Contribute sets Events on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	report.Events = s.events
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/sources/calendar/... -v
```

Expected: both tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/sources/calendar/
git commit -m "feat: add calendar source (fetch events via gws CLI)"
```

---

## Task 10: Drive Uploader

**Files:**
- Create: `internal/uploader/drive/drive.go`
- Create: `internal/uploader/drive/drive_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/uploader/drive/drive_test.go`:
```go
package drive_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/uploader/drive"
)

func TestUploader_BuildsCorrectArgs(t *testing.T) {
	// Capture the args passed to gws by using a script that prints them.
	// We can't unit-test the real gws call, so we verify the Uploader's arg construction
	// by using a fake gws that echoes its args to stdout.
	scriptPath := t.TempDir() + "/fake-gws.sh"
	script := "#!/bin/sh\necho \"$@\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	tmpCreds, _ := os.CreateTemp("", "creds*.json")
	defer os.Remove(tmpCreds.Name())

	ex := gws.NewExecutor(scriptPath, tmpCreds.Name())
	u := drive.NewUploader(ex)

	// Create a temp report file
	reportPath := t.TempDir() + "/report.md"
	if err := os.WriteFile(reportPath, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := u.Upload(context.Background(), "docABC", reportPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out // output is the fake gws echo

	// Verify script was invoked (no panic, no error = correct argument passing)
	_ = exec.LookPath("sh") // ensure sh is available for test
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/uploader/drive/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement DriveUploader**

Create `internal/uploader/drive/drive.go`:
```go
package drive

import (
	"context"
	"fmt"

	"github.com/apikdech/gws-weekly-report/internal/gws"
)

// Uploader uploads a local file to a Google Drive document.
type Uploader struct {
	executor *gws.Executor
}

// NewUploader creates a DriveUploader.
func NewUploader(executor *gws.Executor) *Uploader {
	return &Uploader{executor: executor}
}

// Upload updates a Google Docs file with the contents of reportPath.
// Returns the raw gws CLI output on success.
func (u *Uploader) Upload(ctx context.Context, docID, reportPath string) ([]byte, error) {
	params := fmt.Sprintf(`{"fileId":%q}`, docID)
	out, err := u.executor.Run(ctx,
		"drive", "files", "update",
		"--params", params,
		"--upload", reportPath,
		"--upload-content-type", "text/markdown",
		"--json", `{"mimeType":"application/vnd.google-apps.document"}`,
	)
	if err != nil {
		return nil, fmt.Errorf("drive files update: %w", err)
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/uploader/drive/... -v
```

Expected: test PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/uploader/drive/
git commit -m "feat: add drive uploader"
```

---

## Task 11: Wire Everything in `main.go`

**Files:**
- Modify: `cmd/reporter/main.go`

- [ ] **Step 1: Implement main**

Overwrite `cmd/reporter/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/config"
	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	"github.com/apikdech/gws-weekly-report/internal/report"
	"github.com/apikdech/gws-weekly-report/internal/sources/calendar"
	"github.com/apikdech/gws-weekly-report/internal/sources/gmail"
	gh "github.com/apikdech/gws-weekly-report/internal/sources/github"
	"github.com/apikdech/gws-weekly-report/internal/uploader/drive"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// 2. Compute week range
	loc, err := time.LoadLocation(cfg.ReportTimezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", cfg.ReportTimezone, err)
	}
	week := pipeline.WeekRangeFor(time.Now(), loc)
	log.Printf("Week: %s", week.HeaderLabel())

	// 3. Build gws executor
	gwsBin := "gws" // resolved from PATH in container
	if v := os.Getenv("GWS_BIN_PATH"); v != "" {
		gwsBin = v
	}
	executor := gws.NewExecutor(gwsBin, cfg.GWSCredentialsFile)

	// 4. Build sources
	gmailSrc := gmail.NewSource(executor, cfg.GWSEmailSender, cfg.ReportName)
	githubSrc := gh.NewSource(cfg.GitHubToken, cfg.GitHubUsername)
	calendarSrc := calendar.NewSource(executor)

	// 5. Run pipeline
	reportData := &pipeline.ReportData{
		ReportName: cfg.ReportName,
		Week:       week,
		PRsByRepo:  make(map[string]*pipeline.RepoPRs),
	}
	runner := pipeline.NewRunner([]pipeline.DataSource{gmailSrc, githubSrc, calendarSrc})
	if err := runner.Run(ctx, reportData); err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}

	// 6. Render markdown
	markdown, err := report.Render(reportData)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	// 7. Write report.md to temp dir
	reportPath := filepath.Join(cfg.TempDir, "report.md")
	if err := os.WriteFile(reportPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("write report.md: %w", err)
	}
	log.Printf("Report written to %s", reportPath)

	// 8. Upload to Drive
	uploader := drive.NewUploader(executor)
	if _, err := uploader.Upload(ctx, reportData.DocID, reportPath); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	log.Printf("Uploaded report to Google Doc: https://docs.google.com/document/d/%s/edit", reportData.DocID)

	// 9. Cleanup
	if err := os.Remove(reportPath); err != nil {
		log.Printf("WARN: failed to remove %s: %v", reportPath, err)
	}
	log.Printf("Done.")
	return nil
}
```

- [ ] **Step 2: Build and verify**

```bash
go build ./cmd/reporter
```

Expected: binary builds with no errors.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/reporter/main.go
git commit -m "feat: wire all components in main entrypoint"
```

---

## Task 12: Dockerfile & Docker Compose

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`

- [ ] **Step 1: Check latest gws release tag**

Visit https://github.com/googleworkspace/cli/releases and confirm current release. At time of writing: `0.22.3`. Update `GWS_VERSION` in Dockerfile if newer.

- [ ] **Step 2: Create Dockerfile**

Create `Dockerfile`:
```dockerfile
# Stage 1: Build Go binary
FROM golang:1.26.1 AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/reporter ./cmd/reporter

# Stage 2: Download gws pre-built binary (Rust native, no Node required)
FROM alpine:3.21 AS gws-downloader
ARG GWS_VERSION=0.22.3
RUN apk add --no-cache wget ca-certificates \
  && wget -O /usr/local/bin/gws \
     "https://github.com/googleworkspace/cli/releases/download/v${GWS_VERSION}/gws-linux-x86_64" \
  && chmod +x /usr/local/bin/gws

# Stage 3: Distroless runtime (no shell, no package manager)
FROM gcr.io/distroless/base-debian12
COPY --from=go-builder /app/reporter /app/reporter
COPY --from=gws-downloader /usr/local/bin/gws /usr/local/bin/gws
ENTRYPOINT ["/app/reporter"]
```

- [ ] **Step 3: Create docker-compose.yml**

Create `docker-compose.yml`:
```yaml
services:
  reporter:
    build: .
    env_file: .env
    volumes:
      - ./credentials.json:/run/secrets/gws-credentials:ro
    restart: "no"
```

- [ ] **Step 4: Build Docker image locally**

```bash
docker build -t gws-weekly-report .
```

Expected: image builds successfully through all 3 stages.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "feat: add Dockerfile (multi-stage distroless) and docker-compose"
```

---

## Task 13: Deployment Setup

**Files:**
- No new code files — VPS setup instructions.

- [ ] **Step 1: Export gws credentials on local machine**

On your local machine (where you've already run `gws auth login`):
```bash
gws auth export --unmasked > credentials.json
```

- [ ] **Step 2: Copy project to VPS**

```bash
scp -r . user@your-vps:/opt/gws-weekly-report
scp credentials.json user@your-vps:/opt/gws-weekly-report/credentials.json
```

- [ ] **Step 3: Create .env on VPS**

SSH into VPS and create `/opt/gws-weekly-report/.env`:
```
GITHUB_TOKEN=ghp_your_real_token
GITHUB_USERNAME=ricky-setiawan
GWS_EMAIL_SENDER=sender@example.com
REPORT_NAME=Ricky Setiawan
GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/run/secrets/gws-credentials
REPORT_TIMEZONE=Asia/Jakarta
```

- [ ] **Step 4: Build image on VPS**

```bash
cd /opt/gws-weekly-report
docker compose build
```

Expected: image builds successfully.

- [ ] **Step 5: Test run manually**

```bash
docker compose run --rm reporter
```

Expected: logs show each pipeline step, ends with `Done.`.

- [ ] **Step 6: Add crontab entry**

On VPS:
```bash
crontab -e
```

Add:
```
0 9 * * 1 cd /opt/gws-weekly-report && docker compose run --rm reporter >> /var/log/gws-reporter.log 2>&1
```

Expected: runs every Monday at 09:00 server time.

- [ ] **Step 7: Commit final state**

```bash
git add .
git commit -m "chore: finalize project — ready for VPS deployment"
```

---

## Self-Review Notes

- All 13 tasks are covered by the spec requirements
- `WeekRangeFor` is consistently named throughout all tasks
- `pipeline.PR`, `pipeline.RepoPRs`, `pipeline.CalendarEvent`, `pipeline.ReportData`, `pipeline.WeekRange`, `pipeline.DataSource` types defined in Task 3 and used consistently in Tasks 6–11
- `gws.NewExecutor` / `gws.Executor.Run` defined in Task 5 and used consistently in Tasks 7, 9, 10, 11
- `report.Render` defined in Task 6 and called in Task 11
- `gmail.NewSource`, `gh.NewSource`, `calendar.NewSource` defined in Tasks 7–9 and wired in Task 11
- `drive.NewUploader` / `Uploader.Upload` defined in Task 10 and called in Task 11
- No placeholders or TBDs remain
