# One account, three Codex OAuth authorizations

Interactive sessions retain `~/.codex`. Local missions use `~/.codex-missions`;
cloud provisioning uses `~/.codex-cloud-staging`. Sign into the same ChatGPT
account and selected workspace in each fresh login. These independent sessions
share the subscription limits; separate directories alone do not isolate a
copied authorization. Never seed either automation home with interactive
`auth.json`.

Before provisioning, stop consumers of the automation home at an idle boundary.
These scripts prepare credentials; they do not activate the mission fleet,
start a daemon, or establish cloud execution ownership. Activate only after
those execution changes and infrastructure checks pass.

Run from the repository:

```bash
bash tools/attended/provision_codex_oauth_profile.sh prepare missions
bash tools/attended/provision_codex_oauth_profile.sh prepare cloud
bash tools/attended/provision_codex_oauth_profile.sh login missions
bash tools/attended/provision_codex_oauth_profile.sh login cloud
bash tools/attended/check_codex_oauth_profiles.sh
# Run only when the cloud rollout is ready to consume this authorization:
bash tools/attended/provision_codex_oauth_profile.sh publish-cloud ailang-multivac ailang-codex-auth-json
```

Each `login` launches the installed Codex OAuth flow; the user completes browser
sign-in. Existing auth files are refused rather than overwritten. Reauthorizing
an expired automation session requires an attended decision to retire that
profile's existing credential first. Interactive state is never modified.
`CODEX_INTERACTIVE_HOME`, `CODEX_MISSION_HOME`, and `CODEX_CLOUD_HOME` override the
three paths explicitly; unset variables use the locations above. Parent
directories must already exist.

`prepare` creates mode 0700 directories and mode 0600 configuration with
`cli_auth_credentials_store = "file"`. For a deliberately audited TOML config,
use `prepare missions --config /path/to/audited-config.toml` (or `cloud`). The
helper refuses an existing target config and replaces only a top-level
credential-store setting in the supplied file. Review MCP commands, trust,
models and environment settings before supplying it. Skills, plugins, rules,
MCP definitions and other resources require deliberate provisioning and review;
the helper does not copy interactive resources or credentials automatically.
An existing config must explicitly select file storage. Login additionally
forces file storage and removes API-key environment variables.

On a headless server, append `--device-auth` to each login command. Open the
displayed device-login URL on another device and enter its one-time code. Finish
the mission login before starting the cloud login; they create separate sessions.

After the three-profile checker passes, install the new AILANG binary and mission
drivers together at an idle boundary. Codex 0.162.0 is the validated daemon version.
Start one mission owner with a minimal environment:

```bash
env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR:-/tmp}" \
  CODEX_HOME="$HOME/.codex-missions" codex app-server daemon start
```

For an independently managed foreground service, use the same environment with
`codex app-server --listen unix://` and a mission-only service label. Do not use
the interactive home for either command. Inherited home/project MCP `env_vars`
resolve against the daemon's startup environment: audit their credential needs
explicitly. Task-defined MCP servers receive their authorized filtered environment;
home/project MCP credentials must not be inherited from an unrestricted shell.
Controllers, probes and executor roles connect to the
home's protected Unix WebSocket socket with `AILANG_CODEX_RUNTIME=daemon`; a missing
owner fails rather than starting a private auth manager. Smoke-test quota and a
small inference through this path before resuming mission dispatch. Stop the owner
only after its mission consumers have stopped; use the same explicit home with
`codex app-server daemon stop`.

Cloud rollout must replace every image that consumes the cloud secret before
publishing the fresh cloud authorization. Old executors do not honor the lease:
stop dispatch, wait for all old executions to terminate, deploy both new Codex
variants, then publish and smoke-test. The Codex image sets an isolated explicit
`CODEX_HOME=/home/ailang/.codex`. Check Firestore transactional access and Secret
Manager access/add-version permissions in the secret's project, and ensure the
`credential_leases` collection has no TTL. Contending executions fail explicitly;
dispatch must serialize or retry them. Crash recovery requires attended verification
of termination and credential durability; see `internal/credentiallease/README.md`.

The read-only checker requires mode 0700 homes and mode 0600 auth files, rejects
symlink homes/auth files and canonical directory collisions, requires ChatGPT
mode with complete access/ID/refresh credentials, compares ID-token `sub` for
user identity and `tokens.account_id` for selected workspace, and requires the
workspace to agree with the ID-token OpenAI claim. Refresh credentials must be
pairwise distinct. Unexpected or missing identity fields fail closed. It reports
fixed profile labels and pass/fail outcomes only. Offline claim comparison does
not verify JWT signatures or credential freshness; smoke tests establish live
validity after activation.

`publish-cloud` runs the complete checker before calling Secret Manager. The
credential is supplied as a file path, never a token argument or environment
literal. It adds a version to the specified existing secret and reports only
success or failure; it does not update Cloud Run jobs or restart executions.

Offline verification (fixture JWTs and mock Codex/gcloud, no network):

```bash
bash tools/attended/test_codex_oauth_profiles.sh
bash -n tools/attended/check_codex_oauth_profiles.sh
bash -n tools/attended/provision_codex_oauth_profile.sh
```
