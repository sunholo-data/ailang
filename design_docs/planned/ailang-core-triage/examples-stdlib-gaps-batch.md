# Six language/stdlib gaps found writing `examples/runnable/stdlib/` (reporter commit f06cb6e84)

- **Date**: 2026-11-06
- **Class**: bug (items 1, 3, 4) + feature (items 2, 5) + question/observation (item 6)
- **Recommend**: design-doc
- **Searched**: `exhaustiv|non-exhaustive|exhaustive match` over design_docs/ and docs/ (only VISION.md rhetoric + archived failure analyses; no ruling); `max|min|std/math` over design_docs/planned (nothing on int min/max; `ailang-core-triage/stdlib-number-naming-and-result-option-convention.md` covers naming conventions generally); `yaml` over triage rows (only `yaml-decode-loses-mapping-key-order.md` — key order, not error text); `AILANG_TRACE|tier|deep` over design_docs/ (v0_12_0 m-obs-trace-triage is the tier system's origin doc, historical); `qualif|S\.Set` (prompt/docs only); `unquoted|quoting` + show (nothing). No existing triage row covers any of the six.
- **Estimate**: omitted (design-doc)

All six items were re-verified against HEAD (`d9a6e665`, binary v0.57.4) before
classification — none is stale, none is a duplicate of an existing doc, and all
six are real.

## Item-by-item (rubric row that decides each)

1. **Non-exhaustive `match` passes `ailang check`, fails at runtime** — **design-doc (rows 3/4).**
   Reproduced: a `Shape = Circle(float) | Square(float)` ADT matched on only
   `Circle(r)` prints `✓ No errors found!` from `ailang check`; the only guard is
   the runtime error `"no pattern matched in match expression"`
   (`internal/eval/decision_tree.go:114`, also `internal/eval/eval_patterns.go:101`).
   This is the check-gate's contract changing (row 4), and there is more than one
   acceptable shape (row 3): hard error vs `-Wstyle` warning vs ADT-constructor
   coverage only (literals/wildcards/guards interact — a guard arm makes coverage
   undecidable, so the rule needs a stated boundary). Notably `docs/VISION.md:114-133`
   *sells* "compiler-enforced exhaustiveness" as AILANG's headline advantage over
   dynamic languages — the product claim and the implementation have diverged, which
   is the strongest argument this deserves a doc rather than a silent warn. Also not
   in `docs/LIMITATIONS.md` or the prompt (verified), so the stopgap doc row is also
   missing. A doc should decide the check semantics, the message shape, and add the
   LIMITATIONS row in the meantime.

2. **No integer max/min in std/math or builtins** — **design-doc (row 3).**
   Reproduced: `import std/math (max)` → `IMP010: symbol 'max' not exported by
   'std/math' — did you mean 'tan'?`. No `_int_max`/`_math_max` builtin exists
   (`internal/builtins/` grep clean); `std/math.ail` exports only float functions
   plus `abs_Int`/`abs_Float` (note the `_Int`/`_Float` suffix convention —
   `max`/`min` would follow it as `max_Int`, or differ, or go polymorphic).
   Several acceptable designs: pure-AILANG two-liner (`if a > b then a else b`,
   no builtin needed) vs a builtin pair; int-only vs int+float; `std/math` vs a
   comparison/ordering home; and whether the prompt teaches them (the reporter's
   point — models invent `max` constantly — is a prompt-teaching question as much
   as a stdlib one). Row 3 fires; cheap doc.

3. **std/yaml error text doubles the `yaml:` prefix** — **design-doc (row 5, mechanically).**
   Confirmed at `internal/builtins/yaml.go:84,90,97` — every decode error is
   wrapped as `fmt.Sprintf("yaml: %s", err)`, but the gopkg.in/yaml.v3 errors
   already carry the `yaml: ` prefix, producing `yaml: yaml: line 1: …`.
   The fix itself is mechanical (strip the library prefix or stop re-prefixing),
   but it spans three files: `internal/builtins/yaml.go`, the test that pins the
   doubled text (`internal/builtins/yaml_test.go:211`, `require.Equal(t, "yaml: yaml: map merge …")`),
   and `examples/runnable/stdlib/yaml_to_json_convert.ail` whose expected stdout
   pins it too. Three files > `DIRECT_FIX_MAX_FILES = 1` → row 5 → design-doc,
   though it is a strong candidate for a one-milestone section inside a stdlib-polish
   batch rather than a standalone doc.

4. **std/trace spanStart/spanEnd emit nothing at the default trace tier** — **design-doc (row 3).**
   Confirmed: `traceSpanStartImpl`/`traceSpanEndImpl` (`internal/builtins/trace.go:161-186`)
   call `ctx.RecordFunctionEnter/Exit`, and function-call events are gated to
   `c.tier >= TierDeep` (`internal/trace/collector.go:105`, `RecordsFunctionCalls`);
   default is `TierStandard`, so `--emit-trace jsonl` shows nothing until
   `AILANG_TRACE=deep`. The docs teach the silent path in three places:
   `std/trace.ail:4` ("Works with --emit-trace jsonl"), `docs/docs/guides/traces.md:303-320`
   (spanStart/spanEnd examples with no tier caveat), and the prompt. Three
   acceptable remedies (row 3): document the tier requirement; reclassify explicit
   span events as standard-tier (they are deliberate, not per-call, so the
   superlinear-rendering cost that motivated M-TRACE-TIER-NOT-ENFORCED doesn't
   apply); or emit a distinct span event kind. That last point is the design
   question: spans are semantically *not* function calls and were mapped onto the
   function-call event slot, so the doc should decide the event model, not just
   the wording.

5. **Type names cannot be module-qualified (`Regex.Regex`, `M.Map[...]`)** — **design-doc (rows 3/4).**
   Reproduced: `import std/regex (Regex)` then `r: Regex.Regex` →
   `PAR_UNEXPECTED_TOKEN at :5:23: expected next token to be ), got . instead`.
   The prompt mentions qualified imports only for *values* and name clashes
   (`prompts/v0.16.6.md:641`, `import std/array as A`), and per the reporter
   mentions the type-name restriction only for `S.Set`. Fix could be parser support
   for dotted type names (a grammar/surface change, row 4) or documentation of the
   current rule (which module owns the unqualified name on collision). Row 3/4 →
   design-doc.

6. **`show` on strings inside containers prints them unquoted** — **design-doc (rows 3/4).**
   Reproduced: `show({name: "A", n: 1})` → `{n: 1, name: A}`;
   `show(["a","b"])` → `[a, b]`; top-level `show("A")` → `A`. The behaviour is
   deliberate at one level — `internal/eval/show.go:171` comments "identity for
   strings, no quotes" — but it makes `show([""])` indistinguishable from
   `show(["x"])` at the container boundary, which matters for a language whose
   determinism story leans on byte-exact stdout. It is a Show-format public-surface
   decision (row 4) with multiple acceptable shapes (quote nested strings only vs
   everywhere vs status quo + docs), and it sits in the show-format family that
   iteration 247/248 governing docs already treat as a coordinated surface
   (`design_docs/planned/m-array-show-diverges-run-vs-compile.md` is about backend
   divergence, not quoting — adjacent, not a duplicate). Row 3/4 → design-doc.

## Why one batch file, not six

Per the `mcp-files-0.1.0-friction-batch.md` precedent: the report is one message,
no item is already covered by an existing doc, and items 3 and 2 are small enough
that separate standalone docs would be overhead. Recommendation: one batch doc
(working title M-EXAMPLES-STDLIB-GAPS or per-item split at the planner's
discretion) that (a) scopes item 1 as the headline decision (static
exhaustiveness at `ailang check` — semantics + gate contract + the
`docs/VISION.md` claim it finally makes true, plus the stopgap LIMITATIONS row),
(b) scopes items 4 and 6 as surface decisions (trace span event tier/model;
Show string quoting) with the doc-only stopgaps stated, (c) scopes item 5 as
either parser support or a documented naming rule for type collisions, and
(d) treats items 2 and 3 as small decisions that can ride the same sprint
(`max`/`min` naming + placement under the existing stdlib naming-convention row;
yaml prefix strip with its two pinned expectations).

## Why

Nothing here is a two-line single-file fix once the pinned tests/examples are
counted, and items 1, 4, 5, 6 each have more than one acceptable remedy or touch
a public contract. The reporter did the expensive half — every item is
reproducible at named locations — so the doc-writing cost is decision
selection only, not discovery.

TRIAGE_FILE: design_docs/planned/ailang-core-triage/examples-stdlib-gaps-batch.md
RECOMMEND: design-doc
