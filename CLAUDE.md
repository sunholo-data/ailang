# Claude Instructions for AILANG Development

## 🚀 SESSION START ROUTINE

**At the start of EVERY session, check for agent messages.**

### 📬 Message Commands (Quick Reference)

```bash
# LIST MESSAGES
ailang messages list                 # All messages (alias: ailang msg ls)
ailang messages list --unread        # Only unread messages
ailang messages list --inbox user    # Messages for specific inbox
ailang messages list --json          # JSON output (for scripting)

# READ MESSAGE CONTENT
ailang messages read MSG_ID          # Show full message content
ailang messages read MSG_ID --peek   # View without marking as read
ailang messages read 877             # Short ID prefix (like git) - v0.6.3+

# FORWARD MESSAGES (v0.6.3+)
ailang messages forward --to design-doc-creator 877    # Forward to different inbox
ailang messages forward --to coordinator --reason "Label changed" 877

# INTERACTIVE MODE (v0.6.3+)
ailang messages                      # Interactive menu with keyboard navigation

# ACKNOWLEDGE (mark as read)
ailang messages ack MSG_ID           # Mark specific message as read
ailang messages ack --all            # Mark all as read
ailang messages ack --all --inbox user  # Mark all in inbox as read

# UN-ACKNOWLEDGE (mark as unread again)
ailang messages unack MSG_ID         # Move back to unread

# SEND MESSAGE
ailang messages send INBOX "message" --title "Title" --from "agent-name"

# CLEANUP OLD MESSAGES
ailang messages cleanup --older-than 7d   # Remove messages older than 7 days
ailang messages cleanup --expired         # Remove expired messages
ailang messages cleanup --dry-run         # Preview without deleting

# WATCH FOR NEW MESSAGES
ailang messages watch                # Watch all inboxes
ailang messages watch --inbox user   # Watch specific inbox

# GITHUB SYNC (v0.5.9+)
ailang messages send user "Bug report" --type bug             # bug/feature imply --github
ailang messages send user "Feature" --type feature            # Auto-creates GitHub issue
ailang messages send user "Research" --type research          # Custom type, local only
ailang messages send user "Docs" --type docs --github         # Custom type + explicit GitHub
ailang messages import-github                                  # Import issues as messages
ailang messages import-github --labels bug,feature            # Filter by labels
ailang messages import-github --dry-run                       # Preview without importing

# SEMANTIC SEARCH (v0.5.11+)
ailang messages search "parser error"                         # Search by semantic content
ailang messages search "bugs" --neural                        # Use Ollama embeddings
ailang messages list --similar-to MSG_ID                      # Find similar messages
ailang messages list --collapsed                              # Hide duplicates

# ENVELOPE SEARCH (v0.8.1+ M-SEMANTIC-ENVELOPE)
ailang messages search --space code "internal/types"          # Search by code context
ailang messages search --space intent "fix crash"             # Search by intent
ailang messages search --space resolution "parser"            # Search past resolutions

# SEMANTIC ENVELOPE (v0.8.1+)
ailang messages send INBOX "msg" --envelope-code FILE         # Attach code context embedding
ailang messages send INBOX "msg" --envelope-context "desc"    # Attach session context embedding
ailang messages triage                                        # Cluster unread by intent
ailang messages triage --cluster-by code                      # Cluster by code region
ailang messages triage --cluster-by code --inbox user         # Filter + cluster

# DEDUPLICATION (v0.5.11+)
ailang messages dedupe                                        # Report duplicate groups
ailang messages dedupe --threshold 0.90                       # Custom similarity threshold
ailang messages dedupe --apply                                # Mark duplicates
```

### Session Start Workflow

1. **SessionStart hook runs automatically** - injects unread messages into system reminders
2. **If no messages in reminders**, manually check: `ailang messages list --unread`
3. **When messages exist**: Summarize to user, ask what to do
4. **After handling**: `ailang messages ack --all`
5. **If task fails**: `ailang messages unack MSG_ID` (moves back for retry)

### Message Storage

All messages are stored in the unified SQLite database at `~/.ailang/state/collaboration.db`.
This database is shared between the CLI (`ailang messages`) and the Collaboration Hub dashboard.

**Message statuses:** `unread`, `read`, `archived`, `deleted`

### Responding to External Projects

For bug reports/feature requests from external projects (e.g., stapledons_voyage):

```bash
# Send message to external project inbox
ailang messages send stapledons_voyage "Design doc created for v0.4.9" \
  --title "Bug acknowledged" --from "ailang"
```

**Response workflow:**
1. Review messages (from SessionStart hook or `ailang messages list --unread`)
2. For bugs: Create design docs
3. Send response: `ailang messages send PROJECT "message" --title "Title"`
4. Acknowledge: `ailang messages ack --all`

### Technical Details

<details>
<summary>Click to expand hook configuration details</summary>

**Hooks:**
- Configured in `.claude/settings.local.json`
- Logged to `~/.ailang/state/hooks.log`

**SessionStart Hook** (`scripts/hooks/session_start.sh`):
- Uses `ailang messages list --unread --json` to check for messages
- Outputs to stdout (appears in system reminders)
- Does NOT auto-mark as read

**Stop Hook** (`scripts/hooks/agent_handoff.sh`):
- Detects design docs created in session
- Sends handoff messages to sprint-planner

**Message Lifecycle:**
1. Agent sends → status: `unread`
2. Hook injects into context
3. `ack` → status: `read`
4. `unack` → status: `unread`
5. `cleanup` → permanently deleted

</details>

### GitHub Sync Configuration (v0.5.9+)

Messages can be synced bidirectionally with GitHub Issues. Configure in `~/.ailang/config.yaml`:

```yaml
github:
  # REQUIRED: Must match the active `gh auth status` user
  # HARD FAILS if mismatch - prevents accidental commits to wrong account
  # sunholo-voight-kampff = Claude Code agent account (use for pushes)
  # MarkEdmondson1234 = Human developer account
  expected_user: sunholo-voight-kampff

  # Default repo for issue creation/import
  default_repo: sunholo-data/ailang

  # Labels automatically added to created issues
  create_labels:
    - ailang-message

  # Labels to filter when importing issues
  watch_labels:
    - from:stapledon
    - from:ailang-core

  # Auto-import on session start (default: true)
  auto_import: true
```

**Workflow:**
1. **Outgoing:** `ailang messages send ... --github` creates local message + GitHub issue
2. **Incoming:** `ailang messages import-github` (or session start hook) imports issues as messages
3. **Deduplication:** Issues are tracked by `github_issue_number` - no duplicates

**Error handling:**
- Messages are ALWAYS saved locally first
- GitHub sync failures don't lose messages
- Account mismatch = HARD FAIL with fix instructions (`gh auth switch --user`)

### Design Docs Search (v0.5.11+)

Search design documentation using SimHash or neural embeddings:

```bash
# Basic search (SimHash - fast)
ailang docs search "parser error"

# Filter by stream
ailang docs search --stream planned "type inference"
ailang docs search --stream implemented "builtin"

# Neural search (uses Ollama embeddings - requires local Ollama)
ailang docs search --neural "semantic search"
ailang docs search --stream planned --neural "lazy embedding"

# JSON output for scripting
ailang docs search --json "query"
```

**Note:** Flags must come BEFORE the query.

