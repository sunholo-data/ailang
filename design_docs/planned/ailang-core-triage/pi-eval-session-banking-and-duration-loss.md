# pi eval runs: rejected runs bank no session, no real duration, and no importer

- **Date**: 2026-11-05
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `ailang-pi-sessions`, `non-agentic`, `import-motoko`, `ImportPiSession`, `duration` in `internal/executor/pi`, `bufio` across `cmd/ailang/eval_suite*` and `internal/eval_harness`; browsed `design_docs/planned/` and `design_docs/implemented/v0_54_19/`
- **Estimate**: (b) alone would be ~8 lines in `internal/eval_harness/agent_runner_multi.go`; the report as a whole spans ≥4 files plus a new ~300-line importer — design-doc

## Why

The report is one incident with four asks, and they share one root shape: **the
pi path was built cloud-first and the eval path only ever learned the motoko
tricks.** Verified in code:

- **(a) no pi session exists to bank.** `buildPiArgsWithSession`
  (`internal/executor/pi/args.go`, the `CloudRunExecution()` branch) passes
  `--no-session` for any non-cloud run, and `nativesession.Bank` is only
  called under `config.CloudRunExecution() != ""` (`internal/executor/pi/pi.go`,
  the deferred call in `executeStreaming`). So a local eval run leaves nothing
  under `$TMPDIR/ailang-pi-sessions`. Even if it did, the harness extracts only
  `motoko_session_jsonl` from `ProviderData`
  (`internal/eval_harness/agent_runner_multi.go`, `sessionJSONLPath := ...`),
  and `cmd/ailang/eval_benchmark_agent.go` gates import on
  `executorName == "motoko"` in both the success and failed-run paths. The
  plumbing exists end-to-end for exactly one provider.
- **(b) duration is dropped at the rejection branch, not by the executor.** pi
  sets `DurationMS` correctly (`internal/executor/pi/pi.go:519,579`). The
  executor-failure branch in `agent_runner_multi.go` carefully carries spend
  (`diag.DurationMS = result.DurationMS`, plus tokens/cost) after the
  2026-10-01 under-count fix — but the *adjacent* non-agentic branch
  (`return diagnosticsOnly(), fmt.Errorf("... non-agentic result: ...")`)
  returns bare `diagnosticsOnly()`, whose measurement fields are deliberately
  zero. The same "it still spent real time" argument applies: the gateway
  ledger shows 487–773 s, the banked row says 0 ms. This is the one piece that
  is a genuine direct-fix (~8 lines, 1 file, mirroring the branch above it) —
  it could land ahead of the doc.
- **(c) no pi importer.** `ImportMotokoSession`
  (`internal/observatory/importer_motoko.go`, 357 lines) +
  `chains import-motoko` (`cmd/ailang/chains_import.go`) parse motoko's session
  JSONL into chains. A pi sibling is a new public CLI surface plus a new
  file-format parser — unambiguously design-doc territory, and it only pays
  off once (a) stops discarding the sessions.
- **Minor (buffering):** no `bufio` writer exists in the eval-suite output
  path (`cmd/ailang/eval_suite*.go` — only `run_policy_stderr_guard.go` uses
  one), so the reported block-buffering mechanism was not located in a quick
  scan; Go's `os.Stdout` writes are unbuffered. The symptom is credible but the
  mechanism needs one session of tracing before it can be sized. Fold it into
  the same doc as a diagnostic sub-task rather than guessing a fix.

**Adjacent prior art, not a duplicate:** `design_docs/implemented/v0_54_19/m-cloud-native-session-jsonl.md`
solved the sibling problem for *cloud executor* artifacts (it is where
`nativesession` and `PiDir` come from) but explicitly targets
`writeTaskArtifacts` in the cloud coordinator; the eval-harness path —
`--no-session` for local runs, motoko-only extraction/import, and the
diagnosticsOnly spend drop — is outside its scope. The design doc should extend
its pattern ("every provider's native session is evidence; no silent misses")
to eval runs, including rejected ones.

## Suggested doc scope

1. Session evidence for eval pi runs: choose between always-on
   `--session-dir` (pollution/isolation trade-off — the reason `--no-session`
   exists), banking only the final assistant message, or a temp-dir session
   with harness-side cleanup; bank it for rejected runs too
   (`diagnosticsOnly` already carries `SessionJSONLPath` — the extract just
   needs to stop being motoko-only).
2. Carry spend (duration/tokens/cost) on the non-agentic rejection branch,
   mirroring the executor-failure branch — extractable as a direct fix now.
3. `internal/observatory` pi importer + `ailang chains import-pi`, mirroring
   the motoko importer's schema mapping, and route it from
   `eval_benchmark_agent.go` alongside the motoko branch.
4. Trace the eval-suite stdout path to find the actual buffering layer before
   fixing the live-tail gap.
