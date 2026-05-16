package apiserver

import (
	"encoding/json"
	"log"
	"net/http"
)

// ArgSource records the provenance of resolved handler arguments.
//
// The serve-api dispatcher composes several argument sources (body, query,
// zero-value padding for arity safety). Tracking which source produced
// the final args slice lets the fallback rule key on PROVENANCE instead
// of SHAPE (e.g. "args is non-empty but synthesized"), which closes a
// class of bug where defensive zero-padding silently shadowed the query
// fallback for GET handlers with declared params.
//
// See: design_docs/planned/v0_21_0/m-serveapi-get-query-shadow.md
type ArgSource int

const (
	// ArgSourceNone means no parsing happened (e.g., @raw handlers, or
	// no declared params on a body-less request).
	ArgSourceNone ArgSource = iota
	// ArgSourceReal means at least one argument was sourced from real
	// request data (a matched body key, a positional {"args": [...]} body,
	// or query parameters).
	ArgSourceReal
	// ArgSourceZeroPadded means every argument was synthesized to satisfy
	// declared arity — the request carried no usable data for any param.
	ArgSourceZeroPadded
)

// resolveArgs is the single source of truth for JSON-body argument resolution
// in the serve-api dispatcher. It implements the precedence:
//
//  1. Parse the JSON body via parseArgsWithNamesEx (gets args + provenance).
//  2. If body provenance is not Real and query params are present, use the
//     query as a real source. This is the "GET handlers carry args in URL"
//     path. Crucially, the guard is provenance (`!= Real`), not shape
//     (`len(args) == 0`) — so zero-value padding cannot shadow query data.
//  3. If the final source is ZeroPadded, emit an INFO breadcrumb so the
//     fact that we ran a handler with synthesized inputs is visible to
//     operators (not silent).
//
// Raw and multipart handlers do NOT route through this function; their
// argument sources are unambiguous and unaffected by the shadowing class.
func resolveArgs(r *http.Request, body []byte, paramNames, paramTypes []string) ([]interface{}, ArgSource, error) {
	args, source, err := parseArgsWithNamesEx(body, paramNames, paramTypes)
	if err != nil {
		return nil, source, err
	}

	// Query fallback fires when the body did NOT produce real data.
	// This includes both "empty body, args=nil" and "empty body, args
	// zero-padded to match arity."
	if source != ArgSourceReal && len(r.URL.Query()) > 0 {
		if qa := parseQueryArgs(r.URL.Query()); len(qa) > 0 {
			return qa, ArgSourceReal, nil
		}
	}

	if source == ArgSourceZeroPadded {
		// Not behind DEBUG_*: a silent zero-pad is the bug class this design
		// closes. Operators should see WHEN a handler ran with synthesized
		// args so misconfigured callers surface quickly.
		log.Printf("[apiserver] zero-padded args for %s %s (no body, no query)", r.Method, r.URL.Path)
	}

	return args, source, nil
}

// parseArgsWithNamesEx is the provenance-aware variant of parseArgsWithNames.
// It returns the resolved args plus an ArgSource indicating where they came
// from. The plain parseArgsWithNames is preserved as a thin wrapper for
// callers and tests that don't need the source.
//
// Provenance rules:
//   - Empty body + declared params  → ZeroPadded (synthesized)
//   - {"args": [...]} body          → Real (caller provided positional data)
//   - JSON object with matched keys → Real (named binding succeeded)
//   - JSON object with NO matched keys + declared params → ZeroPadded
//   - JSON non-object body          → Real (single-arg passthrough)
//   - Empty body + no declared params → None (no args, no synthesis)
func parseArgsWithNamesEx(body []byte, paramNames, paramTypes []string) ([]interface{}, ArgSource, error) {
	if len(paramNames) == 0 {
		args, err := parseArgs(body)
		if err != nil {
			return nil, ArgSourceNone, err
		}
		if len(args) == 0 {
			return args, ArgSourceNone, nil
		}
		return args, ArgSourceReal, nil
	}

	if len(body) == 0 {
		if len(paramTypes) == 0 {
			args, err := parseArgs(body)
			return args, ArgSourceNone, err
		}
		args := make([]interface{}, len(paramNames))
		for i := range paramNames {
			if i < len(paramTypes) {
				args[i] = zeroValueForType(paramTypes[i])
			}
		}
		return args, ArgSourceZeroPadded, nil
	}

	// Structured {"args": [...]} body → Real (with possible zero-pad of
	// trailing missing positionals; the caller still supplied real data).
	var req FunctionCallRequest
	if err := json.Unmarshal(body, &req); err == nil && req.Args != nil {
		if len(req.Args) < len(paramNames) && len(paramTypes) > 0 {
			padded := make([]interface{}, len(paramNames))
			copy(padded, req.Args)
			for i := len(req.Args); i < len(paramNames); i++ {
				if i < len(paramTypes) {
					padded[i] = zeroValueForType(paramTypes[i])
				}
			}
			return padded, ArgSourceReal, nil
		}
		return req.Args, ArgSourceReal, nil
	}

	// Named binding: JSON object whose keys match declared param names.
	var obj map[string]interface{}
	bodyIsObject := json.Unmarshal(body, &obj) == nil
	if bodyIsObject && len(obj) > 0 {
		if named := parseNamedArgs(obj, paramNames, paramTypes); named != nil {
			return named, ArgSourceReal, nil
		}
	}

	// JSON object body with NO key matches but declared params → ZeroPadded.
	// (Avoids passing the raw object as a single Record arg to a handler
	// that declared positional params — would give a Record where a string
	// was expected.)
	if bodyIsObject && len(paramTypes) > 0 {
		args := make([]interface{}, len(paramNames))
		for i := range paramNames {
			if i < len(paramTypes) {
				args[i] = zeroValueForType(paramTypes[i])
			}
		}
		return args, ArgSourceZeroPadded, nil
	}

	// Non-object body (string, number, array without "args" key) → single
	// arg passthrough. Real data was supplied even if it doesn't match arity.
	args, err := parseArgs(body)
	if err != nil {
		return nil, ArgSourceNone, err
	}
	if len(args) == 0 {
		return args, ArgSourceNone, nil
	}
	return args, ArgSourceReal, nil
}
