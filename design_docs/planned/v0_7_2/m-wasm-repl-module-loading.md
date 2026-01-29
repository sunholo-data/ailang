# M-WASM-REPL: Browser Module Loading for WASM REPL

**Status**: 📋 PLANNED
**Priority**: P1 (High - blocking browser demos)
**Estimated**: 400 LOC (300 impl + 100 tests)
**Duration**: 3-4 days
**Dependencies**: None (WASM build already working)
**Created**: 2026-01-28
**Blocking**: Invoice processor demo, browser-based AILANG demos

## Problem Statement

### The Core Issue

**AILANG's WASM REPL cannot execute multi-function programs because function definitions don't persist in scope.**

```javascript
// Current behavior (BROKEN):
const program = `
func calculateTotal(x: float) -> float = x * 1.08
calculateTotal(100.0)
`;

window.ailangEval(program);
// Error: undefined variable: calculateTotal at <repl>:2:1
```

### Why This Happens

The WASM REPL evaluates expressions line-by-line:
1. Line 1: `func calculateTotal(...) = ...` → Parsed and evaluated, but definition is **ephemeral**
2. Line 2: `calculateTotal(100.0)` → Tries to reference function, but **it's not in scope anymore**

**Root cause**: REPL is designed for interactive expression evaluation, not for loading complete programs with function definitions.

### Impact

This limitation breaks **all** realistic AILANG browser demos:
- ❌ Invoice processor with business logic functions
- ❌ Data validation pipelines
- ❌ JSON transformation tools
- ❌ Any demo with more than trivial inline expressions

**Current workaround**: None that works. Users see cryptic "undefined variable" errors.

## Goals

### Primary Goals (Must Achieve)
1. **Module registration API** - `window.ailangLoadModule(name, code)` to load AILANG modules
2. **Function persistence** - Functions defined via `ailangLoadModule` stay in scope for `ailangEval`
3. **Import resolution** - `ailangEval` can import and call functions from loaded modules
4. **Error reporting** - Clear errors if module not loaded or function not exported

### Stretch Goals
1. Virtual filesystem API - `window.ailangMountFS({'/path': code})`
2. Direct call API - `window.ailangCall(module, func, args)` to skip REPL
3. Module introspection - `window.ailangListModules()` for debugging

### Non-Goals (Out of Scope)
- Multi-module dependency resolution (single module is enough for demos)
- Package management in browser
- Hot module reloading
- TypeScript type definitions (future nice-to-have)

## Solution Design

### Architecture Overview

Add a **module registry** to the WASM REPL that:
1. Accepts module code via JavaScript API
2. Compiles module to Core AST
3. Type checks and registers exported functions
4. Makes functions available to subsequent `ailangEval` calls

### API Design

```javascript
// 1. Load module into registry
window.ailangLoadModule('invoice_processor', `
module invoice_processor

type LineItem = { description: string, quantity: int, unit_price: float }

func lineItemTotal(item: LineItem) -> float =
  float(item.quantity) * item.unit_price

export func processInvoice(items: [LineItem]) -> float =
  foldl(lineItemTotal, 0.0, items)
`);
// Returns: {success: true, exports: ['processInvoice']}
// Or: {success: false, error: 'Type error: ...'}

// 2. Import and use in REPL
const result = window.ailangEval(`
import invoice_processor (processInvoice)
processInvoice([{description: "Widget", quantity: 10, unit_price: 25.0}])
`);
// Returns: "250.0 :: float"
```

## Axiom Compliance

This feature aligns well with AILANG's design axioms:

| Axiom | Score | Justification |
|-------|-------|---------------|
| **A1: Determinism** | 0 | Neutral - module loading is deterministic given same source code |
| **A2: Replayability** | 0 | Neutral - doesn't affect trace generation |
| **A3: Effect Legibility** | 0 | Neutral - no new effects introduced |
| **A4: Explicit Authority** | 0 | Neutral - uses existing REPL capabilities |
| **A5: Bounded Verification** | +1 | Module compilation checked at load time, not runtime |
| **A6: Safe Concurrency** | 0 | Neutral - WASM is single-threaded |
| **A7: Machines First** | +1 | JavaScript API improves toolability for browser demos |
| **A8: Minimal Syntax** | 0 | Neutral - no new AILANG syntax, just JavaScript API |
| **A9: Cost Visibility** | 0 | Neutral - no resource tracking added |
| **A10: Composability** | +1 | Modules compose via import statements |
| **A11: Structured Failure** | +1 | Returns `{success, error}` not JavaScript exceptions |
| **A12: System Boundary** | +1 | Explicit `ailangLoadModule()` call marks boundary crossing |

