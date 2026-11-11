### Added — pi runner: --tool-stall-seconds bounds an open tool call separately (2026-11-11)

A pi executor or evaluator that runs a long named gate command (a full test target) was killed at the
single 600 s no-progress bound, because the runner had one bound for everything. Ticket
`tool-hang:pi-evaluator:stall-bound-below-single-test-run` (stapledon iter 86): the command was measured
at 396-573 s under load 12-25 and died at 600 s under load 30-60.

`scripts/mission_pi_run.sh` now takes `--tool-stall-seconds N` (env `MISSION_PI_TOOL_STALL_SECONDS`).
While a tool call is OPEN the no-progress bound is this value; while the model is silent with no tool
open, the bound is still `--stall-seconds`. Defaults are unchanged: when not given, the tool bound is the
effective `--stall-seconds`, so a caller that passes neither flag gets the same kill time and verdict.
A given value is validated (positive integer of at most 9 digits, no leading zero) and refused with rc 14
and a verdict file.

The verdict JSON gains `stall_seconds`, `tool_stall_seconds` (configured; `null` if non-numeric),
`stall_bound_seconds` (the bound that fired; 0 when the run did not end on a stall) and `poll_seconds`
(the poll interval after the clamp, which now follows the smaller of the two bounds). The `tool_hang` stderr line names the bound.

The mission-control skill (`resources/gate-3-route.md`, mirrored to `.agents/skills`) gets rule (d): a
named long gate command is given `--tool-stall-seconds`, derived from its measured duration (never a
raised `--stall-seconds`), and the `tool_hang` retry may re-run such a command with that flag.
