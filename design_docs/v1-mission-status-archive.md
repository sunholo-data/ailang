# V1 Mission — STATUS stamp archive (rotated out of the charter)

Newest first. Rotation rule lives in the charter's STATUS section. Full per-iteration
detail is in v1-mission-log.md — these are the headline stamps only.

## STATUS 2026-07-14 (midday) — ITERATION 29: m-dx-examples-coverage LANDED (PR #392 `3d451947c`, all 3 workflows green observed) + FIRST LIVE QUORUM (5 rounds, 5 real catches, ~$0.16)

The clause-3 queue head shipped end-to-end headless: the stale v0.10.1-era doc was re-scoped on
live HEAD data, then became the **first live Tier-1 design-quorum subject** — 5 reject-by-default
rounds, EVERY objection a real spec gap (installed-binary path resolution; downloader-manifest
premise; CI step lifecycle; modules-field drift enforcement; parser-backed extraction; the
`known-broken` status that didn't exist). Result: the Opus planner found ZERO premise
discrepancies (first time in 5 iterations — the quorum front-loaded the corrections mid-sprint
planners had been making). Shipped: 5 red examples quarantined under issue #386 (bisect
inconclusive→quarantine per decision rule; real trigger root-caused: `show()` in effectful-lambda
interpolation collapses effect rows — fix forbidden in-sprint, routed to #386); 6 new stdlib
examples (all 6 zero-importer modules covered); the triple-defeated verify-examples gate made
REAL (3 `|| true` layers fixed + self-test + `validate_manifest --ci` wired, non-vacuity proven
both directions); `docs --examples` un-inert (manifest `modules` field, parser-backed backfill +
drift lint, installed-binary test). Evaluator round 1 FAIL 81/100 on ONE Windows path-separator
defect → one-line hardening `881711325` → all checks green incl. both Windows jobs = round-2
PASS. Nightly "regression" (state_machine_vending) RULED OUT as model variance — yesterday's
passing solution compiles clean at HEAD. Quorum frictions recorded: no termination rule
(reject-by-default can block forever; controller synthesized after round 5 with recorded
dissent), gemini-3-1-pro unreachable 3/5 calls. Detail: log entry 32.

## STATUS 2026-07-14 (evening) — ITERATION 30: m-dx-ai-discovery LANDED (PR #393 `c07c36b25`, eval PASS 93/100 round 1) — a RESUMED iteration; interleave: dev-red from two sibling merges fixed forward (3 causes)

The last clause-3 Prelude/discovery starter is in: `ailang docs --all-functions [filter]`
(one grep-able line per stdlib export, AST-rendered signatures — also fixing the V16
effect-row truncation in per-module docs), unknown-stdlib-module recovery (`import std/time`
→ `did you mean: std/clock?` + module list; curated alias table + Levenshtein≤2 reusing
importhint), and `ailang docs prelude` (rendered from live mechanisms, bidirectional drift
test). NOTE this iteration was a RESUME: the 15:30 scheduled run re-scoped the doc, ran the
quorum-refined plan, and died rc=1 (transient Anthropic error, pre-dating the 17:16
driver-retry fix) with uncommitted executor work in the sprint worktree; this run detected the
mid-flight artifacts at Gate 2, verified them (all KEEP), and completed execution. Evaluator
round-1 PASS 93/100; hardening `ea6069815` (arrays→array alias misdirection) + `0ad27444c`
(Windows separator in the docs-search guard). INTERLEAVE (Gate-1 red-dev rule): sibling
M-STD-YAML/M-SMT merges turned dev red mid-iteration with THREE distinct causes — missing
builtin golden regen, z3-less Windows runner on ungated verify e2e tests, and >800-line
file-size overflow — fixed forward direct-to-dev `9a314772d` + `4caddfd23` (mechanical split
of the sprint's own additions into verify_callee_gate.go / codegen_sig_sorts.go). Retro:
sprint-executor gains the Windows-proofing core principle (3 same-class frictions recorded);
"all-skipped PR checks = conflict, poll mergeable" saved to memory (friction #1, no skill
edit yet).

## STATUS 2026-07-14 — ITERATION 28: fleet Phases A+B LANDED (Mark's mission-infra interleave) → PR #383 `1186a48e6`, eval PASS 94/100 round 1; design-doc QUORUM live (`ailang design-review`/`design-quorum`), Phase A found already-deployed at Gate 2

Mark's prioritized fleet slice is DONE: Phase A (quota-aware multi-candidate model probing in
the driver) was found ALREADY LANDED+DEPLOYED at Gate 2 (`3bee6b6df`, direct-to-dev by the
2026-07-14 interactive session ~3h before planning — the sprint re-scoped to verification, six
driver safety invariants confirmed intact). Phase B shipped as new `internal/mission/quorum` +
`ailang design-review`/`design-quorum`: N-reviewer design-doc quorum (gpt-5.6-sol via
OPENAI_API_KEY + gemini-3-1-pro via Vertex ADC — the GEMINI_API_KEY-absent risk mitigated and
live-proven at $0.002/call — + the Claude controller in-session), reject-by-default with
required strongest-objection, N−1 graceful degrade with NAMED absences (never silent), budget
caps with zero-spend pre-flight refusal, JSON artifact + mission-log block (seed data for Phase
E). Full quorum live test: $0.0074. Evaluator round-1 PASS 94/100 (independent live re-runs,
spend figures reproduced within 3%, prompt-injection probe held); 4 warts hardened pre-merge
(`027523b44`). Phases C/D/E remain queued opt-in. Detail: log entry 31.

## STATUS 2026-07-12 (morning) — v1.0 SCOPE SET via full backlog triage (Mark, interactive)

Triaged all 69 non-gating planned docs (5 parallel reality-check agents). Result: the "~85
backlog" is really **~12 ghosts** (already shipped, headers stale) + **~30 not-v1.0** (eval-infra
rig/harness, cloud-infra, motoko-fork, post-v1) + **~18 genuine v1.0 candidates**, almost all
clause-3 accessibility. Mark chose FULL SCOPE: the whole clause-3 footgun/DX/prompt cluster +
both DX tooling investments (fmt, deterministic-tooling) + the full clause-4 orchestration surface
are IN; rig/cloud/motoko stay OUT. v1.0 queue ~14 → ~33 open items (clause-grouped below).
10 confirmed ghosts reconciled to implemented/; 2 conflicted docs kept OPEN pending repro. Full
evidence: log entry 10.

