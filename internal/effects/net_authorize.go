package effects

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// M-EXECUTOR-POLICY-HARDENING M2 (D2) — ONE destination authorizer.
//
// Net and Stream used to carry two copies of the hostname/protocol/IP rules
// (isAllowedDomain vs isStreamAllowedDomain, validateIP vs isPrivateIP), the
// Net copy was applied once before the first request and never per hop, and
// the SSE/NDJSON transports applied neither per hop nor on the dial. N1 from
// the audit: with only allowed.example listed, a redirect to denied.example
// was followed.
//
// destinationPolicy is built from a NetContext or a StreamContext and is
// consulted at EVERY round trip (netProxyRoundTripper.RoundTrip, which the
// redirect-following client calls per hop) and at every connection
// (resolvePinned → the dialer gets the validated address, never the name).

// destinationPolicy is the resolved destination rule set for one effect.
type destinationPolicy struct {
	// kind selects the error-code family: "NET" or "STREAM".
	kind string
	// secureSchemes are always allowed; insecureSchemes need allowHTTP.
	secureSchemes   []string
	insecureSchemes []string

	allowHTTP      bool
	allowLocalhost bool
	allowMetadata  bool
	// blockPrivate: RFC1918/ULA/link-local/multicast/unspecified are refused.
	// Net has no override; Stream exposes BlockPrivateIPs (default true).
	blockPrivate   bool
	allowedDomains []string
	// refuseProxy (restricted mode, M3): a configured proxy is a named
	// refusal, never a direct fallback.
	refuseProxy bool
	// sensitiveHeaders are stripped on a cross-origin redirect.
	sensitiveHeaders []string
	// connectTimeout bounds dial + TLS + response headers (Stream); zero
	// leaves the transport unbounded (Net bounds the whole request instead).
	connectTimeout time.Duration
	maxRedirects   int

	lookupIP        func(hostname string) ([]net.IP, error)
	dialContext     func(ctx context.Context, network, addr string) (net.Conn, error)
	proxySelector   func(req *http.Request) (*url.URL, error)
	tlsClientConfig *tls.Config
}

// defaultSensitiveHeaders never survive a cross-origin redirect.
var defaultSensitiveHeaders = []string{"Authorization", "Cookie", "Proxy-Authorization"}

// netPolicy derives the policy from ctx.Net.
func netPolicy(ctx *EffContext) destinationPolicy {
	n := ctx.Net
	if n == nil {
		n = NewNetContext()
	}
	return destinationPolicy{
		kind:             "NET",
		secureSchemes:    []string{"https"},
		insecureSchemes:  []string{"http"},
		allowHTTP:        n.AllowHTTP,
		allowLocalhost:   n.AllowLocalhost,
		allowMetadata:    n.AllowMetadata,
		blockPrivate:     true,
		allowedDomains:   n.AllowedDomains,
		refuseProxy:      n.RefuseProxy,
		sensitiveHeaders: append(append([]string{}, defaultSensitiveHeaders...), n.SensitiveHeaders...),
		maxRedirects:     n.MaxRedirects,
		lookupIP:         n.lookupIP,
		dialContext:      n.dialContext,
		proxySelector:    n.proxySelector,
		tlsClientConfig:  n.tlsClientConfig,
	}
}

// streamPolicy derives the policy from ctx.Stream. Stream transports never
// used a proxy (bare http.Transport, Proxy nil); that stays explicit here.
func streamPolicy(ctx *EffContext) destinationPolicy {
	s := ctx.Stream
	if s == nil {
		s = NewStreamContext()
	}
	return destinationPolicy{
		kind:             "STREAM",
		secureSchemes:    []string{"wss", "https"},
		insecureSchemes:  []string{"ws", "http"},
		allowHTTP:        s.AllowHTTP,
		allowLocalhost:   s.AllowLocalhost,
		allowMetadata:    false,
		blockPrivate:     s.BlockPrivateIPs,
		allowedDomains:   s.AllowedDomains,
		refuseProxy:      false,
		sensitiveHeaders: append([]string{}, defaultSensitiveHeaders...),
		connectTimeout:   s.ConnectTimeout,
		maxRedirects:     5,
		lookupIP:         s.lookupIP,
		dialContext:      s.dialContext,
		proxySelector:    func(*http.Request) (*url.URL, error) { return nil, nil },
		tlsClientConfig:  s.tlsClientConfig,
	}
}

