# GWS Weekly Report

A Go tool that automatically generates your weekly report and uploads it to Google Docs. Runs as a Docker container via crontab on a VPS.

## How it works

Each run executes a pipeline:

1. Calculates the current Sunday–Saturday week range
2. Searches Gmail for the weekly report email from the agent and extracts the Google Doc ID
3. Fetches GitHub PRs you authored and reviewed during the week (via GraphQL)
4. Fetches your Google Calendar events for the week
5. Fetches Google Chat messages for Key Metrics (optional)
6. Fetches Hacker News technology highlights via LLM (optional)
7. Renders a markdown report from the collected data
8. Uploads the report to the Google Doc via the [`gws` CLI](https://github.com/googleworkspace/cli)
9. Sends Discord webhook notifications if configured (optional)
10. Cleans up the temporary file

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
| `LLM_PROVIDER` | No | `gemini` | LLM provider for Technology Highlights: `gemini`, `openai`, `anthropic` |
| `LLM_API_KEY` | No | — | API key for the selected LLM provider |
| `LLM_MODEL` | No | `gemini-3-flash` | Model name (provider-specific, see LLM Configuration section) |
| `LLM_BASE_URL` | No | — | Base URL for OpenAI-compatible providers (optional, see LLM Configuration) |
| `REPORT_TIMEZONE` | No | `UTC` | Timezone for week range calculation (e.g. `Asia/Jakarta`) |
| `TEMP_DIR` | No | `/tmp` | Directory for the temporary `report.md` file |
| `GWS_BIN_PATH` | No | `gws` | Path to the gws binary (resolved from `PATH` by default) |
| `DISCORD_WEBHOOK_URL` | No | — | Discord webhook URL for notifications |
| `DISCORD_TIMEOUT` | No | `30` | HTTP timeout for Discord webhook in seconds |
| `DISCORD_RETRY_COUNT` | No | `1` | Retries on Discord webhook failure |

## LLM Configuration (for Technology Highlights)

The application supports multiple LLM providers via the [any-llm-go](https://github.com/mozilla-ai/any-llm-go) library. Configure via environment variables:

### Environment Variables

```bash
# Provider selection: gemini | openai | anthropic (default: gemini)
LLM_PROVIDER=gemini

# API key for the selected provider
LLM_API_KEY=your-api-key-here

# Model name (provider-specific)
# Gemini: gemini-3-flash, gemini-2.5-pro, gemini-2.0-flash, etc.
# OpenAI: gpt-4, gpt-3.5-turbo, gpt-4o, etc.
# Anthropic: claude-3-5-sonnet-20241022, claude-3-opus, etc.
LLM_MODEL=gemini-3-flash

# Base URL for OpenAI-compatible providers (optional)
# Only used with LLM_PROVIDER=openai
# Examples:
# - OpenAI default: https://api.openai.com/v1
# - OpenRouter: https://openrouter.ai/api/v1
# - Fireworks: https://api.fireworks.ai/inference/v1
# - Kimi: https://api.moonshot.cn/v1
LLM_BASE_URL=https://api.openai.com/v1
```

### Provider Examples

**Google Gemini (default):**
```bash
LLM_PROVIDER=gemini
LLM_API_KEY=your-gemini-key
LLM_MODEL=gemini-3-flash
```

**OpenAI:**
```bash
LLM_PROVIDER=openai
LLM_API_KEY=your-openai-key
LLM_MODEL=gpt-4
```

**Anthropic Claude:**
```bash
LLM_PROVIDER=anthropic
LLM_API_KEY=your-anthropic-key
LLM_MODEL=claude-3-5-sonnet-20241022
```

**OpenRouter (Claude, Llama, etc.):**
```bash
LLM_PROVIDER=openai
LLM_BASE_URL=https://openrouter.ai/api/v1
LLM_API_KEY=your-openrouter-key
LLM_MODEL=anthropic/claude-3.5-sonnet
```

**Fireworks.ai:**
```bash
LLM_PROVIDER=openai
LLM_BASE_URL=https://api.fireworks.ai/inference/v1
LLM_API_KEY=your-fireworks-key
LLM_MODEL=accounts/fireworks/models/llama-v3p1-70b-instruct
```

**Kimi:**
```bash
LLM_PROVIDER=openai
LLM_BASE_URL=https://api.moonshot.cn/v1
LLM_API_KEY=your-kimi-key
LLM_MODEL=kimi-k2-5-turbo
```

## Discord Webhook Notifications

The application can send notifications to a Discord channel when report generation starts, fails, or completes.

### Setup

1. Create a Discord webhook in your server:
   - Go to Server Settings → Integrations → Webhooks
   - Click "New Webhook", choose a channel, and copy the webhook URL

2. Add the webhook URL to your `.env`:

```bash
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN
```

### Notification Types

Three types of notifications are sent as rich Discord embeds:

1. **Start** (Blue) - When report generation begins
2. **Failed** (Red) - When an error occurs
3. **Finished** (Green) - When report is successfully uploaded to Google Docs

The Finished notification includes:
- Link to the Google Document
- The markdown report file attached (respecting Discord's 25MB limit)

### Configuration Options

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_WEBHOOK_URL` | No | "" | Discord webhook URL. If empty, notifications are disabled |
| `DISCORD_TIMEOUT` | No | "30" | HTTP client timeout in seconds |
| `DISCORD_RETRY_COUNT` | No | "1" | Number of retries on webhook failure |

### Error Handling

Discord webhook failures are logged as warnings but never block the main report generation pipeline. If the webhook is temporarily unavailable or the attachment is too large (>25MB), notifications will be gracefully skipped.

## Docker

The image uses a 3-stage build:

| Stage | Base | Purpose |
|---|---|---|
| `go-builder` | `golang:1.26.1` | Compiles the Go binary (static, CGO disabled) |
| `downloader` | `alpine:3.21` | Downloads the gws pre-built binary |
| runtime | `gcr.io/distroless/static-debian12` | Minimal static runtime (~2 MiB base image; final image adds the Go binary and `gws`) |

The distroless runtime has no shell or package manager, reducing the attack surface.

### Compose-only: `DEPLOY_IMAGE`

`docker-compose.yml` uses `image: ${DEPLOY_IMAGE:-gws-weekly-report:local}`. For local development you can omit it and run `docker compose build`. On a VPS that runs **pre-built images from GitHub Container Registry**, set `DEPLOY_IMAGE` in `.env` to the same image reference you pull (for example `ghcr.io/your-user/gws-weekly-report:main`, all lowercase). **Crontab runs a new shell without the variables from CI**, so this line must live in `.env` on the server; otherwise Compose falls back to `gws-weekly-report:local` and will not use the registry image.

## CI/CD (GitHub Actions)

Pushing to `main` runs [`.github/workflows/docker-build-deploy.yml`](.github/workflows/docker-build-deploy.yml): the image is built and pushed to **GHCR** (`ghcr.io/<owner>/<repo>`, tags `main` and a per-commit SHA), then the workflow connects to your VPS over **SSH**, runs `docker compose pull` for the `reporter` service, and `docker image prune -f` to remove dangling untagged layers. It does **not** run `docker compose up`; your existing crontab keeps scheduling runs.

### Repository secrets

| Secret | Description |
|--------|-------------|
| `VPS_HOST` | VPS hostname or IP address. |
| `VPS_USER` | SSH login user (must be able to run `docker` and `docker compose` in the app directory). |
| `VPS_SSH_KEY` | **Private** half of an SSH key pair used only for this deploy (see below). |
| `GHCR_PULL_TOKEN` | Optional. GitHub PAT with `read:packages`, if the container package is **private**. Omit for public packages. |

### Repository variables

| Variable | Description |
|----------|-------------|
| `VPS_APP_DIR` | Optional. Absolute path on the VPS where `docker-compose.yml` and `.env` live. Default when unset: `$HOME/gws-weekly-report` for the SSH user. |

### What to put in `VPS_SSH_KEY`

`VPS_SSH_KEY` is the **private** key whose matching **public** key is authorized on the VPS (usually listed in `~/.ssh/authorized_keys` for `VPS_USER`). It is often the same *kind* of key you use from your laptop (for example `~/.ssh/id_ed25519` or `~/.ssh/id_rsa`), but for automation it is safer to create a **dedicated deploy key** and only add its public part to the server:

```bash
ssh-keygen -t ed25519 -f ./gws-deploy -N ""
# Put gws-deploy.pub in ~/.ssh/authorized_keys on the VPS
# Put the entire contents of gws-deploy (including BEGIN/END lines) into the VPS_SSH_KEY secret
```

Paste the **whole private key file** into the GitHub secret (multi-line PEM). Do not commit it.

### One-time VPS checklist for CI

1. App directory on the server contains `docker-compose.yml`, `.env`, and `credentials.json`.
2. `.env` includes `DEPLOY_IMAGE=ghcr.io/<owner>/<repo>:main` (owner/repo **lowercase**, same as the workflow pushes).
3. The SSH user can run Docker (for example membership in the `docker` group).
4. For private GHCR images, either set `GHCR_PULL_TOKEN` or run `docker login ghcr.io` once on the VPS.

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
  notifier/                       # Discord webhook notifications
  pipeline/                       # DataSource interface, WeekRange, Runner
  gws/                            # gws CLI executor wrapper
  llm/                            # LLM provider integration for Technology Highlights
  sources/
    gmail/                        # Gmail source (finds Google Doc ID)
    github/                       # GitHub GraphQL source (authored + reviewed PRs)
    calendar/                     # Calendar source (weekly events)
    gchat/                        # Google Chat source (Key Metrics / OMTM)
    hackernews/                   # Hacker News source (Technology Highlights)
  uploader/drive/                 # Google Drive uploader
  report/                         # markdown template renderer
Dockerfile
docker-compose.yml
```
