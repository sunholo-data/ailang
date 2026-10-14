# M-WORLD-RELEASE-PNG-ALLOWANCE — a path-scoped, content-verified PNG allowance in World's tracked-binary gate

**Status**: Planned — APPROVED by Mark 2026-10-14 (attended): allowance mechanism + caps 1 MiB/8 MiB ratified; quorum r3 waived. Sprint M-WORLD-PNG-ALLOWANCE.
**Target**: fleet-authored commit to `sunholo-data/ailang-world` (no AILANG release needed)
**Priority**: P1 — the last World v1.0.0 release row (150) is blocked on it, and the loop has idled 9 fires
**Estimated**: ~0.5 day, attended
**Dependencies**: None. D-WORLD-69 is RESOLVED (A); rows 26, 105 and 172 on its exception path have landed.
**Ticket**: `ci:world:tracked-binary-rejects-release-png` (filed `inbox_1791466110794_90c512a9`, iter-246; re-filed `inbox_1791881840796_78032a58`, iter-257; attended nudge `inbox_1791998663234_28670c30`, 2026-10-14)

## Prior-work check (2026-10-14)

Before this doc was written, I searched for any executor triage or design of the gate change:

| Where | Result |
|---|---|
| `ailang` `design_docs/`, `.ailang/state/sprints/` (grep `tracked-binary`, `release-png`, `img/v1`) | none |
| `ailang mission ticket open` | ticket listed open, item-blocking; no triage attached |
| Messages, all inboxes (`tracked-binary-rejects`, `release-png`) | 13 messages, all World filings, status reports or the attended nudge. **0 `harness-resolved` replies, 0 fleet-side triage.** |
| Coordinator tasks (`--remote gcp`) | none |
| PRs on both repos (`png`, `tracked-binary`, `image`) | `ailang-world#42` (the gate's origin), `#236` (row 150 parked); no gate change |
| World `design_docs/planned/w-marketing-capture.md` | the **row-150 capture** design. Its Non-goals say "no shared driver or verification-gate edits", so it designs the screenshots, not the gate |
| World row 173 | "fleet decides the asset/gate mechanism, World changes neither gate nor PNG deliverable" |

No overlapping doc exists. The related-doc search scored nothing above 0.31.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The binary decision currently depends on in-tree `.gitattributes` and an 8000-byte heuristic (V6, V16). A whole-blob NUL scan makes it a function of the blob bytes alone. |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect change |
| A4: Explicit Authority | +1 | Replaces an all-or-nothing rule with one named, path-scoped grant. Nothing ambient is added. |
| A5: Bounded Verification | +1 | Each allowed file is checked locally and finitely: chunk walk, CRCs and size caps |
| A6: Safe Concurrency | 0 | n/a |
| A7: Machines First | +1 | Every rejection prints the path and one reason token, so the loop's predicate can test it |
| A8: Minimal Syntax | 0 | n/a |
| A9: Cost Visibility | 0 | Repo-size cap is explicit, but it is not runtime cost |
| A10: Composability | 0 | n/a |
| A11: Structured Failure | +1 | Each failure has a distinct reason (`not-allowlisted`, `bad-signature`, `trailing-bytes`, …) instead of one blanket FATAL |
| A12: System Boundary | 0 | n/a |

**Net Score: +5** → **Move forward.** No −1 on A1/A3/A4/A7.

## Problem Statement

`ailang-world/scripts/verify_go.sh:212–246` is the tracked-binary hygiene gate added in `ailang-world#42` after SM.A committed a 15.7 MB Mach-O. It fails CI on **any** blob that git's `--numstat` marks binary (`-\t-`). It has no path or type exception.

World row 150 (the v1.0.0 release showcase, `w-marketing-capture.md`) requires six committed PNGs under `website/static/img/v1/`. On PR #236, CI run 37783874633 rejected two instrument PNGs. The gate can't tell a release screenshot from a compiled artifact, so row 150 can never land.

**Impact:**
- Row 150 is the last row on the D-WORLD-69 exception path (26 ✔ → 105 ✔ → 172 ✔ → 150). The World loop has recorded "fleet PNG ticket open" on 9 consecutive idle fires (iterations 257–265).
- The ticket has been filed twice. The first filing was marked read without a resolution, which counts as closed in the fleet queue, so it dropped out silently.

**Two latent defects found while scoping this.** The current gate's detector is git's binary classifier, and it can already be defeated with no allowance at all:
- **Attribute steering (V6).** One committed `.gitattributes` line `*.png diff` makes `--numstat` count a Mach-O renamed to `.png` as **334 text lines**, and the gate passes.
- **8000-byte prefix (V16; quorum r2, gemini-4-argon).** Git only looks for a NUL in the first 8000 bytes. A Mach-O with 8,100 ASCII bytes in front is counted as **335 text lines** and passes, even with the attribute source pinned.

Any allowance built on git's classifier inherits both holes. The fix below replaces the classifier for the **whole tree**, not only the new directory.

## Goals

**Primary Goal:** let World commit release PNGs in one fixed directory, and keep the gate's guarantee: no compiled or arbitrary binary payload anywhere in the tree.

**Success Metrics:**
- A valid Chrome-produced PNG under `website/static/img/v1/` passes the gate.
- Every adversarial fixture in the Testing Strategy (2–12) fails with its own reason.
- Today's tree (3470 files, 0 binaries) still passes with an unchanged count line.
- World's own unblock predicates flip: the `png` count in `verify_go.sh` goes above 0, and a `harness-resolved` reply arrives in `mission-world` for the signature.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Mechanism: path-scoped allowance in the gate, rather than Git LFS or external hosting | Sets how World ships every future image | human (recommended: allowance) | design | med |
| PNG validity = full chunk walk (signature, IHDR first, CRC on every chunk, chunk-type allowlist, IEND last, **0 trailing bytes**), not magic bytes only | Magic bytes alone admit a polyglot: a valid PNG with a Mach-O appended after IEND | agent (security floor) | design | low |
| Chunk-type allowlist `{IHDR, PLTE, IDAT, IEND, sRGB, gAMA, cHRM, iCCP, pHYs}`; text and EXIF chunks rejected | `tEXt`, `zTXt`, `iTXt` and `eXIf` can carry arbitrary payloads. Chrome's capture uses only IHDR/IDAT/IEND (V8). | agent | design | low |
| Size caps: 1 MiB per file, 8 MiB for the directory | The gate exists to stop permanent git bloat. Six 1440×1100 shots fit easily. | human (defaults proposed) | design | low |
| Detector = whole-blob NUL scan of every committed blob (`git ls-tree -r` → `git cat-file --batch` → python), replacing git's `--numstat` classifier | Closes both V6 and V16 for the **whole** tree. It does not depend on attributes, git version, or any prefix length. | agent (security floor) | design | low |
| Gate code moves into `scripts/tracked_binary_gate.sh`, called from `verify_go.sh`; its test is also invoked **from `verify_go.sh`**, so no `.github/workflows/` file changes | Testable in isolation. The fleet does not need workflow-file authority or a `workflow`-scoped token (quorum r2, oc-kimi-k3). | agent | design | low |

### Design Freeze

- [x] Mark: mechanism = path-scoped allowance (not LFS or external hosting)
- [x] Mark: caps 1 MiB/file, 8 MiB/dir (or other numbers)

## Solution Design

### Overview

Replace git's classifier with a content rule that nothing in the tree can steer: **a blob is binary if it contains a NUL byte anywhere**. Every Mach-O, ELF and PNG contains NULs, and the current tree has none (V17). Then split the binaries found into two sets:

- **Allowed:** path matches `^website/static/img/v1/[a-z0-9][a-z0-9-]*\.png$` **and** the committed blob passes the PNG validator.
- **Rejected:** everything else, exactly as today.

Both the scan and the validator read committed blobs through `git cat-file --batch`. They never read the working tree, because the gate's claim is about what is committed.

### Architecture

1. **`scripts/tracked_binary_gate.sh`.** This is the existing block, moved out of `verify_go.sh` and extended. **The allowance is declared at the call site, not inside the helper.** `verify_go.sh` calls `scripts/tracked_binary_gate.sh --allow-png 'website/static/img/v1/' --png-file-cap 1048576 --png-dir-cap 8388608`, and the helper has no default allowance (no flag means no image is allowed). The policy then stays readable in `verify_go.sh`, and World's existing unblock predicate, which greps that file's source for `png` (V14), flips with no change on World's side:
   - `git ls-tree -r -z HEAD` lists every blob with its path, and one `git cat-file --batch` stream feeds a `python3 -I` scanner that tests each **whole** blob for `\x00` (0.07s on today's 3471 blobs, V17). Submodule (`commit`) entries are skipped by type, and so are symlinks (mode 120000), which are path strings.
   - The non-vacuity check carries over: the blob count must be > 0, else FATAL. The scanner also counts bytes read and must equal the sum of the sizes `cat-file` reported.
   - For each binary path that matches the allowlist regex, pipe the blob into the validator. Collect `path reason` pairs for every failure.
   - A second non-vacuity line: `✓ N allowlisted PNG(s) validated, M bytes` whenever any binary matched the regex. The loop can then tell "validated 6" apart from "none present".
   - The output stays `✓ 0 binary blobs among N tracked files` when no binaries exist, so today's logs are byte-identical.
2. **Validator** (embedded `python3 -I` heredoc; the gate already uses `python3` at lines 24 and 350):
   - signature `89 50 4E 47 0D 0A 1A 0A`
   - first chunk IHDR, length 13
   - every chunk's CRC32 matches
   - every chunk type in the allowlist
   - exactly one IEND, last, with `offset == len(blob)`
   - per-file size cap

   It returns one reason token per failure.
3. **Directory cap.** The gate sums the sizes of all allowlisted blobs and fails `dir-cap-exceeded` past 8 MiB.
4. **Test script `scripts/test_tracked_binary_gate.sh`.** It builds throwaway repos in `mktemp -d` and runs the gate in each. `verify_go.sh` runs it immediately before the gate itself, and `verify_go.sh` already runs in CI (`ci.yml:266`). No workflow file is touched.

### Implementation Plan

**Phase 1: Detector hardening, no behaviour change on today's tree** (~1h)
- [ ] Extract lines 212–246 into `scripts/tracked_binary_gate.sh`; `verify_go.sh` calls it (no `--allow-png` yet, so behaviour is identical)
- [ ] Replace the `--numstat` classifier with the whole-blob NUL scan; today's tree must still read `✓ 0 binary blobs` (V17: 0 of 3471 blobs contain NUL)
- [ ] Mutation-test: swap the scanner back to git's `--numstat` classifier and confirm fixtures 9, 10 and 12 go green, which proves the fixtures can see both holes

**Phase 2: Allowance plus validator** (~2h)
- [ ] Path regex, python validator, directory cap, second non-vacuity line
- [ ] Add `--allow-png 'website/static/img/v1/' …` to the `verify_go.sh` call site; assert `grep -ci png scripts/verify_go.sh` ≥ 1
- [ ] `test_tracked_binary_gate.sh` with all fixtures, invoked from `verify_go.sh`

**Phase 3: Hand-off** (~0.5h)
- [ ] Fleet-authored commit or PR on `ailang-world` dev (the D-WORLD-DRIVER-1 pattern: the fleet commits and World's controller never edits); CI green
- [ ] Reply to `mission-world` with category `harness-resolved`, signature in title and payload, naming the commit and the allowance contract. Include World's two predicate reads so the loop can re-run them: the png count in `verify_go.sh` source, and the reply itself.
- [ ] Ack the two fleet-inbox filings only after the reply is sent

### Files to Modify/Create

- `ailang-world/scripts/tracked_binary_gate.sh` (new) — the detector, allowance and validator, ~110 LOC
- `ailang-world/scripts/test_tracked_binary_gate.sh` (new) — adversarial fixtures, ~120 LOC
- `ailang-world/scripts/verify_go.sh` — replace lines 212–246 with one call that declares the allowance (`--allow-png` + caps), about −30/+5 LOC
- `ailang-world/website/static/img/v1/README.md` (new) — states the allowance contract where an author will see it, ~15 LOC

## Examples

**Before** (PR #236, CI 37783874633):
```
verify_go.sh: FATAL: compiled/binary artifacts are tracked in git:
    design_docs/verification/world-row150-design/control1.png
```
**After**, valid capture in the allowed directory:
```
   ✓ 6 allowlisted PNG(s) validated, 1843210 bytes (website/static/img/v1/)
   ✓ 0 other binary blobs among 3482 tracked files
```
**After**, renamed Mach-O in the allowed directory:
```
verify_go.sh: FATAL: binary blob(s) rejected:
    website/static/img/v1/evil.png  bad-signature
```

## Success Criteria

- [ ] Today's `origin/dev` tree: the gate passes with the identical `✓ 0 binary blobs among N` line
- [ ] Fixture: a valid 3-chunk PNG in `img/v1/` passes
- [ ] All rejection fixtures below fail with the named reason
- [ ] With the scanner mutated back to `--numstat`, fixtures 9, 10 and 12 pass. This proves those fixtures are load-bearing.
- [ ] `test_tracked_binary_gate.sh` runs inside `verify_go.sh` in CI, green on dev; `git diff --stat` of the landing commit touches no `.github/` path
- [ ] `harness-resolved` reply is in `mission-world` for the signature
- [ ] `grep -ci png scripts/verify_go.sh` at the landed SHA ≥ 1. This is World's own predicate (V14), re-run against the commit, not assumed.

## Testing Strategy

The `test_tracked_binary_gate.sh` fixtures (temp repos, committed blobs):

| # | Fixture | Expected |
|---|---|---|
| 1 | valid PNG, `website/static/img/v1/a.png` | pass |
| 2 | valid PNG, `website/static/img/a.png` (outside the directory) | `not-allowlisted` |
| 3 | `/bin/ls` copied to `img/v1/evil.png` | `bad-signature` |
| 4 | valid PNG plus appended Mach-O after IEND | `trailing-bytes` |
| 5 | valid PNG with a `tEXt` chunk | `chunk-not-allowed:tEXt` |
| 6 | valid PNG with one flipped IDAT byte | `bad-crc` |
| 7 | 1.1 MiB valid PNG | `file-cap-exceeded` |
| 8 | nine 1 MiB-ish PNGs | `dir-cap-exceeded` |
| 9 | `.gitattributes` `*.png diff` plus Mach-O at `img/v1/x.png` | `bad-signature` (the hole is closed) |
| 10 | `.gitattributes` `*.bin diff` plus Mach-O at `tools/x.bin` | `not-allowlisted` (tree-wide hole closed) |
| 11 | empty repo | FATAL non-vacuity (existing control kept) |
| 12 | 8,100 ASCII bytes + `/bin/ls` at `tools/padded` | `not-allowlisted` (V16 prefix hole closed) |
| 13 | `workbench-light.png` … `phone.png` (the six capture names from `w-marketing-capture.md`), valid PNGs | pass. This proves the regex admits the real deliverable names. |

Manual check: run the gate on the two World control PNGs (`~/.ailang/state/world-iter246/chrome-control/control{1,2}.png`, copied into a temp repo under `img/v1/`). Both should pass.

## Verification Log

| ID | Claim | Method | Result |
|---|---|---|---|
| V1 | The gate lives at `verify_go.sh:212–246` | `git show origin/dev:scripts/verify_go.sh \| grep -n` | lines 212 (header) and 246 (✓ line) at `d0b32af` |
| V2 | The gate rejects every git-binary blob with no exception | read lines 232–245 | single awk `$1=="-" && $2=="-"`, no path filter |
| V3 | Current tree has 0 binaries | numstat over `origin/dev` | 3470 files, 0 `-\t-` rows |
| V4 | No `.gitattributes` exists in World today | `git show origin/dev:.gitattributes` | `fatal: path does not exist` |
| V5 | PNG and Mach-O are both git-binary by default | scratch repo | both `-\t-` |
| V6 | `.gitattributes` `*.png diff` defeats the current gate | scratch repo, `/bin/ls` as `evil.png` | numstat `334 0` (text): **gate passes a Mach-O** |
| V7 | `--attr-source=<empty tree>` (and `GIT_ATTR_SOURCE`) restores the classification | same repo, git 2.54.0 | both back to `-\t-` |
| V8 | Chrome's captures use only IHDR/IDAT/IEND, with 0 trailing bytes | chunk walk of World control1/2.png (16,271 B) | `['IDAT','IEND','IHDR']`, trailing 0 |
| V9 | `verify_go.sh` already depends on python3 | grep | lines 24, 350 |
| V10 | World CI has a test-script pattern to follow | `ci.yml` | `bash scripts/test_gate1_range_check.sh` at :339 (plus 3 siblings) |
| V11 | No fleet triage or design existed | see Prior-work check | 0 hits across 6 instruments |
| V12 | Fleet commits to World-owned verification are the ratified pattern | World ledger D-WORLD-DRIVER-1 | "fleet-authored commits … World's controller still never edits them" |
| V13 | D-WORLD-69 no longer gates row 150 | World ledger line 955 | RESOLVED A, 2026-10-09; 26/105/172 ✔ per iter-265 status |
| V14 | World's unblock predicate reads the **source** of `verify_go.sh` | World log `world-mission-log.md:1462,1502`: "`scripts/verify_go.sh` at HEAD has 0 png mentions"; `block-predicates.txt` "png allowance count" | it is a source grep, so the allowance must be declared in `verify_go.sh` (call site), not only inside the helper |
| V15 | World CI runner git version (was PENDING P1, closed after quorum r1) | `gh run view 37783874633 --log`, `actions/checkout` step | `git version 2.55.0` (2026-10-08). After r2 the design no longer needs `--attr-source`, so it no longer depends on git version. |
| V16 | Git's classifier misses NULs beyond 8000 bytes | scratch repo: 8,100×`A` + `/bin/ls` | `--attr-source=<empty>` numstat `335 0` (text) though the blob has 106,833 NULs: **current gate passes it** |
| V17 | Whole-blob NUL scan is safe on today's tree and cheap | `git ls-tree -r origin/dev` → `cat-file --batch` → python | 3471 blobs, 0 contain NUL, 0.07s. (numstat lists 3470: one empty blob has no numstat row.) |
| V18 | Row 150's real deliverable names match the allowlist regex | World `w-marketing-capture.md:90–95,118` | `workbench-light/dark.png`, `packet.png`, `graph.png`, `provenance.png`, `phone.png` under `website/static/img/v1/`; all match `^[a-z0-9][a-z0-9-]*\.png$`. The manifests and captions in the same directory are text (no NUL), so the gate never sees them. |
| V19 | The design needs no workflow-file edit | Architecture §4; `ci.yml:266` runs `./scripts/verify_go.sh` | test runs from `verify_go.sh`. D-WORLD-DRIVER-1's fleet-commit authority is therefore exercised only on `scripts/` plus one README, the same class as the driver-drift gate it already ratified. |

## Deferred Decisions

- Validator placement (embedded heredoc vs `scripts/check_png_blob.py`) — agent may choose
- Exact reason-token spelling — agent may choose; tokens must be distinct and stable once shipped
- Whether `img/v1/` later becomes a regex list (for example `img/v1.1/`) — agent may choose. Extending it is a reviewed one-line diff, never a wildcard.

## Non-Goals

- **Git LFS / external hosting.** Rejected: LFS adds a server dependency to a repo whose thesis is byte-exact reproducibility from a clone, and external hosting splits the website build from the tree it documents.
- **Other image types (JPEG, WebP, SVG-as-binary).** Row 150 needs PNG only. SVG is text and already passes.
- **Producing the screenshots.** That is row 150 / `w-marketing-capture.md`, which is World's job.
- **Reworking the fleet-ticket queue's read-equals-closed behaviour**, which dropped the first filing. It is real, but it is a separate signature.

## Quorum

Trigger 2 fires: this overrides shared machinery, a CI gate World depends on and is barred from editing. Triggers 1, 3 and 4 do not fire.

- **Round 1 (2026-10-14T17:29:25Z): BLOCKED, 3/3 reviewers.** gemini-4-argon and oc-glm-5-3: extracting the gate empties `verify_go.sh` of `png`, so World's source-grep predicate could never flip. They were right; fixed by declaring the allowance at the call site (V14). oc-kimi-k3: runner git version unmeasured; measured as 2.55.0 (V15). All objections were accepted, none argued.
- **Round 2 (2026-10-14T17:31:28Z): BLOCKED, 3/3.** gemini-4-argon: git only scans the first 8000 bytes. Reproduced (V16), a real pre-existing hole; fixed by the whole-blob NUL scan, which also makes `--attr-source` unnecessary. oc-glm-5-3: deliverable names unverified; verified (V18) and added as fixture 13. oc-kimi-k3: authority to edit `ci.yml` unestablished; the edit is removed (V19). **Re-quorum-once guardrail exhausted. A third round is the human's call.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Whole-blob scan reds a future legitimate non-PNG binary (e.g. a font) | Low | Correct by design: it must go through a reviewed allowance, same as PNG |
| Cap too small for retina captures | Low | World's shot list fixes device scale 1; caps are freeze items Mark can raise |
| Allowance grows into a dumping ground | Med | Single directory, regex reviewed in diff, directory cap, README contract |
| Valid-but-hostile pixel data | Low | Out of threat model: the gate guards against bloat and executables, not image content |

## Related Documents

- `ailang-world/design_docs/planned/w-marketing-capture.md` — the row-150 capture design this unblocks
- `ailang-world#42` — origin of the tracked-binary gate
- `ailang-world` World mission row 173 (harness blocker) and row 150 (release showcase)

---

**Document created**: 2026-10-14
**Last updated**: 2026-10-14
