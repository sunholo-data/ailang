package apiserver

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The request record a @route("WS") handler receives, and the operator-chosen
// headers in it (M-SERVEAPI-OPERATOR-SURFACE D7).
//
// A WS handler may declare a second parameter, a record whose fields are a
// subset of {path, query, origin, headers}. serve-api builds EXACTLY the
// declared fields: a record with extra fields handed to a closed record type
// is a layout mismatch the bytecode VM can read by static index.
//
// headers : [{name: string, value: string}] holds only the headers named by
// --ws-pass-header, lower-cased, one entry per received header line, in flag
// order. The program never chooses which headers it gets, so a page cannot
// smuggle one in; credential headers cannot be named at all.

// wsReqShape is the accepted shape, quoted in every refusal.
const wsReqShape = "{path: string, query: string, origin: string, headers: [{name: string, value: string}]} (any subset of the fields)"

// legacyWSReqFields is what a handler whose req type is not a record literal
// (e.g. a type alias) receives — the pre-v0.45 record.
var legacyWSReqFields = []string{"origin", "path", "query"}

// credentialHeaders can never be passed: serve-api treats them as credentials
// (checkWSKey reads Authorization and the ailang.key.<key> subprotocol) or they
// carry the browser's session state.
var credentialHeaders = map[string]bool{
	"authorization":          true,
	"proxy-authorization":    true,
	"cookie":                 true,
	"sec-websocket-protocol": true,
	"sec-websocket-key":      true,
}

// ValidatePassHeaders checks --ws-pass-header values and returns them
// lower-cased and de-duplicated in flag order. A credential header (including
// the configured --api-key-header) or a name that is not an HTTP token is an
// error: silently dropping it would leave the operator believing it arrives.
func ValidatePassHeaders(names []string, apiKeyHeader string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, raw := range names {
		n := strings.ToLower(strings.TrimSpace(raw))
		if !isHTTPToken(n) {
			return nil, fmt.Errorf("--ws-pass-header %q is not a valid HTTP header name", raw)
		}
		if credentialHeaders[n] || (apiKeyHeader != "" && strings.EqualFold(n, apiKeyHeader)) {
			return nil, fmt.Errorf("--ws-pass-header %q carries a credential and cannot be passed to a program", raw)
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out, nil
}

// isHTTPToken reports whether s is an RFC 9110 token (a valid header name).
func isHTTPToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r > 0x7e || r <= ' ' || strings.ContainsRune("\"(),/:;<=>?@[\\]{}", r) {
			return false
		}
	}
	return true
}

// extractWSReq reads the declared fields of a WS handler's req parameter from
// the AST. It returns the field names (sorted) and a problem, if any.
func extractWSReq(fn *ast.FuncDecl) ([]string, string) {
	if len(fn.Params) < 2 {
		return nil, ""
	}
	rec, ok := fn.Params[1].Type.(*ast.RecordType)
	if !ok {
		return legacyWSReqFields, ""
	}
	var fields []string
	for _, f := range rec.Fields {
		switch f.Name {
		case "path", "query", "origin":
			if !isSimple(f.Type, "string") {
				return nil, fmt.Sprintf("req.%s must be string", f.Name)
			}
		case "headers":
			if !isHeaderList(f.Type) {
				return nil, "req.headers must be [{name: string, value: string}]"
			}
		default:
			return nil, fmt.Sprintf("req has no field %q", f.Name)
		}
		fields = append(fields, f.Name)
	}
	sort.Strings(fields)
	return fields, ""
}

func isSimple(t ast.Type, name string) bool {
	s, ok := t.(*ast.SimpleType)
	return ok && s.Name == name
}

func isHeaderList(t ast.Type) bool {
	var elem ast.Type
	switch l := t.(type) {
	case *ast.ListType:
		elem = l.Element
	case *ast.TypeApp: // the parser's form of [T]: list applied to T
		if l.Constructor != "list" || len(l.Args) != 1 {
			return false
		}
		elem = l.Args[0]
	default:
		return false
	}
	rec, ok := elem.(*ast.RecordType)
	if !ok || rec.Row != nil || len(rec.Fields) != 2 {
		return false
	}
	names := map[string]bool{}
	for _, f := range rec.Fields {
		if !isSimple(f.Type, "string") {
			return false
		}
		names[f.Name] = true
	}
	return names["name"] && names["value"]
}

// wsReqRecord builds the handler's req record from exactly its declared fields.
func wsReqRecord(r *http.Request, fields, pass []string) *eval.RecordValue {
	rec := &eval.RecordValue{Fields: make(map[string]eval.Value, len(fields))}
	for _, f := range fields {
		switch f {
		case "path":
			rec.Fields[f] = &eval.StringValue{Value: r.URL.Path}
		case "query":
			rec.Fields[f] = &eval.StringValue{Value: r.URL.RawQuery}
		case "origin":
			rec.Fields[f] = &eval.StringValue{Value: r.Header.Get("Origin")}
		case "headers":
			rec.Fields[f] = passedHeaders(r.Header, pass)
		}
	}
	return rec
}

// passedHeaders is the allowlist projection: only the named headers, one
// {name, value} per received line, in allowlist order. An absent header has
// no entry; a present empty one has value "".
func passedHeaders(h http.Header, pass []string) *eval.ListValue {
	out := &eval.ListValue{Elements: []eval.Value{}}
	for _, name := range pass {
		for _, v := range h.Values(name) {
			out.Elements = append(out.Elements, &eval.RecordValue{Fields: map[string]eval.Value{
				"name":  &eval.StringValue{Value: name},
				"value": &eval.StringValue{Value: v},
			}})
		}
	}
	return out
}
