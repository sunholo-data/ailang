M6 is committed on `fleet/i118-rulings` as `880c6f78a`. It is not pushed. Every gate is green, and both mutants were killed and restored.

**Files:** 15 files, +227 lines. New: `design_docs/fleet-rulings.md` (27), `internal/mission/rulings_live_test.go` (106), four testdata snapshots, and `changelogs/unreleased/2026-11-17-fleet-rulings-ledger.md` (8). Edited (+2 each): five charters, the charter template, and both Gate 0 copies (`.claude` and `.agents`).

**Gates:**
- `gofmt -l internal/mission`: no output, rc 0
- `go vet ./internal/mission/`: rc 0
- `golangci-lint run ./internal/mission/...`: "0 issues.", rc 0
- `go test -run Ruling -count=1`: ok, rc 0
- `go test ./internal/mission/ -count=1`: ok, rc 0
- `mission_decisions.sh --check` on the ledger: "decision ledger valid: 2 rows", rc 0
- `--check` on the five charters and the template: all rc 0
- `sync-agents-skills.sh --check`: "in sync", rc 0; mirror `cmp`: rc 0
- `make check-changelog`: rc 0
- Charter link count: 1 per charter outside the ledger block
- Seed literal scan: 0 hits for the retired literal, 2 for the control

**Mutants (scratchpad copy, restored byte-identical):**
- M1: `forbid skills` → `forbid skill`. FAIL `rulings_live_test.go:47`: `unknown scope "skill"`. Killed by `TestRulingsLiveLedgerParses`.
- M2: pre-fix snapshot replaced with the current one. FAIL `rulings_live_test.go:82`: `pre-fix gate-3-route hits = [], want exactly 1 line`. Killed by `TestRulingsSeedQuietOnActivation`.

After restoring, `git diff` was clean. Both tests pass again.

**Deviations:**
1. **Commit message:** I used the message from your brief, not the plan's `docs(fleet): …` wording.
2. **Gate 0 insertion point:** the plan says "after line 4", but line 4 is a wrapped continuation of the First-action line. The `Attended setup:` line is line 5, so I inserted after line 5.
3. **Live test size:** it is 106 lines against the plan's ~70 estimate. The content follows the plan's table.
