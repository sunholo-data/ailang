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

### 0.1. VERIFY GITHUB ACCOUNT BEFORE RELEASES/TAGS

**⚠️ CRITICAL: Multi-account authentication issues (November 2025)**

The developer uses multiple GitHub accounts (personal and work projects). The `gh` CLI may have the wrong account active, causing release/tag operations to fail.

**The GitHub Account Mismatch (November 2025):**
```bash
# ❌ WRONG - Active account is for wrong project
gh auth status
# Shows: Active account: rw-markedmondson (Rockwool project)
# But this repo needs: MarkEdmondson1234

gh release create v0.4.4  # FAILS with auth error
git push origin v0.4.4     # FAILS with auth error
```

**✅ CORRECT approach before ANY release or git push operations:**

1. **ALWAYS check active GitHub account:**
   ```bash
   gh auth status
   ```

2. **Verify the active account matches the repo owner:**
   - This repo (sunholo-data/ailang) needs: `MarkEdmondson1234`
   - If active account is `rw-markedmondson` → WRONG ACCOUNT

3. **Switch to correct account if needed:**
   ```bash
   gh auth switch --user MarkEdmondson1234
   ```

4. **Then proceed with release operations:**
   ```bash
   git push origin v0.4.4              # Push tag
   gh release create v0.4.4 ...        # Create release
   ```

**Checklist for releases:**
- [ ] Check `gh auth status` - verify active account
- [ ] Switch account if needed: `gh auth switch --user MarkEdmondson1234`
- [ ] Push tag to remote BEFORE creating release
- [ ] Create release using `gh release create`

**Why this matters:**
- Wrong account = authentication failures
- Tag must be on remote before release creation
- Saves frustration and debugging time

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

**ENABLED BY DEFAULT**: Polymorphic lambdas specialized at call sites (M-POLY-A).

**Quick reference:**
```bash
# Normal compilation (enabled by default)
ailang run --entry main --caps IO module.ail

# Debug mode (show specialization stats)
ailang run --debug-compile module.ail

# Disable (emergency escape hatch)
ailang run --no-mono module.ail
```

**What works (v0.4.0):**
- ✅ Inline lambdas: `(\x. \y. if x > y then x else y)(3.14)(2.71)`
- ✅ Comparison operators: `>`, `<`, `>=`, `<=`, `==`, `!=`

**Known issue (v0.4.0):**
- ❌ Arithmetic operators need type annotations: `let add: float -> float -> float = \x. \y. x + y`
- **Workaround**: Use inline lambdas or add type annotations
- **Fix**: Coming in v0.4.2 (Phase 2)

**Resource limits:**
- Per-function: 16 specializations
- Per-module: 512 specializations

**For complete documentation:**
- See [design_docs/implemented/v0_4_0/monomorphization.md](design_docs/implemented/v0_4_0/monomorphization.md)
- See [M-POLY-B-PHASE1-COMPLETION-REPORT.md](M-POLY-B-PHASE1-COMPLETION-REPORT.md)

**When asked to run evals, compare benchmarks, or update benchmark results:**

→ **ALWAYS use the [eval-orchestrator](.claude/agents/eval-orchestrator.md) agent**

The agent knows how to:
- Run benchmarks with cost-conscious defaults (cheap models for dev, --full for releases)
- Compare results, validate fixes, generate reports
- Update the benchmark dashboard (docs/static/benchmarks/latest.json)
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

### Adding Builtin Functions

**To add a builtin function, use the `builtin-developer` skill.**

**Quick Reference:**
- **Development time**: ~2.5 hours (down from 7.5h with legacy system)
- **Status**: M-DX1 COMPLETE - 52 builtins, fully documented
- **Key benefit**: 67% faster with single-file registration

**Validation commands:**
```bash
ailang doctor builtins              # Validate all 52 builtins
ailang builtins list --by-module    # Browse by module
ailang builtins check-migration     # Check for orphaned builtins
```

