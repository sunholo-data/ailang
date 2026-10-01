### Security — agent children no longer see the container's secrets (M-EXECUTOR-ENV-HARDENING PR-B)

Phases 2 and 4 of [M-EXECUTOR-ENV-HARDENING](../../design_docs/planned/v0_49_1/m-executor-env-hardening.md)
— the 2026-10-01 executor audit's H-4, D1/D2/D6 approved by Mark 2026-10-01. Before this, every
model-facing CLI (claude, codex, pi, opencode, motoko) inherited the host's complete environment:
in a cloud executor that is the fleet `GITHUB_TOKEN`, the superuser `AILANG_REGISTRY_API_KEY` and
every provider key the lane holds, one `printenv` from a prompt injection.

**Default-deny child env (D1).** `BuildEnvironment` now passes a host variable to the child only
when its name is on an allowlist (process basics, locale, toolchains, the harness's own
`AILANG_*`/`OTEL_*`/`CLAUDE_*`/`PI_*`/`MOTOKO_*`/`MISSION_*` configuration, …) **and** it is not
credential-shaped (`*_KEY`, `*_TOKEN`, `*SECRET*`, `*PASSWORD*`, `*CREDENTIALS*`, `*WEBHOOK*`,
`SSH_AUTH_SOCK`, `AWS_*`, …). Credentials pass only as a named grant:

| Executor | Granted |
|---|---|
| claude | none — OAuth via the credentials file; `ANTHROPIC_API_KEY` only under `AILANG_AUTH_MODE=apikey` |
| codex | `OPENAI_API_KEY`, `CODEX_API_KEY` (cloud codex uses the auth.json the parent installs) |
| pi, opencode | the key of the model's `provider/` prefix (`openrouter/`→`OPENROUTER_API_KEY`, `google/`→`GEMINI_API_KEY`+`GOOGLE_API_KEY`, `ollama/`→`OLLAMA_API_KEY`, …) |
| motoko | `OPENROUTER_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY` |
| any, with a program policy | the keys that policy's restricted worker may select (pinned `ai_provider`, std/web's `OLLAMA_API_KEY`) — one list now shared with `ailang run --policy`'s `workerEnv` |

No executor is granted `GITHUB_TOKEN`, `GH_TOKEN`, `AILANG_REGISTRY_API_KEY`,
`CLAUDE_CODE_OAUTH_TOKEN`, `AILANG_KMS_KEY` or `AILANG_CODEX_AUTH_SECRET`. claude's ad-hoc strip of
two variables is retired into this seam. Each build prints the credential *names* it withheld.
`AILANG_EXECUTOR_ENV_INHERIT=NAME,…` (operator env, never a task) forwards extra names — the
rollback lever for a lane that turns out to need one.

**Git (D2).** The cloud parent clones, pushes and opens the PR; the child never needed the fleet
token. Its global credential helper used to embed the token as a literal in `~/.gitconfig`, which
the child shares — so stripping the env alone would have hidden nothing. The helper now reads
`$GITHUB_TOKEN` at call time: the parent's git authenticates as before, the child's gets nothing.
For the child's own optional push/fetch, `AILANG_CHILD_GIT_CREDENTIALS=repo` (default) writes a
0600 credential-store file in a 0700 dir outside the workspace, consulted by git only for the
task's own repository URL and removed when the task ends; `=none` withholds even that.

**Banked (D6).** `Result.EnvNamesDigest` — sha256 of the child env's sorted names, never values —
on every result shape of all five executors, and on eval rows as `env_names_digest`.

**Limit, stated plainly.** The child runs as the parent's UID, so this is defence in depth: a
determined same-user process can still read `/proc/<ppid>/environ` or the credential file. The
boundary for that is the UID split (audit H-6) and the egress lock.
(`internal/executor/envpolicy.go`, `gitcred.go`, `environment.go`, the five executors,
`cmd/ailang/coordinator_cloud.go`, `coordinator_cloud_executor.go`, `run_policy_supervise.go`,
`internal/config/executor.go`, `docs/docs/guides/agent-tool-policy.md`)
