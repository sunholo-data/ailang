### Changed

- `mission-control` skill: the "read the environment by NAME, never by value" rule now sits at the top of `SKILL.md`, above the Repo Profile. Before, it lived only in `resources/gate-0-preflight.md`, and controllers read env values before reaching it (fleet iterations 51, 52 and 78, World iteration 332). The `.agents/` mirror is synced.
