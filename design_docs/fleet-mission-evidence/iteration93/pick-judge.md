# Fleet iteration 93 — independent idle-screen judge (R2)

**VERDICT: BUILD — D-FLEET-25 option A's undrafted mechanism step.** As an **unmerged proposed diff**: (1) design doc for
one fleet-wide rulings ledger every charter references and cannot override (quorum → plan → judge); (2) the
`ailang mission doctor` check reporting charter/skill text contradicting a RESOLVED fleet-wide ruling
(`internal/mission/doctor.go`, `cmd/ailang/mission_cmd.go`). The A/B-shared World-fork correction is **already drafted**
(#2082 = `a58e08e68` + `399cf799b`) and must not be rebuilt. No other OPEN row admits an undrafted in-scope step.

Judge: pi, provider/model **openrouter/minimax/minimax-m3**. Reviewed SHA **a3bb8629106a4b8459df5aa33f867b4ed4dd84a5**,
worktree `/tmp/fleet-i93-evaluator`. Sources: `design_docs/fleet-mission.md` (26 OPEN rows, Authority, Guardrails),
mission-control SKILL + `resources/gate-2-pick.md` (fleet branch) + `gate-5-retro.md`, `sprint-evaluator/SKILL.md`,
controller facts `iteration93/{tickets,drafts,dfleet25-*}-input.json` + `input-provenance.md`, `git show` of all 8 draft
heads. Provenance kept: the sandbox cannot reach api.github.com, so ticket/draft/PR states are the controller's captures,
not my re-verification. No sibling role reports, older judge files or the iteration-92 judge were read.
**Code tests and mutation drills: N/A** — no code change is under review.

## 1. All-26-OPEN-row screen (by STEP, not by row)

- **D-16** NONE: A's first step = attended rig action outside the repo; driver arms = Phase 3b routing; no shared step.
- **D-17** NONE: A = attended fixes; B = authority grant needing Mark; no shared step.
- **D-18** NONE: guard arm drafted (#2110); probe work awaits ruling + arm landing.
- **D-19** NONE: A = attended rig op; B = `messages_send.go` out of scope.
- **D-20** NONE: A = `design_quorum.go` (out of scope, HD-2a); B = no code.
- **D-21** NONE: A drafted (#2067, both mirrors + rule-6 clause); B shares no step with A.
- **D-22** NONE: A = prod coordinator config (multivac) + `internal/coordinator/**`; B = attended; recurrence 18× (last 11-07T00:04:58Z) adds no in-scope step.
- **D-23** NONE: A = `coordinator_cloud_task.go` (out of scope); B = authority grant needing Mark.
- **D-25** **BUILD** — A's mechanism step; see §3.
- **D-26** NONE: A drafted (#2065); other half is "a separate design"; "World fork needs a port" stale (de-forked 09-06).
- **D-27** NONE: A (recommended) = keep luna, already shipped; B opposite; no shared step.
- **D-28** NONE: A = `design_quorum.go` (out of scope, HD-2a); B = env one-off.
- **D-29** NONE: A = attended `make quick-install`; B = driver change that "needs your ruling"; no shared step.
- **D-31** NONE: A (recommended) = no change; B = seating policy (HD-2a); C out of scope.
- **D-32** NONE: recommended B = waiver, no diff; A = quota-policy design (HD-2a).
- **D-33** NONE: recorded diff exists (iter 66); `.pi/extensions/**` parks for Mark (D-3); B non-recommended.
- **D-34** NONE: A re-opens D-4/-7 thresholds; B (recommended) = waive + resolve, no code.
- **D-35** NONE: A's paths outside allowlist; B = D-11/13 policy.
- **D-36** NONE: guard arm drafted (#2108, `bcf7f97ac`); the 8-site edit awaits that landing (iter-16 rule).
- **D-37** NONE: A needs your confirmation of ask 2's home; B (widen coordinator) "not recommended".
- **D-38** NONE: guard arm drafted (#2108, `261fb3b64`); sync extension follows only after it lands.
- **D-39** NONE: all fix paths (`internal/coordinator/**`, `credentiallease/**`) out of scope; A/A2/B attended or V1.
- **D-40** NONE: A = coordinator shutdown fix (out of scope); B = attended design; C no change.
- **D-41** NONE: A = Dockerfile (out of scope); B (skills, in scope) non-recommended, no shared step; ticket absent.
- **D-42** NONE: "every buildable step of A is now a draft" (#2141/#2143/#2150); rest = ruling, host line, rotation, redaction.
- **D-43** NONE: "built as a proposed diff only if you choose A"; no ticket in capture — queue precondition fails.

## 2. Ticket map (25 live tickets → rows)

reporting:daneel-reports→D-22, quorum:gpt6-1-sol→D-20, messaging:pubsub-notify→D-19, ci:launchd-suite-flakes→D-18,
eval:motoko-probe-orphaned→D-18, **policy:fleet-rulings-not-propagated→D-25**, codex:server-capacity→D-26,
quota:admission-headroom→D-32, pi:unowned-dirty→D-33, skill:design-doc-creator-grep→D-36,
skills:agents-copies-stale→D-38, pi:openrouter:uncapped-max-tokens→D-35, stall:watchdog→D-34,
crash:sprint-executor:sigbus→D-23, sprint-evaluator:worktree-cap→D-37, executor:3h-timeout→D-40,
executor:timeout-loses-work→D-40, coordinator:cloud-lane-capped→D-37/D-40 (no row of its own),
executor:codex-credential-lease→D-39, coordinator:job-retry-success-dropped→D-39,
executor:codex-credential-collision→D-39, ci:motoko-probe-refusing-live-path→D-18 (no row of its own),
quorum:oc-glm-5-3→D-31, quorum:claude-p-ration→D-28, ci:distinct-build-test-failures→D-17.
Seven OPEN rows have no ticket in the capture (D-16, D-21, D-27, D-29, D-41, D-42, D-43); each was screened on its own
gates above. Only D-25 is live with an undrafted in-scope step.

## 3. Authority reasoning for D-FLEET-25

Exact wording: A = "a design doc (quorum, plan, judge) for one fleet-wide rulings ledger that every charter links to and
cannot override, plus an `ailang mission doctor` check that reports charter or skill text contradicting a RESOLVED
fleet-wide ruling. The fleet also corrects the World fork claim…"; B = only the World fork claim; C = no change; default C.
`git show 399cf799b`/`a58e08e68` proves #2082 holds **only** the shared mechanical sub-item (8 files: `missions/world.toml`,
both `mission-loop-change` mirrors, script/test comments, changelog) — no ledger, no doctor check. A retains undrafted steps.

1. **Step test.** "Screen by step, not by row … a row qualifies when its recommended option … begins with an in-scope step
   that has no diff … This holds even when the row's eventual target is outside fleet scope." A begins with an undrafted
   in-scope step (the design doc); one step's diff does not disqualify the row, and D-42's M2/M3 precedent ("the row-level
   reading would have idled") confirms later steps of a recommended option stay admissible.
2. **Scope.** `design_docs/**` is a support path admitted by the pre-push guard (`_scope_is_fleet_support`);
   `internal/mission/**` and `cmd/ailang/mission*` are harness paths (`_scope_is_harness`). The charter-link sub-step
   touches other missions' charters (path-admitted but cross-mission governance): keep it in the unmerged draft, flagged for
   Mark — it does not block the two in-scope steps.
3. **Ruling.** "Default if unanswered: C" governs the row's resolution, not the draft-build the guardrail exists to perform
   ("parks with the proposed diff, done rather than deferred"). Precedents: #2065, #2067, #2108, #2110, #2141/#2143/#2150
   were built under defaults C/B. D-25 carries **no** build-blocking condition — unlike D-43.
4. **Sequencing.** A's first deliverable is the design doc (quorum → plan → judge); the doctor check follows. One backlog
   item (Standing rule 1).

## 4. Strongest objection to my own verdict

*A's first deliverable is a design doc, not code; the guardrail admits "the … row whose recommendation is code … and build
that diff", and the code half presupposes a ledger design that does not exist — so on a strict reading D-25 offers no
buildable code step and should idle.* It does not change the verdict: the guardrail's operative test is "an in-scope step
that has no diff" (the design doc qualifies), the fleet's own proposed diff #2141 ships a design doc
(`design_docs/planned/m-fleet-secrets-scope-per-lane.md`), and the guardrail tolerates a target outside fleet scope. The
objection narrows the step (design doc first, doctor check second) rather than defeating it. Weaker second objection: a row
note cannot re-impose a gate over the charter's adopted default, so D-43 might qualify — I answer that D-43's sentence is
a diff-specific condition, not a default; the specific instruction controls, and D-43 also fails the queue precondition.
Recorded as ambiguity, not invented authority.

## 5. Limitations

- No network: ticket/draft/PR states are the controller's captures; I verified only the 8 draft heads as local git objects.
- I did not audit other missions' charters for the "every charter links to" sub-step; amendments there stay unmerged, flagged for Mark.
- Ambiguities recorded, not invented: (a) whether "recommendation is code" admits a design-first option's first step — resolved in favour of admission (step test + #2141 precedent); (b) D-43's diff-specific condition vs the general fallback — specific controls; (c) D-26's stale "World fork needs a port" line.
- Code tests, mutants, done-gate: **N/A** (no code change under review).
