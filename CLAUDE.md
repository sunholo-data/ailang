# Claude Instructions for AILANG Development

## 🚀 SESSION START ROUTINE

**At the start of EVERY session, I MUST check for agent messages.**

The SessionStart hook automatically runs and should inject inbox messages into my context. If messages appear in system reminders, I acknowledge them. If not, I manually check the inbox.

### What I Do at Session Start

**Step 1: Check system reminders for agent messages**
- The SessionStart hook injects messages automatically
- If I see inbox messages in system reminders → proceed to Step 3

**Step 2: Fallback - Manual inbox check (if no messages in system reminders)**
```bash
ailang agent inbox --unread-only claude-code
```
- This checks both inbox locations (see below)
- If messages exist but didn't appear in system reminders, they'll show here

**Step 3: When I find messages (from hook OR manual check)**
1. **Tell you about them**: "I found 2 unread messages from agents..."
2. **Summarize content**: Explain what the agents reported
3. **Ask what to do**: "Would you like me to review the sprint plan?"

**Step 4: After handling messages**
```bash
ailang agent ack <message-id>    # Acknowledge specific message
# OR
ailang agent ack --all           # Acknowledge all messages
```

**Step 5: If I fail to complete a task**
```bash
ailang agent unack <message-id>  # Moves back to _unread for next session
```

### Two Inbox Locations

**User Inbox** (`~/.ailang/state/messages/inbox/user/`):
- Home directory, persists across projects
- For messages TO the user from any agent
- Check with: `ailang agent inbox --unread-only user`

**Claude-Code Inbox** (`.ailang/state/messages/claude-code/`):
- Project directory, specific to this codebase
- For messages TO claude-code agent (checked by SessionStart hook)
- Check with: `ailang agent inbox --unread-only claude-code`

The SessionStart hook checks BOTH locations automatically.

### Technical Details (Background Info)

**Hooks Configuration:**
- Hooks are configured in `.claude/settings.local.json`
- Hook commands use `$CLAUDE_PROJECT_DIR` for absolute paths
- All hook activity logged to `~/.ailang/state/hooks.log`

**SessionStart Hook:**
- Script: `scripts/hooks/session_start.sh`
- Reads hook event JSON from stdin (Claude Code standard)
- Checks both inbox locations on every session start
- Outputs message summaries to stdout (appears in system reminders)
- **Does NOT auto-mark as read** (prevents race conditions)

**Stop Hook (Agent Handoff):**
- Script: `scripts/hooks/agent_handoff.sh`
- Reads hook event JSON from stdin
- Detects design docs created in session (last 5 minutes)
- Sends handoff messages to sprint-planner agent

**Message Lifecycle:**
1. Agent sends message → lands in `_unread` or `.pending.json`
2. SessionStart hook injects into Claude's context (or manual check)
3. Claude acknowledges → moves to `_processed` or `_read`
4. If Claude fails → un-acknowledge → back to `_unread` for retry

**Manual inbox commands:**
```bash
# Check user inbox (home directory: ~/.ailang/state/messages/inbox/user/)
ailang agent inbox user
ailang agent inbox --unread-only user

# Check claude-code inbox (project directory: .ailang/state/messages/claude-code/)
ailang agent inbox claude-code
ailang agent inbox --unread-only claude-code

# NOTE: Flags must come BEFORE agent ID!
# ✅ Correct:   ailang agent inbox --unread-only claude-code
# ❌ Incorrect: ailang agent inbox claude-code --unread-only

# Acknowledge a specific message (moves to _processed)
ailang agent ack msg_20251025_155729_a5f3e77ee975

# Acknowledge all unread messages at once
ailang agent ack --all

# Un-acknowledge a message if task failed (moves back to _unread)
ailang agent unack msg_20251025_155729_a5f3e77ee975

# Archive old messages
ailang agent inbox user --archive
```

---

## ⚠️ CRITICAL PRINCIPLES

### 0. NEVER DESTROY LOCAL WORK WITH GIT OPERATIONS

**🚨 ABSOLUTELY FORBIDDEN GIT OPERATIONS:**

**NEVER run these commands - they destroy uncommitted work:**
- ❌ `git checkout <branch>` when there are uncommitted changes
- ❌ `git pull` on a branch with local commits (use `git status` first!)
- ❌ `git reset --hard`
- ❌ `git clean -fd`
- ❌ `git stash` followed by branch switching that causes fast-forward
- ❌ Any command that would overwrite or discard user's work

**The Dev Branch Disaster (October 2025):**
```bash
# ❌ WRONG - This destroyed local work on dev branch
git stash                    # Hides user's work
git checkout dev             # Switches branch
git pull                     # Fast-forwards, making stashed work incompatible
# Result: User's uncommitted work on dev branch is LOST
```

**✅ CORRECT approach when user asks to fix something on another branch:**

1. **ALWAYS check git status first:**
   ```bash
   git status  # See what branch we're on and what's uncommitted
   ```

2. **If there are uncommitted changes, ASK THE USER:**
   ```
   "I see you have uncommitted changes on the current branch.
    How would you like me to handle them before switching branches?
    Options:
    1. Commit them first
    2. Create a new branch from here for the fix
    3. Tell me it's safe to stash them"
   ```

3. **If fixing docs/CI issues, work on CURRENT branch:**
   - If it's a simple fix (broken links, typos, CI failures), make the fix on whatever branch you're on
   - Push to that branch, don't force a branch switch
   - User can merge/cherry-pick later if needed

4. **NEVER assume it's safe to discard or stash work:**
   - User's uncommitted work may represent hours of effort
   - Stashing can cause conflicts or make work hard to recover
   - Switching branches can trigger unwanted merges/rebases

**Recovery if you've already done damage:**
```bash
git reflog                   # Find lost commits
git stash list               # Find stashed work
git stash show -p stash@{0}  # Preview stashed changes
git fsck --lost-found        # Find orphaned commits
```

**Safe alternatives:**
- ✅ Make fixes on current branch, push, let user handle merging
- ✅ Ask user before ANY branch switching
- ✅ Create new branches instead of switching to existing ones
- ✅ Use `git status` before every git operation

### 1. ALWAYS USE EXISTING TOOLS FIRST

**Before writing ANY new script or code:**
1. ✅ Check `make help` for existing targets
2. ✅ Check `tools/` directory for existing scripts
3. ✅ Check this CLAUDE.md for documented workflows
4. ✅ Search codebase: `grep -r "function_name" internal/`

**Common mistakes to avoid:**
- ❌ Writing new bash scripts when `make` targets or `ailang` commands exist
- ❌ Creating new analysis tools when M-EVAL-LOOP Go implementation exists
- ❌ Guessing model names instead of checking `internal/eval_harness/models.yml`
- ❌ Ignoring documented workflows in CLAUDE.md
- ❌ Manually extracting/formatting data when automated tools exist
- ❌ Guessing which tools to use for benchmarks/evals - ALWAYS use eval-orchestrator agent

### 2. NO SILENT FALLBACKS - FAIL LOUDLY

**CRITICAL LESSON**: Silent fallbacks hide bugs and produce wrong data that users trust.

**The Cost Calculation Bug (Oct 2024):**
```go
// ❌ WRONG - Silent fallback hid 61x cost overestimation for YEARS
rate, ok := rates[model]
if !ok {
    rate = 0.03  // Default to GPT-4 pricing if unknown
}
return float64(tokens) / 1000.0 * rate
```

**Impact**: All modern models (GPT-5, Gemini 2.5, Claude Sonnet 4.5) used wrong pricing.
Users trusted inflated costs. Bug was invisible until someone questioned the numbers.

**The Principle:**
```go
// ✅ CORRECT - Return 0 or error to force investigation
if GlobalModelsConfig == nil {
    return 0.0  // Better to see $0.00 than trust wrong data
}

cost, err := GlobalModelsConfig.CalculateCostForModel(model, inputTokens, outputTokens)
if err != nil {
    return 0.0  // NO SILENT FALLBACKS - we want to know when pricing is missing
}
```

