# M-EXECUTOR-UID-SPLIT: Running the Agent CLI Under Its Own UID (Audit H-6)

**Status**: Planned — **awaiting approval** (design only; no implementation). The recommendation is gated on a spike, see [Verdict](#verdict-value-vs-effort).
**Target**: Phase 0 in the next security release after v0.49.0. Phases 2–4 have no target: they wait on the spike result and a human go/no-go.
**Priority**: P1 for Phase 0, which includes a live credential leak found while writing this doc ([F-H6-1](#f-h6-1-claude-oauth-credentials-are-persisted-to-the-shared-artifacts-bucket)). P2 / conditional for the full UID split.
**Estimated**: Phase 0 ~1.5 days · spike ~1 day · full split ~10–14 engineering days if the spike says go.
**Dependencies**: composes with [M-EXECUTOR-ENV-HARDENING](https://github.com/sunholo-data/ailang/pull/1417) (H-4, PR #1417, planned). Uses the M-SEC2 lane split (multivac `internal-docs/M-SEC2-cloud-executor-hardening.md`, Phase 1 landed) and the egress lock (B1, dev only).
**Source**: ailang-multivac `internal-docs/SECURITY-AUDIT-2026-10-01-ailang-executor-escape-hardening.md` §4 Phase C, H-6. Mark approved turning this item into a design doc on 2026-10-01.
**Created / updated**: 2026-10-01

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to program semantics or scheduling |
| A2: Replayability | 0 | Nothing new is banked. The child's UID is recorded on `Result` for audit only |
| A3: Effect Legibility | +1 | The launcher's authority (push credential, deploy key, parent env) stops being implicitly shared with the model-facing process |
| A4: Explicit Authority | +1 | The child's file and process reach becomes a stated ownership table instead of "everything `ailang` can touch" |
| A5: Bounded Verification | +1 | Each claim comes with an adversarial probe that can be run in a container (read `/proc/$PPID/environ`, rewrite hooks, read a parent-owned file) |
| A6: Safe Concurrency | 0 | Process-group kill semantics are kept, see Conflict Surface |
| A7: Machines First | 0 | Operator-facing infrastructure. Not a language change |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No cost-path change |
| A10: Composability | +1 | Splits the work cleanly with H-4 (what the child *sees* in its env) and M-SEC2 (what its SA *can do*) |
| A11: Structured Failure | +1 | A launch as the wrong UID, or as root, is a named pre-spawn error, not a silent fallback to same-UID |
| A12: System Boundary | +1 | Makes the parent→child line a kernel-enforced boundary instead of a convention |

**Net Score: +6** → **Decision: Move forward with Phase 0 and the spike. The full split is conditional, see Verdict.**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects. Phase 0 *removes* one (credentials silently written to a shared bucket)
- [x] A4 (Authority): no ambient access granted. The design narrows ambient access
- [x] A7 (Machines First): no human-convenience trade against machine analysis

## Problem Statement

Every cloud executor job runs one process tree as one UID. `execute-job` (the **parent**: clones,
installs credentials, runs git after the agent, opens the PR, publishes the completion) and the AI
CLI it spawns (the **child**: `claude`/`codex`/`pi`/`opencode`/`motoko`, run with
`--dangerously-skip-permissions` or equivalent, with bash) both run as the image user `ailang`.
The child is model-facing, and prompt injection reaches it through issue/PR text, cloned repos and
benchmark files. Same UID means that anything the parent can read, write or signal, the child can
too.

H-6 asks whether a distinct UID for the child would close that. This doc answers with the code
and the deployed images. The short version: **a UID split protects less than it first appears to,
because the child can already reach most of what the parent holds through the metadata server.
Several of the things H-6 names can also be closed more cheaply without a split. The split is
worth its cost only if it also lets us deny the child the metadata server**, and the spike in
Phase 1 has to establish that first.

### What a compromised child can reach today (same UID)

Evidence: the `dev` branch at `85f863b87`, the images in `docker/`, and live reads of the dev,
test and prod deployments made 2026-10-01 (read-only). See the Verification Log.

| # | Asset held by the parent | How the child reaches it today | Does a UID split stop it? | Cheaper control |
|---|---|---|---|---|
| R1 | **Parent's initial environment**: every `secret_key_ref` the job injects (internal lane: up to `claude_code_oauth_token`, `github_token`, `google_api_key`, `openai_api_key`, `openrouter_api_key`, `ollama_api_key`, `registry_api_key`) plus per-execution overrides (apikey-mode user key) | `cat /proc/$PPID/environ` (or `/proc/1/environ`, because `ENTRYPOINT ["/usr/local/bin/ailang"]` makes the parent PID 1). Reading `environ` needs `PTRACE_MODE_READ`. Yama `ptrace_scope` restricts only *attach*, so a same-UID reader passes. **This bypasses H-4 entirely**: H-4 cleans the child's env, but the parent's env is one `cat` away. Go's `os.Unsetenv` does not scrub the initial `environ` block that `/proc` shows | Yes | **Yes: `prctl(PR_SET_DUMPABLE, 0)` in the parent** makes `/proc/<parent>/{environ,mem}` root-owned and refuses ptrace from same-UID processes. This is the `ssh-agent` technique. ~10 LOC, no UID split |
| R2 | **Parent memory**: decrypted KMS API key (apikey mode), the codex credential baseline, Secret Manager reads made at runtime (SSH deploy key), the future per-task push token (H-4 D2) | `/proc/$PPID/mem` or `ptrace(PTRACE_ATTACH)`. Both need `PTRACE_MODE_ATTACH`, which Yama scope ≥1 restricts to descendants, and the child is not the parent's ancestor. Live Yama value on gen2 is **unmeasured** (spike item S3) | Yes | Yes: the same `PR_SET_DUMPABLE=0` |
| R3 | **Fleet GitHub token in `~/.gitconfig`**: Step −1 writes `credential.helper '!f() { …password=<GITHUB_TOKEN>…}'` with `git config --global` (`coordinator_cloud.go:336-345`) | `cat ~/.gitconfig`, or just `git push`, which uses it | Only if the parent's gitconfig is unreadable to the child **and** the child is not supposed to push. Some lanes push by design (`AILANG_PUSH_BRANCH`, guardrails mode), so the child legitimately needs *a* push credential | H-4 D2 (scoped per-task credential file instead of the fleet token) already owns this. The UID split only adds that the parent's *own* credential can be kept from the child |
| R4 | **SSH deploy key**: `~/.ssh/<alias>` 0600 plus `~/.ssh/config` (`coordinator_cloud_sshkey.go:98-126`) | `cat ~/.ssh/*` (same UID, so 0600 means nothing) | Yes, if `~/.ssh` belongs to the parent UID | No same-UID fix. But see R7: the lane SA can read the secret anyway |
| R5 | **The CLI's own credential**: `~/.claude/.credentials.json`, `~/.codex/auth.json`, pi/opencode provider keys in env | The CLI reads it, by design | **No, and it cannot.** The CLI must read its own auth, and claude and codex **rewrite** it on refresh, so the file must belong to the child UID | None. This is an accepted residual. M-SEC2 (one inference credential per lane) and H-4 (only that one in env) bound it |
| R6 | **Hooks, plugins and settings**: pre-baked `/plugins/ailang_bootstrap` (**`chown -R ailang` at build**, `Dockerfile.agent-base`), runtime-cloned `/plugins/<taskID>/…`, `~/.ailang/hooks/claude_telemetry.sh` and `~/.ailang/claude/settings.json` (rewritten at runtime by `GetClaudeSettingsPath`), the workspace's `.claude/settings.json`, and `CLAUDE_CONFIG_DIR` on `/artifacts` | Write over them (same owner) | Partly. Root- or parent-owned plugin and hook dirs become tamper-proof. The workspace `.claude/` and `CLAUDE_CONFIG_DIR` stay child-writable by necessity | Partly: **root-own `/plugins` at build** (drop it from the `chown -R`). That fixes the pre-baked copy for the same UID. **And it matters little**: hooks are advisory (H-5, done), and an agent with bash bypasses `git_guard.sh` without touching it (aliases, scripts, `env git`) |
| R7 | **GCP metadata server → lane SA token → Secret Manager / GCS / Pub/Sub / KMS** | `curl -H Metadata-Flavor:Google http://169.254.169.254/…/token` | **No.** Any UID in the container reaches `169.254.169.254`. M-SEC2 notes it is outside the egress lock's VPC path | The controls are M-SEC2 IAM (lane-scoped secrets) and the egress lock (bounds where stolen data can go). A UID split could stop this **only** with per-UID packet filtering or a network namespace for the child (spike S1/S2) |
| R8 | **`/artifacts` GCS mount** (gcsfuse, `read_only = false`, every lane, objectAdmin) | Read and write every task's artifacts. **This includes past tasks' Claude OAuth credentials**: see F-H6-1 | No. gcsfuse presents one uid/gid for the whole mount, and authorization is the job SA's IAM, not file modes. Splitting the UID cannot make one task's prefix unreadable to the child | F-H6-1 fix (Phase 0) plus M-SEC2 SEC2.5 (versioning, per-lane buckets) |
| R9 | **The materialized agent policy** (0444 file in a 0555 dir, outside the workspace, `agent_policy.go:46-56`) | A same-UID child can `chmod` it back and rewrite it | Yes: the dir becomes parent-owned | Not needed today: the only lane that reads it (`ailang_only`) has no bash. Noted for completeness |
| R10 | **The parent's post-run git**: `git add`/`git commit`/`git push` run *by the parent* inside the agent-written tree (`coordinator_cloud.go:563-600`, Step 5c) | `.git/hooks/pre-commit`, `core.fsmonitor`, `core.hooksPath`, `filter.*.clean` + `.gitattributes`, `include.path` in `.git/config`. The agent plants them and the **parent** runs them | **This is how a UID split gets defeated, not a gain.** Today it is harmless only because both sides are the same UID. After a split, the parent running git in a child-owned repo is the escalation path back to the parent UID | Must be designed as part of the split (D4). Without it, the split is cosmetic |

### F-H6-1: Claude OAuth credentials are persisted to the shared artifacts bucket

Found while tracing R5, and **independent of whether H-6 goes ahead**:

- `execute-job` sets `CLAUDE_CONFIG_DIR=/artifacts/tasks/<taskID>/claude` (`coordinator_cloud.go:523-529`)
  so session JSONL streams to GCS.
- `writeCredentialsFile` (`claude_auth.go:114-123`) then writes the OAuth credential
  (`accessToken` + `refreshToken` for the Claude Max subscription) **into `CLAUDE_CONFIG_DIR` as
  well**, mode 0600. gcsfuse ignores that mode, and the object is readable by every principal with
  read on the bucket.
- **Measured 2026-10-01** (object *names* listed only, nothing read): `tasks/*/claude/.credentials.json`
  exists **56×** in dev (2026-04-23 → 2026-08-25), **4×** in test, and **8×** in prod
  (2026-04-28 → 2026-08-27). The prod bucket has no lifecycle rule. Every lane mounts
  `/artifacts` read-write, **including the external (apikey) lanes**, which by design must never
  hold the Max OAuth credential (multivac `cloud_run_jobs.tf` header).
- **Why nothing newer than 2026-08-27:** the claude-lane job's execution history matches the
  objects. Dev's executions before today end on 2026-08-25 18:30 and prod's latest is 2026-08-27
  05:42. The only later ones are two dev SEC2 probe executions today
  (`sec2probe-default-093320`, `sec2probe2-default-093758`), both **failed**, leaving an empty
  `tasks/<id>/claude/` and no credential object. So the gap is explained by "no completed
  claude-lane runs", not by a fix. The write path (`claude_auth.go:114-123` +
  `coordinator_cloud.go:523-529`) is unchanged on `dev`. **That a deployed image still does this
  is an inference** from the code plus that history. Deployed images come from promoted tags,
  which have not been diffed here. P0.1's acceptance (AC1) therefore starts with one dev
  claude-lane task run *before* the fix, a listing to confirm or refute, and then the fix.

Whether these refresh tokens are still valid is **not established** (they may have rotated). That
should be treated as unknown, not as safe. Phase 0 P0.1 fixes the write, and the cleanup and
rotation decision is D1.

**Impact:** the UID split is a real boundary for exactly R1, R2, R4, R9, the parent-owned part of
R6, and the parent's part of R3. R1 and R2 have a 10-line same-UID fix. R5 and R8 are out of
reach of any UID scheme. R7, the most important path, is untouched unless the split also carries
network policy. R10 turns into a new hole if the split ships without a git-ingestion redesign.

## The deciding question: does the child keep the metadata server?

Under M-SEC2 the job's lane SA **must** hold `secretAccessor` on every secret the job injects,
because Cloud Run resolves `secret_key_ref` as the job's SA. The child can mint that SA's token
from the metadata server (R7). So:

> **With the metadata server reachable, the child can already read every secret that sits in the
> parent's environment, and can read the SSH deploy key from Secret Manager directly.** Protecting
> `/proc/$PPID/environ` (R1) or a parent-owned `~/.ssh` (R4) then only raises the cost from one
> `cat` to one `curl` plus one REST call.

That is not worthless. H-4 makes the same trade, because a casual `printenv` exfil is the common
injection payload and the metadata dance is not. But it means **H-4 and H-6 protect lane secrets
from a determined child only if the child cannot reach `169.254.169.254`**. A UID split is the
precondition for a per-UID network rule (`iptables -m owner --uid-owner agent -d 169.254.169.254 -j REJECT`)
or for a child network namespace. The parent keeps metadata, because it publishes the completion,
writes GCS and Secret Manager, and fetches the deploy key. Whether Cloud Run gen2 gives a
container root the capabilities for either (`CAP_NET_ADMIN` / `CAP_SYS_ADMIN`) is **unknown** and
is the first thing the spike measures.

Some CLIs authenticate *through* the metadata server (ADC: Gemini on Vertex, any Vertex-routed
model). Denying the child metadata therefore needs a per-lane inventory, because those lanes would
need a token minted by the parent and handed to the child.

## Goals

**Primary Goal:** a prompt-injected agent CLI cannot read or modify the executor parent's
credentials, memory, or launcher-owned files. Where the platform allows it, it also cannot mint
the lane SA's token. Every protection claimed has an adversarial probe that proves it.

**Success Metrics:**
- F-H6-1: `tasks/*/claude/.credentials.json` objects created after Phase 0 lands: **0** in all
  envs (bucket listing).
- `cat /proc/$PPID/environ` and `/proc/1/environ` from the child: **EACCES** (Phase 0, same UID).
- Write to `/plugins/ailang_bootstrap/**` from the child: **EACCES** (Phase 0).
- (Full split) child reads of the **parent's** `/home/ailang/.ssh`, `/home/ailang/.gitconfig` and
  the parent's push credential: **EACCES**. On self-push lanes the child holds only its own scoped
  per-task token (`/home/agent/.git-credentials`); on other lanes it holds no git credential. A planted `.git/hooks/pre-commit`, `core.fsmonitor` or `filter.*.clean` in the
  agent tree **never executes as the parent UID** (probe asserts on a sentinel file).
- (Full split, only if spike S1/S2 = feasible) child `curl` to `169.254.169.254` fails, and the
  parent's completion publish still succeeds.

## High-Impact Decisions

These are proposals for Mark's approval, not ratified decisions.

| ID | Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----|----------|-----------------|-----------|----------|-------------|
| D1 | **F-H6-1 cleanup**: stop writing credentials to `CLAUDE_CONFIG_DIR` when it is under `/artifacts`. Instead, point the CLI at a local credential and copy only the JSONL. Delete the 68 existing objects. **Rotate the Claude OAuth credential**, or record that we accept not rotating | A live credential exposure across lanes, including the external lanes. Rotation interrupts every Max-backed lane until the secret is re-set | human | design (now) | low (code) / med (rotation) |
| D2 | **Take Phase 0 (same-UID hardening) independently of H-6**: `PR_SET_DUMPABLE=0` in the parent, root-owned `/plugins` at build, no fleet token literal in `~/.gitconfig` once H-4 D2 lands | Closes R1/R2 and the pre-baked half of R6 for ~1.5 days, and removes most of the split's marginal value | human | design | low |
| D3 | **Split mechanism, if the spike says go**: container starts as root, and `execute-job` drops to `ailang` keeping **exactly** `CAP_SETUID`, `CAP_SETGID` and `CAP_KILL` (the "capability-holding parent"). No DAC capability: every parent access to child-owned files goes through a one-shot process spawned as `agent`, or happens before the `chown`. Adding any capability to this set is a design change that needs re-ratification, never an implementation choice. The child is spawned with `SysProcAttr.Credential{agent}` and `NoNewPrivs`. Alternatives: a setuid launcher, `setpriv`/`su-exec`, sudo, or a root parent | Defines what runs privileged in every job. A wrong choice adds an escalation surface (setuid binary) or a root process that parses agent output | human | design | high |
| D4 | **Git ingestion after the child exits**: the parent never runs git *with the agent's tree as its repository*. It fetches the agent's commits into a parent-owned clone under a hardened `-c` set, and copies working-tree changes symlink-safely (R10) | Without this the split is cosmetic: the agent writes `.git/config` and the parent runs it | human | design | high |
| D5 | **Per-UID metadata denial** (spike S1/S2): if feasible, does the child lose `169.254.169.254`, and which lanes need a parent-minted token handed over instead (ADC/Vertex lanes)? | The only part of H-6 that protects lane secrets from a determined child, and the one that changes the verdict from park to do | human | after spike | high |
| D6 | **Mac Studio host**: treat the local executor (coordinator `launchd` job, Daneel) as a separate track with a dedicated macOS user for the CLI, not as part of this cloud design | The Mac child currently runs as the operator account, with operator `gcloud`, `gh` and ssh credentials. That threat is different and larger | human | design | med |

### Design Freeze

- [ ] D1: F-H6-1 cleanup and the rotation decision. **This does not wait on anything else.**
- [ ] D2: approve Phase 0 as its own sprint.
- [ ] Approve the spike (Phase 1, ~1 day, dev only) and its go/no-go rule ([Verdict](#verdict-value-vs-effort)).
- [ ] D3, D4, D5: ratify only after the spike report. They are listed here so the spike measures what they need.
- [ ] D6: Mac Studio as a separate doc (yes/no).

## Feasibility

### Cloud Run execution environment

- **Every agent job runs gen2.** Terraform's `google_cloud_run_v2_job` resources do not set
  `execution_environment` (`cloud_run_jobs.tf`; only `cloud_run.tf:21` sets it, for the
  coordinator). The **live** dev job template carries
  `run.googleapis.com/execution-environment: gen2`, which the `/artifacts` gcsfuse volume
  requires. Gen1 (gVisor) does not apply, so there is no gVisor ptrace or `/proc` emulation to
  reason about. Gen2 runs a full Linux kernel in a microVM, so standard `/proc`, Yama and
  `PR_SET_DUMPABLE` semantics are expected. These are **verified in the spike**, not assumed.
- **What runs as which user today.** `Dockerfile.agent-base` ends with `useradd -m ailang`,
  `chown -R ailang:ailang /workspace /plugins`, `USER ailang`. Every lane image (`agent`,
  `agent-codex`, `agent-pi`, `agent-opencode`, `agent-motoko`, `agent-eval`, `-go` variants)
  switches to `USER root` only to `npm install -g` and switches back. So the global CLIs and
  `/opt/motoko_agent` are root-owned already, and `/plugins` is not.
- **Claude Code refuses `--dangerously-skip-permissions` as root** (stated in the H-6 request,
  and the reason the image is non-root). So whatever mechanism is chosen, **the child must be non-root**.

### Options

| Option | How | Verdict |
|---|---|---|
| A. Capability-holding parent (**recommended if go**) | `USER root` in the image. `execute-job` first: `setresgid/setresuid(ailang)` with `PR_SET_KEEPCAPS`, then reduce the effective/permitted sets to `CAP_SETUID`, `CAP_SETGID` and `CAP_KILL` (the last because `proctree.KillGroup` must signal a process group the parent no longer owns). Set `PR_SET_DUMPABLE=0`. Spawn the child with Go's native `SysProcAttr.Credential{Uid,Gid,Groups}` plus `Setpgid` (already set by `proctree.SetGroup`) and `NoNewPrivs` via the `prctl` in the forked child. The child's `execve` as a non-root UID with no file caps drops every capability | No new binary, no setuid file. The privileged part is the trusted parent, which is never model-facing. Go applies `setuid` to all threads since 1.16. Cost: capability code (`golang.org/x/sys/unix` `Capset`, or libcap's `cap`/`psx`), plus the requirement that gen2 grants these caps to container root (spike S4) |
| B. Root parent | Run `execute-job` as root throughout and spawn the child as `agent` | Simplest. But the parent parses agent output and runs git on agent-produced content (R10) **as root**, and any bug in D4 then hands over root. Rejected unless A is infeasible |
| C. Setuid-root launcher | Non-root parent calls a small setuid helper that switches to `agent` and `exec`s the CLI, refusing unless `getuid()==ailang` | Works without starting as root. But it adds a setuid binary on disk that the *child* can also invoke, so it has to be argument-validated forever. The parent still cannot signal the child's group without `CAP_KILL` (it would have to go through the helper). Rejected |
| D. `setpriv` / `su-exec` / `sudo` | Wrap the CLI command | Still requires `CAP_SETUID` in the caller (`setpriv`, `su-exec`) or a setuid binary plus policy (`sudo`), so it reduces to A or C with an extra dependency. `sudo` additionally brings PAM and env-scrubbing surprises. Rejected |
| E. User namespaces | `unshare -U` for the child | Unprivileged userns availability in gen2 is unknown (spike S2). Mapping more than one UID needs setuid `newuidmap`. A userns would also give the child `CAP_SYS_ADMIN` *inside* it, which needs careful reasoning. Kept only as the vehicle for a child **network** namespace if S2 shows it works |

### The Mac Studio host

The local coordinator runs from `~/Library/LaunchAgents/dev.ailang.coordinator.plist` as
`/bin/bash -lc "exec ailang coordinator start"`: the operator account (uid 501) under a login
shell, with the full profile environment and a coordinator API key in the plist's
`EnvironmentVariables`. Every local agent CLI it spawns runs as uid 501 and can read
`~/.config/gcloud` (the operator's ADC, an owner of the ailang projects), `~/.config/gh`,
`~/.ssh`, `~/.claude/.credentials.json`, and every checkout under `~/dev`. macOS has no
`/proc`. Reading a same-UID process's memory needs `task_for_pid`, which is entitlement-gated, so
memory is mostly out of reach. Same-UID **environment** reads through `ps -E`/`KERN_PROCARGS2` are
expected to work (spike S6). There is no metadata server and no lane SA, so **the only boundary
on the Mac is the UID**. That makes a split *more* valuable there than in the cloud: the mechanism
would be a dedicated `_ailangagent` user, a group-readable workspace, and a `sudoers` rule limited
to the CLI launcher (or a `launchd` job per user). That is host-ops work in the Daneel estate with
a different threat model, so D6 proposes a separate doc.

## Solution Design

### Overview

Three tiers, each independently shippable, each with an explicit stop point:

1. **Phase 0, same-UID hardening (do now).** F-H6-1 fix, `PR_SET_DUMPABLE=0`, root-owned
   `/plugins`, an adversarial probe test. No UID change.
2. **Phase 1, spike (1 day, dev).** Measure S1–S6 below. Write a one-page report into this doc and
   make the go/no-go call by the rule in the Verdict.
3. **Phases 2–4, the split (only on go).** Capability-holding parent (D3), file ownership layout,
   git ingestion (D4), per-UID metadata denial (D5), per-lane rollout.

### Spike checklist (Phase 1)

Run one throwaway dev job execution on an image built with `USER root` and a probe entrypoint.
Never run it in test or prod.

| ID | Question | Instrument |
|---|---|---|
| S1 | Does container root on gen2 hold `CAP_NET_ADMIN`, and does `iptables -m owner --uid-owner` filter `169.254.169.254` for one UID only? | `grep Cap /proc/self/status` + `capsh --decode`; install `iptables` in the probe image; curl metadata as both UIDs |
| S2 | Can the child get its own network namespace (`CAP_SYS_ADMIN` or unprivileged userns) with a veth/proxy back to the parent? | `unshare -n true`; `unshare -Ur true`; `/proc/sys/user/max_user_namespaces` |
| S3 | Yama `ptrace_scope`, and whether `PR_SET_DUMPABLE=0` makes `/proc/1/environ` EACCES for a same-UID process | `cat /proc/sys/kernel/yama/ptrace_scope`; probe binary sets dumpable 0, child `cat`s |
| S4 | Does container root hold `CAP_SETUID`, `CAP_SETGID` and `CAP_KILL`, and does `setresuid` + keepcaps work under gen2? | Probe binary implementing option A's first 20 lines |
| S5 | gcsfuse volume ownership as seen from both UIDs. Can the Cloud Run GCS volume take `uid=`/`gid=` mount options so the child can write session JSONL? | `stat /artifacts`; terraform `mount_options` in a dev-only branch, plan only |
| S6 | (Mac) Can uid 501 read another uid-501 process's environment via `ps -E` on the Studio's macOS build? | Local, one command, no change |

### Architecture of the full split (Phases 2–4, only on go)

**UIDs and groups:** `ailang` (uid 1000, the parent) and `agent` (uid 1001, the child), plus a
shared group `agentio`. In the image: `useradd -m -u 1001 agent`, and `/home/agent` is the child's
`HOME`.

**File ownership layout** (✱ = changed from today):

| Path | Owner : mode | Child access | Why |
|---|---|---|---|
| `/usr/local/bin/ailang`, global npm CLIs, `/opt/motoko_agent` | root : 0755 | r-x | Unchanged (already root-owned) |
| `/plugins/ailang_bootstrap` (pre-baked) ✱ | root : 0755/0644 | r-x | Tamper-proof hooks and skills. Phase 0 already does this |
| `/plugins/<taskID>/…` (runtime clone) ✱ | ailang : 0755/0644 | r-x | Parent clones, child reads |
| `/home/ailang` (`~/.gitconfig`, `~/.ssh`, the **parent's** push credential, `~/.ailang/hooks`, `~/.ailang/claude/settings.json`) ✱ | ailang : 0700 | **none** | R3 (parent part), R4, R6 (telemetry hook). The parent's push and PR credential, used for Steps 5c/5d from the parent clone |
| Agent policy dir ✱ | ailang : 0555 / file 0444 | r | R9: the child can no longer `chmod` it |
| `/home/agent/.claude/.credentials.json`, `/home/agent/.codex/auth.json`, `/home/agent/.pi/…` ✱ | agent : 0600 | rw | R5: the CLI must read **and refresh** its own auth. Not protectable, by construction |
| `/home/agent/.gitconfig` + `/home/agent/.git-credentials` ✱ (**only on self-push lanes**: `AILANG_PUSH_BRANCH` set or the agent's git mode permits push) | agent : 0600 | r | The **child's** push credential: H-4 D2's scoped per-task token, installed by the parent before the `chown` (`credential.helper=store --file=/home/agent/.git-credentials`). On every other lane the file is absent and the child's git has no credential at all. This is distinct from the parent's own credential in `/home/ailang` |
| `/workspace/<taskID>` (clone) ✱ | agent : 0755 after the parent `chown`s it post-clone | rw | The child's working tree. **Untrusted to the parent from this point on** (D4) |
| `/artifacts` (gcsfuse) | one uid per mount (S5) | rw | R8: IAM, not file modes, governs it |

**What is actually protected, stated plainly:** the parent's environment and memory (Phase 0
already does this for the same UID), the parent's git and SSH credentials, the launcher-owned
hook, telemetry and policy files, and the parent process itself (signals, ptrace). What **stays
reachable**: the CLI's own inference credential (R5), the artifacts bucket (R8), the workspace and
everything in it, and, unless D5 lands, the metadata server and therefore every secret on the
lane's SA (R7).

**Capability contract:** the post-drop set is exactly {`CAP_SETUID`, `CAP_SETGID`, `CAP_KILL`}
and a test asserts it (`/proc/self/status` CapEff/CapPrm). There is deliberately no
`CAP_DAC_OVERRIDE`/`CAP_DAC_READ_SEARCH`. Parent work on child-owned files is ordered before the
`chown`, or done by a one-shot child spawned as `agent` (codex read-back, the JSONL copy if S5
leaves `/artifacts` child-owned).

**Spawn path (option A):** one helper, `executor.ConfigureChildIdentity(cmd)`, called next to
`proctree.Configure(cmd)` in the five executors that spawn a CLI (claude, codex, pi, opencode,
motoko). It merges `Credential` and `NoNewPrivs` into the `SysProcAttr` that `proctree.SetGroup`
already creates, and sets `HOME=/home/agent` in the env that H-4's canonical builder produces (one
seam, not five). It is a **named error, never a fallback**, if the parent is still uid 0 or the
child UID equals the parent UID, unless `AILANG_EXECUTOR_UID_SPLIT=off` is set explicitly. The
local and Mac path keeps that flag off.

**Credential install (per executor):**

| Executor | Credential the child needs | Install as | Parent reads back? |
|---|---|---|---|
| claude | `.credentials.json` (OAuth) or `ANTHROPIC_API_KEY` (apikey mode, env via H-4) | parent writes into `/home/agent/.claude/` and `chown agent`. **Never into `CLAUDE_CONFIG_DIR` on `/artifacts`** (F-H6-1) | No |
| codex | `~/.codex/auth.json` (subscription), refreshed by codex | parent writes, `chown agent` | **Yes**: write-back to Secret Manager after the run (`coordinator_cloud_codexauth.go:96`). The parent reads the child-owned file **by spawning a one-shot read as the `agent` UID** (the parent re-execs itself, `ailang` reading one fixed path and writing it to a pipe, under the same `SysProcAttr.Credential` it already uses for the CLI). That uses only `CAP_SETUID`/`CAP_SETGID`, no DAC capability. Codex rewrites the file on refresh, so group-read modes cannot be relied on. Parse with the existing `ParseSubscriptionAuth` validation, because the content is now child-controlled input |
| pi / opencode | provider key in env (H-4's single ratified grant) | env | No. pi's global extensions under `~/.pi/agent/extensions` move to `/home/agent`. The parent's collision-resolver (`pi_extension_collision.go`) runs **before** the home and workspace are `chown`ed to `agent`, so it needs no capability |
| motoko | provider key in env; per-task `AILANG_CACHE_DIR` (`motoko/isolation.go`) | env; cache dir created by parent and `chown agent` | No |

**Git ingestion (D4).** After the child exits, the parent treats `/workspace/<taskID>` as
untrusted data:

1. Never `git -C <agent tree> add/commit/push`. Those load the agent's `.git/config`, hooks,
   `.gitattributes` filters and fsmonitor.
2. The parent keeps its own clone, made before the child ran and owned by `ailang`. It **fetches**
   the agent's commits from the agent repo with `-c core.hooksPath=/dev/null -c core.fsmonitor=false
   -c protocol.file.allow=always`, an explicit `safe.directory=<agent tree>` for that one
   invocation, `GIT_CONFIG_GLOBAL=<parent-owned minimal file>` and `GIT_CONFIG_NOSYSTEM=1`.
   Fetching only *reads* objects, and git honours `uploadpack.packObjectsHook` only from protected
   config. Whether a local-path fetch loads any other executable config key from the source repo
   is **spike/test item T4**, proven with a hostile-config fixture.
3. Uncommitted working-tree changes (today's Step 4/5a `stageForCommit`) are copied into the
   parent clone with a copier that **never follows symlinks** (it stores links as links). That
   blocks the "plant a symlink to `/home/ailang/.gitconfig` and let the parent commit its content"
   exfil. The parent then runs `add`/`commit` in **its own** clone, under the same hardened `-c`
   set.
4. Push, PR creation and the completion publish run from the parent clone, as today.

The confined-git work in `internal/effects/process_confined.go` (hooks off, fsmonitor off, global
and system config nulled) is the precedent. D4 applies the same discipline to the launcher. Note
that debian bookworm's git (2.39) already enforces `safe.directory`, so a careless implementation
that runs git as `ailang` in an `agent`-owned tree **fails closed** ("dubious ownership") rather
than silently trusting the config. That is a useful tripwire, but not a design.

**Per-UID metadata denial (D5, only if S1 or S2 is green).** The parent installs one rule at
startup, before dropping caps: `iptables -A OUTPUT -m owner --uid-owner agent -d 169.254.169.254 -j REJECT`
(S1). The alternative is to put the child in a network namespace whose only route is a
parent-owned forward proxy that refuses link-local addresses (S2). Lanes whose CLI authenticates
by ADC get a short-lived, minimal-scope access token minted by the parent and passed through H-4's
grant mechanism. A lane inventory decides which lanes those are, and with the gemini executor jobs
retired (multivac `cca6912`) it may be none. This composes with the egress lock: the lock bounds
where data can go, and the rule removes the token that reads it.

### Implementation Plan

**Phase 0: same-UID hardening (~1.5 days), independent of the verdict**
- [ ] P0.1 (F-H6-1): `writeCredentialsFile` refuses to write into a `CLAUDE_CONFIG_DIR` under
      `/artifacts` (named error). `execute-job` points `CLAUDE_CONFIG_DIR` at a **local** dir
      (credentials live there) and makes its `projects/` entry a **symlink** to
      `/artifacts/tasks/<id>/claude/projects/`. Claude then still appends the session JSONL
      **directly to the gcsfuse path while the task runs**, exactly as today. No post-run-only
      copy is introduced, so an OOM, preemption or `KillGroup` timeout loses no more log than it
      does today. (The existing code already notes that gcsfuse staged appends may not flush
      before exit and re-writes `session.jsonl` at the end, `coordinator_cloud_github.go:538-557`.
      That is unchanged.) `findSessionJSONL` uses `filepath.WalkDir`, which does not follow a
      symlinked directory, so it is pointed at `/artifacts/tasks/<id>/claude` directly. The object
      layout under `/artifacts` is byte-for-byte the same, so artifact readers need no change. Add
      a regression test that no credential file lands under the artifact root. Clean up the
      existing objects per D1 (multivac ops, out of repo)
- [ ] P0.2: `prctl(PR_SET_DUMPABLE, 0)` at the start of `execute-job`. `DisableDump()` returns a
      typed error on failure **and** on unsupported platforms (`ErrDumpProtectionUnsupported`); it
      never pretends. `execute-job` (cloud, always Linux) treats any error as **fatal** before
      dispatching the CLI. The local coordinator path (darwin) does not call it in Phase 0. If it
      later does, it must log a named warning and record `parent_dump_protected=false` on `Result`,
      so the gap is visible in the data rather than silently assumed
- [ ] P0.3: `Dockerfile.agent-base`: `/plugins` and the pre-baked `/plugins/ailang_bootstrap`
      stay root-owned (0755/0644). The build creates **one** writable subdirectory,
      `/plugins/tasks` (owner `ailang`), and the runtime per-task clone moves from
      `/plugins/<taskID>/ailang_bootstrap` to `/plugins/tasks/<taskID>/ailang_bootstrap`
      (`coordinator_cloud.go:364`). The parent, still non-root `ailang` in Phase 0, can create it.
      **Honest limit:** in Phase 0 a runtime-cloned plugin is still same-UID writable. Only the
      pre-baked copy becomes tamper-proof. The split (Phase 2) closes the runtime copy
- [ ] P0.5: positive-path tests: (a) a task with a runtime plugin repo still clones it and the CLI
      receives `--plugin-dir /plugins/tasks/<id>/…`; (b) the pre-baked fallback still resolves;
      (c) one dev probe task on the claude lane after the image change, before the tag
- [ ] P0.4: adversarial probe test (Linux CI): spawn a child through the real executor spawn path,
      and have it try `/proc/$PPID/environ`, `/proc/$PPID/mem` and a write under a root-owned dir.
      Each must be EACCES/EPERM

**Phase 1: spike (~1 day, dev, throwaway image)**
- [ ] Measure S1–S6. Append a "Spike report" section to this doc. Apply the go/no-go rule

**Phase 2: capability-holding parent + child UID (~4 days, only on go)**
- [ ] Image: `USER root`, `agent` user, ownership layout above, for all lane images
- [ ] `execute-job` privilege drop (option A) with a named error on failure.
      `ConfigureChildIdentity` wired into the five spawners. `proctree` kill verified across UIDs
- [ ] Per-executor credential install as `agent`. codex read-back validated

**Phase 3: git ingestion (~4 days, only on go)**
- [ ] Parent-owned clone, hardened fetch, symlink-safe copy, commit/push from the parent clone
- [ ] Hostile-repo fixture suite (T1–T5 below)

**Phase 4: metadata denial + rollout (~2–4 days, only if S1/S2 green)**
- [ ] Per-UID rule or netns. ADC-lane token handoff if any lane needs it
- [ ] Rollout per lane, see Rollout

### Files to Modify/Create

**Phase 0:**
- `internal/executor/claude/claude_auth.go` — refuse to write credentials under the artifact root (~15 LOC)
- `cmd/ailang/coordinator_cloud.go` — local `CLAUDE_CONFIG_DIR` plus a post-run JSONL copy; `PR_SET_DUMPABLE` at entry (~30 LOC)
- `internal/proctree/dumpable_linux.go` — new: `DisableDump()` Linux implementation; the non-Linux build returns `ErrDumpProtectionUnsupported` and is never a silent no-op (~30 LOC)
- `docker/Dockerfile.agent-base` — drop `/plugins` from the `chown`; create `/plugins/tasks` owned by `ailang` (~3 lines)
- `cmd/ailang/coordinator_cloud.go` — the runtime plugin clone path moves to `/plugins/tasks/<taskID>` (also listed above; ~2 LOC of the total)
- `internal/executor/uidprobe_linux_test.go` — new adversarial probe test (~120 LOC)

**Phases 2–4 (only on go):**
- `internal/executor/identity_linux.go` — new: `ConfigureChildIdentity` and the privilege drop (~150 LOC)
- `internal/executor/claude/claude.go` — one `ConfigureChildIdentity` call; the same pattern repeats in `codex/codex.go`, `pi/pi.go`, `opencode/opencode.go` and `motoko/motoko.go`, plus credential-install ownership (~60 LOC total)
- `cmd/ailang/coordinator_cloud.go` — restructure Steps 1, 4 and 5 around the parent-owned clone (~250 changed LOC)
- `cmd/ailang/coordinator_cloud_ingest.go` — new: hardened fetch and symlink-safe copy (~200 LOC)
- `docker/Dockerfile.agent-base` — `USER root`, `agent` user, ownership; the lane images inherit it (~10 lines; small follow-ups in the 7 lane Dockerfiles)

Outside this repo: if S5 requires gcsfuse mount options, those go in ailang-multivac's
terraform/cloud_run_jobs.tf (a separate PR there), together with the per-lane
`AILANG_EXECUTOR_UID_SPLIT` env var.

## Conflict Surface

Not a parser, typechecker or codegen change. The surface is enumerated anyway because the launcher
is shared by every cloud lane:

| Existing behaviour | Requirement / intentional change |
|---|---|
| `proctree.Configure` / `KillGroup` (kill a process group on timeout, `proctree.go:27-42`) | **Must still work across UIDs**: the parent keeps `CAP_KILL` (option A). Test: a timeout kills a child-UID grandchild |
| `proctree.SetGroup` creates `SysProcAttr` with `Setpgid` | `ConfigureChildIdentity` **merges** into it and never replaces it (otherwise process-group kill silently breaks) |
| Agents that commit and push themselves (`AILANG_PUSH_BRANCH`, guardrails mode; `branchNeedsPush` handles "agent pushed its own branch") | Preserved: on those lanes the parent installs H-4 D2's scoped token at `/home/agent/.git-credentials` with a child-owned `/home/agent/.gitconfig` pointing at it (ownership table). The parent's **fetch** in D4 then sees already-pushed commits, and `branchNeedsPush` logic is retained. Which lanes are self-push comes from H-4's migration inventory (its AC6). Before that inventory exists, the split is not enabled on a lane |
| Runtime per-task plugin clone into `/plugins/<taskID>` (`coordinator_cloud.go:361-380`) and the pre-baked fallback | **Phase 0 collision**: a root-owned `/plugins` makes `mkdir /plugins/<taskID>` EACCES for the non-root parent. Resolved by moving runtime clones to the `ailang`-owned `/plugins/tasks/` (P0.3) and pinned by the positive-path test P0.5 |
| `stageForCommit` excludes the agent scratch dir (`coordinator_cloud_scratch.go`) | The same exclusion applies to the symlink-safe copy |
| SSH deploy-key preflight `verifyDeployKeyWrite` runs **inside the clone** (`coordinator_cloud_sshkey.go:200-206`) | Runs inside the parent-owned clone, which stays a real repo |
| `CLAUDE_CONFIG_DIR` on `/artifacts` for session JSONL streaming; `findSessionJSONL` walks `CLAUDE_CONFIG_DIR` (`coordinator_cloud_github.go:538-570`) | **Preserved**: `projects/` is symlinked into the same gcsfuse path, so live appends and the object layout are unchanged. `findSessionJSONL` is pointed at the artifacts path because `WalkDir` does not follow the symlink. Only `.credentials.json` (and the rest of Claude's local config) leaves the bucket |
| codex auth write-back (`coordinator_cloud_codexauth.go`) | Preserved. The file is now child-controlled input and is validated before write-back |
| Local/Mac executor path (`ailang coordinator start`, mission dispatch) | Unchanged: the UID split is cloud-only behind the flag. `DisableDump` is called only by the Linux cloud `execute-job` (fatal on error); on darwin it returns a typed unsupported error, never a silent success |
| H-4 `BuildEnvironment` canonical builder | `HOME` for the child is set **there** (one seam). H-4's deny-list already keeps `HOME` out of `ExtraEnv` |

## Examples

### Example 1: the probe, before and after Phase 0

**Before (same UID, today):**
```text
agent$ tr '\0' '\n' < /proc/$PPID/environ | grep -E 'TOKEN|_KEY' | cut -d= -f1
GITHUB_TOKEN           # …every secret_key_ref the job was started with
agent$ echo x >> /plugins/ailang_bootstrap/scripts/hooks/git_guard.sh && echo tampered
tampered
```

**After Phase 0 (still same UID):**
```text
agent$ cat /proc/$PPID/environ
cat: /proc/1/environ: Permission denied
agent$ echo x >> /plugins/ailang_bootstrap/scripts/hooks/git_guard.sh
bash: …/git_guard.sh: Permission denied
agent$ curl -s -H Metadata-Flavor:Google http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token | head -c 20
{"access_token":"ya2  # ← still works; only D5 closes this
```

### Example 2: R10, why D4 is mandatory under a split

```text
# agent, before exiting:
git config core.fsmonitor '/bin/sh -c "cat /home/ailang/.ssh/* > /workspace/t/leak; false"'
# parent, naive Step 4/5a (today's code, after a UID split):
git -C /workspace/t add -A     # runs the fsmonitor AS ailang → key copied into the tree → committed → pushed in the PR
# with D4: the parent never runs git with /workspace/t as its repository; the hardened fetch disables core.fsmonitor
```

## Success Criteria

- [ ] AC1 (P0.1): (i) **before** the fix, one dev claude-lane task, then list
      `tasks/<id>/claude/.credentials.json`. This confirms or refutes that the deployed image still
      writes it. (ii) After the fix: a unit test that no credential file is written under the
      artifact root, and the same listing shows none. (iii) `tasks/<id>/claude/projects/…/*.jsonl`
      and `tasks/<id>/session.jsonl` still arrive at the same paths as today
- [ ] AC2 (P0.2): probe test, child → `/proc/$PPID/{environ,mem}` is EACCES on Linux CI and in one dev job run
- [ ] AC3 (P0.3): probe test, child write under `/plugins/ailang_bootstrap` is EACCES in the built image
- [ ] AC3b (P0.5): a task with a runtime plugin repo still clones into `/plugins/tasks/<id>/` and the CLI loads it. The pre-baked fallback still resolves
- [ ] AC4 (spike): S1–S6 answered with transcripts in this doc, and the go/no-go recorded with Mark's ratification
- [ ] AC5 (go only): T1–T5 hostile-repo fixtures never execute a sentinel as the parent UID
- [ ] AC6 (go only): every lane passes one probe task per variant in dev, the same gate M-SEC2 uses (multivac rollout table)
- [ ] AC7: `make test`, `make lint`, `make check-boundaries` green. Guide or ops note updated

## Testing Strategy

**Adversarial probe (the core test, runnable in CI on Linux and as a dev job):** a fixture "CLI"
script spawned through the real executor spawn path attempts:
1. read `/proc/$PPID/environ`, `/proc/1/environ`, `/proc/$PPID/mem`; `ptrace` attach (gdb-less: a small Go helper)
2. write to the pre-baked plugin hooks, `~/.ailang/hooks/claude_telemetry.sh`, the agent-policy dir (`chmod` then write)
3. (split) read `/home/ailang/.gitconfig`, `/home/ailang/.ssh/*`, the parent's push credential (expect denied); on a self-push lane, `git push` to the task branch with the child's own scoped token (expect allowed) and to another branch or repo (expect denied by the token's scope)
4. (split) `kill -9 $PPID`
5. (D5) curl the metadata token endpoint

It writes one line per attempt (`denied`/`ALLOWED`). The test asserts the phase's expected matrix.

**Hostile-repo fixtures for D4 (T1–T5):** T1 `.git/hooks/pre-commit`, T2 `core.fsmonitor`,
T3 `.gitattributes` + `filter.x.clean`, T4 `include.path` → a config that sets any of the above,
plus a hostile `uploadpack.*`, T5 a symlink in the tree to a parent-owned file. Each plants a
sentinel write, and the test asserts the sentinel is absent and the PR diff contains no
parent-owned content.

**Manual:** one dev dispatch per lane (claude, codex, pi, opencode, motoko, eval) with the probe
directive, the same as M-SEC2's probe-task rollout.

## Rollout

Phase 0 rides the normal path: ailang `dev` → `v*` tag → test → `promote`. It changes the image
(`/plugins` ownership) and the binary, nothing in multivac. D1 cleanup is a separate multivac ops
action.

The split (on go) rolls out **per lane, dev first**, behind `AILANG_EXECUTOR_UID_SPLIT`. That is
an env var on the job template (multivac terraform), so a lane can be rolled back by a config-only
push without an image change. Order: eval (least production impact) → pi → opencode → codex →
motoko → claude (the Max OAuth lane last). Each lane needs one green probe task in dev and one
real task before test. Test and prod follow by the usual infra-then-tag order (multivac CLAUDE.md).

## Verdict: value vs. effort

| Tier | Effort | Closes | Leaves open |
|---|---|---|---|
| Phase 0 (same UID) | ~1.5 days | F-H6-1 (a live leak), R1, R2, pre-baked R6 | R3/R4 parent files, R7, R8, R9 |
| + H-4 (planned) | 8–12 days (its own doc) | casual `printenv` exfil, `ExtraEnv` injection, the fleet token in the child | R7 still yields the lane's secrets to a determined child |
| + M-SEC2 + egress lock | landed / dev-only | *what* the lane SA can read, *where* data can go | the child can still mint the token |
| + full UID split, no D5 | ~10 days + spike | R3 (parent part), R4, R9, the rest of R6 | **R7**, so in practice the child can still fetch R4's deploy key from Secret Manager. The marginal gain is small |
| + full UID split **with D5** | ~12–14 days + spike | R7 for the child: the lane's secrets are no longer reachable by the model-facing process at all | R5 (its own inference credential), R8 (artifacts), by construction |

**Recommendation:**
1. **Do Phase 0 now**, F-H6-1 first. It is cheap and closes a real leak.
2. **Run the 1-day spike.**
3. **Go/no-go rule:** proceed to the full split **only if S1 or S2 shows that per-UID metadata denial
   works on gen2** (and S4 shows option A is possible). If not, **park H-6**. Without D5 the split
   costs ~2 weeks plus a git-ingestion redesign (R10) that is itself new risk, and it protects
   files whose contents the child can re-fetch through the metadata server. H-4 + M-SEC2 + the
   egress lock + Phase 0 then cover most of the remaining value. Re-open H-6 if Cloud Run gains
   per-container metadata controls, or if a lane appears whose parent holds a credential **not**
   readable by the lane SA (for example a coordinator-minted per-task GitHub App token handed in per
   execution).
4. **Mac Studio (D6):** a separate, higher-value track, because there the UID is the only boundary
   and the child runs as the operator.

## Deferred Decisions

- Capability library (`x/sys/unix` raw `Capset` vs libcap `cap`/`psx`): agent may choose, as long as the post-drop capability set is asserted in a test
- Child UID number and group name: agent may choose. Must be stable across images
- Spike report format: agent may choose. Must include raw transcripts

## Non-Goals

- **Stopping the child from using its own inference credential (R5).** Impossible by construction. Bounded by M-SEC2 and H-4.
- **Per-task isolation of `/artifacts` (R8).** M-SEC2 SEC2.5 (per-lane buckets, versioning) owns it. Phase 0 only removes credentials from it.
- **Making hooks a security boundary.** H-5 is settled: hooks are advisory. This design root-owns them for integrity, not as enforcement.
- **gVisor or a sandbox runtime for the child.** A different layer. Gen2 is required by the gcsfuse mount.
- **The Mac Studio implementation.** It goes in a separate doc if D6 = yes.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| R10 shipped without D4: the split becomes an escalation path into the parent | High | D4 is a hard prerequisite in the same phase. T1–T5 fixtures gate it. bookworm git's `safe.directory` fails closed as a tripwire |
| Gen2 does not grant the needed caps (S4), so option A is impossible | Med | The spike decides before any code. Fallback B (root parent) only with D4 shipped and Mark's explicit approval |
| P0.1's symlink is not honoured (the CLI resolves or replaces `projects/`) | Med | AC1(iii) checks that the JSONL arrives at the unchanged path in a dev run. Fallback: set `CLAUDE_CONFIG_DIR` on `/artifacts` as today but write credentials only to `~/.claude` and point the CLI there through its own credential-path setting, if the spike shows one. Never back to credentials in the bucket |
| CLI behaviour differs as a non-`ailang` user (HOME-relative caches, npm global paths, `~/.pi` extensions) | Med | Per-lane dev probe before test. The flag allows per-lane rollback by config push |
| Rotating the Claude OAuth credential (D1) interrupts Max-backed lanes | Med | Rotate in a quiet window. `make secrets` path per multivac CLAUDE.md §4b |
| Capability code is security-critical and rarely touched | Med | Small surface (one file), with an asserted post-drop cap set in tests and a named error on any deviation |

## Verification Log

| ID | Claim | Instrument / evidence | Result |
|---|---|---|---|
| V1 | Agent jobs run gen2, not gVisor gen1 | `grep execution_environment` multivac `terraform/` (only `cloud_run.tf:21`, coordinator); live `gcloud run jobs describe ailang-dev-agent-executor` → annotation `run.googleapis.com/execution-environment: gen2` | Confirmed (dev). Test/prod use the same terraform |
| V2 | Images run as non-root `ailang`; `/plugins` is `ailang`-owned; CLIs and `/opt/motoko_agent` are root-owned | Read `docker/Dockerfile.agent-base` (`chown -R ailang:ailang /workspace /plugins`, `USER ailang`), `Dockerfile.agent{,-codex,-pi,-opencode,-motoko,-eval}` (`USER root` only around installs) | Confirmed |
| V3 | Parent is PID 1 | `ENTRYPOINT ["/usr/local/bin/ailang"]` exec form, `CMD ["coordinator","execute-job"]` | Confirmed from the Dockerfile; the live PID is to be re-checked in the spike |
| V4 | Fleet GitHub token is written into global gitconfig as a literal credential helper | Read `cmd/ailang/coordinator_cloud.go:336-345` | Confirmed |
| V5 | Claude credentials are written to `CLAUDE_CONFIG_DIR`, which execute-job points at `/artifacts` | Read `claude_auth.go:114-123`, `coordinator_cloud.go:523-529`; `grep credentials.json` shows no cleanup path | Confirmed |
| V6b | The newest-object date matches the claude-lane run history rather than a fix | `gcloud run jobs executions list --job=<prefix>-agent-executor` (dev, prod) + `executions describe` of today's two dev runs (`AILANG_TASK_ID`, failedCount) + listing of their `tasks/<id>/` | dev: last runs before today 2026-08-25; today's 2 probe runs failed with an empty `claude/`. prod: latest 2026-08-27. Consistent with no completed runs. That deployed images still write is an inference (V5 is on `dev` source) and is re-checked first in AC1 |
| V6 | Credential objects exist in the artifacts buckets | `gcloud storage ls 'gs://<proj>-ailang-artifacts/tasks/*/claude/.credentials.json'` (names and dates only, contents **not** read) | dev 56 (2026-04-23→08-25), test 4, prod 8 (2026-04-28→08-27). No lifecycle rule on the prod bucket |
| V7 | Every lane, including external, mounts `/artifacts` read-write | multivac `cloud_run_jobs.tf` volume blocks (`read_only = false` at 190, 371, 542, …); the apikey job at 224 has the mount | Confirmed |
| V8 | Lane SA can read every secret the job injects | multivac `terraform/iam.tf` `agent_lane_secrets` (internal lane: 7 secrets) and the `cloud_run_jobs.tf` header ("A lane SA can read ONLY the secrets in … agent_lane_secrets") | Confirmed. This is the basis of the "metadata ⊇ parent env" argument |
| V9 | Metadata server is outside the egress lock | M-SEC2 doc ("Not covered: the metadata server (169.254.169.254) is outside the VPC path") | Confirmed (doc) |
| V10 | Hooks are rewritten at runtime by the parent in the same `$HOME` | `internal/executor/environment.go:420-452` (`GetClaudeSettingsPath` writes `~/.ailang/hooks/claude_telemetry.sh` 0755 and settings 0644) | Confirmed |
| V11 | Parent runs git add/commit in the agent tree after the run | `coordinator_cloud.go:563-600` (`stageForCommit`, `git -C workDir commit`) | Confirmed |
| V12 | No privilege-drop, credential-switch or dumpable control exists in the repo today (negative) | `grep -rn "syscall.Credential\|Credential{\|PR_SET_DUMPABLE\|Setresuid\|Setuid(\|Prctl\|NoNewPrivs\|Pdeathsig" --include='*.go'` (non-test) | Only unrelated `AnthropicCredential{…}` and `codexCredential{…}` struct literals. **None exists** |
| V13 | Launcher git has no hooks/fsmonitor/safe.directory hardening (negative) | `grep -rn "safe.directory\|core.hooksPath\|fsmonitor" cmd/ailang/ internal/gitexec` (non-test) | Empty. The hardening exists only in `internal/effects/process_confined.go` for confined git (audit §2) |
| V14 | Process-group kill depends on signalling the child's group | Read `internal/proctree/proctree.go:27-42` (`Configure` → `SetGroup`, `Kill` → `KillGroup(pid)`) | Confirmed, so `CAP_KILL` is required under option A |
| V15 | Codex auth is read back by the parent after the run | `cmd/ailang/coordinator_cloud_codexauth.go:25-96` (install before, write-back after) | Confirmed |
| V16 | Mac local coordinator runs as the operator account under a login shell | `plutil -p ~/Library/LaunchAgents/dev.ailang.coordinator.plist` (`/bin/bash -lc "exec …/ailang coordinator start"`); `id` → uid 501 | Confirmed (the plist also holds a coordinator API key in plaintext env; its value is deliberately not reproduced here) |
| V17 | H-4 scope: env only, plus a credential file "outside the workspace" | Read PR #1417's `design_docs/planned/v0_49_1/m-executor-env-hardening.md` (D1–D6, Non-Goals: "OS-level sandboxing … container hardening … different layers") | Confirmed. H-6 owns the process and file layer, and its D2 file is only protected from the child under a split |
| PENDING | Yama scope, caps available to container root, iptables/netns, gcsfuse uid options, macOS `ps -E` | Spike S1–S6 | **Unmeasured.** The verdict is gated on these |
| V18 | Duplicate/coverage gate | `create_planned_doc.sh` neural search: top planned 0.35 (`m-codex-billing-lane-resolution`), top implemented 0.49 (`M-DX11-STRING-SPLIT`, an unrelated keyword match on "split"). Read H-4 (#1417), M-SEC2 and the audit for scope | No overlap. H-4 is env, M-SEC2 is IAM and network, this doc is process and file |
| V19 | No AILANG language claims | N/A | This design changes no language surface; `ailang check` gate not applicable |

**Quorum trigger:** #1 fires (design-freeze items) and #4 fires (load-bearing premises about Cloud
Run gen2, an external system). The external-system premises are deliberately labelled PENDING and
gated behind the spike rather than asserted, per the skill's "stop and hand over" rule.

**Quorum round 0 (2026-10-01, $0.31, blocked 3 of 3 present reviewers; gpt6-1-sol unreachable):**
every objection was accepted and fixed in this revision.
(1) glm-5.3: P0.3 root-owned `/plugins` would break the runtime per-task plugin clone, and no test
covered the positive path. Fixed: runtime clones move to `ailang`-owned `/plugins/tasks/`, the
Conflict Surface row and positive-path test P0.5/AC3b were added, and the honest limit is stated.
(2) kimi-k3: D3's capability set {SETUID, SETGID, KILL} contradicted the DAC capabilities used
by the codex read-back and pi rows. Fixed: no DAC capabilities. The read-back runs as a one-shot
`agent`-UID process, the collision-resolver runs before the `chown`, the set is asserted in a
test, and any addition needs re-ratification. (3) gemini-3.1-pro: the non-Linux `DisableDump`
no-op stub was a silent fallback. Fixed: it returns a typed unsupported error, cloud treats any
error as fatal, and a local use must record `parent_dump_protected=false`.

**Quorum round 1 (2026-10-01, $0.40, blocked 3 of 3 present; gpt6-1-sol unreachable): the
re-quorum-once guardrail is now spent.** All three objections were accepted and fixed here,
without a third round, so the doc goes to Mark with the fixes recorded. (1) glm-5.3: F-H6-1's
"the next run writes another" was an unverified forward claim. Fixed: the execution history was
read (V6b), the claim is labelled an inference, and AC1 confirms it first. (2) kimi-k3: the child
had no slot for its self-push credential, which contradicted the EACCES metric. Fixed: the
`/home/agent/.git-credentials` row, a parent-vs-child credential distinction in metrics and
probes, and a gate on H-4's push inventory. (3) gemini-3.1-pro: P0.1's post-run-only JSONL copy
would lose logs on hard exit. Fixed: a `projects/` symlink keeps live appends to the same gcsfuse
path, and the stale "check the dashboard" deferral is gone.

## Related Documents

- ailang-multivac `internal-docs/SECURITY-AUDIT-2026-10-01-ailang-executor-escape-hardening.md`: source finding (H-6), F-A5, H-4, H-5
- ailang-multivac `internal-docs/M-SEC2-cloud-executor-hardening.md`: lane SAs, secret-level IAM, egress lock (SEC2.3b), artifacts integrity (SEC2.5)
- ailang-multivac `internal-docs/design/git-guardrails.md`: "Not a Security Boundary" (H-5)
- [M-EXECUTOR-ENV-HARDENING](https://github.com/sunholo-data/ailang/pull/1417) (planned, PR #1417): the env half. Composes, does not overlap
- [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md) (v0.41.0): AILANG program runtime confinement. The confined-git hardening there is D4's precedent

**Auto-search results (none relevant, recorded per the gate):** implemented: `M-DX11-STRING-SPLIT-IMPLEMENTATION-REPORT` (0.49), `m-gemini-evaluator-diff-bridge` (0.35), `m-process-subcmd-allowlist-sprint-plan` (0.29). Planned: `m-codex-billing-lane-resolution` (0.35), `m-model-registry-single-source-sprint-plan` (0.34), `m-daneel-deep-research` (0.34).

## References

- [Design Axioms](/docs/references/axioms)
- Linux `proc(5)` (`/proc/pid/environ`, `mem` access checks), `prctl(2)` (`PR_SET_DUMPABLE`, `PR_SET_KEEPCAPS`, `PR_SET_NO_NEW_PRIVS`), Yama LSM docs, `capabilities(7)`
- git `safe.directory` (2.35.2+), `git-config(1)` protected configuration

## Future Work

- Per-task GitHub App tokens minted by the coordinator and passed per execution. This would make
  the parent hold a credential the lane SA **cannot** read, which is the scenario where a split
  without D5 becomes worthwhile.
- Mac Studio dedicated agent user (D6).

---

**Document created**: 2026-10-01
**Last updated**: 2026-10-01
