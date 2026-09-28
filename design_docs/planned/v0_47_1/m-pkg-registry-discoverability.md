# M-PKG-REGISTRY-DISCOVERABILITY — the registry as the first stop

**Status**: Planned
**Target**: v0.47.1
**Priority**: P1 (Medium)
**Estimated**: 3 phases, ~2.5 days total (Phase 1: 0.5 d · Phase 2: 1.25 d · Phase 3: 0.5 d)
**Dependencies**: None (builds on M-PKG-QUALITY-LADDER Sprint 1, shipped v0.40.0; independent of its planned Sprints 2–3)

> **Terminology note.** The request says "agents.md". AILANG's package-level
> convention is **`AGENT.md`** — the file `ailang pkg docs` displays, the
> tarball includes, and the registry banks (V1). The repo-root `AGENTS.md`
> convention (repo guidance for coding agents) is a different file. This doc
> uses `AGENT.md` throughout; implementers must not introduce a second
> convention.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All new checks are pure string/set comparisons over static inputs; the quality report stays byte-identical for identical inputs on publisher and server (the existing seam doctrine) |
| A2: Replayability | 0 | No trace/runtime changes |
| A3: Effect Legibility | 0 | No language-surface changes; effect data is only *read*, never re-ranked (EffectPrivilegeRank untouched) |
| A4: Explicit Authority | 0 | No new capabilities; registry reads are unauthenticated public data, same as today |
| A5: Bounded Verification | +1 | New checks are static and bounded (no execution, no Z3, no network on the server for D2 — except the already-required index) |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | D1 surfaces the *machine-facing* AGENT.md to every consumer; D3 makes agent planners check the registry before writing code; D2 feedback is structured `PUBnnn` findings, not prose |
| A8: Minimal Syntax | +1 | No new language syntax; zero new surface in `.ail` files |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | +1 | The entire point: reuse existing packages instead of parallel implementations |
| A11: Structured Failure | +1 | New findings carry allocated error codes (`PUB018` etc.) with actionable messages, in the existing report schema |
| A12: System Boundary | 0 | No new boundary crossings; website fetches the same public GCS artifacts it already snapshots |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — all checks are deterministic functions of static inputs
- [x] A3 (Effects): No hidden side effects — nothing executes package code
- [x] A4 (Authority): No ambient access granted — read-only public registry data
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis — AGENT.md is the machine-facing guide; planners (agents) are the primary beneficiaries

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

The AILANG package registry exists, validates, and version-controls packages — but it is
invisible at exactly the three moments it should be the loudest: on the docs website, in
publish feedback, and in sprint planning.

**Current State:**

1. **The website hides the agent guide.** Every published version uploads its `AGENT.md`
   to GCS as a first-class artifact (`packages/<vendor>/<name>/<version>/AGENT.md`, V1),
   and the index carries `has_agent_doc` — but the generated package pages
   (`docs/scripts/sync-registry.sh`) render only `ai_summary`, a fields table, exports and
   provenance. `has_agent_doc` is never even read by the page generator (V3). A human or
   agent browsing `ailang.sunholo.com/docs/packages/<vendor>/<name>` cannot see the one
   document written to explain the package.
2. **Quality feedback stops too early.** The v0.40.0 quality report (M-PKG-QUALITY-LADDER
   Sprint 1) checks 13 codes (PUB000–PUB021 minus reserved gaps, V7) — but three
   author-facing gaps are measured-but-silent or unmeasured: a missing `ai_summary` is
   banked in `DocsSection.AISummary` yet produces **no finding at all** (V8); an AGENT.md
   that exists but mentions none of the package's exported modules (stale guide) is
   indistinguishable from a good one (V9); and a package that re-implements an existing
   package's exports gets no signal — the author finds out from nobody.
3. **Sprint planners never look.** The sprint-planner skill (both the `.agents/` Codex
   variant and the `.claude/` Claude variant) plans milestones from design docs and
   velocity — it contains **zero** references to the registry, `ailang pkg search`, or
   package reuse (V6). A sprint that re-implements `sunholo/auth` from scratch is
   procedurally indistinguishable from one that legitimately needs new code. The registry
   is the system's compounding-returns asset; planners bypassing it re-spend the
   principal every sprint.

**Impact:**

