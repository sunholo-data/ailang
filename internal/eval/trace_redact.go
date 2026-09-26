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
// and println are unchanged.

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
	w := boundedWriter{buf: make([]byte, 0, min(maxBytes, 256)), budget: maxBytes, redact: true}
	w.render(v)
	if w.overflow == 0 {
		return string(w.buf)
	}
	return ShowBoundedOverflow(w.buf, w.overflow)
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
