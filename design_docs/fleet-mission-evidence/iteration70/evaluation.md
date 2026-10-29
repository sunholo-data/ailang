# Iteration 70 — independent evaluation

Judge: `sonnet` via the Agent tool, in its own detached worktree `.wt-fleet-iter70-judge`. Planner was `opus`, executor `claude:claude-haiku-5-5`, so the judge is a different model from both. They share a vendor, so vendor-level independence is not claimed.

## Round 1 — at `5d5d5850f`: PASS 88/100, 0 blocking
Rubric: correctness and claim accuracy 36/40 · acceptance and gates 24/25 · mutation resistance 17/20 · regression safety 8/8 · docs and changelog 3/7.
- **Ghost claim confirmed first-party.** pi-ai `openai-completions.js:1208-1228` `mapStopReason` builds `Provider finish_reason: ${reason}`. `print-mode.js:115-117` writes it to stderr with rc 1. The driver captures it with `2>&1`, `TRANSIENT_SIG` matches it case-insensitively, and `RUNTIME_QUOTA_SIG` does not, so it is not demoted. There are 3 attempts with 45 s/90 s backoff, and the retry resumes by session id. No `MISSION_TRANSIENT_RETRIES` override exists in `tools/launchd/mission-env/*.env`, `~/.config/ailang/mission-*.env`, the stapledon plist or `launchctl getenv`.
- **Gates:** suite 16/0; `make test-launchd-drivers` rc 0 (52); `check-changelog` rc 0; shellcheck +5, all SC2319.
- **Strongest objection:** the suite did not defend its own stderr claim. With the fake's stderr selector broken and `2>&1` dropped, the suite still passed 16/16 (T5b+M2).
- **Non-blocking:** the backoff values and the 3-attempt bound are unpinned; the changelog category is `Changed` for a test-only change; the driver slug is logical while pi's is physical (latent); `network_error`/`content_filter` are outside the literal.
- **Verdict on resolving the ticket:** agree, citing #1391 plus this merge and saying it is test-only.

## Controller fix — `2c456d263`
The fake records which stream it wrote (`pi.stream`), and the provider-error case asserts that attempt 1 used stderr. Controller drills: T5a, T5b and T5b+M2 are each red on the new assertion only, and the file restores byte-identically (`cmp`).

## Round 2 — at `2c456d263` (resumed): PASS 91/100
- The diff matches the controller's list exactly; suite 16/0; shellcheck 70 (SC2319 34; round 1 had 33, base 28).
- T5a, T5b and T5b+M2 are each red on the new assertion only.
- New mutant LIE (the fake prints stdout but labels it `stderr`) passes 16/16, with or without M2. The judge's ruling: the assertion is a fidelity pin on the fake, not an independent observation of the stream. Combined with round 1's "did not escape to stderr" assertions it defends the seam. A paired edit of two adjacent lines can only be caught against real pi, and that residual is accepted.
- Round-1 findings: stderr-selector gap CLOSED; shellcheck counts CLOSED; backoff, 3-attempt bound, changelog category and logical slug still open, non-blocking.
- **Strongest remaining objection:** nothing in the suite checks the fake's stream against real pi; `network_error`/`content_filter` deserve a line in the resolution note.
