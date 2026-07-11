# m-stdlib-url-parse — RFC-3986 URL Parsing via Go net/url

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (v1.0 bar clause 4 — ORCHESTRATION FLAGSHIP, strategy R7)
**Estimated**: ~1 day (0.5d Go builtins+tests, 0.25d std/net additions+examples, 0.25d docs+integration)
**Dependencies**: None (purely additive — new pure builtins + new `std/net` exports)

**Mission**: [v1-mission.md](../../v1-mission.md) queue item #12. The v1.0 bar clause 4 mandates
"**linear-time regex + URL-parse builtins** (both verified absent — an orchestration 1.0 without
them is a credibility hole)." The regex half shipped as
[m-stdlib-regex](../../implemented/v0_30_0/m-stdlib-regex.md) (iter 11). This doc covers the
**URL-parse** half: parse/split a URL into scheme/host/path/query/fragment. It is the *inverse
complement* of the already-shipped `urlEncode`/`urlEncodeForm`
([m-stdlib-url-encode](../../implemented/v0_19_2/m-stdlib-url-encode.md)) — encoding builds URLs,
parsing takes them apart.

---

## Problem Statement

AILANG can **build** URL-encoded strings (`urlEncode`, `urlEncodeForm` in `std/net`, v0.20.0) but
cannot **take a URL apart**. Verified absent at `dev` (v0.29.2 stdlib):

```
$ grep -rn "_net_url_parse\|parseUrl\|parseQuery" internal/ std/   # → 0 matches in std/net surface
$ grep -rn "url_parse\|urlParse"                     internal/ std/   # → nothing
```

The current `std/net` URL surface is encode-only and pure:

- `urlEncode(s: string) -> string` — RFC-3986 percent-encode one value.
- `urlEncodeForm(params: List[{name, value}]) -> string` — build a form body.

There is no way to:

- split a URL into its **scheme / host / port / path / query / fragment** components,
- read query parameters back out of a `?a=1&b=2` string into `{name, value}` pairs
  (the inverse of `urlEncodeForm`),
- extract the host to route on, the path to dispatch on, or a query param to branch on.

**Current State:**
- **0** URL-parsing functions in the stdlib. An agent handed a URL (an OAuth redirect, a webhook
  callback, an LLM-produced link, a paginated `next` URL) must hand-roll a `std/string.split`
  chain — order-sensitive, RFC-ignorant, and a correctness trap (percent-decoding, `://`,
  `?` vs `#` precedence, optional `:port`).
- Every mainstream orchestration/scripting language ships URL parsing in its stdlib
  (Python `urllib.parse.urlparse`, JS `new URL()`, Go `net/url.Parse`). Its absence — while
  *encoding* is present — is an asymmetric discoverability dead-end.

**Impact:**
- **Who**: AI authors writing web/API orchestration (the v1.0 headline persona). Parsing the URL
  is step zero of consuming any redirect, webhook, or link.
- **How significant**: a **credibility hole** in the 1.0 orchestration claim (bar clause 4,
  explicit — regex *and* URL-parse are named release gates). Not a nice-to-have.

**Why wrap Go `net/url` specifically**: `net/url.Parse` is the reference RFC-3986 parser — battle-
tested, handles the edge cases (userinfo, IPv6 host brackets, percent-decoding, `Port()` vs
`Host`), and is *pure* (no I/O, no capability). Wrapping it makes this a ~1-day builtin instead of
a from-scratch parser with its own correctness surface. Exactly the "wrap-don't-build" decision
that kept regex at 2 days.

---

## Goals

**Primary Goal:** Extend `std/net` with **pure** URL-parsing functions backed by Go `net/url`,
giving AILANG authors `parseUrl` (URL → structured record) and `parseQuery` (query string →
`{name, value}` pairs) — closing v1.0 bar clause 4's URL-parse half.

**Success Metrics:**
- `std/net` gains `parseUrl` + `parseQuery`; both `ailang check`-clean (verified below).
- `parseUrl("https://user@host:8443/a/b?q=1#frag")` returns a `Url` record with
  `scheme="https"`, `host="host"`, `port="8443"`, `path="/a/b"`, `query="q=1"`, `fragment="frag"`.
- `parseQuery("q=hello%20world&r=2")` returns `[{name:"q", value:"hello world"}, {name:"r", value:"2"}]`
  (percent-**decoded** values — the round-trip inverse of `urlEncode`).
