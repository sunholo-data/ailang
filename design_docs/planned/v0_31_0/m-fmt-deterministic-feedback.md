# M-FMT-DETERMINISTIC-FEEDBACK: `motoko_ext_fmt` Teaches, Instead of Tidying in Secret

**Status**: Planned
**Target**: v0.31.0
**Priority**: P1
**Estimated**: 1 day
**Created**: 2026-07-29
**Dependencies**: `sunholo/motoko_ext_fmt` 0.1.1 (published), [M-EVAL-MEASUREMENT-CONTRACT](../../implemented/v0_31_0/m-eval-measurement-contract.md) (shipped — supplies the treatment-integrity gate this is measured by)

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The correction is produced by the formatter, not by a model guessing — same input, same guidance, every time |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | 0 | No language effects |
| A4: Explicit Authority | 0 | No new access; the extension already writes and execs |
| A5: Bounded Verification | +1 | Feedback is bounded to a diff/diagnostic, not an open-ended explanation |
| A6: Safe Concurrency | 0 | Unchanged |
| A7: Machines First | +1 | The whole point: the correction is delivered where a machine consumes it, in the tool result |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | This is a TOKEN argument. A 3-line diff (~30 tokens) replaces a multi-turn blind spiral (thousands) |
| A10: Composability | +1 | Same sink + same tool-result channel; nothing new to integrate |
| A11: Structured Failure | +1 | exit-3 stops being an opaque number and becomes the diagnostic that already exists |
| A12: System Boundary | 0 | Unchanged |

**Net Score: +7** → **Decision: Move forward**

- [x] A1, A3, A4, A7 — no hard violations

---

## Problem Statement

`motoko_ext_fmt` exists so that **motoko auto-formats `.ail` on write** — the model should not spend tokens invoking `ailang fmt` itself — and so the model gets **deterministic guidance to self-correct**, cutting the tokens a weak model burns rediscovering AILANG syntax.

It does the first half. It does not do the second at all.

### What the model actually receives

Every `WriteFile` returns exactly one of two strings:

```
Wrote 797 bytes to .../solution.ail; formatted (ailang fmt --write)
Wrote 797 bytes to .../solution.ail; fmt exit 3 — left as written
```

The file is corrected **on disk**. The model's context still holds *its own drifted text*. It never sees what changed, so next turn it edits from the drifted version and reintroduces the same mistake — which fmt silently fixes again. The loop never closes. That is the exact token waste this extension was built to prevent, running in reverse.

### The measured shape of the waste

From a live ON-arm run on `run_length_encode` (2026-07-29, session `9ecca597`):

```
step 0: ReadFile
step 1: WriteFile → fmt exit 3 — left as written
step 3: WriteFile → fmt exit 3 — left as written
step 4: WriteFile → fmt exit 3 — left as written
step 5: WriteFile → fmt exit 3 — left as written
step 6: WriteFile → fmt exit 3 — left as written
step 7: WriteFile → fmt exit 3 — left as written
step 8: WriteFile → fmt exit 3 — left as written
step 9: BashExec   ← the first time anything is checked
```

**Seven consecutive writes of unparseable code, and the model never reacts** — because `fmt exit 3` names an exit code. There is nothing in it to correct *from*: no location, no rule, no example. Those seven steps are pure token burn on a 35B local model, and they are precisely the "compile-stuck spiral" the fmt thesis targets.

### The guidance already exists and is discarded

`ailang fmt` on that same drifted input emits (verified directly, 2026-07-29):

```
Error: solution.ail: parse error: PAR020 at solution.ail:4:7:
missing ';' between block statements (found `x` where `;` was expected)

Did you mean one of these?
  Statements inside a `{ }` block are separated by `;`:
      { let x = e1; let y = e2; result }
  Add a `;` after the previous statement, before this one.
  The block's LAST expression is the return value — no `;` after it.
```

That is deterministic, rule-level teaching with a worked example — exactly the artifact this extension was supposed to deliver. `run_fmt` throws it away:

```ailang
func run_fmt(path: string) -> int ! {Process} {
  match exec("ailang", ["fmt", "--write", path]) {
    Err(_) => 1,
    Ok(out) => out.exitCode      -- out.stdout / out.stderr DISCARDED
  }
}
```

The correction is in `out`, in hand, at the moment of the write. It is narrowed to an integer and rendered as `exit 3`.

### And when fmt succeeds, the correction is equally invisible

For parseable-but-drifted code, fmt rewrites the file and the model is told `"formatted"`. A real example:

