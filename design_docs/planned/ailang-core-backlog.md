# AILANG Core Backlog (triaged reports)

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | Notify daemon's message receiver dies silently within ~2 min of start; dead subscription is indistinguishable from "nothing sent" | bug | direct-fix | The reporter located the mechanism (receiver goroutine goes quiet after an error/codesign-invalidation, launchd still reports `running`, no error logged) and proposed the fix (log receive errors, resubscribe on error, expose a last-consumed timestamp or exit non-zero so launchd restarts) — each is an obvious-once-seen change, not a disputed design trade-off; related but not covered by `m-message-plane-trust.md` AC3, which observably gates the *cloud* push plane, not the local daemon receiver. Note also the contributing cause worth one line in the fix: replacing the ad-hoc-signed binary while the daemon runs (OS_REASON_CODESIGNING) killed the image without a logged exit. |
