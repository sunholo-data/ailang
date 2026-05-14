# M-STDLIB-URL-ENCODE: URL form-encoding in std/net

**Status**: Planned
**Target**: v0.20.0 (size permitting, candidate for v0.19.2 patch — see Decisions)
**Priority**: P1 (Medium — paper cut on every OAuth/webhook integration)
**Estimated**: 3–4 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure function over input string; identical input always yields identical output (RFC 3986). |
| A2: Replayability | 0 | No trace impact; no effect row. |
| A3: Effect Legibility | +1 | Explicitly pure (no `! {Net}`), reinforcing the separation between byte-shaping and effect-bearing IO. |
| A4: Explicit Authority | 0 | No new authority granted. |
| A5: Bounded Verification | +1 | Total function with a closed RFC-3986 spec; no unbounded behavior. |
| A6: Safe Concurrency | 0 | Stateless. |
| A7: Machines First | +1 | Removes the discovery-time stumbling block agents hit when synthesizing OAuth/webhook clients (LinkedIn demo lost 3 round-trips to 400/401 on this). |
| A8: Minimal Syntax | +1 | No new syntax; two regular function bindings. |
| A9: Cost Visibility | 0 | O(n) on string length, allocation-bounded. |
| A10: Composability | +1 | Composes with `httpRequest` body, `httpPost`, and any caller building `application/x-www-form-urlencoded` payloads. |
| A11: Structured Failure | 0 | Cannot fail on valid UTF-8 string input (Go's `url.QueryEscape` is total). |
| A12: System Boundary | +1 | Makes the bytes-on-the-wire shape of an HTTP form body fully visible at the call site instead of hand-rolled inside user code. |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check
- [x] A1: No nondeterminism — pure RFC-3986 transformation.
- [x] A3: No hidden effects — function does not carry `! {Net}`.
- [x] A4: No ambient access.
- [x] A7: Optimized for machine code synthesis, not human ergonomic preference.

## Problem Statement

LinkedIn OAuth2 token exchange (and every other `application/x-www-form-urlencoded` API) requires the client to percent-encode body parameters. Today `std/net` and `std/string` ship no helper for this.

**Current State** (per agent feedback msg `9e25539f` from `demos/linkedin`):
- A real OAuth integration hit `400 invalid_request "client_secret missing"` because `client_secret` contained `=`, `+`, and `/`.
- The agent worked around it by hand-rolling a 6-line `formUrlEncode` via chained `std/string.replace` calls. The chain is **order-sensitive** (`%` must be encoded first to avoid double-encoding) — a non-obvious correctness trap.
- Discovery cost: ~3 rounds of `400`/`401` errors before the workaround was found.

**Impact:**
- Any agent synthesizing OAuth, webhook, or traditional form-POST clients hits this.
- The asymmetry with `std/bytes.toBase64` (present) is a discoverability dead-end — agents reasonably assume the URL-encode counterpart exists and silently produce broken bodies when it doesn't.
- This is a **systemic stdlib gap**, not a one-off bug. Severity: low-medium.

## Goals

**Primary Goal:** Expose RFC-3986 percent-encoding for query/form bodies in `std/net` so OAuth and form-POST flows are one function call away.

**Success Metrics:**
- `urlEncode("a=b/c+d e")` returns `"a%3Db%2Fc%2Bd+e"` (or `%20` for space — see Decisions).
- `urlEncodeForm([{name: "client_id", value: "x"}, {name: "client_secret", value: "a/b=c"}])` returns `"client_id=x&client_secret=a%2Fb%3Dc"`.
- The LinkedIn OAuth demo can replace its 6-line hand-rolled `formUrlEncode` with one stdlib call.
- All 7 OAuth-style edge-case inputs covered by unit tests pass (see Testing).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Home module (`std/net` vs `std/string`) | Wrong home = future migration breaks user imports | human | design | med |
| D2: KV shape — `{name, value}` record vs `(string, string)` tuple | Sets precedent for future form/header builders; must match existing `httpRequest` headers signature | human | design | med |
| D3: Space encoding — `+` (form) vs `%20` (RFC 3986) | Wrong default silently breaks one of two use cases (query string vs form body) | human | design | low |
| D4: Release vehicle — v0.20.0 minor vs v0.19.2 patch | Patch ships within days; minor ships on regular cadence | human | design | low |
| D5: Surface — two functions (`urlEncode` + `urlEncodeForm`) vs only the primitive | Only-primitive forces every caller to re-derive the join logic | agent | design | low |

### Design Freeze

Before implementation begins:
- [x] **D1**: `std/net` — sits next to `httpPost`/`httpRequest`, the actual call site. Rejected `std/string` because the operation is URL-spec-specific, not a general string transform.
- [x] **D2**: `List[{name: string, value: string}]` — matches the existing `headers` parameter shape in [std/net.ail:18,133](std/net.ail#L18). Rejected tuples for consistency.
- [x] **D3**: `urlEncode` uses `%20` for space (RFC 3986, safe for both query strings and form bodies). `urlEncodeForm` uses `+` for space inside values (standard `application/x-www-form-urlencoded` per WHATWG URL spec, what every server expects). Documented at the call site.
- [x] **D5**: Ship both functions. The primitive is needed for one-off values (auth headers); the form builder is needed for the common case.
- [ ] **D4**: Pick release vehicle (default v0.20.0; flip to v0.19.2 only if release-manager wants the user-facing patch sooner).

## Solution Design

### Overview

Two new pure builtins backed by Go's `net/url` package, exposed via `std/net`.

| AILANG function | Builtin | Go backing |
|-----------------|---------|------------|
| `urlEncode(s: string) -> string` | `_net_url_encode` | `url.QueryEscape` + `+`→`%20` (RFC 3986: space → `%20`) |
| `urlEncodeForm(params: List[{name: string, value: string}]) -> string` | `_net_url_encode_form` | `url.Values.Encode()` (form: space → `+`) |

Both are pure (no effect row), total on valid UTF-8 input.

### Architecture

**Components:**
1. **Builtin registration** ([internal/builtins/net.go](internal/builtins/net.go)) — Two `RegisterEffectBuiltin` calls with `IsPure: true, Effect: ""`, following the exact pattern of `_bytes_to_base64` at [internal/builtins/bytes.go:136-167](internal/builtins/bytes.go#L136-L167).
2. **Type signatures** — `string -> string` for the primitive; `List[{name: string, value: string}] -> string` for the form builder, matching the existing `headers` parameter at [std/net.ail:133](std/net.ail#L133).
3. **AILANG wrapper** ([std/net.ail](std/net.ail)) — Two thin `export func` wrappers calling the underscore-prefixed builtins.

### Implementation Plan

**Phase 1: Builtins** (~1 hour)
- [ ] Add `registerNetUrlEncode()` to [internal/builtins/net.go](internal/builtins/net.go) — implementation calls `url.PathEscape(s)`.
- [ ] Add `registerNetUrlEncodeForm()` to [internal/builtins/net.go](internal/builtins/net.go) — implementation walks the `*eval.ListValue` of records, builds a `url.Values`, returns `values.Encode()`.
- [ ] Register both in the package `init()`.
- [ ] Verify pipeline tests still pass: `make test`.

**Phase 2: Stdlib wrappers** (~30 min)
- [ ] Add `urlEncode` and `urlEncodeForm` exports to [std/net.ail](std/net.ail) with doc comments calling out the space-encoding asymmetry.
- [ ] `make verify-examples`.

**Phase 3: Tests + example + docs** (~1.5 hours)
- [ ] Unit tests in `internal/builtins/net_test.go` (or `url_encode_test.go`): 7 OAuth-style inputs.
- [ ] `examples/url_encode.ail` — minimal OAuth-style form-POST body construction, ~15 LOC.
- [ ] CHANGELOG entry under `changelogs/v0.10-current.md` (current cycle file).
- [ ] `make ci`.

### Files to Modify/Create

**New files:**
- `internal/builtins/net_url_encode_test.go` — ~120 LOC (or extend an existing `net_test.go` if present).
- `examples/url_encode.ail` — ~15 LOC.

**Modified files:**
- `internal/builtins/net.go` — +~100 LOC (two `register*` + two `make*Type` + two `*Impl` functions).
- `std/net.ail` — +~25 LOC (two `export func` + doc comments).
- `changelogs/v0.10-current.md` — +~10 LOC.

## Examples

### Example 1: OAuth2 token exchange (the motivating case)

**Before** (from `demos/linkedin/scripts/oauth_server.ail`):
```ailang
func formUrlEncode(s: string) -> string {
  let s1 = replace(s, "%", "%25");
  let s2 = replace(s1, "+", "%2B");
  let s3 = replace(s2, "/", "%2F");
  let s4 = replace(s3, "=", "%3D");
  let s5 = replace(s4, "&", "%26");
  let s6 = replace(s5, " ", "%20");
  s6
}

let body =
  "grant_type=authorization_code" ++
  "&code=" ++ formUrlEncode(code) ++
  "&client_id=" ++ formUrlEncode(clientId) ++
  "&client_secret=" ++ formUrlEncode(clientSecret) ++
  "&redirect_uri=" ++ formUrlEncode(redirectUri)
```

**After:**
```ailang
import std/net (urlEncodeForm)

let body = urlEncodeForm([
  {name: "grant_type",    value: "authorization_code"},
  {name: "code",          value: code},
  {name: "client_id",     value: clientId},
  {name: "client_secret", value: clientSecret},
  {name: "redirect_uri",  value: redirectUri}
])
```

### Example 2: Single-value encode (auth header, signed-URL parameter)

```ailang
import std/net (urlEncode)

let signed = "https://example.com/upload?token=" ++ urlEncode(authToken)
```

## Success Criteria

- [ ] `urlEncode` and `urlEncodeForm` callable from a `.ail` file via `import std/net`.
- [ ] Round-trip test: `urlEncodeForm([{name: "k", value: "a=b&c"}])` equals `"k=a%3Db%26c"` exactly.
- [ ] Space behavior is documented and tested: `urlEncode(" ")` returns `"%20"`; `urlEncodeForm([{name: "x", value: " "}])` returns `"x=+"`.
- [ ] `demos/linkedin/scripts/oauth_server.ail` updated to use the stdlib version (out of scope for this sprint but tracked).
- [ ] All tests passing; `make ci` green.
- [ ] CHANGELOG entry written.
- [ ] One example file (`examples/url_encode.ail`) added and runs clean.

## Testing Strategy

**Unit tests** (`internal/builtins/net_url_encode_test.go`):
- `urlEncode` on each of: `=`, `+`, `/`, `&`, ` ` (space), `%`, multibyte unicode (`"café"`), empty string.
- `urlEncodeForm` on: empty list → `""`; single pair; multi-pair preserves order; values with all unsafe chars; empty value; UTF-8 keys/values.
- Regression: the exact LinkedIn-style payload from the feedback message round-trips correctly.

**Integration tests:**
- Add the new example to `verify-examples`.
- Smoke: import `std/net`, call both functions in one `.ail` program.

**Manual:**
- Run the example with `ailang run examples/url_encode.ail` and confirm stdout.

## Deferred Decisions

- **Reverse direction (`urlDecode`/`urlDecodeForm`)** — agent may add later; out of scope here (no current pain point). If added, mirror this design.
- **Whether to also expose under `std/string`** — agent may not. If a future caller wants it there, alias rather than reimplement.
- **Streaming/large-body encoding** — agent may defer; OAuth bodies are <1 KB in practice.

## Non-Goals

- **URL parsing / reconstruction** — out of scope; this is encoding only.
- **`urlDecode`** — symmetric counterpart deferred (no reported pain).
- **Query-string builder that takes a record** (e.g. `urlEncodeForm({client_id: "x"})`) — record-key iteration is not currently a first-class operation in AILANG; sticking with the `List[{name, value}]` shape that matches `httpRequest` headers.
- **Migrating the LinkedIn demo** — tracked separately; this sprint ships the stdlib, not the migration.

## Timeline

**Single sprint, ~3–4 hours total:**
- Hour 0–1: Phase 1 (builtins).
- Hour 1–1.5: Phase 2 (wrappers).
- Hour 1.5–3: Phase 3 (tests, example, CHANGELOG).
- Hour 3–4: `make ci`, fix any issues, sprint-evaluator pass.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Space-encoding asymmetry (`+` vs `%20`) confuses future callers | Med | Spell it out in both function doc comments and the example file; cover both in tests. |
| `url.Values.Encode()` sorts keys alphabetically, but order matters for some signing schemes (AWS SigV4, etc.) | Low | Use `url.Values.Encode()` for the common case; document the sort. If signing-grade ordering is needed later, add `urlEncodeFormOrdered` then. |
| Builtin name collision with future planned URL APIs | Low | Namespace via `_net_url_*` prefix on builtins; keep AILANG-level names un-prefixed. |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_3/M-R6_clock_net_effects.md](design_docs/implemented/v0_3/M-R6_clock_net_effects.md) — Original `std/net` design; this extends it with pure helpers.
- [design_docs/implemented/v0_11_0/m-std-string-perf-sprint-plan.md](design_docs/implemented/v0_11_0/m-std-string-perf-sprint-plan.md) — Pattern for shipping focused stdlib improvements.

**Planned (check for overlap):**
- [design_docs/planned/v0_21_0/m-stdlib-html-streaming.md](design_docs/planned/v0_21_0/m-stdlib-html-streaming.md) — Adjacent (HTML escaping vs URL escaping); no overlap, but worth aligning naming conventions.

## References

- [Design Axioms](/docs/references/axioms)
- Feedback msg `9e25539f` from `demos/linkedin` (2026-05-12): `[stdlib] URL form-encoder missing from std/net (OAuth pain)`.
- Go [`net/url`](https://pkg.go.dev/net/url) — `PathEscape`, `Values.Encode`.
- RFC 3986 §2.1 (percent-encoding); WHATWG URL Living Standard (form encoding).
- Builtin registration template: [internal/builtins/bytes.go:136-167](internal/builtins/bytes.go#L136-L167).
- KV-record precedent in stdlib: [std/net.ail:18,133](std/net.ail#L18).

## Future Work

- `urlDecode` / `urlDecodeForm` (when a real use case appears).
- Ordered/signing-grade form encoding (`urlEncodeFormOrdered`) for AWS SigV4, OAuth1.0a, etc.
- Once shipped, audit other stdlib gaps surfaced by the LinkedIn demo (msg `5963a6a5` stderr flag, msg `0c4a790b` trailing-slash redirect — separate docs).

---

**Document created**: 2026-05-14
**Last updated**: 2026-05-14

DESIGN_DOC_PATH: design_docs/planned/v0_20_0/m-stdlib-url-encode.md
