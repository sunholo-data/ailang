---
name: github-issue-triage
description: Triage open GitHub issues on sunholo-data/ailang by verifying each one against the current code on origin/dev, then closing (with evidence), labelling, transferring or escalating it. Use when the user asks to "check issues", "triage issues", "sync issues", "what issues are open", "clean up the issue tracker", or after a release to find what shipped.
---

# GitHub Issue Triage

**Every verdict must come from the issue read against the code on `origin/dev`, not from a keyword match.** The scripts only say where to start.

Measured 2026-09-28, against 100 issues verified one by one:
- The old "closable" heuristic (`#N` mentioned in a changelog or an implemented doc) flagged 25 issues. 7 were fixed and 18 were live bugs, several of them P0/P1. Its `--close` flag would have closed all 25.
- It also missed 6 real fixes whose commits never named the issue.
- `--close` has been removed.

## Current State

- **GitHub auth**: !'gh auth status 2>&1 | grep -E "Active account: true" -B1 | head -1'
- **Open issues**: !'gh issue list --state open --limit 1000 --json number --jq length 2>/dev/null || echo "gh not available"'
- **Missing a priority label**: !'gh issue list --state open --limit 1000 --json labels --jq "[.[] | select([.labels[].name] | any(startswith(\"priority:\")) | not)] | length" 2>/dev/null'

## Workflow

### 1. Auth

This repo is worked as **`sunholo-voight-kampff`**, the bot account; `check_auth.sh` compares against `github.expected_user` in `~/.ailang/config.yaml`. `MarkEdmondson1234` is the human's account, and `rw-markedmondson` is a different project. If the wrong one is active, run `gh auth switch --user sunholo-voight-kampff`.

### 2. Starting-point report

```bash
.claude/skills/github-issue-triage/scripts/triage_report.sh --output <scratchpad>/triage.md
```

It gives counts, **close candidates (unverified)**, issues missing a priority label, `needs-decision`, doc references and stale issues. All of it is read from `origin/dev`, never the working tree, because a worktree may sit on an old branch.

What the candidate list is worth:
- In the same measurement, its signals caught 3 of 13 real fixes.
- **An issue missing from the list may still be fixed.**
- An issue on the list may still be live: a commit once said "closes #612" wrongly.

### 3. Verify every issue against the code (the actual triage)

Split the open issues into groups of about 20 and give each group to a **read-only** agent, running in parallel, using [`resources/verify_agent_prompt.md`](resources/verify_agent_prompt.md). Each agent:
- reads the issue and its comments;
- searches `git log origin/dev` for fix commits, by `#N` and by keywords;
- checks `design_docs/planned/` against `implemented/`;
- reads the current code path;
- reproduces with a build of origin/dev where that's cheap.

An empty search is a claim, not a fact: an agent should try a second phrasing before concluding "not fixed".

Each agent returns exactly one verdict per issue:

| Verdict | Meaning | Action |
|---|---|---|
| `CLOSE-FIXED` | Fixed on origin/dev | Close with the commit SHA, PR or release, and how it was verified |
| `CLOSE-OBSOLETE` | Premise moot, or the week or feature is gone | Close with the evidence |
| `CLOSE-DUPLICATE` | Same defect as #X | Close in favour of the **sharper, current** one (see below) |
| `TRANSFER` | The fix belongs in another repo | `gh issue transfer N sunholo-data/<repo>` |
| `KEEP-PLANNED` | Real, and a planned doc covers it | Label it; note the doc |
| `KEEP-ORPHAN` | Real, no doc | Label it with a priority; a doc if P0/P1 |
| `NEEDS-DECISION` | Needs a maintainer ruling | Label `needs-decision`; state the decision in one line |

**Show the user the combined verdict list, and get approval before closing, transferring or relabelling anything.**

### 4. Act, with evidence