**Neural Search Behavior:**
- **When using `--neural`**, embeddings are computed lazily only for the bounded SimHash candidate set
- Do NOT attempt to embed the entire doc corpus - that's the whole point of lazy embeddings
- First search may be slower (computing embeddings), subsequent searches reuse cached embeddings
- Cache stored at `~/.ailang/cache/doc_embeddings.json` with model version tagging
- Model change triggers re-embedding on next search

**Configuration:** Uses same Ollama config as messages in `~/.ailang/config.yaml`:
```yaml
embeddings:
  # Provider: "ollama" (local), "openai", "gemini", or "none"
  provider: ollama
  ollama:
    model: embeddinggemma  # or nomic-embed-text
    endpoint: http://localhost:11434
  # openai:               # v0.8.1+ (uses OPENAI_API_KEY env var)
  #   model: text-embedding-3-small
  # gemini:               # v0.8.1+ (uses GOOGLE_API_KEY env var)
  #   model: text-embedding-004
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
# But this repo needs: sunholo-voight-kampff (Claude Code agent account)

gh release create v0.4.4  # FAILS with auth error
git push origin v0.4.4     # FAILS with auth error
```

**✅ CORRECT approach before ANY release or git push operations:**

1. **ALWAYS check active GitHub account:**
   ```bash
   gh auth status
   ```

2. **Verify the active account matches the agent account:**
   - This repo (sunholo-data/ailang) needs: `sunholo-voight-kampff`
   - `MarkEdmondson1234` is the human developer account
   - If active account is `rw-markedmondson` → WRONG ACCOUNT

3. **Switch to correct account if needed:**
   ```bash
   gh auth switch --user sunholo-voight-kampff
   ```

4. **Then proceed with release operations:**
   ```bash
   git push origin v0.4.4              # Push tag
   gh release create v0.4.4 ...        # Create release
   ```

**Checklist for releases:**
- [ ] Check `gh auth status` - verify active account
- [ ] Switch account if needed: `gh auth switch --user sunholo-voight-kampff`
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
- ❌ Guessing which tools to use for benchmarks/evals - use `eval-analyzer` skill or `ailang eval-*` commands

**The `ailang` CLI exists to make YOUR life easier.** Commands like `ailang chains`, `ailang messages`, `ailang eval-*`, and `ailang dashboard` are designed so you can analyze past sessions, audit agent work, and understand system state without resorting to raw SQLite queries or ad-hoc scripts.

**If the CLI output is insufficient for what you need:**
- ✅ **Propose a CLI improvement** — suggest a new flag, subcommand, or output format
- ✅ **File a message** — `ailang messages send user "ailang chains needs X" --title "CLI: chains improvement"`
- ✅ **Create a design doc** — for larger gaps, propose the enhancement properly
- ❌ **Don't silently work around it** — writing one-off sqlite3 queries or bash pipelines means the next session will hit the same gap
- ❌ **Don't accept poor ergonomics** — if you're doing 5 commands to get info that should be 1, say so

**Examples of good proposals:**
- "ailang chains view needs a `--failures-only` flag to skip successful stages"
- "ailang chains stats should break down by model, not just agent"
- "ailang eval-chains failures should include the actual compiler error, not just 'failed'"

**The goal:** Every analysis task you commonly perform should be a single CLI command with clear output.

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

### 4. SAFE TYPE TRAVERSAL - CYCLE PROTECTION (v0.5.9)

**API DESIGN RULE** (from M-PERF2 post-mortem):

> Every function of shape `func(types.Type) T` MUST document cycle-safety.
> Either use `traverse.Walk` or add a `visited` parameter.

**The traverse package** (`internal/types/traverse/`) provides safe type traversal:

```go
// Safe - automatic cycle detection
vars := traverse.CollectFreeVars(someType)

// Safe - manual callback
traverse.Walk(someType, func(t types.Type) {
    // process type
})

// Check for cycles
if traverse.HasCycles(recursiveType) {
    // handle recursive ADT
}
```

**When to use:**
- ✅ Code outside `internal/types/` that needs to traverse types
- ✅ Pipeline code, evaluator, code generators
- ✅ Any new type analysis functions

**When NOT needed:**
- Code inside `internal/types/` already has internal cycle protection
- Simple type checks that don't recurse

**Implementation:**
- Core visitor: `internal/types/traverse/traverse.go` (~185 LOC)
- Safe wrappers: `internal/types/traverse/wrappers.go` (~160 LOC)
- Tests: `internal/types/traverse/traverse_test.go` (~560 LOC, 31 tests)

### 5. MONOMORPHIZATION - CALL-SITE SPECIALIZATION (v0.4.0)

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

### 6. SYSTEMIC FIXES - AUDIT BEFORE PATCHING

**Before fixing a bug, ALWAYS ask: "Is this part of a larger pattern?"**

**The Anti-Pattern (incremental special-casing):**
```
v1: Add feature for case A
v2: Bug! Add special case for B
v3: Bug! Add special case for C
v4: Bug! Add special case for D
...forever patching
```

**The Pattern to Follow:**
```
v1: Bug report for case B
    BEFORE fixing:
    1. Search for similar code paths
    2. Check if A, C, D have same gap
    3. Design ONE fix covering ALL cases
v2: Unified fix - no future bugs in this area
```

**Example (M-CODEGEN-UNIFIED-SLICE-CONVERTERS, Dec 2025):**
- Bug: `[SolarPlanet]` return type panics
- ❌ Quick fix: Add `ConvertToSolarPlanetSlice`
- ✅ Audit found: `[]float64` ALSO broken, `[]*ADTType` partially broken
- One unified fix covers all 3 gaps

**Warning signs of fragmented design:**
- Multiple maps tracking similar things (`adtSliceTypes`, `recordTypes`...)
- Switch statements with growing case lists
- Bug fixes that add `|| specialCase` conditions

**When planning bug fixes:**
- Allocate 30-60 min for systemic analysis
- Search codebase for similar code paths
- Check git history - has this area been patched repeatedly?

**When asked to run evals, compare benchmarks, or update benchmark results:**

→ **Use the `eval-analyzer` skill or `ailang eval-*` commands directly**

Common eval commands:
```bash
ailang eval-suite --models gpt5-mini,claude-haiku-4-5  # Run benchmarks
ailang eval-compare baseline1 baseline2                 # Compare results
ailang eval-report results/ v0.5.10 --format=json      # Update dashboard
```

**DO NOT:**
- ❌ Write custom Python/bash scripts for benchmark analysis
- ❌ Manually regenerate dashboard files

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
- **collaboration-hub** - Develop and modify the Collaboration Hub UI (React frontend)
- **codebase-organizer** - Monitor and refactor large files into AI-friendly modules
- **design-spec-auditor** - Verify code implementation matches design specifications
- **github-issue-triage** - Monitor and triage GitHub issues against design docs, identify closable issues
- **test-coverage-guardian** - Analyze test coverage, identify gaps, improve test quality
- **perf-reviewer** - Review code for performance and run cross-language benchmarks (AILANG vs Python vs Go)
- **trace-debugger** - Debug performance issues using OTEL traces, analyze bottlenecks, suggest new instrumentation

**Complete skill documentation**: See [.claude/skills/README.md](.claude/skills/README.md)

**Creating new skills**: Use the `skill-builder` skill or see [.claude/skills/SKILLS_GUIDE.md](.claude/skills/SKILLS_GUIDE.md)

