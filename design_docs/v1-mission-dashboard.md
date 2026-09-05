# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History: `v1-mission.md` STATUS + `v1-mission-log.md`.*

**Last iteration:** 331 — 2026-09-05 — [PRODUCT] — the compile cache now keys on the bytes the lexer actually parsed. M2 of 4 landed; judge PASS 96/100, zero blocking.

**Release:** v0.35.1. **Goal:** N = **13** docs before v1.0.0 (±0, *goal unmoved* — a doc leaves the count when it LANDS, and this one is 2 of 4 milestones in).

## In flight

**[IN-SPRINT] `m-compile-cache-unverified-artifacts`** — **M2/4 LANDED**
([#1053](https://github.com/sunholo-data/ailang/pull/1053) →
[`f5edd569a`](https://github.com/sunholo-data/ailang/commit/f5edd569a)). The loader retains the exact
lexer source as an immutable `SourceContent *string`; the pipeline hashes it instead of re-reading
disk; a module with no snapshot bypasses both lookup and publication rather than being hashed as
`""`. End-to-end: cold `42` → warm `42` → edit `99` → warm `99`. Judge `sonnet` **PASS 96/100, 0
blocking**; its one finding, anchored to the diff rather than the plan's table, was that "bypasses
publication" meant *fails to publish* — an arm now pins *never attempts*. **Next: M3**, 0.5 d.

## Up next (banked, ranked)

0. **`ci-red-mission-loop-workbench` — OUTRANKS EVERYTHING.** `dev` is red on SIX jobs (`lint`,
   `docs-gate`, `docs-build`, `launchd drivers`, `test-windows`, `Build windows-latest`), all
   inherited from a concurrent attended session's four workbench commits — the same set is red on
   the commit before iteration 331's merge. **`docs-gate` blocked every PR in the repo** — one
   broken link failing the Docusaurus build. It and `lint` are fixed in #1054; four are not. A red
   dev is the owning mission's first deliverable and V1 owns this repo.
1. **M3 then M4 of the in-flight sprint** — plan verified, cumulative gate already accepts both.
2. `m-cachesrc-cognitive-complexity` — **new**: SonarCloud new-code maintainability red on M2's diff
   (rating 4, six `go:S3776`); non-required, but M3/M4 re-enter the same function and inherit it.
3. `m-cache-artifact-adversarial-decode` — the `D-55` hardening lane; below the correctness fix by
   construction, since HEAD is strictly worse on every axis it names.

## Loop health

- Iterations 326–331 all produced records — no reaped slots.
- Routing: controller opus · designer and planner **not spawned** (doc and plan both existed —
  Fable budget unspent, rotation pointer stays `codex:gpt-6-astra`) · executor `codex:gpt-5.6-sol`
  (probe rc=0, resolver agreed) · evaluator `sonnet` in its own worktree. generator≠judge held by
  vendor as well as model. `metered=$0.00`.
- **Lesson: a baselined gate list can still be a short one.** Every command I handed the executor
  was green at base and on the milestone; CI failed on `make check-home-isolation`, never on the
  list. Deriving the `test` job's 57 steps from `ci.yml` costs one line; not doing it cost a CI cycle.

## Parked on Mark

**`D-55` (OPEN)** — adversarial gob-decode scope. Its default (a) is applied and M1+M2 have now
shipped under it, so **nothing is stalled**. Answer only to change course.

**Quota:** subscription/quota buckets only. Metered $0.00 of the $5 ceiling; no quorum round ran.
