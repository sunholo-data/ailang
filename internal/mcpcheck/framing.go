package mcpcheck

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// checkAuthorizationFraming judges only a final successful HTML page. Dummy
// clients can be rejected before consent, which cannot establish protection.
func (s *session) checkAuthorizationFraming(ctx context.Context, endpoint string) Finding {
	f := Finding{Check: "authorization-framing", Status: Warn, Source: "RFC 9700 §4.16"}
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
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", "ailang-mcp-check-dummy")
	q.Set("redirect_uri", "https://example.invalid/ailang-mcp-check/callback")
	q.Set("state", "ailang-mcp-check-dummy-state")
	challenge := sha256.Sum256([]byte("ailang-mcp-check-dummy-verifier-01234567890123456789"))
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		f.Message = fmt.Sprintf("cannot construct framing probe: %v", err)
		return f
	}
	req.Header.Set("Accept", "text/html")
	resp, err := s.opts.Client.Do(req)
	if err != nil {
		f.Message = fmt.Sprintf("authorization framing probe inconclusive: %v", err)
		return f
	}
	defer resp.Body.Close()
	media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !strings.EqualFold(media, "text/html") {
		f.Message = fmt.Sprintf("authorization framing probe inconclusive: final HTTP %d, Content-Type %q (requires 2xx HTML)", resp.StatusCode, resp.Header.Get("Content-Type"))
		return f
	}
	protected := false
	for _, value := range resp.Header.Values("X-Frame-Options") {
		if strings.EqualFold(strings.TrimSpace(value), "DENY") || strings.EqualFold(strings.TrimSpace(value), "SAMEORIGIN") {
			protected = true
		}
	}
	for _, policy := range resp.Header.Values("Content-Security-Policy") {
		for _, directive := range strings.Split(policy, ";") {
			fields := strings.Fields(directive)
			if len(fields) > 1 && strings.EqualFold(fields[0], "frame-ancestors") {
				protected = true
			}
		}
	}
	if protected {
		f.Status = Pass
		f.Message = "final authorization HTML has X-Frame-Options or CSP frame-ancestors"
		return f
	}
	if s.opts.Target != "openai" {
		f.Status = Fail
	}
	f.Message = "final authorization HTML lacks valid X-Frame-Options and CSP frame-ancestors; consent pages can be framed"
	return f
}
