package eval

import (
	"math"
	"strings"
)

// Trace redaction of credentials (M-SERVEAPI-WS-BRIDGE G7).
//
// Effect arguments, builtin arguments and function-call arguments are all
// rendered into the trace through one renderer — ShowTraceBounded, reached
// from EffContext.RenderTraceValue and the typed evaluator's boundedShow.
// Before this, `connect(url, {headers: [{name: "Authorization", value:
// "Bearer …"}]})` wrote the bearer into the trace verbatim at every tier,
// because the sensitive-name patterns covered environment variable names
// only. The renderer now withholds, wherever they appear in a value:
//
//   - the value of a header-shaped record {name|key: <sensitive>, value: v}
//     (std/stream headers, std/net headers, @raw request JObject entries);
//   - the second element of a pair (<sensitive>, v);
//   - a record field whose name is sensitive ({authorization: v});
//   - any string that is itself an HTTP credential ("Bearer …", "Basic …").
//
// The redaction applies to the trace rendering only: program values, show()
// and println are unchanged. Apart from the withheld values a trace spells a
// value exactly as show() does (same walk, show.go).

// RedactedMarker replaces a withheld value in a trace rendering.
const RedactedMarker = "[REDACTED]"

// sensitiveNames are header / field names whose values are credentials,
// compared after lower-casing and mapping '_' to '-'. Deliberately exact
// names rather than substrings: a substring rule on "token" would also hide
// {input_tokens: 50} in every AI trace.
var sensitiveNames = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"cookie":              true,
	"set-cookie":          true,
	"x-api-key":           true,
	"api-key":             true,
	"apikey":              true,
	"x-goog-api-key":      true,
	"x-auth-token":        true,
	"access-token":        true,
	"refresh-token":       true,
	"id-token":            true,
	"client-secret":       true,
	"password":            true,
	"secret":              true,
	"bearer":              true,
}

// IsSensitiveHeaderName reports whether name (a header or record field
// name) carries a credential.
func IsSensitiveHeaderName(name string) bool {
	return sensitiveNames[strings.ReplaceAll(strings.ToLower(name), "_", "-")]
}

// isCredentialString reports whether s is itself an HTTP credential value.
func isCredentialString(s string) bool {
	for _, scheme := range []string{"bearer ", "basic "} {
		if len(s) > len(scheme) && strings.EqualFold(s[:len(scheme)], scheme) {
			return true
		}
	}
	return false
}

// ShowTraceBounded renders v for a trace: ShowBounded's bounded rendering
// with credentials withheld. A budget <= 0 means unbounded.
func ShowTraceBounded(v Value, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = math.MaxInt
	}
	return showBoundedWith(v, maxBytes, inspectTrace)
}

// redactedValue stands in for a withheld value; inspectTrace renders it as the
// bare RedactedMarker (never quoted, so it cannot be mistaken for a string).
type redactedValue struct{}

func (redactedValue) Type() string   { return "redacted" }
func (redactedValue) String() string { return RedactedMarker }

// inspectTrace is inspectShow with credentials replaced by redactedValue.
func inspectTrace(x Value) ShowNode[Value] {
	switch val := x.(type) {
	case redactedValue:
		return ShowNode[Value]{Text: RedactedMarker}
	case *StringValue:
		if isCredentialString(val.Value) {
			return ShowNode[Value]{Text: RedactedMarker}
		}
	case *TupleValue:
		if len(val.Elements) == 2 {
			if name, ok := val.Elements[0].(*StringValue); ok && IsSensitiveHeaderName(name.Value) {
				return ShowNode[Value]{Kind: ShowTuple, Items: []Value{name, redactedValue{}}}
			}
		}
	case *RecordValue:
		redactValue := false
		if name, ok := headerEntryName(val); ok && IsSensitiveHeaderName(name) {
			redactValue = true
		}
		names := make([]string, 0, len(val.Fields))
		items := make([]Value, 0, len(val.Fields))
		for k, fv := range val.Fields {
			if (redactValue && k == "value") || IsSensitiveHeaderName(k) {
				fv = redactedValue{}
			}
			names = append(names, k)
			items = append(items, fv)
		}
		return ShowNode[Value]{Kind: ShowRecord, Names: names, Items: items}
	}
	return inspectShow(x)
}

// headerEntryName returns the name of a header-shaped record
// ({name|key: string, value: _}) and whether it is one.
func headerEntryName(r *RecordValue) (string, bool) {
	if _, ok := r.Fields["value"]; !ok || len(r.Fields) != 2 {
		return "", false
	}
	for _, k := range []string{"name", "key"} {
		if sv, ok := r.Fields[k].(*StringValue); ok {
			return sv.Value, true
		}
	}
	return "", false
}
