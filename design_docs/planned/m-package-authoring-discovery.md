# M-PACKAGE-AUTHORING: Offline guidance and evidence inventory

**Status**: Implemented on feature branch; not released (user approved followups 2026-09-08)
**Target**: next release; P1; estimated one focused sprint (~400 LOC plus tests)

## Problem and systemic audit
The Discord demo compiled and passed integration checks but its new packages lacked native contracts/tests and the agent missed the existing package skill. Consumer repo instructions did not route to the binary first. The existing skill validator runs only lock and check. `publish --dry-run` returns before remote validation (`cmd/ailang/pkg_publish.go`). Fix discovery and evidence reporting across consumers, rather than another Discord-specific workaround.

## Design
1. Embed a short authoring guide in `ailang docs package-authoring`; no source checkout, stdlib lookup or network needed. Cover full prompt discovery, package docs, pure logic/contracts, tests/properties, effect ceilings/budgets, validation and publishing.
2. Add `ailang pkg quality [--json] [--strict] [DIR]`: read manifest and existing AST only; inventory native tests, pure-function contracts, explicit unbudgeted effects, and AGENT.md. Default reports; strict rejects missing evidence. Parse/read/manifest errors always fail. This is an inventory, never proof, coverage or test execution. Compiler and native test commands remain separate gates.
3. Strengthen both core package skill variants and their validation scripts. Install the agent variant globally for local discovery; add portable consumer skills and AGENTS.md in demos and packages.

## Boundaries and risks
No grammar, typechecker, runtime, publisher or registry policy changes. No automatic network use or test execution in quality. Contract presence cannot establish usefulness; inferred effects cannot be established from syntax. Report that limitation. Native declaration counts cannot establish behavioral coverage. Strict is an opt-in authoring policy, not a publication rule. Existing packages may fail it; do not retrofit meaningless contracts to green the report.

## Verification log and related documents
- Inspected AST FuncDecl.Tests, Properties (RequiresKind/EnsuresKind vs PropertyKind), Effects.Budget and package discovery API.
- Existing docs search performed locally; top SimHash matches were unrelated billing responsibilities, early implementation report, rejected aliases (not neural duplicates).
- `implemented/v0_10_0/m-dx-package-test.md` supplies test execution; this adds evidence inventory, not another runner.
- Local guides/packages.md, testing.md, contracts.mdx and reference/capability-budgets.mdx remain detailed references.

## Axioms
A1 +1 deterministic sorted output; A2 +1 repeatable report; A3 +1 explicit effect-budget gaps; A4 0 read only; A5 +1 bounded inventory distinguished from proof; A6 0; A7 +1 JSON; A8 0 no syntax; A9 0; A10 +1 composes existing check/test; A11 +1 explicit gaps/errors; A12 0. Net +7, no hard violations.

## Sprint and acceptance
- M1 (~90 LOC): offline embedded guide and help routes. Verify outside checkout without stdlib.
- M2 (~300 LOC plus tests): AST inventory and strict/JSON modes. Fixtures: comments containing fake evidence, empty tests, valid inline tests/contracts, package test functions, malformed source, private helpers, unbudgeted effects. Verify existing Discord/AGUI gaps are visible.
- M3 (~100 LOC docs/scripts): consumer routing, globally discoverable skill, validator does not call compilation alone complete. Check shell syntax and skill metadata.
- M4: targeted Go tests/build, smoke new CLI, record limitations and review diff. Preserve unrelated changes in all checkouts.

## Evaluation
M1–M4 complete. Eight quality tests plus adjacent docs/signature tests pass; `go vet ./cmd/ailang` passes. Temporary binary built and guide read from /private/tmp with nonexistent stdlib path. Native fixture: one compiled module; two inline examples plus contract property execution = 3 passed, 0 failed, 0 skipped. Strict inventory passes. Real packages: Discord 19 gaps, AGUI 8; both have zero native tests/properties/contracts, as expected. No runtime/code retrofit or package publication performed.

Consumer AGENTS.md and portable skills updated in demos and packages. Core .agents/.claude skill variants and validators updated; .agents variant installed globally for future local sessions. Shell syntax and diff checks pass; official skill validator unavailable because PyYAML absent (frontmatter/references inspected manually). Canonical GCP inbox listing timed out, no read/ack performed. Full repository suite/coverage were not run; focused CLI regression checks and vet cover this change. Existing Observatory startup retention warnings remain unrelated.

Known scope limits: syntax cannot infer effects, judge assertions, or establish behavioral coverage. Strict is an opt-in policy; declared evidence is never reported as proof. Reports show execution not_run and explicit limitations in JSON. Guide command works offline but existing global startup hooks may emit Observatory warnings.
