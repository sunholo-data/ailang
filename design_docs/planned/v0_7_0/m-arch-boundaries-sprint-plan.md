# Sprint Plan: M-ARCH-BOUNDARIES (Phases 1–3)

**Design doc**: [m-arch-boundaries.md](m-arch-boundaries.md)
**Iteration**: 68 (V1 mission-control loop)
**Worktree**: `.claude/worktrees/iter68-arch-boundaries` (checked out at `origin/dev`, HEAD `d5e849abe`)
**Scope**: Phases 1–3 ONLY (Mark, 2026-07-14 strategic audit). Phase 4 physical restructure and dual release tracks are **explicitly out of scope**.
**Risk level**: Low
**Total estimate**: ~1.5 days (~10 hours), ~260 LOC (mostly docs + a small script + config)

---

## Goal

Give the v1.0 stability promise a mechanical boundary to scope to, without moving a single file. Deliver exactly three things:

1. **A boundary-enforcement CI gate** — `scripts/check_boundaries.sh` + `make check-boundaries` + a CI step.
2. **Boundary docs** — augment the existing `ARCHITECTURE.md`, add an agent-boundary section to `CLAUDE.md`.
3. **CODEOWNERS** — `.github/CODEOWNERS` on the **existing** `internal/` layout.

The logical layers are enforced/documented over the **current physical tree**. No `core/`/`apps/` directories are created (that is Phase 4, deferred to the v1.0→v1.1 boundary).

---

## Reality reconciliation (premises corrected during planning)

The design doc predates the current tree. Verified against HEAD in the worktree:

| Doc / adaptation claim | Reality | Action |
|---|---|---|
| Create `ARCHITECTURE.md` | **Already exists** (committed `fa64b79e2`, CII silver-tier prep) | **Augment**, don't create. Real LOC ≪ doc's ~200. |
| Create `CONTRIBUTING.md` | **Already exists** | Light touch only if a natural hook exists. |
| `core/` and `apps/` dirs (doc "Proposed State") | **Do not exist** | Gate/CODEOWNERS/docs target real `internal/` paths only. |
| Core includes `internal/typeclass` (controller note) | **No such directory** | Drop `typeclass` from the core set. Type classes live in `internal/types` + `internal/dispatch`. |
| `internal/dictionary` (referenced in current ARCHITECTURE.md ~L69) | **No such directory** | Fix/drop that stale line while editing (M2). |
| Doc Example-1 `grep -r '(a|b)'` (no `-E`, un-anchored) | **Never matches** | Script uses `grep -rE` / ripgrep against quoted Go import paths. |
| core→dashboard imports = 0; dashboard→core direct = 0 | **Confirmed at HEAD** (dashboard reaches compiler via `internal/embed` + `internal/runtime`) | Gate **passes** on current tree. |

**Confirmed real layout:**
- **CORE**: `internal/{parser,types,eval,core,elaborate,effects,builtins,lexer,ast,pipeline,runtime,link,iface}`
- **DASHBOARD (apps)**: `internal/{server,coordinator,observatory,messaging}` (+ `ui/`)
- **TOOLS**: `internal/{eval_harness,eval_analysis,ai}`
- **BRIDGE (sdk)**: `internal/embed` (+ `internal/runtime`, `internal/schema`)

---

## Milestones

### M1 — Boundary enforcement gate (~4h, ~90 LOC)

**Files**
- `scripts/check_boundaries.sh` (new, ~60 LOC) — `set -euo pipefail`; correct import matching via `grep -rE` / ripgrep on `"github.com/sunholo/ailang/internal/<pkg>"`.
- `Makefile` — add `check-boundaries` target (~6 LOC).
- `.github/workflows/ci.yml` — add a gate step named e.g. *"Check architecture boundaries"* running `make check-boundaries`, placed right after the `check-file-sizes` step (~ci.yml L89).

**Rules the script enforces**
1. No CORE package imports any DASHBOARD package.
2. No DASHBOARD package imports `parser/types/eval/core/elaborate/pipeline` directly (must go through `internal/embed`).

