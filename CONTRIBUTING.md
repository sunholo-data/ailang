# Contributing to AILANG

AILANG is an AI-first open source project. The compiler, runtime, and standard library are built almost entirely by AI agents operating through a structured pipeline. This isn't exclusionary — it means contributors get to focus on the highest-leverage work: ideas, design, documentation, and developer experience.

## How Your Contribution Becomes Code

```
You open an issue
    ↓
Auto-imported into agent inbox
    ↓
Triaged against existing design docs
    ↓
Design doc created (human-approved)
    ↓
Sprint planned with acceptance criteria
    ↓
AI agent implements with TDD
    ↓
Independent AI evaluator scores quality
    ↓
Human reviews and merges
```

Your issue drives the entire pipeline. The more detail you provide — use cases, reproduction steps, examples — the better the result.

## Ways to Contribute

### Issues (the primary path)

Issues are the entry point for all new work. Use the templates:

- **Bug Report** — Something broken? Include AILANG version, minimal reproduction, expected vs actual behavior.
- **Feature Request** — Focus on the use case and the "what/why", not implementation details.
- **DX/UX Feedback** — Error messages confusing? CLI awkward? Install painful? This feedback is gold.
- **Design Discussion** — Proposing a major architectural change? This template feeds directly into the design doc system.

### Documentation PRs (always welcome)

Documentation pull requests are reviewed and merged directly. This includes:

- Docs site content (`docs/`)
- Example programs (`examples/`)
- Error message improvements
- Tutorials and guides
- Editor/tooling integration docs
- README improvements

Run `make verify-examples` before submitting to ensure examples compile.

### Design Proposals

For larger changes, use the **Design Discussion** issue template. If accepted, the maintainer creates a formal design document in `design_docs/planned/` using the project's standard format (axiom scoring, decision tables, impact assessment). Your proposal may be modified during this process.

You don't need to write AILANG code to make a significant design contribution.

### AI Agent Contributors

If you're an AI agent (or operating one) and want to contribute, the preferred path is programmatic issue creation via the `gh` CLI. This feeds directly into our auto-triage pipeline.

**Preferred: `gh` CLI with structured labels**

The general pattern:

````bash
gh issue create \
  --repo sunholo-data/ailang \
  --title "[your-agent-name] Brief description" \
  --label "CATEGORY,ailang-message" \
  --body "BODY"
````

Use these category labels and body templates:

**Bug report** — label: `bug,ailang-message`

````markdown
## What happened?
Description of the bug

## Steps to Reproduce
Minimal AILANG code or CLI commands that trigger the bug

## Expected Behavior
What should happen

## Actual Behavior
What actually happens (include error messages)

## AILANG Version
v0.9.0

## Component
Parser / Type System / Runtime / etc.
````

**Feature request** — label: `enhancement,ailang-message`

````markdown
## What problem does this solve?
Use case description

## Proposed Solution
How it should work

## Area
Language Syntax / Type System / CLI / etc.
````

**Design discussion** — label: `design-discussion,ailang-message`

````markdown
## Problem Statement
What problem does this solve?

## Design Proposal
High-level approach, trade-offs, alternatives

## Impact Assessment
Which components are affected?

## Alternatives Considered
Other approaches
````

**Key details for programmatic submissions:**

- **Always include the `ailang-message` label** — this is how our import pipeline discovers your issue
- **Prefix the title with `[your-agent-name]`** — helps us track which agent submitted it
- **Use category labels**: `bug`, `enhancement`, `documentation`, `design-discussion`, `dx`, `feedback`
- **Structure the body** with the section headers shown above — our triage agents parse these

**What happens next:**

1. `ailang messages import-github` picks up issues labeled `ailang-message`
2. The issue enters the agent inbox with your agent name and category
3. It's triaged against existing design docs
4. If accepted, a design doc is created and enters the sprint pipeline
5. Your issue is auto-closed with a version reference when the fix ships

**Important:**

- Wait for design doc approval before any implementation work
- Implementation is coordinated through our pipeline to ensure TDD, axiom compliance, and independent evaluation
- Direct code PRs from external agents will be closed, same as human code PRs

## What About Code PRs?

We do not accept code pull requests for compiler, runtime, or standard library changes. This isn't about the quality of your code — it's about the pipeline:

1. **TDD enforcement** — The sprint executor writes tests before implementation and maintains them throughout
2. **Axiom compliance** — Every change is scored against AILANG's 12 design axioms
3. **Independent evaluation** — A separate AI evaluator scores the implementation against acceptance criteria
4. **Architectural coherence** — Design doc approval ensures changes fit the larger vision

If you submit a code PR, we'll close it with thanks and ask you to open an issue instead. Your idea is valuable — the issue is how we capture it.

**Exceptions:** Documentation-only PRs, CI/build fixes, and critical security patches.

## Getting Credit

Contributors whose issues lead to implemented features are credited in:

- The design document's References section (links back to your issue)
- The CHANGELOG entry for the release
- Git commit messages (`Fixes #N` links and closes your issue automatically)

## Development Setup (for documentation contributors)

```bash
# Install
go install github.com/sunholo-data/ailang/cmd/ailang@latest

# Or build from source
make build
make install

# Run tests
make test

# Verify examples compile
make verify-examples

# Start the REPL
make repl
```

## Code of Conduct

We follow the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). Be kind, be constructive, and remember that behind every issue is someone trying to make AILANG better.

## Questions?

Open a [DX/UX Feedback](../../issues/new?template=dx_ux_feedback.yml) issue if something about the contribution process is unclear. That's exactly the kind of feedback we want.
