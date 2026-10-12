# Sprint retrospective: M-SERVEAPI-RESPONSE-HEADERS

Implemented 2026-10-09, target v0.54.0. Refs #1597. Refs #1609.

M1 is self-contained first commit `42b444ed` (302 added lines; 360 estimated),
M2 `d250d0b5` (641 added; 540 estimated), M3 `fc7cb61a` (273 added; 220 estimated).
Total 1,216 added lines against 1,120 estimated. One cloud session; M3 checker work
ran independently while the root handled M1/M2. Independent evaluation followed.

Static routing includes ServeMux clean-path redirects: the wrapper checks the
selected fallback so API redirects stay isolated. server.go is 794 lines (+6).

## Verification

- Fresh make build and focused apiserver/mcpcheck suites pass.
- CLI tests pass with `ServeAPI|Static|^TestHelp`, which excludes unrelated Helpers timing tests.
- make lint passes after warm-cache retry (first run: zero findings but timed out).
- fmt-check, check-file-sizes, check-cli-docs, check-changelog, check-boundaries pass.
- verify-examples passes: 234 passed, 9 skipped; manifest module drift zero.
- Both replacement guide snippets extracted and checked with fresh binary; response example checks/runs (`ok`), auth fixture checks, request-header test suite passes.
- Removing record remap makes raw-record/nowrap-record wire cases fail; refusing raw Json makes raw-json/Result cases fail. Restoring source makes the matrix pass.
- MCP checker tests assert all exact dummy params/S256 digest, redirects, final page cases, discovery composition and failures.

## Baseline and environment limits

Full make test remains deferred to PR CI per approved plan. test-core hits eight
existing brain SQLite tests because CGO_ENABLED=0 and gcc is absent. No effects
or compiler code changed. The broad planned CLI filter also matches
TestStdListHelpersIterativeAtScale; its 50,000-item interpreter case took 34.89s
against 30s during concurrent lint/build. Intended CLI tests pass using the
anchored Help filter above.

The unchanged examples/runnable/mcp_tools.ail compares Option[string] with string
at line 25. Fresh and installed pre-sprint v0.52.5 binaries reproduce the same
type error against current stdlib; the fixture was deliberately left unchanged.
The request-header auth fixture and suite pass. Manifest validation also warns
about the preexisting missing lambda_expressions.ail entry; no modules drift.

## Showcase disposition

AILANG prompt version loaded: v0.16.6. Handler signatures are pure, with no new
effects. Contracts skipped: the property is HTTP metadata at the Go boundary.
Inline tests skipped: no-input response handlers are exercised by production
wire tests, including Result.Ok.

## Fresh binary wire evidence

Commands used curl -sS -D - -o /dev/null (plus --path-as-is for clean redirects)
against the freshly built binary.

```text
defaults /
HTTP/1.1 200 OK
Accept-Ranges: bytes
Content-Length: 20
Content-Security-Policy: frame-ancestors 'none'
Content-Type: text/html; charset=utf-8
Last-Modified: Fri, 09 Oct 2026 08:07:42 GMT
Referrer-Policy: no-referrer
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT

defaults /./index.html
HTTP/1.1 307 Temporary Redirect
Content-Security-Policy: frame-ancestors 'none'
Content-Type: text/html; charset=utf-8
Location: /index.html
Referrer-Policy: no-referrer
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT
Content-Length: 47

defaults /headers/raw-record
HTTP/1.1 200 OK
Content-Security-Policy: frame-ancestors 'none'
Content-Type: text/plain; charset=utf-8
X-Elapsed-Ms: 38
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT
Content-Length: 2

defaults /headers/raw-json
HTTP/1.1 200 OK
Content-Security-Policy: frame-ancestors 'none'
Content-Type: text/plain; charset=utf-8
X-Elapsed-Ms: 0
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT
Content-Length: 2

defaults /headers/nowrap-record
HTTP/1.1 200 OK
Content-Security-Policy: frame-ancestors 'none'
Content-Type: application/json
X-Elapsed-Ms: 0
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT
Content-Length: 19

defaults /headers/nowrap-json
HTTP/1.1 200 OK
Content-Security-Policy: frame-ancestors 'none'
Content-Type: application/json
X-Elapsed-Ms: 0
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT
Content-Length: 19

defaults /headers/result
HTTP/1.1 200 OK
Content-Security-Policy: frame-ancestors 'none'
Content-Type: text/plain; charset=utf-8
X-Elapsed-Ms: 0
X-Frame-Options: DENY
Date: Fri, 09 Oct 2026 08:07:43 GMT
Content-Length: 2

override /
HTTP/1.1 200 OK
Accept-Ranges: bytes
Content-Length: 20
Content-Security-Policy: frame-ancestors 'none'
Content-Type: text/html; charset=utf-8
Last-Modified: Fri, 09 Oct 2026 08:07:42 GMT
Referrer-Policy: no-referrer
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
Date: Fri, 09 Oct 2026 08:07:44 GMT

optout /
HTTP/1.1 200 OK
Accept-Ranges: bytes
Content-Length: 20
Content-Type: text/html; charset=utf-8
Last-Modified: Fri, 09 Oct 2026 08:07:42 GMT
Date: Fri, 09 Oct 2026 08:07:44 GMT

```

