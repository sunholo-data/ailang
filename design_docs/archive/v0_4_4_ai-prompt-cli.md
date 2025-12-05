# AI Prompt CLI - First-Class Syntax Reference

**Status**: Planned
**Target**: v0.4.4
**Priority**: P1 (Medium)
**Estimated**: 1 day
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need to know file paths or prompt locations |
| Preserve Semantic Clarity | + | +1 | Canonical, versioned syntax reference always available |
| Increase Determinism | + | +1 | Version-locked prompts ensure consistent AI understanding |
| Lower Token Cost | + | +1 | Simple `ailang prompt` vs reading files/searching docs |
| **Net Score** | | **+4** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- AILANG teaching prompts live in `prompts/v0.3.24.md` etc.
- AIs must know file paths to access syntax reference
- Eval harness reads files directly (`internal/eval_harness/runner.go`)
- No CLI command to display canonical syntax
- Version-to-prompt mapping is implicit (filename convention)

**Impact:**
- **AIs**: Need file path knowledge to access AILANG syntax
- **Developers**: Can't quickly check syntax reference
- **Eval harness**: Tightly coupled to file structure
- **Maintainability**: Prompt location/format could change, breaking downstream users

**Example workflow today:**
```bash
# AI or developer wants AILANG syntax
cat prompts/v0.3.24.md  # Must know the path and version
# Or worse, search for it
find . -name "*.md" | grep prompt
```

## Goals

**Primary Goal:** Make AILANG teaching prompt a first-class CLI feature accessible via `ailang prompt`

**Success Metrics:**
- ✅ `ailang prompt` displays current version's syntax reference
- ✅ `ailang prompt --version v0.3.24` displays specific version
- ✅ Eval harness uses CLI instead of file I/O (single source of truth)
- ✅ Zero file path knowledge required for users
- ✅ Prompts versioned and locked to implementation

## Solution Design

### Overview

Add `ailang prompt` subcommand that:
1. Reads prompt from `prompts/` directory (single source of truth)
2. Displays to stdout (pipe-friendly)
3. Supports `--version` flag for specific versions
4. Defaults to current AILANG version

**Philosophy:** The prompt is part of the language - it should be as accessible as `--help` or `--version`.

### Architecture

**Components:**
1. **CLI command** (`cmd/ailang/main.go`) - Adds `prompt` subcommand
2. **Prompt loader** (`internal/prompt/loader.go`) - Reads prompts from `prompts/`
3. **Version resolution** - Maps `--version` flag to prompt file

**Data flow:**
```
User: ailang prompt --version v0.3.24
  ↓
CLI: parse flags, resolve version
  ↓
Prompt Loader: read prompts/v0.3.24.md
  ↓
CLI: write to stdout
```

**Single source of truth:**
- Prompts remain in `prompts/*.md` (existing location)
- CLI reads from there (no duplication)
- Eval harness calls `ailang prompt` instead of reading files
- Future: Could embed prompts in binary for offline use (Phase 2)

### Implementation Plan

**Phase 1: Read-only CLI** (~6 hours)
- [ ] Create `internal/prompt/loader.go` with `LoadPrompt(version)` function
- [ ] Add `prompt` subcommand to `cmd/ailang/main.go`
- [ ] Implement `--version` flag parsing
- [ ] Default to current version if no flag provided
- [ ] Add error handling (version not found, file missing)
- [ ] Write unit tests for loader
- [ ] Write integration tests for CLI command

**Phase 2: Documentation & Integration** (~2 hours)
- [ ] Update CLAUDE.md with `ailang prompt` workflow
- [ ] Update eval harness to use CLI (deprecate direct file reads)
- [ ] Add examples to `docs/guides/`
- [ ] Update CHANGELOG.md

### Files to Modify/Create

