### Fixed — sprint-evaluator skill mirror drift (2026-10-27)

The evaluator instruction update left the Codex/pi skill mirror stale, failing required
mirror checks and retaining the obsolete `EVALUATION_RESULT: pass|fail` marker.
Re-synced the mirror to emit the final coordinator-readable `EVALUATION_VERDICT:` line.
See the [repair plan](../../design_docs/planned/sprint-plan-agents-mirror-sync-red.md).
