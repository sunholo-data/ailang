---
name: sonarcloud-triage
description: Triage SonarCloud issues and hotspots via the SonarCloud Web API — check gate status, list open BLOCKER/CRITICAL bugs + vulns, group hotspots by rule, bulk-mark hotspots Safe per rule, transition issues to False Positive, and change the New Code period. Use whenever the user mentions SonarCloud, Sonar, quality gate red/failing, code smells to triage, marking hotspots safe, marking issues as false positive, setting new code period, or asks "why is the Sonar gate red" — even if they only reference a specific rule like go:S2245 without naming SonarCloud.
---

# SonarCloud Triage

Repeatable triage of SonarCloud findings for `sunholo-data_ailang` via the Web API.
Everything the web UI does for triage is available through the API, and most of it
is faster from the shell.

## Prerequisites

- **`SONAR_PAT`** — personal admin token, exported in `~/.zshenv`. Get one at
  <https://sonarcloud.io/account/security>. Read-only scripts (`gate_status`,
  `fetch_*`) don't need it; mutating scripts (`mark_*`, `set_new_code`) do.
- Optional overrides: `SONAR_PROJECT` (default `sunholo-data_ailang`),
  `SONAR_ORG` (default `sunholo-data`), `SONAR_HOST` (default `https://sonarcloud.io`).

## Standard triage workflow

```
1. scripts/gate_status.sh           → see which conditions fail and on what metric
2. scripts/fetch_hotspots.sh        → group TO_REVIEW hotspots by rule + directory
3. For each rule listed in resources/known_fp_rules.md:
     scripts/mark_safe.sh RULE_KEY "<comment from known_fp_rules.md>"
4. scripts/fetch_issues.sh          → list remaining BLOCKER/CRITICAL bugs + vulns
5. For each issue that matches a known-FP row:
     scripts/mark_fp.sh ISSUE_KEY "<comment from known_fp_rules.md>"
6. scripts/set_new_code.sh 30       → rolling 30-day New Code window (if not already)
7. scripts/gate_status.sh           → confirm conditions moved
```

Read [`resources/known_fp_rules.md`](resources/known_fp_rules.md) before step 3 —
each entry has the rule key, verdict, and the exact comment to paste.
[`resources/triage_playbook.md`](resources/triage_playbook.md) has the "when you
see X in Y, do Z" reference.

## Scripts

All scripts live in `scripts/` and source `_lib.sh` for shared config.

| Script | Auth | What it does |
|--------|------|--------------|
| `gate_status.sh` | none | Summarize assigned gate, new-code period, condition pass/fail, top-level counts |
| `fetch_issues.sh [SEVERITIES]` | none | List open BUG + VULNERABILITY at given severities (default `BLOCKER,CRITICAL`) |
| `fetch_hotspots.sh [STATUS]` | none | Group hotspots by category/rule/directory (default `TO_REVIEW`) |
| `mark_safe.sh RULE_KEY "comment"` | `$SONAR_PAT` | Bulk mark all TO_REVIEW hotspots with that rule as REVIEWED+SAFE |
| `mark_fp.sh ISSUE_KEY "comment"` | `$SONAR_PAT` | Transition one issue to False Positive with audit-trail comment |
| `set_new_code.sh VALUE` | `$SONAR_PAT` | Set `sonar.leak.period` (number-of-days / `date:YYYY-MM-DD` / `previous_version` / version tag) |

## When to use this skill

Invoke whenever the user:
- Asks about the SonarCloud **quality gate** status or why it's failing
- Wants to **triage**, **clean up**, or **mark safe/false positive** SonarCloud findings
- Mentions a specific **rule key** (e.g. `go:S2245`, `javascript:S5852`) in context of Sonar
- Wants to change the **New Code period** (rolling days, since-date, previous version)
- Asks for a **summary of open bugs/vulns/hotspots** without opening the web UI
- Says something like "our Sonar is noisy" or "make the gate useful"

## Guardrails

- **Do not bulk-mark a rule as Safe without consulting `known_fp_rules.md` first.**
  That file is the source of truth for our standing decisions. If the rule isn't
  listed there, add it (with a `Review required` verdict) before acting.
- **SQL-injection hotspots (`go:S2077`) require a spot check** of 2–3 files before
  bulk-marking — we use dynamic table/column names with whitelisted values; we want
  to be sure format-string inputs never reach user content.
- `mark_safe.sh` only scans the first page (500 hotspots). If the rule has more,
  rerun until the count drops to zero.
- `mark_fp.sh` is per-issue; for bulk issue triage use the SonarCloud UI or loop
  in bash: `for k in K1 K2 K3; do mark_fp.sh $k "..."; done`.

## Progressive disclosure

- Always loaded: this SKILL.md
- Read on demand:
  - [`resources/known_fp_rules.md`](resources/known_fp_rules.md) — standing per-rule decisions + exact triage comments
  - [`resources/triage_playbook.md`](resources/triage_playbook.md) — "when you see X in Y, do Z"
- Run on demand: scripts in `scripts/`
