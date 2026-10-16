# M-MAC-AGENT-UID: Local Agent CLIs Under a Dedicated macOS Service User

**Status**: Planned — commissioned 2026-10-16 (attended ruling, H-6 D6 = yes) from the
design-doc-creator lane (inbox_1792160182981_f0573def). Ratified by the ruling: **a dedicated
hidden macOS service account for every local agent CLI; the existing `daneel` user (uid 502,
logged-in automation, group `rig`) must NOT be reused.** Everything below that the ruling did
not already fix is a proposal for Mark's approval, not a decision.
**Target**: Phase 0 probe + host account next rig-window after this doc's review; the spawn seam
behind a flag rides the next security release after v0.54.12.
**Priority**: P1 — a live, measured credential-exposure path on the rig (S6 below), but bounded by
"the rig is a single-operator machine" and by no observed exploitation.
**Estimated**: Phase 0 ~1 day · spawn seam + ingestion ~5–7 days · credentials + rollout
~3–4 days · host-ops (account, sudoers, home deny) ~0.5 day. Total ~10–12 engineering days.
**Dependencies**: [M-EXECUTOR-ENV-HARDENING](../v0_49_1/m-executor-env-hardening.md)
(H-4, implemented per its status header, still filed under `planned/v0_49_1/` —
`internal/executor/envpolicy.go` is the env seam this design reuses);
[M-EXECUTOR-UID-SPLIT](../v0_49_1/m-executor-uid-split.md) (H-6 cloud, D4 git-ingestion design
reused here, D6 commissioning ruling); [M-RIG-GPU-ADMISSION-GATEWAY](../m-rig-gpu-admission-gateway.md)
(the ollama lease the agent user keeps using). Separate track from the cloud H-6 split (a Non-Goal
there).
**Created / updated**: 2026-10-16

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to program semantics; agent spawn becomes a deterministic function of (lane, task) — a per-machine flag, not ambient state |
| A2: Replayability | +1 | The child's UID and launcher spec digest are banked on `Result` (audit), so a banked run states which identity the model-facing process held |
| A3: Effect Legibility | +1 | The operator's host authority (keychain, gcloud, gh, ssh, the coordinator's env incl. `COORDINATOR_API_KEY`) stops being implicitly shared with every model-facing process |
| A4: Explicit Authority | +1 | What the agent user may read becomes a stated ownership/ACL table instead of "everything uid 501 can touch" |
| A5: Bounded Verification | +1 | Every protection claim has an adversarial probe with a positive control (the S6 `ps -wwE` sentinel must leak before, be denied after) |
| A6: Safe Concurrency | 0 | Process-group kill semantics are preserved but re-routed (see Conflict Surface); no new shared state |
| A7: Machines First | 0 | Operator-rig infrastructure. Not a language change |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No cost-path change |
| A10: Composability | +1 | Reuses H-4's `BuildEnvironment` unchanged (env), H-6 D4 unchanged (git ingestion), the rig gateway unchanged (ollama lease); adds only the identity layer |
| A11: Structured Failure | +1 | A spawn as the wrong user, a missing launcher, or a sudo failure is a named pre-spawn error, never a silent same-UID fallback |
| A12: System Boundary | +1 | Makes the operator→agent line a kernel-enforced UID boundary instead of a convention — on the Mac the UID is the *only* boundary (no metadata server, no lane SA, no container) |

**Net Score: +7** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects — the design removes a hidden one (ambient operator authority)
- [x] A4 (Authority): no ambient access granted — the design *revokes* ambient access
- [x] A7 (Machines First): no human-convenience trade against machine analysis

## Problem Statement

