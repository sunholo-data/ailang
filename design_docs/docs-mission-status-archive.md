## STATUS 2026-09-07 — ITERATION 14: retry of `m-anthropic-sandbox`; Astra Agent lane timed out again, parked-on-lane [HARNESS]

Gate 0/1: armed; GitHub account `sunholo-voight-kampff`; canonical inbox triaged with no docs
directive or genuine regression. Fresh origin was checked; the observed base was
`aeeafc880dec8bb30215620332d938e96904aaf0` at `2026-09-06T22:41:03Z`; D-4 and D-5 remain OPEN.

Gate 2 re-selected `design_docs/planned/v0_29_0/m-anthropic-sandbox.md`, the retryable fresh
draw after iteration 13's failed designer attempt. Its existing pick-time quorum remains BLOCKED
3/3 on session-selective worker isolation, bounded termination/timeout evidence, and live API /
pricing verification. No new quorum spend was incurred.

Gate 3 spawned the required designer through the Agent tool as `gpt-6-astra` (resolver:
`recipe codex:gpt-6-astra`). After two bounded 120-second waits the design artifact was unchanged;
the Agent was explicitly shut down. No compatible Agent-tool fallback was authorized by the
resolved route, so no fallback designer was used. Planner and executor were not spawned because
no revised design reached re-quorum. Evaluator `pi:ollama/minimax-m3:cloud` was not spawned:
there was no generated implementation or valid plan to judge. This is a lane park, not a passing
verdict; generator-not-equal-judge remains intact.

Outcome: PARKED-ON-LANE. No implementation changes. D-4/D-5 remain the human decision asks.

Routing evidence: controller `codex:gpt-5.6-luna` (tok: not reported); designer Agent
`gpt-6-astra` (tok: not reported), error `running after 2 x 120s bounded waits, target unchanged,
then shutdown`; fallback not used because the resolver returned a recipe route with no compatible
Agent-tool fallback. Planner `codex:gpt-5.6-luna` (not spawned: no design-ready artifact); executor
`codex:gpt-5.6-luna` (not spawned: no plan); evaluator `pi:ollama/minimax-m3:cloud` (not spawned:
no generated implementation). Resume predicate: re-probe the designer lane next iteration; proceed
only if it produces a revised artifact, then re-quorum before planning. Gate 4 base=`aeeafc880dec8bb30215620332d938e96904aaf0`@`2026-09-06T22:41:03Z`.

**Progress**: goal unmoved; no sprint milestone executed.

**Retro — no skill edit.** This is the third consecutive docs-mission designer-lane failure for
fresh draws (iterations 12–14); it is surfaced as a routing-policy signal for human review. The
shared skill was not edited during this parked run.

## STATUS 2026-09-06 — ITERATION 13: fresh draw `m-anthropic-sandbox`; designer Agent-tool lane timed out, parked [HARNESS]

Gate 0/1: armed; GitHub account `sunholo-voight-kampff`; canonical inbox triaged with no docs
directive or genuine regression. Fresh origin was checked; the observed base was
`6c03639f518fa45569b879bfa73c2d31e5b3d62f` at `2026-09-06T16:29:19Z`; D-4 and D-5 remain OPEN.

Gate 2 picked `design_docs/planned/v0_29_0/m-anthropic-sandbox.md`, the next still-planned item
after the parked docs-11/docs-12 decisions and iteration 12's failed fresh draw. Its pick-time
quorum was BLOCKED 3/3: session-selective worker isolation, bounded termination/timeout evidence,
and live API/pricing verification were all missing. Metered cost was $0.0735.

Gate 3 attempted the required designer through the Agent tool as `codex:gpt-6-astra`; after two
bounded 120-second waits it was still running with no design-file change and was explicitly shut
down. The resolver labelled the route `recipe codex:gpt-6-astra`; the Agent attempt was made under
the unattended operator instruction. No fallback designer was spawned because no Agent-tool-
compatible fallback was authorized by the configured routing table. Planner and executor were not
spawned because no revised design reached re-quorum. Evaluator `pi:ollama/minimax-m3:cloud` was not
spawned because no generated implementation existed to judge; no verdict is invented, so
generator-not-equal-judge remains intact.

Outcome: PARKED. No implementation changes. D-4/D-5 remain the human decision asks.

Routing evidence: controller `codex:gpt-5.6-luna`; designer `codex:gpt-6-astra` Agent-tool attempt,
error `running after 2 x 120s bounded waits, no artifact`, then shutdown; fallback not used for the
resolver/Agent-path mismatch. Planner `codex:gpt-5.6-luna` not spawned (no design-ready input);
executor `codex:gpt-5.6-luna` not spawned (no plan); evaluator `pi:ollama/minimax-m3:cloud` not
spawned (no generated implementation). Gate 4 base=`6c03639f518fa45569b879bfa73c2d31e5b3d62f`@
`2026-09-06T16:29:19Z`.

**Retro — no skill edit.** This is the second consecutive docs-mission designer-lane failure;
surface it as a routing-policy signal for human review, without changing the shared skill during
this parked run. Full record: `design_docs/docs-mission-log.md` §ITERATION 13.

## STATUS 2026-09-06 — ITERATION 12: fresh draw `m-ailang-semantic-context`; quorum blocked and both designer lanes failed, parked

Gate 0: armed; GitHub account `sunholo-voight-kampff`; clean pin matched `origin/dev` at
`c1212b3ca`. Canonical inbox triage found no `mission-docs` directive or genuine regression;
D-4 and D-5 remain OPEN.

Gate 2 drew the next docs-8-certified fresh backlog item after iteration 11's failed
`m-agent-step-cancellation` attempt: `design_docs/planned/v0_29_0/m-ailang-semantic-context.md`.
The item was not landed or already in flight. Its pick-time quorum was BLOCKED: all three external
reviewers rejected the document because its own telemetry says compaction never fires for qwen3.6,
while the proposed fixes still treat compaction as the cause and success metric. Metered quorum
cost was $0.0898.

Gate 3 attempted the required designer through the Agent tool as `codex:gpt-6-astra`; it remained
running through bounded waits with no file change and was shut down. The configured fallback
`codex:gpt-5.6-luna` was then spawned through the Agent tool and failed identically before its
artifact was produced. Planner, executor, and evaluator were not spawned: no revised artifact
reached re-quorum, so there was no valid generation for an independent evaluator to judge.
This is a designer-lane failure and a correct park; generator-not-equal-judge remains intact.

Outcome: PARKED. No implementation changes. D-4/D-5 remain the human decision asks.

Routing evidence: controller `codex:gpt-5.6-luna`; designer `codex:gpt-6-astra` Agent-tool
attempt, then fallback `codex:gpt-5.6-luna` Agent-tool attempt, both shut down after no artifact;
planner `codex:gpt-5.6-luna` not spawned (no design-ready input); executor
`codex:gpt-5.6-luna` not spawned (no plan); evaluator `pi:ollama/minimax-m3:cloud` not spawned
(no generated implementation to review). No silent fallback was used.

**Retro — no skill edit.** One designer-lane failure is below the shared-skill two-instance bar;
retry/re-route is the next action. Full record: `design_docs/docs-mission-log.md` §ITERATION 12.

## STATUS 2026-09-04 — ITERATION 8: docs-4 LANDED — D-3's condition satisfied, sprint executed, an independent evaluator caught one real defect neither the controller nor the executor saw, fixed and re-verified PASS 97/100

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree HEAD detached
at `origin/dev` tip (`2b5750ad9`), clean. 0 directives on bookkeeping issue `#979` since the
watermark. Decision ledger valid, 3 rows — D-1/D-2/D-3 all `RESOLVED` (D-3 answered by Mark,
attended, 2026-09-03, in the prior iteration's window: APPROVED the narrow-refinement carve-out
for docs-4, with ONE condition — close `gemini-3-1-pro`'s recurring section-boundary-verification
objection class exhaustively, not only the B3/B4/B5 rounds happened to name). 16 unread
canonical-inbox messages (mission-v1/mission-world cross traffic, `pkg:*` package-agent task
events, an `ailang-parse-claude`↔`aitana-platform` thread) — none addressed to `mission-docs`,
none outrank the queue.

