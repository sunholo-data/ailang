# Sprint Plan — M-PROMPT-VERSION-FREEZE

**Design doc**: [`m-prompt-version-freeze-on-first-bank.md`](m-prompt-version-freeze-on-first-bank.md) (PLANNED, quorum-cleared at r3)
**Sprint ID**: `M-PROMPT-VERSION-FREEZE`
**Ruling**: decision-ledger row **D-41**, option **(c)** — a prompt version is mutable until first banked use, then frozen.
**Base commit**: `70f4e0660` (`origin/dev`). Worktree HEAD `fa23e950e` adds only the design doc, so the code tree under test **is** `70f4e0660`.
**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter291`, branch `sprint/iter291-prompt-freeze`
**Risk**: M1 **medium** (one blocking pre-existing defect, §2; one existing-test constraint, §3). M2 medium (git plumbing). M3 **high** (behaviour change on the live eval path). M4 medium.

---

## 0. SCOPE RULING (binding, from the mission controller)

**M1 is the ONLY milestone to be EXECUTED this iteration.** M1 is the design doc's own declared
*"minimum viable ruling-compliance"*: it makes the BANKED/NEVER-BANKED state representable,
records it for all 59 versions in **both** registries, and enforces it with a teaching error on
the standard-mode path.

**M2, M3 and M4 are planned in full here but are follow-on iterations.** Each is specified to be
**independently landable** on top of M1 alone: no milestone depends on a later one, each ends with
its own green gate set, and §9 gives the file-ownership split that keeps the commits bisectable.

If the executor finishes M1 early it must **stop** and report, not start M2.

---

## 1. EXECUTOR GROUND RULES — read before anything else

1. **The executor may NOT run any git write operation.** No `git add`, `commit`, `stash`,
   `checkout`, `branch`, `restore`, `rebase`, `tag`. The controller builds commits. Read-only git
   (`git ls-files`, `git status`, `git diff`, `git merge-base`, `git show`) is fine and is required
   by the corpus predicate.
2. **Snapshots**: on finishing each milestone, copy the **full content** of every file that
   milestone created or modified into `.snap/M<k>/`, preserving repo-relative paths, **cumulatively**
   (`.snap/M2/` contains M1's files too). This iteration that means `.snap/M1/` only.
3. **`go test -run <Nonexistent>` exits rc=0.** Any gate of the form "this test exists and passes"
   MUST be `go test <pkg> -run '<Name>' -v > /tmp/o 2>&1; grep -c -- '--- PASS: <Name>' /tmp/o`
   and assert the expected integer. A bare `rc=0` is a vacuous gate here.
4. **Capture rc without a pipe**: `cmd > /tmp/out 2>&1; rc=$?`. Never `${PIPESTATUS[0]}` (empty in
   zsh), never `|| echo 0` wrapped around `grep -c`.
5. **`go build ./...` is rc=1 at base** (pre-existing, unrelated to this work). It is
   **DISQUALIFIED** as an acceptance command. Scope every build gate to touched packages.
6. Tests must never mutate the shared working tree. All fixtures under `t.TempDir()`. The one
   exception is the two *real* registry files, which M1's migration legitimately rewrites — that
   is the product, not a test side effect.

---

## 2. Measured baselines

### 2.1 Supplied by the controller at `70f4e0660` (use as-is, do not re-derive)

| # | Command | rc |
|---|---|---|
| B1 | `test -z "$(gofmt -l internal/eval_harness cmd/ailang)"` | 0 |
| B2 | `go build ./cmd/ailang ./internal/eval_harness/...` | 0 |
| B3 | `go build ./...` | **1 — RED AT BASE, DISQUALIFIED** |
| B4 | `go vet ./internal/eval_harness/... ./cmd/ailang/` | 0 |
| B5 | `make check-file-sizes` | 0 |
| B6 | `go test ./internal/eval_harness/...` | 0 |
| B7 | `go test ./cmd/ailang/ -run 'TestAILANGPromptLoading|TestPromptDisambiguation'` | 0, **2 PASS** |

### 2.2 Measured first-party by the sprint-planner at `fa23e950e` (this plan adds `internal/prompt` to the gate scope; M1 edits that package, so it needed its own baseline)

| # | Command | Observed |
|---|---|---|
| P1 | `test -z "$(gofmt -l internal/eval_harness cmd/ailang internal/prompt)"` | rc=0 |
| P2 | `go build ./cmd/ailang ./internal/eval_harness/... ./internal/prompt/...` | rc=0 |
| P3 | `go vet ./internal/prompt/` | rc=0 |
| P4 | `go test ./internal/prompt/` | rc=0, `ok … 0.350s` |
| P5 | `go test ./cmd/ailang/ -run 'TestAILANGPromptLoading\|TestPromptDisambiguation' -v \| grep -c -- '--- PASS'` | `2` (confirms B7 in the grep form gates must use) |
| P6 | corpus sweep — `git ls-files 'eval_results/baselines/*.json' \| xargs jq -r 'if has("prompt_version") then .prompt_version else "ABSENT" end' \| sort \| uniq -c \| sort -rn` | `9522 ABSENT, 6537 python, 210 v0.3.21, 138 v0.5.0, 138 v0.4.8, 114 v0.6.2, 92 v0.6.6, 76 v0.4.1, 62 v0.4.7, 62 v0.11.4, 61 v0.7.4, 57 v0.9.0, 51 v0.12.1, 38×(v0.4.0,.2,.4,.5,.6), 32 v0.3.23, 1 v0.6.5` — **17,343 total, agrees exactly with doc rows V-F and N-2** |
| P7 | corpus sweep wall-clock | **1.87 s** over 195 MB / 17,343 files. `--check`'s derivation (Q1) is cheap enough for `make ci`. |
| P8 | registry∩corpus (re-derivation of V-I) — `jq --arg k … '.versions\|has($k)'` for all 19 corpus ids | 19 × `true`; **positive control** `python` → `true`, **negative control** `v9.9.9-nonexistent` → `false` |
| P9 | split arithmetic — `comm -23 <(all 59 keys) <(19 corpus ids)` | rest = 40; minus active `v0.16.6` = **39**; non-AILANG members of the 39 are exactly `aver, go, javascript, moonbit` — **19 + 39 + 1 = 59 confirmed** |
| P10 | `shasum -a 256 prompts/versions.json cmd/ailang/prompts/versions.json` | identical `97decb4d7dd3…ef06c99e` (re-confirms V-J); `diff <(jq -S '.versions\|keys' src) <(… mirror)` empty |
| P11 | **whole-registry hash audit** (new; not in the doc) | **1 mismatch: `aver`** — see §2.3. Control: the other 58 agree. |
| P12 | mirror `.md`/hash audit over `cmd/ailang/prompts/` | same single `aver` hit; no missing mirror files (all 59 present) |
| P13 | registry shape — `jq -r '[.versions[]\|keys[]]\|unique[]'` / `keys_unsorted\|join(",")` | entry keys are exactly `file,hash,description,created,tags,notes`, **in that order, for all 59**; top-level keys exactly `active,notes,schema_version,versions`; version keys are in **chronological, NOT alphabetical** order (`v0.16.6` last); file ends with `\n` |
| P14 | `git ls-files 'eval_results/baselines/'` vs `…'*.json'` | 17,372 vs 17,343 — the 29 extras are `summary.jsonl` files; the `*.json` pathspec is depth-agnostic (git fnmatch), covering all 4/5/6-segment paths |

**Gates newly proposed by this plan and measured green at base**: P1, P2, P3, P4. No proposed gate is red at base.

### 2.3 BLOCKER — `aver` carries a stale hash at HEAD in BOTH registries

```
HASHMISMATCH aver exp=190bf4244a00bb0605d41ac01d4de415a6131a3a961aa6bca1689b23bc4d521d
                  act=1f5c9654e39c411bcbedc35446ffa96c1d6dc8d7b9dd0fbec01b95ecf35f0157
