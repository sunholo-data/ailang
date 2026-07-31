# M-FMT-DIALECT-ALIGNMENT

**Status:** planned
**Created:** 2026-07-31
**Deadline:** before Wednesday 2026-08-05 03:00 (the scheduled fmt A/B)
**Owner:** next session — this doc exists so nothing below has to be re-derived

---

## The finding

`ailang fmt` canonicalizes AILANG **into a dialect `ailang prompt` never teaches**. Every
eval model is handed the prompt as its system text, then told by the formatter — on every
single write — that its correct code is non-canonical. It obeys, because the formatter
speaks last.

From a model's own context (`session_graph_bfs_6eb48f03.jsonl`, verbatim):

```
Wrote 1075 bytes to benchmark/solution.ail
  fmt: your file is UNCHANGED on disk. Canonical AILANG would differ here:
+pure func contains(x: int, xs: list[int]) -> bool = match xs {
+  ::(h, t) => if x == h then true else contains(x, t)
  (replacing:)
-pure func contains(x: int, xs: [int]) -> bool =
-    h :: t => if x == h then true else contains(x, t)

  ailang check: OK - file type-checks.
```

`check` confirms the code is fine; `fmt` calls it non-canonical in the same message. The
model then wrote, in its own words:

> *"The file type-checks but the formatter wants canonical style. Let me apply the formatting."*
> *"I need to rewrite using the canonical AILANG formatting style that the formatter expects: `list[int]`, `::(h, t)`, etc."*

It switched dialect and broke working code. Measured cost, arms differing only by the fmt
extension: **+62% output tokens, median +2,693, 18/23 pairs, Wilcoxon p=0.0112.**

**No type-check can catch this.** Both spellings parse and mean the same thing;
`Parse(fmt(x)) ≡ Parse(x)` holds throughout. Only an assertion about *which of several valid
spellings we teach* can catch it.

### Why it took two weeks to find

The banked `agent_transcript` records tool **calls**, not tool **results**. The fmt message
was never in eval data. Three sessions of analysis blamed the model for "inventing" syntax
it had been instructed to use.

---

## Divergence inventory

Counted against the **active** prompt versions only (`active` in `prompts/versions.json` =
v0.16.3, `prompts/agent/versions.json` = v0.9.0). Walking all historical prompts reports
~457 and is meaningless — those are prompts for syntax that has since changed.

| # | Divergence | Status |
|---|---|---|
| 1 | `[int]` → `list[int]` | **FIXED** — `ca3d04cd8` |
| 2 | `h :: t` → `::(h, t)` (pattern position) | **FIXED** — `ca3d04cd8` |
| 3 | `"${x}"` → `concat_String(...)` chains | **OPEN — this doc's main task** |
| 4 | `_r0` internal row-variable name leaking into output | **OPEN — small** |

Divergence 3 dominates: ~25 of 30 real token-level differences in the 2026-07-31 audit.
String interpolation appears in nearly every prompt example.

---

## Work items

### W1 — Re-sugar string interpolation (blocking)

`"a ${x} b"` desugars in `internal/parser/parser_literals.go` to a left-associative
`concat_String` chain with each hole wrapped in `show()`. **Nothing in the AST records that
it was written as an interpolation**, so the formatter must reconstruct it — exactly as it
already reconstructs cons (`FuncCall{"::"}` → infix) in `internal/format/expr.go`.

- **WIP implementation exists:** `scratchpad/evidence/expr_interp_WIP.go` (a full copy of
  `expr.go` with `interpolationString`, `flattenConcatChain`, `escapeInterpText`, `subExpr`).
- **It works** on every constructed case: round-trips exactly, preserves semantics
  (`Alice is 30 years old` before and after), escapes `\n` byte-correctly, and leaves
  hand-written `concat_String(a, b)` alone (no `show()` wrapper → no match).
- **It regresses 7 corpus files' round-trip**, which is why it was NOT shipped:
  `polymorphic_adt.ail`, `serve_api_webhook.ail`, `effectful_list_t8_string_list.ail`,
  `wasm_friendly_patterns.ail`, +3 (full list in the `TestCorpusComment` failure output).
  `fmt.go` fails closed on these, so they become unformattable rather than corrupt — safe,
  but a regression.

**Reproduce the regression:**
```bash
cp scratchpad/evidence/expr_interp_WIP.go internal/format/expr.go
go test ./internal/format/ -run Corpus -count=1
```
Confirmed the failures are caused by the WIP, not pre-existing: disabling only the re-sugar
call (`if s, ok := ...; false && ok {`) makes the corpus pass.

