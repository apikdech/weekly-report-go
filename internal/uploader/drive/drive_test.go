package drive_test

import (
	"context"
	"os"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/gws"
	"github.com/apikdech/gws-weekly-report/internal/uploader/drive"
)

func TestUploader_BuildsCorrectArgs(t *testing.T) {
	scriptPath := t.TempDir() + "/fake-gws.sh"
	script := "#!/bin/sh\necho \"$@\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	tmpCreds, err := os.CreateTemp("", "creds*.json")
	if err != nil {
		t.Fatalf("failed to create temp creds file: %v", err)
	}
	defer os.Remove(tmpCreds.Name())

	ex := gws.NewExecutor(scriptPath, tmpCreds.Name())
	u := drive.NewUploader(ex)

	reportPath := t.TempDir() + "/report.md"
	if err := os.WriteFile(reportPath, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := u.Upload(context.Background(), "docABC", reportPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out
}
