package dashboard_transforms

import (
	"testing"

	"github.com/sunholo/ailang/internal/embed"
)

// BudgetConfig mirrors the AILANG type
type BudgetConfig struct {
	WorkspaceBudget  float64 `json:"workspaceBudget"`
	DailyBudget      float64 `json:"dailyBudget"`
	TaskMaxCost      float64 `json:"taskMaxCost"`
	WarningThreshold float64 `json:"warningThreshold"`
}

// BudgetStatus mirrors the AILANG type
type BudgetStatus struct {
	Allowed            bool    `json:"allowed"`
	RemainingWorkspace float64 `json:"remainingWorkspace"`
	RemainingDaily     float64 `json:"remainingDaily"`
	WarningLevel       string  `json:"warningLevel"`
	Message            string  `json:"message"`
}

// Go implementations (baseline)

func goCheckTaskBudget(config BudgetConfig, estimatedCost, currentWorkspaceSpend, currentDailySpend float64) BudgetStatus {
	remainingWorkspace := config.WorkspaceBudget - currentWorkspaceSpend
	remainingDaily := config.DailyBudget - currentDailySpend
	minRemaining := remainingWorkspace
	if remainingDaily < minRemaining {
		minRemaining = remainingDaily
	}

	if estimatedCost > minRemaining {
		return BudgetStatus{
			Allowed:            false,
			RemainingWorkspace: remainingWorkspace,
			RemainingDaily:     remainingDaily,
			WarningLevel:       "exceeded",
			Message:            "Task cost exceeds remaining budget",
		}
	}

	if estimatedCost > config.TaskMaxCost {
		return BudgetStatus{
			Allowed:            false,
			RemainingWorkspace: remainingWorkspace,
			RemainingDaily:     remainingDaily,
			WarningLevel:       "exceeded",
			Message:            "Task exceeds maximum single-task cost",
		}
	}

	usageRatio := (currentDailySpend + estimatedCost) / config.DailyBudget
	level := "ok"
	if usageRatio > 0.9 {
		level = "critical"
	} else if usageRatio > config.WarningThreshold {
		level = "warning"
	}

	return BudgetStatus{
		Allowed:            true,
		RemainingWorkspace: remainingWorkspace - estimatedCost,
		RemainingDaily:     remainingDaily - estimatedCost,
		WarningLevel:       level,
		Message:            "Task approved",
	}
}

func goUsagePercent(spent, budget float64) float64 {
	if budget <= 0 {
		return 0
	}
	return (spent / budget) * 100.0
}

func goIsWarningZone(spent, budget, threshold float64) bool {
	if budget <= 0 {
		return false
	}
	return (spent / budget) >= threshold
}

func goRemaining(spent, budget float64) float64 {
	diff := budget - spent
	if diff < 0 {
		return 0
	}
	return diff
}

// Test data

func defaultConfig() BudgetConfig {
	return BudgetConfig{
		WorkspaceBudget:  100.0,
		DailyBudget:      50.0,
		TaskMaxCost:      25.0,
		WarningThreshold: 0.8,
	}
}

// Benchmarks - Go baseline

func BenchmarkCheckTaskBudget_Go(b *testing.B) {
	config := defaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goCheckTaskBudget(config, 5.0, 20.0, 30.0)
	}
}

func BenchmarkUsagePercent_Go(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goUsagePercent(75.0, 100.0)
	}
}

func BenchmarkRemaining_Go(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goRemaining(75.0, 100.0)
	}
}

// Benchmarks - AILANG

func BenchmarkCheckTaskBudget_AILANG(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	config := defaultConfig()

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/budget_checker", "checkTaskBudget",
		config, 5.0, 20.0, 30.0)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/budget_checker", "checkTaskBudget",
			config, 5.0, 20.0, 30.0)
	}
}

func BenchmarkUsagePercent_AILANG(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/budget_checker", "usagePercent", 75.0, 100.0)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/budget_checker", "usagePercent", 75.0, 100.0)
	}
}

func BenchmarkRemaining_AILANG(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/budget_checker", "remaining", 75.0, 100.0)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/budget_checker", "remaining", 75.0, 100.0)
	}
}

// Functional correctness tests

