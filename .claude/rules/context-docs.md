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
procedure in `SKILL.md`, move the detail into `resources/` beside it — the layout every
multi-file skill here already uses (`resources/` for reference, `scripts/` for
executables). Existing breaches are grandfathered in `scripts/context_docs_baseline.txt`;
they may shrink, never grow.

**What splits and what stays.** `mission-control/SKILL.md` reached 4,201 lines with zero
reference files — ~96k tokens before the skill does anything. Its Gate-2 verification
protocol moved out whole (1,378 lines → `resources/verification-protocol.md`, 4,201 →
2,854) because of its shape: 18 rules whose *titles state the rule* and whose *bodies
carry the measured evidence*. The titles stayed in `SKILL.md` as a numbered index; the
war stories, commands and tells went behind the link. Test a candidate section against
that shape — if the agent needs it to take the next action, it stays; if it needs it to
*justify or debug* an action, it splits. Move it **verbatim**, and diff the result: a
split is a move, not a rewrite.

**A pointer you cannot follow is worse than no pointer.** Once detail lives behind a
link, that link is load-bearing. The gate checks every relative link in `CLAUDE.md`, the
rules, and every `SKILL.md`; known-broken ones burn down through
`scripts/context_docs_links_baseline.txt`, and an entry that starts resolving must be
removed. This is not hypothetical: `parser-developer` advertised five `resources/*.md`
files that were never written, so every agent it sent for the detail found nothing and
carried on without it.

## Hook output is context too

`SessionStart` and `UserPromptSubmit` hooks prepend to the context window on every
session and every prompt respectively. Same budget, same rule: surface the *signal* (an
unread count, a changed status) and let the agent run the command for the detail.