**When to apply:**
- ✅ Pricing/cost calculations (return $0.00 if unknown)
- ✅ Model configurations (fail if model not in models.yml)
- ✅ Required environment variables (fail if missing, don't use defaults)
- ✅ Data validation (reject invalid data, don't silently fix)
- ✅ Configuration loading (fail if config invalid, don't use built-in defaults)

**When fallbacks ARE okay:**
- ✅ UI defaults (empty state, placeholder text)
- ✅ Optional features (graceful degradation of non-critical features)
- ✅ Caching (miss → fetch from source)
- ✅ Performance optimizations (slow path if fast path unavailable)

**Rule of thumb:** If the fallback value affects data integrity, business logic, or user decisions → **NO FALLBACK**. Return zero, null, or error instead.

### 3. CORETYPEINFO INVARIANT - COMPLETE TYPE COVERAGE

**CRITICAL COMPILER INVARIANT**: Every Core node must have an entry in CoreTypeInfo before lowering; validation enforces this (M-DX4).

**The Contract:**
```go
// ✅ ENFORCED - Validation runs before lowering in all paths
// File pipeline:   internal/pipeline/pipeline.go:228
// Module pipeline: internal/pipeline/pipeline.go:631
// REPL:            internal/repl/repl_eval.go:113

if err := ValidateCoreTypeInfo(coreProg, typeChecker.CoreTI); err != nil {
    return result, fmt.Errorf("CoreTypeInfo validation failed: %w", err)
}
```

**Why this matters:**
- Lowering relies on CoreTypeInfo for type-directed code generation
- Missing types cause "cannot lower unknown variant" panics with no context
- Validation provides clear diagnostics: NodeID, ExprKind, Position, Hint
- Forward-compatible: Polymorphic types (type variables) are accepted

**Validated properties:**
- ✅ All 20+ Core node types checked (Var, Lit, Lambda, Let, LetRec, App, If, Match, BinOp, UnOp, Intrinsic, Record, List, Tuple, DictAbs, DictApp, etc.)
- ✅ Presence required, not concreteness (type variables OK for polymorphic code)
- ✅ Performance: O(n) linear, zero allocations (191ns for 10 nodes, 34μs for 1000 nodes)

**Error example:**
```
CoreTypeInfo validation failed: missing type information for Core nodes

Missing Lit(Float) types (1 nodes):
  • NodeID 42 at line 5, col 12
    Hint: This usually means defaulting/substitution wasn't applied to CoreTI.

Debug with: ailang debug ast <file> --show-types --compact
```

**Monomorphization compatibility (v0.4.0+):**
Validated nodes may carry polymorphic types (α, β, etc.); these are accepted as long as a type exists. Monomorphization will later specialize them to concrete types before final codegen. The validator checks *presence*, not *concreteness*.

**Implementation:**
- Validator: `internal/pipeline/validate_coretypeinfo.go` (343 LOC)
- Tests: `internal/pipeline/validate_coretypeinfo_test.go` (417 LOC, 8/8 passing)
- Benchmarks: `internal/pipeline/validate_coretypeinfo_bench_test.go` (117 LOC)

### 4. MONOMORPHIZATION - CALL-SITE SPECIALIZATION (v0.4.0)

**ENABLED BY DEFAULT**: Polymorphic lambdas are automatically specialized at call sites with concrete types (M-POLY-A).

**What it does:**
- Specializes polymorphic functions (type `α -> α`) when called with concrete types
- Eliminates runtime type resolution overhead
- Foundation for future cross-module optimization (v0.5.0)

**The Pipeline:**
```go
// Phase 3.5: Monomorphization (runs after type checking, before lowering)
// File pipeline:   internal/pipeline/pipeline.go:228-265
// Module pipeline: internal/pipeline/pipeline.go:680-723

specializer := NewSpecializer(&typeChecker.CoreTI)
specializedProg, err := specializer.Specialize(coreProg)
if err != nil {
    return result, fmt.Errorf("monomorphization error: %w", err)
}
coreProg = specializedProg
```

**Resource Limits (prevent runaway specialization):**
- **Per-function cap**: 16 specializations per function
- **Module-wide cap**: 512 specializations per module
- Both limits enforced automatically with clear diagnostics

**CLI Flags:**
```bash
# Normal compilation (monomorphization enabled)
ailang run --entry main --caps IO module.ail

# Debug mode (show specialization stats)
ailang run --entry main --caps IO --debug-compile module.ail
# Output: [DEBUG] Monomorphization: 5 specializations, 2 skipped (cache: 3 hits, 2 misses)

# Emergency escape hatch (disable monomorphization)
ailang run --entry main --caps IO --no-mono module.ail
```

**Debug Output Example:**
```
[DEBUG] Monomorphization (module mymodule): 5 specializations, 2 skipped (cache: 3 hits, 2 misses)
[DEBUG] Module mymodule per-function specializations:
[DEBUG]   map: 3
[DEBUG]   filter: 2
[DEBUG] Module mymodule skipped functions:
[DEBUG]   recursiveSum: Recursive function not specialized in v0.4.0
[DEBUG]   mutualGroup: Mutually recursive bindings not specialized in v0.4.0
```

**What Gets Specialized (v0.4.0):**
- ✅ **Inline lambda applications**: `(\x. \y. if x > y then x else y)(3.14)(2.71)` ← Works!
- ✅ **Var-bound lambdas with comparison operators**: `let max = \x. \y. if x > y then x else y in max(3.14)(2.71)` ← **Fixed in M-POLY-B Phase 1!**
- ✅ Concrete argument types that can be statically determined
- ✅ Non-recursive lambdas

**What Gets Skipped (v0.4.0):**
- ❌ **Var-bound lambdas with arithmetic operators**: `let add = \x. \y. x + y in add(3.14)(2.71)` ← Runtime panic!
  - **Why**: Type inference defaults arithmetic to `int` (Num typeclass defaulting)
  - **Workaround 1**: Type annotations: `let add: float -> float -> float = \x. \y. x + y`
  - **Workaround 2**: Inline lambdas: `(\x. \y. x + y)(3.14)(2.71)` (works!)
  - **Fix**: v0.4.2 (Phase 2) will fix type inference defaulting (~4-8 hours)
- ⏭️ Recursive functions (diagnostic message explains why)
- ⏭️ Mutually recursive groups (diagnostic message explains why)
- ⏭️ Functions hitting per-function cap (16 limit)
- ⏭️ Modules hitting module cap (512 limit)

**M-POLY-B Phase 1 Complete (v0.4.0):**
- ✅ Comparison operators work: `>`, `<`, `>=`, `<=`, `==`, `!=`
- ✅ Fixed 5 bugs: dictionary elaboration, type substitution, cloneExpr, substituteType, operator resolution
- ❌ Arithmetic operators broken: `+`, `-`, `*`, `/`, `%` (Phase 2 deferred to v0.4.2)
- See: [M-POLY-B-PHASE1-COMPLETION-REPORT.md](M-POLY-B-PHASE1-COMPLETION-REPORT.md)

**Key Discovery:**
Hindley-Milner type inference already specializes simple polymorphic lambdas during type checking. The monomorphization pass is designed for:
- More complex polymorphic patterns
- Cross-module polymorphism (v0.5.0)
- Persistently polymorphic values in let-polymorphism contexts

**Implementation:**
- Core: `internal/pipeline/specialize.go` (1002 LOC)
- Unit tests: `internal/pipeline/specialize_test.go` (461 LOC, 12/12 passing)
- Integration tests: `internal/pipeline/specialize_integration_test.go` (331 LOC, 7/7 passing)
- Pipeline integration: `internal/pipeline/pipeline.go` (+120 LOC)

**Performance:**
- O(n) traversal of Core AST
- Negligible overhead for non-polymorphic code
- Cache deduplication prevents redundant specializations

**Troubleshooting:**

If you encounter issues:
1. Check debug output: `ailang run --debug-compile your_file.ail`
2. Look for skip reasons in the output
3. Verify you're not hitting caps (16 per-function, 512 per-module)
4. As last resort, disable with `--no-mono` flag

**When asked to run evals, compare benchmarks, or update benchmark results:**

→ **ALWAYS use the [eval-orchestrator](.claude/agents/eval-orchestrator.md) agent**

The agent knows how to:
- Run benchmarks with cost-conscious defaults (cheap models for dev, --full for releases)
- Compare results, validate fixes, generate reports
- Update the benchmark dashboard (docs/BENCHMARK_COMPARISON.md)
- Use all available models and their pricing
- Route to appropriate `ailang eval-*` commands

**DO NOT:**
- ❌ Try to guess which make targets or scripts to use
- ❌ Write custom Python/bash scripts for benchmark analysis
- ❌ Manually regenerate dashboard files
- ❌ Call `ailang eval-*` commands directly (let the agent handle it)

---

## 🛠️ AVAILABLE SKILLS

**Skills are Anthropic Agent Skills (Oct 2025 spec) with progressive disclosure, executable scripts, and resource files.**

### Available Skills

- **use-ailang** - Write correct AILANG code with syntax reference and validation scripts
- **skill-builder** - Create new skills with automated scaffolding and validation (meta-skill!)
- **release-manager** - Create releases with automated pre-flight checks and verification
- **post-release** - Run eval baselines and update website dashboard automatically
- **sprint-planner** - Analyze design docs and create realistic, data-driven sprint plans
- **sprint-executor** - Execute sprints with TDD, continuous linting, and progress tracking

**Complete skill documentation**: See [.claude/skills/README.md](.claude/skills/README.md)

**Creating new skills**: Use the `skill-builder` skill or see [.claude/skills/SKILLS_GUIDE.md](.claude/skills/SKILLS_GUIDE.md)

### Using Skills

Skills are invoked automatically by Claude when appropriate for the task. Just describe what you want:

- "Create a skill for managing database migrations" → `skill-builder` skill
- "Ready to release v0.3.14" → `release-manager` skill
- "Update benchmarks" → `post-release` skill
- "Help me write AILANG code" → `use-ailang` skill
- "Plan the sprint" → `sprint-planner` skill

### Skills vs Agents vs Commands

- **Skills** (.claude/skills/): Focused workflows with progressive disclosure and automation
- **Agents** (.claude/agents/): Complex autonomous reasoning (eval-orchestrator, codebase-organizer)
- **Commands** (.claude/commands/): **Deprecated** - use skills instead

---

## Project Overview

**AILANG is a deterministic language designed for autonomous AI code synthesis and reasoning.**

Unlike human-oriented languages built around IDEs and concurrency, AILANG prioritizes:
- **Machine decidability** - Deterministic semantics for AI reasoning
- **Semantic transparency** - Every construct can be reflected and verified
- **Compositional determinism** - Predictable evaluation and type inference

File extension: `.ail`

## What AILANG Can Do (Implementation Status)

**✅ Core Language** (v0.3.14 - Stable):
- Pure functional programming (lambda calculus, closures, recursion)
- Hindley-Milner type inference with row polymorphism and TApp unification
- Built-in type class instances (Num, Eq, Ord, Show) - hardcoded, not user-extensible
- Algebraic effects with capability-based security (IO, FS, Clock, Net)
- Pattern matching with ADTs and exhaustiveness checking
- Module system with runtime execution and cross-module imports
- Interactive REPL with full type checking
- Block expressions `{ e1; e2; e3 }` for sequencing
- JSON support (parsing via `std/json.decode`, encoding via `std/json.encode`)

**✅ Development Tools** (Stable):
- M-EVAL: AI code generation benchmarks (multi-model support)
- M-EVAL-LOOP v2.0: Native Go eval tools with 90%+ test coverage
- Structured error reporting with JSON schemas
- Builtin registry with hermetic testing (`MockEffContext`)

**🔜 Deterministic Tooling** (v0.3.15 - Next):
- Canonical normalization (`ailang normalize file.ail`)
- Import suggestion (`ailang suggest-imports file.ail`)
- Deterministic code edits (`ailang apply plan.json file.ail`)
- Training data export (`--emit-trace jsonl` for AI self-training)

**🔮 Reflection & Meta-Programming** (v0.4.0+ - Future):
- Typed quasiquotes (deterministic AST templates) - lexer token exists
- Structural reflection (`reflect(typeOf(f))`) - replaces hardcoded type classes
- Schema registry (machine-readable type/effect definitions)
- Capability budgets (resource-bounded effects: `! {IO @limit=2}`)

**❌ Removed: Human-Oriented Features**
- CSP concurrency / session types → Replaced by static effect-typed task graphs
- LSP server → Replaced by deterministic JSON-RPC API (`ailangd`)
- IDE-centric DX (autocompletion, hover) → AIs use CLI/API
- User-defined type classes → Deferred to structural reflection (v0.4.0+)

**Quick Test:**
```bash
make test                # Run all tests
make verify-examples     # Check example files
ailang repl             # Start REPL
```

**For detailed version history, see [CHANGELOG.md](CHANGELOG.md)**

**🎉 MAJOR MILESTONE:** Module files now execute! Use `ailang run --caps IO,FS --entry main module.ail` to run module code with effects.

**⚠️ Important**: Flags must come BEFORE the filename when using `ailang run`.

## Key Design Principles
1. **Explicit Effects**: All side effects must be declared in function signatures
2. **Everything is an Expression**: No statements, only expressions that return values
3. **Type Safety**: Static typing with Hindley-Milner inference + row polymorphism
4. **Deterministic**: All non-determinism must be explicit (seeds, virtual time)
5. **AI-Friendly**: Generate structured execution traces for training

## Project Structure (v0.3.0+)
```
ailang/
├── cmd/ailang/         # CLI entry point ✅ COMPLETE
├── internal/
│   ├── ast/            # Surface AST ✅ COMPLETE
│   ├── lexer/          # Tokenizer ✅ COMPLETE
│   ├── parser/         # Parser ✅ COMPLETE
│   ├── core/           # Core AST (ANF) ✅ COMPLETE
│   ├── elaborate/      # Surface → Core elaboration ✅ COMPLETE
│   ├── types/          # Type system ✅ COMPLETE
│   ├── typeclass/      # Type classes ✅ COMPLETE (stub)
│   ├── link/           # Dictionary linking ✅ COMPLETE
│   ├── pipeline/       # Full compilation pipeline ✅ COMPLETE
│   ├── eval/           # Evaluator ✅ COMPLETE (Core + module support)
│   ├── repl/           # Interactive REPL ✅ COMPLETE
│   ├── runtime/        # Module execution runtime ✅ COMPLETE (v0.2.0)
│   ├── effects/        # Effect system runtime ✅ COMPLETE (v0.2.0)
│   ├── loader/         # Module loader ✅ COMPLETE
│   ├── errors/         # Error reporting ✅ COMPLETE
│   ├── schema/         # JSON schemas ✅ COMPLETE
│   ├── eval_harness/   # AI evaluation framework ✅ COMPLETE (M-EVAL)
│   ├── eval_analysis/  # Go eval tools ✅ COMPLETE (M-EVAL v2.0)
│   ├── eval_analyzer/  # Failure analyzer ✅ COMPLETE (M-EVAL v2.0)
│   ├── planning/       # Plan validation & scaffolding ✅ COMPLETE
│   ├── builtins/       # Builtin definitions ✅ COMPLETE
│   ├── dtree/          # Decision trees (pattern matching) ✅ COMPLETE
│   ├── iface/          # Interface definitions ✅ COMPLETE
│   ├── manifest/       # Module manifests ✅ COMPLETE
│   ├── module/         # Module system ✅ COMPLETE
│   ├── typedast/       # Typed AST ✅ COMPLETE
│   ├── channels/       # CSP implementation ❌ TODO (v0.4.0+)
│   └── session/        # Session types ❌ TODO (v0.4.0+)
├── stdlib/             # Standard library ✅ COMPLETE (std/io, std/fs, std/prelude)
├── tools/              # Development tools ✅ (eval, benchmarking, verification)
├── benchmarks/         # AI code generation benchmarks ✅
├── examples/           # Example .ail programs (66 files, 48 passing)
├── tests/              # Test suite ✅
└── docs/               # Documentation ✅ COMPLETE
```

## Development Workflow

### Building and Testing
```bash
make build          # Build the interpreter to bin/
make install        # Install ailang to system (makes it available everywhere)
make test           # Run all tests
make run FILE=...   # Run an AILANG file
make repl           # Start interactive REPL
```

### Adding Builtin Functions (✅ M-DX1 - COMPLETE!)

**AILANG has a modern builtin development system that reduces implementation time from 7.5h to 2.5h (-67%).**

**Status**: 🎉 **M-DX1 COMPLETE (Oct 2025)** - All 52 builtins migrated, organized, and fully documented!

#### Quick Start (2.5 hours instead of 7.5)

**Step 1: Register the builtin with metadata** (~30 min)
```go
// internal/builtins/string.go (or appropriate module file)
func init() {
    registerMyBuiltin()
}

func registerMyBuiltin() {
    RegisterEffectBuiltin(BuiltinSpec{
        Module:  "std/string",
        Name:    "_str_reverse",
        NumArgs: 1,
        IsPure:  true,        // or false with Effect: "IO"
        Type:    makeReverseType,
        Impl:    strReverseImpl,
        Metadata: &BuiltinMetadata{
            Description: "Reverse a string (Unicode-aware)",
            Params: []ParamDoc{
                {Name: "s", Description: "String to reverse"},
            },
            Returns: "Reversed string",
            Examples: []Example{
                {Code: `_str_reverse("hello")`, Description: "Returns \"olleh\""},
                {Code: `_str_reverse("🎉🎊")`, Description: "Returns \"🎊🎉\""},
            },
            Since:     "v0.3.15",
            Stability: StabilityStable,
            Tags:      []string{"string", "reverse", "unicode"},
            Category:  "string",
        },
    })
}

func makeReverseType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.String())
}

func strReverseImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    str := args[0].(*eval.StringValue).Value
    runes := []rune(str)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return &eval.StringValue{Value: string(runes)}, nil
}
```

**Step 2: Write hermetic tests** (~1 hour)
```go
// internal/builtins/register_test.go
func TestStrReverse(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        input    string
        expected string
    }{
        {"hello", "olleh"},
        {"", ""},
        {"🎉", "🎉"},
    }

    for _, tt := range tests {
        result, err := strReverseImpl(ctx, []eval.Value{
            testctx.MakeString(tt.input),
        })
        assert.NoError(t, err)
        assert.Equal(t, tt.expected, testctx.GetString(result))
    }
}
```

**Step 3: Validate and inspect** (~30 min)
```bash
# Validate the builtin (no feature flag needed since v0.3.10!)
ailang doctor builtins
# ✅ All builtins are valid!

# List all builtins
ailang builtins list --by-module
# # std/string (8)
#   _str_compare                   [pure]
#   _str_find                      [pure]
#   _str_len                       [pure]
#   _str_lower                     [pure]
#   _str_reverse                   [pure]
#   _str_slice                     [pure]
#   _str_trim                      [pure]
#   _str_upper                     [pure]

# Test in REPL (when M-DX1.6 is implemented)
ailang repl
> :type _str_reverse
string -> string
```

**Step 4: Wire to runtime** (~30 min)
- Already done! The registry automatically wires to runtime/link (no feature flag needed since v0.3.10)

#### Key Components

**Central Registry** (`internal/builtins/spec.go`):
- Single-point registration with `RegisterEffectBuiltin()`
- Compile-time validation (arity, types, impl, effects)
- ✅ No feature flag needed (default since v0.3.10)
- Freeze-safe (no registration after init)
- 49 builtins migrated (v0.3.10)

**Type Builder DSL** (`internal/types/builder.go`):
- Fluent API: `T.Func(args...).Returns(ret).Effects(effs...)`
- Reduces type construction from 35→10 lines (-71%)
- Methods: `String()`, `Int()`, `Bool()`, `List()`, `Record()`, `Func()`, `Returns()`, `Effects()`

**Test Harness** (`internal/effects/testctx/`):
- `MockEffContext` with HTTP/FS mocking
- Value constructors: `MakeString()`, `MakeInt()`, `MakeRecord()`, etc.
- Value extractors: `GetString()`, `GetInt()`, `GetRecord()`, etc.
- Hermetic testing (no real network/FS)

**Validation & Inspection**:
- `ailang doctor builtins` - Health checks with actionable diagnostics
- `ailang builtins list` - Browse registry (--by-effect, --by-module)
- 6 validation rules: type, impl, arity, effect consistency, module

#### Examples

**Pure function:**
```go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "std/string",
    Name:    "_str_len",
    NumArgs: 1,
    IsPure:  true,
    Type:    func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String()).Returns(T.Int())
    },
    Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        s := args[0].(*eval.StringValue).Value
        return &eval.IntValue{Value: len([]rune(s))}, nil
    },
})
```

**Effect function with HTTP:**
```go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "std/net",
    Name:    "_net_httpRequest",
    NumArgs: 4,
    Effect:  "Net",
    Type:    makeHTTPRequestType,
    Impl:    effects.NetHTTPRequest,  // Uses ctx.GetHTTPClient()
})
```

**Complex types with records:**
```go
func makeHTTPRequestType() types.Type {
    T := types.NewBuilder()

    headerType := T.Record(
        types.Field("name", T.String()),
        types.Field("value", T.String()),
    )

    responseType := T.Record(
        types.Field("status", T.Int()),
        types.Field("headers", T.List(headerType)),
        types.Field("body", T.String()),
    )

    return T.Func(
        T.String(),           // url
        T.String(),           // method
        T.List(headerType),   // headers
        T.String(),           // body
    ).Returns(
        T.App("Result", responseType, T.Con("NetError")),
    ).Effects("Net")
}
```

#### Testing Patterns

**Hermetic HTTP tests:**
```go
func TestNetHTTPRequest(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status": "ok"}`))
    }))
    defer server.Close()

    ctx := testctx.NewMockEffContext()
    ctx.GrantAll("Net")
    ctx.SetHTTPClient(server.Client())

    result, err := effects.NetHTTPRequest(ctx,
        testctx.MakeString(server.URL),
        testctx.MakeString("GET"),
        testctx.MakeList([]eval.Value{}),
        testctx.MakeString(""),
    )

    assert.NoError(t, err)
    resp := testctx.GetRecord(result)
    assert.Equal(t, 200, testctx.GetInt(resp["status"]))
}
```

#### Migration from Legacy Registry

**Before (legacy, 4 files, 35 lines of types):**
```go
// internal/eval/builtins.go
registry.Register("_str_len", func(args []Value) (Value, error) { ... })

