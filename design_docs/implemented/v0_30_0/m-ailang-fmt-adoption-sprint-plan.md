# Sprint Plan: M-AILANG-FMT-ADOPTION

**Design doc**: [design_docs/planned/v0_30_0/m-ailang-fmt-adoption.md](m-ailang-fmt-adoption.md)
**Target**: v0.30.0
**Priority**: P2
**Planned by**: sprint-planner (mission-control iteration)
**Date**: 2026-07-19
**Estimated duration**: 1.25 days
**Risk level**: LOW
**Conflict surface**: LOW — prompts / docs / build-glue / hooks only. **No** parser, lexer, AST, type, codegen, eval, effects, or runtime change of any kind (verified against HEAD).

---

## Summary

Make `ailang fmt` (shipped Phase 1 iter-56, made lossless Phase 2 iter-63) discoverable to agents
and wire **opt-in, non-blocking, non-silent** harness hooks so canonical form spreads without ever
being forced. Three independent tracks:

1. **Teaching-prompt discoverability** via a NEW prompt version (v0.16.3), not an in-place edit.
2. **`make fmt-check`** drift instrument + a `docs/docs/reference/formatter.md` Adoption section.
3. **Opt-in Claude Code PostToolUse hook** (`scripts/hooks/format_ail.sh`) that always exits 0,
   surfaces every non-exit-3 `fmt` failure as advisory context, and is bounded by a portable
   timeout with the **Mark-approved SIGTERM→grace→SIGKILL** escalation.

**Zero forced adoption**: no `make ci` gating change, no default-on-everywhere hook mandate, no
mass reformat.

---

## Premise Verification (all checked live against HEAD, 2026-07-19)

The design doc carries several **stale gate/quorum sections** that the mission has since resolved.
These are recorded here for the executor; the planner does **not** edit the design doc itself.

