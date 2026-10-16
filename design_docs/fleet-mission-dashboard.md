# Fleet mission — iteration 43, 2026-10-16

- Latest released binary: v0.54.12. dev: `a8042fa3f` (this iteration's landing).
- LANDED this fire: #1853 → `a8042fa3f`, `skill:gate3b-ci-poll-exceeds-foreground-bash` (motoko, 10-01).
  Gate 3b's 30-min CI poll now runs as a backgrounded Bash call (`timeout: 2000000`), with a SHA-bearing verdict file and bounded reads. Previously it was killed at the 600 s foreground cap.
- Judge (sonnet, Agent tool, own worktree): PASS 74 with a blocker (the foreground-slice form parked live runs), then PASS 90 after that form was removed.
- Planner and executor: codex `gpt-6.1-sol` (recipe). The codex sandbox refuses writes to `.agents/`, so the controller mirrored that copy. Controller: opus. Metered $0.
- Dev CI on `c4fab92c2`: required checks green. The non-required ubuntu Build was red on `TestWorkerCLIProcessExitLifecycle` (not fleet code).
- Observation: dev CI runs 31–55 min, against Gate 3b's 30-min cap.
- Next (banked): `skill:gate4-status-stamp-shape-unprescribed`, then `mission:index-regenerate-needs-rotatable-archive` (both motoko, 10-01, in scope). #16 `quorum:tier2-salvage-fold` is not live. The agent-tool `slots>1` tickets stay PARKED-ON-LANE.
- Parked on Mark:
  - D-FLEET-16: skill source (A recommended / C default).
  - D-FLEET-18: `tools/eval` scope (A / C).
  - D-FLEET-19: ADC quota project (A / C).
  - D-FLEET-20 (new): the quorum's OpenAI seat bills an out-of-credit API key (A: route via the codex subscription / B: top up / C default).
  - D-FLEET-17 is moot; Mark may close it.
- Cadence: fleet 6h, ticket-driven. Thread #1768; no new human directive. Nothing reloaded.
- Clauses: 1 unmeasured, 2 unmet, 3 met (48 open), 4 prior evidence, 5 preserved.
