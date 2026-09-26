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
// request-aware RoundTripper when destination authorization, target
// resolution or IP validation fails (before any dial is attempted). It
// carries the original E_NET_* / E_STREAM_* category so that public callers
// can surface the stable category even after http.Client.Do wraps it in a
// *url.Error. Production code in this package never treats it as a user
// visible error type — it is unwrapped through url.Error instead.
type targetValidationError struct {
	cause error
}

func (e *targetValidationError) Error() string { return e.cause.Error() }
func (e *targetValidationError) Unwrap() error { return e.cause }

// netProxyRoundTripper is the package-private, request-aware RoundTripper
// every Net and Stream HTTP request goes through. http.Client calls
// RoundTrip once per hop, so the destination policy is applied to EVERY
// hop — the initial request and each redirect — before anything is dialed
// (M-EXECUTOR-POLICY-HARDENING M2, N1).
//
// Security contract:
//   - every round trip: pol.authorizeURL (scheme, host, allowlist, literal IP)
//     runs first; a refusal is returned as targetValidationError with no dial.
//   - no proxy selected: resolvePinned is called exactly once, and the
//     returned IP is handed to a direct transport whose dialer connects to
//     that IP with no hostname re-resolution (anti-DNS-rebinding pinning).
//   - proxy selected: literal target IPs are validated without DNS before
//     ordinary proxy dialing; hostnames receive zero local target resolution.
//     The proxy address never enters any target-IP substitution closure.
//     Under pol.refuseProxy a selected proxy is a named refusal, never a
//     direct fallback.
//
// It owns separate transport creation paths for the two modes and never
// mutates one shared transport between them. Each round trip builds a fresh
// transport so no pin or route decision can bleed across requests.
type netProxyRoundTripper struct {
	pol destinationPolicy
}

// newNetRoundTripper is the Net transport for ctx.
func newNetRoundTripper(ctx *EffContext) *netProxyRoundTripper {
	return &netProxyRoundTripper{pol: netPolicy(ctx)}
}

// newStreamRoundTripper is the Stream (SSE/NDJSON) transport for ctx.
func newStreamRoundTripper(ctx *EffContext) *netProxyRoundTripper {
	return &netProxyRoundTripper{pol: streamPolicy(ctx)}
}

// RoundTrip implements http.RoundTripper.
func (rt *netProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Route selection is a pure decision (no dial); it comes first so the
	// proxy-route tests can see the route was chosen, and so a proxy that is
	// refused is named before the destination is judged.
	proxyURL, err := rt.pol.selectProxy(req)
	if err != nil {
		return nil, &targetValidationError{cause: err}
	}
	if err := rt.pol.authorizeURL(req.URL); err != nil {
		return nil, &targetValidationError{cause: err}
	}
	if proxyURL != nil {
		return rt.proxyRoundTrip(req, proxyURL)
	}
	return rt.directRoundTrip(req)
}

// directRoundTrip resolves+validates the target exactly once and dials the
// validated IP with no hostname re-resolution.
func (rt *netProxyRoundTripper) directRoundTrip(req *http.Request) (*http.Response, error) {
	validatedIP, err := rt.pol.resolvePinned(req.URL.Hostname())
	if err != nil {
		return nil, &targetValidationError{cause: err}
	}
	tr := rt.directTransport(validatedIP, req.URL)
	// A per-request transport has no shared keep-alive pool to preserve; close
	// idle conns so the response body read is unaffected and nothing lingers.
	// (For a streaming body the response holds its own connection; closing
	// idle ones does not touch it.)
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

// proxyRoundTrip validates literal target IPs without DNS (authorizeURL
// already did), then performs ordinary proxy dialing. Hostname targets
// receive no local resolution.
func (rt *netProxyRoundTripper) proxyRoundTrip(req *http.Request, proxyURL *url.URL) (*http.Response, error) {
	tr := rt.proxyTransport(proxyURL)
	defer tr.CloseIdleConnections()
	return tr.RoundTrip(req)
}

// directTransport builds a transport whose dialer replaces the requested dial
// host with the pre-validated target IP. It never uses a proxy. The URL's
// hostname stays on the request, so TLS verification/SNI and the Host
// header use the name, not the pinned address.
func (rt *netProxyRoundTripper) directTransport(validatedIP string, u *url.URL) *http.Transport {
	return &http.Transport{
		// nil Proxy: this route was already selected as the direct path.
		DialContext: func(ctxDial context.Context, network, addr string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(addr)
			if port == "" {
				port = defaultPort(u.Scheme)
			}
			dialAddr := net.JoinHostPort(validatedIP, port)
			return rt.pol.dial(ctxDial, network, dialAddr)
		},
		TLSHandshakeTimeout:   rt.pol.connectTimeout,
		ResponseHeaderTimeout: rt.pol.connectTimeout,
		TLSClientConfig:       rt.pol.tlsClientConfig,
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
			return rt.pol.dial(ctxDial, network, addr)
		},
		TLSHandshakeTimeout:   rt.pol.connectTimeout,
		ResponseHeaderTimeout: rt.pol.connectTimeout,
		TLSClientConfig:       rt.pol.tlsClientConfig,
	}
}

// defaultPort is the port a URL scheme implies when none is given.
func defaultPort(scheme string) string {
	switch scheme {
	case "http", "ws":
		return "80"
	default:
		return "443"
	}
}

// pinnedDialer returns a dialer for the WebSocket transport (and any other
// non-http.Client consumer): the URL is authorized and resolved ONCE here,
// and the returned DialContext connects only to the pinned address for that
// URL's port, whatever host:port the caller passes.
func (p destinationPolicy) pinnedDialer(u *url.URL) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	if err := p.authorizeURL(u); err != nil {
		return nil, err
	}
	validatedIP, err := p.resolvePinned(u.Hostname())
	if err != nil {
		return nil, err
	}
	port := u.Port()
	if port == "" {
		port = defaultPort(u.Scheme)
	}
	dialAddr := net.JoinHostPort(validatedIP, port)
	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		return p.dial(ctx, network, dialAddr)
	}, nil
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
// preserving the original E_NET_* / E_STREAM_* text when the failure is our
// typed target-validation error, and the url.Error text otherwise.
func transportMessage(err error) string {
	if orig := unwrapTargetValidation(err); orig != nil {
		return orig.Error()
	}
	return err.Error()
}