func (p destinationPolicy) errf(code, format string, a ...any) error {
	return fmt.Errorf("E_%s_%s: %s", p.kind, code, fmt.Sprintf(format, a...))
}

// authorizeURL is the per-hop gate the RoundTripper applies: scheme, host
// shape, allowlist, and (for a literal IP) the IP rules. It resolves nothing.
func (p destinationPolicy) authorizeURL(u *url.URL) error {
	if err := p.authorizeURLLexical(u); err != nil {
		return err
	}
	if ip := net.ParseIP(literalHost(u.Hostname())); ip != nil {
		if err := p.validateIP(ip); err != nil {
			return err
		}
	}
	return nil
}

// authorizeURLLexical is the preflight the entrypoints run before building a
// request: everything authorizeURL checks EXCEPT the literal-IP rule, which
// is left to the round trip so that both the direct and the proxy route are
// proven to apply it (net_proxy_target_validation_test.go).
func (p destinationPolicy) authorizeURLLexical(u *url.URL) error {
	if u == nil {
		return p.errf("INVALID_URL", "nil URL")
	}
	if err := p.authorizeScheme(u.Scheme); err != nil {
		return err
	}
	if u.User != nil {
		return p.errf("INVALID_URL", "userinfo in URL is not supported")
	}
	host := u.Hostname()
	if host == "" {
		return p.errf("INVALID_URL", "missing hostname")
	}
	if !isAllowedDomain(host, p.allowedDomains) {
		if p.kind == "STREAM" {
			return p.errf("DISALLOWED_HOST", "domain not in allowlist: %s", host)
		}
		return p.errf("DOMAIN_BLOCKED", "domain not in allowlist: %s", host)
	}
	if !p.allowLocalhost && isLocalhost(host) {
		if p.kind == "STREAM" {
			return p.errf("DISALLOWED_HOST", "localhost connections not allowed")
		}
		return p.errf("IP_BLOCKED", "localhost IP blocked: %s (use --net-allow-localhost to enable)", host)
	}
	return nil
}

// authorizeScheme applies the scheme rule with the error text each effect's
// callers and tests already know.
func (p destinationPolicy) authorizeScheme(scheme string) error {
	for _, s := range p.secureSchemes {
		if scheme == s {
			return nil
		}
	}
	for _, s := range p.insecureSchemes {
		if scheme == s {
			if p.allowHTTP {
				return nil
			}
			if p.kind == "STREAM" {
				return p.errf("PROTOCOL_ERROR", "insecure protocol %q not allowed (use wss:// or set --stream-allow-http)", scheme)
			}
			return p.errf("PROTOCOL_BLOCKED", "http:// blocked (use --net-allow-http to enable)")
		}
	}
	if p.kind == "STREAM" {
		return p.errf("PROTOCOL_ERROR", "unsupported protocol %q (expected wss:// or ws://)", scheme)
	}
	switch scheme {
	case "file", "ftp", "data", "gopher", "":
		return p.errf("PROTOCOL_BLOCKED", "unsupported protocol: %s", scheme)
	default:
		return p.errf("PROTOCOL_BLOCKED", "unknown protocol: %s", scheme)
	}
}

// validateIP is the one IP rule set: loopback needs allowLocalhost; private,
// unique-local, link-local (except the metadata server with allowMetadata),
// unspecified and multicast are refused when blockPrivate.
func (p destinationPolicy) validateIP(ip net.IP) error {
	code := "IP_BLOCKED"
	if p.kind == "STREAM" {
		code = "DISALLOWED_HOST"
	}
	if ip.IsLoopback() {
		if !p.allowLocalhost {
			return p.errf(code, "localhost IP blocked: %s (use --net-allow-localhost to enable)", ip)
		}
		return nil
	}
	if !p.blockPrivate {
		return nil
	}
	if ip.IsPrivate() {
		return p.errf(code, "private IP blocked: %s (no override available)", ip)
	}
	if ip.IsLinkLocalUnicast() {
		if p.allowMetadata && ip.Equal(net.IPv4(169, 254, 169, 254)) {
			return nil
		}
		return p.errf(code, "link-local IP blocked: %s (use --net-allow-metadata for cloud metadata server)", ip)
	}
	if ip.IsUnspecified() {
		return p.errf(code, "unspecified IP blocked: %s", ip)
	}
	if ip.IsMulticast() {
		return p.errf(code, "multicast IP blocked: %s", ip)
	}
	return nil
}

