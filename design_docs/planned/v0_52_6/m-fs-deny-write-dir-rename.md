# M-FS-DENY-WRITE-DIR-RENAME: Deny patterns must protect their ancestor directories

Refs #1569 (do not open a new issue; tracked there). Triage 2026-10-08 verified the
defect live on `origin/dev` 658ff76a3; this session reproduced it and two sibling
variants against the installed v0.52.5 binary (commit 7200786) through the real
`--policy` gate (Verification Log, V1–V4).

**Status**: Planned
**Target**: v0.52.6 (next patch after v0.52.5)
**Priority**: P0 — sandbox soundness: an advertised filesystem restriction is bypassable
**Estimated**: 1–2 days (matcher + wiring + denial tests)
**Dependencies**: None new; builds on M-EXECUTOR-POLICY-HARDENING M6/M7 (implemented
v0.41.x) and the unified folded matcher from #1559 (v0.52.1)
**Routing**: AILANG fix (internal/fileguard + internal/effects); no language-semantics
change, no motoko core change

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | The matcher is pure and lexical; no new nondeterminism |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | No effect-row changes; FS ops keep their signatures |
| A4: Explicit Authority | +1 | The authority an operator grants via `fs_deny_write` is now enforced across structural moves, not just direct writes |
| A5: Bounded Verification | 0 | Local tests only; no verification-surface change |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | The refusal names the pattern and the protected relationship; an agent can self-correct without human help |
| A8: Minimal Syntax | 0 | No new syntax; no new pattern grammar |
| A9: Cost Visibility | 0 | No resource accounting changes |
| A10: Composability | +1 | One rule in the ONE shared matcher; every current and future structural op on both enforcement paths inherits it |
| A11: Structured Failure | +1 | Refusal is a typed, named reason (`E_FS_PROTECTED`) that distinguishes ancestor-protection from a direct pattern match |
| A12: System Boundary | +1 | The deny boundary inside the sandbox stays closed when the ground under it is moved |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): closes an ambient-authority hole; grants none
- [x] A7 (Machines First): refusal message designed for agent self-correction

## Problem Statement

`fs_deny_write` patterns are matched against the path an operation names
(`fileguard.MatchDenyWrite`, `internal/fileguard/protect.go`). A rename or removal
does not name the paths it moves: renaming a directory relocates every path beneath
it, and a deny pattern that points inside that directory never sees the operation.
A confined program can therefore:

1. `renameFile(".claude", "tmp")` — neither operand matches `.claude/settings.json`;
2. `writeFile("tmp/settings.json", …)` — the protected file, at its new spelling, matches nothing;
3. `renameFile("tmp", ".claude")` — the tampered file is back at the protected path.

**Reproduced live this session** (installed v0.52.5 binary, real `ailang run --policy`
gate, restricted mode, `fs_deny_write = [".claude/settings.json"]`): the three-step
program exited 0 and `.claude/settings.json` contained the attacker's content. The
direct write to the same path is correctly refused with `E_FS_PROTECTED` — the deny
itself works; only the move evades it (V1, V2). Triage reproduced the same defect on
`origin/dev` 658ff76a3 on 2026-10-08.

Two sibling variants the issue does not mention are **also** live:

