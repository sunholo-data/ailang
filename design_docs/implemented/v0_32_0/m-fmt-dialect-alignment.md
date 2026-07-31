# M-FMT-DIALECT-ALIGNMENT

**Status:** implemented 2026-07-31 (W1–W4 landed; AC5 pending the scheduled run)
**Created:** 2026-07-31
**Deadline:** before Wednesday 2026-08-05 03:00 (the scheduled fmt A/B) — **MET**
**Owner:** next session — this doc exists so nothing below has to be re-derived

> **Wednesday decision: let the A/B run.** W1 landed, so `AILANG_AB_FMT` stays at
> its default of 1 (`tools/launchd/nightly-eval.sh:181`, day gate `%u == 3`). The
> abort clause at the bottom of this doc does NOT fire. See the Verification Log.

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

---

## Verification Log (2026-07-31)

### The failing shape, finally isolated — it was never in the formatter

W1 said "the failing shape has NOT been isolated — that is the first task." It is
now, and the answer is that **the WIP was not the bug**. The 6 corpus files
(the doc said 7; the measured count is 6) did not fail a round-trip *comparison* —
their formatted output **did not re-parse at all**:

```
PAR_UNEXPECTED_TOKEN: expected next token to be }, got STRING_PART
```

Minimal repro, with no formatter involved:

```ailang
export func f() -> string {
  let a = "x"
  "${a} y"        -- expected }, got STRING_PART
}
```

`ailang fmt` drops `;` and relies on the R2 newline soft separator
(`peekStartsNewlineBlockStatement`), whose statement-starter set was
`let/letrec/if/match/identifier` — **every literal was excluded**. So a block
whose last statement began with a literal did not parse. It failed for `"s"`,
`"${a}"`, `42`, `3.5`, `true` alike; only re-sugaring made fmt start *emitting*
that shape. The fix adds the literal starters (`internal/parser/parser_func.go`,
`peekIsLiteralStatementStart`). `(`, `[`, `{` stay excluded — they genuinely
continue an expression (call / indexing / record-update), the M-GAP2 ambiguity R2
was built around; `a\n+ 2` and `a\n++ "y"` are still one expression, and that is
tested.

This also means a model taught both newline separators and `"${x}"` was hitting an
unactionable parse error. That is a dialect trap of exactly the kind this doc exists
to remove.

### The stored WIP had four latent bugs beyond that

`artifacts/expr_interp_WIP.go.txt` was re-derived from scratch rather than copied,
which turned out to matter. The WIP re-sugared any left-spine chain of literals and
`show()` calls with `len(parts) >= 2` and no further conditions. Missing:

1. **No "at least one hole" check.** `concat_String("a", "b")` — hand-written, no
   interpolation — would have been rewritten to `"ab"`. Silent corruption.
2. **No adjacent-literal check.** `concat_String(concat_String("a","b"), show(n))`
   → `"ab${n}"`, which re-parses with ONE text part. Round-trip break.
3. **No empty-literal check.** The parser *elides* empty parts, so a hand-written
   empty segment would vanish.
4. **No newline / `--` refusal** when rendering a hole.

It also hand-rolled its escaping instead of reusing `escapeString`, so it dropped
`\r` and the `\u{...}` control-character forms. None of these caused the corpus
failure — the parser gap did — but all four are in the shipped guards, each with a
test in `internal/format/interp_test.go`.

### Divergence inventory, closed out

Scoped to ACTIVE prompt versions. Five spellings realigned; in every case both
spellings parse to an **identical** AST, which is what makes emitting the taught
one round-trip exactly:

| # | Divergence | Status |
|---|---|---|
| 1 | `[int]` → `list[int]` | FIXED — `ca3d04cd8` |
| 2 | `h :: t` → `::(h, t)` | FIXED — `ca3d04cd8` |
| 3 | `"${x}"` → `concat_String(...)` | **FIXED** — `internal/format/interp.go` |
| 4 | `_r0` row variable leaking | **FIXED** — `{ email: string, ... }` |
| 5 | `int -> int` → `(int) -> int` | **FIXED** — 10 prompt blocks, the second-largest divergence and NOT in the original inventory |
| 6 | `now()` → `now(())` | **FIXED** — 5 prompt blocks, also not in the original inventory |
| 7 | `{name, age}` → `{name: name, age: age}` | **FIXED** — 2 prompt blocks, also not in the original inventory |

Divergences 5–7 were invisible because `knownDivergence` excused any block whose
output gained a `concat_String` — which waved through blocks drifting for
interpolation **and** another reason. The true residual was **16, not 9**. The
allowlist is deleted; an allowlist that excuses nothing is a rubber stamp.

### Acceptance criteria

| # | Criterion | Result |
|---|---|---|
| 1 | `go test ./internal/format/` green incl. `TestCorpusComment` + drift gate | ✅ green; `./internal/parser/` green too |
| 2 | `knownDrift` lowered from 9 | ✅ **9 → 2** (9→4 formatter, 4→2 prompt v0.16.4) |
| 3 | `ailang fmt` a no-op on prompt interpolation examples | ✅ verified on the installed binary — returns `println("${n} x")`, byte-identical input/output |
| 4 | `make install` succeeds, `ailang --version` matches HEAD | ✅ `v0.31.0-21-g349d28c2e`, commit `349d28c` = HEAD. The stated blocker (`UpdateStageMetrics` arity) was resolved independently and did not recur. |
| 5 | Re-run the A/B, report paired Wilcoxon on tokens | ⏳ **NOT DONE — a ~4h rig run.** Fires Wed 2026-08-05 03:00. This is the first run that measures fmt rather than measuring our own confusion. |

