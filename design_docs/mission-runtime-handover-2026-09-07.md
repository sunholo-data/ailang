# Mission runtime handover — 7 September 2026

> ## STATUS BLOCK — added 2026-09-08 (attended, Mark). READ THIS FIRST.
>
> This document was written on 2026-09-07 and was **still untracked** until now, which is why
> it is being committed rather than rewritten: the body below is preserved verbatim as the
> record of what was true then. Four of its statements are no longer true.
>
> | Body says | Now |
> |---|---|
> | "The four live loops are NOT fully orchestrated by the binary" (§Where we actually are) | Still true of the ARCHITECTURE, but the four loops are **PAUSED** — kill switches armed on all of v1/world/docs/motoko, zero mission processes running |
> | All four envs pin `AILANG_DRIVER_REF=48c4a6e49…` with `…-ollama-gauge-48c4a6e49` pin dirs (§Merged code) | **Removed.** `D-61` ruled (A) attended 2026-09-08; pins returned to the `origin/dev` default after an ancestor check, 100 commits stale by then. `AILANG_DRIVER_PIN=1` unchanged, so pinning is still ON |
> | "Last observed fleet status — 17:02 Copenhagen, 7 September" (V1/World running, Docs/Motoko idle) | Superseded. World's final iteration was **KILLED at gate-3** by its own stall watchdog 2026-09-08 09:32; V1 was terminated attended; all four are down |
> | Codex 32% / Ollama session 3.2%, weekly 36.1% | Superseded. Ollama weekly reached **69.4%** by 09-08 09:06 — and ollama and anthropic are now **rationed**, which they were not when this was written |
>
> Also changed since: `test-windows`, `check-git-exec` and `check-home-isolation` were all red
> on dev when this was written and are now green; the seven stranded agent PRs are resolved;
> every mission ledger is at **zero** open decisions.
>
> The §Recommended next phase section is UNCHANGED and still the plan. What it asks for is
> unaffected by any of the above.
>
> Companion document: [`planned/m-mission-loop-lifetime-audit.md`](planned/m-mission-loop-lifetime-audit.md)
> — the full-lifetime audit of all four loops, whose §3.1 ("declared but not walked", found
> four separate times) is the single finding the migration should design against.


## Purpose and next-session instruction

Continue the mission runtime project: make `ailang` the preferred, dependable way to
coordinate long-running projects across providers. The user wants a workflow such as
`ailang mission add` in any folder/project/repository, followed by useful autonomous work.
**`mission add` is a desired interface, not an implemented command.**

Read `CLAUDE.md` and repository instructions first. This document is a handover and a
proposed scope for the next phase, not an approved new sprint. Inspect the implemented
interfaces and live state before designing their replacement or extending them.
Do not restart active missions merely to bring them onto a newer commit.

The user's overriding concern: too much time and quota goes into repairing the harness
instead of delivering project work. Keep the next phase bounded, reuse existing tools,
and demonstrate useful work through the migrated path before expanding it.

## Where we actually are

**The runtime is still hybrid. The four live loops are NOT fully orchestrated by the binary.**

| Responsibility | Current implementation |
|---|---|
| Mission registration/configuration, doctor, staging/application | `ailang mission` commands and `missions/*.toml` |
| Provider quota observations and admission verdicts | Binary, invoked by shell driver |
| Explicit role dispatch and durable attempt fencing | New binary interfaces; not yet fully adopted by live outer loops |
| Scheduling, outer iteration, role probing/fallback wiring, watchdogs | launchd, `tools/launchd/mission-control.sh`, helper scripts |
| Design/planning/execution/evaluation methodology | Skills and mission documents |
| Actual model execution | Claude/Codex/Pi and provider APIs |
| Mission records and attended decisions | Mission documents, logs, weekly GitHub threads |

External model executables and an OS scheduler can remain dependencies. The migration
objective is to move orchestration/state transitions into the binary, not to pretend
that the binary contains the providers themselves.

## Merged code and deployed code are different facts

