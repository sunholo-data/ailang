# M-AILANG-FMT-ADOPTION: `ailang fmt` Adoption — Discoverability + Opt-in Harness Hooks

**Status**: ⛔ **PARKED — needs-human-review (design-quorum BLOCKED, 2 rounds)** + GATED behind [M-AILANG-FMT-PHASE2](m-ailang-fmt-phase2.md) — see the ⛔ Quorum Block below
**Target**: v0.30.0
**Priority**: P2 (valuable, but strictly sequenced after Phase 2)
**Estimated**: 1–1.5 days
**Dependencies**: [M-AILANG-FMT-PHASE2](m-ailang-fmt-phase2.md) (hard gate — comment preservation AND the exit-code split (3 = parse error) this doc's hook contract distinguishes on), [M-AILANG-FMT Phase 1](../../implemented/v0_30_0/m-ailang-fmt.md) (implemented)

## ⛔ Quorum Block (iteration 59, 2026-07-19) — needs-human-review

Created + revised this iteration on Mark's #399 directive. Ran the design-quorum twice (bounded
one-revision cap). R1's silent-fallback objection (the `2>/dev/null` hook) was resolved; R2
surfaced **two new, small, fixable defects** in the revised hook:

1. **`gpt5-6-sol` — no timeout → bounded-waits violation.** The synchronous PostToolUse wrapper
   runs `ailang fmt --write` with no ceiling; a hang wedges the agent turn regardless of the
   always-exit-0 posture. **Fix (small):** wrap the `fmt` call in a bounded timeout (portable
   `date +%s` deadline or a backgrounded-kill guard, no GNU `timeout` dependency) and on expiry
   surface an advisory "fmt timed out" note, still exit 0.
2. **`gemini-3-1-pro` — residual silent fallback on the FIRST `jq`.** `file_path=$(… | jq -r … 2>/dev/null)`
   still swallows a missing-`jq` "command not found", yielding an empty path that no-ops via the
   wildcard case — contradicting this doc's own "a missing jq cannot silence a real error" claim.
   **Fix (small):** probe `jq` presence up front (or drop the first-`jq` stderr suppression) so a
   broken dependency surfaces as advisory context, never a silent skip.

**Both are mechanical hook-script corrections, not contract rejections.** Per the mission's bounded
quorum gate, parked for the human alongside the companion Phase-2 doc: authorize one short revision
round to fix both. Quorum artifacts:
`.ailang/state/mission-quorum/m-ailang-fmt-adoption-2026-07-19T07-02-40Z.json` (R1),
`…-07-23-50Z.json` (R2). Metered cost both rounds: ~$0.13.

## ⛔ Execution Gate

**This doc must not be scheduled, sprint-planned, or executed until Phase 2 (lossless comment
preservation) has landed.** Rationale, measured live 2026-07-19: Phase 1 refuses 372/393 (94.7%)
of `examples/**/*.ail` because they contain comments. A teaching-prompt line or a post-write hook
that invokes a formatter which refuses 94.7% of real files is **worse than no adoption at all** —
agents would burn turns on exit-2 refusals and learn to distrust the tool (no-premature-adoption).
The gate check is mechanical: the Phase 2 acceptance criteria "corpus comment-refusals = 0" AND
"exit-code split landed (`fmt` on a non-parsing file exits 3, not 2)" must both be green before M1
here starts — the hook contract's silent/surface distinction is built on that split.

## Problem Statement

`ailang fmt` shipped in iter-56 (Phase 1) with a solid CLI, but nothing outside the CLI knows it
exists:

**Current State (all verified live 2026-07-19, see Verification Log):**

- **Agents don't know the command exists.** `ailang prompt | grep -i fmt` returns nothing — the
  active teaching prompt (v0.16.2, embedded) teaches `check`/`run`/`test` workflows but never
  mentions `fmt`. An agent that has only the teaching prompt cannot discover the formatter.
- **CLI discoverability is already decent** — `ailang --help` lists
  `fmt <file.ail>  Format AILANG source (stdout / --write / --check)` and `ailang fmt --help` is
  complete (usage, flags, exit codes, current limitation). The audit item is to confirm and keep
  this, not to build it.
