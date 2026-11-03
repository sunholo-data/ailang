# Iteration 81 — evaluations (sonnet, Agent tool)

## Gate-2 idle-verdict judge: REFUTED (narrow)

The controller claimed that no ticket was routable: all 25 open tickets map to OPEN ruling-gated rows, and every in-scope code recommendation already has a diff.

The judge confirmed C1 (25 open tickets, newest first_seen 10-28), C2 (every ticket maps; the two signature misses are tracked under D-FLEET-18 in the queue), C3 (no directive on #2039, no charter commit since `99ef89c26`, no comment on #2065/#2067) and C5 (dev green at `b23cadced`, SonarCloud the only red).

It refuted C4. D-FLEET-25 options A and B both contain a mechanical, in-scope (`missions/**`) sub-item, the stale `missions/world.toml` "WORLD IS A FORK" header, and no diff existed for it. The controller verified this first-party: the World plist runs the shared driver, and ailang-world has no driver copy.

## Diff judge, round 1: PASS-WITH-EDITS 90 (`a58e08e68`)

- Premise holds: plist, no ailang-world driver, `golden_live_test.go:367`, `doctor.go:281`.
- The old snippet's grep counts 0 against both the live and the base driver. The new snippet reads YES for 7/7 plists under /bin/bash, and a fake plist pointing at a non-sourcing script reads NO.
- Every Go, sh and toml change is a comment. Mirror `cmp` matches and `--check` is rc 0. `go test ./internal/mission/...`, `check-context-docs`, `check-skills` and `check-changelog` are all rc 0.
- Mutants:
  - M1 (revert the `.agents` mirror): killed by `sync-agents-skills.sh --check`.
  - M2 (old `$REPO` pattern): killed, all 7 read NO.
  - M3: not applicable, because no test guards the skill text.
- Required edit: `scripts/test_mission_directives.sh:104` still said "(world fork depends on it)".
- Strongest objection: nothing mechanical guards the snippet's regex against the real driver, so it can rot again.

## Diff judge, round 2: PASS 94 (`399cf799b`)

The required edit is applied, and the diff is one comment line. `test_mission_directives.sh` 33/33. A widened residual grep across the Authority allowlist found only dated or historical hits. The snippet-guard objection is non-blocking: a follow-up candidate, and scope creep for this draft.
