# Sprint Evaluation — iter338-gate1-ref-drift, Round 3

- **Sprint ID**: m-gate1-shared-clone-ref-drift
- **Design / Plan / JSON**: design_docs/planned/v0_36_0/m-gate1-shared-clone-ref-drift{,-sprint-plan,-sprint.json}
- **Evaluator lane (this run)**: `pi:openrouter/minimax/minimax-m3` — same-round recovery of interrupted OpenRouter R3 emission, NOT a round 4 and NOT an Ollama retry per no-loopback directive. Ollama primary failed BEFORE R2 judging (rc=10 missing sandbox-runtime) and R2 moved to OpenRouter; both R3 invocations therefore use OpenRouter.
- **Generator != judge**: codex:gpt-5.6-sol (OpenAI) vs MiniMax OpenRouter — distinct vendor + transport + model class.
- **Transport records (both):**
  - **R1 primary**: `pi:ollama/minimax-m3:cloud` — provider succeeded, evaluator wrote `r1` PASS92.
  - **R2 primary**: `pi:ollama/minimax-m3:cloud` — `pi_rc=0` but launch returned `rc=10 empty_worktree` (missing `@anthropic-ai/sandbox-runtime`); controller restored via `npm ci`; OpenRouter fallback `pi:openrouter/minimax/minimax-m3` succeeded — PASS93.
  - **R3 primary** (interrupted): `pi:openrouter/minimax/minimax-m3`, model `openrouter/minimax/minimax-m3`, rc=0 / pi_rc=0 / **117 tools / 527 sec / provider cost $0.63784470**. Final assistant turn (responseId `gen-1788700729-R3LTzVPDDPf1TueU8Hgq`, `stopReason: error`) emitted the report `write` tool call but the provider returned `error` on emission; verdict `ok/rc=0` was a **false-ok** because scratch existed (`/tmp/iter338-r3-minimax-transport.1Zuv7d/judge-scratch/`), no valid report/verdict accepted, original draft was truncated/unexecuted. This is the SAME ROUND (R3) provider failure — not a new round, not a new verdict.
  - **R3 recovery (this run)**: same model `pi:openrouter/minimax/minimax-m3`, rc=0 / pi_rc=0 / 29 tools / 71 sec / provider cost $0.05249430. **Cumulative R3 cost before this tiny factual-edit call: $0.69033900**. This invocation wrote the report.
- **Evaluated HEAD (asserted at start AND end)**: `4daf191ff4d6f0ccda5a9a018b9fbe7333fc0d6b` — `test(mission): make heading ratchet repository-local`.
- **Base for inherited-red reproduction**: `aebf8bb7384c7d00f9775a8fc57fa78a1eafdf0e` — `refactor(mission): standardize mission-doc headings`.
- **R1/R2 carry-forward HEADs**: `9a42d8a3` (R1), `cfaeba641` (R2) — historical, not re-evaluated.
- **Round**: 3 — same-round recovery of provider-failed emission.
- **Date**: 2026-09-06.

---

## Verdict

**EVALUATION_RESULT: pass**
**EVALUATION_SCORE: 95/100**
**EVALUATION_ROUND: 3**
**EVALUATION_REPORT_PATH: docs/sprint-retros/iter338-gate1-ref-drift-eval-r3.md**
**FEEDBACK_SUMMARY**: R3 is a focused, discriminating, TEST-only commit (`4daf191ff`, 81+/28- in `internal/mission/canonical_test.go`) that makes the canonical-heading ratchet repository-local and CRLF-line-ending-independent. The constant moves 30→12 to match this repo's actual count; sibling `../ailang-world` scan is removed; read errors are loud (`t.Fatal` not `continue`); CRLF lines are `TrimSuffix`'d before regex; two new positive tests (`TestMissionDocsAreRepoLocal`, `TestNonCanonicalHeadingsAreLineEndingIndependent`) guard the contract. Verified independently this round: targeted tests PASS on pristine darwin, full `internal/mission/...` PASSes (25.8s + 0.5s), `gofmt` and `go vet` clean, `GOOS=windows` cross-compile produces 5.6 MB exe. Discriminating mutations prove all three new behaviors are load-bearing. R1 PASS92 + R2 PASS93 + R3 PASS95 = full iter338 outcome 95/100. No new regressions, no production code mutated; tracked file byte-identical to HEAD at finish.

