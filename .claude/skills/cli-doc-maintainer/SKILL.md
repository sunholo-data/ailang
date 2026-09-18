---
name: cli-doc-maintainer
description: Keep the AILANG CLI and its docs true to the dispatch table. Use when adding or renaming a command, adding a flag or env var, auditing CLI docs, fixing a doc that cites a command the binary rejects, or when `ailang --help` and a page disagree.
---

# CLI Documentation Maintainer

**There is nothing here to keep in sync by hand any more.** `cmd/ailang/commands.go` is the only
place a command name is written down; `ailang --help`, every group's help, and
`docs/docs/reference/cli.md` are all rendered from it. This skill is the map of which instrument
answers which question — not a set of audits you run.

## Adding or changing a command

1. Edit the `Command` row in `cmd/ailang/commands.go` (or `commands_language.go` /
   `commands_platform.go` / `commands_eval.go`, which assemble it). `Name`, `Aliases`, `Group`,
   `Hidden`, `Language`, `Summary`.
2. `make docs-cli` — regenerates `docs/docs/reference/cli.md`.
3. Commit the page with the code. `make check-cli-docs` fails if they disagree, and so does
   `go test ./cmd/ailang/...`, which CI already runs.

A command filed in a group **keeps its top-level spelling** (D1). Do not rename a route: 593 fleet
references call `ailang messages`, 332 `ailang coordinator`, 216 `ailang chains`. The caller sweep
is a separate, later change.

## Which instrument answers which question

| Question | Instrument |
|---|---|
| Does the reference page match the table? | `make check-cli-docs` (byte-compares a fresh render against the checked-in page) |
| How many commands does `ailang --help` show? | `make simplicity-metrics-fast` → `commands_top_level`, gate 20 |
| Does every command answer `--help`? | same → `help_exit0_rate`, gate 100 |
| How many spellings answer "machine-readable output"? | same → `output_format_spellings`, gate 4 (a ratchet; `cmd/ailang/output_flags.go` is the design record) |
| Does a teaching prompt cite a command the binary rejects? | `make check-prompt-commands` |
| Is an environment variable documented? | `make docs-env` regenerates `docs/docs/reference/env-vars.md` from `internal/config`'s Registry; `env_vars_documented_pct` gates it at 100 |

## Never probe with a captured token

Auditing docs means running `<name> --help` for **bare command names only**, read out of the
binary's own generated help and validated against `^[a-z][a-z0-9-]*$` first. The first version of
`tools/check_prompt_commands.sh` passed a token it had scraped from text, and two faults compounded:
`[a-z0-9-]+` matches a flag (`-` is literal inside a bracket expression), so `ailang test --format
json` yielded the opener `ailang test --format`; and `ailang test --format --help` does not print
help, it **runs the test suite**. Several `ailang test` processes ground for half an hour.
`tools/check_prompt_commands.sh` and `tools/simplicity_metrics.sh` both carry the safe pattern —
copy one of them rather than writing a third.

## Three audit scripts were retired (M-V1-SIMPLIFY-S5 M6)

`audit_commands.sh`, `suggest_improvements.sh` and `audit_env_vars.sh` all measured
`cmd/ailang/help.go` as a hand-written file and `main.go` as a `case` switch. S5 M1 generated the
first and emptied the second, so by M6 they were not merely stale, they were **actively wrong**:
`audit_commands.sh` read "Commands in main.go: 1" and demanded that an empty string be documented;
`suggest_improvements.sh` asked for "See also:" cross-references in generated output;
`audit_env_vars.sh` reported 272 variables as undocumented by comparing against a `help.go` that has
not listed environment variables since S4 moved them to `internal/config`'s Registry. The gates in
the table above replace all three, structurally. Note for anyone reviving them: BSD grep has no
`\s`, which two of the three used.

`resources/best_practices.md` and `resources/help_template.md` describe how to hand-write
`help.go`'s prose and are pre-S5 history; they are not instructions for today's CLI.
