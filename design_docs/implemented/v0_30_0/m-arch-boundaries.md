# M-ARCH-BOUNDARIES: Formalize Dashboard/Core Separation

> **REVIVED 2026-07-14 (Mark, strategic audit)** — moved deferred/ → planned/v0_30_0.
> **Phases 1–3 (boundary docs, check_boundaries.sh CI gate, CODEOWNERS) are APPROVED for loop
> execution PRE-v1.0** — they let the 1.0 stability promise scope to `core/` (~120k LOC) instead
> of the full ~270k. **Phase 4 (physical git mv) is deliberately scheduled AT the v1.0→v1.1
> boundary** — import churn lands in one re-baselining moment. The doc's separate-repos
> REJECTION is REAFFIRMED: this week's loop velocity depends on atomic cross-cutting commits
> (parser+prompt+fixtures+docs in one evaluated PR); repo splits would tax every iteration.


**Status**: IMPLEMENTED 2026-07-20 (iter 68, Phases 1-3; Phase 4 physical mv deferred to v1.0→v1.1) — PR #420 `ee97fada6`, eval PASS 88/100 r1
**Target**: v0.7.0
**Priority**: P1 (Medium)
**Estimated**: 2-3 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | No change to trace system |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | +1 | Enforces clearer boundaries on what code can access |
| A5: Bounded Verification | +1 | Enables local CI checks per subsystem |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Better toolability - agents can be scoped to subsystems |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No change to cost tracking |
| A10: Composability | +1 | Cleaner composition between dashboard and core |
| A11: Structured Failure | 0 | No change to error handling |
| A12: System Boundary | +1 | Formalizes the boundary between products |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have -1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

The AILANG repository contains two products with fundamentally different characteristics:

1. **Core Language**: Parser, type system, evaluator, effects - requires conservative changes, deterministic behavior, formal verification
2. **Dashboard (Collaboration Hub)**: UI, coordinator, observatory - requires rapid iteration, UX experimentation, frequent releases

**Current State:**
- Dashboard commits outnumber core commits 13:1 (last 2 weeks)
- Dashboard code (52,867 LOC) exceeds core compiler (38,787 LOC)
- Both products share same version number and release cadence
- No formal boundary enforcement (could drift over time)
- Features like M-STDLIB-DATETIME are driven by dashboard needs

**Impact:**
- Dashboard velocity is constrained by conservative compiler release cadence
- Risk of architectural drift as dashboard grows
- Agents cannot be easily scoped to subsystems
- New contributors unclear on what they can modify safely

## Goals

**Primary Goal:** Formalize the existing dashboard/core separation with mechanical enforcement, enabling independent evolution while preserving monorepo benefits.

**Success Metrics:**
- CI boundary check catches any cross-boundary imports
- Dashboard can be released independently of compiler
- CODEOWNERS restricts agent/contributor access by subsystem
- Zero files in `core/` importing from `apps/`
- Documentation clearly explains the architecture

## Solution Design

### Overview

The architecture is already 80% correct - dashboard has ZERO imports from parser/types/eval. The work is to:
1. Physically reorganize directories to make boundaries visible
2. Add CI enforcement to prevent drift
3. Enable separate release tracks
4. Document the architecture for agents and contributors

### Architecture

**Current State (Clean but Implicit):**
```
internal/
├── parser/          # Core
├── types/           # Core
├── eval/            # Core
├── server/          # Dashboard (no core imports)
├── coordinator/     # Dashboard (no core imports)
├── observatory/     # Dashboard (no core imports)
└── embed/           # Bridge (controlled)
```

