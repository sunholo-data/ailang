### Added: re-exports (`export import`)

Phase 2 of [M-PKG-PARALLEL-AGENTS](../../design_docs/planned/v0_63_0/m-pkg-parallel-agents.md): a module can re-export names it imports, so a module can move without breaking the modules that import it.

- **`export import M (names)`** in the import header makes a module a facade. The listed names become its exports with the original's identity: a constructor imported through a facade is the original's, so values built on either side match the other's patterns, on the evaluator and the strict VM. Names can be renamed with `as`, and facades can chain.
  - A dependent package needs no direct dependency on the package a facade re-exports from.
  - Through a module alias (`import facade as F`), re-exported functions and types are reachable as `F.f` and `F.T`.
- **A facade's interface shows its re-exports as the origin declares them** (type, constructors, effects, contracts). A module moved behind a facade therefore keeps its signature lines and measures as no removal, while dropping a re-export measures as class C. The facade's interface digest covers its re-export list, so a cached dependent recompiles when the list changes.
- **`@deprecated("new/path")` on an `export import` line** marks it as moved. Every name imported through it warns `TC_DEPRECATED_001` at the name, with the fix `import X from new/path`. This covers types and constructors as well as functions.
- **Effect ceiling:** a facade package's `[effects].max` must cover the functions it re-exports. A violation names the function and where it is defined.
- **New diagnostics:**
  - `MOD016`: a re-export cycle, naming the chain.
  - `PAR_REEXPORT_FORM`: `export import` without a name list, or with `as`.
  - `PAR_REEXPORT_ANNOTATION`: an annotation other than `@deprecated` on the line.
  - Re-exporting a name the source does not export is `IMP010`.
- **Removed `MOD003`** ("Re-export not supported"). It was defined but never emitted.
- New example: `examples/runnable/reexport_facade/`.
- **Known limitation, unchanged:** qualified constructor access through a module alias (`C.Red`) is `IMP010` for every module, facade or not.
