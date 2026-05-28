# M-MCP-UNIT-PARAM-BINDING: Reject omitted MCP tool params instead of binding Unit

**Status**: Implemented (v0.23.0, 2026-05-28; commit f2253d88). @route path audited → confirmed-safe (type-aware zero-pad, not nil→Unit); regression guard added.
**Target**: v0.23.0
**Priority**: P1 (Medium-High) — breaks the documented first-run agent auth flow for any MCP tool with a guard on an omittable param; observed in prod on docparse MCP skill
**Estimated**: 0.5 day (~40 LOC + tests; the fix is localized to one dispatch function)
**Dependencies**: None.
**Bug Report**: msg_..._b08617cb (from sunholo/docparse via `cli` inbox, 2026-05-27, against deployed ailang-api 0.8.1)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism change. |
| A2: Replayability | 0 | No trace-shape change. |
| A3: Effect Legibility | 0 | No effect change. |
| A4: Explicit Authority | 0 | No authority change. |
| A5: Bounded Verification | 0 | No analysis-surface change. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | An MCP client (an agent) that omits a required param currently gets an internal `_str_len: expected String, got *eval.UnitValue` crash with no actionable message. The fix returns a structured "missing required parameter: X" — a machine-actionable error the agent can correct from, instead of an opaque stdlib panic. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | 0 | No cost change. |
| A10: Composability | 0 | No composition change. |
| A11: Structured Failure | +1 | Core of the fix: convert an unstructured internal crash (stdlib panic on Unit) into a structured, typed dispatch-layer error returned cleanly to the MCP client. |
| A12: System Boundary | +1 | The MCP boundary is exactly where external (client-supplied) input crosses into the AILANG runtime. Validating that boundary — rejecting incomplete tool calls before they reach the engine — is the boundary-crossing explicitness the axiom demands. The current code lets an incomplete external request leak past the boundary and crash deep in stdlib. |

**Net Score: +3** → **Decision: ✅ Proceed**

### Hard Violation Check
- [x] A1: No implicit nondeterminism
- [x] A3: No hidden side effects
- [x] A4: No ambient authority
- [x] A7: Optimises for machine-actionable errors

## Problem Statement

When an MCP `tools/call` omits a declared (non-optional) parameter, the dispatcher binds that parameter to Go `nil`, which the engine converts to `Unit`. The AILANG function then runs with a `Unit` where it expects a `string` (or any other type) and crashes deep in stdlib with a non-actionable message.

### The bug, exactly

[internal/apiserver/mcp.go:201-205](../../../internal/apiserver/mcp.go) in `makeToolHandler`:

```go
// If no "args" key and we have param names, resolve named params.
if len(args) == 0 && len(paramNames) > 0 {
    args = make([]any, len(paramNames))
    for i, name := range paramNames {
        args[i] = argMap[name]   // ← BUG: omitted key → nil → binds to Unit
    }
}
```

`argMap[name]` on a missing key returns Go's zero value `nil`. That `nil` flows into `args[i]`, then into `engine.CallPreserveFloats(...)`, which binds it to `*eval.UnitValue`. The AILANG function executes with Unit in a string slot.

### Repro (from the bug report)

Tool signature:
```ailang
export func mcpParse(filepath: string, outputFormat: string, apiKey: string, requestId: string) -> string ! {...}
```

`tools/call` with `{"filepath":"sample_docx_formatting","outputFormat":"markdown"}` — `apiKey` and `requestId` omitted — returns:
```
isError=true, content: "function call failed: _str_len: expected String, got *eval.UnitValue"
```

The handler's first line is `if length(apiKey) == 0 then ...` (`std/string.length` → `_str_len`). The omitted `apiKey` arrives as Unit, so `_str_len` crashes **before** the guard can run.

**Smoking gun:** passing `apiKey="dp_anything"` works (returns INVALID_API_KEY cleanly); omitting it crashes. So a declared `string` param is bound to Unit purely because the client omitted it.

### Systemic scope (audit, per CLAUDE.md §3)

This is **not a string-specific bug**. The faulty line is type-agnostic: ANY omitted parameter of ANY type binds to `nil`→Unit. So:
- Omitted `string` param → `_str_len` / any string builtin crashes on Unit (the reported case)
- Omitted `int` param → `_add` / arithmetic builtins crash with `expected Int, got Unit`
- Omitted `bool` param → `if` condition or bool builtins crash
- Omitted record/list param → field-access / list builtins crash

