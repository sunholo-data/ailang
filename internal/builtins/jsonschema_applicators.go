package builtins

import (
	"net/url"
	"strconv"
	"strings"
)

func applicatorSchema(n *schemaNode, m map[string]any, doc any, path string, scope []*schemaNode, depth map[string]int, r *schemaResult) {
	apply := func(child *schemaNode) schemaResult {
		s := evaluateSchema(child, doc, path, scope, depth)
		r.hardErrors = append(r.hardErrors, s.hardErrors...)
		return s
	}
	for _, keyword := range []string{"$ref", "$dynamicRef"} {
		target := n.ref
		if keyword == "$dynamicRef" {
			target = n.children["$dynamicRef"]
		}
		if target == nil {
			continue
		}
		fragment, _ := url.PathUnescape(strings.TrimPrefix(n.dynamicRef, "#"))
		if keyword == "$dynamicRef" && fragment != "" && !strings.HasPrefix(fragment, "/") && target.dynamicAnchor == fragment {
			for _, outer := range scope {
				if outer.resource == nil {
					continue
				}
				if candidate := outer.resource.anchors[fragment]; candidate != nil && candidate.dynamicAnchor == fragment {
					target = candidate
					break
				}
			}
		}
		depth[path]++
		s := apply(target)
		depth[path]--
		if len(r.hardErrors) > 0 {
			return
		}
		r.errors = append(r.errors, s.errors...)
		if len(s.errors) == 0 {
			r.merge(s)
		}
	}
	for _, k := range []string{"allOf", "anyOf", "oneOf"} {
		if branches, ok := m[k].([]any); ok {
			success := []schemaResult{}
			for i := range branches {
				s := apply(schemaChild(n, k, strconv.Itoa(i)))
				if len(r.hardErrors) > 0 {
					return
				}
				if len(s.errors) == 0 {
					success = append(success, s)
				} else if k == "allOf" {
					r.errors = append(r.errors, s.errors...)
				}
			}
			valid := true
			switch k {
			case "allOf":
				valid = len(success) == len(branches)
			case "anyOf":
				valid = len(success) > 0
			case "oneOf":
				valid = len(success) == 1
			}
			if valid {
				for _, s := range success {
					r.merge(s)
				}
			} else {
				r.fail(path, k, "applicator match count is invalid")
			}
		}
	}
	if child := n.children["not"]; child != nil {
		if len(apply(child).errors) == 0 {
			r.fail(path, "not", "value matches excluded schema")
		}
	}
	if child := n.children["if"]; child != nil {
		predicate := apply(child)
		if len(r.hardErrors) > 0 {
			return
		}
		branch := "else"
		if len(predicate.errors) == 0 {
			branch = "then"
			r.merge(predicate)
		}
		if selected := n.children[branch]; selected != nil {
			s := apply(selected)
			r.errors = append(r.errors, s.errors...)
			if len(s.errors) == 0 {
				r.merge(s)
			}
		}
	}
	if obj, ok := doc.(map[string]any); ok {
		if deps, ok := m["dependentSchemas"].(map[string]any); ok {
			for _, name := range schemaKeys(deps) {
				if _, present := obj[name]; present {
					s := apply(schemaChild(n, "dependentSchemas", name))
					r.errors = append(r.errors, s.errors...)
					if len(s.errors) == 0 {
						r.merge(s)
					}
				}
			}
		}
	}
}
