# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~02:10 local (iteration 173)

## Now
- **v0.33.0** · `origin/dev` `ff1fa0760` · Standing SonarCloud red (`#615`) unchanged — non-required,
  inherited (negative control: `failure` on 6 analysed commits, *absent* on the doc-only ones).
- ✅ **`#618` M2 LANDED** — PR [#648](https://github.com/sunholo-data/ailang/pull/648) → squash
  `ff1fa0760`. Gate 3b **GREEN**: all 4 REQUIRED contexts from real `pull_request` events,
  `mergeable=MERGEABLE state=CLEAN`, SHA-addressed `checks=20`, zero not-green.
  Evaluator sonnet **97/100 PASS, zero blocking**.
- **The inert-deadline fix (planner refutation R1) is closed.** `step.go` diff is **+15/−0** —
  the streaming branch derives from the context captured *above* `ollamaCallContext`'s 300s wrap.
  AC-M2.4 is its gate and is the **sole killer**: move the capture below the wrap and the read-back
  reads `299.999963583s` against a configured `3600s`. Package **30 → 38** `--- PASS`.
- 🔴 **pi executor failed a THIRD time, same shape — now instance 2 across iterations.**
  `stopReason=length`, **rc=0**, zero files, after 10 healthy tool calls. Also wrote a **1.2 GB**
  NDJSON in 6 min. Fell back to opus (**FLAGGED**), which delivered. Skill edit landed (Gate 5).
- 🟡 **My own ruling E-3 was REFUTED by the executor and confirmed by measurement**: the error
  *type* cannot discriminate the R1 defect (precedence maps any `streamCtx` deadline to
  `ErrStreamDeadlineExceeded`), so only the read-back value can. Read-back arm only, as delivered.

## In flight / queued
- **`#618` M3** (parity: `Reasoning` + Hermes tool-call recovery — the disengagement regression)
  → M4 (fixture replay + rig validation + stopgap removal, **GPU**).
- **Lane B1 M4** (`#517`) — ADTs, recursion/size budgets, `TypeApp`; resumes after.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2`.

## Next
**`#618` M3** — response parity. It exists because streaming the motoko path *without* it trades a
timeout bug for a **disengagement** bug (this mission's most-studied failure mode): `streamstep.go`
sets `out.Reasoning` **0** times and has **0** Hermes references. Ruling E-1 (non-nil `onChunk`) and
E-2 (export `extractHermesToolCalls`) are the seams. M2 left the callback wired but counting only.

## Loop + routing
Controller **opus** · designer/planner **not fired** (doc + plan exist; rotation pointer unchanged,
next `codex:gpt-5.6-sol`) · executor **opus FALLBACK** after a 3rd pi failure · evaluator **sonnet**
— generator≠judge holds. Metered **$0.021** (the dead pi run) against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. **(A)** as-written ·
  **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-7`** (iter-173, NEW): pi has now failed 3/3 real sprint runs. Keep it as the pinned executor
  with the new guard, or fall back to codex (refilled 08-10) until it earns a datapoint? **(A)** keep
  pi · **(B)** flip to codex · **(C)** opus-only.

Full record: charter `## STATUS … ITERATION 173` + `v1-mission-log.md`.
