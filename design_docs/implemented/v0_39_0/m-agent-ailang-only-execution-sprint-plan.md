# Sprint Plan: M-AGENT-AILANG-ONLY-EXECUTION

**Design doc**: [m-agent-ailang-only-execution.md](m-agent-ailang-only-execution.md) · **Created**: 2026-09-16 · **Freeze**: D1–D6 ratified (Mark, 2026-09-16); parent M-PI-HARNESS-UPGRADE complete in dev (`3395d7ccf`, images at 0.85.1, extension probe green in-container)
**Duration**: 4 days (~24h) · **Risk**: medium — the gate and the tool are small; the risk is the resident cutover (a live instance) and the D7 name-mapping change touching every pi caller
**Sprint ID**: `M-AGENT-AILANG-ONLY-EXECUTION`

## Goal

One admission gate (`ailang run --policy`), reached through one pi tool (`ailang_run`/`ailang_check`), selected by one registry field (`tool_policy`) — so an agent on `ailang_only` reads and writes files normally and executes programs **only** by submitting AILANG source to a policy it cannot edit. Banked on every row; the resident flips by default; the fleet stays `full`.

## Pre-sprint corrections to the design doc (measured 2026-09-16)

- **V14 is refuted, in the helpful direction.** A clean-row entry that calls an imported `! {Net}` function is rejected by the typechecker even under `DryLink` (`policy-check` → `typecheck_failed`, `ailang check` rc=1 "Missing effects: Net"). Transitive import effects are enforced **by construction**; M2.2 "closure in `Check`" is not work. The test in M2 pins that property instead.
- **Policy `[budgets]` have no run-time hook.** Budgets are enforced from source annotations (`internal/effects/budget_frame.go`), not from a CLI/policy input. `run --policy` therefore **refuses** a policy that declares budgets rather than silently not enforcing them (CLAUDE.md §2). Enforcing policy budgets is a follow-up, filed in the doc's Future Work.
- `main_run.go`'s `runFile` takes ~60 positional args. `--policy` is resolved **before** the call into the existing flag values (`caps`, `net-allow-domains`, FS sandbox via `os.Setenv(config.EnvFSSandbox)` — `os.Setenv` is not forbidigo-forbidden and is how `check.go` already sets task env) rather than threading a 61st parameter.

## Milestones

### M1 — Bank the policy before changing it (~2h, ~70 LOC) ✅ 2026-09-16
**Files**: `internal/executor/executor.go`, `internal/executor/pi/pi.go`, `internal/eval_harness/agent_runner.go`, `agent_runner_multi.go`, `metrics.go`, `cmd/ailang/eval_benchmark_agent.go`, tests
1. `Result.ToolPolicy []string` (the **effective** list passed to the CLI, or the sentinel `["<pi default>"]` when `AllowedTools == nil`) and `Result.PolicyDigest string` (sha256 of the policy file, when one was passed).
2. Plumb → `AgentBenchmarkResult` → `RunMetrics.tool_policy` / `policy_digest`, `omitempty`, absent ⇒ unmeasured (same voice as `executor_version`).
3. `PiExecutor` stamps both on every result shape.
**Accept**: nil `AllowedTools` banks the sentinel, not `[]`; an explicit list banks verbatim; empty digest omitted.

