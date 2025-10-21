package testctx

import (
	"net/http"
	"time"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// MockEffContext provides a test-friendly effect context with mocking capabilities
//
// The mock context extends the standard EffContext with test-specific features:
//   - Mock HTTP clients for hermetic network tests
//   - Pre-configured capability grants
//   - Simplified network security settings for testing
//   - Value constructor/extractor helpers
//
// Example usage:
//
//	ctx := testctx.NewMockEffContext()
//	ctx.GrantAll("IO", "Net")
//	ctx.SetHTTPClient(mockClient)
//
//	// Test a builtin
//	result, err := myBuiltin(ctx, arg1, arg2)
//	assert.NoError(t, err)
//	assert.Equal(t, expected, testctx.GetString(result))
type MockEffContext struct {
	*effects.EffContext
	HTTPClient *http.Client // Mock HTTP client for testing
}

// NewMockEffContext creates a new mock effect context for testing
//
// The mock context is pre-configured for hermetic testing:
//   - No capabilities granted by default (must call Grant or GrantAll)
//   - Deterministic seed (AILANG_SEED=42)
//   - Short timeouts (5s)
//   - Localhost allowed for testing
//   - HTTP allowed for testing
//
// Returns:
//   - A new MockEffContext ready for testing
func NewMockEffContext() *MockEffContext {
	ctx := effects.NewEffContext()

	// Configure for testing
	ctx.Env.Seed = 42 // Deterministic seed
	ctx.Net.Timeout = 5 * time.Second
	ctx.Net.AllowHTTP = true
	ctx.Net.AllowLocalhost = true

	return &MockEffContext{
		EffContext: ctx,
		HTTPClient: nil, // Will use default if not set
	}
}

// GrantAll grants multiple capabilities at once
//
// This is a convenience method for tests that need multiple capabilities.
//
// Parameters:
//   - caps: Capability names to grant (e.g., "IO", "Net", "FS")
//
// Example:
//
//	ctx.GrantAll("IO", "Net", "FS")
func (m *MockEffContext) GrantAll(caps ...string) {
	for _, cap := range caps {
		m.Grant(effects.NewCapability(cap))
	}
}

// SetHTTPClient sets a mock HTTP client for network tests
//
// The mock client will be used by Net effect operations instead of the
// default http.DefaultClient. This enables hermetic testing without
// real network requests.
//
// Parameters:
//   - client: A mock http.Client (can use httptest.NewServer)
//
// Example:
//
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(200)
//	    w.Write([]byte(`{"status": "ok"}`))
//	}))
//	defer server.Close()
//
//	ctx.SetHTTPClient(server.Client())
func (m *MockEffContext) SetHTTPClient(client *http.Client) {
	m.HTTPClient = client
}

// SetAllowedHosts restricts network access to specific hosts
//
// This is useful for testing network security features or ensuring
// tests only access specific mock servers.
//
// Parameters:
//   - hosts: List of allowed hostnames (e.g., ["example.com", "api.test.com"])
//
// Example:
//
//	ctx.SetAllowedHosts([]string{"api.example.com"})
func (m *MockEffContext) SetAllowedHosts(hosts []string) {
	m.Net.AllowedDomains = hosts
}

// SetNetTimeout sets the network request timeout
//
// Parameters:
//   - timeout: Request timeout duration
//
// Example:
//
//	ctx.SetNetTimeout(1 * time.Second)
func (m *MockEffContext) SetNetTimeout(timeout time.Duration) {
	m.Net.Timeout = timeout
}

// GetHTTPClient returns the HTTP client to use for requests
//
// Returns the mock client if set, otherwise returns http.DefaultClient.
//
// Returns:
//   - HTTP client for network operations
func (m *MockEffContext) GetHTTPClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return http.DefaultClient
}

// Value Constructor Helpers
//
// These helpers make it easy to construct AILANG values from Go values
// for use in builtin tests.

// MakeString creates an AILANG StringValue from a Go string
//
// Parameters:
//   - s: Go string value
//
// Returns:
//   - AILANG StringValue
//
// Example:
//
//	url := testctx.MakeString("https://example.com")
func MakeString(s string) eval.Value {
	return &eval.StringValue{Value: s}
}

// MakeInt creates an AILANG IntValue from a Go int
//
// Parameters:
//   - n: Go int value
//
// Returns:
//   - AILANG IntValue
//
// Example:
//
//	timeout := testctx.MakeInt(5000)
func MakeInt(n int) eval.Value {
	return &eval.IntValue{Value: n}
}

// MakeBool creates an AILANG BoolValue from a Go bool
//
// Parameters:
//   - b: Go bool value
//
// Returns:
//   - AILANG BoolValue
//
// Example:
//
//	verbose := testctx.MakeBool(true)
func MakeBool(b bool) eval.Value {
	return &eval.BoolValue{Value: b}
}

