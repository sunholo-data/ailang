package testctx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo/ailang/internal/eval"
)

func TestNewMockEffContext(t *testing.T) {
	ctx := NewMockEffContext()

	// Should be initialized with test-friendly defaults
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.EffContext)
	assert.Equal(t, int64(42), ctx.Env.Seed)
	assert.Equal(t, 5*time.Second, ctx.Net.Timeout)
	assert.True(t, ctx.Net.AllowHTTP)
	assert.True(t, ctx.Net.AllowLocalhost)
	assert.Nil(t, ctx.HTTPClient)
}

func TestGrantAll(t *testing.T) {
	ctx := NewMockEffContext()

	// Grant multiple capabilities
	ctx.GrantAll("IO", "Net", "FS")

	// Should have all three capabilities
	assert.True(t, ctx.HasCap("IO"))
	assert.True(t, ctx.HasCap("Net"))
	assert.True(t, ctx.HasCap("FS"))
	assert.False(t, ctx.HasCap("Clock")) // Not granted
}

func TestSetHTTPClient(t *testing.T) {
	ctx := NewMockEffContext()

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("mock response"))
	}))
	defer server.Close()

	// Set the mock client
	ctx.SetHTTPClient(server.Client())

	assert.NotNil(t, ctx.HTTPClient)
	assert.Equal(t, server.Client(), ctx.GetHTTPClient())
}

func TestGetHTTPClient(t *testing.T) {
	ctx := NewMockEffContext()

	// Without setting a mock client, should return default
	assert.Equal(t, http.DefaultClient, ctx.GetHTTPClient())

	// After setting a mock client, should return it
	mockClient := &http.Client{Timeout: 1 * time.Second}
	ctx.SetHTTPClient(mockClient)
	assert.Equal(t, mockClient, ctx.GetHTTPClient())
}

func TestSetAllowedHosts(t *testing.T) {
	ctx := NewMockEffContext()

	hosts := []string{"example.com", "api.test.com"}
	ctx.SetAllowedHosts(hosts)

	assert.Equal(t, hosts, ctx.Net.AllowedDomains)
}

func TestSetNetTimeout(t *testing.T) {
	ctx := NewMockEffContext()

	timeout := 10 * time.Second
	ctx.SetNetTimeout(timeout)

	assert.Equal(t, timeout, ctx.Net.Timeout)
}

// Value Constructor Tests

func TestMakeString(t *testing.T) {
	v := MakeString("hello")
	assert.IsType(t, &eval.StringValue{}, v)
	assert.Equal(t, "hello", v.(*eval.StringValue).Value)
}

func TestMakeInt(t *testing.T) {
	v := MakeInt(42)
	assert.IsType(t, &eval.IntValue{}, v)
	assert.Equal(t, 42, v.(*eval.IntValue).Value)
}

func TestMakeBool(t *testing.T) {
	vTrue := MakeBool(true)
	vFalse := MakeBool(false)

	assert.IsType(t, &eval.BoolValue{}, vTrue)
	assert.IsType(t, &eval.BoolValue{}, vFalse)
	assert.True(t, vTrue.(*eval.BoolValue).Value)
	assert.False(t, vFalse.(*eval.BoolValue).Value)
}

func TestMakeFloat(t *testing.T) {
	v := MakeFloat(3.14159)
	assert.IsType(t, &eval.FloatValue{}, v)
	assert.InDelta(t, 3.14159, v.(*eval.FloatValue).Value, 0.00001)
}

func TestMakeList(t *testing.T) {
	items := []eval.Value{
		MakeInt(1),
		MakeInt(2),
		MakeInt(3),
	}
	v := MakeList(items)

	assert.IsType(t, &eval.ListValue{}, v)
	assert.Len(t, v.(*eval.ListValue).Elements, 3)
}

