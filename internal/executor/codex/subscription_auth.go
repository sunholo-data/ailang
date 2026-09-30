package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cloud codex jobs bill the ChatGPT SUBSCRIPTION, not the metered API (Mark,
// 2026-09-30 — OpenAI has confirmed this use is fine). The credential is a codex
// auth.json from a dedicated `codex login` (its own token family, so a refresh in
// the cloud never logs the rig out), stored in Secret Manager and installed here.
//
// A ChatGPT auth.json refreshes itself: codex rewrites the file with new tokens
// and a later last_refresh. A container is thrown away after one task, so the
// caller writes a refreshed file back to Secret Manager (see NewerRefresh) or the
// stored copy goes stale and every job after it fails auth.

// subscriptionAuth is the part of a ChatGPT auth.json this package inspects.
type subscriptionAuth struct {
	AuthMode    string `json:"auth_mode"`
	LastRefresh string `json:"last_refresh"`
	Tokens      struct {
		RefreshToken string `json:"refresh_token"`
	} `json:"tokens"`
}

// ParseSubscriptionAuth validates a ChatGPT-subscription auth.json and returns its
// last_refresh. An api-key file is REJECTED: installing one under the subscription
// path would silently move the job onto metered billing.
func ParseSubscriptionAuth(data []byte) (time.Time, error) {
	var a subscriptionAuth
	if err := json.Unmarshal(data, &a); err != nil {
		return time.Time{}, fmt.Errorf("codex auth.json is not valid JSON: %w", err)
	}
	if a.AuthMode != "chatgpt" {
		return time.Time{}, fmt.Errorf("codex auth.json has auth_mode %q, want \"chatgpt\" (subscription) — refusing a credential that would bill the metered API", a.AuthMode)
	}
	if a.Tokens.RefreshToken == "" {
		return time.Time{}, errors.New("codex auth.json has no refresh_token — re-run `codex login` to produce it")
	}
	t, err := time.Parse(time.RFC3339Nano, a.LastRefresh)
	if err != nil {
		return time.Time{}, fmt.Errorf("codex auth.json last_refresh %q: %w", a.LastRefresh, err)
	}
	return t, nil
}

// NewerRefresh reports whether cur is a valid subscription auth.json refreshed
// strictly after prev. It is the write-back guard: an unchanged file, an older
// one, or anything that is not a subscription credential is never persisted.
func NewerRefresh(prev, cur []byte) bool {
	pt, err := ParseSubscriptionAuth(prev)
	if err != nil {
		return false
	}
	ct, err := ParseSubscriptionAuth(cur)
	if err != nil {
		return false
	}
	return ct.After(pt)
}

// InstalledAuthPath is where codex reads its credential.
func InstalledAuthPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory for codex auth: %w", err)
	}
	return filepath.Join(home, ".codex", "auth.json"), nil
}

// InstallSubscriptionAuth writes a validated subscription auth.json to
// ~/.codex/auth.json. It refuses to replace an existing file: on a machine that
// already has a credential (the rig) that file is the one in use.
func InstallSubscriptionAuth(data []byte) error {
	if _, err := ParseSubscriptionAuth(data); err != nil {
		return err
	}
	path, err := InstalledAuthPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, authFileMode)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("refusing to overwrite existing %s", path)
		}
		return fmt.Errorf("create codex auth.json: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write codex auth.json: %w", err)
	}
	return nil
}
