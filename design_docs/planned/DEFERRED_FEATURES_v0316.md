# Deferred Features from Original Sprint Ticket

**Original Ticket**: v0.3.12 Sprint - Benchmark Recovery + Deterministic Tooling
**Date Created**: 2025-10-18
**Reason for Split**: Scope too large (38 hours); split into Phase 1 (v0.3.14) and Phase 2 (v0.3.15)

This document tracks features from the original sprint ticket that were deferred to future versions.

---

## Original Sprint Scope

**Full Original Ticket** (5 days, 38 hours):

| Component | Est. Time | Status | Target Version |
|-----------|-----------|--------|----------------|
| std/json.decode | 12h | ✅ Planned | v0.3.14 (Phase 1) |
| CLI: normalize | 6h | ✅ Planned | v0.3.15 (Phase 2) |
| CLI: suggest imports | 6h | ✅ Planned | v0.3.15 (Phase 2) |
| CLI: apply | 4h | ✅ Planned | v0.3.15 (Phase 2) |
| Schemas + golden tests | 4h | ✅ Planned | v0.3.15 (Phase 2) |
| import M (*) syntax | 2h | ❌ Deferred | v0.3.16+ |
| FX001 fix-it diagnostic | 2h | ❌ Deferred | v0.3.16+ |
| CI integration | 2h | ✅ Planned | v0.3.15 (Phase 2) |

**Total Planned**: 34h (v0.3.14 + v0.3.15)
**Total Deferred**: 4h (v0.3.16+)

---

## Deferred to v0.3.16+

### 1. Import Wildcard Syntax: `import M (*)`

**Original Spec**:
- Allow `import std/io (*)` to import all exported symbols
- Parser desugaring: expand `(*)` into explicit list at parse time
- Estimated: 2 hours

**Why Deferred**:
- Language feature (requires parser changes)
- Not critical for benchmark recovery (explicit imports work fine)
- Phase 2 (v0.3.15) already has substantial scope
- Low ROI: Convenience feature, not functional blocker

**Implementation Notes** (when picked up):
1. Modify parser to recognize `*` in import symbol list
2. During elaboration, resolve module exports
3. Expand `import M (*)` → `import M (f1, f2, f3, ...)` in AST
4. No runtime changes needed (desugared at compile time)

**Files to Change**:
- `internal/parser/parser.go` - Recognize `*` token
- `internal/elaborate/elaborate.go` - Expand wildcard
- `internal/loader/loader.go` - Cache module exports
- Tests: `internal/parser/import_test.go`

**Design Doc**: Create `design_docs/planned/M-PARSER-IMPORT-WILDCARD.md` when scheduled

---

### 2. FX001 Diagnostic Fix-It (Auto-Add Effect Annotations)

**Original Spec**:
- Add automatic `! {IO}` fix-it for FX001 diagnostic (missing effect annotation)
- New entry in diagnostics registry
- Estimated: 2 hours

**Why Deferred**:
- Requires diagnostic infrastructure enhancement (not yet mature)
- Phase 1's `normalize` command will handle this for fragments (different approach)
- Not critical for benchmark recovery (normalize handles it)
- Better to wait for full diagnostic system redesign (v0.4.0?)

**Current Workaround**:
- Use `ailang normalize` to infer and add effects (Phase 2, v0.3.15)
- Manual annotation by developers

**Implementation Notes** (when picked up):
1. Extend `internal/errors/diagnostics.go` with fix-it support
2. Add effect inference to diagnostic emission
3. Generate JSON edit suggestions (compatible with `apply` command)
4. IDE integration via LSP (v0.4.0+)

**Files to Change**:
- `internal/errors/diagnostics.go` - Fix-it infrastructure
- `internal/types/effects.go` - Effect inference (reuse from normalize)
- `internal/pipeline/pipeline.go` - Emit fix-it JSON
- Tests: `internal/errors/diagnostics_test.go`

**Design Doc**: Create `design_docs/planned/M-DIAG-FIXIT.md` when scheduled

---

## Features Pushed to v0.4.0+ (Beyond Original Scope)

These were mentioned in the original ticket's "Next Sprint Preview" but not part of core scope:

### 1. Local Daemon + LSP Bridge (v0.4.0)

**Description**: Long-running daemon for real-time tooling

**Why v0.4.0**:
- Requires stable CLI tools first (v0.3.15)
- Needs LSP protocol implementation (large scope)
- IDE integration is separate milestone

**Components**:
- Daemon: `ailangd` - Long-running process
- LSP server: JSON-RPC over stdio
- IDE plugins: VSCode, Vim, Emacs
- Real-time: normalize, suggest imports, diagnostics

**Estimated**: 10-15 days

---

### 2. Effect Composer (v0.4.1)

**Description**: Auto-infer minimal effect sets from function bodies

**Why v0.4.1**:
- Requires complete effect system (v0.3.14 has basics)
- Needs dependency analysis (which calls require which effects)
- Complex algorithm (effect inference with constraints)

**Example**:
```ailang
-- User writes:
func processFile(path: string) {
  let content = readFile(path)  -- requires FS
  println(content)              -- requires IO
}

-- Composer infers:
func processFile(path: string) -> () ! {IO, FS} { ... }
```

**Estimated**: 5-7 days

---

### 3. Test Generator (v0.4.2)

**Description**: Generate test cases from function signatures

**Why v0.4.2**:
- Requires property-based testing framework (not implemented)
- Needs type-driven generation (complex)
- Lower priority than core features

**Example**:
```ailang
-- User writes:
export func add(x: int, y: int) -> int { x + y }

-- Generator creates:
-- Property: add(x, y) == add(y, x)  (commutative)
-- Property: add(add(x, y), z) == add(x, add(y, z))  (associative)
-- Property: add(x, 0) == x  (identity)
```