**System overview:**
- **Central Registry** (`internal/builtins/spec.go`) - Single registration point
- **Type Builder DSL** (`internal/types/builder.go`) - Fluent type construction
- **Test Harness** (`internal/effects/testctx/`) - Hermetic testing with mocking
- **Auto-wiring** - Registry connects to runtime/link automatically (no feature flag since v0.3.10)

**For complete guide:**
- Use the `builtin-developer` skill for step-by-step workflow
- See [M-DX1-FINAL-SUMMARY.md](M-DX1-FINAL-SUMMARY.md) for detailed summary
- See `design_docs/planned/easier-ailang-dev.md` for design rationale

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

# Generate markdown dashboard (DEPRECATED - use JSON dashboard instead)
# ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=markdown > docs/BENCHMARK_COMPARISON.md

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

**Quick reference table:**

| Flag | Purpose | Use When |
|------|---------|----------|
| `DEBUG_STRICT=1` | Fail loudly on unhandled cases | Development, CI |
| `DEBUG_MONO_VERBOSE=1` | Monomorphization tracing | Type issues |
| `DEBUG_OPERATOR_LOWERING=1` | Operator resolution | Dispatch issues |
| `DEBUG_PARSER=1` | Token position tracing | Parser bugs |

**Recommended combinations:**
```bash
# Development mode
DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang run test.ail

# CI mode
DEBUG_STRICT=1 make test

# Parser debugging
DEBUG_PARSER=1 ailang run test.ail
```

**For detailed documentation**: See [docs/guides/debugging.md](docs/guides/debugging.md)

#### Keeping `ailang` Up to Date

**After making code changes:**
```bash
make quick-install  # Fast reinstall (recommended)
make install        # Full reinstall with version info
```

**Important**: `ailang` in PATH points to `/Users/mark/go/bin/ailang` (system), NOT `bin/ailang` (local). Always reinstall after building.

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

#### 4. Documentation Website - Example Import Pattern

**CRITICAL: Never embed code examples directly in documentation!**

The website uses **raw-loader** to import actual example files from `examples/`. This ensures:
- Examples always match working code
- Syntax changes automatically propagate to docs
- No manual updates needed when examples change
- One less maintenance burden

**Correct pattern** (import from examples/):
```mdx
import CodeBlock from '@theme/CodeBlock';
import HelloExample from '!!raw-loader!@site/../examples/runnable/hello.ail';

<CodeBlock language="typescript" title="examples/runnable/hello.ail">
  {HelloExample}
</CodeBlock>
```

**Wrong pattern** (embedded code):
```mdx
❌ DON'T DO THIS:
```typescript
module examples/hello
-- This code will drift out of sync!
```
```

**Files that use example imports:**
- `docs/docs/intro.mdx` - Hello world and factorial
- `docs/docs/examples.mdx` - Multiple working examples
- `docs/docs/guides/getting-started.mdx` - Tutorial examples

**Configuration:**
- Raw-loader configured in `docs/docusaurus.config.js`
- Uses webpack plugin to handle `.ail` files
- Examples imported from `@site/../examples/` path

### Writing AILANG Code

**When writing AILANG code during development:**
Refer to the **AI Teaching Prompt** for comprehensive syntax guidance:

**Get the teaching prompt:**
```bash
ailang prompt                           # Display current/active prompt
ailang prompt --version v0.3.24         # Display specific version
ailang prompt --list                    # List all available versions
ailang prompt > syntax.md               # Save to file
```

- Prompts are version-locked and tracked in `prompts/versions.json`
- Validated through multi-model testing (Claude, GPT, Gemini)
- Covers syntax, limitations, common pitfalls, and working examples
- Each prompt version corresponds to a specific AILANG version

**Quick reference:**
```bash
ailang run --caps IO,FS --entry main module.ail  # Run module
ailang repl                                        # Start REPL
:type expr                                         # Check type in REPL
```

