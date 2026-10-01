### Added — `pi-or-glm-5-3-flash`, the pi arm of the lane A/B

The pi arm of the pi-vs-motoko `ailang_only` A/B, on the same model and route as
the 34 production lane agents. Its budgets match `motoko-lane-or-glm-5-3-flash`.

### Fixed — motoko's version query timed out silently under load

`HealthCheck` gave `motoko --version` 5 seconds. That limit was sized for a
retired fork that hung on the flag. On a loaded box bun's startup exceeded it,
and the query failed silently, leaving `motoko_repo` empty. Without it, session
logs can't be located and `ailang_only` lane tasks are refused. The query now
gets 30 seconds (once per executor), and a failure prints a warning.

### Fixed — the motoko lane canary names its file absolutely

Lane tools resolve paths from the policy's sandbox root, not the task
workspace. The canary asked for a bare `canary.txt`, so the model didn't find it
and used up the canary's time budget looking. It now gets the absolute path, as
benchmark prompts do.
