# Mission runtime handover — 2026-09-16 (attended)

Read before touching the mission loops, the binary iteration path, or the uncommitted
working tree. Supersedes nothing wholesale: `mission-runtime-handover-2026-09-14.md` still
holds except where contradicted here.

## 0. There are now TWO live handovers, on different threads

| thread | handover | state |
|---|---|---|
| Codebase simplification for v1.0.0 | `design_docs/planned/HANDOVER-m-v1-simplification-phase3.md` | Phases 0–2 **done** (S1–S4, on origin); Phase 3 (CLI dispatch table) is next and that doc tells you how to run it |
| Mission runtime / binary iteration path | `mission-runtime-handover-2026-09-14.md` + this file | M4 criterion 2 still half-open |

**This file does not repeat the Phase 3 handover.** If you are picking up simplification,
go there; it carries the gate table, rulings D1–D9 and the ordered Phase 3 deliverables.

## 1. The 09-14 handover listed a top three. One of three got done.

1. **Capture one claude-code and one pi stream fixture — DONE** (`44e104518`). It did not
   confirm the per-harness accumulation table; it **falsified the claude row**, in the
   direction that disarms a guard. Detail in §2.
2. **Re-freeze `budget-accounting`'s A1, then dispatch — NOT DONE.** M4 criterion 2 remains
   half-closed (`review-packet` done, `budget-accounting` blocked). Still needs attended
   authority: its frozen criterion asserts cache counters stay separate from the runtime
   sum, which F1 (`134e3ebfe`) made false. It is a re-freeze, not a retry.
3. **Ledger row 4 — one role table — NOT DONE.** Re-verified 2026-09-16: `grep -rn "models
   role" tools/launchd/ scripts/` returns **nothing**. Two independently maintained role
   tables, still. It has now caused three incidents (evaluator lane `7423434b4`, executor
   lane `e9e8ce32e`, both "no admissible rung"), and it remains the cheapest real win.

The session spent itself on the simplification program instead. That is consistent with the
standing position — attended sessions clear harness debt so the loops can restart on a
stable base — but items 2 and 3 are still open and should not be re-discovered from scratch.

## 2. What the stream fixture settled (the one mission-runtime item that closed)

Recorded under the executor's own flags: `internal/executor/claude/testdata/claude_stream_partial.ndjson`.
Full record: `design_docs/verification/mission-iteration-reliability/claude-stream-fixture.md`.

Two documented beliefs died. claude's usage is cumulative **within** a turn and **resets at
every `message_start`**, so a run total is the SUM of the per-turn finals — they reproduce
the result event exactly (50 input / 49,214 cache creation / 315,648 cache read / 706
output). And `cache_creation_input_tokens` is present **in flight**, inside both
`message_start.message.usage` and `message_delta.usage`; the in-flight kill had been
abandoned on the opposite premise.

Compounded, the thrash guard weighed **62 tokens against a cap of 20,000** on a run that
processed 49,970. The claude in-flight cap had never been able to fire. Two further defects
were latent behind that: a thrash kill was banked as `FinishReason: "stop"`, and the kill
path omitted the cache bucket it was killed for.

**The rule this earned:** a row in that table with no recorded stream behind it is a belief,
not a fact. **opencode, codex and motoko are still beliefs.** Inference about a provider's
wire format tends to fail toward *disarming* the guard that depends on it.

**Trap:** capture under the executor's exact argv. The first attempt omitted
`--include-partial-messages` (`claude.go:179`) and produced no `stream_event` lines at all —
a fixture captured without the executor's flags is a fixture of a different program.

## 3. The working tree is NOT clean — and one piece is finished work

`ailang messages health` gained a judgement window. **Complete, green, and uncommitted:**

```
 M cmd/ailang/messages_health.go        (+328/-92)
?? cmd/ailang/messages_health_window.go
?? cmd/ailang/messages_health_window_test.go
 M changelogs/v0.32-current.md          (entry written)
```

Why it exists: the banner read `DEGRADED 15` on a plane that had dispatched everything it
received for 24 hours — the newest of those fifteen was 35 hours old, the oldest fifteen
days, and nothing in the output said so. A counter that cannot reach zero is one readers
learn to skip. `--since` (default `24h`) now sets what the **verdict** judges; older debt
reports as `BACKLOG n` on its own line; `--json` emits the judgement for hooks with the
human banner on stderr.

Verified 2026-09-16 with the working tree as it stands: **full suite `go test ./...` exit 0,
145 packages, no FAIL lines**; `make lint` 0 issues; `go build ./...` clean (`cmd/wasm` and
`gen/main` report "function main is undeclared" under a host build — pre-existing, they are
build-tagged, and `make build` does not include them).

