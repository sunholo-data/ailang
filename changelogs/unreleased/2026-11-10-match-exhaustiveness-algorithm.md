### Added — match-exhaustiveness algorithm package (internal, not yet wired) (2026-11-10)

`internal/types/exhaust` implements Maranget's usefulness algorithm over an abstract
pattern matrix: exhaustiveness, a concrete missing pattern rendered in surface syntax
(`Blue`, `Some(Err(_))`, `[]`, `[_, ..._]`, `(false, false)`, `{a: false, ...}`,
`_ (any int other than 0, 1)`), redundant-arm detection, and a 10^6-step budget that
fails closed. It covers every pattern form the parser produces — constructors (nested),
`[]` / fixed-length / spread / `h :: t` lists, tuples, unit, records (fields = union
named in the column), bool, and int/float/string literals as infinite domains — and
guarded arms never count toward coverage. An ADT's full constructor list comes from a
`Universe` interface, so the qualified type identity of
m-v1-nominal-type-identity.md plugs in when it lands.

Nothing calls it yet: `ailang check` accepts exactly what it accepted before. Wiring
it into the type checker as `TC_MATCH_002` (error) / `TC_MATCH_003` (warning) is
milestone M2, which waits on nominal type identity. Evidence: table tests per pattern
form, a 4,000-trial differential property test against the evaluator's own matcher
(exhaustive ⇔ every enumerated value matched; every witness unmatched; redundant ⇔
unreachable), and benchmarks — a 1,000-arm string-literal match in ~2 ms and a
12-column bool-tuple match with 2^10 rows in ~29 ms (budgets 10 ms / 100 ms). Milestone
M1 of design_docs/planned/v1_0_0/m-v1-match-exhaustiveness.md (#2146).
