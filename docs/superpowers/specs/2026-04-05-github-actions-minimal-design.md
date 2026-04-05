# GitHub Actions Minimal Setup - Design Document

**Date:** 2026-04-05  
**Feature:** Minimal GitHub Actions Setup Example  
**Status:** Approved for Implementation

## Overview

Create a self-contained example directory (`examples/github-actions/`) that allows users to run the GWS Weekly Report tool entirely within GitHub Actions, eliminating the need for a VPS, Docker installation, or crontab management.

## Goals

1. **Zero Infrastructure**: Users don't need a VPS or Docker knowledge
2. **Simple Setup**: Copy workflow, add secrets, done
3. **Automatic Execution**: Runs on schedule via GitHub Actions cron
4. **No Code Changes**: Uses the existing pre-built GHCR image

## Target User

- Wants weekly reports without managing servers
- Comfortable with GitHub (has a GitHub account)
- Has Google Workspace credentials already
- Values simplicity over customizability

## Directory Structure

```
examples/github-actions/
├── README.md                      # Quick start guide
└── .github/
    └── workflows/
        └── weekly-report.yml      # The scheduled workflow
```

## Workflow Design

### Triggers
- **Schedule:** `cron: '0 13 * * 5'` (Friday 20:00 UTC+7 / 13:00 UTC)
- **Manual:** `workflow_dispatch` (for testing)

### Execution Flow
1. Checkout (not needed but kept for consistency)
2. Login to GHCR (public image, may not need auth)
3. Decode `GWS_CREDENTIALS_BASE64` secret to `credentials.json`
4. Pull pre-built image `ghcr.io/apikdech/gws-weekly-report:main`
5. Run container with all required environment variables
6. Cleanup credentials file

### Authentication Strategy

**GHCR Image Access:**
- Image is public, no authentication needed
- If private, use `GITHUB_TOKEN` with `read:packages`

**Google Workspace Credentials:**
- Store base64-encoded `credentials.json` as `GWS_CREDENTIALS_BASE64` secret
- Decode at runtime: `echo "$GWS_CREDENTIALS_BASE64" | base64 -d > credentials.json`
- Mount as volume to container

## Required GitHub Secrets

| Secret | Required | Description |
|--------|----------|-------------|
| `GWS_CREDENTIALS_BASE64` | Yes | Base64-encoded credentials.json from `gws auth export --unmasked` |
| `GITHUB_TOKEN` | Yes | GitHub personal access token with `repo` and `read:user` scopes |
| `GITHUB_USERNAME` | Yes | Your GitHub username |
| `GWS_EMAIL_SENDER` | Yes | Email address that sends the weekly report prompt |
| `REPORT_NAME` | Yes | Your full name as it appears in the email subject |
| `LLM_API_KEY` | No | API key for LLM provider (for Technology Highlights) |
| `DISCORD_WEBHOOK_URL` | No | Discord webhook URL for notifications |

## Main README Update

Add a new "Quick Start" section at the top of the main README.md that:
1. Highlights the two deployment options (VPS vs GitHub Actions)
2. Links to the `examples/github-actions/` directory
3. Briefly explains which option to choose

Example text:
```markdown
## Quick Start

Choose your deployment method:

- **[GitHub Actions (Recommended for simplicity)](examples/github-actions/)** - Runs entirely on GitHub. No VPS needed, no Docker setup. Just configure secrets and go.
- **[VPS with Docker](README.md#setup)** - Run on your own server with full control. Good for users who already have infrastructure.
```

## Workflow File Content

```yaml
name: Weekly Report

on:
  schedule:
    - cron: '0 13 * * 5'  # Every Friday 20:00 UTC+7 (13:00 UTC)
  workflow_dispatch:  # Manual trigger for testing

jobs:
  generate-report:
    runs-on: ubuntu-latest
    steps:
      - name: Decode GWS credentials
        run: |
          echo "${{ secrets.GWS_CREDENTIALS_BASE64 }}" | base64 -d > credentials.json

      - name: Run weekly report
        run: |
          docker run --rm \
            -e GITHUB_TOKEN="${{ secrets.GITHUB_TOKEN }}" \
            -e GITHUB_USERNAME="${{ secrets.GITHUB_USERNAME }}" \
            -e GWS_EMAIL_SENDER="${{ secrets.GWS_EMAIL_SENDER }}" \
            -e REPORT_NAME="${{ secrets.REPORT_NAME }}" \
            -e GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/run/secrets/gws-credentials \
            -e LLM_API_KEY="${{ secrets.LLM_API_KEY }}" \
            -e DISCORD_WEBHOOK_URL="${{ secrets.DISCORD_WEBHOOK_URL }}" \
            -v "$(pwd)/credentials.json:/run/secrets/gws-credentials:ro" \
            ghcr.io/apikdech/gws-weekly-report:main

      - name: Cleanup credentials
        if: always()
        run: rm -f credentials.json
```

## README.md for examples/github-actions/

Should include:
1. **Prerequisites** - What you need before starting
2. **Setup Steps** - Numbered list of what to do
3. **How to get credentials** - `gws auth export` instructions
4. **Testing** - How to run manually
5. **Troubleshooting** - Common issues

## Success Criteria

- [ ] A new user can go from zero to working weekly reports in under 15 minutes
- [ ] No VPS, Docker, or server knowledge required
- [ ] Workflow runs successfully on schedule
- [ ] Main README clearly presents both deployment options

## Out of Scope

- Self-building the Docker image (use pre-built GHCR image)
- Custom schedule configuration (use cron syntax in workflow)
- Multiple timezones support (keep simple, use UTC)
- Complex secret management (keep it simple with GitHub secrets)

## Future Enhancements (Not in this spec)

- Template repository for one-click setup
- GitHub Action input parameters for customization
- Support for multiple report schedules

## Implementation Notes

- Use the existing `ghcr.io/apikdech/gws-weekly-report:main` image
- Workflow should be copy-paste ready
- Secrets naming should match environment variable names for clarity
- Include a manual trigger (`workflow_dispatch`) for testing
- Always clean up credentials file even on failure (`if: always()`)
