# Daemon re-stalls ~2 min after clean start; dead subscription indistinguishable from idle

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `daemon`, `pubsub`, `subscription`, `liveness`, `heartbeat`, `launchd`, `watchdog`, `re-stalls`, `last-consumed` across `design_docs/`
- **Estimate**: omitted (design-doc)

## Why

The follow-up sharpens the reproduction (backlog drains on startup, receiver goes quiet
within ~2 min, `launchctl print` says `running` throughout, a 5-day-undelivered approval
proves the subscription was dead across a 1d16h uptime) but does not change the decision
shape: there are still **multiple acceptable liveness remedies** — last-consumed timestamp,
logged receive error + resubscribe, non-zero exit so launchd restarts, or surviving binary
replacement (OS_REASON_CODESIGNING on an ad-hoc-signed, continuously rebuilt binary). Three
acceptable remedies is a design decision, not a direct fix; an earlier backlog row labelled
the same shape `direct-fix` and the triage skill's "two traps" section explicitly flags that
as a mislabelling.

Coverage check: `design_docs/planned/m-message-plane-trust.md` AC3 observably gates the
*cloud* push plane, not the local daemon receiver; the archived
`design_docs/archive/v0_5_1_m-background-agent-daemon.md` covers background agent execution
only; existing backlog rows (`design_docs/planned/ailang-core-backlog.md`, both 2026-09-15
daemon rows) record the same report but decided nothing. No existing doc rules on daemon
receiver liveness — this is a real gap, not a bad query.

Recommended design-doc scope: (1) observability — a last-consumed timestamp or health
surface that distinguishes "idle" from "subscription dead"; (2) reconnect-with-backoff on
receiver error instead of going quiet; (3) a logged, non-silent failure path when the image
is invalidated (codesigning) so launchd or a human can see it. Consumer impact: every Daneel
escalation and watchdog alert is written but never announced while stalled.
