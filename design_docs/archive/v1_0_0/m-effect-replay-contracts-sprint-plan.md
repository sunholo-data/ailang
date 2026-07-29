# Sprint Plan: M-EFFECT-REPLAY-CONTRACTS (Rand pilot)

**Design doc**: [m-effect-replay-contracts.md](m-effect-replay-contracts.md)
**Status**: PLANNED — ready for sprint-executor
**Planned**: 2026-07-24 (iter-99+; baseline `v0.30.0-155-g541c1950f`)
**Duration**: 3 days (~20h) · **Risk**: Medium
**Sprint JSON**: `.ailang/state/sprints/sprint_M-EFFECT-REPLAY-CONTRACTS.json`

Mark's option-(b) decision (2026-07-24) is a HARD constraint set — see the design doc Status
header. This plan does NOT re-open any of it. It resolves the two plan-time spike/decision tasks
the doc delegates to the planner, corrects the doc's drifted premises, and lays out day-by-day
milestones mapped to the doc's Success Criteria.

---

## 0. Premise verification against the current tree

Rebuilt `make quick-install`; binary at **`v0.30.0-155-g541c1950f-dirty`** (doc pinned
`v0.30.0-154-gb326c3fd3` — 1 dev commit ahead, no material drift). All probes below run at HEAD.

