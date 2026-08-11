# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~05:50 local (iteration 174)

## Now
- **v0.33.0** · `origin/dev` `86f7f1c32` · Standing SonarCloud red (`#615`) unchanged — non-required,
  inherited (negative control: `failure` on recent code commits, absent on doc-only ones).
- ✅ **`#618` M3 LANDED** — PR [#650](https://github.com/sunholo-data/ailang/pull/650) → squash
  `86f7f1c32`. Gate 3b GREEN: 4/4 REQUIRED contexts from real `pull_request` events,
  `state=CLEAN`, `checks=20` zero not-green; merge-commit CI + Build-and-Release green.
  Evaluator sonnet **93/100 PASS, zero blocking**.
- **The streaming /v1 path now has full response parity** — `resp.Reasoning` populated from
  thinking deltas (non-nil `onChunk`, E-1) + Hermes `<tool_call>` recovery on zero native tool
  calls, mirroring `openai/step.go:610-616`. This closes REFUTATION #2 (the disengagement
  regression streaming-without-parity would have shipped). Controller re-ran R8/R9 mutations
  first-party: each builds, its parity test reds, suite-minus-test stays green, restore sha256-clean.
- 🟢 **Deploy-Docs 500 was a GitHub Pages outage, not us** — docs-gate (required) passed; only the
  Pages *publish* step 500'd; re-run on a byte-identical tree flipped failure→success.
- 🔴 **pi executor PRE-EMPTED to opus this iteration (FLAGGED)** — 3/3 identical `stopReason=length`
  failures on this sprint + a 1.2 GB-in-6 min disk-fill hazard made a 4th run pure downside. D-7 open.

## In flight / queued
- **`#618` M4** (fixture replay + rig validation + stopgap removal + docs, **GPU**; owns the doc
  renumber) — the default-flip precondition (E-4). M3 landing does NOT close the sprint.
- **Lane B1 M4** (`#517`) — ADTs, recursion/size budgets, `TypeApp`; resumes after.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2`.

## Next
**`#618` M4** — rig validation + stopgap removal (**GPU**, `rig.lock` around AC-M4.2 only). It is the
gate ruling E-4 requires before the v0.35.0 default flip: parity (M3, done) AND rig validation both
must hold. M4 also owns the design-doc renumber the plan introduced.

## Loop + routing
Controller **opus** · designer/planner **not fired** (doc + plan exist; rotation pointer unchanged,
next `codex:gpt-5.6-sol`) · executor **opus** (pi pre-empted after 3rd failure) · evaluator **sonnet**
— generator≠judge holds. Metered **$0.00** (pi not run) against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. (A) as-written ·
  (B) narrow to literal-IPs · (C) rethink. — `#613` blocked on this.
- **`D-2`**: `#604`/`#614`.
- **`D-7`** (iters 172-174): pi executor 3/3 failed (silent `stopReason=length`, disk-fill hazard).
  **(A)** keep pinned with the guard · **(B)** flip to codex (bucket refilled 08-10) · **(C)** opus-only.
  Loop is falling back to opus each time; pre-empting the run now to avoid the rig hazard.
