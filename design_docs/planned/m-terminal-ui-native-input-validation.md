# Native terminal sprint validation and delivery

Implementation: core PR [#1756](https://github.com/sunholo-data/ailang/pull/1756),
package PR [#117](https://github.com/sunholo-data/ailang-packages/pull/117).
Core feature commit 2f807c563; integrated current dev acc227e55 at cdcea3bdb.
Package feature commit c0ec852, integrated compiler evidence e52ae17.
Local implementation validation passes. Independent provisional review: 77/100
plus 10 regression bonus points. Repaired Windows and Linux runtime CI passed at
274f0be88 (CI run 38037895269); Sonar alone rejected new coverage at 69.1%, below
its unchanged 80% gate. Meaningful additional host/VM/image/EOF controls address
that gap. Final-head CI, publication and fresh registry consumer checks remain
required; this is not a completed sprint. The user explicitly approved both PR
merges, a supporting core release and package publication.

## Concrete evidence

| Check | Result |
|---|---|
| Release preflight | All five gates pass: tests, lint, file sizes, import goldens and eval configuration; `/private/tmp/terminal-pre-release-checks-final.log` |
| v0.54.0 metadata checks | Post-update full make test and lint exit 0; boundaries and changelog/design references pass |
| Integrated full `make test` | Exit 0, `/private/tmp/terminal-integrated-test.log` |
| Integrated lint | 0 issues, `/private/tmp/terminal-integrated-lint.log` |
| Architecture / file sizes | No violations, all files <=800 lines |
| Example manifest | Verified; no module drift; pre-existing missing-example warning retained |
| Stdlib interfaces | All 48 verified; only additive io and new terminal goldens changed |
| Platform builds | Darwin native; Linux/Windows CLI and WASM compile |
| Actual PTY cleanup | 12 cases: eval/strict VM normal, explicit exit, runtime error, SIGINT, SIGTERM; evaluator budget; host panic. Active ICANON/ECHO off, ISIG on, termios and cursor/alternate-screen restored |
| Ownership/lifecycle races | 100-repeat queued-signal race controls pass; process-wide sessions restored; async unread lines retained; concurrent output race regression passes |
| Mutation controls | Arrow Up mapped to Down fails decoder control; omitted device restore fails lifecycle controls; original sources restored |
| Existing IO fixtures | Echo, greeting, greeting with args and progress rendering: evaluator/strict VM endpoint parity; failed native attempt is not re-executed |
| Installed package bin | Unrelated cwd: Down+Enter selects item 2 immediately; 20x8 compact view fits; enlargement retains state; q restores terminal |
| Redirected input | Blank line renders a second frame; exact EOF ends line/plain; plain and redirected auto contain no ESC |
| Named package tests | 39 passed / 0 failed / 0 skipped: 37 named bodies + 2 properties, 100 generated inputs each |
| Strict VM package tests | 37 named bodies / 0 fallback; property tests run on evaluator by CLI definition |
| Source inline tests | 42 passed / 0 failed / 41 skipped; 26 missing-generator skips are not proof |
| Contracts | 12/34 verified / 0 refuted / 21 skipped / 1 Option[string] Z3 encoding error; no uncontracted exports |
| Strict package quality | 0 gates; compile 12 files; 77 signatures; 30/34 exports pure; smoke passed |
| Prompt | v0.16.7 teaches terminal/EOF; old v0.16.6 bytes preserved and legacy-frozen; single active mutable prompt, source/mirror consistency; prompt EOF example executed |
| Extra fresh-binary CLI suite | All controls pass except an environment-only shim mismatch; rerun without external stdlib override passes (fixture deliberately builds a dev-stamped binary) |

## Publication attempt

Actual authorized publication on 2026-10-10 reached the registry and was rejected.
Validator `/version` reported v0.53.2, commit bebfe6818. Local quality and dry-run
passed. Remote compilation passed 8 pure files and failed `_smoke.ail`,
`adapter.ail`, `adapter_test.ail`, `demo.ail` because `std/terminal` is absent.
Full response: `/private/tmp/terminal-ui-publish-attempt.txt`.
No package version was accepted/published. Fresh registry installation is pending.
Package PR crew CI also fails its terminal-tests stage for the same absent
std/terminal API on the released compiler; a released supporting CLI is required.

Archive content hash: `sha256:9d041ee63b3aa5768af55cb3bf03e722a36a1d825957223f1b105f851f5ed4a8`.
Interface v2: `e6b520bcefb43b7d8275594cad099203a701bb154dd53980ad8bddb6d908aab2`.
Validation documentation is excluded from the source archive identity.

## Required release and consumer validation

1. Finish core PR CI/independent evaluation and merge the reviewed implementation.
2. Release a supporting core version (proposed v0.54.0), with matching std/VERSION,
   folded changelog and version metadata. Tag must be on current dev HEAD, as the
   release-manager workflow requires.
3. Verify the release-tagged registry validator contains std/terminal. Its release-
   only rollout policy is in `cloudbuild-registry-validator.yaml`; do not bypass it
   or relax the package runtime floor.
4. With the released CLI/stdlib, repeat package quality/tests and publish 0.2.0.
5. From a fresh consumer directory, install `sunholo/terminal_ui@0.2.0` into an
   isolated bin directory. Run terminal-ui-demo --help and plain redirected EOF,
   then native arrow/resize/cleanup on a PTY from an unrelated cwd.
6. Record accepted registry identity and fresh-consumer evidence; only then finish
   M5, final evaluation and move the design plus sprint plan to implemented.

The core release tag also triggers the normal CLI/WASM/test deployment pipelines.
The package publication permission does not itself decide unrelated cloud approval
items, which remain untouched.

## Wider consumer control discovered during release preparation

The existing crew CI workflow was pinned to v0.52.0. Package PR #117 now proposes
v0.54.0, matching the manifest floor, and refreshes the crew consumer's path lock.
Its downloadable runtime remains dependent on the supporting core release.

Running the complete existing crew validation with the integrated compiler stopped
in the unchanged content-library package: frameTail line 71:344 has a missing/
malformed ApplicationEffects (LatentParamMask) invariant. The exact error also
occurs with an untouched current-dev acc227e55 build, with independent stdlib and
cache disabled. It predates the terminal implementation. The M4 integration
addendum scopes its repair to complete alias resolution at the row-unification
boundary; ApplicationEffects invariants and real-effect rejection remain intact.
Four/five/eight-arm recursion and effect/record alias controls pass. Evidence:
`/private/tmp/terminal-package-crew-integrated.log`
and `/private/tmp/terminal-content-library-dev-control.log`. Terminal-ui's own
39 tests, strict quality and PTY controls remain green. This wider consumer gate
must be resolved before claiming package repository CI/final release readiness.

## Windows CI repair

The original Windows effects job failed descriptor alias exclusion, followed by a
leaked test reservation. The repair uses borrowed HANDLE queries: disk files share
volume/file-index identity; pipe/character input uses handle identity without disk
metadata queries. Native mode remains Unsupported. The test now defers its lease
release immediately. Added Windows controls cover disk aliases/distinct identities,
handle lifetime, pipe blank/final-partial/EOF, invalid handles and NUL character
input. Windows/WASM effects tests compile, Windows vet, native race controls and
final lint pass. Independent review found no new defect. Actual Windows runtime CI
on repaired commit 274f0be88 passed (run 38037895269, test-windows); Linux tests,
lint, vulnerability, float determinism, CodeQL and platform builds also passed.
Original Windows log: `/private/tmp/terminal-ci-windows.log`.

## Coverage repair without changing the gate

The original PR Sonar report measured 69.1% new coverage; reliability, security,
maintainability, duplication and reviewed-hotspot conditions passed. Instrumented
Go-process PTY tests now exercise the actual terminal host while Python's standard
library holds the PTY master. Host coverage increased by 105 statements: terminal.go
215/232 (92.7%), terminal_posix.go 22/28 (78.6%), Darwin operations 7/7 (100%).
The effects race/coverage suite and vet passed. VM boundary coverage is 52/53
statements (98.1%); terminal ADT identity and new image admission/disassembly are
100%; EOF builtin logic is 10/11, with the remaining guard an init panic.
These controls assert errors, authority, ownership, events and restoration;
thresholds and exclusions are unchanged. Missing Python is a failure in CI.
Final Linux Sonar measurement remains required.
