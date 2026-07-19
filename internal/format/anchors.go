package format

import (
	"reflect"

	"github.com/sunholo-data/ailang/internal/ast"
)

// anchors.go walks an AST subtree collecting every ast.Pos it carries. The
// envelope uses this to compute a node's MinAnchor — the leftmost recorded token
// position over its whole subtree — because a node's own Position() is often NOT
// its textual start (a BinaryOp is positioned at its operator; a paren-wrapped
// head loses its `(`; design V16). Rather than maintain a 49-node typed visitor
// in lockstep with the AST, we reflect over exported fields: every ast.Pos found
// anywhere in the subtree is a real token position (design V17: all parser Pos
// construction copies a real token), so the minimum over them is the subtree's
// leftmost token — exactly what boundary resolution needs.
//
// Reflection here is confined to READING positions for anchoring; it never
// mutates the AST and never drives emission (emission is the exhaustive typed
// printer). It is bounded by subtree depth and runs a handful of times per file.

var posType = reflect.TypeOf(ast.Pos{})

// visitAnchors calls fn for every ast.Pos value reachable from n (including n
// itself) through exported struct fields, slices, maps, pointers, and interfaces.
func visitAnchors(n any, fn func(ast.Pos)) {
	if n == nil {
		return
	}
	seen := make(map[uintptr]bool)
	walkAnchors(reflect.ValueOf(n), fn, seen, 0)
}

// maxAnchorDepth bounds the reflective walk defensively; real ASTs are far
// shallower, but a cycle or pathological nesting must never wedge the formatter.
const maxAnchorDepth = 256

func walkAnchors(v reflect.Value, fn func(ast.Pos), seen map[uintptr]bool, depth int) {
	if depth > maxAnchorDepth || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Ptr {
			ptr := v.Pointer()
			if seen[ptr] {
				return
			}
			seen[ptr] = true
		}
		walkAnchors(v.Elem(), fn, seen, depth+1)
	case reflect.Struct:
		if v.Type() == posType {
			fn(v.Interface().(ast.Pos))
			return
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.PkgPath != "" {
				continue // unexported
			}
			walkAnchors(v.Field(i), fn, seen, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkAnchors(v.Index(i), fn, seen, depth+1)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			walkAnchors(v.MapIndex(k), fn, seen, depth+1)
		}
	}
}
