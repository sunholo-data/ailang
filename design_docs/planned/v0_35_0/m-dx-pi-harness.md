# M-DX-PI-HARNESS — pi as a first-class development harness for AILANG (two streams)

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1
**Estimated**: Phase 1 ~1 session; Phase 2 ~1 session; Phase 3 hardening ~0.5 session
**Dependencies**: M-DX-SESSION-GATE (shipped, `8ea0ed62b`) — this doc extends, never bypasses, its gate

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Repo developer-experience tooling (pi extensions), not language surface. Scores on the axioms that concern agent workflow and authority.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect system change |
| A4: Explicit Authority | +1 | Stream A adds mechanical guards (binary-freshness, tree-ownership, sprint-schema) — authority checks that today are tribal knowledge |
| A5: Bounded Verification | +1 | Every wrapper subprocess runs under an explicit timeout contract (see Subprocess Contract) — a hang degrades to a structured TIMEOUT result, never an indefinite block; Stream B additionally puts the real builtin inventory inside the agent's context |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | The entire workstream is machine ergonomics: structured diagnostics over CLI text, mechanical guards over conventions |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | Extensions add no metered calls |
| A10: Composability | +1 | Every extension wraps existing `ailang` CLI capabilities — never re-implements them (CLAUDE.md principle 1); composes with the session gate and skills |
| A11: Structured Failure | +1 | B1 turns `ailang check` text into structured, code-tagged diagnostics the model can act on directly |
| A12: System Boundary | +1 | Stream B tools make the .ail↔toolchain boundary explicit per call |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no violations (A4/A7 strengthened)

## Verification Log

pi-platform claims — verified against pi 0.84.3 and demonstrated by SHIPPED code in this repo (the
session gate, e2e-validated 2026-08-28); the load-bearing rows are inlined here so this doc is
self-contained (round-2 quorum fix):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V11 | `pi.registerTool` registers LLM-callable tools (`parameters` + `execute` → `{content, details}`) | extensions.md "Custom Tools"; **demonstrated live**: the shipped session-protocol-gate.ts registers `session_protocol_ack` this way, exercised end-to-end in `-p` mode today | Confirmed by working code |
| V12 | `tool_call` interception can inspect bash input and warn without blocking; per-session state reconstructs via `getBranch()` | extensions.md `tool_call` contract (gate doc V1); the gate's interceptor + getBranch ack-reconstruction run in production in this repo today; warnings use `ctx.ui.notify` (documented fire-and-forget in TUI+RPC) | Confirmed by working code |
| V1 | `ailang builtins list --json` emits structured inventory (name, module, signature, effect, num_args, description) | Ran it live: single-line JSON array covering all std modules incl. `std/fs` (20 builtins) | Confirmed |
| V2 | `.pi/extensions/` currently contains exactly the session gate + README (+ dotfile test) | `ls .pi/extensions/` | Confirmed — no existing extension does anything proposed here |
| V3 | `ailang check` failures carry structured codes (IMP010/TCxxx/MODxxx/EFF_*) with file:line:col spans | Live: `builtinHint` e2e output 2026-08-28; import-hint machinery in `internal/importhint` | Confirmed |
| V4 | Scratch binaries evaporate: this session's scratch binary was deleted during cleanup and had to be rebuilt mid-investigation | This session, 2026-08-28 (~11:40) | Confirmed — binary-freshness tooling (A1) addresses real, recurring friction |
| V5 | Shared checkouts carry cross-agent dirty files | This session: `internal/ai/ollama/client.go` modified by another agent mid-sprint; stash collision | Confirmed |
| V6 | Sprint JSONs use a constrained-modification contract (only `passes`/timestamps/`notes` change) | Schema created today: `.ailang/state/sprints/sprint_M-*.json` | Confirmed — currently enforced by nothing mechanical |
| V7 | Full `make test` takes minutes (coordinator suite alone 28s, and flaky) | Timed this session | Confirmed — targeted ceremony has real value |
| V8 | No prior design doc covers pi as a *development* harness | `m-exec-pi-harness.md` (v0.14.2, implemented) covers pi as an *executor* for evals — complementary, cited below | Confirmed |
| V9 | A1's freshness stamp mechanism exists and is machine-readable | Ran `ailang version` live: prints `Commit: 46abe77` + `Full: <hash>[-dirty]` + `Built: <RFC3339>`. The `-dirty` suffix on Full is itself the DIRTY-TREE signal — no stamp extraction beyond parsing this output. Fallback: version output without a Commit line → A1 reports UNKNOWN (fail-closed, never false-FRESH) | Confirmed |
| V10 | A4's automated commands exist and behave as claimed | `UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot` is the documented update path (builtin_golden_types_test.go:28) and was RUN LIVE this session (regenerated the golden for the two new builtins); `ailang doctor builtins` was run live (clean; per-module counts incl. `std/fs: 20`) | Confirmed |

