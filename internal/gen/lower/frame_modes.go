package lower

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// Dynamic frame modes and the bytecode VM (#1522, #1545).
//
// Some effect annotations are not just types: the evaluator enforces them by
// pushing a frame onto the effect context when it applies the function and
// popping it on return, so everything in the function's dynamic extent runs
// under the frame. Today those are:
//
//   - Net[scope=public]       → PushNetScope   (public-only network policy)
//   - Rand[mode=seeded|crypto] → PushRandMode   (deterministic / crypto/rand draws)
//   - @limit / @min budgets    → PushBudgetFrame (per-call effect budgets)
//
// (eval.FunctionValue's EffectNetScope / EffectRandMode / EffectBudgets /
// EffectMinBudgets, pushed in applyFunctionValue — keep this list in step.)
//
// The VM has no moded-frame hook: a VM frame pushes nothing, and every effect
// the VM performs goes through the interop bridge to the evaluator, which then
// sees the default (os mode, no scope, no budget). So a function declaring one
// of these modes is lowered as an EvalOnly stub: the call routes through the
// bridge to the evaluator, which pushes the frame and runs the whole body —
// including every callee — on the evaluator. Under --strict-bytecode there is
// no bridge, so the call fails loudly with the reason below.

// frameModeReason returns a non-empty reason when the function type t
// declares a frame mode only the evaluator can enforce, or "" when it does
// not. Mirrors the evaluator's extractNetScope / extractRandMode /
// extractEffectBudgets / extractEffectMinBudgets.
func frameModeReason(t types.Type) string {
	fn, ok := t.(*types.TFunc2)
	if !ok || fn.EffectRow == nil {
		return ""
	}
	row := fn.EffectRow
	var modes []string
	if row.Params["Net"]["scope"] != "" {
		modes = append(modes, "Net[scope="+row.Params["Net"]["scope"]+"]")
	}
	if mode, ok := types.EffectModeFor(row, "Rand"); ok && mode != "os" {
		modes = append(modes, "Rand[mode="+mode+"]")
	}
	modes = append(modes, budgetModes(row.Budgets, "@limit")...)
	modes = append(modes, budgetModes(row.MinBudgets, "@min")...)
	if len(modes) == 0 {
		return ""
	}
	return fmt.Sprintf("%s frame needs the evaluator (the bytecode VM cannot push it)", strings.Join(modes, ", "))
}

// budgetModes renders the set budgets of one kind ("@limit" / "@min") in a
// stable order, e.g. ["IO @limit=2"].
func budgetModes(budgets map[string]*int, kind string) []string {
	var out []string
	for eff, n := range budgets {
		if n != nil {
			out = append(out, fmt.Sprintf("%s %s=%d", eff, kind, *n))
		}
	}
	sort.Strings(out)
	return out
}

// lambdaFrameModeReason is frameModeReason for a Core lambda's inferred type.
func lambdaFrameModeReason(lam *core.Lambda, cti types.CoreTypeInfo) string {
	return frameModeReason(cti[lam.NodeID])
}

// frameModePanic is raised by lowerLambda when a NESTED lambda declares a
// frame mode. A VM closure cannot cross the bridge, so the only sound place
// for it is the evaluator: lowerTopLevelDeclSafe recovers the panic and turns
// the enclosing top-level function into an EvalOnly stub with this reason.
type frameModePanic struct{ reason string }

func (p frameModePanic) String() string { return "nested lambda: " + p.reason }
