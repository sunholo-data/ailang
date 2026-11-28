# Sprint Plan: M-PKG-PARALLEL-AGENTS Sprint 4 (Phase 4: module moves)

**Design:** [M-PKG-PARALLEL-AGENTS, C5 and Phase 4](m-pkg-parallel-agents.md#implementation-plan)  
**Sprint JSON:** `.ailang/state/sprints/sprint_M-PKG-PARALLEL-AGENTS-S4.json`  
**Created:** 2026-11-27  
**AILANG prompt version loaded:** v0.16.20

## Scope and readiness

- The design freeze ratified D1–D4 and D7 on 2026-11-25. D6 and checkpoint GC are agent-resolvable. There are no open human decisions for this sprint.
- Phase 1 shipped in v0.64.0. Phase 2 merged in [#2475](https://github.com/sunholo-data/ailang/pull/2475). The planning checkout is at that merge, with `std/VERSION` at v0.64.1. The status refresh is docs-only and was open at planning time. [#2483](https://github.com/sunholo-data/ailang/pull/2483)
- Sprint 4 uses path dependencies. Sprint 3 owns the two C5 follow-ups: write `resolve = "checkpoint"` when the source uses checkpoints, and apply PUB026 to checkpoint dependencies during publish. Checkpoint and lease work can proceed independently.
- Ship the local move, the target-first minor release, the later major release with mechanical migration metadata, deterministic-first downstream repair, and a package-publishing walkthrough.

## Estimate and sequencing

**Three working days, six executor sessions of 60–90 minutes each, about 1,270 changed lines including 520 lines of tests and fixtures.** Risk is medium-high across filesystem edits, class measurement and the cascade transport. The design estimates two to three days. The velocity script found only the #2475 merge in this shallow checkout, so the prior plans provide the baseline: Sprint 1 budgeted about 930 lines across six sessions, and Sprint 2 tracked 730 estimated lines across six passing milestones. These are planning estimates. Use the upper end of the C5 duration and reserve time on day 3 for integration and gate failures.

| Day | Sessions | Exit point |
| --- | --- | --- |
| 1 | M1, M2 | A local move and multiple sequential moves compile with compatible facades. |
| 2 | M3, M4 | The source minor release measures B and requires the exact target version. The major release produces a validated C migration-request. |
| 3 | M5, M6 | The wrapper repairs the dependent or escalates on failure. The two-release fixture, guide and gates are green. |

## M1_MOVE_METADATA — Resolve the move and persist its identity

**Dependencies:** Phase 1 and #2475. **Estimate:** 120 implementation + 70 test lines.

- Define and validate `[[migrations]]` in `internal/pkg/manifest_migrations.go`, using `from, to, symbols, to_package, since`. Store canonical external import paths, stable unique public names and the source's staged move-release version. Document bumping that version before moving. Validate path and package consistency, symbols, version and duplicate source entries. Old manifests with no entries remain readable.
- Resolve the source module through its owning manifest, require an actual manifest at the target directory, and derive the destination file and declared module from its target module prefix. Reuse `ResolveModuleToFile` and the manifest import mapping functions. For example, `docparse/types/document` in `sunholo/ailang_parse` is imported as `pkg/sunholo/ailang_parse/types/document`.
- Build a read-only preflight that collects the module's complete public interface (functions, types, constructors and existing re-exports), checks destination and manifest conflicts, and determines source-relative path dependency edits. Refuse unresolvable imports or dependency cycles before edits. For a group of modules, plan dependency-first ordering and use existing moves as the source of already-relocated paths.
- **Acceptance:** Valid metadata survives load and append across multiple moves. Invalid target, missing source, collision, conflicting dependency and malformed metadata have targeted tests. A nested target with a distinct prefix maps to the expected file and import path.

## M2_MOVE_AND_REWRITE — Apply the move and preserve old imports

**Dependencies:** M1. **Estimate:** 150 implementation + 100 test lines.

- Add CLI routing and help for `ailang pkg move <module> --to <pkgdir>` in `cmd/ailang/pkg_move.go`. Account for positional-first flag parsing. Stage the destination, both manifests and import edits before applying. Keep the old module in source exports as the facade, add the new module to target exports, and add or reuse a relative target path dependency. Preserve unrelated TOML, file modes and existing exported modules. Write the facade at the old path using C1's explicit full name list and `@deprecated("<new path>")`. A bare whole-module re-export raises PAR_REEXPORT_FORM.
- Share parsed import-path editing between the local move and M5, with source-span edits that preserve selective lists, symbol aliases, module aliases and comments. Resolve package ownership per file, including nested packages, and select local or `pkg/` spelling accordingly. Declare the target in each in-repo package whose changed imports need it to avoid LDR006. Include imports inside the moved module and relevant existing generated facades. Qualified constructors through aliases remain IMP010. Exercise selective constructor imports and function/type access through module aliases.
- Preflight the complete edit set, stage writes and use the existing atomic file writer with rollback of touched paths on apply failure. Detect conflicting edits and safely handle retries. Avoid writing partial moves, crossing package boundaries by accident, or replacing unrelated user edits.
- **Acceptance:** The nested fixture has both manifests, the destination declaration, a complete facade and valid rewritten imports. Comments and string lookalikes survive. The old facade still works for an external consumer. Two sequential local moves preserve prior migrations, and a failing edit leaves the fixture consistent.

## M3_MOVE_RELEASE — Enforce target-first publishing and measure B

**Dependencies:** M2. **Estimate:** 105 implementation + 85 test lines.

- Introduce PUB027 in `cmd/ailang/pkg_publish_moves.go` and register the code through the existing error-code workflow. Check each live move target's exact publish-bound version against the registry before quality builds, tarball or upload. Reuse the version resolved by the existing path-to-registry conversion. Distinguish an absent version from a registry lookup failure. Verify multiple moves to one package use the same registry lookup and preserve the working manifest on refusal. Exercise the dry-run path.
- Phase 1 returns A for equal signatures. Reuse that classifier, then treat a measured A as B only for a verified newly introduced live move (current release matches the recorded `since`, the old and new public names and contracts agree, and the deprecated facade points at the declared target). Keep ordinary equal-signature releases A, and preserve measured C and U. Normalize this decision once for PUB025, legacy inbox and cascade-topic emission. Reuse the existing effect-widening and other cascade guards.
- **Acceptance:** TST-PKGPAR-26 publish-order and classification assertions. Source-first or an older target version yields PUB027, target-first permits a source minor, and notification class agrees with publish quality. Cover a later release with a retained migration entry, an unrelated equal-signature release, and a real breaking change alongside a move.

## M4_MIGRATION_EVENT — Emit the facade-removal request

**Dependencies:** M1 for schema, M3 for publisher integration. **Estimate:** 100 implementation + 80 test lines.

- Add an optional typed `import_rewrites` list to the existing `migration-request` envelope. Each entry carries `from, to, symbols, add_dependency: {name, constraint}`. Reuse package path and dependency constraint validation as layer boundaries permit. Reject empty or invalid paths, inconsistent destination/dependency names and conflicting rewrites. Retain existing version and block-reason requirements and compatibility with requests without rewrites.
- Measure the source major release as C using old/new signatures. Derive rewrites from persistent manifest entries whose facade just disappeared, using previous exports/signatures and current package-owned files/exports to distinguish removal from older history. Use the target dependency's publish-bound constraint. Retain the records through notification and avoid re-emitting old removals on subsequent releases. Emit a single logical request per dependent through the existing inbox and cascade channels, with the same kind, class and rewrite data. Preserve ordinary C notifications when no facade was removed.
- **Acceptance:** TST-PKGPAR-11 rejects empty `from` or `to` in an otherwise valid request. Valid payloads round-trip, legacy requests still decode, and publishing the major emits the expected typed request. A live facade, unrelated C change and a previously removed facade do not generate rewrites.

## M5_DETERMINISTIC_MIGRATION — Carry the event to the wrapper and repair imports

**Dependencies:** M2 and M4. **Estimate:** 195 implementation + 115 test lines.

- Extend the typed cascade data through Pub/Sub decoding, coordinator tasks and their SQLite/Firestore persistence, dispatch parameters, and cloud job config. Follow the existing `RootChangeClass` path. Test the round trip from published event to wrapper input, including the legacy payload shape, and preserve the trusted cascade ingress and existing execution guards.
- Update `classifyDispatchPath` and its caller in `coordinator_cloud_task.go`. A valid C `migration-request` with non-empty rewrites takes the deterministic path. A/B retain their existing route, and all other C cases retain AI dispatch. Invalid rewrite data must never drive filesystem edits.
- Implement the C operation in `cmd/ailang/coordinator_cloud_migration.go`. Update the source version and add the target dependency if missing before changing imports. Preserve compatible existing constraints and surface conflicting ones. Use the shared editor within the consumer package, then run `ailang lock`, `ailang check --package .` and package tests. Use a package-test fixture with a discoverable `*_test.ail` to prove tests ran. Commit only after the checks pass. On failure, restore deterministic edits and lock before handing the original migration context to the existing AI fallback. Keep retries safe for already-applied manifest and import edits.
- **Acceptance:** TST-PKGPAR-12 runs the real dependency-add, rewrite, lock, check and test sequence and observes zero AI dispatches. TST-PKGPAR-27 covers C requests with rewrites, the same request without rewrites, other C kinds and a failed deterministic check followed by one AI dispatch. Inject a failed step and verify no partial commit.

## M6_E2E_AND_GUIDE — Verify both releases against the docparse split shape

**Dependencies:** M2–M5. **Estimate:** 80 guide/changelog/example + 70 test lines.

- Add an isolated scratch-registry fixture under `cmd/ailang/testdata/pkg_move/` or generate the same package layout in `cmd/ailang/pkg_move_e2e_test.go`. Model the 46-module split from `sunholo/ailang_parse` into nested `packages/docparse_core`, using a small genuine ADT with exported constructors and a dependent that selectively imports and pattern-matches them. Include more than one interdependent moved module and exercise repeated single-module commands in dependency order. As a scale acceptance run, replay the 46-module list in a scratch checkout of the pre-split project and record failures by module/dependency edge. Keep the automated fixture independent of that external checkout. [docparse_core split, #183](https://github.com/sunholo-data/ailang-parse/pull/183)
- **TST-PKGPAR-26:** Publish the baseline, attempt the source minor before target publication, then publish target and source in order against the fixture registry. Resolve the new source minor with ordinary `ailang lock` in a dependent that declares only the source. Assert the flat lock includes the target transitively, the old import produces `TC_DEPRECATED_001` with the new path, and constructor identity matches on the evaluator and strict VM. Then remove the facade and its manifest export, publish the source major, apply its actual event through M5, and confirm the dependent works on both engines with a declared direct target dependency.
- Extend [Moving a Module Without Breaking Dependents](../../../docs/docs/guides/package-publishing.md#moving-a-module-without-breaking-dependents) with the `pkg move` invocation, generated facade, target-first order and PUB027, minor/major sequence, transitive lock and downstream migration. Preserve the effect-ceiling note. Use the fixture as the tested example behind the walkthrough and validate the guide's AILANG block. Add a dated `changelogs/unreleased/` fragment opening with `### `.
- **Acceptance:** TST-PKGPAR-26 passes end to end, the guide uses the command, and the validation gates below pass.

## Validation and syntax checklist

- Run `make test-changed`, `make lint`, `make check-file-sizes`, `make check-changelog`, `make verify-examples` and `make check-doc-samples`. Run targeted tests for the changed CLI, messaging, package, coordinator and persistence paths. Pin parity to evaluator and strict VM with fallback disabled.
- Map executable tests to TST-PKGPAR-11, -12, -26 and -27 in their names or comments. Cover successful and rejected branches in each modified path. `pkg_commands_test.go` is already 788 lines. Put move, publish and migration tests in companion files. Use `pkg_publish_moves.go`, `manifest_migrations.go` and `coordinator_cloud_migration.go` to keep feature logic out of large existing files.
- The guide's facade example delegates its fixture behavior to the origin and dependent. **Contracts:** skip for the facade, which declares no function. Include a contracted public origin function in the fixture to prove contracts survive the move. **Effects:** skip for the facade and pure ADT module, which need no effect row. **Inline tests:** skip for the facade, which declares no body. Use a discoverable dependent test module to exercise package tests and ADT identity.
- Registry searches for `module move`, `import rewrite` and `package publish` returned no packages. Registry reuse for M1–M6 is `none`. These milestones extend Go CLI, package, messaging and coordinator internals and their integration coverage. The fixtures use the existing C1 and registry test infrastructure.
