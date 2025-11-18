# M-LANG-CLI-ARGS: Command-Line Arguments Access

**Status**: Planned
**Version Target**: v0.4.6
**Priority**: HIGH
**Effort**: 3-4 hours
**Impact**: +2.1% eval success rate (77% total with prompt improvements)

## Problem Statement

The `cli_args` benchmark fails across all 6 models (0% success rate) because **AILANG has no way to access command-line arguments**.

**Current workaround**: None - programs cannot access `argv`

**User request**: Common use case for CLI tools, utilities, scripts

**Eval impact**: 6/284 failures (2.1%), but represents fundamental limitation for real-world programs

## Goals

1. **Provide access to command-line arguments** passed to `ailang run`
2. **Functional, effect-typed design** (not global mutable state)
3. **Simple API** that models can understand and use correctly
4. **Consistent with AILANG philosophy** (deterministic, explicit effects)

**Non-goals**:
- Complex argument parsing libraries (flags, options, etc.)
- Env variables (separate feature)
- Process spawning, piping

## Design

### API: `std/env.getArgs`

**Module**: `std/env` (new stdlib module for environment/OS interaction)

**Function signature**:
```ailang
export func getArgs() -> [string] ! {IO}
```

**Returns**: List of command-line arguments (excluding program name)

**Effect**: `! {IO}` - reading from environment is an effect

**Example usage**:
```ailang
module myapp/main

import std/env (getArgs)
import std/io (println)
import std/list (length)

export func main() -> () ! {IO} {
  let args = getArgs();
  println("Number of arguments: " ++ show(length(args)));

  match args {
    [] => println("No arguments provided"),
    [name] => println("Hello, " ++ name ++ "!"),
    _ => println("Too many arguments")
  }
}
```

**Running**:
```bash
$ ailang run --entry main --caps IO myapp/main.ail Alice
Number of arguments: 1
Hello, Alice!

$ ailang run --entry main --caps IO myapp/main.ail
Number of arguments: 0
No arguments provided
```

### Alternative: Main Function Signature (Rejected)

**Option**: Automatically pass args to `main` function
```ailang
export func main(args: [string]) -> () ! {IO} {
  -- args available as parameter
}
```

