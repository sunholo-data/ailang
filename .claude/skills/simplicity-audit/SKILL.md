---
name: simplicity-audit
description: Weekly codebase-complexity audit — re-measures the v1.0.0 simplicity gates (language-core closure, command and flag counts, os.Getenv routes, duplicate symbol names across packages, packages without a doc comment, skill trees, instruction-surface bytes, tracked files, root clutter), diffs against the last banked snapshot, and names what got more complex. Use when the user says "simplicity audit", "weekly audit", "is the codebase getting more complex", "find duplicate implementations", "codebase health check", or before a release. NOT for file sizes (codebase-organizer) or Sonar issues (sonarcloud-triage).
---

# Simplicity Audit

The 800-line file gate cannot see the thing that actually makes this repo hard for an
agent: **count and duplication**. Two implementations of one concept, a sixth backend
switch, a package with no comment, a command with no `--help` — none of those move a
file over 800 lines. This skill measures them and diffs week over week.

It is the standing instrument for
[M-V1-SIMPLIFICATION-PROGRAM](../../../design_docs/planned/m-v1-simplification-program.md):
the program closes the gaps once; this skill keeps them closed.

## Run it

```bash
make simplicity-audit          # full: re-measures, diffs, banks, exit 2 on regression
make simplicity-audit-fast     # same without the timed make test-core row
```

Under the hood: `.claude/skills/simplicity-audit/scripts/audit.sh` runs
`tools/simplicity_metrics.sh`, compares the JSON with the newest snapshot under
`.ailang/state/simplicity/`, prints a before/now/delta table, then the concrete names
behind the counts (which symbols are duplicated where, which packages lack a comment,
which platform packages the language core reaches, which skill cites a command that
does not exist), banks today's snapshot, and exits 2 if any gated metric moved the
wrong way. Snapshots are tracked in git so the diff works on any machine.

`scripts/dup_symbols.sh [min]` on its own lists every top-level function name declared
in `min` or more packages (default 3), with the packages. That list is where duplicate
implementations hide — `truncate` ×5 with three semantics, `writeJSON` ×4,
`setProcessGroup` ×4 were all found this way.

## What to do with the output

1. **A `WORSE` row on a gated metric** is a regression against the program's release
   gate. Find the commit: `git log --since=<last snapshot date> -- <the area>` and either
   revert the route or route it through the survivor the program names (one config
   package, one plane switch, one loader per concept).
2. **A new duplicate symbol name** is not automatically wrong (`NewClient` per provider is
   honest). Open the two definitions; if they do the same job, the one with more callers
   survives and the other becomes a call to it. Cite `git log -S` in the commit, per
   `.claude/rules/coding-standards.md` — never delete on "unused" alone.
3. **A package without a comment** gets a three-line `doc.go`: what it is, what it is not,
   which confusable sibling to use instead.
4. **A skill citing a missing command** is a stale instruction an agent will follow;
   fix the citation the same day.
5. **Closure growth** (`closure_internal_packages`, `closure_platform_packages`,
   `closure_leak_roots`) is caught earlier and harder by `internal/diag/closure_test.go`,
   which fails CI. If the audit shows growth that test did not catch, the test's root
   list in `tools/simplicity_metrics.sh` is stale — fix the list, not the number.

Report to the user in this shape: the delta table, the ranked "new since last audit"
names, and one recommended action per regression. Do not fix during the audit unless
asked; the audit is the observe step.

## Weekly schedule

The skill is meant to run every Monday alongside the Dependabot batch. Two ways:

- **Claude Code routine** (cloud): `/schedule` with the prompt
  `run the simplicity-audit skill and report regressions` on `0 9 * * 1`.
- **Rig launchd**: a `dev.ailang.simplicity-audit` job is a mission-loop-change task, not
  a copy-paste — go through that skill so the plist inherits the memory gate and the
  HOLD protocol.

Either way the run must **commit the new snapshot** (`.ailang/state/simplicity/<date>.json`)
or the next week's diff is against a stale base.

## Positive control

Before trusting a green run, prove the instrument sees a positive once per quarter:

```bash
printf 'package diag\nimport "os"\nvar _ = os.Getenv("AILANG_PLANTED")\n' > internal/diag/planted.go
make simplicity-audit-fast    # must report getenv_outside_config WORSE and exit 2
rm internal/diag/planted.go
git checkout -- .ailang/state/simplicity/   # discard the planted snapshot
```

## Boundaries

- This skill measures; the program's sprints change code. Route findings into a sprint
  via `sprint-planner`, not into ad-hoc edits during the audit.
- `codebase-organizer` owns file size; `sonarcloud-triage` owns Sonar; `docs-sync` owns
  the docs site. Overlap is deliberate zero.
- The metrics script is bash 3.2 and BSD-grep safe (the rig). Any edit to it must keep
  `shellcheck -S warning` clean and must not use `grep -q` on a pipe under `pipefail`
  (SIGPIPE turns a hit into a miss) or BRE `\|` alternation (BSD grep has none).
