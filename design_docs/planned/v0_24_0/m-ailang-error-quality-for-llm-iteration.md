# M-AILANG-ERROR-QUALITY: Error Messages as the Lever for Small-Model Success

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 — every persistently-failing benchmark with low recovery rate identifies a specific error-message improvement worth ~1 benchmark of pass rate per fix
**Estimated**: 3-5 days (compiler-side error-message work)
**Dependencies**: M-EVAL-METRICS-TAXONOMY (planned) provides `error_actionable_rate` and per-error-code recovery metrics; this sprint then makes that metric trend up

## Problem statement

Big models brute-force through bad error messages because they have priors and capacity to guess what AILANG meant. Small models — specifically the kind we can run continuously on a local Mac Studio — can't. **The marginal value of every error-message improvement is concentrated at the low end of the model-capability curve.** Improving for small models is strict upside for big ones; the reverse isn't true.

This sprint takes a concrete corpus — the 3 failing benchmarks from the 2026-05-23 rotation of `gemma4:26b-ailang` on the smoke tier (14/17 PASS, 3 FAIL, all in agent mode) — and treats each failure as a falsifiable hypothesis: **with a more actionable error message, would the agent have converged?**

The rotation's data is unambiguous when read against the right metric. `first_attempt_ok` at the harness level is misleading in agent mode (it measures whether the agent's *final* output passes verification on its first try, not how many internal iterations the agent took). The real iteration count is the `agent_turns` field in the result JSON:

- **Easy benchmarks (12/17)**: agent converges in **~4 turns** each (one write, one verify, maybe one fix, final verify). Total tokens ~117k. The error messages the agent saw mid-iteration were actionable — each error led to a fix.
- **Harder benchmarks that still passed (2/17)**: agent took 6-10 turns to converge. Pattern: same as above, plus one or two error-recovery rounds where the agent had to consult MCP or re-read docs.
- **The 3 failures**: agent took **5-17 turns and STILL didn't converge**, because each turn's compile error gave the agent no usable signal for the next fix. The model rewrote the same broken syntax over and over with minor variations.

The pattern is consistent: **when errors are actionable, the agent converges quickly. When errors are unactionable, the agent burns iterations on the same wrong guess.**

## Methodology

For each of the 3 failures, we extract:

1. The model's final code (what the agent thinks the answer is)
2. The error AILANG emitted
3. A rubric score of the error's actionability
4. What a "good" error would have said
5. Whether the proposed fix would have converged the agent in ≤ 3 iterations

## The 3 failures from 2026-05-23 rotation

### Failure 1: balanced_parens — `*types.TList` leaks Go-internal naming

**Model code (final attempt of 8):**
```ailang
pure func checkBalanced(chars: [string], count: int) -> bool =
  if count < 0 then false
  else
    match chars {
      [] => count == 0,
      c :: rest =>
        if c == "(" then checkBalanced(rest, count + 1)
        else if c == ")" then checkBalanced(rest, count - 1)
        else checkBalanced(rest, count)
    }
```

**Error emitted by AILANG:**
```
Warning: stdlib version mismatch: expected v0.21.0-46-g9022b26b-dirty, found v0.21.0
Error: type error in benchmark/solution (decl 0): type unification failed at [list pattern]: cannot unify function type with *types.TList
```

**Rubric scoring:**

| Quality | Score | Notes |
|---|---|---|
| File:line:column anchor | ❌ FAIL | Just says "decl 0" — 8 attempts, model never figured out which line |
| Uses AILANG-level type names | ❌ FAIL | Leaks `*types.TList` (Go internal); should be `[string]` or `[T]` |
| Names the offending construct in source | ❌ FAIL | Says "function type" — which function? `checkBalanced`? `chars`? `::`? |
| Provides "Did you mean…" suggestion | ❌ FAIL | None |
| Quotes the problematic source token | ❌ FAIL | None |
| **Actionability score** | **0/5** | The agent had no anchor |

**Likely root cause** (what the model actually got wrong): the cons pattern `c :: rest`. AILANG uses different list-pattern syntax (likely `[c, ...rest]` or similar) — the model wrote Haskell-style.

