# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Last iteration:** 312 · 2026-09-01 · LANDED · [PRODUCT]

## Latest release
v0.34.0. `dev` is 265 commits past the tag.

## In flight
- **`m-registry-interface-hash-blind-to-signatures`** (IN-SPRINT, Sprint 1 of 2).
  M1 landed iter-311, **M2 landed iter-312**. M3–M5 remain (~3d).
  Sprint 2 (irreversible: registry writes, backfill) is DEFERRED by design.
  ⚠ M3 must pass the module's **package root** as `internal-dump-iface <package-dir>`, never the CWD
  (ailang#671 class; pinned by `TestInternalDumpIface_WrongPackageDirFailsLoudly`).

## Next picks (banked, ready)
1. `m-registry-interface-hash-blind-to-signatures` **M3** — subprocess wrapper + `PublishLimits`.
2. `m-registry-validator-unbounded-compile` — public HTTP server compiles untrusted uploads with no
   timeout/cancellation. Confirmed at HEAD; `validate.go:76` uses `exec.Command`, not `CommandContext`.
3. `m-canonical-json-drylink-unpinned` — `DryLink: false` mutant survives the whole suite; M3 will
   call that library function from the publish path. <0.5d.

## Loop health
- Cadence: launchd, ~16 fires/day. Reaped-slot rate ~40% over 296–310 (see D-52).
- Routing this iteration: controller opus · executor `codex:gpt-5.6-sol` (×2) · evaluator sonnet (×2).
  Designer and planner not spawned — doc and plan already existed.
- Metered spend: **$0.00** of the $5 ceiling. All lanes were quota buckets.
- Evaluator rounds: 1 = **FAIL 66/100**, 2 = **PASS 95/100**. generator≠judge held (OpenAI vs Anthropic).

## dev CI
16 checks. Sole non-green: **SonarCloud** — *inherited*, `failure` on 5 consecutive commits
(`origin/dev~0..~4`), two of them docs-only. **Not a required context**
(required = `test`, `lint`, `build`, `docs-gate`). Failing on 72.4% coverage-on-new-code (needs ≥80%)
and B security-rating-on-new-code (needs A). Named, not picked.

## Parked on Mark
- **D-51** — ratify or replace the charter's countable finish-line unit. Loop recommends (b),
  milestone burn-down. Default if unanswered: status quo, every Progress line stays provisional.
- **D-52** — is the ~40% reaped-slot rate worth one iteration to diagnose? Loop recommends (a),
  a per-gate heartbeat artifact. Default if unanswered: (b), nothing changes.

Both filed by iteration 311; neither has been answered. Nothing else is blocked on a human.

## Quota posture
Anthropic available (`MISSION_ANTHROPIC_AVAILABLE=1`). codex probe rc=0. No lane parked on quota.
