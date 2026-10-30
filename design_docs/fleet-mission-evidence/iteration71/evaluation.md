# Iteration 71 — independent evaluation (sonnet, Agent tool, read-only on the pin)

66,858 subagent tokens / 11 tool uses / 99 s. Input: `draft-records-as-judged.md`.

- **Record 1, ghost resolution of `gate0:watermark-advanced-by-dying-fire` + `gate0:watermark-advanced-before-processing`: PASS-WITH-EDITS 88.**
  - Every ask in both tickets is built at `d45cf4e6b`: the inclusive window at `scripts/mission_directives.sh:199`, `--mark`-only writes (temp file, `mv -f` and a lock), and PROCESS-THEN-MARK in both skill copies, which are byte-identical. `5ad75cad9` is an ancestor of HEAD.
  - Suite: 33 passed, 0 failed.
  - Mutation drill via `MD_SCRIPT_UT` on `mktemp -d` copies: `>=` changed to `>` at :199 gave 21 pass / 12 fail. Forcing `--mark` to the newest boundary gave 31 pass / 2 fail ("3 out-of-order", "3 late-arrival"). `git status --porcelain scripts/mission_directives.sh` stayed empty.
  - Edits: name the `planned/` design doc as a hygiene residual (applied). The judge did not verify the issue-close time. The controller's `gh issue view 981` read CLOSED 2026-10-29T07:43:12Z, so it is kept, with that instrument named.
- **Record 2, D-FLEET-34: PASS-WITH-EDITS 78.**
  - Reproduced 133 pi-controller slots since 07:24Z, 103 paused at fire, 30 past admission (5 COMPLETED, 2 CRASHED, 1 DIED-PRE-GATE-0, 1 KILLED, 18 PAUSED at gates 1–3b, 1 REAPED), and 1 KILLED.
  - Exactly one pi controller spawn exists (`mission-control.sh:2480`). Resume reuses it, and the other pi calls are `--no-tools` probes.
  - Edits (all applied): D-FLEET-4/-7 rule the sample count and the 600-s budget, not the grace. Quantify the sample saving (240–360 s). Add the 1-kill-in-30 caveat. Name `STALL_CHILD_AGE`.
  - Strongest objection: 30 slots is thin. It supports "no evidence for A", not "A is unneeded".
- **Overall 83** (70 passes).