- Malformed input surfaces as `Err(message)` from `parseUrl`, never a panic (CLAUDE.md CP2).
- Round-trip property holds for the common case: `parseQuery(urlEncodeForm(pairs))` recovers the
  pairs (modulo `urlEncodeForm`'s documented key-sorting).
- ≥2 runnable examples in `examples/` + `make verify-examples` green.
- Additive only — every existing `std/net` program (httpRequest, urlEncode round-trips) still
  type-checks and runs unchanged.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Wrap Go's `net/url` rather than hand-roll a parser** | `net/url.Parse` *is* the reference RFC-3986 parser — correctness is inherited from the Go stdlib, zero parser code to write/verify. Keeps this a ~1-day item, not a multi-day parser project with its own edge-case surface (IPv6 brackets, userinfo, percent-decoding). | human (ratify) / agent (implement) | design | high |
| **Extend `std/net` — NO new module, NO Net capability** | Parsing is a *pure string → record* transform (no network I/O). It belongs beside the existing pure `urlEncode`/`urlEncodeForm` in `std/net`, registered `IsPure: true` exactly like them. A new module or a `! {Net}` requirement would be wrong: it grants no authority and reaches no network. | human | design | med |
| **`Url` is a plain record, not an opaque newtype** | Unlike regex's `Regex` (an opaque *validated-pattern handle* that must only be constructed via `compile`), a parsed URL is *data* whose whole value is field access (`u.host`, `u.path`). A record gives direct `.field` reads with no accessor boilerplate. Opaque would add friction for zero benefit. | agent | design | med |
| **`parseUrl -> Result[Url, string]`** | Go `url.Parse` *can* error (control chars, invalid `%`-escape, bad port). On error return `Err(msg)` — NO SILENT FALLBACK on structural fields (CP2). One fallible boundary; the record fields are then total. | human | design | med |
| **`port` is a `string`, not `int`** | Mirrors Go `url.Port()` which returns `""` for "no port present" and only the digits when present. An `int` would force a sentinel (`-1`/`0`) for absent — a silent-fallback ambiguity (is `0` "port zero" or "no port"?). Empty string is the honest "absent" (CP2). | human | design | low |
| **`parseQuery` preserves declaration order** and returns a **list of `{name, value}`**, not a map | The inverse of `urlEncodeForm` (which takes `[{name,value}]`). Go's `url.ParseQuery` returns `url.Values` (a **sorted map**, lossy for order + duplicate keys); we parse the raw string ourselves to preserve source order and repeated keys (`?a=1&a=2` → two entries). Same shape as the encode side → clean round-trip. | human | design | med |
| **All functions pure (`! {}`)** | Parsing is deterministic pure computation over a string; no effects. Usable in contracts and pure code. | compiler | design | low |

### Design Freeze

Resolve before sprint-executor starts:

- [x] Wrap Go `net/url` (ratified here; mandate says "URL-parse builtins").
- [x] Extend `std/net`, pure builtins (`IsPure: true`), no Net capability, no new module.
- [x] `Url` is a flat record (fields verified `ailang check`-clean below); `parseUrl -> Result[Url, string]`.
- [x] `port: string` (Go `url.Port()` semantics; empty = absent).
- [x] `parseQuery -> [{name: string, value: string}]`, order-preserving, values percent-decoded.
- [ ] Exact `Url` field set — whether to also expose `userinfo`/`rawQuery`-vs-`query` naming, final at review.
- [ ] Whether `parseQuery` is a standalone export or also surfaced as `Url.params` (deferred; see below).

---

## Deferred Decisions

- **Relative→absolute URL resolution / joining** (`url.URL.ResolveReference`, "join base + relative")
  — *deferred*; a separate `resolveUrl(base, ref)` follow-up. Parsing ≠ resolving.
- **IDNA / punycode host normalization** (`xn--` ↔ unicode) — *deferred*; Go doesn't do it in
  `Parse` either. Host is returned as-written.
- **A `params` accessor on `Url`** (auto-parsed query pairs as a field) — *deferred*; `parseQuery`
  is the explicit path. Adding it later is additive.
- **Building a URL back from a `Url` record** (`formatUrl : Url -> string`, `url.URL.String()`) —
  *deferred*; the encode side (`urlEncode`/`urlEncodeForm`) already covers the common "build a
  request" case. A follow-up if round-tripping structured URLs is demanded.
- **`userinfo` (user:password) field** — *agent may add* if cheap (Go exposes `url.User`);
  otherwise a follow-up. Security note: passwords in URLs are deprecated; expose read-only if at all.
- **Multi-value query map accessor** (`getParam(name) -> Option[string]` convenience over the
  list) — *agent may add*; the list is the primitive.

---

## Solution Design

### Overview

Add two **pure** builtins to the existing `internal/builtins/net.go` and two `export func`
wrappers to the existing `std/net.ail`. No new files strictly required (mirrors how `urlEncode`
was added to the same module). The builtins delegate to Go's `net/url`:

1. `parseUrl(s) -> Result[Url, string]` — `url.Parse(s)`, map the resulting `*url.URL` into the
   `Url` record. On parse error, `Err(err.Error())`. This is the **only** fallible entry point.
2. `parseQuery(s) -> [{name, value}]` — parse the raw `a=1&b=2` string *ourselves* (splitting on
   `&` then `=`, percent-decoding each half with `url.QueryUnescape`), preserving source order and
   duplicate keys. Total: malformed pairs decode best-effort (a bare `foo` with no `=` yields
   `{name:"foo", value:""}`, matching `url.ParseQuery` leniency), never panics.

`parseQuery` is deliberately **not** `url.ParseQuery` verbatim: that returns `url.Values`
(a `map[string][]string`), which loses declaration order and is awkward to marshal as ordered
`{name,value}` pairs. Parsing the raw string keeps it order-preserving and duplicate-safe — the
exact inverse of `urlEncodeForm`.

### Architecture

**Components:**
1. **`internal/builtins/net.go`** (extend): register `_net_url_parse` (`NumArgs: 1`, `IsPure: true`)
   and `_net_url_parse_query` (`NumArgs: 1`, `IsPure: true`) alongside the existing
   `registerNetURLEncode()`/`registerNetURLEncodeForm()` in `init()`. Marshal results using the
   record/list/Result helpers already used in this file (`eval.RecordValue`, `eval.ListValue`,
   `eval.StringValue`) and the same `Result`/`Ok`/`Err` value shape the module uses elsewhere.
2. **`std/net.ail`** (extend): `export type Url = { ... }` plus two `export func` wrappers calling
   the builtins. This is the documented public surface, sitting beside `urlEncode`/`urlEncodeForm`.
3. **Examples**: `examples/url_parse_basics.ail` (parse + field access), one orchestration example
   (route/dispatch on host+path, read a query param) feeding clause-4.

### Public API (verified `ailang check`-clean — see Verification below)

```ailang
module std/net

import std/result (Result, Ok, Err)

-- A parsed URL as a plain record for direct field access.
-- `port` is a string ("" when absent, matching Go url.Port()); `query` and
-- `fragment` are the raw substrings after `?` and `#` (query still encoded —
-- use parseQuery to decode it into pairs). Values in host/path are decoded.
export type Url = {
  scheme: string,     -- "https"        ("" if the URL is scheme-relative)
  host: string,       -- "example.com"  (hostname only, no port, no brackets for IPv6)
  port: string,       -- "8443"         ("" when no explicit port)
  path: string,       -- "/a/b"         (percent-decoded)
  query: string,      -- "q=1&r=2"      (raw, still percent-encoded — feed to parseQuery)
  fragment: string    -- "frag"         (decoded; "" when absent)
}