- **No harness integration point exists.** There is no `fmt-check` make target, no CI drift check,
  no post-write hook for `.ail` files (a `format_go.sh` PostToolUse hook exists for Go — precedent,
  but nothing for AILANG source).

**Impact:** the canonical form the formatter defines cannot actually converge the ecosystem —
generated code, examples, and agent edits keep drifting among equivalent spellings because no
touchpoint runs `fmt`.

## Goals

**Primary Goal:** After Phase 2 lands, make `ailang fmt` discoverable to agents (teaching prompt)
and wire **opt-in** harness hooks (`--check` in CI, `--write` post-edit) so canonical form spreads
without ever being forced.

**Success Metrics:**

- `ailang prompt | grep -i fmt` returns the new teaching line (from the activated prompt version).
- `make fmt-check` exists and reports drift over `examples/` + `stdlib/` with exit 1 on drift, 0 when canonical.
- A documented, opt-in Claude Code PostToolUse hook formats `.ail` files after Edit/Write, never blocks a turn.
- **No silent fallback:** the hook never discards `fmt` diagnostics. Exactly ONE case is silent —
  exit 3, "input does not parse yet" (the expected mid-edit state; contract clause 5). Every other
  failure (read, print, round-trip, envelope, write, panic, missing binary) surfaces to the agent
  as advisory context while the wrapper still exits 0.
- Hook contract (exit codes, idempotence, failure behavior) documented in `docs/docs/reference/formatter.md` — one page a harness author reads to integrate safely.
- Zero forced adoption: no change to `make ci` gating, no default-on hook, no prompt teaching before Phase 2.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Teaching happens via a NEW prompt version entry (v0.16.3), not by editing `prompts/v0.16.2.md` in place | The prompt registry (`prompts/versions.json`) records file + hash per version; in-place edits corrupt the versioned/hashed registry contract and eval reproducibility | human | design | med |
| All hooks are opt-in, non-blocking, AND non-silent: the wrapper always exits 0, captures `fmt` stderr (never `2>/dev/null`), defers silently ONLY on exit 3 (input-not-parseable, the expected mid-edit state), and surfaces every other failure as advisory context | A blocking formatter hook on partial/broken edits would wedge agent loops; a diagnostic-swallowing hook is a silent fallback that hides parse-tool breakage, panics, and FS errors from the agent — both are mission-axiom violations (design-quorum finding) | human | design | med |
| The parse-vs-operational distinction comes from a `fmt` **exit-code split (3 = input parse error, 2 = operational)**, folded into Phase-2 scope — NOT from hook-side stderr message inspection | Verified (V10): today ALL `fmt` failures exit 2, distinguishable only by fragile stderr text matching; an exit code is a stable machine contract, and Phase 2 already owns `cmd/ailang/fmt.go`'s exit paths | human | design | low |
| CI drift check is a standalone `make fmt-check` target, NOT added to the `make ci` gate in this doc | Forcing canonical form repo-wide is a separate, larger decision (mass-reformat commit, churn); this doc only builds the instrument | human | design | low |
| Motoko per-edit integration is specified as a *contract* here; wiring lands in the motoko fork, not this repo | Cross-repo change; this repo can only define the interface the fork consumes | human | design | low |

### Design Freeze

- [x] New prompt version entry (not in-place edit); activation of the new version is the LAST step and only after Phase 2 is verified landed
- [x] Non-blocking, non-silent hook wrapper semantics: always exit 0; stderr always captured;
  exit 3 (input parse error) defers silently; every other nonzero outcome surfaces as advisory
  context via `hookSpecificOutput.additionalContext` (the mechanism `format_go.sh` already uses
  for lint findings — V6)
- [x] The distinction rides on the Phase-2 exit-code split (3 vs 2); this doc's hook is gated on
  that split landing (it is part of the same Phase-2 gate this doc already waits on)
