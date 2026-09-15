---
name: ailang-core-triage
description: Triage an incoming AILANG core report — bug, feature request or observation — into a backlog row with a recommendation, cross-referenced against existing design docs. Use when a message arrives on the ailang-core inbox. Does NOT write design docs; it decides whether one is warranted.
---

# AILANG Core Triage

You have been handed ONE report about AILANG itself — a bug, a feature request,
a verification result, a gap someone hit. Your job is to say what should happen
to it, not to do it.

## Why this exists

Reports accumulated in the `user` inbox because there was nowhere better: 23 of
32 messages Daneel sent there were AILANG engineering items, 21 unread. Firing a
design doc at each is real money (~$1–3 and ~20 minutes of a full-size model
apiece) and wrong for the ones that are a two-line fix or already covered.

You are the cheap judgement in front of that spend.

## What you produce

ONE file, `design_docs/planned/ailang-core-triage/<slug>.md`, where `<slug>` is
a short kebab-case name for the report (`ailang-lock-absolute-paths`,
`serve-api-port-collision`). Nothing else.

**One file per report, NOT a shared table.** The first batch used a single
append-only backlog and all 13 runs dispatched within 26 seconds of each other
— `max_concurrent_tasks: 1` does not serialise cloud dispatch, because each task
is its own Cloud Run Job. Every branch appended to the same line of the same
file and all 13 PRs came back DIRTY. Agents that may run in parallel cannot
share an append target.

```markdown
# <Title, one line>

- **Date**: 2026-09-15
- **Class**: bug | feature | question | already-covered | not-actionable
- **Recommend**: design-doc | direct-fix | duplicate-of <path> | drop
- **Searched**: the terms you actually used
- **Estimate**: `<n> lines in <file>` (required for direct-fix; omit otherwise)

<One paragraph: the reason, not a restatement of the title. Name the file and
the mechanism if you found them.>
```

## How to decide

**Search before you classify.** Most of the value here is catching what is
already handled:

```bash
rg -il "<two or three distinctive words from the report>" design_docs/ | head
ls design_docs/planned/ | head -50
```

A report covered by an existing doc is `already-covered` / `duplicate-of <path>`.
Say which doc. This is the single most useful answer you can give.

### The thresholds

Tunable — change the numbers here and the rubric below follows.

| | |
|---|---|
| `DIRECT_FIX_MAX_LINES` | **2** |
| `DIRECT_FIX_MAX_FILES` | **1** |

### The rubric

Apply in order. The FIRST row that matches decides.

| # | Test | Answer |
|---|---|---|
| 1 | An existing design doc already rules on this | `duplicate-of <path>` |
| 2 | Not actionable as written — no reproduction, no subject | `drop`, and say what would make it actionable |
| 3 | There is **more than one acceptable way** to do it | `design-doc` |
| 4 | It changes a semantics, a public surface, a file format, or a gate's contract | `design-doc` |
| 5 | The change spans **more than DIRECT_FIX_MAX_FILES file** | `design-doc` |
| 6 | The change is **more than DIRECT_FIX_MAX_LINES lines** | `design-doc` |
| 7 | Otherwise | `direct-fix` |

**State your estimate, then OBEY it.** Every `direct-fix` must carry
`Estimate: <n> lines in <file>`. Write the estimate BEFORE the recommendation,
and if it exceeds `DIRECT_FIX_MAX_LINES` or names more than
`DIRECT_FIX_MAX_FILES` file, go back to row 5/6: the answer is `design-doc`.

Measured 2026-09-15: one row estimated `~15–30 lines` and still said
`direct-fix`. The estimate was right and was then ignored. A number you write
and disregard is worse than no number, because it looks like evidence.

That is what makes the call auditable: when the fix lands, its diff either
matches or it does not, and a pattern of underestimates is a reason to raise the
thresholds rather than trust the label.

If you cannot estimate the size, you do not understand the fix well enough to
call it direct — say `design-doc`.

### Two traps, both measured on the first batch

**A thorough report is not evidence of a small fix.** An earlier version of this
skill said "when the report already tells you the fix, that is evidence for
direct-fix". That is wrong and it was actively harmful: Daneel writes thorough
reports, so the rule fired on almost everything. A reporter who has located the
line has saved you the search — it says nothing about how many lines change.

**"Here are three acceptable fixes" is row 3, not row 7.** The daemon-liveness
report offered three remedies (log the error / exit non-zero / expose a
timestamp) and was labelled `direct-fix`. Three acceptable remedies is the
definition of a decision someone could disagree with.

## Rules

1. **Read the report before searching.** A search for the wrong words returns
   nothing and looks like "no existing coverage", which is the failure this role
   exists to prevent.
2. **One row per invocation.** You are handed one report.
3. **Never write outside `design_docs/planned/ailang-core-triage/`.** It is
   your only declared artifact; anything else is refused at merge. Do not touch
   another report's file.
4. **An empty search is a claim.** If you find no existing doc, say `none found`
   in Why and name the terms you searched, so the next reader can tell a real
   gap from a bad query.

## Citing a line number

When you quote a file:line from an existing design doc, say so — and check the
line still holds. Measured on this skill's first run: it cited
`resolver.go:127` because the design doc says `resolver.go:127`, and the call
has since moved to `internal/pkg/resolver.go:136`. The citation was faithful to
its source and sent a reader to the wrong line.

Prefer `<file> (`<the symbol>`)` over a bare line number, or write
"doc says :127, now :136".

## If you cannot proceed

If the report has no discernible subject, or the repo is missing something you
need, end with:

```
BLOCKED: <one line: what stopped you>
BLOCKED_ON: <the file, config or agent that must change first>
```

Do not write a file you do not believe.

## Output markers

End your response with:

```
TRIAGE_FILE: <the path you wrote>
RECOMMEND: <design-doc|direct-fix|duplicate-of <path>|drop>
```