## STATUS 2026-07-11 (evening) — CONTROLLER MODEL: Fable → Opus (quota relief, Mark)

The outer-loop controller runs on **Opus** (`claude-opus-4-8`) through a **time-boxed override**
that AUTO-REVERTS to Fable at **Mon 2026-07-13 07:00 CEST** (when Fable quota resets) — no session
or human needed. Mechanism: driver default is Fable; `~/.ailang/state/mission-model` holds
`claude-opus-4-8 <expiry-epoch>`; the first iteration past expiry deletes it, falls back to Fable,
and posts to #329. Reason: Fable quota relief (Mark). Both paths test-verified. Consequence in the
routing table: while on Opus, evaluation is model-homogeneous (Opus judges Opus) — mitigation
available (distinct-model skeptic evaluator) but left off. To go back to Fable EARLY (more quota
sooner): `rm ~/.ailang/state/mission-model`. To extend Opus: bump the epoch in that file.

## STATUS 2026-07-14 (night) — ITERATION 27: `m-prelude-option-result` VERIFIED REAL + EXECUTED + LANDED (round-1 PASS 98/100, mission high) → PR #382 `d26215341`; Option/Result now prelude in entry modules

The #1 structural AI-DX friction (6% of recent compile failures, "forgot `import std/option`")
is CLOSED: entry modules get implicit lowest-precedence std/option + std/result imports at ONE
loader call-site consumed by both compile and runtime; explicit imports + user-local types
shadow cleanly; library modules unchanged (still explicit, self-documenting). The planner's
verify-before-planning CORRECTED the design doc's whole mechanism (its `InjectPreludeValues`
path never existed) — third consecutive iteration where doc-claim verification changed the
plan. Full inner loop round-1 clean: Opus plan → Opus execute (15 new tests, 0 deletions) →
independent Fable evaluation (model diversity RESTORED — controller back on Fable; 20
adversarial probes, PASS 98/100). Bonus closeout: m-prompt-option-none-idiom SUPERSEDED
(the band-aid this fix retires) → archive/. Detail: log entry 28.

## STATUS 2026-07-14 — ITERATION 26: `m-xmod-alias-poly` VERIFIED REAL + EXECUTED + LANDED (round-1 PASS 93/100, first zero-correction pass) → PR #381 `fd1b11a47`; parameterized type aliases now instantiate

The clause-3 parser/type footgun row is CLEARED: `type Box[a] = { items: [a] }` used as
`Box[int]` now instantiates (single- + cross-module), with `TC_ALIAS_ARITY_001` on arity
mismatch and ADTs proven nominal (strict alias-env keying). Gate-2's VERIFY-FIRST probe
confirmed the bug REAL at HEAD in 10 minutes — and caught the NEW-DOC tag WRONG again (full doc
at planned/v0_29_0; 2 of 2 recent NEW-DOC tags wrong → Gate-3 grep rule added to the skill).
Full inner loop: Opus plan (all 3 doc root-cause claims verified against live code first) →
Opus execute (3 milestone commits, 25 new tests, 0 deletions) → independent Fable evaluation
(12 adversarial probes, non-vacuity both directions, PASS 93/100 round 1 — no corrections
needed, a mission first). Bonus DX: wrong-body programs now get precise field-level type errors.
Detail: log entry 27.

## STATUS 2026-07-13 (evening) — ITERATION 25: R4a+R4b exposed as GHOSTS at Gate-2 (guards PR #379 `ea8116f83`) + `m-lambda-open-record-pattern` EXECUTED + LANDED (round-1 PASS 92/100 + hardening) → PR #380 `47576e25d`; `{name, ...}` in lambda params now truly OPEN

