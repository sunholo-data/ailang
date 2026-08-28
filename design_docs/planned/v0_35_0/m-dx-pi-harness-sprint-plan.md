# Sprint Plan: M-DX-PI-HARNESS — pi dev-harness extensions (Streams A+B)

## Summary
Ship the five ratified extensions (4 Stream A guards/commands + 2-in-1 Stream B tools), each self-contained, each with dotfile predicate tests, all wrapping the real `ailang` CLI under the Subprocess Contract.

**Design doc:** [m-dx-pi-harness.md](m-dx-pi-harness.md) (ratified 2026-08-28; quorum 2 rounds, A3 redesigned as git-authority warning)
**Duration:** 1–2 sessions (~350 LOC + tests)
**Dependencies:** M-DX-SESSION-GATE (shipped — composes, never bypasses)
**Risk Level:** Medium-low (new pi API surface, but all platform claims demonstrated by shipped gate code; subprocess timeouts are the core hardening)

## Proposed Milestones

### M1: Stream A — four guard/ceremony extensions (~1.5h)
**Estimated:** ~260 LOC

**Tasks:**
- `binary-freshness.ts` (~80): parse `ailang version` (Commit/Full[-dirty]/Built, V9), `git rev-parse HEAD`, `git status --porcelain`; classify FRESH/STALE/DIRTY/UNKNOWN (fail-closed); `freshness_report` tool + `/fresh` command
- `sprint-steward.ts` (~70): `/sprint-start <id>`, `/sprint-complete <milestone-id>` — constrained modification (only passes/started/completed/notes) + schema validation; refuse everything else
- `unowned-dirty.ts` (~60): track session `edit`/`write` toolCalls; on bash `git add|stash|checkout` run `git status --porcelain` (10s) and `notify` unowned dirty files (warn-only, names itself a heuristic)
- `builtin-sprint.ts` (~50): `/builtin-finish` chains golden refresh → `doctor builtins` → `builtins list --json` diff

**Acceptance criteria:**
- [ ] Predicate tests: freshness classification (incl. `-dirty` + UNKNOWN fallback), JSON-constrained write refuses non-conforming edits, unowned-set intersection, version-output parsing
- [ ] All load clean alongside the session gate

### M2: Stream B — ailang-lsp-lite (~1h)
**Estimated:** ~90 LOC

**Tasks:**
- `ailang-lsp-lite.ts`: `ailang_check(path)` tool (30s timeout, parse errors/warnings into `{code,message,file,line,col,hint}`) + `builtins_search({query?,module?})` tool (15s, filtered from `--json` inventory, cap 10 on query)
- Shared subprocess helper honoring the Subprocess Contract (64KB caps, structured TIMEOUT)

**Acceptance criteria:**
- [ ] B1 returns code-tagged diagnostics for a seeded IMP010 and a TC error
- [ ] B2 ≤10 matches on query, full inventory without query
- [ ] Parser handles the live error formats (verified V3)

### M3: Hardening + docs (~0.5h)
**Tasks:**
- F5: verify coordinator-session extension inheritance (run a coordinator-spawned session or document the gap)
- Compat matrix in README (pi 0.84.3 per extension)
- README index update + CHANGELOG DX entries

**Acceptance criteria:**
- [ ] All design-doc Success Criteria boxes pass or are explicitly deferred with a register entry
- [ ] Session gate regression check: arms/acks with all extensions loaded
- [ ] `make test` green (no Go surface touched — sanity only)