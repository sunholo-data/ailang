# Iteration 69 — done-gate evidence

Ticket: `gate0:session-protocol-deadlock:agent-policy-unset` (stapledon, filed 2026-10-07).
Branch `fleet/i69-pi-controller-protocol-note`; base `65253e86e68be251eecd796d92327e4d4828c234`; head `fc646ca2543dc66e5515db2d6594c4aff7dd305d`.

## Reality check (controller, first-party)
- `~/.pi/agent/sessions/--Users-voightkampff-dev-sunholo-data-stapledons-godot--/`: 40 session files; 11 contain a `session_protocol_ack` call (planner recount; the controller's first count of 12 was wrong). 5 of the 11 got at least one `REFUSED: session protocol prerequisites not met`: 09-29T12-53 (deadlocked: 0 bash `ailang messages` calls, 5 `ailang_cli` calls, including `{"argv":["messages","list","--unread","--json"]}`, every one refused with `AILANG_AGENT_POLICY is unset`), and 10-02, 10-09 ×2 and 10-19 (each recovered later via bash).
- `.pi/extensions/session-protocol-gate.ts` `prerequisitesMet()` counts the inbox step only from a `bash` call whose command contains `ailang messages`. `.pi/extensions/ailang-exec.ts` registers `ailang_cli` with no policy and refuses every call. Its "You have NO shell" lane prompt is injected only when a policy is attached (comment, measured 2026-09-19), which refutes ask 1: exporting a policy would inject that prompt into a controller that needs bash.
- Ask 2 (let the ack pass without the policy) and any extension text edit are `.pi/extensions/**` changes, which park under D-FLEET-3.

## Executor gates (claude-haiku-5-5, reported; the judge re-ran the suites)
- `make test-launchd-drivers`: base rc 0 (51 suites). Head, first run: rc 2, `tools/eval/test_motoko_connection_probe.sh` "host too slow … instrument failure" (load-sensitive, outside fleet scope); that suite alone rc 0 (59/59); full re-run rc 0 (52 suites).
- New `tools/launchd/test_pi_controller_note.sh`: 28 passed. Ten mutations (M1–M10 + capacity cross-checks) all red; M8 is the T6b control.
- shellcheck: new files clean; the driver holds the same 30 findings as base. `check-changelog`, `check-file-sizes` and `git diff --check` all rc 0.
- Executor instrument stall: its first drill used bash 3.2 `read -d ''` plus `${s//…}` on the 164 KB driver, and ran 17 min at about 82% CPU. The controller killed that one process at 13:23:59Z; the executor rewrote the drill with perl and finished. The executor cap was extended from 30 to about 47 min (launch 12:55:04Z, rc file by 13:41Z) because the implementation was already committed.

## Controller dry-runs (world: idle, armed; real HOME; `AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED=fc646ca25…`)
- Healthy: rc 0, `DRY RUN ok: mission=world`, `lanes=DEGRADED(2)`. Codex is over its daily ration this fire (probe rc 75), so planner → opus and executor → `claude:claude-haiku-5-5`, as at iteration 66.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i69`): rc 0, `codex model 'bogus-model-i69' unusable: quota-limited`, executor → `claude:claude-haiku-5-5`, `DRY RUN ok`.
- The worktree-sweep log hash (`f26d4f9674a96a3f`) and the pin listing hash (`54060050452dcdd5`) were unchanged; no world pidfile was written.

## Real pi accepts the flag (controller probe, flat-rate)
- `pi --model ollama/glm-5.3:cloud --no-extensions --no-tools --no-session --append-system-prompt "OVERRIDE: … reply with exactly the single word PERSIMMON …" -p "say hello"` → `PERSIMMON`, rc 0.
- Control (same command without the flag) → `Hello! 👋 I'm your coding assistant …`, rc 0.
- So on the route that deadlocked, the system-prompt append reaches the model with `-p`. Not measured: a resumed `--session-id` session, and the glm-5.3-flash route that drops the system role (the user-prompt copy covers that).

## Controller hardening after the judge (commits `9a9c95e58`, `57add5d09`)
- Round 1 (PASS 91) showed three meaning-level mutations passing T6: F ("NOT granted" → "granted"), F2 (inbox step "via the ailang_cli tool") and H (chain rule inverted). `9a9c95e58` adds T6c, which pins five load-bearing sentences and refuses `(via|with|through|using|use) (the )?ailang_(cli|run)`. It also drops "single" from "bash runs only single read-only commands", the judge's looseness finding.
- Round 2 (PASS 93) found N5 ("Prefer to call ailang_cli for the inbox.") and N7 ("every call succeeds") still passing. Both reopen the bug. `57add5d09` pins "every call refuses" and fails if either tool is named outside the one NOT-granted sentence.
- Controller drills (perl on the lib, `cmp` restore): F, F2, F2b (all keywords kept, an ailang_cli route added), H, N3 (in the note sentence), N5 and N7 are all red. The suite passes 36/36 after restore. The judge did not re-score `57add5d09`; it implements the fix the round-2 judge named, and these drills are its evidence.
- Still unpinned (round 2): N1 ("skip session_protocol_ack"), N2 ("last, in any order") and N6 ("read CLAUDE.md with the bash tool"). Prose correctness beyond the pinned sentences is by review only.
- Drill instrument note: the first N3 drill read "missed" because perl's non-global `s///` hit the lib's header comment, which carries the same phrase. Aimed at the note sentence, N3 is red.
