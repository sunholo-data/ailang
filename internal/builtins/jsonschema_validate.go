package builtins

import (
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"
)

type schemaResult struct {
	errors     []schemaError
	hardErrors []schemaError
	properties map[string]bool
	items      map[int]bool
}

func newSchemaResult() schemaResult {
	return schemaResult{properties: map[string]bool{}, items: map[int]bool{}}
}
func (r *schemaResult) fail(path, keyword, message string) {
	r.errors = append(r.errors, schemaError{path, keyword, message})
}
func (r *schemaResult) merge(s schemaResult) {
	for k := range s.properties {
		r.properties[k] = true
	}
	for k := range s.items {
		r.items[k] = true
	}
}
func validateCompiledSchema(p *schemaProgram, doc any) []schemaError {
	r := evaluateSchema(p.root, doc, "", nil, map[string]int{})
	return append(r.errors, r.hardErrors...)
}
func evaluateSchema(n *schemaNode, doc any, path string, scope []*schemaNode, depth map[string]int) schemaResult {
	r := newSchemaResult()
	if n == nil {
		return r
	}
	if depth[path] > 64 {
		r.hardErrors = append(r.hardErrors, schemaError{path, "$ref", "schema evaluation exceeded 64 reference hops at the same instance location"})
		return r
	}
	if b, ok := n.value.(bool); ok {
		if !b {
			r.fail(path, "false", "boolean schema rejects value")
		}
		return r
	}
	m, ok := n.value.(map[string]any)
	if !ok {
		return r
	}
	scope = append(append([]*schemaNode{}, scope...), n)
	scalarSchema(n, m, doc, path, &r)
	containerSchema(n, m, doc, path, scope, depth, &r)
	if len(r.hardErrors) > 0 {
		return r
	}
	applicatorSchema(n, m, doc, path, scope, depth, &r)
	if len(r.hardErrors) > 0 {
		return r
	}
	unevaluatedSchema(n, m, doc, path, scope, depth, &r)
	return r
}
func scalarSchema(n *schemaNode, m map[string]any, doc any, path string, r *schemaResult) {
	for _, k := range schemaKeys(m) {
		v := m[k]
		valid := true
		switch k {
		case "type":
			valid = false
			if s, ok := v.(string); ok {
				valid = schemaType(s, doc)
			} else if a, ok := v.([]any); ok {
				for _, t := range a {
					if s, ok := t.(string); ok && schemaType(s, doc) {
						valid = true
					}
				}
			}
		case "enum":
			valid = false
			for _, x := range v.([]any) {
				if reflect.DeepEqual(x, doc) {
					valid = true
					break
				}
			}
		case "const":
			valid = reflect.DeepEqual(v, doc)
		case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf":
			if x, ok := doc.(float64); ok {
				a := v.(float64)
				switch k {
				case "minimum":
					valid = x >= a
				case "maximum":
					valid = x <= a
				case "exclusiveMinimum":
					valid = x > a
				case "exclusiveMaximum":
					valid = x < a
				case "multipleOf":
					q := x / a
					valid = q == math.Trunc(q)
				}
			}
		case "minLength", "maxLength":
			if s, ok := doc.(string); ok {
				length := float64(utf8.RuneCountInString(s))
				if k == "minLength" {
					valid = length >= v.(float64)
				} else {
					valid = length <= v.(float64)
				}
			}
		case "pattern":
			if s, ok := doc.(string); ok {
				valid = n.patterns["pattern"].MatchString(s)
			}
		}
		if !valid {
			r.fail(path, k, fmt.Sprintf("value does not satisfy %s", k))
		}
	}
}
func schemaType(t string, v any) bool {
	switch t {
	case "null":
		return v == nil
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "number":
		_, ok := v.(float64)
		return ok
	case "integer":
		x, ok := v.(float64)
		return ok && math.Trunc(x) == x
	case "string":
		_, ok := v.(string)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	}
	return false
}
func schemaChild(n *schemaNode, key, member string) *schemaNode {
	return n.children[schemaPointer(key, member)]
}
