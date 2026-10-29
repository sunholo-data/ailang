# Sprint plan: quota opencode is not a pool

Ticket: `quota:opencode-reported-as-a-pool` (attended filing 2026-10-01). Fleet iteration 52; planner only, 2026-10-21.
Authority: `design_docs/fleet-mission.md`, **D-FLEET-12 item 2**, Mark's pre-authorization: “remove opencode from the quota report; count opencode-routed models under their backend bucket.” No further ruling is needed. D1–D9 below are the controller's fixed design, not open questions.
Deliverables here are this plan and its sprint JSON only. No implementation, git writes, handoff dispatch, or changes to the unrelated active shadow sprint; the controller commits these artifacts and routes execution/evaluation.

## Scope and estimate

One milestone **M1_QUOTA_OPENCODE_NOT_A_POOL**, one working day, approximately 60–90 minutes executor work plus separate controller verification. Estimate **300 LOC**: 80 implementation/comments, 216 tests, 4 changelog; includes buffer. This is a scoped estimate, not a measured velocity claim. Recent fleet sprint 51 provides the one-milestone pattern, not a comparable timing benchmark.
Dependencies: established controller measurements M1–M8, existing ledger/CLI test helpers, and fixed D1–D9. No `.ail` code or showcase module is needed; generated Go fixtures exercise the CLI behavior.
Registry reuse: **none**. Existing Go mission ledger/admission and CLI infrastructure are reused; this is no package-like AILANG capability, so registry package searches are inapplicable. No new package, dependency, or script.

## Verification: established controller measurements

These are controller measurements made **outside any sandbox**, using a HEAD-built binary at `origin/dev` **1579ef50c**; they are accepted evidence, not planner reruns.

- **M1 — reproduced:** scratch HOME with a copy of the rig's real `quota-ledger.json`, no journal, prints `opencode 5h not-paced 0 tok ...`, `opencode long capacity-unknown 0 tok consumed ...`, two `quota: opencode/5h|long has no known capacity — UNRATIONED ...` lines and two locally derived window-boundary lines. Cached opencode 5h/long rows have tokens=0, stages=0, window_start=`0001-01-01T00:00:00Z`. No rig journal segment has opencode; segment buckets: anthropic 12, codex 10, ollama 1.
- **M2 — control:** an empty scratch home prints zero opencode lines; the phantom rows come from the cache.
- **M3 — persistence:** `quota_ledger.go` `rebuild()` (~528) loads the cache, clears every WindowStart, folds journal events, and retains cached rows; `expire()` skips zero-WindowStart rows. A once-recorded/seeded bucket therefore persists indefinitely.
- **M4 — write seam:** one scratch journal line `<unix>\topencode\t5\t1` creates 5-token opencode rows. The sole production writer is `chains_post.go` `recordQuotaSpend` → `AppendSpend`; `observatory.CanonicalQuotaBucket` (~222) lowercases unknown values unchanged, including bare and backend-qualified opencode. Both upstream files are outside this ticket's scope.
- **M5 — admission:** Candidate.Transport is registry provider (`dispatch/resolve.go:74`). Registry providers/counts: anthropic 18, chatgpt 1, google 18, lyceum 3, ollama 36, ollama-rig 12, openai 25, openrouter 56, zai 1; zero opencode. Executors outside claude/codex/pi are skipped with “executor budget contract is not admitted by role-run”. The opencode acceptance arm (~77) is unreachable; admitted models already use registry backend providers.
- **M6 — shell seam:** `lane-probe.sh:194` accepts opencode in its awk alternation. `mission quota --over` only emits Over() ledger verdicts; zero-capacity rows never qualify, so this alternation entry is dead code.
- **M7 — CLI:** flag help already advertises codex, anthropic, openrouter, ollama; `--bucket opencode` currently succeeds when cached. Text/JSON filtering (~148–170) removes only codex/ollama ledger rows.
- **M8 — base gates:** `go test ./internal/mission/` ok (26.6s); `go test ./cmd/ailang/ -run 'MissionQuota|Quota' -count=1` ok; `gofmt -l internal/mission cmd/ailang` empty. Required changelog/file-size gates are green at base. Clean-env routing suite rc 0 (108 PASS), and anthropic/codex/ollama admission shell suites rc 0. Live mission-shell routing rc 1 is caused by leaked `MISSION_PLANNER_ALLOWLIST`; always use `env -i`. Base sizes: quota_ledger.go 713, quota_ledger_test.go 610, mission_quota_cmd.go 277, mission_quota_cmd_test.go 137, admission.go 98 lines.

## Fixed design D1–D9