**Acceptance**
- Executable; exits `0` on current HEAD tree.
- Matching is anchored on quoted Go import paths (not the doc's broken naive grep).
- Emits offending `file:line` + clear `BOUNDARY VIOLATION` message on failure.
- **Self-test proven**: plant a violating import into a core file → script exits non-zero → **fully revert the plant** → script exits `0` again. (Prove both directions; leave the tree clean.)
- `make check-boundaries` invokes it; CI step runs after `check-file-sizes`.
- `go build ./...` and `make check-file-sizes` still pass.

**Risk**: Low. Script does not mutate code. Only real risk is a mis-written matcher (false pass) — mitigated by the mandatory self-test.

---

### M2 — Boundary docs (~2–3h, ~120 LOC)

**Files**
- `ARCHITECTURE.md` (augment) — add an **"Architecture boundaries"** section: the four logical layers mapped to real paths, the allow/deny import table, and a pointer to `make check-boundaries`. Also fix the stale `internal/dictionary` reference.
- `CLAUDE.md` (augment) — add a concise **"Architecture boundaries (agents)"** subsection: which subsystem you're touching, the never-cross-import rule, run `make check-boundaries` before committing cross-cutting changes.
- `CONTRIBUTING.md` — optional light touch only.

**Allow/deny table to document**

| Direction | Allowed | Enforced by |
|---|---|---|
| apps → core | NO (only via `internal/embed`) | `check_boundaries.sh` |
| core → apps | NO | `check_boundaries.sh` |
| tools → core | YES | — |
| tools → apps | NO | (documented; extendable) |
| bridge (embed) → core | YES (controlled) | design |

**Acceptance**
- Docs describe **logical** layers over the **existing** `internal/` tree — must not claim a physical `core/`/`apps/` dir exists.
- No `core/doc.go` or `apps/doc.go` created (Phase 4 non-goal); boundary-comment intent folded into ARCHITECTURE.md.
- Stale `internal/dictionary` path fixed/dropped.
- Links valid; any docs lint still passes.

**Risk**: Low (docs only).

---

### M3 — CODEOWNERS (~1h, ~40 LOC)

**Files**
- `.github/CODEOWNERS` (new, ~35 LOC) — path-based ownership on the **existing** `internal/` layout.

**Ownership (notification-only; teams are placeholders)**
- Core (`@ailang/compiler`): `/internal/parser/ /internal/types/ /internal/eval/ /internal/core/ /internal/elaborate/ /internal/effects/ /internal/builtins/ /internal/lexer/ /internal/ast/ /internal/pipeline/ /internal/runtime/ /internal/link/ /internal/iface/ /stdlib/`
- Dashboard (`@ailang/dashboard`): `/internal/server/ /internal/coordinator/ /internal/observatory/ /internal/messaging/ /ui/`
- Bridge (both teams): `/internal/embed/`

**Acceptance**
- Every path corresponds to a directory that actually exists (no `core/`/`apps/` prefixes).
- Valid CODEOWNERS syntax.
- Team handles documented as placeholders a maintainer must create/rename; notification-only, non-blocking (matches doc risk mitigation).

**Risk**: Low. If team handles don't exist yet, GitHub simply ignores them — no merge blocking.

---

## Day-by-day

**Day 1 (~6h)**
- M1: write + self-test the gate script, add Makefile target, wire CI. Prove fail-then-pass, leave tree clean.
- M2: augment ARCHITECTURE.md + CLAUDE.md; fix stale `dictionary` reference.

**Day 1.5 (~4h)**
- M3: CODEOWNERS.
- Full verification: `make check-boundaries`, `make check-file-sizes`, `go build ./...`; confirm CI step ordering; commit.

---

## Success metrics
- `make check-boundaries` passes on the current tree.
- Gate proven to fail on a planted violation, then reverted (self-test evidence in the executor report).
- ARCHITECTURE.md + CLAUDE.md document the boundaries accurately (logical, not physical).
- `.github/CODEOWNERS` valid, real-path ownership.
- No file moves, no new layer dirs, no release-track changes.

## Non-goals (Deferred — DO NOT implement)
- **Phase 4 physical restructure**: no `git mv`, no `core/`/`apps/` dirs, no import-path rewrites, no `core/doc.go`/`apps/doc.go`. → v1.0→v1.1 boundary.
- **Dual release tracks**: no `make release-dashboard`, no `dashboard/CHANGELOG.md`, no `release-manager` skill changes.
- Automated agent scope enforcement beyond docs.
- Separate git repos (reaffirmed REJECTED).

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_7_0/m-arch-boundaries-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-ARCH-BOUNDARIES.json`
