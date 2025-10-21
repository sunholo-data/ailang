# M-DX-TOOLS: Developer Tools Documentation Audit

**Status**: Draft
**Version**: v0.3.16
**Date**: 2025-10-21
**Author**: Claude (DX Audit)

## Executive Summary

AILANG has accumulated a rich ecosystem of developer tools across multiple discovery mechanisms:
- 93 `make` targets
- 15+ `ailang` CLI commands
- 11 `tools/` scripts
- 7 skills
- 6 agents

**Problem**: These tools are not uniformly documented, making them hard to discover and use effectively.

**Goal**: Ensure all developer tools are documented well enough that Claude (and human developers) can discover and use them in day-to-day AILANG development work.

## Tool Inventory

### 1. Make Targets (93 total)

#### Core Build & Development (Well documented ✅)
- `make build` - Build the binary
- `make install` - Install binary to GOPATH/bin
- `make quick-install` - Fast reinstall without version info
- `make dev` - Quick development build
- `make clean` - Clean build artifacts
- `make deps` - Download dependencies

#### Testing (Well documented ✅)
- `make test` - Run Go unit tests
- `make test-coverage` - Run tests with coverage
- `make test-coverage-badge` - Quick coverage check
- `make ci` - Run full CI verification
- `make ci-strict` - Extended CI with A2 milestone gates

**Specialized Test Targets** (Documented in `make help` but not in CLAUDE.md):
- `make test-parser` - Run parser tests
- `make test-parser-update` - Update parser golden files
- `make test-lowering` - Run operator lowering golden tests
- `make test-imports` - Test import system (success + errors)
- `make test-import-errors` - Test import error goldens
- `make test-regression-guards` - Run regression guard tests
- `make test-builtin-consistency` - Test builtin three-way parity
- `make test-stdlib-canaries` - Test stdlib health (std/io, std/net)
- `make test-row-properties` - Test row unification properties
- `make test-golden-types` - Test builtin type snapshots
- `make test-repl-smoke` - REPL smoke tests (:type command)
- `make test-stdlib-freeze` - Verify stdlib interfaces haven't changed
- `make test-recursion` - Test recursion limits
- `make test-parity` - Test REPL/file parity (manual only)
- `make test-iface-determinism` - Test interface determinism
- `make test-builtin-freeze` - Freeze builtin type signatures
- `make test-operator-assertions` - Test operator desugaring

**Coverage Targets** (Not documented in CLAUDE.md):
- `make cover-lines` - Show parser line coverage
- `make cover-branch` - Open parser branch coverage HTML
- `make cover-lexer` - Lexer coverage
- `make cover-parser` - Parser coverage
- `make cover-all-packages` - Coverage across all packages

**Gate Targets** (Not documented in CLAUDE.md):
- `make gate-lexer` - Lexer quality gates
- `make gate-parser` - Parser quality gates
- `make gate-all-packages` - Quality gates for all packages

#### Code Quality (Well documented ✅)
- `make fmt` - Format code
- `make fmt-check` - Check code formatting
- `make vet` - Run go vet
- `make lint` - Run linter
- `make install-lint` - Install golangci-lint

#### File Size Management (Well documented in CLAUDE.md ✅)
- `make check-file-sizes` - Check for files >800 lines (AI-friendly)
- `make report-file-sizes` - Report all files >500 lines
- `make codebase-health` - Full codebase health metrics
- `make largest-files` - Show 20 largest files

#### Example Management (Well documented ✅)
- `make verify-examples` - Verify all example files work/fail
- `make update-readme` - Update README with example status
- `make flag-broken` - Add warning headers to broken examples
- `make verify-examples-all` - Verify all examples (including broken)
- `make examples-status` - Show example status

#### Evaluation & Benchmarking (Partially documented)
**Well documented in CLAUDE.md:**
- `make eval-suite` - Run AI benchmark suite
- `make eval-report` - Generate evaluation report
- `make eval-analyze` - Analyze failures, generate design docs (with dedup)
- `make eval-baseline` - Run baseline evaluation
- `make eval-clean` - Clean evaluation results