### Using Skills

Skills are invoked automatically by Claude when appropriate for the task. Just describe what you want:

- "Create a skill for managing database migrations" → `skill-builder` skill
- "Ready to release v0.3.14" → `release-manager` skill
- "Update benchmarks" → `post-release` skill
- "Help me write AILANG code" → `use-ailang` skill
- "Plan the sprint" → `sprint-planner` skill
- "Add a feature to the monitoring dashboard" → `collaboration-hub` skill
- "Triage GitHub issues" or "What issues are open?" → `github-issue-triage` skill
- "Benchmark AILANG vs Python" or "Review for performance" → `perf-reviewer` skill
- "Why is compilation slow?" or "Debug with traces" → `trace-debugger` skill
- "Start dev cycle" → `dev-cycle` agent (messages → design → sprint → implement)

### Skills vs Agents vs Commands

- **Skills** (.claude/skills/): Focused workflows with progressive disclosure and automation
- **Agents** (.claude/agents/): Multi-stage orchestration coordinating multiple skills
  - `dev-cycle` - Full workflow: messages → design → sprint → implementation
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
│   ├── ai/             # Unified AI providers ✅ COMPLETE (v0.5.10)
│   │   ├── anthropic/  # Claude API client (text generation)
│   │   ├── openai/     # OpenAI API client (Chat + Responses)
│   │   ├── gemini/     # Gemini API client (AI Studio + Vertex)
│   │   └── ollama/     # Ollama client (local models)
│   ├── executor/       # Agentic CLI executors ✅ COMPLETE (v0.6.1)
│   │   ├── claude/     # Claude Code CLI (headless mode)
│   │   └── gemini/     # Gemini CLI (agentic coding)
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
├── stdlib/             # Standard library ✅ COMPLETE (27 modules: std/ai, std/array, std/bytes, std/clock, std/datetime, std/debug, std/embedding, std/env, std/fs, std/game, std/io, std/json, std/list, std/math, std/net, std/option, std/process, std/rand, std/result, std/sem, std/sharedindex, std/sharedmem, std/simhash, std/stream, std/string, std/xml, std/zip)
├── tools/              # Development tools ✅ (eval, benchmarking, verification)
├── benchmarks/         # AI code generation benchmarks ✅
├── examples/           # Example .ail programs (66 files, 48 passing)
├── tests/              # Test suite ✅
└── docs/               # Documentation ✅ COMPLETE
```

### AI Provider vs Executor Architecture (IMPORTANT!)

**⚠️ CRITICAL DISTINCTION: There are TWO different ways to use AI in AILANG:**

#### 1. `internal/ai/` - API Providers (Text Generation Only)
- **Purpose**: Simple text generation via HTTP APIs
- **Interface**: `ai.Provider` with `Generate(ctx, *Request) (*Response, error)`
- **Use for**: Research, documentation, simple Q&A, text completion
- **Packages**:
  - `ai/gemini/` - Gemini API (AI Studio or Vertex AI)
  - `ai/anthropic/` - Claude API (Messages API)
  - `ai/openai/` - OpenAI API (Chat or Responses)
  - `ai/ollama/` - Ollama (local models)

```go
// Example: Simple text generation
client := gemini.NewClient(apiKey)
resp, err := client.Generate(ctx, &ai.Request{
    Model:      "gemini-2.5-flash",
    UserPrompt: "Explain recursion",
})
```

#### 2. `internal/executor/` - CLI Executors (Agentic Coding)
- **Purpose**: Full agentic coding with file editing, code execution, tool use
- **Interface**: `executor.Executor` with `Execute(ctx, *Task) (*Result, error)`
- **Use for**: Bug fixes, feature implementation, code refactoring, anything requiring file changes
- **Packages**:
  - `executor/claude/` - Claude Code CLI (`claude -p --output-format json`)
  - `executor/gemini/` - Gemini CLI (`gemini --output-format json`)

```go
// Example: Agentic coding task
exec, _ := executor.GlobalFactory().GetExecutor("claude")
result, err := exec.Execute(ctx, &executor.Task{
    Directive: "Fix the null pointer bug in parser.go",
    Workspace: "/path/to/repo",
})
```

**When to use which:**
| Task Type | Package | Example |
|-----------|---------|---------|
| Bug fix | `executor/` | "Fix the type error in line 42" |
| New feature | `executor/` | "Add a --verbose flag to the CLI" |
| Refactoring | `executor/` | "Split this file into smaller modules" |
| Documentation | `ai/` or `executor/` | Simple docs: `ai/`, complex: `executor/` |
| Research | `ai/` | "What are best practices for X?" |
| Code review | `ai/` | "Review this diff for issues" |

**Key difference**: `executor/` runs CLI tools that can edit files and execute commands. `ai/` just generates text responses.

## Development Workflow

### Building and Testing
```bash
make build          # Build the interpreter to bin/
make install        # Install ailang to system (makes it available everywhere)
make test           # Run all tests
make run FILE=...   # Run an AILANG file
make repl           # Start interactive REPL
```

### Collaboration Hub Server

**For developing or modifying the Collaboration Hub UI**, use the `collaboration-hub` skill.

**Starting services (recommended):**
```bash
make services-start             # Start server + coordinator together
make services-status            # Check both services
make services-stop              # Stop both services
make services-restart           # Rebuild and restart both
```

**Starting server only:**
```bash
ailang serve                    # Start on default port 1957
ailang serve --port 8080        # Use custom port
ailang serve --db /tmp/test.db  # Use custom database
```

**Key endpoints:**
- **UI**: http://localhost:1957/
- **WebSocket**: ws://localhost:1957/ws
- **REST API**: http://localhost:1957/api/
- **Coordinator Events**: http://localhost:1957/api/coordinator/events
- **Health**: http://localhost:1957/health

**After UI changes:**
```bash
cd ui && npm run build                              # Build React app
cp -r ui/dist/* internal/server/dist/               # Copy to server
make services-restart                               # Rebuild and restart
```

**Architecture:**
- **Backend**: `internal/server/` (Go HTTP server with SQLite)
- **Frontend**: `ui/` (React + TypeScript + Vite)
- **Database**: `~/.ailang/state/collaboration.db`
- **Event Handler**: `internal/server/handlers_coordinator.go` (receives coordinator events)

**For complete guide**: Use the `collaboration-hub` skill

### Coordinator Daemon (Task Delegation)

**The Coordinator is an always-on daemon that can execute tasks autonomously using AI agents (Claude Code or Gemini CLI).**

**When to use the coordinator:**
- Delegate long-running tasks that don't need immediate attention
- Run multiple tasks concurrently in isolated environments
- Let background agents handle bug fixes, features, or research while you continue other work
- Chain agents together (design-doc-creator → sprint-planner → sprint-executor)

**Service Management (Recommended):**
```bash
# Start both server and coordinator (correct order)
make services-start

# Check status of both services
make services-status

# Stop both services
make services-stop

# Restart with fresh build
make services-restart
```

**Individual Commands:**
```bash
# Start the daemon
ailang coordinator start

# Check if running
ailang coordinator status

# Stop the daemon
ailang coordinator stop

# Start with custom settings
ailang coordinator start --poll-interval 10s --max-worktrees 5
```

**Agent Configuration (v0.6.2+):**

Agents are configured in `~/.ailang/config.yaml`:

```yaml
coordinator:
  default_provider: claude

  agents:
    # Design Doc Creator - reads GitHub issues, creates design docs
    - id: design-doc-creator
      label: "Design Doc Creator"
      inbox: design-doc-creator
      workspace: /path/to/project
      capabilities: [research, docs]
      trigger_on_complete: [sprint-planner]
      auto_approve_handoffs: false
      session_continuity: true

    # Sprint Planner - creates sprint plans from design docs
    - id: sprint-planner
      label: "Sprint Planner"
      inbox: sprint-planner
      workspace: /path/to/project
      trigger_on_complete: [sprint-executor]
      auto_approve_handoffs: false

    # Sprint Executor - implements approved sprint plans
    - id: sprint-executor
      label: "Sprint Executor"
      inbox: sprint-executor
      workspace: /path/to/project
      trigger_on_complete: []
      auto_merge: false

  github_sync:
    enabled: true
    interval_secs: 300
    target_inbox: design-doc-creator
```

**Agent Configuration Fields:**
| Field | Description |
|-------|-------------|
| `id` | Unique agent identifier |
| `inbox` | Message inbox to watch |
| `workspace` | Base directory for worktrees |
| `trigger_on_complete` | Agent IDs to trigger when this agent completes |
| `auto_approve_handoffs` | Skip approval for agent-to-agent handoffs |
| `auto_merge` | Automatically merge approved changes |
| `session_continuity` | Use `--resume` (Claude) or `--conversation-id` (Gemini) |
| `timeout` | Hard ceiling execution timeout as Go duration (e.g., "30m", "1h"). Default: 60m |
| `idle_timeout` | Kill if no output for this long (e.g., "3m", "5m"). Default: 3m |
| `approval` | Config-driven GitHub labels (see below) |

**Approval Configuration (v0.8.0+):**

Agents can define custom GitHub labels for approval workflows. Known agents (`design-doc-creator`, `sprint-planner`, `sprint-executor`) have defaults; custom agents require explicit config.

```yaml
coordinator:
  agents:
    - id: custom-agent
      inbox: custom-agent
      workspace: /path/to/project
      approval:
        needs_label: needs-custom-approval    # Added when work complete
        approved_label: custom-approved       # Human adds to approve
        github_comment_template: generic      # Comment template (optional)
```

**Default labels for known agents:**
| Agent ID | needs_label | approved_label |
|----------|-------------|----------------|
| `design-doc-creator` | `needs-design-approval` | `design-approved` |
| `sprint-planner` | `needs-sprint-approval` | `sprint-approved` |
| `sprint-executor` | `needs-merge-approval` | `merge-approved` |

**How it works:**
1. Agent completes work → `needs_label` added to GitHub issue
2. Human reviews and adds `approved_label`
3. Coordinator detects label → triggers `trigger_on_complete` agents (if any)
4. Labels are removed to prevent re-triggering

**Script Invoke Type (v0.6.4+):**

Any agent can run deterministic shell scripts instead of AI by setting `invoke.type: script`:

```yaml
- id: eval-runner
  inbox: eval-runner
  workspace: /path/to/project
  invoke:
    type: script                    # Run script instead of AI
    command: ./scripts/run-eval.sh  # Script to execute
    env_from_payload: true          # JSON payload → env vars
    timeout: 2h                     # Execution timeout
  output_markers: ["EVAL_RESULT:", "PASS_RATE:"]
```

**Usage:**
```bash
# Send JSON payload - becomes environment variables
ailang messages send eval-runner '{"model": "gpt5", "benchmarks": "all"}' \
  --title "Run v0.6.4 baseline"

# Script receives: MODEL=gpt5 BENCHMARKS=all ./scripts/run-eval.sh
# Plus: AILANG_TASK_ID, AILANG_MESSAGE_ID, AILANG_WORKSPACE
```

**Benefits:**
- Deterministic (same input = same execution)
- Zero cost ($0.00 - no AI tokens)
- Composable with AI agents (AI → Script → AI pipelines)
- Same approval workflow, dashboard streaming, output markers

**Multi-Agent Workflow:**
```
GitHub Issue → design-doc-creator → [Approval] → sprint-planner → [Approval] → sprint-executor → [Approval] → Merged
```

**Delegating Tasks from Claude Code:**
```bash
# Send to design-doc-creator (first stage)
ailang messages send design-doc-creator "Create design doc for semantic caching" \
  --title "Feature: Semantic Caching" --from "user"

# Send to sprint-planner (or let design-doc-creator trigger it)
ailang messages send sprint-planner "Plan sprint for M-CACHE" \
  --title "Sprint: M-CACHE" --from "user"

# Send to general coordinator (ad-hoc tasks)
ailang messages send coordinator "Fix the null pointer bug in parser.go" \
  --title "Bug: Parser NPE" --from "claude-code" --type bug
```

**Approval Workflow:**
- Task completes → Approval request created
- Dashboard shows pending approvals with git diff viewer
- Approve → Changes merged to dev, next agent triggered (if configured)
- Reject → Feedback loop with re-trigger support

**Approval Types (v0.6.5+):**

When an agent has `trigger_on_complete` configured with `auto_approve_handoffs: false`, approvals are **combined**:

| Type | CLI Display | On Approve |
|------|-------------|------------|
| `merge` | `[merge]` | Merges code to dev branch only |
| `merge_handoff` | `[merge+handoff] → agent` | Merges code AND triggers next agent |

**Session Continuity:**
- **On Approve**: Handoff message sent to next agent with `session_id` for context
- **On Reject**: Same agent retries with same worktree, same session, feedback added
- Both preserve git worktree and conversation context via `--resume <sessionId>`

**Approval Response Format (API):**
```json
{
  "id": "apr-123",
  "type": "merge_handoff",
  "context_json": "{\"handoff_targets\":[\"sprint-planner\"],\"session_id\":\"...\",\"source_agent\":\"design-doc-creator\"}"
}
```

**CLI Approval Actions (v0.6.4+):**
When reviewing pending approvals with `ailang coordinator pending`:

| Key | Action | Description |
|-----|--------|-------------|
| `a` | Approve | Merge changes to dev branch; trigger next agent if `merge_handoff` |
| `r` | Reject | Prompt for feedback, send to agent inbox, re-trigger task |
| `c` | Chat | View conversation history (turn-by-turn with tool calls) |
| `d` | Diff | View git diff of changes |
| `q` | Quit | Exit without action |

**Feedback Loop (M-TRANSCRIPT):**
When you reject a task, the coordinator:
1. Prompts for feedback reason (why the work needs revision)
2. Stores feedback as `human_feedback` event in task_events table
3. Sends message to agent's inbox with feedback content
4. Re-triggers task with `iteration=2` (same task ID, preserves context)
5. Claude uses `--resume <sessionId>` to continue with full conversation history
6. Max 3 iterations to prevent infinite loops

```bash
# Manual rejection with feedback
ailang coordinator reject <task-id> --feedback "Need to add error handling for edge cases"

# Skip prompt (for scripted use)
ailang coordinator reject <task-id> --feedback "..." --no-prompt

# Reject without re-triggering (permanent rejection)
ailang coordinator reject <task-id> --no-retrigger
```

**Real-Time Dashboard Streaming:**
The coordinator streams task execution events to the Collaboration Hub dashboard:
- Start the server first (`ailang serve` or `make services-start`)
- Events are POSTed to `http://127.0.0.1:1957/api/coordinator/events`
- Server broadcasts via WebSocket to all connected browsers
- View at http://localhost:1957 in the Task Execution tab

**Task Routing:**
| Task Type | Primary Provider | Use Case |
|-----------|------------------|----------|
| Bug Fix | Claude Code CLI | Code changes requiring file edits |
| Feature | Claude Code CLI | New functionality implementation |
| Refactor | Claude Code CLI | Code restructuring |
| Test | Claude Code CLI | Writing or fixing tests |
| Docs | Gemini API | Documentation writing |
| Research | Gemini API | Investigation, exploration |

**Storage:**
- Task state: `~/.ailang/state/coordinator.db` (SQLite)
- Worktrees: `~/.ailang/state/worktrees/<agent-id>/<task-id>/`
- Logs: `~/.ailang/logs/coordinator.log`
- Config: `~/.ailang/config.yaml`

**Architecture:**
- **Daemon** (`internal/coordinator/daemon.go`) - Main loop, lifecycle
- **Agent Registry** (`internal/coordinator/agent_registry.go`) - Agent configuration management
- **HTTP Broadcaster** (`internal/coordinator/http_broadcaster.go`) - Streams events to dashboard
- **Analyzer** (`internal/coordinator/analyzer.go`) - Task classification, deduplication
- **Executors** (`internal/executor/`) - Claude Code CLI, Gemini CLI
- **Store** (`internal/coordinator/store_sqlite.go`) - Neutral storage layer

**Future: Cloud Storage**
The storage layer is designed for cloud backends (Firestore, DynamoDB, etc.). Currently uses SQLite locally, but the neutral `Store` interface enables future cloud deployment without code changes.

**For complete guide**: See [docs/docs/guides/coordinator.md](docs/docs/guides/coordinator.md)

### Chain Execution Monitoring (v0.7.2+)

**`ailang chains` is the canonical CLI for examining all past and current executions.** Works offline (direct SQLite), no server required.

```bash
# LIST CHAINS
ailang chains list                       # List all chains (most recent first)
ailang chains list --status active       # Filter by status
ailang chains list --source github_issue # Filter by source type
ailang chains list --agent design-doc-creator  # Filter by agent (v0.8.1+)
ailang chains list --since 24h           # Created after (24h, 7d, 2026-02-01)
ailang chains list --agent sprint-executor --since 7d --json
ailang chains list --limit 50            # More results
ailang chains list --full                # Show full chain IDs
ailang chains list --json                # JSON output

# ACTIVE CHAINS (shortcut)
ailang chains active                     # Currently running chains

# VIEW CHAIN DETAILS
ailang chains view <chain-id>            # Chain + stages overview
ailang chains view <chain-id> --spans    # Include session/tool details per stage
ailang chains view <chain-id> --json     # JSON output

# TREE VIEW
ailang chains tree <chain-id>            # ASCII tree of chain hierarchy
ailang chains tree <chain-id> --detailed # With session + tool timeline per stage

# VIEW CHANGES (v0.8.1+)
ailang chains diff <chain-id>            # Git diff across all stages
ailang chains diff <chain-id> --stat     # Diffstat summary only

# FIND CHAIN BY REFERENCE (v0.8.1+)
ailang chains find --task-id <task-id>          # Find by coordinator task
ailang chains find --message-id <uuid>          # Find by message
ailang chains find --github owner/repo#123      # Find by GitHub issue
ailang chains find --github owner/repo#123 --json

# COST & TOKEN STATS
ailang chains stats                      # All-time cost/token summary
ailang chains stats --hours 168          # Last week
ailang chains stats --by-agent           # Breakdown by agent
ailang chains stats --json               # JSON output

# DIAGNOSTICS
ailang chains diagnose <chain-id>        # Quick health report for specific chain
ailang chains health                     # System-wide data capture validation

# INTERACTIVE MODE
ailang chains                            # Interactive menu (in terminal)
```

**Short chain IDs:** Use first 8+ characters (like git) — `ailang chains view e9c7501d`

**Dashboard vs Chains:**
- **`ailang chains`**: Chain-centric, execution flow, works offline (direct SQLite)
- **`ailang dashboard`**: Span-centric, provider/model telemetry, requires running server
- Both are complementary — different abstraction levels, neither is deprecated

**REST API equivalents (require `ailang serve` running):**

| CLI Command | API Equivalent |
|------------|----------------|
| `ailang chains list` | `GET /api/chains` |
| `ailang chains list --agent X --since Y` | `GET /api/chains?agent_id=X&since=Y` |
| `ailang chains active` | `GET /api/chains/active` |
| `ailang chains view <id>` | `GET /api/chains/{id}` |
| `ailang chains stats` | `GET /api/chains/stats?hours=720&by_agent=true` |
| `ailang chains find --message-id <id>` | `GET /api/chains/by-message/{id}` |
| `ailang chains find --task-id <id>` | `GET /api/chains/by-task/{id}` |
| `ailang chains find --github owner/repo#N` | `GET /api/chains/by-github/{owner}/{repo}/{number}` |
| (pending) | `GET /api/chains/pending` |

### Auditing Agent Work

After a coordinator task completes, audit what the agent actually did using these commands:

**View chain execution flow (recommended first step):**
```bash
ailang chains view <chain-id> --spans    # See full execution with sessions + tools
ailang chains tree <chain-id> --detailed # Tree view with tool timeline
ailang chains diagnose <chain-id>        # Quick health check
```

**View per-turn conversation text (reasoning + tool calls):**
```bash
# Raw log stream (quick look)
ailang coordinator logs <task-id> --limit 500

# Grouped by turn (shows reasoning per turn with tool names)
ailang coordinator logs <task-id> --limit 1000 --json | python3 -c "
import json, sys
data = json.load(sys.stdin)
events = data.get('events', [])
turns = {}; tools = {}
for evt in events:
    tn = evt.get('turn_num', 0); st = evt.get('stream_type', '')
    if st == 'text': turns.setdefault(tn, []).append(evt.get('text', ''))
    elif st == 'tool_use': tools.setdefault(tn, []).append(evt.get('tool_name', '?'))
for tn in sorted(turns.keys()):
    text = ''.join(turns[tn]).strip()
    if len(text) > 20:
        print(f'=== Turn {tn} (tools: {\", \".join(tools.get(tn, []))}) ===')
        print(text[:600]); print()
"
```

**View tool usage timeline and spans:**
```bash
ailang chains view <chain-id> --spans                           # Chain-level tool usage (offline)
ailang dashboard spans --task-id <task-id> --limit 200          # Span-level timeline (requires server)
ailang dashboard spans --task-id <task-id> --include-chat --json # With chat context (if available)
```

**View git changes made by the agent:**
```bash
ailang coordinator diff <task-id>
```

**Key things to check when auditing:**
- **Model used**: Check `executor.model` attribute in spans - Haiku may be too weak for compiler tasks
- **Turn count & cost**: `ailang chains stats --by-agent` for cost overview; high turns + low cost = Haiku
- **Code changes**: Did `coordinator diff` show changes to `internal/` or just docs/examples?
- **Skipped tasks**: Did the agent say "already works" without fixing the specific reported bug?
- **Runtime vs compile**: Did it test with `ailang run` (runtime) or just `ailang check` (compile)?

**Per-agent model config (v0.8.0+):**
Agents can specify which Claude model to use in `~/.ailang/config.yaml`:
```yaml
coordinator:
  agents:
    - id: sprint-executor
      model: opus          # Use Opus 4.5 for complex coding tasks
    - id: design-doc-creator
      model: opus          # Use Opus for design documents
    - id: cheap-agent
      # model not set = executor default (haiku)
```

### Adding Builtin Functions

**To add a builtin function, use the `builtin-developer` skill.**

**Quick Reference:**
- **Development time**: ~2.5 hours (down from 7.5h with legacy system)
- **Status**: M-DX1 COMPLETE - 72 builtins, fully documented
- **Key benefit**: 67% faster with single-file registration

**New in v0.5.11 (M-DX15):**
- `_bytes_from_string`, `_bytes_to_string` - UTF-8 encoding/decoding
- `_bytes_to_base64`, `_bytes_from_base64` - Base64 encoding/decoding
- `_bytes_length` - Byte slice length
- `_simhash`, `_hamming_distance` - Locality-sensitive hashing for near-duplicate detection
- `_int_to_float`, `_float_to_int` - Numeric conversions (for JSON encoding)

**Validation commands:**
```bash
ailang doctor builtins              # Validate all 72 builtins
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

### M-EVAL-LOOP: AI Evaluation & Self-Improvement (✅ COMPLETE - v3.0)

**When user asks about evaluations, benchmarks, or testing AI code generation:**

→ **Use the `eval-analyzer` skill or `ailang eval-*` commands**

**Two eval modes:**
- **Standard** (0-shot API): Fast, cheap, file-based results
- **Agent** (agentic CLI coding): Multi-turn, tool-using, chain-based results (v0.8.0+)

Common workflows:
```bash
# Standard eval (0-shot API)
ailang eval-suite --models MODEL1,MODEL2
ailang eval-compare baseline1 baseline2
ailang eval-report results/ VERSION --format=json

# Agent eval (agentic coding via Claude/Gemini CLI)
ailang eval-suite --agent --models claude-haiku-4-5,gemini-2-5-flash
ailang eval-chains list                           # List recent eval chains
ailang eval-chains view <chain-id>                # Per-benchmark assessment
ailang eval-chains stats <chain-id>               # Pass rate by model/language
ailang eval-chains failures <chain-id>            # Failures with error details
ailang eval-report --from-chain <chain-id> v0.8.0 # Report from chain
ailang eval-compare --chain <id1> --chain <id2>   # Compare two chains
```

- Analyzing failures: Use `eval-analyzer` skill

**Documentation** (for detailed reference):
- [Architecture Overview](docs/docs/guides/evaluation/architecture.md) - Commands & workflows
- [Evaluation README](docs/docs/guides/evaluation/README.md) - Quick start guide

**⚠️ CRITICAL: Running Multiple Models (Standard Eval)**

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

# ✅ Resume interrupted runs without losing progress
ailang eval-suite --full --skip-existing  # Skips benchmarks with existing results
```

**Note:** Agent eval (`--agent`) stores results in `observatory.db` chains, not files — no overwrite risk.

**Default model sets:**
- `ailang eval-suite` → Reads from `dev_models` in models.yml (currently: gpt5-mini, claude-haiku-4-5, gemini-2-5-flash)
- `ailang eval-suite --full` → Reads from `extended_suite` in models.yml (all 6 models: gpt5, gpt5-mini, claude-sonnet-4-5, claude-haiku-4-5, gemini-2-5-pro, gemini-2-5-flash)

**Quick reference - Common eval commands:**
```bash
# Standard eval - update benchmark dashboard (PRESERVES HISTORY!)
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json

# Agent eval - report from chain
ailang eval-report --from-chain <chain-id> v0.8.0 --format=json
ailang eval-report --from-latest-chain v0.8.0

# Run baseline (REQUIRES explicit version!)
make eval-baseline EVAL_VERSION=v0.3.10              # Uses dev models (3 cheap models)
make eval-baseline EVAL_VERSION=v0.3.10 FULL=true    # Uses ALL 6 models (extended_suite)

# Compare two baselines
ailang eval-compare eval_results/baselines/v0.3.8 eval_results/baselines/v0.3.9
ailang eval-compare --chain <id1> --chain <id2>      # Chain-based comparison

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

**Quick reference table (environment variables):**

| Flag | Purpose | Use When |
|------|---------|----------|
| `DEBUG_STRICT=1` | Fail loudly on unhandled cases | Development, CI |
| `DEBUG_MONO_VERBOSE=1` | Monomorphization tracing | Type issues |
| `DEBUG_OPERATOR_LOWERING=1` | Operator resolution | Dispatch issues |
| `DEBUG_PARSER=1` | Token position tracing | Parser bugs |
| `DEBUG_CODEGEN=1` | Warn on record type fallback | Record compiles to map instead of struct |
| `DEBUG_APPROVAL_WATCHER=1` | Verbose ApprovalWatcher polling | GitHub label detection issues |

**CLI flags for `ailang check` (v0.5.9+):**

| Flag | Purpose | Use When |
|------|---------|----------|
| `--timeout 30s` | Compilation timeout with stack dump | Detecting cyclic type hangs |
| `--debug-compile` | Show phase timing breakdown | Performance analysis |

**Recommended combinations:**
```bash
# Development mode
DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang run test.ail

# CI mode
DEBUG_STRICT=1 make test

# Parser debugging
DEBUG_PARSER=1 ailang run test.ail

# Detect compilation hangs (v0.5.9+)
ailang check --timeout 30s file.ail

# Analyze which phase is slow
ailang check --debug-compile file.ail

# Coordinator GitHub label debugging (v0.6.2+)
DEBUG_APPROVAL_WATCHER=1 ailang coordinator start
ailang coordinator watcher-status  # Check watcher state
```

**For detailed documentation**: See [docs/guides/debugging.md](docs/guides/debugging.md)

### Telemetry & Trace Debugging (v0.6.3+)

**Use the `trace-debugger` skill for performance analysis and debugging with distributed traces.**

**Quick Start:**
```bash
# Check telemetry configuration
ailang trace status

# List recent traces (requires GOOGLE_CLOUD_PROJECT set)
ailang trace list --hours 1 --limit 10

# View trace hierarchy with timing
ailang trace view <trace-id>

# Filter by operation type
ailang trace list --filter "compile"
ailang trace list --filter "eval.suite"
```

**When to use traces vs debug flags:**

| Issue Type | Use Traces | Use Debug Flags |
|------------|------------|-----------------|
| Slow compilation | ✅ `ailang trace list --filter compile` | `--debug-compile` for phase timing |
| Hang/infinite loop | ✅ Trace shows where it stopped | `--timeout 30s` for stack dump |
| Type inference | 🔜 Future (`types.unify` span) | `DEBUG_MONO_VERBOSE=1` |
| Eval performance | ✅ `eval.suite`, `eval.benchmark` spans | N/A |
| Codegen fallbacks | 🔜 Future (`codegen.*` spans) | `DEBUG_CODEGEN=1` |

**Instrumented components:**
- Compiler pipeline (`compile.parse`, `compile.typecheck`, etc.)
- Eval harness (`eval.suite`, `eval.benchmark`)
- Messaging (`messages.send`, `messages.list`, `messages.search`)
- AI providers (`anthropic.generate`, `openai.generate`, etc.)

**Important limitation:** Traces only cover AILANG tooling, NOT generated Go code runtime.

**For detailed patterns**: Use the `trace-debugger` skill or see [docs/docs/guides/telemetry.md](docs/docs/guides/telemetry.md)

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

**Three prompt types** for different audiences:

| Command | Purpose | Size |
|---------|---------|------|
| `ailang prompt` | Full language syntax reference (0-shot generation) | ~1600 lines |
| `ailang devtools-prompt` | Toolchain/CLI reference (debugging, testing, tracing) | ~600 lines |
| `ailang agent-prompt` | Minimal agent coding guide (iterative write/check/run/fix) | ~180 lines |

**Get prompts:**
```bash
ailang prompt                           # Full syntax reference
ailang devtools-prompt                  # Toolchain reference
ailang agent-prompt                     # Minimal agent coding guide
ailang prompt --list                    # List all versions (works for all three)
```

- All prompts are version-locked and tracked in `prompts/*/versions.json`
- Validated through multi-model testing (Claude, GPT, Gemini)
- Share the same embedded FS (`//go:embed all:prompts`)

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
- **Relaxed Module Matching (v0.5.2):** Allow module declaration to mismatch file path
  - **Default behavior:** Strict - module path must match canonical file path (MOD010 validation)
  - **CLI flag:** `ailang run --relax-modules --caps IO --entry main file.ail`
  - **Environment variable:** `AILANG_RELAX_MODULES=1 ailang run ...`
  - **Auto-relaxation:** Files in temp directories (`/tmp/`, `/var/folders/`) auto-relax with warning
  - **Use case:** Quick prototyping, AI-generated temp files, experimentation
  - **Warning:** Relaxed mode emits warnings - check your module paths before packaging