**Fixed today while reviewing it:** the changelog entry had created a **second
`## [Unreleased]` heading** below the S4 content instead of going under the existing one.
Merged into the single heading at line 5.

**Decision needed: commit it.** It is finished work sitting in a shared checkout.

### Also in the tree, deliberately not touched

- `.ailang/state/simplicity/2026-09-15.json` — commit ref and instruction-surface bytes moved.
- `docs/static/benchmarks/os/{history,latest}.json` — modified before this session began;
  unrelated to any of the above.
- **52 untracked sprint JSONs** under `.ailang/state/sprints/`. All predate the 09-14
  gitignore fix (22 May – 11 Sep); 91 others are tracked. They were ignored wholesale until
  `670b03aa4` un-ignored the directory, so they now surface as untracked noise in every
  `git status`. Commit them as history or leave them — but decide, because otherwise every
  future session re-reads 52 lines of `??` looking for its own work.

## 4. Live state

- **All four kill-switch markers present and unchanged** (`mission-control`, `world`,
  `docs`, `motoko`). Loops stay off until Mark says otherwise.
- **codex is over ration until 2026-09-19T09:45Z** — 73.0% used / 38.4% allowed, observed
  09-15. The CLI says it plainly: *"the last reading was ALSO over ration, so refreshing it
  will not unblock Codex."* A fleet restart before then runs degraded.
- **anthropic usage is unreadable** from a shell: `unknown [ENFORCED] — no Anthropic OAuth
  credential`. Expected; launchd has keychain access, shells do not.
- **openrouter**: `$0.00 of $2.33 today, $91.98 of $100.00 left this month`.
- Three releases shipped since 09-14: **v0.38.7, v0.38.8, v0.38.9**. 163 commits.

## 5. Breaking changes that will bite an agent following older instructions

- **`AILANG_MESSAGES_STORE` is REMOVED.** It is now a hard error that names its replacement.
  Verified 2026-09-16:

  ```
  Error: removed environment variable: AILANG_MESSAGES_STORE was removed in v1.0.0
  (M-V1-SIMPLIFY-S3); set AILANG_STORAGE_MESSAGING=gcp
  ```

  Same for `AILANG_COORDINATOR_REMOTE`, `AILANG_CHAINS_READ`, `AILANG_CHAINS_CLOUD`. The one
  plane switch is `AILANG_STORAGE` with per-store overrides. **Fix the export; do not work
  around it.** CLAUDE.md is already updated — a stale copy in a skill or a shell profile is
  the thing to check.
- **`os.Getenv` outside `internal/config` is 0, and `forbidigo` enforces it.** A new env var
  goes in the config Registry or the build fails.

## 6. Traps measured this session

**A launchd installer re-renders from its template, so hand-added plist keys are LOST.**
`bd30614b3`: the installed coordinator plist carried `AILANG_STORAGE=gcp` added by hand; the
template never did. Re-running `install_coordinator.sh` silently brought the rig daemon up on
**local SQLite** — caught only by diffing the installer's own `.bak`. The plane now lives in
the template. Before trusting any re-install, diff `.bak` against the new plist.

**Interactive `grep` is ugrep; scripts get BSD grep.** They differ on BRE `\|`, and `grep -q`
on a pipe under `pipefail` can read false. Verify shell claims under `/bin/bash`.

**Mutation-test your own tests.** Revert the fix and watch the test fail. First attempts
catch about half.

**Run the FULL suite after touching shared config** — carried forward from 09-14 and still
true; `models.yml` and now `internal/config` are read by more packages than seem relevant.

## 7. Suggested next steps, in order

1. **Commit the `messages health` work.** Finished and green; it only needs a decision.
2. **Ledger row 4 — one role table.** Three incidents, cheapest real win, no provider spend
   (which matters while codex is rationed).
3. **Re-freeze `budget-accounting`'s A1** with attended authority, then dispatch to close M4
   criterion 2. Use the backstop cap convention, not a derived budget — and remember a kill
   point is a lower bound, not a requirement.
4. **`claude.go` never calls `Budget.AddCache`** — verified still absent 2026-09-16. The
   claude *cost* budget is blind to cache tokens, the same class as the pi fix in
   `b3ed92633` ("69% of a real bill"). Left out of `44e104518` deliberately: it changes
   cost-kill behaviour in evals, a wider blast radius than a token cap.
5. **Declare real cache read/write rates** for the fleet's models (unchanged from 09-14).
6. **Phase 3 of the simplification program**, per its own handover.
7. **F5** — motoko and managed_agents still have no thrash guard. Deferred by Mark, not
   resolved.
