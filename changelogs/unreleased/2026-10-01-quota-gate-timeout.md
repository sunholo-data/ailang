### Fixed — mission ration gate blocked Codex on every fire once the keychain token went stale

The Anthropic usage reader read only the keychain credential, which Claude Code lets go stale.
The HTTP read then failed with 401, and the `claude -p /usage` fallback (178s measured, cut at 30s)
pushed `ailang mission quota --over` to 31-39s, past the driver's 15s bound. The gate failed
closed, so from 2026-09-30 21:40 all seven fires blocked Codex while it sat at 51% of a 68% allowance.
The reader now takes the fresher of the keychain and `~/.claude/.credentials.json` tokens
(the live read is now 4s), and the driver's bound is 60s (`MISSION_QUOTA_TIMEOUT`), covering
the reader's own budget.
