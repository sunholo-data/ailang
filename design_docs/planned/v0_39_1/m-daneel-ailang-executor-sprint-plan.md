# Sprint Plan: M-DANEEL-AILANG-EXECUTOR

## Summary

Deliver Daneel's host-dispatched `ailang_only` executor, including the missing Gemini Google Search grounding surface required by its first `ext/search` package. The critical path is binary work first, a release, then multivac configuration, and finally Daneel dispatch/completion consumption and a live grounded-search round trip.

**Source design:** `design_docs/planned/v0_39_1/m-daneel-ailang-executor.md` (PR #1240, auto-merging when this plan was created)
**Duration:** 5 engineering days (approximately 35 hours)
**Estimated change:** approximately 1,650 LOC including tests, fixtures, examples, configuration, and docs
**Risk level:** High — cross-repository release/config ordering plus a new provider-specific AI result shape

## Binding Decisions and Corrections

Mark ratified D1–D4 in an attended session on 2026-09-16. These decisions are acceptance criteria, not open questions:

- **D1:** completion `summary` is the primary answer channel and must end with an `ANSWER:` block within the 1,200-byte tail; the transcript artifact is the overflow/debug channel.
- **D2:** `allowed_caps = [IO, FS, Clock, Net, AI, Process]`; `process_allow = ["git:status", "git:diff", "git:log"]`; no `Env`; `net_allow` is enumerated from Daneel's host source; `cli_allow` is the complete default set plus `lock`; work stays under `.ailang-scratch/`.
- **D3:** `glm-5.3-flash` is the v1 executor model.
- **D4:** only Daneel host dispatch is supported; a malformed Request ends in `BLOCKED`.

The design doc still contains pre-ratification examples that say Process is absent. Implementation must update those stale lines and negative probes: `Env` and non-allowlisted Process commands are refused, while `git status`, `git diff`, and `git log` are admitted.

There are two distinct model choices and the implementation report must name both: the executor model remains D3's `glm-5.3-flash`; the AILANG AI handler used by grounded search must resolve to the Gemini provider because Google Search grounding is refused elsewhere. M1/M2 must prove policy pinning cannot be bypassed by a cross-provider per-call model. If the current single `ai_provider` field cannot express both facts, configuration is blocked until the policy uses a Gemini provider for the AILANG program while the registry separately pins the executor model to D3.

## Current Status and Velocity

- v0.39.1 is at this checkout's HEAD and contains `ailang_cli`, `cli_allow`, and `.ailang-scratch/` support.
- The source design is complete and ratified in PR #1240, but had not yet appeared in the worktree when planning began.
- `cmd/ailang/run_policy.go` does not carry or enforce `Policy.AIProvider`.
- `internal/ai/gemini/types.go` only represents `functionDeclarations`; `std/ai` has no grounded-search call or grounding-source result type.
- Repository history in this coordinator checkout is shallow (one release commit), so the velocity script could not derive a trustworthy LOC/day figure. The plan uses the design's three-day estimate, adds the new provider milestone, and reserves a full release/deployment gate, yielding five days.

## Dependency and Release Sequence

```text
M1 policy binding
  -> M2 Gemini grounded search
  -> M3 release and binary verification
  -> M4 multivac registry + policy + template
  -> M5 Daneel host dispatch + completion consumer + ext/search
  -> M6 live round trip and boundary probes
```

M4 must not begin until M3 proves the released binary understands every emitted policy/registry key. The loader drops unknown keys silently, so deploying configuration early would create apparent authority without enforcement. Templates now deploy on push after trigger fix `57099e2`; M4 verifies that path rather than requesting an image rebuild.

## Milestones

### M1 — Bind `ai_provider` in `run --policy`

**Estimate:** 120 implementation + 150 test/docs LOC; 4 hours
**Owned repository:** AILANG
**Example files:** `cmd/ailang/run_policy.go`, `cmd/ailang/main_run.go`, `cmd/ailang/run_policy_ai_test.go`, `docs/docs/guides/agent-tool-policy.md`, `changelogs/v0.32-current.md`

Tasks:

- Add `aiProvider` to `runPolicyResolved`, derive it solely from `Policy.AIProvider`, and include it in the admission diagnostic.
- Bind `stub` and named providers through the same handler setup used by `--ai`.
- Refuse `--ai` with `--policy`, AI without `ai_provider`, and `ai_provider` without AI.
- Add a provider-boundary test proving a per-call model cannot escape to a different provider.
- Audit callers of `--policy`; document that the `ailang_run` tool supplies no widening AI flag.

Acceptance criteria:

- `AI + ai_provider="stub"` runs an AI-cap program without `--ai`.
- All three configuration/widening errors exit non-zero with named, stable reasons.
- Policy diagnostics include the selected provider/model without exposing credentials.
- `go test ./cmd/ailang/...` and policy guide parity tests pass.

### M2 — Add Gemini Google Search grounding to `std/ai`

**Estimate:** 300 implementation + 300 tests/fixture/example LOC; 1.5 days
**Owned repository:** AILANG
**Example files:** `std/ai.ail`, `internal/builtins/ai_step.go`, `internal/effects/ai_step.go`, `internal/ai/provider.go`, `internal/ai/gemini/types.go`, `internal/ai/gemini/step.go`, `internal/ai/gemini/step_test.go`, `examples/runnable/ai_google_search_grounded.ail`

Technical contract:

- Add a dedicated Result-returning surface, provisionally `searchGrounded(prompt) -> Result[GroundedResult, AIError] ! {AI}`, rather than pretending Gemini's built-in tool is a host-dispatched function.
- `GroundedResult` returns answer text and normalized sources (at minimum URI and title; include provider grounding/support fields only when stable and useful).
- Gemini serializes the native `tools: [{"google_search": {}}]` block and parses grounding metadata from a recorded response fixture.
- Every non-Gemini provider returns `CapabilityNotSupported` with a clear message; no provider silently ignores grounding.
- The request/result representation must not change wire output for ordinary `call`, `callJson`, or `step` requests.

Tasks:

- Extend the shared request/response contract with an explicit grounded-search operation and normalized source metadata.
- Implement Gemini request mapping and response metadata parsing.
- Implement the common non-Gemini refusal path and test at least one direct provider plus the handler dispatch boundary.
- Add provider-level request/response tests using a checked-in recorded fixture with secrets and unstable IDs removed.
- Add a runnable AILANG example and freeze/update stdlib interface artifacts as required.

Acceptance criteria:

- Recorded Gemini fixture proves the native Google Search tool block and returns non-empty answer text plus sources.
- Non-Gemini calls return typed `CapabilityNotSupported`, naming Google Search grounding and the selected provider.
- Existing provider request-shape tests remain byte-for-byte unchanged for non-grounded calls.
- `make test-core`, targeted provider/builtin tests, `ailang check` on the example, `make freeze-stdlib`, and `make lint` pass.

### M3 — Release gate for binary-side semantics

**Estimate:** 40 release notes/verification LOC; 0.5 day plus CI latency
**Owned repository:** AILANG/release pipeline
**Example files:** `changelogs/v0.32-current.md`, generated release artifacts as directed by `release-manager`

Tasks:

- Complete core tests, provider tests, stdlib freeze verification, `make check-boundaries`, and `make simplicity-audit`.
- Create the next patch release containing M1 and M2 through the release workflow.
- Verify the exact released binary accepts `ai_provider`, the grounded-search builtin, `cli_allow`, and Process narrowing.
- Record the release version/commit in the implementation report and provide it to multivac configuration.

Acceptance criteria:

- Release CI is green and the deployed binary reports the expected version.
- A smoke policy using all M4 keys loads without unknown-key loss and produces a non-empty policy digest.
- M4 remains blocked until these checks pass.

### M4 — Deploy multivac agent, policy, and task template

**Estimate:** 160 config/template + 100 tests/probes LOC; 1 day
**Owned repositories:** multivac deployment config and Daneel template
**Example files:** `policies/daneel-executor.toml`, the multivac agent registry YAML, Daneel `templates/daneel-executor-task.md`

Tasks:

- Enumerate `net_allow` from Daneel's host source and record the exact hosts and source locations; do not copy a remembered list.
- Configure caps `[IO, FS, Clock, Net, AI, Process]`, the three read-only git commands, no Env, `${WORKSPACE}` sandbox, HTTPS-only network, full default CLI set plus `lock`, and the grounded-search-compatible AI provider.
- Register `daneel-executor` as `acknowledge_only: true`, work tier 2, with executor model `glm-5.3-flash` and workspace-resolved template.
- Write the template with Request schema, scratch-package/lock workflow, package-context checking, final `ANSWER:` convention, and malformed/out-of-policy `BLOCKED:` convention.
- Push and verify template deployment via the trigger path fixed by `57099e2`; no image rebuild.
- Run the path-only dependency/lock/admission probe inside the actual agent container.

Acceptance criteria:

- `ailang coordinator agents daneel-executor` shows declared and effective policy values.
- Banked task data contains the expected non-empty policy digest.
- Allowed git commands work; `git push`, arbitrary Process, Env, and an unlisted domain are refused.
- A clean answer completes rather than reporting `no_changes`.
- A template-only follow-up push becomes visible to the next task without an image rebuild.

### M5 — Implement Daneel host dispatch, completion consumption, and `ext/search`

**Estimate:** 250 implementation + 140 tests/smokes LOC; 1 day
**Owned repository:** sunholo-data/daneel
**Example files:** `ext/search/ailang.toml`, `ext/search/register.ail`, `ext/search/_smoke.ail`, host dispatch/completion consumer files, `skills/daneel-capability-SKILL.md`

Tasks:

- Run `ailang prompt` before writing `.ail`, then confirm the current `ext/abi/types.ail` Request/Authority/Outcome/Host shapes.
- Create `ext/search` with Authority caps `[AI, Net]`; use `std/ai.searchGrounded` and preserve answer text plus source metadata for the host response.
- Add the capability rung: compose a well-formed Request, borrow host `dispatch()` to the fixed `daneel-executor` inbox, and return a dispatched Outcome.
- Consume completion JSON: extract the final `ANSWER:` block from `summary`, retain the transcript artifact reference for overflow/debug, mail success via the lent host function, and mail structured blocked/failed reasons.
- Add package smoke programs and host parser tests for completed, blocked, failed, missing-answer, truncated-tail, and malformed payload cases.

Acceptance criteria:

- `ext/search` declares exactly `[AI, Net]` and its smoke returns answer text plus at least one fixture source.
- Capabilities cannot select model, repository, arbitrary inbox, domain, or mail recipient through this rung.
- Malformed Request is dispatched only far enough to produce the required `BLOCKED` completion; it is never treated as a human prompt.
- Completion parsing is deterministic and never treats absent `ANSWER:` as success.

### M6 — End-to-end probe, negative boundaries, and implementation record

**Estimate:** 90 probes/docs LOC; 0.5 day
**Owned repositories:** AILANG, multivac, and Daneel
**Example files:** implementation verification log in the moved design doc and relevant guide/changelog entries

Tasks:

- Dispatch Mark's first-use scenario: an `ext/search` request whose AILANG program asks Gemini to perform Google Search grounding.
- Assert task NDJSON, policy admission JSON, provider request trace/recorded shape, completion payload, answer mail, model used, sources, tokens, and cost.
- Probe malformed Request → `BLOCKED`; unlisted domain → `E_NET_DOMAIN_BLOCKED`; Env → policy violation; `git push` → Process refusal; `git status/diff/log` → admitted.
- Modify an `ext/` or template file between two tasks and prove the second checkout sees it without an image rebuild.
- Update stale pre-ratification Process statements, document enumerated hosts and release version, then move the design doc to implemented only after every assertion is banked.

Acceptance criteria:

- One live grounded answer with source metadata reaches the requester through D1's summary contract.
- All positive and negative authority assertions are preserved as reproducible evidence, never categorized as generic `api_error`.
- `make test`, `make lint`, `make check-boundaries`, and `make simplicity-audit` pass in AILANG; Daneel package smokes pass.

## Day-by-Day Plan

| Day | Work | Exit condition |
|---|---|---|
| 1 | M1 and start M2 shared contract | Policy tests green; grounded API/type contract compiles |
| 2 | Finish M2 provider mapping, fixture, refusal tests, example | Gemini fixture and all non-Gemini refusal tests green |
| 3 | M3 release gate, then M4 config/template | Released binary verified before any multivac rollout |
| 4 | Finish M4; implement M5 | Effective config visible; Daneel smokes and completion parser green |
| 5 | M6 live and negative probes; docs/implementation record | Grounded answer mailed with sources and all boundary evidence banked |

## Success Metrics

- Six milestones populated in sprint state with no placeholders.
- Approximately 1,650 LOC maximum planning envelope; variance above 25% requires a plan note.
- New grounded surface has provider fixture coverage, non-Gemini refusal coverage, a runnable example, and frozen stdlib interface coverage.
- M1's four policy cases plus provider-escape case pass.
- D2's exact positive and negative capability/process/network matrix passes.
- `ext/search` Authority is exactly `[AI, Net]`.
- A real host-dispatched Gemini Google Search answer returns normalized sources and reaches mail through `summary`.
- No new environment-variable route, no image rebuild for template changes, and no configuration deployed before its binary release.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Executor model D3 and grounded-call provider are conflated | Keep registry executor model and AILANG handler provider explicit; prove cross-provider policy enforcement before deployment. |
| Grounding metadata differs across Gemini API versions | Normalize only stable source fields; bank a sanitized recorded fixture and preserve unknown provider data only if bounded. |
| Dedicated grounded call expands shared provider interfaces | Use one explicit operation with default typed refusal; verify ordinary request bytes remain unchanged. |
| Unknown multivac config keys disappear silently | Hard M3 release/version smoke gate before M4. |
| Design examples retain the rejected no-Process proposal | M6 includes a doc consistency audit and positive tests for the three allowed git commands. |
| Summary tail truncates the answer or sources | Template pins final `ANSWER:` within approximately 1,100 bytes; normalized sources are concise; transcript artifact remains overflow/debug evidence. |
| Live Gemini fixture contains secrets or unstable data | Record through a sanitizer, inspect before commit, and use deterministic fixture replay in tests. |

## Approval and Handoff

This plan is ready for human review. Per repository workflow, implementation begins only after the user says **execute sprint**; approval of this planning task alone does not authorize code or deployment changes.