**Critical syntax notes:**
- **Import Aliasing (v0.4.8):** Rename modules and symbols on import
  - **Module alias:** `import std/list as List` → enables qualified access `List.length(xs)`, `List.map(f, xs)`
  - **Symbol alias:** `import std/list (length as listLength)` → use `listLength(xs)` directly
  - **Combined:** `import std/list as List (map, filter)` → direct access to `map`, `filter` + qualified `List.*`
  - **Use case:** Resolve name clashes when importing from multiple modules
- **⚠️ NULLARY FUNCTIONS BROKEN (v0.4.5):** Nullary functions (zero-arg) cannot be called from AILANG code
  - **Affected**: `_env_getArgs`, `_clock_now`, `_io_readLine`
  - **Issue**: No syntax to call them - `f` returns function object, `f()` is arity mismatch
  - **Workaround**: Use direct Go implementation via `effects.Call()` (Go tests only)
  - **Fix planned**: M-DX10 in v0.4.6 (see [design doc](design_docs/planned/v0_4_6/m-dx10-nullary-function-calls.md))
  - **Status**: CLI args feature implemented but not usable from AILANG until M-DX10 is fixed
- **Type parameters:** Use `[T]` NOT `(T)` - distinguishes type/term application
- **Match in blocks:** Known parser bug (nested delimiter tracking) - extract to helper function

**For detailed syntax, limitations, and examples:**
- Use `ailang prompt` to get the latest teaching prompt
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

## 🎯 RELEASE WORKFLOW

**For releases**: Use the `release-manager` skill
**After release**: Use the `post-release` skill to update dashboard

**Quick reference:**
- **release-manager** handles: verification, tagging, CI/CD monitoring
- **post-release** handles: eval baselines, dashboard updates, docs

**For manual workflows**: See [docs/guides/release.md](docs/guides/release.md)

---

## 📐 Code Organization Principles

**AILANG is designed for AI maintenance. Keep files small and focused.**

**File size targets:**
- Sweet spot: 200-500 lines
- Acceptable: 500-800 lines
- Critical: 1200+ lines (MUST split)

**Check file sizes:**
```bash
make check-file-sizes    # Fails CI if >800 lines
make report-file-sizes   # Show files >500 lines
```

**For refactoring**: Use `codebase-organizer` agent  
**For detailed patterns**: See [docs/guides/file-organization.md](docs/guides/file-organization.md)

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

## Parser Development

**For parser development, use the `parser-developer` skill.**

**Critical conventions (saves 30% development time):**
1. **Token position**: Parser leaves cursor AT last token (not after)
2. **No NEWLINE tokens**: Lexer skips `\n` as whitespace
3. **Use `DEBUG_PARSER=1`**: Trace token positions for debugging
4. **Use `make doc PKG=<package>`**: API discovery (80% faster)

**Common gotchas:**
- ❌ IntLit is `int64`, not `int` (will panic if wrong!)
- ❌ Never check for `lexer.NEWLINE` (lexer never generates it)
- ❌ Print errors BEFORE `t.Fatalf` in tests
- ❌ `string(rune(i))` produces unprintable chars (use `fmt.Sprintf`)

**Quick reference:**
```bash
# Debug parser token flow
DEBUG_PARSER=1 ailang run test.ail

# Find API signatures
make doc PKG=internal/parser | grep "parseExpression"

# Check AST types
grep "^type.*struct" internal/ast/ast.go | head -20
```

**Common constructors:**
- `parser.New(lexer)` - Takes lexer instance
- `elaborate.NewElaborator()` - No arguments (Surface → Core)
- `types.NewTypeChecker(core, imports)` - Takes Core prog + imports
- `link.NewLinker()` - No arguments (Dictionary linking)

**For detailed guide:**
- Use the `parser-developer` skill for complete conventions
- See `design_docs/planned/v0_3_15/m-dx9-parser-developer-experience.md` for design docs
- See `docs/CONTRIBUTING.md` for development workflows

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
