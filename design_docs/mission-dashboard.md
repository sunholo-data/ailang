# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~19:55 local (iteration 186)

## Now
- **v0.33.0** · merge `905722f28` — **`#617` M4 LANDED, `#617` COMPLETE** (M1–M4, AC-1..AC-7;
  PR #681). Gate 3b GREEN: `checks=21`, `pending=0`, zero NOT-GREEN, 4/4 REQUIRED from real
  `pull_request` events. Evaluator **sonnet PASS 98/100 r1, zero blocking**. `$0.00`.
- 🟢 `ailang check` now warns `LIST_TAKE_AFTER_FLATMAP`, pointing at `takeFlatMap`. **Reachability
  verified through the real CLI** — the point, since `#617` *is* a shipped-but-unreachable defect.

## 🔴 The plan would have shipped a detector that never fires
It specified matching `App(take, [n, App(flatMap, …)])`; elaboration **always** emits ANF
(`let tmp = flatMap(f, xs) in take(n, tmp)`). The codex executor **self-reported** it; I
adjudicated both arms — neutering the ANF arm LANDS, BUILDS (rc=0), reds `/direct_trap` with
`got warnings: []`. Plan-as-written = a green suite pinning nothing: `#617`'s own failure mode
recreated inside the sprint fixing it (3rd form — iter-182 frozen prompt, iter-183 no CI gate).
**Rule 3h(d) vindicated**: a "deviations are suspect" prior loses this finding.

## Also worth knowing
- The unexercised nested-`App` arm was *named* (rule 3i(d)), then **pinned** once SonarCloud
  red-lit new-code coverage on exactly those lines (control: `dev` green ⇒ not inherited).
- **`#680`** filed: nested composition warns only on the OUTERMOST trap (1 of 2). Two of my own
  instruments failed and were caught by their paired controls.

## Queued
**Next: `m-bytecode-vm-parity-bugs`** (clause-2 SOUND residue, ≤2d). Then **`#616`** `D-10` ·
**`#619`** `D-9` · **`#618`** `D-8` · #636 · #613 · #604/#614 · #649 · #651 · #654 · #669 · #670 · #680.

## Loop + routing
Controller **opus** · executor **codex `gpt-5.6-sol`** (~13 min; no designer/planner — doc
quorum-resolved, plan existed) · evaluator **sonnet**; generator≠judge held. ⚠ **Running skill
still NOT origin's, 2nd iteration**: `~/.claude/skills/…` → *main checkout*, 5 behind but clean —
driver-pin fixed the *code* path, not the *skill* path.

## PARKED ON MARK — #635
**`D-11`** slot-death guard (driver `rc=0` when elapsed ≪ claimed work?) · **`D-10`** `#616` ·
**`D-9`** `#619` · **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`** codex · **`D-8`** `#618`.
