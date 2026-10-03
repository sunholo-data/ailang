### Added — port-qualified `net_allow` entries; restricted mode admits one loopback port (#1558) (2026-10-03)

A `net_allow` entry (and `--net-allow-domains` / `--stream-allow-domains`) may now be `host:PORT`
— `api.example.com:8443`, `*.svc.example:9000`, `127.0.0.1:7655`, `localhost:7655`, `[::1]:7655`
— admitting that host on that port only. A port-qualified **loopback** entry is itself the loopback
grant for exactly that host and port, so a host serving a local mock (the `api_call_json`
benchmark) no longer has to open all of loopback to a confined program:
`net_allow = ["127.0.0.1:7655"]` + `net_allow_http = true` reaches the mock and nothing else on
loopback. Restricted mode now admits a port-qualified loopback **literal**. The rule lives in the one
destination authorizer (`internal/effects/net_allow.go`), so it applies to Net and Stream (SSE,
NDJSON, WebSocket), at every round trip and at the pinned dial, which connects only to the
authorized port. Redirect hops and `Net[scope=public]` frames never receive the grant.

### Fixed — malformed and unconfinable `net_allow` entries are refused at policy load (#1558) (2026-10-03)

A malformed entry (`127.0.0.1:0`, `http://host`, an unbracketed IPv6 with a port) used to load and
silently admit nothing; every mode now refuses it by name. Restricted mode refuses a bare loopback
entry (`127.0.0.1` — it would open every port; the error names `127.0.0.1:PORT`), a loopback name
(`localhost:PORT` — list the literal), and private, link-local (metadata included), unspecified and
multicast literals, all of which it previously accepted and then refused at runtime with
`E_NET_IP_BLOCKED`. `trusted_host` keeps a bare loopback entry as an all-port grant.
