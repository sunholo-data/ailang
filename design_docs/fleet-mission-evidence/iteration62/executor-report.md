Implemented M1_CAP_HIT_LOUD and marked it completed with evidence. Exactly four files changed:

- [run.go](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter62-cap-trunc/internal/mission/quorum/run.go)
- [cap_hit_test.go](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter62-cap-trunc/internal/mission/quorum/cap_hit_test.go)
- [Changelog fragment](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter62-cap-trunc/changelogs/unreleased/2026-10-27-quorum-reviewer-cap-hit-loud.md)
- [Sprint state](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter62-cap-trunc/.ailang/state/sprints/sprint_M-QUORUM-REVIEWER-CAP-HIT-LOUD.json)

All ten mutants exited **1** with failing assertions. Every restoration passed `cmp` with exit **0**. Tests below are subtests of `TestRunReviewerWith_CapHit`; assertion excerpts reference `cap_hit_test.go`.

| Mutation | Test row | Failing assertion excerpt |
|---|---|---|
| Delete inferred disjunct | a | `:130: error missing "truncated"` |
| Inference `>=` → `==` | a2 | `:130: error missing "truncated at 65537"` |
| Delete `length` disjunct | b | `:130: error missing "finish_reason=length"` |
| Hardcode label to `inferred` | b | `:130: error missing "reported by provider"` |
| Drop limit’s `n >= budget` check | b2 | `:135: error must exclude "consumed"` |
| Drop limit’s positive-budget guard | b3 | `:135: error must exclude "consumed"` |
| Drop inference’s empty-reason guard | c | `:123: ordinary parse error changed` |
| Drop inference’s token comparison | d | `:135: error must exclude "truncated"` |
| Drop inference’s positive-budget guard | f | `:135: error must exclude "truncated"` |
| Move cap rejection before parsing | e | `:103: valid pass at cap must remain present with no absence/error and pass result` |

Final gates:

| Command | Exit | Key output |
|---|---:|---|
| `go test ./internal/mission/quorum/...` | 0 | `ok …/internal/mission/quorum 5.920s` |
| `go vet ./internal/mission/quorum/...` | 0 | No output |
| `gofmt -l internal/mission/quorum/` | 0 | No output |
| `make check-changelog` | 0 | `5 changelog fragment(s) valid`; root index valid |
| `grep -c 'FinishReason' internal/ai/ollama/client.go` | 1 | `0` — expected no-match exit |
| `grep -n 'Model:' internal/mission/quorum/call.go` | 0 | `44: Model: c.handler.Model(),` |
| `grep -n 'out := &ReviewerOutcome' internal/mission/quorum/run.go` | 0 | `113: out := &ReviewerOutcome{Model: modelID}` |
| `git status --porcelain` | 0 | Exactly the four files listed above |

No behavioral disagreement or unfinished work. Decision 1 required one syntax correction: `hit = …` because `hit` is already a named return variable. No gate required a sandbox qualification. `.mut/` was removed; no git writes or message commands were run.