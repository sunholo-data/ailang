# V1 Mission — STATUS stamp archive (rotated out of the charter)

Newest first. Rotation rule lives in the charter's STATUS section. Full per-iteration
detail is in v1-mission-log.md — these are the headline stamps only.

## STATUS 2026-07-21 — ITERATION 74: **M2b matched A/B EXECUTED + banked → M2b LANDED (3/5 milestones); treatment delivery PROVEN, headline trends NEUTRAL-at-ceiling** — resumed the active M-EVAL-FMT-WEAKMODEL-AB sprint at its next milestone M2b (the matched haiku ON/OFF run). **Cloud-haiku is API/subscription, NOT GPU → NO rig.lock** (`-no-rig-lock`; sprint-plan rig.lock text superseded by mission commit `69501e6dd`). Build: worktree at origin/dev `3045f92f5` (installed binary `af6ea1d89` PREDATED the iter-73 file-sink fix `ac5e1735e` → rebuilt clean). Benchmarks verified UNCHANGED since frozen SHA `2bb1820d6`. Verify-before-scaling: 1× ON smoke first → banked `fmt_hook_state:on` + a `formatted` fmt event (file-sink capture confirmed live) → then ran both arms. **60 runs banked** (`eval_results/fmt_ab_haiku_M2b/{on,off}/`): 30/arm = 5 trials × 6 benchmarks. Config-diff clean (both arms prompt_version v0.16.3, model claude-haiku-4-5, seed 42, parallel 4, trials 5 — ONLY `fmt_hook_state` differs). **Arm gating**: OFF banked 0 fmt_hook_events (all 30 labeled off); ON delivered treatment in **29/30 runs** (32 `formatted` events) vs the ~8% baseline the prereg flagged → the file-sink fix works end-to-end. **Headline** (M3 does rigorous Wilson CIs): OFF 29/30, ON 30/30, +1-run delta driven entirely by `cli_args` (4/5→5/5) = within noise at haiku near-ceiling → trending **NEUTRAL/NULL-at-ceiling with treatment delivery PROVEN**. All roles quota-bucket/subscription; `metered=$0.00`. **NEXT = M3 analysis + M4 verdict** (no-GPU, on banked data). Detail: log entry 79.

## STATUS 2026-07-21 — ITERATION 73: **M2b `TODO(M2b)` live-verified → treatment-integrity capture FIXED (file sink); M2b now UNBLOCKED-for-GPU** — resumed the active M-EVAL-FMT-WEAKMODEL-AB sprint at M2b; a subscription-haiku smoke (verify-before-scaling) exposed a data-proven BLOCKER: the hook FIRES (sentinel-proved) + emits `✓ Formatted` on stderr+exit0, but **Claude Code SWALLOWS exit-0 hook stderr** in stream-json → the iter-72 stream-scan capture was structurally always-empty → §5.3 treatment-delivery gate unmeasurable → every M2b verdict would be "unevaluable". Pivoted the deliverable to the unblocker: executor (opus, worktree) moved capture to an **out-of-band file sink** (`<cwd>/.claude/fmt_hook_events.jsonl`, cwd-derived → no env-forward, NON-contaminating — `additionalContext` explicitly rejected as it would corrupt the ON arm); evaluator (sonnet, generator≠judge) PASS **88/100 r1**, no must-fix. Live-verified: ON smoke banks a `formatted` event, OFF banks none. Prereg §Amendments records the MEASUREMENT-only, pre-scored-run correction. Commits `647deadbb`+`8d45e4a63`; PR + squash in log #78. The 60-run A/B is deferred to a GPU slot (config-only now). `metered=$0.00`. Detail: log entry 78.

## STATUS 2026-07-21 — ITERATION 72: **m-eval-fmt-weakmodel-ab UNPARKED (Mark #422 green-light) → M1 prereg + M2a fmt-hook toggle LANDED** — PR #438 squash `260faa42a`; planner → executor (opus, worktree) M1+M2a → evaluator (sonnet, generator≠judge) PASS **80/100 r1** + round-2 fix; **3 Gate-3b reds all fixed-forward** (check-file-sizes claude.go 799→829, Windows path-assertion, a self-corrected poll-on-non-required-SonarCloud); M2b GPU-gated + M3/M4 deferred; `metered=$0.00`

Mark's #422 comment **"Green light weakmodel ab"** (`2026-07-21T12:23:01Z`, allowlisted principal) UNPARKED the iter-71 doc that was parked at the quorum gate awaiting exactly this human sign-off → a human directive outranks the queue = the pick. Gate-0/1 CLEAN (killswitch armed; billing **CLEAN** both Anthropic keys empty; gh `sunholo-voight-kampff`; `MODEL` empty → controller **opus**; **#422** created 07-20 07:00, 6 comments <80 → no rotation; predecessor #399 rotation-catch clean; watermark advanced 12:23:01Z BEFORE routing; 4 `eval-suite` informational acked). Local dev == origin/dev `2bb1820d6`; dev CI green. **Gate-2 QUORUM-AT-PICK**: doc pre-existed + 2 quorum artifacts on disk + Mark's green-light IS the human review → no re-quorum. **Route**: no plan → **sprint-planner (opus)** — repo-grounded finding: the fmt hook ON/OFF toggle **did not exist** (the active agent-mode path wrote no `.claude/settings.json`); precedent `microrag_mode.go`; split M2 into M2a (build, no-GPU) + M2b (execute, GPU). Scope = **M1+M2a only** (headless-safe). **Executor (opus, worktree)**: M1 prereg (6 `.ail`-editing benchmarks frozen, N=5/arm, Wilson CIs, refutation threshold) + M2a `FmtHookMode` on/off + `-fmt-hook` flag + fail-closed hook-reality capture; corrected the plan's wiring (ACTIVE path is `agent_runner_multi.go`, not the legacy runners). **Evaluator (sonnet, generator≠judge)** PASS **80/100 r1** — caught a MUST-FIX (dead `OnToolResult` on the active path → empty `fmt_hook_events`); round-2 fix via opt-in `RawStreamLineHandler` raw-line scan + `TODO(M2b)` for the one live-unverifiable detail. **Gate-3b did its job 3×**: check-file-sizes (extracted `claude/helpers.go`, claude.go→785), Windows path-assertion (`jsonContains` escape-aware), and a self-caught poll bug (bailed on non-required SonarCloud — required set is only test/lint/build). SonarCloud new-coverage 39.3% is non-required/expected for integration wiring (M2b live-verifies). Controller finalized bookkeeping (this stamp, queue tag, log #77) + one Gate-5 skill edit (executor done-gate += `make check-file-sizes`) in worktree `iter72-bookkeeping` → PR. `metered=$0.00` (all quota-bucket; ≤ `MISSION_METERED_BUDGET_USD=$5`). Detail: log entry 77.

## STATUS 2026-07-21 — ITERATION 70: **orphaned-crashed-run RESUMED + LANDED** — m-ailang-fmt-inline-interior (executed by the crashed iter-70 as PR #434) squash `3c1cec57d`; planner (opus) → executor (opus, worktree) M0–M3 → evaluator (sonnet, generator≠judge) PASS **91/100 r1**; `let-chain-interior` comment-refusal sub-class fully eliminated (59→32 total, 15.28%→8.27%); stale-base mermaid docs-build red fixed via `gh pr update-branch`; `metered=$0.00`

Mark #422 "Continue the AILANG fmt sprint" (`bc61ea8ce`: fmt quorum objections DATA-REFUTED → proceed, no re-quorum) covers the whole fmt polish pair; iter-69 landed m-fmt-properties, this pick is the other half — **m-ailang-fmt-inline-interior**. Gate-0/1 CLEAN (killswitch armed; billing **CLEAN** both Anthropic keys empty; gh `sunholo-voight-kampff`; `MODEL` empty → controller on **opus**; issue **#422** created 07-20 07:00 (this Monday), 3 comments <80 → no rotation; **no new Mark comment** since watermark `2026-07-20T07:11:34Z`; nightly #384 already-triaged kept OPEN). Inbox: 1 `eval-suite` informational → acked. **Origin-sync + Gate-2 caught the item was ALREADY IN-FLIGHT**: open **PR #434** (bot, `sprint/m-ailang-fmt-inline-interior`) held a FULL evaluated inner loop — an **orphaned crashed iter-70** that an 18h reboot outage killed pre-Gate-3b/4 (per the sibling `b5b9899a0`/`38aa1fb5e` RunAtLoad+heartbeat commits; no log entry existed). **THIS run resumed at Gate-3b** (did NOT re-spawn the loop — Standing rules 4/5): diagnosed the sole red (docs-`build`) as a **stale-base mermaid npm ERESOLVE** (`@mermaid-js/layout-elk`; branch cut before `08da65dc4`/#435 re-pinned to `^0.1.9` — NOT a sprint defect, data-before-conclusions), `gh pr update-branch 434` merged dev in → all checks green → auto-merge squashed `3c1cec57d` @ 09:01:26Z. Controller finalized Gate-4 (queue tag→LANDED, this stamp, log #75) in worktree `iter70-bookkeeping` → PR. `metered=$0.00` (all quota-bucket; ≤ `MISSION_METERED_BUDGET_USD=$5`). Detail: log entry 75.