Gate-2 reality check paid for the whole iteration: the clause-3 "NEW-DOC footgun" cluster was
sourced from a strategy review whose own Verification Log said the rows were never individually
re-verified — live probes showed R4a (match-in-HOF-lambda) and R4b (poly-arith-lambda) were
ALREADY FIXED (R4a's doc archived Not-Applicable back in May; R4b fixed v0.7.0). Both closed
with CI-enforced guard examples. The one REAL row, m-lambda-open-record-pattern, ran the full
inner loop same-iteration: the executor found the true primary cause (unifyRecord's pre-row
field-count rejection) was absent from BOTH the design doc and the plan; the independent
evaluator caught an arm-order-dependent acceptance that got hardened before merge. Remaining
clause-3 starter: m-xmod-alias-poly — tagged VERIFY-FIRST (3 of 5 cluster rows were
ghosts/mislabeled). Skill edit: Gate-2 now mandates live-repro before routing any survey-sourced
row. Detail: log entry 26.

## STATUS 2026-07-13 — ITERATION 24: `m-public-feedback-delivery-audit` (Mark's NEXT-FIRST) EXECUTED + LANDED (round-1 clean, eval PASS 97/100) → PR #378 `4fee247a8`; live prod verification PARKED for Mark (daemon reload)

The feedback flywheel's blind spot is code-fixed: Defect A — `pkg:*` package-feedback inboxes now
tagged `EventType: "public-feedback"` (🌐, Discord-accepted) via `isExternalFeedbackInbox`;
Discord allow-list untouched, internal traffic still dropped. Defect B — daemon dual-subscribe:
`Daemon` refactored to N message sources; prod fetcher scoped via new
`firestore.NewClientForProject` (explicit project, no env mutation); opt-in
`extra_message_envs: [prod]` / `--also-subscribe prod`, default OFF byte-identical. Plan-stage
reality check corrected the design doc (dual-subscribe is a real multi-project fan-in, NOT "structural,
not novel") and KILLED two feared ops steps (prod sub `ailang-messages-laptop` already exists; ADC
owner on both projects — no Terraform, no IAM). Fable evaluator PASS 97/100 round 1 (base-binary
non-vacuity both defects; conflict surface fully intact; executor's local-docs-build claim
confirmed pre-existing but re-rooted: CI-generated `packages/sunholo/*` pages, not
`reference/errors/*`). ⚠ HUMAN (Mark): live end-to-end proof needs the daemon reload —
`extra_message_envs: [prod]` in daemon.yaml (or plist `--also-subscribe prod`), launchctl
reload, then the 2 prod test-sends — exact checklist in the sprint plan §Parked-for-human and
docs/docs/guides/notify-daemon.md. Detail: log entry 25.

## STATUS 2026-07-13 (earlier) — ITERATION 23: `m-module-let-func-resolution` EXECUTED + LANDED (round-1 clean, eval PASS 98/100) → PR #368 `fd38ec14e`; dev CI-red (gofmt) fixed forward; ⚠ NEXT-FIRST missed at pick — iteration 24 HARD-PINNED to m-public-feedback-delivery-audit

Full inner loop headless: CI-red fix first (gofmt miss from `366c5bbb2`, 2 red runs → `39171a4f9`
observed green) → Opus plan (caught the doc's wrong #327-matrix test path: `internal/pipeline/`,
not `internal/types/`) → Opus execute in worktree (M0 spike GO → unified SCC over lets+funcs,
wrapInLets DELETED; module letrec via core.LetRec; MOD007 dup-name hard error; truthful hint) →
independent **Fable** evaluator (diversity restored) PASS 98/100 round 1 w/ base-binary
non-vacuity + adversarial probes. Module lets now resolve module funcs uniformly — the 4th
position-divergence family member CLOSED at the decl-class level. Sibling-agent Build-and-Release
red at `b293331f2` triaged = `TestReferenceSolutions_JS/fizzbuzz` Windows 60s-timeout infra flake
(rerun green; dev-health ledger). ⚠ Gate-2 friction recorded: Mark's [NEXT-FIRST]
m-public-feedback-delivery-audit (added 13:04, pre-session) was missed at pick time — caught
mid-flight, sprint already through eval → landed it; iteration 24 MUST take the NEXT-FIRST.
Detail: log entry 24.

## STATUS 2026-07-13 (earliest) — ITERATION 22: nightly-regression triage → REAL resolver gap found (#366); `m-module-let-func-resolution` DOC-READY → PR #367 `7c0d91c4c`

Gate 0.4 fired: 2 fresh nightly regressions (opencode-qwen3-5) outranked the queue. Triage
(data-led, error-stream first): `adt_option` = thrash_aborted ~8% over token cap, 2-trial noise,
no action; `higher_order_functions` = model dialect variance (module-level lets, never attempted
yesterday) — NOT a binary regression, BUT the shape exposed a real pre-existing decl-class gap,
live-reproduced 10 ways at HEAD: **module-level `let`/`letrec` values can NEVER reference module
`func`s** (any order; letrec can't self-reference), while the hint cites the CLOSED #327 with a
no-op workaround — an agent following it loops forever (which is exactly what the model did).
Mechanism read from code: funcs-only `BuildCallGraph` (scc.go:111) + `wrapInLets` checking let
values outside every func binding (file.go:279–302). 4th member of the #323/#327
position-divergence family, at the decl-class level. Filed #366; design doc (unified SCC over
lets+funcs, delete wrapInLets, duplicate-name pinning, truthful hint) → PR #367 squash-merged
`7c0d91c4c` on observed-green checks. Controller back on Fable (Opus override expired on
schedule). Neural doc-search skipped (qwen3.6 eval-suite held the GPU — iteration-20 precedent).
Next: EXECUTE m-module-let-func-resolution (Phase-1 spike gates approach), then R4a. Detail: log
entry 23.

## STATUS 2026-07-13 — ITERATION 21: clause-3 R4c `m-arity-style-diagnostic` EXECUTED + LANDED (full inner loop, round-1 clean) → PR #363 `5b54509d1`

Executed R4c exactly as iteration-20's Next specified: Opus **sprint-planner** (resolved the #1
risk — direction pinned from `inferApp`'s constraint construction: `fp1`=declared/EXPECTED,
`fp2`=call-site/ACTUAL; found 2 design-doc premise errors: `TestCurriedMismatchStillFails` asserts
only `err != nil` [the doc's "one intentional test-text change" was fictional], and both cited
example fixtures don't exist) → Opus **sprint-executor** (worktree off origin/dev: `TC_ARITY_001`
const + `arityMismatchMsg` helper inline-coded per the TC_REC_00X convention, wired at the
post-curry-flatten `else`; 5 golden/regression tests; docs + CHANGELOG) → **Fable evaluation
PASS 97/100 round 1** (independence restored — mission-model override expired on schedule) with
independent reproduction on an evaluator-built worktree binary + base-binary non-vacuity proof.
All 3 arity footguns (partial-app/too-many/too-few) now emit code + direction + `Suggestion:`
(the under-supply hint names AILANG's no-partial-application rule); positive controls (2-arg,
curried↔tupled) unchanged. PR #363 → squash-merge `5b54509d1`, required checks green + post-merge
dev CI verified. Design → implemented/v0_30_0. Retro skill-fix: design-doc-creator gained claim
class 3 (cited regression fixtures must exist; test-behavior claims read from the test body — 2
same-class frictions this iteration). Next: R4a `m-dx-match-hof` (design-doc-creator). Detail:
log entry 22.

## STATUS 2026-07-13 — ITERATION 20: clause-3 R4c `m-arity-style-diagnostic` DESIGN DOC (full inner-loop NEW-DOC, design stage) → PR #361; also unwedged a broken main-tree autostash left by the os-rotation cron

Picked the queue's `[NEXT]` R4c (cheapest clause-3 footgun fix), routed through **design-doc-creator**
(design stage of a full inner-loop NEW-DOC sprint). **Gate 0 first had to unwedge the shared main
tree**: an interrupted `git pull --rebase --autostash` (from the sibling os-rotation data cron) had
left `.git/AUTO_MERGE` + a `both-modified` conflict on BOTH mission docs. Resolved **losslessly** —
the `Stashed changes` side was an *empty deletion* of already-landed iteration-19 content, so
`git checkout HEAD -- <both docs>` (= origin/dev, the rich side) lost nothing; removed the vestigial
`AUTO_MERGE` so the next data-cron pull won't re-collide. Left the cron's 5 staged data files
untouched. Gate-1: local dev == origin/dev `d6b22b75d` (no stale-local this iter); dev CI **green
per-workflow** (CI/Build/Docs all success @ d6b22b75d). Inbox: 3 routine (eval-suite ×2 @ 85%,
self-note) — none outranked the queue. **Reality-check + design (all live-verified, binary
d6b22b75d)**: no prior doc/PR; the 3 arity footguns (partial-app/too-many/too-few) all emit the same
weak `arity mismatch: 2 vs 1` — no error code (clause-3 gate unmet), no direction, no suggestion.
Mechanism traced: emission = bare `fmt.Errorf` at `unification_types.go:39` (post curry-flatten
`else`); plain-`%w` wrap at `inference_helpers.go:187` (why no Suggestion renders); no `errors.As`
recovery of `*TypeCheckError` anywhere → fix must embed code+hint INLINE (matches `TC_REC_00X`).
`TC_ARITY_001` confirmed free. Design: allocate the code + emit coded/directional/style-aware text,
NO arity-semantics change. Shipped the doc via a worktree off origin/dev → **PR #361** (auto-merge
SQUASH). **Design only — execution (M1/M2, ~1–2d) queued next iteration.** Next: EXECUTE R4c
(now `[DOC-READY]`) or start R4a `m-dx-match-hof`. Detail: log entry 21.

