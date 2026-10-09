package apiserver

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/httpjson"
)

const responseHeadersShape = "_headers must be {x_y: string, ...} (labels remapped _ -> -) or Json (JObject of strings)"

// responseHeaders validates the complete set before the writer is touched.
// Record iteration is sorted; JObject entries retain list order (last wins).
func responseHeaders(value eval.Value) (http.Header, error) {
	headers := make(http.Header)
	rec, ok := value.(*eval.RecordValue)
	if !ok {
		return headers, nil
	}
	v, present := rec.Fields["_headers"]
	if !present {
		return headers, nil
	}
	add := func(name, value string) error {
		if !isHTTPToken(name) {
			return fmt.Errorf("%s; invalid name %q", responseHeadersShape, name)
		}
		if !validResponseHeaderValue(value) {
			return fmt.Errorf("%s; field %q contains a control character", responseHeadersShape, name)
		}
		lower := strings.ToLower(name)
		if lower == "x-elapsed-ms" || lower == "vary" || strings.HasPrefix(lower, "access-control-") {
			return fmt.Errorf("%s; field %q is owned by the server", responseHeadersShape, name)
		}
		if isFramingHeader(lower) {
			return fmt.Errorf("%s; field %q is a message-framing header owned by the server", responseHeadersShape, name)
		}
		headers.Set(name, value)
		return nil
	}
	switch h := v.(type) {
	case *eval.RecordValue:
		keys := make([]string, 0, len(h.Fields))
		for k := range h.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sv, ok := h.Fields[k].(*eval.StringValue)
			if !ok {
				return nil, fmt.Errorf("%s; field %q is not string", responseHeadersShape, k)
			}
			if err := add(strings.ReplaceAll(k, "_", "-"), sv.Value); err != nil {
				return nil, err
			}
		}
	case *eval.TaggedValue:
		if h.CtorName != "JObject" || len(h.Fields) != 1 {
			return nil, fmt.Errorf("%s; got %s", responseHeadersShape, h.CtorName)
		}
		entries, ok := h.Fields[0].(*eval.ListValue)
		if !ok {
			return nil, fmt.Errorf("%s; JObject entries must be a list", responseHeadersShape)
		}
		for i, e := range entries.Elements {
			entry, ok := e.(*eval.RecordValue)
			if !ok {
				return nil, fmt.Errorf("%s; JObject entry %d must be a key/value record", responseHeadersShape, i)
			}
			key, ok := entry.Fields["key"].(*eval.StringValue)
			if !ok {
				return nil, fmt.Errorf("%s; JObject entry %d key must be string", responseHeadersShape, i)
			}
			tagged, ok := entry.Fields["value"].(*eval.TaggedValue)
			if !ok || tagged.CtorName != "JString" || len(tagged.Fields) != 1 {
				return nil, fmt.Errorf("%s; JObject field %q is not JString", responseHeadersShape, key.Value)
			}
			sv, ok := tagged.Fields[0].(*eval.StringValue)
			if !ok {
				return nil, fmt.Errorf("%s; JObject field %q is not JString(string)", responseHeadersShape, key.Value)
			}
			if err := add(key.Value, sv.Value); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("%s; got %T", responseHeadersShape, v)
	}
	return headers, nil
}

// framingHeaders frame the HTTP message itself. A route that set them could
// emit duplicate Transfer-Encoding headers or a Content-Length that truncates
// the body, so they are refused like the D7 server-owned set.
var framingHeaders = map[string]bool{
	"content-length":    true,
	"transfer-encoding": true,
	"connection":        true,
	"keep-alive":        true,
	"upgrade":           true,
	"trailer":           true,
	"te":                true,
}

// isFramingHeader reports whether a lower-cased name is a framing or
// hop-by-hop header (including every proxy-* name).
func isFramingHeader(lower string) bool {
	return framingHeaders[lower] || strings.HasPrefix(lower, "proxy-")
}

func withoutResponseHeaders(value eval.Value) eval.Value {
	rec, ok := value.(*eval.RecordValue)
	if !ok {
		return value
	}
	if _, has := rec.Fields["_headers"]; !has {
		return value
	}
	copy := &eval.RecordValue{Fields: make(map[string]eval.Value, len(rec.Fields)-1)}
	for k, v := range rec.Fields {
		if k != "_headers" {
			copy.Fields[k] = v
		}
	}
	return copy
}

func setResponseHeaders(w http.ResponseWriter, headers http.Header) {
	for k, values := range headers {
		w.Header()[k] = append([]string(nil), values...)
	}
}

func writeResponseHeaderError(w http.ResponseWriter, err error, elapsed int64) {
	log.Printf("ERROR: response headers: %v", err)
	httpjson.Write(w, http.StatusInternalServerError, FunctionCallResponse{Error: err.Error(), ElapsedMs: elapsed})
}
