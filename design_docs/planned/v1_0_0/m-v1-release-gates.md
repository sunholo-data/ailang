# M-V1-RELEASE-GATES: Release gates, release candidates and the v1.0.0 runbook

**Status**: Planned
**Target**: v0.58.x (gates that protect v0.58.0 land first), complete before v1.0.0-rc.1
**Priority**: P0 for M1–M3a (they protect the v0.58.0 tag); P1 for the rest
**Estimated**: ~12 agent-days, plus Mark's decisions and a multivac apply (estimate; see Timeline)
**Dependencies**: [v1 release-readiness review](../v1-release-readiness-review-2026-11-09.md) §4.2–4.3;
siblings `m-v1-downstream-compat-gate.md` (downstream gate — referenced, not designed here) and
`m-v1-stdlib-surface-polish.md` (per-function Experimental marking — interface defined in §4.3)
**Planner-Lane**: opus-required (cross-repo, release path, fails-closed semantics)

Measured 2026-11-10 at `dev` HEAD (`2ce73c3fa`). Run history from `gh` over every `push` run on
`dev` since 2026-10-28 (244 runs of `ci.yml`, 244 of `build.yml`). Read-only throughout: nothing
was triggered, re-run, tagged or changed outside this file.

## Problem Statement

The review's §4.2 table names six gaps between "a tag was pushed" and "a release we can promise".
Each was re-checked against the files. Five hold as written; two are subtler than the review says.

| # | Claim | Verified? | Evidence |
|---|---|---|---|
| 1 | `release.yml` publishes with no test dependency | **Yes** | Only `needs:` are between build jobs (`release.yml:266,449,465`). Trigger `release.yml:3-7`. |
| 1a | v0.57.4's tag commit had a failed macOS build and still shipped | **Yes, with a correction** | SHA `dec3e8756`: `Build and Release` run 42011521825 = one `Build macos-latest` **failure** (step "Run tests", `TestMemWatchdogKillsAllocator` 60.01s timeout), `Build windows-latest` **cancelled** (fail-fast), and `Release` run 42013607303 **success**. But `ci.yml`'s `test`, `checks`, `test-windows` were all green. The "macOS build" is `build.yml`, whose jobs run the **full Go suite** per OS (`build.yml:71-82`) — it failed a test, not a compile. |
| 2 | A `v*` tag, rc included, redeploys the one prod validator with no CI gate | **Yes** | multivac `terraform-triggers/triggers.tf:208-226` (`tag = "^v.*"`); `cloudbuild-registry-validator.yaml:43-73` is build → push → `gcloud run services update`, no gate step. The core pipeline *does* gate (`cloudbuild-release.yaml:52-80`, `test` check-run only). |
| 3 | No rc since v0.3.0-alpha1 | **Yes** | The only hyphenated tag in the clone's refs is `v0.3.0-alpha1`. **New:** `release.yml:7` excludes `v*-*` tags entirely, so an rc today gets **no GitHub release, no binaries** — downstream could not install it. |
| 4 | Freeze gate checks code==goldens, re-freeze accepts anything | **Yes** | `tools/freeze-stdlib.sh:4-6` says so in its header. Rendering: `.stdlib-golden/list.json` `mapE` is `"((a)->b,list[a])->[b]"`, `"effects":[]`, `"pure":false` — both spellings in one signature, and the callback's row is not recorded. Across the goldens: 231 `list[` vs 131 `[x]`. |
| 5 | Dangling `pre-push`; installer refuses | **Yes** | `.git/hooks/pre-push` → `.claude/worktrees/fleet-rulings/scripts/hooks/pre-push` (dated 2026-09-27); that worktree no longer exists. `install_hooks.sh:21-24` refuses any existing path incl. a dangling `-L`. SessionStart runs it every session (`session_start.sh:45`) and swallows the refusal (`\|\| true`). **Correction:** `checks` still runs `make verify-stdlib` on every dev push (`ci.yml:431-435`), so drift turns dev red after the push; what is lost is the *pre-push* layer. |
| 6 | No `-race` anywhere | **Yes** | 0 hits for ` -race` in workflows, `Makefile`, `make/*.mk`. 28 non-test packages start goroutines (incl. `internal/eval`, `internal/effects`, `internal/pipeline`). |
| 7 | Dev CI ~81% green, mostly macOS timing flakes | **Rate yes, cause no** | `ci.yml` 190/244 green (77.9%), `build.yml` 189/244 (77.5%); both green on 161/244 SHAs (66%). Of 126 failed jobs, timing flakes are a minority — see §6. |
| 7a | Required checks / `enforce_admins` | **Yes** | dev protection: required `test, lint, build, docs-gate, checks` (`UI build gate` removed by Mark 2026-11-12, stub workflow deleted); `strict=false`; `enforce_admins=false`. `test-windows` and every `build.yml` job are **not** required. The `dev` ruleset only blocks deletion and force-push. |

