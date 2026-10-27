PR #1937 is merged; **iteration 59 recovery remains incomplete** pending the full outside-sandbox launchd result, remaining done-gate evidence, and fresh independent evaluation. This response is the planning deliverable. No files, Git state, messages, tickets, or remote services were changed; no driver gates or executor dispatch ran.

Required planner lane: `codex:gpt-6.1-sol`. Runtime model identity was not exposed locally; retain the controller’s actual launch provenance rather than treating this label as verification.

1. **Keep the existing contract and plan path.**

   Retain `design_docs/planned/sprint-plan-stall-own-session-transcript.md` and append a recovery/completion section. Both JSON fields, `design_doc` and `markdown_plan`, point there, as does the released entry at `changelogs/v0.32-current.md:80`. Relocation would create unnecessary link changes. Create no duplicate design or sprint.

   The appended section should distinguish:

   - Implementation merged through #1937: head `da3726052d3a6eb34e6245baffdb99ac49eb6810`, merge `6c16b4ce8c961e2c3afc9cb4ba766d09cde4496f`.
   - Recovery inspected at `13092528c8ae406c78054f58846b00572edceeae`.
   - Recovery completion time, captured only when recovery actually finishes.
   - Release v0.54.18 consolidated the fragment through `329c9808fee16fca9846eaaea0f569ebb5af4469`. Do not recreate the deleted fragment or add a second changelog entry.

   Local read-only comparison confirms the driver, test, plan, and sprint JSON are unchanged between the candidate head and this worktree’s HEAD.

2. **Bank evidence before editing completion metadata.**

   Copy the original judge report **verbatim** from the supplied judge worktree into:

   `design_docs/fleet-mission-evidence/iteration59/evaluation.md`

   Its measured SHA-256 is:

   ```text
   2ecd4c2d1a293805bd65b7558a4718ff995f217139b22f2b26039bf5549174ab
   ```

   Verify the banked copy against that digest. Preserve its original snapshot time, pending-CI statement, `CODE_ACCEPT`, and stated score of 90.

   Put amendments in separate `controller-corrections.md`. Specifically:

   - The rubric rows total **100**; the narrated five-point deduction would yield **95**, not 90. Record the inconsistency without replacing the judge’s score.
   - Its sandbox result was 49 passing / 1 failing and cannot establish D7 completion.
   - Its blanket “no file exceeds 800 lines” claim is inaccurate: the unchanged candidate/current driver has 2,691 lines; the test has 641.
   - Later controller observations supersede the historical CI snapshot only as separately attributed evidence.

   Bank the supplied controller evidence separately with SHA, observation time, run/check identifiers, and raw-log references: exact-merge CI and Build success, all 17 checks successful, and fresh stall **50/0**. These are controller observations, **not planner measurements**. The full launchd rerun remains pending.

3. **Use the recovered original numstat for LOC.**

   Read-only command:

   ```text
   git diff --numstat 763d336f18208035b78595f53e0c0dac0491c7b6..da3726052d3a6eb34e6245baffdb99ac49eb6810
   ```

   | Original path | Added | Deleted |
   |---|---:|---:|
   | Sprint JSON | 58 | 0 |
   | Changelog fragment | 5 | 0 |
   | Sprint plan | 82 | 0 |
   | `tools/launchd/mission-control.sh` | 44 | 15 |
   | `tools/launchd/test_mission_stall.sh` | 202 | 11 |
   | **Total** | **391** | **26** |

   Set the sprint’s actual LOC metric to **272: driver plus test additions + deletions**, excluding documentation and metadata. Record that definition in `features[0].notes`. For comparison, original total PR churn is **417**, net growth **365**; including the changelog but excluding planning artifacts gives **277** churn. Recovery documentation must not inflate the original implementation measurement.