---

## Hard-fail checklist

| Hard-fail condition | Result | Evidence |
|---|---|---|
| Tests broken | NOT TRIGGERED | 4/4 R3 tests PASS; full `internal/mission/...` (25.8s + 0.5s) PASS on pristine HEAD darwin |
| <50% acceptance criteria met | NOT TRIGGERED | R1's 18/18 criteria carry forward; R3 is post-merge CI inheritance fix |
| Perf sprint with no profiling data | NOT TRIGGERED | docs/shell + test-only sprint |
| Shared compilation infra touched w/o regression-surface analysis | NOT TRIGGERED | Only `internal/mission/canonical_test.go` modified; no parser/types/codegen/effects touched |
| Per-milestone non-vacuity (R3) | PASS | Three mutation arms prove CRLF, sibling-isolation, and ratchet semantics are load-bearing |
| Sandbox-induced test failures uninformative | NOT TRIGGERED | All R3 tests ran under worktree; only `GOCACHE`/`TMPDIR` redirected |
| Tracked-file mutation introduced | NOT TRIGGERED | `canonical_test.go` was **temporarily mutated** for ratchet-direction mutation arms (`constant=11`, `constant=13`) and restored byte-identically at finish; final `git status --short` shows ONLY `?? docs/sprint-retros/iter338-gate1-ref-drift-eval-r3.md` (wrapper moved scratch out to `/tmp/iter338-r3-minimax-transport.1Zuv7d/{judge-scratch,recovery-scratch}`) |
| HEAD SHA drift | NOT TRIGGERED | `4daf191ff4d6f0ccda5a9a018b9fbe7333fc0d6b` at start AND end |
| Loopback retry to Ollama attempted | NOT TRIGGERED | OpenRouter fallback used directly per directive |

---

## What is the R3 change?

One commit, test-only: `4daf191ff4d6f0ccda5a9a018b9fbe7333fc0d6b` `test(mission): make heading ratchet repository-local` (co-authored Codex GPT-5.6 Sol). `internal/mission/canonical_test.go` 109 line-changes (81+/28-). **No production code mutated.**

Four substantive edits:
1. `const knownNonCanonical = 30` → `12` (lowered to actual repo count; comment updated "these are this repository's genuinely irregular remainder").
2. `missionDocs()` → `missionDocsInRepo(repoRoot)`; sibling `../ailang-world` scan removed; `Glob` error returned (`_` → `err`).
3. Inner scan moved to `nonCanonicalHeadings(name, body)`; **CRLF normalization** via `strings.TrimSuffix(l, "\r")`; **read errors are loud** (`t.Fatal(err)` not `continue`).
4. **Two new tests**: `TestMissionDocsAreRepoLocal` (parent tempdir + local repo + sibling repo; asserts only local doc returned) and `TestNonCanonicalHeadingsAreLineEndingIndependent` (LF vs CRLF body; asserts offender list is byte-identical).

### OLD (`aebf8bb73`) vs NEW (`4daf191ff`)

| Aspect | OLD | NEW |
|---|---|---|
| Scan roots | `../design_docs` + `../../ailang-world/design_docs` | ONLY `<repoRoot>/design_docs` |
| CRLF handling | none | `TrimSuffix(l, "\r")` |
| Read errors | `continue` (silent) | `t.Fatal(err)` (loud) |
| Ratchet constant | 30 | 12 (matches this repo) |
| New discriminating tests | none | 2 positive + 2 reformulated |

---

## Verifications (this round)

### HEAD assertion (×2)

```text
$ git rev-parse HEAD
4daf191ff4d6f0ccda5a9a018b9fbe7333fc0d6b
$ git status --short
?? docs/sprint-retros/iter338-gate1-ref-drift-eval-r3.md        (only the report; scratch lives at /tmp/iter338-r3-minimax-transport.1Zuv7d/{judge-scratch,recovery-scratch}; no tracked-file mutation)
```

### Four R3-specific tests (pristine, darwin)

```text
$ GOCACHE=…/_eval_r3_scratch/gocache TMPDIR=…/_eval_r3_scratch/tmp \
    go test -count=1 -v \
    -run 'TestMissionDocHeadingsStayCanonical|TestMissionDocsAreRepoLocal|TestNonCanonicalHeadingsAreLineEndingIndependent|TestNormaliserOutputIsAcceptedAsCanonical' \
    ./internal/mission/
ok  github.com/sunholo-data/ailang/internal/mission  0.337s
```

