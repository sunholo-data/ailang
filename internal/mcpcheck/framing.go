package mcpcheck

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProbeClientURL is the CIMD client the framing probe presents to an AS that
// advertises client_id_metadata_document_supported. The docs site serves the
// document from docs/static/mcp-check/client-metadata.json; its client_id is
// this URL. It is a public URL we keep serving (M-MCP-OAUTH-CLIENT-MODELS D7).
const ProbeClientURL = "https://ailang.sunholo.com/mcp-check/client-metadata.json"

// ProbeRedirectURI is the probe client's one registered redirect. Loopback
// matching ignores the port (RFC 8252 §7.3), and the probe never follows a
// redirect here, so nothing needs to listen on it.
const ProbeRedirectURI = "http://127.0.0.1/ailang-mcp-check/callback"

// probeState is echoed back by the AS; unreserved characters only (V29).
const probeState = "ailang-mcp-check-dummy-state"

// errReachedCallback stops the probe before it follows a hop to its own
// redirect_uri.
var errReachedCallback = errors.New("authorization server redirected to the probe's redirect_uri")

// authServerMetadata is the part of RFC 8414 metadata the checks read.
type authServerMetadata struct {
	PKCE                  []string `json:"code_challenge_methods_supported"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	CIMD                  bool     `json:"client_id_metadata_document_supported"`
	AuthMeth              []string `json:"token_endpoint_auth_methods_supported"`
	Register              string   `json:"registration_endpoint"`
}

type probeClient struct{ id, redirect string }

// selectProbeClient picks the client the probe presents: the CIMD probe
// client when the AS offers CIMD to public clients, else the operator's
// --probe-client-id/--probe-redirect-uri, else none (skip reason returned).
// It never registers a client through DCR: that would write to the
// adopter's server.
func (s *session) selectProbeClient(md authServerMetadata) (probeClient, string) {
	switch {
	case md.CIMD && contains(md.AuthMeth, "none"):
		return probeClient{ProbeClientURL, ProbeRedirectURI}, ""
	case s.opts.ProbeClientID != "" && s.opts.ProbeRedirectURI != "":
		return probeClient{s.opts.ProbeClientID, s.opts.ProbeRedirectURI}, ""
	default:
		return probeClient{}, "no probe client: the AS offers no CIMD; pass --probe-client-id and --probe-redirect-uri for a pre-registered client"
	}
}

// newPKCEChallenge returns the S256 challenge of a fresh random verifier
// (32 bytes -> 43 unreserved characters, within RFC 7636's 43-128). The
// verifier itself is discarded: the probe never reaches a token endpoint.
func newPKCEChallenge() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	verifier := base64.RawURLEncoding.EncodeToString(raw[:])
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// sameEndpoint reports whether u is the redirect URI ref, ignoring the port
// (loopback redirects match on any port) and the query.
func sameEndpoint(u *url.URL, ref string) bool {
	r, err := url.Parse(ref)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, r.Scheme) && strings.EqualFold(u.Hostname(), r.Hostname()) && u.Path == r.Path
}

// checkAuthorizationFraming judges only a final successful HTML page reached
// with a client the AS accepts. Side effect: one pending authorization
// request on the target, which expires with the AS's request TTL.
func (s *session) checkAuthorizationFraming(ctx context.Context, md authServerMetadata) Finding {
	f := Finding{Check: "authorization-framing", Status: Warn, Source: "RFC 9700 §4.16"}
	endpoint := md.AuthorizationEndpoint
	if endpoint == "" {
		f.Status = Skip
		f.Message = "no authorization_endpoint advertised; framing probe skipped"
		return f
	}
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		f.Message = fmt.Sprintf("cannot probe authorization_endpoint %q: invalid HTTP URL", endpoint)
		return f
	}
	if !contains(md.PKCE, "S256") {
		f.Status = Skip
		f.Message = "AS does not advertise S256; framing probe skipped"
		return f
	}
	client, skip := s.selectProbeClient(md)
	if skip != "" {
		f.Status = Skip
		f.Message = skip
		return f
	}
	challenge, err := newPKCEChallenge()
	if err != nil {
		f.Message = fmt.Sprintf("cannot generate a PKCE verifier: %v", err)
		return f
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", client.id)
	q.Set("redirect_uri", client.redirect)
	q.Set("state", probeState)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	if s.opts.URL != "" {
		q.Set("resource", s.opts.URL) // RFC 8707, for servers that require it
	}
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		f.Message = fmt.Sprintf("cannot construct framing probe: %v", err)
		return f
	}
	req.Header.Set("Accept", "text/html")
	hc := *s.opts.Client
	hc.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if sameEndpoint(next.URL, client.redirect) {
			return errReachedCallback
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	resp, err := hc.Do(req)
	if errors.Is(err, errReachedCallback) {
		f.Message = fmt.Sprintf("AS redirected without a consent page (probe client %s): no page to judge", client.id)
		return f
	}
	if err != nil {
		f.Message = fmt.Sprintf("authorization framing probe inconclusive: %v", err)
		return f
	}
	defer resp.Body.Close()
	media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !strings.EqualFold(media, "text/html") {
		f.Message = fmt.Sprintf("authorization framing probe inconclusive (probe client %s): final HTTP %d, Content-Type %q (requires 2xx HTML)", client.id, resp.StatusCode, resp.Header.Get("Content-Type"))
		return f
	}
	if framingProtected(resp.Header) {
		f.Status = Pass
		f.Message = fmt.Sprintf("final authorization HTML has X-Frame-Options or CSP frame-ancestors (probe client %s)", client.id)
		return f
	}
	if s.opts.Target != "openai" {
		f.Status = Fail
	}
	f.Message = "final authorization HTML lacks valid X-Frame-Options and CSP frame-ancestors; consent pages can be framed"
	return f
}

// framingProtected: X-Frame-Options DENY/SAMEORIGIN, or an enforced CSP with
// a frame-ancestors directive (Report-Only does not count).
func framingProtected(h http.Header) bool {
	for _, value := range h.Values("X-Frame-Options") {
		if strings.EqualFold(strings.TrimSpace(value), "DENY") || strings.EqualFold(strings.TrimSpace(value), "SAMEORIGIN") {
			return true
		}
	}
	for _, policy := range h.Values("Content-Security-Policy") {
		for _, directive := range strings.Split(policy, ";") {
			fields := strings.Fields(directive)
			if len(fields) > 1 && strings.EqualFold(fields[0], "frame-ancestors") {
				return true
			}
		}
	}
	return false
}