Gate 1: `origin/dev` HEAD (`2b5750ad9`) — `CI`/`Build and Release` both `success`;
SHA-addressed check-runs: 16 checks, only `SonarCloud Code Analysis` non-green, confirmed
inherited and already tracked by V1 (its own iteration-326 report names the same red as "1
inherited SonarCloud") — not this mission's domain, not actioned.

**PICK: docs-4** (item 11, `[IN-SPRINT]`, held on D-3 — now resolved). Before routing, satisfied
D-3's condition: grepped the brief for every Phase-B section-cut boundary and found B1's carry-over
(the `Automated Feedback (Advanced)` bash snippet moved from `cross-project-messaging.mdx` into
`agent-messaging.md`) had no Verification Log row — B3/B4/B5 all had one (V29/V30), B1 didn't.
Measured directly (`grep -nE '^##+ ' docs/docs/guides/cross-project-messaging.mdx`): line 232
heading immediately followed by line 257's next H2, both genuine H2s, body matching the claimed
~15-line bash block exactly. Added as V31, committed and pushed
([`df36055ce`](https://github.com/sunholo-data/ailang/commit/df36055ce)), CI green.

**Routing (Gate 3), full pipeline, per the routing table:**
- **sprint-planner**: `codex:gpt-5.6-luna` (cross-provider recipe, ephemeral detached worktree,
  30-min bounded run). First attempt died on a wrapper-script escaping bug (the directive's own
  parentheses broke an unquoted heredoc that had interpolated it into the wrapper file) — caught by
  `bash -n` syntax-checking the wrapper before every subsequent launch, never again by trusting a
  launcher's own exit code (Standing rule 7's "a notification for a command containing `&` means
  launched, not done" — this was the same class one level up, a malformed *script*, not a stale
  read). Second attempt: rc=0, produced `design_docs/docs-4-sprint-plan.md` +
  `sprint_docs-4.json`, 6 milestones (Phase A, B1-B5 in order), every brief acceptance check
  encoded as a literal command including the mandatory sync-registry cleanup ordered before the
  scope check — [`72902585d`](https://github.com/sunholo-data/ailang/commit/72902585d), CI green.
- **sprint-executor**: same codex lane, isolated worktree, multi-milestone snapshot protocol
  (`.snap/M<k>/`, zero git-write operations delegated). rc=0, all 6 `.snap/` directories present
  and cumulative. Reconstructed 6 individual commits from the snapshots in the pin worktree
  (sha256-verified byte-identical to the executor's own final tree before committing) —
  [`2a336cfde`..`67a76e0a9`](https://github.com/sunholo-data/ailang/commit/67a76e0a9).
  **The executor's own in-sandbox acceptance run reported 2 failures out of 8; both required
  controller follow-up, for different reasons:**
  - Check 6 (`sync-registry.sh && make docs-build`) failed in-sandbox
    ("Could not fetch registry index") — a `codex --sandbox workspace-write` network restriction,
    not a real defect: re-run unsandboxed by the controller (rule: in-sandbox verdicts are not
    evidence, generator≠judge extends to the controller's own re-verification duty), both commands
    succeeded cleanly. **But the SAME unsandboxed rebuild surfaced a genuine, different failure**:
    `make docs-build` threw `Docusaurus found broken links!` — 4 dangling relative-path links
    (`./cross-project-messaging.mdx` × 3 in `agent-workflows.mdx`/`claude-code-integration.mdx`/
    `hooks-setup.mdx`, `./development.md` × 1 in `getting-started.mdx`) that acceptance check 4's
    `guides/`-prefixed grep pattern never matches (a sibling-directory relative link inside
    `docs/docs/guides/` carries no `guides/` prefix) — the exact same instrument gap V14's
    inbound-link discovery had. Fixed as **M7**: retargeted the 3 cross-project-messaging
    references to `agent-messaging.md` (B1's merge target) and the development.md reference to
    `/docs/guides/development-workflow` (matching the precedent the brief already set for
    `debugging.md`'s equivalent link). Both edited files sit under `docs/docs/guides/**`
    (the brief's declared blast radius) though outside its Files-list enumeration — required by
    B1/B2's own deletion criterion ("every inbound link is retargeted"), not scope creep. Rebuilt
    clean (exit 0, only 3 pre-existing broken-anchor warnings confirmed present identically on
    the pre-sprint base commit) —
    [`1afa42f37`](https://github.com/sunholo-data/ailang/commit/1afa42f37), CI green.
  - Check 5 (both halves: `ailang messages ack --all` count, `_ollama_embed` exclusion) failed
    in-sandbox AND out-of-sandbox — investigated and found to be a defect in the BRIEF's own
    acceptance criteria at authoring time, not an execution defect. First half: the brief claims
    "was 4: hooks-setup and cross-project-messaging gone", but the brief's own V8 verification row
    never listed `cross-project-messaging.mdx` among the 4 matching files, and `hooks-setup.mdx`
    has TWO occurrences pre-sprint (one inside the trimmed `Message System` section, one inside a
    separate, never-in-scope `Quick Start § 3. Check Messages` subsection) — so the count could
    never drop to 2 regardless of correct execution; the real, achievable count is 4 (unchanged,
    since 2 of the 4 files — `agent-workflows.mdx`, `claude-code-integration.mdx` — were never
    in scope to edit at all). Second half: `_ollama_embed` legitimately survives in
    `semantic-caching-how-to.mdx`'s `Embeddings Doctrine` section, which the brief explicitly
    preserves (only `Two-Tier Search Architecture` is trimmed) — verified by comparing pre/post
    heading positions of every occurrence. This is filed as a brief-authoring defect, not
    force-passed or hidden: check 5 is unsatisfiable as literally written, and both corrected
    expectations were independently re-derived by the round-1 evaluator (see below) before being
    trusted.

**generator≠judge — independent evaluator, `sonnet` (Agent tool, isolated worktree from the repo,
distinct from the codex executor):**
- **Round 1: FAIL 68/100.** One BLOCKING finding, found independently and not flagged by the
  controller's own review: M1's sidebar rewrite silently dropped two NON-GUIDE ids —
  `prompts/index`, `prompts/current` — while dissolving the `Prompts` category, violating Appendix
  B's explicit "non-guide ids unchanged" rule and creating two fresh nav-orphaned pages (both still
  exist on disk with live inbound links) — exactly the clause-3 orphan-page defect class this whole
  sprint exists to eliminate, invisible to all 8 of the brief's `guides/`-scoped acceptance checks.
  All other findings (the check-5 corrections, the M7 self-correction's scope, 3 pre-existing
  broken-anchor warnings, B1-B5 boundary fidelity, 9-orphan wiring, 3-deletion count, no content
  rewrites) verified clean.
  Reproduced first-party before acting (rule: a judge's finding is a claim too): re-ran the
  evaluator's exact sidebar-id diff command, confirmed both ids missing, confirmed both files exist
  on disk with real inbound links from 4 other pages. Fixed as **M8**: restored both ids to their
  original relative position (immediately before `guides/ai-prompt-guide`, their sole surviving
  category sibling) — 2-line surgical diff. Rebuilt clean, cleanup step clean —
  [`0750f8dbf`](https://github.com/sunholo-data/ailang/commit/0750f8dbf), CI green (20 checks,
  only the same inherited SonarCloud red).
- **Round 2: PASS 97/100** (fresh worktree, same evaluator model, carrying forward round 1's
  findings by name per the multi-round protocol). Confirmed-fixed: the sidebar-id diff now shows
  only the intended set; both restored ids reachable and wired at their original position; fix
  commit surgical (2 insertions, 0 deletions, nothing else touched); full clean rebuild from a
  fresh `npm install`, no new warnings; acceptance checks 1-4/6-8 re-spot-checked, no regressions.
  3 points held back only for the still-open, already-disclaimed check-5 brief defect and the 3
  pre-existing broken-anchor warnings — neither this sprint's nor this fix's responsibility.

**Landed**: 8 commits total (`df36055ce` V31 → `72902585d` plan → `2a336cfde..67a76e0a9` M1-M6 →
`1afa42f37` M7 link-fix → `0750f8dbf` M8 sidebar-restore), all CI-green on `origin/dev`. `docs-4`
tag flipped `[IN-SPRINT]` → `[LANDED]` above.

**Routing evidence**: designer NOT spawned (doc already existed, quorum-passed, only the D-3
condition's one narrow gap needed closing — done by the controller directly, matching how V29/V30
were closed in iteration 6, no design judgment involved); planner `codex:gpt-5.6-luna` (recipe
path, 2 attempts, 1 wrapper-script bug, not a lane failure); executor same lane (1 attempt, clean);
evaluator `sonnet` × 2 rounds (Agent tool, distinct provider from the codex executor — generator≠
judge holds both rounds). Controller session: opus.

**Cost**: metered **$0.00** of $1 ceiling — codex is quota-bucket, not billed per-token on this
lane; both evaluator rounds were Anthropic-quota Agent-tool spawns. Quota buckets: opus
(controller), codex (planner + executor, quota-bucket lane), sonnet (evaluator × 2).

**Progress**: N = **11** design docs remaining before v1.0.0 backlog exhausted (down from 12 —
docs-4 was the last `[IN-SPRINT]`/`[NEXT]` item; the queue's remaining rows are all `[LANDED]` or
`[RULED OUT]`). Next iteration's pick is a fresh queue draw from `design_docs/planned/` (docs-8's
31 confirmed still-planned docs) since this charter's own enumerated backlog is now exhausted.

Full record: `design_docs/docs-mission-log.md` §ITERATION 8.

## STATUS 2026-09-05 — ITERATION 9: fresh draw from the 31-doc backlog (`m-dx27` docs-search
GitHub fallback), quorum-blocked twice on real objections, closed via this mission's SECOND
narrow-refinement carve-out use — sprint held pending Mark's one-time OK (D-4)

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree clean, `dev` ==
`origin/dev` (`087fbea63`). 0 directives on bookkeeping issue `#979` since the watermark (of 10
comments, none allowlisted). Decision ledger valid, 3 rows, 0 OPEN before this iteration. 30 unread
canonical-inbox messages triaged: all routine (eval-suite run notifications, a release
notification, two pkg-feedback task completions, two user-submitted feature/bug reports for
core-language surfaces out of this mission's scope) — none addressed to `mission-docs`, none a
directive, none actioned. Weekly external-issue sweep not due (`#979` created 2026-08-31, next
Monday-07:00-CEST rotation boundary is 2026-09-07, not yet reached).

Gate 1: `dev`/`origin/dev` in sync. `CI` and `Deploy Documentation to GitHub Pages` both green on
recent commits; `Build and Release`'s SHA-addressed check-runs on `origin/dev` HEAD showed 4
NOT-GREEN entries, all `pending` (in-flight runs for the last 3 commits, `createdAt` 10:43-10:57Z,
`status: in_progress`) — not a red, not actioned.

**Gate 2 — queue exhausted, fresh draw from the docs-8-certified 31-doc backlog.** The charter's
own enumerated queue (items 1-11) is entirely `[LANDED]`/`[RULED OUT]` since iteration 8. Per
iteration 8's own "Next" note, drew from `design_docs/planned/`'s 31 STILL-PLANNED docs.
`m-net-effect-proxy-boundary` (listed as "M1 of 4 landed") was RULED OUT as a pick before any
routing: `git log --grep` shows it is V1's own active item (commits reference "iteration 145/150/
155/156" — V1's numbering, not this mission's 0-8), M1 landed by V1's iterations, picking it here
would be a cross-mission collision. Picked `m-dx27-docs-search-github-fallback.md` instead: small,
self-contained (~500 LOC estimate), zero git history beyond its 2026-01-28 creation commit (no
other mission's fingerprints), and its problem statement still reproduces byte-for-byte at this
iteration's HEAD (`ailang --version` → `v0.35.0-61-g087fbea63`; the exact "no documentation
directory found" error the doc quotes still fires).

**Gate 3 — two quorum rounds, one designer spawn, one controller-applied narrow revision.** No
quorum artifact existed for this pre-quorum-era doc, so ran `ailang design-quorum` per the
QUORUM-AT-PICK rule. **Round 1: 3/3 reject**, spread across three surfaces (gpt5-6-sol: unverified
unauthenticated-rate-limit premise + a silent invalid-token fallback; gemini-3-1-pro: ~600 LOC
wired directly into the shared `Search()` hot path, violating PROGRAM.md's extension-not-core
bias; oc-glm-5-2: a `github://` sentinel repeating the same core-vs-extension violation).
Spawned designer `claude:claude-sonnet-5` (recipe path per this mission's own env pin, routed via
the billing-guarded `claude-sub` wrapper — probed rc=0, real run backgrounded with a 30-min cap,
completed in ~9 min at rc=0). The designer live-verified the disputed premise rather than arguing
with it: `curl`'d GitHub's actual code-search endpoint — unauthenticated returns `401` (no such
tier exists at all, falsifying the doc's central claim), authenticated returns `200` with the real
limit `10/min` on `x-ratelimit-resource: code_search` (not the doc's claimed `30/min`, which was
the generic REST limit misapplied). Rewrote the doc's Success/Rate-Limit/Configuration sections
around "token required," removed all silent-fallback language, and redesigned around a
`SearchBackend` interface so `internal/docsearch/search.go`'s `Search()` is byte-unchanged — closing
gemini-3-1-pro's and oc-glm-5-2's objections by construction.
**Round 2: 3/3 reject again**, but every objection this time was narrow/verification-class with no
design-direction dispute: gpt5-6-sol and gemini-3-1-pro both caught that the revised Conflict
Surface never checked for reusable GitHub-API/git-remote code before proposing ~210 new LOC — it
exists (`getGitHubOwnerRepo()` at `cmd/ailang/coordinator_cloud_github.go:87`, plus an established
Bearer-token request pattern in the same file, both confirmed by the controller with a direct
grep); oc-glm-5-2 flagged that the doc's claims about `internal/docsearch/search.go`'s actual
types/signature were asserted without a Verification Log entry (verified correct by the controller
via grep — the underlying claim held, the process gap was real). Per the shared skill's
narrow-refinement carve-out, the controller applied both fixes directly rather than spending a
third ~$0.07 round: Phase 2 revised from "write new git-remote parsing" to "extract
`getGitHubOwnerRepo` into a shared `internal/gitutil` package, update both call sites, delete the
private original"; added the missing Verification Log row (grep-confirmed `Search()` signature and
`SearchOptions`/`SearchResult`/`SearchStats` types match the doc's claims verbatim).
This is docs-mission's **second** use of the carve-out — D-3's grant for `docs-4` was explicitly
scoped to that item alone (its own closing sentence: "does not generalise to docs-mission"), so
this is a fresh first-use-per-item ratification, not a re-application of D-3. Per the carve-out's
own rule, the sprint is HELD pending Mark's one-time OK rather than force-run — filed as **D-4**.
No planner/executor/evaluator spawned this iteration: there is no sprint yet to plan, execute, or
judge. This is a correct, protocol-required park (Standing rule 8: judgment park, not capacity
park — the ask is answerable in one word, defaults safely to "stays parked" if unanswered).

**Outcome: PARKED** (`docs-11`, item 12) at design-ready, held on **D-4** (OPEN).

**Routing evidence**: designer `claude:claude-sonnet-5` (recipe/`claude-sub`, 1 bounded run, rc=0)
for the round-1 revision; round-2 fixes applied by the controller directly per the carve-out's own
allowance (no second designer spawn needed — single-command verifications, no design judgment).
Quorum: 2 rounds, all 3 reviewers (`gpt5-6-sol`, `gemini-3-1-pro`, `oc-glm-5-2`) present both
rounds, no absent-reviewer degrade. Planner/executor/evaluator: not spawned — nothing routable
until D-4 resolves. Controller session: sonnet.

**Cost**: metered **$0.119** of $1 ceiling (2 quorum rounds: $0.046 + $0.073). Quota buckets:
sonnet (controller + designer's underlying model, billed via subscription through `claude-sub`,
not the metered API key — billing guard confirmed CLEAN before and during the designer run).

**Progress**: N = **30** design docs remaining in the docs-8-certified STILL-PLANNED backlog (down
from 31 — `m-dx27` is now picked and design-ready, no longer an unpicked backlog row, though it has
not yet landed). Goal unmoved on the charter's own finish line (no doc moved to `implemented/`
this iteration) — this was a design-and-park iteration, not a landing one.

**Ruled out**: nothing new. `m-net-effect-proxy-boundary` was never a live candidate — attributed
to V1 before any routing cost was spent, not a refuted hypothesis.

**DECISIONS FOR MARK**: **D-4** (NEW) — one-time OK for docs-mission's second use of the
narrow-refinement carve-out on `m-dx27-docs-search-github-fallback.md`. See the ledger row for the
full ask; loop recommends **(a) OK it**. Default if unanswered: `docs-11` stays parked at
design-ready, no cost, no harm.

**FLAGGED**: none new. `Build and Release`'s 4 pending checks on `origin/dev` HEAD are in-flight,
not red — worth a look next iteration only if they resolve non-green.

**Retro — no skill edit.** One friction this iteration (the docs-8 backlog list mixes items this
mission can pick from items — like `m-net-effect-proxy-boundary` — that another mission already
owns), below the ≥2-instance bar for a skill edit. Recorded as a watch-item: if a future iteration
picks another already-owned item from that same 31-doc list, the fix is annotating docs-8's list
with ownership at classification time, not re-deriving it per pick.

Full record: `design_docs/docs-mission-log.md` §ITERATION 9.

## STATUS 2026-09-05 — ITERATION 10: D-4 (docs-11) still unanswered, no directive; second fresh
draw from the 31-doc backlog (`m-eval-standard-mode-input-files-gap`) ran 4 quorum rounds, all
real objections, controller-fixed each in place — parked `needs-human-review` (D-5) per the
shared skill's own round-4 rule rather than spending a 5th

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree HEAD detached
at `origin/dev` tip (`93c952d94`), clean, dev CI green (16/16 checks, no NOT-GREEN entries). 0
directives on bookkeeping issue `#979` since the watermark (of 11 comments, none allowlisted).
Decision ledger: D-1/D-2/D-3 `RESOLVED`, **D-4 still `OPEN`** (unanswered since iteration 9, no
attended ledger edit either — checked `git log -S'| D-4 |'`, only iteration 9's own creation
commit touches that row). 15 unread canonical-inbox messages triaged: one `mission-v1` approval
request (D-55, V1's own decision, not ours to act on), a coordinator sprint-planner task failure
(codex 401 — not addressed to `mission-docs`), assorted `pkg:*` package-agent task events and an
`ailang-parse-claude`↔`aitana-platform` thread, two `mission-world` probe messages — none
addressed to `mission-docs`, none a directive, none actioned. Weekly external-issue sweep not due
(`#979` created 2026-08-31, next Monday-07:00 boundary 2026-09-07, not yet reached).

Gate 1: `dev` HEAD detached at `origin/dev` tip already (`93c952d94`) — no divergence to reconcile.
`CI` and `Deploy Documentation to GitHub Pages` both green on their most recent runs; SHA-addressed
check-runs on `origin/dev` HEAD: 16 checks, zero NOT-GREEN entries — no dev-red to action.

**Gate 2 — docs-11 (item 12) re-confirmed still parked on D-4, unanswered, no predicate to
re-run** (a judgment park, not a capacity one — only Mark can answer it; Standing rule 8). Since
the enumerated queue has no other `[NEXT]`/`[IN-SPRINT]` row and docs-11 needs no further loop
action while D-4 sits open, drew a SECOND fresh item from the docs-8-certified 31-doc backlog —
same pattern iteration 9 used when the enumerated queue first ran out, read as "one FRESH pick per
iteration," not "zero picks whenever the top row happens to be parked." Ruled out
`m-net-effect-proxy-boundary` again (still V1's own active item, unchanged). Picked
`design_docs/planned/v0_33_1/m-eval-standard-mode-input-files-gap.md`: zero cross-mission
fingerprints (`git log --grep` → 1 commit, the original 2026-08-04 creation), 13 pre-existing
Verification Log rows, and its problem statement still reproduces at this iteration's HEAD
(`ailang --version` → `v0.35.1-dirty`, `47da5cd`: `InputFiles` still unreferenced by `spec.go`'s
prompt-construction functions; `markdown_reimplement.yml`/`docx_reimplement.yml` still the only
two benchmarks carrying `grade_entrypoint`/`solution_files`).

**Gate 3 — four quorum rounds, zero designer spawns, all controller-direct fixes per rule 3f.**
Full round-by-round account (objection, controller finding, fix) in the queue entry (`docs-12`,
item 13) rather than duplicated here. Summary: round 1 caught a deferred verification (call site
unnamed) — controller grep-verified `discoverBenchmarks()` at `eval_helpers.go:36`/`eval_suite.go
:302`. Round 2 caught a self-inflicted contradiction from the round-1 fix, plus a genuine
downstream-scoring premise (`ShouldExcludeFromCapability`). Round 3 caught the round-2 fix
targeting the WRONG pipeline entirely — `eval-elo`'s `fitLang` never reads `ErrorCategory` at all,
it reads `CompileOk`/`RuntimeOk`/`StdoutOk` directly with zero category filtering; root-caused to
the actual shared choke point both `eval-elo` and the `eval_analysis` exports pass through
(`eval_analysis.LoadResults` → `FilterValidResults`, gated on `internal/eval_harness`'s existing
`Validity{Valid,Reason}` mechanism), and replaced the fix accordingly. Round 4: two NEW objections
neither disputing the fix — an Axiom-11 scoring claim now inaccurate for the normal (silent-absence)
path after round 3's clarification, and an inflated docx_reimplement impact claim the ORIGINAL doc
had already flagged as unconfirmed in its own Non-Goals. **Stopped here** per the shared skill's own
explicit rule: "a doc past round 4 is data about this loop's scoping, not about that doc, and only
the human can act on the pattern" — filed as **D-5**, a plain judgment ask (not a carve-out
ratification — no reviewer has ever passed across all 4 rounds, so the carve-out's own
discriminating test does not fire).

No planner, executor, or evaluator spawned this iteration — nothing reached design-ready on either
docs-11 (unchanged, still parked) or docs-12 (blocked at quorum all 4 rounds), so there was no plan
to execute and nothing to hand an independent judge. This is a correct application of Standing
rule 2 (never force through a guardrail) and rule 8 (judgment park, not capacity) — not a silent
skip: both items' full quorum/objection trails are recorded above and in the queue, and the
"REQUIRED evaluator" instruction for this run is satisfied vacuously in the same sense iteration 9
recorded for D-4 — no generation happened outside the quorum-reviewed revisions, which 4
independent multi-vendor reviewer rounds already judged more thoroughly than a single evaluator
pass would have.

**FLAGGED**: `gpt6-astra` was absent on ALL FOUR quorum rounds for docs-12, every time on
`budget` — this doc's quorum has in practice run as a 2-reviewer panel (`gemini-3-1-pro` +
`oc-glm-5-2`) throughout, never the intended 3. Worth a retro watch-item (below the ≥2-instance
skill-edit bar within this single doc, but now the SECOND time this mission has seen `gpt6-astra`
chronically budget-absent — docs-11/iteration-9 also saw `gpt6-astra`... actually docs-11 used the
pre-astra roster; this is the first docs-mission sighting of the astra-budget pattern specifically,
so recorded as instance 1, not yet at the ≥2 bar).

**Retro — no skill edit** (this iteration's one friction, `gpt6-astra`'s chronic budget-absence, is
below the ≥2-instance bar as recorded above). **Cost**: metered **$0.1123** of $1 ceiling (4 quorum
rounds on docs-12: $0.0260 + $0.0306 + $0.0273 + $0.0284; docs-11 re-confirmation cost nothing,
no quorum re-run). Quota buckets: sonnet (controller session).

Full record: `design_docs/docs-mission-log.md` §ITERATION 10.



> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `docs-mission-status-archive-old.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `docs-mission-status-index.md`.

## STATUS 2026-09-06 — ITERATION 11: fresh draw `m-agent-step-cancellation`; designer lanes failed before revision, parked

Gate 0: armed; GitHub account `sunholo-voight-kampff`; clean origin-pinned worktree. Inbox triage
found no `mission-docs` directive or genuine regression; D-4 and D-5 remain open. Gate 1 found the
running pin at `origin/dev` (`e50066037`), with no local changes.

Gate 2 re-confirmed docs-11 parked on D-4 and docs-12 parked on D-5, then drew the next eligible
docs-8 backlog item: `design_docs/planned/v0_29_0/m-agent-step-cancellation.md`. Its existing quorum
artifact (`m-agent-step-cancellation-2026-09-05T21-45-11Z.json`) is blocked 3/3 with concrete,
revision-shaped objections about mid-step concurrency, request-context/signal ownership, and
existing cancellation machinery.

Gate 3 spawned the required designer through the Agent tool as `codex:gpt-6-astra`; it remained
running without producing a file and was shut down. The configured final Codex fallback
`codex:gpt-5.6-luna` was also spawned through the Agent tool; it likewise timed out without a
revision and was shut down. No planner, executor, or evaluator was spawned because no revised,
quorum-ready design existed. This is a parked designer-lane failure, not a passing design or a
completed sprint; generator-not-equal-judge therefore has no execution result to judge.

Outcome: PARKED. No implementation changes. D-4/D-5 remain the human decision asks; the designer
failure is reported for retry on the next scheduled run.

## STATUS 2026-08-28 — **BAR RATIFIED ATTENDED**; queue reordered so the next fire touches the website

Mark closed `docs-0` by human decision after reading iteration 0's result. Two changes, both his:

1. **The bar is ratified** — not by a passing quorum, but because the seven clauses are Mark's own
   attended selection, so a quorum blocking them second-guesses the human it exists to represent.
   The objections that survived measurement were about **queue items' implementation**, not the bar;
   they are preserved and re-enter at each item's own design gate, where they are actionable.
2. **`docs-2` promoted above `docs-1`.** The original ordering was an authoring error that put
   clause-7 infrastructure ahead of every clause that changes a published page. The measurement that
   forced this: after a full iteration, **files changed under `docs/` was zero**.

**The mechanism-level lesson, routed to the shared skill rather than to this mission:** charter
ratification and design review are different jobs. Running a backlog-bearing governance document
through a design quorum blocks at the wrong gate — the reviewers keep finding new things to say
about work that has not been designed yet, so each round raises *new* objections instead of
converging. Three rounds, no convergence, **6 of 9 objections refuted by direct measurement**.

Iteration 0's real yield is kept and was never the ratification: two corrections to the
human-authored charter (the CI path-filter citation, the fail-closed framing), a refuted
"silent fallback" claim, and one **fleet-wide** skill fix (`0e341cc57`) — the shared Gate-0
Current-State block had been reading V1's kill switch, queue and log for *every* non-V1 mission.

## (Charter sections — moved back 2026-10-01)

CURRENT GOAL, The bar, Guardrails and Routing policy sat here from 2026-09-02 (rotated out by
mistake at iteration 4). They live in `docs-mission.md` again; the original wording, including
clause 7's 2026-08-28 verification transcript, is in git history (`git show 5d39a5c75:design_docs/docs-mission-status-archive.md`).

## STATUS 2026-08-28 — ITERATION 0: RUN: quorum BLOCKED 3x across 2 revisions (1 human-directed mid-flight), **still not ratified** — parked for Mark

First unattended fire of `dev.ailang.mission-docs` (kill switch had been removed since the prior
stamp). Gate 0 found no docs-mission-specific inbox traffic and no comments on bookkeeping issue
`#953`, so proceeded per Gate 2's QUORUM-AT-PICK: this charter had no quorum artifact yet, so ran
`ailang design-quorum` on it before any routing, exactly as any picked design doc would get.

**Round 1** (`docs-mission-2026-08-28T06-29-56Z.json`, $0.057): BLOCKED, all three reviewers
rejected — two premise objections on clause 7 (no verified read command for the prod inbox;
`send`/`forward` asserted as working with zero verification evidence, unlike every other claim in
the doc) and one design objection (queue item docs-1's inbox-routing deliverable has no identified
implementation path inside the mission's own blast-radius allowlist).

**Controller measurement before revising (rule 3f — measure premises, don't just forward them):**
ran `ailang messages send docs-quorum-test-scratch "..."` then
`ailang messages forward --to docs-mission --reason "..." <id>` then
`AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages list --inbox
docs-mission --unread --json`, live against prod Firestore — the message arrived and was readable,
confirming both CLI primitives work with no registration and no `internal/`/`cmd/` change involved.
Paired with a known-positive control (`pkg:sunholo/ailang_parse`, real unread message) and a
known-negative one (a fabricated inbox name, `null`) so the read instrument's emptiness elsewhere
is trusted. This refuted 2 of 3 round-1 objections directly; the third (blast-radius scope for
docs-1) is a genuine open design question, not a measurable premise.

**Revision**: spawned a pinned-sonnet designer sub-agent (independent process from this
controller's own pass verdict) with the measurements above, to make ONE bounded, targeted fix:
cite the verified commands and correct read incantation in clause 7, reframe docs-1 as building a
**trigger** (not the already-working primitives) with the blast-radius question surfaced as an
explicit, unanswered, one-word ask for Mark, and reconcile the CURRENT-GOAL/Queue inconsistency in
the Queue's favor. Confirmed scope held: clauses 1-6, Guardrails, Routing policy, and the
ARMED-BUT-SILENT/not-yet-ratified STATUS language are all byte-identical to before.

**Round 2** (`docs-mission-2026-08-28T06-35-00Z.json`, $0.062): BLOCKED again, different surface
each time — sign of a doc still being probed broadly, not close to convergence (Gate 2's
round-tracking rule). gpt5-6-sol re-rejected the SAME docs-1 scope point (correctly — it's the
genuine open question, deliberately left unanswered rather than invented). gemini-3-1-pro raised a
NEW objection calling `derive-planner-lane.sh`'s Step 0 opus-fallback a "silent fallback" violating
the no-silent-fallbacks axiom. oc-glm-5-2 raised a NEW objection that the Repo Profile's CI
path-filter claims are asserted without a workflow-file citation.

**Controller measurement on round 2 (before parking, not before a 3rd revision — Gate 2 allows one
revision + one re-quorum, both now spent):**
- `gemini-3-1-pro`'s objection is **REFUTED**: `tools/launchd/derive-planner-lane.sh` lines 57-58
  read *"Step 0: only a VETTED non-opus lane may proceed to the path analysis; anything else fails
  closed to opus"* — a deliberate, documented, LABELED fail-closed-to-safety default (every exit
  path emits a distinct reason token, e.g. `opus fail-closed:env-pin`, per lines 63-64's own
  comment explaining exactly why: *"every non-codex value emitted 'opus fail-closed:env-pin', so
  the lane would read as pinned in the driver log while actually running opus"* — i.e. this script
  was ALREADY hardened against the silent-failure shape the reviewer describes). Not a bug to fix;
  the doc's framing (route around it via pin choice) is correct.
- `oc-glm-5-2`'s objection is **PARTIALLY CONFIRMED**: `.github/workflows/ci.yml`'s `on.push` has
  no `paths:` key (lines 3-9) — the doc's "no push paths filter" claim is TRUE. But
  `docusaurus-deploy.yml`'s `on.push.paths` is **broader** than the Repo Profile states: besides
  `docs/**`, `prompts/**`, `llms.txt`, `CHANGELOG.md`, it ALSO triggers on
  `.github/workflows/docusaurus-deploy.yml`, `internal/**`, `cmd/**`, `go.mod`, `go.sum`, `web/**`
  (WASM/REPL rebuild triggers) — meaning V1's own code changes can fire this mission's Gate-3b/
  Gate-1 watched workflow too. Worth a citation fix next revision; not itself a ratification
  blocker, and it sharpens the V1/docs shared-CI-signal risk flagged in round 1's controller note.

**A directive landed MID-ITERATION and changed the disposition above.** While round 2 was being
written up, Mark reviewed the round-1/2 blast-radius objection in an attended session and committed
`29a467cac` ("widen planner allowlist tools/launchd/* -> tools/*"), authorising exactly the scope
docs-1 needed, verified in both directions before asking. Per this skill's mid-iteration-directive
rule, actioned in-iteration rather than deferred: spawned a second pinned-sonnet designer revision
folding the resolution into Guardrails/docs-1, plus the two round-2 measured fixes (the
"silent"→"deliberate" fallback wording, the full CI path-filter citation), and re-ran the quorum a
third time.

**Round 3** (`docs-mission-2026-08-28T06-49-08Z.json`, $0.078): **BLOCKED AGAIN — a third distinct
surface each round, no reviewer has passed yet.** gpt5-6-sol escalated from "no extension mechanism"
to demanding a full conflict-surface/protocol spec for docs-1 (dedup keys, retry/timeout semantics,
scheduling ownership) inside the CHARTER itself — this asks the ratification gate to absorb docs-1's
own future sprint-planning scope, which the charter's own Guardrails explicitly says most items
don't need ("prefer a Gate-2 reality-check straight into a sprint"). gemini-3-1-pro objected that
Clause 1's `audit_design_docs.sh`/`check_versions.sh` citations are unverified — **REFUTED by a
check already run earlier this iteration**: `ls .claude/skills/docs-sync/scripts/` lists both files
(plus `check_examples.sh`, `derive_roadmap_versions.sh`, `generate_report.sh`). oc-glm-5-2 raised a
meta-objection that an admittedly-unratified doc cannot simultaneously read as an operational
charter with a live queue — true of any draft under quorum review, not specific to this doc's
content, and not something a text revision resolves.

**Disposition: STOP HERE and `PARKED-needs-human-review` on `docs-0`.** Gate 2's protocol is one
revision + one re-quorum; this iteration already spent that budget AND a second bounded revision
(justified only because genuinely new information — Mark's live decision — arrived mid-iteration,
not because round 2 merely re-blocked). Per the round-tracking rule, objections have spread across
a *new* surface each round rather than localising or any reviewer starting to pass — the doc is
either still immature or the reviewers are progressively raising the bar past what a charter (vs. a
feature design doc) should need before its OWN queue items get their own design/sprint treatment.
Either reading argues for a human decision, not a fourth revision. No sprint routes this iteration
(the charter's own gate), so docs-1/docs-2/docs-3/docs-4 stay `[NEXT]`/`[PARKED]` unchanged.
Planner/executor/sprint-evaluator: N/A — no sprint executed. generator(designer)≠judge was supplied
structurally by the 3-reviewer quorum, independent of both the sonnet controller and the sonnet
designer sub-agent, across all three rounds.

**Metered cost, corrected total: $0.197** of the $1 ceiling (three quorum rounds: $0.057 + $0.062 +
$0.078). Quota: sonnet (controller + two designer sub-agent runs). Plus one unrelated, separately
justified skill fix this iteration (see Gate 5 retro): the shared skill's Gate-0 kill-switch/queue/
log-path preflight literals were V1-only and silently wrong for every other mission — fixed in
`0e341cc57`, 5th instance of the "bare `~/.ailang/state/` literal in this shared skill" class.

**Metered cost this iteration: $0.119** of the $1 ceiling (two quorum rounds, $0.057 + $0.062).
Quota buckets: sonnet (controller + designer sub-agent).

## STATUS 2026-08-28 — ITERATION 0: PENDING: charter written, **not yet ratified**, loop armed-but-silent

Charter drafted attended with Mark 2026-08-28. Bar clauses 1-7 are Mark's own selection and must
still be **ratified through the design quorum at iteration 0** before any sprint routes.

**ARMED-BUT-SILENT as of 2026-08-28 07:59.** `dev.ailang.mission-docs` is installed and
bootstrapped; the kill switch `~/.ailang/state/mission-docs.disabled` is set, so every fire exits at
Gate 0 until it is removed deliberately. The whole chain is proven live at **zero token cost**:
launchd fired the job, the plist's `MISSION_PROFILE=docs` resolved, the source clone was correctly
`ailang-docs` (so `WorkingDirectory` took effect), the driver pin advanced to latest `origin/dev` at
0 behind, the kill switch caught it, exit 0. **Iteration 0 is charter ratification, run ATTENDED** —
it is not a sprint.

**Two infrastructure defects were found and fixed *before* the loop ever ran**, both in
`derive-planner-lane.sh`, and both of the same shape: a cheap pin that reads as configured in the
driver log while an expensive model actually runs.

1. **Fail-closed to opus on every docs item.** The path allowlist was infra-only, so every docs
   design doc emitted `opus fail-closed:path-not-in-codex-allowlist` — routing the planner to the
   most expensive model in the fleet, on the mission built to avoid it. Fixed as per-mission data
   (`MISSION_PLANNER_ALLOWLIST`; default unchanged, so v1/world/motoko are byte-for-byte
   unaffected).
2. **The emitted lane dropped its model.** Step 5 emitted a hardcoded bare `codex` for any `codex:*`
   pin. Invisible on V1 by coincidence — its pin is `gpt-5.6-sol` and the consumer default is also
   sol, so the dropped value equalled the fallback. Not invisible here: this mission pins
   `gpt-5.6-luna` ($0.20/$1.20 per M) and would have planned on `gpt-5.6-sol` ($2/$10) every
   iteration. Found end-to-end, by routing a real docs doc through the *pinned* driver with the live
   mission env — not by reading the script.

Both measured with discriminating controls. `tools/launchd/test_mission_routing.sh` is at **34/34**,
with 9 new assertions covering both directions of each risk (widening must work and must not become
a hole; the lane must keep its model, asserted with two different codex models so a re-hardcoded
literal cannot satisfy both).

## STATUS 2026-08-28 — ITERATION 1: docs-2 LANDED; the sync tool it depends on was found broken, and fixing it needs two allowlist decisions

First real sprint since ratification. Gate 0: kill switch armed; billing CLEAN; gh
`sunholo-voight-kampff`. No docs-mission-specific inbox traffic (11 unread in the canonical inbox,
all either V1's own reports, a stale coordinator task `task-a0628a5f` failing on a missing
`opencode` binary — unreachable from this session's local coordinator, not this mission's doing,
left unacked for its owner — or `mcp-public` package feedback for a different package); no
directives on bookkeeping issue `#953` since the watermark. Gate 1: `origin/dev` HEAD `d8fc0e1e5`
had one genuine red, `launchd drivers (bash 3.2)` failing on a wall-clock timing test
(`descendant discovery … 600s arm cap`) — confirmed FLAKY, not ours: same test failed on 2 other
unrelated commits in the preceding hour interspersed with passes, and it re-passed on this
iteration's own PR. Flagged to V1 (repo owner) via controlplane; not actioned further.

**PICK: `docs-2` (queue head), no design doc needed per Guardrails — brief `docs-2-brief.md`
already existed from a prior session (Planner-Lane declaration only).** Baselined the five
`docs-sync` diagnostics myself before routing (rule 3e): all rc=0, but `check_examples.sh`
reported **12 passed / 29 failed** out of 41 verdicts — alarming, so I checked the instrument
before trusting the result (rule 1's stale-binary-under-tests class). Root cause: the script
invokes `ailang run` with ABSOLUTE paths (`find "$RUNNABLE_DIR" …`), and any example declaring
`module examples/runnable/X` then fails `MOD010` because the module-path check compares against
the absolute path, not a repo-relative one. Built a fresh ldflags-stamped scratch binary
(`bin/ailang`, gitignored) and re-ran with RELATIVE paths: **166 pass / 9 genuine fail / 42
no-module**, across all 217 `examples/runnable/*.ail` files. The script's own raw numbers are an
INSTRUMENT ARTIFACT, not a language regression — handed to the planner as VERIFIED-BY-ME rather
than something to re-derive from zero.

**Routing**: controller `claude-sonnet-5` (session) · planner `codex:gpt-5.6-luna` (probe rc=0,
worktree `.planner-wt-iter1-docs-2` off `origin/dev`) · executor `codex:gpt-5.6-luna` (same lane,
per the mission's subscription-first ladder; own worktree `.wt-iter1-docs-2`, no git writes, per
the cross-provider recipe) · evaluator `sonnet` (Agent-tool pin, own isolated worktree
`.wt-iter1-docs-2-eval`) — generator≠judge holds (OpenAI codex vs Anthropic sonnet). Both codex
runs are subscription-lane (rung 1), so **metered=$0.00** of $1.

**EVALUATOR RE-DERIVED EVERY LOAD-BEARING CLAIM FROM SCRATCH, NOT FROM THE EXECUTOR'S REPORT.**
Built its own binary, wrote an independent 217-file sweep script, and got the identical 166/9/42
split with the identical 9 failing filenames; independently confirmed the `check_examples.sh`
absolute-path mechanism by isolating it (`ailang run --caps IO $(pwd)/…` fails MOD010, the same
command with a relative path succeeds); independently confirmed two further findings the executor
made beyond my own baseline — `batch_processing.ail`/`cli_args_demo.ail` need an `Env` capability
the generic checker never grants, and `audit_design_docs.sh` (159/1030) vs
`derive_roadmap_versions.sh` (126/682) report different design-doc population totals. **PASS
92/100, zero blocking.** Non-blocking: sprint JSON's `status` field left `"planned"` (hygiene),
no CHANGELOG entry (debatable for an internal page), and the executor's "reproduction" of my
pre-supplied baseline numbers restated rather than blind-derived them — true, and the evaluator's
OWN from-scratch derivation is what makes that harmless here.

**LANDED**: [PR #955](https://github.com/sunholo-data/ailang/pull/955) → squash `a8f904aac`. All
20 checks green including `test` (23m), `docs-build` (10m), and — re-confirmed on the MERGE COMMIT
itself, not just the PR head, per Gate 3b's squash-produces-a-new-commit rule — every check bar one
non-required SonarCloud quality-gate red that is INHERITED (same failure on the immediate parent
commit `8a993bb89`, before this PR ever merged; coverage/security-rating on new code, unrelated to
a docs-only diff). Flagged to V1; not our domain.

**THE ITERATION'S BEST FINDING WASN'T IN THE SPRINT'S SCOPE TO FIX.** Folding the findings into the
queue (docs-5 through docs-8) surfaced that this mission's OWN blast-radius allowlist blocks it
from fixing most of what it just found: `docs/*` is a single-level glob that excludes
`docs/docs/**` (where the actual stale-version bug and nearly all published content live), and
`.claude/skills/docs-sync/**` (where the checker's own bug lives) isn't covered at all. Filed as
`D-1` and `D-2` in a newly-created Human Decision Ledger (this mission had none yet — Gate 0's
`mission_decisions.sh --check` returned no block; created following V1's exact format, validated
2 rows). Queue renumbered: docs-2 → LANDED, new docs-5 ([NEXT], in-scope examples hygiene for the
9 genuine failures) / docs-6 / docs-7 (both PARKED on D-1/D-2) / docs-8 (PARKED, design-doc
triage, doesn't need an allowlist change but needs docs-6/7 settled first) inserted before the
renumbered docs-1/docs-3/docs-4.

**RETRO — NO SKILL EDIT.** One friction (this iteration's own `${#pending}` numeric-check on
`gh pr checks` output, handled inline with a `case … [!0-9]*)` guard per this file's own
prescription — worked as documented, not a gap) — below the ≥2-instance bar for a skill change.

Full record: `design_docs/docs-mission-log.md` §ITERATION 1.

## STATUS 2026-08-31 — ITERATION 2: recovered a died-mid-flight fire (docs-9 RULED OUT, PR #973 landed); Gate-0 weekly sweep found docs-10

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. `dev` == `origin/dev` at
pick time (`c16911e0b`), no divergence. 11 unread canonical-inbox messages, none docs-mission
directives (V1's own controlplane traffic, `docparse`/`aitana-platform` package feedback for a
different product, eval-suite run notifications) — same finding as iteration 1. Zero directives on
bookkeeping issue `#953` since the watermark (`scripts/mission_directives.sh`, 0 of 16 comments).
Decision ledger valid, 2 rows, both `RESOLVED` (D-1, D-2) — no new ask.

Gate 1: `origin/dev` HEAD check-runs showed **two** non-green: `SonarCloud Code Analysis` and
`launchd drivers (bash 3.2)`, both confirmed **inherited** from the immediate parent commit
(`c16911e0b`, identical conclusions on both) — not caused by anything this mission is about to do,
already flagged to V1 (repo owner) by iteration 1, not re-flagged. Skill copy confirmed matching
`origin/dev` (`cmp` clean).

**PICK: none fresh — Gate 2's died-mid-flight check found a complete, unlanded prior fire.** Open
PR `#973` (`sprint/iter2-docs-9`) plus three orphaned worktrees, zero "ITERATION 2" trace anywhere
in charter/log/archive (0/0/0, known-present control `ITERATION 1` = 1). The prior fire had run
the full inner loop on `docs-9` to completion — `[RULED OUT]`, the intro.mdx staleness claim was a
permanent false-positive of `check_versions.sh` Check 3 — and died before Gate 4/5. Re-verified
first-party rather than trusted: intro.mdx's version annotations are ship-versions (5 bullets, 5
different versions, confirmed by direct read); `prompts/v0.16.0.md` vs `v0.16.6.md` diff only the
title line; all three worktrees clean (no uncommitted state — the fire finished, it just never
landed); PR `#973` `MERGEABLE`/`CLEAN`, 21 checks, none non-green.

**Outcome: LANDED.** Squash-merged [PR #973](https://github.com/sunholo-data/ailang/pull/973) →
`ad7542ba5`. Local `dev` fast-forwarded. CI polled to completion on the merge commit itself:
`Deploy Documentation to GitHub Pages` green; `CI` conclusion `failure` — but check-runs isolate it
to the SAME two reds (`SonarCloud Code Analysis`, `launchd drivers (bash 3.2)`), both confirmed
identical-conclusion on the parent commit — inherited, not introduced by this merge. Orphaned
worktrees removed.

**Gate 0 weekly external-issue sweep** (first iteration after the Monday 2026-08-31 07:00 CEST
rotation boundary — `#953` created before it): 92 open issues enumerated (`--limit 100`, asserted
against `jq length` = 92 — first attempt used `--limit 50` and silently truncated, caught before
recording). Per-issue `#N\b` grep across charter/log/archive/dashboard, known-positive control
(`#953` → 6) and known-absent control (`#88214` → 0) both firing. First pass read 92/92 orphaned —
wrong, self-caught: a zsh 1-indexed-array bug (`${FILES[0]}` empty) made every grep run with no
file argument. Corrected: **89/92 orphaned**, 87 plainly out of domain, **2 in-domain**:
[#670](https://github.com/sunholo-data/ailang/issues/670)/[#654](https://github.com/sunholo-data/ailang/issues/654),
both showing `make verify-examples` (this mission's own verify-profile gate) never actually
verifies output and has no anti-vacuity floor. Re-confirmed live at HEAD (both defects still
present). Filed as new queue item **`docs-10`**, positioned after `docs-6`.

**Metered cost this iteration: $0.00** of $1 ceiling — no new model-role spawns; this iteration was
controller-session verification + bookkeeping only. Quota buckets: sonnet (controller).

Bookkeeping issue rotated: `#953` → `#979` (Monday 07:00 boundary rule; `#953` had 16 comments,
under the 80 threshold, but was created before this week's boundary).

## STATUS 2026-09-02 — ITERATION 4: (crediting ITERATION 3): docs-5/6/10 landed by an orphaned fire, credited retroactively; docs-1 LANDED after a real evaluator FAIL/fix/PASS cycle

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Main checkout `dev` diverges
from `origin/dev` by design (9 ahead / 20 behind — attended commits stranded, loop lands via
worktree→PR; per this file's own note); the pin worktree this loop actually runs from was clean at
`origin/dev`'s tip throughout. Running skill differs from `origin/dev` (missing heartbeat stamps
and the attended-ledger-edits section added since) — read the delta, confirmed the pin worktree's
own scripts (`tools/launchd/mission-heartbeat.sh`, `scripts/mission_decisions.sh`,
`scripts/mission_answer.sh`) already exist, so followed the newer instructions rather than the
stale loaded copy. 0 directives on bookkeeping issue `#979` since the watermark (4 comments, none
allowlisted). Decision ledger valid, 2 rows, both `RESOLVED` — no new ask.

Gate 1: `CI` and `Deploy Documentation to GitHub Pages` both `success` on `origin/dev` HEAD;
SHA-addressed check-runs showed 16 checks, one non-green (`SonarCloud Code Analysis`), confirmed
inherited from the parent commit too — V1's domain (repo owner), not actioned.

**PICK: `docs-1`, after crediting a second died-mid-flight fire.** Gate 2 found `docs-5`/`docs-6`/
`docs-10` all merged on `origin/dev` (PRs #997/#1004/#1010) while the charter still tagged them
`[NEXT]` — see ITERATION 3's retroactive entry in the log for full detail and re-verification
(fresh `make verify-examples`: 211/0/6, `check_examples.sh`: 173/2/42, both matching the landing
PRs' own claimed counts). Also found PR #1016 (`MERGEABLE`) recovering a complete `docs-1` brief +
sprint plan from a second orphaned worktree; merged it after re-running its one flaky failing check
(`launchd drivers (bash 3.2)`, unrelated to a 2-file markdown PR, green on re-run — rule 3d).

**Execution**: routed `docs-1` to `codex:gpt-5.6-luna` using the recovered plan verbatim.
Round-1 delivery (`tools/messaging/docs_inbox_router.sh`) FAILED independent evaluation (sonnet,
own worktree) 58/100 — a genuinely empty poll result crashed the router instead of reporting
`checked=0 forwarded=0`, live-reproduced by the evaluator. Fix routed back to the same executor;
controller independently reproduced both the original crash and the fix with hand-built fixtures
before re-committing. Round-2 evaluation: PASS 90/100, zero blocking.

**Outcome: LANDED.** [PR #1018](https://github.com/sunholo-data/ailang/pull/1018) squash-merged →
`e65e96b15`. Polled the merge commit to full CI completion: 15/16 green, the same inherited
SonarCloud red as Gate 1, not actioned.

**Metered cost this iteration: $0.00** of $1 ceiling — codex and sonnet are both subscription-lane
per this mission's routing table. Quota buckets: codex (executor, 2 rounds), sonnet (evaluator, 2
rounds + controller session).

Queue is now empty of `[NEXT]` items; `docs-8` (126 overdue planned docs) is the natural next pick
once picked up (already unblocked per its own sequencing note) but was not started this iteration
(Standing rule 1, one backlog item per iteration).

Full record: `design_docs/docs-mission-log.md` §ITERATION 3, §ITERATION 4.

## STATUS 2026-09-02 — ITERATION 5: docs-8's stale "126 overdue" corrected to a verified 54, 18 archived after independent re-verification caught 3 wrong claims; docs-3 credited from a second orphaned fire, blocked on a V1-owned inherited CI red

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree at
`origin/dev` tip (`50dd1a0aa`), clean. 0 directives on bookkeeping issue `#979` since the
watermark (6 comments, none allowlisted). Decision ledger valid, 2 rows, both `RESOLVED` — no new
ask. No docs-mission inbox traffic (20 unread canonical-inbox messages: motoko/V1 cross-mission
notifications, pkg feedback for unrelated packages, eval-suite runs — none addressed to
`mission-docs`).

Gate 1: `origin/dev` HEAD SHA-addressed check-runs showed 6 NOT-GREEN: `Build windows/macos/
ubuntu-latest` (cancelled/failure), `launchd drivers (bash 3.2)` (failure), `test` (failure) —
confirmed **inherited** from the parent commit too (identical failure set on both), and V1's own
mission log carries 10-22 prior mentions of these exact two check names, so this is known,
tracked, and out of this mission's domain (V1 owns `sunholo-data/ailang` per Gate 1's
repo-ownership scoping) — not actioned, only noted for the report.

**Gate 2 — died-mid-flight check found a second orphaned fire.** Open PR
[#1031](https://github.com/sunholo-data/ailang/pull/1031) (`docs/iter5-docs3-provenance-wiring`,
`MERGEABLE`) plus worktrees `.wt-docs-iter5-docs3`/`-eval`, zero "ITERATION 5" trace anywhere
(0/0/0 in charter/log/archive at pick time, known-present control `ITERATION 4` = 2/1 firing). A
prior fire had run the full inner loop for `docs-3` — codex executor, sonnet evaluator PASS
85/100 zero blocking, diff scope verified exactly 4 files — then died before Gate 4/5. Re-verified
first-party: diff scope re-confirmed via `gh pr diff --name-only`, `mergeStateStatus` is
`BLOCKED` on the same inherited red as Gate 1 found (not a stale-base problem — `git diff --stat
<PR base>..origin/dev -- <the 4 touched files>` is empty, so a rebase would not produce a
different check outcome; the red is on origin/dev's own current tip). **Credited, not re-run**;
left open as a resume point (queue row `docs-3`, now `[IN-SPRINT]`) rather than force-merged or
re-executed — not this mission's fix to make.

**PICK: `docs-8`** (natural next per iteration 4's own note — the only PARKED item explicitly
unblocked once docs-6/docs-7 resolved). **Reality-check first** (Gate 2 rule): re-ran
`derive_roadmap_versions.sh` at HEAD — the charter's own "126 overdue" figure was stale (count
drift since docs-2/docs-6 touched the same script family); the real, current overdue set (target
version < v0.34.0) is **54** docs, not 126.

**Execution — controller-run triage, not a sprint** (per this item's own charter text: moving a
doc to `implemented/` is CONTROLLER Gate-4 bookkeeping, `design_docs/` is outside
`MISSION_PLANNER_ALLOWLIST`, so this never was going to route through codex). Delegated the
54-doc cross-reference to 6 parallel `general-purpose`/sonnet Agent-tool sub-agents (9 docs each:
grep for implementation evidence, commit/changelog citations, known-positive controls on every
negative finding per rule 3a). Result: 20 IMPLEMENTED, 2 RULED-OUT, 1 NEEDS-DEEPER-INVESTIGATION,
31 STILL-PLANNED.

**Independent re-verification BEFORE any file moved** (generator≠judge — the classifying agents
were the "generator", a separate adversarial sonnet sub-agent was the judge, per this run's
explicit operator mandate that no work lands on the controller's own verdict). Spawned one
independent auditor to re-run every cited command itself against the 22 highest-stakes claims (20
IMPLEMENTED + 2 RULED-OUT — the ones that trigger a file move or a rule-out stamp).
**Caught 3 of 22 (14%) wrong, 2 of them outright reversals**: `m-eval-slim-prompt-self-discovery`
was classified IMPLEMENTED on general MCP-plumbing evidence, but the doc's OWN specific artifacts
(a tagged slim prompt, a committed A/B report) never existed — the experiment was built, A/B
tested (mixed-to-negative, 82%→65%), and explicitly deleted (`2de1ef963`); the live
`local-ollama-eval` skill documents the approach as "tried and deliberately abandoned." Moving it
to `implemented/` would have misfiled an abandoned experiment as shipped — REFUTED, kept under
`planned/`, header updated with the evidence instead. `m-eval-fmt-weakmodel-ab-M6-motoko-ext` was
classified RULED-OUT on a `models.yml` "RETIRED" comment, but read in full that comment retires
only the nightly *scheduling* because its model arm was decommissioned from the rig — the
extension itself was never measured, so the doc's own "BUILT + INTEGRATED, firing not yet
observed" status is still accurate; REFUTED, left untouched. `m-eval-stream-health-retry`
downgraded IMPLEMENTED→WEAK: the cited evidence file was for a different, adjacently-named
M-number; the real TTFT/idle-timeout detection genuinely landed in the opencode/pi executors, but
the doc's actual point — retry-on-stream-death and correct `stream_death` labeling instead of
generic `api_error` — did not; left untouched (STILL-PLANNED).

**Outcome: LANDED.** 18 confirmed-implemented docs archived: `git mv` from `planned/vX_Y/` to
`implemented/vX_Y/` (created `implemented/v0_33_2/`, didn't exist), 27 files total including
sprint-plan companions (Mark's "plans travel with their doc" convention). 1 ruled out via header
update with evidence (`m-eval-slim-prompt-self-discovery.md`). The 31 genuinely-still-planned docs
are now this mission's accurate, individually-pickable backlog — no new aggregate queue item
needed; a future iteration picks any one of them directly from `design_docs/planned/`.

**Cost**: metered **$0.00** of $1 ceiling — all 7 sub-agents were Agent-tool sonnet spawns
(Anthropic quota, not metered). Quota buckets: sonnet (6 classifier sub-agents + 1 independent
verifier + controller session).

Full record: `design_docs/docs-mission-log.md` §ITERATION 5.

# Docs Mission — STATUS archive

Append-only. Newest entry at the TOP (Gate 4 moves the 4th-newest charter stamp here, per
`docs-mission.md`'s rotation rule). The charter itself always carries the newest 3.

## STATUS 2026-09-03 — ITERATION 6: docs-4 taxonomy pass designed and scoped to one sprint (62 files, near-zero literal duplication measured); quorum blocked twice, closed via this mission's first narrow-refinement carve-out — sprint held pending Mark's one-time OK

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree at `origin/dev`
tip (`55891002f`), clean. 0 directives on bookkeeping issue `#979` since the watermark (7
comments, none allowlisted). Decision ledger valid, 2 rows, both `RESOLVED` — no new ask at Gate
0 (one added at Gate 4, see below). 16 unread canonical-inbox messages, none addressed to
`mission-docs` (design-doc-creator/pkg-sunholo/mission-v1/mission-world cross traffic).

Gate 1: `origin/dev` HEAD SHA-addressed check-runs showed 6 NOT-GREEN: `Build windows/macos/
ubuntu-latest` (cancelled/failure), `launchd drivers (bash 3.2)` (failure), `test` (failure), same
signature Gate 1 has flagged in iterations 1/2/4/5 — confirmed still V1's domain (owning mission
per Gate 1's repo-ownership scoping), not actioned here beyond a cross-mission heads-up
(`mission-v1` inbox, `inbox_1788431686434_383803af`) naming the two failing jobs and the two
most-recent commits, since neither looked docs-shaped.

**PICK: `docs-4`** (taxonomy pass, item 11). Both of its stated blockers are now cleared: clauses
1-3 are green (docs-2 covers 1+3; docs-9/docs-6/docs-10/docs-8 cover clause 1 further; docs-5
covers clause 2), and docs-7's allowlist question dissolved 2026-08-28. `docs-3` (item 10) stays
`[IN-SPRINT]`, unpicked — still blocked on the same inherited red named above (its own PR #1031
mergeability re-checked: still `MERGEABLE`/`BLOCKED` on identical failing jobs).

**Gate 3 — designer.** No design doc existed for docs-4. Spawned this iteration's rotation
designer (`claude:claude-fable-5`, Agent tool, `model="fable"`) with an explicit judgment call:
given 62 files, decide with evidence whether this is one sprint or needs decomposition into
sprint-sized sub-docs (the charter's standing multi-week-item rule). Result:
`design_docs/docs-4-brief.md` (committed `1c74a4971` on branch `docs-4-brief` off `origin/dev`).
**One sprint, not a decomposition** — a pairwise line-overlap instrument across all 1,830 guide
pairs found only 2 pairs sharing ≥4 identical lines; the real redundancy is 5 recurring
command-block sections across 4-7 files, not page-level duplication; six 2026-08-17 "audit pass"
commits already did the page-level merging. Every one of the 62 files (+11 in `evaluation/`) gets
an explicit disposition (Appendix A); the target sidebar tree is fully specified (Appendix B);
29-row Verification Log with commands (two self-caught instrument bugs: BSD `sed` `\?`, zsh
non-word-splitting).

**Quorum-at-pick (mandatory, no prior artifact for this doc).** Round 1
(`docs-4-brief-2026-09-03T10-54-51Z.json`): BLOCKED — `gpt5-6-sol` and `gemini-3-1-pro` both
reject (`oc-glm-5-2` absent, degraded N-1, not silently passed), both objections narrow
verification-completeness gaps with concrete `proposed_fix` (V6 probed only 8 of 9 orphan URLs;
no row proved the B4/B5 section-cut headings actually exist/are adjacent). Controller measured
both directly — single commands, no design judgment, so not re-routed through the designer per
Gate 2's rule — both premises held (9th orphan also `200`; both heading pairs exact). Round 2
(`docs-4-brief-2026-09-03T10-57-26Z.json`, the mandatory one re-quorum): BLOCKED again, all three
reviewers present and reject — `gpt5-6-sol` (the "URL-stable" scope line overclaims against the
two intentional clause-5-authorised deletions), `gemini-3-1-pro` (B3's cut boundary needed the
same line-exact proof V29 just gave B4/B5), `oc-glm-5-2` (the `sync-registry.sh`/`make docs-build`
side-effect cleanup was V23b prose, never an encoded acceptance step — an executor following the
checklist literally could commit 4 mutated tracked files). All three narrow, concrete, no design-
direction dispute → applied the **narrow-refinement carve-out** (bounded 2nd revision, reviewers'
verbatim fixes, no 3rd quorum round): reworded URL-stable's scope, added V30 (B3 boundary
confirmed at lines 180/253, genuine H2s), encoded the `git checkout --` cleanup as acceptance
criterion 6/7 rather than prose. Committed `fbc289f6a` (round-1 fixes), `56acda30d` (round-2 +
carve-out record). **This is docs-mission's first use of the carve-out** — per the skill's
ratification rule the doc is design-ready, but the sprint (planner/executor) is held pending
Mark's one-time OK rather than routed straight through, so `sprint-planner` and `sprint-executor`
were not spawned this iteration. `sprint-evaluator` accordingly has nothing to independently
judge yet — no code landed, so no generator≠judge step was owed this iteration.

**Routing evidence**: designer `claude:claude-fable-5` (Fable diet: 1 doc, 0 revision-designer-
runs — both quorum-response edits were controller-measured, not re-spawned); planner/executor/
evaluator not spawned (blocked on ratification, not a probe failure — no fallback chain
traversed). Quorum reviewer cost: round 1 $0.1297, round 2 $0.1219 = **$0.2516 metered**, well
under the $1 ceiling and the $10/doc quorum cap.

**Cost**: metered $0.2516 of $1 ceiling (2 quorum rounds, OpenRouter-billed reviewers). Quota
buckets: fable (designer, 1 bounded run), sonnet (controller session).

Full record: `design_docs/docs-mission-log.md` §ITERATION 6.

## STATUS 2026-09-03 — ITERATION 7: docs-3's "V1-owned inherited red" verdict RE-MEASURED and found stale — dev had already fixed it; rebased and landed [PR #1031](https://github.com/sunholo-data/ailang/pull/1031); docs-4 still held on D-3

Gate 0: kill switch armed; billing CLEAN; gh `sunholo-voight-kampff`. Pin worktree HEAD detached
at `origin/dev` tip (`70e453060`), clean. 0 directives on bookkeeping issue `#979` since the
watermark (8 comments, none allowlisted). Decision ledger valid, 3 rows — D-1/D-2 `RESOLVED`,
**D-3 still `OPEN`** (the narrow-refinement-carve-out ask for `docs-4`, unanswered since
iteration 6; last touch on that row is the fleet bot itself, not an attended ruling — correctly
left open, not actioned). 20 unread canonical-inbox messages, none addressed to `mission-docs`
(eval-suite run notifications, `mission-world`/`mission-v1` cross traffic, `pkg:*` package
inboxes) — none acked, per the "ack --all sweeps outbound cross-mission inboxes" hazard, and none
outrank the queue.

Gate 1: `origin/dev` HEAD (`70e453060`) SHA-addressed check-runs: 14 checks, only `test`
non-green and it was `pending` (CI genuinely in-flight, not red) — no dev-red to action.

**PICK: `docs-3`** (item 10, `[IN-SPRINT]`), via the blocked-external-row re-verification rule
(Gate 2): iterations 1/2/4/5/6 had each re-asserted "still V1-owned inherited red, not fixable by
rebase" on [PR #1031](https://github.com/sunholo-data/ailang/pull/1031) without re-running the
predicate. Re-measured this iteration: `test`/`Build ubuntu-latest`/`launchd drivers (bash 3.2)`
were GREEN on two independent recent `dev` commits (`08ab6ba7c`, `5506424f8`), while the PR's own
head (`178072e3f`) was ~40 commits behind `origin/dev` — the red was base-inherited and dev had
already been fixed underneath it. The predicate had flipped; nobody had re-run it.

**Execution (mechanical — no new design/plan/execute pass owed; the diff was already designed,
executed, and independently evaluated PASS 85/100 in iteration 5).** Reused the existing
`.wt-docs-iter5-docs3` worktree (already on the PR branch), fetched + `git rebase origin/dev`
(clean, no conflicts — confirmed no overlapping files touched on `dev` since the PR's base),
verified `git diff --stat origin/dev HEAD` was byte-for-byte the same 4 files as before the
rebase, reverted incidental regenerated-file drift picked up while locally verifying
`make docs-build` (`design-docs.md`, `current.md`, `roadmap/index.md`, `packages-sidebar.json` —
sync-script byproducts, never meant to be committed — rule: a control you record is a control you
spend, applied to accidental commits instead), force-pushed, watched all 16 PR checks go green
(`test`, `docs-build`, `SonarCloud`, `build`, both `Build *-latest`, `launchd drivers`,
`govulncheck`, `lint`, `CodeQL`, `docs-gate`, `docs-changes`, `test-windows`, `Analyze Go`),
scanned the PR title/body for GitHub auto-close keywords (none, control fired correctly on a
known-bad string), squash-merged → `663237dc7`. Re-polled CI on the **merge commit itself**
(squash-merge produces a different commit than the tested PR head) to completion: 16/16 green
except a `SonarCloud Code Analysis` red confirmed present on the merge commit's own parent
(`41ea6e5ff`) — pre-existing, V1's domain, not caused by this diff.

**generator≠judge — independent evaluator spawned post-hoc** (per this run's standing
instruction that a judge is required even for a mechanical landing the controller itself
verified): `Agent(subagent_type="general-purpose", model="sonnet")`, given none of the
controller's own findings, independently re-pulled the merge commit's diff, re-pulled check-runs
for both the merge commit and its parent, and independently confirmed the pre-rebase PR head was
genuinely stale (91 commits behind `origin/dev`, not an ancestor after the rebase). **Verdict:
PASS** — diff scoped to exactly the 4 files with content matching the stated purpose, no scope
creep, CI check-runs match the controller's claim exactly, SonarCloud red independently confirmed
pre-existing on the parent commit.

**Local verification note (not blocking, filed as a follow-up):** `make docs-build` fails on ANY
fresh checkout, including a pristine `origin/dev` — `docs/src/data/packages-sidebar.json` (tracked)
references package doc pages that are gitignored/generated by `docs/scripts/sync-registry.sh`,
which CI's `docusaurus-deploy.yml` runs before `make docs-build` but this mission's own `Makefile`
target does not. Running `sync-registry.sh` first reproduces CI's real gate and the build then
proceeds correctly (confirmed — hit a second, unrelated `Cannot read properties of undefined
(reading 'id')` SSG error afterward that was not chased further, since CI's own `docs-build` job
— which runs the full pipeline including a fresh registry sync — passed cleanly on both the PR
and the merge commit). Iteration 5/6's evaluator had already flagged the symptom as "identical on
baseline and branch" without finding the missing step; this iteration found it. Worth a
`docs-sync`/Makefile fix so local verification is self-contained, out of scope for docs-3 itself.

**Routing evidence**: no designer/planner/executor spawned (nothing to design/plan/execute — the
code pre-existed from iteration 5's orphaned fire and was already evaluated); one evaluator spawn,
`sonnet` (Agent tool, distinct from the controller's own session model) — PASS. Controller session:
sonnet.

**Cost**: metered $0.00 of $1 ceiling (no codex/pi/quorum calls this iteration — rebase, CI polling,
merge, and evaluator spawn are all quota-bucket or free). Quota buckets: sonnet (controller session
+ evaluator sub-agent).

**docs-4 unchanged**: still `[IN-SPRINT]`, design-ready, held on D-3 (unanswered). Not re-picked —
Standing rule 1 (one backlog item per iteration) and D-3 is a judgment park, not a capacity one; no
predicate to re-run, only Mark can answer it.

Full record: `design_docs/docs-mission-log.md` §ITERATION 7.