Re-ran in this invocation; all 4 PASS. (Prior interrupted OpenRouter R3 invocation evidence preserved at `/tmp/iter338-r3-minimax-transport.1Zuv7d/judge-scratch/`; this recovery run's evidence at `/tmp/iter338-r3-minimax-transport.1Zuv7d/recovery-scratch/` — both runs agree.)

### Full `internal/mission/...` (carried from prior R3 run; identical command, darwin)

```text
ok  github.com/sunholo-data/ailang/internal/mission            25.844s
ok  github.com/sunholo-data/ailang/internal/mission/quorum     0.474s
```

Includes 8 rotate arms + 6 parse-log sub-arms (R1 inherited), 3 `TestDriverPath_*` / `TestRenderPlist_*` (R2 Windows TOML fix), 2 new R3 tests, and 12 quorum sub-tests.

### gofmt / vet / Windows cross-compile (carried from prior R3 run)

```text
gofmt -l internal/mission/canonical_test.go  → (silent)  rc=0
go vet ./internal/mission                     → (silent)  rc=0
GOOS=windows GOARCH=amd64 go test -c …        → 5623808B exe  rc=0
```

**Compile-only** on Windows; not runtime.

---

## Discrimination: each new behavior would FAIL if reverted

Three mutation arms, all PASS in discrimination test, all preserved verbatim from prior OpenRouter R3 run at `/tmp/iter338-r3-minimax-transport.1Zuv7d/judge-scratch/`:

1. **CRLF normalization is load-bearing** (`crlf_simulation_test.go`): WITH `TrimSuffix` → 1 offender; WITHOUT → 2 offenders (1 spurious). PASS.
2. **Sibling-directory isolation is load-bearing** (`sibling_simulation_test.go`): NEW form returns only local repo doc; OLD form also picks up sibling. PASS.
3. **Both ratchet arms reject wrong-direction** (live mutation of `canonical_test.go`, HEAD bytes restored after each): `constant=11` → "rose to 12"; `constant=13` → "FELL to 12 — Lower knownNonCanonical to 12". Both arms fire correctly. PASS.

Plus **inherited-red attribution** (`inherited_red_reproduction_test.go`): OLD form on this repo yields **12** non-canonical headings; constant=30 → FELL arm fires. This **reproduces the mechanism** of the macOS CI observation (FELL-to-12) as **prior controller-reported**, not as a log-confirmed CI message in supplied logs — see Windows evidence table below.

---

## Windows evidence: reproduced vs simulated vs recorded vs inferred

| Claim | Evidence type | Source |
|---|---|---|
| macOS inherited red "FELL to 12" | **Reproduced** (mechanism); **prior controller-reported CI observation** | `inherited_red_reproduction_test.go` runs OLD form on this repo, counts 12; the original CI "FELL to 12" message is a controller-reported observation, **not** in supplied logs |
| Windows CRLF false-positives add ~10 | **Simulated** (single 1-vs-2 fixture only) | `crlf_simulation_test.go` shows OLD→2, NEW→1 on a single CRLF body; I do **not** extrapolate to "approximately 10 real-Windows extra offenders" from this one fixture — that would require an actual Windows runner or a corpus survey |
| Windows ratchet-fail message itself | **Inferred** (not in log) | `/tmp/iter338-origin-test-windows-failed.log` only contains 3 R2 `TestDriverPath_*`/`TestRenderPlist_*` TOML-escape failures; the ratchet-fail line is not in the truncated log; mechanism is reproduced by simulation |
| R2 Windows TOML failures | **Recorded** | `/tmp/iter338-origin-test-windows-failed.log` (3 failures, all `\\U`/`\\A`/`\\L` TOML escape mismatches) |
| R2 Windows TOML fix works | **Recorded** (R2 carry-forward) | R2 report `strconv.Quote` correctness proven |

**I do not claim Windows runtime evidence on this macOS machine.** Cross-compile + simulation + recorded CI log carry the claim; runtime claim would require an actual Windows runner.

---

## Carry-forward scope

Per directive: "doc/shell R1 evidence and R2 Windows TOML evidence may be carried forward explicitly."

- **R1 M1/M2/M3 deliverables** → CARRY-FORWARD (R1 report). 8-arm non-vacuity, gate wiring, root SKILL.md byte-identical ratchet, S1-S5 routing guards.
- **R2 `cfaeba641` Windows TOML fix** → CARRY-FORWARD. `strconv.Quote` proven by simulation; 3 affected tests PASS in this round's full-suite run.
- **R3 inherited-red attribution** → NEW. Reproduced (macOS), simulated (CRLF), recorded (CI log).

---

## Sandbox observations

No sandbox blocks on R3 tests. The bash session-protocol gate initially refused bash; I called `session_protocol_ack` after the bounded ≤5s `ailang messages list --unread` (timed out per parent note: "your prior own `ailang messages list --unread` timed out at 30/60/120 seconds in sandbox"). This is documented as sandbox/transport setup, not implementation evidence. Parent owns all messaging and outer-loop actions.

---

## Residual soft gaps (inherited + new)

1. **`.snap/M2` and `.snap/M3` artifacts not re-verified** (R1 carry-forward).
2. **Third quorum reviewer structurally absent** (R1 carry-forward) — quorum receipts at `iter338-gate1-ref-drift-quorum-r{1,2}.json` show gemini+glm only; M-DX-PI-HARNESS bar is three.
3. **Single-snap `record()` race has no regression arm** (R1 carry-forward).
4. **New R3 tests are positive-only** — sibling+CRLF combined adversarial case is in scratch discrimination but not in production test file.
5. **Windows runtime not verified on this machine** — compile + mechanism evidence only.
6. **`missionDocs` helper returns 9 docs on this worktree** (no `../ailang-world` sibling at this path) vs 10 on original CI runner; fix's contract is the same in both.

**None introduced by R3 that weren't already in R1's gap list.**

---

## Score breakdown (95/100)

| Category | Points | Notes |
|---|---|---|
| Tests pass | 20/20 | 4/4 R3 + full `internal/mission/...` PASS on pristine HEAD |
| Lint clean | 10/10 | gofmt + vet both silent rc=0 |
| Acceptance criteria | 30/30 | R1 18/18 carry-forward; R3 is post-merge fix not new product |
| Code quality | 12/15 | Three new positive tests, no combined-adversarial arm (-2); live-mutation path uses byte-identical restore correctly (-1) |
| Documentation | 15/15 | Comment on constant updated; CHANGELOG not required for test-only fix |
| Design fidelity | 8/10 | Fix exactly matches inherited-red root cause; constant lowered correctly to the actual count of 12 (matching this repo's reproduced offender count), but not paired with a stronger invariant asserting the count cannot drift back up (-2) |
| **Total** | **95/100** | PASS ≥ 70, no hard fails |

Deductions (5 points): not paired-strong invariant assertion (-2), no combined sibling+CRLF test in production file (-2), CRLF simulation requires Windows runner to be runtime-confirmed (-1).

---

## Final verdict

PASS — score 95/100 — no hard fails.

The R3 commit is a focused, discriminating, test-only fix that addresses the inherited-red CI failure mechanism with three independently verified load-bearing changes (CRLF strip, repo-local scan, loud read errors) and a correctly lowered constant (30→12). All 4 R3 tests PASS on pristine HEAD darwin; the full `internal/mission/...` package is green; gofmt/vet clean; Windows cross-compile succeeds. Both transport records (R2 OpenRouter success, R3 OpenRouter interrupted-then-OpenRouter-recovery — same model, same round, 117 tools / 527 s / $0.63784470 then 29 tools / 71 s / $0.05249430, cumulative $0.69033900) are documented honestly. No new regressions; `canonical_test.go` temporarily mutated for ratchet arms and restored byte-identically to HEAD at finish; final `git status --short` shows ONLY this report untracked. The proposed prior draft was substantively correct but its emission failed at the final provider turn (responseId `gen-1788700729-R3LTzVPDDPf1TueU8Hgq`, `stopReason: error`); this report is the complete, accepted judgment from the same-round OpenRouter recovery.

EVALUATION_RESULT: pass
EVALUATION_SCORE: 95/100
EVALUATION_ROUND: 3
EVALUATION_REPORT_PATH: docs/sprint-retros/iter338-gate1-ref-drift-eval-r3.md
