# M-RUN-POLICY-WORKER-CACHE: evaluation feedback repair

Refs #1547; PR #1743; evaluation round 2 requested.

Based on PR head 67af4370341d25e0ce00b54bab9bb143e38385eb.
Split TestPolicyCachePlacement into independent subtests. The relative-path case skips
with runtime.GOOS and the error when filepath.Rel cannot represent a cross-volume path.
Absolute, missing-leaf and outside cases continue on every platform; the symlink case
runs wherever symlink creation is supported and skips only itself otherwise.

Validation:
- go test ./cmd/ailang/... -run 'TestWorkerCache|TestPolicyCache' -count=1: PASS.
- gofmt and git diff --check: PASS.
- Direct equivalent of make test-core: all packages passed except internal/effects,
  whose existing SQLite brain tests fail because CGO is disabled and no C compiler is installed.
- Sprint JSON parsed and checked for populated milestones and valid dependencies via Python.
  The prescribed validator cannot run because jq is missing.
- make/linter checks unavailable: make and golangci-lint are absent.
- Windows execution awaits CI; this Linux environment cannot run Windows tests.

Publishing: local repair branch coordinator/task-ed800792-repair. PR source branch is
coordinator/task-00bafb12. No push performed: GITHUB_TOKEN is absent and the GitHub
connector reports reauthentication required. Coordinator must publish this repair to
PR #1743 and trigger independent evaluation round 2. No merge performed.

Evaluator handoff attempted via sprint-executor skill: CLI refused because no agent
serves sprint-evaluator on this environment. No forced send; coordinator must dispatch it.
