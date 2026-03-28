package drive

import (
	"context"
	"fmt"

	"github.com/apikdech/gws-weekly-report/internal/gws"
)

// Uploader uploads a local file to a Google Drive document.
type Uploader struct {
	executor *gws.Executor
}

// NewUploader creates a DriveUploader.
func NewUploader(executor *gws.Executor) *Uploader {
	return &Uploader{executor: executor}
}

// Upload updates a Google Docs file with the contents of reportPath.
// Returns the raw gws CLI output on success.
func (u *Uploader) Upload(ctx context.Context, docID, reportPath string) ([]byte, error) {
	params := fmt.Sprintf(`{"fileId":%q}`, docID)
	out, err := u.executor.Run(ctx,
		"drive", "files", "update",
		"--params", params,
		"--upload", reportPath,
		"--upload-content-type", "text/markdown",
		"--json", `{"mimeType":"application/vnd.google-apps.document"}`,
	)
	if err != nil {
		return nil, fmt.Errorf("drive files update: %w", err)
	}
	return out, nil
}
