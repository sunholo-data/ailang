# Iteration 140 — controller gates (outside the sandbox)

Base `03612b56c` (baseline, before any change), all rc 0:

- `go build ./cmd/ailang`
- `go test ./cmd/ailang -run Rotate -count=1`
- `go test ./internal/mission -count=1` (27.4 s)
- `make check-context-docs`
- `tools/launchd/test_agents_skills_sync.sh` 7/7
- `tools/launchd/test_skill_sync.sh`

Candidate `62f569387`, all rc 0:

- `go build ./cmd/ailang`
- `go test ./cmd/ailang -run Rotate -count=1`
- `go test ./internal/mission -count=1` (27.1 s)
- `make check-context-docs`
- `test_agents_skills_sync.sh` 7/7
- `test_skill_sync.sh`
- `gofmt -l` on the three Go files (empty)
- `go vet ./cmd/ailang`

Reconstruction: M1 snapshot → `go test ./cmd/ailang -run Rotate` ok → commit. M2 snapshot → ok → commit. The cumulative `git diff --stat 71a111c68` matched the executor's final-tree stat (4 files, 290+/15−).

Smoke, ticket case (built binary, driver `AILANG_MISSION_REGISTRY` = pin `missions/`):

- A scratch record checkout holding World's origin/dev log and archive.
- `rotate-log world --index-only --workdir "$PWD"` printed `INDEX` under the scratch checkout (416 iterations).
- The World main checkout's porcelain and `world-mission-*` shasums were identical before and after.
- `--workdir /tmp/<missing>` gave `Error: --workdir target <abs>/design_docs/world-mission-log.md: stat …: no such file or directory`.
- Old binary (`~/go/bin/ailang`, v0.61.1-dirty `cb6bb75`): `Error: unknown flag "--workdir"`.

PR #2471 CI on `62f569387`:

- green: `lint`, `build`, `checks`, `docs-gate`, `launchd drivers (bash 3.2)`, `govulncheck`, CodeQL;
- red: `test` (job 139367533573, log 251,079 B, 170 `ok` package lines). Its failures are `TestRealRegistry_PostMigrationSplitCounts` and `TestCachePipeline_EmbeddedKeys`, the same two as dev `e3595b4f9`;
- red, non-required: `test-windows` and the matrix `Build` jobs.

The record's index was regenerated with this iteration's binary: `rotate-log fleet --index-only --workdir "$PWD"` from the record worktree. INDEX in the record worktree (140 iterations); pin porcelain 0.