Any MCP tool whose handler touches an omittable param before a nil-guard (which AILANG can't express anyway — no Unit-detection or null-safe ops) will crash. The fix must therefore be **type-agnostic at the dispatch layer**, not a string-specific special-case.

### Impact

Breaks the documented first-run agent auth flow for the docparse MCP skill: an unauthenticated `mcpParse` call is *supposed* to return `AUTH_REQUIRED` (the handler explicitly builds that response), but instead crashes with an internal error. This is the **exact path a brand-new skill user hits** on their first call.

## Goals

**Primary goal:** A declared MCP tool param that the client omits produces a **structured "missing required parameter" error returned to the client**, never a `nil`→Unit binding that crashes stdlib.

**Success metrics:**
1. `tools/call mcpParse {"filepath":"x","outputFormat":"markdown"}` (apiKey + requestId omitted) returns a clean MCP error: `missing required parameter(s): apiKey, requestId` — `isError=true`, no stdlib crash.
2. Passing all params still works unchanged.
3. The fix is type-agnostic — verified with an omitted `int` param crashing the same way pre-fix and erroring cleanly post-fix.
4. The error message names *which* params are missing so the agent can self-correct.

## Solution Design

### Decision: reject (not default)

Two candidate behaviors for an omitted declared param:
- **(a) Reject** with a structured "missing required parameter: X" error.
- **(b) Default** to a type-appropriate zero (`""` for string, `0` for int, etc.).

**Decision: reject.** Rationale:
- AILANG has no optional-parameter or default-value syntax today, so **every declared param is semantically required**. Silently defaulting a missing required param to `""` would mask client bugs and produce subtly-wrong downstream behavior (e.g. `mcpParse` with `apiKey=""` proceeds to an INVALID_API_KEY path rather than the intended AUTH_REQUIRED path — wrong vs. the explicit "you forgot apiKey" the agent actually needs).
- Reject is type-agnostic and needs no per-type defaulting logic.
- Reject matches MCP spec intent: tool input schemas declare `required` params, and a missing required param is a client error, not something the server silently fills in.

If/when AILANG gains optional-param syntax, defaulting for *genuinely optional* params becomes a clean follow-up (see Out of Scope).

### Implementation

`internal/apiserver/mcp.go::makeToolHandler`, replace the omit-blind loop:

```go
if len(args) == 0 && len(paramNames) > 0 {
    var missing []string
    args = make([]any, len(paramNames))
    for i, name := range paramNames {
        v, present := argMap[name]
        if !present || v == nil {
            missing = append(missing, name)
            continue
        }
        args[i] = v
    }
    if len(missing) > 0 {
        return mcpError(fmt.Sprintf(
            "missing required parameter(s): %s", strings.Join(missing, ", "),
        )), nil
    }
}
```

Minimal, type-agnostic fix: detect omitted-or-null declared params, collect their names, return a structured MCP error *before* calling the engine. No engine change, no stdlib change.

The `!present || v == nil` distinction:
- `!present` — key absent from the JSON object entirely (the reported case)
- `v == nil` — key present with explicit JSON `null` (also rejected, same reasoning)

### Files to modify

| File | Change | LOC |
|---|---|---|
| `internal/apiserver/mcp.go` | omit-detection in `makeToolHandler` arg loop | +12 |
| `internal/apiserver/mcp_test.go` (or `feedback_tool_test.go` sibling) | omitted-string → structured error; omitted-int → same; all-present → unchanged; explicit-null → rejected | +35 |
| `docs/docs/guides/serve-api.md` | note: omitted MCP tool params return a structured error (no optional-param support yet) | +8 |
| `changelogs/v0.10-current.md` | Entry | +6 |

## Conflict Surface

Touches `internal/apiserver/mcp.go` — the request-dispatch path, not the parser/typechecker/codegen. No language-semantics surface. The relevant conflict consideration is the **legacy positional `{"args": [...]}` path** (mcp.go:195-198): unaffected — it only runs when an `args` key is present, and the new omit-check is in the mutually-exclusive named-params branch (`len(args) == 0`). Programs that MUST still work post-change:
1. `tools/call` with all named params present → unchanged (binds all, calls engine)
2. `tools/call` with legacy `{"args": ["a","b"]}` positional form → unchanged (never enters the named branch)
3. `tools/call` for a zero-param tool → unchanged (`len(paramNames) == 0`, loop skipped)
4. `tools/call` omitting a param → NOW returns structured error (was: crash) — the intended behavior change

## Success Criteria

- [ ] `tools/call` omitting `apiKey` returns `missing required parameter(s): apiKey, requestId` with `isError=true` — no `_str_len` crash
- [ ] All-params-present call still succeeds (regression check)
- [ ] Legacy positional `{"args":[...]}` form still works (regression check)
- [ ] Type-agnostic: a test tool with an omitted `int` param returns the same structured error (proves it's not a string special-case)
- [ ] Explicit JSON `null` for a param is also rejected (not bound to Unit)
- [ ] Test runs deterministic at `-count=20`
- [ ] `docs/docs/guides/serve-api.md` documents the behavior
- [ ] CHANGELOG entry
- [ ] Reply to the bug report (msg_..._b08617cb) confirming the fix + the reject-vs-default decision

**Determinism note:** the fix iterates `paramNames` (a stable slice), appending to `missing` in declaration order — so the error message is deterministic without a sort. The `-count=20` test guards against any accidental switch to map iteration.

## Out of Scope

- **Optional-parameter / default-value syntax in AILANG** — the bug report notes AILANG offers "no optional params / default values / Unit detection / null-safe string ops." A genuine language-design gap, but a much larger feature (parser + typechecker + codegen). This doc scopes to the dispatch-layer boundary fix that makes the *current* all-params-required contract behave correctly. Optional params would be a separate design doc; once they exist, the dispatch layer can default genuinely-optional omitted params instead of rejecting them.
- **Null-safe string ops / Unit-detection builtins** — also language-level, out of scope for this boundary fix.

## AUDIT follow-up (do during implementation)

The same `argMap[name]` omit-blind binding may exist in the **`@route` HTTP handler path** (`internal/apiserver/routes.go::callFunction()`, which also does named-param binding + has the `_headers` injection). A missing `@route` body/query param could hit the identical nil→Unit crash. **Action:** grep `routes.go` for the param-binding loop during implementation; if it has the same pattern, fix both in this sprint (unified fix per CLAUDE.md §3). If it uses a different (safe) binding, note it and scope this doc to MCP only.

## Related Documents

- **MCP dispatch**: [internal/apiserver/mcp.go](../../../internal/apiserver/mcp.go) — `makeToolHandler` (the fix site), `buildNamedInputSchema` (where param schemas/types are available if we later want type-aware defaulting).
- **API server rules**: [.claude/rules/api-server.md](../../../.claude/rules/api-server.md) — `isExposed()` filtering, MCP tool-name generation.
- **Sibling dispatch path to audit**: `internal/apiserver/routes.go::callFunction()` (the `@route` HTTP param-binding path).

---

**Document created**: 2026-05-28
**Last updated**: 2026-05-28
