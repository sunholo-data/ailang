package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestLocalResolutionHintFamily is the truth-telling residual diagnostic for the
// "resolution diverges by syntactic position" family (#323 → #327 → #366). The
// known members are fixed (expression positions via #327; the module-let/letrec
// DECL class via #366, m-module-let-func-resolution). This clause is the residual
// net for any not-yet-discovered position: when "undefined variable: X" is raised
// for X that IS a module-level function, the message must (a) tell the truth
// (X exists), (b) carry the VERIFIED workaround (declare X as a `func`), and
// (c) NOT cite the now-closed #327 as a live bug.
//
// This is deliberately unit-level: the DIAGNOSTIC contract lives entirely in
// inferVar's error construction.
func TestLocalResolutionHintFamily(t *testing.T) {
	// A bare reference to an undefined name drives inferVar's error path.
	undefVar := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "extendFm"}

	t.Run("known local func gets the truthful residual clause", func(t *testing.T) {
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
		// Must cite the LIVE issue #366, never the closed #327.
		if !strings.Contains(msg, "#366") {
			t.Errorf("expected the live #366 reference in the message, got: %s", msg)
		}
		if strings.Contains(msg, "#327") {
			t.Errorf("must NOT cite the closed #327 as a live bug, got: %s", msg)
		}
		// The workaround must be the VERIFIED one (declare it as a func), not the
		// old no-op "bind it with let first".
		if !strings.Contains(msg, "declare extendFm as a `func`") {
			t.Errorf("expected the verified func workaround in the message, got: %s", msg)
		}
		if strings.Contains(msg, "bind it with let first") {
			t.Errorf("must NOT carry the old no-op let workaround, got: %s", msg)
		}
		if !strings.Contains(msg, "defined in this module but not resolvable in this position") {
			t.Errorf("expected the truth-telling clause, got: %s", msg)
		}
	})

	t.Run("genuinely undefined var does NOT get the residual clause", func(t *testing.T) {
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
		if strings.Contains(msg, "#366") {
			t.Errorf("genuine undefined var must NOT carry the residual clause, got: %s", msg)
		}
	})

	t.Run("nil moduleFuncNames is safe and adds no clause", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		// SetModuleFuncNames never called: moduleFuncNames is nil.

		_, _, err := tc.CheckCoreExpr(undefVar, NewTypeEnv())
		if err == nil {
			t.Fatal("expected an undefined-variable error, got nil")
		}
		if strings.Contains(err.Error(), "#366") {
			t.Errorf("nil moduleFuncNames must not add the residual clause, got: %s", err.Error())
		}
	})
}
