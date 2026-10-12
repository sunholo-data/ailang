### Fixed — mission driver exit-path notices are bounded and reach the canonical inbox (2026-10-12)

The no-usable-controller refusal, the late-kill-after-record notice and the rc-fail notice in
`tools/launchd/mission-control.sh` called `ailang messages send` and `gh issue comment` unbounded, so
a hung send wedged the exiting process, and with it the launchd job: launchd does not start a second
instance of a live job, so that mission's next fire was blocked too. All three now go through one
helper, `_mc_exit_notice`, which makes one `_mc_bounded "$NOTIFY_TIMEOUT"` attempt per channel, never
aborts the exit, and logs a failure with its reason. The sends are also store-scoped per command to
prod Firestore, exactly as `_mc_notify` scopes them: under launchd they previously landed in the rig's
local SQLite inbox and exited 0, so nobody read them. The slot-verdict notice gets the same store
scoping. Notice titles, bodies and exit codes are unchanged. One consequence to know: the message
titles are not date-stamped yet, so a repeat of the same title is refused by the inbox and logged as
"already in the inbox", not re-sent (follow-up `driver:exit-notice-titles-undated`).
