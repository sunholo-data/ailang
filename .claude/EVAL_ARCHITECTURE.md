# M-EVAL-LOOP Architecture (v2.0)

## Overview

Clean two-tier architecture: **Native Go commands** + **Smart agents**

```
User Input ("validate my fix")
    ↓
Smart Agent (eval-orchestrator)
    ↓
Native Go Commands (ailang eval-validate)
    ↓
Results + Interpretation
```

## Components

### Tier 1: Native Go Commands (Fast & Reliable)

Location: `internal/eval_analysis/` + `cmd/ailang/`

```bash
ailang eval-compare <baseline> <new>        # Compare two runs
ailang eval-matrix <dir> <version>          # Generate performance matrix
ailang eval-summary <dir>                   # Export to JSONL
ailang eval-validate <benchmark> [version]  # Validate specific fix
ailang eval-report <dir> <version> [format] # Generate reports (MD/HTML/CSV)
```

**Characteristics:**
- ⚡ Fast (native Go, no bash/jq overhead)
- ✅ Type-safe (90%+ test coverage)
- 🔧 Maintainable (2,070 LOC with tests)
- 🪟 Cross-platform
- 🤖 CI/CD friendly (proper exit codes)

### Tier 2: Smart Agents (Intelligence Layer)

#### A. eval-orchestrator (.claude/agents/eval-orchestrator.md)

**Role**: Intelligent workflow orchestration

**What it does:**
- Interprets user intent ("How's AILANG doing?")
- Maps to appropriate `ailang` command
- Runs command and interprets results
- Provides recommendations for next steps

**Example:**
```
User: "validate the float_eq fix"

Agent:
1. Understands: user wants to check if fix works
2. Chooses: ailang eval-validate float_eq
3. Runs command
4. Interprets: "✓ FIX VALIDATED: Benchmark now passing!"
5. Recommends: "Run full comparison? Update baseline?"
```

#### B. eval-fix-implementer (.claude/agents/eval-fix-implementer.md)

**Role**: Automated fix implementation

**What it does:**
- Reads design docs from `design_docs/planned/EVAL_ANALYSIS_*.md`
- Implements the proposed fix
- Runs tests
- Validates with `ailang eval-validate`
- Reports before/after comparison

**Example:**
```
User: "implement the float_eq fix from the design doc"

Agent:
1. Reads: design_docs/planned/EVAL_ANALYSIS_float_eq.md
2. Implements fix in internal/types/builtins.go
3. Runs: go test ./internal/types/
4. Validates: ailang eval-validate float_eq
5. Shows: ✓ FIX VALIDATED + before/after metrics
```

### Make Targets (Convenience Layer)

```bash
make eval-baseline    # Store baseline
make eval-suite       # Run all benchmarks
make eval-analyze     # Analyze failures → design docs
make eval-diff        # Calls: ailang eval-compare
make eval-matrix      # Calls: ailang eval-matrix
make eval-summary     # Calls: ailang eval-summary
```

## User Experience

### Natural Language (Recommended)

Users don't need to know command names:

```
✅ "validate my fix for records"
✅ "how is AILANG performing?"
✅ "generate a release report"
✅ "compare the baseline to current results"
```

Agent automatically:
- Chooses correct command
- Runs it with appropriate options
- Interprets results
- Suggests next steps

### Direct Commands (Power Users)

```bash
ailang eval-validate records_person
ailang eval-compare baselines/v0.3.0 current
ailang eval-report results/ v0.3.1 --format=html > report.html
```

### Make Targets (Workflows)

```bash
make eval-baseline              # Before starting work
make eval-suite                 # Full validation
make eval-diff BASELINE=... NEW=...
```

## Decision Flow

```
User Input
    ↓
Is it a question/request? ────YES───→ eval-orchestrator agent
    ↓                                    ↓
    NO (direct command)             Interprets intent
    ↓                                    ↓
Is it a make target? ──YES──→ Run make    ↓
    ↓                                Chooses command
    NO                                   ↓
    ↓                            ailang eval-* command
Run ailang command directly              ↓
                                  Interprets results
                                         ↓
                                  Provides recommendations
```

## Why This Architecture?

### 1. Separation of Concerns
- **Go layer**: Fast, tested, type-safe execution
- **Agent layer**: Intelligence, interpretation, recommendations

### 2. Flexibility
- Users can use natural language OR direct commands
- Agents add value (interpretation) without forcing specific syntax

### 3. Maintainability
- Native Go code is easy to test and debug
- Agents are pure routing logic (no complex bash)
- Clear boundaries between layers

### 4. Performance
- Native Go = 5-10x faster than old bash scripts
- No overhead from slash command parsing
- Direct execution path for power users

## Anti-Patterns (What We Removed)

### ❌ Slash Command Layer
**Removed**: `/eval-loop validate`

**Why removed:**
- Redundant with agents (eval-orchestrator does the same thing better)
- Forced users to learn special syntax
- Added no intelligence (just aliases)

**Better approach:**
```
# OLD (removed):
/eval-loop validate float_eq

# NEW (simpler):
"validate float_eq"  → agent handles it
# OR
ailang eval-validate float_eq  → direct command
```

### ❌ Bash Script Wrappers
**Removed**: `tools/eval_diff.sh`, `tools/generate_matrix_json.sh`, etc.

**Why removed:**
- Brittle (division by zero bugs)
- Slow (jq overhead)
- Untested
- Hard to maintain

**Better approach:**
Native Go implementation in `internal/eval_analysis/`

## File Organization

```
.claude/
  agents/
    eval-orchestrator.md     # Smart workflow router
    eval-fix-implementer.md  # Automated fix implementation
  commands/
    (eval-loop.md deleted)   # Removed: redundant with agents

internal/eval_analysis/      # Native Go implementation
  types.go                   # Data structures
  loader.go                  # Load results
  comparison.go              # Diff logic
  matrix.go                  # Aggregates
  formatter.go               # Terminal output
  validate.go                # Fix validation
  export.go                  # MD/HTML/CSV export
  *_test.go                  # Tests (90%+ coverage)

cmd/ailang/
  eval_tools.go              # CLI integration
  main.go                    # Command routing

tools/
  eval_baseline.sh           # Calls: ailang eval-matrix
  (bash wrappers deleted)    # Removed: redundant

Makefile                     # Convenience targets
```

## Migration from Old Architecture

### Before (v1.0)
```
User → /eval-loop command → Bash script → jq → Results
        (brittle, slow, untested)
```

### After (v2.0)
```
User → eval-orchestrator agent → ailang command → Go implementation → Results
        (smart, fast, tested)
```

**Benefits:**
- 5-10x faster
- Type-safe (no division by zero!)
- 90%+ test coverage
- Better UX (natural language)

## Success Metrics

This architecture succeeds when:
- [x] Users can speak naturally without learning syntax
- [x] Power users can use direct commands
- [x] All commands are fast and reliable
- [x] Easy to extend (add new features to Go package)
- [x] Clear separation between execution and intelligence

---

**Version**: 2.0
**Updated**: 2025-10-10
**Status**: Production Ready
