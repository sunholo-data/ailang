package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// TestCompile_PureUnportedBuiltinSaysPure is #1447: a pure registry builtin the
// VM cannot run must be reported as pure, with the reason — never as an
// "effectful builtin … Phase 2E". __map_size stays unported (no VM Map value).
func TestCompile_PureUnportedBuiltinSaysPure(t *testing.T) {
	reason := evalReasonOf(t, "__map_size")
	if !strings.Contains(reason, `pure builtin "__map_size" has no VM implementation (map-value`) {
		t.Errorf("EvalReason = %q, want the pure-unported message with its map-value reason", reason)
	}
	if strings.Contains(reason, "effectful") {
		t.Errorf("EvalReason calls a pure builtin effectful: %q", reason)
	}
}

// TestCompile_EffectfulBuiltinStillPhase2E pins the other side: effectful
// builtins keep the Phase 2E message.
func TestCompile_EffectfulBuiltinStillPhase2E(t *testing.T) {
	reason := evalReasonOf(t, "__io_println")
	if !strings.Contains(reason, `effectful builtin "__io_println" not yet wired (Phase 2E)`) {
		t.Errorf("EvalReason = %q, want the Phase 2E effectful message", reason)
	}
}

func evalReasonOf(t *testing.T, builtin string) string {
	t.Helper()
	img, err := Compile(&stmt.Program{FuncDecls: []stmt.FuncDecl{{
		Name:     "f",
		Exported: true,
		Return:   stmt.BuiltinCall{Name: builtin, Args: []stmt.Expr{stmt.LitInt{Value: 1}}},
	}}})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	for _, p := range img.Prototypes {
		if p.Name == "f" {
			if !p.EvalOnly {
				t.Fatalf("f compiled natively; expected EvalOnly")
			}
			return p.EvalReason
		}
	}
	t.Fatalf("prototype f missing")
	return ""
}