**Not in CLAUDE.md:**
- `make eval` - Single benchmark evaluation
- `make eval-analyze-fresh` - Force new design docs (disable dedup)
- `make eval-to-design` - Full workflow: evals → analysis → design docs
- `make eval-auto-improve` - Auto-improve prompts based on eval results
- `make eval-auto-improve-apply` - Apply auto-improved prompts
- `make eval-diff` - Diff two eval runs
- `make eval-matrix` - Performance matrix with stats
- `make eval-models` - List available models
- `make eval-prompt-ab` - A/B test prompts
- `make eval-prompt-hash` - Hash prompts for caching
- `make eval-prompt-list` - List prompt versions
- `make eval-summary` - Summarize eval results
- `make eval-validate-fix` - Validate specific fix

#### Stdlib Management (Not documented in CLAUDE.md ⚠️)
- `make freeze-stdlib` - Generate SHA256 golden files for stdlib interfaces
- `make verify-stdlib` - Verify stdlib hasn't changed
- `make test-stdlib-freeze` - Verify stdlib interfaces haven't changed
- `make test-stdlib-canaries` - Test stdlib health (std/io, std/net)

#### Documentation (Partially documented)
**In CLAUDE.md:**
- `make docs-clean` - Clear Docusaurus build cache
- `make docs-restart` - Clear cache and restart dev server

**Not in CLAUDE.md:**
- `make docs` - Build documentation site (alias for docs-serve)
- `make docs-install` - Install documentation dependencies
- `make docs-serve` - Serve documentation locally
- `make docs-preview` - Preview documentation build
- `make docs-build` - Build documentation for production
- `make sync-prompts` - Sync prompts/ to docs/prompts/
- `make generate-llms-txt` - Generate llms.txt for LLM context

#### WASM (Mentioned but not detailed)
- `make build-wasm` - Build WASM binary for browser REPL

#### Other Targets (Not documented)
- `make repl` - Start the REPL (✅ documented)
- `make run FILE=...` - Run an AILANG file (✅ documented)
- `make watch` - Watch mode (local build) (✅ documented)
- `make watch-install` - Watch mode (auto-install to PATH) (✅ documented)
- `make doctor` - Validate builtin registry (✅ documented)
- `make check-golden-drift` - Check for drift in golden files
- `make regen-import-error-goldens` - Regenerate import error goldens
- `make verify-lowering` - Verify operator lowering
- `make verify-no-shim` - Verify no shim code exists
- `make fuzz-parser` - Fuzz test parser
- `make fuzz-parser-long` - Long-running fuzz test
- `make help` - Show help (✅ documented)
- `make help-release` - Show release workflow

### 2. `ailang` CLI Commands (15+ commands)

#### Well Documented (in --help and CLAUDE.md) ✅
- `ailang run [flags] <file>` - Run an AILANG program
- `ailang repl` - Start the interactive REPL
- `ailang check <file>` - Type-check a file without running
- `ailang doctor builtins` - Validate builtin registry
- `ailang builtins list [--by-effect|--by-module]` - List all registered builtins

#### Evaluation Commands (Documented in help but not detailed in CLAUDE.md)
- `ailang eval [flags]` - Run AI benchmarks (AILANG vs Python)
- `ailang eval-suite [flags]` - Run full benchmark suite (parallel)
- `ailang eval-analyze [flags]` - Analyze eval results and generate design docs
- `ailang eval-report <results_dir> <version>` - Generate comprehensive eval report
- `ailang eval-compare <baseline> <new>` - Compare two eval runs
- `ailang eval-matrix <results_dir> <version>` - Performance matrix with stats
- `ailang eval-summary <results_dir>` - Summarize eval results
- `ailang eval-validate <benchmark> [baseline]` - Validate specific fix

#### Partially Documented
- `ailang iface <module>` - Output normalized JSON interface for a module (mentioned in tools/freeze-stdlib.sh but not in CLAUDE.md)
- `ailang test [path]` - Run tests (in --help but not detailed)
- `ailang watch <file>` - Watch file for changes and auto-reload (in --help but not detailed)
- `ailang export-training` - Export training data (in --help but not detailed)

