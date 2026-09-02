package anthropic

import "net/http"

// AuthMode selects how this client authenticates to the Anthropic API.
//
// The two modes are not interchangeable, and the difference is a BILLING
// difference, not a cosmetic one:
//
//   - AuthAPIKey sends `x-api-key: <key>`. The key is an sk-ant-... API key and
//     the account is charged per token. This is the metered lane.
//   - AuthOAuth sends `Authorization: Bearer <token>` plus
//     `anthropic-beta: oauth-2025-04-20`. The token is a short-lived OAuth
//     access token from a Claude subscription profile, and usage draws on that
//     subscription's quota rather than being billed per token.
//
// Sending an OAuth token in `x-api-key` does NOT work — the header is what
// selects the credential type, so a mode mismatch is a 401, not a fallback.
//
// Callers that record cost MUST branch on this: a run on AuthOAuth has a
// list-price-equivalent cost (real arithmetic over real tokens that nobody was
// charged), never a metered one. Reporting an OAuth run as metered invents
// spend that never happened.
type AuthMode int

const (
	// AuthAPIKey is the default: metered `x-api-key` auth.
	AuthAPIKey AuthMode = iota
	// AuthOAuth is subscription-quota auth via a Bearer access token.
	AuthOAuth
)

// oauthBeta is the beta flag Anthropic requires alongside a Bearer OAuth token.
// Without it the token is rejected even though the header shape is correct.
const oauthBeta = "oauth-2025-04-20"

// WithOAuth switches the client to OAuth Bearer auth, meaning the credential
// passed to NewClient is treated as an OAuth access token rather than an API key.
func WithOAuth() ClientOption {
	return func(c *Client) {
		c.authMode = AuthOAuth
	}
}

// AuthMode reports how this client authenticates. Cost accounting reads it to
// decide whether a run was billed.
func (c *Client) AuthMode() AuthMode { return c.authMode }

// applyAuthHeaders sets the credential and version headers on an outbound
// request.
//
// Every request path in this package (messages, step, streamstep) MUST route
// through here. When auth was written inline at each call site, adding a second
// mode meant remembering three places, and the ones that were missed would fail
// as a 401 on a code path with no test rather than at construction.
func (c *Client) applyAuthHeaders(h http.Header) {
	switch c.authMode {
	case AuthOAuth:
		h.Set("Authorization", "Bearer "+c.apiKey)
		h.Set("anthropic-beta", oauthBeta)
	default:
		h.Set("x-api-key", c.apiKey)
	}
	h.Set("anthropic-version", c.apiVersion)
}
