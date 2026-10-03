# CLI: flags after the positional argument are silently dropped, across most `flag.NewFlagSet` subcommands (#534, #1383)

- **Date**: 2026-10-03
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `trailing flag`, `flags after`, `interspers`, `parseTestArgs`, `normalizeArgsForFlags`, `NArg()`, `must appear before` across `design_docs/`, `cmd/ailang/`; `ailang-core-triage/`; `git log origin/dev --grep 534`. Found: `cmd/ailang/test.go:21` `parseTestArgs` (an interspersed parser, `test` only), `cmd/ailang/messages_send.go:488` `normalizeArgsForFlags` (`messages send` only; its own defect is #1276, see `messages-send-unknown-flags-stored-as-body.md`). No doc covers the systemic fix.
- **Estimate**: omitted (design-doc)

**Still live at origin/dev** (dev binary built from `790169359`). `ailang check t.ail --totally-bogus` exits 0. So do `iface`, `verify` and `ai-check` with a trailing unknown flag. `ailang unpublish pkg@ver --force` ignores `--force` and then fails the no-TTY prompt (#1383, `cmd/ailang/pkg_unpublish.go:18-22`). Go's `flag` stops at the first non-flag. Most of the 157 `flag.NewFlagSet` call sites in `cmd/ailang` then read only `Arg(0)` and never look at the rest. `ailang test` is **fixed**: `parseTestArgs` loops `fs.Parse` over the remainder and treats tokens after `--` as paths, so `test t.ail --format json` emits JSON and `test t.ail --totally-bogus` exits 2.

So the repo already contains one answer, applied to one command. Two commands now behave GNU-style, and everything else drops flags silently. The embedded prompt teaches "Flags must come BEFORE the filename!", which documents the trap instead of closing it.

**Decision needed.**

1. **Accept interspersed flags everywhere.** Promote `parseTestArgs` to a shared `parseInterspersed(fs, args)` and route every subcommand through it. This is the most forgiving option, and it matches `test` and the usage lines (`unpublish <pkg@ver> [--force]`).
2. **Reject loudly.** After `Parse`, any leftover token starting with `-` (before a literal `--`) is an error: `flag --x must appear before <arg>`, exit 2. The change is smaller, but it breaks the documented `unpublish` order and is inconsistent with `test`.
3. **Both, by command class.** Commands that take paths or package ids get (1). Commands that forward the remaining argv to something else must stop at the first positional or at `--`, and they get (2) for unknown flag-shaped tokens before it. That covers `run <file> [program args]` and free-text payloads like `messages send <inbox> <text...>`.

**Recommendation: (3).** The forwarding commands are why a blanket (1) is unsafe: permuting `run prog.ail --verbose` would steal the program's own argument. Deliverables: one helper in `cmd/ailang` with a forwarding mode, a table-driven test enumerating every registered subcommand from the command table (`commands_table_test.go` already lists them) that asserts `<cmd> <positional> --definitely-not-a-flag` never exits 0 silently, an `unpublish pkg@ver --force` no-TTY test (the #1383 case), and removal of the "flags before the filename" rule from the prompt once it no longer applies. #1276's normalizer should converge on the same helper, but its body-text semantics stay in that doc.

Issues: https://github.com/sunholo-data/ailang/issues/534 (systemic); #1383 (instance, closed as duplicate of #534)