#### Important Flags (Well documented in CLAUDE.md ✅)
```bash
ailang run --caps IO,FS,Net --entry main --args-json '{"foo": 42}' file.ail
```

### 3. `tools/` Scripts (11 scripts)

#### Well Documented (used by make targets or skills) ✅
- `tools/eval_baseline.sh` - Run baseline evaluation (used by make eval-baseline)
- `tools/freeze-stdlib.sh` - Generate stdlib golden files (used by make freeze-stdlib)
- `tools/verify-stdlib.sh` - Verify stdlib (used by make verify-stdlib)
- `tools/audit-examples.sh` - Audit examples (used by make verify-examples)
- `tools/sync-prompts.sh` - Sync prompts to docs (used by make sync-prompts)
- `tools/generate-llms-txt.sh` - Generate llms.txt (used by make generate-llms-txt)

#### Partially Documented (mentioned but not detailed)
- `tools/eval-to-design.sh` - Full eval → design doc workflow
- `tools/eval_auto_improve.sh` - Auto-improve prompts
- `tools/eval_prompt_ab.sh` - A/B test prompts
- `tools/generate_marketing_table.sh` - Generate marketing table (not mentioned anywhere)
- `tools/fix_module_paths.sh` - Fix module paths (not mentioned anywhere)

### 4. Skills (7 skills) - Well Documented ✅

**Location**: `.claude/skills/`

All skills are well documented in `.claude/skills/README.md` and have YAML frontmatter for auto-discovery:

1. **use-ailang** - Write correct AILANG code
2. **skill-builder** - Create new Anthropic Agent Skills
3. **design-doc-creator** - Create AILANG design documents
4. **release-manager** - Create new releases
5. **post-release** - Post-release tasks
6. **sprint-planner** - Create comprehensive sprint plans
7. **sprint-executor** - Execute approved sprint plans

**Status**: ✅ Excellent documentation. Skills are auto-discoverable and include progressive disclosure.

### 5. Agents (6 agents) - Well Documented ✅

**Location**: `.claude/agents/`

1. **eval-orchestrator** - Comprehensive eval workflow management
2. **eval-fix-implementer** - Implement fixes from eval failures
3. **codebase-organizer** - Refactor large files, manage code organization
4. **docs-sync-guardian** - Keep docs in sync with code
5. **design-spec-auditor** - Verify code matches design specs
6. **test-coverage-guardian** - Analyze test coverage, identify gaps

**Status**: ✅ Well documented in CLAUDE.md with clear "when to use" guidance.

## Documentation Gaps

### Critical Gaps (High Priority) ⚠️

1. **Stdlib Management Tools** - Not in CLAUDE.md
   - `make freeze-stdlib`
   - `make verify-stdlib`
   - `make test-stdlib-freeze`
   - Use case: When modifying stdlib, verify no breaking changes

2. **Advanced Eval Commands** - Not detailed in CLAUDE.md
   - `ailang eval-validate` - Validate specific fix
   - `ailang eval-matrix` - Performance matrix
   - `make eval-auto-improve` - Auto-improve prompts
   - Use case: Iterative improvement of AI code generation

3. **Golden File Management** - Not documented
   - `make test-parser-update` - Update parser goldens
   - `make regen-import-error-goldens` - Regenerate import error goldens
   - `make check-golden-drift` - Check for drift
   - Use case: When parser/type system changes, update test expectations

4. **Documentation Generation** - Not in CLAUDE.md
   - `make generate-llms-txt` - Generate llms.txt for LLM context
   - `make sync-prompts` - Sync prompts to docs
   - Use case: Keep docs site and LLM-friendly docs in sync

5. **Interface/Module Tools** - Not documented
   - `ailang iface <module>` - Output normalized JSON interface
   - Use case: Verify module interfaces for stdlib freeze

### Medium Priority Gaps

6. **Specialized Test Targets** - In `make help` but not in CLAUDE.md
   - `make test-regression-guards`
   - `make test-builtin-consistency`
   - `make test-row-properties`
   - `make test-golden-types`
   - Use case: Targeted testing for specific subsystems

