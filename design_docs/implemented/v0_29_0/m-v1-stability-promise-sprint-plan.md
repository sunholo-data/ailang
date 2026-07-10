# Sprint Plan: M-V1-STABILITY-PROMISE — The 1.x Stable-Surface Promise

**Design doc:** [m-v1-stability-promise.md](m-v1-stability-promise.md) (committed d53d7d800, pushed to origin/dev)
**Sprint ID:** M-V1-STABILITY-PROMISE
**Target:** v1.0.0 · **Priority:** P0 (ratified v1.0 bar clause)
**Planned:** 2026-07-10, mission iteration 5 (headless)
**Planner model:** `claude-opus-4-8` (Claude Opus 4.8)

## Summary

Docs-only sprint that (M1) publishes the 1.x stability promise page defining Stable/Experimental/Internal tiers and enumerating the stdlib + CLI surface; (M2) rewrites `docs/LIMITATIONS.md` so every entry carries a live-verified repro + date and fixed items move to a "Resolved" section; (M3) fixes the four stale website version-promises and re-runs the sweep grep to zero. No parser/type/codegen code.

**Duration:** ~2.5 days (3 milestones)
**Dependencies:** None (m-diagnostic-coverage CI-fixture mechanism reusable but not required)
**Risk Level:** Low (docs-only; the only high-consequence surface is the *promise wording*, which is gated by human ratification at release, not at merge)

**EXECUTION ENVIRONMENT (non-negotiable):** Execute in an **isolated git worktree branched from `origin/dev`**, never the shared main working tree (a sibling agent shares this tree and can revert uncommitted edits / switch branches). Commit promptly. Open a PR into `dev`; docs CI (Docs-Deploy, Gate 3b) must be green before merge.

## Current Status Analysis

### Verification of design-doc premises (done during planning, 2026-07-10)

Every premise the plan is built on was re-checked at `v0.28.0-140-gd53d7d800`. Results — including
**discrepancies the executor must handle** — are in the "Premise Verification" section below. Headline:
the two "fixed" repros reproduce green (`5.85`, `[zero, ok, ok]`), all 4 website promises + 3 in-file
LIMITATIONS promises are present at the cited locations, but the **stdlib module count and the LIMITATIONS
file topology differ from the doc** and must be reconciled.

### Velocity

- Recent mission cadence is docs-heavy: iterations 3–4 (m-diagnostic-coverage) were docs/fixture sprints
  landing per-milestone in a day each. The velocity script finds no LOC/day metric (docs commits aren't
  LOC-instrumented), which is expected and fine — this sprint is bounded by *verification throughput*
  (running N repros live), not by code volume.
- Estimated capacity: comfortably within 2.5 days for one agent. The binding constraint is M2 (must
  execute every LIMITATIONS repro at HEAD and capture transcripts), not typing volume.

### Remaining from Design Doc

- 📋 M1 — Stability promise page (~250 lines new)
- 📋 M2 — LIMITATIONS.md accuracy pass (526 lines → likely shorter)
- 📋 M3 — vX-promise sweep (4 website + 3 in-file fixes)

## Premise Verification (repo reality vs design doc)

