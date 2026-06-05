# M-MULTILINE-ADT-PARSER-BUG (REJECTED): Parser does NOT fail on multi-line ADTs

**Status:** REJECTED (2026-06-05)
**Reason:** Core claim false — multi-line ADT (sum type) definitions parse correctly in
  AILANG. Seven layout variants (incl. the exact reproduction from the bug report) all pass
  `ailang check` with "No errors found". The cited crash location `parser.go:234` is the
  graceful `expectPeek` error helper, **not** a panic site — `internal/parser/parser.go`
  contains zero `panic(` calls. There is no bug to fix.
**Target:** N/A (not adopted)
**Priority:** N/A
**Estimated:** 0 days (investigation only)
**Dependencies:** None
**Source:** Two duplicate messages titled *"Bug: Parser fails on multi-line ADT definitions"*
  from the `e2e-test` agent (2026-03-06 19:51 and 20:19 UTC) — synthetic end-to-end test
  fixtures, not real eval failures or human reports.

---

> **⚠️ This task arrived with no feature description.** The coordinator dispatched a
> `design-doc-creator` task (`task-91ae3040`, correlation `msg_20260605_035529_91ae3040`)
> whose payload contained only the instruction "invoke the design-doc-creator skill" — the
> topic did not survive into the agent prompt. With no interactive user reachable (cloud
> mode) and GitHub unreachable from the sandbox, the topic was reconstructed from the local
> agent-message backlog. The single concrete, actionable bug claim in that backlog was the
> multi-line ADT report below. Per the design-doc-creator HARD GATE, it was verified before
> any design work — and it does not reproduce. This doc records that verification so the
> claim is not re-triaged into wasted work.

## The Claim

From the bug message payload (verbatim):

> The AILANG parser crashes when encountering ADT definitions that span multiple lines.
> Reproduce: create a file with `'type Result = Ok(value) | Err(msg)'` split across lines.
> Expected: parse succeeds. Actual: panic in parser.go line 234.

## Verification (HARD GATE — `ailang check`)

**1. The exact reproduction from the report — split across lines:**

```ailang
module test/repro
type Result =
    Ok(value)
  | Err(msg)
export func main() -> () ! {} = ()
```

```
$ ailang check repro.ail
→ Type checking repro.ail...
→ Effect checking...
✓ No errors found!
```

**2. Seven multi-line ADT layout variants — all pass:**

| # | Layout variant | Result |
|---|----------------|--------|
| 1 | Canonical multi-line (`= Ok(int)` newline `\| Err(string)`) | ✓ No errors |
| 2 | Leading `\|` on same line as `=` (`= \| Red \| Green \| Blue`) | ✓ No errors |
| 3 | `=` on its own next line, then leading-pipe constructors | ✓ No errors |
| 4 | Constructor arguments split across lines (`MkPair(\n int,\n string\n)`) | ✓ No errors |
| 5 | Type parameters with multi-line body (`type Tree[a] =\n Leaf \| Node(...)`) | ✓ No errors |
| 6 | Blank line **between** constructors | ✓ No errors |
| 7 | Record-style constructor body split across lines | ✓ No errors |

**3. The cited crash site does not exist as described.**

`internal/parser/parser.go:234` is inside `expectPeek` — the parser's *graceful*
error-recovery helper, which records a diagnostic via `peekError` and returns `false`:

```go
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)   // line ~234: records a diagnostic, does NOT crash
	return false
}
```

`grep -n "panic(" internal/parser/parser.go` returns **zero matches**. The parser cannot
"panic in parser.go line 234" because there is no panic anywhere in that file, and the
function at that line is specifically the structured-error path, the opposite of a crash.

## Frequency / Provenance

The claim's only occurrences are **two identical synthetic messages from the `e2e-test`
agent on 2026-03-06**, three months before this investigation. They are end-to-end pipeline
test fixtures (the same backlog contains dozens of `e2e-test`, `live-test`, `filtertest-*`,
and "smoke test" messages). There are **zero** real eval failures, human reports, or
reproductions attributable to multi-line ADT parsing. The fabricated specificity ("panic in
parser.go line 234") is itself a tell that the payload is canned test text rather than an
observed failure.

## What (if anything) remains

Nothing requires building. Multi-line ADTs are a long-supported, working feature.

**Optional, low-value hardening (not required):** the seven layout variants above could be
committed as a parser regression fixture (e.g. `examples/adt_multiline_layouts.ail` plus a
golden `make verify-examples` entry) so the claim stays false under future parser changes.
Given that all forms already pass and there is no evidence of regression risk, even this is
discretionary.

## Root-Cause Recommendation (process, not language)

This task is itself an instance of a real *process* gap worth noting: the agent-message
auto-triage pipeline (`design_docs/planned/v0_24_0/m-msg-auto-triage-pipeline.md`,
`m-msg-triage-router-sprint-plan.md`) appears able to escalate stale synthetic `e2e-test`
"Bug:" messages into design-doc tasks. A cheap guard — **reproduce a reported parser/compiler
bug with `ailang check` before spawning a design-doc task, and skip messages whose
`from_agent` is a known test harness (`e2e-test`, `live-test`, `filtertest-*`)** — would have
prevented this task entirely. That guard belongs in the triage pipeline, not in the language.

## Axiom Compliance

Not applicable — no change proposed (claim disproved). The feature already works.

## Lesson

Preserved (not deleted) as a record matching the [`m-import-alias.md`](./m-import-alias.md)
and [`m-type-constraints` corrections](../planned/v0_24_0/m-type-constraints.md): a claimed
language limitation that a 10-second `ailang check` retracts. Always verify "AILANG does /
does not support X" before writing the design around it. Here the verification cost ~1 minute
(7 temp files) and saved an entire speculative parser sprint.

## Related Documents

- [`m-import-alias.md`](./m-import-alias.md) — the direct precedent: a false "unsupported"
  language claim retracted by `ailang check`.
- [`m-msg-auto-triage-pipeline.md`](../planned/v0_24_0/m-msg-auto-triage-pipeline.md) — where
  the recommended "verify-and-filter-test-harness-senders before dispatch" guard belongs.
- [`m-msg-triage-router-sprint-plan.md`](../planned/v0_24_0/m-msg-triage-router-sprint-plan.md)
  — triage routing that dispatched this topic.