- **Nullary functions (v0.4.6+):** `getArgs()`, `readLine()`, `clockNow()` all work correctly
  - Fixed in M-DX10 (v0.4.6) — call with empty parens: `let args = getArgs()`
  - `readLine()` uses persistent buffered reader — works correctly with piped stdin (v0.8.0+)
- **Type parameters:** Use `[T]` NOT `(T)` - distinguishes type/term application
- **Match in blocks:** Known parser bug (nested delimiter tracking) - extract to helper function

**For detailed syntax, limitations, and examples:**
- Use `ailang prompt` for full syntax reference
- Use `ailang devtools-prompt` for toolchain/CLI reference
- Use `ailang agent-prompt` for minimal iterative coding guide
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

**For refactoring**: Use `codebase-organizer` skill
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

### ast.Type Switch Exhaustiveness - ALL 8 Variants Must Be Handled!

**⚠️ CRITICAL LESSON (Feb 2026): Silent `default` cases in ast.Type switches corrupt ALL imported polymorphic types!**

**The Constructor Type Variable Bug (Feb 2026):**

Three separate functions convert `ast.Type` to `types.Type`:
1. `iface/builder.go:astTypeToInternalType()` — builds interface constructor schemes from imported ADTs
2. `types/typechecker.go:astTypeToType()` — converts type annotations during type checking
3. `elaborate/file.go:astTypeToInternalType()` — converts types during elaboration