| # | Doc claim | Verified reality | Action for executor |
|---|-----------|------------------|---------------------|
| 1 | New page at `docs/docs/reference/stability.md` | ✅ `docs/docs/reference/` exists and holds the peer pages (`language-syntax.md`, `implementation-status.md`, `stdlib.md`, `limitations.md`). **Correct path.** | Create the page there. |
| 2 | Axioms link `/docs/references/axioms` | ✅ Correct — a SEPARATE `docs/docs/references/` (plural) dir holds `axioms.mdx`, `design-lineage.mdx`, `philosophical-foundations.mdx`. Both `reference/` and `references/` legitimately exist. | Do not "fix" the axioms link; do not put stability.md under `references/`. Two dirs are intentional. |
| 3 | `docs/LIMITATIONS.md` frozen at v0.13.0 (99f76ec7a), 526 lines | ✅ Confirmed: last substantive commit 99f76ec7a (v0.13.0, 2026-04-20), 526 lines. | Rewrite per entry policy. |
| 4 | **[DISCREPANCY]** doc treats LIMITATIONS as one file; Deferred #4 asks *whether* it's mirrored to the site | ⚠ It is ALREADY mirrored **and diverged**: `docs/docs/reference/limitations.md` (338 lines, front-matter "Last verified against v0.14.2") is a SEPARATE, independently-maintained copy, and it (not the root file) is the one in the website sidebar (`docs/sidebars.js:124`). The root `docs/LIMITATIONS.md` is NOT auto-published. | **Resolve Deferred #4 as: fix BOTH files.** The website copy is the public one and is *also* stale (v0.14.2). Recommend making the website copy the canonical published surface and either (a) rewriting both to match, or (b) reducing root to a stub pointer. Executor picks; note the choice in the PR. |
| 5 | "39 stdlib modules" | ⚠ Reality: **42** top-level `std/*.ail` modules (excluding `trace_test.ail`), **43** counting the subdir module `std/ai/streaming.ail`. (`ls std/*.ail | grep -v _test` = 42.) Note `smoke.ail` and `trace.ail` are utility modules; executor judges their tier. | Enumerate from `ls std/*.ail` (+ `std/ai/`), NOT the doc's "39". Acceptance = every module appears in exactly one tier. |
| 6 | "~40 CLI commands" | ⚠ `ailang --help` is noisy (subcommand/repl artifacts inflate a naive count to ~88). The clean top-level set is ~11 primary (`version run repl test watch check ai-check sandbox-check iface export-training replay`) plus families (`eval-*`, `pkg`/`add`/`lock`/`tree`, `messages`, `serve`/`server`/`serve-api`/`lsp`, `coordinator`/`chains`/`dashboard`/`observatory`, `docs`/`prompt`/`doctor`/`builtins`/`debug`/`editor`/`axioms`/`examples`/`search`, `install`/`publish`/`unpublish`/`pkg-docs`, `init`, `trace`/`budget`/`exec`/`compile`/`access-control`). | Tier by **top-level command / family**, not by raw `--help` line count. The `eval-*`, `debug`, `doctor`, `iface`, `coordinator`/eval-harness family are the doc's named Internal candidates. Acceptance = every top-level command appears in exactly one tier. |
| 7 | Repro A: poly-arith lambdas FIXED | ✅ `ailang check` clean; `ailang run --caps IO` → `5.85`. Pre-verified, goes to Resolved. | — |
| 8 | Repro B: match-in-HOF FIXED | ✅ `ailang check` → "No errors found!"; `ailang run --caps IO` → `[zero, ok, ok]`. Pre-verified, goes to Resolved. | — |
| 9 | 4 website promises at cited lines | ✅ All present (line numbers drift ±2 but content unambiguous): benchmarking.md ("M-EVAL2 planned for v0.3.0"), module_execution.mdx ("capability checks planned for v0.3.0 (M-R2)"), ai-api-integration.mdx ("JSON decoding is planned for v0.4.0"), why-ailang.mdx ("Empirical validation planned for v0.8.0"). | Fix all 4 per doc's retract-with-pointer approach. |
| 10 | Full sweep grep | ✅ Returns exactly the 4 website + 3 in-file LIMITATIONS promises + the legitimate roadmap page (`docs/docs/roadmap/index.md:50` "Planned for v0.29.0"). No hidden extras. | The roadmap-page hit is the allowlisted future-language location; acceptance grep should return only it. |
| 11 | roadmap page + README exist | ✅ `docs/docs/roadmap/index.md` and `README.md` both exist. | Link stability page from README + `docs/docs/intro.mdx` (intro is `.mdx`, not `.md`). |

