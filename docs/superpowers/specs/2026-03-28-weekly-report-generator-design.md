# Design: GWS Weekly Report Generator

**Date:** 2026-03-28  
**Status:** Approved

---

## Overview

A Go binary (`gws-weekly-report`) that runs as a Docker container triggered by crontab on a VPS. Each week it automatically:

1. Finds the weekly report Google Doc via Gmail
2. Fetches GitHub PRs (authored and reviewed) for the week
3. Fetches calendar events for the week
4. Renders a markdown report from a fixed template
5. Uploads the report to Google Docs via the `gws` CLI
6. Cleans up temp files

---

## Architecture

**Approach:** Single binary with a `DataSource` interface (plugin architecture). Each data source fetches its data and contributes its section to a shared `ReportData` struct. The pipeline runner executes all sources in order.

### `DataSource` interface

```go
type DataSource interface {
    Name() string
    Fetch(ctx context.Context, week WeekRange) error
    Contribute(report *ReportData) error
}
```

### Package layout

```
cmd/reporter/main.go              # entrypoint, wires pipeline
internal/config/                  # env-based config, fails fast on missing vars
internal/pipeline/                # pipeline runner + DataSource interface + WeekRange
internal/sources/gmail/           # finds Google Doc ID via gws CLI (DataSource)
internal/sources/github/          # GraphQL client for authored + reviewed PRs (DataSource)
internal/sources/calendar/        # fetches calendar events via gws CLI (DataSource)
internal/uploader/drive/          # uploads report.md to Google Drive (post-render step, not a DataSource)
internal/report/                  # ReportData struct + markdown template renderer
internal/gws/                     # shared gws CLI executor wrapper (runs gws commands, parses JSON output)
```

Note: `DriveUploader` is a distinct post-render step, not a `DataSource`. The pipeline runs: fetch sources → render report → upload → cleanup.

---

## Configuration

All config from environment variables. Fails fast at startup if required vars are missing.

### Required

| Variable | Description |
|---|---|
| `GITHUB_TOKEN` | GitHub personal access token |
| `GITHUB_USERNAME` | GitHub username (e.g. `ricky-setiawan`) |
| `GWS_EMAIL_SENDER` | Email sender filter (e.g. `agent@gdplabs.id`) |
| `REPORT_NAME` | Full name for report header and email search (e.g. `Ricky Setiawan`) |
| `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` | Path to exported gws credentials JSON |

### Optional (with defaults)

| Variable | Default | Description |
|---|---|---|
| `REPORT_TIMEZONE` | `UTC` | Timezone for week range calculation |
| `TEMP_DIR` | `/tmp` | Directory for temporary `report.md` |

---

## Report Template

```markdown
# [Weekly Report: <NAME>] <START_DATE> - <END_DATE>

## **Issues**

## **Accomplishment**
### <REPO_NAME>
#### Implemented PR
1. [<PR_TITLE>](<PR_LINK>)

#### Reviewed PR
1. [<PR_TITLE>](<PR_LINK>)

## **Meetings/Events/Training/Conferences**
- <EVENT_TITLE> (<EVENT_DATE>)

## **Key Metrics / OMTM**

## **Next Actions**
1. Continue implement admin dashboard features

## **Technology, Business, Communication, Leadership, Management & Marketing**

## Out of Office
```

Sections that cannot be auto-populated (`Key Metrics / OMTM`, `Technology/Business/...`, `Out of Office`) are left empty for the user to fill in Google Docs after upload.

---

## Pipeline Flow

### Step 1 — Compute week range
Calculate the most recent Sunday (inclusive start) and the following Saturday (inclusive end) relative to today's date in `REPORT_TIMEZONE`.

Example: run on 2026-03-28 (Saturday) → range is 2026-03-22 (Sunday) to 2026-03-28 (Saturday).

### Step 2 — Find the Google Doc ID (`GmailSource`)

Execute via `gws` CLI:
```
gws gmail users messages list --params '{
  "userId": "me",
  "q": "from:(agent@gdplabs.id) [Fill Weekly Report: Ricky Setiawan] 22 March 2026"
}'
```
Parse the first message ID from the JSON response. Then read the email body:
```
gws gmail +read --id <message_id>
```
Extract the first Google Docs URL via regex (`docs.google.com/document/d/([^/]+)`). Parse the doc file ID.

### Step 3 — Fetch GitHub PRs (`GitHubSource`)

Uses `github.com/shurcooL/githubv4` GraphQL client with `GITHUB_TOKEN`.