### M2 — One gate: `ailang run --policy` (~5h, ~180 LOC) ✅ 2026-09-16
**Files**: `cmd/ailang/main_run.go`, new `cmd/ailang/run_policy.go`, `cmd/ailang/policy_check.go` (extract the admission path into a shared `admitProgram(policyPath, file) (decision, exit)`), `internal/policy/policy.go`, tests, `examples/safety/`
1. `--policy <toml>` on `run`: load policy; admit via the shared path (size cap → typecheck → entry export → `CheckScheme`); on denial print the `Decision` JSON to stdout and exit 2 — **no execution**.
2. On admission derive from the policy, overriding whatever the caller passed: `caps` = `allowed_caps` joined; `net-allow-domains` = `net_allow`; `AILANG_FS_SANDBOX` = `fs_sandbox` (set in-process, and `sandbox-check`-style refuse an empty value when the policy admits `FS`).
3. Refuse `--policy` combined with `--caps`, `--no-budgets`, `--allow-env`, or a policy carrying `[budgets]` (unenforceable today) — each a one-line named error, exit 1.
4. `--json` with `--policy` wraps the run result as `{admitted:true, decision:{…}, policy_digest, exit_code, stdout…}` — the M-AGENT-SAFE-RUNNER `program-result` shape.
5. `policy.ErrorKind` → executor `error_category`: `policy_violation` → `ErrorCategoryPolicyViolation` (`"policy_violation"`), `typecheck_failed` → existing `compile_error`; both new categories verified unallocated by grep.
6. Test that the caps table cannot drift: the policy package derives its known-caps list from `runner.CapsList` (one table).
7. Tests pin: a lying entry (clean row, imported `Net`) → `typecheck_failed` **through `run --policy`**; `allowed_caps = []` refuses a `! {IO}` program; `--policy` + `--caps` exits 1 naming the conflict; an admitted `! {IO}` program runs and its stdout appears in the JSON.
**Accept**: all seven tests red→green; `examples/safety/README` names the command.

### M3 — One tool: `ailang-exec.ts` (~4h, ~150 LOC TS + probe) ✅ (+ carried extensions, found on the rig) 2026-09-16
**Files**: `.pi/extensions/ailang-exec.ts` (+ `make pi-assets` → `cmd/ailang/pi_assets/`), `.pi/extensions/README.md`, `cmd/ailang/pi_setup.go` (`tool-profile` subcommand), `docker/test-agent-pi.sh`
1. Registers `ailang_run {file, args_json?}` → spawns `ailang run --policy $AILANG_AGENT_POLICY --json <file>`, returns the JSON verbatim as the tool result (text + details); `ailang_check {file}` → `ailang check --json <file>`.
2. **Default-deny at load**: if `AILANG_AGENT_POLICY` is unset or unreadable, the extension registers `ailang_run` as a tool that **refuses with a named reason** (so the model sees why, rather than a missing tool it reaches around with bash); it also refuses at load if the policy's `fs_sandbox` contains the policy file's directory (D4).
3. `ailang pi tool-profile <name>` prints the flag expansion (`ailang_only` → `--no-builtin-tools --tools read,edit,write,ailang_check,ailang_run`; `full` → nothing) so `pi.go` and the resident's `pi.mjs` read one string — the same seam discipline as the pi pin.
4. `docker/test-agent-pi.sh` §6: the tool is registered (`pi --list-tools`? — verify the flag exists; else a load-time sentinel that asserts `ailang_run` is in the registered set via the extension API) and, with no policy env, refuses with the named reason.
**Accept**: probe in-container green; a bash-free directive on the rig (`--no-builtin-tools --tools read,write,ailang_run`, real model) produces `tool_execution_start ailang_run` with `admitted:true`.

### M4 — One field: `tool_policy` (~4h, ~140 LOC) ✅ 2026-09-16
**Files**: `internal/coordinator/agent_registry.go`, `provider_executor.go`, `cmd/ailang/coordinator_agents_list.go`, new `internal/executor/pi/toolnames.go`, `args.go`, tests
1. `AgentConfig.ToolPolicy string` (`yaml:"tool_policy"`): `full` | `ailang_only` | comma list; default `full` (D6). Resolved into `Task.AllowedTools` (canonical names) next to the question-kind block; `question` becomes canonical `["Read","Grep","Glob","WebFetch","WebSearch"]` (unchanged names, now mapped).
2. D7: `internal/executor/pi/toolnames.go` — canonical→pi table (`Read→read`, `Write→write`, `Edit→edit`, `Bash→bash`, `Grep`/`Glob`/`WebFetch`/`WebSearch` → **unmapped** (pi has no such builtins), `AilangRun→ailang_run`, `AilangCheck→ailang_check`); `buildPiArgs` errors on an unmapped name, naming it. Profile `ailang_only` expands via the same string `tool-profile` prints.
3. `ailang coordinator agents <id>` shows `tool_policy` declared vs effective.
4. Negative test at the executor boundary: `ailang_only` args contain `--no-builtin-tools` and no `bash`; the question-kind list on pi now **errors** (it names Grep/Glob/WebFetch/WebSearch) instead of silently yielding zero tools — and the coordinator's question path is changed to a pi-valid list (`Read` + `AilangCheck`) so the error is a guard, not a regression.
**Accept**: unmapped name errors; `ailang_only` never emits `bash`; agents list shows the field.

