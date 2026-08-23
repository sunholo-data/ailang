# Mission Dashboard — Motoko

**Snapshot** (overwritten every iteration; history lives in the charter STATUS + log)
Last iteration: **19** · 2026-08-23 · queue row **6c** LANDED

## Where we are
- **Latest landing**: [`c1950750c`](https://github.com/sunholo-data/ailang/commit/c1950750c) (PR
  [#831](https://github.com/sunholo-data/ailang/pull/831)) — the connection probe's self-test was
  wired into CI for the first time, and two of the probe's own refusals were made able to refuse.
  Evaluator **PASS 93/100, zero blocking**. Gate 3b green, observed.
- **Probe self-test**: 8 arms → **34**, all mutation-proven, now running in
  `make test-launchd-drivers` (`ci.yml:507`).

## Next picks
1. **6d** *(new)* — the declared `MISSION_WORKDIR` holds a rulebook 160 commits stale; decide
   whether the `#558` pin root is canonical and say so, or bring the workdir current.
2. **6** — **M2 (`AC-D1-live`) is still the resume point**. V38 is *not* isolated: the mechanism by
   which the probe as shipped breaks the runs it observes is still unknown. It now has the driver
   logs it needs, and isolating it **needs the rig** (`rig.lock`).
3. **7** profile restoration · **8** repin the stale OpenRouter motoko models.

## Blocked / parked
- **Phase 0 CLOSED** on G1 alone: `arniwesth/motoko_agent#154` is `OPEN` (control: `#161` MERGED
  with a non-null `mergedAt`). Rows **10/11/12** stay parked. G5 — Arni's ABI-settled declaration —
  is a permanent human residual by design.
- Rows **9** and **13/14** wait on a green tree / on 13 first.
- Upstream `#165` OPEN, its fix `#166` OPEN against `main_dst`. Nothing local waits on either.

## Loop health
- Cadence 12h; runs from the `#558` driver pin root. Pin / `~/.claude` resolved target / origin
  skill copies byte-identical three ways.
- Routing: controller opus-5 · executor `codex:gpt-5.6-sol` · evaluator sonnet (own worktree).
- **Metered $0.00 of $5.** No GPU this iteration.

## Parked on Mark
**Nothing.** Decision ledger: 4 rows, **0 OPEN**. 0 directives on `#743` since the watermark.
