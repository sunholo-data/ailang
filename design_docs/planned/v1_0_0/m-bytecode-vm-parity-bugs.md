# M-BYTECODE-VM-PARITY-BUGS — Three VM soundness bugs surfaced as "output divergences" (Lanes A + B, scope FROZEN)

**Status**: Planned — **SPLIT by Mark's decision, 2026-08-04 (option C, see below).** The #505
pattern-arity fix (the former Milestones B1-quicksort-half + B2) has been spun out, UNBLOCKED, into
[m-bytecode-pattern-arity-fix.md](m-bytecode-pattern-arity-fix.md) — route that doc through
sprint-planner/sprint-executor directly, it does not wait on this doc.
**This doc's remaining scope (A1, A2, B1-closure-half, B3, B4) is UNPARKED — D-25, Mark attended 2026-08-22: A2 = semantic effect extraction via a new CLI surface that emits a file's resolved effect row (feasibility spike first; compiler packages stay OUT of the harness; stderr sniffer retired).** Routable to the second design round with that question answered. Was: PARKED `needs-human-review` pending it.
Nothing should be planned from *this* doc's remaining scope until that round happens.
**Revision 1 (post-quorum)**: both round-0 objections (unsafe effect replay mis-filed as benign;
Net/Clock inventory arithmetic) confirmed first-party and fixed — see Verification Log.