- Package authors get no feedback on discoverability-relevant gaps (`ai_summary`,
  license, stale AGENT.md, duplicate functionality) until a human happens to notice.
- Agents and humans evaluating a package must leave the website to read its guide
  (`ailang pkg docs` requires installing/caching the tarball first).
- Every mission/sprint that could have depended on an existing package instead
  duplicates it — measured friction: the coordinator message plane already routes
  `pkg:` inboxes per package (autonomous-package-updates), so a duplicated package also
  fragments its feedback channel.

## Goals

**Primary Goal:** Make the registry the first stop — visible on the website, loud in
publish feedback, and mandatory in sprint planning — so existing packages are found and
reused before new code is written.

**Success Metrics:**

- Every package detail page on `ailang.sunholo.com` renders its `AGENT.md` (build-time
  snapshot, same-origin static fetch) — 100% of packages with `has_agent_doc: true`.
- `ailang pkg quality .` emits findings for: missing `ai_summary` (PUB018), missing
  license link (PUB008), AGENT.md that references none of the package's exported modules
  (PUB022), and export overlap with an existing registry package (PUB023) — with the
  publisher and validator assembling identical `server` sections (seam test).
- Both sprint-planner skill variants contain a mandatory registry-reuse gate step, and
  the sprint JSON template carries a `registry_reuse` block the executor can verify.
- No new npm dependencies; Docusaurus build stays MDX-injection-safe.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: AGENT.md rendered as **raw markdown text** in a styled panel, not parsed/rendered markdown | AGENT.md is untrusted package-author content; MDX parses bare `<`/`{` and can break the whole site build (measured: Docs-Deploy 2026-07-23 `ai_summary` incident, V10). No markdown renderer dep exists today | agent | design | low |
| D2: Build-time static snapshot (sync-registry.sh fetch from public GCS) as the data path; live API endpoint deferred | The GCS artifact already exists and the sync script already snapshots registry data; a new API endpoint adds server surface for marginal freshness | agent (endpoint deferred for human to prioritize) | design | med |
| D3: All four new checks are **badge-level feedback at every stability** (never hard gates here); `--strict` still promotes warns as today | The ladder's philosophy: feedback first, gates by stability. These are discoverability hints, not integrity failures — blocking on them would stall the ecosystem | agent | design | low |
| D4: PUB023 overlap check runs on **both** publisher and server from the same index; publisher network failure yields an explicit "not run" badge, never silence | Server/publisher identical `server` sections is the existing seam doctrine; silent skip violates no-silent-fallbacks (CLAUDE.md principle 2) | agent | design | med |
| D5: Error-code allocation PUB008, PUB018, PUB022, PUB023 (verified unallocated, V7) | PUB codes are a shared namespace; a collision mislabels an existing/planned diagnostic (PUB003/004/013/017 are reserved by planned docs) | agent (verified free) | design | low |
| D6: The planner gate is procedural (SKILL.md + sprint JSON field), not a new CLI enforcement tool in this doc | Keeps scope at stated change scale; tool enforcement (`ailang pkg audit-plan`) is plausible future work once the process is measured | agent | design | low |
| D7: Update **both** skill variants (`.agents/` Codex + `.claude/` Claude) with the same gate, preserving each harness's paths | The two SKILL.md files intentionally differ in harness names (V6); editing one leaves the other lane's planners unguarded | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — raw-text rendering (frozen: build safety)
- [x] D2 — static snapshot first, live endpoint deferred (frozen for this doc)
- [x] D3 — badges, never gates, in this doc (frozen)
- [x] D4 — identical publisher/server overlap check with loud not-run badge (frozen)
- [x] D5 — PUB008/PUB018/PUB022/PUB023 allocation (frozen: verified free)
- [ ] D2-live — *deferred*: `GET /api/packages/:vendor/:name/agentmd` endpoint (human may pull it into scope or leave deferred)

## Solution Design

### Overview

Three independently shippable deliverables sharing one theme — the registry as the
first stop:

1. **D1 (website):** the registry sync script snapshots each package's `AGENT.md`
   alongside its JSON; the package detail page fetches the snapshot and shows it.
2. **D2 (quality feedback):** four new badge-level findings in the ONE quality report
   (`internal/pkg/quality.go`), measured on both sides of the publisher/server seam.