**A better error would say:**

```
Error at solution.ail:11:9:
  Pattern `c :: rest` not recognized as a list pattern.
  AILANG uses [head, ...tail] for cons patterns, not the Haskell-style ::.
  
  Did you mean:
    [c, ...rest] => ...
  
  See: `ailang docs std/list` (pattern matching examples)
```

**Expected impact**: agent converges in ≤ 2 iterations (one to read the error, one to apply the suggestion).

### Failure 2: canonical_convergence — error IS decent, but suggestion is generic

**Model code:**
```ailang
pure func count_pos_even(xs: [int]) -> int =
  let evens_and_pos = filter(\x. x > 0 && x % 2 == 0, xs);  -- ← bug here
  length(evens_and_pos)

export func main() -> () ! {IO} =
  let input = [1, 2, 3, 4, -5, -6, 8, 0, 10];
  println(show(count_pos_even(input)))
```

**Error emitted by AILANG:**
```
PAR_NO_PREFIX_PARSE at solution.ail:5:58: unexpected token in expression: ;
Suggestion: This token cannot start an expression
```

**Rubric scoring:**

| Quality | Score | Notes |
|---|---|---|
| File:line:column anchor | ✅ PASS | `solution.ail:5:58` — precise |
| Uses AILANG-level type names | n/a | Parser error |
| Names the offending construct | ✅ PASS | "unexpected token in expression: ;" |
| Provides "Did you mean…" suggestion | ⚠️ WEAK | "This token cannot start an expression" — true but vague |
| Quotes the problematic source token | ✅ PASS | Says `;` |
| **Actionability score** | **3.5/5** | Good location + token, weak suggestion |

**Root cause**: the model wrote `let evens_and_pos = filter(...); length(evens_and_pos)` — using `;` to chain a let binding with an expression in expression-body function form. AILANG's expression-body chains use `let x = a in let y = b in finalExpr`, not `;`. To use `;`, the function body must be wrapped in `{}` (block form).

**A better error would say:**

```
PAR_LET_CHAIN_SYNTAX at solution.ail:5:58:
  Found `;` after a `let` binding in an expression-body function.
  AILANG expression-body functions chain lets with `let .. in ..`, not `;`.
  
  Two valid forms:
  
  (1) Expression body — use `in`:
      pure func count_pos_even(xs: [int]) -> int =
        let evens_and_pos = filter(...) in
        length(evens_and_pos)
  
  (2) Block body — use `{}` with `;`:
      pure func count_pos_even(xs: [int]) -> int = {
        let evens_and_pos = filter(...);
        length(evens_and_pos)
      }
  
  See: ailang docs let-bindings
```

**Expected impact**: agent converges in 1 iteration. This is a high-frequency error (the let-syntax-styles confusion is in the v0.9.0 agent prompt's "common mistakes" but the parser doesn't reinforce it).

### Failure 3: dense_operator_program — `|` token without operator-name guidance

**Model code (relevant excerpt):**
```ailang
let val1 = (5 << 3) + (16 >> 2) in
let v1 = 12 & 10 in
let v2 = 4 ^ 6 in
let val2 = v1 | v2 in       -- ← bug: AILANG doesn't allow `|` as binary operator here
```

**Error emitted by AILANG:**
```
PAR_NO_PREFIX_PARSE at solution.ail:9:17: unexpected token in expression: |
Suggestion: This token cannot start an expression
```

**Rubric scoring:**

| Quality | Score | Notes |
|---|---|---|
| File:line:column anchor | ✅ PASS | `solution.ail:9:17` |
| Uses AILANG-level type names | n/a | Parser error |
| Names the offending construct | ✅ PASS | "unexpected token in expression: \|" |
| Provides "Did you mean…" suggestion | ❌ FAIL | "This token cannot start an expression" — tells the model NOTHING about what bitwise-OR is called in AILANG |
| Quotes the problematic source token | ✅ PASS | Says `\|` |
| **Actionability score** | **3/5** | Anchor good, but no operator-name guidance |

