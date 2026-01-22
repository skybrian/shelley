package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	startupHookFilename = "on-conversation-start"
	startupHookTimeout  = 5 * time.Second
)

// StartupHookResult contains the result of running the startup hook.
type StartupHookResult struct {
	Output string
	Error  error
}

// RunStartupHook runs the startup hook if it exists.
// It looks for ~/.config/shelley/on-conversation-start
// The hook runs with the working directory set to cwd.
// Returns nil if no hook exists or if SHELLEY_DISABLE_STARTUP_HOOK is set.
func RunStartupHook(ctx context.Context, cwd string) *StartupHookResult {
	if os.Getenv("SHELLEY_DISABLE_STARTUP_HOOK") != "" {
		return nil
	}

	hookPath := findStartupHook()
	if hookPath == "" {
		return nil
	}

	// Check if executable
	info, err := os.Stat(hookPath)
	if err != nil {
		return &StartupHookResult{Error: fmt.Errorf("failed to stat hook: %w", err)}
	}
	if info.Mode()&0111 == 0 {
		return &StartupHookResult{Error: fmt.Errorf("hook is not executable: %s", hookPath)}
	}

	// Run with timeout
	ctx, cancel := context.WithTimeout(ctx, startupHookTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, hookPath)
	if cwd != "" {
		cmd.Dir = cwd
	}

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return &StartupHookResult{
			Output: string(output),
			Error:  fmt.Errorf("hook timed out after %v", startupHookTimeout),
		}
	}
	if err != nil {
		return &StartupHookResult{
			Output: string(output),
			Error:  fmt.Errorf("hook failed: %w", err),
		}
	}

	return &StartupHookResult{Output: string(output)}
}

// findStartupHook returns the path to the startup hook if it exists.
func findStartupHook() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check ~/.config/shelley/on-conversation-start
	hookPath := filepath.Join(home, ".config", "shelley", startupHookFilename)
	if _, err := os.Stat(hookPath); err == nil {
		return hookPath
	}

	return ""
}
