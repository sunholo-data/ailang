### Fixed — pi agent evals no longer run under the attended-session protocol gate

The pi session-protocol gate is a global pi extension for attended dev sessions. It refuses
write/edit/mutating bash until the agent reads CLAUDE.md and acks the inbox protocol, and it was
armed in every pi agent eval on the rig. Models that happened to call `session_protocol_ack`
could work (every Phase 0 base-12B pass acked first). A fine-tune that never learned the ritual
scored 0/24 with every write refused. Measured 2026-11-06 by capturing pi's wire traffic against
a scripted model.

- `executor.Task.EvalRun` (set only by `BuildEvalExecutorTask`) makes `BuildEnvironment` inject
  `AILANG_EVAL_RUN=1`. The variable is unset for every other child, so a value in the operator's
  shell cannot leak through, and it is refused in `Task.ExtraEnv`, so a task cannot disarm the
  guard itself.
- `session-protocol-gate.ts` stands down when `AILANG_EVAL_RUN` is exactly `1`. This applies to
  the repo source and the embedded pi asset; tests cover the exemption, its controls and the
  leak.
- **Baseline boundary:** pi agent results from before this change ran under the gate. Compare
  them only within one side of the boundary.

### Added — fine-tune pilot-2 tooling (`tools/finetune/`)

- `agenttask/`: renders the exact agent task prompt the harness sends.
- `trace_proxy.py` and `fake_upstream.py`: capture an agent harness's full wire traffic, and
  script a model with no GPU.
- `ft_clean.py`: normalises stale eval-workspace paths and noise in trajectories (55% of samples
  carried another run's absolute workspace).
- `stale_fix.py`: verified staleness pairs (passing-when-banked programs that fail today, plus
  a model fix kept only when its output matches the benchmark exactly).
