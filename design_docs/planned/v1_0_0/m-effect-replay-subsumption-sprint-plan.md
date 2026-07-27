# Sprint Plan: M-EFFECT-REPLAY-SUBSUMPTION

**Design doc**: [m-effect-replay-subsumption.md](m-effect-replay-subsumption.md) (quorum-cleared 2026-07-27)
**Target**: v1.0.0 · mission clause 4 (orchestration flagship), SHARED GATE
**Sprint ID**: `M-EFFECT-REPLAY-SUBSUMPTION`
**Progress JSON**: `.ailang/state/sprints/sprint_M-EFFECT-REPLAY-SUBSUMPTION.json`
**Planned at**: HEAD `54d6bd191e1696526052952000279aafc8bfb9bd` (`v0.30.0-205-g54d6bd191`)
**Duration**: 2.5 days nominal · 3.5 days hard ceiling
**Risk**: HIGH — compiler-core (type/pipeline) semantics change with a soundness direction
**Milestones**: 4 (M0 make room · M1 preserve rows · M2 asymmetric subsumption + diagnostics · M3 end-to-end)

---

## 0. Planner verification (what I re-checked first-party)

The design doc's premises were re-verified at HEAD `54d6bd191` with freshly rebuilt binaries
(`make quick-install && make build`; both report `AILANG v0.30.0-205-g54d6bd191-dirty`, the
`-dirty` being solely the three tracked `docs/static/benchmarks/*.json` files —
`git diff --stat HEAD -- '*.go' 'go.mod' 'go.sum' 'Makefile'` is empty).

### 0.1 The 9-fixture matrix reproduces EXACTLY (design doc CONFIRMED, not refuted)

Fixtures written to `/tmp/subsum_verify_108/`, each run unpiped as
`./bin/ailang check /tmp/subsum_verify_108/<name>.ail`, exit code read directly:

| Fixture | Doc claims | Observed at `54d6bd191` | Verdict |
|---|---|---|---|
| `blocker` (seeded → imported bare `rand_int`) | exit 1, empty `Missing effects:` | exit 1, empty payload | ✅ |
| `c1` (explicit os → imported bare) | exit 0 | exit 0 | ✅ |
| `c2` (crypto → imported bare) | exit 1, empty payload | exit 1, empty payload | ✅ |
| `c3` (bare os caller → local seeded callee) | exit 0 | exit 0 | ✅ |
| `c4` (pure caller → local seeded callee) | exit 1, `Missing effects: Rand` | exit 1, `Missing effects: Rand` | ✅ |
| `c6` (seeded → local bare wrapper) | exit 1, empty payload | exit 1, empty payload | ✅ |
| `c7` (explicit os caller → local seeded callee) | exit 0 | exit 0 | ✅ |
| `c8` (bare os caller → local crypto callee) | exit 0 | exit 0 | ✅ |
| `clock` (`Clock[mode=pinned]`) | exit 1 `EFF_PARAMS_NOT_SUPPORTED` | identical | ✅ |

**The doc's "Correction" section stands: effect modes are NOT currently invariant across local
calls.** The one-directional enforcement is real and reproducible.

### 0.2 Citations re-checked (all VERIFIED, line numbers exact or ±1)

