# `messages send` silently stores unknown flags as the message body

- **Date**: 2026-09-23
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `body-file`, `normalizeArgsForFlags`, `unknown flag`, `argparse` across `design_docs/`; checked `design_docs/planned/ailang-core-backlog.md` and the `ailang-core-triage/` directory for a prior row
- **Estimate**: n/a (design-doc)

The mechanism is confirmed in `cmd/ailang/messages_send.go` (`normalizeArgsForFlags`): the normalizer builds its `takesValue` map from the `*flag.FlagSet` and routes any token not in that map into the *positional* list. `--body-file` is not a declared flag, so both it and its path become positionals, and `runMessagesSend` joins `fs.Args()[1:]` into the payload with `strings.Join`. Go's `flag` package stopping at the first positional is not even needed — the normalizer itself absorbs the unknown flag. Exit 0, `✓ Message sent`, body = `--body-file <path>`. Real damage already measured: two messages on `pkg:sunholo/ailang_parse` (2026-09-23, `inbox_1790154667634_483d7d81`, `inbox_1790155250762_e81a8546`) carried only the misparsed flag, one was dispatched as `task-483d7d81` and completed `no_changes` — content lost, sender believed delivered, agent ran on nothing.

This is a **new defect in the same function** `design_docs/planned/m-coordinator-execution-trust.md` already triaged once: V29/M5 fixed dash-prefixed flag *values* being refused by the normalizer, misrouting the inbox and title. That doc's thesis ("a misrouted message that reports success") covers this failure class, but its verdict does not rule on unknown flags falling through to positionals — so it is related context, not a duplicate.

Recommendation is `design-doc` for two rubric reasons:

1. **Row 3 — more than one acceptable way.** The report itself proposes (a) reject unknown flags with non-zero exit, (b) add a real `--body-file <path>` (and `-` for stdin), plus a store-level guard on payloads beginning with `--`. (a) and (b) are not mutually exclusive — the report argues (b) is worth doing *as well as* (a) — but which combination to ship, and whether the store-level guard belongs in the message plane at all, is a decision someone could disagree with. Note (b) has a precedent in this repo: `ailang mission report --body-file <path>` already exists (`design_docs/planned/v0_36_0/m-mission-comms-into-the-binary-sprint-plan.md:79`), so consistency with that surface is part of the decision.
2. **Row 4 — public surface.** Both (a) (rejecting previously-accepted invocations — any script that today relies on trailing `-`-prefixed text will start failing) and (b) (new flag on a heavily-used command) change the CLI contract.

Related existing coverage found: none on this exact defect. Terms searched: `body-file`, `normalizeArgsForFlags`, `unknown flag`, `argparse`. The nearest neighbours are `m-coordinator-execution-trust.md` (V29/M5, same function, different defect) and `docs-10-sprint-plan.md:297` (unknown-flag ingestion in a different subsystem). Suggested routing: fold this into the next revision of `m-coordinator-execution-trust.md` if that milestone is still open, since the failure class and the file are the same; otherwise a small standalone design doc.