- **The `dir/**` workaround has the same hole one level up.** Issue #1569 states
  "`/**` patterns are not affected". That is true for renames at or below the
  pattern's directory, but false for its ancestors: with `fs_deny_write =
  ["a/b/**"]`, `renameFile("a", "t")` succeeds (reproduced, V3) — every match
  beneath `a/b` is moved out from under the pattern in one step.
- **Mid-glob patterns lose wildcard-level directories.** With `["a/*/x.txt"]`,
  `renameFile("a/b", "c")` succeeds (reproduced, V4) — `a/b/x.txt` matches the
  pattern, `c/x.txt` does not.

**Impact:** every policy that uses a file-level or mid-glob deny pattern — the
operator's supply-chain surfaces (`.github/workflows/*`, `Makefile`, editor hook
config) — is writable by the very programs the pattern exists to fence off. This is
the same asset class #1559 closed (editor hooks run *outside* the sandbox); the
confidentiality/integrity claim of `--policy` and the agent lanes is currently false
for these spellings. Both enforcement paths are exposed: standalone
`ailang run --policy`, and the agent lanes via `ailang_run`, which submits programs
to the same gate.

**Root cause:** `fsCheckMutation` (and policy-tool's `protected`) ask "does the path
I am naming match a deny pattern?" A structural mutation of a directory must instead
ask "can a deny pattern match anything in the subtree I am about to relocate?"

## Goals

**Primary Goal:** a confined program cannot move, remove or plant a deny-listed
path out from under its pattern by renaming or removing an ancestor directory.

**Success Metrics:**
- The three reproduced counterexamples (V1 literal, V3 `/**`-ancestor, V4
  mid-glob) become permanent passing denial tests through the real effect
  dispatcher — the M-EXECUTOR-POLICY-HARDENING evidence invariant.
- No allowed operation regresses: with no deny patterns in force, every FS op
  behaves identically (fsCheckMutation's early-return already guarantees this
  structurally); with patterns in force, unrelated renames — including the
  `run.json.tmp → run.json` atomic publish and base-glob `*.yml` policies — still
  succeed (V5 shows base patterns follow the file, so nothing needs to change).
- The rule lives in the ONE shared matcher (`internal/fileguard`), so the
  running-program path (`internal/effects`) and policy-tool
  (`internal/policytool`) inherit it together, as #1569 requests.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Ancestor protection is a **rename/remove-time check in the shared matcher**, not a load-time expansion of patterns into extra deny entries | Load-time expansion cannot express glob ancestors (`a/*/x.txt` protects `a`, `a/**` protects every ancestor chain) and would silently fork the matcher's two consumers again — the exact duplication M-EXECUTOR-POLICY-HARDENING M1 removed | human | design | med |
| Refusal predicate: pattern **can match the named path or anything strictly beneath it** (component-wise prefix match, base-name matching deliberately excluded) | This is the complete fix: it covers literal, `/**`, and mid-glob patterns in one rule; base-name patterns must be excluded or every directory rename under a `*.yml` policy is refused for no security gain (V5) | human | design | med |
| Rename checks **both operands** with the new rule | Renaming `tmp → .claude` plants a whole tree at the protected path without ever matching the pattern; checking only the source leaves the write-side of the bypass open | agent | design | low |
| Writes and mkdirs stay on the plain check | Writing a file at an ancestor path (`writeFile(".claude")`) cannot plant content at a protected target — the target write itself already matches the pattern; over-denying there breaks legitimate file creation | human | design | low |
| Over-denial is the fail-closed direction | Precedent: #1559 folding ("over-denying an exotic spelling costs nothing"); removing an empty ancestor directory is refused rather than permitted-by-technicality | human | design | low |
| No policy-tool call-site change | policy-tool exposes no rename/remove op today (V7); it inherits the rule through the shared matcher for any future structural op | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] The refusal predicate (see Solution Design): direct match OR can-match-strictly-beneath, with base-name matching excluded from the beneath-rule
- [x] Both rename operands are checked; remove ops are checked with the same rule
- [x] The rule lives in `internal/fileguard` beside `MatchDenyWrite`; no second matcher
- [ ] Exact refusal wording (the message shape below is the design; final phrasing may be tuned at review)

## Solution Design

### Overview

A deny pattern is a description of a set of paths. Today the matcher only answers
"is this path in the set?" for the path an operation names. This design adds the
question a structural mutation actually needs: "is any path in the set at or
beneath the directory this operation relocates?" If yes, the operation is refused
— moving the directory would move the pattern's targets out from under the
pattern, and (for the destination operand) could plant attacker content at a
protected path.

The rule is one function in the existing shared matcher, wired into exactly the
ops that relocate subtrees: rename and remove. Nothing else changes — not the
pattern grammar, not the policy file format, not the error code, not the
folding rules from #1559.

### The refusal predicate

For a rename or removal of root-relative path `rel`, refuse iff:

**(a)** today's `Protection.Check(rel)` refuses it — a `.git` component, or a
direct deny match (full-path or base-name). Unchanged; this already refuses
renaming/removing the protected file itself, and renaming a `/**` pattern's own
directory (the `CutSuffix` equality: `dir/**` matches `dir`).

**(b)** NEW: some pattern, folded and read as a slash-separated component
sequence (a trailing `"/**"` becomes a final `"**"` component), component-wise
prefix-matches `rel` **with at least one pattern component left over** — i.e.
the pattern can match a path *strictly beneath* `rel`:

- each of `rel`'s components is matched against the corresponding pattern
  component: a literal component must be equal (after the #1559 case/NFC fold),
  a glob component (`*`, `?`, `[…]`) is matched with `path.Match` per component;
- a pattern component that is exactly `**` matches any remainder — it returns
  "refuse" immediately (this both implements the documented `dir/**` semantics
  and fails closed if an operator writes `**` in the middle of a pattern, which
  policy validation does not currently restrict);
- a `path.Match` error (malformed glob) counts as "can match" — fail closed, the
  same direction #1559 chose for exotic spellings.

Base-name matching (pattern against the file's base) is **deliberately not
consulted** in (b). A base pattern like `*.yml` or `Makefile` matches at any
depth, so it *follows the file wherever it is moved* — verified live: under
`fs_deny_write = ["*.txt"]`, `renameFile("a", "t")` is correctly allowed and
`writeFile("t/x.txt", …)` is correctly refused (V5). Counting base matches in
(b) would refuse every directory rename under such a policy and close no hole.

Why this one rule covers every reproduced variant:

| Pattern | `rel` being renamed/removed | Component test | Verdict |
|---|---|---|---|
| `.claude/settings.json` | `.claude` | `[.claude]` matches, `settings.json` remains | refuse (closes V1) |
| `a/b/**` | `a` | `[a]` matches, `[b, **]` remain | refuse (closes V3) |
| `a/*/x.txt` | `a/b` | `[a, *]` matches, `x.txt` remains | refuse (closes V4) |
| `a/*/x.txt` | `a` | `[a]` matches, `[*, x.txt]` remain | refuse |
| `*.yml` | `src` | one component, none left over; `*.yml` ≠ `src` | allow (correct: pattern follows the file) |
| `Makefile` | `src` | ditto | allow (same reasoning) |
| `.claude/settings.json` | `.claude/sub` | `settings.json` ≠ `sub` at the second component | allow (correct: this move does not relocate the target) |
| `a/**` | unrelated `x` | `a` ≠ `x` at the first component | allow |

Note the M6 `.git` guard needs no ancestor rule at all: `UnderGitDir` tests for a
`.git` *component anywhere in the path*, so `.git` protection already travels
with the directory wherever it is renamed (reading `protect.go`, V6).

### API: the shared matcher

In `internal/fileguard/protect.go` (the ONE matcher both enforcement paths share):

```go
// Violation gains one field:
type Violation struct {
    GitDir   bool
    Pattern  string // first fs_deny_write pattern involved
    Ancestor bool   // NEW: the path is an ancestor of a deny target, not a match itself
}

// MatchDenyMove returns the first pattern that can match rel itself or any
// path strictly beneath it, or "". A rename or removal of rel relocates
// that whole subtree (#1569).
func MatchDenyMove(patterns []string, rel string) string

// CheckMove applies the protection to a structural mutation of rel — a
// rename or removal. It refuses everything Check refuses, and additionally
// any path a deny pattern can match BENEATH: moving an ancestor directory
// moves the pattern's targets out from under the pattern.
func (p Protection) CheckMove(rel string) Violation
```

`MatchDenyMove` shares `cleanRel`/`foldKey` and the same `./`-trimming and
`ToSlash` normalization as `MatchDenyWrite` — no second path parser, no second
folding rule. `MatchDenyWrite` keeps its signature and behavior (its callers:
`Protection.Check` only, V8).

### Call sites (internal/effects)

A new `fsCheckMove` in `internal/effects/fs_root.go`, mirroring `fsCheckMutation`
but calling `CheckMove` and phrasing the refusal for moves:

```
E_FS_PROTECTED: <path> is an ancestor of fs_deny_write "<pattern>" — moving or
removing it would relocate the protected target out from under its pattern
```

(Reuse of `E_FS_PROTECTED` is intentional: no new error-code allocation, V12. The
`Ancestor` flag distinguishes this refusal from a direct match in the message.)

Wired into exactly the ops that relocate or delete subtrees — every call site was
enumerated (V9):

| Op | File / line (current) | Change |
|---|---|---|
| `FS.renameFile` | `fs_dir.go:130` (checks at 135, 138) | both operands → `fsCheckMove` |
| `FS.renameFileResult` | `fs_dir.go:176` (checks at 181, 184) | both operands → `fsCheckMove` |
| `FS.removeFile` | `fs_dir.go:91` (check at 96) | → `fsCheckMove` |
| `FS.removeFileResult` | `fs_dir.go:156` (check at 161) | → `fsCheckMove` |
| `FS.removeDirResult` | `fs_dir.go:242` (check at 247) | → `fsCheckMove` |
| `FSRemove` (exported helper: std/zip, std/gzip, std/tar, zip_xml) | `fs_root.go:270` (check at 271) | → `fsCheckMove` |

Unchanged (plain `fsCheckMutation`, deliberately — see High-Impact Decisions):
all write/append ops (`fs_write.go`, `fs_bytes_result.go`), `FSCreate` /
`FSWriteFile` (`fs_root.go:231/244`), and every mkdir op (`fsMkdir`,
`fsMkdirAll`, `fsMkdirAllResult`, `fsMkdirResult`, `FSMkdirAll`). Creating an
ancestor directory moves nothing and plants nothing.

### policy-tool

No call-site change. policy-tool's write surface (`write`, `edit`, and the
schema-gated CLI writes) never renames or removes, and its request fields admit
no move op (V7) — so there is nothing to wire today. The point of putting the
rule in `internal/fileguard` (as #1569 asks) is that policy-tool's `protected()`
gate and any future structural op inherit it automatically, and the agent lanes
are closed *indirectly* the moment the effect path is fixed, because `ailang_run`
submits to `ailang run --policy` — the same binary, the same matcher.

### Interaction with the `/**` workaround

After this fix the two spellings are equivalent in ancestor protection:

- `dir/x` (file-level): the pattern's directory chain is now structure-frozen —
  `renameFile("dir", "tmp")` and `removeFile("dir")` are refused, so the
  three-step dance cannot start, and `renameFile("tmp", "dir")` is refused, so
  a prepared tree cannot be planted at the protected path.
- `dir/**`: keeps its existing, already-correct refusal for moves at or below
  `dir` (the `CutSuffix` equality and prefix rule), and *gains* the ancestor
  level it was missing (V3): with `a/b/**`, renaming `a` is now refused.

The workaround therefore remains valid (it is still required for its real
purpose — denying a directory's *siblings*, i.e. everything under `dir`, not just
one file) and becomes strictly stronger. No policy that used the workaround
loses protection; no policy that didn't will need it for this class of bypass
any more.

### Implementation Plan

**Phase 1: matcher** (~2 hours)
- [ ] Add `Ancestor` to `Violation`; add `MatchDenyMove` + `mayMatchBeneath` helper + `Protection.CheckMove` in `protect.go`
- [ ] Unit table in `protect_test.go`: every row of the predicate table above, plus case/NFC-fold variants (`renameFile(".CLAUDE", …)` under `.claude/settings.json`), `**`-mid-pattern fail-closed, malformed-glob fail-closed

**Phase 2: wiring** (~2 hours)
- [ ] `fsCheckMove` in `fs_root.go`; swap the six call sites in `fs_dir.go` / `fs_root.go`
- [ ] Denial tests in `fs_deny_write_test.go`: the three reproduced dances (V1, V3, V4) as `TestFSDenyWrite_AncestorRenameRefused`, asserting `E_FS_PROTECTED` on each step and that the file content is unchanged on disk

**Phase 3: docs & changelog** (~2 hours)
- [ ] `std/fs.ail`: extend the `renameFile`/`removeFile` doc comments (deny patterns protect ancestor directories)
- [ ] `docs/docs/guides/agent-tool-policy.md`: extend the `fs_deny_write` paragraph (line ~118)
- [ ] `changelogs/v0.32-current.md`: entry under Fixed, referencing #1569 and the two sibling variants
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries` (fileguard is a leaf; effects does not import policytool — boundaries unchanged)

### Files to Modify/Create

**Modified files:**
- `internal/fileguard/protect.go` (+~55 LOC) — `Ancestor` field, `MatchDenyMove`, `mayMatchBeneath`, `CheckMove`
- `internal/fileguard/protect_test.go` (+~80 LOC) — matcher unit table
- `internal/effects/fs_root.go` (+~20 LOC) — `fsCheckMove`
- `internal/effects/fs_dir.go` (~6 one-line call-site swaps) — rename/remove ops
- `internal/effects/fs_deny_write_test.go` (+~70 LOC) — the #1569 denial tests
- `std/fs.ail` (+~6 LOC doc comments)
- `docs/docs/guides/agent-tool-policy.md` (+~4 LOC)
- `changelogs/v0.32-current.md` (+1 entry)

**New files:** none.

## Examples

### The #1569 repro, before and after

**Before** (live, v0.52.5, `fs_deny_write = [".claude/settings.json"]`):

```ail
module bypass1569
import std/fs (renameFile, writeFile)

export func main() -> () ! {FS} {
  renameFile(".claude", "tmp");                        -- allowed  (hole)
  writeFile("tmp/settings.json", "{\"hooks\":\"PWNED\"}"); -- allowed (matches nothing)
  renameFile("tmp", ".claude")                        -- allowed  (hole)
}
```
Result: exit 0, `.claude/settings.json` now contains attacker content — the exact
editor-hook asset class #1559 was filed about.

**After**: step 1 fails with
`E_FS_PROTECTED: .claude is an ancestor of fs_deny_write ".claude/settings.json"
— moving or removing it would relocate the protected target out from under its
pattern`, step 3 fails identically (the plant direction), and the file is
untouched. The direct write continues to be refused exactly as today.

### What still works (regression intent, not new behavior)

```ail
renameFile("run.json.tmp", "run.json")   -- atomic publish (std/fs @example): unchanged
renameFile("olddir", "newdir")          -- TestFSRenameFile_DirectoryWithinSandbox: unchanged
renameFile("a", "t")                    -- under fs_deny_write = ["*.txt"]: still allowed (V5);
                                         --   t/x.txt remains protected at its new location
```

## Success Criteria

- [ ] The V1, V3, V4 counterexamples are permanent passing denial tests through the real effect dispatcher (`TestFSDenyWrite_AncestorRenameRefused` + matcher table)
- [ ] `TestFSDenyWrite_PatternsProtectInsideRoot`, `TestFSDenyWrite_CaseVariantsRefused`, `TestFSCheckMutation_SharedMatcher`, `TestFSRenameFile_Success`, `TestFSRenameFile_DirectoryWithinSandbox`, `TestFSContainment_RenameDestination` and `TestPolicyTool_DenyWriteCaseVariants` pass unchanged
- [ ] With no deny patterns and no `.git` protection, no FS op changes behavior at all (structural early-return in the check)
- [ ] All tests passing (`make test`), `make fmt`/`make lint`/`make check-boundaries` clean
- [ ] Documentation updated (`std/fs.ail`, agent-tool-policy guide, changelog)

## Conflict Surface

This design touches `internal/effects` (op wiring) and the shared matcher — not
the parser, lexer, types or codegen; no language syntax changes. Per the
structure guide, the surface is enumerated anyway.

**Positions touched:** the mutation-gate call sites of the six structural FS
ops, and the `fileguard.Protection` API (one added method, one added struct
field).

**What else lives here:**

| Position | Existing behavior that shares it | Interaction |
|---|---|---|
| rename gate, both operands | Sandbox containment (both operands through the root, `fs_containment_test.go:292`), `.git` protection, direct deny matches | CheckMove strictly *adds* refusals; containment and `.git` verdicts are unchanged (CheckMove starts from Check) |
| remove gate (`removeFile`, `removeDirResult`) | The rig-lock release: `mkdirResult("lock.d")` / `removeDirResult("lock.d")` (fs_dir.go:220/242, "Daneel's rig-lock client") | Lock dirs are not deny ancestors under any real policy; if an operator denies `locks/**`, both the create and the release are already refused today by the direct match — the new rule adds nothing there |
| write/mkdir gates | NOT touched — `fsCheckMutation` keeps its exact semantics | A file write at an ancestor path (`writeFile(".claude")`) stays allowed: it cannot plant content at a protected target (the target write matches the pattern) and cannot move anything |
| `fileguard.Violation` consumers | `fsCheckMutation` (pattern → error text), policy-tool `protected()` (pattern → refusal text) | New `Ancestor` field is additive; both consumers render it in their existing message shapes |

**Programs that MUST still work** (regression fixtures, all exist and were read):

1. `internal/effects/fs_test.go:340` `TestFSRenameFile_Success` — the `run.json.tmp → run.json` atomic publish that `std/fs.ail` documents as renameFile's raison d'être
2. `internal/effects/fs_test.go:374` `TestFSRenameFile_DirectoryWithinSandbox` — plain directory rename
3. `internal/effects/fs_containment_test.go:292` `TestFSContainment_RenameDestination` — in-root renames into `a/b/c/`
4. `internal/effects/fs_deny_write_test.go:15` `TestFSDenyWrite_PatternsProtectInsideRoot` — existing refusal matrix incl. rename-into/​out-of-protected-path
5. `std/fs.ail` removeDirResult/mkdirResult lock pattern (rig-gate lease, `internal/effects/fs_dir.go:220`)

**What deliberately changes:** renaming or removing any directory from which a
deny pattern can be matched beneath — previously allowed, that permission *was*
the vulnerability (V1/V3/V4). Removing an *empty* ancestor directory is refused
too, where the kernel would have allowed the rmdir: fail-closed by design
(#1559 precedent), and it prevents delete-then-recreate dances at the protected
spelling.

## Testing Strategy

**Unit tests** (`internal/fileguard/protect_test.go`):
- Full predicate table (Solution Design), including fold variants and the
  `**`-mid / malformed-glob fail-closed rows
- `MatchDenyWrite` behavior is bit-identical (existing tests untouched)

**Integration tests** (`internal/effects/fs_deny_write_test.go`):
- `TestFSDenyWrite_AncestorRenameRefused`: each reproduced dance (literal
  pattern, `/**` ancestor, mid-glob), each step asserted `E_FS_PROTECTED`, file
  content on disk asserted unchanged — the M-EXECUTOR-POLICY-HARDENING evidence
  invariant: every original counterexample becomes a permanent passing denial test
- Allowed-side pins: rename of an unrelated dir under a base-glob policy stays
  allowed; `writeFile` at an ancestor path stays allowed

**Regression surface:** the five Conflict Surface fixtures above, run unchanged.

**Manual:** re-run this session's live `--policy` repro scripts (kept in the test
as fixtures) — expect the three-step program to fail on step 1.

## Deferred Decisions

- Exact `Violation` shape for the ancestor case (dedicated `Ancestor bool` field vs. a second struct) — agent may choose; the rendered refusal must distinguish it from a direct match
- Exact refusal wording (the freeze leaves the message shape; phrasing may be tuned) — agent may choose
- Whether `MatchDenyMove` returns the first or the most specific pattern when several match — agent may choose (first, like `MatchDenyWrite`, is the default)
- Whether policy load should *also* reject `**` in non-terminal pattern positions (it currently doesn't; the matcher fails closed meanwhile) — human at review; separate one-liner if wanted

## Non-Goals

**Not attempted in this feature:**
- Write-through-symlink at an operator-planted in-root symlink (`writeFile("alias")` where `alias → .claude/settings.json`): a distinct mechanism (lexical deny vs. in-root resolution), requires operator-planted state, and `fileguard`'s documented os.Root contract deliberately resolves in-root symlinks. Flagged for a separate issue if not already tracked; out of #1569's scope.
- Protection against *other processes* renaming directories in a shared sandbox: same trust domain as the operator who chose the sandbox root; no in-process check can help.
- Kernel-level immutability (mount namespaces, file flags): the runtime's deny list is a policy construct, not an OS guarantee; making that claim is out of scope.
- Recursive delete ops: none exist (`removeDirResult` is empty-only by design).
- New pattern grammar, new policy keys, new error codes: none.

## Timeline

**Day 1** (~4 hours): Phase 1 matcher + unit table; Phase 2 wiring + denial tests
**Day 2** (~2 hours): Phase 3 docs/changelog; full `make test` + boundaries; buffer

**Total: ~6 hours of work across 1–2 days** (planning estimate, not a sprint commitment).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Over-denial: a lane's legitimate rename of a directory that happens to be a deny ancestor is refused | Medium | The refusal names the pattern and the relationship (agent can self-correct); the operator re-scopes the pattern; allowed-side tests pin the `*.yml` and publish cases |
| Component-glob semantics surprise an operator (`*` per component, `**` special-cased) | Low | Documented in the guide + `std/fs.ail`; unit table is the executable spec |
| Mid-pattern `**` fail-closed refusals look arbitrary | Low | Message names the pattern; optional policy-load rejection is a deferred one-liner |
| A future structural FS op forgets to use `fsCheckMove` | Medium | This doc's call-site table + the one-matcher doctrine; a `make`/test sweep for `fsBackend` methods that relocate subtrees is the implementer's checklist item, not new tooling |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_41_0/m-executor-policy-hardening.md](../../implemented/v0_41_0/m-executor-policy-hardening.md) — M6 (`.git` guard) and M7 (`fs_deny_write`), the one-matcher doctrine this fix extends; its evidence invariant ("every original counterexample becomes a permanent passing denial test") is adopted here
- [design_docs/implemented/v0_19_0/m-fs-sandbox-diagnostics.md](../../implemented/v0_19_0/m-fs-sandbox-diagnostics.md) — sandbox refusal diagnostics (`E_FS_*`), the pattern the new refusal follows
- [changelogs/v0.32-current.md](../../../changelogs/v0.32-current.md) — #1559 (one folded matcher, fail-closed over-denial) and #1554 (policy-tool write gate) entries, the immediate lineage

**Planned (check for overlap):**
- [design_docs/planned/v0_35_0/m-fs-rename.md](../v0_35_0/m-fs-rename.md) — the `FS.renameFile` design (both operands confined); no overlap, its contract is a regression fixture here

**Search:** SimHash/neural searches over implemented/ and planned/ found no
existing or queued doc on deny-pattern ancestor protection (top hits unrelated);
no duplicate/coverage rejection applies.

## Verification Log

Every load-bearing claim in this doc, with how it was checked. Claims V1–V5 were
run live against the installed binary (`ailang version` → v0.52.5, commit 7200786,
built 2026-10-07) through the real `ailang run --policy` gate, restricted mode, in
temp sandboxes; triage's independent verification on origin/dev 658ff76a3 (2026-10-08)
is cited from the task brief.

| # | Claim | Evidence |
|---|---|---|
| V1 | Literal-pattern parent-rename bypass: under `fs_deny_write=[".claude/settings.json"]`, rename `.claude`→`tmp`, write `tmp/settings.json`, rename back → exit 0, protected file overwritten | Live repro this session (program + policy kept as test fixtures); matches issue #1569 and triage's 658ff76a3 run |
| V2 | The deny itself works: direct `writeFile(".claude/settings.json")` → `E_FS_PROTECTED` … matches fs_deny_write | Live control run this session |
| V3 | The `/**` workaround has the same hole for ancestors: under `["a/b/**"]`, `renameFile("a","t")` + write + rename back → exit 0, file overwritten (contrary to the issue's "not affected" claim, which holds only at/below the pattern dir) | Live repro this session |
| V4 | Mid-glob: under `["a/*/x.txt"]`, `renameFile("a/b","c")` + write + rename back → exit 0, file overwritten | Live repro this session |
| V5 | Base patterns follow the file: under `["*.txt"]`, `renameFile("a","t")` allowed, `writeFile("t/x.txt")` refused `E_FS_PROTECTED` — ancestor over-denial for base patterns is both unnecessary and wrong | Live probe this session |
| V6 | `.git` protection is component-based and already travels with renames; no ancestor rule needed for M6 | Read `UnderGitDir` (protect.go:63-71): matches a `.git` component anywhere in the path |
| V7 | policy-tool has no rename/remove op; the exploit reaches agent lanes via `ailang_run` | Enumerated `cliSchemas` (21 CLI ops, none rename/remove) + `Request` fields (path/module/query/content/old_text/new_text/package) in `internal/policytool/`; `.pi/extensions/ailang-exec.ts:332` — `ailang_run` submits to `ailang run --policy` |
| V8 | `MatchDenyWrite` has exactly one non-test caller (`Protection.Check`, protect.go:55); `Protection{}` is constructed in exactly two non-test places (effects fs_root.go:149, policytool fs_ops.go:110) — adding a sibling function touches no other consumer | `grep -rn "MatchDenyWrite(\|Protection{" internal/ cmd/` (non-test) |
| V9 | The structural-op call sites are exactly the six listed; writes/mkdirs are deliberately excluded | `grep -n "fsCheckMutation"` over internal/effects (every non-test hit mapped to the functions read in fs_dir.go / fs_root.go / fs_write.go / fs_bytes_result.go) |
| V10 | Rename/remove are the only in-process subtree moves: FS registers 22 ops, none create symlinks or hard links | Enumerated `RegisterOp("FS", …)` in internal/effects/fs.go:43-69 |
| V11 | Regression fixtures exist and assert what the doc says | Read: `TestFSRenameFile_Success` (fs_test.go:340, asserts publish semantics), `TestFSRenameFile_DirectoryWithinSandbox` (fs_test.go:374), `TestFSContainment_RenameDestination` (fs_containment_test.go:292), `TestFSDenyWrite_PatternsProtectInsideRoot` (fs_deny_write_test.go:15, asserts the E_FS_PROTECTED matrix), `TestFSCheckMutation_SharedMatcher` (fs_deny_write_test.go:115) |
| V12 | No new error code: `E_FS_PROTECTED` already exists in effects (fs_root.go:151/155, context.go) | `grep -rn "E_FS_PROTECTED" internal/` (non-test) |
| V13 | No planned/implemented doc covers this topic (duplicate gate) | `ailang docs search` over implemented/ (1159 docs) and planned/ (394 docs): no hit ≥0.75 relates to deny-pattern ancestor protection |
| V14 | Version targeting: current release v0.52.5 (std/VERSION), next patch folder `v0_52_6` | `std/VERSION`; create-doc script version detection |
| V15 | `dir/**` already matches `dir` itself, so renaming the pattern's own directory is refused today | Read `MatchDenyWrite` CutSuffix branch (protect.go:87-90): `key == dir` returns the pattern |