| Claim | File:line at HEAD | Status |
|---|---|---|
| `declaredEffects` is `map[string][]string` built with `ast.EffectNames` | `internal/pipeline/validate_effects.go:110-114` | ✅ |
| `EffectNames` returns names only | `internal/ast/ast.go:116-123` | ✅ |
| Local call reconstructs a label-only row | `internal/pipeline/validate_effects.go:366-373` | ✅ |
| Bare Rand normalises to `mode=os` | `internal/types/effects.go:219-232` (`effectiveParamsOf`), `:165-170` (`DefaultModeFor`) | ✅ |
| Exactly 3 production `SubsumeEffectRows` call sites | `validate_effects.go:160,221,245` (rg over `*.go` minus tests) | ✅ |
| Label-only `EffectRowDifference` → empty `Missing effects:` | `internal/types/effects.go:653-669`; used at `validate_effects.go:161,542` | ✅ |
| Unification is a separate invariant path | `internal/types/unification_records.go:411,427-441` → `effectParamsCompatible` at `effects.go:251-253` | ✅ |
| Existing direct guard on subsumption | `internal/types/effect_params_test.go:372-394` (`TestSubsumeEffectRows_InvariantOnParams`) | ✅ |
| `effectSchema` is the single source of truth | `internal/types/effects.go:36-48` | ✅ |
| `ElaborateEffectRowWithBudgets` handles params/budgets/row-vars | `internal/types/effects.go:369-417+` | ✅ |
| Replay-contract labels | `internal/replay/contracts.go:23-40,67-75` | ✅ |
| Runtime pushes declared non-os Rand mode | `internal/eval/value.go:313` (`EffectRandMode`), `internal/eval/eval_expressions.go:192,199-215` (`extractRandMode`) | ✅ |
| Seeded runtime source + `AILANG_SEED` exist | `internal/effects/context.go:497-500` (`seedSet`), `internal/effects/rand_mode_test.go` | ✅ |

### 0.3 NEW findings the design doc does NOT contain (they change the plan)

**F1 — `examples/ai_modes.ail` is RED at HEAD with the IDENTICAL defect on the AI effect.**

```
$ ./tools/verify_examples.sh; echo $?
✗ TYPE-CHECK FAIL: examples/ai_modes.ail
    Error: effect checking failed in examples/ai_modes: effect checking failed:
    lambda at examples/ai_modes.ail:53:8 uses effects not declared in its
    ! {AI[mode=routeable]} annotation
examples: 41 type-checked · 40 ran clean · 1 run-skipped
✗ verify-examples FAILED
1
```

`summarize_routeable` (`examples/ai_modes.ail:53`) declared `! {AI[mode=routeable]}` calls
`std/ai.call` declared bare `! {AI}` (`std/ai.ail:418`, default `mode=fixed`) — the exact
blocker pattern, with the same empty `Missing effects:` payload. The bug is therefore **not
Rand-specific**, and one currently-shipped example is already broken by it. See PARKED Q1.

**F2 — the design doc's acceptance gate (`make verify-examples`) does not cover
`examples/modal_rand.ail` at all.**

`make verify-examples` (`make/examples.mk:18-28`) runs `scripts/verify_examples.go`, which
walks `examples/runnable/` (178 files). `modal_rand.ail` and `ai_modes.ail` are TOP-LEVEL
`examples/*.ail`, covered only by `make verify-examples-toplevel` → `tools/verify_examples.sh`.
M3's example gate must be `./tools/verify_examples.sh`, not `make verify-examples`.
Also: `tools/verify_examples.sh` is **not referenced by any CI workflow**
(`grep -rn verify-examples .github/workflows/` → only `make verify-examples` at
`ci.yml:216-227`). See PARKED Q2.

**F3 — `tools/verify_examples.sh:65` hardcodes `--entry main`, which constrains M3's example
shape.** After this sprint an `os`-declared `main` may not call a seeded/crypto callee
(that is precisely the c3/c7/c8 direction being fixed), and a `seeded`-declared `main` run
without `AILANG_SEED` is a loud `RAND_SEEDED_NO_SEED` error. Therefore `main` must stay
os-only and the seeded/crypto demos need **separate entrypoints** exercised by a dedicated
test, not by `main`. Furthermore, because `seeded` and `crypto` are incomparable, **no single
function can invoke both** — that is inherent to the ratified rule, not a bug.

**F4 — `UnionEffectRows` silently drops one of two conflicting modes** — a live silent
fallback that M1 turns into a soundness hole if unhandled.

`internal/types/effects.go:585-598`:

```go
switch {
case len(ap) > 0:      // 'a' wins
    ...
case len(bp) > 0:
```

Today this is harmless because `collectRequiredEffects` erases callee params anyway. **After
M1 preserves them**, a body that calls both a seeded local helper and an os local helper
produces a union that keeps only ONE mode, and the declaration is then validated against the
survivor. If `os` survives, an `os` declaration wrongly ACCEPTS a seeded callee — reintroducing
the very hole M1 exists to close. Handling multi-mode requirements is therefore a **required
M1 deliverable**, not the "non-blocking catch" the quorum log implies.

