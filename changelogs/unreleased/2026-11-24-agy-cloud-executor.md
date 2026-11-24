### Added — agy on the cloud executor plane (M-AGY-CLOUD Phase 1)

Cloud executor jobs can run agy (Google Antigravity CLI) on the Google AI subscription,
through a cloud-only sign-in leased from Secret Manager (`AILANG_AGY_AUTH_SECRET`).
Subscription only: there is no API-key agy cloud lane.

- Images `agent-agy` and `agent-agy-go` (pinned agy at `$AILANG_AGY_HOME_ROOT/bin/agy`),
  built by `cloudbuild-dev.yaml` and `cloudbuild-release.yaml`; dispatcher variants
  `agy` and `agy-go`.
- Before cloning, an agy job reads the account's quota with `agy -p /quota` and refuses
  when it is over the shared `google` pace line (`AILANG_GOOGLE_DAILY_PACE`), the same
  evaluator the rig's mission loops use.
- A second agy task waits on the `agy-subscription` credential lane instead of launching
  to fail on the lease.

### Changed — one rotating-credential mechanism for codex and agy

The cloud codex credential code (exclusive Firestore lease, install, write-back of a
refreshed file, quarantine) is now provider-neutral (`cmd/ailang/coordinator_cloud_rotatingcred.go`);
codex and agy differ only in a small spec. Codex behaviour is unchanged: its tests keep
every assertion. `executor.ErrProcessTerminationUnconfirmed` is shared, and the agy
executor now proves its process stopped (bounded kill-and-join) before a credential is
released.

### Fixed — the eval images accepted executors they do not install

`eval`/`eval-go` were marked "any CLI", so a motoko (or agy) dispatch to them was
accepted, cloned the repo, then failed `executable file not found`. They now list the
four CLIs `Dockerfile.agent-eval` installs, and such a dispatch is refused up front.