**Rejected because**:
1. Breaks existing code (all `main` functions have signature `() -> ()`)
2. Less flexible (what if you don't need args?)
3. Inconsistent with other effects (readFile, getEnv, etc.)
4. Makes `main` special (not just another function)

**Decision**: Use explicit `getArgs()` function for clarity and consistency

### Effect Choice: Why `! {IO}`?

**Rationale**: Arguments come from external environment (OS)

**Arguments for `! {IO}`**:
- ✅ Consistent with other OS interactions (env vars, stdin)
- ✅ External input (not pure)
- ✅ Could theoretically change between runs (even if rare)
- ✅ Matches common practice (Haskell's `getArgs :: IO [String]`)

**Arguments against (pure)**:
- Arguments don't change during execution
- Could be treated as compile-time constant

**Decision**: `! {IO}` for consistency with external input paradigm

### Return Value: `[string]` vs Other Options

**Option 1**: `[string]` (CHOSEN)
```ailang
getArgs() -> [string]
-- Returns: ["arg1", "arg2", "arg3"]
```
**Pros**:
- ✅ Simple, functional
- ✅ Matches Haskell (`getArgs :: IO [String]`)
- ✅ Easy to pattern match

**Option 2**: `{argc: int, argv: [string]}`
```ailang
getArgs() -> {argc: int, argv: [string]}
-- Returns: {argc: 3, argv: ["arg1", "arg2", "arg3"]}
```
**Cons**:
- ❌ Redundant (can compute `argc` from `length(argv)`)
- ❌ More complex API

**Option 3**: Include program name
```ailang
getArgs() -> [string]
-- Returns: ["program_name", "arg1", "arg2"]
```
**Cons**:
- ❌ Program name not useful in AILANG context
- ❌ Inconsistent with modern CLIs (most languages skip it)

**Decision**: Return `[string]` (arguments only, no program name)

## Implementation Plan

### Phase 1: Builtin Function (1 hour)

**File**: `internal/builtins/spec.go`

**Add builtin**:
```go
{
	Name:      "getArgs",
	Module:    "std/env",
	Type:      types.Func(types.TList(types.TString), types.TIO(types.TUnit)),
	GoImpl:    builtins.GetArgs,
	Pure:      false,
	Doc:       "Get command-line arguments as list of strings",
},
```

**File**: `internal/builtins/env.go` (new file)

```go
package builtins

import (
	"os"
	"github.com/yourusername/ailang/internal/eval"
)

// GetArgs returns command-line arguments (excluding program name)
func GetArgs(ctx eval.EffectContext, args []eval.Value) (eval.Value, error) {
	// Get args from context (set by runtime)
	runtimeArgs := ctx.GetArgs()

	// Convert to AILANG list
	var result eval.Value = eval.EmptyList
	for i := len(runtimeArgs) - 1; i >= 0; i-- {
		result = eval.Cons(eval.String(runtimeArgs[i]), result)
	}

	return result, nil
}
```

**File**: `internal/effects/context.go`

**Add to EffectContext**:
```go
type EffectContext interface {
	// ... existing methods ...

	// GetArgs returns command-line arguments
	GetArgs() []string
}

type RealEffContext struct {
	// ... existing fields ...

	args []string  // CLI arguments
}

func NewRealEffContext(args []string) *RealEffContext {
	return &RealEffContext{
		args: args,
		// ... other fields ...
	}
}

func (ctx *RealEffContext) GetArgs() []string {
	return ctx.args
}
```

**File**: `internal/runtime/runtime.go`

**Pass args to effect context**:
```go
func Run(prog *core.Program, caps []string, args []string) error {
	// ... existing code ...

	// Create effect context with CLI args
	effCtx := effects.NewRealEffContext(args)

	// ... rest of execution ...
}
```

**File**: `cmd/ailang/main.go`

**Collect args and pass to runtime**:
```go
func runCommand(c *cli.Context) error {
	// ... existing code ...

	// Get CLI args (everything after the .ail file)
	programArgs := c.Args().Slice()[1:]  // Skip filename

	// Run with args
	if err := runtime.Run(prog, caps, programArgs); err != nil {
		return err
	}

	return nil
}
```

### Phase 2: Standard Library Module (30 minutes)

**File**: `std/env.ail` (new file)

```ailang
-- Environment and OS interaction
module std/env

-- Get command-line arguments
-- Returns arguments passed to the program (excluding program name)
-- Example: ailang run app.ail arg1 arg2 → ["arg1", "arg2"]
export func getArgs() -> [string] ! {IO} {
  _builtin_getArgs()
}

-- Note: Environment variables and other OS interactions
-- will be added in future versions
```

### Phase 3: Tests (1 hour)

**File**: `tests/cli_args_test.ail`

```ailang
module tests/cli_args_test

import std/env (getArgs)
import std/io (println)
import std/list (length)

export func main() -> () ! {IO} {
  let args = getArgs();

  -- Test 1: Print number of args
  println("argc: " ++ show(length(args)));

  -- Test 2: Print each arg
  func printArgs(xs: [string]) -> () ! {IO} =
    match xs {
      [] => (),
      x :: rest => {
        println("arg: " ++ x);
        printArgs(rest)
      }
    };

  printArgs(args)
}
```

**Test cases**:
```bash
# Test: No arguments
ailang run --caps IO tests/cli_args_test.ail
# Expected: argc: 0

# Test: Single argument
ailang run --caps IO tests/cli_args_test.ail hello
# Expected: argc: 1
#           arg: hello

# Test: Multiple arguments
ailang run --caps IO tests/cli_args_test.ail foo bar baz
# Expected: argc: 3
#           arg: foo
#           arg: bar
#           arg: baz

# Test: Arguments with spaces (quoted)
ailang run --caps IO tests/cli_args_test.ail "hello world"
# Expected: argc: 1
#           arg: hello world
```

**File**: `internal/builtins/env_test.go`

```go
package builtins_test

import (
	"testing"
	"github.com/yourusername/ailang/internal/builtins"
	"github.com/yourusername/ailang/internal/effects/testctx"
)

func TestGetArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"no_args", []string{}, "[]"},
		{"single_arg", []string{"hello"}, "[\"hello\"]"},
		{"multiple_args", []string{"foo", "bar"}, "[\"foo\", \"bar\"]"},
		{"with_spaces", []string{"hello world"}, "[\"hello world\"]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.New().WithArgs(tt.args)
			result, err := builtins.GetArgs(ctx, nil)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}
```

### Phase 4: Documentation & Teaching Prompt (30 minutes)

**File**: `prompts/v0.4.6.md`

**Add to stdlib section**:
```markdown
### Command-Line Arguments (std/env)

**Import**: `import std/env (getArgs)`

**Access CLI arguments**:
```ailang
import std/env (getArgs)
import std/io (println)

export func main() -> () ! {IO} {
  let args = getArgs();  -- Returns [string]

  match args {
    [] => println("No arguments"),
    [name] => println("Hello, " ++ name),
    _ => println("Multiple arguments")
  }
}
```

**Running with arguments**:
```bash
ailang run --entry main --caps IO app.ail Alice Bob
# args = ["Alice", "Bob"]
```

**Notes**:
- `getArgs()` returns arguments only (no program name)
- Requires `! {IO}` effect
- Empty list if no arguments provided
```

**File**: `docs/guides/cli-arguments.md` (new guide)

**Content**: Usage examples, common patterns (help text, flag parsing, etc.)

### Phase 5: Eval Testing (1 hour)

**Run targeted eval**:
```bash
# Test cli_args benchmark specifically
ailang eval-suite --benchmarks cli_args \
  --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash \
  --output eval_results/cli_args_test

# Expected: 0% → 80%+ success rate
```

**If failures persist**: Iterate on teaching prompt examples

**Run full baseline**:
```bash
make eval-baseline EVAL_VERSION=v0.4.6 FULL=true
```

**Expected improvement**:
- cli_args: 0% → 80%+
- Overall: 75% → 77%+ (with prompt improvements)

## Security Considerations

**No special security concerns**:
- ✅ Read-only access (can't modify args)
- ✅ Requires explicit `! {IO}` effect (capability-based)
- ✅ No shell injection (just strings)
- ✅ No process spawning or elevation

**Best practices**:
- ✅ Don't trust user input (validate in application code)
- ✅ Sanitize before using in file paths or commands

## Alternatives Considered

### Alt 1: Environment Variables Only
**Rejected**: CLI args are more common for simple tools

### Alt 2: Complex Argument Parser
**Rejected**: Too complex for v0.4.6. Keep simple, add parsing library later if needed.

### Alt 3: REPL Support
**Question**: Should REPL support args?
**Answer**: No - REPL is interactive, doesn't make sense. Only `ailang run` supports args.

## Migration and Compatibility

**Breaking changes**: None (new feature)

**Deprecations**: None

**Backwards compatibility**: Existing code unaffected

## Success Criteria

1. ✅ `getArgs()` builtin implemented and tested
2. ✅ `std/env` module created
3. ✅ Unit tests pass (100% coverage)
4. ✅ Integration tests pass (cli_args_test.ail)
5. ✅ Teaching prompt includes CLI args example
6. ✅ `cli_args` benchmark: 0% → 80%+ success
7. ✅ Overall eval: 75% → 77%+ success (with prompt improvements)

## Timeline

- **Phase 1** (Builtin): 1 hour
- **Phase 2** (Stdlib): 30 minutes
- **Phase 3** (Tests): 1 hour
- **Phase 4** (Docs): 30 minutes
- **Phase 5** (Eval): 1 hour

**Total**: 4 hours

**Target completion**: v0.4.6 release (early December 2025)

## Related Work

- **Prompt improvements**: `teaching-prompt-improvements.md`
- **Eval analysis**: `EVAL_FAILURE_ANALYSIS_v0_4_5.md`
- **Implementation gaps**: `implementation-gaps-analysis.md`