**F5 — the suggested-fix line provably keeps the wrong mode.**
`validate_effects.go:565` computes `types.UnionEffectRows(declared, required)` with `declared`
first, which under F4's `case len(ap) > 0` returns the *declared* params. So the "Suggested
fix" would echo `! {Rand[mode=os]}` back at a user whose problem is that they need
`mode=seeded`. M2's diagnostic work must fix this, not just the empty-payload symptom.

**F6 — effect-row VARIABLES change shape under M1 and must be regression-guarded.**
`ast.EffectNames` returns row-variable names as if they were labels, so `mapE`'s declared
`! {e}` is today a row with a bogus **label** `"e"`. Under `ElaborateEffectRowWithBudgets` it
becomes `Labels:{}, Tail: e`. Eleven stdlib declarations are affected:
`std/list.ail:217,228,239,250,261` (`mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE`, declared
`! {e}`), `std/stream.ail:100,146,178,237` (declared `! {Stream, e}`), and
`std/smoke.ail:43,62` (`dispatchAllTools`/`dispatchTool`, declared `! {IO | e}` — the pipe-tail
spelling, which must also survive). These pass today only because their collected requirement
is `nil`; M1 must prove they still pass.

**F7 — file-size gate headroom is thin.** `internal/types/effects.go` is **720/800** and
`internal/pipeline/validate_effects.go` is **570/800**; `make check-file-sizes`
(`make/code-health.mk:122-137`) is a hard CI gate at 800. New code must go into NEW files
(`internal/types/effect_subsumption.go`, and `internal/pipeline/validate_effects_rows.go` if
plumbing exceeds ~150 lines), not appended.

---

## 1. Conflict Surface (as this plan understands it)

### 1.1 Must CHANGE — all three production `SubsumeEffectRows` call sites

| # | Site | Today | After |
|---|---|---|---|
| 1 | `internal/pipeline/validate_effects.go:160` — inline/lambda annotations | `declared` = true source row from `typeChecker.DeclaredLambdaEffectRow`; `required` has callee modes ERASED. This is the site that actually rejects `blocker`/`c2`/`c6`. | Adopts asymmetric rule + structured diagnostic; `required` carries real callee modes |
| 2 | `internal/pipeline/validate_effects.go:221` — top-level `LetRec` bindings | `declared` = `stringSliceToEffectRow(...)`, i.e. **the declaration's own modes are erased too** — this path can never see `mode=seeded` | Receives the full elaborated declaration row; adopts asymmetric rule |
| 3 | `internal/pipeline/validate_effects.go:245` — top-level `Let` bindings | same erasure as #2 | same as #2 |

> Planner note: it is worth the executor internalising that sites #2/#3 are **doubly blind**
> today (both sides label-only), which is why `c3`/`c7`/`c8` pass and why the rejection message
> users see always comes from site #1. Fixing only the callee side leaves #2/#3 blind.

### 1.2 Must CHANGE — the erasure sources

| Location | Change |
|---|---|
| `validate_effects.go:110-114` | `declaredEffects map[string][]string` → `map[string]*types.Row` built via `types.ElaborateEffectRowWithBudgets(funcDecl.Effects)` (errors returned, never swallowed) |
| `validate_effects.go:366-373` (App/local-callee) | stop calling `stringSliceToEffectRow`; use the preserved row |
| `validate_effects.go:270-285` (`stringSliceToEffectRow`) | delete, or keep ONLY for the REPL path (`internal/repl/repl_eval.go:125` passes `surfaceAST == nil`) with a comment saying why |
| `validate_effects.go:161,542` (`EffectRowDifference`) | replaced by a structured difference |
| `validate_effects.go:565` (suggested fix) | must not echo the declared (wrong) mode — see F5 |

### 1.3 Must NOT change — the invariant path

| Location | Why it must not change |
|---|---|
| `internal/types/unification_records.go:411-441` (`Unifier.unifyRows`) | Function-VALUE effect rows stay invariant. Mark's ratification is validate-path only. |
| `internal/types/effects.go:251-253` (`effectParamsCompatible`) | The invariant predicate. Must NOT learn about subsumption edges. Verify with `git diff` that these 3 lines are untouched. |
| `internal/types/effects.go:219-232` (`effectiveParamsOf`) | Default normalisation only. A registered default must never imply an edge. |
| `internal/types/effects.go:165-170` (`DefaultModeFor`) | same |
| `internal/replay/contracts.go` | No contract-label changes. |
| `internal/eval/*` runtime dispatch | No runtime changes at all. |

