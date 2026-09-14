# Mission runtime handover — 2026-09-14 (attended session)

Written for the next session. Read this before touching the mission loops or the binary
iteration path. Supersedes nothing; `mission-runtime-handover-2026-09-07.md` still holds
for anything not contradicted here.

## 1. The fleet is OFF, deliberately, and the pause now actually holds

All four missions are paused by kill-switch marker. **Do not clear these without Mark's
say-so** — his standing position is that loops stay off "until we can get real work out of
them".

```
~/.ailang/state/mission-control.disabled    (v1)
~/.ailang/state/mission-world.disabled
~/.ailang/state/mission-docs.disabled
~/.ailang/state/mission-motoko.disabled
```

**Why the previous pause failed, because it will look like a mystery otherwise.** On
2026-09-14 at 07:00:01 a **five-week-old one-shot launchd job** deleted the v1 and world
markers and the loops restarted. `dev.ailang.mission-resume.plist`, written 2026-08-08 to
lift a *different* (August quota) pause at the weekly reset — `Weekday=1 Hour=7` — ran:

```bash
rm -f "$HOME/.ailang/state/mission-control.disabled" "$HOME/.ailang/state/mission-world.disabled"
```

It was meant to self-delete, but its cleanup was `launchctl bootout <self>; rm <own plist>`
and **bootout kills the shell before the `rm`**, so it disarmed (vanishing from
`launchctl list`) while staying on disk. The 2026-09-09 15:25 reboot re-loaded it. Docs and
motoko survived only because that August job names v1 and world by filename.

The plist is now removed (preserved in the session scratchpad). **If a pause ever fails
again, `launchctl list` is the wrong instrument** — audit `~/Library/LaunchAgents/*.plist`
for `rm -f` and `.disabled`. The culprit is invisible to any repo grep. Only one such job
existed; the rest (daneel briefs, nightly-eval, orphan-sweep) are legitimate.

V1's in-flight iteration 354 was allowed to drain and completed at ~15:07; its next fire was
correctly skipped by the marker.

## 2. M4 is PASSED on criterion 1 of 2 — the binary path works

`docs-canary-guide-review-4` completed on the binary path: `outcome: pass`, **5/5 criteria**,
zero blocking findings, 53 tool calls with 0 repeated, $0.176 metered. Its **completed
replay made zero provider calls** (receipts 1→1, attempts 2→2, spend $0.55→$0.55).

Full record: `design_docs/verification/mission-iteration-reliability/live-trial.md`.

Three blockers had to be cleared first, and each is worth knowing:

