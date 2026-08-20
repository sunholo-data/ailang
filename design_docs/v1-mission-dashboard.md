# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History: `v1-mission.md` STATUS + `v1-mission-log.md`.*

**Last iteration**: 235 · 2026-08-20 09:03–10:2x CEST · **LANDED**
**Release**: v0.33.1 · `origin/dev` @ `0a9367937` → +`e5ee6c5e5`

## What just happened

**`D-1` discharged and PR [#613](https://github.com/sunholo-data/ailang/pull/613) landed after 13 days
as a DO-NOT-MERGE draft.** M1's request-aware Net RoundTripper performed **zero** target validation on
the proxy route. Reproduced first-party under the ci.yml proxy poison, both arms, same command:
branch **rc=1, 4 of 7** subtests failing · pristine dev **rc=0, 7/7** — a firing negative control.
Mark's `D-1` ruling (2026-08-19) was **RETAIN zero-DNS literal-IP validation**, and it is now shipped:
literal IPs are validated with no resolver; hostnames stay the accepted `D-5` residual.

**The evaluator earned its keep.** Sonnet PASS **93/100** with two reproducible findings, both fixed
*before* merge rather than filed as follow-ups:

- **`net.ParseIP` rejects RFC 4007 zone identifiers**, so `http://[fe80::1%25eth0]/x` fell through to
  the hostname branch and reached the proxy **unvalidated** — the exact hole `D-1` exists to close,
  wearing an encoding the guard did not recognise. The **direct** route was measured, not assumed:
  it fails *closed* (`E_NET_DNS_FAILED`). Only the proxy route failed open.
- **The arm named for the mechanism did not test it** — `proxy_literal_blocked_before_dial` survived
  having its own precondition removed, because the *direct* route refuses the same literal with the
  same text and the same zero counters. It now asserts the proxy selector was consulted.

## Next picks

1. **LC-1 `m-list-repr-spike`** — gates the whole cons-cells programme (carries its kill criterion).
2. `m-ui-dependency-tree-unbuildable` — `ui/` unbuildable 40 days ([#503](https://github.com/sunholo-data/ailang/issues/503)).
3. `m-stdlib-reverse-delegates-to-builtin` — cheap, and *required* under cons cells.

## Loop health

- Cadence normal. **Zero Fable runs** (no designer fired — existing doc + existing plan).
- Executor `codex:gpt-5.6-sol` rc=0, one bounded run; evaluator sonnet (generator≠judge holds).
- Cost: **$0.00 metered** of $5. Quota buckets: opus (controller), codex (executor), sonnet (judge).
- Gate 3b on the PR head: **21 checks, 0 not-green**, 4/4 required, `MERGEABLE/CLEAN`.
- `make check-file-sizes` caught a 921-line test file **before** push — the derived-gate sweep working.

## Parked on Mark

- **Rotate `AILANG_REGISTRY_API_KEY`** — carried from iter-232 (its value was printed into a
  transcript; not exposed externally). Not a ledger row; the decision ledger is **21 rows, 0 OPEN**.

## Known bookkeeping defect

`v1-mission-log.md` has a **duplicate entry number 232** (230–233 also out of order), so entry
numbers are not a reliable index. Recorded, not silently renumbered — queue row
`m-mission-log-entry-numbering`.