**Proposed State (Explicit Boundaries):**
```
ailang/
├── core/                        # Compiler implementation
│   ├── ast/
│   ├── lexer/
│   ├── parser/
│   ├── types/
│   ├── typeclass/
│   ├── elaborate/
│   ├── link/
│   ├── pipeline/
│   ├── eval/
│   ├── runtime/
│   ├── effects/
│   └── builtins/
│
├── apps/                        # Product code
│   └── dashboard/
│       ├── server/              # HTTP API layer
│       ├── coordinator/         # Task execution engine
│       ├── observatory/         # Telemetry system
│       ├── messaging/           # Message bus
│       └── ui/                  # React frontend
│
├── tools/                       # Development tooling
│   ├── eval_harness/           # AI evaluation
│   ├── eval_analysis/          # Result analysis
│   └── ai/                     # AI provider clients
│
├── sdk/                        # Stable interfaces
│   ├── embed/                  # AILANG embedding API
│   ├── schema/                 # Versioned trace schemas
│   └── api/                    # HTTP API types
│
├── stdlib/                     # Standard library (unchanged)
├── cmd/ailang/                 # CLI entry point (unchanged)
└── internal/                   # Truly internal utilities
```

**Key Boundaries:**

| Direction | Allowed | Enforced By |
|-----------|---------|-------------|
| apps/ → core/ | NO (except via sdk/) | CI grep check |
| core/ → apps/ | NO | CI grep check |
| tools/ → core/ | YES | None needed |
| tools/ → apps/ | NO | CI grep check |
| sdk/ → core/ | YES | Design (controlled bridge) |

### Implementation Plan

**Phase 1: Boundary Enforcement** (~4 hours)

Before moving any files, add enforcement to catch violations:

- [ ] Create `scripts/check_boundaries.sh` with import validation
- [ ] Add `make check-boundaries` target to Makefile
- [ ] Add to CI pipeline (fail on violation)
- [ ] Verify current state passes (it should)

**Phase 2: Documentation** (~2 hours)

- [ ] Create `ARCHITECTURE.md` documenting the layers
- [ ] Add boundary rules to CLAUDE.md for agents
- [ ] Update CONTRIBUTING.md with subsystem guidance
- [ ] Create `core/doc.go` and `apps/doc.go` with boundary comments

**Phase 3: CODEOWNERS & Releases** (~2 hours)

- [ ] Create `.github/CODEOWNERS` with path-based ownership
- [ ] Add `make release-dashboard` target for dashboard-only releases
- [ ] Update release-manager skill to support dual tracks
- [ ] Create `dashboard/CHANGELOG.md` for dashboard-specific history

**Phase 4: Physical Restructure** (~8 hours) [Optional - Can defer]

Moving files has significant cost (import path updates, test fixes). Can defer to v0.8.0 if boundaries hold with current structure.

If proceeding:
- [ ] Create new directory structure
- [ ] Move files with `git mv` (preserves history)
- [ ] Update all import paths
- [ ] Fix tests
- [ ] Verify `make test` passes

### Files to Modify/Create

**New files:**
- `scripts/check_boundaries.sh` - Import validation script (~50 LOC)
- `ARCHITECTURE.md` - Architecture documentation (~200 LOC)
- `.github/CODEOWNERS` - Ownership rules (~30 LOC)
- `core/doc.go` - Boundary documentation (~10 LOC)
- `apps/doc.go` - Boundary documentation (~10 LOC)

**Modified files:**
- `Makefile` - Add `check-boundaries`, `release-dashboard` targets (~20 LOC)
- `CLAUDE.md` - Add architecture section, agent boundaries (~50 LOC)
- `CONTRIBUTING.md` - Add subsystem guidance (~30 LOC)

**Total new code:** ~400 LOC

## Examples

### Example 1: Boundary Check Script

**scripts/check_boundaries.sh:**
```bash
#!/bin/bash
set -e

echo "Checking import boundaries..."

# Core must not import dashboard
if grep -r 'import.*"github.com/sunholo/ailang/internal/(server|coordinator|observatory)"' \
    internal/parser internal/types internal/eval internal/core 2>/dev/null; then
    echo "BOUNDARY VIOLATION: core imports dashboard"
    exit 1
fi

# Dashboard must not import core directly (except embed)
if grep -r 'import.*"github.com/sunholo/ailang/internal/(parser|types|eval|core)"' \
    internal/server internal/coordinator internal/observatory 2>/dev/null; then
    echo "BOUNDARY VIOLATION: dashboard imports core directly"
    exit 1
fi

echo "Boundaries clean"
```