-- Parse an RFC-3986 URL into its components. Err on malformed input
-- (control chars, invalid %-escape, bad port) — never a silent fallback.
export func parseUrl(s: string) -> Result[Url, string] { ... }

-- Parse a query string ("a=1&b=2") into order-preserving {name,value} pairs.
-- Values are percent-DECODED (inverse of urlEncodeForm). Duplicate keys are
-- kept as separate entries. Total: a bare "foo" yields {name:"foo", value:""}.
export func parseQuery(s: string) -> [{name: string, value: string}] { ... }
```

*(Both `pure`/`! {}` — no effects, no Net capability, exactly like `urlEncode`. Shown as `func`
for brevity; they carry no effect row. Signatures type-checked verbatim — see Verification.)*

### Implementation Plan

**Phase 1: Go builtins + tests** (~0.5 day)
- [ ] Extend `internal/builtins/net.go`: add `registerNetURLParse()` and
      `registerNetURLParseQuery()`, both `IsPure: true`, `Effect: ""`, `NumArgs: 1`; call them in
      `init()`.
- [ ] `_net_url_parse`: `url.Parse(s)`; on error return the module's `Err(err.Error())` value; on
      success build the `Url` `RecordValue` (`u.Scheme`, `u.Hostname()`, `u.Port()`,
      `u.EscapedPath()` or `u.Path`, `u.RawQuery`, `u.Fragment`). Decide `Path` (decoded) vs
      `EscapedPath()` — default `u.Path` (decoded), documented.
- [ ] `_net_url_parse_query`: split raw string on `&`, each on first `=`, `url.QueryUnescape` both
      halves, build ordered `[]{name,value}` `ListValue` of `RecordValue`s (reuse the
      field-extraction shape from `netURLEncodeFormImpl`, inverted).
- [ ] Type specs: `makeNetURLParseType` (`string -> Result[Url, string]`, no effects) and
      `makeNetURLParseQueryType` (`string -> List[{name,value}]`). Reuse `httpHeaderType(T)` for the
      `{name,value}` record and add a `urlRecordType(T)` builder for `Url`.
- [ ] Go unit tests: full URL, scheme-relative, no-port, IPv6 host, userinfo, `?`+`#`, empty query,
      percent-decoded query values, duplicate keys, and an **error case** (invalid `%` / control
      char) asserting `Err`, never panic.