**Two of these had silent fallbacks** that returned `TVar2{Name: "unknown"}` for unrecognized variants:
```go
// ❌ WRONG - Silently corrupts ALL imported ADT schemes
default:
    return &types.TVar2{Name: "unknown"}
```

**Impact:** Constructor field type variables (`a` in `Ok(a)`) are `*ast.TypeVar`, NOT `*ast.SimpleType`. The missing case caused `Ok`/`Err` constructors to get `unknown` type parameters instead of proper `a`/`e` variables. This broke ALL cross-module ADT pattern matching (Result, Option, etc.) with 6 different bug symptoms.

**The 8 ast.Type variants (ALL must be handled):**

| Variant | Example | Common conversion |
|---------|---------|-------------------|
| `*ast.SimpleType` | `int`, `string`, `Result` | `TCon` or `TVar2` (lowercase = type var) |
| `*ast.TypeVar` | `a`, `e` (type parameters) | `TVar2` |
| `*ast.FuncType` | `(int) -> string` | `TFunc` |
| `*ast.ListType` | `[int]` | `TList` |
| `*ast.ArrayType` | `Array[int]` | `TApp(Array, [int])` |
| `*ast.TupleType` | `(int, string)` | `TTuple` |
| `*ast.TypeApp` | `Result[a, e]` | `TApp` |
| `*ast.RecordType` | `{name: string}` | `TRecord` |

