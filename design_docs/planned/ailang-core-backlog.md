# AILANG Core Backlog (triaged reports)

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | Daemon goes silent on binary rebuild: dead subscription indistinguishable from idle, approvals undelivered 5 days | bug | direct-fix | The report locates the cause (running image invalidated by codesigning when the binary is rebuilt underneath it) and enumerates three acceptable liveness fixes — log the receive error, exit non-zero so launchd restarts, or expose a last-message-consumed timestamp — so implementing any/all of them needs no design decision; note the archived `design_docs/archive/v0_5_1_m-background-agent-daemon.md` covers background agent execution only, not message-delivery liveness (searched: daemon, codesign, launchd, kickstart, re-exec — no other coverage found). |
