# Discord Webhook Notification System Design

**Date:** 2025-04-05  
**Author:** OpenCode  
**Status:** Approved

## 1. Overview

This design document describes an event-based Discord webhook notification system for the GWS Weekly Report generator. The system will notify a Discord channel when the report generation starts, fails, or completes successfully.

## 2. Requirements

### 2.1 Functional Requirements

- Send notifications to Discord when:
  1. Report generation **starts** (with week range info)
  2. Report generation **fails** (with error details)
  3. Report generation **finishes** (with Google Docs URL and markdown file attachment)
- Include rich Discord embeds with appropriate colors:
  - Blue (#3498db) for start
  - Red (#e74c3c) for failure
  - Green (#2ecc71) for success
- Upload the markdown report file as an attachment to the finished notification
- Make notifications optional via environment variable

### 2.2 Non-Functional Requirements

- Notifications should not block or fail the main pipeline
- Network failures should be handled gracefully with retry logic
- File attachments should respect Discord's 25MB limit
- System should be extensible for future notification channels

## 3. Architecture

### 3.1 High-Level Design

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   main.go       │────▶│  Event Emitter   │────▶│ Discord Handler │
│                 │     │  (internal/      │     │ (internal/      │
│                 │     │  notifier/       │     │ notifier/       │
│                 │     │  events.go)      │     │ discord.go)     │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │  Future Handlers │
                        │  (Slack, Email)  │
                        └──────────────────┘
```

### 3.2 Event System

The event system uses a simple synchronous emitter pattern suitable for a CLI tool:

#### Event Types

```go
// NotificationEvent is the base interface for all notification events
type NotificationEvent interface {
    Type() string
    Timestamp() time.Time
}

// StartEvent is emitted when report generation begins
type StartEvent struct {
    WeekRange string    // e.g., "22 March 2026 - 28 March 2026"
    Timestamp time.Time
}

// FailedEvent is emitted when pipeline fails
type FailedEvent struct {
    WeekRange string
    Error     error
    Timestamp time.Time
}

// FinishedEvent is emitted when report is successfully uploaded
type FinishedEvent struct {
    WeekRange    string
    DocID        string
    DocURL       string
    ReportPath   string
    Timestamp    time.Time
}
```

#### Event Emitter

```go
type EventEmitter struct {
    handlers map[string][]EventHandler
}

type EventHandler interface {
    Handle(event NotificationEvent)
    Supports(eventType string) bool
}

func (e *EventEmitter) Register(handler EventHandler)
func (e *EventEmitter) Emit(event NotificationEvent)
```

### 3.3 Discord Handler

The Discord handler implements the `EventHandler` interface:

```go
type DiscordHandler struct {
    webhookURL string
    httpClient *http.Client
    retryCount int
}

func (d *DiscordHandler) Handle(event notifier.NotificationEvent)
func (d *DiscordHandler) Supports(eventType string) bool
```

#### Discord Embed Structure

**Start Event:**
- Title: "📊 Weekly Report Generation Started"
- Color: 0x3498db (Blue)
- Fields:
  - Week Range
  - Started At (timestamp)

**Failed Event:**
- Title: "❌ Weekly Report Generation Failed"
- Color: 0xe74c3c (Red)
- Fields:
  - Week Range
  - Error Message (truncated to 1024 chars)
  - Failed At (timestamp)

**Finished Event:**
- Title: "✅ Weekly Report Generation Complete"
- Color: 0x2ecc71 (Green)
- Fields:
  - Week Range
  - Google Docs URL (clickable link)
  - Completed At (timestamp)
- File Attachment: report.md

## 4. Implementation Details

### 4.1 File Structure

```
internal/
  notifier/
    events.go          # Event types and emitter
    discord.go         # Discord webhook handler
    discord_test.go    # Tests for Discord handler
```

### 4.2 Configuration

New environment variables (all optional):

| Variable | Default | Description |
|----------|---------|-------------|
| `DISCORD_WEBHOOK_URL` | "" | Discord webhook URL. If empty, notifications are disabled |
| `DISCORD_TIMEOUT` | "30" | HTTP client timeout in seconds |
| `DISCORD_RETRY_COUNT` | "1" | Number of retries on failure |

### 4.3 Main.go Integration

```go
func run() error {
    // 1. Load config
    cfg, err := config.Load()
    // ...

    // 2. Initialize event emitter and register Discord handler if configured
    emitter := notifier.NewEventEmitter()
    if cfg.DiscordWebhookURL != "" {
        discordHandler := notifier.NewDiscordHandler(cfg.DiscordWebhookURL, cfg.DiscordTimeout, cfg.DiscordRetryCount)
        emitter.Register(discordHandler)
    }

    // 3. Compute week range
    week := pipeline.WeekRangeFor(time.Now(), loc)
    
    // 4. Emit start event
    emitter.Emit(&notifier.StartEvent{
        WeekRange: week.HeaderLabel(),
        Timestamp: time.Now(),
    })

    // 5. Run pipeline with error handling
    reportData := &pipeline.ReportData{...}
    runner := pipeline.NewRunner(sources)
    
    if err := runner.Run(ctx, reportData); err != nil {
        emitter.Emit(&notifier.FailedEvent{
            WeekRange: week.HeaderLabel(),
            Error:     err,
            Timestamp: time.Now(),
        })
        return fmt.Errorf("pipeline: %w", err)
    }

    // 6. Render and write report
    // ...

    // 7. Upload to Drive
    // ...

    // 8. Emit finished event (before cleanup so file exists)
    emitter.Emit(&notifier.FinishedEvent{
        WeekRange:  week.HeaderLabel(),
        DocID:      reportData.DocID,
        DocURL:     fmt.Sprintf("https://docs.google.com/document/d/%s/edit", reportData.DocID),
        ReportPath: reportPath,
        Timestamp: time.Now(),
    })

    // 9. Cleanup
    os.Remove(reportPath)
    return nil
}
```

### 4.4 Discord Webhook Payload

**JSON Payload Structure:**

```json
{
  "embeds": [{
    "title": "✅ Weekly Report Generation Complete",
    "color": 3066993,
    "fields": [
      {
        "name": "Week Range",
        "value": "22 March 2026 - 28 March 2026",
        "inline": true
      },
      {
        "name": "Google Docs",
        "value": "[Open Document](https://docs.google.com/document/d/.../edit)",
        "inline": true
      },
      {
        "name": "Completed At",
        "value": "2026-03-28 09:15:00 UTC",
        "inline": false
      }
    ],
    "timestamp": "2026-03-28T09:15:00Z"
  }]
}
```

**Multipart Form Data for File Upload:**

For finished events with file attachments, use `multipart/form-data`:
- `payload_json`: JSON string with embeds
- `file`: The report.md file content

### 4.5 Error Handling Strategy

1. **Webhook URL not configured**: Skip registration, no events emitted to Discord
2. **Network error**: Log warning, retry with exponential backoff (1s, 2s), then give up
3. **HTTP error (4xx/5xx)**: Log warning with status code, don't retry 4xx errors
4. **File too large (>25MB)**: Log warning, send notification without attachment
5. **Discord rate limit (429)**: Read Retry-After header, wait, then retry

All errors are logged but never returned to the caller - notifications are best-effort.

## 5. Testing Strategy

### 5.1 Unit Tests

- Mock HTTP server for Discord webhook testing
- Test all three event types produce correct JSON payloads
- Test file attachment multipart encoding
- Test retry logic with mock failures
- Test error handling (non-blocking behavior)

### 5.2 Integration Tests

- Test with actual Discord webhook URL (optional, manual)
- Verify embeds render correctly in Discord
- Test with large files (>25MB) to verify attachment skipping

## 6. Security Considerations

- Webhook URL should be treated as a secret (stored in environment variable, not committed)
- No sensitive data in notifications beyond what's already in the report
- HTTPS only for webhook URLs (validate in code)

## 7. Future Extensibility

The event-based design allows easy addition of new notification channels:

1. Create a new handler implementing `EventHandler`
2. Register it with the emitter
3. Configure via environment variables

Example future handlers:
- Slack webhook handler
- Email notification handler
- Telegram bot handler

## 8. Migration Plan

This feature is purely additive - no changes to existing functionality:

1. Add `internal/notifier/` package with events and Discord handler
2. Update `internal/config/config.go` with new environment variables
3. Modify `cmd/reporter/main.go` to initialize emitter and emit events
4. Update `.env.example` with new optional variables
5. Update README.md with Discord configuration instructions

## 9. Open Questions

None - all requirements clarified during design phase.

---

**Approved by:** User via conversational design review  
**Implementation Plan:** See `docs/superpowers/plans/2025-04-05-discord-webhook-notification.md`
