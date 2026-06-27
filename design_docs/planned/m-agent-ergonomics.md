# M-AGENT-ERGONOMICS: One-Cycle Self-Correction for AILANG Agents

**Status**: Planned
**Target**: v0.27.0
**Priority**: P1 (the dominant cost in agentic AILANG benchmarks is the model fighting the dialect, not the algorithm)
**Estimated**: ~3-4 days (4 milestones)
**Dependencies**: M-EVAL-RELIABLE-GRADING (the benchmark instrument that surfaces this); the LSP (`internal/lsp`)

## Problem

On the `docx_reimplement` benchmark (motoko-local, session `b5c19e15`, a *passing* run), only ~10 of
61 steps were implementation. **~9 steps were the model fixing AILANG-dialect slips** it had already
been told about — each a 2-3 step cycle (failed run → read error → fix → re-run):

| Slip (from the docx trace) | Count | Today's error |
|---|---|---|
| `length`/`join`/`repeat` used without `import std/list`/`std/string` | 3 | bare `undefined variable: length` — no remedy |
| `->` where `=>` expected (match arm) | 1 | `PAR_UNEXPECTED_TOKEN: expected =>, got ->` — no remedy |
| `,` where `;`/`}` expected (block sequencing) | 1 | `expected }`, no remedy |
| `${}` interpolation parse breaks | 2 | `PAR_NO_PREFIX_PARSE` — no remedy |
| scratch test run without `--caps IO,FS` | 1 | `effect 'IO' requires capability` — no remedy |

These are **slips of rules the model already has** (the system-prompt syntax reference *and* the
"dialect traps" card are both present every turn). A small local model reverts to Python/JS habits
under task load. The current mitigation — `prompts/agent/dialect-traps.md`, prepended every turn — is
a **band-aid we want to retire**: prose telling the model "don't do X" doesn't prevent the slip; it
just adds tokens. The structural fix is to make AILANG's *feedback loop* sharp enough that a slip is
corrected in **one cycle** (or auto-applied), so the model spends steps on the algorithm.

This is iteration 1 of the standing loop: **write a big AILANG benchmark → categorize the per-step
friction → smooth the language/tooling rough edge → re-measure → move to a harder benchmark.** The
benchmark is the instrument; this doc is the first smoothing pass.

## Findings (de-risk the build)

- **The parser already has a structured-suggestion framework** — `ParserError.Suggestions`
  (`parser_error.go`), `NewSuggestionError` (`parser_decl.go`), and effect errors already emit
  "Did you mean '<effect>'?" (`parser_effect.go:83`). It is applied inconsistently — the slips above
  have no suggestion. Extending it is additive, not new infrastructure.
- **Type errors have no suggestion at all** — `undefined variable: %s` is a bare `fmt.Errorf`
  (`typechecker_literals.go:62`, `inference.go:142`). The fix needs a **symbol→module index**
  (which module exports `length`?). That index already exists for the LSP (`internal/lsp/index.go`)
  and `ailang docs`/`iface`; reuse it.
- **LSP `CodeAction` and `Completion` are unimplemented** (`internal/lsp/unimplemented.go:64,76`) —
  hover/definition/symbols/references work, but the two capabilities that would let an LSP-consuming
  agent *self-correct* (quick-fix "add import") and *discover* stdlib (completion) are empty.
- **`ailang docs <module>` / `docs --list` exist** (stdlib discovery via CLI) but agents underuse
  them; no `ailang fix` exists.

## High-Impact Decisions

| Decision | Why |
|---|---|
| **One suggestion engine, three surfaces** — a single `Diagnostic{code, span, message, suggestions[]}` (each suggestion optionally carrying a text edit) feeds CLI error output, `ailang fix`, and LSP `CodeAction` | Avoids three drifting copies of "the fix for `->` is `=>`"; the engine is the structural asset, the surfaces are thin |
| **Suggestions reuse the stdlib symbol index** (symbol → exporting module) | `undefined variable: length` → `import std/list (length)` needs the index the LSP/iface already builds |
| **`ailang fix` applies only *unambiguous* edits** | Auto-adding `import std/list (length)` when `length` is unique is safe; ambiguous cases stay a suggestion, not an auto-edit |
| **Retire the dialect-traps card once M1-M3 land** | The card is an anti-pattern (prose nags ≠ structural fix). Success is measured by removing it *without* regressing step counts |

## Solution Design / Sprint

**M1 — Actionable diagnostics for the top slips** (~1.5d)
- Extend the suggestion framework to the slip taxonomy above: `->`→`=>` in match arms, `,`→`;`/`}` in
  blocks, `${}`-in-context, and the effect-capability error → "re-run with `--caps IO,FS`". Add a
  **symbol-resolution suggestion** to `undefined variable`: look the name up in the stdlib symbol
  index and emit `import std/<mod> (<name>)` (list all exporters if >1).
