package effects

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

const headerContentType = "Content-Type"

// init registers Net effect operations
func init() {
	RegisterOp("Net", "httpGet", netHTTPGet)
	RegisterOp("Net", "httpPost", netHTTPPost)
	RegisterOp("Net", "httpRequest", NetHTTPRequest)
	RegisterOp("Net", "httpRequestBytes", NetHTTPRequestBytes)
}

// netHttpGet implements Net.httpGet(url: String) -> String
//
// Deprecated: Prefer httpRequest for access to status codes, headers, and structured errors.
// This function will be removed in v0.4.0.
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

	// Step 1: Parse and authorize the URL (scheme, host, allowlist, literal
	// IP) — the same authorizer runs again inside the RoundTripper on every
	// hop, so a redirect cannot reach what the first request could not.
	pol := netPolicy(ctx)
	u, err := url.Parse(urlStr.Value)
	if err != nil {
		return nil, fmt.Errorf("E_NET_INVALID_URL: %w", err)
	}
	if err := pol.authorizeURLLexical(u); err != nil {
		return nil, err
	}

	// Step 2: Build HTTP client with security config. Route selection (direct
	// IP-pinned vs proxied) and target resolution+validation happen inside the
	// request-aware RoundTripper, once per round trip (see net_proxy.go).
	client := &http.Client{
		Timeout:       ctx.Net.Timeout,
		CheckRedirect: pol.checkRedirect(u.Host),
		Transport:     newNetRoundTripper(ctx),
	}

	// Step 3: Make request with proper headers
	req, err := http.NewRequestWithContext(requestContext(ctx), "GET", urlStr.Value, nil)
	if err != nil {
		return nil, fmt.Errorf("E_NET_REQUEST_FAILED: %w", err)
	}
	req.Header.Set("User-Agent", ctx.Net.UserAgent)
	req.Host = u.Host // Set Host header to original hostname (for virtual hosting)

	resp, err := client.Do(req)
	if err != nil {
		if orig := unwrapTargetValidation(err); orig != nil {
			return nil, orig
		}
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
// Deprecated: Prefer httpRequest for access to status codes, headers, and structured errors.
// This function will be removed in v0.4.0.
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

	// Step 1: Parse and authorize the URL; re-authorized per hop in the
	// RoundTripper (net_proxy.go).
	pol := netPolicy(ctx)
	u, err := url.Parse(urlStr.Value)
	if err != nil {
		return nil, fmt.Errorf("E_NET_INVALID_URL: %w", err)
	}
	if err := pol.authorizeURLLexical(u); err != nil {
		return nil, err
	}

	// Step 2: Build HTTP client with security config (see net_proxy.go for
	// direct/proxy routing and once-per-round-trip target resolution).
	client := &http.Client{
		Timeout:       ctx.Net.Timeout,
		CheckRedirect: pol.checkRedirect(u.Host),
		Transport:     newNetRoundTripper(ctx),
	}

	// Step 3: Make POST request
	req, err := http.NewRequestWithContext(requestContext(ctx), "POST", urlStr.Value, strings.NewReader(bodyStr.Value))
	if err != nil {
		return nil, fmt.Errorf("E_NET_REQUEST_FAILED: %w", err)
	}
	req.Header.Set("User-Agent", ctx.Net.UserAgent)
	req.Header.Set(headerContentType, "application/json") // Default to JSON
	req.Host = u.Host

	resp, err := client.Do(req)
	if err != nil {
		if orig := unwrapTargetValidation(err); orig != nil {
			return nil, orig
		}
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

// NetHTTPRequest implements Net.httpRequest(method, url, headers, body) -> Result[HttpResponse, NetError]
//
// Advanced HTTP client with custom headers, status codes, and structured error handling.
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD)
//   - url: Target URL (must pass allowlist/security validation)
//   - headers: List of {name, value} records
//   - body: Request body (empty string for GET)
//
// Returns:
//   - Ok(HttpResponse) on success (includes 4xx/5xx status codes)
//   - Err(NetError) for transport/validation failures
//
// Security:
//   - Blocks hop-by-hop headers (Connection, Transfer-Encoding, etc.)
//   - Blocks Host, Accept-Encoding, Content-Length overrides
//   - Strips Authorization on cross-origin redirects
//   - Case-insensitive header matching, preserves order
//   - Method whitelist (GET, POST, PUT, PATCH, DELETE, HEAD)
//
// Example AILANG code:
//
//	let headers = [{name: "Authorization", value: "Bearer token"}];
//	match httpRequest("POST", url, headers, body) {
//	  Ok(resp) -> if resp.ok then resp.body else "Error: " ++ show(resp.status)
//	  Err(err) -> match err {
//	    Transport(msg) -> "Network error: " ++ msg
//	    DisallowedHost(host) -> "Blocked: " ++ host
//	    InvalidHeader(hdr) -> "Bad header: " ++ hdr
//	    BodyTooLarge(size) -> "Response too large: " ++ show(size)
//	  }
//	}
//
// NetHTTPRequest implements the _net_httpRequest builtin
// Exported for use in new builtin registry (internal/builtins/register.go)
func NetHTTPRequest(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Step 0: Capability check
	if !ctx.HasCap("Net") {
		return nil, NewCapabilityError("Net")
	}

	// Step 1: Parse arguments
	if len(args) != 4 {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequest: expected 4 arguments (method, url, headers, body), got %d", len(args))
	}

	methodVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequest: method must be String, got %T", args[0])
	}
	urlVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequest: url must be String, got %T", args[1])
	}
	headersList, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequest: headers must be List, got %T", args[2])
	}
	bodyVal, ok := args[3].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequest: body must be String, got %T", args[3])
	}

	var reqBody io.Reader
	if bodyVal.Value != "" {
		reqBody = strings.NewReader(bodyVal.Value)
	}

	client, req, errVal := buildSecureRequest(ctx, methodVal.Value, urlVal.Value, headersList, reqBody)
	if errVal != nil {
		return errVal, nil
	}

	return executeAndBuildResponse(ctx, client, req)
}