```diff
-  let xs = [1,2,3];
-  let ys = map(\x. x * 2,
-    xs);
+  let xs = [1, 2, 3]
+  let ys = map(\x. x * 2, xs)
```

fmt removed the trailing `;` — the "`;` separates, it does not terminate; the block's last expression is the return value" rule, one of the headline dialect traps in the teaching card. The model got that wrong, was corrected on disk, and told `"formatted"`. It will get it wrong again next turn.

**Who is affected:** every weak-model AILANG run. This is the mechanism meant to make weak models cheap to teach.

---

## Goals

**Primary goal:** When fmt corrects the model, the model *sees the correction* — deterministically, inline, at the moment it made the mistake.

**Success metrics:**

1. A write of drifted-but-parseable code returns the **diff** of what fmt changed.
2. A write of unparseable code returns the **parse diagnostic including its `Did you mean` block** — never a bare exit code.
3. A write of already-canonical code returns **no extra tokens** (silence is correct when there is nothing to teach).
4. Measurable: on the drift-prone benchmark set, ON-arm runs show fewer consecutive unparseable writes before first correction than today's 7.

---

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who decides | Cost to change |
|---|---|---|---|---|---|
| D1 | What to return when fmt CHANGED the file | (a) full canonical content · (b) unified diff · (c) status only (today) | **(b)** — the diff isolates the delta so the model reads "what I got wrong" rather than diffing 200 lines itself; and it costs tokens only when there IS drift | Mark | Low |
| D2 | What to return on exit 3 | (a) silent (CLI contract clause 5) · (b) full diagnostic incl. `Did you mean` | **(b)** — clause 5's silence is right for a *background CLI hook* where a human sees their editor. Here the model is the only consumer and silence is what produced the 7-write spiral | Mark | Low |
| D3 | Diff size ceiling | (a) unbounded · (b) truncate with a count | **(b)** — a whole-file rewrite must not blow the context window the extension exists to conserve | Agent | Low |

### Design Freeze

- [ ] D1 confirmed (b) — return the diff
- [ ] D2 confirmed (b) — surface the diagnostic, deliberately diverging from clause 5 for the extension

---

## Solution Design

### Overview

Stop discarding what `ailang fmt` already produces. Three cases, three behaviours.

| fmt outcome | today | proposed |
|---|---|---|
| exit 0, file unchanged | `"formatted (ailang fmt --write)"` | `"canonical"` — terse, no teaching needed |
| exit 0, file **changed** | `"formatted (ailang fmt --write)"` | the **diff**, labelled as canonical AILANG |
| exit 3, unparseable | `"fmt exit 3 — left as written"` | the **parse diagnostic + `Did you mean` block** |

### Architecture

```
WriteFile(path, content)
   │
   ├─ writeFile(path, content)            unchanged
   ├─ before := content                   NEW: remember what the model wrote
   ├─ (code, output) := run_fmt(path)     NEW: keep stdout/stderr, not just exitCode
   ├─ after := readFile(path)             NEW: what fmt made of it
   │
   └─ tool result message:
        code == 3        → "NOT formatted — " + output      (the teaching block)
        before != after  → "reformatted to canonical AILANG:\n" + diff(before, after)
        else             → "canonical"
```

`emit_fmt_event` is unchanged — 0.1.1 already fixed the sink. This is purely the agent-facing channel.

### Implementation Plan

**M1 — `run_fmt` stops narrowing to an int (~0.25 day).** Return `(int, string)`; keep `out.stderr`/`out.stdout`. This alone unblocks the exit-3 case, which is the highest-value half.

**M2 — diff on change (~0.5 day).** Capture `before`, re-read `after`, emit a compact unified diff. Truncate past a line ceiling (D3) with `… N more lines`.

**M3 — measure it (~0.25 day).** The treatment-integrity gate from M-EVAL-MEASUREMENT-CONTRACT already proves fmt fired. Add the metric this design is actually about: consecutive-unparseable-writes-before-first-correction, banked per run so the A/B can show the spiral shortening.

### Files to Modify

| File | Change | LOC |
|---|---|---|
| `packages/motoko-ext-fmt/register.ail` | `run_fmt` returns output; message construction; diff | +120 |
| `packages/motoko-ext-fmt/ailang.toml` | 0.1.1 → 0.2.0 (agent-visible behaviour change) | +1 |
| `mk-ast/ailang.toml` + lock | repin | +2 |
| `internal/eval_harness/fmt_treatment.go` | bank the spiral-length metric | +40 |

---

## Examples

### Today, seven times in a row

