# M-LANG-CLI-ARGS: Command-Line Arguments Access

**Status**: Planned
**Target**: v0.4.6
**Priority**: P1 (High)
**Estimated**: 6 hours
**Dependencies**: Env capability (already implemented in v0.4.4)
**Impact**: +2.1% eval success rate (cli_args benchmark: 0% → 80%+)

## AI-First DX Scoring

**Feature evaluation against AILANG's design principles:**

| Axis | Score | Justification |
|------|-------|---------------|
| **Syntactic Noise** | +1 | Removes need for complex arg parsing boilerplate |
| **Semantic Clarity** | +1 | Explicit `! {Env}` effect makes external dependency clear |
| **Determinism** | +1 | Read-only, capability-gated access to fixed process state |
| **Token Cost** | +1 | Simple API (`getArgs()`), no arg parsing DSL needed |
| **Total** | **+4** | ✅ Strong alignment with AI-first principles |

**Decision**: **APPROVE** - Net score +4 indicates excellent fit for AILANG.

**Why this feature aligns with AILANG's vision:**
- **Minimal API surface**: Single function `getArgs()` vs complex flag parsing libraries
- **Compositional**: Returns `[string]`, works with existing pattern matching and list operations
- **Explicit effects**: `! {Env}` makes external dependency visible to AI models
- **No magic**: No automatic `main(args)` injection, no implicit behavior
- **Capability-based**: Requires explicit `--caps Env` grant, security model clear
- **Consistent**: Extends existing `std/env` (`getEnv`, `hasEnv`) with same patterns

## Problem Statement

The `cli_args` benchmark fails across all 6 models (0% success rate) because **AILANG has no way to access command-line arguments**.

**Current State:**
- Programs cannot read arguments passed to `ailang run program.ail arg1 arg2`
- No workaround available (unlike other missing features where patterns exist)
- Benchmark impact: 6/284 failures (2.1%)
- Real-world impact: CLI tools, scripts, utilities cannot be built

**Why This Matters:**
- **For AI models**: Common benchmark pattern across all languages (Python `sys.argv`, Node `process.argv`)
- **For users**: Essential for command-line tools (the primary use case for AILANG programs)
- **For ecosystem**: Blocks development of CLI utilities, testing frameworks, build tools

**User request**: Consistent with AILANG philosophy (deterministic, explicit effects, capability-based)

## Goals

**Primary Goal**: Enable programs to access command-line arguments through a simple, effect-typed API

**Success Metrics:**
- `cli_args` benchmark: 0% → 80%+ success rate
- API simplicity: Single function `getArgs() -> [string] ! {Env}`
- Consistency: Aligns with existing `std/env` module (`getEnv`, `hasEnv`)
- Security: Capability-gated (requires `--caps Env`)
- Teaching: Models can learn from clear examples in one prompt iteration

