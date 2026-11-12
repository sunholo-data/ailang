package apiserver

// The A2A card's OAuth 2.0 security scheme (M-EXECUTOR-GE-A2A follow-up).
// Under --mcp-connect-only, /a2a/ needs a Bearer token from --oauth-issuer,
// so the card says so: securitySchemes.oauth is an authorization-code flow
// whose authorizationUrl and tokenUrl come from the issuer's RFC 8414
// metadata, and security requires it. A client (or Gemini Enterprise's
// registration form, which pre-fills its URL fields from it) learns how to
// sign in from the card alone. The client id is never on the card: each
// registration has its own.
//
// The metadata is fetched on demand and cached, outside the server lock. If
// the issuer cannot be read, or its metadata is not for that issuer, the card
// is served without the scheme and the reason is logged once per failure
// window: the gate still answers 401 with resource_metadata, as before.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	issuerMetaPath    = "/.well-known/oauth-authorization-server"
	issuerMetaTTL     = time.Hour
	issuerMetaRetry   = time.Minute // after a failed read
	issuerMetaTimeout = 3 * time.Second
	issuerMetaMaxBody = 64 << 10
)

// issuerMetaClient fetches the issuer's metadata. A var for tests.
var issuerMetaClient = &http.Client{Timeout: issuerMetaTimeout}

type issuerMeta struct {
	authorizationURL string
	tokenURL         string
	scopes           []string
}

type issuerMetaCache struct {
	mu      sync.Mutex
	meta    *issuerMeta
	err     error
	expires time.Time
}

// get returns the issuer's endpoints, reading them at most once per TTL (or
// per retry window after a failure).
func (c *issuerMetaCache) get(ctx context.Context, issuer string) (*issuerMeta, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.expires) {
		return c.meta, c.err
	}
	c.meta, c.err = fetchIssuerMeta(ctx, issuer)
	if c.err != nil {
		c.expires = time.Now().Add(issuerMetaRetry)
		log.Printf("  A2A card: no OAuth security scheme (%v)", c.err)
	} else {
		c.expires = time.Now().Add(issuerMetaTTL)
	}
	return c.meta, c.err
}

func fetchIssuerMeta(ctx context.Context, issuer string) (*issuerMeta, error) {
	issuer = strings.TrimRight(issuer, "/")
	ctx, cancel := context.WithTimeout(ctx, issuerMetaTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+issuerMetaPath, nil)
	if err != nil {
		return nil, err
	}
	resp, err := issuerMetaClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reading %s%s: %v", issuer, issuerMetaPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s%s answered %d", issuer, issuerMetaPath, resp.StatusCode)
	}
	var m struct {
		Issuer                string   `json:"issuer"`
		AuthorizationEndpoint string   `json:"authorization_endpoint"`
		TokenEndpoint         string   `json:"token_endpoint"`
		ScopesSupported       []string `json:"scopes_supported"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, issuerMetaMaxBody)).Decode(&m); err != nil {
		return nil, fmt.Errorf("issuer metadata is not JSON: %v", err)
	}
	// RFC 8414 §3.3: the metadata must name the issuer it was read from.
	if strings.TrimRight(m.Issuer, "/") != issuer {
		return nil, fmt.Errorf("issuer metadata names %q, want %q", m.Issuer, issuer)
	}
	for _, u := range []string{m.AuthorizationEndpoint, m.TokenEndpoint} {
		if !sameSchemeURL(issuer, u) {
			return nil, fmt.Errorf("issuer endpoint %q is not a %s URL", u, strings.SplitN(issuer, ":", 2)[0])
		}
	}
	return &issuerMeta{authorizationURL: m.AuthorizationEndpoint, tokenURL: m.TokenEndpoint, scopes: m.ScopesSupported}, nil
}

// sameSchemeURL: an https issuer's endpoints must be https; an http issuer
// (a local test) may use http.
func sameSchemeURL(issuer, u string) bool {
	if strings.HasPrefix(issuer, "https://") {
		return strings.HasPrefix(u, "https://")
	}
	return strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")
}

// addA2ASecurity puts the OAuth scheme on the card when /a2a/ is gated by an
// OAuth issuer's tokens.
func (s *Server) addA2ASecurity(ctx context.Context, card map[string]any) {
	if !s.mcpConnectOnly || s.a2aHandlerFn == nil || s.oauthIssuer == "" {
		return
	}
	m, err := s.issuerMeta.get(ctx, s.oauthIssuer)
	if err != nil {
		return
	}
	scopes := map[string]string{}
	for _, sc := range m.scopes {
		scopes[sc] = sc
	}
	card["securitySchemes"] = map[string]any{
		"oauth": map[string]any{
			"type":              "oauth2",
			"description":       "Sign in with the service's OAuth 2.0 authorization server (authorization code with S256 PKCE).",
			"oauth2MetadataUrl": strings.TrimRight(s.oauthIssuer, "/") + issuerMetaPath,
			"flows": map[string]any{
				"authorizationCode": map[string]any{
					"authorizationUrl": m.authorizationURL,
					"tokenUrl":         m.tokenURL,
					"scopes":           scopes,
				},
			},
		},
	}
	card["security"] = []map[string][]string{{"oauth": {}}}
}