### 1.4 Callers of `ValidateEffects` (blast radius)

- `internal/pipeline/pipeline_single.go:412` — passes `typeChecker.DeclaredLambdaEffectRow`
- `internal/pipeline/pipeline_module_compile.go:341` — same
- `internal/repl/repl_eval.go:125` — `surfaceAST = nil`, no lambda lookup ⇒ declared map is
  empty; must keep working (regression check: REPL effect errors unchanged)
- `internal/pipeline/effect_soundness_test.go:39` and `validate_effects_test.go:33,69,75,117,159`
  — direct test callers; signature-compatible changes only, or update them

### 1.5 Tests that guard the boundary

| Test | Action |
|---|---|
| `internal/types/effect_params_test.go:372-394` `TestSubsumeEffectRows_InvariantOnParams` | **Rewrite as a table test of the new asymmetric rule. Do NOT delete.** Rename to `TestSubsumeEffectRows_AsymmetricValidationOrdering`. |
| `internal/types/effects_test.go:432-521` `TestSubsumeEffectRows_NoHierarchy` | Keep unchanged (label-level, no effect covers another effect). |
| `internal/types/effects_test.go:522-548` `TestSubsumeEffectRows_FS_Does_Not_Cover_Env` | Keep unchanged. |
| NEW unification invariance test | Prove a `! {Rand[mode=seeded]}` function value still fails to unify where `! {Rand}` is expected. |

---

## 2. Milestones

### M0 (folded into M1, do FIRST) — make room

- Create `internal/types/effect_subsumption.go` for the edge table + validation predicate +
  structured diff (do **not** append to the 720-line `effects.go`).
- If pipeline plumbing exceeds ~150 lines, create `internal/pipeline/validate_effects_rows.go`.
- `make check-file-sizes` green immediately after, before adding behaviour.

### M1 — Preserve declared rows through validation; pin the pre-relaxation matrix (0.75 day, ~230 impl + ~200 test LOC)

**Files touched**

- `internal/pipeline/validate_effects.go` (declared map type, `validateDecl`,
  `validateLambdaAnnotations`, `collectRequiredEffects` App case)
- `internal/pipeline/validate_effects_rows.go` (NEW, if needed)
- `internal/pipeline/effect_mode_subsumption_test.go` (NEW — c1–c8 fixtures)
- `internal/pipeline/validate_effects_test.go`, `internal/pipeline/effect_soundness_test.go`
  (call-site updates only if signatures move)

**Acceptance criteria**

1. `declaredEffects` is `map[string]*types.Row`, built from `funcDecl.Effects` via
   `types.ElaborateEffectRowWithBudgets`. An elaboration error is **returned**, never
   swallowed into a nil row (CLAUDE.md no-silent-fallbacks).
2. Local-call collection (`validate_effects.go:366-373`) uses the preserved row; the
   `stringSliceToEffectRow` label-only path is gone from that site.
3. Stored declaration rows are immutable to collection: validating a decl twice yields
   identical `Params`/`Budgets`/`MinBudgets` on the stored row (explicit test; the row is
   copied before any union).
4. **Pre-relaxation matrix pinned as a test** (`ailang check` exit codes, unpiped): `c3` 0→**1**,
   `c7` 0→**1**, `c8` 0→**1**; `blocker`/`c2`/`c6` remain **1**; `c1` remains **0**;
   `c4` remains **1** with `Missing effects: Rand`.
5. **F6 guard**: `ailang check` still passes for every effect-row-variable declaration —
   at minimum `std/list.ail` `mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE`,
   `std/stream.ail` `onEvent`/`selectEvents`/`withStream`/`withSSE`, `std/smoke.ail:42,61`.
   A pipeline test compiles a module importing `std/list` and using `mapE` with an effectful
   callback.