**Estimated**: 8-10 days

---

### 4. Extended JSON Features (v0.4.3+)

**Description**: Beyond basic encode/decode

**Features Deferred**:
- Unicode escape sequences: `\uXXXX` (v0.3.14 skips this)
- Streaming JSON parser (for large files >1MB)
- JSON Schema validation (validate JSON against schema)
- Pretty-printing with indentation
- `decodeInto[T]` - Type-safe decode directly into ADTs

**Why Deferred**:
- MVP decode/encode is sufficient for benchmarks
- Unicode escapes rarely used in practice
- Streaming not needed (benchmark JSON <10KB)
- Schema validation is advanced feature

**Estimated**: 3-5 days (when needed)

---

## Prioritization for Future Sprints

**Recommended Order**:
1. ✅ **v0.3.14** (Phase 1): JSON Decode - **IMMEDIATE**
2. ✅ **v0.3.15** (Phase 2): Deterministic Tooling - **NEXT**
3. **v0.3.16**: Import Wildcard + Minor Ergonomics (1-2 days)
4. **v0.4.0**: LSP + Daemon (10-15 days)
5. **v0.4.1**: Effect Composer (5-7 days)
6. **v0.4.2**: Test Generator (8-10 days)
7. **v0.4.3**: Extended JSON (3-5 days)

**Rationale**:
- v0.3.14-15: Benchmark recovery is **critical** (enables AI evaluation)
- v0.3.16: Low-hanging ergonomics (quick wins)
- v0.4.0+: Infrastructure for production use (LSP, tooling maturity)

---

## Decision Log

### Why Split Original Sprint?

**Original Estimate**: 38 hours (4.75 days at 8h/day)

**Actual Constraints**:
- Recent velocity: ~500-700 LOC/day (from v0.3.10-13 work)
- Original scope: ~2,355 LOC (tooling) + ~1,145 LOC (JSON) = ~3,500 LOC
- Timeline: 3,500 LOC ÷ 600 LOC/day = **5.8 days**
- Risk: Single large release increases bug risk

**Decision** (2025-10-18):
- **Phase 1 (v0.3.14)**: JSON Decode (2-3 days) - **Critical path**
- **Phase 2 (v0.3.15)**: Tooling (3-4 days) - **Depends on Phase 1**
- **Deferred (v0.3.16+)**: Import wildcard, FX001 fix-it (1-2 days) - **Nice-to-have**

**Benefits**:
- ✅ Faster feedback loop (test JSON decode immediately)
- ✅ Lower risk (smaller releases, easier to debug)
- ✅ Better documentation (focused design docs)
- ✅ Parallel work possible (tooling team can plan Phase 2 while Phase 1 executes)

### Why Defer Import Wildcard?

**Arguments For Deferring**:
1. Not on critical path for benchmark recovery
2. Explicit imports work fine (verbose but clear)
3. Parser changes are risky (could break existing code)
4. Low ROI (convenience feature, not functionality)

**Arguments Against Deferring**:
1. Would reduce AI-generated import boilerplate
2. Only 2 hours (small scope)
3. User ergonomics improvement

**Decision**: Defer to v0.3.16
- Reasoning: Phase 2 already has 3-4 days of work; adding parser changes increases risk
- Mitigation: `suggest imports` command (Phase 2) auto-generates explicit imports

### Why Defer FX001 Fix-It?

**Arguments For Deferring**:
1. Diagnostic system needs redesign (not mature yet)
2. `normalize` command (Phase 2) handles this for fragments
3. Manual annotation works for developers
4. LSP integration (v0.4.0) is better venue for fix-its

**Arguments Against Deferring**:
1. Would improve DX for new users
2. Only 2 hours (small scope)

**Decision**: Defer to v0.3.16+
- Reasoning: `normalize` provides same functionality for AI-generated code (main use case)
- Better to do it right in v0.4.0 with full LSP support than patch it in v0.3.15

---

## Tracking

**Issue Tracker**: (If using GitHub Issues)
- Create milestone: "v0.3.16 - Ergonomics"
- Create issues:
  - #XXX: Import wildcard syntax `import M (*)`
  - #XXX: FX001 diagnostic fix-it (auto-add effects)

**Design Docs**: (When scheduled)
- `design_docs/planned/M-PARSER-IMPORT-WILDCARD.md`
- `design_docs/planned/M-DIAG-FIXIT.md`

**Dependencies**:
- Import wildcard: None (can be done anytime)
- FX001 fix-it: Depends on `normalize` command (v0.3.15)

---

## Summary

**Original Sprint Scope**: 38 hours across 8 components

**Execution Plan**:
- ✅ **v0.3.14** (Phase 1): JSON Decode (12h) - **Planned**
- ✅ **v0.3.15** (Phase 2): Tooling (22h) - **Planned**
- ❌ **v0.3.16+** (Deferred): Import wildcard + FX001 (4h) - **Future**

**Rationale**: Focus on critical path (benchmark recovery) first, defer ergonomics.

**Next Actions**:
1. Execute Phase 1 (v0.3.14) - JSON Decode
2. Validate benchmarks pass
3. Execute Phase 2 (v0.3.15) - Deterministic Tooling
4. Schedule v0.3.16 for deferred features (or skip to v0.4.0)

---

## References

- **Phase 1 Design**: design_docs/20251018/M-LANG-JSON-DECODE.md
- **Phase 2 Design**: design_docs/planned/M-TOOLING-DETERMINISTIC.md
- **Original Sprint Ticket**: /plan-sprint command arguments (2025-10-18)
- **Velocity Data**: CHANGELOG.md (v0.3.10-13 entries)