7. **Coverage/Gate Targets** - Not documented
   - `make cover-lines`, `make cover-branch`, `make cover-lexer`, etc.
   - `make gate-lexer`, `make gate-parser`, `make gate-all-packages`
   - Use case: Detailed coverage analysis and quality gates

8. **WASM Build** - Mentioned but not detailed
   - `make build-wasm`
   - Use case: Build browser REPL

9. **Fuzz Testing** - Not documented
   - `make fuzz-parser`
   - `make fuzz-parser-long`
   - Use case: Stress testing parser with random inputs

### Low Priority Gaps

10. **Marketing/Utilities** - Not documented
    - `tools/generate_marketing_table.sh` - Unknown use case
    - `tools/fix_module_paths.sh` - Legacy migration tool?

## Recommendations

### 1. Create Developer Tools Reference (High Priority)

**Create**: `docs/docs/guides/developer-tools.md`

Structure:
```markdown
# AILANG Developer Tools Reference

## Quick Reference by Use Case

### When Building/Installing
- `make build` - Local build
- `make install` - Install to system
- `make quick-install` - Fast reinstall

### When Testing
- `make test` - All tests
- `make test-coverage` - With coverage
- `make ci` - Full CI locally

### When Modifying Parser/Types
- `make test-parser` - Parser tests
- `make test-parser-update` - Update goldens
- `make check-golden-drift` - Check drift

### When Modifying Stdlib
- `make freeze-stdlib` - Freeze interfaces
- `make verify-stdlib` - Verify no breakage
- `make test-stdlib-canaries` - Health checks

### When Running Evals
- `make eval-baseline EVAL_VERSION=vX.Y.Z` - Full baseline
- `ailang eval-compare baseline1 baseline2` - Compare runs
- `ailang eval-validate benchmark baseline` - Validate fix

### When Releasing
- Use `release-manager` skill
- Use `post-release` skill
- `make help-release` for quick ref

### When Organizing Code
- `make check-file-sizes` - Check for large files
- Use `codebase-organizer` agent

## Detailed Command Reference
[Full list with descriptions]
```

### 2. Update CLAUDE.md Sections (High Priority)

Add new sections:

#### A. Stdlib Management (after "Adding Builtin Functions")
```markdown
### Stdlib Development & Verification

**When modifying stdlib modules:**

1. **Make changes to stdlib/std/*.ail**
2. **Verify interfaces haven't broken:**
   ```bash
   make verify-stdlib  # Checks SHA256 hashes match
   ```
3. **If intentional breaking change:**
   ```bash
   make freeze-stdlib  # Update golden hashes
   ```
4. **Run health checks:**
   ```bash
   make test-stdlib-canaries  # Tests std/io, std/net work
   ```

**Tools:**
- `make freeze-stdlib` - Generate SHA256 golden files for stdlib interfaces
- `make verify-stdlib` - Verify stdlib hasn't changed (CI check)
- `make test-stdlib-freeze` - Test stdlib interface freeze
- `ailang iface <module>` - Output normalized JSON interface
```

#### B. Golden File Management (in Testing section)
```markdown
### Golden File Testing

**What are golden files?**
Pre-recorded expected outputs for deterministic testing (parser ASTs, type errors, import errors).

**When parser/types change:**
1. Run tests: `make test-parser`
2. If expected failures, update goldens: `make test-parser-update`
3. Review changes: `git diff tests/golden/`
4. Check for drift: `make check-golden-drift`

**Other golden targets:**
- `make regen-import-error-goldens` - Regenerate import error goldens
- `make test-golden-types` - Test builtin type snapshots
- `make test-lowering` - Test operator lowering goldens
```