4. **Complete the existing acceptance contract outside the sandbox.**

   The controller should finish the already-running full launchd rerun and retain its terminal exit code and complete log. Recover trustworthy evidence for the original four commands, or obtain missing results outside the sandbox:

   ```text
   /bin/bash tools/launchd/test_mission_stall.sh
   bash -n tools/launchd/mission-control.sh
   /bin/bash -n tools/launchd/test_mission_stall.sh
   make test-launchd-drivers
   ```

   Also establish the fleet done-gate: correct running surface, reach through merged pinned dev, healthy **and** degraded dry-runs pinned with `AILANG_DRIVER_PINNED=<validated-full-sha>`, and no mid-iteration reload. Record the actual profile, SHA, commands, times, and outcomes. Use existing valid logs where available; do not infer missing dry-runs from CI success.

   Before any outside-sandbox rerun, inspect external writes. Observe real shared artifacts by hash/count, permitting absence; never seed or truncate them. Rehearse potentially external-writing steps under an isolated synthetic root.

   Any socket-, HOME-, or live-kernel-dependent sandbox result is **UNINFORMATIVE UNDER SANDBOX**. An acceptance command already red on the untouched base is a **finding to report**: stop completion, preserve evidence, and introduce no workaround, weakened assertion, or source fix under this recovery.

5. **Require fresh independent review before resolution.**

   Have a fresh independent evaluator review the exact recovered state, D1–D7 evidence, original mutations, corrections, and proposed metadata changes. Preserve its substantive verdict separately as:

   `design_docs/fleet-mission-evidence/iteration59/recovery-evaluation.md`

   Record actual model/provider, inspected SHA, limitations, and reported usage; use “not reported” for unavailable usage. This planner performs no dispatch. The original MiniMax verdict and controller corrections do not substitute for that fresh review.

   Stop if review is unavailable, raises blockers, or relies on unresolved gates. If concurrent work changes the relevant blobs, re-establish identity and review applicability before proceeding.

6. **Finalize only the existing sprint and ticket.**

   After done-gates and independent acceptance, update `.ailang/state/sprints/sprint_M-FLEET-STALL-OWN-TRANSCRIPT.json`:

   | Field | Final value |
   |---|---|
   | `status` | `"completed"` |
   | `features[0].status` | `"completed"` |
   | `features[0].passes` | `true` |
   | `features[0].actual_loc` | `272` |
   | `velocity.actual_total_loc` | `272` |
   | `velocity.actual_days` | `null` |
   | `velocity.actual_loc_per_day` | `null` |
   | `velocity.actual_milestones_per_week` | `null` |
   | `features[0].started` | Retain `null`; historical start unmeasured |
   | `features[0].completed`, `last_checkpoint` | Actual UTC recovery-completion timestamp |
   | `features[0].notes`, `last_session` | Merge/recovery distinction, metric definition, evidence links, unmeasured effort |

   Preserve IDs, `created`, estimates, acceptance criteria, dependencies, registry decision, and both stable plan references. Do not backdate completion to the merge timestamp or use elapsed calendar time as measured effort.

   Resolve only the existing signature, after the gates and independent review:

   ```text
   ailang mission ticket resolve stall:watchdog:foreground-agent-and-wrong-transcript --resolution <evidence-backed-summary> --sha 6c16b4ce8c961e2c3afc9cb4ba766d09cde4496f
   ```

   Verify the resulting resolution and queue count. Then inspect #1929’s current state, existing comment count, and comment bodies. Reuse an existing adequate supersession comment; otherwise add one linking #1937 and its merge/evidence, verify its ID and count change, then close #1929 and verify closure. Do not assume zero comments or duplicate prior actions.

7. **Record iteration 59 once and publish narrow bookkeeping.**

   Append the recovery to `design_docs/fleet-mission-log.md`; refresh `fleet-mission.md` and `fleet-mission-dashboard.md`. Record the post-merge capacity stop, historical merge time, actual recovery-completion time, independent verdict, gate results, ticket resolution, and old-PR disposition. Preserve earlier records and unchanged decision rows.

   Regenerate `fleet-mission-index.md` using the existing index-only workflow with `AILANG_MISSION_REGISTRY` pointing explicitly to the record worktree. Apply existing STATUS rotation rules losslessly where necessary. Publish only the reviewed metadata/evidence changes through the normal guarded process. Take no second ticket.

The existing `registry_reuse` action **`none`** remains applicable: this is internal shell instrumentation and bookkeeping, with no package capability or dependency change. Registry network searches are inapplicable. No `.ail` is written, so teaching-prompt, AILANG syntax/contracts/effects/inline-test, stdlib, and example gates are also inapplicable. Source, routing policy, shared binaries, and live mission configuration remain outside this recovery plan.