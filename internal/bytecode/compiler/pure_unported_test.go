package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// TestCompile_PureUnportedBuiltinSaysPure is #1447: a pure registry builtin the
// VM cannot run must be reported as pure, with the reason — never as an
// "effectful builtin … Phase 2E". __xml_nodeKind remains unported (opaque XmlNode values).
func TestCompile_PureUnportedBuiltinSaysPure(t *testing.T) {
	reason := evalReasonOf(t, "__xml_nodeKind")
	if !strings.Contains(reason, `pure builtin "__xml_nodeKind" has no VM implementation (opaque-type`) {
		t.Errorf("EvalReason = %q, want the pure-unported message with its opaque-type reason", reason)
	}
	if strings.Contains(reason, "effectful") {
		t.Errorf("EvalReason calls a pure builtin effectful: %q", reason)
	}
}

// TestCompile_EffectfulBuiltinStillPhase2E pins the other side: effectful
// builtins keep the Phase 2E message.
func TestCompile_EffectfulBuiltinStillPhase2E(t *testing.T) {
	reason := evalReasonOf(t, "__net_httpRequest")
	if !strings.Contains(reason, `effectful builtin "__net_httpRequest" not yet wired (Phase 2E)`) {
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
