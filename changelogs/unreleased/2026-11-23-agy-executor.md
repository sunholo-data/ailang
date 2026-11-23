### Added — `agy` executor: Google's Antigravity CLI as a local-editing Google lane

`internal/executor/agy` runs Google's Antigravity CLI (`agy`) as an executor, so Gemini 4 can edit the
local workspace the way `claude` and `codex` do. It is the lane for the Google AI Pro/Ultra
subscription (design: `design_docs/planned/m-agy-executor.md`).

- **Two auth routes, picked by the model string.** A bare model id (`gemini-4-neon-high`) uses the
  subscription and banks cost as `list-price-equivalent`. An `api/` prefix (`api/gemini-4-neon-high`)
  uses the locked `GEMINI_AGENTS_API_KEY` and banks as `metered`. On the API-key route the child
  receives only `GEMINI_API_KEY`. Each route runs under its own isolated HOME
  (`AILANG_AGY_HOME_ROOT`, default `~/.ailang/state/agy`), never the operator's `~/.gemini`.
- **Exit 0 is not trusted.** agy exits 0 with `status:"SUCCESS"` when it soft-denies a tool. The
  executor banks that as a failure naming the denied action. A missing sign-in becomes an error that
  points at `make agy-signin`. No result event banks as `wire_drift`, and a hang is killed with its
  process group at `Task.Timeout`.
- **Pinned and quiet.** `make install-agy` installs agy 1.5.15, verified against its SHA512, and
  leaves shell rc files untouched. Every invocation sets `AGY_CLI_DISABLE_AUTO_UPDATE=true`, because
  plain runs were self-updating the CLI. `HealthCheck` refuses any other version.
- **Sign-in and rows.** `make agy-signin` signs the subscription route in, attended and once per
  machine. `make agy-check` verifies it. New opt-in rows `agy-gemini-4-neon`, `agy-gemini-4-carbon`
  and `agy-api-gemini-4-neon` are in no suite.
- **Status.** Live-verified on the API-key route: `ailang exec agy --model api/gemini-4-neon-high`
  edited the workspace, priced at $0.043. The subscription route waits on the sign-in.

### Added — `agy:` mission role lane (wired, in no chain)

The mission driver now accepts `agy:<model>` role pins:
- **Buckets:** `agy:gemini-4-neon-high` is bucket `google` (subscription), and
  `agy:api/gemini-4-neon-high` is `gemini-api` (metered). All three bucket maps (driver, resolver,
  spawn-pin hook) agree, and R13 checks it. Before this, an `agy:` pin fell through to the
  **anthropic** bucket.
- **Probe:** `_mc_probe_agy` goes through `ailang exec agy --json` inside a new `agy-role` secrets
  scope, which keeps only `GEMINI_AGENTS_API_KEY`. The verdict is the executor's `success` field, not
  the exit code.
- **Role pre-flight:** walks `agy:` pins with the pi loop's advance-along-the-chain contract. The
  Anthropic pre-flight no longer probes them as Claude models.
- **Controller:** controller selection skips `agy:` entries by name.

Covered by `tools/launchd/test_agy_lane.sh`, which is wired into `make test-launchd-drivers` and
mutation-checked. No mission's chain names agy yet, so it changes no fire until a pin is set.

### Added — `google` quota bucket: the agy subscription is rationed at 15% a day

`ailang mission quota` reads the Google AI subscription through `agy -p /quota` under the agy
executor's subscription HOME. That command makes no model call. Only the Gemini weekly window is
paced: `AILANG_GOOGLE_DAILY_PACE` percent (default 15) per calendar day since the weekly reset,
capped at 100, with a 2pp start margin. The five-hour window blocks only when exhausted. The
Claude/GPT rows are reported and not paced.

An unreadable quota fails closed, as for Codex and Ollama: agy missing, not signed in, a bad
pace value or unparseable output all block. `--over` prints `google`, and the mission driver blocks
it. A failed quota command now blocks `codex ollama google`, so an `agy:` subscription rung is
refused unprobed when over. The metered `agy:api/` route has its own bucket (`gemini-api`) and is
not paced by this.

### Changed — agy runs in its terminal sandbox with all commands allowed (D3 option B)

agy now runs with `--sandbox`, and its rules allow every command (`command(*)`). The sandbox is the
boundary, not a command allowlist.

- **Allowed and denied:** git-write commands, `rm -rf`, `sudo`, `curl` and `wget` stay denied.
  Writes outside the workspace are allowed only through absolute `write_file()` grants, for the Go
  build cache, the Go module cache and a fixed per-route `TMPDIR`.
- **Measured:** writes outside the workspace and reads of `~/.config` are refused. Go builds and
  `ailang run` work, and the shell starts in the workspace.
- **Two traps avoided:**
  - A relative path rule silently breaks the sandbox: every command fails while agy reports
    SUCCESS. Such runs are now failed by the executor, and the rules carry absolute paths only.
  - `--add-dir` moves the shell's starting directory away from the workspace, so it is not used.
- **Why:** the narrow allowlist voided 2 of 23 agent smoke runs on the model's own self-checks.

### Changed — Gemini 4 Neon (agy) joins the stapledon mission's executor chain

`agy:gemini-4-neon-high` heads stapledon's executor fallback, ahead of the deepseek rungs. It
fires when codex and the Anthropic rung are unavailable, and the `google` ration gates it.

The agent smoke gate is 23/23 on the subscription.
