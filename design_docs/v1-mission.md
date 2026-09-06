# V1 Mission — work the backlog to a v1.0.0 release

**Type**: Long-running mission (peer of [motoko-mission.md](motoko-mission.md)); advanced by a
scheduled outer loop on the always-on rig, coordinated by Anthropic with a Codex Sol subscription
fallback when the Anthropic quota is unavailable.
**North star**: Ship AILANG v1.0.0 — a release whose bar is *written down, met, and verified*,
with the backlog worked through the honed inner loop (design-doc → sprint-plan → execute → evaluate)
rather than ad-hoc sessions.
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration. **Scheduling: launchd `dev.ailang.mission-control`** (CONTINUOUS since
2026-07-10 per Mark: StartInterval 2h + overlap guard = back-to-back iterations, ≤2h idle; was
22:00-nightly for the first supervised runs) behind the billing guard — API keys are stripped from the environment
(subscription-or-nothing by construction) and a cheap auth probe runs first: keychain OAuth
suffices while the rig is logged in (verified 2026-07-10); `CLAUDE_CODE_OAUTH_TOKEN` in
secrets.env is an optional belt-and-braces for post-reboot login screens. Exhausted/unavailable
Anthropic probes fall through to the ChatGPT-subscription Codex Sol controller; the driver refuses
loudly only when both provider lanes fail. The Claude Code
scheduled-tasks path was TESTED AND RULED OUT for this job (2026-07-10 canary): that system is
desktop-side — tasks landed on /Users/mark (Mark's machine, not the rig) and a probe task never
dispatched even there (a June one-time task was also found a month overdue). Wrong machine +
unreliable dispatch → launchd is primary, not fallback.
**Log**: [v1-mission-log.md](v1-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue
[#329](https://github.com/sunholo-data/ailang/issues/329) — every iteration posts its morning
report there as a comment (Mark follows by email via issue subscription, no Claude login
needed); driver crashes post there too.

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

The single source of truth for the values that differ per mission. The **one** `mission-control`
skill reads this block (and the driver env it exports) instead of hardcoding — so the same skill,
unforked, runs any mission. V1's values are the skill's built-in defaults, so nothing here changes
current behavior.

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/v1-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `v1` (driver: `MISSION_NAME`; v1 keeps the legacy un-namespaced
  `~/.ailang/state/mission-*` paths bit-for-bit — see M1)
- **Bookkeeping issue**: origin `#329`, rotates weekly; live number in
  `~/.ailang/state/mission-gh-issue` (this week `#422`), watermark in `~/.ailang/state/mission-329-last-seen`
- **CI workflows Gate 3b / Gate 1 poll**: `CI`, `Build and Release`, `Deploy Documentation to GitHub Pages`
- **Verify profile**: `go-compiler` — this repo compiles the AILANG toolchain, so gates rebuild
  BOTH binaries (`make quick-install && make build`) and run `make test`; `~/go/bin/ailang` (PATH)
  and `bin/ailang` go stale independently (confirm `--version` == `git describe`). (The alternative
  profile `ailang-code`, for an AILANG-code repo like Ailang World, uses the shipped binary's own
  gates — `ailang check` / `ailang test` / `ailang ai-check` — with no compile step.)

---

## Human Decision Ledger (authoritative current state)

This marked table—not STATUS prose and not the rolling GitHub thread—is the source of truth for
which decisions are open. Validate with `scripts/mission_decisions.sh --check`; list the asks with
`scripts/mission_decisions.sh --open`. Rows are append-only, IDs are never reused, and a human
answer changes the row to `RESOLVED` in the same iteration that consumes the directive. Historical
STATUS sentences such as “D-1–D-14 stay parked” are snapshots and MUST NOT override this ledger.

<!-- decision-ledger:start -->
| ID | Status | Decision / recorded answer | Evidence |
|---|---|---|---|
| D-1 | RESOLVED | **RETAIN zero-DNS literal-IP validation on the proxy route** (Mark, attended 2026-08-19) — the THIRD option this row surfaced, not either arm as originally posed. `resolveAndValidateIP`'s existing `// Special case: raw IP address (skip DNS)` branch validates a literal-IP target with ZERO network I/O, so the doc's rationale for skipping validation (TOCTOU/DNS-rebinding on the proxy route) never reached that case. Restores parity, un-reds CI, and shrinks the residual trade to HOSTNAMES ONLY, which stays accepted per D-5. | Held because Standing rule 2 forbids narrowing a quorum-cleared security boundary unattended; evaluator endorsed the same option independently. Un-parks `#613` M1 (PR is a DO-NOT-MERGE draft). Acceptance must show `TestNetIPValidation` rc=0 with 7 PASS under the ci.yml proxy poison (`:89-91`, `:350-352`) — it is currently rc=1 with 4 of 7 failing on the branch against rc=0/7 PASS on pristine dev, a firing negative control. Doc + Non-Goals text must be revised so it no longer claims proxied literal IPs are unvalidated. |
| D-2 | RESOLVED | **B — widen to close the nested blocks** (Mark, attended 2026-08-19). Option A was declined precisely because it ships `#614` as accepted behaviour: `ailang test` returns rc=0 while nested checks fail, which is a silent false-green and collides with Principle 2 (no silent fallbacks). That is the same objection that BLOCKED round 2, so A would force through what a reviewer already refused. | Answerable because the bound is measured: exactly **27** expression node types in `internal/ast/ast_expr.go`, so an exhaustive type switch with a LOUD `default:` makes the silently-missed-node failure impossible by construction — the doc's stated reason for rejecting the close-it-properly options (no generic Walk/Inspect in `internal/ast`) does not survive that count. Acceptance must red on BOTH shapes reproduced in the park note: bare `{ a==99; a==2 }` and `if true then { a==99; a==2 } else false`, each with the false check LAST as the firing control. Option C (static error) declined as needlessly destructive given B is feasible. |
| D-3 | RESOLVED | Bound SessionStart brain lookup with timeout behavior. | Landed as commit `1239d9ec6`; current hook contains the bounded lookup. |
| D-4 | RESOLVED | Use the feasible warning-test mechanism rather than the impossible fixture-row mechanism. | Recorded resolved in the `#635` decision thread and implemented in iteration 186. |
| D-5 | RESOLVED | Choose option B: queue the production proxy-boundary change after the test-only sprint. | Mark directive 2026-08-05; recorded in the `m-net-effect-proxy-boundary` queue row. |
| D-6 | RESOLVED | Historical D-6 asks are closed: re-auth completed and the earlier planner/config option A was selected. | `#635` records re-auth resolved; mission history records option A. ID reuse is retained only for history and forbidden going forward. |
| D-7 | RESOLVED | Keep CodeQL on its weekly cadence; wait on a release until World needs one. | Mark attended ruling 2026-08-06, recorded in the findings-batch row. |
| D-8 | RESOLVED | **AUTHORIZE the ordered rig rollout** (Mark, attended 2026-08-19). PARTIAL is an acceptable field result here; the repo work is complete and the remaining risk is deployment ordering, not correctness. | ⚠ **MEASURED 2026-08-19: the rig is ALREADY in the pre-`#618` configuration, so this is live cost, not a hypothetical.** `launchctl getenv AILANG_OLLAMA_V1_STREAM` = empty and `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` = empty (control var also empty, so empty is a measurement), and **no INSTALLED plist sets `AILANG_OLLAMA_V1_STREAM`** — only the repo source copies (`nightly-eval`, `os-rotation-filler`) do, and those are never read by launchd. Flag-off means `ollamaV1Timeout()` falls back to **300s**, which is the shape that cost 895 retries / ~74.6 GPU-hours. **ORDER IS BINDING: install the flag-on plists FIRST, `launchctl unsetenv` SECOND** — the reverse re-creates the defect. NB the 6 `api_error` rows in the 2026-08-19 qwen3.8 run are NOT this: they read `finish_reason=tool_calls and no run_summary`, a different class, checked rather than assumed. |
| D-9 | RESOLVED | **SPLIT W8 out and re-quorum it** (Mark, attended 2026-08-19). W9 disputes design direction OUTSIDE W8, so holding W8 makes an undisputed item wait on a disputed sibling. | `#619` parked at quorum round 2 for a reason that does not reach W8. Split, re-quorum W8 alone, and leave W9 in the umbrella to be routed on its own merits. |
| D-10 | RESOLVED | **B — hold and route next** (Mark, attended 2026-08-19). No third revision. | Two rounds are spent, so Standing rule 2 binds. Decisive on top of that: a third revision widens the Conflict Surface into `internal/types` (effect-row unification at the App constraint), which is a CORE change — and the north star's default routing is extension-not-core. Round-2 measurements stay banked so a future resume is cheap; re-open only with a named trigger, not on a fresh attempt at the same seam. |
| D-11 | RESOLVED | **ADD the short-success guard** (Mark, attended 2026-08-19). | Observed first-party during the 2026-08-19 steering session, which is why this stopped being theoretical: `mission-world` logged `iteration complete (rc=0)` at **2m33s** (00:24:21 → 00:26:54), an elapsed time far below the work the record claimed, and NEITHER watchdog fired because a clean exit code reads as success. A null iteration is currently indistinguishable from a real one at the driver level. Guard on elapsed-vs-claimed, and note the sibling lesson from the pi lane: `stopReason`/rc are evadable in both directions, so the load-bearing assertion is a worktree/output diff, not the exit code. |
| D-12 | RESOLVED | **YES — a human ruling that unblocks a design doc MUST create or un-park its queue row in the same iteration that consumes the directive** (Mark, attended 2026-08-19). | A resolved ruling previously left a P0 doc absent from the queue for **eight days**, and the same class recurred as 'a follow-up filed in another row's prose is not a backlog item' (iteration 197). Immediately live: the 2026-08-19 session resolved `D-1`, `D-18`, `D-MOTOKO-FMT-1` and `D-WORLD-21`, so each of those rows must be un-parked under this rule rather than waiting to be noticed. |
| D-13 | RESOLVED | **C — keep scope, re-root durable writes through `$AILANG_DRIVER_SRC`** (Mark, attended 2026-08-19). | A was declined because M2 pins the filler itself, so excluding it deletes the only delivery path for every later milestone. B (fast-forwarding the source clone from `pin-root.sh`) was declined as fixing the wrong thing: the source clone's lag is EXPECTED and harmless — measured 2026-08-19, motoko's clone was 66 behind while the loop correctly ran `~/.ailang-driver-pin/motoko` at `origin/dev` exactly. The real hazard C addresses is durable state written into an EPHEMERAL pinned worktree. Rollout covers the remaining five entry points (`nightly-eval.sh`, `nightly-lang-eval.sh`, `mission-recovery.sh`, `os-rotation-filler.sh`, `rig-watchdog.sh`); note `rig-watchdog.sh` gained the mission-job re-bootstrap + `launchd-hold` block on 2026-08-19 (`3fc8a46e0`, `c9593936f`), so rebase the rollout onto that. |
| D-14 | RESOLVED | **A — recovery-site detection at `parser_literals.go:562`** (Mark, attended 2026-08-19). | Chosen because it needs NO rewind and fits the parser as-is. The R2 blocker is a fact about the parser, not the doc: fixed 4-token lookahead (`parser.go:134-139`) and `nextToken()` (`:214-220`) is a pure forward shift with ZERO save/restore/rewind methods (control: 125 `*Parser` methods), so the doc's 'lookahead/reparse past an arbitrary subject expression' is infeasible as written. **C is explicitly refused**: adding backtracking touches all 125 methods and is a core change, against the frozen-core north star. B fits 4 tokens only for simple subjects. D (drop) declined — the diagnostic is worth its 1-1.5d at `case`-only scope. Run the transcript scan: it is cheap and sizes the deferred eight spellings behind their named evidence trigger. |
| D-15 | RESOLVED | Remote READ scope is `view`. | Mark ratified; implemented and landed in iteration 198 / PR `#710`. |
| D-16 | RESOLVED | Keep the mission-control runtime skill pinned to the authoritative `.claude` copy. | Attended decision recorded and verified by iteration 205. |
| D-17 | RESOLVED | Handle clean exit separately at each of the three route/A2A/MCP call sites. | Mark ratified; landed in iteration 200 / PR `#714`. |
| D-COV-1 | RESOLVED | **LOCALITY — keep own-package coverage semantics for the gated/badged/Sonar metric; add a SEPARATE non-gating `test-coverage-xpkg` diagnostic** (Mark, attended 2026-08-19). This ratifies the design doc's own recommendation to refuse the change. | The crux inverts the intuition and is why this is a refusal rather than an upgrade: iteration 204's defect was a function with NO own-package test, so `-coverpkg` would have painted it GREEN and suppressed the true-positive signal that forced the fix. A gate that can be satisfied by a distant package's incidental execution is not a locality gate. Un-parks the sprint on `design_docs/planned/v0_33_2/m-coverage-cross-package-attribution.md`. |
| D-ROUTE-1 | RESOLVED | Coordinator and Anthropic-required planner routes fall back to Codex Sol when Anthropic quota is unavailable; executor remains Codex Sol primary with DeepSeek v4 Flash backup and Opus last. | Mark directive, attended session 2026-08-15; unique ID avoids reusing historical D-7. |
| D-18 | RESOLVED | **Ownership scoping, not a claim protocol** (Mark, attended 2026-08-19) — superseded in substance by `c2022c7fa` before the ask was answered: a RED dev now outranks the queue ONLY for the mission that OWNS the repo (V1 for `sunholo-data/ailang`), and motoko hands anchor reds over instead of fixing them. That removes the duplication for reds without any claim file or issue marker, so options A and B are declined and C (accept) stands for any residual non-red overlap. | Prompted by #758/#759: both loops preempted onto the same red and produced the same six files four minutes apart. Gate 2's open-PR check cannot catch it — point-in-time at pick time, aimed at a PAST iteration, blind to a concurrent peer (V1 queried ~18:58Z; #758 appeared 19:05Z). Root cause is structural: the overlap guard is per-mission (`mission-${MISSION_NAME}.pid`) and iterations never take `rig.lock`, so NO cross-mission mutex exists. Revisit only if a non-red collision is observed. |
| D-19 | RESOLVED | **B — true cons cells / structural sharing** (Mark, directive on `#745` at `2026-08-19T10:58:40Z`, body exactly `D-19 : B`; consumed by iteration 229). The front-slack arena is DECLINED as the permanent answer. ⚠ **Letter collision — read carefully:** the superseded doc `m-list-cons-quadratic.md` labels its own options A–D, and **its Option A ("true cons cells") is what this answer B selects**, while **its Option B (the arena), which that doc marked CHOSEN, is now declined**. The ruling itself states the work "needs decomposition first", so iteration 229's deliverable was the decomposition, not the implementation. | Decomposed into **8 sprint-sized pieces, 15.5–21.5 person-days**, in `design_docs/planned/m-list-cons-cells-decomposition.md` (505 lines, 20 first-party verification rows): LC-0 interim communication · **LC-1 representation spike, runs FIRST with an explicit kill criterion** · LC-2 accessor seam + `listrep` ratchet analyzer · LC-3a/3b/3c mechanical migrations · **LC-4 the swap (riskiest)** · LC-5 post-swap tuning. Quorum round 1 **BLOCKED** 2-of-2 external ($0.0884, `absent_reviewers` EMPTY, both `present=true` — a genuine block, not an N−1 degrade). Both objections are narrow-refinement shaped with verbatim reviewer-authored fixes and neither disputes the DIRECTION: `gpt5-6-sol` correctly holds that deleting `ListValue.Elements` does NOT give compiler-enforced immutability (Go has no immutable fields; the owning package is `internal/eval`, which is also the evaluator), so the `listrep` analyzer must be RETAINED and immutability/safe-publication stated as required properties; `gemini-3-1-pro` caught LC-3c being assigned escape sites that live in LC-3a's and LC-3b's packages, which would guarantee merge conflicts across the parallel lanes. **⚠ SUPERSEDED — the owed revision was DISCHARGED at iteration 234** (round 2 blocked 2-of-2, both
premises measured, narrow-refinement carve-out applied; the doc's own Status header and its
"Quorum verification log — round 2" section are the record). The sentence that follows described the
state at iteration 229 and was never updated, so it read as live owed work for 17 iterations — the
transcription class this loop keeps closing, here in its own charter. Historical text follows:
**The owed revision was parked on a LANE, not a human** — codex probed rc=1 (quota, resets 2026-08-20 05:34) and gemini is read-only under `CapRemoteSandbox`, so the designer fell back to Fable, whose diet allows ONE run per iteration. No decision is open. |
| D-22 | RESOLVED | **`C1` — plain cons cells** (Mark, directive on `#745` at `2026-08-22T11:36:26Z`, body exactly `C1 `; consumed by iteration 251). LC-2…LC-5 build for LC-1's candidate (i), `{head Value, tail *cell, n int}` with a cached length — **not chunked**. `C2K32` is DECLINED. The answer CONFIRMS the decomposition's existing scope rather than re-basing it: `m-list-cons-cells-decomposition.md` was written around plain cons cells throughout, so the 15.5–21.5 person-day estimate stands unchanged and no downstream milestone is re-scoped. Note the tie-break the doc would have applied selects `C2K32` on per-element memory; Mark's ruling overrides it, and the spike's own clause (b) supports that — C1 iterates **faster** than the slice control (0.950×) where C2K32 is 1.081×, i.e. the memory win was paid for in iteration speed. | **Exercised the same iteration, 251.** LC-2 `m-list-accessor-api` was unblocked and routed: a new 793-line design doc with 28 first-party verification rows, quorum round 1 **BLOCKED** 2-of-2 external ($0.104328, `absent_reviewers` EMPTY), designer revision, round 2 **BLOCKED** 2-of-2 ($0.132845, `absent_reviewers` EMPTY), then **narrow-refinement carve-out** applied with all three surviving objections' `proposed_fix` texts taken VERBATIM. D-22 discipline held and is checkable: all four `chunk` mentions in the doc are exclusions ("Not chunked", "no chunk-boundary invariant anywhere", "Any chunk-aware design — D-22 = C1 forecloses it"). Prior evidence: Clause (c) B/element ÷ measured C0 (**16.418**): C1 **1.952**, C2K8 **1.340**, C2K32 **1.070**. Clause (b) worst-n (65536): C1 **0.950**, C2K32 **1.081** — C1 is actually *faster* to iterate. Full arithmetic and all 420 trials: `design_docs/implemented/v0_34_0/m-list-repr-spike-M6-report.md` and `tools/internal/spike-listrep/testdata/m6-matrix.json`; PR [#810](https://github.com/sunholo-data/ailang/pull/810). |
| D-23 | RESOLVED | **YES — the controller MAY `git checkout -B dev origin/dev` in the main checkout once it has MEASURED that every ahead-commit's content is present upstream and that dirty ∩ incoming is empty** (Mark, `#745` comment `2026-08-22T07:43:04Z`, verbatim: *"D-23: yes"*). This EXTENDS `D-16` from the 0-ahead case to the content-duplicated-ahead case, which is the state every PR-landing iteration ends in, and it closes the one-way divergence that had kept the RUNNING skill 7–11 commits stale across iterations 195–249. | **Exercised the same iteration, all four obligations measured, iteration 249.** (1) Content presence per ahead-commit, not by patch-id — patch-id matched **0/3** because each landed upstream inside a SQUASH that bundled other commits, so the rigorous-but-unnecessary test fails on a healthy tree; measuring the predicate directly, the two skill commits' **157** added lines are **100%** present upstream, and the record commit's log (+120) and archive (+1) likewise, with the dashboard superseded by design (overwrite-every-iteration) and the charter's 2 residual lines being the `ITERATION 246` stamp (found IN THE ARCHIVE — rotated, not lost) and a queue-row header iteration 247 legitimately retagged on landing. Controls fired both ways (a known-absent line absent, a known-present line found). (2) dirty ∩ incoming = the skill file ALONE, whose on-disk content already EQUALED `origin/dev` (`cmp` rc=0) — the documented expected-refusal case; staged origin's blob for that one path and confirmed **no byte on disk changed** (sha256 identical before and after). Control: the intersection against a known-changed set returned 3. (3) All 6 dirty files backed up outside the repo and re-verified **byte-identical** afterwards. (4) `git checkout -B dev origin/dev` rc=0, `Reset branch 'dev'`, carrying the two modified benchmark JSONs across. Result: **0 ahead / 0 behind**, three untracked eval scripts intact, and the running skill `cmp`-clean against `origin/dev`. |
| D-24 | RESOLVED | **`accept` — take 4.5 d and re-base the cons-cells programme total by +0.5 d** (Mark, attended steering session 2026-08-22). The planner surfacing the overrun with costed line items rather than compressing to fit is the behaviour to reward; all three items are real work. Original ask: **LC-2's sprint plan lands at 4.5 days against the roadmap's declared 3–4 day band for this piece.** The planner surfaced the +0.5 with three costed line items rather than compressing to fit, which is the behaviour we want — but the programme's 15.5–21.5 person-day total is denominated on those per-piece bands, so an accepted overrun on LC-2 is a (small) re-basing of the whole cons-cells estimate, and eight pieces each doing this is how a programme silently doubles. **Not blocking** — the executor can run M1+M2 next iteration either way, and this row does not gate it. The three costed items are the `x/tools` module addition being a fresh dependency rather than an indirect promotion (measured, DEFECT-4), the load-contract work DEFECT-1/-2 forced (the `packages.Config.Tests` flag moves the census denominator by ~380 sites), and the dual-config AC-13 fixtures. Options: **`accept`** — take 4.5 d and re-base the programme total by +0.5 d; **`cut`** — drop scope to fit 4 d, in which case the controller proposes cutting the second (LC-4-shaped cell) config fixture, which is the only item that buys future confidence rather than present correctness. | Raised by iteration 252 from the landed plan `design_docs/planned/m-list-accessor-api-sprint-plan.md`. |
| D-25 | RESOLVED | **A2 = semantic effect extraction via a NEW CLI SURFACE that emits a file's resolved effect row** (Mark, attended 2026-08-22) — e.g. an `ailang` subcommand/flag printing the resolved effect row (JSON), consumed by `scripts/verify_bytecode_parity.go`; compiler packages stay OUT of the harness; the stderr sniffer is retired. Feasibility spike first per the doc's own sizing note, then re-size A2 in the second design round. Unparks the A1/A2 harness lane + B1-closure-half/B3/B4 (incl. `#506` unsafe replay). Per D-12, the queue row and doc header are un-parked in this same attended edit. | Options A/B/C in `planned/v1_0_0/m-bytecode-vm-parity-bugs.md` header box; C (split) was taken 2026-08-04 and the spun-out `#505` fix routed separately — this answers the REMAINING semantic-extraction question with the shape the gpt5-6-sol reviewer's fix pointed at, choosing the CLI surface over importing compiler packages into the harness. |
| D-26 | RESOLVED | **FIRE M4b — the cost-per-verified-success baseline cohort run — next iteration** (Mark, attended 2026-08-22). The $20-total cap ratified 2026-07-27 stands; raise `MISSION_METERED_BUDGET_USD` to 20 for that single iteration if needed. Outranks the queue next iteration under D-28. | The ~2026-08-01 quota-reset gate passed three weeks ago with ZERO M4b activity since (measured: no M4b mention in the log after ~iter 117); this converts the standing approval into a scheduled action. Closes clause 5's last mile — the KPI machinery (M1–M4a incl. the BF-1 fix) is already landed and waiting for its number. |
| D-27 | RESOLVED | **`m-effect-scope-params` (effect sprint 4/4) RE-SCORED to v1.1 — it leaves the v1.0.0 bar** (Mark, attended 2026-08-22). Clause 4 now requires effect sprints 1–3 only. Doc Target updated in this same edit. | The doc's own release-gate note names it the weakest forcing function of the four carve-outs: no public doc promises scope semantics (guide lists it under "Future work"), the whole surface is Experimental on the stability page, and it was ordered last precisely to keep this option open. Cuts ~2.5 d off the bar without breaking any published promise. |
| D-28 | RESOLVED | **BAR-FIRST ordering until v1.0.0 is declarable** (Mark, attended 2026-08-22) — items serving an open bar clause (M4b per D-26 · `m-effect-clock-net-fs-modes` · `m-v1-orchestration-flagship` · `m-run-selector-enumeration-floor` sweep · the unparked A1/A2 lane per D-25 · clause-3 teaching-prompt A/B confirmation) OUTRANK the cons-cells programme until all five clauses close. LC-2 execution (M1+M2) resumes after; D-24's `accept` stands unchanged. Recorded as temporary rule 0 in the Backlog ordering policy — delete that rule when the bar closes. | Chosen over interleave and cons-cells-first in the attended session: remaining bar-gating work is roughly 8–12 sprint-days and clause 5 is a single funded run, so bar-first makes v1.0.0 declarable in ~2 weeks of loop time instead of 4–6. |
| D-29 | RESOLVED | **BOTH — publish a STRICT and an EFFECTIVE cost/verified-success, and separately add `ensures` where it makes sense** (Mark, `#745` comment `2026-08-23T08:30:56Z`, verbatim: *"D-29 - both but update prompts to use ensures for benchmarks that make sense"*; consumed by iteration 258). Option **(c)** carries the headline: the strict arm keeps today's `VerifySkipped == 0` reading (**$0.7778**, denominator **3**) and the effective arm exempts `not_applicable` (**$0.2121**, denominator **11**); neither replaces the other, so no published number is silently restated and the 2026-07-27 ratification is not overturned. **Both arms REQUIRE the `skipped`/`not_applicable` split**, so the publishing milestone lands INSIDE `m-contract-verification-coverage` and inherits its `D-30` block — `D-29` no longer blocks anything on its own. The second clause is **scoped, not blanket (b)**, and iteration 258 measured the scope first-party: of the **5** candidate functions only **`minor3` is non-recursive**, and for the other four an added `ensures` would be graded a **spurious counterexample**, i.e. STRICTLY WORSE than today's `not_applicable` — see the `m-verify-bounded-unrolling-false-counterexample` and `m-benchmark-ensures-coverage` rows. | Answered after 5 iterations open. The ruling is recorded verbatim; the controller did NOT infer a resolution from adjacent work. |
| D-30 | RESOLVED | **How must the harness&harr;`ai-check` version coupling be enforced BEFORE the `not_applicable` split lands?** Round-2 quorum on `m-contract-verification-coverage` was **blocked** by `gpt5-6-sol` (2/2 reviewers present, `gemini-3-1-pro` **pass**) on the one residual the doc had named honestly and mitigated only by convention. **Controller-measured, first-party, not forwarded (rule 3f):** `RunAICheck` defaults `ailangPath` to the bare string `"ailang"` and `exec.Command` resolves it via **PATH** (`internal/eval_harness/verify.go:47-53`); **2 of 2** live non-test call sites pass `""` (`repair.go:76`, `verify.go:123`; control `PopulateVerifyMetrics` 2, negative control 0). So the parent harness and its verifier child are **independently versioned**, and reader-before-writer *commit* order cannot buy reader-before-writer *deployment*. **The skew is live on this rig right now, not hypothetical**: PATH `ailang` is `v0.33.1-211-g626f5e54b-dirty` against a repo/parent at `v0.33.1-216-g30176187f`. Post-split, an old reader driving a new writer silently banks a **reduced** `verify_skipped` and drops the `not_applicable` count entirely &mdash; a no-silent-fallback violation (CLAUDE.md &sect;2) on the exact data path clause 5's headline KPI rests on. **(a) SCHEMA** &mdash; version `ai-check`'s JSON (`--json-schema=2`), emit `not_applicable` only for v2, and make the reader **reject** missing/unknown versions rather than bank partial counters (`gpt5-6-sol`'s primary fix; a new wire contract, and the largest option). **(b) SAME-BINARY** &mdash; bind `RunAICheck` to `os.Executable()` instead of PATH, making skew impossible by construction (`gpt5-6-sol`'s own stated alternative; ~1 line, and the idiom already has **9** in-repo precedents incl. `cmd/ailang/replay.go:153` &mdash; **but** it silently changes how *every* eval run resolves its verifier, and collides with this loop's own mandated scratch-build-and-prepend-PATH discipline). **(c) ACCEPT** &mdash; keep PATH, ship the split with the residual named plus a post-merge spot-check (what the doc proposes today; `gemini-3-1-pro` passed it, `gpt5-6-sol` rejects it as not fail-closed). **ANSWERED &mdash; (b) SAME-BINARY, with two amendments the option text does not carry.** (Mark, attended 2026-08-26) Bind `RunAICheck` to `os.Executable()` so harness and verifier are the same build by construction; same-binary makes the schema question moot, because a parent and child cut from one binary cannot disagree about the wire format. **Amendment 1: NO PATH FALLBACK.** All 9 in-repo precedents fall back to the bare string `"ailang"` when `os.Executable()` errors (`cmd/ailang/replay.go:153` is explicit: `binary = "ailang" // fallback to PATH`); on THIS data path that fallback re-creates exactly the skew being closed and is a CLAUDE.md &sect;2 violation, so it must return an error instead. Do not copy the precedent shape uncritically. **Amendment 2: TESTS NEED AN INJECTED PATH, NOT A FALLBACK.** Under `go test` `os.Executable()` resolves to the *test* binary, so any test exercising `RunAICheck` must pass an explicit path through the existing `ailangPath` parameter; adding a fallback to make tests pass would reintroduce the defect through the back door. The stated objection &mdash; that (b) collides with the loop's scratch-build-and-prepend-PATH discipline &mdash; was weighed and rejected: PATH resolution is the CAUSE of the measured skew, and under (b) a scratch-built harness runs its own scratch-built verifier, which is what a scratch build is supposed to mean. (a) is held in reserve for the day `ai-check` becomes an externally-invoked contract; it is not needed while the only callers are in-process siblings. | Controller did **not** self-rule: the narrow-refinement carve-out requires a fix that needs no controller judgment, and choosing among a new wire protocol, a repo-wide resolution change with rig-operational consequences, and accepting a P0 data-integrity residual is judgment (standing rule 2). Note the coupling is a **pre-existing** property of HEAD that the split would make consequential &mdash; it is worth a ruling whichever way `D-29` goes. |
| D-31 | RESOLVED | **Split the designer rotation into AUTHORING lanes and REVIEW lanes (or widen it)?** The rotation is `claude:claude-fable-5` &rarr; `codex:gpt-5.6-sol` &rarr; gemini, and **two of its three entries cannot serve as designer for STRUCTURAL reasons no probe can clear**: `codex:gpt-5.6-sol` IS one of the two default quorum reviewers (`gpt5-6-sol`), so routing it makes a doc's author its own judge; gemini/managed_agents is read-only under `CapRemoteSandbox` and cannot write a file at all. The usable authoring rotation therefore has ONE entry, and every doc that blocks at round 1 collides with the Fable diet by construction. Iteration 251 measured this at 3 instances and said the fix *"needs a human, because it is a routing-policy change on a shared file"*; iteration 255 amended the DIET (the unit is one bounded DOC, not one bounded RUN) but explicitly did **not** widen the rotation &mdash; and **neither filed a decision ID**, so the ask has never reached Mark. Iteration 256 is instance **4** and files it. **(a) SPLIT** &mdash; mark lanes authoring-capable vs review-capable and rotate only over the former; **(b) WIDEN** &mdash; add a fourth model that is neither a quorum reviewer nor sandbox-read-only; **(c) ACCEPT** &mdash; keep collapsing onto Fable and stop recording it as an anomaly. **ANSWERED &mdash; (a) SPLIT into authoring-capable and review-capable lanes.** (Mark, attended 2026-08-26) Rotate the designer role only over authoring-capable lanes. (b) WIDEN was considered and has no available candidate: `opus` is the controller (author-is-own-judge), `codex:gpt-5.6-sol` is a standing quorum reviewer (same objection), gemini/managed_agents is read-only under `CapRemoteSandbox`, and the pi/DeepSeek-Flash lane fails silently at rc=0 with an empty worktree. `sonnet` is the only real widening candidate and it carries a constraint (something other than `sonnet` must evaluate on any iteration where it authors), so widening is a follow-up, not this ruling. (c) ACCEPT declined: lane capability is a structural property, and recording it once as policy is strictly better than re-discovering it as a per-iteration anomaly &mdash; which it has been for 4 instances across iterations 251, 255 and 256. | Filed 2026-08-23 by iteration 256 under the decision-recording contract: a park-for-human with no ledger row can never be generated into a `DECISIONS FOR MARK` section, so it gets re-discovered every iteration and answered in none. |
| D-32 | RESOLVED | **Should an `inconclusive` verification obligation be EXEMPTED from the effective `cost_per_verified_success` arm, the way your `D-29` ruling exempts `not_applicable`?** **ANSWERED &mdash; (b) KEEP STRICT, with a reporting requirement.** (Mark, attended 2026-08-26) `inconclusive` is NOT the same axis as `not_applicable` one status along, and the `D-29` analogy does not carry: `not_applicable` means the obligation does not exist (the benchmark states no `ensures`), so it does not belong in a denominator at all; `inconclusive` means the obligation DOES exist and our own encoder could not discharge it. Exempting it would let encoder weakness inflate `cost_per_verified_success`, i.e. the KPI would improve when our tooling got worse &mdash; the opposite of what clause 5 exists to measure. So `inconclusive` stays in the denominator and out of the numerator. **Requirement attached:** it must be published as its own NAMED bucket rather than pooled silently with verification failures, so the encoder-capability signal stays visible instead of being hidden inside a rejection count. (c) DEFER declined: the ruling does not depend on the population size, and the frozen cohort already localises all 8 onto two recursive-ADT benchmarks uniformly across 4 model families. | Iteration 259 measured the class first-party: **8 of 30** frozen `v1.0` runs are graded verification-FAILED on a counterexample, and **all 8** land on the two recursive-ADT benchmarks (`contract_sorted_merge` &times;4, `contract_bst_validate` &times;4), uniform across 4 model families &mdash; the signature of a TOOL defect, not a model one. `m-verify-bounded-unrolling-false-counterexample` reclassifies those from `counterexample` to a new `inconclusive` status, which is honest. **But the published number cannot move either way without this ruling**: that doc's `isVerifiedSuccess` gains `&& VerifyInconclusive == 0`, so a reclassified run stays rejected, numerator and denominator are unchanged, and `$0.7778187072` is bit-identical before and after (measured; `verified_successes` 3, `known_cost_usd` `$2.3334561216`). So this is the same axis you already ruled on, one status along. **(a) EXEMPT** &mdash; `inconclusive` joins `not_applicable` in the effective arm (the strict arm is untouched, per your `D-29` both-arms ruling); **(b) KEEP STRICT** &mdash; an obligation the tool could not discharge counts against the model in both arms, i.e. the fix buys honesty and repair-loop correctness but no KPI change, which is the status quo the doc ships under; **(c) DEFER** until the `inconclusive` population is observed on a fresh cohort rather than inferred from the frozen one. The doc deliberately does not decide this and does not depend on it. |
| D-33 | RESOLVED | **Cross-mission blockers are PRIORITIZED in the queue ordering** (Mark, attended 2026-08-23; recorded as a charter stamp because the session authenticated as the bot account, which `mission_directives.sh`'s self-direction guard rightly refuses as a directive principal). The `cross-mission` label was created and applied to the 8 open sibling asks (`#764`, `#757`, `#756`, `#755`, `#715`, `#713`, `#712`, `#498`), and `#764` went to the queue head as P1. | **ID-COLLISION REPAIR, iteration 260.** The attended ruling was recorded in the charter's RATIFIED block under the label `D-31`, which was ALREADY IN USE by an OPEN ledger row (designer-rotation split, filed by iteration 256). Measured: the ledger row's own commit precedes the attended block &mdash; iteration 259's record landed `2fde160db` at `15:53:16+02:00`, the attended block `4e7c32ce0` at `18:08:23+02:00`, and `4e7c32ce0` touched **zero** ledger rows. Per the decision-recording contract (append-only, no ID reuse, ledger is authoritative), the pre-existing OPEN `D-31` keeps its ID and the attended ruling is re-filed here as `D-33`. No ruling is altered and no question is re-asked; only the label moves. |
| D-34 | RESOLVED | **Standing release decision: when `#764` lands, cut v0.34.0** (Mark, attended 2026-08-23). World consumes upstream via **pinned releases only**, so merging the fix does not reach World &mdash; the tag is the delivery. Surface it in the report's DECISIONS row once the fix is green on `dev`. Releases remain Mark's sole decision: this ruling pre-authorizes the ASK, not the tag. | **ID-COLLISION REPAIR, iteration 260**, same mechanism as `D-33`: the attended block labelled this `D-32`, which was already an OPEN ledger row (the `inconclusive` KPI-exemption question, filed by iteration 259 in the very commit that preceded the attended session). Re-filed as `D-34`; the OPEN `D-32` keeps its ID and is re-asked unchanged this iteration. Note the consequence this repair prevents: generating `DECISIONS FOR MARK` from OPEN rows would have re-asked `D-31` and `D-32` hours after Mark ruled on two *different* questions bearing those labels, which reads as the loop ignoring him.  **DISCHARGED by Mark 2026-08-24T23:35:48Z on `#852` ("D-34 is discharged"), actioned iteration 272.** The standing ask is retired because the delivery it authorised had already happened under a different tag: re-measured first-party at iteration 272, `serveapi/protocol` ships at `v0.33.2` (5 files; negative control on a non-existent sibling path 0), both named milestone commits `ba2eeb4b4`/`7e7bdffcb` are ancestors of the tag while a post-tag commit correctly is not, and the tag was published `2026-08-24T19:26:28Z` non-draft. No `v0.34.0` is owed for `#764`. Consequence actioned in the same iteration: `#764`'s only stated reason for remaining open (a release-pinning consumer could not yet import the surface) had lapsed on its own evidence, so it was closed with the verdict and the pin instruction (comment count asserted 5 &rarr; 6 before closing). This row must no longer generate a DECISIONS ask. |
| D-35 | RESOLVED | **What module boundary should `serveapi/protocol` have &mdash; (a) a plain package owned by the main module, (b) a nested Go module, or (c) a separate repository?** This is `#764`'s design-freeze gate and it is **BLOCKING**: implementation does not begin until it is answered. Raised by `gpt5-6-sol` at round 2 of the `m-serveapi-protocol-only-module` quorum and applied VERBATIM. The doc had proceeded under (a) while asserting that a later conversion to (b) is additive because import paths are stable; **only the import path is stable.** Moving a directory into a nested module changes module ownership and version resolution, can require new `require` directives and coordinated tags, and can make a consumer that requires only `github.com/sunholo-data/ailang` **stop resolving the package**. The claim was never measured, so it is WITHDRAWN rather than repaired, and the decision it was suppressing is now surfaced. **(a) PLAIN PACKAGE** &mdash; ships with **no promise of transparent later conversion**; consumer requires the main module, inherits its `go 1.26.6` floor, and allowlists the narrow prefix `github.com/sunholo-data/ailang/serveapi/protocol` (World's matcher is prefix-based &mdash; measured first-party at `ailang-world@48ef275`, `d == m \|\| strings.HasPrefix(d, m+"/")`). **(b) NESTED MODULE** &mdash; cleanest possible seam (empty consumer dependency graph), but its tag scheme, main-module `require`, release workflow and CI matrix must be specified NOW, plus the reviewer's demanded verification row: a temporary external consumer running `go mod tidy` / `go list -deps` / `go test` before and after the conversion, recording required directives, selected versions, downloaded module roots, and whether resolution succeeds without a `replace`. **(c) SEPARATE REPO** &mdash; maximum isolation, maximum ops cost, drift risk; not recommended. | **CONTROLLER RECOMMENDATION: (a)**, and the reason is your own `D-34`: World consumes via pinned releases and the delivery is a v0.34.0 tag, which is exactly what a main-module package gives you. But `D-34` ruled on *release delivery*, not on *module boundary*, and the decision-recording contract forbids inferring a resolution from adjacent work &mdash; so it is recorded as a pointer, not as an answer. **This is the only thing standing between `#764` and a sprint plan**; every other reviewer objection across four reviews is answered, and no reviewer has ever disputed the design direction. One word unblocks it. **ANSWERED &mdash; (a) PLAIN PACKAGE.** Mark, directive `D-35 A` on bookkeeping issue `#745` at `2026-08-23T19:01:24Z` (author `MarkEdmondson1234`; read first-party by `scripts/mission_directives.sh --issue 745 --since 2026-08-23T08:30:56Z`, which reported **1** directive of 73 comments &mdash; the allowlist instrument fires). Applied at iteration 261 in the SAME iteration it was read, before the watermark moved, per the decision-recording contract. `serveapi/protocol` ships as a plain package owned by the main module, **with no promise of transparent later conversion**; the withdrawn (a)&rarr;(b) migration claim stays withdrawn and a future move to (b) is a breaking-change project with its own doc and the reviewer's demanded external-consumer verification row. Design doc frozen the same iteration: 9 targeted substitutions, each asserted to match exactly once, all four Design-Freeze boxes now checked (control: `RESOLVED 2026-08-23` present 4&times;, fresh negative control fired). |
| D-36 | RESOLVED | **When an evaluator FAILS all three permitted rounds but every remaining finding is a small, mechanical fix, should the item PARK or LAND?** The skill says round-3 fail &rarr; `needs-human-review`, park, message controlplane &mdash; a rule whose purpose is to stop an unbounded fix/judge loop and get a human's eyes on work that is not converging. **(a) LAND-AND-FLAG** &mdash; ratify what iteration 275 did and make it the rule: a round-3 FAIL whose remaining findings are mechanical may be fixed and landed, provided the FAIL and the missing review are surfaced as a decision. **(b) PARK, STRICTLY** &mdash; three rounds means three; a fourth pass is by definition unreviewed and waits for you, whatever the findings look like. **(c) RAISE THE BUDGET** &mdash; allow N rounds while findings keep shrinking, and park only when a round finds something the previous round did not anticipate in kind. Note the loop cannot choose this for itself: the rule governs when the loop is allowed to stop asking. **ANSWERED &mdash; (c) RAISE THE BUDGET, with an explicit convergence criterion and a hard cap.** (Mark, attended 2026-08-26) Continue past round 3 while each round's findings keep SHRINKING IN KIND; park `needs-human-review` the moment a round raises something the previous round did not anticipate in kind (iteration 275's round-2 step-level `if:` finding is the worked example of a park trigger &mdash; it was worse in kind, not merely one more of the same). Hard cap **5** rounds regardless. (a) LAND-AND-FLAG was declined for a structural reason rather than a judgment about iteration 275: *"the remaining findings are mechanical"* is assessed by the controller, who is also the party that wants to land, so ratifying it installs a self-judged escape hatch that widens under pressure. (c) puts the criterion in the JUDGE's output instead. (b) PARK STRICTLY declined: on the measured evidence parking iteration 275 would have preserved three live defects to avoid one hypothetical. **Precondition, ruled the same session:** this rule is inert while no evaluator can be spawned at all &mdash; see `D-39`. | Iteration 275 hit the letter of the rule and, on the evidence, not its spirit. `sonnet` scored **63 / 45 / 38**, all FAIL, and **all six BLOCKING findings were real and reproduced first-party** &mdash; the reviewer was working perfectly. The scores fell because each round was asked to attack harder and the rubric weights moved, not because the work degraded: round 1's blockers were bypass idioms (`&#124;&#124; exit 0`, `set +e`, a background `&`), round 2's were four more plus one worse in kind (a step-level `if:` that stops the gate running at all), and round 3's were a permissive character class inside the round-3 redesign (`make X FOO=a&#124;&#124;true` parsed as canonical &mdash; `bash -eo pipefail` confirms rc=0) and an exemption the judge **proved unnecessary by measurement**. The last two were closed by the controller using the evaluator's own reproduction commands as controls, with **no independent fourth review** &mdash; that is the genuinely unreviewed part and the reason this is a decision rather than a note. It LANDED (`fde5ea067`, PR #878, 21 checks green, 4/4 required) because it is strictly stronger than HEAD on every measured axis: **20** distinct suppression shapes caught against **zero** coverage before, two live stdlib interface drifts fixed, and `make ci`'s red surfaced and attributed. Parking would have preserved three current defects to avoid a hypothetical one. |
| D-37 | RESOLVED | **May a function declared `!{AI[mode=routeable]}` call `std/ai.call`, which declares bare `!{AI}` and therefore requires `mode=fixed`?** This is not a new question and not a fresh defect &mdash; it was measured and parked as *"Q1"* / *"(0-subsum-ai)"* by iterations 110/111 on 2026-07-28 (`63b0ba3dd`'s own commit message: *"No AI edge was registered even though examples/ai_modes.ail is red at HEAD with the identical defect; that is PARKED for Mark as Q1"*), **and it never became a ledger row**, so it has never actually reached you. It is live today: it is the sole reason `make ci` is RED at HEAD. **(a) REGISTER THE EDGE** &mdash; one line in `internal/types/effect_subsumption.go`'s `subsumptionEdges` (`{Effect:"AI", Declared:"routeable", Required:"fixed"}`), which is exactly the shape of the two existing `Rand` edges (`seeded`&rarr;`os`, `crypto`&rarr;`os`: a declared mode covering the effect's *registered default*, and `fixed` IS AI's registered default). The open question is whether the analogy holds in *direction*: `seeded` is strictly more constrained than `os`, whereas `routeable` is arguably more **permissive** than `fixed`, so the edge would let a routing-capable function satisfy a requirement that promised a direct provider call. **(b) MAKE `std/ai.call` MODE-POLYMORPHIC** &mdash; the principled fix: a stdlib primitive should not pin a caller's mode at all. Larger, touches stdlib effect signatures and the elaborator, and would generalise to `Clock`/`Net`/`FS`. **(c) MIGRATE THE EXAMPLE** &mdash; rewrite `examples/ai_modes.ail` so nothing declares `routeable`. Cheapest, and it makes the feature undemonstrable: `mode=routeable` would then have **zero** in-repo users (measured: 1 file today, and it is this one) and the shipped design doc's Example 2 becomes unimplementable. **(d) QUARANTINE** &mdash; move the example out of the gated set and document `mode=routeable` as accepted-but-unusable, which greens `make ci` while preserving the evidence. The loop cannot choose: (a) and (b) are language-semantics rulings, and standing rule 2 forbids forcing one. **ANSWERED &mdash; (b) MAKE `std/ai.call` MODE-POLYMORPHIC.** (Mark, attended 2026-08-26) The principled fix, chosen over the (a) stopgap. Grounding measured this session: **every** stdlib AI primitive declares bare `!{AI}` (`std/ai.ail` &mdash; `call:443`, `callJson:448`, `callJsonSimple:454`, `callImage:463`, `callImageBase64:470`, `callResult:175`), which desugars to `mode=fixed`, so **no** non-default AI mode can reach any AI primitive. `mode=routeable` is unusable BY CONSTRUCTION, not merely unused in this repo &mdash; and `replay-only`, already legal in `effectSchema`, hits the identical wall the first time anyone writes it. That is what makes (a) a patch-per-case in the sense of CLAUDE.md &sect;3: it would need one edge per non-default mode today and one per (mode &times; effect) once Phase 5 ports Clock/Net/FS, whose schema rows are already stubbed in `internal/types/effects.go`. **Consequence accepted deliberately:** (a) was the stopgap that would have greened the gate today, so declining it means `verify-examples-toplevel` stays RED and `make ci` stays red until (b) lands. That red is a REGISTERED exemption, not a new one &mdash; `internal/cihygiene/gate_wiring_test.go:43` carries it with the standing instruction *"wire it when it is green, do not widen the exemption"*, which independently rules out (d) QUARANTINE. **Scope correction to this row's premise:** GitHub CI is GREEN on `dev` and is unaffected; `tools/verify_examples.sh:38` skips `ai_modes` (*"needs AI capability + live API key (network)"*) so the CI `verify-examples` gate never runs it. The redness is local `make ci` / `verify-examples-toplevel` only. Sequence (b) before the Phase-5 Clock/Net/FS port, since after that port every stdlib primitive carries the same pin. | **Measured first-party this iteration, 2026-08-25, every arm with its exit code captured without a pipe and a firing negative control.** The example was **GREEN when shipped**: at `01642550e` (2026-05-04, the commit that shipped it as M3 of M-AI-EFFECT-MODES) a binary built at that commit prints `✓ No errors found!` (rc=0), while the same binary refuses a deliberately ill-typed file with rc=1. It is **RED at HEAD** (`f4828cc89`, correctly stamped binary, `git describe` == `--version`): *"Effect mode mismatch: AI requires mode=fixed; declaration provides mode=routeable"* for `summarize_routeable`, and it is the **sole** failure of **42** type-checked examples, so `verify-examples-toplevel` is rc=1 and `make ci` &mdash; **27** prerequisites, enumerated from `make -pn` rather than the file bytes &mdash; cannot pass. **Two attributions in the queue row and the original park are REFUTED.** (i) It is not stale sugar: `design_docs/implemented/v0_15_x/m-ai-effect-modes.md` Example 2 is verbatim `export func summarize_routed(text: string) -> string ! {AI[mode=routeable]} = call(text)`, i.e. the example matches the shipped doc it exists to demonstrate. (ii) It was **not** caused by the `m-effect-replay-subsumption` sprint it was parked under: at `7fb69c50e` (pre-M1) the file is already rc=1, with a *worse* diagnostic (`Missing effects:` empty) &mdash; M2 only improved the message, exactly as its commit claimed. Bracket established: GREEN `01642550e` (2026-05-04) … RED `1282767ca` (2026-07-22, re-measured cleanly), and RED at every point after. Narrowing inside that bracket is a separate queue row; an automated bisect run this iteration returned a **docs-only** commit and was discarded as an instrument failure (its GOOD seed was poisoned by a `$?` read that a `$(git …)` substitution in the same argument list had already clobbered &mdash; reproduced two-armed: buggy form prints `rc=0` for a command that exited 1). |
| D-38 | RESOLVED | **Should the 341 non-canonical `.ail` files be reformatted to `ailang fmt`'s output, or is the formatter's canonical form itself wrong?** A per-file scan over `examples/` + `std/` at HEAD measures **ok=63, drift=341, err=46** &mdash; only **14%** of this repo's AILANG corpus is in the form `ailang fmt` emits. That is not a drift backlog to work through, it is a disagreement about what canonical AILANG *is*, and it is load-bearing well beyond this one gate: `ailang fmt` is taught to eval models, and a formatter that calls 76% of the repo's own examples non-canonical has cost real tokens before. It is also asked while the formatter has a live soundness defect (`std/cognition.ail`, below). **(a) REFORMAT** &mdash; `ailang fmt --write` across the corpus and freeze; a very large diff, and it trusts the emitter. **(b) FIX THE EMITTER FIRST** &mdash; treat 341/450 as evidence the formatter is wrong rather than the corpus, converge it toward the corpus, then reformat the residue. **(c) SCOPE THE GATE** &mdash; declare only `std/` gate-eligible (29 of 46 drift) and leave `examples/` advisory. **(d) NEVER WIRE** &mdash; keep it opt-in permanently and drop the exemption bookkeeping. The loop cannot choose: (a) and (b) are rulings about the language's canonical form, which is standing rule 2 territory. | Measured iteration 277 with a freshly built `v0.33.2-17-g1d7040de4`: per-file scan of 450 files (roots asserted with `test -d`); the gate's own scan reports only 2 because `ailang fmt --check` aborts at its first error. **RESOLVED 2026-08-26 by MarkEdmondson1234** on [#852](https://github.com/sunholo-data/ailang/issues/852) @ `2026-08-26T04:49:43Z`: *"D38 - those examples were done pre formatter so let's update them to respect the new formatter"* &mdash; i.e. option **(a) REFORMAT**, on the stated ground that the corpus predates the formatter, which explicitly rejects (b) (the emitter is not what is wrong). **EXECUTED iteration 282**: `ailang fmt --write` over the 342 drift files under a freshly built ldflags-stamped `v0.33.2-26-gfadbdc4e2`; corpus canonical **63 &rarr; 405** of 450, drift **342 &rarr; 0**, attach-refusals **38** and parse-failures **7** both UNCHANGED (untouched by design &mdash; they fail closed). Comment preservation asserted with the repo's own `lexer.CollectComments`: total **7865 &rarr; 7865**, per-file delta **0 of 450** joined pairs, instrument proven by a poisoned arm that FIRED. `ailang check` rc unchanged on **342 of 342** joined pairs (279 rc=0 / 63 rc=1 both before and after). Gates two-armed against a pristine base: `verify-examples`, `verify-stdlib`, `test-stdlib-ail`, `go test ./internal/format/...` all rc=0 in BOTH arms. **Residual consequence, flagged not buried:** the printer has no line-width limit, so collapsing multi-line bodies raised max line length **267 &rarr; 1315** chars and lines >120 chars **57 &rarr; 147** across the reformatted set &mdash; filed as `m-fmt-printer-no-line-width-limit`, and it is a NEW question for Mark (`D-39`), not a re-litigation of this one. | **PROVENANCE NOTE, added by the attended session 2026-08-26 &mdash; the cited comment is NO LONGER RETRIEVABLE, and this is expected, not a discrepancy to re-investigate.** A first-party audit of `#852` at 2026-08-26T05:3x`Z` enumerated **22** comments, **all** authored by `sunholo-voight-kampff`, latest `2026-08-26T03:29:47Z`; the `timeline` endpoint agreed, and a repo-wide `since=2026-08-26T04:00:00Z` sweep returned **zero** issue comments (control: the same sweep at `since=03:00:00Z` DOES return the 03:29:47Z comment, so the instrument fires and the empty result is a measurement). **Mark confirmed in the attended session that he posted the directive and then deleted it** &mdash; deleted comments vanish from both endpoints, which fits the evidence exactly. So iteration 282 read a REAL directive; the quote and the ruling stand. Recorded here because the ledger cites a URL that now resolves to nothing, and ghost discipline would otherwise re-open this every time the row is re-read.
| D-39 | RESOLVED | **`ailang fmt` has no line-width limit &mdash; should it?** Executing `D-38`(a) made this visible and measurable rather than theoretical: the printer collapses a multi-line equation body onto ONE line regardless of length, so across the 342 reformatted files max line length went **267 &rarr; 1315** characters and lines >120 chars went **57 &rarr; 147**. This is not a defect of the reformat &mdash; the reformat is exactly what `D-38`(a) asked for &mdash; it is a property of the emitter that the corpus was previously hiding. It is load-bearing for the same reason `D-38` was: `examples/` is the teaching corpus shown to eval models and rendered on the website, and a 1,315-character line is bad for both. **(a) ADD A WIDTH LIMIT** &mdash; the printer wraps at N columns (gofmt-style); a second large corpus diff, and it is a change to canonical AILANG, so it is standing-rule-2 territory exactly as `D-38` was. **(b) ACCEPT IT** &mdash; one-line bodies are canonical, and long lines are the price; then say so in `ailang prompt` so eval models are not taught to wrap. **(c) LIMIT ONLY THE `let ... in` CHAIN COLLAPSE**, which is where the worst cases come from, and leave short bodies inline. The loop cannot choose: this is a ruling about the language's canonical form. **ANSWERED &mdash; YES, the printer needs a line-width limit, and the corpus is reformatted a SECOND time once it lands.** (Mark, attended 2026-08-26) **The scope of `D-38`(a) is ruled explicitly, because this row's own framing would otherwise over-read it:** (a) ratifies the DIALECT and the direction of travel (the corpus predates the formatter, so corpus-vs-formatter disagreement is not evidence against the emitter) &mdash; it does **NOT** ratify the LINE LAYOUT. So a queued printer-form change is *not* a proposal against an affirmed form where line breaking is concerned, and `m-fmt-typedecl-printer-needs-multiline-emit` is UNBLOCKED by this ruling rather than re-gated. **Measured at the attended session, worst offender first-party:** `examples/runnable/list_extremes.ail` went from a multi-line `let &hellip; in` chain with max line **139** chars to the ENTIRE `main()` body on ONE line of **1315** chars. Corpus-wide max **267 &rarr; 1315**, lines >120 chars **57 &rarr; 147**. **What the same session measured in the emitter's FAVOUR, so this is scoped to layout and nothing wider:** the July dialect defect has NOT recurred &mdash; `TestFmtOutputMatchesTaughtDialect` is still present in `internal/format/format_test.go` and `go test ./internal/format/...` is rc=0 in both of `#893`'s arms; comments were preserved 7865 &rarr; 7865 with a poisoned control that fired; `ailang check` rc unchanged on 342/342. The formatter is correct on dialect, comments and types, and wrong only on width. **One over-stated risk withdrawn:** the attended brief argued reformatting `examples/` would move the teaching corpus. Measured: `prompts/` is hand-written versioned markdown and is NOT generated from `examples/`, so `ailang prompt` does not move with the corpus. The cost of the interim window is website rendering and human/agent readability, not a direct eval-teaching regression &mdash; which is why merging `#893` first and reformatting again after the width fix was preferred over holding it. **Sequencing:** width limit at the queue head; do NOT wire or freeze the `fmt` gate until it lands and the second `fmt --write` pass has run, or the collapsed form gets frozen as canonical by the gate rather than by a ruling. | Measured iteration 282 under `v0.33.2-26-gfadbdc4e2` over the 342 files reformatted for `D-38`; longest offender `examples/runnable/list_extremes.ail` at 1315 chars. Filed as `m-fmt-printer-no-line-width-limit`. **SCOPE WIDENED in the same iteration, deliberately, to keep this ONE ask rather than two:** `D-38`(a) RATIFIES `ailang fmt`'s current output as canonical, which re-frames every queued printer-form change as a proposal against a form Mark has just affirmed &mdash; not as a bugfix. The live instance is the line width above. The queued instance is `m-fmt-typedecl-printer-needs-multiline-emit`, which needs a multi-line form for type-decl bodies and `tests [...]` lists so an attached comment has somewhere to go (without it, registering those lists silently DELETES comments &mdash; iteration 281 measured `std/dom.ail` 54 &rarr; 50). Both ask the same thing: **which changes to the ratified canonical form are authorized?** Answering width alone leaves the type-decl row blocked. |
| D-40 | RESOLVED | **The unattended loop had NO INDEPENDENT JUDGE for 4 consecutive iterations, and the cause is a session-level instruction the loop cannot see. FIX THE DRIVER PROMPT** (Mark, attended 2026-08-26). Iterations 278&ndash;281 each recorded the same deviation: designer/planner/executor/**evaluator** not spawned, because the session's operating instructions forbid the Agent tool unless the user requests it. Iteration 281 states the consequence plainly &mdash; *"the formatter change would have shipped on my own verdict alone; the repo's own corpus gate, not a judge, is what caught the first half."* **Measured first-party this session:** the instruction is NOT in any local configuration &mdash; `~/.claude/settings.json`, `~/.claude/remote-settings.json`, `~/.claude/policy-limits.json` and the repo's `.claude/` all come back with zero matches for it (negative control: the same grep over `tools/launchd/` finds the driver's own text, so the instrument fires). It is harness-injected and present in every session on this rig, including attended ones. **So no settings edit can fix it, and the loop could never have fixed it for itself: the deviation it kept recording was invisible in every file it could read.** **The fix is the escape clause.** The instruction reads *"unless the user requested it"*, and under `claude -p` the driver's `$PROMPT` **is** the user turn &mdash; while `tools/launchd/mission-control.sh` never mentioned the Agent tool at all. An explicit authorization sentence in that prompt satisfies the condition by construction, in a file we control, delivered through the existing driver pin (`~/.ailang-driver-pin/<mission>`), with no dependency on harness or settings behaviour. **Verification obligation:** the next unattended fire must record designer/planner/executor/evaluator as actually spawned. If the routing block still reads `NOT spawned`, the escape-clause theory is REFUTED and this row re-opens &mdash; do not report the class as closed on the basis of the edit landing. **Ports owed:** `ailang-motoko` is a second clone of the same repo and takes the fix by sync; `ailang-world`'s driver is a hand-synced fork (different repo) and takes it only by port. | Surfaced 2026-08-26 in an attended steering session while triaging the open decision ledger; it was not itself a ledger row, which is why 4 iterations recorded it as a deviation and none escalated it. Note the interaction that makes it worth a row rather than a commit message: **`D-31` and `D-36` are both questions about the judging apparatus**, and both were being asked while the apparatus was switched off &mdash; `D-36`'s round-budget rule is inert if no evaluator can be spawned at all. Same class as the `#558` driver-drift family: a defect in how the loop is INVOKED cannot be diagnosed from inside the loop, and the deviation line is the only evidence that reaches a human. |
| D-41 | RESOLVED | **May an ACTIVE prompt version be edited in place, or must a fix bump the version?** `1a3104a49` fixed a genuine defect in `prompts/v0.16.6.md` (a taught import that cannot parse) by editing the file directly. It did not update `prompts/versions.json`, so the integrity check fired and **dev went red on the REQUIRED `test` context**, blocking every PR in the repo until iteration 283 regenerated the hash in `#899` (`26623ca4a`). The mechanical fix is settled; the policy question it exposes is not, and it bears on eval reproducibility, which is a mission KPI. `versions.json`'s own note says *"v0.16.5 remains served byte-identical for pinned eval baselines"*, and `1a3104a49`'s message says frozen archives *"are a record of what was taught, not a live surface"* &mdash; i.e. the author treated the ACTIVE version as mutable and the frozen ones as immutable, which is coherent but nowhere stated as policy. The consequence: any eval baseline pinned to `v0.16.6` now refers to different bytes than when it was banked, with no version change to signal it. **(a) ACTIVE IS MUTABLE** &mdash; ratify current behaviour; the hash in `versions.json` is a change-detector to be regenerated, and pinned baselines must pin a FROZEN version, never the active one. **(b) IMMUTABLE ALWAYS** &mdash; any content change bumps to `v0.16.7`; the hash check becomes a hard invariant and this class of red cannot recur. **(c) MUTABLE UNTIL FIRST BANKED USE**, then frozen. The loop cannot choose: this is a ruling about what a pinned eval baseline means. **ANSWERED &mdash; (c) MUTABLE UNTIL FIRST BANKED USE, then frozen.** (Mark, `#852` comment `2026-08-27T07:26:21Z`, verbatim: *"D41 - C"*; consumed by iteration 291.) So a prompt version has TWO lifecycle states, and `versions.json` records neither today: while a version has never been used in a banked eval run its file is editable in place and its `hash` is a change-detector to be regenerated; from its first banked use onward the bytes are frozen and the hash becomes a HARD invariant, so a content change must bump the version. This ratifies neither (a) nor (b): (a) would leave a pinned baseline silently referring to different bytes, which is the defect this row was opened for, and (b) would forbid the ordinary editing of a version nobody has measured against yet. **The ruling is recorded verbatim; the controller did NOT infer scope beyond the letter "C".** **What the ruling does NOT settle, and is therefore the implementing sprint's job to design rather than assume:** what exactly counts as a "banked use" and who performs the freeze transition. **The controller's first draft of this row asserted that banked results live in a per-machine store and are therefore invisible to CI; that was REFUTED by the controller's own measurement minutes later and is corrected here rather than shipped**: `.gitignore:91-93` ignores `eval_results/` but NEGATES `!eval_results/baselines/` and `!eval_results/performance_tables/`, so **17,343** baseline JSON files are repo-TRACKED (`git ls-files`, positive control `bin/ailang` ignored at `.gitignore:46` firing), and **7,821 of them carry a `prompt_version`** (0 unparseable) &mdash; so first-banked-use IS derivable in CI from the repo alone. Two measured caveats the design must face: **9,522** tracked baselines carry NO `prompt_version` field at all, and the observatory's `eval_baselines` table has no `prompt_version` column (`PRAGMA table_info` &mdash; 7 columns, none of them the version), so the tracked corpus is the ONLY attributable bank. | Surfaced 2026-08-26 by iteration 283 while clearing an inherited dev red. Measured: `1a3104a49` changed 2 files and `prompts/versions.json` was **0** of them; recorded hash `875b9cb2…` vs actual `43d4ebd6…`; the embedded mirror `cmd/ailang/prompts/v0.16.6.md` was already consistent with the new content, so only the manifest was stale. The two failing tests are `TestAILANGPromptLoading` and `TestPromptDisambiguation`, both rc=1 on a pristine `origin/dev` worktree and rc=0 after the one-line fix. **Answered after 1 day open (surfaced 08-26 iteration 283, answered 08-27).** Iteration 291 measured the implementation surface first-party before routing: `prompts/versions.json` carries **59** versions and **0** of them have any `frozen`/`banked` field (`jq` over the whole map), so the state the ruling turns on is not representable today; hash verification is LOAD-TIME only at `internal/eval_harness/prompt_loader.go:77-86` and is a pure change-detector that fires identically whether or not a version has ever been banked; banked results DO record the version (`prompt_version` field, `cmd/ailang/eval_suite_manifest.go:77`), so "first banked use" is observable in principle. Routed to a new design doc rather than implemented directly, because the repo-visibility question above is a design decision, not a transcription of the ruling. **The ruling's first consequence, measured: the ACTIVE version `v0.16.6` has ZERO banked uses** &mdash; no `v0.16.x` version appears in any tracked baseline (20 distinct `prompt_version` values, top `v0.3.21` at 210; control: the enumeration returns non-empty and 0 files failed to parse). So under (c) `v0.16.6` is still MUTABLE, the in-place edit `1a3104a49` that opened this row was LEGITIMATE, and its only defect was the stale hash. The freeze mechanism is therefore prospective, not retroactive-corrective. |
| D-42 | RESOLVED | **ANSWERED &mdash; (a) YES, standing, with the disjointness predicates as hard guards** (Mark, attended 2026-08-28, interactive steering session; verbatim: *"yes those are correct for d42"*, endorsing the brief's recommendation. Recorded as a charter stamp because the session authenticates as the bot account, which `mission_directives.sh`'s self-direction guard refuses as a directive principal &mdash; D-33 precedent.) Guardrails as endorsed: reconcile unattended ONLY when **0 ahead** (any unpushed local commit &rarr; FLAG and skip; an unattended rebase is never authorized); dirty tracked files permitted only when **disjoint from incoming** (`comm -12` empty, with a firing control); use the protective form (`git merge --ff-only origin/dev`, or `git checkout -B dev origin/dev` which refuses rather than clobbers); record the measurement in the evidence row each time. This resolves motoko's `D-MOTOKO-WORKDIR-2` by the same ruling. First application, same session: 23-behind/1-ahead reconciled attended by measured-disjoint merge `1c7073f02` and pushed &mdash; the attended case (non-zero ahead) is exactly the case the standing grant does NOT cover unattended. **Standing authorization to reconcile this checkout to `origin/dev` unattended?** The cost of the drift is now **MEASURED, not predicted**. The sibling motoko mission landed the driver-pin fix `ff0da7445` (#923) at 2026-08-26T20:42Z; V1's next fire at 00:58Z still logged `DRIVER PIN FAILED` and ran the working tree, because `ff0da7445` is **not an ancestor of local HEAD** (`git merge-base --is-ancestor` &rarr; false) and the driver executes from THIS clone. So a fleet-wide fix cannot reach a mission whose clone never advances, and the failure is silent: the pin failure path reports staleness only when it fires, and the growth sits on the success path. This iteration the four Gate-1 obligations all held: **0 ahead**; the only dirty files (`docs/static/benchmarks/os/{history,latest}.json`) are **not touched by any incoming commit** (`comm -12` empty, control non-empty at 10+ files); `git checkout -B dev origin/dev` is protective and refuses rather than clobbering; none of Principle 0's four forbidden operations is involved. I did NOT reconcile &mdash; per the skill, standing authorization is a human decision and Mark authorized only the one-time 2026-08-03 reconcile. `mission-motoko` has now asked the same question **four times** (`D-MOTOKO-WORKDIR-2`). **(a) YES, standing** &mdash; reconcile unattended whenever those predicates hold, recording the measurement each time. **(b) NO** &mdash; keep asking per-iteration, and accept that every mission routes around a stale clone indefinitely. **(c) YES, but only when 0-ahead AND 0-dirty** (today's case is 0-ahead / 2-dirty-but-disjoint). | iteration 288, 2026-08-27 |
| D-43 | RESOLVED | **ANSWERED &mdash; (c) DEPRECATE `charAt` IN THE TEACHING PROMPT NOW; (a) rides behind it at the next prompt-version boundary as the endorsed follow-on** (Mark, attended 2026-08-28, interactive steering session; verbatim: *"c for charAt, your recommendation"* &mdash; the recommendation being endorsed was stated as "(c) now, (a) at the boundary". The ruling's letter is (c); the (a) component is endorsed-in-principle and must be RE-CONFIRMED with Mark when a prompt-version boundary is actually cut, before shipping the breaking totality change &mdash; do not auto-execute it.) Immediate work for the queue: the teaching prompt steers agents to `charAtOpt`/`charAt_or` and marks `charAt` deprecated. Under D-41(c) the active `v0.16.6` is still MUTABLE (zero banked uses), so the deprecation note can land in place without a version bump &mdash; verify with `ailang check`-backed examples per prompt discipline. Recorded as a charter stamp per the D-33 precedent. **Should `std/string.charAt` itself become total (`-> Option[string]`), or does the new `charAtOpt`/`charAt_or` pair close it?** `4d8705699` (a concurrent attended session) resolved the external report the NON-breaking way: it added `charAtOpt -> Option[string]` and `charAt_or`, left `charAt` aborting, and documented the abort. That commit explicitly defers the totality question to "a prompt-version boundary" without filing it in this ledger, so iteration 294 files it here rather than let it live only in a commit message. The reporter's second-order argument is the one that matters and is unaddressed by the additive fix: because an abort exits non-zero, negative tests that assert rejection BY a non-zero exit stay GREEN while the program crashes on every input &mdash; so a panicking accessor makes exit-code-based suites vacuous, and a caller who never reaches for `charAtOpt` is still exposed. **(a) Make `charAt` total at the next prompt-version boundary** (breaking; matches every sibling accessor `list.head`/`list.nth`/`stringToInt`/`json.asString`/`bytes.slice`). **(b) Keep the additive pair and close this** &mdash; `charAt`'s abort is documented and intentional. **(c) Deprecate `charAt` in the teaching prompt** so agents are steered to `charAtOpt` without a breaking change. | Measured at `4d8705699` by iteration 294: `std/string.ail:160` is still `export pure func charAt(s: string, i: int) -> string = _str_charAt(s, i)` with the comment `ABORTS out of bounds`; `:167` is the new `charAtOpt -> Option[string]`. Original repro at `v0.34.0-114-gcaea1f9e1` (iteration 293, ghost-disciplined): `charAt("", 0)` &rarr; `Error: execution failed: _str_charAt: index 0 out of bounds`, rc=1, against an in-bounds positive control at rc=0. Sibling accessors confirmed total in `std/list`, `std/string`, `std/json`, `std/bytes`. The deferral is stated in `4d8705699`'s own commit body ("Whether charAt itself should become total is D1, deferred to a prompt-version boundary") &mdash; that `D1` is the authoring session's local numbering, NOT this ledger's, so no ID is reused. |
| D-44 | RESOLVED | **ANSWERED &mdash; (b) CORRECT `ai_check.go` AND BANK UNDER A NEW METRIC NAME, preserving the old series** (Mark, attended 2026-08-28, interactive steering session; verbatim: *"d44 b"*; recorded as a charter stamp per the D-33 precedent.) The existing cost-per-verified-success series FREEZES with an annotation pointing at its successor; the corrected denominator banks under a new metric name from its first run; no banked row is restated. This is the same shape as the two standing precedents: the v0.30.0 cost data (annotated, never re-banked) and the 2026-08-23 skips-exemption correction. Implementation note carried from the row below: the containment proof already exists (ai-check does not call `printVerifyJSON`; `eval_harness` decodes only the four fields), so the fix + rename is self-contained in `internal/eval_harness/ai_check.go` and the banking metadata &mdash; naming the successor metric is the implementing sprint's first task. **May `ai_check.go`'s verify denominator be corrected, given that it moves a KPI with a recorded baseline?** `4d8705699` fixed `ailang verify` to stop skipping contract-less functions with a bare `continue` &mdash; it had reported "11 functions: 11 verified", true only because the denominator was chosen to make it true. That commit states `ai_check.go:289` **has the identical blindness and was deliberately NOT touched**, because it is a separate implementation feeding **cost-per-verified-success**, this mission's headline KPI, which has a banked baseline. It verified the fix cannot leak into banking (ai-check does not call `printVerifyJSON`; `eval_harness` decodes only `verified`/`counterexample`/`skipped`/`errors`; nothing uses `DisallowUnknownFields`). So the divergence is contained but real: two verify implementations now disagree about what counts as verified. **(a) Correct `ai_check.go` and RE-BASELINE** the KPI, annotating the discontinuity. **(b) Correct it and bank under a new metric name**, preserving the old series. **(c) Leave it** and record in code that the KPI's denominator deliberately excludes uncontracted exports. | `4d8705699` commit body, verified against the diff: `cmd/ailang/verify.go` and `verify_print.go` changed (32 and 33 lines); `internal/eval_harness/ai_check.go` is absent from the commit's 18-file stat, so the divergence is real and deliberate rather than an oversight. That commit records its own containment proof: ai-check does not call `printVerifyJSON`, `eval_harness` decodes only `verified`/`counterexample`/`skipped`/`errors`, and nothing decodes with `DisallowUnknownFields`. KPI provenance for cost-per-verified-success is the mission's north-star metric with a first measured baseline of $0.7778187072 (2026-08-22), later corrected to $0.2121 once `no ensures clause` skips were exempted &mdash; i.e. this denominator has already moved once for exactly this class of reason, which is why it is a decision and not a fix. |
| D-45 | RESOLVED | **WITHDRAWN BY MEASUREMENT — this was never a human decision, and the iteration-295 evaluator was right to say so.** Filed as *'bind chain-only or chain+stage?'*. The evaluator (`sonnet`) refuted the premise and the controller reproduced the refutation first-party: at `cmd/ailang/eval_benchmark.go:112`, the **sole** production `SetCorrelation` call site, `stageID` is **already a local variable in scope**, populated 16 lines earlier at `:96` from `evalChain.Store.CreateStage(...)`. So threading it is a parameter addition, not a fork — the answer is `stage`, because the stage id is already there. **Recorded answer: bind chain+stage; it is ordinary remaining work, not an ask.** | The controller filed this because `AIAgent.SetCorrelation` takes only `chainID`, and stopped at the function signature instead of reading its caller. That is a decision manufactured out of an unexamined premise, which Standing rule 8 exists to prevent — a park whose resume condition is 'a human answers' when nothing was in doubt. Caught by generator≠judge: the claims were generated by two `opus` agents and the `sonnet` judge, given this as a named attack target, called it *'avoidance dressed as a design decision'*. Scope is now settled at ~15 LOC in the queue row. |
| D-46 | RESOLVED | **ANSWERED &mdash; `loop`** (Mark, attended 2026-08-28, interactive steering session; verbatim: *"d46 loop"*; recorded as a charter stamp per the D-33 precedent.) The next iteration may write the truthful compound status &mdash; iteration 295's measurement (15/15 ACs green at HEAD, exactly one task &mdash; M1's writer side &mdash; unbuilt) suggests the precedent shape `completed_except_m1_writer`. Premise update measured in this attended session: Mark's `requeued` reset is NO LONGER uncommitted &mdash; it landed on `origin/dev` (local `completed` was just this clone's 23-behind staleness, since reconciled under D-42's first application) &mdash; so the Critical-Principle-0 clobber hazard that stopped the loop from writing is gone. **Who reconciles the `M-MISSION-LOOP-UNIFIED-TELEMETRY` sprint JSON?** One word: `mine` (Mark will set it) or `loop` (the next iteration may write it). Its committed state on `origin/dev` says `completed`; Mark's 2026-08-27 reset to `requeued` with `passes: null` sits **UNCOMMITTED in the main checkout**. Neither value is now accurate: iteration 295 verified **15 of 15** acceptance criteria green at HEAD, with exactly one TASK (M1's writer side) unbuilt. | The file is **tracked, not ignored** (`git check-ignore` rc=1, control also rc=1 — instrument read correctly), so it is committable. The controller did not write it because doing so from a worktree would clobber a human's in-flight uncommitted edit (Critical Principle 0). A precedent for a truthful compound status already exists in this repo: `completed_except_parked_m8`. |
| D-47 | RESOLVED | **ANSWERED &mdash; (a) Register CHAIN-ONLY** (Mark, attended 2026-08-28, interactive steering session; verbatim: *"d47 chain grained"*; recorded as a charter stamp per the D-33 precedent, same session as D-42.) Consequence: `m-openrouter-session-chain-registration` is UN-PARKED &mdash; take the ~2 LOC chain-only change off PR #945's branch (omit `StageID`; the receiver's existing `if stageID == "" { stageID = ss }` fallback carries it), and drop the changelog's *"and stage"* claim so stage attribution is honestly absent rather than misattributed. Option (b)'s per-request correlation id remains available later as a SEPARATE design if stage-bound attribution is ever needed; it was not chosen now because it changes eval-path wire bytes feeding a banked KPI. **How should the OpenRouter `session_id` be registered, given that it is chain-grained and the design doc asks for a stage-bound row?** The iteration-296 evaluator found, and I reproduced first-party, that the delivered mechanism misattributes spans rather than merely failing to resolve them. One `evalChain` is created per **`eval-suite` invocation** covering every benchmark (`cmd/ailang/eval_suite.go:500`), `sessions.session_id` is `TEXT PRIMARY KEY` (`internal/observatory/migrate.go:184`), and the upsert sets `stage_id = COALESCE(excluded.stage_id, sessions.stage_id)` with `excluded.stage_id` non-NULL on every write (`internal/observatory/store_sessions.go:88`) &mdash; so each benchmark **overwrites** the single row and `otlp_receiver.go:468` then gives that one `stage_id` to every span resolving by chain. Bare `ailang eval-benchmark` builds no chain at all (`createEvalChain` in `eval.go` = **0**, control in `eval_suite.go` = **1**), so the only reachable caller is the multi-stage one and the single-stage case where it is correct is the degenerate exception. The contradiction is in the design doc: `m-mission-loop-unified-telemetry.md:146` specifies a row *'bound to the stage'*, keyed by a value `internal/ai/correlation.go:19` documents as *'the chain ID, the finest grain already persisted per run'*. **(a) Register CHAIN-ONLY** &mdash; omit `StageID`; the receiver's `if stageID == "" { stageID = ss }` is already a fallback, so spans join the correct chain and stage attribution is honestly absent. No misattribution, strictly better than today's zero resolution, ~2 LOC off the parked branch, and the changelog's *'and stage'* claim is dropped. **(b) Thread a per-request unique correlation id** onto the wire so the row can be genuinely stage-bound &mdash; the full fix, but it changes eval-path wire bytes and touches a live path feeding a banked KPI. **(c) Redesign the join** so it does not depend on `sessions.session_id` being 1:1 with a stage. | PR [#945](https://github.com/sunholo-data/ailang/pull/945) (DRAFT, not merged) &rarr; `d86399f0a`; evaluator `sonnet` **FAIL 58/100**, two blocking, both reproduced by the controller; commands and rcs in the iteration-296 STATUS stamp and log entry. |
| D-48 | RESOLVED | **V1 designer rotation: replace `pi:ollama/kimi-k3:cloud` with `pi:ollama/deepseek-v4-flash:0731-cloud`** (Mark, attended 2026-08-28, interactive steering session; recorded as a charter stamp per the D-33 precedent). Mark's constraints, verbatim: *"fable kimi sonnet will fail when anthropic quota runs out. kimi seems very slow if its doing this, we need to downgrade to a better model."* &mdash; i.e. the second lane must be (i) NON-Anthropic, so the rotation still has a working designer when the Anthropic bucket dries out (rules out adding sonnet), and (ii) actually working, which kimi is not (first real use: wall_timeout 1802s, 73 tool calls, **0 files written**, fell back to fable; flagged twice, dashboard instance 2 after iter-294 &mdash; the auto-downgrade WORKS, the flag was the un-advanceable rotation + un-appliable policy fix). Lane chosen by Mark from four options offered attended (deepseek-v4-flash:cloud recommended / kimi-via-OpenRouter / minimax-m3:cloud / fable-only): **deepseek-v4-flash** &mdash; flat-rate, vendor-independent of all three quorum reviewers, the most-proven pi lane on the rig, flash-class fast. Applied same session to the shared rotation in `mission-control/SKILL.md` (fleet-wide by construction &mdash; every mission reads that list; live immediately via the skill symlink). Docs-mission seats needed NO change: the same morning's subscription-first ladder (`dc1e9c5fa`) had already moved its designer seed to sonnet and planner/executor to the gpt-5.6-luna/glm-5.3-flash ladder; only a stale guardrail sentence still cited kimi, fixed in this commit. Rotation pointers need no migration (last-used `claude:claude-fable-5` everywhere; deepseek is simply the new next entry). | Attended session 2026-08-28. Supersedes the kimi half of the 2026-08-26 rotation fix while preserving its structural requirement: a second authoring lane independent of every quorum reviewer. |
| D-49 | RESOLVED | **ANSWERED — (a) REPO-LOCAL WINS, duplicates suppressed** (Mark, thread `#852`, TWICE: `2026-08-29T09:06:45Z` verbatim *"D-49: A"* and `2026-08-30T08:11:12Z` verbatim *"D49 - A"*. Recorded as a charter stamp by the attended 2026-08-31 steering session — per the D-33 precedent — because Gate-0 never consumed either comment: iteration 308's STATUS reads *"0 directives since watermark"* while both answers sat on the then-ACTIVE bookkeeping thread for 1–2 days. The weekend's rc=1 crash fires advanced the directive watermark without processing — the crash-swallows-directive class; a controlplane message filed the defect the same session, and the loop owes it a root-cause + fix as a queue item.) Implementation is now an ordinary queue item: repo-local `.pi/extensions` takes precedence and global-installed duplicates are suppressed. **Choose the Pi extension precedence policy when both the global binary-installed directory (`~/.pi/agent/extensions`) and this repo's local `.pi/extensions` contain the same AILANG extensions.** Options: **(a)** repo-local wins and duplicates are suppressed; **(b)** global install is skipped for AILANG-repo sessions; **(c)** retire one distribution channel. The iteration-301 correction PR deliberately does not infer a policy. | Independent designer and evaluator audits found that Pi discovers both distinct paths and the Distribution-v2 design retains both tiers without specifying precedence/deduplication. Mechanical installer defects are corrected in PR #961; this policy remains parked for Mark. |
| D-50 | RESOLVED | **ANSWERED — APPROVE; `execute sprint` is authorized** (Mark, thread `#852` `2026-08-30T08:11:12Z` verbatim: *"D50 - approve"*. Same provenance and watermark-miss note as D-49's stamp — Gate-0 never consumed the comment; recorded attended 2026-08-31 per the D-33 precedent.) The narrowed recovery plan is approved AS DESIGNED — un-park the queue row and execute; this also unblocks #968's disposition. **Approve the narrowed recovery design for `m-coordinator-child-env-opencode-retry-storm` and authorize `execute sprint`?** Recommended: **approve** the recovery plan, which preserves M1 route coherence, narrows executable pinning to the incident-local OpenCode child, and implements durable attempt accounting plus terminal CAS/notification ownership fresh. Alternative: **revise** with the requested scope change. | Iteration 305 recovered iteration 302's unlanded artifacts. The stored round-2 quorum verdict is explicitly BLOCKED, so the old plan's carve-out cannot authorize unattended execution. Designer and planner independently narrowed the overbroad five-adapter sweep; executor proved M1 cherry-picks cleanly and targeted tests pass; independent evaluator PASS 96/100 confirmed the park. Recovery plan: `design_docs/planned/v0_35_0/m-coordinator-child-env-opencode-retry-storm-recovery-plan.md`. |
| D-51 | RESOLVED | **Ratify or replace the charter's countable finish-line unit.** Iteration 309 added one because the comms contract requires every report to state distance to the finish line, and the charter had none; it is explicitly marked PROVISIONAL pending your ruling. The unit is **open queue rows** — measured at iteration 311: **68** `[NEXT]` + **1** `[PARKED]`, distance = that reaching zero. **The problem is that it is anti-correlated with good work.** Iteration 311 landed a milestone, fixed a real encoding defect and corrected two earlier iterations' framing, and moved the number by **0** — while an iteration that files five triage rows moves it *backwards*. Options: **(a) RATIFY as-is** — cheap, derivable, and every report keeps saying "provisional"; **(b) REPLACE with milestone burn-down** — count open milestones across `[IN-SPRINT]`+`[NEXT]` design docs rather than rows, which is what iterations actually move (recommended: it is still derivable, and it would have read "1 of 5" today instead of "0"); **(c) REPLACE with an outcome unit you name** — e.g. eval pass-rate on a fixed set, cost-per-verified-success, or a dated release milestone; **(d) DROP the unit** and let reports say "no finish line", accepting that the Progress row becomes narrative. **Loop's recommendation: (b).** **Default if unanswered:** the loop keeps quoting (a) marked provisional and re-asks nothing — this row simply stays OPEN, which is the status quo and costs nothing but leaves every Progress line misleading.  **ANSWERED — (c) REPLACE. The countable unit is **the number of design docs that remain before v1.0.0 can be declared** — scoped by the bar's own cutoff rule ("a design doc gates v1.0 only if it serves an open clause"), applied to clauses 2–5 (clause 1 is CLOSED and contributes 0). A design doc is the unit, not a queue row and not a milestone: one doc counts once however many milestones it decomposes into, and sprint plans never count. A doc leaves the count when it LANDS, is RULED OUT, or is re-scored off the bar (as m-effect-scope-params was by D-27). Folder location is evidence, not truth — design_docs/planned/v1_0_0/ is a starting inventory, not the definition. A clause naming work that has no doc yet contributes one NEW-DOC unit per named piece of work (clause 4 currently names at least the orchestration flagship and the linear-time regex + URL-parse builtins in that state). **The number is allowed to go UP** — new bugs and features that must ship before 1.0 enter the count, and that is the unit working, not failing; report it as N remaining (was M, ±k this iteration, reason). **Owed in the iteration that consumes this ruling:** publish the first-party inventory — in-scope docs grouped by clause, each with the tag it currently carries — and the resulting N, classifying against the clause TEXT rather than the folder, and excluding rows filed under a clause heading whose own text disclaims bar membership (m-mcp-exact-tool-surface says exactly that about itself). Anything you cannot confidently classify goes in a named UNCLASSIFIED bucket for Mark to rule on — never silently included or dropped. Update the Goal block to this unit and drop PROVISIONAL.** (Mark Edmondson, attended 2026-09-01, recorded directly in this ledger.)| Charter Goal block (added iteration 309, `## Goal — the countable unit (PROVISIONAL, ...)`), which itself says "Mark to ratify or replace" and "defining the true finish line is a judgment only Mark can make". Measured at iteration 311 with `grep -cE '^\*\*\[<TAG>' design_docs/v1-mission.md`. This row exists because that ask was written as charter prose and never as a ledger row, so `mission_decisions.sh --open` returned zero and no report ever surfaced it.  **Attended ruling 2026-09-01** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue; provenance is the commit author — an attended identity, which the fleet bot does not hold and the loop may not author with. Attended session measured the starting inventory first-party before the ruling: design_docs/planned/v1_0_0/ holds 13 files, of which 4 are sprint plans and m-effect-scope-params was re-scored to v1.1 by D-27, leaving ~8 candidates; clause 4 names at least 3 further pieces of work with no doc at all; clause 2 bar text names m-bytecode-vm-parity-bugs as its residue. Honest range at ruling time 10–13, deliberately NOT recorded as a number because clauses 3 and 4 are prose with strikethrough history and need the classification pass this row asks for.|
| D-52 | RESOLVED | **Is the ~40% reaped-slot rate worth an iteration to diagnose?** Of iterations 296–310, **six** (299, 300, 302, 303, 306, 307) produced no record at all — `git log -S "ITERATION <n>"` shows **0** charter commits for each, so they are dying mid-flight, not losing their record to a rotation bug (that half was audited clean this iteration: 270 archive stamps, every other iteration accounted for). The loop can see the frequency and cannot diagnose the cause: a slot that dies exits `rc=0`, so neither watchdog fires and the next iteration inherits a silently skipped number. Recovery costs a whole later iteration each time (308 spent itself recovering 306 and 307). Options: **(a) SPEND ONE ITERATION** on it — instrument the driver to write a heartbeat artifact per gate so a dead slot names the gate it died in (recommended; it is the one thing that would turn six anecdotes into a mechanism); **(b) ACCEPT the rate** as the cost of unattended operation and keep recovering opportunistically via the Gate-2 died-mid-flight traces, which do work; **(c) REDUCE THE FIRE RATE** so fewer slots overlap, trading throughput for completion. **Loop's recommendation: (a)** — the traces recover work but never explain it, and the rate has not fallen. **Default if unanswered:** (b), i.e. nothing changes; the loop keeps reporting the rate each iteration and keeps recovering by hand.  **ANSWERED — (a) SPEND ONE ITERATION on it. Instrument the driver to write a per-gate heartbeat artifact so a slot that dies names the gate it died in. A ~40% reaped-slot rate is too high to keep absorbing by hand, and the recovery iterations cost more than the diagnosis will. Report the MECHANISM when you have it, not just the rate — the traces recover work but have never once explained it.** (Mark Edmondson, attended 2026-09-01, recorded directly in this ledger.)| Measured iteration 311, first-party: charter+archive membership for iterations 296–310, with `git log -S` distinguishing "never written" from "written then removed". Controls fired (ITERATION 310 present in charter = 1; ITERATION 295 present in archive = 1). Prior instances the loop recorded but never aggregated: iterations 306 and 307 (recovered by 308), iteration 302 (recovered by 305), and the standing-rule-7 background-reaper class documented in the skill.  **Attended ruling 2026-09-01** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue; provenance is the commit author — an attended identity, which the fleet bot does not hold and the loop may not author with. Attended ruling on the loop's own recommendation; the rate has not fallen across iterations 296-310 (six slots, 299/300/302/303/306/307, each with 0 charter commits) and iteration 308 spent itself recovering two of them.|
| D-53 | RESOLVED | **Rule on the 4 UNCLASSIFIED docs from the D-51 inventory — they are the difference between N=10 and N=14.** Your D-51 ruling said anything I could not confidently classify goes in a named UNCLASSIFIED bucket for you to rule on, never silently included or dropped. Here they are, as a ledger row rather than charter prose — because D-51's own row exists precisely because that ask was written as prose and `--open` therefore returned zero. **(1) `m-serve-api-live-tool-registry`** and **(2) `m-agent-step-cancellation`**: both sit under the `### Clause 4` *section heading*, which reads *"(Mark: full surface in)"*, while clause 4's *text* names only effect sprints 1–3, the flagship pipeline and the two (now-landed) builtins. `m-agent-step-cancellation` also carries a `**[GATING clause-4]**` queue tag. **Question: does the heading's "full surface in" extend the clause text, or was it superseded by the 2026-07-11 bar v2 wording?** D-51 says the clause text governs, which is why I am asking rather than deciding. **(3) `m-benchmark-ensures-coverage`** (no doc): is adding `ensures` to benchmarks part of clause 5's *"measured baseline"*, or corpus work on the normal road? D-29's second clause was ruled "scoped, not blanket", and iteration 258 measured only **1 of 5** candidates as safe. **(4) `m-module-cache-identity-not-compiler-bytes`** (no doc): its own row says it is *"PRE-EXISTING and independent of that doc's fate"* — clause 5, clause 2 (soundness), or post-v1? **Loop's recommendation: rule 1 and 2 IN** (the `[GATING clause-4]` tag and the heading both point that way, and orchestration is the flagship), **3 OUT** (corpus work), **4 OUT of the bar but keep the queue row** (a real defect, not a release gate) — which would give **N = 12**. **Default if unanswered:** the loop keeps reporting **N = 10** with the bucket named in every report, so nothing is hidden and nothing stalls; the cost is only that the headline number is understated by up to 4.  **ANSWERED — N = 12. (1) and (2) IN, (3) and (4) OUT. (1) `m-serve-api-live-tool-registry` and (2) `m-agent-step-cancellation` are IN. The question this row asked — whether clause 4's heading "(Mark: full surface in)" extends the clause text or "was superseded by the 2026-07-11 bar v2 wording" — is settled by the history, not by judgement: the heading POSTDATES bar v2 by a day and cannot have been superseded by it. `3e230273c` (2026-07-11) is "v1.0 bar v2 — product-shaped". The parenthetical was added the next day by `6f1d1c7a1` (2026-07-12), "v1.0 scope set via full backlog triage (Mark, full-scope)", whose own message records the ruling verbatim: "Mark chose FULL SCOPE: whole clause-3 footgun/DX/prompt cluster + both DX tooling investments + full clause-4 orchestration surface all IN; infra OUT." The heading is therefore a recorded attended scope decision that came after the clause text, not stale prose competing with it. D-51's "the clause text governs" still stands as the default; this is a later explicit ruling that scoped clause 4 wider, which is exactly the case D-51 could not decide alone. (3) `m-benchmark-ensures-coverage` is OUT — corpus work on the normal road, not clause 5's "measured baseline". Independently weakened by iteration 258 measuring only 1 of 5 candidates as safe, and by D-29's second clause having been ruled "scoped, not blanket". (4) `m-module-cache-identity-not-compiler-bytes` is OUT OF THE BAR, and its queue row STAYS — a real, measured defect, not a release gate. `ModuleCacheKey` (internal/pipeline/cache_key.go:37) hashing neither `runtime.Version()` nor build tags/flags is build-cache identity, not language soundness: clause 2 is about the type and effect system, and the same 2026-07-12 triage that ruled clause 4 wide ruled infra OUT. Noted for the record, since it is the one item here that could be read the other way: if a cache that can serve artifacts built under different flags is treated as a soundness hole, clause 2's "no new holes found in triage" would need revisiting. It is not being treated as one. Update the Goal block to N = 12 and retire the UNCLASSIFIED bucket — it is now empty, so reports should stop carrying it.** (Mark Edmondson, attended 2026-09-02, recorded directly in this ledger.)| First-party inventory, iteration 315, classified against clause TEXT with `test -d`-asserted scopes and same-scope controls; published in the charter's Goal block. The four are listed there with the same reasons.  **Attended ruling 2026-09-02** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue; provenance is the commit author `3155884+MarkEdmondson1234@users.noreply.github.com`, which the fleet bot does not hold and the loop may not author with.|
| D-54 | RESOLVED | **ANSWERED — (b): the loop is authorised to branch the main checkout's unpushed `dev` commits, push, and open a PR, leaving the merge to CI** (Mark, thread [#972](https://github.com/sunholo-data/ailang/issues/972) `2026-09-02T07:17:34Z`, verbatim: *"D-54 b"*). The grant is standing and unattended-safe by construction: it is a ref push plus a PR, never `checkout -B`, never a rebase, never a force-push, and nothing is discarded — Critical Principle 0 is untouched, and if the commits should not land the PR is closed rather than merged. **The premise it was filed about is already gone, and by the other option.** Twenty-one minutes after answering, Mark cleared the divergence himself, attended: merge `40f0554c1` (*"Merge origin/dev into dev — sync 25 stranded local commits (Mark, attended)"*) at `2026-09-02T07:38Z`, pushed by the Stop hook. So the authorisation has no work to do TODAY and remains live for the next time the tree goes ahead — which is the point, since the row exists because a divergence that grows at the rate Mark uses that tree cannot wait for him to be at a keyboard. Note what the merge also delivered: `m-spawn-pin-enforcement` and its design doc are now ON ORIGIN and therefore visible to unattended picks for the first time, which was the cost this row was written to stop paying. **Nine commits of real work — including a V1 queue row and its design doc that you filed attended — have been sitting UNPUSHED in the main checkout since 2026-08-31, and the standing reconcile authorisation can never fire to clear them.** `D-42` authorises an unattended reconcile **only at 0 ahead**; the main checkout is **9 ahead / 22 behind with 4 dirty files**, so every iteration correctly FLAGs and skips, and the divergence grows without bound. It is not the content-duplicated case `D-23` covers: `git cherry origin/dev dev` marks **all 9 `+`** (not upstream by patch-id), so this is unpushed work, not stale bookkeeping. **The cost is not cosmetic and it is already being paid.** `design_docs/m-spawn-pin-enforcement.md` and its charter queue row exist only locally — `m-spawn-pin-enforcement` occurrences: **0** in `origin/dev`'s charter, **2** in the local one, **0** files on origin vs 1 locally, against a firing control (`m-docparse-v0340-reports` = 1 on origin). Unattended iterations run from a pin at `origin/dev`, so **the loop has been picking from a queue that does not contain work you filed.** The same 9 commits carry the Fable 5.1 and Tencent Hy4 model registrations, a macOS hook-timeout fix, the CLAUDE.md right-sizing and the launchd cadence change — 21 files, +640/−335. Separately, the running skill resolved through the main checkout was **147 lines behind origin** this iteration, the largest drift recorded. **Options. (a) You push them** — one `git push` from the attended session; nothing else changes, and `D-42`'s 0-ahead precondition is satisfied from then on. **(b) Authorise the loop to open a PR from the main checkout's `dev`** (branch the 9 commits off, push, PR, leave the merge to CI) — no `checkout -B`, no rebase, no clobber, and Critical Principle 0 is untouched because nothing is discarded; the loop still never force-pushes or reconciles. **(c) Extend `D-42` to permit an unattended reconcile at N-ahead once every ahead-commit is proven upstream by patch-id** — does NOT help here, since all 9 are genuinely absent. **(d) Accept it** — the loop keeps flagging and skipping, and the local-only queue row stays invisible to every unattended pick. **Loop's recommendation: (b)**, because it is the only option that does not depend on you being at the keyboard, and it degrades safely — if the commits should not land, the PR is closed rather than merged. (a) is faster if you are there now. **Default if unanswered:** the loop keeps FLAGging and skipping per `D-42`, reports the ahead/behind counts every iteration, and re-states in each report that `m-spawn-pin-enforcement` is invisible to unattended picks. Nothing stalls and nothing is lost — the work simply stays unpushed and the queue stays incomplete. | Measured first-party, iteration 320, 2026-09-02, in the pinned worktree at `origin/dev` `28002af1e`: main checkout `dev` vs `origin/dev` = **0 ahead / 0 behind** (`git rev-list --left-right --count dev...origin/dev` -> `0 0`), against iteration 319's Gate-4 reading of 22 ahead / 31 behind. Iteration 319's own Gate-5 skill edit `7292ec780` **is now an ancestor of `origin/dev`** (`git merge-base --is-ancestor` rc=0, control on `28002af1e` firing), and the RUNNING skill — `cmp` against the **resolved** `readlink -f` symlink target, i.e. the main checkout, not the pin's own copy — is **byte-identical to `origin/dev`** for the first time in at least four iterations. Reflog attribution: `dev@{2026-09-02 09:38:35 +0200}: commit (merge)`, i.e. attended, not the loop. Directive read with `scripts/mission_directives.sh --issue 972 --since 2026-09-02T07:05:25Z`, which reported 1 directive of 21 comments; the allowlist is enforced in the script. Measured first-party, iteration 317, 2026-09-02, on `origin/dev` `20cce785e`: `git cherry origin/dev dev` = **9 `+`, 0 `-`**; `git diff --stat origin/dev...dev` = 21 files, +640/−335; charter occurrences of `m-spawn-pin-enforcement` **0** on origin vs **2** local, design-doc files **0** vs **1**, control `m-docparse-v0340-reports` = **1** on origin. Dirty tracked files in the main checkout: `.claude/skills/mission-control/SKILL.md` (an uncommitted Gate-5 edit), two `docs/static/benchmarks/os/*.json`, `std/fs.ail`. `D-42` (attended 2026-08-28) and `D-23` (`#745`, 2026-08-22) are the standing authorisations this row does not re-litigate; it asks the question neither anticipated — what to do when 0-ahead is never reached. |
| D-55 | OPEN | **Does the compile-cache artifact-verification design have to bound ADVERSARIAL gob-decode work, or is the accidental-corruption threat model enough to unblock it?** `m-compile-cache-unverified-artifacts` (iteration 328) is BLOCKED after one revision and one re-quorum. `gemini-3-1-pro` passed both rounds; `oc-glm-5-2`'s objection is CLOSED (the judge invoked the CLI and the three serve-api flags are real and work). The single live objection is `gpt5-6-sol`, rejecting twice on the same point: SHA-256 beside the blobs proves CONSISTENCY, not that the bytes came from our compiler, so the doc's "compiler-origin" trust assumption is not enforced by key, stamp or hash. **Options.** (a) RULE THE THREAT MODEL SUFFICIENT — accept accidental-corruption scope, unblock, the doc goes straight to the planner next iteration, and adversarial hardening is filed as its own backlog row. (b) REQUIRE ADVERSARIAL HARDENING NOW — the doc gains a decode-bounding section and a third quorum round, costing roughly one extra iteration before any fix ships. (c) SPLIT — land the correctness fix now under (a) and open a separate hardening doc immediately. **Loop's recommendation: (a).** The live user-facing defect is silent wrong-program execution; adversarial cache hardening is a separate problem that HEAD is already strictly worse at, and blocking a strict improvement on it leaves users exposed to the confirmed bug meanwhile. **Default if unanswered: (a), applied at the next iteration** and recorded as a controller routing call rather than as a ruling, with the hardening row filed alongside it. | Measured first-party, iteration 328, 2026-09-05, both directions. PRE-EXISTING and REDUCED by the doc: at HEAD `internal/pipeline/cache_store.go`'s `LoadArtifacts` reads unbounded and gob-decodes with **0** byte ceilings and **0** hash checks in the whole file (known-positive control `os.ReadFile` = 5 hits in that file), while the doc adds a 16 MiB per-blob ceiling AND hash-verification before any decode. Judge-verified independently: Go's `encoding/gob` already caps preallocation against remaining input, so a 16 MiB-capped blob cannot be amplified arbitrarily. AGAINST: a poisoned cache blob is decoded on every routine `ailang run` **without the victim knowingly compiling anything**, unlike a malicious `.ail` file — though the cache sits in the user's own project directory, where anyone able to write it can already edit the sources the compiler reads. Quorum artifacts `.ailang/state/mission-quorum/m-compile-cache-unverified-artifacts-2026-09-05T11-31-14Z.json` (round 1) and `-2026-09-05T11-43-52Z.json` (round 2). Defect reported at [#1046](https://github.com/sunholo-data/ailang/issues/1046). Evaluator `sonnet` PASS 92/100 concurred that PARK was the correct disposition, on the ground that this objection needs controller judgement and carries no verbatim fix. |
| D-56 | OPEN | **`gpt-6-astra` is BOTH a designer-rotation entry and a quorum reviewer, so on its turn it judges the doc it just wrote — which rule should the loop follow permanently?** Added by Mark (attended 2026-09-05) as a third rotation entry, and on the same day it straight-swapped `gpt5-6-sol` out of the design-quorum roster (`cmd/ailang/design_quorum.go`). The rotation pointer now sits at `claude:claude-fable-5-1`, so **the NEXT iteration's designer is astra** and the collision is live rather than hypothetical. This is the strong, model-level form of the self-marking defect, not the weaker vendor-level one, and your stated principle is *"ideally no model provider marks its own work"*. The mission-control skill records an interim workaround (author with the next rotation entry, or substitute `gpt5-6-sol` back into the OpenAI seat for that doc only) and a PROPOSED-not-ratified fix. **Options.** (a) RATIFY THE PROPOSED FIX — the OpenAI quorum seat becomes *the OpenAI model that did not author this doc*: astra by default, `gpt5-6-sol` when astra designed it. Keeps three independent vendors AND every reviewer independent of the author; costs one small change to the reviewer-selection code. (b) DROP ASTRA FROM THE ROTATION, keep it as a reviewer — simplest, but it undoes your 09-05 decision to vary between two fable-class authoring lanes. (c) DROP ASTRA FROM THE QUORUM, keep it as a designer — restores `gpt5-6-sol` as the permanent OpenAI reviewer; also simple, but you swapped it out deliberately. (d) ACCEPT THE COLLISION and let the loop note it each time. **Loop's recommendation: (a).** It is the only option that keeps both of your 09-05 decisions intact. **Default if unanswered:** the loop keeps applying the skill's interim workaround — author with the next rotation entry on astra's turn and record the substitution — which is correct but silently costs astra its authoring slot every cycle. | Iteration 334, 2026-09-06. First-party: the mission-control roles table names astra as rotation entry 2 AND flags the quorum collision on the same line; `ailang design-quorum --help` reports the default reviewer set as `gpt6-astra,gemini-3-1-pro,oc-glm-5-2`; this iteration's own two quorum artifacts both list `gpt6-astra` as a reviewer. Rotation pointer `~/.ailang/state/mission-v1-designer-rotation` = `claude:claude-fable-5-1` after this iteration, i.e. astra is next. |
| D-57 | OPEN | **Which cache module-directory naming direction should resume after the bounded re-quorum rejected the hybrid?** (a) RETAIN the 38-byte ASCII prefix plus 16-hex full-ID hash, explicitly accepting limited readability and collision resistance rather than injectivity; smallest correction, existing stamp mismatch rejects wrong-ID artifacts. (b) PURE HASH; remove the slug and choose digest width in the design (full SHA-256 means 64 hex characters), trading readable names for simplicity. (c) BASENAME/PARENT REDESIGN; specify portable component parsing and test representative Windows paths, requiring another design pass. **Recommendation: (a)** — the prefix is a limited hint, never an identity guarantee; GLM’s claim that a single separator produces two underscores is false under the measured byte algorithm. This counterevidence does not override its requested change of direction. **Default if unanswered: HOLD implementation immediately and indefinitely; no runtime snapshot copy.** A human ruling, completed design gate, and planner synchronization are all required before M1. | Iteration336: both quorum rounds 2 rejects/1 pass, all three external reviewers present; raw artifacts in `docs/sprint-retros/iter336-cache-module-id-encoding-quorum/`. Sol’s final source-attribution objection answered with an eight-file exact-commit diff (zero changes); GLM’s remaining alternatives dispute the direction, so no narrow-refinement carve-out or third quorum. Independent evaluator reviews the park, not permission to override quorum. PR [#1061](https://github.com/sunholo-data/ailang/pull/1061). |
| D-58 | OPEN | **Pi runner content-delta contract:** (A) authorize one fresh designer revision and quorum to settle snapshot failure ordering, bounded comparison and physical artifact aliases; (B) keep this approach parked and commission a Git-native content-comparison design. Recommend A. Default: HOLD immediately until a human ruling; neither option skips design/planner/executor/evaluator gates. | Iteration337, two complete quorum rounds rejected by all three external reviewers. Narrow-refinement considered but not used: remaining controller blockers require design judgment beyond verbatim reviewer fixes. See planned/v0_35_2/m-pi-runner-worktree-assertion-vacuous-on-revision.md and docs/sprint-retros/iter337-pi-runner-quorum-r2.json. Approval inbox_1788680529734_00877707 delivered byte-exact. |
<!-- decision-ledger:end -->

## Goal — the countable unit (RATIFIED 2026-09-01 by Mark, attended; ledger `D-51`)

The 2026-08-31 comms contract requires every report to state **distance to the charter's finish line
in the charter's own countable unit**. Iteration 309 added a provisional one (open queue rows);
`D-51` replaced it, because that unit was anti-correlated with good work — an iteration that landed
a milestone and fixed a real defect moved it by 0, while an iteration filing five triage rows moved
it backwards.

**Unit: the number of DESIGN DOCS that remain before v1.0.0 can be declared.**

Scope is the v1.0 bar's own cutoff rule, one section below — *a design doc gates v1.0 only if it
serves an open clause* — applied to clauses **2–5**; clause 1 is CLOSED and contributes 0.

- The unit is a **design doc**, never a queue row and never a milestone. One doc counts **once**,
  however many milestones it decomposes into. Sprint plans (`*-sprint-plan.md`) are not design docs
  and never count.
- A doc **leaves** the count when it LANDS, is RULED OUT, or is re-scored off the bar (as
  `m-effect-scope-params` was by `D-27`).
- **Folder location is evidence, not truth.** `design_docs/planned/v1_0_0/` is a starting inventory;
  the clause text is the definition. A row filed under a clause heading whose own text disclaims bar
  membership does **not** count (`m-mcp-exact-tool-surface` says exactly that about itself).
- A clause naming work that has **no doc yet** contributes one **NEW-DOC** unit per named piece of
  work. Clause 4 currently names at least the orchestration flagship and the linear-time regex +
  URL-parse builtins in that state.
- **This number is allowed to go UP, and that is the unit working rather than failing.** New bugs and
  new features that must ship before 1.0 enter the count. Report it every iteration as
  **`N remaining (was M, ±k this iteration, reason)`**.

**DELIVERED — first-party inventory, iteration 315 (2026-09-01).** Classified against the clause
TEXT, not the folder, per the cutoff rule above. Scope asserted (`test -d`), and every count below
carries a same-scope control; `planned/v1_0_0/` holds **13** `.md` files = **6** sprint plans
(which never count) + **7** design docs.

**N = 12** (iteration 333: was 13, **−1** — `m-compile-cache-unverified-artifacts` LANDED complete and left the count) — 9 existing design docs + 3 NEW-DOC units. Inside the attended session's honest range
of 10–13. Was 10; `D-53` (attended 2026-09-02) ruled the clause-4 pair IN, +2.

| clause | IN (existing docs) | IN (NEW-DOC units) | subtotal |
|---|---|---|---|
| 2 — soundness | `m-bytecode-vm-parity-bugs` (named verbatim as the clause's residue); `m-effect-row-var-unification` (P0 effect-soundness hole filed as `#616` under the `### Clause 2` heading — it falsifies the clause's dated "zero P0s ✅") | `m-run-selector-enumeration-floor` (no doc anywhere; `find` → **0**, control → **1**) | **3** |
| 3 — fleet-tier accessibility | `m-eval-slim-prompt-self-discovery` (= R3.1, named inside the clause's own gate sentence) | prompt-deletion pass **R1.2** — the clause's `≤1,500`-line teaching-prompt bar is unmet and measurable: active prompt `v0.16.6` is **2552** lines | **2** |
| 4 — orchestration flagship | `m-effect-clock-net-fs-modes` (effect sprint 3/4; the only unlanded doc of the sprints 1–3 the clause requires); `m-serve-api-live-tool-registry` and `m-agent-step-cancellation` (both ruled IN by `D-53`) | `m-v1-orchestration-flagship` (the verified multi-step AI pipeline; `find` → **0**) | **4** |
| 5 — cost credibility | `m-contract-verification-coverage`; `m-verify-bounded-unrolling-false-counterexample`; `m-cohort-manifest-build-provenance` | none | **3** |

**Correction of record — clause 4's own text is stale, and it moved N down by 2.** The clause says
the *"linear-time regex + URL-parse builtins (both verified absent)"*. **Both landed on
2026-07-11/12** and are live at HEAD: `m-stdlib-regex.md` and `m-stdlib-url-parse.md` are both in
`design_docs/implemented/v0_30_0/` (queue item 12's own row says *"Closes v1.0 bar clause 4's
URL-parse half"*), and `internal/builtins` carries `_regex_` in **1** file and `_net_url_parse` in
**2** (negative control, a fabricated builtin name: **0**). So clause 4 names **1** undoc'd piece,
not the ≥3 the attended estimate read out of that stale parenthetical.

**OUT, with reasons** (the ones worth recording): `m-effect-scope-params` — re-scored to v1.1 by
`D-27`, the precedent `D-51` names. `m-dialect-keyword-diagnostics` — self-disclaims bar membership
(*"not a v1.0 release bar"*), the `m-mcp-exact-tool-surface` exclusion. `m-effect-refinement` —
umbrella doc whose decomposition routes every open phase to a sprint doc; counting it would
double-count sprint 3. `m-bytecode-pattern-arity-fix` — LANDED (`git merge-base --is-ancestor
0625059d3 HEAD` rc=0) but never moved out of `planned/`, i.e. folder-as-evidence failing exactly as
the unit's rules anticipate. `m-contracts-as-code-vertical` — folds in as the flagship's worked
example; one doc counts once.

**UNCLASSIFIED — RETIRED, empty. Ruled by `D-53` (Mark, attended 2026-09-02): 1 and 2 IN, 3 and 4
OUT, giving N = 12.** Reports should no longer carry this bucket. The decisive evidence on 1 and 2
was historical, not editorial: clause 4's heading *"(Mark: full surface in)"* POSTDATES the
2026-07-11 bar v2 (`3e230273c`) by a day — it was added by `6f1d1c7a1` (2026-07-12), *"v1.0 scope
set via full backlog triage (Mark, full-scope)"*, whose message records *"full clause-4
orchestration surface all IN; infra OUT"*. So it could not have been superseded by bar v2; it is a
later attended scope ruling. **3** is OUT as corpus work; **4** is OUT of the bar with its queue row
kept — build-cache identity, not language soundness, and the same triage ruled infra OUT. The four
items as originally recorded, for the audit trail:
1. `m-serve-api-live-tool-registry` and 2. `m-agent-step-cancellation` — both sit under the
   `### Clause 4` **section heading**, which reads *"(Mark: full surface in)"*, while clause 4's
   **text** names only effect sprints 1–3, the flagship pipeline and the two (landed) builtins.
   `m-agent-step-cancellation` additionally carries a `**[GATING clause-4]**` queue tag. Question:
   does the heading's "full surface in" extend the clause text, or was it superseded by the
   2026-07-11 bar v2 wording? The tag is a queue annotation and `D-51` says the clause text governs
   — which is why this is a question and not a classification.
3. `m-benchmark-ensures-coverage` (NEW-DOC) — is adding `ensures` to benchmarks part of clause 5's
   *"measured baseline"*, or corpus work on the normal road? `D-29`'s second clause was ruled
   "scoped, not blanket", and iteration 258 measured only **1 of 5** candidates as safe.
4. `m-module-cache-identity-not-compiler-bytes` (NEW-DOC) — its own row says it is *"PRE-EXISTING
   and independent of that doc's fate"*. Clause 5, clause 2 (soundness), or post-v1?

**Reporting convention from here:** `N remaining (was M, ±k this iteration, reason)`. Iteration 315
establishes the baseline, so it reports `N = 10 (baseline established; ±0)`. `D-53` then moved it
to `N = 12 (was 10; +2, clause-4 pair ruled IN)`.

## STATUS 2026-09-06 — ITERATION 338: **Shared-ref provenance LANDED; independent MiniMax PASS95; inherited red fixed forward.** Recovered the died-mid-flight Gate-1 drift sprint, ran designer/planner/executor/evaluator roles through the declared pins, and landed PR #1063 as `0b7f3e3af`. Gate 1/3/3b/4 now carry one full-SHA/read-time observation and distinguish missing evidence from drift; `mission-base.sh` has 8 non-vacuous arms. PR #1063 was fully green including Windows before merge. Its post-merge tree inherited the already-red `aebf8bb73` heading ratchet; Sol fixed the environment-dependent sibling/CRLF census and MiniMax R3 PASS95, then PR #1064 landed as `b50bb366e` with 20/20 checks green. Evaluator route: Ollama transport failed closed (missing sandbox-runtime); configured OpenRouter MiniMax fallback judged R2 PASS93 and R3 PASS95, generator!=judge; first R3 emission errored and same-round MiniMax continuation completed it, no controller score. R1 PASS92. N=12, goal unmoved. Ledger58/four OPEN D-55–D-58; no human input inferred. Actual metered $1.48796278; GLM flat-rate imputation $0.06206547 separate. Main11dirty paths untouched. Full record: v1-mission-log.md iteration338.

## STATUS 2026-09-06 — ITERATION 337: **Pi runner delta design PARKED on D-58 after two complete quorum rejections; no implementation.** DeepSeek designer and one revision ran; all three external reviewers rejected both rounds, none absent. Sol planner/executor Agent roles performed read-only readiness audits; design gate prevented planning/execution. Independent MiniMax docs-only PASS82/100. No self-approval or third quorum. Dirty/no-op and untracked/no-op both reproduce false ok; status snapshots alone miss re-edits. API cost$0.213206; GLM flat-rate imputation$0.0271354 separate. N=12, goal unmoved. Ledger58/four OPEN D-55,D-56,D-57,D-58. Main14dirty paths untouched. Full record: v1-mission-log.md iteration337.

## STATUS 2026-09-06 — ITERATION 336: **Cache encoding PARKED on D-57; independent MiniMax PASS 14/15, docs-only correction/park review.** Astra designer clarified byte mapping and reproduced 16/16 examples; two complete quorum rounds both BLOCKED (2 rejects/1 pass), and the remaining naming-direction dispute forbids the narrow-refinement carve-out. Sol planner made plan/snapshot explicitly blocked; Sol executor audited zero production changes and 0/4 milestones. Actual MiniMax judge ran via Agent wrapper, generator≠judge by vendor; all four roles dispatched, no implementation gate bypass. PR #1061 banks evidence. N=12 remaining, unchanged. Ledger57/three OPEN D-55,D-56,D-57; no human answer inferred. API-priced cost$0.224037, flat-rate GLM imputation$0.05664417 reported separately. Main14dirty paths untouched. Full record: v1-mission-log.md iteration336.

---

## STATUS (rotation rule — added 2026-07-14, Fable-quota diet)

Newest **3** STATUS stamps live here; older ones move to
[v1-mission-status-archive.md](v1-mission-status-archive.md). **Loop: at Gate 4, after
adding your stamp, move the now-4th stamp to the TOP of the archive file.** **SELF-HEAL
(added 2026-07-22 iter-83, 2nd instance of drift after iters 77+82 hand-corrected an
already-drifted N>4): if MORE than 3 stamps remain after adding yours, move ALL but the
newest 3 to the archive top, newest-first — do not assume exactly one over-count.**
Rationale: every iteration re-reads this charter — 30+ stamps were ~500 lines of history
tax per read, on the scarcest model budget. The append-only history lives in the log + archive.


## The v1.0 bar — v2, PRODUCT-SHAPED (RATIFIED 2026-07-11, Mark; supersedes the 2026-07-10 hygiene bar)

**The 1.0 claim**: ***the verified AI-orchestration language*** — an AI author gets a
verified-correct program at the lowest cost, and AI orchestration is type-checked. Derived from
[m-fable-strategy-review](planned/m-fable-strategy-review.md) (Design Freeze items 1+2 ratified
by Mark 2026-07-11: cost-per-success is the headline KPI; orchestration is the vertical. Item 3,
trace publication, stays deferred — post-v1).

**The cutoff rule**: a design doc gates v1.0 **only if it serves an open clause below.**
Everything else ships on the normal v0.2x road or is post-v1 — regardless of folder history.
The v1 hygiene bar (2026-07-10) is absorbed: its clauses are 1–2 below, both essentially done.

1. **STABLE** ✅ — the 1.x surface promise (docs/docs/reference/stability.md, iteration 5;
   tier assignments RATIFIED by Mark 2026-08-04, attended — clause 1 fully CLOSED; was: ratification parked for Mark).
2. **SOUND** — zero P0s ✅ (all four closed, iterations 1–4); residue: ~~m-check-strict-fallbacks~~
   **[LANDED iter 101, PR #479]**, m-bytecode-vm-parity-bugs (≤2d, queued).
3. **ACCESSIBLE TO THE FLEET TIER** (strategy R1+R4): the finite, documented mid-tier footgun
   list burned down — the 3 parser/type inconsistencies fixed (match-in-HOF-lambda parse,
   polymorphic-arithmetic panic, arity call-style diagnostic), m-syntax-ai-forgiving landed
   (kills the ~32% small-model failure class), and the teaching prompt ≤1,500 lines with a
   rig-A/B showing no pass-rate loss (R3.1 measures the curve first; the deletion pass stays
   gated on replacement diagnostics landing, per m-diagnostic-coverage's deferred section).
   **Gate = this finite work.** The sonnet-class ≥ −5pts outcome is measured and published at
   release, NOT blocking (per Mark: partially vendor-dependent).
4. **ORCHESTRATION FLAGSHIP** (R6 + R7 + effect refinement): the four effect sprints (public
   docs promise; sprint 4/4 `m-effect-scope-params` RE-SCORED to v1.1 per **D-27**, Mark attended
   2026-08-22 — clause 4 requires sprints 1–3 only); a **verified multi-step AI pipeline** as the flagship example (typed LLM calls
   + budgets + secret-flow + replay) with orchestration benchmarks promoted into the default
   rotation and README/site positioning led by it; **linear-time regex + URL-parse builtins**
   (both verified absent — an orchestration 1.0 without them is a credibility hole).
5. **COST CREDIBILITY** (R3): the dashboard headline KPI flips to **cost-per-verified-success
   vs Python, per tier**, and v1.0 ships with the measured baseline + trajectory. The ≤3×
   zero-shot / ≤1.5× agent targets are the tracked post-1.0 trajectory, NOT release gates.

## How the mission runs (each iteration — codified in the mission-control skill)

1. **OBSERVE** — read this doc's backlog + last log entry + agent inbox + eval health. Deterministic, cheap.
2. **PICK** — top open queue item per the ordering policy. **Verify against repo reality first**
   (git log + code + tests), never trust a status header — stale-status docs are how we shipped
   M-EVAL-BENCH-UI twice (2026-07-10 lesson: doc said Planned, all 4 milestones were long done).
3. **ROUTE + EXECUTE** — through the honed inner loop with the model routing policy below:
   design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator. Sprint work runs in
   an isolated git worktree (concurrent-agent safety). Max 3 evaluator rounds, then park as
   `needs-human-review` and move on.
4. **RECORD** — append a log entry (fixed template in v1-mission-log.md): what shipped, evaluator
   score, routing evidence row, ruled-out ledger additions, next.
5. **RETRO** — route observed friction into exactly one lane: **skill fix** (edit the offending
   SKILL.md — max ONE skill edit per iteration, each traced to ≥2 recorded frictions), **process
   fix** (edit this doc), or **backlog item** (new/re-prioritized design doc). Then send the
   morning report to controlplane.

## Model routing policy (evidence-updated, not vibes)

| Role | Model | Why / evidence |
|---|---|---|
| Mission controller (this loop: triage, pick, judge, retro) | **Opus** — opus-first PREFS since 2026-07-16 (Mark: "Fable for real high cognition stuff not execution"; the long orchestration session is mechanical and was the residual Fable drain even after M1a). Fable = emergency fallback only | The 07-14 Fable revert burned the weekly bucket at 2h cadence; orchestration doesn't need the top tier |
| Design docs (create/review) | **ROTATION across top-of-line models (Mark 2026-07-17)**: `claude:claude-fable-5` (via `claude-sub`) ⇄ `codex:gpt-5.6-sol`. **gemini caveat (iter 53):** G4 clone-over-egress LIVE-LANDED, so gemini is fleet-ready — but as an in-sandbox **evaluator**, not a designer: `CapRemoteSandbox` means it cannot edit a worktree, so a designer spawn can't write the doc without the text-bridge (unwired). gemini's designer-rotation entry is therefore PARKED for Mark's fleet-role ratification (evaluator recommended). Each new-doc iteration takes the next designer in rotation; record `(designer, quorum outcome)` in the evidence row | Every design passes the QUORUM regardless of author — the quorum is the quality gate, so authorship diversity is free comparative signal on which frontier model designs best for AILANG. Fires only when a doc is created/revised |
| Sprint planning | **Opus** (claude-opus-4-8) | Plan quality determined execution success historically |
| Sprint execution | **Opus** — the default, per Mark 2026-07-10 | Sonnet execution was a false economy (needed corrections); also `dev-cycle.md` had silently pinned sonnet |
| Sprint evaluation | **Sonnet** — `$MISSION_EVALUATOR_MODEL`-PINNED sub-agent (default changed fable→sonnet 2026-07-16, Mark directive #399; see below). generator≠judge holds STRUCTURALLY (sonnet ≠ the opus executor pin) AND is now ENFORCEABLE (sonnet is an Agent-tool alias; fable was not — F1 — so the fable default re-routed to sonnet every iteration anyway: 31, 36) | Behavioral independence (fresh sub-agent, re-runs tests, adversarial probes) retained on top |

> **✅ Evaluation-independence (RESOLVED 2026-07-16, iteration 38):** the evaluator default is now
> **Sonnet** — a distinct model from the Opus executor, so generator≠judge model-diversity is
> restored (the 2026-07-11 "Opus-evaluates-Opus rubber-stamp risk" is gone) AND it is *enforceable*
> (sonnet is an Agent-tool pin; the old `fable` default was not — F1 — so it silently re-routed to
> sonnet every iteration anyway; this makes that the standing state, not a per-iteration patch).
> Behavioral value (independent test re-runs, cross-history non-vacuity, distinct-sample recounts)
> is unchanged. Fable is retired from the every-iteration evaluator slot to protect the weekly quota
> (it fires every iteration, unlike the designer which fires only on new docs).
>
> **⚠ CORRECTION (2026-07-16 evening, Mark + interactive session): "Fable quota-exhausted until
> 2026-08-01" was a MISDIAGNOSIS — OAuth Fable was available the whole time.** The tell: OAuth
> buckets reset **weekly Monday 07:00**; an "until the 1st" date is the **API key's monthly
> cycle**. Root cause: `~/.zshenv` sources `secrets.env`, so every tool shell re-exports
> `ANTHROPIC_API_KEY`; nested `claude -p` calls (the `claude:` CLI lane) therefore billed the
> METERED API — iteration 37's fable designer+evaluator runs were API-billed $, and the key's cap
> then produced the fake "Fable exhausted" error. Fixed in the skill: every nested `claude` call
> now strips the keys at the call-site (`env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN`).
> **The `claude:claude-fable-5` designer lane is AVAILABLE again — do not treat Fable as gone
> until 08-01.** Any future "quota" error naming a reset date that is not a Monday = you are on
> the API key; fix the leak, don't fall back.
| Mechanical tasks (doc moves, regen, banking) | Sonnet allowed | Only with deterministic verification; promotion beyond this requires evidence |

**DEMAND-EVIDENCE GATE (Mark 2026-07-23 — ?-op, block-let-separator, and |> ALL failed it in one week):** ERGONOMICS-SUGAR items (new operators, syntax conveniences) require DEMONSTRATED DEMAND BEFORE PICK — a 60-second corpus grep (parse-error/usage counts in eval banks) is mandatory Gate-2 evidence; zero-demand items go to EVIDENCE-GATED ICEBOX without spending a quorum round. Technical soundness is not the bar; observed need is.

**FORUM RULE (Mark 2026-07-21 — "it takes too long to do quick tasks like ab evals… reserve
this for features"):** the full iteration pipeline (quorum → plan → execute → evaluate → CI) is
for FEATURE-shaped work: code that ships, with quality risk worth the ceremony. Quick
EXPERIMENT/EVAL-class tasks (A/Bs, probes, demos, measurements) do NOT ride full iterations —
they run DIRECT: an interactive session, or a single bounded controller-lane step banking results
+ one evidence row. The fmt A/B's 3-day design→park→greenlight→integrity→execute arc vs the
58-cent 20-minute interactive demo is the canonical example. In-flight experiments finish as
planned; new ones default to the direct lane.

**Evidence rule**: every sprint's log entry records `(model, task class, evaluator round-1 score,
rounds-to-pass, corrections)`. A routing change (either direction) requires ≥3 data points and is
made in RETRO, recorded here with a dated stamp.

> **ENFORCED 2026-07-15 (m-mission-agentic-provider-routing M1):** this table is no longer prose.
> The driver exports `$MISSION_PLANNER_MODEL` / `$MISSION_EXECUTOR_MODEL` / `$MISSION_EVALUATOR_MODEL`
> and mission-control Gate 3 spawns each heavy role as a model-PINNED sub-agent (the controller
> session runs `$MODEL` only). **Before M1, every role inherited the single session model → 100%
> Fable burn** (the driver had been Fable-first since 07-14). Execution now bills the executor pin
> (Opus), not the controller (Fable); generator≠judge is restored (Fable evaluator ≠ Opus executor).
> M2 extends the evidence rows with `(provider, agent, $/quota)`; M3 A/Bs the **sprint-planner
> down-tier** (kept at Opus until ≥3 datapoints — do NOT lower it on this hypothesis alone).
> Cross-provider AGENT executors (codex/motoko/managed_agents) ride the same env once fleet Phase C
> resolves a value like `codex:gpt-5.6` in the spawn.
>
> **AMENDED 2026-07-16 (Mark: "Fable for real high cognition stuff not execution"):** the
> controller session itself was the residual Fable drain after M1a (a ≤6h mostly-mechanical
> orchestration session on the scarcest model). Driver PREFS are now **opus-first**
> (`claude-opus-4-8,claude-fable-5`; Fable = emergency fallback only) and design-doc-creator moved
> from inline to a **`$MISSION_DESIGNER_MODEL`-pinned sub-agent (fable)**. Net Fable spend per
> iteration = ~~two bounded sub-agents: designer (only when a new doc is needed) + evaluator~~
> ONE bounded sub-agent: the **designer** only (fires only when a new doc is needed). The evaluator
> moved OFF Fable to **sonnet** in iteration 38 (below) — this also RESOLVES the iteration-36/37
> inconsistency between this clause and the "evaluator→sonnet unless ≥3 datapoints" rule (Mark's
> #399 directive settles it: not fable).
>
> **AMENDED 2026-07-16 iteration 38 (Mark directive #399: "once we have gemini via managed agents
> and openai we can use one of those instead for evaluator? so default can be gemini (if able to git
> clone the codebase etc)? otherwise sonnet-5"):** evaluator default moved **fable → sonnet**.
> gemini (managed_agents) — Mark's *preferred* default — is NOT viable as the evaluator today, on
> two independent counts VERIFIED this iteration: **(1) architectural (code-proven)** — the
> managed_agents request body carries only `Directive`+`SystemPrompt` over a server-side
> `CapRemoteSandbox` (`internal/executor/managed_agents/managed_agents.go:164`); there is no repo
> upload, so the agent cannot see the sprint's UNCOMMITTED worktree changes nor re-run local tests
> (at most it could `git clone` the *public* origin/dev, which lacks the changes) — exactly the
> "if able to git clone the codebase" gap Mark flagged; **(2) operational (live-observed)** — a
> bounded `ailang exec gemini` probe timed out (`http2 timeout awaiting response headers`), same
> class as iterations 36-37. Per Mark's own ladder this resolves to **sonnet-5**. gemini-as-evaluator
> is a queued follow-up (**m-gemini-evaluator-diff-bridge**): needs a bridge that ships the sprint
> diff + changed files into the directive text AND the Vertex backend returning reliably. NOTE:
> **codex (openai)** is a viable local distinct-provider evaluator alternative (it runs a sandboxed
> local CLI → CAN read the worktree + re-run tests; openai≠anthropic satisfies generator≠judge) —
> but Mark's stated default ladder is gemini→sonnet-5, so sonnet is the default; codex-as-evaluator
> requires the executor NOT be codex (generator≠judge) and stays opt-in.

### Right-sizing table — the (provider, agent, tier) hypothesis (M2)

Landed 2026-07-16 (m-mission-agentic-provider-routing M2). This is the *hypothesis* that the routing
evidence rows below test — updated by the ≥3-datapoint evidence rule, never by vibes. Canonical source:
[design_docs/planned/v0_30_0/m-mission-agentic-provider-routing.md](planned/v0_30_0/m-mission-agentic-provider-routing.md).

| Role | Agentic? | Needs | Tier hypothesis | Agent candidates |
|---|---|---|---|---|
| Controller (pick/judge/retro) | agent (claude-code) | orchestration judgment | **mid** | claude-code (home harness) |
| Design-doc-creator | agent (`check` in loop) | deep spec reasoning (highest leverage) | **strong** | strong claude/codex + live quorum |
| **Sprint-planner** | agent-capable | decompose a quorum-reviewed doc | **MID (down-tier)** — kept at Opus until M3's ≥3-datapoint A/B | mid codex/gemini/motoko |
| Sprint-executor | AGENT (heavy) | tool-using coding | **strong AGENT** (not just a model) | **codex / motoko / claude**; motoko may over-perform on AILANG (M1b wired codex) |
| Sprint-evaluator | AGENT (re-runs tests) | behavioral verification | **mid**, distinct provider from executor | gemini/codex ≠ executor |
| Mechanical (moves/regen) | no | deterministic | **low / local** | local-GPU (Phase D) |

> The model-routing table above (Opus-first) is the CURRENT enforced assignment; this right-sizing
> table is the tier *hypothesis* those assignments are converging toward as evidence accrues. Where
> the two differ (e.g. controller runs Opus today but the hypothesis is mid-tier), the gap is a
> deliberate, evidence-gated decision — a routing change requires the ≥3-datapoint rule.

## Rig integration — the two-tier rule

`rig.lock` (`~/.ailang/state/rig.lock.d`) is a **GPU mutex, nothing more** (Mark, 2026-07-10).

1. **Default iteration (cloud models: Fable/Opus coding, `make test`, git): NEVER touches
   rig.lock.** CPU/disk co-tenancy with the eval rotation is fine; the loop runs 24/7 without
   starving the rotation and vice versa.
2. **GPU-touching steps only** (a sprint whose acceptance includes local-model validation, wire
   diagnostics, anything driving ollama): `rig_lock_acquire wait` for **that step only** — never
   held across a whole sprint. Same discipline as `os-rotation-filler.sh`.

Hygiene: a sprint must not *accidentally* reach the GPU (the port-8080-zombie class). "Does this
step touch the GPU?" is an explicit routing question in the skill, not an accident of what a test
invokes.

## Guardrails (the loop may not…)

- **No releases** by the loop — but a rolling release cadence (Mark, 2026-07-12): the loop lands
  to `dev` continuously and never cuts a release; **Mark snapshots interim releases (v0.30.x,
  v0.31.x…) as needed**, each carrying whatever's accumulated. **v1.0.0 is a MILESTONE declared
  when all five bar clauses are satisfied — not a single big-bang release.** Implications: (1) dev
  must stay release-ready at EVERY commit — the "Dev stays GREEN" guardrail already enforces this
  and it is now load-bearing (any commit may become a release point); (2) each iteration's #329
  report should note when it CLOSES a bar clause (e.g. "clause 3 footgun burn-down: N of M
  landed"), so Mark can watch the bar fill and time the v1.0 call; (3) a version bump mid-mission
  is expected, not a stop signal — the loop already handled v0.29.0/.1/.2 landing between iterations.
- **No pushes without account check** (`gh auth status` → `sunholo-voight-kampff`).
- **No work on a dirty main worktree** — sprints run in coordinator-managed worktrees; the
  controller session itself is read-mostly + doc edits.
- **Budgeted**: hard wall-clock kill in the driver (default 6h); one backlog item per iteration.
- **Kill switch**: `touch ~/.ailang/state/mission-control.disabled` (checked in preflight) or
  `launchctl unload ~/Library/LaunchAgents/dev.ailang.mission-control.plist`.
- **Subscription billing only** (2026-07-10): the nightly bills the Claude subscription via
  `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`, stored in secrets.env) — the driver strips
  `ANTHROPIC_API_KEY` and refuses to start without the token. The first kickstarted run billed
  ~13 min of API credits before this was caught; never again.
- **Escalation**: evaluator `needs-human-review`, merge conflicts, or any guardrail trip →
  `ailang messages send controlplane`, park the item, pick the next; never force through.
- **Skill edits**: max one per iteration, ≥2 recorded frictions each, called out in the morning
  report (git history is the rollback).
- **Dev stays GREEN** (2026-07-10, Mark): an item is not [LANDED] until remote CI passes on its
  merge commit (Gate 3b — local gates miss fmt-check/govulncheck/file-sizes/docs build), and a
  red dev CI outranks the queue at OBSERVE, including time-based reds from newly published vuln
  advisories. **V1 OWNS that red — it is not shared** (2026-08-17). The motoko mission runs the same
  `MISSION_REPO=sunholo-data/ailang` from a separate clone, so it sees every red you see, and the
  skill's "hits whoever observes next" assumes a single observer that does not exist: the overlap
  guard is per-mission (`mission-${MISSION_NAME}.pid`) and there is no cross-mission mutex. When both
  loops fired together they produced [#758](https://github.com/sunholo-data/ailang/pull/758) and
  [#759](https://github.com/sunholo-data/ailang/pull/759) — same red, same six files, duplicated
  end-to-end. motoko's charter now hands anchor reds here instead of fixing them, so **expect those
  hand-offs on the cross-mission channel and treat them as OBSERVE input**; a red nobody picks up
  because each loop assumed the other owned it is the failure mode this creates. Conversely, a red in
  motoko/eval-lane territory stays motoko's — do not adopt one you have no domain knowledge for.
- **BENCHMARK CURATION CYCLES RUN THROUGH THE LOOP, NOT AS ATTENDED SIDE-SESSIONS** (RATIFIED
  2026-08-04, Mark: *"Route curation through mission loop"* — his one-line answer to iteration 140's
  DECISIONS ask). A curation cycle (tier promotion/demotion, retirement, rotation — the operations
  `benchmarks/CURATION.md` governs) is a **queued mission item** from now on, picked and routed like
  any other, never applied by a concurrent attended session. **The evidence is iteration 140's whole
  iteration**: `f574c4b58` (the v0.32.0 curation cycle, run from a non-mission session) moved 12
  benchmarks between tiers and updated **neither** of the two gates that pin the tier distribution
  (`TestAllBenchmarksHaveTierAndTags`, `TestFilterBenchmarksByTier`) — dev CI was red on every commit
  for ~2h, iteration 139 had already misfiled that red as a known runner flake, and v0.33.0 came
  within minutes of shipping on a red dev. The defect was 0.5s-reproducible and the tests document
  their own remedy; nothing about it was hard *except that nobody whose job it was ever saw it*.
  Two consequences: (1) the curator inherits the loop's gates — Gate 2's reality-check, Gate 3b's
  SHA-addressed CI green, and the Gate-4 record — so a tier move cannot land without the
  distribution gates being re-centered in the same commit; (2) `benchmarks/CURATION.md` §5
  *"Applying tier moves — REQUIRED follow-through"* (added iteration 140) stays the operative
  checklist, and this guardrail is what guarantees somebody actually reads it. Attended curation is
  still fine as *authoring* — writing or scoring benchmarks — but the **tier-move commit** goes
  through the loop.
- **A POSITIVE result from ONE confirming instance is not a general claim** (process fix, iteration
  122; the mirror-image of the skill's rule 3a, which covers *empty/negative* readings). Rule 3a
  made this loop good at distrusting silence — an empty `grep`, a failed handshake, a vacuous pass.
  Iteration 122 produced **three** misses of the opposite shape in one run, all from *positive*
  evidence generalised past what it supported: (1) I reproduced a `case`-vs-`match` diagnostic
  cascade first-party with a clean positive control, then inferred it was the *sustained* cause of a
  7-night benchmark failure — refuted by my own per-night scan, which found `case` on exactly one
  night; (2) I wrote `#538`'s headline ("a strictly worse benchmark gets the quieter label") from the
  0/10-vs-1/10 boundary alone, without reading the design rationale that made the asymmetry
  deliberate; (3) I proposed replacement text for the very message whose falseness I had filed the
  bug about, and **my replacement was also false** — and then the correction I posted was wrong on
  mechanism, which only the evaluator's reachability check exposed. The tell is identical each time:
  *one measurement that is true, restated as a claim about all cases.* Before a positive finding
  becomes a general claim — especially one handed to a sub-agent or written into an issue — ask
  **"how many instances is this true of, and did I count them?"** and prefer the cheap census (all N
  nights, all call sites, all reachable shapes) over the single confirming probe. Corollary earned
  the same iteration: **a reachability question is a census question** — "can this path fire?" is
  answered by enumerating the state space (19,607 streams → exactly one escalation pair), never by
  finding one case where it does. Watch-item, not yet a skill edit: needs one more independent
  instance before it earns the one-edit-per-iteration slot.

## Backlog ordering policy

0. **BAR-FIRST (D-28, Mark attended 2026-08-22 — TEMPORARY until all five v1.0 bar clauses
   close)**: items serving an open bar clause outrank everything below, including the cons-cells
   programme. Delete this rule when the bar closes.
1. Open **P0s** first (list above), oldest-known-risk first.
2. **Unblockers** — items other queued items depend on (e.g. m-effect-row-poly-params blocks
   sunholo/demos). **CROSS-MISSION BLOCKERS RANK HERE (`D-31`, Mark attended 2026-08-23), and
   they outrank rule 0's BAR-FIRST clause when a sibling mission's milestone is stopped by them.**
   A sibling's dependency ask is an unblocker whose dependent lives in another repo, which is the
   only reason it was invisible: it never appeared in this queue at all. Issues carrying the
   `cross-mission` label are enumerated at Gate 0 and MAY NOT sit unrouted for more than one
   weekly sweep. Measured at ruling time: **8 open**, and of the ten such asks ever filed, the
   only two that ever carried labels (`#662`, `#656`) are the only two that were ever picked.
3. **P1 by impact-per-day** (the census has estimates; prefer ≤3-day items to keep iterations
   sprint-sized).
4. **Strategic multi-week items enter only after decomposition** into sprint-sized design docs
   (a decomposition is itself a valid iteration deliverable).
5. Anything re-scored `post-v1` in iteration 0 leaves the queue.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

**[RESOLVED 2026-09-06 (iter-333) — motoko's [#1055](https://github.com/sunholo-data/ailang/pull/1055) merged as [`45bbcf625`](https://github.com/sunholo-data/ailang/commit/45bbcf625) at `00:03:40Z`, and V1 iteration 333 verified `dev` afterwards at **15 checks, ZERO not-green**. Iteration 333's own Gate-1 read the three surviving reds first-party before the merge — `launchd drivers (bash 3.2)` dying on `make[1]: go: No such file or directory` in `test-mission-registry` (a deliberately Go-less job that had acquired a `go test` dependency), and both Windows jobs on `internal\mission\kill_unix.go:6:51: undefined: syscall.Kill`, since `_unix` is a build TAG and never a filename suffix — recorded them, and left them alone. HANDED OVER, not duplicated.] `ci-red-mission-loop-workbench` — `dev` was red on SIX jobs, all inherited from the four `m-mission-loop-workbench` commits that landed 2026-09-05 ~19:00–20:30Z.**
Measured at iteration 331. **Attribution is first-party and unambiguous:** the identical failure set is already present on [`6536cfb98`](https://github.com/sunholo-data/ailang/commit/6536cfb98) (workbench M4), the commit immediately BEFORE this iteration's merge, and `a9de67fe6` (workbench M1) was already red on both Windows jobs. Iteration 331's own PR [#1053](https://github.com/sunholo-data/ailang/pull/1053) was green on all five runs and all four required contexts at `ca3b63085` before merging, so none of this is V1's. Failing jobs and their failing STEP (read from `actions/jobs/<id>`, not from the check name): `lint` → *Install and run golangci-lint*; `docs-gate` → *Adjudicate docs build*; `docs-build` → *Build Docusaurus site*; `launchd drivers (bash 3.2)` → *Run launchd driver tests*; `test-windows` → *Run Go test suite*; `Build windows-latest` → *Run tests*. **TWO of the six are already fixed by iteration 331's record PR ([#1054](https://github.com/sunholo-data/ailang/pull/1054)), because both are REQUIRED contexts and the reds would otherwise have stranded the iteration's own record.** (i) `docs-gate` adjudicates `docs-build`, and `docs-build` died on *"Docusaurus found broken links"*: `docs/docs/guides/mission-bootstrap.md:18` links a design doc by a relative path that escapes the docs tree (`../../../design_docs/…`). It is the ONLY such link in `docs/docs` (grep 1 → 0) and the repo's own precedent is an absolute `blob/dev/` URL, which it now uses. **This one is the urgent member of the set**: `docs-gate` is required and fires for any diff touching a docs-relevant path, so a broken link blocks EVERY pull request in the repo, not just the docs mission's. Local verification is partial and stated as such — `make docs-build` clears the design-doc sync and the stdlib-index check and then dies on `docusaurus: command not found` (no `node_modules` in a fresh worktree), so the remote gate is the verifier. (ii) The `lint` red was a single `ineffectual assignment to snap` at `internal/mission/doctor_test.go:229`; iteration 331 fixed it in its record PR `make lint` goes 1 issue → **0 issues** with that one line. **Of the four left, one is characterised and NOT fixed:** `go test ./internal/mission` fails `TestLive_DoctorReproducesTheMeasuredDivergences` — *"GATE: docs env-source-drift not reported (measured as a 4-line divergence)"* — and the negative control fires, i.e. it fails identically at unmodified HEAD, so it is the workbench's own and not a side effect of the lint fix. It reads live rig state, so it may be environment-dependent rather than a code defect; that is the first thing to establish. The launchd-driver and both Windows reds are uncharacterised. **Handoff note:** these commits came from a CONCURRENT attended session, not from this loop and not from a sibling mission loop; `dev` has had no commits since, and no open PR touches `internal/mission`. If that session is still live it may be fixing them, so re-measure before starting rather than assuming the set is unchanged. **UPDATE, iteration 332 — re-measured, and the disposition changed.** Two things happened between fires. (a) **The fix is already in flight and it is NOT ours.** [#1055](https://github.com/sunholo-data/ailang/pull/1055) `sprint/m-mission-workbench-ci-red` opened `22:08:02Z`, six minutes before this fire, `MERGEABLE`, five milestones covering `lint`, `docs-build`/`docs-gate`, `launchd drivers (bash 3.2)`, `test-windows` and `Build windows-latest` — every remaining member of the set. `--author sunholo-voight-kampff` returns it under "your own account" because the whole fleet pushes as one bot, so it was ATTRIBUTED before anything was done with it: `git worktree list` in **motoko's** clone carries `.wt-iter36-ci-red` on that branch, while V1's clone lists **69** worktrees with **zero** matches — disjoint by construction — and the PR body says "Generated by motoko-mission iteration 36". Rebasing, commenting on or duplicating it is the `#758`/`#759` collision, so this row is HANDED OVER, not worked. (b) **A SEVENTH red appeared that #1055 does not cover, and it was the only REQUIRED one**: `test`, from `TestDriverCopiesDoNotMultiply` in the attended session's Phase 3 commit `19d6b03c7`. Required means it blocked every PR in the repo, motoko's fix included, so V1 fixed it forward in `275a72c80` (shipped in [#1056](https://github.com/sunholo-data/ailang/pull/1056)). It is not a stale constant: the probe reads SIBLING directories of the checkout, so it passes from the main checkout at `distinct=2` and fails from every worktree and every CI runner at `distinct=1` **by construction** — it could never have passed in CI. Its own message asks for `knownDriverCopies` to be lowered to 1, which would have greened CI, reddened the main checkout at "grew to 2", and retired the invariant while world's fork is still on disk. It now SKIPS when no sibling is observable, constant untouched, proven non-vacuous three ways (one fork → evaluates and passes; a third → "grew" fires red; none → skip). **Remaining after this iteration: `test-windows`, `Build windows-latest`, `launchd drivers (bash 3.2)` — all three non-required, all three motoko's, all three fixed by #1055.** Resume predicate: `gh pr view 1055 --json state` is `MERGED`, then re-read `commits/<dev-head>/check-runs` and close this row; if #1055 is still open after 2026-09-08, ask motoko on the cross-mission channel rather than fixing it here.

**[LANDED COMPLETE 2026-09-06 (iter-333) — ALL FOUR MILESTONES; doc + sprint plan moved to `design_docs/implemented/v0_35_2/`; issue [#1046](https://github.com/sunholo-data/ailang/issues/1046) CLOSED with the verdict. **M4/4 LANDED 2026-09-06 (iter-333)** ([#1058](https://github.com/sunholo-data/ailang/pull/1058) → [`761b37e64`](https://github.com/sunholo-data/ailang/commit/761b37e64); judge `sonnet` FAIL 68/100 round 1 with TWO blocking, PASS 93/100 round 2 with zero). **M3/4 LANDED 2026-09-05 (iter-332)** ([#1056](https://github.com/sunholo-data/ailang/pull/1056) → [`d14bd42cc`](https://github.com/sunholo-data/ailang/commit/d14bd42cc), judge PASS 96/100). **M1/4 LANDED 2026-09-05 (iter-330)** ([#1051](https://github.com/sunholo-data/ailang/pull/1051) → [`3d7bbfad8`](https://github.com/sunholo-data/ailang/commit/3d7bbfad8), judge PASS 91/100) · **M2/4 LANDED 2026-09-05 (iter-331)** ([#1053](https://github.com/sunholo-data/ailang/pull/1053) → [`f5edd569a`](https://github.com/sunholo-data/ailang/commit/f5edd569a), judge `sonnet` PASS 96/100 round 1, 0 blocking). **`D-55` REMAINS OPEN by design** — its default (a) carried all four milestones and the loop may not resolve its own row; a later (b)/(c) answer would land as follow-up work, not as a revision of what shipped.] `m-compile-cache-unverified-artifacts`** — the compile cache verifies the manifest and never the artifacts it actually executes.
Doc landed at `design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md` with both quorum artifacts; defect reported at [#1046](https://github.com/sunholo-data/ailang/issues/1046). CONFIRMED first-party at HEAD and reproduced end-to-end: source says `99`, the manifest `cache_key` is the CORRECT key for the `99` source, the blobs are from the `42` compile, and `ailang run` prints **42** with no diagnostic. Negative control — ordinary edit-and-re-run invalidation works (42 → 99). Mechanism: `LoadArtifacts(moduleID)` (`cache_store.go:185`) receives no key and verifies nothing (`grep -n CacheKey cache_store.go` = exactly 2 lines, neither in the artifact path; control `modDir` = 11), while `pipeline_module.go:369` writes the manifest entry unconditionally and `:377` discards the artifact-write error. **Iteration 329: the pre-registered default fired.** `D-55` was unanswered at the next iteration, which is the condition its own row named, so option (a) was applied as a **controller routing call, not a ruling** — the loop may not resolve a ledger row on its own behalf, so `D-55` is still OPEN. Sprint plan at `design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts-sprint-plan.md` (planner `codex:gpt-5.6-sol`; the machine-readable companion is `.ailang/state/sprints/sprint_M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS.json`, which is deliberately UNTRACKED because `.gitignore:82` ignores `.ailang/` — the markdown is the decision-bearing artifact). Milestones unchanged at M1 2d / M2 0.75d / M3 0.5d / M4 0.75d. **The acceptance gate was proven non-vacuous by the controller OUTSIDE the sandbox** at `137842bfd`: M1 returns `go returncode=0` with **zero** selected passes — Go's vacuous zero-selected-tests success, which the gate's `assert set(names) <= passed` correctly rejects — failing on missing test names, not a compile or invocation error; positive control `TestAliasPolyE2E_RecordSingleModule` reports a pass, so the instrument can see one. **M1 LANDED (iter-330).** `9cb3e711b` (milestone) + `726dd1866` (judge findings) → squash `3d7bbfad8`. Artifact loads verify a v4 stamp binding the exact module ID, the caller-computed EXPECTED cache key and SHA-256 for all four payloads; blobs read once under the design's ceilings and decoded only after every hash passes; stamp written last, manifest entry after the artifacts, optional-persistence failures on stderr. **Every gate re-run OUTSIDE the executor sandbox**: the cumulative gate prints six `M1 PASS` lines here and fails at the parent with all six names missing and `go returncode=0` (Go's vacuous zero-selected success, which the gate correctly rejects) — non-vacuous in both directions; whole-package `internal/pipeline` `ok` at both commits; `internal/loader` and `cmd/ailang` `ok`; fmt/vet/file-sizes clean (`pipeline_module.go` **776** of 800). End-to-end smoke: cold `42`, warm `42` from cache, `99` after edit. **The judge's two findings are the substance**: anchoring its mutation set to the DIFF rather than to the plan's table (rule 3n), it found the write-side aggregate module ceiling and the `mkdirAll` guard each survived deletion with the whole package GREEN — a milestone whose purpose is *"stop swallowing the artifact-write error"* had shipped a new swallowed-error path of its own. Both reproduced first-party, both fixed, and each new arm proven the SOLE killer of its mutation with byte-identical `shasum` restore. Issue #1046 deliberately stays OPEN: M1 is one of four. **M2 LANDED (iter-331).** `41320c8ff` (milestone) + `089ae50f2` (judge finding) + `ca3b63085` (CI red) → squash `f5edd569a`. The loader retains the exact lexer source as an immutable `SourceContent *string` (the POINTER distinguishes a known-empty source from an unavailable one; the embedded `std.FS` fallback flows through the same buffer), and the pipeline hashes `*mod.SourceContent` — the opportunistic `os.ReadFile` and its `""` default are gone. A module with no snapshot bypasses BOTH lookup and publication, emits `CACHE_SOURCE_UNAVAILABLE`, and is never hashed as `""`. Cumulative gate: ten `PASS` lines here, `('M2', 'missing or skipped', {'TestCacheSource_ExactSnapshot'})` at base — non-vacuous both ways; both single-test acceptance commands rc=0 AT BASE, i.e. regression controls not new proof. `internal/loader`·`internal/pipeline`·`cmd/ailang`·`internal/apiserver` all `ok`; end-to-end cold `42` → warm `42` → edit `99` → warm `99`. **The judge's finding (rule 3n, anchored to the DIFF):** deleting `&& moduleCacheKey != ""` from the publication guard left the WHOLE `internal/pipeline` package green — reproduced first-party. Safe, but one layer lower than the design says: a nil-source module DOES reach `cacheRuntime.publish` and only M1's empty-key rejection stops it, emitting `CACHE_WRITE_FAILED`. One arm now pins *never ATTEMPTS* rather than *fails to*, proven the sole killer with byte-identical restore. **The CI red was a gate my own list never contained:** `make check-home-isolation` (remote-only) refused T7's hand-rolled `HOME` override, because `os.UserHomeDir` reads `USERPROFILE`/`$home` elsewhere — rule 3g, and baselining cannot see a gate you never listed. Eight further `ci.yml`-derived gates run and green. Issue #1046 stays OPEN: M2 is two of four. **M3 LANDED (iter-332).** `a92fa666b` (milestone) + `672eb1c0f` (judge finding) + `275a72c80` (an inherited required-context red, unrelated to this sprint) → squash [`d14bd42cc`](https://github.com/sunholo-data/ailang/commit/d14bd42cc) via [#1056](https://github.com/sunholo-data/ailang/pull/1056). `Clear()` saves an empty v4 manifest AND removes `<cs.dir>/modules`, contextual error on either half, so the CLI cannot print success over a failed deletion; removal rides a new injectable `removeAll` seam on `cacheArtifactIO` beside the existing `writeManifest` one. Deletion is narrow — `manifest.json`, root siblings, package caches and another session's override all survive, under both root variants. End-to-end with a branch-built binary: 20 blobs → `Cleared 4 cached compilation entries` → subtree removed (V25 measured it SURVIVING at HEAD), sentinel preserved, manifest `0`, re-run correct. Cumulative gate: twelve PASS lines here, `('M3', 'missing or skipped', {'TestCacheStore_ClearArtifacts'})` at base — non-vacuous both ways. Nine further gates DERIVED from `ci.yml` (after iter-331's `check-home-isolation` miss) all rc=0. The executor's `cmd/ailang` red was the sandbox loopback-bind denial: rc=0, zero FAILs outside it. **Judge `sonnet` PASS 96/100 round 1, ZERO blocking**, and its diff-anchored finding (rule 3n, third milestone running) is the substance: swapping the two halves of `Clear()` left the WHOLE package green plus T10 — reproduced first-party, the shipped order is correct but load-bearing (the manifest AUTHORISES the artifacts, so a clear that cannot record itself must not already have destroyed them), and the `save failure` subtest now requires the subtree to survive a failed save, **proven the SOLE killer** with byte-identical restore. Issue #1046 stays OPEN: M3 is three of four. **Resume predicate: execute M4** (route integrity diagnostic and MCP regression, 0.75 d) — the last milestone; `#1046` closes with it.

**[PARKED — needs-human-review, D-58, iteration337] `m-pi-runner-worktree-assertion-vacuous-on-revision` — `mission_pi_run.sh` reports verdict `ok` for a run that wrote nothing, whenever the target file was already dirty or untracked before the run.**
Iteration337 confirmed dirty/no-op and untracked/no-op both return rc0/ok; clean/no-op returns rc10. Comparing porcelain before/after is insufficient: a second content edit can preserve status/name. DeepSeek authored a bounded content-snapshot design and one revision; both complete quorum rounds rejected (all3 present). Concrete external fixes are banked, but failure-ordering and artifact-alias interactions still need judgment, so the verbatim-only narrow-refinement carve-out was not exercised. **Resume: D-58 ruling → fresh designer/quorum gate → planner → executor → independent evaluator.** No implementation or sprint plan exists. Design: `planned/v0_35_2/m-pi-runner-worktree-assertion-vacuous-on-revision.md`; rejected prose is retained with authoritative controller corrections.

**[PARKED — needs-human-review, D-57, iteration336] `m-cache-module-id-encoding` absorbs `m-cache-sanitize-module-id-windows-colon` and `m-cache-sanitize-module-id-collision`.** Design and inherited plan exist, but **0/4 milestones executed**. Both iteration336 quorum rounds BLOCKED (2 rejects/1 pass, none absent); GLM disputes hybrid vs pure-hash/basename direction. **Resume: human D-57 ruling → design gate → planner resynchronizes blocked plan and snapshot → M1. No runtime copy while blocked.** M3/M4 remain regression/platform work with no new production mutation of their own.
Current evidence: the old separator-only mapping preserves a drive colon, which is illegal as a Windows path component, and maps `a/b` and `a__b` to the same directory. The previous claimed Windows diagnostic is **reconstructed, not a captured CI transcript**; the fetched successful Windows job log contains no such diagnostic. Actual publication proof remains future M3 work. The exact-module-ID stamp backstop already rejects wrong-ID artifacts and recompiles, but directory contention can still cause churn. Historical iteration333–335 observations remain in the mission log; the current corrected evidence and all16 measured examples are in `planned/v0_36_0/m-cache-module-id-encoding.md`.

**[LANDED iter-338 — PR [#1063](https://github.com/sunholo-data/ailang/pull/1063) → [`0b7f3e3af`](https://github.com/sunholo-data/ailang/commit/0b7f3e3af); inherited post-merge red fixed forward by PR [#1064](https://github.com/sunholo-data/ailang/pull/1064) → [`b50bb366e`](https://github.com/sunholo-data/ailang/commit/b50bb366e); MiniMax R1 PASS92, R2 PASS93, R3 PASS95, zero hard failures] [iter-332; 2 instances, both first-party, one recorded by the PREVIOUS iteration] `m-gate1-shared-clone-ref-drift` — every mission on this rig shares one clone's `.git`, so `origin/dev` advances when a SIBLING fetches, and Gate 1's sync verdict silently expires.**
Gate 1 opens `git fetch origin; git rev-parse dev origin/dev` and every later gate treats that SHA as the base. That is a point-in-time reading of a ref **other processes mutate**, and nothing in the skill says so — the Gate-4 rule that re-confirms the base before writing exists, but it is scoped to the charter write and reads as protection against a stale LOCAL checkout, not against a moving REMOTE one. Instance 1 (iter-331, in its own stamp): dev went red mid-iteration from a concurrent attended session, found only because the red stranded the record — *"Gate 1's health check runs once, at the start, and the iteration that does real work is exactly the one during which the tree can change underneath it."* Instance 2 (iter-332): Gate 1 read `origin/dev` = `95e4d20c8`; a `git worktree add … origin/dev` issued minutes later came up at **`19d6b03c7`**, four commits ahead, with no fetch of mine in between — the attended session's fetch had advanced my ref. Benign both times only because the surprise was visible. The general shape is the skill's own: **a ref is a claim about when you read it, not about what it points at now**, and it is the same class as the `MISSION_WORKDIR` and `readlink` notes one level down. **Candidate fix (cheap):** re-read `git rev-parse origin/dev` immediately before creating any worktree and before Gate 3b, and record the base SHA *with its read time* wherever a gate quotes one — so a later disagreement reads as drift rather than as an error. **Why it is a queue row and not this iteration's skill edit:** the context-docs ratchet baseline for `SKILL.md` is **2781**, exactly its current size, so any addition must first pay for its space by restructuring — and iteration 329's restructure attempt tripped a two-day-old grep guard in `tools/launchd/test_mission_routing.sh`. Doing that unattended, in a repo with three concurrent writers, is a worse trade than a row with the measurement attached.

**[NEXT] [iter-337, measured pre-existing] `m-pi-runner-shell-suite-coverage`** — existing runner suite is not called by Make/CI, and its reasoning-stall timing arm passed only8/9 on the planner's default baseline versus9/9 with controller POLL1. Scope the coverage wiring and deterministic timing separately; do not treat an unwired local suite as remote proof or fold this inherited defect into D-58.

**[NEXT] [iter-336; first recorded session-protocol instance, no skill edit spent] `m-pi-evaluator-session-handshake`** — the independent pi evaluator was told to reuse controller inbox triage, but the loaded session protocol requires a CLAUDE read and an `ailang messages` command in the SAME session before `session_protocol_ack`. Repeated denied writes produced no report (446s,226tools,rc10 empty_worktree after verified-process termination). Same MiniMax model succeeded after an explicit bounded inbox list and local acknowledgement; no guard removed. Record this handshake in the evaluator recipe if the same friction recurs; supplied controller triage does not satisfy the per-session guard. Raw receipt and successful invocation metadata: `docs/sprint-retros/iter336-cache-module-id-encoding-transport.json`.

**[NEXT] [iter-331, measured first-party on PR [#1053](https://github.com/sunholo-data/ailang/pull/1053); NEW and MINE, not inherited] `m-cachesrc-cognitive-complexity` — the SonarCloud new-code maintainability gate is red on the M2 diff, and it will stay red for M3 and M4.**
`new_maintainability_rating` **4** against a threshold of 1; every other condition passes (`new_coverage` 100.0, 0 bugs, 0 vulns, 0 duplication, hotspots 100% reviewed). Six `go:S3776` cognitive-complexity smells: four on the new test functions (`cache_invalidation_test.go:22/145/195` at 32/19/28, `loader_test.go:160` at 16) and one MINOR `godre:S8193`, plus `pipeline_module.go:31` — `runModuleWithCacheDependencies` at **112** against 15. **Negative control fires the right way:** M1's PR `#1051` reads `pass` on the same check and the `dev` branch gate reads `OK`, so this is attributable to the M2 diff and is not a standing red. It is NON-REQUIRED, so `UNSTABLE` is not `BLOCKED` and it did not gate the merge. Filed rather than fixed in-iteration because the only structural fix is extracting control flow out of `runModuleWithCacheDependencies` — a production refactor the judge has not seen, in the exact function M3 and M4 both re-enter, which is a worse trade than a named row. **Note the consequence for the next two milestones:** they touch the same function, so they inherit this red and cannot tell it from one of their own unless they read this row. Pick it up with M4, or as its own row once the sprint is out of that file.

**[NEXT] [iter-330, Gate-0 triage; HALF measured first-party — the asymmetry is the finding, the cause is not yet established] `m-coordinator-codex-401` — a coordinator-dispatched codex task 401s on the websocket endpoint while this loop's own `codex exec` lane is rc=0 in the same window.**
Inbox row `inbox_1788607698893_969e1ac7`, `sprint-planner`, `2026-09-05T11:28:18Z`: *"executor failed: codex task failed: exit status 1"* with **seven** repetitions of `ERROR codex_api::endpoint::responses_websocket: failed to connect to websocket: HTTP error: 401 Unauthorized, url: wss://api.openai.com/v1/responses`. Measured first-party in the same iteration, ~20 minutes later: this loop's `codex exec --model gpt-5.6-sol` probe returned **rc=0** and the real 30-minute sandboxed executor run completed **rc=0** with a full milestone on disk. So the two paths differ — the coordinator reaches OpenAI over an API-key websocket, `codex exec` over OAuth — and the coordinator's is dead while ours is healthy. **What is NOT established and must be measured at pick time (ghost discipline):** whether this is a stale/absent `OPENAI_API_KEY` in the coordinator's environment, a revoked key, an endpoint change, or a per-task auth-forwarding bug; and how many dispatched tasks it has silently failed. Note the consequence is the dangerous kind — a dispatched coordinator task that 401s reports `failed` into an inbox nobody blocks on, so the work simply does not happen. Related: `project_feedback_dispatch_never_worked`, and the message-plane rows already queued.

**[NEXT] [iter-328, the doc's own out-of-scope list plus the round-2 objection; PRE-EXISTING at HEAD, not introduced by anything this iteration touched] `m-cache-artifact-adversarial-decode`** — hash-verification proves consistency, not provenance, and `encoding/gob` is documented as not hardened for hostile input.
This is the substance of `gpt5-6-sol`'s twice-repeated objection, filed here on its own first-party evidence rather than left to block a strict improvement. Measured at HEAD: `LoadArtifacts` gob-decodes with **0** byte ceilings and **0** hash checks. Judge-verified counterweight: Go's gob decoder already caps preallocation against remaining input, so a size-capped blob cannot be amplified arbitrarily. Ranks below the correctness fix by construction — it is a hardening item, and HEAD is strictly worse than the parked doc's design on every axis it names.

**[ABSORBED into PARKED `m-cache-module-id-encoding`, D-57, iteration336] `m-cache-sanitize-module-id-collision`** — old `a/b` and `a__b` map to one artifact directory. The proposed bounded hash is collision-resistant, **not injective**; it separates the measured fixtures. No implementation has landed; do not pick this separately.
Verified at `cache_store.go:239-251` and independently by the iteration-328 judge. Two distinct manifest entries can therefore share one artifact directory and evict each other. Under the parked doc's design this is a correctness NON-issue (the stamp's exact `module_id` comparison prevents cross-module consumption) but remains a storage/recompilation defect. Do not pick this before `m-compile-cache-unverified-artifacts` lands — the design's stamp is what makes it merely a performance bug.

**[NEXT] [iter-327, surfaced by the round-1 judge while fixing its own blocking finding; CONFIRMED first-party at HEAD; a defect in a SHARED gate, so it is deliberately NOT bundled into the sprint that found it] `m-gate-wiring-classifier-prefix-blind` — `TestGateTargetsAreWiredIntoAWorkflow` cannot see a gate whose name does not start `check-` or `test-check-`.**
`internal/cihygiene/gate_wiring_test.go:130` classifies a gate target with `strings.HasPrefix(target, "check-") || strings.HasPrefix(target, "test-check-")`. That is the whole enumerator. So `fmt-check`, `shellcheck-autopush`, `test-fmt-check` and `test-shellcheck-autopush` are **invisible to it by construction** — and iteration 327 shipped two new self-tests into the `ci:` aggregate with no CI step, which is precisely what this gate exists to prevent, while the gate stayed green. Measured at HEAD: **0** invocations of `make ci` across all workflows (control `make fmt-check` = 1), and ci.yml:228 states the rule in the repo's own words (*"Adding a target to `ci:` is necessary and NOT sufficient"*). The two rows were wired by hand in `cacd35103` and are now pinned by `scripts/test_shellcheck_autopush.sh`, so the immediate hole is closed; **this row is the systemic half** (CLAUDE.md principle 3). Scope when picked: widening the prefix rule will light up other targets — enumerate what it catches BEFORE changing it, and expect the answer to be a list of real gaps rather than a clean run. Same hand-picked-subset shape as rule 3g, one level down: the gate that checks wiring has its own hand-maintained notion of what counts as a gate.

**[NEXT] [iter-327, round-1 judge finding N2; PRE-EXISTING at `2b5750ad9`, not introduced by the sprint — but the lines were rewritten by M3, so they are in its diff] `m-autopush-hook-failure-branches-untested` — two failure guards inside the auto-push hook's fmt loop survive deletion with the suite at 26/26.**
Judge-measured, both directions, with `shasum` restore: removing the rc check on `bounded 10 git show "dev:$go_file"` and, separately, removing the rc check on `bounded 10 gofmt < "$FMT_TMP/committed"` each leave the harness at **26 passed, 0 failed**. Confirmed pre-existing via `git show 2b5750ad9:scripts/hooks/push_dev_on_stop.sh`. The malformed-blob case has *incidental* cover — gofmt emits empty stdout on a parse failure, so `cmp` catches the difference against a non-empty committed file — but that is a side effect, not a designed arm, and the `git show`-failure branch has no cover at all. Both are fail-safe in direction (they strand, never corrupt), which is why this is a queue row and not a blocker. Scope when picked: one arm each, driving the hook against a bare-origin fixture where `git show` genuinely fails, per the existing arm convention.

**[NEXT] [iter-327, first-party; the two instruments the loop routes on disagree by construction on any doc-less pick] `m-resolver-hook-disagree-on-docless-pick`
**INSTANCE 2 — iter-328, and it is MUCH broader than "doc-less".** With a real design doc in hand the disagreement still fires, for a different reason, and the control proves it is not about my doc: `derive-planner-lane.sh <doc>` returns `opus fail-closed:planner-lane-field-missing` for the picked doc AND for `design_docs/v1-mission.md` itself, because the script requires a `planner_lane` field that only **2** design docs in the entire repo carry (`m-feature-provenance-chains`, `m-mission-elo-routing`). So for essentially every real pick the resolver says `opus` and the spawn-pin hook then DENIES opus, since planner is pinned to `codex:gpt-5.6-sol`. A third arm: run from the driver's own CWD (the pin worktree) against a doc authored in the sprint worktree and it fails one step earlier still — `design document is missing or unreadable` — because the path is resolved relative to CWD, which is this file's own recurring shape (*a relative path is a claim about where you are standing*). The fix belongs in the TOOL, not in the skill: either default the planner lane when the field is absent, or stop requiring a field almost no doc has.
**INSTANCE 3 — iter-329, same mechanism, reproduced exactly.** `resolve-role-spawn.sh planner <doc>` returned `agent-tool opus fail-closed:planner-lane-field-missing` for a pick WITH a complete design doc, and the spawn-pin hook denied that spawn verbatim as in instance 1. The controller followed the hook and the planner ran on `codex:gpt-5.6-sol` as configured — the right lane, reached by being denied rather than by routing. Three instances clears the Gate-5 skill-edit bar (≥2) and the routing-policy bar (≥3); iteration 329 spent its one skill edit on the SKILL side of the pair (Gate 3 Step 1b now tells the controller that a `fail-closed:*` answer beneath a `provider:model` role pin will be denied, and to route to the pin). **That does not close this row**: the durable fix is still the TOOL, exactly as iteration 328 said, and until it lands the resolver keeps emitting an answer the hook refuses.
 — `resolve-role-spawn.sh planner` with no design-doc argument returns an alias the spawn-pin hook will always deny.**
Measured this iteration: `tools/launchd/resolve-role-spawn.sh planner` (no doc) returned `agent-tool opus fail-closed:no-doc`; spawning exactly that was **DENIED** by the PreToolUse hook with `deny:provider-pin — planner is pinned to codex:gpt-5.6-sol; Agent-tool alias spawn refused — use the cross-provider recipe (resolve-role-spawn.sh planner)`. Note the denial message tells you to run the very command that produced the refused answer. The hook was RIGHT and the iteration followed it, so nothing was lost — but Gate 3 says of the resolver *"do not second-guess it"*, and a controller who obeys that on a doc-less pick burns a spawn on a guaranteed denial. Both behaviours are individually defensible; the pair is not. Scope when picked: decide whether the resolver should consult `MISSION_<ROLE>_MODEL` before falling back to `fail-closed:no-doc`, or whether the hook should permit the resolver's own fail-closed answer — then make the two agree, and say which is authoritative in the skill.

**[LANDED 2026-09-04 (iter-327) — ALL FIVE, PR [#1044](https://github.com/sunholo-data/ailang/pull/1044) → squash [`da2b6689b`](https://github.com/sunholo-data/ailang/commit/da2b6689b); **20 checks, 0 pending, 0 failures**, all four required contexts green AND SonarCloud green on the PR; judge `sonnet` PASS 85/100, round 1, one blocking finding fixed in-iteration. Plan at `design_docs/planned/v0_36_0/m-autopush-gate-followups-sprint-plan.md`. M1 `c717ee8a6` (harness HOME + 8 SC2164) · M2 `9bf9c5db2` (both fmt gates read gofmt's rc) · M3 `d08352176` (NUL pathnames) · M4 `62d7aad4b` (SKIP_ROOT/SKIP_NOT_GIT) · M5 `2eb6e25e4` (scoped shellcheck gate) · `8547d0dab` (CI `timeout-minutes`) · `cacd35103` (judge finding B1). Harness 18 → **26** arms. **The cost line, recorded because it is the finding: M1's `caller sentinel` test row, as the plan specified it, seeded a write into the REAL shared `~/.ailang/state/autopush.log` and truncated it from 92 lines to 1 — unrecoverable.** The write is denied under the codex sandbox, so the executor could not see it; only the controller's out-of-sandbox re-run revealed it. Arm is now read-only, propagated into all five snapshots.] [FIVE FINDINGS FROM THE ITER-326 JUDGE, ALL NON-BLOCKING, ALL MEASURED] `m-autopush-gate-followups` — the committed-Go formatting gate on the dev auto-push works, and here is everything it does not cover.**
Landed at `d2ef77e09`; judge `sonnet` PASS 86/100, zero blocking, and these are its non-blocking findings, each reproduced first-party before being written down. **(a) A zero-byte `.go` file passes both the gate AND `make fmt-check`.** `gofmt -l empty.go` is rc=2 with the error on *stderr*; `gofmt < empty.go` is rc=0 with empty stdout. The gate feeds gofmt on stdin so it reads clean — but `make/code-health.mk:19` tests `[ -n "$(gofmt -l .)" ]`, **stdout only, never gofmt's exit code**, so CI's own gate has the identical blind spot. Narrowed from the judge's filing: a NON-empty unparseable file returns rc=2 both ways and is correctly refused. Fix both together or neither; the gate must not be *stronger* than the CI gate it mirrors or it starts refusing pushes CI would accept. **(b) A filename needing git C-quoting causes a FALSE REFUSAL of well-formatted code.** `git diff --name-only` quotes/escapes paths containing a quote, a backslash, or (with `core.quotepath` on, the default) any non-ASCII byte, so `café.go` comes back as the literal `"caf\303\251.go"`, `git show dev:<that>` 404s, and the failure is classed as unformatted. Fail-safe direction (strands, never corrupts) and no such filenames exist in this repo today; a space in a filename works fine. **(c) The hook's two EARLIEST guards exit silently** — `cd "$ROOT" || exit 0` and `git rev-parse --git-dir || exit 0` produce no log line and no output, which is inconsistent with the hook's own stated rationale (every downstream refusal was deliberately made loud) and is undiagnosable from the shared log. **(d) `shellcheck` is wired into NOTHING** — `grep -rn shellcheck .github/workflows/ make/ Makefile` = 0 hits, control firing on `gofmt`. So "the hook is shellcheck-clean" is a one-time claim nothing regression-tests, on a script that is now a fleet-wide integrity gate. **(e) `scripts/hooks/test_push_dev_on_stop.sh` pollutes the REAL shared `~/.ailang/state/autopush.log`** because it does not override `$HOME`; the iter-326 executor's runs left four synthetic `[local] pushed …`/`[local] REFUSED …` rows a later iteration could read as fleet evidence. The harness should set its own temp HOME — that one is cheap and should probably go first.

**[NEXT] [DEBT WITH A NAMED OWNER — V1 iter-325 loosened a ratchet it did not break, and owes the real fix] `m-release-manager-skill-split` — move the 18-image pipeline walkthrough out of `.claude/skills/release-manager/SKILL.md` and ratchet its context-docs baseline back DOWN from 625 to 596.**
Commit `5abe0411d` (2026-09-03 22:05, fleet account, +52/−23) added the by-version tag build and its stop-at-test gate to that skill, growing it 596 → 625 and turning **dev RED** on `check-context-docs` for every mission on this rig within minutes. V1 owns this repo, so per Gate 1 the red outranked the queue; iteration 325 fixed it forward with the escape valve the gate itself names — a baseline bump carrying a why — because that is the smallest change that unblocks four loops, and because surgery on another agent's freshly-landed document, blind to a series still in flight (S6, S8 landed; S7/S9 presumably coming), is the more dangerous option. **It is not the right fix.** `scripts/context_docs_baseline.txt`'s own header states it: retiring an entry means moving its war stories, tables and worked examples into sibling files that SKILL.md links to, not deleting them, and a bumped ratchet re-authorises the next regression silently. Whoever picks this does the split and the down-ratchet together — the gate refuses to keep an entry once the doc drops back under its cap, so the down-ratchet is forced rather than optional. **Also record what the red revealed a second time:** step 40 of the `test` job failing left steps 41–61 — **21 gates** — reading `skipped`, the exact masking shape iteration 323 measured at 45 gates. That is already the queue row `m-ci-serial-gate-masking`; this is its second independent instance in three days.

**[NEXT] [PRE-REGISTERED — instance 1 of 2 recorded; the SECOND instance is the Gate-5 skill-edit trigger, do not spend the budget before then] `m-acceptance-criterion-green-at-base` — rule 3e(a) catches a gate RED at base and is silent on one that is GREEN at base and GREEN after, which measures nothing.**
Instance 1, V1 iteration 325, first-party and judge-confirmed: `m-spawn-pin-enforcement`'s A3.2 (`MISSION_DRY_RUN=1 bash tools/launchd/mission-control.sh` → rc=0) was the plan's ONLY criterion aimed at M3's production code running, and it cannot reach that code on any path — the dry-run branch `exit 0`s at line 858, the Layer-3 block starts at 1008. Proven by running the criterion against the BASE copy of the script for byte-identical output. Rule 3e(a) reads "a gate ALREADY RED at base measures the repo, not your change" and its whole remedy is "delete or repair anything already red" — so a controller who follows it exactly, as I did, baselines A3.2, sees GREEN, records "must stay green", and learns nothing. The symmetric half is: **a criterion whose reading is identical at base and after is not a weak gate, it is not a gate**, and the cheap discriminator is one command — run it against the base artifact, not just on the base tree. Note what made this recoverable rather than silent: arm D2 covered the behaviour independently, so the milestone was genuinely verified and only the criterion was dead. When instance 2 arrives, the fix is one clause in rule 3e(a) (baseline for DISCRIMINATION, not merely for redness), not a new rule.

**[NEXT] [WORLD-DEMAND — `mission-world` iter-152, triaged and CONFIRMED first-party by V1 iter-325] `m-messages-send-type-misfiled` — `ailang messages send --type` binds to `Category`, not `MessageType`.**
Measured at HEAD: `cmd/ailang/messages_send.go:42` declares the flag as "Message category" and assigns it to `Category`, while `:132` hardcodes `MessageType: messaging.InboxTypeNotification`. So every mission-authored approval row this skill's Gate-5 snippet sends carries `type=notification, category=approval_request`, while coordinator-authored rows carry the reverse. Consequences World measured and V1 has not re-verified beyond the binding itself: `ailang messages activity` aggregates `ByType` and therefore reports ZERO `approval_request` while an approvals row is live; `agent_registry.go` routes inbound templates by `TemplateByMessageType`, so a mission approval can never match an `approval_request` template. NOT a broken human channel — the Discord push switches on `ToInbox`, so approvals still reach Mark. Scope when picked: decide whether `--type` should set `MessageType`, `Category`, or both (a rename is the cheap honest option), then re-check whether this skill's Gate-5 approvals snippet needs to change in the same breath — it is the fleet-wide producer of the wrong type. Sibling issues already filed by World: `#984` (corrected), `#1036`, `#1037`.

**[LANDED — ALL 4 MILESTONES. M1+M2 2026-09-03 (iter-324), PR [#1038](https://github.com/sunholo-data/ailang/pull/1038) → `70e453060`; M3+M4 2026-09-03 (iter-325), `11aff5819` + `e21c3f1bd` — the hook is now ARMED: the driver exports `MISSION_CONTROL_ACTIVE=1` post-degradation, so an Agent/Task spawn with no `MISSION-ROLE:` line is DENIED at the tool boundary. Doc + plan moved to `design_docs/implemented/v0_35_0/`.] [Mark, attended 2026-09-01 — design APPROVED by Mark in session; fleet infra, this loop owns the shared skill] `m-spawn-pin-enforcement` — a `provider:model`-pinned role can be silently spawned on an Anthropic alias (sonnet/opus) because the Agent tool's enum (`sonnet|opus|haiku|fable`) cannot carry a `provider:model` pin, and the skill's advisory text about that limitation has been skipped twice — measured.**
Two opus burns with HEALTHY cheaper lanes: (1) docs-9 orphaned fire — executor declared `codex:gpt-5.6-luna` (driver probe `lanes=ok`) but the in-session Agent tool degraded it to the session alias `sonnet`, and the evaluator was then re-routed to **opus** to preserve generator≠judge (PR #973 body, FLAGGED); (2) docs-10 (PR #1010) — planner AND executor pinned **opus** via the Agent tool, again FLAGGED, and the planner would have been opus-required anyway (`scripts/*` missing from the docs mission's `MISSION_PLANNER_ALLOWLIST`). Known mechanism, documented but not enforced: SKILL.md:2561 records the alias-only enum; AGENTS.md itself says "advisory text gets skipped — this has been measured". Per Critical Principle 2 a fallback affecting cost must not be silent, so enforcement moves from prose to code: a `tools/launchd/resolve-role-spawn.sh` resolver (mirror of `derive-planner-lane.sh`: pin → `recipe <provider>` VERBATIM, alias → agent-tool + generator≠judge assertion, missing → `refuse fail-closed:*`), a PreToolUse hook that hard-denies contradicting Agent spawns, driver export of the resolved per-role plan, the docs mission's `scripts/*` allowlist widening, and `test_mission_routing.sh` controls replaying both incidents. The opus tail itself STAYS (designed last resort) — the bug is the shortcut past healthy rungs. **Full design: [m-spawn-pin-enforcement.md](m-spawn-pin-enforcement.md) (Mark approved attended 2026-09-01).** Skill-text-only fix explicitly REJECTED in the design. Note the working tree currently holds this loop's own uncommitted Gate-5 edits (iter-311 quorum-path fix, iter-314 mutation-nondeterminism rule) — landed separately first; do not fold them into this sprint. Issue: [#1012](https://github.com/sunholo-data/ailang/issues/1012) (cross-mission; cross-ref #493, #902).
**M1+M2 LANDED (iter-324).** `tools/launchd/resolve-role-spawn.sh` (planner consumes `derive-planner-lane.sh` verbatim) and `tools/launchd/spawn-pin-hook.sh` wired as a SECOND `Agent|Task` PreToolUse entry: while `MISSION_CONTROL_ACTIVE=1`, a role is declared by a first-line `MISSION-ROLE: <role>` token (Explore is the read-only exception), a `provider:model`-pinned role is denied on ANY alias, unset pins / unknown roles / unparsable payloads / generator=judge are denied; marker absent → the hook prints NO decision. Quorum 2 rounds (3/3 reject each; carve-out applied, every premise measured), judge sonnet **FAIL 66 → PASS 92**, its blocking F1 (explicit allow on passthrough) fixed. **Inert until M3** — nothing exports the marker yet (`grep -c MISSION_CONTROL_ACTIVE mission-control.sh` = 0). Plan: [m-spawn-pin-enforcement-sprint-plan.md](m-spawn-pin-enforcement-sprint-plan.md). **When M3 lands, every controller Agent spawn in a mission session must carry the token or be Explore — the running skill (main checkout) will not say so until M4 lands AND the main checkout is reconciled; the denial reason names the fix.**


**[NEXT &mdash; iter-323, the systemic finding behind a 24-hour red] `m-ci-serial-gate-masking`** &mdash; a red that fails EARLY in a long ordered CI job silently suspends every gate behind it, and the check set then reports ONE failure where there are several. Measured 2026-09-03: `check-file-sizes` is **step 15** of the `test` job and the job died at **step 11** for ~24h, so steps 12&ndash;60 &mdash; **45 gates** &mdash; read `skipped`. Two files crossed the 800-line limit inside that window (`backend_gcp.go` 788&rarr;811, `inbox.go` 773&rarr;850, both under the limit and step 15 `success` at the last green `7668ed9df`) and nothing could see it. `SonarCloud` reads `none` on every dev commit in the window for the same reason (it is step 58). The identical mechanism operates twice more with different clothes: within the `lint` job's step list (a `golangci-lint unused` red sat behind an `fmt-check` red), and across the build matrix via fail-fast (`Build windows-latest` reads `cancelled` on every recent dev commit, hiding a 21-test failure). **This is not a bug in any gate; it is the job SHAPE.** Candidate remedies, none costed yet: split the pure gates out of `test` into their own job(s) so a Go test failure cannot mask them; or `continue-on-error` on independent gate steps with a final aggregating step; or `fail-fast: false` on the build matrix. Wants a design doc &mdash; the trade-off is CI minutes against observability, and the answer changes which gates are "required".

**[NEXT &mdash; iter-323, a debt with a named owner and a trigger] `m1b-nolint-suppression-owed`** &mdash; `cmd/ailang/coordinator_cloud_evidence.go`'s `diffResultFromEvidence` carries a `//nolint:unused` added 2026-09-03 to clear a required-context `lint` red. It is NOT dead code: its consumer is **M1b of M-COMPLETION-PATH-PARITY**, which that sprint's plan records as deliberately outstanding. **The suppression must come off WITH M1b, not survive it.** No gate enforces that today, which is exactly how a temporary suppression becomes permanent. Cheap remedy: a check that every `//nolint` naming a milestone fails once that milestone is marked landed. Filed rather than fixed because it is a gate-design question, not a one-liner, and because the owning workstream may prefer to land M1b first.

**[LANDED 2026-09-02 (iter-320) &mdash; PR [#1025](https://github.com/sunholo-data/ailang/pull/1025); `test-windows` AND `Build windows-latest` both `success` on the PR head, which is the only instrument that can verify this fix. Evaluator PASS **95/100** round 2, zero blocking, after **86/100** round 1 with one BLOCKING that the judge found and I could not have] `m-home-isolation-windows-red` &mdash; `t.Setenv("HOME")` never redirected `os.UserHomeDir()` on windows, and the private fix had been written three times.**
`os.UserHomeDir` consults **exactly three** variables &mdash; USERPROFILE on windows, `$home` on plan9, HOME elsewhere &mdash; and errors when the chosen one is empty (verified against GOROOT, not from memory). So four arms across `internal/ai` and `internal/eval_harness` failed **for the platform rather than for the code** on both Windows jobs, for four consecutive `dev` commits, with the pre-merge commits reading NO-RUN because only a push tip gets a run. Production code was correct throughout.
**Why a sweep and a gate, not a patch:** `setHomeDir` (cmd/ailang), `setHomeDirForTest` (internal/effects) and an inline GOOS-branched pair at 2 of internal/executor's 6 sites already existed &mdash; three correct local fixes, and the fourth call site went red anyway. One `testutil.SetHomeDir` now (16 bare sites &rarr; 1; control `t.Setenv(` 334 &rarr; 318, delta exactly 16), plus `make check-home-isolation` wired into `make/code-health.mk`, `make/ci.mk`'s `ci:` **and** `.github/workflows/ci.yml`.
**The thesis, measured (rule 3n):** reverting the whole `internal/ai` hunk leaves `go test ./internal/ai` **rc=0 on darwin** &mdash; no local killer, by construction &mdash; while the gate reds. The gate is the only thing on this machine that can see the defect the sprint exists to fix.

**[NEXT &mdash; iter-320, all four found by the round-2 judge on the 87 lines of shell this iteration shipped; each is cheap and none blocked the landing] `m-home-isolation-gate-hardening` &mdash; the anti-vacuity floor is itself vacuous one level down.**
**(1) The fixture floor counts, it does not check SHAPES.** `FIXTURE_COUNT -ge 3` is satisfied by three copies of the same bare call, so a future fixture edit could silently drop the `os.Setenv` or multi-line coverage with the gate still green. **Demonstrated:** the judge mutated `normalized_hits()` to require a literal `t.` prefix (dropping `os.Setenv` coverage entirely) and G1 caught it **only because** today's fixture happens to carry an `os`-prefixed shape. Fix: three named counts (`bare -ge 1`, `os -ge 1`, `multiline -ge 1`).
**(2) The self-test's five new arms have NO killer.** Reverting them leaves `make test-check-home-isolation` rc=0 at 3 arms instead of 8; nothing asserts a minimum arm count. This is (1)'s counterpart &mdash; the shape-blind floor and the shape-testing arms are currently each other's only backstop.
**(3) The line-number reporter hardcodes `(t|os)` receivers.** A call on any other receiver is correctly CAUGHT (the normalised matcher sees it) but mis-described as *"whitespace-spanning"*. Detection sound, diagnosis lies. Purely theoretical today &mdash; zero non-`testing.T`/`os` `Setenv` methods exist repo-wide, control firing &mdash; but real by construction.
**(4) It costs 22.55s** (9.53 user / 13.09 sys) over 2467 `.go` files, three subprocesses per file plus a per-file parent walk. Fine inside `make ci`; annoying for a contributor iterating on one violation. Architectural, not algorithmic &mdash; a single-pass awk/perl or a Go checker is sub-second.

**[LANDED 2026-09-02 (iter-318) &mdash; PR [#1015](https://github.com/sunholo-data/ailang/pull/1015) &rarr; squash `bd28f845c`, 21 checks zero not-green, BOTH windows jobs green] `m-message-watcher-windows-wallclock-flake` &mdash; `TestMessageWatcherStart` reds on the Windows runner from an absolute wall-clock bound.**
**Frequency CORRECTED at pick time: 1 of the last 40 CI runs (2.5%), not "every PR"** &mdash; the other 3 reds in that window were the `launchd drivers (bash 3.2)` race. Fix: explicit `cancel()` as the stop stimulus, and a budget of `max(20 &times; measured scheduling latency, 10 &times; pollInterval)` so a derivation can only ever RAISE a timeout; floor = 1 s = the historical bound, and effective grace after the stimulus roughly doubles. **Three executor rounds. R1 derived the budget with only a 1 &micro;s floor &mdash; a ~400&times; TIGHTENING (2.3&ndash;4.2 ms vs 1000 ms) on the only machine where the arm has ever flaked &mdash; and shipped a ratio assertion that was a tautology by inspection; I sent it back on my own measurement.** R2 fixed the direction and the judge passed it 94/100 &mdash; then **CI reddened BOTH windows jobs deterministically** (`--- FAIL: TestMessageWatcherStart (0.00s)`, `instrument failure: initial task scheduling latency 0s is outside (0, 1s)`), firing the degeneracy guard that the executor, the round-1 judge (explicitly asked to break it) and I had all three certified unreachable &mdash; all three of us reasoning on darwin/arm64, where `time.Since` has nanosecond granularity. On a coarse-clock platform a sub-tick interval reads back as exactly `0s`. R3 narrows the predicate to `< 0` and gives the milestone its **first genuine local killer**: two arms, latency forced to zero, R2 code rc=1 with a message byte-identical to CI, R3 code rc=0. Round-2 judge **PASS 93/100, zero blocking**.
Observed on PR [#1009](https://github.com/sunholo-data/ailang/pull/1009), whose entire diff is **one markdown file** under `.claude/` &mdash; it cannot affect a Go test. `test-windows` step 10 (`Run Go test suite`): `--- FAIL: TestMessageWatcherStart (1.18s) &mdash; watcher_test.go:146: timeout waiting for watcher to stop`, `FAIL github.com/sunholo-data/ailang/internal/coordinator 80.594s`.
**Attribution, measured, not assumed.** Negative control &mdash; the same job on the four most recent `dev` commits (`227d1e370`, `48817dcdd`, `10448bad5`, `95396664b`) is **`success` on all four**, so it is NOT an inherited red. Divergence control (iteration 153's, the strongest one available) &mdash; `gh run rerun --failed` on the **byte-identical tree** returned **`success`**. Outcome moved with only the environment varying, so the variable is the environment and not the diff. `test-windows` is **not** a required context (the four required are build &middot; docs-gate &middot; lint &middot; test), so it blocks nothing &mdash; it just makes every PR look red.
**The likely mechanism is rule 3m, and the honest caveat matters.** `watcher_test.go:146` waits on an absolute wall-clock deadline for the watcher to stop. The re-run proves the outcome is nondeterministic; it does **NOT** prove the margin was not moved. This tree includes iteration 316's M5, which added two cross-boundary coordinator tests that open real `messaging.OpenStore` SQLite stores &mdash; on a package that took **80.594 s** on that runner. So "my change did not cause it" is supported and "my change did not shift the margin" is not. **Fix: derive the bound from a stimulus measured in-test rather than hardcoding a wall-clock constant calibrated on a laptop**, and give a degenerate stimulus a loud floor so it reports instrument failure rather than passing quietly. Do not "fix" it by enlarging the timeout &mdash; rule 3n's intermittent-kill clause: making a flaky assertion pass more often is the same defect with a longer mean time to discovery. Related standing exposure worth measuring in the same pass: **51** `_test.go` files under `internal/`+`cmd/` contain a hardcoded `N * time.Millisecond` bound (control &mdash; 52 files mention `time.Millisecond` at all), and **zero** test files anywhere vary `GOMAXPROCS`.


**[LANDED 2026-09-02 (iter-319) &mdash; PR [#1020](https://github.com/sunholo-data/ailang/pull/1020) &rarr; squash `f5d031161`, 21 checks zero not-green, CLEAN. Claim CONFIRMED first-party before routing: the full revert passed 42/42 rc=0 in 112s against a 50s baseline. Now a SOLE killer; the missing `PROBE_TREE_DISCOVERY_SECS` suite-scope leak guard landed with it (kills 3/3 by name on the file-global shape, by name on the ambient shape, and rc=0 when neutered). **CI overruled the judge and me and vindicated the executor**: we deleted two arm-scoped stabilizers as unnecessary on 8 local runs; `launchd drivers (bash 3.2)` reddened deterministically on the runner (`driver_rc=0`, empty peer set) because `run_lane` reaches `sample_tree` only from inside its sampling loop, so a stub driver that exits first means the walk is never entered. Both restored. Residual, NOT fixed and now their own rows: both leak guards fire only retrospectively after arm 41; the suite has ambient-contention flakiness independent of this diff; `run_lane`'s driver is never killed on the `instrument_failure` path] `m-probe-derace-has-no-killer` &mdash; a full revert of the process-tree de-race passes the whole suite, so CI cannot see the fix disappear.**
PR [#1013](https://github.com/sunholo-data/ailang/pull/1013) split the probe's single shared deadline into a lane bound and a discovery bound (`PROBE_TREE_DISCOVERY_SECS`), which is what took `launchd drivers (bash 3.2)` from red to green. **Reverting `sample_tree`'s signature change and the `run_lane` call site together, back to the pre-PR shared-deadline design, still passes all 42 arms rc=0** &mdash; the only difference is suite wall time, 147 s against a 47&ndash;90 s baseline, because the old code still eventually emits the identical wall-clock message at t&asymp;60 s (the arm's own `PROBE_TIMEOUT_SECS=60`), comfortably inside `ARM_CAP_SECS=120`. So the hunk this whole iteration exists for is pinned by **nothing**, and a future accidental revert lands green. This is rule 3n(b) &mdash; a hunk with no killer is a queue row, not a sprint to widen.
**Proposed fix, from the judge:** raise the wall-clock arm's `PROBE_TIMEOUT_SECS` well past `ARM_CAP_SECS` (e.g. 150) so pre-fix code trips the arm cap &mdash; a different, *failing* signature &mdash; instead of succeeding slowly. On the fixed code the arm still refuses in ~1 s via `PROBE_TREE_DISCOVERY_SECS=1`, so the pin costs no wall time. **Verify it is a SOLE killer before believing it**, and read which arm reds rather than the exit code.
**Same pass, second unpinned surface:** there is no suite-scope leak guard for `PROBE_TREE_DISCOVERY_SECS`, though its sibling `PROBE_MAX_TREE_NODES` has one protecting the D4 attended ruling's arm-scoped override. A silent promotion of the new knob to a file-global or export would go uncaught.

**[NEXT &mdash; iter-317, judge-measured on this machine; a production-path behaviour change that shipped inside a CI fix] `m-probe-discovery-default-30s-unpinned` &mdash; the new discovery budget is a tightening nobody chose and no test pins.**
`PROBE_TREE_DISCOVERY_SECS` defaults to **30 s**. Before #1013, discovery shared the lane deadline, which defaults to `PROBE_TIMEOUT_SECS=900`. **Measured by the evaluator on this rig**, replicating `descendant_pids`' exact fork shape (one `pgrep -P` per BFS pop): **57&ndash;66 nodes/s quiet, 26&ndash;28 nodes/s under 8&times; CPU contention**. At those rates 30 s reaches only ~**800&ndash;2000** nodes against the default ceiling `PROBE_MAX_TREE_NODES=4096`, so on a contended machine the **wall-clock** branch now wins where the **node-ceiling** branch used to &mdash; a real production probe run gets a different refusal message, and a long-but-healthy walk that previously completed can now refuse.
**Severity is genuinely moderate, not high**, and the judge said so: real `ailang eval-suite` process trees are small, so reaching 4096 nodes is already anomalous. The defect is diagnostic precision on an already-broken run, not normal-path correctness. **The unpinnedness is the durable half:** a mutant changing the default `30 &rarr; 5` passes all 42 arms.
**Fix options, in the order the judge ranked them:** document the rationale for 30 with the measured nodes/s this repo relies on; or derive the bound from a stimulus measured in-test rather than hardcoding a constant calibrated on one machine (rule 3m); or pin the default with an arm that reds when it moves. Do not simply enlarge it &mdash; that is rule 3n's intermittent-kill clause wearing a config value's clothes.

**[NEXT &mdash; VERIFY-then-route; iter-316 Gate-0 triage of three NEW `docparse` reports; the cache bug is the one that matters] `m-docparse-v0340-reports-2026-09-01` &mdash; a live downstream consumer's three defects at v0.34.0, one of them a silent export drop that kills a hosted endpoint.**
All three arrived 2026-09-01 09:33&ndash;09:34 from `docparse` (sunholo/docparse v0.22.0) on the canonical inbox. None is tracked anywhere: charter/log mentions **0** for `process-timeout`, `process-max-output`, `MCP tool schema` and `iface cache` (control &mdash; `serve-api` reads 8/11, so the instrument sees positives), and no open issue covers them (adjacent but distinct: [#498](https://github.com/sunholo-data/ailang/issues/498) is World's caller-supplied-surface ask, not this).
**(1) Stale per-directory `.ailang` iface cache silently drops exports, at compile AND at `serve-api` runtime.** Their evidence is first-party and concrete: `docparse_api/services/.ailang/cache/compile/modules/docparse_api__services__api_server/iface.json` holds **14** exports with `convertDocument` absent, while `ailang check` on that module alone passes and the whole-program check passes. At runtime the route registers and then every request to it 500s for the life of the process &mdash; on Cloud Run, a dead endpoint with **no error at startup**. This is a CLAUDE.md &sect;2 silent-fallback violation on the data an importer resolves against.
**GHOST DISCIPLINE ATTEMPTED AND IT DOES NOT REPRODUCE IN THE TWO OBVIOUS SHAPES** &mdash; recorded so the next iteration does not re-buy the easy half. On a freshly `-ldflags`-built binary (`v0.34.0-277-g48817dcdd`, not the stale PATH copy): shape A, one directory, two modules, add an export to the dependency and re-check the importer &rarr; the dependency's cached iface went **1&rarr;2 exports with the new symbol present** and the importer type-checked clean. Shape B, importer and dependency in **separate directories** so two `.ailang` dirs exist &rarr; same result, `has_bar=True`, rc=0. So plain invalidation works at HEAD and the trigger is more specific than either shape. Note what their report says that neither shape reproduces: the lying cache sat in `services/.ailang` while the module lives in `services/api_server`, i.e. the stale entry is one the *importer's* run consulted and did not refresh. **Next step: reproduce with the module in a SUBdirectory of the cache-owning directory, and with a cache written by an older binary or before an edit the staleness detector misses (mtime granularity, or a content key that did not change).** Do not route a sprint until it reproduces &mdash; 4 of 7 historical survey-sourced rows were ghosts, and this one is a report from a real consumer, which is stronger evidence but still not a repro.
**(2) `serve-api` cannot configure the Process effect.** It never calls `setupProcessHandler`, so `NewProcessContext()`'s defaults (30 s timeout, 10 MB combined output) apply and neither is reachable: no `--process-timeout` / `--process-max-output` flag (only `ailang run` has them) and no env var is read. Measured by them on 2026-08-31: `TIMEOUT at 30001ms`, `OUTPUT CAPPED at 10485760 bytes`. Cost: docling needs ~185 s on a 9-page PDF, so two of their four PDF backends are unusable on the hosted API. They would accept env vars over flags (Cloud Run sets env more easily than argv).
**(3) MCP tool schemas mark every parameter required, so a tool's arity can never grow.** `serve-api` derives the schema from the handler signature and marks all parameters required, so adding one optional parameter breaks every existing client immediately &mdash; measured: adding three optional params to `mcpConvert` made every prior `tools/call` fail `missing required parameter(s)`. They shipped reference-doc templating on the REST path and deliberately withheld it from MCP for this reason alone. Ask: any mechanism that makes a parameter optional (annotation, defaulted parameter, or "empty-string default means optional"). Note the incentive shape they name: the safe move for any growable handler is to abandon generated schemas or pass a JSON blob in one string &mdash; both worse than what the design is reaching for.

**[NEXT &mdash; iter-316, surfaced by the round-3 judge and by my own incomplete enumeration; small, and it is a PRE-CONDITION for Sprint 2 rather than a defect today] `m-changeclass-unknown-consumers` &mdash; `U` is a fourth change-class value in a codebase whose switches were written for three.**
M5 introduced `U`. Two consumers were taught about it in-milestone (`coordinator.ClassifyChange`, `messaging.TriagePackageMessage`). **A third was missed and the miss is instructive: my enumeration grepped the CamelCase `ChangeClass`, while `cmd/ailang/coordinator_cloud.go:537 classifyDispatchPath` spells its parameter `changeClass`** &mdash; a case-sensitive pattern reported as a complete enumeration, which is the scope trap this charter already records for `stdlib/` vs `std/`. Not a live defect: that switch's `default` arm is already `DispatchAI` (the conservative side, unlike the two real bugs which defaulted unknown into full autonomy), its doc comment already anticipates `(unknown / empty) &rarr; AI`, and its only producer (`mapChangeClassToSchema`) recomputes A/B/C from hashes and structurally cannot emit `U`. **Also in this row, and genuinely dead today: the router's `ChangeClass == "major"` / `== "minor"` comparisons.** No producer emits those words &mdash; measured **1** and **1** occurrence, both being the comparisons themselves, against **6** non-test lines using `"A"`/`"C"` and **62** total non-test `ChangeClass` lines (fresh negative literal 0). Consequence: an interface-change notice classified `"C"` with no `breaking` flag falls through to `ChangeClassB`. Deliberately left untouched by M5 (a pre-existing defect surfaced during a milestone is a queue row, not scope growth). **Do this BEFORE M6 wires any producer that emits `U`**, and start by re-running the enumeration case-insensitively rather than trusting the counts above.


**[NEXT &mdash; filed iter-315 from the heartbeat sprint's own design review; ~0.5&ndash;1d total, all four are cheap and independent] Four PRE-EXISTING driver defects the `m-mission-slot-heartbeat` work surfaced and deliberately did NOT absorb.**
All four were measured first-party during iteration 315 and routed as queue rows rather than scope growth, per the rule that a pre-existing defect surfaced during design is a queue row.
**(R7) The driver's late-kill record detector is structurally dead.** `mission-control.sh:1040-1041` compares `post_last_record` against `pre_last_record` to decide whether a rc&ne;0 exit landed its work &mdash; but it reads the mission log from the **pin worktree's working tree**, which `pin-root.sh` checks out ONCE at fire start (driver line 396) and never refreshes (`grep -cE '^[^#]*git (checkout|fetch)'` in `mission-control.sh` = **0**, same-file control `^[^#]*git ` = **2**; controller launches at line 968). Records land by PR merge to `origin/dev`, and the pin is DETACHED, so no merge can advance its working tree. Measured consequence: the branch has fired **0** times in the whole driver log (positive control: **10** `iteration exited rc=` lines; fresh negative literal **0**). Fix: read via `git show origin/dev:<log>` after a **bounded** fetch, with a named verdict for fetch failure &mdash; never an unbounded network call on the exit path.
**(R8) Two exit-path notifications are unbounded.** `mission-control.sh:1046` and `:1063` call `ailang messages send controlplane` with no deadline, on the exit path. `_mc_bounded()` already exists at line **194** (deadline + `kill` &rarr; `kill -9`, `( exec "$@" )`, returns 124 on timeout) and is already used **8** times, so this is a wrapper swap, not new machinery. The heartbeat sprint routes its OWN phase-2 notices through `_mc_bounded`; it deliberately left these two alone.
**(R9) `UNCLASSIFIED` is specified and unreachable.** The heartbeat design names it 5 times, including in the Q6 anti-vacuity argument (*"its fall-through is `UNCLASSIFIED`, which is loud"*), and the shipped classify `case` ends `*:*) CRASHED`, which matches unconditionally once a colon is present. `grep -c UNCLASSIFIED` = **0** in the driver against **5** in the doc (control: `CRASHED` = 1). **The floor property still holds** &mdash; `CRASHED` is loud, nothing passes vacuously &mdash; but the distinct "instrument confusion at rc=0" signal is relabelled as "the process crashed". Found by the round-3 evaluator, reproduced first-party. Either make the branch reachable or delete it from the decision table and say why.
**(R10) The heartbeat helper's unknown-label refusal has no killer.** `mission-heartbeat.sh` refuses an out-of-enum label (exit 2); the suite never exercises it (`grep -c 'unknown label'`: helper **1**, suite **0**; control: suite mentions `stamp` **22** times). Rule 3j's shape on the one guard that keeps R9's state unreachable.

**[NEXT &mdash; RE-SCOPED iter-310; HALF OF THIS ROW IS UNROUTABLE AT HEAD] `m-coordinator-config-route-preflight` &mdash; the `config diff` defect is real and standalone; the `config check` half cites a symbol that does not exist.**
**Measured at HEAD `bd17d9643`, iteration 310:** `ExecutionRoute` is **0** occurrences across `internal/` and `cmd/` (non-test), with the same-scope control `func .*Dispatch` = **9** and both roots asserted with `test -d`. `ResolveExecutionRoute`/`ValidateExecutionRoute` exist ONLY on the unmerged evidence branch `mission/iter309-route-authority-parity` @ `8c8c29864`. So this row's prescribed fix &mdash; *"calls the same `ResolveExecutionRoute` the daemon does"* &mdash; cannot be implemented until that branch lands, and the row's stated sequencing purpose (*"Sequenced BEFORE the route freeze"*) gates a sprint iteration 309 **PARKED**. **Split the row.** (a) **Landable now, independent of everything above:** move `validateCoordinatorConfigBytes` ABOVE the `identical` early return in `coordinatorConfigDiff` &mdash; verified first-party at `cmd/ailang/coordinator_config.go:327-332`, where the equality branch prints `identical to gs://...` and `return nil`s while validation runs only in the DIFFERS branch and only against `local`. An operator who runs `config diff`, sees `identical`, and believes the config was validated has been told something false; that is a CLAUDE.md &sect;2 violation on its own and needs no route machinery. (b) **Blocked on the route freeze:** the new `config check` subcommand (there is no `check` case today &mdash; `coordinator_config.go` has `get`/`set`/`diff`/`help` only). Do not route (b) until `ExecutionRoute` is on `dev`.
`ailang coordinator config diff <file>` is cited by the route-authority design as the operator preflight that catches a matrix which would permanently reject live routes. It cannot. Measured at `cmd/ailang/coordinator_config.go` (`coordinatorConfigDiff`): it byte-compares local against live, and on equality prints `identical to gs://...` and **returns nil**; `validateCoordinatorConfigBytes(local)` runs **only inside the DIFFERS branch, and only against `local`, never against `live`**. So the exact scenario it is cited for &mdash; deploy new code, config unchanged &mdash; validates nothing at all. **Correction of record:** this row's first draft said the command performs "no route evaluation whatsoever"; the iteration-309 planner refuted that and the sharper mechanism above is the correct one. Fix: a `config check [<file>]` that defaults to fetching **live**, applies daemon defaults and expands pipelines, calls the same `ResolveExecutionRoute` the daemon does, and **exits non-zero** on any refusal; plus move validation above the `identical` early return in `diff`. Sequenced BEFORE the route freeze &mdash; it is what makes landing that matrix observable.

**[IN-SPRINT &mdash; iter-316: **SPRINT 1 COMPLETE, M1&ndash;M5 all LANDED**; Sprint 2 (M6&ndash;M9) DEFERRED behind a blocking precondition the loop cannot satisfy] `m-registry-interface-hash-blind-to-signatures` &mdash; a BREAKING API change is classified `patch` to every downstream consumer.**
**M5 LANDED 2026-09-01 (iter-316) &mdash; PR [#1007](https://github.com/sunholo-data/ailang/pull/1007). SPRINT 1 IS NOW COMPLETE.** `classifyChange` rewritten to the plan's D3 **2&times;2** (the doc's D5 1-D table stays refuted); `PackageVersionInfo.Signatures`; `ChangeClass`+`Breaking` on both emitters; `U` routed to review; `TriagePackageMessage` given a `U` arm. Evaluator `sonnet`, three rounds in its own worktree: **FAIL 65 &rarr; FAIL 66 &rarr; PASS 91/100 zero blocking**. **Both blocking findings were the SAME SHAPE, and it is the shape to expect from every remaining milestone: M5 introduces a fourth enum value `U` into a system whose switches knew only A/B/C.** **(B1)** Setting `Breaking` for a **wholly-legacy** pair was NOT dark, because `coordinator.ClassifyChange` short-circuits on `breaking` *before* it reads the kind &mdash; measured first-party (A=0,B=1,C=2): upgrade-available class C went **0&rarr;2**, interface-change **1&rarr;2**. The plan's "Sprint 1 is dark / byte-identical" claim was false for every ordinary publish. Fixed by `breakingFlag`, which returns **nil** unless a side carries signatures. **(B2)** `U` was special-cased only in the `PkgMsgInterfaceChange` arm, so a **MIXED** pair emitted as upgrade-available classified `ChangeClassA` &mdash; autonomous auto-merge of a change just declared unmeasurable &mdash; and a mixed pair is the unavoidable shape of every package's FIRST post-migration publish. Fixed by **hoisting** the `U` check above the kind switch and deleting the per-kind copy, so a forgotten arm is impossible rather than merely absent. **Both defects shipped green because nothing crossed the messaging&rarr;coordinator boundary**; both fixes are pinned by cross-boundary locks that emit through `internal/messaging` against a real store and route the envelope through `internal/coordinator`. **Three things M6&ndash;M9 must carry forward.** **(1)** The `U`-into-an-A/B/C-world shape is not exhausted &mdash; see the `m-changeclass-unknown-consumers` queue row; re-run `grep -rniE 'changeclass'` (CASE-INSENSITIVE) fresh rather than trusting iteration 316's count, which missed `classifyDispatchPath` for exactly that reason. **(2)** The legacy path is byte-identical to pre-M5 and there is a test that fails if that stops being true; if a later milestone deliberately changes it, that is a decision to state, not a diff to make. **(3)** No CHANGELOG entry yet, by the same reasoning as M1&ndash;M4: nothing user-visible moves until a producer emits signatures. The entry is owed with Sprint 2's landing, and the judge has now deducted for its absence three rounds running &mdash; if Sprint 2 slips, land the entry anyway.

**M3 LANDED iteration 313** &mdash; `internal/pkg/iface_subprocess.go`: `BuildModuleIface(ctx, packageDir, modulePath, lim)` shells out to M2's hidden `internal-dump-iface` via `exec.CommandContext`, `cmd.Dir` = the **absolute package root**, its own process group, and a per-module `context.WithTimeout`; plus `PublishLimits` (60s / 10s / 64 modules), injectable so the deadline arms run in 50ms. Signature is the plan's D2 form `(*iface.InterfaceJSON, error)`, **not** the doc's `(*iface.Iface, error)`. **Three things M4&ndash;M5 must carry forward.** **(1) D2's conclusion is right and its stated PREMISE is false** &mdash; it claims no deserializer exists; `internal/types/json.go:723` defines `UnmarshalScheme`. The conclusion survives only because that function consumes the types package's own Scheme JSON while canonical JSON carries a rendered type **string**. Do not inherit the premise. **(2) The import direction was measured BEFORE routing, not after** &mdash; `internal/iface` does not reach `internal/pkg` (**0** of **104** transitive deps, control **13**), which is M2's lesson applied ahead of the defect; M4 folds `BuildModuleIface` into `InterfaceHashV2` inside `internal/pkg`, so that direction still holds. **(3) An assertion that needs `internal/pipeline` must live in `package pkg_test`** &mdash; measured both arms: an internal test importing it is rc=1 `import cycle not allowed in test`, an external one is rc=0, and Go links both into one test binary so `TestMain`'s resolver override still applies. **Judge (sonnet) PASS 79/100 with TWO BLOCKING findings, both reproduced first-party and both PROVEN closed by mutation:** the M3-A5 deviation's import-cycle justification was false (a **controller** error &mdash; the directive authorised the fallback and never named the external-test route), and `resolveIfaceBinary` had **no coverage on either branch** &mdash; its whole body replaced by an always-failing stub left all six named arms PASSing. Fixed by extracting the pure `isAilangBinaryPath` and a named `realResolveIfaceBinary`; the same mutant is now SOLE-killed by `TestResolveIfaceBinary_FallsBackToPath`. A third, non-blocking: the export-limit `>` &rarr; `>=` mutant reddened **nothing**, now SOLE-killed by `TestBuildModuleIface_ExportLimitBoundary`. Doc **AC14** (`TestBuildModuleIface_ModulePathResolution`) was absent from the plan's M3 list and shipped anyway &mdash; the doc wins on scope. **Three residuals declared, not papered over:** the `ctx.Err()` early return and the `filepath.Abs` normalization have **NO killer** (judge-confirmed independently), and `PublishLimits.Overall` has **no consumer** until publish orchestration lands. &rarr; PR [#1000](https://github.com/sunholo-data/ailang/pull/1000)

**M2 LANDED iteration 312, and the plan's M2 was UNIMPLEMENTABLE as written** &mdash; it specified `iface.BuildCanonicalJSON` in `internal/iface`, which cannot import `internal/pipeline` (pipeline imports iface in **7** non-test production files; measured two-arm, `go build ./internal/iface/` rc=1 `import cycle not allowed` with the import, rc=0 without). Shipped as **`pipeline.BuildCanonicalJSON`** in `internal/pipeline` &mdash; the derived home, not a preferred one: it already imports iface, already owns `RunWithContext`/`Config`, `cmd/ailang` already imports both, and it need not be importable by `internal/pkg` because pipeline transitively imports `internal/pkg` &mdash; which is precisely why **M3 is a subprocess wrapper, and is unaffected**. Plus the hidden `internal-dump-iface` subcommand and `outputInterface` repointed in-process. The sprint plan's M2 section is annotated with the correction and with the **packageDir contract M3 must honour** (pass the module's PACKAGE ROOT, never the CWD &mdash; pinned by `TestInternalDumpIface_WrongPackageDirFailsLoudly`). Evaluator round 1 **FAIL 66/100** (3 blocking), round 2 **PASS 95/100**, all three confirmed CLOSED. &rarr; `c8211b211`

**M1 LANDED iteration 311** &mdash; `internal/iface/hash_projection.go`: `HashProjection` (pure, deterministic, alias-dropping projection of the normalized JSON) + `SignatureSet` (sorted, **injective** `module:kind:name:signature` strings). Unwired dead code by design; `HashProjection`/`SignatureSet` have **0** references outside the new file, so nothing changes behaviour until M5. Evaluator `sonnet` **PASS 92/100, zero blocking**. **Two things M2&ndash;M5 must carry forward.** **(1) The encoding is injective only because of `escapeSigField`** &mdash; the judge found, and the controller reproduced, that the unescaped form collides (`{Type:"A:B",Effects:[]}` and `{Type:"A",Effects:["B:"]}` both render `mod:func:run:A:B:`). M5 diffs these sets to decide ADDED/REMOVED/RETYPED and M7 feeds them from untrusted uploads, so a collision is a wrong `ChangeClass`. `TestSignatureSet_EncodingIsInjective` pins it, PROVEN by restoring the pre-fix encoding as a mutant (LANDED, BUILDS, reds the named arm; blast radius 2, the second member explained by its golden's escaped comma). A consequence to expect: ctor strings containing commas now render escaped (`Rectangle(float\, float)`). **(2) One coarse assertion is doing four jobs** &mdash; `TestHashProjection_Deterministic` is the SOLE killer for the types sort, the funcs sort, the effects sort AND ctor retention (measured, iteration 311 + the evaluator independently). It detects each defect correctly, so this is not a coverage gap; it gives zero diagnostic signal about WHICH invariant broke. Split it into per-invariant assertions when M2 next touches this file. **Deliberately NOT fixed, with reasons:** alias-body retype is invisible to both functions &mdash; that is design doc **D6**'s named, accepted scope limit, mirroring the pre-existing `TestXModAlias_DigestIgnoresTypeAliases` lock, not a hole M1 missed; and no CHANGELOG entry was added, because the milestone ships unwired dead code and the entry belongs with the sprint's user-visible landing (`make check-changelog` rc=0).
**Design:** [`design_docs/planned/v0_35_0/m-registry-interface-hash-blind-to-signatures.md`](planned/v0_35_0/m-registry-interface-hash-blind-to-signatures.md) &rarr; `66add3b48`. Authored by rotation designer `pi:ollama/deepseek-v4-flash:0731-cloud` (three runs, all verdict `ok`, containment clean each time). **Quorum BLOCKED twice, 3/3 external reviewers present in both rounds** &mdash; neither verdict is an N&minus;1 degrade. Round-1 objections spread across three surfaces; **round-2 objections localised onto one** (compiling untrusted package source in-process), resolved under the narrow-refinement carve-out with the reviewers' verbatim fixes: a hidden `internal-dump-iface` subcommand invoked as a subprocess by BOTH binaries (which also dissolves the two-binary problem), `context.Context` propagation with per-module and overall deadlines and an exported-module cap, and a cursor-based resumable backfill. The doc's own Quorum verification log carries the full record, including the controller premise (`V9`) the review REFUTED. Sizing is the open question the planner answers: the doc claims 2&ndash;4 days and the carve-out fixes added a CLI subcommand and a backfill job to its scope.
**M4 LANDED 2026-09-01 (iter-314) &mdash; PR [#1002](https://github.com/sunholo-data/ailang/pull/1002).** `pkg.InterfaceHashV2` + `InterfaceHashVersion`, pure library (nothing calls it yet; legacy `InterfaceHash` untouched; `outputInterface` and `cmd/registry-validator` deliberately not rewired, per plan D4/D5). Twelve arms. Evaluator `sonnet` in its own worktree, **PASS 89/100, ZERO blocking**. **Three of its seven non-blocking findings were one shape &mdash; a shipped hunk with no killer &mdash; and all three were reproduced first-party and closed in-iteration:** (1) M4's export-limit check was **fully masked** by M3's identical check, which re-loads the manifest from **disk**, so neutering M4's own check left all 11 arms green; the new arm discriminates them via an in-memory manifest that lists more exports than the on-disk one, and asserts the limit message rather than `err != nil`. (2) `name:`/`edition:`/`ailang:` had **zero** coverage &mdash; each mutant reddened nothing; `TestInterfaceHashV2_SensitiveToPackageIdentity` makes each a SOLE killer. (3) The signature set was built by **ranging a map**, so its order was nondeterministic before the sort &mdash; a latent defect, since M5 diffs these sets; collection is now encounter-order behind a seen-set. **The controller's own first fix for (3) was refuted by re-measuring it:** widening the fixture from two exports to six was claimed in a code comment to bound the flake at `1/6! = 1/720`, and the re-measure returned **4 kills in 8 runs** (Go rotates single-bucket map iteration rather than permuting it). The real fix was removing the nondeterminism, not enlarging the sample; the outer sort is now a **DECLARED RESIDUAL** whose mutant reds **nothing**, with the three invariants that make it redundant written into the code, and the arm is 0 failures in 10 runs. `export:` is likewise declared unpinnable-by-construction &mdash; the module path is already inside each folded projection. **M5 is [NEXT]** &mdash; signature-set classification with the `U` class, per the plan's 2&times;2 (not D5's 1-D test).

**Sprint plan LANDED (opus planner): PROCEED WITH RE-SCOPE &mdash; 9 days, not the doc's 4, split into a reversible Sprint 1 (4d) and a deferred irreversible Sprint 2 (5d).** The planner verified all 8 controller facts and refuted **three blocking design defects the two quorum rounds missed**: **(1) 16 of the doc's 19 acceptance gates are GREEN AT BASE and can never go red** &mdash; `go test -run <non-matching-regex>` exits **0** ("no tests to run"), indistinguishable from a pass, so an executor could ship zero tests and report all-green; compounded by `testutil.FindAilangBinary` calling `t.Skipf` on a stale binary (`ailangbin.go:74-79`), also rc=0. Every gate in the plan now asserts `--- PASS: <Name>` **and** rc=0, measured in both directions. **(2) `BuildModuleIface(...) (*iface.Iface, error)` is not implementable** &mdash; canonical JSON renders types as strings (`json.go:33`) while `IfaceItem.Type` is `*types.Scheme` (`iface.go:29`), and no deserializer or type-string parser exists (both greps empty, controls hit); corrected to return `*iface.InterfaceJSON`, which also removes a deserializer from scope. **(3) D5's classification table would stall every cascade the day it lands** &mdash; "old side absent &rarr; `U`" makes every pre-migration pair legacy-vs-legacy &rarr; `U`, so auto-apply stops for **every package** until backfill completes: the doc's own harm, inverted. Replaced with a 2&times;2 (legacy/legacy &rarr; today's hash-only A/C, unchanged and not a fallback; legacy-old/v2-new &rarr; `U`; v2/v2 &rarr; A/B/C). Also refuted the doc's **AC10 mutation as invalid** (`TestXModAlias_DigestIgnoresTypeAliases` calls `b.computeDigest` directly, so no new serialization touches it and the mutant stays green). **Sprint 1 (M1&ndash;M5, all reversible, zero registry writes)** is ready to execute; each milestone's riskiest gate is RED at base, and mutations are anchored to the whole diff (the `case "internal-dump-iface":` arm, `cmd.Dir`, `setProcessGroup`, the `PublishLimits` defaults, `sort.Strings(exports)`, the `case "U":` router arm) so the supporting hunks are pinned rather than shipping unguarded. **Sprint 2 (M6&ndash;M9) carries a BLOCKING PRECONDITION the loop cannot satisfy:** the design's blast-radius measurement &mdash; how many already-published versions the new publish-time type-check gate would newly refuse &mdash; is UNVERIFIED, and the live registry is unreachable from the loop's session. Shipping M6 without it ships an unbounded regression. Plan: [`...-sprint-plan.md`](planned/v0_35_0/m-registry-interface-hash-blind-to-signatures-sprint-plan.md).
`InterfaceHash` (`internal/pkg/hasher.go:73-100`) hashes exactly five things: package name, edition, ailang version, the sorted list of exported **module paths**, and the sorted max effects. **Zero signature or type data enters it** (measured: 0 matches for `Signature|Type|Func|Arity|Param` in the function body, same-scope control = 5 `Fprintf` calls). `Exports.Modules` is a `[]string` of paths. So the hash is invariant under adding an export, **removing** one, or **changing an exported function's type or arity** &mdash; and `EmitInterfaceChangeNotice` (`internal/messaging/pkg_events.go:57`) short-circuits on hash equality, so the cascade emits no interface-change notice and labels the release `patch`. The reporter observed the addition case in the wild (`sunholo/external_backend` 0.1.0&rarr;0.2.0, byte-identical interface hash, notified as `patch`) and flagged the removal/retype cases as an unverified worry; they are confirmed here by construction. A consumer pinning by `interface_hash` cannot see a breaking change. This is a CLAUDE.md &sect;2 no-silent-fallback violation on data a downstream agent uses to decide whether to adopt a release.

**[NEXT] [iter-314, first-party on CI, pre-existing from iter-313] `m-publish-permodule-deadline-encodes-one-machine` &mdash; M3's 10s per-module compile deadline is an absolute wall-clock constant calibrated on this laptop, and it has now failed on a Windows runner.**

`TestBuildModuleIface_ExportLimitBoundary` failed on `test-windows` for PR [#1003](https://github.com/sunholo-data/ailang/pull/1003) &mdash; a **docs-and-skill-only** PR touching **zero** Go files (`gh pr diff --name-only`: 5 files, all `.md`) &mdash; with `exactly-at-limit package must be allowed, got: building interface for module "test/pkg/boundary" timed out after 10s` ([job 99749688041](https://github.com/sunholo-data/ailang/actions/runs/33474114813/job/99749688041), failed step `Run Go test suite`, 15 steps, so NOT the provider-outage signature). So the red is not attributable to any hunk in the diff. It is **intermittent, not inherited**: `test-windows` reads `success` on **9 of 9** recent `origin/dev` commits including this PR's own base `0e8314549`, which is what makes it a flake rather than a standing red. `test-windows` is **not** a required context on `dev` (required: `test`, `lint`, `build`, `docs-gate`), so it blocks nothing today &mdash; it will simply red other people's PRs at random.

The mechanism is rule 3m's, one layer down from where that rule was written: `DefaultPublishLimits().PerModule = 10 * time.Second` (`internal/pkg/iface_subprocess.go`) is an absolute constant, while the stimulus &mdash; an `ailang internal-dump-iface` **subprocess** that compiles a module &mdash; scales with the machine. The Windows runner is slow enough to cross it. Note the exposure grows with the sprint: **M4's `InterfaceHashV2` calls `BuildModuleIface` in a loop** over up to `MaxExportedModules` (64) modules, each under the same 10s, so every added arm is another Windows coin flip. Fix candidates, in preference order: **(a)** derive the per-module bound from a measured in-test compile of a trivial module (the 3m remedy &mdash; makes the ratio hold by construction on any machine) with an absolute floor so a degenerate measurement fails loudly; **(b)** raise the constant and say in the code which machine class it encodes; **(c)** make the deadline injectable in the boundary arms only, which fixes the test and leaves the production constant encoding one machine. **(a)** is the only one that survives the next slower runner. Do NOT diagnose this from the exit code &mdash; read WHICH test failed, since the same job carries a benign MOD010 temp-path warning that looks like a cause and is not.

**[NEXT] [iter-312, surfaced by the round-2 judge, CONFIRMED first-party] `m-canonical-json-drylink-unpinned` &mdash; a one-word flip would silently start EVALUATING modules at the publish gate, and nothing reds.**
`pipeline.BuildCanonicalJSON` passes `Config{DryLink: true}` &mdash; "don't evaluate, just check". The judge mutated it to `DryLink: false`: mutant LANDED (sha256), BUILDS rc=0, and the **entire suite stayed green** (`internal/pipeline` 819 PASS, `cmd/ailang` scoped arms all PASS). Pre-existing rather than introduced &mdash; the line was moved verbatim from `outputInterface`, which never had a test either &mdash; but the risk is strictly larger now: the plan's **M3 will call this library function directly from the publish path**, so an accidental flip means `ailang publish` starts executing untrusted package code instead of type-checking it. Wants one arm that observes an effect only evaluation could produce. Cheap; sized at &lt;0.5d.

**[NEXT] [iter-312, surfaced by the round-2 judge] `m-compact-iface-glue-untested` &mdash; `iface --compact` is pinned for SHAPE, not CONTENT.**
`TestIface_CompactFlagRendersCompactView` (added iteration 312) asserts the compact view is non-empty, not JSON-prefixed, and differs from the JSON view. The judge replaced `printCompactInterface`'s body with a static garbage string unrelated to the interface and the test **still passed**. Narrow: the formatter itself IS content-pinned by the pre-existing `TestCompactInterface` (`cmd/ailang/check_compact_test.go`); what is missing is glue-level coverage that `printCompactInterface` feeds `compactInterface` the right bytes and prints the result faithfully. Queue material, not a sprint.

**[NEXT] [iter-309, defect in this loop's own rulebook] `m-quorum-absent-reviewers-key-does-not-exist` &mdash; the mission-control skill tells controllers to read a key the artifact never writes.**
The skill's quorum rule says: *"Before acting on any synthesis whose verdict is `proceed`, read `absent_reviewers`."* Measured on this iteration's own artifact: the top-level keys are `doc`, `iso_ts`, `reviewers`, `controller_in_session`, `synthesis` &mdash; there is **no `absent_reviewers` key at all**, so the documented read yields `null`. Absence IS recorded, correctly, at `reviewers[].present == false` with `absent_reason` (this iteration: `oc-glm-5-2`, `present: false`, `absent_reason: "invalid"`). The writer is right and the RULE is wrong &mdash; and it fails in the dangerous direction: a controller following it literally on a `proceed` verdict reads `null`, concludes no reviewer was absent, and banks exactly the vacuous pass the rule exists to prevent. Fix the rule (and note it is a *docs* fix, not a code fix). Instance 1 recorded here. **INSTANCE 2, iteration 310 &mdash; the bar is now MET.** Re-measured on iteration 310's own round-2 artifact: `jq -r 'has("absent_reviewers")'` &rarr; **`false`**, and the verdicts are not at `.reviewers[].verdict` either &mdash; the reviewer object's keys are `cost_usd, landed, model, present, result, tokens_in, tokens_out`, with the verdict, the `strongest_objection` and the `proposed_fix` all nested under **`.result`**. So the skill's prescribed read yields `null` twice over. The correct reads are `.reviewers[] | select(.present==false)` for absence and `.reviewers[].result.verdict` for the verdict. This is a Gate-5 **skill fix** lane item with two recorded frictions; it was not taken this iteration because Gate 5 allows one skill edit per iteration and this iteration had a larger deliverable.

**[NEXT] `m-probe-discovery-arm-nondeterminism` — diagnose [#975](https://github.com/sunholo-data/ailang/issues/975) on a CI runner BEFORE any fourth calibration.** The `descendant discovery refuses on the real wall-clock deadline` arm of `tools/eval/test_motoko_connection_probe.sh` has **three** measured failure modes in five days, and the decisive datum is that commit `8a384e81b` ran **success at 05:55Z and failure at 06:33Z** on a byte-identical tree ([run 33362245049](https://github.com/sunholo-data/ailang/actions/runs/33362245049) vs [33364662742](https://github.com/sunholo-data/ailang/actions/runs/33364662742)). Modes: (1) `0b35abd5d` — arm 27 got `process-tree discovery failed` where it expected `bounded termination deadline`; (2) `b64ee358e` — arm 33 refused via discovery emitting **neither** inner message; (3) `8a384e81b` run 2 — arm 33 did not refuse at all, both lanes `driver_rc=0`, empty peer set, so the `PROBE_TEST_PGREP_LOOP` fixture never engaged. Not reproducible locally: two attempts on the identical shell (`/bin/bash 3.2.57(1)-release arm64-apple-darwin25`) both produced the correct inner message. Ruled out: bash 3.2 empty-array/`set -u` (`declare -a arr=(); echo "${#arr[@]}"` → `0`, no error). Also open: **B1**, the arm asserts the *generic wrapper* `process-tree discovery failed` that `sample_tree` emits for every `descendant_pids` failure, so neutering the in-loop `date` check leaves the suite 41/41 GREEN — **pre-existing**, present at `c29ec1d00` line 363, so a queue row and not a revision. PR [#971](https://github.com/sunholo-data/ailang/pull/971) holds the deadline-decoupling work as a **draft**: structurally sound (a judge's reversion experiment showed that hunk is load-bearing) but it does not deliver the determinism its message claims and it retires the `ARM_CAP_SECS * 5` tolerance without a replacement. **Do not merge it until #975 is understood.** First step: gate `PS4`/`set -x` tracing around `descendant_pids` and the stub behind an env flag and run the arm N times *on a runner* — the flake does not live on this laptop.

**[NEXT] [iter-310, surfaced by a quorum reviewer, CONFIRMED first-party at HEAD, pre-existing] `m-registry-validator-unbounded-compile` &mdash; a public HTTP server compiles untrusted uploads with no timeout and no cancellation.**
`cmd/registry-validator` is a live HTTP server (`main.go:70` `log.Fatal(http.ListenAndServe(":"+port, nil))`; **9** non-test `HandleFunc`/`ListenAndServe` sites). Its compile gate shells out to the compiler on caller-supplied package source at `validate.go:76` (`exec.Command("ailang", "check", "--package", ".")`), `:95` (`exec.Command("ailang", "check", f)`) and `:116` (`exec.Command("ailang", "verify", "--json", f)`) &mdash; same-file control `grep -c '^func '` = **10**. All three are `exec.Command`, **not** `exec.CommandContext`: no deadline, no cancellation, no bound. A single malformed or adversarial upload that makes the compiler loop holds a validator process indefinitely. The subprocess boundary means it cannot *crash* the server, which is why this has survived &mdash; but it can exhaust it. **This is true at HEAD independent of any design**, so it is a queue row and not a revision to `m-registry-interface-hash-blind-to-signatures`, whose own reviewers surfaced it while objecting to that doc's in-process compile path (the doc names it in Risks as explicitly out of scope). Fix: `exec.CommandContext` with a strict per-invocation deadline on all three sites, plus a test that asserts a hanging child is killed. Check whether the effect/verify paths share the same shape before patching site-by-site (CLAUDE.md &sect;3).

**[NEXT] `m-weekly-sweep-orphans-2026-08-31` — triage-lite the 5 zero-mention open issues from this week's sweep.** Enumerated **90** open issues, per-issue counts across charter/log/archive/dashboard; **5** had zero mentions in all four (the 6th zero, `#953`, is the *Docs* mission's own bookkeeping thread and correctly not ours). Negative control fired on a fresh literal, deliberately not published here (a control you record is a control you spend); positive control `#852` = 29. Rows: [#963](https://github.com/sunholo-data/ailang/issues/963) `list[i] == list[j]` crashes at eval, derived Eq for lists · [#962](https://github.com/sunholo-data/ailang/issues/962) formatter round-trip fails on multi-space inside comments · [#960](https://github.com/sunholo-data/ailang/issues/960) derive/allow Eq for records and `Option[int]` · [#959](https://github.com/sunholo-data/ailang/issues/959) `test --package` should also run inline test blocks · [#941](https://github.com/sunholo-data/ailang/issues/941) design-quorum records a reviewer absent for a reason that should not be fatal. Ghost-discipline each at HEAD before routing — a survey-sourced row is a claim, and 4 of 7 such rows have historically been ghosts. Positioned by normal ordering; a sweep never outranks an existing pick.

**[NEXT &mdash; UN-PARKED iter-310; `D-47` was answered on 2026-08-28 and the queue row was never updated] m-openrouter-session-chain-registration**
**Stale park, found by iteration 310's Gate-2 predicate re-check.** The authoritative decision ledger marks `D-47` **RESOLVED**, and its answer names this row's consequence verbatim: *"`m-openrouter-session-chain-registration` is UN-PARKED &mdash; take the ~2 LOC chain-only change off PR [#945](https://github.com/sunholo-data/ailang/pull/945)'s branch (omit `StageID`; the receiver's existing `if stageID == "" { stageID = ss }` fallback carries it), and drop the changelog's 'and stage' claim so stage attribution is honestly absent rather than misattributed."* Mark answered that attended on 2026-08-28; iterations 305&ndash;309 all ran and none applied it, because nothing in the loop reconciles a queue row's PARK tag against the ledger row it cites. The work is fully specified and ~2 LOC. `scripts/mission_decisions.sh --open` returns **zero** rows, so this park was invisible to the one instrument that looks.
&mdash; **ITERATION 296: IMPLEMENTED, JUDGED, AND DELIBERATELY NOT MERGED.** The registration landed on PR [#945](https://github.com/sunholo-data/ailang/pull/945) &rarr; `d86399f0a` and is **converted to DRAFT**. The chain-level half works and is verified; the stage-level half **cannot** work as specified, because the wire `session_id` is chain-grained while `sessions.session_id` is a PRIMARY KEY, so N benchmarks in one `eval-suite` run collapse onto one row (last write wins) and every span resolving by chain is handed the **wrong** stage. That is worse than the gap it closes: misattribution instead of absence. Full measurement and the three options are in **`D-47`**; do not re-route this row until that decision lands. The executor's own test also failed to pin the CALL SITE (deleting it left the whole `cmd/ailang` suite green), and the AST arm the controller added to fix that is itself defeated by `if false { … }` &mdash; both reproduced, both recorded, neither merged.

&mdash; **`M-MISSION-LOOP-UNIFIED-TELEMETRY` M1's writer side never landed, and M1 passes 5/5 of its own acceptance
criteria anyway.** The read side is live and correct: `otlp_receiver.go:467` resolves `chain_id` from a span's
`session.id` via `LookupChainBySessionID`, with `ailang.chain_id` keeping precedence, and four committed tests pin it
(`internal/observatory/session_chain_linkage_test.go`). The write side does not exist. `UpsertSessionWithCorrelation`
is implemented on **all four** backends (`backend_sqlite.go:341`, `backend_gcp.go:616`, `backend_jaeger.go:258`,
`backend_composite.go:349`) and **nothing calls it with a chain id**. Measured, iteration 295, first-party:
`internal/ai/openrouter/correlation.go` &mdash; the design doc's own planned file for this (`m-mission-loop-unified-telemetry.md:187`,
*"emit the session registration alongside the correlation fields, ~40 LOC"*) &mdash; has **0** registration hits
(`grep -nE 'Upsert|Register|observatory|sessions'` rc=1) against a same-file control `session_id` rc=0 at **2**.
&mdash; **Scope the claim precisely** (iteration-295 evaluator, reproduced first-party): `UpsertSessionWithCorrelation` **is** called
with a `ChainID` at two production sites &mdash; `cmd/ailang/eval_benchmark_agent.go:444` and `cmd/ailang/observatory_writer.go:42` &mdash;
plus the hook handlers. Those key on the **Claude Code / agent-mode session id**, which is why the control reads 19262. What is
missing is a row keyed by the **OpenRouter `session_id`**, and that value *is the chain id*. The two namespaces are different, and
only the second is what a Broadcast span carries back.
On the live 434&nbsp;MB observatory: `sessions_keyed_by_a_chain_id` = **0**, controls `sessions_with_chain_id_set` =
**19262** and `chain_stages_with_session_id` = **18947**. Provider-side, prod, 72&nbsp;h window: **159** `LLM Generation`
spans, **97** carrying `session.id`, **0** resolving to a chain. So every OpenRouter Broadcast trace this mission emits
is still unjoinable &mdash; the exact problem the sprint was authorised to fix.
&mdash; **Scope is small and the call site is known**: `AIAgent.SetCorrelation` (`internal/eval_harness/ai_agent.go`)
already sets the OpenRouter `session_id` **to the chain id**, so the sessions row to register is
`session_id = <chainID>` with `SessionCorrelation{ChainID: chainID}`. ~40 LOC plus a test.
&mdash; **Scope is SETTLED &mdash; no decision blocks this** (`D-45` withdrawn by measurement). Register the sessions row in the
standard-mode path at `cmd/ailang/eval_benchmark.go:112`, where **both** ids are already local: `evalChain.ChainID` and the
`stageID` created at `:96`. Agent mode already registers its own session at `eval_benchmark_agent.go:444`; this is the
standard-mode gap. ~15 LOC plus a test that fails if the row is absent &mdash; and the test must exercise the DISPATCH path, not
seed the row by raw `INSERT`, which is precisely how M1's five existing tests stay green while the writer does not exist.
&mdash; **The write must never block or fail the eval call** &mdash; best-effort, exactly as the rest of this design is.
&mdash; This row is placed above `m-prompt-freeze-mirror-all-versions` because it is the direct continuation of a
human-prioritised directive, not because a sweep outranked a pick. Ordering is the controller's call and reversible.


**[LANDED 2026-08-27 (iter-292) — PR [#937](https://github.com/sunholo-data/ailang/pull/937) &rarr; squash `445ccb550`; all 4 required contexts green (build · docs-gate · lint · test), evaluator `sonnet` FAIL 78/100 r1 on a real blocking defect, fixed before merge in `f99705be0`. **The gate is PROVEN to run in CI, not merely declared**: the runner reports `Check frozen prompt immutability and mirror agreement: completed/success` on both PR heads. The plan's wiring alone would NOT have run it &mdash; `make ci` is never invoked by CI (0 occurrences in `ci.yml`; at base 21 of 28 `ci:` prerequisites had their own step and the other 7 ran by another explicit invocation), so an explicit step in the `fetch-depth: 0` `test` job was added, preceded by an explicit `git fetch` of `origin/dev` for the L3(c) merge-base arm] m-prompt-version-freeze-on-first-bank M2 — the CI gate**
&mdash; `make check-prompt-freeze` (corpus-derivation audit, frozen-entry immutability vs merge-base,
mirror-registry agreement), wired into `make ci`. **M1 LANDED 2026-08-27 (iteration 291), PR
[#936](https://github.com/sunholo-data/ailang/pull/936) &rarr; squash `ed5600da6`.** M2 is what makes the
D-41 class unable to recur silently, and iteration 291 measured its argument rather than assuming it: the
registry carried a **stale hash for `aver`** in both registries and `LoadPrompt("aver")` **failed
outright**, with nothing in CI able to see it (audit 58 ok / 1 bad &rarr; 59 ok / 0 bad after repair).
Design doc + sprint plan both landed; M2's tasks are `design_docs/planned/m-prompt-version-freeze-on-first-bank-sprint-plan.md` §7.

**[LANDED 2026-08-28 (iter-297) &mdash; PR [#949](https://github.com/sunholo-data/ailang/pull/949) &rarr; squash `5e5c77dee`; all four required contexts green ON THE MERGE COMMIT (build &middot; docs-gate &middot; lint &middot; test), 19 of 20 checks `success`, the one non-green being the SonarCloud red inherited across the 8 preceding commits. Evaluator `sonnet` **PASS 96/100, zero blocking**; its one substantive finding — the `checked N registry entries` stdout hunk had no test killer — was reproduced first-party and fixed before merge with a sole-killer arm (`b764d2d7b`). Quorum blocked TWICE, 3/3 reviewers present both rounds; r2's three objections were all premises about one surface, measured by the controller under the narrow-refinement carve-out, and one of them CONFIRMED a real gap (two `checkRegistries` callers missing from Files-to-modify) that would have broken the build] m-prompt-freeze-mirror-all-versions**
&mdash; The queue-head decision below is **RESOLVED: EXTEND**. Iteration 293's planner measured a 7-cell matrix with
frozen positive controls (`fz_b`/`fz_g` &rarr; rc=1) proving the hole is WIDER than "the embedded `.md` is unchecked":
for a mutable entry the mirror may be **deleted** (rc=0), **diverged** (rc=0), carry a **stale hash** (rc=0), or be
**absent in BOTH copies** (rc=0). And `cmd/ailang/main.go:21` is `//go:embed all:prompts`, a **directory** glob, so a
missing active prompt **compiles clean**, falls back to disk via `findProjectRoot()` on any dev machine, and fails only
for a user of a released binary &mdash; CI is structurally blind by construction. Scope: mirror byte-agreement + registry
hash vs file bytes + file existence, for ALL entries; **immutability stays frozen-only** (the merge-base loop keeps
`if baseEntry.Frozen == nil { continue }` unchanged &mdash; mutable means editable, that is the point of the state).
Full milestone spec, ACs with measured base rcs, and the refusal-branch test plan are in the iteration-293 planner
output; carry it into the doc rather than re-deriving. Note the deferral reason in the parent doc's Open Question 2
(N-7 manual workflow) was already obsoleted by `f99705be0`.

**[LANDED 2026-08-27 by a concurrent attended session &mdash; `4d8705699`, option (2)] [iter-293, from the external report, ghost-disciplined REAL at HEAD] m-string-charat-totality**
&mdash; **RESOLVED WITHOUT THE BREAKING CHANGE.** `4d8705699` adds `charAtOpt -> Option[string]` and `charAt_or`, both rune-indexed and agreeing with `charAt` on every in-range index; `charAt` keeps its aborting behaviour and now documents it. Whether `charAt` itself should become total is deferred by that commit to a prompt-version boundary &mdash; iteration 294 files that as **`D-43`** rather than leaving it inside a landed row. Verified at `4d8705699`: `std/string.ail:160` still `charAt`, `:167` `charAtOpt`. The original row text follows.

&mdash; `std/string charAt` **panics** on an out-of-bounds index instead of returning `Option`, alone among the stdlib's
partial accessors (`list.head`, `list.nth`, `string.stringToInt`, `json.asString`, `bytes.slice` all return `Option`).
Reproduced at `v0.34.0-114-gcaea1f9e1`: `charAt("", 0)` &rarr; `Error: execution failed: _str_charAt: index 0 out of
bounds for string of length 0`, rc=1, against an in-bounds positive control at rc=0. Its signature
`(string, int) -> string` reads as total and `ailang docs std/string` carries no note that it can abort. **The
second-order cost is the real one and is what makes this more than a papercut:** the reporter used it inside a `map`,
and because a panic exits non-zero, every negative test asserting "bad input is rejected via non-zero exit" kept
passing while the program crashed on all input &mdash; a panicking accessor makes exit-code-based test suites vacuous.
Options: (1) `charAt(string, int) -> Option[string]`, breaking but the shape every sibling already has; (2) add
`charAtOr` and document the panic. Needs the breaking-change decision before routing.

**[PARTIALLY LANDED 2026-08-27 &mdash; `4d8705699`] [iter-293, from the external report] m-prompt-teaching-gaps-yaml**
&mdash; **The lambda half LANDED; the `std/yaml` half is explicitly declined.** `4d8705699` rewrites the teaching prompt's "does NOT have" row to name the curried multi-param lambda form, verified by RUNNING `foldl(\acc. \x. ..., 0, xs)` rather than by inspection &mdash; that row was actively steering agents into the reporter's parse error. That commit states `std/yaml`'s absence is **curation, not a defect**, so this row's remaining scope is a curation decision, not a bug. The original row text follows.

&mdash; `std/yaml` occurs **0** times in the served teaching prompt (controls firing: `std/json` 15, `std/list` 15,
`std/string` 18) while `std/yaml.ail` ships in `std/`; corpus-wide it is **0 of 124** prompt files against `std/json` in
**100**. Deliberately NOT bundled with iteration 293's header fix: it is a CONTENT change, so it needs an A/B and an
edit-vs-cut-`v0.16.7` decision under `D-41`, and bundling would have muddied that bisect. Same row should cover the
reporter's other content gaps (`ProcessOutput`'s shape, and that multi-param lambdas need the curried `\acc. \c.` form
&mdash; the prompt's "What AILANG Does NOT Have" table offers only `func(a,b)`, which does not work for an inline fold
callback).

**[2 of 3 LANDED 2026-08-27 &mdash; `4d8705699`; the third RE-ATTRIBUTED to `#934`] [iter-293, from the external report, three separate defects] m-dx-papercuts-docs-verify-parser**
&mdash; **LANDED:** (1) `ailang docs` assigned the description on every comment line so an export got the LAST line of its block (`exec` documented itself as `}`); first summary line now wins, and a `## Types` section was added because only `export func` was matched, leaving every exported stdlib type undocumented. (2) `ailang verify` skipped contract-less functions with a bare `continue`, so "11 functions: 11 verified" was true with a denominator chosen to make it true; it now reports uncontracted/total_exported. **NOT LANDED, and correctly re-attributed:** the mislocated multi-line lambda error is not lambda-specific &mdash; the parser emits the correct diagnostic FIRST and buries it under a cascade, which is `#934` in the orphan batch below. **`4d8705699` also declares a live divergence worth its own decision:** `ai_check.go:289` has the identical verify blindness and was deliberately NOT touched, because correcting it moves **cost-per-verified-success**, a KPI with a recorded baseline. Iteration 294 files that as **`D-44`**. The original row text follows.

&mdash; One reporter, one real build, three papercuts sharing a theme: *the tool knows something useful and does not say
it*. (a) `ailang docs std/process` output is **mangled** &mdash; descriptions are offset from their functions (one renders
as `}`, another as a call to a different function) and `ProcessOutput` is never described at all, so the `.stdout` /
`.stderr` / `.exitCode` shape, and that `.stdout` is `bytes` not `string`, had to be discovered by probing. For a module
whose whole purpose is shelling out, the result record's shape is the one thing the docs must carry. (b) `ailang verify`
reports `11 functions: 11 verified` and never says what it SKIPPED &mdash; the module exports 25, so 14 were silently
dropped as recursive/higher-order. The teaching prompt explicitly tells agents to "maximize the surface area of verified
code", so the tool asks for a metric it then refuses to report, and a function that quietly falls out of the decidable
fragment is invisible &mdash; exactly the regression the feature exists to prevent. `--show-skipped` would be enough.
(c) A multi-line multi-param lambda reports `PAR_NO_PREFIX_PARSE` at the closing paren of the whole call, several lines
below the mistake, with a hint about unbalanced delimiters; the single-line form gets the correct
`expected '.' after lambda parameter`. The parser recognises the construct one line up and should keep that diagnostic.
Each needs its own repro at HEAD before routing (all three are UNVERIFIED by iteration 293 &mdash; only reported).

**[APPROVED by D-50 BUT BLOCKED ON A PREREQUISITE FOUND AFTER THE APPROVAL — iteration 309 measured that M1 as committed REGRESSES 39 of 104 routing cells] [iter-305 root-caused] m-coordinator-child-env-opencode-retry-storm**
&mdash; **ITERATION 309: do NOT land `3500db0a7` as committed.** Mark's D-50 `execute sprint` authorization stands, but it predates this measurement. `Dispatcher.Dispatch` calls `coordinator.ValidateExecutionRoute` (`internal/dispatch/cloudrun/dispatcher.go:184`) **before** `checkVariantProviderAgreement` (`:195`), so M1 becomes the OUTER gate and the dispatchable set collapses to the two matrices' intersection the moment it lands &mdash; there is no interval in which dev's current behaviour survives. Enumerated over both REAL functions: **104 cells, 39 over-rejects, 0 under-rejects**, from FOUR causes: `managed_agents` (13 cells), the empty provider (13), the `eval`/`eval-go` wildcard images (12), and `pi-go` (1). Three independent instruments agree (controller replica; the codex executor's `route_matrix_test.go` driving both real functions; the planner's analytic 49&minus;10). **Blast radius on the DEPLOYED config is ZERO** &mdash; live prod `gs://ailang-multivac-ailang-config/config.yaml` generation `1788171270744533` carries 35 agents (pi&times;32, codex&times;2, motoko&times;1), every route inside M1's 10-cell accept set &mdash; so this is latent, not active, and nothing is on fire. **M1 and M1r are therefore inseparable and must land as ONE commit**; no existing gate could see an M1-only landing, because dev's own tests call `checkVariantProviderAgreement` directly. Evidence branch `mission/iter309-route-authority-parity` @ `8c8c29864` holds M1 + the parity probe, UNMERGED. Design `planned/v0_35_0/m-coordinator-route-authority-recovery.md` is **BLOCKED at quorum round 2** on the row below.

&mdash; **Seven identical task-failure notices overnight (00:21Z&ndash;05:30Z, all ONE task `task-a0628a5f`, sprint-planner lane), every one `executor failed: opencode execution failed: failed to start opencode: exec: "opencode": executable file not found in $PATH`.** The task is still `Pending` with `Completed: 0`, i.e. it retries indefinitely against a condition that never clears, and each retry writes to the cloud message store.
&mdash; **Two obvious explanations were measured and BOTH REFUTED.** (a) *Stripped launchd PATH*: the daemon's own environment (`ps -Eww` on pid 7239) contains `/opt/homebrew/bin`, and `launchctl getenv PATH` is empty for every process (control: a `launchctl setenv` probe reads back `present`, so empty is a measurement, not a broken instrument). (b) *Dangling symlink*: `/opt/homebrew/bin/opencode` resolves, its target exists (107 MB), and `opencode --version` returns `1.15.7` rc=0. (c) *The daemon restarted since, so I measured the wrong environment*: refuted &mdash; `ps -o lstart` shows it up since Thu 2026-08-27 07:29:48, **25 h**, spanning the entire failure window.
&mdash; **Root cause confirmed in iteration 305:** cloud dispatch independently selected the global default provider (`opencode`) and the agent's executor image variant (`codex`), so the codex image was instructed to run an absent OpenCode binary. Unconditional reset-to-pending and non-atomic terminal writes then resurrected the task and duplicated notifications. This is not a stripped child environment.
&mdash; **Why it belongs in this queue rather than being noise:** it is the same failure CLASS the sweep exists to close, arriving from a different direction &mdash; a bare binary name resolved through an inherited `PATH` that turns out not to be the `PATH` anyone measured. Note also that the daemon's own `PATH` puts `/Users/voightkampff/.local/bin` and `/usr/local/bin` AHEAD of the system directories, which is precisely the writable-precedence shape `go:S4036` names.
&mdash; Recovery plan written at `design_docs/planned/v0_35_0/m-coordinator-child-env-opencode-retry-storm-recovery-plan.md`. Resume only after fresh quorum PASS and explicit `execute sprint`; preserve M1 commit `3500db0a7`, implement OpenCode-only pinning plus M3/M4 fresh, and do not adopt the parked broad M2-M4 diff wholesale.

**[NEXT] [iter-293, external feature request, needs a design doc] m-std-smt**
&mdash; Expose the embedded Z3 to AILANG programs as `std/smt`. The argument is that contracts verify CODE over all
inputs and cannot say anything about DATA at runtime, so AILANG serves one half of verification and the requester had to
hand-roll the other: ~130 lines generating SMT-LIB2 as strings, `std/process.exec("z3", ...)`, and parsing stdout, with
every term untyped so a typo is a runtime z3 parse error rather than a type error. **Demand evidence exists** &mdash;
`sunholo/deontic` in the registry already advertises "Z3-provable settlement math", i.e. a second consumer solving this
privately. **One trap worth encoding in the library whoever builds it:** a `contains(output, "unsat")` verdict parser is
WRONG, because on `sat` z3 emits `(error "... unsat core is not available")` from an unconditional `(get-unsat-core)`,
so the substring appears in the ERROR text and every satisfiable model reads as unsatisfiable. The requester shipped that
bug. Smaller first step they suggest: a thin `solveSmtLib(script) -> Answer + core + model` that removes the process
handling and the verdict-parsing trap, with typed terms following. **Cost note:** the solver is already a dependency
(`ailang verify` shells to z3), so this adds capability rather than weight. Needs a design doc and a quorum; the
`toSmtLib`-as-auditable-artefact requirement is a real design constraint, not a nicety.

**[LANDED 2026-08-27 &mdash; `4d8705699`] [iter-293, planner side-find, UNVERIFIED by the controller] m-docs-site-prompt-version-drift**
&mdash; `4d8705699` makes `ailang prompt --version` distinguish a prompt version from a binary version, points at `--list`, shows the namespace paragraph only when the request sorts above every prompt on record (so an in-series typo does not collect it as noise), and stops `--help` hardcoding `v0.16.0` as active. The original row text follows.

&mdash; `docs/docs/prompts/current.md` carries **three contradictory version claims** (frontmatter v0.16.3, a comment
v0.16.3, an H1 v0.16.2) for an active version of v0.16.6, and its generator `sync-active-prompt.sh` is wired into
nothing (0 hits, controls at 4 and 6). The deploy DOES run `sync-prompts.sh` before build, so the live site plausibly
publishes v0.16.6 bytes under a v0.16.2 header &mdash; which is very likely what the external reporter actually read,
since iteration 293 fixed the binary's copy. Re-verify at HEAD before routing.

**[NEXT] [iter-294 &mdash; PREMISE REFUTED AND RE-MEASURED; the row below it is iteration 293's superseded text] sonarcloud-new-code-gate-red**
&mdash; **Iteration 293's central premise is a MEASUREMENT ARTIFACT, and the artifact is a silently-ignored API parameter.** That row said the red is "not attributable" because "the configured new-code period spans **2404** issues", which read as *the gate's window is far too wide to act on*. Measured at iteration 294 with controls that narrow: `issues/search?...&inNewCodePeriod=true` returns **2404**, `inNewCodePeriod=false` returns **2404**, and NO filter returns **2404** &mdash; the parameter does not narrow *at all*, so 2404 is the whole-project total wearing a new-code label. The working parameter is `sinceLeakPeriod=true`, which returns **19**. Controls proving the endpoint does respond to filters: `types=VULNERABILITY`&rarr;46, `statuses=OPEN`&rarr;1785, `createdAfter=2026-08-26`&rarr;20, and a future `createdAfter` errors LOUDLY rather than returning empty. Independently reproduced by iteration 294's verifier (`sonnet`) with its own controls.
&mdash; **The real window is narrow and the gate is measuring what its name says.** `qualitygates/project_status` reports the period as `mode=previous_version, parameter=v0.33.2, date=2026-08-26T09:50:00+0000`. `measures/component` reports `new_lines`=**6698**, `new_violations`=**19**, `new_vulnerabilities`=**4**, `new_lines_to_cover`=**879**, `new_uncovered_lines`=**323**. So **disposition (1) of iteration 293's two candidates &mdash; "narrow the new-code period so the gate measures what its name says" &mdash; is UNNECESSARY**, and was proposed only because of the broken filter.
&mdash; **It is also NOT new, which the previous row asserted in the opposite direction.** Iteration 293 recorded "`SonarCloud` `failure` is NEW at `caea1f9e1` (4 preceding analysed commits green)". Measured at iteration 294 by enumerating ALL Sonar-matching check-runs per commit (not `[0]`), with the full check-name list of one commit as a matcher control: `caea1f9e1` **failure**, `445ccb550` **failure**, `0f9abeee1` **failure**, `8baf4b8e3` **failure**, `d5fa2cceb` **failure** &mdash; **five consecutive analysed commits red**, with `e38c0c493` and `c3d9ae0d9` carrying no Sonar check at all (absent &ne; green). Verifier reproduced all seven. This is exactly the standing-red-nobody-reads shape Gate 1's own check-set rule was written for.
&mdash; **Disposition (2) is the live one, and it now has content.** `new_security_rating`=B is driven by exactly **4** issues, all rule `go:S4036` ("the PATH variable only contains fixed, unwriteable directories"), all MINOR, all OPEN: `cmd/ailang/prompt_freeze_check_git.go:82`, `cmd/ailang/prompt_freeze_core.go:194`, `internal/coordinator/worktree.go:308`, `cmd/ailang/coordinator_cloud.go:459`. **Fixing only those 4 would be gate-satisfying, not a fix** &mdash; they are flagged solely because they were touched inside the leak period, while **92** identical bare-name `git` exec sites exist repo-wide (independently measured at 92 by BOTH the designer and the verifier; my own first pass said ~95 and was wrong). That is the sweep now specified in `m-git-binary-resolution-sweep` below. The remaining coverage half (`new_coverage` 63.3%, 323 uncovered new lines) stays open and should be read per rule 3n(d) as a machine naming shipped lines nothing exercises &mdash; **not** as a threshold to satisfy.
&mdash; Iteration 293's superseded text follows.

**[M1 LANDED 2026-08-28 (iter-298) &mdash; PR [#954](https://github.com/sunholo-data/ailang/pull/954) &rarr; squash `8a993bb89`; all four required contexts **success ON THE MERGE COMMIT** (build &middot; docs-gate &middot; lint &middot; test), 20 checks, sole non-green the SonarCloud red inherited across the 4 preceding commits. Evaluator `sonnet` **PASS 91/100 with ONE BLOCKING**, reproduced first-party and fixed before merge. **M2, M3 and M4 are [NEXT] &mdash; 89 sites remain**] m-git-binary-resolution-sweep**
&mdash; **Quorum BLOCKED TWICE (3/3 present, 0 absent both rounds); every objection was MEASURED before routing, not forwarded.** R1 CONFIRMED two: the security claim was wider than the mechanism (`resolveGit` checks `filepath.IsAbs` and nothing else &mdash; no stat, no ownership, no write-permission check &mdash; so "or writable" was false and the residual-S4036 "reviewed-safe" disposition was unearned), and caching resolution FAILURE in a `sync.Once` is a NEW failure mode rather than a preserved one (43 of 93 sites are the long-lived coordinator daemon, so one early miss breaks every later git task until restart; the doc had also PINNED the regression with a test). R1 REFUTED one: **0** in-repo string-matches of `os/exec`'s not-found text, control **2918**.
&mdash; **R2 localised onto ONE surface and gemini flipped to PASS, so the narrow-refinement carve-out applied.** Both survivors shared a premise that measurement REFUTED: they read `internal/eval_harness/watchdog.go:57` as an existing git site the regex misses; it is `exec.Command("bash", "-c", …)`, and a multi-line-aware enumeration over **1,254** non-test Go files returns **93** &mdash; identical to the single-line count, differing-file list EMPTY. The blind spot was **PROSPECTIVE, not present**, which changed the urgency and not the fix. **HID-6** now makes the enumerator `go/ast`-based as a committed decision rather than a `SHOULD`.
&mdash; **The count was re-derived at HEAD rather than inherited: 93, not the doc's 92** (cmd/ailang 44, not 43), with 9 of the 18 files its Verification Log cites changed since its base `e38c0c493`. Milestone residuals and the gate baseline are now **derived by measurement at execution time**, not hardcoded.
&mdash; **The planner refuted the controller.** HID-6 clauses 2 and 3, written by the controller under the carve-out, were **JOINTLY UNSATISFIABLE** &mdash; clause 2 made any AST/regex disagreement a gate failure, clause 3 requires shipping a fixture whose purpose is to make them disagree (measured: regex **0**, AST **1**). An executor obeying both literally would have built a gate that fails on its own test fixture. Fixed before routing; clause 2 is now scoped to the real tree.
&mdash; **The evaluator's BLOCKING find is the one worth carrying forward.** The `exactly one LookPath("git")` positive invariant was a **line-anchored grep sitting inside the AST-based gate** &mdash; this mission's own *guard the helper, miss the call site* shape, one level DOWN: inside a single gate rather than across a package. Reproduced first-party and **stronger than filed**: THREE gofmt-canonical evasions planted a second resolver at rc=0 (multi-line comment-anchored, aliased `os/exec` import, constant-not-literal). Now AST-based with import resolution; multi-line and aliased are CAUGHT, the constant case is pinned as a DECLARED residual arm so the known gap is asserted rather than silent.
&mdash; **Gate 3b caught a Windows regression nothing local could see.** `test-windows` and `Build windows-latest` went red while dev's HEAD was green on both. `TestCommand_UsesAbsolutePath` asserted a Unix-only ordering: Windows resolves an absolute path through `lookExtensions`, refusing a suffix-less name at `exec.Command` time, so the assertion tested the fixture rather than the code. **The darwin arm cannot discriminate the fix** &mdash; it passed before and after &mdash; so the verdict was CI's. Second instance of this class in the charter after iteration 120's `.exe` find.
&mdash; **M2-M4 remain: 89 bare-name sites** (`internal/coordinator` 43 &rarr; M2, `cmd/ailang` 40 &rarr; M3, `internal/pkg` 5 + `internal/eval_harness` 1 &rarr; M4). They are mechanical conversions against a contract M1 froze, and the ratchet baseline must be RE-SEEDED by measurement at each milestone rather than read from the doc.
&mdash; Original row text follows.

**[was NEXT] [iter-294, design doc WRITTEN that iteration, quorum NOW RUN] m-git-binary-resolution-sweep**
&mdash; **An absolute-path git resolver already exists in this repo and production code calls it from nowhere.** `resolveGit` (`cmd/ailang/help.go:173`) and its `sync.Once` wrapper `gitBinary()` (`:185`) already require an ABSOLUTE path, with a doc comment stating the rationale verbatim: the tool "must not execute whatever a relative or writable PATH entry happens to call git". It is in `package main`, so no `internal/` package can import it. Measured: **92** bare-name git exec sites in non-test `cmd/`+`internal/` (43 `cmd/ailang`, 43 `internal/coordinator`, 5 `internal/pkg`, 1 `internal/eval_harness`; 69 `Command` + 23 `CommandContext`), of which **0** production sites route through the helper. This is this mission's own named recurring shape &mdash; **guard the helper, miss the call site** &mdash; at a scale of 92.
&mdash; Design doc: `design_docs/planned/v0_35_0/m-git-binary-resolution-sweep.md` (440 lines, authored by the Fable rotation designer, iteration 294). Failure contract decided rather than hand-waved: a new stdlib-only leaf package `internal/gitexec` whose `Command`/`CommandContext` return an `*exec.Cmd` with the `Err` field pre-set (wrapping a typed `ErrUnresolvable`) when git cannot be resolved absolutely &mdash; mirroring `os/exec`'s own Go 1.19+ `Cmd.Err` contract, so all 92 sites keep their existing error-handling control flow. Bare-`"git"` fallback was rejected as a silent fallback that re-creates S4036 exactly when it matters; process-level fail-closed was rejected because the coordinator daemon (43 sites) must fail the *task*, not the process. Milestones carry EXPLICIT residual counts (88 after M1 &rarr; 46 &rarr; 6 &rarr; 0) so the doc cannot be read as claiming coverage it does not have.
&mdash; **NOT YET ROUTED TO A SPRINT.** The doc has had no `ailang design-quorum` run and no sprint plan; iteration 294 spent its slot on the Gate-1 red and the measurement above. Quorum first, then plan.
&mdash; `SonarCloud Code Analysis` went `failure` at `caea1f9e1` (the `#939` merge) on TWO conditions: **63.3%
coverage on new code** (needs &ge;80%) and **B security rating on new code** (needs A). The 4 preceding analysed commits
are green, so it is NEW rather than inherited &mdash; but it is **not attributable to `#939`**, because the configured
new-code period spans **2404** issues and **203** hotspots, far wider than one merge (rule 3b(ix): the scope travels
with the count). Non-required (`test`/`lint`/`build`/`docs-gate` are the four required contexts), so it never blocked.
Two candidate dispositions and they are different work: narrow the new-code period so the gate measures what its name
says, or treat the coverage red as rule 3n(d) intended &mdash; a machine naming shipped lines nothing exercises. Decide
which before spending effort.

**[LANDED 2026-08-28 (iter-297) &mdash; SUPERSEDED AND CLOSED BY `5e5c77dee`; this row IS the defect `m-prompt-freeze-mirror-all-versions` was written to fix.** Its own named reproduction, measured two-arm on ONE tree with the only variable being the binary: mutable mirror `.md` deleted, source intact, registries synced &rarr; **pre-change rc=0**, **post-change rc=1** with `mutable version v0.16.6: mirror file missing at cmd/ailang/prompts/v0.16.6.md`. The decision the row posed — extend the check or declare mutable mirrors unenforced — was answered EXTEND, and immutability stayed frozen-only] m-prompt-freeze-mutable-mirror-unchecked**
&mdash; **For a MUTABLE version the embedded `.md` is checked by NOTHING.** `L3(d)` is frozen-only by design
(the doc's Open Question 2 / Non-goal 2), so a tree whose `cmd/ailang/prompts/versions.json` entry has been
hand-synced while `cmd/ailang/prompts/<version>.md` is stale or **absent** passes `make check-prompt-freeze`
at rc=0. Reproduced at `445ccb550`: registry synced, mirror `.md` deleted &rarr; gate **rc=0**. That is the
active, currently-served prompt (`v0.16.6`) on the one path agent mode reads FIRST, so a green gate can sit
over a missing embedded prompt. Iteration 292 narrowed the exposure by making `create_prompt_version.sh`
write both `.md` copies (`f99705be0`), which closes the common path but not the invariant. The design
question the doc defers is exactly this one, so this row is the decision, not a bug fix: extend the `.md`
mirror check to all versions, or state in code and in an AC that mutable mirrors are unenforced.

**[NEXT] [iter-292, from the round-1 judge] m-prompt-freeze-dead-registry-block-comment**
&mdash; Small and purely honesty-of-code. `checkGitPromptFreezeInvariants`' registry-entry block in
`cmd/ailang/prompt_freeze_check_git.go` is **dead by construction on the current call path**: `checkRegistries`'
pre-existing whole-entry marshal strictly dominates the narrower `sameFrozenRegistryEntry` compare and always
runs first, and the helper has exactly ONE call site &mdash; so its comment, *"retain this check here so L3(d)
remains complete when this helper is called independently"*, describes a path that does not exist. The judge
proved it with stderr probes: the append branch never fired across the whole `TestFreezeCheck_*` suite.
Measured as a **mutually-masking pair**: neuter either block alone &rarr; 0 FAIL; neuter both &rarr; RED. Left in
place because the doc's L3(d) names it as the mirror-agreement enforcer, but the doc requires an unreachable
branch to be declared in the code and the AC, and it is not. Fix: correct the comment, or make the block
reachable by scoping the broad check.

**[NEXT] [iter-292 Gate-0 sweep, batched] external-issue orphan batch — 7 of 85 open issues with zero charter mentions**
&mdash; All seven are fleet-filed (author `sunholo-voight-kampff`), none external, and none outranks the picks
above it &mdash; a sweep never outranks an existing pick. Each needs the ghost discipline (live-repro at HEAD)
before earning its own row. `#934` **[model-manager-session] parser emits a 342-error cascade from one bad
token** is the most actionable and arrived independently on the agent inbox the same morning, with measured
cascade sizes across four benchmark/model pairs (342 / 153 / 29 / 15) and the observation that the FIRST error
is the real defect every time &mdash; an eval-wide repair-prompt cost, not a one-model problem, and testable as
an A/B on the existing harness. `#920`/`#921` are dashboard approval-surface security rows (no auth when no
token is supplied; approve-without-seeing-what-you-approve). `#902`/`#903` are ChatGPT-subscription billing
and codex default-model rows. `#904`/`#905` are `motoko_agent` language requests (same-named exported record
types across sibling packages; optional/defaultable record fields). Sweep verdict: **7 orphans of 85
enumerated**, per-issue counts printed for all 85 across charter/log/archive/dashboard, positive control
firing at 168 and the negative control firing (literal not published, per the spend-a-control rule).

**[NEXT] m-prompt-version-freeze-on-first-bank M3 — close the agent-mode verification hole**
&mdash; **A live provenance gap in a mission KPI**, found by the designer at iteration 291 and reproduced
first-party by the controller: `internal/prompt/loader.go` has **1** `Hash` occurrence (the struct field
only) against **8** in the same-scope standard-mode loader, so **agent mode never verifies the prompt
hash at all**; and `internal/eval_harness/langreg/ailang.go`'s `LoadSyntaxRef` returns
`(a.DefaultPrompt(), "default", nil)` &mdash; i.e. it converts a load FAILURE into a **SUCCESS** attributed
`"default"`, which is then banked via `eval_benchmark_agent.go:315` (`python.go:36` is identical). Worst
case the last-resort fallback is a **one-sentence** prompt. The bank is uncontaminated **today** &mdash; 0
`default` and 0 `agent-prompt` rows across 17,343 tracked baselines, control `v0.3.21` = 210 exactly &mdash;
so the fix is prospective, but nothing prevents the first contamination. **Note M3 depends on M1's
migration having written BOTH registries** (agent mode reads the embedded one first, `loader.go:106-108`),
which it does.

**[NEXT] m-prompt-version-freeze-on-first-bank M4 — bank-time byte evidence**
&mdash; record `prompt_sha256` of the served bytes on every new banked row, so the freeze predicate can prove
WHICH bytes a row measured rather than only which version id it named. Closes the first-bank bytes-swap hole
`gpt5-6-sol` identified at quorum. Carries r3's merge-base cutoff rule: field absence warns only for baseline
files present at the M4 cutoff commit; every newly added row must carry a valid 64-hex digest or CI fails.

**[NEXT] m-prompt-freeze-split-test-pins-totals-not-attribution**
&mdash; **Found by the iteration-291 evaluator, and the controller's first attempt to refute it was itself
refuted.** `TestRealRegistry_PostMigrationSplitCounts` asserts the 19/39/1 totals and the registries'
byte-equality, but NOT the id&rarr;reason assignment. Measured: swapping one `banked` and one `legacy` reason
**in both registries** (totals preserved, mirror assertion disarmed) leaves the test **GREEN** while `python`
reads `legacy` and `v0.3.0-baseline` reads `banked`. Non-blocking &mdash; the two reasons are behaviourally
identical, since `LoadPrompt` branches on `Frozen != nil` and the reason is provenance metadata only &mdash;
so this is an attribution pin, not a correctness one. Fix: have the test re-derive from the corpus rather
than read back the migration's own output. **Instrument note worth keeping**: the controller's first mutant
touched only the SOURCE registry and was killed by the **mirror-equality** assertion, a bystander &mdash; a
two-file invariant makes every single-file mutant a bystander kill.

**[NEXT] [cross-mission, rule 2] [filed by attended session 2026-08-27, at Mark's request] ailang#885 — serveapi/protocol has no MCP dispatch**
&mdash; **Cross-mission blocker, now labeled `cross-mission` (was unlabeled since filing 2026-08-25, which
per rule 2's own measurement is why 8 of 10 prior asks like it never got picked).** Filed by the Ailang
World mission with concrete demand evidence, same shape as `ailang#477`: World's zero-cloud dependency
gate (`TestDaemonDependencyAllowlist`) measured that pulling `modelcontextprotocol/go-sdk/mcp` adds 34
packages across 5 new module roots, 28 of them disallowed (an OAuth/credential stack that collides with
clause 2's zero-cloud-core and clause 3's no-ambient-authority rules) &mdash; so World's `serveapi/protocol`
consumer is stuck between reimplementing MCP JSON-RPC dispatch by hand (forbidden by the design freeze
that #498/#764 exist to enforce) and breaching its own dependency guardrail. **The ask is narrowly scoped
and has an existence proof already in this repo**: `serveapi/a2a_handler.go` already implements the A2A
half of the same seam in 180 lines, stdlib + `serveapi/protocol` only, zero SDK imports (verified:
`git show v0.33.2:serveapi/a2a_handler.go | grep -c modelcontextprotocol/go-sdk` &rarr; 0). `protocol`
itself already carries the MCP envelope helpers (`WriteMCPEnvelope`, `RequestID`, `ValidateMCPName`,
`AuthorizationStatus`) &mdash; only method dispatch is missing. This blocks World's approval-inbox chain
(row 39 &rarr; row 40 &rarr; item 5 &rarr; clause 4), which is the dependency standing between World and
actual human use of it &mdash; concretely why this is ranked here rather than left in the generic
iter-285 orphan-sweep batch it was found in.

**[NEXT] [iter-290, from the iteration-290 judge, per `D-39` sequencing] m-fmt-corpus-gate-freeze** &mdash;
**The corpus's canonical form is now evidence-gated, not CI-gated, and that is MEASURED rather than
argued.** Iteration 290's evaluator reverted `examples/ai_modes.ail` and `std/crypto.ail` to their
pre-M3 spellings (sha-confirmed mutants) and re-ran every gate &mdash; `go test ./internal/format/...`,
`verify-examples`, `verify-stdlib`, `test-stdlib-ail` &mdash; and **all stayed rc=0**. The cause is
structural, not an oversight: `TestCorpusCommentFreeRoundTrips` asserts `Format(Format(x)) ==
Format(x)` and AST round-trip; it never asserts `data == Format(data)`, so no gate in this repo can
see a non-canonical corpus file. So all **450** files can silently de-canonicalise &mdash; by a hand
edit, a concurrent agent, or a merge &mdash; with CI fully green. `D-39` sequences the fmt gate freeze
explicitly BEHIND the width work, and the width work is now complete (`0911d1089`), so this is the
row that ruling points at. Scope for the design doc: a `fmt --check`-over-corpus gate with an
**anti-vacuity floor** (an empty enumeration must FAIL LOUDLY, not print a checkmark &mdash; the
`fmt-check-ail` defect iteration 187 found, where the gate enumerated a `stdlib/` that has never
existed and reported 400 of 446 files as "all canonical"), plus a decision on what to do with the
**45 fail-closed files** (38 attach-refusals, 7 parse-failures) which a naive gate would red on
forever.

**[NEXT] [iter-290, AITANA-DEMAND] [ghost-disciplined REAL at HEAD] m-browser-session-serving-mode** &mdash;
A downstream consumer request from `aitana-platform` (the Python protocol sibling), which already
consumes M-REMOTE-BROWSER-SESSION-PROVIDERS + M-BROWSER-AUTH-PROFILES. It wants a serving mode &mdash;
`ailang browser serve` &mdash; so an external non-Go platform can consume browser sessions as a deployed
network service (Cloud Run per customer GCP project, via ADK `McpToolset` over streamable HTTP)
rather than as an in-process Go library. **Live-repro'd at HEAD before filing, per the cross-mission
contract (a real downstream consumer IS the demand evidence, but the CLAIM still gets checked):**
`git grep` at `origin/dev` finds **no** `browser serve` verb anywhere under `cmd/` or
`internal/browser` (control: the `browser` token itself hits 5 files in `cmd/ailang`, so the
instrument fires), and **no** `ListenAndServe`/`http.Server` anywhere under `internal/browser`
(control: the same pattern hits `internal/apiserver`, `internal/coordinator`, `internal/server`).
The local provider spawns `@playwright/mcp@0.0.79` as a child process
(`internal/browser/local/playwright.go:177`). So the request is real, not a ghost. It does **NOT**
outrank the queue &mdash; a consumer cannot set this mission's priorities. Requested surface: HTTP open
returning a session id + MCP endpoint URL, HTTP close + terminal manifest, auth-profile refs from
the v0_33_4 registry unchanged, a single container image, bearer/OIDC on the control API, and OTEL
propagation into session spans. They explicitly do NOT want changes to the provider-neutral
contract, Browserbase support, or the MCP tool surface. Full request in inbox message
`inbox_1787768830607_f877e688`.

**[LANDED iter-290 `0911d1089` PR #931 — SPRINT COMPLETE] [iter-283, per `D-39`] [M0+M1+M1b `0c7f58351` iter-287] [M2 `3ee848bb0` PR #928 iter-289] m-fmt-printer-no-line-width-limit**
&mdash; **M2 IS DONE AND MERGED (iteration 289, `3ee848bb0`, PR #928).** `funcBody` (equation form) and
`topLevelLet` (nil Body) now emit the EXISTING continuation layout &mdash; ` =` + hardline + body indented
one level &mdash; when the one-line `<signature> = <body>` would exceed MaxWidth. Two hunks, 14 lines, plus
a 166-line test file. Predicate order preserved: the M1 chain branch is evaluated first, M2's generic
branch is strictly its else. **Evaluator PASS 90/100, zero blocking.** Mutation drill anchored to the
DIFF: both hunks have a killer (`topLevelLet`'s is a **sole killer**), prefixes&rarr;0 reds 4 tests.
**Known weak pin, filed not hidden:** `TestWideChainKeepsChainContinuationLayout` is documented as
killing predicate reordering and does NOT &mdash; it passes under the chain-branch mutant while a
pre-existing test carries that pin. **The judge proposed that this is because the two paths converge
byte-identically for wide chains, and recommended editing the doc's rationale; I MEASURED IT AND IT IS
REFUTED** &mdash; pristine emits a wide chain over 3 lines, the mutant inline on 1, because after M2's
hardline+indent `letIn`'s own check runs at the NEW column. Predicate order IS load-bearing for wide
chains; the doc is correct and the edit was NOT made.
**M3 IS DONE AND MERGED (iteration 290, `0911d1089`, PR #931). THE SPRINT IS COMPLETE.** Corpus
lines >120 runes **159 &rarr; 100** (AC3 needs &le;105), max line **1315 &rarr; 316** (needs &le;350),
`LET_CHAIN_2PLUS` residual **20 &rarr; 0**. 50 files (38 `examples/`, 12 `std/`); no printer code
touched. Every invariant controller-verified first-party with a control that fires: comments
7,865&rarr;7,865 over 450 joined pairs (poison arm FIRED; corroborated by a second, lexer-independent
instrument at 7,225&rarr;7,225); `ailang check` 450 pairs, **0** rc changes; the 45 fail-closed files
byte-identical (`comm -12` EMPTY, control 50); idempotence 0 differing sha rows; AC9 0 forbidden
paths. All six gates re-run OUTSIDE the sandbox, each 0/0 against its pristine base.
**Evaluator PASS 94/100, zero blocking**, in its own worktree, with a check neither the executor nor
the controller ran: whitespace-stripped before/after content is identical across all 50 files, so the
diff is a pure layout transformation with zero token-level change. It found **two real defects in the
controller's own record** &mdash; an AC8 Base column summing to 167 against a stated 159, and AC10's
"112 runes" that was 112 bytes (110 runes) &mdash; both reproduced first-party and fixed in
`bc7c1de4e`. AC3/AC8 are now MET; **doc and sprint plan moved to `design_docs/implemented/v0_35_0/`**,
and the per-construct residual table is published there as the input to the reflow follow-on.

&mdash; **Round 2 is DONE and MERGED.** `writer.effectiveCol()` returns the pending indentation when
`atBOL`, and `exceedsWidth` consults it in place of the raw `col`, so the width predicate no longer
reads a column the deferred indentation has not reached. The round-1 blocking defect (a **122-rune
line under `MaxWidth=120`**) is gone, verified three independent ways: the controller's sole-killer
mutation (revert the call site &rarr; all 4 subtests fail with `MaxWidth=121 emitted a 122-rune line`;
`-skip` inverse **rc=0, 0 failures**), the evaluator's 13-mutant drill, and the evaluator's
**compiled-product** reproduction (mutant binary max line **121**, fixed binary **66**).
**Evaluator PASS 85/100, "safe to land".** Corpus-neutral, re-derived independently by the judge:
**405** formattable / **45** fail-closed on both sides, hash listings `diff` empty &mdash; so M1b cannot
disturb AC2/AC4/AC5/AC6 or change what M3 would write.
**REMAINING WORK, and the ordering is load-bearing: M2 (continuation layout) THEN M3 (corpus
reformat).** M3 first would bank lines M2 would have wrapped, undermining AC3. AC3/AC8 are unmet by
design and the doc **stays in `design_docs/planned/`**. Measured residual: ~**116** corpus lines still
&gt;120 runes, all in categories the doc defers to M2/M3 (func-decl equation bodies, string literals,
record/list literals) &mdash; e.g. `std/string.ail:123`, `examples/web_api_demo/api/math.ail:123`.
**Provenance note:** the round-2 code was built by **iteration 286**, which was killed by the stall
watchdog (`rc=143`, *"idle with a descendant alive ≥2400s"*) after force-pushing to #918 and before
recording anything &mdash; **0** charter rows, **0** log entries (control: `ITERATION 285` &rarr; 3).
Iteration 287 found it via Gate 2's died-mid-flight worktree trace, verified it first-party, rebased
it onto `ff0da7445` (the `internal/format` tree hash is **identical** across the rebase, so the
judge's verification carries), and landed it. **ORIGINAL FRAMING BELOW.**

**[LANDED iter-288 PR #926] [iter-287, from the evaluator] m-fmt-measurement-att-isolation-unpinned** &mdash;
**PINNED.** Reproduced first-party (mutant LANDED/EFFECT/BUILDS, suite green), then pinned by
`TestMeasurementIgnoresInheritedAttachments`, which compares the isolated measurement against a
hand-built *inheriting* shadow &mdash; it observes a value written BY the mechanism, not alongside it.
Mutant A is its **sole killer** (`-skip` inverse rc=0, 0 FAIL, on a mutant measured single-test).
**Latent, not live, and now MEASURED:** double-masked by the `hasAnyAttachment(X) || exceedsWidth(X,...)`
short-circuits at `expr.go:266`/`decl.go:174`/`decl.go:572` AND by a fail-closed attachment boundary set
that makes nested-interior comments unrepresentable. Compiled-product sweep 405 identical / **0**
divergent / 45 fail-closed of 450, known-positive control (`defaultMaxWidth` 120&rarr;60) = **21**. ORIGINAL BELOW.
`newMeasurementPrinter` sets `att: nil` (`internal/format/width.go:36`) to isolate measurement from the
real attachment index. The design doc names this load-bearing (V21/V23, *"the byte-identity
invariant"*), and it has **ZERO** coverage: the mutant `att: p.att`, which reinstates attachment
inheritance, **LANDS, BUILDS and survives the entire suite green**. Found by the iteration-287
evaluator, which then tried and failed to build a live `.ail` repro within its budget &mdash; so this may
be latent rather than currently reachable, and the honest statement is *unpinned invariant*, not
*live bug*. Fix before M2/M3 extend this code, since M2's continuation layout is exactly what would
make measurement/real divergence observable. Distinct from the `measurementErr` row below: different
hunk, different mutant, both surviving.

**[LANDED iter-288 PR #926] [iter-285, re-confirmed iter-287] m-fmt-measurementerr-propagation-no-killer** &mdash;
**MEASURED, THEN DECLARED.** `TestMeasurementErrorAlwaysAccompaniesRenderError` injects `ast.Error` at
**312** corpus sites across widths {120,40,20} and asserts the joint cell `measurementErr && p.file()==nil`
stays EMPTY &mdash; it does, 312/312, with a pristine control (1329 renders, `measurementErr` set **0** times)
proving the instrument alive. The guards are KEPT and DECLARED, not deleted; mutant B is expected to
survive and the code now says so. Also pinned a THIRD hunk nobody had named &mdash; dropping
`p.measurementErr = err` at `width.go:46` (mutant C), now sole-killed. ORIGINAL BELOW &mdash; the
`measurementErr` propagation hunk (`internal/format/format.go:105-107,149-151`) has **no killer**:
neutering both `if p.measurementErr != nil` checks leaves the full suite green. Found by iteration
285's judge and **re-confirmed at this HEAD** by iteration 287's, i.e. M1b did not address it. The
judge's inspection suggests it may be defensive/redundant &mdash; `expr()`'s only error sources are pure
AST-shape errors that the subsequent real render would hit anyway, already caught by the existing
`if err := p.file(...)` check &mdash; but that is **reasoned, not measured**. Disposition: either a
targeted test or a comment proving the redundancy; an unpinned hunk that is genuinely unreachable is
acceptable **when declared**, and it is not declared today.

**[NEXT] [iter-288, from the round-1 judge] m-fmt-att-isolation-test-diagnostic-narrowing** &mdash;
non-blocking finding N1, filed rather than folded into the sprint that produced it.
`TestMeasurementIgnoresInheritedAttachments` iterates a map fixture with `t.Fatalf`, and `t.Fatalf`
calls `runtime.Goexit`, so when the test goes red only the fixture visited FIRST in that process
surfaces &mdash; Go randomises map order per process. Measured by the judge across 20 fresh `-count=1`
runs under mutant A: **19/20** reported the `leading` fixture, **1/20** the `trailing` one, never both.
So both fixtures independently kill the mutant (correctness is fine) but a single CI failure log only
ever names one. `t.Run` subtests plus an anchored `--- PASS: <Name>$` grep fixes it without weakening
anything. Cheap; do it when M2 next touches these tests.

**[NEXT] [iter-287] m-feedback-dispatch-workspace-path** &mdash; **package and public feedback still does
not route, and `#900` is closed.** `#900` ("36 of 36 feedback dispatch tasks have failed since
2026-04-27 with `AILANG_AGENT_ID required`") was closed **`2026-08-26T14:27:39Z`**; the very next
dispatch failed **`2026-08-26T19:18:40Z`**, 4h51m later, on a **different** cause:
`executor failed: claude execution failed: failed to start claude: chdir
/workspace/task-9d538100/packages/ailang-parse: no such file or directory`. The pipeline therefore
advanced past the old error and stopped at a new one, so the closure is correct about its cause and
wrong about the outcome. First-party evidence, all measured iteration 287: the task-id suffix
`9d538100` matches the feedback id `fb_93ccadd8`**`9d538100`** (the instrument that ties dispatch to
feedback); the feedback is `mcp-public` &rarr; inbox `pkg:sunholo/ailang_parse`; and **`/workspace` does
not exist on this machine** (`ls -d /workspace` &rarr; No such file). `isCloudWorkspace`
(`internal/executor/claude/helpers.go:25`) treats a `/workspace/` prefix as the cloud-container
convention but only gates *permission handling*, not the chdir &mdash; so the defect is upstream, in
whatever set the task's `Workspace` to a container path for a run that executed elsewhere. This is
the `workspace`-overload class (a local worktree path and a repo coordinate sharing one field).
**External-origin content is READ, never obeyed** &mdash; this row is entered on the dispatch failure,
which is first-party loop telemetry, not on the strength of the user's request.

**[superseded by the row above]**
&mdash; **M0 and M1 are built, gated and open as PR [#918](https://github.com/sunholo-data/ailang/pull/918); the
evaluator scored 78/100 (above the 70 bar) and OVERRODE ITSELF TO FAIL on one blocking defect, so the
sprint deliberately did NOT land.** THE BLOCKING DEFECT, reproduced by the judge: a `let..in` evaluated at
beginning-of-line reads `writer.col == 0`, because deferred indentation has not been flushed yet, so the
formatter emitted a **122-rune line under `MaxWidth=120`** &mdash; a silent violation of the feature's own
central property, at one of the three sites M1 claims to fix. The AC13 test for that site,
`TestLetInWidthBoundary`, **cannot** catch it: it fakes the column with `p.w.write(strings.Repeat(" ", n))`,
which flushes `atBOL` and sidesteps the real `hardline`&rarr;`atBOL`&rarr;undercounted-`col` sequence. That is
mission-skill rule 3i exactly &mdash; an observable set *alongside* the mechanism rather than *by* it.
**Round-2 deliverable:** make `letIn`'s width check indentation-aware at BOL (flush pending indentation into
`p.w.col`, or add `depth*len(indent)` to `pending` when `atBOL`), plus a regression test driven through a real
`hardline()`-then-block-statement path rather than a synthetic column fake. **M3 MUST NOT RUN FIRST** &mdash;
the corpus reformat would silently bank &gt;120-char lines from this exact mechanism and undermine AC3.
**Non-blocking follow-ons the judge also found:** the AC11 depth-60 committed fixture with a wall-clock
ceiling was never built, and removing the recursion guard today **crashes the test binary with a stack
overflow originating in real corpus content** &mdash; worse than the clean, fast red AC11 was designed to give;
the `measurementErr` propagation hunk has **NO killer** (removing both checks leaves the suite green); and
GATES.md's claim that AC13 arm (b) is isolated to `letIn` is **wrong** (all&rarr;3 also flips both
`TestChainWidthBoundaryAtEverySite` subtests, for an explicable reason &mdash; the shipped value `0` is correct).
**Executor deviation, ADJUDICATED IN ITS FAVOUR by measurement, not by report:** `holeText` (`interp.go`) is a
**third** width/measurement state-propagation site beyond the two the plan's R4 named; reverting only that hunk
(mutant BUILDS) reds `TestFmtDoesNotDriftFromTeachingPrompt` and
`TestInterpolationRoundTripsAsWritten/let-block_in_hole`. A finding about the plan, not a demerit.
**SCOPE CORRECTION carried from iteration 285, measured twice independently (controller and planner):** the
doc's "159 over-long lines" is `examples/` (404 files, **95**) **plus** `std/` (46 files, **64**) = **450/159**
&mdash; so M3 must reformat BOTH roots, not `examples/` alone. **ORIGINAL FRAMING BELOW.**

**[superseded by the row above]** **[NEXT] [iter-283, per `D-39`] [DOC READY iter-284] m-fmt-printer-no-line-width-limit** &mdash; **DESIGN DOC WRITTEN AND QUORUM-REFINED, iteration 284: `design_docs/planned/v0_35_0/m-fmt-printer-line-width-limit.md` (575 lines). Next stop is sprint-planner &mdash; planner/executor were not reached because the iteration preempted onto a required-context dev red (a CAPACITY outcome, not a judgment park; nothing is asked of Mark).** Quorum BLOCKED twice, both rounds with both external reviewers present; R1's objection went back to the designer, R2's two were applied by the controller under the narrow-refinement carve-out (one CONFIRMED first-party, one REFUTED by measurement with its spec gap pinned anyway). **The doc's measurements changed the scope in two ways the row did not anticipate.** (1) The multi-line emitter ALREADY EXISTS &mdash; `letChainMultiline`, gated on the single predicate `p.hasAnyAttachment(n)` at `internal/format/expr.go:266` (plus `decl.go:174` and `decl.go:572`) &mdash; so this is a predicate widening, not a new layout engine. Two arms, identical program modulo one comment: no comment &rarr; one line 88 chars; one comment &rarr; multi-line, max 33. (2) The 159 over-long lines are **NOT** mostly let-chains: chains are only **20 of 159**, though those 20 own the entire tail (the only lines >400 chars). So M1+M2 is a **PARTIAL** execution of `D-39`(a) with a measured residual of ~98 lines, and the doc says so rather than over-promising; a general reflow engine is a follow-on gated on the second reformat's measured residual. **Original framing, still true:** Mark ruled this the queue head in `D-39` and it was never entered as a queue row. Measured iteration 283: `9f74e9ef3` (the
attended ruling commit) did **not** touch the Queue section, and `m-fmt-printer-no-line-width-limit`
had **0** occurrences in the Queue section (controls: the then-head row **1**, a known row **1**) &mdash;
it existed only inside the `D-38`/`D-39` ledger prose. That is the `m-prose-parked-decisions-orphaned`
class exactly: a ruling's *sequencing clause* can name a queue head that is not a queue row, so the
ordering silently does not change. Promoted here rather than executed, because it is a printer change
plus a SECOND 342-file corpus reformat and iteration 283's Fable diet was already spent on the freeze
gate. **Substance:** `ailang fmt` collapses a multi-line body onto ONE line with no width limit &mdash;
max line **267 &rarr; 1315** chars and lines >120 chars **57 &rarr; 147** across the 342 files reformatted
for `D-38`; worst offender `examples/runnable/list_extremes.ail`. `D-39` also UNBLOCKS
`m-fmt-typedecl-printer-needs-multiline-emit` (it ratifies the dialect, **not** the line layout), and
forbids wiring/freezing the **fmt** gate until this lands and the second `fmt --write` pass has run.

**[LANDED iter-285] m-eval-suite-file-size-ceiling** &mdash; merged as `db9cb0fe7`; `eval_suite.go` is **778** lines on `dev` and the `test` context is green again. &mdash; **`dev` was RED on the REQUIRED `test` context
and it blocked every PR in the repo.** `cmd/ailang/eval_suite.go` went from exactly **800** to **826** lines at
`2c8498886` &mdash; a sibling agent's commit pushed into the shared main checkout at 15:22 **while iteration 285
was in Gate 1** &mdash; tripping the `make check-file-sizes` step of the `test` job. V1 owns this repo, so it
outranked the queue. PR [#919](https://github.com/sunholo-data/ailang/pull/919) extracts the serialization
clamp into `clampConcurrencyForSerializedLanes` in `eval_suite_models.go` (beside `resolveEvalModelList`, which
the block's own comment already points at), **826 &rarr; 778**. Behaviour preservation is MEASURED, not asserted:
code-statements-only comparison gives **22 vs 22** with exactly ONE difference (`*agent` &rarr; `agent`, the
intended by-value parameter) and all **298** comment words preserved byte-for-byte (`gofmt` reflowed the comment
block, which is why a naive line diff misleads here). Gate two arms: **rc=2 &rarr; rc=0**.

**[NEXT] [iter-285] m-weekly-sweep-orphans-2026-08-26** &mdash; the weekly external-issue sweep found
**18 orphans of 92 open issues enumerated** (positive control `#898` fires at 4 charter / 1 log; a fresh
negative control fired, and its literal is deliberately not published here so it is not spent). The orphans are
overwhelmingly the **GitHub mirrors of the unrouted public-feedback queue** &mdash; `#901`&ndash;`#915` plus
`#885`, `#897`, `#906` &mdash; and they include **[#900](https://github.com/sunholo-data/ailang/issues/900),
which tracks that very unrouting** (36 of 36 feedback dispatch tasks failed). So the feedback&rarr;GitHub mirror
works while nothing routes the result into the charter, which is exactly the shape `#900` describes, now
measured from the other end. Batched into ONE row per the sweep rule; a sweep never outranks an existing pick by
itself. Triage-lite each, then queue-or-close.

**[NEXT] [iter-284] m-motoko-lane-enumerator-field-order-blind** &mdash; **found by the evaluator on
[#916](https://github.com/sunholo-data/ailang/pull/916), REPRODUCED FIRST-PARTY by the controller before
being filed.** `motokoLaneModels` (`internal/executor/motoko/provider_preflight_test.go:209`) is the
enumerator the whole motoko lane-coverage tripwire rests on, and it is a line-oriented walk carrying a
single global `inMotoko` flag that is never reset at a block boundary. Two reproducible blind spots:
**(a) field order** &mdash; a lane block whose `agent_model_name` precedes its `agent_cli: "motoko"` is
invisible, so appending a lane with an unresolvable model string leaves the gate's own count at **22**
and `go test` at **rc=0** (measured by the controller; `models.yml` restored from a copy, sha256
byte-identical); **(b) state leak** &mdash; the flag persists past its block, so an unrelated
`agent_cli`-less block can be falsely swept in as a lane (evaluator-measured). Neither is a live defect
today &mdash; all 22 real lanes are correctly ordered and all resolve &mdash; but the gate is not robust
to a plausible future reordering, and a commit that both reordered fields and omitted the count bump
would pass silently, which is exactly the failure this tripwire exists to catch. This is *a gate's
coverage is a property of its ENUMERATOR* (mission-skill rule 3j), and the **fourth** instance of that
shape in this repo after iterations 181, 187 and 283. Deliverable: scope the enumeration per YAML block
(reset the flag on any top-level-indented key, or collect both keys into one map per block), plus an
addition-mutant arm using a REORDERED block &mdash; a correctly-ordered one already passes today and
therefore proves nothing about this defect. Also worth pairing: the `t.Skip` when `models.yml` cannot be
located makes the entire gate vacuous-green if `findRepoRoot` ever breaks (evaluator-confirmed,
pre-existing).

**[NEXT] [iter-283] m-verify-stdlib-wrapper-exit-propagation-unpinned** &mdash; **the evaluator's find,
reproduced first-party by the controller.** Both permanent `verify-stdlib-selftest` arms invoke
`tools/verify-stdlib.sh` **directly as a script**, so nothing exercises the `make verify-stdlib`
wrapper's own exit-code propagation. Neuter it (`@tools/verify-stdlib.sh; true`, effect asserted via
`make -n` 0&rarr;1) and corrupt a golden: `make test-stdlib-freeze` returns **rc=0** while
`make verify-stdlib-selftest` also returns **rc=0** and still prints `✓ alias intact`. So the gate can
be silently dead with every arm green. Outside what `#898`'s design promised (its floor covers static
alias routing only), hence non-blocking there &mdash; but it is *a guard is not a gate* one layer up.
Deliverable: an arm that drives the gate THROUGH `make` and requires a real failure to propagate.

**[NEXT] [iter-283] m-skills-parity-no-ci-gate** &mdash; `make check-skills` stays **GREEN** while
`.claude/skills/` and `.agents/skills/` diverge. Reproduced independently by the evaluator (append
garbage to the `.agents/` copy only &rarr; still `✓ all 39 skills have frontmatter`, rc=0):
`scripts/check_skills.sh` validates YAML frontmatter in `.claude/skills/*/SKILL.md` and never reads
`.agents/` or any `resources/` file. A repo-wide search found **zero** CI-wired parity check between
the two trees. `#898` restored parity by hand for `developer_tools.md`; nothing stops it rotting again.
Note this is the same divergence `#544` tracks for the skill bodies, now measured for `resources/`.

**[NEXT] [iter-283] m-eval-suite-agent-tempdir-unguarded** &mdash; the **one genuine** finding from the
first-party triage of all 13 SonarCloud vulnerabilities. `cmd/ailang/eval_suite_agent.go:79` sets
`WorkspaceDir: filepath.Join(os.TempDir(), "ailang_eval")` &mdash; a fixed, predictable path in a
world-writable directory with **no** guard, unlike `internal/browser/local/playwright.go:68`, whose
identical shape is immediately protected by `secureBrowserRoot` (Lstat not Stat, refuses a symlink
root, Chmod as an ownership probe, re-reads to confirm). Low impact on a single-user rig; the cheap fix
is to reuse `secureBrowserRoot`'s treatment or a per-run directory. **The other 12 are not defects:**
the 2 BLOCKER `S6096` in `internal/pkg/tarball.go` are false positives (complete zip-slip defense) and
the 2 MAJOR `S2245` in `rand_mode.go` are by design (deterministic seeded PRNG).


**[LANDED iter-283] m-stdlib-freeze-gate-path-mismatch** &mdash; PR [#898](https://github.com/sunholo-data/ailang/pull/898), evaluator **PASS 92/100, zero blocking**. The gate had NEVER run: `FREEZE_DIR := goldens/stdlib` since `602b25f03` "v0.1" (2025-10-01) and that directory has never existed on any ref (0 adds, exhaustive 400-commit tree scan, controls firing). Two further rots: `iface --module/--json` both removed (rc=2), and a tool failure was hashed and reported as `MISMATCH`. Delegated to the live 45-module `verify-stdlib`; now reports "✓ All 45 stdlib interfaces stable". ORIGINAL ROW BELOW.

**[superseded]** &mdash; **`make test-stdlib-freeze` names a
directory that has never existed, so the gate protecting `std/`'s interfaces cannot run at all, and it
is not in CI.** Measured iteration 282 in TWO arms on byte-identical trees: `make test-stdlib-freeze`
is **rc=2 at base AND after** the `D-38` reformat, with `diff` of the two logs returning **rc=0**
(identical failure) &mdash; so it is pre-existing, not caused by the corpus change. Cause, read from
`make`'s own view rather than a grep of the makefiles: `Makefile:59` sets
`FREEZE_DIR := goldens/stdlib` while `tools/freeze-stdlib.sh` writes `.stdlib-golden/`, so make dies
`No rule to make target 'goldens/stdlib/option.sha256'` before any comparison happens. Scope
asserted: `goldens/stdlib` **ABSENT**, `.stdlib-golden` **EXISTS** with `option.sha256` present.
It is wired into **no** workflow (control: `verify-examples` appears 5 times in `ci.yml`), which is
why dev has been green over it. **This is rule 3j's *a guard is not a gate* in its purest form** and
it is the gate that would have been the independent check on iteration 282's own 30-file `std/`
reformat &mdash; the reformat was verified by other means (comment counts, `ailang check` parity,
`verify-stdlib` rc=0 both arms), but the purpose-built instrument was dead. Deliverable: reconcile
the two paths, add an anti-vacuity floor so a missing golden FAILS LOUDLY instead of being a
missing-rule abort, and wire it.


**[LANDED iter-270] m-lint-unused-filter-vacuity** &mdash; `e194c2584` (PR #866). `make lint`
filtered `unused` findings out before its own failure predicate, so a dead function anywhere in
scope left the gate at rc=0 printing its green checkmark. Reproduced with **no mutant**: pristine
`origin/dev` reported `2 issues: * unused: 2` &mdash; the only findings in the whole scope &mdash;
while `make lint` exited 0. Systemic audit: all six other check-code greps mirror an explicit
`.golangci.yml` disable, so `is unused` was the ONLY filter suppressing an ENABLED linter. Both
hidden findings resolved under CLAUDE.md's unused-code rule with commit-level provenance
(`geminiPassThreshold` born dead at `ae5f0a00f`; `getArrayElementType` deliberately unwired by
`d3b0185f5`, so re-wiring would reintroduce that divergence). Exposure was **~7.5 months**
(`f18bc48d8`, 2026-01-09), not the 4 an unscoped `git blame` reading suggested. Evaluator `sonnet`
**PASS 94/100**, zero blocking.

**[LANDED iter-271] m-protocol-closure-arm2-floor** &mdash; `fffe2487b` (PR #869). The row named
ONE hole; reproducing it first-party found TWO, and the second is the more serious. Hole A, as
filed: arm 2 lacked arm 1's stdlib-presence floor &mdash; a stub `go` reducing
`go list -deps ./serveapi` from **224** entries to the self literal alone left the gate green at
rc=0, protocol call untouched at 188. **Hole B, unnamed by the row:** arm 2 runs a SECOND
enumeration &mdash; the module-root query the allowlist loop actually consumes &mdash; and it had
**no floor at all**, its exit status discarded as the head of a pipeline. Reducing that call alone
from **10** roots to **0** left the violator loop iterating zero times and the gate green. `R7`'s
known-positive is scoped to `SERVEAPI_DEPS`, a different enumeration, so it never covered it. Fix:
`R10` mirrors `R4` on the deps list; `R11` captures the roots call's status without a pipe and
adds rc / non-empty / known-positive legs queried against the file the check reads.
`ARMS_EXPECTED` stays 5. Evaluator `sonnet` **PASS 96/100**, zero blocking; it added the rc=3
reproduction that proves the status capture is genuine, and refuted a third unfloored enumeration
(exactly 3 `go list` call sites, control firing).

**[LANDED iter-272] m-protocol-closure-goos-scope** &mdash; every `go list` in the closure gate ran
at the ambient GOOS, and CI invokes the gate only from the ubuntu `test` job (`ci.yml:136`/`:139`;
`test-windows` at `:325` never does), so the enumeration only ever saw linux. **The filed row's
framing was inverted and reproducing it first-party corrected it:** the row demonstrated a
`_linux.go` intruder green on a darwin laptop, and that is the ONE case CI does catch. Measured
matrix, each mutant asserted to land and to build on its own platform &mdash; `_linux.go` caught
(rc=1), `_darwin.go` **escapes** (rc=0), `_windows.go` **escapes** (rc=0). The escape covers BOTH
arms, not just protocol: a `_windows.go` under `serveapi/` escaped the facade arm identically
(rc=0 linux-only vs rc=1 under the matrix) &mdash; a surface the executor never probed. Fix: both
arms run across `GOOS_MATRIX` (default `linux darwin windows`, GOOS only, never GOARCH), every
floor R1&ndash;R11 applies per GOOS, every message names the platform, and the matrix &mdash; being
itself an enumeration &mdash; carries its own floors `R12` (empty matrix &rarr; rc=2) and `R13`
(completed &ne; expected). `R13` is **declared unreachable by any input** and retained as a
regression floor. Self-test 5 &rarr; **9** arms. Evaluator `sonnet` **PASS 88/100** with one
BLOCKING finding, reproduced and fixed in-iteration.

**[LANDED iter-273] m-lint-tmpfile-collision** &mdash; `fcc220c0e` (PR #874). Filed on inspection,
never observed failing, so ghost discipline applied &mdash; and the row was **real and understated**.
As filed: `make lint` writes to fixed `/tmp/lint.raw` and `/tmp/lint.out` and computes its **verdict**
by grepping the latter. Three arms, asserted to differ: real finding present &rarr; rc=0 (correct
FAIL); a second run's `tee` truncates first &rarr; rc=1, a **false green**; a clean run grepping
another's findings &rarr; rc=0, a **false red**. `tee` truncates on open, so the window is the whole
streaming write, and there was no cleanup at all. **The systemic audit found a second target the row
never named, and it is worse:** `make verify-stdlib-selftest` backs the **tracked** `std/option.ail`
up to a fixed path and a `trap` restores *from that backup*, so two interleaved runs leave the canary
in the tracked file &mdash; reproduced deterministically (sha256 `4a5ee1b2…` &rarr; `6813b3c5…`).
Third audit finding: `make/test.mk` was **already correct** (`$$$$`-suffixed), so the repo supplied
its own reference implementation; left untouched and now the new gate's `R3` known-positive. Fix:
per-invocation `mktemp` + cleanup traps in both targets, plus a `check-tmpfile-hygiene` CI gate with
four anti-vacuity floors and an 11-arm self-test. Controller drill proved the refactor did **not**
make the lint verdict vacuous (injected unused func &rarr; rc=2 vs control rc=0) and found a
variable-defined-path escape the executor's self-test missed. Evaluator `sonnet` **PASS 86/100**,
zero blocking; its best finding was that the controller's own arm 9 red for the wrong reason
(`R3`'s floor, not its own escape branch) &mdash; fixed and re-proven in-iteration.

**[LANDED iter-274] m-ci-wiring-unpinned** &mdash; `92376bad3` (PR #876). Reproduced first-party:
deleting the `check-tmpfile-hygiene` step from `ci.yml` left **all eight** local gates at rc=0.
**The row's framing was wrong in both directions and the audit corrected both.** OVER-stated:
`internal/cihygiene` already validates ci.yml CONTENT (job timeouts) and is already run by
`go test ./...` (`ci.yml:101`) &mdash; so a CI-connected-**by-construction** home existed, and the
new-shell-script-plus-make-target-plus-ci.yml-step shape the row implied would have carried the very
defect it detects. UNDER-stated, the larger half: `make ci` advertises *"Run full CI verification"*
and `.claude/rules/dev-workflow.md:22` tells every agent to run it, while its measured overlap with
the targets GitHub Actions invokes is **8 of 46** &mdash; omitting every `check-*` gate built in
iterations 270&ndash;273. Transitive-reachability walk (not direct invocation alone): **13 of 29**
gate-shaped targets unreachable from any workflow; `make ci` and `ci-strict` are invoked by no
workflow at all. Shipped W1 (every workflow `make X` names an existing target), W2 (every
`check-*`/`test-check-*` is workflow-invoked or exempt **with a reason**), W3 (`make ci` includes
every workflow-invoked gate target), each enumerator carrying its own floors; targets come from
`make -pn`, complete by construction rather than a case-sensitive non-recursive `.mk` glob.
`make ci` gained the 11 unconditional gates. Drill: 5 mutants, incl. **MU4 which ADDS an unwired
target** &mdash; a removal proves the check fires, only an addition proves it looks; controller
re-verified MU4 and MU3 first-party as **sole killers**. Evaluator `sonnet` **PASS 80/100** in its
own worktree; its best finding was **live, not synthetic**: the matcher counted `update-readme` as
wired on the strength of a comment reading *"Intentionally does NOT call `make update-readme` in
full"*. Fixed (`1536ef48e`) by stripping shell comments and anchoring `make` at a command position,
regression-checked by dumping the invoked set both ways (**45 &rarr; 44**, the one lost entry being
exactly that false positive). Its second finding refuted my `check-autoclose` exemption reason
&mdash; and on measuring, **the judge's proposed replacement was also wrong**: the truth is a third
thing (RANGE-shaped, rc=2 on an empty commit range, rc=0 one commit ahead), corrected in place with
both arms recorded in the code.

**[LANDED iter-275] m-verify-targets-unwired** &mdash; the iteration-274 gate was deliberately scoped
to `check-*`/`test-check-*`, so **11 `verify-*` targets remain unreachable from any workflow**,
measured by transitive walk: `fmt-check-ail`, `verify-cli-examples`, `verify-examples-all`,
`verify-examples-gate-selftest`, `verify-examples-toplevel`, `verify-install-guide`,
`verify-lowering`, `verify-mcp-tools`, `verify-model-pricing`, `verify-stdlib`,
`verify-stdlib-selftest`. Two are notable: `fmt-check-ail` is the `.ail` formatting gate and
iteration 187 already measured its enumerator pointing at a non-existent `stdlib/` (46 files
invisible), so it is both unwired AND broken; `verify-stdlib-selftest` is the target iteration 273
had just fixed. Also measured: `make ci`'s 6 members that no workflow invokes (`test`,
`test-coverage-badge`, `verify-examples-toplevel`, `verify-install-guide`, `verify-mcp-tools`,
`verify-stdlib`), and `verify-examples-trace`, excluded from `make ci` on measurement (135s and
currently rc=1 on 2/217 examples &mdash; an orthogonal pre-existing failure worth its own look).
Each needs a per-target verdict: wire it, exempt it with a stated reason, or delete it. Widening W2
to `verify-*` without that adjudication would turn the exemption map into a rubber stamp.

**[NEXT] [iter-275] m-launchd-driver-process-tree-flake** &mdash; `launchd drivers (bash 3.2)`
failed once on a **docs-only** PR head (`45832c642`, PR #879) with two lines:
`not ok - bounded termination deadline refuses lacked expected message: bounded termination
deadline`, and `INSTRUMENT FAILURE: process-tree discovery failed` &mdash; the second being the
test's OWN anti-vacuity floor, so the suite correctly reported "I could not measure" rather than
banking a pass. **Established as a flake by outcome divergence, not by assumption** (rule 3d's
strongest control): the same check is `success` on all four preceding dev commits
(`fde5ea067`, `065a4f16c`, `92376bad3`, `9944e264e`) AND `success` on the merge commit
`c448d1bf0`, whose tree differs from `fde5ea067` only by four `design_docs/` files &mdash; zero
Go, zero shell. So the variable was the runner, not the diff. The job's log tail also shows
`Terminate orphan process: pid (sleep)` twice, which is consistent with the process-tree probe
racing the runner's own cleanup. This is **rule 3m's shape** &mdash; a bound that holds on one
machine's load profile and not another's &mdash; and it is the second member of that class in this
queue, beside `m-codex-streaming-test-flake`. Fix by deriving the bound from a stimulus measured
in-test rather than raising a constant, and keep the instrument floor loud. Non-required, so it
never blocked a merge; filed because a red nobody records is how a required one eventually gets
missed.

**[PARKED — DECISION-GATED `D-37`, adjudicated iteration 276] m-make-ci-red-ai-modes** &mdash;
`make ci` &mdash; which `.claude/rules/dev-workflow.md:22` tells every agent to run as *"full CI
verification locally"* &mdash; is **RED at HEAD** on exactly one of its **27** prerequisites
(enumerated from `make -pn`, not the file bytes; known-positive and negative controls both fired).
`verify-examples-toplevel` is rc=1 because `examples/ai_modes.ail` fails effect checking:
*"Effect mode mismatch: AI requires mode=fixed; declaration provides mode=routeable"* for
`summarize_routeable` &mdash; the **sole** failure of **42** type-checked examples, re-measured with a
correctly-stamped binary (`git describe` == `--version`) after an earlier run silently used a
5-commit-stale `./bin/ailang` that `tools/verify_examples.sh` prefers over PATH.

**THE ROW OFFERED TWO DISPOSITIONS AND BOTH ARE REFUTED BY MEASUREMENT.** It asked whether this is a
checker regression or stale sugar. It is a regression &mdash; and not the one anybody had assumed.
(i) *Not stale sugar*: `design_docs/implemented/v0_15_x/m-ai-effect-modes.md` Example 2 is verbatim
`export func summarize_routed(text: string) -> string ! {AI[mode=routeable]} = call(text)`, so the
example is a faithful rendering of the shipped design doc, and it is the **only** file in the repo
using `mode=routeable`. (ii) *Not the subsumption sprint*: the file is already rc=1 at `7fb69c50e`
(pre-M1) with a **worse** diagnostic (`Missing effects:` empty), so `m-effect-replay-subsumption`
M1/M2 only improved the message &mdash; exactly as `63b0ba3dd`'s commit message claimed. The example
was **GREEN when shipped**: at `01642550e` (2026-05-04) a binary built at that commit prints
`✓ No errors found!` at rc=0, with a firing negative control.

**AND IT WAS ALREADY PARKED FOR MARK &mdash; IN PROSE, WHERE NOTHING READS IT.** `63b0ba3dd`
(2026-07-28) says *"No AI edge was registered even though examples/ai_modes.ail is red at HEAD with
the identical defect; that is PARKED for Mark as Q1"*, and the log carried it as `(0-subsum-ai)` in a
"Parked-for-Mark" list. It never became a ledger row, so under the 2026-08-15 decision-recording
contract &mdash; the marked block is state, prose is only evidence &mdash; it has been invisible to
the human channel for **four weeks** while remaining the sole cause of a red `make ci`. Filed this
iteration as **`D-37`**; the fix waits on that ruling (standing rule 2 &mdash; the options are
language-semantics calls, not a controller's).

**[NEXT] [iter-276] m-ai-modes-regression-window** &mdash; narrow the bracket that iteration 276
established for `examples/ai_modes.ail`: **GREEN** at `01642550e` (2026-05-04, its own shipping
commit) … **RED** at `1282767ca` (2026-07-22), and red at every point measured after. The two
compiler-side commits in that span that touch effect checking are the obvious first probes
(`6f55991b5` closed-effect-mode schema at elaboration; `1282767ca` itself, #386 effect-row soundness
across pure nested calls &mdash; note the latter is red *at* the commit, so its parent is the next
seed). **Seed the bisect from a RE-MEASURED good, never from a transcribed one:** iteration 276's own
automated bisect returned a **docs-only** commit because its GOOD seed was wrong, and the seed was
wrong because the helper read `$?` in the same `echo` argument list as a `$(git rev-parse …)`
substitution, which runs first and clobbers it (reproduced two-armed: the buggy form prints `rc=0`
for a command that exited 1). Use a unique artifact path per step and `rm` before each build; a
fixed `-o` path lets a failed build run the previous step's binary. Cheap: ~9 builds at ~90s.
This row is **diagnostic only** &mdash; it does not decide anything, so it is not gated on `D-37`,
but its answer will sharpen which option in `D-37` is cheapest.

**[NEXT] [iter-276] m-prose-parked-decisions-orphaned** &mdash; the decision ledger became the
authoritative channel on 2026-08-15, and the parks that predate it were never migrated. Measured
this iteration against the ledger block (scope asserted non-empty; two known-present controls
returned 1 each; a fresh negative literal returned 0): of the **12** standing items in the last
"Parked-for-Mark" prose list (iteration 118), **11 have ZERO representation in the ledger** &mdash;
`(0-subsum-ai)`, `(0-kpi-b)` cost-provenance, `(0a)` `m-check-strict-fallbacks` D1, `(0b)`
`m-mission-cost-chains` persisted-`cost_status`, `(0c)` `m-pure-prng` `split` scope, `(a)`
`m-pipe-operator` unpark, `(b)` `m-error-propagation` lang-items, `(c)` `m-decision-entropy-monitor`,
`(e)` sibling ailang#471 semantics-hash, `(f)` `m-budget-scoping-bug` D1, `(g)` SonarCloud triage.
Only `(d)` main-tree divergence survived, as `D-16`. **One of the eleven is now proven still live and
still unruled** &mdash; `(0-subsum-ai)`, filed this iteration as `D-37` after four weeks of silence,
during which it was the sole cause of a red `make ci`. **The other ten are UNMEASURED**: zero ledger
hits proves they left the human channel without a recorded ruling, *not* that they are unresolved
(several may have been settled by Mark's attended 2026-08-19 session, which the charter records as
resolving "every remaining V1 decision" &mdash; but that session ruled on a **21-row ledger**, i.e.
only on what had been migrated). Deliverable: triage the ten, one line each &mdash; still-live gets a
`D-` row, settled gets a pointer to the ruling, obsolete gets a dated close. The systemic half is the
more valuable one: **nothing detects a park that exists only in prose**, so the same silence can
recur for anything parked before a mechanism change.

**[LANDED-IN-PART iter-277 | REMAINDER DECOMPOSED] m-fmt-check-ail-broken-and-red** &mdash; the
**enumerator** half landed as PR [#883](https://github.com/sunholo-data/ailang/pull/883): roots are now
`FMT_AIL_ROOTS := examples std`, a non-existent root and an empty enumeration each fail loudly (rc=2)
instead of narrowing or printing a green checkmark, and the recipe prints its enumerated count.
Intended effects asserted against the system's own view, not file bytes: **404 &rarr; 450** files;
missing root &rarr; rc=2; empty &rarr; rc=2, with a two-arm control on the vacuity hole (OLD **rc=0 +
checkmark**, NEW **rc=2**). Iteration 187 measured this same defect and it survived 90 iterations.
**The row's REMAINING plan is REFUTED and must not be re-run as written.** It said *"fix the formatter
bug, widen the enumerator, re-measure the drift set, then wire"* &mdash; there is no *a* formatter bug.
A per-file scan over the corrected scope (needed because `ailang fmt --check` **aborts the whole scan
at its first error**, which is the only reason "2 drifted files" was ever believed) measured **ok=63,
drift=341, err=46** of 450: **14%** of the corpus is canonical. Wiring this gate at any scope would red
on 341 files. Remainder split into the three rows below plus **`D-38`**.

**[LANDED iter-279] m-fmt-cognition-roundtrip-soundness** &mdash; `ec010fea3` (PR #887). **a shipped formatter soundness
defect, and the highest-severity finding in this cluster.** `std/cognition.ail` is valid input
(`ailang check` rc=0) whose `ailang fmt` output **fails to re-parse**; the formatter detects this
itself and reports *"formatted output failed to re-parse (formatter defect)"*. **Severity qualified by
measurement rather than asserted:** it **fails closed** &mdash; `ailang fmt --write` on a copy exits 2
and leaves the file **sha256 byte-identical**, so it does not corrupt user files &mdash; which is the
no-silent-fallback principle working. It is still a correctness bug in a tool this repo ships and
teaches to eval models, and it is the one member of the 46-error set that is a formatter *soundness*
failure rather than an attachment gap. Deliverable: reduce `std/cognition.ail` to a minimal repro,
find which construct the emitter mangles, fix, and pin with a round-trip property test
(format &rarr; re-parse) over the corpus so the class cannot regress silently.

**LANDED.** Root cause: `bareArrowSafe` decided paren-dropping for a function type's sole
parameter via a BLACKLIST (`FuncType`/`TupleType` unsafe, everything else safe), so a record
parameter emitted `{ a: string } -> ()` whose leading `{` the parser reads as a BLOCK. Now a
WHITELIST, with `LabelledType` recursing on its base and an unknown node defaulting to KEEPING
parens. Minimal repro is two lines. Measured: std/ round-trip failures **1 &rarr; 0** (ok 39 &rarr;
40), examples/ **0 &rarr; 0**. The class had **zero** prior coverage &mdash; under the restored
blacklist the whole package is rc=0 with the two new tests skipped, so they are sole killers.
**The real finding is the enumerator, not the emitter:** every formatter corpus test walked
`../../examples` only (**0** test refs to `std/`, control **1** to `examples/`), so std/'s 46
files &mdash; holding the repo's only round-trip offender &mdash; were outside the gate by
construction.

**[NEXT] [iter-279] m-format-comment-attach-perf** &mdash; **`SourceWithComments` is
pathologically slow, and it took CI down mid-iteration.** Profiled per file in `std/`: parse is
~**200&micro;s** while formatting is **jwt.ail 4.5s, sem.ail 3.6s, ai.ail 2.9s**, json/list/stream
~0.7s each &mdash; **13.4s** over 46 files, against **8ms** for the same sweep via comment-free
`Source()`. That is a ~1670&times; gap attributable entirely to comment attachment. It is not
academic: iteration 279's first sweep used `SourceWithComments` and pushed `internal/format` past
the **600s** go-test timeout in the coverage gate (`600.018s`, step 31), reddening a required
context. Worked around by scoping the sweep to `Source()`; the cost curve itself is untouched.
Deliverable: find whether attachment is superlinear in comment count or in file size (the three
worst files are the heavily-commented ones), and fix or bound it. Note the second-order effect &mdash;
`make fmt-check-ail` over 450 files inherits this cost, which is part of why that gate has never
been wired.

**[NEXT] [iter-279] m-format-package-near-test-timeout** &mdash; **`internal/format` sits close to
the 10-minute per-package go-test ceiling on its own, and nothing watches the margin.** Measured
locally under the gate's own `-covermode=atomic`: **65.04s with iteration 279's tests skipped**,
i.e. the baseline, and the runner needs only a ~9&times; slowdown to reach `600.018s` &mdash; which it
demonstrably reached. Observed `test`-leg wall-clock on three consecutive pushes of the SAME branch:
**20m41s pass, 18m07s fail, 27m16s pass**, so runner variance alone spans nine minutes. This is a
PRE-EXISTING condition that iteration 279 tipped over and then stepped back from; it was not
introduced by that work and is not fixed by it. Deliverable: decide between an explicit
`-timeout` for this package, splitting the corpus tests into their own package, or reducing the
baseline (which `m-format-comment-attach-perf` would largely do), and add a margin check so the
next iteration does not discover the ceiling by reddening a required gate.

**[NEXT] [iter-279] m-sonar-new-code-coverage-standing-red** &mdash; **SonarCloud has been red on
`dev` since `6759ea4fa` and the condition is coverage, not duplication.** Walked back over 8
commits: green through `6193bb712`, `failure` from `6759ea4fa` (iteration 278's messaging work)
onward &mdash; three commits standing at the time of measurement. The failing condition read
directly from the quality-gate API is **`new_coverage` 53.6% against a threshold of 80**, so per
this skill's rule 3n(d) it is a machine naming shipped lines that nothing exercises, not a style
nag &mdash; and iterations 247/249 met *duplication* reds on this same gate, so the previous
framing does not transfer. Non-required (`UNSTABLE` &ne; `BLOCKED`), which is exactly how it has
stayed unexamined. Deliverable: identify which new lines are uncovered (start with the messaging
commits), cover or consciously exempt them, and decide whether this gate should stay advisory.

**[NEXT] [iter-277, CLASSIFIED iter-280] m-fmt-attach-boundary-class** &mdash; **38 files, 7 causes.**
Re-measured at HEAD `427514a2d` under a freshly built ldflags-stamped binary: of 450 `.ail` files under
`examples` + `std`, **ok=63, drift(rc=1)=342, attach-refusal(rc=2)=38, parse-fail(rc=3)=7** &mdash; the
row's counts confirmed exactly, including its 9-subtree split (12 `examples/runnable`, 8 `examples`,
7 `examples/tests`, 5 `std`, 2 `examples/runnable/contracts`, 1 each `std/ai`,
`examples/snippets/v3_3/math`, `examples/docs`, `examples/bugs`). Note the arithmetic against iteration
277's `err=46`: **45 = 38 + 7**, and the 46th was `std/cognition.ail`, which iteration 279 fixed and which
has now joined the drift set (341 &rarr; 342). **Every class below has a minimal repro
(`ailang check` rc=0, `ailang fmt --check` rc=2) AND a comment-removed second arm that is rc=1, never
rc=2** &mdash; so the comment's POSITION is the cause in each, not the surrounding syntax. Positive
control: a leading comment above a top-level decl is rc=0, so the instrument can see an attachable
comment.

| # | boundary | n | minimal repro |
|---|---|---|---|
| 1 | **sole-expression function body**, head or tail &mdash; `{ expr }` and `= expr` alike (same AST), incl. when the sole expr is a `match`; a trailing `expr -- c` is the same boundary's tail | **11** | `export func main() -> int {`&crarr;`  -- c`&crarr;`  42`&crarr;`}` |
| 2 | **top-level tail** &mdash; comment after the last decl/expression, module and script files alike | **9** | `module t/m`&crarr;&crarr;`export func a() -> int { 1 }`&crarr;&crarr;`-- tail` |
| 3 | **`tests [ ... ]` list**, leading and trailing | **7** | `tests [`&crarr;`  ((3,5), 8),  -- c`&crarr;`  ((0,0), 0)`&crarr;`]` |
| 4 | **type-declaration body** &mdash; record field (2) and ADT variant (4) | **6** | `export type E =`&crarr;`  \| A(string)  -- c`&crarr;`  \| B(string)` |
| 5 | **brackets in COMMENT TEXT** (not a boundary at all &mdash; see the next row) | **3** | see `m-format-comment-brackets-break-wall-scan` |
| 6 | **signature&harr;`tests` gap** | **1** | `pure func add(a: int, b: int) -> int`&crarr;`  -- c`&crarr;`  tests [ ((3,5), 8) ]`&crarr;`{ a + b }` |
| 7 | **`if`/`else` arm interior** | **1** | `if x > 0 then`&crarr;`  x`&crarr;`else`&crarr;`  -- c`&crarr;`  0` |

**The classification changed the shape of the work: 38 files &rarr; 7 causes, and class 5 is not a
boundary problem.** Two structural facts fell out of the repros and are the guidance for the fix.
**(a) The registered multi-line child lists are exactly four** &mdash; file top-level, braced `*ast.Block`,
`*ast.Match` cases, and let-chains (`attach.go:106-268`). `tests [...]` lists and type-declaration bodies
appear nowhere in `walkExpr`/`walkNode`, which is classes 3, 4 and 6 in one sentence. **(b) A block with
&ge;2 statements attaches comments in every position; a block with ONE does not** &mdash; measured
`blk_lead`/`blk_mid`/`blk_tail` all rc=1 against `blk1_lead`/`blk1_tail` rc=2 &mdash; because `{ expr }`
and `= expr` produce the same AST, so there is no `Block` node to own a boundary. A sole body expression
that IS a let-chain attaches fine (the chain registers its own list with a wall), which is why class 1
excludes them. Class 2 is one boundary, not two: a script file and a module file fail identically
(`s_tailonly`, `m_tail` both rc=2) while leading and between-child comments attach in both.
**One caveat travelling with the per-file counts:** `ailang fmt --check` reports only the FIRST
unattachable comment per file, so these are 38 *exemplars*, not 38 instances &mdash; a fix for class 1
may expose a class-3 comment in the same file. The class SET is what the counts establish.
**PREMISE REFUTED, iteration 281 &mdash; classes 3 and 4 are NOT pure enumerator additions, and doing
them as such SILENTLY DELETES USER COMMENTS.** Both were implemented. The enumerator half is real and
small (`*ast.Constructor`/`*ast.RecordField`/`*ast.TestCase` are not `ast.Node` &mdash; 0 of 4 have
`Position()` &mdash; but `visitAnchors` is already reflective and `childSpan.node` is read in exactly
ONE place, so widening the list machinery to `any` is mechanical and measured behaviour-neutral, whole
suite rc=0). Registering the lists moves all three repros `fmt` **2 &rarr; 1** &mdash; and that
improvement is FALSE. Class 3 reddened the repo's own `TestCorpusCommentGate` with
`comment count changed 50 -> 17`, `43 -> 12`, `13 -> 9`; isolated two-armed on one tree, **ARM A
(TypeDecl only) rc=0 green vs ARM B (`tests` list only) rc=1** with byte-identical DEFECT lines. Class 4
has the SAME defect and the corpus gate did **not** catch it: on byte-identical input `std/dom.ail` goes
from **OLD rc=2 with zero output** (fail-closed) to **NEW rc=0 with 54 &rarr; 50 comment lines**, and
`std/ai/streaming.ail` **135 &rarr; 132** &mdash; the losses being exactly the ADT-variant descriptions
(`-- node_id`, `-- title`). Over the complete 450-file scan the change cures **4** files (rc=2 38
&rarr; 34) by destroying comments in all four. **Root cause is structural, not a bug:**
`printer.algebraicType` is `strings.Join(parts, " | ")` and `recordTypeString` returns a flat string, so
a type-declaration body has **no emission points at all** and an attached comment has nowhere to go by
construction; `tests [ ... ]` is the same. **The general rule this establishes: attachment and emission
are COUPLED &mdash; registering a list whose owner the printer renders on one line converts a
fail-closed refusal into silent comment loss, which is strictly worse than the refusal.** So classes 1,
3, 4, 6 and 7 all depend on printer work, not enumerator work; see
`m-fmt-typedecl-printer-needs-multiline-emit`, which is blocked on `D-38`. Nothing was shipped.
Also measured: the per-file class counts are predictions, not instances &mdash; class 4 was assigned 6
files and **4** actually move. Remaining deliverable: the two classes needing NO printer change &mdash;
class 5 (`m-format-comment-brackets-break-wall-scan`, lexical) and class 2 (top-level tail, whose owner
IS a multi-line list already).

**[BLOCKED on D-39] [iter-281] m-fmt-typedecl-printer-needs-multiline-emit** &mdash; **the formatter
can attach a comment inside a type-declaration body or a `tests [ ... ]` list, but it cannot PRINT one,
so making the attacher succeed there deletes the comment.** Measured first-party at `2fc0c8b77`:
`printer.algebraicType` (`internal/format/decl.go:437`) is a bare `strings.Join(parts, " | ")` and
`recordTypeString` returns a flat string, so the whole body renders on ONE line with **no emission
points**. Registering those lists in `collectLists` therefore moves `ailang fmt --check` from rc=2 to
rc=0/1 while silently dropping comments &mdash; two arms on byte-identical input, `std/dom.ail`
**OLD rc=2 / zero output** vs **NEW rc=0 / 54 &rarr; 50 comment lines**, `std/ai/streaming.ail`
**135 &rarr; 132**, and the repo's own `TestCorpusCommentGate` catches the `tests`-list half at
`50 -> 17`. **A refusal fails CLOSED; this does not** &mdash; which makes the enumerator-only route
strictly worse than the bug it fixes. The work is therefore a PRINTER change: a multi-line form for
type-decl bodies and inline-test lists, with comment emission hooks, gated on the same
comment-presence condition the braced-block path already uses. **Why it is blocked rather than queued:**
emitting a type declaration across several lines changes what canonical AILANG *looks like*, which is
exactly `D-38`'s open question (*is the formatter's canonical form itself wrong?*) &mdash; and it would
move files that are currently rc=0. Standing rule 2 forbids the loop choosing that for itself. Two
things are cheap and safe to do first and need no ruling: class 5
(`m-format-comment-brackets-break-wall-scan`, purely lexical) and confirming class 2's owner is the
already-registered file top-level list. The `any`-widening groundwork
(`MinAnchor`/`addList`/`subtreeEnd`/`openBraceBefore`, verified behaviour-neutral with the full suite
rc=0) was NOT landed, deliberately: it has no consumer until the printer work is authorised, and
CLAUDE.md's unused-code rule makes dead groundwork a liability rather than a head start.

**[NEXT] [iter-280] m-format-comment-brackets-break-wall-scan** &mdash; **a `{` or `[` inside COMMENT
TEXT makes `ailang fmt` refuse the file, and the invariant that forbids this is written in the code as a
doc comment that was never implemented.** `openBraceBefore` (`internal/format/attach.go:411-419`) scans
left byte-by-byte for the enclosing open delimiter and skips only `a.env.inStringSpan(j)`. There is **no
comment-span predicate anywhere in `internal/format`** &mdash; measured **0** occurrences against a control
of **9** `inStringSpan` uses (negative control on an invented literal: **0**) &mdash; so a brace in a
comment is read as the block's opening wall and the comment is then orphaned. `WidenLeft`
(`envelope.go:309-334`) has the identical shape, and its own comment at `envelope.go:324` reads *"Never
widen into a literal region **or comment**"* while the code beneath it tests `inStringSpan` alone.
**Measured two-armed on byte-identical files differing only in the text of one comment:**
`-- Header: {"alg":"RS256"}` &rarr; **rc=2**, `-- plain comment text with no braces` &rarr; **rc=1**;
same for `-- Returns [] if missing` vs `-- Returns empty if missing`. String literals are correctly
exempt (`let s = "a { b"` &rarr; rc=1), so this is specific to comments. **Scope, measured not
estimated:** neutralising bracket characters in ALL comment text cures **3 of the 38**
(`examples/runnable/jwt_decode.ail`, `examples/runnable/structured_ai_schema.ail`,
`examples/runnable/json_array_extraction.ail`); control &mdash; the same transformation on a
currently-passing file leaves it passing. This is a lexical defect, not a missing boundary: no amount of
list registration fixes it, and it will silently re-appear on any comment anyone writes containing JSON,
a record shape or an empty list. It also fails CLOSED (`--write` refuses), so it is a correctness bug
rather than a corruption risk.

**[NEXT] [iter-280] m-parser-script-leading-minus-glues-statements** &mdash; **in a script-style file a
top-level statement beginning with `-` is silently absorbed into the previous statement.** Found while
adjudicating why `examples/runnable/typeclasses.ail` reports its attach failure at byte 282 rather than at
the file tail. `printf '1 + 1\n-1 + 1\n'` formats to **`1 + 1 - 1 + 1`** &mdash; two statements become one
&mdash; while the control `1 + 1` / `2 + 1` round-trips as two. The formatter is faithful here; the
parse is what changed. Confirmed on the real file two-armed: `typeclasses.ail` as-is reports byte **282**,
and with its single `-42 + 100` changed to `42 + 100` the reported offender moves to byte **2148**, the
genuine file tail &mdash; so that file is an ordinary class-2 case and the gluing is a separate defect.
**Why it is worth a row beyond the formatter:** `typeclasses.ail` is a *teaching* example whose text
documents `-42 + 100` with `-- Output: 58`; under the actual parse that expression is not evaluated as
written. `docs/LIMITATIONS.md:68-70` documents forgiving statement separators **inside `{ }` blocks** and
says nothing about top level. Open question for whoever picks this up: is newline-separated top-level
statement sequencing meant to be terminator-free at all, in which case the fix is a grammar decision, not
a patch.

**[NEXT] [iter-277] m-fmt-gate-corpus-eligibility** &mdash; **7** of the 46 errors are `rc=3` parse
failures in corpora that are *deliberately invalid*: `examples/archive/broken` (2),
`examples/experimental` (4), `examples/bugs` (1) &mdash; files whose whole purpose is to be unparseable
or unsupported. A formatter gate that scans them can never be green, so "widen the enumerator" (which
iteration 277 did, correctly, for `std/`) and "which subtrees are gate-eligible" are **different
questions**, and the row this decomposes conflated them. Deliverable: decide and encode the eligible
subtree set with its own anti-vacuity floor, so the exclusion is a stated, testable predicate rather
than a `find` argument nobody re-reads. **UNGATED iteration 282:** `D-38` resolved **(a)**, not (c) &mdash; so no scope ruling is coming from there and this row must decide the eligible subtree set on its own evidence. It is also now SMALLER than filed: after the `D-38` reformat the corpus is 405/450 canonical, so the only non-green files left are the 38 attach-refusals and the 7 deliberately-invalid parse failures, and the latter are exactly the `examples/archive/broken` + `examples/experimental` + `examples/bugs` subtrees this row exists to exclude.

**[SUPERSEDED by the four rows above, iter-277] m-fmt-check-ail-broken-and-red (original filing)** &mdash; `fmt-check-ail` is both **unwired
and broken**, and its breakage is the enumerator-scope class this loop has now met three times.
(1) Its `find examples stdlib -name '*.ail'` scans a `stdlib/` directory that **has never existed
in this repo** &mdash; measured 2026-08-25: `test -d stdlib` NO, `test -d std` YES, as-written
**404** files vs corrected **450**, so **46** stdlib `.ail` files are invisible to the gate.
Iteration 187 measured the identical defect (400 vs 446) and filed it; it is still here, and the
`2>/dev/null` swallows `find`'s complaint. (2) It is ALSO rc=2 at HEAD on real drift in
`examples/pattern_matching_adt.ail` and `examples/record_in_result.ail`. (3) And `ailang fmt`
itself **errors** on `examples/snippets/v3_3/math/gcd.ail`: *"comment at byte 353 could not be
attached to any boundary"* &mdash; a formatter bug, not drift, and the third of these is the one
that must be adjudicated first because a gate whose tool crashes cannot be wired at any scope.
Order: fix the formatter bug, widen the enumerator to `std`, re-measure the drift set (it will
GROW by up to 46 files' worth), then wire.

**[NEXT] [iter-278] m-firestore-index-provenance** &mdash; prod `ailang-multivac` carries **10**
composite indexes; **nothing in the `ailang` repo records which ones the code requires** (0 `.tf`
files tracked, controls firing). They are added *reactively*, one `FailedPrecondition` at a time
&mdash; `firestore.tf`'s own `messages_inbox_only` comment documents a prior instance in exactly
those terms, and iteration 278 is instance **3**. The failure mode is uniform and expensive: a query
shape ships, nobody exercises it, and the first caller gets a runtime refusal on a command the docs
prescribe. Note the two halves are *independently* broken &mdash; a declaration in the deploy repo
does nothing until an apply runs (measured: declared 19:15, prod still bare at 20:33), so a check
that reads the Terraform is not a check that the index is live. Candidate deliverable: enumerate the
composite-index-requiring query shapes from the code (equality `Where` + `OrderBy` on another field),
diff that set against live prod, and fail loudly on a gap &mdash; an enumeration complete by
construction rather than by inspection (rule 3a(i-e)). **Gated on nothing; cross-repo** &mdash; the
declarations live in `ailang-multivac`, so scope the ask before routing.

**[NEXT] [iter-275] m-cli-examples-fixture-rot** &mdash; `verify-cli-examples` is rc=2 with
**9 of 26** documented CLI commands failing against `examples/cli_examples.txt`. Two are
substantive rather than cosmetic and they point in different directions: `list_sum.ail` is
documented to print `(15, 15)` and actually prints **`(15, 5)`** &mdash; either the doc or the
example regressed, and which one is the whole question; and `examples/docs/lambdas_full.ail`
fails to type-check because it uses `++` on strings, which has been **list-only since v0.13.0**,
i.e. a documentation example teaching removed syntax. The remaining seven are `--entry main`
invocations. This is a docs-correctness gate that has been dark long enough for the docs to rot
under it; the fix is per-command adjudication, not a bulk re-record of actual output (that would
convert a gate into a snapshot of whatever the code does today).

**[NEXT] [iter-275] m-verify-examples-trace-suppressed** &mdash; the only make invocation in any
workflow whose failure is suppressed. `ci.yml:322` runs `make verify-examples-trace ... || true`,
so the target is CONNECTED and cannot REFUSE; it is rc=2 on 2 of 217 examples and takes 42s.
Iteration 275 added `TestWiredGatesCanFailTheJob` and this is its single declared exemption,
carrying an instruction to REMOVE the entry rather than leave it once the target is green.
Measured repo-wide in the same sweep: exactly **one** suppressed make invocation and **zero**
`continue-on-error` on any make-gate step, so the exposure is this row and nothing else &mdash;
which is why the exemption is a row and not a policy. Also unresolved: `verify-examples-all`
(rc=2, a 60% threshold gate over ALL examples) overlaps this surface and should be adjudicated
with it rather than separately.

**[NEXT] [JUDGE-FOUND iter-274] m-ci-composite-action-blind-spot** &mdash; declared in
`gate_wiring_test.go` as a KNOWN LIMITATION rather than fixed. `loadWorkflows` reads only
`.github/workflows/*.yml|*.yaml` and the scan reads only `step.Run`, so a `make` invocation inside a
composite action (`uses: ./.github/actions/x`) or a first-party reusable workflow is invisible.
Dormant at HEAD &mdash; evaluator measured **zero** of either &mdash; and the failure direction is a
false RED (a wired target reads as unwired), which is loud rather than silent. If either is ever
added, widen the enumerator; do **not** answer the red with an exemption-map entry.

**[SUPERSEDED by iter-274 &mdash; original row text kept for provenance]** &mdash; nothing in this repo validates the
CONTENT of `.github/workflows/ci.yml`. Measured by the iteration-273 evaluator: deleting the two
newly-added gate steps from `ci.yml` leaves **every** local gate rc=0 &mdash; `check-changelog`,
`check-file-sizes`, `check-boundaries`, `check-protocol-closure`, `test-check-protocol-closure`,
`check-tmpfile-hygiene`, `test-check-tmpfile-hygiene`, `build`. So a CI step can be silently
un-wired and no local signal fires; the only detection is a human noticing a job is missing. This is
the enumerator question one level up from `m-protocol-closure-*`: **the repo pins what its gates
CHECK and not that its gates are CONNECTED.** Note the fleet has ~20 required/expected checks, and
this loop's own Gate 1 enumerates workflows by hand-maintained NAME for the same reason. Options:
(a) a gate asserting every `make check-*` target appears in `ci.yml`; (b) the inverse &mdash; every
`ci.yml` `run: make X` names a target that exists; (c) declare it and rely on review. (a)+(b) is one
small script and closes both directions. Same row also covers the changelog hunk having no killer,
which is a weaker instance of the same class.

**[NEXT] [iter-273] m-tmpfile-hygiene-residual** &mdash; residuals from `check_tmpfile_hygiene.sh`,
all **declared in the script header** rather than left implicit, filed here rather than absorbed
into the sprint (rule 3n(b): a gap found is a queue row, not a silent widening). (1) `R2b`, the
awk-failure detection branch, has no killer &mdash; the evaluator neutered it and the self-test still
reported a clean pass. It is hard to trigger honestly and was deliberately **not** faked. (2) Three
enumerator blind spots, each measured: a `.mk` file whose name differs in case (the glob is
case-sensitive), a `.mk` file nested in a subdirectory of `make/` (the glob is not recursive), and an
alternate root makefile name such as `GNUmakefile` (which trips `R1` rather than being scanned).
(3) `MK_FILES_EXPECTED=12` is a floor, so growth is safe but a legitimate `.mk` REMOVAL requires a
manual bump &mdash; loud, but a maintenance edge worth knowing about.

**[NEXT] [iter-270] m-gemini-verdict-score-threshold** &mdash; `ValidateVerdict` in
`internal/eval_harness/gemini_evaluator_bridge.go` enforces only half of the documented verdict
invariant: it rejects `Blockers` non-empty with `Pass==true`, but never checks `Pass` against
`Score >= 70`, so a model returning `{score: 65, pass: true}` validates clean. Surfaced while
resolving `geminiPassThreshold`, whose deletion this row deliberately does NOT reverse &mdash;
enforcing the score half is a **runtime behaviour change** to a live evaluator lane and was
correctly kept out of iteration 270's lint-gate scope. Options: (a) enforce and reject; (b) enforce
and warn; (c) declare in the type's comment that the threshold is model-applied by design.

**[NEXT] [JUDGE-FOUND iter-270] m-codex-streaming-test-flake** &mdash;
`TestExecuteStreaming_TolerantToNonJSONPreamble` (`internal/executor/codex`) fails with
`signal: killed` under full-parallel `make test` on the rig, and passes every time in isolation
(`-count=3`) and under `./internal/executor/...`. Pre-existing and unrelated to any iteration-270
file; found by the `sonnet` evaluator during its independent full-suite re-run. This is rule 3m's
shape &mdash; a bound that holds on one machine's load profile and not another's; the fix is to
derive the bound or reduce the arm's resource appetite, not to raise a constant.

**RATIFIED (attended, 2026-08-03 evening) — Mark's two one-word rulings on the iter-135 digest asks:**
1. **Standing fast-forward authorization: YES** — when local dev is **0 commits ahead** AND the
   working tree is clean apart from the known rig-synced dirty files, the controller MAY
   `git merge --ff-only origin/dev` without a per-instance ask (safe by construction: nothing
   local exists to lose). ANY other state still triggers Critical Principle 0. Stop re-asking.
2. **recorded-stream S2 does NOT jump the queue: NO** — order stands `#498 Lane B` → S2
   (S1 already covers the author's primary need; #498 remains World's sole clause-6 blocker
   per the 2026-08-03 directive).

**RATIFIED (attended, 2026-08-14 morning) — Mark's three rulings on the parked decisions**
(recorded here rather than as an `#635` comment: the session was steered by Mark in person but
authenticated as the bot account, which `mission_directives.sh`'s self-direction guard rightly
refuses as a directive principal — the charter stamp is the durable channel):
1. **`D-17`: RATIFIED AS SHIPPED** — `exit(N)` from an embedded call surfaces as
   `*embed.ExitError{Code: N}`, and **`exit(0)` is an `ExitError` too, NOT a nil error**.
   Consequence for `#706`: the fix direction is teach hosts to branch on `Code == 0`
   (serve-api / a2a / mcp map it to a success response), NOT nil-at-source.
2. **`D-16`: YES** — the controller MAY `git merge --ff-only origin/dev` in the MAIN checkout when
   (a) local dev is **0 commits ahead** AND (b) the dirty files **provably do not collide** with
   the incoming commits (path-disjoint, measured not assumed). This EXTENDS the 2026-08-03
   standing authorization above (which required clean-apart-from-rig-files) to the
   dirty-but-disjoint case that kept the running skill 7–9 commits stale across iterations
   195–197. Any collision, ahead-state, or doubt still triggers Critical Principle 0.
3. **`D-15`: `view`** — opt-in remote read reaches the `Backend`-shaped `chains` view commands
   only; `eval-*` errors loudly on `--remote` (turning assumed demand into a dated signal). The
   ratified freeze item is knowingly NARROWED by this ruling. Resume `#698` part 1 M4 per its
   queue row.


**RATIFIED (attended, 2026-08-23) — Mark's ruling on cross-mission interdependencies.**
Recorded as a charter stamp for the same reason the 2026-08-14 block was: the session was steered
by Mark in person but authenticated as the bot account, which `mission_directives.sh`'s
self-direction guard rightly refuses as a directive principal.

> **⚠ LABEL CORRECTION (iteration 260, bookkeeping only — no ruling is changed).** This block was
> written using the labels `D-31` and `D-32`, both of which were **already in use** by OPEN rows in
> the marked `decision-ledger` block (the designer-rotation split, filed iteration 256; and the
> `inconclusive` KPI-exemption question, filed iteration 259 in the commit immediately preceding
> this session). Measured: `2fde160db` landed `15:53:16+02:00`, this block's commit `4e7c32ce0` at
> `18:08:23+02:00`, and `4e7c32ce0` touched **zero** ledger rows. The decision-recording contract
> makes the ledger authoritative and forbids ID reuse, so **Mark's two rulings below are now
> ledger `D-33` (cross-mission prioritization) and `D-34` (the v0.34.0 release decision)**, and the
> OPEN `D-31`/`D-32` keep their IDs and their questions. The text below is left as ruled; read
> `D-31` → `D-33` and `D-32` → `D-34`. Left uncorrected, this would have made the loop re-ask
> `D-31`/`D-32` hours after Mark answered two different questions wearing those labels.

1. **`D-31` [= ledger `D-33`]: cross-mission blockers are prioritized** — see ordering policy rule 2 above. The
   `cross-mission` label was created and applied 2026-08-23 to the 8 open sibling asks
   (`#764`, `#757`, `#756`, `#755`, `#715`, `#713`, `#712`, `#498`).
2. **`#764` TO THE QUEUE HEAD as P1** — the row immediately below.
3. **`D-32` [= ledger `D-34`] — standing release decision: when `#764` lands, cut v0.34.0.** World consumes upstream
   via **pinned releases only**, so merging the fix does not reach World; the tag is the delivery.
   Surface it in the report's DECISIONS row once the fix is green on dev. `dev` was **224 commits**
   ahead of `v0.33.1` (2026-08-13) at ruling time. Releases remain Mark's sole decision — this
   ruling pre-authorizes the ASK, not the tag.

**0a. [LANDED — SPRINT COMPLETE. M1 `d54672b85` (iter 266) · M2 `4a813b2c0` (iter 267) · M3 `ba2eeb4b4` (iter 268) · **M4 `7e7bdffcb` (iter 269; PR #864 non-closing; `sonnet` evaluator PASS 89/100, generator≠judge, own worktree)**. M4 corrected a plan defect: `make lint` computes its verdict from a SECOND path list (`grep -qE "^(internal|cmd|testutil)"` at `code-health.mk:89`), so adding `./serveapi/...` to the scan list alone would have left the gate unable to REFUSE anything in `serveapi/`. Both lines changed; three-arm drill — both hunks rc=2, either hunk alone rc=0. **ISSUE #764 DELIBERATELY LEFT OPEN:** World pins upstream by RELEASE, so the merge does not deliver it; the delivery is the **v0.34.0** tag. `D-34` pre-authorises the ASK — surfaced to Mark this iteration. Do NOT close `#764` until that tag exists] `#764` — a protocol-only serveapi module**
Doc: [planned/m-serveapi-protocol-only-module.md](planned/m-serveapi-protocol-only-module.md)
(authored + one revision, iteration 260; **two full quorum rounds, 2/2 reviewers present both
times, `absent_reviewers` EMPTY both times**, four objections, all four answered). Was parked on
**exactly one axis** — `D-35`, the module boundary — which the round-2 reviewer promoted from a
deferred footnote to a blocking design-freeze gate by striking the unverified claim that held it
open. Never parked on design direction: no reviewer has disputed that a protocol-only package should
exist, across four reviews. Never `PARKED-ON-LANE`: nothing here unblocked on a clock.
**RESUME PREDICATE DISCHARGED 2026-08-23:** Mark answered `D-35` with **(a) PLAIN PACKAGE** at
`19:01:24Z` on `#745`. Iteration 261 froze the doc on that ruling in the same iteration it was read
(D-A checked, ledger RESOLVED) and routed it straight to `sprint-planner` on the
`opus fail-closed:planner-lane-field-missing` lane. Under (a) the packaging milestone is the
ordinary main-module tag, which is exactly the delivery `D-34` already pre-authorized.

([`sunholo-data/ailang#764`](https://github.com/sunholo-data/ailang/issues/764)) · clause-6 ·
**This is Ailang World's SOLE remaining blocker to its M4 — the value gate, and the first moment
any agent can use World at all.** Chain, measured 2026-08-23:
`#764` → world item 5 `w-mcp-projection` [BLOCKED] → world item 6 `w-agent-floor-m4` [PARKED]
→ World DESIGN.md M4. World items 2, 3 and 4 are LANDED (M1 kernel, M2 daemon `ailang-worldd`,
M3 effect broker), so item 5 is the only one of 2–5 still open.
**THIS IS NOT `#498` RE-OPENED, AND THAT DISTINCTION IS THE WHOLE ROW.** `#498` Lane A and Lane B
both landed and both SHIPPED — `f5ebcc0b5` in **v0.33.0**, `6166adab8` and `b8c038647` in
**v0.33.1** — and World's charter records that prerequisite DISCHARGED on exactly that basis.
`#764` was filed 2026-08-17, four days AFTER v0.33.1, when World imported the released seam and
found it is an API seam but not a DEPENDENCY seam: importing it links **479 non-stdlib packages,
304 of them cloud/telemetry**. So this is what shipping `#498` revealed, not a regression of it.
**WHY IT SAT.** Filed with **0 labels and 0 comments**, it was invisible to every sweep for six
days while this loop ran eleven iterations (245–256). The defect was the routing mechanism, not
the priority call — `#498`'s equivalent row was raised P2→P1 by directive and *was* worked. Fixed
structurally by `D-31` above rather than by this row alone.
Ask: a protocol-only module that does not drag the cloud/telemetry dependency tree.

**Required-for-v1 (the bar's critical path):**

1. [LANDED 2026-07-10] m-named-test-blocks closeout (iteration 1a; deontic criterion deferred,
   package absent locally)
2. [LANDED 2026-07-10] m-typeenv-sub-fix (iteration 1b: RESOLVED — pre-closed by adjacent
   M-TYPE-LIST-SOUND work, regression-guarded, eval 92/100, merge f59421ac8)
3. [LANDED 2026-07-10] m-feedback-triage-gate (iteration 2: full loop headless, eval 93/100
   PASS round 1, merge 40f1cdc3f, remote CI green on dev post-merge; gate logic complete +
   merged, off by default — production activation gated on the next item)
4. [LANDED 2026-07-10] m-feedback-gate-cloud-adapter (iteration 3: full loop headless, round-1
   eval FAIL → round-2 PASS 97/100, merge 842d7d501, dev CI fully green on 4c22032de; gate
   code complete incl. cloud adapters, OFF by default — production enablement is a HUMAN ops
   task: sibling-repo terraform TTL + ANTHROPIC_API_KEY secret, then DRY_RUN=1 week 1)
5. [LANDED 2026-07-10] m-diagnostic-coverage (iteration 4: M1–M3 found pre-shipped 2026-07-09
   under a stale "Planned" status; remainder sprint M-DIAG-FIXTURE-PROMOTION promoted 4 rows
   to covered — 7 CI fixtures across 6 footgun rows — eval PASS 96/100 round 1, PR #336 →
   fe807aac8, dev CI green per-workflow. DEFERRED, rationale in doc: deletion pass + rig A/B
   until deletable surface ≥ 100 lines; PARKED for human: haiku causal re-run, API-billed)
6. [LANDED 2026-07-10] m-v1-stability-promise (iteration 5: FULL loop headless round-1 clean —
   Fable design doc → Opus plan (caught 2 premise errors: 42 modules not 39; LIMITATIONS
   double-maintained + diverged, both copies fixed) → Opus execute in worktree → Fable eval
   PASS 96/100 round 1. Stability page docs/docs/reference/stability.md (3 tiers, full stdlib +
   CLI tables), both LIMITATIONS files live-verified at HEAD, 4 website vX-promises retracted,
   PR #337 → fcccd7208, dev CI green per-workflow. PARKED for human at RELEASE gate: tier-
   assignment ratification — ⚠ proposed: std/net, crypto, jwt, xml, zip, process, CLI
   watch/serve-api)
7. [LANDED 2026-07-11] m-effect-refinement **decomposition** (iteration 6: repo-verified phase
   census — P1/P2 + AI port shipped v0.15.0 under the parent's stale "Planned"; P7 CryptoRand
   never existed (m-cryptorand.md swept to implemented/ in error — header corrected); P6 routed
   OUT to M-ENTROPY. Remaining ~64h split into 4 sprint docs (below, items 9/12/13/14) with live-
   verified premises; parent doc now the umbrella. BONUS finding: the public guide's "typechecker
   rejects unknown values" is FALSE (`Rand[mode=banana]` passes check) — interim accuracy note
   shipped, enforcement is sprint 1)
8. [LANDED 2026-07-11] m-eval-frontier-tier (iteration 7: full loop headless, round-1 clean —
   Opus plan (9 discrepancies) → Opus execute (frontier tier + 8 re-tiered + prefix_line
   structural grader + 7 core→stretch demotions via 4-dim rule from banked data) → Fable eval
   PASS 96/100 round 1 w/ independent distinct-sample recount. PR #339 → 0515578ae, dev CI
   green per-workflow. PARKED for human: frontier-failure validation of the 8 (API-billed —
   each must fail ≥1 frontier model or demote back per CURATION.md §5) + 4 remaining sketches)
*(Queue re-derived 2026-07-11 from bar v2 — clause tag on every open item. NEW-DOC items start
with design-doc-creator; existing-doc items start at reality-check.)*

9. [LANDED 2026-07-11] m-effect-mode-validation (iteration 8: full loop headless, round-1 clean —
   Opus plan (2 discrepancies: bridge carries no params, scope-reduced; EFF_* codes frozen) →
   Opus execute (effectSchema + validateEffectParams at elaboration, 3 fix-carrying diagnostics
   CI-fixtured, guide truth-up: the public closed-set claim is now TRUE, prompt names the codes)
   → Fable eval PASS 96/100 round 1 w/ independent transcript re-production. PR #340 → 8faa49de9,
   dev CI green per-workflow. Unlocks effect sprints 2-4. BONUS: dev-health issue #341 filed
   (5 pre-existing example type-check failures; verify-examples not a CI gate))
10. [LANDED 2026-07-11] m-syntax-ai-forgiving (iteration 9 — the first iteration SPLIT ACROSS
    TWO scheduled runs: run A did reality-check 192a79149 + Opus plan a7bd8257c + Opus execute
    (worktree, M1–M4 64ddd6021) then died pre-evaluation; run B resumed at sprint-evaluator.
    Fable eval PASS 96/100 round 1 (FIFTH consecutive) w/ independent fuzz-gate re-run (zero
    AST diffs over 389 currently-valid corpus files), rebuilt-binary transcripts, non-vacuity
    vs v0.29.2 (PAR017/PAR020 fire on exactly the now-accepted fixtures). R1+R2 BOTH landed —
    R2 systemically patched FOUR block loops (plan's D6 knew two; if/then + \-lambda route via
    parseRecordLiteral). PR #342 → merge, dev CI green per-workflow. DEFERRED: ailang fmt →
    m-ailang-fmt.md stub (D1). PARKED for controller/human: the rig A/B compile_error Δ on
    ;-family benchmarks — the REAL success metric, GPU step, rotation held the rig)
11. [LANDED 2026-07-11] m-stdlib-regex (iteration 11: full loop headless, round-1 clean — Opus
    plan (3 de-risking findings: F1 `_str_slice`/`_str_len` are RUNE-indexed but Go `regexp`
    returns BYTE offsets → span conversion is load-bearing; F2 embed is a glob; F4 changelog
    path) → Opus execute (worktree: 6 `_regex_*` builtins in the MODERN `internal/builtins/`
    RegisterEffectBuiltin system — NOT the doc's outdated `internal/eval` path, **D-ARCH**;
    memoized RE2 cache; `std/regex.ail` + 3 examples incl. the log-orchestration clause-4 use
    case) → Opus eval PASS 97/100 round 1 w/ INDEPENDENT reproduction (backref reject, CJK
    `日本語 world` rune span [4,9) not byte [10,15), findAll). PR #343 → squash-merge 0b0ed7ea0,
    all required checks green. `std/regex` = linear-time (RE2): compile/isMatch/findFirst/findAll/
    replaceAll/split; RE2 subset (no backref/lookaround) → `compile` Err, never panics. Docs:
    LIMITATIONS + stability (Experimental) + CHANGELOG. Design → implemented/v0_30_0)
12. [LANDED 2026-07-12] m-stdlib-url-parse (iteration 13: full build loop headless — Opus executor
    (worktree: `_net_url_parse` + `_net_url_parse_query` pure builtins in the modern
    `internal/builtins/net.go`, wrapping Go `net/url`; `Url` record + wrappers in `std/net.ail`;
    26 non-vacuous tests incl. IPv6 `[::1]:80`, error-never-panics, order+dupe preservation,
    round-trip; 2 examples; docs) → independent Opus evaluator round-1 FAIL 80/100 (single BLOCKER:
    stale `builtin_types.golden` not regenerated → repo-wide `make test` red) → round-2 golden
    regen → PASS 100/100. Design → implemented/v0_30_0. PR #347 → squash-merge `a8628a40c`,
    auto-merge on green required checks. `std/net` now parses URLs: `parseUrl(s) -> Result[Url,string]`
    (Err on malformed, never panics/fallbacks — CP2; `port:string` ""=absent) + order-preserving
    percent-decoded `parseQuery` (inverse of `urlEncodeForm`). Pure `! {}`, no Net cap. Closes v1.0
    bar clause 4's URL-parse half (regex half = #11). BONUS finding: `cmd/ailang`
    `TestRunCommand_PipedStdoutFlushesPerLine` is a pre-existing flaky under parallel `make test`
    load — passes 3/3 in isolation, unrelated to this sprint; flagged for dev-health, not a gate)
13. [LANDED 2026-07-12] m-module-less-run-fail-loud (iteration 14: full build loop headless, round-1
    clean — reality-check caught the doc's **MOD011 collision** (already the module-path-collision
    code) → reassigned **MOD014**; Opus plan → Opus execute (worktree: `validateModulePath` early-
    accept replaced with a loud MOD014 error gated on `len(Funcs) > 0`, fires for both `run` AND
    `check`; the doc's 3-way `Funcs||Statements||Decls` guard was code-refuted mid-sprint — a bare-
    expr FILE does reach `validateModulePath`, so the OR would break `ailang run 1+1`; block_demo
    remediated; footgun fixture 17→18) → independent Opus evaluator PASS 100/100 round 1 w/ a
    base-origin/dev binary proving test non-vacuity + pre-existing-failure claims. PR #349 →
    merge `c2ffd1b5c`, post-merge dev CI green per-workflow. Design → implemented/v0_30_0. Module-less
    files now FAIL LOUDLY (CP2). Skill-fix: design-doc-creator error-code + mechanism verification gates)
14. [LANDED 2026-07-12] m-match-xcheck-error-quality (iteration 15: full build loop headless, round-1
    clean — Gate-1 origin-sync caught local dev 4 commits behind origin/dev (iter 14 landed via #350),
    read state from origin; reproduced the empty `Option's constructors are: ` line live at HEAD →
    **Option A** (design doc's own recommendation): a diagnostic-only `Constructor→ADT` registry
    (`moduleImports.AllCtorTypes`) built from ALL transitively-loaded ifaces via
    `modLinker.GetLoadedModules()`, plumbed via new `SetDiagnosticConstructorTypes`, consulted by
    `lookupADTConstructors` ONLY when the primary direct/local scan is empty — never enters scope
    (`types` can't import `link` → passed as a plain `map[string]string`). Opus plan → Opus execute
    (worktree, commits `3ded459cc`/`f5498ca0e`/`ecca08b3b`) → independent Opus evaluator **PASS 96/100
    round 1** w/ base-binary non-vacuity proof + scope-non-leak + format-unchanged checks; 2
    non-blocking deductions folded into the hardening commit
    (`TestSchemeImport_DiagnosticRegistryDoesNotLeakIntoScope` + collision note). PR #352 →
    squash-merge `5aaaff2ed`, required checks green (auto-merge). Design → implemented/v0_30_0.
    Foreign-ctor errors now enumerate transitively-known constructors (`None, Some` + did-you-mean).
    SonarCloud PR gate red = advisory/non-required (merge succeeded) — flagged for sonarcloud-triage)
15. [LANDED 2026-07-13] m-module-let-func-resolution (iteration 23: full build loop headless, round-1
    clean — first CI-red fix-forward (gofmt miss from `366c5bbb2` broke dev fmt-check 2 runs →
    `39171a4f9`, observed green); Opus plan (caught the design doc's WRONG test path: the #327
    40-cell matrix is `internal/pipeline/record_update_positions_test.go`, NOT `internal/types/`;
    proposed MOD007 from the reserved block) → Opus execute (worktree: M0 spike **GO** — evaluator
    binds any `core.Let`, `CheckCoreProgram` threads forward env → unified SCC over lets+funcs,
    `wrapInLets` + BOTH re-elaboration loops DELETED; module `letrec` SUPPORTED via `core.LetRec`;
    dup module-scope name → **MOD007** hard error, zero corpus collisions; hint truth pass — 0
    `known bug #327` hits, residual hint cites #366 + real workaround "declare it as a func") →
    independent **Fable** evaluator (model diversity restored — controller reverted from Opus)
    **PASS 98/100 round 1** w/ own worktrees+binaries, base-binary non-vacuity (v3/v7/v8 fail at
    `116ebcb49` → run 16/0/4 post-fix; v10 silent shadow → MOD007), adversarial probes (func→let→func
    topo chain, let↔func cycle → LetRec no crash, effectful module let rejected identically).
    PR #368 → squash-merge `fd38ec14e`, post-merge dev CI green per-workflow. Design →
    implemented/v0_30_0. Module lets now resolve module funcs uniformly (4th family member CLOSED).
    ⚠ PICK-ORDER MISS recorded: Mark's [NEXT-FIRST] below (added 13:04, pre-session) should have
    outranked this pick; Gate-2 read the queue head + prior log's Next but not the fresh directive.
    Sprint was already through eval when caught → landed; iteration 24 is HARD-PINNED to it)
**[LANDED 2026-07-13, iteration 24 — was Mark's NEXT-FIRST, ⚠ missed by iteration 23, taken
first by iteration 24 as pinned]** m-public-feedback-delivery-audit
([implemented/v0_30_0](implemented/v0_30_0/m-public-feedback-delivery-audit.md), P1): full inner
loop headless, round-1 clean — Opus plan (killed 2 feared ops steps: prod sub exists, ADC owner
on both projects; corrected "structural, not novel" → real multi-project fan-in) → Opus execute
in worktree (Defect A: `isExternalFeedbackInbox` tags `pkg:*` as `public-feedback`, allow-list
untouched; Defect B: `Daemon` N-message-sources refactor + `firestore.NewClientForProject` +
opt-in `extra_message_envs`/`--also-subscribe`, default OFF byte-identical) → Fable evaluator
**PASS 97/100 round 1** (base-binary non-vacuity both defects; 0 test deletions; conflict surface
intact). PR #378 → `4fee247a8`, post-merge dev CI green per-workflow (observed). ⚠ PARKED for
Mark: daemon reload + 2 live prod test-sends (checklist: sprint plan §Parked-for-human +
docs/docs/guides/notify-daemon.md); until reloaded, prod feedback still doesn't ping — the CODE
is landed, the OPS switch is human.

16. [LANDED 2026-07-13] iteration 25 — **R4a+R4b GHOST-CLOSE + m-lambda-open-record-pattern
    EXECUTED**: Gate-2 reality check live-probed the queue's R4 rows (the sourcing strategy
    review admitted they were never individually re-verified) → R4a `m-dx-match-hof` GHOST
    (retired `match … with` syntax was the culprit; design doc archived Not-Applicable
    2026-05-09; `\x ->` already has a teaching diagnostic) + R4b `m-poly-arith-lambda` GHOST
    (fixed v0.7.0) — guards `examples/match_hof_lambda.ail` + `poly_arith_lambda.ail`, PR #379
    → `ea8116f83`, CI green observed. Then the full inner loop on m-lambda-open-record-pattern
    (REAL at HEAD; mislabeled NEW-DOC — full design doc existed at planned/v0_29_0): Opus plan
    (refuted the doc's H3-primary via an IIFE probe) → Opus execute (found the TRUE primary
    site absent from doc+plan: `unifyRecord` rejected on field-count BEFORE consulting row
    variables; `core.RecordPattern.Rest` + `unifyOpenRecords` row-polymorphic subsumption;
    closed-pattern strictness preserved) → independent Fable evaluator **PASS 92/100 round 1**
    (own base+sprint worktrees/binaries, non-vacuity both directions, 8 adversarial probes, 0
    test deletions; found an arm-order-dependent acceptance) → hardening commit `89b75bd3f`
    (order-independence fix proven load-bearing, dead-code removal, cacheKeyVersion v2 for the
    gob-struct change). PR #380 → `47576e25d`, dev CI green per-workflow observed. Design +
    sprint plan → implemented/v0_30_0.

**[LANDED 2026-07-16 (M1a+M1b+M2) / M3 PARKED-protocol]** **m-mission-agentic-provider-routing**
([planned/v0_30_0](planned/v0_30_0/m-mission-agentic-provider-routing.md)) — mission-infra P0.
Fixed the routing-never-enforced bug (memory `project-mission-routing-table-never-enforced`).
**M1a LANDED 2026-07-15** (interactive, 8ee07ef23 + amended d545d4a9e): per-role env pins, opus-first
controller, fable designer/evaluator by inheritance. **M1b+M2 LANDED 2026-07-16 iteration 31**
(direct-on-dev main checkout, zero Go — the planner found registry/DryRun/codex executor all
pre-exist since v0.22.0): Gate-3 `provider:model`→bounded `codex exec` recipe (probe live-verified:
gpt-5.6-sol exit 0; default-env fire = no-op, codex strictly opt-in) `956fda55c` + charter
right-sizing table & provider/agent/cost evidence-row schema `8d12e8e9c`; eval PASS 87/100 round 1;
hardening `1c964aae2` — **F1: the Agent tool pins only sonnet|opus|haiku, `fable` is REJECTED**
(fable roles run by session inheritance only; alias-lane generator≠judge guard added: evaluator
never falls back to bare $MODEL, re-routes to sonnet + FLAG) + F2 `exec` orphan-kill fix.
**Open by design**: first REAL cross-provider fire (opt-in `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol`,
= the doc's M1b acceptance) · **M3** (planner down-tier A/B) PARKED with a concrete protocol in the
sprint plan until 3 quorum-reviewed docs accrue. Doc stays in planned/ until those close.

**[NEXT-FIRST, Mark 2026-07-16 — FLEET ROLLOUT ("should be awesome")]** The ratified starting
fleet is **claude (Anthropic) + codex gpt-5.6-sol (OpenAI) + managed_agents gemini (Google) +
motoko/qwen3-6 (local GPU)**. Sequenced, one per iteration:
- **(a) ~~Iteration 32 — codex LIVE-FIRE~~ DONE 2026-07-16**: FIRST real cross-provider fire landed.
  `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol` (one-shot override consumed) executed `20251013_auto_caps`
  M1 (`--caps auto`) end-to-end: Opus planner → **codex/gpt-5.6-sol executor** (OpenAI, ~4.5-min run,
  metered) → **Sonnet evaluator** (generator≠judge: openai≠anthropic; fable pin unenforceable →
  re-routed to sonnet + FLAGGED per the F1 guard) PASS 98/100 r1. PR #397 → `e542065c0`. **Recipe
  frictions found & fixed (Gate-5 skill edit): the codex real-run recipe had only ever been verified
  against the text probe** — a real coding run needs `--sandbox workspace-write` + `--add-dir` for
  GOCACHE/GOMODCACHE, cannot self-commit (worktree `.git` lives under the main checkout →
  controller finalizes the commit from the uncommitted worktree diff), and must run backgrounded
  (the 30-min cap exceeds the harness's 10-min foreground bash limit).
- **(b) ~~M1c — gemini managed_agents recipe branch~~ DONE 2026-07-16 (iteration 33, PR #398 →
  `bd89418a6`)**: the "no new plumbing" claim was REFUTED — `ailang exec gemini` (agentic) was
  unreachable (`unknown executor: gemini`; managed_agents registers under its own name, no gemini
  alias). Landed a real ~30-LOC `exec.go` fix (`resolveAgenticExecutorName`: gemini→managed_agents,
  `--api-only` untouched) + test + the Gate-3 `PROVIDER=gemini` recipe branch. **CapRemoteSandbox
  scoping**: the lane serves READ-ONLY roles (evaluator/reviewer/quorum-verifier) only — the
  server-side sandbox never writes the local worktree, so the file-editing executor role needs a
  bridge (follow-up). Sonnet eval PASS 96/100 r1. First LIVE gemini fire deferred to (c).
- **(c) [CORE LANDED 2026-07-16 iter 36 (M1-M3) — PR #400 → `0e83a1b12`; M0/M4/M5 now UNBLOCKED — (c0) plumbing landed iter 37, this is the `[← NEXT fleet step]`]** m-mission-quorum-agentic-verify+HONE
  — **M1-M3 shipped**: `agenticCaller` behind the `JSONCaller` seam (frozen verdict JSON via the coordinator
  executor layer), `ShouldEscalate` two-tier trigger + additive-optional `proposed_fix` (option (a), contract
  frozen), Tier-2 codex+claude read-only verify. 43 tests pass, verdict contract independently verified
  unchanged, evaluator PASS 91/100 r1. **M0 (gemini network probe) BLOCKED**: `ailang exec gemini` fails
  `GCP project not set` — `cmd/ailang/exec.go` never plumbs `Task.GCPProject` outside the eval harness
  (fix = item (c0)). Once (c0) lands: M0 (live gemini probe) → M4 (conditional on M0 result) → M5 (live-fire
  + doc → implemented/). Watch items carried: `agentic_caller.go:85` ctx.Background→caller-ctx before a live
  Tier-2 fire; `premiseSignals` breadth; M4 fallback must carry an explicit `VerificationDegraded` marker.
  — iteration 34's Gate-2 quorum-at-pick park is RESOLVED: Mark chose **(a) `proposed_fix` optional,
  not validated, contract frozen** (doc's HONE section stamped; the code-cited Verification-Log rows
  for the refuted sol objection added — provider_executor.go exposes ctx-cancel/Timeout/CostUSD/
  read-only-AllowedTools/WorkingDir, reuse premise HOLDS). Doc is quorum-cleared for routing: **start
  at sprint-planner** (both quorum rounds + revisions already done; do NOT re-quorum — the two rounds
  + resolved authorial decision ARE the quorum outcome). M0 = the managed-sandbox network probe (doc
  §Agentic reviewer backend). Meta-finding stands (text quorum blocked premises TRUE-in-code — the
  motivating case for this doc). Meta-finding (Gate-5): the TEXT quorum-at-pick blocked a doc whose premises are TRUE-in-code
  precisely because text reviewers can't read code — the motivating case for this very doc. Original ask:
  reviewers become tool-using agents (codex/managed_agents/
  claude-CLI, read-only worktrees) that VERIFY premises against the repo AND attach a concrete
  `proposed_fix` per objection; the AUTHOR (designer role, now true-Fable via the Gate-3
  `claude:claude-fable-5` CLI lane — driver default updated) accepts/rejects each by name.
  Single-author + adversarial-proposers, NOT co-authoring. Preconditions all satisfied (doc
  updated). Two-tier stays: text quorum always, agentic escalation when contested/high-stakes.
- **(c0) [LANDED 2026-07-16 iter 37 → implemented/v0_30_0; PR #401 → `60351087b`, eval PASS 96/100 r1]**
  m-gemini-exec-project-plumbing — `resolveGCPProjectEnv()` (`AILANG_CLOUD_PROJECT` → `GOOGLE_CLOUD_PROJECT`,
  coordinator precedence) + `GCPProject`/`GCPLocation` now set on the shared `executor.Task` in
  `cmd/ailang/exec.go:executeCLI`; empty location defers to executor `defaultLocation="global"`, unset
  project keeps the loud error (no silent default). **Live-verified**: env-unset → loud error preserved;
  `AILANG_CLOUD_PROJECT` set → error moved past "GCP project not set" to Vertex `HTTP 400: Resource setup
  has just started` (project REACHES the backend — the "resource setup" state is fleet (c) M0/M4 territory).
  Non-vacuous `t.Setenv` regression test. **Fleet (c)'s M0/M4 gemini reviewer lane is now UNBLOCKED** —
  next fleet step is (c) M0 (live gemini network probe) → M4 (conditional on M0) → M5 (bounded live-fire +
  doc → implemented/).
- **(c1) [LANDED 2026-07-17 iter 39 → implemented/v0_30_0; PR #405 → `ae5f0a00f`, eval PASS 96/100 r1]** m-gemini-evaluator-diff-bridge — Mark's #399
  directive ("default evaluator = gemini if able to git clone the codebase, otherwise sonnet-5")
  forced fleet (c)'s M0 live probe early. **Two findings**: (1) **M0 live gemini probe TIMED OUT**
  (`ailang exec gemini` → `http2: timeout awaiting response headers` on the Vertex
  `interactions` POST — the request reaches the backend but no response returns; same class as the
  iter-37 "Resource setup has just started"). Backend reliability is still unproven — M4/M5 stay
  blocked on it. (2) **The evaluator role needs a diff-bridge, not just the executor** (extends
  fleet (b)'s note): the managed_agents request body carries only `Directive`+`SystemPrompt`
  (`managed_agents.go:164`), so even a READ-ONLY evaluator sees NO local repo — it cannot inspect
  the sprint's uncommitted worktree changes nor re-run tests. To make gemini a real evaluator: ship
  the `git diff` + changed files INTO the directive text (mirror `managed_agents_bridge.go`), accept
  it's reasoning-only (no local test re-runs), AND land backend reliability. **BOTH DONE (iter 39)**:
  backend reliability confirmed (4/4 bounded probes SUCCESS); the diff-bridge capability shipped
  (`internal/eval_harness/gemini_evaluator_bridge.go` — `BuildDiffBundle` untracked-inclusive +
  reasoning-only directive + `GeminiVerdict` + `RunGeminiEvaluator` injectable caller seam +
  caller-enforced `VerificationDegraded`; PASS 96/100 r1). Default evaluator STAYS **sonnet** —
  capability only; a gemini-default flip needs a live diff-bridge fire + the ≥3-datapoint evidence rule.
**[GAP CLOSURE PRIORITY — Mark 2026-07-17: "I want the gaps here worked on as priority"]**
Work these BEFORE returning to the clause queue; one per iteration, cheapest-confirmation-first:
- ~~**(G1) gemini FIRST LIVE ROLE FIRE**~~ **CONFIRMED iter 43** — live `ailang design-quorum` →
  `gemini-3-1-pro` **present, verdict=reject, $0.023** (its first clean live reviewer verdict). The
  evaluator arm (`RunGeminiEvaluator`, PR #405) has no CLI seam yet, so the **quorum-reviewer seat**
  (G1's explicit OR) carried it. Reliability blocker found+fixed same iteration: gemini's THINKING
  tokens overran the `reviewMaxTokens=4096` cap → intermittent silent-truncation N-1 quorum (PR #408
  → `885725f06`: cap→16384, fail-loud on `finish_reason=length`, wired gemini `finishReason`). Log 48.
- ~~**(G2) 3-provider quorum CONFIRMATION round**~~ **CONFIRMED iter 43** — same live quorum:
  `gpt5-6-sol` (OpenAI, restored post-#407) + `gemini-3-1-pro` (Google) BOTH present + claude
  controller = 3 providers, both `reject`. The solo-gemini-veto era is over. Log 48.
- ~~**(G3) DESIGNER ROTATION live test**~~ **CONFIRMED iter 44** — `codex:gpt-5.6-sol` (rotation next
  after `claude:claude-fable-5`) authored the G4 design doc via the cross-provider `workspace-write`
  worktree recipe carrying the design-doc-creator directive (**first codex-designer fire**), then ran a
  competent objection-addressing revision. The rotation MECHANISM works end-to-end (design → quorum-gate →
  revise). Evidence row: `(designer=codex:gpt-5.6-sol, quorum=reject→revise→reject over 2 rounds × 3
  providers)` — the content reject is the quorum enforcing data-before-conclusions (unverified external
  contract), NOT a designer failure. Rotation state advanced to `codex:gpt-5.6-sol`; next new-doc iteration
  returns to `claude:claude-fable-5` (gemini joins after G4). Log 49.
- **(G4) gemini REPO-MOUNT upgrade** — **[✅ FULLY LANDED + LIVE-VERIFIED iter 53 — Mark "vertex git clone
  test granted" (#399 2026-07-18T11:59:47Z). The last INCORPORATED premise (provider `git fetch --depth 1
  <sha>` support) is now VERIFIED-LIVE: `TestLiveCloneOverEgressE2E` pinned a real non-HEAD SHA `80cbd9612…`
  through the production `Executor.Execute` path → fetch-by-SHA → exact-SHA echo → `CLONE_OK` → PASS (113.6s,
  $0.865, 527k in/8.2k out). Doc + sprint-plan MOVED to `implemented/v0_30_0/`. **Fleet role (reported to Mark,
  ratification parked):** gemini/managed_agents is now a proven in-sandbox EVALUATOR/reviewer (clone→`ailang
  check`→verdict; Google provider = valid generator≠judge) — but `CapRemoteSandbox` means it canNOT edit a
  worktree, so "gemini joins the DESIGNER rotation" needs the text-bridge and is NOT auto-wired; recommend
  gemini enter as evaluator. **→ RATIFIED by Mark 2026-07-18 (interactive, after cost review):
  gemini is ADMITTED to the fleet as the ESCALATION-TIER in-sandbox evaluator/reviewer — NOT
  every-iteration (sonnet stays the default evaluator) — with THREE mandatory cost guards:
  (1) ENVIRONMENT REUSE ("environment reuse for sure"): clone once per review target, persist the
  `env_<id>`, reuse across rounds — never re-clone per round; (2) tight directives (targeted
  `ailang check`/grep, no repo wandering); (3) two-tier discipline (text quorum first; in-sandbox
  only when a premise is contested/high-stakes). Cost basis VERIFIED against official docs
  2026-07-18: NO managed-agents premium — standard Gemini token rates only ($1.50/M in, $9/M out
  incl. thought tokens at output rate; our client math reconciles $0.865 = 0.79 in + 0.07 out);
  sandbox compute is FREE during preview. ⚠ WATCH ITEM: at GA, Google adds environment-compute
  charges — re-benchmark the escalation-tier economics when GA pricing lands. Designer rotation
  UNCHANGED (claude⇄codex) pending the text-bridge. Next iteration wires the evaluator seat +
  env-reuse.** Prior: LANDED (code) iter 52; both approved fixes (typed `RequiresEgress`/
  `CapNetworkEgress` gate + `ValidateTaskCapabilities`; bounded-execution) + iter-52 shallow-fetch-by-SHA fix;
  opus executor, sonnet evaluator 91/100. Log 57–58.]** iter-45 refuted the `repository`/`inline`
  mount model (only `gcs`+`skill_registry`; egress OFF by default; egress param "undiscovered"). iter-46
  (Mark #399 → philschmid.de/managed-agents-gh) **found the egress param and superseded the mount model**:
  it is a structured list `environment.network.allowlist:[{domain,transform}]` (not iter-45's scalar
  guesses). Re-probing OUR Vertex endpoint (probes O–R, same ADC harness): `network.allowlist:[{domain:"*"}]`
  is **accepted and provisions an egress-enabled sandbox** (Vertex allows wildcard `*` only today;
  per-domain + header-`transform` = "not supported now"). Probe **R**: an egress-only env (NO data source)
  **cloned the public ailang repo end-to-end** (`git clone` OK, `rev-parse HEAD`=`806b3b4a4`=current dev,
  file listing + `go.mod` returned). **⇒ new dominant option (d) CLONE-OVER-EGRESS:** give the executor an
  egress env + have the agent `git clone` the public repo at a SHA itself, then `ailang check`/review
  in-sandbox — no encoder/GCS/inline/mount. Small; directly delivers #399's "gemini can git clone the
  codebase" for the reviewer role. **Recommendation: (d)** (fallbacks: (a) GCS for *private* code, (b)
  shelve, (c) skill_registry). **DECIDED by Mark 2026-07-27 (attended interactive session, quota-relief
  directive): GREENLIGHT Phase-2 clone-over-egress.** Scope: wire gemini managed-agents into the
  DESIGNER rotation via option (d) (egress env + in-sandbox `git clone` of the public repo at a SHA —
  design docs need only committed HEAD); follow-on under the same greenlight = evaluator-REVIEW lane
  (executor pushes the sprint branch pre-merge → gemini clones the branch, CI stays the test oracle —
  the iter-38 uncommitted-worktree objection does not apply to a pushed branch). Same
  `MISSION_METERED_BUDGET_USD` ceiling; public trace comment on #399. [Historical ask was:
  greenlight the Phase-2 clone-over-egress decomposition, or shelve.] Reproducible probe: `internal/executor/managed_agents/managed_agents_live_test.go`
  (`AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`, CI-inert, probes O–R). Doc RESHAPED with full VERIFIED-LIVE
  contract. Log 51. **Note:** the blog is the Gemini *Developer* API surface (`ai.google.dev`, API-key) —
  a different contract from the Vertex executor; our-project Developer-API confirm is parked (the available
  `GOOGLE_API_KEY` is invalid even for generateContent).
- ~~(G5)~~ **REMOVED from the gap path (Mark 2026-07-17): the qwen3-6 lane is a NICE-TO-HAVE,
  not a gap.** See (d) below — sequenced only after the cloud fleet is fully proven (G1–G4
  done), and NOT at gap priority: after G4, the loop returns to the clause queue; (d) is picked
  on normal cheapest-impact ordering.

- **(d) Phase D — motoko + qwen3-6 local-GPU lane** (fleet doc Phase D, ~2–3d) — **NICE-TO-HAVE,
  post-cloud (Mark 2026-07-17)**: the standing role of this lane is the **local assignee for
  slow-but-free task classes** — long-running, low-urgency work with deterministic verification
  (bulk regens, wide test sweeps, corpus churn) where wall-clock doesn't matter and $0/token does.
  It is NOT a peer of the cloud lanes for interactive-cadence roles. HARD constraints unchanged:
  `rig.lock` two-tier discipline (GPU mutex per-step, never iteration-wide), the port-8080 zombie
  hazard (memory: a hung motoko holding 8080 breaks all later runs), and the same evaluator gate
  as cloud work — no quality discount for free tokens.

**[LANDED 2026-07-23 iter-92 — was HARD PIN, Mark 2026-07-23: PORTABILITY M2+M3]** Mark asked "do we have the ability to work on multiple missions yet?" — answer was NO because M2 (skill repo/verify profiles) + M3 (public bootstrap guide) had been skipped for 8 iterations. **Both landed iter-92** (headless-greenlit, no quorum, Mark-ratified split): M2 = `## Repo Profile` block in SKILL.md + mission-doc charter header (two verify profiles `go-compiler`/`ailang-code`; 3 `--repo` args parameterized); M3 = public `docs/docs/guides/mission-bootstrap.md` + `design_docs/mission-charter-TEMPLATE.md`; dry-run isolation acceptance PROVEN (`mission-worldtest.pid` distinct, v1 untouched); evaluator sonnet PASS 83/100. **Ailang World launch is now UNBLOCKED** (its iteration 0 = charter ratification, attended w/ Mark). The triage sweep below is COMPLETE — its NEXT-FIRST is spent:

**[completed 2026-07-23 iters 85–87: FULL BACKLOG RE-TRIAGE]**
("lets do another review of the docs in planned and see which we can put into the cycle.") The
third triage pass, against July reality: fleet live, fmt shipped, v0.30.0 released, arch-boundaries
landed, quorum-at-pick in force. Sweep **ALL planned/ folders** (~114 docs: root 14 · v0_29_0 38 ·
v0_30_0 19 · v0_31_0 3 · v1_0_0 5 · v1_1_0 30 · docparse-billing 5). Rules:
- **Sequencing**: run AFTER the currently-authorized work (raw-handler M1 · reasoning-effort final
  round · fmt polish pair · strict-fallbacks) — those are decided; triage before picking anything
  NEW beyond them. May span 2–3 iterations (folder-group per iteration; oldest first: root +
  v0_29_0, then v0_30/31/v1_0_0, then v1_1_0 + docparse).
- **Per doc**: reality-check the status claim FIRST (the ghost discipline — cheap live probes for
  bug-claims, `git log --grep` for landed-claims; iteration-0/25/48 precedent: statuses LIE), then
  tag exactly one: **[GATING clause-N]** (serves an open bar clause → queue placement) ·
  **[CYCLE]** (non-gating but net-valuable now → normal v0.3x road, loop may pick when gating
  queue is blocked) · **[POST-V1]** · **[GHOST/SUPERSEDED → close with a CI-enforced guard where
  the claim was a bug]** · **[FOLD-INTO <doc>]**.
- **Controller-lane** (read+verify, no generation — iterations 45/48 pattern; no quorum during
  triage, quorum-at-pick fires when a doc is actually PICKED). Deliverable: triage table on the
  bookkeeping issue + charter queue rewrite + archive moves for ghosts/superseded. Docs promoted
  to [CYCLE] get an explicit one-line WHY (what changed since they were shelved).
- **(B) GITHUB-ISSUE TRIAGE (Mark 2026-07-20: "for v1.0.0 we should also triage all the other
  github issues — see if they have design docs, are stale or defunct superseded etc or need a new
  doc")** — the ~19 open non-thread issues (span Dec 2025–Jul 2026: CI flakies #351/#338,
  test-runner litter #328, runtime/test paths #324/#326, effect-row bug #386, the May
  motoko_explore trio #231/#226/#225, CLI asks #223/#157/#155, Z3 #215, docparse #224/#153/#143,
  Sonar #104, stapledons #18, nightly-watch #384). Use the **github-issue-triage skill**. Per
  issue: reality-check at HEAD (cheap live repro where the claim is a bug — issue bodies age like
  doc statuses), then exactly one: **FIXED → close citing the commit** · **STALE/SUPERSEDED →
  close with evidence** · **COVERED-BY an existing/queued doc → link both ways + tag** ·
  **NEEDS-NEW-DOC → note for the designer rotation (doc authored on pick, quorum applies)** ·
  **GENUINE v1-GATING → clause-tag + queue placement** · **POST-V1 → label + say so on the
  issue**. External-author issues ALWAYS get a reply (public repo — same courtesy as #417).
  Runs alongside/after the doc sweep; same 2–3-iteration budget, same evidence discipline.

  **RE-TRIAGE BATCH 1 — root + v0_29_0 (iteration 85, 2026-07-23; 52 docs, controller-lane, 4
  read-only sonnet workers @ $0).** Full evidence table on #422. Outcomes:
  - **LANDED → swept to implemented/** (7): `m-eval-elo-priority-rotation` (`b3de7e70f` #423 →
    v0_30_0) · `m-eval-local-cloud-unify` (`c533bb51c` → v0_30_0) · `m-eval-regression-detector-
    contract` (all 3 clauses shipped `9a1c43f34` → v0_29_0) · sprint-plan stubs whose design docs
    already landed: `m-eval-bounded-pipeline-sprint-plan` (`d41e43894` → v0_29_0) ·
    `m-file-handling-improvements-sprint-plan` (`8697c9d01` → v0_29_0) · `m-pattern-and-invocation-
    repair-sprint-plan` (v0_22_0 designs → v0_22_0) · `m-ailang-fmt.md` planned STUB deleted
    (superseded by `implemented/v0_30_0/m-ailang-fmt.md` + phases).
  - **GHOST/SUPERSEDED → archived** (`design_docs/archive/2026-07/`, 4): `M-TOOLING-DETERMINISTIC`
    (Mark self-closed `3df673994`) · `m-motoko-editdecl-astedit` (A/B NEUTRAL `978bd371a` — not a
    pass-rate lever) · `motoko-agent-v0.15.0-migration` + `motoko-integration-sequence` (both track
    the arniwesth fork migration; rig moved to internal `mk-ast` → headline outcome unreachable).
    None were bug-claims → archive-with-stamp, no CI guard needed.
  - **GATING-candidate → queued (clause 3/4; VERIFY-FIRST at pick — survey-sourced, inherit ghost
    debt)** (5): **[GATING clause-3, PARKED iter-93 needs-human-review]** `m-pure-prng` (pure cap-free
    `std/prng`; removes the widen-`--caps`-for-reproducible-randomness footgun; small, stdlib-only —
    **CORE proven bit-exact SplitMix64 at HEAD**, revised+re-quorumed; sole block = `split` scope,
    Path X defer / Path Y 2-word `Gen` — Mark decides, then unpark) · **[LANDED iter-98 → implemented/v0_29_0]**
    `m-budget-scoping-bug` (effect `@limit`/`@min` per-function vs cumulative-across-chain — bug-claim;
    live-repro REAL, quorum-ratified hierarchical per-invocation budget frames. Mark 2026-07-24
    "apply and route" ratified the QUORUM narrow-refinement carve-out's FIRST USE; controller applied
    both reviewer-verbatim fixes; planner opus → executor opus (worktree; `internal/effects/budget_frame.go`,
    defer-guarded frame stack, bubbling charge, +2 latent bugs fixed) → evaluator sonnet PASS 87/100 r1;
    PR #474 squash `f1bf7b77c`, required checks green. Follow-up D1 [non-blocking]: budget error omits
    the frame's function-name — see the doc's Future-work). **[GATING clause-4]** `m-agent-step-cancellation` (`std/ai.step` graceful
    abort/SIGINT — no impl) · `m-serve-api-live-tool-registry` (MCP live re-registration of new
    `.ail` tool files mid-session — watcher exists, ~50 LOC missing) · `m-contracts-as-code-vertical`
    (deontic engine landed as `sunholo/deontic` 0.1.0; the four-moat orchestration docs flagship +
    `examples/contracts/` remain — clause-4 showcase).
  - **CYCLE** (net-valuable now, non-gating; picked when the gating queue is blocked): `m-eval-data-
    hosting-decouple` (W6 prod-promote) · `m-eval-os-version-trend-redesign` · `m-eval-reasoning-
    model-fairness` (D4 re-run) · `m-eval-validity-discipline` (W4 remainder) · `m-motoko-compaction-
    quality` · `m-motoko-self-improvement-loop` · `m-ui-vite8-migration` · `m-ailang-semantic-context`
    (R2/R3/R6) · `m-concurrency-leverage` · `m-coordinator-inbox-wildcards` · `m-dashboard-
    simplification` · `m-dx27-docs-search-github-fallback` · `m-eval-slim-prompt-self-discovery` ·
    `m-eval-stream-health-retry` · `m-motoko-ext-per-task`(+sprint-plan) · `m-ollama-v1-streaming-
    idle-timeout` · `m-stdlib-html-streaming` · `m-verify-stdlib-stale-path` (1h dead-gate fix).
  - **POST-V1**: `m-ailang-native-harness` · `m-dynamic-data-runtime-plane` · `m-anthropic-sandbox` ·
    `m-apple-container-local-eval-sandbox` · `m-cascade-observability` · `m-coord-thinking-levels` ·
    `m-eval-openrouter-baseline-rotation` · `m-eval-results-folder-structure` · `m-fable-strategy-
    review` (strategy index, kept as reference).
  - **UNSURE → left in planned, need a cheap follow-up probe before tag**: `m-bytecode-vm-parity-bugs`
    (run parity verify) · `m-contracts-as-code-sprint-plan` (docs-flagship portion open) · `m-dx-agent-
    eval-gaps` (gaps 2–4 status) · `m-eval-rig-reliability`(+`m-rig-reliability-sprint-plan`) (2 P1
    open: docx recording, A/B contamination) · `m-pkg-feedback-loop`(+sprint-plan) (M3/M4 status).
  - `20251013_auto_caps` intentionally kept in planned (M1 `e542065c0` landed; 3 follow-ups open).
  - **Next batch (iteration 86): v0_30_0 + v0_31_0 + v1_0_0**, then v1_1_0 + docparse.

  **RE-TRIAGE BATCH 2 — v0_30_0 + v0_31_0 + v1_0_0 (iteration 86, 2026-07-23; 30 docs,
  controller-lane, 4 read-only sonnet workers @ $0).** Full evidence table on #422. Outcomes:
  - **LANDED → swept to implemented/v0_30_0/** (11): `m-mission-agentic-provider-routing`(+sprint-
    plan) (M1a `8ee07ef23` · M1b `956fda55c` · M2 `8d12e8e9c`; M3 = parked-with-protocol, the
    documented outcome) · `m-mission-fleet-ab-sprint-plan` (A `3bee6b6df` + B PR #383 `1186a48e6`;
    parent `m-mission-adaptive-multiprovider-routing` STAYS planned — C/D/E opt-in open) ·
    `m-mission-quorum-agentic-verify` (M1-M3 PR #400 `0e83a1b12`; header was stale-PARKED) ·
    `m-ailang-fmt-phase2-sprint-plan` (`3815ba617` PR #414) · `m-fmt-properties-printer-roundtrip-
    sprint-plan` (`942931816` PR #424) · `m-smt-callee-sort-gate`(+sprint-plan) (`94e2a5d27` +
    `efd251f16`) · `m-std-yaml`(+sprint-plan) (`62d681a8e`). Plus `m-ailang-fmt-inline-interior-
    sprint-plan` planned-copy DELETED (canonical already in implemented/ from PR #434 `3c1cec57d`).
  - **GHOST/SUPERSEDED**: none this batch — every doc traced to real commits or a live queue item.
  - **GATING-candidate → queued (VERIFY-FIRST at pick — bug-claims must live-repro before routing)**
    (6): **[EVIDENCE-PARKED iter 91 2026-07-23]** `m-parser-block-let-separator` (bug REAL at HEAD
    but evidence gate MEASURED NEGLIGIBLE — 0 decisive occurrences in 27,359 eval files; every
    attributable case is cascade-noise in already-broken output → stays parked, do NOT route a core
    parser change; re-open only on a decisive rotation case) · `m-diag-
    primitive-field-suggestions` (primitive-field "no methods" hint — severed Part C of the landed
    footguns-to-diagnostics doc; P3/extension-lane, frozen-core + ADT-name premise still to resolve).
    ~~**[GATING clause-4]** `m-check-strict-fallbacks`~~ **[LANDED iter 101 → implemented/v1_0_0;
    `STRICT_FALLBACK_001` Core-level post-name-resolution pass, GlobalRef-keyed known-empty-builder
    registry + ANF Let-env resolver, dual channel (dev WARNING / `check --package` HARD ERROR),
    `@allow_empty_ok` opt-out; evaluator sonnet PASS 88/100 r1; PR #479 `1978ab44b`. D1 (match-arm
    scoping for bare `Ok("")`) → Future Work]** · `m-parmap-effectful` (in-AILANG fan-out for the
    orchestration flagship; M0 `EffContext.Clone()` fork-safety `22e4c11b7` is a HARD prerequisite —
    shallow copy panics under concurrency) · `m-effect-replay-contracts` (effect sprint 2/4) ·
    `m-effect-clock-net-fs-modes` (effect sprint 3/4). `m-effect-scope-params` (sprint 4/4) is a
    release-gate RE-SCORE candidate (Mark may push to v1.1); `m-effect-refinement` is the decomposed
    UMBRELLA — stays planned tracking 3 open children (sprints 2/3/4), sweeps only when all ship.
  - **CYCLE** (net-valuable now, non-gating): `m-mission-adaptive-multiprovider-routing` (phases
    C/D/E opt-in loop infra) · `m-mission-portability` (✅ COMPLETE iter-92 — M1 `825e77c64`, M2+M3
    landed; Ailang World UNBLOCKED) · `m-mission-cost-chains` (clause-5 cost-per-verified-success substrate) · `m-ai-
    structured-step`(+sprint-plan) (composable structured output → vision+JSON grading) · `m-
    comments-for-ai-authors` (M1 = $0 prompt-manager lane, Mark-ratified) · `m-eval-kimi-k3-agentic`
    (standard done; agentic entries gated on `m-eval-reasoning-model-fairness` P1) · `m-managed-
    agents-model-eval` (Gemini Developer API pivot design-frozen; blocked only on an AI Studio key)
    · `m-mem-budget-runtime` (P1 host-safety `MEM001` runtime cap — motivated by the 2026-07-20
    kernel panic; design complete, no impl).
  - **POST-V1**: `m-arch-boundaries-eval-exclusion-tighten` (evidence-gated, trigger unmet — no
    second dashboard `internal/eval` import at HEAD).
  - **PARKED (leave as-is)**: `m-decision-entropy-monitor` (needs-human-review since iter 84,
    quorum-blocked ×2). ⚠ INTEGRITY NOTE: the MAIN checkout has UNCOMMITTED in-progress edits to
    this doc (V11–V13 producer-side evidence rows answering the quorum objections) + 3 unpushed
    local commits (`ff089b7eb`/`5753897e1`/`faeb16d13`, `m-managed-agents-model-eval` doc) — local
    `dev` has DIVERGED from origin/dev. NOT touched this iteration (Critical Principle 0); flagged
    for human sync.
  - **Next batch (iteration 87): v1_1_0 (30) + docparse-billing (5)** — closes the full sweep. **DONE below.**

  **RE-TRIAGE BATCH 3 — v1_1_0 + docparse-billing (iteration 87, 2026-07-23; 35 docs,
  controller-lane, 4 read-only sonnet workers @ $0).** Full evidence table on #422. **This CLOSES
  the full planned/ sweep** (batches 1+2+3 = root + v0_29_0 + v0_30_0 + v0_31_0 + v1_0_0 + v1_1_0 +
  docparse-billing). Outcomes:
  - **LANDED → swept to implemented/** (4 docs / 2 features): `m-type-v2-migration`(+sprint-plan) →
    **v0_10_0** (`b29c391ff` delete legacy TFunc + `a314e7fca` open-effect-row fix; verified `TFunc`
    fully gone from `internal/types/` — comment refs only) · `m-executor-variants`(+sprint-plan) →
    **v0_15_0** (`721550fdb` M1 wiring + `c07bf73c1` codex/opencode images + `af36a00a1` gemini/eval
    images; `ExecutorVariant` live in `internal/coordinator/`; the FULL design promoted OVER the
    rough-draft stub that already lived at v0_15_0 and cross-linked back to planned).
  - **GHOST/SUPERSEDED**: none this batch — every doc traced to real commits or clearly-open work.
  - **GATING-candidate → queued (VERIFY-FIRST at pick — survey/bug-claims must live-repro before
    routing)** (11):
    - **[GATING clause-3]** accessibility / syntax-ergonomics (5): `m-error-propagation` (`?`
      operator — **LIVE-CONFIRMED bug at HEAD**: `PAR_NO_PREFIX_PARSE: unexpected token: ?`;
      LIMITATIONS lists "not yet implemented"; pure desugar ~2d — cheapest high-confidence pick) ·
      `m-pipe-operator` (`|>` — no `PIPE_GREATER` token / `PipeApp` node; clean design, zero deps,
      ~6–8h) · `m-dx-package-dogfooding` [**LANDED 2026-07-23 iter 90** — jint `ceecdd0f1` (PR #467);
      Issues 1/2 prior `7d1e4b82a`/`99f76ec7a`; doc → `implemented/v1_1_0/`] · `m-call-sugar-optional`
      (`f()` optional call sugar — still parse-errors at HEAD; ~1–2d) · `m-forall-properties-direct-
      core-eval` (`properties [forall(...)]` → "empty program" via the broken source-synthesis path
      `internal/testing/runner.go`; sibling `ensures`/`requires` fixed `3ebf60b1b`; doc self-labels
      low-priority "zero users"; ~3–4h).
    - **[GATING clause-4]** agent-orchestration surface (6): `m-effect-handlers` (Koka-style algebraic
      effect handlers — enables deterministic AI-mock handlers from `.ail`; Phase 1 ~38h, cross-cuts
      parser/types/elaborate/eval; LARGE, high-value) · `m-ai-effect-modes-followups` (replay/byok/
      reroute runtime — TODO at `internal/ai/routing.go:157`; items 1+2+4 close the A1/A2 replay-
      determinism story ~3–4d; VERIFY-FIRST item-1 vs already-shipped M-AI-TOOL-LOOP) · `m-agent-
      safe-runner` (safe runner — **M1 policy spike LANDED `4effc002d`** [`internal/policy/`]; M2–M5
      unstarted ~5–7d; one transitive-import-closure design-freeze item) · `m-agent-loop-architecture`
      (runTools hook-extension ADTs — design decision A/B/C unresolved, needs arni input; ~5d;
      VERIFY-FIRST the design freeze first) · `m-process-modes` (`Process[mode=mocked]` runtime replay
      — parser `[mode=...]` landed but the runtime is absent; ~36h; blocked on M-EFFECT-REFINEMENT
      full landing) · `m-agent-orchestration` (`std/agent` effect — LARGE ~2–3wk, the big open
      orchestration surface; DECOMPOSE before executing).
  - **CYCLE** (net-valuable loop/rig/DX infra, non-gating) (9): `m-oracle-adequacy` (eval evidence-
    bundles) · `m-trace-feedback` (`ailang trace diagnose`) · `m-entropy-budgets` (design-doc
    completeness infra) · `m-d4-design-doc-driven-development` (`ailang verify --spec` compliance;
    budget substrate partial) · `m-pkg-inflight` (cloud/Firestore package events) · `m-zero-language-
    learnings` (Phase-1 `check --json` landed; Ph1.5–3 rig/DX meta) · `m-eval-finetuning-data-pipeline`
    (rig fine-tuning loop) · `m-eval-trust-signals` (eval credibility / HumanEval port) · `dx-
    improvements-from-billing-packages` (partial-landed external DX friction log — FIXED items code-
    verified, open child items live in their own `m-dx-package-check`/`-test` docs).
  - **POST-V1** (real but out of the v1 LANGUAGE scope) (11): `m-csp-session-types` (6–8wk session
    types) · `m-eu-compliance-effects` (author-downgraded, domain lib) · `m-game-engine-effects` (v1.1
    domain lib) · `m-perf4-bytecode-interpreter` (perf stretch, doc says v2.0) · `m-quasi-typed-
    quasiquotes` (parse-only lexer/AST, no runtime; 3–4wk) · `m-reflect-structural-reflection` (class/
    instance parser TODO, 2wk) · `global-collaboration-hub` (cloud infra, Mark-downgraded non-gating)
    · the **docparse-billing/ cluster** = `m-billing-docparse-billing-agent-payment` +
    `responsibility-docparse` + `responsibility-multivac` + `responsibility-packages` (external
    DocParse-billing SaaS spanning `docparse`/`ailang-multivac`/`ailang-packages` repos — NOT AILANG
    language work).
  - **Tally**: 4 LANDED / 0 GHOST / 11 GATING-candidate / 9 CYCLE / 11 POST-V1 = 35. Zero GATING-
    candidate touches the frozen core beyond parser/desugar surface; the clause-4 orchestration items
    are the heavy ones (effect-handlers, std/agent, safe-runner) and need decomposition before route.
  - **Sweep-complete → NEXT for iteration 88**: the GATING backlog is now fully surfaced across all
    three batches. Cheapest-first pickable order (once live-verified at pick): ~~`m-error-propagation`
    (clause-3, live bug, ~2d)~~ **[PARKED needs-human-review iter 88 — `?`-op REAL at HEAD but the
    design quorum-blocked ×2 (Rev-0 fatal local-desugar flaw fixed → Rev-1 re-quorum surfaced deeper
    open questions: compiler↔stdlib lang-items linkage for user-space `Ok`/`Err` ConstructorIDs +
    unverified `core.go:309`/`normalizeToAtomic` lowering premises). Rev-1 design + open questions
    preserved in `planned/v1_1_0/m-error-propagation.md`; unpark needs an arni/human architecture
    decision]** → ~~`m-pipe-operator` (clause-3, ~6–8h)~~ **[PARKED needs-human-review iter 89 — `|>`
    REAL at HEAD (`42 |> show` → `PAR016`) but Rev-1 re-quorum came back 1 pass (gemini) / 1 reject
    (gpt5-6-sol). Rev-1 fixed both Rev-0 objections (LHS-first `Let` desugar for eval-order; 14-row
    Conflict Surface + frozen-core/AILANG-fix-lane justification). The sole remaining objection
    (non-callable/arity>1 RHS "may fail only at runtime") was **controller-VERIFIED REFUTED** — both
    `x |> 42` and arity>1 RHS are caught at type-CHECK time (`No instance for Num[int -> int]`;
    `TC_ARITY_001`). Parked per the one-re-quorum gate, NOT force-passed. **Unpark is LOW-RISK: route
    straight to sprint-planner, no design change needed.** Rev-1 + refutation in
    `planned/v1_1_0/m-pipe-operator.md`; PR #466]** → **iter 90 LANDED `m-dx-package-dogfooding`
    jint micro (PR #467 `ceecdd0f1`; doc closed → implemented/v1_1_0). iter 91 EVIDENCE-PARKED
    `m-parser-block-let-separator` (measured negligible, 0/27,359 decisive). iter 92 LANDED
    `m-mission-portability` M2+M3 (loop now portable). iter 93 `m-pure-prng`: quorum caught real
    soundness defects → 1 designer-revision fixed the CORE (proven bit-exact SplitMix64) → re-quorum
    still blocks on `split` → PARKED needs-human-review (Path X defer / Path Y 2-word `Gen`). NEXT for
    iter 94:** `m-budget-scoping-bug` (GATING clause-3 bug-claim, live-repro first) OR unpark
    `m-pure-prng` on Mark's Path pick. clause-4 effect chain (replay→clock/net/fs→scope)
    stays sequential; the big orchestration items (`std/agent`, effect-handlers) need decomposition
    iterations first.

**[NEXT]** clause-3 accessibility cluster (the bulk of v1.0). Loop ordering within a group:
P0/unblockers first, then cheapest impact-per-day. The DOC-READY/small diagnostics AND the
VERIFY-then-route backlog are now EXHAUSTED (module-less/xcheck/json-bool/split-arg landed iters
14–17; both VERIFY-then-route items closed as ghosts iter 18). **Iteration 25 (2026-07-13)
Gate-2 reality check found the strategy review's R4 rows were never individually re-verified:
R4a `m-dx-match-hof` and R4b `m-poly-arith-lambda` are BOTH GHOSTS** (R4a: original failure used
the retired `match … with` syntax, brace-form works in every probed position, design doc was
already archived Not-Applicable 2026-05-09; R4b: fixed v0.7.0, verified incl. one let-bound
lambda at BOTH int and float) — guard examples `match_hof_lambda.ail` + `poly_arith_lambda.ail`,
PR #379 → `ea8116f83`, dev CI green observed. Same iteration EXECUTED
**m-lambda-open-record-pattern** (REAL at HEAD — doc existed at planned/v0_29_0, so NOT NEW-DOC;
see queue item 16). The parser/type footgun row is now FULLY BURNED DOWN (m-xmod-alias-poly
landed iter 26). **Iteration 27 (2026-07-14) opened the Prelude/discovery group:
m-prelude-option-result LANDED (PASS 98/100 round 1, mission high; PR #382 → `d26215341`) +
m-prompt-option-none-idiom closed SUPERSEDED by it. **Iteration 29 (2026-07-14) EXECUTED
m-dx-examples-coverage → LANDED** (doc re-scoped through the FIRST LIVE 5-round quorum, PR #392
→ `3d451947c`, eval round-2 PASS after a one-line Windows fix `881711325`; 5 red examples
quarantined under #386, verify-examples now a REAL CI gate, docs --examples un-inert —
doc → implemented/v0_30_0). **Iteration 30 (2026-07-14) EXECUTED the last clause-3 starter
m-dx-ai-discovery → LANDED** (RESUMED from a died-mid-execution prior run [transient Anthropic
rc=1 at 16:05, pre-dating the 17:16 driver-retry fix]; doc re-scoped + quorumed by that run at
`39d671a52`, executor completed M1/M3/M4/M5, PR #393 → squash `c07c36b25`, eval round-1 PASS
93/100 + hardening `ea6069815` [arrays→array alias] + Windows guard fix `0ad27444c`. Interleave:
dev went RED mid-iteration from sibling M-STD-YAML/M-SMT merges — fixed forward direct-to-dev
`9a314772d` [yaml builtin golden + Z3-gating verify e2e] + `4caddfd23` [>800-line split of
verify.go/codegen.go]; `ailang docs --all-functions [filter]`, unknown-module did-you-mean +
module list, `ailang docs prelude`, V16 effect-row fix — doc → implemented/v0_30_0).**
**Iteration 22 (2026-07-13) front-ran R4a with a regression-derived NEW-DOC pick** (nightly
`higher_order_functions` triage → real decl-class resolver gap #366); **iteration 23 EXECUTED it
→ LANDED** (PR #368 → `fd38ec14e`, eval PASS 98/100 round 1 — queue item 15). Full inner-loop
sprints, NOT bookkeeping.
*(m-match-xcheck-error-quality LANDED iter 15; m-dx-json-bool-coercion in-repo half LANDED iter 16
[`std/json.asBoolLoose`; Phase-1 firestore fix PARKED out-of-repo]; m-dx-split-argument-warning LANDED
iter 17; m-dx-record-cons-pattern + m-dx-tapp-trecord-unification GHOSTS/verified-closed iter 18;
m-arity-style-diagnostic (R4c) LANDED iter 21 [TC_ARITY_001, PR #363 → `5b54509d1`] —
all → implemented/v0_30_0.)*

*(SCOPE EXPANDED 2026-07-12, Mark — full-v1.0 triage of the 69 non-gating docs. The clause-3
accessibility cluster, BOTH DX tooling investments, and the FULL clause-4 orchestration surface
are all IN. v1.0 = the complete "verified AI-orchestration language, accessible to mid-tier
models" — ~33 open items, ~40–55 sprint-days. Rig/cloud/motoko/post-v1 infra stays OUT. Full
triage evidence = log entry 10.)*

### Clause 3 — fleet-tier accessibility (the footgun burn-down; the thesis's core deficit)
- **Parser/type footgun fixes** (NEW-DOC, Conflict Surface mandatory): ~~m-module-let-func-resolution~~
  **[LANDED iter 23 → implemented/v0_30_0; unified SCC over lets+funcs, wrapInLets deleted, module
  letrec via core.LetRec, MOD007 dup-name, truthful hint; PR #368 → `fd38ec14e`, eval PASS 98/100
  round 1]** ·
  ~~m-dx-match-hof (R4a)~~ **[GHOST iter 25 — retired `match … with` syntax was the real culprit,
  brace-form match works in every probed position (block-body/direct/mid-block/nested-HOF/curried
  foldl); `\x ->` wrong-arrow already has a teaching diagnostic; guard
  `examples/match_hof_lambda.ail`, PR #379 → `ea8116f83`]** ·
  ~~m-poly-arith-lambda (R4b)~~ **[GHOST iter 25 — fixed v0.7.0 (m-poly-arithmetic-fix); verified
  incl. let-bound lambda at BOTH int and float; guard `examples/poly_arith_lambda.ail`, PR #379 →
  `ea8116f83`]** · ~~m-arity-style-diagnostic (R4c, 1–2d)~~ **[LANDED iter 21 →
  implemented/v0_30_0; `TC_ARITY_001` coded/directional/style-aware arity diagnostic at
  `unification_types.go`, 5 golden/regression tests, eval PASS 97/100 round 1, PR #363 →
  `5b54509d1`]** · ~~m-lambda-open-record-pattern (1d)~~ **[LANDED iter 25 → implemented/v0_30_0;
  `{name, ...}` in lambda params now infers OPEN `{name: τ | r}`; PRIMARY root cause was
  `unifyRecord`'s pre-row field-count rejection (deeper than the doc's hypotheses) + Rest erased
  at AST→Core; closed-pattern strictness preserved + arm-order-independence hardened + cacheKey
  v2; eval PASS 92/100 round 1, PR #380 → `47576e25d`, dev CI green per-workflow observed]** ·
  ~~m-xmod-alias-poly (1–2d, VERIFY-FIRST)~~ **[LANDED 2026-07-14, iter 26 →
  implemented/v0_30_0; VERIFY-FIRST probe confirmed REAL at HEAD (NOT a ghost — but the NEW-DOC
  tag was wrong, full doc existed at planned/v0_29_0); parameterized aliases now instantiate
  (`Box[int]` → `{items: [int]}`, single- + cross-module) via `expandAlias` `*TApp` branch keyed
  strictly on alias-env membership (ADTs stay nominal, proven); `TC_ALIAS_ARITY_001`; cacheKey
  v3; eval PASS 93/100 round 1 (first zero-correction pass); PR #381 → `fd1b11a47`, dev CI green
  per-workflow observed]** · **m-parser-block-let-separator** (PARKED, evidence-gated, split out
  of m-dx-expected-fail-fixes iter 40 → planned/v0_30_0): a simple-RHS `let x = e` tolerates
  eliding the statement separator before a trailing expr, but a block-RHS `let x = match{...}`
  does not — a minor parser ASI inconsistency. NOT auto-fixed (default-bias-not-core); route only
  with a measured eval failure-rate + Conflict Surface.
- **VERIFY-then-route** (ran the doc repro FIRST — both were ghosts): ~~m-dx-record-cons-pattern~~
  **[LANDED/GHOST iter 18 → implemented/v0_30_0; `{…} :: rest` type-checks; guard
  `TestListConsPatternWithRecord` + `examples/record_cons_pattern.ail`, PR #358 → `adde9e9d0`]** ·
  ~~m-dx-tapp-trecord-unification~~ **[LANDED/GHOST iter 18 → implemented/v0_30_0; `[[TableCell]]`
  extraction type-checks; guard `examples/record_list_extraction.ail`, PR #358 → `adde9e9d0`]**
- **Diagnostics** (DOC-READY / small): ~~m-module-less-run-fail-loud (MOD014)~~ **[LANDED iter 14 →
  implemented/v0_30_0]** · ~~m-match-xcheck-error-quality~~ **[LANDED iter 15 → implemented/v0_30_0]** ·
  ~~m-dx-json-bool-coercion~~ **[in-repo half LANDED iter 16 → implemented/v0_30_0 (`std/json.asBoolLoose`);
  Phase-1 firestore-package fix PARKED out-of-repo in `ailang-packages`]** ·
  ~~m-dx-split-argument-warning (1d)~~ **[LANDED iter 17 → implemented/v0_30_0; compile-time
  non-blocking reversed-`split` warning, extensible `swapTraps` table, PR #356 → `8339b6421`]**
- **Prelude / discovery**: ~~m-prelude-option-result (Some/None no-import, 1.5d)~~ **[LANDED
  2026-07-14, iter 27 → implemented/v0_30_0; Gate-2 probe confirmed REAL at HEAD (`undefined
  variable: Some`/`Err` without import); planner CORRECTED the doc's mechanism (the proposed
  `InjectPreludeValues` never existed — real fix = implicit lowest-precedence std/option +
  std/result imports at ONE loader call-site consumed by both compile and runtime, entry-modules
  only); explicit imports + local types shadow cleanly, library modules unchanged, no
  cacheKeyVersion bump (verified); 15 new tests, 0 deletions; eval PASS 98/100 round 1 (mission
  high; 20 adversarial probes incl. entry-only through real multi-module runs + PR-#381 alias-env
  non-interaction); PR #382 → `d26215341`, dev CI green per-workflow observed]** ·
  ~~m-dx-ai-discovery (2d)~~ **[LANDED iter 30 → implemented/v0_30_0; re-scoped (one-shot
  discovery: docs --all-functions, unknown-module recovery, docs prelude, V16 fix); PR #393 →
  `c07c36b25`, eval PASS 93/100 round 1]** · ~~m-dx-examples-coverage (1d)~~ **[LANDED iter 29 →
  implemented/v0_30_0; first live 5-round quorum subject; PR #392 → `3d451947c`; 5 red examples
  quarantined under #386; verify-examples now a real gate + validate_manifest --ci wired;
  docs --examples un-inert via manifest `modules` field]** ·
  ~~20251013_auto_caps (infer caps, 2d)~~ **[M1 LANDED iter 32 (kept in planned/v0_29_0 — 1 of 4
  phases): `ailang run --caps auto` infers the entrypoint's effect row + grants exactly those
  (planner refuted the doc's ~200-LOC new-package mechanism → 74-line reuse of the existing
  `iface`/`TFunc2`/`EffectRow` path); FIRST cross-provider codex live-fire (executor = OpenAI
  gpt-5.6-sol, evaluator = Sonnet PASS 98/100 r1), PR #397 → `e542065c0`, all required checks green
  observed. Deferred: `--auto-caps` flag, `AILANG_AUTO_CAPS` env, always-on preflight+exit-2,
  bench-harness integ, cap manifest]** · ~~m-dx-expected-fail-fixes (1–2d)~~ **[GHOST-CLOSED
  2026-07-17 iter 40 → implemented/v0_30_0; Gate-2 live-repro CONFIRMED largely-ghost — 0 of 4
  "bugs" needed a language fix. Bug4 effect_budgets: `@limit` enforcement WORKS at runtime
  ("budget exhausted: semantic limit=3"); the doc's repro put `--caps` AFTER the filename where
  it's ignored (flag must precede the file). Bugs1/2 (arrow-lambda, multi-`requires`) + the 2
  match_foreign files: good teaching diagnostics / intended type-rejections, not bugs. Bug3
  serve_api_webhook: non-canonical example (omitted `;`/`in` after a block-RHS `let`, deprecated
  string `++`). CLOSED with regression guards: the 3 parser-bug examples fixed to canonical syntax
  + promoted to `examples/runnable/` (now CI-gated), effect_budgets README corrected, manifest
  de-drifted (2 mispathed contracts entries repaired). Executor Opus / evaluator Sonnet PASS
  92/100 r1 (generator≠judge). Split-out: the block-RHS-`let` separator ASI inconsistency →
  new backlog `m-parser-block-let-separator` (evidence-gated, default-bias-not-core). PR #406]**
- **Prompt teaching** (batchable, ~0.5d each): ~~m-prompt-option-none-idiom~~ **[SUPERSEDED
  2026-07-14 by m-prelude-option-result's structural fix (its own doc named this band-aid as
  superseded-on-ship); prompt v0.16.2 already teaches the prelude availability; doc → archive/
  with library-module caveat noted]** · ~~m-prompt-single-file-module · m-prompt-split-list-operations ·
  m-prompt-log-file-analyzer-string-ops~~ **[CONSOLIDATED iter 47 into `m-prompt-footguns-to-diagnostics`,
  RATIFIED by Mark 2026-07-18, LANDED iter 54 (2026-07-18) → implemented/v0_30_0.** Part A (PRIMARY):
  wired dormant `MOD002` + new `PAR_MODULE_PLACEMENT` at `parseTopLevelDecl` (mirrors `reportMisplacedImport`)
  + gemini's error-recovery state-isolation fix (two late modules → `PAR_MODULE_PLACEMENT`×1 + `MOD002`×1
  genuine-dup, never a FALSE MOD002) — the ~10% multi-module footgun's opaque `PAR_NO_PREFIX_PARSE` cascade
  is now a coded teaching diagnostic. Part B: split-list-operations GHOST-closed with CI-gated
  `examples/runnable/split_map_join.ail`. Part C (primitive-field, ~2%) SEVERED → extension backlog stub
  `m-diag-primitive-field-suggestions.md`. Planner opus / executor opus / evaluator **sonnet** (generator≠judge)
  **PASS 91/100 round 1**; 3 superseded prompt docs → archive/. No re-quorum (Parts A+B unanimous both rounds).
  Log entry 59.]**
- **DX tooling** (Mark: both in → resolved 2026-07-18: M-TOOLING-DETERMINISTIC **CLOSED-SUPERSEDED**
  by Mark; fmt is the DX item): **m-ailang-fmt [LANDED 2026-07-19 (iter 56)]** — `ailang fmt [--write]
  [--check]` canonical formatter shipped, doc → [implemented/v0_30_0/m-ailang-fmt.md](implemented/v0_30_0/m-ailang-fmt.md).
  New `internal/format` package (exhaustive precedence-aware AST→source printer, no `String()` fallback),
  `cmd/ailang` fmt subcommand (stdout/`--write`/`--check`, atomic same-dir-temp+`os.Rename`, exit 0/1/2),
  opt-in lossless lexer comment scan; `internal/ast/print.go` untouched; newline-per-statement braced
  canonical form, Phase-1 fail-CLOSED on comments (exit 2, byte-identical). Author codex-rotation doc
  (quorum-complete, no re-quorum per Mark ratify). Planner opus / executor opus / evaluator **sonnet**
  (generator≠judge) **PASS 87/100 round 1**. **Controller independent verification caught + fixed a real
  defect the corpus test missed** — the explicitly-pure empty effect row `! {}` was dropped (round-trip
  failed on the doc's own V2–V5 idiom; `ast.FormatEffects` collapses nil vs non-nil-empty; no comment-free
  example uses `! {}`) → `formatEffectRow` helper at all 3 sites + regression fixtures (`0b983a8f8`); 2
  evaluator lint nits cleaned (`305a37dd6`). `metered=$0.00`. Log entry 61. ·
  **m-ailang-fmt-phase2 [LANDED 2026-07-19 (iter 63) — PR #414 squash `3815ba617`; evaluator (sonnet) PASS 78/100 r1; doc → `implemented/v0_30_0/m-ailang-fmt-phase2.md`]**
  — Executor (opus, isolated worktree) shipped M0–M3 (6 commits `83f7ebf23`→`b29e871c4` + lint fix `fe236572c`).
  **Corpus gate V22**: 386 parse-valid → 327 formatted, 0 comment-loss, 0 Phase-2 round-trip regressions, interpolation-refusal 0/386, idempotence 299/299.
  **Calibrated fail-closed boundaries** (both never-lossy, in Future Work): (1) 15.28% (59/386) inline-interior refusal
  (`let … in` chains the parser collapses → no stable idempotent boundary → exit 2, byte-identical) → **follow-up sub-sprint queued below**;
  (2) 28 pre-existing Phase-1 `properties[...]` printer round-trip bugs surfaced (verified not caused; fail comment-free r/t on dev too) → **separate item queued below**.
  Controller fixed 3 sprint-introduced lint issues + moved the doc. Was: DOC CREATED + QUORUM-BLOCKED ×3 → PARKED; UNPARKED by Mark option (b) `c624b456d` (iter 62); planned iter 62.
  — Phase-2 lossless comment preservation, the UNBLOCK for fmt on the 94.7% commented corpus. Doc authored iter-59,
  Rev-3 iter-60 (`design_docs/planned/v0_30_0/m-ailang-fmt-phase2.md`, `d1ed2fe57`); token-anchored envelope
  (AST spans proven unusable at design time). Rev-3 FIXED the 2 R2 defects (V21: **386/393 parse-valid** via
  `ailang check`; hard-left-wall widening clause), but the re-quorum surfaced **2 NEW architecture-level objections**:
  (a) gpt5-6-sol → attacher-totality inventory unproven (no code-audit of all printer child-list boundaries —
  params/type-args/ctor-args/record-fields/annotations); (b) gemini → interpolation clamping structurally fatal
  (collapses inner-AST boundaries; would silently delete comments in `${…}`). **→ RESOLVED by Mark 2026-07-19
  (interactive, option (b) + recommendations): UNPARKED, [NEXT] route to sprint-planner, do NOT re-quorum.**
  (1) M0 of the sprint = the PRINTER CHILD-LIST CODE AUDIT — proven inventory folded into the design before
  attachment code; (2) interpolation = FAIL-CLOSED CARVE-OUT (preflight refuses files with comments inside
  `${…}` holes — silent deletion structurally impossible; full interpolation-aware attachment deferred,
  evidence-gated on measured refusal rate, expected ≈0). Doc Status stamped. ·
  **m-ailang-fmt-adoption [LANDED 2026-07-20 (iter 65) — PR #415 squash `b787bb98f`; executor (opus, worktree) M1–M3, evaluator (sonnet, generator≠judge) PASS 89/100 r1; teaching prompt v0.16.3 (append-only) + `formatter.md` Adoption section + `make fmt-check-ail` (renamed off the Go gofmt gate; `make ci` byte-identical) + opt-in `format_ail.sh` hook w/ Mark-approved SIGTERM→grace→SIGKILL escalation; doc → `implemented/v0_30_0/m-ailang-fmt-adoption.md`. Controller disproved a false "docs build failure" (skipped CI-only sync-registry.sh gen step). Was: IN-SPRINT plan iter 64; UNBLOCKED iter 63; DOC CREATED + QUORUM-BLOCKED ×3 → PARKED; Rev-3 iter 60]**
  — discoverability + opt-in hooks. **NOTE (iter 63): with phase2's 15.28% inline-interior fail-closed refusal, a per-turn
  auto-`fmt --write` hook still no-ops on ~15% of real commented files — adoption scope should account for this (teach
  fmt in the prompt + `--check`/`--write` hooks are now viable for the ~85% lossless majority; the ~15% refuse cleanly, exit 2).**
  Rev-3 iter-60 (`…/m-ailang-fmt-adoption.md`, `d1ed2fe57`). Rev-3 FIXED the jq
  defect (`command -v jq` probe + dropped first-jq `2>/dev/null`); re-quorum accepted it but both reviewers reject
  the timeout fix (SIGTERM-then-unbounded-`wait` wedges on a signal-ignoring proc) — **1 trivial SIGKILL-escalation
  from clean**, but hard-gated behind phase2. **→ Mark 2026-07-19: SIGKILL-escalation correction APPROVED as
  written in the doc; no re-quorum; still rides behind phase2** (which is now unparked). Original scope retained below.
  — Mark #399: "Is ailang fmt discoverable by agents via prompt… run every turn after .ail writes by Motoko or a
  hook in other harnesses?" iter-58 findings: (1) `ailang fmt` is **NOT** in `ailang prompt` (embedded v0.16.2
  teaches `check`/`run`/`test`, not `fmt`) → agents don't know it exists. (2) A per-turn auto-`fmt --write` hook
  (Claude Code PostToolUse on `*.ail`, Motoko per-edit) is **near-useless pre-Phase-2** — it would exit-2/no-op on
  87.5% of real files. Scope once Phase-2 lands: teaching-prompt line + CLI discoverability + opt-in harness hooks
  (`--check` in CI, `--write` post-write). Deliberately NOT teaching fmt in the prompt yet — teaching a tool that
  refuses 87.5% of commented files would frustrate agents (no-premature-adoption). ·
  **m-ailang-fmt-inline-interior** (**[LANDED 2026-07-21 (iter 70) — PR #434 squash `3c1cec57d`; UNPARKED by
  Mark #422 "Continue the AILANG fmt sprint" (bc61ea8ce: quorum objections DATA-REFUTED, proceed on the data, no
  re-quorum); planner (opus) → executor (opus, worktree) M0–M3 → evaluator (sonnet, generator≠judge) PASS 91/100 r1.
  Printer-local conditional multi-line let-chain emitter (option (a)); `internal/parser`+`internal/ast` READ-ONLY.
  **M0 surface-AST gate** (`TestInlineInterior_LetChainSurfaceShape`) proved all 28 targets chain via nested
  `*ast.Let.Body` (0 `Block.Exprs`) — the iter-67 R2 quorum objection data-refuted by a real TEST, not an assertion.
  `comment-unattached` refusals **59→32** (15.28%→8.27%, 27-file/45.76% reduction); `let-chain-interior` sub-class
  **fully eliminated (==0)**; residual 32 = deferred non-let/inline-tests/no-enclosing-list classes (records.ail's
  footer comment keeps it fail-closed → 27 achievable not 28). Idempotent, never-lossy (marker-fail=0,
  PHASE2-rt-regression=0). doc → `implemented/v0_30_0/`. **Orphaned-crashed-run recovery**: iter-70's inner loop
  ran + finalized + opened PR #434, then an 18h reboot outage killed it pre-Gate-3b/4; THIS run (same iter-70)
  resumed at Gate-3b — diagnosed the docs-`build` red as a **stale-base mermaid npm ERESOLVE** (branch cut before
  `08da65dc4`/#435 re-pinned `@mermaid-js/layout-elk` to `^0.1.9`), `gh pr update-branch` merged dev in → CI green
  → auto-merge squashed `3c1cec57d`. `metered=$0.00`]**) →
  [implemented/v0_30_0/m-ailang-fmt-inline-interior.md](implemented/v0_30_0/m-ailang-fmt-inline-interior.md).
  Log entries 72 (design/park) + 75 (land). ·
  **m-fmt-properties-printer-roundtrip** (**[LANDED 2026-07-20 (iter 69) — PR #424 squash `942931816`;
  UNPARKED by Mark #422 "Continue the AILANG fmt sprint and go to sprint planner"; planner (opus) →
  executor (opus, worktree) M1–M2 → evaluator (sonnet, generator≠judge) PASS 98/100 r1; contract-clause
  printer round-trip fix + silent-contract-deletion data-loss fix (`parser_func.go` `=`→append) + 2
  adjacent Phase-1 printer bugs the full-corpus sweep exposed (precedence-driven `;`-separation;
  `@verify(depth:)` key re-synthesis) → `preexisting-Phase1-rt-bug` gate 28→0, hardened; 30 contract
  examples reformatted; doc → `implemented/v0_30_0/`. Was: PARKED needs-human-review iter 66 (quorum R2
  data-refuted). `metered=$0.00`]** →
  [implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md](implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md),
  see its ⛔ Quorum Record. **Controller repo-wide re-check DATA-REFUTES the residual objection**: the
  only `ast.FuncDecl.Properties` consumers repo-wide are exactly the V17 sites (`internal/elaborate`
  + `internal/testing`); the `cmd/ailang/test.go` `.Properties` hits are a distinct `[]PropertyResult`
  results field, not the AST field; no accessor/interface/visitor indirection. Scope-corrected doc:
  the real defect is `requires`/`ensures` contract clauses (NOT `properties[...]` blocks) failing the
  Phase-1 printer round-trip (exit 2) on 30 corpus files, PLUS a latent silent-contract-deletion
  data-loss bug (`parser_func.go:169` `=`→append). **Human fork on #399:** (1) authorize routing to
  sprint-planner [RECOMMENDED — sole objection data-refuted, not fmt-phase2's deepening gaps],
  (2) authorize one more bounded round to fold the repo-wide audit into the Verification Log,
  (3) keep parked. ~1d impl, LOW risk/conflict, metered $0.1347 quorum (iter-66). Log entry 71. ·
  ~~M-TOOLING-DETERMINISTIC (normalize/suggest-imports/apply, 3–4d)~~ **[REALITY-CHECKED iter 48
  (2026-07-18) → PREMISE SUPERSEDED; scope-close PARKED for Mark.** The CLI trio doesn't exist, but
  its premise (single-shot fragment + LLM repair) is obsolete — `prompts/repair_prompts/` deleted,
  eval flow is agentic w/ per-edit `ailang check` feedback — and its core capability already ships as
  `normalizeProgram` (`internal/eval_harness/normalize.go`, deterministic module-wrap + std/io inject).
  Per-goal: G1 normalize=SHIPPED; G2 suggest-imports=PARTIAL/ABSORBED (std/io only; general
  symbol→import now met by implicit prelude + agentic feedback + `ailang docs`); G3 apply=obsolete
  (agents edit files directly). Regression guard `TestNormalizeProgram_MToolingMotivatingFragment`
  landed. Doc header → REALITY-CHECKED w/ per-goal table. **Mark scoped DX tooling "both in" → not
  ruled out unilaterally; awaiting his SUPERSEDED-close vs. much-smaller "expose `ailang normalize`"
  decision on #399.** Recommend prefer m-ailang-fmt for any DX budget. Log entry 53.]**
- **Prompt-diet** (GATED — unblocks once the diagnostics above land + the curve authorizes):
  m-eval-slim-prompt-self-discovery (R3.1 pass-rate-per-token curve, 2d) → prompt-deletion pass R1.2

### Clause 4 — orchestration flagship (Mark: full surface in)
- **Effect sprints** (decomposed): **m-effect-replay-contracts (2/4, 3d) [LANDED PARTIAL iter-99 →
  implemented/v0_30_0 — Mark option-(b) unparked it; controller fold → planner opus → executor opus/worktree
  → evaluator sonnet PASS 86/100 r1. Registry + os/seeded/crypto dispatch machinery + trace + golden
  gate all GREEN. BUT seeded/crypto not `.ail`-reachable → new gating item below]** ·
  **m-effect-replay-subsumption [M0+M1 LANDED iter-108 `b8249ef35`; M2 LANDED iter-110 `63b0ba3dd` — validate-path relaxation, evaluator sonnet PASS 88/100 r1 zero blocking, asymmetry proven by a two-binary matrix (`blocker`/`c2`/`c6` flip 1→0 while `c3`/`c4`/`c7`/`c8` stay rejected); **M3 LANDED iter-111 `12e5df162` → SPRINT COMPLETE**, doc moved to `implemented/v0_30_0/`. End-to-end acceptance: `main` held os-only per F3 (`verify_examples.sh` hardcodes `--entry main`), seeded/crypto behind separate entrypoints, and BOTH test controls **mutation-proven** — a constant `seeded_roll` breaks the different-seed assertion (line 67), flipping the `Rand/os` taxonomy entry to `Opaque` breaks the os-trace assertion (line 77); both reverted SHA-256-identical. Evaluator sonnet **PASS 84/100 r1, zero blocking**; its NB-1 (the guide taught `deterministic_roll`, a name absent from the very example file the guide tells you to run) and NB-2 (os-mode never exercised at runtime) were both reproduced first-party and FIXED before landing. `ai_modes.ail` red confirmed pre-existing against a pristine-base capture taken before handoff, and deliberately NOT greened with an AI edge (Q1)]**
  → [implemented/v0_30_0/m-effect-replay-subsumption.md](implemented/v0_30_0/m-effect-replay-subsumption.md),
  doc `011da3c81`, plan [m-effect-replay-subsumption-sprint-plan.md](planned/v1_0_0/m-effect-replay-subsumption-sprint-plan.md)
  `23a7a8210`; P0, ~2.5–3d, **4** milestones (the planner added M0 — `effects.go` is 720/800 against a
  hard `check-file-sizes` gate). **M0+M1 landed iter-108**: planner opus → executor `codex:gpt-5.6-sol`
  → evaluator sonnet **PASS 81/100 r1, zero blocking**. M1 is deliberately a **STRICTNESS increase, not
  the relaxation** — declarations now carry full elaborated `*types.Row` through validation instead of
  `ast.EffectNames` name-only erasure, so `c3`/`c7`/`c8` flip 0→1 (controller-verified BEFORE/AFTER with
  two binaries on its own fixtures); `blocker`/`c2` stay 1, which is M2's job. **The real M1 deliverable
  turned out to be F4**, which the quorum logged non-blocking and the planner promoted to hard:
  `UnionEffectRows` prefers its LEFT param map on conflict, so once M1 stopped erasing modes, a body
  calling both a `seeded` and an `os` helper would collapse to one mode — and if `os` survived, an `os`
  declaration would wrongly ACCEPT the `seeded` callee, reopening the exact hole M1 closes, in a build
  where every fixture is green and the suite passes. Closed with a conflict-preserving union;
  **mutation-tested non-vacuous** (restoring the left-preferring bug compiles and turns all three F4
  tests RED). **M2 carries two evaluator findings**: a conflict rejection exits 1 but does not NAME the
  offending mode (it falls into "no specific missing effects identified" — the doc's defect (2) in a new
  costume), and the conflict set can render as `! {Rand[mode=os|seeded]}` in a Suggested-fix line, which
  is not valid AILANG. Both are M2's structured-diagnostics deliverable. **DECIDED by Mark 2026-07-27 (attended, `4d32c71bb`):
  YES** — an explicitly declared mode SUBSUMES the bare/os effect requirement; implement as the narrow
  relaxation on the `SubsumeEffectRows` validate path ONLY (function-value mode distinctness unchanged).
  Unparks effect sprints 2–4. (Was: NEW GATING clause-4, PARKED needs-human-review iter-99.)
  **SHARED gate for the whole effect-mode-dispatch line** — the parent doc's `Clock[mode=pinned] = now()`
  examples have the identical structure, so sprints 3/4 (clock/net/fs) hit it too.
  ⚠ **CORRECTION (this row's own prior text was WRONG, refuted by iter-107's live repro at a CLEAN build
  of `4d32c71bb`)**: the claim *"`SubsumeEffectRows` treats effect modes as INVARIANT"* — inherited from
  the parent doc — is **FALSE** as observed on the `.ail` path. Enforcement is **one-directional**: an
  os caller silently ACCEPTS a `seeded` or `crypto` callee (exit 0), while the reverse rejects. Cause is
  structural, not a rule: `validate_effects.go:109-114` threads `declaredEffects map[string][]string`
  built via `ast.EffectNames` (labels only), so `stringSliceToEffectRow` **cannot carry mode params** —
  the required row is **mode-blind** for every declared-function call. Two further defects found: the
  rejection prints **`Missing effects:` with an EMPTY payload** (label-set difference; a mode mismatch
  has identical labels), and naively relaxing the blocker without closing the mode-blindness would make
  seeded/crypto declarations **vacuous**. The doc scopes full-row propagation in and flips the
  wrongly-accepted direction to rejected — measured in-repo breakage: **ZERO** `.ail` files declare a
  non-default mode outside comments. **Quorum**: designer `codex:gpt-5.6-sol`; reviewers `gpt5-6-sol` +
  `gemini-3-1-pro` + controller opus; 3 rounds — R1 narrowed the rule from "any effect with a registered
  default" to explicit **schema-registered edges, Rand-only**; R2's attribution objection was answered by
  rebuilding clean at the exact SHA and re-running all 9 fixtures (**identical**), then the
  narrow-refinement carve-out. **NEXT**: route to sprint-planner. ·
  m-effect-clock-net-fs-modes
  (3/4, 3d — BLOCKED behind sprint-2 AND the subsumption decision) · m-effect-scope-params (4/4, 2.5d — **RE-SCORED to v1.1, D-27, Mark attended 2026-08-22; leaves the v1.0 bar**)
- **Flagship + surface**: m-v1-orchestration-flagship (verified AI-pipeline example + orchestration
  benchmarks into rotation + README/site lead, 2–3d; m-contracts-as-code-vertical folds in as the
  worked example) · m-serve-api-live-tool-registry (hot MCP tool registry, 3–4d) ·
  **m-serveapi-raw-handler-mcp** (**[LANDED 2026-07-22 (iter 78) → implemented/v0_30_0; M1 `@nomcp`
  shipped, M2 DROPPED → doc COMPLETE. Planner opus → executor opus (`2d6596292`) → evaluator sonnet
  (generator≠judge) PASS 96/100 r1, no defects; PR #452 squash `ee04f13d0`, dev CI green per-workflow
  (19 checks). Closes the live docparse `getKeyUsage`/`requestHistory` MCP capability leak with a
  one-line annotation; diff confined to parser allowlist + `internal/apiserver/` — NO eval-core change.
  Was: DECIDED by Mark 2026-07-20 → ROUTABLE; M2 fake-envelope DROPPED, no re-quorum. Historical park:
  PARKED iter 57 — QUORUM-AT-PICK BLOCKED ×2]** →
  [implemented/v0_30_0/m-serveapi-raw-handler-mcp.md](implemented/v0_30_0/m-serveapi-raw-handler-mcp.md), see its ⛔ Quorum Reblock section;
  M1 `@nomcp` MCP-exclusion annotation — keep a `@route` on HTTP but off the `--mcp-http` tool surface
  (`@noexpose` can't: it also kills HTTP + is overridden by `@route`) — closes the live docparse
  `getKeyUsage`/`requestHistory` MCP leak; **M1 is CLEAN + unobjected in both rounds → independently shippable**.
  M2 `@raw`-over-MCP twice-rejected: R1 default-on = authority-widening + silent header-fabrication; R2 the
  `@mcp`-opt-in + typed-sentinel fix itself violates the frozen core (`headers`/`query` are `Json` → a non-`Json`
  sentinel type-panics at binding; a `Json` sentinel needs core `std/json` changes). **Human fork on #399:**
  (1) split+ship M1 now [RECOMMENDED], (2) pick an M2 arch — valid-`Json` provenance marker
  `{"_transport":"MCP_UNAVAILABLE"}` + require `req.method=="MCP"` branch, OR drop the fake-envelope entirely,
  (3) keep parked. Unblocks docparse quota-hardening item 5. Log entry 62. ~0.5d for M1 alone) ·
  m-agent-step-cancellation (1.5d) ·
  **m-ai-reasoning-effort** (**[LANDED 2026-07-22 (iteration 83) → implemented/v0_31_0; full inner
  loop headless — planner opus (7-milestone plan, caught OpenRouter-already-has-Effort premise drift
  → M5 replace-not-extend) → executor opus (M0–M6, 7 commits `08e2aa935`→`3e784748c`) → evaluator
  sonnet (generator≠judge) PASS 96/100 round 1, ZERO defects. Typed `ai.Request.ReasoningEffort` +
  ONE shared fail-loud resolver across all 12 Generate/Step/streaming constructors of 4 clients;
  5 typed sentinels reuse `AIError`/`CodeSchemaValidation` (via `Unwrap()`); capability table ships
  EMPTY → unknown model + explicit control = typed reject (no silent fallback; OpenAI `"off"` never
  → `"minimal"`); Anthropic hook precedes `MaxTokens=4096` default; Gemini/Anthropic `B=0` exemption;
  OpenRouter Effort-wins branch REPLACED. AC #14 byte-identical goldens per provider; 16/17 ACs
  network-free (4 NEEDS-LIVE-SMOKE cap-entries + AC17 notify = parked M7 metered follow-up).
  Minimal-Frozen-Core (`internal/ai/**` only). PR #460. Mark fork (a) authorized; M0 code-audit
  executed gpt5-6-sol's inventory ask; no re-quorum. `metered=$0.00`.]**
  **[was: Mark 2026-07-22 fork (a) → ROUTE TO SPRINT-PLANNER: both authorized
  fixes GREEN (gemini PASS); Sol's machinery-inventory ask folds into the planner's mandatory M0
  code-audit. Doc front-matter stamped. Historical: PARKED — Mark's authorized bounded round EXECUTED
  iter 81 (2026-07-22): BOTH named R2 objections RESOLVED (reasoning_max_tokens 4th resolver input;
  Gemini B=0 MaxTokens exemption); re-quorum gemini→PASS but gpt5-6-sol→REJECT on a NEW out-of-scope
  "inventory existing AI-package machinery" objection → bounded round consumed → RE-PARKED. Rev-2 doc
  merged PR #457 → `893873c81` (docs-only, CI green). Human fork: (a) route to sprint-planner folding
  gpt5-6-sol's inventory ask into planner M0 code-audit [RECOMMENDED]; (b) one more bounded revision;
  (c) keep parked. Designer=claude-fable-5 ($0), re-quorum metered=$0.062. Log entry 86.]**
  **[was: AUTHORIZED by Mark 2026-07-20 → ROUTABLE: ONE more bounded
  revision+re-quorum round, scoped to the 2 named R2 objections only — doc front-matter stamped.
  Historical park:]** **[was: PARKED iter 61 — QUORUM-AT-PICK: R1 BLOCKED
  (no-silent-fallback + missing MaxTokens conflict surface) → codex-designer Rev-1 resolved both
  (fail-loud contract w/ 5 typed errors + capability gating + full Conflict Surface) → R2
  re-quorum BLOCKED on 2 NEW *narrower converging* fixes]** →
  [planned/v0_29_0/m-ai-reasoning-effort.md](planned/v0_29_0/m-ai-reasoning-effort.md), see its
  ⛔ Quorum Record. R2 objections: (1) resolver omits OpenRouter's `reasoning_max_tokens` 4th input;
  (2) Gemini rule over-reaches forcing `MaxTokens` for `B=0` "off" (breaks docparse consumer).
  Both small/concrete — NOT fmt-phase2's deepening gaps. **Human fork on #399:** (1) authorize
  ONE more bounded round [RECOMMENDED — close to green], (2) amend scope (drop `reasoning_max_tokens`
  from the typed resolver), (3) keep parked. ~14h impl (doc est), metered $0.23 (iter 61). Log entry 66.
  **REALITY-CHECK (iter 66, log entry 71): STILL PARKED — the feature did NOT land out-of-loop.** The
  iter-65 "Next" flagged commit `5afa9a1e1` ("feat(eval): reasoning_effort knob") as a possible
  out-of-loop landing. REFUTED: that commit is an EVAL-HARNESS-only OpenRouter `reasoning.effort` knob
  (`models.yml` + `internal/eval_harness/` + `openrouter/chat.go`), NOT this doc's typed
  `ai.Request.ReasoningEffort` field. Verified absent on origin/dev: `git show origin/dev:internal/ai/provider.go`
  has no `ReasoningEffort`, and the 5 sentinel errors (`ErrUnsupportedReasoningEffort`, …) do not exist.
  The v0.31.0 cross-provider feature is unbuilt; the R2 fork above still awaits Mark.)

### Clause 5 — cost credibility

- **[LANDED 2026-08-22 (iter-253) — M4b FIRED; CLAUSE 5's LAST MILE IS CLOSED]** M4b baseline
  cohort run, on `D-26`. **`cost_per_verified_success_usd = $0.7778187072`** for baseline `v1.0`
  (agent mode · ailang · 6 contract benchmarks × the 5-model `agent_suite` = **30 of 30** runs
  banked; cohort hash `526fe7240112bb16238f91a077487dc9fbf0be5c0fbca723c2a21e6a52bd0f40`; chain
  `219b1fbc`). Strict command `ailang chains stats --cost-per-verified-success --baseline v1.0
  --json --strict` **rc=0, `available:true`** (AC2). **AC3 satisfied by re-derivation**: summing
  the 30 archived run JSONs at `Decimal` precision gives `2.333456121600000036` vs the KPI's
  `known_cost_usd 2.3334561216` (delta **3.6E-17**), and an independently re-implemented
  `isVerifiedSuccess` predicate returns **3**, matching. Reproducibility artifacts on a TRACKED
  path (`eval_results/` is git-ignored — see the defect row below):
  [m-cost-per-success-kpi-baseline-v1.0-cohort-manifest.json](implemented/v1_0_0/m-cost-per-success-kpi-baseline-v1.0-cohort-manifest.json)
  + [m-cost-per-success-kpi-baseline-v1.0-kpi.json](implemented/v1_0_0/m-cost-per-success-kpi-baseline-v1.0-kpi.json).
  **Provenance labelled per the ratified 2026-07-27 decision**: **$0.4586 real OpenRouter metered**
  vs **$1.8749 list-price-equivalent** subscription reporting — actual spend **$0.46 of the $20 cap**.
  **The denominator is non-zero for the first time**: `verify_verified` distribution `{0:16, 1:2,
  2:11, 4:1}`, against a repo-wide historical total of **zero** across 19,027 banked files. **The
  finding is the shape of the 26 non-verified passes, not the headline number** — see the row below.

- **[NEXT — DESIGN DOC LANDED 2026-08-23 (iter-259), quorum &times;2 + a restored absent reviewer, narrow-refinement
  carve-out applied; ROUTABLE TO SPRINT-PLANNER; AILANG fix; ~1–2d, P0 for the clause-5 KPI]**
  Doc: [planned/m-verify-bounded-unrolling-false-counterexample.md](planned/m-verify-bounded-unrolling-false-counterexample.md)
  (706 lines, 28 verification rows, 13 acceptance criteria). Design: a new first-class `inconclusive` status;
  classifier is the structural predicate `smt.IsRecursiveFunc && recursiveDepth > 0 && sat` — which the two
  ladders **already compute** for `BoundedDepth`, so `internal/smt` needs **zero** code changes; the two
  duplicated ladders (`ai_check.go:370-420`, `verify.go:388-435`) collapse into one `applySolveOutcome` helper
  with a drift test. **Blast radius measured at iter-259: 8 of 30 frozen-cohort runs, all on
  `contract_sorted_merge` (&times;4) and `contract_bst_validate` (&times;4), uniform across 4 model families.**
  **RESIDUAL — `D-30`, and this is the SECOND doc blocked by it independently:** `gpt5-6-sol` (restored after a
  budget drop-out) sustained that reader-before-writer *commit* order cannot prevent new-binary/old-harness skew,
  because `RunAICheck` resolves its verifier child via PATH — an old harness sees `counterexample == 0`, sets
  `VerifyOk` true, and a mixed `verified > 0` result inflates. AC-13 (the reviewer's verbatim ask) DETECTS this;
  only a `D-30` ruling PREVENTS it. **`D-32` filed** on whether `inconclusive` joins the effective KPI arm.
  **The KPI number is bit-identical with or without this fix** under the doc's own predicate — its value is
  honest labels, `repair.go:76-86` no longer coaching models to break correct code, and `D-29`'s second clause
  unblocked. Original filing evidence follows.
  `m-verify-bounded-unrolling-false-counterexample` — **the verifier reports its own incompleteness as a
  `counterexample`, so a CORRECT recursive implementation is graded a verification FAILURE.** Found while
  scoping `D-29`'s second clause. Bounded unrolling havocs the recursive frontier; Z3 then instantiates that
  unconstrained frontier and the result is emitted as a violation with a `model`, indistinguishable from a real
  counterexample. **Measured at `9417c5ff7` with a same-file control, `ailang ai-check` on a stamped scratch
  build (`v0.33.1-222-g9417c5ff7`, ldflags per the binary-provenance rule):** two functions, byte-identical
  `requires`/`ensures` (`length(result) <= 2 * length(s)`), differing ONLY in recursion —
  `encFlat` (non-recursive) **verified**, `encRec` (recursive) **counterexample**. The postcondition genuinely
  holds: `ailang run --caps IO` prints `true` on all four inputs incl. a 6-char string.
  **It is STRUCTURAL, not a depth-tuning matter — the counterexample input GROWS with the unrolling depth:**
  `-verify-recursive-depth 2` → `"AAA"` (3 chars, `bounded_depth 2`); `4` → `"ABDCA"` (5); `8` →
  `"ABCDEFGHAB"` (10); the control `encFlat` verifies at every depth. For any finite `k`, Z3 picks an input
  longer than `k`. A second shape, `ensures { result >= 0 }` on a recursive ADT length, is counterexampled at
  depths 2/4/8 alike with a 4-element list as the "witness" — **and `benchmarks/contract_sorted_merge.yml:23-24`
  ships exactly that clause on `sLength` today**, so this is not hypothetical for the frozen cohort.
  **Why it outranks benchmark authorship**: `isVerifiedSuccess` fails on `VerifyCounterex > 0`, so this
  converts correct model output into a *verification failure* in BOTH the strict and the effective arm `D-29`
  just ratified — it corrupts the headline in a direction no exemption can repair, and it is the same
  data-integrity class as the `skipped`/`not_applicable` conflation (CLAUDE.md §2 — no silent fallbacks).
  **Fix direction**: an unprovable-under-unrolling obligation must be reported as `skipped`/`unknown` with a
  structured reason (the `UNENCODABLE_TYPE` machinery already exists — a sibling probe returned
  `skipped` + `"calls user function \"sLen\" that is not SMT-encodable in this context"`), never as a
  `counterexample`. Distinguishing a frontier-havoc model from a genuine one is the design question.
  Depends on nothing; blocks the useful half of `m-benchmark-ensures-coverage`.

- **[NEXT — filed 2026-08-23 (iter-258) from Mark's `D-29` ruling; BLOCKED on `m-verify-bounded-unrolling-false-counterexample` for 4 of its 5 candidates; ~0.5d for the safe subset]**
  `m-benchmark-ensures-coverage` — **Mark's second `D-29` clause: "update prompts to use `ensures` for
  benchmarks that make sense."** Iteration 258 enumerated the whole surface first-party rather than guessing at
  "makes sense". **7 of 92 benchmark YAMLs carry a `contract_spec`** (`contract_bst_validate`, `contract_leap_year`,
  `contract_matrix_determinant`, `contract_rle_roundtrip`, `contract_roman_numeral`, `contract_sorted_merge`,
  `prompt_injection`). Their declared functions partition **exactly three ways**: **16** carry an `ensures`;
  **10** carry NO clauses at all (`insert`, `contains`, `det2`, `det3`, `fromRoman`, `sInsert`, `sMerge`,
  `sIntersect`, `isSorted`, `toStr`) and so raise no obligation; and **5** carry `requires` with **no** `ensures`
  — `isBST`, `minor3`, `encode`, `decode`, `toRoman` — which is **precisely** `D-29`'s named five, so the
  "no ensures clause" skip class has a clean syntactic definition.
  **Of those five, only `minor3` is non-recursive and therefore safely addable today.** The other four
  (`isBST` over `Tree`, `encode`/`decode` over strings, `toRoman` greedy-recursive) would, per the row above,
  turn a `not_applicable` into a **spurious `counterexample`** — moving those runs from "nothing to verify" to
  "verification FAILED" and lowering both KPI arms while measuring nothing about the model.
  **`encode` is the strongest candidate on merit and the clearest casualty**: `contract_rle_roundtrip.yml`
  states the invariant in its own prose — *"encoded string is never longer than 2 \* length(original)"* — and
  ships no `ensures` for it; that exact clause is what iteration 258 measured as verified-when-flat,
  counterexampled-when-recursive. **Encodability of the clause LANGUAGE is not the blocker** (probe: three
  string-length postconditions all `verified`, control `controlDouble` verified, `skipped 0`); the blocker is
  the recursive body. **Scope when unblocked**: add `ensures` to `minor3` now if a meaningful postcondition
  survives review, and to the remaining four only after the unrolling defect is fixed; re-run the frozen
  cohort, since this changes what the benchmarks measure (`D-29` option (b) says so explicitly).

- **[READY — D-30 was ANSWERED (b) same-binary 2026-08-26 (see ledger); tag un-parked 2026-08-31 attended after five days of no iteration consuming the answer — DESIGN DOC written and revised, TWO full quorum rounds; ~1–2d, P1]**
  `m-contract-verification-coverage` — **iteration 253 filed this as "the encoder's coverage is the
  cheapest lever". Measured against the banked `verify.results[].status` rows, that is WRONG BY 8:1.**
  The 53 skips partition cleanly, exhaustively, with zero residue:
  **24 `no ensures clause (nothing to verify)`** · **20 declaration-closure (callee not
  SMT-encodable)** · **5 callee signature uses an unencodable TYPE** · **4 unencodable BUILTIN**.
  All 29 non-`no-ensures` skips carry rejection code `UNENCODABLE_TYPE` — one coherent class.
  **The 24-skip plurality is NOT an encoder limitation. It is the benchmark specs' own design, and
  the models complied exactly.** The five functions carrying it — `isBST`, `encode`, `decode`,
  `toRoman`, `minor3` — are declared in the benchmarks' own `contract_spec` with `requires` and
  **no `ensures`** (**CORRECTED at iteration 258**: this row previously said `minor3` carries no clauses at all — it carries `requires { row >= 0, row <= 2, col >= 0, col <= 2 }` at `benchmarks/contract_matrix_determinant.yml:24`, unchanged since `fe92c172f` 2026-04-21, so the class is uniformly *`requires` and no `ensures`*, all five); the contract lives in the separate *proof*
  functions (`roundtrip`, `insertPreservesBST`, …), which DO carry `ensures { result == true }`.
  Control fires: functions that verified (`roundtrip`, `isLeapYear`) carry explicit `ensures` in the
  same files, so the instrument distinguishes. Each of the five is skipped in **all 5 models** —
  uniformity that is a property of the spec, not of any model.
  **So `isVerifiedSuccess`'s `VerifySkipped == 0` clause (`internal/observatory/cost_per_verified_success.go:95`)
  disqualifies runs for functions the benchmark deliberately left unconstrained** — because the flat
  `verify_skipped` counter aggregates "the encoder could not" and "there is nothing here to verify"
  into one number. **Priced on the frozen cohort** (numerator fixed at `known_cost_usd
  $2.3334561216`; the re-implemented predicate reproduces the published `$0.7778187071999999`
  **exactly**, so all three arms come from one instrument with one clause changed):
  | option | denominator | cost/verified success |
  |---|---|---|
  | **A — as published** (`skipped == 0`) | **3** | **$0.7778** |
  | **B — exempt "nothing to verify"** | **11** | **$0.2121** |
  | **C — exempt every skip** (upper bound) | **12** | **$0.1945** |
  **8 of the 9 recoverable runs come from the PREDICATE (A→B); exactly 1 from the ENCODER (B→C)** —
  that one being `contract_sorted_merge` / `gpt5-6-luna`, blocked on `sLength`. Encoder work is real
  and is *not* the cheap lever on this KPI.
  **ROUTED — AILANG fix, one systemic change (Principle 3), NOT a benchmark edit and NOT an encoder
  sprint**: split the verifier's `skipped` status into `skipped` (the encoder could not encode this)
  and `not_applicable` (no `ensures` clause — nothing to verify), propagate both as distinct
  counters, and have `isVerifiedSuccess` ignore `not_applicable`. That conflation is a data-integrity
  defect on its own merits regardless of the KPI ruling (CLAUDE.md §2): today the encoder-coverage
  number a reader needs — **29** — is hidden inside a **53** that is **45% not-about-the-encoder**.
  **`D-29` RESOLVED 2026-08-23 (iter-258) — Mark ruled BOTH (strict + effective published side by side), so this row is no longer D-29-blocked and gains a post-split milestone: publish the two arms rather than flip the predicate. `D-30` remains its sole blocker.** Original framing: whether the published headline moves $0.7778 → $0.2121 changes a
  Mark-ratified definition (2026-07-27, "Yep ok") and the number clause 5 rests on; the status split
  is correct either way, the predicate's *use* of it is Mark's call.
  Related: `#513` (per-call bounds do not bound a whole run).
  **ITERATION 255 — the unblocked half was designed, reviewed twice, and PARKED one objection short.**
  Doc: [planned/m-contract-verification-coverage.md](planned/m-contract-verification-coverage.md) (730 lines,
  Fable designer ×2 — a knowing Fable-diet overspend, FLAGGED). Scope was deliberately the half that is correct
  **regardless of `D-29`**: split the status/counter, and preserve today's KPI semantics exactly by making both
  predicates read `VerifySkipped + VerifyNotApplicable`. The `D-29` flip stays out of scope, with mutation `M-8`
  as a tripwire against it landing by accident.
  **Round 1 BLOCKED, 2/2 external reviewers present** (`absent_reviewers` empty — a full-strength reject, not a
  degraded pass), both independently naming the SAME defect: the milestone order was **writer-before-reader**, so
  between the emitter split and the harness field the `not_applicable` count is dropped outright, and between the
  harness and the predicate the observatory reads a shrunken `VerifySkipped` — i.e. it ships the banned `D-29`
  flip for those rows. I confirmed the premise first-party rather than forwarding it (rule 3f):
  `AICheckVerifyResult` (`internal/eval_harness/verify.go:36-42`) has **exactly five** fields and no
  `not_applicable`. Both verbatim fixes applied: order reversed to reader-before-writer (M1 the whole read path
  incl. both predicate sums while emitters are untouched, so every intermediate state is behaviour-identical **by
  construction** rather than by argument; M2 the only writer change), plus `gpt5-6-sol`'s end-to-end banking test
  as `AC-8`, which forced extracting the three previously-unkilled banking literals into unit-callable
  constructors.
  **Round 2: `gemini-3-1-pro` PASS, `gpt5-6-sol` REJECT** — on the one residual the doc had named honestly and
  mitigated only by convention, and which I had explicitly asked the reviewers to attack. That objection is now
  **`D-30`**, with its premise measured rather than inherited (2/2 live callers resolve the verifier via PATH; the
  parent/child skew is **live on this rig today**). Parked rather than force-passed: the narrow-refinement
  carve-out requires a fix needing no controller judgment, and this one is a choice between a new wire protocol, a
  repo-wide resolution change, and accepting a P0 residual (standing rule 2).
  **Nothing was implemented** — no planner, executor or evaluator ran, and no code shipped.


- **[NEXT — RE-SCOPED TO DECOMPOSITION 2026-08-23 (iter-257); filed 2026-08-22 (iter-253); DESIGN DOC written + revised ×3, FIVE quorum rounds, every objection real; at round 5 `gemini-3-1-pro` PASSES and the surviving rejection is confined to Consumer 2; ~2–3d for the split remainder, P0]**
  `m-cohort-manifest-build-provenance` — **the cohort manifest cannot identify its own build.** M4b's
  manifest recorded `"ailang_version":"dev"` and `"git_commit":"dev"` because
  `internal/version/version.go:20` defaults `Version = "dev"` and only `make`'s `-ldflags` overrides
  it (`Makefile:42`) — so the **scratch-directory `go build`** that this loop's own binary-provenance
  rule mandates (precisely to avoid `make quick-install` clobbering the shared `~/go/bin`) leaves the
  manifest's two provenance fields blank. The artifact's stated purpose is that "a reviewer can
  independently recompute it"; without a build SHA they cannot. Iteration 253's build was
  `498a64d38`, recorded in the STATUS stamp because the artifact could not record it. **Fix
  direction**: either stamp from `debug.ReadBuildInfo()`/VCS at runtime when ldflags are absent, or
  make the manifest writer FAIL LOUDLY on an unstamped build rather than writing the literal `"dev"`
  (CLAUDE.md §2 — no silent fallbacks). **Second, related defect found the same way**:
  **`eval_results/` is git-ignored** (`.gitignore:91`), so M4b's own acceptance criterion "archive
  its full output" would have produced **no archive at all** under `git add -A` — silently, the same
  class as iteration 195's `.ailang/` finding. Caught by `git check-ignore` before committing.

  **ITERATION 256 — the root cause is WIDER than this row states, and the doc is one bounded
  revision from routable.** The row says only `make`'s ldflags can set `Version`. True, and it
  misses that `Commit` HAS a working `debug.ReadBuildInfo()` fallback which **a linked git worktree
  silently disables** — so the defect is not "we forgot ldflags", it is that *this loop's own
  isolation rules guarantee unidentifiable binaries*. Four arms, `go version -m`, dotted `vcs\.`
  pattern, control = total `build` settings: pin worktree (detached) **0** vcs / 10 settings; main
  checkout, a real `.git` DIRECTORY, **4** / 14 with `Commit: db71d2a…-dirty`; `.wt-iter216-record`,
  a linked worktree **on a branch**, **0** / 10 — so it is the worktree, not the detached HEAD;
  and `-buildvcs=true` in a worktree **rc=0, 0 vcs lines, no error**. **Three decision-bearing
  consumers** silently accept the `"dev"`: `eval_suite_manifest.go:212-213` (frozen cohort
  manifest — the live `v1.0` artifact reads `dev`/`dev` with `frozen_at`/`chain_id`/`cohort_hash`
  populated as controls), `pipeline_module.go:276` (module cache **compiler identity**, whose own
  comment says a new commit must invalidate), and `eval_suite.go:191` (`--bank-by-version` bucket
  — `releaseTag("dev")` is `"dev"`, re-creating cross-build pooling).
  Doc: [planned/m-cohort-manifest-build-provenance.md](planned/m-cohort-manifest-build-provenance.md)
  (872 lines, 4 milestones, 14 ACs, 24 verification rows; Fable designer ×2 — diet-COMPLIANT under
  iteration 255's DOC-unit amendment; the carve-out revision went to sonnet as mechanical
  propagation, not a third Fable run).
  **Three quorum rounds, all blocked, all three finding REAL defects — and rounds 2 and 3 blocked
  on defects the CONTROLLER introduced.** R1: gpt5-6-sol (V17's remediation premise never
  executed) + gemini (a `-dirty` commit aliases compiler bytes) + controller (the proposed
  full-executable sha256 costs **232/215/214 ms** per process on a 96 MB binary, control 29 ms,
  on a path where caching is on by default). R2: `gpt5-6-sol` **dropped out on `budget`** — the
  N−1 degrade hole — and was **restored** by a solo `design-review --max-cost-usd 0.30`
  ($0.09095) rather than deciding the round short; restored, it rejected the stat-derived id for
  claiming "never under-invalidates" while naming a case that does. Its own conditional
  ("if hashing cost remains unacceptable, route to cache bypass") was **measured, not chosen**:
  the cache's entire benefit is ≈12 ms (`asset_path.ail`) and **nil** on the heaviest example, so
  bypass won. R3 was a **confirming quorum the carve-out did not require**, and is the only reason
  the next defect surfaced: both reviewers independently found that `CommitClean()` checks
  `-dirty` on **`Commit`**, while the ldflags contract sets `Commit=$(git rev-parse HEAD)` — a
  plain SHA — and dirtiness rides on **`Version`**. Measured read-only in the dirty main checkout
  (doc V24): `git describe --tags --always --dirty` → `v0.33.1-218-gdb71d2a16-dirty`,
  `git rev-parse HEAD` → `db71d2a1638bf1…`; the ReadBuildInfo arm from the same tree puts `-dirty`
  on `Commit`. **The two stamping paths dirty different fields.**
  **RESUME — bounded, unanimous, in-loop. NOT `needs-human-review`** (nothing awaits Mark; filing
  it as a human park would manufacture a decision he does not have — standing rule 8) and not
  `PARKED-ON-LANE` (nothing unblocks on a clock). Next iteration: (1) widen the predicate to both
  fields per the reviewers' verbatim text —
  `CommitKnown() && !HasSuffix(Commit,"-dirty") && !HasSuffix(Version,"-dirty")`; (2) document the
  `Version=="dev"` + clean-`Commit` case; (3) add the test arms both reviewers asked for (plain-SHA
  `Commit` + `-dirty` `Version` MUST select bypass); (4) re-quorum ONCE. Full detail in the doc's
  "Quorum revision log — round 3".

  **ITERATION 257 — the resume was applied, and TWO MORE ROUNDS each found real defects in
  consumers the previous round never touched. The lane is now DECOMPOSITION, not a sixth
  revision.** Round 4 (designer revision, reviewers' verbatim widening applied) came back
  **BLOCKED 2/2, `absent_reviewers` empty**, on two PRE-EXISTING holes: `gpt5-6-sol` — the M4
  **freeze gate** refuses only *unstamped* builds, so a `-dirty` binary can freeze release
  evidence; `gemini-3-1-pro` — **AC-6**'s "cache dir stays empty" proves `Store` was bypassed but
  not `Lookup` (a fresh dir makes a wrong Lookup miss silently). Both premises measured
  first-party rather than forwarded (rule 3f) and both hold: M4's branches are knownness-only
  (`RefusesUnstampedVersion`/`_RefusesUnstampedCommit`, AC-11 asserting the substring
  `unstamped`) and a `-dirty` value **is** known; and the live release-evidence artifact carries
  exactly **20** top-level keys with **no** source diff and **no** compiler-content identity
  (`cohort_hash` is over cohort *composition* — preimage confirmed at
  `cmd/ailang/eval_suite_manifest.go:118-145`).
  **Round 5 was directed at the PATTERN rather than the two patches.** Four blocked rounds had
  each found one class — *a gate or assertion whose satisfying-state set is wider than the purpose
  it is cited for* — in a different consumer (R3 cache gate, R4a freeze gate, R4b an AC), so a
  systemic sweep (CLAUDE.md §3) of every gate and every AC against that one question was the
  directive. **It found three too-wide items no reviewer had named**: AC-9's unstamped arm
  ("prefix + non-empty suffix") is satisfied by a *constant*, i.e. by the shared-bucket defect
  itself; AC-13's "entry present" was prose, not an assertion (`make check-changelog` is rc=0 on
  pristine dev); and — consequentially — **the new strict freeze gate would have created a
  REFUSAL LOOP with the doc's own M4 remediation recipe**, which runs `git describe … --dirty`
  from a dirty tree and therefore re-fails the gate it exists to satisfy. AC-6 now pre-populates
  the cache with a dummy keyed by the ambiguous identity (proving Lookup bypass), pinned by
  mutation row **8c**, which names both that it kills the Lookup-hoisting mutant and that the old
  empty-dir assertion does not.
  **Round 5: `gemini-3-1-pro` PASS — the first pass in five rounds — `gpt5-6-sol` reject**, again
  2/2 present. Its surviving objection is the same class one level deeper and is real:
  `CommitClean()` is weaker than the cache-correctness purpose it gates, because **a clean commit
  identifies SOURCE state, not compiler bytes** — two clean builds at one commit can differ by Go
  toolchain, build tags or flags and still share a cache key. **Measured, and PRE-EXISTING at
  HEAD, not introduced here**: `ModuleCacheKey` (`internal/pipeline/cache_key.go:37`) hashes only
  `cacheKeyVersion`, the compiler-identity string, the source hash and dep digests;
  `runtime.Version()` appears **0** times in `internal/pipeline` against **4** repo-wide (control
  fires), zero build-tag/flag terms in `cache_key.go`, negative control **0**, scope asserted with
  `test -d`; exactly **one** live call site (`pipeline_module.go:276`) passing `version.Commit`.
  So M2 strictly NARROWS today's accepted path and leaves a pre-existing residual — filed
  separately as `m-module-cache-identity-not-compiler-bytes` below.
  **RESUME — SPLIT, do not revise again.** The doc bundles three consumers under one shared cause,
  and "what identifies compiler bytes" is a strictly harder question than "what identifies release
  evidence". Next iteration: (1) split **Consumer 2 (module-cache identity, M2)** out into its own
  doc, carrying `gpt5-6-sol`'s round-5 objection and its `proposed_fix` verbatim as that doc's
  opening problem statement; (2) leave M1/M3/M4 (identity API, banking bucket, freeze refusal) in
  this doc, which is one reviewer-clean and whose remaining objection is entirely Consumer-2
  scoped; (3) re-quorum the reduced doc ONCE. Not `needs-human-review` (nothing awaits Mark — the
  split is a controller routing call and the residual design question belongs to a designer plus a
  quorum) and not `PARKED-ON-LANE` (nothing unblocks on a clock).
- **[NEXT — filed 2026-08-23 (iter-257), first-party measured; ~1–2d, P1]**
  `m-module-cache-identity-not-compiler-bytes` — **the module cache keys on a git commit and calls
  it a compiler identity; a commit identifies SOURCE state, not bytes.** Surfaced by `gpt5-6-sol`
  at round 5 of `m-cohort-manifest-build-provenance` and **confirmed first-party at HEAD**, where
  it is PRE-EXISTING and independent of that doc's fate: `ModuleCacheKey`
  (`internal/pipeline/cache_key.go:37`) hashes exactly four things — `cacheKeyVersion` (`"v3"`,
  hand-bumped by a human noticing a format change), the `compilerVersion` string, the module
  source hash, and sorted dep digests. **Nothing about the toolchain reaches it**:
  `runtime.Version()` appears **0** times anywhere in `internal/pipeline` against **4** occurrences
  repo-wide (control fires, so the instrument can see a positive), `cache_key.go` carries **0**
  build-tag/`GOFLAGS`/`gcflags` terms, a fresh negative-control literal returns **0**, and the
  scope was asserted with `test -d`. There is exactly **one** live call site,
  `pipeline_module.go:276`, passing `version.Commit`. **So two clean builds at the same commit —
  different Go toolchain, different build tags, different build flags, or different generated
  inputs — share a module-cache key today and can serve each other's blobs.** The `cacheKeyVersion`
  bump is the only existing defence and it is manual, which is precisely the class CLAUDE.md §2
  forbids on a correctness surface. `gpt5-6-sol`'s `proposed_fix` names both directions: stamp a
  deterministic fingerprint over all compiler-relevant inputs (commit/tree state, toolchain, tags,
  flags) and key on that, bypassing Lookup **and** Store whenever it is absent; **or**, if a sound
  fingerprint cannot be delivered, bypass the module cache for all builds rather than claim the
  commit is a bytes identity. Note the second option is cheaper than it sounds — the cache's whole
  measured benefit is **0–12 ms** (`m-cohort-manifest-build-provenance` V23: `asset_path.ail`
  33/34/33 ms ON vs 46/46/44 OFF; `regex_log_orchestration.ail`, the heaviest at 6 imports, shows
  no measurable difference). Needs a design doc and a quorum; do NOT fold it back into
  `m-cohort-manifest-build-provenance`, whose five blocked rounds are the evidence that bundling
  this consumer with the release-evidence one is what prevented convergence.
- **[LANDED 2026-07-27 — M1–M3; M4 parked-for-Mark]** m-cost-per-success-kpi (dashboard KPI flip to cost-per-verified-success + v1.0 measured baseline) — **DESIGN DOC iter-103** ([implemented/v1_0_0/m-cost-per-success-kpi.md](implemented/v1_0_0/m-cost-per-success-kpi.md), quorum-cleared via carve-out). **Iter-104: M1–M3 shipped** via full sprint loop (planner/executor opus, evaluator sonnet **PASS 86/100 r1**, dev CI green @ `d869ec12d`): M1 `9bdc9319c` observatory strict `cost_per_verified_success` rollup (single `isVerifiedSuccess` predicate reusing `ClassifyStageCost`/`TotalKnownCost()`; `verify_*` propagated into both `EvalAssessment` constructors; cohort filter) · M2 `2a2a40f31` CLI `--cost-per-verified-success --baseline --strict` + HTTP + additive `latest.json headlineKpis` (one struct, field-for-field) · M3 `2d76b2cc3` headline `ValueDashboard` card + Fallback/stale badge + available/zero-denom/incomplete/absent states. Doc stays in `planned/` until M4b lands. **Iter-105: Mark RATIFIED all three M4 inputs** (#484 comment `2026-07-27T07:53:53Z` — verified-success definition "Yep ok"; cohort "assume current cohort but this may have light changes depending on release date"; headline placement "Fine") → M4 unparked and **split M4a/M4b**. **M4a LANDED** (planner opus → executor opus [codex probe FAILED, fallback FLAGGED] → evaluator sonnet **PASS 96/100 r1, zero blocking**): M4a-0 `37c070dd9` file-size extraction · M4a-1 `612cb78af` `--baseline <id>` cohort-freeze flag + BF-2 SQL-`LIKE` `_`-wildcard escape via ONE validator shared write+read · M4a-2 `fa4c1d095` data-driven `cohort_manifest.json` + `cohort_hash` (models resolve from `models.yml`, zero model names in Go → re-freezable per Mark's caveat) · M4a-3 `522ad61f1` **closes BF-1** · M4a-4 `6b252b9b1` docs/changelog/doc-split. **BF-1 was the iteration's real find**: agent-mode contract verification was NEVER wired on the live path — the only agent `RunAICheck` sat in `RunAgentBenchmark`, a function whose sole repo reference is a comment saying it must not be used, while `RunAgentBenchmarkWithExecutor` had zero Verify. So `verify_verified` was always 0 → `isVerifiedSuccess` always false → **the M1–M3 headline KPI structurally could never produce a number**; a cohort run before this fix would have burned metered dollars banking a guaranteed zero denominator. **M4b DECIDED by Mark 2026-07-27 (attended; was PARKED-for-Mark)**: (i) metered spend APPROVED,
  cap **$20 total** for the cohort run (raise `MISSION_METERED_BUDGET_USD` to 20 for that single
  iteration if needed) — BUT the cohort run **WAITS for the Anthropic key-quota reset (~2026-08-01)**
  per Mark's same-day call ("we can wait a few days for an actual eval run") — do NOT fire it
  before; (ii) ACCEPTED — publish the explicitly-labelled *list-price-equivalent* KPI with a
  distinct provenance status, per the iter-105 recommendation. Historical ask: (i) approval to spend metered dollars on the real cohort run (OpenRouter lanes are the real-dollar exposure; `$5` iteration ceiling); (ii) **cost-provenance decision** — `ClassifyStageCost` rule 1 treats any `cost>0` as authoritative "reported", but the *subscription* claude CLI reports non-zero `total_cost_usd` while nothing is billed (live-probed with both Anthropic keys stripped: 10 in/46 out → `$0.0108355`), so a claude+OpenRouter cohort blends list-price-equivalent and truly-metered dollars under one label, against the doc's "attributable **metered** dollars" goal.

### Clause 2 — soundness (near-done; no new holes found in triage)

- **[NEXT — world-DEMAND, filed 2026-08-22 (iter-252); repo-wide sweep, ~0.5–1d]** `m-run-selector-enumeration-floor` — **`go test -run <selector>` exits 0 when the selector matches NOTHING**, so every acceptance criterion shaped *"`go test -run 'TestA|TestB'` passes"* is **green before either test is written**, and stays green if a later rename orphans the selector. Reported by `mission-world` iter-110 from its own two vacuous ACs; **live-REPRODUCED at V1's HEAD** (ghost discipline — not a ghost): `go test ./internal/eval -run 'TestZzNoSuchTestIter252' -count=1 -v` → **rc=0**, **0** `=== RUN` lines, `ok ... [no tests to run]`; control `-run '^TestTaggedValue$'` → **rc=0**, **5** `=== RUN` lines — **identical exit codes**, which is what makes it undetectable from the code alone. This is the mission's own **vacuous-pass class** (after the silent `z3` skip, CI `t.Skip`, and `#517`'s zero-case properties): a check reporting success for work it never performed. **Exposure measured, not estimated**: **54** such invocations across **16** files in `design_docs/planned` (same-scope control **229** `go test` occurrences; negative control **0**; scope asserted with `test -d`), plus **104** in `design_docs/implemented`. **The fix is an enumeration floor**, not sharper prose: assert the count of top-level `=== RUN   Test` lines **equals** the number of names in the selector, and make a zero count FAIL LOUDLY as an instrument failure rather than pass. `m-list-accessor-api`'s sprint plan (iter-252) already ships this shape on all **7** of its `-run` criteria plus its shared helper, so the pattern to copy exists and is committed. **Scope note before routing**: the 54 are not all live gates — some sit in Verification-Log rows *describing* a measurement rather than gating one (iter-252 hit exactly that distinction and had to separate 11 raw matches from 7 real ACs), so the sweep must classify before it edits, or it will 'fix' prose. **Demand evidence satisfied by construction** — a real downstream consumer hit it first-party. Does NOT outrank the queue (cross-mission rule); positioned by normal ordering.
- **[LANDED 2026-07-28 — iteration 116] m-z3-hard-timeout** (`#510`; PR **#514** → squash `9253ec8a8`, dev CI green on all 14 checks incl. `test-windows`; plan → [implemented/v0_31_0/m-z3-hard-timeout-sprint-plan.md](implemented/v0_31_0/m-z3-hard-timeout-sprint-plan.md); evaluator **sonnet PASS 90/100 r1, zero blocking**) — **Mark's option (B) pick**, the precondition for `m-z3-adt-record-sort`. Both Z3 `exec.Command` sites in `internal/smt` are now bounded: `Solve` (was `solver.go:147-148`) at `max(config.Timeout, effective -T: secs) + 2s`, and — **systemic twin the issue never named** — `Z3Version()` (was `solver.go:271`), which is NOT cold since `cmd/ailang/verify_print.go:23` calls it on every human-mode `verify` header. `grep "exec.Command(" internal/smt/` now returns **zero** non-Context sites. Process-group kill (`Setpgid`, SIGKILL to `-pid`, ESRCH tolerated) + `cmd.WaitDelay`; the deadline is classified **BEFORE** output parsing so a truncated prefix from a killed process can never be read as a verification result. Caller-visible shape preserved (`StatusUnknown` + `"solver timeout"`), so `verify.go:427`/`ai_check.go:370` see nothing new. **Non-vacuity proved by the controller in THREE directions**, each restored byte-identical (sha256 `b9e65c78…`): pre-fix `Solve` → FAIL 30.36s · **the NAIVE fix** (deadline kept, group-kill removed) → passes timing AND status, fails only on `child process 88072 still exists after 2s` · pre-fix `Z3Version` → FAIL 30.35s. The middle mutation is the valuable one — it proves the orphan assertion catches the half of the bug a plain `CommandContext` would miss. Tests use a fake solver ignoring `-T:`, need **no real z3**, run in CI, **zero `t.Skip`**. Closes a Standing-rule-6 violation on the verification surface for every `ai-check` caller incl. sibling Ailang World. **Not fixed, filed as `#513`**: per-call bounds do not bound a whole run of N functions.
- **[LANDED 2026-07-29 — iteration 117; PR #516 → squash `5998f4039`, dev CI green; docs → [implemented/v0_31_0/m-z3-adt-record-sort.md](implemented/v0_31_0/m-z3-adt-record-sort.md) + its sprint plan; evaluator sonnet r1 **FAIL 59/100** → r2 **PASS 83/100**] [world-DEMAND] m-z3-adt-record-sort** — **the defect was TWO layers**, not the one the doc localised: the encoder's Step-0 alias fixpoint dropped the record declaration silently (`if !progress { break }`) AND `filterADTTypesForFunction` never walked record-alias fields, so the ADT's variant list never reached `EncodeFunction` — an encoder-only fix could not have worked. Both drivers now share one `filterSMTInputsForFunction` (`ai-check` had NO demand filter, so the KPI was **deflated**, not inflated). `validateDeclarations` now covers `declare-const`, typed `define-const`, recursive `(Seq X)` and plural groups **atomically** — closing a pre-existing hole where `HasPrefix(decl, "(declare-datatype ")` never matched `(declare-datatypes (`, so every mutually-recursive group bypassed the guard. `ai-check` exits 1 on `verify.errors > 0` (**breaking** for out-of-repo shell callers; JSON still emitted first). Declaration order is now deterministic (**A1**) — measured 40 runs → 3 orders before, 40/40 identical after. Corpus verified **76→81**, skipped **10→7** (a coverage fix, NOT a model improvement — flagged in CHANGELOG). ⚠ **Follow-up owed**: AC1.3 verify/ai-check parity table, AC3.3 JSON-before-exit subprocess guard, AC2.4/2.5 named mutations. Was: (DOC WRITTEN `planned/v0_31_0/m-z3-adt-record-sort.md`, designer `codex:gpt-5.6-sol`, quorum BLOCKED ×2; **park is ONE decision for Mark** — options (A)/(B)/(C) in the doc's "Quorum round 2 — PARKED" section; recommended **(B)** sprint `#510` first, then this routes to sprint-planner unchanged. Controller re-reproduced the bug at `f495885b1` and **REFUTED the issue's stated root cause** — the record's `declare-datatype` is ABSENT, not mis-ordered, and a direct ADT param verifies, so the ADT machinery already works. Round-2 objections split: gemini's CLOSED by controller measurement, gpt5's CONFIRMED → filed **`#510`** (unbounded Z3 exec, `solver.go:147-148`). Original sizing: P1 for clause-4/5 credibility; two lanes, ~0.5d + ~2–3d) — **`ailang#477`, filed by the Ailang World mission 2026-07-24, live-REPRODUCED at HEAD by iter-106** on a freshly-built `v0.30.0-197-g22c1eecd5` (ghost discipline applied — not a ghost). A `requires`/`ensures` contract on a function whose parameter is a **record transitively containing a user sum type** cannot be verified: Z3 gets `Invalid constant declaration: unknown sort '<Record>'` because the encoder declares the record's sort without declaring a sort/datatype for the contained ADT — **and `ai-check` exits 0** with `verify.errors: 1`. Sibling's bisection (confirmed): scalars / `list[string]` / nested user *records* encode fine; **any** sum-type field breaks the enclosing record, even single-constructor, even unreferenced by the predicate body. **Lane A — exit-code honesty** (small): `ai-check` must exit non-zero when `verify.errors > 0`; audit every caller FIRST, since gates that pass today only because of the bug will start failing (that is the point, but it should be a decision, not a surprise). **Lane B — the encoder** (real work): declare Z3 datatypes for user ADTs reachable from a contract's parameter types. **Controller-verified impact bound (do not overstate it)**: this does NOT corrupt the v1.0 headline KPI — `VerifyOk` is derived from the JSON block, not the exit code (`internal/eval_harness/verify.go:141`), and `isVerifiedSuccess` independently requires `verify_errors == 0` (`internal/observatory/cost_per_verified_success.go:94`), so an encoder error correctly EXCLUDES a run rather than counting it as proved. The real exposure is (1) any gate consuming `ai-check` by **exit code** — the normal shell/CI idiom, and a NO-SILENT-FALLBACKS violation on the verification surface — and (2) contract coverage silently bounded by an undeclared type-shape restriction. Repro fixture: the sibling's `adtsort.ail` (in #477). Triage verdict posted to ailang#477 and to the World bookkeeping thread `sunholo-data/ailang-world#9`. **Demand evidence is satisfied by construction** — a real downstream consumer is blocked (World's `w-m1-ailang-hardening` drops from 7 to 4 provable predicates).
- **[LANDED-LANE-A 2026-07-29 (iter-120) — PR #529 → squash `aa02f0d9f`, dev CI GREEN (15 checks, 0 failures, SHA-addressed); 5 commits incl. a Windows `.exe` fix Gate 3b caught; evaluator sonnet PASS 89/100 r1, zero blocking. Lane A shipped `serve-api --no-feedback-tool` (DEFAULT UNCHANGED — the live public Cloud Run service runs `--mcp-http --routes-only` and depends on the built-in surviving; tightening `--routes-only` would open a version-skew window that silently kills the public feedback channel) + the `--caps` discovery-vs-execution docs. **BONUS, and the more serious defect: `#528`** — A2A `tasks/send` never checked `isExposed`, so `@noexpose`/`--routes-only` hid a function from the agent card while leaving it CALLABLE; HTTP/MCP/OpenAPI and A2A's own card all gated, dispatch was the only hole, violating `.claude/rules/api-server.md`'s documented single-filtering-point invariant. Planner marked it cuttable; controller confirmed it first-party and upgraded it to REQUIRED (Principle 3). **Two premises REFUTED**: `--a2a` discovery is NOT affected (`buildAgentCard` never touches the MCP registry — `#498`'s title over-reaches and the controller's restatement inherited it; a milestone 'fixing' it would have been a no-op shipping green), and the `std/io` leak did not reproduce at HEAD. Five mutations all controller-run red, reverted byte-identical. **LANE B STILL OPEN — see the [NEXT] row below**] [world-DEMAND] m-mcp-exact-tool-surface Lane A** (was: NEW-DOC needed; P2 — **not a v1.0 bar item**, filed in this section beside its sibling World filing for discoverability, NOT because it is a soundness row; interim lane ~0.5d, broad lane ~2–3d) — **`ailang#498`, filed by the Ailang World mission 2026-07-28, live-REPRODUCED at HEAD by iter-110** on a freshly-built `v0.30.0-215-g7c0568797` (ghost discipline applied — not a ghost). `ailang serve-api --mcp/--mcp-http/--a2a` cannot project an **exact caller-supplied tool surface**: the built-in `submit_feedback` tool is advertised under EVERY flag combination — measured `unfiltered [addOne, submit_feedback]` · `--routes-only [submit_feedback]` · `--caps '' [addOne, submit_feedback]` · `--caps IO [addOne, submit_feedback]` (the last row is the controller's addition and corroborates that **`--caps` gates effect execution, not discovery**). **The sibling labelled its cause a HYPOTHESIS; the controller VERIFIED it** — `internal/apiserver/mcp.go:43` calls `registerFeedbackTool()` unconditionally inside `NewMCPServer`, *after* the `registerTools()` surface that `--routes-only` filters; the function takes no predicate, has no other caller, and **no env or flag off-switch exists anywhere in the repo** (`AILANG_FEEDBACK_GATE_*` is the coordinator's triage gate — `internal/feedbackgate/decide.go:14` explicitly says do not entangle the two). **Controller-verified impact bound (do NOT overstate it — this is the P0/P2 hinge)**: the *discovery* defect is unconditional and real (every connected model is told a public-feedback egress tool exists), but *egress itself* is gated on `AILANG_STORAGE=gcp` **and** `AILANG_CLOUD_PROJECT` (`internal/feedback/publisher.go:123-129`) — without both, the first call returns a structured error envelope and opens no client. So a default local server **advertises an egress tool it cannot perform**: a false capability claim to every connected model, plus a live path for environments that happen to carry those two vars — NOT a default-config exfiltration. **Lane A — interim** (small, unblocks the consumer soonest): a flag/option that suppresses `submit_feedback`, plus documenting the `--caps` discovery-vs-execution split. **Lane B — the real ask** (design-doc-sized, quorum first): export the existing serving machinery behind a narrow callback-driven Go API — caller-owned mux, principal/session resolved BEFORE discovery *and* invocation, caller-supplied exact descriptors, MCP tools and A2A skills generated from that one set, nothing built-in unless the caller supplies it. **Demand evidence satisfied by construction** — World's `w-mcp-projection` is recorded BLOCKED on it. Verdict posted to ailang#498 and to `ailang-world#9`, with the explicit note that **no date was promised**, so World keeps its item BLOCKED rather than waiting on us. Informational, independently confirmed: **`#145` is genuinely fixed** (`--routes-only` did suppress the 8 embedded `std/io` exports in every run).
- **[DOC LANDED + QUORUM-CLEARED 2026-08-04 (iter-137) — PR #582 → squash `2629ad8fa`, dev CI GREEN
  SHA-addressed (20 checks, 0 non-success) + per-workflow confirm. Doc:
  [implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md](implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md),
  785 lines, 28 verification rows, 10-row Conflict Surface. Designer `codex:gpt-5.6-sol`; reviewers
  `gpt5-6-sol` + `gemini-3-1-pro` BOTH PRESENT both rounds (no N−1 degrade); R1 BLOCKED ×2 → revision;
  R2 BLOCKED ×2 → **narrow-refinement carve-out** (both objections carried concrete reviewer-authored
  `proposed_fix`, neither disputed the design DIRECTION) → reviewers' VERBATIM fixes applied and
  recorded in the doc's Quorum Verification Log. Metered $0.1910. **The premise that shrank the work**:
  the MCP Go SDK already hands us `getServer func(*http.Request) *mcp.Server` and `mcp.go:303` calls it
  while DISCARDING the request — so #498 reqs 2–3 are a wiring/authority problem, not a transport one.
  **Two reviewer claims MEASURED, not forwarded**: (1) "per-request SDK servers break SSE" REFUTED on
  this path (Stateless mode answers GET/DELETE 405 `Allow: POST`, so no stream exists to correlate) —
  but its real adjacent landmine is closed: `Stateless:true` frozen, stateful/resumable MCP an explicit
  non-goal, GET⇒405 asserted; (2) "A2A timeout mapping unverified" — process point correct, corruption
  impossible: `a2a.go:304` already writes HTTP 200 + JSON-RPC (V27), `-32603` new to the file with a
  known-positive control (V28). R2 also caught that a deadline bounds the WAIT not the RESOURCE →
  `MaxConcurrentCallbacks` with the token held until the goroutine EXITS, plus the honest statement
  that in-process Go callbacks cannot be forcibly terminated.
  **[PLAN-READY 2026-08-04 (iter-138) — commit `6e82d2a1b`: plan
  [implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b-sprint-plan.md](implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b-sprint-plan.md)
  (631 lines) + `sprint_M-MCP-EXACT-TOOL-SURFACE-LANE-B.json` (`jq -e` rc=0, repo validator PASSED,
  zero placeholders with a known-positive control). Planner **opus**, fired via the Agent tool
  because `derive-planner-lane.sh` returned `opus declared:opus-required` — so NO codex probe or
  spawn happened for the planner role, the first end-to-end exercise of the iter-136 flip's opus arm.
  **Estimate revised 17h → 25h** (test LOC ≈ 70% of impl LOC; calibrated against Lane A's ~490 LOC
  of real work), no AC cut. **The planner refuted FIVE doc premises, all reproduced first-party by
  the controller → rows V29–V33 + an AC-corrections section:** two of M3's acceptance criteria
  **could not fail** (`make check-file-sizes` enumerates `find internal cmd` only, so it is blind to
  the `serveapi/` package this sprint creates; `make check-boundaries` iterates three fixed package
  sets containing neither `apiserver` nor `serveapi`) — and the doc's own row **V18 had already
  measured the second one**, cleared by two reviewers across two quorum rounds, with the AC citing
  that gate left standing. Both replaced with checks that can fail. **New requirement from V31:**
  `mcp.Server.AddTool` PANICS on a missing/non-object input schema (SDK v1.7.0 `server.go:282,294`)
  and the design calls it PER REQUEST from CALLER-supplied descriptors, so nil-schema rejection must
  precede any `AddTool` and the adapter must recover into the frozen `-32603` envelope. V32: `@nomcp`
  is already a second MCP-only filter, so it must NOT be folded into the shared gate. V33
  (favourable): `internal/apiserver` binds zero sockets, so most of the sprint is authoritative
  inside the codex sandbox. Milestones reordered — three of M1's six ACs are unsatisfiable at the M1
  boundary, so the callback runner becomes protocol-neutral and the `isExposed` generalization moves
  M1 → M3 as the sharpest bisect boundary. Systemic gate-scope defect filed as **`#584`**.
  **NEXT: EXECUTE** — executor `codex:gpt-5.6-sol` with the no-git-writes + cumulative `.snap/M<k>/`
  snapshot protocol, worktree pinned to `.wt-iter139-mcp-lane-b` (sibling of the repo, never
  `/tmp`), evaluator sonnet.
  **[M1 LANDED 2026-08-04 (iter-139) — PR #585 → squash `f5ebcc0b5`, Gate 3b GREEN SHA-addressed
  (20 check-runs, 0 non-success) + per-workflow confirm; evaluator sonnet PASS 94/100 r1, zero
  blocking. Shipped the public `serveapi` package (139 LOC, stdlib types only), `callerSurface`
  deep-copy/validate/sort gateway, and the bounded callback runner whose capacity token is held
  until the callback goroutine EXITS — 691 insertions, 6 files, no existing call site touched and
  no wire behaviour changed (`MCPHandler`/`A2AHandler` are declared for the compile fixture and
  return `NotFoundHandler` until M2/M3). **Controller mutation testing found TWO VACUOUS
  ASSERTIONS the executor shipped**, both fixed and both now mutation-killed: (1) the
  "next callback never entered" clause was guarded on `if err == nil`, provably false there, so
  the counter assertion was a tautology (proven — an instrumented marker never printed while the
  test ran); (2) the deep-copy proof compared two `All()` results, which under a shallow clone
  both alias caller storage and change together — **the shallow-clone mutation SURVIVED, refuting
  the sprint plan's own "would still pass if the claim were false? No" answer for AC3**. 4
  mutations, each proven applied by `cmp` before its result was read, all reverted byte-identical.
  **M2 PREREQUISITE, evaluator NB-1, controller-reproduced**: a FIFTH `AddTool` panic case —
  `validateParamHeaderAnnotations` (`mcp/server.go:312-313`, invalid/duplicated/non-primitive
  `x-mcp-header`) — is NOT covered by M1 validation, so **M2 AC5 is not sound as written**;
  recorded in the plan (`63e051de6`) demanding BOTH loud validation AND a `recover()` backstop.
  `go test ./...` rc=1 was NOT this sprint: `TestNetHttpPost` on a live httpbin.org 504,
  reproduced at the untouched base, tracked as `#561`, CI-safe (proven both ways). Latent fixture
  traps filed as **`#586`**. **M2 + M3 REMAIN** — M2 next (~8.5h, architecture pinned in plan
  §0.6), then M3 (~10h).]**
  **[M2 LANDED 2026-08-05 (iter-144) — PR #592 → squash `6166adab8`, Gate 3b GREEN SHA-addressed
  (19 check-runs, 19 completed, 4/4 required contexts success, 0 non-success) + per-workflow
  confirm; evaluator sonnet PASS 94/100 r1, zero blocking (2 own mutations, both red,
  byte-identical reverts). Shipped the request-scoped MCP adapter (`embedded_mcp.go`, 232 LOC +
  414 test) per the plan's pinned §0.6 architecture: exact per-session surfaces, no ambient
  `submit_feedback`, frozen `-32603` timeout/capacity/cancel envelopes, 401/403 typed
  authorization, `Stateless: true`, and BOTH §0.5 halves — loud gateway rejection of the
  `x-mcp-header` annotation panic class AND a `recover()` backstop. **Controller mutation testing
  found the shipped AC5 test NON-DISCRIMINATING** (third sprint running for this class): the
  gateway-validation-removed mutant survived because the backstop also writes an error envelope;
  fixed by asserting the gateway's specific rejection messages, mutant now red. Evaluator NB-1
  (overload envelope HTTP-200 assertion) reproduced + applied (`811ac16b4`); NB-4
  (context.Canceled envelope vs the doc's "no completed wire response" prose) deferred to M3.
  **[M3 LANDED 2026-08-06 (iter-151) — PR #601 → squash `b8c038647`. Gate 3b GREEN SHA-addressed
  on the PR head: all FOUR required contexts (`test`/`lint`/`build`/`docs-gate`) `success`,
  20 check-runs completed, 14 success / 5 skipped. ⚠ ONE non-required failure, NOT buried:
  **CodeQL** `go/reflected-xss` high at `embedded_mcp.go:119` — established by measurement as a
  SCAN-CADENCE artifact describing **M2's** already-landed code, not this diff (file byte-identical
  between dev and PR head; CodeQL analyses dev only WEEKLY and last ran `2026-08-04T09:04`, ~22h
  BEFORE M2 landed `2026-08-05T07:33`, so that file had never been analysed on dev). Filed as
  **`#603`** with the true-positive question left open. The 2nd alert (`#129`) is pre-existing on
  dev since 2026-04-21 — my first read said "0 on dev" from a `per_page=50` page against ≥100
  alerts, corrected by paginating. Executor codex `gpt-5.6-sol`, evaluator **sonnet PASS 81/100 r1,
  zero blocking**. **LANE B IS COMPLETE.** Shipped `embedded_a2a.go` (162 LOC — cards + `tasks/send`
  projected from the SAME `AuthorizedSurface` as MCP, dispatch via `Lookup` so an unauthorized send
  is `-32602` and never reaches the invoker), live `A2AHandler()`, `Mount` onto a caller-owned mux,
  and `loadedExportMember` as the single protocol-neutral MEMBERSHIP gateway behind all 6 production
  `isExposed` sites with `@nomcp` kept as an MCP-only PROJECTION applied after membership (the §0.7
  hazard; controller-mutated, caught by THREE tests incl. the pre-existing
  `TestNoMCP_StillServedOverHTTPAndOpenAPI`). **THE FIND — the 4th consecutive non-discriminating
  test in this sprint, and the first where the REVIEWER'S PROPOSED FIX was also wrong**: the
  evaluator caught `TestMountRecorderRoutesAndMCPStripPrefix` passing with `StripPrefix` REMOVED and
  proposed a sub-path POST; measuring instead of applying it showed the MCP SDK's
  `StreamableHTTPHandler` **never dispatches on path** (`URL.Path` = 0 across `go-sdk@v1.7.0/mcp`
  non-test, `.URL` = 12 control), so the wrapper is behaviourally INERT and the AC was vacuous BY
  CONSTRUCTION — no assertion can distinguish it. Fixed by removing the false claim (test renamed,
  comment records the measurement and that the wrapper is deliberately uncovered) and replacing it
  with a `/mcp/` SUBTREE assertion that IS discriminating. Executor also REFUTED the planned
  transitive-import boundary check as impossible (`serveapi` transitively reaches 13 compiler
  packages; the correct gate is DIRECT-import, 0 with control 1). Doc + plan →
  `implemented/v1_0_0/`. **→ RELEASE ASK now owed to Mark** (World consumes pinned releases only).]**
  Was: M3 REMAINS — the FINAL Lane B milestone (~10h: A2A projection, Mount, exposure
  generalization, docs, CHANGELOG, gates).]**
  Was: NEXT — route to sprint-planner; the QUEUE HEAD after m-planner-codex-lane LANDED at iter-136.
  Was NEXT #3, REORDERED by Mark 2026-08-03 afternoon when the PLAN-READY m-planner-codex-lane
  execution jumped ahead (one ~8h mechanical sprint; every later iteration burns less opus).
  Original directive: PRIORITIZED for Ailang World — was pick #2 after m-recorded-stream-api] [world-DEMAND]
  m-mcp-exact-tool-surface LANE B** (doc DONE — was: NEW-DOC needed, quorum required; P2→**P1 by directive** —
  still not a v1.0 bar item, but it is World's SOLE clause-6 external blocker (their
  w-mcp-projection is BLOCKED on it, recorded in their charter) and Lane A alone does not
  give them the caller-supplied per-session surface. RELEASE NOTE for the controller: World
  consumes upstream via PINNED RELEASES only — when Lane B lands, surface a release ask to
  Mark in the report's DECISIONS row (releases are Mark's sole decision); a tag would also
  carry the already-landed #510/#477/#498-Lane-A fixes World is waiting on. (Original row:  ~2–3d) — the REAL ask in `ailang#498`, untouched by iteration 120's Lane A. Export the existing serving machinery behind a narrow callback-driven Go API: caller-owned mux, principal/session resolved BEFORE discovery *and* invocation, caller-supplied exact descriptors, MCP tools and A2A skills generated from that one set, nothing built-in unless the caller supplies it. **Demand evidence satisfied by construction** — World's `w-mcp-projection` remains BLOCKED on this (Lane A only unblocks it if a suppression flag suffices; the sibling was told explicitly that no date is promised). Lane A's landing means the interim workaround exists, so this is no longer urgent — but it is still the thing `#498` actually asked for.
- **[LANDED-LANE-A 2026-07-30 (iter-121) — PR #536 → squash `a81d66983`, dev CI green on the PR (15 pass, 0 failures); 5 commits. Doc RESUMED from a prior iteration-121 attempt that died 14:14 the same day leaving a 453-line doc on an UNMERGED branch (invisible to a `design_docs/` grep and to Gate 2's origin/merged-PR checks — see the log's process-fix note). Quorum: designer `claude:claude-fable-5`; R1 BLOCKED (both objections Lane-B2-only) → B2 DEFERRED on `gpt5-6-sol`'s own proposed option, blocked on a deterministic evaluator fuel budget; R2 BLOCKED **N−1 DEGRADED** (⚠ gpt5-6-sol absent) with a genuine new B1 catch from `gemini-3-1-pro` (a `()` generator would have hit the new loud-error default arm) → narrow-refinement carve-out, fix applied verbatim + acceptance row B-4a. Planner opus **refuted 5 controller premises**, incl. my inference that skip sites 331/454 were a live bug (they are unreachable dead code — I verified the refutation) and my 17-file blast-radius claim (really 4 silent-green — **my own later baseline confirmed the planner against me**). Executor codex `gpt-5.6-sol` (first run correctly REFUSED to start: the state validator rejects `estimated_loc == 0`, and M5 is docs-only). Evaluator sonnet **PASS 88/100 r1, zero blocking**; both NBs re-verified first-party and NB-2 was **bigger than filed**. Five mutations controller-run RED, reverted sha256-identical, incl. one the evaluator ran that I had not. Shipped: reachable `TypeApp{"list"}` arm (the `ListType` arm dead since `b9ab84e6f`) · total fail-closed skip taxonomy over all SIX `StatusSkip` sites · `Success()` requires `VacuousSkips == 0` with `--allow-skips` the single opt-out · `--format json` emits only JSON to stdout. Strictness is CLASS-SCOPED — `out_of_contract` stays forgiven, `cross_module_functions_lib.ail` is the discriminator and stays rc 0. ⚠ `runner.go` now **790/800** lines, so Lane B1 must route additions elsewhere. **THE BIGGER FIND IS `#535`**: property generation is wall-clock seeded, so the same file on the same binary gives rc=1, rc=1, rc=0 — pre-existing, violates Principle 4, no `--seed` flag exists, and it retroactively explains why the executor's and my corpus numbers differed (neither was wrong). **LANE B1 STILL OPEN — see the [NEXT] row below**] [world-DEMAND] m-property-generator-coverage Lane A** (was: NEW-DOC needed, quorum required; P2 — **not a v1.0 bar item**, filed in this section beside its sibling World filings for discoverability, NOT because it is a soundness row; Lane A ~0.5d, Lane B ~2–3d) — **`ailang#517`, filed by the Ailang World mission 2026-07-29, live-REPRODUCED at HEAD by iter-118** on a binary built from `5998f4039` (ghost discipline applied — not a ghost). Contract-derived property tests **run zero cases and report `skip`** for any parameter the generator table does not cover, while the suite still reports **`success: true` and exits 0**. This is the mission's own **vacuous-pass class**, third instance in this repo after the silent-`z3` skip and CI `t.Skip` — a check reporting success for work it never performed. **The controller's repro is WIDER than the filing on two counts, both measured, and each changes the fix**: (1) it is **not "ADTs and records"** — `createGeneratorForType` (`internal/testing/runner.go:630`) has exactly two arms, `*ast.SimpleType` in {`int`,`float`,`bool`,`string`} and `*ast.ListType`, so **tuples, ADT-free plain records, AND `list[T]` all skip** (measured in one file: `c: Color` · `r: { a: int, b: string }` · `t: (int, int)` · `xs: list[int]`, all `tests_run=0`); (2) **the list arm is DEAD CODE** — `DX-17 Phase 2` (`b9ab84e6f`, "Normalize [T] syntax to TypeApp at parse time") made the parser emit `*ast.TypeApp` for **both** `[int]` and `list[int]`, and this consumer was never updated, so `NewListGenerator` is unreachable from any real program; `&ast.ListType{}` is now constructed **only in test files** (`internal/types/cycles_test.go`, `internal/gen/golang/adt_test.go`), which is exactly what kept it looking alive. That is the repo's recurring **guard-the-call-site-not-the-helper** shape. **Controller-verified impact bound (do NOT overstate it)**: the guard is **half-present, not absent** — `SuiteResult.Success()` (`internal/testing/result.go:97`) is `ran > 0 && FailedTests == 0`, and an `AllSkipped()` sentinel plus `--allow-skips` already exist with the comment "an all-skipped suite is NOT success". Measured: **1 pass + 1 skip → `success:true`, exit 0**; **all-skipped → `success:false`, exit 1**. So the silent shape requires **≥1 passing test alongside** — which is every real module, and means **a minimal one-function repro exits 1 and will read as already-fixed**. Any regression test MUST use the mixed shape or it passes for the wrong reason. Human mode is softer than it looks too: it prints `⊘` and a `1 skipped` line but headlines **`✓ All tests passed!`** and drops the reason entirely — `no generator for parameter …` lives **only** in JSON `properties[].error`, and `properties[]` is a **separate array from `tests[]`** (which is why the sibling's `len(tests[])` count check was blind). **Lane A — make it loud** (small, lands first): surface partial skips at the exit-code/`success` level (or at minimum on stderr), and fix the dead `TypeApp` list arm — that one is a straight bug fix, not a design question. **Lane B — derive generators structurally** (the doc; product-of-fields for records, sum-over-constructors for ADTs, with recursion depth/size bounds, shrinking, and a user-supplied-generator escape hatch for what cannot be derived — the sibling's ask (3), folded in here because Lane B needs it). **Demand evidence satisfied by construction** — World's CI gate printed `✓ all 14 required named tests pass` while five properties over its core world types ran zero cases. Verdict posted to `ailang#517` and to `ailang-world#9`, with the explicit note that **no date was promised** and the interim assertion that actually holds (`skipped_tests == 0`), so World keeps its local gate fix rather than waiting on us.
- **[M1 LANDED 2026-08-05 (iter-143) — PR #591 → squash `c440a1628`, Gate 3b GREEN SHA-addressed (19 check-runs, 19 completed, all 4 required contexts `test`/`lint`/`build`/`docs-gate` `success`); 2 commits; evaluator sonnet **PASS 92/100 r1, zero blocking**, and the evaluator independently re-verified three mutations, the inert-gate removal and the skip-honesty question rather than taking the controller's word. Shipped `internal/testutil` **357 LOC**: a THREE-state `LiveNetworkStatus()` predicate (skip/fatal/run) deliberately extracted — you cannot assert "`t.Skip` was NOT called" from inside a test that would be skipped, so testing the wrapper is vacuous *by construction*, and testing the predicate makes both directions real · `RequiresLiveNetwork` fail-loud on the poisoned-live-lane mis-combination, which **never unsets** (V29: Go caches proxy env process-wide, so a runtime unset silently does nothing) · `HangGuard`/`HangGuardContext` · `RunBounded`. Package stays stdlib-only (`go list -deps | grep sunholo` = **1**; control: `internal/effects` = 11). All **6** assertions mutation-proven by the controller OUTSIDE the sandbox, negative control green first, each mutation proven landed by byte-diff, both sources reverted sha256-identical. **THE FIND: the executor shipped an inert `testing.Short()` gate INTO THE PACKAGE BUILT TO REPLACE THEM** — `-short` is passed nowhere in this repo, so it is defect class **C4**, and M3's gatelint **R1 (zero exceptions)** would have red-lighted this very sprint two milestones later; measured **7→8** with the doc's own **V2** command, removed, back to **7**. **CI caught a second defect unreproducible locally**: Windows env vars are case-INSENSITIVE, so `http_proxy`/`HTTP_PROXY` are ONE variable and the predicate correctly reports the upper-cased name — a case-sensitive assertion can never match there (production code was right; fixed `f45e8bab1`, still non-vacuous). ⚠ **M5 FOLLOW-UP, measured not guessed**: SonarCloud `new_coverage 67.9% < 80%` (**non-required**; control shows `success` on `97a4ac9d3`, so it is genuinely new with M1). The real gap is `String()` at **0.0%**; `RequiresLiveNetwork` also reads 0.0% but that is a **re-exec artifact** — it executes in the subprocess child, invisible to the parent coverage profile, and the evaluator confirmed that path IS exercised. Sonar therefore **UNDERSTATES** real coverage: do NOT "fix" it by weakening the subprocess pattern. **`D5` IS DECIDED (iter-145, Option A) AND M2/M3/M4 ARE UNBLOCKED — see the sequencing note below**] m-ci-flake-systemic-fix — **M1 LANDED** (iter-143 `c440a1628`) · **M2 LANDED** (iter-145 `368f940cf`, PR #593) · **M3 LANDED** (iter-146 `13c570063`, PR #597 — gatelint R1/R2/R3 + AC8/AC10(a-d), 941 files scanned / 0 violations; allowlist seed RE-MEASURED post-M2 because the plan's predated it: R1 **empty**, R2 **one** entry reusing the reason M2 authored in-file, R3 **five**; `net_test.go`/`gitcache_test.go` are GATED and deliberately NOT allowlisted) → **M4 LANDED** (iter-147 `4b47f8b0a`, PR #599 — poison wired across all **6** legs + AC9 gatelint registration, closing **AC9/AC11/AC12**; evaluator sonnet PASS 88/100 r1 zero blocking; Gate 3b 20 checks / 0 non-success incl. all 4 build legs. `#569` was re-verified BY PURPOSE and merged first as `bc30912ea` to clear the collision. **The first CI run went RED on M4's own AC9 step**: `go mod download all` touches the tracked `go.sum` and the staleness detector compares binary mtime against newest Go source, so prefetching AFTER `Build binaries` made every binary read STALE and silently skipped 3 binary-gated tests — fixed by moving the prefetch BEFORE the build in both legs, as build.yml already did. **`test-windows` green closes the PowerShell guard, which was unverifiable locally.** C2 watch-item re-measured and WEAKENED: the 30s budget wraps only `probeServeAPIMCPTools` (subtests 0.75s/0.03s), so the margin is ~40× not 2.9×) → **M5 LANDED** (iter-148 authored / iter-149 merged: `c9e1a4f98`, PR #600 — changelog, the `Deterministic Test Boundaries` guide section, and the doc↔plan reconciliation; evaluator sonnet PASS 88/100 r1 zero blocking, full AC1–AC12 sweep re-run out-of-sandbox, all pass) — **⚠ THE SPRINT IS COMPLETE; DOC + PLAN MOVED TO `design_docs/implemented/v0_33_1/`** (plans travel with their doc). Closes `#583`/`#494`/`#509`/`#587`/`#561`, each verified in code before closing rather than on the doc's `Closes:` claim. **M5 was written by an UNRECORDED iteration 148** that opened PR #600, went green, then died before Gate 3b — leaving zero charter/log/STATUS trace (`grep -c 'ITERATION 148'` = **0** in both files) and an **OPEN** PR that Gate 2's *merged*-PR search structurally cannot see; the worktree `.wt-iter148-ci-flake` was the corroborating trace. See ITERATION 149's stamp** (`#583`, `#494`, `#509`, `#587`, `#561`; **P0** — flakes red-light `dev`, and a red `dev` outranks this queue every time it occurs. Doc `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` LANDED `dec17dab1`, quorum-cleared over 3 rounds; **sprint plan LANDED `7cb798d98`**: 5 milestones / 1265 LOC / **26h ≈ 4.5d**, revised up from the doc's 3–4d. No new doc and no re-quorum — the iter-142 corrections are *measurements*, not design changes) — **⚠ THIS ROW WAS MISSING UNTIL ITER-142.** Iteration 141 landed the design doc under Mark's directive ("Yes sprint a CI flake fix") and recorded it **only in its STATUS stamp**; a grep of the queue for the item returned **zero** rows. Filed retroactively, and the generalisable point is worth more than the fix: *a doc that lands without a queue row is invisible to every later Gate-2 pick* — only iter-141's own hand-written "Next" line kept it alive. **SEQUENCING IS NOT FREE CHOICE**: (1) **M1** (`internal/testutil` gate + bounded-subprocess helpers, 370 LOC / 5h) is the **only unblocked milestone** — new package, nothing imports it, **zero blast radius**; take it first. (2) ~~**M2 and M4 are BLOCKED on `D5`**~~ → **`D5` DECIDED 2026-08-05 (iter-145) = OPTION A, by Mark** (verbatim: *"D5 - option A and then queue the B afterwards. I'm cool with 2."*). AC3 was **vacuous** because the poisoned-proxy boundary does not cover AILANG's own `Net` effect (6 hand-built `http.Transport{}` in `internal/effects`, none setting `Proxy`; `ProxyFromEnvironment` in **0** first-party files — **V33**). **Applied:** AC3 → **AC3′(a/b/c)** narrowed to `./internal/pkg/`, the `internal/effects` egress tests move behind `RequiresLiveNetwork`, and the residual is asserted *as open* by the new **AC10(d)** — which carries a `Proxy: http.ProxyFromEnvironment` arm as its known-positive control and will go RED if Option B ever lands, making it the tripwire that retires itself. **Option (B) is queued separately** (see the Option-B row) — it must not ride in on a sprint scoped and reviewed as test-only. (3) **M3 depends on M2** (`TestGateLint_Repo` scans the real tree), so M2 goes first; **M5** is docs-only. (4) **M4 is the ONLY CI-touching commit** — it wires the poison across **6** legs (**V34**: `build.yml`'s matrix has 4 jobs, not 3 — `macos-latest` appears twice, amd64 + arm64 — plus ci.yml `test` and `test-windows`); watch the first `dev` run after it lands, with `git revert --no-edit <M4-sha>` staged. ⚠ **COLLISION — DECISION TAKEN iter-145, plan §6.1 option (b) not the recommended (a):** PR **`#532`** rewrites `buildAilang` in `cmd/ailang/main_test.go` and the body under the `testing.Short()` gate in `serve_api_mcp_surface_test.go`, and touches `ci.yml` — **exactly M2's surface**. **M2 goes first; #532 rebases onto it after.** Measured basis: #532 is authored by `sunholo-voight-kampff` (*this loop's own PR* — no external author to coordinate with), and it has been `CONFLICTING`/`DIRTY` against `dev` since **2026-07-29**, untouched for 7 days — so **M2 does not make it any more conflicted than it already is**; the resolve cost is pre-existing debt, not one M2 creates. Re-application is symmetric whichever side goes second, so ordering was chosen on unblocking value. A comment recording this is posted on #532 so its rebase re-applies `HangGuardContext` to the new `sync.Once` body. `#569` (dependabot actions bump) touches `ci.yml` + `build.yml` = M4's surface, re-check before M4. ✅ **RESOLVED iter-145:** the iter-141 narrow-refinement carve-out (R3 quorum fix applied with no re-quorum on that fix) is **RATIFIED by Mark** — *"ACCEPTED as-is. No re-quorum needed"* — **veto window closed**.
- **[LANDED 2026-08-20 (iter-235) — `D-1` APPLIED AND THE HOLD RELEASED.** PR [#613](https://github.com/sunholo-data/ailang/pull/613) → squash **`e5ee6c5e5`**; Gate 3b SHA-addressed on head `a54a8624f` = **21 checks, ZERO not-green**, 4/4 required (`test` 18m09s, `lint` 3m00s, `build` 2m09s, `docs-gate` 4s), `MERGEABLE/CLEAN` before merge. The held regression was reproduced first-party, same command both arms under the ci.yml poison: branch **rc=1, 4 of 7** subtests failing vs pristine dev **rc=0, 7/7**. `proxyRoundTrip` now validates literal-IP targets with `net.ParseIP` + `validateIP` — **zero resolver calls** — and refuses before any transport is built, wrapped in the existing `*targetValidationError` so `E_NET_IP_BLOCKED` survives `*url.Error`; hostnames stay the accepted `D-5` residual. **Round-2 fixed two evaluator findings BEFORE merge**: (a) `net.ParseIP` rejects RFC 4007 zone identifiers, so `http://[fe80::1%25eth0]/x` reached the proxy **unvalidated** (`resolverCalls=0 dialCalls=1`, 200) — `literalHost()` trims at the first `%`, and the DIRECT route was measured to fail CLOSED (`E_NET_DNS_FAILED`), so the leak was proxy-only; (b) `proxy_literal_blocked_before_dial` **survived its own precondition being removed** because the direct route refuses the same literal identically — it now asserts the proxy selector was consulted. Five arms; two sole-killer mutations (inverse `-skip` rc=0 both) plus one 6-subtest broad-blast mutant with every member explained. Evaluator sonnet **PASS 93/100**. Executor `codex:gpt-5.6-sol`; **zero Fable runs** (no designer fired). Superseded row text below.] [M1 IMPLEMENTED + HELD 2026-08-07 (iter-156) — PR [#613](https://github.com/sunholo-data/ailang/pull/613), a DRAFT titled DO-NOT-MERGE; branch `sprint/iter156-net-proxy-boundary-m1`, commit `c44e3d8b2` (984 insertions: `net_proxy.go` 150 + `net_proxy_test.go` 781 + the three routed constructors). **M1's OWN acceptance PASSES** — AC-M1.1 controller-run `rc=0`, `=== RUN` **17** (≥4), all four top-level `--- PASS`, against a pristine-tree baseline of `rc=0` / `=== RUN` **0** / `[no tests to run]` (vacuously green, so the count is the load-bearing half); AC-M1.2/1.3/1.4 each closed by a named assertion. Every plan-cited line number re-derived first-party and all exact. Executor **pi `deepseek-v4-flash-0731`** (production + first test draft) **+ an opus FALLBACK, FLAGGED** — spawned on a **FALSE death diagnosis of mine** (`pgrep -f` false negative plus a truncated tail of a 150 MB NDJSON still being written; pi in fact ran to `rc=0` over 47 turns), which put two agents in one worktree. Benign only because the second detected the collision, refused to clobber, and found **four genuine vacuity defects** in the first's tests — chief among them that the AC-M1.4 subtests sat in a **fifth top-level test the graded `-run` regex EXCLUDED**, so they never ran under the gate at all. Evaluator sonnet **HOLD 71/100**, generator≠judge intact; it reproduced both arms of the finding below, ran three further distinct mutations (all sha256-restored), and measured **zero goroutine delta** across 20 sequential / 50 concurrent requests (refuting my per-request-transport leak worry) with **no behavioural drift** from the original bare transports. ⚠⚠ **WHY IT IS HELD — AND IT IS NOT A DEFECT IN THE CODE: IMPLEMENTING THIS DESIGN FAITHFULLY REMOVES AN EXISTING SECURITY CONTROL.** Validation now happens only on the DIRECT route, so a target that is a **literal private IP** (`https://10.0.0.1/…`) is handed to the proxy **unvalidated**. Under the exact poison `ci.yml` sets on its own test legs (`:89-91`, `:350-352`), `TestNetIPValidation` is **rc=1 with 4 of 7 subtests FAILING on this branch** and **rc=0 with 7 PASS on pristine dev** — a proper negative control, so the attribution is mechanism rather than co-occurrence; the three loopback subtests survive precisely *because* `NO_PROXY` routes them direct. So M1 alone **reds CI**, and **`D-1` is no longer an abstract trade — it is a demonstrated loss.** ⚠ **A THIRD OPTION THE DOC DOES NOT CARRY, endorsed independently by the evaluator**: `resolveAndValidateIP` **already** has a `// Special case: raw IP address (skip DNS)` branch calling `validateIP` with **zero network I/O**, so validating a literal-IP target on the proxy route needs no DNS and carries no TOCTOU/rebinding risk — the doc's own rationale for skipping validation does not reach that case. It would restore parity and shrink `D-1` to **hostnames only**. NOT implemented unattended: rule vii says the reviewed doc wins, and Standing rule 2 forbids narrowing a quorum-cleared security boundary without the human. Two further honest gaps: `proxySelector` is a test seam pi added to PRODUCTION that the doc never specified (nil in production, so behaviour is unchanged) and it leaves the production `http.ProxyFromEnvironment` default with a **single point of coverage** — my mutation D reds the gate but via exactly ONE subtest; and **no test covers a redirect that CHANGES proxy applicability per hop**, which the doc advertises (evaluator `NB2`, UNVERIFIED). M2–M4 unstarted. Was:] [NEXT — SPRINT-PLANNED, READY FOR M1 2026-08-06 (iter-155): UNPARKED by Mark's `D-6 = (A)`; BOTH quorum objections resolved with NO third round; plan + sprint JSON landed (`m-net-effect-proxy-boundary-sprint-plan.md`, 636 lines, 4 milestones / 12 ACs each owned exactly once, 3-day shape kept; `sprint_M-NET-EFFECT-PROXY-BOUNDARY.json`). Commits `945f36727` (doc revision: rows **V20**/**V21**, D-6 recorded) + `7c7e5e58a` (plan) — DOC-ONLY and **UNPUSHED pending the Actions outage**. `gemini-3-1-pro`'s unverified-premise objection was satisfied **by measurement, not a revision round** (row V20 names the two call sites the error-mapping must update: `net.go:567` preflight, `:631` post-`client.Do`); `gpt5-6-sol`'s AST-analyzer objection was resolved by the human ruling, with the analyzer **FILED AS `#612`** at resume time — Option A is "cheap gate now AND the durable gate filed", so that filing is part of its definition of done. Planner opus, lane `opus fail-closed:env-pin` used verbatim. Planner's baseline sweep (rule 3e) found `go build ./...` **already rc=1 on unmodified dev** (excluded, scoped build substituted) and all three M1/M2 named-test gates **rc=0 with 0 `=== RUN` lines at base** — every named-test AC now asserts a `=== RUN` count, not just an exit code. ⚠ **`D-1` STILL OWED A HUMAN RATIFICATION** (this design knowingly trades target-IP SSRF pinning on PROXIED routes; preserved on direct/`NO_PROXY`, and the doc never claims equivalence). Was: PARKED needs-human-review 2026-08-06 (iter-150) — DESIGN DOC LANDED (`design_docs/planned/v0_33_1/m-net-effect-proxy-boundary.md`, 662 lines incl. its quorum log), QUORUM BLOCKED ×2, ONE DECISION OWED. Designer codex `gpt-5.6-sol` (rotation advanced). R1 blocked on a REAL defect: `gemini-3-1-pro` caught target-IP resolution specified in TWO places (preflight `resolveAndValidateIP` AND the new RoundTripper) = a TOCTOU DNS-rebinding race, plus a broken-proxied-request risk on hosts without external DNS; fixed by making the direct RoundTripper the sole resolve-validate-dial site and skipping local target DNS entirely on proxy routes (V17/V18/V19 added, 520→592 lines). R2's `gemini-3-1-pro` objection is CARVE-OUT-ELIGIBLE and its answer is ALREADY MEASURED, so the resume is cheap: `E_NET_IP_BLOCKED` at `net_security.go:27,34,46,51,56`, `E_NET_DNS_FAILED` at `:90,94`, surfaced via `makeResultErr("Transport", …)` at `net.go:551,556,567,605,631,639` (control 11 sites; `:567` is the preflight path being moved, `:631` the post-`client.Do` path where a `url.Error` arrives). **R2's `gpt5-6-sol` objection is what parks it**: replace V2/V17 AND the M4 completeness gate with a checked-in `go/packages` AST/type analyzer plus positive fixtures, because textual matching cannot see aliased imports, `new(http.Transport)`, post-construction `Client.Transport =`, transport-returning factories, or custom `RoundTripper`s. That materially expands scope and needs a judgment call → Standing rule 2: park, do not force through. **The decision is cheap to answer because I tested the reviewer's own hypothesis: ALL FIVE shapes are ZERO at HEAD**, each with a firing control (aliased imports 0 / control 1505; `new(http.Transport)` 0 / control 4; `.Transport =` 0 / control 8; transport factories 0 / control 2; `RoundTrip(` 0). The seven-site claim is therefore empirically COMPLETE today; the live argument is only about the gate's DURABILITY against future escapes. ⚠ Two of my five controls failed on first run (the alias matcher matched the `import` keyword itself; the `new(` control returned 0, which rule 3a makes uninformative) and were re-run before any number was used. **D-6 ANSWERED 2026-08-06 (Mark, attended): (A)** — grep gate now, sprint stays 3d; the AST analyzer is to be FILED AS A SEPARATE FOLLOW-UP row when this sprint resumes (that filing is part of option A's definition of done). Option (B) declined. Row is UNPARKED — resume via the pre-measured R2 carve-out above. Non-blocking but worth ratifying alongside: **D-1 knowingly trades target-IP SSRF pinning on PROXIED requests** (preserved on direct/`NO_PROXY`; the doc is explicit and never claims equivalence). Was:] **[NEXT] [NEW-DOC] m-net-effect-proxy-boundary — `D5` OPTION B, queued by Mark's directive 2026-08-05** (*"queue the B afterwards"*; **P2**, sequenced AFTER the CI-flake sprint completes — it is the better end state, but it is a **production runtime change** and must not ride in on a sprint scoped, reviewed and quorum-cleared as test-only). **The work:** set `Proxy: http.ProxyFromEnvironment` on the proxy-ignoring hand-built `http.Transport{}` literals, bringing AILANG's own `Net` effect inside the poisoned-proxy egress boundary. ⚠ **SCOPE CORRECTED 2026-08-06 (iter-149) — it is 7 literals across 4 files, NOT the 6-across-3 this row used to claim.** Re-measured first-party with controls (the matcher sees **11** including tests; `ProxyFromEnvironment` appears only in M3's deliberate control arm, so V33's production-zero still holds): `internal/effects/net.go:96,212,587` · `internal/effects/stream_ndjson.go:80` · `internal/effects/stream_sse.go:70,329` (= 6 across 3 files, the old count) **PLUS `internal/executor/managed_agents/client.go:141`**, which sets only timeouts and so bypasses the poison identically — but lives OUTSIDE `internal/effects`. Surfaced by iteration 148's M5 sweep and re-derived rather than inherited. **The design pass must decide whether the `managed_agents` client is in scope at all** — it is not part of the `Net` effect, so "bring `Net` inside the boundary" and "close the first-party residual" are two different jobs, and AC10(d) asserts the *mechanism*, which is file-independent. **Why it needs its own design pass + quorum, not a queue row's worth of thought:** a proxy resolves the hostname *itself*, which is exactly what `net.go`'s **pinned-IP SSRF guard** exists to prevent — so this plausibly breaks a security control, and the interaction is the whole design question. It is also a behaviour change for every AILANG program that uses `Net`, in a repo whose CI-flake doc explicitly disclaims runtime changes. **⚠ CORRECTED 2026-08-06 (iter-155) — THE "ACCEPTANCE SIGNAL ALREADY BUILT AND WAITING" WAS FALSE, AND THIS ROW ASSERTED IT FOR THREE ITERATIONS.** The row used to read: the CI-flake sprint's **AC10(d)** "measures this residual as OPEN and will go **RED** when Option B lands — so the tripwire that tells this item it succeeded already exists". **It cannot go red.** `testEffectsProxyResidual` (`internal/testutil/egress_posture_test.go:66-85`) builds its **OWN** `&http.Transport{}` at `:74` and exercises that, while its comment claims it trips when `internal/effects` gains `ProxyFromEnvironment`. Measured with the tool that cannot miss: `go list -f '{{join .TestImports}}' ./internal/testutil` = **12 imports, ALL stdlib, ZERO ailang packages** (control: `./internal/effects` = **6** ailang imports, so the instrument sees them when they exist). No production change can alter its outcome — a tripwire watching a local replica of its own subject, shipped through a full sprint, a quorum and an evaluator PASS. Surfaced by the iter-155 sprint-planner refuting the controller's own briefing, then confirmed first-party. **The design doc was right all along** ("**helper-only** residual logic"), so **M4 is a deliberate DELETION, not an observed red**, and no AC depends on watching that tripwire flip; retiring it + the matching Non-Goals text stays part of this item's definition of done. The old sequencing constraint ("do not start before M2–M5 land") is **satisfied** — M1–M5 all landed (`c440a1628`…`c9e1a4f98`) — and was in any case predicated on the same false coupling. Sibling defect, same root cause: `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix-sprint-plan.md:460` describes an AC10(d) falsification drill that is not executable.
- **[CLOSED 2026-08-06 (iter-152)] [SWEEP-BATCH iter-145] Three zero-mention open issues found by the weekly external-issue sweep** — **ROW RESOLVED, all three dispositioned.** `#588` closed at iter-150. **`#590` FIXED** — PR #605 → squash `34951811d`, Gate 3b GREEN SHA-addressed (20 checks, 0 pending, all four REQUIRED contexts `test`/`lint`/`build`/`docs-gate` **success**; the single non-success is SonarCloud, **non-required**: 78.7% new-code coverage vs 80% and 8.0% duplication vs 3%, the same class as iter-126, analysed revision == PR head so it is a genuine reading and not the stale-Sonar trap — the duplication is the 7 near-identical `.ail` fixtures and is worth revisiting, not hiding). Reproduced 100% at HEAD before routing; **the find is wider than the filing** — `assert` is parseable in exactly ONE construct (`registerPrefix(lexer.ASSERT)` = **0**, control **26**) and that construct was 100% broken, so the keyword had **no working call site anywhere in the language**. Lowered in the **fold**, not the printer, because the printer cannot see sequencing (a per-node lowering would make `{ assert false; assert true }` PASS). Evaluator sonnet **PASS 88/100 r1, zero blocking**, 7 mutations killed + an 8th controller mutation (5 tests red, reverted byte-identical). **`#589` NOT REPRODUCED, deliberately left OPEN** — 0/10 on the closest available shape with a firing control, but this repo's widest multi-inline-test file has **1** import against the report's 15-module closure, so the negative is equally consistent with "fixed" and "the instrument cannot express the trigger". That is an unmet repro burden, not a refutation; verdict posted naming what would move it (the reporter's module, or a synthetic wide-closure module run in a loop). Was:] (Gate 0.5; measured 2026-08-05 against 45 open issues, control `#498` = 8 charter mentions so the instrument sees positives). None outranks the queue — a sweep never does by itself. (1) ~~**`#588`**~~ **DONE — closed 2026-08-05 citing the M2 squash; iter-150 confirmed the close had already happened (my close attempt was a no-op and posted no comment).** ⚠ Two corrections to how this row described it, both measured first-party rather than inherited: M2 gated the live-network **subtest** (`internal/effects/net_test.go:364`), **not** the whole `TestNetHttpPost` function — whose remaining subtests are pure `E_NET_TYPE_ERROR` type-checking and are correctly left ungated (control: the same gate appears at `net_test.go:441` and `internal/pkg/gitcache_test.go:51`, so the ungated outer function is a measurement). Practical effect for the issue is unchanged. (2) **`#589`** *[motoko_agent] `ailang test`: cluster harness nondeterministically fails a passing test (6/10) with 'record has no field' naming a field outside the call's dependency closure* — a genuine language/runtime bug from a real downstream consumer, and **nondeterministic**, which makes it a soundness smell rather than a cosmetic one; needs ghost-discipline live-repro at HEAD before it earns a design doc. (3) **`#590`** *[motoko_agent] Named test blocks with `assert` always fail: `EvaluateNamedTestBodyExprs` round-trips `AssertStmt` through the general parser, which doesn't accept it* — reads as a small, well-localised parser/eval defect with the mechanism already named by the reporter; likely a same-day AILANG-fix lane item, **not** a sprint. ⚠ Both `#589`/`#590` are `from:motoko_agent` reports, so the standing rule applies: **verify against HEAD before filing** — motoko_agent reports pin a stale `ailang_version` and have twice been already-fixed or superseded.
- **[LANDED 2026-08-07 (iter-156) — BOTH HALVES MERGED. `#606` → squash `49a6af789` (`#603`), `#608` → squash `1d355245a` (`#602`); both issues auto-closed. All four required contexts (`test`, `lint`, `build`, `docs-gate`) **pass on a real `pull_request` check suite**, `mergeStateStatus=CLEAN` on both before merge. These sat blocked for three iterations not on their code but on the Actions outage, and unblocking them is a finding in its own right: `workflow_dispatch` was NOT enough — its checks land on the head SHA but do not satisfy branch protection, which is gated on the `pull_request` suite the outage had wedged (`jobs=0`, uncancellable, 7h+). The unwedge was a **tree-identical empty commit through the git API** (`POST git/commits` with the existing tree sha + current head as parent, then `PATCH git/refs/heads/<br>`), which fires a genuine `synchronize` and needs no local checkout. Recorded in the shared skill's Gate 3b. Was:] [NEXT — HALF DONE] [FINDINGS-BATCH iter-151] Two CI/security findings measured during M3 verification, batched into one row.** ⚠ **`#603` FIXED at iteration 153 (PR #606, 2 commits; evaluator sonnet PASS 91/100 r1, zero blocking) — NOT yet marked LANDED, because a declared GitHub Actions major outage meant Gate 3b saw 0 failures but never a full green, and a timed-out wait is not green. Resume = confirm #606 green and merge.** The triage verdict: CodeQL's taint trace is CORRECT (2 of 12 hostile shapes reflect a literal `<script>`) but NOT exploitable (Content-Type present 12/12, reflecting paths carry `nosniff`, json escapes) — fixed rather than dismissed because all three guards were inherited from `go-sdk` and asserted nowhere locally. Two mutations proved the fix; a THIRD showed the Content-Type default was unreachable through the public path and therefore not a guard at all until an injected-transport test reached it. `writeMCPEnvelope` was labelled to match. ⚠ **`#602` FIXED at iteration 154 (PR #608, 2 commits; evaluator sonnet PASS 88/100 r1, zero blocking) — NOT marked LANDED: the SAME outage was still live, and `#608` had ZERO workflow runs created at all, so Gate 3b never even had an instrument. Resume = confirm #606 AND #608 green and merge both.** It was picked precisely because its investigation needs no landing gate. **It did NOT reproduce** — two full-suite arms rc=0, the first reported WITH its limitation (86 of 108 packages cached, only 22 ran), the second fully cold (108 ran, 0 cached). It was confirmed instead by measuring the race directly, and that measurement refuted BOTH halves of the issue: the stated cause is off by 3× (the budget is `max(Timeout,effective)+solverKillGrace` = **3s**, not 1s — corroborated by the isolated arm's 3.274s), and the suggested fix (poll for the pidfile) **cannot work**, because a lost race means the shell was killed before `echo $! > file` ever ran so the file never appears. 500-trial probe: idle 200/200 mean 209ms/max 232ms, 0 over; under a full `-count=1` suite, **1 trial in 300 exceeded 3s at 3.435s** = ~0.3%/spawn, ~0.7%/suite run — which is exactly why CI stays green and why iter-151 saw it once. The over-budget trial landed at load **13.6**, not the higher **24.4**: a tail effect, not a smooth function of load. Fix = treat a trial that never recorded a child PID as **inconclusive** and retry, bounded at 3 with a loud `Fatalf`. Two controller mutations, sha256-proven and reverted byte-identical: killing `pid` instead of `-pid` still reds on **every** attempt (so the retry cannot mask a real regression), and forcing the race permanently lost reds with the exhaustion message rather than passing silently. The evaluator's NON-BLOCKING finding was **acted on and was a genuine scope error of mine**: `Z3Version` uses `versionProbeTimeout` (**5s**) directly, so the two tests do NOT share a budget and 0 of 500 trials exceeded 5s. The weekly-CodeQL-cadence question was `D-7`, ANSWERED 2026-08-06 (Mark, attended): KEEP WEEKLY — no cadence change. The RELEASE ask (iter-151) was answered the same session: WAIT, World does not need a pinned release yet. (neither outranks the queue — a finding never does by itself; both were established with negative controls, not assumed). **(1) `#602`** — `internal/smt TestSolve_HardTimeout_FakeSolverIgnoringT` reds `go test ./...` on **clean dev**: a load-sensitive 3s deadline for a fake solver's child to write its pidfile, which parallel load can beat. Controlled both ways (full suite → rc=1 on unmodified dev with no concurrent load; isolation → rc=0 in both trees), so it is pre-existing and in the TEST, not the product. **A survivor of the CI-flake sprint that just closed**, green in CI only because the runners' load profile stays on the winning side — i.e. a latent CI flake, and the natural home for the fix is `internal/testutil`'s bounded helpers from `#591`. ~2-3h. **(2) `#603`** — CodeQL `go/reflected-xss` **high** at `embedded_mcp.go:119`, which is **M2's** code and byte-identical between dev and the PR head. It surfaced on PR #601 only because CodeQL analyses `dev` **weekly** and last ran ~22h BEFORE M2 landed, so that file had never been scanned. Two decisions owed: is it a true positive (the replayed `Content-Type` comes from the SDK, which would make HTML interpretation unlikely — but the headers are copied wholesale from a buffer, so confirm rather than assume), and should CodeQL run on push to `dev` so a finding is attributed to the change that caused it instead of appearing as noise on an unrelated PR. ~2-4h incl. the cadence change. **Method note worth keeping**: my first read said "0 reflected-xss on dev" from a `per_page=50` page against ≥100 open alerts — rule 3b(v)(a) truncation, caught by paginating. An enumeration you truncated is not an enumeration.
- **[PARKED needs-human-review 2026-08-07 (iter-157) — DESIGN DOC LANDED (`design_docs/planned/v0_33_1/m-named-test-body-check-semantics.md`, Planned, 21 verification rows, base `74dd06bb6`), QUORUM BLOCKED ×2 THEN BLOCKED AGAIN AFTER ITS ONE REVISION, **ONE DECISION OWED (`D-2`)**. Designer `claude:claude-fable-5` (rotation advanced to `codex:gpt-5.6-sol`). **The bug is NOT a ghost** — reproduced at HEAD with a discriminating control (bad check first → rc=0 `All tests passed!`; order swapped → rc=1), and #590's assert lowering did NOT fix it. Direction chosen and UNDISPUTED by both reviewers: option 2, every top-level bare non-final expression is a check, lowered through #590's existing sentinel path — which `test_body_lowering.go:126-131` already calls *"a one-case extension of the switch"*. **The issue's OWN suggested fix is unimplementable**: the fold is purely syntactic (AST → printed source → full pipeline), so "conjoin all *bool-valued* expressions" cannot be done at fold time — 2nd instance of *an issue's bug gets verified and its suggested fix does not* (1st: `#602`, iter-154). **R1 objections were both MEASURED by me before routing, and both UPHELD**: `gpt5-6-sol`'s trigger/uniformity objection (`test { 42 }` → `expected bool result, got *eval.IntValue`, the runner contract at `runner.go:198-211`, not a lowered-`if` type error) and `gemini-3-1-pro`'s **nested-block vector**, confirmed first-party in two shapes each with a discriminating control (`if true then { a==99; a==2 } else false` → rc=0; bare `{ a==99; a==2 }` → rc=0; false check moved LAST → rc=1 both) and **FILED AS `#614`**. Revision took (a) narrow uniformity claims to the sentinel path + (ii) scope the nested vector out honestly, deleting *"structurally impossible"* throughout. **R2 returned `proceed` rc=0 but with `gpt5-6-sol` ABSENT for `budget` — I did NOT bank that degraded pass** ("absent" is not "satisfied"); the targeted re-run at `--max-cost-usd 0.30` **BLOCKED on a NEW objection**: pinning the residual silent false-green as expected behaviour via a `nested_block_residual.ail` fixture violates no-silent-fallbacks / Machines First — *"nested checks can still fail while `ailang test` returns exit 0."* That disputes a direction choice and both rounds are spent → Standing rule 2, park, do not force through. **`D-2` IS CHEAP TO ANSWER BECAUSE THE BOUND IS MEASURED**: the doc rejects the close-it-properly options because `internal/ast` has no generic Walk/Inspect (verified: 0 hits, control **168** funcs; the walker at `internal/types/traverse` is `func Walk(t types.Type,…)` — types, not AST) and a hand-written walk would "silently re-open the gap" — but there are exactly **27** expression node types in `internal/ast/ast_expr.go`, so an exhaustive type switch with a **loud `default:`** makes that failure mode impossible by construction (the repo's own Principle 2). Options: **(A)** ship top-level-only with `#614` open; **(B)** widen the sprint to close the nested vector; **(C)** make a multi-expression block inside a test body a static error. Resume is cheap either way — the doc is written and every premise is measured. Was:] **[NEXT] [iter-152] `#604` — named test blocks check only the LAST expression; earlier failing checks are discarded** (P2, ~0.5-1d; **not** a v1.0 bar item, but it is the **vacuous-pass class** this mission has now closed four times elsewhere — the silent `z3` skip, CI `t.Skip`, `#517`, `#524`). Found while planning the `#590` fix and **reproduced first-party with a discriminating control before filing**: `{ add_one(1) == 99; add_one(1) == 2 }` reports **`All tests passed!` rc=0**, while the reverse order correctly fails — so it is position, not luck. `FoldBodyExprs` binds every non-final expression to a dead `_seq` and `EvaluateNamedTestBodyExprs` returns only the final value, which `runner.go:156-158` documents as intentional. Defensible for an effectful sequence; wrong for a **pure** test body, where a discarded `bool` is either a swallowed check or dead code. ⚠ **Deliberately scoped OUT of the `#590` sprint** and verified unchanged by it — fixing it alters the type obligation on currently-passing tests, so it must not ride in on a bug-fix-lane commit. The `#590` lowering already short-circuits correctly *within* assert-bearing bodies, so this row covers only the non-assert path. Design question to settle first: require every non-final expression to be `()`-typed (making a discarded `bool` a type error), or conjoin all `bool`-valued expressions so every check counts.
- **[LANDED 2026-08-07 (iter-158)] [ORPHAN-PR iter-150] `#545` — "fix(eval): agent cost was the wrong model's price, and the budget was the wrong unit"** (P2; rebase-and-revalidate, ~0.5–1d; **not** a v1.0 bar item). Found by iteration 149's new died-mid-flight check on its **first independent use** — the check looks for OPEN PRs authored by this loop, and this is one nobody ever picked up. Opened **2026-07-31**, last touched the same day, **CONFLICTING**, **125 commits behind** dev, with **zero** mentions in the charter (control `#532` = 2) and **zero** in the mission log (control `#544` = 4) — invisible to every existing surface, because the weekly sweep covers *issues* and the already-landed check covers *merged* PRs. **Its purpose is NOT superseded — measured at HEAD, not assumed** (the standing rule treats OPEN + long-untouched as evidence *toward* superseded, so this had to be checked): `ResolveCostModel` **0** hits, `CostProvenance`/`cost_provenance` **0**, `internal/eval_harness/cost_tally.go` and `internal/executor/codex/cost.go` **absent**, with `internal/executor/cost.go` present as the control proving the absences are measurements. So both defects are still live: agent cost banked from the executor's hardcoded table rather than per-model rates (the PR documents banked `$0.34259` vs a budget that saw `$0.26980` — two different price tables, so the kill threshold and the banked number disagreed), and `cost_usd` summing subscription list-price-equivalents together with genuinely metered spend under one label. That second one is **KPI provenance**, which this mission's own cost reporting depends on. ⚠ 47 files across `eval_harness`, `executor/*`, `observatory`, `storage/firestore` — a surface that moved a lot in those 125 commits (v0.32.0 confidence-gating, v0.33.0 recorded-stream), so this is a rebase-and-revalidate job, **not** a merge. Triage comment posted on the PR. Decide on pick-up: rebase it, or close it and re-cut the fix from the two defects it documents.
  **UNBLOCKED 2026-08-07 (iteration 158) — and the ⚠ above is REFUTED, kept here as the evidence.** The warning ("a surface that moved a lot ... a rebase-and-revalidate job, **not** a merge") is true in every number and wrong in its conclusion; three iterations deferred this item on it. Measured: `git merge origin/dev` produces **3** conflicted files out of 47, one hunk each — a changelog (union, **zero** content overlap; control **6** `Added` headings on dev's side), a **two-line** struct-literal union, and one that looks enormous (427 ours vs 1 theirs) but is a code MOVE (dev extracted the agent path into `cmd/ailang/eval_benchmark_agent.go`), whose real delta against the merge base is **four lines**. Merge commit `3efa2cd77` pushed; PR went **`CONFLICTING` → `MERGEABLE`**. Validation outside any sandbox: `make test` **rc=0 / 107 packages / 0 FAIL** (control **7070** `--- PASS` lines); `go build ./...` red only on `cmd/wasm`, which is **identically red on pristine dev** (rule 3e baseline); `check-changelog`/`check-file-sizes`/`check-boundaries`/`fmt-check`/`vet` all rc=0, gate list DERIVED from `ci.yml` per rule 3g; `models.yml` lost nothing (**111** keys both sides, `comm -23` empty, incl. the post-branch `pi-or-deepseek-v4-flash` lane); migration numbering clean (dev v16 → this adds v17); `go.sum`/`go.mod` byte-identical. The union in `agent_runner_multi.go` was JUSTIFIED, not just applied: dev's new zero-cost fallback resolves through `GlobalModelsConfig.Models[lookupKey]`, the **same** models.yml rates `#545` installs as `task.Pricing`, so it does not reintroduce the two-price-table defect. Remaining: land on its own CI green — it is mergeable on sight.
- **[LANDED 2026-08-11 (iter-177) — LANE B1 COMPLETE (M1–M4 + M6; M5 descoped); plan: [m-property-generator-coverage-lane-b1-sprint-plan.md](planned/v0_31_0/m-property-generator-coverage-lane-b1-sprint-plan.md), 6 milestones / 2.25d. **M2 LANDED 2026-08-10 (iter-169), PR #638 → squash `632024121`, Gate 3b GREEN** (`total=20`, pending 0, 0 not-green, all four REQUIRED contexts, `CLEAN`). **M3 LANDED 2026-08-10 (iter-170), PR #645 → squash `48cf25cff`, all four REQUIRED contexts green from real `pull_request` events. **M4 LANDED 2026-08-11 (iter-176), PR #653 → squash `03ab3e7de`, Gate 3b GREEN** (`checks=20`, zero not-green incl. SonarCloud, 4/4 REQUIRED from real `pull_request` events, `CLEAN`); 4 commits; executor `codex:gpt-5.6-sol` in 23 min — the first real fire of the `#611` fallback chain. Corpus: `park.ail` 6 vacuous → **0** (rc 1 → 0), `record_adt_cycle_verify.ail` 1 → **0** inside 60 s, `hof_verify`/`list_verify` unchanged (B1-13 holds); package **265 → 274** `--- PASS`. ⚠ **Three defects landed with the milestone and NONE was visible to its green suite** — the never-drop-a-constructor invariant had zero coverage (mutating it left 272/272 green); the only end-to-end pin on `ailang test` exiting 1 for a vacuous suite built its fixture from an ADT M4 derives, and **CI caught it after the controller's local sweep did not**; and `typeDefReferencesDecl` (indirect recursion through a *named* type) sat at **0.0%** coverage, i.e. deletable with the suite still green — found via SonarCloud's coverage gate, which the negative control proved was NOT the standing `#615` red. All three now pinned, each verified as the sole killer of its own mutation. **M6 LANDED 2026-08-11 (iter-177), PR #655 → squash `2cab77966`**, 2 commits / 7 files, Gate 3b GREEN (4/4 REQUIRED from real `pull_request` events, `checks=20`, zero not-green, `MERGEABLE`; non-required `govulncheck` wedged `in_progress` and NAMED — the diff touches no Go source and no `go.mod`). **M5 DESCOPED** per the plan's own §6 (F-5: derived shrinkers have no downstream observable — both contract paths discard the shrinker), filed as a follow-up. **The payoff, measured first-party by the controller BEFORE routing and re-derived independently by the evaluator from a binary rebuilt at the pre-B1 base `22ba8626d`: vacuous skips 111 → 24, i.e. 87 previously-never-executed contract properties across 15 examples now actually run**; 8 files flip rc 1 → 0; all 5 false-red guards hold rc 0. The surviving 24 are imported + refined types — **F-1 confirmed in the field: B1 does NOT fix the prompt-injection safety demos**, so the doc's original Success metric was wrong. Triage (B1-16): all 3 newly-failing properties are **(a) deliberate**, zero (b) or (c), no contract logic changed — `insurance`/`scoring` gained the `Expect:` header they lacked. ⚠ **The closing finding: those files were WRITTEN to demonstrate Z3 catching contract violations and their properties had never executed — the demos were vacuous.** Two vacuity defects caught in-milestone: the executor shipped `ensures { result == result }` for the new example's tuple property (a contract that cannot fail — strengthened, both arms measured), and its header claimed a clean `ailang verify` when Z3 cannot encode tuple patterns at all (reproduced with the tautological form too, so it predated the strengthening; now stated honestly and filed as a follow-up).** Structural derivation behind the seam: anonymous+nested records, tuples, unit `()`, same-file named record TypeDecls, `TypeAlias` recursion — depth-budgeted, ADTs honest-nil until M4; `createGeneratorForType` moved to the new `derive.go` (`runner.go` 766→722 vs the 800 gate). Carries the mandatory F-3 fix (sorted field order in `RecordGenerator`; B1-4's 200-draw byte-identity pin kills the map-range revert, sole-killer). ⚠ Controller find at review: the plan's own "preserve the list arms untouched" instruction carried a STACK OVERFLOW — fresh-root element derivation let `type Tree = { val: int, kids: [Tree] }` (reachable for the first time once named types derive) recurse unboundedly; fixed in-PR, pinned by a test measured to crash pre-fix. Mutation drill: B1-4/5/8/10/11 all KILLED; rule-3j sweep found the tuple arm's refusal branch unprotected (whole package green under neuter) — pinned with a proven killer. THREE fixture-vehicle swaps record→ADT (two executor-self-reported F-M3-1, one controller wide-sweep find in `cmd/ailang` — `make test` 2→0), each adjudicated both-arms per rule 3h. Executor pi **67 turns, `metered=$0.160`** (5th prescriptive-lane datapoint). Evaluator sonnet **95/100 PASS, zero blocking**; it REFUTED the controller's own B1-10 sub-claim (`emptyCell_property_1` passes 100 cases — the executor had been right); its NB — the depth-3 budget makes record-via-list types UNCONDITIONALLY underivable even when legitimately inhabited (`{val:1, kids:[]}`) — is FOLDED INTO M4. `valueToLiteral` returns `(ast.Expr, error)` and **refuses** instead of fabricating `()`; ensures/requires/forall each turn a refusal into `StatusFail` naming the Go type, **never a skip**. The seam landed as an unexported `Runner.genForType` bound in `NewRunnerWithConfig` — needed because the four call-site branches were unreachable at M2 **and stay unreachable after the whole lane**, so they would have shipped as decorative guards. Six controller-run mutants, each LANDED (sha256) + BUILDS + `-run`-scoped + inverse-armed + `cp`-restored: N-1 **KILLED** (also reds N-2..N-4), N-2/N-3/N-4 **KILLED sole-killer**, B1-2's own named mutation **KILLED**. **N-5 SURVIVED and is DECLARED redundant** (rule 3j's escape clause) — the executor self-reported it, its reason was *checked* not adopted (`EvaluateExpression` string-formats the AST, so a nil expr errors and the adjacent branch continues), and the implicit contract is now pinned by `TestShrinkNilExprContract`, itself proven non-vacuous. Executor `pi:deepseek-v4-flash-0731`, 51 turns, `metered=$0.086`. Evaluator **sonnet PASS 96/100 r1, zero blocking**; all three NBs reproduced first-party and closed **in-PR** (`fdec563d2`) — incl. **N-4 deserved N-5's disclosure and did not get it** (neutered, `Status` is still `StatusFail`; only `.Error`'s text moves, so that guard buys diagnostic quality, not the verdict) and a `754`→**766** line-count correction I had transcribed rather than measured. **B1-2 (round-trip through the evaluator) is PAID** — the debt iter-168 named as owed. ⚠ **M1 had been recorded as landed while PR #637 sat OPEN** — merged here as `59b74e06d` after re-verifying its three-dot diff (the two-dot diff is a stale-base artifact and reads as a mass revert; do not trust it). **Next: M3.**] [world-DEMAND] m-property-generator-coverage LANE B1** — **M1 done**: `internal/testing/value_splice.go` splices Unit/Record/Tuple/Tagged values, inert by construction, `runner.go` 749→710. Evaluator sonnet 82/100. Three mutation pins, each proven sole killer. **Three planner refutations changed the sprint**: there are THREE generator call sites (`contract_domain.go:74` is missing from the design doc's file list), `RecordGenerator.Generate` ranges a Go map so a fixed seed does NOT reproduce a counterexample (refutes the doc's own A1=0 score — mandatory M3 fix), and **B1 fixes zero properties in either prompt-injection demo** (`inbox_v2_app.ail` imports its `Mail` type; `inbox_injection_v2.ail` is all refined `string<email>`), so the doc's "five silent-shape files" success metric is false for two of five. **M2 NEEDS A TEST SEAM before it can be routed** — its N-2…N-5 refusal branches are unreachable at M2 and stay unreachable after the whole lane, so they would ship as decorative guards (controller correction in the plan; the evaluator narrowed it: N-5 IS reachable today, the other three are not). **Owed**: acceptance criterion B1-2 (round-trip through the evaluator) is M1-scoped and unmet. **Next: M2 (with the seam) → M3.** Was: (design doc EXISTS + quorum-cleared for B1 — no new doc, no re-quorum needed; P2 — not a v1.0 bar item; ~1.5–2d) — structural generator derivation: product-of-fields for records, sum-over-constructors for ADTs, tuples, with recursion depth/size bounds and shrinking. This is the REAL fix for the shapes Lane A can only make LOUD: after Lane A, 6 in-repo contract files still carry vacuous skips (records, ADTs, `string<email>`, `list[Tree]`), and World's five `w-mcp-projection` properties over its core types still run zero cases. **Two hard constraints from iter-121, both measured**: (1) `runner.go` is at **790/800** lines, so B1's additions MUST land in a new file in the same package — the CI gate has 10 lines of headroom (evaluator NB-3); (2) the `valueToLiteral` arm list MUST add `UnitValue → ast.Literal{Kind: ast.UnitLit}` BEFORE the default arm becomes a loud error, else the `()` generator fails the harness (quorum R2, `gemini-3-1-pro`, now recorded in the doc with acceptance row B-4a). **B2 (user-supplied `gen<TypeName>` escape hatch) stays DEFERRED** — blocked on a deterministic evaluator fuel/step budget, per `gpt5-6-sol`'s own proposed option; do not fold it back in without that budget. Expect Lane B1 to surface GENUINE contract violations as previously-vacuous properties start running (Lane A already exposed two in `list_verify.ail`) — budget triage time rather than treating them as regressions. ⚠ **Sequencing: `#535` (wall-clock seeding) should land FIRST or alongside**, because B1 multiplies the number of properties actually executing and every one of them inherits a non-reproducible verdict. ⚠ **UPDATE 2026-07-31 (iter-126): B1's `runner.go` 790/800 constraint is RELIEVED — it is now 670**, because M1 below moved `runEnsuresProperty` into a new `internal/testing/contract_domain.go`. B1 additions still belong in a new file, but the gate is no longer 10 lines from the cap.
- **[IN-SPRINT 2026-08-10 — `#618`: **M1 LANDED (iter-172)** PR `#647` → squash `752f997d1`, Gate 3b GREEN (5/5 workflows, all four REQUIRED contexts from real `pull_request` events, `CLEAN`), evaluator sonnet **95/100 PASS zero blocking** — 872 lines, 2 new files, **zero call sites** so runtime behaviour is unchanged by construction; package `--- PASS` 23→30. ⚠ **The pi executor lane failed TWICE with `rc=0` and zero files** — both runs ended on a ~63k-char `thinking` block hitting the 16,384-token output cap (`stopReason=length`), which pi treats as a normal terminal state; fell back to **opus, FLAGGED**. Deviation adjudicated ACCEPTABLE in both arms: the plan put the hard deadline in M2, leaving AC-M1.3's third sentinel unreachable, so the executor added an in-reader budget — neutering it reds ONLY the deadline case, and with it neutered + that test skipped the package is rc=0 (strictly additive). **CARRY INTO M2**: `streamBaseTransport()`'s `sync.Once` FREEZES `ResponseHeaderTimeout` at first use (reproduced first-party — froze at 1m51s while the resolver tracked 999s, control firing), so any M2 test varying `AILANG_OLLAMA_TTFT_TIMEOUT_SEC` per case reads stale unless it runs first or resets the `sync.Once` — it passes GREEN while measuring nothing. **Rule 3b(vii)**: the design doc's Implementation-Plan section is STALE on milestone numbering — its M3 is the plan's M4 and it mentions `M4` **zero** times against the plan's 23 (control: doc `M1`=3); plan wins; the renumber is cleanup owned by M4 (docs). **UPDATE 2026-08-11 (iter-173): M2 LANDED and the R1 inert-deadline fix is CLOSED** — PR `#648` → squash `ff1fa0760`, Gate 3b GREEN with completeness asserted (4/4 REQUIRED contexts from real `pull_request` events, `mergeable=MERGEABLE state=CLEAN`, SHA-addressed `checks=20`, zero not-green). `step.go` **+15/−0**, new `streamstep.go` (318) + `streambranch_test.go` (670) + a changelog entry to `changelogs/v0.18-current.md` (M1 shipped none by design, so M2 owed it); `step_test.go` and `idlereader.go` sha256-**identical** to HEAD; package **30 → 38** `--- PASS`, one per AC. Evaluator **sonnet 97/100 PASS, zero blocking**. **The milestone is two lines**: the streaming branch derives from the context captured ABOVE the 300s wrap, and **AC-M2.4 is the sole killer** — move the capture below it (mutant LANDED by sha256, BUILDS rc=0) and the read-back reads `299.999963583s` against a configured `3600s`; `-skip` that one test and the package is rc=0. ⚠ My FIRST C1 mutant reds on `outerCtx declared and not used` — a build failure arriving in exactly the predicted direction, the one class rule 3d cannot catch; re-cut as a capture-POSITION move before the kill counted. ⚠ **RULING E-3 IS RETRACTED — the executor refuted it and the measurement agrees**: precedence rule (b) (`streamstep.go:273-275`) maps ANY `streamCtx` `DeadlineExceeded` to `ErrStreamDeadlineExceeded` regardless of provenance, and both clocks read the same env var, so the error TYPE cannot discriminate the R1 defect even in principle (under the defect, `TestStreamBranch_HardDeadlineBeatsKeepAlive` still passes rc=0). Read-back arm only. Second deviation adjudicated in BOTH arms: `idleReaderConfig.Hard` stays **0** in production — with only `streamCtx` armed, neutering it reds AC-M2.2 AND AC-M2.4; with BOTH armed the same neuter leaves AC-M2.2 GREEN, so a redundant guard would make neither timer accountable (M1's `Hard` is consequently exercised only by M1's own tests). The `sync.Once` freeze carried in from iter-172 was **REFUTED as a vacuity risk** by the evaluator (`stepV1Stream` reads `ollamaTTFTTimeout()` fresh per call, bypassing the frozen shared transport) — but it is why the executor used the `base` injection seam, so the warning earned its place. ⚠ **The pi executor failed a THIRD time**, pre-registered shape, under a MORE prescriptive directive: `stopReason=length`, rc=0, clean worktree, after 10 healthy tool calls — and the runaway block was **`text`** not `thinking`, so the tell is the stop reason, never the type; `turn_start=7/turn_end=6/agent_end=0`; NDJSON grew ~3 MB/s to **1.2 GB in six minutes**. Fell back to opus, FLAGGED; **skill edit landed** (three mandatory post-run assertions + an NDJSON size ceiling). **NEXT = M3** (parity: `Reasoning` + Hermes tool-call recovery — the disengagement regression; rulings E-1/E-2 are the seams, and M2 left `onChunk` wired but counting only), then M4 (rig, **GPU**) which also owns the doc renumber. Routed 2026-08-10 (iter-171): doc + fixture `3e1f63f7a`, plan + state JSON + controller rulings `eecb4d011`. Quorum: designer `claude:claude-fable-5` (3 bounded passes), R1 BLOCKED (`gpt5-6-sol` reject — no finite total bound; measured first-party rather than forwarded, CORRECT and understated: `ParseChatStepSSEStream` has ZERO time/deadline/limit bounds, control `scanner`=5 — reviewer's `proposed_fix` applied VERBATIM), R2 BLOCKED on two premise/completeness objections → **narrow-refinement carve-out**, both measured not forwarded: the `eval_publish.go` validity claim verified TRUE (0 hits, control `PassRate`=6), and the FEASIBILITY premise settled by a **controller rig probe under the GPU lock** — ollama 0.32.1 `/v1` DOES stream tools on `qwen3.6:35b-a3b-mxfp8`: 52 `data:` events, tool call complete in ONE chunk with `"index":0`, `finish_reason:"tool_calls"`, usage 294/80/374, one `[DONE]`, ZERO `:`-keep-alives (control: 52 blank separators). **The 13,077-B capture is committed as `internal/ai/ollama/testdata/ollama_v1_stream_toolcall.sse`, so M3's capture task was done before the sprint began.** ⚠ **PLANNER opus REFUTED THREE PREMISES AND THE FIRST IS LOAD-BEARING: `ollamaCallContext(ctx)` — `context.WithTimeout` at the SAME 300s default — wraps at `step.go:266`, SEVENTEEN LINES ABOVE the `/v1` branch at `:283`, so `Client.Timeout: 0` is necessary and NOT sufficient; with M4 removing the 1800s plist pin both clocks fall to defaults and the effective bound is 300s — the feature would have SHIPPED INERT.** Three designer passes and four quorum reviews all missed it. R2: `streamstep.go` sets `out.Reasoning` 0 times (control 6) and has 0 Hermes refs (control 8), so streaming motoko's path without parity work trades a timeout bug for the **disengagement** failure mode → new M3, 1–2d → **3d**. R3: goleak in `go.sum` (2) not `go.mod` (0). Rulings E-1..E-4 in plan §10. Remaining batch rank unchanged: `#619` → `#616` → `#617`. **UPDATE 2026-08-11 (iter-175): M4 LANDED — the sprint's REPO work is COMPLETE (M1 `752f997d1`, M2 `ff1fa0760`, M3 `86f7f1c32`, M4 PR `#652` → squash `08eef7760`).** Gate 3b GREEN with completeness asserted on the PR head: 4/4 REQUIRED contexts (`build`/`docs-gate`/`lint`/`test`) from real `pull_request` events, `checks=20` zero not-green, `mergeable=MERGEABLE state=CLEAN`. 8 files `+467/−39`, exactly one Go file and it is a `_test.go`. Evaluator **sonnet 93/100 PASS, zero blocking**; both NBs reproduced first-party and fixed in-PR, one of them UNDERSTATED by the judge. **AC-M4.3 WAS NECESSARY BUT NOT SUFFICIENT AND ITS OWN GREP WOULD HAVE CERTIFIED THE HAZARD AWAY**: the stopgap has a SECOND delivery site — `launchctl getenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC` = **1800**, a launchd user-domain global no plist edit touches, named in `b67d415cd`'s own body (*"also set live via launchctl setenv"*) while the plan's §2.4 measured only the plists. **And the tidy order is the harmful one**: installed plists are regular files byte-identical to HEAD, so the flag is OFF everywhere installed and clearing the global FIRST drops uncovered calls to the 300s default (`step.go:24`), reintroducing this very issue at 895 retries / ~74.6 GPU-h. AC-M4.3 now carries an ORDERED rollout (install+load, THEN `unsetenv`), marked rig-state work OUTSIDE the repo deliverable; the rig was left untouched. **AC-M4.2 FIELD-VALIDATED, VERDICT PARTIAL**: n=**98** streamed requests on `docx_reimplement` × `motoko-local-qwen3.6-35b` flag-ON — `effective_deadline_sec` == configured on ALL 98 (REFUTATION #1 closed in the field), **zero requests in the 299–301s band** against the 47-in-one-day pre-fix signature, zero idle/TTFT trips, `saw_first_byte` true throughout; headroom TTFT 5.6× / idle **3.8× (tightest freeze margin)** / total 4.7×. **But NO X/17 grade** — `step budget exhausted` after 60 steps (`error_category=step_exhausted`, chain `6bb6e5ed`), an agent-convergence outcome in the docx compile-stuck class. **So E-4's precondition is NOT discharged** and the rig rollout is parked as `D-8`. Register: R10 and R11 are both non-unique (no `openai/streamstep.go` mutation can be unique — AC-M4.1's object is the captured WIRE SHAPE); **R12b** (fixture line 97) is the sole killer, arm B rc=0. ⚠ A mutant can LAND (sha256) and BUILD and still not land ON THE MECHANISM — the first R12 hit thinking text, not the tool args, and the test correctly stayed green. ⚠ The field capture was CONTAMINATED by the executor's concurrent `go test` via the `AILANG_OLLAMA_LOG_REQUESTS` HOME sentinel; the first partition would have banked AC-M4.2 as RED, and only the records' own `idle_window_sec`/`ttft_window_sec` separated them. **NEXT for this batch: `#619` → `#616` → `#617`, after Lane B1 M4. **UPDATE 2026-08-11 (iter-178): `#619` PICKED AND PARKED `needs-human-review` AT THE QUORUM GATE — no code, no sprint, no PR.** Its doc (`planned/m-eval-validity-discipline.md`) had no quorum artifact, so quorum-at-pick fired: **2 rounds, both BLOCKED, both reviewers present** (`absent_reviewers: []`), metered `$0.0955`; artifacts `m-eval-validity-discipline-2026-08-11T17-46-03Z.json` / `…T17-49-21Z.json`. Every objection was measured not forwarded and every `proposed_fix` adopted VERBATIM, but round 2's surviving `gpt5-6-sol` objection disputes the design **direction** of the newly-recorded **W9**, so the narrow-refinement carve-out does not apply and Standing rule 2 binds. Structural cause: quorum reviews the whole 5-work-item umbrella while only **W8** was being routed, and both rounds' blockers landed on content W8 does not touch — **`D-9` on `#635`** asks Mark the one-word question (split W8 into its own scoped doc and re-quorum, or hold). **The Gate-2 reality-check is banked regardless and changes the sprint**: the defect site is **`SummarizeRotation`** (`internal/eval_harness/rotation_summary.go:246`, plus a second unguarded numerator at `ModelRollupStats.PassAt1`), **NOT** `cmd/ailang/eval_publish.go` which only sums rotation summaries; the `--skip-existing` third of the scope is **already closed** by `f3189541a`; `eval_analysis.FilterValidResults` **cannot** be reused there (`go list -deps`: `eval_analysis → eval_harness` 2, reverse 0, control 25 — an import cycle); the fix must follow the `TokensCacheUnaccounted` idiom; **160** invalid rows sit in the live bank (control firing); and the published board carries **no denominator field at all**, so surfacing the excluded count is a schema change. **New `W9` recorded** (measured): the coverage gate compares benchmark COUNTS, never set identity — `coverageGate.js` reads 0 benchmark IDs (controls 6/13) while `ratings_export.go:117` exports `len()` of a set it holds, so equal-count disjoint-set models rank as comparable; **AC1 of that doc is consequently marked NOT SATISFIED** rather than left claimed. Batch order while `D-9` is open: **`#616` → `#617`**. **UPDATE 2026-08-12 (iter-180): `#616` REVISED AND PARKED `needs-human-review` AT ROUND 2 — `D-10`.** Doc revised to 1,028 lines (designer `codex:gpt-5.6-sol`, rotation; chose **A3**, R1's own `proposed_fix`; V31–V39 added). **Round 2 BLOCKED, both reviewers present** (`absent_reviewers: []`), metered `$0.1365`, artifact `m-effect-row-var-unification-2026-08-11T23-31-11Z.json`; per-reviewer cap raised to `$0.25` PRE-EMPTIVELY because the doc grew 845→1,028 lines against round 1's `$0.0778` on a `$0.10` default (the iter-175 trap fires by arithmetic — at the default this round would have lost the very reviewer under repair; it billed `$0.0958`). **RULE 3f EXECUTED: I BUILT AND RAN R1's PROBE.** Occurrence entries are PRESENT, occurrence-shaped and **UNINSTANTIATED** — arm b stores `() -> int -> int ! {...ρ3}`, `EffectRow RAW={labels=[] tail=ρ3}` (V31), so the doc's expected "V2 resolves to a closed empty row" is FALSE. The correct information lives in the **argument**: arm l's two occurrences carry `param[0]` rows `{labels=[] tail=nil}` (appID=21) vs `{labels=[IO] tail=nil}` (appID=25) while BOTH result rows are unsolved fresh vars `ρ8`/`ρ9` (V32). **Hence `e` in the parameter and `e` in the return are NOT the same variable** (V33) — a shared var would have been solved by the unification that concretised the param; CONTROL: the same shape with a concrete row stores `() -> int ! {IO} -> int ! {IO}`, rc=0 (V34), so storage/zonking is sound and the failure is row-VARIABLE-specific. That refutes iteration 179's own premise that the type layer needs no change. R2 confirmed at an exact line: `UnionEffectRows` has **0** `Tail` occurrences in its body, control `grep -c Tail` in that file = **3** (`:359`, `:467`, `:614`), the `:614` hit being its own `Tail: nil` return; fix site `internal/types/effects.go:606-616` (V37). Caller census re-derived first-party: 6 hits = 1 def + 3 tests + **2** production callers (`validate_effects.go:541`, `validate_effects_rows.go:81`), control `SubsumeEffectRows(`=**14** (V39). **THE FIX SITE MOVED A FOURTH TIME, AND IT REFUTES R1's REPLACEMENT TOO.** R1 asks to allocate one effect-row metavariable per source row-var name and reuse it at every occurrence — **that already exists**: `Scheme.InstantiateWithConstraints` (`internal/types/types_v2.go:533-561`) keys `subs` by NAME and applies one `Substitute`; `TFunc2.Substitute` (`:383-401`) applies it to `Params`, `EffectRow` AND `Return`. The real disconnect: at an application inference mints an INDEPENDENT `resultEffects := ctx.freshEffectRow()` (`internal/types/inference.go:203-215`) and leaves row unification to tie it to the callee's row, which does not survive a label-empty-plus-tail row — corroborated by the repo's own comments in 4 places (`validate_effects.go:100,136,159`, `typechecker_core.go:78`). So **both** architectures the doc has considered are aimed at the wrong layer: A2's sharing exists, A3 would publish an unsolved var. Site history: parser → types → effect-checking pass → App-site publication → **row unification at the App constraint**. **INSTRUMENT HAZARD (V38): `ailang check` caches by content, and it only hides the arms that PASS** — a second run on an unchanged passing file printed **0** probe lines while the failing known-positive control printed **10**; one appended newline made it **25**. The soundness arms are exactly the passing ones, so a row taken by re-running an unchanged passing file is uninformative. Now a Testing-Strategy constraint, enforced in the ACs. **CONTROLLER RESTORED THE NO-REGRESSION GATE**: the revision cut 14 ACs to 10 and every one it kept is a MECHANISM pin — old AC4–AC9 (arms k, a, d+e, n, m, h/i), AC12 (runnable stdlib example), AC13 (71b610d68 contamination class) and AC14 (docs) had no counterpart, while this design's dominant risk is OVER-rejection. Restored as **AC11**/**AC12** with AC9 made to name AC11; recorded in the doc's Round-2 section, nothing removed, no objection overridden. Final **12** ACs. **`D-10` (one word)**: (A) authorize a third revision re-architected around effect-row unification at the App constraint, accepting the widened Conflict Surface into `internal/types`; (B) hold `#616` and route the next item. Both objections' measurements are banked either way. ⚠ Designer/reviewer **provider collision** FLAGGED (rotation put the designer on R1's provider) — instance 1, in policy, but it weakens the independence of the objection under repair. Batch order while `D-9` AND `D-10` are open: next unblocked item is **`#617`**.**]** `#618` ollama 300s cap < a median qwen3.6 turn — 80 runs lost / ~74.6 GPU-h over 43 days, accelerating; the live cause of the 08-10 eval noise (34/40 failures `non_agentic`); const 300 confirmed at `internal/ai/ollama/step.go:24`; design doc EXISTS: [m-ollama-v1-streaming-idle-timeout](planned/m-ollama-v1-streaming-idle-timeout.md). `#619` eval publisher counts `validity.valid=false` harness errors as capability failures (~4× live understatement on the v0.33.0 OS board; `eval_publish.go` has ZERO `validity` reads, control firing; W8 P0 in [m-eval-validity-discipline](planned/m-eval-validity-discipline.md); Critical Principle 2). `#616` effect-row variables parse but never unify — REPRODUCED: `! {e}` becomes a phantom concrete effect with a blank error whose Suggested fix is byte-identical to Current signature; uppercase control correctly rejects; NEW-DOC needed. `#617` strict-eval `take(n, flatMap(f, xs))` cannot bound peak memory (2 host OOMs in prod `ailang_parse`); ~~sharpening from triage: stdlib has NO `flatMap` (grep 0, control firing) so the class is user-written eager flatMaps~~ **← RETRACTED at iteration 181: that grep ran against `stdlib/`, a directory that has never existed here (the path is `std/`), and its "control" was scoped to a DIFFERENT path so it fired on the pattern rather than on the scope. `std/list.ail:202` exports `flatMap`, `:250` `flatMapE`, `:99` `take` (control `^export`=38) — BOTH halves of the trap are the stdlib's own exported, taught surface, which moves the lane from docs/lint to stdlib/builtin. The false row stood for 11 iterations; rule 3a gained `(i-d)` for it.** design-first — budgeted traversal helpers + LIMITATIONS/prompt entry. A sweep row never outranks; its ordering vs Lane B1 M4–M6 is the controller's normal call at next pick. **[DOC LANDED + QUORUM-RESOLVED 2026-08-12 (iter-181) — `planned/m-take-flatmap-peak-memory.md`, 878 lines, 8 ACs. Repro confirmed at HEAD with mechanism attribution (cap fixed at 5, cost tracks the SOURCE: 5.84 s/205 MB at n=50 → 78.57 s/397 MB at n=200 vs 0.04 s/84 MB budgeted-walk, floor 46 MB, outputs byte-identical). THE DISCOVERY: issue resolution (1) — a fused `takeFlatMap` — ALREADY SHIPPED in v0.10.0 as `d41e43894` (M-EVAL-BOUNDED-PIPELINE), motivated by the SAME DocParse Moby Dick OOM, and has been unreachable since: never exported (`takeFlatMap|takeMap` in `std/list.ail`=0, control 38 → `IMP010`), and registered as `TCon{"Int"}` at `list_bounded.go:68,150` against the surface `int` — confirmed behaviourally, a literal cap checks clean while an annotated `n: int` fails `cannot unify type constructors: Int vs int`, so the ONE shape a stdlib export needs has never worked and the shipped unit tests pass only on literals. Recommendation is expose + repair + teach, not build. Quorum 2 rounds, both BLOCKED, both reviewers present both rounds, metered `$0.2200`; all four objections MEASURED not forwarded and all four true. `takeFlatMap` does NOT bound peak by `n` (`O(source held) + O(largest single f(x)) + O(n)`, three arms). `take(n, map(f, xs))` amplifies identically once `f` allocates — fused 0.08 s/101 MB vs unfused 18.78 s/559 MB, 5.5×/235× — refuting the doc's own V7 and reversing R2's round-1 cut of `takeMap`. Round 3 closed under the narrow-refinement carve-out (both objections = AC instrument + premise attribution, reviewer text applied verbatim). NEXT: route to sprint-planner.]**
- **[LANDED-M1 2026-07-31 (iter-126) — PR #549 → squash `a9e26ffd6` (commits `940d1108e` + `3ebd4d19a`); doc `01c36db8d`, plan `7df443e25`. **Gate 3b GREEN SHA-addressed**: all FOUR required checks pass (`test`/`lint`/`build`/`docs-gate`), 13 success + 5 skipped/N-A, 0 required failures. ⚠ Non-required **SonarCloud RED**, deliberately not hidden: 77.9% coverage on new code (gate ≥80%) and 4.6% duplication (gate ≤3%). Required contexts on dev are exactly `["test","lint","build","docs-gate"]` (verified via branch protection), so UNSTABLE ≠ BLOCKED — but the duplication is the price of splitting the ensures path out of `runner.go` and should be revisited in M2. Designer `codex:gpt-5.6-sol` (rotation advanced from claude), planner opus, executor `codex:gpt-5.6-sol`, evaluator sonnet **PASS 95/100 r1, zero blocking** (8 independent mutations, 7 killed). M2/M3 PARKED to the 2026-08-03 re-arm. ⚠ **UPDATE 2026-08-01 (iter-127): M2 IS NO LONGER BLOCKED.** The AC9 fixture parse failure that §5.3 required be resolved before M2 starts was a symptom of `#548` (declaration-aware strip, landed `f64659b12`), NOT of the contract form. AC9's ORIGINAL module-less fixture now passes unmodified — paired control: pre-fix rc=1 `PAR_NO_PREFIX_PARSE ... ensures`, post-fix rc=0 — so **keep AC9 as written**; no substitute fixture and no restatement over a module-bearing file are needed. The plan's §0.3/§5.3 and risk-row B3 are updated in place, and `internal/testing/testdata/strip/moduleless_contract.ail` is the committed regression guard. **UPDATE 2026-08-07 (iter-161): M2A LANDED** — `7be6a2b8a` + platform fix `ac306084b`, `internal/testing/config.go` (+98) and `config_test.go` (+191, rows S1..S9). Internal only: zero call sites outside the two files, so the binary is unchanged. Evaluator **sonnet PASS 97/100 r1, zero blocking**. Provenance: the implementation was produced by iteration 160's `pi` executor and left **uncommitted** when that iteration died; recovered under the died-mid-flight rule, and **2 of the 9 tests were RED as delivered** — both TEST defects (`config.go` matches plan §5.3 byte-for-byte). **§5.6's S6 row is UNREALIZABLE as written** and was repaired in place: `filepath.Rel` between two absolute POSIX paths never errors, so the row now pins a RELATIVE workspace root, and the out-of-workspace case is asserted as an explicit NON-error (a `"../"` identity still satisfies AC9). A THIRD row, `TestTestConfig_Validate`, was **vacuous on Windows** as delivered — `filepath.IsAbs("/w")` is false there, so its bogus-`SeedMode` case was rejected on the root before reaching the mode switch; all path literals now come from `t.TempDir()`. **UPDATE 2026-08-07 (iter-162): M2B LANDED** — PR **#620** → squash `bd5f74362`, Gate 3b GREEN with completeness asserted (`present==expected==4`, `total=20` checks, **zero** non-green including SonarCloud `success`). All three RNG sites swept, `newRNG`'s wall clock deleted. End-to-end with a negative control on `list_recursive_verify.ail` (`duration` stripped): pre-M2B **5 distinct output hashes / 5 runs**, post-M2B **1**. Evaluator **sonnet PASS 91/100 r1, zero blocking**. **The headline is a defect in the PLAN, not the code**: §5.6's S11 row claimed to kill "guarding two sites and missing `contract_domain.go:89`" but observes `PropertyResult.Seed`, which every path stamps **alongside** the RNG rather than **by** it — so a constant-seed mutant at the ensures site (LANDED + BUILDS asserted) left the whole suite **green**, and two of the three swept sites were unguarded. S11b (`TestRunner_DerivedSeedDrivesSampleStreams`) closes it by observing the generator stream (ensures accept/discard counts, requires counterexample text); both mutants now red. **The forall site has no stream observable in this package** — every forall property in a unit-test fixture dies on its first generated input with `evaluation failed: empty program`, which the evaluator independently failed to refute across four fixture shapes and the real CLI — so it is pinned by AC arm (c) and the stamp only; **M3 owes it a CLI-level arm**. AC-SEED-SWEEP-M2 arms (b)/(c) were also repaired (their glob counted S10's own `newRNG` calls: **8**/**4** as written on a correct tree, **3**/**3** production-scoped). **M2C is the resume point** (§5.5 — `--seed`/`--random-seed`, both `cmd/ailang/test.go` aggregates, JSON+human reporting, S15–S16), and it carries one inherited task: replace the swallowed `filepath.Abs` fallback in `NewRunner`/`RunTestsFromFile` (CLAUDE.md §2 — traced inert today, but M2C builds on those wrappers).]** m-property-test-trust (`#535` + **`#547`**, doc → [planned/v0_31_0/m-property-seed-determinism.md](planned/v0_31_0/m-property-seed-determinism.md) + its `-sprint-plan.md`; P0 prerequisite for LANE B1 above) — picked because `#546` is PARKED on Mark's a/b/c call and both quota offloads are date-gated to 08-03, leaving LANE B1 as the live queue head; its own row names `#535` as the thing that must land first. **The pick was `#535`; the FIND was `#547`.** Reality-checking `#535` (reproduced 5/5, exit codes 1,0,0,1,1 on unchanged input) surfaced a larger, previously unknown defect: `runEnsuresProperty` evaluated a function's postcondition **without ever evaluating its `requires` precondition**, so it reported `ensures violated` for inputs the contract EXCLUDES — a **vacuous FAILURE**, mirror of the vacuous-pass class `#517` Lane A closed. Decisive argument = the **asymmetry**: `runRequiresProperty` meets the identical condition and reports `skip`, its own comment saying such inputs "aren't a function bug". Minimal repro 6/6: `requires { x > 100 }`, body `x > 100`, every reported counterexample ≤ 100. This **corrected the read of `list_recursive_verify.ail`** — its ~50% failure is a FALSE POSITIVE, not a genuine violation (the designer reached that independently before being told). **M1 shipped the discard filter** (100 accepted / 1000 attempts; cap exhaustion → `skip:out_of_contract` + `unverified:`, never pass never fail); `#535` stays OPEN by design as M2. **Quorum BLOCKED ×2, direction never contested**, resolved under the narrow-refinement carve-out with both reviewers' verbatim fixes; **both rounds blocked on the SAME root cause — a codebase premise asserted rather than measured** (R1 metadata enumeration; R2 `TestConfig.WorkspaceRoot`, which the controller measured as NOT EXISTING). The controller's own check of R1 came back **better than the objection assumed and SHRANK scope**: repeated `requires` blocks are impossible by construction (`PAR_DUPLICATE_REQUIRES`). **Planner refuted 3 doc premises**, incl. that M1's ACs were written against a `--seed 42` flag M1 never adds — **none of them could have run**. **Two further defects filed**: `#547` and `#548` (named test + contract in one file breaks the named-test path via orphaned `requires`/`ensures` in the generated temp module — 0 in-repo instances, a latent user trap). ⚠ **Behaviour change**: `list_recursive_verify.ail` goes flaky `1,0,0,1,1` → stable `0 pass / 0 fail / 6 skip`; `extractBounded` is now honestly *unverified* rather than luckily passing. **No CI impact — `make verify-examples` runs `ailang run`, never `ailang test`** (controller-verified, and it is why the seed pin is low-risk). **A surviving mutation is recorded, not buried**: relaxing the guard to accept 99 as a pass SURVIVED; the test claiming to pin that boundary was renamed `TestEnsuresNinetyNineAcceptedIsNotAPass` → `TestEnsuresSparseDomainIsSkipNotPass`, because acceptance at `x > 900` is ~5% so `TestsRun` never reaches 99 — pinning it genuinely requires `#535`/M2. The evaluator's NON-BLOCKING F1 was **acted on anyway** (a severity label is an opinion, not a measurement): the negative control only rejected negatives and `0`, so `x=50` would have passed an assertion whose own message claimed to prove `x > 100`; strengthened, and **proven to catch** the evaluator's mutation 8 (`x=-612`) which it previously missed. **M2C LANDED 2026-08-08 (iter-163 built it, iter-164 verified and merged it) — PR `#621` → squash `8d9c58780`.** `--seed N` / `--random-seed` (presence via `flag.Visit`, so `--seed 0` is explicit not unset), `SetSeedMetadata` at **both** `cmd/ailang/test.go` aggregates, `seed_mode`/`seed`/`seed_derivation` + per-property seed + failure-only `replay`, seeds as decimal strings (a random master is a full int64 and loses precision as a JSON float), and the two swallowed `filepath.Abs` errors deleted. Iteration 163 opened the PR green and **died before Gate 3b, writing no charter row and no log entry**; iteration 164 verified rather than redid it. Gate 3b GREEN SHA-addressed on the PR: `total=20`, **0 not-green**, all four required contexts from real `pull_request` events, `mergeStateStatus=CLEAN`. S17 re-verified first-party at **both** call sites (mutants LANDED via sha256, BUILDS rc=0, S17 red each time with the mode-specific message; source restored byte-identical), which is the M2B failure mode closed. ⚠ **But D2's mutual-exclusion refusal shipped with no gate**: covered only by `AC6(d)`'s one-shot `conflict.err` grep — no make target, no CI job, no `*_test.go` — and `rc == 2` is vacuous by §5.11's own baseline (`flag.ExitOnError` exits 2 on the then-unknown `-seed`), so the stderr substring is the only discriminating arm. `if false && seedSet && randomSet` left the entire rest of `cmd/ailang` **rc=0**. Closed by **S18** (`TestSeedAndRandomSeedAreMutuallyExclusive`, exit-2 + substring + a solo-flag control arm), the sole killer of that mutant, and generalised into skill rule **3j**. **UPDATE 2026-08-08 (iter-166): M3 LANDED and `#535` is CLOSED** — PR **#625** → squash `2ab7b3d31`, **Gate 3b GREEN with completeness asserted** (`total=20`, pending **0**, failed **0**; all four REQUIRED contexts `test`/`lint`/`build`/`docs-gate` pass; `mergeStateStatus=CLEAN`). Evaluator **sonnet PASS 95/100 r1, zero blocking**. **The headline is a defect in the thing the sprint exists to deliver**: `reporter.go:113`/`:265` built the replay command from `SuiteResult.ModulePath`, which on the ordinary CLI path is the aggregate *display label* `"All Tests"` (`cmd/ailang/test.go:49`) — so every failing property printed `ailang test --seed 0 All Tests`, which is not a path and cannot run. Unrunnable since M2C, through a quorum, a plan, an evaluator PASS and a Gate-3b green, because `AC6-M2` **reconstructs** `--seed=${seed} "$tmp/multi.ail"` from `.seed` instead of **executing** `.properties[0].replay`. Fixed with ONE `replayCommand(result)` helper behind both reporter sites, fed by a shell-safe `TestConfig.ReplayTarget`/`SuiteResult.ReplayTarget`; pinned by a test that executes the emitted string verbatim. Generalised into skill rule **3k**. Three mutation arms, all LANDED + BUILDS + restored byte-identical, each with the rest of `cmd/ailang` at rc=0 so the new test is the sole killer: **M-a** (reporter ignores `ReplayTarget`), **M-b** (constant ensures seed), **M-c** (drop the `--package` prefix — the evaluator's own find, reproduced first-party, and closed in-PR rather than fast-followed because it is the same class as the headline). **Three executor deviations, all self-reported, all adjudicated CORRECT by command** — including the load-bearing one: the directive's byte-identity assertion cannot kill a constant-seed mutant, so the executor added a two-distinct-masters SAMPLE comparison, and under M-b that addition is the ONLY assertion that fires (confirmed independently by the evaluator, which extracted the directive's version and watched the mutant survive). **The forall CLI arm is RETRACTED, not paid**: measured first-party (`generated_inputs: 1`, `evaluation failed: empty program`, identical under `--seed 42`) and un-refuted by the evaluator across four further fixture shapes — tracked in **`#624`**, which iteration 165 filed independently while scoping the same arm. Executor `pi:deepseek-v4-flash-0731`, `metered=$0.2530`. **Sprint COMPLETE — Lane B1 unblocks.**
- **[LANDED 2026-08-01 (iter-127) — PR #550 → squash `f64659b12`. **Gate 3b GREEN SHA-addressed**: 20 checks, 0 pending, **0 failures**; all four REQUIRED contexts (`test`/`lint`/`build`/`docs-gate`) success and SonarCloud green (unlike iter-126's red). Planner opus, executor `codex:gpt-5.6-sol`, evaluator sonnet **PASS 81/100 r1, zero blocking**; designer NOT fired (bug-fix lane, no new doc, no quorum — the `#524` precedent), so the rotation did NOT advance. `metered=$0.00`.]** m-strip-decl-aware (`#548`, CLOSED; plan → [planned/v0_31_0/m-strip-contract-awareness-sprint-plan.md](planned/v0_31_0/m-strip-contract-awareness-sprint-plan.md); P1 — unblocks the seed sprint's M2 above) — filed by iter-126, never routed. `stripNonPureFunctions` deleted exactly ONE LINE of a declaration that may span many, and treated any function not written with the `pure` keyword as effectful, so the temp modules `ailang test` generates were corrupted and the error was reported against a `_namedtest_body_*.ail` the user never wrote. **The controller's first diagnosis was too narrow and the opus planner REFUTED it** (re-verified first-party): the defect is **not contract-specific** — a plain multi-line function with no contract at all corrupts identically (`unexpected token: }`), and `@verify` annotations sit ABOVE `Span.Start` so even a span-only strip misses them. It also refuted `#548`'s own known-positive control as passing for the wrong reason (its test body calls nothing). **The obvious fix was rejected on MEASURED blast radius**: deriving purity from the effect annotation would flip a flag that `internal/format/decl.go:57` uses to emit source, making **`ailang fmt` insert a `pure` keyword the author never typed** — so inference stayed LOCAL to `internal/testing`. Shipped one unified fix across all THREE call sites (Principle 3) in a new `internal/testing/source_strip.go`; `executor.go` 739 → **654**. **A surviving mutation is recorded, not buried**: the evaluator found the `endLine < startLine` disjunct untested; the controller reproduced the survival, made the fallback test a table over both disjuncts, and **proved** the new subtest kills it. All three non-blocking findings acted on. Zero seed work — `#535` stays OPEN.]

- **[LANDED 2026-08-01 (iter-128) — FILED AND FIXED IN ONE ITERATION. PR #552 → squash `9c2081b05`; Gate 3b GREEN SHA-addressed (20 checks, 0 pending, 0 failures, all four REQUIRED contexts + SonarCloud). Planner opus, executor codex `gpt-5.6-sol`, evaluator sonnet **PASS 91/100 r1, zero blocking** with **6 independent mutations all killed** and the threshold margins re-derived first-party from both the live `/tmp` dirs and the new in-repo fixtures (identical → fixtures faithful). Not a queue pick: a Gate-0.4 measurement-validity regression, which outranks the queue.]** m-nightly-unmeasured-category-gate (`#551`; no new doc, no quorum — a bug fix inside `tools/nightly_classify.py`, the `#524`/`#548` lane) — the run-validity gate `#524` built counted only `INFRA_CATEGORIES = {api_error, timeout, executor_error}`, so the 2026-08-01 nightly, in which **12/12 benchmarks failed with `non_agentic`**, scored `tainted = 0/12` and passed as **VALID**: it entered the trend and emitted 11 verdicts (5 SUSPECTED-FLAKE), which is verbatim the second-order harm `#524` exists to prevent. Measured: `non_agentic` was **0 across all 336 rows on the eight prior nights**, then 12/12; pass rate 56/84 → 1/24; every trial `duration_ms=0` with `executor "opencode" produced non-agentic result: 1 turns, 0 tool calls` — the tool-delivery branch `error_categorizer.go:5-9` names itself, i.e. infra, not capability. **Containment ran BEFORE the sprint** (08-01 marked invalid via the tool's own `--mark-invalid`, 348 lines before and after). **Three controller premises REFUTED by the opus planner and re-verified first-party**, one of which was already public in `#551` and required a correction comment: a category-agnostic concentration gate at 0.30 is unsafe (`thrash_aborted` = **13/42 = 0.31 on 07-28 AND 07-31, both good nights**); an all-trials `duration_ms==0` rule does not fire on its own incident (**22/24**, and zero-duration is 17–21% of a *healthy* night); and the `validity_backstop_test.go` objection was mis-aimed (it governs Go's per-row backstop). **Shipped**: `INFRA_CATEGORIES` untouched for per-benchmark taint + a new `RUN_UNMEASURED_CATEGORIES` consulted **only** by `run_validity()` — widening `INFRA_CATEGORIES` instead would have permanently silenced every INDIVIDUAL non-agentic benchmark, since that set has two callers. Threshold **unchanged at 0.30**. Suite 74 → **88**, CI floor 70 → **84**. ⚠ **Time-critical side effect, done first**: the `/tmp` corpora are being reaped (07-24…07-27 trial data already gone), so the five surviving nights are now frozen as in-repo fixtures. ⚠ **STILL OPEN, PARKED FOR HUMAN**: the opencode/**qwen3.5** lane itself is broken and will keep producing unmeasurable nights — `opencode-qwen3-6` and `pi-qwen3-6` passed 4/4 on the same rig the same night, so the fault looks lane-specific. The gate now marks those nights INVALID instead of trusting them, but it cannot make the lane produce data.
- **[LANDED 2026-07-29 (iter-119) — PR #526; M1–M4 + an evaluator BLOCK-1 fix, three commits. Executor codex `gpt-5.6-sol` (two bounded runs), planner opus, evaluator sonnet PASS 86/100 r1. Decisive evidence is a replay against the REAL `/tmp/nightly_eval_20260729_rag_on/agent`: old code emits 7 filings incl. 4 `REGRESSION` = exactly `#520`–`#523`, new code emits one `INVALID` line and zero verdicts. Suite 40→60; CI anti-skip floor 20→55 (it had drifted to guard a 40-test suite at half strength). **#524 rejected in part**: the pass-rate-deviation disjunct was dropped — it cannot distinguish "we failed to measure" from "the subject genuinely broke", so it would silence the detector on a real 40/42-benchmark regression. **Deployment step owed**: `--mark-invalid 2026-07-29` on live history (with a `cp` backup) before the 2026-07-30 05:00 nightly]** m-nightly-run-validity-gate (`#524`; was ~0.5d, actually 9h, no new doc needed — this is a bug fix inside `tools/nightly_classify.py`, sized like `m-nightly-flake-guard` itself) — **found by iter-118 while triaging the 2026-07-29 nightly, which filed FOUR regression issues (`#520`–`#523`) that were all noise**, during a total serving failure of `opencode-qwen3-5-35b-a3b-mxfp8`. Measured: **42 of 42 benchmarks hit `api_error`** (baseline 1–2/night across the prior five), and suite passes fell `52/54/61/65/54` → **14/84**. The detector filed anyway. **Two distinct defects.** (1) **The infra filter is defeated by MIXED trials** — `nightly_classify.py:98-102` excludes a benchmark only when `set(cats) - INFRA_CATEGORIES` is *empty*, and all four filed rows had one trial dying `api_error` while the other survived to a `compile_error`/`thrash_aborted`, so a single non-infra category passes the filter. With `trials: 2` a *partial* outage produces mixed pairs routinely, so the filter guards the case that never happens (clean total outage) and not the one that does. (2) **There is no run-level validity gate at all** — nothing notices 100% infra-tainted benchmarks plus a ~75% pass-total collapse. **Second-order harm, and the reason this is [NEXT] rather than P3**: the invalid run is now IN the history file, so it becomes the baseline the next nights are compared against — tomorrow's genuine results will read as a dramatic *improvement*, and `m-nightly-flake-guard`'s trailing-window solidity check is computed over polluted data. Fix shape: run-level gate FIRST (infra-fraction threshold or trailing-window deviation → mark `INVALID`, file nothing, notify once, and keep it out of the trend or flag it excluded), plus treat a benchmark as infra-tainted if **any** trial hit an infra category rather than all (a benchmark whose only clean signal is one trial has `n=1`, which the variance guard already calls insufficient). This is the same measurement-validity contract as **M-EVAL-MEASUREMENT-CONTRACT** (`970d90e29`, "invalid rows never enter a trend"), which the nightly detector simply never consults. **Ruled out at triage**: fallout from `5998f4039` making `ai-check` exit non-zero on `verify.errors > 0` — it landed hours before this nightly and `contract_roman_numeral` made it the obvious suspect, but an exit-code change cannot produce `api_error` on all 42 benchmarks, including ones with no contracts at all.
- **[LANDED 2026-07-30 (iter-122) — PR #542 → squash `df9466c0d`, Gate 3b GREEN SHA-addressed on the PR (20/20 checks: 14 success, 6 skipped/N-A, **0 failures**); 4 commits. `#538` auto-closed via `Fixes` in the PR body. Evaluator sonnet **PASS 90/100 r1, zero blocking**, and it ran three mutations the controller had not — all RED. Shipped: escalation emits a distinct `SUSTAINED-FAILURE` terminal label (still `--type bug`, honest title/body, **no Discord ping** — the ping means "something broke tonight", and a sustained failure is the absence of a change) · `PAGING_CLASSES = {"regression","sustained-failure"}` with `already_regressed`→`already_paged`, the **legacy** string retained because live 07-29/07-30 rows carry it and dropping it re-pages tomorrow · shell `SUSTAINED` extractor + filing block, with the now-dead `if [[ "$ESCALATED" != "-" ]]` branch removed · emitter↔router vocabulary-lockstep guard · CI anti-skip floor **55→67** (suite 60→71). **The GAP exemption was deliberately PRESERVED and is now mutation-locked** — `#538`'s own headline ("a strictly worse benchmark gets the quieter label") was REFUTED by the planner: the ladder is ordered by *evidence of achievability*, and flake-guard D4 guarantees "no benchmark that has ever passed goes unpaged past its 3rd consecutive failing night". M1/M2/M3 are ONE commit because the routing block **silently drops unrecognised labels**, so the classifier change alone would make chronic failures *invisible* — splitting is not bisect-safe. **Nine mutations total, all RED, all reverted byte-identical.** Replay moves non-vacuously (`guarded_regressions 2→1`, `sustained=1`; bug-reaching total unchanged at 2 — a mislabel suppressed, not a signal). Live-stream check: 07-30 → `SUSTAINED-FAILURE`, 07-31/08-01 suppressed, i.e. pages ONCE not nightly; an `INVALID` night still files nothing, so `#524` is not bypassed. **BONUS reachability find, larger than the evaluator filed it**: D4's "label-agnostic across SUSPECTED-FLAKE and INSUFFICIENT-HISTORY" wording — added in the flake-guard's **quorum round 1** to close "unbounded low-history chains" — guards a state that **CANNOT OCCUR** (`consecutive>=3` already implies `nights>=3`, `trials>=6`); brute-forced 19,607 streams, the only escalation pair that exists is `(SUSTAINED-FAILURE, SUSPECTED-FLAKE)`. A quorum objection satisfied by a fix for a phantom — same shape as iter-121's dead `ast.ListType` arm and iter-105's unreachable `RunAgentBenchmark`. Condition kept (correct if thresholds are retuned), docs corrected]** m-nightly-sustained-failure-label (`#538`; ~0.9d, no new doc needed — a bug fix in `tools/nightly_classify.py` + `tools/launchd/nightly-eval.sh`, same shape as `m-nightly-run-validity-gate`) — **found by iter-122's Gate-0 triage of `#537`**, which the nightly filed as "Nightly regression: `config_file_parser`" and which **refuted under measurement**: 1 pass in 14 trials over 7 nights, never solid, with a *different cause every night* (`TC_ARITY_001` ×2, an `Option[α50]`/`int` unification, a `logic_error`, `undefined variable: null`, a `case`-vs-`match` parse cascade, `thrash_aborted` on the trial-2s). Measured boundary: **0/10 never-green → `GAP` (exempt), 1/10 → `REGRESSION`**, positive control 10/10→break still pages. The consumer already computed the distinction and threw it away, and `:674` banked `label.lower()` into the sole trend input. Sibling filings: `#540` (the honest capability-gap record, so closing `#537` didn't discard the real signal), `#539`, `#541`.
- **[LANDED 2026-07-30 (iter-123) — PR #543 → squash `46d508e7b`, Gate 3b GREEN SHA-addressed (20/20 completed, 0 failures: 14 success + 6 N/A-skipped); 2 commits. Executor codex `gpt-5.6-sol`, evaluator sonnet **PASS 81/100 r1, zero blocking**. Suite 71 → 74, CI anti-vacuity floor 67 → 70. Four mutations controller-run RED **outside the codex sandbox**, every restore sha256-identical, and the evaluator independently reproduced all four plus every claimed gate. **The executor REFUTED the controller's fix directive**: `\\n` → `\n` alone is insufficient at the suspected-flake site, because command substitution strips trailing newlines and the old literal `\n` was what separated the last row from `Model:` — the message template needed a real newline, pinned by MUT-4 (revert ONLY the template newline, keep the awk fix → still red). **Principle-3 census closed**: the three `wc -l` sites (`RCOUNT`/`SCOUNT`/`GCOUNT`) carry the same phantom-row shape but are all guarded, and the `$CLASSIFIED` extractors are filter-based, so `INSUFFICIENT_BODY` was the SINGLE unguarded instance — the fix is complete, not a one-off patch. **Gate 3b green doubles as the cross-awk portability proof** (the new tests assert REAL newlines and `test` passes on ubuntu's awk, not just the rig's BSD awk). Evaluator NOB-1 **ACTED ON** (changelog — measured 2 of 2 for this file family, so skipping would have been inconsistent with the immediately preceding commit) and NOB-5 **REFUTED by measurement** (with BOTH D2 mutations applied, that test's scenario contains no literal `\n` at all → an unreachable guard, declined deliberately). Known residual, out of scope: with no insufficient rows the summary ends with one trailing blank line]** m-nightly-summary-text-fix (`#541`; no doc needed, ~3 lines / <0.5h; P3 but trivial and it is noise a human reads every morning) — found by iter-122's planner (F-4) and **confirmed first-party against a live 06:47 inbox message**, not just read from source. TWO defects in `tools/launchd/nightly-eval.sh`: (1) `:578` `INSUFFICIENT_BODY` is unguarded, unlike its three siblings which are all wrapped in `if [[ -n "$X" ]]` — `echo ""` still emits one line, so awk prints `insufficient history:  ( over  nights, failing /3 toward escalation)` in **EVERY** nightly summary; (2) same line and `:548` use `\\n` inside a **single-quoted** awk program, so awk's `printf` emits a **literal `\n`** instead of a newline (visible mid-message as `…toward escalation)\nModel: …`). Deliberately kept out of `#542` so a behavioural fix was not diluted with cosmetics — the same call `#524` made for its own F-4.
- **[LANDED 2026-08-13 (iter-189) — commit `fc7fc67b4` direct-to-dev, Gate 3b GREEN (SHA-addressed `checks=16`, `pending=0`, **ZERO NOT-GREEN**; `CI` + `Build and Release` both `success`; `launchd drivers (bash 3.2)` green). `#665` CLOSED citing the commit]** `#665` nightly-eval banks ZERO rows — a genuine regression that OUTRANKED the human-gated queue (every top item parked on Mark). `tools/launchd/nightly-eval.sh` never sourced `~/.config/ailang/secrets.env`, so `OPENROUTER_API_KEY` was absent in the non-login launchd env → the local ollama models' motoko canary pre-flight failed → every local model skipped → `eval-suite` exited *"No models support agent evaluation"* → 0 rows banked. Ghost-verified at HEAD (`secrets.env`=**0**, `OPENROUTER`=**0**, same-path source control firing =**1**; both siblings source it: `mission-control.sh`=3, `os-rotation-filler.sh`=2). Fix = mirror the two established siblings — `os-rotation-filler.sh:22-25` carries the identical rationale for the identical canary — so the "which path carries the key" concern `#665` flagged is answered by convention (script-sourced `secrets.env`, the more-secure path, not the plist). No doc needed (AILANG-fix/mission-infra lane, same class as `m-nightly-run-validity-gate`); controller **opus** only, `metered=$0.00`. ⚠ **Not-yet-confirmed end-to-end**: mechanism proven (launchd-shaped env → key SET, value never printed; `bash -n` clean), but the full bank confirms only at the next 03:00 run.
- **[LANDED 2026-08-12 (COMPLETE, M1–M4) — `#617` (`m-take-flatmap-peak-memory`): design doc landed iter-181; **sprint plan + M1 LANDED (iter-182)** PR [#661](https://github.com/sunholo-data/ailang/pull/661) → squash `aec905da2`, Gate 3b GREEN on the merge (3/3 workflows `success`, SHA-addressed `checks=19`, **ZERO NOT-GREEN**, all four REQUIRED contexts passed on the PR), evaluator sonnet **PASS 97/100 r1, zero blocking**. Planner **opus** (lane `opus fail-closed:planner-lane-field-missing`), executor **codex `gpt-5.6-sol`**, `metered=$0.00`. **M1 = the type repair**: both `TCon{Name: "Int"}` sites → `T.Int()`, `Since` re-derived to `v0.10.0` by command, `LongDesc` + the file-header semantic note corrected (the header still carried the identical false effectful claim, found by the evaluator and fixed in-iteration), pipeline golden regenerated with a diff of **exactly** lines 138-139. Mutation drill four arms — LANDED, BUILDS, reds at base with `Int vs int` at BOTH sites, rest-of-package rc=0 under the same mutant; evaluator independently reproduced it plus a stricter per-site variant. **Three blocking plan discrepancies against the doc, all controller-verified**: `D-1` the doc teaches into **frozen** `prompts/v0.16.2.md` while active is `v0.16.5` in BOTH trees (content-hashed, eval-pinned, `grep -c prompts Makefile` = **0**) — as written it would have recreated `#617`'s own shipped-but-unreachable failure one layer up, retargeted to a new `v0.16.6`; `D-2` "no golden churn" is false and `check-golden-drift` cannot see it; `D-4` AC-6's fixture convention fatals on the `err == nil` a non-blocking note returns. **AC-3a rewritten (rule 3e)**: quorum's prescribed instrument was **already green at base** and its tests call `takeFlatMapImpl` directly (**22** sites), bypassing the export layer the sprint adds — right instrument, wrong layer. **M2 LANDED (iter-183)** PR [#668](https://github.com/sunholo-data/ailang/pull/668) → squash `6a67bb7a7`, Gate 3b GREEN (SHA-addressed `checks=16`, ZERO NOT-GREEN), evaluator **sonnet PASS 95/100 r1, zero blocking** with every drill independently reproduced: `takeFlatMap`/`takeMap` exported, runnable example, parity suite (8 arms) + the `boom`-divergence delegation instrument. **AC-1, AC-2(.ail), AC-3a, AC-4 all closed.** ⚠ **The pins had NO GATE**: `grep -rn "ailang test" make/ Makefile` → **0** (control `.ail` in `make/` = **35**) — no make target and no CI job ran ANY `.ail` suite, so both instruments were one-shot commands against a dead tree. Added `make test-stdlib-ail` + `ci.yml` step, anti-vacuity floor on both loops, `.expected` captured from the product's own stdout (rule 3k); unfused-body mutant LANDS + TYPE-CHECKS + reds it rc=2. Two bugs filed from the drills: **`#669`** (`ailang test` FALSE FAILURES on same-module delegation; 3 hypotheses refuted) and **`#670`** (`expected.stdout` display-only for all 194 examples — corrupt it and `verify-examples` is still rc=0). ****M3 LANDED (iter-185)** PR [#675](https://github.com/sunholo-data/ailang/pull/675) → squash `ebbc5a749`, Gate 3b GREEN (SHA-addressed `checks=21`, `pending=0`, **ZERO NOT-GREEN**, all 4/4 REQUIRED contexts from real `pull_request` events, `CLEAN`), evaluator **sonnet PASS 91/100 r1, zero blocking**. **AC-5 closed**: `takeFlatMap` AND `takeMap` ≥1 in all five files (baseline 0/0, same-path controls firing), `active` = `v0.16.6` in both trees, recorded hash == file sha256 in both, cross-tree `diff -q` rc=0, `v0.16.2`/`v0.16.5` untouched so pinned eval baselines still resolve byte-identical. **Reachability verified — the exact check `#617` itself failed**: the built binary's `prompt --source=embedded` genuinely serves the new teaching (control: v0.16.5's `toInts` still present), so `D-1`'s frozen-prompt trap was not recreated one layer up. **Unlike M2's pins, this one HAS a gate**: corrupting the recorded hash (mutant LANDED by sha256) reds `TestAILANGPromptLoading` + `TestPromptDisambiguation` with `hash mismatch for "v0.16.6"` — right mechanism, not a bare rc. Evaluator's one real finding fixed in-iteration (`f9fd21e3f`): AC-3b requires **both** repro pairs and the changelog shipped only the `takeFlatMap` half; V25/V26 rows + the V7 non-allocating caveat added. Ruled out by measurement: the `docs/docs/reference/...` link is the file's own convention (3 existing, `12473d76c8`) and its anchor resolves under real `github-slugger`; the duplicate `latest` tag is vestigial (`findLatestVersion` filters on `production` + max `Created`; 25 versions carry it); `m-verify-stdlib-stale-path`/`make verify-stdlib` are both real. ****M4 LANDED (iter-186)** PR [#681](https://github.com/sunholo-data/ailang/pull/681) → squash `905722f28`, Gate 3b GREEN (SHA-addressed `checks=21`, `pending=0`, **ZERO NOT-GREEN**, all 4/4 REQUIRED contexts from real `pull_request` events, `CLEAN`), evaluator **sonnet PASS 98/100 r1, zero blocking**. **AC-6 closed**, so `#617` is COMPLETE (AC-1..AC-7) and doc + plan moved to `implemented/v1_1_0/`. M4 shipped the non-blocking `LIST_TAKE_AFTER_FLATMAP` note: a new `internal/pipeline/warn_take_after_flatmap.go` modelled on `strict_fallbacks.go`, wired into BOTH pipelines, with four warning-based fixtures as a NEW test func (D-4 — a `footgunFixtures` row is impossible, `:212` fatals on the `err == nil` a non-blocking warning returns; the doc's AC-6 text describes that impossible mechanism, so the plan wins and its 4th `sortBy` fixture implements a Risks row the doc's AC-6 omitted). ⚠ **THE PLAN'S CORE SHAPE WOULD HAVE SHIPPED A DETECTOR THAT NEVER FIRES** — `#617`'s own shipped-but-unreachable failure recreated inside the sprint fixing it. The executor SELF-REPORTED it (rule 3h(d): better evidence than a silent deviation) and I adjudicated in both arms: elaboration always emits ANF `let tmp = flatMap(f, xs) in take(n, tmp)`, never the plan's nested `App`; neutering the ANF arm LANDS (`21a22852`→`7b042ce3`), BUILDS (rc=0) and reds `/direct_trap` with `got warnings: []`. The converse was measured too — the nested-`App` arm is unexercised — first named in a comment, then **pinned** after SonarCloud red-lighted new-code coverage at 56.1% (negative control: `dev` is `success`, not inherited) on exactly those lines: `TestDetectTakeAfterFlatMap_NestedAppArm` drives it against a hand-built Core program (the only way to reach it), non-vacuity proved (`expected 1 warning, got 0`), coverage `Position` 0→100%, `Detect` 85.7→100%, Sonar green. **Reachability verified through the real CLI** — the exact check `#617` is — `ailang check` prints the note with its `takeFlatMap` fix at rc=0, `✓ No errors found!` intact, fused file silent; ⚠ my first probe's CONTROL read 0 because `MOD010` killed the run before the detector executed. Evaluator wrote 8 adversarial fixtures of its own (aliases, shadowing, lambdas, intervening calls), all correct; its one non-blocking finding was reproduced first-party and filed as **`#680`** — nested composition reports only the OUTERMOST trap (1 of 2), ANF wrapping the inner in an extra `Let` — and recorded as a named blind spot in the footguns row. Plan: `design_docs/implemented/v1_1_0/m-take-flatmap-peak-memory-sprint-plan.md`; sprint JSON is **gitignored** (`.gitignore:77 .ailang/`) so it needs `git add -f` — the 49 existing ones are grandfathered]**
- **[DOC LANDED + RE-SCOPED, PARKED `needs-human-review` on `D-14` 2026-08-13 (iter-190)** — doc at `design_docs/planned/v1_0_0/m-dialect-keyword-diagnostics.md`, squash `9ebdad07c`, Gate 3b GREEN (SHA-addressed `checks=16`, `pending=0`, zero not-green). Ghost discipline at HEAD confirmed ALL NINE shapes (control `match` rc=0 firing, `--relax-modules` narrowing travels with the finding). Quorum BLOCKED x2, `absent_reviewers: []` both rounds. R1 re-scoped the doc HARD on a correct Minimal-Frozen-Core objection: **9 spellings / 2.5-3.5d -> `case` only / 1-1.5d**, other eight deferred behind a named evidence trigger. **R2 blocker is a fact about the parser, not the doc**: it has fixed 4-token lookahead (`parser.go:134-139`) and `nextToken()` (`parser.go:214-220`) is a pure forward shift with **ZERO** save/restore/rewind/backtrack methods (control: 125 `*Parser` methods), so the designed "lookahead/reparse past an arbitrary subject expression" is INFEASIBLE as written. Neither R2 objection carries a reviewer `proposed_fix`, and choosing a mechanism would be controller-invented, so the narrow-refinement carve-out does NOT apply (Standing rule 2). **`D-14` on `#635`, one word: (A)** recovery-site detection at `parser_literals.go:562` (no rewind needed, fits the parser as-is); **(B)** statement-initial soft keyword (fits 4 tokens only for simple subjects); **(C)** add parser backtracking (touches all 125 methods; a core change, against the north star); **(D)** drop it (one observed occurrence; `ailang fmt` cannot be the auto-fix - it parses before formatting and leaves parse-invalid input byte-identical, sha256-verified). Secondary: is a transcript scan across 468 recorded nights worth the rig time to settle the "zero occurrences for the other eight" premise, or should the doc just narrow its claim to "no occurrence is *recorded*"? Original row: (`#539`; **NEW-DOC needed + quorum**, parser-touching so a Conflict Surface is required; P2 — not a v1.0 bar item; ~1–2d) — found by iter-122 while triaging `#537`. A wrong-*dialect* keyword produces a cascade whose first and most-read entry prescribes **a fix that cannot work**. Measured first-party with a one-token positive control (`match c {…}` → rc=0, `✓ No errors found!`; `case c {…}` → rc=1 with **8** errors): the top diagnostic is `PAR020 missing ';' between block statements … Add a ';' after the previous statement`, and **nothing in the 8 errors names `match`** — `case` is not a keyword, so it parses as a variable reference. Full measured table: `case`/`switch`/`return` → misleading `PAR020`; `data C = A | B` → `PAR015 bare assignment … Use: let C = … in`, i.e. a **wrong** fix (the answer is `type`); `enum` → `expected }, got ,`; `elif` → never names `else if`; bare `fn` (no `export`) → useless, because the good `PAR_EXPORT_REQUIRES_FUNC` message is **`export`-gated**; `struct`/`class` → **degrade to a *type* error** (`undefined variable: struct`), so the agent is not even told the problem is syntactic. **The repo already ships the right shape for this class** — `PAR_MATCH_ARROW` ("match arms use '=>' … not '->'" + verbatim fix) and `PAR_EXPORT_REQUIRES_FUNC` — it just does not cover the rest; follow that template and suppress the downstream cascade. ⚠ **Demand evidence is deliberately WEAK and must not be oversold**: the `case` instance appeared in the 2026-07-30 nightly (`config_file_parser` trial 1: 12 turns, 4867 output tokens, ending in this cascade), but the hypothesis that it drives that benchmark's sustained failure was **REFUTED by per-night scan** — `case` appears on that one night only; 07-26/27/28/29 used `match` correctly 3–7× and failed for unrelated reasons. So this is a real DX defect with ONE observed occurrence. Note the standing counter-argument from `project_docx_stuck_dialect_confusion`: sharper error text may not be the lever, auto-fix might be — the design doc should decide between a dedicated diagnostic and accepting `case` as a `match` alias (`m-syntax-ai-forgiving` established that lane).
- **[RESOLVED 2026-08-13 (iter-191)] [SWEEP iter-158] external-issue batch — all six dispositioned; two split out as their own rows below (per this row's own instruction).** Ghost discipline at HEAD `b60e41946` (v0.33.1-1; both binaries rebuilt, `--version` == `git describe`). **`#609` CLOSED — fixed-since-filing**: `toInts` shipped at `std/bytes.ail:99` (`1677fcff9`, contained in release **v0.33.1**), live-verified with the issue's own example (`toInts(fromString("é"))` → `[195, 169]`). **`#611` CLOSED**: the driver half was ALREADY LANDED at `d14f106bb` (codex → `MISSION_<ROLE>_FALLBACK` → pi loop → opus; both env hard-pins reverted; `:floor` verified = OpenRouter price-routing variant, part of the model ID — read from `mission-control.sh:356-371`, not guessed), and the missing in-iteration half — the issue's own explicit constraint, since a codex probe can rc=0 on a spent bucket — is closed by this iteration's Gate-5 skill edit `14efcae22` (both Gate-3 Fallback bullets now follow the chain). **`#581` re-CONFIRMED at HEAD**, stays open low-P: fresh discriminating pair (control `codex declared:codex-ok` vs +fenced-block `opus fail-closed:path-not-in-codex-allowlist`); script untouched since filing (`git log --since 2026-08-04` empty); fail-safe direction re-affirmed, and a doc without the `**Planner-Lane**:` declaration never reaches path parsing at all (re-observed: `planner-lane-field-missing` fires first) — items 2/3 remain audit debt. **`#554` quiescent, stays open (ops watch)**: eval-shaped (≥30k-token first-step) emission per day since 08-01 — the filed collapse (31/56) never recurred at full severity, one moderate dip 08-09 (36/58), 08-11..13 all ~100% (23–24/24); root cause still open, no new mechanism evidence. **`#607` CONFIRMED / `#610` direction-CONFIRMED** → the two rows below. Verdict comments posted on all five surviving issues. Original row: (P3 triage-lite; **not** a v1.0 bar item; ~0.5d). Surfaced by the Gate-0 weekly external-issue sweep, which greps every open issue's `#N` against this charter: **7** returned zero mentions (control `#545` → **3**, so the detector fires). `#598` was resolved on the spot — a **duplicate of `#602`**, whose fix `1d355245a` was verified live at HEAD (`readChildPIDIfPresent` + retry to `maxRaceAttempts`, non-vacuous because exhausting the attempts calls `t.Fatalf`) — and closed with that evidence. The remaining **six**, each needing the ghost discipline (live-repro at HEAD) before it earns a row of its own: `#611` mission-driver executor fallback chain codex→pi-deepseek→opus (Mark-ratified 2026-08-06; currently hard-pinned by hand in both missions' env files — this is **mission-infra and the most actionable of the six**); `#581` planner-lane derivation parses bullets inside fenced code blocks as real paths, silently degrading routing to opus (+2 audit-debt follow-ups from the iter-137 evaluation); `#554` nightly `non_agentic` outage = recurring ollama tool-call emission collapse, with "switch to qwen3.6" already REFUTED by measurement (root cause still open; ops-flavoured); and three `from:cli` reports from a real downstream consumer — `#610` retaining `mapE`-over-query-rows values costs **~49×** more memory than the same values from a range loop (carries its own control), `#609` `std/bytes` has `fromInts` but no `toInts` so a string's bytes cannot be walked to transcode legacy charsets, `#607` batch mode: `exit()` inside one batch item panics the whole run instead of failing that item. ⚠ Ordering: a sweep NEVER outranks an existing pick — this sits below the picks above it by construction, and `#610`/`#607` should be split out into their own rows if their repros confirm, since a memory blow-up and a whole-run panic are both bar-relevant reliability defects rather than triage debt.
- **[LANDED 2026-08-13 (iter-192) — PR [#690](https://github.com/sunholo-data/ailang/pull/690) → squash `7bad0e609`, Gate 3b GREEN (SHA-addressed **21** checks, `pending=0`, **zero failures**, all 4/4 REQUIRED contexts from real `pull_request` events, `CLEAN`; count climbed 17→21 during the poll so `pending=0` was required, not inferred), evaluator **sonnet PASS 96/100 r1, zero blocking**. Controller **opus** direct-fix (the row's own sanctioned lane), evaluator sonnet → generator≠judge holds; `metered=$0.00`. `#607` CLOSED with the evidence. **Ghost discipline re-run at HEAD `47c00318d`, three arms**: defect (rc=2, `panic: (*eval.EvalExitCode)` through `io.go:145` → `run_helpers.go:656`, `[2/2]` never runs), mechanism-removed control (rc=0, `2/2 succeeded`), path-specificity control (single-file, same `exit(1)` → rc=1, **zero** panic frames) — outcomes differ across the mechanism arm, so it is `exit()` and not the environment. **Fix**: `executeBatchItem` now routes through `runBatchItemEntrypoint` → `recoverBatchItemExit`; `exit(N != 0)` fails THAT item and the loop continues, `exit(0)` is a success, non-exit panics **re-raised unchanged** so a real crash stays loud. **Principle-3 census run before patching**: `executeModuleEntrypoint` has exactly **two** call sites, both now recovered — complete, not a one-off. **Mutation drill, one per branch (rule 3j)**, each LANDED (sha256) + BUILDS (rc=0) + redding ONLY its own arm (remove-recover → *"leaked a Go panic"* + *"[2/2] never started"*; non-zero arm → the `1/2` count and rc; exit(0) arm; re-panic arm → *"a non-exit panic was swallowed"*). **Inverse arm: recover removed AND the new tests skipped → rest of `cmd/ailang` rc=0**, i.e. batch mode had NO regression tests at all and the defect shipped entirely undetected; five now, covering all four branches. The re-panic branch is unreachable from any `.ail` fixture, so the recover logic is split out and unit-tested directly rather than shipping unguarded. ⚠ **I caught my own test being hollow before shipping**: the first `ExitCodeZeroCountsAsSuccess` asserted on inputs that took the println arm and never called `exit(0)` (rule 3i aimed at my own test); shipped version asserts on the exit(0) arm's own stdout line and `t.Fatal`s if absent. `go build ./...` rc=1 **baselined as a BASE failure** (`cmd/wasm`, identical on the pristine tree — rule 3e). **The evaluator's best finding, reproduced before acting on it: the census was right as scoped and too narrow ONE LEVEL UP** — `runtime.CallEntrypoint` has two further call sites at `internal/embed/embed.go:237,314` with **zero** `recover()` in the package (control `run_helpers.go`=2), so `exit()` in embedded AILANG still panics the HOST → filed **`#691`** (needs its own contract decision: a host has no `os.Exit` to map onto, so a typed error is proposed). Second finding filed as **`#692`** (batch never calls `flushDebugOutput`, so Debug output works per-file and vanishes per-batch). ⚠ **Bookkeeping instrument defect**: the PR's `Fixes #607` auto-closed the issue first, and **`gh issue close --comment` on an already-closed issue silently DROPS the comment while exiting 0** — caught by re-reading the comment count rather than trusting rc; re-posted via `--body-file` and verified]** m-batch-exit-panic (`#607`) — batch mode: `exit()` in one item panics the whole run** ([cli-DEMAND]; P1 reliability; small fix ~0.5d; NO new doc — the fix shape is established by the sibling path, so this starts at sprint-planner or direct-fix with regression guard). CONFIRMED at HEAD `b60e41946` frame-for-frame as filed (iter-191): item `[1/2]` panics `*eval.EvalExitCode` through `effects/io.go:145` → `run_helpers.go:656` (`executeBatchItem` has **no recover**), rc=2, raw Go stack to the user, item `[2/2]` never runs. Control proving path-specificity: the single-file path recovers the sentinel at `main_run_exec.go:549-555` → clean rc=1 with no stack. The recurring guard-the-helper-miss-the-call-site class: mirror the recover in `executeBatchItem`, mark the item failed, continue — the `Batch complete: X/Y succeeded` summary already implies that contract. Acceptance must include a CI batch fixture where item 1 calls `exit(1)` and item 2 STILL runs and banks `1/2 succeeded` (and per rule 3j, a neutering mutation on the new recover must red it). Demand: real downstream consumer (2,500-file PDF batch job).
- **[LANDED 2026-08-14 (iter-197)] `#691` — `exit()` in an embedded module panicked the HOST** ([cli-DEMAND]; P1 reliability; no design doc — the fix shape was established by the two sibling recover paths, same direct-fix basis as `#607` at iteration 192). PR [#705](https://github.com/sunholo-data/ailang/pull/705) → squash `20d538a43`, Gate 3b GREEN (**21** checks, `pending=0`, 4/4 required, platform legs named), evaluator **sonnet PASS 96/100 r1, zero blocking**, `metered=$0.00`. Contract decided by the controller and **flagged for Mark**: `exit(N)` → typed `*embed.ExitError{Code: N}`, and **`exit(0)` is an `ExitError` too, not a nil error** (the CLI batch path diverges deliberately — it owns a process, embed does not). `runtime.CallEntrypoint` stays panic-based on purpose; census re-derived independently by the judge, 4 sites repo-wide. Mutation drill one-per-branch, both call sites pinned INDEPENDENTLY; inverse arm rc=0 / **60 PASS** proved the defect had shipped entirely undetected. ⚠ **PROCESS FINDING — this item never had a queue row.** It was filed inside iteration 192's `m-batch-exit-panic` row as prose, so it was invisible to every `[NEXT]` scan and survived only because two consecutive iterations happened to name it in their *Next* line. Same class as iteration 195's "ratified item vanished as a task": **a follow-up filed in another row's prose is not a backlog item.** The three rows below exist so this cluster cannot vanish the same way.
- **[LANDED 2026-08-14 (iter-199)] `#692` — batch mode silently dropped `Debug` ghost-effect output** ([cli-DEMAND]; P2; direct-fix lane, no design doc — same basis as `#691`/`#607`). PR [#711](https://github.com/sunholo-data/ailang/pull/711) → squash `29ad1c559`. Reproduced live at `6dd525c58` before routing (single-file **1×**, `--batch` over 2 inputs **0×** while reporting `2/2 succeeded`). Executor codex `gpt-5.6-sol`, evaluator sonnet **PASS 99/100 r1, zero blocking**. Gate 3b: **21** checks, `pending=0` (count climbed 17→21 during the poll, so it was required not inferred), **4/4** REQUIRED, `state=CLEAN`, all Windows/ubuntu/macos legs `success`. Two controller design calls made before routing: the flush is **deferred** (a failing item's logs survive, incl. the `exit()` path from `#607`) and carries its **own label** (the `[i/n]` header is inside `if !quiet`, so a bare flush is unattributable under `--quiet`). **The finding worth carrying forward:** the executor's severity-filter test asserted only an ABSENCE and **survived the neutering mutant** — an absence is satisfied equally by "the filter worked" and "nothing was ever flushed" — caught only because rule 3i's per-row drill was run instead of the suite-wide one; repaired with a known-positive control and it now reds on that assertion. codex correctly labelled the full-package gate AND the mutation inverse arm `UNINFORMATIVE UNDER SANDBOX`; both re-run outside it by the controller (inverse arm **rc=0, 0 FAIL** — the proof the new tests are killers, not bystanders). Non-blocking residue named by the judge: `setupEnvContext`'s `os.Exit(1)` arms bypass deferred functions, so that path does not flush (pre-existing; fires before any AILANG code runs).
- **[LANDED 2026-08-14 (iter-200)] `#706` — `exit(0)` in a route/A2A/MCP handler returned HTTP 500** ([cli-DEMAND]; P1 reliability; direct-fix lane, no design doc — same basis as `#691`/`#692`/`#607`). PR [#714](https://github.com/sunholo-data/ailang/pull/714) → squash `1c7fa675b`. Reproduced live at `66abbc660` before routing with `--caps IO`: unit-return route **200**, `exit(0)` route **500** `{"error":"program called exit(0)"}`, host ALIVE both — `#691`'s incomplete half, not a regression. **The control arm earned its keep**: the first repro attempt omitted `--caps IO` and BOTH arms returned 500 for an unrelated capability reason, so the subject would have "reproduced" for the wrong cause. Shipped per `D-17`: one `isCleanExit` classifier plus a **separate branch at each of the three call sites**, since guard-the-helper-miss-the-call-site is exactly how this `#607`→`#691`→`#706` chain arose. Executor codex `gpt-5.6-sol`, evaluator sonnet **90/100 PASS** in its OWN worktree (iter-199's skill edit, first use). Gate 3b: **20** contexts, 15 pass / 5 skipped / **zero failures**, 4/4 REQUIRED, `MERGEABLE CLEAN`, all three platform legs + `test-windows` green. **Two findings worth carrying forward.** (1) The judge's BLOCKING call, reproduced first-party: `TestA2AExitNonzeroFails` asserted only `state == "failed"` — a state **every** failure reaches, including one where `exit()` never ran. Neutering the IO grant showed **5 of 6 arms correctly failed and that one PASSED**. It is iteration 199's absence-assertion class one iteration later in a new disguise: not an absence, but a **low-cardinality enum whose value is over-subscribed**. Fixed `bd9984084`; it also refuted the controller's own commit message ("every arm would pass for the wrong reason" — only one did) and showed the positive control never proved the grant took, since `no_exit.ail` is `main() -> int = 42` with **no IO effect**. (2) The controller's directive shipped a **broken acceptance criterion** — `go build ./...` is rc=1 on pristine `origin/dev` (`cmd/wasm`), i.e. already red at base (rule 3e(a)); codex reported it rather than papering over it, which is the deviation behaviour the rule wants. Executor design call not asked for and correct: `embed.ToGo(result)` guarded in a2a/mcp because a clean exit returns nil; `ToGo` maps `UnitValue` → `nil`, so the clean-exit body is shape-identical to a unit return.
- **[LANDED 2026-08-14 iter-202, PR #719 → `afe06487e`, filed as `#718`] [email-parse-DEMAND] `ailang install` on an already-declared dependency corrupts `ailang.toml`** (reported by the `email-parse` mission 2026-08-14; **reproduced first-party at `66abbc660`**, so it enters on its own evidence, not on the strength of the request). `install` inserts the dep line unconditionally, so a second `ailang install sunholo/gemini_files@0.2.1` in a project already declaring it yields a **duplicate TOML key** and an unparseable manifest (`ailang lock` → rc=1, `Key 'dependencies."…"' has already been defined`); control on a healthy manifest rc=0. Two sibling helpers share the defect — `appendDependencyToFile` and `appendGitDependencyToFile` (`cmd/ailang/pkg_commands.go:151`/`:185`) — with four call sites (`pkg_commands.go:90`/`:106`/`:148`, `pkg_install.go:130`, `pkg_init.go:88`), so this is the guard-the-helper-miss-the-call-site shape again and the fix must be an idempotent upsert reached by **every** site. **One half of the report is REFUTED and must not be re-chased**: `ailang lock` does NOT silently pass — it is rc=1 and names the real TOML parse error. (My own first reading said rc=0; that was `tail`'s status through a pipe, the exact trap the verification protocol's step 3 exists for.) The compounding half is structurally confirmed and NOT yet live-repro'd: `internal/pipeline/package_resolver.go:40-43` swallows the `LoadManifest` error and returns nil, so the user-visible failure is an unrelated "requires ailang.toml and ailang.lock" from `loader.go:148` — a Critical-Principle-2 silent fallback, and the reason the true cause is only visible via `ailang lock`. **OUTCOME (iter-202):** both halves fixed. Half 1 = one shared idempotent `upsertDependencyLine` reached by both writers, so all 5 call sites inherit the guard; `add` keeps its explicit already-exists error, `install` reports `Updated` on a version change. Half 2 = `tryLoadPackageResolver` returns `(resolver, error)`; absent manifest still `(nil, nil)`, existing-but-broken names its path + the TOML error, and the lock load is split the same way (missing optional, malformed loud). **Round 1 FAILED 66/100** on four TOML-legal scanner blind spots, one of them a regression the PR introduced (a header with a trailing comment produced a SECOND `[dependencies]` table, reproducing this very bug on input the pre-fix code handled). Answered structurally by `writeManifestChecked`, which re-parses and rolls back any write that would leave a manifest less parseable than it found it; round 2 **PASS 95/100, zero blocking**. Residual scanner gaps → `#720`.
- **[LANDED 2026-08-15 iter-203, PR #722 → `3ec1dcb02`] `#720` — `ailang.toml` upsert: scanner gaps the rollback net catches but does not fix** (filed by iteration 202 from the round-2 evaluator's attack on the fresh code; P3). None ships corruption — `writeManifestChecked` refuses the write — so each surfaces as a refused operation, not a broken file. (a) `openMultilineString` false-positives on a *literal* string containing `"""` (e.g. a `notes` field documenting TOML), skipping every following line; the most plausible of the set. (b) A TOML-legal quoted table header `["dependencies"]` is unrecognised, so a second semantic table is appended. (c) `stripLineComment` has no escape awareness (latent: needs a key containing both a quote and a `#`, which no real `vendor/name` can). (d) The test helper `countDependencyKey` has the same gap in a new shape — single-quote parity untracked. (e) Appending to an ALREADY-broken manifest still prints `✓ Added`. **The decision worth making rather than drifting into**: whether this write path should hand-edit TOML text at all, or parse → mutate → re-serialize and accept losing comment/formatting preservation. The rollback net makes the current approach *safe*; it does not make it *right*. **OUTCOME (iter-203): the row's own premise was FALSE and that was the find.** "None ships corruption" holds only for manifests that pass semantic validation: `writeManifestChecked`'s "was it parseable before?" probe was `pkg.LoadManifest` — parse **plus `Validate`** — so a perfectly good TOML manifest missing `edition` (or with a one-level `name`) counted as ALREADY BROKEN, the re-check was skipped, and a **duplicate key landed on disk under a `✓ Added`**. Measured on two fixtures differing by exactly one line. Surfaced by a CONTROL that failed (rule 3a) while reproducing item (e), not by any of (a)–(e). Fixed with a parse-only `pkg.ParseManifestFile`. Items (a)–(e) all fixed too, each reproduced first-party first. **The decision is MADE: keep hand-editing TOML text** — comment/formatting preservation is a real product property and the fixed net bounds the blind spots by construction; the judge was invited to argue against it and concurred. Guarded by a 12-shape differential corpus with an anti-vacuity floor (the instrument iter-202 lacked). Evaluator sonnet **94/100 PASS, zero blocking**; its three non-blocking findings were unpinned refusal branches, reproduced by the controller and closed in `72b60b153` — one of them `appendGitDependencyToFile`'s copy of the very warn branch this sprint added, i.e. guard-the-helper-miss-the-call-site one commit after fixing that same shape.
- **[LANDED 2026-08-15 iter-206, PR #726 → `640bab054`] `#717` — module-only govulncheck allowlist entries skipped expiry and malformed-date validation.** Expired module-only findings now print `[allowlisted, EXPIRED <date>]` but remain non-gating per `#703`; malformed expiries exit 2 before success output. Shared classification prevents reaching/module-only boundary drift. The first non-gating test was vacuous and survived a gate-everything mutant; replacement `decide` arms kill it. Evaluator sonnet PASS (score unavailable after the original controller quota exit); recovery independently rebuilt and reran all relevant gates. Issue verdict comment posted after merge.
- **[LANDED 2026-08-14 iter-201, PR #716 → `ba501607d`] `#703` — `govulncheck-filter` silently drops module-level findings** (filed by iteration 196; the gate prints a confident green whose denominator excludes a whole class, and one dropped finding is a real unallowlisted Ollama advisory, `GO-2026-5750`/`CVE-2026-7020`, with no upstream fix). P1 gate-integrity: a vacuous green on the security gate is worse than a red. **Fix**: `readFindings` partitions OSVs into reaching vs module-only across all frames (also fixed a second latent `Trace[0]`-only drop); module-only findings reported in a third named bucket, non-gating; `GO-2026-5750` allowlisted (expiry 2026-10-29). Evaluator sonnet 96/100. Follow-up `#717` filed (module-only entries skip expiry-check).
- **[LANDED 2026-08-13 (iter-194, inherited from the orphaned iter-193)] M-MISSION-LOOP-UNIFIED-TELEMETRY M2+M3** — directive-driven sprint (three Design Freeze items ratified by Mark 2026-08-13); it never had a queue row, which is why the charter could not see that iteration 193 had finished it. M2: mission stages report a real `status` and their tokens, and stage cost/tokens roll up into the chain total (measured on the iter-190 shape through the built binary: 4 stages `pending` / $0.0000 / 0 tokens → completed+failed / $0.1077 / 37,414 tokens). M3: `chains post-iteration` dual-writes to a remote observatory named by `AILANG_CHAINS_CLOUD`/`--cloud`, under the SAME chain and stage ids, with a per-target bounded spool and rc=0 on an unreachable remote. PR [#697](https://github.com/sunholo-data/ailang/pull/697) → merge `5a4dac723`; evaluator sonnet **PASS 82/100**; Gate 3b GREEN (SHA-addressed **21** checks, zero NOT-GREEN, 4/4 REQUIRED). **Two gaps reproduced first-party and filed as `#698`, NOT closed by this row**: ratified freeze item 3 (opt-in remote READ) has **zero** implementation, so the design's own Primary Goal is unreachable by any shipped tool; and the pinned-ID retry guard at `store_chains.go:334` **survives a landed, building mutation** with the whole `internal/observatory` package green. M1 landed earlier at `56b449d01`.
- **[LANDED 2026-08-13 (iter-195)] `#698` fast-follow — M-TELEMETRY-REMOTE-READ-FASTFOLLOW M1–M3** (the buildable two-thirds of the gaps iter-194 filed; **`#698` deliberately stays OPEN** for its part 1). **M1** pins the `CreateStage` pinned-ID retry guard that survived iter-194's mutation — and `#698` §2's own account of it is WRONG: `chain_stages` carries `id TEXT PRIMARY KEY` (`schema_chains.sql:37`) as well as `UNIQUE(chain_id, stage_number)`, so a duplicate PINNED id enters the retry branch **deterministically** and the test needs no concurrency. Re-derived first-party, three arms: mutant LANDED (sha `54432ddb…`→`05892ff1…`) + BUILDING, new test alone rc=1, **inverse (new test `-skip`ped) rc=0/0 FAIL** so it is the sole killer, and the property-named `TestPostIteration_PinnedIDIsNotSilentlyReplaced` still rc=0 under the mutant. **M2** arms 4 of the 5 unpinned error branches; the 5th is **unreachable by construction** (`EvalAssessment` is 26 scalar fields → `json.Marshal` cannot fail) and ships a type guard rather than a fake arm. **M3** restores the orphaned M-MISSION-LOOP-UNIFIED-TELEMETRY sprint JSON and names the defect that orphaned it: **`.gitignore:77` ignores `.ailang/` with no negation**, so a NEW sprint JSON is skipped by `git add -A` **silently** (empty output, 0 staged) and hidden from `git status` — every sprint JSON needs `git add -f` until the rule gains a negation. Tests + state artifacts only, **zero production code**. Evaluator sonnet **PASS 83/100**, whose one BLOCKING find was the controller's own: the new arms were **Windows-red** because `os.UserHomeDir()` reads `USERPROFILE`, not `HOME` — they failed for the PLATFORM, not the code. Fixed, re-drilled, Windows green. PR [#699](https://github.com/sunholo-data/ailang/pull/699) → merge `8e8447f51`; Gate 3b GREEN (SHA-addressed **21** checks, zero NOT-GREEN, 4/4 REQUIRED, count climbed 14→21 so `pending=0` was required not inferred). **ROOT CAUSE of the vanished freeze item, and the reusable lesson**: M3's task list contained "Opt-in remote read"; its acceptance criteria contained five entries and **zero** mentioning read or remote (controls: task list 1, AC bullets 5). **A task with no acceptance criterion is invisible to the gate** — a distinct shape from rule 3b(vi)'s vacuous AC.
- **[LANDED 2026-08-14 (iter-198)]** `#698` part 1 — opt-in remote READ, shipped at the **`view`** scope Mark ratified as `D-15`. M4 had shipped DECISION-GATED with **no acceptance criteria by design**; iteration 198 wrote them first (the plan's own rule: every task carries a criterion that can FAIL and names the mutation that kills it), then executed T1–T7 across two codex runs. `openChainsReadBackend` mirrors the write half (`--remote` beats `AILANG_CHAINS_READ` beats local; **local stays the default**); 12 helper signatures widened to `observatory.Backend` in 3 files, 13 read sites swapped, and a source guard pins the direct-open allowlist to 4 survivors. The surfaces that physically cannot go remote **error** rather than silently reading local (Critical Principle 2) — `chains live`, `journey`, `find --task`, `stats --cost-per-verified-success` — via ONE helper with all **4** call sites asserted to reach it, and all 13 `eval-*` commands refuse by name so demand becomes a dated signal. **Planning refuted six inherited claims**, all re-verified first-party: 17-not-20 call sites; 12-not-15 helpers; Discovery #5's "clean boundary" (it grepped only `DB()` — **`Store()` is a second escape hatch**); `GetChainJourney`/`GetTaskSpanSummary` are `*SQLiteBackend`-only; ~400 LOC/~7h not ~230/~4h; and the 48.4s baseline (21.7s). **It also found a defect M4 would have CREATED** — firestore `GetMissionRollups` returned `nil, nil` (the only such stub, control 26 methods), so `chains stats --by-mission --remote gcp` would have printed "no missions" for a store it never queried; closed as T6. Evaluator sonnet **88/100 PASS**, one BLOCKING that was the controller's own (two arms read the developer's real `$HOME/.ailang`, red-lighting `Build macos-latest`; the judge reproduced it independently) plus a real `-remote`-single-dash bypass that lost the `D-15` text — both fixed in-iteration with drills and an over-match control. PR [#710](https://github.com/sunholo-data/ailang/pull/710) → squash `4942362f4`; Gate 3b GREEN (21 checks, `pending=0`, 4/4 REQUIRED, 5 platform legs named). **`#698` may now be CLOSED** — parts 2 and 3 landed at iter-195, part 1 here.
- **[LANDED 2026-08-15 iter-204, PR #723 → `54fdcb32c`] SonarCloud new-code-coverage triage — VERDICT: REAL, and iteration 203's own premise REFUTED.** The queue row said *"0.0% on a heavily-tested diff is implausible — instrument or real?"*. The diff **was** heavily tested; all 358 of its new test lines target `cmd/ailang/**`, which `sonar.coverage.exclusions` omits, so neither they nor the 113 production lines they cover enter the metric. The countable remainder was 14 lines — `pkg.ParseManifestFile` — and `make test-coverage` runs **without `-coverpkg`**, so `cmd/ailang`'s call sites attribute nothing to `internal/pkg`. 0/7 → 0.0%, correct arithmetic on a real gap (Sonar API reconciles exactly: 143 to-cover / 29 uncovered, `manifest.go` = 7 of the 29, the only file at 0.0%). **The gap was worse than a metric**: `ParseManifestFile` — `#720`'s load-bearing parse-without-validate predicate — had **zero tests in its own package** (0 hits / 14 files; control `LoadManifestFile` → 9), and reverting the mechanism left the **entire `internal/pkg` package green**. Fixed with 4 arms (discriminating accept-vs-reject pair, both refusal branches pinned by message, positive control, anti-vacuity floor); 0.0% → **100.0%**; Sonar Quality Gate **passed** on the PR. Evaluator sonnet 96/100, its precondition-neutering drill killing all 4 arms. **RULED OUT by measurement**: removing the `cmd/ailang/**` coverage exclusion — that package measures **9.3%** unit coverage, so the exclusion stays and its file comment's invariant is aspirational.
- **[DESIGN DOC LANDED 2026-08-15 iter-205, PR #724 → `c095f1f0e`; SPRINT PARKED on `D-COV-1`] `-coverpkg`: cross-package coverage is unattributed repo-wide.** Doc: `design_docs/planned/v0_33_2/m-coverage-cross-package-attribution.md` (Planned, 571 lines, 8 Verification-Log rows + 8 controller M-facts). **RECOMMENDATION: refuse the change** — keep own-package (LOCALITY) semantics for the gated/badged/Sonar metric, add a separate non-gating `test-coverage-xpkg` diagnostic. **The crux inverts the intuition**: iteration 204's defect was a function with NO own-package test, so `-coverpkg` would have painted it green and suppressed the true-positive signal that forced the fix. **Two of this row's own premises below are REFUTED by first-party A/B (105 packages, 3 replicates/arm, at `376e19284`)**: runtime **89/78/82s → 92/79/83s (~+1–4%, NOT "material")**, and `total:` **45.5% → 48.1%** — the number moves **UP**, both arms far above the 29% gate, so the hazard is the gate silently LOOSENING, not breaking. **The real cost was never named here**: the merged profile goes **5.7 MB → 599 MB** (~105×), and the SonarCloud step is `continue-on-error: true` (`ci.yml:258`), so an ingest failure would be SILENT. Quorum BLOCKED ×2 with both reviewers present both rounds (`metered=$0.1454`); round 2 resolved under the narrow-refinement carve-out, both objections measured and CONFIRMED (`XC1` was vacuous — presence ≠ execution, demonstrated on two all-zero packages; the "and by whom" claim unsupported — no originating-test column, V17). **BLOCKED ON `D-COV-1` (Mark, one word): does the coverage number mean LOCALITY or EXECUTION?** Recommendation LOCALITY; an EXECUTION answer routes the doc's decomposed 3–4 day Option-A sprint, which STARTS by measuring Sonar ingesting the 599 MB profile. **Independent finding, do NOT fold into this decision**: CI runs the coverage suite **TWICE** — 492s of the 1127s critical-path `test` job — filed in the doc as `D-COV-4`. Original row follows. —
- **[SUPERSEDED BY THE ROW ABOVE — kept for the refuted premises] `-coverpkg`: cross-package coverage is unattributed repo-wide.** `make/coverage.mk:19` runs `go test -coverprofile` with no `-coverpkg`, so a package's coverage counts only its OWN tests. Consequence, generalised from iter-204's instance: any helper added to a non-excluded package **in order to serve** `cmd/ailang` reads 0% by construction, however thoroughly `cmd/ailang` exercises it — and `cmd/ailang/**` is itself coverage-excluded, so that work is invisible from both sides. Iteration 204 fixed one instance, not the class. Changing it moves every number in the repo (the badge, the 29% `test-coverage-gate` threshold, Sonar's whole baseline) and would slow a 120k-line suite materially, so it is a **decision with tradeoffs**, not a mechanical edit: NEW-DOC, quorum, then route. Do NOT patch `coverage.mk` in passing.
- **[MECHANISM REFUTED iter-209; issue remains workload evidence] m-mapE-queryall-retention (`#610`) — reported 49× retain cost does NOT reproduce on the exact available package stack.** duckdb v1.5.5 + `sunholo/duckdb@0.1.1`; 300 `queryAll` rows × 32 KiB, fresh 768-int vector per row, alternating retain/discard runs with `/usr/bin/time -l` + `ailang run --memprofile`: discard **165.6/163.6 MB**, retain **188.4/187.2 MB** (~14%, consistent with retained vectors). End heaps 7.2–7.7 MiB, no retained query graph. This refutes “each result retains its row/parse tree or whole query result”; no design doc. Keep open only for the original 5 GB DB + embedding workload (or equivalent fixture) reproducing 49×. Earlier iter-191 std/json synthetic showed ~2.4×; `queryAll`'s ~1.2 GB buffer pool remains DuckDB documentation territory (`SET memory_limit`).
- **issue #386 — effect-row inference regression** **[LANDED 2026-07-22 (iteration 82) →
  implemented/v0_31_0; full loop headless — planner opus → executor opus (M1 `6c7a92570` / M2+M3
  `b690a33e0` / M4 `b85860382` / round-2 `456d05afd`) → evaluator sonnet (generator≠judge) PASS
  95/100 after ONE round-2 fix. Application-local equality solver realizing Mark's ratified
  replace-not-delete as NON-deletion (keep constraints + local closed-row arg substitution =
  strictly stronger, guarded by `TestReplaceNotDelete_LetBoundaryPropagation`); `row_unification.go`
  UNCHANGED, no `EffectJoin`; row-var generalization (`RowVars: []string{}` → full free-row-var
  collection, `std/list.ail` untouched); + 2 in-scope secondary fixes (`ValidateEffects` walker
  skipped let/letrec bodies = the `println(show(x))` hole; `std/stream.ail`+`std/ai/streaming.ail`
  handlers row-polymorphic). Non-vacuity independently re-verified (controls still reject undeclared
  IO/FS/Env). All 4 quarantined #386 examples un-quarantined + `working`; `make verify-examples`
  green. Round-1 FAILED 61/100 (stale binary hid 3 stream examples still failing → verify-examples
  red), fixed round 2. PR #459. Known in-scope gap: explicit `! {}` inline-lambda reject is
  elaboration-erased (parser change, out of scope) — proxy fixture uses a non-empty wrong
  annotation. `mcp_tools.ail` stays quarantined (separate `Option[string]` bug)]** — DESIGN DOC created via codex-rotation
  designer: `design_docs/planned/v0_31_0/m-effect-row-show-interp.md` (PR #456). Controller
  live-verified + SHARPENED the root cause: NOT show-specific — two interacting mechanisms
  (`combineEffects` tail-drop → nested pure call erases IO [`println(show(x))` accepted as pure];
  `RowVars: []string{}` never generalized → repeated combinator uses collide). Proven: pure
  `mapE(\x. x*2)` then effectful `foldlE` ALSO fails. Quorum (gemini-3-1-pro, generator≠judge)
  REJECTED ×2: R1 EffectJoin/UnifyRows gap RESOLVED in revision; **R2 OPEN (the human decision)** —
  how the application-local solver drains/preserves solved constraints without breaking
  let-boundary global propagation (gemini: replace, don't delete → flattened-substitution). Parked
  per Gate-2 bounded-quorum rule (1 revision + 1 re-quorum). **UNPARK:** Mark ratifies the
  constraint-preservation mechanism → route straight to sprint-planner, no re-quorum. Est ~1.5–2d.
- **m-check-strict-fallbacks** **[LANDED iter 101 2026-07-24 → implemented/v1_0_0; PR #479 `1978ab44b`,
  evaluator sonnet PASS 88/100 r1. The historical decision record follows:]** **[DECIDED by Mark 2026-07-18 — option 2: post-name-resolution
  pass + curated known-empty-builder registry (catches `Ok(jo([]))`), warning-in-dev / hard-error at
  `check --package`; doc Status stamped UNPARKED → route to sprint-planner, no re-quorum. The historical
  park record follows:]** — iter-42 re-attempt (both iter-41 blockers cleared: Fable designer back; #407 quorum
  fix). Resolved the iter-41 "OPEN decision" to option (a) (syntactic surface-AST pass, hooks
  live-verified) + grounded Pattern C in the language-enforced uppercase-constructor rule — BUT a
  clean re-quorum (on a REBUILT binary; the #407 fix was NOT in the stale installed binary, so
  gpt5-6-sol had been silently unreachable) **BLOCKED** on a goal-contradicting objection:
  **the purely-syntactic pass cannot catch its own motivating incident** `None => Ok(jo([]))` —
  `jo` is a LOWERCASE function call, which the doc (and Pattern C) never flags; catching it needs
  resolved callee identity (name resolution), refuting option (a). Human decision = the architecture
  fork: (1) run after name resolution, (2) narrow the goal (literal-empties only), or (3) curated
  known-empty-builder list; + resolve the warning-vs-error/exit-1 `--package` channel. Doc has the
  full REBLOCK write-up. See log entry 47.
- **m-bytecode-pattern-arity-fix** — **[LANDED 2026-08-12 (iter-187) — PR #684 → squash `0625059d3`,
  Gate 3b GREEN (`checks=22`, `pending=0`, ZERO not-green, 4/4 REQUIRED from real `pull_request`
  events); evaluator sonnet PASS 110/120 r1, zero blocking; `#505` CLOSED]**. The P0 spun out of
  `m-bytecode-vm-parity-bugs` by Mark's option-C ruling (2026-08-04). **⚠ THIS ROW DID NOT EXIST
  UNTIL IT LANDED** — the ruling created an unblocked, ready-to-sprint doc and nothing wrote it
  into the queue, so a P0 soundness bug was invisible here for 8 days while iterations declared
  the still-PARKED parent as Next (controls at pick time: `pattern-arity` = **0** mentions,
  `bytecode` = **15**, `#505` = **2**). That gap is `D-12`. Fix: `internal/gen/lower/match.go`
  `lowerPatternCond` emitted `stmt.OpGte` for every list pattern but the empty one, so `[x]`
  matched any non-empty list and silently discarded the tail (`recursion_quicksort` printed
  `[3]` at rc=0, no error, no fallback). Now `OpEq` when `core.ListPattern.Tail == nil`, `OpGte`
  when non-nil — `internal/vm`/`internal/bytecode` were never involved (**0** `Pattern` refs;
  control 13 files in `internal/gen`). Quorum BLOCKED **twice**, both rounds premise objections
  the controller measured rather than forwarded (V-A..V-I), resolved via the narrow-refinement
  carve-out. **Disclosed residual, NOT a hole**: nested tail sub-patterns (`::(x, [])`) still
  over-match because `lowerPatternCond` does not recurse into `p.Tail` — named in the doc, the
  plan, and confirmed by the evaluator; wants a follow-up doc.
- **m-bytecode-vm-parity-bugs** — **[UNPARKED 2026-08-22 — D-25, Mark attended: A2 = semantic
  effect extraction via a NEW CLI SURFACE emitting the resolved effect row (feasibility spike
  first; compiler packages stay OUT of the harness); route the A1/A2 lane to its second design
  round. Was iter-114: DOC RE-SCOPED A+B then PARKED `needs-human-review`
  on the A2 classification question; ONE decision needed from Mark, options A/B/**C recommended**,
  in the doc's header box]**. Not "output divergences" — **three soundness bugs**: **#505** the VM
  ignores fixed-length **list-pattern length** (`[x]` matches any non-empty list → `recursion_quicksort`
  silently returns `[3]`; general at n=1,2,3; no error, no fallback), the `arith-on-Closure` dispatch
  family (`array_basic`, `array_grid`, `module_let_helpers`), and **#506** the VM→eval fallback
  **restarts the program after committed effects** (`tar_gzip_reader` prints its header twice;
  duplicates a `println` here, would duplicate an FS write / HTTP POST). Fresh iter-114 data at
  `33be8f5a7`: **149 / 2 / 7 / 16** — and the **MATCH headline is inflated by 6 fake rows**: the
  harness passes `--quiet` (`verify_bytecode_parity.go:235`), which suppresses the fallback warning
  (`run_helpers.go:375`), so the evaluator re-runs and matches itself while the VM never ran the
  program (6/6 controller-verified). Quorum BLOCKED twice; both round-0 objections were real and
  reproduced first-party before adoption. **Recommended unblock (C): split — sprint `#505` (B1+B2)
  now** (root cause settled, minimal repro, acceptance test is a pattern table independent of the
  harness) and send the A1/A2 harness lane for a 2nd design round on semantic effect extraction.
  *Superseded below: [RE-SCOPED + Lane-B PARKED
  for Mark, iter-102 Gate-2 fresh data at `64f1e2924`]* — live `verify_bytecode_parity.go` showed
  **150 MATCH / 2 NON_DET / 6 DIVERGE / 16 EVAL_SKIP** (86.2%). Fresh per-file categorization (doc's
  Reality-check-refresh box is authoritative): old bug #3 `string_parsing` now MATCHes; the character
  changed. **Lane A (loop-fixable, ~1d, NO bytecode internals)**: eval `builtinShow` tuple gap
  (`pattern_sugar` — eval wrong, VM correct) + parity-harness honesty (exclude timing `xml_walk_perf`
  + Net `claude_haiku_call`; count clean eval-fallback `tar_gzip_reader` as VM_BRIDGE not DIVERGE).
  **Lane B (DECIDED by Mark 2026-07-27 (attended): GO A+B — full scope INCLUDING the VM codegen
  soundness fixes; was PARKED-for-Mark (h))**: `recursion_quicksort` (VM **silently returns `[3]`** — soundness)
  + `array_basic` (`GET_TAG on Closure` dispatch) are genuine bytecode-VM codegen bugs in the
  "Go/bytecode compilation story" Mark hand-holds; a silent wrong result exceeds the *"3 small output
  divergences"* delegation → Mark scope call before routing Lane B. On his decision: route Lane A (and
  Lane B if in-scope) to the ROTATION designer (next = `codex:gpt-5.6-sol`) → quorum → planner.

**HUMAN-LED lanes (Mark, 2026-07-14 — the loop keeps HANDS OFF unless a sub-item is
explicitly delegated in this queue):**
- **The coordination dashboard** (`ui/` Collaboration Hub + internal/server): six build passes
  since v0.4.4, feature-complete but architecturally unfinished (simplification 1-of-6 PRs done;
  EvolutionTree 2,061 lines), unmaintained since Feb. Day-to-day coordination-watching has moved
  to `ailang chains` CLI + issue #329. Mark hand-holds; in/out decided at the release gate.
- **The Go/bytecode compilation story** (`internal/gen/*`, `internal/vm`, `internal/bytecode`):
  strategy of record = evaluator-first + Statement IR + bytecode (Tier B perf path, ~95% parity);
  Go source emission DEMOTED to diagnostic projection; emit-go-v2 PAUSED (415 symbols short, open
  design-committee question). Mark hand-holds; posture ratified at the release gate.
  (Exception already delegated: m-bytecode-vm-parity-bugs — 3 small output divergences — stays
  in the clause-2 queue.)

**v1.0 RELEASE-GATE AUDIT (one human session, Mark + controller, when the gating queue is
empty — the bundled in/out + ratification calls):**
1. Stability tier assignments (parked since iteration 5: std/net, crypto, jwt, xml, zip,
   process, CLI watch/serve-api).
2. Dashboard: in/out of 1.0 (evidence: dashboard-lineage review 2026-07-14; keeping it OUT costs
   nothing user-facing — chains CLI covers the live path; IN = commit 4–7d to finish the
   abandoned simplification).
3. Compilation posture: ratify evaluator-first for 1.0 (`--bytecode` labeled experimental).
   **PRE-DECIDED (Mark 2026-07-14): emit-go-v2 FROZEN** (contracts projection stays live) —
   formal ratification here; VERIFY the contracts codegen caveat (gen/golang/contracts is live
   via --verify-contracts — if 1.0 materials mention contract compilation, that ships).
4. Boundary split: **PRE-DECIDED (Mark 2026-07-14): m-arch-boundaries ADOPTED** — Phases 1–3
   pre-1.0 (queued, loop-executable), Phase 4 physical restructure AT this boundary (schedule it
   here), separate repos rejected. Audit confirms Phase 1–3 landed + green, then greenlights
   Phase 4 as the first v1.1 act.
5. Effect-scope-params re-score (standing flag from iteration 6).

**The v1.1 arc (spine, Mark 2026-07-14):** *"the bytecode VM grows up, proven by a game"* —
m-arch-boundaries Phase 4 (physical core/apps/tools split) → the game engine as typed effects
(`m-game-engine-effects`, [planned/v1_1_0](planned/v1_1_0/m-game-engine-effects.md)): Stapledon's
Voyage revived on `!{Render, Input, Clock}` host effects, evaluator-first, with the game's
frame-budget as the VM's standing flagship KPI. Go source codegen stays demoted (emit-go-v2
frozen; contracts projection live).

**Mission-infrastructure backlog** (improves HOW the loop runs; not a v1.0 gate):
- **[NEW 2026-07-31, iter-124 retro] m-vuln-allowlist-expiry-warning** (P3, ~2–3h, no design doc
  needed — a flag on `tools/govulncheck-filter`): the allowlist gate fails **ON** the expiry date
  with zero advance warning, so all 8 Ollama entries — which shared a single `expires: 2026-07-31`
  — fired together at midnight and took dev CI red before any human or loop was looking. Iter-124
  re-armed them (`73f4e38bf`) after verifying upstream is still unpatched, but the *mechanism* is
  the finding: an expiry that only ever announces itself by breaking `dev` converts routine
  hygiene into an outage. Fix shape: warn (non-fatal) when any entry is within N days of expiring
  so it surfaces in a normal green run, and have the nightly or `post-release` path report
  upcoming expiries; optionally stagger the dates so a whole cohort cannot fire at once. Note the
  file header already claims "Reviewed: post-release skill prompts a check of expiries" — that
  prompt only fires around releases, which is not often enough to catch a dated fuse. **Demand
  evidence: 1 real outage (this one)** — deliberately P3, and it should NOT be oversold beyond it.
- **[S1 LANDED 2026-08-03 (iter-135) — PR #577 → squash `ab209fcbf`, dev CI GREEN on the merge (19 checks, 0 failures, SHA-addressed); evaluator sonnet 92/100 r1. Patch adopted VERBATIM in a credited commit (`fd838911f`); shared core `ai_stream_core.go` `{record, failLoud}`; fail-loud latch (public prefix `unencodable stream chunk`) + bounded INERT drain (256 chunks / 1 MiB, NO panic/sentinel); 14-row matrix + budget-contract pinning after a controller mutation probe caught the budget test self-referential. Follow-up `#578` filed; `IncompleteStream` question PENDING with @arniwesth on #546. **S2 REMAINS (~1.5–2d)**: example file, both false open-row repairs + CI guard, prompt/μRAG/website, exhaustiveness guard — plan already cut; queues by normal ordering. Was:] [IN-SPRINT 2026-08-03 (iter-133) — PARK CLEARED; DOC REVISED + SPRINT PLANNED, PR #562.**
  Doc → [planned/v0_32_0/m-recorded-stream-api.md](planned/v0_32_0/m-recorded-stream-api.md)
  (moved from `v0_31_0/` — **v0.31.0 SHIPPED 2026-07-29**, so the target and every
  `Since: "v0.31.0"` string were stale; now **v0.32.0**), plan →
  [planned/v0_32_0/m-recorded-stream-api-sprint-plan.md](planned/v0_32_0/m-recorded-stream-api-sprint-plan.md),
  state `sprint_M-RECORDED-STREAM-API-S1.json` (validator **rc=0** after the controller fixed
  `S1_M1`'s `estimated_loc == 0`, which the validator rejects as an unfilled placeholder — see
  `#563`). Designer `claude:claude-fable-5` (rotation advanced), planner **opus**; executor and
  evaluator **NOT fired — S1 execution is the NEXT iteration's Gate 3**, a deliberate deferral
  because the doc's estimate grew to 5.5–6.0 d and a 4–5 day sprint must not be rushed into an
  iteration tail (iter-125 precedent). **Cut into TWO sprints**: S1 = 3.75 d / 5 milestones (file
  split for headroom → verbatim credited patch adoption → shared core → fail-loud + inert bounded
  drain → test matrix → contract text); S2 ≈ 1.5–2.0 d (example, both false "open row" repairs +
  CI guard, prompt/μRAG/website, exhaustiveness guard). **THE FIND: the designer's own
  sentinel-panic abort is UNSOUND ON WASM** — `cmd/wasm/effects.go` hands `onChunk` to JS as a
  `js.FuncOf` wrapper and awaits a promise, and Go's own `syscall/js` doc plus that file's comment
  agree that such callbacks run on a NEW GOROUTINE, so a `recover` scoped to the `StepWithStream`
  call cannot catch the sentinel and an unrecovered panic there is fatal to the module; the file
  is `//go:build js && wasm`, so the proposed containment test would have passed green while the
  WASM path crashed. Controller refuted it, planner confirmed it against the Go source and ruled
  **drain mode ships, the sentinel does not** — every element of Mark's option (c) survives
  (locality, no interface change, 256-chunk/1-MiB budgets, `drain_exhausted` trace, preserved
  typed `Internal`); only a post-failure WALL-CLOCK bound is lost, on a path unreachable for any
  `StreamChunk` constructible today. **That loss is the ONE open decision for Mark.** Verified:
  the reference patch applies to current dev **rc=0** (all 5 touched files byte-unchanged since
  the v0.31.0 tag; controls `--reverse` rc=1 and re-apply rc=1), and the four offered tests pass
  against dev (4/4; package-wide minus two live-network tests = **483 PASS / 0 FAIL**). Was:]
  **[PARKED needs-human-review 2026-07-31 (iter-124) — DESIGN DOC LANDED, quorum BLOCKED ×2, ONE
  SCOPE CALL OWED BY MARK.** Doc was `planned/v0_31_0/m-recorded-stream-api.md`,
  commit `d85934df4`. **Ghost discipline: NOT a ghost — the claim is REAL and understated.**
  Verified first-party at HEAD `130ad1da2` on a freshly rebuilt binary, positive control beside the
  negative probe: an `{IO}` rendering callback checks (rc=0), an `{FS}` recording callback FAILS
  rc=1 with `incompatible closed rows: r1 has extra labels [IO], r2 has extra labels [FS]`;
  `StepResult` carries no chunks; `std/io` has no file write. So live-streaming and chunk-recording
  ARE mutually exclusive. **ADR-009 line 134 independently reproduces the same result against
  v0.30.0** — two parties, same finding. **BONUS DEFECT, folded in as a milestone**: the repo
  TEACHES the opposite in two live places — `std/ai.ail:324` ("the callback's effect row is open")
  sits directly above the closed-row declaration contradicting it, and
  `examples/runnable/ai_streaming.ail:40-42` promises websocket/TUI/metrics side-channels that
  cannot type-check; adopting the recorded sibling does NOT widen the row, so both stay false
  unless fixed. Designer `codex:gpt-5.6-sol` (rotation advanced). Verdict **ADOPT with
  productionization**, routing judged **core, not extension**. **Quorum R1 BLOCKED** (gemini: an
  UNVERIFIED premise — fair, resolved by the controller running the 4 offered tests outside the
  sandbox, all PASS; gpt5-6-sol: silently skipping unencodable chunks contradicts "lossless" and
  Critical Principle 2 — accepted, designer chose FAIL-LOUD, est. 3–4d → 4–5d). **R2 BLOCKED**
  (gemini: my own `-run` isolation was too narrow — resolved,
  `go test ./internal/effects -skip TestNetHTTPRequestBytes_RoundTripSHA` rc=0 with **658 PASS**,
  the patch breaks nothing; **gpt5-6-sol: the fail-loud drain is UNBOUNDED** — and its fix's own
  conditional FIRES, because `internal/effects/ai.go:87` takes **no `context.Context`** and has
  **7 implementers across 6 files including `cmd/wasm`**, so closing it destroys the "purely
  additive" property the ADOPT verdict rests on). **Deliberately NOT force-passed and NOT taken
  under the narrow-refinement carve-out** — that carve-out covers only objections leaving the
  design DIRECTION intact, and this one changes scope; Standing rule 2 → park. **DECIDED by Mark 2026-08-03 (attended): OPTION (c) — bound the drain LOCALLY inside the
  recorded op, no interface change.** Endorsed independently by the AUTHOR (@arniwesth,
  #546 comment 2026-08-01): `StreamChunk` is a SEALED interface (unexported marker,
  exactly 3 implementers, all handled by `encodeStreamChunk`; the one variable-forwarding
  call site is nil-guarded) → the fail-loud drain trigger is UNREACHABLE at current code,
  so (c) is proportionate and (b) is over-engineering — the working iteration inherits
  that evidence from the issue comment, verify the sealed-interface claim first-party
  before relying on it. → ROUTABLE. [Was: **THE ASK (a/b/c in
  the doc header): (a) land now with a documented unbounded-drain caveat, (b) take the `AIHandler`
  cancellation change as a blocking dependency, or (c) bound the drain locally inside the recorded
  op with no interface change — controller's read is (c), avoid (a).**] Quorum metered $0.1086 of
  the $5 ceiling. Was:] **[NEXT-ON-RESUME #1, Mark directive 2026-07-31 (attended): MOTOKO DEMAND — pick FIRST at the
  2026-08-03 re-arm, ahead of the offloads] m-recorded-stream-api** (`ailang#546`, filed by
  arniwesth 2026-07-31 — the STRONGEST demand class: a real external consumer with a WORKING
  IMPLEMENTATION OFFERED): `std/ai.stepWithStream`'s contract (unit-returning `{IO}`-closed
  callback; `Result[StepResult, AIError]` carries no chunks) makes live-streaming and
  chunk-recording mutually exclusive — motoko's deterministic-replay testing needs BOTH at once.
  Arni ships the reference implementation as a PR on his fork —
  **https://github.com/arniwesth/ailang/pull/2** (branch `spike/motoko-009-prototype-v031`) —
  plus a patch verified `git apply --check`-clean against v0.31.0. REQUIRED design context
  (Arni, Discord 2026-07-31): the two motoko DST ADRs this must serve —
  Project 009 Deterministic Test-World Architecture:
  https://github.com/arniwesth/motoko_agent/blob/arniwesth/mot-44-motoko_dst_execution_primer/.agent/projects/009_motoko_dst_execution/ADR-001-deterministic-test-world-architecture.md
  and Project 007 DST definition/taxonomy:
  https://github.com/arniwesth/motoko_agent/blob/arniwesth/mot-44-motoko_dst_execution_primer/.agent/projects/007_dst_consolidation/ADR-001-motoko-dst-definition-and-taxonomy.md
  — the designer/quorum read BOTH before judging the patch. The iteration should
  (1) ghost-discipline the repro at HEAD, (2) evaluate ADOPTING the offered implementation
  (review-the-patch lane — do not reinvent; quorum reviews the DESIGN it embodies, incl. the
  core-vs-extension routing call on a std/ai surface change), (3) credit authorship in the
  commit. Ack posted on #546. Note: the five upstream motoko PRs (#73/#76/#96/#97/#98) are all
  green as of 2026-07-31 (the #98 AILANG_REF guard fixed attended, `324d86d`) and await ARNI's
  review — not our work.
- **[LANDED 2026-08-04 (iter-136) — PR #580, six commits; ALL FOUR MILESTONES + THE FLIP.
  `MISSION_PLANNER_MODEL` now defaults to `codex:gpt-5.6-sol`, so opus is controller-only
  (Mark quota-offload #1). Shipped: `tools/launchd/derive-planner-lane.sh` (Bash 3.2, pure text,
  fail-closed) + 12 fixtures; the driver's role-generic D3 probe loop + the `mission-<name>.env`
  rollback source that D6 already claimed existed; the skill's MANDATORY Gate-3 step 1b; and the
  line-286 flip, gated on a REAL codex planner rehearsal the repo's own validator passed rc=0.
  Doc + plan → `implemented/v1_0_0/`. **Controller mutation probe found a SURVIVOR** — neutering
  the `__UNPARSABLE_PATH_ENTRY__` sentinel changed nothing, because fixture (j) is caught one arm
  earlier by the path-shape check, so a whole arm had zero coverage while the matrix read green;
  fixture (n) closes it. **The plan's own AC9 is VACUOUS** (it pins `opus`, which pre-flip is also
  the built-in default, so it passes whether or not the file is sourced) — only a sentinel makes
  it a measurement. **`#486` non-regression re-proven**: the pinned probe fails with a 400 naming
  the model while the model-LESS control returns rc=0 `ok`. **Engagement rate, not oversold**: 41
  planned docs carry a Files heading, 40 fail closed, exactly ONE declares `**Planner-Lane**`
  today — the lane engages on newly authored infra docs only. **World driver deliberately NOT
  synced** (the plan's recommendation assumed World was kill-switched; it is live) — World fails
  closed to opus via two independent guards; cross-mission message sent; the sync is a PENDING OPS
  CALL for Mark. ⚠ **Evaluator NOT fired — a sonnet judge is owed and is the first task of
  iter-137.** Was:] [(1) DOC + SPRINT PLAN LANDED 2026-07-31 (iter-125);
  EXECUTION was parked to the 2026-08-03 re-arm — PLAN-READY, NO human decision owed.** Doc →
  [implemented/v1_0_0/m-planner-codex-lane.md](implemented/v1_0_0/m-planner-codex-lane.md), commit
  `e980c72d5` (rev 3, 685 lines); sprint plan `b43af2a3e` — 4 milestones / ~307 LOC / ~8h,
  `validate_sprint_json.sh` **rc=0** (the sprint JSON is gitignored local rig state: it survives
  on this machine but is NOT in the repo). Designer `claude:claude-fable-5` (rotation advanced),
  planner **opus**; executor/evaluator **NOT fired** — deliberate, see below.
  **THE FIND: quorum R1 caught a `declare -A` in the proposed driver loop that would have WEDGED
  THE LOOP on the next launchd fire, before any role spawned.** Controller-confirmed first-party:
  the driver's `#!/usr/bin/env bash` resolves to `/bin/bash` **3.2.57**, and **no 4.x bash exists
  on this rig at all** (`/opt/homebrew/bin/bash` and `/usr/local/bin/bash` both absent — which is
  precisely why the launchd `PATH` listing those dirs looked safe). Live repro emits both
  `declare: -A: invalid option` and bash-3.2 arithmetic-evaluating the model name
  (`gpt-5.6-sol: syntax error: invalid arithmetic operator (error token is ".6-sol")`); the
  driver's own shebang form prints `ASSOC UNSUPPORTED`. Second `declare -A` defect here after
  iter-107. The designer then found a **second** Bash-4.0-ism neither reviewer caught
  (`${role,,}` → `bad substitution`), also reproduced first-party. **Quorum BLOCKED ×2**, design
  DIRECTION never contested in either round: R1 (gemini: bash 3.2; gpt5-6-sol: D2 enforced by
  controller judgement rather than deterministically) → revision; R2 (gpt5-6-sol: the classifier
  was **denylist-** not allowlist-based, so unlisted/future language paths silently received the
  same-model planner+executor pairing D2 claims to prohibit; gemini: `derive-planner-lane.sh`
  ignored `$MISSION_PLANNER_MODEL`, **breaking the doc's own D6 rollback**) → **narrow-refinement
  carve-out**, both reviewers' verbatim `proposed_fix` applied, no controller-invented resolution
  substituted. **Then the opus planner refuted FIVE premises**, the two decisive ones
  controller-verified: **R1 — D6's "rollback is one env var" is a NO-OP FOR V1, and BOTH quorum
  rounds accepted it** (`~/.config/ailang/mission-v1.env` does not exist — control:
  `mission-world.env` does — and the driver sources the profile file only when `$MISSION_PROFILE`
  is set, while the V1 plist sets only `PATH`/`HOME`; it works for World, the mission the doc
  called insulated). **R4 — "World blast radius = zero until synced" is FALSE**: `ailang-world`
  has **no repo-local `.claude/skills/` directory at all**, so it loads the GLOBAL skill copy D4
  mandates editing, reaching World at its next fire. R4 refutes a sentence in the controller's
  own commit message and is a **rule-3b SCOPE error by the controller** — the *driver* really is
  byte-identical (verified), but that fact was cited for a broader *blast-radius* claim it does
  not support. R2 — AC1/AC2/AC7 unrunnable as written (the overlap guard at line 334 yields
  BEFORE the dry-run exit at 351, so the ACs reproduce the very vacuous-pass class D5 was written
  to kill). R3 — AC4 can never pass (the main checkout is chronically rig-dirty). R5 — the
  controller's Files-to-Modify ruling is necessary but not sufficient (fixture prose backticks
  `internal/parser/...`, so a literal "every declared path" reader still fail-closes the doc).
  **WHY EXECUTION IS PARKED — a scheduling call, not a blocker**: `tools/launchd/` has **ZERO CI
  coverage** (no workflow references it, no shellcheck/`bash -n` gate, no test file —
  control-verified, since the same grep DID find `tools/` in two workflows), so a green Gate 3b
  says nothing about the driver and the ACs are the only real gate; the point of no return is
  **M2, not M4** (it edits the file launchd fires every 5400 s, and its revert is a *code* revert
  — an env var cannot un-break a driver that fails to parse). Landing unattended on a Friday
  before a quiet weekend buys no capacity, because the offload's whole purpose is stretching the
  week Monday-to-Monday — which is the date Mark set. Also surfaced: only **1 of 102** planned
  docs carries a `Planner-Lane` field and 8 heading spellings exist, so the lane engages only on
  newly authored infra docs (stated plainly, not sold as immediate); the header template lives in
  `design-doc-creator/resources/design_doc_structure.md`, not its `SKILL.md`; and a **third**
  git-tracked skill copy (`.agents/`, 44067 B vs 70544 B) is already drifted (iter-123's `#544`).
  Quorum metered **$0.189** of the `$5` ceiling. Was:] **[NEXT-ON-RESUME, Mark directive 2026-07-30 (attended): QUOTA OFFLOADS — pick these FIRST
  when the loops re-arm 2026-08-03 07:00, before M4b and effect sprints 3/4]** ⚠ **GATE EXPIRED
  2026-08-03 07:00 and item (1) was DEFERRED ONE ITERATION by iter-131, on evidence, not
  preference**: `m-planner-codex-lane`'s HIGH-risk milestone edits `tools/launchd/mission-control.sh`,
  and `#558` measured that launchd executes that file from the **stale main checkout** — so the
  sprint would land green, pass CI, and change nothing about which model the planner actually runs
  on, while the report claimed the capacity was gained. It stays the **next pick** and needs no
  re-greenlight; route it after the checkout reconcile, or land it with an explicit inert-until-
  reconcile note. — two sibling
  recipe/lane items (mission infra, not language changes; neither gates v1.0 but both protect
  its timeline — capacity multipliers land before capacity consumers):
  **(1) m-planner-codex-lane** — route `MISSION_PLANNER_MODEL` through the hardened
  `PROVIDER=codex` spawn recipe (M1b machinery; stdin `< /dev/null` + directive-delivery
  assertion per codex-spawn-recipe-false-greens): planner moves to the ChatGPT-subscription
  bucket, opus remains controller-only. **(2) m-evaluator-gemini-review-lane** — the #399
  Phase-2 follow-on Mark greenlit 2026-07-27: evaluator-as-reviewer over a PRE-MERGE PUSHED
  sprint branch via managed-agents clone-over-egress (CI stays the test oracle;
  generator≠judge preserved — gemini judges codex/opus executors; sonnet remains the fallback
  evaluator when the gemini lane is unavailable). Rationale: the opus bucket dried Thursday
  this week at ~55% duty cycle; these two move the remaining Anthropic-heavy sub-agent roles
  off-bucket so the week stretches toward Monday-to-Monday.
- **[GATE MET + DOC LANDED, PARKED `needs-human-review` on `D-13` 2026-08-13 (iter-188)** — the
  evidence gate below is **DISCHARGED**: `grep -c "driver pin: running committed origin/dev @"`
  = **5** in `/tmp/ailang-mission-control.log`, `DRIVER PIN FAILED` = **0** with its
  known-positive control firing (the string exists once in `mission-control.sh`; log readable at
  **126** `mission iteration starting` lines), and fires **185/186/187** each pinned → started →
  completed with charter records. ⚠ My first control was BROKEN — I grepped `UNPINNED`, which is
  **0** in `pin-root.sh` too, so the log's zero was uninformative until I read the emitter and
  greped what it actually writes. Doc: `planned/m-driver-pin-rollout.md` (434 lines, **24**
  verification rows), designer `claude:claude-fable-5` (rotation, 2 bounded passes). **Gate 2
  corrected the scope before the designer ran**: `nightly-lang-eval.sh` is **not** an entry point
  (no plist installed or in-repo, no crontab, no log — control: `nightly-eval.plist` in both), so
  the "five" below is **four**; and those four split into repo-rooted (`nightly-eval.sh:28`,
  `os-rotation-filler.sh:12` derive `REPO` and `cd`) vs script-only (`mission-recovery.sh`,
  `rig-watchdog.sh` — no `REPO=`, zero repo reads). **The per-checkout onboarding prerequisite
  below applies to ZERO of them**: `claude` refs are 0/0/0/0, `mission-recovery.sh:50` being a
  comment (control: `mission-control.sh` = 24). **Two hazards found and confirmed first-party**:
  `dev.ailang.mission-recovery.plist` injects `MISSION_NAME=v1` at `StartInterval` **240**, so the
  helper's default pin dir would have `checkout --force`d **the live V1 mission worktree every
  four minutes** under a running iteration (fix: explicit `AILANG_DRIVER_PIN_DIR` per driver); and
  `make test-launchd-drivers` is **9 passed / 26 failed rc=2** inside a pinned env vs **17/0
  rc=0** with `AILANG_DRIVER_*`/`MISSION_WORKDIR`/`MISSION_NAME` sanitized — the acceptance gate
  is red at base in exactly the environment the loop runs in. **Quorum BLOCKED ×2, both reviewers
  present both rounds, all four objections measured not forwarded** (`$0.0794` + `$0.0984`). R1
  resolved by one bounded revision (D6's self-contradictory "continue and `exit 0`" for a script
  whose line 114 is its only exit; and the fetch-timeout objection whose premise was REFUTED —
  `pin-root.sh:103/114/116` already bounds it via `_pin_bounded` at 120s — while its substance
  was adopted as per-driver bounds below each cadence). **R2 BLOCKED on two NEW objections and
  measuring them made both WORSE than filed, which is why the narrow-refinement carve-out does
  NOT apply**: (V23) `os-release-snapshot.sh:24` and `publish-unified-dashboard.sh:26-27` resolve
  their root from **`$0`, not cwd**, so D2's `cwd=$REPO_DURABLE` strategy cannot work and
  gemini's own `proposed_fix` presumes an audit result that is false; (V24) the automatic
  source-clone updater **does** exist — `os-rotation-filler.sh` `git pull`s at `:197/398/426/458`
  every 45 min, which is why the clone reads **0 behind** — refuting gpt5's literal premise, but
  **M2 pins that very filler**, so the rollout would delete the only delivery path for every
  later milestone. Both need a controller-invented resolution → Standing rule 2 binds.
  **`D-13` on `#635`, one word: (A)** exclude the filler and keep it as updater; **(B)** move the
  source-clone fast-forward into `pin-root.sh`; **(C)** keep scope, re-root durable writes through
  `$AILANG_DRIVER_SRC`.] m-driver-pin-rollout**
  (`#558`; mission infra, not a language change; does not gate v1.0). **Prerequisite already
  shipped**: PR `#666` (2026-08-12, attended) adds `tools/launchd/lib/pin-root.sh` and wires it
  into `mission-control.sh` ONLY — the driver re-execs from a worktree pinned to committed
  `origin/dev`, moving the driver, the skill and the charter together, and posting *"driver ran
  UNPINNED"* on both human channels when the pin fails. This item is the rollout to the
  remaining **five** entry points: `nightly-eval.sh`, `nightly-lang-eval.sh`,
  `mission-recovery.sh`, `os-rotation-filler.sh`, `rig-watchdog.sh`.

  **Why one root, not five patches** (Principle 3): everything a fire reads hangs off `REPO` at
  `mission-control.sh:40`, so the driver (`$0`), the skill (`cwd`) and the charter (`cwd`) go
  stale *together*. Measured four times — `#556`'s retired qwen3.5 running 24/24 (iter-131), a
  stale skill (iter-128), a stale charter (iter-129), and `564cc4640` inert on V1 at 12 commits
  behind (2026-08-12) while both sibling missions had it. Two one-time human reconciles, zero
  durable fixes, until `#666`.

  **INERT UNTIL RECONCILED, and say so rather than reporting the capability as gained** — the
  same trap that deferred `m-planner-codex-lane` at iter-131. `#666` cannot take effect until
  the shared clone receives it once, and that is a human branch op (the standing fast-forward
  authorisation above does NOT cover it: local dev is 1 ahead, not 0). **Order matters:** merge
  `#666` FIRST, reconcile SECOND — reconciling first brings the lane fix but not the pin, and
  the clone simply starts drifting again.

  **PER-CHECKOUT PREREQUISITE, and it applies to EVERY entry point that spawns `claude`:** a pin
  worktree is a path Claude Code has never seen, and an un-onboarded checkout hangs every
  headless probe (charter V22, `76ee4056c`). The pin now refuses rather than walking into it, so
  a fire stays alive — but it stays *unpinned* until a human runs `cd
  ~/.ailang-driver-pin/<mission> && claude` once. Budget one such action per pinned entry point
  that runs `claude`, and **measure rather than assume which ones do**: `nightly-eval.sh` looks
  exempt (it builds and runs the binary, not `claude`) but that has not been verified, and
  assuming it would repeat exactly the mistake this row exists to prevent.

  **THE GATE (evidence, not a delay):** ≥3 consecutive V1 fires logging `driver pin: running
  committed origin/dev @ <sha>` with a normal iteration completing. Read it from
  `/tmp/ailang-mission-control.log`, not from the file's presence on disk — measuring the wrong
  copy is the same class of error as the bug. **Known limit of that gate, stated up front:** three
  green fires exercise the re-exec, the root move and the fetch, but NOT the failure path, which
  production will not produce on demand. That arm is covered only by `make test-launchd-drivers`
  (37 assertions, bash 3.2 CI job) — so a passing gate is evidence about the happy path alone and
  must not be reported as whole-fix confidence.

  **Not in scope, and not ours to schedule:** `ailang-world` is a **different repo**
  (`sunholo-data/ailang-world`) whose driver is a hand-synced *fork* — 513 lines against this
  repo's 671, with a differently-shaped fallback site. It can never take this by copy, only by
  port. Handed over on the cross-mission channel as `msg_20260812_085746`; World's loop decides
  when. Do not "fix" it from here.
- **[QUORUM-BLOCKED 2026-08-11 (iter-179) — doc LANDED, PR `#657` → squash `0a84f5377`, Gate 3b GREEN
  (4/4 required, `checks=20`, zero not-green). **Reclassified P0**: `#616` is a STATIC EFFECT-SOUNDNESS
  hole, not the DX/error-message item it was queued as — a function with NO effect annotation calling a
  row-poly function TWICE passes `check` rc=0 and EXECUTES IO; a WRONG row (`! {FS}` over `{IO}`) also
  passes. Bounded by measurement: the capability layer backstops (no `--caps IO` ⇒ refused), so it
  defeats capability PLANNING from signatures, not runtime enforcement. `e` is a row **TAIL**, not a
  phantom concrete effect — `Required`/`Declared` are BOTH empty at the label layer while the check
  still fails, which is also the mechanism of the blank `Missing effects:` output. Fix site is the
  effect-checking pass ONLY (parser + `internal/types` are already row-var aware). *Reject at parse* is
  REFUTED — **13** row-var signatures ship in `std`. Quorum r1 BLOCKED, both reviewers present, BOTH
  objections measured first-party and CONFIRMED (rule 3f). **Needs ONE revision + ONE re-quorum; NO
  human input required** — the direction is settled by the stdlib measurement]
  **`#616` m-effect-row-var-unification** ([planned/v1_0_0](planned/v1_0_0/m-effect-row-var-unification.md))

- **[NEW 2026-08-03, Mark directive (attended): TRIAGE BATCH — queue BELOW the reordered top
  picks (#546 → planner-codex → #498 Lane B → evaluator-gemini), never above them]
  m-github-issue-triage-batch** — 12 open issues have ZERO charter mentions (measured 2026-08-03
  ~16:30). Highest-signal: the FIVE `[motoko_agent]` integration defects filed 2026-08-03
  (**#572** step/stepWithStream result omits the required `images` field; **#573** effect checker
  not transitive through function-valued record fields — potential soundness, triage FIRST;
  **#574** `iface` pure-vs-effects contradiction on 12 std/ai exports; **#575** `iface` MOD010 on
  package files with both suggested fixes failing; **#576** unreachable-match-arm warning gap) +
  **#495** (contract/test trio — THIRD surfacing, repeatedly slipping triage) + **#493** (driver
  launchd PATH omits /opt/homebrew/bin — FACT-CHECK against the live codex lane before dismissing
  as stale) + #534/#533/#509/#494/#476. Treatment: triage-lite per issue (ghost-discipline the
  repro, verdict comment on the issue, queue-or-close), NOT a mega-sprint; anything genuine
  soundness (#573 candidate) may then outrank normally per the standing regression rule.
- **[NEW 2026-07-29, Mark directive (attended)] m-outage-triage-lane** (NEW-DOC needed; P3
  resilience — design-only until a second outage recurs; does NOT gate v1.0): when ALL Anthropic
  controller probes fail with a SERVER-error signature (500 Internal / 529 Overloaded — distinct
  from `QUOTA_SIG`, which must keep its existing quota semantics), the driver falls through to a
  bounded `codex exec` OUTAGE-TRIAGE iteration instead of aborting: a distilled Gate-0/1 recipe
  (inbox triage, nightly-regression ghost-probe, bookkeeping, park-everything-else) — NO sprints,
  NO skill edits, NO quorum (planner/evaluator/controller-verification stay Anthropic-only).
  Motivation: 2026-07-29 evening Anthropic 500/529 outage — the 21:22 iteration aborted after 3
  bounded retries (rc=1, correct behavior) and the 13:52 sprint iteration had already stretched
  into the 6h hard timeout under the same degradation; the loop was pause-only because the
  controller is structurally Claude Code (per-model failover exists, per-provider does not).
  Demand evidence: first outage in ~3 weeks of continuous operation → LOW priority by design;
  the doc should reuse the hardened `PROVIDER=codex` spawn recipe (stdin `< /dev/null`,
  directive-delivery assertion) from codex-spawn-recipe-false-greens below.
- **[LANDED 2026-07-28] [world-DEMAND] codex-spawn-recipe-false-greens** (iter-112 Gate 5 — the one skill edit; BOTH fixes applied to `mission-control/SKILL.md`'s `PROVIDER=codex` recipe: `< /dev/null` on both the probe and the real run, plus a directive-delivery assertion (exists · ≥200 bytes · non-empty loaded prompt → `exit 64`). Iter-112 also **used** the fixed recipe first-party for its own codex executor run, and the secret-safe `[ -n "$VAR" ]` env-probe form is documented alongside. Original row: **NEW iter-111**; SKILL FIX, ~10 min, no design doc — apply at the NEXT iteration's Gate 5, since iter-111 had already spent its one skill edit) — the shared `PROVIDER=codex` spawn recipe in `mission-control/SKILL.md` has **two false-greens**, proposed by `world-coordinator` (mission-world iter-26, which cannot edit the shared skill itself) and **corroborated first-party by iter-111 rather than taken on trust**. **Defect 1 — stdin is never redirected**: `codex exec` reads stdin IN ADDITION to the positional prompt, so under a backgrounded launch with an open (never-EOF) stdin it prints `Reading additional input from stdin...` and blocks until the 30-min cap — a hang that *looks* like normal long work. World observed a 39-byte log and zero diff after 6 minutes. **Iter-111 corroboration: that exact line appears once in this iteration's own `codex_out.log`** — the read happens here too; the run only survived because stdin happened to EOF under the harness's background launch. Fix: append `< /dev/null` to the `codex exec` invocation. **Defect 2 — no assert that the directive was delivered**: with the directive file absent, `"$(cat …)"` expands to empty, codex replies "What would you like me to work on?" and **exits rc=0**, and the wrapper reports success for a run that did nothing. **Iter-111 corroboration: a genuine near-miss — this iteration's FIRST `Write` of `/tmp/codex_directive.txt` FAILED** (pre-existing file, "has not been read yet"); had the runner been launched before that was noticed, the spawn would have been a silent rc=0 no-op. Fix: assert the file exists and is non-trivially sized AND the loaded prompt variable is non-empty, aborting loudly otherwise. This is the same vacuous-pass class the mission has closed twice elsewhere (silent z3 skip, silent `t.Skip`): **an exit code reporting success for work never requested**. Meets the ≥2-frictions bar on its own (2 distinct defects, same gap, one iteration) and is independently corroborated here. **Hygiene note, broadcast with it (not a skill defect)**: a shell "is this env var set?" probe written `${VAR:+YES}${VAR:-NO}` **prints the variable's value** — World leaked `OPENAI_API_KEY` into a transcript this way. Safe form: `[ -n "$VAR" ] && echo SET || echo UNSET`. Any preflight env check in this loop must avoid the `${VAR:-…}` form for secrets.
- **[LANDED 2026-07-28] m-docs-gate-not-required** (iter-112, FULL loop in one iteration — planner **opus** → executor **`codex:gpt-5.6-sol`** → evaluator **sonnet PASS 88/100 r1, zero blocking**; PR **#501** → `a3e781b26`, PR **#502** → `cdac5bf04`; sprint plan → `implemented/v0_30_0/`. **`docs-gate` is now a REQUIRED check on `dev`** — `["test","lint","build","docs-gate"]`, applied via `PATCH .../required_status_checks` with the whole protection object diffed byte-wise outside that key. Option (a) as filed: `build`→`docs-build` (collision gone, measured exactly ONE `build` check-run), `on.pull_request.paths` dropped, `docs-changes` detector (`git diff` vs merge base, path list in `.github/docs-build-paths.txt`, no new third-party action), always-reporting `docs-gate` with a **mandatory catch-all arm that exits 1**. **Both branches OBSERVED green before the flip**: #501 build branch (`docs_changed=true`, build success) and #502 skip branch (one `design_docs/` file, `docs_changed=false`, `docs-build` **skipped**, gate success in **38 s**) — then #502 merged under live protection as the canary. Concurrency also rescoped per-ref (the shared `pages` group had already cancelled 4 real PR runs in 60 s; `deploy` keeps the singleton and was proven still green on the dev push). **Refuted premise worth keeping**: a job skipped by an `if:` reports **Success**, not Pending — so requiring `docs-build` directly would have made "not checked" indistinguishable from "passed". Systemic twin → **[#503](https://github.com/sunholo-data/ailang/issues/503)** (`ui-build`, same not-required half, live `/ui` npm Dependabot; carries the deferred least-privilege `permissions` tightening). #497 CLOSED with the verdict + rollback command. Original row: **NEW iter-109**, filed as **[#497](https://github.com/sunholo-data/ailang/issues/497)**; P2, ~0.5d; no design doc needed — the issue enumerates the options) — the docs build **never actually gates Dependabot PRs**, so a docs-breaking bump auto-merges and takes dev red. Two independent causes, both first-party verified this iteration: (1) `dev` branch protection requires only `["test","lint","build"]`, and `dependabot-automerge.yml` runs `gh pr merge --auto`, which merges the moment those go green — on #488 the docs run was **cancelled mid-flight by the merge** (`Deploy Documentation to GitHub Pages | pull_request | completed/cancelled` @ `23c428378`, PR merged `13:00:32Z`, run started `12:45`); (2) `docusaurus-deploy.yml:58` names its job **`build`**, colliding with `ci.yml:315`'s `build` — required contexts match by NAME, so ci.yml's Go build satisfies the requirement while the identically-named docs job is silently irrelevant, i.e. **the gate appears wired in the checks list without gating anything**. `ci.yml:337`'s `docs` job is NOT a docusaurus build (prompt-sync only, and `if: push && ref==dev` so it is skipped on every PR). ⚠ **The naive fix is WRONG and must not be applied**: `docusaurus-deploy.yml` is path-filtered, so making it a required check would leave it permanently pending — and thus blocking — on every non-docs PR. Options in #497: **(a)** rename `build`→`docs-build` (worth doing regardless — the collision is a latent trap) + an always-reporting wrapper job that internally skips when no docs paths changed, which CAN safely be required; **(b)** exclude the `/docs` npm ecosystem from auto-merge (cheapest stopgap, costs the batching auto-merge exists for); **(c)** have auto-merge wait explicitly on the docs run. Blast radius already demonstrated twice: #488 took dev red ~7h, and #490's react/recharts bumps merged 60s later with a docs build that **never once ran green** (verified fine only because iter-109 built it locally). Mitigation shipped in `4b757f63d`; the gate itself is untouched.
- **[LANDED 2026-07-28 — iteration 113]** m-nightly-flake-guard (PR #504, squash `038d9322d`, dev CI green; evaluator sonnet **PASS 87/100 r1, zero blocking**; doc + sprint plan → `implemented/v1_0_0/`). Shipped: classifier extracted to `tools/nightly_classify.py` (behavior-preserving, byte-identical over all five surviving nights) · history off `/tmp` onto a JSONL with atomic write + ownership-checked lock (PID + random token, `os.kill(pid,0)` liveness gating any steal) · trailing-window solidity rule (W=5, MIN 2 nights/4 trials) with `suspected-flake`/`insufficient-history` labels where only `REGRESSION` reaches `--type bug` · replay over the real corpus: **`filed=5 → guarded=2, suppressed=4`**, non-vacuous because **#483 `higher_order_functions` (4/4 solid) still pages the SAME night** and `csv_to_json_converter` newly escalates on 07-27. Controller mutation-proved the guard in **three** directions (suppress-everything → 3 red incl. the same-night test; suppress-nothing → 6 red; liveness-blind steal → 2 red incl. `test_Lock_stale_but_alive_holder_is_not_stolen_from`), each reverted byte-identical (`sha256 a53f1e3a…`). CI wiring is real, not vacuous: **`make ci` is run by NO workflow**, so the tests attach directly to `ci.yml`'s test job + a step failing under 20 `--- PASS:` lines — and the job log proves it RAN (40 PASS in Actions). Deviation: `consec >= K AND not already-regressed` replaces the doc's literal `consec == K`, which loses the escalation forever on a missed night. ⚠ **Landing action**: run `--bootstrap` once before the next 05:00 nightly, else night 1 is loudly DEGRADED with zero issues filed. Was: (**DOC LANDED + quorum-cleared iter-106** → [planned/v1_0_0/m-nightly-flake-guard.md](planned/v1_0_0/m-nightly-flake-guard.md), commit `6ad39b863`; P2, ~1.3d, 4 milestones) — the nightly eval regression detector has **no variance/flakiness guard**: it compares this run against the single previous run (N=2 trials each) and files a GitHub issue on any pass→fail flip, so a benchmark that is merely *bimodal on a local model* generates recurring false alarms. Evidence: `json_parse` on `opencode-qwen3-5-35b-a3b-mxfp8` produced **four** nightly issues (#286, #292, #480, #485), all closed as noise; banked history 1/2 → 2/2 → **0/2** → 2/2 → **0/2** (07-23…27, rag_on), i.e. the 07-25 and 07-27 alarms are the SAME flip and 07-25 self-recovered with zero action. Each false alarm costs a Gate-0 triage slot and, left open, reads to external viewers as an unresolved regression (#417). **Design decides**: history off `/tmp` (the amnesia's cause — only 6 nights survive) onto a classifier-owned JSONL with an explicit idempotency/atomicity contract + ownership-checked lock; trailing-window solidity (W=5, MIN 2 nights/4 trials) with a label-agnostic K=3 escalation backstop, replayed on the real history to show #480/#485 suppressed while the solid→broken control still pages SAME night; new `suspected-flake` / `insufficient-history` labels where only `REGRESSION` reaches `--type bug` (which is what creates the issue); explicit `--bootstrap` so an absent history file is ALWAYS the loud DEGRADED state. **Quorum**: designer `claude:claude-fable-5`; reviewers `gpt5-6-sol` + `gemini-3-1-pro` + controller opus; R1 blocked → designer revision; R2 blocked on new narrower objections → **narrow-refinement carve-out** (both reviewers' `proposed_fix` applied verbatim). ⚠ **CORRECTION (V8, this row's own prior text was WRONG)**: the earlier bullet *"fold in: compare like-for-like CONDITIONS — #485 compared against yesterday's `_rag_on` while today produced both"* is **false and is dropped** — both sides of the comparison already use the rag_on arm (`nightly-eval.sh:233` and the `*_rag_on/agent` glob at `:245`). Do not design a fix for it. **NEXT**: route to sprint-planner.
- **m-mission-adaptive-multiprovider-routing** ([planned/v0_30_0](planned/v0_30_0/m-mission-adaptive-multiprovider-routing.md); EXPANDED 2026-07-14 per Mark — quota now the binding constraint) — the heterogeneous model FLEET. **[Phases A+B LANDED 2026-07-14, iteration 28]**: Phase A (quota-aware multi-candidate probing in the driver) landed `3bee6b6df` direct-to-dev by the interactive session + verified/hardened by the sprint; Phase B (design-doc QUORUM: gpt-5.6-sol + gemini-3-1-pro-via-Vertex-ADC + Claude controller in-session, reject-by-default, N−1 named-absence degrade, budget-capped) landed PR #383 → `1186a48e6`, eval PASS 94/100 round 1 — `ailang design-review`/`design-quorum` live, artifacts under `.ailang/state/mission-quorum/`. REMAINING (opt-in as evidence accrues): Phase C cross-provider executors (re-scoped ~1d, audit binding); Phase D local-GPU lane (~2–3d); Phase E full (provider, model)×task-class assignment (~3–4d). Quorum-on-sprint-plans deferred (hook scoped to design docs). Requested + prioritized by Mark.
- **m-arch-boundaries Phases 1–3** **[LANDED 2026-07-20 (iter 68) — PR #420 squash `ee97fada6`; evaluator (sonnet, generator≠judge vs opus executor) PASS 88/100 r1; doc → [implemented/v0_30_0/m-arch-boundaries.md](implemented/v0_30_0/m-arch-boundaries.md)]** — `scripts/check_boundaries.sh` self-testing import-boundary CI gate (Rule 1: no core→dashboard; Rule 2: no dashboard→compiler-surface except via `internal/embed`; MODULE-vs-`go.mod` drift guard; `eval` excluded from Rule 2 for the sanctioned `eval.Value` bridge type, documented) + `make check-boundaries` + CI step + `ARCHITECTURE.md`/`CLAUDE.md` boundary docs + `.github/CODEOWNERS`. **No physical restructure** — Phase 4 (`git mv` core/apps split) reserved for the v1.0→v1.1 boundary; dual-release-tracks out of scope. Planner (opus) corrected 5 stale doc premises; executor (opus) caught 2 real defects (wrong module anchor → false-pass; `server→eval` bridge import); `metered=$0.00`. **Follow-on queued: m-arch-boundaries-eval-exclusion-tighten** (evidence-gated — tighten the `eval` exclusion package→file level; only 1 file uses it today).
- **m-mission-quorum-agentic-verify** ([planned/v0_30_0](planned/v0_30_0/m-mission-quorum-agentic-verify.md), 2026-07-14; P1) — the shipped text quorum REASONS but cannot VERIFY (no repo access); this makes reviewers tool-using agents (codex/managed_agents, read-only worktree) that actually run `ailang check`/grep to confirm premises, two-tier (cheap text first → agentic escalation only when a premise is contested). Reuses the quorum contract + executor registry. Sequenced after fleet Phase C. Precondition: confirm Tier-1 has fired LIVE (no artifacts found yet). Requested by Mark.

- **m-mission-portability** — **✅ COMPLETE (M1 attended 2026-07-21, M2+M3 landed iter-92 2026-07-23).
  M1: driver parameterized (MISSION_PROFILE/NAME/REPO/DOC, v1-legacy-exact vs namespaced state,
  template plist). M2+M3: `## Repo Profile` in SKILL.md + charter header (verify profiles
  go-compiler/ailang-code), public bootstrap guide + charter template, dry-run isolation proven,
  evaluator sonnet PASS 83/100. Doc → implemented/v0_30_0/. Ailang World launch UNBLOCKED:**
  ([planned/v0_30_0](planned/v0_30_0/m-mission-portability.md), 2026-07-18;
  **P1 mission-infra — GATES THE AILANG WORLD MISSION LAUNCH**, Mark: "design doc this up and plan
  it in") — extract the loop into a portable template: M1 driver parameterization + per-mission
  state namespace (`MISSION_NAME/REPO/DOC` profile env; backward-compatible defaults — this
  mission's behavior unchanged), M2 skill repo/verify profiles (go-compiler vs ailang-code —
  World verifies via `ailang check/test/ai-check`, which the binary ships), M3 bootstrap kit +
  charter template + scratch-repo dry-run (no state collision with the live loop). ~1–1.5d, zero
  language surface. **Pick order: after the greenlit clause-3 trio (fmt → footguns → strict-
  fallbacks) — OR earlier if the clause queue blocks on anything.** ONE skill parameterized, never
  forked (Gate-5 retro fixes must keep benefiting all missions). Expect quorum-at-pick (doc
  authored interactively, no creation-time quorum).
- **m-eval-reasoning-model-fairness** ([planned/](planned/m-eval-reasoning-model-fairness.md);
  authored 2026-07-11, **QUEUED by Mark 2026-07-19, P1**: "why does GLM 5.2 perform worse than
  5.1? We think it may be our eval harness's fault — thinking tokens/limits with OpenRouter") —
  the doc already carries the evidence: GLM-5.2 40/56 vs 5.1 48/56 with negative token counts,
  empty `code` fields despite compile_ok, and NO reasoning request/budget (MaxTokens bounds total
  output → thinking crowds out the answer). Iteration 43 proved the same mechanism live in our
  quorum (PR #408) — apply the same remedies: reasoning-aware budgets, fail-loud on
  `finish_reason=length`, per-turn finish_reason capture, then RE-RUN the GLM pair to split
  harness-artifact from genuine regression. ~1–2d, metered-cheap (OpenRouter), no GPU. Expect
  quorum-at-pick. Eval-infra (non-gating for v1.0) but Mark-prioritized — pick after the
  greenlit clause-3 trio unless the queue blocks. **RE-RUN VERIFICATION MODELS: the GLM 5.1/5.2
  pair + Kimi K3 (top OpenRouter model, 97/109 standard — also reasoning-class).**
- **m-comments-for-ai-authors** ([planned/v0_31_0](planned/v0_31_0/m-comments-for-ai-authors.md),
  **direction RATIFIED by Mark 2026-07-20**: prompt style guidance + first-class `---`
  doc-comments + contracts/tests-as-self-documentation "as much as is reasonable" + the eval) —
  M1 prompt comment-style section (≤15 net lines, prompt-manager lane, ~0.5d) · M2 the
  comment-variant A/B (V-strip / V-keep / V-migrate on MODIFICATION tasks, haiku, N-run
  aggregates; registered hypotheses; SHARES m-eval-fmt-weakmodel-ab's variant machinery — build
  once, run both) · M3 first-class `---` doc-comments as AST nodes (v0.31; dissolves fmt
  attachment at the root for the doc position; sequence AFTER the fmt polish pair) · M4
  contracts-as-docs exemplars (rolling). First measured comment semantics for AI authors.
- **m-eval-fmt-weakmodel-ab [M1+M2a LANDED iter-72 (PR #438 squash `260faa42a`); M2b LANDED iter-74
  (60 runs banked, 30/arm × 5-trials × 6 frozen benchmarks; cloud-haiku via `claude` CLI on
  SUBSCRIPTION, NO rig.lock; treatment delivery PROVEN 29/30 ON runs fired the hook vs ~8% baseline;
  arm gating clean OFF=0 fmt events); M3_ANALYSIS LANDED iter-76 (PR #450; opus executor + sonnet
  evaluator PASS 87/100; **VERDICT = NULL published**: delta ON−OFF +0.033, Newcombe95 [−0.083,+0.167]
  includes 0 & < +0.10 → not H1-supported/not harm; treatment 32/32=100% exit-0 → true null NOT void;
  green-stability NOT-COMPUTABLE from banked data, reported honestly; helper `tools/analyze_fmt_ab.py`);
  **M4 VERDICT LANDED iter-77 (PR TBD): final verdict = NEUTRAL / true NULL, treatment proven delivered
  — hook stays opt-in, NO adoption-policy change; doc set moved to implemented/v0_31_0/ → SPRINT COMPLETE 5/5]** →
  [implemented/v0_31_0](implemented/v0_31_0/m-eval-fmt-weakmodel-ab.md)** — Mark #422 "Green light weakmodel
  ab" (2026-07-21) UNPARKED it → planner (opus) → executor (opus, worktree) → evaluator (sonnet,
  80/100 PASS r1) + round-2 fix. **LANDED: M1 preregistration** (`-prereg.md`: 6 `.ail`-editing
  benchmarks frozen — fizzbuzz/gcd_lcm/adt_option/higher_order_functions/json_parse/cli_args — N=5/arm,
  Wilson CIs, refutation threshold +0.10 AND CI-excludes-0) + **M2a fmt-hook ON/OFF toggle**
  (`-fmt-hook on|off` CLI flag, default off; `FmtHookMode` on the `microrag_mode.go` precedent; ON
  emits workspace `.claude/settings.json` registering `format_ail.sh` PostToolUse Edit|Write; active
  path `agent_runner_multi.go`; fail-closed hook-reality capture banked as `fmt_hook_events`).
  **M2b DONE (iter-74)**: haiku ON vs OFF ran at N=5 on the 6 frozen benchmarks, banked to
  `eval_results/fmt_ab_haiku_M2b/{on,off}/` (60 run files). Cloud-haiku is an API/subscription model,
  NOT GPU → NO rig.lock (`-no-rig-lock`; the sprint-plan's rig.lock text was superseded by mission
  commit `69501e6dd`). Config-diff clean (both arms prompt_version v0.16.3, model, seed 42, parallel
  4, trials 5 — only `fmt_hook_state` differs). `TODO(M2b)` RESOLVED: the file-sink capture works —
  29/30 ON runs banked a `formatted` fmt event (vs ~8% baseline); OFF banked 0 (arm gating).
  **NEXT = M3 analysis + M4 verdict** (no-GPU, on the banked data): rigorous Wilson-CI deltas,
  convergence, per-turn fmt exit-code coverage; the headline (OFF 29/30, ON 30/30, +1-run driven by
  cli_args only) trends NEUTRAL/NULL-at-haiku-ceiling with treatment delivery proven. SonarCloud
  new-coverage 39.3% (non-required) is expected for the integration wiring. (Mark 2026-07-20
  — "fmt should be a real help for weaker
  models creating AILANG… can we do a test with a weak model to see if its making a difference?"
  + his #422 directive "test it's used by small model such as haiku"): A/B agent-mode evals,
  ONE weak model (haiku first; optionally a local small model as replication), fmt PostToolUse
  hook ON vs OFF, same benchmarks/N-runs. Metrics: pass rate + compile-stuck/green-stability
  convergence (the noisy-agentic-metrics rule: N-run aggregates, never single runs) + per-turn
  fmt exit codes (was fmt actually invoked/useful). Hypothesis: canonical formatting reduces
  weak-model syntax drift. Depends: fmt+adoption (LANDED); sequence AFTER the fmt polish pair
  below lands (test the finished tool, not the interim). ~0.5d + eval time, subscription/cheap.
- **m-eval-kimi-k3-agentic** ([planned/v0_31_0](planned/v0_31_0/m-eval-kimi-k3-agentic.md),
  Mark 2026-07-19: "Kimi K3 did very well — look into using it within the suite via OpenRouter
  and Pi or motoko harness") — K3 = **97/109 (89%) standard, the strongest OpenRouter model on
  the v0.30.0 board** (beats GLM-5.2 88, K2.7-code 88, GLM-5.1 85). Onboard it AGENTICALLY:
  `motoko-or-kimi-k3` + `pi-or-kimi-k3` roster entries (K2.6 precedent, mechanical), smoke→core
  tiered runs, 4-way comparison (vs its own standard score, vs K2.6, vs GLM-5.x, motoko-vs-pi
  harness effect), routing-evidence rows; if it clears the sweet-spot bar → PROPOSE for the
  fleet's Phase-E table (admission stays a routing-policy decision). ~0.5–1d, metered-cheap,
  no GPU. **HARD-SEQUENCED AFTER m-eval-reasoning-model-fairness** — K3 is always-reasoning;
  measuring it agentically on the pre-fix harness = the broken ruler. Expect quorum-at-pick.
- **m-mission-loop-heartbeat [NEW, 2026-07-21 — born from the 18h reboot outage]**: a tiny
  SECOND launchd agent (independent of the loop it watches) that every ~2h checks: newest driver-log
  line older than ~4h AND no kill switch AND no live pidfile → send a controlplane alert + ⚠ comment
  on the bookkeeping issue + `launchctl kickstart` the mission job (recovery, not just alarm). The
  2026-07-20 reboot silenced the loop for 18h and only a human ping caught it — the loop needs a
  pulse that does not share its failure domain. ~0.5d; pairs with RunAtLoad=true (b5b9899a0: repair)
  as detect+repair. Also: the driver should DELETE a stale pidfile whose boot-time predates uptime
  (reboot invalidates PIDs — a reused PID would false-yield every fire; cleared by hand this time).
- **m-mission-cost-chains** ([planned/v0_30_0](planned/v0_30_0/m-mission-cost-chains.md), 2026-07-18;
  **P1½ — the clause-5 KPI's data substrate**, Mark: "keep an eye on these budgets… that should
  all appear in ailang chains CLI") — **[LANDED iter-100 (2026-07-24) — Mark scoped-inference
  decision `4e1348adb` folded → FULL sprint loop, evaluator sonnet PASS 92/100 r1, PR #478 → squash
  `08f9204d0`, all required checks green; doc → implemented/v0_30_0]** (parked-history below). Mark chose SCOPED read-side
  inference (estimate only token-bearing/no-cost/no-quota-bucket/non-zero-rate; quota lanes
  `$0`-by-design; NO schema migration), overriding the reviewers' persistence-required conclusion.
  Controller folded it into the M1 body (no re-quorum, apply-verbatim precedent iters 98/99); planner
  (opus) proved the M2 quota-marking is structurally sound with no migration + found M1 is smaller
  than framed (rate registry already wired into `observatory/pricing.go`); executor (opus, worktree)
  shipped M1 classifier + M2 `chains post-iteration`/bounded-loud spool + M3 `--by-mission`; the
  round-2 soundness hole is closed (quota gate fires before model recovery + a `Validate()` guard).
  **Gate-2 reality-check (iter-97, `v0.30.0-147`):** the doc's Defect-A headline (**cost=$0.0000
  everywhere**) is STALE — recent eval chains attribute cost ($9.59/48h; at-source fix `43333e7a8`
  2026-07-19). Residual M1 (rollup rate-fallback), M2 (mission-ingest — 0 mission chains/14d), M3
  (`--by-mission` absent) all confirmed REAL → valid pick, not a ghost; baseline re-pinned in the
  doc. **QUORUM-AT-PICK:** r1 BLOCKED → 1 fable designer revision (cost provenance + registry
  verification + bounded/loud spool) → re-quorum r2 STILL BLOCKED on a **convergent soundness**
  objection: M1's CLI-side `cost==0 → estimated` inference would corrupt legitimately-free/quota $0
  stages (incl. **M2's own quota-lane `cost_usd=0`**) because the `float64` schema can't distinguish
  absent-cost from reported-$0. Converged fix (both reviewers): persist provenance (nullable
  `Cost *float64` OR `cost_status` field + `quota_bucket`) as a schema migration. Genuine
  schema-design gap → parked. **Human fork (in the doc's Quorum record):** (1) pick M1 provenance
  persistence (pointer vs field); (2) authorize `quota_bucket` on `ChainStage`. `metered=$0.0705`.
  M2/M3 direction unobjected. Original scope: M1 rollup fallback · M2 Gate-4 posts one chain per
  iteration (`mission:<name>/iter-N` — portability-ready for World; bounded+loud spool) · M3
  `chains stats --by-mission`. ~1.5–2d + the migration. **Sequence BEFORE m-cost-per-success-kpi.**
- **m-public-feedback-delivery-audit** ([planned/v0_30_0](planned/v0_30_0/m-public-feedback-delivery-audit.md), 2026-07-12; **P1**) — external user feedback (Kevin's) silently lost: ROOT-CAUSED: dev/prod env split (Mark). Public MCP writes feedback to PROD (`ailang-multivac`) — Kevin's June-30 messages are there, triaged; the rig daemon subscribes to DEV only, so external feedback never pings Discord. Fix = daemon dual-subscribes dev+prod; plus the latent pkg:*-inbox Discord-filter bug. The human-input channel that feeds the data-led loop — prioritize. Requested by Mark.

- **m-mem-budget-runtime** ([planned/v0_31_0](planned/v0_31_0/m-mem-budget-runtime.md), 2026-07-21;
  **P1 — host-safety, DOC-READY**, Mark: "make a design doc for this to insert into our mission
  loop sequence") — the 2026-07-20 rig kernel panic (watchdogd starved under swap-thrash; Jetsam:
  3 model-generated Python procs at ~80-120GB, ailang at 7.7GB) proved generated code WILL
  occasionally be a memory bomb. AILANG's protection today is incidental (no while/mutation +
  interpreter speed) — this makes it guaranteed: `--max-mem`/`AILANG_MAX_MEM` → Go soft limit +
  memguard monitor + cooperative unwind → typed `MEM001` (verified unallocated) instead of host
  death; harness banks it as a distinct error category (model signal, not rig outage). Extension
  lane, zero syntax change, `Mem`-as-effect explicitly rejected (A3/A8). Complements (does not
  replace) the harness-side RSS watchdog task covering the Python/JS/Go lanes. Verification Log
  complete incl. negative-existence rows; Design Freeze needs quorum ratify of the two frozen
  decisions (runtime-control-not-effect; default-off CLI / explicit-on harness). ~2-3d. Phase 2
  (deterministic logical meter, replayable exhaustion) split to a future `m-mem-meter-logical`.

- **m-decision-entropy-monitor** ([planned/v0_31_0](planned/v0_31_0/m-decision-entropy-monitor.md),
  2026-07-22; **[PARKED needs-human-review] — quorum-blocked ×2 (iter 84 2026-07-22).** Rev-1
  flaw (ifaceSeverity can't produce JSON on the broken post-edit file it targets) was fixed by the
  Rev-2 designer pass, but re-quorum surfaced DEEPER blocks needing Mark's design call: (1) A1/A2
  non-determinism — the grader runs the LIVE binary under a wall-clock timeout on reconstructed
  single files with no banked compiler-identity/workspace closure; (2) A5 — `ailang iface` needs
  the workspace to resolve imports, so single-file `/tmp` extraction fails; (3) Conflict-Surface
  omission + overlap with `analyze_stuck.py`/`analyze_run_steps.py`. **Human fork (in the doc's
  ⛔ Quorum Record):** bank iface-JSON + compiler-identity at COLLECTION time (fixes 1+2) vs
  hermetic replay; OR ship M2 `iface --diff`-only (unblocked, independently useful) + defer the
  `D`-grade iface feature. Was: **P2 eval/mission-infra, DOC-READY**, Mark: "detect when big decisions (that have
  large entropy consequences) are made during AILANG code generation — a way to grade when we need
  to closely examine the decisions") — grade every agent step with a decision-weight `D` from
  signals already banked (per-edit `typecheck` green→red, move class WriteFile/bash-write vs
  EditFile/EditDecl, per-path churn) plus the AILANG-native consequence measure: **interface
  delta** — `ailang iface` already emits normalized signatures+effect rows; the diff over them is
  the ONE unbuilt piece (V1–V10 verification log in doc, incl. the negative-existence greps).
  Grounded in green-stability: decision class predicts convergence, so grade the fork-step, not
  just the post-mortem. M1 offline validation on the labeled docx spiral/converger corpus
  (prereg + honest-null, fmt-weakmodel-ab template) → M2 `ailang iface --diff` severity
  none/additive/breaking (additive, independently useful — an agent can check its own blast
  radius pre-commit) → M3 `decision_profile` on RunMetrics + observatory top-`D` view,
  **evidence-gated on M1's report + human review**. Extension lane, zero language surface, no
  motoko-fork changes (pure session-JSONL consumer). Future work under its own evidence bar:
  best-of-N branching / reasoning-effort escalation AT high-`D` steps (consumer:
  m-ai-reasoning-effort). ~2.5–3d, no GPU. Expect quorum-at-pick.

**[LANDED 2026-08-18 (iter-218) — PR `#762` → squash `c307db03b`; PR head 21 checks, zero
not-green, 4/4 required contexts green, `MERGEABLE CLEAN`. THE ROW UNDER-STATED ITSELF: two of the
three deltas were shape pins (`## [Unreleased]` and `## [v0.32.0]` measure `rc=1` at HEAD — already
refused by the structural rule, just unpinned), but the third was a missing REFUSAL BRANCH, not a
missing arm. A missing `changelogs/` directory measured `rc=0`, as did a `changelogs/` holding only
`v0.17-old.md`, both printing `✓ CHANGELOG.md is index-only and links changelogs/` with a blank
filename — the gate certifying a link to a file that does not exist. `if [ -n "$ACTIVE" ] && ! grep
…` skipped the link check whenever `ls changelogs/ | grep current` came back empty. Shipped a second
anti-vacuity floor beside the archive-heading one and DELETED the now-provably-true guard rather
than leaving a branch that can never fire. Self-test 9 → 13 arms. Drills: `if false && [ -z
"$ACTIVE" ]` left all eleven other arms green and failed ONLY arms 12–13 (the new arms are the
branch's killers, not bystanders); `if false && [ -n "$OFFENDERS" ]` failed arms 10–11 alongside
arms 2–5, which share that branch — shape pins, recorded as such. Mutants LANDED by sha256, valid by
`bash -n`, restore byte-identical to `9707d1185`. Baselined on pristine `origin/dev` (9/9) first.
The ubuntu `test` job's own log names all four new arms and `changelog index gate: OK (13 arms)`, so
the pins are proven on CI rather than only on darwin/arm64] m-changelog-gate-deltas — fold the three
arms `#758` had and `#759` lacks into the landed gate.** Iteration 217 closed motoko's `#758` as superseded under `c2022c7fa`'s ownership
rule, and its self-test carried three cases the landed one does not: `## [Unreleased]` and
`## [v0.32.0]` named explicitly (V1's structural rule covers them, but nothing PINS them), and a
missing `changelogs/` directory. Small, mechanical, controller-inline; the reason it is a row and
not an inline addendum is that a follow-up which never gets written is how a closed PR's work is
actually lost. Acceptance: three arms added to `scripts/test_check_changelog.sh`, each shown to
red under an `if false && …` neutering of the branch it targets (rc AND message, per the arm
contract already in that file).

**[NEXT — UNPARKED BY MARK'S ATTENDED RULINGS, 2026-08-19] the nine decision-gated rows.**
Mark resolved **every** remaining V1 decision in an attended session (`1a3ca2d5f` 10:08,
`c29c48e96` 10:23), landing them straight on the charter. Those commits changed the **ledger block
only** (one hunk, 9 insertions / 9 deletions) and touched no queue row — and the same session
ratified **`D-12`: a human ruling that unblocks a design doc MUST create or un-park its queue row in
the iteration that consumes the directive.** This iteration is the first loop pass to see them, so
this block is that un-parking. Each row below is now **routable**; they are listed in ruling order,
not priority order, and a later iteration should order them normally.

| Ruling | Disposition | Row it unblocks |
|---|---|---|
| `D-1` | **RETAIN** zero-DNS literal-IP validation on the proxy route — a THIRD option, neither arm as offered | **DISCHARGED 2026-08-20 (iter-235)** — PR [`#613`](https://github.com/sunholo-data/ailang/pull/613) rebased (310 behind; zero dev commits touched its five files), re-titled, un-drafted and **squashed as `e5ee6c5e5`**. Gate 3b on head `a54a8624f`: **21 checks, zero not-green**, 4/4 required, `MERGEABLE/CLEAN`. |
| `D-2` | **B — widen to close the nested blocks**; A declined because it ships `#614` as accepted behaviour | `m-named-test-body-check-semantics` / `#604` |
| `D-8` | **AUTHORIZE** the ordered rig rollout; PARTIAL is acceptable, remaining risk is deployment ordering | `#618` rollout |
| `D-9` | **SPLIT W8 out and re-quorum it** — W9 disputes direction outside W8 | `m-eval-validity-discipline` W8 |
| `D-10` | **B — hold and route next.** No third revision | `#616` |
| `D-11` | **ADD the short-success guard** | the short-success guard row |
| `D-12` | **YES** — the process rule above | *(process; discharged by this block)* |
| `D-13` | **C — keep scope, re-root durable writes through `$AILANG_DRIVER_SRC`** | filler disposition row |
| `D-14` | **A — recovery-site detection at `parser_literals.go:562`** | dialect-recovery row |
| `D-COV-1` | **LOCALITY** for the gated/badged/Sonar metric, plus a SEPARATE non-gating `test-coverage-xpkg` diagnostic | coverage doc |
| `D-18` | **Ownership scoping, not a claim protocol** — superseded in substance by `c2022c7fa` | no action |

Note `D-1` and `D-10` both came back as something other than the arms the row offered — a row that
forces a binary on a human gets answered outside it, which is worth remembering when phrasing the
next one. **`D-19` (this iteration) is now the ONLY OPEN decision in a 21-row ledger.**

**[NEXT — filed iteration 267] m-sweep-orphans-2026-08-24 — the 8 zero-mention issues from this week's sweep**
The Monday 2026-08-24 external-issue sweep enumerated **76** open issues (`gh` count control 76 = 76) and
found **10** with zero mentions across charter, log, status archive and dashboard. Two are not real
orphans — `#852` is V1's own bookkeeping thread and `#850` is Motoko's. The remaining **8** need
triage-lite (ghost-discipline the repro at HEAD → verdict comment → queue-or-close), batched here as
ONE row per the Gate-0 sweep rule; a sweep never outranks existing picks by itself.
`#847` nightly `explicit_dataflow_ssa` (already triaged on-issue by iter 266 as a local-model
capability gap — needs a charter row so it stops re-orphaning) · `#842` motoko_agent: provider failure
masked as successful empty completion · `#839` motoko_agent: `std/net` ignores `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`
while the AI-provider path honours them · `#800` motoko_agent: `generate-extension-registry` emits
`[effects] max` as the dispatch row · `#754` `messages send --github` loses the message on a GitHub 5xx
and the retry hint names a non-existent subcommand · `#753` Z3 verify excludes most idiomatic AILANG
(3 of 9 contracts verified) — relates to the `D-30`/`cost_per_verified_success` work · `#752` and `#751`
email-parse IFC (Declassify is whole-body authority; docs show grammar the v0.33 parser rejects).
The three `motoko_agent`-authored rows carry cross-mission demand evidence by construction.

**[NEXT — first-party, iteration 267; surfaced by the sprint evaluator's diff-anchored mutation drills] m-serveapi-moved-code-coverage**
Two lines moved into `serveapi` by `#764` M2 are exercised by nothing. Both were verified
**pre-existing** — byte-identical copies of the pre-move code — so they are not M2 regressions and were
deliberately NOT folded into that sprint. (1) `serveapi/a2a_handler.go`: the card-path error envelope's
JSON **key shape** is untested — mutating `"error"` → `"errX"` leaves the whole package green, because
`TestEmbeddedA2AFrozenCallbackEnvelopes` only `strings.Contains`-matches the message text. (2)
`serveapi/mcp_handler.go`: `IsError` on the moved private `mcpError` copy — flipping it `true`→`false`
leaves the package green, while its apiserver-side sibling **is** directly pinned
(`internal/apiserver/protocol_test.go:460-462`). Both mutants were asserted LANDED and BUILDING before
their results were read. Deliverable: one arm each, driving a failing embedded tool call / callback
error through the live path and asserting the wire shape rather than a substring.

**[NEXT — 12 of 15 dispositioned; 3 remain] m-sweep-orphans-2026-08-17 — the zero-mention issues from
iteration 216's weekly external-issue sweep.** ONE batched row per the Gate-0 sweep rule (a sweep never outranks
existing picks by itself; only a confirmed soundness/regression finding does). Sweep verdict:
**15 orphans of 75 enumerated** — `gh` count control 75 = 75, `\b` matcher validated in-call
(`#613`→5 positive, `#99999`→0 negative, `#613` vs `#6130` discriminated), all four scoped files
asserted non-empty. `#696` was picked and LANDED this iteration (`3a75ec7d2`), leaving 14:

- **Mission/loop infra** — ~~`#727`~~ **[LANDED 2026-08-18 (iter-219) — PR `#765` → squash
  `6ff68eda9`, 21 checks / zero not-green, 4/4 required]** (`govulncheck-filter`: REAL at HEAD —
  `duplicate` and `parse stdin` both 0 in `main_test.go` with controls firing at 4 and 2. Enumeration
  re-anchored to `decide()` itself and it agreed: exactly four `return 2` branches, two pinned, two
  not. `TestDecideExitCodes` 5 arms → 7, each asserting a **branch-unique message** because exit 2 is
  over-subscribed across all four. Both mutants LANDED+BUILDS; each redded ONLY its own arm with
  `-skip TestDecideExitCodes` rc=0, so each arm is its branch's killer. Ubuntu `test` log names both
  new arms 4× against a 4× control, so the pins are proven on CI not just darwin), ~~`#708`~~ **[LANDED 2026-08-18 (iter-220) — PR `#767` → squash
  `904cb9b0d`, 21 checks / zero not-green, 4/4 required]** (`design-quorum` recorded no
  per-reviewer token usage, making Gate-3's chain-telemetry token mandate unsatisfiable — the same
  defect the skill's own iter-190 measurement recorded as "two OpenRouter quorum stages at ZERO
  tokens". REAL at HEAD, and the counts were not merely unrecorded: `run.go` READ them, spent them
  on `estimateCost`, and dropped them; the agentic tier returned a literally empty `&ai.Response{}`
  while `coordinator.ExecuteResult` had carried them all along. Enumerating the construction sites
  first turned a one-line patch into one sweep across BOTH tiers. **The drill found a hole the issue
  did not describe:** zeroing the token mapping in the production `ExecuteResult` → `AgenticRun`
  adapter left the ENTIRE package green — it sits behind `NewExecutorProvider`, so no test could
  reach the one place the executor's real counts enter the quorum. Extracted and pinned. 10 arms,
  8 drills, every inverse `-skip` rc=0; the artifact arm is the sole catcher of a `json:"-"` tag,
  which is why it asserts on emitted JSON keys rather than on the struct),
  ~~`#687`~~ **DONE (iter-222)** — REAL, and the row's guess that it was probably a ghost was wrong
  for the second time running (`#708` was the first). Repro'd as a differential whose only variable
  is the working directory. Fixed by making the mtime walk a pre-filter and requiring content proof
  from the binary's embedded commit before warning. **The mission-infra lane is now COMPLETE**
  (`#696` already-fixed, `#727` real, `#708` real, `#687` real) — 3 of 4 were real, so the lane's
  prior expectation that these were mostly stale is refuted.
- **Language / stdlib** — ~~`#688`~~ **[LANDED 2026-08-18 (iter-223) — PR `#775` → squash
  `a1dad782a`, 21 checks / zero not-green, 4/4 required]** (REAL at HEAD and the report UNDERSTATED
  it: `charAt` is not O(i) but **O(n) with an O(n) allocation independent of the index** — `B/op` was
  exactly `4n+16` at every index, now **20 bytes/op flat from len=80 to len=80000** against a firing
  327,767 bytes/op control. Enumerating the sites first turned a perf fix into a correctness one:
  `charAt` has THREE implementations and the `emit-go` one indexed by **byte**, so compiled output
  silently diverged from interpreted — `charAt("héllo",1)` = `"Ã"` vs `"é"`, and index 5 returned
  `"o"` where the interpreter raises out-of-bounds. Two more defects surfaced only because the
  emitted package was compiled and RUN rather than re-derived — a Helper body cannot introduce a Go
  import, and `charCode` had no codegen spec at all. 9 drills, **7 unique killers**, cost arms
  asserting allocated BYTES because the pre-fix alloc COUNT was flat and would have passed.
  **NOT closed by this:** O(1) access needs a representation change and is not claimed, and the
  report's `find(s, needle, from)` / `lastIndexOf` / `indexOfAny` asks are API additions — see the
  row below), ~~`#689`~~ **[LANDED 2026-08-18 (iter-224) — PR `#778` → squash `32ee90ed9`, 21 checks /
  zero not-green, 4/4 required]** (REAL at HEAD and the report's CHARACTERISATION was refuted:
  it blames "a record containing an ADT-typed field"; neither the record nor the ADT is the
  trigger. A five-case matrix isolated the **element type** — a `list[string]` field fails with
  no ADT anywhere, a `list[int]` field verifies, and a plain `-> list[string] { [] }` with no
  record at all fails the same way. An empty list literal carries no element type and SMT-LIB
  is monomorphically sorted, so the encoder's hardcoded `(Seq Int)` was ill-sorted everywhere
  the expected element type was not `int` — under a source comment claiming the solver would
  unify sorts, which it does not. **7 contexts**, enumerated before patching; **ANF** is why
  the obvious fix fails, since every literal is hoisted into a temporary before it reaches the
  site whose declared type would supply the sort. Verified in BOTH directions: 5 negative arms
  still report violations, 34/34 shipped contract examples byte-identical, fixture is 7 errors
  before / 7 verified after. 9 drills, 8 unique killers. **The report's item 1 — reclassify as
  `skipped` — is NOT closed by this and is queued below on its own evidence**), ~~`#662`~~ **[LANDED 2026-08-19 (iter-225) — PR `#780` → squash `d5831af9b`, 21 checks /
  zero not-green, 4/4 required]** (WASM type-checker budget is wall-clock, so module loading is
  hardware-dependent. REAL at HEAD — hardcoded `2 * time.Second`, **zero** setters (same-scope
  control: 2 `Begin.*TypeCheck`) — and the report UNDERSTATED it: the guard had **no test coverage
  by construction**, `go list` reporting **0** native files containing it against a `_native.go`
  control of 1 and a `GOOS=js` reading of 1, so nothing could red when it broke. Shipped asks
  1/3/4 — `ailangSetTypeCheckBudget(ms)` with a rejected value leaving the previous budget in
  force, `typeCheckMs`/`typeCheckSteps`/`budgetMs` on **every** load outcome, and an error message
  naming the budget in force rather than a constant. Ask 2 — gating on the deterministic step
  count — is **not** closed and `#662` stays OPEN for it, because picking a ceiling from our own
  guesses is exactly how the 2 s was chosen; the counter ships first and the reporter has been
  asked for per-module counts from the harness they already have. 13 drills, 8 unique killers;
  **M11 exposed a hollow arm** — deleting `begin()`'s sticky reset left the whole package green,
  because the arm asserting it called `end()` first and `end()` clears the same flag. **The Sonar
  red was attributed to this diff by negative control** rather than waved through as standing:
  `#778`/`#775`/`#767` were all Sonar-green on their heads, and the finding was real
  (`go:S1764` on `ms != ms`), as was an untested exported accessor the bridge calls on every
  load), ~~`#646`~~ **[LANDED 2026-08-19 (iter-226) — PR `#782` → squash `47760b931`, 21 checks /
  zero not-green, 4/4 required]** (REAL at HEAD, and the fix needed a second half the reporter
  could not have seen. The report is exact and the drop is at PARSE, not in `getText`:
  `findAllTexts` returns `[plain , bold, , italic]`, so the `<t>` ELEMENT survives and its text
  child was never created. **Two** drop sites, enumerated before patching, with every entry point
  funnelling through one of them. The second half: `xml:space` was **unreachable from AILANG** —
  names resolve through the in-scope `xmlns` declarations and the `xml` prefix is bound BY
  DEFINITION and must never be declared, so `getAttr(t, "xml:space")` returned **None** on a
  document that plainly had one and `serialize` round-tripped `<t xml:space="preserve"/>` out as
  `<t space="preserve"/>`. Control in the same document: a DECLARED `xmlns:w` resolved
  `w:p`/`w:t`/`w:val` correctly, so only the never-declarable prefix was affected. **The default
  did not move**, which is why no `parsePreserveSpace` flag was needed. 14 drills; **M11 initially
  SURVIVED with zero killers** — the three streaming seeds are a code path neither recursive
  parser's arms reach — and the arm added for it then kills M11/M12/M13 alone. Arms proven on
  **linux** CI at 4× against a 4× control, 0 skips. **NOT closed by this:** the general
  undeclared-prefix drop and the streaming scanners' `nil` match-name prefix map — see the row
  below), **[LANDED 2026-08-19 (iter-227) — PR `#784` → squash `b2bbac8d9`, head `448cebd11` at 21 checks, zero not-green, 4/4 required. `std/zip.buildArchive` / `buildArchiveWithBytes`: PURE, base64 out, no `FS`, so OOXML/ODF GENERATION now works in a WASM/browser host — verified present in `bin/ailang.wasm`, and the runnable example's `main` is `-> () ! {IO}`. Shipped BOTH the text and bytes variants rather than the one requested, because the FS side has both and shipping one would force base64 on every XML part. All four builders now share ONE serialiser (Principle 3; the two FS impls were ~90% duplicated). Purity verified as a property of the OUTPUT — `archive/zip` uses the MS-DOS epoch, not the wall clock — pinned across a 2.1s gap, and `buildArchive` == `createArchive` byte-for-byte. NEW total-content cap on the in-memory pair only, with a negative control pinning that the FS pair did not inherit it. 10 drills, 5 sole killers with inverse `-skip` rc=0; the other 5 broad-blast, recorded as enumerated red sets — which became this iteration's skill edit]** `#644` (`std/zip` needs an in-memory archive builder).
- **Downstream-consumer reports** (all bot-imported via the agent-message channel, so they carry
  the demand-evidence a sibling-mission request would) — ~~`#679`~~ **[DISPOSITIONED 2026-08-19 (iter-230) — REAL at
  HEAD, reporter's MECHANISM REFUTED, kept OPEN: fix site is `ailang-parse`, outside `MISSION_REPO`]** (`--deep` is NOT
  skipped on the CLI path — the warning is STALE. `parseDocumentPure` bakes it into `outcome.warnings` whenever `deep`
  is set; `parseDocument` consumes that outcome and THEN does the real deep work via `orchEmailDeep`, retracting
  nothing, so the user is told to use the function they are already using. Proven by an arm needing no subprocess: a
  DOCX attachment's inner content came back — `$425,000`, `[table] 3 rows, 2 cols` — with the warning still firing.
  The same string is TRUE on the MCP path (`parseDocumentAI`, no email-deep branch), because the shared pure helper
  cannot know its caller — and `orchestrator.ail:723` names the cause: **no effect-row polymorphism (`#616`)**, so the
  capability tiers are three entry points. First-party downstream demand evidence for `#616`, which is HELD by `D-10`.
  Ruled out: that the stale warning causes `eparse expand`'s data loss — email-parse branches on `warnings` 0 times,
  control firing), ~~`#676`~~ **[DISPOSITIONED 2026-08-19 (iter-228) — REAL at HEAD, reporter's
  attribution REFUTED, design doc PARKED on `D-19`]** (the report blames hand-rolled recursion vs
  builtin `map`; the cause is neither — **`::` itself is O(n)**, because `eval.ListValue` is a flat
  Go slice and `listConsImpl` copies the whole tail every call, so ANY list built by prepending is
  Θ(n²). `map` is fast only because `_list_map` is an iterative Go builtin that never conses.
  Derivation, not correlation: pprof gives **95.25%** of alloc_space to `listConsImpl` against a
  closed-form prediction **2.6%** away. Two more defects found: the tree-walking evaluator has **no
  tail-call elimination** (control fires), so the same idiom dies at `RT_REC_003` depth 10,000 —
  split out as Sprint 2, `m-eval-tail-calls`; and `std/list.reverse` is quadratic while the
  iterative `_list_reverse` builtin has **0** callers in `std/` — split out below. Doc
  `design_docs/planned/m-list-cons-quadratic.md`, quorum r1 BLOCKED/BLOCKED, r2 pass/reject,
  `absent_reviewers` empty in both. Ruled out: the `evalCoreApp` trace recorder, by an A/B that
  came back identical),
  ~~`#671`~~ **[LANDED 2026-08-19 (iter-231) — PR `#790` → squash `3b18f60ce`, head `9a196ad56` at 21 checks /
  zero not-green, 4/4 required]** (REAL at HEAD and the message was **provably false**. Reproduced as a
  differential whose only variable is the working directory: cwd inside → resolves, cwd `$HOME` → fails, cwd in a
  SUBDIRECTORY → resolves, and cwd outside **with a real `ailang.lock` from `ailang lock` (rc=0)** → fails
  identically. The manifest search was anchored at `"."` — the process CWD — while `FindManifest` walks
  **upward**, so an installed CLI invoked from anywhere but its own project root wired no package resolver at
  all. A second defect the reporter could not see: relative `./sibling` imports fail from outside too, with the
  SAME string — one root cause, one unified fix. And that string was emitted for **two opposite situations**
  (manifest elsewhere vs no manifest at all), making it the third consecutive consumer report in which
  **the message is the bug**. The fix already existed at `internal/testing/executor.go:283-286`, the ONLY
  non-test `PackageDir` setter, with a comment naming this hazard — guard the helper, miss the call site.
  The existing intra-package suite could not catch it because all three of its arms `chdir` INTO the package
  first; the new arms carry an `assertNotAncestor` anti-vacuity floor. 4 drills, 2 sole killers with inverse
  `-skip` rc=0 and 2 broad-blast recorded as enumerated red sets. **Two first-party defects in my own work,
  both caught by gates:** `check-file-sizes` went red and it was MINE (pristine base 792/rc=0, my draft 813;
  the first "baseline" was contaminated by running in the modified tree — rule 3e(b)); and CodeQL flagged
  `go/path-injection` [high] on my own new `os.Stat`, which I resolved by REMOVING the probe as unnecessary
  rather than dismissing the alert, after auditing first-party that no HTTP handler reaches that path),
  ~~`#694`~~ **[LANDED 2026-08-20 (iter-232) — PR `#792` → squash `241221047`, 21 checks / zero not-green, 4/4 required]** (REAL at HEAD. Since VS Code 1.74
  `extensions.json` is the REGISTRY, not a cache, and the installer got it wrong in both directions at once: an
  UNVERSIONED folder nothing scans, then `invalidateVSCodeExtensionCache` REMOVING the entry — which is an uninstall.
  **The cause is the NAME**: `uninstallVSCode()` calls the same helper, where removing the entry is exactly right, so
  one function was correct on one caller and inverted on the other. Renamed `deregisterVSCodeExtension`. Fourth
  consecutive report in this group where the artefact DESCRIBING the behaviour is the bug — the first three were
  emitted strings, this one a function name, so the class is wider than diagnostics. Registry schema measured
  first-party off a real 9-entry `extensions.json` on this rig, not inherited from the report. `editor.go` had ZERO
  test references before this, control 77 `_test.go` files. Same defect second surface: two guides taught
  `cp -r … ~/.vscode/extensions/` and verified with `ls | grep` — both fixed. 19 tests, 11 drills, 6 sole killers with
  inverse `-skip` rc=0, 5 broad-blast enumerated; M11's first form did not compile and was repaired, not read.
  NOT closed: whether VS Code accepts the VSIX is unrunnable here — no editor CLI on this box),
  `#672` (housemove2026: `eparse thread` msg-id format — eparse-side, outside `MISSION_REPO`),
  `#656` (`/api/v1/convert`, feature — ailang-parse-side).

**The in-repo half of this sweep is CLOSED at iteration 232**: both remaining orphans (`#672`, `#656`) have their fix
sites outside `MISSION_REPO`, so neither is routable here. Final in-repo score: **9 real of 10 dispositioned** — the
row's original expectation that these were mostly stale is refuted decisively.

Per-issue triage-lite is OWED and NOT yet done for the remaining 3 (`#672`, `#694`, `#656`) — ghost-discipline each repro
at HEAD before routing. (The header counter read "7 of 15 / 8 remain" through iteration 230 while the
enumeration below listed 11 dispositioned and 4 remaining; corrected at iteration 231 by counting the rows.) Three of fifteen are now dispositioned and they have NOT come out one way, which
is the whole argument for the discipline: `#696` was already fixed (iter-216), `#727` was real
(iter-219), `#708` was real (iter-220). Note the prediction that failed — this row guessed `#708`
was "a strong candidate to be a ghost or already-fixed" because the loop had since edited that code;
it was real, and its fix uncovered a second, larger coverage hole. `#687`'s mtime heuristic carries
the same guess and inherits the same warning: guessing is not ghost-disciplining. Sequence them by lane: mission-infra first (cheapest to
verify, and `#708`/`#727` both block gates this loop depends on), then language, then consumer
reports. **Mission-infra closed at iteration 222; the language/stdlib group is CLOSED at 5 of 5 (`#688` iter-223, `#689` iter-224, `#662` iter-225, `#646` iter-226, `#644` iter-227) — the remaining group is the downstream-consumer reports below.** The lane's running score is now 7 real of 8 dispositioned, so the row's original expectation that these were mostly stale is refuted twice over. Do NOT batch-close: each needs its own measurement, and a CI-enforced guard where the
verdict is ghost-or-fixed.

**[PROGRAMME · roadmap **QUORUM-CLEARED 2026-08-20 (iter-234)** — created by Mark's `D-19 : B` ruling, iteration 229] m-list-cons-cells — true cons
cells / structural sharing.** Mark ruled **B** on `#745` at `2026-08-19T10:58:40Z`: the front-slack
arena is declined as the permanent answer and the language takes true cons cells, whose invariant
(**INV-1**) is that `x :: xs` is O(1) worst-case *regardless of sharing* — not merely along a linear
use chain, which is the exact gap `gpt5-6-sol` blocked the arena on. Roadmap:
`design_docs/planned/m-list-cons-cells-decomposition.md` (505 lines, 20 first-party verification
rows). ⚠ **Letter collision:** the superseded `m-list-cons-quadratic.md`'s own **Option A** is what
this ruling selects; its **Option B** (arena) is declined. **The eight pieces, queued individually —
the roadmap's owed revision + re-quorum LANDED at iteration 234, so routing is now unblocked — take them in order, LC-1 first:**

> **`PARKED-ON-LANE` RESOLVED 2026-08-20 (iter-234).** The predicate was re-run as a COMMAND, not transcribed (Standing rule 8(d)): `codex exec --model gpt-5.6-sol 'reply with exactly: ok'` → **rc=0** at 06:12 CEST. The lane returned, so per Gate 2's blocked-external-row rule (d) the row became that iteration's pick regardless of queue position. The owed revision RAN on the rotation's next designer entry (`codex:gpt-5.6-sol`, so the Fable diet was untouched), and the **re-quorum blocked 2-of-2 on two NEW premise objections** — both measured rather than forwarded (rule 3f) and resolved under the **narrow-refinement carve-out** (ratified iter-98). PR [#798](https://github.com/sunholo-data/ailang/pull/798) → `03e3e6057`.
>
> **The correction that matters for routing the pieces:** `gemini-3-1-pro` caught that N16 had intersected ListValue×ArrayValue and omitted **TupleValue**. The symmetric-switch surface is **7** non-test files, not 3 (**8** including tests, N16's own scope) — and one of them, `internal/eval/eval_patterns.go`, is **LC-3b's**, refuting the roadmap's claim that they are "all in LC-3a/LC-3c clusters". **All three migration lanes carry symmetric-switch work, not two.** `gpt5-6-sol`'s aliasing objection went the other way: N22 measures zero production callers, zero mutating call sites, and `internal/`-only reachability, so the escape-site copy is an intentional documented snapshot change with a per-lane mutation AC, not an API-versioning problem.
>
> **The eight pieces are now routable in order.** LC-1 (`m-list-repr-spike`) runs FIRST and carries the programme's kill criterion; nothing else may be routed before it lands.

1. `m-list-repr-spike` **[LC-1 — COMPLETE. LANDED 2026-08-20 (iters 237+238+239), PRs [#804](https://github.com/sunholo-data/ailang/pull/804) + [#808](https://github.com/sunholo-data/ailang/pull/808) + [#810](https://github.com/sunholo-data/ailang/pull/810). **VERDICT: GO.** All three candidates pass all five ratified clauses; the control leg fires at 9.854×/10.384× against a required ≥8, so the gate was falsifiable and did not fire. LC-2…LC-5 are UNBLOCKED and `D-19` is NOT re-opened — **but `D-22` (OPEN) gates which representation they build for**]** **Iteration 238 executed M3, M4 and M5, and C2 was reinstated with the number that justifies it.** M3 is the always-copy leading chunk (no ownership tracking, no refcounting, no atomics, no locks), meeting doc:93-100's O(K) = O(1) bound by construction. **Provisional clause (c) under M5's OWN five-trial protocol** (fresh processes, ordinal-paired, `-benchtime=1s`; darwin/arm64; NOT M6's full matrix): C1 median **1.946×** vs ≤ 2.5, and B6 B/element **C0 16.34–16.50** · **C1 31.95–32.00** · **C2K8 ≈22.0** · **C2K32 ≈17.57** — so C2(K=32) is ≈**1.07×** where C1 is **1.95×**, on the clause where C1 is most at risk. That is the number whose absence made iteration 237's descope a STOP-by-descope. Clause (d): **1.000/0.999/0.999/1.002** across all four arms vs ≤ 1.2 — and the protocol's `benchtime` is load-bearing, measured: at `-benchtime=100x` C1 swings to **2.26×**, a spurious FAIL. Evaluator (sonnet, own worktree) PASS **84/100**; its three best findings fixed BEFORE merge — **`main()`'s rerun orchestration was pinned by nothing** (rerun batch 5→4 and `allowRerun` false→true BOTH survived; now sole-killed by two new arms over an injected collector), `newChunkList`'s **defensive copy was uncovered** (an aliasing mutant kept the whole suite green), and **AC-7's base SHA was a stale literal** (6 commits behind; 10 spurious violations vs 0 on the derived base, control 15 — now DERIVED via `git merge-base`, after iteration 237 fixed it in its directive only). Controller-found: **AC-4 leg 3's gate matched `==`**, returning 3 comparison hits outside constructors so clause (e) fails on correct code, while composite-literal constructor writes are invisible to it entirely — corrected with an assignment-only sweep, a deliberate liveness probe, and a separate constructor check. **Iteration 237 executed M1 and M2 and the branching control FIRES.** C0's `time(L=16384)/time(L=1024)` median **11.60×** (m=1024) / **12.37×** (m=4096) against AC-2's ≥ 8, with C1 flat at **0.95×** / **1.08×** against clause (a)'s ≤ 1.5 — heaviest cell **112 ms** C0 vs **62 µs** C1. Three independent readings (executor, controller, evaluator's 5-trial) agree in order with no threshold-crossing ambiguity; all are **provisional dev-loop, darwin/arm64, NOT M5's five-trial protocol**. Evaluator (sonnet) PASS **93/100**, zero blocking, and its best find was fixed before merge: the shared-scratch-buffer mutant SURVIVED the whole suite, because nothing tested that two prepends off ONE shared base stay independent — the property every B1 number is credited with. Now pinned by `TestBranchingIndependence`, a **sole killer** (mutant LANDED+BUILDS asserted first; new test rc=1, `-skip` rc=0). **M3 IS RE-OWED: the executor's C2 infeasibility ruling was REFUTED by the controller and independently confirmed refuted by the evaluator.** Its premise (the immutable API cannot detect unique chunk ownership) is true; the conclusion is not, because (a) the doc's C2 bound is already the CONTENDED one — copy ≤ K into a fresh node, O(K) = O(1) worst case — which always-copy meets with no ownership detection at all, preserving the per-element amortization (`16 + 16/K` = **16.5 B at K=32**, matching doc:417's ~16-18 B estimate); and (b) doc:419 tolerates infeasible C2 columns only *“if C1 passes”*, and C1's clause-(b)/(c) numbers are **M4** work that does not exist. C2 is the candidate designed to pass exactly where C1 is most at risk — (b) iteration at n=65536 and (c) per-element memory (C1 at 32 B/cell) — so descoping it before either is measured makes a **STOP reachable by descope rather than by measurement**. Revised instruction recorded in the plan's §4 M3; §7 R1 narrowed. **Next iteration: M3 always-copy C2, then M4–M6.** — doc 495 lines / 20 verification rows, plan 6 milestones / ~1,800 LOC / 3.0 d. Quorum BLOCKED 2-of-2 twice (both reviewers present, `absent_reviewers` empty both rounds), resolved under the narrow-refinement carve-out; `$0.1605` metered. Two reviewer findings changed the design: gemini's *false premise* that Go cannot forbid same-module imports (MEASURED false — a nested `tools/internal/` package is compiler-refused to production importers, `rc=1`, positive control `rc=0`), so the spike moved to `tools/internal/spike-listrep/` behind a **compiler gate** instead of a README; and gpt5-6-sol's *non-deterministic gate* (`-count=3` with no aggregation rule), now five fresh-process trials + median of paired ratios + a predeclared tie/spread rerun. **The planner then refuted two of the doc's own ACs, both confirmed first-party: AC-3 was UNSATISFIABLE** (clause (d) needs `Len()`, and 0 of 9 benchmark rows measured it) **and AC-7 was VACUOUSLY GREEN** (scoped diff 0 lines, unscoped control 3 — it passes on a branch that did nothing). Kill criterion ratified clause-by-clause; no threshold weakened at any step — throwaway benchmark of cons-with-cached-length
   vs chunked-cons vs the slice control. Carries the programme's **kill criterion**: if no candidate
   achieves O(1) worst-case prepend *under branching* within ~2× iteration / ~2.5× per-element memory
   / O(1) length, the programme STOPS and `D-19` is re-opened with measurements. Everything else is
   gated on this.
2. `m-list-accessor-api` **[NEXT — SPRINT PLAN LANDED 2026-08-22 (iter-252); route to
   sprint-executor, M1+M2 (independently committable). LC-2 — plan says 4.5d, +0.5 OVER the
   roadmap's 3–4 band, surfaced with three costed line items rather than compressed to fit.
   Plan: `design_docs/planned/m-list-accessor-api-sprint-plan.md` (455 lines, 6 milestones,
   ~1,660 Go LOC, 19 named mutations one-per-refusal-branch); sprint JSON
   `.ailang/state/sprints/sprint_M-LIST-ACCESSOR-API.json` (`git add -f` — `.gitignore:82`
   ignores `.ailang/` while 56 sprint JSONs are tracked). **⚠ SETTLE DEFECT-1 IN THE EXECUTOR
   DIRECTIVE BEFORE M2**: `packages.Config.Tests` is unspecified and defaults to `false`, while
   380 of 903 `.Elements` and 291 of 388 `ListValue{` sites are in `_test.go` — the census
   denominator, which every downstream lane is measured against, moves by ~380 sites on an
   unstated flag. Nine further doc defects are carried in the plan, not silently fixed in the
   doc; DEFECT-2/-7/-10 also change what the sprint must do]** — **UNBLOCKED by `D-22` = `C1`** (Mark, `#745`,
   `2026-08-22T11:36:26Z`). Doc: `design_docs/planned/m-list-accessor-api.md`, 793 lines, **28
   first-party verification rows**, cleared via the **narrow-refinement carve-out** after two 2-of-2
   external blocks (round 1 $0.104328, round 2 $0.132845, `absent_reviewers` EMPTY both rounds — no
   N−1 degrade). **Three findings the quorum bought, each worth more than the round it cost.**
   (a) **The seam as the roadmap words it would ship a gate that never gates**: `make ci` appears
   **0** times in `.github/workflows/ci.yml` (same-scope control: **37** `make ` invocations;
   negative control 0; scope asserted), so "make/ci.mk wiring" alone wires nothing — the gate adds
   its own ci.yml step. (b) **A scan scope is invisible to a ratchet BY CONSTRUCTION**: round 1
   scoped the analyzer to `internal/`+`cmd/` to dodge the LC-1 spike; measured, that costs **zero**
   baseline sites today (`tools/` `.Elements` = 7, **all 7** in the spike, **0** outside;
   `examples/` = **0**; controls 31 and 18 files), which is exactly why widening to `./...` with a
   package exemption is free — a removal proves a check FIRES, only an addition proves it LOOKS.
   (c) **Two correct objections pointing opposite ways, resolved by placement not by choosing**:
   `gpt5-6-sol` required composite-literal coverage in the permanent Rule 2 (Rule 1 retires at LC-4,
   taking that coverage with it); `gemini-3-1-pro` then measured that the same addition flags all
   **388** existing `ListValue{…}` sites on day one and breaks CI, in a piece that migrates no
   consumers. Both confirmed first-party (388 exactly; control `ArrayValue{` = 14; negative 0). The
   class therefore lives in the **ratcheted** Rule 1 through LC-2/LC-3 and becomes an explicit
   **LC-4 obligation** with its fixtures already written. **Rule 3f earned its keep twice**: one
   premise confirmed exactly, one refuted in the doc's favour (re-running L13 without its
   `grep -v _test.go` filter finds **2** hits, **0** in tests — the filter was hiding nothing), so a
   whole revision round was replaced by one command. — the accessor seam over the UNCHANGED slice, plus a
   type-aware `listrep` `go/analysis` ratchet. Its FIRST deliverable is a type-checker-driven census,
   because **grep cannot size this migration**: `Elements` is a field name owned by **22 struct
   types** across `internal/{ast,core,eval,typedast,types}`, so the 902/386 textual counts are
   contaminated upper bounds. Every migration AC is denominated in the analyzer's count, not grep's.
3. `m-list-migrate-builtins` **[LC-3a — 2–3d]**, 4. `m-list-migrate-runtime-effects` **[LC-3b —
   2–3d]**, 5. `m-list-migrate-periphery` **[LC-3c — 2d]** — mechanical, parallelizable in worktrees.
   **Per `gemini-3-1-pro`'s round-1 objection, each escape site goes to its OWNING package's piece**
   (`builtins/safe_cast.go` → LC-3a, `effects/testctx/mock_context.go` → LC-3b, `embed/convert.go` →
   LC-3c); the roadmap as written assigns all three to LC-3c and must be corrected before routing, or
   the parallel lanes conflict by construction.
6. `m-list-cells-swap` **[LC-4 — RISKIEST, 3–4d]** — the representation swap inside `internal/eval`,
   with deletion of the `Elements` field as the compile-time proof migration was complete. **Per
   `gpt5-6-sol`'s round-1 objection the `listrep` analyzer is RETAINED, not retired**: Go has no
   immutable fields, so unexporting confines mutation to the owning package (`internal/eval`, which
   is also the evaluator) rather than eliminating it. Immutability and safe publication are
   **required properties to be verified**, not established facts — LC-4 owes a verification row per
   publication path with its Go happens-before edge, plus race tests.
7. `m-list-post-swap-tuning` **[LC-5 — 2–3d]** — honest-cost pass: `_list_nth` becomes O(i)
   (`ArrayValue` is the documented O(1)-indexing alternative), per-element memory rises from 16 B
   contiguous to an estimated ~32 B per cell (a derivation LC-1 must measure), benchmark matrix, docs.
8. `m-list-interim-communication` **[LANDED 2026-08-21 (iter-240) — PR #811 → squash `7db3db2d9`, Gate 3b GREEN on the merge (3/3 workflows, 20 checks ZERO not-green); evaluator sonnet 74/100 FAIL r1 on the `++` carve-out gap, fixed before merge and the correction posted to `#676` too. Entry lives in `docs/docs/reference/limitations.md` (canonical) + a summary row in `docs/LIMITATIONS.md`; `#676` REOPENED after an accidental auto-close]** ~~[LC-0 — 0.5d]~~ — `docs/LIMITATIONS.md` entry + a `#676` comment,
   because the programme is multi-week and `#676` is a live user-reported defect meanwhile.

**Two already-queued rows are load-bearing for this programme and must NOT be duplicated into it:**
`m-stdlib-reverse-delegates-to-builtin` becomes **REQUIRED, not optional** — `reverse`'s
`concat(growing, [x])` left-append shape stays quadratic even under cons cells, because cons-concat
copies the left spine each step; and `m-eval-tail-calls`, since `std/list`'s nine right-recursion
builders remain depth-capped even after both changes (the recursive call sits inside `++`, so TCO
does not reach it). That residual is named honestly in the roadmap rather than hidden.

**[LANDED 2026-08-21 (iter-241) — PR [#812](https://github.com/sunholo-data/ailang/pull/812) → squash `ab7b71ffa`; Gate 3b GREEN twice SHA-addressed on the full 40-char head (`5f090ea45` then `ac833780f`, 21 checks / zero not-green each, 4/4 required, `MERGEABLE/CLEAN`), both new CI steps confirmed RUN. Discriminator MEASURED before any code: 1,913 commits since 2026-06-01, 24 keyword-carrying, **19 ship code / 5 docs-only**; the gate reproduces it 5-red/19-pass with ZERO false positives. SWEEP CLOSED — beyond `#676`/`#612` no further closed-but-undone issues (3 more docs-only commits + 3 docs-shaped PRs, all benign, several only by ordering luck). Evaluator sonnet **83/100 PASS**; both BLOCKING findings reproduced first-party and fixed before merge — the text-file and unknown-arg guards existed but NO fixture killed them (rule 3j), fixtures 10→16, and merge commits were invisible without `-m`. `#676`/`#612` asserted still OPEN post-merge] m-commit-autoclose-guard — this loop's own record commits have
auto-closed two issues that were not done, one of them a live user-reported OOM.** GitHub closes on
`fix|close|resolve #N` in a commit message, PR title or PR body, without reading the sentence around
it — so a mission record *arguing about* a candidate fix closes the issue tracking it. Measured:
`#676` (live `from:email-parse` OOM, triaged REAL at HEAD) closed by `dedf3b91f`, a docs-only record
of 7 files and zero code, over the phrase *"the arena fixes #676 completely"*; and `#612` closed by
`7c7e5e58a`, which shipped one 636-line sprint plan while its deliverable never landed (`go/packages`
importers 0, `x/tools` in `go.mod` 0; controls 2 and 99). Audit of all `docs(mission)` commits since
June: 4 keyword hits, 2 benign, these 2 wrong. **Not a one-off**: the charter records a planner in
that same sprint stripping an auto-linked `Fixes #612` — the guard was applied to the document and
never to the commit message. Gate 4 now carries the controller-side scan (iteration 240's skill
edit); this row is the DURABLE half — a repo gate (commit-msg hook and/or a CI check on the PR body)
that fails loudly when a closing keyword appears in a commit that ships no code for that issue, plus
a sweep for further already-closed-but-undone issues beyond the four audited. Both reopened issues
are the regression fixtures. Scope: small, mechanical, no design doc needed beyond an AC list.

**[LANDED 2026-08-21 (iter-242) — PR [#814](https://github.com/sunholo-data/ailang/pull/814) → squash [`728ca8f3e`](https://github.com/sunholo-data/ailang/commit/728ca8f3e), 5 commits. Gate 3b GREEN twice SHA-addressed on the full 40-char head: PR head `48cbc17ad` **21 checks / zero not-green**, 4/4 required, `MERGEABLE/CLEAN`; merge `728ca8f3e` **20 checks / zero not-green**. Evaluator sonnet **91/100 PASS, zero blocking**. `std/list.reverse` now delegates to `_list_reverse`: O(n²)+depth-O(n) → O(n)+O(1) stack. Pinned by a **DEPTH** discriminator, not a timing one — a 20,000-element reverse errors `RT_REC_003` on the old form and returns `7` delegated, measured in both directions BEFORE the sprint was specified, so it cannot flake on a loaded runner. **Principle-3 half:** the audit found **31** registered `_list_*` builtins, only **12** called from `std/`, **19 with zero callers**; the other 18 are deliberately NOT delegated (each needs its own semantic-equivalence check → `m-stdlib-list-delegation-sweep` below) and are instead pinned by `TestEveryListBuiltinIsDelegatedOrExplained`, which requires each to be delegated or carry an explicit DELEGATION-CANDIDATE / NOT-NEEDED reason. **The evaluator falsified two of that gate's claims about itself by ADDING code rather than removing it** — a builtin registered as `Name: someConstant` was invisible to the AST scan, and a comment mentioning `_list_reverse(` laundered a real revert — both fixed before merge by reading the LIVE registries (neither complete alone: 18 and 26, union 31) and stripping comments. 5 mutants, all red, all restores sha256-identical. Measured base at HEAD, first-party: n=2000 **4.12 s / 130.7 MB** against a 0.03 s / 87.6 MB control; the designer's iter-228 figure (10.2 s / 172.7 MB) was NOT re-derived and the gap is unresolved] ~~[NEXT — cheap, first-party, iteration 228]~~ m-stdlib-reverse-delegates-to-builtin — AILANG shipped an
O(n) `reverse` that nothing calls, while `std/list.reverse` is O(n²) and non-tail-recursive.**
`std/list.ail:29-35` is `match xs { [] => [], [x, ...rest] => concat(reverse(rest), [x]) }`, whose own
source comment says "not tail-recursive". The iterative `_list_reverse` builtin is registered at
`internal/builtins/list.go:550` and has **0** callers in `std/` (same-scope control: `_list_map` → 1).
So `reverse` is quadratic *and* capped by the depth-10,000 wall, on the function that terminates the
canonical `keepExisting(rest, p :: acc)` accumulator idiom `#676` reports. Independent of `D-19`: this
is a one-line delegation plus an allocation-pinned regression arm, and it does not touch the list
representation. Measured base before writing an AC — the designer recorded 10.2 s / 172.7 MB at
n=2,000. Do NOT bundle: it lands whichever way `D-19` is answered.

**[LANDED 2026-08-21 (iter-244)] m-stdlib-ail-suite-enumerator-blind — both loops now enumerate with a sorted `find -L` against an EXACT count pin, and the evaluator's BLOCKING find (a committed SYMLINKED fixture is type `l`, so `-type f` skipped it while both counts still agreed with the pins) was reproduced first-party and fixed in round 2; dangling symlinks rejected by path, file list read from a temp file so a space-bearing path survives, missing root reports its own failure. 11 mutants red, base+final green, the two symlink arms red because the suites ACTUALLY RUN. Residual DECLARED in the target's comment: matching stays case-sensitive (`*_TEST.ail`, `*.EXPECTED`). PR [#816](https://github.com/sunholo-data/ailang/pull/816); evaluator sonnet 85/100. Original row: the
required `make test-stdlib-ail` gate cannot see a fixture that is renamed or moved, and says rc=0
while blind.** Found by iteration 243's judge applying rule 3a(i-e) one step further than the
controller did: the controller's ADD-direction check stayed inside `tests/stdlib/` and correctly saw
the enumerated count move 2 → 3, but a fixture placed in `tests/stdlib/subdir_probe/` is invisible —
the target reports **rc=0, "3 .ail test suite(s) passed"**. Reproduced first-party at `705e5f6b6`
(`tests/stdlib/probe243/`, count unmoved, rc=0, removed after). Two causes, both at
`make/test.mk:266,274`: the glob `tests/stdlib/*_test.ail` is **non-recursive**, and the anti-vacuity
floor is `[ suites -ge 1 ]` rather than an exact count — so the removal direction is undefended too
(renaming a fixture drops the count to 2 and still exits 0). PRE-EXISTING, not introduced by
iteration 243. Why it matters now rather than in the abstract: the row directly below will add ~13
more fixtures of exactly this shape, and each one's protection would be contingent on its filename
never drifting. Fix shape: pin an exact expected suite count (or a manifest), and/or make the
enumeration recursive; the `.expected` pairing loop below it has the same two properties and should
be checked in the same pass. Small — well under a day. Independent of `D-22`.

**[PARTIALLY LANDED 2026-08-21 (iter-245) — PR [#817](https://github.com/sunholo-data/ailang/pull/817)
→ squash [`d8f07c9e5`](https://github.com/sunholo-data/ailang/commit/d8f07c9e5); 21 checks / zero
not-green, 4/4 required. Evaluator sonnet **93/100 PASS**, zero blocking.
**THE ROW'S SCOPE BELOW IS WRONG AND WAS FALSIFIED AT GATE 2, BEFORE ROUTING.** It scoped **13
delegation candidates, ~2–3 days**. Only **TWO** of the thirteen can be delegated at all. The two
registries the gate unions describe **different execution paths**, which the exemption table never
distinguished: `AllNames()` (the interpreter's spec registry) holds **18** `_list_*` names,
`GetBuiltinNames()` (codegen metadata) holds **26**, union **31** — and only `_list_take` and
`_list_drop` are in the RUNTIME one. The other eleven (`any`, `zip`, `foldr`, `findIndex`,
`flatMap`, `sortBy`, `mapE`, `filterE`, `foldlE`, `flatMapE`, `forEachE`) are **codegen-only: the
interpreter has no implementation**, so delegating them would not make `std/list` faster, it would
**break it**. Three-arm discriminating probe on a fresh binary, `ailang run --caps IO`:
`_list_drop(1,[1,2,3])` → rc=0 `[2, 3]`; `_list_reverse` (control, already delegated) → rc=0
`[3, 2, 1]`; `_list_zip([1,2],[3,4])` → **rc=1 `undefined variable: _list_zip`**. So the blocker was
never "the `std/list` form is recursive" — it is a missing interpreter implementation, which is
materially larger work. **Shipped:** `std/list.drop` delegates (the recursive form recursed `n` deep,
so `drop(12000, <16384-element list>)` died `RT_REC_003`), pinned by a new `.ail` fixture under the
required `make test-stdlib-ail` asserting `length(drop(12000, big)) == 4384` — the judge confirmed it
reds with `RT_REC_003` at the exact assertion when the delegation is reverted, and that the make
target halts on first suite failure rather than hiding it. `STDLIB_AIL_SUITES_EXPECTED` 3 → 4
(equality, not a floor — iteration 244's gate correctly forced the move). **And the classification is
now machine-checked**: each exemption carries a category (`DelegableNow`/`NoRuntimeImpl`/`NotNeeded`)
plus a runtime-backed flag, asserted against `AllNames()`, so a wrong classification reds instead of
reading as documentation. 4 mutants, each LANDED (sha256) + BUILDS asserted BEFORE any test result,
each a **sole killer** (`-skip` the gate → rc=0): misclassify `_list_zip` → reds naming it; register a
new codegen-only builtin → count **31 → 32** and reds; revert the delegation → reds naming
`_list_drop`; leave `_list_drop` exempted → reds `stale exemption`. A removal proves the check FIRES;
only the addition proves it LOOKS. The evaluator re-derived the 18/26/31 counts and **all 17
exemption rows name-by-name** from the live registries, independently] m-stdlib-list-delegation-sweep — 18 registered `_list_*`
builtins have ZERO callers in `std/`, and several shadow a recursive `std/list` export that is
asymptotically worse than the builtin beside it.** Measured at `728ca8f3e` by
`TestEveryListBuiltinIsDelegatedOrExplained` (`internal/builtins/stdlib_delegation_test.go`),
which enumerates from the union of the two live registries: **31** registered, **12** with `std/`
call sites. `_list_reverse` was the 19th and landed at iteration 242. The remaining 18 are already
classified in that test's exemption table, so this row is enumerated rather than exploratory:
**DELEGATION-CANDIDATE (13)** — `take` (`[x] ++ take(n-1, rest)`, quadratic), `drop`, `foldr`,
`sortBy` (also builds on the recursive `take`/`drop`/`mergeBy`), `zip`, `any`, `findIndex`,
`flatMap`, and the five effectful `mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE`;
**NOT-NEEDED (5)** — `head`/`tail` (already O(1) via pattern matching), `last` (already composes
the delegated `_list_length`/`_list_nth`), `contains` (no `std/list` counterpart; `member` uses
`_list_member`), `extract` (no `std/list` operation at all). Scope: **each delegation needs its own
semantic-equivalence verification** — argument order, empty/edge cases, effect-row behaviour for
the `*E` variants, and whether the builtin's error text matches — so this is a sequence of small
independent changes, not one sweep commit. Deleting an exemption row is the acceptance signal: the
gate reds if a name is exempted after it gains a call site. Independent of `D-22` and of the
cons-cells programme. Estimate ~2–3d for the 13, splittable.

**[NEXT — first-party, iteration 245; surfaced by the sprint evaluator] m-stdlib-take-recursion —
`std/list.take` still fails `RT_REC_003` on large `n`, the same crash `drop` was just fixed for, and
the delegation that would fix it would make a `pure` function write to stderr.**

> Measured twice, independently: the controller at Gate 2 before routing, and the iteration-245
> evaluator, which correctly called the first record of it **under**-stated. `take(12000, <16384-element
> list>)` → `RT_REC_003: max recursion depth 10000 exceeded`. `std/list.ail:95` is still
> `[x] ++ take(n - 1, rest)` — byte-for-byte the shape `drop` had.
>
> The obvious fix is the one `drop` got, and it is blocked on a real behavioural delta rather than on
> effort. **`_list_take` is the ONLY list builtin that writes to stderr** (control: exactly **1**
> `Fprintf` in `internal/builtins/list.go`, at `:645`). It fires on
> `len > 10000 && n < len/2`, and measured, `_list_take(100, <16384-element list>)` emits a two-line
> `note:` recommending `takeFlatMap`/`takeMap`. Because `_list_take` has **zero** `std/` call sites
> today the note is dormant; delegating ACTIVATES it on the common `take(small_n, big_list)` call,
> making a function declared `pure` emit output. Note the advice is also inapplicable to the call that
> would trigger it — `take(n, xs)` has no `f` to fuse — which is the rule-3k emitted-string class this
> mission already knows.
>
> So this is a decision, not a sweep: (a) delegate and delete the note; (b) delegate and gate the note
> behind an opt-in; (c) give `_list_take` a quiet sibling builtin and delegate to that; (d) rewrite
> `std/list.take` iteratively in AILANG without delegating. The exemption row records the crash and
> the deferral reason verbatim so the next reader does not re-derive either.

**[IN-SPRINT-PARTIAL 2026-08-21 (iter-246) — the DISPATCH half landed as PR
[#818](https://github.com/sunholo-data/ailang/pull/818); the DIFFERENTIAL half is still open, and the
count in the row below is **13**, not eleven (`_list_last` and `_list_tail` were missed — measured
first-party against both live registries: runtime 18, codegen 26, codegen-only 13)] m-list-builtins-codegen-only — eleven `_list_*` builtins exist
for generated Go only, so the interpreter cannot call them and `std/list` cannot delegate to them.**

> `_list_any`, `_list_zip`, `_list_foldr`, `_list_findIndex`, `_list_flatMap`, `_list_sortBy`,
> `_list_mapE`, `_list_filterE`, `_list_foldlE`, `_list_flatMapE`, `_list_forEachE` — all present in
> `GetBuiltinNames()`, all absent from `AllNames()`. Enforced since iteration 245: the gate reds if any
> is reclassified as runtime-backed.
>
> Consequence beyond delegation, and the reason this is a row rather than a note: **the same `std/list`
> function has two implementations with no gate that they agree** — a recursive AILANG one the
> interpreter runs, and a Go codegen helper compiled programs run. Nothing tests them against each
> other, so `ailang run` and a compiled binary can silently disagree on `sortBy`, `zip`, `foldr` or any
> of the eleven. That is a soundness question and it is worth measuring BEFORE writing eleven
> interpreter implementations: a differential fixture per name (same input, `run` vs compiled, byte-equal
> stdout) is cheap and would either close the row or turn it into a bug list. Do that first; the
> implementations are the expensive half and may not all be wanted.
>
> Estimate unknown until the differential runs — deliberately NOT the ~2–3d the superseded row assumed.

**[NEXT — iteration-246 evaluator, round 2] m-list-option-helpers-unwrapped — the compiled `ListHead`
helper does not wrap its result in `Option`, so `std/list.head`/`tail`/`nth` panic in a compiled
binary where the interpreter returns `Some`/`None`.**

> Found by the iteration-246 evaluator while sweeping all **465** `.ail` files under `examples/` and
> `tests/` on a pre-sprint and a post-sprint binary. It is **latent today** only because
> `examples/runnable/list_patterns.ail` was blocked earlier by an unrelated `undefined: utf8` — which
> iteration 246 incidentally fixed, so this one is now reachable. `git diff 8040dfd41 624629a37`
> confirms the sprint touches neither helper, so it is pre-existing, not a regression. Same family as
> `m-codegen-claim-must-match-source`: a Go helper standing in for an AILANG function with a
> different *type*, not merely different performance.

**[NEXT — iteration-246 evaluator, round 2; small] m-golden-fixture-for-pseudo-module-builtins — the
`not(x)` regression pin hand-builds Core IR, so a parser or op-lowering rename would walk past it.**

> Iteration 246's two pseudo-module tests construct `VarGlobal{Module: "$builtin", Name: "not_Bool"}`
> directly and assert the helper is referenced AND emitted — genuine sole killers (both red under a
> mutant that reproduces the bug, `-run`-scoped, no collateral). What they do not do is go through
> parse → elaborate → op-lowering from real `.ail` source, and `tests/golden/codegen/` has **no**
> fixture exercising `not(...)` or a bare string `==`, which is exactly why 22 CI checks were green
> over a compiled `not(true)` that could not build. Add the fixtures; the gap is the corpus, not the
> assertions.

**[NEXT — first-party, iteration 246; the evaluator demonstrated it end-to-end] m-codegen-claim-must-match-source — the
codegen resolution gate cannot tell a correct stdlib claim from a fabricated one, because it checks
IDENTITY and never SEMANTICS.**

> Iteration 246 made codegen resolution module-qualified, which closes the *cross-module* half: a
> `std/list` call can no longer be lowered to a `std/string` builtin. What it cannot check is a spec
> that claims a genuine export **of its own module**, with a matching family prefix, and wrong
> behaviour. The iteration-246 evaluator proved it: a fabricated `_list_evilZipWith` claiming
> `std/list.zipWith` (a real, unclaimed export) with `return xs` LANDED (sha `5e44be32…` →
> `c6a78903…`), BUILT, and left `TestStdlibCodegenResolutionIsModuleQualified` **PASSING**, while the
> interpreter printed `[11, 22, 33]` and the compiled program printed `[1, 2, 3]`.
>
> The gap is DECLARED in that test's own comment rather than assumed away, which is the cheap half.
> The durable fix is a **source-level invariant**: a spec claiming `std/M.f` must either (a) be a
> real delegation — `std/M.ail`'s `f` calls that builtin — or (b) be an explicitly declared
> codegen-only *substitution* with a reason. (b) is exactly the set the row above enumerates, so the
> two rows share an instrument: the substitution table IS the list of places where `ailang run` and a
> compiled binary can disagree, and once it is explicit, the differential fixtures have a
> denominator. Do this row and the one above together, or the second one first.

**[LANDED 2026-08-22 (iter-247) — PR [#822](https://github.com/sunholo-data/ailang/pull/822) → squash
[`058feebc3`](https://github.com/sunholo-data/ailang/commit/058feebc3), 6 commits. Evaluator sonnet **three
rounds, 78 → 80 → 83/100 PASS**, each in its own worktree, each of the first two returning a real BLOCKING
defect that was reproduced first-party before being acted on. The row said cosmetics; the measured headline is
a **determinism defect** — the generated `Show` fell through to Go's `%v`, an ADT is a pointer, and a compiled
binary printed its own heap addresses, so three runs of one unchanged binary gave three different strings
(control: a list-only program byte-identical across runs). The interpreter half was 1-of-7, not 1-of-1:
`showValue` handled **10 of 17** live `eval.Value` types. Both now render the same AILANG surface syntax, all
17 handled with exhaustiveness machine-checked against the live source, and `tests/golden/codegen` **runs**
what it compiles for the first time — the differential is the instrument the two rows above were waiting for.
Rounds 2 and 3 each closed another divergence (named-record key sort; depth/width caps; float `%g` giving
`+Inf.0`/`1e+21`) **and widened the fixture set each time, 1 → 4 → 5** — a fix without a fixture leaves the
next member of the class equally invisible, proved three times in one sprint. Gate 3b caught a Windows red no
darwin command could see: the new differential exec'd `fixture-bin` with no `.exe`. Executor deviation (a
distinct generated `Tuple` type) adjudicated by a 465-file compile sweep run after every round — 0, 0, and 2
divergences, both of the last being `pre=1 → post=0` improvements] m-show-diverges-between-run-and-compile — `show` rendered the
same value differently under `ailang run` and in a compiled binary, and cannot render tuples at all
under the interpreter.**

> Measured at `8040dfd41`, same program both ways: `show([1, 2, 3])` → `[1, 2, 3]` interpreted,
> `[1 2 3]` compiled (Go's `%v`). `show(zip([1,2,3],[4,5,6]))` → `[<*eval.TupleValue>,
> <*eval.TupleValue>, <*eval.TupleValue>]` interpreted against `[[1 4] [2 5] [3 6]]` compiled.
>
> Two consequences worth more than the cosmetics. **(a)** The interpreter's tuple rendering is a
> plain defect — a user-facing `show` that prints a Go type name. **(b)** It makes the obvious
> instrument for the two rows above — "compile it, run it, diff stdout byte-for-byte" — report a
> divergence on *every* list-returning fixture, so the real semantic disagreements would be buried
> in noise. Either fix `show` first or have the differential compare structure rather than rendered
> text; note which, because a fixture harness written without knowing this will look like it works.

**[M1+M2+M3 LANDED 2026-08-22 (iterations 249, 250) — PR #826 → `d3b0185f5`, PR #828 → `bbf672df0`; M4 [NEXT]] m-array-show-diverges-run-vs-compile — `show` on an
array prints `#[1, 2, 3]` under the interpreter and `[1, 2, 3]` compiled, and no change to `Show` can fix it.**

> The last surviving member of the class iteration 247 closed. Reproduced both ways, one line of user code:
> `show(fromList([1,2,3]))` → `#[1, 2, 3]` interpreted, `[1, 2, 3]` compiled.
>
> This one is NOT a `Show` bug, which is why it was scoped out rather than forced into that sprint.
> `internal/gen/golang/codegen_ops.go:357` compiles `Array` to the **same Go type as `List`** — its own comment
> says so: *"M-TYPE1: Arrays use the same Go representation as lists (slices)"*. So by the time a value reaches
> the generated `Show`, there is no runtime tag left to discriminate on, and `showReflect`'s
> `reflect.Slice, reflect.Array` branch cannot tell them apart even in principle.
>
> The fix is a distinguishing wrapper at the codegen representation layer — exactly the trick iteration 247's
> executor already used for tuples, which is evidence the approach works and a rough guide to its cost. Note
> that M-TYPE1's representation-sharing was a deliberate decision, so this needs a design pass rather than a
> patch: check what else depends on Array and List being the same Go type before wrapping. The differential
> harness added in 247 gives it a ready-made acceptance test — add a fixture and the count literal moves 5 → 6.
>
> **ITERATION 248 — design doc + sprint plan landed, quorum-cleared via the ratified carve-out.**
> [`planned/m-array-show-diverges-run-vs-compile.md`](planned/m-array-show-diverges-run-vs-compile.md)
> + its `-sprint-plan.md` companion (plans travel with their doc) and
> `.ailang/state/sprints/sprint_M-ARRAY-SHOW-DIVERGES-RUN-VS-COMPILE.json`. **Next iteration executes
> M1/M2**; M1–M4 is a 4-day sprint that does not fit one slot.
>
> **ITERATION 249 — M1 AND M2 LANDED.** PR [#826](https://github.com/sunholo-data/ailang/pull/826) →
> squash [`d3b0185f5`](https://github.com/sunholo-data/ailang/commit/d3b0185f5), three bisectable commits,
> executor `codex:gpt-5.6-sol`, evaluator `sonnet` **93/100 PASS round 1, zero blocking**.
> Arrays now carry a distinct Go identity (`type ArrayVal []interface{}`, the in-repo `Tuple` precedent),
> `showValue` renders them `#[…]`, the helpers preserve that identity, and the five literal converters
> `panic` with a converter-specific message instead of `return nil`. End to end both backends now emit
> identical bytes for the doc's repro (`cmp` rc=0, against rc=1 at base). `expectedDifferentialFixtureCount`
> is **6**; the fixture count steps to 7 at M3.
>
> **M3/M4 REMAIN and are this row's [NEXT].** M3 is the typed-aggregate work — both array type mappings
> (`types.TArray`, `ast.ArrayType`) to `ArrayVal`, plus the ADT-constructor and both record-generation paths,
> which is what closes the measured `MkBox(Array[int])` user-visible defect (doc VL-16). The plan re-scoped
> it from 1 day to 2 and it carries the whole sprint's risk (plan §2 R2/R3/R4). M4 is the second fixture,
> count 6 → 7, plus CHANGELOG.
>
> **ITERATION 250 — M3 LANDED.** PR [#828](https://github.com/sunholo-data/ailang/pull/828) →
> squash [`bbf672df0`](https://github.com/sunholo-data/ailang/commit/bbf672df0), executor
> `codex:gpt-5.6-sol`, evaluator `sonnet` **91/100 PASS round 1, zero blocking**. Both array type
> mappings return `ArrayVal`; the ADT-constructor and both record paths now emit a bare
> `tmp.(ArrayVal)` assertion instead of a `ConvertTo*Slice` call, and all three measured cases
> (`MkBox(Array[int])`, `{items: Array[int]}`, `MkPlan(Array[Dir])`) agree byte-for-byte across
> backends. `expectedDifferentialFixtureCount` is **7**. Lists pinned untouched by one generated
> line carrying both halves: `&Both{Arr: tmp8.(ArrayVal), Lst: ConvertToInt64Slice(tmp9)}`.
>
> **THE SPRINT'S SCOPE CLAIM WAS WRONG AND THAT IS THE FINDING.** The doc's Dependencies line says
> *"self-contained to `internal/gen/golang/` plus two golden fixtures"* and the plan's R3 enumerated
> **2** array mappers. The executor found a third and fourth in `cmd/ailang/compile_types.go`
> (`ailangTypeToGo`, `ailangTypeToGoWithValueRecords`) — load-bearing, adjudicated by measurement:
> that file alone reverted, mutant LANDED (sha256) and BUILDS (rc=0), reds
> `TestTypedAggregateArrayGeneratedOutput` and the `show_differential_array_adt_field` subtest while
> the delivered tree is rc=0. `adt.go` writes the *declaration*, `compile_types.go` populates the
> *call-site registry*; two sources of truth for one fact, and the generated Go does not compile
> unless they agree. The judge then found a fifth and sixth in `internal/gen/lower/typeres.go`
> (inert behind the off-by-default `--emit-go-v2`); a repo-wide re-derivation puts the surface at
> **7+ sites across 4 packages**. Filed as `m-duplicate-go-type-mappers`.
>
> **M4 GAINED A FIRST TASK: two lines shipped in M3 are pinned by nothing.** Reverting `types.go`'s
> `TArray` case reds only its own unit test — golden and differential stay rc=0 — because
> `funcTypeOverrides` short-circuits `ExtractFuncSignature` for every AST-declared function.
> Reverting `IsUserDefinedType`'s `"ArrayVal"` case reds **nothing at all**. Both found by the judge
> and reproduced first-party. SonarCloud independently flagged the same gap (**66.7% coverage on new
> code** vs an 80% bar — *not* the duplication signal iterations 247/249 met, and `dev` is `success`,
> so PR-scoped). Filed as `m-array-typed-boundary-lines-unpinned`.
>
> **Pre-existing, NOT ours, two-arm verified:** `[Array[int]]` (a list whose element is an array)
> panics in the compiled backend — base binary `not [][]int64`, post-M3 binary
> `not []main.ArrayVal`, same defect either side. Filed as `m-list-of-array-compiled-panic`.
> **Bonus:** `Array[Array[int]]` was broken at base and is now fixed.
>
> **Do NOT re-inherit VL-9's "7 silent sites": it is 5.** The two template-loop converters
> (`codegen_runtime_slices.go:192`/`:315`) already fail loudly and are the model to copy, not a defect to
> fix — the planner measured this at iteration 248 and iteration 249 confirmed it at the emitter: all **7**
> remaining `return nil` writes in that file are legitimate `v == nil` guards, **zero** are `!ok` branches.
>
> **This row's own stated blocker is REFUTED — do not re-inherit it.** "M-TYPE1's
> representation-sharing was a deliberate decision" is false: `git show --stat 743f6a539` is **one
> file**, `internal/types/unification.go`, **8 insertions**, type-checker only, and it never touched
> `internal/gen/golang/`; the `// M-TYPE1:` codegen comments came from `ea88158ef` / `474adf0cf`
> (`git log -S`). It is an implementation convenience wearing an unrelated fix's tag. M1 rewrites it.
>
> Also corrected here: it is **not a soundness bug** (`ailang check` rejects List-where-Array,
> *"cannot unify array type Array[α] with [α]"*); the fix shape is a **defined slice type**
> `type ArrayVal []interface{}` — the mechanism 247 already proved for tuples
> (`type Tuple []interface{}`, `case Tuple:`), **not** a struct wrapper, which is what the designer
> refuted in the controller's own framing; and the count literal moves **5 → 7**, not 5 → 6, because
> the typed ADT/record-field path needs its own fixture — `type Box = MkBox(Array[int])` diverges in
> one line of user code via `ConvertToInt64Slice`. A quorum reviewer caught that and the controller
> confirmed it first-party.

**[NEXT] m-array-typed-boundary-lines-unpinned — two lines shipped in M3 that no test kills, found by two independent instruments.**

> Filed by iteration 250, reproduced first-party before filing. Two mutants, each LANDED (sha256) and
> BUILDS (`go build` rc=0) asserted before any test result was read:
>
> - Revert `internal/gen/golang/types.go`'s `case *types.TArray:` to `[]%s` → reds **only**
>   `TestTypeMapper_MapType_ArrayPreservesIdentityAndListControl`. The whole golden + differential
>   suite stays **rc=0 with 0 FAIL**. Root cause (judge's, worth re-deriving): every AST-declared
>   function registers a `funcTypeOverrides[name]` entry from `cmd/ailang/compile_types.go`, which
>   short-circuits `TypeMapper.ExtractFuncSignature` before `types.go`'s case is ever consulted.
> - Revert `IsUserDefinedType`'s `"ArrayVal"` case (`adt.go`) → reds **nothing anywhere**: unit rc=0,
>   golden rc=0, `make verify-examples` rc=0 at `211 passed, 0 failed, 6 skipped`. It is not dead —
>   it changes generated code for the `[Array[int]]` shape — it is simply unpinned.
>
> **SonarCloud found the same gap independently**, which is the part worth keeping: PR #828's red was
> **66.7% coverage on new code** against an 80% bar, *not* the new-code duplication signal iterations
> 247 and 249 met, and `dev` was `success`, so it was PR-scoped rather than a standing red. Checking
> the condition instead of inheriting the previous framing is what surfaced the convergence.
>
> Deliverable: either wire an integration path that genuinely exercises `ExtractFuncSignature` for
> arrays and pin the `IsUserDefinedType` line behaviourally, or establish that the `types.go` path is
> permanently unreachable via `funcTypeOverrides` and **say so in the code**. A named dead branch is
> cheap; an assumed-live one is not. **Take this as M4's first task**, not as a follow-up.

**[NEXT] m-duplicate-go-type-mappers — one language, seven places that decide what `Array[T]` is in Go.**

> The class defect behind iteration 250's scope miss. The design doc called that sprint
> "self-contained to `internal/gen/golang/`" and its plan enumerated **2** array→Go mapping sites.
> Re-derived repo-wide at `bbf672df0` (`grep -rn 'ast.ArrayType\|types.TArray' --include='*.go'`,
> non-test; control: the same grep for `ast.ListType` returns 20, and `test -d internal/gen/lower`
> confirms the scope exists): the Go-codegen mapping surface is at least
> `internal/gen/golang/types.go:79`, `internal/gen/golang/adt.go:382`,
> `cmd/ailang/compile_types.go:320` and `:387`, `internal/gen/lower/typeres.go:59` and `:198`, and
> `internal/gen/lower/expr.go:516` — **7 sites in 4 packages**. The `internal/gen/lower` pair still
> maps arrays to a plain slice and is inert only because `--emit-go-v2` is off by default and does
> not currently build at all; it is a landmine, not a fix.
>
> This is rule 3a(i-e) in the wild: every mutation drill in the sprint was removal-shaped, and no
> removal can detect a mapper nobody enumerated. Deliverable is the **enumeration made complete by
> construction** — one mapper the others delegate to, or a test that fails when a new `*ast.ArrayType`
> case appears anywhere outside it — not a seventh hand-audited list.

**[NEXT] m-list-of-array-compiled-panic — `[Array[int]]` panics in the compiled backend. PRE-EXISTING.**

> Found by iteration 250's evaluator, and the pre-existing verdict is two-arm measured rather than
> assumed: identical source, base binary (built from `b59255831`) panics
> `interface conversion: interface {} is []interface {}, not [][]int64`; the post-M3 binary panics
> `… not []main.ArrayVal`. Same defect, different message — **not** an M3 regression, which is why it
> did not block the merge. The interpreter is correct throughout (`[#[1, 2], #[3, 4]]`).
>
> Mechanism: `*ast.ListType`'s element recursion computes `elemType = "ArrayVal"`,
> `IsUserDefinedType("ArrayVal")` is false, so it falls to the plain `[]%s` branch giving `[]ArrayVal`
> with no registered converter, and the fallback is a bare panicking type assertion. Same family as
> the sprint plan's R5 (`std/array.empty`/`append` have no emitter) and R6 (`toSlice`'s silent nil) —
> file, do not widen. Note the reverse nesting `Array[Array[int]]` was broken at base and iteration
> 250 **fixed** it incidentally.

**[NEXT] m-array-record-slice-converter-arm-untested — a generated converter arm that no test reaches
and that may be unreachable by construction.**

> Filed by the iteration-249 evaluator (`sonnet`, 93/100 PASS, NON-BLOCKING) and not fixed there, because a
> fix without a reachability answer is guesswork. M2 added an `ArrayVal` acceptance arm to **both** template
> loops in `internal/gen/golang/codegen_runtime_slices.go` (`:192` ADT, `:315` record). The ADT arm is pinned;
> the **record** arm is the one mutant of six that **nothing killed** — removing it left the whole of
> `internal/gen/golang` and `tests/golden/codegen` green.
>
> The judge then tried to reach it from real AILANG source, twice — an ADT-constructor field and a plain
> record field, both `Array[Point]` — and **both routed to the ADT loop instead**, because `adt.go`'s
> `adtSliceTypes` registration intercepts any user-defined element type before `writeRecordSliceConverters`
> sees it. So the arm may be dead defensive code mirroring a proven pattern.
>
> **The deliverable is the reachability verdict, not a test.** Either construct a program that reaches
> `writeRecordSliceConverters` (then pin it), or establish it is unreachable and say so in the code — an
> undeclared unreachable branch is a guard nobody is protecting. Note the enumerator hazard: a mutation that
> removes a branch proves the check FIRES; only reaching the branch proves anything LOOKS at it.

**[NEXT] m-emitter-lint-evadable-by-rewording — the source-text gate that a rename walks past.**

> Filed by the iteration-249 evaluator, NON-BLOCKING, and the sprint plan predicted it in writing: M2.6
> asserts "the count of `return nil` writes in the `!ok` branches is 0" by inspecting emitter source text.
> The judge demonstrated the evasion — it rewrote `ConvertToInt64Slice`'s `default:` panic to a silently
> different textual form (`_ = x`), and the **lint passed** while the behavioural test
> `TestGeneratedSliceConverters` caught it as the sole failure.
>
> That is the intended layering (the plan calls the lint "deliberately secondary") and it is worth recording
> as a class rather than an instance: this repo has several source-text gates standing in for behavioural
> ones. The deliverable is an audit — for each such gate, either name the behavioural test that backstops it,
> or replace it. A grep can always be satisfied by moving the text; the mission's own recurring shape is
> *guard the helper, miss the call site*.


**[NEXT — first-party, iteration 247; small, and a decision as much as a fix] m-sonar-differential-fixture-duplication — SonarCloud reds the
repo's new-code duplication gate at 35.7%, and the duplication is the differential fixtures, which are
near-identical on purpose.**

> `#822` moved SonarCloud from `success` on the base to `failure`, on one condition: *35.7% duplication on new
> code (required ≤ 3%)*. Non-required, so it did not block the merge, but base was green and this is ours.
>
> **ITERATION 248 GATE-1 CORRECTION — the red is PR-scoped, and there is NO standing dev red.**
> Measured across the four most recent `dev` commits (`404226a48`, `058feebc3`, `6cb53ddd1`,
> `eadad9fef`): SonarCloud is **`success` on every one**, including `058feebc3`, the squash-merge of
> `#822` itself. PR analysis and branch analysis use different new-code periods, so the 35.7% was a
> property of the PR's diff, not of `dev`. Option (c) ("accept a standing red") is therefore moot,
> and the row's real question is narrower: whether *future* PRs touching these fixtures keep
> tripping the gate. Re-measure on the next fixture-touching PR — the array sprint above adds two —
> before spending an iteration on (a) or (b).
>
> The duplicated lines are the five `show_differential_*.ail` fixtures. They are minimal near-identical
> programs that vary exactly one thing each, and that is the point: rounds 2 and 3 of that sprint exist
> *because* a single fixture could not see the named-record, depth/width and float divergences. Collapsing them
> to satisfy the metric would undo the sprint's central lesson, so this is not a straightforward cleanup.
>
> Options, in rough order of preference: (a) exclude `tests/golden/codegen/*.ail` fixtures from Sonar's
> duplication analysis, which is what the `sonarcloud-triage` skill is for; (b) generate the fixtures from a
> table so the shared scaffolding exists once — but check first whether that weakens the gate, since a
> table-driven fixture is one artifact again; (c) accept a standing red, which this mission should not do
> quietly. Decide and record which, rather than letting the gate sit red for six commits the way Gate 1 caught
> it doing before.

**[NEXT — first-party, iteration 248; PRE-EXISTING, small, verified at `404226a48`] m-prelude-diagnostic-names-absent-module — the
equality type error tells the user to import `std/prelude`, and `std/prelude` does not exist in this stdlib.**

> Found while measuring a quorum reviewer's objection, not while looking for it. `[1,2] == [1,2]` fails
> `ailang check` with: *"No instance for Eq[[int]] in scope. Equality (==, !=) needs an Eq instance;
> [int] has none. **Import std/prelude**, or derive/define one."* Measured, same call, controls firing:
> `test -f std/prelude.ail` → **ABSENT** (control `test -f std/list.ail` → EXISTS); `import std/prelude`
> occurs **0** times across `examples/` and `std/` (control: `import std/` lines are plentiful); and
> typing it is itself a parse error — `IMP012_UNSUPPORTED_NAMESPACE`, namespace imports not supported.
> `ls std/` shows 45 `.ail` modules and neither `prelude.ail` nor `eq.ail` is among them; `instance Eq`
> appears **0** times in any of them (control: 45 files match `export`).
>
> So a first-contact diagnostic — this is what a user or an agent hits the first time they write `==`
> on a list — sends them to a module that cannot be imported and does not exist. That is worse than a
> missing feature, because it reads as a solved problem. Two things to decide, and they are different:
> **(a)** the message, which should name something real or say plainly that structural equality on
> lists/arrays is not available; **(b)** whether `Eq` for lists/arrays is a gap worth filing on its own
> — no stdlib module provides it today, so the answer the message implies exists, does not.
>
> Small and self-contained. Do NOT bundle it into the array-`show` sprint above; it was deliberately
> scoped out of that doc and recorded there as an adjacent finding (VL-17).

**[NEXT — first-party, iteration 246 evaluator; PRE-EXISTING, verified on `8040dfd41`] m-math-abs-stdlib-name-mismatch — `std/math.abs_Int`
and `abs_Float` do not compile: the codegen registry spells their `StdlibName` without the underscore.**

> `registry_codegen_math.go` registers `StdlibName: "absInt"` / `"absFloat"` while the real exports
> are `abs_Int` / `abs_Float`, so a compiled program calling either dies `undefined: AbsInt`.
> Reproduces identically before and after iteration 246, so it is not that sprint's regression. Small
> and self-contained; the natural companion to the row above, since a name that matches no export is
> exactly what the new export-set check was added to reject.

**[LANDED 2026-08-21 (iter-243) — PR [#815](https://github.com/sunholo-data/ailang/pull/815) → squash
[`705e5f6b6`](https://github.com/sunholo-data/ailang/commit/705e5f6b6), 3 commits. Evaluator sonnet
**97/100 PASS**, zero blocking. **THE FILED CHARACTERISATION BELOW IS WRONG AND WAS FALSIFIED IN BOTH
DIRECTIONS BEFORE ROUTING**: the `let`-binding is irrelevant — it fails with and without one, and
passes with and without one. The real variable is whether the module declares **any function of its
own**. Root cause: `pipeline.Run` wires the `$builtin` lookup only under `cfg.GlobalResolver != nil`
(`internal/pipeline/pipeline_module.go:442`) and in `ModeEval` evaluates `Core.Decls[0]` through it
(`:494-501`); the elaborator emits function/let decls before the trailing test-body block, so with no
functions `Decls[0]` IS the body. Any function absorbs index 0 and the executor's own correctly-wired
second pass then succeeds — which is the whole of the shape-dependence. Fixed by building a
builtin-backed resolver once per `Executor` and supplying it to **all four** `ModeEval`
`pipeline.Config` sites (Principle 3; `NewExecutor` is the sole construction path). Pinned at two
layers: a Go arm and a **function-less** `.ail` fixture under the required `make test-stdlib-ail`
(enumerated count moves 2 → 3). Both verified SOLE KILLERS by the controller — mutant asserted LANDED
+ BUILDS first; the `.ail` red set was enumerated by running the three suites individually, because
the make target short-circuits. Scope stated honestly: sites 95/468/645 changed for consistency with
**no reproducing arm**, and the `property`/`forall` path fails EARLIER (`PAR_UNEXPECTED_TOKEN`, source
synthesis — `runner.go:267,307,312`) so it is a separate defect, measured and left alone] 
— `ailang test` cannot resolve `$builtin.*` when a builtin-delegated stdlib call is `let`-bound
inside a `test { }` block.** Found in passing by iteration 242's evaluator while writing coverage
for the `reverse` delegation: the same call succeeds under `ailang run` and fails under
`ailang test` with `EVA002: module not compiled: $builtin`. Reproduced for `_list_reverse` **and**
for the untouched `_list_length`, so it predates that sprint and is not specific to either. Why it
matters beyond the one error: `std/list` is now deliberately moving *toward* builtin delegation
(the row above), so every function this row touches becomes harder to test under the product's own
test runner exactly as it gets faster — and `tests/stdlib/*_test.ail` is a required CI surface.
Iteration 242's correctness suite happens to sidestep it by not `let`-binding the call, which
nobody designed, so the coverage that exists is luck. Live-repro before routing (both runners, both
builtins) and check whether the two runners share a module-compilation path at all.

**[LANDED 2026-08-19 (iter-230) — PR [#788](https://github.com/sunholo-data/ailang/pull/788) → squash `98b704723`, head `9a25fa253` at 21 checks / zero not-green, 4/4 required. Confirmed at HEAD and SHARPER than filed: the only two mentions of tail recursion anywhere in `internal/eval` are a test comment and **the message itself**. Message now names remedies that exist — a smaller input, `std/list` `foldl`/`map` (both delegate to iterative builtins), or `--max-recursion-depth`; the substituted remedies were themselves ghost-disciplined first, after `^export func` returned 5 against `^export` = 40 (the real shape is `export pure func`). Pinned by two arms that take the remedy OUT of the product's own output and EXECUTE it, with an anti-vacuity floor. 2 mutants, both LANDED+BUILDS asserted before any test result: restoring the old message and advertising a fake `--enable-tail-recursion` each red exactly one arm, inverse `-skip` rc=0 both — two sole killers. Satisfies AC-8 of the superseded `m-list-cons-quadratic` doc] m-rt-rec-003-advertises-nonexistent-option.**
`internal/eval/eval_operations.go:58` tells the user to "enable tail recursion", an option that does
not exist anywhere — `grep -ril 'tail.?call' internal/eval/` → **0** (control `recursionDepth` → 2
files). This is the emitted-string class rule 3k names: the product hands a human an instruction and
nothing tests that it is actionable. Fix the message to name the real remedies
(`--max-recursion-depth`, or restructuring), and pin it with an arm that reads the message out of the
product's own output rather than reconstructing it.

**[BLOCKED ON `D-19`] m-eval-tail-calls — the tree-walking evaluator has no TCO, so the canonical
accumulator idiom is capped at 10,000 elements** (Sprint 2 of the `#676` pair; split from
`m-list-cons-quadratic` on the designer's recommendation and the controller's agreement — disjoint
subsystems, each honestly 3–4 days, neither subsuming the other). `grep -ril 'tail.?call'
internal/eval/` → **0**; the machinery exists only in `internal/vm/` + `internal/bytecode/` (9 files),
which `ailang run` does not use. Measured: the `#676` repro at n=12,800 fails
`RT_REC_003: max recursion depth 10000 exceeded` and succeeds under `--max-recursion-depth 200000`.
Ordered AFTER Sprint 1 because RSS is the binding harm at realistic n and `--max-recursion-depth` is a
real user-facing workaround for the wall, whereas the quadratic has none but rewriting user code.
Exit criterion is clean: the repro at n=12,800 under default flags prints `12800`. Sequencing depends
on `D-19` only in that answer **B** (cons cells) may change what the evaluator work has to preserve.

**[LANDED 2026-08-20 (iter-233) — PR [#796](https://github.com/sunholo-data/ailang/pull/796) → squash `547803584`; 4/4 required pass] m-ci-no-job-timeouts — a wedged step burns 6 h of a REQUIRED check with no signal**

> **Scope was 3.5x the row.** The row argues from `ci.yml`; measured repo-wide at pick time, `timeout-minutes` appeared **0** times across **all 10** workflow files (control: `runs-on` = **27**), so all 27 jobs inherited the 6-hour default. Shipped as one sweep (Principle 3): **27/27 jobs bounded** at ~2x their observed max over the last 18-20 successful runs; the 28th (`release.yml` `provenance`) is a reusable-workflow call, where GitHub rejects the key, and that exemption is **asserted by the gate**, not assumed. The three `apt-get` steps also carry 5-minute **step** bounds, so the wedged step is named rather than merely the job. Durable guard: `internal/cihygiene` under `make test`, parsing with a real YAML parser (rule 3j) with three anti-vacuity floors; **9 mutation drills**, 7 sole-killer + 2 broad-blast with enumerated red sets. Bounds validated by the PR's own runs (`test` 1059s/2700s). **Residual:** `release.yml`'s 7 bounds fire only on tags and are unexercised until the next release; `dashboard-ui-build`'s 20-min bound is still a guess, since that job has never run long enough to produce data.


> **First-party evidence, iteration 232** (this row was argued rather than measured until now). TWO
> workflows wedged in one iteration, both on `apt` package-install steps, neither declaring
> `timeout-minutes`: `CI/test` step 9 `Install z3` hung **>26 min** (attempt 1) and **17m37s**
> (attempt 2) against a control of **49s / 100s / 9s** on the last three completed dev runs; and
> `Deploy Documentation/docs-build` step 5 `Install jq` sat **1h30m**. `docs-gate` is a REQUIRED
> context, so the second one blocked the merge outright. The provider's status API read *All Systems
> Operational* throughout, and **attempt 3 cleared z3 in ~1 minute on a byte-identical tree** — the
> outcome divergence that pins it to the environment rather than any diff. Cost this iteration: three
> CI attempts and ~1h of wall clock, all of which a `timeout-minutes` on the install steps would have
> converted into a fast, legible red.
(first-party, iteration 227). `ci.yml` declares **`timeout-minutes` on no job at all** (`grep -c` → 0 over the file; control: `runs-on` appears throughout), so every job inherits GitHub's default **6-hour** limit. Measured today: `Install z3 …` — `sudo apt-get update && apt-get install z3`, an unbounded shell-out to a package mirror — hung **39+ minutes** on the `#784` run, having taken **1m41s** on the immediately preceding run (`32206877998`, 02:01:11Z → 02:02:52Z). Attribution is not a guess: the **dev** run for `171e2f2ef`, a SKILL.md-only docs commit touching no Go code, was wedged on the **identical step** in the same window, and a cancel+re-run of both cleared it on a byte-identical tree — outcome divergence with the code held constant. Provider status read *All Systems Operational* throughout, so a status-API check would NOT have caught it. Two consequences worth the row: (a) the loop's own Gate-3b bounded poll expires and reports a non-verdict while the run is neither green nor red for hours, and Standing rule 6 then costs an iteration; (b) `test` is a **required** context, so a wedged mirror blocks every merge in the repo, for both missions, with no alarm. Scope: add `timeout-minutes` to each job sized from observed p95 (the `test` job's own history is the data), and give the apt step its own bound — a package install that cannot finish in minutes will not finish. Do NOT simply retry: the failure to fix is the *unboundedness*, and a retry loop around an unbounded command inherits it. Note this is the same shape as the `AILANG_OLLAMA_*` streaming-deadline work — an unbounded external wait inside a job nobody is watching.

**[NEXT — bookkeeping, cheap, first-party iteration 234] m-mission-log-entry-numbering — the mission
log's entry numbers are no longer a usable index: there are TWO entries numbered `## 232`, and
230–233 are out of chronological order.**

> Measured at `746686cb6`: `grep -c '^## 232 — ' design_docs/v1-mission-log.md` → **2** (control:
> `'^## 233 — '` → 1). The collision is the iteration-232 record (`editor install vscode`, dated
> 2026-08-20) and the iteration-231 record (package imports, dated 2026-08-19); reading the headings
> in file order gives 225, 226, 227, 228, 229, **232**, 231, 230, **232**, 233, 234, so neither the
> number nor the position orders the log. Note also that entry number and iteration number drifted
> apart earlier (entry 230 = Iteration 229, entry 231 = Iteration 230) and then re-converged, so the
> two are not reliably offset by a constant either.
>
> Why this is a row and not an inline fix: renumbering historical entries would invalidate every
> cross-reference already written against them — the charter, the status archive and prior issue
> comments all cite entries by number. The durable options are (a) leave the numbers alone and add a
> generated index keyed on the `Iteration N` token that already appears in each heading, or (b)
> renumber once with a redirect table recorded in the archive. Choosing needs a design pass, not a
> controller edit. Cheap to close and it protects the loop's own memory: Gate 4 appends by "highest
> existing number", which a duplicate silently corrupts.

**[NEXT] m-quorum-artifact-path-is-cwd-relative — Gate 2's own quorum check reads a different
directory depending on where the iteration ran, so a four-times-reviewed doc reads as unreviewed.**
Found at iteration 261 while picking `#764`. `ArtifactDir = ".ailang/state/mission-quorum"`
(`internal/mission/quorum/artifact.go:14`) is a **relative** path resolved against the process CWD,
and `.ailang/` is gitignored (`.gitignore:82`; `git check-ignore` control fired), so a quorum
artifact lands wherever that iteration happened to run and **never travels with the repo**. Under
the pin-worktree driver regime the CWD is a pinned worktree, and record worktrees are removed after
their PR merges — so the artifact is routinely unreachable by the next iteration.
**Measured, with controls:** `m-serveapi-protocol-only-module` has had **four reviews** across two
full quorum rounds (both recorded in the doc's own quorum log, with per-reviewer costs in iteration
260's STATUS), and its artifact count is **ZERO** — zero in every worktree of this clone
(97 + 14 + 1 = **112** artifacts exist, so the scan sees positives) and zero across a bounded `find`
over `dev/sunholo-data`, `.ailang-driver-pin` and `/tmp` (control pattern returned **5**).
**Why it matters rather than being untidy:** Gate 2 mandates *"if the picked doc has NO quorum
artifact, run the text quorum BEFORE routing"*. Followed literally here, that spends real money
re-litigating a doc whose every objection is already answered — and worse, a fresh quorum on a
revised doc can raise new objections and re-park an item the human has just unblocked. Iteration 261
avoided it only because the doc's own log recorded the rounds.
**Scope:** anchor the artifact directory to the mission repo rather than the CWD (the
`--artifact-dir` flag already exists, so the smallest fix is to make the *default* repo-anchored,
e.g. via `git rev-parse --show-toplevel`), and decide whether quorum artifacts should be tracked —
`.ailang/state/sprints/` shows the precedent, where **57** sprint JSONs are committed despite the
same blanket ignore. Include the Gate-2 wording so a genuinely-absent artifact and an
unreachable-by-construction one stop being the same observation. **Instance 1**; recorded rather
than skill-edited, per the ≥2-friction bar.

**[NEXT] m-launchd-probe-timing-arm-flaky-on-runners — a non-required CI job that fails, passes
and cancels on the same tree, measured across four runs.** Found at iteration 259 while landing a
**docs-only** diff (5 markdown files, zero `tools/launchd` files — control: 5 `design_docs` hits,
0 `tools/launchd` hits). The `launchd drivers (bash 3.2)` job's motoko-connection-probe suite arm
`bounded termination deadline refuses` failed with `lacked expected message: bounded termination
deadline`, alongside `process-tree discovery deadline expired` / `INSTRUMENT FAILURE: process-tree
discovery failed`. **Four observations, three different outcomes, on effectively identical trees:**
PR head attempt 1 → **FAIL**; PR head attempt 2 → **PASS**; merge commit attempt 1 → **CANCELLED**
after the job ran **15m20s** (13:53:23Z→14:08:43Z) while all seven sibling jobs succeeded; merge
attempt 2 → **PASS**. Locally the whole target is `make test-launchd-drivers` **rc=0, 74 PASS /
0 FAIL**. The job is green on `bc3f80884`, `9417c5ff7` and `ad6d08050` (`not ok - bounded
termination` count **0** on each — and note the naive `grep -c "bounded termination deadline"`
returns **1** on a GREEN run too, because it matches the arm's own NAME, so the discriminating
pattern is `not ok -`).
**FIFTH OBSERVATION, iteration 261 — and the CANCEL DURATION REPRODUCES TO WITHIN 3 SECONDS, which
is evidence for a ceiling rather than for flakiness.** PR `#843`, again a **docs-only** diff (6
`design_docs` files + 1 sprint JSON; control: `design_docs` hits **6**, `tools/launchd` hits **0**).
The job ran `20:40:24Z` → `20:55:41Z` = **15m17s** and concluded **`cancelled`**, against iteration
259's merge-attempt-1 cancel at **15m20s**. All four REQUIRED contexts passed on the same SHA
(`build`, `docs-gate`, `lint`, `test`), so the PR was `MERGEABLE`/**`UNSTABLE`**, not `BLOCKED`, and
this did not gate the landing — recorded rather than triaged as a red, per Gate 1's non-required
rule. **Two cancels three seconds apart is not the profile of a race**; it is the profile of
something hitting a fixed limit, so the scoping below should first establish whether the arm is
being killed by a job/step timeout (in which case the wall-clock bound is a symptom and the real
finding is that the suite got slower) before deriving the bound from the stimulus.

**This is rule 3m's shape exactly**: a wall-clock bound calibrated on the machine it was written on,
where both the bound and the stimulus scale with the host — the arm passes on this laptop and is
unstable on a GitHub macOS runner. **Scope it as deriving the bound from the stimulus measured
in-test** (the fix World applied to the same class: `readTimeout := hold / 20`, making the ratio
true by construction on any machine), plus an absolute floor on the *stimulus* so a degenerate
runner reports instrument failure loudly instead of passing quietly — never a bigger absolute
constant. **Not urgent**: the job is NOT in the required-contexts set (`build`, `docs-gate`, `lint`,
`test`), so it produces `UNSTABLE`, never `BLOCKED`. **It is worth fixing anyway**, because an
amber that everyone learns to re-run is how a required one eventually gets waved through — and
because every iteration that trips it currently pays the same diagnosis twice (this one did).

**[NEXT] m-ui-dependency-tree-unbuildable — `ui/` has not installed since 2026-07-10, and the check
that would have said so is path-filtered and non-required** ([`#503`](https://github.com/sunholo-data/ailang/issues/503),
whose title names the MECHANISM; this row is the measured consequence, filed as a comment there rather
than as a duplicate issue). Found at iter-233 only because that iteration's diff touched
`.github/workflows/dashboard-ui-build.yml`, one of the workflow's own path triggers, so it ran.

> **Last green `2026-07-10T12:22:41Z`; every run since is `failure` — 10 of the last 10, forty days.**
> Zero charter mentions before this row (`grep -c dashboard-ui-build` = **0**; control
> `m-ci-no-job-timeouts` = **4**). The failure is **`RUN npm ci` in `docker/Dockerfile.dashboard`'s
> `ui-builder` stage**, so it is not workflow config — the image cannot be built. Reproduced locally as a
> differential (`npm install --package-lock-only --dry-run`), arm A failing exactly as CI does.
> **THE OBVIOUS FIX IS ALREADY REFUTED**, which is why this is a row and not a patch: pinning
> `@vitejs/plugin-react` back to `^4.7.0` still fails and surfaces a SECOND conflict beneath it. Three
> stacked causes: (1) `@vitejs/plugin-react@^6.0.5` peers on `vite@^8` against a pinned `vite@^5.0.0`,
> from `b6dc8c76c` (#566, `2026-08-03 13:04:10Z` — the next run, `13:06:45Z`, is red and so is every run
> after); (2) `eslint@^10.8.1` vs `eslint-plugin-react@^7.37.5`, which peers on `eslint <= ^9.7`, from
> `01d1e6eed` (#371, `2026-07-14`); (3) a THIRD, unidentified, since the first red (`2026-07-11`)
> predates both. **Severity is above a non-required PR check**: `cloudbuild-release.yaml:105` and
> `cloudbuild-dev.yaml:68` build and `--push` the same Dockerfile and the same stage — identical
> mechanism, though no Cloud Build history was consulted, which is a limit rather than a finding.
> The work is a real bump-vs-hold decision across vite/eslint plus identifying cause 3, and the UI build
> must actually be exercised afterwards. The dependabot grouping that split `vite` from its own plugin is
> the systemic half; repairing the tree is the immediate half.

**[NEXT — BLOCKED ON EXTERNAL DATA, predicate re-read at Gate 1 each iteration, NOT on a date]
m-wasm-deterministic-typecheck-budget — gate the WASM type-check on the deterministic STEP count
and retire the wall-clock limit** (`ailang#662` ask 2, the reporter's own preferred fix; accepted at
iter-225 and deliberately NOT bundled into that iteration's configurability work). Iteration 225
shipped asks 1/3/4 and, critically, the *instrument*: `typeCheckSteps` counts instrumented
type-checker entries and is hardware-independent — identical source, identical count, on every
machine — reported by `ailangLoadModule` on every outcome. **What is missing is the ceiling, and it
is not ours to guess**: choosing one from our own probing is precisely how the 2 s wall-clock number
was picked, and `ailang#662` is the report that it was wrong. The blocking input is a step
distribution over a real corpus — legitimate modules near the top, pathological ones (citizen.ail
shape) above it — and the strongest one available is the reporter's own 25-module, ~367 KB corpus
with the Playwright + CDP-throttling harness they already have and offered to share. Asked for in
the `#662` verdict comment (2026-08-19). **Predicate to re-read at Gate 1**: has `#662` gained a
comment carrying per-module `typeCheckSteps`? — `gh issue view 662 --json comments`, run as a
command with its control, never inferred from the row's own text. When it flips, this row is the
pick regardless of position. **Design note for whoever takes it:** a step ceiling and a wall-clock
limit are not substitutes — steps bound *work*, the clock bounds *user-visible freeze*, and the
guard's original purpose was the second. The likely end state is a step gate as the primary,
reproducible limit with the clock retained as a configurable backstop, which is a design question
and therefore NEW-DOC with quorum at creation.

**[NEXT] m-xml-unresolvable-prefix-dropped — a namespace prefix that is neither declared nor
`xml:` is silently dropped, and the streaming scanners resolve the MATCH name against a `nil`
prefix map** (both found at iter-226 while fixing `#646`; pre-existing, deliberately NOT bundled).
Iteration 226 fixed the `xml:` half, because that prefix is bound by definition and can never be
declared, so it was unresolvable *by spec* and made the `xml:space` signal invisible. Two neighbours
survive, and both are one-command reproducible:
(a) `resolveTagName` returns the bare local name whenever `lookupPrefix` finds nothing, so parsing a
DOCX FRAGMENT — `<w:p>…</w:p>` extracted from a part, with no `xmlns:w` in scope — silently yields
`p`, `t`, `val` for every prefixed name. Measured at iter-226: `<r><t w:val="x"/></r>` →
`getAttr(t,"w:val")` **None**, `getAttr(t,"val")` **Some("x")**; control in the same call, a
DECLARED `xmlns:w` resolves `w:val` correctly. Fragment parsing is exactly what an OOXML consumer
does, so this is a live exposure, not a latent one.
(b) `scanForElements` (`internal/builtins/xml.go`) and both fold scanners (`xml_fold.go`) call
`resolveTagName(t.Name, nil)` for the MATCH test, so in a document declaring `xmlns:w` on its root
`parseElements(doc, "w:t")` finds nothing while `parseElements(doc, "t")` works. Self-consistent and
undocumented; the matched subtree is then built with a local prefix map, so the RETURNED tags may
carry a prefix the caller could not search for.
The design question is what to do with an unresolvable prefix — preserve the raw prefix as written
(lenient, matches `parseLenient`'s contract), drop it as today (namespace-correct, lossy), or expose
both spellings — plus whether the streaming scanners should thread an ancestor prefix map at all,
which costs the allocation-free scan they exist for. So this is NEW-DOC with quorum at creation, not
a controller-inline fix. Demand evidence: the `#646` reporter is doing OOXML extraction today and
was told about both in the verdict comment, with an invitation to say if (a) bites them.

**[NEXT] m-verify-unencodable-reported-as-error — "cannot encode this shape" is reported as
ERROR where the comparable case is reported as `skipped`** (`ailang#689` claim 1, accepted at
iter-224 and deliberately NOT bundled into that iteration's encoding fix). Iteration 224 fixed the
underlying sort defect, which made the reporter's own contract verify, so reclassifying it would
have hidden a real bug behind an honest-looking hint. The complaint itself is still valid for the
RESIDUAL surface, and it reproduces **first-party on a SHIPPED example**, independently of `#689`:
`examples/runnable/contracts/shapes_verify.ail` reports `4 functions: 3 verified, 1 errors`, whose
ERROR is `encoding error: cannot encode function body: match pattern: unsupported pattern type
*core.TuplePattern in SMT encoding` — while `unencodable_callee_skip.ail` reports `1 skipped` with a
structured `UNENCODABLE_TYPE` reason for the same class of "the tool declined to try". So the repo's
own example corpus carries both treatments of one concept. The user-visible cost is the one the
reporter named: in a 38-module `--prove` run the ERROR line is the one that reads like a contract
FAILED. Scope this as a CLASSIFICATION question, not a per-case patch — enumerate every site that
can emit an encoding refusal and ask which are limitations (⇒ `skipped` + hint, with the existing
`UNENCODABLE_TYPE` machinery) versus genuine errors, since the `verify --strict` exit code and the
`--json` `status` field both key off it. **Ghost-discipline the enumeration**: the sites are already
partly structured, so the work may be smaller than it looks, and `shapes_verify.ail` doubles as the
regression fixture (its expected verdict changes, which is itself the acceptance test).

**[NEXT] m-string-search-offset — `find` has no offset, so "find the next X after i" copies the
tail on every search** (`ailang#688` claim 2, accepted at iter-223 and deliberately NOT bundled into
that iteration's perf/correctness fix). The workaround the reporter had to write is
`find(substring(s, i, length(s)), needle) + i`, which is quadratic for exactly the reason given, and
it is still true at HEAD. Shape of the ask: `find(s, needle, from)`, plus `lastIndexOf(s, needle)`
and `indexOfAny(s, [needles])` (`std/string` already has `splitAny`, so `indexOfAny` is its natural
companion). This is an API addition — signatures, three codegen specs, SMT mapping in
`internal/smt/types.go`, docs, examples — so it needs a design doc (NEW-DOC, quorum at creation),
not a controller-inline fix. Demand evidence is satisfied by construction: a real downstream
consumer is blocked today.

**[NEXT] m-codegen-helper-imports-inert — `GoCodegenSpec.Imports` is silently ignored for EVERY
Helper spec** (found at iter-223 while fixing `#688`; a latent trap, not a live break). `runtime.go`'s
import block is a closed allowlist in `codegen.go` — `fmt`, `reflect`, and `sort`/`strconv`/`strings`/
`math` selected by **substring-scanning the emitted helper code** — while `codegen_registry.go`
explicitly skips `spec.Imports` for Helper specs ("Don't track imports here"). So a helper needing any
other import emits Go that does not compile, and the spec field that looks like the way to declare it
does nothing. Measured: a body using `unicode/utf8` emitted `undefined: utf8`; existing specs such as
`_stringToInt` appear to work only because `strings`/`strconv` happen to be in the allowlist and their
bodies contain the matching substring. Live exposure today is zero (no shipped Helper needs an
outside import), which is why this is a row and not a fix — the durable options are to honour
`spec.Imports` for Helper specs or to reject at registration time, and choosing needs a design pass.
`internal/builtins/string_char_index_test.go` currently pins the constraint for `charAt`/`charCode`
only.

**[NEXT] [world-DEMAND] m-serveapi-protocol-only-module — `serveapi` is an API seam but not a
DEPENDENCY seam** (`ailang#764`, filed by `mission-world` iteration 90; P2, NOT a v1.0 bar item —
filed here for discoverability beside the other cross-mission rows). World's charter carries a
committed zero-cloud gate (`TestDaemonDependencyAllowlist`, 11 module roots, graph 239 pkgs / 46
non-stdlib); importing `serveapi` would add hundreds of disallowed packages, so its item 5 is
blocked on us rather than on a human decision. Mark's 2026-08-17 ruling to World pre-authorized
exactly this route (ask upstream for a protocol-only module, never a broad relaxation).

**Ghost-disciplined first-party at HEAD `0524a0fc5` before entering the queue** — a sibling
mission's request never earns a row on the strength of the request (Gate 0 cross-mission contract);
it earns one on measurement, and this one CONFIRMS: `go list` shows `serveapi`'s only non-stdlib
import is `github.com/sunholo-data/ailang/internal/apiserver`, and `go list -deps ./serveapi`
closes over **486** non-stdlib packages by our filter (World measured **479** with theirs — same
order of magnitude, different pattern, and the delta is a fact about the two filters). Two of
World's three controls reproduced exactly (`cmd/wasm` **12**, `cmd/astdump` **14**); the third did
**not** (`cmd/registry-validator` **467** here vs World's **6**) and is recorded rather than
smoothed over — the load-bearing claim does not rest on it.

Shape of the ask: a protocol-only module (request/response types + the wire contract, no server
implementation) that a caller can import without linking `internal/apiserver`. Note this SATISFIES
the demand-evidence gate by construction — a real downstream consumer blocked today is the
strongest demand signal there is. Sequence AFTER the sweep-orphan lane; it needs a design doc
(NEW-DOC, quorum at creation), not a controller-inline fix.

**Not gating** (the ~30 non-gating docs (eval-infra rig/harness, cloud-infra, motoko-fork, post-v1)): ship on the normal v0.2x road or post-v1 per the
clause rule. `planned/v1_0_0/` now contains ONLY gating docs (17 non-gating docs re-bucketed to
v1_1_0 on 2026-07-11); v0_29_0 docs that appear above gate v1 via the queue, not the folder.

**Post-v1**: everything in `planned/v1_1_0/`.

## Ruled out / resolved

- **Sonnet as default executor** — ruled out 2026-07-10 (Mark: corrections needed; false economy).
  Re-entry only via the evidence rule.
- **Scheduling via cron / scheduled-tasks MCP** — ruled out; this rig's substrate is launchd
  (nightly-eval + os-rotation-filler precedents), and the coordinator has no internal timer.

## Done / superseded

*(nothing yet — mission initialized 2026-07-10)*
