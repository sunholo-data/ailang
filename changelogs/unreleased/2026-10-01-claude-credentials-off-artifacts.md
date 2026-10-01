### Security — the Claude OAuth credential no longer lands in the shared artifacts bucket (F-H6-1)

Phase 0 item P0.1 of M-EXECUTOR-UID-SPLIT (`design_docs/planned/v0_49_1/m-executor-uid-split.md`,
on branch `docs/h6-executor-uid-split`), approved by Mark 2026-10-01.

**The leak.** `execute-job` set `CLAUDE_CONFIG_DIR=/artifacts/tasks/<id>/claude` so the session
JSONL streamed to GCS. `writeCredentialsFile` then wrote `.credentials.json` (the Claude Max
access **and refresh** token) into that dir too. `/artifacts` is a gcsfuse mount, read-write on
every executor lane including the external apikey lanes, and gcsfuse ignores file modes, so the
`0600` meant nothing. Everything else Claude Code keeps in its config dir went there as well:
`.claude.json` (account identity), shell snapshots, todos and file history.

**The fix.**

- `CLAUDE_CONFIG_DIR` is now a local per-task dir, `$HOME/.claude-tasks/<taskID>` (mode `0700`).
  It sits outside the cloned workspace, so the agent cannot commit it. The credential, and the
  CLI's refresh rewrites of it, stay on container disk.
- Only `projects/` in that dir is a symlink to `/artifacts/tasks/<id>/claude/projects`. Claude
  still appends the session JSONL straight to the bucket while the task runs, at the same object
  paths as before. `session.jsonl`, `transcript.txt` and `metrics.json` are unchanged.
- `writeTaskArtifacts` looks for the JSONL under the bucket path directly, because `WalkDir`
  does not follow the symlink. The config dir is the fallback.
- `writeCredentialsFile` refuses, before writing anything, when `~/.claude` or
  `CLAUDE_CONFIG_DIR` is under `/artifacts` or resolves there through a symlink
  (`ErrCredentialsUnderArtifactRoot`). The claude executor treats that as fatal.

**Not covered by this change:** the credential objects already in the dev, test and prod buckets
(`tasks/*/claude/.credentials.json`) have to be deleted, and the credential rotated, as an ops
step (D1). The local (macOS) coordinator never set `CLAUDE_CONFIG_DIR` and is unaffected.
