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

- `design-doc` — the fix needs a decision someone could disagree with: a
  semantics change, a new surface, a trade-off between two workable designs.
- `direct-fix` — the change is obvious once seen, and a doc would be ceremony.
  A wrong error message, a missing flag, an absolute path where a relative one
  belongs.
- `drop` — not actionable as written. Say what would make it actionable.

**When the report already tells you the fix, that is evidence for
`direct-fix`, not against it.** A reporter who has located the line does not
need a design doc to re-derive it.

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
