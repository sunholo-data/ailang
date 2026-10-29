# M-DOCS-SEARCH-FALLBACK-CONFINEMENT — executor evidence

Issue #1965 only. Approved design and four-milestone plan are siblings of this file.
This is an implementation handoff, not an independent score or acceptance verdict.

## Source and scope

Source checkpoint `5a1a21267` contains all guard/CLI behavior and test changes. Its parents
bank M1 `9eee44912` and M2 `3630ec86c`; M3 is test-only. Actual dev
`6d0d3a9da2af2eadc504f7cb050aacde72cc3cca` was normally integrated before implementation.
Integrated source checkpoint `e9889a284ac4955768ccbcfaea01eea52edcb36b` normally
integrates accepted dev565fc1b04. That integration adds only the separately owned
runner test repair and its design/state; every cmd/ailang and docs backend byte is
identical to804502687, the corrected source used by the successful affected-package
run. No standalone runner gate is attributed to this sprint.

Final documentation/state changes include the approved CLI-reference wording in
its authoritative `cmd/ailang/cli_reference.go` constant and regenerated page.
The initial direct page edit was rejected by `TestCLIReferenceMatchesTable`;
that real failure is preserved before the generator correction. A final test-only cleanup synchronizes HTTP-handler counters with atomic reads;
its backend package rerun passes3.373s with focused lint/format checks. No guard,
search algorithm, dispatch or assertion policy changed after5a1.

Production changes are limited to the GitHub backend's narrow authority helper,
constructor, Search, direct cache writer, CLI refusal hints and generated CLI
reference prose. Existing
`pkg.Require`/`ConfinedError`, local SimHash/neural algorithms, policy-tool schema,
output caps, child-environment semantics and package-publish smoke behavior remain
unchanged. No .ail source, example, language prompt or compiler/cache identity changed.

| Design criterion | Discriminating implementation evidence |
|---|---|
|AC1 local SimHash|`TestDocsSearch_ConfinementLocalAndDisabled` runs actual confined CLI with explicit and discovered corpus, validates JSON title/path/score/order parity and unchanged corpus/private state; existing `TestDocsSearch_StillDesignDocSimHash` remains green. Paths identify the same real file after macOS symlink spelling normalization.|
|AC2 constructor|`TestDocsConfinementConstructorBeforeHelpers` proves fake gh and git execute unconfined, then absent/present-token confined arms return typed AuthNetwork with zero new helper calls. Source-order check places Require before token lookup/helper/git.|
|AC3 direct backend/cache|`TestDocsConfinementSearchAndCache` proves loopback request then valid cache hit before denying reused, test-constructor and manually populated/nil-client backends; direct writer refuses with AuthHomeWrite, including missing HOME, and full private state is unchanged. No cache read spy is claimed: pre-cache ordering is also established directly in source.|
|AC4 safe failures|`TestDocsSearch_ConfinementActualCLI` proves active fake helpers and deny-CONNECT proxy, then absent/dummy-token actual CLI arms fail canonically with no new helper/request/state effects. Disabled flag/env retain local failure without an unset/authentication hint; invalid explicit path remains local.|
|AC5 real tool child|`TestDocsSearch_ConfinementActualPolicyTool` starts the real policy-tool with parent policy marker absent; Host pins its actual child. Process exit0 with OK=false/child nonzero is distinct from actual local OK=true/child0; stdout remains valid existing JSON. Unknown neural/path flags are rejected without an argv.|
|AC6 unconfined/late transitions|`TestDocsUnconfinedPagination` proves two pages/101 results. `TestDocsConfinementHTTPAndWriteTransitions` prevents page2 after page1 installs policy and propagates typed home-write failure after an allowed response; ordinary unconfined cache error is still best-effort. Existing authenticated/repository/error/cache TTL/isolation tests stay green.|
|AC7 scoped docs|CLI reference, semantic-search guide, SECURITY and unreleased fragment state whole-GitHub denial including cache hits and local/operator remedy; explicit neural/cache management remain separately scoped.|

