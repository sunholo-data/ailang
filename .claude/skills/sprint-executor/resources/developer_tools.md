# AILANG Developer Tools Quick Reference

**For sprint-executor skill** - Load this when you need to know which tools are available for specific development tasks.

## Core Development Tools

### Building & Installing

```bash
# Local build
make build                 # Build to bin/ailang

# System installation
make install               # Full install with version info (slow)
make quick-install         # Fast reinstall (recommended during dev)

# Development workflow
make dev                   # Quick development build
make watch                 # Watch mode (local build)
make watch-install         # Watch mode (auto-install to PATH)
```

**After code changes, always run:**
```bash
make quick-install && ailang run --caps IO examples/hello.ail
```

### Testing

#### Core Testing
```bash
make test                  # All tests (excludes scripts/)
make test-coverage         # Tests with coverage report (coverage.html)
make test-coverage-badge   # Quick coverage check (shows percentage)
make ci                    # Full CI verification locally
make ci-strict             # Extended CI with A2 milestone gates (pre-release)
```

#### Specialized Testing
```bash
# Parser
make test-parser           # Parser tests with golden files
make test-parser-update    # Update parser golden files after changes

# Import system
make test-imports          # Test import system (success + errors)
make test-import-errors    # Test import error goldens
make regen-import-error-goldens  # Regenerate import error goldens

# Operators
make test-lowering         # Test operator lowering (desugaring)
make test-operator-assertions    # Test operator desugaring assertions
make verify-lowering       # Verify operator lowering

# Type system
make test-row-properties   # Test row unification properties
make test-golden-types     # Test builtin type snapshots
make test-iface-determinism # Test interface determinism

# Builtins
make test-builtin-consistency # Test builtin three-way parity
make test-builtin-freeze   # Freeze builtin type signatures
make doctor                # Validate builtin registry

# Stdlib
make test-stdlib-canaries  # Test stdlib health (std/io, std/net)
make test-stdlib-freeze    # Verify stdlib interfaces haven't changed
make freeze-stdlib         # Generate SHA256 golden files for stdlib
make verify-stdlib         # Verify stdlib against golden hashes

# REPL
make test-repl-smoke       # REPL smoke tests (:type command)
make test-parity           # Test REPL/file parity (manual, requires interactive REPL)

# Regression & Guards
make test-regression-guards # Run regression guard tests
make test-recursion        # Test recursion limits
make verify-no-shim        # Verify no shim code exists

# Golden file management
make check-golden-drift    # Check for drift in golden files
```

#### Coverage & Quality Gates
```bash
# Coverage reports
make cover-lines           # Show parser line coverage
make cover-branch          # Open parser branch coverage HTML
make cover-lexer           # Lexer coverage
make cover-parser          # Parser coverage
make cover-all-packages    # Coverage across all packages

# Quality gates
make gate-lexer            # Lexer quality gates
make gate-parser           # Parser quality gates
make gate-all-packages     # Quality gates for all packages
```

### Code Quality

```bash
# Formatting
make fmt                   # Format all Go code (gofmt)
make fmt-check             # Check if code is formatted (CI)

# Linting
make lint                  # Run golangci-lint
make install-lint          # Install golangci-lint

# Static analysis
make vet                   # Run go vet
```

### File Size Management (AI-Friendly Codebase)

```bash
# Check file sizes
make check-file-sizes      # Fail if any file >800 lines (CI check)
make report-file-sizes     # Report all files >500 lines
make codebase-health       # Full codebase health metrics
make largest-files         # Show 20 largest files

# Target: 0 files over 800 lines, <5 files between 500-800 lines
# Use codebase-organizer agent to refactor large files
```

## Stdlib Development

### When Modifying Stdlib

```bash
# 1. Make changes to stdlib/std/*.ail

# 2. Verify interfaces haven't broken
make verify-stdlib         # Checks SHA256 hashes match golden files

# 3. If intentional breaking change
make freeze-stdlib         # Update golden hashes

# 4. Run health checks
make test-stdlib-canaries  # Tests std/io, std/net work
make test-stdlib-freeze    # Verify stdlib interface freeze
```

### Stdlib Tools

```bash
# Output normalized JSON interface
ailang iface stdlib/std/io.ail    # See module interface

# Freeze/verify workflow
tools/freeze-stdlib.sh            # Generate SHA256 golden files
tools/verify-stdlib.sh            # Verify against goldens
```

