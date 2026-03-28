package pipeline

import (
	"context"
	"fmt"
	"log"
)

// Runner executes a sequence of DataSources against a ReportData.
type Runner struct {
	sources []DataSource
}

// NewRunner creates a Runner with the given sources (executed in order).
func NewRunner(sources []DataSource) *Runner {
	return &Runner{sources: sources}
}

// Run fetches from each source then collects contributions into report.
// Stops and returns an error if any source fails.
func (r *Runner) Run(ctx context.Context, report *ReportData) error {
	for _, src := range r.sources {
		log.Printf("[pipeline] fetching: %s", src.Name())
		if err := src.Fetch(ctx, report.Week); err != nil {
			return fmt.Errorf("source %q fetch failed: %w", src.Name(), err)
		}
		log.Printf("[pipeline] contributing: %s", src.Name())
		if err := src.Contribute(report); err != nil {
			return fmt.Errorf("source %q contribute failed: %w", src.Name(), err)
		}
	}
	return nil
}
