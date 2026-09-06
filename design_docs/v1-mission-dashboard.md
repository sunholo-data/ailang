# Mission Dashboard — V1
*Iteration 340, 2026-09-07. History: v1-mission-log.md and charter STATUS.*

## Goal and delivery
- Release v0.35.1; N=12 design docs before v1.0.0; goal unmoved.
- `m-pi-evaluator-session-handshake` LANDED via PR #1069 at `aeeafc880`.
- Pi children receive canonical messaging store/project values; `AILANG_STORAGE` is preserved.
- Gate 3 now carries one source-bound evaluator read/list/summarize/ack/judge preamble.

## Up next (banked)
1. m-cachesrc-cognitive-complexity — attributable Sonar maintainability red on cache M2.
2. m-coordinator-codex-401 — coordinator websocket auth diverges from healthy OAuth CLI.
3. m-cache-artifact-adversarial-decode — separate hardening after correctness work.

## Routing and cadence
- Authoritative .claude mission-control skill; scheduled Codex Sol controller.
- Agent roles: Astra designer; Sol planner/executor; gpt-5.5 evaluator transport wrapper.
- Ollama MiniMax violated the bare-command handshake; its PASS90 was rejected as transport.
- OpenRouter MiniMax obeyed the handshake and independently passed 99/100; generator != judge.

## Parked on Mark
- 59 ledger rows, five OPEN: D-55 threat scope; D-56 reviewer independence;
  D-57 cache naming; D-58 pi-runner snapshot direction; D-59 iter339 cap disposition.
- No unattended answer was inferred; recorded defaults remain in force.

## CI, quota and workspace
- PR #1069 and merge SHA `aeeafc880`: CI, Build and Release, and docs all green.
- Metered $0.43507610; GLM flat-rate imputation $0.01772879 reported separately.
- Role tokens: designer/planner/executor/wrapper not reported; Pi 4,532,187 invalid +
  4,796,786 accepted tokens. Gate-4 base `aeeafc880` at 2026-09-06T22:13:01Z.
