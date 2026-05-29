package apiserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestNowrapResponse(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Call the "hello" function with nowrap option — should return raw JSON
	req := httptest.NewRequest("POST", "/api/test/api/greet/hello", strings.NewReader(`{"args": ["World"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call directly with nowrap
	srv.callFunction(w, req, "test/api/greet", "hello", callOpts{Nowrap: true})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Response should be raw string, not wrapped in FunctionCallResponse
	body := strings.TrimSpace(w.Body.String())
	if body != `"Hello, World!"` {
		t.Errorf("expected raw JSON string, got %s", body)
	}

	// Should not contain envelope fields
	if strings.Contains(body, "module") || strings.Contains(body, "elapsed_ms") {
		t.Errorf("expected no envelope, got %s", body)
	}

	// Should have timing header
	if w.Header().Get("X-Elapsed-Ms") == "" {
		t.Error("expected X-Elapsed-Ms header")
	}
}

func TestNonNowrapResponse(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Call without nowrap — should return envelope
	req := httptest.NewRequest("POST", "/api/test/api/greet/hello", strings.NewReader(`{"args": ["World"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.callFunction(w, req, "test/api/greet", "hello")

	body := w.Body.String()
	if !strings.Contains(body, `"module"`) || !strings.Contains(body, `"elapsed_ms"`) {
		t.Errorf("expected FunctionCallResponse envelope, got %s", body)
	}
}

func TestParseArgs_EmptyBody(t *testing.T) {
	args, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args != nil {
		t.Errorf("expected nil args for empty body, got %v", args)
	}
}

func TestParseArgs_EmptyString(t *testing.T) {
	args, err := parseArgs([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args != nil {
		t.Errorf("expected nil args for empty string, got %v", args)
	}
}

func TestParseArgs_ArgsArray(t *testing.T) {
	args, err := parseArgs([]byte(`{"args": ["hello", 42]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "hello" {
		t.Errorf("expected first arg 'hello', got %v", args[0])
	}
}

func TestParseArgs_SingleValue(t *testing.T) {
	args, err := parseArgs([]byte(`"just a string"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "just a string" {
		t.Errorf("expected 'just a string', got %v", args[0])
	}
}

func TestParseQueryArgs_Positional(t *testing.T) {
	query := url.Values{"args": {"3", "5"}}
	args := parseQueryArgs(query)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	// Numbers should be parsed as float64
	if args[0] != float64(3) {
		t.Errorf("expected 3, got %v (%T)", args[0], args[0])
	}
	if args[1] != float64(5) {
		t.Errorf("expected 5, got %v (%T)", args[1], args[1])
	}
}

func TestParseQueryArgs_Named(t *testing.T) {
	query := url.Values{"name": {"Alice"}, "age": {"30"}}
	args := parseQueryArgs(query)
	if len(args) != 1 {
		t.Fatalf("expected 1 record arg, got %d", len(args))
	}
	record, ok := args[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", args[0])
	}
	if record["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", record["name"])
	}
	if record["age"] != float64(30) {
		t.Errorf("expected age=30, got %v (%T)", record["age"], record["age"])
	}
}

func TestParseQueryArgs_Empty(t *testing.T) {
	args := parseQueryArgs(url.Values{})
	if args != nil {
		t.Fatalf("expected nil for empty query, got %v", args)
	}
}

func TestParseQueryArgs_StringValues(t *testing.T) {
	query := url.Values{"query": {"hello world"}, "flag": {"true"}}
	args := parseQueryArgs(query)
	if len(args) != 1 {
		t.Fatalf("expected 1 record arg, got %d", len(args))
	}
	record := args[0].(map[string]interface{})
	if record["query"] != "hello world" {
		t.Errorf("expected 'hello world', got %v", record["query"])
	}
	if record["flag"] != true {
		t.Errorf("expected true, got %v (%T)", record["flag"], record["flag"])
	}
}

// TestParseQueryArgs_UnderscoreStaysString verifies that strings with underscores
// between digits (e.g., "2026_04") are NOT parsed as numbers.
// Regression: Go's strconv.ParseFloat treats underscores as digit separators,
// so "2026_04" became float64(202604), which then became IntValue(202604),
// causing concat_String failures when building Firestore paths.
func TestParseQueryArgs_UnderscoreStaysString(t *testing.T) {
	query := url.Values{"args": {"test-user", "2026_04"}}
	args := parseQueryArgs(query)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	// "2026_04" must stay a string, NOT become float64(202604)
	s, ok := args[1].(string)
	if !ok {
		t.Fatalf("expected string for '2026_04', got %T (%v)", args[1], args[1])
	}
	if s != "2026_04" {
		t.Errorf("expected '2026_04', got %q", s)
	}
}

// TestTryParseJSON_NumericEdgeCases tests the looksNumeric guard.
func TestTryParseJSON_NumericEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		wantType string // "float64", "string", "bool"
	}{
		{"42", "float64"},
		{"-7", "float64"},
		{"3.14", "float64"},
		{"1e5", "float64"},
		{"true", "bool"},
		{"false", "bool"},
		{"hello", "string"},
		{"2026_04", "string"},   // underscore between digits
		{"1_000_000", "string"}, // Go numeric literal with separators
		{"test_123", "string"},  // mixed alpha and underscore
		{"_leading", "string"},  // leading underscore
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tryParseJSON(tt.input)
			var gotType string
			switch got.(type) {
			case float64:
				gotType = "float64"
			case bool:
				gotType = "bool"
			case string:
				gotType = "string"
			default:
				gotType = fmt.Sprintf("%T", got)
			}
			if gotType != tt.wantType {
				t.Errorf("tryParseJSON(%q) type = %s, want %s (value: %v)", tt.input, gotType, tt.wantType, got)
			}
		})
	}
}

func TestBuildHttpRequestRecord(t *testing.T) {
	req, _ := http.NewRequest("POST", "/webhooks/stripe?event=checkout&debug=true", nil)
	req.Header.Set("Stripe-Signature", "t=123,v1=abc")
	req.Header.Set("Content-Type", "application/json")

	body := []byte(`{"type": "checkout.session.completed"}`)
	rec := buildHttpRequestRecord(req, body)

	// Check body
	if rec["body"] != `{"type": "checkout.session.completed"}` {
		t.Errorf("unexpected body: %v", rec["body"])
	}

	// Check method
	if rec["method"] != "POST" {
		t.Errorf("unexpected method: %v", rec["method"])
	}

	// Check path
	if rec["path"] != "/webhooks/stripe" {
		t.Errorf("unexpected path: %v", rec["path"])
	}

	// Check headers are JObject
	headersJObj, ok := rec["headers"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("headers is not *eval.TaggedValue: %T", rec["headers"])
	}
	if headersJObj.CtorName != "JObject" {
		t.Fatalf("headers should be JObject, got %s", headersJObj.CtorName)
	}
	// Verify Stripe-Signature header is present (hyphenated key!)
	found := findJObjectValue(headersJObj, "Stripe-Signature")
	if found != "t=123,v1=abc" {
		t.Errorf("expected Stripe-Signature 't=123,v1=abc', got %q", found)
	}
	found = findJObjectValue(headersJObj, "Content-Type")
	if found != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", found)
	}

	// Check query is JObject
	queryJObj, ok := rec["query"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("query is not *eval.TaggedValue: %T", rec["query"])
	}
	if queryJObj.CtorName != "JObject" {
		t.Fatalf("query should be JObject, got %s", queryJObj.CtorName)
	}
	if findJObjectValue(queryJObj, "event") != "checkout" {
		t.Errorf("expected query event 'checkout', got %q", findJObjectValue(queryJObj, "event"))
	}
}

// findJObjectValue extracts a string value from a JObject by key (for testing).
func findJObjectValue(jobj *eval.TaggedValue, key string) string {
	if len(jobj.Fields) == 0 {
		return ""
	}
	list, ok := jobj.Fields[0].(*eval.ListValue)
	if !ok {
		return ""
	}
	for _, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			continue
		}
		kv, ok := rec.Fields["key"].(*eval.StringValue)
		if !ok || kv.Value != key {
			continue
		}
		valTag, ok := rec.Fields["value"].(*eval.TaggedValue)
		if !ok || valTag.CtorName != "JString" || len(valTag.Fields) == 0 {
			continue
		}
		sv, ok := valTag.Fields[0].(*eval.StringValue)
		if ok {
			return sv.Value
		}
	}
	return ""
}

func TestBuildHttpRequestRecord_EmptyBody(t *testing.T) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rec := buildHttpRequestRecord(req, nil)

	if rec["body"] != "" {
		t.Errorf("expected empty body, got %q", rec["body"])
	}
	if rec["method"] != "GET" {
		t.Errorf("unexpected method: %v", rec["method"])
	}

	queryJObj, ok := rec["query"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("query should be *eval.TaggedValue, got %T", rec["query"])
	}
	if queryJObj.CtorName != "JObject" {
		t.Fatalf("query should be JObject, got %s", queryJObj.CtorName)
	}
	list := queryJObj.Fields[0].(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Errorf("expected empty query JObject, got %d elements", len(list.Elements))
	}
}
