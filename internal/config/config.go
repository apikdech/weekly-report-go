package config

import (
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
	GWSChatSpacesID    string // Google Chat space ID, e.g. "AAQAE4zqbX4"
	GWSChatSenderName  string // sender.name to filter, e.g. "users/102650500894334129637"
	ReportTimezone     string
	TempDir            string
	// NextActions are weekly "next action" bullets, from REPORT_NEXT_ACTIONS (comma-separated).
	NextActions []string
	// GeminiAPIKey is optional. If not set, technology section will be skipped.
	GeminiAPIKey string
	// GeminiModel is the Gemini model to use. Defaults to "gemini-3-flash".
	GeminiModel string
	// LLMProvider is the LLM provider to use (gemini, openai, anthropic). Defaults to "gemini".
	LLMProvider string
	// LLMAPIKey is the API key for the LLM provider.
	LLMAPIKey string
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
		GWSChatSpacesID:    os.Getenv("GWS_CHAT_SPACES_ID"),
		GWSChatSenderName:  os.Getenv("GWS_CHAT_SENDER_NAME"),
		ReportTimezone:     os.Getenv("REPORT_TIMEZONE"),
		TempDir:            os.Getenv("TEMP_DIR"),
	}

	if cfg.ReportTimezone == "" {
		cfg.ReportTimezone = "UTC"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "/tmp"
	}
	cfg.NextActions = parseCommaSeparated(os.Getenv("REPORT_NEXT_ACTIONS"))

	cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	cfg.GeminiModel = os.Getenv("GEMINI_MODEL")
	cfg.LLMProvider = os.Getenv("LLM_PROVIDER")
	cfg.LLMAPIKey = os.Getenv("LLM_API_KEY")

	type requiredVar struct {
		name string
		val  string
	}
	required := []requiredVar{
		{"GITHUB_TOKEN", cfg.GitHubToken},
		{"GITHUB_USERNAME", cfg.GitHubUsername},
		{"GWS_EMAIL_SENDER", cfg.GWSEmailSender},
		{"REPORT_NAME", cfg.ReportName},
		{"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", cfg.GWSCredentialsFile},
		{"GWS_CHAT_SPACES_ID", cfg.GWSChatSpacesID},
		{"GWS_CHAT_SENDER_NAME", cfg.GWSChatSenderName},
	}
	var missing []string
	for _, r := range required {
		if r.val == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// parseCommaSeparated splits on commas, trims spaces, and drops empty segments.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
