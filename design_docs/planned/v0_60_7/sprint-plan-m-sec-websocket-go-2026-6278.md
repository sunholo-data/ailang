# Sprint plan for GO-2026-6278

Target v0.60.7. Update Gorilla WebSocket to the first fixed version above
v1.5.3 and obtain a green vulnerability gate and required checks on the
dedicated PR to `dev`. The Windows compile request
`inbox_1795447998263_b0372c15` and the pipeline package timeout are separate
triage items. [Approved design](m-sec-websocket-go-2026-6278.md)

## Baseline and decisions

- The baseline is `dev` at `d3c2c7850b4be0015861ee395db18f87b5f5f92f`.
  The current checkout also pins Gorilla WebSocket v1.5.3. The public job
  page confirms gate exit 1. The advisory ID comes from the triage report
  because the job logs require sign-in. [CI run](https://github.com/sunholo-data/ailang/actions/runs/45289159090/job/138093356254)
- D1 pins `v1.5.4-0.20240701034025-d67f41855da4`, the first fixed version
  above v1.5.3. The module proxy still resolves `@latest` to v1.5.3.
  [Go advisory](https://pkg.go.dev/vuln/GO-2026-6278) · [Fixed module](https://proxy.golang.org/github.com/gorilla/websocket/@v/v1.5.4-0.20240701034025-d67f41855da4.info) · [Latest module](https://proxy.golang.org/github.com/gorilla/websocket/@latest)
- The full tag-to-pin comparison contains 14 commits, including client,
  server, and proxy changes beyond the mask-key fix commit. The new module
  requires `golang.org/x/net v0.26.0`, below AILANG's existing v0.63.0 pin.
  Confirm the selected graph and test all six direct consumer packages.
  [Upstream comparison](https://github.com/gorilla/websocket/compare/v1.5.3...d67f41855da4) · [Upstream module manifest](https://proxy.golang.org/github.com/gorilla/websocket/@v/v1.5.4-0.20240701034025-d67f41855da4.mod)
- D2 keeps the empty vulnerability allowlist. D4 calls for a dedicated PR.
  There are no human-decision gates. D3 remains a maintenance follow-up
  when a tagged release containing the fix is available.

## Day 1

Budget half a day, with about two hours of active work and CI turnaround on
the critical path. Estimate eight changed lines and zero new test or example
lines. The seven-day velocity script saw only the design commit in this
shallow checkout, so the estimate uses the size of the manifest/changelog
diff and the checks below. The v0.53.1 x/net advisory bump provides the
release-note precedent.

### M1_PIN — Update and verify the selected Go module

Allow 30 minutes and three changed manifest lines. Use the CI Go toolchain
(1.26.11), starting from current `dev` when execution is authorized. If the
fixed version is already selected there, verify it before making any edits.

```bash
go get github.com/gorilla/websocket@v1.5.4-0.20240701034025-d67f41855da4
go mod tidy
go list -m github.com/gorilla/websocket golang.org/x/net
go mod verify
git diff -- go.mod go.sum
```

- [ ] Gorilla resolves to the exact fixed pin and x/net remains v0.63.0
  against the reviewed baseline. Compare the selected module list before
  and after the bump. Investigate any unexpected version, checksum, or
  toolchain drift before proceeding.
- [ ] The manifest diff changes the single requirement and replaces its
  two checksums. `go mod verify` succeeds.

### M2_VERIFY — Run focused checks and collect CI evidence

Depends on M1_PIN. Allow 90 minutes of active work plus CI wait. Add a
roughly five-line `### Security` fragment at
`changelogs/unreleased/YYYY-MM-DD-websocket-go-2026-6278.md` using the
execution date. Record the old and fixed pins, the advisory, and the
follow-up to use the first tag containing the fix. The fragment follows
current sprint-planner conventions.

Run the focused executor checks, including explicit consumer tests because
`make test-changed` discovers packages from changed Go source files.

```bash
go build ./...
make test-changed
go test -count=1 ./internal/websocket ./internal/platform/streamws ./internal/apiserver \
  ./internal/effects ./internal/executor/codex ./internal/server
make check-changelog
```

Mirror the repository's vulnerability job with a Bash shell. Preserve the
JSON stream for confirmation that the advisory is absent from findings.

```bash
go install golang.org/x/vuln/cmd/govulncheck@v1.3.0
go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter
set -o pipefail
scan_json="$(mktemp)"
govulncheck -format json ./... | tee "$scan_json" | ./bin/govulncheck-filter
jq -es 'length > 0 and all(.[]; .finding.osv? != "GO-2026-6278")' "$scan_json"
```

- [ ] All focused checks succeed. The scan pipeline exits 0 and the raw
  stream contains no GO-2026-6278 finding at any level. The allowlist
  remains empty. Investigate scanner errors or unrelated new findings
  separately and record them if they prevent a green gate.
- [ ] The PR contains the manifest changes and the security fragment.
  Record the candidate SHA and CI run URL. Confirm the PR runs the code
  lane and that `govulncheck (vuln gate)` and `checks` actually succeed.
  Confirm `test`, `lint`, `build`, `test-windows`, and
  `float determinism (arm64)` also pass before merge. If a separate
  incident blocks a required context, record its job link and keep
  remediation in its own work item.
- [ ] After merge through the existing process, record the first `dev`
  push CI run and confirm the vulnerability gate and `checks` succeed.

## Registry reuse

- M1_PIN — `none` in the AILANG registry. This milestone changes an
  existing Go module pin and introduces no package-like AILANG capability.
- M2_VERIFY — `none` in the AILANG registry. This milestone uses the
  repository's current verification targets and CI workflow.
