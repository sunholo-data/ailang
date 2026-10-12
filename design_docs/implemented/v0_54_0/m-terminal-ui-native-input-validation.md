# Native terminal sprint validation and delivery

Implementation: core PR [#1756](https://github.com/sunholo-data/ailang/pull/1756),
package PR [#117](https://github.com/sunholo-data/ailang-packages/pull/117).
Core feature commit 2f807c563; final reviewed head beb681d76 passed every required
CI check (run 38047208336), including actual Linux/Windows tests, all platform
builds and Sonar. Core PR #1756 merged at cc0bf4453 on 2026-10-10. Current release
includes the concurrent deployment-only dev merge 361caeda1. Supporting release
v0.54.0 is public at that exact commit; release-head CI 38048763041 and signed
CLI/WASM/provenance workflow 38050365288 passed. The released registry validator
reports v0.54.0 at the same full commit. Package feature commit c0ec852 is
integrated with current package main 9437336 at final delivery head 85ccdecb.
The original Sonar coverage rejection was repaired with meaningful controls;
final new-code coverage is 89.7% against the unchanged 80% threshold.
M4 technical and consumer gates pass. Package CI 38050960642 passed all crew,
eight behavioral mutation and social regression gates at the exact final head.
PR #117 merged at acfe51adcbb8905a126128f016b09e394b05b4a1 on 2026-10-10.
Publication was accepted; the fresh registry consumer passed all checks below.
Independent round2 PASS (82/100 plus 10 regression points, 92/110) found no
implementation or delivery blockers. Final artifacts complete M5 and are archived
with this validation record. Clerical deductions reflect the evaluation snapshot
before archival; the historical evaluation is preserved.
The user explicitly approved both PR merges, the core release and publication.

## Concrete evidence

| Check | Result |
|---|---|
| Release preflight | All five gates pass: tests, lint, file sizes, import goldens and eval configuration; `/private/tmp/terminal-pre-release-checks-final.log` |
| v0.54.0 metadata checks | Post-update full make test and lint exit 0; boundaries and changelog/design references pass |
| Supporting released CLI | Public v0.54.0 at 361caeda1; matching CI 38048763041 and signed release/provenance workflow 38050365288 passed; all platform archives, WASM, bootstrap and SLSA assets present |
| Released validator | Build aab70f9f-bd2c-4122-a848-403f2cfb8efc SUCCESS; /version reports v0.54.0 and full commit 361caeda13830b0a3faa11919cc2b5d37a14dde1 |
| Cloud TEST release | Build 72de552e-f408-42a2-910c-6e13d95dbd14 SUCCESS; all 16 images, resident acceptance, deployment and smoke gates passed |
| Guarded PROD promotion | Dry-run 45389ccd-3e71-43f2-9dab-499f03a4e90c and promotion 71ada3e1-589d-4983-b4fc-e1c4ba07ec1d SUCCESS; standard versioned image copy/digest pin/roll path; public MCP latest 0.54.0, `/private/tmp/terminal-v54-prod-mcp-versions.json` |
| Release brain refresh | Exact linked release checkout, rerun after public release: 68 syntax, 394 builtins and 250 example frames; active prompt v0.16.7; `/private/tmp/terminal-v54-brain-index-final.log` |
| Final package CI | [Run 38050960642](https://github.com/sunholo-data/ailang-packages/actions/runs/38050960642) SUCCESS at 85ccdecb; complete crew, 8/8 compiling behavioral mutation controls and social regression; `/private/tmp/terminal-package-final-ci.log` |
| Accepted registry publication | `sunholo/terminal_ui@0.2.0`, actual publish exit 0; `/private/tmp/terminal-ui-v02-published.log` |
| Fresh registry consumer | Official v0.54.0; real registry installation into isolated bin directory, all archive bytes and identities verified; redirected and native PTY checks pass; `/private/tmp/terminal-registry-consumer-keiwz44a/evidence/report.json` |
| Official package gates | Checksummed darwin arm64 archive b67b8931875e8a6dd96bc34a6b912657c25b08905f817d20a15cb484c1d4580e; exact-head evaluator/strict VM 39 passed each (37 named VM bodies, zero fallback), strict network quality 0 gates; embedded stdlib traced; `/private/tmp/terminal-v54-official/` |
| Final reviewed-head CI | Run 38047208336 at beb681d76: Linux/Windows, lint, vulnerability, CodeQL, float determinism, docs and all platform builds passed |
| Final reviewed-head Sonar | All conditions passed; 89.7% new coverage, 0% duplication and 100% reviewed hotspots; `/private/tmp/terminal-sonar-final-integration-gate.json` |
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

## Initial publication failure and accepted publication

Actual authorized publication on 2026-10-10 reached the registry and was rejected.
Validator `/version` reported v0.53.2, commit bebfe6818. Local quality and dry-run
passed. Remote compilation passed 8 pure files and failed `_smoke.ail`,
`adapter.ail`, `adapter_test.ail`, `demo.ail` because `std/terminal` is absent.
Full response: `/private/tmp/terminal-ui-publish-attempt.txt`.
That initial request accepted no package version. The original package CI likewise
failed because the released compiler lacked std/terminal. Both failures established
the supporting-core-release dependency; the runtime floor stayed >=0.54.0.

After the supporting CLI and validator release, actual publication succeeded:
`sunholo/terminal_ui@0.2.0`, `/private/tmp/terminal-ui-v02-published.log`.
The exact final package CI also passed before PR #117 merged.

Archive content hash: `sha256:9d041ee63b3aa5768af55cb3bf03e722a36a1d825957223f1b105f851f5ed4a8`.
Interface v2: `e6b520bcefb43b7d8275594cad099203a701bb154dd53980ad8bddb6d908aab2`.
Validation documentation is excluded from the source archive identity.
Compressed registry archive: 18070 bytes,
`sha256:f2d1d92914a7fe231faa3e68031278fcb7151f5f3204a2aaef3b88fb84b6c606`.
Legacy interface: `sha256:131db35611e00407e57cbe851f3d9b3ff6686747d366f2cd99e9d83efb4a6768`.

## Completed release and consumer validation

1. Reviewed core head passed required CI and merged through PR #1756.
2. v0.54.0 released at current dev HEAD with matching metadata, signed assets and SLSA provenance.
3. Release-tagged registry validator contains std/terminal; its release-only rollout policy was followed.
4. Official released CLI passed exact package tests/quality; publication accepted 0.2.0.
5. Fresh unrelated consumer installed the real registry package into an isolated bin directory and passed --help, line/plain/auto blank/final-partial/sticky-EOF checks and native arrow/resize/cleanup.
6. Accepted identities and fresh consumer evidence are recorded; final evaluation governs M5 closure and archival.

Fresh consumer evidence authenticates the official archive/checksum/binary bytes,
registry metadata, compressed archive and all 15 installed archive members. The
shim anchors the exact official binary and registry cache with main and IO,Env.
Down changes selection before Enter; Enter confirms item 2. A 20x8 viewport fits,
80x24 preserves state, and q exits 0. Post-exit termios matches completely except
the documented transient PENDIN mask; cursor and alternate screen are restored.
Redirected plain/auto contain no ESC. Final log:
`/private/tmp/terminal-registry-consumer-final.log`.

Earlier fixture failures remain in the evidence trail: Python's default CA bundle
failed verification, corrected by using the installed system CA with TLS verification
retained; install flags must precede the package positional argument; and the macOS
PTY fixture was changed to retain the parent controller/slave in the inherited
session, matching the core CLI harness. Full restoration assertions were retained.
No production implementation change was needed for these fixture corrections.

The core release tag also triggers the normal CLI/WASM/test deployment pipelines.
The package publication permission does not itself decide unrelated cloud approval
items, which remain untouched.

## Wider consumer control discovered during release preparation

The existing crew CI workflow was pinned to v0.52.0. Merged package PR #117 pins
v0.54.0, matching the manifest floor, and refreshes the crew consumer's path lock.
Its final CI downloaded the supporting official release successfully.

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
39 tests, strict quality and PTY controls remain green.

The scoped row-alias repair now resolves that consumer failure without weakening
ApplicationEffects invariants. Full crew validation passed with 149/149 crew and
14/14 content-library tests in evaluator and strict VM, identical replay/parity,
installed CLI/store/play/journey/watch flows and terminal package quality. All
eight mutations compiled and were caught by behavioral assertions. The existing
social lab passed 117 package tests, installed CLI, five replay/parity recordings,
strict quality and publication dry-run. Watch used zero live provider calls and
one deterministic AI stub attempt. With `AILANG_NO_CACHE=1`, crew took about
20m56s, mutations 21m57.09s and social 2m42.59s: about 45m36s together. Package CI
retains every gate and allows 60 minutes rather than 25. Local development logs:
`/private/tmp/terminal-crew-row-repair.log`,
`/private/tmp/terminal-crew-mutations-row-repair.log`,
`/private/tmp/terminal-social-row-repair.log`. Released-runtime package CI 38050960642 subsequently passed all three
complete consumer gates at the exact merged package head.

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
Final Linux Sonar passed at both 02d174c54 and beb681d76 with 89.7% new coverage.

## Final evaluation and scope

Round2 report: `.ailang/state/evaluations/eval_M-TERMINAL-UI-NATIVE-INPUT_round_2.json`.
Its independent verdict is PASS, 82/100 plus 10 regression points (92/110).
All milestone state is completed and the design, plan and validation travel together
in implemented/v0_54_0. Round1 is preserved as a historical pending verdict.

General worker report inbox_1791630246888_28589e92 remains unresolved. Separate
provider-free reproductions on a local exact-tag v0.54.0 build leave both managed
and async process workers alive after host exit. No AI-call cancellation test or
fix is claimed. This requires a dedicated runtime lifecycle/cancellation follow-up.
