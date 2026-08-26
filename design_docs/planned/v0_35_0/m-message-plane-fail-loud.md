# M-MESSAGE-PLANE-FAIL-LOUD: the remaining silent seams

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1 — the loud half shipped 2026-08-26; what remains is the half that still
reports success while doing nothing
**Estimated**: 3 days, ~600 LOC + one config-plane change
**Dependencies**: none blocking. Builds on work that landed 2026-08-26 (see *Already Shipped*).
**Author**: Claude Opus 5 + Mark
**Created**: 2026-08-26

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes CWD-dependent daemon behaviour: today the same binary is healthy or inert depending on the working directory launchd happens to give it |
| A2: Replayability | 0 | No trace-format change |
| A3: Effect Legibility | 0 | No language-level effects |
| A4: Explicit Authority | 0 | No change to capability grants |
| A5: Bounded Verification | +1 | A machine can check "is this node actually processing?" locally, without grepping a log |
| A6: Safe Concurrency | +1 | Compare-and-swap on the shared config object closes a demonstrated lost-update |
| A7: Machines First | +1 | Degraded state becomes a structured field, not a one-line English Warning emitted once at startup |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Stops burning Cloud Run executions and GPU-hours on work that cannot succeed |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | +1 | The core of the doc: a component that cannot do its job says so in a typed way instead of no-opping |
| A12: System Boundary | +1 | Makes the local/cloud execution-lane boundary explicit rather than implied by a string's shape |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — this *removes* some
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): strictly improves machine-readability of degraded state

---

## Problem Statement

On 2026-08-26 a full triage of the prod message plane found 41 feedback tickets going back to
2026-04-27, of which the newest 19 had never been seen by anyone. Investigating why produced
seven distinct defects. **Not one of them was a crash.** Every single one was a component that
was alive, reporting healthy, and doing nothing:

| Symptom | What it looked like |
|---|---|
| Rig coordinator in standby | `▶ running`, health port answering, `Checking for new tasks...` every 30s, forever |
| Unrouted inbox | Task created, job dispatched, job died, message acked, publish CLI printed success |
| Phantom inbox (`ailang-parse` with a hyphen) | Feedback accepted and stored; nobody watching |
| Missing Firestore index | Polled correctly every 30s; every poll failed; message sat unread |
| Malformed/no-op dispatch | `exit 0`, banked `completed`, `files=0`, PR 422 |
| Capability manifest | Advertised a `submit_feedback` tool that was dead |
| Session-start hook | Reported "16 unread" when the canonical store held 74 |

The project's own [Critical Principle 2](../../../CLAUDE.md) is **"NO SILENT FALLBACKS — FAIL
LOUDLY."** The message plane violated it at nearly every hop. The architecture described in
[message-plane-topology.md](../../../docs/internal/message-plane-topology.md) is sound; the gap
is that its seams degrade quietly.

The standby case is the sharpest illustration. `initTaskProcessing` resolves a git root from
**CWD**; launchd's `WorkingDirectory` was `$HOME`, which is not a checkout. So the daemon
dropped into standby — announced once, as a Warning, and never again. Log evidence shows **20
standby startups against 11 healthy ones, first on 2026-05-25**. The same binary was healthy
when launched from a terminal inside the repo and inert under launchd, which is exactly why the
topology doc could honestly record "verified end to end 2026-08-26" from a manual run while
production claimed nothing for three months. Cloud then absorbed `eval-rig` work it structurally
cannot run — no GPU, no ollama — and failed 10 consecutive times on a Mac Studio filesystem path
where a clone URL belongs.

### Already shipped 2026-08-26 (do NOT re-implement)

Substantial parts of this problem were fixed the same day. Verified in-tree at HEAD:

