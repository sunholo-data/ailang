M2 is committed on `fleet/i118-rulings-ledger` as `358e9b94d`, not pushed. All four listed mutants were killed, and the tree is clean.

**Files**
- `internal/mission/rulings_corpus.go`: 138 lines
- `internal/mission/rulings_corpus_test.go`: 187 lines

**Gates (final state)**
- `gofmt -l internal/mission`: no output, rc 0
- `go vet ./internal/mission/`: rc 0
- `golangci-lint run ./internal/mission/...`: `0 issues.`, rc 0
- `go test ./internal/mission/ -run 'Rulings' -count=1`: ok, rc 0
- `go test ./internal/mission/ -count=1`: ok, rc 0 (27s)

**Mutants** (scratchpad copy in `/tmp/i118-mut`, restored byte-identical after each, then the test passed again)
1. `filepath.Glob` instead of WalkDir → `--- FAIL: TestRulingSkillFiles_WalkDirDepthAndFilter`
2. Disagreement check disabled → `rulings_corpus_test.go:72` and `--- FAIL: TestRulingsRoot_Derivation`
3. `charterPath` in-repo uses `m.Workdir` → `rulings_corpus_test.go:90: got "/wd/design_docs/x.md" want "/reg/design_docs/x.md"`
4. `literalHits` ignores charter exclusion → `rulings_corpus_test.go:147: got [2 5] want [2]`

All four killed; none survived.

**Deviations (stated)**
- The plan says to add `rulingsFixture` "under M3", but `TestRulingsRoot_Derivation` needs it now. I put only `newRulingsFixture`, `write` and `mission` in `rulings_corpus_test.go`. The `ledger` helper and `linkedCharter` are left for M3, since unused helpers would fail `unused` lint.
- I added `TestRulingsDisplayPath`, which the plan's table doesn't list. `displayPath` is only used by M3, so without a test in M2 the `unused` linter would flag it.
- Mutant 2 is written as `if false && len(seen) > 1`, and mutant 4 as `if false && !outside[i]`. Both are hand-applied equivalents of the plan's descriptions.
- The commit message is the plan's "root, charters, skills walk, matcher" subject, not the shorter subject given in the plan's M2 acceptance list.
- Git showed a pre-push hook warning at session start: the hook is not ours and the stdlib/ci-quick gates aren't running on push. I did not push, so this doesn't affect this commit.
