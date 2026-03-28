package config_test

import (
	"os"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/config"
)

func TestLoad_AllRequiredPresent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.GitHubToken != "ghp_test" {
		t.Errorf("expected GitHubToken=ghp_test, got %q", cfg.GitHubToken)
	}
	if cfg.GitHubUsername != "testuser" {
		t.Errorf("expected GitHubUsername=testuser, got %q", cfg.GitHubUsername)
	}
	if cfg.GWSEmailSender != "agent@example.com" {
		t.Errorf("expected GWSEmailSender=agent@example.com, got %q", cfg.GWSEmailSender)
	}
	if cfg.ReportName != "Test User" {
		t.Errorf("expected ReportName=Test User, got %q", cfg.ReportName)
	}
	if cfg.GWSCredentialsFile != "/tmp/creds.json" {
		t.Errorf("expected GWSCredentialsFile=/tmp/creds.json, got %q", cfg.GWSCredentialsFile)
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	os.Unsetenv("REPORT_TIMEZONE")
	os.Unsetenv("TEMP_DIR")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReportTimezone != "UTC" {
		t.Errorf("expected default timezone UTC, got %q", cfg.ReportTimezone)
	}
	if cfg.TempDir != "/tmp" {
		t.Errorf("expected default TempDir=/tmp, got %q", cfg.TempDir)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_USERNAME")
	os.Unsetenv("GWS_EMAIL_SENDER")
	os.Unsetenv("REPORT_NAME")
	os.Unsetenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing required vars, got nil")
	}
}