3. **D3 (planner gate):** a mandatory "check the registry" step in both sprint-planner
   skill variants, with a `registry_reuse` block in the sprint JSON handoff.

### Architecture

**Components:**

1. **Snapshot layer (D1)** — `docs/scripts/sync-registry.sh` already iterates every
   package with `name` and `latest` in hand; for each it fetches
   `$REGISTRY_GCS/packages/<vendor>/<name>/<latest>/AGENT.md` to
   `docs/static/registry/<vendor>/<name>/AGENT.md`. Failure modes follow the script's
   existing graceful pattern (warn + skip), EXCEPT the skip is logged per-package so a
   missing AGENT.md is visible in CI output, not silent.
2. **Render layer (D1)** — `PackageDetail.jsx` gains an "Agent Guide (AGENT.md)" section:
   a same-origin `fetch('/registry/<vendor>/<name>/AGENT.md')`, rendered inside a
   scrollable `<pre>` panel with a monospace font and a collapse toggle. No markdown
   parsing, no `dangerouslySetInnerHTML`, no new deps. A 404 renders the placeholder
   "This package ships no AGENT.md (PUB020 flags this at publish)."
3. **Quality layer (D2)** — new `QualityInputs` fields (`AgentDocContent`, plus
   `Overlap []string` and `OverlapErr string`), new findings in `BuildQualityReport`,
   measurement wired in `measurePackageQuality` (publisher) and the validator's report
   assembly (server, `cmd/registry-validator/main.go:330` call site). New
   `DocsSection`/report fields are omitempty so v1 readers interoperate.
4. **Process layer (D3)** — SKILL.md edits (both variants) + `create_sprint_json.sh`
   template gains a top-level `registry_reuse` array; the sprint-plan template resource
   gains a "Registry Reuse Audit" section; the planner handoff JSON message carries the
   audit result.

### Implementation Plan

**Phase 1: Website AGENT.md (D1)** (~4 hours)

- [ ] `docs/scripts/sync-registry.sh`: in the per-package loop, curl
      `packages/<vendor>/<name>/<latest>/AGENT.md` → `docs/static/registry/<vendor>/<name>/AGENT.md`;
      `--fail --max-time 10`; on failure print `  ⚠ no AGENT.md snapshot for <name> (<reason>)` and continue
- [ ] `docs/src/components/PackageExplorer/PackageDetail.jsx`: new section above
      "Exported Modules" — fetch snapshot, collapse toggle (default collapsed above
      ~2 KB, expanded below), `<pre>` render, 404 placeholder
- [ ] `docs/src/components/PackageExplorer/styles.module.css`: `agentDocPanel`
      (monospace, max-height ~480px, overflow auto) + toggle button style
- [ ] `docs/docs/packages/index.mdx`: one sentence in "How It Works" — package pages
      include the package's AGENT.md agent guide
- [ ] Verify: run the sync script against the live registry, `npm run build` in `docs/`,
      confirm a package with AGENT.md renders and a package without degrades gracefully

**Phase 2: Quality checks (D2)** (~10 hours)

- [ ] `internal/pkg/quality.go`: extend `QualityInputs` (`AgentDocContent string`,
      `Overlap []string`, `OverlapErr string`); extend `DocsSection` with
      `AgentDocExportsCovered int` (omitempty JSON); four findings in
      `BuildQualityReport` (see table below)
- [ ] `cmd/ailang/pkg_quality.go` `measurePackageQuality`: read AGENT.md content
      (replacing the bare `os.Stat`); fetch registry index via
      `pkg.RegistryClient.FetchIndex()` and compute overlap → `Overlap`/`OverlapErr`
- [ ] `cmd/registry-validator/main.go`: same two measurements server-side — AGENT.md
      from the extracted tempDir, overlap from `cache.GetIndex`
