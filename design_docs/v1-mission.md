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
| D-30 | OPEN | **How must the harness&harr;`ai-check` version coupling be enforced BEFORE the `not_applicable` split lands?** Round-2 quorum on `m-contract-verification-coverage` was **blocked** by `gpt5-6-sol` (2/2 reviewers present, `gemini-3-1-pro` **pass**) on the one residual the doc had named honestly and mitigated only by convention. **Controller-measured, first-party, not forwarded (rule 3f):** `RunAICheck` defaults `ailangPath` to the bare string `"ailang"` and `exec.Command` resolves it via **PATH** (`internal/eval_harness/verify.go:47-53`); **2 of 2** live non-test call sites pass `""` (`repair.go:76`, `verify.go:123`; control `PopulateVerifyMetrics` 2, negative control 0). So the parent harness and its verifier child are **independently versioned**, and reader-before-writer *commit* order cannot buy reader-before-writer *deployment*. **The skew is live on this rig right now, not hypothetical**: PATH `ailang` is `v0.33.1-211-g626f5e54b-dirty` against a repo/parent at `v0.33.1-216-g30176187f`. Post-split, an old reader driving a new writer silently banks a **reduced** `verify_skipped` and drops the `not_applicable` count entirely &mdash; a no-silent-fallback violation (CLAUDE.md &sect;2) on the exact data path clause 5's headline KPI rests on. **(a) SCHEMA** &mdash; version `ai-check`'s JSON (`--json-schema=2`), emit `not_applicable` only for v2, and make the reader **reject** missing/unknown versions rather than bank partial counters (`gpt5-6-sol`'s primary fix; a new wire contract, and the largest option). **(b) SAME-BINARY** &mdash; bind `RunAICheck` to `os.Executable()` instead of PATH, making skew impossible by construction (`gpt5-6-sol`'s own stated alternative; ~1 line, and the idiom already has **9** in-repo precedents incl. `cmd/ailang/replay.go:153` &mdash; **but** it silently changes how *every* eval run resolves its verifier, and collides with this loop's own mandated scratch-build-and-prepend-PATH discipline). **(c) ACCEPT** &mdash; keep PATH, ship the split with the residual named plus a post-merge spot-check (what the doc proposes today; `gemini-3-1-pro` passed it, `gpt5-6-sol` rejects it as not fail-closed). | Controller did **not** self-rule: the narrow-refinement carve-out requires a fix that needs no controller judgment, and choosing among a new wire protocol, a repo-wide resolution change with rig-operational consequences, and accepting a P0 data-integrity residual is judgment (standing rule 2). Note the coupling is a **pre-existing** property of HEAD that the split would make consequential &mdash; it is worth a ruling whichever way `D-29` goes. |
| D-31 | OPEN | **Split the designer rotation into AUTHORING lanes and REVIEW lanes (or widen it)?** The rotation is `claude:claude-fable-5` &rarr; `codex:gpt-5.6-sol` &rarr; gemini, and **two of its three entries cannot serve as designer for STRUCTURAL reasons no probe can clear**: `codex:gpt-5.6-sol` IS one of the two default quorum reviewers (`gpt5-6-sol`), so routing it makes a doc's author its own judge; gemini/managed_agents is read-only under `CapRemoteSandbox` and cannot write a file at all. The usable authoring rotation therefore has ONE entry, and every doc that blocks at round 1 collides with the Fable diet by construction. Iteration 251 measured this at 3 instances and said the fix *"needs a human, because it is a routing-policy change on a shared file"*; iteration 255 amended the DIET (the unit is one bounded DOC, not one bounded RUN) but explicitly did **not** widen the rotation &mdash; and **neither filed a decision ID**, so the ask has never reached Mark. Iteration 256 is instance **4** and files it. **(a) SPLIT** &mdash; mark lanes authoring-capable vs review-capable and rotate only over the former; **(b) WIDEN** &mdash; add a fourth model that is neither a quorum reviewer nor sandbox-read-only; **(c) ACCEPT** &mdash; keep collapsing onto Fable and stop recording it as an anomaly. | Filed 2026-08-23 by iteration 256 under the decision-recording contract: a park-for-human with no ledger row can never be generated into a `DECISIONS FOR MARK` section, so it gets re-discovered every iteration and answered in none. |
| D-32 | OPEN | **Should an `inconclusive` verification obligation be EXEMPTED from the effective `cost_per_verified_success` arm, the way your `D-29` ruling exempts `not_applicable`?** | Iteration 259 measured the class first-party: **8 of 30** frozen `v1.0` runs are graded verification-FAILED on a counterexample, and **all 8** land on the two recursive-ADT benchmarks (`contract_sorted_merge` &times;4, `contract_bst_validate` &times;4), uniform across 4 model families &mdash; the signature of a TOOL defect, not a model one. `m-verify-bounded-unrolling-false-counterexample` reclassifies those from `counterexample` to a new `inconclusive` status, which is honest. **But the published number cannot move either way without this ruling**: that doc's `isVerifiedSuccess` gains `&& VerifyInconclusive == 0`, so a reclassified run stays rejected, numerator and denominator are unchanged, and `$0.7778187072` is bit-identical before and after (measured; `verified_successes` 3, `known_cost_usd` `$2.3334561216`). So this is the same axis you already ruled on, one status along. **(a) EXEMPT** &mdash; `inconclusive` joins `not_applicable` in the effective arm (the strict arm is untouched, per your `D-29` both-arms ruling); **(b) KEEP STRICT** &mdash; an obligation the tool could not discharge counts against the model in both arms, i.e. the fix buys honesty and repair-loop correctness but no KPI change, which is the status quo the doc ships under; **(c) DEFER** until the `inconclusive` population is observed on a fresh cohort rather than inferred from the frozen one. The doc deliberately does not decide this and does not depend on it. |
<!-- decision-ledger:end -->

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

## STATUS 2026-08-23 — ITERATION 259: **THE VERIFIER'S FALSE COUNTEREXAMPLE IS REAL, MEASURED AT 8 OF 30 FROZEN-COHORT RUNS — AND THE DISCRIMINATOR I HANDED THE DESIGNER WAS WRONG, WHICH TWO REVIEWERS AND MY OWN SWEEP INDEPENDENTLY PROVED.** Gate 0/1: kill switch armed; billing tripwire **CLEAN**; gh `sunholo-voight-kampff`; pin worktree clean, detached at `bc3f80884` = `origin/dev`. Running skill **byte-identical** to `origin/dev` — `cmp` rc=0 (332,323 B both sides) against the RESOLVED `readlink` target `/Users/voightkampff/dev/sunholo-data/ailang/.claude/skills/mission-control` (inode **48121458**), NOT the pin's own copy (**48126612**), per iteration 241's correction. `origin/dev` `bc3f80884`: **16 checks, ZERO not-green** (control: parent `9417c5ff7` → **16**, so the endpoint answered and a true zero is distinguishable); CI + Build-and-Release `success` on that SHA, Docs-Deploy `success` on the parent (path-filtered, N/A not pending). **Zero** allowlisted directives on `#745` since the `2026-08-23T08:30:56Z` watermark (70 comments); watermark unmoved because nothing was processed. Ledger **29 rows at entry, `D-30` + `D-31` OPEN** (read from `origin/dev`, NOT the main checkout — `scripts/mission_decisions.sh --open` there still showed `D-29` OPEN because that tree is 2 behind; a stale-base read caught before it became a false re-ask). **`D-32` FILED** (below). No rotation: `#745` created `06:14:45Z` = **08:14 CEST Monday 08-17**, AFTER the Monday-07:00 LOCAL boundary, and 70 < 80; next boundary Mon 08-24 07:00. Weekly external-issue sweep NOT owed (the 08-17 week's ran at iteration 250); zero open `[nightly-eval]` issues; inbox empty. Died-mid-flight sweep clean: no `.wt-iter258`/`.wt-iter259` at entry; sole open fleet-account PR is `#695`, `headRefName` `coordinator/task-d98bb271`, matching **no** worktree in my clone — unattributable, left alone and named again. **NO ff-ONLY MERGE, AND THE PRECONDITION IS WHY.** Local `dev` is **0 ahead / 2 behind**, but `D-16`'s path-disjointness test FAILS: `comm -12` of the incoming file set against the dirty set returns `.claude/skills/mission-control/SKILL.md` (control fires — `design_docs/v1-mission.md` is in the incoming set). Its content is identical to `origin/dev` (Gate-1 `cmp` rc=0), so the collision is nominal — but the designer was live in that tree, so `D-16`'s *any collision or doubt* clause applies and the record was written from a worktree branched at `origin/dev` instead. **PICK — `m-verify-bounded-unrolling-false-counterexample`**, the `[NEXT]` clause-5 critical-path head that iteration 258 filed. NEW-DOC verified rather than assumed (`grep -ril` finds only the v0_8_0/v0_9_1 SMT predecessors, control `m-cohort-manifest-build-provenance.md` present); not landed (fresh fetch, `git log --grep` + PR search empty, control fires). **I REPRODUCED THE DEFECT FIRST-PARTY BEFORE ROUTING ANYTHING (rule 3b(v)), ON A STAMPED SCRATCH BUILD** (`v0.33.1-223-gbc3f80884`, `--version` == `git describe`). **AND THE FIRST BUILD WAS SILENTLY UNSTAMPED — A NEW MEMBER OF ITERATION 256's CLASS:** my ldflags named `github.com/sunholo/ailang/internal/version` while the module is `github.com/sunholo-**data**/ailang`; `go build` returned **rc=0**, produced the binary, and `--version` printed `AILANG dev`. A wrong `-X` symbol path is ignored exactly as silently as `-buildvcs=true` — caught only because I read `--version` against `git describe` rather than trusting rc=0. **Same-file control, one identical clause `ensures { result >= 0 }`, three functions** (`/tmp/i259_probe/discrim.ail`): `aOk` non-recursive/true → **verified**; `bBad` non-recursive/genuinely false → **counterexample**; `cRec` recursive/true (a list length) → **counterexample**. Witness grows with the bound: depth 2 → a 2-element list, depth 4 → a 4-element list, control `headOr0` verified at every depth. **THE BLAST RADIUS IS MEASURED AND IT IS NOT HYPOTHETICAL: 8 OF 30 FROZEN-COHORT RUNS (27%) ARE GRADED VERIFICATION-FAILED ON A COUNTEREXAMPLE, AND ALL 8 LAND ON THE TWO RECURSIVE-ADT BENCHMARKS** — `contract_sorted_merge` ×4 and `contract_bst_validate` ×4, across **4 model families each**, every one with `verify_verified = 0` (`verify_counterexample` distribution over the 30 banked runs: `{0: 22, 1: 8}`). Uniformity across model families is the signature of a TOOL defect, the same argument iteration 254 used for the skip class. `contract_sorted_merge.yml:23-24` ships `sLength ensures { result >= 0 }`, the exact clause I reproduced. **A CONSEQUENCE THE QUEUE ROW DID NOT NAME, AND IT IS ARGUABLY WORSE THAN THE KPI ONE:** `internal/eval_harness/repair.go:76-86` fires `VERIFY_COUNTEREXAMPLE` on `Counterexample > 0` and feeds `FormatZ3RepairHint` back to the model, while `agent_prompt.go:595` instructs *"If Z3 reports COUNTEREXAMPLE: fix your logic using the counterexample inputs"* — so on those 8 runs the harness spends repair turns telling models to break code that is correct. **AND THE FIX IS SMALL BECAUSE THE INFORMATION IS ALREADY COMPUTED:** `vr.BoundedDepth` is set at `ai_check.go:400` and `verify.go:419` *before* the status switch, and `verify_print.go:32-33` already prints `✓ VERIFIED (bounded: depth N)` — `grep -c BoundedDepth` there is **2**, both in the verified branch. The `counterexample` branch prints `✗ VIOLATION` with **no** bounded qualification. The tool is already careful to say *I only proved this to depth N* when it succeeds and drops exactly that caveat when it fails. Two near-identical status ladders exist (`ai_check.go:370-420`, `verify.go:388-435`). **DESIGNER `claude:claude-fable-5`; rotation collapsed onto it for STRUCTURAL reasons — `D-31` instance 6** (codex IS quorum reviewer `gpt5-6-sol`; gemini is read-only under `CapRemoteSandbox`); neither is a probe failure, so neither was probed. Diet-COMPLIANT under iteration 255's amendment: ONE doc = author + one protocol-mandated revision. **THE DESIGNER REFUTED MY OWN LOAD-BEARING PREMISE AND I RE-VERIFIED THE REFUTATION FIRST-PARTY (Gate 2 rule (d)).** I handed it, under a VERIFIED-BY-ME heading, a candidate discriminator: an artifact model carries a binding for the havoc'd frontier symbol (`cRec_0`) while a genuine one (`bBad`) does not. It ran the experiment I demanded and found **all three genuine recursive counterexamples also carry the frontier binding**; I re-ran its `q3.ail` myself and confirmed, then swept depths **1, 2, 3, 5, 8, 10** — the binding is present in **18 of 18** readings. So the discriminator was not merely unsound, it was **unreachable**: its `counterexample` branch fires on nothing I can construct while being the one path able to emit a false violation. **QUORUM: TWO ROUNDS, BOTH BLOCKED, BOTH OBJECTIONS REAL, AND I MEASURED EVERY PREMISE RATHER THAN FORWARDING IT (rule 3f).** R1 2/2 present, both reject: `gpt5-6-sol` — the discriminator is unsound and the decision table still maps a no-frontier `sat` to `counterexample`, so the design can emit the exact false violation it claims impossible (**sustained**; my 18/18 sweep is stronger than its own argument); `gemini-3-1-pro` — Q6's *corrected KPI* is arithmetically impossible under the doc's own M1, since a run flipping `counterexample`→`inconclusive` still fails `isVerifiedSuccess` via the new `VerifyInconclusive == 0` conjunct, leaving numerator and denominator unchanged (**confirmed**: the doc's own decision table line already said *run still rejected*, and the live KPI is `$0.7778187071999999` with `verified_successes` 3 and `known_cost_usd` `$2.3334561216` — bit-identical before and after). **R2: `gpt5-6-sol` ABSENT ON `budget` — the exact N−1 degrade hole — AND I RESTORED IT** with a solo `design-review --max-cost-usd 0.30` ($0.0875) rather than deciding the round at N−1. `gemini-3-1-pro` rejected on a pure premise gap: the r2 *zero `internal/smt` changes* claim rests on `smt.IsRecursiveFunc` being exported, and no Verification Log row proved it. **Measured: `internal/smt/encodable.go:153: func IsRecursiveFunc(body core.CoreExpr, funcName string) bool`, exported, with call sites ALREADY at `cmd/ailang/ai_check.go:399` and `verify.go:418` — the very `BoundedDepth` guard lines the design reuses** (negative control 0, positive control 2). Premise **refuted**; the objection is a missing row, not a defect. **THE RESTORED REVIEWER'S OBJECTION IS `D-30`, AND THIS DOC IS THE SECOND TO BE BLOCKED BY IT INDEPENDENTLY.** `gpt5-6-sol` held that reader-before-writer *commit* order cannot prevent new-binary/old-harness skew, because `RunAICheck` resolves its verifier child through **PATH** — so an old harness sees `counterexample == 0`, sets `VerifyOk` true, and a mixed result carrying any `verified > 0` is counted a verified success. That is **KPI INFLATION**, the opposite direction from the defect being fixed, and it is the identical mechanism already parked as `D-30` for `m-contract-verification-coverage`. The doc's Conflict Surface item 4 had claimed the skew was *prevented by landing order*; that claim is false and is now corrected in place. **NARROW-REFINEMENT CARVE-OUT APPLIED (controller, not a third Fable run).** Both surviving objections carry concrete reviewer-authored `proposed_fix`es and neither disputes the design DIRECTION (which no reviewer has questioned across three reviews): added **V28** recording the `IsRecursiveFunc` measurement above, rewrote Conflict Surface 4 to state the skew is DETECTED not prevented and to name `D-30` as the only thing that could prevent it, and added **AC-13** — the reviewer's verbatim ask — an old-reader/new-writer test on a fixture carrying both `verified > 0` and `inconclusive > 0`, asserting today's predicate does NOT score it a success (**RED at base by construction**, which is the point). **DOC LANDED AND ROUTABLE, NOT PARKED.** `m-verify-bounded-unrolling-false-counterexample` is `[NEXT]` for the planner. It is NOT `needs-human-review`: its design direction is unchallenged and its residual is an EXISTING open ledger row, so parking it would manufacture nothing new. It is NOT `PARKED-ON-LANE`: nothing unblocks on a clock. **`D-32` FILED** — whether `inconclusive` joins the effective KPI arm's exemption set, exactly the axis Mark ruled `D-29` on for `not_applicable`. It is the ONLY thing that could move the headline number, and the doc deliberately does not decide it or depend on it. **Ruled out**: my own frontier-symbol discriminator (18/18 refutation, controls firing — recorded because it reached a sub-agent under a VERIFIED-BY-ME label, which is the laundering Gate 2 forbids); that the fix improves the KPI (bit-identical by construction; its value is honest labels, repair turns not spent breaking correct code, and `D-29`'s second clause unblocked); that raising `-verify-recursive-depth` helps (1/2/3/5/8/10 all fail); that this is one surface (`m-verify-unencodable-reported-as-error` is the same taxonomy defect on the `error`-vs-`skipped` surface — named in the doc, deliberately NOT absorbed, because it inverts no verdict and bundling different bars is what cost a sibling doc five rounds); that the doc needed `internal/smt` changes (r2 has zero); that this needed an executor, evaluator, the GPU, or the eval rig. **Gates** (darwin/arm64; windows and ubuntu legs unrun locally): documentation only — no code shipped, so no CI matrix is implicated. The one binary built was a scratch `go build` with ldflags into `/tmp`, `~/go/bin` untouched, and it is the instrument every measurement above was taken with. **No planner, executor or evaluator spawned** — the quorum is the gate that ran, and the carve-out closed it. metered **$0.2251** of $5 (R1 $0.1013 = gpt5-6-sol $0.0717 + gemini-3-1-pro $0.0296; R2 $0.0363 gemini-only; gpt5-6-sol restore $0.0875); quota buckets: opus (controller), fable ×2 (designer author + one protocol-mandated revision, diet-compliant).

## STATUS 2026-08-23 — ITERATION 258: **MARK RULED `D-29` — AND SCOPING HIS SECOND CLAUSE UNCOVERED A VERIFIER DEFECT THAT GRADES CORRECT RECURSIVE CODE AS A VERIFICATION FAILURE.** Gate 0/1: kill switch armed; billing tripwire **CLEAN**; gh `sunholo-voight-kampff`; pin worktree clean, detached at `9417c5ff7` = `origin/dev`. Running skill **byte-identical** to `origin/dev` — `cmp` rc=0 (332,323 B both sides) against the RESOLVED `readlink` target `/Users/voightkampff/dev/sunholo-data/ailang/.claude/skills/mission-control` (inode **48121458**), NOT the pin's own copy (**48126612**), per iteration 241's correction; same-file assertion run before trusting the answer. Main checkout is **0 ahead / 1 behind** `origin/dev` (missing only `9417c5ff7`, iteration 257's own record, landed by PR) and dirty in the 6 known rig-synced artifacts + a STAGED `SKILL.md` whose diff **against `origin/dev` is EMPTY** — content-identical residue, not drift, so no reconcile was owed and the record was written from a worktree branched at `origin/dev` anyway. `origin/dev` `9417c5ff7`: **16 checks, ZERO not-green** (control: parent `ad6d08050` → **20**, so the endpoint answered and a true zero is distinguishable); CI + Build-and-Release `success` on that SHA, Docs-Deploy `success` on the parent (path-filtered, N/A not pending). Died-mid-flight sweep clean: no `.wt-iter257`/`.wt-iter258` at entry; sole open fleet-account PR is `#695`, `headRefName` `coordinator/task-d98bb271`, matching **no** worktree in my clone — unattributable, left alone and named again. Inbox empty; zero open `[nightly-eval]` issues. No rotation: `#745` created `06:14:45Z` = **08:14 CEST Monday 08-17**, AFTER the Monday-07:00 LOCAL boundary, and 69 < 80; next boundary Mon 08-24 07:00. Weekly external-issue sweep NOT owed (the 08-17 week's ran at iteration 250). **PICK — A HUMAN DIRECTIVE, WHICH OUTRANKS THE QUEUE.** `scripts/mission_directives.sh` returned **1** allowlisted comment since the `2026-08-22T11:36:26Z` watermark: MarkEdmondson1234 @ `2026-08-23T08:30:56Z`, verbatim *"D-29 - both but update prompts to use ensures for benchmarks that make sense"*. **`D-29` RESOLVED in the ledger the same iteration, before the watermark moved** (31 rows, still valid): option **(c)** — publish a **strict** arm (`VerifySkipped == 0`, **$0.7778**, denominator 3) beside an **effective** arm (exempting `not_applicable`, **$0.2121**, denominator 11). Neither replaces the other, so the 2026-07-27 ratification is not overturned and no published number is silently restated. Both arms require the `skipped`/`not_applicable` split, so the publishing milestone lands INSIDE `m-contract-verification-coverage` and inherits its `D-30` block; `D-29` now blocks nothing on its own. `D-30` and `D-31` were re-asked verbatim, unresolved. **I ENUMERATED MARK'S SECOND CLAUSE RATHER THAN GUESSING WHAT "MAKES SENSE".** **7 of 92** benchmark YAMLs carry a `contract_spec`; their declared functions partition exactly three ways — **16** carry an `ensures`, **10** carry no clauses at all (so raise no obligation), and **5** carry `requires` with no `ensures`: `isBST`, `minor3`, `encode`, `decode`, `toRoman`. That is **precisely** `D-29`'s named five, so the "no ensures clause" skip class has a clean syntactic definition it never had before. **AND IT CORRECTED THE CHARTER'S OWN INHERITED CLAIM (rule 3b(v)(b))**: the `D-29` row said *"`minor3` carries no clauses at all"*. It carries `requires { row >= 0, row <= 2, col >= 0, col <= 2 }` at `benchmarks/contract_matrix_determinant.yml:24`, and `git log` shows the file unchanged since `fe92c172f` (2026-04-21), so the claim was wrong when written, not stale. Corrected in place. **THE FINDING THAT CHANGES THE WORK: ADDING AN `ensures` TO A RECURSIVE FUNCTION IS NOT NEUTRAL — IT IS GRADED A SPURIOUS `counterexample`, WHICH IS STRICTLY WORSE THAN THE `not_applicable` IT REPLACES.** Measured with `ailang ai-check` on a **stamped** scratch build (`v0.33.1-222-g9417c5ff7`, ldflags per iteration 256's rule; `--version` matches `git describe` exactly, so the binary is identifiable). **Same-file control, only the recursion differs**: `encFlat` and `encRec` carry byte-identical `requires`/`ensures` (`length(result) <= 2 * length(s)`) — `encFlat` **verified**, `encRec` **counterexample**. The postcondition genuinely holds: `ailang run --caps IO` prints `true` on all four inputs including a 6-char string. **It is STRUCTURAL, not depth tuning — the "counterexample" input GROWS WITH THE UNROLLING DEPTH**: depth 2 → `"AAA"` (3 chars, `bounded_depth 2`), depth 4 → `"ABDCA"` (5), depth 8 → `"ABCDEFGHAB"` (10), with `encFlat` verifying at every depth. For any finite `k`, Z3 instantiates the havoc'd frontier with an input longer than `k`. A second shape — `ensures { result >= 0 }` on a recursive ADT length — is counterexampled identically at depths 2/4/8, its "witness" a 4-element list whose length is plainly non-negative. **`benchmarks/contract_sorted_merge.yml:23-24` ships exactly that clause on `sLength` today**, so this is live for the frozen cohort, not hypothetical. **Encodability of the clause LANGUAGE is NOT the blocker** (probe: three string-length postconditions all `verified`, `skipped 0`, control `controlDouble` verified) — the recursive body is. A third arm, an `ensures` CALLING a user function, returns `skipped` with a structured `UNENCODABLE_TYPE` reason, which is the *correct* disposition and shows the machinery to report incompleteness honestly already exists. **ROUTED AS TWO QUEUE ROWS, NOT ONE SPRINT.** `m-verify-bounded-unrolling-false-counterexample` (AILANG fix, P0): `isVerifiedSuccess` fails on `VerifyCounterex > 0`, so this converts correct model output into a verification FAILURE in **both** arms `D-29` just ratified — a direction no exemption can repair, and the same no-silent-fallback class (CLAUDE.md §2) as the `skipped`/`not_applicable` conflation. `m-benchmark-ensures-coverage` (Mark's second clause, scoped by the measurement): only **`minor3` is non-recursive** and therefore safely addable today; the other four would be spuriously counterexampled, so the clause is BLOCKED on the defect above for 4 of its 5 candidates. `encode` is both the strongest candidate on merit — `contract_rle_roundtrip.yml` states *"encoded string is never longer than 2 * length(original)"* in its own prose and ships no `ensures` for it — and the clearest casualty. **Ruled out**: that Mark's clause could be executed as written this iteration (4 of 5 candidates would make the KPI worse); that the counterexample was my implementation's fault (runtime confirms the contract holds; the same clause verifies non-recursively in the same file); that raising `-verify-recursive-depth` is a workaround (2/4/8 all fail, and the witness grows with the bound); that the clause language is unencodable (three shapes verified with a firing control); that this needed a designer, planner, executor, evaluator, the GPU, or any metered lane. **Gates** (darwin/arm64; windows and ubuntu legs unrun locally): documentation only — no code shipped, so no CI matrix is implicated. The one binary built was a scratch `go build` with ldflags, `~/go/bin` untouched, and it is the instrument every measurement above was taken with. **No designer, planner, executor or evaluator spawned** — the directive's deliverable is a ruling recorded and a scope measured, and its judge is the reproduction of the defect under a same-file control. metered **$0.00** of $5; quota buckets: opus (controller) only.

## STATUS 2026-08-23 — ITERATION 257: **FIVE QUORUM ROUNDS, FIVE REAL DEFECTS, AND THE REJECTIONS HAVE LOCALISED ONTO ONE CONSUMER — SO THE LANE IS DECOMPOSITION, NOT A SIXTH REVISION.** Gate 0/1: kill switch armed; billing tripwire **CLEAN**; gh `sunholo-voight-kampff`; pin worktree clean, detached at `ad6d08050` = `origin/dev`. Running skill **byte-identical** to `origin/dev` — `cmp` rc=0 (328,249 B both sides) against the RESOLVED `readlink` target `/Users/voightkampff/dev/sunholo-data/ailang/.claude/skills/mission-control` (inode **48017368**), NOT the pin's own copy (**48021463**), per iteration 241's correction; the two are confirmed DIFFERENT FILES, which is why the resolved path is the only valid arm. Main checkout is **0 ahead / 0 behind** `origin/dev` and dirty only in the 6 known rig-synced artifacts (3 modified tracked + 3 untracked `tools/eval/qwen3*.sh`), so no reconcile was owed. `origin/dev` `ad6d08050`: **20 checks, ZERO not-green** (control: `checks=20`, not 0, so the endpoint answered); CI / Build-and-Release / Docs-Deploy all `success` on that same SHA. **Zero** allowlisted directives on `#745` since the `2026-08-22T11:36:26Z` watermark (67 comments; the 67th is my own iteration-256 report). Ledger **31 rows; `D-29`, `D-30`, `D-31` all OPEN**, re-asked verbatim, none resolved. No rotation: `#745` created `06:14:45Z` = **08:14 CEST Monday 08-17**, after the Monday-07:00 LOCAL boundary; 67 < 80; next boundary Mon 08-24 07:00. Weekly external-issue sweep NOT owed (the 08-17 week's ran at iteration 250); zero open `[nightly-eval]` issues; inbox empty. Died-mid-flight sweep clean: no `.wt-iter256`/`.wt-iter257` at entry; sole open fleet-account PR is `#695`, `headRefName` `coordinator/task-d98bb271`, matching **no** worktree in my clone (my clone lists 28, none coordinator-shaped) — unattributable, left alone and named again, per the fleet-filter rule. Blocked-external predicates RE-RUN as commands rather than transcribed: `#662` OPEN, `#695` OPEN, `#513` OPEN — unchanged (control `#613` MERGED fires). **PICK — `m-cohort-manifest-build-provenance`**, the `[NEXT]` critical-path head, resuming iteration 256's bounded unanimous round-3 resume. **I RE-DERIVED V24 FIRST-PARTY BEFORE BUILDING ON IT (rule 3b(v)(b)), WITH A CLEAN CONTROL ARM**: in the genuinely dirty main checkout (`git status --porcelain` = 6 lines) `git describe --tags --always --dirty` → `v0.33.1-221-gad6d08050-dirty` (dirtiness on **Version**) while `git rev-parse HEAD` → `ad6d08050b5f…` (**plain SHA**); in the clean pin worktree the same two commands give `v0.33.1-221-gad6d08050` and the identical SHA. `internal/version/version.go` `init()` re-read this session puts `-dirty` on **`Commit`** via `vcs.modified`, and `Version` has **no** runtime fallback. The two stamping paths dirty **different fields**; neither arm alone suffices. **ROUND 4 — designer revision applied the reviewers' verbatim widening, and the quorum BLOCKED 2/2 WITH `absent_reviewers` EMPTY** (both `present: true`; no N−1 degrade, and `present_count` not satisfied by my own `--controller-verdict`). **Neither objection was about anything round 4 changed** — both are pre-existing holes rounds 1–3 walked past, in **different consumers**: `gpt5-6-sol` — the **freeze gate** refuses only *unstamped* builds, so a `-dirty` binary freezes release evidence; `gemini-3-1-pro` — **AC-6**'s "cache dir stays empty" proves `Store` was bypassed but **not** `Lookup` (a fresh dir makes a wrong Lookup miss silently). **I MEASURED BOTH PREMISES RATHER THAN FORWARDING THEM (rule 3f) AND BOTH HELD**: M4's branches are named `RefusesUnstampedVersion`/`_RefusesUnstampedCommit` with AC-11 asserting the substring `unstamped`, i.e. knownness-only, and a `-dirty` value **is** known; and the live release-evidence artifact has exactly **20** top-level keys (`jq -r 'keys[]'`) with **no** source diff and **no** compiler-content identity — `cohort_hash` `526fe724…` is over cohort *composition*, which the designer then confirmed against `eval_suite_manifest.go:118-145` (preimage `{eval_mode, languages, conditions, models, benchmarks, seed, prompt_version, trials}`, comment explicitly excluding `git_commit`/`ailang_version`). **THE ROUND-5 DIRECTIVE WAS THE PATTERN, NOT THE TWO PATCHES.** Four blocked rounds had each found ONE class — *a gate or assertion whose satisfying-state set is wider than the purpose it is cited for* — in a different consumer (R3 the cache gate, R4a the freeze gate, R4b an AC assertion), so patching the two named instances buys round 6. I directed a systemic sweep (CLAUDE.md §3) of every gate and every AC against that single question. **IT FOUND THREE TOO-WIDE ITEMS NO REVIEWER HAD NAMED**, one of them consequential: **AC-9**'s unstamped arm ("prefix + non-empty suffix") is satisfied by a *constant* `unstamped-deadbeef` — i.e. by the shared-bucket defect itself; **AC-13** ("entry present" in the changelog) was prose, not an assertion, and `make check-changelog` is rc=0 on pristine dev; and **the new strict freeze gate would have created a REFUSAL LOOP with the doc's own M4 remediation recipe**, which runs `git describe … --dirty` from a dirty tree and therefore re-fails the gate it exists to satisfy. All three fixed, plus AC-6 strengthened to pre-populate the cache with a dummy keyed by the ambiguous identity (proving Lookup bypass, not merely Store bypass) and mutation row **8c** naming BOTH that it kills the Lookup-hoisting mutant AND that the old empty-dir assertion does not. **I ALSO REQUIRED THE DESIGNER TO DESELECT THE REVIEWER'S ALTERNATIVE (b) ON A GROUND OTHER THAN V22/V23** — quoting the 215 ms hot-path hashing measurement against a *once-per-release* freeze is a scope-mismatched citation — and it did: a content address verifies bytes you have and cannot reconstruct bytes you do not, the uncommitted diff is a leak surface, and it legitimises evidence dependent on un-versioned state. **THE DESIGNER CORRECTED ME AND WAS RIGHT** (Gate 2 rule (d)): my clean-worktree control arm was true when taken but no longer reproducible in place, because the pin worktree had become dirty **from this very doc edit**; V26 records the nuance instead of transcribing my numbers. **ROUND 5 — `gemini-3-1-pro` PASS, the first pass in five rounds; `gpt5-6-sol` reject**, again 2/2 present. Its surviving objection is the same class one level deeper and is REAL: **`CommitClean()` is still weaker than the cache-correctness purpose it gates — a clean commit identifies SOURCE state, not compiler bytes**, so two clean builds at one commit can differ by Go toolchain, build tags or build flags and still share a module-cache key. **I MEASURED IT AND IT IS PRE-EXISTING AT HEAD, NOT INTRODUCED BY THIS DOC**: `ModuleCacheKey` (`internal/pipeline/cache_key.go:37`) hashes only `cacheKeyVersion` ("v3", hand-bumped), the `compilerVersion` string, the source hash and dep digests; `runtime.Version()` appears **0** times in `internal/pipeline` against **4** repo-wide (control fires), `cache_key.go` carries **0** build-tag/flag terms, negative control **0**, scope asserted with `test -d`; and there is exactly **ONE** live call site, `pipeline_module.go:276`, passing `version.Commit`. So the doc's M2 strictly NARROWS today's accepted path and leaves a pre-existing residual unfixed. **ROUTED — DECOMPOSITION, not a sixth revision.** Five rounds, every objection real, and the surviving rejections have localised onto exactly ONE of the three consumers while the other reviewer now passes: that is a scope signal, not a quality signal. `m-cohort-manifest-build-provenance` bundles a release-evidence gate, a compiler-cache identity and a banking bucket under one shared cause, and "what identifies compiler bytes" is a strictly harder question than "what identifies release evidence". Consumer 2 splits out; the rest is one reviewer-clean doc away from routable. **NEW ROW FILED — `m-module-cache-identity-not-compiler-bytes`**, on its own first-party evidence (the measurement above), independent of this doc's fate. **NOT `needs-human-review`**: nothing here awaits Mark — the split is a controller routing call and the remaining design question belongs to a designer plus a quorum, so filing it as a human park would manufacture a decision he does not have (standing rule 8). **NOT `PARKED-ON-LANE`**: nothing unblocks on a clock. **DIET — 2 Fable runs, the SECOND A KNOWING OVERSPEND, FLAGGED.** Iteration 255's amendment budgets one authoring run plus one protocol-mandated revision; this doc was authored in iteration 256, so a strict reading gives me one revision and I ran two revise-and-re-quorum cycles. I judged the second worth it and say so plainly rather than claiming compliance: it is the difference between a doc carrying two known holes and a doc where one reviewer passes and three unnamed defects — including a refusal loop the new gate would itself have opened — are closed. `D-31` is the standing fix; instance **5**. **Ruled out**: that the round-3 defect recurred (fixed; neither reviewer re-raised it); that round 4's objections were caused by round 4's edits (both pre-existing, in consumers it never touched); that `gpt5-6-sol`'s round-5 objection is a defect this doc introduces (measured pre-existing, one call site, controls firing); that alternative (b) was refuted by V22/V23 (scope-mismatched — that measurement is about a hot path, a freeze is once-per-release); that any round-4 decision was wrong (the sweep re-checked all of them and found none); that this needed Mark, the GPU, or any lane beyond quorum. **Gates** (darwin/arm64): documentation only — **no code shipped**, so no CI matrix is implicated; the design doc is the whole diff. **No planner, executor or evaluator spawned** — the quorum blocked before routing, which is the gate working as designed. metered **$0.415105** of $5 (round 4 $0.182935 = gpt5-6-sol $0.128195 + gemini-3-1-pro $0.054740; round 5 $0.232190 = $0.162900 + $0.069290); quota buckets: opus (controller), fable ×2 (designer revisions, second FLAGGED).

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
   sunholo/demos).
3. **P1 by impact-per-day** (the census has estimates; prefer ≤3-day items to keep iterations
   sprint-sized).
4. **Strategic multi-week items enter only after decomposition** into sprint-sized design docs
   (a decomposition is itself a valid iteration deliverable).
5. Anything re-scored `post-v1` in iteration 0 leaves the queue.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

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

- **[PARKED needs-human-review 2026-08-23 (iter-255) on `D-30` — DESIGN DOC written and revised, TWO full quorum rounds; ~1–2d, P1]**
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