**Net Score: +5** → **APPROVED** (exceeds +2 threshold)

**Key strengths:**
- **A12**: Module loading is explicit - no implicit module discovery
- **A11**: Compilation errors return structured data JavaScript can handle
- **A5**: Type errors caught at load time, not runtime
- **A7**: JavaScript API enables browser-based tooling without server infrastructure

### Implementation Plan

#### Phase 1: Module Registry (Go WASM side) - 2 days, ~200 LOC

**Files to create**:

1. **`internal/repl/module_registry.go`** (~100 LOC)
   ```go
   package repl

   import (
       "fmt"
       "github.com/sunholo-data/ailang/internal/eval"
       "github.com/sunholo-data/ailang/internal/pipeline"
       "github.com/sunholo-data/ailang/internal/types"
   )

   // ModuleRegistry stores compiled modules for REPL access
   type ModuleRegistry struct {
       modules map[string]*RegisteredModule
   }

   type RegisteredModule struct {
       Name    string
       Exports map[string]*Export  // function name -> export
   }

   type Export struct {
       Name   string
       Value  interface{}           // Compiled function closure
       Scheme *types.Scheme         // Type signature
   }

   func NewModuleRegistry() *ModuleRegistry {
       return &ModuleRegistry{
           modules: make(map[string]*RegisteredModule),
       }
   }

   // LoadModule compiles and registers a module
   func (mr *ModuleRegistry) LoadModule(name, sourceCode string) error {
       // 1. Parse module source
       prog, err := pipeline.Parse(sourceCode)
       if err != nil {
           return fmt.Errorf("parse error: %w", err)
       }

       // 2. Elaborate to Core
       core, err := pipeline.Elaborate(prog)
       if err != nil {
           return fmt.Errorf("elaboration error: %w", err)
       }

       // 3. Type check
       typed, err := pipeline.TypeCheck(core)
       if err != nil {
           return fmt.Errorf("type error: %w", err)
       }

       // 4. Evaluate module to get compiled function closures
       evaluator := eval.NewEvaluator()
       env := eval.NewEnvironment()

       for _, decl := range typed.Declarations {
           val, err := evaluator.EvalDeclaration(decl, env)
           if err != nil {
               return fmt.Errorf("eval error in %s: %w", decl.Name, err)
           }
           env.Set(decl.Name, val)
       }

       // 5. Extract exports
       exports := make(map[string]*Export)
       for _, decl := range typed.Declarations {
           if decl.Exported {
               exports[decl.Name] = &Export{
                   Name:   decl.Name,
                   Value:  env.Get(decl.Name),  // Evaluated closure
                   Scheme: decl.Scheme,
               }
           }
       }

       // 5. Register
       mr.modules[name] = &RegisteredModule{
           Name:    name,
           Exports: exports,
       }

       return nil
   }

   // GetExport retrieves a specific export
   func (mr *ModuleRegistry) GetExport(moduleName, funcName string) (*Export, error) {
       mod, ok := mr.modules[moduleName]
       if !ok {
           return nil, fmt.Errorf("module %s not loaded", moduleName)
       }

       exp, ok := mod.Exports[funcName]
       if !ok {
           return nil, fmt.Errorf("function %s not exported by %s", funcName, moduleName)
       }

       return exp, nil
   }
   ```

2. **`internal/repl/repl.go` - Add registry field** (~10 LOC)
   ```go
   type REPL struct {
       // ... existing fields ...
       registry *ModuleRegistry  // NEW
   }

   func New() *REPL {
       return &REPL{
           // ... existing init ...
           registry: NewModuleRegistry(),
       }
   }
   ```

3. **`cmd/ailang/wasm.go` - Expose JS API** (~90 LOC)
   ```go
   //go:build wasm

   package main

   import (
       "encoding/json"
       "syscall/js"
   )

   // Exposed as window.ailangLoadModule
   func ailangLoadModule(this js.Value, args []js.Value) (result interface{}) {
       // Panic recovery - prevents WASM runtime crash on Go panic
       defer func() {
           if r := recover(); r != nil {
               result = map[string]interface{}{
                   "success": false,
                   "error":   fmt.Sprintf("internal error: %v", r),
               }
           }
       }()

       if len(args) != 2 {
           return map[string]interface{}{
               "success": false,
               "error":   "Expected 2 arguments: moduleName, sourceCode",
           }
       }

       moduleName := args[0].String()
       sourceCode := args[1].String()

       // Load into registry
       err := globalREPL.registry.LoadModule(moduleName, sourceCode)
       if err != nil {
           return map[string]interface{}{
               "success": false,
               "error":   err.Error(),
           }
       }

       // Get export names
       mod := globalREPL.registry.modules[moduleName]
       exports := make([]string, 0, len(mod.Exports))
       for name := range mod.Exports {
           exports = append(exports, name)
       }

       return map[string]interface{}{
           "success": true,
           "exports": exports,
       }
   }

   func init() {
       js.Global().Set("ailangLoadModule", js.FuncOf(ailangLoadModule))
   }
   ```