## Golden File Testing

**What are golden files?** Pre-recorded expected outputs for deterministic testing.

### When Parser/Types Change

```bash
# 1. Run tests
make test-parser

# 2. If expected failures (e.g., parser produces new AST format)
make test-parser-update    # Update golden files

# 3. Review changes
git diff tests/golden/

# 4. Check for drift
make check-golden-drift
```

### Other Golden File Targets

```bash
make test-golden-types     # Test builtin type snapshots
make test-lowering         # Test operator lowering goldens
make regen-import-error-goldens  # Regenerate import error goldens
```

## Evaluation & Benchmarking

### Running Evals

```bash
# Development (cheap models: gpt5-mini, claude-haiku, gemini-flash)
make eval-suite
ailang eval-suite          # Same as above

# Full suite (all 6 models: GPT-5, GPT-5-mini, Claude Sonnet/Haiku, Gemini Pro/Flash)
make eval-suite FULL=true
ailang eval-suite --full

# Baseline for release (REQUIRES version!)
make eval-baseline EVAL_VERSION=v0.3.15              # Dev models
make eval-baseline EVAL_VERSION=v0.3.15 FULL=true    # All models

# Resume interrupted baseline (v0.3.14+)
ailang eval-suite --full --skip-existing  # Skip benchmarks with existing results
```

**CRITICAL**: `ailang eval-suite` OVERWRITES output directory by default!
```bash
# ❌ WRONG - Second run overwrites first
ailang eval-suite --models gpt5
ailang eval-suite --models claude-sonnet-4-5  # DELETES gpt5 results!

# ✅ CORRECT - Run all models in ONE command
ailang eval-suite --models gpt5,claude-sonnet-4-5,gemini-2-5-pro
```

### Comparing & Reporting

```bash
# Compare two baselines
ailang eval-compare eval_results/baselines/v0.3.14 eval_results/baselines/v0.3.15

# Generate comprehensive report
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=markdown

# Update dashboard JSON (preserves history!)
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=json
# ✅ Automatically writes to docs/static/benchmarks/latest.json with history

# Performance matrix
ailang eval-matrix eval_results/baselines/v0.3.15 v0.3.15

# Summary
ailang eval-summary eval_results/baselines/v0.3.15
```

### Advanced Eval Workflows

```bash
# Validate specific fix
ailang eval-validate fizzbuzz eval_results/baselines/v0.3.14
# Runs just fizzbuzz against baseline to verify fix works

# A/B test prompts
make eval-prompt-ab PROMPT_A=v0.3.8 PROMPT_B=v0.3.9
tools/eval_prompt_ab.sh v0.3.8 v0.3.9

# Auto-improve prompts
make eval-auto-improve          # Analyzes failures, suggests improvements
make eval-auto-improve-apply    # Applies suggestions

# Full workflow: evals → analysis → design docs
make eval-to-design
tools/eval-to-design.sh
```

### Eval Analysis

```bash
# Analyze eval failures and generate design docs
make eval-analyze              # With deduplication
make eval-analyze-fresh        # Force new design docs (disable dedup)
ailang eval-analyze eval_results/ --output design_docs/planned/

# List available models
make eval-models
ailang eval-suite --list-models
```

### Eval Cleanup

```bash
make eval-clean                # Remove eval_results/
```

## Example Management

```bash
# Verify examples
make verify-examples           # Verify all working examples
make verify-examples-all       # Verify all examples (including broken)
make examples-status           # Show example status

# Update documentation
make update-readme             # Update README with example status
make flag-broken               # Add warning headers to broken examples

# Audit examples
tools/audit-examples.sh        # Comprehensive example audit
```

## Documentation

### Documentation Site (Docusaurus)

```bash
# Install dependencies
make docs-install              # cd docs && npm install

# Development
make docs-serve                # Start dev server (http://localhost:3000)
make docs                      # Alias for docs-serve
make docs-preview              # Preview production build

# Build
make docs-build                # Build for production

# Troubleshooting
make docs-clean                # Clear Docusaurus cache
make docs-restart              # Clear cache + restart dev server

# Manual cache clearing (for webpack chunk errors)
cd docs && npm run clear && rm -rf .docusaurus build && npm start
```

### Documentation Sync

