# GWS Weekly Report

A Go tool that automatically generates your weekly report and uploads it to Google Docs. Runs as a Docker container via crontab on a VPS.

## How it works

Each run executes a pipeline:

1. Calculates the current Sunday–Saturday week range
2. Searches Gmail for the weekly report email from the agent and extracts the Google Doc ID
3. Fetches GitHub PRs you authored and reviewed during the week (via GraphQL)
4. Fetches your Google Calendar events for the week
5. Renders a markdown report from the collected data
6. Uploads the report to the Google Doc via the [`gws` CLI](https://github.com/googleworkspace/cli)
7. Cleans up the temporary file

## Report output

```markdown
# [Weekly Report: Your Name] 22 March 2026 - 28 March 2026

## **Issues**

## **Accomplishment**
### org/repo-name
#### Implemented PR
1. [Add feature X](https://github.com/org/repo-name/pull/1)

#### Reviewed PR
1. [Fix bug Y](https://github.com/org/repo-name/pull/2)

## **Meetings/Events/Training/Conferences**
- Sprint Planning (23 March 2026)

## **Key Metrics / OMTM**

## **Next Actions**
1. Continue implement admin dashboard features

## **Technology, Business, Communication, Leadership, Management & Marketing**

## Out of Office
```

Sections that require manual input (Key Metrics, Next Actions, Technology/Business, Out of Office) are left empty for you to fill in Google Docs after the upload.

## Prerequisites

- A VPS with Docker and Docker Compose installed
- A GitHub personal access token with `repo` and `read:user` scopes
- A Google Workspace account with the [`gws` CLI](https://github.com/googleworkspace/cli) authenticated locally

## Setup

### 1. Authenticate gws locally

Install and authenticate the `gws` CLI on your **local machine**:

```bash
npm install -g @googleworkspace/cli
gws auth setup   # one-time: creates GCP project, enables APIs, logs you in
```

### 2. Export credentials for headless use

```bash
gws auth export --unmasked > credentials.json
```

This file will be volume-mounted into the Docker container. Keep it secret — do not commit it.

### 3. Clone and configure

```bash
git clone https://github.com/apikdech/gws-weekly-report
cd gws-weekly-report
cp .env.example .env
```

Edit `.env` with your values:

```bash
# Required
GITHUB_TOKEN=ghp_your_token_here
GITHUB_USERNAME=your-github-username
GWS_EMAIL_SENDER=sender@example.com       # who sends the "fill weekly report" email
REPORT_NAME=Your Full Name              # must match the name in the email subject
GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/run/secrets/gws-credentials

# Optional
REPORT_TIMEZONE=Asia/Jakarta            # default: UTC
TEMP_DIR=/tmp
```

### 4. Copy to VPS

```bash
scp -r . user@your-vps:/opt/gws-weekly-report
scp credentials.json user@your-vps:/opt/gws-weekly-report/credentials.json
```

### 5. Build and test

```bash
cd /opt/gws-weekly-report
docker compose build
docker compose run --rm reporter
```

A successful run logs each step and ends with `Done.`

### 6. Schedule with crontab

Add this to the VPS crontab (`crontab -e`) to run every Monday at 09:00:

```
0 9 * * 1 cd /opt/gws-weekly-report && docker compose run --rm reporter >> /var/log/gws-reporter.log 2>&1
```

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `GITHUB_TOKEN` | Yes | — | GitHub personal access token |
| `GITHUB_USERNAME` | Yes | — | Your GitHub username |
| `GWS_EMAIL_SENDER` | Yes | — | Email address that sends the weekly report prompt |
| `REPORT_NAME` | Yes | — | Your full name as it appears in the email subject |
| `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` | Yes | — | Path to the exported gws credentials JSON inside the container |
| `REPORT_TIMEZONE` | No | `UTC` | Timezone for week range calculation (e.g. `Asia/Jakarta`) |
| `TEMP_DIR` | No | `/tmp` | Directory for the temporary `report.md` file |
| `GWS_BIN_PATH` | No | `gws` | Path to the gws binary (resolved from `PATH` by default) |

## Docker

The image uses a 3-stage build:

| Stage | Base | Purpose |
|---|---|---|
| `go-builder` | `golang:1.26.1` | Compiles the Go binary (static, CGO disabled) |
| `gws-downloader` | `alpine:3.21` | Downloads the gws pre-built binary |
| runtime | `gcr.io/distroless/base-debian12` | Minimal runtime (~20 MiB image) |

The distroless runtime has no shell or package manager, reducing the attack surface.

## Development

```bash
# Run tests
go test ./...

# Build binary
go build ./cmd/reporter

# Run a single test package
go test ./internal/sources/gmail/... -v
```

## Project structure

```
cmd/reporter/main.go              # entrypoint
internal/
  config/                         # environment variable loading
  pipeline/                       # DataSource interface, WeekRange, Runner
  gws/                            # gws CLI executor wrapper
  sources/
    gmail/                        # Gmail source (finds Google Doc ID)
    github/                       # GitHub GraphQL source (authored + reviewed PRs)
    calendar/                     # Calendar source (weekly events)
  uploader/drive/                 # Google Drive uploader
  report/                         # markdown template renderer
Dockerfile
docker-compose.yml
```
