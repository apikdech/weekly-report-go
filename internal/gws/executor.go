package gws

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Executor runs the gws CLI binary with credentials injected via environment.
type Executor struct {
	gwsBinPath      string
	credentialsFile string
}

// NewExecutor creates an Executor using the given binary path and credentials file path.
func NewExecutor(gwsBinPath, credentialsFile string) *Executor {
	return &Executor{
		gwsBinPath:      gwsBinPath,
		credentialsFile: credentialsFile,
	}
}

// Run executes the gws binary with the given arguments.
// It injects GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE into the process environment.
// Returns stdout bytes on success, or an error containing stderr on non-zero exit.
func (e *Executor) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.gwsBinPath, args...)
	cmd.Env = append(cmd.Environ(),
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE="+e.credentialsFile,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gws command %v failed: %w\nstderr: %s", args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