func TestMakeRecord(t *testing.T) {
	fields := map[string]eval.Value{
		"name":  MakeString("Alice"),
		"age":   MakeInt(30),
		"admin": MakeBool(true),
	}
	v := MakeRecord(fields)

	assert.IsType(t, &eval.RecordValue{}, v)
	assert.Len(t, v.(*eval.RecordValue).Fields, 3)
	assert.Equal(t, "Alice", v.(*eval.RecordValue).Fields["name"].(*eval.StringValue).Value)
}

func TestMakeUnit(t *testing.T) {
	v := MakeUnit()
	assert.IsType(t, &eval.UnitValue{}, v)
}

// Value Extractor Tests

func TestGetString(t *testing.T) {
	v := MakeString("hello")
	assert.Equal(t, "hello", GetString(v))
}

func TestGetInt(t *testing.T) {
	v := MakeInt(42)
	assert.Equal(t, 42, GetInt(v))
}

func TestGetBool(t *testing.T) {
	vTrue := MakeBool(true)
	vFalse := MakeBool(false)
	assert.True(t, GetBool(vTrue))
	assert.False(t, GetBool(vFalse))
}

func TestGetFloat(t *testing.T) {
	v := MakeFloat(3.14159)
	assert.InDelta(t, 3.14159, GetFloat(v), 0.00001)
}

func TestGetList(t *testing.T) {
	items := []eval.Value{MakeInt(1), MakeInt(2), MakeInt(3)}
	v := MakeList(items)

	result := GetList(v)
	assert.Len(t, result, 3)
	assert.Equal(t, 1, GetInt(result[0]))
	assert.Equal(t, 2, GetInt(result[1]))
	assert.Equal(t, 3, GetInt(result[2]))
}

func TestGetRecord(t *testing.T) {
	fields := map[string]eval.Value{
		"name": MakeString("Alice"),
		"age":  MakeInt(30),
	}
	v := MakeRecord(fields)

	result := GetRecord(v)
	assert.Len(t, result, 2)
	assert.Equal(t, "Alice", GetString(result["name"]))
	assert.Equal(t, 30, GetInt(result["age"]))
}

func TestIsUnit(t *testing.T) {
	unit := MakeUnit()
	str := MakeString("not unit")

	assert.True(t, IsUnit(unit))
	assert.False(t, IsUnit(str))
}

// Integration Tests

func TestComplexRecordConstruction(t *testing.T) {
	// Build a complex nested record like an HTTP response
	response := MakeRecord(map[string]eval.Value{
		"status": MakeInt(200),
		"headers": MakeList([]eval.Value{
			MakeRecord(map[string]eval.Value{
				"name":  MakeString("Content-Type"),
				"value": MakeString("application/json"),
			}),
			MakeRecord(map[string]eval.Value{
				"name":  MakeString("Content-Length"),
				"value": MakeString("42"),
			}),
		}),
		"body": MakeString(`{"success": true}`),
	})

	// Extract and verify
	fields := GetRecord(response)
	assert.Equal(t, 200, GetInt(fields["status"]))
	assert.Equal(t, `{"success": true}`, GetString(fields["body"]))

	headers := GetList(fields["headers"])
	assert.Len(t, headers, 2)

	firstHeader := GetRecord(headers[0])
	assert.Equal(t, "Content-Type", GetString(firstHeader["name"]))
	assert.Equal(t, "application/json", GetString(firstHeader["value"]))
}

func TestMockHTTPRequest(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/users", r.URL.Path)
		w.WriteHeader(200)
		w.Write([]byte(`{"users": []}`))
	}))
	defer server.Close()

	// Create mock context with server client
	ctx := NewMockEffContext()
	ctx.GrantAll("Net")
	ctx.SetHTTPClient(server.Client())

	// Verify setup
	assert.True(t, ctx.HasCap("Net"))
	assert.NotNil(t, ctx.GetHTTPClient())

	// Make a request using the mock client
	resp, err := ctx.GetHTTPClient().Get(server.URL + "/api/users")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}