- [x] `make ci` unchanged by this doc
- [ ] Final wording of the teaching line (draft below; human may tune at review)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact filename/registry description text for the v0.16.3 prompt entry — agent may choose
- Whether `make fmt-check` covers `examples/` only or `examples/ + stdlib/` in its first cut — agent may choose (metric above assumes both; shrink with a note if stdlib has blockers)
- Hook script placement/naming under `scripts/hooks/` (mirroring `format_go.sh` is the obvious choice) — agent may choose
- Whether the Claude Code hook example ships enabled in this repo's `.claude/settings.json` or only as documented copy-paste — human at review (default: documented + enabled here, since this repo's agents edit `.ail` files constantly)

## Solution Design

### Overview

Three small, independent tracks, all gated behind Phase 2: (1) one concise teaching line in a new
prompt version so agents know `fmt` exists and when to use it; (2) an audit pass confirming CLI
help remains good and the reference docs cover adoption; (3) opt-in hook wiring with a precisely
specified contract — `--check` for CI drift reporting, `--write` for post-edit convergence in
agent harnesses (Claude Code PostToolUse here; Motoko per-edit as a documented cross-repo
contract).

### Track 1: Teaching-Prompt Discoverability

Verified premise: `ailang prompt` (active = `v0.16.2` per `prompts/versions.json`, served from the
embedded FS) contains no mention of `fmt`. The prompt loader resolves `""`/`"latest"` →
`manifest.Active` → versioned file, embedded-first.

Mechanism: add `prompts/v0.16.3.md` (v0.16.2 content + a short tooling addition), register it in
`prompts/versions.json` with its hash, and flip `"active"` to `v0.16.3`. **Activation is the final
step of this doc and is blocked on the Phase 2 gate check.**

Draft teaching addition (command usage only — deliberately contains no AILANG code snippet, so no
`ailang check` verification surface; wording tunable at review):

```markdown
## Formatting

`ailang fmt --write file.ail` rewrites a file into canonical form (idempotent; preserves
comments). Use it after your edits type-check. `ailang fmt --check file.ail` reports drift
without writing. Formatting never changes program meaning: `Parse(fmt(x)) ≡ Parse(x)`.
```

Keep it to a handful of lines: the prompt is token-budgeted and served once in the system role;
`fmt` is a convenience, not a correctness gate, and must not crowd out syntax teaching.

### Track 2: CLI + Docs Discoverability (audit)

Verified current state: `ailang --help` line 13 lists `fmt`; `ailang fmt --help` documents usage,
flags, exit codes, and the Phase 1 limitation; `docs/docs/reference/formatter.md` exists.