### M5 — Resident cutover, eval lane, boundary (~5h) ✅ code + A/B; live resident verify PENDING on a Cloud Run Preview platform error 2026-09-16
**Files**: `docker/resident/lib/pi.mjs`, `docker/resident/Dockerfile`, `docker/resident/test-image.sh`, delete `docker/resident/resident-run` + `allowlist.example.json`, `ailang-multivac/scripts/resident-instance.sh`, `cmd/ailang/eval_suite*.go` (`--tool-policy`), `docs/docs/guides/agent-tool-policy.md`, runbook, charter, memory
1. `pi.mjs`: `RESIDENT_TOOLS` default `ailang_only`, expanded via `ailang pi tool-profile`; policy path from `AILANG_AGENT_POLICY`; `resident-run` deleted; `test-image.sh` §4 targets `ailang run --policy` (default-deny, caps subset, `--policy`+`--caps` refused).
2. `resident-instance.sh update --policy-file <toml>` mounts it read-only at `/etc/resident/policy.toml` and sets `AILANG_AGENT_POLICY`; redeploy `resident-pi-ailang` (dev); `verify` gains "a bash request is refused; an admitted `.ail` runs".
3. `eval-suite --tool-policy ailang_only` → `Task.AllowedTools` for the pi lane; banked per M1. Boundary note in runbook/charter/memory **before** the first banked row; core-tier comparator on the rig (`--models pi-qwen3-8-27b`, explicit) reported with `eval-paired`.
**Accept**: resident verify green on the live instance; first `ailang_only` rows carry `tool_policy`; comparator report committed.

## Day plan
| Day | Work |
|---|---|
| 1 | M1 · M2 |
| 2 | M3 · M4.1–4.2 |
| 3 | M4.3–4.4 · M5.1–5.2 (resident) |
| 4 | M5.3 (eval lane + comparator) · docs · changelog · evaluator handoff |

## Risks
| Risk | Mitigation |
|---|---|
| Models on `ailang_only` reach for bash reflexively and fail more | That is the measurement (D5); the lane's prompt names its tools |
| `--no-builtin-tools` + `--tools` interplay differs from the help text | M3 probes it live on 0.85.1 before M4 relies on it |
| Resident redeploy breaks a live instance | dev estate only; `verify` before and after; the instance is stopped (idle sweep) so nothing is mid-conversation |
| Threading a 61st arg through `runFile` | Resolved before the call into existing flags instead |

## Outcome (2026-09-16)

- Commits on dev: `aaef58683` M1 · `6236deb24` M2 · `db0df13aa` M3 · `bcfef8b7d` M4 · `a62287b36` `e02fdf8a2` M5 · fixes `377ac98a8` (pi.mjs shadowing, caught by test-resident-pi) · `0b9e89019` (profile carries its extensions, caught on the rig) · `a6996f9c4` · `173564f20` (failed rows keep provenance, caught by the A/B). multivac: `ef4403b`, `f7e2689`.
- Cloud Build `dfb980b0`: `test-agent-pi` 20/20, `test-resident-pi` 130/0.
- Live on the rig: bash-free run → `write`, `ailang_check`, `ailang_run` admitted (`42`); `! {Net}` → `policy_violation`; no policy → named refusal; model-visible tools exactly `read, edit, write, ailang_check, ailang_run`.
- A/B (OpenRouter minimax-m3, 5 benchmarks): bash 5/5, `ailang_only` 4/5, the miss a model stop; `docs/sprint-retros/m-agent-ailang-only-execution-ab-2026-09-16.md`.

**Residuals:** dev resident updated with the policy but not started — the Cloud Run instances Preview API answered "internal error" for 12 min, for the untouched control instance too (platform); live `verify` pending. Rig 27B comparator deferred (GPU contention; rig lock not exclusive — memory). Prod images roll on the next release.
