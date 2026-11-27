package agy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/proctree"
)

// The subscription route's sign-in is a FILE agy rewrites on every access-token
// refresh (measured 2026-11-23/24, agy 1.5.15): <HOME>/.gemini/antigravity-cli/
// antigravity-oauth-token, JSON {token:{access_token,token_type,refresh_token,expiry,
// expires_in}, auth_method, id_token_email}. A cloud job installs it from Secret
// Manager and writes it back after the task when it changed
// (design_docs/planned/m-agy-cloud-executor.md).

// tokenFileMode keeps the credential owner-only.
const tokenFileMode = 0o600

// tokenRelPath is the token file under an agy HOME.
var tokenRelPath = filepath.Join(".gemini", "antigravity-cli", "antigravity-oauth-token")

type subscriptionToken struct {
	Token struct {
		RefreshToken string `json:"refresh_token"`
		Expiry       string `json:"expiry"`
	} `json:"token"`
	AuthMethod string `json:"auth_method"`
}

// ParseSubscriptionToken validates an agy subscription token file and returns its
// access-token expiry. Anything without a refresh token is refused: installing it
// would leave the job unable to sign in, and nothing else belongs in this secret
// (the API-key route is not a cloud lane).
func ParseSubscriptionToken(data []byte) (expiry string, err error) {
	var t subscriptionToken
	if err := json.Unmarshal(bytes.TrimSpace(data), &t); err != nil {
		return "", fmt.Errorf("agy token: not JSON: %w", err)
	}
	if t.Token.RefreshToken == "" {
		return "", errors.New("agy token: no token.refresh_token (not an agy subscription sign-in)")
	}
	if t.AuthMethod == "" {
		return "", errors.New("agy token: no auth_method (not an agy subscription sign-in)")
	}
	return t.Token.Expiry, nil
}

// InstalledTokenPath is where the subscription route's token file lives. It
// requires an explicit AILANG_AGY_HOME_ROOT: a cloud job must never fall back to a
// HOME it did not choose.
func InstalledTokenPath() (string, error) {
	if config.AgyHomeRoot() == "" {
		return "", fmt.Errorf("agy cloud credential requires an explicit %s", config.EnvAgyHomeRoot)
	}
	home, err := routeHome(routeSubscription)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, tokenRelPath), nil
}

// SubscriptionHome is the subscription route's isolated HOME (where the token file
// and settings live), for callers that run agy themselves, such as RunQuota.
func SubscriptionHome() (string, error) { return routeHome(routeSubscription) }

// InstallSubscriptionToken validates data and writes it as the subscription route's
// token file (directories 0700, file 0600, atomic rename).
func InstallSubscriptionToken(data []byte) error {
	if _, err := ParseSubscriptionToken(data); err != nil {
		return err
	}
	path, err := InstalledTokenPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("agy token: create %s: %w", dir, err)
	}
	f, err := os.CreateTemp(dir, ".antigravity-oauth-token.*")
	if err != nil {
		return fmt.Errorf("agy token: %w", err)
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("agy token: write: %v %v", werr, cerr)
	}
	if err := os.Chmod(tmp, tokenFileMode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("agy token: install: %w", err)
	}
	return nil
}

// quotaTimeout bounds one `agy -p /quota` read.
const quotaTimeout = 45 * time.Second

// RunQuota runs `agy -p /quota` under home (no model call) and returns its output.
// It is a refresh-capable agy process like any other, so it gets the executor's
// lifecycle: own process group, hard deadline, and executor.ErrProcessTerminationUnconfirmed
// when something it spawned still holds its output after the kill. Evaluate the
// output with mission.EvaluateGoogleQuota.
func RunQuota(ctx context.Context, home string) ([]byte, error) {
	if err := EnsureLockedKeychain(home); err != nil {
		fmt.Fprintf(os.Stderr, "agy /quota: %v (a macOS keychain dialog may appear)\n", err)
	}
	ctx, cancel := context.WithTimeout(ctx, quotaTimeout)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, defaultAgyPath(), "--print-timeout", "30s", "-p", "/quota")
	proctree.Configure(cmd)
	cmd.Dir = os.TempDir()
	// Minimal env: the HOME, never an API key (it would switch agy to the key
	// route), never the self-updater.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "AGY_CLI_DISABLE_AUTO_UPDATE=true"}
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.WaitDelay = terminationGrace
	err := cmd.Run()
	if cmd.Process != nil {
		_ = proctree.KillGroup(cmd.Process.Pid)
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return out.Bytes(), fmt.Errorf("%w: a process spawned by agy /quota still held its output %v after exit",
			executor.ErrProcessTerminationUnconfirmed, terminationGrace)
	}
	if ctx.Err() == context.DeadlineExceeded {
		return out.Bytes(), fmt.Errorf("agy /quota timed out after %v", quotaTimeout)
	}
	return out.Bytes(), err
}
