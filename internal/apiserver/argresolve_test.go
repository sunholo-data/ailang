package apiserver

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

// TestArgResolution_AllSources is the combinatorial regression test that
// would have caught the v0.19.2 "zero-padding shadows query fallback" bug.
//
// It exercises the cross-product of:
//
//	method     ∈ {GET, POST}
//	body       ∈ {nil, "{}", named JSON, positional {"args":[...]}, partial match}
//	query      ∈ {none, ?args=...&args=..., ?name=...&age=...}
//	paramTypes ∈ {empty, single-string, two-strings, single-record}
//
// The acceptance rule: handler args MUST come from real request data when
// any real data is available. Synthesized zero-pad values may ONLY survive
// when neither body nor query carried real data.
func TestArgResolution_AllSources(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		body       []byte
		paramNames []string
		paramTypes []string
		wantArgs   []interface{}
		wantSource ArgSource
	}{
		// --- The docparse repro (would FAIL on v0.19.2) ---
		{
			name:       "GET positional query args populate declared params",
			method:     "GET",
			url:        "/api/foo?args=smoke-test-claude&args=2026_04",
			body:       nil,
			paramNames: []string{"uid", "period"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"smoke-test-claude", "2026_04"},
			wantSource: ArgSourceReal,
		},
		{
			name:       "GET named query args populate single-record param",
			method:     "GET",
			url:        "/api/foo?name=Alice&age=30",
			body:       nil,
			paramNames: []string{"req"},
			paramTypes: []string{"record"},
			wantArgs: []interface{}{
				map[string]interface{}{"name": "Alice", "age": float64(30)},
			},
			wantSource: ArgSourceReal,
		},

		// --- POST with empty body still zero-pads (preserves M-SERVE-API-EMPTY-BODY-NAMED-ZEROPAD) ---
		{
			name:       "POST empty body no query zero-pads to arity",
			method:     "POST",
			url:        "/api/foo",
			body:       nil,
			paramNames: []string{"a", "b"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"", ""},
			wantSource: ArgSourceZeroPadded,
		},
		{
			name:       "POST empty object body zero-pads to arity",
			method:     "POST",
			url:        "/api/foo",
			body:       []byte(`{}`),
			paramNames: []string{"a", "b"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"", ""},
			wantSource: ArgSourceZeroPadded,
		},

		// --- Partial body match wins over query (preserves user intent) ---
		{
			name:       "POST partial body match beats query",
			method:     "POST",
			url:        "/api/foo?b=from-query",
			body:       []byte(`{"a":"from-body"}`),
			paramNames: []string{"a", "b"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"from-body", ""},
			wantSource: ArgSourceReal,
		},

		// --- Structured {"args": [...]} bodies ---
		{
			name:       "POST positional args body",
			method:     "POST",
			url:        "/api/foo",
			body:       []byte(`{"args":["X","Y"]}`),
			paramNames: []string{"a", "b"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"X", "Y"},
			wantSource: ArgSourceReal,
		},
		{
			name:       "POST positional args body padded to arity",
			method:     "POST",
			url:        "/api/foo",
			body:       []byte(`{"args":["X"]}`),
			paramNames: []string{"a", "b"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"X", ""},
			wantSource: ArgSourceReal,
		},

		// --- Empty body + empty query, no declared params ---
		{
			name:       "no params no body no query → None source",
			method:     "GET",
			url:        "/api/foo",
			body:       nil,
			paramNames: nil,
			paramTypes: nil,
			wantArgs:   nil,
			wantSource: ArgSourceNone,
		},

		// --- POST with empty body + positional query (shadow-class case) ---
		{
			name:       "POST empty body with positional query promotes query to Real",
			method:     "POST",
			url:        "/api/foo?args=X",
			body:       nil,
			paramNames: []string{"a"},
			paramTypes: []string{"string"},
			wantArgs:   []interface{}{"X"},
			wantSource: ArgSourceReal,
		},

		// --- GET no body no query → falls through to zero-pad ---
		{
			name:       "GET no body no query zero-pads (no fallback available)",
			method:     "GET",
			url:        "/api/foo",
			body:       nil,
			paramNames: []string{"a", "b"},
			paramTypes: []string{"string", "string"},
			wantArgs:   []interface{}{"", ""},
			wantSource: ArgSourceZeroPadded,
		},

		// --- JSON object body that doesn't match any param ---
		{
			name:       "POST unmatched JSON object body zero-pads (does not become single record arg)",
			method:     "POST",
			url:        "/api/foo",
			body:       []byte(`{"unrelated":"x"}`),
			paramNames: []string{"a"},
			paramTypes: []string{"string"},
			wantArgs:   []interface{}{""},
			wantSource: ArgSourceZeroPadded,
		},

		// --- Non-object JSON body ---
		{
			name:       "POST raw string body passes through as single arg",
			method:     "POST",
			url:        "/api/foo",
			body:       []byte(`"plain string"`),
			paramNames: []string{"a"},
			paramTypes: []string{"string"},
			wantArgs:   []interface{}{"plain string"},
			wantSource: ArgSourceReal,
		},

		// --- Typed zero-pads for non-string types ---
		{
			name:       "POST empty body pads int param with float64(0)",
			method:     "POST",
			url:        "/api/foo",
			body:       nil,
			paramNames: []string{"count"},
			paramTypes: []string{"int"},
			wantArgs:   []interface{}{float64(0)},
			wantSource: ArgSourceZeroPadded,
		},
		{
			name:       "POST empty body pads bool param with false",
			method:     "POST",
			url:        "/api/foo",
			body:       nil,
			paramNames: []string{"flag"},
			paramTypes: []string{"bool"},
			wantArgs:   []interface{}{false},
			wantSource: ArgSourceZeroPadded,
		},

		// --- GET with both body AND query (POST-style request); body wins when source is Real ---
		{
			name:       "GET with full body match ignores query",
			method:     "GET",
			url:        "/api/foo?a=from-query",
			body:       []byte(`{"a":"from-body"}`),
			paramNames: []string{"a"},
			paramTypes: []string{"string"},
			wantArgs:   []interface{}{"from-body"},
			wantSource: ArgSourceReal,
		},

		// --- GET with empty body + named query → record arg ---
		{
			name:       "GET named query for single-record-arg handler",
			method:     "GET",
			url:        "/api/foo?uid=X&period=Y",
			body:       nil,
			paramNames: []string{"req"},
			paramTypes: []string{"record"},
			wantArgs: []interface{}{
				map[string]interface{}{"uid": "X", "period": "Y"},
			},
			wantSource: ArgSourceReal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			args, source, err := resolveArgs(req, tc.body, tc.paramNames, tc.paramTypes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if source != tc.wantSource {
				t.Errorf("source = %v, want %v", source, tc.wantSource)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("args mismatch\n  got:  %#v\n  want: %#v", args, tc.wantArgs)
			}
		})
	}
}

