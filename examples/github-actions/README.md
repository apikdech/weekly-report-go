# GitHub Actions Minimal Setup

Run your weekly report entirely within GitHub Actions - no VPS or server management required.

## Overview

This example shows how to run the GWS Weekly Report tool using GitHub Actions scheduled workflows. The pre-built Docker image from GitHub Container Registry (GHCR) is pulled and executed on GitHub's hosted runners.

## Prerequisites

- A GitHub account
- A Google Workspace account with [gws CLI](https://github.com/googleworkspace/cli) authenticated locally
- A GitHub personal access token with `repo` and `read:user` scopes
- A private GitHub repository to store this workflow and secrets

## Setup

### 1. Create a new private repository

Create a new **private** repository on GitHub (don't fork this repo - you just need the workflow file).

### 2. Copy the workflow file

Copy `.github/workflows/weekly-report.yml` from this directory to your new repository at the same path.

### 3. Export Google Workspace credentials

On your local machine where `gws` is authenticated:

```bash
gws auth export --unmasked > credentials.json
```

Then base64 encode it for use as a GitHub secret:

```bash
cat credentials.json | base64
```

Copy the output (a long string of characters).

### 4. Configure GitHub Secrets

Go to your repository → Settings → Secrets and variables → Actions → Repository secrets.

Add the following secrets:

| Secret | Value |
|--------|-------|
| `GWS_CREDENTIALS_BASE64` | The base64 output from step 3 |
| `GITHUB_TOKEN` | Your GitHub personal access token |
| `GITHUB_USERNAME` | Your GitHub username (e.g., `octocat`) |
| `GWS_EMAIL_SENDER` | Email address that sends the weekly report prompt (e.g., `hr@company.com`) |
| `REPORT_NAME` | Your full name as it appears in the email subject (e.g., `John Doe`) |

**Optional secrets:**

| Secret | Value |
|--------|-------|
| `LLM_API_KEY` | API key for LLM provider (Gemini/OpenAI/Anthropic) if you want Technology Highlights |
| `DISCORD_WEBHOOK_URL` | Discord webhook URL if you want notifications |

### 5. Test manually

Go to your repository → Actions → Weekly Report → Run workflow.

The workflow will execute immediately. Check the logs to see if the report was generated and uploaded to Google Docs.

### 6. Verify the schedule

The workflow is configured to run automatically every Monday at 9:00 AM UTC. No further action needed!

## How It Works

1. **On schedule**: GitHub Actions triggers the workflow every Monday
2. **Decode credentials**: The base64-encoded credentials are decoded to a JSON file
3. **Run container**: Docker pulls the pre-built image and mounts the credentials
4. **Generate report**: The tool fetches data from Gmail, GitHub, Calendar, etc.
5. **Upload**: Report is uploaded to your Google Doc
6. **Cleanup**: Credentials file is deleted

## Troubleshooting

### Workflow fails with "credentials not found"

- Verify `GWS_CREDENTIALS_BASE64` secret is set correctly
- Check that you used `cat credentials.json | base64` and copied the entire output
- Ensure the secret is set at the repository level (not environment level)

### "No Google Doc ID found"

- Make sure you've received the weekly report email in your Gmail
- Verify `GWS_EMAIL_SENDER` matches the actual sender email
- Check `REPORT_NAME` matches exactly how it appears in the email subject

### Docker pull fails

- The image `ghcr.io/apikdech/gws-weekly-report:main` is public, but if GHCR has issues, the workflow will retry automatically

### Need to change the schedule?

Edit the `cron` line in the workflow file:
```yaml
- cron: '0 9 * * 1'  # Format: minute hour day month day-of-week
```

Use [crontab.guru](https://crontab.guru/) to generate your desired schedule.

## Security Notes

- **Keep your repository private** - it contains workflow files that reference your secrets
- **Use a dedicated GitHub token** - create a fine-grained token with only `repo` and `read:user` scopes
- **Rotate credentials periodically** - re-export your gws credentials every few months
- **Never commit credentials.json** - the workflow creates it temporarily at runtime

## Customization

Want to customize the report? See the [full documentation](../../README.md) for all available environment variables and options.
