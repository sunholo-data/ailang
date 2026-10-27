# Cloud Executors: Persist Every Provider's Native Session JSONL

**Status**: Planned
**Target**: v0.54.19
**Priority**: P0 (High)
**Estimated**: 2 days (4h executor banking + 3h writeTaskArtifacts + 4h tests + 3h cloud validation/buffer)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is harness/infrastructure Go — it does not touch language semantics. Scores reflect
what the change delivers for machine analysis of agent runs.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No execution-semantics change; artifact copying is deterministic |
| A2: Replayability | +1 | The native session JSONL is the only complete record of what the agent was told and what tools returned; copying it makes every cloud run inspectable/replayable after the container dies |
| A3: Effect Legibility | 0 | No effect-surface change |
| A4: Explicit Authority | 0 | No capability changes; reads are confined to per-task provider dirs |
| A5: Bounded Verification | 0 | No verification-surface change |
| A6: Safe Concurrency | 0 | Sequential copy inside the single execute-job process |
| A7: Machines First | +1 | Holosun's `holosun-ops.agent_sessions` BigQuery mirror consumes the JSONL directly; lossy human-readable transcripts stop being the only artifact |
| A8: Minimal Syntax | 0 | No language syntax involved |
| A9: Cost Visibility | 0 | Per-call usage was already in metrics.json for the runs that had it; session JSONL adds the raw evidence behind it |
| A10: Composability | +1 | Reuses the existing `ProviderData` channel (motoko precedent) and the existing artifact bucket layout |
| A11: Structured Failure | +1 | Every miss becomes a loud warning naming the paths searched — replacing today's silent skip, which violates CLAUDE.md "No silent fallbacks" |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

Cloud executors never persist the full agent session. In the last 30 days of
ailang-multivac logs (27 Sep – 27 Oct 2026): **308 completed executor runs, 0
`execute-job: session.jsonl written` lines**. By job: pi 161/0, codex 68/0,
codex-go 56/0, pi-go 17/0, go-claude-credit 5/0, motoko 1/0. All 308 wrote
`transcript.txt` + `metrics.json`. The artifacts bucket
(`gs://ailang-multivac-ailang-artifacts/tasks/`) holds 722 task dirs but only 19
`.jsonl` files (8.6 MB total — local runs only).

**Current State — three structural causes in `writeTaskArtifacts`
(cmd/ailang/coordinator_cloud_github.go:505):**

1. **pi runs have no session file on disk at all.** `buildPiArgs`
   (internal/executor/pi/args.go:29-35) passes `--no-session` — pi is invoked in
   ephemeral mode, so nothing exists to copy for the 178 pi/pi-go runs. The
   executor even receives the session id (the `session` event, pi.go:321-324)
   but never persists the file.
2. **codex runs write rollouts that are never searched.** Codex writes
   `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl` (layout verified in
   internal/mission/codex_quota.go:241); cloud runs get an isolated CODEX_HOME
   (cmd/ailang/coordinator_cloud_codexauth.go:89 refuses to run without one).
   `writeTaskArtifacts` only calls `findSessionJSONL` over Claude's
   `CLAUDE_CONFIG_DIR/projects` — the 124 codex/codex-go rollouts are
   structurally invisible to it.
3. **Even the 5 Claude (go-claude-credit) runs wrote nothing, silently.** The
   lookup (coordinator_cloud_github.go:556-578) has four silent-miss paths, all
   violating CLAUDE.md §2 "No silent fallbacks":
   - `findSessionJSONL` returns "" → no log line at all;
   - `os.ReadFile` returns an error → no log line (the `err == nil &&` guard);
   - the file reads as 0 bytes (gcsfuse "legacy staged writes" not yet flushed
     for an append-only file — the exact failure mode the copy exists to work
     around) → no log line;
   - `setupClaudeConfigDir` failed (coordinator_cloud_task.go:268 prints a
     one-line warning) → `CLAUDE_CONFIG_DIR` is unset → Claude writes to the
     default `~/.claude/projects`, which is **never searched as a fallback**.
