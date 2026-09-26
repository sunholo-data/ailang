# M-PROMPT-FREEZE-MIRROR-ALL-VERSIONS: `prompt freeze --check` must refuse divergence, deletion and unenforceable hashes for ALL registry entries, not only frozen ones

**Status**: Planned
**Target**: v0.34.x
**Priority**: P1 (data-integrity — the one entry the gate cannot see is the active prompt every agent reads)
**Estimated**: 1 day (single milestone)
**Dependencies**: parent mechanism landed — `design_docs/planned/m-prompt-version-freeze-on-first-bank.md`, shipped via PR #937 (squash `445ccb550`)
**Author**: design-doc-creator role, mission iteration 297 (2026-08-28), worktree at origin/dev = `0917693792d07c75ae0c4d233d2cc7e592fa9272`, darwin/arm64
**Scope ruling**: the mission's queue-head decision was answered **EXTEND** — widen the file-level arms to all entries. This doc designs within that ruling; it does not re-open it.
**Revision**: r2 — round-1 quorum verdict was **blocked** (3/3 reviewers present, two blocking
objections, neither disputing the design direction). Both accepted: arm independence (no
violation masking) and the checked-count observable made a product of the per-entry work. See
the [Quorum revision log](#quorum-revision-log-round-1--r2).

---

## Problem Statement

`make check-prompt-freeze` (CI gate, `make/code-health.mk:184`, member of `ci:` at `make/ci.mk:11`
and an explicit step in `.github/workflows/ci.yml:205`) runs `ailang prompt freeze --check`. Its
three file-level arms are ALL gated on `Frozen != nil`:

1. `cmd/ailang/prompt_freeze_core.go` `checkRegistries`: hash-is-enforceable-sha256 and
   file-bytes-vs-recorded-hash live inside `if e.Frozen != nil {` (line 357).
2. `cmd/ailang/prompt_freeze_check_git.go:28` `if baseEntry.Frozen == nil { continue }` gates
   merge-base immutability (correct — stays as-is, see Non-Goals).
3. `cmd/ailang/prompt_freeze_check_git.go:51` `if entry.Frozen == nil { continue }` gates mirror
   byte-agreement between `prompts/<x>.md` and `cmd/ailang/prompts/<x>.md`.

The only all-entries check is the registry-vs-registry JSON entry comparison in `checkRegistries`
(lines 366-370) — and it never opens a `.md` file, so it is structurally incapable of seeing file
divergence, deletion, or a stale hash.

The registry holds **59 versions, 58 frozen, exactly one mutable: `v0.16.6` — which is also the
`active` entry** (V1). So the single entry the gate cannot see is the prompt every agent actually
reads. Measured consequences (V4, V5): a diverged or deleted `prompts/v0.16.6.md`, a diverged
mirror copy, or a `PLACEHOLDER` hash on the mutable entry all pass the gate at rc=0. Two
amplifiers:

- `cmd/ailang/main.go:21` is `//go:embed all:prompts`, a **directory** glob (V8) — a deleted
  prompt file compiles clean; nothing at build time notices.
- The enumerator iterates **source registry keys only**: an entry present only in the mirror
  registry `cmd/ailang/prompts/versions.json` is invisible to every arm — measured rc=0 (V4 cell 8).

Two adjacent defects surfaced by the same measurement: a **missing frozen source file** aborts the
whole scan as a hard rc=2 error instead of a named rc=1 violation (V5 cell 11), hiding any other
violations; and a mirror **deletion** is reported with the misleading text "mirror bytes differ"
(V4 cell 4).

---

## Goals

1. Every registry entry — frozen AND mutable — gets, at `--check` time:
   (a) mirror byte-agreement (`prompts/<file>` vs `cmd/ailang/prompts/<file>`),
   (b) recorded-hash vs actual source file bytes,
   (c) file existence in both trees as a **named violation** (not a crash, not a silent pass),
   (d) hash-is-enforceable-sha256.
   The arms are **independent** (r2): for a given entry EVERY applicable violation is emitted —
   one defect never masks another. The only permitted dependencies are the two byte-comparisons'
   semantic prerequisites (Solution Design §3).
2. The enumerator becomes auditable: mirror-only registry entries are a named violation, and
   `--check` reports the **checked-entry count** so an addition the enumerator misses is
   detectable by count, not only by verdict.
3. All existing frozen-arm behavior and violation strings stay byte-identical except the one
   deliberate rename (mirror deletion gets an honest "missing" message).

## Non-Goals

- **Merge-base IMMUTABILITY stays frozen-only.** The `if baseEntry.Frozen == nil { continue }` in
  `checkGitPromptFreezeInvariants` is UNCHANGED. Mutable means editable — an edit+hash-regen in
  the same commit is the legitimate workflow (`create_prompt_version.sh` step 3, V10) and must
  stay green. The obvious "just delete the Frozen guards" refactor would break exactly this;
  this doc widens three arms and deliberately not the fourth.
- No change to either loader (`internal/eval_harness/prompt_loader.go` keeps its runtime
  `PLACEHOLDER` hatch for mutable versions; `internal/prompt/loader.go` still performs no hash
  verification — that is the parent doc's still-open M3, V9).
- No registration of the 3 unregistered `.md` files under `prompts/` (V11) — enumeration stays
  registry-driven; see Open Questions.
- No change to `freezeVersion` (single-version freeze already validates its target).

---

## Verification Log

All rows measured first-party by this role on **darwin/arm64**, in the clean worktree
`/Users/voightkampff/dev/sunholo-data/.wt-iter297` at origin/dev =
`0917693792d07c75ae0c4d233d2cc7e592fa9272`. Probe binary: `go build -o /tmp/i297bin/ailang
./cmd/ailang` from this worktree (`--version` reports `AILANG dev` — no ldflags stamp; the binary
is trusted **behaviorally** via the positive-control cells in V4, which reproduce the known frozen
refusals). Synthetic probe repo for V4/V5: `/tmp/i297probe/repo` containing copies of `prompts/`,
`cmd/ailang/prompts/` and an empty `eval_results/baselines/`; `git init` + one commit +
`git update-ref refs/remotes/origin/dev HEAD`; a tar snapshot (`--exclude .git`) restored before
each cell; invoked as `/tmp/i297bin/ailang prompt freeze --check --repo /tmp/i297probe/repo`.

**Refuted hypothesis (kept per the hard gate — measure, don't reason):** the iteration controller
initially inferred from reading `checkRegistries` alone that the mirror `.md` tree was never
hashed for ANY entry. FALSE — the frozen-only mirror arm exists in the *other* file
(`prompt_freeze_check_git.go:49-72`, violation text `"frozen version %s: mirror bytes differ at
%s"` at line 70). I re-verified that arm by reading both files and by V4 cells 3-4 firing red.

| # | Claim | Command | Observed output |
|---|-------|---------|-----------------|
| V1 | Registry shape: 59 versions, 58 frozen, 1 mutable = `v0.16.6` = `active`; mirror registry also 59; 62 `.md` files in each tree | `jq -r '.versions\|length' prompts/versions.json` ; `jq -r '[.versions\|to_entries[]\|select(.value\|has("frozen"))]\|length' …` ; `jq -r '[.versions\|to_entries[]\|select(.value\|has("frozen")\|not)\|.key]\|join(",")' …` ; `jq -r .active …` ; `ls prompts/*.md \| wc -l` ; `ls cmd/ailang/prompts/*.md \| wc -l` ; `jq -r '.versions\|length' cmd/ailang/prompts/versions.json` | `59` / `58` / `v0.16.6` / `v0.16.6` / `62` / `62` / `59` |
| V2 | Gate wiring: make target + `ci:` membership + explicit CI step | `grep -n "check-prompt-freeze" make/code-health.mk make/ci.mk .github/workflows/ci.yml` | `make/code-health.mk:5` + `:184` (target = `go run ./cmd/ailang prompt freeze --check`), `make/ci.mk:11`, `.github/workflows/ci.yml:205: run: make check-prompt-freeze`; grep rc=0 |
| V3 | The three file-level arms are gated on `Frozen != nil`; the registry-vs-registry compare is the only all-entries check and opens no `.md` | code read of `cmd/ailang/prompt_freeze_core.go:355-371` and `prompt_freeze_check_git.go:27-72` (full files read this session) | `if e.Frozen != nil {` guards hash-enforceable + bytes-vs-hash; `if baseEntry.Frozen == nil { continue }` (line 28); `if entry.Frozen == nil { continue }` (line 51); JSON compare at lines 366-370 uses `json.Marshal` only |
| V4 | **The live matrix.** Cells, each restored from snapshot first: pristine; frozen `v0.3.21` source `.md` diverged; frozen mirror diverged; frozen mirror deleted; mutable `v0.16.6` source diverged; mutable source deleted; mutable mirror diverged; mirror-registry EXTRA entry (`v9.9.9-extra` added to mirror `versions.json` only); restored | probe-repo protocol above; divergence = `printf 'x' >> <file>`; extra entry via `jq '.versions["v9.9.9-extra"] = {"file":…,"hash":"PLACEHOLDER"}'` on the mirror registry only | `1 pristine rc=0` · `2 frozen-src-diverged rc=1` (`frozen version v0.3.21: file bytes do not match recorded hash` + `…immutability violation in prompts/v0.3.21.md bytes` + `…mirror bytes differ at cmd/ailang/prompts/v0.3.21.md`) · `3 frozen-mirror-diverged rc=1` · `4 frozen-mirror-deleted rc=1` (`frozen version v0.3.21: mirror bytes differ at cmd/ailang/prompts/v0.3.21.md` — the misleading text) · `5 mutable-src-diverged rc=0` **HOLE** · `6 mutable-src-deleted rc=0` **HOLE** · `7 mutable-mirror-diverged rc=0` **HOLE** · `8 mirror-registry-EXTRA-entry rc=0` **HOLE (new this iteration)** · `9 restored rc=0`. The four rc=1 cells in the same harness are the positive controls that make the four rc=0 cells measurements, not claims |
| V5 | Two further cells: mutable entry with `hash="PLACEHOLDER"` in BOTH registries passes; a deleted **frozen source** file is a hard rc=2 abort, not a named violation | same protocol: `jq '.versions["v0.16.6"].hash = "PLACEHOLDER"'` applied to both registry files → run → restore; `rm prompts/v0.3.21.md` → run → restore → run | `CELL10 rc=0` **HOLE (arm d)** · `CELL11 rc=2`, stderr `open /tmp/i297probe/repo/prompts/v0.3.21.md: no such file or directory` · `CELL12 restored rc=0` (restore was byte-faithful) |
| V6 | The mutable entry's recorded hash is currently FRESH in both trees and both registries agree — so widening arms (a)(b)(d) is green at base, not red | `jq -r '.versions["v0.16.6"].hash'` on both registries; `shasum -a 256 prompts/v0.16.6.md cmd/ailang/prompts/v0.16.6.md`; string compare | `reg=2a868106214d3a4fc94dcb5431560cebacbec05c1e1835a168b8dcb15f804b67`, `mirror_reg_same=yes`, `src_match=yes`, `mirror_match=yes` |
| V7 | `make check-prompt-freeze` is rc=0 at base (pristine worktree, real repo) | `make check-prompt-freeze > /tmp/i297probe/mk.out 2>&1; printf 'rc=%s' "$?"` | `rc=0` |
| V8 | The embed is a directory glob — a missing prompt compiles clean | `grep -n "go:embed" cmd/ailang/main.go` | `21://go:embed all:prompts` |
| V9 | The agent-mode loader parses `Frozen` but performs **no** hash verification (parent M3 still open); and NO test references either loader error constructor by name | `grep -n "sha256\|Mismatch\|verify\|Verify" internal/prompt/loader.go` → empty. **Control, same instrument, same package**: the same grep over `internal/prompt/*.go` hits `fresh.go` at lines 17/162/184 (`crypto/sha256`, `sha256.Sum256`), so the instrument sees sha256 where it exists. `grep -rn "FrozenHashMismatchError\|MutableHashMismatchError" internal/ cmd/ --include='*_test.go'` → empty. **Control, same pattern, non-test files**: hits `internal/eval_harness/prompt_loader.go:89` and `prompt_frozen.go:32` | loader.go: no hits; fresh.go: sha256 hits; test grep: no hits; non-test grep: 2 hits |
| V10 | `create_prompt_version.sh` lives at `.claude/skills/prompt-manager/scripts/` (NOT `scripts/` — the queue text's path is wrong), computes a real 64-hex hash at creation (line ~91-93), and its "Next steps" already instruct "Update hash" (line 129) and "Run make check-prompt-freeze before committing" (line 134) — so the widened gate matches the documented workflow | `ls scripts/create_prompt_version.sh` → `No such file or directory` with **control in the same call** `ls .claude/skills/prompt-manager/scripts/create_prompt_version.sh` → exists; `grep -n "hash\|freeze\|frozen" .claude/skills/prompt-manager/scripts/create_prompt_version.sh` | as stated; lines 91-93, 129, 134 |
| V11 | Exactly 3 `.md` files under `prompts/` are referenced by no registry entry (docs, not prompt versions) | `comm -23 <(ls prompts/*.md \| sort) <(jq -r '.versions[].file' prompts/versions.json \| sort)` | `prompts/CHANGELOG_v0.3.15.md`, `prompts/feedback_gate_classifier.md`, `prompts/testing_guide_ai.md` |
| V12 | `cmd/ailang/eval_suite_manifest.go` only *writes* `prompt_version` strings into manifests; it reads neither registry — no conflict with this change | `grep -n "prompt_version\|PromptVersion\|versions.json" cmd/ailang/eval_suite_manifest.go` | lines 77, 133, 205, 222 — all struct fields/assignments; zero `versions.json` hits |
| V13 | `migrateRegistries`' pre-flight validation loop skips the active entry AND frozen entries (`if id == r.Active \|\| e.Frozen != nil { continue }`), so today `--migrate` would succeed on a tree where the widened `--check` is red | code read, `cmd/ailang/prompt_freeze_core.go:251-266` | as stated |
| V14 | The migrate-test fixture writes FRESH hashes for every entry including `active`, so widening the migrate validation loop (Solution Design §4) does not break `TestPromptFreezeMigrate_*` | read of `freezeFixture` in `cmd/ailang/prompt_freeze_test.go:14-44` | fixture computes `sha256(content)` per entry incl. `"active"` and writes matching files to both trees |
| V15 | The existing `--check` test harness builds a synthetic repo, drives `checkRegistries` in-process AND the rc via a helper subprocess, and has a `frozen bool` parameter on its registry writer — mutable fixtures are constructible without new scaffolding. Existing tests: `TestFreezeCheck_MergeBaseEditPlusRegenIsRed`, `_FrozenPlaceholderIsRed`, `_MirrorMdDivergenceIsRed` (divergence, not deletion — the deletion rename in §2 breaks no existing assertion), `_MirrorRegistryDivergenceIsRed`, `_UnmodifiedTreeIsGreen` | read of `cmd/ailang/prompt_freeze_check_git_test.go` (166 lines, full) | as stated; `writeFreezeCheckRegistries(t, root, sourceHash, mirrorHash, frozen bool)`; helper = `TestFreezeCheckHelperProcess` + `AILANG_FREEZE_CHECK_HELPER=1` |
| V16 | The standard-mode loader ALREADY hash-verifies mutable versions at load time (`MutableHashMismatchError`) unless `hash == "PLACEHOLDER"` — the widened gate makes CI agree with the loader, not stricter than it (except for PLACEHOLDER, adjudicated in §5) | read of `internal/eval_harness/prompt_loader.go:76-93` | `if version.Frozen != nil {…}` / else `if version.Hash != "PLACEHOLDER" { … MutableHashMismatchError … }` |
| V17 | **Compound hole (r2)**: a mutable entry with an INVALID hash AND a MISSING source file passes at base — neither S2 nor S5 fires | probe protocol: `jq '.versions["v0.16.6"].hash = "PLACEHOLDER"'` applied to BOTH registries + `rm prompts/v0.16.6.md` → `--check` → restore | `CELL13 rc=0`, no output beyond the Observatory log line |
| V18 | **Compound abort (r2)**: a frozen entry with BOTH source and mirror `.md` deleted is a hard rc=2 abort naming only the FIRST missing file — the mirror deletion is invisible | `rm prompts/v0.3.21.md cmd/ailang/prompts/v0.3.21.md` → `--check` → restore → re-run restored control | `CELL14 rc=2`, stderr exactly one line: `open /tmp/i297probe/repo/prompts/v0.3.21.md: no such file or directory` (no mirror violation) · `CELL15 restored rc=0` |
| V20 | **`checkRegistries` call graph (r3)**: 4 callers, all in `package main` under `cmd/ailang` — 1 production + 3 test; the r2 "Files to modify" list named only ONE of the two test files, so the objection found a real gap | `grep -rn "checkRegistries" --include='*.go' .` (rc=0), same-scope control `grep -rc "promptRegistryPair" --include='*.go' .` | 6 lines: def `prompt_freeze_core.go:335`; comment `prompt_freeze_check_git.go:56`; callers `prompt_freeze.go:69`, `prompt_freeze_check_git_test.go:134`, `prompt_freeze_test.go:125`, `prompt_freeze_test.go:138`. Control fires: `prompt_freeze_core.go:4`, `prompt_freeze_test.go:1` |
| V21 | **`prompt freeze --check` stdout consumers (r3)**: exactly one, and it ignores stdout — so adding an unconditional count line breaks nothing | `grep -rn "freeze --check\|freeze\", \"--check\|check-prompt-freeze" --include='*.sh' --include='*.yml' --include='*.yaml' --include='*.mk' --include='Makefile' --include='*.go' .` (design_docs excluded) | `make/code-health.mk:185` is the sole invocation (`@go run … --check`, no pipe, no parse); reached from `.github/workflows/ci.yml:205` and `make/ci.mk:11`. Non-consumers: `help.go:285`, `prompt.go:154` (help strings) and `.claude`/`.agents` `create_prompt_version.sh:119,134` (a comment and an echo). **No consumer asserts on stdout or requires it empty** |
| V22 | **`freezeVersion` pre-write validation (r3)**: the Non-Goals/Conflict-Surface claim is TRUE and is now measured rather than asserted | code read `cmd/ailang/prompt_freeze_core.go:298-333`; write-surface probe `sed -n '298,334p' … \| grep -c "WriteFile\|\.md"` with same-file control `grep -c writeOrderedRegistry …` | `:316` `!IsHexSHA256(e.Hash)` → error · `:319-322` `fileHash(...)` error propagated (covers a missing source file) · `:323` `actual != e.Hash` → error — **all three precede** the first write at `:329`. Write-surface probe = **0** (writes no `.md`), control = **5**. Writes both registries from the same `r` (`:329` source, `:332` mirror) |
| V19 | `source.Versions[id] == nil` compiles: the registry map is a POINTER map, and the identical nil-deref pattern already ships in the same file (first-party re-derivation of the controller's measurement refuting gemini-3-1-pro's pass-note; no design change) | `sed -n '29p;351p' cmd/ailang/prompt_freeze_core.go` | `Versions      map[string]*registryEntry` · `if e := source.Versions[id]; e != nil && e.Frozen == nil {` |

---

## Solution Design

One unified change: the file-level arms iterate **all** entries and carry a `state` label
(`"frozen"` / `"mutable"`) in their violation strings; frozen-path strings stay byte-identical.
The tests match on these exact strings, so they are normative:

| # | Violation string (format) | Status |
|---|---------------------------|--------|
| S1 | `frozen version %s: recorded hash is not a 64-hex sha256 (unenforceable freeze)` | unchanged |
| S2 | `mutable version %s: recorded hash is not a 64-hex sha256 (unenforceable)` | **new** |
| S3 | `frozen version %s: file bytes do not match recorded hash` | unchanged |
| S4 | `mutable version %s: file bytes do not match recorded hash` | **new** |
| S5 | `frozen version %s: prompt file missing at %s` / `mutable version %s: prompt file missing at %s` | **new** (replaces the rc=2 abort, V5 cell 11) |
| S6 | `frozen version %s: mirror bytes differ at %s` | unchanged (divergence only) |
| S7 | `mutable version %s: mirror bytes differ at %s` | **new** |
| S8 | `frozen version %s: mirror file missing at %s` / `mutable version %s: mirror file missing at %s` | **new** (deletion no longer reported as "differ"; no existing test asserts the deletion case, V15) |
| S9 | `cmd/ailang/prompts/versions.json: entry %s differs from source` | unchanged (already all-entries; also fires when a source entry is absent from the mirror) |
| S10 | `cmd/ailang/prompts/versions.json: entry %s missing from source registry` | **new** (mirror-only entries, V4 cell 8) |
| S11 | `frozen version %s: immutability violation …` (both variants) | unchanged — frozen-only by design |

### 1. `checkRegistries` (`cmd/ailang/prompt_freeze_core.go`) — widen the per-entry loop

Replace the `if e.Frozen != nil { … }` block inside the `source.VersionKeys` loop with three
**independent** arms (r2 — the round-1 draft chained them with `else if`, so one violation masked
the others; V17 measures the compound state that masking cannot see):

```go
state := "mutable"
if e.Frozen != nil {
    state = "frozen"
}
// Arm 1 — hash enforceability. Unconditional.
hashOK := eval_harness.IsHexSHA256(e.Hash)
if !hashOK {
    suffix := "(unenforceable)"
    if e.Frozen != nil {
        suffix = "(unenforceable freeze)" // preserves S1 byte-for-byte
    }
    v = append(v, fmt.Sprintf("%s version %s: recorded hash is not a 64-hex sha256 %s", state, id, suffix))
}
// Arm 2 — source existence. Unconditional; independent of arm 1.
sourceBytes, srcErr := os.ReadFile(filepath.Join(repoRoot, e.File))
sourceExists := !errors.Is(srcErr, fs.ErrNotExist)
if !sourceExists {
    v = append(v, fmt.Sprintf("%s version %s: prompt file missing at %s", state, id, e.File)) // S5
} else if srcErr != nil {
    return nil, 0, srcErr // environmental read errors (perms, I/O) stay hard errors
}
// Arm 3 — source bytes vs recorded hash. The ONLY conditional source arm, gated on
// hashOK (without an enforceable hash there is no referent to compare against) and
// sourceExists (without the file there are no bytes). Both dependencies are semantic
// necessities, not sequencing accidents — nothing else may gate this arm.
if hashOK && sourceExists {
    sum := sha256.Sum256(sourceBytes)
    if hex.EncodeToString(sum[:]) != e.Hash {
        v = append(v, fmt.Sprintf("%s version %s: file bytes do not match recorded hash", state, id))
    }
}
checked++ // LAST statement of the body: this entry's applicable arms have all run
```

(`fileHash` stays for the `--migrate`/`freezeVersion` paths; here the file is read once and
hashed inline — `crypto/sha256` and `encoding/hex` are already imported in this file.)

**Missing-file adjudication (scope item c):** a missing file becomes a **violation** (rc=1), not
an `error` (rc=2), for `fs.ErrNotExist` specifically. Rationale: (i) a deleted prompt is exactly
the corruption class this gate exists to name; (ii) the current `fileHash` error propagation
aborts the scan at the first missing file and hides every other violation (V5 cell 11 shows one
bare `open … no such file` line and nothing else); (iii) rc=1 vs rc=2 is the gate's documented
contract (`prompt_freeze.go`: "0 green; 1 violations; 2 usage/IO error") — absence is a
repo-state fact, not an IO accident. Environmental read errors (permissions, etc.) remain rc=2.

**Enumerator audit (r2 — count is a PRODUCT of the per-entry work, never a `len()` computed
beside it):** `checked` starts at 0 and is incremented exactly once per visited entry, as the
last statement of the loop body that performs that entry's arms (see arm snippet above). The
mirror-only sweep after the source loop does the same:

```go
for _, id := range mirror.VersionKeys {
    if source.Versions[id] == nil { // pointer map; identical deref already ships (V19)
        v = append(v, fmt.Sprintf("cmd/ailang/prompts/versions.json: entry %s missing from source registry", id)) // S10
        checked++
    }
}
```

Keys present in both registries are counted once (the sweep counts only source-absent keys), so
the arithmetic is order-independent and unambiguous: pristine real registry → `checked 59`; one
mirror-only entry added → `checked 60`; the two-entry test fixture (1 source + 1 mirror-only) →
`checked 2`. `checkRegistries` gains the count in its signature — `func checkRegistries(repoRoot
string) ([]string, int, error)` — and `runPromptFreeze --check` prints
`checked %d registry entries\n` to stdout on every completed run, green or red.

**Honest scope of the count observable (r2):** because the increment lives inside the visiting
body, an entry no loop visits contributes neither its violations nor its count — so the count
detects **enumeration blindness** (the M-ADD case). It structurally CANNOT detect a neutered arm
inside a visited body: the entry is still visited and still counted. Those defects are killed by
the per-string mutants M1–M9. The two observables are complementary, and both are asserted
(AC6, M-ADD vs AC2–AC5/AC9/AC10).

### 2. Mirror arm (`cmd/ailang/prompt_freeze_check_git.go`, second loop) — widen and rename deletion

Replace `if entry.Frozen == nil { continue }` with the state label; keep the frozen-only
`sameFrozenRegistryEntry` dedupe recheck exactly as-is (it exists for independent callers and is
already covered for all entries by S9 in `checkRegistries`). Body becomes (r2 — the round-1 draft
`continue`d on a missing source BEFORE checking mirror existence, exactly the masking V18
measures; mirror existence is now unconditional):

```go
sourceBytes, sourceErr := os.ReadFile(filepath.Join(repoRoot, entry.File))
sourceExists := !errors.Is(sourceErr, fs.ErrNotExist)
if sourceExists && sourceErr != nil {
    return nil, sourceErr
}
// Mirror existence is checked UNCONDITIONALLY — a missing source must not hide a
// missing or diverged mirror (V18). S5 for the missing source itself is emitted by
// checkRegistries, not here; this loop never double-reports it.
mirrorBytes, mirrorErr := os.ReadFile(filepath.Join(repoRoot, mirrorPath))
if errors.Is(mirrorErr, fs.ErrNotExist) {
    violations = append(violations, fmt.Sprintf("%s version %s: mirror file missing at %s", state, id, filepath.ToSlash(mirrorPath))) // S8
} else if mirrorErr != nil {
    return nil, mirrorErr
} else if sourceExists && !bytes.Equal(sourceBytes, mirrorBytes) {
    violations = append(violations, fmt.Sprintf("%s version %s: mirror bytes differ at %s", state, id, filepath.ToSlash(mirrorPath))) // S6/S7
}
```

(The `else if` chain here switches on the mutually exclusive outcomes of ONE read — missing /
error / readable — which is not arm-chaining; the independence requirement is across arms.)

The merge-base loop above it (`if baseEntry.Frozen == nil { continue }`, line 28) is **not
touched** (Non-Goals).

### 3. Arm independence (r2 — quorum objection 1)

The round-1 draft chained the arms and documented one masking corner as "acceptable". The quorum
correctly rejected that: masking contradicts Goal 1(c), the stated motivation (the current rc=2
abort already hides violations, V5 cell 11 / V18), and the no-silent-fallback axiom — and
V17/V18 measure real compound states at base. The redesign makes the five questions independent;
the COMPLETE dependency list, each one a semantic necessity:

| Arm | Depends on | Why |
|-----|------------|-----|
| hash-enforceable (S1/S2) | nothing | — |
| source existence (S5) | nothing | — |
| source bytes vs hash (S3/S4) | hashOK ∧ sourceExists | no enforceable hash → no referent; no file → no bytes |
| mirror existence (S8) | nothing | — |
| mirror bytes agreement (S6/S7) | sourceExists ∧ mirrorExists | a byte comparison needs both operands |

No other dependency is permitted. An entry in a compound bad state therefore emits EVERY
applicable named violation: invalid hash + missing source → S2 **and** S5 (AC9); source and
mirror both deleted → S5 **and** S8 (AC10).

### 4. `--migrate` consistency (`migrateRegistries` pre-flight)

Hazard adjudicated: today `--migrate`'s validation loop skips the active entry (V13), so it could
succeed on a tree where the widened `--check` is red (stale active hash) — a gate that `--check`
refuses but `--migrate` mints. Fix: the **validation** loop iterates ALL entries (drop
`id == r.Active || e.Frozen != nil` from the validation loop only; frozen entries on a green tree
match by definition, V6). The **freeze-writing** loop keeps both skips unchanged — migrate still
never freezes the active entry. `TestPromptFreezeMigrate_*` fixtures already carry fresh hashes
for every entry including `active` (V14), so no existing test breaks; `_RefusesStaleHash` keeps
its meaning.

### 5. PLACEHOLDER adjudication (scope item d)

Widening hash-enforceability to mutable entries makes a committed `PLACEHOLDER` hash a CI
violation (S2) while the standard-mode **loader** keeps honoring the hatch at runtime (V16,
Non-Goals). Position: the hatch remains a *local, uncommitted* development convenience; a
registry **committed to dev** must be enforceable, because the mirror/bytes arms are vacuous for
an entry whose hash verifies nothing. Cost today: zero — 0 of 59 entries use PLACEHOLDER (V6
shows the one mutable entry is 64-hex; the 58 frozen were already banned). `create_prompt_version.sh`
never emits PLACEHOLDER (V10), so the scripted workflow is unaffected.

### Files to modify

- `cmd/ailang/prompt_freeze_core.go` — widen per-entry loop, S5 adjudication, mirror-only sweep,
  count in signature (~+30/-10 LOC)
- `cmd/ailang/prompt_freeze_check_git.go` — widen mirror loop, S8 (~+12/-6 LOC)
- `cmd/ailang/prompt_freeze.go` — count printout, adapt to new `checkRegistries` signature (~+4 LOC)
- `cmd/ailang/prompt_freeze_check_git_test.go` — new tests per the mutation table (~+170 LOC)
- `cmd/ailang/prompt_freeze_test.go` — **REQUIRED, added r3 (quorum objection, `oc-glm-5-2`): it holds
  TWO further in-process `checkRegistries` callers (`:125`, `:138`) that the signature change breaks.
  The r2 list named only `prompt_freeze_check_git_test.go` and was incomplete — see V20** (~+4 LOC)
- `make/code-health.mk` — comment only: target help text says "frozen prompt immutability"; update
  to "prompt registry integrity (all entries) + frozen immutability" (1 line)
- `CHANGELOG.md`

---

## Conflict Surface

What else reads these registries/files, and why each survives the widening:

| Surface | Reads | Verdict |
|---------|-------|---------|
| `internal/eval_harness/prompt_loader.go` | both trees via `rootDir`; already refuses stale mutable hashes at load (`MutableHashMismatchError`, V16) unless PLACEHOLDER | **Agrees.** The widened gate makes CI match the loader. The one divergence (PLACEHOLDER) is adjudicated in §5: loader hatch untouched, CI refuses committed placeholders |
| `internal/prompt/loader.go` (agent mode) | embedded-first registry + `.md`; performs NO hash verification (V9) | **No conflict** — it never enforces, so it cannot disagree. Widened mirror agreement *helps* it: embedded bytes = source bytes for all entries once green |
| `.claude/skills/prompt-manager/scripts/create_prompt_version.sh` (NOT `scripts/`, V10) | writes both registries + new `.md` in both trees, computes real hash at creation, already tells the user to run `make check-prompt-freeze` before committing | **Agrees.** The mutable edit→regen-hash workflow stays green when done in one commit, which is what its own step 3 + step 6 prescribe |
| `cmd/ailang/eval_suite_manifest.go` | neither registry — only records `prompt_version` strings (V12) | **No conflict** |
| `migrateRegistries` (`--migrate`) | validation loop currently skips active + frozen (V13) → could mint what `--check` refuses | **Real hazard; adjudicated** in Solution Design §4 (validate all, freeze-write unchanged; fixture already compatible, V14) |
| `freezeVersion` (single-version path) | validates its own target's hash-enforceability and source-bytes-vs-hash before any write, and writes ONLY the two registry files (never a `.md`) — **measured r3, V22** | **No conflict** — it cannot mint a violation of any widened arm: the two hash arms are pre-validated, the two `.md` arms are untouched because it writes no `.md`, and registry-vs-registry agreement holds by construction because both registries are written from the same in-memory `r` |
| Existing tests (V15) | exact violation strings + rc via helper subprocess | S1/S3/S6/S9/S11 unchanged; the only renamed case (mirror deletion → S8) has no existing assertion; `_UnmodifiedTreeIsGreen` stays green because the fixture writes fresh hashes and matching mirrors for its single entry regardless of the `frozen` flag |
| **`checkRegistries` call graph (r3)** | the signature changes `([]string, error)` → `([]string, int, error)`; repo-wide inventory is **4 callers + 1 definition + 1 comment** (V20): production `prompt_freeze.go:69`; tests `prompt_freeze_check_git_test.go:134`, `prompt_freeze_test.go:125`, `prompt_freeze_test.go:138` | **Real gap, now closed.** All four callers are in `package main` in `cmd/ailang`; there is no external consumer, so the change cannot break another package. `prompt_freeze_test.go` is added to Files to modify. A caller ignoring the new `int` cannot silently receive a stale zero, because Go forbids assigning 3 results to 2 variables — the compiler is the enforcer |
| **`prompt freeze --check` stdout consumers (r3)** | inventory across `*.sh`/`*.yml`/`*.mk`/`Makefile`/`*.go` (V21): the ONLY consumer is `make/code-health.mk:185` (`@go run ./cmd/ailang prompt freeze --check`), reached from `.github/workflows/ci.yml:205` and `make/ci.mk:11`. Remaining hits are help text (`help.go:285`, `prompt.go:154`) and shell comment/echo lines in both `create_prompt_version.sh` copies | **No conflict.** The recipe neither pipes nor parses stdout and asserts nothing about it — the exit code is the whole verdict. Nothing anywhere requires stdout to be empty, so the unconditional `checked N registry entries` line is safe |
| CI / Makefile | invokes the same command | **No wiring change**; stdout gains one `checked N registry entries` line (safe — see the row above) |

---

## Acceptance Criteria

Every command runs against the probe protocol of the Verification Log (same construction, same
binary rebuild post-change) unless marked repo-root. **Base rc measured this iteration** is
stated per AC; an AC already at its target state at base would be broken — none is.

| # | Criterion | Command (probe repo unless noted) | rc at base (measured) | rc required post-change |
|---|-----------|-----------------------------------|----------------------|------------------------|
| AC1 | Pristine stays green, and the count line appears | repo-root: `make check-prompt-freeze`; probe: pristine `--check` | 0 (V7, V4 cell 1) | 0, stdout contains `checked 59 registry entries` (can fail: any widened arm misfiring on the real registry, or count wrong) |
| AC2 | Mutable source divergence refused | `printf 'x' >> prompts/v0.16.6.md` → `--check` | **0** (V4 cell 5 — the hole) | 1, stderr contains `mutable version v0.16.6: file bytes do not match recorded hash` |
| AC3 | Mutable source deletion refused, named | `rm prompts/v0.16.6.md` → `--check` | **0** (V4 cell 6) | 1, contains `mutable version v0.16.6: prompt file missing at prompts/v0.16.6.md` |
| AC4 | Mutable mirror divergence refused | `printf 'x' >> cmd/ailang/prompts/v0.16.6.md` → `--check` | **0** (V4 cell 7) | 1, contains `mutable version v0.16.6: mirror bytes differ at cmd/ailang/prompts/v0.16.6.md` |
| AC5 | Mutable PLACEHOLDER refused | set `v0.16.6` hash to `PLACEHOLDER` in BOTH registries → `--check` | **0** (V5 cell 10) | 1, contains `mutable version v0.16.6: recorded hash is not a 64-hex sha256 (unenforceable)` |
| AC6 | Mirror-only entry refused AND counted | add `v9.9.9-extra` to mirror registry only → `--check` | **0**, and no count line exists at base (V4 cell 8) | 1, contains `entry v9.9.9-extra missing from source registry`, and stdout reads `checked 60 registry entries` — 59 source-visited + 1 mirror-only, each incremented inside its visiting loop body (§1); pristine prints `checked 59` (AC1), so the move 59→60 is the assertion, not the verdict alone |
| AC7 | Missing FROZEN source is a named violation, not an abort | `rm prompts/v0.3.21.md` → `--check` | **2**, bare `open …: no such file` (V5 cell 11) | 1, contains `frozen version v0.3.21: prompt file missing at prompts/v0.3.21.md` |
| AC8 | Frozen arms unchanged | V4 cells 2-4 re-run; `go test ./cmd/ailang -run 'TestFreezeCheck\|TestPromptFreeze'` at repo root | cells rc=1 with the strings quoted in V4 (positive controls) | identical rc; cells 2-3 identical strings; cell 4 now S8 (`frozen version v0.3.21: mirror file missing at cmd/ailang/prompts/v0.3.21.md`); existing tests pass (can fail: any accidental string drift) |
| AC9 | **Compound (r2): invalid hash AND missing source emit BOTH named violations** | set `v0.16.6` hash to `PLACEHOLDER` in BOTH registries AND `rm prompts/v0.16.6.md` → `--check` | **0** (V17 cell 13 — the compound hole) | 1, stderr contains BOTH `mutable version v0.16.6: recorded hash is not a 64-hex sha256 (unenforceable)` AND `mutable version v0.16.6: prompt file missing at prompts/v0.16.6.md` |
| AC10 | **Compound (r2): source AND mirror both deleted emit BOTH named violations** | `rm prompts/v0.3.21.md cmd/ailang/prompts/v0.3.21.md` → `--check` | **2**, only the first missing file named, mirror deletion invisible (V18 cell 14) | 1, stderr contains BOTH `frozen version v0.3.21: prompt file missing at prompts/v0.3.21.md` AND `frozen version v0.3.21: mirror file missing at cmd/ailang/prompts/v0.3.21.md` |

---

## Mutation / refusal-branch test plan

The headline verb is "refuse", so every refusal branch gets one neutering mutation of the form
`if false && <cond>` — or, for the r2 independence branches (M8/M9), a condition-WRAPPING mutant
that reintroduces the round-1 masking dependency (never delete a block — every import stays
used, so "mutant does not compile" can never masquerade as "guard fired"). All new tests use the existing harness
(`newFreezeCheckRepo` + `writeFreezeCheckRegistries(…, frozen=false)` for mutable fixtures) and
observe **both** the in-process `checkRegistries` violation slice and the helper-subprocess rc —
both strictly downstream of the mechanism (the violation string is *produced by* the guarded
`append`; no test asserts a value set alongside it).

| # | Refusal branch | Neutering mutation | Test that must go red | Downstream observable |
|---|----------------|--------------------|-----------------------|----------------------|
| M1 | S2 mutable hash-enforceable | `if false && !eval_harness.IsHexSHA256(e.Hash)` | `TestFreezeCheck_MutablePlaceholderIsRed` (new) | violation containing `unenforceable` + rc 1→0; only this branch emits S2 |
| M2 | S4 mutable bytes-vs-hash | `if false && (h != e.Hash)` (mutable-state fixture) | `TestFreezeCheck_MutableSourceDivergenceIsRed` (new) | S4 string + rc; the frozen branch cannot fire on a `frozen:false` fixture |
| M3 | S5 missing source file | `if false && errors.Is(xerr, fs.ErrNotExist)` | `TestFreezeCheck_SourceMissingIsViolationNotError` (new; runs one frozen + one mutable sub-case) | with the branch neutered the read error propagates → `checkRegistries` returns `err` → in-process `t.Fatal` path + helper rc=2, not 1: the test asserts rc==1 AND the S5 string, both of which only the branch produces |
| M4 | S7 mutable mirror divergence | `if false && !bytes.Equal(sourceBytes, mirrorBytes)` | `TestFreezeCheck_MutableMirrorDivergenceIsRed` (new) | S7 string + rc; no other arm reads mirror `.md` bytes (V3) |
| M5 | S8 mirror file missing | `if false && errors.Is(mirrorErr, fs.ErrNotExist)` | `TestFreezeCheck_MirrorMissingIsRed` (new; frozen + mutable sub-cases) | S8 string; neutered → mirrorErr falls to the `bytes.Equal` arm with nil bytes? No — neutered branch falls through to `else if mirrorErr != nil → return err` → rc=2, so the rc==1 assertion goes red for the reason it claims |
| M6 | S10 mirror-only entry | `if false && (source.Versions[id] == nil)` | `TestFreezeCheck_MirrorOnlyEntryIsRed` (new) | S10 string + rc; no other loop visits mirror-only keys (V3, V4 cell 8) |
| M7 | migrate validation widening | restore `if id == r.Active` skip inside validation via `if false \|\| id == r.Active { continue }` | `TestPromptFreezeMigrate_RefusesStaleActiveHash` (new: fixture with stale `active` hash must make `migrateRegistries` return error) | migrate's returned error is the direct product of the validation loop; with the skip restored, migrate succeeds and the test's error assertion fails |
| M8 | **arm independence (r2)**: source-existence must not depend on hash validity | wrap arm 2's emission as `if hashOK && !sourceExists` (reintroduces round-1's masking; no block deleted, all imports stay used) | `TestFreezeCheck_InvalidHashAndMissingSourceEmitsBoth` (new; AC9 fixture: `frozen=false`, hash `PLACEHOLDER`, source file absent) | the test asserts BOTH S2 and S5 in the violation slice; under the mutant only S2 is emitted, so the test goes red specifically on the S5 assertion — the string only arm 2 produces |
| M9 | **arm independence (r2)**: mirror-existence must not depend on source existence | wrap the S8 emission as `if sourceExists && errors.Is(mirrorErr, fs.ErrNotExist)` | `TestFreezeCheck_BothTreesDeletedEmitsBoth` (new; AC10 fixture: both `.md` files removed) | asserts BOTH S5 and S8; under the mutant only S5 is emitted, red specifically on the S8 assertion — no other arm produces the mirror-missing string (V3) |
| **M-ADD** | enumerator completeness (**addition**, not removal: every removal proves the check FIRES; only an addition proves it LOOKS) | fixture-side: ADD `v9.9.9-extra` to the mirror registry of an otherwise-green fixture | `TestFreezeCheck_CheckedCountMovesOnAddition` (new) | `checked` is incremented only inside the loop bodies that visit an entry (§1, r2) — it counts VISITS, never a `len()` computed beside the work. Fixture arithmetic: 1 source entry alone → count 1; with the mirror-only entry added → count MUST move to 2 AND S10 must fire. An implementation whose enumerator never visits mirror-only keys contributes neither the increment nor the violation → red **on the count**, not merely the verdict. Honest scope (r2): the count detects enumeration blindness only; a neutered arm inside a visited body leaves the count intact and is killed by M1–M9 — the two observables are deliberately complementary |

Test-design constraint honored: every test drives the REAL check path (`checkRegistries` +
helper subprocess) against synthetic git repos — no test seeds a verdict by raw fixture, which is
how the parent sprint's tests once stayed green while a writer did not exist.

---

## Milestones

### M1 — the whole change (≤1 day)

Solution Design §1-§5 + all tests in the mutation table + AC1-AC8 + CHANGELOG. Single milestone:
the diff is ~50 production LOC across three files already read end-to-end this iteration, and the
test harness exists (V15).

---

## Testing Strategy

- Extend `cmd/ailang/prompt_freeze_check_git_test.go` using `newFreezeCheckRepo` /
  `writeFreezeCheckRegistries` (the `frozen bool` parameter already exists, V15); mutable
  fixtures pass `frozen=false`. Migrate test extends `freezeFixture` in `prompt_freeze_test.go`.
- Assert exact violation strings (they are normative, table S1-S11) via the in-process slice, and
  rc via the existing `TestFreezeCheckHelperProcess` subprocess.
- Run the V4/V5/V17/V18 probe matrix once by hand against the rebuilt binary before merging: all
  five former rc=0 holes (cells 5, 6, 7, 8, 10) and the compound cell 13 must read rc=1; both
  rc=2 aborts (cells 11, 14) must become rc=1 with every applicable violation named (cell 14:
  both S5 and S8); cells 2-3 byte-identical; cell 4 = S8; pristine rc=0 with
  `checked 59 registry entries`.
- `make check-prompt-freeze` at repo root must stay rc=0 (AC1) — guaranteed non-vacuously by V6
  (mutable hash fresh in both trees at base).

---

## Open Questions

1. The 3 unregistered `.md` files under `prompts/` (V11) are invisible to a registry-driven
   enumerator by construction. Should the gate also refuse *unregistered* files in the prompt
   trees (a directory sweep), or are docs-alongside-prompts legitimate? Deferred — separate
   decision, touches the prompt-manager workflow.
2. Loader-side PLACEHOLDER hatch retirement (V16): with CI refusing committed placeholders (§5),
   the runtime hatch protects only uncommitted local state. Retire it entirely? Parent doc Q6
   deliberately kept it; not re-opened here.
3. `MutableHashMismatchError` has no test referencing it (V9). The loader behavior predates this
   doc; a loader-side test is cheap but belongs to the parent doc's test surface, not this gate's.

---

## Quorum revision log (round 1 → r2 → r3)

Round-1 verdict: **blocked**, 3/3 reviewers present (no degrade). Neither blocking objection
disputed the design direction; both were accepted without argument.

| Objection | Reviewer | Disposition in r2 |
|-----------|----------|-------------------|
| "The proposed control flow silently suppresses independent integrity violations … an entry with an invalid hash and missing source, or with both source and mirror deleted, does not receive all applicable named violations" | gpt5-6-sol | **Accepted.** §1/§2 reworked into independent arms; the complete dependency list is now a table in §3 with each remaining dependency justified as a semantic necessity (only the two byte-comparisons are conditional). The round-1 §3 "corner case, stated deliberately" — which documented the masking as acceptable — is deleted. Both compound states were MEASURED at base (V17: rc=0, the compound hole; V18: rc=2 naming only the first missing file) and are pinned by new AC9/AC10 plus independence mutants M8/M9 |
| "`checked` … is computed entirely outside the per-entry work … structurally decoupled from whether any entry's violation arms actually fire; AC6 and M-ADD assert a coverage property the count variable cannot deliver" | oc-glm-5-2 | **Accepted.** `checked` is now incremented exactly once per entry, as the last statement of the loop body that performs that entry's arms — never via `len()` beside the work (§1). Arithmetic restated unambiguously: pristine real registry 59 → mirror-only-added 60; test fixture 1 → 2. M-ADD's kill condition stays on the COUNT, and the doc now states the observable's honest scope: it detects enumeration blindness (unvisited entries), not neutered arms inside visited bodies — those belong to M1–M9; the observables are complementary |
| (pass-note, not blocking) "`source.Versions[id] == nil` … if `source.Versions` is a map of structs, this will trigger a compilation error" | gemini-3-1-pro (pass) | **Refuted by controller measurement, re-derived first-party as V19**: `Versions map[string]*registryEntry` (`prompt_freeze_core.go:29`) is a pointer map, and the identical nil-deref already ships at line 351. No design change |


### Round 2 → r3 (controller, narrow-refinement carve-out)

Round-2 verdict: **blocked**, 3/3 reviewers present (`absent_reviewers` empty, no degrade). All
three objections were **premise/completeness** objections about ONE surface — the call graph and
consumers of the two interfaces this change touches. None disputed the design direction and each
carried a concrete reviewer-authored fix, so the controller applied them under the
narrow-refinement carve-out rather than spending a third designer run (the Fable diet's one-doc
ceiling — authoring plus one protocol-mandated revision — was already reached). Per rule 3f the
controller **measured** each premise rather than forwarding it; the fixes below are the
measurements, not controller-invented resolutions.

| Objection | Reviewer | Disposition in r3 |
|-----------|----------|-------------------|
| "The doc proposes changing `checkRegistries`' signature … but never verifies the call graph … if `prompt_freeze_test.go` or any other file calls `checkRegistries` directly … the list is incomplete and the signature change silently breaks compilation" | oc-glm-5-2 | **CONFIRMED by measurement — a real gap.** `prompt_freeze_test.go` holds two further in-process callers (`:125`, `:138`) and was absent from "Files to modify". Added, with the complete 4-caller inventory as **V20** and a new Conflict-Surface row. The reviewer's stale-zero worry is separately answered: Go rejects a 2-variable assignment from 3 results, so a caller cannot silently ignore the new `int` |
| "…no repository-wide inventory of Go call sites or shell/CI consumers that may parse or require empty stdout" | gpt5-6-sol | **Inventory produced; the risk is REFUTED.** **V20** covers the Go call sites; **V21** covers the stdout consumers and finds exactly one invocation (`make/code-health.mk:185`), which neither pipes nor parses stdout — the exit code is the whole verdict. Every other hit is help text or a shell comment. Both are now Conflict-Surface rows. The reviewer was right that nobody had checked; checking clears it |
| "…`freezeVersion` 'validates its own target's hash/bytes before writing' … is a load-bearing behavioral claim used to justify omitting `freezeVersion` from the widening scope, yet it entirely lacks a Verification Log entry" | gemini-3-1-pro | **CONFIRMED by measurement, and the row is now stronger than the claim it replaces.** **V22**: all three validations (`:316`, `:319-322`, `:323`) precede the first write at `:329`; and the write-surface probe shows `freezeVersion` writes **no `.md` at all** (0 hits, control 5), so it cannot mint a violation of the two byte-arms either, and writes both registries from one `r` so registry agreement holds by construction |

---

## Related Documents

- **Parent**: `design_docs/planned/m-prompt-version-freeze-on-first-bank.md` — the freeze
  mechanism this doc widens; its Q6/V-C define the PLACEHOLDER hatch, its L3 defines the gate.
- **PR #937** (squash `445ccb550`) — landed the gate and its explicit CI step (V2).
- `design_docs/PROGRAM.md` — routing: harness/data-integrity lane, not a core change.
- `make/code-health.mk:184`, `.github/workflows/ci.yml:205` — the wiring under change-freeze here.