- [ ] `internal/pkg/registry.go`: `ExportOverlap(index *RegistryIndex, name string, exports []string) []string` — names of OTHER packages whose `exports` intersect this one's, sorted (determinism)
- [ ] Tests: `quality_test.go` (each finding fires/doesn't, strict promotion, determinism),
      `registry_test.go` (`ExportOverlap` incl. self-exclusion and no-dependents),
      `pkg_quality_test.go` (measurement wiring), validator-side assembly test mirroring
      `quality_shadow_test.go`
- [ ] `docs/docs/guides/package-publishing.md`: add the four codes to the PUB reference
      with one-line author actions; `docs/docs/guides/packages.md`: link the website
      AGENT.md panel from the docs section
- [ ] CLI help text: `ailang pkg quality --help` "Sections:" line unchanged (findings,
      not sections — no edit needed; verify)

**Phase 3: Sprint-planner reuse gate (D3)** (~4 hours)

- [ ] `.agents/skills/sprint-planner/SKILL.md` AND `.claude/skills/sprint-planner/SKILL.md`:
      new numbered step **"0. Check the registry (mandatory reuse gate)"** before
      "1. Read and Analyze Design Document" — for every planned capability that could
      plausibly be a package (auth, storage adapters, parsing, formatting, CLI helpers…):
      run `ailang pkg search <keywords>`, `ailang pkg info <vendor/name>`,
      `ailang pkg docs <vendor/name>`; classify each milestone as
      `depend` (covered — plan `ailang install` + version pin, no new implementation),
      `contribute` (partially covered — plan the upstream `pkg:<name>` inbox milestone
      before fresh code), or `none` (no overlap — proceed fresh, record the audit)
- [ ] Same files: add "Registry Reuse Audit" to the Analysis Framework checklist and
      require the audit result in the sprint plan document (template resource)
- [ ] `.agents/skills/sprint-planner/scripts/create_sprint_json.sh` +
      `.claude/...` variant: emit `"registry_reuse": []` placeholder; planner MUST
      populate it (add to the existing placeholder-validation checklist)
- [ ] Handoff message template in SKILL.md: include `"registry_reuse"` summary in the
      `ailang agent send sprint-executor` JSON

**New quality findings (all badge-level; `--strict` promotes warns exactly as today):**

| Code | Level | Fires when | Author action |
|------|-------|-----------|---------------|
| PUB018 | warn | `[metadata] ai_summary` absent — it is the index card on the website and the `ailang pkg search` preview line, and today its absence is banked silently (V8) | Add a one-line `ai_summary` to `[metadata]` |
| PUB008 | info | `[metadata] license_url` absent — the website's License link then goes nowhere for this package | Add `license_url` |
| PUB022 | warn | AGENT.md exists but references **zero** of the package's exported modules — the guide predates or ignores the interface; agents reading it get no usage guidance for what actually ships | Mention each exported module in AGENT.md (a header or example per module) |
| PUB023 | info | This package's exported module names intersect another registry package's — likely parallel functionality; `OverlapErr` non-empty instead ⇒ badge "overlap check not run: <reason>" (never silent) | Depend on the existing package, or state the distinction in `ai_summary`; on overlap the author decides, the registry informs |

Overlap definition (deterministic): a package P overlaps Q when at least one exported
module name of P equals one of Q's (exact string, the `exports` lists both sides already
bank in the index). PUB023 names the overlapping packages (sorted, capped at 3 + "and N
more") so the author can inspect each.

### Files to Modify/Create

**New files:** none (no new Go files, no new components — one new section inside an existing component).

**Modified files:**
- `docs/scripts/sync-registry.sh` — per-package AGENT.md snapshot (~15 LOC)
- `docs/src/components/PackageExplorer/PackageDetail.jsx` — Agent Guide section (~60 LOC)
- `docs/src/components/PackageExplorer/styles.module.css` — panel + toggle styles (~25 LOC)
- `docs/docs/packages/index.mdx` — one sentence (~2 LOC)
- `internal/pkg/quality.go` — inputs, DocsSection field, four findings (~70 LOC)
- `internal/pkg/registry.go` — `ExportOverlap` (~30 LOC)
- `cmd/ailang/pkg_quality.go` — measure AGENT.md content + index overlap (~25 LOC)
- `cmd/registry-validator/main.go` — server-side assembly inputs (~15 LOC)
- `internal/pkg/quality_test.go`, `registry_test.go`, `cmd/ailang/pkg_quality_test.go`, `cmd/registry-validator/quality_shadow_test.go` — tests (~180 LOC)
- `docs/docs/guides/package-publishing.md`, `docs/docs/guides/packages.md` — PUB reference + docs section (~20 LOC)
- `.agents/skills/sprint-planner/SKILL.md`, `.claude/skills/sprint-planner/SKILL.md` — step 0, checklist, handoff (~60 LOC each)
- `.agents/skills/sprint-planner/scripts/create_sprint_json.sh`, `.claude/...` variant — `registry_reuse` block (~10 LOC each)
- `.agents/skills/sprint-planner/resources/sprint_plan_template.md`, `.claude/...` variant — "Registry Reuse Audit" section (~15 LOC each)

## Examples

### Example 1: A package page before/after (D1)

**Before** (`docs/docs/packages/sunholo/auth.mdx`, generated): a one-line `ai_summary`
blockquote, a fields table, exports, dependencies, version timeline. `has_agent_doc: true`
in the snapshot JSON is unused. To learn *how* to use the package you must
`ailang install` it first.

**After:** same page, plus:

```
┌─ Agent Guide (AGENT.md) ──────────────────── [show] ─┐
│ # sunholo/auth                                         │
│                                                        │
│ Capability-scoped authentication keys. Import:        │
│   import pkg/sunholo/auth/keys (mintKey, revokeKey)    │
│ ...                                                    │
└────────────────────────────────────────────────────────┘
```

Snapshot served same-origin from `/registry/sunholo/auth/AGENT.md`; fetched by the
component at render time; no MDX parsing of its content.

### Example 2: Quality report before/after (D2)

**Before** (`ailang pkg quality .` on a package with no `ai_summary` and a stale AGENT.md):

```
  docs:      AGENT.md true  ai_summary false
  ...
  ✓ no gates
```

`ai_summary: false` — silent. Nothing actionable.

**After:**

```
  docs:      AGENT.md true  ai_summary false  exports_covered 0/3
  ...
  · PUB018 ai_summary missing — it is your package's index card on the website and in
    `ailang pkg search`; add [metadata] ai_summary = "..."
  ⚠ PUB022 AGENT.md references 0 of 3 exported modules — agents get no usage guidance
    for what actually ships; mention each module (keys, roles, rotate)
  · PUB023 exports overlap sunholo/gcp-auth (keys) — depend on it or say why not
```

### Example 3: Sprint-planner step 0 (D3)

**Before:** planner reads design doc → velocity → milestones. A design doc milestone
"token refresh helper" becomes "implement token refresh (~200 LOC)" without any registry
consultation.

**After:** Step 0 runs first:

```bash
$ ailang pkg search "token auth keys"
sunholo/auth@0.2.0 — Capability-scoped auth keys [Net, Env]
sunholo/gcp-auth@0.1.0 — GCP OIDC token minting [Net]
$ ailang pkg docs sunholo/auth | head   # read the AGENT.md before planning
```

→ milestone classified `depend`: `ailang install sunholo/auth@0.2.0`, version-pin in
`ailang.toml`, ~0 LOC instead of ~200; recorded in the sprint JSON:

```json
"registry_reuse": [
  {"package": "sunholo/auth", "action": "depend", "reason": "covers mint/revoke; rotate planned as upstream contribution via pkg:sunholo/auth inbox"}
]
```

## Success Criteria

- [ ] A docs-site build with live registry data renders the AGENT.md section on every package page that has one, and the graceful placeholder on pages that don't (acceptance: run `docs/scripts/sync-registry.sh` + `npm run build`, inspect 2 generated packages)
- [ ] `ailang pkg quality --json .` on a fixture package with the four gaps emits PUB008/PUB018/PUB022/PUB023; the same fixture clean emits none (acceptance: table-driven test in `quality_test.go`)
- [ ] Publisher and server assemble identical `server` sections including the new fields (acceptance: seam test mirroring `quality_shadow_test.go`)
- [ ] `ExportOverlap` excludes the package itself, is sorted, and `OverlapErr` produces the not-run badge — never silence (acceptance: `registry_test.go`)
- [ ] Both sprint-planner SKILL.md variants contain the step-0 gate with identical semantics and harness-appropriate paths (acceptance: `diff` of the two step-0 blocks differing only in harness names)
- [ ] `create_sprint_json.sh` emits `registry_reuse` and the placeholder-validation checklist rejects an unpopulated one (acceptance: run the script, then `jq`)
- [ ] All tests passing (`make test`), `make fmt`/`make lint` clean, `make check-boundaries` clean (no layer changes)
- [ ] Documentation updated (package-publishing.md PUB table, packages.md, packages/index.mdx)
- [ ] No new npm dependencies (acceptance: `git diff docs/package.json` empty)

## Testing Strategy

**Unit tests:**
- Each new finding: fires on the gap, silent on the clean fixture, `--strict` promotes warn-level (PUB018, PUB022) to gates, info (PUB008, PUB023) stays informational — matches existing `finding()`/`strict` semantics
- `ExportOverlap`: self-exclusion, multi-package overlap sorted + capped, empty index, no exports
- AGENT.md coverage: module mention counted by substring on exported module names; 0/3 vs 3/3

**Integration tests:**
- `measurePackageQuality` with a fixture package directory: AGENT.md content read (not just stat), overlap from a stubbed index (httptest registry, mirroring PUB017's test pattern)
- Validator assembly: server-side `BuildQualityReport` gets the same inputs → identical JSON `server` sections (the existing seam doctrine)
- Sync script: run against a local fixture GCS layout; assert snapshot files exist, and the script exits 0 when AGENT.md is absent for some packages

**Manual testing:**
- `cd docs && npm run build` with real registry data; inspect a package page with and without AGENT.md; verify the collapse toggle and the 404 placeholder
- Run `ailang pkg quality .` in a real package (`sunholo/auth` checkout if available) and read the human output

## Verification Log

Every load-bearing claim above was checked against the live code this session (the
design-doc-creator hard gate):

| # | Claim | How verified |
|---|-------|--------------|
| V1 | AGENT.md is uploaded per version to `packages/<vendor>/<name>/<version>/AGENT.md` as `text/markdown` | `cmd/registry-validator/main.go:393-401` (upload block; logged "first-class artifact", M-PKG-AUTONOMOUS-UPDATES) |
| V2 | The registry API has no AGENT.md route: only `/api/packages`, `/api/packages/…[/version]`, `/api/stats` | `cmd/registry-validator/main.go:91-93` (HandleFunc registrations); `handlers_api.go` in full |
| V3 | Generated package pages show only `ai_summary` as narrative; `has_agent_doc` is never read by `sync-registry.sh` | Read `docs/scripts/sync-registry.sh` end-to-end — no `has_agent_doc` reference; card/blockquote fields enumerated |
| V4 | `ailang pkg search <query>` exists and prints `name@latest — ai_summary [effects]` | `cmd/ailang/commands_pkg.go:59`; `cmd/ailang/pkg_search.go:61` |
| V5 | `ailang pkg docs <pkg>` displays a package's AGENT.md (cache/tarball) | `cmd/ailang/commands_pkg.go:25`; `cmd/ailang/pkg_docs.go:13-95` |
| V6 | Both sprint-planner SKILL.md variants lack any registry/reuse step; the variants differ only in harness names (`.Codex/` vs `.claude/`, "Codex" vs "Claude") | `grep -n "registry\|package" .agents/skills/sprint-planner/SKILL.md` → 0 hits; `diff .agents/... .claude/...` → 16 lines, all harness-name substitutions |
| V7 | PUB codes allocated (live): 000,001,002,005,006,010,011,012,014,015,016,020,021; reserved by planned docs: 003,004 (m-pkg-quality-ladder Sprint 2), 013 (ladder), 017 (m-publish-path-dep-smoke); PUB007/008/009/018/019/022/023 grep to **zero hits** across `internal/ cmd/ docs/ design_docs/ changelogs/ .agents/` | `grep -rhoE "PUB[0-9]{3}" … \| sort -u` + per-code `grep -rl` (each candidate: 0 files) |
| V8 | Missing `ai_summary` produces no finding today — banked in `DocsSection.AISummary`, silent | `internal/pkg/quality.go:293` — `r.Docs = DocsSection{…}` with no badge/gate on `!hasSummary`; read `BuildQualityReport` in full |
| V9 | AGENT.md is existence-checked (`os.Stat`), content never examined; PUB020 fires only on absence | `cmd/ailang/pkg_quality.go:133`; `internal/pkg/quality.go:294-296` |
| V10 | The docs site has no markdown-renderer dependency; MDX-injection from package text is a measured failure (2026-07-23 Docs-Deploy) | `docs/package.json` dependency list; sanitization comment + incident note in `sync-registry.sh` |
| V11 | The validator holds the full registry index in memory — overlap is computable server-side without new fetches | `cmd/registry-validator/cache.go:54` (`GetIndex`); `handlers_api.go` `handleAPIPackages` |
| V12 | Publisher and server both call `BuildQualityReport`; server call site is `main.go:330`, publisher measurement `measurePackageQuality` (`pkg_quality.go:87`, called from `pkg_publish.go:109`) | `grep -rn "BuildQualityReport" cmd/registry-validator/` + read both call paths |
| V13 | `EffectPrivilegeRank` and `ReleaseGateHardFrom` are NOT touched by this design | Read `quality.go` — this doc adds findings only; effects/release/rank logic unchanged |
| V14 | The quality-ladder Sprints 2–3 (planned) cover intent-consistency (PUB003/004), frozen-class (PUB013) and `effects_measured` over-declaration — this doc adds none of those, only author-feedback badges | `design_docs/planned/v0_40_0/m-pkg-quality-ladder.md` §Sprint 2/3 and PUB table (lines 147-149, 191-197) |

**Negative-existence sweep** (per the design-doc-creator rule for "no X exists" claims):
- "No AGENT.md route exists" → V2 (route registrations enumerated).
- "No markdown renderer dep exists" → V10 (dependency list read).
- "`has_agent_doc` is never read by the page generator" → V3 (full script read).
- "No finding on missing ai_summary" → V8 (full function read).
- "PUB008/018/022/023 unallocated" → V7 (grep across all doc/code surfaces).
- "No registry check in planner skills" → V6 (grep 0 hits on both variants).

## Conflict Surface Analysis

This doc does not touch `internal/parser/`, `internal/types/`, `internal/codegen/`,
`internal/eval/` or `cmd/ailang/exec.go` — the mandatory parser/conflict gate does not
apply. Two surfaces it *does* extend are enumerated here because they are shared:

1. **`ailang.package-quality/v1` JSON schema.** Existing readers (validator cache,
   `pkg versions`, dashboards) decode with Go structs — unknown fields are ignored;
   all new fields are omitempty. `DocsSection` gains `agent_doc_exports_covered`;
   findings are appended to the existing `badges` array. No existing code index
   changes. Programs/tests that assert on exact badge lists per fixture WILL need the
   new fixtures — the four findings fire only on gaps, and current tests' fixtures
   already avoid the gaps where they assert absence.
2. **Registry API surface.** No routes change in this doc (the optional live
   AGENT.md endpoint is explicitly deferred to Design-Freeze D2-live). The API
   remains GET-only with the same CORS policy.
3. **Docs-site static layout.** `docs/static/registry/<vendor>/<name>/` already holds
   `index.json` per package (V3); adding `AGENT.md` beside it introduces no new
   path-convention collision. MDX pages themselves are unchanged in structure —
   the new section is inside the existing `<PackageDetail>` component, so no new
   generated-MDX sanitization surface opens (D1 keeps author content out of MDX).

## Deferred Decisions

The following are intentionally left open:

- **Live AGENT.md hydration** (`GET /api/packages/:vendor/:name/agentmd` proxying GCS,
  `registryCache` entry, CORS already covers the docs origin) — human may pull into
  scope; otherwise Phase 2 of some later doc. Staleness cost is bounded: snapshots
  refresh on every docs-site build, which already re-syncs the whole index.
- **Rendering AGENT.md as formatted markdown** (client-side renderer, sanitized) —
  agent may choose a renderer only if it introduces no new dependency tree beyond
  one small package AND passes an injection fixture; raw text is the frozen default.
- **PUB022 threshold** — currently "zero of N exported modules mentioned". Whether a
  ratio (e.g. <50%) is a better staleness signal is an implementer decision after
  seeing real AGENT.md files; must remain deterministic and tested both sides.
- **sprint-evaluator enforcement** — a check that executor milestones honored
  `registry_reuse` entries is plausible; leave to the sprint-evaluator skill's own
  evolution, out of scope here.
- **Overlap semantics beyond exact module-name equality** (signature-similarity,
  ai_summary embeddings) — deliberately simple now; the `docsearch`/simhash machinery
  could power a smarter signal later.

## Non-Goals

**Not attempted in this feature:**

- **Any change to gating semantics of existing PUB codes** — the ladder's Sprints 2–3
  (intent consistency, effects measured) remain exactly as planned in their doc (V14).
- **Rendering AGENT.md in the docs *build*** (MDX embedding) — rejected: build safety
  (D1, V10).
- **A `pkg search`-based website search** — the Package Explorer already filters the
  snapshotted index client-side.
- **Enforcing the planner gate with CLI tooling** (D6) — process first; tool later if
  the process measurably fails.
- **Changing `AGENT.md` → `AGENTS.md` naming** — the package convention is `AGENT.md`
  (tarball, validator, `pkg docs` all match on it); renaming would orphan every
  published version's artifact.

## Timeline

**Day 1** (~4 hours):
- Phase 1 implementation + docs-site build verification

**Day 2** (~10 hours):
- Phase 2 implementation, tests, guide updates, `make test` / `make lint`

**Day 3, half** (~4 hours):
- Phase 3 skill + script + template edits, dry-run of `create_sprint_json.sh`

**Total: ~18 hours across 2.5 days** (2× the naive 9-hour estimate, per skill guidance)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| AGENT.md files are large; snapshot fetches slow the docs build | Med | Fetch is per-package, `--max-time 10`, parallel-friendly; registry has ~53 packages (bounded); failure skips one file, not the build |
| Author AGENT.md contains text that breaks the `<pre>` render or misleads (e.g., fake HTML) | Low | `<pre>` renders raw text only — React escapes content; nothing is parsed; a reviewer note in the publishing guide covers misleading guides |
| PUB023 false positives annoy authors of legitimately distinct packages | Med | info-level, names the overlapping packages so the author judges; docstring says "registry informs, author decides" |
| Publisher-side overlap fetch adds a network dependency to `pkg quality` | Med | Explicit "not run: <reason>" badge (D4); `--no-run` semantics unchanged; offline authors still get PUB008/018/022 |
| Skills drift between `.agents/` and `.claude/` variants | Med | D7: one commit edits both; acceptance criterion diffs the step-0 blocks |
| Planner gate becomes a rubber stamp ("none" always) | Med | `registry_reuse` is in the sprint JSON the executor consumes and the evaluator can audit; template forces one entry per implementable milestone |

## Related Documents

**Implemented (may inform design):**
- [m-pkg-quality-ladder Sprint 1](../implemented/v0_40_0/m-pkg-quality-ladder-sprint-plan.md) — the quality report this doc extends (schema, seam doctrine, provenance model)
- [M-PKG-AUTONOMOUS-UPDATES](../implemented/v0_10_0/m-pkg-autonomous-updates.md) — made AGENT.md a first-class GCS artifact (the data source D1 consumes)

**Planned (check for overlap):**
- [m-pkg-quality-ladder](../v0_40_0/m-pkg-quality-ladder.md) — Sprints 2–3 (PUB003/004/013, `effects_measured`): **explicitly distinct** — that doc gates intent/measurement consistency; this doc adds author-feedback badges for discoverability ( PUB008/018/022/023). Neither subsumes the other; codes were cross-checked unallocated (V7, V14)
- [m-publish-path-dep-smoke](../v0_42_1/m-publish-path-dep-smoke.md) — owns PUB017; its docs PUB-reference update pattern is reused here for the new codes

*Auto-search (SimHash + neural) on "pkg registry discoverability" returned only
keyword-noise matches (parser/model-registry docs, ≤0.90 SimHash, no semantic overlap);
the two registry docs above were located by direct code/doc exploration and are the
authoritative nearest neighbors.*

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [Package publishing guide](/docs/guides/package-publishing) — the PUB-code reference this doc extends
- [Packages guide](/docs/guides/packages) — URL conventions and install flow
- [Autonomous package updates guide](/docs/guides/autonomous-package-updates) — `pkg:` inboxes and the AGENT.md artifact
- Skills: `.agents/skills/ailang-packages`, `.agents/skills/sprint-planner` (both harness variants)

## Future Work

- Live AGENT.md hydration endpoint (deferred, see D2-live)
- `sprint-evaluator` audit of `registry_reuse` entries
- Signature/embedding-based overlap detection beyond exact module names
- `ailang pkg audit-plan` — tooling that checks a sprint plan against the registry automatically (replacing the procedural gate if it measurably fails)
- Registry-aware `ailang init package` hinting ("a package named X already exports module Y")

---

**Document created**: 2026-09-28
**Last updated**: 2026-09-28
