# Iteration 77 — independent evaluation

Judge: `sonnet`, spawned with the Agent tool, foreground, read-only (agent `a92b8995c2ab0eef5`; 124,936 subagent tokens, 31 tool
uses, 272 s). It judged [the draft log entry](draft-log-entry-as-judged.md) and [the draft charter diff](draft-charter-diff-as-judged.patch),
and was told to refute them. Its report is below, condensed only to drop the formatting. The controller's disposition of
each finding follows the report.

## Verdict: PASS-WITH-EDITS, 84/100

Rubric: 100 points across the seven checks in the directive (census, causal chain, fold vs missed harness surface,
option-B scoping, no-other-routable, sweep, charter integrity), 10–15 each. Points were taken off for the sweep
count (F1), the over-strong "cannot occur" claim (F2), and stale "six"/"13 slots" text (F4). No blocking defect in the
disposition: the fold into D-FLEET-24 holds, no second ticket is routable, and the skill does not instruct the transport.

### F1 — BLOCKING-IF-LEFT: "7 new since the last sweep" is 14

`gh issue list -R sunholo-data/ailang --state open --limit 500` returned 57, matching `issue-sweep.tsv` (identical
number set). Diffing against `iteration61/issue-sweep.tsv` (committed 2026-10-26 10:13Z) gives 14 numbers not in it:
#1945 (fleet bookkeeping), #1947, #1950, #1953, #1954, #1955, #1956 (six `[ailang-explainers]` language/verify/bytecode
reports filed 10-26 12:06–13:32Z), #1974, #1980, #2029, #2034–#2037. "0 new queue items" survives, since all are
outside Authority. "55 with zero mentions" is correct; the non-zero rows are #1945 (19) and #581 (7).

### F2 — NON-BLOCKING: "A and B both remove the transport, so this error cannot occur" overreaches

The timeout and the 10-09 401 are codex multi-agent-layer errors. The 401 hit an attended session's
`/root/crew_final_evaluator` sub-agent and a `guardian` sub-agent, neither of them a `claude-sub` transport (`session_meta`
of the three 10-09 files: `codex-tui 0.162.0 … source vscode`; `subagent /root/crew_final_evaluator`; `subagent
guardian`). Under B, the skill's invariant (`gate-3-route.md:8-12`, every heavy role "spawned as a model-PINNED
Agent/Task/provider sub-agent, never inline") is a plausible reason a codex controller would still wrap the shell
recipe in a `gpt-6.1-sol` sub-agent. The parent's spawn message is encrypted (`gAAAAA…`), so the controller's reason
cannot be read.

### F3 — NON-BLOCKING: scope the option-B reading

One event, codex-cli 0.159.2, `danger-full-access` / `approval_policy: never` in both parent and sub-agent
`turn_context`. `--dangerously-bypass-approvals-and-sandbox` at `mission-control.sh:2431` is confirmed, present since
`de0e41099` (2026-08-15). That this fire came from the driver is inferred: the ticket says "API scheduled invocation… no
terminal driver verdict for this invocation". It ran in a sub-agent's background shell. `turn_context` policy counts
across the store: `read-only` 487+43, `workspace-write` 311, `danger-full-access` 151. Under a restricted sandbox it is
unmeasured.

### F4 — NON-BLOCKING: stale text in D-FLEET-24

The title still says "Six `agent-tool` tickets", the default "all six stay parked", and the ticket list "13 slots"
beside the new insert's 15. `ailang mission ticket open --json` returns 15, `last_seen` 10-21.

### Checks that passed (re-derived)

- Census: 1,049 files; 27 match `routing discovery`; `task_complete.error`: 1 timeout (10-02), 6 × 401 (10-09), 0 after
  10-09; 652 later files, 39 with 107 `function_call`/`spawn_agent` calls. Quote vs event classification correct. A
  looser pattern (`routing discovery`) yields 78 records against the controller's 65, in the same 27 files.
- Chain (UTC): 08:18:10 `pi:ollama/kimi-k3:cloud` refused; 08:18:16 ``Unknown model `sonnet` …``; 08:28:55
  `evaluator_transport` on `gpt-6.1-sol`; 08:49:56 `claude-sub --model claude-sonnet-5-5 … --print --output-format json`,
  backgrounded with a 1800 s cap; 09:03:15 turn error; 09:04:32 `followup_task`; 09:19:08 `JUDGE_RC=0`; 09:19:15 the judge
  JSON shows real Claude output (`total_cost_usd` 0.85, Claude Code result shape); 09:19:56 "PASS, 91/100". The child
  survived the codex error.
- Skill: nothing in the repo instructs a codex transport sub-agent (`gate-3-route.md:509` is an unrelated pi handshake);
  the stapledon charter only records the ticket (`stapledon-mission.md:512`). The fold stands.
- Tickets: 32 open, 63 slots; class 7 tickets, 25 slots (39.7%); every signature has a charter mention; 25 OPEN
  decisions; row 16's predicate unfired.
- Charter: 3 stamps (77, 76, 75); stamp 74 appended unchanged to the archive; numbered queue rows differ only by `59.`;
  `mission_decisions.sh --check` → `decision ledger valid: 40 rows`; numbers agree across STATUS, log, D-FLEET-24, row 59.

Only read, not re-run: the Preflight bullets (CI, ration, skill drift, directives, self-notices), the 14-driver-log zero,
and the repo `git grep`.

## Controller disposition

- **F1**: re-derived before applying. `comm -13` of the two tables' issue columns printed the same 14 numbers. Cause: the
  controller's filter was `createdAt > 2026-10-26`, which drops the issues filed on 10-26 after that sweep ran. The
  charter paragraph and the log bullet now carry the judge's wording, and the log names the error.
- **F2**: applied to the STATUS body, the D-FLEET-24 insert, row 59, the log's Ruled-out bullet and `census.md`. The
  controller checked the cited invariant at `gate-3-route.md:10-12` (resolved skill copy) before quoting it.
- **F3**: applied in the same places. The driver origin is now stated as inferred.
- **F4**: applied. D-FLEET-24's title reads "Seven", its default reads "all seven", and the ticket list says
  "15 slots at iteration 77; 13 at iteration 52".
- No round 2: every edit was a wording or count correction the judge supplied or that the controller re-derived, and the
  disposition did not change.
