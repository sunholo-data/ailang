# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~08:00 local (iteration 175)

## Now
- **v0.33.0** · `origin/dev` `48e08b168` · SonarCloud **green** on the last head (the standing `#615`
  red did not recur — `checks=15`, zero not-green at Gate 1).
- 🟡 **`#618` M4 in flight** — PR [#652](https://github.com/sunholo-data/ailang/pull/652), 8 files
  `+467/−39`, `MERGEABLE`, checks pending at time of writing. Repo gates all rc=0 (controller-run).
- ✅ **AC-M4.2 field validation PASSED on the rig** — `docx_reimplement` × `motoko-local-qwen3.6-35b`,
  flag ON. Across 85+ streamed requests: **`effective_deadline_sec` == configured on every one**
  (REFUTATION #1 closed in the field, not just in `httptest`), **zero idle/TTFT trips**,
  `saw_first_byte` true throughout. Headroom TTFT **23.5×**, total **4.7×**, idle **3.9×** — idle is
  the tightest Design-Freeze margin and the one to watch.
- 🔴 **The stopgap had a SECOND delivery site the plan missed.** `launchctl getenv
  AILANG_OLLAMA_HTTP_TIMEOUT_SEC` = **1800**, a launchd domain global no plist edit touches;
  `b67d415cd` says so in its own body. AC-M4.3's grep criterion would have gone green with the
  hazard live. And the obvious cleanup order is the harmful one — clearing the global *before*
  installing flag-on plists drops uncovered calls to the 300s default and reintroduces `#618`.
  AC-M4.3 now carries an **ordered** rollout, marked as rig-state work outside the repo deliverable.
- 🔴 **`#651` NEW — `design-quorum`'s zero-signal guard is vacuous.** The controller's own
  `--controller-verdict` increments the same `presentCount` the guard tests, and Gate 2 mandates
  that flag (86 of 87 artifacts carry one). Three docs shipped on `proceed` with **0 of 2** reviewers
  present — and all three had said *reject*, still recoverable from the raw text on disk.

## In flight / queued
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, not done.
- **Lane B1 M4** (`#517`) — ADTs, recursion/size budgets, `TypeApp`; resumes next.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2` · **#649** local-model capability gap (triaged, not a regression).

## Next
**Lane B1 M4 (`#517`)** once `#652` is green and merged. `#618`'s repo work ends with M4; what
remains is the rig rollout, which is `D-8` below.

## Loop + routing
Controller **opus** · designer/planner **not fired** (doc + plan exist; rotation pointer unchanged,
next `codex:gpt-5.6-sol`) · executor **opus** (pi pre-empted, 2nd iteration running) · evaluator
**sonnet** — generator≠judge holds. Metered **$0.00** against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. (A) as-written ·
  (B) narrow to literal-IPs · (C) rethink. — `#613` blocked on this.
- **`D-2`**: `#604`/`#614`.
- **`D-7`** (iters 172–175): pi executor 3/3 failed; pre-empted to opus a 2nd time. **(A)** keep
  pinned · **(B)** flip to codex · **(C)** opus-only. The env pin is now effectively fiction.
- **`D-8` NEW**: authorise the `#618` rig rollout (install flag-on plists, then clear the launchd
  global) — or hold the rig on the stopgap until attended.