Work: (a) after Phase 2, confirm `fmt --help` and `formatter.md` no longer describe the comment
refusal (Phase 2's M3 owns that edit; this track verifies, not duplicates); (b) extend
`formatter.md` with an **Adoption** section — the hook contract below, the make target, and the
copy-paste Claude Code hook config; (c) cross-link from the development-workflow guide.

### Track 3: Opt-in Harness Hooks

#### Hook Contract (the normative core of this doc)

The `ailang fmt` exit-code contract **after the Phase-2 exit split** (Phase 1 shipped 0/1/2 with
every failure on 2; Phase 2 splits input-parse errors onto 3 — a change owned by
[M-AILANG-FMT-PHASE2](m-ailang-fmt-phase2.md), required by this contract and part of the same gate
this doc waits on. Verified against today's code, V10: all `formatOne` failure paths currently
share exit 2 and differ only in stderr text, so without the split a hook could distinguish the
classes only by fragile message matching — rejected):

| Exit | Meaning | Hook interpretation |
|---:|---|---|
| 0 | Formatted / all canonical (`--check`) | Success; file may have been rewritten (`--write`). Optional one-line confirmation. |
| 1 | `--check` only: at least one non-canonical file | Drift signal — CI reports it; post-write hooks never see it (they use `--write`) |
| 2 | Genuine operational error: usage, read, print, round-trip, envelope, or write failure (an unrecovered Go panic also exits 2) | **Surface stderr to the agent as advisory context** (`hookSpecificOutput.additionalContext`), then exit 0. Never silent, never blocking. |
| 3 | Input does not parse (Phase 2+) | **The ONLY silent case.** Expected in normal operation: agents write partial files mid-edit; the hook defers until the file parses (clause 5). |

Contract clauses harness authors rely on:

1. **Idempotence**: `fmt(fmt(x)) == fmt(x)` byte-for-byte (Phase 1 property, extended to comments
   by Phase 2). Hooks may fire repeatedly — after every edit, in loops — with no oscillation, and
   compose with other write-then-read tooling.
2. **Meaning preservation**: `Parse(fmt(x)) ≡ Parse(x)`. A hook can never change what the agent's
   code does — formatting is safe to interleave anywhere in an edit loop.
3. **Atomicity**: `--write` validates in memory and replaces atomically per file; a mid-hook crash
   never leaves a truncated file.
4. **Non-blocking, NON-SILENT wrappers**: every hook wrapper MUST exit 0 regardless of `fmt`'s
   exit code — a formatter must never wedge an agent turn or a commit. But the wrapper MUST
   capture `fmt`'s stderr (never `2>/dev/null`) and MUST surface every non-exit-3 failure to the
   agent as advisory context. Discarding diagnostics conflates "file not parseable yet" with read,
   round-trip, envelope, write, and panic failures — a silent fallback (no-silent-fallbacks
   axiom; design-quorum finding on the original draft of this doc). Precedent, re-verified (V6):
   `scripts/hooks/format_go.sh` always exits 0 AND surfaces lint diagnostics via
   `hookSpecificOutput.additionalContext` on stdout so "the next turn sees them" (its own header);
   its one flaw — `gofmt -w "$file" 2>/dev/null` discards gofmt's stderr — is exactly what this
   hook must NOT copy.
5. **Never format what doesn't parse**: `--write` on a not-yet-parseable file exits **3** and
   writes nothing — the hook defers silently until the file parses again. This is the sole
   sanctioned silence: mid-edit parse failure is an expected state, not a fault. (After Phase 2,
   comment-refusal is gone; exit 3 covers only genuine parse incompleteness.)

#### Hook 1: CI drift check (`--check`)

- New Makefile target `fmt-check`: `ailang fmt --check` over `examples/**/*.ail` and
  `stdlib/**/*.ail`; exit 1 lists drifted paths. (Verified: no `fmt-check` target exists today.)
- **Opt-in**: documented, runnable locally and as a standalone/advisory CI step. NOT added to the
  `make ci` gate by this doc — gating CI on canonical form requires a one-time mass-reformat
  decision that belongs to a future doc (see Non-Goals).

#### Hook 2: Claude Code PostToolUse (`--write`)

`scripts/hooks/format_ail.sh`, using `format_go.sh`'s matcher mechanism and always-exit-0 posture
plus its `additionalContext` surfacing mechanism (V6) — but, unlike the Go hook's gofmt line,
never discarding stderr:

```bash
#!/bin/bash
# PostToolUse hook for Edit/Write on .ail files: canonical-format the edited file.
# Non-blocking (always exits 0) and NON-SILENT (contract clause 4):
#   exit 0 -> optional confirmation
#   exit 3 -> input does not parse yet (expected mid-edit) -> defer silently (clause 5)
#   anything else (2 = operational error / panic; 127 = ailang missing; ...) ->
#     surface captured output to the agent via additionalContext, then exit 0
set +e
HOOK_JSON=$(cat 2>/dev/null || echo "{}")
file_path=$(echo "$HOOK_JSON" | jq -r '.tool_input.file_path // ""' 2>/dev/null)
case "$file_path" in
  *.ail) ;;
  *) exit 0 ;;
esac

FMT_OUT=$(ailang fmt --write "$file_path" 2>&1)   # capture stderr — NEVER 2>/dev/null
FMT_RC=$?

case "$FMT_RC" in
  0) echo "✓ Formatted $file_path" >&2 ;;
  3) : ;;  # not parseable yet — the ONLY silent case (contract clause 5)
  *)
    jq -n --arg ctx "ailang fmt failed (exit $FMT_RC) on $file_path — file left as written:
$FMT_OUT" '{
      hookSpecificOutput: {
        hookEventName: "PostToolUse",
        additionalContext: $ctx
      }
    }' 2>/dev/null || echo "ailang fmt failed (exit $FMT_RC) on $file_path: $FMT_OUT" >&2
    ;;
esac
exit 0
```

Registered under `hooks.PostToolUse` with matcher `Edit|Write` in `.claude/settings.json`
(alongside the existing Go entry). Opt-in per repo; this repo enables it (deferred-decision row).

Design notes: (a) the `jq` fallback line means even a missing `jq` cannot silence a real error —
it degrades to plain stderr, which PostToolUse also surfaces; (b) exit 127 (`ailang` not on PATH)
and exit 2 from an unrecovered Go panic both land in the surface-it branch by construction —
the silent branch is pinned to exactly `3`, never a catch-all.

#### Hook 3: Motoko per-edit (`--write`) — cross-repo contract

The motoko harness already returns per-edit `ailang check` feedback (a `typecheck` field on every
edit result). The adoption contract for the fork: **after an edit whose `typecheck` is green, run
`ailang fmt --write <file>`; on exit ≠ 0, keep the file as-written, continue, and attach `fmt`'s
captured stderr to the edit result** (the same channel the `typecheck` field rides on) — never
discard it (clause 4). Note the asymmetry with the Claude Code hook: because this contract only
fires after a green type-check (which implies the file parses), **exit 3 is unexpected here too**
and is surfaced like any other failure rather than silently deferred. Formatting only type-checked
files means the model always sees its own code in canonical form on the next read — passive
convergence pressure with zero added failure modes and zero swallowed diagnostics. This repo ships
the contract text in `formatter.md`; the wiring is a motoko-fork change outside this doc's scope
and this repo's tree.

## Files to Modify/Create

**New files:**
- `prompts/v0.16.3.md` (~copy of v0.16.2 + ~6 lines) — teaching addition
- `scripts/hooks/format_ail.sh` (~40 LOC) — PostToolUse hook (non-blocking + non-silent contract)

**Modified files:**
- `prompts/versions.json` (+1 entry; `"active"` flip as final gated step)
- `Makefile` (+~8 LOC) — `fmt-check` target
- `.claude/settings.json` (+~10 LOC) — opt-in hook registration
- `docs/docs/reference/formatter.md` (+~60 LOC) — Adoption section: contract table, clauses, make target, hook config, motoko contract
- `docs/docs/guides/development-workflow.md` (+~5 LOC) — cross-link

**No changes** to `internal/**`, `cmd/ailang/**` (help text is already good; Phase 2 owns the
limitation-paragraph removal), or any compiler/runtime path.

## Milestones

### M1: Gate Check + Teaching Prompt — 0.5 day

- [ ] **Gate**: verify Phase 2 landed — corpus sweep shows 0 comment-refusals; `ailang fmt --help` no longer documents the Phase 1 limitation; `ailang fmt` on a deliberately non-parsing temp file exits **3** (exit split live). If red, STOP (doc stays parked)
- [ ] Create `prompts/v0.16.3.md` with the Formatting section; register in `versions.json` with hash
- [ ] Flip `"active"` to v0.16.3; verify `ailang prompt | grep -i fmt` returns the teaching line (requires rebuilt binary — prompts are embedded)

### M2: Docs + CLI Audit — 0.25 day

- [ ] Audit `ailang --help` / `ailang fmt --help` post-Phase-2 (expect: already good; fix drift only)
- [ ] Add the Adoption section (contract table + clauses + motoko contract) to `docs/docs/reference/formatter.md`; cross-link from development-workflow guide

### M3: Opt-in Hooks — 0.5 day

- [ ] `make fmt-check` target; run it once and record the drift count (informational baseline — mass reformat is out of scope)
- [ ] `scripts/hooks/format_ail.sh` + `.claude/settings.json` registration; manual tests: (a) edit an `.ail` file via Claude Code → formatted, turn not blocked; (b) deliberately non-parsing file → exit 3 path, silent defer, turn not blocked; (c) simulated operational error (e.g. chmod-unreadable file) → advisory context surfaced to the agent, turn not blocked
- [ ] `make test` green (nothing in `internal/` changed; this is a regression tripwire, not a coverage claim)

**Total: 1.25 days.**

## Conflict Surface

This doc touches **no parser, lexer, AST, type, codegen, eval, or effects code** — it is
prompts/docs/build-glue only. The section exists to prove that claim, not to enumerate grammar
positions (there are none).

| Area | Relationship and constraint |
|---|---|
| `prompts/` + `prompts/versions.json` | New version entry + active flip. The registry is versioned and hashed; **never edit an existing registered prompt file in place**. Eval baselines pin prompt versions — flipping `active` changes what `latest` serves, which is the intended, gated effect. |
| `cmd/ailang` | **Audit only.** Help text shipped in Phase 1; Phase 2 owns the limitation-paragraph edit. This doc changes no Go code. |
| `Makefile` | Additive target `fmt-check`; `ci` target untouched (verified: no existing `fmt-check` to collide with). |
| `scripts/hooks/`, `.claude/settings.json` | Additive, opt-in; follows the existing `format_go.sh` PostToolUse pattern verbatim. |
| `docs/docs/` | Additive sections in existing pages. |
| `internal/**`, compiler/runtime | **Untouched.** No core change of any kind. |
| motoko fork (external) | Receives a documented contract only; no cross-repo commit from this sprint. |

### Programs that MUST still work

Not applicable in the parser-fixture sense (no language surface is touched). The regression
equivalents:

1. `ailang prompt --version v0.16.2` still serves the old prompt byte-identical (registry is append-only)
2. Existing eval-harness runs pinning `v0.16.2` are unaffected by the active flip
3. `make ci` behavior is byte-identical before/after (only a new independent target exists)
4. The Go PostToolUse hook (`format_go.sh`) continues to fire unchanged alongside the new `.ail` hook

### What deliberately changes

- `ailang prompt` (unpinned / `latest`) serves v0.16.3 including the Formatting section — the
  intended teaching change, taken only after the Phase 2 gate.

## Testing Strategy

**Unit/integration:** none required in `internal/` (no Go changes). `make test` runs as a tripwire.

**Manual/scripted verification (recorded in the implementation report):**
- `ailang prompt | grep -i fmt` → teaching line present (post-rebuild)
- `ailang prompt --version v0.16.2 | grep -i fmt` → still empty (append-only registry proof)
- `make fmt-check` on a tree with one deliberately drifted file → exit 1, path listed; on canonical tree → exit 0
- Hook: edit `.ail` via Claude Code → file canonicalized, turn not blocked; repeat on a file with
  a syntax error → exit 3, file untouched, NO context emitted, turn not blocked (clause 5); repeat
  with a forced operational error (unreadable file / renamed binary) → file untouched, advisory
  context visible to the agent on the next turn, turn not blocked (clause 4)
- Silence audit: grep the shipped hook for `2>/dev/null` on the `ailang fmt` invocation — must be
  absent (the only sanctioned redirect is `2>&1` into the captured variable)
- Idempotence spot-check: run the hook twice on the same file; second run is a no-op

## Non-Goals

- **No teaching before Phase 2 lands** — the execution gate is the point of this doc's sequencing (a prompt line about a tool that refuses 94.7% of commented files frustrates agents; no-premature-adoption)
- **No forced adoption**: `make ci` does not gain a formatting gate; no hook is mandatory; drift is reported, not rejected
- **No mass reformat** of `examples/`/`stdlib/` — a future decision once `fmt-check` gives a stable baseline (churn + blame noise tradeoff belongs to its own doc)
- **No editor/LSP formatting integration** (`ailang lsp` `textDocument/formatting`) — natural follow-up, separate design
- **No motoko-fork implementation** — contract only
- **No prompt-content overhaul** — v0.16.3 is v0.16.2 + the Formatting section, nothing else moves

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Doc executed before Phase 2 (agents taught a refusing tool) | High | Explicit execution gate at top; M1 first task is the mechanical gate check; sprint-planner must treat the dependency as hard |
| Prompt registry corruption (in-place edit of a hashed version) | Medium | Design-Freeze rule: new version entry only; verification step proves v0.16.2 serves byte-identical |
| Teaching line induces premature `fmt` calls on broken code | Low | Line says "after your edits type-check"; clause 5 means the worst case is a no-op exit 3 |
| Hook surfaces noisy advisory context on repeated operational errors (e.g. broken install) | Low | Context fires only on non-0/non-3 exits, which are genuine faults the agent SHOULD see and fix (e.g. reinstall); silence here was the design flaw the quorum blocked, not the noise |
| Hook-side `jq` missing → surfacing path degrades | Low | Explicit fallback to plain stderr in the script (also agent-visible in PostToolUse); the silent branch remains pinned to exit 3 only |
| Post-write hook fights another write-path tool | Low | Idempotence clause 1 + PostToolUse ordering (runs after the write completes); manual double-run test |
| `fmt-check` baseline shows huge drift and invites a rushed mass reformat | Low | Explicit non-goal + informational-baseline framing in M3 |
| Stale installed binary serves old embedded prompt after flip | Low | M1 verification requires rebuild (`make quick-install`) before the grep check |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No execution-path change; formatting remains presentation-only |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | Hooks are opt-in, explicitly registered; no ambient authority added |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Agents (the primary users) gain discoverability of a machine-friendly canonicalizer; per-edit hooks show models their own code in one stable form |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | `--check` makes drift visible and countable (baseline metric) instead of latent |
| A10: Composability | +1 | Contract clauses (idempotence, meaning-preservation, non-blocking) are exactly what makes the tool composable with CI, Claude Code, and motoko without coordination |
| A11: Structured Failure | +1 | The hook contract classifies every failure into exactly one of surface-as-context (operational) or defer-silently (expected mid-edit parse state), keyed on a machine-stable exit code (Phase-2 split) — no diagnostic is ever swallowed; the CLI-side code change itself is owned by Phase 2 |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Proceed (after gate)**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Verification Log

All commands run 2026-07-19 in the main checkout (`dev`), installed binary
`AILANG v0.29.2-421-g81a45f2d8`.

| # | Command / inspection | Observed result |
|---|---|---|
| V1 | `ailang prompt \| grep -in fmt` (negative-existence: fmt absent from teaching) | No matches. Prompt header confirms `# AILANG v0.16.2 - AI Teaching Prompt`. |
| V2 | `ailang --help \| grep -in fmt` | Line 13: `fmt <file.ail>     Format AILANG source (stdout / --write / --check)` — top-level discoverability already present. |
| V3 | `ailang fmt --help` | Exit 0; complete usage/flags/exit-code table + Phase 1 limitation paragraph. CLI help needs no work beyond Phase 2's limitation-text removal. |
| V4 | `grep -rln "ailang fmt" docs/docs/` | `docs/docs/reference/formatter.md` exists (plus roadmap mention) — the Adoption section extends an existing page. |
| V5 | `grep -rn "fmt-check" Makefile` (negative-existence) | Empty — the target name is unallocated. |
| V6 | Read `scripts/hooks/format_go.sh` (all 74 lines) + `.claude/settings.json` PostToolUse block | **Precedent re-verified precisely (revision pass — the original draft over-summarized it):** Edit\|Write matcher; always exits 0 ("so partial-syntax edits don't block the workflow"); lint diagnostics are captured with `2>&1` and SURFACED to the agent via `hookSpecificOutput.additionalContext` JSON on stdout ("so the next turn sees them" — its own header, lines 4–7, 36–71). It is NOT a swallow-everything hook. Its one silent spot is `gofmt -w "$file_path" 2>/dev/null` (line 19), which discards gofmt's own stderr. The `.ail` hook adopts the matcher + exit-0 posture + `additionalContext` surfacing mechanism, and explicitly does NOT copy the gofmt-line stderr discard. |
| V7 | Read `prompts/versions.json` + `internal/prompt/loader.go:39-70` | Registry maps version → `{file, hash, description, …}`; `"active": "v0.16.2"`; loader resolves `""`/`"latest"` → `manifest.Active`, reads embedded FS first. Grounds the new-version-entry mechanism (and the rebuild-required risk). |
| V8 | Phase-2 gating numbers | Corpus sweep (recorded as V4 in [M-AILANG-FMT-PHASE2](m-ailang-fmt-phase2.md)): 372/393 comment-refused (94.7%) today — the quantitative basis for the execution gate. Iter-58's 344/393 (87.5%) was a narrower sweep; the live recursive number is used here. |
| V9 | Hook-script code review (no AILANG code in this doc) | This doc contains no AILANG snippets, so the `ailang check` syntax gate has no surface here; the one code block is bash following the verified `format_go.sh` mechanisms (V6). The draft teaching line likewise cites commands only. |
| V10 | **Revision pass:** read every error path in `cmd/ailang/fmt.go` (`runFmtCommand`, `formatOne`, `parseForFmt`, `fmtStdout`/`fmtCheck`/`fmtWrite`, `atomicWriteFile`) | ALL failure classes — usage, read, comment-preflight, input parse (`"<path>: parse error: …"`), print, round-trip re-parse, round-trip AST diff, temp/chmod/rename write errors — exit **2**, distinguishable only by stderr message text. A hook therefore CANNOT separate "mid-edit file, defer" from "formatter broke, surface" on today's contract without fragile message matching. → The exit-code split (3 = input parse error) is a REQUIRED enhancement, folded into Phase-2 scope (its doc's Design Freeze + M3 now own it); this doc's gate check verifies it landed. |
| V11 | **Revision pass:** hook redesign audited against the no-silent-fallbacks axiom | Original draft's hook ran `ailang fmt --write "$file" 2>/dev/null` and emitted nothing on failure — a silent fallback conflating expected parse-deferral with read/round-trip/envelope/write/panic failures (design-quorum finding, upheld). Redesigned: stderr always captured (`2>&1`), silence pinned to exactly exit 3, all other nonzero exits surfaced via `additionalContext` with a plain-stderr fallback if `jq` is absent. Claude Code visibility of the surfaced context relies on the same mechanism `format_go.sh` already documents and uses in this repo (V6) — not on an assumed external behavior. |