`go test ./...` is otherwise green. Two unrelated failures were confirmed not to
be from this work: `TestNetHttpPost` (httpbin.org returned 503) and
`TestSolve_HardTimeout_FakeSolverIgnoringT` (passes on re-run; flaky child-pid
race). Neither package carries any change from this sprint.

### The 4 remaining drifts — diagnosed, not unknown

Recorded in full at `internal/format/dialect_drift_test.go`:

- **#19** — a block containing the prompt's own ❌-WRONG example, which fmt
  reformats. Nothing to fix in fmt. *(Side finding: it parses today, so the
  prompt's "parse error" claim is stale — that belongs to prompt-manager.)*
- **#63** — `=> { -- comment \n expr }` unwraps to `=> expr`; the block existed
  only to host a comment, and `Source()` drops comments by construction.
- **#71** — the prompt writes `Result[List[string], string]`, fmt emits
  `Result[[string], string]`. `[T]` is taught 64 times, but `List[T]` is what
  `std/json.ail:146` actually writes, so "fix the prompt" would put it at odds
  with stdlib source. **Needs a decision on which spelling is canonical** — not a
  formatter change. Deliberately not touched: perturbing the active prompt days
  before the A/B would confound its control text.
- **#86** — `price - (price * pct) / 100` → `price - price * pct / 100`. The AST
  has no ParenExpr node (design V20), so redundant source parens are absent from
  the tree and unrecoverable. Semantically identical.

### End-to-end verification (2026-07-31, after the code landed)

Unit tests are not evidence that the loop works, so both halves were run for real.

**fmt, against the exact input from the bug report.** Piping the original
`contains` function through the actual `scripts/hooks/format_ail.sh`:

| | before (the bug) | now |
|---|---|---|
| `xs: [int]` | → `list[int]` | **preserved** |
| `h :: t` | → `::(h, t)` | **preserved** |
| `"${n}: ${show(..)}"` | → `concat_String(..)` chain | **preserved** |

The only remaining changes are layout (`match` pulled onto the `=` line, a space
after commas) — precisely what the drift gate's token-MULTISET rule exists to
allow.

**W3, by running a motoko benchmark.** `ailang eval-suite --agent --models
motoko-gemma-4 --benchmarks recursion_fibonacci`. A `motoko:session_*` chain
appeared with NO manual `ailang chains import-motoko`, and `ailang chains chat
<id> --stage 1` shows what was previously absent:

```
─── Turn 1 (assistant) ───   [tool] ReadFile
─── Turn 2 (user) ───        [result] {"tool":"ReadFile","path":"..."}
─── Turn 3 (assistant) ───   [tool] BashExec
─── Turn 4 (user) ───        [result] {"tool":"BashExec","cmd":"cat ..."}
```

The `[result]` turns are the point. Before this, the banked transcript held only
the `[tool]` lines.

**And the run found a gap the unit tests could not.** A second run was killed at
the wall clock, and produced NO `motoko:` chain. Cause: when the executor returns
a Go error, `RunAgentBenchmarkWithExecutor` returned `nil` for the result, so
`SessionJSONLPath` never reached the caller — the import could not fire on
crashed or thrashing runs, which are exactly the runs whose transcript you need,
and are how this bug was found in the first place. Fixed: the two post-execution
error paths now return a diagnostics-only result carrying the session path and
nothing else, and the caller banks the transcript before writing the `api_error`
row. Half-closing this blind spot would have repeated the original mistake.

**Not covered by this run:** the `cloud` motoko profile does not load
`motoko_ext_fmt`, so the transcript contains zero fmt messages. This verified
transcript banking, NOT fmt-inside-the-agent-loop; the latter is verified
directly against the binary above, and measured by AC5 on Wednesday.

### W3 — tool results now reach eval data

`internal/executor/motoko/motoko.go` already resolved the session JSONL path
(`findSessionJSONL`) and threw it away. It now travels as
`ProviderData["motoko_session_jsonl"]` → `AgentBenchmarkResult.SessionJSONLPath` →
`ImportMotokoSession` in `cmd/ailang/eval_benchmark.go`, as a sibling to the Claude
`claudehistory` branch that was already there. Best-effort by design: a failed
import reports on stderr and never fails a benchmark row — it is diagnostics, not
measurement. This is the blind spot that hid the bug for two weeks.

---

## References

- `ca3d04cd8` fmt dialect fix · `6d96994da` drift ratchet · `d7140fe7b` non_agentic ·
  `ae12799d4` validity backstop · `6f8a38ddd` CLAUDE.md question index
- Data: `eval_results/fmt_ab/tokens_20260730` (write-mode),
  `eval_results/fmt_ab/reportonly_v042_20260730` (report-only)
- Evidence: `scratchpad/evidence/REAL_conversation_fmt_ON.txt` (full session, tool results)
- Memory: `project_fmt_silent_rewrite_costs_tokens.md`