**Rules:**
1. **When adding a new `ast.Type` variant**, grep for ALL type switches: `grep -rn "case \*ast\." internal/`
2. **`default:` in ast.Type switches MUST `panic()`**, never return fake data
3. **Audit pattern**: Run `grep -rn "astTypeToInternalType\|astTypeToType" internal/` to find all converters

**Files:**
- `internal/iface/builder.go` — Fixed: added TypeVar, TypeApp cases + panic default
- `internal/types/typechecker.go` — Fixed: added TypeVar, RecordType, TypeApp cases + panic default
- `internal/elaborate/file.go` — Already correct (had all 8 variants)
- `internal/iface/constructor_test.go` — Regression tests for all variant conversions

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

### Claude Code CLI - TRACEPARENT is NOT Propagated!

**⚠️ KNOWN LIMITATION: Claude Code does NOT propagate TRACEPARENT to subprocess environments.**

When Claude Code runs a Bash tool (e.g., `ailang check`), the child process does NOT
receive the trace context. Child spans are created in a DIFFERENT trace entirely.

**This is a KNOWN limitation. DO NOT attempt to fix it at runtime. DO NOT "discover" it again.**

**What happens:**
```
claude.execute (trace_id: abc123)
  ├── exec.turn (trace_id: abc123)
  ├── exec.tool_use: Bash "ailang check ..." (trace_id: abc123)
  │     └── [EXPECTED: child spans here]
  │
  └── [ACTUAL: no children - child spans are in a different trace]

ailang.check (trace_id: xyz789)  ← Different trace!
  └── compile.parse
  └── compile.typecheck
```