// internal/link/builtin_module.go
iface.Decls["_str_len"] = &iface.FuncDecl{
    Type: &types.TFunc2{
        Params: []types.Type{&types.TCon{Name: "String"}},
        Return: &types.TCon{Name: "Int"},
        EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
    },
}

// internal/runtime/builtins.go
br.RegisterPure("_str_len", ...)

// internal/types/builtins.go
builtinTypes["_str_len"] = ...
```

**After (new registry, 1 file, 10 lines):**
```go
// internal/builtins/register.go
RegisterEffectBuiltin(BuiltinSpec{
    Module:  "std/string",
    Name:    "_str_len",
    NumArgs: 1,
    IsPure:  true,
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String()).Returns(T.Int())
    },
    Impl: strLenImpl,
})
```

#### Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Files to edit | 4 | 1 | -75% |
| Type construction LOC | 35 | 10 | -71% |
| Development time | 7.5h | 2.5h | -67% |
| Test setup LOC | ~50 | ~15 | -70% |

#### Status

**🎉 M-DX1 COMPLETE (October 2025) - 90% done!**

**Core Infrastructure (v0.3.9-alpha3 through v0.3.10):**
- ✅ M-DX1.1: Central Registry with validation
- ✅ M-DX1.2: Type Builder DSL
- ✅ M-DX1.3: Doctor + List CLI commands
- ✅ M-DX1.4: Test Harness with mocking
- ✅ M-DX1.5: Complete builtin migration (52 builtins)
- ✅ Removed feature flag - new registry is default
- ✅ 100% test coverage on new code

**Documentation & Organization (October 2025):**
- ✅ M-DX1.11: Enhanced metadata system with 11 fields
- ✅ M-DX1.12: File organization (split 785-line file into 7 modules)
- ✅ M-DX1.13: Migration safety validator (`ailang builtins check-migration`)
- ✅ M-DX1.14: **Complete documentation (52/52 builtins = 100%)** 🎉
  - All builtins have descriptions, params, returns, examples
  - Searchable tags, version tracking, stability indicators
  - Files: string.go (9), math.go (37), io.go (3), net.go (1), show.go (1), json_decode.go (1)

**Optional Polish (v0.3.15+):**
- ⏳ Enhanced CLI (`--verbose`, `search` command) (~2h)
- ⏳ REPL :type command (~0.5h)
- ⏳ Error diagnostics improvements (~0.5h)

**Verify builtin health:**
```bash
ailang doctor builtins              # Validate all 52 builtins
ailang builtins list                # List all builtins
ailang builtins list --by-module    # List by module
ailang builtins check-migration     # Check for orphaned builtins
```

**For full documentation, see:**
- Session summary: [M-DX1-FINAL-SUMMARY.md](M-DX1-FINAL-SUMMARY.md)
- Design rationale: `design_docs/planned/easier-ailang-dev.md`
- Test coverage: `internal/builtins/*_test.go`, `internal/effects/testctx/*_test.go`
- Changelog: See v0.3.10+ entries in `CHANGELOG.md`

### M-EVAL-LOOP: AI Evaluation & Self-Improvement (✅ COMPLETE - v2.0)

**When user asks about evaluations, benchmarks, or testing AI code generation:**

→ **Use the [eval-orchestrator](.claude/agents/eval-orchestrator.md) agent**

The agent handles all eval workflows:
- Running benchmarks (defaults to cheap/fast models)
- Comparing results and validating fixes
- Generating reports and interpreting metrics
- Routing to appropriate `ailang eval-*` commands

**For automated fix implementation:**

→ **Use the [eval-fix-implementer](.claude/agents/eval-fix-implementer.md) agent**

**Documentation** (for detailed reference):
- [Architecture Overview](docs/docs/guides/evaluation/architecture.md) - Commands & workflows
- [Evaluation README](docs/docs/guides/evaluation/README.md) - Quick start guide

**⚠️ CRITICAL: Running Multiple Models**

**The `ailang eval-suite` command OVERWRITES the output directory by default!**

```bash
# ❌ WRONG - Second run overwrites first run's results
ailang eval-suite --models gpt5
ailang eval-suite --models claude-sonnet-4-5  # DELETES gpt5 results!

# ✅ CORRECT - Run all models in ONE command
ailang eval-suite --models gpt5,claude-sonnet-4-5,gemini-2-5-pro

# ✅ ALSO CORRECT - Use different output directories
ailang eval-suite --models gpt5 --output eval_results/gpt5_only
ailang eval-suite --models claude-sonnet-4-5 --output eval_results/claude_only

# ✅ NEW (v0.3.14+) - Resume interrupted runs without losing progress
ailang eval-suite --full --skip-existing  # Skips benchmarks with existing results
```

**Resuming interrupted eval runs (v0.3.14+):**
- Use `--skip-existing` flag to skip benchmarks that already have result files
- Useful when eval baseline times out or is interrupted
- Checks for existing result files (pattern: `benchmarkID_lang_model_*.json`)
- Example: If 219/264 runs completed before timeout, `--skip-existing` runs only the missing 45
- Added in v0.3.14 to handle long-running baselines on slower machines

**Default model sets:**
- `ailang eval-suite` → Reads from `dev_models` in models.yml (currently: gpt5-mini, claude-haiku-4-5, gemini-2-5-flash)
- `ailang eval-suite --full` → Reads from `extended_suite` in models.yml (all 6 models: gpt5, gpt5-mini, claude-sonnet-4-5, claude-haiku-4-5, gemini-2-5-pro, gemini-2-5-flash)

**Quick reference - Common eval commands:**
```bash
# Update benchmark dashboard (PRESERVES HISTORY!)
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json
# ✅ Automatically writes to docs/static/benchmarks/latest.json
# ✅ Preserves all historical versions
# ✅ Validates before writing
# ✅ Atomic writes (no corruption)

# Generate markdown dashboard
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=markdown > docs/BENCHMARK_COMPARISON.md

# Run baseline (REQUIRES explicit version!)
make eval-baseline EVAL_VERSION=v0.3.10              # Uses dev models (3 cheap models)
make eval-baseline EVAL_VERSION=v0.3.10 FULL=true    # Uses ALL 6 models (extended_suite)

# Compare two baselines
ailang eval-compare eval_results/baselines/v0.3.8 eval_results/baselines/v0.3.9

# Generate performance matrix
ailang eval-matrix eval_results/baselines/v0.3.9 v0.3.9
```

**⚠️ CRITICAL - Dashboard Update Workflow (v0.3.10+)**

**The dashboard JSON now preserves history automatically!**

```bash
# ✅ CORRECT - Safe, preserves history
ailang eval-report eval_results/baselines/v0.3.10 v0.3.10 --format=json
# Reads existing latest.json → merges history → validates → writes atomically

# ❌ WRONG - Don't redirect stdout (bypasses history preservation)
ailang eval-report ... --format=json > docs/static/benchmarks/latest.json
```

**How it works:**
1. Loads existing `docs/static/benchmarks/latest.json`
2. Builds new entry from current results
3. Merges into history (updates if version exists, appends if new)
4. Validates JSON structure (no duplicates, required fields present)
5. Writes atomically (temp file + rename, no partial writes)
6. Also prints to stdout (for backwards compatibility)

**DO NOT**:
- ❌ Create new bash scripts for evals - agents use existing `ailang eval-*` commands
- ❌ Duplicate agent logic - just invoke the appropriate agent
- ❌ Write custom analysis tools - extend `internal/eval_analysis/` if needed
- ❌ Run multiple `ailang eval-suite` commands to same directory - results will be overwritten!
- ❌ Search for dashboard generation scripts - just use `ailang eval-report`
- ❌ Redirect `--format=json` to file (bypasses history preservation logic!)
- ❌ Manually edit latest.json (use eval-report to update it)

### Code Quality & Coverage
```bash
make test-coverage-badge  # Quick coverage check (shows: "Coverage: 29.9%")
make test-coverage        # Run tests with coverage, generates HTML report
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make fmt-check            # Check if code is formatted
make vet                  # Run go vet
make ci                   # Run full CI verification locally
make ci-strict            # Extended CI with A2 milestone gates (pre-release)
```

### Example Management
```bash
make verify-examples      # Verify all example files work/fail
make update-readme        # Update README with example status
make flag-broken          # Add warning headers to broken examples
make test-parity          # Test REPL/file parity (manual only, requires interactive REPL)
```

### Development Helpers
```bash
make deps                 # Install all dependencies
make clean                # Remove build artifacts and coverage files
make help                 # Show all available make targets
```

### Debug Flags

**AILANG provides environment variables for verbose debugging and strict error checking:**

#### `DEBUG_STRICT=1` - Catch Silent Failures Early

**What it does**: Makes incomplete switch statements and unhandled cases **fail loudly** with panic instead of silently returning unchanged values.

**When to use**:
- During development of new compiler passes
- When debugging AST traversal code
- To catch missing cases in switch statements
- In CI to enforce completeness

**Example**:
```bash
# Normal mode - unhandled cases return unchanged (silent failure)
$ ailang run test.ail
# ✓ May complete successfully even with bugs!

# Strict mode - unhandled cases panic immediately
$ DEBUG_STRICT=1 ailang run test.ail
panic: cloneExpr: unhandled node type *core.Record (NodeID 42).
    Add a case for this type or explicitly mark as unsupported.
# ✓ Bug caught immediately!
```

**Affected functions** (as of v0.4.1):
- `internal/pipeline/specialize.go`:
  - `cloneExpr()` - Cloning during monomorphization
  - `specializeExpr()` - Specializing expressions

**See Also**: [M-DX8: Silent Failure Prevention](design_docs/planned/v0_4_1/m-dx8-silent-failure-prevention.md)

#### `DEBUG_MONO_VERBOSE=1` - Monomorphization Tracing

**What it does**: Logs detailed information about monomorphization (polymorphic function specialization).

**When to use**:
- Debugging type substitution issues
- Understanding which functions are specialized
- Tracking down operator re-linking problems

**Example**:
```bash
$ DEBUG_MONO_VERBOSE=1 ailang run --entry main --debug-compile test.ail
[DEBUG_MONO_VERBOSE] Found lambda, type=α2 -> α2 -> α2, isPoly=true
[DEBUG_MONO_VERBOSE] lambda type from CoreTI: α2 -> (α2 -> α2)
[DEBUG_MONO_VERBOSE] extracted paramTVars: [α2]
[DEBUG_MONO_VERBOSE] typeSubst built: map[α2:float]
[DEBUG_MONO_VERBOSE] Cloning DictApp: method=gt
[DEBUG_MONO_VERBOSE]   Original DictRef: class=Ord, type=Int, NodeID=15
[DEBUG_MONO_VERBOSE]   Cloned DictRef: class=Ord, type=float, NodeID=42
```

#### `DEBUG_OPERATOR_LOWERING=1` - Operator Resolution Tracing

**What it does**: Logs operator lowering decisions (BinOp/DictApp → Intrinsic).

**When to use**:
- Debugging operator dispatch issues
- Understanding which builtin is selected
- Tracking type-guided operator selection

#### Combining Debug Flags

**Recommended combinations**:

```bash
# Development mode - catch bugs early + verbose output
$ DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang run test.ail

# CI mode - strict checking only (no verbose output)
$ DEBUG_STRICT=1 make test

# Deep debugging - all flags
$ DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 DEBUG_OPERATOR_LOWERING=1 ailang run --debug-compile test.ail
```

#### Keeping `ailang` Up to Date

**After making code changes to the ailang binary:**
```bash
make quick-install  # Fast reinstall (recommended for development)
# OR
make install        # Full reinstall with version info
```

**Important**: The `ailang` command in your PATH points to `/Users/mark/go/bin/ailang` (system install), NOT `bin/ailang` (local build). Always run `make install` or `make quick-install` after building to update the system binary. Otherwise your changes won't be used when running `ailang` commands.

**For local testing without install:**
```bash
./bin/ailang <command>  # Use local build directly
```

### IMPORTANT: Keeping Documentation Updated

**Required documentation updates for every change:**

#### 1. README.md
- Update implementation status when adding new features
- Update current capabilities when functionality changes
- Update examples when they're fixed or new ones added
- Keep line counts and completion status accurate
- Document new builtin functions and operators
- Update the roadmap as items are completed

#### 2. CHANGELOG.md
**Must be updated for every feature or bug fix:**
- Follow semantic versioning (vMAJOR.MINOR.PATCH)
- Group changes by category: Added, Changed, Fixed, Deprecated, Removed
- Include code locations for new features (e.g., `internal/schema/`)
- Note breaking changes clearly
- Add migration notes if needed
- Include metrics (lines of code, test coverage)

Example entry:
```markdown
## [v3.2.0] - 2024-09-28

### Added
- Schema Registry (`internal/schema/`) - Versioned JSON schemas
- Error JSON Encoder (`internal/errors/`) - Structured error reporting
- Test coverage: 100% for new packages
- Total new code: ~1,500 lines
```

#### 3. Design Documentation
- **Before starting**: Create design doc in `design_docs/planned/`
- **After completing**: Move to `design_docs/implemented/vX_Y/`
- Include implementation report with metrics and limitations

**CRITICAL: Example Files Required**
**Every new language feature MUST have a corresponding example file:**
- Create `examples/feature_name.ail` for each new feature
- Include comprehensive examples showing all capabilities
- Add comments explaining the behavior and expected output
- ⚠️ **Test that examples actually work with current implementation**
- ⚠️ **Add warning headers to examples that don't work**
- These examples will be used in documentation and tutorials
- Always test examples before documenting them as working

### Writing AILANG Code

**When writing AILANG code during development:**
Refer to the **AI Teaching Prompt** for comprehensive syntax guidance:
- **Current version**: [prompts/v0.3.8.md](prompts/v0.3.8.md)
- Validated through multi-model testing (Claude, GPT, Gemini)
- Covers syntax, limitations, common pitfalls, and working examples

**Quick reference:**
```bash
ailang run --caps IO,FS --entry main module.ail  # Run module
ailang repl                                        # Start REPL
:type expr                                         # Check type in REPL
```

**For detailed syntax, limitations, and examples:**
- See [prompts/v0.3.8.md](prompts/v0.3.8.md) - Complete AILANG teaching prompt
- See [docs/LIMITATIONS.md](docs/LIMITATIONS.md) - Known limitations and workarounds
- See [examples/](examples/) - 66 example files (48 working)

### Common Tasks

#### Adding a New Language Feature
1. Update token definitions in `internal/lexer/token.go`
2. Modify lexer in `internal/lexer/lexer.go` to recognize tokens
3. Add AST nodes in `internal/ast/ast.go`
4. Update parser in `internal/parser/parser.go`
5. Add type rules in `internal/types/`
6. Implement evaluation in `internal/eval/`
7. Write tests in corresponding `*_test.go` files
8. **Add examples in `examples/`** (REQUIRED!)
9. **Update CHANGELOG.md** (REQUIRED!)
10. **Update README.md** if public-facing (REQUIRED!)

**For detailed contributing guidelines:**
- See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) - Full development guide
- See [design_docs/](design_docs/) - Architecture and design decisions

## 🎯 RELEASE WORKFLOW - READ THIS FIRST!

**When user says "ready to release" or "update dashboard":**

### Quick Reference

**Run release with the release-manager skill** (handles everything):

When ready to release, invoke the `release-manager` skill with the version number.

The release-manager skill handles:
- Pre-release verification (tests, linting, file sizes)
- Version updates in documentation
- Git tagging and pushing
- CI/CD monitoring
- Release verification

After release completes, use the `post-release` skill to:
- Run baseline evaluations
- **Update website dashboard** ← Critical!
- Update design docs and public documentation

### Manual Dashboard Update (if needed)

**Update website dashboard for specific version**:
```bash
# Generate dashboard files (markdown + JSON with history)
# Note: 2>/dev/null suppresses progress messages that would appear in the markdown
ailang eval-report eval_results/baselines/v0.3.12 v0.3.12 --format=docusaurus 2>/dev/null > docs/docs/benchmarks/performance.md
# DO NOT redirect JSON to file - it writes to docs/static/benchmarks/latest.json automatically with history preservation
ailang eval-report eval_results/baselines/v0.3.12 v0.3.12 --format=json

# Verify JSON is valid
jq -r '.version, .aggregates.finalSuccess' docs/static/benchmarks/latest.json

# Clear Docusaurus cache (prevents webpack errors)
cd docs && npm run clear

# Test locally (optional)
cd docs && npm start
# Visit: http://localhost:3000/ailang/docs/benchmarks/performance

# Commit and push
git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
git commit -m "Update benchmark dashboard for v0.3.12"
git push
```

### Common Issues

**Problem**: Dashboard shows old version (e.g., v0.3.9 instead of v0.3.12)
**Solution**: Use `ailang eval-report` with specific baseline directory

**Problem**: "Uncaught runtime errors" / webpack chunk errors
**Cause**: Docusaurus build cache stale
**Solution**: `cd docs && npm run clear && rm -rf docs/.docusaurus docs/build && npm start`

**Problem**: Dashboard JSON shows "null" for aggregates
**Cause**: Used wrong JSON file (performance matrix vs dashboard JSON)
**Solution**: Use `ailang eval-report` output, not files from `eval_results/performance_tables/`

---

## 📐 Code Organization Principles (AI-First Design)

### File Size Guidelines

**AILANG is designed to be maintained by AI assistants. Keep files small and focused.**

**Target file sizes:**
- **Sweet spot**: 200-500 lines per file
- **Acceptable**: 500-800 lines
- **Problematic**: 800-1200 lines (consider splitting)
- **Critical**: 1200+ lines (MUST split before adding features)

**Why small files matter for AI:**
- Fits in AI context window (I can see the whole file at once)
- Single responsibility principle naturally enforced
- Easy to understand the full structure in one read
- Reduces merge conflicts
- Enables better testing isolation

**Check file sizes:**
```bash
make check-file-sizes    # Fails CI if any file >800 lines
make report-file-sizes   # Shows all files >500 lines
wc -l internal/path/file.go  # Check specific file
```

### Current Technical Debt

**Check current status:**
```bash
make report-file-sizes    # Detailed report of files >500 lines
make codebase-health      # Overall codebase metrics
make largest-files        # Top 20 largest files
```

As of October 2025, ~10 files exceed the 800 line limit (out of 183 total). Run `make report-file-sizes` for the current list.

**Before modifying these files:**
1. Check if splitting is needed first
2. Run tests before/after: `make test`
3. Use the `codebase-organizer` agent for safe refactoring

### File Organization Patterns

#### Pattern 1: One Concept Per File

```
❌ BAD: Everything in one file
internal/parser/parser.go (2518 lines)
  - Expression parsing
  - Statement parsing
  - Type parsing
  - Pattern parsing
  - Module parsing

✅ GOOD: Split by responsibility
internal/parser/
  ├── parser.go (200 lines)         # Main struct, entry points, package docs
  ├── expressions.go (300 lines)    # parseExpression, parseLambda, parseCall
  ├── statements.go (250 lines)     # parseLetDecl, parseFuncDecl, parseType
  ├── types.go (200 lines)          # parseType, parseEffects, parseTypeParams
  ├── patterns.go (280 lines)       # parsePattern, parseConstructor
  ├── modules.go (150 lines)        # parseModule, parseImport, parseExport
  └── helpers.go (140 lines)        # parseParams, parseBlock, utility functions
```

#### Pattern 2: Main File as Table of Contents

Every package should have a main file (usually `pkg.go` or matching package name) that serves as navigation:

```go
// internal/parser/parser.go (200 lines max)
package parser

// Package parser implements AILANG source code parsing.
//
// # Architecture
//
// The parser is split into several files by responsibility:
//   - parser.go: Main Parser struct and entry points (THIS FILE)
//   - expressions.go: Expression parsing (literals, lambdas, calls, etc.)
//   - statements.go: Top-level declarations (func, type, let)
//   - types.go: Type annotation parsing
//   - patterns.go: Pattern matching syntax
//   - modules.go: Module system (import/export)
//
// # Usage
//
//   p := parser.New(lexer)
//   file, err := p.Parse()
//
// # See Also
//
//   - internal/ast: AST node definitions
//   - internal/lexer: Token generation
//   - docs/parser/README.md: Detailed parser documentation

// Parser is the main entry point for parsing AILANG source code.
type Parser struct { /* ... */ }

