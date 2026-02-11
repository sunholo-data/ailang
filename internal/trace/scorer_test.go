package trace

import (
	"math"
	"testing"
)

func TestScoreTrace_Empty(t *testing.T) {
	score := ScoreTrace(nil)
	// Empty trace: completion=0, complexity=0, contracts=0.5 (neutral), budget=0.5 (neutral), effects=0
	// Weighted: 0.20*0.5 + 0.15*0.5 = 0.175
	if score.Total < 0 || score.Total > 0.2 {
		t.Errorf("empty trace should score ~0.175, got %f", score.Total)
	}
	if score.Completion != 0 {
		t.Errorf("completion should be 0 for empty trace, got %f", score.Completion)
	}
	if score.Complexity != 0 {
		t.Errorf("complexity should be 0 for empty trace, got %f", score.Complexity)
	}
}

func TestScoreTrace_SimpleComplete(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "test", Caps: []string{"IO"}}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventEffect, Depth: 1, Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main", Result: "()"}},
		{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "test"}},
	}

	score := ScoreTrace(events)

	// Should complete successfully
	if score.Completion != 1.0 {
		t.Errorf("completion should be 1.0 for clean trace, got %f", score.Completion)
	}

	// Should have some complexity
	if score.Complexity <= 0 {
		t.Error("complexity should be > 0 for trace with function + effect")
	}

	// Total should be reasonable
	if score.Total < 0.2 || score.Total > 1.0 {
		t.Errorf("total score out of range: %f", score.Total)
	}

	// Stats should be populated
	if score.Stats.FunctionCalls != 1 {
		t.Errorf("expected 1 function call, got %d", score.Stats.FunctionCalls)
	}
	if score.Stats.EffectCalls != 1 {
		t.Errorf("expected 1 effect call, got %d", score.Stats.EffectCalls)
	}
	if score.Stats.DistinctEffects != 1 {
		t.Errorf("expected 1 distinct effect, got %d", score.Stats.DistinctEffects)
	}
}

func TestScoreTrace_ErrorPenalty(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventError, Error: &ErrorEvent{Message: "division by zero"}},
	}

	score := ScoreTrace(events)

	if score.Completion != 0.0 {
		t.Errorf("completion should be 0.0 for errored trace, got %f", score.Completion)
	}
}

func TestScoreTrace_NoModuleEnd(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main", Result: "42"}},
	}

	score := ScoreTrace(events)

	// No module end but no errors = 0.8
	if score.Completion != 0.8 {
		t.Errorf("completion should be 0.8 for no-module trace, got %f", score.Completion)
	}
}

func TestScoreTrace_ComplexProgram(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "test", Caps: []string{"IO", "FS"}}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 2, Function: &FunctionEvent{Name: "processFile"}},
		{Version: "1.0", Event: EventEffect, Depth: 2, Effect: &EffectEvent{EffectName: "FS", OpName: "readFile"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 3, Function: &FunctionEvent{Name: "transform"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 4, Function: &FunctionEvent{Name: "helper"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 4, Function: &FunctionEvent{Name: "helper", Result: "ok"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 3, Function: &FunctionEvent{Name: "transform", Result: "done"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 2, Function: &FunctionEvent{Name: "processFile", Result: "()"}},
		{Version: "1.0", Event: EventEffect, Depth: 1, Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main", Result: "()"}},
		{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "test"}},
	}

	score := ScoreTrace(events)

	// Should score higher than simple program
	if score.Total < 0.4 {
		t.Errorf("complex trace should score > 0.4, got %f", score.Total)
	}

	// Multiple distinct functions
	if score.Stats.DistinctFunctions != 4 {
		t.Errorf("expected 4 distinct functions, got %d", score.Stats.DistinctFunctions)
	}

	// Multiple effects
	if score.Stats.DistinctEffects != 2 {
		t.Errorf("expected 2 distinct effects, got %d", score.Stats.DistinctEffects)
	}

	// Max depth should be 4
	if score.Stats.MaxDepth != 4 {
		t.Errorf("expected max depth 4, got %d", score.Stats.MaxDepth)
	}

	// Effect diversity should be 0.7 (2 effects)
	if score.EffectDiversity != 0.7 {
		t.Errorf("expected effect diversity 0.7, got %f", score.EffectDiversity)
	}
}

func TestScoreTrace_ContractCoverage(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventContractCheck, Contract: &ContractEvent{Kind: "requires", Passed: true, Message: "x > 0"}},
		{Version: "1.0", Event: EventContractCheck, Contract: &ContractEvent{Kind: "ensures", Passed: true, Message: "result >= 0"}},
		{Version: "1.0", Event: EventContractCheck, Contract: &ContractEvent{Kind: "requires", Passed: false, Message: "y != 0"}},
	}

	score := ScoreTrace(events)

	// 2/3 contracts passed
	expected := 2.0 / 3.0
	if math.Abs(score.ContractCoverage-expected) > 0.001 {
		t.Errorf("expected contract coverage %f, got %f", expected, score.ContractCoverage)
	}
}

func TestScoreTrace_NoContracts(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1, Function: &FunctionEvent{Name: "main"}},
	}

	score := ScoreTrace(events)

	// No contracts = neutral 0.5
	if score.ContractCoverage != 0.5 {
		t.Errorf("expected 0.5 for no contracts, got %f", score.ContractCoverage)
	}
}