## STATUS 2026-07-13 — ITERATION 18: m-dx-record-cons-pattern + m-dx-tapp-trecord-unification BOTH GHOSTS — verified-closed + CI-regression-guarded (clause 3, VERIFY-then-route)

The clause-3 **VERIFY-then-route** pair. Ran each doc's repro LIVE at HEAD (`v0.29.2`): both bugs are
**ghosts** — already fixed by prior record-pattern/`::`-sugar + type-unification work. `{text: s, bold:
b} :: rest` type-checks (infix + canonical); `makeTable(rows: [[TableCell]]) -> Block` type-checks in
both `func` and `let`-lambda forms. The doc's `cannot unify type application with *types.TRecord` (`%T`)
error was reworded to `.String()` (`5cf6287bf`); the surviving fallback is for unrelated type apps only.
NO inner-loop skills invoked — Gate 2 "already done → bookkeeping" path; deliverable is verification +
a CI-enforced regression guard (the record-cons doc flagged its OWN missing-test gap). Shipped one
worktree commit: parser test `TestListConsPatternWithRecord` + two runnable examples
`record_cons_pattern.ail` (→ `hello`) / `record_list_extraction.ail` (→ `headers: 2`), both gated by
`verify-examples-toplevel` in CI; both docs → implemented/v0_30_0; CHANGELOG. Zero Go production-code
change. PR #358 → squash-merge `adde9e9d0`, required checks green (bounded 2-chunk poll, 30-min cap);
post-merge dev CI green (one `gh run rerun --failed` cleared a transient `proxy.golang.org` deps
flake in the `CI` job's install step — NOT code; all 3 workflows green on `adde9e9d0`). Retro: no
skill/process edit (see log 19). VERIFY-then-route
backlog now EXHAUSTED. Next: NEW-DOC footgun fixes (R4c arity-style / R4a match-hof) via
design-doc-creator — full inner-loop sprints. Detail: log entry 19.

## STATUS 2026-07-12 — ITERATION 17: m-dx-split-argument-warning BUILT + LANDED (clause 3 — compile-time non-blocking warning for the reversed-`split` footgun)

Full build loop headless, round-1 clean. Reality-check verified the footgun live (`split("/", name)` →
`["/"]` silently) and caught the doc's stale premise: there is NO generic "warning infrastructure" — the
only one was exhaustiveness-specific (`[]*elaborate.ExhaustivenessWarning`). **Conflict Surface** → M1
generalized it to an `elaborate.Warning { String() string }` interface (render sites call `.String()`,
untouched). Opus plan → Opus executor (worktree) → **independent Opus evaluator PASS 97/100 round 1**,
no blockers, non-vacuity reproduced TWO ways (arg0/arg1 flip breaks trigger tests; a user-`Var` match
branch breaks the module-guard test — proving `std/string.split` is genuinely distinguished from a
user-defined `split`). Extensible `swapTraps` table (only split armed); detection runs over final
module Core so it fires on cold AND cache-hit compiles, surfaced by both `run` and `check` (bonus:
`ailang check` previously dropped ALL pipeline warnings — now fixed). Heuristic: arg0 a 1–3-rune string
literal + arg1 not a literal → warn; non-blocking (program still runs, exit 0). PR #356 →
squash-merge `8339b6421`, required checks green (auto-merge); post-merge dev CI green. Runnable example
`split_argument_order.ail` (Phase-2 teaching in header — embedded prompt NOT edited, prompt-diet gated).
Design → implemented/v0_30_0. Retro: no skill/process edit. Next: NEW-DOC footgun fixes (m-dx-match-hof
R4a) via design-doc-creator, or VERIFY-then-route m-dx-record-cons-pattern. Detail: log entry 18.

## STATUS 2026-07-12 — ITERATION 16: m-dx-json-bool-coercion BUILT + LANDED (in-repo half, clause 3 — `std/json.asBoolLoose`; Phase-1 firestore-package fix parked out-of-repo)

Reality-check split the item: `std/json` already round-trips booleans (`jb`+`asBool` verified live), so
the doc's "encode/decode is broken" premise is stale — the real bug is wrong-constructor use
(`js("true")` = a JSON string) in `sunholo/firestore/fields.ail`, which lives in the SEPARATE
`ailang-packages` repo (absent here, gated on M-DX-XPKG-RESOLVE) → **Phase 1 PARKED for a coordinator
task**. Shipped the in-repo half: **`asBoolLoose(j) -> Option[bool]`** — the system-boundary (A12)
decoder that accepts `JBool(b)` OR `JString "true"/"false"`, else `None` (structured failure, never a
silent default — CP2), for APIs (Firestore `booleanValue`) that may return a boolean stringified.
Purely additive (no existing json fn touched). Runnable example `json_bool_encoding.ail` (carries the
`jb`-vs-`js` teaching — embedded prompt deliberately NOT edited; prompt-budget GATED) + Go regression
`json_asboolloose_test.go` (7-case Firestore round-trip; non-vacuity PROVEN — plain `asBool` flips the
stringified cases to NOT_BOOL). Controller (Opus) executed directly given sub-0.5d additive scope,
then routed an independent Opus evaluator → **PASS 92/100 round 1**, no blockers (rebuilt binary,
reproduced non-vacuity itself, verified out-of-repo parking is honest). Gate 3b: bounded ~19-min CI
poll observed all required checks green → PR #354 squash-merge `5b41b3835`; origin/dev advanced. Design
→ implemented/v0_30_0. Retro: no skill/process edit (candidate watch-item logged: authorize a
plan-skip for sub-0.5d de-scoped mechanical items once a 2nd instance appears). Next:
m-dx-split-argument-warning. Detail: log entry 17.

## STATUS 2026-07-12 — ITERATION 15: m-match-xcheck-error-quality BUILT + LANDED (queue #14, clause 3 — foreign-ctor errors now enumerate transitively-known constructors)

`MatchForeignConstructorError` used to print an EMPTY `<scrutinee ADT>'s constructors are:` line
whenever the scrutinee's ADT was known only *transitively* (e.g. `std/json` returns `Option` but the
module imported only `std/result`). Now it enumerates them (`Option's constructors are: None, Some` +
a `did you mean` hint). Full build loop headless, round-1 clean: Gate-1 origin-sync caught local dev
4 commits behind (iter 14 landed via #350) — read state from origin; reproduced the empty line live;
**Option A** (design doc's recommendation) — a diagnostic-only `Constructor→ADT` registry from all
transitively-loaded ifaces, consulted only when the primary scan is empty, never entering scope. Opus
plan → Opus execute (worktree) → independent Opus evaluator **PASS 96/100 round 1**, no blockers
(base-binary non-vacuity proof + scope-non-leak verified); 2 non-blocking deductions folded into a
hardening commit (scope-leak regression test + collision note). PR #352 → `5aaaff2ed`, required checks
green (auto-merge). Design → implemented/v0_30_0. Retro: no skill/process change (clean round-1, the
sole friction was the already-handled stale-local-dev case). SonarCloud PR gate red = advisory
(non-required; merge succeeded) — flagged for sonarcloud-triage. Next: m-dx-json-bool-coercion /
m-dx-split-argument-warning. Detail: log entry 16.

## STATUS 2026-07-12 — ITERATION 14: m-module-less-run-fail-loud BUILT + LANDED (queue #13, clause 3 — footgun burn-down opened, MOD014)

Module-less `.ail` files (top-level decls, no `module`) now **FAIL LOUDLY** with `MOD014` on both
`run` and `check` — previously they printed `✓ Running` and exited 0 with the entry never running
(a silent-success footgun, CP2 violation). Full build loop headless, round-1 clean: reality-check
caught the doc's proposed `MOD011` was already taken (module-path-collision) → reassigned `MOD014`;
Opus plan → Opus execute (worktree; guard = `len(Funcs) > 0`, the doc's 3-way OR was code-refuted
mid-sprint) → independent Opus evaluator PASS 100/100 round 1 (base-binary non-vacuity proof). PR
#349 → `c2ffd1b5c`, post-merge dev CI green per-workflow. Design → implemented/v0_30_0. Skill-fix:
design-doc-creator gains error-code-allocation + mechanism-claim verification gates. Next: continue
the clause-3 diagnostics (m-match-xcheck-error-quality, …). Detail: log entry 15.

## STATUS 2026-07-12 — ITERATION 13: m-stdlib-url-parse BUILT + LANDED (queue #12, clause 4 — URL-parse half CLOSED)

`std/net` now parses URLs. `parseUrl(s) -> Result[Url,string]` (Err on malformed, never panics —
CP2; `port:string`, ""=absent) + order-preserving percent-decoded `parseQuery` (inverse of
`urlEncodeForm`), both pure builtins (no Net cap) wrapping Go `net/url`. Full build loop headless:
Opus executor → independent Opus evaluator (round-1 FAIL 80/100 on a stale `builtin_types.golden`
→ round-2 PASS 100/100 after regen). PR #347 → `a8628a40c`, auto-merge on green. Design →
implemented/v0_30_0. With regex (#11), v1.0 bar **clause 4's regex+URL-parse gate is now fully
closed**. Next: the clause-3 accessibility cluster. Detail: log entry 14.

## STATUS 2026-07-12 — ITERATION 12: m-stdlib-url-parse DESIGN DOC CREATED (NEW-DOC stage, queue #12, clause 4)

Design doc `implemented/v0_30_0/m-stdlib-url-parse.md` shipped (PR #344 → `7ca58f86f`) — wrap Go
`net/url` (RFC-3986), extend `std/net`, pure `! {}`, Public API `ailang check`-clean:
`parseUrl(s) -> Result[Url, string]` + `parseQuery(s) -> [{name,value}]` (order-preserving inverse
of `urlEncodeForm`). Conflict Surface additive namespace-only. Queue #12 → [DOC-READY]; NEXT stage
sprint-planner → executor → evaluator (~1d, GPU: none). SIDE-STORY: this run booted on a STALE
local checkout and nearly re-did the already-LANDED iteration-11 regex sprint (ran a full redundant
re-eval, PASS 96/100, which now corroborates the landed work) — the Gate-3b `git fetch` caught it
before any duplicate merge. Fix: Gate-1 "sync to origin FIRST" skill edit (2nd instance of the same
gap; see log ## 13).

## STATUS 2026-07-11 (night) — ITERATION 11 COMPLETE: m-stdlib-regex LANDED (AILANG now has linear-time RE2 regex; bar clause 4's regex half closed)

Full loop headless, round-1 clean (sixth consecutive). `std/regex` ships — `compile → Result[Regex,
string]` + total `isMatch`/`findFirst`/`findAll`/`replaceAll`/`split`, backed by Go's `regexp` (RE2,
linear-time by construction; `(a+)+$` < 100ms). RE2 subset (no backref/lookaround) → `compile` Err,
never panics. Opus plan→execute→evaluate; eval PASS 97/100 round 1 with INDEPENDENT reproduction
(CJK rune-span proof of the byte→rune conversion). PR #343 → squash-merge 0b0ed7ea0, all required
checks green. Builtins landed in the modern `internal/builtins/` system (D-ARCH: the doc's
`internal/eval` path was outdated). Next: queue #12 m-stdlib-url-parse (clause 4's other half).
See log entry 12.

## STATUS 2026-07-11 (night) — ITERATION 10: m-stdlib-regex DESIGN DOC CREATED (clause-4 regex, NEW-DOC stage)

Queue #11 (NEW-DOC) routed to design-doc-creator; deliverable is the verified design doc, next
scheduled iteration plans+executes. `planned/v0_30_0/m-stdlib-regex.md` created — a linear-time
(RE2) regex builtin closing the first half of bar clause 4's "linear-time regex + URL-parse
builtins" mandate. **Scope-defining decision: wrap Go's stdlib `regexp` (which IS RE2 —
linear-time guaranteed) instead of hand-rolling an engine → a 2-day builtin, not a multi-week
project.** Two-stage API (`compile -> Result[Regex,string]` then total match fns), RE2 subset
(no backreferences/lookaround — the documented price of linearity), all pure `! {}`. Hard gates
passed: binary rebuilt (v0.29.2-29-gc533bb51c = `git describe`), the `Regex`/`RegexMatch` + 6
signatures `ailang check`-clean; Conflict Surface = purely additive (namespace-collision only,
no grammar); Axiom net +8. Dev CI green per-workflow on HEAD (the two prior tier-drift reds were
fixed by c423490d8, included in the green HEAD run). No skill fix (clean stage). Next: #11
sprint-planner → executor → evaluator.

## STATUS 2026-07-11 (evening) — ITERATION 9 COMPLETE: M-SYNTAX-AI-FORGIVING LANDED (the ~32% small-model parse-failure class is dead)

m-syntax-ai-forgiving landed via the full loop — the first iteration split across two scheduled
runs (run A: reality-check + Opus plan + Opus execute, died pre-evaluation; run B resumed cleanly
at sprint-evaluator from the committed artifacts). The parser now accepts both statement-separator
forms small models naturally write: R1 `;`-sequences in `=` bodies (kills PAR017) and R2
newline-as-soft-separator in blocks (narrows PAR020 to the genuine same-line case, actionable
error preserved). Backward-compatibility PROVEN, not asserted: corpus AST-diff fuzz gate
(old parser rebuilt from pinned base via temp worktree + new cmd/astdump) = zero re-parse diffs
over all 389 currently-valid corpus files, re-run independently by the evaluator. Executor's
systemic find: FOUR block loops needed the R2 patch, not the plan's two — if/then blocks and
\-lambda bodies route through parseRecordLiteral. Eval PASS 96/100 round 1 (fifth consecutive).
PR #342 merged, dev CI green per-workflow. Bar v2 clause 3's centerpiece is done: remaining
clause-3 items are the three R4 diagnostics (#13–15) + prompt work (#16, #23). PARKED: rig A/B
compile_error Δ (GPU; rotation held the rig) — the real success metric, measured post-merge.
Skill fix (2 frictions, its 8+9): sprint-executor completion gate for sprint artifacts.
Next: #11 m-stdlib-regex (NEW-DOC, clause 4).

## STATUS 2026-07-11 (afternoon) — ITERATION 8 COMPLETE: EFFECT SPRINT 1/4 LANDED (closed mode set ENFORCED)

m-effect-mode-validation landed via the full loop headless, round-1 clean (FOURTH consecutive).
The public guide's "typechecker rejects unknown values" claim is now TRUE: frozen `effectSchema`
(Rand + AI) enforced at effect-row elaboration with 3 fix-carrying diagnostics (EFF_UNKNOWN_MODE /
EFF_UNKNOWN_PARAM_KEY / EFF_PARAMS_NOT_SUPPORTED), CI-fixtured in the footgun harness; interim
accuracy note removed; teaching prompt names the codes. Eval PASS 96/100 round 1 with the
evaluator re-producing the acceptance transcript from a self-rebuilt worktree binary. PR #340 →
8faa49de9, dev CI green per-workflow. Unlocks effect sprints 2-4 (replay-contracts, clock-net-fs,
scope-params). EN ROUTE: dev-health issue #341 — 5 runnable examples fail type-check on dev
(pre-existing, proven twice independently; verify-examples is NOT a remote CI gate — same
invisibility class as iteration 3's gofmt miss). Next: #10 m-syntax-ai-forgiving.

## STATUS 2026-07-11 (day) — ITERATION 7 COMPLETE: EVAL-BAR CLAUSE MACHINERY LANDED (frontier tier + curation)

m-eval-frontier-tier landed via the full loop headless, round-1 clean (third consecutive
round-1-clean full loop). The suite regains discrimination structure: `frontier` tier exists
(8 benchmarks re-tiered with parked-validation provenance), 7 saturated core benchmarks demoted
to stretch via a conservative 4-dimension rule computed ONLY from banked re-graded v0.25.0 data
(now codified in CURATION.md §5, both directions), and the decision_block_capture free-text
exact-match anti-pattern is retired via a new `grading: prefix_line` structural grader
(GradeStdout centralizes 6 call sites). Eval PASS 96/100 round 1 with independent
distinct-sample recount (5 benchmarks × 4 dims from raw JSONs — all matched). PR #339 →
0515578ae, dev CI green per-workflow. Tier distribution now smoke 23 / core 19 / stretch 29 /
frontier 8 / vision 9. **The eval-bar clause is NOT fully closed**: frontier-failure validation
(each of the 8 must fail ≥1 frontier model, else demote back) is API-billed → PARKED for
human/next frontier rotation; 4 sketched benchmarks remain unauthored. Next: #9
m-effect-mode-validation (effect-refinement sprint 1/4).

## STATUS 2026-07-11 (morning) — ITERATION 6 COMPLETE: EFFECT-REFINEMENT DECOMPOSED (last strategic v1.0 item now sprint-sized)

Queue #7 executed as a decomposition iteration (standing rule for multi-week items; Fable lane —
no Opus sprint needed). Reality check found the parent doc's ~90h claim STALE by more than
a third: Phases 1+2 AND the AI port shipped v0.15.0 (M-EFFECT-REFINEMENT-PHASE1 +
M-AI-EFFECT-MODES), Phase 7's CryptoRand alias was scope-reduced away in v0.15.0 because
**M-CRYPTORAND never landed at all** — its doc sits in implemented/v0_15_0 with "Status:
Planned", swept there by the 48-doc bulk relocation 645467e13 (header corrected to Superseded).
Remaining ~64h decomposed into 4 sprint docs (all premises live-verified at
v0.28.0-148-g6c25f45e9): m-effect-mode-validation (1d) → m-effect-replay-contracts (3d) →
m-effect-clock-net-fs-modes (3d) → m-effect-scope-params (2.5d, release-gate re-score
candidate). Phase 6 (M-ENTROPY) routed OUT — ships with M-ENTROPY itself, not v1.0-required.
**Discovered en route**: the public parameterised-effects guide claims "the typechecker rejects
unknown values" — FALSE (`Rand[mode=banana]` passes `ailang check`, live transcript in sprint-1
doc); interim accuracy note shipped, enforcement is sprint 1. With this, every required-for-v1
queue item is sprint-sized: the bar's critical path is eval-frontier-tier (#8, [NEXT]) + the
four effect sprints + two 1–2d P1s.

## STATUS 2026-07-10/11 (night) — ITERATION 5 COMPLETE: STABILITY-PROMISE BAR CLAUSE CLOSED

m-v1-stability-promise landed via the full loop headless, round-1 clean at every stage — the
first queue item of the mission that was genuinely NEW (no stale status to catch; instead the
Opus planner caught 2 premise errors in Fable's design doc: stdlib is 42 modules not 39, and
LIMITATIONS.md is double-maintained with a diverged public website copy — both copies fixed).
Shipped: the 1.x stability promise page (Stable/Experimental/Internal tiers, full stdlib + CLI
tier tables, RATIFICATION-pending stamp), live-verified accuracy pass on BOTH LIMITATIONS files
(every entry re-verified at HEAD with transcripts; poly-arith-lambda and match-in-HOF moved to
Recently Resolved — they had been documented as broken for ~15 minor versions), 4 stale website
version-promises retracted. Eval PASS 96/100 round 1 (independent distinct-sample verification).
PR #337 → fcccd7208, dev CI green per-workflow. Two bar clauses now satisfied: "stability
promise defined" ✓ and LIMITATIONS-accuracy under core-frozen ✓ (ratification of tier
ASSIGNMENTS parked for Mark at the v1.0.0 release gate — not a merge blocker). Backlog: issue
#338 (deflake TestRunCommand_PipedStdoutFlushesPerLine — hit twice this iteration, proven
non-regression twice). Critical path remaining: effect-refinement decomposition (#7, [NEXT]) →
eval-frontier-tier (#8).

## STATUS 2026-07-10 (night) — ITERATION 4 COMPLETE: LAST SPRINT-SIZED P0 CLOSED FOR V1

m-diagnostic-coverage reality-check found its "Planned" status STALE (M1–M3 shipped 2026-07-09,
ff58a3259/e59197554 — second stale-status catch of the mission); the genuinely open remainder ran
as sprint M-DIAG-FIXTURE-PROMOTION via the full loop headless: Opus plan (2 real discrepancies
found live: %/Fractional's claimed diagnostic is UNREACHABLE via `ailang check`; stdlib-hint
fixture needs TestMain wiring) → Opus execute in worktree → Fable eval **PASS 96/100 round 1**
(non-vacuity proven twice, independently). 4 footgun rows promoted to `covered` → 7 CI-enforced
fixtures. Integrated via **PR #336 → fe807aac8** — a sibling agent held a conflicted merge open
in the shared main tree all iteration, so integration went worktree-branch→PR→auto-merge without
ever committing in the main tree (new Gate-2 rule 4 codifies this). Dev CI green per-workflow on
the merge. DEFERRED with rationale in the design doc: prompt-deletion pass + rig A/B (35
deletable lines < the ≥100 gate — widen coverage first); PARKED for human: haiku causal re-run
(API-billed, blocked by the headless billing guard). With this, all four sprint-sized P0s from
the ratified bar are closed; the critical path is now the stability promise + effect-refinement
decomposition + eval-frontier tier.

## STATUS 2026-07-10 (evening) — ITERATION 3 COMPLETE: P0 OPERATIONALLY CLOSED (CODE-SIDE), DEV UN-REDDED

m-feedback-gate-cloud-adapter landed via the full loop headless: Fable design doc (all 6
premises later verified accurate by the Opus planner — zero discrepancies, a first), Opus
plan + execute in isolated worktree, **first round-1 evaluation FAIL of the mission** (a
fail-open bug: Firestore read errors silently reset cooldown/budget windows; numeric 92 but
CLAUDE.md CP2 policy violation) → surgical round-2 fix → PASS 97/100, merge 842d7d501.
En route, OBSERVE's "dev green" turned out WRONG — dev had been red since 12:57Z behind a
dependabot-flooded run list (Gate-1 gap, skill-fixed): three pre-existing breaks fixed forward
(Windows codegen escape bug in contract panic literals 9d2e32ac1 + docs npm peer-dep pin +
go-test timeout 60s→300s 4c22032de). Dev fully green on 4c22032de. CALIBRATED: gate code is
now complete INCLUDING cloud adapters, still off by default; production activation awaits
HUMAN ops (terraform TTL + ANTHROPIC_API_KEY secret in sibling repo, then DRY_RUN week 1) —
parked in #329.

## STATUS 2026-07-11 (midday) — v1.0 BAR v2 RATIFIED (product-shaped): "the verified AI-orchestration language". Cutoff rule live: gates-v1 ⟺ serves an open clause. Queue re-derived (16 open, 7 NEW-DOC); 17 docs v1_0_0→v1_1_0; strategy review ACTIVE w/ Design Freeze 1+2 ratified. See bar section + log entry 9.

## STATUS 2026-07-10 (afternoon) — ITERATION 2 COMPLETE: FIRST HEADLESS FULL-LOOP RUN, P0 GATE LANDED

m-feedback-triage-gate landed via the full inner loop with NO human present — the friction
flagged at 13:30 (headless run reached planning without routing evidence) is answered: both
plan and execute carried verified `claude-opus-4-8` attestations, evaluation by Fable with
independent re-verification (full make test rc=0 in worktree, drill run live). Eval 93/100 PASS
round 1, merge 40f1cdc3f. The killed 13:03 run's leftovers (dead-locked worktree + uncommitted
scaffold) were quarantined and cleared. CALIBRATED: gate logic is complete and merged, off by
default; production protection is rules-only until m-feedback-gate-cloud-adapter (queued [NEXT])
ships the Firestore adapters — the P0 is not operationally closed yet. New "Dev stays GREEN"
guardrail honored: merge pushed, remote CI verified before tagging [LANDED] (result recorded in
the queue entry).

## STATUS 2026-07-10 (13:30) — SCHEDULING MOVED INTO CLAUDE CODE; BILLING INCIDENT CLOSED

The first kickstarted headless run billed ~13 min of API credits (`ANTHROPIC_API_KEY` leaked
from secrets.env into `claude -p`; killed rc=143). Two fixes: (1) the launchd driver gained a
billing guard (strips API keys, refuses without a subscription token) and was then (2)
**superseded as primary by the Claude Code scheduled task `v1-mission-nightly`** — runs inside
the app on the session OAuth, no token handling at all. The killed run's unreviewed plan
artifacts for m-feedback-triage-gate were deleted (produced in 13 min, no Opus routing
verified, no controller review) — tonight replans through the proper loop. Friction recorded:
the headless controller reached Gate 3 planning without evidence it honored the model routing —
watch for this in tonight's #329 report.

## STATUS 2026-07-10 (evening) — ITERATION 1 COMPLETE: FIRST FULL INNER-LOOP RUN, 2 QUEUE ITEMS LANDED

- **1a** m-named-test-blocks closed out (shipped 2026-07-09, verified live incl. duckdb's
  formerly-silent-skipped tests now 2/2; deontic criterion deferred — package absent locally).
- **1b** m-typeenv-sub-fix RESOLVED via the full loop: Opus planner (found 6 stale design-doc
  items incl. 2 interacting post-doc fixes) → Opus executor in isolated worktree (proved the P0
  no longer reproduces; declined the 135-LOC repair; shipped regression guards instead) → Fable
  evaluation 92/100 PASS with adversarial non-vacuity proof (tests FAIL at the bug-live commit,
  PASS from M-TYPE-LIST-SOUND round 3). Merged f59421ac8.
- **Found + fixed en route**: stale `bin/ailang` (v0.26.0) silently breaking `make test` on this
  rig — test helpers prefer `bin/ailang`; systemic fix spun off as a background task.
- **Retro executed the ≥2 rule for real**: three same-class frictions (stale binaries ×2,
  parked-test-as-status ×1) → ONE skill edit: Gate-2 verification protocol added to
  mission-control (rebuild both binaries, un-skip-and-run parked tests, PIPESTATUS).
- Routing evidence rows 1–2 recorded (Opus plan + execute: both high quality; 1 attribution
  correction at evaluation).

## STATUS 2026-07-10 (later) — ITERATION 0 COMPLETE: BAR RATIFIED, BACKLOG RE-SCORED

Mark ratified the v1.0 bar (interactive session) and made the scope calls:
- **Effect refinement IN** (public docs promise; decompose ~90h into sprints before executing);
  **effect handlers OUT → v1.1** (largest new surface; no bake time under a fresh stability
  promise is the risky combination).
- **CSP session types, quasiquotes, perf4-bytecode, D4 all OUT → v1.1** (plus agent-orchestration
  and zero-language-learnings by ordering policy — multi-week strategic, not selected IN).
- **Both v1_0_0 P0s downgraded to P1/nice-for-v1** (global-collaboration-hub,
  m-eu-compliance-effects — multi-week non-language items; dated notes in the docs).
- Reality-check finding: **m-named-test-blocks core scope already SHIPPED** (M1 commits
  ec4996e45/7389e84c1 + fixes; verified live post-rebuild: failing named test → FAIL + exit 1).
  Reduced to a closeout item. m-feedback-triage-gate confirmed genuinely open (the shipped
  M-MCP-EDGE-THROTTLE rate limit is its precondition, not its scope).

## STATUS 2026-07-10 — MISSION INITIALIZED, ITERATION 0 PENDING

Exploration findings that shaped this charter (full census in log entry 0):

- **No written v1.0.0 criteria exist anywhere.** Scope is currently implicit folder membership:
  66 docs in `planned/v0_29_0/`, 27 in `planned/v1_0_0/`, 4 in `planned/v1_1_0/`. Iteration 0
  must define the bar before any sprint is picked.
- **6 P0s are open**: m-typeenv-sub-fix (type-safety hole), m-feedback-triage-gate (public-endpoint
  cost/safety), m-named-test-blocks (silent-green test runner), m-diagnostic-coverage (cheapest
  cost-per-success lever) in v0_29_0; global-collaboration-hub, m-eu-compliance-effects in v1_0_0.
- **The inner-loop skills are sound but had no model policy and no self-improvement path** —
  `dev-cycle.md` pinned `model: sonnet` (fixed 2026-07-10 → opus), retros written to
  `docs/sprint-retros/` were never folded back into the skills.

