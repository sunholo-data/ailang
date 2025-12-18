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
		// Note: EnvError is a non-parameterized ADT (arity 0), use T.Con not T.App
		envErrorType := T.Con("EnvError")
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

	// _env_getArgs
	impl3 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		// Validate unit argument (defense against type system bugs)
		if len(args) != 1 {
			panic("internal invariant violation: _env_getArgs expects exactly 1 argument (unit)")
		}
		if _, ok := args[0].(*eval.UnitValue); !ok {
			panic("internal invariant violation: _env_getArgs expected unit argument")
		}
		// Pass unit argument to effect handler (unit-argument model)
		return effects.Call(ctx, "Env", "getArgs", args)
	}
	type3 := func() types.Type {
		T := types.NewBuilder()
		// Unit-argument model: () -> [string] ! {Env}
		return T.Func(T.Unit()).Returns(T.List(T.String())).Effects("Env")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/env", Name: "_env_getArgs", NumArgs: 1, IsPure: false, Effect: "Env", Type: type3, Impl: impl3,

		Metadata: &BuiltinMetadata{
			Description: "Get command-line arguments as list of strings",
			Params:      []ParamDoc{},
			Returns:     "[String] - List of command-line arguments (excluding program name)",
			Examples: []Example{
				{Code: `match _env_getArgs() { [] => "No args", [name] => "Hello " ++ name, _ => "Multiple args" }`, Description: "Pattern match on arguments"},
				{Code: `length(_env_getArgs())`, Description: "Count number of arguments"},
			},
			LongDesc: `Security features:
- Requires Env capability (use --caps Env)
- Read-only access to fixed arguments
- Arguments exclude program name (argv[1:] in C terms)
- Empty list if no arguments provided

CLI usage:
  ailang run --caps Env program.ail arg1 arg2  # Returns ["arg1", "arg2"]
  ailang run --caps Env program.ail            # Returns []`,
			SeeAlso:   []string{"_env_getEnv", "_env_hasEnv"},
			Since:     "v0.4.6",
			Stability: StabilityStable,
			Tags:      []string{"env", "cli", "arguments", "argv"},
			Category:  "env",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _env_getArgs: %v", err))
	}
}
