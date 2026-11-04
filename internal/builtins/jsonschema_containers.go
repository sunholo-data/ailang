package builtins

import (
	"fmt"
	"reflect"
	"strconv"
)

func containerSchema(n *schemaNode, m map[string]any, doc any, path string, scope []*schemaNode, depth map[string]int, r *schemaResult) {
	apply := func(child *schemaNode, value any, p string) {
		s := evaluateSchema(child, value, p, scope, depth)
		r.errors = append(r.errors, s.errors...)
		r.hardErrors = append(r.hardErrors, s.hardErrors...)
	}
	if obj, ok := doc.(map[string]any); ok {
		for _, k := range schemaKeys(m) {
			v := m[k]
			switch k {
			case "minProperties", "maxProperties":
				count := float64(len(obj))
				if (k == "minProperties" && count < v.(float64)) || (k == "maxProperties" && count > v.(float64)) {
					r.fail(path, k, "object property count outside limit")
				}
			case "required":
				for _, name := range v.([]any) {
					if _, ok := obj[name.(string)]; !ok {
						r.fail(path, k, "missing required property "+name.(string))
					}
				}
			case "dependentRequired":
				for _, name := range schemaKeys(v.(map[string]any)) {
					if _, ok := obj[name]; ok {
						for _, dep := range v.(map[string]any)[name].([]any) {
							if _, ok := obj[dep.(string)]; !ok {
								r.fail(path, k, "property "+name+" requires "+dep.(string))
							}
						}
					}
				}
			}
		}
		patterns, _ := m["patternProperties"].(map[string]any)
		for _, name := range schemaKeys(obj) {
			value := obj[name]
			p := schemaPointer(path, name)
			matched := false
			if child := schemaChild(n, "properties", name); child != nil {
				apply(child, value, p)
				matched = true
			}
			for _, pattern := range schemaKeys(patterns) {
				if re := n.patterns[schemaPointer("patternProperties", pattern)]; re != nil && re.MatchString(name) {
					apply(schemaChild(n, "patternProperties", pattern), value, p)
					matched = true
				}
			}
			if matched {
				r.properties[name] = true
			} else if child := n.children["additionalProperties"]; child != nil {
				apply(child, value, p)
				r.properties[name] = true
			}
			if child := n.children["propertyNames"]; child != nil {
				apply(child, name, p)
			}
		}
	}
	if array, ok := doc.([]any); ok {
		prefix, _ := m["prefixItems"].([]any)
		for i, value := range array {
			child := (*schemaNode)(nil)
			if i < len(prefix) {
				child = schemaChild(n, "prefixItems", strconv.Itoa(i))
			} else {
				child = n.children["items"]
			}
			if child != nil {
				apply(child, value, schemaPointer(path, strconv.Itoa(i)))
				r.items[i] = true
			}
		}
		for _, k := range []string{"minItems", "maxItems"} {
			if v, ok := m[k].(float64); ok && ((k == "minItems" && float64(len(array)) < v) || (k == "maxItems" && float64(len(array)) > v)) {
				r.fail(path, k, "array length outside limit")
			}
		}
		if unique, _ := m["uniqueItems"].(bool); unique {
			duplicate := false
			for i := range array {
				for j := 0; j < i; j++ {
					if reflect.DeepEqual(array[i], array[j]) {
						duplicate = true
					}
				}
			}
			if duplicate {
				r.fail(path, "uniqueItems", "array contains duplicate values")
			}
		}
		if child := n.children["contains"]; child != nil {
			matches := []int{}
			for i, value := range array {
				s := evaluateSchema(child, value, schemaPointer(path, strconv.Itoa(i)), scope, depth)
				r.hardErrors = append(r.hardErrors, s.hardErrors...)
				if len(s.errors) == 0 {
					matches = append(matches, i)
				}
			}
			min := 1.0
			if v, ok := m["minContains"].(float64); ok {
				min = v
			}
			max, limited := m["maxContains"].(float64)
			if float64(len(matches)) < min || (limited && float64(len(matches)) > max) {
				r.fail(path, "contains", fmt.Sprintf("contains matched %d items", len(matches)))
			} else {
				for _, i := range matches {
					r.items[i] = true
				}
			}
		}
	}
}
