# M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE: A bounded-unrolling `sat` is not a violation

**Status**: Planned
**Target**: v0.33.2
**Priority**: P0 (verdict inversion — corrupts BOTH arms of the `D-29` KPI and spends harness repair turns telling models to break correct code)
**Estimated**: 2–3 days (2 milestones, **reader-before-writer**, each independently committable)
**Dependencies**: None hard. **Coordinate textually** with
[m-contract-verification-coverage](m-contract-verification-coverage.md) (iteration 255,
currently parked after quorum round 2) — both docs edit `isVerifiedSuccess` and the
`verify_*` counter plumbing; whichever lands second rebases a ~6-line predicate edit.
**Author**: design-doc-creator role, mission iteration 259 (2026-08-23, at `origin/dev` = `bc3f80884`)
**Planner-Lane**: opus-required (touches the KPI predicate and a solver-facing classification)
**Revision**: r2 — round-1 quorum verdict was **blocked** (2/2 external reviewers present,
no N−1 degrade). Both objections applied; every revision number re-measured first-party
(V23–V27). See [Revision history](#revision-history-r1--r2).

---

## Problem Statement

AILANG's contract verifier handles recursion by **bounded unrolling**
(`internal/smt/unroll.go`): a recursive function `f` is expanded into a chain
`f_N → f_{N-1} → … → f_1 → f_0`, where `f_0` is an **uninterpreted stub**
(`declare-fun f_0 …`, `unroll.go:44`) that havocs everything beyond the unrolling
frontier. The verification condition negates the `ensures` clause and asks Z3 for a model.

That encoding has a built-in asymmetry the status ladder ignores:

- **`unsat` is sound**: the property holds under *every* interpretation of the stub,
  including the real function — so `verified` is honest.
- **`sat` is NOT a violation**: Z3 may satisfy the negated obligation by *inventing an
  interpretation for the stub* — an execution no real run can produce.

Today both ladders (`cmd/ailang/ai_check.go:404-407`, `cmd/ailang/verify.go:428-431`) map
every `sat` to `status: "counterexample"`, so **a correct recursive implementation is graded
a verification FAILURE**. Measured (V1): in one file with the identical clause
`ensures { result >= 0 }`, non-recursive-true `aOk` → `verified`, non-recursive-false
`bBad` → `counterexample` (genuine), and recursive-true `cRec` (a list length, whose result
is unconditionally ≥ 0) → `counterexample` with a "witness" that references the havoc'd stub
`cRec_0`. Raising the depth does not help: the artifact witness just grows to length N at
depth N (V2) — for any finite bound Z3 picks an input that crosses it.

The tool even knows it is in the bounded regime and already says so — **on the branch where
it doesn't matter**. `vr.BoundedDepth` is populated before the status switch in both ladders
(`ai_check.go:399-400`, `verify.go:418-419`), and `verify_print.go:32-33` prints
`✓ VERIFIED (bounded: depth N)` on success. Both `BoundedDepth` mentions in
`verify_print.go` are in the verified branch (V6); the `counterexample` branch
(`verify_print.go:38-46`) prints `✗ VIOLATION` plus the model with no bounded
qualification. The caveat is dropped exactly where it is the difference between
"your code is wrong" and "I could not tell".

**This is the third face of one taxonomy defect.** One concept — *"the verifier could not
discharge this obligation"* — is currently reported THREE ways depending on which code path
hit the limitation: `error` (unencodable pattern types), `skipped` (unencodable builtins),
and — this doc's surface — `counterexample` (bounded-unrolling artifacts). Only `skipped`
is honest. The fix must therefore be a **taxonomy decision** (a first-class "attempted,
undecided" outcome), not a patch on one branch — but this doc fixes only the surface that
*inverts a verdict*; see [Relationship](#relationship-to-m-verify-unencodable-reported-as-error).

### Impact (all verified, see [Verification Log](#verification-log))

1. **27% of the frozen `v1.0` cohort is mis-labeled by this defect — uniformly across
   model families.** Across the 30 banked v1.0 runs, `verify_counterexample` distributes
   **{0: 22, 1: 8}** (V25): 8 of 30 runs are graded verification-FAILED on a
   counterexample, and all 8 land on exactly the two recursive-ADT benchmarks —
   `contract_sorted_merge` ×4 (`claude-haiku-4-5`, `claude-sonnet-4-6`,
   `opencode-or-deepseek-v4-flash`, `opencode-or-glm-5-2`) and `contract_bst_validate`
   ×4 (`claude-haiku-4-5`, `claude-sonnet-4-6`, `gpt5-6-luna`, `opencode-or-glm-5-2`) —
   every one with `verify_verified = 0`. A hit uniform across 4 model families on
   precisely the two recursive-ADT benchmarks is the signature of a TOOL defect, not a
   model defect (the same uniformity argument iteration 254 used for the skip class).
   `isVerifiedSuccess` (`cost_per_verified_success.go:95-105`) rejects all 8 on
   `VerifyCounterex == 0` (V10). Note what this does NOT mean: the computed `D-29`
   number itself does not move under this fix — see [Q6](#q6--frozen-v10-cohort-blast-radius-is-real-and-large-and-the-kpi-number-cannot-move-under-this-fix).
2. **The harness spends repair turns breaking correct code.** On
   `Verify.Counterexample > 0`, `internal/eval_harness/repair.go:79-83` sets
   `errCode = VERIFY_COUNTEREXAMPLE` and feeds `FormatZ3RepairHint(rawJSON)` back to the
   model (V11), while `agent_prompt.go:595` instructs *"If Z3 reports COUNTEREXAMPLE: fix
   your logic using the counterexample inputs"* (V12). For a recursive artifact, those
   "counterexample inputs" are not violating inputs at all.
3. **It fires at the harness default.** `-verify-recursive-depth` defaults to **2** in both
   commands (V4), and `repair.go` calls `RunAICheck` with no depth flag (V11) — no opt-in
   is required to hit this.
4. **It is live in the frozen `v1.0` cohort.** `benchmarks/contract_sorted_merge.yml`
   ships `export pure func sLength(xs: SList) -> int ensures { result >= 0 }` (V16) — a
   recursive ADT length with exactly the measured clause shape.
5. **The exit code follows the lie.** `ai-check` exits 1 on `counterexample > 0` (V8), and
   `verify` exits 1 unconditionally on `counterexample > 0` (V9), so agent convergence
   loops treat correct code as non-converged.

---

## Goals

**Primary Goal:** a `sat` obtained under bounded unrolling whose model depends on the
unrolling frontier is never reported as a violation of the user's contract — it becomes a
first-class, honestly-worded **`inconclusive`** outcome at every layer.

**Success Metrics:**
- `adt.ail` fixture (recursive `sLength`, true clause): `ai-check` goes from
  `rc=1, counterexample=1` (measured baseline, V18) to `rc=0, counterexample=0, inconclusive=1`.
- `discrim.ail` fixture: genuine `bBad` violation still reported (`counterexample=1`,
  `rc=1`) with its concrete model — no loss of genuine-counterexample reporting on
  non-recursive functions.
- Zero repair turns triggered by artifact counterexamples
  (`VERIFY_COUNTEREXAMPLE` requires a real `counterexample > 0`).
- `isVerifiedSuccess` never counts an undecided obligation as a verified success
  (`VerifyInconclusive == 0` required) — and, via reader-before-writer landing order,
  never transiently inflates during the rollout.
- The 8 mis-labeled v1.0 runs (27% of the frozen cohort) reclassify
  `counterexample → inconclusive` under the fixed binary (AC-12). The KPI number itself
  is **bit-identical by construction** (Q6): this fix's declared value is honest labels,
  ended repair-turn waste, and unblocking `D-29`'s second clause — NOT a KPI move.

## Non-Goals

- **No change to the `verified` surface.** `unsat` under havoc is sound (the real function
  is one interpretation of the stub); `✓ VERIFIED (bounded: depth N)` wording stays as-is.
- **No witness replay in this doc.** Concretely re-executing the model's inputs to promote
  `inconclusive` back to a proven violation is the decisive second signal — deliberately
  deferred (see [Future Work](#future-work)) because the conservative direction chosen here
  never inverts a verdict without it.
- **No induction-hypothesis encoding.** Assuming the contract at the frontier stub
  (Dafny-style assume-guarantee) would make `cRec`/`sLength` verify outright, but it is an
  encoder-semantics change that alters the `verified` surface and introduces quantified
  assertions; it needs its own design (see [Future Work](#future-work)).
- **No KPI restatement.** This doc states what re-derivation is needed (Q6) but publishes
  no new number.

## Relationship to `m-verify-unencodable-reported-as-error`

A sibling queue row covers the same underlying taxonomy defect on a different surface: an
encoding refusal reported as `error` (e.g.
`ailang ai-check examples/runnable/contracts/shapes_verify.ail` → `3 verified, 1 errors`,
`cannot encode function body: match pattern: unsupported pattern type *core.TuplePattern`)
where the comparable unencodable-builtin case is a structured `skipped`. That surface is
**explicitly OUT of scope here and is not absorbed**: it is a classification/UX question
with no verdict inversion, while this doc's surface inverts a verdict and corrupts a
published KPI. The two have different correctness bars; bundling different bars is what
cost a sibling doc five quorum rounds. This doc's `inconclusive` status and partition
invariant (INV-1) are designed so that row can later route "unencodable" into `skipped`
without touching this mechanism.

---

## Open Design Questions — resolved

The six questions from the task directive, answered with the measurements that decided them.

### Q1 — What status does a bounded-unrolling `sat` carry? → a NEW first-class `inconclusive`

Reusing `unknown` is rejected: both ladders route `StatusUnknown` to `section.Errors++`
(V5), so the run would move from "counterexample" to "error" and `isVerifiedSuccess`
(`VerifyErrors == 0`, V10) rejects it anyway — and `unknown` already has a meaning ("the
solver itself returned unknown/timeout") that this outcome does not share: here the solver
answered `sat`, and *our encoding* is what can't vouch for it. Reusing `skipped` is
rejected: `skipped` means "never attempted / nothing to attempt", this obligation WAS
attempted and partially explored — and iteration 255's doc is busy un-conflating `skipped`;
re-overloading it would undo that. The token `inconclusive` appears only in comments/log
prose in the Go tree, never as a status wire value (V13) — it is free.

**Per-consumer after-state** (the directive requires this explicitly):

| Consumer | Today (artifact sat) | After |
|---|---|---|
| `ai-check` JSON | `status:"counterexample"`, `counterexample:1` | `status:"inconclusive"` + `reason`, new top-level `inconclusive:1` counter; model omitted (kept under `--verbose` via `verify`) |
| `aiCheckExitCode` (`ai_check.go:180-185`) | rc=1 | rc=0 — same contract as `skipped`: "not proved" is not "disproved" (V8) |
| `verify` exit (`verify.go:453-459`) | rc=1 | rc=0 non-strict; `--strict` exits 1 (`inconclusive > 0` joins `skipped > 0 || errCount > 0`) |
| `verify_print.go` | `✗ VIOLATION` + model | `? INCONCLUSIVE (bounded: depth N)` + reason ([Q5](#q5--what-does-the-user-see)) |
| `eval_harness` `VerifyOk` (`verify.go:142,155`) | false (`Counterexample>0`) | true — no violation claimed; `Inconclusive` carried in its own new field |
| `repair.go:79` | `VERIFY_COUNTEREXAMPLE` + Z3 hint fed to model | no verify-driven repair (trigger stays `Counterexample > 0`, now honest) |
| `agent_prompt.go:595` | model told to "fix" correct code | new line: INCONCLUSIVE means the verifier could not decide — do not change logic because of it |
| `isVerifiedSuccess` (V10) | run rejected (counterex>0) | run still rejected — via new `VerifyInconclusive == 0` conjunct. Undecided ≠ verified; the KPI does not silently inflate |
| `isVerificationFailure` (V10) | true (counterex>0) | true — `VerifyInconclusive > 0` joins the attempted-but-not-discharged disjunction, same class as `skipped` |

### Q2 — Soundness direction: err toward `inconclusive`, and neither error mode can produce a false `verified`

The classifier's two possible mistakes are asymmetric in a way the consumers make concrete:

- **False violation** (artifact reported as counterexample — today's behavior) *inverts* a
  verdict: correct code is graded wrong, both `D-29` KPI arms are corrupted in an
  irreparable direction, and the harness actively coaches models to damage correct code
  (V11/V12). All three harms are measured and live.
- **False undecided** (genuine bug reported as inconclusive) ships no wrong claim: the
  verdict degrades to the same trust level as `skipped` — "could not tell" — and runtime/
  stdout grading still stands between the bug and a pass. Nothing published becomes false.

Additionally, measurement shows today's "genuine reporting" on recursive functions is worth
less than it appears: in the Q3 experiment (V3), Z3's witness for the genuinely-buggy
`lenAtLeast1` was `(SCons 3 (SCons 2 SNil))` — an input that **satisfies** the clause
(length 2 ≥ 1); the genuine witness `SNil` existed and was not chosen. So the current
counterexample surface on recursive functions is unreliable in both verdict *and* witness.

Therefore: **the classifier must err toward `inconclusive`.** Can it produce either error?
It CAN produce false-undecided (measured: `sumNonNeg`'s genuine bug is a bounded-unrolled
sat → classified inconclusive). It CANNOT produce a false violation (a sat on a
bounded-unrolled obligation is NEVER reported as counterexample — r2's structural
predicate has no model-dependent branch that could leak one) and CANNOT touch `verified`
(`sat` never maps there; `unsat` is unchanged).

### Q3 — The frontier-binding discriminator, stressed by measurement: **dropped entirely** (r2)

This was the load-bearing experiment. Probe `q3.ail` (V3): three recursive functions with
**genuinely false** `ensures`, run at depth 2:

| function | genuine violation | model carries frontier binding? | witness genuine? |
|---|---|---|---|
| `lenAtLeast1` (length, `ensures result >= 1`) | yes — `SNil`, within bound | **YES** (`lenAtLeast1_0`) | **NO** — witness satisfies the clause |
| `sumNonNeg` (sum, `ensures result >= 0`) | yes — any negative element | **YES** (`sumNonNeg_0`) | yes (happens to be `-1` total) |
| `lenAtMost2` (length, `ensures result <= 2`) | yes — only beyond the bound | **YES** (`lenAtMost2_0`) | no (length-2 input) |

**Every genuine counterexample on a recursive function also carried the frontier binding**,
and `result` was frontier-dependent in all three. The structural reason: at depth k,
`f_k` fully evaluates only inputs of recursion depth **< k**; an input of depth exactly k
already reaches the uninterpreted `f_0` (e.g. `len_2(SCons(a, SCons(b, SNil))) =
2 + len_0(SNil)` — the *base case* lands on the stub). Z3 is free to pick such inputs and
measurably prefers them, even when a concrete within-bound witness exists (`lenAtLeast1`).

So "presence of a frontier binding" cannot separate genuine from artifact. The r1 draft
still used its *absence* to keep a `counterexample` branch for unrolled functions; the
quorum correctly rejected that (gpt5-6-sol): absence of a symbol from Z3's parsed/raw
model is not a dependency analysis — solvers may omit interpretations they consider
irrelevant, and `parseDefineFun` measurably mangles function-sorted entries (V1) — so the
absence branch was the one remaining path that could emit exactly the false violation
this doc exists to eliminate. And a depth sweep shows that branch is **unreachable in
every case either side could construct**: at depths 1, 2, 3, 5, 8 and 10, all three
genuinely-buggy `q3.ail` functions carry their frontier binding in **18 of 18 readings**
(V23); combined with the artifact probes (`cRec`, `sLength` — V1/V2), there are **zero**
observed recursive `sat` models without one. The branch bought nothing and cost the
central soundness claim.

**Resolution: the model-inspection discriminator is dropped entirely.** The classifier is
the conservative structural predicate both ladders already compute to set `BoundedDepth`:

```
smt.IsRecursiveFunc(innerBody, funcName) && recursiveDepth > 0 && solver says sat
    ⇒ inconclusive
```

This is behaviourally identical to the discriminator on every measured case (18/18 sweep
plus all artifact probes), sound by construction (no model inspection — nothing for the
solver to omit or the parser to mangle), and strictly less code: no `EncodeResult`
change, **no `internal/smt` code change at all**. Non-recursive obligations are
untouched: `bBad`'s genuine counterexample — two concrete bindings, no frontier symbol
(V1) — still reports as `counterexample`.

**The declared cost, stated plainly (this is the doc's main limitation):** a **genuine**
counterexample on a recursive function is also classified `inconclusive`. This is real
and measured — `sumNonNeg`'s model binds `$p_xs = (SCons 11797 (SCons (- 11798) SNil))`,
whose sum is −1: an actual violating witness that the fix will now report as undecided
(V3). Per Q2 this errs in the survivable direction: no false `verified` is possible,
`verify --strict` still exits non-zero on `inconclusive` (no bug ships silently through
a strict gate), and runtime/stdout grading still stands. The signal that recovers these
cases is **witness replay** (concretely evaluate the model's `$p_*` inputs against the
real function and check the `ensures`), deferred to [Future Work](#future-work) — where
the frontier-symbol machinery is demoted to a replay *input*, never a verdict gate.

### Q4 — Where the decision lives: ONE shared ladder in `cmd/ailang`; `internal/smt` unchanged (r2)

The classification needs exactly two facts, and after r2 both are already computed in
`cmd/ailang` at the precise point of the status switch: `smt.IsRecursiveFunc(innerBody,
funcName)` and the effective `recursiveDepth` — the same guard that sets
`vr.BoundedDepth` today (`ai_check.go:399-400`, `verify.go:418-419`). So:

- `internal/smt`: **no code change.** The r1 `EncodeResult.FrontierSymbols` /
  `SatDependsOnFrontier` machinery is deleted per quorum objection 1 (model inspection
  must not gate a verdict); V14/V15 are retained in the log as background for the
  future witness-replay design only. `internal/smt` stays on the conflict surface as a
  READ dependency: the shared helper must remain exhaustive over `SolverStatus`
  (`solver.go:24-31`).
- `cmd/ailang`: the status switch exists twice, near-identically
  (`ai_check.go:370-420`, `verify.go:388-435`, both mapping `StatusUnknown` →
  errors — V5). Both switches are replaced by ONE shared helper in a new
  `cmd/ailang/verify_status.go` — `applySolveOutcome(solve *smt.SolverResult,
  boundedDepth int) (status string, model []smt.ModelBinding, reason string)` — so the
  ladders cannot drift; an exhaustiveness test over `SolverStatus` plus a cross-command
  fixture-equality test lock it (see [Mutation Table](#mutation-table)).

`Solve()` itself is untouched: it reports raw solver facts (`sat`/`unsat`) and has no
access to encode metadata; classification is a policy on top of it.

### Q5 — What does the user see?

The verified branch already models the honest wording (V6). The new branch mirrors it:

```
? INCONCLUSIVE (bounded: depth 2) sLength
    No counterexample within depth 2. The solver's candidate model depends on
    the unrolling frontier (sLength_0) and is not a real input.
    Note: raising --verify-recursive-depth cannot decide this obligation
    (the candidate model grows with the depth).
```

The model is NOT printed by default (measured: it is garbage — a `let`-expression over the
stub, V1) but remains available under `ailang verify --verbose` for debugging. JSON carries
`status: "inconclusive"` plus a machine `reason` (e.g.
`bounded unrolling exhausted at depth 2; model depends on frontier symbol "sLength_0"`).
The "raising the depth cannot decide this" sentence is load-bearing: V2 measured the
witness growing from length 2 to length 4 as depth went 2 → 4, so depth-tuning advice would
send users on a treadmill. `agent_prompt.go` gains the matching one-liner for agents.

### Q6 — Frozen `v1.0` cohort: blast radius is real and large, and the KPI number cannot move under this fix

Measured first-party at HEAD (V24–V27):

- Live KPI: **30 runs, 29 passed, 3 verified successes, 26 verification failures,
  `known_cost_usd` $2.3334561216, `cost_per_verified_success_usd`
  $0.7778187071999999** (`ailang chains stats --cost-per-verified-success
  --baseline v1.0 --json`, V24), matching the banked
  `eval_results/baseline_v1_0/cost_per_verified_success_v1.0.json` field-for-field.
- `verify_counterexample` across the 30 banked run JSONs: **{0: 22, 1: 8}** — 8 runs
  (27%) graded verification-FAILED on a counterexample, all 8 on the two recursive-ADT
  benchmarks (`contract_sorted_merge` ×4, `contract_bst_validate` ×4), across 4 model
  families, every one with `verify_verified = 0` (V25). `contract_sorted_merge` ships
  the exact reproduced clause (V16); `contract_bst_validate` is recursive over
  `Tree = Leaf | Node(Tree, int, Tree)` with the spec instructing recursion into
  subtrees (V26).

**Three rulings follow (r2 — replaces the r1 "corrected companion derivation", which was
arithmetically vacuous):**

1. **No re-banking, and no "corrected KPI" either — because the number cannot move.** A
   run flipping `counterexample → inconclusive` still fails `isVerifiedSuccess`: M1's
   own predicate adds `&& a.VerifyInconclusive == 0` (this doc's Q1 table already says
   "run still rejected"). `verified_successes` stays 3, `known_cost_usd` stays
   $2.3334561216, so `$0.7778187072` is **bit-identical before and after this fix**. Any
   deliverable that recomputes it is vacuous and has been deleted (quorum objection 2).
2. **What the fix is actually worth on this cohort:** (a) the verification-failure label
   on those 8 runs stops asserting a violation the tool did not find; (b)
   `repair.go:79-83` stops firing `VERIFY_COUNTEREXAMPLE` on them — the harness stops
   spending repair turns telling models to break correct code (V11/V12); (c) it unblocks
   `D-29`'s ratified second clause — adding `ensures` to the recursive benchmark
   functions is TODAY guaranteed to make the KPI *worse*, because every recursive
   `ensures` mints a spurious counterexample.
3. **The only lever that could ever move the number is a policy question, filed — not
   answered — here:** see [Open for human ruling — D-32](#open-for-human-ruling--d-32).
   AC-12 is accordingly a status-flip confirmation on the 8 affected runs (their banked
   `code` fields are populated — V27), never a KPI derivation.

---

## Open for human ruling — D-32

Should `inconclusive` be **exempted** from the effective KPI arm, the way Mark's `D-29`
ruling exempts the `no ensures clause` skips from the `$0.2121` effective figure? That is
the only decision that could ever move the cohort number; it is exactly `D-29`-shaped (a
denominator-predicate ruling on the same axis); and it is a human call about what the KPI
should claim, not an engineering call. **This doc deliberately does not decide it and does
not depend on it** — every milestone, predicate, and acceptance criterion here is correct
under either ruling. The coordinator files it in the mission ledger as `D-32`; no
milestone in this doc covers it.

---

## Verification Log

All commands run 2026-08-23. Repo facts read from `origin/dev` = `bc3f80884` via
`git show`/`git grep` in `/Users/voightkampff/dev/sunholo-data/ailang`; probe runs used the
stamped scratch binary (`/tmp/i259_bin/ailang --version` → `AILANG v0.33.1-223-gbc3f80884`,
matching `git describe` at the pin — V17). Probe `.ail` files were all accepted by the
binary's own check phase (`"check": {"passed": true}` in every run below), satisfying the
verify-snippets gate.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | Same-clause discrimination + model shapes | `PATH="/tmp/i259_bin:$PATH" ailang ai-check -verify-recursive-depth 2 /tmp/i259_probe/discrim.ail` | `aOk` `verified`; `bBad` `counterexample`, model = `$p_x=(- 1)`, `result=$p_x` (concrete, no frontier symbol); `cRec` `counterexample`, model contains binding named `cRec_0` (sort mangled to `SList))`, value `Int`) and `result = (let ((a!1 (+ 1 (cRec_0 (SCons_1 (SCons_1 $p_xs)))))`; `bounded_depth: 2`; exit rc=1 |
| V2 | Artifact witness grows with depth; control unaffected | same binary, `-verify-recursive-depth 2` then `4` on `/tmp/i259_probe/adt.ail` | depth 2: `sLength` counterexample, `$p_xs=(SCons 3 (SCons 2 SNil))` (len 2); depth 4: counterexample, `$p_xs=(SCons 2 (SCons 4 (SCons 5 (SCons 3 SNil))))` (len 4); `headOr0` `verified` at both depths |
| V3 | **Q3 experiment**: genuine recursive violations also carry frontier bindings | wrote `/tmp/i259_probe/q3.ail` (3 recursive funcs, genuinely false clauses; `check.passed=true`), ran `-verify-recursive-depth 2` | all 3 `counterexample` with a `<fn>_0` model binding and frontier-dependent `result`; `lenAtLeast1` witness `(SCons 3 (SCons 2 SNil))` — length 2 **satisfies** `result >= 1`, i.e. a non-violating "witness" for a genuinely buggy function; rc=1 |
| V4 | Depth defaults to 2 in both commands | `git grep -n "verify-recursive-depth" origin/dev -- cmd/ailang/` | `ai_check.go:49` and `verify.go:26`: `fs.Int("verify-recursive-depth", 2, …)`; also `verify.go:308-309` has a per-function `@verify(depth: N)` override |
| V5 | Two duplicated ladders; `StatusUnknown` → errors in both | `git show origin/dev:cmd/ailang/ai_check.go \| sed -n '340,440p'`; same for `verify.go` 370–450 | both switch on `solveResult.Status`; both map `StatusCounterexample` → `"counterexample"` + `vr.Model`; both map `StatusUnknown` → `"unknown"` + `section.Errors++`/`errCount++` |
| V6 | Bounded caveat exists ONLY on the verified branch | `git show origin/dev:cmd/ailang/verify_print.go \| grep -c BoundedDepth` and read of lines 29–46 | count = `2`, both inside `case "verified"` (`✓ VERIFIED (bounded: depth %d)`); `case "counterexample"` prints `✗ VIOLATION` + raw model, no qualification |
| V7 | Frontier stub is `<fn>_0`, uninterpreted | `git show origin/dev:internal/smt/unroll.go` | line 44: `level0Name := fmt.Sprintf("%s_0", cfg.FuncName)`; emitted as `(declare-fun %s (…) %s)`; levels 1..N are `define-fun` with self-calls replaced |
| V8 | `ai-check` exit contract | read `ai_check.go:163-185` (origin/dev) | `aiCheckExitCode`: `!check.Passed \|\| verify.Counterexample > 0 \|\| verify.Errors > 0` → 1; documented `skipped-only -> 0  (a skip is "not proved", not "disproved")` |
| V9 | `verify` exit contract | read `verify.go:453-459` (origin/dev) | `if counterexample > 0 { os.Exit(1) }`; `--strict` additionally on `skipped > 0 \|\| errCount > 0` |
| V10 | KPI predicate rejects on counterexample; failure predicate counts it | read `internal/observatory/cost_per_verified_success.go:95-125` (origin/dev) | `isVerifiedSuccess` requires `VerifyVerified > 0 && VerifyCounterex == 0 && VerifySkipped == 0 && VerifyErrors == 0`; `isVerificationFailure` = pass && (`VerifyCounterex > 0 \|\| VerifySkipped > 0 \|\| VerifyErrors > 0`) |
| V11 | Harness repair trigger + no depth flag | read `internal/eval_harness/repair.go:74-86` (origin/dev) | `RunAICheck("", …+"/benchmark/solution.ail", r.verifyTimeout)` (no depth flag → default 2); `if verifyResult.Verify.Counterexample > 0 { errCode = VERIFY_COUNTEREXAMPLE; hint = FormatZ3RepairHint(rawJSON) … }` |
| V12 | Agent prompt coaches on counterexamples | read `internal/eval_harness/agent_prompt.go:588-600` (origin/dev) | `- If Z3 reports COUNTEREXAMPLE: fix your logic using the counterexample inputs`; `- If Z3 reports VERIFIED: your function is correct for ALL inputs` |
| V13 | `"inconclusive"` unallocated as a wire value (negative + control, same tree) | `git grep -rin -c "inconclusive" origin/dev -- cmd internal` (control in same call: `git grep -c "counterexample" origin/dev -- internal/smt/solver.go` → `2`) | 9 files hit, all comments/log prose (`solver_timeout_test.go` t.Logf, `eval_censored_test.go` comment, riglock comments, eval_analysis censored-verdict domain); none is a verify per-function `status` value. Verify status namespace today: `verified, counterexample, skipped, error, unknown` (`verify.go:466` comment) + planned `not_applicable` (sibling doc) |
| V14 | `EncodeResult` has no frontier field *(background after r2 — no longer gates the design; retained for the witness-replay follow-up)* | `git show origin/dev:internal/smt/codegen.go \| sed -n '/type EncodeResult/,/^}/p'` | exactly four fields: `SMTLib`, `Declarations`, `Assertions`, `BodyExpr` — the full struct is the positive control for the negative claim |
| V15 | All stub emission sites; mutual path emits none (same-call control) *(background after r2, as V14)* | `git grep -n "declare-fun" origin/dev -- internal/smt/codegen_mutual.go internal/smt/list_unroll.go`; plus `git grep -n "TopLevelName" origin/dev -- internal/smt/` | `list_unroll.go:48,82,115` hit (`_list_reverse_0`, `_list_take_0`, `_list_drop_0`); `codegen_mutual.go` zero hits in the same call; `hof_inline.go:371` builds `UnrollResult` for recursive HOF callees (same `%s_0` mechanism) |
| V16 | Frozen cohort ships the vulnerable shape | `git show origin/dev:benchmarks/contract_sorted_merge.yml \| sed -n '18,30p'` | `export pure func sLength(xs: SList) -> int` / `ensures { result >= 0 }` |
| V17 | Scratch binary matches origin/dev | `/tmp/i259_bin/ailang --version`; `git rev-parse HEAD` in the bc3f80884 pin worktree | `AILANG v0.33.1-223-gbc3f80884`; `bc3f80884132d8c738cdd40835a4557f396abaf4` |
| V18 | Acceptance baseline for the fix gate (RED at base BY DESIGN) | `PATH="/tmp/i259_bin:$PATH" ailang ai-check -verify-recursive-depth 2 /tmp/i259_probe/adt.ail; echo rc=$?` | `rc=1`; counters `{verified: 1, counterexample: 1, skipped: 0, errors: 0}`; `sLength=counterexample`, `headOr0=verified` |
| V19 | Green gates are green at base (pin worktree = origin/dev) | `go build ./cmd/ailang && go build ./internal/smt/...`; `go test ./internal/smt/ ./internal/eval_harness/ ./internal/observatory/ ./cmd/ailang/` | both builds rc=0; all four packages `ok` (smt 8.4s, eval_harness 16.8s, observatory 1.9s, cmd/ailang 31.3s) |
| V20 | Harness parses ONLY flat counters (skew hazard is real) | `git grep -n "Counterexample" origin/dev -- internal/eval_harness/verify.go` + read of lines 35–46; corroborated by sibling doc V9 | `AICheckVerifyResult` = `{Available; Verified; Counterexample; Skipped; Errors}` — no `Inconclusive`, no `Results`; `VerifyOk = Available && Counterexample == 0 && Errors == 0` (lines 142, 155) |
| V21 | Drift: depth override exists in `verify` only (positive + negative, same call) | `git grep -c "@verify" origin/dev -- cmd/ailang/verify.go cmd/ailang/ai_check.go` | `verify.go:1`; `ai_check.go` absent from output (zero matches) — evidence the two ladders already drift, motivating the shared helper |
| V22 | `cmd/ailang/testdata/` exists for fixtures | `git ls-tree origin/dev cmd/ailang/testdata/` | `debug_ast_simple.ail`, `.golden`, `ext_registry_gen/` — convention confirmed |
| V23 | **r2 sweep**: frontier binding present in EVERY recursive sat, at every depth | `for d in 1 2 3 5 8 10; do PATH="/tmp/i259_bin:$PATH" ailang ai-check -verify-recursive-depth $d /tmp/i259_probe/q3.ail; done`, extracting model binding names per function | **18 of 18 readings** (3 genuinely-buggy recursive functions × 6 depths) are `counterexample` with a `<fn>_0` frontier binding (`lenAtLeast1_0`, `lenAtMost2_0`, `sumNonNeg_0`); zero recursive sats without one across all probes this doc measured |
| V24 | Live frozen-cohort KPI (re-measured, not transcribed) | `PATH="/tmp/i259_bin:$PATH" ailang chains stats --cost-per-verified-success --baseline v1.0 --json` (run 2026-08-23T12:48Z) | `total_runs: 30, passed_runs: 29, verified_successes: 3, unverified_passes: 0, verification_failures: 26, known_cost_usd: 2.3334561216, cost_per_verified_success_usd: 0.7778187071999999, available: true` — field-for-field equal to banked `eval_results/baseline_v1_0/cost_per_verified_success_v1.0.json` (generated 2026-08-22) |
| V25 | 8-of-30 blast radius, concentrated + uniform | python over `eval_results/baseline_v1_0/agent/*.json`: count `verify_counterexample`, list rows > 0 | `files: 30`; distribution `{0: 22, 1: 8}`; the 8: `contract_bst_validate` × {claude-haiku-4-5, claude-sonnet-4-6, gpt5-6-luna, opencode-or-glm-5-2}, `contract_sorted_merge` × {claude-haiku-4-5, claude-sonnet-4-6, opencode-or-deepseek-v4-flash, opencode-or-glm-5-2}; all 8 have `verify_counterexample = 1` and `verify_verified = 0` |
| V26 | `contract_bst_validate` is the recursive-ADT shape | `git show origin/dev:benchmarks/contract_bst_validate.yml \| grep -n "type Tree\|isBST\|ensures\|recursive"` | `export type Tree = Leaf \| Node(Tree, int, Tree)` (line 14); `ensures { result >= 0 }` (line 21); `isBST(t: Tree, lo: int, hi: int)` (line 24); "when recursing into subtrees in isBST" (line 41) |
| V27 | Banked rows carry the solution code (AC-12 implementable) | python: key list + `len(code)` on `contract_sorted_merge_ailang_claude-haiku-4-5_1787428845.json` | keys include `code` and `verify_json`; sampled `code` length 2395 bytes (non-empty) |
| V28 | **`smt.IsRecursiveFunc` exists, is EXPORTED, and is already called from both `cmd/ailang` ladders** — the premise the r2 "zero `internal/smt` changes" claim rests on (added at controller carve-out, answering `gemini-3-1-pro` r2) | `grep -rn "func IsRecursiveFunc" internal/smt/` ; `grep -rn "smt.IsRecursiveFunc" cmd/ --include='*.go'` ; negative control `smt.IsRecursiveFuncZZZ` ; positive control `smt.Solve` | `internal/smt/encodable.go:153: func IsRecursiveFunc(body core.CoreExpr, funcName string) bool` (capitalised ⇒ exported, signature exact). Call sites **already present**: `cmd/ailang/ai_check.go:399` and `cmd/ailang/verify.go:418` — the very `BoundedDepth` guard lines this design reuses. Negative control **0**, positive control **2**, so the greps fire. ⇒ no new export, no new import, no `internal/smt` change is required by the classifier. |

---

## Solution Design

### Overview

Classify, at the one seam both ladders share, whether a `sat` can be vouched for — and
give the un-vouchable case its own honest, first-class outcome.

```
Solve() == sat
   │
   ├─ obligation NOT bounded-unrolled                        → counterexample (unchanged)
   │    (BoundedDepth guard false: non-recursive, or depth 0)
   └─ obligation bounded-unrolled                            → inconclusive  (NEW)
        (smt.IsRecursiveFunc && recursiveDepth > 0 — the exact
         guard both ladders already use to set vr.BoundedDepth)
        status "inconclusive", reason "bounded unrolling exhausted at depth N; …",
        model suppressed from default output, BoundedDepth carried as today
```

`unsat` → `verified` and solver-level `unknown`/`error` are untouched.

### Classification decision table

| Solver says | Bounded-unrolled? | Status | Rationale |
|---|---|---|---|
| `unsat` | any | `verified` (+ bounded caveat as today) | sound under havoc — real fn is one stub interpretation |
| `sat` | no | `counterexample` | fully concrete obligation; the model is a real witness (measured: `bBad`) |
| `sat` | **yes** | **`inconclusive`** | a sat under havoc carries no vouchable verdict, and model inspection cannot upgrade it (Q3: 18/18 sweep, zero frontier-free recursive sats ever observed) — conservative direction per Q2 |
| `unknown` | any | `unknown` (unchanged) | solver-side inability, distinct cause |
| solver error | any | `error` (unchanged) | infrastructure failure |

### Landing order (reader-before-writer — the skew is live, not theoretical)

`RunAICheck` resolves the `ailang` child via PATH (`D-30`), so harness and verifier are
independently versioned. If the **emitter** landed first, a new binary under an old harness
would report `counterexample: 0` for artifact sats while the old parser (V20) has no
`inconclusive` field — `VerifyOk` flips true and `isVerifiedSuccess` could count an
undecided run as a verified success: **silent KPI inflation**. Readers therefore land
first; they read a field that is always 0 until M2 ships, which is harmless.

**M1 — readers.**
- `internal/eval_harness/verify.go`: `AICheckVerifyResult` gains
  `Inconclusive int json:"inconclusive"`; `PopulateVerifyMetrics`/`populateVerifyFields`
  copy it.
- `internal/eval_harness/metrics.go` + `agent_runner.go`: `VerifyInconclusive int
  json:"verify_inconclusive"`.
- `internal/observatory/cost_per_verified_success.go`: `EvalAssessment` gains
  `VerifyInconclusive`; `isVerifiedSuccess` gains `&& a.VerifyInconclusive == 0`;
  `isVerificationFailure` gains `|| a.VerifyInconclusive > 0`.
- `internal/eval_harness/repair.go`: no code change needed (trigger is already
  `Counterexample > 0`); add the regression test that it must NOT trigger on
  inconclusive-only JSON (MUT-REPAIR-WIDEN).

**M2 — mechanism + emitters.**
- `internal/smt`: **no code change** (r2; see Q4). Classification keys on the
  `BoundedDepth` guard both ladders already evaluate.
- `cmd/ailang/verify_status.go` (NEW): shared `applySolveOutcome` helper; both
  `ai_check.go` and `verify.go` switches replaced by calls to it; both section structs
  gain an `inconclusive` counter (JSON `"inconclusive"`), and the summary lines include it.
- Exit codes: `aiCheckExitCode` — inconclusive-only stays 0 (documented alongside the
  existing `skipped-only -> 0` contract, V8); `verify` — non-strict 0, `--strict` 1.
- `cmd/ailang/verify_print.go`: new `case "inconclusive"` branch per Q5.
- `internal/eval_harness/agent_prompt.go`: one line teaching INCONCLUSIVE
  (verified against the live prompt-serving path, per prompt-manager conventions).
- Fixtures: copy `/tmp/i259_probe/{discrim,adt,q3}.ail` into
  `cmd/ailang/testdata/verify_bounded/` (V22) — `/tmp` paths must not be referenced by
  tests.
- Flip confirmation on the 8 affected banked runs (AC-12): statuses only, no KPI
  derivation (Q6 ruling 1).

### Invariant INV-1 (partition)

On every fixture and in every emitted JSON:
`verified + counterexample + skipped + errors + inconclusive` (+ `not_applicable`, if the
sibling doc has landed) `== len(results)`. The counters partition the per-function
results — this is the observable that catches any status added to one layer and not the
others, and it is how this doc composes with iteration 255's split instead of fighting it.

### Files to Modify

No `internal/smt` files are modified (r2 — see Q4).

- `cmd/ailang/verify_status.go` — NEW: shared `applySolveOutcome` (~50 LOC + drift/exhaustiveness tests)
- `cmd/ailang/ai_check.go` — call helper; `inconclusive` counter; exit-code doc (~15 LOC)
- `cmd/ailang/verify.go` — call helper; counter; exit codes (~15 LOC)
- `cmd/ailang/verify_print.go` — inconclusive branch (~15 LOC)
- `internal/eval_harness/verify.go` — reader field + propagation (~6 LOC)
- `internal/eval_harness/metrics.go` — `VerifyInconclusive` (~2 LOC)
- `internal/eval_harness/agent_runner.go` — `VerifyInconclusive` (~2 LOC)
- `internal/eval_harness/agent_prompt.go` — INCONCLUSIVE teaching line (~2 LOC)
- `internal/observatory/cost_per_verified_success.go` — predicate conjuncts (~4 LOC)
- `cmd/ailang/testdata/verify_bounded/{discrim,adt,q3}.ail` — NEW fixtures

---

## Conflict Surface

This change touches `cmd/ailang/` and depends on (reads, without modifying after r2)
`internal/smt/`'s `SolverStatus` enum, plus the two consumer packages. Enumerated:

1. **The verify status namespace is a shared, growing enum.** Live values:
   `verified, counterexample, skipped, error, unknown` (V13). The parked sibling doc adds
   `not_applicable`. This doc adds `inconclusive`. Every consumer that switches on the
   status string or sums counters is on the surface: `verify_print.go` (its
   `case "error", "unknown"` branch must NOT swallow the new value), both section structs'
   JSON, `aiCheckExitCode`, `verify`'s exit ladder, the harness parser (V20), and the
   observatory predicates (V10). INV-1 plus the exhaustiveness test make an unhandled
   member loud rather than silent.
2. **Two near-identical status ladders already drifting.** `@verify(depth: N)` exists in
   `verify.go` only (V21). Replacing both switches with one helper shrinks this surface;
   the cross-command equality test (AC-6) pins it.
3. **`m-contract-verification-coverage` (parked, r2) edits the same lines.** Both docs
   touch `isVerifiedSuccess`, `AICheckVerifyResult`, `metrics.go`, and the section
   counters. The designs are semantically compatible (disjoint new statuses; both
   sum-preserving; both reader-before-writer) but will conflict textually — whichever
   lands second rebases. This doc deliberately reuses that doc's landing pattern so the
   merge is mechanical.
4. **Version skew across the PATH-resolved child (`D-30`).** Old binary + new harness:
   `inconclusive` absent from JSON → parsed as 0 → behavior identical to today. New binary
   + old harness: `counterexample` drops to 0 → `VerifyOk` true, and `isVerifiedSuccess`
   without M1 **inflates** — a mixed result carrying any `verified > 0` alongside
   `inconclusive > 0` is counted as a verified success. **Landing order does NOT prevent
   this** (corrected at the controller carve-out, `gpt5-6-sol` r2, whose objection is
   sustained): commit order orders *repository commits*; it neither negotiates the runtime
   JSON schema nor prevents an old harness **binary** from invoking a newer `ailang`, because
   `RunAICheck` resolves its verifier child through **PATH** (`internal/eval_harness/verify.go:47-53`;
   both live call sites pass `""`). That is exactly the open ledger decision **`D-30`**, and this
   doc is the SECOND to be blocked by it independently — `m-contract-verification-coverage` is
   already `PARKED needs-human-review` on the same mechanism.
   **Mitigation, per the reviewer's own verbatim `proposed_fix`, added to M1:** an explicit
   old-reader/new-writer acceptance test using a fixture carrying BOTH `verified > 0` and
   `inconclusive > 0`, asserting the old reader does **not** score it a verified success.
   This DETECTS the skew; only a `D-30` ruling (option (b), binding `RunAICheck` to
   `os.Executable()`) can PREVENT it. The residual is named here rather than mitigated away —
   no silent fallback (CLAUDE.md §2).
5. **Exit-code consumers.** `ai-check` rc feeds agent convergence loops and
   `RunAICheck`'s "complete report on failing exit" contract (`eval_harness/verify.go:80`);
   `verify` rc feeds `--strict` CI-style use. The changes are strict relaxations on the
   artifact path (1 → 0) and a strict tightening under `--strict` (inconclusive now
   fails strict) — both stated in the acceptance criteria.
6. **Programs that MUST still work (regression set):**
   - `examples/runnable/contracts/shapes_verify.ail` — 3 verified / 1 error today; the
     error surface is the SIBLING row's scope and must be byte-identical after this change.
   - `testdata/verify_bounded/discrim.ail` — `aOk` verified, `bBad` genuine
     counterexample with concrete model, rc=1.
   - `testdata/verify_bounded/adt.ail` `headOr0` — verified at every depth (the control
     that unrolling-free functions are untouched).
   - Non-recursive verified/counterexample behavior across the existing
     `cmd/ailang` and `internal/smt` test suites (V19 baseline: all `ok`).
7. **What deliberately changes:** bounded-unrolled sat obligations (recursive function,
   depth > 0) stop exiting 1 and stop feeding repair hints; `verify --strict` newly
   fails on them; the human output for that case changes wording entirely; and —
   declared cost, Q3 — genuine counterexamples on recursive functions are reported
   `inconclusive` until witness replay lands.

---

## Acceptance Criteria

Every criterion is a command that can fail. Base status measured on `origin/dev`
(`bc3f80884`) via the pin worktree and the stamped scratch binary (V17–V19).
"RED at base" gates are the fix's proof obligations; "GREEN at base" gates must stay green.

| # | Command | Base (measured) | Required after |
|---|---|---|---|
| AC-1 | `go build ./cmd/ailang && go build ./internal/smt/...` (never `go build ./...` — rc=1 on pristine dev by design) | rc=0 (V19) | rc=0 |
| AC-2 | `go test ./internal/smt/ ./cmd/ailang/ ./internal/eval_harness/ ./internal/observatory/` | all `ok` (V19) | all `ok` |
| AC-3 | `ailang ai-check -verify-recursive-depth 2 cmd/ailang/testdata/verify_bounded/adt.ail`; assert rc=0, `verify.counterexample==0`, `verify.inconclusive==1`, `sLength` status `"inconclusive"` with nonempty `reason` and `bounded_depth:2`, `headOr0` `"verified"` | **RED**: rc=1, `counterexample=1` (V18) | GREEN |
| AC-4 | same on `discrim.ail`; assert rc=1, `bBad` `"counterexample"` with concrete model (`$p_x`, `result`, no `cRec_0`-style binding), `aOk` `"verified"`, `cRec` `"inconclusive"`, `verify.counterexample==1` | **RED**: `counterexample=2`, `cRec` misreported (V1) | GREEN |
| AC-5 | same on `q3.ail`; assert rc=0, all three functions `"inconclusive"`, `counterexample==0` (the DELIBERATE conservative direction, Q2) | **RED**: 3 counterexamples, rc=1 (V3) | GREEN |
| AC-6 | cross-command drift gate: per-function `{function,status}` pairs from `ailang verify --json` and `ailang ai-check` on `adt.ail` are identical (jq-compare in a Go test) | vacuously green (same ladder text today) | GREEN, and RED under MUT-ONE-LADDER |
| AC-7 | unit: `aiCheckExitCode` table extended — inconclusive-only → 0; counterexample>0 → 1 unchanged; `verify --strict` with inconclusive>0 → exit 1 | new tests (absent at base) | GREEN |
| AC-8 | unit: `isVerifiedSuccess{Verified:1, Inconclusive:1, others 0}` == false; `isVerificationFailure{pass, Inconclusive:1}` == true | new tests (absent at base) | GREEN |
| AC-9 | unit: repair categorization on a JSON with `inconclusive:1, counterexample:0` yields no `VERIFY_COUNTEREXAMPLE` errCode and no Z3 repair hint | new test (absent at base) | GREEN |
| AC-10 | INV-1 partition check asserted on every fixture in AC-3..AC-5 | n/a (counter absent) | GREEN |
| AC-11 | `examples/runnable/contracts/shapes_verify.ail` output unchanged (`3 verified, 1 errors`, same error text) — sibling surface untouched | measured this session | GREEN (byte-diff of verify section) |
| AC-12 | Flip confirmation, statuses only: extract the banked `code` field from each of the 8 affected v1.0 runs (V25; fields populated — V27) into temp `.ail` files, run the FIXED `ailang ai-check -verify-recursive-depth 2` on each; assert every one reports `verify.counterexample == 0` and `verify.inconclusive >= 1`. Explicitly NOT a KPI derivation — the number is bit-identical by construction (Q6 ruling 1) | **RED**: all 8 banked rows carry `verify_counterexample = 1` (V25) | GREEN |
| AC-13 | **old-reader / new-writer skew (`gpt5-6-sol` r2 verbatim `proposed_fix`; the `D-30` detection gate).** Unit test feeds the CURRENT (pre-M1) `isVerifiedSuccess`/`isVerificationFailure` an assessment built from a NEW-emitter JSON fixture carrying BOTH `verify_verified > 0` AND `verify_inconclusive > 0` with `verify_counterexample == 0`. Assert the old reader does NOT score it a verified success. | **RED at base** — with the `inconclusive` field unknown to it, today's predicate sees `verified>0, counterex==0, skipped==0, errors==0` and returns **true**, i.e. it inflates. This is the failure the gate exists to pin. | rc=0: predicate returns false, via M1's `VerifyInconclusive == 0` conjunct. **Detection only — a `D-30` ruling is what would PREVENT the skew; see Conflict Surface 4.** |

## Mutation Table

Each mutation names the downstream observable that kills it; the observable's value set is
never wider than the mechanism it guards.

| Mutation | What it breaks | Killed by (downstream observable) |
|---|---|---|
| MUT-REMOVE-BOUNDED-CHECK — classifier ignores the bounded-unrolling condition and maps every sat → `counterexample` (today's behavior restored) | the fix itself | AC-3: `sLength` per-function status string + `verify.counterexample` counter in emitted JSON |
| MUT-OVERSUPPRESS — classifier maps every sat → `inconclusive` (bounded condition dropped the other way, e.g. keying on `recursiveDepth > 0` alone without `IsRecursiveFunc`) | genuine counterexamples on NON-recursive functions lost | AC-4: `bBad` (non-recursive, at depth 2) must remain `"counterexample"` with a concrete model and rc=1 |
| MUT-DROP-BOUNDED-DEPTH — helper returns `"inconclusive"` without carrying `BoundedDepth`/reason (verdict and qualification desync) | user/JSON lose the depth the verdict is relative to | AC-3 asserts `bounded_depth: 2` AND a nonempty `reason` on the inconclusive result |
| MUT-ONE-LADDER — classification applied in `ai_check.go` but `verify.go` keeps its own switch | ladder drift returns | AC-6: cross-command per-function status equality on the same fixture |
| **MUT-ADD-STATUS (addition)** — a new `SolverStatus` member (e.g. `StatusResourceOut`) added to `internal/smt/solver.go` with no mapping | unmapped enum member silently mis-handled | `TestSolveOutcomeExhaustive`: iterates all `SolverStatus` members (bounded by the `String()` sentinel default) and requires `applySolveOutcome` to return a status from the allowed set with no fallthrough; goes RED on the addition — proves the mapping LOOKS at the enum, removal-shaped tests only prove it fires |
| **MUT-ADD-JSON-STATUS (addition)** — helper emits a new per-function status string without extending the counters | counters stop partitioning results | AC-10: INV-1 sum-vs-`len(results)` fails on any fixture |
| MUT-READER-DROP — `&& a.VerifyInconclusive == 0` removed from `isVerifiedSuccess` | KPI counts undecided runs | AC-8 predicate unit test (field set, expect false) |
| MUT-REPAIR-WIDEN — repair triggers on `Inconclusive > 0` too | repair turns burned again | AC-9 (errCode/hint observable, downstream of the trigger) |
| MUT-PRINT-SWALLOW — `verify_print.go` routes `"inconclusive"` into the existing `case "error", "unknown"` | user sees `! ERROR` for undecided | golden-output test on `adt.ail` human output containing `? INCONCLUSIVE (bounded: depth 2)` |

## Milestones

### M1 — Readers (independently committable; no behavior change at base)
Fields + predicates in `eval_harness` and `observatory`; repair non-trigger regression
test. **Acceptance: AC-1, AC-2, AC-8, AC-9.** All-zero `inconclusive` inputs make every
M1 change a no-op against current binaries — assert that with a
`VerifyInconclusive: 0` predicate test equal to today's truth table.

### M2 — Mechanism + emitters
Shared `applySolveOutcome` (structural bounded-sat classification; zero `internal/smt`
changes) + counters, exit codes, print branch, prompt line, fixtures, and the AC-12
status-flip confirmation.
**Acceptance: AC-1..AC-7, AC-10..AC-12 and the full mutation table.**

## Testing Strategy

- Fixture-driven CLI tests on the three copied probes (real Z3, matching how the defect
  was found); unit tests for the predicate, exit-code tables, and observatory predicates;
  golden test for the human output branch; cross-command equality test for drift.
- Per coding standards, no backward-compat shims: the old "every sat is a counterexample"
  expectation in any existing test is out-of-date by definition and is rewritten, not
  special-cased.
- Z3-availability: fixture tests skip (loudly) when no `z3` is on PATH, same as the
  existing `internal/smt` solver tests.

## Future Work

1. **Witness replay (the decisive second signal, Q3).** Parse the model's `$p_*`
   constructor terms, evaluate the real function on them with the interpreter, check the
   `ensures`: violation → promote to `counterexample` (with a *replayed*, trustworthy
   witness — fixing the measured garbage-witness problem too); satisfied → stays
   `inconclusive`. Deterministic-by-design language makes this well-defined for pure
   functions. Separate design; depends on this doc's `inconclusive` plumbing. The r1
   `FrontierSymbols` collection (emission sites `unroll.go:44`, `hof_inline.go:371`,
   `list_unroll.go:48/82/115` — V7/V15) is demoted to THIS future design as a replay
   *input* (identifying which stub the model exercised); per quorum objection 1 it must
   never gate a verdict.
2. **Induction-hypothesis frontier encoding.** Assume the function's own `ensures` on the
   stub (`f_0`) — standard assume-guarantee. Measured consequence sketch: `cRec`/`sLength`
   would verify outright (inductively), and two of the three Q3 probes would regain
   concrete genuine witnesses. Alters the `verified` surface and introduces quantified
   assertions (decidability risk) — its own design with its own A/B.
3. **The sibling row** `m-verify-unencodable-reported-as-error` routes `error`-shaped
   encoding refusals into `skipped` — after which all three faces of "could not discharge"
   are honest and INV-1 still partitions.

## Axiom Compliance

Tooling/reporting change, no language-surface impact. A1 (determinism): classification is
a pure function of solver output + encode metadata — deterministic. A3 (explicitness): the
new status makes an implicit incompleteness explicit; reasons are structured. A4
(fail-loud, CLAUDE.md P2): replaces a silently-wrong verdict with a typed outcome; the
exhaustiveness test makes future enum growth loud. A7 (AI-friendliness): agents stop
receiving actively-misleading repair instructions. No axiom is violated; net positive.

## Related Documents

- [m-contract-verification-coverage.md](m-contract-verification-coverage.md) — iteration
  255, parked r2. Same counters/predicate, disjoint status (`not_applicable` = "nothing to
  verify" vs this doc's `inconclusive` = "attempted, undecided"). Landing-order pattern
  reused; textual rebase required by whichever lands second.
- `design_docs/implemented/` M-SMT-BOUNDED-RECURSION (introduced the unrolling +
  `BoundedDepth` reporting this doc completes on the failure branch).
- `design_docs/PROGRAM.md` — routing: this is an **AILANG fix** lane item (verifier
  correctness), not a motoko extension.

## References

- Probes (to be copied into `cmd/ailang/testdata/verify_bounded/`):
  `/tmp/i259_probe/discrim.ail`, `/tmp/i259_probe/adt.ail`, `/tmp/i259_probe/q3.ail`
- Measured artifacts: `/tmp/i259_probe/discrim.json`, `adt_2.json`, `adt_4.json`,
  `q3_2.json` (session 2026-08-23); frozen-cohort measurements in V24–V27
- `D-29` (KPI ruling), `D-30` (PATH-resolved verifier child), `D-32` (inconclusive-exemption
  policy question filed by the coordinator — see
  [Open for human ruling](#open-for-human-ruling--d-32)), CLAUDE.md Principles 2 & 3

## Revision history (r1 → r2)

Round-1 quorum: **blocked**, 2/2 external reviewers present (no N−1 degrade). Both
objections were verified by the controller before forwarding, and every number below was
re-measured first-party in this session rather than transcribed (V23–V27).

- **Objection 1 (gpt5-6-sol) — discriminator unsound, its safe branch unreachable.**
  Absence of a frontier symbol from Z3's parsed/raw model is not a dependency analysis
  (solvers may omit interpretations; `parseDefineFun` measurably mangles function-sorted
  entries), so the r1 decision table's "unrolled sat with no observed frontier binding →
  `counterexample`" row was the one path that could still emit a false violation — while
  a 6-depth sweep (V23) shows the frontier binding present in 18/18 genuinely-buggy
  recursive readings, i.e. the row is unreachable on every constructible case.
  **Fix applied:** discriminator deleted; classification is the structural predicate
  `IsRecursiveFunc && recursiveDepth > 0 && sat ⇒ inconclusive` (Q3/Q4, decision table,
  overview); `EncodeResult.FrontierSymbols` / `SatDependsOnFrontier` / `frontier.go`
  removed from the design (`internal/smt` now has ZERO code changes), demoted to
  witness-replay inputs in Future Work; the cost — genuine recursive counterexamples
  such as `sumNonNeg`'s real witness become `inconclusive` — is now stated plainly in Q3
  as the doc's main limitation, with the Q2 argument for why it errs survivably.
- **Objection 2 (gemini-3-1-pro) — the r1 "corrected companion KPI" was arithmetically
  false.** Under M1's own `&& VerifyInconclusive == 0` conjunct, flipped runs still fail
  `isVerifiedSuccess`, so `verified_successes` (3) and `known_cost_usd` ($2.3334561216)
  are unchanged and `$0.7778187072` is bit-identical before/after (V24). **Fix
  applied:** Q6 rewritten as three rulings; the measured blast radius — 8 of 30 frozen
  runs (27%), all on the two recursive-ADT benchmarks, uniform across 4 model families
  (V25/V26) — promoted into the Problem Statement as headline evidence; the vacuous
  corrected-KPI deliverable and AC deleted; AC-12 rewritten as a status-flip
  confirmation over the 8 banked `code` fields (V27); the fix's actual value stated
  (honest labels, no repair turns spent breaking correct code, `D-29` second clause
  unblocked); and the only number-moving decision — exempting `inconclusive` from the
  effective arm, `D-29`-style — filed as **`D-32`** for human ruling, deliberately
  unanswered and undepended-upon here.

Settled and unchanged per the revision protocol: Q1 `inconclusive` status/counter, Q2
soundness direction, Q4 single `applySolveOutcome` seam + drift test, Q5 wording, the
scope ruling on `m-verify-unencodable-reported-as-error`, reader-before-writer landing
order, addition-shaped mutations, and the baselined acceptance gates.
