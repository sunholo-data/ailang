package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// installPlugins registers marketplaces and installs third-party plugins.
// This runs before task execution. Best-effort: failures are logged but don't block execution.
func (e *ClaudeExecutor) installPlugins(ctx context.Context, plugins *executor.PluginsConfig, workspace string) {
	if plugins == nil {
		return
	}

	for _, mkt := range plugins.Marketplaces {
		cmd := exec.CommandContext(ctx, e.claudePath, "plugin", "marketplace", "add", mkt)
		if workspace != "" {
			cmd.Dir = workspace
		}
		if e.nvmBinDir != "" {
			cmd.Env = os.Environ()
			for i, v := range cmd.Env {
				if strings.HasPrefix(v, "PATH=") {
					cmd.Env[i] = "PATH=" + e.nvmBinDir + ":" + v[5:]
					break
				}
			}
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to add marketplace %s: %v (%s)\n", mkt, err, strings.TrimSpace(string(output)))
		}
	}

	for _, plugin := range plugins.Install {
		cmd := exec.CommandContext(ctx, e.claudePath, "plugin", "install", plugin)
		if workspace != "" {
			cmd.Dir = workspace
		}
		if e.nvmBinDir != "" {
			cmd.Env = os.Environ()
			for i, v := range cmd.Env {
				if strings.HasPrefix(v, "PATH=") {
					cmd.Env[i] = "PATH=" + e.nvmBinDir + ":" + v[5:]
					break
				}
			}
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to install plugin %s: %v (%s)\n", plugin, err, strings.TrimSpace(string(output)))
		}
	}
}

// writeCredentialsFile writes ~/.claude/.credentials.json from the
// CLAUDE_CODE_OAUTH_TOKEN environment variable (M-CLOUD-OAUTH).
//
// Claude Code authenticates locally via ~/.claude/.credentials.json.
// In cloud containers (Cloud Run Jobs), the OAuth token is injected as an
// env var from Secret Manager. This function bridges the two:
//
//	env var (inner):  {"accessToken":"...","refreshToken":"...","expiresAt":...}
//	file (wrapper):   {"claudeAiOauth":{"accessToken":"...","refreshToken":"...","expiresAt":...}}
//
// Returns nil if CLAUDE_CODE_OAUTH_TOKEN is not set (no-op for local dev).
func writeCredentialsFile() error {
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if token == "" {
		return nil
	}

	// Validate the token is valid JSON
	var inner json.RawMessage
	if err := json.Unmarshal([]byte(token), &inner); err != nil {
		return fmt.Errorf("CLAUDE_CODE_OAUTH_TOKEN is not valid JSON: %w", err)
	}

	// Wrap in the credentials file format that Claude Code expects
	wrapper := map[string]json.RawMessage{
		"claudeAiOauth": inner,
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write to ~/.claude/.credentials.json (default path)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	claudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		return fmt.Errorf("failed to create .claude dir: %w", err)
	}

	credPath := filepath.Join(claudeDir, ".credentials.json")
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}
	fmt.Fprintf(os.Stderr, "claude-auth: wrote credentials to %s (%d bytes)\n", credPath, len(data))

	// When CLAUDE_CONFIG_DIR is set it overrides ~/.claude/ entirely, so credentials
	// must also exist there — otherwise Claude prompts for login.
	if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" && configDir != claudeDir {
		if err := os.MkdirAll(configDir, 0700); err == nil {
			altPath := filepath.Join(configDir, ".credentials.json")
			if err := os.WriteFile(altPath, data, 0600); err == nil {
				fmt.Fprintf(os.Stderr, "claude-auth: also wrote credentials to %s\n", altPath)
			}
		}
	}

	return nil
}