```

`prompts/aver.md` and `cmd/ailang/prompts/aver.md` are byte-identical to each other (`1f5c9654…`);
only the recorded hash is stale, in both registry files identically. Introduced by
`0fcae6055 fix(eval): adopt upstream Aver prompt + enrich failures with aver check`.

**Why it blocks M1 as written**: Q4 puts `aver` in the 39-entry `legacy` bucket; Q3 requires
`freeze` to refuse when file bytes ≠ recorded hash; L3(b) makes such a frozen entry a violation.
A literal M1 therefore either refuses to migrate or writes a frozen lie, and AC5's
*"on unmodified post-migration `dev`, `--check` exits 0"* becomes unreachable. It also pre-breaks
M3: once `internal/prompt.LoadPrompt` enforces frozen hashes, every `aver` eval hard-fails.
Today the defect is invisible because `internal/prompt` never compares the hash (doc N-3) and
`langreg/aver.go:33-37` has the same silent `(DefaultPrompt(), "default", nil)` fallback as
`ailang.go`/`python.go` — a **third** instance of the N-4 pattern, which the doc does not list.

**Resolution adopted (R-A)** — M1 **Task 0**: repair `aver`'s `hash` to `1f5c9654…` in both
registries before migrating. This is the never-banked workflow the ruling explicitly permits:
`aver` has **zero** corpus citations (P6 has no `aver` row), so under D-41(c) it is NEVER-BANKED
and in-place editing with hash regeneration is legal. Freezing then locks the *current* bytes and
breaks no provenance claim, and AC1's 19/39/1 is preserved exactly.
Rejected: R-B (leave `aver` unfrozen) → 19/38/2, contradicts AC1.
**Controller must confirm R-A.** If it declines, M1 cannot satisfy AC1 and AC5 simultaneously.

Add `langreg/aver.go:33-37` to M3's fallback-removal list (a doc omission, not a plan deviation).

---

## 3. Constraints imposed by existing tests (M1 must not break these)

Both live in `internal/eval_harness/prompt_loader_test.go` and are covered by gate B6.

| Existing test | Constraint on M1 |
|---|---|
| `TestLoadPrompt_HashMismatch` (`:159` asserts `contains(err.Error(), "hash mismatch")`) | **The doc's Q5 never-banked message text — `hash for "v0.16.6" is stale.` — does NOT contain the substring `hash mismatch` and would turn this test red.** M1 MUST keep `hash mismatch for %q:` as the leading clause of the never-banked message and append the teaching lines after it. This satisfies both the existing test and AC3. |
| `TestLoadPrompt_PlaceholderHash` (`:205` asserts an **unfrozen** `PLACEHOLDER` entry loads successfully) | Q6's refusal must be scoped to `Frozen != nil` only. A blanket PLACEHOLDER ban turns this red and would also break the documented dev workflow. This test *is* the control that keeps mutation 5's fix from over-tightening. |

`TestAILANGPromptLoading` / `TestPromptDisambiguation` (V-H) load the active `v0.16.6`, which stays
mutable with a valid hash — unaffected, as Conflict Surface 8 predicts.

---

## 4. AC id cross-check against the design doc (the doc WINS; these are doc-internal disagreements)

| # | AC | Header tag in the doc's AC list | Milestones section says | Verdict |
|---|---|---|---|---|
| 1 | **AC8** | `(M3, mirror agreement)` | M2: *"Tests: AC4, AC5, second half of AC6, **AC8**, AC9's --check half"* | **REAL CONTRADICTION.** AC8 exercises `ailang prompt freeze --check`'s mirror `.md` comparison — that is L3(d), which the doc's own M2 description and its L3 table both place in M2. M3 touches only `internal/prompt` + `langreg` and adds nothing to `--check`. Two of three sites say M2. **This plan puts AC8 in M2** and flags the header tag as the defect. Controller to adjudicate. |
| 2 | **AC6** | `(M2, frozen PLACEHOLDER ban)` | M1: *"first half of AC6"*; M2: *"second half of AC6"* | Labelling only. AC6's own text has two clauses — `LoadPrompt` refuses (loader → M1) and `--check` non-zero (CI → M2). Plan follows the split; header tag should read `(M1+M2)`. |
| 3 | **AC9** | `(M2, mirror REGISTRY agreement)` | M1: *"AC9's jq-count/diff half"* | Labelling only. AC9's text has a migration half (counts on the mirror + empty `jq -S` diff → M1) and a tamper half (`--check` red → M2). Plan follows the split. |
| 4 | **AC5** | `(M2, derivation audit)` | M1: *"`--check` (derivation + hash-integrity subset, i.e. **L3(a)**(b) callable locally)"* | **Coverage gap.** M1 *implements* L3(a) but no M1-assigned AC exercises it, so an M1-only iteration would ship the derivation untested. This plan **pulls AC5 into M1** (both halves). This adds coverage without contradicting the doc — AC5's own text is fully satisfiable by M1's code, and M2 re-runs it as a regression pin. |

No other mismatches. `Testing Strategy`, the Mutation Table (rows 1,2,3,5,8,10 → M1) and the
Enforcement-layer table are internally consistent with the milestone assignments used here.

---

## 5. M1 — the milestone to execute (detailed, literal)

> Doc scope: *"`FrozenMarker` structs in both loaders; `ailang prompt freeze <version>` / `--migrate` /
> `--check` (L3(a)(b)); every marker write to BOTH registry files; the migration commit (19/39/1);
> the two-state error in `internal/eval_harness/prompt_loader.go`; frozen-PLACEHOLDER refusal."*

### 5.1 Files

**Create**
| Path | Purpose | est. LOC |
|---|---|---|
| `internal/eval_harness/prompt_frozen.go` | `FrozenMarker`, `IsHexSHA256`, the three error constructors | 90 |
| `internal/eval_harness/prompt_frozen_test.go` | AC2, AC3, AC6(loader half) | 200 |
| `cmd/ailang/prompt_freeze.go` | `runPromptFreeze` — flag parsing, dispatch, exit codes, help | 130 |
| `cmd/ailang/prompt_freeze_core.go` | ordered registry I/O, corpus scan, migration, L3(a)(b) check | 300 |
| `cmd/ailang/prompt_freeze_test.go` | AC1, AC5, AC9(migration half) | 320 |

**Modify**
| Path | Change | est. LOC |
|---|---|---|
| `internal/eval_harness/prompt_loader.go` | `PromptVersion` gains `Frozen *FrozenMarker`; rewrite the `LoadPrompt` verification block (currently `:76-91`) | +18 / −12 |
| `internal/prompt/loader.go` | `FrozenMarker` type + `VersionMetadata` gains `Frozen *FrozenMarker`. **Schema only — no enforcement in M1** (that is M3/L2) | +12 |
| `internal/prompt/loader_test.go` | one round-trip test | +30 |
| `cmd/ailang/prompt.go` | intercept the `freeze` subcommand before `promptFS.Parse`; add a `freeze` block to `printPromptHelp()` | +14 |
| `prompts/versions.json` | Task 0 `aver` hash repair + 58 `frozen` markers (tool output) | generated |
| `cmd/ailang/prompts/versions.json` | byte-identical copy of the above | generated |

Size check: `prompt_loader.go` 189→~195, `prompt_loader_test.go` 570 (untouched — new tests go in the
new file, deliberately, to stay clear of the 800-line gate), `prompt.go` 265→~279, `help.go`
untouched (CLI docs are M2). All new files ≪ 800. **`make check-file-sizes` stays green.**

`FrozenMarker` is declared **twice**, once per package, with identical JSON tags.
`internal/prompt` must NOT import `internal/eval_harness` — `langreg` (inside `eval_harness`)
imports `internal/prompt`, so the reverse edge is an import cycle.

### 5.2 Task order

**Task 0 — `aver` hash repair (blocker, §2.3).** Set `.versions.aver.hash` to
`1f5c9654e39c411bcbedc35446ffa96c1d6dc8d7b9dd0fbec01b95ecf35f0157` in **both** registries.
Nothing else in either file changes. Verify `shasum -a 256` of the two registries still match each
other, and that the whole-registry audit (P11) now reports 0 mismatches.

**Task 1 — `internal/eval_harness/prompt_frozen.go`.** Exactly these declarations:

```go
package eval_harness