**Solution: Timestamp Correlation (Virtual Re-parenting)**

We correlate spans at query time using timestamps, not runtime trace linking:
- Implementation: `internal/observatory/hierarchy.go:applyTimestampCorrelation()`
- Tests: `internal/observatory/hierarchy_test.go:TestTimestampCorrelation*`
- Works for spans that ARE in the same trace (e.g., `exec.tool_use` ↔ `ailang.run` from script agents)

**When timestamp correlation WORKS:**
- Script-based agents (`invoke.type: script`) - TRACEPARENT is properly propagated
- Direct `ailang exec` calls - parent sets TRACEPARENT in environment

**When timestamp correlation CANNOT help:**
- Claude Code tool execution - child spans are in different traces
- Gemini CLI tool execution - same limitation

**DO NOT:**
- ❌ Try to inject TRACEPARENT into Claude Code's subprocess environment (we don't control it)
- ❌ Attempt runtime fixes for this (Claude Code controls subprocess spawning)
- ❌ Spend time re-investigating this - it's documented and accepted

**Workaround for full hierarchy tracking:**
- Use `task_id` and `parent_task_id` attributes instead of trace_id
- Query spans by task_id for cross-trace linking
- The coordinator sets `AILANG_PARENT_TASK_ID` environment variable

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