// Parse parses a complete AILANG source file.
// Implementation delegates to parseFile() in statements.go.
func (p *Parser) Parse() (*ast.File, error) { /* ... */ }
```

#### Pattern 3: Tests Next to Implementation

```
✅ GOOD: Focused test files
internal/parser/
  ├── expressions.go
  ├── expressions_test.go (300 lines focused tests)
  ├── statements.go
  ├── statements_test.go (250 lines focused tests)
  └── integration_test.go (end-to-end tests)

❌ BAD: One giant test file
  └── parser_test.go (5000 lines)
```

#### Pattern 4: Clear File Naming

File names should match the main functions they contain:

```
✅ GOOD:
expressions.go → parseExpression(), parseCall(), parseLambda()
statements.go  → parseLetDecl(), parseFuncDecl(), parseTypeDecl()
patterns.go    → parsePattern(), parseConstructor()

❌ BAD:
parse_stuff.go → everything mixed together
utils.go       → vague, no clear responsibility
```

### Adding New Features (File Size Rules)

**Before adding any new feature to a file:**

```bash
# 1. Check current file size
wc -l internal/types/typechecker_core.go
# Output: 2736 lines

# 2. If >800 lines, STOP and split first
# 3. If 500-800 lines, consider if new feature pushes it over 800
# 4. If <500 lines, proceed normally

