### Fixed — sandboxed mission roles no longer hang on the session-protocol inbox step

- The pi session-protocol gate waives its `ailang messages` prerequisite for unattended sessions (`AILANG_MISSION_STAGE` or `AILANG_TASK_ID` set by the harness), matching CLAUDE.md's "unattended: skip the inbox". Reading CLAUDE.md and `session_protocol_ack` stay required, and attended sessions are unchanged.
- `scripts/mission_pi_run.sh` launches its pi children with `AILANG_MISSION_STAGE=1`. The mission sandbox cannot reach the message store, so the required inbox call hung every executor run of World iter-208 (banked as `stream_dead`) and timed out every pi evaluator handshake, sending each one to the fallback judge.
- The pi evaluator handshake (gate-3-route.md) drops its inbox step: read CLAUDE.md, classify, ack, then judge. The mission controller is deliberately not a stage; it still triages the inbox at Gate 0.
