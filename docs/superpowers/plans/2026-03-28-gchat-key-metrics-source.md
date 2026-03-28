# Google Chat Key Metrics Source Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Google Chat spaces source that fetches the latest bot message from a configured space and populates the Key Metrics / OMTM section of the weekly report.

**Architecture:** A new `gchat` source package follows the existing `DataSource` interface pattern (Fetch / Contribute). Two new env vars (`GWS_CHAT_SPACES_ID`, `GWS_CHAT_SENDER_NAME`) are added to `Config`. `pipeline.ReportData` gains a `KeyMetrics string` field. The render template emits the raw text when the field is non-empty.

**Tech Stack:** Go 1.26.1, `encoding/json`, `time`, existing `gws.Executor` CLI wrapper, `text/template`.

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Modify | `internal/config/config.go` | Add `GWSChatSpacesID` and `GWSChatSenderName` fields + load + validate |
| Modify | `.env.example` | Document the two new env vars |
| Modify | `internal/pipeline/types.go` | Add `KeyMetrics string` to `ReportData` |
| Create | `internal/sources/gchat/gchat.go` | GChat `DataSource` implementation |
| Create | `internal/sources/gchat/gchat_test.go` | Unit tests for JSON parsing and filtering |
| Modify | `internal/report/render.go` | Emit `KeyMetrics` in the template section |
| Modify | `cmd/reporter/main.go` | Instantiate and wire the new source |

---

### Task 1: Add `KeyMetrics` to `ReportData`

**Files:**
- Modify: `internal/pipeline/types.go`
- Modify: `internal/pipeline/types_test.go`

- [ ] **Step 1: Write the failing test**

Open `internal/pipeline/types_test.go` and add:

```go
func TestReportDataHasKeyMetrics(t *testing.T) {
    rd := pipeline.ReportData{KeyMetrics: "some metrics text"}
    if rd.KeyMetrics != "some metrics text" {
        t.Fatalf("expected KeyMetrics to be set, got %q", rd.KeyMetrics)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ricky-setiawan/Other/ok/gws-weekly-report
go test ./internal/pipeline/... -run TestReportDataHasKeyMetrics -v
```

Expected: compile error — `unknown field KeyMetrics`

- [ ] **Step 3: Add the field to ReportData**

In `internal/pipeline/types.go`, add `KeyMetrics` after `OutOfOfficeDates`:

```go
// ReportData holds all collected data used to render the weekly report.
type ReportData struct {
	ReportName       string
	Week             WeekRange
	DocID            string
	PRsByRepo        map[string]*RepoPRs // keyed by repo NameWithOwner
	Events           []CalendarEvent
	OutOfOfficeDates []string // sorted unique, formatted as "2 January 2006"
	KeyMetrics       string   // raw text from Google Chat spaces bot message
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/pipeline/... -run TestReportDataHasKeyMetrics -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pipeline/types.go internal/pipeline/types_test.go
git commit -m "feat(pipeline): add KeyMetrics field to ReportData"
```

---

### Task 2: Add config fields for GChat

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `.env.example`

- [ ] **Step 1: Write the failing test**

Open `internal/config/config_test.go`. Find (or add) a test that checks config loading. Add a test case for the new fields:

```go
func TestConfigLoadsGChatFields(t *testing.T) {
    t.Setenv("GITHUB_TOKEN", "tok")
    t.Setenv("GITHUB_USERNAME", "user")
    t.Setenv("GWS_EMAIL_SENDER", "sender@example.com")
    t.Setenv("REPORT_NAME", "Test User")
    t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
    t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
    t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")

    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.GWSChatSpacesID != "AAQAE4zqbX4" {
        t.Errorf("GWSChatSpacesID: got %q, want %q", cfg.GWSChatSpacesID, "AAQAE4zqbX4")
    }
    if cfg.GWSChatSenderName != "users/102650500894334129637" {
        t.Errorf("GWSChatSenderName: got %q, want %q", cfg.GWSChatSenderName, "users/102650500894334129637")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/... -run TestConfigLoadsGChatFields -v
```

Expected: compile error — `unknown field GWSChatSpacesID`

- [ ] **Step 3: Add fields to Config struct and Load()**

In `internal/config/config.go`, update the `Config` struct:

```go
type Config struct {
	GitHubToken        string
	GitHubUsername     string
	GWSEmailSender     string
	ReportName         string
	GWSCredentialsFile string
	ReportTimezone     string
	TempDir            string
	GWSChatSpacesID    string // Google Chat space ID, e.g. "AAQAE4zqbX4"
	GWSChatSenderName  string // sender.name to filter, e.g. "users/102650500894334129637"
}
```

In `Load()`, populate from env:

```go
cfg := &Config{
    GitHubToken:        os.Getenv("GITHUB_TOKEN"),
    GitHubUsername:     os.Getenv("GITHUB_USERNAME"),
    GWSEmailSender:     os.Getenv("GWS_EMAIL_SENDER"),
    ReportName:         os.Getenv("REPORT_NAME"),
    GWSCredentialsFile: os.Getenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE"),
    ReportTimezone:     os.Getenv("REPORT_TIMEZONE"),
    TempDir:            os.Getenv("TEMP_DIR"),
    GWSChatSpacesID:    os.Getenv("GWS_CHAT_SPACES_ID"),
    GWSChatSenderName:  os.Getenv("GWS_CHAT_SENDER_NAME"),
}
```

Add them to the `required` slice:

```go
required := []requiredVar{
    {"GITHUB_TOKEN", cfg.GitHubToken},
    {"GITHUB_USERNAME", cfg.GitHubUsername},
    {"GWS_EMAIL_SENDER", cfg.GWSEmailSender},
    {"REPORT_NAME", cfg.ReportName},
    {"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", cfg.GWSCredentialsFile},
    {"GWS_CHAT_SPACES_ID", cfg.GWSChatSpacesID},
    {"GWS_CHAT_SENDER_NAME", cfg.GWSChatSenderName},
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/config/... -v
```

Expected: all PASS (existing tests still pass; new test passes)

- [ ] **Step 5: Update `.env.example`**

Add to `.env.example`:

```
# Google Chat (for Key Metrics / OMTM section)
GWS_CHAT_SPACES_ID=AAQAE4zqbX4
GWS_CHAT_SENDER_NAME=users/102650500894334129637
```

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go .env.example
git commit -m "feat(config): add GWSChatSpacesID and GWSChatSenderName env vars"
```

---

### Task 3: Implement the `gchat` source

**Files:**
- Create: `internal/sources/gchat/gchat_test.go`
- Create: `internal/sources/gchat/gchat.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/sources/gchat/gchat_test.go`:

```go
package gchat_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/sources/gchat"
)

var sampleResponse = []byte(`{
  "messages": [
    {
      "name": "spaces/AAQAE4zqbX4/messages/aaa",
      "createTime": "2026-03-22T08:00:00.000000Z",
      "sender": {"name": "users/other", "type": "BOT"},
      "text": "old message"
    },
    {
      "name": "spaces/AAQAE4zqbX4/messages/bbb",
      "createTime": "2026-03-27T08:43:10.336799Z",
      "sender": {"name": "users/102650500894334129637", "type": "BOT"},
      "text": "latest metrics text"
    },
    {
      "name": "spaces/AAQAE4zqbX4/messages/ccc",
      "createTime": "2026-03-25T10:00:00.000000Z",
      "sender": {"name": "users/102650500894334129637", "type": "BOT"},
      "text": "earlier metrics text"
    }
  ]
}`)

func TestPickLatestBySender_Found(t *testing.T) {
	text, err := gchat.PickLatestBySender(sampleResponse, "users/102650500894334129637")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "latest metrics text" {
		t.Errorf("got %q, want %q", text, "latest metrics text")
	}
}

func TestPickLatestBySender_NoMatch(t *testing.T) {
	text, err := gchat.PickLatestBySender(sampleResponse, "users/nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string for no match, got %q", text)
	}
}