```bash
# Sync prompts to docs site (Docusaurus)
make sync-prompts              # Copies prompts/ to docs/docs/prompts/
docs/scripts/sync-prompts.sh   # (Active prompt + recent versions)

# Generate LLM-friendly context
make generate-llms-txt         # Creates docs/static/llms.txt
tools/generate-llms-txt.sh
```

### Update Benchmark Dashboard

```bash
# Generate dashboard files (markdown + JSON with history)
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=docusaurus 2>/dev/null > docs/docs/benchmarks/performance.md

# Update JSON (preserves history automatically!)
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=json
# Writes to docs/static/benchmarks/latest.json

# Verify JSON is valid
jq -r '.version, .aggregates.finalSuccess' docs/static/benchmarks/latest.json

# Clear cache and test
cd docs && npm run clear && npm start
```

## WASM

```bash
make build-wasm                # Build WASM binary for browser REPL
# Output: docs/static/wasm/ailang.wasm
```

## Fuzz Testing

```bash
make fuzz-parser               # Fuzz test parser (short run)
make fuzz-parser-long          # Long-running fuzz test
```

## Debug & Introspection

### Debug AST Command

**NEW in v0.3.16**: Inspect Core AST (ANF) and inferred types

```bash
# Show Core AST with node IDs
ailang debug ast example.ail

# Show Core AST with inferred types
ailang debug --show-types ast example.ail

# Compact output (no indentation)
ailang debug --compact ast example.ail
```

**Example output** (`ailang debug --show-types ast concat.ail`):
```
=== Core AST (ANF) ===
Program:
  [0] Let(xs) [#13] :: [α7]:
    Value: List[3] [#4] :: [α1]:
      [0]: Lit(1) [#1] :: α1
      [1]: Lit(2) [#2] :: α2
      [2]: Lit(3) [#3] :: α3
    Body:  Let(ys) [#12] :: [α7]:
      Value: List[3] [#8] :: [α4]:
        [0]: Lit(4) [#5] :: α4
        [1]: Lit(5) [#6] :: α5
        [2]: Lit(6) [#7] :: α6
      Body:  Intrinsic(11) [#11] :: [α7]:
        Arg[0]: Var(xs) [#9] :: [int]
        Arg[1]: Var(ys) [#10] :: [int]
```

**Use cases**:
- Debug operator lowering (see which types were inferred for operators)
- Understand ANF transformations (see Let bindings, Intrinsics)
- Validate type inference results (verify CoreTypeInfo is populated correctly)
- Learn AILANG internals (see how surface syntax becomes Core AST)
- Investigate type errors (see where inference stopped)

**What you see**:
- `[#N]` - Node ID (used by CoreTypeInfo for type-guided lowering)
- `:: Type` - Inferred type from type checker (when --show-types used)
- `Intrinsic(N)` - Operator intrinsics (e.g., 11 = OpConcat for `++`)
- `Let` bindings - ANF explicit sequencing
- `VarGlobal` - Builtin function references

## Utilities

```bash
# Dependencies
make deps                      # Download Go dependencies (go mod download)

# Cleanup
make clean                     # Remove build artifacts and coverage files

# Help
make help                      # Show all available make targets
make help-release              # Show release workflow
```

## Skills (Auto-Invoked)

**Don't call these manually - they're auto-invoked by Claude when appropriate.**

```bash
# Use via natural language:
"Ready to release v0.3.15"     → release-manager skill
"Update benchmarks"            → post-release skill
"Plan the next sprint"         → sprint-planner skill
"Execute the sprint plan"      → sprint-executor skill (YOU ARE HERE!)
"Create a new skill"           → skill-builder skill
"Create a design doc"          → design-doc-creator skill
"Help me write AILANG code"    → use-ailang skill
```

## Agents (Auto-Invoked)

**Don't call these manually - they're auto-invoked by Claude when appropriate.**

```bash
# Use via natural language:
"Run evals and compare results"        → eval-orchestrator agent
"Implement fix from eval failure"      → eval-fix-implementer agent
"Split this large file"                → codebase-organizer agent
"Sync docs with code changes"          → docs-sync-guardian agent
"Verify code matches design spec"      → design-spec-auditor agent
"Analyze test coverage"                → test-coverage-guardian agent
```

## Common Workflows

### During Sprint Execution