// FrozenMarker records that a prompt version's bytes are immutable because the version
// has been used in at least one banked eval baseline. Decision D-41(c).
// ABSENT (nil) means never-banked, i.e. mutable.
type FrozenMarker struct {
	At              string `json:"at"`               // YYYY-MM-DD
	Reason          string `json:"reason"`           // "banked" | "legacy" | "manual"
	EvidenceCount   int    `json:"evidence_count"`
	EvidenceExample string `json:"evidence_example"` // "" for reason:"legacy"
}

// IsHexSHA256 reports whether s is exactly 64 lowercase hex characters.
func IsHexSHA256(s string) bool

// FrozenHashMismatchError builds the teaching error for a FROZEN version whose file
// bytes no longer match its recorded hash. (Q5, mutation 1.)
func FrozenHashMismatchError(versionID string, v PromptVersion, actualHash string) error

// MutableHashMismatchError builds the teaching error for a NEVER-BANKED version with a
// stale change-detector hash. MUST retain the substring "hash mismatch" (see plan §3).
func MutableHashMismatchError(versionID string, v PromptVersion, actualHash string) error

// FrozenUnenforceableHashError builds the refusal for a frozen version whose recorded
// hash is not a 64-hex digest. (Q6, mutation 5.)
func FrozenUnenforceableHashError(versionID string, v PromptVersion) error
```

Message templates — **the quoted literals are load-bearing; tests assert on them**:

```go
// FrozenHashMismatchError
"prompt version %q is FROZEN: it is cited by %d banked baseline files under " +
"eval_results/baselines/ (e.g. %s; frozen %s, reason: %s — decision D-41c). " +
"Its bytes are immutable: editing %s in place would silently change what those " +
"baselines measured.\n" +
"To change the teaching prompt, create a NEW version instead:\n" +
"  .claude/skills/prompt-manager/scripts/create_prompt_version.sh <new-id> %s \"<why>\"\n" +
"(expected sha256 %s, got %s)"
//  args: versionID, v.Frozen.EvidenceCount, v.Frozen.EvidenceExample, v.Frozen.At,
//        v.Frozen.Reason, v.File, versionID, v.Hash, actualHash

