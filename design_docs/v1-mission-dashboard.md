# Mission Dashboard — V1

_Snapshot, overwritten each iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Last iteration:** 320 · 2026-09-02 · LANDED · [HARNESS]
**Latest release:** v0.34.0

## Goal distance
**N = 10 design docs remaining before v1.0.0** — unmoved this iteration (HARNESS work moves the goal by 0).
D-53's **UNCLASSIFIED bucket of 4** (would make it 14) is still named and unruled.

## Just landed
**dev was RED on both Windows jobs for four consecutive commits, and V1 owns this repo, so it outranked the queue.**
`t.Setenv("HOME", dir)` does not redirect `os.UserHomeDir()` on windows — it reads USERPROFILE there, `$home` on plan9 — so four arms across `internal/ai` and `internal/eval_harness` failed **for the platform, not for the code**. The same private helper had already been written **three** times in this repo; the fourth call site went red anyway. Swept to one `testutil.SetHomeDir` (16 bare sites → 1) and gated with `make check-home-isolation`, wired into code-health.mk, `ci:` **and** ci.yml. PR [#1025](https://github.com/sunholo-data/ailang/pull/1025).

## Up next (banked queue head)
1. `m-spawn-pin-enforcement` — **now visible to unattended picks for the first time**, since Mark's attended merge put it on origin (this is what D-54 was about). Design approved attended 2026-09-01.
2. `m-probe-discovery-default-30s-unpinned` — a production-path tightening nobody chose and no test pins (mutant 30→5 passes 42/42).
3. `m-docparse-v0340-reports-2026-09-01` — VERIFY-then-route; a live consumer's silent export drop, already failed to reproduce in two shapes.

## Loop health / routing
- Controller `claude:claude-opus-5` · executor `codex:gpt-5.6-sol` (probe rc=0, one sandboxed 30-min-capped run) · evaluator `sonnet` in its **own** worktree, two rounds (generator≠judge holds) · designer rotation at `claude:claude-fable-5`, **did not run** (a dev-red fix-forward needs no design doc).
- Per-gate `mission-heartbeat.sh` stamps in use (D-52).
- **The running skill is byte-identical to origin for the first time in at least four iterations** — Mark's attended merge cleared the main checkout's divergence (0 ahead / 0 behind, against 22/31 at iteration 319's Gate 4).
- **The judge earned its slot twice.** It broke the new gate with a gofmt-canonical multi-line call the line-oriented matcher could not see, and it found a live unswept instance (four `os.Setenv("HOME")` sites reaching `os.UserHomeDir()`). Both closed in round 2.

## Parked on Mark
- **D-53 (OPEN)** — rule on the 4 UNCLASSIFIED docs (N=10 vs N=14). Loop recommends N=12. Default: keep reporting 10 with the bucket named.
- **D-54 — RESOLVED**, answered "D-54 b" on #972 at `07:17:34Z`. The loop may now branch the main checkout's unpushed `dev`, push and open a PR. Mark then cleared the divergence himself 21 minutes later, so the grant is standing rather than pending.

## Cost posture
Metered **$0.00** of the $5 iteration ceiling. Every lane a quota bucket; no quorum round, no designer, no planner.