6. **F4 multi-mode requirement**: a body calling BOTH a seeded local helper and an os local
   helper must not collapse to one mode. Two tests:
   - declared `! {Rand[mode=os]}` + demands `{seeded, os}` → **reject**, naming `seeded`;
   - declared `! {Rand[mode=seeded]}` + demands `{seeded, os}` → still rejects at M1
     (relaxation lands in M2), and **accepts** at M2.
   Plus the imported-callee analogue. Order-independence: swapping the two call sites in the
   source does not change the verdict.
7. `go test ./internal/pipeline/... ./internal/types/... ./internal/repl/...` green;
   `make check-file-sizes`, `make check-boundaries`, `make lint` green.

**Independently verifiable**: collected requirements retain `seeded`/`crypto`, and both
directions of the matrix are now accurate (over-strict, but never blind).

**Risk**: this milestone alone makes the tree *more* rejecting. Expect fallout in
`examples/runnable/` and stdlib; re-run `make verify-examples` AND `./tools/verify_examples.sh`
at the end of M1 and record any new red before proceeding.

### M2 — Validate-only asymmetric subsumption + structured diagnostics (1 day, ~200 impl + ~260 test LOC)

**Files touched**

- `internal/types/effect_subsumption.go` (NEW: `subsumptionEdges`, `ModeSubsumes`,
  structured `EffectRowDiff`)
- `internal/types/effects.go` (`SubsumeEffectRows:628-650` consults the edges; doc-comment it
  as **validation subsumption, not function-type compatibility**)
- `internal/pipeline/validate_effects.go` (`:158-166` lambda error, `:540-570`
  `formatEffectError`)
- `internal/types/effect_params_test.go` (rewrite `TestSubsumeEffectRows_InvariantOnParams`)
- `internal/types/effect_subsumption_test.go` (NEW)
- `internal/types/unification_invariance_test.go` (NEW or fold into existing unification tests)
- `internal/pipeline/effect_mode_subsumption_test.go` (flip expectations to the final matrix)

**Acceptance criteria**

1. `subsumptionEdges` registers **only** `Rand: seeded → os` and `Rand: crypto → os`.
   A test asserts the table has exactly those two entries and that AI/Clock/Net/FS have none.
2. Ordered-pair table test, `(declared, required)`:
   `(os,os)`✓ `(seeded,seeded)`✓ `(crypto,crypto)`✓ `(seeded,os)`✓ `(crypto,os)`✓ ·
   `(os,seeded)`✗ `(os,crypto)`✗ `(seeded,crypto)`✗ `(crypto,seeded)`✗.
3. Bare vs explicit-`os` equivalence holds on BOTH sides (bare declared covers explicit-os
   required and vice versa; bare declared does NOT cover seeded/crypto required).
4. Non-`mode` parameter keys stay invariant — test with AI's `scope=byok`.
5. **A registered default grants no subsumption**: AI has `DefaultModeFor = fixed` and no
   edges, so declared `AI[mode=routeable]` does NOT cover required `AI[mode=fixed]` (test).
6. `effectParamsCompatible` (`effects.go:251-253`) and `unifyRows`
   (`unification_records.go:411-441`) are byte-unchanged in the diff, and a unification test
   proves a `! {Rand[mode=seeded]}` function value still fails where `! {Rand}` is expected.
7. Structured diagnostic replaces `EffectRowDifference` at BOTH `:161` and `:542`, reporting
   effect name, param key, required value and declared value. Wording is semantic, e.g.
   `Effect mode mismatch: Rand requires mode=seeded; declaration provides mode=os`.
   **No rejection may ever print an empty `Missing effects:`** — asserted by a test that greps
   the output of every red fixture.
8. The "Suggested fix" line no longer echoes the declared mode (F5). Either drop the
   suggestion for mode mismatches or emit the required mode.
9. **Final matrix**: `blocker` **0** · `c1` **0** · `c2` **0** · `c3` **1** (names `Rand`,
   required `seeded`, declared `os`) · `c4` **1** (`Missing effects: Rand`) · `c6` **0** ·
   `c7` **1** (required `seeded`, declared `os`) · `c8` **1** (required `crypto`, declared `os`).
   Exit codes recorded by direct invocation, never through a pipe.
