### Fixed — Stall watchdog tracks the controller's own session (2026-10-22)

Claude attempts now receive a fresh session id; progress includes that session's parent and foreground subagent transcripts across project paths.
Unrelated Claude and pi sessions no longer mask stalls, and unavailable Claude ids disable early kills while retaining the hard timeout.
Design: [M-FLEET-STALL-OWN-TRANSCRIPT](../../design_docs/planned/sprint-plan-stall-own-session-transcript.md).
