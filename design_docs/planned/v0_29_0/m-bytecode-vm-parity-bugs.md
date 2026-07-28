# M-BYTECODE-VM-PARITY-BUGS — VM/Eval Divergences (Lanes A + B, scope FROZEN)

**Status**: Planned — **SCOPE FROZEN as A+B per Mark's attended GO, 2026-07-27.** Lane B is no
longer parked. This doc supersedes its own iter-102 refresh box (now itself drifted — see
"Current state"); plan from THIS version.
**Target**: v1.0.0 (clause-2 soundness residue on the V1 mission queue)
**Priority**: **P0** — `recursion_quicksort.ail` is a **silent wrong result** under `--bytecode`
(no error, no fallback, wrong list). Under NO-SILENT-FALLBACKS this outranks everything else here.
**Estimated**: 3–4 days total (Lane A ~1d; Lane B ~2–3d, investigation-first)
**Dependencies**: M-BYTECODE-MULTIMODULE M1 (complete — surfaced these)

## Current state — measured at HEAD `33be8f5a7`, 2026-07-28

Controller-verified (`go run ./scripts/verify_bytecode_parity.go`), re-confirmed per-file
first-party in this worktree (see Verification Log):

```
MATCH 149 (85.6%) | NON_DET 2 (1.1%) | DIVERGE 7 (4.0%) | EVAL_SKIP 16 (9.2%)
```

The iter-102 box claimed 150/2/6/16. **DIVERGE grew 6 → 7**: `http_simple.ail` is new. It is a
`! {IO, Net}` example doing a live `httpGet("https://httpbin.org/get")` — the same
harness-false-positive class as `claude_haiku_call.ail`. **This row appeared precisely because
the harness's exclusion mechanism is a hand-maintained filename map** (`nonDeterministic`,
`scripts/verify_bytecode_parity.go:58`) — a filename list drifts as examples are added. Lane A's
exclusion rule is therefore **effect-driven** (by declared `Net`/`Clock` effect), not another
filename entry.

### The 7 DIVERGE files (eval = ground truth, except where noted)

| # | File | Symptom at HEAD | Class | Lane |
|---|------|-----------------|-------|------|
| 1 | `recursion_quicksort.ail` | VM **silently** returns `[3]` for BOTH `Quicksort:` and `sortBy:`; eval returns `[1, 1, 2, 3, 4, 5, 6, 9]`. Exit 0, **no error, no fallback** (stderr verified clean) | **VM correctness — SOUNDNESS, silent wrong result** | **B (P0)** |
| 2 | `array_basic.ail` | VM prints `Length: <Closure>`, `numbers[0] (first): <Closure>`, … then `vm: GET_TAG on Closure (in array_basic.showOpt at array_basic.ail:32)` → **falls back loudly**, evaluator re-runs and re-prints correctly; stdout = wrong prefix + full correct output | **VM dispatch bug** (array length/index produced as unforced closures) — fails loudly, self-heals | **B** |
| 3 | `pattern_sugar.ail` | **eval** prints `firstPair(...) = <*eval.TupleValue>`; **VM is CORRECT** (`(a, 1)`). Single-line diff, verified | **eval-side show bug** — harness ground truth is the wrong side | **A** |
| 4 | `tar_gzip_reader.ail` | VM hits bridge limit (`TaggedValue (Result.Err) not yet supported (M-BYTECODE-2E scope)`) mid-run → falls back → evaluator re-runs → **first stdout line duplicated**. Fallback itself is correct | **VM_BRIDGE mis-filed as DIVERGE** (harness can't see exit-0 fallbacks) | **A** |
| 5 | `http_simple.ail` | `! {IO, Net}`, live httpGet — nondeterministic upstream body | **Harness false-positive** (NEW row; filename-list drift) | **A** |
| 6 | `claude_haiku_call.ail` | `! {Net, IO}`, live API call | **Harness false-positive** | **A** |
| 7 | `xml_walk_perf.ail` | `! {IO, Clock}`, prints `time_ms=42` vs `43` (timing jitter) | **Harness false-positive** | **A** |

Load-bearing contrast: **#2 fails LOUDLY and self-heals via fallback; #1 fails SILENTLY with a
wrong answer.** #1 is the P0. Both collapsing to the same `[3]` (hand-written quicksort AND the
`sortBy` variant) hints at a shared list/recursion codegen path — but the root cause is **not
known** and Lane B is investigation-first (Milestone B1). Do not anchor on the 2026-04-08
hypotheses in the appendix; the symptom has changed since (`<List>` → `[3]`).