// NetHTTPRequestBytes implements Net.httpRequestBytes(method, url, headers, body: bytes)
//
// Like NetHTTPRequest, but the request body is raw bytes. Defaults Content-Type
// to "application/octet-stream" if the caller doesn't supply one. Sets an explicit
// Content-Length to suppress chunked transfer encoding (binary upload servers
// often require a fixed length).
//
// Use cases: PUT/POST raw image/PDF/binary uploads, S3-style object uploads,
// gRPC-style binary protobufs, multipart forms (caller assembles the bytes).
//
// All security checks (DNS rebinding, IP block, redirect validation, header
// validation, response size limit) are identical to NetHTTPRequest — the only
// difference is the request body type.
//
// Example AILANG code:
//
//	import std/fs (readFileBytes)
//	import std/bytes (fromBase64)
//	let Ok(b64) = readFileBytes("photo.png")
//	let Some(bytes) = fromBase64(b64)
//	httpRequestBytes("PUT", uploadUrl, [], bytes)
func NetHTTPRequestBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Capability check
	if !ctx.HasCap("Net") {
		return nil, NewCapabilityError("Net")
	}

	if len(args) != 4 {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequestBytes: expected 4 arguments (method, url, headers, body), got %d", len(args))
	}

	methodVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequestBytes: method must be String, got %T", args[0])
	}
	urlVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequestBytes: url must be String, got %T", args[1])
	}
	headersList, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequestBytes: headers must be List, got %T", args[2])
	}
	bodyVal, ok := args[3].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequestBytes: body must be Bytes, got %T", args[3])
	}

	var reqBody io.Reader
	if len(bodyVal.Value) > 0 {
		reqBody = bytes.NewReader(bodyVal.Value)
	}

	client, req, errVal := buildSecureRequest(ctx, methodVal.Value, urlVal.Value, headersList, reqBody)
	if errVal != nil {
		return errVal, nil
	}

	// Default Content-Type to application/octet-stream if caller didn't set one.
	// (buildSecureRequest applies user headers before we get here.)
	if req.Header.Get(headerContentType) == "" {
		req.Header.Set(headerContentType, "application/octet-stream")
	}
	// Set explicit Content-Length to suppress chunked encoding —
	// many binary-upload servers (S3, LinkedIn images, etc.) require a fixed length.
	req.ContentLength = int64(len(bodyVal.Value))

	return executeAndBuildResponse(ctx, client, req)
}