10. `make test`, `make lint`, `make check-boundaries`, `make check-file-sizes` green.

### M3 — End-to-end Rand acceptance + docs (0.75–1 day, ~90 example/docs + ~120 test LOC)

**Files touched**

- `examples/modal_rand.ail` (delete the `KNOWN LIMITATION` block at lines 28-37; add seeded +
  crypto functions and separate entrypoints)
- `internal/eval/` or `cmd/ailang/` integration test for seeded repeatability (NEW)
- `docs/docs/guides/parameterised-effects.md`
- `design_docs/implemented/v0_30_0/m-effect-replay-contracts.md` (status note — remove the
  obsolete limitation only)
- `CHANGELOG.md`
- `design_docs/planned/v1_0_0/m-effect-replay-subsumption.md` → move to
  `design_docs/implemented/` per repo convention (parent went to `implemented/v0_30_0/`)

**Acceptance criteria**

1. `./bin/ailang check examples/modal_rand.ail` exits 0.
2. **F3 shape**: `main` stays declared `! {Rand, IO}` and calls only os-mode functions, because
   `tools/verify_examples.sh:65` hardcodes `--entry main` and a seeded `main` without
   `AILANG_SEED` is a loud `RAND_SEEDED_NO_SEED` error. Seeded and crypto demos live in
   separate entrypoints (e.g. `main_seeded`, `main_crypto`), documented in the example header.
   Do NOT attempt one function that calls both seeded and crypto — they are incomparable.
3. Determinism: two runs of `--entry main_seeded` with the SAME `AILANG_SEED` produce
   byte-identical output; a DIFFERENT `AILANG_SEED` produces different output (non-vacuity
   control — without it the test passes on a constant).
4. `--entry main_crypto` runs clean with no `AILANG_SEED` and is not asserted deterministic.
5. Trace coverage still reports `deterministic` for seeded and `opaque` for crypto
   (`internal/replay/contracts.go`).
6. **F2 gate correction**: `./tools/verify_examples.sh` (`make verify-examples-toplevel`) is
   run and its output recorded. It is expected to still fail on `examples/ai_modes.ail` —
   that failure is PRE-EXISTING (reproduced at the sprint base before any change) and must be
   documented as such. **Do not register an AI subsumption edge to make it green** — see
   PARKED Q1.
7. `make test`, `make lint`, `make verify-examples`, `make check-boundaries`,
   `make check-file-sizes` green (or any red reproduced on a pristine base and documented).
8. CHANGELOG entry under `[Unreleased]`.

---

## 3. Day-by-day

| Day | Work |
|---|---|
| 0.25 | M0 file extraction + `make check-file-sizes` green |
| 0.5–1.0 | M1 full-row plumbing, F4 multi-mode requirement, F6 row-var guard, matrix pinned pre-relaxation |
| 1.0–2.0 | M2 edge table, asymmetric predicate, structured diagnostic, all guard tests, final matrix |
| 2.0–2.75 | M3 example rework (separate entrypoints), determinism + non-vacuity test, docs, CHANGELOG, gates |
| 2.75–3.5 | Buffer (ceiling). If full-row propagation needs a type-INFERENCE change, STOP — see §5. |

Velocity basis: the previous mission sprint (`M-COST-KPI-M4A`, 2026-07-27) landed ~500 LOC in
0.75 day at a 900 LOC/day target. This sprint is ~1120 LOC but in compiler core with a
soundness direction, so the plan deliberately budgets ~2.5x that per-LOC time.

---

## 4. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| **F4** multi-mode union silently keeps the wrong mode, reintroducing the hole M1 closes | HIGH (soundness) | M1 AC-6: two-helper tests, both declaration directions, order-independence |
| Relaxation leaks into function-value unification | HIGH | `effectParamsCompatible` + `unifyRows` byte-unchanged (diff-asserted) + a direct unification test |
| **F6** row-variable declarations change shape and break stdlib combinators | HIGH (breaks `std/list`) | M1 AC-5 explicit stdlib guard before anything else lands |
| M1's stricter direction breaks unrelated `examples/runnable/` files | MEDIUM | Run BOTH example gates at the end of M1, not only at M3 |
| **F7** file-size gate trips mid-sprint | MEDIUM | M0 extraction first, `check-file-sizes` after each milestone |
| `ElaborateEffectRowWithBudgets` errors get swallowed into `nil` (a silent fallback) | MEDIUM | M1 AC-1 |
| **F5** suggested fix echoes the wrong mode | MEDIUM | M2 AC-8 |
| REPL path (`surfaceAST == nil`) regresses | LOW | `go test ./internal/repl/...` in M1 gates |
| Executor "helpfully" registers an AI edge to green `ai_modes.ail` | MEDIUM (scope breach) | M2 AC-1 asserts the table has EXACTLY two entries; M3 AC-6 forbids it |