func TestPickLatestBySender_EmptyMessages(t *testing.T) {
	data := []byte(`{"messages":[]}`)
	text, err := gchat.PickLatestBySender(data, "users/102650500894334129637")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestPickLatestBySender_InvalidJSON(t *testing.T) {
	_, err := gchat.PickLatestBySender([]byte(`not json`), "users/x")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/sources/gchat/... -v
```

Expected: compile error — package `gchat` does not exist yet

- [ ] **Step 3: Implement `gchat.go`**

Create `internal/sources/gchat/gchat.go`:

```go
package gchat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

type chatMessage struct {
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
	Sender     struct {
		Name string `json:"name"`
	} `json:"sender"`
	Text string `json:"text"`
}

type messagesResponse struct {
	Messages []chatMessage `json:"messages"`
}

// PickLatestBySender parses a gws chat spaces messages list JSON response,
// filters by senderName, and returns the text of the message with the latest
// createTime. Returns an empty string (no error) when no message matches.
func PickLatestBySender(data []byte, senderName string) (string, error) {
	var resp messagesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse chat messages response: %w", err)
	}

	var latestTime time.Time
	var latestText string
	for _, msg := range resp.Messages {
		if msg.Sender.Name != senderName {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, msg.CreateTime)
		if err != nil {
			// Fall back to RFC3339 without nanoseconds
			t, err = time.Parse(time.RFC3339, msg.CreateTime)
			if err != nil {
				continue
			}
		}
		if t.After(latestTime) {
			latestTime = t
			latestText = msg.Text
		}
	}
	return latestText, nil
}

// Source fetches the Key Metrics / OMTM text from a Google Chat space.
type Source struct {
	executor   *gws.Executor
	spacesID   string
	senderName string
	keyMetrics string
}

// NewSource creates a GChat Source.
func NewSource(executor *gws.Executor, spacesID, senderName string) *Source {
	return &Source{
		executor:   executor,
		spacesID:   spacesID,
		senderName: senderName,
	}
}

// Name implements DataSource.
func (s *Source) Name() string { return "gchat" }

// Fetch lists messages from the configured space starting from week.Start and
// picks the latest message matching the configured sender.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	parent := "spaces/" + s.spacesID
	filter := fmt.Sprintf(`createTime > "%s"`, week.Start.UTC().Format("2006-01-02T15:04:05Z"))
	params := fmt.Sprintf(`{"parent":%q,"filter":%q}`, parent, filter)

	out, err := s.executor.Run(ctx, "chat", "spaces", "messages", "list", "--params", params)
	if err != nil {
		return fmt.Errorf("gchat messages list: %w", err)
	}

	text, err := PickLatestBySender(out, s.senderName)
	if err != nil {
		return err
	}
	s.keyMetrics = text
	return nil
}

// Contribute sets KeyMetrics on the report. A missing message is not an error.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	report.KeyMetrics = s.keyMetrics
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/sources/gchat/... -v
```

Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/sources/gchat/gchat.go internal/sources/gchat/gchat_test.go
git commit -m "feat(sources/gchat): add Google Chat source for Key Metrics / OMTM"
```

---

### Task 4: Update the report template

**Files:**
- Modify: `internal/report/render.go`

- [ ] **Step 1: Update `templateData` struct and template**

In `internal/report/render.go`, add `KeyMetrics` to `templateData`:

```go
type templateData struct {
	ReportName       string
	Week             pipeline.WeekRange
	SortedRepos      []*pipeline.RepoPRs
	Events           []pipeline.CalendarEvent
	OutOfOfficeBlock string
	KeyMetrics       string
}
```

Update the `reportTemplate` constant — replace the Key Metrics section:

```
## **Key Metrics / OMTM**
{{ if .KeyMetrics }}
{{ .KeyMetrics }}
{{ end }}
```

Update the `td` struct literal in `Render()` to pass the field:

```go
td := templateData{
    ReportName:       data.ReportName,
    Week:             data.Week,
    SortedRepos:      repos,
    Events:           data.Events,
    OutOfOfficeBlock: oooBlock,
    KeyMetrics:       data.KeyMetrics,
}
```

- [ ] **Step 2: Run all tests**

```bash
go test ./... -v
```

Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/report/render.go
git commit -m "feat(report): render KeyMetrics in Key Metrics / OMTM section"
```

---

### Task 5: Wire the gchat source in `main.go`

**Files:**
- Modify: `cmd/reporter/main.go`

- [ ] **Step 1: Add import and source wiring**

In `cmd/reporter/main.go`, add the import:

```go
gchat "github.com/apikdech/gws-weekly-report/internal/sources/gchat"
```

After `calendarSrc := calendar.NewSource(executor)`, add:

```go
gchatSrc := gchat.NewSource(executor, cfg.GWSChatSpacesID, cfg.GWSChatSenderName)
```

Add `gchatSrc` to the runner sources slice:

```go
runner := pipeline.NewRunner([]pipeline.DataSource{gmailSrc, githubSrc, calendarSrc, gchatSrc})
```

- [ ] **Step 2: Build to verify it compiles**

```bash
go build ./cmd/reporter/...
```

Expected: exits 0 with no output

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/reporter/main.go
git commit -m "feat(main): wire gchat source into pipeline"
```
