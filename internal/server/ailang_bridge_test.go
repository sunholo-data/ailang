package server

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/embed"
)

const budgetModulePath = "internal/dashboard_transforms/budget_checker"

// The dashboard transforms have ONE implementation, in AILANG. These pin the
// two properties that removing the Go copies depends on: a transform that
// cannot run reports it, and the values it receives survive the trip.

func TestBridgeReady_ReportsWhyItCannotRun(t *testing.T) {
	// A bridge that failed to load must not be silently inert. Before the Go
	// copies were removed this state was invisible: IsEnabled() returned false,
	// every method quietly answered from Go, and the dashboard looked healthy.
	if err := (&AILANGBridge{}).Ready(); err == nil {
		t.Error("a bridge with no engine must report that it cannot run")
	}
	if err := (*AILANGBridge)(nil).Ready(); err == nil {
		t.Error("a nil bridge must report that it cannot run")
	}
}

func TestBridgeMethods_ErrorRatherThanAnswerWhenUnavailable(t *testing.T) {
	// Every method, so a future one cannot quietly reintroduce a default. The
	// zero value stands in for "modules not present", which is exactly the state
	// the deployed image was in.
	b := &AILANGBridge{}

	if _, err := b.SummarizeEvents(nil); err == nil {
		t.Error("SummarizeEvents must error, not return an empty summary")
	}
	if _, err := b.CountTurns(nil); err == nil {
		t.Error("CountTurns must error, not return 0")
	}
	if _, err := b.Truncate("text", 2); err == nil {
		t.Error("Truncate must error, not return the text")
	}
	if _, err := b.BuildHeatmapGrid(nil, 0, 0, 7); err == nil {
		t.Error("BuildHeatmapGrid must error, not return an empty grid")
	}
	// The one that gates spending.
	if _, err := b.CheckTaskBudget(DefaultBudgetConfig(), 1, 0, 0); err == nil {
		t.Error("CheckTaskBudget must error, not return an allowed=false or allowed=true guess")
	}
	if _, err := b.CalculateBurnRate(nil, 3600000); err == nil {
		t.Error("CalculateBurnRate must error, not return 0")
	}
	if _, err := b.ForecastExhaustion(10, 1); err == nil {
		t.Error("ForecastExhaustion must error, not return -1")
	}
}

// TestBudgetRule_WholeNumberBudgetsReachAILANGAsFloats is the regression arm for
// the defect the removed fallback was hiding.
//
// Engine.Call folds a whole float64 into an IntValue for JSON compatibility, so
// checkTaskBudget's `float`-typed parameters received ints and the call died
// with "missing dictionary method: prelude::Fractional::Int::add". Budgets are
// whole numbers in every realistic config ($100, $50, $25), so this fired on
// essentially every invocation — and nothing noticed, because the error was
// discarded and a Go copy answered instead. CallPreserveFloats is the fix.
//
// If someone switches these call sites back to Call, this goes red.
func TestBudgetRule_WholeNumberBudgetsReachAILANGAsFloats(t *testing.T) {
	// The same resolution production uses: walk up from the package directory
	// to the tree that holds the module.
	eng, err := embed.NewForModule(budgetModulePath)
	if err != nil {
		t.Skipf("budget_checker module not reachable from the test tree: %v", err)
	}
	defer func() { _ = eng.Close() }()

	// Deliberately all whole numbers — the shape that used to fail.
	config := map[string]interface{}{
		"workspaceBudget":  100.0,
		"dailyBudget":      50.0,
		"taskMaxCost":      25.0,
		"warningThreshold": 0.8,
	}

	if _, err := eng.CallPreserveFloats(
		budgetModulePath, "checkTaskBudget",
		config, 10.0, 20.0, 10.0,
	); err != nil {
		t.Fatalf("whole-number budgets must reach AILANG as floats: %v", err)
	}

	// And the negative arm: plain Call reproduces the original failure, so the
	// distinction is real rather than cargo-culted.
	_, err = eng.Call(
		budgetModulePath, "checkTaskBudget",
		config, 10.0, 20.0, 10.0,
	)
	if err == nil {
		t.Skip("Call no longer folds whole floats to ints — CallPreserveFloats may no longer be required here")
	}
	if !strings.Contains(err.Error(), "Fractional") {
		t.Logf("Call failed differently than the recorded defect: %v", err)
	}
}