// MutableHashMismatchError  — leading clause preserved for TestLoadPrompt_HashMismatch (§3)
"hash mismatch for %q: expected %s, got %s. This version is not yet banked, so " +
"in-place editing is allowed (D-41c) — regenerate the change-detector hash:\n" +
"  shasum -a 256 %s   # then update the \"hash\" field in prompts/versions.json"
//  args: versionID, expectedPreview, actualPreview, v.File

// FrozenUnenforceableHashError
"prompt version %q is FROZEN but its recorded hash %q is not a 64-hex sha256: refuse " +
"to load — a frozen version with an unenforceable hash is a freeze with no invariant " +
"(D-41c, Q6). Restore the recorded hash from git history, or bump to a new version."
```

`expectedPreview`/`actualPreview` reuse the existing 16-char truncation from `prompt_loader.go:82-88`.

**Task 2 — rewrite the verification block in `LoadPrompt`** (`internal/eval_harness/prompt_loader.go`,
replacing lines 76-91). Exact structure and branch order:

```go
	if version.Frozen != nil {
		if !IsHexSHA256(version.Hash) {
			return "", FrozenUnenforceableHashError(versionID, version)   // Q6 / AC6(a)
		}
		if actual := computeSHA256(content); actual != version.Hash {
			return "", FrozenHashMismatchError(versionID, version, actual) // Q5 / AC2
		}
		return string(content), nil
	}
	// never-banked: V-C semantics preserved exactly, message upgraded
	if version.Hash != "PLACEHOLDER" {
		if actual := computeSHA256(content); actual != version.Hash {
			return "", MutableHashMismatchError(versionID, version, actual) // AC3
		}
	}
	return string(content), nil
```

**Task 3 — `internal/prompt/loader.go` schema field.** Add the `FrozenMarker` type (same four
fields, same JSON tags) directly above `VersionMetadata`, and one field to `VersionMetadata`:
`Frozen *FrozenMarker \`json:"frozen,omitempty"\``. **Do not touch `LoadPrompt`** — enforcement is M3.

**Task 4 — `cmd/ailang/prompt_freeze_core.go`.** Exactly these declarations:

```go
// registryPair is the two files every marker write must touch in lockstep (Q3, V-J).
type registryPair struct{ Source, Mirror string } // "prompts/versions.json", "cmd/ailang/prompts/versions.json"

// registryEntry mirrors one versions.json entry, field order matching the file (plan P13).
type registryEntry struct {
	File, Hash, Description, Created string
	Tags                             []string
	Notes                            string
	Frozen                           *eval_harness.FrozenMarker
}

// orderedRegistry preserves version-key order so a migration diff shows ONLY added markers.
type orderedRegistry struct {
	SchemaVersion string
	VersionKeys   []string            // chronological order as read from disk
	Versions      map[string]*registryEntry
	Active        string
	Notes         []string
}

func loadOrderedRegistry(path string) (*orderedRegistry, error)
func writeOrderedRegistry(path string, r *orderedRegistry) error

type corpusEvidence struct {
	Count   int
	Example string // first tracked path, in git ls-files order
}

// scanCorpus derives banked(V) from `git ls-files 'eval_results/baselines/*.json'` (Q2).
func scanCorpus(repoRoot string) (map[string]corpusEvidence, error)

// corpusScanner is indirected so tests can inject a fixture corpus without a git repo.
var corpusScanner = scanCorpus

// migrateRegistries applies the Q4 split to BOTH files of the pair, writing them from ONE
// in-memory registry so byte-identity is structural, not checked after the fact.
// Returns (banked, legacy, mutable) counts.
func migrateRegistries(repoRoot string, today string) (int, int, int, error)

// freezeVersion writes a reason:"banked" marker for one version to BOTH files.
func freezeVersion(repoRoot, versionID, today string) error

// checkRegistries runs L3(a) (corpus derivation) + L3(b) (hash integrity) and returns
// human-readable violations. Empty slice => green.
func checkRegistries(repoRoot string) ([]string, error)
```

Behavioural requirements, each one load-bearing:

- **`loadOrderedRegistry`** captures version-key order via a `json.Decoder` token stream over the
  raw `versions` object (do **not** round-trip through `map[string]…` and re-marshal: Go sorts map
  keys alphabetically and the file is chronological, P13 — that would produce a 59-entry reorder
  diff and bury the actual change).
- **`writeOrderedRegistry`** emits top-level keys in the order `schema_version, versions, active,
  notes`; each entry's keys in the order `file, hash, description, created, tags, notes, frozen`;
  2-space indent; **trailing newline**; `frozen` omitted when nil. Result on an unmigrated input
  must be byte-identical to the input — **assert this in a test before trusting the writer.**
- **`scanCorpus`** runs `git ls-files 'eval_results/baselines/*.json'` from `repoRoot` and decodes
  only `prompt_version` from each file. Files without the field contribute nothing (Q2). Measured
  1.87 s (P7).