// resolvePinned resolves hostname once, validates EVERY candidate, and
// returns the one address the dialer may use. A literal IP skips DNS.
func (p destinationPolicy) resolvePinned(hostname string) (string, error) {
	if ip := net.ParseIP(literalHost(hostname)); ip != nil {
		if err := p.validateIP(ip); err != nil {
			return "", err
		}
		return literalHost(hostname), nil
	}
	var ips []net.IP
	var err error
	if p.lookupIP != nil {
		ips, err = p.lookupIP(hostname)
	} else {
		ips, err = net.LookupIP(hostname)
	}
	if err != nil {
		return "", p.errf("DNS_FAILED", "%v", err)
	}
	if len(ips) == 0 {
		return "", p.errf("DNS_FAILED", "no IPs found for %s", hostname)
	}
	for _, ip := range ips {
		if err := p.validateIP(ip); err != nil {
			return "", p.errf("DNS_REBINDING", "%s resolves to blocked IP: %v", hostname, err)
		}
	}
	return ips[0].String(), nil
}

// dial dispatches to the injected dialer when configured (tests), else to
// an ordinary net.Dialer bounded by connectTimeout.
func (p destinationPolicy) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if p.dialContext != nil {
		return p.dialContext(ctx, network, addr)
	}
	return (&net.Dialer{Timeout: p.connectTimeout}).DialContext(ctx, network, addr)
}

// selectProxy returns the proxy for req: the injected selector, else
// http.ProxyFromEnvironment. Under refuseProxy a selected proxy is an error.
func (p destinationPolicy) selectProxy(req *http.Request) (*url.URL, error) {
	var proxyURL *url.URL
	var err error
	if p.proxySelector != nil {
		proxyURL, err = p.proxySelector(req)
	} else {
		proxyURL, err = http.ProxyFromEnvironment(req)
	}
	if err != nil {
		return nil, err
	}
	if proxyURL != nil && p.refuseProxy {
		return nil, p.errf("PROXY_REFUSED", "restricted mode cannot pin the destination address behind a proxy (%s); unset the proxy or run in trusted_host mode", proxyURL.Redacted())
	}
	return proxyURL, nil
}

// checkRedirect is the http.Client.CheckRedirect every entrypoint installs:
// hop limit, re-authorization of the destination BEFORE any dial, and
// sensitive-header stripping when the origin changes.
func (p destinationPolicy) checkRedirect(originalHost string) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= p.maxRedirects {
			return p.errf("TOO_MANY_REDIRECTS", "exceeded max redirects (%d)", p.maxRedirects)
		}
		if err := p.authorizeURL(req.URL); err != nil {
			// Typed so callers unwrap the E_* category out of the url.Error
			// http.Client wraps around a CheckRedirect refusal.
			return &targetValidationError{cause: err}
		}
		if !strings.EqualFold(req.URL.Host, originalHost) {
			for _, h := range p.sensitiveHeaders {
				req.Header.Del(h)
			}
		}
		return nil
	}
}

// requestContext is the Go context a request runs under: the effect's GoCtx
// when the runner set one (cancellation propagates), else Background.
func requestContext(ctx *EffContext) context.Context {
	if ctx != nil && ctx.GoCtx != nil {
		return ctx.GoCtx
	}
	return context.Background()
}

// isAllowedDomain: empty allowlist = every domain; otherwise exact or
// label-boundary wildcard match after lowercase + trailing-dot normalisation
// of BOTH sides.
func isAllowedDomain(hostname string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	hostname = normalizeHost(hostname)
	for _, pattern := range allowed {
		if matchDomain(hostname, normalizeHost(pattern)) {
			return true
		}
	}
	return false
}

func normalizeHost(h string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(h), "."))
}

// matchDomain: exact, or `*.example.com` matching any host that ends in
// `.example.com` (label boundary — evil-example.com does not match, nor does
// the apex example.com).
func matchDomain(hostname, pattern string) bool {
	if hostname == pattern {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		return len(hostname) > len(suffix) && strings.HasSuffix(hostname, suffix)
	}
	return false
}

// isLocalhost reports whether a hostname names the loopback host by NAME or
// as a literal; resolved loopback addresses are caught by validateIP.
func isLocalhost(hostname string) bool {
	hostname = strings.ToLower(hostname)
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return true
	}
	return strings.HasPrefix(hostname, "127.")
}

// validateIP is the Net IP rule (kept as a plain function: the IP-policy
// tests exercise it directly).
func validateIP(ip net.IP, ctx *EffContext) error {
	return netPolicy(ctx).validateIP(ip)
}