# 5. After changes, verify size
wc -l internal/types/typechecker_core.go
make check-file-sizes  # Fails if >800 lines
```

**Splitting workflow:**

```bash
# Option 1: Use the codebase-organizer agent (recommended)
# This agent safely refactors files while ensuring tests pass

# Option 2: Manual split (if you understand the code deeply)
make test                    # Baseline - all tests pass
# ... split files ...
make test                    # Verify - all tests still pass
git add internal/types/*.go
git commit -m "Split typechecker_core.go into 8 files (AI-friendly)"
```

### Package Documentation Standards

Every package with >3 files MUST have a README.md:

```markdown
# internal/parser

Parser for AILANG source code.

## Files

- `parser.go` - Main Parser struct, entry points
- `expressions.go` - Expression parsing: literals, lambdas, calls, operators
- `statements.go` - Declarations: func, type, let, import, export
- `types.go` - Type annotations: simple types, effects, type parameters
- `patterns.go` - Pattern matching: constructors, literals, wildcards, guards
- `modules.go` - Module system: module declarations, import resolution
- `helpers.go` - Shared utilities: parameter parsing, block parsing

## Entry Points

- `Parse()` → `parseFile()` in statements.go
- `parseExpression()` in expressions.go
- `parseType()` in types.go
- `parsePattern()` in patterns.go

## Cross-references

- Consumes: `internal/lexer` (tokens)
- Produces: `internal/ast` (syntax tree)
- Used by: `internal/pipeline`, `internal/repl`
```

### Automated Code Organization

**Use the codebase-organizer agent** for safe refactoring:

The `codebase-organizer` agent is available in `.claude/agents/codebase-organizer.md`. It:
- Monitors file sizes across the codebase
- Identifies files that need splitting
- Safely refactors large files into smaller, focused modules
- Ensures all tests pass before/after refactoring
- Maintains git history and commit hygiene

**Example usage:**
```bash
# Ask Claude to invoke the agent:
"Please use the codebase-organizer agent to check for files that need splitting"

# Or for specific refactoring:
"Use the codebase-organizer agent to split internal/parser/parser.go"
```

### Measuring Success

```bash
# CI checks (automatically run on PRs)
make check-file-sizes     # Fails if any file >800 lines

# Status reports
make report-file-sizes    # Lists all files >500 lines
make codebase-health      # Full codebase metrics
```

**Goal metrics:**
- 0 files over 800 lines ✅
- <5 files between 500-800 lines ⚠️
- Average file size: 300-400 lines 🎯

---

## 🚨 CRITICAL WARNINGS

### Testing Policy
**ALWAYS remove out-of-date tests. No backward compatibility.**
- When architecture changes, delete old tests completely
- Don't maintain legacy test suites
- Write new tests for new implementations
- Keep test suite clean and current

### Linting & "Unused" Code Warnings

**⚠️ LESSON LEARNED: Never blindly delete "unused" functions without understanding WHY they're unused!**

**The Import System Disaster (September 2025)**
In commit `eae08b6`, working import functions were deleted because linter said they were "unused".

**What actually happened:**
1. Function **calls** were renamed from `parseModuleDecl()` to `_parseModuleDecl()` (note underscore)
2. Function **definitions** kept original names (no underscore)
3. Calls were then **commented out**
4. Linter correctly said "hey, `parseModuleDecl` is never called!"
5. Functions were **blindly deleted**
6. Result: **Working import system completely broken** 💥

**Rules to Prevent This:**

1. **NEVER delete functions just because linter says "unused"**
   - First understand WHY they're unused
   - Check git history - were they just commented out?
   - Search entire codebase for references (including comments)
   - Run `make test-imports` and `make test` BEFORE deleting anything

2. **If renaming function calls, rename definitions too**
   - Use IDE refactoring tools, not manual find/replace
   - If adding `_` prefix to mark as TODO, add to BOTH call and definition
   - Better: use TODO comments instead of renaming

3. **Test between each change**
   - Don't combine: rename + comment out + delete
   - Run tests after EACH step:
     - After rename → `make test`
     - After commenting out → `make test-imports`
     - After deleting → `make test && make lint`

4. **When linter complains about unused code:**
   ```bash
   # Step 1: Check if it's really unused
   git log -p --all -S 'functionName' internal/
   grep -r "functionName" internal/

   # Step 2: Check recent changes
   git log --oneline internal/parser/parser.go | head -5
   git diff HEAD~1 internal/parser/parser.go | grep functionName

   # Step 3: If truly unused AND you know why, document it
   git commit -m "Remove unused parseOldFormat() - replaced by parseNewFormat() in commit abc123"
   ```

5. **Special warning for parser/module/import code**
   - These are **critical** for language functionality
   - If you break these, **nothing imports work**
   - Always run `make test-imports` before committing parser changes
   - Check that example files still work: `make verify-examples`

**Recovery Checklist (if this happens again):**
1. Find last working commit: `git log --all --oneline | grep "import"`
2. Check what was deleted: `git diff working_commit broken_commit`
3. Restore deleted functions: `git show working_commit:file.go`
4. Test imports: `make test-imports`
5. Document in commit message what was broken and how it was fixed

### Lexer/Parser Architecture - NEWLINE Tokens Don't Exist!

**⚠️ CRITICAL LESSON: The lexer NEVER generates NEWLINE tokens - it skips them as whitespace!**

**The Multi-line ADT Parser Bug (October 2025)**
While implementing multi-line ADT syntax support, code was written assuming the parser could see NEWLINE tokens:
```go
// ❌ WRONG - This code will never work!
p.skipNewlinesAndComments()  // Tries to skip NEWLINE tokens
if p.curTokenIs(lexer.NEWLINE) {  // This condition is NEVER true!
    ...
}
```

**The Root Cause:**
In `internal/lexer/lexer.go`, the `NextToken()` function calls `skipWhitespace()` which does this:
```go
func (l *Lexer) skipWhitespace() {
    for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
        l.readChar()
    }
}
```

This means `\n` characters are **consumed and never returned as tokens**. Even though `lexer/token.go` defines a NEWLINE token type, the lexer never generates them!

**Implications for Parser Development:**

1. **Never check for NEWLINE tokens**
   ```go
   // ❌ WRONG - lexer skips newlines
   if p.curTokenIs(lexer.NEWLINE) { ... }
   if p.peekTokenIs(lexer.NEWLINE) { ... }

   // ✅ CORRECT - rely on lexer skipping whitespace
   // After RPAREN of Leaf(int), next token is PIPE or TYPE (not NEWLINE)
   if p.curTokenIs(lexer.PIPE) { ... }
   ```

2. **Multi-line syntax "just works"**
   - The lexer automatically handles multi-line constructs
   - For ADTs, after `Leaf(int)` on line 4, the next token is `|` on line 5
   - No need to explicitly skip newlines - the lexer already did it

3. **Token stream example:**
   ```ailang
   type Tree =
     | Leaf(int)
     | Node(Tree, int, Tree)
   ```

   **Token stream the parser sees:**
   ```
   TYPE Tree ASSIGN PIPE Leaf LPAREN int RPAREN PIPE Node LPAREN Tree COMMA int COMMA Tree RPAREN ...
   ```

   **NOT**:
   ```
   TYPE Tree ASSIGN NEWLINE PIPE Leaf LPAREN int RPAREN NEWLINE PIPE Node ...
   ```

4. **When you think you need newline handling:**
   - You probably don't! The lexer handles it
   - Focus on the semantic tokens (PIPE, TYPE, IDENT, etc.)
   - Trust that whitespace (including newlines) is already skipped

5. **If you genuinely need layout-sensitive parsing:**
   - Would require modifying the lexer's `skipWhitespace()` function
   - Would affect the ENTIRE language parsing
   - This is a breaking architectural change - avoid if possible
   - Consider alternative approaches (explicit delimiters, etc.)

**Debugging tip:** If you see unexpected tokens or "skipped too far" issues, check if:
1. You're assuming NEWLINE tokens exist (they don't!)
2. You're calling `skipNewlinesAndComments()` (usually unnecessary)
3. The lexer is already doing what you want (it skips whitespace automatically)

**Files to check if modifying lexer/parser interaction:**
- `internal/lexer/lexer.go` - `NextToken()` and `skipWhitespace()`
- `internal/parser/parser.go` - `nextToken()` wrapper
- Any parser code that checks for or handles whitespace

---

## Parser Developer Experience Guide

**⚠️ READ THIS BEFORE WRITING PARSER CODE - Saves 30% of development time!**

This section documents critical parser conventions that prevent token position bugs (the #1 time sink in parser development).

### Parser Token Position Convention

**CRITICAL:** AILANG parser functions follow this convention:
- **Input:** Parser is AT the first token to parse
- **Output:** Parser is AT the last token of what was parsed (NOT after it)

**Example:**
```go
// To parse "42" followed by a comma:
p.nextToken() // move to 42
expr := p.parseExpression(LOWEST)  // parses "42", leaves cur=42 (NOT comma!)
p.nextToken() // NOW we're at comma ✓
```

**Functions following this convention:**
- `parseExpression()` - Leaves parser AT the last token of the expression
- `parseType()` - Leaves parser AT the last token of the type
- `parsePattern()` - Leaves parser AT the last token of the pattern
- Most parser functions follow this pattern

**When writing new parser functions:**
- ✅ ALWAYS call `p.nextToken()` AFTER calling these functions
- ❌ DON'T call `p.nextToken()` BEFORE - the caller handles positioning
- ✅ Document your function if it deviates from this convention

**Debugging token positions:**
```go
// Use DEBUG_PARSER=1 to see token flow:
// $ DEBUG_PARSER=1 ailang run test.ail
// [ENTER parseExpression] cur=INT peek=COMMA
// [EXIT parseExpression] cur=INT peek=COMMA  ← Still AT the INT!
```

### Common AST Types Reference

**Quick type lookup:**
```bash
grep "^type.*struct" internal/ast/ast.go | head -20
```

**Expression types:**
- **Literals**: `ast.Literal` with `Kind` field (IntLit, StringLit, BoolLit, FloatLit, UnitLit)
  - ⚠️ **GOTCHA**: Lexer returns `int64`, not `int` for IntLit
  - ✅ Access: `lit.Value.(int64)`
  - ❌ Wrong: `lit.Value.(int)` (will panic!)
- **Lists**: `ast.List` with `Elements []Expr`
- **Variables**: `ast.Variable` with `Name string`
- **Function calls**: `ast.FuncCall` with `Func Expr, Args []Expr`
- **Lambdas**: `ast.Lambda` with `Params []*ast.Param, Body Expr`
- **Blocks**: `ast.Block` with `Exprs []Expr`

**Type types:**
- **Simple types**: `ast.SimpleType` with `Name string` (e.g., "int", "bool", "string")
- **List types**: `ast.ListType` with `Element Type`
- **Function types**: `ast.FuncType` with `Params []Type, Return Type, Effects *ast.EffectRow`
- **Type applications**: `ast.TypeApp` with `Con string, Args []Type`

**Pattern types:**
- **Variable pattern**: `ast.VarPattern` with `Name string`
- **Constructor pattern**: `ast.ConstructorPattern` with `Name string, Args []Pattern`
- **Literal pattern**: `ast.LiteralPattern` with `Value Literal`
- **Wildcard pattern**: `ast.WildcardPattern` (matches anything)

### Quick Token Lookup

**Check if a keyword exists:**
```bash
grep -i "forall" internal/lexer/token.go
# Output: FORALL token exists!
```

**Common testing keywords (already in lexer):**
- `FORALL`, `EXISTS` - Quantifiers
- `TEST`, `TESTS` - Test blocks
- `PROPERTY`, `PROPERTIES` - Property-based tests
- `ASSERT` - Assertions

**If you see an identifier instead of a keyword:**
- Lexer returns token type, not `lexer.IDENT` for keywords
- ✅ Use `lexer.FORALL`, not `lexer.IDENT + literal check`
- ❌ Wrong: `p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "forall"`
- ✅ Right: `p.curTokenIs(lexer.FORALL)`

### Parsing Optional Sections After Optional Sections

**Pattern for parsing `properties` that can appear after optional `tests`:**

```go
// properties can be in PEEK (no tests) or CUR (after tests)
if p.peekTokenIs(lexer.PROPERTIES) || p.curTokenIs(lexer.PROPERTIES) {
    // If in peek, advance to it
    if p.peekTokenIs(lexer.PROPERTIES) {
        p.nextToken()
    }
    // Now always at PROPERTIES
    properties := p.parsePropertiesBlock()
}
```

**Why:** Previous optional section may or may not advance the parser, so check both positions.

### Test Error Printing Pattern

**❌ WRONG - Errors are hidden:**
```go
if len(p.Errors()) != 0 {
    t.Fatalf("parser had %d errors:", len(p.Errors()))
    // ⚠️ This never executes! t.Fatalf stops immediately
    for _, err := range p.Errors() {
        t.Errorf("  %s", err)
    }
}
```

**✅ CORRECT - Errors are visible:**
```go
if len(p.Errors()) != 0 {
    // Print errors BEFORE Fatalf
    for _, err := range p.Errors() {
        t.Errorf("  %s", err)
    }
    t.Fatalf("parser had %d errors", len(p.Errors()))
}
```

### Debug Mode (Coming in v0.3.15)

**Enable token position tracing:**
```bash
DEBUG_PARSER=1 ailang run test.ail
```

**Output example:**
```
[ENTER parseTestsBlock] cur=LBRACE peek=TEST
[ENTER parseTestCase] cur=TEST peek=LPAREN
[EXIT parseTestCase] cur=RPAREN peek=COMMA
[EXIT parseTestsBlock] cur=RBRACE peek=PROPERTIES
```

**See also:** M-DX9 Parser Developer Experience (design_docs/planned/v0_3_15/m-dx9-parser-developer-experience.md)

---

## Reference Documentation

**For detailed guides, see:**
- **AILANG Syntax**: [prompts/v0.3.8.md](prompts/v0.3.8.md) - Complete teaching prompt
- **REPL Guide**: [docs/guides/repl.md](docs/guides/repl.md) - Interactive development
- **Limitations**: [docs/LIMITATIONS.md](docs/LIMITATIONS.md) - Known issues and workarounds
- **Contributing**: [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) - Development workflow
- **Design Docs**: [design_docs/](design_docs/) - Architecture decisions
- **Examples**: [examples/](examples/) - 66 example programs

**For architecture details, see:**
- [design_docs/20250926/initial_design.md](design_docs/20250926/initial_design.md) - Original design
- [design_docs/implemented/](design_docs/implemented/) - Completed features
- [design_docs/planned/](design_docs/planned/) - Future work

## Important Notes
1. The language is expression-based - everything returns a value
2. Effects are tracked in the type system - never ignore them
3. Pattern matching must be exhaustive
4. All imports must be explicit
5. Row polymorphism allows extensible records and effects
6. Session types ensure protocol correctness in channels (when implemented)

## Quick Debugging Checklist
- [ ] Check lexer is producing correct tokens
- [ ] Verify parser is building proper AST
- [ ] Ensure all keywords are in the keywords map
- [ ] Confirm precedence levels are correct
- [ ] Check that all AST nodes implement correct interfaces
- [ ] Verify type substitution is working correctly

---

**Remember**: This is a living document. Update it when workflows change, but keep it focused on **actionable instructions** for Claude, not reference material that belongs in docs/.
