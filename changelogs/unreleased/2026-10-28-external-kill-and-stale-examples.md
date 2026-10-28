### Added — `external_kill` error category for agent evals (2026-10-28)

An agent CLI that is killed by a signal the harness did not send is now banked as `error_category: external_kill`. Before, such a claude run was banked as `api_error`, and a codex run was graded on its partial solution. The harness's own kills (timeouts, cost and thrash caps, cancellation) are all SIGKILL, so a SIGTERM death came from outside. On the rig on 2026-10-27/28 that was a sibling eval agent's `pkill -f "ailang run"`.

- **claude:** exit 143 (Claude Code traps the signal) and an untrapped `signal: terminated` both map to the `external_kill` finish reason.
- **codex:** a SIGTERMed codex exits 0, because the node wrapper and the native binary both shut down cleanly. Its only trace is a `turn.started` with no `turn.completed` or `turn.failed`. That shape is now `external_kill`, with an `Error` set, so the runner banks the run instead of grading a half-written workspace (previously `compile_error`).
- **Other executors:** the categoriser also maps an executor error string that carries `exit status 143` or `signal: terminated` (for example opencode's) to `external_kill`.
- **Validity:** like `api_error`, the rows are banked invalid (harness error) and retried by `--skip-existing`.
- **Reporting:** they are excluded from capability stats, counted in the sweet-spot report (`external_kill_count`, CSV column, `error_categories`), bucketed `provider_blocked` rather than `capability_blocked`, and listed as run-unmeasured in `tools/nightly_classify.py`.

### Changed — the quorum's claude seat reads its system prompt from a file (2026-10-28)

The Anthropic quorum seat (`internal/mission/quorum/claude_reviewer.go`) now passes its system prompt with `--system-prompt-file`, through the same helper as the claude executor. The prompt no longer appears on the process table.

### Fixed — an old `ailang examples download` no longer hides newer examples (2026-10-28)

`~/.ailang/examples` used to win over the corpus embedded in the binary unconditionally. A user with an older download therefore never saw examples added since, and the CLI listed 238 of the binary's 239. A download that lacks any example the binary carries is now skipped in favour of the embedded corpus, with a one-line stderr note saying how many examples it is missing and how to refresh it. Paths are compared under `runnable/`, so either manifest spelling matches. A download at least as new as the binary still wins, and `AILANG_EXAMPLES` always wins.
