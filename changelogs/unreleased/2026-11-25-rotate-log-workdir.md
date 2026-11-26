### Added — Target a record checkout with `rotate-log --workdir` (2026-11-25)

`ailang mission rotate-log <name> --workdir <dir>` uses the selected log or status stream under
`<dir>/design_docs` for rotation and index regeneration, including `--index-only`. The
record checkout can be separate from the mission registry.
[Fleet iteration 140 plan](../../design_docs/fleet-mission-evidence/iteration140/plan.md)