- **D1:** add `internal/mission/quota_bucket.go` with small exported `QuotaBucketFor(raw string) (bucket string, err error)`. Define one named list/constant set containing exactly **anthropic, codex, ollama, openrouter**, reused for bucket checks/mapping in D3–D6 (the shell alternation mirrors these four names). Recognize case-insensitive opencode harness names: bare `opencode`, `opencode:<backend>`, `opencode/<backend>`. For qualified names whose backend begins with a listed bucket, return that bucket; e.g. `opencode:openrouter/z-ai/glm-5.3` → openrouter, `opencode:ollama/glm-5.3:cloud` → ollama. Bare or unsupported-backend opencode returns an error saying **“opencode is a harness, not a quota bucket; record the backend bucket instead”**. Return every other raw value unchanged with nil error; do not normalize unrelated names or broaden this fix.
- **D2:** `AppendSpend` calls QuotaBucketFor before any journal write: mapped values are written under backend, unmappable opencode errors without changing the journal. `chains_post.go` already reports errors to stderr and continues the iteration; do not edit it.
- **D3:** rebuild journal fold calls QuotaBucketFor: map qualified harness events to backend; skip unmappable harness events and increment the existing `skipped` return. Drop **all opencode harness cache rows**, mapped or unmappable, rather than migrating cached spend; the journal remains spend authority. No other cache pruning: real-bucket capacity and provider-reset rows survive exactly as today.
- **D4:** `missionQuotaWithPaths` rejects `--bucket` naming **any** opencode harness value, including mappable qualified/case variants, with the same harness diagnostic, before reporting. A mapped helper result is not permission to select the harness name. Flag help stays unchanged or uses D1's four names.
- **D5:** remove opencode from admission's accepted-bucket condition (~77), using D1's list for the remaining check. Anthropic/openrouter behavior stays unchanged; preserve earlier codex and ollama switch arms.
- **D6:** at `tools/launchd/lib/lane-probe.sh:194`, only remove `|opencode` from the awk alternation. Preserve bash 3.2/BSD awk compatibility. No other driver edit; four remaining names are D1's buckets.
- **D7:** add the behavioral arms and mutation checks below; existing tests/shell suites stay unchanged. Leave `TestMissionQuotaAdvertisedBucketsAreAllSelectable` and its historical `(have: [anthropic ollama opencode])` comment intact.
- **D8:** fragment `changelogs/unreleased/2026-10-21-quota-opencode-not-a-pool.md`, 2–5 lines total, exact first line: `### Fixed — mission quota no longer reports the opencode harness as a quota pool; opencode-routed spend counts under its backend bucket`.
- **D9:** exclude `internal/observatory/**`, `cmd/ailang/chains_post.go`, `internal/modelreg/**`, real `~/.ailang/state` files, `tools/launchd/mission-control.sh`, every skill file, and unrelated changes. Retain only the named implementation scope below; follow-up is recorded without filing a ticket.

## One milestone: M1_QUOTA_OPENCODE_NOT_A_POOL

Files to create/update during execution:
- `internal/mission/quota_bucket.go`, `internal/mission/quota_bucket_test.go`.
- `internal/mission/quota_ledger.go`, new `internal/mission/quota_ledger_opencode_test.go` (companion to `quota_ledger_test.go`; do not grow the existing 610-line file).
- `internal/mission/admission.go`, `internal/mission/admission_test.go`.
- `cmd/ailang/mission_quota_cmd.go`, `cmd/ailang/mission_quota_cmd_test.go`.
- `tools/launchd/lib/lane-probe.sh` (one alternation edit only).
- `changelogs/unreleased/2026-10-21-quota-opencode-not-a-pool.md`.

Day 1 sequence: 20 min generated fixtures/red arms → 25 min helper/write/rebuild/admission/CLI edits → 10 min shell edit/fragment → 25 min focused gates and mutation drills → 10 min buffer/evidence handback. No milestone dependencies or dummy milestone. Keep quota_ledger.go below 800 lines; the new helper and test companion contain growth.

### Required test arms (D7)

- **Helper table:** `opencode` → error; `OpenCode` → error; `opencode:openrouter/z-ai/glm-5.3` → openrouter; `opencode/ollama/glm-5.3:cloud` → ollama; `opencode:anthropic/claude-sonnet-5-5` → anthropic; `opencode:openai/gpt-5.5` → error; `anthropic` → anthropic; `codex` → codex; `unlabeled` → unlabeled. Include case-insensitive qualified and codex backend coverage using the same list.
- **Ledger (a):** generate an M1-shaped ledger.json in `t.TempDir()` (never read the real home): anthropic 5h/long rows with tokens, opencode 5h/long tokens=0 and zero WindowStart, plus an anthropic capacity-only zero-token control row. `LoadLedger` removes opencode and preserves anthropic rows, capacity, and existing reset fields. Make the capacity assertion independently observable even when the harness-removal assertion fails; do not turn this into general zero-token pruning.
- **Ledger (b):** one valid-timestamp `opencode` journal event → rebuild skipped count exactly 1, no opencode row.
- **Ledger (c):** `opencode:openrouter/x` journal event, 7 tokens → openrouter 5h/long rows with 7 tokens each, no opencode row.
- **Append (d):** `AppendSpend("opencode", ...)` errors; compare journal SHA before/after, or assert it remains absent. Cover pre-existing and absent journal states so a write-before-error is detected.
- **Append (e):** `AppendSpend("opencode:ollama/m:cloud", 3, ...)` succeeds; parsed journal line names ollama and 3 tokens. Each ledger/append arm uses a fresh `t.TempDir()` home and fixed live-window time.
- **CLI:** `missionQuotaWithPaths([]string{"--bucket", "opencode"}, ...)` returns the harness error; table-cover qualified/mappable and case variants too. Existing advertised-buckets test stays green unchanged.
- **Text report:** generated M1-shaped cache → output contains no `opencode` substring. Neutralize credentials, CODEX_HOME and executable PATH following existing test patterns; fixtures must not read real quota or start provider processes.
- **Admission:** directly construct Candidate with Transport `opencode` and an executor/wire model reaching the ledger branch; inject observers. Check `unsupported quota bucket` error, as for unknown transport.

