# foldl over [string] with an `ensures` reports `! ERROR` instead of the honest skip/inconclusive

> **Reproduced 2026-11-25** on v0.61.1 (commit cb6bb75, the reporter's build) with Z3 4.8.12:
> the reported shape (`foldl` over `[string]`, record accumulator, `ensures { result.signedUp >= 0 && … }`)
> → `! ERROR referralTotals`, `reason_code: solver_unknown`, `bounded_depth: 3`. **The record accumulator is
> not the differentiator** — a plain `foldl(\acc r. acc + 1, 0, refs)` over `[string]` with
> `ensures { result >= 0 }` errors identically. What differs is the element sort: the same foldl over
> `[int]` gives the honest `? INCONCLUSIVE (bounded: depth 3)` — with `(Seq Int)` Z3 builds a sat model off
> the unconstrained frontier stub (`_foldl_spec_*_0` is only *declared*, so it returns anything, e.g. −1),
> which the bounded classifier turns into inconclusive; with `(Seq String)` Z3 4.8.12 returns `unknown`,
> which the v0.61.1 binary leaves in the ERROR lane. The dev source at 124913d0 (squash, 2026-11-25
> 11:53) already contains the fix-shaped downgrade — `unrolledUndecided(vr, callees, encResult.HOFBoundedDepth > 0)`
> in `internal/smt/verify.go` (`Verify` loop) appending `"its map/filter/foldl chains"` — but the running
> binary does not contain that string (`grep -ac` on the binary: 0, while
> "could not decide this contract through the bounded unrolling of" is present), i.e. the v0.61.1 build
> predates the HOF-aware `unrolledUndecided` branch the [program doc](../v0_61_0/m-verify-soundness-coverage-program.md)
> M2 row describes ("bounded (depth 3) and classified like recursion"). Adjacent evidence on the same
> toolchain: `examples/runnable/contracts/hof_verify.ail` `incrementAll` (map) hard-errors with
> `unknown constant seq.map` — the map frontier in `internal/smt/hof_inline.go` (`hofFrontier`) emits
> `seq.map`, which Z3 4.8.12 does not know — so ERROR-lane brittleness on old Z3 is not foldl-specific.
>
> **Action before any new design doc:** re-run `ailang verify` on a dev build (or confirm the shipped
> v0.61.1 lacks the `unrolledUndecided` HOF branch). If the ERROR becomes the M2-classified skip
> ("the solver could not decide this contract through the bounded unrolling of its map/filter/foldl
> chains"), this row is fully absorbed and nothing else is owed; making `referralTotals` actually
> VERIFIED is the induction lever the program doc already tracks (§Remaining work item 6). The reporter's
> `listLength`/`filter` rewrite verifying cleanly is consistent: it removes the foldl specialisation
> (and its frontier stub) from the proof.

- **Date**: 2026-11-25
- **Class**: bug
- **Recommend**: duplicate-of ../v0_61_0/m-verify-soundness-coverage-program.md
- **Searched**: `foldl` and `smt|z3|verify` across design_docs/ (hit:
  `planned/v0_61_0/m-verify-soundness-coverage-program.md` — M2/M5 foldl specialisation, the
  editApply_* "foldl the solver cannot decide" row, §Remaining work item 6 "Inconclusives and undecided
  bounded unrollings: induction"); `ailang-core-triage/verify-hof-inline-behind-callee-sort-gate.md`
  (RESOLVED v0.61.0, same ERROR-vs-honest-verdict lane); `referralTotals`/`referral` — none found.
- **Estimate**: omitted (duplicate-of; the one live check is `grep -ac "its map/filter/foldl chains"`
  on a v0.61.1 release build vs dev, plus a re-run of the repro above)
