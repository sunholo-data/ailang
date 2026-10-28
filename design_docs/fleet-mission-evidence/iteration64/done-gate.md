# Done-gate and reality check (iteration 64, out of sandbox)

## Reality check (controller, first-party, before routing)

- `cmd/ailang/messages_crud.go` `runMessagesRead`: `--peek` ("Show without marking as read") guards `MarkInboxMessageRead` on the by-id, `--latest` and `--all-unread` paths. No CLI change was needed.
- Prod store (`AILANG_STORAGE_MESSAGING=gcp`): `ailang messages read --peek --json <id>` on an unread controlplane message printed it, and the message still listed under `messages list --unread --json` (count 1; control: unread total 200).
- Flag order: `ailang messages read --peek <id> --json` printed HUMAN output, so the trailing `--json` was ignored (Go `flag` stops at the first positional). A trailing `--peek` is therefore ignored too, and the read marks the message. The judge confirmed this with a standalone `flag.NewFlagSet` program: `id --peek` gives `peek=false`.
- Base gates at `9646caebd`, all rc 0: `sync-agents-skills.sh --check`, `test_agents_skills_sync.sh`, `make check-changelog`, `scripts/check_context_docs.sh`.

## Done-gate

- `env -i HOME PATH USER TMPDIR make test-launchd-drivers` at `92a5420e2`: rc 0, `run-suites: all 49 suites passed`.
- The same at the final head `3f320eac4`: rc 0, 1114 PASS lines, 0 FAIL, 49 suites, `launchd drivers: tests + bash 3.2 syntax OK`.
- Dry-runs under the idle, armed `triage` profile (no pidfile before or after): rc 0 both. Triage's idle short-circuit (`queue unchanged … idle, nothing spent`) exits BEFORE the lane probe, so neither run exercised lanes. They are not counted.
- Dry-runs `MISSION_PROFILE=fleet` under a synthetic HOME (every `~/.ailang/*` except `state/`, plus `.config .codex .claude .local .pi .gitconfig` symlinked), `AILANG_DRIVER_PINNED=3f320eac4`:
  - healthy: rc 0, `lanes=ok`, `DRY RUN ok`, pidfile under the synthetic `state/`
  - degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i64`): rc 0, `lanes=DEGRADED(1)`, `codex model 'bogus-model-i64' unusable: probe failed (rc=1)`, executor handed to `claude:claude-haiku-5-5`
  - the real `~/.ailang/state/mission-fleet.pid` sha was unchanged before and after.

## Controller follow-up on judge finding 1 (`3f320eac4`)

- BSD awk 20200816: `split("a`b\nc`d", s, "`")` returns 4, so newlines also split and the inline-code parity flips after a wrapped line.
- The fix is `gsub(/\n/, " ", clause)` before the split.
- Mutant: a second inline `ailang messages read <id>` appended to clause (c)'s second line. With the pre-fix test it survives (rc 0); with the fixed test it dies (rc 1). The no-`--peek` mutant still dies (rc 1). Clean: rc 0.
- Restored by `cp`, `cmp` 0. The judge re-reviewed in round 2 (PASS 91) and replayed its survivors itself.
