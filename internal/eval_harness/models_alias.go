package eval_harness

import "github.com/sunholo-data/ailang/internal/modelreg"

// M-MODEL-REGISTRY-SINGLE-SOURCE M1 (decision D4(a), 2026-08-27).
//
// The model registry moved to internal/modelreg so it could become a LEAF that
// internal/executor is free to import. eval_harness cannot host it: eval_harness
// imports executor for ten symbols (Task, Result, EventHandler, the cost
// classes, ValidateTaskCapabilities), so executors resolving roles through a
// registry in eval_harness would close the cycle executor -> eval_harness ->
// executor, which does not compile.
//
// These aliases keep eval_harness's public registry surface intact so the ~66
// call sites that name these types and functions did not have to change. They
// are aliases (=), not definitions, so a *modelreg.ModelsConfig and an
// *eval_harness.ModelsConfig are the SAME type and cross freely.
//
// GlobalModelsConfig is deliberately NOT re-exported here. It is a mutable
// package-level variable that InitModelsConfig writes; a second declaration in
// this package would be a COPY of the pointer, silently stale for any caller
// that read it before initialization. Callers name modelreg.GlobalModelsConfig
// directly — one variable, one source, which is the point of this sprint.

type (
	ModelConfig      = modelreg.ModelConfig
	ModelsConfig     = modelreg.ModelsConfig
	Pricing          = modelreg.Pricing
	ScheduledPricing = modelreg.ScheduledPricing
	Budgets          = modelreg.Budgets
)

var (
	LoadModelsConfig   = modelreg.LoadModelsConfig
	InitModelsConfig   = modelreg.InitModelsConfig
	FindModelsConfig   = modelreg.FindModelsConfig
	ResolveModelName   = modelreg.ResolveModelName
	IsOllamaCloudRoute = modelreg.IsOllamaCloudRoute
)