- **Implemented PRs:** search query `author:<username> is:pr created:<sunday>..<saturday>`
- **Reviewed PRs:** search query `reviewed-by:<username> is:pr created:<sunday>..<saturday>`

Results are grouped by repository (`NameWithOwner`). Each repo appears once in the `Accomplishment` section with both `Implemented PR` and `Reviewed PR` subsections.

PR titles have `…` stripped and are trimmed of whitespace.

Pagination handled via `PageInfo.HasNextPage` / `EndCursor`.

### Step 4 — Fetch Calendar Events (`CalendarSource`)

Execute via `gws` CLI:
```
gws calendar events list --params '{
  "calendarId": "primary",
  "timeMin": "<sunday>T00:00:00Z",
  "timeMax": "<saturday>T23:59:59Z",
  "singleEvents": true,
  "orderBy": "startTime"
}'
```
Parse JSON output to extract event summaries and start dates/times. Filter out all-day events with no meaningful title if desired.

Contributes to the `Meetings/Events/Training/Conferences` section.

### Step 5 — Render markdown (`internal/report`)

`ReportData` is populated by all sources. The renderer uses Go's `text/template` to produce `report.md` in `TEMP_DIR`.

### Step 6 — Upload to Drive (`DriveUploader`)

Execute via `gws` CLI:
```
gws drive files update \
  --params '{"fileId": "<doc_id>"}' \
  --upload "report.md" \
  --upload-content-type "text/markdown" \
  --json '{"mimeType": "application/vnd.google-apps.document"}'
```

### Step 7 — Cleanup

Delete `report.md` from `TEMP_DIR`. Log success with the Google Doc URL.

---

## GWS CLI Executor (`internal/gws`)

A thin wrapper around `os/exec` that:
- Runs `gws` commands with `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` injected into the environment
- Captures stdout as `[]byte` for JSON parsing
- Captures stderr for error logging
- Returns a typed error on non-zero exit codes

```go
type Executor struct {
    credentialsFile string
    gwsBinPath      string
}

func (e *Executor) Run(ctx context.Context, args ...string) ([]byte, error)
```

---

## Docker & Deployment

### Go version
`golang:1.26.1` (confirmed current stable at go.dev/dl)

### Dockerfile (multi-stage, 3 stages)

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.26.1 AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/reporter ./cmd/reporter

# Stage 2: Download gws pre-built binary (Rust, no Node required)
FROM alpine:3.21 AS gws-downloader
ARG GWS_VERSION=0.22.3
RUN wget -O /usr/local/bin/gws \
    "https://github.com/googleworkspace/cli/releases/download/v${GWS_VERSION}/gws-linux-x86_64" \
  && chmod +x /usr/local/bin/gws

# Stage 3: Distroless runtime
FROM gcr.io/distroless/base-debian12
COPY --from=go-builder /app/reporter /app/reporter
COPY --from=gws-downloader /usr/local/bin/gws /usr/local/bin/gws
ENTRYPOINT ["/app/reporter"]
```

### docker-compose.yml

```yaml
services:
  reporter:
    build: .
    env_file: .env
    volumes:
      - ./credentials.json:/run/secrets/gws-credentials:ro
    restart: "no"
```

### .env (on VPS, not committed)

```
GITHUB_TOKEN=ghp_...
GITHUB_USERNAME=ricky-setiawan
GWS_EMAIL_SENDER=agent@gdplabs.id
REPORT_NAME=Ricky Setiawan
GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/run/secrets/gws-credentials
REPORT_TIMEZONE=Asia/Jakarta
```

### GWS Credentials (CI/headless export flow)

On local machine (once):
```
gws auth export --unmasked > credentials.json
```
Copy `credentials.json` to VPS. Volume-mount into container as read-only. No keyring backend required.

### Crontab (VPS)

Runs every Monday at 09:00 server time:
```
0 9 * * 1 cd /opt/gws-weekly-report && docker compose run --rm reporter >> /var/log/gws-reporter.log 2>&1
```

---

## Error Handling

- Config validation fails fast at startup with a descriptive message
- Each pipeline step logs its name and outcome
- If any step fails, the run exits with a non-zero code (cron captures this in the log)
- `gws` CLI non-zero exit codes are surfaced as Go errors with stderr content included
- Cleanup runs only if upload succeeds (no partial cleanup on failure)

---

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/shurcooL/githubv4` | GitHub GraphQL v4 client |
| `golang.org/x/oauth2` | OAuth2 token source for GitHub client |
| Standard library only otherwise | `os/exec`, `text/template`, `encoding/json`, `time` |
