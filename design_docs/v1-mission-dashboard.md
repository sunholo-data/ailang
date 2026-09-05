# V1 Mission Dashboard — snapshot, overwritten every iteration

**Updated**: 2026-09-04 (iteration 327) · **Release**: v0.35.0 · **dev**: GREEN

## Where the goal stands
**N = 12 design docs remaining before v1.0.0** — unmoved this iteration (HARNESS work).

## Last iteration (327) — LANDED [HARNESS]
All five follow-ups on the committed-Go auto-push gate: harness gets its own `HOME`; both formatting
gates now read gofmt's **exit code** (a zero-byte `.go` cleared both before); NUL-delimited pathnames
(a committed `café.go` used to strand a clean push); the two earliest guards log rather than exit
silently; a **scoped** ShellCheck CI gate with its own mutation control. Harness 18 → 26 arms.
PR #1044 → `da2b6689b`, 20 checks / 0 failures. Judge sonnet PASS 85/100.

**The finding — it cost 92 lines of fleet evidence.** M1's test row was specified as a "caller
sentinel": seed a line into the caller's REAL `~/.ailang/state/autopush.log`, then assert it is
unchanged. That write is DENIED under the codex sandbox, so the executor's run looked correct; the
first out-of-sandbox re-run truncated the shared log 92 → 1, unrecoverably. **A sandboxed executor
cannot tell "my destructive step was denied" from "my step was harmless", so any acceptance step
writing outside the worktree is unverifiable on that lane by construction.** The arm is now an
observation, never a seeded write.

## Next picks
1. `m-release-manager-skill-split` — standing head: 18-image walkthrough out of
   `release-manager/SKILL.md`; ratchet `check-context-docs` back down 625 → 596.
2. `m-gate-wiring-classifier-prefix-blind` — NEW, systemic half of the judge's blocking finding:
   `gate_wiring_test.go:130` classifies gates by `check-`/`test-check-` prefixes, so it cannot see
   `fmt-check`, `shellcheck-autopush` or either new self-test.
3. `m-acceptance-criterion-green-at-base` — pre-registered; instance 2 is the skill-edit trigger.

## Loop cadence + routing
Controller `claude:claude-opus-5` · planner + executor `codex:gpt-5.6-sol` (probe rc=0) · evaluator
`sonnet`, own worktree. Designer did not run (no doc owed); rotation pointer
`pi:ollama/deepseek-v4-flash:0731-cloud`, untouched. **The spawn-pin hook fired and was right**:
`resolve-role-spawn.sh planner` (no doc) answered `agent-tool opus`, the hook DENIED it because the
planner is env-pinned to codex. Filed as `m-resolver-hook-disagree-on-docless-pick`.

## Parked on Mark
**Nothing for V1** — ledger 54 rows, **0 OPEN**; no directives outstanding on #972. Two items that are
not V1 decisions: `mission-world`'s approval **D-WORLD-31**, open since 2026-09-03T15:12Z; and **PR
#1041** (884 insertions of Go), blocked ONLY on a base-inherited `lint` red iteration 326 already fixed
— a rebase clears it. V1 left it alone: unattributable to any mission, so it looks attended.

## Cost + sore spots
metered **$0.00** of the $5 ceiling (codex + sonnet are quota buckets; no quorum ran); tripwire CLEAN.
**SonarCloud red on dev and unowned** (inherited; it passed on #1044). `m-ci-serial-gate-masking`: one
early red in a long sequential job still hides every gate behind it.
