# Sprint plan: M-FLEET-QUORUM-SALVAGE

Date: 2026-10-14. Fleet iteration 36, planner stage. Verified base:
`4c5070c5010ebc15abadb7cb5de6edc7d036b7f1` (detached HEAD, initially clean).
Spec: [m-quorum-salvage-retry.md](m-quorum-salvage-retry.md), read in full; its approved
round-2 revision governs this plan. Tickets: `quorum:invalid-absent-on-quoted-literals`
and `quorum:reviewer-invalid-json-discarded`; GitHub #941.

## Goal, scope and estimate

Preserve a strict-token recovered reject as an absent reviewer's veto in Tier-1
synthesis, locate JSON failures by byte offset, and correct the absent-reviewer
remedy. Three sequential, separately testable milestones; each leaves the package
compiling and its tests green. No implementation or git writes occur in this
planning stage. The controller owns execution dispatch, commits and landing.

Budget one working day: M1 60–90 minutes, M2 60–90 minutes, M3 30–45 minutes,
with the remaining time for controller gates and review. Estimated changed LOC:
M1 300 (120 implementation, 180 tests), M2 240 (50 implementation, 190 tests),
M3 40 documentation; total 580, excluding progress/evidence updates. These are
planning allowances, not measured throughput. Recent package history provides
only two seven-day samples: `a5f9af982` added 190/deleted 12 lines and
`f17314335` added 8/deleted 8. The velocity script's archive grep includes ancient
entries and its piped log stops early, so it supplies no reliable LOC/day rate.
The previous 140-LOC zero-signal estimate versus 190 actual additions argues for
buffering this test-heavy change. All existing quorum Go files are below 800
lines (largest 303); new tests go in the companion `salvage_test.go`.

No `.ail` code or new language example is needed: the F1–F3 fixtures and stubbed
seated-quorum regressions are the runnable examples. No new dependencies,
retry machinery, repair, routing, roster, caps, billing or threshold changes.
`tier2.go`, `agentic_caller.go`, their tests, and both CLI consumers stay unedited.
`ShouldEscalate`, seating and present counts retain their contracts.

## HEAD verification and corrections

All references below were read at the verified base; implementation will shift
line numbers, so use symbols thereafter.