**Phase 2: std/net module + examples** (~0.25 day)
- [ ] Add `export type Url` + `parseUrl` + `parseQuery` wrappers to `std/net.ail` (beside the
      encode functions), with the doc comments above.
- [ ] `examples/url_parse_basics.ail` (parse + all fields), one orchestration example (dispatch on
      `u.host`/`u.path`, read a query param via `parseQuery` + `nth_or`).
- [ ] `make verify-examples` green; register examples on the website per coding-standards.

**Phase 3: Docs + integration** (~0.25 day)
- [ ] Stdlib-level test (`.ail` golden or Go harness) over the public API incl. the
      `parseQuery(urlEncodeForm(pairs))` round-trip.
- [ ] Update `docs/LIMITATIONS.md` (URL parsing now present), the CLI/stdlib reference for
      `std/net`, and the stability tier table (new functions → Experimental at introduction, or
      Stable-adjacent to the existing encode functions — decide at review).
- [ ] CHANGELOG entry; teaching-prompt note if the prompt enumerates `std/net`'s surface.
- [ ] Coordinate one URL-parse-flavored example/benchmark into the clause-4 flagship (#19) rotation.

### Files to Modify/Create

**Modified files:**
- `internal/builtins/net.go` (~+130 LOC) — 2 builtins + 2 type specs + `Url` record type builder +
  registration in `init()`.
- `internal/builtins/net_test.go` (~+120 LOC) — unit tests incl. error case + query decode/order.
- `std/net.ail` (~+45 LOC) — `Url` type + `parseUrl` + `parseQuery` wrappers with doc comments.
- `docs/LIMITATIONS.md`, `docs/docs/reference/stability.md` (tier rows), `CHANGELOG.md`.

**New files:**
- `examples/url_parse_basics.ail`, `examples/url_route_dispatch.ail` (~80 LOC).

*(No new Go file and no new stdlib module — the change lives inside the existing `net.go` /
`net.ail`, exactly as `urlEncode` was added. A separate file is optional if `net.go` grows past
the size target; the builtin surface is small enough to co-locate.)*

---

## Examples

### Example 1: Route/dispatch on a parsed URL (the orchestration use case)

```ailang
import std/net (parseUrl, parseQuery)
import std/result (Result, Ok, Err)
import std/list (nth_or)
import std/io (println)

export func main() -> () ! {IO} {
  match parseUrl("https://api.example.com:8443/v1/users?id=42&fmt=json") {
    Err(e) => println("bad url: ${e}"),
    Ok(u) => {
      let params = parseQuery(u.query);
      let first  = nth_or(params, 0, {name: "", value: ""});
      println("${u.scheme}://${u.host}:${u.port}${u.path}");  -- https://api.example.com:8443/v1/users
      println("first param: ${first.name}=${first.value}")     -- first param: id=42
    }
  }
}
```

*(NOTE: list access uses `nth_or` — AILANG lists have **no `xs[i]` subscript**. String building
uses `"${...}"` interpolation — `++` is **list-only**. Both traps were caught by `ailang check`
while verifying this doc; both snippets above are check-clean — see Verification.)*

### Example 2: Round-trip with the encode side

```ailang
import std/net (parseQuery, urlEncodeForm)
import std/list (nth_or)
import std/io (println)

export func main() -> () ! {IO} {
  let body   = urlEncodeForm([{name: "q", value: "hello world"}, {name: "n", value: "42"}]);
  -- body == "n=42&q=hello+world" (urlEncodeForm sorts keys)
  let pairs  = parseQuery(body);
  let q      = nth_or(pairs, 1, {name: "", value: ""});
  println("decoded: ${q.name}=${q.value}")   -- decoded: q=hello world  (percent-DECODED)
}
```

**Before this feature:** no URL parsing — the author hand-rolls a `std/string.split` chain,
mishandling `://`, `?` vs `#`, optional `:port`, and percent-decoding.
**After:** one call gives a typed, RFC-3986-correct record.

---

## Success Criteria

- [ ] `std/net` gains `parseUrl` + `parseQuery` + the `Url` type — all `ailang check`-clean.
- [ ] Both builtins registered `IsPure: true`, `Effect: ""` — usable in pure code, **no Net
      capability** required (asserted by a pure-context test that runs them without `--caps Net`).
- [ ] `parseUrl` returns `Err(msg)` for malformed input (invalid `%`-escape / control char) —
      never panics (CLAUDE.md CP2). Test asserts `Err`.
- [ ] Field correctness: `parseUrl("https://user@host:8443/a/b?q=1#f")` → scheme/host/port/path/
      query/fragment all correct; IPv6 host (`http://[::1]:80/`) → `host="::1"`, `port="80"`;
      no-port URL → `port=""`.
- [ ] `parseQuery` percent-**decodes** values, preserves source order, keeps duplicate keys
      (`?a=1&a=2` → two entries), handles bare keys (`?flag` → `{name:"flag", value:""}`).
- [ ] Round-trip: `parseQuery(urlEncodeForm(pairs))` recovers the pairs (modulo encode key-sort).
- [ ] 2 examples runnable (basics, route-dispatch); `make verify-examples` green.
- [ ] Go test coverage ≥80% on the new `net.go` code.
- [ ] `make lint` (0 issues) + docs (LIMITATIONS, stability tier, CHANGELOG, reference) updated;
      `make test` green.

---

## Conflict Surface

This is a **purely additive, namespace-only** change: two new pure builtin names and two new
`std/net` exports (+ one new module-scoped type). It touches `internal/builtins/` (builtin
registration) — hence this section is mandatory — but it **adds no grammar, no new syntactic
position, and no new AST node**. The "conflict surface" is therefore namespace/registry collision,
not parser disambiguation.

### Syntactic positions touched

**None.** No lexer, parser, AST, type-syntax, or elaboration changes. URLs are ordinary `string`
literals passed to ordinary function calls; `Url` is an ordinary record type built from existing
type syntax. No new operators, no new type syntax, no new keywords.

### What else lives here

The only shared positions are two flat namespaces plus one module-scoped type name:

| Position | Existing occupants | This change adds | Collision? |
|----------|--------------------|--------------------|------------|
| builtin-name registry (`RegisterEffectBuiltin`) | `_net_httpRequest`, `_net_url_encode`, `_net_url_encode_form`, `_str_*`, `_json_*`, … | `_net_url_parse`, `_net_url_parse_query` | **No** — verified: `grep _net_url_parse internal/` → 0 matches |
| `std/net` export namespace | `httpGet`, `httpPost`, `httpRequest`, `httpRequestBytes`, `urlEncode`, `urlEncodeForm`, `NetError`, `HttpResponse` | `parseUrl`, `parseQuery`, `Url` | **No** — verified: no `parseUrl`/`parseQuery`/`Url` in `std/net.ail` |
| module-scoped type names in `std/net` | `NetError`, `HttpResponse` | `Url` | **No** — module-scoped; distinct name |

### Disambiguation strategy

Not applicable — builtin names are exact-string keys resolved by exact match, module export names
are resolved within the `std/net` namespace, and none of the new names collide with existing ones
(verified above). `Url` is a plain record type distinct from `HttpResponse`/`NetError`.

### Programs that MUST still work

Since nothing is modified (only added), regression risk is limited to (a) the `net.go` `init()`
wiring not disturbing existing registration, and (b) the new `std/net.ail` exports not breaking the
module's existing type-check/embed. Fixtures to keep green:

- **Existing `std/net` consumers** — every `examples/` and demo using `httpRequest`, `httpGet`,
  `urlEncode`, `urlEncodeForm` still type-checks and runs unchanged.
- **`urlEncode` / `urlEncodeForm` round-trip** — the encode functions are untouched; their output
  is now *also* consumable by `parseQuery`, but their own behavior is identical.
- **`std/net` module type-check + embed** — `ailang check std/net.ail` clean after the additions;
  the stdlib embed still loads.
- **`make verify-examples`** over the full corpus — no example regresses.
- **Builtin-registration tests** — every prior `_net_*` builtin still registered after the two
  new registrations are added to `init()`.
- **Full `make test`** — startup/registry tests catch any accidental name shadowing.

### What deliberately changes

Nothing existing changes or breaks. New surface only. The single **intentional design choice**
(not a regression): `parseQuery` returns an **order-preserving list**, deliberately *not* Go's
sorted `url.Values` map — so it round-trips cleanly with `urlEncodeForm` and preserves duplicate
keys. Documented in the module doc.

---

## Testing Strategy

**Unit tests (`internal/builtins/net_test.go`):**
- `parseUrl` full URL (scheme/host/port/path/query/fragment all correct).
- `parseUrl` edge cases: scheme-relative (`//host/path`), no-port (`port=""`), IPv6 host
  (`http://[::1]:80/` → `host="::1"`), userinfo (`https://user@host/`), path-only, empty query.
- `parseUrl` **error case**: invalid `%`-escape / control char → `Err`, never panics (CP2).
- `parseQuery`: percent-decode (`q=hello%20world` → `hello world`), order preservation,
  duplicate keys (`a=1&a=2` → two entries), bare key (`flag` → `{name:"flag", value:""}`),
  empty string (`""` → `[]`).
- **Purity**: both builtins registered `IsPure: true`, `Effect: ""` — run in a pure context with
  no `Net` capability granted; assert success (they must NOT require `--caps Net`).

**Integration tests:**
- `.ail`-level golden over `examples/url_parse_*.ail` (parse → field access → query read path).
- **Round-trip**: `parseQuery(urlEncodeForm(pairs))` recovers the pairs (modulo encode key-sort).
- Cross-module: `std/net` URL parse used alongside `std/list.nth_or` and `std/result`.

**Regression-surface tests** (from Conflict Surface):
- `make verify-examples` (full corpus) — no pre-existing example regresses.
- Builtin-registration test — every prior `_net_*` builtin still present.

**Manual testing:**
- `ailang run examples/url_route_dispatch.ail` prints host/path/param.
- `ailang check std/net.ail` clean.

---

## Non-Goals

**Not in this feature:**
- **Relative→absolute URL resolution / joining** (`ResolveReference`, base+relative) — deferred to
  a `resolveUrl` follow-up. Parsing ≠ resolving.
- **IDNA / punycode host normalization** — deferred; Go `Parse` doesn't do it either. Host as-written.
- **Building a URL string from a `Url` record** (`formatUrl`) — deferred; the encode side already
  covers the "build a request" case.
- **A `params` field auto-parsed onto `Url`** — deferred; `parseQuery` is the explicit path.
- **A query-map convenience accessor** (`getParam(name) -> Option[string]`) — deferred; the
  `{name,value}` list is the primitive.
- **Regex** — the *other* half of clause 4, shipped as
  [m-stdlib-regex](../../implemented/v0_30_0/m-stdlib-regex.md) (#11).

---

## Timeline

**Day 1** (~1 day total):
- Phase 1 (~0.5d): Go builtins (`_net_url_parse`, `_net_url_parse_query`) + type specs +
  `Url` record builder + registration + unit tests (incl. error case, query decode/order, purity).
- Phase 2 (~0.25d): `std/net.ail` additions (`Url` + 2 wrappers) + 2 examples + verify-examples.
- Phase 3 (~0.25d): round-trip test, docs (LIMITATIONS, stability, CHANGELOG, reference), buffer.

**Total: ~1 day.** (The wrap-Go-`net/url` decision is what keeps this at 1 day — the parser is Go
stdlib, and the pattern already exists in `net.go` from `urlEncode`.)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Rune-vs-byte index hazard (the F1 class)**: AILANG `_str_len`/`_str_slice` are rune-indexed, while some Go APIs return byte offsets | **Low / non-issue** | `net/url` returns **decoded string fields** (`Scheme`, `Hostname()`, `Port()`, `Path`, `RawQuery`, `Fragment`) — **no offsets/indices are exposed**. Every `Url` field is a plain string value, so there is no offset to reconcile. **Confirmed: no exposed field is an index.** (Contrast regex, which *did* return spans and had to convert.) Documented here so the implementer doesn't reintroduce an index field. |
| Go `url.Parse` is lenient — "invalid" is narrow (it accepts a lot) | Medium | Document what `Err` means (control chars, invalid `%`-escape, bad port) vs what parses-but-is-odd. Don't over-promise validation; `parseUrl` is a *parser*, not a *validator*. A caller wanting strict validation composes with `std/regex`. |
| `parseQuery` semantics differ from Go `url.ParseQuery` (order-preserving list vs sorted map) | Medium | Intentional (round-trips with `urlEncodeForm`, keeps duplicates/order). Documented in the module doc + Conflict Surface "What deliberately changes". Test asserts order + duplicates. |
| `Path` decoded vs `EscapedPath()` raw — wrong choice surprises users with `%2F` in paths | Low | Default `u.Path` (decoded) for ergonomics; document it; a raw-path field is a deferred add if demanded. Test a path with an encoded segment. |
| Marshalling the `Url` record into `eval.Value` | Low | Copy the exact `RecordValue`/`StringValue`/`ListValue` patterns already in `net.go` (`netURLEncodeFormImpl`) — proven code, don't invent. |
| `parseUrl` panics on some exotic input instead of erroring | Low (CP2) | Go `url.Parse` returns `error`, doesn't panic; wrap defensively and test the error path. Fuzz a few malformed inputs. |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure deterministic transform over the input string (RFC-3986); identical input → identical record. |
| A2: Replayability | 0 | Pure computation; no state/trace to replay. |
| A3: Effect Legibility | +1 | All functions pure (`! {}`) — no hidden effects, **no Net capability** despite living in `std/net`; reinforces byte-shaping vs IO separation (same stance as `urlEncode`). |
| A4: Explicit Authority | 0 | No capabilities / ambient access; parsing reaches no network. |
| A5: Bounded Verification | +1 | Total (`parseQuery`) / single-`Result` (`parseUrl`) functions over a closed RFC-3986 spec; O(n) on input length, no unbounded behavior. |
| A6: Safe Concurrency | 0 | Stateless; `net/url` is goroutine-safe. |
| A7: Machines First | +1 | Gives AI authors a standard, well-understood primitive instead of a hand-rolled `split` chain (lower token cost, fewer RFC bugs). |
| A8: Minimal Syntax | 0 | No new syntax — URLs are string literals, parsing is function calls, `Url` is a plain record. |
| A9: Cost Visibility | +1 | O(n) linear parse; predictable cost, no surprise. |
| A10: Composability | +1 | Composes with `urlEncode`/`urlEncodeForm` (round-trip inverse), `std/result`, `std/list.nth_or`, `httpRequest` (parse a redirect, then re-fetch). |
| A11: Structured Failure | +1 | `parseUrl` failure is a typed `Result[Url, string]`, not a panic — structured, catchable (CP2). |
| A12: System Boundary | +1 | Makes the structure of a URL (host to route on, path to dispatch on, params to branch on) visible at the call site instead of buried in string surgery. |

**Net Score: +8** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — pure RFC-3986 transform.
- [x] A3 (Effects): No hidden side effects — all `pure`, no Net capability.
- [x] A4 (Authority): No ambient access granted.
- [x] A7 (Machines First): Optimizes for machine analysis (standard primitive, bounded cost).

---

## Verification (HARD GATE — language claims proven with `ailang check`)

Run in the worktree at `dev` (v0.29.2-39-gc88e1cf93; `./bin/ailang --version` → `AILANG dev`):

**Claim: "AILANG has no URL-parsing surface today."** ✅ VERIFIED absent:
```
$ grep -rn "_net_url_parse\|parseUrl\|parseQuery" internal/ std/
  # → only Go-internal apiserver helpers (parseQueryArgs); NOTHING in std/net.ail
  #   or the AILANG net builtin surface. No _net_url_parse builtin.
```
(The existing `std/net` URL surface is encode-only: `urlEncode`, `urlEncodeForm` — read directly
from `std/net.ail` lines 204/228 and `internal/builtins/net.go` `registerNetURLEncode*`.)

**Claim: the proposed public signatures are valid AILANG.** ✅ VERIFIED — the exact `Url` record
type and both function signatures `ailang check`-clean (`✓ No errors found!`, only the benign
MOD010 temp-path + stdlib-version warnings):
```
$ ./bin/ailang check /tmp/urlparse_claim.ail
→ Type checking ...
→ Effect checking...
✓ No errors found!
```
File checked (`/tmp/urlparse_claim.ail`):
```ailang
module test/claim
import std/result (Result, Ok, Err)
export type Url = {
  scheme: string, host: string, port: string,
  path: string, query: string, fragment: string
}
export func parseUrl(s: string) -> Result[Url, string] { Err("stub") }
export func parseQuery(s: string) -> [{name: string, value: string}] { [] }
export func main() -> () ! {} = ()
```

**Claim: a realistic usage example (record `.field` access + `nth_or` list access + `${}`
interpolation) is valid AILANG.** ✅ VERIFIED — check-clean **after** two corrections the
type-checker forced (documented so the implementer avoids them):
- `params[0]` subscript is **not** valid → use `nth_or(params, 0, default)` (AILANG lists have no
  subscript operator — the same trap that bit the regex doc's `m.groups[i]`).
- `a ++ b` on strings is **rejected** (`` `++` is for lists only ``) → use `"${a}${b}"`
  interpolation.
```
$ ./bin/ailang check /tmp/urlparse_usage.ail
→ Type checking ...
→ Effect checking...
✓ No errors found!
```
(The corrected `/tmp/urlparse_usage.ail` is Example 1 above, verbatim — it type-checks with
`main() -> () ! {IO}`, `match` on the `Result`, `u.field` record access, `nth_or` on the
`parseQuery` list, and `${...}` string building. Both traps are called out inline in the Examples.)

**Claim: parsing belongs in `std/net` as a pure builtin (no Net capability).** ✅ Confirmed by the
existing precedent — `urlEncode`/`urlEncodeForm` are registered in `internal/builtins/net.go` with
`IsPure: true, Effect: ""` and exposed as plain (non-`! {Net}`) `export func`s in `std/net.ail`.
The new parse builtins mirror this exactly (read from `net.go` lines 176–219, 241–291).

**Claim: Go `net/url` returns decoded string fields, not byte offsets** (the F1 hazard is a
non-issue). ✅ Confirmed from the `net/url` API: `URL.Scheme`, `URL.Hostname()`, `URL.Port()`,
`URL.Path`, `URL.RawQuery`, `URL.Fragment` are all `string`s; **no offset/index is exposed** →
nothing to reconcile against AILANG's rune-indexed `_str_slice`. (Citation, not a language claim.)

---

## References

- **Mission**: [v1-mission.md](../../v1-mission.md) — queue item #12, bar clause 4 (R7).
- **Strategy**: R7 (regex + URL mandated as the clause-4 builtin pair).
- **Sibling (regex half, template)**:
  [m-stdlib-regex](../../implemented/v0_30_0/m-stdlib-regex.md) (#11) — same wrap-Go-stdlib shape,
  two-stage API, additive Conflict Surface, Risks table.
- **Encode complement (inverse)**:
  [m-stdlib-url-encode](../../implemented/v0_19_2/m-stdlib-url-encode.md) (v0.20.0) —
  `urlEncode`/`urlEncodeForm`; house style + the pure-`std/net`-builtin precedent.
- **Builtin conventions**: `internal/builtins/net.go` (`registerNetURLEncode*`,
  `netURLEncodeFormImpl` record/list marshalling, `IsPure: true`), `std/net.ail` (module exports).
- **List access**: `std/list.ail` `nth_or` (no list subscript operator).
- **Prior art**: Go `net/url.Parse` (RFC-3986), Python `urllib.parse.urlparse`, JS `URL`.
- **Axiom reference**: [Design Axioms](/docs/references/axioms).

---

## Future Work

- `resolveUrl(base, ref)` — relative→absolute URL resolution (`ResolveReference`).
- `formatUrl(u: Url) -> string` — rebuild a URL string from a `Url` record.
- `getParam(name)` convenience accessor over the `parseQuery` list.
- IDNA/punycode host normalization if internationalized-domain workloads demand it.
- A URL-parse-flavored orchestration benchmark promoted into the default rotation (feeds flagship #19).

**DESIGN_DOC_PATH**: `design_docs/planned/v0_30_0/m-stdlib-url-parse.md`