**Impact.** Every downstream user, the package registry and the 1.0 promise sit behind these
gates. The ones that fire on the tag (1, 2) are cheapest to fix now and most expensive to get
wrong on the v1.0.0 day.

## Goals

**Primary Goal:** a `v*` tag cannot publish binaries, move the registry validator or reach prod
unless the exact tagged SHA is green on every release-relevant check, and the Stable surface
cannot shrink between two final tags without a visible, reviewed override.

**Success Metrics:**
- Tagging a known-red SHA (control: `dec3e8756`) is refused by `release.yml` and the validator build — 0 today.
- A `v1.0.0-rc.N` tag publishes a GitHub **pre-release** and deploys TEST only; prod, `latest`, `install.sh`, docs version and validator unchanged.
- Removing or re-typing a Stable stdlib export fails CI against the last final tag; adding one passes.
- Dev push runs green ≥95% over the 7-day soak with quarantined tests excluded (66% both-green today).
- `pre-push` resolves on the main checkout and SessionStart reports it when it does not.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Release gate queries check-runs for the tagged SHA (not re-running tests) | Re-running in the deploy path doubles CI and re-rolls flakes; matches `cloudbuild-release.yaml:53` | human | design | med |
| Validator moves on explicit promote, not on the tag | Changes what every `ailang publish` accepts, fleet-wide, instantly | human | design | high |
| Pre-release tags: GitHub pre-release + TEST only | Defines what an rc means for users and infra | human | design | med |
| Stable-surface baseline = goldens at the last **final** tag, read with `git show` | No second banked copy to drift; no old binary to rebuild | agent | design | low |
| Override marker = annotated-tag line / overrides file, both surfaced in release notes | The escape hatch is the gate's weakest point | human | design | med |
| Flake policy: quarantine-with-issue + expiry, no blanket retry | Decides whether the soak signal means anything | human | compile | low |

### Design Freeze

- [ ] Decision 1 (validator on promote) — Mark
- [ ] Which check-runs a release requires (§1 list) — Mark
- [ ] Override marker form (§1.3) and who may use it — Mark
- [ ] rc may never be promoted to prod (§3) — Mark

## Deferred Decisions

- Script language for `require_checks` (bash+jq vs a small Go tool under `tools/ci/`) — agent
- Exact normalisation algorithm for type strings (α-renaming method) — agent, with the selftest fixtures in §4.6
- Nightly race schedule hour and runner size — agent
- Quarantine helper name and env var — agent
- Whether the CLI-flag snapshot renders `--help` or walks FlagSets — agent

## Solution Design

### 1. Release binaries wait on the tagged SHA's checks (`release.yml`)

**1.1 Mechanism.** A first job `ci-gate` in `release.yml`; every build job gains
`needs: [ci-gate]`, so nothing is built, signed or drafted on red. It calls one shared script,
`tools/ci/require_checks.sh <sha> <name>...`, that polls
`GET /repos/{owner}/{repo}/commits/{sha}/check-runs?filter=latest&per_page=100` (paginated) and
requires `conclusion == success` for **every** check-run carrying each required name. "Every" matters:
`build.yml` emits two runs both named `Build macos-latest` (amd64 and arm64, `build.yml:28-37`);
the current core gate takes `rs[0]` only (`cloudbuild-release.yaml:71`) and would miss the red one.
Missing or pending → keep polling (15 s × 180 = 45 min; `ci.yml` takes ~19–23 min per
`cloudbuild-release.yaml:56`); any failure/cancelled/timed_out → exit 1. Job permission `checks: read`.

**Required set (proposal, decision 2):** from `ci.yml` — `test`, `checks`, `lint`, `build`,
`test-windows`, `govulncheck (vuln gate)`, `float determinism (arm64)`; from `build.yml` —
`Build ubuntu-latest`, `Build windows-latest`, both `Build macos-latest`. The same list is a
data file (`tools/ci/release_required_checks.txt`) read by `release.yml`, the validator build and
the core build, so one edit moves all three. `ci.yml`'s `changes` lane only skips jobs on PRs
(`ci.yml:53-57`); pushes run the full matrix, so every name exists for a dev SHA.

**1.2 Failure message.**
`release gate: v0.58.1 (abc1234) is not releasable — Build macos-latest: failure (run 4201…/job 1271…). Re-run the failed job if it is a quarantined flake; otherwise fix and re-tag. Override: §1.3.`
A known flake is handled by **re-running the failed job** (the check-run API then reports the
latest attempt), never by the override.

**1.3 Escape hatch.** An annotated tag whose message contains a line
`release-gate-override: <issue-url> <reason>`. The gate reads it
(`git for-each-ref refs/tags/$TAG --format='%(contents)'`), emits `::warning::`, skips only the
named failing checks, and `create-release` prepends **"Released with a CI-gate override: …"** to the
release notes. The marker lives in the tag object forever and is visible on GitHub. The same
marker is honoured by the validator and core Cloud Build gates (§2), so an emergency release is
one decision, not three.

