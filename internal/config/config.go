package config

import (
	"errors"
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
		ReportTimezone:     os.Getenv("REPORT_TIMEZONE"),
		TempDir:            os.Getenv("TEMP_DIR"),
	}

	if cfg.ReportTimezone == "" {
		cfg.ReportTimezone = "UTC"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "/tmp"
	}

	var missing []string
	required := map[string]string{
		"GITHUB_TOKEN":                          cfg.GitHubToken,
		"GITHUB_USERNAME":                       cfg.GitHubUsername,
		"GWS_EMAIL_SENDER":                      cfg.GWSEmailSender,
		"REPORT_NAME":                           cfg.ReportName,
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE": cfg.GWSCredentialsFile,
	}
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, errors.New(fmt.Sprintf("missing required environment variables: %s", strings.Join(missing, ", ")))
	}

	return cfg, nil
}