- **`migrateRegistries`** reads **`prompts/versions.json` only**, builds one migrated
  `orderedRegistry`, then writes it to **both** paths. Rule per key `k` in `VersionKeys`:
  - `k == r.Active` → leave `Frozen` nil.
  - `corpus[k].Count > 0` → `{At: today, Reason: "banked", EvidenceCount: corpus[k].Count, EvidenceExample: corpus[k].Example}`.
  - else → `{At: today, Reason: "legacy", EvidenceCount: 0, EvidenceExample: ""}`.
  **Pre-flight refusals — write nothing and return an error if any of:** `r.Active` is `"latest"`
  or not a registry key (the migration must not guess which entry stays mutable); any entry about
  to be frozen has `!IsHexSHA256(hash)`; any entry about to be frozen has on-disk bytes ≠ hash
  (Q3, "you may not freeze a lie" — this is the check `aver` trips before Task 0).
  **Idempotent**: entries that already carry a `Frozen` marker are left untouched.
- **`checkRegistries`** violations, each string naming the offending version id:
  - L3(a) `corpus-evidenced but not frozen: <id> (<n> citations, e.g. <path>)`
  - L3(b) `frozen version <id>: file bytes do not match recorded hash`
  - L3(b) `frozen version <id>: recorded hash is not a 64-hex sha256 (unenforceable freeze)`
  - mirror-registry parity `cmd/ailang/prompts/versions.json: entry <id> differs from source`
  L3(c) merge-base immutability and L3(d) `.md` mirror bytes are **M2** — leave hooks, not code.

**Task 5 — `cmd/ailang/prompt_freeze.go` + subcommand wiring.**

`runPrompt` currently does `_ = promptFS.Parse(flag.Args()[1:])`. Go's `flag` **stops at the first
non-flag argument**, so `ailang prompt freeze --check` would leave `--check` unparsed. Intercept
*before* the FlagSet is built — insert at the top of `runPrompt()`:

```go
	sub := flag.Args()[1:]
	if len(sub) > 0 && sub[0] == "freeze" {
		runPromptFreeze(sub[1:])
		return
	}
```

```go
// runPromptFreeze implements `ailang prompt freeze [<version>] [--migrate] [--check] [--repo DIR]`.
// Exit codes: 0 green; 1 violations found (--check) ; 2 usage/IO error.
func runPromptFreeze(args []string)
```
`--repo` defaults to the repo root discovered by walking up for `go.mod`/`.git`; it exists so the
tests can point the command at a `t.TempDir()` fixture. Exactly one of `<version>`, `--migrate`,
`--check` may be given. `--check` prints every violation to stderr, one per line, then exits 1.
Add a `freeze` section to `printPromptHelp()`. **`cmd/ailang/help.go` is M2's** (CLI docs).

**Task 6 — run the migration.**
```
go build -o /tmp/i291/ailang ./cmd/ailang        # rc must be 0
/tmp/i291/ailang prompt freeze --migrate         # rc must be 0
/tmp/i291/ailang prompt freeze --check           # rc must be 0
```
Then re-run `--migrate` and confirm both files are byte-identical to the previous run (idempotence).

**Task 7 — tests** (§6).

### 5.3 M1 acceptance criteria (closes doc AC1, AC2, AC3, AC5, AC6-loader-half, AC9-migration-half)

Every command below was verified green-at-base in its baseline form (§2), or is new code.

| id | Command | Expected | Doc AC |
|---|---|---|---|
| M1.1 | `test -z "$(gofmt -l internal/eval_harness cmd/ailang internal/prompt)"` | rc=0 | — (base P1=0) |
| M1.2 | `go build ./cmd/ailang ./internal/eval_harness/... ./internal/prompt/...` | rc=0 | — (base P2=0) |
| M1.3 | `go vet ./internal/eval_harness/... ./internal/prompt/ ./cmd/ailang/` | rc=0 | — (base B4/P3=0) |
| M1.4 | `make check-file-sizes` | rc=0 | — (base B5=0) |
| M1.5 | `go test ./internal/eval_harness/...` | rc=0 — **this is the §3 constraint gate**: `TestLoadPrompt_HashMismatch` and `TestLoadPrompt_PlaceholderHash` must still pass | — (base B6=0) |
| M1.6 | `go test ./internal/prompt/` | rc=0 | — (base P4=0) |
| M1.7 | `go test ./cmd/ailang/ -run 'TestAILANGPromptLoading\|TestPromptDisambiguation' -v > /tmp/o; grep -c -- '--- PASS: ' /tmp/o` | `2` | V-H pin (base P5=2) |
| M1.8 **AC1** | `jq '[.versions[]\|select(.frozen.reason=="banked")]\|length' prompts/versions.json` / `…=="legacy"…` / `jq '[.versions\|to_entries[]\|select(.value\|has("frozen")\|not)]\|length'` / `jq -r '[.versions\|to_entries[]\|select(.value\|has("frozen")\|not)][0].key'` | `19` / `39` / `1` / `v0.16.6` | AC1 |
| M1.9 **AC9a** | the same four jq commands against `cmd/ailang/prompts/versions.json` | `19` / `39` / `1` / `v0.16.6` | AC9 |
| M1.10 **AC9a** | `diff <(jq -S .versions prompts/versions.json) <(jq -S .versions cmd/ailang/prompts/versions.json)` | rc=0, empty | AC9 |
| M1.11 **AC5-green** | `/tmp/i291/ailang prompt freeze --check` on the migrated tree | rc=0, no output | AC5 |
| M1.12 (§2.3) | `jq -r .versions.aver.hash prompts/versions.json` **equals** `shasum -a 256 prompts/aver.md \| cut -d' ' -f1` **equals** the same pair for the mirror | all `1f5c9654e39c…` | — |
| M1.13 | whole-registry audit (P11 command) over **both** registries | **0** mismatches (base: 1, `aver`) | — |
| M1.14 | `go test ./internal/eval_harness/ -run 'TestLoadPrompt_Frozen\|TestLoadPrompt_NeverBanked\|TestPromptRegistry_Frozen' -v > /tmp/o; grep -c -- '--- PASS: ' /tmp/o` | `5` | AC2, AC3, AC6a |
| M1.15 | `go test ./cmd/ailang/ -run 'TestPromptFreeze\|TestRealRegistry' -v > /tmp/o; grep -c -- '--- PASS: ' /tmp/o` | `7` | AC1, AC5, AC9a |
| M1.16 | `go test ./internal/prompt/ -run 'TestVersionMetadata_FrozenFieldRoundTrips' -v > /tmp/o; grep -c -- '--- PASS: ' /tmp/o` | `1` | schema |
| **M1.X** | `go build ./...` | **NOT A GATE — rc=1 at base (B3). Any AC using it is broken on arrival.** | — |

