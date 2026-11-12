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
bash .claude/skills/simplicity-audit/scripts/audit.sh --fast --offline    # no network: external rows null
bash .claude/skills/simplicity-audit/scripts/audit.sh --fast --services   # + Cloud Run usage (minutes)
tools/simplicity_external.sh [--services]   # just the external rows, as JSON
```

Under the hood: `.claude/skills/simplicity-audit/scripts/audit.sh` runs
`tools/simplicity_metrics.sh`, compares the JSON with the newest snapshot under
`.ailang/state/simplicity/`, prints a before/now/delta table, then the concrete names
behind the counts (which symbols are duplicated where, which packages lack a comment,
which platform packages the language core reaches, which skill cites a command that
does not exist), banks today's snapshot, and exits 2 if any gated metric moved the
wrong way. Snapshots are tracked in git so the diff works on any machine.

## Health rows beyond count and duplication

Added 2026-11-12 so the audit tracks what
[HANDOVER-v1-codebase-health-2026-11-09](../../../design_docs/planned/HANDOVER-v1-codebase-health-2026-11-09.md)
§2 measured by hand:

| Row | Source | Gated? |
|---|---|---|
| `unreachable_funcs` | `go run golang.org/x/tools/cmd/deadcode@v0.28.0 -test=false ./cmd/...`, `third_party/` excluded (counted apart as `unreachable_funcs_third_party`); detail = top packages | `le`, ratchet at 1,301 (2026-11-12) |
| `prod_loc_core` / `prod_loc_platform` / `prod_loc_platform_pct` | tracked non-test Go under `internal/`, split by the core list **read from** `.claude/rules/architecture.md` | no (informational) |
| `sonar_gate_passing`, `sonar_{security,reliability,maintainability}_rating` | SonarCloud public API (`api/qualitygates/project_status`, `api/measures/component`); ratings 1=A .. 5=E | no |
| `codeql_high_open`, `dependabot_open_{critical,high,medium,low}`, `secret_scanning_enabled` | `gh api` code-scanning (CodeQL, high+critical), Dependabot alerts, repo `security_and_analysis` | no |
| `cloud_run_services_used` | `--services` only: `gcloud logging read` of 30 days of 2xx request logs per Cloud Run service in prod/dev/test, `--account m@sunholo.com`, read-only; per-service request classes (otlp, health, static, `/api/<x>`, `/<x>`) and distinct IPs in `.services` | no |

The external rows live in `tools/simplicity_external.sh` and are **optional and
offline-safe**: no network, no `gh` credential, no `gcloud`, or `--offline` records
`value: null` with a `reason` (listed under "unmeasured" and in the what-to-look-at
block), never 0 and never a failed run. A 0 means the API answered "none". A crash
of that script is a bug and does fail the run. `--services` takes minutes (the dev
dashboard alone is ~86k requests a month); a service at the 250k-entry ceiling is
marked capped.

`unreachable_funcs` counts test-only helpers as dead and cannot see reflection or
data coupling: it tracks **direction**, it is not a deletion list. Each removal needs
`deadcode -test=true` and `git log -S` (`.claude/rules/coding-standards.md`). It
only sees packages linked into a `cmd/` binary — a func in a package no binary
imports is invisible to it, not "reachable".

In the diff, a row absent from the old snapshot shows `—` before and `?` as the
verdict, and a null on either side is `?`; neither can be a regression.

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

For `unreachable_funcs`, plant in a package a binary links (`internal/diag` is not
linked by any `cmd/`, so a plant there is invisible to deadcode):

```bash
printf 'package types\nfunc plantedDeadA() int { return 1 }\nfunc plantedDeadB() int { return plantedDeadA() }\n' > internal/types/zz_planted_dead.go
tools/simplicity_metrics.sh --no-test --offline --json --out /tmp/plant.json >/dev/null
jq .metrics.unreachable_funcs.value /tmp/plant.json   # must be 2 above the last snapshot
rm internal/types/zz_planted_dead.go
```

For the external rows, prove the null path: `HTTPS_PROXY=http://127.0.0.1:9
tools/simplicity_external.sh` must exit 0 with every Sonar and GitHub row null and
a `reason` naming the connect failure.

## Boundaries

- This skill measures; the program's sprints change code. Route findings into a sprint
  via `sprint-planner`, not into ad-hoc edits during the audit.
- `codebase-organizer` owns file size; `sonarcloud-triage` owns Sonar triage (this
  audit only records the gate and ratings week over week); `docs-sync` owns the docs
  site. Overlap is deliberate zero.
- The metrics script is bash 3.2 and BSD-grep safe (the rig). Any edit to it must keep
  `shellcheck -S warning` clean and must not use `grep -q` on a pipe under `pipefail`
  (SIGPIPE turns a hit into a miss) or BRE `\|` alternation (BSD grep has none).