func TestCheckTaskBudget_Allowed(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	config := defaultConfig()
	goResult := goCheckTaskBudget(config, 5.0, 20.0, 10.0)

	ailangResult, err := engine.Call("internal/dashboard_transforms/budget_checker", "checkTaskBudget",
		config, 5.0, 20.0, 10.0)
	if err != nil {
		t.Fatalf("AILANG checkTaskBudget failed: %v", err)
	}

	goVal, _ := embed.ToGo(ailangResult)
	resultMap := goVal.(map[string]interface{})

	if goResult.Allowed != resultMap["allowed"].(bool) {
		t.Errorf("allowed mismatch: Go=%v, AILANG=%v", goResult.Allowed, resultMap["allowed"])
	}
	if goResult.WarningLevel != resultMap["warningLevel"].(string) {
		t.Errorf("warningLevel mismatch: Go=%v, AILANG=%v", goResult.WarningLevel, resultMap["warningLevel"])
	}

	t.Logf("checkTaskBudget: allowed=%v, level=%v", resultMap["allowed"], resultMap["warningLevel"])
}

func TestCheckTaskBudget_Exceeded(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	config := defaultConfig()
	// Spend 45 of 50 daily budget, then try to spend 10 more
	goResult := goCheckTaskBudget(config, 10.0, 20.0, 45.0)

	ailangResult, err := engine.Call("internal/dashboard_transforms/budget_checker", "checkTaskBudget",
		config, 10.0, 20.0, 45.0)
	if err != nil {
		t.Fatalf("AILANG checkTaskBudget failed: %v", err)
	}

	goVal, _ := embed.ToGo(ailangResult)
	resultMap := goVal.(map[string]interface{})

	if goResult.Allowed != resultMap["allowed"].(bool) {
		t.Errorf("allowed mismatch: Go=%v, AILANG=%v", goResult.Allowed, resultMap["allowed"])
	}
	if goResult.WarningLevel != resultMap["warningLevel"].(string) {
		t.Errorf("warningLevel mismatch: Go=%v, AILANG=%v", goResult.WarningLevel, resultMap["warningLevel"])
	}

	t.Logf("checkTaskBudget (exceeded): allowed=%v, level=%v", resultMap["allowed"], resultMap["warningLevel"])
}

func TestUsagePercent_Correctness(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	testCases := []struct {
		spent, budget, expected float64
	}{
		{0.0, 100.0, 0.0},
		{50.0, 100.0, 50.0},
		{100.0, 100.0, 100.0},
		{75.0, 100.0, 75.0},
	}

	for _, tc := range testCases {
		goResult := goUsagePercent(tc.spent, tc.budget)

		ailangResult, err := engine.Call("internal/dashboard_transforms/budget_checker", "usagePercent",
			tc.spent, tc.budget)
		if err != nil {
			t.Fatalf("AILANG usagePercent(%v, %v) failed: %v", tc.spent, tc.budget, err)
		}
		ailangFloat, _ := embed.ToFloat(ailangResult)

		if goResult != ailangFloat {
			t.Errorf("usagePercent(%v, %v): Go=%v, AILANG=%v", tc.spent, tc.budget, goResult, ailangFloat)
		}
	}
}

func TestIsWarningZone_Correctness(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	testCases := []struct {
		spent, budget, threshold float64
		expected                 bool
	}{
		{0.0, 100.0, 0.8, false},
		{79.0, 100.0, 0.8, false},
		{80.0, 100.0, 0.8, true},
		{90.0, 100.0, 0.8, true},
	}

	for _, tc := range testCases {
		goResult := goIsWarningZone(tc.spent, tc.budget, tc.threshold)

		ailangResult, err := engine.Call("internal/dashboard_transforms/budget_checker", "isWarningZone",
			tc.spent, tc.budget, tc.threshold)
		if err != nil {
			t.Fatalf("AILANG isWarningZone failed: %v", err)
		}
		ailangBool, _ := embed.ToBool(ailangResult)

		if goResult != ailangBool {
			t.Errorf("isWarningZone(%v, %v, %v): Go=%v, AILANG=%v",
				tc.spent, tc.budget, tc.threshold, goResult, ailangBool)
		}
	}
}

func TestRemaining_Correctness(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	testCases := []struct {
		spent, budget, expected float64
	}{
		{0.0, 100.0, 100.0},
		{50.0, 100.0, 50.0},
		{100.0, 100.0, 0.0},
		{150.0, 100.0, 0.0},
	}

	for _, tc := range testCases {
		goResult := goRemaining(tc.spent, tc.budget)

		ailangResult, err := engine.Call("internal/dashboard_transforms/budget_checker", "remaining",
			tc.spent, tc.budget)
		if err != nil {
			t.Fatalf("AILANG remaining(%v, %v) failed: %v", tc.spent, tc.budget, err)
		}
		ailangFloat, _ := embed.ToFloat(ailangResult)

		if goResult != ailangFloat {
			t.Errorf("remaining(%v, %v): Go=%v, AILANG=%v", tc.spent, tc.budget, goResult, ailangFloat)
		}
	}
}