---

## 6. M1 test plan — each test named, each mapped to the mutation it kills

Naming is literal; the gates in §5.3 grep for these exact strings.

### `internal/eval_harness/prompt_frozen_test.go`

All fixtures use `t.TempDir()/prompts/{versions.json,<name>.md}` and `NewPromptLoader`, following
the shape already in `prompt_loader_test.go:110-160`.
**Fixture constant: `EvidenceCount = 4242`** — chosen so the asserted phrase cannot be produced by
any other code path (see the justification column).

| Test | Fixture | Assertions | Kills | Why the observable is downstream & unique |
|---|---|---|---|---|
| `TestLoadPrompt_FrozenTamperEmitsTeachingError` | one entry with `Frozen{At:"2026-08-27",Reason:"banked",EvidenceCount:4242,EvidenceExample:"eval_results/baselines/x/y.json"}`, correct 64-hex hash, then **append one byte** to the `.md` | `err != nil` **and** `err.Error()` contains ALL of: `is FROZEN`, the full phrase `cited by 4242 banked baseline files`, `create_prompt_version.sh` | **mutation 1** (freeze branch deleted) | Doc AC2's hollow-pass note: the pre-existing V-C check *also* errors on this input, so a bare-error assertion passes on unmodified code. The surviving message is `hash mismatch for … (file may have been modified)` — it can contain neither the word `FROZEN` nor the phrase `cited by 4242 banked baseline files` (the count is a fixture constant read from the marker, not derivable from the bytes; asserting the whole phrase rather than the bare token `4242` also rules out an accidental hex-preview substring match). |
| `TestLoadPrompt_NeverBankedTamperRegenSucceeds` | entry with **no** marker; tamper the `.md` **and** rewrite `hash` to the new sha256 | `err == nil` **and** returned content `==` the tampered bytes exactly | **mutation 2** (inverted predicate: everything frozen) | A freeze-everything loader still recomputes and compares — it would *pass* a naive "error is nil" check only if it also allowed the edit. The content-equality clause makes the observable the *served bytes*, downstream of the branch. Doc AC3 first half. |
| `TestLoadPrompt_NeverBankedStaleHashTeachesRegen` | entry with no marker; tamper the `.md`, leave `hash` stale | `err != nil` and message contains ALL of `hash mismatch` (§3 constraint), `not yet banked`, `in-place editing is allowed`, `shasum -a 256` | mutation 2 (second arm) | Doc AC3 second half. Unmodified code emits only `hash mismatch …` and fails the three teaching substrings. |
| `TestLoadPrompt_FrozenPlaceholderRefused` | entry with a `Frozen` marker **and** `hash: "PLACEHOLDER"`; `.md` bytes arbitrary | `err != nil` and message contains `is FROZEN` and `unenforceable hash` | **mutation 5** (V-C skip retained for frozen) | Doc AC6 hollow-pass: under V-C semantics the load *succeeds*. The observable is a refusal carrying freeze-specific wording, producible only by the Q6 branch. |
| `TestLoadPrompt_UnfrozenPlaceholderStillLoads` | entry with **no** marker and `hash: "PLACEHOLDER"` | `err == nil`, content matches file bytes | over-tightening of mutation 5's fix | **Mandatory control.** Without it, "refuse every PLACEHOLDER" also passes the previous row while breaking the documented never-banked workflow and `TestLoadPrompt_PlaceholderHash`. |
| `TestPromptRegistry_FrozenFieldRoundTrips` | marshal a `PromptRegistry` with a marker, unmarshal | `Frozen` survives with all four fields; an entry with `Frozen == nil` marshals **without** a `frozen` key | schema regression | Cheap; pins `omitempty` so unmarked entries don't gain `"frozen": null` and corrupt AC1's `has("frozen")\|not` count. |

### `cmd/ailang/prompt_freeze_test.go`

All tests build a fixture repo under `t.TempDir()` (a `prompts/` + `cmd/ailang/prompts/` pair, 8
entries) and inject the corpus by replacing `corpusScanner` — **no git repo, no network, no
mutation of the real tree.**

Fixture shape: 3 ids with corpus evidence (counts 210 / 38 / 1), 4 ids with none, 1 active id with
none → expected split **3 banked / 4 legacy / 1 mutable**.