Every model-facing agent CLI on the Mac Studio rig runs as the operator account
`voightkampff` (uid 501): the coordinator daemon (`ailang coordinator start`, launchd
LaunchAgent, `/bin/bash -lc`), every mission driver (`mission-control.sh` et al., LaunchAgents in
the operator's gui domain), and — the part that matters — every agent CLI those processes spawn
(claude, codex, pi, opencode, motoko; also mission controllers and designers, which are the same
class of process). The agent CLIs are model-facing and run with `--dangerously-skip-permissions`
or equivalent, with bash. Untrusted content reaches their prompts through issue/PR text, benchmark
files and cloned repos.

**Measured (spike S6, 2026-10-16, macOS 26.6):** a uid-501 process can read the **full
environment** of any non-platform uid-501 process via `ps -wwE -p <pid>` (homebrew `python3` with
`H6PROBE=sentinel-x` → visible; only Apple platform binaries are hidden). The agent CLIs (node,
Go) are not platform binaries. Consequences:

| # | Asset | How a local agent reaches it today |
|---|---|---|
| M1 | **The coordinator's environment** — `COORDINATOR_API_KEY` (plaintext in the plist's `EnvironmentVariables`, `dev.ailang.coordinator.plist.template`), plus whatever the login shell profile exported (`bash -lc` runs the full profile) | `ps -wwE -p <coordinator pid>` — **this bypasses H-4 entirely**: H-4's landed `EnvPolicy` filters what the *child* inherits, but the parent's env is one `ps` away |
| M2 | **Mission drivers' environment** — they source `~/.config/ailang/mission-<name>.env` and `~/.config/ailang/secrets.env` (API keys, model routing); v1-mission line 442: `CLAUDE_CODE_OAUTH_TOKEN` lives in `secrets.env` | same `ps -wwE`, against the driver or the coordinator |
| M3 | **Operator keychain + credential files**: `~/.claude/.credentials.json` / login keychain (Claude OAuth), `~/.codex/auth.json`, `~/.config/gh` (the fleet's `gh auth git-credential` push path), `~/.config/gcloud` (operator ADC — an **owner** of the ailang GCP projects), `~/.ssh` | direct file reads — same UID, so 0600 means nothing |
| M4 | **Every checkout under `~/dev`** and all coordinator state under `~/.ailang` | same UID |
| M5 | **Memory of the coordinator** | mostly out of reach on macOS: `task_for_pid` is entitlement-gated; no `/proc`. Not counted as a protected asset — the UID split does not change it |

macOS has no `/proc`, no metadata server and no lane SA, so **the UID is the only real
boundary on this host** — the cloud H-6 doc's D6 ruling (2026-10-16): a dedicated hidden service
account, e.g. `_ailang_agent`, never the existing `daneel` user (uid 502), because agents would
then share a UID with Daneel's own logged-in automation and secrets.

**Impact:** any prompt-injected local agent can exfiltrate the operator's cloud and GitHub
credentials and the coordinator's API key with one shell command. Higher expected value than the
cloud H-6 split (where the metadata server dominates) and cheaper: no spike needed for the
mechanism itself (sudo/sudoers is well-understood); the one unmeasured premise (cross-UID `ps`
denial) is a one-command Phase 0 probe with a built-in positive control.

## Goals

**Primary Goal:** a prompt-injected agent CLI on the rig cannot read the operator's credentials,
keychain, checkouts, coordinator state, or the environment of any uid-501 process. Every
protection claimed has an adversarial probe that proves it.

**Success Metrics:**
- `ps -wwE -p <coordinator pid>` run **as `_ailang_agent`** shows no environment (and the
  same probe as uid 501 still shows the sentinel — the control that makes the denial real).
- `_ailang_agent` reads of `/Users/voightkampff/**` (`.ssh`, `.config/gcloud`, `.config/gh`,
  `.claude`, `.codex`, `.config/ailang/secrets.env`, `~/dev`): **denied**; as uid 501: allowed
  (control).
- Agent runs still work end to end per lane: workspace writes, git commits ingested by the
  coordinator via hardened fetch, ollama via the rig gateway lease, provider auth per lane
  (credential table below).
- Bounded termination preserved: a deadline or coordinator cancellation still kills the agent's
  whole process group (re-routed through the launcher — see D7).
- A run launched as the wrong user, or with the launcher missing: **named pre-spawn error**, task
  failed loudly. Zero silent same-UID fallbacks.

## High-Impact Decisions

The 2026-10-16 rulings are marked **[ruled]**; the rest are proposals for Mark.

| ID | Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----|----------|-----------------|-----------|----------|-------------|
| D1 | **The account**: create `_ailang_agent` — hidden (uid < 500, `IsHidden`), no password, no GUI login, shell `/usr/bin/false`, underscore service-account convention. **Never `daneel`.** Plus one shared group `_ailang_agentio` (members: `voightkampff`) for read-side ingestion | Defines the identity every local agent runs as; reusing any logged-in user would re-create the same-UID hole one UID up | human — **ruled 2026-10-16** (account + not-daneel); group is this doc's proposal | design | med (host-ops) |
| D2 | **Launch mechanism**: one narrow sudoers rule for a fixed, root-owned launcher binary; alternatives weighed below (sudo+`/usr/bin/env`, a root LaunchDaemon spawnd, setuid — rejected). Env crosses via the launcher's stdin spec, never via sudo's environment | Defines what runs privileged-adjacent on the rig forever; a sloppy rule ("run anything as the agent user") or a setuid binary adds surface | human — **ruled 2026-10-16** ("one narrow sudoers rule or an equivalent; weigh alternatives"); specific option A proposed here | design | med |
| D3 | **Credentials: one holder per credential, never share a refresh token** (the cloud model, env-vars.md: `CLAUDE_CODE_OAUTH_TOKEN` is "held by ONE consumer … never both"). Three holders now exist: the operator's interactive logins (keychain), the cloud executors, and the rig's `_ailang_agent`. Per lane: claude = its **own** setup-token; codex = its **own** `auth.json` (subscription lease already enforces one holder) or apikey mode; pi/opencode/motoko = provider keys in env via H-4 grants. The agent user has no keychain and never receives the operator's blobs | Sharing the operator's OAuth between an interactive session and the agent user would have them rotate each other's refresh tokens (the codex lane-lease lesson, `credential_lane.go`); copying keychain blobs defeats the whole split | human | design (per-lane inventory during rollout) | med |
| D4 | **Workspace + git ingestion**: reuse H-6 D4 verbatim — the coordinator (operator UID) **never runs git with the agent's tree as its repository**. It prepares a parent-owned bare clone in a staging area the agent can read, the agent clones/fetches from it, and after the run the coordinator ingests via a hardened fetch (`core.hooksPath=/dev/null`, `core.fsmonitor=false`, `GIT_CONFIG_GLOBAL=<parent-owned>`, `GIT_CONFIG_NOSYSTEM=1`, one-invocation `safe.directory`) plus a symlink-safe copy of uncommitted changes. The local auto-commit-in-worktree path (`daemon_tasks_worktrees.go`) is retired on split lanes | Without this the split is an **escalation path**: the operator running git in an agent-writable tree executes planted `.git/config`/hooks as uid 501 (R10 in the cloud doc; the local code has exactly this shape today) | human — **ruled 2026-10-16** (reuse H-6 D4) | design | high |
| D5 | **What the agent user may reach**: no read on `/Users/voightkampff` — enforced by `chmod 0700` on the operator home (preferred, inventory Daneel's cross-user reads first) or a targeted deny-ACL for `_ailang_agent` (fallback). `~/.config/ailang/secrets.env` (0600), the keychain (per-user; the agent has none), gcloud/gh config (under the home) are then denied by construction. **Network unchanged** — pf per-UID filtering is not cheap on macOS (no `owner` match); the rig gateway's lease already bounds ollama. Stated as accepted residual | The home-deny mechanism is the actual enforcement of decision point 3; the wrong choice either breaks Daneel (0700) or fails open silently (a dropped ACL) | human (0700 vs ACL, after the Daneel-dependency inventory) | design | low (one command) / med (inventory) |
| D6 | **Env passed to the child**: H-4's `BuildEnvironment` output is the *only* env source; the launcher applies it verbatim from the stdin spec (sudo's `env_reset` is not fought). The seam overrides `HOME=/Users/_ailang_agent`, `USER`/`LOGNAME=_ailang_agent`, and a `PATH` containing no operator-home entries | Keeps one env authority (H-4) instead of growing a second one inside sudo; a home-`PATH` entry would re-open `~/go/bin` and `~/dev` to the agent | agent (mechanics), per the H-4 seam | design | low |
| D7 | **Bounded termination across UIDs**: uid 501 cannot `kill` an `_ailang_agent` process group (EPERM). The launcher (running as the agent user) enforces the deadline itself, and a `signal` subcommand re-routes the coordinator's cancellation — it can only ever target the agent user's own processes | A rig where a looping agent can no longer be killed burns subscription quota forever; silently broken `KillGroup` is the most likely way this design regresses operations | agent (contract design), human (ratify the operational semantics) | design | med |
| D8 | **launchd/rig surface unchanged for trusted processes**: the coordinator daemon, mission drivers, rig-watchdog, cron-kicker and ollama stay as operator LaunchAgents. Only the agent CLI child crosses. Ollama access: localhost gateway + `AILANG_RIG_LEASE` (harness-injected, H-4 layer 2) — works identically for the agent user. Any shell glue is bash-3.2-safe (this rig ships 3.2.57, no assoc arrays) | Moving the coordinator itself to the agent user would invert the split (it must keep gh/gcloud); moving ollama would be host-ops churn for no boundary gain | agent | design | low |

### Design Freeze

- [x] D1: dedicated hidden `_ailang_agent` account; never `daneel`. — **Ruled 2026-10-16 (attended)**
- [x] D2: launch through one narrow sudoers rule **or an equivalent**, alternatives weighed. — **Ruled 2026-10-16**; this doc recommends option A below and asks Mark to ratify the specific choice
- [x] D4: reuse H-6 D4 (parent never runs git with the agent's tree as its repository). — **Ruled 2026-10-16 (commissioning)**
- [ ] D3: per-lane credential table (which lane gets which holder's credential). Inventory + Mark's ratification before any lane cutover
- [ ] D5: home-deny mechanism (0700 vs ACL), after the Daneel cross-user-read inventory
- [ ] D7: ratify the kill re-routing contract (launcher-enforced deadline + `signal` subcommand)
- [ ] Phase 0 probe result: **cross-UID `ps -wwE` must be denied** (expected, unmeasured). The rollout is gated on it exactly as the cloud split was gated on its spike

## Solution Design

### Overview

Three pieces, each independently shippable:

1. **A host account + one narrow privilege-granting rule** (`_ailang_agent` + sudoers + home
   deny). Host-ops in the Daneel estate plus a versioned install script in `tools/launchd/`.
2. **A spawn seam in this repo**: on darwin, the five executors' spawn path launches the CLI
   through a fixed launcher binary via `sudo -u _ailang_agent`, with the H-4-built env passed in
   a stdin spec, behind `AILANG_AGENT_UID_SPLIT=1` (default off, per-machine).
3. **Workspace + ingestion rework** for split lanes: staging bare repo both directions, hardened
   fetch in, agent commits, coordinator ingests and pushes as the operator.

Plus a probe suite whose positive control is the S6 sentinel: the same `ps -wwE` command that
leaks today must leak as uid 501 and be denied as `_ailang_agent`, or the rollout stops.

### The account (D1)

```bash
# Phase 0 host-ops (run once by the operator; scripted in tools/launchd/install_agent_user.sh)
sudo dscl . -create /Users/_ailang_agent UniqueID 275          # < 500 = hidden; exact UID deferred, must be stable
sudo dscl . -create /Users/_ailang_agent UserShell /usr/bin/false
sudo dscl . -create /Users/_ailang_agent RealName "AILANG agent executor"
sudo dscl . -create /Users/_ailang_agent Password '*'           # no password, no login
sudo dscl . -create /Users/_ailang_agent IsHidden 1
sudo dscl . -create /Users/_ailang_agent PrimaryGroupID <agent-gid>
sudo dscl . -append /Groups/_ailang_agentio GroupMembership voightkampff
sudo createhomedir -c -u _ailang_agent                          # /Users/_ailang_agent, never logged into
```

No GUI login (no loginwindow entry with `IsHidden` + `UserShell /usr/bin/false`), no keychain is
ever created for it (nothing runs `security` as this user; credentials are files and env — see
D3). The shared group `_ailang_agentio` exists for exactly one direction: the **operator reads
agent-owned files** (ingestion, session JSONL). It grants the agent user nothing.

### Launch mechanism (D2) — options weighed

The coordinator runs as uid 501 and cannot `setuid` by itself; on macOS there is no capability
model to hold. The ways to cross the UID line:

| Option | How | Verdict |
|---|---|---|
| **A. sudoers → fixed launcher (recommended)** | `/etc/sudoers.d/ailang-agent` (root:wheel 0440): `voightkampff ALL = (_ailang_agent) NOPASSWD: /opt/ailang/bin/ailang-agent-launcher`. The coordinator execs `/usr/bin/sudo -n -u _ailang_agent /opt/ailang/bin/ailang-agent-launcher run` with a JSON spec on **stdin** (argv, cwd, env pairs — the H-4 output —, deadline, workspace). The launcher validates argv, `chdir`s to the workspace, `execve`s the CLI with exactly the spec env, and enforces the deadline (D7) | One rule, one fixed root-owned 0755 binary (the agent user cannot modify it — and even if it could, it only ever runs **as the agent user**, so a plant gains no privilege: this is why the setuid direction matters). sudo's `env_reset` is sidestepped by design (env rides stdin, not the environment). Honest limits: (a) a sudoers command with no arg spec permits **any arguments** — the launcher's own subcommand validation (`run`/`signal`/`cleanup`) is the contract; (b) any uid-501 process can invoke the rule, i.e. can run things **as the low-privilege user** — bounded, and the threat model here is the reverse direction |
| B. sudoers → `/usr/bin/env` | `sudo -u _ailang_agent /usr/bin/env -i K=V … CLI args` — no new code | Zero-code cutover, but the rule reads "run anything as the agent user with any env", which is the opposite of narrow, and there is no place to enforce the deadline or validate the argv. Rejected as the standing mechanism; acceptable only as a Phase 0 manual probe vehicle |
| C. Root LaunchDaemon "spawnd" | A root daemon owns a 0600 Unix socket; spawns as `_ailang_agent` on request | Equivalent trust to the cloud "capability-holding parent", but it is a new always-on root service that parses request data — strictly more code and more surface than one sudoers line for the same boundary. Kept as the fallback if sudo is ruled out (e.g. an org-wide sudoers policy) |
| D. setuid-root launcher | A setuid helper that drops to `_ailang_agent` | The cloud doc rejected this shape (its option C) for the same reason here: a setuid binary the *child* can also invoke has to be argument-validated forever, and it adds a setuid file to the rig. **Rejected** |
| E. launchd per-user domain for `_ailang_agent` | Pre-defined LaunchAgents in the agent user's domain | `launchctl asuser`/submit for another user's domain requires root; a hidden no-login user has no live gui domain. A per-task `launchctl` dance is more moving parts than sudo for zero added boundary. **Rejected** |
| F. macOS Seatbelt (`sandbox-exec`) instead of a UID | Profile the CLI process | `sandbox-exec` is deprecated and not a security boundary Apple commits to; a profile also cannot express "read nothing of the operator's but keep these five CLI binaries" as legibly as ownership. **Rejected** (the ruling also names a user, not a profile) |

**Recommended: A.** The launcher is a small Go binary from this repo
(`cmd/ailang-agent-launcher`), so its validation, deadline and signal handling are CI-tested
rather than shell glue. It is installed root:wheel 0755 under `/opt/ailang/bin/` together with
symlink-free copies of the agent-reachable CLIs (see the bin-root inventory below). If any shell
glue is ever needed around it, it is bash-3.2-safe (D8).

### Spawn seam (in this repo)

All local agent spawns — coordinator dispatch (`internal/coordinator/provider_executor.go`),
mission dispatch (`internal/mission/dispatch`), and the eval harness — go through the five
executor packages' `exec.CommandContext` + `proctree.Configure` path (verified: `pi.go:149-160`
and the shared `executor.BuildEnvironment`). So one seam covers every local lane:

- **`internal/executor/agentlaunch_darwin.go`** (new): `ConfigureAgentUserLaunch(cmd, spec)`
  rewrites the command to the sudo/launcher form when `AILANG_AGENT_UID_SPLIT=1`, merging into
  the `SysProcAttr` that `proctree.SetGroup` already created (never replacing it). Non-darwin
  (and darwin with the flag off) returns a typed `ErrAgentUIDSplitDisabled` — **never a silent
  same-UID fallback**; darwin with the flag on but the launcher missing/unrunnable is a named
  pre-spawn error and the task fails loudly (A11).
- **Env (D6)**: the spec's env is exactly `BuildEnvironment`'s output (H-4's allowlist, grants,
  harness-injected `AILANG_RIG_LEASE` etc.), with the seam overriding `HOME`, `USER`, `LOGNAME`
  and `PATH` (agent bin root first, then `/opt/homebrew/bin:/usr/bin:/bin` — no operator-home
  entries). One env authority, no second builder.
- **Timeout/kill (D7)**: `proctree.KillGroup` from uid 501 cannot signal an agent-UID group
  (`kill(2)` EPERM). In split mode the seam replaces `cmd.Cancel` with a call to
  `ailang-agent-launcher signal <pgid> TERM|KILL` (same sudoers rule). The launcher, running as
  the agent user, can only signal its own UID's processes — it is structurally unable to aim at
  the coordinator — and the launcher itself enforces the spec's wall-clock deadline on its own
  group even if the coordinator dies. `ResolveGroupError` must not misreport the EPERM shape;
  pinned by test.

### Workspace, staging and git ingestion (D4)

macOS has no `chown`-as-non-root: the operator **cannot hand** an operator-built checkout to the
agent user, and the agent user **cannot write** into operator-owned worktrees. So the data flow
inverts to two one-way fetches through a staging area, which is also exactly the H-6 D4 shape
(parent never runs git in the agent's tree):

1. **Prepare (operator, pre-spawn)**: the coordinator creates a parent-owned bare clone for the
   task at `<STAGING>/<taskID>.git` (staging root default
   `/Users/Shared/ailang/staging`, owner `voightkampff:_ailang_agentio`, mode 0750 — group
   read/traverse only, so the agent can fetch from it but not write to it). Today's worktrees
   come from operator checkouts under `~/dev`; the clone is made from the same source, outside
   the operator home.
2. **Run (agent)**: the launcher (as `_ailang_agent`, umask 027) clones/fetches
   `<STAGING>/<taskID>.git` into `/Users/_ailang_agent/workspaces/<taskID>` (agent-owned,
   group-readable), then execs the CLI with that cwd. The agent user's `~/.gitconfig` carries the
   agent identity from the registry's `GitIdentity` (installed per run via the spec's env,
   `GIT_AUTHOR_NAME`/`GIT_COMMITTER_NAME`/`EMAIL` — no operator gitconfig is ever read).
3. **Ingest (operator, post-run)**: the coordinator fetches from the agent workspace into its
   own parent-owned clone under `~/.ailang/state/ingest/<taskID>` with the hardened `-c` set
   (hooks off, fsmonitor off, parent-owned `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_NOSYSTEM=1`, and a
   one-invocation `safe.directory=<workspace>`); copies uncommitted working-tree changes with the
   symlink-safe copier (stores links as links — blocks the "plant a symlink to
   `~/.gitconfig` and let the parent commit it" exfil); commits and pushes **as the operator**
   with the operator's gh credentials, which the agent user never sees. The local auto-commit in
   the worktree (`daemon_tasks_worktrees.go:140-170`) — today's R10-shaped code — is retired on
   split lanes.
4. **Cleanup**: staging dir removed by the coordinator; workspace removed by the launcher's
   `cleanup` subcommand (or retained for debugging, bounded by a sweeper).

If the agent chmods its tree private, ingestion fails **loudly** (named error, task banked
failed) — the agent can only break its own task. Cross-task isolation is **unchanged from
today**: all agent tasks share `_ailang_agent`, exactly as they share uid 501 today.

### Credentials (D3) — per lane, one holder per credential

The rule is the cloud model (env-vars.md: a credential is "held by ONE consumer … never both",
because codex/claude **rotate** their refresh tokens and two holders revoke each other — the
measured codex lane-lease failures, `credential_lane.go`). With the split there are three
holders: the operator's interactive logins, the cloud executors, and the rig's `_ailang_agent`.
Never copy a blob between holders.

| Executor | Credential the agent child gets | Delivery |
|---|---|---|
| claude | the **rig agent user's own** `claude setup-token` (`sk-ant-oat…`) — a third token, distinct from the operator's login and from the cloud's token; setup-tokens do not refresh and last ~a year | H-4 grant (`CLAUDE_CODE_OAUTH_TOKEN` env) or a file under `/Users/_ailang_agent/.claude/`; never the operator's `.credentials.json` or keychain |
| codex | **its own** `auth.json` (a dedicated subscription login for the rig agent user), exclusively leased per run by the existing credential lane — or `apikey` mode if a second subscription is not worth it | file under `/Users/_ailang_agent/.codex/`, 0600, installed once by the operator; the lane lease already serializes holders |
| pi / opencode | provider keys (`OPENROUTER_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, `OLLAMA_API_KEY` per model routing) — unchanged, these are keys not rotating refresh tokens | H-4 env grants, exactly as today |
| motoko | provider keys in env; per-task `AILANG_CACHE_DIR` under the agent workspace | H-4 env grants |

The operator's keychain is never consulted by any agent process: the agent user has no login
keychain, and nothing in this design runs `security` as that user.

### What the agent user may reach (D5)

| Path | Owner : mode | Agent access | Why |
|---|---|---|---|
| `/Users/voightkampff` | voightkampff : **0700** (or + deny-ACL for `_ailang_agent`) | **none** | M3/M4 in one line. Default macOS home modes vary by OS version — the rig's current mode is a Phase 0 measurement; the design sets it explicitly |
| `~/.config/ailang/secrets.env` | already 0600 operator | none (subsumed) | belt-and-braces even if the home deny is ever relaxed |
| Login keychain | per-user | none (no keychain exists for the agent user) | — |
| `/opt/ailang/bin/**` (launcher, CLI copies), `/opt/homebrew/**` | root : 0755 | r-x | the CLIs the agent may exec. **Inventory item:** every CLI path that currently resolves under `/Users/voightkampff` (`~/go/bin/ailang`, pi/motoko checkouts under `~/dev/mk-*` per MOTOKO.md, node CLIs) must be relocated or copied to the agent bin root — a symlink into the operator home does not help |
| `/Users/Shared/ailang/staging/<taskID>.git` | voightkampff : 0750 (`_ailang_agentio`) | r-x | group read for the inbound fetch |
| `/Users/_ailang_agent/workspaces/<taskID>` | _ailang_agent : 0750 (umask 027) | rw | the agent's tree — **untrusted to the coordinator from spawn on** (D4) |
| `/Users/_ailang_agent/.{claude,codex}` | _ailang_agent : 0600/0700 | rw | the agent's own credentials (R5-equivalent: not protectable, by construction) |
| ollama `localhost:11434` (gateway) | — | reachable, lease-gated | M-RIG-GPU-ADMISSION-GATEWAY; `AILANG_RIG_LEASE` is harness-injected. Network otherwise unchanged (D5 residual) |

**Stated residual:** network egress from the agent user is not restricted (pf per-UID filtering is
not cheap on macOS and the ruling scopes network to "unchanged unless cheap to restrict"). A
compromised agent can still talk to the internet with whatever credential its lane holds. The
lease-gated GPU path is the one already-bound resource.

### launchd / rig interactions (D8)

- **Coordinator daemon**: stays a `gui/501` LaunchAgent running as the operator — it is the
  trusted parent that ingests and pushes. Its plist (`dev.ailang.coordinator.plist`) is
  unchanged; `COORDINATOR_API_KEY` stays in its env, which is fine once the agent UID cannot read
  uid-501 environments (the Phase 0 probe gates exactly this).
- **Mission drivers, rig-watchdog, cron-kicker, nightly-eval, ollama serve**: unchanged operator
  LaunchAgents. They invoke agent CLIs through `ailang` (mission iterate → dispatch → executors),
  so the seam covers them without touching the bash drivers. Any *new* glue is bash 3.2-safe.
- **Session/JSONL reads**: motoko's `.motoko/logfile/` and claude's session JSONL land inside the
  agent workspace (group-readable via umask 027); the observatory importers
  (`ailang chains import-motoko`) read them as the operator through group read. If a CLI's umask
  defeats that, the launcher's `collect` fallback copies the logs out as the agent user.
- **State dirs**: the agent child's `AILANG_CACHE_DIR` and scratch move under the agent
  workspace; coordinator state stays under `~/.ailang` (operator).

## Implementation Plan

**Phase 0: probe + host account (~1 day, gates everything)**
- [ ] **S7 probe (the positive control)**: start a decoy uid-501 process with
      `H6PROBE2=sentinel`; from the operator, `ps -wwE -p <pid>` shows it (leak reproduced —
      today's state). Then, via `sudo -u _ailang_agent`, run the same `ps -wwE`: **no
      environment** (expected, unmeasured). Record the transcript here. If cross-UID env reads
      succeed, **stop and report** — the design's core premise fails and needs an ACL/other
      boundary instead
- [ ] Create `_ailang_agent` + `_ailang_agentio` (script `tools/launchd/install_agent_user.sh`,
      versioned like the plists); `/opt/ailang/bin` root-owned
- [ ] Measure and then set the home-deny (0700 vs ACL) after the Daneel cross-user-read
      inventory; probe `/Users/voightkampff` access both ways
- [ ] File reads probe as `_ailang_agent`: `.ssh`, `.config/gcloud`, `.config/gh`, `.claude`,
      `.codex`, `secrets.env`, `~/dev` → all denied; control as uid 501 → allowed
- [ ] CLI bin-root inventory: every agent-exec'd binary/checkout that resolves under
      `/Users/voightkampff` today, listed with its relocation plan

**Phase 1: launcher + spawn seam (~3 days)**
- [ ] `cmd/ailang-agent-launcher`: run/signal/cleanup subcommands, stdin spec (argv, cwd, env,
      deadline), deadline enforcement, umask 027, exec validation; unit + integration tests
- [ ] `internal/executor/agentlaunch_darwin.go` seam (+ `_other.go` typed-error stub), wired
      next to `proctree.Configure` in the five executors; `AILANG_AGENT_UID_SPLIT` flag (default
      off); named errors, no fallback
- [ ] Kill re-routing (D7) + `ResolveGroupError` handling; cross-UID kill test (deadline fires;
      cancellation fires; a stuck sudo fails loudly)
- [ ] Env seam test: child env == `BuildEnvironment` output + HOME/USER/PATH overrides, byte for
      byte; no operator-home PATH entry

**Phase 2: staging + ingestion (~3 days)**
- [ ] Staging prepare/ingest in the local coordinator; hardened fetch + symlink-safe copy
      (port the cloud D4 discipline; precedents: `internal/effects/process_confined.go`,
      `internal/gitexec`)
- [ ] Retire auto-commit-in-worktree on split lanes (`daemon_tasks_worktrees.go`); hostile-repo
      fixtures T1–T5 (hooks, fsmonitor, filter, include.path, symlink) never execute a sentinel
      as the operator
- [ ] Workspace lifecycle + sweeper; loud failure when the agent tree is unreadable

**Phase 3: credentials + rollout (~3–4 days)**
- [ ] Per-lane credential install per the D3 table; claude setup-token minted for the agent
      user (never the operator's or cloud's); codex holder decision (own login vs apikey)
- [ ] Lane-by-lane rollout behind the flag: coordinator dispatch lanes first (claude last, as
      in the cloud order), then mission loops (one mission, one iteration, watched), then
      nightly eval; one green probe task per lane before the next
- [ ] Ops runbook (`docs/docs/guides/` or `tools/launchd/`): account, sudoers, bin root,
      rollout/rollback (flag off), rotation procedure

### Files to Modify/Create

**New files:**
- `cmd/ailang-agent-launcher/main.go` — the launcher (run/signal/cleanup, spec, deadline) (~250 LOC)
- `internal/executor/agentlaunch_darwin.go` + `agentlaunch_other.go` — the seam and typed
  errors (~150 LOC)
- `internal/coordinator/daemon_tasks_ingest.go` — staging prepare + hardened fetch + symlink-safe
  copy (~220 LOC)
- `tools/launchd/install_agent_user.sh` — account, group, sudoers, bin root, home deny
  (bash 3.2-safe, ~80 LOC)
- `tools/launchd/agent_uid_probe.sh` — the S6/S7 probe suite with positive controls (~100 LOC)
- `internal/executor/agentlaunch_test.go`, `internal/coordinator/daemon_tasks_ingest_test.go`
  (~400 LOC)

**Modified files:**
- `internal/executor/{pi,claude,codex,opencode,motoko}/*.go` — one `ConfigureAgentUserLaunch`
  call next to `proctree.Configure` (~30 LOC total)
- `internal/coordinator/daemon_tasks_worktrees.go` — retire auto-commit on split lanes (~40 LOC)
- `docs/docs/reference/env-vars.md` — document `AILANG_AGENT_UID_SPLIT` (~10 LOC)
- `docs/docs/guides/` — rig ops runbook (~1 page)

## Examples

### Example 1: the S6 bypass, before and after

**Before (today):**
```text
agent (uid 501)$ ps -wwE -p $(pgrep -f 'ailang coordinator start') | tr ' ' '\n' | grep -E 'KEY|TOKEN'
COORDINATOR_API_KEY=…          # plaintext in the plist env
OPENROUTER_API_KEY=…           # inherited through bash -lc's profile
H6PROBE2=sentinel              # any sentinel a 501 process ever held
```

**After (split on):**
```text
agent (_ailang_agent)$ ps -wwE -p <coordinator pid>
ps: (uid 501)…                # environment withheld for another user's process
agent (_ailang_agent)$ cat /Users/voightkampff/.ssh/id_ed25519
cat: …: Permission denied
# control, run as the operator from a second terminal:
operator$ ps -wwE -p <pid> | grep H6PROBE2
H6PROBE2=sentinel              # the probe still leaks for a same-UID reader — the denial is real
```

### Example 2: why D4 is mandatory (local R10)

```text
# agent, before exiting:
git config core.fsmonitor '/bin/sh -c "cat /Users/voightkampff/.ssh/* > leak; false"'
# coordinator today (daemon_tasks_worktrees.go): runs in the worktree as uid 501
git -C <worktree> add -A       # executes the fsmonitor AS THE OPERATOR → keys copied into the tree
# with D4: the coordinator never runs git with the agent tree as its repository;
# it fetches with core.fsmonitor=false from its own clone
```

## Success Criteria

- [ ] AC1 (Phase 0): S7 probe transcript in this doc: same-UID `ps -wwE` leaks the sentinel;
      agent-UID `ps -wwE` shows no environment; file-read probe matrix all-denied with uid-501
      controls all-allowed
- [ ] AC2: a split-mode dispatch runs one real task per lane in dev-on-the-rig: workspace
      writes, commits ingested via hardened fetch, PR pushed by the coordinator, session JSONL
      imported, ollama lane admitted by lease
- [ ] AC3: hostile-repo fixtures T1–T5 (planted hooks/fsmonitor/filter/include.path/symlink)
      never execute a sentinel as the operator; the ingested PR diff contains no operator-owned
      file content
- [ ] AC4: deadline and cancellation kill the agent's whole group (launcher-enforced and
      signal-routed); a deliberately hung CLI dies within the deadline in a test
- [ ] AC5: wrong-user or broken-launcher spawn is a named pre-dispatch error (zero silent
      same-UID runs — asserted by a fault-injection test)
- [ ] AC6: per-lane credential inventory recorded (which holder each lane uses); no credential
      blob exists in two holders' storage at once (probe lists)
- [ ] AC7: `make test`, `make lint`, `make check-boundaries` green; env-vars.md + runbook
      updated; `tools/launchd/` scripts `bash -n` clean and bash-3.2-safe

## Testing Strategy

**The probe suite (`tools/launchd/agent_uid_probe.sh`, the core test):** a decoy operator process
holds `H6PROBE2=sentinel`; the suite runs, as both the operator and `_ailang_agent`:
1. `ps -wwE -p <pid>` (environment read — **must leak as 501, be empty as agent**)
2. file reads: `.ssh/*`, `.config/gcloud`, `.config/gh`, `.claude/.credentials.json`,
   `.codex/auth.json`, `.config/ailang/secrets.env`, `~/dev`, `~/.ailang`
3. keychain: `security find-generic-password` as the agent user (no keychain / denied)
4. write attempts to coordinator state, staging (0750 group-read: agent write denied)
5. positive path: workspace write, own-credential read, ollama gateway with lease
It writes one `denied`/`ALLOWED` line per attempt; the matrix is asserted in CI where uid
switching is impossible (Linux CI asserts the seam's env/error contract) and on the rig
(`ailang` probe task, like M-SEC2's probe rollout). **A probe that cannot see the leak fails
rather than passes** — the uid-501 control run is the vacuity guard (the same discipline as the
cloud P0.4 probe test).

**Unit/integration (CI, no rig needed):** launcher spec validation and arg contract; kill
re-routing semantics; seam env byte-equality; hostile-repo fixture suite (T1–T5) against the
ingestion code as a plain-UID round trip (the fixtures test *git* discipline, not UIDs).

**Manual (rig):** one probe dispatch per lane per the rollout order; then one real task per lane
before the next.

## Rollout

Per-lane behind `AILANG_AGENT_UID_SPLIT` (a host-side env var on the coordinator, never
task-settable), so a lane rolls back by config change: coordinator dispatch lanes first
(pi → opencode → motoko → codex → claude last, the subscription lanes last as in the cloud
order), then one mission loop for one watched iteration, then nightly eval. The flag defaults
**off**; a lane is flipped only after its green probe task. Mission drivers keep working because
their agent invocations route through the executor seam; their *post-run git inspection* of the
workspace runs as the operator through group read (and any driver step that commits in the agent
tree must migrate to the ingestion path — part of the mission cutover inventory).

## Deferred Decisions

- Exact UID/GID numbers for `_ailang_agent` / `_ailang_agentio` — agent may choose, must be
  stable and recorded in the runbook
- Launcher implementation language beyond "Go in this repo" internals (spec schema details,
  logging shape) — agent may choose
- Staging root location (`/Users/Shared/ailang/staging` vs `/var/ailang/staging`) — agent may
  choose, documented
- Whether the claude lane uses env (`CLAUDE_CODE_OAUTH_TOKEN`) or a file under the agent home —
  agent may choose; never the operator's blob either way
- Codex: own subscription login vs apikey mode — human at rollout, priced by usage
- Whether the `collect` fallback (launcher copies logs as the agent user) is needed on any lane,
  or umask-027 group read suffices — decided by the first green probe tasks
- Workspace retention/sweep policy — agent may choose

## Non-Goals

- **The cloud H-6 split.** Separate track (its Non-Goal names this doc, and its D6 ruling
  commissions it). No Cloud Run change here.
- **Moving the coordinator, mission drivers or watchdog off the operator account.** They are the
  trusted parent (D8). The plist's `bash -lc` profile-env question is noted as hygiene elsewhere,
  not this design.
- **Restricting agent network egress on macOS.** Not cheap (no pf owner-match); recorded as the
  accepted residual. The rig gateway lease already bounds the GPU path.
- **Cross-task isolation between agent runs.** Same as today (one uid 501 ↔ one `_ailang_agent`).
  A per-task account would be a future extension.
- **Protecting the agent user's own inference credential.** Impossible by construction (the CLI
  must read it); bounded by one-holder-per-credential (D3) and H-4's single-grant rule.
- **TCC / Full Disk Access games.** Not a boundary we design against or rely on.

## Timeline

**Week 1** (~4 days): Phase 0 (probe, account, home deny, inventory) + launcher start.
**Week 2** (~4 days): seam + kill re-routing + staging/ingestion core + fixture suite.
**Week 3** (~3–4 days): credentials per lane, rollout (dispatch lanes), runbook, first mission
iteration.
**Total: ~10–12 days across ~3 weeks** (planning estimate at 2× the raw guess).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cross-UID `ps -wwE` is NOT denied on macOS 26.6 (premise fails) | High | It is Phase 0's first gate, before any code beyond the probe; fallback = deny-ACLs on the coordinator's env are not possible for env — the design would need re-scoping to file protections only, escalated to Mark |
| The sudoers rule lets any uid-501 process run as `_ailang_agent` | Med | Bounded to the low-privilege side; the launcher validates argv; the rule is one fixed path; recorded honestly here rather than papered over |
| A CLI or checkout the agent needs still resolves under the operator home → lane breaks at cutover | Med | Phase 0 bin-root inventory before any cutover; per-lane probe task gates each flip; flag-off rollback is config-only |
| 0700 home breaks a Daneel (uid 502) cross-user read nobody remembered | Med | Inventory first; ACL fallback targets only `_ailang_agent` |
| Kill re-routing regresses bounded termination (a looping agent burns quota) | High | Launcher-enforced deadline independent of the coordinator (D7); AC4's hung-CLI test; rollout watches one mission iteration before the next lane |
| Mission drivers' workspace/git habits silently keep running git in the agent tree as the operator | High | The local auto-commit path is retired on split lanes; the mission cutover inventory lists every driver git step; T1–T5 fixtures gate the ingestion path |
| Credential sprawl (a third claude token, a second codex login) | Med | One-holder rule recorded in env-vars.md lineage; rotation procedure in the runbook; AC6 probes for double-holding |

## Verification Log

| ID | Claim checked | Instrument / evidence | Result |
|---|---|---|---|
| V1 | Local coordinator runs as the operator under a login shell with `COORDINATOR_API_KEY` in the plist env | Read `tools/launchd/dev.ailang.coordinator.plist.template` (`/bin/bash -lc "exec … coordinator start"`, `EnvironmentVariables` block); rig plist read in the cloud doc's V16 | Confirmed |
| V2 | Same-UID env reads via `ps -wwE` work on macOS 26.6 and bypass H-4 | Spike S6 (m-executor-uid-split.md Progress 2026-10-16): homebrew python3 sentinel visible; platform binaries hidden | Confirmed (S6) |
| V3 | **Cross-UID** `ps -wwE` denial (the design's premise) | **Unmeasured** — Phase 0 S7 probe, gated, with the same-UID control | **Pending by design** |
| V4 | All local agent spawns route through the five executors' `exec.CommandContext` + `BuildEnvironment` path | Read `provider_executor.go:84-175` (dispatch → `p.exec.Execute`), `internal/mission/dispatch/run.go` (Runner → executor factory), `pi/pi.go:149-160`; EXECUTOR_SHAPE.md ("auto-discovered by both the coordinator and the eval harness") | Confirmed — one seam covers coordinator, missions and eval |
| V5 | H-4 landed: `BuildEnvironment` is default-deny and the env authority | Read `internal/executor/envpolicy.go` (allowlist, credential-shaped withholding, `AILANG_EXECUTOR_ENV_INHERIT`), `environment.go:BuildEnvironment` layers; m-executor-env-hardening.md status "Implemented" | Confirmed |
| V6 | The local coordinator runs `git add`/`git commit` **in the agent worktree** after the run (the R10 shape) | Read `daemon_tasks_worktrees.go:140-170` (`gitexec.Command("add","-A")` with `Dir=worktreePath`, then commit) | Confirmed — retired on split lanes by D4 |
| V7 | No privilege-drop / agent-user launch mechanism exists in the repo today (negative) | `grep -rn "sudo\|setuid\|_ailang" internal/ cmd/ --include='*.go'` (non-test): only an unrelated `apt install gh` hint in messaging | Confirmed — none exists |
| V8 | Worktrees today live under operator state; missions source operator-home env files | Read `daemon_tasks_init.go:236-242` (worktree base `StateDir/worktrees/<agent>`), `tools/launchd/mission-env/README.md` (drivers source `~/.config/ailang/mission-<name>.env`), v1-mission.md:442 (`CLAUDE_CODE_OAUTH_TOKEN` in `secrets.env`) | Confirmed |
| V9 | uid 501 cannot signal an `_ailang_agent` process group | `kill(2)` permission model (real/effective UID match or root); proctree's group kill (`proctree.go`) assumes same authority | Design reasoning + AC4 test; EPERM shape pinned by test |
| V10 | Non-root cannot `chown` to another user on macOS (workspace flow must invert) | POSIX `chown` semantics; no capability model on darwin | Confirmed (design constraint, drives the staging flow) |
| V11 | One-holder-per-credential is the established model for rotating OAuth | env-vars.md `CLAUDE_CODE_OAUTH_TOKEN` ("held by ONE consumer … never both"); `credential_lane.go` header (codex refresh-token rotation, measured two-lost-per-pair failures) | Confirmed |
| V12 | Ollama access stays lease-gated regardless of UID; bash on the rig is 3.2 | m-rig-gpu-admission-gateway.md (gateway :11434, lease token; `AILANG_RIG_LEASE` harness-injected via envpolicy layer 2); `cron-kicker.sh:43`, `mission-base.sh:3` ("bash 3.2.57") | Confirmed |
| V13 | No sudoers/setuid/plist convention for service accounts exists in the repo to extend (negative) | `ls tools/launchd/`, grep for sudoers | Confirmed — new ground, hence the options table |
| V14 | Duplicate/coverage gate: no planned or implemented doc owns the local agent-user boundary | `create_planned_doc.sh` search (simhash + neural): no matches above threshold for "mac agent uid"; read the closest docs: m-executor-uid-split (cloud, D6 explicitly defers this doc), m-rig-gpu-admission-gateway (GPU admission), m-motoko-group-kill (probe kill mechanics), m-git-guardrails-hookless (git-mode guardrails) — all state disjoint scope | Confirmed — distinct topic, commissioned by ruling |
| V15 | No AILANG language claims | N/A | This design changes no language surface; `ailang check` gate not applicable |

**Quorum trigger:** #1 fires (design-freeze items D3/D5/D7 open) and #3 partially (load-bearing
premise V3 about an external system, macOS, is deliberately gated behind Phase 0 rather than
asserted). Run `ailang design-quorum` before planning, per the skill's trigger rules.

## Related Documents

- [M-EXECUTOR-UID-SPLIT](../v0_49_1/m-executor-uid-split.md) (planned, cloud track): D6 ruling
  commissions this doc; D4's git-ingestion design is reused verbatim; S6 is the shared evidence
- [M-EXECUTOR-ENV-HARDENING](../v0_49_1/m-executor-env-hardening.md) (implemented per its
  status header, filed under planned/): the env allowlist this design inherits as its only env
  authority
- [M-RIG-GPU-ADMISSION-GATEWAY](../m-rig-gpu-admission-gateway.md) (Phase 2 cut over): the
  ollama lease the agent user keeps using
- [M-GIT-GUARDRAILS-HOOKLESS-EXECUTORS](../m-git-guardrails-hookless-executors.md) (proposed):
  the git-mode guardrail layer above the launcher; composes
- ailang-multivac `internal-docs/SECURITY-AUDIT-2026-10-01-ailang-executor-escape-hardening.md`:
  the audit H-4/H-6 family this belongs to
- [env-vars.md](../../docs/docs/reference/env-vars.md) `CLAUDE_CODE_OAUTH_TOKEN`: the
  one-holder-per-credential rule D3 extends to the third holder

## References

- [Design Axioms](/docs/references/axioms)
- macOS `dscl(1)`, `sudoers(5)`, `kill(2)`, `chmod(1)` (ACLs), Apple's underscore service-account
  convention (`_www`, `_softwareupdate`, …)
- H-6 cloud doc's D4 hardening set (`core.hooksPath`, `core.fsmonitor`, `safe.directory`,
  protected config) — `git-config(1)`
- MOTOKO.md — which motoko checkouts the rig's CLIs actually run (bin-root inventory input)

## Future Work

- Per-task accounts (`_ailang_agent_<task>`) if cross-task isolation is ever wanted
- Revisit network restriction if pf grows a workable per-UID match or a local forward proxy
  becomes cheap (mirror of the cloud D5 direction)
- The coordinator plist's `bash -lc` full-profile environment (it exports more than the daemon
  needs even before any agent can read it) — hygiene, separate lane

---

**Document created**: 2026-10-16
**Last updated**: 2026-10-16
