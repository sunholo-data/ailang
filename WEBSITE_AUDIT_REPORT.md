# AILANG Website Audit Report

**Date**: December 5, 2025
**Current Version**: v0.5.6
**Auditor**: Claude Code

## Executive Summary

This audit identifies discrepancies between what the AILANG website claims and what is actually implemented. The website has significant issues with:

1. **Future features presented as current** - Shared Semantic State, Execution Profiles, Deterministic Tooling
2. **Outdated version references** - Teaching prompts reference v0.4.4, current is v0.5.6
3. **Missing documentation for implemented features** - Go codegen improvements, Array support, Import aliasing
4. **Broken/dead pages** - execution-profiles.mdx describes unimplemented features as current

---

## Critical Discrepancies (MUST FIX)

### 1. Shared Semantic State - NOT IMPLEMENTED

**Location**: [docs/architecture/shared-semantic-state.mdx](docs/docs/architecture/shared-semantic-state.mdx)

**Website Claims**:
- "AILANG is the first language designed for distributed cognition"
- `SharedMem` effect with CAS operations
- `sem_frame` type with embeddings
- `AI.embed` effect for generating embeddings
- Multi-agent plan coordination example code

**Reality**:
- **NONE of this is implemented**
- No `SharedMem` effect exists
- No `sem_frame` type
- No embedding support
- Design doc exists at `design_docs/planned/v0_5_0/semantic-caching.md` but never implemented

**Fix Required**:
- Move entire page to "Future Vision" or "Roadmap" section
- Add clear "PLANNED FOR v0.8.0" banner
- Remove code examples that show non-existent syntax
- Link to design doc as the authoritative source

---

### 2. Execution Profiles - NOT IMPLEMENTED

**Location**: [docs/architecture/execution-profiles.mdx](docs/docs/architecture/execution-profiles.mdx)

**Website Claims**:
- Three profiles: SimProfile, ServiceProfile, CliProfile
- `--profile sim` and `--profile cli` flags
- Profile validation by compiler
- Go runtime structure with multiple files

**Reality**:
- **Profiles do not exist in the compiler**
- No `--profile` flag implemented
- Design doc exists at `design_docs/planned/v0_6_0/execution-profiles.md`
- Only basic `--emit-go` codegen exists (no profile selection)

**Fix Required**:
- Move to "Planned Features" or "Roadmap"
- Add "PLANNED FOR v0.6.0" banner
- Current Go codegen should be documented separately

---

### 3. Deterministic Tooling - NOT IMPLEMENTED

**Location**: [docs/intro.mdx](docs/docs/intro.mdx) lines 95-102, [docs/vision.mdx](docs/docs/vision.mdx)

**Website Claims**:
- `ailang normalize` for canonical formatting
- `ailang suggest-imports` for import fixes
- `ailang apply` for automated code repair
- `fold`, `unfold`, `iterateN` total recursion combinators
- `reflectType`, `reflectEffect` runtime reflection
- `--emit-trace` for training data export

**Reality**:
- **NONE of these commands exist**
- No `normalize`, `suggest-imports`, or `apply` commands
- No total recursion combinators
- No runtime reflection
- No trace export