// MakeFloat creates an AILANG FloatValue from a Go float64
//
// Parameters:
//   - f: Go float64 value
//
// Returns:
//   - AILANG FloatValue
//
// Example:
//
//	pi := testctx.MakeFloat(3.14159)
func MakeFloat(f float64) eval.Value {
	return &eval.FloatValue{Value: f}
}

// MakeList creates an AILANG ListValue from a slice of values
//
// Parameters:
//   - items: Slice of AILANG values
//
// Returns:
//   - AILANG ListValue
//
// Example:
//
//	headers := testctx.MakeList([]eval.Value{
//	    testctx.MakeRecord(map[string]eval.Value{
//	        "name":  testctx.MakeString("Content-Type"),
//	        "value": testctx.MakeString("application/json"),
//	    }),
//	})
func MakeList(items []eval.Value) eval.Value {
	return &eval.ListValue{Elements: items}
}

// MakeRecord creates an AILANG RecordValue from a map
//
// Parameters:
//   - fields: Map of field names to AILANG values
//
// Returns:
//   - AILANG RecordValue
//
// Example:
//
//	user := testctx.MakeRecord(map[string]eval.Value{
//	    "id":    testctx.MakeInt(123),
//	    "name":  testctx.MakeString("Alice"),
//	    "admin": testctx.MakeBool(true),
//	})
func MakeRecord(fields map[string]eval.Value) eval.Value {
	return &eval.RecordValue{Fields: fields}
}

// MakeUnit creates an AILANG unit value
//
// Returns:
//   - AILANG unit value
//
// Example:
//
//	unit := testctx.MakeUnit()
func MakeUnit() eval.Value {
	return &eval.UnitValue{}
}

// Value Extractor Helpers
//
// These helpers extract Go values from AILANG values for assertions.

// GetString extracts a Go string from an AILANG StringValue
//
// Parameters:
//   - v: AILANG value (must be StringValue)
//
// Returns:
//   - Go string value
//   - Panics if v is not a StringValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	assert.Equal(t, "expected", testctx.GetString(result))
func GetString(v eval.Value) string {
	return v.(*eval.StringValue).Value
}

// GetInt extracts a Go int from an AILANG IntValue
//
// Parameters:
//   - v: AILANG value (must be IntValue)
//
// Returns:
//   - Go int value
//   - Panics if v is not an IntValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	assert.Equal(t, 42, testctx.GetInt(result))
func GetInt(v eval.Value) int {
	return v.(*eval.IntValue).Value
}

// GetBool extracts a Go bool from an AILANG BoolValue
//
// Parameters:
//   - v: AILANG value (must be BoolValue)
//
// Returns:
//   - Go bool value
//   - Panics if v is not a BoolValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	assert.True(t, testctx.GetBool(result))
func GetBool(v eval.Value) bool {
	return v.(*eval.BoolValue).Value
}

// GetFloat extracts a Go float64 from an AILANG FloatValue
//
// Parameters:
//   - v: AILANG value (must be FloatValue)
//
// Returns:
//   - Go float64 value
//   - Panics if v is not a FloatValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	assert.InDelta(t, 3.14, testctx.GetFloat(result), 0.01)
func GetFloat(v eval.Value) float64 {
	return v.(*eval.FloatValue).Value
}

// GetList extracts a slice of values from an AILANG ListValue
//
// Parameters:
//   - v: AILANG value (must be ListValue)
//
// Returns:
//   - Slice of AILANG values
//   - Panics if v is not a ListValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	items := testctx.GetList(result)
//	assert.Len(t, items, 3)
func GetList(v eval.Value) []eval.Value {
	return v.(*eval.ListValue).Elements
}

// GetRecord extracts a map from an AILANG RecordValue
//
// Parameters:
//   - v: AILANG value (must be RecordValue)
//
// Returns:
//   - Map of field names to AILANG values
//   - Panics if v is not a RecordValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	fields := testctx.GetRecord(result)
//	assert.Equal(t, "Alice", testctx.GetString(fields["name"]))
func GetRecord(v eval.Value) map[string]eval.Value {
	return v.(*eval.RecordValue).Fields
}

// IsUnit checks if a value is a unit value
//
// Parameters:
//   - v: AILANG value
//
// Returns:
//   - true if v is a UnitValue
//
// Example:
//
//	result, _ := myBuiltin(ctx, arg)
//	assert.True(t, testctx.IsUnit(result))
func IsUnit(v eval.Value) bool {
	_, ok := v.(*eval.UnitValue)
	return ok
}
