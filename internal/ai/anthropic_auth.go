package ai

import (
	"fmt"
	"os"
)

// Anthropic credential env vars, in resolution order.
const (
	// EnvAnthropicAPIKey holds an sk-ant-... API key. METERED.
	EnvAnthropicAPIKey = "ANTHROPIC_API_KEY"
	// EnvAnthropicAuthToken holds an OAuth access token from a Claude
	// subscription profile. SUBSCRIPTION QUOTA, not billed per token.
	EnvAnthropicAuthToken = "ANTHROPIC_AUTH_TOKEN"
)

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
	return AnthropicCredential{}, fmt.Errorf(
		"no Anthropic credential: set %s (metered API key) or %s (OAuth access token, subscription quota)",
		EnvAnthropicAPIKey, EnvAnthropicAuthToken)
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
