package config_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/config"
)

func TestLoad_AllRequiredPresent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")

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
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")
	os.Unsetenv("REPORT_TIMEZONE")
	os.Unsetenv("TEMP_DIR")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReportTimezone != "Asia/Jakarta" {
		t.Errorf("expected default timezone Asia/Jakarta, got %q", cfg.ReportTimezone)
	}
	if cfg.TempDir != "/tmp" {
		t.Errorf("expected default TempDir=/tmp, got %q", cfg.TempDir)
	}
}

func TestLoad_ReportNextActions(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_USERNAME", "user")
	t.Setenv("GWS_EMAIL_SENDER", "sender@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")
	t.Setenv("REPORT_NEXT_ACTIONS", " Continue dashboard , Ship feature X , ")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"Continue dashboard", "Ship feature X"}
	if !reflect.DeepEqual(cfg.NextActions, want) {
		t.Errorf("NextActions: got %#v, want %#v", cfg.NextActions, want)
	}
}

func TestLoad_ReportNextActionsEmpty(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_USERNAME", "user")
	t.Setenv("GWS_EMAIL_SENDER", "sender@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")
	t.Setenv("REPORT_NEXT_ACTIONS", "  ,  , ")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NextActions != nil {
		t.Errorf("NextActions: got %#v, want nil", cfg.NextActions)
	}
}

func TestConfigLoadsGChatFields(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_USERNAME", "user")
	t.Setenv("GWS_EMAIL_SENDER", "sender@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GWSChatSpacesID != "AAQAE4zqbX4" {
		t.Errorf("GWSChatSpacesID: got %q, want %q", cfg.GWSChatSpacesID, "AAQAE4zqbX4")
	}
	if cfg.GWSChatSenderName != "users/102650500894334129637" {
		t.Errorf("GWSChatSenderName: got %q, want %q", cfg.GWSChatSenderName, "users/102650500894334129637")
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

func TestLoad_LLMConfiguration(t *testing.T) {
	// Save and restore original env vars
	origVars := map[string]string{
		"GITHUB_TOKEN":                          os.Getenv("GITHUB_TOKEN"),
		"GITHUB_USERNAME":                       os.Getenv("GITHUB_USERNAME"),
		"GWS_EMAIL_SENDER":                      os.Getenv("GWS_EMAIL_SENDER"),
		"REPORT_NAME":                           os.Getenv("REPORT_NAME"),
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE": os.Getenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE"),
		"GWS_CHAT_SPACES_ID":                    os.Getenv("GWS_CHAT_SPACES_ID"),
		"GWS_CHAT_SENDER_NAME":                  os.Getenv("GWS_CHAT_SENDER_NAME"),
		"LLM_PROVIDER":                          os.Getenv("LLM_PROVIDER"),
		"LLM_BASE_URL":                          os.Getenv("LLM_BASE_URL"),
		"LLM_API_KEY":                           os.Getenv("LLM_API_KEY"),
		"LLM_MODEL":                             os.Getenv("LLM_MODEL"),
	}
	defer func() {
		for k, v := range origVars {
			os.Setenv(k, v)
		}
	}()

	// Set required env vars
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_USERNAME", "test-user")
	os.Setenv("GWS_EMAIL_SENDER", "test@example.com")
	os.Setenv("REPORT_NAME", "Test Report")
	os.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	os.Setenv("GWS_CHAT_SPACES_ID", "SPACE123")
	os.Setenv("GWS_CHAT_SENDER_NAME", "users/123")

	// Set LLM configuration
	os.Setenv("LLM_PROVIDER", "openai")
	os.Setenv("LLM_BASE_URL", "https://api.openai.com/v1")
	os.Setenv("LLM_API_KEY", "test-llm-key")
	os.Setenv("LLM_MODEL", "gpt-4")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %v, want openai", cfg.LLMProvider)
	}
	if cfg.LLMBaseURL != "https://api.openai.com/v1" {
		t.Errorf("LLMBaseURL = %v, want https://api.openai.com/v1", cfg.LLMBaseURL)
	}
	if cfg.LLMAPIKey != "test-llm-key" {
		t.Errorf("LLMAPIKey = %v, want test-llm-key", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "gpt-4" {
		t.Errorf("LLMModel = %v, want gpt-4", cfg.LLMModel)
	}
}

func TestLoad_DiscordConfig(t *testing.T) {
	// Set required vars
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")

	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123/abc")
	t.Setenv("DISCORD_TIMEOUT", "60")
	t.Setenv("DISCORD_RETRY_COUNT", "3")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DiscordWebhookURL != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("DiscordWebhookURL: expected webhook URL, got %q", cfg.DiscordWebhookURL)
	}

	if cfg.DiscordTimeout != 60 {
		t.Errorf("DiscordTimeout: expected 60, got %d", cfg.DiscordTimeout)
	}

	if cfg.DiscordRetryCount != 3 {
		t.Errorf("DiscordRetryCount: expected 3, got %d", cfg.DiscordRetryCount)
	}
}

func TestLoad_DiscordDefaults(t *testing.T) {
	// Set required vars
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_USERNAME", "testuser")
	t.Setenv("GWS_EMAIL_SENDER", "agent@example.com")
	t.Setenv("REPORT_NAME", "Test User")
	t.Setenv("GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE", "/tmp/creds.json")
	t.Setenv("GWS_CHAT_SPACES_ID", "AAQAE4zqbX4")
	t.Setenv("GWS_CHAT_SENDER_NAME", "users/102650500894334129637")
	// Don't set any Discord env vars

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DiscordWebhookURL != "" {
		t.Errorf("DiscordWebhookURL: expected empty, got %q", cfg.DiscordWebhookURL)
	}

	if cfg.DiscordTimeout != 30 {
		t.Errorf("DiscordTimeout: expected default 30, got %d", cfg.DiscordTimeout)
	}

	if cfg.DiscordRetryCount != 1 {
		t.Errorf("DiscordRetryCount: expected default 1, got %d", cfg.DiscordRetryCount)
	}
}