## STATUS 2026-07-20 — ITERATION 65: queue pick **m-ailang-fmt-adoption EXECUTED + LANDED** — PR #415 squash `b787bb98f`; executor (opus, worktree) M1–M3, evaluator (sonnet, generator≠judge) PASS **89/100 r1**; `ailang fmt` now discoverable (teaching prompt v0.16.3) + opt-in auto-format PostToolUse hook with the Mark-approved SIGTERM→grace→SIGKILL escalation; `metered=$0.00`

iter-64 produced the ready 3-milestone plan → this iteration executed it (Gate-3 plan→execute). Preflight CLEAN (killswitch armed, billing **CLEAN** both keys empty, gh `sunholo-voight-kampff`, main tree carried 3 pre-existing dirty generated docs — left untouched, Critical Principle 0; no `MERGE_HEAD`; issue **#399**, 40 comments <80, today 07-20 00:10 local is Monday but BEFORE the 07:00 quota-reset boundary → most-recent-passed boundary is 07-13, #399 created after it → **no rotation**; no new Mark comment on #399 OR predecessor #329 since watermark `2026-07-19T07:52:58Z`). Local dev == origin/dev `5afa9a1e1`; dev CI CI/Build/Docs all `success` @ `5afa9a1e1`. Inbox: 1 `eval-suite` informational → acked. **Gate-2**: NOT landed/ghost — only plan commit `52ed0204c`, doc in `planned/`, no branch/PR; the intervening `5afa9a1e1`/K3 commits are a separate eval stream (CI green, not the fmt loop). Quorum satisfied by Mark decision (no re-quorum). **EXECUTOR (opus, pinned Agent, isolated worktree)** shipped M1 prompt v0.16.3 (append-only, hashed) → M2 `formatter.md` Adoption section + dev-workflow cross-link → M3 `make fmt-check-ail` (renamed off the pre-existing Go `fmt-check` gate; `make ci` byte-identical) + opt-in `format_ail.sh` w/ SIGKILL escalation. 5 commits `7de65dc4b`→`f1546e4a7`. **EVALUATOR (sonnet, generator≠judge)** PASS **89/100 r1** — independently re-ran hook tests (b/c/d incl. the SIGTERM-ignoring-stub reap in ~11s) + append-only proof + `make ci` byte-identical + `make test` green; 2 MINOR gaps (CHANGELOG, doc-move) = controller finalization. **Controller DISPROVED the executor's flagged "Docusaurus build failure"**: it was the skipped CI-only `docs/scripts/sync-registry.sh` gen step (generates untracked `packages/sunholo/*` + sidebar entries) + stale `.docusaurus` cache — after the CI-faithful step the build is green with the sprint edits. Controller finalized (CHANGELOG + doc→`implemented/v0_30_0/` + noise-revert `a813fe1b2`), PR #415 auto-merge squash, Gate-3b bounded-poll (2 windows, 0 failures) → merged `b787bb98f`. `metered=$0.00` (all quota-bucket; no metered lane fired; ≤ `MISSION_METERED_BUDGET_USD=$5`). Detail: log entry 70.

## STATUS 2026-07-19 — ITERATION 64: queue pick **m-ailang-fmt-adoption** (fmt follow-up; phase2 gate now SATISFIED) → **routed to sprint-planner (opus); 3-milestone plan produced (teaching prompt v0.16.3 · `make fmt-check` · opt-in PostToolUse hook w/ Mark-approved SIGKILL escalation), READY FOR EXECUTOR** — no re-quorum (Mark-approved iters 60/62); exit-code contract VERIFIED matches (0/1/2/3); `metered=$0.00`

## STATUS 2026-07-19 — ITERATION 63: queue pick **m-ailang-fmt-phase2 EXECUTED + LANDED** — PR #414 squash `3815ba617`; executor (opus, worktree) shipped M0–M3, evaluator (sonnet, generator≠judge) PASS **78/100 r1**; `ailang fmt` now preserves comments losslessly on ~85% of the corpus, fail-closed (never lossy) on the rest; `metered=$0.00`

## STATUS 2026-07-19 — ITERATION 62: HUMAN DIRECTIVE (#399) — Mark UNPARKED m-ailang-fmt-phase2 (option b, commit `c624b456d`) → **routed to sprint-planner (opus); 4-milestone plan produced (M0 = printer child-list audit, interpolation = fail-closed carve-out), READY FOR EXECUTOR** — no re-quorum per Mark; `metered=$0.00`

## STATUS 2026-07-19 — ITERATION 60: HUMAN DIRECTIVE (#399) — Mark "Yep one more short decision round" → **Rev-3 revision fixed the 4 R2 defects, but the re-quorum surfaced 2 NEW architecture-level objections on phase2 → both fmt docs STILL PARKED needs-human-review** (iter-59's "few fixes from green" REFUTED)

Mark's #399 reply @`2026-07-19T07:52:58Z` authorized iter-59's option 1 (one revision + re-quorum on the 4 R2 defects) → outranks queue, unparks the fmt docs. Preflight CLEAN (killswitch armed, billing CLEAN, gh `sunholo-voight-kampff`, tree clean, no MERGE_HEAD; issue **#399**; dev CI CI/Build/Docs all green @ `e37b370d1`). Local dev == origin/dev `e37b370d1`. **Controller (Opus)** produced phase2-defect-1's datum directly: parser-level parse-validity sweep (`ailang check --json` per file @v0.30.0) → **386/393 (98.2%) parse-valid**; 7 non-parsing all in `archive/broken`+`bugs`+`experimental`. **Designer** = `claude:claude-fable-5` via `claude-sub` ($0 subscription, probe rc=0), held at author for revision continuity (not advanced — a revision ≠ new-doc). Bounded 30-min run, surgical fixes to BOTH docs (phase2: V21 + corpus gate over the 386 subset + hard-left-wall clause + fixtures; adoption: bounded-timeout hook + `command -v jq` probe + dropped first-jq `2>/dev/null`). **Re-quorum (bounded ONE round, gpt5-6-sol+gemini) — BOTH STILL BLOCKED:** the 4 R2 defects were resolved (not re-raised), but phase2 surfaced **2 NEW architecture-level objections** — (1) gpt5-6-sol → attacher-totality inventory unproven (partial child-list inventory; no code-audit of params/type-args/ctor-args/record-fields/annotations); (2) gemini → interpolation clamping structurally fatal (collapses inner-AST boundaries + would silently delete comments in `${…}`; promotes V19 to a blocker). adoption: jq fix accepted, but both reviewers reject the timeout fix (SIGTERM-then-unbounded-`wait` wedges on a signal-ignoring proc) — 1 trivial SIGKILL-escalation from clean, but hard-gated behind phase2. Bounded gate (one revision + re-quorum) CONSUMED + Mark's authorization singular → **PARKED needs-human-review**; docs updated with "Rev-3 Re-Quorum Outcome" sections, committed `d1ed2fe57` (doc-only). **phase2 is NOT "a few fixes from green"** — 3 rounds, each a deeper premise gap; needs a design-verification sprint (printer child-list audit + interpolation-aware attachment). `metered=$0.23` (phase2 R3 degraded $0.033 + complete $0.112 + adoption $0.084; Fable designer $0 subscription; Anthropic-quota controller separate). Detail: log entry 65.

## STATUS 2026-07-19 — ITERATION 59: HUMAN DIRECTIVE (#399) — Mark "Yep do the fmt design docs next" → **created m-ailang-fmt-phase2 + m-ailang-fmt-adoption; both design-quorum BLOCKED ×2 → PARKED needs-human-review** (design converged, 2–3 small fixes from green)

Mark's #399 reply @`2026-07-19T06:14:47Z` greenlit iter-58's proposal → this iteration's pick (outranks queue). Preflight CLEAN (killswitch armed, billing CLEAN, gh `sunholo-voight-kampff`, tree clean, no MERGE_HEAD; issue **#399**; dev CI last-completed green @ `c0f0ccde9`, HEAD in-progress = rig dashboard commits). Local dev == origin/dev `5cdfb912c`. ROTATION designer (next after codex = **claude:claude-fable-5**, gemini gated behind G4) via `claude-sub` (subscription, $0 metered), probe green → authored BOTH planned docs. **phase2** (refine-into-planned of the Phase-1 §168-332 Lossless Attachment design) + **adoption** (discoverability + opt-in non-blocking/non-silent hooks, gated behind phase2 + a fmt exit-code split). **Design-quorum ×2 (bounded one-revision cap): BLOCKED both rounds** but CONVERGING. R1: phase2 → central-envelope-premise unverified (both reviewers); adoption → `2>/dev/null` silent-fallback hook (both). Reviser (same Fable designer) did design-time verification (throwaway Go probes): column unit resolved (1-based NFC rune), AST spans proven UNUSABLE (only 3 node kinds carry Span, End excludes token text, Offset unpopulated) → design **pivoted to a token-anchored envelope** off the byte-exact lossless scan (corpus-swept 81,224 tok/393 files); hook redesigned to capture stderr + defer silently ONLY on a new exit-3 (input-not-parseable). R2: STILL BLOCKED on new, narrower, fixable defects — phase2: (a) gpt5-6-sol → "393 files parse" rests on V4 which ran Phase-1 fmt (refuses 372 pre-parse) so parse-validity of the refused majority is unproven; (b) gemini → left-widening rule makes the first child of a bracketed construct consume the parent's open delimiter (`[ /* C */ x ]` traps `C` in `x`), breaking attacher totality. adoption: (a) gpt5-6-sol → hook has no timeout (bounded-waits violation); (b) gemini → the FIRST `jq` still `2>/dev/null` (residual silent-swallow of missing-`jq`). Gate = ONE revision + re-quorum → **PARKED needs-human-review**; both docs committed (`ad14dfc19`, doc-only) with ⛔ Quorum Block sections + crisp unblock (authorize one short revision to fix the 4 concrete defects, or amend). No 3rd round (respects the cap + cost discipline). Designer rotation advanced → `claude:claude-fable-5`. `metered=$0.30` (quorum R1 $0.127 + R2 $0.170; Fable designer = $0 subscription; Anthropic-quota controller separate). Detail: log entry 64.

## STATUS 2026-07-19 — ITERATION 58: HUMAN DIRECTIVE (#399) — Mark "is ailang fmt discoverable by agents via prompt? can we run it every turn after .ail writes (Motoko / harness hooks)?" → **answered from reproduced evidence, $0.00 metered** (no heavy-role spawn). Findings: (1) `ailang fmt` NOT in the teaching prompt (agents don't know it exists); (2) Phase-1 fmt REFUSES comment-bearing files → **344/393 (87.5%) of `examples/*.ail` un-formattable today** → auto-run-every-turn is near-useless pre-Phase-2. Both threads gated on **Phase-2 comment preservation** (design pre-exists). Queued **m-ailang-fmt-phase2** (unblock) + **m-ailang-fmt-adoption** (gated) as NEW-DOC candidates awaiting Mark greenlight. m-serveapi M1 (`@nomcp`) still PARKED for the human fork.

## STATUS 2026-07-19 — ITERATION 57: queue pick **m-serveapi-raw-handler-mcp** (clause-4 unblocker) → **QUORUM-AT-PICK BLOCKED ×2 → PARKED needs-human-review**; M1 (`@nomcp`) clean+shippable, M2 architecture is a human fork

Clause-3 routable items exhausted (remaining PARKED/evidence-gated or GATED); per "unblockers first" the top clause-4 candidate was the DOC-READY unblocker **m-serveapi-raw-handler-mcp** (closes the live docparse `getKeyUsage`/`requestHistory` MCP leak). Preflight CLEAN (killswitch armed, billing CLEAN, gh `sunholo-voight-kampff`, tree clean, no MERGE_HEAD; CI+Build+Docs green @ `b205df841`). No new #399 directive (the one Mark cost-comment @`2026-07-18T16:09:03Z` was already answered iter-55; watermark advanced). Gate-2 reality-check CONFIRMED premise live (`@nomcp` absent; `routes.go:106` `IsNoExpose=false // @route overrides @noexpose`; `mcp.go:60/188` + `routes_dispatch.go:51` match doc line-refs). **QUORUM-AT-PICK (no artifact → required): BLOCKED ×2.** R1 (Rev 1): gpt5-6-sol → M2 default-on `@raw`-over-MCP silently fabricates headers = authority-WIDENING + silent-fallback (Crit. Principle 2); gemini → threads singular `RouteMethod`/`RoutePath` but MCP registers per-FUNCTION (0/>1 route). Routed to ROTATION designer **codex:gpt-5.6-sol** (bounded, workspace-write, doc-only 186+/96−) → Rev 2: `@mcp` opt-in + function-keyed envelope (`method="MCP"`, `path="/mcp/tool/"+funcName`) + typed unavailable-context sentinels. **R2 (Rev 2): STILL BLOCKED** — the sentinel fix is itself flawed: `headers`/`query` are typed `Json`, so a non-`Json` sentinel **type-panics at parameter binding** (before any projection — gemini), and a `Json`-valued sentinel would need core `std/json` changes = **Minimal-Frozen-Core violation** (PROGRAM.md); gpt5-6-sol concurred (unjustified `internal/eval/` core expansion). Gate = ONE revision + re-quorum → **PARKED needs-human-review**. **Both rounds objected ONLY to M2**; M1 `@nomcp` is clean+unobjected. Human fork on #399: (1) split+ship M1 now [RECOMMENDED], (2) pick an M2 arch (gemini's valid-`Json` provenance-marker+`req.method=="MCP"` branch, or drop the fake-envelope), (3) keep parked. Rev-2 doc preserved with a ⛔ Quorum Reblock section. Designer rotation advanced → codex. `metered=$0.15` (quorum R1 $0.066 + R2 $0.082; codex designer = OpenAI-key spend ~$0.25 est. from 83K tok — separate from the Anthropic-quota controller). Detail: log entry 62.

## STATUS 2026-07-18 — ITERATION 54: queue pick **m-prompt-footguns-to-diagnostics LANDED** (clause-3 fleet-tier accessibility; eval 91/100 PASS r1) — the ~10% multi-module footgun is now a coded teaching diagnostic

First non-G4 pick after the fleet-rollout arc — the loop returns to the clause-3 queue. Mark-ratified queue item (commit `3df673994`, 2026-07-18 "greenlight/close/ratify"; **reconciled a stale-record discrepancy**: the iter-52/53 log "still parked" lists were carry-forward that predated the 08:24Z ratify commit — the queue tag + this commit are ground truth). No new #399 human directive (watermark `2026-07-18T11:59:47Z` unchanged, no Mark comment since). Preflight CLEAN (killswitch armed, billing CLEAN, gh `sunholo-voight-kampff`, CI green @ `b4f763e51`). Gate-2 live-repro CONFIRMED the PRIMARY premise REAL at HEAD (multi-`module` file → opaque `PAR_NO_PREFIX_PARSE` cascade; `MOD002` defined at `codes.go:67` but dormant/unwired; `PAR_MODULE_PLACEMENT` absent). Full inner loop, each role pinned/bounded: PLANNER (opus) → 3-milestone ~1.25d plan, all anchors re-verified live (no material drift). EXECUTOR (opus, isolated worktree) → M1 Part A wires `MOD002` + adds `PAR_MODULE_PLACEMENT` at `parseTopLevelDecl` (mirrors `reportMisplacedImport`) + **gemini's error-recovery state-isolation fix** (two late modules → `PAR_MODULE_PLACEMENT`×1 + `MOD002`×1 genuine-dup, never a FALSE MOD002; recovery path never sets `seenModule`/`firstModulePos`); M2 Part B ghost-closes split-list-operations with CI-gated `examples/runnable/split_map_join.ail`; M3 regen `error_codes.json`, CHANGELOG, severed **Part C → backlog stub** `m-diag-primitive-field-suggestions.md` (extension-lane, per Mark's park-note). Controller independently verified: build clean, targeted tests green, gofmt clean, all files <800L, and behaviorally proved all four diagnostic cases fire (MOD002 no-cascade / PAR_MODULE_PLACEMENT / 2× state-isolation / clean single-module). EVALUATOR (**sonnet**; generator≠judge: opus exec vs sonnet judge) → **91/100 PASS r1**, all findings non-blocking. Doc + sprint-plan → `implemented/v0_30_0/`; 3 superseded prompt docs → `archive/`. No designer/Fable/cross-provider spawn (existing quorumed doc). Detail: log entry 59.

## STATUS 2026-07-18 — ITERATION 53: HUMAN DIRECTIVE (#399) "vertex git clone test granted" → **G4 live E2E PASSED — last premise LIVE-VERIFIED; doc → implemented/**; gemini fleet-role assessment reported to Mark

Mark on #399 (`2026-07-18T11:59:47Z`): *"Authorization to do a vertex git clone test granted - report back if suitable for fleet"* — unparks the iter-52 live-E2E hand-off, becomes the pick (outranks queue). Preflight CLEAN (killswitch armed, billing CLEAN, gh `sunholo-voight-kampff`, tree clean, CI green @ `bbd615d45`). **ADC present** (token len 257, quota project `aitana-multivac-dev`). Ran `TestLiveCloneOverEgressE2E` (`AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`, project `ailang-dev`/`global`) pinning a REAL non-HEAD SHA `80cbd9612…` to genuinely exercise **fetch-by-SHA** (not the HEAD path): through the production `Executor.Execute` path the sandbox ran `git fetch --depth 1 origin <sha>` + `git checkout --detach FETCH_HEAD`, **echoed the EXACT pinned SHA** back, emitted `CLONE_OK` + verdict JSON → **PASS in 113.6s, $0.865, 527221 in / 8201 out tokens**. The last INCORPORATED premise (provider `git fetch --depth 1 <sha>` support) is now **VERIFIED-LIVE**. Bookkeeping (earned): doc + sprint-plan `git mv`'d to `implemented/v0_30_0/`, status header → LIVE-VERIFIED, Premise Log row + M4 + acceptance-criteria all flipped. **Fleet-suitability finding (reported to Mark, decision parked):** gemini/managed_agents is now a proven in-sandbox **EVALUATOR/reviewer** (clone → `ailang check` → verdict; independent Google provider = valid generator≠judge) — but `CapRemoteSandbox` means it CANNOT edit a worktree, so the pre-planned "gemini joins the DESIGNER rotation" needs the text-bridge and is NOT wired; recommend gemini enter the fleet as evaluator, ratify designer-role separately. Did NOT flip the rotation env (routing-policy change → Mark's call). No sprint (live-verify + bookkeeping pick). Detail: log entry 58.

## STATUS 2026-07-18 — ITERATION 52: HUMAN DIRECTIVE (#399) "apply both fixes, ship it" → **G4 Phase-2 clone-over-egress IMPLEMENTED + LANDED** (dev CI green `80cbd9612`); live E2E pending Mark's ADC

Mark on #399 (`2026-07-18T08:54:45Z`): *"apply both fixes, ship it"* — unparks G4, resolves iter-51's re-quorum block. **Full inner loop ran end-to-end and landed.** DESIGNER (`claude:claude-fable-5`, revision pass — not new-doc, so rotation unchanged) applied both Mark-approved fixes: (1) typed `RequiresEgress bool` + `CapNetworkEgress` + shared `ValidateTaskCapabilities` pre-dispatch gate replacing the Metadata-key opt-in (closes the programmatic silent-fallback hole; re-widens `executor.Task` per Mark — Conflict Surface flips executor.go to TOUCHED, frozen-core holds); (2) "Bounded execution & timeout reuse" section grounding Phase 2 in the existing `WithTimeout`→`sendInteraction` deadline + eval-bridge ctx threading (Standing Rule 6) → **PLAN-READY**. **Re-quorum** (the single allowed round): `gpt5-6-sol` **budget-absent** (doc >$0.10 cap → N-1; its own objection #2 already applied); `gemini-3-1-pro` raised ONE new sound objection (arbitrary-SHA **full clone** was an unverified/unbounded premise — Probe R only proved `--depth 1`) → applied its **verbatim** recipe (**shallow fetch-by-SHA** `git fetch --depth 1 origin <sha>`; both clone modes now bounded) inline under Mark's "ship it" (which outranks the re-quorum-once park; surfaced for veto). PLANNER (opus) → 4-milestone sprint (~145 LOC). EXECUTOR (opus, isolated worktree) → M1 typed gate+env-wiring, M2 CLI flags, M3 eval-bridge clone-review, M4 gated live-E2E+docs; build+tests+gofmt+file-sizes green (controller-independently verified). EVALUATOR (**sonnet**; generator≠judge: opus exec vs sonnet judge) → **91/100 PASS**; F1 (stale `TestCapabilities`) + F2 (doc status) folded in. FF-merged to dev `80cbd9612`; **Gate-3b: CI + Build-and-Release + Deploy-Docs all GREEN observed**. The ONE INCORPORATED-not-live premise (provider `fetch --depth 1 <sha>` support) → live E2E SKIPs without ADC → **hand-off to Mark** (doc stays in `planned/` until confirmed). Rotation write-back: unchanged `claude:claude-fable-5`. Detail: log entry 57.

## STATUS 2026-07-18 — ITERATION 51: HUMAN DIRECTIVE (#399) "clone over egress approved" → **G4 Phase-2 clone-over-egress DECOMPOSED + 2-round quorum-hardened → PARKED needs-human-review**

Mark greenlit the clone-over-egress *scope* on #399 (`2026-07-18T06:58:06Z`), unparking G4 (iter-46's exact parked ask). Routed the decomposition to the ROTATION designer **`claude:claude-fable-5`** (next after codex; probe rc=0; billing-guarded `claude-sub`, backgrounded ≤30-min): authored a new **Phase 2 — Clone-over-egress capability** (one-`Task.Metadata`-key opt-in, agent-side `git clone` at probe-Q/R egress shape, 4 milestones ≤1d, **≤120 LOC**, frozen-core-confirmed Conflict Surface, no-live-call goldens + live-gated E2E), superseding the iter-45-REFUTED mount Phase-2 while preserving the evidence trail. Bounded quorum (`gpt5-6-sol` + `gemini-3-1-pro` + controller): **round 1 BLOCKED** on a real HEAD-review evidence-check bug (echoed-SHA `==` empty `CloneSHA`) → **FIXED** (conditional check, +positive test); **re-quorum BLOCKED again** on **two NEW, sound, convergent** objections — gpt5-6-sol: unbounded clone/download/`ailang check` violates **Standing Rule 6 (bounded waits)** (add a bounded-execution/timeout-reuse section); gemini: the `Metadata` opt-in is a programmatic silent-fallback hole on the shared `executor.Task` API (add typed `RequiresEgress`+`CapNetworkEgress` gate). Bounded budget exhausted (1 revision + 1 re-quorum) → **PARKED needs-human-review** with a decision-ready ⛔ PARK-NOTE (both objections + convergent fixes + one-pass unblock recommendation). Fix #1 re-widens the shared executor contract (the scope call the designer deliberately avoided) → Mark-level decision. Doc-only commit to dev (`dcdfbab29`); Gate-3b: CI has **no** `on.push.paths` filter → CI + Build-and-Release run on every dev push (Deploy-Docs is `docs/`-filtered → N/A); bounded-polled to green. Rotation write-back → `claude:claude-fable-5`. Detail: log entry 56.

## STATUS 2026-07-18 — ITERATION 50: **dev CI RED outranked the queue** — `make fuzz-parser` fuzztime-boundary flake fixed-forward (commit `c8f61e212`, dev CI green observed)

Gate-1 caught **dev CI RED** @ `3556e9377` (a *data-only* dashboard commit) with `FuzzParseExpr: context deadline exceeded`. Diagnosed as a **flaky fuzztime-boundary artifact, not a regression**: parser + testdata + fuzz test unchanged 7 days; the previous **11** dev CI runs green on the same code; local repro `go test -fuzz=FuzzParseExpr -fuzztime=2s` **PASS 3/3**; **no crasher persisted** (timeout ≠ panic); CI ran **4 workers** vs 16 local → a slow deeply-nested input (DELIM_STACK depth 9) in-flight when `-fuzztime=2s` expires gets its worker context cancelled and reported as a FAIL. Fix-forward (Gate-1 small): `make fuzz-parser` now retries **only** the transient `context deadline exceeded` (no `Failing input written`) once; a real crasher fails fast, a persistent slow-parse timeout still fails on the second attempt. Commit `c8f61e212` (`make/test.mk`+CHANGELOG) direct-to-dev; **Gate-3b CI green observed** on `c8f61e212` (~9 min, Fuzz step success + Build-and-Release success; Docs-Deploy path-filtered N/A). Inbox: 2 nightly "regressions" (`fold_reduce`/`cli_args`, local qwen3, 2 trials) **RULED noise** — data/docs-only build delta (no compiler code in 36h) + prev run was `rag_on` vs today non-rag → config delta, not a code regression → gap-finder candidates. No queue item routed (red-dev consumes the iteration). No skill/process/routing change. Parked-for-Mark backlog unchanged. Detail: log entry 55.

## STATUS 2026-07-18 — ITERATION 48: DX-tooling pick **M-TOOLING-DETERMINISTIC REALITY-CHECKED → premise SUPERSEDED**; regression guard landed, scope-close **PARKED for Mark**

Both parked-for-human items (m-prompt-footguns-to-diagnostics iter-47; m-check-strict-fallbacks) had **no new `@MarkEdmondson1234` answer** on #399/#329 since watermark (the only Mark comment, the 17:06:40Z philschmid link, was already actioned by iter-46 — a stale-watermark-file split re-surfaced it: `mission-329-last-seen` held it but `mission-399-last-seen` lagged; advanced). Fell back to the queue [NEXT] → DX-tooling group. Reality-checked the older, fuller of the two: **M-TOOLING-DETERMINISTIC** (Oct-2025, v0.3.15-era, 898 lines). Live-repro at HEAD `v0.29.2-362`: the `ailang normalize`/`suggest-imports`/`apply` CLI trio **does not exist**, BUT its **premise is obsolete** (`prompts/repair_prompts/` deleted; eval flow is now agentic multi-turn with per-edit `ailang check` feedback — not single-shot fragments needing LLM repair) AND its **core capability already ships** as `normalizeProgram` in `internal/eval_harness/normalize.go` (deterministic module-wrap + std/io inject + bare-call fix; internal fn, not a CLI trio). Per-goal: G1 normalize=**SHIPPED**; G2 suggest-imports=**PARTIAL/ABSORBED** (std/io only; general symbol→import now met by implicit prelude + agentic feedback + `ailang docs`); G3 apply=**obsolete** (agentic agents edit files directly). Further eroded by MOD014 + `--caps auto`. **Durable close (not bare bookkeeping):** doc header → REALITY-CHECKED with a per-goal disposition table + preserved original; regression guard `TestNormalizeProgram_MToolingMotivatingFragment` pins the shipped capability against the doc's exact json_parse fragment + its std/json boundary + determinism (PASS). **Mark scoped DX tooling "both in" → the controller does NOT unilaterally rule out**: scope-close PARKED for Mark (recommend SUPERSEDED-close, or a much-smaller "expose `normalizeProgram` as `ailang normalize`" doc if a non-agentic external audience is real; prefer `m-ailang-fmt` for any DX budget). **Routing (FLAGGED):** controller-lane reality-check (objective repo evidence, no generation layer → no evaluator/generator≠judge needed), same pattern as iters 45/46. Detail: log entry 53.

## STATUS 2026-07-17 — ITERATION 47: clause-3 prompt-teaching cluster REALITY-CHECKED → consolidated into a diet-aligned **diagnostics** design doc, **PARKED needs-human-review** after the bounded quorum round (commit `a7b484395`)

G4 (#399) stayed parked awaiting Mark's greenlight → fell back to the queue [NEXT] (clause-3). Live-repro'd the three stale prompt-teaching docs (v0.24.0-era, Apr–Jun data) at HEAD `v0.29.2-354`: **split-list-operations = GHOST** (prompt v0.16.2 already teaches split→map→join via `mapSlicesJoin` + inline return type); **single-file-module = REAL** (10%; error code **MOD002** is defined+published but has ZERO emission sites → parser falls through to opaque `PAR_NO_PREFIX_PARSE` on the 2nd `module` token); **dot-notation = REAL-but-marginal** (2%; opaque record-unify). Authored ONE consolidated doc `planned/v0_30_0/m-prompt-footguns-to-diagnostics.md` routing both real footguns to the **diagnostic lane** (ZERO prompt lines — the prompt is 2535 vs the ≤1500 diet target; diagnostics are the m-diagnostic-coverage-ratified replacement). Designer = `claude:claude-fable-5` (rotation; probe rc=0; billing-guarded claude-sub). Quorum (gpt5-6-sol + gemini-3-1-pro + controller) ran the FULL bounded round — **author → reject → revise → re-quorum → reject**. R1 caught a real frozen-core violation (Phase-3 stdlib catalog in `internal/types`) → revised to generic-only. **Part A (MOD002/PAR_MODULE_PLACEMENT module diagnostics) + Part B (ghost-close guard) UNANIMOUSLY ACCEPTED both rounds**; parked on two narrow named fixes (Phase-3 primitive-detection premise — remedy "defer Phase 3"; Part-A `seenModule`-on-recovery). Recommended unblock in the doc's PARK-NOTE: drop Phase 3 → extension backlog, apply gemini's fix, ship the accepted Part A+B (~1.25d). Detail: log entry 52.

## STATUS 2026-07-17 — ITERATION 46: HUMAN DIRECTIVE (#399, outranks queue) — gap **G4 RESHAPED: egress param FOUND + CLONE-OVER-EGRESS live-verified end-to-end** (answers "can gemini git clone the codebase?" = YES)

Mark replied on #399: *"can you look at this … https://www.philschmid.de/managed-agents-gh"*. Investigated: that post is the Gemini **Developer** API surface (`ai.google.dev`, `google-genai`, API-key), a *different contract* from our executor's Vertex `aiplatform.googleapis.com` (ADC) — but it revealed the insight iter-45 missed: the egress-enable param is a **structured list** `environment.network.allowlist:[{domain,transform}]`, not a scalar flag (iter-45's 6 guesses were all boolean/enum → all missed). Re-probed OUR Vertex endpoint with that shape (same ADC harness, probes O–R): **`network.allowlist:[{domain:"*"}]` is accepted and provisions an egress-enabled sandbox** (specific-domain + header-`transform` are *"not supported now"* on Vertex — wildcard only). Probe **R** = money shot: an egress-only env (NO data source) **cloned the public ailang repo end-to-end** — `git clone` OK, `rev-parse HEAD`=`806b3b4a4` (current dev), file listing + `go.mod` returned verbatim. **So iter-45's "nearest path = GCS mount, large lift" is superseded: for a PUBLIC repo the agent clones itself over egress — no mount/GCS/inline.** New dominant option **(d) clone-over-egress** (small; directly delivers #399's "gemini can git clone the codebase" for the reviewer role) — recommended; pending Mark's greenlight to decompose a small Phase-2 sprint. Couldn't live-confirm the Developer-API surface (available `GOOGLE_API_KEY` invalid even for generateContent → needs a valid interactions-preview key, parked). **Routing (FLAGGED):** controller-lane live-repro (objective API accept/reject + one public-repo clone; continuation of iter-45's spike). Probes O–R added to the CI-inert `managed_agents_live_test.go`. Detail: log entry 51.

## STATUS 2026-07-17 — ITERATION 45: HUMAN DIRECTIVE (#399, outranks queue) — gap **G4 Phase-1 Vertex contract-discovery spike RUN → PREMISE REFUTED**; PARKED on a scope decision for Mark (PR #410 → `24f9e14c9`)

Mark authorized the spike (#399: "yep do the vertex contract spike") — the ADC-gated live probe the G4 quorum demanded before Phase-2 code. Ran it directly as a controller-lane live-repro (Gate-2-class external-ground-truth + security-sensitive credential scrubbing before a public commit; the Phase 2-5 code sprint is NOT this iteration). **14 credential-free probes vs the live Vertex `interactions` endpoint (project ailang-dev, global, Api-Revision 2026-05-20), all cheap request-validation 400s — no sandbox provisioned, negligible cost.** Result **REFUTES the doc's mount model**: `repository` and `inline` source types DON'T EXIST (`Unsupported environment data source type … Must be one of: [gcs, skill_registry]`); data sources are gated behind network egress that is OFF by default (`Network egress is not enabled … Cannot specify data sources`); `environment.network` is real but its egress-enable param is undiscovered (6 idiomatic guesses rejected — needs the Vertex proto). The git-repo+inline design (≤250-LOC, ≤1MB-inline) is not expressible here; nearest real path is a GCS-backed mount = a larger, unscoped redesign. **Decision for Mark:** (a) redesign around GCS, (b) shelve & keep the prompt-packed diff bridge (recommended), (c) probe `skill_registry`. Reproducible env-var-guarded probe (`managed_agents_live_test.go`, CI-inert) + full VERIFIED-LIVE record committed. **Routing (FLAGGED):** spike run inline on the opus controller (not a pinned executor sub-agent) — investigative live-repro + security-sensitive; the WRITE-UP got an independent **sonnet** evaluator PASS (generator≠judge on interpretation). Data-before-conclusions working exactly as designed. Detail: log entry 50.

## STATUS 2026-07-17 — ITERATION 44: gap **G3 designer-rotation live test CONFIRMED** (`codex:gpt-5.6-sol` authored + revised a full design doc); **G4 design PARKED needs-human-review** — quorum gates ratification on running the live contract-discovery spike (PR #409 → `d422f727a`)

Picked gap-priority **G4** (gemini repo-mount); it needs a NEW doc, so authoring it IS the **G3 designer-rotation live test** (two gaps, one iteration). Reality-check: G4 REAL+unstarted (`managed_agents.go:164` hardcodes empty `{"type":"remote"}`, no `--env-repo` flags, `Task` has no `EnvSources`). Rotation last-used=`claude:claude-fable-5` → designer=`codex:gpt-5.6-sol`. **First codex-designer fire = SUCCESS**: authored a format-complete 427-line doc (cited HEAD facts, typed-vs-Metadata tradeoff, Axiom +7) + a competent revision pass — all via the cross-provider `workspace-write` worktree recipe (previously verified only for the executor role; carries the design-doc directive cleanly). **Quorum ×2 rounds (gpt5-6-sol+gemini-3-1-pro+controller, all present both rounds): reject→revise→reject.** R1: unverified wire contract + programmatic silent-fallback hole. Revision: Premise Verification Log + Phase-1 contract-discovery spike hard-gating the encoder + `CapEnvironmentSources` pre-dispatch gate. R2 still BLOCKED — a design on a DOC-ONLY external API contract can't be ratified until the live spike is RUN+RECORDED (reviewers converge on the doc's OWN Phase 1) + a new shallow-clone catch. Per the one-revision cap → **PARKED needs-human-review**; doc landed on dev with a PARK-NOTE (PR #409 → `d422f727a`, auto-merged on green). **G3 mechanism CONFIRMED** (rotation+cross-provider-designer+quorum-gating end-to-end); content-park ≠ mechanism-fail. Codex can't self-commit under the sandbox → controller finalized, crediting codex. G3 evidence row: `(designer=codex:gpt-5.6-sol, quorum=reject→revise→reject)`. **Unblock G4**: human-authorize the ADC-gated live Vertex spike. Detail: log entry 49.

## STATUS 2026-07-17 — ITERATION 43: gap **G1 (gemini FIRST LIVE REVIEWER FIRE) + G2 (3-provider quorum) CONFIRMED**; shipped the reasoning-model truncation fix that makes gemini a RELIABLE reviewer (PR #408 → `885725f06`)

Picked gap-priority **G1** (Mark #399; G1–G4 outrank the clause queue). Live `ailang design-quorum` (default reviewers `gpt5-6-sol`+`gemini-3-1-pro`) → **both present, gemini `reject` $0.023** = G1 (gemini's first clean live reviewer verdict) + G2 (OpenAI+gemini+claude = 3 providers) CONFIRMED. Reality-check surfaced the real blocker: gemini-3.1-pro ("2× reasoning") counts THINKING tokens against the quorum's `reviewMaxTokens=4096` `maxOutputTokens` cap → a substantive review truncates its JSON mid-object → `invalid`/absent → **silent N-1 quorum** (iter-42 artifact, byte-identical reviewer code; intermittent — the exact iter-39+iter-42-logged friction, now "next-blocked" G1). **Fixed root-cause + fail-loudly**: cap 4096→16384 (budget gating unchanged — pre-flight uses a fixed estimate); a residual `finish_reason=length` now surfaces an explicit truncation error not "malformed JSON"; wired the discarded gemini `finishReason`→normalized `ai.Response.FinishReason`. 2 regression tests, gofmt clean, pkg tests green, live post-fix quorum clean. Inline-controller fix (iter-41/42 quorum-tool-fix precedent; no new design doc → designer/planner/executor/evaluator NOT invoked). PR CI green on the merge-test tree → auto-merged; installed binary rebuilt to `885725f06`. Calibrated: one clean run can't PROVE an intermittent bug gone — the deterministic tests + 4× cap raise carry it, and truncation now fails LOUD. **G1/G2 done → G3 (designer rotation, `codex:gpt-5.6-sol`) is next.** Detail: log entry 48.

## STATUS 2026-07-17 — ITERATION 42: re-attempted PARKED m-check-strict-fallbacks (both iter-41 blockers cleared) → **re-PARKED with a sharper, quorum-validated blocker**; found + fixed a stale-binary regression that had silently disabled the #407 quorum fix

Both iter-41 blockers were gone (Fable designer back via `claude-sub`; #407 restored the OpenAI quorum reviewer), so the iter-41 park (run on a BROKEN solo-gemini quorum + degraded controller-as-designer) was never a valid verdict → re-attempted. Resolved the "OPEN design decision" to option (a) (syntactic surface-AST pass) via the **Fable designer** (rotation seed; hooks live-verified: `pipeline_single.go` astFile@159/Warnings@189-198, `pipeline_module.go` mod.File@318/Warnings@337,388) + grounded Pattern C in AILANG's **language-enforced uppercase-constructor rule** (`PAR_VARIANT_NEEDS_UIDENT`; live-probed). **Gate-2 miss caught:** I skipped the Verification-Protocol Rule-1 rebuild — the installed binary was `de9556413` (pre-#407), so my first re-quorum re-hit the exact `Missing 'proposed_fix'` 400 (gpt5-6-sol silently unreachable). **Rebuilt** (`make quick-install` → 326/77e7dccc9); the clean re-quorum made gpt5-6-sol **present** — and it immediately caught a **goal-contradicting** design error the designer AND I both missed: the motivating incident `None => Ok(jo([]))` has `jo` as a **lowercase function call**, which option (a)'s pure-syntax rules NEVER flag → the pass fails its own primary goal; catching it needs resolved callee identity → **refutes option (a)**. Synthesis **BLOCKED** → **re-PARKED needs-human-review** (Standing rule 2). The doc now carries the full REBLOCK write-up + the architecture fork for a human. Doc-only commit `b159305ae`. Gemini truncated its quorum response BOTH rounds (recurring tooling friction, logged). Detail: log entry 47.

## STATUS 2026-07-17 — ITERATION 41: pick-time quorum PARKED m-check-strict-fallbacks (needs-human-review); SHIPPED the mission-infra bug it exposed — `design-quorum` was silently a solo-gemini veto (OpenAI reviewers 400'd on every run)

Picked m-check-strict-fallbacks (clause-2, ~1d, headless-viable — clause-3's remainder needs the Fable designer [quota-gone until 2026-08-01] or an eval rotation). QUORUM-AT-PICK on the pre-quorum doc: round-1 reject (premise gate — `internal/check/` does NOT exist; corrected target → `internal/pipeline/` per the `warn_split_args.go` precedent), round-2 reject (a design-layer AST-vs-Core coherence gap my own revision introduced) → per the ONE-round cap **PARKED needs-human-review** (both reviewers reject → well-supported, not a solo artifact). The friction exposed a real bug: `reviewSchema` omitted `proposed_fix` from `required`, which OpenAI strict `json_schema` mode 400s → gpt5-6-sol `unreachable` every run → quorum degraded to solo-gemini. **Fixed** (proposed_fix → plain `string` in `required`; a `["string","null"]` union satisfies OpenAI but Vertex rejects unions — plain required string is the one cross-provider form); **live-verified both gpt5-6-sol AND gemini-3-1-pro now present**; regression guards + CHANGELOG. Doc hardened (Premise Verification + OPEN layer decision spelled out for the planner). PR #<PENDING>, worktree off origin/dev. Detail: log entry 44.

## STATUS 2026-07-17 — ITERATION 40: clause-3 **m-dx-expected-fail-fixes GHOST-CLOSED** (PR #406, eval PASS 92/100 r1) — Gate-2 live-repro found 0 of 4 "bugs" needed a language fix; closed with CI-gated regression guards; the block-RHS-`let` parser ASI footgun split to a new evidence-gated backlog item

Picked the `[NEXT]` clause-3 sub-item flagged LARGELY-GHOST at iter 32 (over 3 sibling prompt-teaching items, which need an eval rotation to verify — GPU/API-billed — a poor headless fit). Gate-2 live-repro at `origin/dev 1ee919386` CONFIRMED it: **Bug 4 (effect_budgets)** works — the doc's repro put `--caps` AFTER the filename (ignored); with the flag first, `@limit=N` enforcement fires ("budget exhausted: semantic limit=3"). **Bugs 1/2 + 2 match_foreign files** = good teaching diagnostics / intended type-rejections. **Bug 3 (serve_api_webhook)** = non-canonical example (omitted `;`/`in` after a block-RHS `let`, deprecated string `++`). Closed with regression guards: 3 examples fixed to canonical syntax + promoted to `examples/runnable/` (now CI-gated by `verify-examples`), README budget claim corrected, manifest de-drifted (2 mispathed `contracts/` entries repaired — not phantom). **Opus executor / Sonnet evaluator PASS 92/100 r1** (generator≠judge; zero Go/parser changes; `make test` + `verify-examples` green). The real-but-minor block-RHS-`let` separator inconsistency split to `m-parser-block-let-separator` (planned/v0_30_0, evidence-gated — default-bias-not-core). Gate-1: local dev was 2 commits behind origin (iter-39's PR #405) — worked from a worktree off origin. Detail: log entry 43.

## STATUS 2026-07-17 — ITERATION 39: fleet **(c1) m-gemini-evaluator-diff-bridge LANDED** (PR #405 → `ae5f0a00f`, eval PASS 96/100 r1) — a sandboxed gemini evaluator can now review a sprint's UNCOMMITTED worktree diff; the backend-reliability blocker CLEARED

Iteration 38 parked (c1) on Vertex backend reliability ("do not pick until a bounded `ailang exec gemini` probe returns a response"). Gate-2 reality-check ran **4/4 bounded probes → all SUCCESS** (8–11s, ~$0.01 each) → blocker cleared, (c1) pickable (NEXT-FIRST fleet step, serves Mark #399). Full inner loop: **designer** — Fable **quota-exhausted until 2026-08-01** ("reached your specified API usage limits") → fell back to **opus** + FLAGGED (graceful, never wedged); design-doc-creator's creation-time quorum degraded (gpt5-6-sol unreachable via an OpenAI structured-output infra bug; gemini-3-1-pro truncated) → controller PROCEED, gemini's untracked-files objection incorporated. **Opus planner** (verified 8/8 seams; caught the schema-mirror discrepancy — `GeminiVerdict` is a documented adaptation, not a byte-mirror of the frozen `quorum.ReviewResult`; pinned the lowest-risk `LastFencedBlock` thin-wrapper, no call-site rename) → **Opus executor** (worktree `internal/eval_harness/gemini_evaluator_bridge.go`: `BuildDiffBundle` untracked-inclusive + 256 KiB ceiling + LOUD truncation; reasoning-only directive; `GeminiVerdict` parse/validate; `RunGeminiEvaluator` injectable caller seam + caller-enforced `VerificationDegraded`; 12 non-vacuous tests) → **Sonnet evaluator** (generator≠judge: opus≠sonnet) **PASS 96/100 r1**, non-vacuity + frozen-contract independently confirmed; NB-1 folded into a ctx-threading hardening commit (`exec.CommandContext` — the fleet-(c) caller-ctx watch-item). Executor/`quorum`/`exec.go` byte-identical. **Default evaluator STAYS sonnet** — this ships the CAPABILITY, not a routing change (defaulting to gemini needs the evidence rule + a live diff-bridge fire). PR #405 → squash `ae5f0a00f`, dev CI green (test/lint/build required; SonarCloud advisory-red, non-required). Doc → implemented/v0_30_0. Detail: log entry 42.

## STATUS 2026-07-16 — ITERATION 38: HUMAN DIRECTIVE (#399, outranks queue) — evaluator default **fable → sonnet**; gemini-as-evaluator VERIFIED not-viable-today (server-side sandbox can't see the worktree + live probe timed out)

Mark commented on #399: *"once we have gemini via managed agents and openai we can use one of those instead for evaluator? so default can be gemini (if able to git clone the codebase etc)? otherwise sonnet-5"*. Resolved his conditional with data. **gemini (managed_agents) is NOT viable as the evaluator today, two independent counts**: (1) **architectural (code-proven)** — the request body carries only `Directive`+`SystemPrompt` over a server-side `CapRemoteSandbox` (`managed_agents.go:164`); no repo upload, so it cannot see the sprint's UNCOMMITTED worktree changes nor re-run local tests (at most `git clone` the *public* origin/dev, which lacks them) — exactly Mark's "if able to git clone" gap; (2) **operational (live-observed)** — a bounded `ailang exec gemini` probe timed out (`http2 timeout awaiting response headers`, same class as iters 36-37). Per Mark's ladder → **sonnet-5**: Agent-tool-pinnable (fable is not — F1, so the fable default silently re-routed to sonnet every iteration anyway: 31/36), distinct from the opus executor (generator≠judge restored & now ENFORCEABLE), cheap, behavioral. Changed: driver `MISSION_EVALUATOR_MODEL` default fable→sonnet + routing-policy table + independence caveat RESOLVED. Follow-up queued: **fleet (c1) m-gemini-evaluator-diff-bridge** (ship the sprint diff into the directive + backend reliability). No CI risk (doc + driver-comment + one env default). Detail: log entry 41.

## STATUS 2026-07-16 — ITERATION 37: fleet **(c0) m-gemini-exec-project-plumbing LANDED** (PR #401 → `60351087b`, eval PASS 96/100 r1) — `ailang exec gemini` now reaches the Vertex Managed Agents backend; unblocks fleet (c)'s parked M0/M4 gemini reviewer lane

The ≤1d unblocker surfaced by iteration 36. Live-repro confirmed the gap at HEAD (`managed_agents: GCP project not set`), root cause `cmd/ailang/exec.go:executeCLI` built `executor.Task{}` with no `GCPProject`/`GCPLocation` (the eval harness sets them per-model; the CLI path never did). **Fix** (minimal, +13 LOC code): `resolveGCPProjectEnv()` (`AILANG_CLOUD_PROJECT` → `GOOGLE_CLOUD_PROJECT`, coordinator precedence) + set both fields on the shared Task; empty location defers to executor `defaultLocation="global"`, unset project keeps the loud error (no silent default). **Live-verified by the controller**: env-unset → loud error preserved; `AILANG_CLOUD_PROJECT=ailang-multivac-dev` → error moved to Vertex `HTTP 400: Resource setup has just started` (project REACHED the backend). Non-vacuous `t.Setenv` regression test. Full loop: **Fable designer** (`claude:claude-fable-5` CLI lane, N−1 quorum PROCEED) → **Opus planner+executor** → **Fable evaluator** (true-Fable CLI lane, PASS 96/100 r1). ⚠ **Routing FLAG**: evaluator ran on the `claude:claude-fable-5` CLI lane, NOT the doc-prescribed sonnet re-route (iteration 36 hit the identical `MISSION_EVALUATOR_MODEL=fable` env and chose sonnet per the ≥3-datapoint gate) — recorded as an evidence datapoint + a doc-inconsistency retro note, NOT a ratified policy change. Detail: log entry 40.

## STATUS 2026-07-16 — ITERATION 36: fleet (c) m-mission-quorum-agentic-verify **CORE LANDED** (M1-M3, PR #400 → `0e83a1b12`, eval PASS 91/100 r1) — agentic reviewers that VERIFY-not-just-reason; M0/M4 gemini parked on a real `Task.GCPProject` plumbing gap

Mark's option-(a) decision unparked iteration 34's Gate-2 blocker (`proposed_fix` optional, not validated,
verdict contract frozen); doc was already quorum-cleared so routing started at sprint-planner. **Shipped
M1-M3** (the provider-independent core): `agenticCaller` behind the existing `JSONCaller` seam producing the
frozen `{verdict,strongest_objection,catch}` JSON via the coordinator executor layer (post-hoc cost cap vs
`result.Cost`, N-1 degradation, read-only `Kind=="question"`); `ShouldEscalate` two-tier trigger
(premise-class ∨ high-stakes ∨ Tier-1-split) + additive-optional `proposed_fix`; Tier-2 codex+claude
read-only verify. 43 tests pass (29 new, deterministic -count=5); verdict contract independently verified
UNCHANGED. Evaluator **re-routed fable→sonnet** (fable Agent-tool-unpinnable; $MODEL=opus would collide with
the opus executor → alias-lane generator≠judge guard) PASS 91/100 r1. **M0 (gemini network probe) BLOCKED by
a real gap**: `ailang exec gemini` fails `GCP project not set` — `cmd/ailang/exec.go:336` builds `Task{}` with
no `GCPProject`, managed_agents default is `""` (filled per-task only by the eval harness). Fix = new queue
item (c0), prerequisite for M0/M4 + any gemini reviewer/evaluator lane. Detail: log entry 39.
## STATUS 2026-07-16 — ITERATION 35: RED-DEV fix (outranks queue) — CI + Build-and-Release both green on `2bb3de2c5`; weekly bookkeeping thread rotated #329 → #399

Two independent reds observed at HEAD (`fe7c13efa`). **(1) CI `verify-examples`**: the v0.30.0
vision-input merge (`8c3de5ce8`) added `examples/runnable/ai_vision_input.ail` but left the manifest
`statistics` aggregate stale (recorded 185/173 vs calculated 186/174 — `validate_manifest --ci` drift);
bumped total/working/coverage. **(2) Build-and-Release Windows** `TestStreamNDJSONPost_Success`:
`[SSEData Opened SSEData SSEData]` — the reader goroutine raced an `sse_data` event ahead of `Opened`.
Surfaced at a DOC-ONLY commit (`fe7c13efa`) → confirmed pre-existing race, not a regression. Systemic
fix (Critical Principle 3) across ALL FOUR stream connectors (NDJSON, WebSocket, SSE-GET, SSE-POST):
enqueue `Opened` into the buffered eventBuffer BEFORE starting the reader goroutine → `Opened` is always
event[0]. Verified `go test ./internal/effects -race -count=20` (green, 111s). Commit `2bb3de2c5` →
dev CI + Build-and-Release both green (Gate 3b, bounded poll). Weekly rotation (Gate 5): #329 (53
comments, created before Mon 07:00 2026-07-13) closed → #399. Detail: log entry 38.

## STATUS 2026-07-16 — ITERATION 34: fleet item (c) m-mission-quorum-agentic-verify **PARKED needs-human-review** at Gate-2 quorum-at-pick (dogfood: text quorum blocked the AGENTIC-quorum doc for premises TRUE-in-code that text reviewers structurally can't verify)

The picked fleet-(c) doc had no quorum artifact → QUORUM-AT-PICK fired (2 rounds, ~$0.05). Round 1
BLOCKED on a stale Verification-Log row (its `find -name '*quorum*'` couldn't match doc-named
artifacts) + `proposed_fix` wording — controller integrated inline (mechanical, known-correct),
re-quorum. Round 2 still BLOCKED on DEEPER objections: gpt5-6-sol's "reuse premise unverified" the
controller **REFUTED by code-read** (provider_executor.go exposes ctx-cancel/`Timeout`/`CostUSD`/
read-only `AllowedTools` + `WorkingDir` worktree — reuse HOLDS); gemini's is a REAL open authorial
decision (`proposed_fix` required-on-reject vs contract-unchanged → pick optional-not-validated **(a)**
or bounded contract-extension **(b)**). Per the one-bounded-round gate → PARKED. **≈2-min unblock**:
settle (a)/(b) + add the code-cited Verification-Log rows → route to planner. **Meta-finding**: the
TEXT quorum-at-pick can't verify code, so it reject-by-defaults exactly the premise class the agentic
tier exists to check — a live datapoint FOR building item (c), and a Gate-5 process note (the gate
should let a controller code-refutation of a PREMISE-class objection count, not force a park). No code
shipped; doc + queue updated. Detail: log entry 37.

## STATUS 2026-07-16 — ITERATION 32: FIRST cross-provider codex live-fire — `20251013_auto_caps` M1 (`--caps auto`) LANDED (PR #397 → `e542065c0`); executor = OpenAI codex gpt-5.6-sol, evaluator = Sonnet PASS 98/100 r1; codex real-run recipe corrected (Gate-5 skill edit)

The armed one-shot override fired: `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol` executed a real sprint
end-to-end across THREE providers/roles — Opus planner (refuted the doc's ~200-LOC new-package
mechanism → 74-line reuse of the existing `iface`/`TFunc2`/`EffectRow` required-effect path), **codex
gpt-5.6-sol executor** (OpenAI, ~4.5-min metered run, `--caps auto` infers the entrypoint's effect row
and grants exactly those), **Sonnet evaluator** (generator≠judge held: openai≠anthropic; the fable pin
is unenforceable in the Agent tool so the F1 guard re-routed evaluator→sonnet + FLAG) PASS 98/100 r1.
Gate-2 live-repro also caught that the alt candidate `m-dx-expected-fail-fixes` is largely a GHOST
(effect-budget `@limit` runtime enforcement works at HEAD). **The first real codex run exposed that the
Gate-3 codex recipe had only ever been verified against the text probe** — the real-run invocation is
underspecified: it needs `--sandbox workspace-write` + `--add-dir` GOCACHE/GOMODCACHE, cannot self-commit
(the worktree `.git` lives under the non-writable main checkout → controller finalizes from the
uncommitted worktree diff), and must run backgrounded (30-min cap > the harness 10-min foreground bash
limit). Two+ frictions, one gap → the retro-lane skill edit rewrote the recipe to the empirically-verified
form. Detail: log entry 35.

## STATUS 2026-07-16 — ITERATION 31: m-mission-agentic-provider-routing M1b+M2 LANDED (direct-on-dev `956fda55c`+`8d12e8e9c`, eval PASS 87/100 round 1, hardening `1c964aae2`); M3 PARKED w/ protocol; F1: `fable` is NOT a pinnable Agent alias

The mission-infra P0 closed its executable slice headless: Gate-3 `provider:model`→bounded
`codex exec` recipe (zero Go — planner found registry/DryRun/codex executor pre-existing since
v0.22.0; the gap was only the missing spawn recipe), codex probe live-verified (gpt-5.6-sol,
exit 0, executor + evaluator reproduced identically), charter right-sizing table + provider/
agent/cost evidence rows (first new-schema rows written in log entry 34, incl. the loop's first
metered-OpenAI rows). Evaluator (Fable ≠ Opus executor) PASS 87/100 round 1, then surfaced
**F1 HIGH: the Agent tool accepts only sonnet|opus|haiku pins — `fable` is REJECTED** (live
InputValidationError; fable roles run by session inheritance only; with opus-first defaults a
fable evaluator pin would have silently become an opus judge on opus work) → hardened same-day:
alias caveat + alias-lane generator≠judge guard (evaluator never falls back to bare $MODEL;
re-route sonnet + FLAG) + F2 `exec` orphan-kill fix. Open by design: first REAL cross-provider
fire (opt-in env), M3 planner A/B parked until 3 quorum docs accrue. Nightly binary_tree_sum
"regression" triaged as model noise (9/9 same-night rotation pass on qwen3-6; alert was qwen3-5
N=2). Detail: log entry 34.

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