| Spec claim | HEAD evidence | Finding / consequence |
|---|---|---|
| Both invalid twigs share one site | `internal/mission/quorum/run.go:159-175`: ParseReviewResult at 159; invalid assignment 161; length test 169; diagnostic branches 170/172 | Correct. Replace line 161 with recordInvalid before the length split. |
| Validation errors reach that same site | `internal/mission/quorum/reviewer.go:131-141`: Unmarshal error at 134-135, ValidateReviewResult error at 137-138 | Correct. T6c needs no extra branch: both errors return as perr. Gate-violating valid JSON must salvage with Offset=-1. |
| Claude envelope and verdict are distinct | `internal/mission/quorum/claude_reviewer.go:105-126`: envelope Unmarshal 109 and diagnostic 111; inner parse 121 and invalid assignment 123 | Correct. Diagnose envelope only; salvage env.Result only at 123. |
| Only present external outcomes satisfy the guard | `internal/mission/quorum/quorum.go:148-189`: absent branch 159-161, increment 163, controller 171-177, zero-signal guard 183 | Correct. Call foldAbsent only in synthesize's absent branch; retain the continue and guard. |
| Absent Markdown currently hides salvage | `internal/mission/quorum/artifact.go:82-97`, absent line 96 | Correct. Add explicit REJECT/PASS suffixes to the Tier-1 line. |
| Reserve replacement retains absences | `internal/mission/quorum/seating.go:114-120`, final synthesis 128, presentCount 132-139 | Correct. Absent B remains in the four-outcome G3 result; reserve D must still run. No seating edit. |
| Tier-2 follow-up is bounded | `internal/mission/quorum/tier2.go:73,82-84`; non-test caller search over cmd/ and internal/ | Correct. Tier-2 starts with synthesize but its own absent branch only appends a note. RunQuorumWithEscalation has no non-test caller; known positive control is `cmd/ailang/design_quorum.go:89` calling RunSeatedQuorum. |
| The skill describes an unlanded controller-count guard | `.claude/skills/mission-control/resources/gate-2-pick.md:233-240` versus quorum.go:183; commit `a5f9af982` (#1767) | Intentionally stale text confirmed. Replace the historical present-tense claim and “until it lands” with a landed note; do not merely change the line number. |

No load-bearing code claim in the requested citation set was wrong at this HEAD.
The doc's earlier verification bases are older, but these code paths still match.
Historical provider/artifact counts and #941 comments are spec evidence, not
newly re-measured here. The doc mentions an AC10 jq read for residual cases but
does not actually give that command; an audit command is supplied below.

Planner observations: `GOPROXY=off go test ./internal/mission/quorum/ -count=1`
passed (`ok`, 0.448s). `tools/launchd/sync-agents-skills.sh --check` exited 0,
including all six mirrored skill trees. No implementation tests were added.
Mission-stage planning skips inbox handling; the unrelated active OAuth sprint
and ambient approval queue are outside this task.

## Registry reuse audit

`ailang pkg search quorum` failed fetching the registry index with
`lookup storage.googleapis.com: no such host` (and warned that the installed
binary may be stale): **UNINFORMATIVE UNDER SANDBOX**, not “no packages found.”
This is an internal Go transport fix, not an AILANG package capability; no
registry dependency can substitute for its private synthesis and caller seams.
Local search found no salvage/Decoder.Token helper in this package. Decisions:

| Milestone | Decision | Existing machinery reused |
|---|---|---|
| M1 | none | encoding/json Token/Unmarshal, stripCodeFences and existing parser contract; add the missing private helper. |
| M2 | none | stubCaller, cheapModel, present, fakeRunner, stubClaude, Selection/RunSeatedQuorum; fresh per-call stubs avoid concurrent captured-input writes. |
| M3 | none | Existing mission-control resource and skill sync tool; no package-like implementation. |

## Milestones

### M1: Strict salvage scanner and shared diagnostics

Dependencies: none. Estimate: 300 LOC, 60–90 minutes.

Exact implementation/test files:

- NEW `internal/mission/quorum/salvage.go`
- NEW `internal/mission/quorum/salvage_test.go`
- `internal/mission/quorum/run.go` (additive Salvaged field only in this milestone)
- `internal/mission/quorum/reviewer.go`
- `internal/mission/quorum/claude_reviewer.go` (envelope diagnostic only here)

Add SalvagedVerdict and the strict salvageVerdict scan, recordInvalid helper,
describeJSONError and diagHalfWindow=120. The field is
`Salvaged *SalvagedVerdict` tagged `salvaged,omitempty`. The helper can compile
before its production call sites are connected. Diagnose both Unmarshal sites
now, leaving outcome and synthesis behavior unchanged until M2.

The scanner uses the parser's trimmed/fence-stripped bytes, requires the first
token to be `{`, tracks object/array depth, and distinguishes keys from values
at depth 1. Skip nested values without stopping; never inspect nested verdicts.
Only exact string pass/reject values qualify; any reject wins across duplicate
keys. Commit a strongest_objection only after a following top-level key or the
root closing brace decodes. Do not repair or search past leading prose.
Offsets refer to the same trimmed bytes as Unmarshal; valid JSON gate failures
use -1. SyntaxError diagnostics use its Offset and a clamped ±120-byte window
with both boundaries snapped to UTF-8 rune starts. Other errors retain leading
`%.200q` raw context. Keep contextual error prefixes useful to callers.

- [ ] T1, T2 and every T3 subcase below pass, including M5b key/value state.
- [ ] Diagnostic tests cover UTF-8 boundary snapping, start/end clamping, fence/whitespace offsets, non-SyntaxError fallback, and Claude envelope offset/window reporting.
- [ ] Mutation evidence M1, M4b and M5b is recorded with red mutant and green restored runs.
- [ ] `go test ./internal/mission/quorum/ -count=1` and `go vet ./internal/mission/quorum/` pass; the additive field omits itself when nil.

Bisect boundary: diagnostics are independently useful; the dormant salvage
helper and omitted nil field do not yet change quorum decisions.

### M2: Preserve the absent veto through production synthesis and artifacts

Dependencies: M1. Estimate: 240 LOC, 60–90 minutes.

Exact implementation/test files:

- `internal/mission/quorum/run.go`
- `internal/mission/quorum/claude_reviewer.go`
- `internal/mission/quorum/quorum.go`
- `internal/mission/quorum/artifact.go`
- `internal/mission/quorum/salvage_test.go`
- `internal/mission/quorum/reviewer_test.go` (existing truncation test gains only its salvage assertion)

Replace the invalid assignment above the length split with
recordInvalid(out, raw), and the Claude inner-parse assignment with
recordInvalid(out, env.Result). Leave the envelope branch without salvage.
Add `AbsentNote.SalvagedVerdict Verdict` tagged `salvaged_verdict,omitempty` and
one foldAbsent(&s, o, prefix) helper, called from synthesize only. Keep accounting
in its current loop. A salvaged reject blocks even with empty objection, using
`[salvaged from invalid response]` and the spec's unrecoverable-objection text
when necessary. Pass is audit only: no vote, external guard signal or seat.
All salvaged outcomes retain Present=false, ReasonInvalid, Result=nil.

Render `; salvaged REJECT — blocks` and
`; salvaged PASS — audit only (not counted)` on Tier-1 absent Markdown lines.
Add all integration/serialization tests to salvage_test.go, reusing existing
helpers. Claude global-call stubs must be restored with cleanup and not run in
parallel. Build valid envelopes with json.Marshal so F2 is a string payload,
not malformed envelope syntax.

- [ ] T4–T8, T6b and T6c pass, including both seated F2 and F3+length paths, preserved reserve refill, and rejection without a recovered objection.
- [ ] T8 protects both omitempty fields, old artifacts, exact Markdown labels, and unchanged absence/accounting/seat semantics.
- [ ] Existing malformed-response and agentic malformed-output tests and all tier2_test.go tests pass unedited. Preserve all existing truncation assertions and add the T6b assertion only.
- [ ] Mutation evidence M6, M8, M8b and M8c is recorded; no Tier-2 or agentic source/test edits.
- [ ] `go test ./internal/mission/quorum/ -count=1`, `go test -race ./internal/mission/quorum/ -count=1`, and `go vet ./internal/mission/quorum/` pass.

Bisect boundary: the mechanical behavior fix and regressions land together;
every invalid production path is wired before the remedy docs rely on it.

### M3: Correct the remedy, mirror it and complete acceptance

Dependencies: M2. Estimate: 40 LOC, 30–45 minutes plus controller gates.

Exact content files:

- `.claude/skills/mission-control/resources/gate-2-pick.md`
- `.agents/skills/mission-control/resources/gate-2-pick.md` (generated mirror)
- NEW `changelogs/unreleased/2026-10-14-quorum-salvage-verdict.md`

Split the absence remedy: budget retains a raised-cap solo re-run; invalid plus
salvaged reject means revise the real objection, already blocked, without a
re-run; invalid with no salvage permits at most one same-cap solo re-run, never
a cap increase. A second invalid is durable and is quoted as
`PROCEED at N-1, <model> absent (invalid)` wherever a degraded proceed is allowed
by synthesis. Do not promote a salvaged pass into a vote or signal. Replace the
stale guard explanation with the landed #1767 external-only guard; use symbol
references to avoid another fragile line citation.

Use `tools/launchd/sync-agents-skills.sh` to generate the mirror. It syncs six
trees, including sprint-* trees outside executor authority; the planner's clean
baseline --check prevents treating existing drift as this sprint's work. Verify
only the intended mission-control resource changes. Unexpected drift belongs
to the controller, not a broad reconciliation in this sprint.

The changelog fragment opens with `### Fixed` and describes recovered vetoes,
offset diagnostics, audit-only passes and the corrected invalid remedy.

- [ ] Source/mirror are byte-identical; sync --check exits 0; remedy contains salvaged_verdict and correct reason-specific rules.
- [ ] AC10 focused tests, lint and additional focused repository gates below pass outside the sandbox where required.
- [ ] All seven mutation proofs have commands, exit codes, named failing assertions and restored-green results in execution evidence.
- [ ] Controller receives the explicit pre-landing Tier-2 queue-row requirement; executor completion does not assert that it is satisfied.

Shared progress/evidence files for every milestone are this plan and
`.ailang/state/sprints/sprint_M-FLEET-QUORUM-SALVAGE.json`. Executor records
actual acceptance/mutation output in the execution-evidence section below and
updates milestone progress; these are the only additional milestone files.

## Test-arm / mutant matrix

Use these planned test names in salvage_test.go so mutation results identify a
specific contract. M* in the mutant column denotes the spec's mutant, not a
sprint milestone. Copy F1/F2/F3 verbatim from the spec as Go raw strings.

| Arm / planned test | Required assertion | Mutant killed |
|---|---|---|
| T1: TestSalvageDiagnostics/offset | F1/F2/F3 ParseReviewResult errors contain at byte 61/94/59. | M1: offset removed. |
| T2: TestSalvageDiagnostics/centred_window | Pad F2 objection by 300 bytes; break is past byte 300 and window contains GOARCH. | M2: leading %.200q substituted. |
| T3: TestSalvageVerdict/fixtures | F1 reject/empty objection; F2 reject/`asserts "GOOS" absent`; F3 reject/empty objection; SyntaxError offsets preserved. | Loss of strict recoverable verdict or confirmed objection. |
| T3: TestSalvageVerdict/unconfirmed | `{"verdict":"reject","strongest_objection":"the test greps "GOARCH" x"}` has empty recovered objection. | M3: unconfirmed free text accepted. |
| T3: TestSalvageVerdict/nested_only | `{"x":{"verdict":"reject"}}` gives nil. | M4: depth ignored. |
| T3: TestSalvageVerdict/after_nested_object | `{"meta":{"k":1},"verdict":"reject","strongest_objection":"after nested","catch":"x` recovers reject and `after nested`. | M4b: stop at first nested opener. |
| T3: TestSalvageVerdict/after_nested_array | Nested notes array/object verdict is ignored; later top-level reject and confirmed top-level objection survive. | M4/M4b: nested depth or premature-stop error. |
| T3: TestSalvageVerdict/prose | `I would reject this` and prose followed by a JSON-looking object give nil; top-level array is rejected. | M5: search prose for an opener. |
| T3: TestSalvageVerdict/key_value_state | `{"other":"verdict","reject":"pass","verdict":"pass","catch":"x` gives pass, not reject. | M5b: string values mistaken for object keys. |
| T3: TestSalvageVerdict/duplicates_and_pass | Broken pass gives pass; duplicate pass then reject gives reject; reject then pass remains reject. | Last-key-wins or accepting arbitrary verdict strings. |
| T3: TestSalvageVerdict/gate_violation | Valid reject with empty strongest_objection/catch gives reject, Offset=-1. | Gate violations excluded from scan. |
| T4: TestSalvageSynthesis/reject | Present pass + absent invalid salvaged reject + controller pass blocks with one marked objection. Empty objection also blocks with the spec fallback. | M6: foldAbsent not called. |
| T5: TestSalvageSynthesis/pass_not_signal | Absent invalid salvaged pass + controller pass blocks with zero-signal objection; absent note records pass without contributing external signal. | M7: salvaged pass counts toward externalPresentCount. |
| T6: TestSalvageProduction/plain | Selection A/B/C primary, D reserve, seats=3; each runner calls runReviewerWith using a fresh stubCaller; B returns F2, others valid pass. Four outcomes, named absent B, three present seats, final blocked. | M8: run.go recordInvalid omitted. |
| T6b: TestSalvageProduction/truncated | F3 + FinishReason=length remains absent invalid, Result=nil, explicit length error, salvaged reject; same B seated run blocks and D still fills. Extend existing TestRunReviewerWith_TruncatedOutputFailsLoudly with salvage assertion. | M8b: recordInvalid only on non-length twig. |
| T6c: TestSalvageProduction/gate_violation | Valid `{"verdict":"reject","strongest_objection":"","catch":"","proposed_fix":""}` through runReviewerWith is absent invalid, Result=nil, reject salvage, Offset=-1. | M8c: recordInvalid runs only for JSON parse errors. |
| T7: TestSalvageClaude/inner_verdict | Stub valid is_error=false envelope with F2 as encoded Result string; absent with salvaged reject. | M10: Claude inner site omitted. |
| T7: TestSalvageClaude/envelope | Broken `{"verdict":"reject","result":"x` is absent invalid with offset/window error but Salvaged=nil. | M10b: salvage attempted on transport envelope. |
| T8: TestSalvageCompatibility/omitempty | Budget ReviewerOutcome and unsalvaged AbsentNote omit salvaged and salvaged_verdict; HEAD artifact JSON decodes unchanged. | M12: either omitempty tag dropped. |
| T8: TestSalvageCompatibility/markdown | Exact reject/pass suffixes render on absent rows; unsalvaged rows retain their existing format. | Missing veto label or pass misleadingly counted. |
| T8: existing regression suite | Existing malformed/gate test, agentic malformed-output test, all tier2 tests unedited; truncation test retains its existing assertions. | Compatibility regressions in excluded paths. |

Also test D3's multibyte boundaries and non-SyntaxError fallback; these are
requirements even though the spec does not assign separate mutant IDs.

## Mutation proof procedure (AC9)

Required mutants: M1, M4b, M5b in M1; M6, M8, M8b, M8c in M2. Never use git
checkout/reset/stash to make or restore a mutant. Use a scratch copy without
git metadata; leave the actual worktree untouched. For example, outside the
sandbox, create a task-specific directory with `mktemp -d`, then
`rsync -a --exclude=.git --exclude=bin --exclude=node_modules ./ "$quorum_scratch/"`
after assigning that path to quorum_scratch. Confirm the unmutated scratch
package passes first. Save original scratch files, apply one mutant, run its
named test, record a nonzero exit with the intended assertion, restore those
scratch bytes, and rerun that test green before trying the next mutant.

| Mutant | Scratch-only mutation | Focused failing test command suffix (`go test ./internal/mission/quorum/ -count=1 -v -run ...`) |
|---|---|---|
| M1 | Drop offset from describeJSONError output. | '^TestSalvageDiagnostics/offset$' |
| M4b | Stop scanning at the first nested opener. | '^TestSalvageVerdict/after_nested_object$' |
| M5b | Remove the depth-1 key/value distinction. | '^TestSalvageVerdict/key_value_state$' |
| M6 | Replace synthesize's foldAbsent call with old absent-note append. | '^TestSalvageSynthesis/reject$' |
| M8 | Replace only run.go's recordInvalid call with the old ReasonInvalid assignment. | '^TestSalvageProduction/plain$' |
| M8b | Move run.go's recordInvalid to the non-length twig, retaining invalid classification for both. | '^TestSalvageProduction/truncated$' |
| M8c | Restrict recording to malformed JSON, so valid JSON validation failures get ReasonInvalid but no salvage. | '^TestSalvageProduction/gate_violation$' |

A compile error, cache-only result, missing dependency or blocked socket is not
a killed mutant. Record actual failure output and the restored-green control.

## Acceptance commands for the controller outside the sandbox

Run from the repo root on the rebuilt implementation, capturing each exit code.
These are future executor/controller commands, not results of this planner.

```bash
go test ./internal/mission/quorum/ ./cmd/ailang/ -count=1
go test -race ./internal/mission/quorum/ -count=1
go vet ./internal/mission/quorum/
make test-changed TEST_CHANGED_BASE=4c5070c5010ebc15abadb7cb5de6edc7d036b7f1
make lint
make fmt-check check-boundaries check-file-sizes check-changelog
bash -n tools/launchd/sync-agents-skills.sh
tools/launchd/sync-agents-skills.sh
tools/launchd/sync-agents-skills.sh --check
cmp .claude/skills/mission-control/resources/gate-2-pick.md .agents/skills/mission-control/resources/gate-2-pick.md
test "$(grep -c salvaged_verdict .claude/skills/mission-control/resources/gate-2-pick.md)" -ge 1
jq -e . .ailang/state/sprints/sprint_M-FLEET-QUORUM-SALVAGE.json
bash .claude/skills/sprint-executor/scripts/validate_sprint_json.sh M-FLEET-QUORUM-SALVAGE
git diff --check
git diff --name-only
```

AC10 requires the first test command, clean lint, clean sync check and remedy
marker. The other focused gates protect races, package/core regressions, shell
syntax, boundaries, formatting, size and fragment conventions. No full
`make test` is an executor requirement. No stdlib/examples gate is needed for
this Go-only internal fix. Review the changed-file list against the milestone
lists; do not broaden scope to fix unrelated pre-existing failures.

**CI coverage: yes**, the quorum package runs, though not via a literal standalone
`go test ./internal/mission/quorum/` workflow line. `.github/workflows/ci.yml:206`
runs `go test -timeout 416s ./...` (Windows counterpart at 629).
`.github/workflows/build.yml:77` runs the go-list expansion excluding scripts and
examples/agents, which includes quorum. `make/test.mk:27-30` (`make test`) also
includes it; the direct hand-run target is **`make test-mission-registry`**
(`make/test.mk:400-405`, tests `./internal/mission/...`). That target is explicitly
not wired into CI; full-suite workflow commands provide coverage.

**UNINFORMATIVE UNDER SANDBOX:** live registry search (observed DNS failure),
provider calls, live review/quorum commands, and socket-dependent CLI test arms.
For example `cmd/ailang/net_scope_public_e2e_test.go:37` uses httptest.NewServer
and `serve_api_debug_sink_test.go:132` uses net.Listen. A bind denial or unavailable
network in the combined AC10 test cannot establish a code failure or a pass;
rerun outside. CLI tests were not run in this planning stage. The pure stubbed
quorum baseline is informative and passed. Sync --check is read-only and passed;
copy mode writes the protected .agents tree, so perform it outside this sandbox.
Lint/tool installs, embed preparation and make test-changed may require available
tools, warmed dependencies or writes beyond the executor authority; any sandbox
denial is a controller rerun, not permission to edit those paths.

## Controller landing steps (not executor tasks)

Before landing, record `quorum:tier2-salvage-fold` as a queue row in
`design_docs/fleet-mission.md`. Cite spec V19: Tier-2's absent branch at
tier2.go:82-84 does not fold salvage, and RunQuorumWithEscalation currently has
zero non-test callers. Trigger the follow-up at the first non-test caller of
RunQuorumWithEscalation; it covers tier2.go folding plus agentic_caller.go
recording. Do not use `ailang mission ticket file` for fleet bookkeeping.
The controller must verify this queue row exists before landing, separately
from marking M3 executor checks complete.

Controller obtains independent sprint evaluation against the spec and this
matrix, reviews mutation evidence and the file allowlist, performs required
mission landing checks, and handles commits. On landing the controller moves
the spec from `design_docs/planned/m-quorum-salvage-retry.md` to
`design_docs/implemented/m-quorum-salvage-retry.md` and updates references as
needed. Neither that move nor the queue edit is a planning-stage action.

After real artifacts accumulate, this optional read finds candidate residual
invalid absences; a candidate requires manual evidence that the absence changed
the result before reopening retry work. It is not by itself proof of causality.

```bash
jq -r 'select(.reviewers != null) | . as $q | .reviewers[] | select(.absent_reason == "invalid" and .salvaged == null) | [input_filename, $q.synthesis.verdict, .model, .error] | @tsv' .ailang/state/mission-quorum/*.json
```

## Risks and execution evidence

- Prematurely accepting free text or confusing keys and values can invent an
  objection/verdict. Strict Token scanning, T3 and its three mutation proofs
  constrain that risk; do not introduce JSON repair.
- Missing the length or validation branch silently reopens the veto hole.
  T6/T6b/T6c must traverse runReviewerWith, not inject Salvaged into a fake outcome.
- A recovered pass must never satisfy either guard or seat count. Retain absent
  state, prove T5 and reserve refill, and leave seating code untouched.
- Tier-2's own absent rejects remain outside this change. Its explicit queue row
  is an AC10 landing precondition, not optional cleanup.
- UTF-8 errors are byte-based; rune snapping must preserve window validity near
  both ends. Use multibyte fixtures rather than assuming ASCII-only output.
- Global Claude stubs and stubCaller captured fields can race; keep Claude tests
  sequential and create independent API stubs for parallel seats.
- Sync may expose unrelated drift or prohibited writes; baseline check is green,
  but inspect the final diff and let the controller resolve any new drift.

Execution evidence: pending implementation. Append milestone gate results and
the seven scratch-mutation red/restored-green transcripts here. Do not label
planned commands as completed. Planner-only baseline: quorum tests PASS,
mirror --check PASS; registry search UNINFORMATIVE UNDER SANDBOX.
