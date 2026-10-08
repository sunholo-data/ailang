# REST and MCP disagree on numeric param binding — need one type-directed binder

- **Date**: 2026-10-08
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `1631`, `expected int arguments`, `MCP-UNIT-PARAM-BINDING`, `CallPreserveFloats`, `paramZero`; `ls design_docs/planned/ailang-core-triage/` (no numeric-binding row); `design_docs/implemented/v0_23_0/m-mcp-unit-param-binding.md` (adjacent but does not rule on numeric typing)
- **Estimate**: n/a (design-doc; spans `internal/apiserver/argresolve.go`, `internal/apiserver/handler.go`, `internal/apiserver/mcp.go`, `internal/apiserver/param_zero.go`, `internal/embed/embed.go` + cross-surface tests)

**Why.** The report's mechanism checks out in code. The two surfaces call the engine through different conversion modes and neither is type-directed:

- **REST** (`internal/apiserver/routes_dispatch.go`, `s.engine.Call` at ~:143, comment at :138-141): `Call` deliberately converts JSON-decoded whole `float64` to `IntValue` "when they fit" — so `{"x":3}` for `x: float` binds `IntValue(3)`, and `x / 2.0` becomes int÷float, which falls through both numeric branches of the binop shim (`internal/eval/eval_operations.go` ~:404/:479) into `expected int arguments`. The omitted-`@optional` case fails the same way: `paramZero` (`internal/apiserver/param_zero.go`) returns `float64(0)` for a non-record float param, which `Call` then re-reads as `IntValue(0)`.
- **MCP** (`internal/apiserver/mcp.go` ~:452): `CallPreserveFloats` keeps every JSON number a `FloatValue` — correct for float params (the REST failures above pass over MCP) but the mirror bug for `int` params (issue #1631): an integral number arrives as `FloatValue` and int builtins refuse it.

The precedent doc `m-mcp-unit-param-binding.md` (v0.23.0) fixed the omitted-param class but is type-agnostic about zero values and never reconciled the two numeric-conversion modes; `routes_dispatch.go`'s "Use Call (not CallPreserveFloats)" was chosen to fix int record fields from cross-package calls, which is exactly why a one-sided flip (all-REST → CallPreserveFloats or all-MCP → Call) just swaps which type breaks. The real fix the report asks for — bind by the declared param type at the binder (int ← integral JSON number, else refuse; float ← any JSON number; typed zeros for omitted `@optional`) and hand the engine already-typed values, or give it a no-reconversion entrypoint — is a semantics change to two public surfaces (REST `/api/{module}/{func}` and MCP `tools/call`), touches at least five files, and has genuinely debatable placement (binder-level coercion vs. a new typed `Engine.CallTyped`; refusal error shape for fractional→int; compat for callers who relied on REST coercing `3` to int). That is row 3/4/5 territory: design-doc, not direct-fix.

Real-world impact cited by the reporter and verifiable in-repo: docparse declares `mcpParse`'s `maxChars` as a STRING solely to survive the MCP int bug, and any `float`-declared param (e.g. the createUpload `count` workaround from #1631) is broken over REST. Tests should cover int/float × MCP/REST × given/omitted/whole/fractional.

Note: existing typed-zero infrastructure in `param_zero.go` (record fields bind `int`/`FloatValue` correctly) shows the engine already supports type-directed values; the gap is that scalar params are not run through it and both call modes still re-convert raw `float64`s.