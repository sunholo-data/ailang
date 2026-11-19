MISSION-ROLE: executor

You are the EXECUTOR for fleet mission iteration 118 (AILANG repo; unattended, no human present).
Work ONLY in `/Users/voightkampff/.ailang-driver-pin/.fleet-i118-rulings` (branch `fleet/i118-rulings-ledger-build`,
M1–M7 committed). This is a FIX ROUND: CI on PR #2402 is red on Windows only (`test-windows`, `Build windows-latest`).
Linux and macOS are green. Exactly these five tests fail on Windows (CI log excerpts):

1. `TestRulingsCharterPath_Location` — rulings_corpus_test.go:101: `absolute doc must be unchanged: got "\\reg\\abs\\c.md" want "\\abs\\c.md"`.
   Cause: `filepath.Join(string(filepath.Separator), "abs", "c.md")` is NOT absolute on Windows (no volume), so
   `charterPath` treats it as relative. Fix the TEST: build absolute paths from `t.TempDir()` (which has a volume).
2. `TestRulingsDisplayPath` — rulings_corpus_test.go:178: `under root: got "\\reg\\.claude\\skills\\mission-a\\SKILL.md" want ".claude\\skills\\mission-a\\SKILL.md"`.
   Cause: a literal `/reg` root is not absolute on Windows. Fix the TEST to use `t.TempDir()` roots.
3. `TestRulings_SkillHitNamesRulingFileAndLine` — rulings_test.go:69: detail
   `D-FR-1 .claude\\skills\\mission-a\\resources\\g.md:3 contains …` lacks `.claude/skills/mission-a/resources/g.md:3`.
   Fix the CODE: `displayPath` in `internal/mission/rulings_corpus.go` must return slash-separated paths
   (`filepath.ToSlash(rel)`), so finding details are identical on every OS. Keep returning the input unchanged
   (but also `filepath.ToSlash`-ed? decide: details must be OS-independent for repo-relative paths; an absolute
   out-of-root path may stay native) — state your choice. Update the `displayPath` test expectations accordingly.
4. `TestRulings_CharterLocationEndToEnd` — rulings_test.go:375: `want one hit under the root, got [... design_docs\a.md:9 ...]`.
   Same cause as 3; after the ToSlash fix make sure the test's expectation uses forward slashes.
5. `TestDoctor_RulingCheckRunsForLoadedMission` — rulings_test.go:456: `LoadFile: … toml: line 3 (last key "workdir"):
   expected eight hexadecimal digits after '\U'`. Cause: a Windows temp path with backslashes is written inside a
   TOML basic (double-quoted) string. Fix the TEST: write the path as a TOML literal string (single quotes) — e.g.
   `workdir = '` + dir + `'` — or `filepath.ToSlash(dir)`. Check how the existing tests in `internal/mission/*_test.go`
   that write TOML with temp paths do it (`grep -n "workdir = " internal/mission/*_test.go`) and follow that convention.

Also: grep ALL the new rulings test files (`internal/mission/rulings*_test.go`) for any other literal Unix-absolute path
(`"/` at the start of a path string) or a path compared with a hard-coded `/` separator, and fix any you find the same way.
You cannot run Windows here. Compensate: (a) `GOOS=windows go vet ./internal/mission/` must pass; (b) reason per test
about what changes on Windows and say so in the commit body.

Done when: `gofmt -l internal/mission` empty; `go vet ./internal/mission/` and `GOOS=windows go vet ./internal/mission/`
rc 0; `golangci-lint run ./internal/mission/...` 0 issues; `go test ./internal/mission/ -count=1` and
`go test ./internal/mission/ ./cmd/ailang/ -run 'Mission|Doctor|Ruling' -count=1` pass. Mutation: revert your
`displayPath` ToSlash change in a scratchpad copy and show which test now fails on THIS OS (if none does, add an
OS-independent assertion that would — e.g. feed a path built with `filepath.Join` and assert the detail uses `/`).
One commit: `fix(mission): rulings check OS-independent paths (Windows CI)`, ending with
`Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>`. Do not push.
Foreground only; nothing will re-invoke you, so never end your turn announcing a wait. Final message ≤12 lines.