// TestArgResolution_ZeroPadDoesNotShadowQuery is the direct repro of the
// docparse production bug. v0.19.2 fails this test; v0.19.3+ passes.
// Kept as a standalone test (not just a table row) because its independent
// presence in the suite is the single clearest signal of the bug class.
func TestArgResolution_ZeroPadDoesNotShadowQuery(t *testing.T) {
	req := httptest.NewRequest("GET",
		"/api/v1/debug/usage?args=smoke-test-claude&args=2026_04", nil)
	args, source, err := resolveArgs(req, nil,
		[]string{"uid", "period"},
		[]string{"string", "string"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != ArgSourceReal {
		t.Fatalf("source = %v, want ArgSourceReal — zero-padded args must NOT shadow query data", source)
	}
	if len(args) != 2 {
		t.Fatalf("got %d args, want 2: %#v", len(args), args)
	}
	if args[0] != "smoke-test-claude" {
		t.Errorf("args[0] = %v, want \"smoke-test-claude\"", args[0])
	}
	if args[1] != "2026_04" {
		t.Errorf("args[1] = %v, want \"2026_04\"", args[1])
	}
}

// TestParseArgsWithNamesEx_SourceLabels verifies the provenance labels for
// each input shape that parseArgsWithNamesEx classifies. The labels are
// load-bearing for resolveArgs's fallback decision; if these regress, the
// shadowing-class bug returns.
func TestParseArgsWithNamesEx_SourceLabels(t *testing.T) {
	t.Run("empty body declared params → ZeroPadded", func(t *testing.T) {
		_, src, _ := parseArgsWithNamesEx(nil, []string{"a"}, []string{"string"})
		if src != ArgSourceZeroPadded {
			t.Errorf("source = %v, want ZeroPadded", src)
		}
	})
	t.Run("matched named-body keys → Real", func(t *testing.T) {
		_, src, _ := parseArgsWithNamesEx([]byte(`{"a":"x"}`),
			[]string{"a"}, []string{"string"})
		if src != ArgSourceReal {
			t.Errorf("source = %v, want Real", src)
		}
	})
	t.Run("unmatched-key body + declared params → ZeroPadded", func(t *testing.T) {
		_, src, _ := parseArgsWithNamesEx([]byte(`{"other":"x"}`),
			[]string{"a"}, []string{"string"})
		if src != ArgSourceZeroPadded {
			t.Errorf("source = %v, want ZeroPadded", src)
		}
	})
	t.Run("structured args body → Real", func(t *testing.T) {
		_, src, _ := parseArgsWithNamesEx([]byte(`{"args":["x"]}`),
			[]string{"a"}, []string{"string"})
		if src != ArgSourceReal {
			t.Errorf("source = %v, want Real", src)
		}
	})
	t.Run("no declared params + empty body → None", func(t *testing.T) {
		_, src, _ := parseArgsWithNamesEx(nil, nil, nil)
		if src != ArgSourceNone {
			t.Errorf("source = %v, want None", src)
		}
	})
}