## Hermetic and portable instruments

The portable Go fake gh/git binaries never read actual credentials. Their log lives
outside the snapshotted HOME/work tree; each confined arm compares the baseline
log and request count. Actual CLI unconfined positives receive no token from gh;
a dummy token permits repository discovery and exactly one loopback CONNECT
attempt. That proxy always refuses and never forwards, so no live GitHub request
is needed even in counterfactual/red runs.

Tests clear inherited upper/lowercase proxy settings and bypass variables before
installing both cases of HTTP_PROXY/HTTPS_PROXY/ALL_PROXY. They also strip ambient
Google/cloud and OTEL/OTLP export environment. AILANG_TRACE=off is not relied on
as an exporter kill switch; #2009 is separate. HOME, USERPROFILE, home and XDG
config/cache/state/data paths are private. Full snapshots compare every relative
entry's type/mode and each regular file's content hash, rather than a short list
of known cache/device filenames. Filesystem access-time metadata is not asserted.

All new authority/process tests run on Windows, using .exe names and filepath
handling; none calls the old Windows-skipping restricted-runtime/symlink fixtures.
No real restricted AIL program or runtime descendant-termination claim is added.
Existing runCLIIn30s, helper-build60s and shared build120s bounds are unchanged.
Exact-head Windows CI remains required; local macOS proof does not substitute for it.

## Failing-first provenance

All raw logs below are retained under `/tmp/ailang-backlog-20261029/` in the
attended workspace. They are evidence references, not portable repository assets.

- `1965-M1-red.log`: real typed-refusal assertions fail before implementation for
  constructor, valid cache hit, pagination and post-response cache write; helpers
  and loopback are active. `1965-M1-green.log`: entire backend package passes.
- `1965-M2-red.log`: actual CLI already reaches the new canonical backend error,
  but still suggests auth; disabled fallback suggests unsetting controls.
  Those two real failures precede the narrow CLI hint repair.
- `1965-M3-red.log`: actual old-backend policy-tool child returns authentication
  failure instead of confinement. Scratch overlay and `manifest.json` under
  `1965-counterfactual/` identify exact pre-guard file bytes from dev6d0d3a9da.
  Only github.go/cache.go/docs_search.go were replaced in the current compiler;
  this does not claim whole-old-binary equivalence. No mocked runner is used.
- `1965-M2M3-green.log`: all TestDocsSearch controls pass11.128s; an earlier green
  attempt exposed only the existing macOS /var vs /private/var path spelling,
  corrected in the test with real-file canonicalization, without production change.
- `1965-M3-compatibility.log`: unchanged child-policy/CLI-output-cap/JSON-cap/path
  option contracts pass0.425s. No production policy-tool source was edited.

## Final validation and acceptance boundary

The first final affected-package run failed only `TestCLIReferenceMatchesTable`
because the edited mirror did not yet match its generator; all other tests ran.
`1965-M4-test-changed.log` retains that failure. The generator correction,
`make docs-cli` and focused reference gate are recorded in `1965-M4-docs-cli.log`
and `1965-M4-reference-green.log`; the repeated final affected/core gate is
`1965-M4-test-changed-final.log`. No thresholds or assertions were changed.

Final local gates pass: make build (`1965-M4-build-final.log`, stamped e9889a284),
changed packages (`cmd/ailang`203.543s, github3.890s) plus all20 core packages
(`1965-M4-test-changed-final.log`), unchanged policy-tool compatibility0.425s,
lint0/boundaries/home isolation/sizes/changelog/fmt (`1965-M4-structure.log`).
The generated prose correction has refreshed lint/fmt/sizes proof
(`1965-M4-generator-structure.log`); test-only counter synchronization has
backend3.373s proof (`1965-M4-counter-sync.log`) and focused lint/final fmt.
No full local make test is run.
Independent review, actual PR CI and root's final merge/issue decision remain
separate from this executor report. No thresholds, timeouts, policy, authority,
runner source, held coordinator cards or failure evidence are weakened here.
