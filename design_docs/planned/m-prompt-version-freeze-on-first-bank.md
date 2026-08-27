# M-PROMPT-VERSION-FREEZE-ON-FIRST-BANK: Freeze a prompt version's bytes at first banked use

**Status**: Planned
**Target**: v0.34.x
**Priority**: P1 (data-integrity — protects banked-baseline provenance, the substrate of every A/B and ELO claim)
**Estimated**: 3.5–4 days (4 milestones, each ≤1 sprint-day, each independently landable)
**Dependencies**: None
**Author**: design-doc-creator role, mission iteration 291 (2026-08-27, worktree at `70f4e0660`)
**Revision**: r2 — round-1 quorum verdict was **blocked** (3/3 reviewers present, three objections).
This revision adds dual-registry freeze propagation (obj. 2), bank-time byte evidence with a
declared trust-on-first-use residual for legacy rows (obj. 1), and the registry-intersection
verification row (obj. 3). See the [Quorum revision log](#quorum-revision-log-round-1--r2).
**Ruling implemented**: decision-ledger row **D-41**, answered by Mark 2026-08-27T07:26:21Z as
option **(c)**: a prompt version is **MUTABLE UNTIL FIRST BANKED USE, then frozen**. The ruling is
settled; this doc designs the mechanism only.

---

## Problem Statement

`prompts/versions.json` (59 versions at `70f4e0660`) records a `hash` per version, and the
eval harness verifies it at load time (`internal/eval_harness/prompt_loader.go:77-86`). But the
registry has **no representation of lifecycle state at all** — zero of the 59 versions carry any
`frozen` or `banked` field (V-A). So today the hash is simultaneously being used as two different
things:

1. A **change detector** for versions still under development — where regenerating it after an
   in-place edit is legitimate (the workflow `create_prompt_version.sh` explicitly instructs:
   "Update hash: shasum -a 256 ...").
2. A **provenance invariant** for versions that appear in banked eval baselines — where an
   in-place edit silently corrupts the meaning of every banked row that says `prompt_version: vX.Y.Z`.

D-41(c) rules that these are two lifecycle **states** of one version, not one policy:

- **NEVER-BANKED**: the `.md` file may be edited in place; the `hash` is a change detector,
  legitimately regenerated on edit.
- **BANKED** (used in ≥1 banked eval run): the bytes are **frozen**; the hash becomes a hard
  invariant; a content change must bump to a new version id.

Two facts make this urgent rather than theoretical:

- **1,284 of 17,343 repo-tracked baseline files under `eval_results/baselines/` attribute
  themselves to one of 18 distinct AILANG prompt-version ids** (plus 6,537 to `python`). Those
  attributions are only meaningful while the named bytes are recoverable (V-F, re-derived
  first-party below).
- **The agent-mode eval path performs NO hash verification at all**, and on any prompt-load error
  silently substitutes a hardcoded default prompt while banking `prompt_version: "default"`
  (`internal/eval_harness/langreg/ailang.go:32-38` → `internal/prompt/loader.go`, which never
  compares `Hash`). Even a perfect freeze record is unenforced on the path that runs most evals
  today. This is a new first-party finding of this iteration (Verification Log rows N-3/N-4) and
  is closed by Milestone 3.

The mechanism is **prospective**: the active version `v0.16.6` has **zero** banked uses in the
attributable corpus (V-F), so the in-place edit that opened D-41 was legitimate under the ruling —
its only defect was the stale hash.

---

## Goals

1. Make the BANKED/NEVER-BANKED state **representable** in the registry (schema addition), and
   **recorded** for all 59 existing versions via one mechanical, verifiable migration — applied
   to **BOTH** registry files: `prompts/versions.json` and its embedded-source mirror
   `cmd/ailang/prompts/versions.json` (agent mode reads the embedded registry FIRST, V-J/N-5; a
   marker present only in the source registry is invisible to every rig binary).
2. Enforce frozen-bytes immutability at **four layers**: load time (standard-mode loader),
   load time (agent-mode loader — currently unverified), **CI** (which alone can catch the
   "edit the file AND regenerate the hash" bypass), and **bank-time byte evidence**
   (`prompt_sha256` on new banked rows — the only layer that can catch a bytes-swap inside the
   first-bank PR itself, Q7).
3. Make the failure **teach**: the error a developer sees on editing a frozen prompt must state
   why it is frozen, show the banked evidence, and name the bump command.
4. Preserve the never-banked editing workflow byte-for-byte: `v0.16.6` (and future new versions)
   stay editable in place with hash regeneration, exactly as today.

## Non-Goals

- No retroactive repair of historical attribution. The 9,522 tracked baselines with no
  `prompt_version` field, and the fact that `python` has meant different bytes over time, are
  permanent losses; freezing prevents recurrence, it does not reconstruct the past.
- **No byte-fidelity guarantee for rows banked before M4 lands.** Rows that carry only a
  `prompt_version` id (all 17,343 existing rows, and any row banked by a pre-M4 harness binary)
  cannot prove WHICH bytes they measured; for those rows the first-bank commit is
  trust-on-first-use. This residual is bounded, declared (Q7), and **closed-ended** (r3): it
  covers only baseline files already present at the M4 cutoff commit. Every NEWLY ADDED row must
  carry a valid 64-hex `prompt_sha256` or CI fails, so a stale pre-M4 binary cannot enlarge the
  residual. Not silently implied away.
- No change to the devtools prompt registry (`internal/devtoolsprompt/loader.go`, a separate
  `versions.json`) in this doc — see Conflict Surface item 6 and Open Questions.
- No general embedded-mirror sync checker for *unfrozen* versions (see Conflict Surface item 4).
- No change to which runs get banked or how; this doc is about the registry, its loaders, and CI.

---

## Verification Log

Rows **V-A … V-H** were measured first-party by the mission controller at commit `70f4e0660`
(attribution: *verified by controller, iteration 291, commit 70f4e0660*). Rows **N-1 … N-8** were
measured first-party by this role in the same worktree on 2026-08-27. Row N-2 is a deliberate
re-derivation of V-F (load-bearing for Q2/Q4); it agreed exactly.

| # | Claim | Command | Observed output | Source |
|---|-------|---------|-----------------|--------|
| V-A | 59 versions in `prompts/versions.json`; 0 carry any `frozen`/`banked` field | `jq -r '[.versions\|to_entries[]\|select(.value\|has("frozen") or has("banked"))]\|length' prompts/versions.json` (and `.versions\|length`) | `0` / `59` | controller |
| V-B | Active version is `v0.16.6` | `jq -r .active prompts/versions.json` | `v0.16.6` | controller |
| V-C | Hash verification is load-time only, at `internal/eval_harness/prompt_loader.go:77-86`, skipped when `Hash == "PLACEHOLDER"`; no other enforcement point | (controller code reading) | as stated | controller |
| V-D | Banked results record `prompt_version` (`cmd/ailang/eval_suite_manifest.go:77`, `:133`; populated via `cmd/ailang/eval_benchmark.go:172` `loader.GetActiveVersionID()`) | (controller code reading) | as stated | controller |
| V-E | Banked baselines are repo-tracked: 17,343 files under `eval_results/baselines/` (`.gitignore:92-93` negates the `eval_results/` ignore). Positive control: `git check-ignore -v bin/ailang` → `.gitignore:46:/bin` | `git ls-files 'eval_results/baselines/*.json' \| wc -l` | `17343` | controller |
| V-F | Corpus distribution: 9,522 field-absent, 6,537 `python`, 18 distinct AILANG ids, **no v0.16.x at all** | (controller jq sweep) | as stated | controller |
| V-G | Observatory SQLite `eval_baselines` (7 columns, 1,020 rows) has **no** `prompt_version` column — the tracked corpus is the only attributable bank | (controller schema read) | as stated | controller |
| V-H | Tests that went red on the stale hash: `TestAILANGPromptLoading`, `TestPromptDisambiguation` (`cmd/ailang/eval_suite_prompt_test.go:34`, `:87`) | (controller) | as stated | controller |
| V-I | **Registry∩corpus intersection holds exactly** (grounds the 19/39/1 migration split, Q4): registry keys 59; distinct corpus `prompt_version` values 19; corpus ids PRESENT in registry 19; MISSING 0 (including `v0.6.5`, the 1-citation id). Positive control: `python` in registry → `True`; negative control: `v9.9.9-nonexistent` in registry → `False`. Arithmetic: 19 banked + 39 legacy + 1 mutable = 59 | (controller set-intersection over `prompts/versions.json` keys × N-2 corpus values, both controls in the same run) | `present 19 / missing 0`; controls `True`/`False` | verified by controller, iteration 291, commit `70f4e0660` |
| V-J | **Agent mode reads the EMBEDDED registry first**: `internal/prompt/loader.go:106-108` (`loadVersionsManifest`: `if embeddedPrompts != nil { fs.ReadFile(embeddedPrompts, "prompts/versions.json") }`), disk fallback only at `:120`; same embedded-first order for the `.md` read at `:62-74`. Mirror state today: `sha256 prompts/versions.json` = `97decb4d7dd3…ef06c99e` = `sha256 cmd/ailang/prompts/versions.json` — **byte-identical, no drift now; the hazard is prospective** | `shasum -a 256 prompts/versions.json cmd/ailang/prompts/versions.json` + code reading | identical digests `97decb4d…` | verified by controller, iteration 291, commit `70f4e0660` |
| N-1 | **0 of 59** versions in `prompts/versions.json` have `hash == "PLACEHOLDER"`. Positive control, same command shape, same file: **59 of 59** hashes match `^[0-9a-f]{64}$` | `jq '[.versions\|to_entries[]\|select(.value.hash=="PLACEHOLDER")]\|length' prompts/versions.json` ; control `jq '[.versions\|to_entries[]\|select(.value.hash\|test("^[0-9a-f]{64}$"))]\|length'` | `0` ; control `59` | this role |
| N-2 | Re-derivation of V-F over all 17,343 tracked files under `eval_results/baselines/`: `9522 ABSENT, 6537 python, 210 v0.3.21, 138 v0.5.0, 138 v0.4.8, 114 v0.6.2, 92 v0.6.6, 76 v0.4.1, 62 v0.4.7, 62 v0.11.4, 61 v0.7.4, 57 v0.9.0, 51 v0.12.1, 38×(v0.4.0,.2,.4,.5,.6), 32 v0.3.23, 1 v0.6.5` — sums to 17,343; **agrees exactly with V-F**. Also: neither `"default"` nor `"agent-prompt"` appears as a value | `git ls-files 'eval_results/baselines/*.json' \| xargs jq -r 'if has("prompt_version") then .prompt_version else "ABSENT" end' \| sort \| uniq -c \| sort -rn` | as quoted | this role |
| N-3 | The agent-mode prompt loader (`internal/prompt/loader.go`, 209 lines) **never compares the hash**: the only occurrence of `Hash` in the file is the struct field at line 23. Positive control, same grep, the standard-mode loader in the same repo: `grep -c "Hash" internal/eval_harness/prompt_loader.go` → **8** (includes the comparison at :77-86) | `grep -n "Hash" internal/prompt/loader.go` ; control as stated | `23:	Hash string ...` (only) ; control `8` | this role |
| N-4 | Agent mode attributes and falls back silently: `langreg/ailang.go:32-38` `LoadSyntaxRef` calls `promptpkg.LoadPromptWithVersion(version)` and on **any** error returns `(DefaultPrompt(), "default", nil)`; `langreg/python.go:31-39` has the identical pattern returning `("...", "default", nil)`. The result is banked via `result.PromptVersion` at `cmd/ailang/eval_benchmark_agent.go:315` (agent entry: `agent_runner_multi.go:182` → `GenerateAgentPromptsWithSystemPrompt`, `agent_prompt.go:404`) | `sed -n '29,38p' internal/eval_harness/langreg/ailang.go` (and python.go); `grep -n "PromptVersion" cmd/ailang/eval_benchmark_agent.go` | fallback branches as quoted; `:315 PromptVersion: result.PromptVersion` | this role |
| N-5 | `internal/prompt/loader.go` reads the **embedded** copy first and falls back to disk (`LoadPrompt` lines 62-77; `loadVersionsManifest` likewise), so agent-mode bytes come from the binary's build-time snapshot of `cmd/ailang/prompts/` (embedded via `//go:embed all:prompts` at `cmd/ailang/main.go:21-22`), while the standard-mode loader reads **disk only** via `l.rootDir` (V-C) | `sed -n '60,80p' internal/prompt/loader.go` ; `grep -n "go:embed" cmd/ailang/main.go` | embedded-first branches as quoted; `21://go:embed all:prompts` | this role |
| N-6 | The mirror `cmd/ailang/prompts/` currently agrees with `prompts/` for the active version (spot check) and contains 54 `v0*` files + its own `versions.json` | `diff <(shasum -a 256 prompts/v0.16.6.md) <(shasum -a 256 cmd/ailang/prompts/v0.16.6.md)` (hash fields only); `ls cmd/ailang/prompts \| grep -c '^v0'` | in sync; `54` | this role |
| N-7 | **No Makefile involvement in prompt sync**: `grep -c "prompts" Makefile` → **0** occurrences of the string `prompts` anywhere in `Makefile`. Positive control, same instrument, same file: `grep -c "test" Makefile` → **21**. The mirror is maintained by manual copy (e.g. `.claude/rules/api-server.md:28` checklist "same (embedded copy)") | `grep -c "prompts" Makefile` ; control `grep -c "test" Makefile` | `0` ; control `21` | this role |
| N-8 | `create_prompt_version.sh` (`.claude/skills/prompt-manager/scripts/`, 105 lines) computes the hash **at creation** (line ~72 `shasum -a 256`), writes the registry entry, sets `.active`, and its printed "Next steps" instruct a **manual hash update after editing** (step 3). It only ever creates NEW versions (errors if the id or file exists), so it never touches a frozen entry | `sed -n '1,105p' .claude/skills/prompt-manager/scripts/create_prompt_version.sh` | as quoted | this role |
| N-9 | Version creation dates: `v0.16.6` created `2026-08-12`; `v0.16.5` (2026-08-07) carries the note "v0.16.5 remains served byte-identical for pinned eval baselines" — i.e. versions **without** corpus attribution are nevertheless relied upon as pinned (evidence for the legacy-freeze rule in Q2/Q4) | `jq -r '.versions["v0.16.6"], .versions["v0.16.5"].notes' prompts/versions.json` | as quoted | this role |
| N-10 | **No baseline-JSON reader uses strict decoding**, so M4's additive `prompt_sha256` cannot break existing readers (grounds Conflict Surface item 10). `DisallowUnknownFields` occurs **0** times across `cmd/ internal/` and **0** repo-wide; the three named readers (`cmd/ailang/eval_elo.go`, `eval_saturation.go`, `eval_confidence.go`) are 0 each. Positive control per the reviewer's own prescription (`DisallowUnknownFields` is a `Decoder` method, not an `Unmarshal` option): `json.NewDecoder` occurs **10** times in `cmd/ailang/` and **122** repo-wide, so decoder usage exists and the instrument would have found a strict one | `grep -rn 'DisallowUnknownFields' --include='*.go' cmd/ internal/` with control `grep -rn 'json.NewDecoder' --include='*.go' cmd/ailang/` (scopes asserted with `test -d`) | `0` strict-decoder hits; controls `10` / `122` | verified by controller, iteration 291, commit `70f4e0660` |

---

## Design Questions — explicit answers

### Q1. Where is "frozen" RECORDED?

**Both: a recorded marker in `prompts/versions.json`, audited by a CI check that re-derives the
predicate from the tracked corpus.**

- **Recorded** (authoritative for enforcement): a `frozen` object on the version entry (schema
  below). This is what both loaders and the developer-facing error read. *Cheap*: one JSON field,
  zero extra I/O at load time — the loaders already parse this file.
- **Derived** (auditor): CI recomputes `banked(V)` from the 17,343 tracked files under
  `eval_results/baselines/` (one `git ls-files | xargs jq` pass, ~seconds; measured in N-2) and
  fails if any corpus-evidenced version lacks a marker. *Honest*: the record cannot silently
  drift below reality, because every PR re-checks it against the corpus that ships in the same
  repo (V-E — the corpus being repo-tracked is what makes this possible; the observatory DB
  cannot serve this role, V-G).

A purely derived predicate was rejected because both loaders would need the 17k-file corpus at
runtime (agent-mode loads happen on eval rigs from an embedded FS, with no corpus present). A
purely recorded one was rejected because nothing would catch a marker that was never written.

### Q2. What is the exact freeze predicate, given 9,522 attribution-less baselines?

**Corpus predicate**: `banked(V)` ≡ at least one tracked file under `eval_results/baselines/`
(scope: `git ls-files 'eval_results/baselines/*.json'`) has `.prompt_version == V`.

An attribution-less baseline (9,522 of 17,343, N-2) **cannot count as evidence for any specific
version** — it names none. So the corpus predicate errs toward **under-freezing**: a version that
was really used before `prompt_version` was recorded stays mutable under the pure predicate.

**That is the unsafe direction.** For reproducibility the fail-safe direction is
**over-freezing**: the cost of a wrongly-frozen version is one version bump (~1 minute with the
existing `create_prompt_version.sh`, N-8); the cost of a wrongly-mutable version is silent,
later-undetectable corruption of banked provenance. Therefore the predicate is compensated at
migration time (Q4) by a second freeze reason:

- `reason: "banked"` — corpus-evidenced (the 18 AILANG ids + `python`, N-2).
- `reason: "legacy"` — no corpus evidence, but created before this mechanism landed, when the
  9,522 attribution-less rows were being banked; absence of evidence is not evidence of absence
  here. N-9 shows this is not hypothetical: `v0.16.5` has zero corpus rows yet its own registry
  note declares it "served byte-identical for pinned eval baselines".

Going forward (post-migration), only `reason: "banked"` freezes are created, because every new
banked row carries `prompt_version` (V-D) — the incompleteness is a property of the historical
corpus, not of new data. Note this id-level predicate identifies **which version** was banked but
not **which bytes**; the byte-level clause is added in Q7 (post-M4 rows also carry
`prompt_sha256`, and the predicate then additionally requires agreement between the frozen hash
and every byte-carrying row).

### Q3. WHO freezes, and WHEN?

**At PR time, by an explicit command, enforced by CI. The eval harness never writes the
registry.**

- New subcommand: `ailang prompt freeze <version>` — computes the corpus evidence at HEAD, writes
  the `frozen` marker (with count + example path) to **BOTH registry files**
  (`prompts/versions.json` AND `cmd/ailang/prompts/versions.json` — agent mode reads the embedded
  registry first, V-J, so a single-registry write would leave every binary built from that tree
  perceiving the version as never-banked and bypass M3's enforcement exactly on the eval rigs),
  and refuses if the version's current file bytes do not match its recorded hash (you may not
  freeze a lie).
- `ailang prompt freeze --check` — the derivation audit (Q1) plus integrity checks; exits
  non-zero with the list of violations. Wired into `make ci` via a new `check-prompt-freeze`
  target (the Makefile currently references prompts nowhere, N-7, so this is additive).
- **When**: a version becomes banked only when baseline JSONs naming it are committed under
  `eval_results/baselines/` — and that is a repo event (a PR), regardless of which machine ran
  the eval. This dissolves the harness-machine ≠ repo-machine problem: the rig banks locally, the
  bank lands in the repo via the existing sync, and the first PR containing rows for version V
  goes red on `--check` until someone runs `ailang prompt freeze V` in that PR.

Bank-time freezing (harness writes the flag) was rejected: the harness often runs against an
embedded snapshot on another machine (N-5), a write from there is unlandable, and V-G shows the
only authoritative bank is the repo itself.

### Q4. What happens to the versions already banked today?

One mechanical migration, generated and verified by command, landed as a single commit in M1.

Generator for the banked set (this is the same instrument as N-2):

```bash
git ls-files 'eval_results/baselines/*.json' \
  | xargs jq -r 'select(has("prompt_version")) | .prompt_version' \
  | sort | uniq -c | sort -rn
```

Migration rule, applied by `ailang prompt freeze --migrate`:

| Set | Count (of 59 in `prompts/versions.json`) | Action |
|-----|------------------------------------------|--------|
| Corpus-evidenced: 18 AILANG ids + `python` (N-2) | 19 | `frozen` with `reason: "banked"`, `evidence_count` from the generator, `evidence_example` = first matching tracked path |
| Everything else except the active version — 35 AILANG ids (incl. `v0.16.0`–`v0.16.5`) + `go`, `javascript`, `moonbit`, `aver` | 39 | `frozen` with `reason: "legacy"` (fail-safe direction, Q2; concrete justification N-9) |
| `v0.16.6` (active, zero corpus evidence, V-F/V-B) | 1 | **left mutable** — the D-41 ruling itself classified its in-place edit as legitimate |

19 + 39 + 1 = 59 (checks against V-A; and V-I verifies the intersection this split rests on —
all 19 corpus-evidenced ids exist as registry keys, 0 missing, with positive/negative controls).
The migration is applied to **BOTH** registry files (Q3/V-J); today they are byte-identical
(V-J), so the dual write starts from an agreed base. Verifiable: after migration, `--check` must
be green, and — against **each** of the two registry files —
`jq '[.versions[]|select(.frozen.reason=="banked")]|length'` → 19,
`...=="legacy"...` → 39, `[.versions|to_entries[]|select(.value|has("frozen")|not)]|length` → 1,
plus `diff <(jq -S .versions prompts/versions.json) <(jq -S .versions cmd/ailang/prompts/versions.json)`
→ empty.

### Q5. What is the ERROR on editing a frozen prompt?

Today (`prompt_loader.go:81-87`): `hash mismatch for "vX": expected …, got … (file may have been
modified)` — it does not say whether the modification was legitimate, or what to do.

After M1, the standard-mode loader distinguishes the two states:

**Frozen** (hash mismatch, `frozen` marker present):

```
prompt version "v0.3.21" is FROZEN: it is cited by 210 banked baseline files under
eval_results/baselines/ (e.g. eval_results/baselines/<example>.json; frozen 2026-08-27,
reason: banked — decision D-41c). Its bytes are immutable: editing prompts/v0.3.21.md in
place would silently change what those baselines measured.
To change the teaching prompt, create a NEW version instead:
  .claude/skills/prompt-manager/scripts/create_prompt_version.sh v0.3.22 v0.3.21 "<why>"
(expected sha256 db0a593fd2a7…, got <actual>…)
```

**Never-banked** (hash mismatch, no marker):

```
hash for "v0.16.6" is stale. This version has no banked uses, so in-place editing is
allowed (D-41c) — regenerate the change-detector hash:
  shasum -a 256 prompts/v0.16.6.md   # then update the "hash" field in prompts/versions.json
```

M3 emits the same frozen-error text from the agent-mode loader (`internal/prompt`), so both eval
modes teach identically. The `evidence_count`/`evidence_example` are read from the recorded
marker (snapshot at freeze time), not recomputed at load — load-time cheapness per Q1.

### Q6. Does the PLACEHOLDER escape hatch need closing for frozen versions?

**Yes for frozen, no for never-banked — and it costs nothing today.** Measured (N-1): **0 of 59**
versions in `prompts/versions.json` have `hash == "PLACEHOLDER"` (control: 59/59 are 64-hex). A
frozen version with a PLACEHOLDER hash would be a freeze with no invariant, so:

- Loaders: `frozen` marker present AND hash not `^[0-9a-f]{64}$` → hard error ("frozen version
  with unenforceable hash — refuse to load"), instead of the V-C skip.
- CI `--check`: same condition is a violation; `ailang prompt freeze` refuses to create such a
  marker in the first place.
- Never-banked versions keep the hatch exactly as in V-C (zero current users, but it is part of
  the documented dev workflow and this doc's mandate is prospective enforcement, not workflow
  removal).

### Q7 (added in r2). Can the mechanism prove WHICH bytes a banked row measured?

Round-1 review (gpt5-6-sol) identified a real hole: **L3(c) protects only versions already
frozen at the merge-base.** In the first PR that banks rows for an unfrozen version, an author
can run evals against bytes A, edit the prompt to bytes B, regenerate the registry hash, add the
frozen marker, and commit all of it — every layer as designed in r1 sees self-consistent bytes B
plus a marker and passes, yet the banked rows measured bytes A. Because rows record only a
`prompt_version` **id**, no layer can tell which bytes were used. This strikes the core of
D-41(c), which freezes "the bytes at first banked use".

**Chosen direction: (A) — make the banked row carry the byte evidence — with the residual for
legacy rows declared honestly rather than papered over.**

Mechanism (Milestone 4):

- **Bank-time**: new optional field `prompt_sha256` on the banked result/manifest schema
  (alongside `prompt_version`, V-D sites: `cmd/ailang/eval_suite_manifest.go` and both mode
  writers). It is the sha256 of the **bytes actually served** to the model — computed from the
  **fully evaluated prompt string immediately prior to model dispatch (post-variable
  substitution, if any)** — not merely the content the loader returned, and NOT copied from the
  registry's `hash` field (the registry field
  can be stale relative to served bytes; the D-41 incident was exactly that state).
- **Predicate extension** (`ailang prompt freeze <V>` and `--check`): collect the distinct
  `prompt_sha256` values among tracked rows citing V. If ≥1 row carries the field, ALL of them
  must agree with each other AND with the registry hash of the frozen entry; any disagreement is
  red. Replaying the reviewer's scenario: the banked rows carry `sha(A)`, the registry carries
  `hash(B)` — `--check` goes red on the first-bank PR itself.
- **Historical rows vs newly added rows — the legacy exception is defined by the MERGE-BASE, not
  by field absence** (r3, gpt5-6-sol's round-2 `proposed_fix` applied verbatim). r2 defined the
  exception as "row lacks the field", which silently treats *newly added, unverifiable* rows as
  legacy and lets a stale-binary first-bank PR pass the integrity gate. Corrected: `--check` uses
  the merge-base to distinguish historical rows from newly added ones — **field absence may warn
  only for baseline files already present at the M4 cutoff commit, while every newly added banked
  AILANG/control row must contain a valid 64-hex `prompt_sha256`. For any newly frozen version,
  require at least one newly added row with byte evidence and require all newly added rows citing
  it to agree with the frozen registry hash; otherwise fail CI.** So a row banked by a stale
  pre-M4 binary and added in this PR is REJECTED rather than excused.
- **Genuinely historical rows** (the 17,343 files present at the M4 cutoff commit, N-2): id-only
  evidence, warning not failure; for those the first-bank commit is **trust-on-first-use**. This
  is stated in Non-Goals and tracked as Open Question 4, and it is now a CLOSED set — it can
  never grow, because every newly added row must carry byte evidence.

Why (A) over (B) (declare-only): the ruling's subject is byte identity, and the evidence is
available for free at bank time — the loaders already hold the served bytes (L1 already computes
their sha256 for verification). Cost: one additive optional field on banked rows (old readers
ignore unknown JSON fields, same tolerance argument as the registry schema), one cross-check in
`--check`. It does nothing for the 9,522 attribution-less rows or the existing 1,284+6,537
id-attributed rows — nothing can — which is precisely the residual that (B)'s honesty is applied
to. An unearned guarantee would be worse than the gap; the doc therefore claims byte-fidelity
**only** for post-M4 rows.

---

## Solution Design

### Schema (versions.json `schema_version` 1.0 → 1.1)

```jsonc
"v0.3.21": {
  "file": "prompts/v0.3.21.md",
  "hash": "db0a593fd2a7…",            // unchanged meaning; becomes a HARD invariant when frozen
  "frozen": {                          // ABSENT ⇒ never-banked ⇒ mutable
    "at": "2026-08-27",
    "reason": "banked",               // "banked" | "legacy" | "manual"
    "evidence_count": 210,             // tracked files under eval_results/baselines/ citing this id, at freeze time
    "evidence_example": "eval_results/baselines/<file>.json"   // "" for reason:"legacy"
  },
  ...
}
```

Both loader structs (`internal/eval_harness/prompt_loader.go` `PromptVersion`,
`internal/prompt/loader.go` `VersionMetadata`) gain `Frozen *FrozenMarker
\`json:"frozen,omitempty"\``. Go's `json.Unmarshal` ignores unknown fields, so **binaries built
before this change parse the new registry without error** — they simply do not enforce freezing;
the enforcement floor for stale binaries is the CI gate.

### Enforcement layers

| Layer | Site | Catches | Cannot catch |
|-------|------|---------|--------------|
| L1 standard-mode load | `internal/eval_harness/prompt_loader.go` `LoadPrompt` (extends the existing V-C check) | file edited, hash not regenerated → **teaching frozen error** (Q5) | file edited AND hash regenerated |
| L2 agent-mode load (M3) | `internal/prompt/loader.go` `LoadPrompt` — for **frozen** versions only, verify sha256 of the bytes actually read (embedded or disk) against `hash`; plus removal of the silent `DefaultPrompt`/`"default"` fallback in `langreg/ailang.go` and `langreg/python.go` (N-4) | same as L1, on the path that had **zero** verification (N-3); also stops `"default"` ever being banked as an attribution | same as L1 |
| L3 CI (M2) | `ailang prompt freeze --check` via `make check-prompt-freeze`, in `make ci` | (a) corpus-evidenced version without marker; (b) frozen version whose file bytes ≠ hash, or hash not 64-hex; (c) **frozen-entry immutability vs merge-base**: for every version frozen at `origin/dev` merge-base, its `file`/`hash` fields and its `.md` bytes must be unchanged in the PR — this is the only layer that catches the edit+regen bypass; (d) mirror agreement for frozen ids, covering **BOTH** the `.md` bytes (`cmd/ailang/prompts/<file>` == `prompts/<file>`) **AND the mirror registry itself**: the frozen entry in `cmd/ailang/prompts/versions.json` (`file`, `hash`, `frozen`) must be identical to the source entry — the registry file is what carries the markers, and agent mode reads the embedded registry first (V-J), so a `.md`-only mirror check would leave the enforcement bits themselves free to drift (the mirror is manually synced, N-6/N-7/V-J) | runs only where CI runs — a rig with a stale binary and no CI is covered by L1/L2 at next load; and L3(c) alone cannot see a bytes-swap inside the first-bank PR (nothing frozen at merge-base) — that is L4's job |
| L4 bank-time byte evidence (M4) | `prompt_sha256` of the **served bytes** written on every new banked row; `--check` requires all byte-carrying rows citing a frozen version to agree with each other and with the frozen hash (Q7) | the first-bank bytes-swap (rows measured bytes A, registry frozen at bytes B) — the only layer that can | **newly added** rows must carry a valid 64-hex `prompt_sha256` or CI fails (r3); the warning-not-failure path applies ONLY to baseline files already present at the M4 cutoff commit, a set that cannot grow — TOFU residual declared in Non-Goals |

Note on L2 scope: hash verification in `internal/prompt` is applied **only to frozen versions**.
The disk-fallback path exists to allow editing the active (mutable) version without rebuilding
(loader comment, N-5); checking unfrozen versions there would break that workflow for no
provenance gain.

A useful emergent property: because L3(c) makes frozen disk bytes immutable on `dev`, every
binary built after a version freezes embeds **identical** bytes for it — embedded-mirror drift
becomes structurally impossible for frozen versions, without any sync tooling.

---

## Conflict Surface

1. **PLACEHOLDER hatch (V-C)** — narrowed, not removed: banned for frozen versions (Q6), kept
   for never-banked. Measured impact today: zero versions use it (N-1).
2. **`active: "latest"` resolution** (`prompt_loader.go` `findLatestVersion`,
   `internal/prompt` `"latest"` handling) — untouched. A frozen version MAY be active: freezing
   blocks *edits*, not *use* (serving frozen bytes is exactly the point). The migration leaves
   `active: "v0.16.6"` (a literal, V-B) mutable; nothing in the resolution path reads `frozen`.
3. **Non-AILANG control entries** (`python`, `go`, `javascript`, `moonbit`, `aver`) — included in
   the freeze. `python` has 6,537 banked citations (N-2) and is the control arm of the
   local-vs-frontier thesis, so its comparability matters *more*, not less. Consequence: a future
   `python.md` change requires a new key (convention: `python-v2`) **and** a one-line change in
   `langreg/python.go:34`, which hardcodes `LoadPrompt("python")` (N-4). Historical `python`
   attribution is already lossy (the bytes changed over the corpus's lifetime); that is
   unrepairable and out of scope (Non-Goals).
4. **Embedded mirror `cmd/ailang/prompts/`** — read FIRST by agent mode, **including the
   registry file itself** (`loadVersionsManifest` embedded-first, V-J/N-5), maintained by manual
   copy with no Makefile target (N-7). Consequences now designed in, not just noted: the
   migration and every `freeze` write go to BOTH registries (Q3/Q4), and L3(d) checks frozen-id
   agreement for `.md` bytes AND registry entries (a marker present only in
   `prompts/versions.json` would leave every binary built from that tree blind to the freeze —
   M3's enforcement bypassed exactly on the rigs). The two registry files are byte-identical
   today (V-J), so both the dual write and the check start green. General mirror-sync checking
   for mutable versions remains an open question (below).
5. **Silent-fallback removal** (`langreg/ailang.go:35`, `langreg/python.go:35`) — behavior
   change: an eval run whose prompt fails to load now **aborts loudly** instead of running with a
   hardcoded default and banking `prompt_version: "default"`. N-2 shows no `"default"` rows exist
   in the tracked corpus, so no existing data depends on the fallback; CLAUDE.md Principle 2
   (no silent fallbacks on data-integrity paths) mandates the removal.
6. **Devtools prompt registry** (`internal/devtoolsprompt/loader.go`, separate `versions.json`)
   — deliberately untouched: different registry, different attribution field, and D-41 was opened
   about the teaching-prompt registry. Flagged in Open Questions as a candidate for the same
   mechanism.
7. **`create_prompt_version.sh`** (N-8) — no functional change required: it only creates NEW
   versions (never-banked by construction, id/file collision → error) and its manual
   "update hash after edit" step remains legal exactly while the version is unbanked. M2 adds one
   sentence to its printed "Next steps" naming the freeze rule, so the workflow teaches itself.
8. **Existing tests (V-H)** — `TestAILANGPromptLoading` / `TestPromptDisambiguation` load the
   active version, which stays mutable with a valid hash; they are unaffected. New tests are
   added per the Mutation Table, not by editing these.
9. **Stale binaries** — parse the new field harmlessly (unknown-field tolerance, Schema section)
   but do not enforce; enforcement floor is CI (L3). This mirrors the repo's standing posture
   that `~/go/bin/ailang` drifts by design.
10. **Banked-result schema addition** (`prompt_sha256`, M4/Q7) — additive optional JSON field on
   the manifest/result structs (V-D sites). Existing readers of baseline JSON (eval-elo,
   eval-paired, dashboards) unmarshal into structs that ignore unknown fields, so old tooling is
   unaffected; new `--check` logic treats field-absent rows as id-only evidence (warning, never
   failure). No existing row is rewritten — re-banking history is prohibited by standing policy.

---

## Acceptance Criteria

Each AC names its command and what would still pass if the claimed mechanism were absent
(the "hollow-pass check").

- **AC1 (M1, migration correctness)**: after the migration commit,
  `jq '[.versions[]|select(.frozen.reason=="banked")]|length' prompts/versions.json` → `19`;
  `…"legacy"…` → `39`; `jq '[.versions|to_entries[]|select(.value|has("frozen")|not)]|length'`
  → `1` (and that one is `v0.16.6`). *Hollow-pass check*: a migration that marked everything
  frozen would fail the third count; one that only marked corpus-evidenced ids would fail the
  second.
- **AC2 (M1, frozen teaching error)**: in a test fixture (or on a scratch copy of the repo —
  never the working tree), append one byte to the frozen fixture's `.md`, then
  `loader.LoadPrompt("<frozen-id>")` must return an error whose text contains ALL of: `FROZEN`,
  the `evidence_count`, and `create_prompt_version.sh`. *Hollow-pass check*: the pre-existing
  V-C hash check also errors on this input — asserting a bare error would pass without any
  freeze code. The assertion is on the NEW message content, which only the frozen branch emits.
- **AC3 (M1, mutability preserved)**: same fixture, a version WITHOUT a marker: tamper +
  regenerate hash → `LoadPrompt` succeeds; tamper without regenerating → error containing
  `not yet banked` / `in-place editing is allowed`. *Hollow-pass check*: a freeze-everything
  mutation fails the first half; unchanged code fails the second half's message assertion.
- **AC4 (M2, edit+regen bypass)**: on a branch, edit a frozen `.md` AND update its `hash` field
  to the new sha256 → `ailang prompt freeze --check` (with merge-base = unmodified `dev`) exits
  non-zero naming the version and `immutability`. *Hollow-pass check*: L3(b) alone (file-vs-hash
  agreement) stays GREEN on this input — this AC is specifically the check that only the
  merge-base comparison can fail, and it must be demonstrated red.
- **AC5 (M2, derivation audit)**: on a branch, delete the `frozen` marker from `v0.3.21`
  (210 corpus citations, N-2) → `--check` exits non-zero naming `v0.3.21` as corpus-evidenced
  but unmarked. On unmodified post-migration `dev`, `--check` exits 0. *Hollow-pass check*: a
  `--check` that only validated hashes would stay green on the deleted marker.
- **AC6 (M2, frozen PLACEHOLDER ban)**: on a branch, set a frozen version's hash to
  `"PLACEHOLDER"` → `--check` non-zero; and `LoadPrompt` on that fixture errors rather than
  skipping. *Hollow-pass check*: under V-C semantics alone, PLACEHOLDER silently skips
  verification and loads — that is the behavior this AC proves is gone for frozen entries.
- **AC7 (M3, agent-mode hole closed)**: unit test with embedded FS pointed at a fixture whose
  frozen `.md` bytes disagree with its manifest hash: `LoadSyntaxRef("ailang", ...)` (via
  `langreg`) returns a non-nil error whose text contains `FROZEN`, and the returned
  `versionUsed` is NOT `"default"`. *Hollow-pass check*: today's code returns
  `(DefaultPrompt(), "default", nil)` on this exact input (N-4) — asserting only "no panic" or
  "some prompt returned" would pass on unmodified code; the assertions are the error and the
  absence of the `"default"` attribution.
- **AC8 (M3, mirror agreement)**: on a branch, make `cmd/ailang/prompts/<frozen>.md` differ from
  `prompts/<frozen>.md` → `--check` non-zero naming the mirror path. *Hollow-pass check*: no
  current instrument sees this at all (N-7: no Makefile prompt tooling exists to catch it).
- **AC9 (M2, mirror REGISTRY agreement)**: on a branch, leave every `.md` untouched and alter
  ONLY the mirror registry — delete (or change the `hash` of) one frozen entry in
  `cmd/ailang/prompts/versions.json` → `--check` non-zero naming
  `cmd/ailang/prompts/versions.json` and the version id. Additionally, AC1's post-migration jq
  counts (19/39/1) must hold against the MIRROR registry file too, and
  `diff <(jq -S .versions prompts/versions.json) <(jq -S .versions cmd/ailang/prompts/versions.json)`
  must be empty. *Hollow-pass check*: a `.md`-only mirror check (AC8's instrument) stays GREEN on
  this input — the `.md` bytes are identical by construction; only a registry-entry comparison
  can go red, and the registry file is the one that carries the freeze markers agent mode
  actually reads (V-J).
- **AC10 (M4, first-bank bytes-swap caught)**: fixture replaying the reviewer scenario — tracked
  rows citing version V with `prompt_sha256 = sha(A)`, registry entry for V frozen with
  `hash = sha(B)`, file bytes = B (self-consistent at HEAD, nothing frozen at merge-base) →
  `--check` exits non-zero naming V and the byte disagreement. *Hollow-pass check*: L3(a)(b)(c)
  ALL stay green on this input by construction (marker present, hash matches file, merge-base has
  no frozen entry to diff) — this AC can only be turned red by the row-hash cross-check, which is
  the mechanism under test.
  **AC10(b) (r3, stale-binary fixture — gpt5-6-sol's round-2 `proposed_fix`)**: a NEWLY ADDED
  baseline row (absent at merge-base) citing a newly frozen version and carrying NO
  `prompt_sha256` → `--check` exits non-zero naming the row's path and `missing byte evidence`.
  *Hollow-pass check*: under r2's field-absence rule this input was WARNED-and-passed, which is
  exactly the bypass; and a `--check` that keyed on field absence alone rather than on merge-base
  membership stays green on it. The historical arm must pass in the same test: an unchanged
  pre-cutoff fieldless row warns and does not fail.
- **AC11 (M4, served-bytes provenance)**: unit test on the bank path with a fixture registry
  entry that is unfrozen with `hash: "PLACEHOLDER"` (the one state where the loader serves bytes
  WITHOUT any hash agreement, V-C): the banked row's `prompt_sha256` must equal sha256 of the
  **served file bytes**. *Hollow-pass check*: an implementation that copies the registry's `hash`
  field instead of hashing the served content would write the literal `"PLACEHOLDER"` here —
  the assertion on a real 64-hex digest of the fixture bytes can only be satisfied by hashing
  what was actually served (this distinction is the D-41 incident class itself: registry field
  stale relative to served bytes).

None of these commands exists on unmodified `dev` today (the subcommand and tests are new), so no
AC is red-on-dev in the G4 sense; AC1/AC5/AC9's green states are defined post-migration-commit
and that ordering is explicit in M1/M2. No AC claims byte-fidelity for rows banked before M4
(Non-Goals; Q7).

---

## Mutation Table

| # | Mutation (introduced defect) | Killed by | Why the observable is downstream & unique |
|---|------------------------------|-----------|-------------------------------------------|
| 1 | L1 ignores the `frozen` field (freeze branch deleted) | AC2 | The asserted strings (`FROZEN`, evidence count, bump command) are emitted only by the frozen branch; the surviving V-C error cannot produce them |
| 2 | L1 treats every version as frozen (inverted predicate) | AC3 | A frozen-everything loader rejects the legitimate tamper+regen edit of an unmarked version; AC3's success half fails |
| 3 | `--check` skips corpus derivation (validates hashes only) | AC5 | The deleted-marker input has a self-consistent hash — only the derivation can turn it red |
| 4 | `--check` compares file-vs-hash but not vs merge-base | AC4 | The edit+regen input is self-consistent at HEAD by construction; only the merge-base diff sees it |
| 5 | Frozen + `PLACEHOLDER` still skips verification (V-C semantics retained) | AC6 | Load succeeding on a frozen placeholder is exactly the pre-change behavior; the AC asserts the refusal |
| 6 | M3 hash check added but silent `"default"` fallback retained in `LoadSyntaxRef` | AC7 | The fallback converts the new error back into `(default-prompt, "default", nil)`; AC7 asserts on the returned error AND on `versionUsed != "default"` — a value only the fallback path produces |
| 7 | M3 verifies the disk path but not the embedded path | AC7 | The AC7 fixture delivers the tampered bytes **via the embedded FS** (the path agent mode reads first, N-5) |
| 8 | Migration writes `banked` markers but omits `legacy` | AC1 | The `39` count is asserted directly; a corpus-only migration yields 19/0/40 |
| 9 | Mirror check dropped from `--check` | AC8 | No other layer reads `cmd/ailang/prompts/` bytes (N-3/N-7); only L3(d) can go red on that input |
| 10 | Migration/`freeze` writes markers to `prompts/versions.json` only (mirror registry left unmarked) | AC9 | AC9's jq counts run against the MIRROR file and its `--check` input alters only the mirror registry; a source-only writer or a `.md`-only checker leaves both green — the registry-entry comparison is the sole observable that fires |
| 11 | `--check` ignores `prompt_sha256` (or compares rows to each other but never to the frozen hash) | AC10 | AC10's fixture is green under every other sub-check by construction; only the row-hash-vs-frozen-hash cross-check can produce the asserted failure |
| 12 | Bank path populates `prompt_sha256` from the registry `hash` field instead of hashing the served bytes | AC11 | The PLACEHOLDER fixture makes the copied value (`"PLACEHOLDER"`) and the true digest disjoint; the asserted 64-hex digest of the fixture bytes is producible only by hashing served content |

---

## Milestones

### M1 — Schema, freeze command, migration, standard-mode teaching error (≤1 day) — **minimum viable ruling-compliance**

`FrozenMarker` structs in both loaders; `ailang prompt freeze <version>` / `--migrate` /
`--check` (derivation + hash-integrity subset, i.e. L3(a)(b) callable locally), with **every
marker write going to BOTH registry files** (Q3/V-J); the migration commit (19/39/1 per Q4,
applied to both registries, verified by V-I's intersection); the two-state error in
`internal/eval_harness/prompt_loader.go` (Q5); frozen-PLACEHOLDER refusal in the same loader.
Tests: AC1, AC2, AC3, first half of AC6, AC9's jq-count/diff half.
This alone makes the D-41(c) state representable, recorded for all 59 versions **in both the
registry agent mode reads and the one standard mode reads**, and enforced with a teaching error
on the standard-mode path — everything else is hardening.

### M2 — CI gate (≤1 day)

`make check-prompt-freeze` target invoking `ailang prompt freeze --check` with the merge-base
immutability check L3(c) and the mirror check L3(d) covering frozen `.md` bytes AND frozen
entries of `cmd/ailang/prompts/versions.json` (the marker-carrying file, Q7/V-J); wire into
`make ci`; the one-sentence freeze note in `create_prompt_version.sh`'s "Next steps"; CHANGELOG
+ `docs/docs/guides/evaluation.md` paragraph. Tests: AC4, AC5, second half of AC6, AC8, AC9's
`--check` half.

### M3 — Close the agent-mode verification hole (≤1 day)

Frozen-hash verification in `internal/prompt.LoadPrompt` for both embedded and disk reads (L2);
remove the silent `DefaultPrompt`/`"default"` fallbacks in `langreg/ailang.go` and
`langreg/python.go` (Conflict Surface 5); same teaching error text as L1. Tests: AC7.
Independently landable: it depends only on M1's schema field being present in the registry, which
M1's dual-registry migration commit guarantees on `dev` (a source-only migration would make this
milestone vacuous on rig binaries — the embedded registry would carry no markers, V-J).

### M4 — Bank-time byte evidence (≤1 day)

`prompt_sha256` (sha256 of the served bytes) written on every new banked row at the V-D sites for
both modes; `--check` gains the row-hash cross-check (all byte-carrying rows citing a frozen
version must agree with each other and with the frozen hash; zero byte-carrying rows → warning
only). Closes the first-bank bytes-swap hole (Q7); the TOFU residual for pre-M4 rows is declared,
not fixed. Tests: AC10, AC11. Independently landable after M1 (needs the marker schema for the
cross-check; the bank-side field write itself has no dependency and starts accumulating evidence
from the moment it lands).

---

## Testing Strategy

- Unit tests live beside the loaders (`internal/eval_harness/prompt_loader_test.go`,
  `internal/prompt/loader_test.go`, new `cmd/ailang` test for `freeze`/`--check`), using
  `t.TempDir()` fixture registries — never the real `prompts/` tree (House rule: tests must not
  mutate the shared working tree).
- The `--check` tests drive the real binary logic against fixture git state (a scratch repo
  created in the test), because L3(c) is meaningless without a merge-base to diff against.
- M4's bank-path test (AC11) lives beside the result writers (V-D sites) and asserts on the
  written JSON, the observable downstream of the mechanism, not on any intermediate variable.
- V-H's existing tests are left untouched (Conflict Surface 8); per the testing policy
  ("remove out-of-date tests"), nothing here obsoletes them — they test active-version loading,
  which this design preserves.

---

## Axiom Compliance

- **Principle 2 (no silent fallbacks)**: M3 removes two banked-attribution fallbacks (N-4);
  frozen+PLACEHOLDER becomes a refusal, not a skip.
- **Principle 3 (systemic fix)**: the audit found the same defect class (unverified prompt bytes)
  on both eval paths and fixed the class (L1+L2+L3), not the incident.
- **Fail-safe direction stated** (Q2): over-freeze, because the asymmetric cost is provenance
  corruption.

---

## Open Questions

1. **Devtools registry**: should `internal/devtoolsprompt`'s separate `versions.json` get the
   same marker + gate once this lands? (Candidate follow-up; needs its own corpus-attribution
   measurement first.)
2. **General mirror sync**: L3(d) covers frozen ids only. A full `prompts/` ↔
   `cmd/ailang/prompts/` sync check for mutable versions is cheap to add in the same script but
   changes a long-standing manual workflow (N-7) — deferred to its own decision.
3. **`reason: "manual"`**: reserved in the schema for a human choosing to freeze an unbanked
   version (e.g. before a publicized comparison). No workflow ships it in M1–M4; the enum slot
   exists so the schema does not need a second bump.
4. **TOFU sunset (Q7 residual)**: with r3's merge-base cutoff the residual is already closed to
   growth — newly added rows must carry byte evidence or CI fails — so the only remaining question
   is when to retire the warning for the *historical* set entirely (i.e. re-bank or retire the
   17,343 pre-cutoff files). That is a dated policy decision for the ledger, not a code change.

---

## Quorum revision log (round 1 → r2 → r3)

Round-1 verdict: **blocked**, 3/3 external reviewers present (`absent_reviewers: []`).

| Objection | Reviewer | Disposition in r2 |
|-----------|----------|-------------------|
| 1 — "L3(c) protects only versions frozen at merge-base; the first-bank PR can bank bytes A and freeze bytes B undetected" | gpt5-6-sol | **Accepted as a real hole.** Direction (A) chosen: bank-time `prompt_sha256` of the served bytes + `--check` cross-check (new Q7, L4, M4, AC10/AC11, mutation rows 11-12), with the residual for pre-M4 rows declared in Non-Goals and Open Question 4 rather than implied away |
| 2 — "M1's migration writes only the source registry; agent mode reads the embedded registry first, so freeze markers never reach the rigs; L3(d) checked only `.md` files, not the marker-carrying registry" | gemini-3-1-pro | **Accepted, confirmed by controller measurement (V-J).** Migration and every `freeze` write now go to BOTH registry files (Q3/Q4, M1); L3(d) extended to frozen registry entries (M2); AC9 goes red on a mirror-registry-only divergence, with the hollow-pass note that a `.md`-only check stays green on that input; mutation row 10 |
| 3 — "The 19/39/1 split is unverified: nobody checked the corpus ids exist as registry keys" | oc-glm-5-2 | **Refuted by controller measurement (V-I)**: intersection holds exactly (19 present, 0 missing, incl. `v0.6.5`; positive control `python` → True, negative control `v9.9.9-nonexistent` → False; 19+39+1=59). The missing verification ROW was a real doc defect and is now V-I; the design and AC1 are unchanged |

### Round 2 → r3 (controller, narrow-refinement carve-out)

Round-2 verdict: **blocked**, 3/3 external reviewers present (`absent_reviewers: []`), and
`gemini-3-1-pro` **flipped to pass**. Both surviving objections carried a concrete
reviewer-authored `proposed_fix` and neither disputed the design DIRECTION (one refines the L4
predicate, one asks for a verification row), so the controller applied the ratified
**narrow-refinement carve-out**: the reviewers' fix texts are applied VERBATIM, by the controller,
without a third designer run. No controller-invented resolution; no objection overridden.

| Objection | Reviewer | Disposition in r3 |
|-----------|----------|-------------------|
| "M4 does not guarantee byte evidence: the legacy exception is defined by field absence rather than by whether a row predates M4, so a stale-binary first-bank PR still passes" | gpt5-6-sol | **Accepted; fix applied verbatim.** `--check` now uses the MERGE-BASE to separate historical from newly added rows: field absence warns only for baseline files already present at the M4 cutoff commit; every newly added banked row must carry a valid 64-hex `prompt_sha256`; a newly frozen version requires ≥1 newly added row with byte evidence, all agreeing with the frozen registry hash, else CI fails. Q7, L4, Non-Goals, Open Question 4 and AC10 updated; new **AC10(b)** is the stale-binary fixture the reviewer asked for. The residual is now a CLOSED set that cannot grow |
| "Conflict Surface item 10 (readers ignore unknown fields) is asserted bare with no verification row" | oc-glm-5-2 | **Refuted by controller measurement, and the requested row added verbatim as N-10.** `DisallowUnknownFields` = **0** across `cmd/ internal/` and 0 repo-wide; the three named readers are 0 each; the reviewer's own prescribed positive control (`json.NewDecoder`, since it is a `Decoder` method) fires at **10** in `cmd/ailang/` and **122** repo-wide. The non-breaking claim stands; the missing ROW was a real doc defect |
| (non-blocking) "`prompt_sha256` should hash the fully evaluated string, not the loader's return" | gemini-3-1-pro (**pass**) | **Accepted anyway** — the sharpening is free and correct. Q7's bank-time bullet now reads "the fully evaluated prompt string immediately prior to model dispatch (post-variable substitution, if any)" |

---

## Related Documents

- `design_docs/planned/m-contract-verification-coverage.md` — house-style sibling (structure model)
- `design_docs/PROGRAM.md` — routing: this is harness/data-integrity work, not a core change
- `.claude/skills/prompt-manager/` — the workflow this mechanism constrains
- Decision ledger row D-41 (mission log, iteration 291) — the ruling
