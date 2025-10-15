package effects

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/sunholo/ailang/internal/eval"
)

// init registers Net effect operations
func init() {
	RegisterOp("Net", "httpGet", netHTTPGet)
	RegisterOp("Net", "httpPost", netHTTPPost)
}

// netHttpGet implements Net.httpGet(url: String) -> String
//
// Fetches content from an HTTP/HTTPS URL with comprehensive security validation.
//
// Security features (Phase 2 PM - FULL):
//   - Protocol validation (https:// enforced by default, http:// requires flag)
//   - DNS rebinding prevention (resolve → validate IPs → dial validated IP)
//   - Redirect validation (max 5 redirects, re-validate IP at each hop)
//   - Body size limits (5MB default, configurable)
//   - Timeout enforcement (30s default, configurable)
//   - Domain allowlist (optional)
//   - User-Agent header
//
// Parameters:
//   - ctx: Effect context (must have Net capability)
//   - args: [StringValue] - URL to fetch
//
// Returns:
//   - StringValue with response body
//   - Error if capability missing, URL invalid, or request fails
//
// Example AILANG code:
//
//	let html = httpGet("https://example.com")
func netHTTPGet(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Step 0: Capability check
	if !ctx.HasCap("Net") {
		return nil, NewCapabilityError("Net")
	}

	if len(args) != 1 {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpGet: expected 1 argument, got %d", len(args))
	}

	urlStr, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpGet: expected String, got %T", args[0])
	}

	// Step 1: Parse and validate URL
	u, err := url.Parse(urlStr.Value)
	if err != nil {
		return nil, fmt.Errorf("E_NET_INVALID_URL: %w", err)
	}

	// Step 2: Protocol validation
	if err := validateProtocol(u.Scheme, ctx); err != nil {
		return nil, err
	}

	// Step 3: Domain allowlist check (fail fast before DNS)
	if !isAllowedDomain(u.Hostname(), ctx.Net.AllowedDomains) {
		return nil, fmt.Errorf("E_NET_DOMAIN_BLOCKED: domain not in allowlist: %s", u.Hostname())
	}

	// Step 4: DNS resolution + IP validation (prevent DNS rebinding)
	validatedIP, err := resolveAndValidateIP(u.Hostname(), ctx)
	if err != nil {
		return nil, err
	}

	// Step 5: Build HTTP client with security config
	client := &http.Client{
		Timeout: ctx.Net.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return validateRedirect(req, via, ctx)
		},
		Transport: &http.Transport{
			// Force connection to validated IP (prevent DNS rebinding mid-request)
			DialContext: func(ctxDial context.Context, network, addr string) (net.Conn, error) {
				// Replace hostname with validated IP in dial address
				_, port, _ := net.SplitHostPort(addr)
				if port == "" {
					port = "443" // Default HTTPS port
					if u.Scheme == "http" {
						port = "80"
					}
				}
				dialAddr := net.JoinHostPort(validatedIP, port)
				return (&net.Dialer{}).DialContext(ctxDial, network, dialAddr)
			},
		},
	}

	// Step 6: Make request with proper headers
	req, err := http.NewRequest("GET", urlStr.Value, nil)
	if err != nil {
		return nil, fmt.Errorf("E_NET_REQUEST_FAILED: %w", err)
	}
	req.Header.Set("User-Agent", ctx.Net.UserAgent)
	req.Host = u.Host // Set Host header to original hostname (for virtual hosting)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("E_NET_REQUEST_FAILED: %w", err)
	}
	defer resp.Body.Close()

	// Step 7: Read body with size limit
	limitedReader := io.LimitReader(resp.Body, ctx.Net.MaxBytes)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("E_NET_READ_FAILED: %w", err)
	}

	// Check if body was truncated (exceeded size limit)
	if int64(len(body)) == ctx.Net.MaxBytes {
		// Try reading one more byte to see if there's more
		oneByte := make([]byte, 1)
		if n, _ := resp.Body.Read(oneByte); n > 0 {
			return nil, fmt.Errorf("E_NET_BODY_TOO_LARGE: response exceeds %d bytes", ctx.Net.MaxBytes)
		}
	}

	return &eval.StringValue{Value: string(body)}, nil
}