Fresh `ailang mcp check <local-mcp-url> --target <target>` evidence:

```text
protected= True target= both exit= 0
MCP directory check: http://127.0.0.1:45335/mcp/ (target: both)

  PASS  annotations                  all 1 tools have a title and hints [A7]
  PASS  credentials                  no credential-shaped parameters [A2,O1]
  PASS  zero-arg                     no tool requires a placeholder argument
  PASS  lazy-auth                    1 gated tool(s) answer 401 with resource metadata naming http://127.0.0.1:45335 [A3]
  PASS  authorization-server         http://127.0.0.1:45335 advertises S256 and a client registration method [A6,O6]
  PASS  authorization-framing        final authorization HTML has X-Frame-Options or CSP frame-ancestors [RFC 9700 §4.16]
  PASS  file-params                  no tool declares openai/fileParams [O9]
  PASS  security-schemes             1 gated tool(s) declare an oauth2 security scheme [O5,O10]

No failures. Re-fetch the vendor requirements before submitting; they change.

protected= False target= both exit= 1
MCP directory check: http://127.0.0.1:45335/mcp/ (target: both)

  PASS  annotations                  all 1 tools have a title and hints [A7]
  PASS  credentials                  no credential-shaped parameters [A2,O1]
  PASS  zero-arg                     no tool requires a placeholder argument
  PASS  lazy-auth                    1 gated tool(s) answer 401 with resource metadata naming http://127.0.0.1:45335 [A3]
  PASS  authorization-server         http://127.0.0.1:45335 advertises S256 and a client registration method [A6,O6]
  FAIL  authorization-framing        final authorization HTML lacks valid X-Frame-Options and CSP frame-ancestors; consent pages can be framed [RFC 9700 §4.16]
  PASS  file-params                  no tool declares openai/fileParams [O9]
  PASS  security-schemes             1 gated tool(s) declare an oauth2 security scheme [O5,O10]

Not ready to submit. Sources: design_docs/planned/v0_51_0/m-serveapi-directory-ready-sources.md

protected= False target= openai exit= 0
MCP directory check: http://127.0.0.1:45335/mcp/ (target: openai)

  PASS  annotations                  all 1 tools have a title and hints [A7]
  PASS  credentials                  no credential-shaped parameters [A2,O1]
  PASS  zero-arg                     no tool requires a placeholder argument
  PASS  lazy-auth                    1 gated tool(s) answer 401 with resource metadata naming http://127.0.0.1:45335 [A3]
  PASS  authorization-server         http://127.0.0.1:45335 advertises S256 and a client registration method [A6,O6]
  WARN  authorization-framing        final authorization HTML lacks valid X-Frame-Options and CSP frame-ancestors; consent pages can be framed [RFC 9700 §4.16]
  PASS  file-params                  no tool declares openai/fileParams [O9]
  PASS  security-schemes             1 gated tool(s) declare an oauth2 security scheme [O5,O10]

No failures. Re-fetch the vendor requirements before submitting; they change.

```

Independent sprint-evaluator verdict: PASS 95/100. Report: `.ailang/state/evaluations/eval_M-SERVEAPI-RESPONSE-HEADERS_round_1.json`. This is local implementation assessment; full CI and merge approval remain with the coordinator PR.
