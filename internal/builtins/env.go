package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Env effect builtins for AILANG
// These provide environment variable access with capability-based security

func init() {
	registerEnv()
}

// ============================================================================
// Env Effect Builtins (_env_getEnv, _env_hasEnv)
// ============================================================================

func registerEnv() {
	// _env_getEnv
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Env", "getEnv", args)
	}
	type1 := func() types.Type {
		T := types.NewBuilder()
		// Result(String, EnvError) where EnvError = NotFound(String) | NotAllowed(String)
		envErrorType := T.App("EnvError", T.String())
		resultType := T.App("Result", T.String(), envErrorType)
		return T.Func(T.String()).Returns(resultType).Effects("Env")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/env", Name: "_env_getEnv", NumArgs: 1, IsPure: false, Effect: "Env", Type: type1, Impl: impl1,

		Metadata: &BuiltinMetadata{
			Description: "Get environment variable value from immutable snapshot",
			Params: []ParamDoc{
				{Name: "name", Description: "Environment variable name to retrieve"},
			},
			Returns: "Result(String, EnvError) - Ok(value) if exists and allowed, Err(NotFound) if missing, Err(NotAllowed) if not in allowlist",
			Examples: []Example{
				{Code: `match _env_getEnv("API_KEY") { Ok(key) => "Found", Err(_) => "Missing" }`, Description: "Check if API_KEY exists"},
				{Code: `match _env_getEnv("PATH") { Ok(p) => p, Err(_) => "/usr/bin" }`, Description: "Get PATH with fallback"},
			},
			LongDesc: `Security features:
- Snapshot semantics: Uses frozen snapshot from program start (external changes ignored)
- Allowlist enforcement: Blocks non-allowed variables (use --allow-env or --allow-env-file)
- No enumeration: Cannot list all variables (security by design)
- Redaction: Sensitive values redacted in errors (disable with AILANG_REDACT_ENV=off)

CLI usage:
  ailang run --caps Env program.ail                    # Allow all variables
  ailang run --caps Env --allow-env API_KEY program.ail # Allow only API_KEY
  ailang run --caps Env --env API_KEY=test program.ail  # Override variable`,
			SeeAlso:   []string{"_env_hasEnv"},
			Since:     "v0.4.0",
			Stability: StabilityStable,
			Tags:      []string{"env", "environment", "config", "security"},
			Category:  "env",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _env_getEnv: %v", err))
	}

	// _env_hasEnv
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Env", "hasEnv", args)
	}
	type2 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.Bool()).Effects("Env")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/env", Name: "_env_hasEnv", NumArgs: 1, IsPure: false, Effect: "Env", Type: type2, Impl: impl2,

		Metadata: &BuiltinMetadata{
			Description: "Check if environment variable exists in snapshot",
			Params: []ParamDoc{
				{Name: "name", Description: "Environment variable name to check"},
			},
			Returns: "Bool - true if variable exists and is allowed, false otherwise",
			Examples: []Example{
				{Code: `if _env_hasEnv("DEBUG") then "Debug mode" else "Production"`, Description: "Conditional based on DEBUG existence"},
				{Code: `_env_hasEnv("API_KEY")`, Description: "Returns true if API_KEY exists and is allowed"},
			},
			LongDesc: `Security features:
- Snapshot semantics: Checks frozen snapshot from program start
- Allowlist enforcement: Returns false for non-allowed variables (doesn't reveal existence)
- No enumeration: Cannot discover variable names
- Safe: Never throws error (unlike getEnv)

Use hasEnv() for existence checks, getEnv() to retrieve values.

CLI usage:
  ailang run --caps Env program.ail                    # Check all variables
  ailang run --caps Env --allow-env DEBUG program.ail  # Only DEBUG checkable`,
			SeeAlso:   []string{"_env_getEnv"},
			Since:     "v0.4.0",
			Stability: StabilityStable,
			Tags:      []string{"env", "environment", "check", "exists"},
			Category:  "env",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _env_hasEnv: %v", err))
	}
}