> ### ⚠ PARKED — resolved: option C chosen (2026-08-04); doc split, see status line above
>
> The re-quorum **rejected** (at iteration 114), on two NEW objections, both aimed at **A2's
> file-classification mechanism** (not at Lane B, not at the soundness bugs). The loop parked
> rather than forcing a guardrail (Standing rule 2), since the narrow-refinement carve-out did
> **not** apply — see why below.
>
> **gemini-3-1-pro** (narrow, verbatim-fixable): A2 item 3 claims `vmStdout == evalStdout` proves
> "the fallback fired before any observable output". That is false — `FS` writes and `Net` calls
> commit observable effects **without printing**, so a non-printing unsafe replay classifies as a
> benign bucket and is masked. Its fix is a verbatim honesty note (the check detects
> *stdout*-visible replay only; true prevention is B4's VM-level policy).
>
> **gpt5-6-sol** (NOT narrow — this is what parked it): A2 swaps the drifting **filename** map for
> `detectCaps`, which **string-sniffs source text** for `! {Net` / `import std/clock`. That is a
> second non-semantic heuristic — it can silently execute a live or clock-dependent program and
> misreport its parity. It asks for **semantic** effect extraction from the compiler's resolved
> effect row, with a loud classification error when resolution fails.
>
> **Why this needs a human and not a controller edit.** The reviewer's fix is correct in principle,
> but `scripts/verify_bytecode_parity.go` **shells out to the `ailang` binary and imports no
> compiler package at all** (controller-verified). So "use the resolved effect row" is not a
> reword — it requires either pulling compiler packages into the harness or adding a CLI surface
> that emits a file's effect row, and then re-sizing A2. The reviewer itself defers that call
> ("if semantic extraction is genuinely unavailable, explicitly scope and design that extension").
> That is a design decision with a cost, i.e. exactly the controller-judgment case the carve-out
> excludes.
>
> **Options that were put to Mark (loop recommended C):**
> - **(A)** Do it properly: A2.1 becomes semantic effect extraction via a new/reused compiler
>   surface. Correct, but grows A2 beyond its ~0.5d and needs a feasibility spike first.
> - **(B)** Keep the sniffer, document its limits loudly, revisit later. Cheapest — but both
>   reviewers say it repeats the exact sin (non-semantic classification) that produced the
>   `http_simple` drift this doc opens with.
> - **(C) CHOSEN 2026-08-04 — split the doc and unblock the P0 now.** The blocked question is
>   entirely about **A2's harness classification**. The **#505 pattern-arity soundness bug
>   (B1-quicksort-half + B2) does not depend on it at all**: its root cause is settled, its repro
>   is minimal, and its acceptance test (AC8/AC8b) is a table-driven pattern test, not a
>   parity-harness count. B1-quicksort-half + B2 are now spun out into
>   [m-bytecode-pattern-arity-fix.md](m-bytecode-pattern-arity-fix.md), ready to sprint
>   immediately. **A2 (and the A1/A2 harness lane, plus B1-closure-half/B3/B4 which depend on A2's
>   new harness buckets) still need a second design round** with the semantic-extraction question
>   answered before anything in *this* doc can be planned.
>
> Quorum artifacts: `.ailang/state/mission-quorum/m-bytecode-vm-parity-bugs-*.json`
> (round 0 and round 1). Related filed bugs: **#505** (pattern arity — now tracked in the spun-out
> doc), **#506** (unsafe replay — stays here, part of B4, blocked on A2).
**Target**: v1.0.0 (clause-2 soundness residue on the V1 mission queue)
**Priority**: **P0 ×2** — (1) `recursion_quicksort.ail` is a **silent wrong result** under
`--bytecode` (no error, no fallback, wrong list; root cause #505). (2) The VM→evaluator fallback
**re-executes programs from the beginning after observable effects have already been committed**
(unsafe replay — here a duplicated `println`, but the same mechanism would duplicate an FS write,
an HTTP POST, or a Msg send). Both are direct NO-SILENT-FALLBACKS violations and A1 (determinism)
violations. The parity harness has been surfacing **genuine soundness defects as cosmetic "stdout
differs" rows** — "output divergences" was an understatement; this doc's framing is corrected
accordingly.
**Estimated**: ~4–5 days total (Lane A ~1d; Lane B ~3–4d, investigation-first). **This exceeds
the 3–4d sprint box — see "Sprint sizing & proposed split" below; do not silently overrun.**
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

**Revision-1 finding — the MATCH 149 headline is inflated by 6 fake parity rows.** A first-party
scan of the corpus without `--quiet` (which suppresses the fallback warning — see Verification
Log) found **8 files that silently fall back to the evaluator with exit 0**: the two DIVERGE rows
below (#2, #4) **plus 6 files currently counted MATCH** (`array_grid.ail`, `html_parser.ail`,
`map_basic.ail`, `module_let_helpers.ail`, `std_deflate_pdf_objstm.ail`, `stdlib_gzip.ail`).
For those 6 the VM emitted no output before falling back, so the evaluator's re-run byte-matches
the evaluator — **"MATCH" is the evaluator agreeing with itself; the VM never ran the program**.
Two of them (`array_grid`: `arith MUL on Closure`; `module_let_helpers`: `arith ADD: type
mismatch Int vs Closure`) are the **same closure-dispatch bug family as `array_basic`** (Lane B3),
currently laundered as MATCH. This also explains why the VM_BRIDGE/VM_RUNTIME buckets are all 0
at HEAD: the non-strict fallback structurally converts VM failures into exit-0 evaluator runs,
so those buckets are near-unreachable. The gate has been measuring the evaluator, not the VM,
for every fallback file.

### The 7 DIVERGE files (eval = ground truth, except where noted)

| # | File | Symptom at HEAD | Class | Lane |
|---|------|-----------------|-------|------|
| 1 | `recursion_quicksort.ail` | VM **silently** returns `[3]` for BOTH `Quicksort:` and `sortBy:`; eval returns `[1, 1, 2, 3, 4, 5, 6, 9]`. Exit 0, **no error, no fallback** (stderr verified clean) | **VM correctness — SOUNDNESS, silent wrong result** | **B (P0)** |
| 2 | `array_basic.ail` | VM prints `Length: <Closure>`, `numbers[0] (first): <Closure>`, … then `vm: GET_TAG on Closure (in array_basic.showOpt at array_basic.ail:32)` → falls back, evaluator **re-runs the program from the beginning**, re-performing the already-committed printlns; stdout = wrong prefix + full re-run output | **TWO defects in one file**: VM dispatch bug (array length/index produced as unforced closures, → B3) **and** an unsafe-replay instance (committed IO re-executed by the fallback, → B4) | **B (B3 + B4)** |
| 3 | `pattern_sugar.ail` | **eval** prints `firstPair(...) = <*eval.TupleValue>`; **VM is CORRECT** (`(a, 1)`). Single-line diff, verified | **eval-side show bug** — harness ground truth is the wrong side | **A** |
| 4 | `tar_gzip_reader.ail` | VM executes the first `println` (committed IO), then hits bridge limit (`TaggedValue (Result.Err) not yet supported (M-BYTECODE-2E scope)` at `ail:19, ip 9`) → falls back → evaluator **re-runs from the beginning, re-performing the committed effect** → header printed **twice** (eval prints it once; re-verified first-party this revision) | **UNSAFE EFFECT REPLAY — soundness bug in the fallback mechanism itself** (→ B4). Here the duplicated effect is a `println`; the same mechanism would duplicate an FS write, an HTTP POST, or a Msg send. NOT benign, NOT a harness-visibility cosmetic. A2 must classify it `VM_UNSAFE_REPLAY` (loud, red) — never a passing/expected bucket | **B (B4)**; A2 makes it visible |
| 5 | `http_simple.ail` | `! {IO, Net}`, live httpGet — nondeterministic upstream body | **Harness false-positive** (NEW row; filename-list drift) | **A** |
| 6 | `claude_haiku_call.ail` | `! {Net, IO}`, live API call | **Harness false-positive** | **A** |
| 7 | `xml_walk_perf.ail` | `! {IO, Clock}`, prints `time_ms=42` vs `43` (timing jitter) | **Harness false-positive** | **A** |

Load-bearing framing: this doc covers **three distinct soundness bugs**, not "output divergences":
1. **#505 pattern-arity** (#1): fixed-length list patterns match longer lists — silent wrong
   result, no error, no fallback. P0.
2. **Closure-dispatch family** (#2 + `array_grid`, `module_let_helpers`): array/`let`-helper
   values reach arithmetic/dispatch as unforced closures — VM errors out mid-run or pre-run.
3. **Unsafe effect replay** (#4, and #2's fallback leg): the VM→evaluator fallback **restarts
   the whole program after observable effects have been committed**, re-performing them. The
   duplicated effect here is a `println`; the identical mechanism would duplicate an FS write, an
   HTTP POST, or a Msg send. This is a determinism/soundness bug in the fallback mechanism
   itself, P0 alongside #505 — not a harness-visibility cosmetic. (Earlier revisions of this doc
   called the fallback "correct"/benign and routed #4 to Lane A as a VM_BRIDGE re-categorization.
   That was wrong — quorum objection by gemini-3-1-pro, confirmed by controller, re-verified
   first-party here.)

The quicksort root cause is #505 (controller box below); `array_basic`'s code-level cause is
**not known** and Lane B stays investigation-first (Milestone B1). Do not anchor on the
2026-04-08 hypotheses in the appendix; the symptom has changed since (`<List>` → `[3]`).

## Verification Log (first-party, this worktree, 2026-07-28)

All behavior claims re-measured here with the worktree `./bin/ailang` + the parity harness
(`--only` runs); controller HEAD totals inherited as labeled above.

| Claim | How verified |
|-------|--------------|
| quicksort: VM `[3]`/`[3]`, eval correct, exit 0, **stderr has NO fallback warning** | direct `./bin/ailang run --bytecode` with stderr captured separately |
| array_basic: `<Closure>` prefix + `GET_TAG on Closure` + mid-run fallback re-print | harness `--only array_basic` (stdout shows wrong prefix + full correct re-run) |
| pattern_sugar: exactly ONE diff line; eval `<*eval.TupleValue>`, VM `(a, 1)` | unified diff of harness `--only pattern_sugar` outputs |
| tar_gzip: header printed **exactly twice** under `--bytecode` (fallback at `ail:19, ip 9` fires AFTER the first `println` committed), **exactly once** under eval — i.e. a committed IO effect was re-performed by the fallback re-run. **Unsafe replay, not a benign duplicate** | direct `./bin/ailang run --bytecode` vs `run`, stderr captured; grep-counted the header line (2 vs 1) |
| **`--quiet` suppresses the fallback warning entirely** — emission is guarded by `if !params.quiet` (`cmd/ailang/run_helpers.go`, the block at ~:376) and the harness always passes `--quiet` (`scripts/verify_bytecode_parity.go:235`). Consequence: the stderr-sniff A2 as previously written **could never fire**; A2's VM leg must drop `--quiet` | read of both sites + direct A/B run (with `--quiet`: no marker on stderr; without: marker present) |
| **8 files fall back with exit 0** at HEAD; 6 of them are currently counted MATCH (`array_grid`, `html_parser`, `map_basic`, `module_let_helpers`, `std_deflate_pdf_objstm`, `stdlib_gzip` — all verified eval-exit-0, hence MATCH not EVAL_SKIP). `array_grid`/`module_let_helpers` fail with closure-in-arithmetic errors (B3 family); the other 4 with bridge-scope errors. **Caveat: lower bound** — the scan ran each file directly with broad caps and no `--entry` detection, so entrypoint-less examples were not exercised; A2's first full run is authoritative | first-party corpus scan without `--quiet`, grepping stderr for `falling back to evaluator` on exit-0 runs |
| `--strict-bytecode` exists: in strict mode the eval bridge is intentionally NOT wired and any EvalOnly call is a hard VM error (no fallback, no replay) — existing evidence for B4's policy option (a) | read of `cmd/ailang/run_helpers.go` (strict branch above the fallback block) |
| **Net/Clock-declaring runnable examples = 9 total** (reconciled; table below) | controller machine-generated inventory, re-verified first-party by grepping every effect row of all 11 candidate files |
| Effect rows of the 2 EVAL_SKIP `ai_call*` files: `ai_call_json_simple_result.ail` and `ai_call_result.ail` declare `! {AI, IO}` — **no Net, no Clock**; they are EVAL_SKIP for an unrelated reason (evaluator exit 1) and are NOT in the effect inventory | grep of both files |
| eval show gap site: `internal/builtins/show.go` `showValue` has **no `*eval.TupleValue` case**; default at line 173 prints `<%T>` | read of the full switch (lines 27–174) |
| **No test/golden depends on the broken `<*eval.TupleValue>` output** | repo-wide grep — only design docs/mission log mention it |
| Harness exclusion = filename map; **no effect-driven exclusion exists** | read `scripts/verify_bytecode_parity.go:58-61,166` |
| Harness **never inspects vmStderr when vmExit==0** → exit-0 fallbacks invisible | read `verifyOne`, lines 190–218 |
| Fallback warning emitted to **stderr** at `cmd/ailang/run_helpers.go:376`, text embeds the original vmErr | read of the emission site |
| `cmd/ailang/run_bytecode_test.go:63` asserts **absence** of `"falling back to evaluator"` in non-fallback runs (pins the warning text) | read of the test |
| Regression fixtures cited below exist and currently MATCH | `--only cons_expression`, `--only block_recursion` both MATCH |
| `tests/golden/bytecode/` exists (golden_test.go etc.) | ls |

### Net/Clock effect inventory (ground truth — replaces the miscounted revision-0 row)

The revision-0 log row claimed "9 total" but enumerated 3 DIVERGE + 4 MATCH + 4 EVAL_SKIP = 11
(quorum objection by gpt5-6-sol, confirmed). **Provenance of the error, so the correction is
auditable**: the "3 ai_call\*" EVAL_SKIP entries were counted by FILENAME PREFIX.
`ai_call_json_simple_result.ail` and `ai_call_result.ail` are indeed EVAL_SKIP, but declare
`! {AI, IO}` — no Net/Clock — so they do not belong in this inventory. That is precisely the
filename-vs-effect conflation the A2 effect-driven rule exists to eliminate — committed here as
an instance of the failure mode being fixed. The reconciled inventory (controller-generated,
re-verified first-party per file):

| file (examples/runnable/) | declared effects (Net/Clock row) | category at HEAD | post-A2 category |
|---|---|---|---|
| `http_simple.ail` | `! {IO, Net}` | DIVERGE | NON_DET (effect-derived) |
| `claude_haiku_call.ail` | `! {Net, IO}` | DIVERGE | NON_DET (effect-derived) |
| `xml_walk_perf.ail` | `! {IO, Clock}` | DIVERGE | NON_DET (effect-derived) |
| `ai_call.ail` | `! {Net}` (on `chatOpenAI`) | MATCH | NON_DET (effect-derived) |
| `demo_ai_api.ail` | `! {IO, Net}` | MATCH | NON_DET (effect-derived) |
| `http_put_bytes.ail` | `! {IO, Net}` | MATCH | NON_DET (effect-derived) |
| `stdlib_game.ail` | `! {IO, Clock}` | MATCH | NON_DET (effect-derived) |
| `ai_call_stream.ail` | `! {AI, Stream, Net, IO}` | EVAL_SKIP | NON_DET (effect-derived) |
| `ai_stream_openai.ail` | `! {AI, Stream, Net, IO}` | EVAL_SKIP | NON_DET (effect-derived) |

3 DIVERGE + 4 MATCH + 2 EVAL_SKIP = **9**. ✓ (Post-A2 column assumes the classification
precedence settled in the Conflict Surface: effect-derived NON_DET is decided from source
**before either backend runs**, so it outranks EVAL_SKIP.)

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
   runs (also removes live network calls from the parity gate). **This settles the precedence
   question**: effect-derived NON_DET **outranks EVAL_SKIP**, because it is decided statically
   from source with no execution at all — running a live-Net example just to learn the evaluator
   fails is exactly what the rule exists to avoid. The 9 inventory files (table above) all
   become NON_DET, including the 2 formerly-EVAL_SKIP stream files. The filename map stays only
   for the 2 legacy entries (`uuid.ail`, `stream_process_source.ail`) whose non-determinism is
   not effect-visible in this rule. **Measured coverage cost (intentional, re-verified against
   the inventory)**: 4 currently-MATCHing files (`ai_call.ail`, `demo_ai_api.ail`,
   `http_put_bytes.ail`, `stdlib_game.ail`) move to NON_DET. All excluded files must be
   **reported in the NON_DET bucket by name, never silently dropped** — and nothing outside the
   11 (2 legacy + 9 effect) may land there (the `! {AI, IO}` files must NOT be swept in).
2. **The VM leg must NOT pass `--quiet`** (eval leg unchanged). Verified this revision: the
   fallback warning is emitted only `if !params.quiet` (`cmd/ailang/run_helpers.go`), and the
   harness passes `--quiet` unconditionally (line 235) — so a stderr sniff as revision 0
   specified it would never fire. Stderr is sniffed, never diffed, so the extra status lines are
   harmless.
3. **Exit-0 fallback detection**: when `vmExit == 0`, scan the VM leg's stderr for the fallback
   marker (`"falling back to evaluator"`, with the original vmErr embedded). If present:
   - **`vmStdout != evalStdout` → `VM_UNSAFE_REPLAY`** (new status, loud/red): the VM emitted
     output before falling back, so the evaluator re-run **re-performed committed observable
     effects** — the divergent prefix is the direct evidence of the replay. Reason embeds the
     original vmErr so triage can still distinguish bridge-scope from runtime errors. Catches
     `tar_gzip_reader.ail` AND `array_basic.ail`.
   - **`vmStdout == evalStdout`** → the fallback fired before any observable output; classify by
     the embedded error with the existing markers: bridge-scope (`not yet supported` /
     `M-BYTECODE-2E` / `TaggedValue` / `unsupported eval value type`) → `VM_BRIDGE`; **anything
     else (closure-in-arith etc.) → `VM_RUNTIME`**. **Never MATCH** — post-A2, MATCH means "the
     VM itself ran the program to completion and byte-agreed", which is the parity claim the
     gate exists to make. This exposes the 6 fake-MATCH rows (4 → VM_BRIDGE, 2 → VM_RUNTIME).
   - **Do NOT change the warning text** — `run_bytecode_test.go:63` asserts its absence in
     non-fallback runs; the harness sniffs the same string (single source of truth; if it must
     ever change, change both, and note B4 may legitimately change *when* it is emitted).

**Predicted post-Lane-A state** (A1+A2 landed, no Lane-B fix; expectation, deliberately NOT an
acceptance criterion — AC11 is the sprint-exit gate). Arithmetic from HEAD 149/2/7/16 (Σ=174):

| bucket | count | derivation |
|---|---|---|
| MATCH | **140** | 149 − 4 (effect rule) − 6 (fake-MATCH exposed) + 1 (pattern_sugar via A1) |
| NON_DET | **11** | 2 legacy + 9 effect-derived (3 ex-DIVERGE, 4 ex-MATCH, 2 ex-EVAL_SKIP) |
| EVAL_SKIP | **14** | 16 − 2 (moved to NON_DET by precedence) |
| VM_BRIDGE | **4** | html_parser, std_deflate_pdf_objstm, map_basic, stdlib_gzip (ex-MATCH) |
| VM_RUNTIME | **2** | array_grid, module_let_helpers (ex-MATCH; closure family) |
| VM_UNSAFE_REPLAY | **2** | tar_gzip_reader, array_basic (ex-DIVERGE) |
| DIVERGE | **1** | recursion_quicksort (the #505 P0) |

Σ = 140+11+14+4+2+2+1 = **174** ✓. Caveat (honest lower bound): the fallback scan could not
exercise entrypoint-less examples; if A2's first full run finds more exit-0 fallbacks, they move
MATCH → VM_BRIDGE/VM_RUNTIME/VM_UNSAFE_REPLAY and must be reconciled **by name** in the
implementation report — never absorbed silently.

### Lane B — VM correctness (investigation-FIRST, ~3–4 days)

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
  bare `length(arr)` print. The closure family now has **three known members** — `array_basic`
  (`GET_TAG on Closure`), `array_grid` (`arith MUL on Closure`), `module_let_helpers`
  (`arith ADD: type mismatch Int vs Closure`) — B1 must say whether they share one root cause
  (they plausibly do: values reaching dispatch/arith as unforced closures) or not.
- `ailang disasm` the repro; compare the bytecode against the expected lowering; trace VM
  execution (`DEBUG_STRICT=1`, existing VM tracing) to the first wrong value.
- Cross-check the eval path on the same Core IR to localize eval-vs-lower-vs-VM.
- **Deliverable: a written root-cause statement naming the defective function/opcode path**,
  plus the minimal repro committed under `tests/golden/bytecode/`. Prior hypotheses (appendix,
  iter-102) are stale — verify, don't inherit.

**B2 — fix the quicksort-class bug** (P0). Fix at the layer B1 named
(`internal/gen/lower/` / `internal/vm/` / `internal/bytecode/`); golden regression test
asserting the **exact sorted output** under `--bytecode`.

**B3 — fix the closure-dispatch family.** Array length/index and `let`-helper results must be
forced values, not closures; covers whatever subset of {`array_basic`, `array_grid`,
`module_let_helpers`} B1 attributes to the named cause (if B1 finds distinct causes, B3 fixes
the array_basic one and the others are explicitly parked with their own root-cause statements —
not silently dropped). Regression test asserting exact output under `--bytecode` **with no
fallback** (assert stderr does not contain `"falling back to evaluator"`).

**B4 — the fallback mechanism itself: unsafe effect replay (investigation-FIRST, policy
decision).** The VM must not silently restart the evaluator once observable effects have been
committed (Verification Log: tar_gzip's header prints twice; Critical Principle 2 and axiom A1
both violated). **The fix is NOT designed here** — the policy choice is a genuine design
question the milestone's investigation must settle with evidence:

- **Option (a) — abort loudly**: once the VM cannot continue, fail with a hard error instead of
  re-running. Evidence it is feasible: `--strict-bytecode` already implements exactly these
  semantics today (bridge unwired, EvalOnly call = hard VM error, no fallback, no replay —
  verified first-party). Trade-off: loses graceful degradation for the (measured) majority of
  fallback cases — 6 of the 8 known fallbacks fire before any output and currently degrade
  harmlessly.
- **Option (b) — restrict fallback to provably pre-effect**: allow the evaluator restart only if
  no observable effect has been committed yet (needs an effect-commit marker at the capability
  boundary — what counts as "observable" is part of the investigation); abort loudly otherwise.
  Trade-off: keeps degradation for the pre-effect majority, adds VM/runtime state and a new
  invariant to test.

Either option makes the replay impossible; the investigation picks one with evidence (frequency
data from the A2-instrumented harness, implementation cost at the `run_helpers.go` fallback
site) and records the decision. Same discipline as B1: investigation first, no speculative
implementation before the written policy decision.

## Milestones & Acceptance Criteria

Each AC belongs to exactly ONE milestone and names the file that can fail it. Milestones A and B
are separable: a Lane-B overrun cannot strand Lane A (A1+A2 land and are gate-effective alone).

### Milestone A1 — eval tuple show (~2h)
- **AC1**: `examples/runnable/pattern_sugar.ail` is MATCH in the parity harness (fails today:
  eval side prints `<*eval.TupleValue>` at the `firstPair` line).
- **AC2**: unit test in `internal/builtins/` asserting `showValue` on a
  `*eval.TupleValue{("a",1)}` returns `(a, 1)` — fails on HEAD (case absent, default `<%T>`).

### Milestone A2 — harness honesty (~0.5d)
- **AC3**: `examples/runnable/tar_gzip_reader.ail` reports **`VM_UNSAFE_REPLAY`** — not DIVERGE,
  not VM_BRIDGE, not MATCH, not any passing/expected bucket — with the original vmErr embedded
  in the reason. Fails today (reports DIVERGE; the replay is invisible).
- **AC3b** (fake-MATCH exposure): `html_parser.ail`, `std_deflate_pdf_objstm.ail`,
  `map_basic.ail`, `stdlib_gzip.ail` report **VM_BRIDGE** (not MATCH) — fails today (all four
  are counted MATCH while the VM never ran them).
- **AC4**: the NON_DET bucket contains **exactly** the 11 named files — `uuid.ail`,
  `stream_process_source.ail` (legacy filename reasons) plus the 9 inventory files (effect-derived
  reasons) — listed by name in the text/markdown/JSON reports. Fails in either direction:
  silent-drop of a member, or sweeping in a non-member (e.g. the `! {AI, IO}` files
  `ai_call_json_simple_result.ail` / `ai_call_result.ail`).
- **AC5** (anti-vacuous guard): after A2 (with A1 landed, B not landed),
  `recursion_quicksort.ail` still reports **DIVERGE**, `array_basic.ail` reports
  **VM_UNSAFE_REPLAY** (with `GET_TAG on Closure` embedded), and `array_grid.ail` +
  `module_let_helpers.ail` report **VM_RUNTIME** (not VM_BRIDGE — their errors are not bridge
  scope) — i.e. re-categorization did NOT absorb any Lane-B bug into a green or expected bucket.
  If any of these goes green without a Lane-B fix commit, A2 is wrong.

### Milestone B1 — diagnose the VM bug families (#505 + closure-dispatch), NO fix (~1d)

> **The quicksort/#505 half of B1, and all of B2, are SPUN OUT** into
> [m-bytecode-pattern-arity-fix.md](m-bytecode-pattern-arity-fix.md) (2026-08-04, Mark's option-C
> decision) — that doc is unblocked and ready to sprint now. **Only the closure-dispatch-family
> half of B1 (AC7) remains here**, still parked pending the A2 second design round (B3's fix and
> AC11's whole-doc gate both need A2's harness buckets).

- ~~**AC6**: minimal failing repro for the quicksort `[3]` collapse...~~ — moved to
  m-bytecode-pattern-arity-fix.md as AC1–AC3.
- **AC7**: same deliverable for the closure-dispatch family (`array_basic`'s `GET_TAG on
  Closure`, plus an explicit shared-or-distinct verdict covering `array_grid` and
  `module_let_helpers`; may name the same root cause as the pattern-arity bug — must say so
  explicitly if so). **Remains parked here.**

### Milestone B2 — quicksort-class fix (P0, ~0.5–1d) — SPUN OUT, see m-bytecode-pattern-arity-fix.md

> Former AC8/AC8b/AC9 now live as AC1/AC2/AC3 in
> [m-bytecode-pattern-arity-fix.md](m-bytecode-pattern-arity-fix.md), unblocked and ready to sprint.
> Content kept below for historical reference only — **do not plan from here**, plan from the
> spun-out doc.

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

### Milestone B3 — closure-dispatch family fix (~0.5–1d)
- **AC10**: `examples/runnable/array_basic.ail` under `--bytecode` prints `Length: 5` /
  `numbers[0] (first): 10` (exact eval output) with no fallback-warning line on stderr —
  fails on HEAD (`<Closure>` + fallback). If B1 attributed `array_grid.ail` and
  `module_let_helpers.ail` to the same cause, both must also reach MATCH with no fallback; if
  B1 found distinct causes, their park must be recorded per B3's text (AC11 then cannot claim
  VM_RUNTIME 0 — see AC11's escape clause).

### Milestone B4 — unsafe-replay fallback policy (investigation-first, ~1d)
- **AC12**: `examples/runnable/tar_gzip_reader.ail` under
  `--bytecode --caps IO,FS,Clock,Net` prints the line
  `== readFromGzip(paper.tar.gz, main.tex) ==` **exactly once** (count == 1; zero also fails) —
  **fails on HEAD, where it prints twice** (re-verified first-party this revision). Committed as
  a regression test (`tests/golden/bytecode/` or `cmd/ailang/run_bytecode_test.go`).
- **AC13**: a **written policy decision** — option (a) abort-loudly vs option (b)
  pre-effect-only fallback — with the investigation's evidence and the named enforcement site
  (the `run_helpers.go` fallback block), recorded in sprint notes + this doc's implementation
  report; and the full parity harness reports **VM_UNSAFE_REPLAY 0**. Anti-laundering (AC5's
  logic extended, per the quorum): AC13 is INVALID if achieved by weakening or removing the
  harness's replay detection, or by rewording the fallback warning so the sniffer stops firing —
  the zero must come from the fallback mechanism no longer replaying, with `tar_gzip_reader.ail`
  classified loudly (predicted VM_BRIDGE via a non-zero exit, under either policy branch).

### Sprint-exit gate (whole doc)
- **AC11**: full parity harness at sprint end reports, against the 174-file corpus at HEAD
  `33be8f5a7`:
  **DIVERGE 0, VM_UNSAFE_REPLAY 0, VM_COMPILE 0, VM_RUNTIME 0**; **NON_DET exactly the 11 files
  named in AC4**; **EVAL_SKIP 14**; and **MATCH + VM_BRIDGE = 149**, predicted split
  **MATCH 144 / VM_BRIDGE 5** (144 = 140 post-Lane-A + quicksort + array_basic + array_grid +
  module_let_helpers; 5 = the 4 pre-effect bridge fallbacks + tar_gzip aborting loudly
  post-B4). The MATCH/VM_BRIDGE split is scan-derived (lower bound — see A2 caveat): if A2's
  full run surfaces additional exit-0 fallbacks, the reconciliation must be shown by name and
  the split re-derived in the implementation report; the zero-buckets and the NON_DET/EVAL_SKIP
  counts are exact and unconditional. Escape clause: VM_RUNTIME 0 may be relaxed ONLY by B3's
  explicit distinct-cause park, recorded with its own root-cause statement — never by
  re-bucketing. No currently-genuinely-MATCHing file regresses (spot fixtures below). Valid
  ONLY in conjunction with AC8/AC10/AC12 — counts alone are fakeable by re-categorization,
  which AC5/AC13 forbid.

## Sprint sizing & proposed split

Adding B4 (and B3's family growth) pushes the total to **~4–5 days** (A1 ~2h, A2 ~0.5d, B1 ~1d,
B2 ~0.5–1d, B3 ~0.5–1d, B4 ~1d). **This no longer fits the 3–4 day sprint box.** Rather than
silently overrun, the proposal is a split at the already-designed lane seam:

- **Sprint 1 (~2.5–3d)**: A1 + A2 + B1 + B2 — gate honesty plus the silent-wrong-result P0
  (#505). After sprint 1, the unsafe replay and the closure family are **loudly visible in every
  harness report** (`VM_UNSAFE_REPLAY 2`, `VM_RUNTIME 2`) — deferral is not hiding; AC5 pins
  exactly this intermediate state.
- **Sprint 2 (~1.5–2d)**: B3 + B4 — closure-dispatch family fix + replay policy. AC11 is the
  whole-doc exit gate and closes only after sprint 2.

Alternative: run it as one 4–5 day sprint if Mark prefers; the split is the recommendation
because both halves land independently useful, gate-effective states.

## Conflict Surface (mandatory — codegen)

**Positions/files this touches, and what else lives there:**

1. `internal/builtins/show.go` (`showValue`) — the show surface for **every eval-path program**.
   Adding a `*eval.TupleValue` case changes output for any program that shows a tuple. Verified:
   no test/golden anywhere asserts the current `<*eval.TupleValue>` text. The new rendering must
   match the VM's (`(a, 1)`, elements via recursive `showValue`, strings unquoted per this
   file's existing `StringValue` case) or A1 trades one divergence for another.
2. `scripts/verify_bytecode_parity.go` — classification precedence is now SETTLED (the counts
   in A2/AC11 depend on it): **(1) filename NON_DET (2 legacy) → (2) effect-derived NON_DET
   (static, before any execution — outranks EVAL_SKIP) → (3) eval run, EVAL_SKIP on exit≠0 →
   (4) VM run, exit≠0 classes (VM_COMPILE/VM_BRIDGE/VM_RUNTIME by error markers) → (5) exit-0
   fallback sniff (VM_UNSAFE_REPLAY if stdout differs; else VM_BRIDGE/VM_RUNTIME by embedded
   error) → (6) stdout compare, MATCH/DIVERGE.** The sniff MUST discriminate on the embedded
   error, else genuine VM bugs (array_grid, module_let_helpers) get laundered as VM_BRIDGE
   (AC5 guards this). The VM leg drops `--quiet` (A2 item 2); the eval leg keeps it.
3. `cmd/ailang/run_helpers.go` fallback block (~:376) — now DOUBLY load-bearing: the harness
   sniffs the warning text (A2), and B4's policy change lands at this exact site (both policy
   options alter when/whether the evaluator restart happens; the strict-mode branch just above
   it is option (a)'s existing implementation evidence). The text is pinned by
   `cmd/ailang/run_bytecode_test.go:63` (asserts absence in non-fallback runs). **Constraint:
   don't reword it**; if B4 changes when it is emitted, update harness + test in the same
   commit.
4. `internal/vm/` / `internal/gen/lower/` / `internal/bytecode/` (Lane B) — exact files unknown
   until B1 names the root cause; whatever changes, the whole-corpus harness is the blast-radius
   detector. List/closure codegen is shared by *all* list-recursive programs, not just the
   currently-red files.
5. **Intentional incompatibilities**: (a) eval tuple-show output changes (broken → correct);
   (b) 4 currently-MATCHing Net/Clock examples (`ai_call.ail`, `demo_ai_api.ail`,
   `http_put_bytes.ail`, `stdlib_game.ail`) move MATCH → NON_DET by the effect rule — a
   deliberate, reported coverage trade documented in A2; (c) headline MATCH drops 149 → ~140
   post-Lane-A (honest: −4 effect rule, −6 fake-MATCH exposure, +1 pattern_sugar), recovering
   to ~144 post-Lane-B; (d) `tar_gzip_reader.ail` under `--bytecode` **changes behavior by B4's
   policy**: today it "completes" via unsafe replay (exit 0, header twice); post-B4 it aborts
   loudly (either policy branch) until the M-BYTECODE-2E bridge gap it hits is closed —
   an intentional unsound→sound trade.

**Programs that MUST still work post-change (verified to exist; first two verified MATCH at
HEAD):** `examples/runnable/cons_expression.ail`, `examples/runnable/block_recursion.ail`,
`examples/runnable/adt_list_fields.ail`, `examples/runnable/effectful_list.ail`, plus the
non-tuple lines of `pattern_sugar.ail`. And `cmd/ailang/run_bytecode_test.go` must stay green.

## Testing Strategy

- Unit: tuple case in `internal/builtins` show tests (AC2); harness classification is exercised
  via the live corpus (the script is `go:build ignore` — no unit harness; ACs 3–5 are its test).
- Golden: minimal repros from B1 + exact-output goldens (AC8–AC10) under `tests/golden/bytecode/`;
  the AC12 exactly-once effect-replay regression test (fails on HEAD: header ×2).
- Whole-corpus: full parity harness before/after every Lane-B change (AC11 + fixture spot-check).

## Non-Goals

- ADT constructor name rendering (M3 of M-BYTECODE-MULTIMODULE scope).
- Rewriting show dispatch or type-class dictionary lowering (take the targeted fix B1 justifies).
- The EVAL_SKIP set (evaluator itself fails: missing AI keys / intentional exit codes).
- **Closing the M-BYTECODE-2E bridge gaps themselves** (the ~5 predicted VM_BRIDGE rows:
  Result.Ok/Err TaggedValue, MapValue, BytesValue). Post-sprint they are the honest, visible
  backlog — loud non-zero exits, no replay — for a future bridge milestone.
- Any Lane-B fix not named by B1 (bugs) or B4's written policy decision (fallback) — no
  speculative VM patches.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| B1 finds the quicksort cause in a shared lowering path with wide blast radius | High | Whole-corpus harness after every change; fixtures pinned above; B2 lands behind AC8's exact-output golden |
| Closure family (3 files) turns out to be 2–3 distinct bugs, blowing B3's budget | Medium | B1 delivers the shared-or-distinct verdict BEFORE B3 starts; distinct causes → explicit park with root-cause statements (AC10/AC11 escape clause), never silent re-bucketing |
| B4's policy needs VM effect-commit tracking (option b) and grows beyond ~1d | Medium | Option (a) is pre-implemented as `--strict-bytecode` semantics — a hard fallback position that satisfies AC12/AC13 at bounded cost; investigation decides with that floor in hand |
| More exit-0 fallbacks exist among entrypoint-less files (scan lower bound) | Medium | A2's full run is authoritative; AC11 requires by-name reconciliation, forbids silent absorption |
| Effect-driven exclusion hides a future genuine VM bug in a Net/Clock example | Medium | NON_DET bucket always lists members by name (AC4); coverage trade documented here |
| Fallback-marker string drifts from harness sniffer | Low | Single source of truth constraint in Conflict Surface #3; both sites named |
| Lane B overruns even the revised budget | Medium | Split proposal above; lanes separable by design; A1+A2 land independently; B1's named root causes make any handoff/park cheap and concrete |

## Related Documents

- [design_docs/implemented/v0_11_0/m-bytecode-vm.md](../../implemented/v0_11_0/m-bytecode-vm.md) — M-BYTECODE-VM master design
- [design_docs/implemented/v0_11_0/m-bytecode-2d-parity.md](../../implemented/v0_11_0/m-bytecode-2d-parity.md) — parity harness origin; documents the fallback warning
- [design_docs/implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md](../../implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md) — parent sprint (shipped)
- [design_docs/planned/v1_1_0/m-perf4-bytecode-interpreter.md](../v1_1_0/m-perf4-bytecode-interpreter.md)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Silent cross-backend divergence (quicksort) eliminated; **unsafe re-execution of committed effects made impossible (B4)**; backends byte-agree |
| A7: Machines First | +1 | Parity gate becomes a trustworthy machine signal (no filename-drift false rows, no fake MATCH via fallback) |
| A11: Structured Failure | +1 | The *silent* wrong-result path and the *silent* effect-replay path both become impossible to ship past the gate; exit-0 fallbacks become visible, categorized statuses |
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
**Last updated**: 2026-07-28 (revision 1 post-quorum: unsafe-replay soundness bug promoted to P0 + Milestone B4; Net/Clock inventory reconciled to 9 with exact post-A2 totals; fake-MATCH fallback rows exposed; `--quiet`-suppression defect in A2's original sniff design fixed)
