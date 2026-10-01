### Security — one ordered child-env builder; task env and clone URLs are validated (M-EXECUTOR-ENV-HARDENING PR-A)

Phases 1 and 3 of [M-EXECUTOR-ENV-HARDENING](../../design_docs/planned/v0_49_1/m-executor-env-hardening.md)
(the 2026-10-01 executor audit's H-4; D3/D4/D5 approved by Mark 2026-10-01). The secret-stripping
profiles (D1/D2) follow in PR-B; this change does not yet narrow what the child inherits.

**One builder, fixed precedence (D4).** `BuildEnvironment` now constructs the agent child's
environment through a de-duplicating builder in four layers — inherited → harness-injected →
`Task.ExtraEnv` → the executor's required set — so every key appears once and the precedence is the
documented `ExtraEnv > injected > inherited`. Before, `ExtraEnv` was appended *before* the harness
keys, so on a collision the harness silently won (the reverse of the comment), and the result only
worked because Go's `os/exec` keeps the last duplicate. motoko's ten executor-specific variables now
go through the same builder (`EnvironmentOptions.ExecutorEnv`) instead of being appended after it.
`OTEL_RESOURCE_ATTRIBUTES` is now sorted, so an identical task builds an identical env.

**`ExtraEnv` deny-list (D3).** A task — notably a benchmark's `agent_env`, which is a data file —
may no longer set loader, shell start-up, proxy, git, telemetry, program-policy or harness-owned
names (`LD_*`, `DYLD_*`, `PATH`, `HOME`, `SHELL`, `BASH_ENV`, `NODE_OPTIONS`, `GIT_*`, `OTEL_*`,
`HTTP(S)_PROXY`, `AILANG_AGENT_POLICY*`, `TRACEPARENT`, the correlation IDs, …). A refused name is a
loud error naming the variable, raised at spec load (`LoadSpec`), in `ValidateTaskCapabilities`,
and in `BuildEnvironment` itself — the last because the coordinator, cloud job, mission dispatch and
eval agent paths never called `ValidateTaskCapabilities`. Every name the live corpus and dispatchers
set today (`MOTOKO_AST_*`, `AILANG_MISSION_STAGE`, the messaging pins, `PI_WORKSPACE_TRUST_REMOTES`,
the browser lane's CDP endpoint) still passes.

**`AILANG_AGENT_POLICY` is harness-injected from `Task.PolicyPath`.** The three dispatchers that
put it in `ExtraEnv` now set only `PolicyPath`. This also fixes a live bug: the eval agent runner
*replaced* `ExtraEnv` when a benchmark carried `agent_env`, silently dropping the policy for any
`ailang_only` run of such a benchmark.

**Clone URL validation (D5).** `--clone-repo` is interpolated into shell text the agent is told to
run exactly; `ValidateCloneFlags` and `BuildClonePreamble` now accept only a plain
`https://host/path` URL (no userinfo, query, fragment, whitespace or shell metacharacters) and name
the rule otherwise. Both legitimate modes (HEAD clone, fetch-by-SHA) render unchanged.

`BuildEnvironment` now returns `([]string, error)`.
(`internal/executor/envbuild.go`, `extraenv.go`, `environment.go`, `clone_preamble.go`, the five
executors, `internal/eval_harness/spec.go`, `agent_runner_multi.go`, `internal/coordinator/provider_executor.go`,
`cmd/ailang/coordinator_cloud_executor.go`)
