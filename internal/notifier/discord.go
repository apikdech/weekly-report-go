package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

const discordMaxFileSize = 25 * 1024 * 1024 // 25MB

// DiscordEmbed represents a Discord embed object
type DiscordEmbed struct {
	Title     string              `json:"title"`
	Color     int                 `json:"color"`
	Fields    []DiscordEmbedField `json:"fields,omitempty"`
	Timestamp string              `json:"timestamp,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordWebhookPayload represents the JSON payload sent to Discord
type DiscordWebhookPayload struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

// buildStartEmbed creates an embed for start events
func buildStartEmbed(event *StartEvent) DiscordEmbed {
	return DiscordEmbed{
		Title: "📊 Weekly Report Generation Started",
		Color: 0x3498db, // Blue
		Fields: []DiscordEmbedField{
			{
				Name:   "Week Range",
				Value:  event.WeekRange,
				Inline: true,
			},
			{
				Name:   "Started At",
				Value:  event.Timestamp().Format(time.RFC3339),
				Inline: false,
			},
		},
		Timestamp: event.Timestamp().Format(time.RFC3339),
	}
}

// buildFailedEmbed creates an embed for failed events
func buildFailedEmbed(event *FailedEvent) DiscordEmbed {
	errorMsg := event.Error.Error()
	if len(errorMsg) > 1024 {
		errorMsg = errorMsg[:1021] + "..."
	}

	return DiscordEmbed{
		Title: "❌ Weekly Report Generation Failed",
		Color: 0xe74c3c, // Red
		Fields: []DiscordEmbedField{
			{
				Name:   "Week Range",
				Value:  event.WeekRange,
				Inline: true,
			},
			{
				Name:   "Error",
				Value:  errorMsg,
				Inline: false,
			},
			{
				Name:   "Failed At",
				Value:  event.Timestamp().Format(time.RFC3339),
				Inline: false,
			},
		},
		Timestamp: event.Timestamp().Format(time.RFC3339),
	}
}

// buildFinishedEmbed creates an embed for finished events
func buildFinishedEmbed(event *FinishedEvent) DiscordEmbed {
	return DiscordEmbed{
		Title: "✅ Weekly Report Generation Complete",
		Color: 0x2ecc71, // Green
		Fields: []DiscordEmbedField{
			{
				Name:   "Week Range",
				Value:  event.WeekRange,
				Inline: true,
			},
			{
				Name:   "Google Docs",
				Value:  "[Open Document](" + event.DocURL + ")",
				Inline: true,
			},
			{
				Name:   "Completed At",
				Value:  event.Timestamp().Format(time.RFC3339),
				Inline: false,
			},
		},
		Timestamp: event.Timestamp().Format(time.RFC3339),
	}
}

// DiscordHandler sends notifications to Discord via webhook
type DiscordHandler struct {
	webhookURL string
	httpClient *http.Client
	retryCount int
}

// NewDiscordHandler creates a new Discord handler
func NewDiscordHandler(webhookURL string, timeout, retryCount int) *DiscordHandler {
	return &DiscordHandler{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		retryCount: retryCount,
	}
}

// Supports returns true for all notification event types
func (d *DiscordHandler) Supports(eventType string) bool {
	return eventType == "start" || eventType == "failed" || eventType == "finished"
}

// Handle processes the event and sends it to Discord
func (d *DiscordHandler) Handle(event NotificationEvent) {
	switch e := event.(type) {
	case *StartEvent:
		if err := d.sendEmbed(buildStartEmbed(e)); err != nil {
			log.Printf("[discord] failed to send start notification: %v", err)
		}
	case *FailedEvent:
		if err := d.sendEmbed(buildFailedEmbed(e)); err != nil {
			log.Printf("[discord] failed to send failed notification: %v", err)
		}
	case *FinishedEvent:
		if err := d.sendFinishedWithAttachment(e); err != nil {
			log.Printf("[discord] failed to send finished notification: %v", err)
		}
	}
}

// sendEmbed sends a simple embed payload to Discord
func (d *DiscordHandler) sendEmbed(embed DiscordEmbed) error {
	payload := DiscordWebhookPayload{
		Embeds: []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal embed: %w", err)
	}

	return d.sendWithRetry(jsonData, nil)
}

// sendFinishedWithAttachment sends finished event with file attachment
func (d *DiscordHandler) sendFinishedWithAttachment(event *FinishedEvent) error {
	embed := buildFinishedEmbed(event)
	payload := DiscordWebhookPayload{
		Embeds: []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal embed: %w", err)
	}

	// Check file size
	fileInfo, err := os.Stat(event.ReportPath)
	if err != nil {
		return d.sendWithRetry(jsonData, nil) // Send without attachment if file not found
	}

	if fileInfo.Size() > discordMaxFileSize {
		// File too large, send without attachment
		return d.sendWithRetry(jsonData, nil)
	}

	fileData, err := os.ReadFile(event.ReportPath)
	if err != nil {
		return d.sendWithRetry(jsonData, nil) // Send without attachment if read fails
	}

	return d.sendMultipartWithRetry(jsonData, fileData)
}

// sendWithRetry sends JSON payload with retry logic
func (d *DiscordHandler) sendWithRetry(jsonData []byte, fileData []byte) error {
	var lastErr error

	for attempt := 0; attempt <= d.retryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second) // Linear backoff (1s, 2s, 3s...)
		}

		err := d.doRequest(jsonData)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("webhook failed after %d retries: %w", d.retryCount, lastErr)
}

// doRequest sends the actual HTTP request
func (d *DiscordHandler) doRequest(jsonData []byte) error {
	req, err := http.NewRequest("POST", d.webhookURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendMultipartWithRetry sends multipart form with file attachment
func (d *DiscordHandler) sendMultipartWithRetry(jsonData, fileData []byte) error {
	var lastErr error

	for attempt := 0; attempt <= d.retryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		err := d.doMultipartRequest(jsonData, fileData)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("webhook with attachment failed after %d retries: %w", d.retryCount, lastErr)
}

// doMultipartRequest sends multipart form data to Discord
func (d *DiscordHandler) doMultipartRequest(jsonData, fileData []byte) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add payload_json field
	payloadField, err := writer.CreateFormField("payload_json")
	if err != nil {
		return fmt.Errorf("create payload field: %w", err)
	}
	if _, err := payloadField.Write(jsonData); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	// Add file field
	fileField, err := writer.CreateFormFile("file", "report.md")
	if err != nil {
		return fmt.Errorf("create file field: %w", err)
	}
	if _, err := fileField.Write(fileData); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest("POST", d.webhookURL, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
