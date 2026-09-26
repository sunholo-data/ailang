package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// M-SMT-INTERP-SHOW: hygiene for desugar-synthesized builtin references.
//
// The interpolation desugar synthesizes a `show` identifier the user never
// wrote (parser_literals.go). Ordinary identifier resolution consults globalEnv
// and falls through to a local binding, so once a module's own top-level
// definitions are allowed to win — the natural fix for the separately-filed
// shadowing bug, inbox_1788928007403_88e5056a — a user's `show` would capture
// that synthesized reference and silently take over every "${x}" hole.
//
// ResolveAsBuiltin marks such a reference as hygienic: it resolves to $builtin
// regardless of what the module binds.
//
// These tests simulate the hostile precedence directly, by pointing globalEnv's
// `show` at a module-local ref, so they assert the protection TODAY rather than
// waiting for the precedence change to land.

// hijackedElaborator returns an elaborator whose globalEnv resolves `show` to a
// user module — i.e. the world as it will look once module-local definitions
// win over builtins.
func hijackedElaborator() *Elaborator {
	e := NewElaborator()
	e.AddBuiltinsToGlobalEnv()
	e.MergeGlobalEnv(map[string]core.GlobalRef{
		"show": {Module: "user/hijack", Name: "show"},
	})
	return e
}

// TestResolveAsBuiltin_SurvivesAHijackedGlobalEnv is the point of the whole
// mechanism: a marked reference must reach $builtin even when the name has been
// rebound to user code.
func TestResolveAsBuiltin_SurvivesAHijackedGlobalEnv(t *testing.T) {
	e := hijackedElaborator()

	got, err := e.normalize(&ast.Identifier{Name: "show", ResolveAsBuiltin: true})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	vg, ok := got.(*core.VarGlobal)
	if !ok {
		t.Fatalf("resolved to %T, want *core.VarGlobal", got)
	}
	if vg.Ref.Module != "$builtin" || vg.Ref.Name != "show" {
		t.Fatalf("marked identifier resolved to %s.%s, want $builtin.show — a user's own "+
			"`show` can capture the interpolation desugar", vg.Ref.Module, vg.Ref.Name)
	}
}

// TestResolveAsBuiltin_ControlUnmarkedIsCapturable is the non-vacuity control.
// Without it, the test above would still pass if ResolveAsBuiltin were a no-op
// and globalEnv simply never got hijacked.
func TestResolveAsBuiltin_ControlUnmarkedIsCapturable(t *testing.T) {
	e := hijackedElaborator()

	got, err := e.normalize(&ast.Identifier{Name: "show"})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	vg, ok := got.(*core.VarGlobal)
	if !ok {
		t.Fatalf("resolved to %T, want *core.VarGlobal", got)
	}
	if vg.Ref.Module != "user/hijack" {
		t.Fatalf("unmarked identifier resolved to %s.%s; the hijack fixture is not biting, "+
			"so the marked-case test above proves nothing", vg.Ref.Module, vg.Ref.Name)
	}
}

// TestResolveAsBuiltin_UnknownBuiltinFailsLoudly: marking a name that is not a
// registered builtin is a desugar bug. Returning a plausible-looking reference
// to a builtin that does not exist would fail much later and much less clearly.
func TestResolveAsBuiltin_UnknownBuiltinFailsLoudly(t *testing.T) {
	e := NewElaborator()
	e.AddBuiltinsToGlobalEnv()

	_, err := e.normalize(&ast.Identifier{Name: "definitelyNotABuiltin", ResolveAsBuiltin: true})
	if err == nil {
		t.Fatal("marking an unregistered name as a builtin was accepted silently")
	}
	if !strings.Contains(err.Error(), "definitelyNotABuiltin") {
		t.Errorf("error %q does not name the offending identifier", err.Error())
	}
}
