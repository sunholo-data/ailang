package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Anthropic credential env vars, in resolution order.
const (
	// EnvAnthropicAPIKey holds an sk-ant-... API key. METERED.
	EnvAnthropicAPIKey = "ANTHROPIC_API_KEY"
	// EnvAnthropicAuthToken holds an OAuth access token from a Claude
	// subscription profile. SUBSCRIPTION QUOTA, not billed per token.
	EnvAnthropicAuthToken = "ANTHROPIC_AUTH_TOKEN"
	// EnvClaudeCodeOAuthToken holds the JSON credential blob this fleet already
	// injects for headless Claude in cloud containers (M-CLOUD-OAUTH):
	// {"accessToken":"...","refreshToken":"...","expiresAt":...}
	EnvClaudeCodeOAuthToken = "CLAUDE_CODE_OAUTH_TOKEN"
)

// claudeCredentialsPath is where Claude Code keeps its local OAuth credential.
// This is the SAME file the `claude` CLI reads, which is the whole point: agent
// mode has always "just worked" because it execs the CLI and the CLI reads this
// itself, while standard mode makes the HTTP call in-process and so had nothing
// to read. Reading it here closes that gap — the credential a user already has
// is the credential standard mode uses, with no extra setup and no secret
// passing through a human's hands.
func claudeCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", ".credentials.json"), nil
}

// claudeOAuth is the inner shape of both the env blob and the on-disk file's
// claudeAiOauth member. Only the fields we need are declared.
type claudeOAuth struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"` // epoch MILLIseconds
}

// valid reports whether the token is usable, with a reason when it is not.
// An expired token is rejected here rather than sent: the provider would answer
// with a bare 401 that reads like a configuration error, and every benchmark in
// the run would bank as api_error for a cause nobody could see.
func (c claudeOAuth) valid() error {
	if c.AccessToken == "" {
		return fmt.Errorf("credential contains no accessToken")
	}
	if c.ExpiresAt > 0 && time.Now().UnixMilli() >= c.ExpiresAt {
		return fmt.Errorf("Claude OAuth token expired at %s; run any `claude` command to refresh it",
			time.UnixMilli(c.ExpiresAt).Format(time.RFC3339))
	}
	return nil
}

// credentialsFileToken reads the local Claude Code credential.
func credentialsFileToken() (string, error) {
	path, err := claudeCredentialsPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- fixed path under the user's home
	if err != nil {
		return "", err
	}
	var file struct {
		ClaudeAIOauth claudeOAuth `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	if err := file.ClaudeAIOauth.valid(); err != nil {
		return "", err
	}
	return file.ClaudeAIOauth.AccessToken, nil
}

// AnthropicCredential is a resolved Anthropic credential plus the lane it
// implies. OAuth is carried explicitly rather than sniffed from the token's
// shape: guessing the lane from a string prefix is how a subscription run gets
// reported as metered spend.
type AnthropicCredential struct {
	// Value is the raw secret. Never log it.
	Value string
	// OAuth is true when Value is an OAuth access token (Bearer + the oauth
	// beta header) rather than an API key (x-api-key).
	OAuth bool
}

// Lane returns a short human-readable name for the billing lane. Safe to log —
// it never contains the credential.
func (c AnthropicCredential) Lane() string {
	if c.OAuth {
		return "oauth-subscription"
	}
	return "api-key-metered"
}

// ResolveAnthropicCredential picks the Anthropic credential from the
// environment and reports which billing lane it selects.
//
// Order is ANTHROPIC_API_KEY, then ANTHROPIC_AUTH_TOKEN — deliberately matching
// the official Anthropic SDKs and the `claude` CLI, so a machine configured for
// one behaves the same here. The consequence worth knowing: if BOTH are set the
// metered key wins, which is the same precedence that has already caused one
// real mis-billing incident on this rig (an inherited API key silently
// outbidding keychain OAuth on the agent path). Callers should surface
// Lane() rather than assume.
//
// Fails loudly when neither is set: an unauthenticated request would 401 at the
// provider with a far less obvious message, and there is no safe default lane
// to fall back to (CLAUDE.md §2 — a fallback that changes who gets billed is
// exactly the class that must error instead).
func ResolveAnthropicCredential() (AnthropicCredential, error) {
	if k := os.Getenv(EnvAnthropicAPIKey); k != "" {
		return AnthropicCredential{Value: k, OAuth: false}, nil
	}
	if t := os.Getenv(EnvAnthropicAuthToken); t != "" {
		return AnthropicCredential{Value: t, OAuth: true}, nil
	}
	// The fleet's existing cloud convention: a JSON blob in the env.
	if blob := os.Getenv(EnvClaudeCodeOAuthToken); blob != "" {
		var c claudeOAuth
		if err := json.Unmarshal([]byte(blob), &c); err != nil {
			return AnthropicCredential{}, fmt.Errorf("%s is not valid JSON: %w", EnvClaudeCodeOAuthToken, err)
		}
		if err := c.valid(); err != nil {
			return AnthropicCredential{}, fmt.Errorf("%s: %w", EnvClaudeCodeOAuthToken, err)
		}
		return AnthropicCredential{Value: c.AccessToken, OAuth: true}, nil
	}
	// Finally the local file the `claude` CLI itself uses. This is what makes
	// standard mode work with no setup wherever agent mode already works.
	if tok, err := credentialsFileToken(); err == nil {
		return AnthropicCredential{Value: tok, OAuth: true}, nil
	} else if !os.IsNotExist(err) {
		// A file that exists but is expired/malformed is a REAL problem: saying
		// "no credential" would send the reader hunting for a missing key when
		// the actual fix is to refresh an existing one.
		return AnthropicCredential{}, fmt.Errorf("Claude OAuth credential unusable: %w", err)
	}
	path, _ := claudeCredentialsPath()
	return AnthropicCredential{}, fmt.Errorf(
		"no Anthropic credential: set %s (metered API key), %s / %s (OAuth), or log in with the claude CLI so %s exists",
		EnvAnthropicAPIKey, EnvAnthropicAuthToken, EnvClaudeCodeOAuthToken, path)
}

// AnthropicLaneIsOAuth reports whether the CURRENT environment selects the
// OAuth subscription lane for Anthropic.
//
// Cost classification calls this so a standard-mode Anthropic row run on a
// subscription is labelled list-price-equivalent rather than metered. It reads
// the same env, in the same order, as ResolveAnthropicCredential — deliberately
// one implementation, because a classifier that disagrees with the client about
// which lane ran is worse than no classifier at all.
func AnthropicLaneIsOAuth() bool {
	c, err := ResolveAnthropicCredential()
	return err == nil && c.OAuth
}