## Verification Log (first-party, this worktree, 2026-07-28)

All behavior claims re-measured here with the worktree `./bin/ailang` + the parity harness
(`--only` runs); controller HEAD totals inherited as labeled above.

| Claim | How verified |
|-------|--------------|
| quicksort: VM `[3]`/`[3]`, eval correct, exit 0, **stderr has NO fallback warning** | direct `./bin/ailang run --bytecode` with stderr captured separately |
| array_basic: `<Closure>` prefix + `GET_TAG on Closure` + mid-run fallback re-print | harness `--only array_basic` (stdout shows wrong prefix + full correct re-run) |
| pattern_sugar: exactly ONE diff line; eval `<*eval.TupleValue>`, VM `(a, 1)` | unified diff of harness `--only pattern_sugar` outputs |
| tar_gzip: divergence = duplicated first stdout line from mid-run fallback re-run | harness `--only tar_gzip` (VM stdout = header ×2 then identical) |
| Effect rows: http_simple `! {IO, Net}`, claude_haiku_call `! {Net, IO}`, xml_walk_perf `! {IO, Clock}` | grep of the three files |
| eval show gap site: `internal/builtins/show.go` `showValue` has **no `*eval.TupleValue` case**; default at line 173 prints `<%T>` | read of the full switch (lines 27–174) |
| **No test/golden depends on the broken `<*eval.TupleValue>` output** | repo-wide grep — only design docs/mission log mention it |
| Harness exclusion = filename map; **no effect-driven exclusion exists** | read `scripts/verify_bytecode_parity.go:58-61,166` |
| Harness **never inspects vmStderr when vmExit==0** → exit-0 fallbacks invisible | read `verifyOne`, lines 190–218 |
| Fallback warning emitted to **stderr** at `cmd/ailang/run_helpers.go:376`, text embeds the original vmErr | read of the emission site |
| `cmd/ailang/run_bytecode_test.go:63` asserts **absence** of `"falling back to evaluator"` in non-fallback runs (pins the warning text) | read of the test |
| Net/Clock-declaring runnable examples = 9 total; besides the 3 DIVERGE rows: http_put_bytes, stdlib_game, demo_ai_api + 1 ai_call\* currently **MATCH**; ai_stream_openai + 3 ai_call\* are EVAL_SKIP | grep + harness `--only` per file |
| Regression fixtures cited below exist and currently MATCH | `--only cons_expression`, `--only block_recursion` both MATCH |
| `tests/golden/bytecode/` exists (golden_test.go etc.) | ls |

## Solution Design

### Lane A — eval-side fix + harness honesty (~1 day, no VM internals)

**A1 — eval tuple show.** Add a `*eval.TupleValue` case to `showValue` in
`internal/builtins/show.go` (site verified above): `(` + elements via recursive `showValue` +
`, ` join + `)`, matching the VM's `(a, 1)` rendering byte-for-byte. No test depends on the old
`<%T>` output (verified). `internal/eval/eval_simple_helpers.go:22` has the same gap (default
`<unknown>`) — mirror the case there for consistency, but the acceptance target is the
`builtins/show.go` path (it is the one producing the live symptom).

**A2 — harness honesty** (`scripts/verify_bytecode_parity.go` only):

