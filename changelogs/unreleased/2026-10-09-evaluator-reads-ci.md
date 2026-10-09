### Fixed — the sprint evaluator reads the PR's CI instead of re-running `make test`

No cloud evaluation has returned a verdict since 2026-09-29. `make test` is silent for longer
than the executor's 5-minute idle timeout, so every run was killed mid-suite
(`pi idle for 5m0s mid-generation`). `evaluate_sprint.sh` now takes tests and lint from the
PR's CI through the new `scripts/ci_gate.sh <branch>`. Each call waits at most 4 minutes and
reports `pass`, `fail` (naming the failed checks, or "every check skipped" for a PR that
conflicts with its base) or `pending`. The evaluator calls it again until it gets a verdict.
With no PR it runs the gates locally as before (`EVAL_LOCAL_GATES=1` forces that). The JSON
records `gate_source`.

The agent base image now installs `gh` from GitHub's apt repository (bookworm's gh 2.23 predates
`pr checks --json`) and `jq`. `evaluate_sprint.sh` reads the sprint JSON with `jq` under
`set -e`, so in the cloud image it died before running a single gate.