#### C. Advanced Eval Workflows (expand eval section)
```markdown
### Advanced Eval Workflows

**Validate specific fix:**
```bash
ailang eval-validate fizzbuzz eval_results/baselines/v0.3.14
# Runs just fizzbuzz against baseline to verify fix works
```

**A/B test prompt changes:**
```bash
make eval-prompt-ab PROMPT_A=v0.3.8 PROMPT_B=v0.3.9
# Compares two prompt versions side-by-side
```

**Auto-improve prompts:**
```bash
make eval-auto-improve  # Analyzes failures, suggests prompt improvements
make eval-auto-improve-apply  # Applies suggestions
```

**Performance matrix:**
```bash
ailang eval-matrix eval_results/baselines/v0.3.14 v0.3.14
# Generates model×benchmark performance grid with stats
```
```

#### D. Documentation Tools (new section)
```markdown
### Documentation Maintenance

**Sync prompts to docs site:**
```bash
make sync-prompts  # Copies prompts/ to docs/prompts/ with Jekyll frontmatter
```

**Generate llms.txt for LLM context:**
```bash
make generate-llms-txt  # Creates docs/static/llms.txt with structured context
```

**Update benchmark dashboard:**
```bash
ailang eval-report eval_results/baselines/vX.Y.Z vX.Y.Z --format=json
# Automatically writes to docs/static/benchmarks/latest.json with history preservation
```

**Docusaurus issues:**
- Webpack errors: `make docs-clean && make docs-restart`
- Fresh build: `cd docs && npm run clear && npm start`
```

### 3. Create Quick Reference Card (Medium Priority)

