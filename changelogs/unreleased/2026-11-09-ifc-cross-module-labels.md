### Fixed — security: IFC sinks and labelled results now survive module boundaries (M-IFC-CROSS-MODULE, #1134) (2026-11-09)

**Breaking for programs that relied on the gap.** A library's `{not ℓ}` parameter
refinement used to bind only inside the library: an importer could pass a `<secret>` value
to `lib.gate(input: string{not secret})` and compile cleanly, and a library function
returning `<secret>` handed its importer an unlabelled value. The importer's IFC check now
reads each imported function's exported IFC summary (M1–M2, PR #1771) and treats the call
like a local one:

- **Check A** at the imported edge (`sink_refinement`):
  `information-flow violation: value labelled <secret> reaches parameter "input" of imported function lib.gate, which forbids it (string{not secret})`,
  suggestion `edge: app.ail:5:38 → lib.gate(input) — declassify the value through a function declaring ! {Declassify}, or do not pass it across this edge`.
- **Check C** at the imported edge (`param_label_cover`) for a library parameter declared
  `string<sqlsafe>`.
- **Imported result labels** — a declared `-> string<secret>`, or the intrinsic label of an
  unannotated wrapper around `secret()` or around another library's labelled producer —
  taint the caller, so its own sinks reject the value.
- Selective, renamed and qualified (`import lib as L` … `L.gate(x)`) imports, aliases,
  higher-order parameters and local wrappers all reach the same check; a local declaration
  of the same name shadows the import. The browser/WASM REPL (`ModuleRegistry.LoadModule`)
  gives the same verdicts and publishes nothing from a rejected module.
- **Traces:** closes the gap M-TRACE-LABEL-AWARE documented in `std/secret` — a secret
  returned by a function in another user module is now redacted (canary
  `TestTraceRedactsSecretReturnedByAnotherUserModule`: 1 verbatim copy at `standard` and 5 at
  `deep` before, 0 and 0 after, cold and warm cache).

An import whose interface lacks valid IFC metadata is a compile error, never a clean callee.
Not exported: a sink demand the checker *infers* on an unannotated library parameter — declare
`{not ℓ}` on the exported parameter to enforce it across modules. Compile-cache key `v9` →
`v10`: every module recompiles once, because a cached caller is served without re-running the
check. All 891 tracked `.ail` files keep their `ailang check` verdict. Example:
`examples/runnable/ifc_cross_module/`; guide: `docs/docs/guides/ifc-labels.mdx`
("Imported functions"); design: `design_docs/implemented/v0_57_5/m-ifc-cross-module-labels.md`.