```bash
# After each code change
make quick-install             # Fast reinstall
make test                      # Verify tests pass
make lint                      # Verify linting passes

# At milestone checkpoints
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh "Milestone name"

# Before committing
make test                      # All tests must pass
make lint                      # All linting must pass
make check-file-sizes          # Verify no files >800 lines
```

### After Parser/Type System Changes

```bash
# 1. Run tests
make test-parser

# 2. If golden files need updating
make test-parser-update

# 3. Review changes
git diff tests/golden/

# 4. Verify other systems
make test-lowering             # Operator lowering
make test-row-properties       # Row unification
make test-golden-types         # Builtin types
```

### After Stdlib Changes

```bash
# 1. Verify no breaking changes
make verify-stdlib

# 2. If intentional breaking change
make freeze-stdlib

# 3. Run health checks
make test-stdlib-canaries
```

### Before Releasing

```bash
# Use the release-manager skill:
"Ready to release v0.3.15"

# Manual pre-release checks:
make ci-strict                 # Full CI with A2 gates
make check-file-sizes          # No files >800 lines
make verify-stdlib             # Stdlib unchanged (or frozen)
make test-coverage-badge       # Coverage check
```

### After Releasing

```bash
# Use the post-release skill:
"Update benchmarks for v0.3.15"

# Manual post-release:
make eval-baseline EVAL_VERSION=v0.3.15 FULL=true
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=json
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=docusaurus > docs/docs/benchmarks/performance.md
cd docs && npm run clear && npm start
```

## Tool Discovery Cheat Sheet

| I need to... | Tool |
|--------------|------|
| Build locally | `make build` |
| Install to system | `make quick-install` |
| Run all tests | `make test` |
| Run tests with coverage | `make test-coverage` |
| Update parser goldens | `make test-parser-update` |
| Verify stdlib unchanged | `make verify-stdlib` |
| Freeze stdlib after change | `make freeze-stdlib` |
| Run full CI | `make ci` or `make ci-strict` |
| Check file sizes | `make check-file-sizes` |
| Run evals (dev models) | `make eval-suite` |
| Run evals (all models) | `make eval-suite FULL=true` |
| Baseline for release | `make eval-baseline EVAL_VERSION=vX.Y.Z` |
| Compare baselines | `ailang eval-compare <dir1> <dir2>` |
| Validate specific fix | `ailang eval-validate <benchmark> <baseline>` |
| Update dashboard | `ailang eval-report <dir> <ver> --format=json` |
| Format code | `make fmt` |
| Lint code | `make lint` |
| Verify examples | `make verify-examples` |
| Debug AST and types | `ailang debug --show-types ast <file>` |
| Inspect Core AST | `ailang debug ast <file>` |
| Start docs server | `make docs-serve` |
| Clear docs cache | `make docs-clean` |
| Release new version | Use `release-manager` skill |
| Post-release tasks | Use `post-release` skill |
| Refactor large file | Use `codebase-organizer` agent |

## Emergency Recovery

### Tests Failing During Sprint

```bash
# 1. Show test output
make test

# 2. Options:
# (a) Fix now - implement fix
# (b) Revert change - git restore <file>
# (c) Pause sprint - commit WIP, resume later
```

### Linting Failing During Sprint

```bash
# 1. Try auto-fix
make fmt

# 2. If still failing, show output
make lint

# 3. Fix manually or ask for guidance
```

### Stdlib Accidentally Changed

```bash
# 1. Check what changed
make verify-stdlib

# 2. If unintentional
git restore stdlib/std/*.ail

# 3. If intentional
make freeze-stdlib
```

### Docusaurus Webpack Errors

```bash
# Nuclear option - clear everything
cd docs && npm run clear && rm -rf .docusaurus build && npm start
```

### Golden Files Out of Sync

```bash
# Parser goldens
make test-parser-update

# Import error goldens
make regen-import-error-goldens

# Stdlib goldens
make freeze-stdlib

# Review all changes
git diff tests/golden/ .stdlib-golden/
```

---

**Last updated**: 2025-10-21 for v0.3.16 development
**Maintained by**: DX Tools Documentation Initiative
**See also**:
- CLAUDE.md (comprehensive development guide)
- design_docs/planned/v0_3_16/dx-tools-documentation-audit.md (full audit)
- .claude/skills/README.md (skills overview)
