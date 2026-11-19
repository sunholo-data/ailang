# fmt round-trip fails on whole-string-hole and refused-hole interpolations (ResolveAsBuiltin lost in call form)

- **Date**: 2026-11-19
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `round-trip verification failed` (code: `cmd/ailang/fmt.go:formatOne`), `concat_String`, `M-CONCAT-DISAMBIG`, `interpolationString`, `re-sugar` in design_docs/; related doc found: `design_docs/implemented/v0_32_0/m-fmt-dialect-alignment.md` (introduced the re-sugaring and its guards — does NOT rule on these shapes). No existing planned doc covers the residual failure.
- **Estimate**: omit (design-doc)

## Root cause (one mechanism behind all three repros, confirmed empirically at v0.57.4)

`parseInterpolatedString` (internal/parser/parser_literals.go) desugars `"${e}"` into `show(e)` with the synthesized identifier flagged `ResolveAsBuiltin: true`. That flag exists ONLY at parse time and nothing the printer can emit in call form preserves it — re-parsing printed `show(e)` yields a user-written `show` with `ResolveAsBuiltin: false`. `formatOne`'s round-trip gate (cmd/ailang/fmt.go, `cmp.Diff` ignoring only Pos/Span) then correctly detects an AST change and fails closed with "AST changed". Two shapes reach that dead end:

1. **Whole-string single hole** — `"${r == "x"}"`, `"${"x"}"`, `"${join("", [r])}"`, even `"${r}"`: empty STRING_PARTs are elided, so the desugar is a *bare* `show(e)`, not a `concat_String` chain. `interpolationString` (internal/format/interp.go) only matches chains, so it never fires; output is `show(e)`; flag lost. Adding literal text (`"a${...}"`) makes it a chain and it passes — matches the report.
2. **holeText refusal → call-form fallback** — the guards in `interpolationString`/`holeText` (interp.go: newline, `--`, `${`) are not round-trip-safe on refusal: the fallback prints the *synthesized* `show(...)` as ordinary call syntax, so the re-parse loses `ResolveAsBuiltin: true`. Confirmed triggers: a `match` in a hole (`p.match` always emits a `hardline`, so holeText's newline guard refuses — fails with or without surrounding text); a nested interpolation inside the hole (`show("a${r}")` — the inner string re-sugars to text containing `${`, refusing the outer hole). Repros 1 and 3 are both this shape.

Verified negative controls: `println("a${show(r)}")`, `println("a${f(r)}")`, `println("a${concat_String(r, x)}")`, `println("a${r + 1}")` all PASS; `println("${r}")` FAILS — isolating the mechanism precisely.

The fail-closed batch behavior of `fmt --write` (fmtWrite validates all files before touching any) is by design; the defect is that correct, check-passing programs trip it.

## Why design-doc, not direct-fix

- **Three-plus acceptable remedies, a real trade-off**: (a) ignore `ResolveAsBuiltin` in `fmtIgnorePos` — weakens the round-trip gate's contract and could mask genuine show/hygiene defects (the flag exists precisely so a user's `show` can never capture holes — see internal/ast Identifier docs); (b) teach the printer to re-sugar a bare *synthesized* `show(e)` as `"${e}"`; (c) change the desugar to keep a chain or record interpolation provenance in the AST (touches M2_PARSER_TYPECHECK_INTERP territory, i.e. the desugar is a public-ish AST contract).
- **Spans more than one file**: internal/parser/parser_literals.go and/or internal/format/interp.go + expr.go, plus cmd/ailang/fmt.go if the gate contract changes, plus tests.
- Far beyond 2 lines in either case.

## Notes for whoever picks it up

- The doc should state which shapes must round-trip after the fix and re-run `TestFormatterOutputAlwaysReParses`-style sweeps — the existing corpus sweeps (`std/` + `examples/`, internal/format/roundtrip_soundness_test.go) evidently contain none of these shapes, which is why CI missed it; add all three repro families as corpus cases.
- Watch the interaction with (b): re-sugaring a bare synthesized `show(e)` must refuse user-written `show(e)` (the `ResolveAsBuiltin` check interp.go already uses) and holes that render multi-line, or it re-opens the round-trip hole elsewhere.
- Related shipped guards and their rationale live in `design_docs/implemented/v0_32_0/m-fmt-dialect-alignment.md` ("four latent bugs" list); this report is the missing fifth: **the fallback path itself is not round-trip-reversible**.