1. **Effect-driven exclusion**: a file whose detected caps (the existing `detectCaps` sniffer,
   line 263 — already string-sniffs `! {Net`, `import std/clock`, etc.) include `Net` or `Clock`
   is classified `NON_DET` with reason `"declares <effect> effect"` — *before* either backend
   runs (also removes live network calls from the parity gate). The filename map stays only for
   the 2 legacy entries (`uuid.ail`, `stream_process_source.ail`) whose non-determinism is not
   effect-visible in this rule. **Measured coverage cost (intentional)**: 4 currently-MATCHing
   files (`http_put_bytes.ail`, `stdlib_game.ail`, `demo_ai_api.ail`, one `ai_call*`) move to
   NON_DET. They must be **reported in the NON_DET bucket, never silently dropped**.
2. **Exit-0 fallback detection**: when `vmExit == 0`, scan the VM run's **stderr** for the
   fallback marker (`"falling back to evaluator"`, emitted at `cmd/ailang/run_helpers.go:376`
   with the original vmErr embedded). If present, classify by the embedded error using the
   existing bridge-marker rules: bridge-scope markers (`not yet supported` / `M-BYTECODE-2E` /
   `TaggedValue`) → `VM_BRIDGE`; **anything else (e.g. `GET_TAG on Closure`) → `VM_RUNTIME`** —
   NOT MATCH, NOT VM_BRIDGE. This is what keeps `array_basic.ail` visibly red after Lane A.
   **Do NOT change the warning text** at run_helpers.go:376 — `run_bytecode_test.go:63` asserts
   its absence in non-fallback runs; the harness sniffs the same string (single source of truth;
   if it must ever change, change both).

**Predicted post-Lane-A state** (expectation, deliberately NOT an acceptance criterion):
MATCH ~146 (−4 excluded, +1 pattern_sugar) / NON_DET ~9–11 / DIVERGE 1 (quicksort) /
VM_RUNTIME +1 (array_basic) / VM_BRIDGE +1 (tar_gzip) / EVAL_SKIP ~12.

### Lane B — VM correctness (investigation-FIRST, ~2–3 days)