**Non-Goals:**
- Complex argument parsing libraries (flags, options, subcommands)
- Environment variable access (already implemented via `getEnv`/`hasEnv`)
- Process spawning or piping
- REPL support for arguments (doesn't make sense for interactive shell)

## Solution Design

### Design Philosophy: Alignment with Env Capability

**This feature extends the existing `std/env` module and `Env` capability.**

**Conceptual model**: CLI arguments and environment variables are both part of the "process environment":
- Fixed for the lifetime of the process
- Read-only from inside the program
- Come from outside the pure world
- NOT streaming I/O (that's what `IO` is for)

**Consistency with existing implementation:**
- `getEnv(name: string) -> string ! {Env}` - read environment variable
- `hasEnv(name: string) -> bool ! {Env}` - check if env var exists
- `getArgs() -> [string] ! {Env}` - **NEW**: read CLI arguments

**All three functions:**
- Live in `std/env` module
- Use `! {Env}` effect (not `! {IO}`)
- Require `--caps Env` capability at runtime
- Return `E_ENV_CAP_MISSING` if capability not granted

**This design maintains a single coherent story:**
*"All reads from the process environment (env vars, CLI args) are `! {Env}` and require `--caps Env`."*

### API: `std/env.getArgs`

**Module**: `std/env` (existing stdlib module)

**Function signature:**
```ailang
export func getArgs() -> [string] ! {Env}
```

**Returns**: List of command-line arguments (excluding program name)

**Effect**: `! {Env}` - reading from process environment (consistent with `getEnv`, `hasEnv`)

**Example usage:**
```ailang
module myapp/main

import std/env (getArgs)
import std/io (println)
import std/list (length)

export func main() -> () ! {IO, Env} {
  let args = getArgs();
  println("Number of arguments: " ++ show(length(args)));

  match args {
    [] => println("Usage: program <name>"),
    [name] => println("Hello, " ++ name ++ "!"),
    _ => println("Too many arguments")
  }
}
```

**Running:**
```bash
$ ailang run --entry main --caps IO,Env myapp/main.ail Alice
Number of arguments: 1
Hello, Alice!

$ ailang run --entry main --caps IO,Env myapp/main.ail
Number of arguments: 0
Usage: program <name>
```

### Design Decisions

#### 1. Effect Choice: Why `! {Env}` (not `! {IO}`)?

**Rationale**: Arguments are part of the process environment, like environment variables

**Arguments for `! {Env}`:**
- ✅ **Consistent with existing Env capability** (`getEnv`, `hasEnv`)
- ✅ **Conceptually aligned**: CLI args and env vars are both "process environment"
- ✅ **Fixed for process lifetime** (not streaming I/O)
- ✅ **Read-only external input** (same category as env vars)
- ✅ **Enables future testing mechanisms** (override args like env vars)

**Why not `! {IO}`?**
- IO is for actual I/O streaming (stdin/stdout, files, sockets)
- CLI args don't involve ongoing I/O operations
- Would be inconsistent: `getEnv ! {Env}` but `getArgs ! {IO}`?

**Why not pure?**
- Arguments come from outside the program (external input)
- Different invocations can have different arguments
- Must be explicit about external dependencies

**Decision**: `! {Env}` for consistency with existing environment access patterns

#### 2. Return Value: `[string]` vs Other Options

**Option 1: `[string]` (CHOSEN)**
```ailang
getArgs() -> [string]
-- Returns: ["arg1", "arg2", "arg3"]
```
**Pros:**
- ✅ Simple, functional
- ✅ Matches Haskell (`getArgs :: IO [String]`)
- ✅ Easy to pattern match
- ✅ Composes with existing list functions

**Option 2: `{argc: int, argv: [string]}`**
```ailang
getArgs() -> {argc: int, argv: [string]}
-- Returns: {argc: 3, argv: ["arg1", "arg2", "arg3"]}
```
**Cons:**
- ❌ Redundant (can compute `argc` from `length(argv)`)
- ❌ More complex API
- ❌ Doesn't follow functional style

**Option 3: Include program name**
```ailang
getArgs() -> [string]
-- Returns: ["program_name", "arg1", "arg2"]
```
**Cons:**
- ❌ Program name not useful in AILANG context (no `argv[0]` equivalent)
- ❌ Inconsistent with modern languages (most skip program name)
- ❌ Confusing for benchmarks (expect args only)

**Decision**: Return `[string]` (arguments only, no program name)

#### 3. Alternative Rejected: Main Function Signature

**Option**: Automatically pass args to `main` function
```ailang
export func main(args: [string]) -> () ! {IO} {
  -- args available as parameter
}
```

**Rejected because:**
1. Breaks existing code (all `main` functions have signature `() -> ()`)
2. Less flexible (what if you don't need args?)
3. Inconsistent with other effects (`readFile`, `getEnv` are explicit)
4. Makes `main` special (not just another function)

**Decision**: Use explicit `getArgs()` function for clarity and consistency

## Implementation Plan

### Phase 1: Builtin Function (1.5 hours)

**File**: `internal/builtins/spec.go`

**Add builtin** (following M-DX1 convention):
```go
{
	Name:      "_env_getArgs",
	Module:    "std/env",
	Type:      types.NewBuilder().
		Func(nil).              // No parameters (nullary function)
		WithEffect("Env").      // Env effect
		Returns(types.TList(types.TString)),
	GoImpl:    builtins.EnvGetArgs,
	Pure:      false,
	Doc:       "Get command-line arguments as list of strings (requires Env capability)",
},
```

**Note**: Following the M-DX1 builtin convention:
- Builtin name: `_env_getArgs` (underscored, module-prefixed)
- Wrapper function in `std/env.ail`: `getArgs()`
- Go implementation: `EnvGetArgs` in `internal/builtins/env.go`

This matches existing pattern:
- `_env_getEnv` → `getEnv()` → `EnvGetEnv`
- `_env_hasEnv` → `hasEnv()` → `EnvHasEnv`

**File**: `internal/builtins/env.go` (extend existing file)

```go
// EnvGetArgs returns command-line arguments (excluding program name)
// Requires Env capability
func EnvGetArgs(ctx eval.EffectContext, args []eval.Value) (eval.Value, error) {
	// CRITICAL: Check Env capability is granted
	if !ctx.HasCap("Env") {
		return nil, eval.NewError(
			"E_ENV_CAP_MISSING",
			"Env capability not granted - use --caps Env to access CLI arguments",
		)
	}

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

**Note**: Capability enforcement is **required** to align with existing `EnvGetEnv` and `EnvHasEnv` implementations.

**File**: `internal/effects/context.go`

**Add to EffectContext interface:**
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

**Pass args to effect context:**
```go
func Run(prog *core.Program, caps []string, args []string) error {
	// ... existing code ...

	// Create effect context with CLI args
	effCtx := effects.NewRealEffContext(args)

	// ... rest of execution ...
}
```

**File**: `cmd/ailang/main.go`

**Collect args and pass to runtime** (CAREFULLY!):
```go
func runCommand(c *cli.Context) error {
	// ... existing code ...

	// CRITICAL: Parse positional arguments correctly
	// CLI invocations:
	//   ailang run foo.ail a b c              → programArgs = [a, b, c]
	//   ailang run --entry main foo.ail a b   → programArgs = [a, b]
	//   ailang run --caps IO foo.ail a        → programArgs = [a]
	//
	// Strategy: Use CLI library's args parsing (don't hand-roll Slice()[1:])
	// The CLI lib already handles flags vs positional args correctly

	allArgs := c.Args().Slice()

	// Find the .ail file position (first positional arg after flags)
	fileArgIdx := -1
	for i, arg := range allArgs {
		if strings.HasSuffix(arg, ".ail") {
			fileArgIdx = i
			break
		}
	}

	var programArgs []string
	if fileArgIdx >= 0 && fileArgIdx < len(allArgs)-1 {
		// Everything after the .ail file is a program argument
		programArgs = allArgs[fileArgIdx+1:]
	} else {
		// No args provided
		programArgs = []string{}
	}

	// Run with args
	if err := runtime.Run(prog, caps, programArgs); err != nil {
		return err
	}

	return nil
}
```

**Testing CLI argument parsing:**
```go
// Add tests for various invocation styles:
// - ailang run test.ail a b c
// - ailang run --caps IO test.ail a b
// - ailang run --entry main --caps IO,Env module.ail arg1 arg2
```

### Phase 2: Standard Library Module (30 minutes)

**File**: `std/env.ail` (extend existing module)

```ailang
-- Environment and OS interaction
module std/env

-- Existing functions (from previous implementation):
-- export func getEnv(name: string) -> string ! {Env}
-- export func hasEnv(name: string) -> bool ! {Env}

-- Get command-line arguments
-- Returns arguments passed to the program (excluding program name)
-- Example: ailang run --caps Env app.ail arg1 arg2 → ["arg1", "arg2"]
-- Requires: Env capability (--caps Env)
export func getArgs() -> [string] ! {Env} {
  _env_getArgs()
}
```

**Consistency note**: All `std/env` functions now use the `! {Env}` effect:
- `getEnv(name)` - read environment variable
- `hasEnv(name)` - check if env var exists
- `getArgs()` - read CLI arguments

### Phase 3: Tests (1.5 hours)

**File**: `tests/cli_args_test.ail`

```ailang
module tests/cli_args_test

import std/env (getArgs)
import std/io (println)
import std/list (length)

export func main() -> () ! {IO, Env} {
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

**Test cases:**
```bash
# Test: No arguments
ailang run --caps IO,Env tests/cli_args_test.ail
# Expected: argc: 0

# Test: Single argument
ailang run --caps IO,Env tests/cli_args_test.ail hello
# Expected: argc: 1
#           arg: hello

# Test: Multiple arguments
ailang run --caps IO,Env tests/cli_args_test.ail foo bar baz
# Expected: argc: 3
#           arg: foo
#           arg: bar
#           arg: baz

# Test: Arguments with spaces (quoted)
ailang run --caps IO,Env tests/cli_args_test.ail "hello world"
# Expected: argc: 1
#           arg: hello world

# Test: Missing Env capability (should fail with clear error)
ailang run --caps IO tests/cli_args_test.ail hello
# Expected: ERROR - E_ENV_CAP_MISSING
#           "Env capability not granted - use --caps Env to access CLI arguments"
```

**File**: `internal/builtins/env_test.go` (extend existing test file)

```go
func TestEnvGetArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		hasCap   bool
		expected string
		wantErr  bool
		errCode  string
	}{
		{"no_args_with_cap", []string{}, true, "[]", false, ""},
		{"single_arg_with_cap", []string{"hello"}, true, "[\"hello\"]", false, ""},
		{"multiple_args_with_cap", []string{"foo", "bar"}, true, "[\"foo\", \"bar\"]", false, ""},
		{"with_spaces", []string{"hello world"}, true, "[\"hello world\"]", false, ""},
		{"no_cap_fails", []string{"arg"}, false, "", true, "E_ENV_CAP_MISSING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.New().WithArgs(tt.args)
			if tt.hasCap {
				ctx = ctx.WithCap("Env")
			}

			result, err := builtins.EnvGetArgs(ctx, nil)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				// Check error code matches
				if !strings.Contains(err.Error(), tt.errCode) {
					t.Errorf("expected error code %s, got %v", tt.errCode, err)
				}
				return
			}

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

**Critical test coverage:**
1. ✅ Works with Env capability
2. ✅ Fails with clear error without Env capability
3. ✅ Handles empty args list
4. ✅ Handles args with spaces
5. ✅ Returns correct list structure

### Phase 4: Documentation & Teaching Prompt (1 hour)

**File**: `prompts/v0.4.6.md`

**Add to stdlib section (std/env):**
```markdown
### Process Environment (std/env)

**Import**: `import std/env (getEnv, hasEnv, getArgs)`

**Environment variables:**
```ailang
import std/env (getEnv, hasEnv)

export func main() -> () ! {Env} {
  if hasEnv("HOME") then
    println(getEnv("HOME"))
  else
    println("HOME not set")
}
```

**Command-line arguments:**
```ailang
import std/env (getArgs)
import std/io (println)

export func main() -> () ! {IO, Env} {
  let args = getArgs();  -- Returns [string]

  match args {
    [] => println("Usage: program <name>"),
    [name] => println("Hello, " ++ name ++ "!"),
    _ => println("Too many arguments")
  }
}
```

**Running with arguments:**
```bash
ailang run --entry main --caps IO,Env app.ail Alice
# args = ["Alice"]
```

**Important notes:**
- All `std/env` functions require `! {Env}` effect
- `getArgs()` returns arguments only (excludes program name)
- Empty list `[]` if no arguments provided
- Use `--caps Env` (or `--caps IO,Env` if also doing I/O)

**CLI Anti-Patterns** (DON'T do these):
```ailang
-- ❌ WRONG: No argv[0] - first element is first argument
let programName = head(getArgs())  -- This is the first arg, not program name!

-- ❌ WRONG: No argc - use length
let argc = argc(getArgs())  -- No such function! Use length(getArgs())

-- ❌ WRONG: No built-in flag parsing
let verbose = parseFlag("--verbose", getArgs())  -- No such function!

-- ✅ CORRECT: Manual pattern matching for flags
match args {
  ["--help"] => showHelp(),
  ["--version"] => showVersion(),
  [name] => greet(name),
  _ => println("Invalid arguments")
}
```
```

**File**: `docs/guides/cli-arguments.md` (new guide)

**Content**: Usage examples, common patterns (help text, flag parsing, argument validation)

### Phase 5: Eval Testing (1.5 hours)

**Critical: Update eval harness to pass args**

**File**: `internal/eval_harness/runner.go`

```go
// Ensure benchmark-defined args are passed through to the runtime
func RunBenchmark(bench *Benchmark, model string) (*Result, error) {
	// ... existing code ...

	// Pass benchmark's expected args to the AILANG runtime
	if err := runtime.Run(prog, bench.Caps, bench.Args); err != nil {
		return nil, err
	}

	// ... validation ...
}
```

**Verification test:**
```go
// Ensure AILANG sees the args the evaluator thinks it's passing
func TestEvalHarnessPassesArgs(t *testing.T) {
	bench := &Benchmark{
		Args: []string{"test", "arg"},
		// ...
	}
	result := RunBenchmark(bench, "test-model")
	// Verify benchmark output shows it received ["test", "arg"]
}
```

**Run targeted eval:**
```bash
# Test cli_args benchmark specifically
ailang eval-suite --benchmarks cli_args \
  --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash \
  --output eval_results/cli_args_test

# Expected: 0% → 80%+ success rate
```

**If failures persist**: Iterate on teaching prompt examples

**Run full baseline:**
```bash
make eval-baseline EVAL_VERSION=v0.4.6 FULL=true
```

**Expected improvement:**
- cli_args: 0% → 80%+
- Overall: 75% → 77%+ (with prompt improvements)

### Files to Modify/Create

**New files:**
- `tests/cli_args_test.ail` (~30 LOC) - Integration test
- `docs/guides/cli-arguments.md` (~200 LOC) - User guide

**Modified files:**
- `internal/builtins/spec.go` (+10 LOC) - Register `_env_getArgs`
- `internal/builtins/env.go` (+25 LOC) - `EnvGetArgs` implementation
- `internal/builtins/env_test.go` (+45 LOC) - Unit tests
- `internal/effects/context.go` (+10 LOC) - Add `GetArgs()` method
- `internal/runtime/runtime.go` (+5 LOC) - Pass args to context
- `cmd/ailang/main.go` (+20 LOC) - Parse and pass CLI args
- `std/env.ail` (+8 LOC) - Add `getArgs()` wrapper
- `prompts/v0.4.6.md` (+50 LOC) - Teaching prompt section
- `internal/eval_harness/runner.go` (+5 LOC) - Pass args to runtime

**Total new code**: ~408 LOC

## Examples

### Example 1: Basic Argument Access

**AILANG code:**
```ailang
module examples/hello_args

import std/env (getArgs)
import std/io (println)

export func main() -> () ! {IO, Env} {
  match getArgs() {
    [] => println("Hello, World!"),
    [name] => println("Hello, " ++ name ++ "!"),
    _ => println("Too many arguments")
  }
}
```

**Usage:**
```bash
$ ailang run --caps IO,Env examples/hello_args.ail
Hello, World!

$ ailang run --caps IO,Env examples/hello_args.ail Alice
Hello, Alice!

$ ailang run --caps IO,Env examples/hello_args.ail Alice Bob
Too many arguments
```

### Example 2: Argument Validation

**AILANG code:**
```ailang
module examples/validate_args

import std/env (getArgs)
import std/io (println)
import std/list (length)

func validateArgs(args: [string]) -> bool =
  length(args) == 2

export func main() -> () ! {IO, Env} {
  let args = getArgs();

  if validateArgs(args) then {
    match args {
      [input, output] => {
        println("Input: " ++ input);
        println("Output: " ++ output)
      },
      _ => println("Unexpected pattern")
    }
  } else {
    println("Usage: program <input> <output>")
  }
}
```

### Example 3: Flag Parsing (Manual)

**AILANG code:**
```ailang
module examples/flags

import std/env (getArgs)
import std/io (println)

export func main() -> () ! {IO, Env} {
  match getArgs() {
    ["--help"] => println("Usage: program [--help | --version | <name>]"),
    ["--version"] => println("Version 1.0.0"),
    [name] => println("Hello, " ++ name ++ "!"),
    [] => println("Hello, World!"),
    _ => println("Error: Too many arguments")
  }
}
```

**Before this feature (IMPOSSIBLE):**
```ailang
-- No way to access arguments at all!
-- Can only write programs with hard-coded data
```

**After this feature (SIMPLE):**
```ailang
import std/env (getArgs)
-- Simple, explicit, compositional
let args = getArgs();
match args { ... }
```

## Testing Strategy

**Unit tests:**
- `EnvGetArgs` with/without capability (`internal/builtins/env_test.go`)
- Empty args, single arg, multiple args, args with spaces
- Error cases (no capability, invalid inputs)
- Test coverage: 100% on new code

**Integration tests:**
- End-to-end: `ailang run --caps IO,Env tests/cli_args_test.ail arg1 arg2`
- Capability enforcement: Run without `--caps Env`, expect error
- CLI parsing: Test multiple invocation styles
- Eval harness: Verify args passed through correctly

**Manual testing:**
- Run examples from docs (`examples/hello_args.ail`)
- Test various argument patterns (none, one, many, with spaces)
- Verify error messages are helpful

**Benchmark testing:**
- Run `cli_args` benchmark against all 6 models
- Target: 0% → 80%+ success rate
- Analyze failures, update prompt if needed

## Security Considerations

**Capability-based security:**
- Requires explicit `--caps Env` grant
- Without capability, fails with `E_ENV_CAP_MISSING`
- Clear error message guides user to add capability
- Consistent with `getEnv`/`hasEnv` security model

**No special security concerns:**
- ✅ Read-only access (can't modify args)
- ✅ No shell injection (just strings)
- ✅ No process spawning or elevation
- ✅ No file system access (separate capability)

**Best practices for users:**
- Validate argument structure with pattern matching
- Sanitize before using in file paths or external commands
- Don't trust user input (same as any CLI program)

## Success Criteria

- [ ] `_env_getArgs` builtin implemented with capability enforcement
- [ ] `std/env` module extended with `getArgs()` wrapper
- [ ] Unit tests pass (100% coverage, including capability checks)
- [ ] Integration tests pass (`tests/cli_args_test.ail`)
- [ ] Capability enforcement tested (with and without `--caps Env`)
- [ ] CLI argument parsing works for all invocation styles
- [ ] Eval harness passes args to runtime correctly
- [ ] Teaching prompt includes examples with anti-patterns
- [ ] `cli_args` benchmark: 0% → 80%+ success rate
- [ ] Overall eval: 75% → 77%+ success
- [ ] Documentation updated (CLAUDE.md, prompts/, docs/)
- [ ] All tests passing
- [ ] No regressions in existing tests

## Timeline

**Phase 1** (Builtin + Capability): 1.5 hours
- Create builtin spec with correct type signature
- Implement `EnvGetArgs` with capability check
- Add to effect context interface
- Wire through runtime and CLI

**Phase 2** (Stdlib): 30 minutes
- Extend `std/env.ail` with `getArgs()` wrapper
- Document function with examples

**Phase 3** (Tests): 1.5 hours
- Unit tests (capability, args handling)
- Integration tests (`tests/cli_args_test.ail`)
- CLI parsing tests

**Phase 4** (Docs): 1 hour
- Update teaching prompt with examples
- Add anti-patterns section
- Create user guide

**Phase 5** (Eval): 1.5 hours
- Update eval harness
- Run targeted benchmark
- Iterate on prompt if needed

**Total: 6 hours** (accounts for capability enforcement, careful CLI parsing, eval integration)

**Target completion**: v0.4.6 release (December 2025)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| CLI parsing breaks existing invocations | High | Test all invocation styles; careful parsing logic |
| Models hallucinate `argv[0]` or `argc()` | Medium | Add explicit anti-patterns section to prompt |
| Capability check broken | High | Comprehensive unit tests for capability enforcement |
| Eval harness doesn't pass args | High | Add verification test before running benchmarks |
| Teaching prompt unclear | Medium | Include multiple examples with different patterns |

## Non-Goals

**Not in this feature:**
- Complex argument parsing library (flags, options, subcommands) - Keep simple, defer to user code
- Automatic `main(args)` injection - Explicit is better than implicit
- REPL argument support - Doesn't make sense for interactive shell
- Argument validation framework - Use pattern matching
- Process environment modification - Read-only, consistent with Env design

**Deferred to future work:**
- `--env-like` testing flag for overriding args (useful for tests)
- Standard library flag parsing utilities
- Argument schema/validation DSL

## References

- **Existing Env capability**: `std/env` module with `getEnv`, `hasEnv`
- **M-DX1 builtin pattern**: Naming convention (`_env_*`), type builder
- **Haskell prior art**: `getArgs :: IO [String]` (but we use `! {Env}`)
- **Eval framework**: `internal/eval_harness/` for benchmark testing
- **Teaching prompt**: `prompts/v0.4.6.md` with anti-patterns

## Implementation Notes

### Critical Considerations

**1. Type Signature (Builtin Spec)**

The correct implementation must be:

```go
Type: types.NewBuilder().
    Func(nil).              // No parameters (nullary function)
    WithEffect("Env").      // Effect tag
    Returns(types.TList(types.TString))
```

**NOT**:
```go
Type: types.Func(types.TList(types.TString), types.TIO(types.TUnit))
// ❌ This is [string] -> IO(()) which is completely wrong!
```

**2. Capability Enforcement (Required)**

The builtin MUST check the Env capability:

```go
if !ctx.HasCap("Env") {
    return nil, eval.NewError("E_ENV_CAP_MISSING", "...")
}
```

This aligns with `EnvGetEnv` and `EnvHasEnv` implementations.

**3. CLI Argument Parsing (Be Careful!)**

Don't blindly use `Slice()[1:]` - the CLI library may order args differently depending on flags.

Test all these invocation styles:
- `ailang run foo.ail a b c`
- `ailang run --caps Env foo.ail a b`
- `ailang run --entry main --caps IO,Env module.ail arg1 arg2`

**4. Eval Harness Integration**

The `cli_args` benchmark won't pass unless the eval harness actually passes args through to the runtime. Verify with a regression test.

**5. Teaching Prompt**

Include:
- Clear example with `! {IO, Env}`
- Correct shell invocation with `--caps Env`
- **Anti-patterns section** to prevent hallucinated APIs (no `argv[0]`, no `argc()`, no `parseFlag()`)

### Naming Convention (M-DX1)

Following the established pattern:
- **Builtin name**: `_env_getArgs` (underscored, module-prefixed)
- **Wrapper function**: `getArgs()` in `std/env.ail`
- **Go implementation**: `EnvGetArgs` in `internal/builtins/env.go`

This matches:
- `_env_getEnv` → `getEnv()` → `EnvGetEnv`
- `_env_hasEnv` → `hasEnv()` → `EnvHasEnv`

## Future Work

**Potential enhancements (not committed):**
- Testing utilities: `--args "arg1,arg2"` CLI flag to override args (similar to `--env`)
- Standard library: `std/cli` module with common patterns (help text, flag parsing)
- Schema validation: Declarative argument schemas with auto-validation
- Environment parity: Unified `std/env` API for all process environment access

**Builds on this feature:**
- CLI testing framework (can now test programs that take arguments)
- Build tools and scripts (can pass configuration via args)
- Benchmarking tools (can parameterize benchmark runs)
