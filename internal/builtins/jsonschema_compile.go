package builtins

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type schemaError struct{ Path, Keyword, Message string }
type schemaNode struct {
	value         any
	path          string
	children      map[string]*schemaNode
	patterns      map[string]*regexp.Regexp
	ref           *schemaNode
	dynamicRef    string
	dynamicAnchor string
	resource      *schemaNode
}
type schemaProgram struct {
	root            *schemaNode
	nodes           map[string]*schemaNode
	anchors         map[string]*schemaNode
	resourceAnchors map[*schemaNode]map[string]*schemaNode
}

func schemaPointer(path, key string) string {
	return path + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}
func schemaKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// compileSchema audits all schema locations, including branches never evaluated.
// Annotation values are data and deliberately do not become schema nodes.
func compileSchema(value any, strict bool) (*schemaProgram, []schemaError) {
	p := &schemaProgram{nodes: map[string]*schemaNode{}, anchors: map[string]*schemaNode{}, resourceAnchors: map[*schemaNode]map[string]*schemaNode{}}
	errs := []schemaError{}
	add := func(path, key, msg string) { errs = append(errs, schemaError{path, key, "schema: " + msg}) }
	var walk func(any, string, *schemaNode) *schemaNode
	walk = func(v any, path string, resource *schemaNode) *schemaNode {
		n := &schemaNode{value: v, path: path, children: map[string]*schemaNode{}, patterns: map[string]*regexp.Regexp{}}
		p.nodes[path] = n
		if obj, ok := v.(map[string]any); ok {
			if _, hasID := obj["$id"]; hasID {
				resource = n
			}
		}
		if resource == nil {
			resource = n
		}
		n.resource = resource
		if p.resourceAnchors[resource] == nil {
			p.resourceAnchors[resource] = map[string]*schemaNode{}
		}
		if _, ok := v.(bool); ok {
			return n
		}
		obj, ok := v.(map[string]any)
		if !ok {
			add(path, "schema", "expected an object or boolean")
			return n
		}
		for _, k := range schemaKeys(obj) {
			val := obj[k]
			kp := schemaPointer(path, k)
			child := func(v any, rel string) {
				if _, ok := v.(bool); !ok {
					if _, ok := v.(map[string]any); !ok {
						add(path+"/"+rel, k, "expected an object or boolean schema")
						return
					}
				}
				n.children[rel] = walk(v, path+"/"+rel, resource)
			}
			switch k {
			case "$defs", "properties", "patternProperties", "dependentSchemas":
				m, ok := val.(map[string]any)
				if !ok {
					add(kp, k, "expected an object of schemas")
					continue
				}
				for _, name := range schemaKeys(m) {
					rel := schemaPointer(k, name)
					child(m[name], rel)
					if k == "patternProperties" {
						r, e := regexp.Compile(name)
						if e != nil {
							add(schemaPointer(kp, name), k, fmt.Sprintf("pattern %q is unsupported by RE2: %v", name, e))
						} else {
							n.patterns[schemaPointer(k, name)] = r
						}
					}
				}
			case "additionalProperties", "propertyNames", "items", "contains", "not", "if", "then", "else", "unevaluatedProperties", "unevaluatedItems":
				if k == "items" {
					if _, ok := val.([]any); ok {
						add(kp, k, "array-form items is not Draft 2020-12; use prefixItems")
						continue
					}
				}
				child(val, k)
			case "allOf", "anyOf", "oneOf", "prefixItems":
				a, ok := val.([]any)
				if !ok || len(a) == 0 {
					add(kp, k, "expected a nonempty array of schemas")
					continue
				}
				for i, v := range a {
					child(v, fmt.Sprintf("%s/%d", k, i))
				}
			case "type":
				valid := func(v any) bool {
					s, ok := v.(string)
					return ok && strings.Contains("|null|boolean|object|array|number|string|integer|", "|"+s+"|")
				}
				if a, ok := val.([]any); ok {
					seen := map[any]bool{}
					bad := len(a) == 0
					for _, v := range a {
						if !valid(v) {
							bad = true
							continue
						}
						if seen[v] {
							bad = true
						}
						seen[v] = true
					}
					if bad {
						add(kp, k, "expected unique JSON type names")
					}
				} else if !valid(val) {
					add(kp, k, "expected a JSON type name or nonempty unique array")
				}
			case "required":
				if !schemaStringArray(val) {
					add(kp, k, "expected an array of unique strings")
				}
			case "dependentRequired":
				m, ok := val.(map[string]any)
				if !ok {
					add(kp, k, "expected an object of string arrays")
					continue
				}
				for _, name := range schemaKeys(m) {
					if !schemaStringArray(m[name]) {
						add(schemaPointer(kp, name), k, "expected an array of unique strings")
					}
				}
			case "enum":
				a, ok := val.([]any)
				if !ok || len(a) == 0 {
					add(kp, k, "expected a nonempty array")
				} else {
					for i, v := range a {
						for _, other := range a[:i] {
							if reflect.DeepEqual(v, other) {
								add(kp, k, "enum values must be unique")
								break
							}
						}
					}
				}
			case "examples":
				if _, ok := val.([]any); !ok {
					add(kp, k, "expected an array")
				}
			case "const", "default": // arbitrary JSON annotation/constant payload
			case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf":
				f, ok := val.(float64)
				if !ok || math.IsNaN(f) || math.IsInf(f, 0) || (k == "multipleOf" && f <= 0) {
					add(kp, k, "expected a finite number (multipleOf must be positive)")
				}
			case "minLength", "maxLength", "minItems", "maxItems", "minContains", "maxContains", "minProperties", "maxProperties":
				f, ok := val.(float64)
				if !ok || math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || math.Trunc(f) != f {
					add(kp, k, "expected a nonnegative integer")
				}
			case "uniqueItems", "deprecated", "readOnly", "writeOnly":
				if _, ok := val.(bool); !ok {
					add(kp, k, "expected a boolean")
				}
			case "$schema", "$id", "$ref", "$dynamicRef", "$anchor", "$dynamicAnchor", "$comment", "title", "description", "format", "pattern":
				s, ok := val.(string)
				if !ok {
					add(kp, k, "expected a string")
					continue
				}
				if k == "pattern" {
					r, e := regexp.Compile(s)
					if e != nil {
						add(kp, k, fmt.Sprintf("pattern %q is unsupported by RE2: %v", s, e))
					} else {
						n.patterns[k] = r
					}
				}
				if k == "$schema" && s != "https://json-schema.org/draft/2020-12/schema" {
					add(kp, k, "only https://json-schema.org/draft/2020-12/schema is supported")
				}
				if k == "$id" {
					auditSchemaID(s, kp, add)
				}
				if k == "$anchor" || k == "$dynamicAnchor" {
					if !schemaAnchorPattern.MatchString(s) {
						add(kp, k, "invalid anchor name")
					} else if _, exists := p.resourceAnchors[resource][s]; exists && p.resourceAnchors[resource][s] != n {
						add(kp, k, "duplicate anchor "+s)
					} else {
						p.resourceAnchors[resource][s] = n
						if resource.path == "" {
							p.anchors[s] = n
						}
					}
					if k == "$dynamicAnchor" {
						n.dynamicAnchor = s
					}
				}
			case "$vocabulary":
				m, ok := val.(map[string]any)
				if !ok {
					add(kp, k, "expected an object of vocabulary booleans")
					continue
				}
				for _, uri := range schemaKeys(m) {
					if _, ok := m[uri].(bool); !ok {
						add(schemaPointer(kp, uri), k, "expected a boolean")
					}
					if strict && !schemaVocabularies[uri] {
						add(schemaPointer(kp, uri), k, "unsupported vocabulary "+uri)
					}
				}
			default:
				if strict {
					add(kp, k, "unknown keyword in supported Draft 2020-12 vocabulary; use validateLenient to skip unknown keywords")
				}
			}
		}
		return n
	}
	p.root = walk(value, "", nil)
	p.resolveReferences(add)
	return p, errs
}
func schemaStringArray(v any) bool {
	a, ok := v.([]any)
	if !ok {
		return false
	}
	seen := map[string]bool{}
	for _, v := range a {
		s, ok := v.(string)
		if !ok || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}
