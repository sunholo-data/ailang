# Native terminal sprint validation and delivery

Implementation: core PR [#1756](https://github.com/sunholo-data/ailang/pull/1756),
package PR [#117](https://github.com/sunholo-data/ailang-packages/pull/117).
Core feature commit 2f807c563; integrated current dev acc227e55 at cdcea3bdb.
Package feature commit c0ec852, integrated compiler evidence e52ae17.
Local implementation validation passes. Independent technical review: 82/100 plus 10 regression bonus points, with no remaining implementation defect found. CI/final delivery and required release/
publication/registry consumer checks are pending; this is not a completed sprint.

## Concrete evidence

| Check | Result |
|---|---|
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
