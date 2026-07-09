package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestLocalResolutionHint327 is the interim truth-telling diagnostic for #327:
// when "undefined variable: X" is raised for X that IS a module-level function of
// the current module, the message must carry the truth (X exists) plus the
// hoist-to-let workaround — instead of flatly (and misleadingly) claiming X is
// undefined.
//
// This is deliberately unit-level: the real #327 bug only reproduces under
// specific cross-module record-update conditions, but the DIAGNOSTIC contract is
// what this sprint ships, and that contract lives entirely in inferVar's error
// construction. When the resolver fix lands, localResolutionHint becomes dead
// code and both it and this test are removed.
func TestLocalResolutionHint327(t *testing.T) {
	// A bare reference to an undefined name drives inferVar's error path.
	undefVar := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "extendFm"}

	t.Run("known local func gets the #327 clause", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		tc.SetModuleFuncNames(map[string]bool{"extendFm": true})

		_, _, err := tc.CheckCoreExpr(undefVar, NewTypeEnv())
		if err == nil {
			t.Fatal("expected an undefined-variable error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "undefined variable: extendFm") {
			t.Errorf("expected the base undefined-variable message, got: %s", msg)
		}
		if !strings.Contains(msg, "known bug #327") {
			t.Errorf("expected the #327 reference in the message, got: %s", msg)
		}
		if !strings.Contains(msg, "workaround: bind it with let first") {
			t.Errorf("expected the hoist-to-let workaround in the message, got: %s", msg)
		}
		if !strings.Contains(msg, "defined in this module but not resolvable in this position") {
			t.Errorf("expected the truth-telling clause, got: %s", msg)
		}
	})

	t.Run("genuinely undefined var does NOT get the #327 clause", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		// moduleFuncNames does NOT contain "extendFm".
		tc.SetModuleFuncNames(map[string]bool{"somethingElse": true})

		_, _, err := tc.CheckCoreExpr(undefVar, NewTypeEnv())
		if err == nil {
			t.Fatal("expected an undefined-variable error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "undefined variable: extendFm") {
			t.Errorf("expected the base undefined-variable message, got: %s", msg)
		}
		if strings.Contains(msg, "#327") {
			t.Errorf("genuine undefined var must NOT carry the #327 clause, got: %s", msg)
		}
	})

	t.Run("nil moduleFuncNames is safe and adds no clause", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		// SetModuleFuncNames never called: moduleFuncNames is nil.

		_, _, err := tc.CheckCoreExpr(undefVar, NewTypeEnv())
		if err == nil {
			t.Fatal("expected an undefined-variable error, got nil")
		}
		if strings.Contains(err.Error(), "#327") {
			t.Errorf("nil moduleFuncNames must not add the #327 clause, got: %s", err.Error())
		}
	})
}