1. **The evaluator lane was opencode, not pi** (`models.yml:5089`). The binary resolves roles
   from a table the shell driver does not read, and they had diverged. The opencode rung is
   additionally *structurally inadmissible* on this path ("executor budget contract is not
   admitted by role-run"), so the pre-repoint chain could not dispatch at all. Fixed in
   `7423434b4`.
2. **The evaluator's own `--tools` allowlist removed the session gate's only disarm.**
   `session_protocol_ack` is an extension tool; the allowlist (read/bash/grep/find/ls) dropped
   it, so the repo's `session-protocol-gate.ts` armed with no reachable unlock and bash was
   confined to three start-anchored regexes. 51 of 63 calls refused. Author roles were never
   affected because they carry no allowlist and so keep the ack tool. Fixed in `b06f3c148`
   via `--no-extensions --approve` for isolated stages (both flags required — see below).
3. **The 100,000-token cap.** Four runs, two models, two harness configs, all killed within
   1.5% of it.

### The successor, if you need another

`ailang mission retry-review` builds it. Two guards will reject a naive attempt, both
correctly:

- The authority revision must **contain** the artifact it authorizes (`merge-base
  --is-ancestor <candidate> <base>`). Committing the authority on `dev` fails.
- The base may add **only** authority. Branching from `sprint/docs-canary-review`'s tip fails
  with "successor base changes non-authority or product path".

So: branch from the candidate commit itself, add one authority file, push, fetch into the
docs clone. Working example: `mission/docs-canary-review-4-authority` (pushed), authority doc
`design_docs/verification/mission-iteration/canary/review-4-authority.md`, locator
`mark-docs-canary-review-4-approved-20260914`.

### What remains on M4

Criterion 2: *"Two additional frozen bounded tasks finish with hard checks and independent
review."* The tasks are `trials/inputs/review-packet.json` and `budget-accounting.json`, both
frozen and dry-run valid. **Both still declare 100,000 for their evaluator.** They are
smaller briefs so it may suffice — but see the arithmetic warning in §5.

## 3. Token accounting: one cap now means one thing

A cap used to measure a different quantity per harness, because every guard was
`inputTokens + outputTokens` and that omits `CacheCreationInputTokens`. Measured on an
identical five-file read:

```
pi      Input 35,992  CacheCreation      0  → guard saw 36,303
claude  Input     50  CacheCreation 44,841  → guard saw    698
```

That is why an executor looked comfortable inside 70,000 while an evaluator died four times
at 100,000 — **all four failures were pi; the one stage that ever completed ran on claude.**
Any comparison of role caps made before this is meaningless.

Now `executor.Result.TokensProcessed()` (`internal/executor/tokens_processed.go`) is the one
definition, and that file carries the per-harness accumulation table. Cache READS are
excluded on purpose: they are the re-sent prefix, so counting them measures conversation
length.

**codex is deliberately NOT changed** — its `inputTokens` is already the whole input with
`cachedInputTokens` a subset split out later. Both its guard sites are commented so nobody
"fixes" it into double-counting.

**Known gap:** on claude the cap is checked at the *result event*, not killed mid-stream,
because `cache_creation_input_tokens` sits outside the usage block the `message_delta`
handler reads and there is no recorded claude-code stream in this tree to verify its
position. The same missing fixtures are why "pi sums / claude assigns" is documented rather
than asserted. **One stream capture per harness closes both.**

Related: cache **writes** were priced at $0 everywhere (no parameter existed). Now billed at
the input rate when undeclared — which OVERSTATES Anthropic reads ~10x, since only 6 models
declare a read rate and none declared a write rate. Declaring real rates for the handful of
models the fleet uses is an open follow-up.

## 4. Migration status: 1 of 16 responsibilities retired

`m-mission-runtime-contract.md` now carries a migration ledger. The honest score is **one**:
quota ration admission, which `mission-control.sh:684` genuinely delegates to
`ailang mission quota --over`.

Two facts that are easy to get wrong:

- `mission iterate` **is** wired (`mission-control.sh:1637` execs it) but gated on
  `AILANG_MISSION_WORK_ITEM`, which no live mission sets. Retiring row 2 is now an opt-in
  decision, not a feasibility question.
- **No shell script consumes `ailang models role`**, despite `models_cmd.go:79` claiming "the
  mission driver reads field 2". Two independently maintained role tables. That divergence
  caused blocker 1 above. Row 4 is the cheapest real win.

~6,000 lines of production shell remain plus 3,085 of shell tests. Scheduling and the host
watchdog are an explicit non-goal.

### Why this matters more than the line count

By the loop's own tags, **55% of iterations 309–348 are `[HARNESS]`** and 348–351 are all
`[HARNESS]`. Over 2026-08-25 → 09-08 the files touched were 883 mission-docs against **16
lines of compiler and stdlib**, and goal distance moved 10 → 13 → 12. The case for migrating
is that the harness is consuming the program, not that the shell stopped working — iteration
354 landed a real fix judged 96 then 100 by an independent evaluator.

## 5. Traps that will cost you time

**A kill point is a lower bound, not a requirement.** I recommended 150,000 reasoning from
101,542 as "the measured requirement with no waste left to reclaim". The actual need was
**136,478** — 101,542 was where the run was *killed*, mid-final-report. 120,000 would have
failed. Apply this to the two remaining briefs' 100,000.

**`--no-extensions` alone silently disqualifies the judge.** Ablation:

```
current flags              bash REFUSED     sprint-evaluator loaded
--no-extensions --approve  bash ran         loaded
--no-extensions alone      bash ran         NOT loaded
```

`--no-extensions` also disables the globally installed `workspace-trust` extension that grants
project trust headlessly, and without trust pi ignores project-local `.agents/skills`. The
rubric IS the evaluator's terminator, so dropping it trades a visible deadlock for an
invisibly unqualified judge.

**`grep` on this machine is ugrep**, which parses a `--`-prefixed filename as an option and
returns 0 matches with an unread error. It silently produced "0 blocks" for all 16 pi
sessions in this session's first measurement pass. Use Python for anything under
`~/.pi/agent/sessions/`.

**Do not use `pgrep -fl` on mission processes.** Their argv carries injected environment
including base64 inbox blobs and tokens. Use `ps -o pid,etime,command` with a `cut`, or query
the runtime DB.

**Run the FULL test suite after touching shared config.** I reported green three times today
after testing a subset, and `models.yml` is read by more packages than seem relevant. One of
those reds was fixed independently by the V1 loop (#1165) while I fixed it again
(`242075de9`) — two agents, one bug, three pieces of work, and a hand-resolved conflict.

**The auto-push hook refuses ahead-AND-behind.** Work strands silently. Merge by hand,
verify with `scripts/mission_decisions.sh --check`, then push.

## 6. Live state left behind

- **Runtime binding** `~/.config/ailang/mission-runtime.toml` points at
  `~/.ailang/state/mission-iteration-canary/runtime-2.sqlite`. Deliberately NOT `/private/tmp`
  — the 09-09 reboot erased the first trial's entire DB, workspaces and bound validator.
- **Bound validator** rebuilt at `/private/tmp/ailang-docs-canary/validate-example` from the
  committed `validate-example.go.txt`. **It will vanish on the next reboot.** Rebuild:
  compile that source from inside the module (it imports an `internal/` package).
- **`docs-canary-guide-review-3` is preserved in terminal `failed` state** in
  `runtime.sqlite`, with its receipt. Do not clear it; it is the four-run evidence.
- **Branches pushed to origin:** `sprint/docs-canary-review`,
  `mission/docs-canary-review-4-authority`. No PRs, no merges. Deletable if unwanted.
- **Refs fetched into the docs clone:** `refs/canary/reliability`, `refs/canary/review`.
- Quota at handover: codex **over ration until 09-19** (73% of a weekly bucket in 2 days,
  attended consumption); ollama recovering; anthropic ok; OpenRouter $0.55 of $2.33 today,
  $92.02 of $100 this month. A fleet restart will run degraded until codex resets.

## 7. Suggested next steps, in order

1. **Capture one claude-code and one pi stream fixture.** Closes the claude in-flight cap gap
   and lets the per-harness accumulation table be asserted instead of documented. Cheapest
   high-value item.
2. **Decide the two remaining briefs' allowance**, measured not assumed, then dispatch them to
   close M4 criterion 2.
3. **Row 4 of the ledger** — make one role table. It has already caused one incident.
4. **Declare real cache read/write rates** for the fleet's models.
5. **F5** — motoko and managed_agents have no thrash guard. Deferred by Mark, not resolved.
6. Only then: the opt-in decision for row 2 (which mission runs `iterate`, under what
   supervision).