| # | Premise from directive | Status | Evidence |
|---|---|---|---|
| 1 | Phase-2 execution gate is now SATISFIED | **RESOLVED (historical)** | M-AILANG-FMT-PHASE2 landed iter-63 (PR #414, squash `3815ba617`). Implemented doc present at `design_docs/implemented/v0_30_0/m-ailang-fmt-phase2.md` (72 KB). The doc's "⛔ Execution Gate", "⛔ Quorum Block", and "Rev-3 Re-Quorum" sections are now **historical** — the doc is cleared to plan. M1's gate task is kept as a *sanity tripwire*, not a blocker. |
| 2 | SIGKILL escalation approved, no re-quorum | **APPROVED, quorum CLOSED** | Mark approved the `kill "$PID"; sleep 1; kill -9 "$PID" 2>/dev/null; wait "$PID"` fix (doc lines 54-56). Precedent verified live: `.claude/skills/mission-control/SKILL.md:254` uses `kill; sleep 2; kill -9`. **The hook snippet shown in the design doc (lines 282-291) is the PRE-FIX version** (soft `kill` then unbounded `wait`) — M3 must land the escalation, not copy the doc snippet verbatim. |
| 3 | **Exit-code contract check (CRITICAL)** | **VERIFIED — MATCHES** | The shipped binary's exit codes match the hook contract **exactly**. `cmd/ailang/fmt.go` header (lines 24-37): `0`=success/all-canonical, `1`=`--check` drift, `2`=operational error (usage/read/print/round-trip/envelope/write; the 15.28% inline-interior comment refusal lands here too), `3`=input parse error (`const exitParse = 3`). Live tests: non-parsing file `let x = = =` → **exit 3**; `--check` on `examples/ai_modes.ail` → **exit 1**. The doc's "exit 3 = parse error (the one silent case), exit 2 = other refusal" assumption is CORRECT. **No premise conflict** — the silent-vs-surface hook distinction is sound as designed. |
| 4a | `ailang prompt` has no `fmt` mention (v0.16.2 active) | **CONFIRMED** | `ailang prompt \| grep -i fmt` → empty; `prompts/versions.json` `"active": "v0.16.2"`; only `v0.16.0/0.16.1/0.16.2` exist. |
| 4b | CLI `--help` already lists `fmt` | **CONFIRMED** | `ailang --help` line 13: `fmt <file.ail>  Format AILANG source (stdout / --write / --check)`. |
| 4c | No `fmt-check` Makefile target | **CONFIRMED** | `grep fmt-check Makefile` → empty; name unallocated, no collision. |
| 4d | `format_go.sh` precedent exists; no `.ail` hook | **CONFIRMED** | `scripts/hooks/format_go.sh` present (Edit\|Write matcher, always exits 0, surfaces via `hookSpecificOutput.additionalContext`; its one flaw is `gofmt -w … 2>/dev/null` which the `.ail` hook must NOT copy). No `.ail` hook exists. |
| 4e | `formatter.md` reference page exists | **CONFIRMED** | `docs/docs/reference/formatter.md` present (5.8 KB) — the Adoption section extends it. |

**Operational note for the executor:** the installed binary is currently **stale** ("source files
modified after build"). M1's `ailang prompt | grep -i fmt` verification and the gate tripwire both
require `make quick-install` first (already an accepted step / documented risk row).

---

## Milestones

### M1: Gate Sanity-Check + Teaching Prompt — 0.5 day (~40 LOC)

Confirm the (now-satisfied) Phase-2 reality, then land the v0.16.3 teaching prompt.

**Tasks**
- Sanity tripwire (NOT a blocker — Phase 2 landed iter-63): `ailang fmt --help` no longer
  documents the Phase-1 comment-refusal limitation; a non-parsing temp file exits 3; a
  commented-but-parseable file is no longer refused solely for containing comments.
- Create `prompts/v0.16.3.md` = byte-identical copy of `v0.16.2.md` + the ~6-line Formatting
  section (command usage only — deliberately no AILANG code snippet, so no `ailang check` surface).
- Register v0.16.3 in `prompts/versions.json` with file + hash + description; **do not** edit any
  existing registered version file in place.
- Flip `"active"` → `v0.16.3`, `make quick-install`, verify discoverability.

**Acceptance criteria**
- `ailang fmt --help` shows no Phase-1 limitation text; non-parsing file → exit 3.
- `prompts/v0.16.3.md` present; is v0.16.2 + Formatting section only.
- `prompts/versions.json` has a correct v0.16.3 entry (file+hash); no in-place edit of older files.
- `"active": "v0.16.3"`.
- After rebuild: `ailang prompt | grep -i fmt` → teaching line present.
- Append-only proof: `ailang prompt --version v0.16.2 | grep -i fmt` → still empty.

---

### M2: Docs + CLI Audit — 0.25 day (~65 LOC)

**Tasks**
- Audit `ailang --help` / `ailang fmt --help` (expect already good post-Phase-2; fix drift only —
  Phase 2's M3 already owns the limitation-paragraph removal, so verify-not-duplicate).
- Add an **Adoption** section to `docs/docs/reference/formatter.md`: exit-code contract table,
  the five contract clauses, the `make fmt-check` description, the copy-paste Claude Code hook
  config, and the Motoko per-edit cross-repo contract text.
- Cross-link from `docs/docs/guides/development-workflow.md`.

**Acceptance criteria**
- CLI help drift (if any) fixed; otherwise confirmed clean.
- `formatter.md` Adoption section contains contract table + clauses 1-5 + make target + hook config
  + Motoko contract (text only, no cross-repo commit).
- development-workflow guide cross-links the Adoption section.
- Docs build not broken (no bad relative links; page renders).

---

### M3: Opt-in Hooks — 0.5 day (~55 LOC) — carries the Mark-approved SIGKILL fix

**Tasks**
- Add a standalone `make fmt-check` target: `ailang fmt --check` over `examples/**/*.ail` +
  `stdlib/**/*.ail`; exit 1 lists drifted paths. **Not** wired into `make ci`.
- Run `make fmt-check` once; record the drift count as an **informational baseline** (mass
  reformat is out of scope).
- Create `scripts/hooks/format_ail.sh` following `format_go.sh`'s matcher + always-exit-0 +
  `additionalContext` surfacing, but **never** discarding `fmt` stderr.
- **Land the approved timeout fix**: escalate SIGTERM→grace→SIGKILL before `wait` —
  `kill "$PID"; sleep 1; kill -9 "$PID" 2>/dev/null; wait "$PID"`. (The doc's inline snippet is
  the pre-fix soft-kill-then-unbounded-wait version; do not copy it verbatim.)
- Register the hook opt-in under `hooks.PostToolUse` matcher `Edit|Write` in `.claude/settings.json`,
  alongside the existing Go entry.
- `make test` as a regression tripwire.

**Acceptance criteria**
- `make fmt-check` exits 1 + lists paths on drift, 0 when canonical; `make ci` byte-identical
  before/after.
- Drift baseline recorded in the implementation report.
- Hook `ailang fmt` invocation has **no** `2>/dev/null` (only `2>&1` into the capture file); jq
  probed up front (`command -v jq`); first jq call has no stderr suppression; silent branch pinned
  to exactly exit 3.
- **SIGKILL escalation present** in the timeout guard (the load-bearing approved fix).
- `.claude/settings.json` registers the hook; the Go hook still fires unchanged.
- Manual tests pass:
  - (a) valid `.ail` edit → formatted, turn not blocked;
  - (b) non-parsing `.ail` → exit 3 silent defer, file untouched, turn not blocked;
  - (c) unreadable/chmod file → exit 2 advisory context surfaced, turn not blocked;
  - (d) stub `ailang` that ignores SIGTERM and sleeps past the 10s deadline → **SIGKILLed after the
    grace, file byte-identical, timeout advisory surfaced, turn not blocked** (this is the specific
    regression the approved escalation fixes).
- Idempotence spot-check: two hook runs on the same file — second is a no-op.
- `make test` green.

---

## Total: 1.25 days (~130 LOC), matches the design doc's 1–1.5 day estimate.

Velocity reference: fmt Phase-1 (iter-56) and Phase-2 (iter-63, M0–M3 in ~3d, comment-preservation
+ envelope machinery). This adoption sprint is prompts/docs/build-glue only — no compiler surface —
so 1.25 days is realistic and slightly conservative given the manual hook-test matrix in M3.

---

## Success Metrics (from the design doc Goals)

- `ailang prompt | grep -i fmt` returns the new teaching line (from the activated v0.16.3).
- `make fmt-check` reports drift over `examples/` + `stdlib/`, exit 1 on drift, 0 when canonical.
- A documented opt-in Claude Code PostToolUse hook formats `.ail` after Edit/Write, never blocks a
  turn (exit 0 always), surfaces diagnostics as advisory context, and is SIGKILL-escalation bounded.
- Hook contract (exit codes, idempotence, failure behavior) documented in `formatter.md`.
- **Zero forced adoption**: `make ci` unchanged; no mandatory hook; no mass reformat.

---

## Conflict Surface & Regression Equivalents

No language surface is touched, so there is no parser-fixture conflict set. Regression equivalents
(from the design doc) the executor must preserve:

1. `ailang prompt --version v0.16.2` serves the old prompt byte-identical (registry append-only).
2. Eval-harness runs pinning `v0.16.2` are unaffected by the active flip.
3. `make ci` behavior byte-identical before/after (only a new independent target exists).
4. `format_go.sh` continues to fire unchanged alongside the new `.ail` hook.

---

## Files to Modify / Create

**New**
- `prompts/v0.16.3.md` (~copy of v0.16.2 + ~6 lines)
- `scripts/hooks/format_ail.sh` (~40 LOC; non-blocking, non-silent, SIGKILL-escalated)

**Modified**
- `prompts/versions.json` (+1 entry; `"active"` flip as final gated step)
- `Makefile` (+~8 LOC; `fmt-check` target, not wired into `ci`)
- `.claude/settings.json` (+~10 LOC; opt-in hook registration)
- `docs/docs/reference/formatter.md` (+~60 LOC; Adoption section)
- `docs/docs/guides/development-workflow.md` (+~5 LOC; cross-link)

**Untouched**: `internal/**`, `cmd/ailang/**` (Go), any compiler/runtime path.

---

## Stale-Premise Flags for the Executor (do not re-open)

- The **"⛔ Execution Gate" / "⛔ Quorum Block" / "Rev-3 Re-Quorum" sections at the top of the
  design doc are HISTORICAL.** Phase 2 landed (iter-63); the quorum is closed by Mark decision. Do
  not re-run a quorum and do not treat these as blockers.
- The **hook script printed in the design doc (lines 248-314) contains the PRE-FIX timeout guard**
  (soft `kill` + unbounded `wait`). The approved, load-bearing fix is the SIGTERM→grace→SIGKILL
  escalation — implement that, not the doc's verbatim snippet.
- The **exit-code contract is already correct in the shipped binary** — no premise to resolve at
  execution time; the silent-vs-surface hook distinction (exit 3 silent, all else surface) is valid.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-ailang-fmt-adoption-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-AILANG-FMT-ADOPTION.json`
