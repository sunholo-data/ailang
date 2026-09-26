# M-MISSION-ROLE-DISPATCH — First Runtime Slice

**Status:** Approved under Mark's attended 2026-09-07 instruction to get the project running; execution in an isolated worktree.
**Contract:** [M-MISSION-RUNTIME-CONTRACT](m-mission-runtime-contract.md), delivery slice 1.
**Scope:** Opt-in `ailang mission role-run --request FILE --receipt FILE [--dry-run]`.
**Estimate:** 2–3 development days (~900 implementation/test lines), medium risk. Milestones run sequentially.

## Current state and reuse audit

The main checkpoint is 878939117 (preceded by inherited-work checkpoint 8c41d41d4).
Baseline `go test ./internal/mission ./internal/modelreg ./internal/ai -timeout 90s` passed.
The supplied velocity script exited 141 (head/SIGPIPE); direct scoped git history shows repeated
mission increments on September 6–7, but aggregate multi-agent LOC is not an individual velocity.
The estimate is a planning allowance, not a productivity claim.

Reuse `executor.Executor`, `ExecutorFactory`, `Task`, capabilities, cost budget, and provider adapters.
Model rows already identify harness and wire model via `GetExecutorForModel`. The existing registry
role chains are a historical cloud transcription and mission adoption is parked: THIS command
therefore requires an explicit ordered list of friendly model keys in its request. It does not
silently activate those historical defaults or alter the live driver's environment/skill rules.

`ModelConfig.Provider` identifies a transport, not necessarily the model vendor. Add an optional
`model_vendor` fact and one conservative resolver: explicit vendor wins; direct known vendors and
OpenRouter's vendor-prefixed API name can resolve; unknown origins fail loudly. This is model
metadata, not a new role-policy catalog. Different-vendor independence is the ONLY judge policy
in this first slice; it is stronger than merely comparing friendly names or hosting routes.

## Frozen first-slice contract

- Versioned JSON request: mission/work-item/stage/attempt IDs; role; absolute existing workspace;
  input revision; instructions; explicit friendly model candidates; author model keys for evaluator;
  total timeout (1–1800 seconds); positive token and USD ceilings. Unknown JSON fields rejected.
- Caller is an attended/trusted local caller. This command is NOT an untrusted-message consumer,
  authorization service, sandbox, or automatic coordinator advancement path. The caller supplies
  an isolated workspace; provider adapters retain their existing permissions. A declared revision
  is input provenance, not a claim this first slice checks a Git checkout or captures a snapshot.
- Resolve all model rows before dispatch. Unknown/ambiguous origin, missing harness/model,
  local-GPU route (no rig lock integration yet), and evaluator without resolvable authors fail.
  Record each independence/capability rejection with its reason. Never invent an Agent wrapper.
- Evaluator routes must differ in origin vendor from EVERY declared author; apply before each
  fallback. Vendor identity is registry evidence, not cryptographic provider attestation.
- A candidate can fall through only BEFORE ExecuteStreaming: factory/capability/health failure.
  Once execution starts, error/timeout/empty output stops the attempt; no automatic rerun over a
  potentially modified workspace. Cancellation stops dispatching further candidates.
- First-slice adapter admission is limited to Claude, Codex and Pi after review.
  Known wire-route/vendor contradictions fail before dispatch; unsupported/local routes skip.
- The executor receives exact wire model, instructions and parent IDs, bounded context, token/USD
  budget, and canonical command-scoped messaging bindings. Token guards count fresh input
  plus output at usage-event boundaries. Live USD guards estimate those tokens at registry
  prices, excluding cache charges; final metered cost is checked separately. These are
  adapter guards, not hard billing ceilings or fleet quota reservations. Reject ambient Anthropic/OpenAI API
  credentials for subscription-oriented claude/codex routes instead of silently changing billing.
- Request SHA-256 binds the full normalized request. A versioned JSONL receipt is created with
  exclusive create before ANY execution. The parent directory is synced before dispatch; unsupported filesystem sync fails closed.
  Events are appended and synced before dispatch and after
  completion; an existing receipt cannot be overwritten/replayed. This is a local attempt journal,
  not a second task-state authority. Crash recovery/leases/CAS remain slice 2.
