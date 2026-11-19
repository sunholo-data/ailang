# Iteration 102 idle judge (independent, read-only)

VERDICT: NONE

## Inputs check
- cmp tickets.json prev_tickets.json: identical. cmp open-decisions.tsv prev_open.tsv: identical (28 OPEN rows). 0 allowlisted directives; ledger unchanged.
- Gating claims spot-checked against charter.md ledger text (D-FLEET rows at lines ~160+; Guardrails "Ticket-driven only" / "screen by step").

## Per-ticket (27)
| signature | gate / status | evidence |
|---|---|---|
| reporting:daneel-reports-not-declared-triage | gated D-FLEET-22 (prod coordinator config, out of scope) | row A = attended config + revision roll |
| quorum:gpt6-1-sol-openai-api-credits-exhausted | gated D-FLEET-20 (design_quorum.go routing/billing, HD-2a) | row text |
| messaging:pubsub-notify-wrong-project | gated D-FLEET-19 (rig gcloud ADC, attended) | row A |
| ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | gated D-FLEET-18 (tools/eval/ widen; guard arm already drafted #2110) | row 18 |
| eval:motoko-connection-probe-orphaned | gated D-FLEET-18 (item 58) | charter line ~329 |
| ci:motoko-probe-refusing-live-path-unexpected-success | gated D-FLEET-18 class (item 19) | charter line ~253 |
| policy:fleet-rulings-not-propagated-to-charters-and-skill | gated D-FLEET-25 (mechanism choice; drafts #2082, #2165) | row 25 |
| codex:server-capacity-error-crashes-finished-iteration | gated D-FLEET-26; diff exists, draft #2065 | row 26 |
| quota:admission-has-no-headroom-margin-and-flaps | gated D-FLEET-32 (ask 3 = quota policy, HD-2a) | row 32 |
| pi:unowned-dirty-absolute-vs-relative-paths | gated D-FLEET-33 (.pi/extensions, D-FLEET-3); diff in evidence/iteration66 | row 33 |
| skill:design-doc-creator-grep-portability | gated D-FLEET-36 (scope widen); arm drafted #2108 | row 36 |
| skills:agents-copies-stale-non-mission | gated D-FLEET-38; arm drafted #2108 | row 38 |
| pi:openrouter:uncapped-max-tokens-402 | gated D-FLEET-35 (out-of-scope files, owner V1/attended) | row 35 |
| stall:watchdog:slow-on-hung-tool | gated D-FLEET-34 (ask 2 contradicts D-FLEET-4/-7; rec B waive) | row 34 |
| crash:sprint-executor:sigbus-ram-tmp-full-make-test | gated D-FLEET-23 (terraform/coordinator) | row 23 |
| sprint-evaluator:worktree-cap-and-exit1 | gated D-FLEET-37 (coordinator, out of scope) | row 37 |
| executor:3h-timeout-from-queue-and-unsliced-sprints | gated D-FLEET-40 (coordinator) | row 40 |
| executor:timeout-loses-work-and-may-never-start | gated D-FLEET-40 | row 40 |
| coordinator:cloud-lane-capped-by-local-worktree-limit | gated D-FLEET-37/40 | rows |
| executor:codex-credential-lease-collision-fails-concurrent-dispatch | gated D-FLEET-39 (coordinator) | row 39 |
| coordinator:job-retry-success-dropped-after-failed-completion | gated D-FLEET-39 | row 39 |
| executor:codex-credential-collision-drops-automatic-planner-handoffs | gated D-FLEET-39 | row 39 |
| quorum:oc-glm-5-3-output-cap-invalid | gated D-FLEET-31 (rec A = no further change) | row 31 |
| quorum:claude-p-ration-usage-429 | gated D-FLEET-28 (design_quorum.go Blocked(), HD-2a, out of scope) | row 28 |
| ci:distinct-build-test-failures | gated D-FLEET-17 class (item 44); non-required Build reds | charter line ~312 |
| tool-hang:triage-judge:home-metadata-search | gated/parked D-FLEET-44 (design, PR #2171) | charter line ~333 |
| skill:idle-record-barred-on-degraded-no-pick-fire | gated D-FLEET-45 (design blocked twice by quorum) | row 45 |

Buildable now: none. Mechanical-fix class: none found; every ticket's fix is policy class, out of scope, or already has a diff.

## Changes since 101 (64 commits, 8dc92cb52..2a7cee7bf)
- Ticket set and OPEN rows unchanged. No dev commit touches tools/launchd/, tools/eval/, .pi/, mission-control skill resources or scripts/*mission*.
- scripts/hooks/* (204a04e8f installer, f2a9807e2 push-bound/pre-push gates): attended auto-push hooks; no ticket concerns them.
- 5e1484ea1 / ccd80d1d9 / 8ca4d6ec3 (release ci-gate, 90-min poll): release workflow, not Gate 3b. D-FLEET-21's cap (1800 s, draft #2067) is a different gate; premise unchanged, but its re-measurement (tail up to 99 min) is consistent.
- d7ab21338/78e70be97 (ollama local wire types, empty-stream error): no open ticket cites ollama.
- downstream-check (0ad69ae3a etc.), doc-sample gate (701bac931...), types/let-generalisation, ingest hub token, mcp_tools, connectors docs: no ticket link.
- Premise note for D-FLEET-17: dev CI at 2a7cee7bf is green (CI, Build and Release success); the cited git-exec site now uses gitexec.CommandContext and env-vars.md changed. The "required checks red since 10-13" premise looks stale. That un-gates no ticket (D-17 gates only landing), but the controller can note it; confirming it and flipping the row is Mark's ledger edit, not a build.
- Docs-pages deploy had one failed run (13:53) then success (14:05) on the same SHA: transient.

## Screen by step
Candidates examined (in-scope, recommended option begins with an in-scope step, no diff): 
- D-FLEET-43 A (lift fleet idle-fingerprint exclusion): code in tools/launchd/lib/idle-fingerprint.sh, no diff, but no ticket backs it (controller observation); row itself says "built only if you choose A". Fails "backed by a queued ticket"; drafting would be self-sourcing (same ground as iteration 101). Excluded.
- D-FLEET-42 (3 drafts #2141/#2143/#2150), 18 (#2110), 21 (#2067), 25 (#2082, #2165), 26 (#2065), 33 (evidence diff), 36/38 (#2108): diffs exist.
- D-FLEET-29 B (driver builds binary per fire): rec is A (attended quick-install); B is a ruling-needed policy change; no ticket.
- D-FLEET-32 A / 34 A: policy class, non-recommended or design-only.
- D-FLEET-35, 28, 39-41, 19, 20, 22, 23, 31, 37: recommended step is out of scope (attended/V1/coordinator/terraform).
Result: no qualifying row.

## Verdict
VERDICT: NONE

## Strongest objection to own verdict
D-FLEET-43 A is genuinely code in fleet scope with no diff, and repeated all-gated fires burn ~66k judge tokens each; a literal reading of "screen by step" could pick it. Rebuttal: the screen's stated justification is that the ticket is already queued, and D-FLEET-43 has no ticket; its own text defers building to Mark's choice. The charter does not state this exclusion outright (wording gap, as iteration 101's judge also noted). Second: D-FLEET-17's premise looks stale now that dev is green, which could justify a re-measure, but that is a ledger/status question for Mark, not a buildable ticket step.

Confidence: 88%.