**Root cause**: AILANG's `|` token is reserved for pattern alternatives in `match` expressions, not as binary bitwise-OR. AILANG's bitwise OR uses a different operator (`bitOr` builtin? distinct keyword?). The model knows what it WANTS (bitwise OR — they got `<<`, `>>`, `&`, `^` right) but doesn't know AILANG's name for it.

**A better error would say:**

```
PAR_RESERVED_TOKEN at solution.ail:9:17:
  Token `|` is reserved for pattern alternatives in `match` expressions,
  not as a binary operator.
  
  AILANG bitwise operators:
    bitOr(a, b)   for bitwise OR
    a & b         for bitwise AND
    a ^ b         for bitwise XOR
    a << n        for left shift
    a >> n        for right shift
  
  Did you mean:
    let val2 = bitOr(v1, v2) in
  
  See: `ailang docs operators` for the full list
```

**Expected impact**: agent converges in 1 iteration with the operator name in hand. Without it, the model tried 5+ rewrites in 18 minutes without finding the right token (the rotation log shows `dense_operator_program` at 1097s wall, 447k tokens).

## Cross-failure patterns (the rubric distilled)

Five qualities matter most, in this order:

1. **File:line:column anchor** — without this, the LLM has to guess which line. Even big models waste turns on this. (2/3 failures had this; 1 did not.)

2. **AILANG-level type/construct naming** — Go internal types like `*types.TList` are catastrophic; the LLM has never seen the Go source. (1/3 failures leaked Go internals.)

3. **Quoted source token** — "unexpected token in expression: `;`" is dramatically more useful than "syntax error". (2/3 had this.)

4. **"Did you mean" suggestion with concrete alternative** — the difference between converging in 1 turn vs 18 minutes. The "did you mean" should name the AILANG construct that would work. (0/3 had concrete alternatives.)

5. **Pointer to canonical docs** — even a single-line "see `ailang docs <topic>`" gives the agent a tool-call to make for self-recovery. (0/3 had this.)

## Why this matters for the rig's strategic value

The local-Ollama rig produces failure data continuously. Each persistent failure with low recovery rate is **a specific compiler PR worth writing**. The metrics taxonomy doc (M-EVAL-METRICS-TAXONOMY) defines:

- `error_actionable_rate`: % of errors with file:line + suggestion (currently estimated 30-50% on these 3 cases)
- `error_internal_leak_rate`: % of errors leaking Go-internal types (currently 33% on these 3 cases — the balanced_parens one)
- `error_to_recovery_correlation` per error_code: which error codes have 0% next-attempt recovery rate?

The 0%-recovery error codes are the work queue. Every fix-an-error PR is measurable: the same rotation re-runs, the same metric moves up, the same benchmark gains a pass.

**This is the compounding loop:**
1. Eval rig surfaces unactionable errors (free, continuous)
2. AILANG team improves the worst-recovery errors (one PR per error class)
3. Same rotation re-runs and the previously-failing benchmark passes
4. Pass rate ratchets up without any prompt or model changes
5. The improvements help every future model the rig tests — small or large

## Proposed AILANG error-message audit checklist (for every compiler PR)

Add to `CONTRIBUTING.md` and the parser/typechecker review template:

```
Error-message quality checklist (per new error code or modification):

[ ] Has file:line:column position (no "decl 0" or unanchored errors)
[ ] Uses AILANG-level type names (no `*types.TList`, no `internal/types.*`)
[ ] Names the specific offending construct in source (with line snippet ideally)
[ ] Provides a "Did you mean..." suggestion with concrete AILANG alternative
[ ] References a docs section the user can read (`ailang docs <topic>`)
[ ] Tested against the eval rig: does the next-attempt recovery rate move?
```

The last item is the killer: any error code's improvement can be measured against the eval rig's same-rotation re-run.

## The 3 immediate compiler improvements from this analysis

Ranked by expected pass-rate gain on the smoke tier:

