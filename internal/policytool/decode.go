package policytool

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// DecodeRequest parses one JSON request strictly. encoding/json on its own
// matches object keys case-insensitively and lets a later duplicate key win
// (merging a duplicate object into the first), so `FLAGS` aliased `flags`
// and two `flags` objects merged (#1554) — the request the tool validated
// was not the one the caller wrote. Here every key must be spelled exactly
// as a Request field's json name, appear at most once (at the top level and
// inside flags), and the body must be exactly one object.
func DecodeRequest(raw []byte) (Request, error) {
	var req Request
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := expectDelim(dec, '{'); err != nil {
		return req, err
	}
	seen := map[string]bool{}
	// The string fields, by exact json name. A field added to Request but
	// not here is refused as unknown — fail closed (a test decodes every
	// Request field).
	fields := map[string]*string{
		"op": &req.Op, "path": &req.Path, "content": &req.Content,
		"old_text": &req.OldText, "new_text": &req.NewText, "module": &req.Module,
		"query": &req.Query, "package": &req.Package,
	}
	for dec.More() {
		key, err := objectKey(dec)
		if err != nil {
			return req, err
		}
		if seen[key] {
			return req, fmt.Errorf("request key %q appears more than once", key)
		}
		seen[key] = true
		if key == "flags" {
			flags, err := decodeFlags(dec)
			if err != nil {
				return req, err
			}
			req.Flags = flags
			continue
		}
		dst, ok := fields[key]
		if !ok {
			return req, fmt.Errorf("unknown request key %q (keys are exact and case-sensitive: flags, %s)", key, strings.Join(sortedKeys(fields), ", "))
		}
		if err := dec.Decode(dst); err != nil {
			return req, fmt.Errorf("request key %q: value must be a string: %w", key, err)
		}
	}
	if err := expectDelim(dec, '}'); err != nil {
		return req, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, fmt.Errorf("request has data after the JSON object")
	}
	return req, nil
}

// decodeFlags reads the flags object: string values, no duplicate names.
func decodeFlags(dec *json.Decoder) (map[string]string, error) {
	if err := expectDelim(dec, '{'); err != nil {
		return nil, fmt.Errorf("flags: %w", err)
	}
	out := map[string]string{}
	for dec.More() {
		name, err := objectKey(dec)
		if err != nil {
			return nil, fmt.Errorf("flags: %w", err)
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("flag %q appears more than once", name)
		}
		var v string
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("flag %q: value must be a string: %w", name, err)
		}
		out[name] = v
	}
	if err := expectDelim(dec, '}'); err != nil {
		return nil, fmt.Errorf("flags: %w", err)
	}
	return out, nil
}

func expectDelim(dec *json.Decoder, want json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("request is not a JSON object: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != want {
		return fmt.Errorf("request is not a JSON object: expected %q, got %v", want, tok)
	}
	return nil
}

func objectKey(dec *json.Decoder) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", err
	}
	key, ok := tok.(string)
	if !ok {
		return "", fmt.Errorf("expected an object key, got %v", tok)
	}
	return key, nil
}

func sortedKeys(m map[string]*string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