// buildSecureRequest validates inputs and constructs a secured (*http.Client, *http.Request)
// pair ready to execute. It runs all of the network capability validation steps
// (method whitelist, URL parse, protocol/domain/IP validation, header parsing+validation)
// and applies User-Agent, Host, and user-supplied headers to the request.
//
// Caller is responsible for: capability check (HasCap("Net")), argument unpacking,
// and supplying the request body as an io.Reader (or nil for empty body).
//
// Returns:
//   - (client, req, nil) on success — request is ready for client.Do(req)
//   - (nil, nil, errVal) for any validation failure — errVal is a Result.Err to return
func buildSecureRequest(
	ctx *EffContext,
	rawMethod, urlStr string,
	headersList *eval.ListValue,
	body io.Reader,
) (*http.Client, *http.Request, eval.Value) {
	method := strings.ToUpper(rawMethod)

	// Validate HTTP method (whitelist)
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" && method != "HEAD" {
		return nil, nil, makeResultErr("InvalidMethod", fmt.Sprintf("unsupported HTTP method: %s (supported: GET, POST, PUT, PATCH, DELETE, HEAD)", method))
	}

	// Parse and authorize the URL. The allowlist refusal keeps its own Result
	// variant (DisallowedHost); every other refusal is Transport.
	pol := netPolicy(ctx)
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, nil, makeResultErr("Transport", fmt.Sprintf("invalid URL: %v", err))
	}
	if err := pol.authorizeURLLexical(u); err != nil {
		if strings.HasPrefix(err.Error(), "E_NET_DOMAIN_BLOCKED") {
			return nil, nil, makeResultErr("DisallowedHost", u.Hostname())
		}
		return nil, nil, makeResultErr("Transport", err.Error())
	}

	// NB: DNS resolution + IP validation is NOT done here as preflight. It
	// happens exactly once per direct round trip inside the request-aware
	// RoundTripper before dialing (see net_proxy.go), which also re-runs the
	// authorizer on every redirect hop. Proxied round trips perform no local
	// target resolution at all.

	// Parse and validate headers
	userHeaders, err := parseHeaders(headersList)
	if err != nil {
		return nil, nil, makeResultErr("InvalidHeader", err.Error())
	}

	// Build HTTP client with security config (direct/proxy routing lives in the
	// request-aware RoundTripper, net_proxy.go).
	// checkRedirect re-authorizes every hop and strips the sensitive headers
	// (Authorization, Cookie, Proxy-Authorization + operator-designated) when
	// the origin changes.
	client := &http.Client{
		Timeout:       ctx.Net.Timeout,
		CheckRedirect: pol.checkRedirect(u.Host),
		Transport:     newNetRoundTripper(ctx),
	}

	// Build request
	req, err := http.NewRequestWithContext(requestContext(ctx), method, urlStr, body)
	if err != nil {
		return nil, nil, makeResultErr("Transport", fmt.Sprintf("request creation failed: %v", err))
	}

	// Set User-Agent and Host (for virtual hosting)
	req.Header.Set("User-Agent", ctx.Net.UserAgent)
	req.Host = u.Host
	// Let Go handle Accept-Encoding for transparent gzip decompression
	// (Don't allow user override)

	// Apply user headers (with name validation)
	for _, hdr := range userHeaders {
		if err := validateHeaderName(hdr.Name); err != nil {
			return nil, nil, makeResultErr("InvalidHeader", err.Error())
		}
		req.Header.Set(hdr.Name, hdr.Value)
	}

	return client, req, nil
}

