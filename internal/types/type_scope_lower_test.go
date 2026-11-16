package types

import (
	"strings"
	"testing"
)

// ReadsAsTypeVar is the one decision for a bare SimpleType that is a type
// variable (#2211): a lowercase name the scope does not declare.
func TestTypeScope_ReadsAsTypeVar(t *testing.T) {
	s := NewTypeScope("m")
	s.BindLocal("point")
	s.BindExplicit("frame", TypeRef{Module: "std/sem", Name: "frame"})
	s.AddDirect(TypeRef{Module: "q/lib", Name: "state"})
	for name, want := range map[string]bool{
		"map":   true,  // imported value
		"point": false, // declared here
		"frame": false, // imported by name
		"state": false, // exported by a direct import
		"int":   false, // built-in
		"list":  false, // module-less head
		"Foo":   false, // uppercase: a type, opaque if undeclared (#1485)
		"":      false,
	} {
		if got := s.ReadsAsTypeVar(name); got != want {
			t.Errorf("ReadsAsTypeVar(%q) = %v, want %v", name, got, want)
		}
	}
	var nilScope *TypeScope
	if !nilScope.ReadsAsTypeVar("map") || nilScope.ReadsAsTypeVar("int") {
		t.Error("nil scope: an undeclared lowercase name is a variable, a built-in is not")
	}
	if names := s.Names(); len(names) == 0 || !hasName(names, "point") || !hasName(names, "state") || !hasName(names, "int") {
		t.Errorf("Names misses scope or built-in names: %v", names)
	}
}

// TC_UNBOUND_TYPE_001 names the suggested type when there is one, and the
// generic fix otherwise.
func TestUnboundLowerTypeError_Suggestion(t *testing.T) {
	with := UnboundLowerTypeError("option", "Option").Error()
	if !strings.Contains(with, TC_UNBOUND_TYPE_001) || !strings.Contains(with, "did you mean the type `Option`?") {
		t.Errorf("with a suggestion: %s", with)
	}
	without := UnboundLowerTypeError("zz", "").Error()
	if !strings.Contains(without, "type 'zz' is not in scope") || strings.Contains(without, "did you mean") ||
		!strings.Contains(without, "import <module> (zz)") {
		t.Errorf("without a suggestion: %s", without)
	}
}

func hasName(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
