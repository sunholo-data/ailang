package effects

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// targetValidationError is a typed, internal error returned by the
// request-aware RoundTripper when target resolution or IP
// validation fails (before any dial is attempted). It carries the original
// E_NET_DNS_FAILED / E_NET_IP_BLOCKED category so that public callers can
// surface the stable legacy category even after http.Client.Do wraps it in a
// *url.Error. Production code in this package never treats it as a user
// visible error type — it is unwrapped through url.Error instead.
type targetValidationError struct {
	cause error
}

func (e *targetValidationError) Error() string { return e.cause.Error() }
func (e *targetValidationError) Unwrap() error { return e.cause }

// netProxyRoundTripper is the package-private, request-aware RoundTripper that
// routes every Net request through either a direct (IP-pinned) transport or a
// proxy transport, decided per request by http.ProxyFromEnvironment(req).
//
// Security contract:
//   - no proxy selected: resolveAndValidateIP is called exactly once, and the
//     returned IP is handed to a direct transport whose dialer connects to that
//     IP with no hostname re-resolution (anti-DNS-rebinding pinning).
//   - proxy selected: literal target IPs are validated without DNS before
//     ordinary proxy dialing; hostnames receive zero local target resolution.
//     The proxy address never enters any target-IP substitution closure.
//
// It owns separate transport creation paths for the two modes and never
// mutates one shared transport between them. Each round trip builds a fresh
// transport so no pin or route decision can bleed across requests.
type netProxyRoundTripper struct {
	ctx *EffContext
}

// RoundTrip implements http.RoundTripper.
func (rt *netProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	proxyURL, err := rt.selectProxy(req)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		return rt.proxyRoundTrip(req, proxyURL)
	}
	return rt.directRoundTrip(req)
}

// selectProxy returns the proxy to use for the request. It defaults to the
// real http.ProxyFromEnvironment; an injected test hook (ctx.Net.proxySelector)
// can override it for deterministic in-process tests.
func (rt *netProxyRoundTripper) selectProxy(req *http.Request) (*url.URL, error) {
	if rt.ctx != nil && rt.ctx.Net != nil && rt.ctx.Net.proxySelector != nil {
		return rt.ctx.Net.proxySelector(req)
	}
	return http.ProxyFromEnvironment(req)
}

// directRoundTrip resolves+validates the target exactly once and dials the
// validated IP with no hostname re-resolution.
func (rt *netProxyRoundTripper) directRoundTrip(req *http.Request) (*http.Response, error) {
	validatedIP, err := resolveAndValidateIP(req.URL.Hostname(), rt.ctx)
	if err != nil {
		return nil, &targetValidationError{cause: err}
	}
	tr := rt.directTransport(validatedIP, req.URL)
	// A per-request transport has no shared keep-alive pool to preserve; close
	// idle conns so the response body read is unaffected and nothing lingers.
	defer tr.CloseIdleConnections()
	return tr.RoundTrip(req)
}

// literalHost strips an RFC 4007 zone identifier from a URL host so that a
// zone-qualified IPv6 literal is recognised as a literal.
//
// url.URL.Hostname() returns the zone, e.g. "fe80::1%eth0" for
// http://[fe80::1%25eth0]/ — and net.ParseIP REJECTS that form, returning nil.
// Without this, a zone-qualified link-local target would fall through to the
// hostname branch and skip IP-policy validation entirely on the proxy route,
// which is the one thing D-1 exists to prevent. Trimming at the first '%' cannot
// turn a hostname into a literal: '%' is not a legal character in a DNS name, so
// any host containing one is either a zone-qualified literal or already invalid.
func literalHost(host string) string {
	if i := strings.IndexByte(host, '%'); i >= 0 {
		return host[:i]
	}
	return host
}

// proxyRoundTrip validates literal target IPs without DNS, then performs
// ordinary proxy dialing. Hostname targets receive no local resolution.
func (rt *netProxyRoundTripper) proxyRoundTrip(req *http.Request, proxyURL *url.URL) (*http.Response, error) {
	if ip := net.ParseIP(literalHost(req.URL.Hostname())); ip != nil {
		if err := validateIP(ip, rt.ctx); err != nil {
			return nil, &targetValidationError{cause: err}
		}
	}
	tr := rt.proxyTransport(proxyURL)
	defer tr.CloseIdleConnections()
	return tr.RoundTrip(req)
}

// directTransport builds a transport whose dialer replaces the requested dial
// host with the pre-validated target IP. It never uses a proxy.
func (rt *netProxyRoundTripper) directTransport(validatedIP string, u *url.URL) *http.Transport {
	return &http.Transport{
		// nil Proxy: this route was already selected as the direct path.
		DialContext: func(ctxDial context.Context, network, addr string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(addr)
			if port == "" {
				port = "443"
				if u.Scheme == "http" {
					port = "80"
				}
			}
			dialAddr := net.JoinHostPort(validatedIP, port)
			return rt.dial(ctxDial, network, dialAddr)
		},
	}
}

// proxyTransport builds a transport that dials the operator-selected proxy
// using ordinary proxy semantics (CONNECT for TLS, absolute-form otherwise).
// The proxy address never enters a target-IP substitution closure.
func (rt *netProxyRoundTripper) proxyTransport(proxyURL *url.URL) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: func(ctxDial context.Context, network, addr string) (net.Conn, error) {
			// In proxy mode addr is the proxy's address (CONNECT/absolute-form
			// dial destination). No target-IP substitution is applied here.
			return rt.dial(ctxDial, network, addr)
		},
	}
}

// dial dispatches to the injected dialer hook when configured (tests), else to
// the ordinary net.Dialer.
func (rt *netProxyRoundTripper) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if rt.ctx != nil && rt.ctx.Net != nil && rt.ctx.Net.dialContext != nil {
		return rt.ctx.Net.dialContext(ctx, network, addr)
	}
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

// unwrapTargetValidation returns the original typed target-validation error if
// err (which http.Client.Do returns wrapped in a *url.Error) carries a
// targetValidationError from the direct route; otherwise it returns nil.
func unwrapTargetValidation(err error) error {
	var tve *targetValidationError
	if errors.As(err, &tve) {
		return tve.cause
	}
	return nil
}

// transportMessage returns the public message to surface for a client.Do error,
// preserving the original E_NET_DNS_FAILED / E_NET_IP_BLOCKED text when the
// failure is our typed target-validation error, and the url.Error text
// otherwise.
func transportMessage(err error) string {
	if orig := unwrapTargetValidation(err); orig != nil {
		return orig.Error()
	}
	return err.Error()
}
