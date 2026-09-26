package elaborate

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestBangElaboratesToNot pins `!e` as the same boolean negation as `not e`. The parser
// keeps `!` in the surface AST (so `ailang fmt` preserves the user's spelling), but no
// stage after elaboration knows a "!" operator: before this mapping `!done` became a
// generic core.UnOp and failed type checking with "unknown unary operator: !", although
// the teaching prompt promises "both `not` and `!` work" (2026-09-22).
func TestBangElaboratesToNot(t *testing.T) {
	for _, src := range []string{
		"module test\n\nfunc f(done: bool) -> bool = !done\n",
		"module test\n\nfunc f(done: bool) -> bool = not done\n",
	} {
		expr := elaborateExpr(t, src)
		var sawNot, sawRawBang bool
		walkCore(reflect.ValueOf(expr), func(v reflect.Value) {
			switch n := v.Interface().(type) {
			case *core.Intrinsic:
				if n.Op == core.OpNot {
					sawNot = true
				}
			case *core.UnOp:
				if n.Op == "!" {
					sawRawBang = true
				}
			}
		})
		if sawRawBang || !sawNot {
			t.Errorf("%q: want an OpNot intrinsic and no raw \"!\" UnOp (sawNot=%v sawRawBang=%v)", src, sawNot, sawRawBang)
		}
	}
}

// walkCore visits every pointer reachable from v (core trees are plain structs,
// slices and interfaces, so reflection is enough for a test).
func walkCore(v reflect.Value, visit func(reflect.Value)) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			walkCore(v.Elem(), visit)
		}
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.CanInterface() {
			visit(v)
		}
		walkCore(v.Elem(), visit)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				walkCore(v.Field(i), visit)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkCore(v.Index(i), visit)
		}
	}
}
