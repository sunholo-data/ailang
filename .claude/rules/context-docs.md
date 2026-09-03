---
paths:
  - "CLAUDE.md"
  - ".claude/rules/**"
  - ".claude/skills/**"
  - "scripts/hooks/**"
---

# Writing Context Documents

A **context document** is anything the harness injects into a session without being
asked: `CLAUDE.md`, `.claude/rules/*.md`, `.claude/skills/*/SKILL.md`, and hook output.
They are loaded **wholesale** — there is no partial read — so every line is a line every
future session pays for whether or not it is relevant.

The gate is `make check-context-docs`. The convention it enforces:

## Three tiers, and the rule for choosing

| Tier | Loaded | Belongs there |
|------|--------|---------------|
| **Always-on** (`CLAUDE.md`, unscoped rules) | every session | what the session needs *before* it knows what it will touch |
| **Path-scoped** (`paths:` frontmatter) | when a matching file is read or edited | anything you can name a file for |
| **On-demand** (skill reference files, `docs/`, design docs) | when an agent opens it | everything else — war stories, tables, derivations, worked examples |

**Default to on-demand.** Promote to path-scoped only when an agent would otherwise
break something before it thought to look; promote to always-on only when no path can
predict the need. A rule with no `paths:` block loads forever, so the gate requires it to
say why in the first five lines:

```markdown
<!-- always-on: <what makes every session need this> -->
```

## Scope with paths that exist

A `paths:` glob that matches nothing is a rule that **never loads, and never fails**.
`ailang-syntax.md` sat scoped to `stdlib/**` while the tree had `std/` — the AILANG
syntax rule had quietly stopped loading for every stdlib edit, and nothing complained.
The gate now matches each glob against `git ls-files`. When you rename a directory,
grep `.claude/rules/` for the old name.

Scope to where the work happens, not to where the code lives: a rig-memory rule scoped
only to `internal/ai/ollama/**` never fires for a session diagnosing the rig over Bash.
That is why the operational lines also sit behind a pointer in the always-on tier.

## Write the pointer, not the payload

The mechanism is a link an agent follows when the task needs it. So:

- **Say what the reader would not think to look for.** A pointer's value is the fact that
  the document exists and what question it answers — not a summary of it.
- **One home per fact.** Duplicating a paragraph into an always-on file to "make sure it
  is seen" costs every session and produces two copies that drift.
- **Every link must resolve.** A dead pointer turns progressive disclosure into a missing
  fact; the gate checks relative links in `CLAUDE.md` and every rule.

## Caps

Line caps (`make check-context-docs`): 300 for `CLAUDE.md`, 200 per rule, 500 per
`SKILL.md` — the last being Anthropic's own guidance for a skill body.

When a `SKILL.md` outgrows the cap the fix is **layering, not brevity**: keep the
procedure in `SKILL.md`, move the detail into sibling files it links to.
`mission-control/SKILL.md` reached 4,201 lines with zero reference files — roughly 96k
tokens spent before the skill does anything — purely because appending was easier than
filing. Existing breaches are grandfathered in `scripts/context_docs_baseline.txt`; they
may shrink, never grow.

## Hook output is context too

`SessionStart` and `UserPromptSubmit` hooks prepend to the context window on every
session and every prompt respectively. Same budget, same rule: surface the *signal* (an
unread count, a changed status) and let the agent run the command for the detail.