**Fix Required**:
- Move all these to a "Roadmap" page
- Remove from "Coming in v0.4" section (we're at v0.5.6!)
- Create design docs for each planned feature

---

### 4. Vision Page Timeline Out of Date

**Location**: [docs/vision.mdx](docs/docs/vision.mdx) lines 299-308

**Website Says**:
```
v0.3.x | JSON decode + stable core | ✅ Complete
v0.4.x | Monomorphization + type system maturity | ✅ Complete
v0.5.x | Go codegen + AI effect + game support | ✅ In Progress
```

**Reality**:
- v0.5.x is now complete (v0.5.6 released)
- Need to add v0.6.0 as "In Progress"

---

## Major Discrepancies (SHOULD FIX)

### 5. Outdated Teaching Prompt References

**Locations**: Multiple pages reference v0.4.4 prompt

**Website Says**:
- intro.mdx: Links to `/docs/prompts/v0.4.4`
- getting-started.mdx: References v0.4.4

**Reality**:
- Latest prompt should be v0.5.1 or higher
- The `ACTIVE_PROMPT` constant may be outdated

**Fix**: Update all prompt references to latest version

---

### 6. Missing Documentation for Implemented Features

These features ARE implemented but NOT documented on website:

| Feature | Version | Status |
|---------|---------|--------|
| **Import Aliasing** (`import std/list as List`) | v0.5.1 | No dedicated page |
| **Multi-file Go Compilation** | v0.5.2 | Not documented |
| **Named ADT Constructor Fields** | v0.5.3 | Not documented |
| **Typed ADT Slices** | v0.5.3 | Not documented |
| **Array Type Application** (`Array[T]`) | v0.5.6 | Not documented |
| **Array Literals** (`#[1, 2, 3]`) | v0.5.6 | Not documented |
| **Relaxed Module Matching** (`--relax-modules`) | v0.5.2 | Not documented |
| **RecordUpdate in Go Codegen** | v0.5.1 | Not documented |
| **Typed Function Signatures** | v0.5.5 | Not documented |
| **Agent Messaging System** | v0.5.6 | Only in guides |

---

### 7. implementation-status.md is Stale

**Location**: [docs/reference/implementation-status.md](docs/docs/reference/implementation-status.md)

**Issues**:
- Last detailed release is v0.5.1
- Missing v0.5.2 through v0.5.6
- Example status says "48/66" but needs verification
- File size issues section may be outdated

---

### 8. Why AILANG Claims Need Verification

**Location**: [docs/why-ailang.mdx](docs/docs/why-ailang.mdx)

**Claims to Verify**:
- "LLMs can reliably produce correct AILANG on the first attempt" - Need benchmark data
- "Hot-swappable logic" - Not implemented (planned feature)
- "Safe sandbox" - Capability system exists but not full sandboxing
- "Dynamically load behaviors" - Not implemented

---

## Minor Discrepancies (NICE TO FIX)

### 9. Benchmark Data May Be Stale

**Location**: [docs/benchmarks/performance.md](docs/docs/benchmarks/performance.md)

- Claims "264 benchmarks" but should verify against `docs/static/benchmarks/latest.json`
- Version references may be outdated
- Model names may have changed

### 10. Sidebar Organization Issues

Current sidebar mixes:
- Implemented features
- Planned/future features
- Architecture docs (some theoretical)

Should separate into:
- Current Features (what works now)
- Roadmap (what's planned)
- Vision (theoretical/future)

---

## Proposed Remediation Plan

### Phase 1: Critical Fixes (Immediate)

1. **Add "FUTURE" banners to theoretical pages**
   - shared-semantic-state.mdx → Add "PLANNED FOR v0.8.0" banner
   - execution-profiles.mdx → Add "PLANNED FOR v0.6.0" banner

2. **Update intro.mdx "Coming in v0.4" section**
   - Remove entirely or rename to "Roadmap"
   - We're at v0.5.6, this is confusing

3. **Update vision.mdx timeline**
   - Mark v0.5.x as complete
   - Add v0.6.0 as next milestone

### Phase 2: Structural Reorganization (This Week)

1. **Create new "Roadmap" section** with pages:
   - `/docs/roadmap/deterministic-tooling` - normalize, suggest-imports, apply
   - `/docs/roadmap/execution-profiles` - SimProfile, ServiceProfile, CliProfile
   - `/docs/roadmap/shared-semantic-state` - Move current architecture page here
   - `/docs/roadmap/wasm-backend` - Planned WASM support

2. **Create "Release Notes" section** documenting:
   - v0.5.2: Multi-file compilation, relaxed modules
   - v0.5.3: Named ADT fields, typed slices
   - v0.5.4: Integer literals, FieldGet improvements
   - v0.5.5: Typed function signatures
   - v0.5.6: Array support, agent messaging

3. **Update sidebar** to separate Current vs Future:
   ```
   Getting Started
   Language Reference
   Current Features (v0.5.x)
     - Go Codegen
     - Effect System
     - Testing
     - Module System
   Roadmap
     - Execution Profiles (v0.6.0)
     - WASM Backend (v0.7.0)
     - Shared Semantic State (v0.8.0)
   ```

### Phase 3: New Feature Documentation (This Sprint)

Create dedicated pages for undocumented features:

1. **Go Codegen Deep Dive** - `/docs/guides/go-codegen`
   - Multi-file compilation
   - Typed function signatures
   - RecordUpdate support
   - Array codegen
   - Effect handler interfaces

2. **Import System** - `/docs/reference/imports`
   - Import aliasing (`import std/list as List`)
   - Symbol aliasing (`import std/list (length as listLength)`)
   - Relaxed module matching

3. **Arrays in AILANG** - `/docs/reference/arrays`
   - Array type syntax (`Array[T]`)
   - Array literals (`#[1, 2, 3]`)
   - Array runtime functions

4. **ADT Best Practices** - `/docs/guides/adts`
   - Named constructor fields
   - Typed slices in Go
   - Pattern matching

### Phase 4: Quality Assurance (Ongoing)

1. **Automate version checking**
   - Create script to verify `ACTIVE_PROMPT` matches latest
   - CI check for stale version references

2. **Link verification**
   - Verify all internal links work
   - Check design doc links point to correct files

3. **Example verification**
   - All code examples should be tested
   - Use raw-loader pattern consistently

---

## File-by-File Action Items

### MUST UPDATE

| File | Action |
|------|--------|
| `docs/intro.mdx` | Remove/rename "Coming in v0.4" section |
| `docs/vision.mdx` | Update timeline, mark v0.5.x complete |
| `docs/architecture/shared-semantic-state.mdx` | Add PLANNED banner, move to roadmap |
| `docs/architecture/execution-profiles.mdx` | Add PLANNED banner, move to roadmap |
| `docs/reference/implementation-status.md` | Add v0.5.2-v0.5.6 releases |

### SHOULD UPDATE

| File | Action |
|------|--------|
| `docs/why-ailang.mdx` | Verify claims, add caveats for unimplemented features |
| `docs/guides/getting-started.mdx` | Update prompt version references |
| `docs/benchmarks/performance.md` | Verify data against latest.json |
| `docs/sidebars.js` | Reorganize Current vs Roadmap |

### CREATE NEW

| File | Content |
|------|---------|
| `docs/roadmap/index.md` | Roadmap overview |
| `docs/roadmap/deterministic-tooling.md` | Move planned features here |
| `docs/guides/go-codegen.md` | Comprehensive Go codegen docs |
| `docs/reference/imports.md` | Import aliasing documentation |
| `docs/reference/arrays.md` | Array type documentation |
| `docs/releases/index.md` | Release notes hub |

---

## Summary Statistics

| Category | Count |
|----------|-------|
| Critical discrepancies | 4 |
| Major discrepancies | 4 |
| Minor discrepancies | 2 |
| Missing feature pages | 10+ |
| Outdated pages | 5+ |

**Estimated effort**: 2-3 days for Phase 1-2, ongoing for Phase 3-4

---

## Appendix: Feature Implementation Matrix

### Implemented (needs docs)

| Feature | Design Doc | Website Page | Action |
|---------|------------|--------------|--------|
| Import Aliasing | `v0_5_0/m-dx7-import-alias.md` | None | Create page |
| Multi-file Go | `v0_5_2/` | None | Create page |
| Named ADT Fields | `v0_5_3/` | None | Create page |
| Array Support | `v0_5_6/` | None | Create page |
| Relaxed Modules | `v0_5_2/m-dx11-relaxed-module-matching.md` | None | Create page |

### Planned (incorrectly shown as current)

| Feature | Design Doc | Website Page | Action |
|---------|------------|--------------|--------|
| Execution Profiles | `v0_6_0/execution-profiles.md` | `architecture/` | Move to roadmap |
| Shared Semantic State | `v0_5_0/semantic-caching.md` | `architecture/` | Move to roadmap |
| Deterministic Tooling | Various | `intro.mdx`, `vision.mdx` | Move to roadmap |
| WASM Backend | None | `vision.mdx` | Create design doc |

### Not Planned (should remove)

| Feature | Website Mention | Action |
|---------|-----------------|--------|
| CSP Concurrency | implementation-status.md | Remove or clarify replaced by effects |
| Session Types | implementation-status.md | Remove or move to far-future |
