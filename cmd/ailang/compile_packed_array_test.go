package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestCompilePackedArrayTypeMapping(t *testing.T) {
	for _, tc := range []struct{ element, want string }{{"float", "FloatArrayVal"}, {"int", "ArrayVal"}} {
		typ := &ast.ArrayType{Element: &ast.SimpleType{Name: tc.element}}
		if got := ailangTypeToGo(typ); got != tc.want {
			t.Errorf("Array[%s] -> %q", tc.element, got)
		}
		if got := ailangTypeToGoWithValueRecords(typ, nil); got != tc.want {
			t.Errorf("Array[%s] value-record mapping -> %q", tc.element, got)
		}
	}
}
