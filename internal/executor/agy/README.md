# agy executor — Google Antigravity CLI

The Google lane that edits the local workspace. Design:
[`design_docs/planned/m-agy-executor.md`](../../../design_docs/planned/m-agy-executor.md).

## Install and sign in

```bash
make install-agy   # pinned 1.5.15, SHA512-verified, no shell rc edits (tools/agy/install.sh)
make agy-signin    # attended, once per machine: signs the subscription route in
make agy-check     # non-interactive: pin + sign-in + a 1-reply probe on gemini-4-neon-low
```

`HealthCheck` refuses any agy other than `ExpectedVersion`. To bump: update the three `AGY_*`
values in `tools/agy/install.sh` from the manifest, re-capture `testdata/`, then bump
`ExpectedVersion`.

## Auth routes

| `agent_model_name` | Route | HOME | Credential | Cost provenance |
|---|---|---|---|---|
| `gemini-4-neon-high` | subscription (Google AI Pro/Ultra) | `$AILANG_AGY_HOME_ROOT/subscription` | agy's cached sign-in | `list-price-equivalent` |
| `api/gemini-4-neon-high` | API key | `$AILANG_AGY_HOME_ROOT/apikey` | `GEMINI_API_KEY` = `GEMINI_AGENTS_API_KEY`, child only | `metered` |

`AILANG_AGY_HOME_ROOT` defaults to `~/.ailang/state/agy`. The operator's real HOME is never used:
its `~/.gemini` settings can carry a `gcp.project`, which turns a "subscription" run into metered
Vertex. On the subscription route the executor strips `gcp` and `modelProvider` from settings.

Because HOME moves, the child gets `GOPATH`, `GOCACHE` (and `GOMODCACHE` if set) and
`GIT_CONFIG_GLOBAL` pointing at the operator's own, so builds reuse the module cache.

**Subscription sign-in:** a personal Google account (Workspace accounts get no plan quota in agy).
One sign-in per holder: laptop, rig and each cloud environment sign in separately. Never copy the
cached token between machines; a shared refresh token is what killed the Claude loops.

## Flags

`--model <m> --mode accept-edits --output-format stream-json --disable-slash-commands
--print-timeout <Task.Timeout-15s> -p <directive>`, with `AGY_CLI_DISABLE_AUTO_UPDATE=true`. Any agy
invocation otherwise may self-update in the background (measured: 1.3.5 → 1.5.15 during probes).

## Permission posture (D3, proposed)

`accept-edits` lets workspace edits through. Shell commands need an allow rule, written into the route
HOME's `settings.json` (`route.go`: `defaultAllowCommands`, `defaultDeny`). Rules match command
prefixes word by word, and deny beats allow. `--dangerously-skip-permissions` is never passed.

In `-p` mode, an action with no rule is **soft-denied**: exit 0, `status:"SUCCESS"`, listed in
`result.denied_actions`. The executor banks that as a **failure** naming the denied action.

The workspace path is resolved through symlinks, and `PWD` is set to it. Otherwise a model that writes
to the `/var/...` spelling of a `/private/var/...` cwd is soft-denied as "outside the workspace".

## Stream schema (agy 1.5.15)

See `events.go` and `testdata/` (all real runs, 2026-11-23, paths scrubbed). The result line's
`usage` is authoritative. `input_tokens` is fresh input, disjoint from `cache_read_tokens`, and
`thinking_tokens` is separate from `output_tokens`.

| Fixture | What it pins |
|---|---|
| `edit` | clean edit: one `write_file` step (ACTIVE+DONE = one call), usage totals |
| `shell_denied` | soft denial → failure |
| `shell_git_allowed` | `command(git)` allow rule lets `git status` run |
| `unknown_model` | exit 1, status ERROR, available-model list |
| `no_auth` | exit 1, "authentication failed or timed out" → `make agy-signin` hint |

## Tests

`go test ./internal/executor/agy/` replays the fixtures through a POSIX mock agy and also covers:
- process-group kill at `Task.Timeout`
- a missing result event, banked as `wire_drift`
- print-timeout exit 3, banked as `timeout`
- credential isolation per route
- settings merge
- the version pin

Live run: `AILANG_AGY_LIVE=1 [AILANG_AGY_LIVE_MODEL=gemini-4-neon-high] go test ./internal/executor/agy/ -run TestLive -v`.

## Not here yet

- **Cloud (Pillar 2):** on Linux agy wants the Secret Service keyring over D-Bus. Whether its file
  fallback alone authenticates a fresh container is unmeasured, so the image recipe waits on that.
- **Quota observability (P4):** the subscription bucket is account-wide, refreshed every 5 hours with
  weekly caps, and not visible to the mission ration gate yet.