| Doc premise | Verified? | Note / correction |
|---|---|---|
| `internal/replay/` does not exist | ✅ | `ls internal/replay` → no such dir. |
| `internal/builtins/rand.go`: ONE global `math/rand` source, crypto-seeded at init, re-seedable via `SetRandSeed`/`_rand_seed` | ✅ **line refs drifted** | `randSource` is at **rand.go:19** (doc said `:28` — that's the `rand.New(...)` assignment inside `init`); `cryptoSeed` at **:41**; `SetRandSeed` at **:51**. Behaviour exactly as doc states. |
| Runtime effect handlers have no access to declared effect params (`grep Params internal/effects/*.go` non-test empty) | ✅ | Confirmed empty. |
| M-CRYPTORAND never landed (`grep CryptoRand\|crypto_random` empty) | ✅ | Confirmed empty across `internal/ std/ examples/`. |
| `_rand_int` is an effect-tagged builtin (`Effect: "Rand"`) drawing from the global `randSource` in the builtins package | ✅ | `RegisterEffectBuiltin{Effect:"Rand"}`; draws at rand.go:108/165/215 under `randMu`. This is the dispatch site. |
| `examples/modal_rand.ail:43` seeds a **seeded-mode** function via `rand_seed(42)` | ⚠️ **line ref off-by-one** | `export func deterministic_roll() ! {Rand[mode=seeded]}` is at **line 42**; the offending `rand_seed(42)` call is at **line 44** (inside the body). The doc's substance (seeded-mode fn seeded via os-mode `rand_seed`) is CORRECT — this IS a design bug to fix. |
| `examples/expected_fail/effect_budgets_multi.ail:62` seeds a **seeded-mode** function via `rand_seed(42)` | ❌ **PREMISE WRONG — flag to Mark/executor** | Line 62 `rand_seed(42)` is inside `main() -> () ! {IO, Rand, Clock}` — a **bare `!{Rand}` = os-mode** context. There is **NO `mode=seeded` anywhere** in this file (`grep -n seeded` empty). Under the settled contract, an os-mode `rand_seed(42)` is CORRECT and must stay byte-identical — it is NOT a bug. **Correction:** this file needs NO migration. The Success-Criteria checkbox and Verification-Log rows 2 & 4 that name it as a seeded-mode-via-`rand_seed` bug are mistaken. Executor must NOT touch its `rand_seed(42)` (doing so would violate the bare-Rand golden gate). See M4 for the corrected scope. |
| Only two seeded-mode-via-`rand_seed` callers exist in-tree | ⚠️ **corrected to ONE** | After the above: only `modal_rand.ail`'s `deterministic_roll` is a genuine seeded-mode-via-`rand_seed` bug. Executor should re-run the sweep (`grep -rn "mode=seeded" examples/ std/` then check each for `rand_seed` in-body) to confirm no others; the caller sweep otherwise verified (all `std/rand` exports are `mode=os`; no `std/game`). |
| Sprint-1 `m-effect-mode-validation` landed | ✅ | Lives in `internal/types/effects.go` (`effectSchema`, `defaultEffectModes`, `validateEffectParams`, `EffectModeFor`). `design_docs/implemented/v0_30_0/`. |
| `EffContext` carries `Seed int64` from `AILANG_SEED` | ✅ (bonus) | `internal/effects/context.go:128` — already the `AILANG_SEED` carrier (Clock virtual time). Natural home for the seeded-mode source seed. |
| Trace `EffectEvent` is additive-extensible; has `Deterministic *bool`, precedent `Route` | ✅ | `internal/trace/schema.go:65` — add `Mode`/`Contract` as `omitempty` additive fields. |

**Net premise corrections for the executor (do not silently absorb):**
1. **`effect_budgets_multi.ail` is NOT a bug** — remove it from migration scope; its os-mode
   `rand_seed(42)` stays. This actually gives us a ready-made **bare-Rand-plus-`rand_seed` golden
   fixture** (Success Criterion / Verification-Log row 3) — repurpose it as the golden gate input
   instead of migrating it.
2. Line references `rand.go:28`→`:19`, `modal_rand.ail:43`→`:44`, and the `effect_budgets_multi`
   rows in the doc's Verification Log should be corrected when the doc is moved to `implemented/`.

---

## 1. Spike Decision 1 — Mode → dispatch mechanism  ✅ RESOLVED: **option (a) context-threading** (doc's recommended (b) is INFEASIBLE as literally specced)

**The doc recommends (b): elaboration lowers moded ops to distinct builtins
(`_rand_int` vs `_rand_int_seeded` vs `_rand_int_crypto`).** A short code spike against the real
elaboration→builtin path shows **(b) cannot work as written**, for a concrete structural reason:

- The mode lives on a **function's declared effect row** (e.g. `deterministic_roll() ! {Rand[mode=seeded]}`),
  readable via `types.EffectModeFor(row, "Rand")` and per-lambda via
  `CoreTypeChecker.DeclaredLambdaEffectRow(nodeID)` (`typechecker_effect_row_issue386.go:24`).
- BUT the `_rand_int` builtin is **never referenced at a moded site**. It is referenced only inside
  `std/rand.rand_int`, whose own signature is bare `!{Rand}` (mode=os). The call chain is
  `deterministic_roll [seeded]` → `std/rand.rand_int [os]` → `_rand_int`. A lowering pass keyed on
  the effect row at the `_rand_int` reference site would ALWAYS see `os` — the outer `seeded` mode
  never reaches it. Lowering would require inlining/specialising `std/rand.rand_int` per caller-mode
  (a monomorphization-scale change) — far more than "the smallest diff". This is the decisive finding.
- The existing AI-mode precedent (`routing_flags.go:227` reads `EffectModeFor(fn.EffectRow, "AI")`)
  is a **whole-program CLI gate on the entrypoint's row**, not per-call runtime dispatch — it does
  not generalise to per-draw Rand dispatch either.

**Decision: (a) — thread the mode as evaluator/effect context.** Concretely:
- When the evaluator enters a lambda that has a declared effect row (`DeclaredLambdaEffectRow(nodeID)`),
  push the resolved `Rand` mode (`EffectModeFor(row,"Rand")`, default `os`) onto the effect context
  (`EffContext`, already cloned per-request at `eval_evaluator.go:170`). Pop on exit.
- `_rand_int`/`_rand_float`/`_rand_bool` read the current mode off the context and dispatch to the
  right source (os global / seeded per-context source / crypto). The builtin registry is unchanged;
  no new builtin names; no iface/cache churn (this also **retires the doc's "lowering leaks mode into
  iface/caches" risk** — it no longer applies).
- `EffContext` is the right carrier: it already holds `Seed int64` from `AILANG_SEED`
  (`context.go:128`) and is per-request-cloned, satisfying the "separate per-context seeded source"
  and "concurrent-evaluator safety" design constraints directly.

**Rationale recorded:** (b) was recommended for "smallest diff / runtime type-ignorant", but the
diff is only small if the mode is visible at the builtin reference site — it is not, because stdlib
wrappers erase it. (a) keeps the runtime nearly type-ignorant too (it reads a string mode off a
context it already threads), touches `eval` + `builtins/rand.go` + a small `EffContext` field, and is
strictly smaller than the wrapper-inlining (b) would require. **Cost impact vs doc estimate: neutral**
(the doc allotted the mode-threading as "the substantive work" either way; row 30-32 of the doc
already flags Params-threading as the real cost). The M0 spike is confirmation-only, not exploratory.

> **Executor note:** the mode must be re-established at the lambda entry, not the module-load point,
> so a `seeded`-mode function calling `std/rand.rand_int` still draws seeded. Verify with the M2
> cross-wrapper integration test (`deterministic_roll` under a pinned seed → identical sequences).

---

## 2. Spike Decision 2 — Registry-duplication check  ✅ RESOLVED: **distinct concerns, single source of legal modes, no drift — with one guard rail**

- **Sprint-1's registry** is `effectSchema` in `internal/types/effects.go:36` — a **validation**
  table: effect → param-key → the CLOSED set of legal values (`Rand: mode ∈ {os,seeded,crypto}`).
  It answers *"is this (effect,mode) legal?"* and is enforced at elaboration by `validateEffectParams`.
- **This sprint's registry** is `internal/replay/contracts.go` — a **taxonomy** table:
  (effect, mode) → contract label ∈ {deterministic, re-sampleable, opaque}. It answers *"what replay
  semantics does this legal (effect,mode) have?"* Consumed by trace emission now, replay harnesses later.
- **Distinct concerns, confirmed.** No behavioural overlap: validation gates legality at compile time;
  the replay table classifies already-legal pairs at runtime/trace time.
- **Drift guard (single source of legal modes):** the replay table's KEYS must be a subset of
  `effectSchema`'s legal (effect,mode) pairs. To prevent silent drift, add a
  **`TestReplayContractsAreLegalModes`** unit test asserting every `(effect,mode)` key in
  `internal/replay/contracts.go` satisfies `types.EffectModeFor`-legality against `effectSchema`
  (mirrors the existing `TestEffectSchemaDefaultsConsistent` invariant). `effectSchema` stays the
  single source of *which modes exist*; `replay.contracts` only *labels* them. **Finding: no
  duplication; add the cross-table invariant test so it stays that way.**

---

## 3. Milestone Breakdown (day-by-day)

Velocity note: recent V1 sprints land 150-250 LOC/day incl. tests; this is a plumbing sprint
(threading + one small table + tests), so estimates are conservative.

### Day 1 — M0 (spike confirmation + decision record) + M1 (registry)

**M0 — Dispatch-mechanism spike + decision record** (~1h, ~0 LOC)
- Confirm §1 finding in-tree: write a throwaway test that `deterministic_roll`'s `_rand_int` draw is
  reached with `EffContext` mode `os` today (proving lowering-at-site sees `os`). Record decision (a)
  + rationale in the sprint JSON `notes` and in the design doc's Design-Freeze open checkbox.
- **AC:** Design-Freeze "Mode→dispatch mechanism" checkbox resolved to (a) with rationale; registry-
  duplication finding recorded.

**M1 — Replay contract registry** (`internal/replay/contracts.go`, ~150 LOC + ~120 test) (~4h)
- Static table + `ContractFor(effect, mode string) (Contract, bool)`; `Contract` = label enum
  {Deterministic, ReSampleable, Opaque}. Populate: Rand seeded→deterministic, os→re-sampleable,
  crypto→opaque; AI fixed→deterministic, routeable→re-sampleable, replay-only→opaque (labels per
  parent doc Example-4 table; AI is labels-only, no dispatch here).
- Add `TestReplayContractsAreLegalModes` cross-table invariant (spike-2 guard).
- **AC (→ Success Criterion 1):** registry exists, 3 Rand rows + 3 AI label rows; lookup API tested;
  cross-table legality invariant green.

### Day 2 — M2 (three-mode Rand dispatch + tests)

**M2 — Mode-aware Rand dispatch** (`EffContext` mode field + `internal/builtins/rand.go` + eval lambda-entry threading, ~180 LOC + ~200 test) (~7h)
- Thread `Rand` mode onto `EffContext` at moded-lambda entry (§1); default `os`.
- `builtins/rand.go`: three sources —
  - **os**: existing `randSource`/`randMu`, unchanged incl. `_rand_seed` reseeding it.
  - **seeded**: dedicated per-context `*rand.Rand` seeded from `AILANG_SEED` (via `EffContext.Seed`)
    or the dedicated seeded-source API; **no seed → typed error** at first draw with a fix hint
    pointing to the dedicated path (NOT `rand_seed`).
  - **crypto**: direct `crypto/rand` draws (uniform unbiased int range); entropy failure panics loudly.
- Unit: per-mode source isolation (seeded sequence unperturbed by interleaved os draws); crypto range
  smoke; seeded-without-seed typed error.
- Integration: `AILANG_SEED=42 ailang run` twice → identical sequences (incl. the cross-wrapper case
  `deterministic_roll`).
- **AC (→ Success Criteria 2, 3, 4):** seeded determinism test green; crypto draws from crypto/rand
  (source-inspection hook + statistical smoke); seeded-without-seed → typed error w/ fix hint.

### Day 3 — M3 (trace) + M4 (examples/docs/bookkeeping) + M5 (CI gate)

**M3 — Trace integration** (`internal/trace/schema.go` + collector, ~60 LOC + ~60 test) (~2.5h)
- Additive `Mode string` + `Contract string` `omitempty` fields on `EffectEvent`; populate for moded
  Rand ops from `replay.ContractFor`. `ailang trace` surfaces the label.
- Comparator test in `internal/trace` proving old trace events (no fields) still parse.
- **AC (→ Success Criterion 6):** moded events carry contract label; additive schema verified against
  existing readers (`internal/trace/comparator_test.go` precedent).

**M4 — Example fixes + docs + bookkeeping** (~2.5h)
- **`examples/modal_rand.ail`**: migrate `deterministic_roll`'s seed off `rand_seed(42)` onto the
  dedicated seeded-source path (`AILANG_SEED` / seeded-source API); update the header comment block
  (lines 13-16, no longer "runtime treats all modes identically"); runs deterministically under a
  pinned seed. (→ Success Criterion 7.)
- **`examples/expected_fail/effect_budgets_multi.ail`**: **DO NOT MIGRATE** (premise correction §0 —
  os-mode, not a bug). Instead **repurpose it (or a minimal copy) as the bare-Rand-plus-`rand_seed`
  golden fixture** for M5. (→ Success Criterion 8 is satisfied by "no seeded-mode-via-`rand_seed`
  callers remain" — which, post-M4, is true because only `modal_rand` ever was one.)
- **Bare-Rand golden**: capture pre-change output of a bare-`!{Rand}` program that calls `rand_seed`
  (the `effect_budgets_multi` demo or a dedicated fixture) at HEAD; commit it as the golden gate.
- Docs: `docs/docs/guides/parameterised-effects.md` updated (modes now differ at runtime); teaching
  prompt updated; `implemented/v0_15_0/m-cryptorand.md` header → **Superseded (points here)**.
- **AC (→ Success Criteria 5, 7, 8, 9):** examples pass `verify-examples`; docs/prompt/cryptorand
  header updated; golden fixture committed.

**M5 — Golden gate + CI** (~1.5h)
- Bare-`!{Rand}`+`rand_seed` golden test: pre-change output byte-identical post-change (HARD GATE).
- `make test && make verify-examples && make lint` green.
- **AC (→ Success Criterion 5, 10):** golden identity gate green; full CI green.

---

## 4. Success-Criteria → Milestone map

| Doc Success Criterion | Milestone | Gate type |
|---|---|---|
| Registry exists, Rand 3 + AI 3 rows, lookup tested | M1 | unit |
| Seeded determinism integration test | M2 | integration |
| Crypto mode draws from crypto/rand | M2 | unit + smoke |
| Seeded-without-seed → typed error w/ fix hint | M2 | unit |
| Bare-Rand byte-identical incl. `rand_seed` caller | M4 (fixture) + M5 | **golden HARD GATE** |
| Trace events carry contract label, additive | M3 | comparator |
| `modal_rand.ail` rewritten to dedicated path | M4 | verify-examples |
| No seeded-mode-via-`rand_seed` callers remain | M4 (§0 correction: only `modal_rand` was one) | sweep |
| Guide + prompt updated; `m-cryptorand` → Superseded | M4 | manual/docs |
| `make test && verify-examples && lint` green | M5 | CI |

---

## 5. Risks (updated from doc)

| Risk | Status after spike | Mitigation |
|---|---|---|
| Lowering leaks mode into iface/caches | **RETIRED** — (a) context-threading chosen, no builtin-name change | n/a |
| Seeded source shared across concurrent evaluators | Live | per-`EffContext` source (already per-request cloned, `eval_evaluator.go:170`); mutex like `randMu` |
| Behavioural change to bare Rand sneaks in | Live — HIGH | golden identity gate (M5) is a hard merge gate |
| Trace schema change breaks readers | Live — Med | additive `omitempty` fields + comparator test (M3) |
| **Executor migrates `effect_budgets_multi` by mistake** (doc premise wrong) | **NEW** | §0 correction explicit; that file's `rand_seed` MUST stay |

---

## 6. Open items for the executor / Mark

1. **Seeded-source seed API surface**: the doc says "`AILANG_SEED` and/or a dedicated seeded-source
   API". `EffContext.Seed` (from `AILANG_SEED`) is the low-friction path and covers the eval-harness
   use. Recommend: **`AILANG_SEED` as the seed source for v1.0**; a dedicated `_rand_seeded_seed`
   builtin is deferred unless M4 shows `modal_rand` needs author-level seeding independent of the env.
   The teaching diagnostic's fix-hint points to `AILANG_SEED`. (Deferred-decision-compatible.)
2. **`uuid4` mode** (doc Deferred): leave os-only this sprint; not in scope.
3. Doc line-ref + `effect_budgets_multi` premise corrections (§0) folded into the doc when it moves
   to `implemented/`.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-effect-replay-contracts-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-EFFECT-REPLAY-CONTRACTS.json`
