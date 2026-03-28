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