**Revision pass (2026-07-19, same day):** a multi-provider design quorum blocked the original
draft's PostToolUse hook as a silent fallback (discarded stderr, no output on exit 2) violating
both the mission axiom and this doc's own advisory-context decision. The objection was upheld:
V6 was re-verified line-by-line (the precedent actually SURFACES diagnostics — the draft had
copied its one flaw, the gofmt stderr discard, rather than its mechanism), V10 established that
today's exit codes cannot carry the needed distinction, and the contract table, clauses 4–5, the
hook script, the motoko contract, and the Phase-2 gate were all rewritten so that no diagnostic
is ever swallowed while the wrapper remains non-blocking.

## References

- **Gate / dependency**: [M-AILANG-FMT-PHASE2](m-ailang-fmt-phase2.md) — companion doc; its landing removes the comment refusal this doc's gate waits on
- **Foundation**: [M-AILANG-FMT Phase 1 (implemented)](../../implemented/v0_30_0/m-ailang-fmt.md) — CLI contract, exit codes, invariants this doc's hook contract re-exports
- **Superseded stub** (0.50 neural match, distinct scope): [v0.29.0 formatter stub](../../planned/v0_29_0/m-ailang-fmt.md) covers the formatter *implementation*; this doc covers *rollout* only
- **Greenlight**: Mark, GitHub issue #399 ("do the fmt design docs next")
- **Precedents**: `scripts/hooks/format_go.sh` (non-blocking post-write formatting), `prompts/versions.json` (versioned prompt registry)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **CI gating + mass reformat**: once `fmt-check` drift baseline is stable and near zero, a follow-up doc may add `fmt-check` to `make ci` with the one-time reformat commit
- **LSP formatting**: expose `fmt` via `ailang lsp` `textDocument/formatting` so editors format on save
- **Motoko fork wiring**: implement the per-edit contract in the fork (its own change, validated on the rig)
- **Prompt A/B**: measure whether the teaching line changes agent formatting behavior (eval-harness experiment, not assumed)