| Test | Assertions | Kills | Why non-hollow |
|---|---|---|---|
| `TestPromptFreezeMigrate_SplitCounts` | after `migrateRegistries`, re-read the source registry: exactly 3 `reason:"banked"`, 4 `reason:"legacy"`, 1 with `Frozen == nil` and that one **is** the active id; and each banked entry's `EvidenceCount`/`EvidenceExample` equal the injected corpus values | **mutation 8** (legacy bucket omitted) | Doc AC1's hollow-pass logic, in fixture form: a corpus-only migration yields 3/0/5; a freeze-everything migration yields 0-mutable. Asserting the exact triple plus per-entry evidence rules out both, and the evidence fields can only come from the corpus scan. |
| `TestPromptFreezeMigrate_WritesBothRegistries` | (a) the **mirror** file independently reports the same 3/4/1 split; (b) `jq`-equivalent deep-equality of the two `versions` objects; (c) mirror byte-length > 0 | **mutation 10** (source-only write) | Clause (b) **alone is hollow** — a migration that writes *neither* file also leaves them equal. Clause (a) is the load-bearing one: it asserts markers are *present in the mirror*, the file agent mode actually reads (V-J). |
| `TestPromptFreezeMigrate_RefusesStaleHash` | fixture where one to-be-frozen entry's `.md` bytes ≠ its `hash` → `migrateRegistries` returns an error naming that id, **and both registry files are byte-unchanged** | Q3 "may not freeze a lie" | The byte-unchanged clause is what makes it non-hollow: an implementation that errors *after* a partial write passes an error-only assertion. This is the `aver` class (§2.3). |
| `TestPromptFreezeMigrate_Idempotent` | run `migrateRegistries` twice; both files byte-identical after run 2; counts unchanged | re-freeze / marker churn | A second run must not bump `at` or recompute evidence, or the migration is not reproducible. |
| `TestPromptFreezeCheck_GreenOnMigratedFixture` | `checkRegistries` on the just-migrated fixture returns **zero** violations | false-positive `--check` | Doc AC5 green half. Paired with the next row so "always green" and "always red" are both excluded. |
| `TestPromptFreezeCheck_RedOnMissingMarker` | delete the `frozen` marker from the 210-citation id (leaving every hash self-consistent) → violations contains an entry naming that id **and** the substring `corpus-evidenced but not frozen`; and assert the returned slice has **length 1** | **mutation 3** (`--check` validates hashes only) | Doc AC5 hollow-pass: the input's hashes are all self-consistent by construction, so L3(b) stays green; only the corpus derivation can fire. The length-1 clause stops a "flag everything" mutation from passing. |
| `TestRealRegistry_PostMigrationSplitCounts` | reads the **real** `../../prompts/versions.json` and `prompts/versions.json` (mirror): 19 banked / 39 legacy / 1 mutable, mutable id `v0.16.6`, both files' `versions` deep-equal, and `.versions.aver.hash` equals `sha256(prompts/aver.md)` | migration applied to the wrong tree; §2.3 regression | This is doc AC1 + AC9's jq half + M1.12 expressed as a `--- PASS` line, so the gate can use the `grep -c` form the brief mandates. Read-only; it never writes the real tree. |

### `internal/prompt/loader_test.go`
`TestVersionMetadata_FrozenFieldRoundTrips` — the marker survives unmarshal from a manifest
containing `"frozen": {...}`, and an entry without one yields `Frozen == nil`. Pins the schema half
of M3's dependency; **must not** assert any enforcement (M3 owns L2).

---

## 7. M2 — CI gate (follow-on iteration; independently landable on M1)

**Files** — create `cmd/ailang/prompt_freeze_check_git.go` (L3(c) merge-base immutability, L3(d)
`.md` mirror bytes) and `cmd/ailang/prompt_freeze_check_git_test.go`; modify
`make/code-health.mk` (new `check-prompt-freeze` target), `make/ci.mk` (append
`check-prompt-freeze` to the `ci` prerequisite list), `cmd/ailang/prompt_freeze_core.go` (one
dispatch line in `checkRegistries`), `cmd/ailang/help.go`, `cmd/ailang/prompt.go` (help text),
`.claude/skills/prompt-manager/scripts/create_prompt_version.sh` (one sentence in "Next steps"),
`changelogs/v0.18-current.md`, `docs/docs/guides/evaluation.md`.

**Tasks**: (1) `git merge-base HEAD origin/dev` + `git show <base>:prompts/versions.json` to
diff frozen entries' `file`/`hash` and the frozen `.md` blobs — a frozen entry present at the
merge-base whose `hash`, `file` or `.md` bytes changed is a violation naming `immutability`;
(2) L3(d) `.md` mirror byte comparison for frozen ids; (3) Makefile wiring; (4) docs.