## 🔗 ID Relationships Across Databases

**AILANG uses THREE SQLite databases with cross-references. Understanding ID relationships is critical.**

### Database Locations

| Database | Location | Purpose |
|----------|----------|---------|
| **observatory.db** | `~/.ailang/state/observatory.db` | OTEL spans, traces, task/agent hierarchy |
| **coordinator.db** | `~/.ailang/state/coordinator.db` | Task execution state, approvals, events |
| **collaboration.db** | `~/.ailang/state/collaboration.db` | Messages, threads, agents |

### ID Flow Diagram

```
COLLABORATION.DB                    COORDINATOR.DB                     OBSERVATORY.DB
═══════════════                     ══════════════                     ══════════════
threads.id ─────────────────────────► tasks.thread_id
     │
messages.id (UUID) ─────────────────► tasks.message_id
     │                                     │
     │                                     ├── tasks.id = "task-" + first_8(message_id)
     │                                     │        │
     │                                     │        └────────────────────► tasks.id (same)
     │                                     │                                    │
     │                                     │   tasks.session_id ◄───────────────│── spans.attributes.session.id
     │                                     │                                    │
     │                                     │                               agent_assignments.id
     │                                     │                                    │
     │                                     │                               spans.id ◄── span_id (16-char hex)
     │                                     │                                    ├── trace_id (32-char hex)
     │                                     │                                    ├── parent_span_id
     │                                     │                                    ├── task_id (FK)
     │                                     │                                    └── agent_assignment_id (FK)
```

### ID Formats

| ID Type | Format | Example | Generated By |
|---------|--------|---------|--------------|
| `task_id` | `task-<8-char-uuid>` | `task-29404032` | Coordinator (from message UUID) |
| `message_id` | Full UUID | `29404032-74b3-40c6-acc3-23d6bbe14b68` | Collaboration.db |
| `thread_id` | `thread_<ts>_<rand>` | `thread_1768064360112_5f96886d` | Collaboration.db |
| `workspace_id` | `ws_<16-char-hash>` | `ws_a1b2c3d4e5f67890` | SHA256(path) |
| `agent_assignment_id` | `aa_<16-char-hash>` | `aa_f1e2d3c4b5a69870` | SHA256(task+agent+time) |
| `span_id` | 16-char hex | `0f9632b58df815e4` | OTEL SDK |
| `trace_id` | 32-char hex | `0ebf5e64bb654fcc1d19256b59f05ae3` | OTEL SDK |
| `session.id` | UUID | `4df60536-caed-4e2f-af2c-e386c361f4e7` | Claude Code/Gemini CLI |
| `chain_id` | UUID | `e9c7501d-3f6e-48cd-8900-27a2b91d5345` | Observatory (execution_chains table) |
| `stage_id` | UUID | `a1b2c3d4-5678-9abc-def0-123456789abc` | Observatory (chain_stages table) |

### How Spans Link to Tasks

1. **Parent-child (OTEL standard):** `spans.parent_span_id` → builds tree within a trace
2. **Task linkage:** `spans.task_id` → directly references coordinator task
3. **Session correlation:** `spans.attributes.session.id` → links Claude Code sessions
4. **Assignment linkage:** `spans.agent_assignment_id` → links to agent that created span

### spans.task_id Formats (CRITICAL!)

**The `spans.task_id` column can have MULTIPLE formats depending on the source!**

| Format | Example | Source | Count (typical) |
|--------|---------|--------|-----------------|
| `task-<8char>` | `task-29404032` | Coordinator-executed tasks | ~3,000 spans |
| `eval-<timestamp>` | `eval-1768073642020901000` | Eval harness runs | ~100 spans |
| Full UUID | `013aca1d-1475-4fdf-a04b-aec531ed6296` | Claude Code direct sessions | ~25,000 spans |
| Other | varies | Legacy/custom | ~300 spans |

**Why this matters:**
- Filtering by `task_id LIKE 'task-%'` only matches coordinator tasks
- Claude Code sessions use UUID as their task_id (from session ID)
- Eval runs use `eval-<timestamp>` format
- Dashboard filtering must handle ALL formats, not just coordinator format

**Database reality check:**
```bash
sqlite3 ~/.ailang/state/observatory.db "
SELECT
  CASE
    WHEN task_id LIKE 'task-%' THEN 'coordinator'
    WHEN task_id LIKE 'eval-%' THEN 'eval'
    ELSE 'other/uuid'
  END as format,
  COUNT(*)
FROM spans WHERE task_id IS NOT NULL AND task_id <> ''
GROUP BY format"
```

### Common Pitfalls

**❌ DON'T** assume span.task_id is always set - orphan spans from CLI tools may not have it
**❌ DON'T** confuse trace_id with task_id - trace_id groups OTEL spans, task_id links to coordinator
**❌ DON'T** expect TRACEPARENT propagation through Claude Code - it's not supported (see CRITICAL WARNINGS)

**✅ DO** use `session.id` attribute for cross-trace correlation
**✅ DO** use `parent_span_id` for hierarchy within a trace
**✅ DO** use timestamp correlation for linking spans from different traces

### Querying Tips

```bash
# Find spans by task
sqlite3 ~/.ailang/state/observatory.db \
  "SELECT id, name, trace_id FROM spans WHERE task_id = 'task-29404032'"

# Find message that created a task
sqlite3 ~/.ailang/state/coordinator.db \
  "SELECT message_id, thread_id FROM tasks WHERE id = 'task-29404032'"

# Find all turns in a session
sqlite3 ~/.ailang/state/observatory.db \
  "SELECT id, json_extract(attributes, '\$.turn.number') FROM spans \
   WHERE json_extract(attributes, '\$.session.id') = 'UUID'"
```

### Dashboard Architecture Lesson (CRITICAL!)

**LESSON LEARNED FROM 12+ FAILED ATTEMPTS (January 2026):**

When implementing filtering in the dashboard, **use the same data source as other views**.

**The pattern that WORKS (Tree, Timeline, Chat, Waterfall views):**
```
Event selected → spans fetched → `spans` prop passed → view renders only those spans
```

**The pattern that FAILS (TaskHierarchyGraph before fix):**
```
Event selected → separate API call to coordinator.db → ID format mismatch → shows all/nothing
```

**Root cause:** The dashboard has TWO databases with different ID formats:
- `observatory.db` → spans with trace_id, various task_id formats
- `coordinator.db` → tasks with `task-<8char>` format only

**Attempting to cross-reference these databases via ID matching is fragile because:**
1. Only ~10% of events have matching coordinator tasks
2. Claude Code sessions use UUID task_ids (not in coordinator.db)
3. Eval runs use `eval-<timestamp>` task_ids (not in coordinator.db)
4. trace_id doesn't link to coordinator.db at all

**The fix:** Make TaskHierarchyGraph accept and use the `spans` prop like other views. Transform spans into graph format client-side rather than making separate API calls.

**Design doc:** See [design_docs/planned/v0_7_0/m-task-graph-spans-unification.md](design_docs/planned/v0_7_0/m-task-graph-spans-unification.md)

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