// executeAndBuildResponse runs the prepared request, reads the response body
// (capped at MaxBytes), and constructs the AILANG HttpResponse record wrapped
// in Result.Ok. Returns Result.Err for transport or size-limit failures.
func executeAndBuildResponse(ctx *EffContext, client *http.Client, req *http.Request) (eval.Value, error) {
	resp, err := client.Do(req)
	if err != nil {
		// transportMessage unwraps the typed target-validation error out of the
		// *url.Error wrapper to preserve the original E_NET_DNS_FAILED /
		// E_NET_IP_BLOCKED category text from the direct route.
		return makeResultErr("Transport", transportMessage(err)), nil
	}
	defer resp.Body.Close()

	// Read body with size limit
	limitedReader := io.LimitReader(resp.Body, ctx.Net.MaxBytes)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return makeResultErr("Transport", fmt.Sprintf("failed to read response: %v", err)), nil
	}

	// Check if body was truncated (exceeded size limit)
	if int64(len(respBody)) == ctx.Net.MaxBytes {
		oneByte := make([]byte, 1)
		if n, _ := resp.Body.Read(oneByte); n > 0 {
			return makeResultErr("BodyTooLarge", fmt.Sprintf("%d", ctx.Net.MaxBytes)), nil
		}
	}

	// Build HttpResponse record
	// body is the UTF-8 view (existing field, may be lossy for binary payloads).
	// bodyBytes is the raw payload — always populated, even for binary content.
	httpResp := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"status":    &eval.IntValue{Value: resp.StatusCode},
			"headers":   makeHeadersList(resp.Header),
			"body":      &eval.StringValue{Value: string(respBody)},
			"bodyBytes": &eval.BytesValue{Value: respBody},
			"ok":        &eval.BoolValue{Value: resp.StatusCode >= 200 && resp.StatusCode < 300},
		},
	}

	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{httpResp},
	}, nil
}

// parseHeaders extracts {name, value} records from a List
func parseHeaders(headersList *eval.ListValue) ([]httpHeader, error) {
	var headers []httpHeader
	for i, elem := range headersList.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("header at index %d must be a record, got %T", i, elem)
		}

		nameVal, ok := rec.Fields["name"]
		if !ok {
			return nil, fmt.Errorf("header at index %d missing 'name' field", i)
		}
		nameStr, ok := nameVal.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("header at index %d 'name' must be String, got %T", i, nameVal)
		}

		valueVal, ok := rec.Fields["value"]
		if !ok {
			return nil, fmt.Errorf("header at index %d missing 'value' field", i)
		}
		valueStr, ok := valueVal.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("header at index %d 'value' must be String, got %T", i, valueVal)
		}

		headers = append(headers, httpHeader{
			Name:  nameStr.Value,
			Value: valueStr.Value,
		})
	}
	return headers, nil
}

// httpHeader represents a single HTTP header
type httpHeader struct {
	Name  string
	Value string
}

// validateHeaderName checks if a header name is allowed
func validateHeaderName(name string) error {
	lowerName := strings.ToLower(name)

	// Block hop-by-hop headers (per HTTP/1.1 spec)
	blocked := []string{
		"connection",
		"proxy-connection",
		"keep-alive",
		"transfer-encoding",
		"upgrade",
		"trailer",
		"te",
	}
	for _, b := range blocked {
		if lowerName == b {
			return fmt.Errorf("hop-by-hop header not allowed: %s", name)
		}
	}

	// Block headers we control
	switch lowerName {
	case "host":
		return fmt.Errorf("host header override not allowed (SSRF prevention)")
	case "accept-encoding":
		return fmt.Errorf("Accept-Encoding is managed automatically")
	case "content-length":
		return fmt.Errorf("Content-Length is computed automatically")
	}

	return nil
}

// makeHeadersList converts http.Header to AILANG List[{name, value}]
func makeHeadersList(headers http.Header) *eval.ListValue {
	var headerRecords []eval.Value
	for name, values := range headers {
		// Preserve all values (including multiple Set-Cookie, etc.)
		for _, value := range values {
			headerRecords = append(headerRecords, &eval.RecordValue{
				Fields: map[string]eval.Value{
					"name":  &eval.StringValue{Value: name},
					"value": &eval.StringValue{Value: value},
				},
			})
		}
	}
	return &eval.ListValue{Elements: headerRecords}
}

// makeNetError constructs a NetError ADT value
func makeNetError(ctorName, message string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/net",
		TypeName:   "NetError",
		CtorName:   ctorName,
		Fields:     []eval.Value{&eval.StringValue{Value: message}},
	}
}

// makeResultErr wraps a NetError in Result's Err constructor
func makeResultErr(ctorName, message string) eval.Value {
	netErr := makeNetError(ctorName, message)
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{netErr},
	}
}