- Success means `execution_completed`, with `artifact_verified=false` ALWAYS in this slice.
  Failed finish reason, empty output, nil result, deadline, and budget termination cannot succeed.
  Preserve requested/selected route, session ID, actual adapter name, finish reason, token usage,
  cost provenance (including unknown), and output. No evaluator score interpretation or merge.
- Dry-run resolves and reports without executor construction, probes, receipt writes, or spend.
  Receipt is required only for actual run. Plain JSON stdout; errors return nonzero.

## ✅ M1: Requests and conservative model identity

**Estimate:** 180 implementation + 180 test LOC; day 1.
**Files:** internal/modelreg/identity.go and identity_test.go; ModelConfig field in models.go;
internal/mission/dispatch/request.go and request_test.go.
**Acceptance:** malformed version/IDs/budgets/fields rejected; request digest stable and changes
when instructions/input change; OpenRouter/Ollama hosting does not masquerade as model origin;
authors missing/unknown rejected; explicit candidate order preserved and local GPU denied.
**Example:** request JSON under examples/mission/ with a cloud Pi evaluator and OpenAI author.

## ✅ M2: Dispatch and receipts

**Estimate:** 260 implementation + 220 test LOC; day 2.
**Files:** internal/mission/dispatch/run.go, receipt.go, corresponding tests; focused Pi adapter token-budget regression fix.
**Acceptance:** fake factory invokes chosen actual adapter exactly once; health/capability failure
selects the next compatible candidate; same-vendor fallback never executes; active execution failure
never retries; cancellation stops work; receipt failure stops launch; exclusive receipt prevents
repeat execution; nominal output, nil/empty/failed result and cost/timeout signals discriminated.
**Confirmed adapter gap:** Pi accepted `MaxTokensPerBench` without enforcing it. A failing
fixture test demonstrated success with 685 tokens against a limit of 1. Add termination
and a non-success result at this existing adapter boundary.

**Integration:** actual Pi adapter driven by an isolated fake executable emitting its real NDJSON
fixture grammar, with no model/network call; prove exact wire argument and structured output flow.

## ✅ M3: CLI, example, independent review

**Estimate:** 100 implementation + 100 test LOC; day 3 including 25% uncertainty buffer.
**Files:** cmd/ailang/mission_role_cmd.go and tests; mission_cmd.go command/help entry;
examples/mission/evaluator-request.json; docs/docs/guides/mission-role-dispatch.md; changelog.
**Acceptance:** dry-run works through the built binary, strict flags/JSON/trailing input handling,
CLI receipt errors and failed execution return nonzero; focused packages/race/vet/build/boundary
checks pass; independent evaluator audits the exact diff against this plan and fixes are rechecked.
The new command is opt-in; no launchd, existing role chain, or fleet config is modified.

## Verification and limits

Run focused tests before/after, targeted race tests, Go vet and repository lint/boundary/file-size
checks as applicable. Run the required full make test/make lint baseline/final checks with bounded
execution and report environmental/pre-existing failures separately; do not fix unrelated owners'
work to manufacture a green whole-repo report. Fixtures are repository-local and portable; actual
shell fixture integration skips Windows with an explicit reason while pure dispatch tests run there.

No live provider call, workflow A/B, cloud deployment, push, or fleet cutover is part of this sprint.
Success here is a tested reusable dispatch seam and a working opt-in CLI, not a claim the entire
mission runtime or four mission migrations are complete.

## Delivery evidence (2026-09-07)

Independent engineering review: PASS, 92/100, in a separate agent session. It found
and drove fixes for wire/vendor contradictions, unsupported adapter admission, and
receipt directory durability. This was not a provider-backed cross-vendor evaluation.
The local-first/cloud-fallback regression requested by the reviewer also passes.

Full tests, lint, build, architecture boundaries, focused race checks and vet pass.
The built binary resolves the checked-in evaluator example without a provider call.
The file-size check flags only `cmd/ailang/exec.go` at 807 lines; that file is unchanged
from the inherited checkpoint (878939117). Pi remains at the 800-line limit.

The complete runtime contract remains planned: no live fleet cutover or workflow
experiment was performed. Next: durable coordinator-backed mission state and recovery,
then an isolated canary, before project onboarding and migration.