#### Phase 2: Import Resolution (Go WASM side) - 1 day, ~100 LOC

**Files to modify**:

1. **`internal/repl/repl.go` - Resolve imports from registry** (~80 LOC)
   ```go
   // ProcessExpression handles import statements before evaluation
   func (r *REPL) ProcessExpression(input string, out io.Writer) {
       // ... existing parse/elaborate ...

       // NEW: Check for import statements
       if importStmt, ok := surfaceAST.(*ast.Import); ok {
           if err := r.resolveImport(importStmt); err != nil {
               fmt.Fprintf(out, "Import error: %v\n", err)
               return
           }
           fmt.Fprintf(out, "Imported %s\n", importStmt.Module)
           return
       }

       // ... existing evaluation ...
   }

   func (r *REPL) resolveImport(imp *ast.Import) error {
       // Get module from registry
       mod, ok := r.registry.modules[imp.Module]
       if !ok {
           return fmt.Errorf("module %s not loaded (use ailangLoadModule first)", imp.Module)
       }

       // Import requested symbols
       for _, symbol := range imp.Symbols {
           exp, ok := mod.Exports[symbol]
           if !ok {
               return fmt.Errorf("symbol %s not exported by %s", symbol, imp.Module)
           }

           // Add to REPL environment
           r.env.Set(symbol, exp.Value)
           r.typeEnv.BindScheme(symbol, exp.Scheme)
       }

       return nil
   }
   ```

2. **Tests** (~20 LOC)
   ```go
   func TestModuleRegistry(t *testing.T) {
       registry := NewModuleRegistry()

       code := `
       module test
       export func add(x: int, y: int) -> int = x + y
       `

       err := registry.LoadModule("test", code)
       if err != nil {
           t.Fatalf("LoadModule failed: %v", err)
       }

       exp, err := registry.GetExport("test", "add")
       if err != nil {
           t.Fatalf("GetExport failed: %v", err)
       }

       if exp.Name != "add" {
           t.Errorf("Expected export name 'add', got %s", exp.Name)
       }
   }
   ```

#### Phase 3: JavaScript Wrapper (Browser side) - 0.5 days, ~100 LOC

**Files to create**:

1. **`docs/static/js/ailang-wrapper.js`** (~100 LOC)
   ```javascript
   /**
    * High-level API for AILANG WASM REPL
    */
   export class AILANGWrapper {
     constructor() {
       this.ready = false;
       this.loadedModules = new Set();
     }

     async init() {
       // Wait for wasm_exec.js to load
       await this.waitForGo();

       // Initialize Go WASM runtime
       const go = new Go();
       const result = await WebAssembly.instantiateStreaming(
         fetch('/wasm/ailang.wasm'),
         go.importObject
       );
       go.run(result.instance);

       // Wait for AILANG REPL to initialize
       await this.waitForREPL();

       this.ready = true;
     }

     async waitForGo() {
       while (typeof Go === 'undefined') {
         await new Promise(resolve => setTimeout(resolve, 100));
       }
     }

     async waitForREPL() {
       while (typeof window.ailangEval === 'undefined') {
         await new Promise(resolve => setTimeout(resolve, 100));
       }
     }

     /**
      * Load an AILANG module
      * @param {string} name - Module name (e.g., 'invoice_processor')
      * @param {string} code - AILANG source code
      * @returns {Promise<{success: boolean, exports?: string[], error?: string}>}
      */
     async loadModule(name, code) {
       if (!this.ready) {
         throw new Error('AILANG not initialized - call init() first');
       }

       const result = window.ailangLoadModule(name, code);

       if (result.success) {
         this.loadedModules.add(name);
       }

       return result;
     }

     /**
      * Evaluate AILANG expression
      * @param {string} expr - AILANG expression or statement
      * @returns {string} Result string (e.g., "42 :: Int")
      */
     eval(expr) {
       if (!this.ready) {
         throw new Error('AILANG not initialized - call init() first');
       }

       return window.ailangEval(expr);
     }

     /**
      * Call a function from a loaded module
      * @param {string} moduleName - Module name
      * @param {string} funcName - Function name
      * @param {string} argsExpr - AILANG expression for arguments
      * @returns {string} Result string
      */
     call(moduleName, funcName, argsExpr) {
       if (!this.loadedModules.has(moduleName)) {
         throw new Error(`Module ${moduleName} not loaded`);
       }

       const expr = `
       import ${moduleName} (${funcName})
       ${funcName}(${argsExpr})
       `;

       return this.eval(expr);
     }
   }
   ```