func TestScoreTrace_BudgetEfficiency(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventBudgetDelta, Budget: &BudgetEvent{Effect: "IO", Used: 3, Limit: 5, Remaining: 2}},
	}

	score := ScoreTrace(events)

	// Used 3/5 = 60% — in the 20-80% sweet spot = 1.0
	if score.BudgetEfficiency != 1.0 {
		t.Errorf("expected budget efficiency 1.0 for 60%% usage, got %f", score.BudgetEfficiency)
	}
}

func TestScoreTrace_BudgetExhausted(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventBudgetDelta, Budget: &BudgetEvent{Effect: "IO", Used: 5, Limit: 5, Remaining: 0}},
	}

	score := ScoreTrace(events)

	// Used 5/5 = 100% — near-exhaustion = 0.5
	if score.BudgetEfficiency != 0.5 {
		t.Errorf("expected budget efficiency 0.5 for exhausted budget, got %f", score.BudgetEfficiency)
	}
}

func TestScoreTrace_EffectDiversity(t *testing.T) {
	tests := []struct {
		name     string
		effects  []string
		expected float64
	}{
		{"no effects", nil, 0.0},
		{"one effect", []string{"IO"}, 0.4},
		{"two effects", []string{"IO", "FS"}, 0.7},
		{"three effects", []string{"IO", "FS", "Net"}, 1.0},
		{"four effects", []string{"IO", "FS", "Net", "Clock"}, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var events []TraceEvent
			for _, eff := range tt.effects {
				events = append(events, TraceEvent{
					Version: "1.0", Event: EventEffect,
					Effect: &EffectEvent{EffectName: eff, OpName: "op"},
				})
			}

			score := ScoreTrace(events)
			if score.EffectDiversity != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, score.EffectDiversity)
			}
		})
	}
}

func TestScoreTrace_EffectBreakdown(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
		{Version: "1.0", Event: EventEffect, Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
		{Version: "1.0", Event: EventEffect, Effect: &EffectEvent{EffectName: "FS", OpName: "readFile"}},
	}

	score := ScoreTrace(events)

	if score.EffectBreakdown["IO"] != 2 {
		t.Errorf("expected IO=2, got %d", score.EffectBreakdown["IO"])
	}
	if score.EffectBreakdown["FS"] != 1 {
		t.Errorf("expected FS=1, got %d", score.EffectBreakdown["FS"])
	}
}

func TestScoreTrace_FunctionCounts(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1, Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 2, Function: &FunctionEvent{Name: "factorial"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 3, Function: &FunctionEvent{Name: "factorial"}},
		{Version: "1.0", Event: EventFunctionEnter, Depth: 4, Function: &FunctionEvent{Name: "factorial"}},
	}

	score := ScoreTrace(events)

	if score.FunctionCounts["main"] != 1 {
		t.Errorf("expected main=1, got %d", score.FunctionCounts["main"])
	}
	if score.FunctionCounts["factorial"] != 3 {
		t.Errorf("expected factorial=3, got %d", score.FunctionCounts["factorial"])
	}
}

func TestScoreTrace_TotalInRange(t *testing.T) {
	// Verify total is always in [0, 1] for various inputs
	testCases := [][]TraceEvent{
		nil,
		{},
		{{Version: "1.0", Event: EventError, Error: &ErrorEvent{Message: "fail"}}},
		{
			{Version: "1.0", Event: EventModuleStart, Module: &ModuleEvent{Name: "t"}},
			{Version: "1.0", Event: EventModuleEnd, Module: &ModuleEvent{Name: "t"}},
		},
	}

	for i, events := range testCases {
		score := ScoreTrace(events)
		if score.Total < 0 || score.Total > 1 {
			t.Errorf("case %d: total out of range: %f", i, score.Total)
		}
	}
}