---

## 5. Stop conditions (do not silently widen)

The design doc's own escalation trigger is adopted verbatim: **if preserving full declared rows
cannot be confined to `internal/pipeline/validate_effects.go` + existing row elaboration and
would require changing type INFERENCE, STOP** and file a prerequisite item rather than
expanding this sprint. Recommendation on record: stop.

Additional stop conditions from this plan:
- If M1's stricter direction turns >2 `examples/runnable/` files red, stop and report the list
  before "fixing" them — that is evidence the erasure was load-bearing somewhere unexpected.
- If making the diagnostic structured requires changing `types.Row`, stop and report.

---

## 6. PARKED for Mark (headless — no human available in-session)

**Q1 (blocking for scope, non-blocking for the sprint's Rand goal).**
> `examples/ai_modes.ail` is RED at HEAD — `./tools/verify_examples.sh` fails on it — with the
> identical defect on the AI effect: `summarize_routeable` declared `! {AI[mode=routeable]}`
> calls `std/ai.call` declared bare `! {AI}` (= `mode=fixed`) and is rejected with the same
> empty `Missing effects:`. Your ratified wording was effect-generic ("declared mode subsumes
> bare/os requirement"); the design doc instantiates it for Rand only.
> **Should this sprint also register AI's `routeable → fixed` (and `replay-only → fixed`?)
> subsumption edges — which would fix a shipped example that is broken today — or does AI keep
> invariant behaviour, leaving `ai_modes.ail` red until a separate item?**
> Default if unanswered: **Rand-only**; `ai_modes.ail` stays red and is documented as
> pre-existing. Note the counter-argument: `internal/types/effects.go:624-626` comments say
> invariance exists specifically so "the routeable→fixed example becomes a typecheck
> rejection", which may be a deliberate AI-specific choice rather than the same bug.

**Q2 (process, low urgency).**
> `tools/verify_examples.sh` / `make verify-examples-toplevel` is the ONLY gate covering
> top-level `examples/*.ail` (including `modal_rand.ail`, this sprint's acceptance surface),
> and it is **not wired into CI** — `ci.yml:216-227` runs only `make verify-examples`, which
> covers `examples/runnable/`. **Should it be added to CI?** It cannot be added until Q1 is
> answered, because it is red today on `ai_modes.ail`.
> Recommendation: **not in this sprint** — file as a follow-up coupled to Q1.

---

## 7. Handoff

- Sprint JSON: `.ailang/state/sprints/sprint_M-EFFECT-REPLAY-SUBSUMPTION.json`
- Suggested branch: `sprint/m-effect-replay-subsumption`
- Suggested worktree: `.claude/worktrees/effect-subsumption`
- Reusable test harness: copy `check386` from
  `internal/pipeline/effect_row_show_interp_test.go:26-45` — it writes a module to `t.TempDir()`
  and runs the FULL module pipeline (`Config{RelaxModules: true, NoCache: true}`), which is
  required because c1/c2/c6/`blocker` import `std/rand`. The simpler
  `compileAndValidateEffects` helper (`effect_soundness_test.go:16-39`) does NOT resolve
  imports and is not sufficient for those fixtures.
- **Commit note for the controller**: `.ailang/` is gitignored (`.gitignore:77`), but sprint
  JSONs are conventionally tracked (`git ls-files .ailang/state/sprints/` returns 40+ files).
  The progress JSON therefore needs `git add -f`.

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-effect-replay-subsumption-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-EFFECT-REPLAY-SUBSUMPTION.json`
