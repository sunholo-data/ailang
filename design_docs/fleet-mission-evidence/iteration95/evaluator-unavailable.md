# Fleet iteration 95 independent evaluator transport failure

No judge verdict was produced. The requested idle screen remains UNJUDGED; this document is transport evidence, not a NONE verdict or a sprint evaluation.

The prescribed resolved lane was `claude:claude-sonnet-4-6` (`recipe`, `declared:provider-pin`). A bounded subscription-only `claude-sub -p --model claude-sonnet-4-6 --output-format json "reply with exactly: ok"` probe returned rc=1 in 6.78 seconds: HTTP 429, `usage_limit_reached`, weekly reset Nov 9 at 6am Europe/Copenhagen. This lane was recorded dead with `mission-lane-dead.sh`.

The remaining declared chain was `opus`. This Agent transport does not expose a model-pin parameter; its role bridge tested the same alias through the subscription-only provider CLI, `claude-sub -p --model opus --output-format json "reply with exactly: ok"`. It returned rc=1 in 6.16 seconds with the same typed weekly limit. The opus alias was recorded dead. Earlier MiniMax was already exhausted by the driver this fire and was not retried. No undeclared lane or generator-family judge was substituted.

Both probes reported zero input, output, cache tokens and zero cost. No real evaluator prompt was delivered after probe failure; no judge session read CLAUDE.md, no judge protocol acknowledgement occurred, and no report/mutation work ran. The Codex bridge independently read CLAUDE.md and the complete gate-3-route.md, verification-protocol.md, codex-lane-false-greens.md, and sprint-evaluator SKILL.md solely to operate this transport. Its reading is not evidence of a judge handshake.

The isolated sibling worktree is `/Users/voightkampff/.ailang-driver-pin/.wt-fleet-i95-evaluator-20261107`, created from `252680e1beb4e6cf1709a010cc1c45838b91b9f8`. No code, pin, routing policy, ticket or mission ledger was modified. This report is the worktree's only new file. Retain the no-pick decision as unjudged and park substantive work until an independent declared evaluator lane has quota; the controller must not accept this bridge's own verdict.

Transport transcripts and command/status/usage evidence are in `/tmp/fleet-i95-evaluator-3q3dt4gq/transport.json`, `probe.stdout`, `probe.stderr`, `probe-opus.stdout`, and `probe-opus.stderr`.
