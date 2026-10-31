# Fleet iteration 74 — independent evaluation

Judge: `sonnet` via the Agent tool (resolver `agent-tool sonnet declared:alias-pin`), foreground, read-only
on the pin at `3ca6d7e87`. Controller: `claude:claude-opus-5-5`. Round 1 judged
[draft-records-as-judged.md](draft-records-as-judged.md); round 2 judged [records-as-revised.md](records-as-revised.md).

## Round 1 (87,963 subagent tokens, 16 tool uses, 92 s)

| Record | Score | Verdict |
|---|---|---|
| 1 (D-FLEET-38) | 76 | PASS-WITH-EDITS |
| 2 (resolve as ghost of `79ede6ee9`) | 48 | FAIL |
| 3 (D-FLEET-39) | 52 | FAIL |

Headline: **C10 refuted.** The draft read the 10-16 08:48–08:53Z collisions as a dead owner's stuck lease. The
judge extended the log window: execution `ailang-agent-executor-codex-go-dgcmd` attempt 0 was killed by the
first missed heartbeat (08:44:00Z) and published `status=failed` for task-d61964fa (08:44:03Z); Cloud Run's
retry (attempt 1, `max_retries=1`, `internal/coordinator/completion_binding.go:55`) re-acquired the lease at
08:44:53Z and kept working; the coordinator, seeing task-d61964fa `failed`, freed the lane
(`credentialLaneHolder` counts only queued/running tasks) and dispatched task-6dac947e at 08:47:47Z into a
LIVE owner. The retry finished, pushed and opened PR #1849 (09:10Z; later merged). Controller re-ran the
query first-party and confirmed every timestamp ([log](prod-logs-2026-10-16-codex-go-0843-0912.txt));
`gh pr view 1849` → MERGED, head `coordinator/task-d61964fa`.

Other findings, all confirmed by the controller:
- Record 1 C4/C5: `.agents/skills/design-doc-creator/SKILL.md:349` DOES have a "Quorum review (OPTIONAL …)"
  section; the `.claude` copy (`:339`) has the newer wording. The draft's "lacks the quorum section" was wrong.
- Record 1: the proposed `.agents/skills/*` guard arm in `_scope_is_harness` would make every product loop
  refuse any `.agents/skills` edit and let the fleet push arbitrary codex-read skill edits; put it in
  `_scope_is_fleet_support` or scope it.
- Record 1: rename-only copies carry broken paths — 21 non-mission `.agents` files reference `.Codex/skills/…`
  (no such directory; `.codex/` holds only `agents/`, `config.toml`, `hooks.json`), and eval-analyzer,
  benchmark-manager and post-release carry mangled model names such as `Codex-haiku-4-5`.
- Record 2: "0 lease failures after 08:53:51Z" is absence of exposure (11 later codex dispatches, none
  concurrent), not evidence of a fix.
- Record 3: ask (ii) confirmed not built (no re-queue keyed on the error; `ErrDispatchPermanent` at
  `daemon_tasks_exec.go:435` covers dispatch errors only); heartbeat cancels on the first failed write
  (confirmed); the dead-owner case is by design and unmeasured in the 25-day window.

## Round 2 (same judge, resumed via SendMessage; 99,874 subagent tokens, 5 tool uses, 40 s)

| Record | Score | Verdict |
|---|---|---|
| 1 (design-doc-creator → D-FLEET-36; other 26 → D-FLEET-38) | 84 | PASS-WITH-EDITS |
| 2+3 (both credential tickets open → D-FLEET-39) | 80 | PASS-WITH-EDITS |

- (a) The 16 rename-only skills are safe to overwrite: `diff -rq` over all 16 shows no "Only in" lines (so
  `rsync --delete` loses nothing); raw diffs of 8 further skills show only path/name rewrites, some of them
  wrong in `.agents` (`--from "Codex"`, `Codex -p`, `~/.Codex/settings.json`), so a verbatim sync fixes real text.
- Edits required and applied in the final rows: `Codex-haiku-4-5` sits in eval-analyzer (rename-only) AND in
  benchmark-manager and post-release (content), so only the first is repaired by the sync; option A must also
  extend `SKILLS` and `test_agents_skills_sync.sh`; the sync makes codex/pi read Claude wording (the existing
  verbatim precedent); D-FLEET-36 and D-FLEET-38 are one `_scope_is_fleet_support` change.
- Record 2+3 edits applied: PR #1849 opened 09:10Z 10-16, merged 10-29; `max_retries=1` is a code comment
  (`completion_binding.go:55`), the retry itself is observed; "two tickets and one adjacent ticket", not three;
  add option A2 (the lane holder also counts a codex task that failed within N minutes / until its execution is
  observed finished); the unread lease-release path is a must-check-first gate; the ask (ii) re-queue is a
  fallback, not A's core.