- **Close** with `gh issue close N --reason completed|"not planned" --comment "<evidence>"`. The comment cites the fix commit or the reason, and says "verified against origin/dev <sha>".
- **Duplicates.** When two agents disagree about which of a pair to keep, keep the one that describes the current defect most precisely. That's usually the newer re-file, whose older sibling holds a half that has since been fixed or refuted.
- **Transfer.** Fixes that live in `sunholo-data/ailang-parse`, `email-parse` or `daneel` are moved, not closed and not left open here. The bot has admin on those repos.
- **Weekly bookkeeping threads** (`<mission> bookkeeping: week of <date>`). Close a finished week once a newer thread exists and it has no open directives. **Never close the current thread;** the mission loop rotates it.
- **Stale.** "Stale" means the mechanism was refuted or the watch item has gone quiet, not simply 30 days without activity. Close with a "reopen with a fresh repro" note.

### 5. Labels: priority plus area, and never a coordinator-watched label

| Label | Meaning |
|---|---|
| `priority:P0` | Soundness, security or data loss; fix before new features |
| `priority:P1` | A real bug hurting users or agents now |
| `priority:P2` | Worth doing, not urgent |
| `priority:P3` | Nice to have |
| `needs-decision` | Blocked on a maintainer ruling |
| `area:*` | `effects` `ifc` `test-runner` `bytecode` `language` `verify` `iface` `cli` `pkg` `ai` `mission` `ci` `serveapi` `runtime` `messaging` |

**Don't add `bug`, `feature` or `from:*` to existing issues.** They are the coordinator's default `watch_labels` (`internal/coordinator/agent_config.go`), so adding one can import an old issue as a new task. The labels above are not watched.

After a pass, `gh issue list -l priority:P0` is the fix-first list.

### 6. Sync the design-doc tree

- **Shipped work still in `planned/`.** Move its design doc to `design_docs/implemented/<release>/`; `git tag --contains <fix-sha>` gives the release. Replace the status line with "Implemented in vX (sha; closes #N)", and keep the prior status as history. Fix the relative links to and from the moved doc.
- **Why it matters:** a doc left in `planned/` makes the next report list shipped work as open.
- **Real issues with no doc:** P0/P1 soundness or correctness clusters get one design doc per root cause, via `/design-doc-creator`. Check first whether the issues share a root cause.

### 7. Security alerts: check they're stale before acting

Dependabot evaluates GitHub's dependency graph, which can lag behind `go.mod`. On 2026-09-28, 30 of 40 alerts were for versions no file pinned anymore. For each alert:
- compare `security_vulnerability.first_patched_version` with the version pinned on `origin/dev`;
- dismiss it with `dismissed_reason=inaccurate` and both versions in the comment only when the pinned version is at or past the patch;
- keep it open when no patch exists, and note it.

## Scripts

| Script | Purpose |
|---|---|
| `check_auth.sh [--quiet]` | Active `gh` account matches `github.expected_user` |
| `list_open_issues.sh [--json] [--labels L] [--limit N]` | Open issues with labels and age (default limit 1000; `gh` truncates silently at the limit) |
| `triage_report.sh [--output F] [--stale-days N] [--ref R]` | Starting-point report (step 2) |
| `find_closable.sh [--json] [--ref R]` | Close **candidates** only: `fixes #N` commits, and implemented docs naming `#N` in their header. It cannot close. |
| `match_design_docs.sh [--planned\|--implemented] [--json] [--ref R]` | Which docs reference each issue, by exact `#N` or an M-ID filename match. A match means "mentioned", not "covered". |

All of them read docs and history from `--ref` (default `origin/dev`, fetched first) and fail loudly if the fetch fails.

## Gotchas

- `comm` needs **lexically** sorted input. `sort -n` output silently mis-joins issue-number sets.
- The interactive shell is zsh: `set -- $pair` does not word-split, so run transfers and closes explicitly or in `/bin/bash`.
- `ailang iface --json` escapes `<` as `<`, so grep for the escaped form.
- Use a worktree for any doc moves. Never switch branches in a checkout other agents share.