- **Empty-`AgentID` refusal** — `resolveInboxAgent` ([daemon_tasks_polling.go:34](../../../internal/coordinator/daemon_tasks_polling.go#L34)),
  wired at both call sites (`:77`, `:375`); unrouted inboxes now leave messages unread for triage.
- **Unreachable-inbox backstop** — `NormalizeInboxRouting` reroutes `ToInbox == ""` to a real
  `unrouted` inbox ([inbox_routing.go](../../../internal/messaging/inbox_routing.go)).
- **Hyphen package coordinates refused** — `packageRe` + `hyphenPackageRe` (`c9142e3bc`).
- **Empty-branch push on a no-op task** — `0fd7a31d3`.
- **Completion notices no longer re-queued as work** — `e49ed32f3`.
- **Session banner reads the canonical store** — `39e869e98`.
- **Rig standby cause on this host** — `39280bc16` (WorkingDirectory → checkout).
- **Firestore composite indexes** — created via laptop; verified `READY` with matching field
  directions, and a coordinator restart at 16:37 produced **0 `FailedPrecondition`**.

This doc covers only what those did **not** close.

---

## Goals

**Primary goal**: no component in the message/dispatch plane can be simultaneously *reachable*
and *not doing its job* without saying so in a machine-readable way.

**Success metrics**:
1. A coordinator that cannot process messages fails its health check — `ailang coordinator status`
   and `/health` both report `degraded` with a reason, instead of `▶ running`.
2. Two machines editing the shared prod config cannot silently lose each other's writes.
3. `workspace` has exactly one meaning; the local/cloud execution lane is chosen by an explicit
   field, not by inferring intent from whether a string starts with `/`.
4. Every configured inbox has either an agent or an explicit `triage_only: true` — "neither" becomes unrepresentable.
5. Every claim in this list is verifiable by a command, and each has a regression test.

---

## High-Impact Decisions

| # | Decision | Options | Who decides | Cost to change later |
|---|---|---|---|---|
| # | Decision | **RATIFIED (Mark, 2026-08-26)** | Cost to change later |
|---|---|---|---|
| D1 | Standby behaviour | **(a) FATAL EXIT.** `initTaskProcessing` failure exits non-zero. launchd `KeepAlive.SuccessfulExit=false` restarts it, `ThrottleInterval: 10` bounds the loop, and a daemon that cannot work is never reachable-but-inert. | Low — one branch in `Run()` |
| D2 | `public-feedback` routing | **(b) EXPLICIT `triage_only: true`.** No agent — anonymous input is never handed to something that acts on it. Discord is the routing, and the marker makes "no agent" mean *intended* rather than *forgotten*. | Medium — new config field |
| D3 | Execution lane selection | **(a) EXPLICIT `execution_lane: local\|cloud`.** Inference from `workspace` shape is what produced the 3.5-hour worktree loop. | High once agents depend on it |

### Design Freeze

- [x] **D1 decided** — fatal exit
- [x] **D2 decided** — explicit `triage_only`, Discord is the routing
- [x] **D3 decided** — explicit `execution_lane`

**All three ratified 2026-08-26. Sprint may proceed on all phases.**

#### D2 addendum — verified, and it narrows the work

`public-feedback` reaching Discord is **not** an assumption. Verified 2026-08-26 in
`/tmp/ailang-daemon.log`, daemon PID 50131 running `--env prod`, `channels=[discord macos]`:

```
09:02:24  delivered fb_1cce034a84df5cec [from=mcp-public, inbox=public-feedback] -> "🌐 External feedback"
12:03:31  delivered fb_f7ecc535fde19c8e [from=mcp-public, inbox=public-feedback] -> "🌐 External feedback"
12:04:10  delivered fb_c1cf1a339764a683 [from=mcp-public, inbox=public-feedback] -> "🌐 External feedback"
```

All ten `pkg:sunholo/ailang-parse` tickets were delivered the same way at 09:02. `humanTriageInbox`
([handlers.go](../../../internal/daemon/handlers.go), commit `926901474`) already covers
`public-feedback`, `user`, and `pkg:*`.

**Consequence:** the visibility half of D2 already works. The remaining work is only to make the
*intent* legible in config — today "no agent" is indistinguishable from "agent forgotten", which is
what sent this session chasing a phantom routing gap. #900 must be corrected, not implemented:
the 36 dispatch failures were real, but the fix was the `resolveInboxAgent` refusal (shipped), not
registering an agent.

---

## Solution Design

### Overview

Three independent workstreams, ordered by ratio of harm-prevented to risk:

1. **Make degraded states loud** (no config change, no cross-lane impact)
2. **Make the config plane safe to write** (closes a demonstrated lost update)
3. **Disambiguate the execution lane** (needs D3; touches shared agent config)

### Architecture

**Phase 1 — degraded states are structured, not a log line.**

`standby` currently appears in exactly one place in the entire codebase: a `log.Println` at
[daemon.go:349](../../../internal/coordinator/daemon.go#L349). There is no health field, no
status surface, no metric. Add a `DaemonState` with a reason, surfaced through three existing
readers — `/health`, `ailang coordinator status`, and the heartbeat — so a monitor, a human, and
another agent all see the same thing.

Also close the two constructors that accept impossible inputs without complaint:

- `NewWorktreeManager(repoDir, …)` validates `repoDir` **only when it is empty** (it calls
  `findGitRoot`). A supplied-but-nonexistent path is accepted, logs `Worktree manager ready`,
  and fails later on every `CleanupOrphaned` — which is how the rig logged
  `chdir sunholo-data/ailang` every 30s for 3.5 hours while claiming to be ready. Validate at
  construction.
- Startup should assert its own preconditions (CWD is a git repo; every configured agent's
  `workspace` resolves) rather than discovering them per-poll.

**Phase 2 — the config plane gets concurrency control.**

`gs://ailang-multivac-ailang-config/config.yaml` is written with plain `gsutil cp`. There is no
`if-generation-match` anywhere in the repo (verified: zero hits). Demonstrated live during this
session: a correct edit uploaded at 14:24:33Z was clobbered at 14:37:10Z by a copy fetched
before it, restoring a byte-identical earlier version. **No error, no warning, on either side.**

Ship an `ailang coordinator config` command that does read-modify-write with a generation
precondition, validates the YAML, diffs before writing, and refuses on mismatch — telling the
caller to re-fetch. Two machines then cannot silently lose each other's work.

**Phase 3 — one field, one meaning.**

`workspace` is declared "Base directory for worktrees"
([agent_registry.go:105](../../../internal/coordinator/agent_registry.go#L105)) and is chdir'd
into by the worktree manager. Cloud dispatch *also* reads it, deriving a repo URL when it looks
like `org/repo` ([daemon_tasks_exec.go:14](../../../internal/coordinator/daemon_tasks_exec.go#L14)).
Setting it for one consumer breaks the other, which is precisely the loop that ran on the rig.

Per D3(a): add an explicit `execution_lane` and a separate `repo` coordinate, so a bare-metal
worker is bare-metal because it *says so*, not because its path starts with a slash.

### Implementation Plan

**Phase 1 — loud degradation (1 day, ~250 LOC)**
- `DaemonState{Healthy|Degraded, Reason}` set where `initTaskProcessing` fails
- surface in `/health`, `ailang coordinator status`, heartbeat payload
- `NewWorktreeManager` validates a supplied `repoDir` (exists, is a dir, is a git repo)
- startup precondition assertions with actionable messages
- D1 branch: exit non-zero, or stay up degraded

**Phase 2 — config CAS (1 day, ~200 LOC)**
- `ailang coordinator config get|set|diff` with `if-generation-match`
- YAML validation + agent-invariant checks (every agent has a workspace; no duplicate inboxes)
- refuse-on-mismatch with a re-fetch hint

**Phase 3 — lane disambiguation (1 day, ~150 LOC + config)**
- `execution_lane` + `repo` fields, both optional with back-compatible inference
- deprecation warning when a bare `org/repo` `workspace` is used as a repo coordinate
- migrate the `eval-rig` and package agents

### Files to Modify/Create

- `internal/coordinator/daemon.go` — DaemonState, D1 branch (~40 LOC)
- `internal/coordinator/daemon_http.go` — `/health` reports degraded (~30 LOC)
- `internal/coordinator/worktree.go` — validate supplied repoDir (~30 LOC)
- `internal/coordinator/agent_registry.go` — `execution_lane`, `repo` fields (~50 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — lane-aware repo derivation (~40 LOC)
- `cmd/ailang/coordinator_config.go` — **new**, CAS config command (~200 LOC)
- `cmd/ailang/coordinator_status.go` — surface degraded state (~30 LOC)
- `tools/launchd/dev.ailang.coordinator.plist.template` — already carries the CWD fix (`39280bc16`)

---

## Examples

### Example 1: a coordinator that cannot work says so

```
# today — indistinguishable from healthy and idle
$ ailang coordinator status
  State:      ▶ running
  PID:        93739

# proposed
$ ailang coordinator status
  State:      ⚠ degraded — no message processing
  Reason:     default worktree manager: CWD /Users/voightkampff is not a git repository
  Since:      2026-08-26T16:04:53Z
  Fix:        set WorkingDirectory to a checkout in dev.ailang.coordinator.plist
```

### Example 2: a concurrent config edit is refused, not lost

```
$ ailang coordinator config set --agent pkg-sunholo-ailang-parse --workspace sunholo-data/ailang-parse
✗ refused: config changed since you fetched it (yours: 1787754273571345, live: 1787755030049108)
  Someone wrote at 14:37:10Z. Re-fetch and re-apply:
    ailang coordinator config get > cfg.yaml
```

---

## Success Criteria

- [ ] A coordinator started outside a git repo reports `degraded` on `/health` and `status`
- [ ] `NewWorktreeManager` with a nonexistent `repoDir` returns an error at construction
- [ ] A config write against a stale generation is refused with the live generation named
- [ ] `eval-rig` has an explicit lane; no eval-rig task is dispatched to Cloud Run
- [ ] `public-feedback` carries `triage_only: true`; an inbox with neither an agent nor the marker is a startup error
- [ ] Regression test per row in the Verification Log
- [ ] All tests passing; docs updated

---

## Testing Strategy

The 19 triaged feedback messages are the end-to-end fixture set: each exercises a different hop.
Specifically —

- **Unit**: worktree-manager validation; DaemonState transitions; CAS precondition refusal
- **Integration**: a coordinator booted in `$HOME` must report degraded (this is the regression
  test for the three-month standby)
- **End-to-end**: replay the 9 transferred parse tickets
  ([ailang-parse#17–#25](https://github.com/sunholo-data/ailang-parse/issues)) through the parse
  agent once its config is corrected, and confirm each produces a real artifact or a typed refusal
- **Negative**: a stale-generation config write; a hyphen package coordinate (already covered by
  `c9142e3bc` — assert it stays covered)

---

## Deferred Decisions

Agent latitude, no ratification needed:
- Exact wording of degraded-state reasons
- Whether `coordinator config` wraps `gsutil` or the Go GCS client
- Whether the heartbeat carries the full reason or a code

---

## Non-Goals

- **Rewriting the routing architecture.** It is sound; this is about its seams.
- **Cross-host `workers list`.** A known limit (heartbeats are local JSON); tracked separately.
- **The ailang-parse service defects.** Nine issues, own repo, own backlog.
- **The feedback-gate classifier failing closed on the rig** (no `ANTHROPIC_API_KEY`) — documented
  limit, separate decision.

---

## Timeline

| Day | Work |
|---|---|
| 1 | Phase 1 — loud degradation + worktree validation |
| 2 | Phase 2 — config CAS |
| 3 | Phase 3 — lane disambiguation (needs D3), migrate agents, e2e replay |

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| D1(a) turns a degraded daemon into a restart loop | `ThrottleInterval: 10` already set; prefer D1(b) if the loop is noisy |
| `execution_lane` migration breaks other package agents | Optional field, back-compatible inference, deprecation warning first |
| Config CAS blocks an urgent manual fix | `--force` retained, but it must name the generation it is overwriting |
| Health-check change trips external monitors | Degraded is a new state; keep `running` semantics for the healthy path |

---

## Verification Log

Every load-bearing claim, with the command that proves it. Run 2026-08-26 at HEAD `39280bc16`.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | Empty-`AgentID` refusal already exists | read `daemon_tasks_polling.go:34`, grep call sites | **Confirmed shipped** — `:77`, `:375`. NOT re-proposed |
| V2 | Hyphen package coordinates already refused | read `internal/feedback/publisher.go` | **Confirmed shipped** — `packageRe`, `c9142e3bc`. NOT re-proposed |
| V3 | Empty-branch push already fixed | `git show 0fd7a31d3` | **Confirmed shipped**. NOT re-proposed |
| V4 | `standby` has no health surface (**negative**) | `grep -rn "standby" internal cmd` | Confirmed — exactly 1 hit, `daemon.go:349`, a log line |
| V5 | No generation precondition anywhere (**negative**) | `grep -rn "if-generation-match\|IfGenerationMatch" internal cmd tools` | Confirmed — zero hits |
| V6 | Config lost-update is real, not theoretical | `gsutil ls -la` generations + `cmp` | Confirmed — 14:24:33Z write clobbered at 14:37:10Z, restored copy byte-identical to 05:54Z |
| V7 | `NewWorktreeManager` accepts a bad non-empty path | read `worktree.go:38-46` | Confirmed — `findGitRoot` runs only when `repoDir == ""` |
| V8 | `workspace` has two consumers | read `agent_registry.go:105`, `daemon_tasks_init.go:151`, `daemon_tasks_exec.go:14` | Confirmed — declared as worktree base; also used as repo coordinate |
| V9 | Standby predates today by months | `grep "standby mode" coordinator.log` by date | Confirmed — 20 standby vs 11 healthy startups, first 2026-05-25 |
| V10 | Firestore indexes now satisfy both queries | `gcloud firestore indexes composite list` + restart | Confirmed — `stage ASC/created_at DESC` and `status ASC/created_at ASC` READY; restart at 16:37 gave 0 `FailedPrecondition` |
| V11 | `public-feedback` still unrouted | `grep public-feedback` on live config | Confirmed — no agent entry |
| V12 | Parse agent still misconfigured | `gsutil cat` live config | Confirmed — `workspace: sunholo-data/ailang-packages`; the 14:24 fix was clobbered (V6) |
| V13 | Studio lane works end to end | `task-6051f916` in coordinator.log | Confirmed — poll → claim → worktree → run → envelope → awaiting approval. **Single observation, on a no-op probe** |

| V14 | All 39 cloud agents use `org/repo` workspaces | parse live config, count workspace shapes | Confirmed — 39 of 39 `org/repo`, 0 local-path. Phase 3 inference MUST default these to `cloud` |

**Explicitly NOT verified**: that the Studio lane completes a task which actually *changes files*.
V13 is one observation on an acknowledge-only probe. Treat the lane as unproven for
file-mutating work until Phase 3's e2e replay.

---

## Conflict Surface

Not strictly required — this touches no parser/typechecker/codegen path. Written anyway, per the
skill's guidance, and it surfaced two things:

1. **`resolveInboxAgent` is now load-bearing for other lanes.** The motoko mission and the
   package-cascade both rely on unrouted inboxes leaving messages unread. A D2(a) triage agent on
   `public-feedback` changes that inbox from "left unread" to "claimed" — anything that currently
   *reads* `public-feedback` expecting unread messages would go blind. D2(b) avoids this.
2. **`execution_lane` inference must not reclassify existing agents.** 39 agents currently carry a
   bare `org/repo` workspace and are cloud-dispatched. Inference must default those to `cloud`,
   or Phase 3 silently moves the whole fleet to a local lane that does not exist on Cloud Run.

**Programs that must still work post-change** (all verified present):
- `ailang coordinator status` against a healthy daemon
- `ailang messages list --unread` against the canonical store
- a `pkg:sunholo/motoko_ext_*` wildcard dispatch
- an `eval-rig` local claim (`task-6051f916` is the reference trace)

---

## Related Documents

- [message-plane-topology.md](../../../docs/internal/message-plane-topology.md) — the intended
  design; this doc closes the gap between it and measured behaviour
- [cloud-coordinator-config.md](../../../docs/internal/cloud-coordinator-config.md) — agent config reference
- [m-coordinator-inbox-wildcards.md](../v0_29_0/m-coordinator-inbox-wildcards.md) — wildcard routing;
  scoped only `pkg:*`, which is why `public-feedback` was never covered (`9c8e0de33` corrected the
  doc's misleading "Implemented" status)
- [m-pkg-feedback-loop.md](../v0_29_0/m-pkg-feedback-loop.md) *(0.29 neural)* — validates the package
  feedback loop end-to-end. **Distinct**: that doc validates the happy path; this one makes the
  failure paths loud.
- [m-eval-rig-reliability.md](../v0_29_0/m-eval-rig-reliability.md) *(0.28 neural)* — rig
  post-mortem. **Distinct**: eval-harness reliability, not the message/dispatch plane.

## References

- GitHub [#900](https://github.com/sunholo-data/ailang/issues/900) — public-feedback unrouted
- [ailang-parse#17–#25](https://github.com/sunholo-data/ailang-parse/issues) — the e2e fixture set
- Commits: `39280bc16`, `39e869e98`, `c9142e3bc`, `0fd7a31d3`, `e49ed32f3`, `9c8e0de33`

## Future Work

- Cross-host `workers list` (Firestore heartbeats)
- A periodic self-audit: every configured inbox has an agent or an explicit triage-only marker
- Extend the loud-degradation pattern to the notify daemon and the MCP capability manifest
