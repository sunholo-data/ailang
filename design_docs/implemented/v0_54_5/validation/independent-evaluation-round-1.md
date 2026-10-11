# Independent implementation evaluation — M-WORKER-FOLLOWUP, round 1

**PASS, 87 points** (threshold 70). Evaluated `d1f91706dec6ebc933f4933bc5d020e05adf0fce`; full gates on `0306217c7e1ed878409ed6f0514739561ee93927`, baseline `bc078dc1b41308542bbccd3417bf040283e817df`. No implementation blocker/hard failure. **M4, platform CI and release delivery remain pending.**

| Rubric | Points |
|---|---:|
| Full tests / lint | 20 / 10 |
| Acceptance criteria | 20 |
| Code quality | 5 |
| Documentation / fidelity | 15 / 10 |
| Conditional regression surface | 7 |
| Total | 87 |

M1–M3 substantively verify 13/19 sprint criteria; incomplete M4 earns no acceptance points (its showcases separately verified). Quality deductions cover capped long-function deduction and incomplete completion/velocity artifacts. Conflict analysis and named existing fixtures pass; no current-baseline corpus differential was banked, so conditional category earns 7/10. This is not a performance optimization sprint.

Independent local gates ran once each: full evaluator script test/lint/file-size and `make verify-examples`, all exit 0. Examples: 255 total, 246 passed, zero failed, nine skipped;228 manifest modules checked, zero drift. Existing stale `lambda_expressions.ail` warning retained. Script copy changes only passing-log retention. Zero-TODO helper emitted zero twice/invalid JSON; direct diff confirms zero. Criteria helper left 18 prose criteria unverifiable, none missing; independent source/test/evidence judgment supplies verification rather than trusting generator flags. Raw log hashes/sizes are banked in JSON.

Only post-gate diff is one test-fixture shell line: write each positive PID to its same-directory `.tmp`, then atomic `mv` to the ready filename. Runtime unchanged; exact independent diff review approved, targeted supervisor-loss test 20 repeats PASS in 10.149 s. PID ownership, absence, bystander and deadline assertions unchanged. No repeated whole-suite run needed.

Private protocol retains same-file/run-only admission, fd3 separation, bounded frames, close-on-exec endpoints and no transported PID authority. Completion joins receipt, sole waiter and owned IO; root deadlines propagate and hard policy timeout/output boundaries remain immediate. EPERM checked Join and unrelated-worker isolation retained.

All staged findings are corrected with passing regressions: seven legitimate early exits publish receipts; malformed/oversized/version/wrong-kind/closed requests retain failure; standalone admitted supervisors reject missing/failed receipt; local completion is distinct from actual lifeline EOF, which stays effective after receipt. Fourteen managed/two-async cancel/EOF/return/error/exit/signal scenarios prove exact PID absence including zombies and the 2 s + 500 ms bound. Nested supervision, stalled deadline, natural cancel race, hardening/containment/provenance and existing lifecycle fixtures pass.

Lazy AI publishes complete concrete handler/cached error once. Text/JSON/step/cache/stream/image/base64/refs/speech, resolver/options/route metadata/callbacks covered. Denied/exhausted AI cannot initialize. Selected/named providers preserve frozen auth/config through budget/clone views; no-model selected AI remains nil. Direct/installed-bin evaluator/strict offline/help, missing-key first use, local mocks, eager and deterministic stub controls pass without live inference.

AC1–9 verified locally. AC10 awaits platform CI/final evaluation artifacts. AC11 awaits dev merge, supporting release, official checksum/provenance/probes and pinned consumer retest, μRAG, cloud TEST→PROD and public MCP checks. Async consumer EOF evidence is natural completion, not a new EOF notification. Primary will reconcile README and move design/plan/validation together before PR; evaluator made no move or publication.

Machine report: `.ailang/state/evaluations/eval_M-WORKER-FOLLOWUP_round_1.json`.
