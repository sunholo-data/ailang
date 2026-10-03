// Package chatgpt drives OpenAI models on a ChatGPT subscription: the OAuth
// credential the codex CLI keeps in ~/.codex/auth.json, sent to the ChatGPT
// codex backend (Responses API, streaming only). Subscription quota, not
// per-token billing — the lane the mission's codex executors already use,
// made available to anything that calls std/ai (motoko in particular).
//
// The credential is READ, never written: codex owns the refresh, and two
// writers racing on auth.json would corrupt it. An expired token fails loudly
// with the command that refreshes it.
package chatgpt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// Credential is the subscription access token and the account it belongs to.
type Credential struct {
	AccessToken string
	AccountID   string
}

// authFile is the subset of codex's auth.json this lane reads.
type authFile struct {
	AuthMode string `json:"auth_mode"`
	Tokens   struct {
		AccessToken string `json:"access_token"`
		AccountID   string `json:"account_id"`
	} `json:"tokens"`
}

// AuthPath is $CODEX_HOME/auth.json, else ~/.codex/auth.json — where the codex
// CLI keeps it.
func AuthPath() (string, error) {
	if h := config.CodexHome(); h != "" {
		return filepath.Join(h, "auth.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "auth.json"), nil
}

// LoadCredential reads the subscription credential. It refuses an API-key
// auth.json (auth_mode "apikey" bills per token: that is the openai provider's
// lane, not this one) and an expired access token.
func LoadCredential() (Credential, error) {
	path, err := AuthPath()
	if err != nil {
		return Credential{}, err
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- the codex CLI's own credential path
	if err != nil {
		return Credential{}, fmt.Errorf("chatgpt: no ChatGPT credential at %s (log in with `codex login`): %w", path, err)
	}
	var f authFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return Credential{}, fmt.Errorf("chatgpt: %s is not valid JSON: %w", path, err)
	}
	if f.AuthMode != "" && f.AuthMode != "chatgpt" {
		return Credential{}, fmt.Errorf("chatgpt: %s is in %q mode, not a ChatGPT subscription login", path, f.AuthMode)
	}
	if f.Tokens.AccessToken == "" || f.Tokens.AccountID == "" {
		return Credential{}, fmt.Errorf("chatgpt: %s has no access token or account id (log in with `codex login`)", path)
	}
	if exp, ok := jwtExpiry(f.Tokens.AccessToken); ok && !time.Now().Before(exp) {
		return Credential{}, fmt.Errorf("chatgpt: the ChatGPT access token expired at %s; run any `codex` command to refresh it",
			exp.Format(time.RFC3339))
	}
	return Credential{AccessToken: f.Tokens.AccessToken, AccountID: f.Tokens.AccountID}, nil
}

// jwtExpiry reads the exp claim of a JWT without verifying it (the server
// does that); ok is false when the token is not a readable JWT.
func jwtExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
