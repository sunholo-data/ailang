### Fixed — `build.yml`'s test step uses the same 900 s package budget as `ci.yml` on Windows (2026-11-10)

The Build workflow's `Run tests` step ran `go test` with Go's default 600 s per-package timeout, while
`ci.yml`'s `test-windows` job has used a derived 900 s budget since `cmd/ailang` outgrew 600 s on
`windows-latest` in October. The two had drifted. On 2026-11-10 the Build workflow's Windows leg failed
twice on one pull request with "this PACKAGE has outgrown its go test -timeout budget" (600 s and 590 s),
although that PR's own new tests take about two seconds. Both workflows now use 900 s; the release gate
requires the Build workflow's per-OS jobs, so a budget that random load can exceed blocked releases too.
