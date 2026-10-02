# Sprint plan: M-NET-SCOPE-PUBLIC (#1522)

**Design**: [m-net-scope-public.md](m-net-scope-public.md) · **Sprint JSON**: `.ailang/state/sprints/sprint_M-NET-SCOPE-PUBLIC.json`
**Duration**: 1 day, attended, executed in-session · **Risk**: medium (security; touches effects + types + eval)
**Registry reuse**: `none` for every milestone — a core effect-system change, no package capability.

Every acceptance criterion names a test function or a command. A criterion without one is not done.

## M1 — redirect hops are public-only (~60 LOC + ~120 test)
`internal/effects/net_authorize.go` (`forRedirectHop`, `checkRedirect`), `net_proxy.go` (`RoundTrip`).
- `go test ./internal/effects -run TestRedirectHop_` passes:
  - `TestRedirectHop_HostnameToLoopbackRefused` (public httptest → `http://evil.test/` with `lookupIP` → 127.0.0.1, AllowLocalhost+AllowMetadata true)
  - `TestRedirectHop_HostnameToMetadataRefused` (→ 169.254.169.254)
  - `TestRedirectHop_LiteralLinkLocalRefused` (`Location: http://169.254.169.254/…`)
  - `TestRedirectHop_LocalhostNameRefused` (`Location: http://localhost/…`)
  - `TestRedirectHop_PublicTargetStillFollowed` (control)
- `TestDirectMetadataCall_BareNetStillAllowed` passes (hop 0, `lookupIP` → 169.254.169.254, AllowMetadata).
- Existing `go test ./internal/effects/...` stays green.

## M2 — `Net[scope=public]` in the type system (~50 LOC + ~60 test)
`internal/types/effects.go` (schema), `internal/types/effect_subsumption.go` (narrowing params).
- `go test ./internal/types -run 'TestNetScope'` passes:
  `TestNetScopePublic_SchemaLegal`, `TestNetScope_UnknownValueRejected`,
  `TestNetScopePublic_SubsumesBareNet` (both directions), `TestNetScope_DoesNotLeakToOtherEffects`.
- Command: `ailang check` (repo build) on `func f(u: string) -> string ! {Net[scope=public]} = httpGet(u)` → `No errors found`.
- Command: `ailang check` on `examples/net_scope_public.ail` → `No errors found`.

## M3 — runtime enforcement (~80 LOC + ~100 test)
`internal/effects/net_scope.go` (counter, Push/Pop, Clone reset), `netPolicy`; `internal/eval` (`EffectNetScope`, push/pop, `replaceable`); `internal/gen/lower/program.go` (VM fail-closed).
- `go test ./internal/effects -run TestNetScopePublic_` passes:
  `TestNetScopePublic_HostnameToLoopbackRefused`, `TestNetScopePublic_HostnameToMetadataRefused`,
  `TestNetScopePublic_PopRestoresBareBehaviour`, `TestNetScopePublic_CloneResets`.
- `go test ./internal/eval -run TestNetScope` passes: `TestNetScope_ExtractedFromType`, `TestNetScope_FrameNotTailReplaced`.
- `go test ./cmd/ailang -run TestNetScopePublic_EndToEnd` passes (an `.ail` program declaring `Net[scope=public]` run with AllowLocalhost+AllowMetadata against an httptest server on 127.0.0.1 is refused; the bare-Net twin succeeds).
- `go test ./internal/gen/lower -run TestNetScopePublic_LoweredEvalOnly` passes.

## M4 — docs, changelog, gates, mutation (~docs)
- `changelogs/unreleased/2026-10-02-net-scope-public.md`; `docs/docs/guides/parameterised-effects.md` + `docs/docs/reference/serve-api.md` (or the serve-api guide) mention; devtools prompt row.
- Commands: `make check-file-sizes check-boundaries check-architecture-closure` exit 0; `gofmt -l` empty and `golangci-lint run` clean on touched dirs.
- Mutation: each guard reverted in a scratch copy → a named test fails, mutant compiles (`go vet` of the mutant package exit 0).
- CI: `gh run list -R sunholo-data/ailang --branch dev --workflow ci.yml` green for the pushed head.