PR [#1082](https://github.com/sunholo-data/ailang/pull/1082) merged into GitHub `dev`
at **`8e392795052cdf829e8b2e9b0890e0974911af66`**, 2026-09-07 14:15:07 UTC.
Local `dev` was fast-forwarded to that commit. It contains the runtime work, quota guards,
attended rulings incorporated during integration, and final CI fixture repairs.

The installed binary and all four **next-fire pins** were deployed earlier from
**`48c4a6e4975632e1ac3c1452ebbb55e2de52c80f`**. Do not claim that merging the PR
upgraded already-running processes or installed the final merge commit.

- Binary: `/Users/voightkampff/go/bin/ailang`; `~/.local/bin/ailang` is a symlink.
- Deployed binary SHA-256: `179c4f42cd27f8ae7225ff0b9b1dcb280459d2d37942df2e80dddb54130d55b4`.
- Mission envs: `~/.config/ailang/mission-{v1,world,motoko,docs}.env`.
- Each sets `AILANG_DRIVER_PIN=1`, the full revision above, and its own
  `~/.ailang-driver-pin/<mission>-ollama-gauge-48c4a6e49` directory.
- Shared secrets: `~/.config/ailang/secrets.env`. The existing `OLLAMA_API_KEY` was
  added there so unattended quota reads work. **Never print/copy credentials into docs.**
- Atomic binary replacement and configuration backups are recorded in
  [deployment evidence](verification/mission-recovery-2026-09-07/ollama-gauge-deployment.json).
- No active iterations were restarted for the quota deployments.
- A source-newer-than-binary warning is expected after the later merge; it is not
  evidence that the deployed gauge correction is absent.

The review worktree is `/private/tmp/ailang-mission-recovery`, branch
`sprint/mission-runtime-recovery`. It is temporary; durable evidence lives in the repository.

## Implemented building blocks

1. **Explicit role requests and receipts.** `ailang mission role-run` resolves model,
   harness and wire identities, applies declared author-vendor exclusions for evaluation,
   allows fallback only before execution, and writes exclusive synced receipts.
   Execution success is explicitly distinct from artifact acceptance.
2. **Durable attempt fencing.** Optional coordinator SQLite state provides atomic
   admission, ownership/leases, start/renew/complete, cancellation and reconciliation.
   Ambiguous running work is never automatically reclaimed/re-executed.
   CLI: `ailang mission attempt status|reconcile|cancel`.
3. **Pi token guard.** Cumulative token limits are enforced in the Pi execution path.
4. **Safe driver pinning.** World retains its different-origin project repository;
   same-origin fleet clones can execute against pinned source. This fixed the confirmed
   World repository redirect defect, tracked in #1075.
5. **Subscription admission.** Shared shell probe gates consult the binary before
   controller/role inference. Controller pins and one-shot executor overrides cannot
   bypass protected-provider admission. Blocked one-shots remain deferred.

Principal implementation paths:

- `internal/mission/dispatch/`
- `internal/coordinator/mission_attempt.go`
- `cmd/ailang/mission_role_cmd.go`, `mission_role_state.go`, `mission_attempt_cmd.go`
- `internal/mission/codex_quota.go`, `ollama_quota.go`
- `cmd/ailang/mission_quota_cmd.go`
- `tools/launchd/mission-control.sh`, `tools/launchd/lib/pin-root.sh`
- `.pi/extensions/provider-quota.ts` and its embedded `cmd/ailang/pi_assets/` copy

## Quota policy and lessons that must not be lost

### Codex

The guard reads provider rate-limit snapshots from local Codex rollout logs under
`CODEX_HOME` (default `~/.codex`). It uses bounded scans and the latest valid observation,
not a sum of snapshots or a guessed token-to-quota conversion. Missing, invalid or stale
(over 15 minutes) observations block new admission. Weekly pacing is 10% per day,
with a 10% minimum first-day allowance; exhausted short windows also block.

This is new-admission control, not reservation, cancellation of an active role, or a cap
on attended Codex sessions. The attended session and its approval reviews also consumed
substantial Codex usage. Raw token volume does not establish provider quota share.

### Ollama Cloud

**Reuse the existing Pi gauge.** Its hidden read-only endpoint is
`GET https://ollama.com/api/usage`, authenticated with `OLLAMA_API_KEY`.
Inference through the local daemon uses separate device authentication; do not assume
that credential presence alone proves both credentials belong to the same account.

The original implementation mistakenly required manual capacity/reset metadata for any
admission. The user pointed out the existing extension. V50 in the implemented Ollama
provider design measured session usage `0.613 → 0.792 → 0.972 → 1`, then HTTP 429.
That supersedes the old numerator-only interpretation. The extension applies the same
fraction convention to weekly usage; **V50 did not separately measure weekly exhaustion**.

Current policy matches Pi: below 80% is OK, 80% warns but admits, and 95% or higher in
**either** window blocks new cloud work. Missing key, HTTP failure or malformed/unsupported
quota schema remains unknown/blocked. The reader is bounded and makes no inference call.

Reset-aware pacing is optional and currently unconfigured. If explicitly present,
`~/.ailang/state/ollama-quota-limits.json` is strictly validated and credential-bound;
it cannot relax the basic 95% cutoff. Do not invent reset times, or claim daily pacing
from the gauge alone. The legacy window schema must not silently be applied to a future
monthly-credit account schema. Local Ollama model routes remain outside the cloud guard.

**Documentation trap:** earlier sections of `m-ollama-quota-observation.md` and the
implemented provider doc retain historical claims. Read the correction/addenda and V50,
not merely the first matching paragraph. Pi extension comments were reconciled.
After changing `.pi/extensions`, run **`make pi-assets` and `make verify-pi-assets`**.

## Last observed fleet status — 17:02 Copenhagen, 7 September

This is a snapshot, not a live assertion. Refresh it before acting.

| Mission | Observed process/run | Last heartbeat or completion |
|---|---|---|
| V1 | launchd PID 86069; iteration began 14:35, driver `8e5899aff` | Gate 4 at 16:35; still running |
| World | launchd PID 62887; iteration began 16:26, corrected driver `48c4a6e49` | Gate 2 at 16:31; still running |
| Docs | Idle; previous iteration used older `ccd2b4d36` driver | Completed rc=0 at 15:42 |
| Motoko | Idle; previous iteration used older `878939117` driver | Completed rc=0 at 11:00 |

Both active runs selected Claude Opus controller, Fable designer, Ollama Kimi planner,
Ollama DeepSeek executor and Sonnet evaluator. World specifically demonstrated the
corrected deployment: Codex was blocked before its inference probe, Ollama admitted,
and the work repository remained `ailang-world`.

At 17:02, quota CLI reported Codex weekly usage **32%**, allowance **10%**, state `over`;
Ollama session **3.2%**, weekly **36.1%**, state `ok`. Codex was over our ration, not
necessarily provider-exhausted. Coarse gate heartbeat age alone does not prove a stall.

Schedule registry: V1 keepalive/5400s; World keepalive/14400s; Docs interval/21600s;
Motoko interval/46800s. These are not promises of an exact next-start time.

### Remaining observed operational defect

Repeated degradation notifications were rejected as duplicates by `ailang messages`.
V1 also reported undeliverable spooled notices. Runs continued, but retry/deduplication
semantics are not robust yet. Investigate using the actual duplicate response and spool;
do not blindly add `--force` and flood the message plane. No real notices were sent or
acknowledged by the status checks in this session.

## Decisions and authorization

The user authorized the runtime work, deployment, making attended rulings, and merging
onto GitHub `dev`. All 13 then-open rulings were recorded across the AILANG and World
ledgers. Evidence: [decision review](verification/mission-recovery-2026-09-07/decision-review.md).
This is not a claim that no new decisions have arisen since; old STATUS prose often
says OPEN after a ledger ruling, so inspect the canonical decision table.

Future Cloud Run/distributed role execution and integration with `ailang messages` are
part of the vision. They are not the implementation target for the immediately next phase.
Do not turn the next sprint into a cloud deployment project.

## Recommended next phase: binary-owned iteration, one canary

First produce a focused design and acceptance plan for moving one live mission through
a binary-owned outer iteration. Treat the following as a proposal to review with the user.

1. **Inventory current behavior, then freeze the contract.** Map the shell's admission,
   role transitions, fallback, heartbeat, cancellation, failure/ambiguity and completion
   rules onto the existing request/receipt and coordinator attempt interfaces. Identify
   which legacy paths can then be removed; avoid a second parallel vocabulary/state store.
2. **Implement one binary iteration path.** Persist transitions before/after dispatch;
   require evidence for accepted outputs; make recovery explicit for ambiguous work.
   Keep provider adapters and external executables behind the existing dispatch boundary.
3. **Add an approachable project entry point.** Design `mission add`/initialization plus
   status/start/stop/recovery around a project-local manifest and explicit scope. Reuse
   registry/config rendering. Define overwrite behavior and secret references; do not
   copy machine-specific paths or credentials into project files.
4. **Migrate one mission, observe useful work, then expand.** Keep rollback available.
   Prove the live path actually invokes durable binary dispatch before declaring adoption.
   Compare useful completed tasks, harness interventions, quota consumption and duplicate
   work against the legacy path. Expand to the remaining missions after the canary passes.

Acceptance should include: repeat-start admission cannot duplicate work; process death
before/after dispatch produces the right recoverable state; cancellation fences late
completion; provider unavailability triggers fallback only before execution; exhausted or
unknown subscription quota cannot leak through an override; a foreign project stays in
its own repository; completion includes validated artifacts; status clearly distinguishes
running, waiting, quota-blocked, decision-blocked and reconciliation-required.

Do not claim exactly-once external execution, hard quota reservations, verified artifact
acceptance, power-loss durability, full lifecycle migration or distributed-worker safety
until each has an implemented and tested contract. These remain gaps in the current work.

## Verification and merge lessons

- Deployed quota correction passed full local tests/lint/build, focused quota/routing
  tests and independent review. Usage-only live verification was performed without inference.
- Integration with current `dev` passed full local tests and lint.
- CI revealed inherited launchd fixture failures. Fixed ordered trace observation across
  command substitution without weakening the 27 original notification assertions, and
  supplied missing `MC_PAUSED=0` in the heartbeat fixture. Full Bash 3.2 suite passed.
- CI also caught a stale embedded Pi extension copy. Synced with `make pi-assets`.
- All required GitHub gates (`test`, `lint`, `build`, `docs-gate`) passed before normal
  automatic PR merge. No admin bypass or branch-protection change was used.
- SonarCloud separately flagged seven `go:S2077` findings in `mission_attempt.go`.
  Reviewed every site: only fixed `missionNow`/`missionWhere` constants are concatenated;
  inputs use SQL parameters. Apparent false positives, **not dispositioned in SonarCloud**.
  `SONAR_PAT` was unavailable in the session. This check was not a required merge gate.
- `make test`/`make lint` alone do not cover every CI gate. Use relevant asset and shell
  checks before pushing, rather than discovering one missed gate per CI cycle.

## Workspace caution at handover creation

Local `dev` HEAD: `8e392795052cdf829e8b2e9b0890e0974911af66`.
The tree was clean immediately after merge, but these unrelated edits appeared before
this handover was written:

- modified `cmd/ailang/coordinator_approvals_remote.go`
- untracked `cmd/ailang/coordinator_approvals_orphans.go`

They were not created or reviewed as part of this handover. Preserve them; do not stage,
commit, discard, or continue them under the mission migration scope without establishing
ownership and authorization. This handover itself is a new document.

## First checks for the next session

Use purpose-built commands first, with bounded execution:

```bash
ailang mission list
ailang mission doctor
ailang mission quota --json
ailang mission role-run --help
ailang mission attempt status --help
```

The doctor's desired-vs-installed comparison may report deliberate immutable deployment
pin differences. Inspect those differences before applying registry configuration.
`mission install` stages; `mission apply` changes installed configuration/reloads launchd.
Do not apply merely to clear a diagnostic.

For live runs, inspect launchd labels `dev.ailang.mission-{control,world,motoko,docs}`,
`~/.ailang/state/mission-<name>-heartbeat`, `mission-<name>-slot-verdicts.log`, and
`/tmp/ailang-mission-{control,world,motoko,docs}.launchd.log`. Bound log reads and truncate
individual lines: some mission documents have enormous single-line historical records.

Canonical inbox: scoped `AILANG_MESSAGES_STORE=gcp` and
`AILANG_MESSAGES_PROJECT=ailang-multivac`; never change `AILANG_STORAGE` for inbox access.
Use unread JSON listing for triage; `messages read` marks read. Do not acknowledge or send
messages without the applicable authorization. Do not launch provider inference merely
to obtain a status report.

## Supporting documents

- [Runtime contract](planned/m-mission-runtime-contract.md)
- [Recovery sprint plan](planned/m-mission-recovery-sprint-plan.md)
- [Codex quota observation](planned/m-codex-quota-observation.md)
- [Ollama quota observation and correction](planned/m-ollama-quota-observation.md)
- [Ollama provider evidence, especially V50](implemented/v0_34_0/m-ollama-cloud-provider.md)
- [Deployment/verification directory](verification/mission-recovery-2026-09-07/)
- [Program north star](PROGRAM.md)
