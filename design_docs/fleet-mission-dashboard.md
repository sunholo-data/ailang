# Fleet mission — iteration 44, 2026-10-16

- Latest released binary: v0.54.14. dev: `91d5d6390` (this iteration's landing) plus later commits.
- LANDED this fire: #1866 → `91d5d6390`, `skill:gate4-status-stamp-shape-unprescribed` (motoko, 10-01).
  Gate 4 now states the canonical STATUS stamp shape (`canonicalStatusRe`) for every mission, why a bad charter stamp reds only after rotation, a pre-commit grep, and the normalize repair.
- Judge (sonnet, Agent tool, own worktree): PASS 94 with 0 blocking, then PASS 97 after 3 wording fixes. A simulated rotation for 5 missions was red/green as expected.
- Planner and executor: codex `gpt-6.1-sol` (recipe). The controller mirrored `.agents`, which the sandbox refuses. Controller: opus. Metered $0.
- Dev CI on `bd30497a0`: `lint` red on `fmt-check` in `internal/eval_harness` (not fleet code), fixed upstream by `e1ef33fa5`.
- Next (banked): `mission:index-regenerate-needs-rotatable-archive` (motoko, 10-01, `cmd/ailang/mission*`, in scope). Then `eval:motoko-connection-probe-orphaned` (check the path against D-FLEET-18 first). #16 is not live, and the agent-tool `slots>1` tickets stay PARKED-ON-LANE.
- Parked on Mark:
  - D-FLEET-16: skill source (A recommended / C default).
  - D-FLEET-18: `tools/eval` scope (A / C).
  - D-FLEET-19: ADC quota project (A / C).
  - D-FLEET-20: quorum OpenAI seat billing (A / B / C).
  - D-FLEET-21 (new): Gate 3b's 30-min CI cap is shorter than dev CI (A: 3600 s / B: auto re-poll / C default).
  - D-FLEET-17 is moot.
- Known defect, not ticketed: `rotate-log` prepends a duplicate "Older entries are ARCHIVED" banner every run; hand-collapsed.
- Cadence: fleet 6h, ticket-driven. Thread #1768; no new human directive. Nothing reloaded.
- Clauses: 1 unmeasured, 2 unmet, 3 met (47 open), 4 prior evidence, 5 preserved.
