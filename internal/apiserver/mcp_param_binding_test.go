package apiserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// M-MCP-UNIT-PARAM-BINDING (M-DOCPARSE-RESILIENCE-FIXES M2)
//
// A declared MCP tool param that the client omits (absent key) or sends as
// JSON null used to bind to nil, which the engine converts to Unit. The
// AILANG function then crashed deep in stdlib (e.g. _str_len: expected
// String, got Unit) before any guard could run. makeToolHandler now
// presence-checks declared params and returns a structured
// "missing required parameter(s)" error BEFORE the engine call.
//
// The test module test/api/greet (see testServer) exports:
//   hello(name: string) -> string
//   add(x: int, y: int) -> int
// giving us a string param and an int param to prove the rejection is
// type-agnostic, not a string special-case.

func mcpCallReq(toolName, argsJSON string) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      toolName,
			Arguments: json.RawMessage(argsJSON),
		},
	}
}

func mcpResultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		t.Fatal("empty MCP result")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

// TestMCPHandler_OmittedStringParam_Rejected: omitting the sole declared
// param returns the structured error with IsError=true and never reaches
// the engine (no _str_len-on-Unit crash). Deterministic at -count=20.
func TestMCPHandler_OmittedStringParam_Rejected(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	ms := NewMCPServer(srv)
	handler := ms.makeToolHandler("test/api/greet", "hello", []string{"name"})

	res, err := handler(context.Background(), mcpCallReq("hello", `{}`))
	if err != nil {
		t.Fatalf("handler returned go error (should be in-band): %v", err)
	}
	if !res.IsError {
		t.Fatal("omitted required param must set IsError=true")
	}
	text := mcpResultText(t, res)
	if !strings.Contains(text, "missing required parameter(s): name") {
		t.Errorf("want missing-param error naming 'name', got %q", text)
	}
}

// TestMCPHandler_OmittedIntParam_Rejected proves the rejection is
// type-agnostic: a missing int param produces the same structured error,
// not a string-only special case. add(x,y) with only x supplied → y missing.
func TestMCPHandler_OmittedIntParam_Rejected(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	ms := NewMCPServer(srv)
	handler := ms.makeToolHandler("test/api/greet", "add", []string{"x", "y"})

	res, err := handler(context.Background(), mcpCallReq("add", `{"x": 5}`))
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("omitted int param must set IsError=true")
	}
	text := mcpResultText(t, res)
	if !strings.Contains(text, "missing required parameter(s): y") {
		t.Errorf("want missing-param error naming 'y', got %q", text)
	}
}

// TestMCPHandler_MultipleMissing_DeterministicOrder verifies the missing
// list is reported in declaration order (x before y), not Go map order.
// Run at -count=20 to catch any map-iteration nondeterminism.
func TestMCPHandler_MultipleMissing_DeterministicOrder(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	ms := NewMCPServer(srv)
	handler := ms.makeToolHandler("test/api/greet", "add", []string{"x", "y"})

	res, err := handler(context.Background(), mcpCallReq("add", `{}`))
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("both params missing must set IsError=true")
	}
	text := mcpResultText(t, res)
	if !strings.Contains(text, "missing required parameter(s): x, y") {
		t.Errorf("want params in declaration order 'x, y', got %q", text)
	}
}

// TestMCPHandler_ExplicitNull_Rejected: an explicit JSON null for a declared
// param is rejected the same as omission — it must NOT bind to nil→Unit.
func TestMCPHandler_ExplicitNull_Rejected(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	ms := NewMCPServer(srv)
	handler := ms.makeToolHandler("test/api/greet", "hello", []string{"name"})

	res, err := handler(context.Background(), mcpCallReq("hello", `{"name": null}`))
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("explicit null param must set IsError=true (not bound to Unit)")
	}
	text := mcpResultText(t, res)
	if !strings.Contains(text, "missing required parameter(s): name") {
		t.Errorf("want missing-param error for explicit null, got %q", text)
	}
}

// TestMCPHandler_AllParamsPresent_Succeeds is the regression guard: a fully
// specified named call still reaches the engine and returns the real result.
func TestMCPHandler_AllParamsPresent_Succeeds(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	ms := NewMCPServer(srv)
	handler := ms.makeToolHandler("test/api/greet", "hello", []string{"name"})

	res, err := handler(context.Background(), mcpCallReq("hello", `{"name": "World"}`))
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("present param must succeed, got error: %s", mcpResultText(t, res))
	}
	if text := mcpResultText(t, res); !strings.Contains(text, "Hello, World!") {
		t.Errorf("want greeting result, got %q", text)
	}
}

// TestMCPHandler_LegacyPositional_Succeeds is the backward-compat regression
// guard: the {"args": [...]} positional form bypasses named binding and still
// works. The omit-check only fires when no "args" key was supplied.
func TestMCPHandler_LegacyPositional_Succeeds(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	ms := NewMCPServer(srv)
	handler := ms.makeToolHandler("test/api/greet", "hello", []string{"name"})

	res, err := handler(context.Background(), mcpCallReq("hello", `{"args": ["World"]}`))
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("legacy positional must succeed, got error: %s", mcpResultText(t, res))
	}
	if text := mcpResultText(t, res); !strings.Contains(text, "Hello, World!") {
		t.Errorf("want greeting result, got %q", text)
	}
}