// netHttpPost implements Net.httpPost(url: String, body: String) -> String
//
// Sends an HTTP POST request with the given body.
//
// Parameters:
//   - ctx: Effect context (must have Net capability)
//   - args: [StringValue, StringValue] - URL and request body
//
// Returns:
//   - StringValue with response body
//   - Error if capability missing, URL invalid, or request fails
//
// Example AILANG code:
//
//	let response = httpPost("https://api.example.com/data", "{\"key\": \"value\"}")
func netHTTPPost(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Step 0: Capability check
	if !ctx.HasCap("Net") {
		return nil, NewCapabilityError("Net")
	}

	if len(args) != 2 {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpPost: expected 2 arguments, got %d", len(args))
	}

	urlStr, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpPost: expected String for URL, got %T", args[0])
	}

	bodyStr, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpPost: expected String for body, got %T", args[1])
	}

	// Step 1: Parse and validate URL
	u, err := url.Parse(urlStr.Value)
	if err != nil {
		return nil, fmt.Errorf("E_NET_INVALID_URL: %w", err)
	}

	// Step 2: Protocol validation
	if err := validateProtocol(u.Scheme, ctx); err != nil {
		return nil, err
	}

	// Step 3: Domain allowlist check
	if !isAllowedDomain(u.Hostname(), ctx.Net.AllowedDomains) {
		return nil, fmt.Errorf("E_NET_DOMAIN_BLOCKED: domain not in allowlist: %s", u.Hostname())
	}

	// Step 4: DNS resolution + IP validation
	validatedIP, err := resolveAndValidateIP(u.Hostname(), ctx)
	if err != nil {
		return nil, err
	}

	// Step 5: Build HTTP client with security config
	client := &http.Client{
		Timeout: ctx.Net.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return validateRedirect(req, via, ctx)
		},
		Transport: &http.Transport{
			DialContext: func(ctxDial context.Context, network, addr string) (net.Conn, error) {
				_, port, _ := net.SplitHostPort(addr)
				if port == "" {
					port = "443"
					if u.Scheme == "http" {
						port = "80"
					}
				}
				dialAddr := net.JoinHostPort(validatedIP, port)
				return (&net.Dialer{}).DialContext(ctxDial, network, dialAddr)
			},
		},
	}

	// Step 6: Make POST request
	req, err := http.NewRequest("POST", urlStr.Value, strings.NewReader(bodyStr.Value))
	if err != nil {
		return nil, fmt.Errorf("E_NET_REQUEST_FAILED: %w", err)
	}
	req.Header.Set("User-Agent", ctx.Net.UserAgent)
	req.Header.Set("Content-Type", "application/json") // Default to JSON
	req.Host = u.Host

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("E_NET_REQUEST_FAILED: %w", err)
	}
	defer resp.Body.Close()

	// Step 7: Read response with size limit
	limitedReader := io.LimitReader(resp.Body, ctx.Net.MaxBytes)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("E_NET_READ_FAILED: %w", err)
	}

	// Check size limit
	if int64(len(body)) == ctx.Net.MaxBytes {
		oneByte := make([]byte, 1)
		if n, _ := resp.Body.Read(oneByte); n > 0 {
			return nil, fmt.Errorf("E_NET_BODY_TOO_LARGE: response exceeds %d bytes", ctx.Net.MaxBytes)
		}
	}

	return &eval.StringValue{Value: string(body)}, nil
}

// validateProtocol checks if the URL scheme is allowed
//
// Security policy:
//   - https:// always allowed
//   - http:// allowed only if ctx.Net.AllowHTTP is true
//   - file://, ftp://, data://, gopher://, custom schemes blocked
//
// Parameters:
//   - scheme: The URL scheme (e.g., "https", "http", "file")
//   - ctx: Effect context (for AllowHTTP flag)
//
// Returns:
//   - nil if protocol is allowed
//   - Error with E_NET_PROTOCOL_BLOCKED if protocol is blocked
func validateProtocol(scheme string, ctx *EffContext) error {
	switch scheme {
	case "https":
		return nil // Always allowed
	case "http":
		if !ctx.Net.AllowHTTP {
			return fmt.Errorf("E_NET_PROTOCOL_BLOCKED: http:// blocked (use --net-allow-http to enable)")
		}
		return nil
	case "file", "ftp", "data", "gopher", "":
		return fmt.Errorf("E_NET_PROTOCOL_BLOCKED: unsupported protocol: %s", scheme)
	default:
		return fmt.Errorf("E_NET_PROTOCOL_BLOCKED: unknown protocol: %s", scheme)
	}
}

// validateRedirect validates each redirect in the chain
//
// Security checks:
//   - Enforce max redirect limit (default: 5)
//   - Validate protocol for each redirect destination
//   - Re-validate IP for redirect target (prevent DNS rebinding via redirect)
//
// Parameters:
//   - req: The redirect request
//   - via: Previous requests in redirect chain
//   - ctx: Effect context
//
// Returns:
//   - nil if redirect is allowed
//   - Error if too many redirects or redirect destination is blocked
func validateRedirect(req *http.Request, via []*http.Request, ctx *EffContext) error {
	// Enforce max redirects
	if len(via) >= ctx.Net.MaxRedirects {
		return fmt.Errorf("E_NET_TOO_MANY_REDIRECTS: exceeded max redirects (%d)", ctx.Net.MaxRedirects)
	}

	// Validate redirect destination protocol
	if err := validateProtocol(req.URL.Scheme, ctx); err != nil {
		return err
	}

	// Re-validate IP for redirect target (prevent DNS rebinding via redirect)
	_, err := resolveAndValidateIP(req.URL.Hostname(), ctx)
	return err
}

// isAllowedDomain checks if a hostname is in the domain allowlist
//
// Security policy:
//   - If allowlist is empty, all domains are allowed
//   - If allowlist is set, only listed domains (or wildcard matches) are allowed
//   - Supports wildcard: *.example.com matches foo.example.com
//
// Parameters:
//   - hostname: The hostname to check
//   - allowed: The domain allowlist
//
// Returns:
//   - true if domain is allowed, false otherwise
func isAllowedDomain(hostname string, allowed []string) bool {
	if len(allowed) == 0 {
		return true // No allowlist = all domains OK
	}

	// Normalize hostname (lowercase, strip trailing dot)
	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	for _, pattern := range allowed {
		if matchDomain(hostname, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// matchDomain checks if a hostname matches a domain pattern
//
// Supports:
//   - Exact match: hostname == pattern
//   - Wildcard match: *.example.com matches foo.example.com
//
// Parameters:
//   - hostname: The hostname to check
//   - pattern: The domain pattern (may include wildcard)
//
// Returns:
//   - true if hostname matches pattern
func matchDomain(hostname, pattern string) bool {
	// Exact match
	if hostname == pattern {
		return true
	}

	// Wildcard match: *.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		return strings.HasSuffix(hostname, suffix)
	}

	return false
}
