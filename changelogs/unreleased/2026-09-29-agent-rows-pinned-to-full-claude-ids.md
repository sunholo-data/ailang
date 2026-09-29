### Fixed — agent rows benchmarked a different model than their label (2026-09-29)

Eight Claude rows passed CLI aliases (`sonnet`, `opus`, `fable`, `haiku`) as `agent_model_name`.
The aliases follow the newest model. On the CLI the harness ran, `sonnet` meant claude-sonnet-5
and `opus` meant claude-opus-5. So agent_suite's `claude-sonnet-4-6` seat measured Sonnet 5.
Every row now pins the full id, and `TestModels_ClaudeCLIRowsNeverUseAnAlias` enforces it.
`haiku` stays as an `aliases:` entry on claude-haiku-4-5 so cost resolution does not change.
The agent executor also preferred the VSCode-bundled CLI, 2.1.259, which rejected
claude-sonnet-5-5 as `unrecognized_model`. It now prefers the self-updating `~/.local/bin/claude`.