**Create**: `.claude/DX-QUICK-REF.md` (for Claude's quick access)

```markdown
# DX Quick Reference - Common Tasks

## I need to...

### ...build and test changes
- `make quick-install && ailang run --caps IO examples/hello.ail`
- `make test` (all tests)
- `make ci` (full CI locally)

### ...update parser goldens after parser changes
- `make test-parser-update`
- Review: `git diff tests/golden/`

### ...verify stdlib changes don't break compatibility
- `make verify-stdlib` (should pass)
- If intentional break: `make freeze-stdlib`

### ...run evals to test AI code generation
- Dev (cheap models): `make eval-suite`
- Full (all models): `make eval-suite FULL=true`
- Baseline for release: `make eval-baseline EVAL_VERSION=v0.3.X`

### ...validate a specific eval fix
- `ailang eval-validate <benchmark> eval_results/baselines/vX.Y.Z`

### ...check code organization (file sizes)
- `make check-file-sizes` (fails if >800 lines)
- `make report-file-sizes` (lists all >500 lines)
- Use `codebase-organizer` agent to refactor

### ...release a new version
- Use `release-manager` skill
- Then use `post-release` skill for baselines/dashboard

### ...update documentation
- `make sync-prompts` (prompts → docs)
- `make generate-llms-txt` (LLM-friendly context)
- Use `docs-sync-guardian` agent for code↔docs sync

### ...create a new skill
- Use `skill-builder` skill
- See `.claude/skills/SKILLS_GUIDE.md`
```

### 4. Update Skills/Agents (Low Priority)

Consider creating new skills for:
- **stdlib-maintainer** - Wrapper around freeze/verify stdlib workflow
- **golden-updater** - Interactive golden file updates with review

### 5. Improve Tool Discoverability in CLI

Add subcommand grouping to `ailang --help`:
```
Development Tools:
  doctor builtins                 Validate builtin registry
  builtins list [flags]           List all registered builtins
  iface <module>                  Output normalized JSON interface

Testing & Verification:
  check <file>                    Type-check a file without running
  test [path]                     Run tests
  watch <file>                    Watch file for changes

Evaluation & Benchmarking:
  eval [flags]                    Run AI benchmarks
  eval-suite [flags]              Run full benchmark suite
  eval-analyze [flags]            Analyze eval results
  eval-report <dir> <ver>         Generate comprehensive report
  eval-compare <a> <b>            Compare two eval runs
  eval-matrix <dir> <ver>         Performance matrix
  eval-validate <bench> [base]    Validate specific fix
```

## Implementation Plan

### Phase 1: Critical Documentation (2-3 hours)
1. ✅ Create DX tools audit document (this doc)
2. Add stdlib management section to CLAUDE.md
3. Add golden file management section to CLAUDE.md
4. Add advanced eval workflows to CLAUDE.md
5. Add documentation tools section to CLAUDE.md

### Phase 2: Reference Documentation (2-3 hours)
6. Create `docs/docs/guides/developer-tools.md`
7. Create `.claude/DX-QUICK-REF.md`
8. Update skills README with tools cross-references

### Phase 3: CLI Improvements (Optional, 1-2 hours)
9. Add subcommand grouping to `ailang --help`
10. Add `ailang tools list` command for quick reference

### Phase 4: New Skills (Optional, 2-4 hours)
11. Create `stdlib-maintainer` skill
12. Create `golden-updater` skill

## Success Metrics

**Before:**
- 📊 ~30% of tools documented in CLAUDE.md
- 📊 No comprehensive DX tools reference
- 📊 Many tools only discoverable via `make help` or source code

**After:**
- ✅ 90%+ of tools documented in CLAUDE.md (by use case)
- ✅ Complete DX tools reference in docs
- ✅ Quick reference card for common tasks
- ✅ All tools discoverable via one of:
  - CLAUDE.md (contextualized by task)
  - Skills (auto-invoked)
  - Agents (auto-invoked)
  - `docs/docs/guides/developer-tools.md` (comprehensive reference)
  - `.claude/DX-QUICK-REF.md` (quick access)

## Appendices

### Appendix A: Tool Categorization Matrix

| Tool | Documented in CLAUDE.md | Documented in Skills | Discoverable via CLI | Discoverable via Make |
|------|------------------------|---------------------|---------------------|----------------------|
| make build | ✅ | ✅ (use-ailang) | ❌ | ✅ |
| make freeze-stdlib | ❌ | ❌ | ❌ | ✅ |
| make eval-baseline | ✅ | ✅ (post-release) | ❌ | ✅ |
| ailang iface | ❌ | ❌ | ✅ | ❌ |
| ailang eval-validate | ❌ | ⚠️ (eval-orchestrator mentions) | ✅ | ❌ |
| tools/generate-llms-txt.sh | ❌ | ❌ | ❌ | ✅ |

### Appendix B: Documentation Locations

```
CLAUDE.md (primary context for Claude)
├── Critical Principles (git safety, use existing tools, no silent fallbacks)
├── Available Skills (overview)
├── Quick Start (building, testing)
├── Development Workflow (detailed)
├── M-EVAL-LOOP (eval system) ✅
├── Adding Builtin Functions ✅
├── [NEW] Stdlib Management ⚠️
├── [NEW] Golden File Testing ⚠️
├── [NEW] Advanced Eval Workflows ⚠️
├── [NEW] Documentation Tools ⚠️
└── Code Organization Principles ✅

.claude/skills/ (auto-discovered, progressive disclosure)
├── README.md (skills overview) ✅
├── use-ailang/ (AILANG syntax, running code) ✅
├── release-manager/ (release workflow) ✅
├── post-release/ (baselines, dashboard) ✅
├── [NEW] stdlib-maintainer/ (stdlib workflows) ⚠️
└── [NEW] golden-updater/ (golden file workflows) ⚠️

docs/docs/guides/ (comprehensive reference)
├── development.md (general dev guide) ✅
├── benchmarking.md (benchmarking overview) ✅
├── evaluation/ (detailed eval docs) ✅
└── [NEW] developer-tools.md (complete tool reference) ⚠️

.claude/ (quick reference for Claude)
└── [NEW] DX-QUICK-REF.md (common tasks) ⚠️
```

## Next Steps

1. **Immediate** (this session):
   - Update CLAUDE.md with critical gaps (stdlib, goldens, eval, docs)
   - Create `.claude/DX-QUICK-REF.md` for quick access

2. **Next session**:
   - Create comprehensive `docs/docs/guides/developer-tools.md`
   - Update skills README with cross-references

3. **Future enhancements**:
   - CLI improvements (subcommand grouping)
   - New skills (stdlib-maintainer, golden-updater)

---

**Estimated total effort**: 4-6 hours for Phases 1-2 (critical documentation)

**ROI**: Significantly improved developer productivity and reduced time spent hunting for tools.