4. **motoko already solves this and the cloud path ignores it.** The motoko
   executor banks `result.ProviderData["motoko_session_jsonl"]`
   (internal/executor/motoko/motoko.go:598) — and `writeTaskArtifacts` never
   reads `ProviderData` at all, so even the one provider that hands us the path
   on a plate is dropped.

**What is lost:** everything per-call. `transcript.txt` is a turn summary; Cloud
Logging carries `claude-stream:` fragments with tool args truncated at ~4.7k
chars and 30-day `_Default` retention. Tool results, per-call usage and the
model's full output are unrecoverable after the container exits.

**Impact:** Holosun's agent-session archive (BigQuery `holosun-ops.agent_sessions`,
mirroring the artifacts bucket nightly) can count local runs but not cloud runs —
which are ~all fleet runs. Post-hoc debugging of any cloud run is a guessing game
from lossy summaries (CLAUDE.md §1: "Asking what the agent DID and inferring the
rest is how ailang fmt spent two weeks…").

## Goals

**Primary Goal:** Every cloud executor run copies its provider's native session
file(s) to `/artifacts/tasks/{taskID}/session.jsonl`, logging either
`session.jsonl written (N bytes)` or a warning naming every path searched — so
the next 30-day log window shows zero silent misses.

**Success Metrics:**
- `session.jsonl` present in ~100% of new cloud task dirs (vs 19/722 today).
- Every completed non-deterministic run logs exactly one of: "written (N bytes)"
  or "warning: no session.jsonl … searched <paths>".
- At least one test per provider (claude, pi, codex, motoko) asserting the file
  lands at `tasks/{taskID}/session.jsonl`, plus one test asserting the
  missing-file warning names the searched paths.
- Cloud runs become countable by the same Holosun `agent_sessions` views as
  local runs.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Destination stays the flat, already-documented `tasks/{taskID}/session.jsonl` (no `session/<provider>/` subtree) | Downstream readers (Holosun nightly mirror, the path documented in writeTaskArtifacts' doc comment) key on the flat path; a subdir would break them for zero benefit since one task dir holds exactly one run | human | design | high |
| Executors bank their native session path in `Result.ProviderData`; writeTaskArtifacts consumes it, with provider-specific directory search only as fallback | One resolution channel for all providers; reuses the motoko precedent; keeps writeTaskArtifacts from growing a per-provider WalkDir forest | agent | design | med |
| pi persistence switch: `--session-dir <per-task local dir>` when `CLOUD_RUN_EXECUTION` is set, `--no-session` otherwise | Restores pi sessions where we need them (cloud) with zero behavior change for local/eval runs (no `~/.pi` pollution regression, no containment conflict — eval containment is workspace-write-only) | agent | design | low |
| Copy is lossless: no size cap, no truncation of the session JSONL | The entire point is to stop shipping lossy artifacts; a cap would re-create the Cloud Logging problem in GCS | human | design | low |
| Claude lookup keeps its current two search roots (bucket path, then `CLAUDE_CONFIG_DIR`), plus `~/.claude/projects` as a final fallback, and every miss becomes a loud warning | The 5-run miss cause is unproven today; the loud warning + extra fallback turns the next 30-day window into the diagnosis | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Flat `session.jsonl` destination (compatibility with Holosun mirror and existing docs)
- [x] ProviderData banking as the primary resolution channel (motoko key reused as-is)
- [x] `CLOUD_RUN_EXECUTION`-gated pi `--session-dir` (no new env var; reuses the runtime-supplied identity, internal/config/job.go:9)
- [x] Lossless copy, no truncation

## Solution Design

### Overview

Two moves: (1) each executor banks the absolute path of the native session file
it (or its CLI) wrote into `Result.ProviderData`, immediately after the run,
while the container-local file still exists; (2) `writeTaskArtifacts` becomes
provider-aware: it takes the provider name, resolves the session file
(ProviderData first, provider-specific search fallback), copies it to
`tasks/{taskID}/session.jsonl` with `os.WriteFile`, and logs either the byte
count or a warning naming every path searched. The copy is the same
re-write-through-the-mount trick already used for Claude (gcsfuse staged writes
of append-only files do not reliably flush; an explicit `os.WriteFile` does).

### Architecture

**Components:**

1. **Executor-side path banking** (per provider):
   - *pi* (internal/executor/pi/args.go, pi.go): when
     `config.CloudRunExecution() != ""`, replace `--no-session` with
     `--session-dir <dir> --session-id <uuid>` where `<dir> = filepath.Join(os.TempDir(),
     "ailang-pi-sessions", taskID)` and `<uuid>` is generated by the executor
     before launch (V8b: pi creates the session under exactly that id, so the
     file name is known up front, not discovered). After the run, resolve the session file:
     prefer a base name ending `_<uuid>.jsonl` (pi names files
     `<timestamp>_<uuid>.jsonl` and its `session` event carries the same uuid;
     verified against the pi session-format doc — see Verification Log), else
     the newest `*.jsonl` under `<dir>` (the dir is task-scoped: one run per
     dir, so newest is unambiguous). Bank
     `result.ProviderData["pi_session_jsonl"]`. If no file exists, bank nothing —
     the cloud-side warning covers it.
   - *codex* (internal/executor/codex/codex.go): after the run, search
     `config.CodexHome()/sessions/**` for the rollout matching
     `rollout-*<sessionID>*.jsonl`, else the newest
     `rollout-*.jsonl` (isolated per-execution CODEX_HOME contains only this
     run's rollouts). Bank `result.ProviderData["codex_session_jsonl"]`.
   - *claude* (internal/executor/claude/claude.go): after the run, locate
     `{CLAUDE_CONFIG_DIR}/projects/**/<sessionID>.jsonl` (or the default
     `~/.claude/projects` when unset) and bank
     `result.ProviderData["claude_session_jsonl"]`. This is advisory; the cloud
     side keeps its own search as the authoritative path (it already handles
     the symlinked bucket path).
   - *motoko*: no change — already banks
     `ProviderData["motoko_session_jsonl"]` (motoko.go:598).

2. **Cloud-side copy** (cmd/ailang/coordinator_cloud_github.go):
   `writeTaskArtifacts(taskID, provider, result)` — signature gains the provider
   (in scope at the call site, coordinator_cloud.go:104/273). New step 3
   resolves in order:
   1. `ProviderData[<provider>_session_jsonl]` (absolute path);
   2. provider fallback search: claude → existing `findSessionJSONL` over bucket
      path then `CLAUDE_CONFIG_DIR` then `~/.claude/projects`; pi → the per-task
      session dir; codex → `CODEX_HOME/sessions`; motoko → none (banking only);
   3. copy with `os.ReadFile` + `os.WriteFile` (0644) to
      `tasks/{taskID}/session.jsonl`; on read-error/empty-result retry the read
      up to 3× with 2s sleep (gcsfuse staged-write flush), then warn.
   Logging contract (the ask, verbatim): on success
   `execute-job: session.jsonl written (N bytes)` (unchanged format, so the
   existing log-based counting keeps working); on miss
   `execute-job: warning: no session.jsonl for provider %q (session %q): searched
   <path1>, <path2>, …`. The deterministic-bump path (SessionID sentinel
   `deterministic-bump`, coordinator_cloud_task.go:241) skips the session step
   entirely — no provider ran, so no warning is signal, only noise.

3. **Metrics sidecar** (same function): `metrics.json` gains
   `"session_provider": <provider>` and `"session_jsonl_bytes": <N>` so the
   nightly Holosun mirror can distinguish JSONL schemas without opening the
   file.

### Implementation Plan

**Phase 1: Executor-side banking** (~4 hours)
- [ ] pi: swap `--no-session` → `--session-dir` under `CLOUD_RUN_EXECUTION`;
      add `resolvePiSessionFile(dir, sessionID)` + bank in ProviderData
      (internal/executor/pi/args.go, pi.go, new session.go helper)
- [ ] codex: add `resolveCodexSessionFile(codexHome, sessionID)` + bank
      (internal/executor/codex/codex.go, new session.go helper)
- [ ] claude: bank `claude_session_jsonl` (internal/executor/claude/claude.go)
- [ ] Unit tests: pi resolve (suffix match + newest fallback + miss), codex
      resolve (rollout naming + newest fallback), args test asserting
      `--session-dir` present / `--no-session` absent in cloud mode and the
      reverse locally (extends pi_args_extensions_test.go patterns)

**Phase 2: Cloud-side copy + logging** (~3 hours)
- [ ] `writeTaskArtifacts(taskID, provider, result)` signature change + call
      site (cmd/ailang/coordinator_cloud.go:273)
- [ ] Resolution ladder + retry-empty-read + the two log lines
- [ ] `~/.claude/projects` final fallback for the setup-failure path
- [ ] Deterministic-bump skip
- [ ] `metrics.json` sidecar fields

**Phase 3: Tests + validation** (~4 hours)
- [ ] One `writeTaskArtifacts` test per provider (below)
- [ ] Missing-file warning test asserting the searched paths appear in output
- [ ] `make test` (cmd/ailang + internal/executor) and `make lint`
- [ ] Post-merge: dispatch one canary task per variant (pi, codex, claude) and
      verify the artifact + log line in the bucket/Cloud Logging

### Files to Modify/Create

**New files:**
- `internal/executor/pi/session.go` (~60 LOC) — session-dir choice + file resolve
- `internal/executor/codex/session.go` (~60 LOC) — rollout resolve
- `cmd/ailang/coordinator_cloud_session_test.go` (~250 LOC) — per-provider
  writeTaskArtifacts tests + warning test

**Modified files:**
- `internal/executor/pi/args.go` (+15/-3 LOC) — `--session-dir` under cloud
- `internal/executor/pi/pi.go` (+10 LOC) — bank ProviderData path
- `internal/executor/codex/codex.go` (+12 LOC) — bank ProviderData path
- `internal/executor/claude/claude.go` (+15 LOC) — bank ProviderData path
- `cmd/ailang/coordinator_cloud_github.go` (+80/-20 LOC) — provider-aware
  session copy, logging, retry, fallbacks
- `cmd/ailang/coordinator_cloud.go` (+2/-1 LOC) — pass provider
- `internal/executor/pi/session_test.go`, `internal/executor/codex/session_test.go` (~120 LOC each)

**Explicitly not modified:** `internal/config/` (no new env var —
`CLOUD_RUN_EXECUTION` and `CODEX_HOME` accessors already exist), Terraform, the
artifact bucket layout.

## Examples

### Example 1: pi run, today vs after

**Before** (pi, 161/161 runs, last 30 days):
```
pi --mode json --model anthropic/... --no-session -p "<directive>"
# container exits; ~/.pi/agent/sessions never had a file; artifacts:
#   tasks/{id}/transcript.txt   (turn summary)
#   tasks/{id}/metrics.json
# NO session.jsonl; no log line about it either way
```

**After:**
```
pi --mode json --model anthropic/... --session-dir /tmp/ailang-pi-sessions/task-xyz --session-id 3f2a… -p "<directive>"
# executor: result.ProviderData["pi_session_jsonl"] = "/tmp/ailang-pi-sessions/task-xyz/2026-10-27T09-46-35-666Z_3f2a….jsonl"
# writeTaskArtifacts:
execute-job: session.jsonl written (412337 bytes)
# artifacts: tasks/{id}/{transcript.txt, metrics.json, session.jsonl}
```

Miss case (a provider that failed before writing anything):
```
execute-job: warning: no session.jsonl for provider "pi" (session "3f2a…"): searched /tmp/ailang-pi-sessions/task-xyz
```

### Example 2: codex run

```
# codex wrote $CODEX_HOME/sessions/2026/10/27/rollout-2026-10-27T09-41-02-<uuid>.jsonl
execute-job: session.jsonl written (88121 bytes)
```

### Example 3: metrics.json sidecar

**Before:** `{"task_id": …, "session_id": …, …}`
**After:** `{"task_id": …, "session_id": …, "session_provider": "pi", "session_jsonl_bytes": 412337, …}`

## Verification Log

No language claims (pure Go infra), so no `ailang check` rows; every premise
below was verified by reading the code/doc cited, including the
negative-existence claims:

| # | Claim | Evidence |
|---|-------|----------|
| V1 | pi is invoked with `--no-session`, so no session file exists | internal/executor/pi/args.go:29-35 (`--no-session` in the base argv, comment: "ephemeral run") |
| V2 | `writeTaskArtifacts` searches only Claude locations | cmd/ailang/coordinator_cloud_github.go:556-561 — the only session lookup is `findSessionJSONL` over `taskClaudeArtifactDir` / `config.ClaudeConfigDir()`; no pi/codex path appears (grep "session" in the function) |
| V3 | Miss/read-error/empty-read are all silent today | coordinator_cloud_github.go:556-578 — no log line in any of the three miss branches; the only warning is a `session.jsonl` *write* failure |
| V4 | A setup-failure Claude run writes to `~/.claude/projects`, which is never searched | coordinator_cloud_task.go:268-272 (env var only set on setup success); coordinator_cloud_github.go:558-561 (only `config.ClaudeConfigDir()` and the bucket path searched; `~/.claude` appears nowhere in the lookup) |
| V5 | motoko already banks `ProviderData["motoko_session_jsonl"]`; writeTaskArtifacts never reads ProviderData | internal/executor/motoko/motoko.go:598; `writeTaskArtifacts` body contains no `ProviderData` reference |
| V6 | Codex rollout layout is `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl` | internal/mission/codex_quota.go:241 (`filepath.Join(codexHome, "sessions", …Format("2006/01/02"))`) + rollout-/`.jsonl` filters at :251; cloud CODEX_HOME is isolated (cmd/ailang/coordinator_cloud_codexauth.go:88-90 refuses unset) |
| V7 | pi session files live as `<agentDir>/sessions/--<path>--/<timestamp>_<uuid>.jsonl`; the `session` event carries the same uuid | pi session-format doc (`docs/session-format.md` in @earendil-works/pi-coding-agent@0.85.1): file layout line 8, header `{"type":"session","version":3,"id":"uuid"}` line 200; executor parses that id (internal/executor/pi/events.go:93, pi.go:321-324). Suffix-match is therefore the primary resolve; newest-in-dir is the fallback if the event id and file uuid ever diverge |
| V8 | pi supports `--session-dir <dir>` (and `--no-session`) as CLI flags | `pi --help` on this machine (v0.85.1): "--session-dir <dir>  Directory for session storage and lookup" |
| V8b | Headless pi persists a session with `--session-dir` + `--session-id`, flat in the dir, named `<ISO-timestamp>_<id>.jsonl` | MEASURED 2026-10-27 (attended review, pi v0.85.1): `pi --mode json --model openrouter/z-ai/glm-5.3 --session-dir D --session-id 11111111-2222-4333-8444-555555555555 -p "…"` wrote `D/2026-10-27T09-46-35-666Z_11111111-2222-4333-8444-555555555555.jsonl` (1313 B); stderr warns "No project session found with id …; creating a new session with that id"; the `session` event `id` equals the passed id. No `--<path>--` subdir under an explicit `--session-dir` |
| V9 | `CLOUD_RUN_EXECUTION` is runtime-supplied on Cloud Run job executions and readable without new config | internal/config/job.go:9,67 ("Cloud Run execution identity supplied by the runtime"), accessor :281-282; already required for codex credential ownership (coordinator_cloud_codexauth.go:94-96) |
| V10 | The provider name is in scope at the writeTaskArtifacts call site | cmd/ailang/coordinator_cloud.go:104 (`var provider string`), resolved :204, call :273 |
| V11 | The deterministic-bump path fabricates a Result with sentinel SessionID | coordinator_cloud_task.go:241 (`SessionID: "deterministic-bump"`) |
| V12 | No existing test covers writeTaskArtifacts' session copy | grep over cmd/ailang/*_test.go: only coordinator_cloud_claudecfg_test.go:93-96 touches `findSessionJSONL`, nothing drives `writeTaskArtifacts` |
| V13 | The go-claude-credit 5-run miss root cause is UNPROVEN | The four candidate causes (V3, V4, plus SessionID-empty if the final result message fails to parse — claude.go:549 takes `finalResult.SessionID` from the parsed result message) are all made loud/observable by this design; the next 30-day window is the diagnosis. The design deliberately does not assert which one fired |

The 308/0, 722/19 and log-fragment numbers come from the task brief's Cloud
Logging / bucket survey (27 Sep – 27 Oct 2026) and are cited as reported, not
re-measured here.

## Success Criteria

- [ ] pi run (or simulated pi run): `writeTaskArtifacts` copies the banked
      ProviderData path to `tasks/{taskID}/session.jsonl` —
      `TestWriteTaskArtifacts_PiSessionCopied`
- [ ] codex run: rollout resolved from `CODEX_HOME/sessions` and copied —
      `TestWriteTaskArtifacts_CodexSessionCopied`
- [ ] claude run: existing search still lands the file (regression: the
      claudecfg test's bucket-path lookup keeps passing) —
      `TestWriteTaskArtifacts_ClaudeSessionCopied`
- [ ] motoko run: `ProviderData["motoko_session_jsonl"]` copied —
      `TestWriteTaskArtifacts_MotokoSessionCopied`
- [ ] Miss case: warning printed naming every searched path —
      `TestWriteTaskArtifacts_SessionMissingWarns`
- [ ] Success log line keeps the exact format
      `execute-job: session.jsonl written (N bytes)`
- [ ] All tests passing (`make test`, `make lint`)
- [ ] One canary cloud run per variant (pi, codex, claude) shows the artifact in
      `gs://…/tasks/{id}/session.jsonl` and the log line in Cloud Logging
- [ ] Documentation updated (writeTaskArtifacts' doc comment lists all
      providers; docs/docs/guides/coordinator.md artifact section if it exists)

## Testing Strategy

**Unit tests:**
- pi resolve helper: exact-uuid suffix match, newest-fallback, empty dir
- codex resolve helper: rollout matched by session id, newest-fallback, empty
- buildPiArgs: cloud mode ⇒ `--session-dir`, no `--no-session`; local mode ⇒
  unchanged argv (regression for eval/local pollution)
- writeTaskArtifacts per provider (table-driven): fixture session file at the
  provider's native location (or ProviderData path), assert
  `tasks/{taskID}/session.jsonl` contents byte-equal + success log; miss case
  asserts warning lists the searched paths
- Retry behavior: session file readable but 0 bytes on first read (simulated via
  a helper seam) ⇒ retry succeeds ⇒ written; still empty ⇒ warning

**Integration tests:**
- The existing `TestSetupClaudeConfigDir_OnlySessionLogsReachArtifacts` must
  keep passing (it pins the Claude bucket path contract).

**Manual testing:**
- Canary dispatch per variant after merge; check bucket object + Cloud Logging
  line; for pi confirm a session file actually appears under `--session-dir`
  in `-p --mode json` mode (V7 covers naming, not that headless mode persists —
  the resolve fallback makes a mismatch visible rather than fatal).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Session-dir location for pi (TempDir vs a `/home/.../` base mirroring
  `cloudClaudeConfigBase()`) — agent may choose; must stay off `/artifacts` and
  out of the workspace
- Whether the claude executor's banking (advisory path) also verifies the file
  is non-empty at bank time — agent may choose
- Copy mechanics for very large sessions (`io.Copy` streaming instead of
  `os.ReadFile`+`os.WriteFile` above, say, 64 MB) — agent may choose; the ask's
  `os.WriteFile` semantics (explicit re-write ⇒ guaranteed flush) must hold
  either way
- Whether the miss warning also carries `result.EnvNamesDigest`-style
  diagnostics for the claude setup-failure path — agent may choose
- Exact metrics.json field names beyond `session_provider` /
  `session_jsonl_bytes` — agent may choose

## Non-Goals

**Not attempted in this feature:**
- Importing cloud session JSONLs into the observatory or `ailang chains` —
  existing tooling (`ailang chains import-motoko`, OpenRouter traces) covers
  per-provider analysis; the archive just has to exist first
- Streaming sessions to GCS *during* the run (the F-H6-1 symlink does this for
  Claude already; end-of-run copy is sufficient and simpler for pi/codex)
- Backfilling the 308 lost runs — unrecoverable; the fix is forward-looking
- Changing pi/codex local or eval-lane behavior (`--no-session` stays outside
  cloud runs)
- Extending Cloud Logging retention or fixing the `claude-stream:` truncation —
  separate concern; the artifact copy removes the dependency on those logs

## Timeline

**Week 1** (11 hours):
- Phase 1: executor banking + helper tests (4h)
- Phase 2: writeTaskArtifacts + logging + call site (3h)
- Phase 3: per-provider tests, lint, canary dispatch + bucket verification (4h)

**Total: ~11 hours across 1 week** (2× the raw 5.5h estimate)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| pi does not persist a session in `-p --mode json` headless mode even with `--session-dir` | Low (measured: it does, V8b) | Newest-in-dir fallback + loud warning; canary run verifies before the fleet depends on it; if pi truly cannot persist headlessly, the warning names the dir and the fix moves to pi's harness (motoko lane pattern) |
| pi file uuid ≠ `session`-event id (suffix match fails) | Low | Newest-in-dir fallback; dir is task-scoped so newest is unambiguous |
| gcsfuse read of a staged append file keeps returning empty even after retries | Medium | Retry 3×/2s; on final failure the warning names the path and byte count 0 — the next window quantifies it; local config-dir symlink and bucket path are the same inode, so this is observable, not fixable, at this layer |
| Session JSONLs are large (100+ MB pathological runs) — memory blowup in `os.ReadFile` | Low | Deferred decision: streaming copy above a threshold |
| Extra log noise from miss warnings on genuinely sessionless runs (e.g. executor died at startup) | Low | That noise is exactly the signal the 30-day survey was missing; deterministic-bump path is exempted |
| `--session-dir` interacts with eval containment (workspace-write-only) | Low | Gate is `CLOUD_RUN_EXECUTION != ""`; eval runs never see the flag (containment and cloud lanes are disjoint) |

## Related Documents

**Implemented (may inform design):**
- (none found by search)

**Planned (check for overlap):**
- (none found by search)

**Read during authoring (premise sources):**
- `cmd/ailang/coordinator_cloud_github.go` — writeTaskArtifacts, findSessionJSONL
- `cmd/ailang/coordinator_cloud_claudecfg.go` + test — F-H6-1 bucket-path contract
- `internal/executor/pi/args.go`, `internal/executor/codex/codex.go`,
  `internal/executor/claude/claude.go`, `internal/executor/motoko/motoko.go`
- `internal/mission/codex_quota.go` — codex rollout layout
- pi session-format doc (`@earendil-works/pi-coding-agent` v0.85.1)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- CLAUDE.md §1 ("What was the agent actually told?"), §2 (No silent fallbacks)
- Holosun agent-session archive: BigQuery `holosun-ops.agent_sessions`, nightly
  mirror of `gs://ailang-multivac-ailang-artifacts/tasks/`
- Task brief: 30-day log survey 27 Sep – 27 Oct 2026 (308 runs / 0 session
  writes / 19 .jsonl in 722 task dirs)

## Future Work

- Import cloud `session.jsonl` into the observatory for the same
  `ailang chains` tooling local runs get
- Per-provider schema detection in the Holosun mirror driven by
  `metrics.json:session_provider` (landed here as a sidecar field)
- Extend the same banking+copy contract to `opencode` / `managed_agents` /
  `gemini_agents` when those lanes ship cloud volume

---

**Document created**: 2026-10-27
**Last updated**: 2026-10-27
