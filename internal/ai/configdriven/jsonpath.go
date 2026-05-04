// Package configdriven implements a generic AI provider whose behaviour is
// driven by an [[ai_provider]] block in a package's ailang.toml manifest.
// See design_docs/planned/v0_15_0/m-ai-provider-config.md.
package configdriven

import (
	"fmt"
	"strconv"
	"strings"
)

// extractPath walks a parsed-JSON value following a minimal JSONPath subset:
//
//	$            — root
//	$.a          — object field "a"
//	$.a.b        — nested object fields
//	$[0]         — array index 0
//	$.a[0].b     — mixed
//
// No filters, no wildcards, no recursive descent, no expressions. Sufficient
// for the response_path / error_path / streaming.delta_path use cases in
// the [[ai_provider]] schema. Returns nil and a non-nil error if the path
// does not resolve; returns nil and nil if the path resolves to a JSON null.
func extractPath(root any, path string) (any, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	if path == "$" {
		return root, nil
	}
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("path must start with $: %q", path)
	}
	rest := path[1:]
	cur := root

	for len(rest) > 0 {
		switch rest[0] {
		case '.':
			// Field access: .field
			rest = rest[1:]
			end := strings.IndexAny(rest, ".[")
			var field string
			if end == -1 {
				field, rest = rest, ""
			} else {
				field, rest = rest[:end], rest[end:]
			}
			if field == "" {
				return nil, fmt.Errorf("empty field name in path %q", path)
			}
			obj, ok := cur.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("path %q expected object at field %q, got %T", path, field, cur)
			}
			val, exists := obj[field]
			if !exists {
				return nil, fmt.Errorf("path %q missing field %q", path, field)
			}
			cur = val
		case '[':
			// Array index: [N]
			end := strings.Index(rest, "]")
			if end == -1 {
				return nil, fmt.Errorf("unclosed [ in path %q", path)
			}
			idxStr := rest[1:end]
			rest = rest[end+1:]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("path %q has non-integer index %q", path, idxStr)
			}
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("path %q expected array at index %d, got %T", path, idx, cur)
			}
			if idx < 0 || idx >= len(arr) {
				return nil, fmt.Errorf("path %q index %d out of bounds (len=%d)", path, idx, len(arr))
			}
			cur = arr[idx]
		default:
			return nil, fmt.Errorf("unexpected character %q in path %q", rest[0], path)
		}
	}
	return cur, nil
}

// extractString resolves the path against root and asserts the result is a
// string. Empty result + nil error means the path resolved to JSON null or
// an empty string; both are mapped to "" with no error so callers can
// uniformly check for "" rather than distinguishing.
func extractString(root any, path string) (string, error) {
	val, err := extractPath(root, path)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("path %q resolved to %T, want string", path, val)
	}
	return s, nil
}
