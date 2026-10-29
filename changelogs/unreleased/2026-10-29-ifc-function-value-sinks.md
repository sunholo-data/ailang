### Fixed — IFC sink requirements now travel with function values (#1719) (2026-10-29)

**Security fix (P0).** The static information-flow check enforced `T{not ℓ}`
sinks and positive `T<ℓ>` parameter labels only when the callee was written as
the sink's own name. Any other route to the same function dropped the
requirement silently, so `<email>` data reached a `{not email}` sink through:
a `let` alias (`let g = gate in g(x)`), a higher-order parameter
(`apply1(gate, x)`), an annotated binding or parameter type
(`f: string{not email} -> string`), a record field, a list or tuple element, an
`if` branch, a returned or composed function, a function literal, or a wrapper
(`wrap(y) = gate(y)`, `\y. gate(y)`). All named bypass shapes are now
rejected with the existing `sink_refinement` / `param_label_cover` kinds; the
message names the path the value took (`of gate (via apply1 → f)`). Direct-call
diagnostics are unchanged byte for byte.

How: the IFC pass's abstract value carries a callable (what applying the value
requires of each argument) beside its label, and Checks A/C run at every
application. Module functions, lambdas and function literals get summaries:
what each does with the function values it receives (applied-at) and which
requirements flow from its unannotated parameters into a sink (inferred
demands), iterated to a fixed point. Imported higher-order functions (e.g.
`std/list` `map`) take a conservative fallback: a sink passed to them is checked
against the other arguments' labels. Nothing changes in HM types, CoreTI,
codegen or the iface schema. **Programs that compile today can be rejected**:
each such program passes labelled data to a sink through an alias, a HOF or a
wrapper, i.e. it leaked. Cross-module sinks remain out of scope (#1134).

Design: `design_docs/implemented/v0_55_1/m-ifc-function-value-sinks.md`. Examples:
`examples/runnable/secrets/sink_function_value.ail` (rejected) and
`sink_function_value_ok.ail` (accepted). Guide: "Sinks as values" in
`docs/docs/guides/ifc-labels.mdx`.

Silent alias/probe/replay cutoffs are removed. Cycles use repeated analysis states; any defensive resource limit produces an ordinary `ifc_analysis_limit` refusal. The original concrete diagnostic environment and Check B behavior remain unchanged.