**No premise blocks the sprint.** The discrepancies (#4, #5, #6) are scoping clarifications the executor absorbs, not design-freeze reopenings.

## Proposed Milestones

### Milestone 1: Stability promise page (~1 day, ~250 lines)

**Goal:** New `docs/docs/reference/stability.md` defining the three tiers, the 1.x compatibility semantics, and the enumerated stdlib + CLI surface as tier tables; linked from README + docs intro.

**Tasks:**
- Write tier definitions (Stable / Experimental / Internal) + 1.x semantics verbatim from the frozen Design-Freeze text (no re-derivation).
- Draft the **stdlib tier table** from `ls std/*.ail` (42 modules) + `std/ai/streaming.ail`, one-line evidence per module (examples/ + eval + docs coverage → Stable; niche/newer → Experimental; `debug`, utility → Internal per doc's named candidates). Mark genuinely-ambiguous rows `⚠ proposed`.
- Draft the **CLI tier table** by top-level command/family (see verification #6), one-line evidence each. `eval-*`, `debug`, `doctor`, `iface` JSON shape, coordinator/eval-harness → Internal.
- State the **Stable syntax set by reference** to the canonical prompt/reference version (do NOT re-enumerate syntax).
- Add header stamp `RATIFICATION: pending (Mark, at release)`.
- Link from `README.md` and `docs/docs/intro.mdx`; add to `docs/sidebars.js` under reference.

**Acceptance Criteria:**
- [ ] `docs/docs/reference/stability.md` renders in the docs build.
- [ ] Every module in `ls std/*.ail` (+ `std/ai/`) appears in exactly one tier.
- [ ] Every top-level CLI command/family appears in exactly one tier.
- [ ] Ratification stamp present; linked from README + intro + sidebar.

**Risks:** Promise text over-commits (Stable surface breaks in 1.x). Mitigation: ambiguous → Experimental (cheap direction to be wrong); human ratifies at release, not at merge.

### Milestone 2: LIMITATIONS.md accuracy pass (~1 day)

**Goal:** Every remaining LIMITATIONS entry carries a live repro + `ailang` transcript + verified-at date at HEAD; fixed entries move to a dated "Resolved" section with pointers; stale version-promises deleted. **Applies to BOTH** `docs/LIMITATIONS.md` and the diverged website copy `docs/docs/reference/limitations.md` (verification #4).

**Tasks:**
- Script the repros: run every entry's example live at HEAD, capture `ailang check`/`run` transcripts (keep in the PR).
- Move the ≥2 pre-verified fixed entries (poly-arith lambdas → `5.85`; match-in-HOF → `[zero, ok, ok]`) to "Resolved" with pointers (design-doc/changelog).
- Rewrite remaining genuine limitations to the template (Status / Verified-at / repro+output / Workaround); current syntax only (fix any retired `match … with |`).
- Delete the 3 in-file stale promises (`LIMITATIONS.md:309` Deferred to v0.4.2, `:336` planned v0.4.0+, `:490` refined in v0.4.0+).
- Reconcile the two files (verification #4): choose canonical (recommend website copy) + fix both; record the choice in PR.

**Acceptance Criteria:**
- [ ] 0 entries describe fixed behavior as broken; 0 in-file past-version promises remain.
- [ ] Every remaining entry has repro + transcript + verified-at date (PR includes the transcripts).
- [ ] Both `docs/LIMITATIONS.md` and `docs/docs/reference/limitations.md` are accurate + consistent.

**Risks:** Re-verification finds entries still broken but with changed behavior. Mitigation: that's a finding — update repro+date; genuinely-new bugs route to the backlog, not this sprint.

### Milestone 3: vX-promise sweep + CHANGELOG (~0.5 day)

**Goal:** Fix the 4 inventoried website promises; sweep grep returns only the roadmap page.

**Tasks:**
- Fix the 4 website promises (retract-with-pointer for the 3 shipped features; why-ailang.mdx "empirical validation" → pointer to the actual eval program/dashboard).
- Re-run the acceptance sweep grep; confirm only `docs/docs/roadmap/index.md` remains.
- `CHANGELOG.md` entry.

**Acceptance Criteria:**
- [ ] Sweep grep `grep -rniE "(planned for|coming in|will be (added|refined|fixed|implemented)|deferred to) (in )?v0\.[0-9]" docs/ README.md` returns only the roadmap page (grep output in PR).
- [ ] CHANGELOG entry present.

**Risks:** grep false-positives on legit roadmap language. Mitigation: roadmap page is the allowlisted location.

## Stretch (OPTIONAL — Deferred Decisions, NOT core milestones)

Per the mission requirement, these are optional stretch tasks. Do only if they fit without blowing the 2.5-day estimate; otherwise record as a follow-up line in the promise page. **Do not let these gate M1–M3.**

- **S1 — `make check-doc-promises` grep gate + CI wiring** (~30 lines). Include ONLY if implementable in ≤30 lines with zero false-positives on the roadmap page. Otherwise leave a one-line follow-up note.
- **S2 — Wire LIMITATIONS repros as CI fixtures** (reuse m-diagnostic-coverage's fixture mechanism). Nice-to-have only.

## Success Metrics

- Documentation: `docs/docs/reference/stability.md` (new), `docs/LIMITATIONS.md` + `docs/docs/reference/limitations.md` (rewritten), 4 website promise files fixed, `README.md` + `docs/docs/intro.mdx` + `docs/sidebars.js` (links), `CHANGELOG.md` (entry).
- Examples: N/A (no language feature; repro snippets live in the LIMITATIONS files).
- Tests: none new (docs-only); docs build green (CI Docs-Deploy, Gate 3b) on the PR.
- Sweep grep returns only the roadmap page.

## Dependencies

- None blocking. m-diagnostic-coverage's CI-fixture mechanism is reusable for S2 but not required.

## Open Questions / Ratification

- **RATIFICATION: pending (Mark, at release).** The promise *wording* and the per-module/per-command tier *assignments* require human ratification before the v1.0.0 tag — NOT before this sprint or merge (Design Freeze resolved this; the release is human-triggered anyway). The executor drafts headless; ambiguous tiers are marked `⚠ proposed`. This is a release-gate line-item to record in the #329 report, not an execution blocker.

## Notes / Handoff

- **Model attestation (planner):** this plan was produced by **`claude-opus-4-8`** (Claude Opus 4.8) — mission routing evidence.
- **Worktree isolation is mandatory** for execution (see EXECUTION ENVIRONMENT above). Branch from `origin/dev`, commit promptly, PR into `dev`.
- **Design freeze is CLEAN** — all three Design-Freeze items in the doc are resolved (`[x]`). Nothing PAUSES execution. The only human-gated item (ratification) is deliberately deferred to release, not to this sprint.
- Verified planning baseline: `v0.28.0-140-gd53d7d800`, `ailang` binary `v0.28.0-139-gece12eab9` (system PATH `/Users/voightkampff/go/bin/ailang`).
