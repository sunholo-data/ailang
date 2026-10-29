### Fixed — Sprint-plan placement and terminal human-decision gates (2026-10-29)

Sprint planners now keep plans outside `design_docs/planned/ailang-core-triage/`.
The JSON helper refuses raw or canonical triage plan paths before creating state
or importing messages; triage design rows remain valid as the design argument.
Human-decision gates require a cited, still-OPEN decision and terminal behavior:
STOP, commit current work, report `BLOCKED: <decision>`, and end the run.
Already-ruled decisions are copied as rulings, never reopened as gates.
See the [sprint plan](../../design_docs/planned/sprint-plan-planner-placement-stop-gates.md).