Known tooling friction observed while producing this doc (not a doc premise):
`create_planned_doc.sh` exits 1 before scaffolding whenever the related-doc
search finds no matches, because `merge_results`' grep returns 1 under `set -eo
pipefail`; the search and scaffold were run manually. Flagged to the human; no
skill file was modified.

## References

- **Issue:** #1569 (fs_deny_write bypass by parent rename — this doc's tracker; P0, area:runtime)
- **Prior art in repo:** #1559 (one folded matcher, fail-closed over-denial), #1554 (policy-tool write gate), M-EXECUTOR-POLICY-HARDENING M6/M7
- **Matcher:** `internal/fileguard/protect.go`; **enforcement paths:** `internal/effects/fs_root.go` (`fsCheckMutation`), `internal/policytool/fs_ops.go` (`protected`)
- **Policy grammar:** `internal/policy/resolve.go` (fs_deny_write validation), `docs/docs/guides/agent-tool-policy.md`
- [Design Axioms](/docs/references/axioms)

## Future Work

- Reject `**` in non-terminal pattern components at policy load (matcher already fails closed).
- A `sandbox-check --move <path>` mode that prints the `CheckMove` verdict, so operators can dry-run a policy against their lane's rename patterns (extends the M-FS-SANDBOX-DIAGNOSTICS tool).
- Consider the symlink write-through question (Non-Goals) as its own issue if the operator-planted-symlink threat model is ever in scope.

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08
