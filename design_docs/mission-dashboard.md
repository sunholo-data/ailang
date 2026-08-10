# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~00:20 local (iteration 172)

## Now
- **v0.33.0** · `origin/dev` `752f997d1` · Standing SonarCloud red (`#615`) unchanged — non-required,
  inherited (negative control: `failure` on 4 analysed commits, *absent* on the two doc-only ones).
- ✅ **`#618` M1 LANDED** — PR [#647](https://github.com/sunholo-data/ailang/pull/647) → squash
  `752f997d1`. Gate 3b **GREEN**: 5/5 workflows settled, all four REQUIRED contexts from real
  `pull_request` events, `CLEAN`. Evaluator **95/100 PASS, zero blocking**.
- 872 lines, 2 new files, **zero call sites** — runtime behaviour unchanged by construction.
  Package goes **23 → 30** top-level `--- PASS`.
- 🔴 **THE PRIMARY EXECUTOR LANE FAILED TWICE, rc=0, ZERO FILES.** `pi:deepseek-v4-flash-0731`
  ended both runs on a single ~63k-char **thinking** block that hit the 16,384-token output cap
  (`stopReason=length`), emitting no tool call — and pi exits **0** on that. A new false-green
  shape: *success reported for work never done*. Fell back to opus (**FLAGGED**), which delivered.
- 🟡 **Executor deviation, adjudicated ACCEPTABLE by measurement in both arms**: the plan puts the
  hard deadline in M2, which left AC-M1.3's third sentinel unreachable and untestable, so the
  executor added an in-reader budget. Neutering it reds *only* the deadline case; with it neutered
  and that test skipped the package is rc=0 → strictly additive.

## In flight / queued
- **`#618` M2** (flag-gated streaming branch + mandatory hard deadline — owns S8/S5 **and the
  inert-deadline fix**) → M3 (parity: `Reasoning` + Hermes) → M4 (rig, **GPU**).
- **Lane B1 M4** (`#517`) — ADTs, recursion/size budgets, `TypeApp`; resumes after.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2`.

## Next
**`#618` M2** — the milestone where both planner refutations actually get closed, so review it
hardest. **Carry into M2**: `streamBaseTransport()`'s `sync.Once` freezes `ResponseHeaderTimeout`
at first use (reproduced first-party: froze at 1m51s while the resolver correctly tracked 999s).
Any M2 test varying `AILANG_OLLAMA_TTFT_TIMEOUT_SEC` per case gets a stale reading unless it runs
first or resets the `sync.Once` — it will pass green while measuring nothing.

## Loop + routing
Controller **opus** · designer/planner **not fired** (doc + plan exist; rotation pointer unchanged,
next `codex:gpt-5.6-sol`) · executor **opus FALLBACK** after two pi failures · evaluator **sonnet**
— generator≠judge holds. Metered **$0.025** (both dead pi runs) against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. **(A)** as-written ·
  **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.

Full record: charter `## STATUS … ITERATION 172` + `v1-mission-log.md`.
