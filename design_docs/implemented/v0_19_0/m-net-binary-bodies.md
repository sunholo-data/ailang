---
name: M-NET-BINARY-BODIES
description: Binary request/response bodies for std/net (httpRequestBytes + bodyBytes) — unblocks image/PDF/binary uploads
type: implemented
---

# M-NET-BINARY-BODIES: binary request and response bodies for `std/net`

**Status**: IMPLEMENTED (v0.19.0)
**Target**: v0.19.0 (small, additive; co-locate with M-COORDINATOR-INBOX-WILDCARDS)
**Priority**: P1 — hard-blocks any AILANG integration that needs binary upload/download
**Estimated**: ~80 LOC core + ~120 LOC tests + stdlib + docs (~3–5h)
**Dependencies**: none (additive change)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-12
**Source**: agent inbox message [12a77549](msg_20260512_101641_12a77549) from `demos/linkedin`
— LinkedIn's image-attach API requires `PUT <uploadUrl>` of raw PNG/JPG bytes
(step 2 of a 3-step dance). Currently impossible.

---

## Problem statement

`std/net.httpRequest` only accepts a `string` body
([`std/net.ail:129`](../../../std/net.ail#L129),
[`internal/effects/net.go:437`](../../../internal/effects/net.go#L437)).
There is no way to put raw binary bytes on the wire. This blocks:

- **LinkedIn image posts** — step 2 of the upload dance is `PUT <uploadUrl>` with
  raw PNG/JPG bytes. Server rejects base64-in-string; `toString(bytes)` fails on
  non-UTF-8 data.
- **PDF/DocAI uploads** — DocAI flows want to send PDF bytes to a server.
- **S3-compatible uploads** — generic object storage `PUT`.
- **Image-generation pipelines** — uploading generated outputs.
- **Binary protobufs / gRPC-style payloads.**

Symmetrically, `HttpResponse.body: string`
([`std/net.ail:18`](../../../std/net.ail#L18)) blocks **binary downloads** —
you can read binary files from disk via `_fs_readFileBytes`, but you cannot
fetch them over HTTP.

### Note on `std/fs.readFileBytes`

`readFileBytes` is declared as `Result[string, string] ! {FS}` and returns
**base64-encoded** content
([`internal/effects/fs.go:435-477`](../../../internal/effects/fs.go#L435-L477)),
which surprised the inbox reporter. **We are not changing this** in this
milestone:

- It has shipped as base64-string since v0.7
  ([changelogs/v0.7-observatory.md:65-66](../../../changelogs/v0.7-observatory.md#L65-L66))
  and is documented in 6+ active prompt files plus the agent prompts.
- A silent return-type swap would corrupt callers doing
  `match readFileBytes(p) { Ok(s) => writeFile("out", s) }` — they'd write
  raw bytes expecting base64.
- Once `httpRequestBytes` exists, the bridge is one line via
  `std/bytes.fromBase64`:
  ```ailang
  match readFileBytes("photo.png") {
    Ok(b64) => match fromBase64(b64) {
      Some(b) => httpRequestBytes("PUT", url, [], b)
      None    => Err(...)
    }
    Err(e)  => Err(...)
  }
  ```

If we later decide we want native-bytes file IO, the right move is a new
`readFileBinary(path) -> Result[bytes, string]` in a separate (additive)
milestone — not a breaking change to `readFileBytes`.

## Goals

1. **Send binary bodies**: new `httpRequestBytes(method, url, headers, body: bytes)`
   producing the same `Result[HttpResponse, NetError]` shape.
2. **Receive binary bodies**: add a `bodyBytes: bytes` field to `HttpResponse`
   alongside the existing `body: string`. (One response, both views — see
   "Why both" below.)
3. **No migration cost** for existing string-body callers. `readFileBytes`
   semantics unchanged — bridge via `std/bytes.fromBase64` (see "Note on
   `std/fs.readFileBytes`").

## Non-goals

- Multipart/form-data builder. Callers can construct multipart bytes by hand
  using `std/bytes`; a builder is a separate (larger) design.
- Streaming uploads/downloads. Bodies are still bounded by `MaxBytes`
  (5MB default). Larger transfers wait for a streaming design.
- Polymorphic `body` argument on the existing `httpRequest`. We considered a
  sum type `HttpBody = TextBody | BinaryBody | NoBody` (Option B in the
  inbox message). Rejected — it forces every existing caller to wrap in
  `TextBody(...)` and bloats the surface. Additive-new-function is cheaper
  and keeps the common case terse.

## Design

### Stdlib surface (`std/net.ail`)

```ailang
-- New: send raw bytes as the request body.
-- Defaults Content-Type to "application/octet-stream" if caller doesn't set one.
-- Sets Content-Length from the byte length (no chunked encoding).
export func httpRequestBytes(
  method: string,
  url: string,
  headers: list[{name: string, value: string}],
  body: bytes
) -> Result[HttpResponse, NetError] ! {Net} =
  _net_httpRequestBytes(method, url, headers, body)

-- HttpResponse gains bodyBytes alongside body.
-- body is the UTF-8 view (existing behaviour, may be lossy for binary).
-- bodyBytes is the raw payload — always populated.
export type HttpResponse = {
  status: int,
  headers: list[{name: string, value: string}],
  body: string,
  bodyBytes: bytes,
  ok: bool
}
```

**Why both `body` and `bodyBytes`** (instead of replacing one with the other):
existing code does `resp.body` everywhere. Replacing with bytes breaks every
caller. Adding a parallel `bodyBytes` field is O(1) extra memory (Go's
`[]byte` and `string` can share storage) and keeps the textual API ergonomic
for the 95% case (JSON, HTML, text). A future major can deprecate `body` if
desired.

### Builtin (`internal/builtins/net.go`)

```go
// _net_httpRequestBytes — same signature shape as _net_httpRequest but body is Bytes.
RegisterSpec(&Spec{
    Name:    "_net_httpRequestBytes",
    Module:  "std/net",
    NumArgs: 4,
    IsPure:  false,
    Effect:  "Net",
    Type:    makeHTTPRequestBytesType(),  // body parameter is Bytes, not String
    Impl:    func(ctx, args) { return effects.Call(ctx, "Net", "httpRequestBytes", args) },
})
```

### Effect handler (`internal/effects/net.go`)

Add `NetHTTPRequestBytes`. Reuses everything from `NetHTTPRequest` except
the body argument unpacking and the `Content-Type` default:

```go
func NetHTTPRequestBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // Steps 0–8 identical to NetHTTPRequest (cap check, method/URL/headers, security).

    bodyVal, ok := args[3].(*eval.BytesValue)
    if !ok {
        return nil, fmt.Errorf("E_NET_TYPE_ERROR: httpRequestBytes: body must be Bytes, got %T", args[3])
    }

    var reqBody io.Reader
    if len(bodyVal.Value) > 0 {
        reqBody = bytes.NewReader(bodyVal.Value)
    }
    req, err := http.NewRequest(method, urlStr, reqBody)
    // ...
    // Default Content-Type if user didn't set one. String httpRequest currently leaves
    // this to httpPost (which sets application/json). For bytes the safe default is octet-stream.
    if req.Header.Get("Content-Type") == "" {
        req.Header.Set("Content-Type", "application/octet-stream")
    }
    // Set Content-Length explicitly (no chunked encoding for binary uploads).
    req.ContentLength = int64(len(bodyVal.Value))

    // Steps 10–13 identical, except the response build now populates BOTH fields.
}
```

Refactor opportunity: extract steps 0–8 (validation, dial, client setup) into
`buildSecureRequest(ctx, method, url, headers) (*http.Client, *http.Request, error)`
so `NetHTTPRequest` and `NetHTTPRequestBytes` don't drift. Worth doing in
this milestone — diff stays small and prevents the two paths' security
checks falling out of sync.

### Response build (shared change)

Both `NetHTTPRequest` and `NetHTTPRequestBytes` now build:

```go
respBytes, err := io.ReadAll(io.LimitReader(resp.Body, ctx.Net.MaxBytes))
// ...
httpResp := &eval.RecordValue{
    Fields: map[string]eval.Value{
        "status":    &eval.IntValue{Value: resp.StatusCode},
        "headers":   makeHeadersList(resp.Header),
        "body":      &eval.StringValue{Value: string(respBytes)},  // existing
        "bodyBytes": &eval.BytesValue{Value: respBytes},           // new
        "ok":        &eval.BoolValue{Value: resp.StatusCode >= 200 && resp.StatusCode < 300},
    },
}
```

Memory note: `string(respBytes)` copies; `BytesValue{Value: respBytes}`
borrows. Net cost is ~1× body size. Acceptable inside the existing 5MB cap.

Add a one-paragraph CHANGELOG entry under "Breaking changes" calling this
out, and a migration snippet.

## Test plan

New tests in `internal/effects/net_test.go`:

1. **PUT raw PNG bytes round-trip** — start an `httptest.Server` that echoes
   `X-Body-SHA256` of the request body; assert client SHA matches.
2. **Default Content-Type** — assert `application/octet-stream` is set when
   caller omits it; assert caller-supplied `Content-Type` wins.
3. **Content-Length set explicitly** — no `Transfer-Encoding: chunked`.
4. **Empty body** — `httpRequestBytes("PUT", url, [], fromString(""))` should
   send `Content-Length: 0`, no body.
5. **Binary response** — server returns a 1KB random byte payload; assert
   `resp.bodyBytes` matches byte-for-byte and `resp.body` is the (possibly
   lossy) string view.
6. **MaxBytes still enforced** — server streams >5MB, expect `BodyTooLarge`.
7. **Security parity** — repeat the existing `httpRequest` security tests
   (DNS rebinding, IP block, redirect, header validation) against
   `httpRequestBytes` to confirm the shared validation path.

For `readFileBytes` change: update
[`internal/effects/fs_test.go`](../../../internal/effects/fs_test.go) to
expect `BytesValue` instead of base64 `StringValue`. Search the repo for any
stdlib/example caller relying on the old behaviour:

```bash
grep -rn "readFileBytes\|_fs_readFileBytes" examples/ stdlib/ std/ benchmarks/
```

End-to-end test in `examples/runnable/`: a small `.ail` file that reads a
PNG from disk, PUTs it to a local test server, and asserts `resp.ok`. Wire
it into `make verify-examples`.

## Acceptance criteria

- [ ] `httpRequestBytes` callable from AILANG, sends raw bytes, defaults
      `Content-Type: application/octet-stream`, sets `Content-Length`.
- [ ] `HttpResponse.bodyBytes` populated for both `httpRequest` and
      `httpRequestBytes` callers; existing `body` field unchanged.
- [ ] Shared `buildSecureRequest` helper so security checks live in one
      place; both call sites use it.
- [ ] Test suite green: `make test`, `make verify-examples`,
      `make test-imports`.
- [ ] Example file `examples/runnable/http_put_bytes.ail` demonstrating the
      LinkedIn-style upload (against `httptest.Server`, runnable in CI).
- [ ] Reply to inbox message
      [12a77549](msg_20260512_101641_12a77549) when shipped, with the v0.19.0
      version tag.

## Open questions

1. **Transparent gzip on responses**: `NetHTTPRequest` lets Go transparently
   decompress gzipped responses. For binary downloads we want this off (we
   want the raw payload). Does Go's transport actually re-inflate a binary
   PNG response? Test #5 should pin this down. If yes, we need a
   per-request opt-out for `httpRequestBytes`.
2. **Should `httpRequestBytes` also accept a `bytes` body for Content-Type
   `application/json`?** Probably yes — caller can set the header. The
   default just kicks in when they don't.
3. **Naming**: `httpRequestBytes` vs `httpRequestBinary` vs `httpRequestRaw`.
   `Bytes` matches the existing `readFileBytes`/`writeFileBytes` convention;
   defaulting to that.
