package notifier

import "time"

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
