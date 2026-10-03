# Sprint plan: M-NET-SCOPE-PUBLIC (#1522)

**Design**: [m-net-scope-public.md](m-net-scope-public.md) · **Sprint JSON**: `.ailang/state/sprints/sprint_M-NET-SCOPE-PUBLIC.json`
**Duration**: 1 day, attended, executed in-session · **Risk**: medium (security; touches effects + types + eval)
**Registry reuse**: `none` for every milestone — a core effect-system change, no package capability.

Every acceptance criterion names a test function or a command. A criterion without one is not done.

## M1 — redirect hops are public-only (~60 LOC + ~120 test)
`internal/effects/net_authorize.go` (`forRedirectHop`, `checkRedirect`), `net_proxy.go` (`RoundTrip`).
- [x] `go test ./internal/effects -run TestRedirectHop_` passes:
  - `TestRedirectHop_HostnameToLoopbackRefused` (public httptest → `http://evil.test/` with `lookupIP` → 127.0.0.1, AllowLocalhost+AllowMetadata true)
  - `TestRedirectHop_HostnameToMetadataRefused` (→ 169.254.169.254)
  - `TestRedirectHop_LiteralLinkLocalRefused` (`Location: http://169.254.169.254/…`)
  - `TestRedirectHop_LocalhostNameRefused` (`Location: http://localhost/…`)
  - `TestRedirectHop_PublicTargetStillFollowed` (control)
- [x] `TestDirectMetadataCall_BareNetStillAllowed` passes (hop 0, `lookupIP` → 169.254.169.254, AllowMetadata).
- [x] Existing `go test ./internal/effects/...` stays green.

## M2 — `Net[scope=public]` in the type system (~50 LOC + ~60 test)
`internal/types/effects.go` (schema), `internal/types/effect_subsumption.go` (narrowing params).
- [x] `go test ./internal/types -run 'TestNetScope'` passes:
  `TestNetScopePublic_SchemaLegal`, `TestNetScope_UnknownValueRejected`,
  `TestNetScopePublic_SubsumesBareNet` (both directions), `TestNetScope_DoesNotLeakToOtherEffects`.
- [x] Command: `ailang check` (repo build) on `func f(u: string) -> string ! {Net[scope=public]} = httpGet(u)` → `No errors found`.
- [x] Command: `ailang check` on `examples/runnable/net_scope_public.ail` → `No errors found`.

## M3 — runtime enforcement (~80 LOC + ~100 test)
`internal/effects/net_scope.go` (counter, Push/Pop, Clone reset), `netPolicy`; `internal/eval` (`EffectNetScope`, push/pop, `replaceable`); `internal/gen/lower/program.go` (VM fail-closed).
- [x] `go test ./internal/effects -run TestNetScopePublic_` passes:
  `TestNetScopePublic_HostnameToLoopbackRefused`, `TestNetScopePublic_HostnameToMetadataRefused`,
  `TestNetScopePublic_PopRestoresBareBehaviour`, `TestNetScopePublic_CloneResets`.
- [x] `go test ./internal/eval -run TestNetScope` passes: `TestNetScope_ExtractedFromType`, `TestNetScope_FrameNotTailReplaced`.
- [x] `go test ./cmd/ailang -run TestNetScopePublic_EndToEnd` passes (an `.ail` program declaring `Net[scope=public]` run with AllowLocalhost+AllowMetadata against an httptest server on 127.0.0.1 is refused; the bare-Net twin succeeds).
- [x] `go test ./internal/gen/lower -run TestNetScopePublic_LoweredEvalOnly` passes.

## M4 — docs, changelog, gates, mutation (~docs)
- [x] `changelogs/unreleased/2026-10-02-net-scope-public.md`; `docs/docs/guides/parameterised-effects.md` + `docs/docs/reference/serve-api.md` (or the serve-api guide) mention; devtools prompt row.
- [x] Commands: `make check-file-sizes check-boundaries check-architecture-closure` exit 0; `gofmt -l` empty and `golangci-lint run` clean on touched dirs.
- [x] Mutation: each guard reverted in a scratch copy → a named test fails, mutant compiles (`go vet` of the mutant package exit 0).
- [ ] CI: `gh run list -R sunholo-data/ailang --branch dev --workflow ci.yml` green for the pushed head.

## Results (2026-10-02, executed in-session)

- Extra tests beyond the plan: `TestRedirectHop_PrivateHostnameRefused`, `TestDirectLocalhostCall_BareNetStillAllowed`,
  `TestRedirectHop_CheckRedirectRefusesBeforeDial` (added after the CheckRedirect-only mutant SURVIVED: the RoundTrip
  gate masked it), `TestNetScopePublic_LiteralLoopbackRefused`, `TestNetScopePublic_PublicHostStillAllowed`,
  `TestNetScopePublic_EmptyScopeIsNoop`, `TestNetScope_PushedAndPoppedAroundCall`, `TestNetScopeBare_LoweredNormally`.
- Deliberate test change: `TestRunPolicyE2E_RedirectToNonAllowlistedHostDenied` positive control (a same-host
  loopback redirect) now expects `DENIED` (design Conflict Surface §5).
- Mutation (scratch-copy, `go vet` exit 0 for every mutant): 14 mutants, 14 killed after the CheckRedirect test was added.
- Verified on `--bytecode` manually: the public function is refused on the VM path too.