### Example 2: CODEOWNERS

**.github/CODEOWNERS:**
```
# Core compiler - conservative changes
/internal/parser/        @ailang/compiler
/internal/types/         @ailang/compiler
/internal/eval/          @ailang/compiler
/internal/effects/       @ailang/compiler
/internal/builtins/      @ailang/compiler
/stdlib/                 @ailang/compiler

# Dashboard - faster iteration
/internal/server/        @ailang/dashboard
/internal/coordinator/   @ailang/dashboard
/internal/observatory/   @ailang/dashboard
/ui/                     @ailang/dashboard

# Bridge - requires both teams
/internal/embed/         @ailang/compiler @ailang/dashboard
```

### Example 3: Separate Release

```bash
# Conservative compiler release (existing workflow)
make release VERSION=v0.7.0

# Fast dashboard release (new workflow)
make release-dashboard VERSION=dashboard-v1.3.0

# Release creates tag: dashboard/v1.3.0
# Updates: apps/dashboard/CHANGELOG.md
# Does NOT bump core version
```

## Success Criteria

- [ ] `make check-boundaries` passes on current codebase
- [ ] CI fails if boundary is violated (tested with intentional violation)
- [ ] ARCHITECTURE.md accurately describes the layering
- [ ] CODEOWNERS file restricts PRs to appropriate reviewers
- [ ] `make release-dashboard` creates independent dashboard release
- [ ] CLAUDE.md updated with agent boundary guidance
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Boundary check script tested with mock violations

**Integration tests:**
- CI pipeline runs boundary check on every PR
- Verify release-dashboard creates correct tags

**Manual testing:**
- Review that CODEOWNERS triggers correct reviewers
- Verify agents respect boundaries in CLAUDE.md

## Non-Goals

**Not in this feature:**
- Physical directory restructure (core/, apps/) - Can defer to v0.8.0 if boundaries hold
- Separate git repos - Monorepo benefits outweigh cost
- Automated agent scoping - Just documentation for now
- Dashboard-specific CI pipeline - Single pipeline with boundary checks is sufficient

## Timeline

**Day 1** (6 hours):
- Phase 1: Boundary enforcement script and CI integration
- Phase 2: ARCHITECTURE.md documentation

**Day 2** (4 hours):
- Phase 3: CODEOWNERS and release tracks
- Testing and verification

**Day 3** (4 hours, optional):
- Phase 4: Physical restructure (if proceeding)
- Or: Buffer for edge cases and documentation polish

**Total: ~10-14 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Physical restructure breaks imports | High | Defer to Phase 4, do boundary enforcement first |
| CODEOWNERS slows dashboard PRs | Medium | Start with notification-only, enable blocking later |
| Agents ignore CLAUDE.md guidance | Medium | Add to agent system prompts if needed |
| False positives in boundary check | Low | Whitelist legitimate cross-boundary imports (sdk/) |

## Related Documents

**Architectural context:**
- [design_docs/implemented/v0_2_0/when-to-switch-from-go.md](design_docs/implemented/v0_2_0/when-to-switch-from-go.md) - Original Go strategy
- [design_docs/implemented/v0_6_6/m-dashboard-dogfooding.md](design_docs/implemented/v0_6_6/m-dashboard-dogfooding.md) - Dashboard motivation

**Planned features affected:**
- [design_docs/planned/v0_7_0/m-stdlib-datetime.md](design_docs/planned/v0_7_0/m-stdlib-datetime.md) - Dashboard-driven stdlib feature

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Monorepo vs Polyrepo](https://monorepo.tools/) - Industry context
- External feedback that motivated this design

## Future Work

**v0.8.0+ potential enhancements:**
- Physical directory restructure if boundaries prove difficult to maintain
- Automated agent scope enforcement (beyond documentation)
- Separate CI pipelines if test times diverge significantly
- Dashboard deployment pipeline (if external users emerge)

---

**Document created**: 2026-01-15
**Last updated**: 2026-01-15