> #### ⚠ SEMANTIC root cause of the quicksort-class bug is NOW KNOWN — VERIFIED BY THE CONTROLLER
>
> *Added after this doc's revision. The narrowing below was run first-party by the controller in
> this worktree at HEAD `33be8f5a7`; the designer did not have it. It does NOT dissolve B1 — it
> re-scopes B1's quicksort half from "characterize the bug" to "locate the code site". The
> **code-level** cause (which function/opcode) remains genuinely unknown and B1 still owns it.*
>
> **The bytecode VM does not check the LENGTH of a fixed-length list pattern.** A pattern
> `[p1, …, pn]` matches any list of length **≥ n**, binding the first `n` elements and silently
> discarding the tail. The guard is `len >= n` where it must be `len == n`.
>
> Minimal repro (`match` with arms `[] / [x] / [a,b] / [p, ...rest]`):
>
> | input | `ailang run` (correct) | `ailang run --bytecode` |
> |---|---|---|
> | `[]` | `empty` | `empty` |
> | `[7]` | `one:7` | `one:7` |
> | `[7,8]` | `two:7,8` | **`one:7`** |
> | `[7,8,9]` | `many:7` | **`one:7`** |
>
> Confirmed general at n=1, 2 **and** 3 (`[a,b]` arm swallows `[7,8,9]` → `two:7,8`; `[a,b,c]`
> arm swallows `[7,8,9,10]` → `three:7,8,9`). Only the *overflow* direction is unchecked — a list
> **shorter** than the pattern correctly fails to match, which is why this survived so long.
>
> This fully explains `recursion_quicksort`: the `[x] => [x]` arm captures the whole 8-element
> input and returns the head as a singleton, so the answer is `[3]`. `show`, `filter` and `concat`
> were each ruled out first (all MATCH under both engines), as were recursion-as-argument and
> closure capture of a pattern-bound variable. `sortBy`'s identical `[3]` is the same arm in
> `std/list`, not a second bug.
>
> **Impact is far wider than one example**: every fixed-length list pattern in every `--bytecode`
> program can silently return a wrong answer. Filed as **[#505](https://github.com/sunholo-data/ailang/issues/505)**.
> B2's fix and goldens must cover the pattern-arity table above, not merely the quicksort output.
>
> `array_basic` is **not** yet explained by this and may still be an independent defect — B1 must
> not assume one shared cause on the strength of this finding.

**The CODE-LEVEL root cause is UNKNOWN. No fix is designed in this doc**; B2/B3 implement whatever
B1 names, and if B1 finds one shared cause (quicksort's arity bug and array_basic's closure-typed
elements may still share a list/closure codegen path), B2+B3 merge.

**B1 — diagnose (no fix).** For each bug:
- Shrink to a **minimal failing `.ail` repro**. For the quicksort class, start from the
  pattern-arity table above (already minimal — do not re-derive it); for array_basic, from a
  bare `length(arr)` print.
- `ailang disasm` the repro; compare the bytecode against the expected lowering; trace VM
  execution (`DEBUG_STRICT=1`, existing VM tracing) to the first wrong value.
- Cross-check the eval path on the same Core IR to localize eval-vs-lower-vs-VM.
- **Deliverable: a written root-cause statement naming the defective function/opcode path**,
  plus the minimal repro committed under `tests/golden/bytecode/`. Prior hypotheses (appendix,
  iter-102) are stale — verify, don't inherit.

**B2 — fix the quicksort-class bug** (P0). Fix at the layer B1 named
(`internal/gen/lower/` / `internal/vm/` / `internal/bytecode/`); golden regression test
asserting the **exact sorted output** under `--bytecode`.

**B3 — fix the array_basic dispatch bug.** Array length/index results must be forced values,
not closures; regression test asserting exact output under `--bytecode` **with no fallback**
(assert stderr does not contain `"falling back to evaluator"`).

## Milestones & Acceptance Criteria

Each AC belongs to exactly ONE milestone and names the file that can fail it. Milestones A and B
are separable: a Lane-B overrun cannot strand Lane A (A1+A2 land and are gate-effective alone).

### Milestone A1 — eval tuple show (~2h)
- **AC1**: `examples/runnable/pattern_sugar.ail` is MATCH in the parity harness (fails today:
  eval side prints `<*eval.TupleValue>` at the `firstPair` line).
- **AC2**: unit test in `internal/builtins/` asserting `showValue` on a
  `*eval.TupleValue{("a",1)}` returns `(a, 1)` — fails on HEAD (case absent, default `<%T>`).

### Milestone A2 — harness honesty (~0.5d)
- **AC3**: `examples/runnable/tar_gzip_reader.ail` reports `VM_BRIDGE` (not DIVERGE) — fails
  today because exit-0 fallbacks are invisible to the harness.
- **AC4**: `http_simple.ail`, `claude_haiku_call.ail`, `xml_walk_perf.ail` report `NON_DET`
  with an effect-derived reason, and the NON_DET bucket lists every excluded file by name in
  the text/markdown/JSON reports (silent-drop fails this).
- **AC5** (anti-vacuous guard): after A2 (with A1 landed, B not landed),
  `recursion_quicksort.ail` still reports **DIVERGE** and `array_basic.ail` reports
  **VM_RUNTIME** — i.e. re-categorization did NOT absorb either Lane-B bug. If either goes
  green without a Lane-B fix commit, A2 is wrong.

### Milestone B1 — diagnose both VM bugs, NO fix (~1d)
- **AC6**: minimal failing repro for the quicksort `[3]` collapse committed under
  `tests/golden/bytecode/` + a written root-cause naming the defective codegen/VM path
  (in the sprint notes and this doc's implementation report). "It's somewhere in lowering"
  does not pass; a named function/opcode path does.
- **AC7**: same deliverable for array_basic's `GET_TAG on Closure` (may name the same root
  cause; must say so explicitly if so).

### Milestone B2 — quicksort-class fix (P0, ~0.5–1d)
- **AC8**: golden test asserting `examples/runnable/recursion_quicksort.ail` under `--bytecode`
  prints exactly `Quicksort: [1, 1, 2, 3, 4, 5, 6, 9]` and
  `sortBy:    [1, 1, 2, 3, 4, 5, 6, 9]` — fails on HEAD (`[3]`). This AC is the one a
  DIVERGE-count AC cannot fake.
- **AC8b** (generality — the bug is #505, not one example): a table-driven test asserting
  **fixed-length list-pattern arity** under `--bytecode` for n=1, 2 and 3, in both directions:
  a pattern of length `n` must match a list of length exactly `n`, must NOT match a **longer**
  list (fails on HEAD: `[x]` swallows `[7,8]`, `[a,b]` swallows `[7,8,9]`, `[a,b,c]` swallows
  `[7,8,9,10]`), and must NOT match a **shorter** one (passes on HEAD — include it so a fix that
  over-corrects into the underflow direction is caught). AC8 alone is satisfiable by a
  quicksort-shaped special case; this AC is what forces the real fix.
- **AC9**: B1's minimal quicksort repro passes under `--bytecode` with **no fallback**
  (stderr asserted free of `"falling back to evaluator"`).

### Milestone B3 — array dispatch fix (~0.5d)
- **AC10**: `examples/runnable/array_basic.ail` under `--bytecode` prints `Length: 5` /
  `numbers[0] (first): 10` (exact eval output) with no fallback-warning line on stderr —
  fails on HEAD (`<Closure>` + fallback).

### Sprint-exit gate (whole doc)
- **AC11**: full parity harness at sprint end reports **DIVERGE 0 and VM_RUNTIME 0**, with
  MATCH ≥ 146, and no currently-MATCHing file regresses (spot fixtures below). Valid ONLY in
  conjunction with AC8/AC10 — the count alone is fakeable by re-categorization, which AC5
  already forbids.

## Conflict Surface (mandatory — codegen)

**Positions/files this touches, and what else lives there:**

1. `internal/builtins/show.go` (`showValue`) — the show surface for **every eval-path program**.
   Adding a `*eval.TupleValue` case changes output for any program that shows a tuple. Verified:
   no test/golden anywhere asserts the current `<*eval.TupleValue>` text. The new rendering must
   match the VM's (`(a, 1)`, elements via recursive `showValue`, strings unquoted per this
   file's existing `StringValue` case) or A1 trades one divergence for another.
2. `scripts/verify_bytecode_parity.go` — classification precedence changes to:
   effect/filename NON_DET → EVAL_SKIP → exit≠0 classes → **exit-0 fallback sniff**
   (VM_BRIDGE vs VM_RUNTIME) → MATCH/DIVERGE. The sniff MUST discriminate on the embedded
   error, else genuine VM bugs (array_basic) get laundered as VM_BRIDGE (AC5 guards this).
3. `cmd/ailang/run_helpers.go:376` — the fallback warning is now load-bearing for the harness.
   Its text is also pinned by `cmd/ailang/run_bytecode_test.go:63` (asserts absence in
   non-fallback runs). **Constraint: don't reword it**; if reworded, update harness + test
   together.
4. `internal/vm/` / `internal/gen/lower/` / `internal/bytecode/` (Lane B) — exact files unknown
   until B1 names the root cause; whatever changes, the whole-corpus harness is the blast-radius
   detector. List/closure codegen is shared by *all* list-recursive programs, not just the two
   red files.
5. **Intentional incompatibilities**: (a) eval tuple-show output changes (broken → correct);
   (b) 4 currently-MATCHing Net/Clock examples (`http_put_bytes.ail`, `stdlib_game.ail`,
   `demo_ai_api.ail`, one `ai_call*`) move MATCH → NON_DET by the effect rule — a deliberate,
   reported coverage trade documented in A2; (c) headline MATCH count drops ~149 → ~146 for
   honest reasons.

**Programs that MUST still work post-change (verified to exist; first two verified MATCH at
HEAD):** `examples/runnable/cons_expression.ail`, `examples/runnable/block_recursion.ail`,
`examples/runnable/adt_list_fields.ail`, `examples/runnable/effectful_list.ail`, plus the
non-tuple lines of `pattern_sugar.ail`. And `cmd/ailang/run_bytecode_test.go` must stay green.

## Testing Strategy

- Unit: tuple case in `internal/builtins` show tests (AC2); harness classification is exercised
  via the live corpus (the script is `go:build ignore` — no unit harness; ACs 3–5 are its test).
- Golden: minimal repros from B1 + exact-output goldens (AC8–AC10) under `tests/golden/bytecode/`.
- Whole-corpus: full parity harness before/after every Lane-B change (AC11 + fixture spot-check).

## Non-Goals

- ADT constructor name rendering (M3 of M-BYTECODE-MULTIMODULE scope).
- Rewriting show dispatch or type-class dictionary lowering (take the targeted fix B1 justifies).
- The EVAL_SKIP set (evaluator itself fails: missing AI keys / intentional exit codes).
- Any fix not named by B1 — no speculative VM patches.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| B1 finds the quicksort cause in a shared lowering path with wide blast radius | High | Whole-corpus harness after every change; fixtures pinned above; B2 lands behind AC8's exact-output golden |
| Effect-driven exclusion hides a future genuine VM bug in a Net/Clock example | Medium | NON_DET bucket always lists members by name (AC4); coverage trade documented here |
| Fallback-marker string drifts from harness sniffer | Low | Single source of truth constraint in Conflict Surface #3; both sites named |
| Lane B overruns the 3–4d budget | Medium | Lanes separable by design; A1+A2 land independently; B1's named root cause makes any handoff/park cheap and concrete |

## Related Documents

- [design_docs/implemented/v0_11_0/m-bytecode-vm.md](../../implemented/v0_11_0/m-bytecode-vm.md) — M-BYTECODE-VM master design
- [design_docs/implemented/v0_11_0/m-bytecode-2d-parity.md](../../implemented/v0_11_0/m-bytecode-2d-parity.md) — parity harness origin; documents the fallback warning
- [design_docs/implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md](../../implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md) — parent sprint (shipped)
- [design_docs/planned/v1_1_0/m-perf4-bytecode-interpreter.md](../v1_1_0/m-perf4-bytecode-interpreter.md)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Silent cross-backend divergence (quicksort) eliminated; backends byte-agree |
| A7: Machines First | +1 | Parity gate becomes a trustworthy machine signal (no filename-drift false rows) |
| A11: Structured Failure | +1 | The one *silent* wrong-result path becomes impossible to ship past the gate; exit-0 fallbacks become visible, categorized statuses |
| Others | 0 | No language-surface change |

**Net Score: +3** → **Proceed.**
Hard violations: none (A1 improves; A3/A4 untouched — no effect or authority changes; A7 improves).

---

## Appendix — condensed history (superseded; do not plan from this)

- **2026-04-08 original**: 130 MATCH / 3 DIVERGE (`pattern_sugar`, `recursion_quicksort` both
  showing `<List>`; `string_parsing` dup+mojibake). The `<List>`-dispatch hypotheses written
  then are **obsolete** — the symptom has since changed to the `[3]` collapse, and
  `string_parsing.ail` now MATCHes (fixed independently).
- **2026-07-24 (iter-102, at `64f1e2924`)**: 150/2/6/16; introduced the Lane A/B split and
  parked Lane B for Mark's scope call (the item had been delegated as "3 small output
  divergences"; a silent wrong quicksort result exceeded that framing → Standing Rule 2 park).
- **2026-07-27**: Mark attended GO for full A+B → this revision freezes that scope.
- Baseline commit for the original 3 bugs: `f108cceb` (pre-M1). Sprint JSON:
  `.ailang/state/sprints/sprint_M-BYTECODE-MULTIMODULE.json`.

---

**Document created**: 2026-04-08
**Last updated**: 2026-07-28 (scope freeze A+B; fresh HEAD data; effect-driven Lane A; investigation-first Lane B)
