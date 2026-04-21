package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// init registers Env effect operations
func init() {
	RegisterOp("Env", "getEnv", envGetEnv)
	RegisterOp("Env", "hasEnv", envHasEnv)
	RegisterOp("Env", "getArgs", envGetArgs)
}

// envGetEnv implements Env.getEnv(name: String) -> Result(String, EnvError)
//
// Retrieves an environment variable value from the immutable snapshot.
// Respects allowlist if configured. Never accesses os.Getenv() directly.
//
// Parameters:
//   - ctx: Effect context with EnvSnapshot and optional EnvAllowlist
//   - args: [StringValue] - the environment variable name
//
// Returns:
//   - Result(Ok(StringValue), Err(EnvError))
//   - Ok if variable exists and is allowed
//   - Err(NotFound) if variable doesn't exist
//   - Err(NotAllowed) if variable not in allowlist
//   - Error if wrong arguments or missing Env capability
//
// Security:
//   - Snapshot semantics: external changes ignored after program start
//   - Allowlist enforcement: blocks enumeration attacks
//   - No reflection: cannot list all variables
//
// Example AILANG code:
//
//	match getEnv("API_KEY") {
//	  Ok(key) => httpRequest(url, "GET", [("Authorization", "Bearer " ++ key)], "")
//	  Err(NotFound) => fail("API_KEY not set")
//	}
//
// With allowlist:
//
//	ailang run --caps Env --allow-env API_KEY,DEBUG app.ail
func envGetEnv(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Check capability
	if !ctx.HasCap("Env") {
		return nil, fmt.Errorf("getEnv: Env capability required. Use --caps Env flag")
	}

	// Validate arguments
	if len(args) != 1 {
		return nil, fmt.Errorf("getEnv: expected 1 argument, got %d", len(args))
	}

	nameVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("getEnv: expected String, got %T", args[0])
	}

	name := nameVal.Value

	// Check allowlist if configured
	if ctx.EnvAllowlist != nil {
		if !isInAllowlist(name, ctx.EnvAllowlist) {
			// Return Err(NotAllowed) Result
			errValue := makeEnvError("NotAllowed", fmt.Sprintf("environment variable %q not in allowlist. Use --allow-env %s or add to allowlist file", name, name))
			return makeErrResult(errValue), nil
		}
	}

	// Lookup in snapshot
	value, exists := ctx.EnvSnapshot[name]
	if !exists {
		// Return Err(NotFound) Result
		errValue := makeEnvError("NotFound", fmt.Sprintf("environment variable %q not found", name))
		return makeErrResult(errValue), nil
	}

	// Return Ok(value) Result
	return makeOkResult(&eval.StringValue{Value: value}), nil
}

// envHasEnv implements Env.hasEnv(name: String) -> Bool
//
// Checks if an environment variable exists in the immutable snapshot.
// Respects allowlist if configured.
//
// Parameters:
//   - ctx: Effect context with EnvSnapshot and optional EnvAllowlist
//   - args: [StringValue] - the environment variable name
//
// Returns:
//   - BoolValue: true if variable exists and is allowed, false otherwise
//   - Error if wrong arguments or missing Env capability
//
// Security:
//   - Snapshot semantics: external changes ignored after program start
//   - Allowlist enforcement: returns false for non-allowed variables
//   - No enumeration: cannot discover variable names
//
// Example AILANG code:
//
//	if hasEnv("DEBUG") then
//	  print("Debug mode enabled")
//	else
//	  print("Production mode")
func envHasEnv(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Check capability
	if !ctx.HasCap("Env") {
		return nil, fmt.Errorf("hasEnv: Env capability required. Use --caps Env flag")
	}

	// Validate arguments
	if len(args) != 1 {
		return nil, fmt.Errorf("hasEnv: expected 1 argument, got %d", len(args))
	}

	nameVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("hasEnv: expected String, got %T", args[0])
	}

	name := nameVal.Value

	// Check allowlist if configured
	if ctx.EnvAllowlist != nil {
		if !isInAllowlist(name, ctx.EnvAllowlist) {
			// Not in allowlist → return false (don't reveal existence)
			return &eval.BoolValue{Value: false}, nil
		}
	}

	// Check existence in snapshot
	_, exists := ctx.EnvSnapshot[name]
	return &eval.BoolValue{Value: exists}, nil
}

// isInAllowlist checks if a variable name is in the allowlist
func isInAllowlist(name string, allowlist []string) bool {
	for _, allowed := range allowlist {
		if allowed == name {
			return true
		}
	}
	return false
}

// makeEnvError creates an EnvError Result
//
// EnvError ADT:
//
//	type EnvError =
//	  | NotFound(String)   -- Variable doesn't exist
//	  | NotAllowed(String) -- Variable not in allowlist
func makeEnvError(constructor, message string) eval.Value {
	// Create tagged value: NotFound("message") or NotAllowed("message")
	return &eval.TaggedValue{
		ModulePath: "std/env",
		TypeName:   "EnvError",
		CtorName:   constructor,
		Fields:     []eval.Value{&eval.StringValue{Value: message}},
	}
}

// makeOkResult creates Ok(value) Result
func makeOkResult(value eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{value},
	}
}

// makeErrResult creates Err(error) Result
func makeErrResult(errValue eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{errValue},
	}
}

// envGetArgs implements Env.getArgs() -> [String]
//
// Retrieves command-line arguments passed to the AILANG program.
// Arguments exclude the program name (equivalent to argv[1:] in C).
//
// Parameters:
//   - ctx: Effect context with Args field
//   - args: [] - no arguments (nullary function)
//
// Returns:
//   - ListValue containing StringValues for each CLI argument
//   - Empty list if no arguments provided
//   - Error if Env capability not granted
//
// Security:
//   - Requires Env capability
//   - Read-only access to fixed arguments
//   - No modification possible
//
// Example AILANG code:
//
//	match getArgs() {
//	  [] => println("No arguments"),
//	  [name] => println("Hello, " ++ name ++ "!"),
//	  _ => println("Multiple arguments provided")
//	}
//
// CLI usage:
//
//	ailang run --caps Env program.ail arg1 arg2
//	# getArgs() returns ["arg1", "arg2"]
func envGetArgs(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	// Check capability
	if !ctx.HasCap("Env") {
		return nil, fmt.Errorf("getArgs: Env capability required. Use --caps Env flag")
	}

	// Validate unit argument (unit-argument model)
	if len(args) != 1 {
		return nil, fmt.Errorf("getArgs: expected 1 argument (unit), got %d", len(args))
	}
	if _, ok := args[0].(*eval.UnitValue); !ok {
		return nil, fmt.Errorf("getArgs: expected unit argument, got %T", args[0])
	}

	// Convert CLI args to AILANG list
	var result eval.Value = &eval.ListValue{Elements: []eval.Value{}}

	if len(ctx.Args) > 0 {
		elements := make([]eval.Value, len(ctx.Args))
		for i, arg := range ctx.Args {
			elements[i] = &eval.StringValue{Value: arg}
		}
		result = &eval.ListValue{Elements: elements}
	}

	return result, nil
}
