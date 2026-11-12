### Fixed — `TestStreamSSEProxyBoundaryStreaming` flaked and hung under a parallel `make test` (2026-11-12)

The test added by #2192 failed with `proxy observed ["CONNECT example.com:443" "GET /"]` and
once hung the `internal/effects` package until the 10-minute timeout. The cause was in the test,
not in production code. Its forward proxy and origin listen on ordinary 127.0.0.1 ephemeral ports.
Under a parallel `make test`, a foreign client (another test or another process) can reach such a
port after the kernel has handed it to this test, for example a port that client was using, or one
"reserved" by listen-then-close.
The proxy counted any request it received as an observation. It also forwarded a stray
origin-form `GET /` to the single-use origin handler, whose release verdict send then blocked
forever, so the deferred `httptest.Server.Close` never returned.

- The test proxy is now credential-gated. The client under test gets a per-test
  `user:token@` proxy URL and sends `Proxy-Authorization`. Only those requests count as
  observations. Anything else gets a 407, is never forwarded or tunnelled, and is listed in the
  failure message. The assertion is unchanged: exactly one proxied request, and direct routing
  still gives zero.
- The origin streams only on `/sse`, and only once. Any other request gets a 404.
- Each unbuffered subtest now sends a stray `GET /` on purpose, so the condition that used to be
  rare happens on every run.
- Each subtest has a 45s watchdog. A hang now panics within seconds, naming the subtest and
  dumping every goroutine, instead of running into the 10-minute package timeout.
