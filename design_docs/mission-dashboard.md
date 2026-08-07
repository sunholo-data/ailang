# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-07 ~02:45 local (iteration 156)

## Now
- **Latest release**: v0.33.0 (2026-08-04)
- **Actions outage RECOVERING** (incident 15:22Z, still `investigating` at write time). Fresh runs
  work; runs created during the peak are **permanently wedged at `jobs=0` and cannot be cancelled**.
- **THE UNWEDGE RECIPE** (needs no local checkout): `POST git/commits` with the branch's **existing
  tree SHA** + current head as parent, then `PATCH git/refs/heads/<br>`. Tree-identical empty commit
  → real `pull_request: synchronize` → fresh PR-scoped checks. `workflow_dispatch` is **NOT** enough:
  its checks land on the SHA but do **not** satisfy branch protection.
- **`#606` + `#608` MERGED** (squash `49a6af789`, `1d355245a`; `#602`/`#603` auto-closed) — all four
  required contexts green on a real `pull_request` suite. Three iterations of backlog cleared.
- **`#613` = proxy-boundary M1, DRAFT / DO-NOT-MERGE.** Its own ACs pass; it reds CI **by design**.
- **Next picks**: `D-1` decides M1's shape · `#545` · Lane B1 · `#604` · `#612`
- **Routing**: controller opus-5 · executor `pi:…deepseek-v4-flash-0731` (datapoint 4) ·
  evaluator sonnet · designer next `claude:claude-fable-5` · loops v1 90min / world 4h, both armed

## Parked on Mark
- **`D-1` — NOW CONCRETE, AND IT HAS A THIRD OPTION.** Implementing the reviewed design faithfully
  **removes SSRF IP-blocking for literal private IPs** (`https://10.0.0.1/…` goes to the proxy
  unvalidated). Measured with a negative control: `TestNetIPValidation` rc=1 / 4-of-7 fail on the
  branch, rc=0 / 7 pass on pristine dev, under the poison **ci.yml sets on its own legs**.
  **(a)** ratify as written + update those tests (narrows the shipped guarantee); **(b)** validate
  when `net.ParseIP(host) != nil` even on the proxy route — zero DNS, no TOCTOU, shrinks `D-1` to
  **hostnames only**. The independent evaluator recommends **(b)**. One word unblocks M1.

## Recently settled (don't re-ask)
- **D-6 = (A)**, **RELEASE = WAIT**, **D-7 = KEEP WEEKLY** (all Mark, attended 2026-08-06)
- Standing fast-forward authorization **YES**; the 08-03 dev reconcile has held **16 iterations**
- **codex quota dry until ~2026-08-08** — the pi lane covers executor work

## Open findings worth knowing
- **`pgrep -f` is unreliable in BOTH directions.** A false *negative* made me declare a live
  executor dead and spawn a second into the same worktree. Wait on the handle or the artifact.
- **A GREEN during an open incident is not a settlement** — it licenses a code inference, never an
  infrastructure one. Corroborated here: CI[dev] success 17:32Z → failure 20:03Z, wedged runs across.
- **Named-test `-run` gates are rc=0 with `[no tests to run]` at base** — assert a `=== RUN` count.
  A subtest can also hide in a top-level test the graded regex **excludes**, and so never run.
