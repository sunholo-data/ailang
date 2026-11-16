### Fixed — design-quorum: the OpenAI seat runs on the codex subscription (2026-11-16)

D-FLEET-20 A (Mark, attended). The quorum's OpenAI seat was the models.yml row `gpt6-1-sol`,
which bills the metered OpenAI API. That org has no credits, so the seat was recorded absent
(`429: You have no credits remaining`) on every quorum that seated it, and quorums ran with
no OpenAI vendor. The default pool now seats `codex:gpt-6.1-sol`, which runs `codex exec` on
the ChatGPT subscription ($0 metered), the lane the mission roles already use.

- New seat runner `quorum.RunCodexSubscriptionReviewer`
  (`internal/mission/quorum/codex_reviewer.go`). It uses the same prompt and schema as the other
  seats and runs in an empty temporary directory with `--sandbox read-only`, `--ephemeral`,
  spawn overrides off (D-FLEET-12), the prompt on stdin, and `OPENAI_API_KEY` stripped. The CLI
  runtime refuses a non-subscription `auth.json`. `AILANG_CODEX_RUNTIME=daemon` (the missions)
  goes through the managed daemon, as `ailang mission codex-exec` does. Tokens come from the
  stream's `turn.completed` usage. Errors and empty or invalid replies are recorded as named
  absences.
- `cmd/ailang/design_quorum_codex_seat.go` routes any `codex:<model>` pool entry to that lane.
  It gates the seat on the Codex bucket the way the driver gates the roles' codex rungs:
  `over`, `expired` and `unknown` all refuse. Every other seat is unchanged.
- Live probe (CLI runtime, rig): the seat answered present at `cost_usd: 0`.
