package builtins

import "strconv"

// Annotation sets refer only to the current instance location. Child-instance
// annotations never leak into their parent's unevaluated checks.
func unevaluatedSchema(n *schemaNode, m map[string]any, doc any, path string, scope []*schemaNode, depth map[string]int, r *schemaResult) {
	if obj, ok := doc.(map[string]any); ok {
		if child := n.children["unevaluatedProperties"]; child != nil {
			for _, name := range schemaKeys(obj) {
				if !r.properties[name] {
					s := evaluateSchema(child, obj[name], schemaPointer(path, name), scope, depth)
					r.errors = append(r.errors, s.errors...)
					r.hardErrors = append(r.hardErrors, s.hardErrors...)
					r.properties[name] = true
				}
			}
		}
	}
	if array, ok := doc.([]any); ok {
		if child := n.children["unevaluatedItems"]; child != nil {
			for i, value := range array {
				if !r.items[i] {
					s := evaluateSchema(child, value, schemaPointer(path, strconv.Itoa(i)), scope, depth)
					r.errors = append(r.errors, s.errors...)
					r.hardErrors = append(r.hardErrors, s.hardErrors...)
					r.items[i] = true
				}
			}
		}
	}
}