**New files:**
- `internal/prompt/loader.go` - Prompt loading logic (~100 LOC)
- `internal/prompt/loader_test.go` - Unit tests (~150 LOC)
- `cmd/ailang/prompt.go` - CLI command implementation (~80 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Register `prompt` subcommand (~10 LOC)
- `internal/eval_harness/runner.go` - Use CLI instead of file I/O (~20 LOC changed)
- `CLAUDE.md` - Document new workflow (~15 LOC added)

**Total: ~375 LOC (new + modified)**

## Examples

### Example 1: Get Current Prompt

**Before:**
```bash
# AI or developer needs to know file structure
cat prompts/v0.3.24.md
# Or search for it
ls prompts/
```

**After:**
```bash
# Simple, version-agnostic command
ailang prompt
# Output: [Full v0.3.24 teaching prompt]
```

### Example 2: Get Specific Version

**Before:**
```bash
# Must know exact filename convention
cat prompts/v0.3.8.md
# Or guess
cat prompts/v0_3_8.md  # Oops, wrong delimiter!
```

**After:**
```bash
# Explicit, clear version flag
ailang prompt --version v0.3.8
# Output: [Full v0.3.8 teaching prompt]

# Also works
ailang prompt -v v0.3.8
```

### Example 3: Eval Harness Integration

**Before (internal/eval_harness/runner.go):**
```go
// Tightly coupled to file structure
promptPath := filepath.Join("prompts", fmt.Sprintf("v%s.md", version))
content, err := os.ReadFile(promptPath)
if err != nil {
    return fmt.Errorf("failed to read prompt: %w", err)
}
```

**After:**
```go
// Use CLI - single source of truth
cmd := exec.Command("ailang", "prompt", "--version", version)
content, err := cmd.Output()
if err != nil {
    return fmt.Errorf("failed to get prompt: %w", err)
}
```

### Example 4: AI Workflow

**Before:**
```
AI: I need AILANG syntax reference
AI: *searches codebase for prompt files*
AI: *finds prompts/v0.3.24.md*
AI: *reads file*
```

**After:**
```
AI: I need AILANG syntax reference
AI: ailang prompt
AI: *gets canonical syntax immediately*
```

## Success Criteria

- [ ] `ailang prompt` displays current version's prompt
- [ ] `ailang prompt --version v0.3.24` displays specific version
- [ ] `ailang prompt --version invalid` returns helpful error
- [ ] `ailang prompt --help` shows usage
- [ ] Eval harness updated to use CLI (single source of truth)
- [ ] All tests passing (unit + integration)
- [ ] Documentation updated (CLAUDE.md, guides)
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests (`internal/prompt/loader_test.go`):**
- Test `LoadPrompt()` with valid versions
- Test error handling for missing versions
- Test error handling for missing files
- Test version resolution (latest vs specific)
- Mock file system for deterministic tests

**Integration tests (`cmd/ailang/prompt_test.go`):**
- Test `ailang prompt` (default version)
- Test `ailang prompt --version v0.3.24`
- Test `ailang prompt -v v0.3.8`
- Test error cases (invalid version, missing file)
- Test output format (stdout vs stderr)

**Manual testing:**
- Verify output is properly formatted markdown
- Test piping: `ailang prompt | less`
- Test redirecting: `ailang prompt > syntax.md`
- Verify eval harness integration works end-to-end

## Non-Goals

**Not in this feature:**
- **Prompt validation** - Verify prompt accuracy against implementation (Future: Phase 2)
- **Prompt generation** - Auto-generate prompts from grammar/AST (Future: Phase 3)
- **Prompt update workflow** - CLI for updating prompts (Future: Phase 3)
- **Embedded prompts** - Bundle prompts in binary for offline use (Future: Phase 2)
- **Multi-format output** - JSON, HTML, etc. (Future: Nice to have)
- **Prompt diffing** - Compare prompts across versions (Future: Nice to have)

**Rationale:** Keep Phase 1 minimal - just read and display. Validation and generation are complex features that deserve separate design docs.

## Timeline

**Week 1** (8 hours):
- Day 1-2 (6h): Implementation (loader + CLI + tests)
- Day 2-3 (2h): Documentation + integration

**Total: ~8 hours across 1 week**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Prompt file structure changes | Medium | Abstract file reading behind `loader.go` interface |
| Version naming inconsistencies (v0.3.24 vs v0_3_24) | Low | Normalize version strings in loader |
| Eval harness breaks if CLI changes | Medium | Add integration tests, deprecate direct file reads gradually |
| Prompts too large for stdout | Low | Use paging (pipe to `less`) or add `--format` flag in future |

## References

- Current prompts: `prompts/` directory
- Eval harness: `internal/eval_harness/runner.go`
- CLI structure: `cmd/ailang/main.go`
- Related: [M-EVAL architecture](../../implemented/v0_3_14/M-EVAL_architecture.md)

## Future Work

**Phase 2: Validation & Accuracy (v0.4.5+)**
- `ailang prompt --validate` - Test prompt accuracy against implementation
- Compare prompt syntax with parser grammar
- Run sample code from prompt through compiler
- Report discrepancies

**Phase 3: Update Workflow (v0.5.0+)**
- `ailang prompt --generate` - Auto-generate prompt from grammar
- `ailang prompt --update` - Interactive prompt editor
- Version prompts automatically on release
- Single source of truth: grammar → prompt (not manual sync)

**Nice-to-have:**
- `ailang prompt --format json` - Machine-readable format
- `ailang prompt --diff v0.3.8 v0.3.24` - Compare versions
- Embed prompts in binary for offline use
- Web viewer: `ailang prompt --serve` (local HTTP server)

---

**Document created**: 2025-11-08
**Last updated**: 2025-11-08
