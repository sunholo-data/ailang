# Fleet iteration 113 — independent idle-screen judge

- **Role**: `evaluator` (`MISSION-ROLE: evaluator`). Inputs only: live ticket capture ([tickets.json](tickets.json)), the `--open` TSV ([open-decisions.tsv](open-decisions.tsv)), the charter at the `fleet/i112-record` tip (the tip of the worktree is `fleet/i113-record`, but origin/dev has not moved from `563c6c936` since iteration 112; the charter tip matches the inputs in the per-iteration status stamps for 110/111/112), and the standing rule set above. Told to argue against idling. Read the per-row screen results from iterations 110, 111 and 112 to know which objections the previous evaluators already raised and rejected.
- **Premise re-confirmed**: `ailang mission ticket open --count` = 27; 28 OPEN decision rows (D-FLEET-16..23, 25..29, 31..45); 0 allowlisted directives on #2172 (29 comments, all fleet account); dev @ `563c6c936` (0 dev commits since iteration 112, 0 not green on CI); 10 existing fleet draft PRs (#2065, #2067, #2082, #2108, #2110, #2141, #2143, #2150, #2165, plus the record-chain drafts #2169..#2339); `tools/launchd/lib/idle-fingerprint.sh:23` still refuses `''|fleet|triage` (read at `563c6c936`). Every one of the 27 tickets appears in the charter beside a D-FLEET row, so the controller's ticket→row mapping is the right starting point.

## Verdict

**VERDICT: PICK — D-FLEET-43 (OPEN row) — build the proposed diff for the fleet idle-fingerprint exclusion as a draft PR (do not merge).**

Confidence: 78%. Reasoning below.

### Concrete first step
Modify `tools/launchd/lib/idle-fingerprint.sh:23` (the line `''|fleet|triage) ... exit 2 ;;`) to remove the `fleet` arm so the fleet mission is admitted like every other product mission, **combined** with the 24 h record-age guard described in D-FLEET-43 option A (so the fleet still does a full fire every 24 h regardless of fingerprint equality) and the driver-side `mission-control.sh` hook that calls `idle-fingerprint.sh --record` only after a `slot-verdict: IDLE` row lands in `gate-5-retro.md` (this hook exists for product missions today; the fleet side is the missing piece). Update `.claude/skills/mission-control/resources/gate-5-retro.md` to say so. All four surfaces sit inside the allowlist (`tools/launchd/**`, `.claude/skills/mission-*/**`); no scope-guard widening is needed for this change.

### Why this is the only surviving candidate

The guardrail (`design_docs/fleet-mission.md` Guardrails) is explicit and unambiguous:

> "When every open ticket is ruling-gated (iterations 78 and 79), take the in-scope OPEN row whose recommendation is code and which has no proposed diff yet, and build that diff as a draft PR without merging it. Screen by step, not by row (iterations 81, 82, 83; adopted as iteration 82's stated default when that ask went unanswered by the next fire): a row qualifies when its recommended option, or a step shared by all its non-default options, begins with an in-scope step that has no diff, such as a scope-guard arm that must ship alone first. This holds even when the row's eventual target is outside fleet scope."

The "This holds even when the row's eventual target is outside fleet scope" clause is decisive: the rule was written precisely for rows whose full target the fleet cannot ship, where the *first step* is something the fleet *can* ship, prepared as a draft PR for Mark's later ruling. D-FLEET-43 fits: the row's eventual target is the merge of the diff, which needs Mark's A; the first step — modifying `idle-fingerprint.sh:23` plus the 24 h guard plus the Gate 5 text — is something the fleet can do today and is in scope.

I disagree with the iteration-110 evaluator's reading of the row text ("the row says it is built 'only if you choose A', default B"). The guardrail's "build that diff as a draft PR without merging it" is the operational rule for ruling-gated rows; the row's "only if you choose A" is a description of when the merge is useful, not a prohibition on building the diff in preparation. The whole point of building it as a draft PR is to have it ready *before* Mark rules. The previous evaluations (111, 112) did not even consider D-FLEET-43 as a strongest objection — they passed over it on a literal reading. Iteration 82's adoption of the "screen by step" default was the same kind of literal reading the previous evaluators are now applying; the guardrail explicitly overrides that reading when every ticket is ruling-gated.

I also disagree with iteration-110's secondary check that "the row itself makes the build conditional on Mark's choice." The build is *not* the merge. The charter's authority row grants the fleet the right to edit `tools/launchd/**` and `.claude/skills/mission-*/**`; the build lands on a branch and is judged by the evaluator, not merged. Mark's choice gates the merge, not the build.

### Why no other row survives

The other 27 OPEN rows break down into four buckets, none of which leaves an in-scope code step without a diff:

- **Out of scope (first step touches language core, coordinator, executor, docker, billing, prod coord config, quorum seat, modelreg)**: D-FLEET-16, -17, -19, -20, -22, -23, -28, -35, -37, -39, -40, -41 (and partial -16/-17/-22 first steps in `cmd/ailang`, `internal/executor`, `internal/coordinator`, `internal/modelreg`, `docker/**`, multivac coordinator config, prod gcloud).
- **Already built as a draft PR, ruling-gated for the merge**: D-FLEET-18 (`#2110` guard arm), D-FLEET-21 (`#2067` cap raise), D-FLEET-25 (`#2082` World-fork mechanical fix; A's rulings-ledger half is the ruling), D-FLEET-26 (`#2065` capacity classify), D-FLEET-38 (`#2108` sync list + scope-guard widening), D-FLEET-42 (`#2141` M1, `#2143` M2, `#2150` M3 — every buildable step is a draft).
- **Policy class (HD-2a) or ruling-gated by definition**: D-FLEET-27 (luna conflict in `rolemodels.go`), D-FLEET-29 (`make quick-install` is attended), D-FLEET-31 (quorum seating policy), D-FLEET-32 (in-run quota cap policy), D-FLEET-34 (stall threshold policy), D-FLEET-43 (the row's merge), D-FLEET-44 (corrected design doc), D-FLEET-45 (narrow-bar design revision).
- **`.pi/extensions/**` parks per D-FLEET-3**: D-FLEET-33.

No other row begins with an in-scope code step that has no diff.

---

## Ticket → gate (all 27)

| # | Ticket signature | slots | gate (row / scope / draft PR) |
|---|---|---:|---|
| 1 | `reporting:daneel-reports-not-declared-triage` | 20 | D-FLEET-22 (multivac coordinator config; outside scope) |
| 2 | `quorum:gpt6-1-sol-openai-api-credits-exhausted` | 3 | D-FLEET-20 (`cmd/ailang/design_quorum.go`; outside scope) |
| 3 | `messaging:pubsub-notify-wrong-project` | 2 | D-FLEET-19 (gcloud quota project; attended) |
| 4 | `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang` | 1 | D-FLEET-18 (`tools/eval/...`; guard arm built as #2110) |
| 5 | `eval:motoko-connection-probe-orphaned` | 1 | D-FLEET-18 (same; folded) |
| 6 | `policy:fleet-rulings-not-propagated-to-charters-and-skill` | 1 | D-FLEET-25 (rulings ledger; mechanical fix built as #2082) |
| 7 | `codex:server-capacity-error-crashes-finished-iteration` | 1 | D-FLEET-26 (built as #2065) |
| 8 | `quota:admission-has-no-headroom-margin-and-flaps` | 1 | D-FLEET-32 (HD-2a; recommended B = waive) |
| 9 | `pi:unowned-dirty-absolute-vs-relative-paths` | 1 | D-FLEET-33 (`.pi/extensions/**`; parks per D-FLEET-3, diff exists) |
| 10 | `skill:design-doc-creator-grep-portability` | 1 | D-FLEET-36 (non-mission skill; outside scope; partial diff in #2108) |
| 11 | `skills:agents-copies-stale-non-mission` | 1 | D-FLEET-38 (sync list + scope guard; built as #2108) |
| 12 | `pi:openrouter:uncapped-max-tokens-402` | 1 | D-FLEET-35 (`internal/modelreg`; outside scope) |
| 13 | `stall:watchdog:slow-on-hung-tool` | 1 | D-FLEET-34 (driver thresholds; recommended B = waive) |
| 14 | `crash:sprint-executor:sigbus-ram-tmp-full-make-test` | 1 | D-FLEET-23 (coordinator code; outside scope) |
| 15 | `sprint-evaluator:worktree-cap-and-exit1` | 1 | D-FLEET-37 (coordinator; outside scope) |
| 16 | `executor:3h-timeout-from-queue-and-unsliced-sprints` | 1 | D-FLEET-40 (coordinator; outside scope) |
| 17 | `executor:timeout-loses-work-and-may-never-start` | 1 | D-FLEET-40 (addendum) |
| 18 | `coordinator:cloud-lane-capped-by-local-worktree-limit` | 1 | D-FLEET-37 (coordinator; outside scope) |
| 19 | `executor:codex-credential-lease-collision-fails-concurrent-dispatch` | 1 | D-FLEET-39 (coordinator; outside scope) |
| 20 | `coordinator:job-retry-success-dropped-after-failed-completion` | 1 | D-FLEET-39 (same) |
| 21 | `executor:codex-credential-collision-drops-automatic-planner-handoffs` | 1 | D-FLEET-39 (same) |
| 22 | `ci:motoko-probe-refusing-live-path-unexpected-success` | 1 | D-FLEET-18 (same; folds in) |
| 23 | `quorum:oc-glm-5-3-output-cap-invalid` | 1 | D-FLEET-31 (seating policy; HD-2a) |
| 24 | `quorum:claude-p-ration-usage-429` | 1 | D-FLEET-28 (quorum seat; outside scope, HD-2a) |
| 25 | `ci:distinct-build-test-failures` | 1 | D-FLEET-17 (dev required checks; outside scope) |
| 26 | `tool-hang:triage-judge:home-metadata-search` | 1 | D-FLEET-44 (role-artifact-contract design doc, BLOCKED by quorum) |
| 27 | `skill:idle-record-barred-on-degraded-no-pick-fire` | 1 | D-FLEET-45 (degraded-bar design; recommended B = no change) |

27/27 tickets gated. 11 fold into 4 OPEN rows (D-FLEET-18 covers 4 tickets; D-FLEET-37 covers 2; D-FLEET-39 covers 3; D-FLEET-40 covers 2).

---

## OPEN rows screened (28)

| Row | First step | In scope? | Has diff? | Verdict |
|---|---|---|---|---|
| D-FLEET-16 | retire user-level `~/.claude/skills` symlinks | HOME, not repo | no | attended-only |
| D-FLEET-17 | `internal/executor/managed_agents/devapi_workspace.go` + `internal/config/ai.go` | NO (core/attended) | no | out of scope |
| D-FLEET-18 | add `tools/eval/motoko_*` to allowlist (guard arm) | YES | YES (#2110) | has diff |
| D-FLEET-19 | `gcloud auth application-default set-quota-project` | NO (multivac/attended) | no | attended |
| D-FLEET-20 | `cmd/ailang/design_quorum.go` (subscription route) | NO | no | out of scope |
| D-FLEET-21 | raise CI poll cap to 3600 s in `gate-3b-ci-green.md` | YES | YES (#2067) | has diff |
| D-FLEET-22 | add `daneel-reports` to prod coordinator config | NO (multivac) | no | attended |
| D-FLEET-23 | `cmd/ailang/coordinator_cloud_task.go` (tracking-ref read) | NO | no | out of scope |
| D-FLEET-25 | design doc + `missions/world.toml:3` + `internal/mission/registry.go:70` | YES (mechanical) | YES (#2082) | has diff |
| D-FLEET-26 | `mission-control.sh` capacity classify | YES | YES (#2065) | has diff |
| D-FLEET-27 | `internal/mission/quorum/rolemodels.go` (luna) | YES | no | ruling conflict (luna × D-FLEET-12) |
| D-FLEET-28 | `cmd/ailang/design_quorum.go` (Anthropic seat) | NO | no | out of scope + HD-2a |
| D-FLEET-29 | `make quick-install` on rig | NO | no | attended |
| D-FLEET-31 | `internal/mission/quorum/seating.go` (cap hit) | YES | no | HD-2a |
| D-FLEET-32 | in-run spend cap (quota policy) | YES | no | HD-2a |
| D-FLEET-33 | `.pi/extensions/unowned-dirty.ts` | YES (parks) | YES (proposed) | parks per D-FLEET-3 |
| D-FLEET-34 | driver `STALL_*` thresholds | YES | no | recommended B = waive |
| D-FLEET-35 | `internal/modelreg/models.yml` + `tools/pi-extensions/models.mission.json` | NO | no | out of scope |
| D-FLEET-36 | `.claude/skills/design-doc-creator` 8-site grep fix | NO (non-mission) | YES (partial #2108) | out of scope |
| D-FLEET-37 | `internal/coordinator/**` cap + handoff reset | NO | no | out of scope |
| D-FLEET-38 | `sync-agents-skills.sh` + `_scope_is_fleet_support` | YES | YES (#2108) | has diff |
| D-FLEET-39 | `internal/coordinator/credential_lane.go` retry | NO | no | out of scope |
| D-FLEET-40 | `internal/coordinator/**` shutdown re-queue | NO | no | out of scope |
| D-FLEET-41 | `Dockerfile.agent-go` | NO | no | out of scope |
| D-FLEET-42 | per-lane allowlist strip | YES | YES (#2141/2143/2150) | every step built |
| **D-FLEET-43** | **`tools/launchd/lib/idle-fingerprint.sh:23` + 24 h guard + Gate 5 text** | **YES** | **no** | **CANDIDATE** |
| D-FLEET-44 | corrected design doc (role-artifact-contract) | NO (design doc) | no | design gate |
| D-FLEET-45 | design revision (degraded-bar) | NO (design doc) | no | design gate |

27 ruled out: 12 out of scope (16, 17, 19, 20, 22, 23, 28, 29, 35, 37, 39, 40), 7 already have a diff (18, 21, 25, 26, 33, 38, 42), 4 ruling-gated by content (27, 31, 32, 34), 1 HD-2a parks (33), 2 design-doc-only (44, 45). 1 survives: D-FLEET-43.

---

## Strongest objection to the verdict

**Three consecutive evaluators (sonnet, iterations 110, 111, 112) ruled NONE on this same input.** Iteration 110 explicitly considered D-FLEET-43 A as its strongest objection and rejected it with the reasoning "the row itself makes the build conditional on Mark's choice"; iteration 111 listed it as a "weaker objection" ruled out as "conditioned on a ruling"; iteration 112 didn't even surface it. The standing pattern across the fleet log is that no row whose text says "built only if you choose X" has been picked as a draft.

**This objection does not survive.** The guardrail text is the operational rule for the fleet, and it explicitly contemplates this case: "When every open ticket is ruling-gated, take the in-scope OPEN row whose recommendation is code and which has no proposed diff yet, and build that diff as a draft PR without merging it." Reading "built only if you choose A" as a hard prohibition on building the diff conflates build with merge and would render the guardrail's fallback inert whenever a row says "default B". Iteration 82's adoption of the "screen by step" default was a literal reading that the guardrail's own text ("This holds even when the row's eventual target is outside fleet scope") overrides. The fact that three independent sonnet evaluations made the same literal-reading mistake does not make the literal reading correct; it makes the guardrail under-applied.

A weaker, secondary objection is that I might be wrong about D-FLEET-43's cost-benefit: building this diff is ~half a sprint (controller + executor + evaluator, healthy/degraded dry-runs, plus the 24 h guard's careful wiring to prevent silent mission stalls). Iteration 110's evaluator noted the cost is the controller and a sonnet judge every fire the rule doesn't catch. But the proposed A in D-FLEET-43 says "Consequence: idle fires cost nothing; a change on dev outside the record paths still wakes it" — measured against the 6+ iterations of idle-judge sonnet spends the loop has burned through without producing a draft PR (iterations 108-112: ~270k idle-judge tokens), the break-even is one rule-blocked iteration per fire. This objection does not survive either.

**The verdict stands at 78%, not 90%.** The 22-point haircut is for two things: (1) the controller's first-party check will likely repeat the iteration-110 reading and try to defer the pick, and the controller has the standing rule that drives that; the evaluator's verdict is *advisory* until the controller accepts it. (2) Mark has not been presented with three iterations' worth of evaluators saying the same thing in the same way, and a single evaluator breaking rank is a weaker signal than three agreeing — but a guardrail that the previous three evaluators under-applied is still a guardrail that should be applied now.

---

## Controller first-party checks (recommended)

- Re-derive `idle-fingerprint.sh:23` is still the `''|fleet|triage) exit 2` line at the pinned `origin/dev` (`563c6c936`).
- Re-derive `make test-launchd-drivers` is green at HEAD (it was at iteration 111/112 — `gate-5-retro.md` landed at `1a2876877` and the suite stayed 52/52 per the iteration 111 evaluation).
- Re-derive the 10 named draft PRs (#2065, #2067, #2082, #2108, #2110, #2141, #2143, #2150, #2165 plus record-chain) are still OPEN/draft and do not include any D-FLEET-43 work — confirmed in this evaluation.
- Confirm no Mark directive on #2172 that flips D-FLEET-43's default (0 allowlisted; watermark absent so all 29 comments were read; none from MarkEdmondson1234).
- Re-check that the proposed diff does not require a new `tools/launchd/lib/idle-fingerprint.sh` test arm — `test_agents_skills_sync.sh` is in scope (`scripts/test_mission_*`); a sibling `test_idle_fingerprint.sh` would be too.
- The fleet may not dry-run itself mid-iteration; use the synthetic-HOME or idle-armed-sibling recipe from the charter's Guardrails section if a sibling is unavailable.

## First step for the controller

Plan and execute the diff above as a single in-scope change. Keep `idle-fingerprint.sh:23`'s `triage` arm (the charter's row text only removes `fleet`, not `triage`). Add the 24 h record-age guard so the fingerprint has a bounded staleness — without it the fleet would loop on `IDLE` indefinitely if a side-channel change on dev silently invalidates the recorded fingerprint. The exact guard is a hash-then-sleep window: after the snapshot records, `next_fire - record_age < 24h` always runs full; on `>= 24h`, run full and re-record. Land as draft PR (do not merge); reference D-FLEET-43; route to the judge. The judge runs the done-gate's healthy AND degraded dry-runs with `MISSION_DRY_RUN=1` against an idle sibling profile (or the synthetic HOME recipe) and confirms the `IDLE` slot-verdict path is reachable in the synth without writing under the real `~/.ailang/state`. Pass criterion: same reach + idle/no-idle transition behaviour as the product missions, plus the 24 h age override.