```
Wrote 797 bytes to .../solution.ail; fmt exit 3 — left as written
```

### Proposed, once

```
Wrote 797 bytes to .../solution.ail; NOT formatted — the file does not parse:

PAR020 at solution.ail:4:7: missing ';' between block statements
  Statements inside a `{ }` block are separated by `;`:
      { let x = e1; let y = e2; result }
  Add a `;` after the previous statement, before this one.
  The block's LAST expression is the return value — no `;` after it.
```

### Proposed, on successful correction

```
Wrote 797 bytes to .../solution.ail; reformatted to canonical AILANG:
-  let xs = [1,2,3];
+  let xs = [1, 2, 3]
```

---

## Success Criteria

- [ ] exit 3 returns the parse diagnostic including its `Did you mean` block
- [ ] a changed file returns a diff; an unchanged file returns no diff
- [ ] diff truncates past the ceiling rather than flooding context
- [ ] sink behaviour (0.1.1) unchanged — exit 3 still omitted from `fmt_hook_events`
- [ ] `ailang check` clean; `make check_core` 24/24 in mk-ast
- [ ] a live ON-arm run shows the diagnostic reaching the model in the transcript

## Testing Strategy

- **Unit**: message construction for all three cases; diff truncation.
- **Live**: one ON-arm run on a drift-prone benchmark; grep the session JSONL for the diagnostic text in a `native_tool_results` payload. Per this sprint's repeated lesson, the unit tests are not the proof — the transcript is.

## Non-Goals

- Changing `ailang fmt` the CLI. Its exit-3-silent contract is right for a background hook where a human sees the editor; this doc changes only the *extension*, whose sole consumer is a model.
- The Claude-Code `format_ail.sh` hook (different harness, different consumer).
- Teaching-prompt changes.

## Risks & Mitigations

| Risk | L | I | Mitigation |
|---|---|---|---|
| Diff floods context on a whole-file rewrite — the opposite of the token goal | Med | High | D3 ceiling with `… N more lines`; the diff is empty when nothing changed |
| Diagnostic text drifts and the message becomes stale | Low | Low | Pass `fmt`'s output through verbatim; never re-word it in the extension |
| The model ignores the feedback anyway | Med | Med | That is the experiment. The A/B measures it, and the spiral-length metric (M3) shows movement even if pass-rate does not |
| Arms become non-comparable with the Claude-hook arm | Med | Low | Already true (that hook is silent on exit 3); record it as a known asymmetry rather than pretending parity |

## Related Documents

- [M-AILANG-FMT-ADOPTION](../../implemented/v0_30_0/m-ailang-fmt-adoption.md) — governs the **CLI**: opt-in discoverability, exit-3 silent (clause 5). This doc deliberately diverges for the extension, where the consumer is a model rather than a human with an editor.
- [m-eval-fmt-weakmodel-ab-M6-motoko-ext.md](m-eval-fmt-weakmodel-ab-M6-motoko-ext.md) — the A/B this feeds.
- [M-EVAL-MEASUREMENT-CONTRACT](../../implemented/v0_31_0/m-eval-measurement-contract.md) — supplies the treatment-integrity gate.

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | The model sees only a status string | read `auto_write_with_fmt` in `register.ail` | CONFIRMED — two literals, no content |
| V2 | `run_fmt` discards fmt's output | read `run_fmt` | CONFIRMED — `Ok(out) => out.exitCode` |
| V3 | fmt's diagnostic carries actionable teaching | ran `ailang check --format json` on drifted input | CONFIRMED — PAR020 + location + `Did you mean` block with worked example |
| V4 | exit 3 = unparseable, incl. drifted dialect | ran `ailang fmt --write` on truncated and `for`-loop files | CONFIRMED — both exit 3 |
| V5 | 7 consecutive unreacted-to exit-3 writes | read session `run_length_encode_9ecca597` tool sequence | CONFIRMED — steps 1,3,4,5,6,7,8 |
| V6 | fmt makes real dialect corrections silently | diffed a drifted file against `ailang fmt` stdout | CONFIRMED — removed trailing `;`, normalised list spacing, collapsed a wrapped call |
| V7 | `ailang fmt <file>` (no `--write`) prints canonical source to stdout | `ailang fmt --help` + live run | CONFIRMED |

## Future Work

- Feed the same diff into the DP7 per-edit type-check path so one write yields one consolidated correction rather than two.
- If the diff proves effective, consider whether the teaching prompt can shrink — the correction arrives just-in-time, so less of it needs to be front-loaded. That is the real token win.
