1. **Scope and ruling gate.** Base is clean at `aebc3dcbb`, branch `fleet/i83-dfleet18-scope-arm`. Build a standalone **DRAFT PR**, pending Mark’s ruling on `D-FLEET-18`; default C remains no change. Modify exactly four files: the hook, its scope tests, Authority, and the fragment below. Touch **no `tools/eval/**` files**, leave the decision OPEN, and resolve neither ticket. This session made no edits or git writes.

2. **Exact hook change.** In `pre-push`, insert an independent case arm between existing lines 32–33. Before:

   ```bash
       internal/config/mission.go|make/test.mk) return 0 ;;
       tools/pi-extensions/sandbox/*) return 0 ;;   # the pi sandbox fail-closed fix (D-FLEET-6); the eval harness shares the rest of tools/pi-extensions
   ```

   After:

   ```bash
       internal/config/mission.go|make/test.mk) return 0 ;;
       tools/eval/motoko_connection_probe.sh|tools/eval/test_motoko_connection_probe.sh) return 0 ;;   # support, not harness (D-FLEET-18)
       tools/pi-extensions/sandbox/*) return 0 ;;   # the pi sandbox fail-closed fix (D-FLEET-6); the eval harness shares the rest of tools/pi-extensions
   ```

   Literal alternatives are Bash 3.2 compatible and admit exactly two paths. Preserve `_scope_is_harness` at lines 14–23 and `_scope_verdict` at lines 50–63. Putting these paths in harness would deny their original owner, motoko. Keep the Authority pointer comment at lines 12–13 unchanged.

   | Path | Fleet before → after | v1 before → after | Motoko before → after |
   |---|---|---|---|
   | `tools/eval/motoko_connection_probe.sh` | refuse → allow | allow → allow | allow → allow |
   | `tools/eval/test_motoko_connection_probe.sh` | refuse → allow | allow → allow | allow → allow |

   Before verdicts were measured by sourcing the hook with `MISSION_SCOPE_LIB=1`; after verdicts follow from the proposed arm.

3. **Exact test additions.** Insert before `test_mission_scope_guard.sh:53`, using its existing `check` helper at lines 17–20:

   ```bash
   # D-FLEET-18: exact support paths; product loops retain access.
   check fleet tools/eval/motoko_connection_probe.sh allow # kills: drop path 1
   check fleet tools/eval/test_motoko_connection_probe.sh allow # kills: drop path 2
   check motoko tools/eval/motoko_connection_probe.sh allow # kills: move path 1 to _scope_is_harness
   check motoko tools/eval/test_motoko_connection_probe.sh allow # kills: move path 2 to _scope_is_harness
   check v1 tools/eval/motoko_connection_probe.sh allow # kills: move path 1 to _scope_is_harness
   check v1 tools/eval/test_motoko_connection_probe.sh allow # kills: move path 2 to _scope_is_harness
   check fleet tools/eval/qwen38-motoko-rerun.sh refuse # kills: widen to tools/eval/*
   check fleet tools/eval/motoko_connection_probe.sh.bak refuse # kills: path 1 suffix glob
   check fleet tools/eval/test_motoko_connection_probe.sh-x refuse # kills: path 2 suffix glob
   check fleet tools/eval/xmotoko_connection_probe.sh refuse # kills: path 1 basename-prefix glob
   ```

   `git ls-files tools/eval | head` confirms `qwen38-motoko-rerun.sh` exists at HEAD. Each named mutation has a specific rejecting arm; product arms cover moving either path individually or the entire pair.

4. **Exact Authority change.** At `fleet-mission.md:86`, before:

   ```markdown
     `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
     `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
   ```

   After:

   ```markdown
     `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
     `tools/eval/motoko_connection_probe.sh`, `tools/eval/test_motoko_connection_probe.sh` (D-FLEET-18),
     `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
   ```

   This is proposed Authority text in a draft, not a recorded ruling. `design_docs/fleet-mission*` is already harness scope (`pre-push:20`), so the pinned guard admits this documentation change.

5. **Allowlist mirror audit.** Searched all requested roots with `rg -n --hidden '_scope_is_fleet_support|tools/pi-extensions/sandbox|D-FLEET-6'`. Only the hook implements this allowlist; no additional mirror requires updating.

   | File/site | Action |
   |---|---|
   | `tools/launchd/githooks/pre-push:29,33,56` | Change support case only |
   | `tools/launchd/test_mission_scope_guard.sh:50,52` | Add tests above |
   | `design_docs/fleet-mission.md:86–87` | Change Authority above |
   | `tools/launchd/test_mission_pi_run_sandbox.sh:87` | None; sandbox fixture |
   | `tools/launchd/test_pi_ext_args.sh:57` | None; extension argument assertion |
   | `scripts/mission_pi_run.sh:148,176–177` | None; sandbox dependency wiring |
   | `.claude/skills/mission-control/resources/gate-3-route.md:363,376,381` | None; sandbox invocation/docs |
   | `.agents/skills/mission-control/resources/gate-3-route.md:363,376,381` | None; corresponding mirror |
   | `make/test.mk:41` | None; extension tests |
   | `make/test.mk:128,140–141` | None; probe suite/syntax wiring |
   | `scripts/test_check_referenced_paths.sh:45–46` | None; tracked-path fixture |

   Zero matches in `internal/mission`, `cmd/ailang/mission*`, and `docs` were paired with positive controls at `internal/mission/ticket.go:1`, `cmd/ailang/mission_ticket.go:1`, and `docs/sprint-retros/motoko-iter39-residual-audit.md:1`. The search expression itself has positive controls at `pre-push:29,33`.