### Arm → mutation table

Run baseline green, each mutation separately, and restored green. Mutate **disposable copies**: `cp` original to a `mktemp` path and restore by `cp` (or run copied sources through a Go overlay); never `git checkout <file>`. Record rc and the intended failed assertion; compile errors are not evidence of sensitivity.

| Arm | Mutation that must turn its assertion RED |
|---|---|
| Helper bare-opencode row | Return `("opencode", nil)` for bare opencode. |
| Helper mapping rows | Drop `/` separator support; map only `:`. |
| Ledger (a), phantom cache removal | Remove cached-row drop in rebuild. |
| Ledger (b) | Fold an unmappable journal entry instead of skipping it. |
| Ledger (c) | Skip mapped journal entries instead of folding under backend. |
| Append (d) | Write journal before checking QuotaBucketFor. |
| CLI harness rejection | Remove the `--bucket` harness check. |
| Text report | Revert D3 entirely, both cache removal and journal mapping/skipping. |
| Admission unsupported transport | Restore the equivalent of `&& bucket != "opencode"` in accepted-bucket condition. |
| Capacity-survives control in (a) | Must stay **GREEN under the D3 mutation set above**, and go **RED if cache dropping is widened to every zero-token row**: this is the over-pruning guard. |

No new shell arm is required for lane-probe: per M6 the alternation entry is dead code; the existing clean-env launchd suites below must stay green unchanged.

## Acceptance gates and evidence handback

All D1–D9 behaviors, required test arms, independent capacity control, mutation pairs and bounded file list are acceptance criteria. Run foreground commands only. Preserve every rc and one-line result in execution notes. **Any socket-touching command result is UNINFORMATIVE UNDER SANDBOX**, regardless of rc; controller reruns outside the sandbox decide it.

Executor gates (M8 green baseline):
- `go test ./internal/mission/` → rc 0 / ok.
- `go test ./cmd/ailang/ -run 'MissionQuota|Quota' -count=1` → rc 0 / ok.
- `gofmt -l internal/mission cmd/ailang` → rc 0 and empty output.
- `make check-changelog` and `make check-file-sizes` → each rc 0.
- `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR /bin/bash tools/launchd/test_mission_routing.sh` → rc 0 (base 108 PASS).
- Same `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR /bin/bash` prefix with `tools/launchd/test_anthropic_quota_admission.sh`, `tools/launchd/test_codex_quota_admission.sh`, `tools/launchd/test_ollama_quota_admission.sh` → each rc 0. Do not leak `MISSION_PLANNER_ALLOWLIST` from the mission shell.

**Controller alone, outside the sandbox:** run `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR make test-launchd-drivers`; run HEAD-vs-new binary comparisons of `ailang mission quota` and `ailang mission quota --over` on separate **SCRATCH copies** of the rig's real ledger.json and journal segments in temp HOMEs. Compare stdout, stderr and rc under the same provider environment; **only vanished opencode lines are allowed differences** (M6 predicts unchanged --over). Never point either run at real state. This integration step is separate from executor gates; do not prescribe full repository tests or broad build commands.
Planner validation: `jq -e . <sprint-json>` and the existing sprint validator, with rc/summary recorded in JSON notes. One milestone is intentional: if its only complaint is minimum two, record that exception and add no dummy milestone.

## Risks and follow-ups

Risk is bounded but state-sensitive: deleting cached harness rows must not discard legitimate capacities/resets; the independent zero-token capacity control detects over-pruning. Rejecting unmappable journal history increments existing skipped accounting; preserve loud diagnostics rather than silently attributing spend. Legacy mapped journals reattribute to backend exactly once per rebuild, with existing journal/window semantics unchanged. Provider tests may bind sockets; sandbox outcomes cannot replace controller M8 evidence. Live mission environment leakage must be removed via `env -i`.
Follow-up (**not ticketed**): `observatory.CanonicalQuotaBucket` still returns bare `opencode` unchanged; **V1 owns it**. This fix protects AppendSpend without editing that out-of-scope producer. No further routing, quota margin, provider inventory, or pruning work is included.
