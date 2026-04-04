package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/apikdech/gws-weekly-report/internal/config"
	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/llm"
	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	"github.com/apikdech/gws-weekly-report/internal/report"
	"github.com/apikdech/gws-weekly-report/internal/sources/calendar"
	"github.com/apikdech/gws-weekly-report/internal/sources/gchat"
	gh "github.com/apikdech/gws-weekly-report/internal/sources/github"
	"github.com/apikdech/gws-weekly-report/internal/sources/gmail"
	"github.com/apikdech/gws-weekly-report/internal/sources/hackernews"
	"github.com/apikdech/gws-weekly-report/internal/uploader/drive"
	anyllm "github.com/mozilla-ai/any-llm-go"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// 2. Compute week range
	loc, err := time.LoadLocation(cfg.ReportTimezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", cfg.ReportTimezone, err)
	}
	week := pipeline.WeekRangeFor(time.Now(), loc)
	log.Printf("Week: %s", week.HeaderLabel())

	// 3. Build gws executor
	gwsBin := "gws" // resolved from PATH in container
	if v := os.Getenv("GWS_BIN_PATH"); v != "" {
		gwsBin = v
	}
	executor := gws.NewExecutor(gwsBin, cfg.GWSCredentialsFile)

	// 4. Build sources
	gmailSrc := gmail.NewSource(executor, cfg.GWSEmailSender, cfg.ReportName)
	githubSrc := gh.NewSource(cfg.GitHubToken, cfg.GitHubUsername)
	calendarSrc := calendar.NewSource(executor)
	gchatSrc := gchat.NewSource(executor, cfg.GWSChatSpacesID, cfg.GWSChatSenderName)

	// Create LLM provider if configured
	var llmProvider anyllm.Provider
	if cfg.LLMAPIKey != "" {
		var err error
		llmProvider, err = llm.NewProvider(cfg)
		if err != nil {
			log.Printf("[main] WARNING: Failed to create LLM provider: %v", err)
			// Continue without LLM provider - section will be skipped
		}
	}

	hnSrc := hackernews.NewSource(llmProvider, cfg.LLMModel)

	// 5. Run pipeline
	reportData := &pipeline.ReportData{
		ReportName:  cfg.ReportName,
		Week:        week,
		PRsByRepo:   make(map[string]*pipeline.RepoPRs),
		NextActions: cfg.NextActions,
	}
	runner := pipeline.NewRunner([]pipeline.DataSource{gmailSrc, githubSrc, calendarSrc, gchatSrc, hnSrc})
	if err := runner.Run(ctx, reportData); err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}

	// 6. Render markdown
	markdown, err := report.Render(reportData)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	// 7. Write report.md to temp dir
	reportPath := filepath.Join(cfg.TempDir, "report.md")
	if err := os.WriteFile(reportPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("write report.md: %w", err)
	}
	log.Printf("Report written to %s", reportPath)

	// 8. Upload to Drive
	uploader := drive.NewUploader(executor)
	if _, err := uploader.Upload(ctx, reportData.DocID, reportPath); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	log.Printf("Uploaded report to Google Doc: https://docs.google.com/document/d/%s/edit", reportData.DocID)

	// 9. Cleanup
	if err := os.Remove(reportPath); err != nil {
		log.Printf("WARN: failed to remove %s: %v", reportPath, err)
	}
	log.Printf("Done.")
	return nil
}
