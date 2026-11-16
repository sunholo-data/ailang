# Sprint plan: M-FLEET-PI-SANDBOX-WIRING

**Ticket:** P0 #4, `pi-runner:sandbox-extensions-not-wired`  
**Base inspected:** detached `9a2bca8a2`; planning only.  
**Goal:** every `mission_pi_run.sh` executor and evaluator invocation uses both pi write fences and a validated mission policy, or emits a distinct failure before model work.  
**Estimate:** about 165 implementation LOC, 185 test LOC, 1–2 days including the live smoke. This is a loop-harness change; no routing, thresholds, lane order, or quota changes.

## Evidence and premise verdicts

| Premise | Verdict and evidence | Decision |
|---|---|---|
| P1 dependency resolution | **VERIFIED.** `ls -ld ~/dev/sunholo-data/ailang/tools/pi-extensions/sandbox/node_modules tools/pi-extensions/sandbox/node_modules` found the former only. `sandbox/index.ts` imports `@anthropic-ai/sandbox-runtime`; `package.json` pins `^0.0.71`. `git rev-parse --git-common-dir` resolves to the main checkout's `.git`. | Pass **absolute extension paths from the main checkout** (`dirname(realpath(git-common-dir))`), where the dependency resolves from `sandbox/index.ts`. Compare both extension sources and canonical policy to the current checkout before use; if absent or divergent, fail closed. Check `node_modules/@anthropic-ai/sandbox-runtime/package.json` before pi launch. Missing dependency gets `sandbox_unavailable`/rc 15. A bare `-e` without this check may produce a pi load error (usually launch failure), but current runner incorrectly ignores pi's exit through its awk pipe and can report `empty_worktree` or even `ok` from preexisting dirt; do not rely on pi rc 14. |
| P2 policy path | **VERIFIED.** `rg -n 'getAgentDir|projectConfigPath|globalConfigPath|DEFAULT_CONFIG' tools/pi-extensions/sandbox/index.ts` shows `<agentDir>/extensions/sandbox.json` or `<cwd>/.pi/sandbox.json`, then a default `allowWrite` of only `.` and `/tmp`. `ls` found `~/.pi/extensions/sandbox.json` but no `~/.pi/agent/extensions/sandbox.json`. The installed pi 0.85.1 `getAgentDir()` defaults to `~/.pi/agent` (`/opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent/dist/config.js`). | Add an explicit `PI_SANDBOX_POLICY_FILE` input to the extension, with mandatory valid JSON for mission use, and have the runner generate a policy in a private `mktemp` directory from tracked `sandbox.mission.json`. Do not write `<WT>/.pi/sandbox.json`: it dirties the worktree, can create a false `ok` porcelain verdict, and may overwrite a real project config. The rig-local policy is not an authority. Missing/malformed/disabled policy gets `sandbox_policy_invalid`/rc 16 before pi starts. |
| P3 linked-worktree commits | **VERIFIED.** `git rev-parse --git-dir` here returns `/Users/voightkampff/dev/sunholo-data/ailang/.git/worktrees/-planner-wt-iter3`; `git rev-parse --git-common-dir` returns the main `.git`, both outside this worktree. `tools/launchd/test_mission_pi_run_commits.sh` is absent at this base (unmerged PR #1329). | Seatbelt would block executor `git commit` with the current policy. Add the resolved worktree gitdir and common-dir `objects` to generated `allowWrite`; require a detached worktree for this narrow policy, or fail policy preflight for a branch-attached worktree until its exact ref/log paths are deliberately handled. Do not add the whole main checkout or whole common `.git`. After #1329 lands, the clean worktree plus commit must still count as work; add/rebase a compatible test arm. |
| P4 temp directory | **VERIFIED.** `tools/pi-extensions/README.md` and `gate-3-route.md` document sandbox-runtime's pinned `TMPDIR=/tmp/claude` and the `go build` failure if it is absent; `ls -ld /tmp/claude` shows it exists on this rig, which does not make its existence portable. | `mkdir -p /tmp/claude` before launch; if creation fails, emit `sandbox_unavailable`/rc 15. Keep runtime temp and generated policy outside the worktree. |
| P5 opt-out | **VERIFIED for the searched callers.** `rg -n 'mission_pi_run\.sh|pi --mode json|--no-sandbox' tools/launchd .claude/skills scripts make/test.mk` found the Gate-3 executor recipe and evaluator handshake requiring this runner; direct `pi --no-tools` calls are probes, not runner callers. No caller requires an unfenced mission run. | No opt-out. Every emitted verdict has `"fenced": true` only after readiness is proven; preflight failures have `"fenced": false`. Do not allow `--no-sandbox` or an ambient config with `enabled:false` on this lane. |

`sandbox/index.ts` also currently catches `SandboxManager.initialize()` failure and continues with `localBash.execute`; `user_bash` likewise returns without sandbox operations. This is a **fail-open** path even when both `-e` arguments load. The plan must remove it for mission use. The worktree-fence extension covers only `write`/`edit`, not shell commands.

## Milestones

## Execution note (iteration 5)

The runner stages `index.ts`, its local `mission.ts` policy helper, `package.json`, and
`worktree-fence.ts` from the runner's own versioned checkout in a private run directory.
The staged sandbox symlinks runtime `node_modules` from the runner checkout first,
then the main checkout resolved through the worktree's common gitdir. This avoids
requiring source parity with the intentionally drifting main checkout. The generated
policy permits the linked worktree gitdir and the common gitdir's `objects` only.
(The executor's first draft also allowed the common `refs/heads` and `logs`; the controller
removed them, because those directories hold every branch of the main checkout, `dev`
included. A branch-attached worktree therefore cannot commit under the fence; its work stays
uncommitted, porcelain counts it, and the controller commits — the codex lane's contract.)
The controller also fixed `index.ts`'s disabled check to `config.enabled === false`: the
mission policy carries no `enabled` key, and `!config.enabled` disabled the sandbox on every
real mission run, which would have read `sandbox_not_ready` (rc 17) forever.
Iteration 5 round 2 adds `sandbox/index.test.ts` to exercise the registered bash, user_bash, and session_start handlers against stubbed pi and sandbox runtime packages.

### M1 — NEEDS-MARK: explicit policy and fail-closed extension (~45 LOC)

**Files:** `tools/pi-extensions/sandbox/index.ts` and its focused test. This path is **outside the fleet push allowlist**. Mark must approve or route this milestone separately; do not push it as fleet work. M2/M3 cannot claim containment before it lands.

1. Add `PI_SANDBOX_POLICY_FILE` as an explicit, absolute-path policy source. If set, require the file to exist, parse, have `enabled !== false`, and include valid `filesystem`/`network` settings; reject invalid input instead of merging defaults. Keep existing interactive behavior when the env var is absent, but the mission runner always sets it.
2. Add `PI_SANDBOX_READY_FILE` for the mission handshake. Create its parent in the runner's private temp directory; write a ready marker **only after** `SandboxManager.initialize` succeeds. On failure, leave it absent and make both `bash` and `user_bash` reject commands, never call `localBash.execute`. A `session_start` error or unsupported platform must not enable bash. Use a unique per-run path, not a predictable global sentinel.
3. Test valid/missing/malformed/disabled policy and failed initialization through the extension's registered handlers. A failed-init bash invocation must be blocked. No model call.

**Acceptance:** focused extension tests demonstrate failed initialization cannot execute ordinary bash; validated policy and readiness are observable; no default-policy fallback in mission mode. Only this milestone needs the separate allowlist decision.

### M2 — runner wiring and typed verdicts (~120 LOC)

**File:** `scripts/mission_pi_run.sh`. Depends on M1 being present in the extension source selected at launch.

1. Resolve the current worktree's common gitdir with `git -C "$WORKDIR" rev-parse --path-format=absolute --git-common-dir` where supported, with a Bash 3.2-compatible `cd`/`pwd -P` fallback. Derive main checkout as the common gitdir's parent; never hardcode `~/dev/...`. Verify the main checkout is real, the two extension files and canonical policy match the current checkout, and the sandbox module exists beside the main-checkout `index.ts`. Pass **those main-checkout absolute paths** to `-e` for reliable package resolution. Any mismatch/missing module emits `sandbox_unavailable`/rc 15 before pi launch.
2. Validate tracked `tools/pi-extensions/sandbox/sandbox.mission.json` with `jq -e`. In a private `mktemp -d`, use `jq` to add the canonical absolute worktree gitdir and common `objects` directory to `filesystem.allowWrite`; reject a branch-attached worktree until its exact ref/log writes are modeled. Validate the generated policy and set `PI_SANDBOX_POLICY_FILE` to its absolute path. No policy or temp file is placed inside `$WORKDIR` or `$HOME`. Policy errors emit `sandbox_policy_invalid`/rc 16.
3. Create `/tmp/claude`, set `PI_FENCE_ROOT="$WORKDIR"`, set `PI_SANDBOX_READY_FILE` to a private path, then invoke `pi --mode json --no-session -e "$MAIN/tools/pi-extensions/sandbox/index.ts" -e "$MAIN/tools/pi-extensions/worktree-fence.ts" --model "$MODEL"`. Preserve existing message filtering, stall bounds, and messaging env. Propagate **pi's exit status**, not awk's pipe status; a nonzero pi exit is `launch_failed`/rc 14 unless a more specific sandbox failure is established. If readiness never appears, emit `sandbox_not_ready`/rc 17 even if pi exits zero or the worktree is dirty. Clean the private temp directory after verdict construction. A narrowly named test override may redirect the `/tmp/claude` creation check into the test's `mktemp` root; production must always use `/tmp/claude`.
4. Add `fenced` to every verdict. Only successful readiness may set it true. Preflight failures must still write JSON to `--verdict`, with clear `error` text, so callers do not mistake an absent verdict for an empty worktree.

**Acceptance:** both absolute `-e` flags, `PI_FENCE_ROOT`, generated policy path, readiness path, and repo/worktree cwd are visible to a fake pi; missing dependency/policy and missing readiness cannot report `ok`; existing rc 10–13 semantics remain unchanged for a ready fenced run. `bash -n` succeeds on macOS Bash 3.2.

### M3 — regression arms and integration gate (~185 test LOC)

**Files:** new `tools/launchd/test_mission_pi_run_sandbox.sh`, `make/test.mk`; reconcile `tools/launchd/test_mission_pi_run_commits.sh` when #1329 is merged. The test suite creates a fake main checkout and linked worktree entirely under `mktemp -d`, with a fake sandbox `node_modules` package and fake `pi` on `PATH`. It records argv, selected env, cwd, and whether it was called. No real model, `$HOME` write, or `~/.ailang/state` write.

| Arm | Expected observation | Mutation it kills |
|---|---|---|
| happy, clean worktree changed by fake pi | exact two absolute `-e` paths from derived main checkout; correct cwd and `PI_FENCE_ROOT`; policy is outside worktree, includes caches, gitdir and common objects; readiness -> `ok`, `fenced:true` | drop either `-e`, use pin-worktree extension path, omit fence root, omit policy env |
| main checkout dependency removed | pi never invoked; rc 15 `sandbox_unavailable`, `fenced:false` | skip dependency preflight or fall back to unfenced pi |
| canonical policy missing or malformed | pi never invoked; rc 16 `sandbox_policy_invalid`, `fenced:false` | use DEFAULT_CONFIG or rig-local policy |
| disabled policy | pi never invoked; rc 16 | honor `enabled:false` on mission lane |
| fake pi never writes readiness | rc 17 `sandbox_not_ready`, even if pi exits zero and changes a file | trust `-e` flags or porcelain alone |
| fake pi exits nonzero after readiness | rc 14 `launch_failed` | lose pi exit through awk pipeline |
| linked-worktree commit | fake pi makes a real local commit, leaving porcelain clean; after PR #1329, verdict `ok` and generated policy includes required git metadata paths | only count porcelain; omit linked gitdir or common objects |
| temp bootstrap | fake pi observes the test override path under `mktemp`; injected creation failure yields rc 15 | omit temp bootstrap or ignore creation failure |

Each arm restores only paths under its own `mktemp` root. Do not create, delete, or rename the rig's `/tmp/claude` from tests; use the test-only override for this arm. Add the new suite to `make test-launchd-drivers`; run `scripts/test_mission_pi_run.sh` as a regression suite and the new suite with `/bin/bash`.

**Acceptance:** all named arms pass, `make test-launchd-drivers` passes, `git diff --check` passes, and the implementation diff contains only allowed runner/test/Makefile paths **plus a separately approved M1 extension diff**. The controller checks the #1329 merge state before execution and folds its commit-count test into the runner gate. The runner's own existing tests may need test fixture updates because they currently omit the required extension/module readiness setup; confine any update to the test path allowed by fleet routing or seek explicit routing if `scripts/test_mission_pi_run.sh` is outside the specific push allowance.

## Optional attended live smoke (controller only; not CI)

After M1–M3 and dependency installation in the main checkout, from the repository root:

```bash
REPO=$(pwd -P)
WT=$(mktemp -d /tmp/pi-fence-smoke.XXXXXX)
git worktree add --detach "$WT" HEAD
PROBE="fence_probe_$(date +%s)_$$"
mkdir -p /tmp/claude
cd "$WT" && PI_FENCE_ROOT="$WT" \
  PI_SANDBOX_POLICY_FILE="$REPO/tools/pi-extensions/sandbox/sandbox.mission.json" \
  pi --mode json --no-session \
    -e "$REPO/tools/pi-extensions/sandbox/index.ts" \
    -e "$REPO/tools/pi-extensions/worktree-fence.ts" \
    --model openrouter/deepseek/deepseek-v4-flash-0731 \
    -p "Use bash twice. First run: touch \$HOME/$PROBE. Report the exact error. Second run: go build ./cmd/ailang. Report whether it succeeds." \
    > /tmp/pi-fence-smoke.ndjson 2> /tmp/pi-fence-smoke.stderr
```

Expect the first tool result to contain `Operation not permitted` and no `$HOME/$PROBE`; expect the second to succeed. The direct extension invocation checks the runtime; the runner's fake-pi suite checks the generated gitdir policy and readiness wiring. This smoke uses the canonical tracked policy and a detached scratch worktree. The controller owns scratch worktree cleanup. **In-sandbox gate verdicts are UNINFORMATIVE** for repository acceptance: the controller reruns all required gates outside the sandbox before promotion.

## Risks and boundaries

- The main checkout may have a different extension revision from the detached pin. Source and policy parity checks must refuse such a run rather than load mismatched code. The module dependency is untracked; its absence is a normal, typed lane failure until provisioned.
- Allowing writes to linked git metadata permits commits and object creation, so it is a deliberate expansion beyond file edits in `$WT`. Keep the common-dir allowance to `objects`; branch-attached worktrees fail preflight until separately handled. The controller still inspects the main checkout and worktree before accepting work.
- The preflight detects missing modules and policy, but runtime init can still fail. The ready marker plus fail-closed bash handler cover that gap. A pi version that ignores a thrown extension error must not turn the verdict green.
- **Eval impact:** no direct change to `internal/eval*`, `.pi/extensions`, pi EVAL prompts, or benchmark harness. The mission-control pi **evaluator role** uses this same runner and therefore inherits the containment and possible new lane-failure verdicts. This changes execution safety, not eval scoring or thresholds.