**Closes**: AC4, AC5 (re-pin), AC6 second half, **AC8** (see §4 row 1 — the doc's AC header tags it
M3, its milestone text and L3 table say M2), AC9 `--check` half.
**Kills**: mutations 4, 9, and the `--check` arm of 3 and 5.
**Tests** (each needs a scratch git repo built in the test, per the doc's Testing Strategy):
`TestFreezeCheck_MergeBaseEditPlusRegenIsRed` (AC4 — the input is self-consistent at HEAD so
L3(a)(b) both stay green; only the merge-base diff can fire),
`TestFreezeCheck_FrozenPlaceholderIsRed` (AC6b), `TestFreezeCheck_MirrorMdDivergenceIsRed` (AC8 —
assert the violation names the `cmd/ailang/prompts/…` path, not just the id),
`TestFreezeCheck_MirrorRegistryDivergenceIsRed` (AC9 — the `.md` files are identical by
construction, so AC8's instrument stays green; only the registry-entry comparison fires),
`TestFreezeCheck_UnmodifiedTreeIsGreen` (the mandatory always-red control).
**Gates**: M1.1-M1.7 plus `make check-prompt-freeze` rc=0 and `bash -n
.claude/skills/prompt-manager/scripts/create_prompt_version.sh` rc=0.
**Note**: `make ci` as a whole is *not* proposed as a gate — it was not baselined and includes
long-running targets. Gate the new target only.

## 8. M3 / M4 — outlines (follow-on iterations)

**M3 — close the agent-mode hole.** Modify `internal/prompt/loader.go` (verify sha256 of the bytes
actually read against `hash` **for frozen versions only**, on **both** the embedded and the disk
path — `LoadPrompt` `:62-77`), reusing M1's error text; remove the silent
`(DefaultPrompt(), "default", nil)` fallbacks in `langreg/ailang.go:32-38`,
`langreg/python.go:31-39` **and `langreg/aver.go:33-37`** (the third instance, which the doc's N-4
does not list — see §2.3). Closes AC7; kills mutations 6 and 7. Test
`TestLoadSyntaxRef_FrozenMismatchIsLoudError` must deliver the tampered bytes **through the
embedded FS** (mutation 7) and assert both `err != nil` **and** `versionUsed != "default"`
(mutation 6 — the doc's AC7 hollow-pass note: today's code returns
`(DefaultPrompt(), "default", nil)` on exactly this input). Highest-risk milestone: it converts a
silent degradation into a hard abort on the path that runs most evals. **Prerequisite: `aver`'s
hash must be correct (§2.3) or M3 breaks every aver eval on landing.**

**M4 — bank-time byte evidence.** Add `PromptSHA256 string \`json:"prompt_sha256,omitempty"\`` at
`cmd/ailang/eval_suite_manifest.go:77` and `:133`; populate from sha256 of the **fully evaluated
prompt string immediately prior to model dispatch** at `cmd/ailang/eval_benchmark.go` (~`:165-180`,
after the task description is appended) and `cmd/ailang/eval_benchmark_agent.go:315`; add
`cmd/ailang/prompt_freeze_check_rows.go` for the merge-base-scoped row cross-check. Closes AC10,
AC10(b), AC11; kills mutations 11 and 12. AC11's fixture must be **unfrozen + `PLACEHOLDER`** —
the one state where bytes are served with no hash agreement — so that an implementation copying the
registry `hash` writes the literal `"PLACEHOLDER"` and fails the 64-hex assertion. Safe per doc
N-10 (no strict JSON decoders anywhere).

## 9. File-overlap / bisectability audit (as requested)

| File | M1 | M2 | M3 | M4 | Verdict |
|---|---|---|---|---|---|
| `internal/prompt/loader.go` | **type block only** (+12) | — | **`LoadPrompt` body** | — | **OVERLAP, mitigated.** The two hunks are disjoint (lines ~20-36 vs ~40-78). M1 must add the field and *nothing else* — no enforcement, no helper — or M3's diff stops being self-contained. |
| `cmd/ailang/prompt_freeze_core.go` | creates | +1 dispatch line | — | +1 dispatch line | **OVERLAP, mitigated by splitting**: M2 and M4 put their check logic in **new** files (`prompt_freeze_check_git.go`, `prompt_freeze_check_rows.go`) and touch the core only to call them. Keeps each commit's diff readable and each milestone revertible. |
| `internal/eval_harness/prompt_loader.go` | rewrites the verify block | — | — | possibly +1 exported hash helper | Minor. Prefer M4 reuse `ComputePromptHash`/add its own helper rather than editing this file. |
| `cmd/ailang/prompt.go` | subcommand intercept + `printPromptHelp` | help text | — | — | Minor, adjacent hunks. |
| `prompts/versions.json`, `cmd/ailang/prompts/versions.json` | **M1 exclusively** | fixtures only | fixtures only | fixtures only | Clean. The migration is one commit, one milestone. |
| `cmd/ailang/help.go` | **untouched** (deliberate) | M2 | — | — | Clean. |
| `make/ci.mk`, `make/code-health.mk` | — | M2 | — | — | Clean. |
| `langreg/{ailang,python,aver}.go` | — | — | M3 | — | Clean. |
| `cmd/ailang/eval_suite_manifest.go`, `eval_benchmark.go`, `eval_benchmark_agent.go` | — | — | — | M4 | Clean. |

**No M1 file is also written by M2, M3 or M4 except `internal/prompt/loader.go` and
`cmd/ailang/prompt_freeze_core.go`, both handled above.**

## 10. Estimate

Doc says 3.5-4 days for four milestones. This plan re-allocates: **M1 1.25 d** (the doc's ≤1 d does
not budget the `aver` blocker, the order-preserving registry writer, or the two existing-test
constraints in §3), M2 1 d, M3 1.25 d (highest risk), M4 1 d — total 4.5 d, of which **only M1
(~800 LOC: ~420 impl / ~380 test) is executed this iteration**.
```

---

---

## Appendix A — planner's blocking finding (adjudicated by the controller)

**`aver` has a stale registry hash at HEAD, in BOTH registries.** This is a live D-41-class incident already in the tree, and it makes M1-as-written produce a red `--check` on unmodified `dev`.

```
$ jq -r '.versions|to_entries[]|"\(.key) \(.value.file)"' prompts/versions.json \
  | while read k f; do exp=$(jq -r --arg k "$k" '.versions[$k].hash' prompts/versions.json); \
      act=$(shasum -a 256 "$f"|cut -d' ' -f1); [ "$exp" = "$act" ] || echo "HASHMISMATCH $k exp=$exp act=$act"; done
HASHMISMATCH aver exp=190bf4244a00bb0605d41ac01d4de415a6131a3a961aa6bca1689b23bc4d521d act=1f5c9654e39c411bcbedc35446ffa96c1d6dc8d7b9dd0fbec01b95ecf35f0157
```
(58 of 59 clean = the positive control on the same instrument/scope. Mirror scan over `cmd/ailang/prompts/`: same single hit, `MIRROR-HASHMISMATCH aver`; the two `.md` files agree byte-for-byte at `1f5c9654…`, only the recorded hash is stale. Introduced by `0fcae6055 fix(eval): adopt upstream Aver prompt…`.)

Why it blocks: Q4 puts `aver` in the 39-entry `legacy` bucket; Q3 says `freeze` "refuses if the version's current file bytes do not match its recorded hash (you may not freeze a lie)"; L3(b) makes a frozen entry with bytes ≠ hash a violation. So a literal M1 either **refuses to migrate** or **writes a frozen lie**, and AC5's "on unmodified post-migration `dev`, `--check` exits 0" is unreachable. It also pre-breaks M3: once `internal/prompt.LoadPrompt` enforces frozen hashes, every `aver` eval hard-fails.

**Recommended resolution (R-A), used as the default in the plan below:** M1 Task 0 repairs `aver`'s hash to the disk sha256 in both registries *before* migrating. This is exactly the never-banked workflow the ruling permits — `aver` has **zero** corpus citations (`git ls-files 'eval_results/baselines/*.json' | xargs jq -r '…' | sort | uniq -c` → no `aver` row; 17,343 files, agrees with N-2/V-F exactly), so under D-41(c) it is NEVER-BANKED and in-place edit + hash regeneration is legal. Preserves AC1's 19/39/1 exactly. Rejected alternative R-B (leave `aver` unfrozen) yields 19/38/2 and contradicts AC1.

---