**1.4 Cost.** One ubuntu job, ~1 min when CI is already done, up to 45 min wall if the tag is
pushed with its commit. Releases get later, not more expensive.

**1.5 Control.** (a) Fixture tests for `require_checks.sh`: all green, one of two same-name red,
pending→success, missing name, >100 runs (pagination), override present — each arm dies when its
branch is neutered (repo convention, e.g. `verify-stdlib-selftest`). (b) Live control:
`workflow_dispatch` dry-run input `sha=dec3e8756…` (v0.57.4) must go **red** on
`Build macos-latest`; a green SHA must pass. No tag is created for either.

### 2. Registry validator: release-gated, pre-release-blind, promoted with core

**2.1 What it is.** One Cloud Run service with no dev/test ladder
(`cloudbuild-registry-validator.yaml:1-9`), embedding the tagged `ailang` binary
(`docker/Dockerfile.registry-validator:29-35`). Routes (`cmd/registry-validator/main.go:86-97`):
`/publish`, `/unpublish`, `/rebuild-index`, admin keys, `/health`, `/version`, read APIs.
**There is no re-validation route:** packages are checked once, at publish; a validator upgrade
never re-judges what is already published.

**2.2 The semantic question.** A validator upgrade changes what `ailang publish` accepts. Two
measured couplings make "move with the tag" wrong:
- PUB005 tells a publisher to *"upgrade the publisher to <validator version>"*
  (`validate.go:285-300`). On a tag, the validator moves within minutes; the GitHub release is
  published only after five builds, signing and provenance. For that window (and **forever** for an
  rc, which `release.yml:7` never publishes) the validator demands a version nobody can install.
