# Daemon goes silent after its own binary is rebuilt; dead subscription indistinguishable from idle

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `daemon`, `codesign`, `launchd`, `kickstart`, `re-exec`, `subscription`, `last-message` across `design_docs/planned/` and `design_docs/archive/`
- **Coverage**: none found that rules on this. `design_docs/archive/v0_5_1_m-background-agent-daemon.md` covers background agent execution only, not message-delivery liveness. `design_docs/planned/m-message-plane-trust.md` AC3 gates the *cloud* push plane (`chains health` RED on stale push receipt), not the local daemon's Pub/Sub receiver — related, worth citing, not a ruling.

## Why

The mechanism is well located: the daemon runs from `~/go/bin/ailang`, ad-hoc/linker-signed; a rebuild while it runs (fleet builds from source) invalidates the running image (`OS_REASON_CODESIGNING` on `launchctl kickstart -k`), the Pub/Sub receiver dies without logging, and launchd keeps reporting `state = running`. Result: five days of undelivered notifications, including an approval request from 6 September.

This is **not** direct-fix, despite the report enumerating remedies. It offers three acceptable ways to satisfy need (1) — log the receive error, exit non-zero so launchd owns the restart, or expose a watchable last-message-consumed timestamp — plus at least three more for need (2) (re-exec on SIGTERM, launchd-managed restart, codesign-invalidation logging). Choosing among them is a real design decision: exit-non-zero changes the process's exit-contract with launchd; a last-consumed timestamp is a new observable surface someone must watch; re-exec interacts with how launchd treats image invalidation at all. "Three acceptable remedies" is the definition of row 3 in the rubric — a decision someone could disagree with. Note the shared backlog row (`design_docs/planned/ailang-core-backlog.md`, 2026-09-15) labels this `direct-fix`; that label is what the skill's measured-traps section was written to correct.

There is also a follow-up in today's inbox — "daemon re-stalls ~2min after a clean start; only the startup drain delivers" — meaning any liveness fix must handle repeated stall, not just detect the first one. Design doc should cover the full requirement set: (1) degraded subscription visible/observable, (2) survive or at least log image invalidation on a rebuild-from-source box, (3) note that startup drain alone is insufficient without (1).
