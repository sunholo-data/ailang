Huge win. With A1 green, you’re ready to harden and de-paper-cut the module path. Here’s a tight A2 plan (small cuts, maximum unlock) focused on conflict detection, crisp diagnostics, and determinism.

Milestone A2 — Conflicts & Diagnostics

1) Import conflict detection (IMP011)

Goal: When two imports expose the same name into the same scope, fail deterministically with a trace + hints.

Where: internal/link/module_linker.go (during BuildGlobalEnv)

Sketch:

// in BuildGlobalEnv(imports []ast.ImportDecl) (GlobalEnv, LinkReport, error)
seen := map[string]GlobalRef{} // local symbol -> provider
for _, imp := range imports {
    prov := ifaceRegistry[imp.ModuleID] // already built in A1
    for _, sym := range imp.Symbols {   // selective imports only
        if prev, ok := seen[sym]; ok {
            return nil, LinkReport{
                Code: "IMP011",
                Message: fmt.Sprintf("import conflict for '%s'", sym),
                Trace: []string{
                    fmt.Sprintf("%s provides %s.%s", imp.ModuleID, imp.ModuleID, sym),
                    fmt.Sprintf("conflicts with %s.%s", prev.Module, prev.Name),
                },
                Suggest: []string{
                    "remove one import",
                    fmt.Sprintf("rename provider module path (e.g., split exports in %s)", imp.ModuleID),
                },
            }, NewErrorIMP011(sym, imp.ModuleID, prev.Module)
        }
        // record and wire
        seen[sym] = GlobalRef{Module: imp.ModuleID, Name: sym}
        env.Values[sym] = seen[sym]
        env.Types[sym] = prov.Exports[sym].Scheme // already generalized
    }
}

Tests:
	•	examples/v3_3/import_conflict.ail importing (gcd) from two modules → IMP011 stable JSON.
	•	Golden includes trace (ordered), suggest array, and canonical module IDs.

Acceptance: make test-imports includes a failing conflict case that asserts IMP011 JSON.

⸻

2) Missing symbol / module diagnostics (IMP010, LDR001) with search trace

Goal: When a symbol or module is missing, show where we looked and a fix suggestion.

Where:
	•	internal/link/module_linker.go → missing export (IMP010)
	•	internal/loader/loader.go → module not found (LDR001)

Sketch (module not found):

return nil, fmt.Errorf(encoder.JSON(
  Error{
    Code: "LDR001",
    Message: "module not found",
    Data: map[string]any{
      "module": modID,
      "search_trace": []string{
        fmt.Sprintf("relative: %s", relPath),
        fmt.Sprintf("stdlib: %s", stdPath),
        fmt.Sprintf("project: %s", projPath),
      },
      "fix": Suggest("Check file exists and module name matches path", 0.85),
    },
  },
))

Tests:
	•	tests/errors/lnk_unresolved_module.ail → LDR001 with search_trace length ≥ 3.
	•	tests/errors/lnk_unresolved_symbol.ail → IMP010 lists requested symbol and provider module’s available exports (sorted list).

⸻

3) Deterministic interface & trace ordering

Goal: Absolute determinism across OS/env.

Where: internal/iface/builder.go, internal/link/*

Actions:
	•	Sort export names before digest (you already do).
	•	Sort Imports in iface and any trace arrays lexicographically.
	•	Normalize module IDs to canonical form before storing anywhere (already A1, verify all callsites).

Tests:
	•	make test-builtin-freeze remains; add make test-iface-determinism that runs iface dump twice with different TZ, LC_ALL, PATH and compares.

⸻

4) REPL/file parity checks for imports

Goal: Same pipeline, same results.

Where: internal/repl/repl.go (ensure it calls the same Pipeline.Run path with a synthetic module and imports allowed).

Test:
	•	tests/parity/imports_here_doc.sh pipes:

:set module examples/v3_3/imports_basic
show(gcd(48,18))

Compare output to bin/ailang run examples/v3_3/imports_basic.ail. Golden must match exactly.

⸻

5) Guard recursive/value initialization order

Goal: Keep your “capture by reference” invariant safe.

Where: internal/eval/eval_core.go

Action:
	•	Ensure EvalLetRecBindings populates the environment with thunks before evaluating RHS, and cycle detection gives RT_CYCLE with a short path.

Test:
	•	tests/recursion/mutual.ail (even/odd across one file) → works.
	•	tests/recursion/self_bad.ail where RHS immediately forces itself → RT_CYCLE JSON.

⸻

6) Makefile/CI additions (extend A1 gate)

Makefile:

test-imports: ## Run import examples and conflict/error goldens
	@echo "== imports_basic =="
	@bin/ailang run examples/v3_3/imports_basic.ail --json --compact | diff -u goldens/imports_basic.json -
	@echo "== imports (multi-module) =="
	@bin/ailang run examples/v3_3/imports.ail --json --compact | diff -u goldens/imports.json -
	@echo "== import conflict =="
	@bin/ailang run examples/v3_3/import_conflict.ail --json --compact | jq -r .code | grep -q '^IMP011$$'

test-link-errors:
	@echo "== missing module (LDR001) =="
	@bin/ailang run tests/errors/lnk_unresolved_module.ail --json --compact | jq -r .code | grep -q '^LDR001$$'
	@echo "== missing symbol (IMP010) =="
	@bin/ailang run tests/errors/lnk_unresolved_symbol.ail --json --compact | jq -r .code | grep -q '^IMP010$$'

ci-strict: verify-lowering test-builtin-freeze test-imports test-link-errors

CI YAML: add make ci-strict step (you already run A1 gate—append these targets).

⸻

Golden files you’ll want to drop in now
	•	examples/v3_3/imports_basic.ail (already done) with golden goldens/imports_basic.json → "6\n" or your JSON-wrapped stdout.
	•	examples/v3_3/imports.ail (two providers) with golden output.
	•	examples/v3_3/import_conflict.ail → expects IMP011 (no stdout).
	•	tests/errors/lnk_unresolved_module.ail → nonexistent module path.
	•	tests/errors/lnk_unresolved_symbol.ail → import a real module but request (notThere).

⸻

Small correctness nits to sweep now
	•	Export underscore rule: forbid export func _private with MOD_EXPORT_PRIVATE (+ suggestion “remove underscore or drop export”).
	•	Namespace imports: you added IMP012_UNSUPPORTED_NAMESPACE—great; ensure the error message suggests selective form import foo/bar (x, y).
	•	Normalize NFC at lexing: ensures module IDs in traces match file headers even with weird Unicode.

⸻

Acceptance for A2
	•	test-imports and test-link-errors pass in CI.
	•	Deterministic traces (sorted) in goldens.
	•	Conflict and missing cases emit structured JSON with helpful suggest and canonical module_id.

Once this is green, you’ll have “import ergonomics” that feel production-ready: clear failures, stable interfaces, and zero mystery. Then we can roll into A3 (namespace/import alias design, stdlib search path, and :dump-iface/:why hooks).