- v0.58.x is the last breaking window: exhaustiveness errors (review §3.1b) and `pure` ⇒ empty row
  (#2184) make the validator reject packages that 0.57 accepted. That should happen when users can
  also get the compiler that explains it, i.e. at promote.

**2.3 Design (decision 1, recommended).**
- **multivac** `triggers.tf`: the validator trigger's tag regex becomes
  `^v[0-9]+\.[0-9]+\.[0-9]+$` (finals only; Cloud Build tag filters are RE2).
- **ailang** `cloudbuild-registry-validator.yaml`: add the same `ci-gate` step as core (§1, shared
  list, same override marker); add a guard step that exits 0 with "pre-release: not deployed" when
  `TAG_NAME` has a `-` (belt and braces for manual runs); replace `services update` with
  `gcloud run deploy … --no-traffic --tag cand-<ver-dashed>`, and smoke the candidate URL
  (`/health`, `/version == TAG_NAME`, one dry-run validation of a fixture package).
- **multivac** `scripts/release.sh promote core vX` (and `cloudbuild-promote.yaml`): after the core
  images, shift validator traffic to `cand-<ver>` (`update-traffic --to-tags`). Prod gets one
  ordered event: core, MCP and validator move together, after the GitHub release exists (promote
  checks `gh release view vX --json isDraft` is false).
- **Rollback** = `update-traffic` back to the previous revision; images stay tagged `:vX`.
- **Re-validation:** none in prod. A dry-run re-validation of every published manifest against the
  candidate binary belongs to the downstream gate (`m-v1-downstream-compat-gate.md`); it reports
  packages the new rules would reject, so the changelog can name them before promote.

**2.4 Minimal slice for v0.58.0 (M3a).** If the promote coupling (M3b) is not ready by the v0.58.0
tag: trigger regex + `ci-gate` step only. That alone removes the rc hazard and the ungated deploy.

**2.5 Control.** Run the gate step's script locally against `dec3e8756` (red) and a green SHA.
After the trigger change, `gcloud builds triggers describe` shows the new regex (read-only).

### 3. Release candidates

**Naming:** `v1.0.0-rc.N` (SemVer dot form; N ≤ 9 — see `semverLess` below). An rc commit sets
`std/VERSION` to the rc tag (the core smoke gate compares the MCP's `latest` to the tag,
`cloudbuild-release.yaml:514-518`); v1.0.0 is that commit plus `std/VERSION`, docs and changelog.

| Channel | rc must | Today | Change |
|---|---|---|---|
| GitHub release | publish a **pre-release** with signed binaries | excluded: `release.yml:7` `!v*-*` | allow `v*-rc.*`; `prerelease:` already computed (`release.yml:422`); set `make_latest: false` explicitly |
| GitHub "latest" / `install.sh` default | not move | `install.sh:58` uses `/releases/latest`, which excludes pre-releases | none; verify in runbook |
| `go install …@latest` | not move | Go's module proxy excludes pre-releases from `@latest` | none (v1 needs no `/v1` path suffix) |
| Core Cloud Build | build + deploy **TEST** + smoke | fires on `^v.*` | keep; it is the rc's integration test |
| Prod promote | **refuse** | `release.sh:148` regex accepts `-rc.1` | multivac: refuse a pre-release unless `ALLOW_PRERELEASE=1` (decision 4) |
| Registry validator | not move | moves | §2 |
| Docs site version | not move | `docs/scripts/generate-version-constants.sh:18` and `docusaurus-deploy.yml:242` take the top of `git tag --sort=-v:refname`; git and `sort -V` both order `v1.0.0-rc.1` **after** `v1.0.0` (checked with `sort -V`; git's default `versionsort` behaves the same) | filter tags with `-` out of "stable" |
| Release notes | n/a | `release.yml:368` diffs from `git describe --tags --abbrev=0 HEAD^` → v1.0.0's notes would start at rc.N | diff from the previous **final** tag |
| `ailang examples download` | work on an rc binary | needs a release asset (`examples.go:379`) | fixed by publishing the pre-release |
| Homebrew / other channels | — | none found in tracked files | none |

**Version code with pre-release suffixes** (read, and tests cited where they exist):
- `internal/pkg/version_compat.go` `ParseSemver` strips the suffix (`0.9.5-rc1` → 0.9.5, pinned by
  `version_compat_test.go:16`). So `>=0.40.0` **accepts** `1.0.0-rc.1`, and `>=1.0.0` also accepts
  it (SemVer says rc < final). Harmless in practice; record it as intended and add a test for
  `SatisfiesAILANGVersion(">=1.0.0", "v1.0.0-rc.1")`, which has none today.
- `cmd/ailang/check_skew.go` `releaseCore` drops the suffix: a lock cut by rc.1 checked by 1.0.0
  gives no VER001 warning. Acceptable.
- `internal/eval_harness/release_tag.go` keeps pre-release tags (doc example line 18). Correct.
- `internal/pkg/changelog.go` `ReleaseGatesHard` takes the first `x.y.z`; rc counts as ≥ threshold. Fine.
- `internal/effects/pkg.go` `semverLess` sorts pre-release before release (correct) but compares
  the pre-release string lexically (`rc.10` < `rc.2`). Fix to numeric identifiers or cap N ≤ 9.
- Lexical sorts exist only in Internal eval tooling (`internal/eval_analysis/loader.go:222`,
  `tags.go:293`); they already mis-order `v0.9` vs `v0.58` and are out of scope.
- **Seam note:** four separate semver implementations (`version_compat.go`, `effects/pkg.go`,
  `changelog.go`, `executor/environment.go`). Not consolidated here; the rc tests above pin the
  three on the release path.

### 4. Stable-surface gate

**4.1 Mechanism.** `tools/check-stable-surface` (Go) compares, per Stable module,
`git show <baseline>:.stdlib-golden/<m>.json` with the working tree's golden, after normalisation.
`make verify-stdlib` already proves working-tree goldens equal the code, so the two together prove
"code vs last release" **without building an old binary**. Baseline = highest final tag reachable
from HEAD (`git describe --tags --abbrev=0 --exclude='*-*'`). Before v1.0.0 it runs in
**report mode** against the latest v0.58.x (exercises the tool, fails nothing); from v1.0.0 it
**enforces**. Runs in the `checks` job (already `fetch-depth: 0`, `ci.yml:305`) and in `pre-push`
when the stdlib paths change.

**4.2 Rules.** FAIL: Stable export removed; type changed after normalisation; effect row
changed (added or removed — conservative, overridable); tier lowered (Stable → Experimental or
Internal) — read from the **baseline's** manifest, so re-marking cannot launder a removal. ALLOW:
new export, new module, Experimental → Stable, changes to Experimental/Internal items, the `ifc`
block (IFC is Experimental per `stability.md`; diffed and reported, never failed).

**4.3 Tier source (interface for `m-v1-stdlib-surface-polish.md`).** Proposed manifest
`std/STABILITY.toml`: `[modules] list = "stable"`, plus per-function exceptions
`[exceptions."std/json"] repair = "experimental"`. The sibling may instead put per-function tiers
in source annotations; this gate needs only that **`ailang iface std/<m>` emits a `"tier"` field
on every func and type** (enum `stable|experimental|internal`, module default applied), and that
it is recorded in the golden. Either source satisfies that contract. `stability.md` tables then
become generated from the manifest (follow-up, not required for the gate).

**4.4 Normalisation.** (a) `[t]` → `list[t]` everywhere; (b) α-rename type variables by first
occurrence; (c) sort effect lists and record fields; (d) **effect rows on callbacks are a
prerequisite fix, not a normalisation**: `iface` renders `mapE` as `((a)->b,list[a])->[b]` with
`"effects":[]` — the callback row variable is dropped, so the promise "goldens cover effect rows"
(`stability.md` §stdlib) is not true today. `iface` must render function-typed parameters with
their rows (e.g. `((a)->b ! e, list[a])->list[b] ! e`) and the goldens must be re-frozen **in
v0.58.x**, because doing it after the baseline would itself read as a signature change. Owner:
whoever owns `internal/iface` rendering (coordinate with the stdlib-polish sibling, which
re-freezes anyway).

**4.5 Override.** `stability/overrides.toml` entries
`{item, baseline_tag, reason, issue, approved_by}`. The gate passes an item only with a matching
entry, and the release notes list every entry added since the previous final tag. **Honest limit:**
with `enforce_admins=false` and the bot able to push dev directly, no repo-side control makes the
marker unforgeable; it makes it **visible and auditable**. Decision 5 asks whether to tighten.

**4.6 Control.** `make check-stable-surface-selftest` with fixture goldens: remove export → red;
re-type → red; add export → green; rename type var → green; `[a]`↔`list[a]` → green;
Stable→Experimental in head → red; override present → green with notice. Wired into `checks`
(cihygiene's `TestGateTargetsAreWiredIntoAWorkflow` requires that).

**4.7 Other Stable surfaces (cheap first).**
- **MCP tools** (cheap): snapshot `@mcp_name` + signature from `ailang iface` of `mcp_tools/`,
  `executor_tools/`, `executor_oauth/` into `.surface-golden/mcp.json`; same diff rules.
- **CLI commands** (cheap): the dispatch table (`cmd/ailang/commands.go:26-61`) already renders
  `cli.md` (`TestCLIReferenceMatchesTable`). `Command` has **no tier field**; add one, snapshot
  name+group+tier. **Flags** (medium): parsed per command, no table; snapshot rendered `--help`
  for Stable commands only.
- **Error codes** (cheap, partial): `dist/error_codes.json` (58 records from
  `internal/errors/codes.go`); fail on removed code or changed category. It does **not** contain
  `PUBnnn`/`VER001`; say so in `stability.md` or extend the generator.

### 5. Hooks never point into a worktree

- `install_hooks.sh`: if `dst` is a symlink that is **dangling** or resolves under
  `*/.claude/worktrees/*` or `.git/worktrees`, replace it (print old → new). If the main checkout
  lacks `scripts/hooks/pre-push`, **refuse** rather than fall back to the worktree copy (inferred
  cause of the 09-27 link: the hook was installed from a worktree before it landed on dev; the
  fallback at `install_hooks.sh:13-17` permits exactly that). Regular files and foreign symlinks
  stay refused.
- `git_health.sh`: add one check — for each managed hook, missing or `[ -L h ] && [ ! -e h ]` →
  `⚠️ GIT: pre-push hook does not resolve — the stdlib and ci-quick gates are not running on push. make install-hooks`.
- `session_start.sh:45`: stop discarding the installer's refusal; surface one line.
- One-time repair of the main checkout (`make install-hooks` after the fix) is an attended step.
- **Control:** new `scripts/hooks/test_install_hooks.sh` arms (dangling, worktree target, foreign
  file, already ours, main copy missing) and a `test_git_health.sh` arm with a dangling hook;
  both join `make test-launchd-drivers`' shell suite (`make/test.mk:126`).

### 6. Race detection

- **Per-PR / push, required later:** `race-core` = `go test -race` over `CORE_PKGS`
  (`make/test.mk:435-439`) plus `internal/bytecode/...` and `internal/vm/...`. `test-core` was
  measured at 44–48 s (review §11); with the Go race detector's typical 2–10× slowdown, estimate
  **2–8 min** on ubuntu. Not measured here.
- **Nightly:** `go test -race -timeout 1800s ./...` on dev (`schedule:`), estimate **25–45 min**
  against the ~390 s non-race suite (`ci.yml:92`). Adds the platform packages where the goroutines
  are (`coordinator`, `daemon`, `messaging`, `runner`, `executor/*`, `websocket`, `server`).
- **Rollout:** first nightly is discovery; races found are filed and fixed or quarantined; then
  `race-core` becomes required on dev. A race in core blocks the soak; a race in an Internal package
  does not.
- **Control:** a `//go:build racecontrol` test with a deliberate data race, run once per job with
  `-tags racecontrol` and required to **fail** — proves `-race` is on.

### 7. Flakes and the soak signal

**Measured** (244 dev push runs per workflow, 2026-10-28 → 2026-11-10; 126 failed jobs; top-level
`--- FAIL:` from each failed job's log):

| Failure source | Failed jobs | Kind |
|---|---|---|
| `govulncheck` (28; 26 on 10-28 alone) | 29 | external: new advisory, red until allowlisted |
| `TestQuorumArtifactShellSuite` (`internal/mission/quorum`) | 28 on 14 SHAs, 3 bursts, Linux both workflows | real breakage, re-introduced 3× (Internal tooling) |
| `launchd drivers` — `generated_docs_and_mirror` | 15 | real: generated mirror drift |
| `cmd/ailang` package budget on Windows (`build.yml`, default 10 min `-timeout`; hang guard fails ~20 tests at 0.00 s) | 4 | structural; `ci.yml` uses 900 s (`ci.yml:742`) |
| `TestEffectRowGuardRejectsBeforeExecution` | 9 on 3 consecutive SHAs, all OSes | real regression, fixed in ~10 min |
| **Timing flakes** (single test, usually one of the two macOS jobs, non-consecutive SHAs) | ~25 (17 of the 20 macOS failures) | flaky |

**Top timing flakes** (rate = failing SHAs / 244 runs): `TestWorkerTerminalPTYLifecycle` 10
(4.1%; Linux+macOS, all on 10-28/29, none since — confirm fixed); `TestRuntimeNestedStopUsesRemainingDeadline`
5 (2.0%, macOS only, still active 11-09); `TestMemprobeDebugLogDoesNotAccumulate` 4 (1.6%, macOS);
then `TestMemWatchdogKillsAllocator` 2 (macOS 60 s timeout; the v0.57.4 failure),
`TestWorkerCLIProcessExitLifecycle` 2, `TestDaemon_ExcludesMatchedNotificationsSkipped` and
`TestDaemon_TaskDedupSuppressesRepeats` 1 each. This **contradicts the review's "mostly macOS timing
flakes"**: most red is real or structural, which is better news — it is fixable, not noise.

**Policy.**
1. *Flaky* = fails and passes on the same SHA (re-run or sibling job), or fails on non-consecutive
   SHAs with no fix between.
2. **Quarantine = skip-with-issue**: `testutil.Quarantine(t, "#NNNN", "2026-12-01")` skips when
   `AILANG_QUARANTINE=skip` (set in push/PR lanes) and runs verbosely in a nightly `quarantine` job
   that reports pass rate. Expired entries fail `make check-quarantine`. Language-core packages
   (`CORE_PKGS`) may not be quarantined without Mark.
3. **No blanket retry** (`-count`/rerun wrappers hide races). A human re-runs a failed job.
4. Fix structural causes, not symptoms: give `build.yml` the same `-timeout` budget as `ci.yml`;
   decide whether `build.yml`'s ubuntu test duplicate of `ci.yml test` earns its runner (decision 6).
5. **Soak day = green day:** every dev push run of `ci.yml` and `build.yml` green (quarantine
   excluded) or red only on an external cause (`govulncheck` new advisory) fixed within the day.
   `make soak-report` prints the per-day table from `gh run list`; a breaking fragment or a red
   core test resets the clock (review §4.3 step 5).
6. Direct pushes skip PR CI (bot bypass, `enforce_admins=false`): during the soak, every push run
   still runs both workflows, and the soak report counts them — the signal is post-push but complete.

### 8. The v1.0.0 runbook (lives in `release-manager/resources/v1-runbook.md`)

Each step has a verification; stop on the first failure.

| # | Step | Verify |
|---|---|---|
| 1 | Freeze: last v0.58.x soaked ≥7 days, downstream gate green | `make soak-report` shows 7 green days; sibling gate report |
| 2 | `dev` HEAD = last v0.58.x + version/docs/changelog only | `git diff v0.58.N HEAD --stat` touches only `std/VERSION`, docs, changelogs |
| 3 | All required checks green on HEAD | `tools/ci/require_checks.sh $(git rev-parse HEAD) $(cat tools/ci/release_required_checks.txt)` |
| 4 | Stable surface vs last v0.58.x: report empty or overrides listed | `make check-stable-surface` |
| 5 | rc: tag `v1.0.0-rc.1` (`std/VERSION` = rc) | `gh release view v1.0.0-rc.1 --json isPrerelease,isDraft` → `true,false`; latest unchanged: `gh api repos/sunholo-data/ailang/releases/latest --jq .tag_name` |
| 6 | rc TEST deploy + smoke | core release build SUCCESS (release-manager §7.6 command); validator **not** run |
| 7 | rc soak ≥48 h with downstream on rc | sibling gate rerun on rc binaries |
| 8 | Tag `v1.0.0` | release run green incl. `ci-gate`; `gh release view v1.0.0 --json isLatest,isPrerelease` → `true,false` |
| 9 | Signatures | `cosign verify-blob --signature …sig --certificate …pem --certificate-identity-regexp '^https://github\.com/sunholo-data/ailang/\.github/workflows/release\.yml@refs/tags/v.+$' --certificate-oidc-issuer https://token.actions.githubusercontent.com SHA256SUMS` (same identity as `install.sh:123-124`); `slsa-verifier verify-artifact` per `release.yml:443-447` |
| 10 | Installers resolve | `releases/latest` → `v1.0.0`; `install.sh` in a throwaway container prints `AILANG v1.0.0`; `GOPROXY=https://proxy.golang.org go list -m github.com/sunholo-data/ailang@latest` → `v1.0.0` |
| 11 | TEST smoke passed for v1.0.0 | core release build SUCCESS; validator candidate `/version` = v1.0.0 |
| 12 | Promote | `scripts/release.sh promote core v1.0.0 --dry-run` then `YES=1 scripts/release.sh promote core v1.0.0` (multivac checkout; an agent session has no TTY, so `YES=1` is required) |
| 13 | Prod verify | prod MCP `ailang_versions` → `latest == 1.0.0`; `curl -fsS https://registry.ailang.sunholo.com/version` → v1.0.0 |
| 14 | Docs site | version constant shows v1.0.0, not an rc |

**Rollback.** Prod: `YES=1 scripts/release.sh promote core v0.58.N` (images are versioned) plus
validator `update-traffic` to the prior revision (manual today; part of promote after M3b).
GitHub: `gh release edit v0.58.N --latest` — "latest" is only moved by re-marking; do not delete
the v1.0.0 release (assets are immutable, `release.yml:411-416`). Go: the proxy caches v1.0.0
permanently; the only withdrawal is a `retract` directive in a later `go.mod`.

### Files to Modify/Create

**ailang repo**
- `.github/workflows/release.yml` (+40) — `ci-gate` job, `needs:`, rc tag filter, `make_latest`, previous-final-tag notes
- `tools/ci/require_checks.sh` (~120 LOC) + `tools/ci/release_required_checks.txt` + fixture tests (~150)
- `cloudbuild-registry-validator.yaml` (+40) — ci-gate, pre-release guard, `--no-traffic` candidate + smoke
- `cloudbuild-release.yaml` (±15) — use the shared list and all-runs-per-name; honour the override marker
- `tools/check-stable-surface/` (~400 LOC Go + fixtures) — normaliser, diff, overrides, selftest
- `internal/iface/` (±?) — render callback effect rows; emit `tier` (shared with the polish sibling)
- `std/STABILITY.toml`, `stability/overrides.toml` (new)
- `cmd/ailang/commands.go` (+tier field), `.surface-golden/` (CLI, MCP, error-code snapshots)
- `scripts/hooks/install_hooks.sh`, `scripts/hooks/git_health.sh`, `scripts/hooks/session_start.sh`, new `scripts/hooks/test_install_hooks.sh`
- `.github/workflows/ci.yml` — `race-core`, nightly race + quarantine jobs, stable-surface step; `build.yml` `-timeout`
- `internal/testutil/quarantine.go` (~60 LOC), `make/test.mk` (`check-quarantine`, `soak-report`)
- `docs/scripts/generate-version-constants.sh`, `.github/workflows/docusaurus-deploy.yml:242` — exclude pre-release
- `internal/effects/pkg.go` (numeric pre-release compare), `internal/pkg/version_compat_test.go` (+rc cases)
- `.claude/skills/release-manager/SKILL.md` + `resources/v1-runbook.md` — rc procedure, runbook

**multivac repo** (deploys via its own branch triggers; `triggers.tf` is applied by terraform, not by a push)
- `terraform-triggers/triggers.tf:219` — validator tag regex finals-only
- `scripts/release.sh`, `cloudbuild-promote.yaml` — refuse pre-release; validator traffic shift; require published GitHub release

## Milestones (ordered: v0.58.0 protection first)

| M | Scope | Repo | Before | Estimate |
|---|---|---|---|---|
| M1 | Hook installer + `git_health` check + tests; repair main checkout | ailang | v0.58.0 | 0.5 d |
| M2 | `require_checks.sh` + `release.yml` ci-gate + override marker + controls | ailang | v0.58.0 | 1.5 d |
| M3a | Validator: finals-only trigger + ci-gate step + pre-release guard | multivac + ailang | v0.58.0 | 1 d |
| M4 | Flake triage, quarantine helper, `build.yml` timeout, `soak-report` | ailang | soak start | 2 d |
| M5 | rc readiness (§3 table, version tests, promote refuses rc, skill) | both | rc.1 | 1.5 d |
| M3b | Validator candidate revision + promote coupling | both | rc.1 | 1.5 d |
| M6 | Stable-surface gate (report mode) + iface effect-row rendering + tier contract | ailang | last v0.58.x | 3 d |
| M7 | Race: nightly discovery, then `race-core` required | ailang | soak end | 1 d + triage (unknown) |
| M8 | Runbook dry-run on a v0.58.x patch (steps 3–13 minus promote) | both | v1.0.0 | 0.5 d |

## Success Criteria

- [ ] `release.yml` dry-run against `dec3e8756` is red; against a green SHA green; fixture arms each die when neutered
- [ ] An rc tag yields a GitHub pre-release, TEST deploy, no validator build, unchanged `latest`, docs and prod
- [ ] Validator changes only at `promote core`; `/version` matches prod core
- [ ] Stable-surface selftest passes; report mode runs on every push; enforcement switched on at v1.0.0
- [ ] `pre-push` resolves; SessionStart warns on a planted dangling hook
- [ ] Nightly race job exists; `race-core` required; control test proves `-race` is live
- [ ] 7 consecutive soak days ≥95% green with ≤5 quarantined tests, none in `CORE_PKGS`
- [ ] Changelog fragment per milestone; `stability.md` names the gate and the override process

## Testing Strategy

Every gate carries a planted-failure control (§1.5, §2.5, §4.6, §5, §6) wired into CI so a
neutered gate goes red. Shell arms follow `scripts/hooks/test_pre_push.sh` (no network, temp
repos). Version code: table tests for `SatisfiesAILANGVersion`, `releaseCore`, `semverLess` with
`-rc.1`, `-rc.10`, git-describe suffixes. Live checks are read-only `gh`/`gcloud describe` calls.

## Non-Goals

- Downstream compatibility gate and published-package re-validation — `m-v1-downstream-compat-gate.md`
- Stdlib renames and per-function marking itself — `m-v1-stdlib-surface-polish.md`
- Consolidating the four semver implementations; Homebrew/winget channels; publisher signatures
- Making `enforce_admins` true (a decision, below, not work here)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Gate blocks an urgent fix on a flake | High | Re-run the job; tag-message override (§1.3), visible in notes |
| Required-name drift (job renamed) silently passes | High | Missing name = pending → timeout = fail closed; list in one file |
| Check-run pagination / duplicate names | Med | `filter=latest`, paginate, require all runs per name (fixtures) |
| Normaliser false positives block 1.x minors | Med | Report mode through v0.58.x; selftest fixtures; override file |
| Race discovery floods the soak | Med | Nightly first; Internal races don't block |
| multivac change lands late (separate deploy path) | Med | M3a is a one-line regex + an ailang-side guard that works alone |
| Go proxy caches a bad v1.0.0 forever | High | rc soak; `retract` documented |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Releases are a function of the SHA's recorded checks |
| A2 Replayability | +1 | Gate verdicts reproducible from the check-run API and tags |
| A3 Effect Legibility | +1 | Goldens finally record callback effect rows |
| A4 Explicit Authority | +1 | Overrides are explicit, named, published |
| A5 Bounded Verification | 0 | — |
| A6 Safe Concurrency | +1 | First race detection |
| A7 Machines First | +1 | Machine-readable tier manifest and surface snapshots |
| A8 Minimal Syntax | 0 | No syntax |
| A9 Cost Visibility | 0 | CI minutes stated, small |
| A10 Composability | 0 | — |
| A11 Structured Failure | +1 | Gates fail closed with named checks |
| A12 System Boundary | +1 | Prod moves only on promote |

**Net Score: +8.** No −1 on A1, A3, A4, A7.

## Decisions for Mark

1. **Validator on promote, not on tag** (§2.3). *Recommend yes;* M3a alone if time is short.
2. **Release requires `build.yml` jobs too**, incl. both macOS (§1.1). *Recommend yes*, once M4's quarantine lands; until then a red macOS job means re-run.
3. **Override = annotated-tag line** honoured by all three gates (§1.3). *Recommend yes.*
4. **rc never promotable to prod.** *Recommend yes* (`ALLOW_PRERELEASE=1` exists for one-off tests).
5. **Tighten bypass** (`enforce_admins`, code-owner review on `stability/overrides.toml`). *Recommend:* leave dev bypass for the bot, but require overrides via PR; accept "auditable, not unforgeable".
6. **Drop `build.yml`'s ubuntu test leg** (duplicates `ci.yml test`; 26 of the 53 failed `build.yml` jobs). *Recommend yes* after M2 lists the remaining names.
7. **`govulncheck` red does not reset the soak** if fixed same day. *Recommend yes.*

## What I could not verify

The Cloud Build history of the validator trigger (no `gcloud` calls made); whether git silently
skips a dangling hook (inferred from git's hook lookup, not tested); race-run timings; the
`TestQuorumArtifactShellSuite` root cause; whether `TestWorkerTerminalPTYLifecycle` was fixed or
only stopped failing. The hook's 09-27 origin is inferred.

## References

- Review: `design_docs/planned/v1-release-readiness-review-2026-11-09.md` §4
- Prior gates: `design_docs/implemented/v0_24_1/m-release-gate.md`, `design_docs/implemented/v0_35_0/m-unified-release-model.md`
- Promise: `docs/docs/reference/stability.md`; axioms: [Design Axioms](/docs/references/axioms)
