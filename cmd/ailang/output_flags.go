package main

import "flag"

// Output-format flag family (M-V1-SIMPLIFY-S5 M5, Phase 3 item 4).
//
// ONE question — "give me machine-readable output" — had four spellings across
// the CLI: --json (56 sites), --format json (2), --pretty (2) and --stream-json
// (1). `--json` is the canonical spelling and the only one a new command may
// register. This file is the single place that registers it, so the name, the
// default and the usage wording cannot drift apart again.
//
// What this milestone deliberately did NOT do, with the reasons, because a
// later reader will otherwise re-derive them:
//
//   - No superseded spelling emits a per-invocation deprecation line. The
//     sprint plan asks for "warn once on stderr", and every candidate fails a
//     concrete test. `ailang test --format json` is taught by fifteen SHIPPED,
//     FROZEN teaching prompts (cmd/ailang/prompts/*.md, testing_guide_ai.md,
//     devtools/v0.8.0.md) which check-prompt-freeze forbids editing — warning
//     there is the `ailang fmt` defect exactly: telling every eval agent that
//     its correct, prompt-taught invocation is wrong. `--models` on eval-suite
//     has fleet callers, and a warning on every one of them is a de-facto
//     demand for the caller sweep, which the sprint plan puts explicitly OUT of
//     scope. The migration signal is carried by help text and the changelog
//     instead. Reconsider once the caller sweep has landed.
//
//   - `check` keeps --format. It is not an output-format switch there: --format
//     agent is a third rendering, and M2 made `ai-check` an alias of
//     `check --verify --format agent`, a spelling pinned by
//     ai_check_exit_test.go (the DP7 done-gate) and by check-prompt-commands.
//
//   - `exec` keeps --stream-json. `--json` is ALREADY taken on exec and means
//     something else entirely (one JSON object at the end, not an NDJSON event
//     stream), and --stream-json defaults to TRUE. Folding them would either
//     flip a default — forbidden, it changes existing stdout — or put two
//     behaviours on one name, which is the defect this family exists to remove.

// flagJSON is the ONE flag name for machine-readable output. Registration goes
// through the helpers below so this string appears once.
const flagJSON = "json"

// registerJSONFlag registers the canonical --json output flag on fs for a
// command whose default rendering is human-readable.
func registerJSONFlag(fs *flag.FlagSet, usage string) *bool {
	return fs.Bool(flagJSON, false, usage)
}

// registerJSONOnlyOutputFlags registers the output-format pair for a command
// whose ONLY rendering is JSON (eval-paired, eval-censored-pairs): --json names
// the existing default — compact, machine-readable — explicitly, so a caller
// can spell the family's one flag on every command that emits JSON, and
// --pretty is the opt-out for a human reading the bytes directly.
//
// It returns indent(), reporting whether to indent. --json wins over --pretty
// when both are given, so NEITHER flag is silently ignored: the older, unused
// alternative — accepting --json and discarding it — is a fallback that hides
// what the caller asked for.
//
// --pretty is kept rather than deprecated because it does not answer this
// family's question. It selects an indentation, not a format; with JSON the
// only output there is nothing for `--json` to switch it to.
func registerJSONOnlyOutputFlags(fs *flag.FlagSet) func() bool {
	jsonOut := fs.Bool(flagJSON, false, "Compact machine-readable JSON (the default; wins over --pretty)")
	pretty := fs.Bool("pretty", false, "Indent the JSON output for reading")
	return func() bool { return *pretty && !*jsonOut }
}

// aliasStringFlag registers alias as an additional spelling for an already
// registered string flag, writing into the same target. The alias inherits the
// canonical flag's current value as its default, so registration order does not
// change what an un-passed flag resolves to; whichever spelling the caller
// actually passes wins, and passing both resolves last-wins, as a repeated flag
// already does.
func aliasStringFlag(fs *flag.FlagSet, target *string, alias, usage string) {
	fs.StringVar(target, alias, *target, usage)
}
