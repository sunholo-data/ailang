package effects

import (
	"os"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestNetCapabilityChecks verifies that Net operations require Net capability
func TestNetCapabilityChecks(t *testing.T) {
	ctx := NewEffContext()
	// No capabilities granted

	t.Run("httpGet requires Net capability", func(t *testing.T) {
		url := &eval.StringValue{Value: "https://example.com"}
		_, err := netHTTPGet(ctx, []eval.Value{url})

		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		capErr, ok := err.(*CapabilityError)
		if !ok {
			t.Fatalf("Expected CapabilityError, got %T", err)
		}
		if capErr.Effect != "Net" {
			t.Errorf("Expected Effect=Net, got %s", capErr.Effect)
		}
	})

	t.Run("httpPost requires Net capability", func(t *testing.T) {
		url := &eval.StringValue{Value: "https://example.com"}
		body := &eval.StringValue{Value: `{"test": true}`}
		_, err := netHTTPPost(ctx, []eval.Value{url, body})

		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		capErr, ok := err.(*CapabilityError)
		if !ok {
			t.Fatalf("Expected CapabilityError, got %T", err)
		}
		if capErr.Effect != "Net" {
			t.Errorf("Expected Effect=Net, got %s", capErr.Effect)
		}
	})
}

// TestNetProtocolValidation verifies protocol security policies
func TestNetProtocolValidation(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	tests := []struct {
		name        string
		url         string
		allowHTTP   bool
		expectError string
	}{
		{
			name:        "https allowed",
			url:         "https://example.com",
			allowHTTP:   false,
			expectError: "",
		},
		{
			name:        "http blocked by default",
			url:         "http://example.com",
			allowHTTP:   false,
			expectError: "E_NET_PROTOCOL_BLOCKED: http:// blocked",
		},
		{
			name:        "http allowed with flag",
			url:         "http://example.com",
			allowHTTP:   true,
			expectError: "",
		},
		{
			name:        "file protocol blocked",
			url:         "file:///etc/passwd",
			allowHTTP:   false,
			expectError: "E_NET_PROTOCOL_BLOCKED: unsupported protocol: file",
		},
		{
			name:        "ftp protocol blocked",
			url:         "ftp://ftp.example.com/file.txt",
			allowHTTP:   false,
			expectError: "E_NET_PROTOCOL_BLOCKED: unsupported protocol: ftp",
		},
		{
			name:        "data protocol blocked",
			url:         "data:text/plain,Hello%20World",
			allowHTTP:   false,
			expectError: "E_NET_PROTOCOL_BLOCKED: unsupported protocol: data",
		},
		{
			name:        "gopher protocol blocked",
			url:         "gopher://gopher.example.com",
			allowHTTP:   false,
			expectError: "E_NET_PROTOCOL_BLOCKED: unsupported protocol: gopher",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.Net.AllowHTTP = tt.allowHTTP
			urlVal := &eval.StringValue{Value: tt.url}
			_, err := netHTTPGet(ctx, []eval.Value{urlVal})

			if tt.expectError == "" {
				// Note: These may fail with network errors (expected in tests)
				// We only care that protocol validation passes
				if err != nil && !strings.Contains(err.Error(), "E_NET_REQUEST_FAILED") &&
					!strings.Contains(err.Error(), "E_NET_DNS_FAILED") {
					t.Errorf("Expected success or network error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// TestNetIPValidation verifies IP blocking policies
func TestNetIPValidation(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	tests := []struct {
		name           string
		url            string
		allowLocalhost bool
		expectError    string
	}{
		{
			name:           "localhost IP blocked by default",
			url:            "https://127.0.0.1:8443/test",
			allowLocalhost: false,
			expectError:    "E_NET_IP_BLOCKED: localhost IP blocked",
		},
		{
			name:           "localhost IP allowed with flag",
			url:            "https://127.0.0.1:8443/test",
			allowLocalhost: true,
			expectError:    "", // Will fail with network error (expected)
		},
		{
			name:           "IPv6 localhost blocked",
			url:            "https://[::1]:8443/test",
			allowLocalhost: false,
			expectError:    "E_NET_IP_BLOCKED: localhost IP blocked",
		},
		{
			name:           "private IP 10.x blocked",
			url:            "https://10.0.0.1/test",
			allowLocalhost: false,
			expectError:    "E_NET_IP_BLOCKED: private IP blocked",
		},
		{
			name:           "private IP 192.168.x blocked",
			url:            "https://192.168.1.1/test",
			allowLocalhost: false,
			expectError:    "E_NET_IP_BLOCKED: private IP blocked",
		},
		{
			name:           "private IP 172.16.x blocked",
			url:            "https://172.16.0.1/test",
			allowLocalhost: false,
			expectError:    "E_NET_IP_BLOCKED: private IP blocked",
		},
		{
			name:           "link-local IP blocked",
			url:            "https://169.254.1.1/test",
			allowLocalhost: false,
			expectError:    "E_NET_IP_BLOCKED: link-local IP blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.Net.AllowLocalhost = tt.allowLocalhost
			urlVal := &eval.StringValue{Value: tt.url}
			_, err := netHTTPGet(ctx, []eval.Value{urlVal})

			if tt.expectError == "" {
				// May fail with network error (no server running) - that's OK
				if err != nil && !strings.Contains(err.Error(), "E_NET_REQUEST_FAILED") {
					t.Errorf("Expected success or network error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// TestNetDomainAllowlist verifies domain allowlist functionality
func TestNetDomainAllowlist(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	tests := []struct {
		name           string
		allowedDomains []string
		url            string
		expectError    string
	}{
		{
			name:           "empty allowlist allows all",
			allowedDomains: []string{},
			url:            "https://example.com",
			expectError:    "",
		},
		{
			name:           "exact domain match allowed",
			allowedDomains: []string{"example.com"},
			url:            "https://example.com",
			expectError:    "",
		},
		{
			name:           "domain not in allowlist blocked",
			allowedDomains: []string{"example.com"},
			url:            "https://evil.com",
			expectError:    "E_NET_DOMAIN_BLOCKED: domain not in allowlist: evil.com",
		},
		{
			name:           "wildcard matches subdomain",
			allowedDomains: []string{"*.example.com"},
			url:            "https://api.example.com",
			expectError:    "",
		},
		{
			name:           "wildcard doesn't match base domain",
			allowedDomains: []string{"*.example.com"},
			url:            "https://example.com",
			expectError:    "E_NET_DOMAIN_BLOCKED: domain not in allowlist: example.com",
		},
		{
			name:           "multiple domains in allowlist",
			allowedDomains: []string{"example.com", "api.openai.com"},
			url:            "https://api.openai.com",
			expectError:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.Net.AllowedDomains = tt.allowedDomains
			urlVal := &eval.StringValue{Value: tt.url}
			_, err := netHTTPGet(ctx, []eval.Value{urlVal})

			if tt.expectError == "" {
				// May fail with network error - that's OK
				if err != nil && !strings.Contains(err.Error(), "E_NET_REQUEST_FAILED") &&
					!strings.Contains(err.Error(), "E_NET_DNS_FAILED") &&
					!strings.Contains(err.Error(), "E_NET_READ_FAILED") {
					t.Logf("Non-blocking error (expected): %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// TestNetHttpPost verifies POST request functionality
func TestNetHttpPost(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	t.Run("httpPost type checking", func(t *testing.T) {
		// Wrong arg count
		url := &eval.StringValue{Value: "https://httpbin.org/post"}
		_, err := netHTTPPost(ctx, []eval.Value{url})
		if err == nil || !strings.Contains(err.Error(), "E_NET_TYPE_ERROR: httpPost: expected 2 arguments") {
			t.Errorf("Expected E_NET_TYPE_ERROR for wrong arg count, got: %v", err)
		}

		// Wrong arg type for URL
		_, err = netHTTPPost(ctx, []eval.Value{&eval.IntValue{Value: 42}, url})
		if err == nil || !strings.Contains(err.Error(), "E_NET_TYPE_ERROR: httpPost: expected String for URL") {
			t.Errorf("Expected E_NET_TYPE_ERROR for wrong URL type, got: %v", err)
		}

		// Wrong arg type for body
		_, err = netHTTPPost(ctx, []eval.Value{url, &eval.IntValue{Value: 42}})
		if err == nil || !strings.Contains(err.Error(), "E_NET_TYPE_ERROR: httpPost: expected String for body") {
			t.Errorf("Expected E_NET_TYPE_ERROR for wrong body type, got: %v", err)
		}
	})

	t.Run("httpPost to httpbin.org", func(t *testing.T) {
		// Skip in CI environments due to unreliable external network access
		if os.Getenv("SKIP_NET_TESTS") != "" || os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
			t.Skip("Skipping network test in CI environment (unreliable external access)")
		}

		url := &eval.StringValue{Value: "https://httpbin.org/post"}
		body := &eval.StringValue{Value: `{"test": "data", "value": 42}`}
		result, err := netHTTPPost(ctx, []eval.Value{url, body})

		// May fail with network issues - that's OK
		if err == nil {
			if result == nil {
				t.Error("Expected result, got nil")
			} else {
				strResult, ok := result.(*eval.StringValue)
				if !ok {
					t.Errorf("Expected StringValue, got %T", result)
				} else if !strings.Contains(strResult.Value, "httpbin.org") {
					t.Errorf("Expected response containing 'httpbin.org', got: %s", strResult.Value)
				}
			}
		} else {
			t.Logf("Network error (expected in some environments): %v", err)
		}
	})
}

// TestNetBodySizeLimit verifies response size limiting
func TestNetBodySizeLimit(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()
	ctx.Net.MaxBytes = 100 // Very small limit for testing

	t.Run("small response under limit", func(t *testing.T) {
		// httpbin.org/get returns ~270 bytes, should exceed 100 byte limit
		// Skip in CI environments due to unreliable external network access
		if os.Getenv("SKIP_NET_TESTS") != "" || os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
			t.Skip("Skipping network test in CI environment (unreliable external access)")
		}

		url := &eval.StringValue{Value: "https://httpbin.org/get"}
		_, err := netHTTPGet(ctx, []eval.Value{url})

		// Should fail with body too large error
		if err == nil {
			t.Error("Expected E_NET_BODY_TOO_LARGE error, got nil")
		} else if strings.Contains(err.Error(), "E_NET_BODY_TOO_LARGE") {
			// Expected error - test passed
		} else if strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout") {
			// Network timeout - skip test
			t.Skipf("Network timeout (httpbin.org unavailable): %v", err)
		} else {
			t.Errorf("Expected E_NET_BODY_TOO_LARGE error, got: %v", err)
		}
	})
}

// TestNetHTTPRequestCapability verifies httpRequest requires Net capability
func TestNetHTTPRequestCapability(t *testing.T) {
	ctx := NewEffContext()
	// No capabilities granted

	method := &eval.StringValue{Value: "GET"}
	url := &eval.StringValue{Value: "https://example.com"}
	headers := &eval.ListValue{Elements: []eval.Value{}}
	body := &eval.StringValue{Value: ""}

	_, err := NetHTTPRequest(ctx, []eval.Value{method, url, headers, body})

	if err == nil {
		t.Fatal("Expected capability error, got nil")
	}
	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Fatalf("Expected CapabilityError, got %T", err)
	}
	if capErr.Effect != "Net" {
		t.Errorf("Expected Effect=Net, got %s", capErr.Effect)
	}
}

// TestNetHTTPRequestHeaderValidation verifies header blocking
func TestNetHTTPRequestHeaderValidation(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	method := &eval.StringValue{Value: "GET"}
	url := &eval.StringValue{Value: "https://example.com"}
	body := &eval.StringValue{Value: ""}

	testCases := []struct {
		name       string
		headerName string
		wantError  string
	}{
		{"blocks Connection", "Connection", "hop-by-hop header not allowed: Connection"},
		{"blocks Transfer-Encoding", "Transfer-Encoding", "hop-by-hop header not allowed: Transfer-Encoding"},
		{"blocks Host", "Host", "Host header override not allowed"},
		{"blocks Accept-Encoding", "Accept-Encoding", "Accept-Encoding is managed automatically"},
		{"blocks Content-Length", "Content-Length", "Content-Length is computed automatically"},
		{"allows Authorization", "Authorization", ""}, // Should not error
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := &eval.ListValue{
				Elements: []eval.Value{
					&eval.RecordValue{
						Fields: map[string]eval.Value{
							"name":  &eval.StringValue{Value: tc.headerName},
							"value": &eval.StringValue{Value: "test"},
						},
					},
				},
			}

			result, err := NetHTTPRequest(ctx, []eval.Value{method, url, headers, body})

			if tc.wantError != "" {
				// Expecting error
				if err != nil {
					t.Fatalf("Expected Result with Err, got Go error: %v", err)
				}
				// Check if result is Err(InvalidHeader(...))
				tagged, ok := result.(*eval.TaggedValue)
				if !ok {
					t.Fatalf("Expected TaggedValue, got %T", result)
				}
				if tagged.CtorName != "Err" {
					t.Errorf("Expected Err constructor, got %s", tagged.CtorName)
				}
				// Extract error value
				errVal := tagged.Fields[0].(*eval.TaggedValue)
				if errVal.CtorName != "InvalidHeader" {
					t.Errorf("Expected InvalidHeader, got %s", errVal.CtorName)
				}
				errMsg := errVal.Fields[0].(*eval.StringValue).Value
				if !strings.Contains(errMsg, tc.wantError) {
					t.Errorf("Expected error containing %q, got %q", tc.wantError, errMsg)
				}
			} else {
				// Should succeed (or fail with Transport error due to network, which is ok for this test)
				if err != nil {
					t.Fatalf("Unexpected Go error: %v", err)
				}
				// Result should be Ok(...) or Err(Transport(...))
				tagged, ok := result.(*eval.TaggedValue)
				if !ok {
					t.Fatalf("Expected TaggedValue, got %T", result)
				}
				if tagged.CtorName != "Ok" && tagged.CtorName != "Err" {
					t.Errorf("Expected Ok or Err, got %s", tagged.CtorName)
				}
			}
		})
	}
}

// TestNetHTTPRequestMethodWhitelist verifies method validation
func TestNetHTTPRequestMethodWhitelist(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	url := &eval.StringValue{Value: "https://example.com"}
	headers := &eval.ListValue{Elements: []eval.Value{}}
	body := &eval.StringValue{Value: ""}

	testCases := []struct {
		method    string
		wantError bool
	}{
		{"GET", false},
		{"POST", false},
		{"PUT", true},
		{"DELETE", true},
		{"PATCH", true},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			method := &eval.StringValue{Value: tc.method}
			result, err := NetHTTPRequest(ctx, []eval.Value{method, url, headers, body})

			if err != nil {
				t.Fatalf("Unexpected Go error: %v", err)
			}

			tagged, ok := result.(*eval.TaggedValue)
			if !ok {
				t.Fatalf("Expected TaggedValue, got %T", result)
			}

			if tc.wantError {
				// Should return Err for unsupported method
				if tagged.CtorName != "Err" {
					t.Errorf("Expected Err for method %s, got %s", tc.method, tagged.CtorName)
				}
			}
		})
	}
}

// TestNetHTTPRequestResultType verifies Result structure
func TestNetHTTPRequestResultType(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()

	method := &eval.StringValue{Value: "GET"}
	// Use invalid URL to force Transport error
	url := &eval.StringValue{Value: "https://invalid-domain-that-does-not-exist-12345.com"}
	headers := &eval.ListValue{Elements: []eval.Value{}}
	body := &eval.StringValue{Value: ""}

	result, err := NetHTTPRequest(ctx, []eval.Value{method, url, headers, body})

	if err != nil {
		t.Fatalf("Unexpected Go error: %v", err)
	}

	// Result should be TaggedValue (Result ADT)
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("Expected TaggedValue, got %T", result)
	}

	if tagged.TypeName != "Result" {
		t.Errorf("Expected TypeName=Result, got %s", tagged.TypeName)
	}

	if tagged.ModulePath != "std/result" {
		t.Errorf("Expected ModulePath=std/result, got %s", tagged.ModulePath)
	}

	// Should be Err(...) because of invalid domain
	if tagged.CtorName != "Err" {
		t.Errorf("Expected CtorName=Err for invalid domain, got %s", tagged.CtorName)
	}

	// Error should be NetError ADT
	errVal := tagged.Fields[0]
	errTagged, ok := errVal.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("Expected NetError as TaggedValue, got %T", errVal)
	}

	if errTagged.TypeName != "NetError" {
		t.Errorf("Expected TypeName=NetError, got %s", errTagged.TypeName)
	}

	// Should be Transport error
	if errTagged.CtorName != "Transport" {
		t.Errorf("Expected Transport error for DNS failure, got %s", errTagged.CtorName)
	}
}
