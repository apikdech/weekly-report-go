package gws_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/gws"
)

func TestExecutor_Run_EchoBinary(t *testing.T) {
	// Use "echo" as a stand-in for the gws binary to test executor mechanics.
	echoBin, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("echo not available")
	}

	tmpCreds, err := os.CreateTemp("", "creds*.json")
	if err != nil {
		t.Fatalf("failed to create temp creds file: %v", err)
	}
	defer os.Remove(tmpCreds.Name())

	ex := gws.NewExecutor(echoBin, tmpCreds.Name())
	out, err := ex.Run(context.Background(), "hello", "world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", string(out))
	}
}

func TestExecutor_Run_PropagatesError(t *testing.T) {
	// Use "false" binary which always exits non-zero.
	falseBin, err := exec.LookPath("false")
	if err != nil {
		t.Skip("false not available")
	}

	tmpCreds, err := os.CreateTemp("", "creds*.json")
	if err != nil {
		t.Fatalf("failed to create temp creds file: %v", err)
	}
	defer os.Remove(tmpCreds.Name())

	ex := gws.NewExecutor(falseBin, tmpCreds.Name())
	_, err = ex.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from non-zero exit, got nil")
	}
}