**Known-good cases** (all verified passing): `"Ok(${intToStr(n)})"`, `"Err(${e})"`,
`"Parse '42': ${v}"`, `"${a}${b}"`, `"${a}\n${b}\n${c}"`, multi-hole, nested calls.
The failing shape has NOT been isolated — that is the first task.

### W2 — Stop the row-variable leak

`fmt` emits `pure func getEmail(u: { email: string | _r0 }) -> string`. `_r0` is an internal
row-type name. Reproduce: block #9 of `prompts/v0.16.3.md`.

### W3 — Bank tool results in the eval path

The eval harness builds its own `agent_transcript` (calls only) while
`internal/observatory/importer_motoko.go` already parses motoko session JSONL **including
`tool_result`** — but it only runs via the manual `ailang chains import-motoko`. Eval runs
already create chains (`eval_suite:eval-...` sources in `ailang chains list`); the importer
is simply never invoked.

**Fix:** call `ImportMotokoSession` from eval-suite when the executor is motoko. One call
removes the parallel system and the blind spot that hid this bug for two weeks.

### W4 — Install the fixed binary

`make quick-install` is blocked by an unrelated in-flight change
(`cmd/ailang/eval_benchmark.go:700`, `UpdateStageMetrics` signature + `costProvenance`).
Until it builds, the validity backstop (`ae12799d4`) is committed but not running, and
analyses need a manual `api_error` filter.

---

## Acceptance criteria

1. `go test ./internal/format/` green, including `TestCorpusComment` (no new round-trip
   regressions) and `TestFmtDoesNotDriftFromTeachingPrompt`.
2. `knownDrift` in `internal/format/dialect_drift_test.go` **lowered** from 9. The test
   fails if drift rises AND if it falls without the constant being updated — so the number
   is always the truth.
3. `ailang fmt` is a no-op (modulo layout) on the interpolation examples in the active
   prompt. Spot-check:
   ```bash
   ailang fmt <(printf 'module m\n\nexport func main() -> () ! {IO} {\n  let n = "A"\n  println("${n} x")\n}\n')
   ```
   must return `println("${n} x")`, not a `concat_String` chain.
4. `make install` succeeds and `ailang --version` matches HEAD.
5. Re-run the A/B and report paired Wilcoxon on tokens. **This is the first run that
   measures fmt rather than measuring our own confusion.**

---

## Traps (each cost real time; do not re-derive)

- **Scope the drift gate to ACTIVE prompt versions.** All-versions reports ~457 false drifts.
- **Compare token multisets, not sequences.** The formatter may move `match` onto the `=`
  line; that is layout, not dialect. Sequence comparison flagged 8 reorderings per real hit.
- **`ailang chains chat <id> --stage N` shows tool RESULTS.** Do not hand-parse JSONL.
- **`api_error` means "cause unknown", not "model failed."** Six control rows read as motoko
  crashes on 2026-07-30 and were the model answering in one shot (`non_agentic`, fixed in
  `d7140fe7b`).
- **Do not select A/B benchmarks by raw pass rate.** Use `ailang eval-elo <dir> --json` and
  the expected-score logistic; keep E[pass] ≈ 0.20–0.70, nearest 0.50 first. Raw rates are
  polluted by harness crashes and by the 2026-07-29 baseline shift.
- **Pass/fail cannot answer this question on this subject.** Filtered for crashes, only two
  benchmarks have headroom (`red_black_tree` 55%, `config_file_parser` 64%). Measure
  **tokens-to-pass on benchmarks both arms pass** — that is the original hypothesis anyway
  ("use less tokens learning ailang syntax").
- **The extension must never be able to kill a run.** 0.4.1 tried to reason about which
  paths were safe and hit a 100% crash rate; 0.4.2 removed the failure mode
  (`writeFileResult` → `Delegate` on any error). Keep that shape.

---

## Before Wednesday

The scheduled fmt A/B fires **Wed 2026-08-05 03:00** (`tools/launchd/nightly-eval.sh`,
`RUN_AB_FMT`, day gate `%u == 3`). If W1 has not landed, set `AILANG_AB_FMT=0` — otherwise
it burns ~4h of rig measuring a formatter that still contradicts the prompt, and produces a
fourth ambiguous null.

## References

- `ca3d04cd8` fmt dialect fix · `6d96994da` drift ratchet · `d7140fe7b` non_agentic ·
  `ae12799d4` validity backstop · `6f8a38ddd` CLAUDE.md question index
- Data: `eval_results/fmt_ab/tokens_20260730` (write-mode),
  `eval_results/fmt_ab/reportonly_v042_20260730` (report-only)
- Evidence: `scratchpad/evidence/REAL_conversation_fmt_ON.txt` (full session, tool results)
- Memory: `project_fmt_silent_rewrite_costs_tokens.md`