| Priority | Error code | Fix | Expected impact |
|---|---|---|---|
| **P0** | TYPE_UNIFY_LIST_PATTERN | Surface AILANG list-pattern syntax (`[head, ...tail]`) on cons-pattern errors. Strip `*types.*` from emitted error strings everywhere. | +1 benchmark (balanced_parens) |
| **P1** | PAR_NO_PREFIX_PARSE on `|` token | Detect `|` in non-pattern context; emit "use `bitOr(a, b)` for bitwise OR". | +1 benchmark (dense_operator_program) |
| **P2** | PAR_NO_PREFIX_PARSE on `;` in expr body | Detect `let ... ;` pattern; suggest both let-chain and block forms. | +1 benchmark (canonical_convergence) |

Three small compiler PRs → smoke tier moves from 14/17 (82%) to 17/17 (100%) on the same gemma4:26b-ailang model. **Without changing the model, the prompt, or the harness.**

## Conflict surface

Touches:
- `internal/parser/errors.go` (for PAR_NO_PREFIX_PARSE improvements)
- `internal/types/unify.go` (strip Go internals, add AILANG-level type names)
- `internal/elaborate/errors.go` (location info)
- The error catalog in `internal/errors/` (new codes: `TYPE_UNIFY_LIST_PATTERN`, `PAR_RESERVED_TOKEN`, `PAR_LET_CHAIN_SYNTAX`)

Does NOT touch runtime, codegen, harness, or stdlib semantics — pure error-message reporting.

> **Cross-reference (added 2026-06-08):** some *runtime* errors that look like an error-quality
> problem are actually **type-soundness holes** — the program should have been rejected at compile
> time, so the fix is a real type-check, not a better runtime message. The canonical example is
> `_str_join: expected string, got tagged value` from `json_parse`: a `Json`/`[int]` value reaches
> a `[string]` position because list-element unification drops the element-type check. That belongs
> to [M-TYPE-LIST-ELEMENT-SOUNDNESS](./m-type-list-element-soundness.md), not this doc. Rule of
> thumb: if a better *message* would still leave an invalid program running, it's a soundness bug,
> not an error-quality bug.

The Conflict Surface analysis (per CLAUDE.md guidance for parser/type changes):

| Existing construct | Still works after this change? |
|---|---|
| All currently-valid AILANG programs | YES — pure error-message reporting, no syntax/semantics change |
| Error message string matching in tests | Yes if tests use error codes; **NO** if tests match exact string content. Audit needed. |
| LSP error reporting | Should auto-pick up improved messages — LSP reads from the same emitter |

## What "done" looks like

1. Three new error codes registered with the improved messages (`TYPE_UNIFY_LIST_PATTERN`, `PAR_RESERVED_TOKEN`, `PAR_LET_CHAIN_SYNTAX`)
2. Each error includes file:line:column, AILANG-level type names (no Go leakage), "did you mean" with concrete alternative, and `ailang docs <topic>` pointer
3. The audit checklist is in `CONTRIBUTING.md`
4. Re-running the 2026-05-23 rotation under the same gemma4:26b-ailang + same prompt config + same MCP wiring shows the 3 previously-failing benchmarks now pass (verified via the leaderboard's trend-delta cell)
5. The eval-trend candidates command's output for that error_code shows recovery rate > 0% (probably > 80%)

## Why this is the highest-leverage AILANG-side investment right now

We just shipped:
- Sampling-config that prevents repetition collapse (M-EVAL-OS-LONGITUDINAL post-sprint fixes)
- MCP wiring so the agent can discover AILANG dynamically
- Slim seed prompt + env-facts to avoid filesystem thrash
- Modelfile variant with proper sampling tuning baked in

Result: 14/17 pass rate on a 26B local model in agent mode — each pass achieved via 4-10 turn iteration where the type-checker errors were actionable enough to guide each next fix.

The remaining gap (3/17 failures) is **not** a model-capability ceiling — it's an error-message-quality ceiling. In each failure the model wrote plausible AILANG; the type-checker rejected it; the agent rewrote with slight variations for 5-17 turns, all rejected; the agent gave up. Three small compiler PRs, each measurable on the same rotation, each gaining one benchmark.

This is the kind of work that makes AILANG genuinely "the language designed for AI code synthesis" — not a slogan, but an architectural commitment: **errors are part of the human-AI interface, designed to be acted on.**