- Each diagnostic carries a structured `Suggestion{message, edit?}` (the engine), rendered in CLI
  output as a one-line "fix:" and reused by M2/M3. Golden tests per slip.

**M2 — `ailang fix` (auto-apply)** (~1d)
- New non-interactive command: parse/typecheck, collect diagnostics with unambiguous edits, apply
  them, re-check, report. Idempotent; never touches ambiguous cases. The eval harness and agents can
  run `ailang fix <file>` before/after writing — collapsing the mechanical slips to **zero** cycles.

**M3 — LSP `CodeAction` + `Completion`** (~1d)
- Implement `CodeAction` from the same engine (each suggestion → a quick-fix code action with the
  edit) and `Completion` from the symbol index (stdlib functions in scope / importable). Agents
  driving the `ailang-lsp` plugin (motoko, IDEs) then self-correct inline without a run cycle.

**M4 — Discoverability + retire the card** (~0.5d)
- Errors point to the right tool (`undefined variable` → "see `ailang docs std/list`"). Replace the
  *dialect-traps* card with a compact **toolchain card** ("check with `ailang check`, fix with
  `ailang fix`, look up stdlib with `ailang docs <mod>`") — or drop the card entirely if M1-M3 carry
  the load. Re-run `docx_reimplement` (per M-EVAL-RELIABLE-GRADING) and measure the step delta.

## Conflict Surface
- M1 is additive to existing error types (`Suggestions[]` already exists); CLI rendering changes are
  output-only. `ailang fix` is a new command (no existing behavior touched). LSP changes replace
  `unimplemented` stubs. No change to the type checker's *decisions*, only its error *messages*.

## Success Criteria
- [ ] M1: each slip in the taxonomy emits a one-line actionable fix; `undefined variable` suggests the import.
- [ ] M2: `ailang fix` collapses the mechanical slips (imports, `->`→`=>`) to zero re-run cycles on the docx fixture.
- [ ] M3: LSP `CodeAction`/`Completion` implemented and exercised via the `ailang-lsp` plugin.
- [ ] M4: `docx_reimplement` step count drops measurably (target: the ~9 fix cycles → ≤3); the
  dialect-traps card is removed **without** regressing the step count or pass rate.

## Non-Goals
- Stream/connection errors (5 in the docx run) — an infra robustness track in the motoko host
  (ollama reconnect/backoff), tracked separately; not a language/diagnostics concern.
- Making the parser *accept* the wrong dialect (e.g. `->` as a match arrow). We teach the fix fast,
  we don't fork the grammar — except where a token is unambiguous and broadly expected (cf. the hex
  literal win, which was "accept what every language writes," not "accept a second AILANG dialect").

---

## The loop, codified (run this every iteration)

The point of the loop is to reach *mechanism-level* analysis on every study, not only when someone
remembers to grep. The standardized steps, with the exact commands:

1. **Measure** — N-run study on the rig (free local tokens absorb agentic path variance):
   `ailang eval-suite --agent --models <m> --benchmarks <b> --langs ailang --trials 5 ...`
2. **Aggregate friction** — `tools/aggregate_run_friction.py <session…>` → mean ± CV per category.
   High-mean **low-CV** = reliable friction to target; high-CV = noise, never act on one run.
3. **Deep-analyze every stuck run** — `tools/analyze_stuck.py <logdir>` auto-finds the `max_steps`
   sessions and, per repeated error, prints a DOSSIER: the full untruncated error **plus the model's
   own edits across each failed attempt**, ending in the invariant it never changed (the blind spot)
   and a fix-question. This bakes in the two drill-downs that used to be manual (the exact error, and
   "what did it try each time"). Example output: *"10 distinct attempts, every one keeps `show` …
   what error message makes it remove it?"* → became M-AGENT-STUCK-FIXES M2.
4. **Fix the rough edge** — design doc → execute → **deterministic** verify (reproduce the trigger,
   confirm the new message; never rely on the noisy agentic metric to prove a deterministic fix).
5. **Re-measure** — post-fix N-run; the targeted entry should vanish from `analyze_stuck.py`'s
   repeated-error set, and the aggregate `max_steps` rate / step count should move if it was load-bearing.

Tools: `tools/analyze_stuck.py` (where + why it's stuck), `tools/aggregate_run_friction.py` (signal vs
noise across N), `tools/analyze_run_steps.py` (diff two single runs in detail).

---
**Document created**: 2026-06-27

DESIGN_DOC_PATH: design_docs/planned/m-agent-ergonomics.md