### Usage Example (Invoice Processor)

```javascript
// 1. Initialize AILANG
const ailang = new AILANGWrapper();
await ailang.init();

// 2. Load invoice processor module
const moduleCode = await fetch('/wasm/invoice_processor.ail').then(r => r.text());
const result = await ailang.loadModule('invoice_processor', moduleCode);

if (!result.success) {
  console.error('Failed to load module:', result.error);
  return;
}

console.log('Loaded exports:', result.exports);
// ["processInvoice", "lineItemTotal", ...]

// 3. Process invoice
const invoiceJSON = JSON.stringify({
  invoice_number: "INV-001",
  line_items: [...]
});

const output = ailang.call(
  'invoice_processor',
  'processInvoice',
  `"${invoiceJSON}"`
);

console.log('Result:', output);
// "{"total": 250.0, "valid": true} :: string"
```

## Testing Strategy

### Unit Tests (~80 LOC)
- `internal/repl/module_registry_test.go` - Registry operations
- `internal/repl/repl_test.go` - Import resolution
- Test module loading, export retrieval, import statements

### Integration Tests
- WASM test: Load module, import function, call from REPL
- Browser test: Full invoice processor workflow
- Error test: Module not loaded, function not exported

### Manual Testing
- Invoice processor demo (primary use case)
- JSON parser demo
- Data validation demo

## Risk Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| **Module compilation errors in browser** | Medium | Comprehensive error reporting, fail gracefully |
| **Memory leaks from module registry** | Low | Reuse existing REPL environment cleanup |
| **Import cycles** | Low | Single-module loading (no dependencies) |
| **WASM binary size increase** | Low | Registry is small (~200 LOC), minimal impact |

## Success Metrics

| Metric | Target |
|--------|--------|
| **Module loading** | `ailangLoadModule` returns exports correctly |
| **Import resolution** | `import module (func)` makes func available |
| **Invoice demo** | Full invoice processor works in browser |
| **Error messages** | Clear errors for missing modules/functions |
| **WASM binary size** | < 5% increase from v0.7.1 |

## Alternatives Considered

### Alt 1: Virtual Filesystem API
**Rejected**: More complex, requires filesystem emulation in Go WASM. Module registry is simpler and sufficient for demos.

### Alt 2: Direct Call API (`ailangCall(module, func, args)`)
**Considered**: Simpler for demos but bypasses REPL entirely. Keep for stretch goal.

### Alt 3: Persistent REPL State
**Rejected**: Would require major REPL refactor. Module registry is non-invasive.

### Alt 4: Precompile Modules to JS
**Rejected**: Requires AILANG→JS codegen. Too complex, defeats purpose of WASM.

## Out of Scope (Deferred)

- Multi-module dependencies (v0.8.0)
- Module hot reloading (v0.8.0)
- Virtual filesystem API (v0.8.0)
- TypeScript type definitions (v0.9.0)
- npm package for wrapper (v1.0.0)

## Known Limitations

**v0.7.2 Scope (acceptable for demos):**

1. **Single module at a time** - No inter-module dependencies; each module is self-contained
2. **Name collision** - If two modules export same function name, last import wins (implicit shadowing)
3. **No module unloading** - Registry grows indefinitely; acceptable for demos with 1-3 modules
4. **No type introspection** - Can't query module's type signatures from JavaScript
5. **Module name must match** - `ailangLoadModule('foo', code)` requires `module foo` declaration in code

**Future versions:**
- v0.8.0: Multi-module dependencies, `ailangUnloadModule()`, module reloading
- v0.9.0: TypeScript type definitions, type introspection API (`ailangGetExportTypes()`)

## Definition of Done

- ✅ `ailangLoadModule(name, code)` JavaScript API works
- ✅ `import module (func)` resolves from registry
- ✅ Invoice processor demo fully functional
- ✅ Error messages clear and actionable
- ✅ All tests passing (unit + integration)
- ✅ Documentation updated (WASM guide)
- ✅ WASM binary rebuilt and deployed

## Related Documents

- [M-REPL1: REPL Persistent Bindings](design_docs/planned/v0_7_2/M-REPL1_persistent_bindings.md) - CLI REPL module loading
- [Invoice Processor Demo](design_docs/planned/ailang-demo-invoice-processor.md) - Primary use case
- [WASM Integration Guide](docs/docs/guides/wasm-integration.md) - Existing WASM docs

---

**Document Version**: v1.0
**Created**: January 28, 2026
**Target Release**: v0.7.2