6. **Exact changelog fragment.** Create `changelogs/unreleased/2026-11-04-fleet-scope-arm-dfleet18.md:1`, following `changelogs/unreleased/README.md:3–12`:

   ```markdown
   ### Changed — Fleet scope guard: two Motoko probe support paths (PROPOSED, D-FLEET-18 A)

   PROPOSED, pending Mark's ruling on D-FLEET-18. The fleet support allowlist admits exactly `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh`; product missions retain access, and other `tools/eval/` paths remain refused. Guard arm only, with Authority documentation and regression tests. No probe changes or flake fixes are included; those follow only after the ruling and this arm lands on the pinned `origin/dev` copy (iteration-16 rule). See `design_docs/fleet-mission.md`, D-FLEET-18.
   ```

7. **Gates at base AND proposed head, in this order.** Record each command’s exit status separately; controller runs the last outside the sandbox.

   | Command | Socket/write expectations |
   |---|---|
   | `bash -n tools/launchd/githooks/pre-push` | Neither |
   | `/bin/bash tools/launchd/test_mission_scope_guard.sh` | Temporary git writes outside tree at lines 56–77; local filesystem remote |
   | `scripts/mission_decisions.sh --check --file design_docs/fleet-mission.md` | Neither; read-only awk |
   | `make check-changelog` | Recipe reads only; macOS `make` attempted outside-tree cache writes |
   | `make check-context-docs` | Outside-tree `mktemp` writes (`scripts/check_context_docs.sh:47–50`) |
   | `make check-no-personal-email` | Bash here-document temp writes (`scripts/check_no_personal_email.sh:65,67`); make cache |
   | `git diff --check` | Neither; also check committed PR range with `git diff --check origin/dev...HEAD` |
   | `make test-launchd-drivers` | Temporary writes, git fixtures, socket tests; controller outside sandbox |

   Base syntax, ledger (40 rows), and `git diff --check` passed. Attempted changelog/email make gates are **UNINFORMATIVE UNDER SANDBOX**, despite green output: cache creation was denied, and email enumeration’s here-document failed. Scope integration, context, and full launchd gates were not run here. Head gates remain pending implementation. Any socket/outside-tree gate attempted under this sandbox must receive the same uninformative label, never pass/fail.

8. **Eval-harness exposure.** No probe-name references were found in `internal/eval*`, `cmd/ailang/eval*`, `Makefile`, `make/eval.mk`, or the four `tools/eval/qwen*.sh` runners. Positive controls: `internal/eval_harness/agent_runner_multi.go:105` selects an executor; `cmd/ailang/eval_suite.go:42` defines the command; `Makefile:69` includes eval targets; `make/eval.mk:19` invokes `eval-suite`; runners invoke it at `qwen36-stretch-frontier-followon.sh:32`, `qwen38-motoko-phase.sh:26,57`, `qwen38-motoko-rerun.sh:23`, and `qwen38-stretch-frontier.sh:15`. The additional executor search also returned no probe references, controlled by `internal/executor/motoko/motoko.go:36`.

   Actual callers are the launchd serial suite (`make/test.mk:128,148`), syntax checks (`:140–141`), and the self-test selecting its sibling (`tools/eval/test_motoko_connection_probe.sh:5`). `scripts/test_check_referenced_paths.sh:45–46` contains fixture text, not runtime execution.

   The direction matters: **the probe launches eval-suite**, at `tools/eval/motoko_connection_probe.sh:236`; eval runtime does not load the probe. The probe writes a connection artifact at `:298–310` and asserts routing at `:312–313`. Admitting these paths does **not** expose normal benchmark execution/scoring as D-FLEET-3’s shared extensions did. It **does** let fleet alter standalone probe evidence/verdicts when invoked, and its CI tests. The self-test’s socket arm is at `test_motoko_connection_probe.sh:612–646`.

9. **#2108 integration and risks.** Base this PR directly on `origin/dev`, never on #2108. Separate case and Authority lines keep conflict resolution local. If #2108 lands first, rebase onto updated dev, preserve its D-FLEET-36/-38 arms and Authority entries, retain this single literal pair and separate D-FLEET-18 line, preserve both test blocks, and rerun all gates. The fragment has a distinct filename.

   This changes no routing, lane order, quota, ration threshold, or billing guard. **It IS an authority widening**, hence ruling-gated. Risks are accidental glob expansion, denying product owners through harness classification, losing either PR’s entries during rebase, weakening probe evidence in a later change, and mistaking sandbox output for valid gate evidence. Keep the PR draft and unmerged until Mark rules A; after merge, the next fire must consume the updated pinned guard before either live flake signature is worked.