# Fleet iteration 137 independent judge

## Q1. The 3 new tickets (all filed by triage 55, 2026-11-24 16:06Z, blocking none)
Guard: tools/launchd/githooks/pre-push:14-60. `_scope_is_harness` and `_scope_is_fleet_support` are explicit allowlists, and line 76 refuses any path in neither list. In support, only `make/test.mk` is a make file; `make/code-health.mk` and `.github/**` are in neither. This matches the charter "Authority" prose (fleet-mission.md:75-97).

| Signature | Fix lands in | In Authority? | Begins with an in-scope step with no diff? | Open decision row? |
|---|---|---|---|---|
| ci:verify-gap-map-workflow-wiring | `.github/workflows/ci.yml` or an operator-only exemption in `internal/cihygiene` (the test `TestVerifyTargetsAreWiredIntoAWorkflow`); target at make/code-health.mk:349 | NO: CI workflow, `internal/cihygiene`, `make/code-health.mk` all refused | No. The ticket itself says "confirm with the target owner" (V1 / M-VERIFY-SOUNDNESS-COVERAGE, 538cef9d7); no in-scope precursor step | None; D-FLEET-48 names it only as an inherited red |
| ci:agy-pin-test-home-isolation | `internal/executor/agy/pin_test.go:36` (`t.Setenv("HOME", ...)` should become `testutil.SetHomeDir` from `internal/testutil/home.go`) | NO: `internal/executor/**`, `internal/testutil/**` refused | No. Single-file Go test edit, no scope-guard or harness step | None; D-FLEET-48 names it as an inherited red |
| ci:modelreg-qwen-slug-pricing-mismatch | `internal/modelreg/models.yml` (35b6eb7ce), and the ticket itself says correct vendor rates need independent owner evidence | NO: `internal/modelreg` refused; also a pricing/billing-class data decision, so policy-class parked | No | None; D-FLEET-48 names it |

Also note: the "make/test.mk" allowlist entry does not rescue any of them (verified by grep: the wiring is in code-health.mk and ci.yml:453). Lint's unused `vertexModelURL` at `internal/ai/gemini/generate.go:287` has no ticket, and fleet may not self-source one.
Result: 0 of 3 routable.

## Q2. The other 17 tickets
- Ticket diff (iteration137/ticket-diff.txt): "gone since 136: []", so none were resolved elsewhere. I re-read all 20 tickets.json rows; the 17 are the same signatures.
- Decision ledger (fleet-mission.md:149-202): OPEN rows are still exactly D-FLEET-44, 45, 46, 48. The branch diff against origin/dev changes only one ledger row (D-FLEET-48, added by the branch). Nothing flipped to RESOLVED. The charter's last touches on origin/dev were 07:55 and earlier on 11-24, and dev HEAD is `ea6d2f28f` with no charter commit since.
- Per-group: 
  - Eight attended-fixed tickets (quorum:gpt6-1-sol, pi:openrouter:uncapped, executor:codex x3 + coordinator:job-retry, quorum:claude-p-ration, executor:3h-timeout x2 and rows 54-56, 68) wait on D-FLEET-46 (OPEN, default B = no change) (fleet-mission.md:200). N.
  - skill:idle-record-barred... D-FLEET-45 OPEN, default B, design quorum-blocked twice (row 62). N.
  - tool-hang:triage-judge:home-metadata-search D-FLEET-44 OPEN, default B immediately (row 61). N.
  - eval:motoko-connection-probe-orphaned D-FLEET-18 class, folded, ticket stays open (row 58). N.
  - reporting:daneel-reports-not-declared-triage, skill:idle-record..., messaging:pubsub-notify-wrong-project, crash:sprint-executor:sigbus (row 23 RESOLVED/outside scope), ci:distinct-build-test-failures (Go/CI paths), publish:evidence-pack (BUILT, #2463) are held by their existing queue rows; no row changed state.
  - publish:evidence-pack:no-secret-value-guard is not routable as new work: it is already built and judged; only the landing is pending (Q3).
- Re-derived: ledger OPEN row set; ticket diff; pre-push allowlist; the 4 OPEN rows' defaults. Spot-checked only (iteration 134's judge established NONE, byte-identical set): the individual queue-row blockers for the old 16, since no ledger row changed.

## Q3. PR #2463
- Required contexts (gh api branches/dev/protection/required_status_checks): test, lint, build, docs-gate, checks. The API was readable (no 403).
- origin/dev HEAD ea6d2f28f at 2026-11-24 (same SHA as the evidence read): docs-gate success (x2), build success, **test FAILURE, checks FAILURE, lint FAILURE**. dev-ci.txt also shows govulncheck, test-windows, launchd drivers, Windows/macOS builds red. D-FLEET-48's default trigger ("dev's required checks green") is NOT met. Not triggered.
- PR head is now 48eae22 (the record stacked on the 66e0aae56 build), MERGEABLE. Its red required checks (test, lint, checks) have the same failure signatures as dev: lint annotation `func (*Client).vertexModelURL is unused` (generate.go:287); checks exit 2 from the home-isolation scan (agy pin_test.go:36); test red. The PR diff touches 0 `.go` files (git diff --name-only origin/dev...branch | grep -c '\.go$' gives 0; only scripts/, .agents mirrors, .claude/skills/mission-control, make/test.mk, design_docs, changelogs). So the PR's reds are inherited, not caused by it. The one dev red the PR fixes is `launchd drivers`/`TestQuorumArtifactShellSuite` (stale `.agents` sprint mirrors), per iteration 136. Caveat: I could not read raw job logs via the API (terminal escapes), so test-name-level equality rests on annotations plus iteration 136's gates.md.
- Answer: do NOT land #2463 this fire. Any merge would be over three red required contexts, which is an admin bypass the charter forbids (D-FLEET-48 option B is Mark's). Note dev gained the `ci:` tickets after 16d168f19 but dev's red set is the same three, and dev's new commits (agy cloud, verify, Inkling) added more reds (verify-gap-map wiring, slug pricing) not fewer.

## Q4. Verdict
**NONE** (confidence 90%).
- 3 new tickets: all outside Authority, with no in-scope first step.
- 17 old tickets: all ruling-gated (D-FLEET-44/45/46 OPEN, default B) or already built (#2463) or outside scope.
- #2463: blocked on dev's required test/lint/checks, unchanged. D-FLEET-48 stays OPEN; the ticket stays open.

What the controller is likely to get wrong:
1. Treating the three new `ci:` tickets as harness work because they are CI-flavoured, or because "mission triage filed them". The paths are Go, CI-workflow and modelreg, all refused by pre-push line 76. Screen-by-step does not help: there is no in-scope precursor.
2. Reading "docs-gate and build green" as "required checks green". The required set has 5 contexts and 3 are red on dev HEAD.
3. Landing #2463 by treating the PR's reds as "its own" or by an admin merge. They are inherited, but the default says land only once dev is green.
4. Double-counting: the new tickets duplicate the three inherited reds already recorded in D-FLEET-48; they must not be self-sourced into a new fix or into the lint red (no ticket).
5. Inflating the idle "slots lost" accounting for the new tickets (each is 1 slot, blocking none).
