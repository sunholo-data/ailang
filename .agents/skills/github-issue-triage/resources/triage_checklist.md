# GitHub Issue Triage Checklist

Use this checklist for periodic issue triage sessions.

## Pre-Triage Setup

- [ ] Check GitHub authentication: `gh auth status`
- [ ] Switch to correct account if needed: `gh auth switch --user <user>`
- [ ] Update local repo: `git pull`

## Generate Triage Report

```bash
.claude/skills/github-issue-triage/scripts/triage_report.sh
```

## Process Closable Issues

- [ ] Review list of closable issues
- [ ] Verify each is truly implemented (check CHANGELOG, design docs)
- [ ] Preview close actions: `.claude/skills/github-issue-triage/scripts/find_closable.sh --dry-run`
- [ ] Close issues: `.claude/skills/github-issue-triage/scripts/find_closable.sh --close`

## Process Orphaned Issues

For each orphaned issue:

- [ ] **Priority check**: Is this critical, important, or nice-to-have?
- [ ] **Feasibility check**: Can we implement this? Resources available?
- [ ] **Scope check**: Does this fit AILANG's direction?

**Actions:**
- High priority → Create design doc (`/design-doc-creator`)
- Medium priority → Add to backlog, label appropriately
- Low priority/out of scope → Close with explanation

## Process Stale Issues

For each stale issue (no activity 30+ days):

- [ ] Check if still relevant
- [ ] Add comment: "Is this still an issue? Please update or we'll close."
- [ ] Set reminder to close in 14 days if no response

**Comment template:**
```
Hi! This issue hasn't had any activity in over 30 days.

Could you please confirm if this is still a problem?
If we don't hear back within 2 weeks, we'll close this issue.

Feel free to reopen if it becomes relevant again.
```

## Handle Bug Reports

For each bug:

- [ ] Reproducible? Ask for steps if not clear
- [ ] Severity? Label: `critical`, `major`, `minor`
- [ ] Workaround available? Document in comments
- [ ] Create design doc if fix is non-trivial

## Handle Feature Requests

For each feature request:

- [ ] Aligns with AILANG goals?
- [ ] Clear use case provided?
- [ ] Duplicates existing issue?
- [ ] Label: `enhancement`, `discussion`, etc.

## Update Labels

Standard labels to use:

| Label | Purpose |
|-------|---------|
| `bug` | Something broken |
| `enhancement` | New feature |
| `documentation` | Docs improvement |
| `good first issue` | Easy to fix |
| `help wanted` | Need community help |
| `wontfix` | Out of scope |
| `duplicate` | Already reported |
| `question` | Needs clarification |
| `in-progress` | Being worked on |

## Post-Triage Summary

- [ ] Update triage log (if maintained)
- [ ] Note any blocked issues and why
- [ ] Schedule next triage (recommended: weekly)

## Quick Commands Reference

```bash
# List all open issues
.claude/skills/github-issue-triage/scripts/list_open_issues.sh

# Match issues to design docs
.claude/skills/github-issue-triage/scripts/match_design_docs.sh

# Find closable issues
.claude/skills/github-issue-triage/scripts/find_closable.sh

# Close issues
.claude/skills/github-issue-triage/scripts/find_closable.sh --close

# Full triage report
.claude/skills/github-issue-triage/scripts/triage_report.sh
```
