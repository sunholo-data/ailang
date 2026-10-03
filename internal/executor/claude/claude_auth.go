package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/proctree"
)

// installPlugins registers marketplaces and installs third-party plugins.
// This runs before task execution. Best-effort: failures are logged but don't block execution.
func (e *ClaudeExecutor) installPlugins(ctx context.Context, plugins *executor.PluginsConfig, workspace string) {
	if plugins == nil {
		return
	}

	for _, mkt := range plugins.Marketplaces {
		cmd := exec.CommandContext(ctx, e.claudePath, "plugin", "marketplace", "add", mkt)
		proctree.Configure(cmd)
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
		proctree.Configure(cmd)
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

// ErrCredentialsUnderArtifactRoot is returned when the OAuth credential would be
// written under the shared artifacts mount (F-H6-1). Every executor lane, the
// external apikey lanes included, mounts that bucket read-write, and gcsfuse
// ignores file modes — a 0600 file there is readable by all of them.
var ErrCredentialsUnderArtifactRoot = errors.New("refusing to write the Claude OAuth credential under the shared artifacts mount")

// credentialArtifactRoot is the cloud artifacts mount; a var so tests can point it at a temp dir.
var credentialArtifactRoot = "/artifacts"

// underArtifactRoot reports whether dir is, or resolves (through symlinks) to,
// a path inside credentialArtifactRoot.
func underArtifactRoot(dir string) bool {
	roots := []string{credentialArtifactRoot}
	if r, err := filepath.EvalSymlinks(credentialArtifactRoot); err == nil {
		roots = append(roots, r)
	}
	dirs := []string{dir}
	if d, err := filepath.EvalSymlinks(dir); err == nil {
		dirs = append(dirs, d)
	}
	for _, root := range roots {
		for _, d := range dirs {
			rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(d))
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
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
	token, _ := config.ClaudeCodeOAuthToken()
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

	// F-H6-1: check every destination BEFORE writing any of them. The cloud job
	// points CLAUDE_CONFIG_DIR at local disk and symlinks only projects/ into the
	// bucket; a config dir on the mount means that setup was bypassed.
	configDir := config.ClaudeConfigDir()
	for _, dir := range []string{claudeDir, configDir} {
		if dir != "" && underArtifactRoot(dir) {
			return fmt.Errorf("%w: %s", ErrCredentialsUnderArtifactRoot, dir)
		}
	}
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
	if configDir != "" && configDir != claudeDir {
		if err := os.MkdirAll(configDir, 0700); err == nil {
			altPath := filepath.Join(configDir, ".credentials.json")
			if err := os.WriteFile(altPath, data, 0600); err == nil {
				fmt.Fprintf(os.Stderr, "claude-auth: also wrote credentials to %s\n", altPath)
			}
		}
	}

	return nil
}