## Problem Statement

The repo has parallel ergonomics workstreams for Claude Code (hooks, rules, skills, `.md` context) and Codex. Pi is the harness this repo's daily work actually runs in — language development in Go (Stream A) and AILANG code creation (Stream B) — but its extension surface is used for exactly one thing (the session gate). Friction observed first-hand in one working day, all of it recurring:

**Stream A (building AILANG in Go):**
1. **Binary drift is invisible until it bites.** Agents build to ephemeral scratch dirs (V4), install mid-run from dirty trees (the 11:18 install shipped an agent's in-flight ollama edits), or run stale installed binaries and hit confusing in-stdlib type errors (M-FS-RENAME downstream). No command answers "is my binary fresh vs HEAD?"
2. **Sprint JSON is guarded only by convention** (V6): the constrained-modification contract is real but unenforced — a clumsy multi-field edit corrupts the executor's state file.
3. **Cross-agent tree ownership is unguarded** (V5): `git stash`/`git add` can sweep another agent's in-flight work; nothing warns.
4. **Post-builtin ceremony is tribal**: golden refresh (`UPDATE_GOLDEN=1 …`), `doctor builtins`, `builtins list` diff, CHANGELOG — four steps, each rediscovered per sprint.

**Stream B (writing AILANG):**
5. **The check loop is bash-parse-shaped**: agents write `.ail`, run `ailang check` via bash, grep text for codes/spans. The diagnostics are structured at the source (V3) but arrive as prose.
6. **Builtin inventory is a context escape**: `builtinHint` says "run `ailang builtins list`", so the agent shells out and re-parses ~180 signatures of text instead of calling for exactly what it needs.

## Goals

**Primary Goal:** Make pi the best-supported harness for both AILANG work modes, via a small set of single-purpose project extensions that wrap — never re-implement — the existing `ailang` CLI, shipped through git like the session gate.

**Success Metrics:**
- Each extension independently loadable/disableable, e2e-validated in `-p` and TUI
- Zero duplication of `ailang` CLI logic (wrap + structure, not re-implement)
- Every extension documents its tested pi version and degrades gracefully (pi warns, session continues)
- Both streams measurable: Stream A guards fire on their incident patterns; Stream B tools cut check-loop round-trips (code-tagged diagnostics in one tool call)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Several small single-purpose extensions, not one mega-extension | Failure isolation (a broken guard must not take down the gate), independent `/reload`, per-extension disable via pi config | agent | design | low |
| Extensions wrap the CLI as subprocesses; zero re-implementation of `ailang` logic | Re-implementing drifts from the CLI (CLAUDE.md principle 1: existing tools first); wrapping keeps one source of truth | agent | design | low |
| Stream B ships as **custom tools callable by the LLM** (registerTool), not bash advice | Structured diagnostics in one call replace grep-over-CLI-text — the pi-native analogue of Claude Code hooks: capability at the right moment | agent | design | low |
| Stream A guards use `tool_call` interception (A3) and commands (A2/A4); A1 is a report tool + command | Interception for mechanical prevention; commands for human-invoked ceremony | agent | design | low |
| All extensions live in-repo under `.pi/extensions/` (git-distributed), not a global pi package | Cross-machine inheritance via pull is the established pattern (session gate precedent) | human (confirm) | design | med |

### Design Freeze

- [ ] Confirm the wrap-don't-reimplement boundary and the per-extension split (D-rows above)
- [ ] Confirm Stream B tool names (`ailang_check`, `builtins_search`) — they enter every agent's tool surface

## Patterns proven in motoko (adopt selectively, measurement-first)

Motoko (`arniwesth/motoko_agent`, mapped in [MOTOKO.md](../../../MOTOKO.md)) is this repo's own
AILANG-native harness, explicitly modeled on pi's extension system — which makes it both a
sibling workstream and the **experimental lab** for harness techniques: it runs extension
profiles (`compaction_ai`, `context_mode`, `ailang_docs`, `microrag`, `fmt`, `dp7`) as
discrete arms and A/B-measures them on real eval rotations (e.g. `ollama_microrag` vs
`ollama` baseline). Arni is mid-refactor of motoko with further techniques.

Implications for this workstream:
1. **Port proven, not plausible.** Techniques motoko has *measured* as wins (e.g. fmt
   auto-lint closing the fmt-feedback loop; micro-rag for context retrieval; compaction
   that survives long missions) are the priority candidates for pi ports. Techniques still
   under experiment wait for their arms to report.
2. **`ailang fmt` auto-lint is already half-shipped on the pi side** — the repo's `.claude/`
   fmt hooks fire `ailang fmt` on save for Claude Code shells (see `.claude/fmt_hook_events.jsonl`);
   a pi equivalent is a natural early Phase 1.5 candidate: a `tool_call` post-hook (or the
   file-trigger pattern from pi's examples) running `ailang fmt` after `edit`/`write` on `.ail`
   files, so pi agents get the same deterministic-formatting feedback motoko arms get.
3. **Coordination**: motoko is mid-refactor — port techniques from the post-refactor shape,
   coordinate through the motoko-mission channels rather than pulling from stale branches
   (MOTOKO.md: only `dev/mk-ast` at `sunholo/eval-canonical` is canonical).
4. **Symmetry is the point**: pi extensions inform motoko's design; motoko's measured
   techniques feed pi. This workstream is the pi-side half of that loop.

## Subprocess contract (applies to every extension that shells out)

All five extensions spawn subprocesses (`ailang check`, `ailang builtins list --json`, `git`,
`go test`). Bounded waits apply to any tool that spawns a subprocess — a hung `ailang check`
(e.g. a stale or broken binary, exactly A1's DIRTY-TREE scenario) must degrade to a structured
failure, never an indefinite block:

| Property | Contract |
|---|---|
| Timeouts | `ailang check` 30s · `builtins list --json` 15s · `git` read ops 10s · golden refresh (`go test`) 120s · `doctor` 30s |
| On timeout | Kill child; return structured `{code:"TIMEOUT", command, elapsedMs}` — no silent retry; the caller decides |
| Output caps | 64KB per stream; truncated output carries `{truncated: true}` |
| No silent retries | Fail loud once; retrying is the caller's explicit choice |
| Env | Subprocesses inherit cwd; `AILANG_FS_SANDBOX` unset unless a tool explicitly sets it |

## Stream B dogfooding findings (2026-08-28, sunholo/ail_diag@0.1.0 published)

A full package lifecycle (init → write → check → test → publish → install-round-trip)
was executed in pi, cataloguing friction for the next iteration:

**Tooling papercuts (fast-track candidates):**
1. `init package` scaffolds into cwd — pollutes parent dirs that hold sibling projects
2. `init package` generates hyphenated `[exports]` that the validator REJECTS
   (PAR_HYPHEN_IN_MODULE vs "must start with package name") — an unsatisfiable manifest
   from the tool's own output
3. Hyphenated package names are unsatisfiable; the validator error never hints "use
   underscores like email_parse" (the house convention)
4. `test --package` discovers only `*_test.ail` files; inline `test` blocks (the
   convention real packages use) need per-file `ailang test <file>` — two conventions,
   one discoverer
5. Handwritten manifests surface required `edition` only as a lock-time error
6. MOD010 relaxed warnings ×N per test run = agent-visible noise (#575 quirk again)
7. No Eq instances for records/Options → span assertions need match-helpers
   (`derive Eq` or ==-for-Options would streamline test writing)

**Language learnings:** lambda separator is `.` not `->` (error was precise — GOOD);
chained `let` needs `in` per binding (positional errors cost 2 rounds); Option/record
Eq absence shapes test style. Positives: check loop fast, spans precise, pure-effect
package validates with ceiling [], publish one command, registry install round-trip
works.

**Agent-behavior finding:** the author used bash+grep out of habit even where the new
`ailang_check`/`builtins_search` tools applied — tooling exists; agents need to reach
for it (a prompt/gate-level nudge, not new tools).

## Solution Design

### Overview

Five extensions across two streams, each self-contained (≤150 LOC), each wrapping one CLI capability, each with its own dotfile tests. Shared helpers are deliberately **not** extracted until duplication exceeds ~50 LOC across extensions (YAGNI; the session gate stays self-contained).

### Stream A — building AILANG in Go

**A1 `binary-freshness` (report tool + `/fresh` command)**
- Compares: installed `~/go/bin/ailang` build stamp vs `git rev-parse HEAD` vs dirty state. Reports FRESH / STALE(<n> commits) / DIRTY-TREE(warn: rebuilding from this tree ships uncommitted work — the 11:18 incident class).
- Tool form (`freshness_report`) so the LLM can check before invoking `ailang` after pulls; command for humans.
- Never auto-installs — reports and prints the exact safe command. Mid-run installs stay a human decision (trap 6).

**A2 `sprint-steward` (commands)**
- `/sprint:start <id>` / `/sprint:complete <milestone>`: updates only `passes`/`started`/`completed`/`notes` in the named sprint JSON, then schema-validates (no placeholder IDs, LOC sum matches, ≥2 features). Rejects everything else — the constrained-modification contract becomes mechanical.

**A3 `unowned-dirty` warning (tool_call + notify)**
- **Honest scope (round-2 redesign):** parsing "the command's file set" from arbitrary bash is
  impossible pre-execution (`git add .`, `$(find …)` evaluate dynamically), and "ownership" inferred
  from an in-memory session set is a heuristic, not authority. The guard therefore claims neither.
  It is a **visibility warning**, mechanically grounded in git itself:
  - Session's own files = files written via this session's `edit`/`write` toolCalls (getBranch
    reconstruction, V10 pattern — the same mechanism the shipped session gate uses, live-verified).
  - Trigger: bash command matches `git add`/`git stash`/`git checkout`. The hook runs
    `git status --porcelain` (the AUTHORITY for what is dirty — 10s subprocess timeout) and warns:
    `"This operation may sweep N dirty file(s) not written by this session: […]"`.
  - Never blocks, never claims ownership — it converts a silent sweep into a visible one, and its
    output names itself a heuristic.
- If this session wrote no files (e.g. a triage session), the warning fires only when unowned dirty
  files exist — pure git authority, zero inference about other sessions.

**A4 `builtin-sprint` (command)**
- `/builtin:finish`: chains golden refresh → `doctor builtins` → `builtins list --json` diff vs previous → CHANGELOG stub location hint. The four-step ceremony in one command, output structured.

### Stream B — writing AILANG

**B1 `ailang_check` (custom tool)**
- Input: `{path}`. Runs `ailang check <path>` as subprocess, parses output into structured diagnostics: `{code: "IMP010"|"TCxxx"|…, message, file, line, col, hint?}` (codes verified V3).
- Returns the structured array as the tool result — the model acts on codes+spans directly, no bash grep. Composes with `builtinHint`: hint text passes through intact.

**B2 `builtins_search` (custom tool)**
- Input: `{query?: string, module?: string}`. Shells `ailang builtins list --json` (V1), filters (substring over name/module/signature/description), returns the matched slice — not the 180-entry firehose. Composes with `builtinHint` errors: the hint says "run `ailang builtins list`"; the agent calls this tool and gets exactly the candidates.

### Implementation Plan

**Phase 1 — Stream A (fast-track, trivial-scale each, stated changes):**
- [ ] A1 `binary-freshness.ts` (~80 LOC + tests)
- [ ] A2 `sprint-steward.ts` (~70 LOC + tests)
- [ ] A3 `tree-ownership.ts` (~60 LOC + tests)
- [ ] A4 `builtin-sprint.ts` (~50 LOC; thin over shell steps)

**Phase 2 — Stream B:**
- [ ] B1+B2 as `.pi/extensions/ailang-lsp-lite.ts` (~90 LOC + tests; both tools in one file — they share the subprocess+parse helper)

**Phase 3 — Hardening:**
- [ ] F5 (from session gate): verify coordinator-session inheritance; document or follow-up
- [ ] Compat matrix: pin tested pi version per extension; smoke-test after `pi update`
- [ ] Extend `.pi/extensions/README.md` index; AGENTS.md pointer if the tool surface changes agent guidance

### Files to Modify/Create

**New files:**
- `.pi/extensions/binary-freshness.ts`, `sprint-steward.ts`, `tree-ownership.ts`, `builtin-sprint.ts`, `ailang-lsp-lite.ts` (~350 LOC total)
- `.pi/extensions/.<name>.test.ts` for each (dot-prefixed, discovery-skipped)
- `.pi/extensions/README.md` — extend existing

**Modified files:**
- `CHANGELOG` (v0.35.0 DX section), AGENTS.md pointer if tool surface changes

## Examples

### Example 1: Stream B loop with tools (before/after)

**Before:** agent writes `.ail` → bash `ailang check f.ail` → greps text output → repeats. Builtin discovery: bash `ailang builtins list` → 50KB firehose.

**After:** agent writes `.ail` → calls `ailang_check(f.ail)` → `[{code:"IMP010", message:"symbol 'nth' not exported by 'std/fs'", line: 3, …}]` → calls `builtins_search({query:"nth"})` → `[{name:"_list_nth", module:"std/list", signature:"…"}]` → fixes import. Two tool calls, zero parsing.

### Example 2: Stream A freshness guard

`freshness_report` → `STALE: installed binary built from 5b6f259, HEAD is 8ea0ed62 (2 commits ahead); tree has 3 dirty files owned by other sessions (ollama/client.go …) — rebuild with: make quick-install (attended), or scratch-build.`

## Success Criteria

- [ ] Each extension loads clean in TUI and `-p` (no diagnostics), disables independently (acceptance: `pi config`)
- [ ] A1 correctly classifies FRESH/STALE/DIRTY on this repo (acceptance: scripted)
- [ ] A2 refuses non-conforming JSON edits (acceptance: scripted — corrupt attempt leaves file unchanged)
- [ ] A3 warns on intersecting-but-foreign git operations, never blocks (acceptance: scripted)
- [ ] B1 returns code-tagged diagnostics for a seeded IMP010 and a TC error (acceptance: scripted)
- [ ] B2 returns ≤10 matches for typical queries, full inventory on empty query (acceptance: scripted)
- [ ] Session gate still arms/acks correctly with all extensions loaded (acceptance: manual)
- [ ] All tests passing; documentation updated

## Testing Strategy

**Unit tests:** pure predicates per extension (freshness classification, JSON-constrained write, ownership intersection, output parsing) — node strip-types, same pattern as the gate.

**E2E:** `-p` scripted runs per extension against this repo; one TUI manual pass.

**Manual:** Studio + cloud pull, confirm inheritance and no load warnings.

## Deferred Decisions

- Whether A1 eventually gets a *safe* auto-install lane (worktree-isolated build → swap) — agent may propose v2; mid-run installs stay human-owned for now.
- Whether B1 graduates to a real LSP bridge — agent may propose separate doc.
- Extension packaging as a distributable pi package for cross-repo use — deferred (session-gate doc Future Work covers the mechanism).

## Non-Goals

- **Re-implementing any `ailang` subcommand in TS** — wrap and structure only.
- **Claude Code / Codex parity** — separate streams own those harnesses; interop note only (pi's `claude-rules.ts` example shows rules-loading is possible if wanted later).
- **Blocking beyond A3's warn** — the session gate owns blocking; this workstream adds guards, not new gates.
- **Editing `.ailang/state/sprints/*.json` fields other than the four** — the executor contract is fixed.

## Timeline

**Phase 1** (1 session): Stream A, four extensions + tests.
**Phase 2** (1 session): Stream B tools + e2e loops.
**Phase 3** (0.5 session): hardening, compat matrix, README index.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Extension accumulation slows every pi startup | Low | All extensions are event-driven and cheap; measure startup before/after; keep each ≤150 LOC |
| Wrapped CLI flags change under the wrappers | Med | Each wrapper pins + reports the `ailang version` it ran; A1 makes drift visible |
| B1/B2 tool surface confuses models (too many tools) | Low | Two tools, narrowly described; `promptSnippet` guidance per pi docs |
| Tree-ownership heuristic false-positives | Low | Warn-only, never blocks; per-session ownership set is conservative |

## Related Documents

<!-- Auto-populated by neural search on "dx pi harness"; duplicate gate passed (max 0.38 = own sprint plan) -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_14_2/m-exec-pi-harness.md](design_docs/implemented/v0_14_2/m-exec-pi-harness.md) (0.34) — pi as eval EXECUTOR; this doc covers pi as DEV harness (complementary)
- [design_docs/implemented/v0_18_5/m-ext-scaffold-ai-first.md](design_docs/implemented/v0_18_5/m-ext-scaffold-ai-first.md) (0.35) — extension scaffolding precedent

**Planned (check for overlap):**
- [design_docs/planned/v0_35_0/m-dx-session-gate-sprint-plan.md](design_docs/planned/v0_35_0/m-dx-session-gate-sprint-plan.md) (0.38) — shipped sibling; this doc extends the same pattern
- [design_docs/planned/v0_35_0/m-dx-session-protocol-gate.md](design_docs/planned/v0_35_0/m-dx-session-protocol-gate.md) (0.35) — the gate this workstream composes with

## References

- pi 0.84.3 `docs/extensions.md` — registerTool / tool_call / commands (inherited V1–V12)
- [MOTOKO.md](../../../MOTOKO.md) — the motoko checkout map (one clone, worktrees, canonical branch); read before any motoko-side coordination
- `internal/importhint` + `internal/types/import_hint.go` — the hint machinery `builtinHint` extended; B1 parses its output codes
- CLAUDE.md principle 1 (existing tools first) — the wrap-don't-reimplement rule

## Future Work

- Cross-repo distribution as an installable pi package
- Real LSP-grade diagnostics for Stream B (separate doc)
- Coordinator-session coverage once F5 is verified
- pi-side ports of motoko's measured techniques as its post-refactor arms report: compaction_ai, microrag, dp7 (dynamic prompts) — each port gets its own fast-track stated change, measurement cited

---

**Document created**: 2026-08-28
**Last updated**: 2026-08-28