# Mission Dashboard — Motoko

_Snapshot, overwritten each iteration. History: `motoko-mission.md` (STATUS) + `motoko-mission-log.md`.
Written at iteration 25, 2026-08-26._

## Where we are

- **Landed (iter 25)**: the launchd **driver pin had been refusing unconditionally** — `~/.claude.json`'s
  `hasCompletedProjectOnboarding` no longer exists (**0 of 15** live entries; control
  `hasTrustDialogAccepted` **15 of 15**), so every fire of every mission ran unverified working-tree code.
  PR #923 → `ff0da7445`, evaluator **PASS 98/100 zero blocking**, Gate 3b GREEN. Suite 35 → 53 arms.
- **In flight**: nothing.
- **Next**: row **6h** — a provider failure arrives as a *successful empty completion* (`#842`); the guard
  the reporter suggests is not expressible against the current value type, so step 1 is making absence
  observable. Then 6i · 6j · 7.

## Loop health

- **This fire ran UNPINNED — and that is what the pick fixed.** After #923, simulation against the real
  `~/.claude.json`: motoko **PIN-OK**, v1 **PIN-OK**, world **REFUSE-TO-PIN** (a *true* verdict — that
  clone carries neither key). The next fire tests whether pinning actually resumes.
- Source clone: **152 behind**, 0 ahead, 0 dirty. Running skill resolves to V1's checkout,
  byte-identical to `origin/dev`; this checkout's copy is 139 lines short (delta read before the gates).
- **dev CI**: required contexts green. One standing non-required red, `SonarCloud`, inherited from V1's
  commits — 56.9% coverage on new code, plus a **new** second condition, B security rating. Handed to V1.
- **Filed, not picked** (row 6j): `launchd drivers (bash 3.2)` arm 33 is intermittent — 3 non-success of
  58 runs today, all on unrelated commits, all at that arm; ~1s locally against a 120s cap on the runner.

## Routing / cost

controller `claude:claude-opus-5` · executor `codex:gpt-5.6-sol` (1 run, no fallback) · evaluator
`sonnet`, own worktree, generator≠judge holds. Designer rotation at `claude:claude-fable-5`, **Fable
unspent**. Metered **$0.00** of $5. No GPU.

## Parked on Mark

- **`D-MOTOKO-WORKDIR-2` (OPEN, 4th ask)** — grant *standing* authorization to reconcile the source clone
  to `origin/dev` unattended when three predicates hold (ahead == 0; every dirty file's added lines
  byte-present upstream or byte-identical; a sha256-verified backup re-verifies after)? One word **yes**
  or **no**. Today all three hold trivially: 0 ahead, 0 dirty.
- Phase-0 rows 10/11/12 stay parked on upstream `arniwesth/motoko_agent#154`, re-measured **OPEN**
  (control `#175` MERGED). A predicate re-run each iteration, not a human ask.
