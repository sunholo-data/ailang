# M-PKG-SMOKE-STAGED-LOCK — independent round 2: PASS95/100

**Evaluated source:** PR [#2013](https://github.com/sunholo-data/ailang/pull/2013), exact `165f87a12ceb0e20c04216c472004c11685e417b`. This reviewer did not author its product implementation. Root retains formal acceptance, merge and archival authority; the reviewer changed only `/tmp` analysis/reports.

## Actual CI gate

Existing `ci_gate.sh` returned **PASS at20:50:25UTC** (26checks,19green,7skipped). The existing evaluator subsequently confirmed tests/lint from CI, file-size PASS, zero new TODO/HACK/FIXME, no performance trigger. The final exact-head snapshot adds one successful context: **27completed checks,20SUCCESS/7SKIPPED, zero failed/cancelled/pending**, PR OPEN/CLEAN.

Actual new-head primary Linux/Windows tests, all Build matrix siblings, docs, lint, structural checks, Sonar, CodeQL and vulnerability contexts succeed. This replaces the round-1 *gate outcome at the new head*, not the old failure evidence. The successful Build workflow is [40950291051](https://github.com/sunholo-data/ailang/actions/runs/40950291051); main CI is [40950290995](https://github.com/sunholo-data/ailang/actions/runs/40950290995). Full per-context URLs/conclusions are in `pr2013-final.json` and the report JSON.

No workflow was retried by this evaluator and no local product test, CLI proof, full suite or coverage run was duplicated. `automated-evaluation.log` explicitly records `gate_source:ci`; its only local gate was static file-size analysis. Static Go AST parsing is analysis, not product execution.

## Source and proof equivalence

Independently compared **all33 owned paths**, not just implementation filenames. All21 owned Go/AIL files and every static asset/TOML fixture are byte-identical to prior0853. Thirty owned paths are identical; the only three owned updates are sprint JSON, companion plan and validation evidence. The owned delta pathset against accepted dev565 is also identical. Working-tree bytes match immutable165.

Every incoming non-owned path equals accepted `565fc1b0461f1408d930df3029a72beef60f7ff1`. The sole global Go/AIL delta from0853 is the separately accepted `internal/runner/worker_terminal_pty_test.go`. Fleet driver/docs changes are inherited unchanged. This is qualified source/fixture equivalence, not an assertion of whole-binary execution equivalence.

Therefore all21 previously independently reviewed acceptance rows retain their source and scoped runtime proof, with actual final165 terminal CI now supplying the previously failed aggregate gate. All15 retained owner local-log hashes match. Current owner combined selective proof atcc2 also passes (pkg10.222s, CLI10.820s, runner19.027s); it was read, not reexecuted here. See `source-identity.json`, `quality-and-preserved-evidence.json`, `acceptance-matrix-final.json` and original round1 reports.

- M1 preserves validated previous-only D2 matched-git carry-forward, exact registry source/assets and staged root assets, regenerated registry lock, immutable author inputs, silent shared rewrite, fail-closed malformed/missing targets and authority refusal. Exact-version unavailability cannot use newer cached bytes.
- M2 preserves setup/rErr/nil/process/start/timeout causes, first meaningful diagnostic before truncation, UTF8-safe8192/2048-byte budgets, unchanged Notes/PUB015/NotRun/schema/timer policy. Independent D1 signal RED and GREEN plus actual complete-stage owner RED/GREEN are retained. Signal-specific Windows behavior remains explicitly N/A because TerminateProcess uses ordinary exit semantics; common controls execute and actual new Windows CI is green.
- M3 remains warning-only after successful lock save: actual source names, stable ordering, no resolution/precedence/exit/write change. Real warning/LockDrift/path/conflict proofs remain applicable.
- M4 retains a strict actual CLI discriminator: author-local pair must fail exit23; quality/dry-run pass only with REGISTRY_SOURCE + REGISTRY_SOURCE_ASSET + STAGED_ROOT. Observed mock reads/zero writes, unchanged inputs/private state, no message/coordinator DB, no-Z3 and human/JSON proof remain unchanged.

The existing criterion parser marks prose lacking simple anchors as unverifiable; this is resolved by the complete independent21-row source/test/log/CI map. No acceptance points come solely from executor `passes` fields.

## Artifacts and base100 score

JSON status is completed, every milestone has passes=true/start/completion/notes, canonical plan/design paths and velocity metrics are populated, and no plan milestone checkbox remains pending. The design accurately states implementation complete/pending independent acceptance. Guide/changelog and the approved controlled two-package AIL fixtures supply the required example; an unrelated standalone runnable is not required.

| Category | Score | Evidence |
|---|---:|---|
|Tests|20/20|Actual exact165 aggregate CI PASS; no failed/cancelled/pending context|
|Lint|10/10|Exact CI lint SUCCESS, evaluator confirms clean|
|Acceptance|30/30|All21 independently mapped criteria, qualified unchanged source/scoped proof plus current broad CI|
|Code quality|10/15|Literal physical-inclusive FuncDecl spans, including tests:18functions>50lines;−2each capped−5|
|Documentation|15/15|Changelog, approved fixture/guide example and accurate design status|
|Design fidelity|10/10|Staging/D2/assets/diagnostics/warning contracts preserved|
|**Total**|**95/100 — PASS**|No hard failure|

The fresh AST inventory exactly matches round1, including RewritePathDeps57lines, newSmokeCLIProof60 and TestSmokeStage_GitCarryForward88. No owned Go file exceeds800lines, no new TODO/HACK/FIXME, no artifact deduction. `functions-over50.json` records all18 spans. No conditional IR/compiler-regression or performance bonus applies: this is host package/CLI tooling, without compiler/exec infrastructure changes or performance goals.

## Prior failure remains preserved

Round1 remains **FAIL75 at0853**, with both raw4.59MB Build failure logs and their original SHA256s verified unchanged. Attempt1 job123479693310 and attempt2 job123488317858 both failed `TestWorkerTerminalPTYLifecycle/eval/sigterm` from an empty PID read; original durations/package successes and actual failed workflow40883138509 remain in the old report. This evaluator neither reviews/modifies its old operator card nor reclassifies those executions as successes. The accepted dependency was normally integrated, and only new actual165 CI clears the current hard gate.

## Disposition

Independent **PASS95** applies only exact165. Root may formally accept and normally merge; root owns design+companion archival and issue disposition. #1900 implementation still waits root's confirmation of the actual accepted #2013 merge. No source/status/approval/card/PR comment/merge action was performed by this reviewer.
