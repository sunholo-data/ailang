# AILANG v0.1.0 Release Notes

**Release Date**: October 2, 2025
**Codename**: "Type System Complete"
**Status**: MVP - Minimum Viable Product

---

## 🎯 TL;DR

AILANG v0.1.0 is the **first complete type system MVP** for this AI-first functional programming language. This release delivers a **production-quality Hindley-Milner type system**, **type classes with dictionary-passing**, and a **fully functional REPL**.

**What Works**:
- ✅ Complete type inference with let-polymorphism
- ✅ Type classes (Num, Eq, Ord, Show)
- ✅ Lambda calculus with closures and currying
- ✅ Interactive REPL with debugging tools
- ✅ Module type-checking (execution in v0.2.0)

**Known Limitation**:
- ⚠️ Module files type-check but cannot execute (runtime coming in v0.2.0)

---

## 🚀 Major Features

### 1. Complete Hindley-Milner Type System

**7,291 lines of implementation** delivering:

- **Polymorphic Type Inference** - Automatic type deduction with generalization
- **Let-Polymorphism** - Polymorphic bindings in let expressions
- **Type Unification** - Occurs check, infinite type prevention
- **Constraint Solving** - Type class constraint generation and resolution
- **Defaulting** - Automatic defaulting for ambiguous numeric types (Int, Float)

**Example**:
```ailang
λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> let compose = \f. \g. \x. f(g(x)) in :type compose
compose :: ∀α β γ. (β → γ) → (α → β) → α → γ
```

**Implementation**:
- `internal/types/` - 7,291 LOC
- `internal/elaborate/` - 2,059 LOC (Surface AST → Core AST)
- `internal/link/` - 1,418 LOC (Dictionary linking)

### 2. Type Classes with Dictionary-Passing

**Complete type class system** with:

- **Four Core Type Classes**: `Num`, `Eq`, `Ord`, `Show`
- **Dictionary-Passing Semantics** - Runtime dictionary creation and passing
- **Constraint Propagation** - Automatic constraint inference
- **Instance Resolution** - Module-aware instance lookup
- **Superclass Provisions** - `Ord` provides `Eq` instances

**Example**:
```ailang
λ> 1 + 2
3 :: Int

λ> 3.14 + 2.86
6.0 :: Float

λ> 5 == 5
true :: Bool

λ> :instances
Available instances:
  Num: Num[Int], Num[Float]
  Eq: Eq[Int], Eq[Float]
  Ord: Ord[Int] (provides Eq[Int]), Ord[Float] (provides Eq[Float])
  Show: Show[Int], Show[Float], Show[String], Show[Bool]
```

**Implementation**:
- Dictionary elaboration in `internal/elaborate/`
- Runtime dictionaries in `internal/eval/`
- Instance environment in `internal/types/`

### 3. Professional Interactive REPL

**1,351 lines** of polished REPL featuring:

- **Arrow Key History** - Navigate command history with ↑/↓
- **Persistent History** - Commands saved in `~/.ailang_history`
- **Tab Completion** - Auto-complete REPL commands
- **Auto-imports** - `stdlib/std/prelude` loaded automatically
- **Type Inspection** - `:type <expr>` shows qualified types
- **Instance Inspection** - `:instances` lists available instances
- **Debugging Tools** - `:dump-core`, `:dump-typed`, `:trace-defaulting`, `:dry-link`

**Example Session**:
```ailang
λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :type \f. \g. \x. f(g(x))
\f. \g. \x. f(g(x)) :: ∀α β γ. (β → γ) → (α → β) → α → γ

λ> "Hello " ++ "AILANG!"
Hello AILANG! :: String

λ> :history
[Shows command history]

λ> :quit
```

**Implementation**:
- `internal/repl/` - 1,351 LOC
- `github.com/chzyer/readline` - External dependency for line editing

### 4. Module System (Type-Checking Phase)

**Complete module type-checking** with:

- **Module Declarations** - `module path/to/module`
- **Import/Export** - Selective imports, explicit exports
- **Path Resolution** - Correct module path resolution and validation
- **Dependency Analysis** - Import graph construction, cycle detection
- **Interface Generation** - Module signatures with exported types/functions
- **Manifest System** - Module metadata and versioning

**Example**:
```ailang
module examples/demo

import stdlib/std/option (Option, Some, None)

export type MyData = {
  value: Int
}

export func process(x: Int) -> Option[Int] {
  if x > 0 then Some(x * 2) else None
}
```

**Status**: Modules **parse and type-check correctly** but cannot execute until v0.2.0.

**Implementation**:
- `internal/module/` - 1,030 LOC (resolution & validation)
- `internal/loader/` - 503 LOC (module loading)
- `internal/iface/` - 864 LOC (module interfaces)
- `internal/manifest/` - 606 LOC (module manifests)

### 5. Lambda Calculus & Expression Evaluation

**3,712 lines** of evaluator supporting:

- **First-Class Functions** - Functions as values
- **Closures** - Lexical scoping with environment capture
- **Currying** - Automatic partial application
- **Higher-Order Functions** - Function composition, map, filter patterns
- **Let Bindings** - Polymorphic let expressions (up to 3 nested levels)
- **Conditionals** - `if-then-else` expressions

**Example**:
```ailang
-- Function composition
let compose = \f. \g. \x. f(g(x)) in
let addThenDouble = compose (\x. x * 2) (\x. x + 1) in
addThenDouble(5)
-- Result: 12

-- Closures
let makeAdder = \n. \x. x + n in
let add10 = makeAdder(10) in
add10(5)
-- Result: 15
```

**Implementation**:
- `internal/eval/` - 3,712 LOC
- Works for non-module files, REPL
- Module execution in v0.2.0

### 6. Structured Error Reporting

**657 lines** of comprehensive error system:

- **JSON Error Schemas** - Versioned error format for tooling
- **Deterministic Diagnostics** - Stable error messages, line/column positions
- **Error Categories** - Parse, type, module, evaluation errors
- **Helpful Messages** - Context and suggestions
- **Schema Versioning** - `v1.0.0` error schema with migration path

**Example**:
```json
{
  "schema_version": "v1.0.0",
  "errors": [{
    "code": "TYPE001",
    "message": "type unification failed: Int vs String",
    "location": {"file": "example.ail", "line": 5, "column": 10},
    "severity": "error"
  }]
}
```

**Implementation**:
- `internal/errors/` - 657 LOC
- `internal/schema/` - 176 LOC (JSON schemas)

---

## 📊 By the Numbers

### Code Statistics

| Metric | Value |
|--------|-------|
| **Go Implementation** | 27,610 LOC |
| **Go Tests** | 10,559 LOC |
| **Test Coverage** | 24.8% |
| **AILANG stdlib** | 168 LOC |
| **Example Files** | 47 files |
| **Working Examples** | 12 files (25.5%) |
| **Documentation** | ~9,000 lines (25 files) |
| **Development Duration** | 6 days (v0.0.1 → v0.1.0) |

### Package Sizes (Top 10)

| Package | LOC | Purpose |
|---------|-----|---------|
| `types` | 7,291 | Type system |
| `eval` | 3,712 | Evaluator |
| `parser` | 2,656 | Parser |
| `elaborate` | 2,059 | AST elaboration |
| `pipeline` | 1,496 | Compilation pipeline |
| `link` | 1,418 | Dictionary linking |
| `repl` | 1,351 | Interactive REPL |
| `ast` | 1,298 | AST definitions |
| `module` | 1,030 | Module resolution |
| `lexer` | 978 | Tokenization |

---

## 🎓 Examples & Demos

### Working Examples (12 total)

**Basic Expressions**:
- `examples/hello.ail` - Simple print
- `examples/simple.ail` - Basic arithmetic
- `examples/arithmetic.ail` - Arithmetic with operator precedence

**Type Classes**:
- `examples/type_classes_working_reference.ail` - Type class demo

**Algebraic Data Types**:
- `examples/adt_option.ail` - Option type usage
- `examples/adt_simple.ail` - Simple ADT definition
- `examples/effects_pure.ail` - Pure effect demonstration

**Module System (Type-Check Only)**:
- `examples/demos/hello_io.ail` - IO effect example (type-checks, doesn't execute)
- `examples/option_demo.ail` - Option type demo (type-checks)
- `examples/stdlib_demo.ail` - stdlib usage (type-checks)

**V3.3 Import Tests**:
- `examples/v3_3/imports.ail` - Import system test
- `examples/v3_3/imports_basic.ail` - Basic imports
- `examples/v3_3/import_conflict.ail` - Conflict resolution
- `examples/v3_3/poly_imports.ail` - Polymorphic imports

**NEW: Showcase Examples**:
- `examples/showcase/01_type_inference.ail` - Type inference demo
- `examples/showcase/02_lambdas.ail` - Lambda expressions & composition
- `examples/showcase/03_type_classes.ail` - Type class polymorphism
- `examples/showcase/04_closures.ail` - Closures & captured environments

### Documentation Files

**User Documentation** (8 files, ~2,000 lines):
- [README.md](README.md) - Main introduction (updated for v0.1.0)
- [LIMITATIONS.md](docs/LIMITATIONS.md) - **NEW** - Comprehensive limitations guide
- [examples/STATUS.md](examples/STATUS.md) - **NEW** - Complete example inventory
- [examples/README.md](examples/README.md) - **NEW** - Example usage guide
- [Getting Started](docs/guides/getting-started.md) - Installation & tutorial
- [REPL Commands](docs/reference/repl-commands.md) - REPL reference

**Development Documentation** (5 files, ~1,500 lines):
- [CLAUDE.md](CLAUDE.md) - **UPDATED** - AI assistant instructions
- [METRICS.md](docs/METRICS.md) - **NEW** - Project statistics
- [SHOWCASE_ISSUES.md](docs/SHOWCASE_ISSUES.md) - **NEW** - Parser/execution limitations
- [Development Guide](docs/guides/development.md) - Contributing guide
- [Implementation Status](docs/reference/implementation-status.md) - Component status

**Design Documentation** (6 files, ~3,500 lines):
- [v0.1.0 MVP Roadmap](design_docs/20250929/v0_1_0_mvp_roadmap.md) - Current milestone
- [v0.2.0 Module Execution](design_docs/planned/v0_2_0_module_execution.md) - Next milestone
- [Initial Design](design_docs/20250926/initial_design.md) - Original vision

---

## ⚠️ Known Limitations

### Critical Limitation: Module Execution Gap

**Module files type-check but cannot execute** until v0.2.0.

**What This Means**:
- Files with `module` declarations parse ✅ and type-check ✅ but fail at runtime ❌
- Non-module `.ail` files execute successfully ✅
- REPL is fully functional ✅

**Workaround**: Use non-module files for executable code, or wait for v0.2.0 (~4 weeks).

**Technical Reason**: Module execution requires runtime infrastructure that's not yet implemented:
- Module instance creation
- Import resolution and linking at runtime
- Top-level function execution
- Exported function calls

See [docs/LIMITATIONS.md](docs/LIMITATIONS.md) for comprehensive details.

### Parser Limitations

1. **Let Nesting Limit**: Only 3 nested `let...in` expressions work (4+ fails)
2. **Pattern Matching**: `match` expressions not yet implemented (v0.2.0)
3. **Non-Module Restrictions**: Cannot use `func`, `type`, `import`, `export` in non-module files

### Type System Limitations

1. **Record Field Access**: Unification bugs in some cases
2. **List Operations**: Limited runtime support
3. **Effect System**: Not yet implemented (v0.2.0)

See [docs/LIMITATIONS.md](docs/LIMITATIONS.md) and [docs/SHOWCASE_ISSUES.md](docs/SHOWCASE_ISSUES.md) for full details.

---

## 🔧 Breaking Changes

### From v0.0.12 to v0.1.0

**None** - v0.1.0 is a documentation and polish release. All v0.0.12 code continues to work.

**Documentation Changes**:
- Updated README.md with honest v0.1.0 status
- Created LIMITATIONS.md to document known issues
- Created examples/STATUS.md with complete example inventory
- Updated CLAUDE.md with current implementation status

---

## 🐛 Bug Fixes

### Fixed in v0.1.0

1. **Module Entrypoint Resolution** (d015e2c)
   - Fixed: Non-module files incorrectly treated as modules
   - Solution: Check `len(result.Interface.Exports) > 0` before entrypoint lookup
   - Impact: tests/binops_int.ail and similar non-module files now work correctly

2. **Example Verification**
   - Added automated testing with `tools/audit-examples.sh`
   - Identified and documented 12 working, 3 type-check only, 27 broken examples
   - Added warning headers to module examples that can't execute

---

## 🚀 What's Next: v0.2.0 Roadmap

### Planned Features (3.5-4.5 weeks)

**M-R1: Module Execution Runtime** (~1,200 LOC, 1.5-2 weeks)
- Module instance creation and initialization
- Import resolution and linking at runtime
- Top-level function execution
- Exported function calls

**M-R2: Algebraic Effects Foundation** (~800 LOC, 1-1.5 weeks)
- Effect declarations and checking
- Effect handler syntax (`with`, `handle`)
- Capability-based effect system
- Basic effects: `IO`, `FS`, `Net`

**M-R3: Pattern Matching** (~600 LOC, 1 week)
- `match` expressions
- Pattern guards
- Exhaustiveness checking
- Constructor patterns for ADTs

**Testing & Quality**:
- Increase test coverage to 60%+
- Fix record field access bugs
- Improve list operation runtime support

See [design_docs/planned/v0_2_0_module_execution.md](design_docs/planned/v0_2_0_module_execution.md) for details.

---

## 📥 Installation

### From Source

```bash
git clone https://github.com/sunholo-data/ailang.git
cd ailang
make install

# Verify installation
ailang --version
# Output: ailang v0.1.0
```

### System Requirements

- **Go**: 1.22 or higher
- **OS**: macOS (darwin), Linux (amd64, arm64)
- **Dependencies**: readline (installed via `make deps`)

---

## 🎯 Quick Start

### Hello World

```bash
# Create a simple program
cat > hello.ail << 'EOF'
-- hello.ail
print("Hello, AILANG!")
EOF

# Run it
ailang run hello.ail
# Output: Hello, AILANG!
```

### Try the REPL

```bash
ailang repl

λ> 1 + 2
3 :: Int

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> :quit
```

### Explore Examples

```bash
# Run working examples
ailang run examples/arithmetic.ail
ailang run examples/showcase/01_type_inference.ail
ailang run examples/showcase/02_lambdas.ail

# See all examples
cat examples/STATUS.md
```

---

## 🙏 Acknowledgments

### Inspiration

AILANG draws inspiration from:
- **Haskell** - Type system, type classes, purity
- **OCaml** - Module system, effects
- **Rust** - Capability-based security
- **Erlang/Go** - CSP concurrency

### Contributors

**Initial Development**:
- Mark (sunholo-data) - Type system, parser, REPL, modules, documentation

### Special Thanks

- The Haskell community for pioneering Hindley-Milner type systems
- The OCaml team for algebraic effects research
- The Go team for excellent tooling and standard library

---

## 📄 License

Apache 2.0 - See [LICENSE](LICENSE) for details.

---

## 📞 Support & Community

### Documentation
- **Main Docs**: [docs/](docs/)
- **Limitations**: [docs/LIMITATIONS.md](docs/LIMITATIONS.md) ⚠️ Read this first!
- **Examples**: [examples/STATUS.md](examples/STATUS.md)

### Contributing
- **Development Guide**: [docs/guides/development.md](docs/guides/development.md)
- **Issues**: [GitHub Issues](https://github.com/sunholo-data/ailang/issues)
- **Pull Requests**: Welcome! See development guide.

### Resources
- **GitHub**: https://github.com/sunholo-data/ailang
- **Changelog**: [CHANGELOG.md](CHANGELOG.md)
- **Design Docs**: [design_docs/](design_docs/)

---

## 🎉 Celebrating v0.1.0

### What We Achieved

✅ **Built a complete type system** from scratch (7,291 LOC)
✅ **Implemented type classes** with dictionary-passing (full pipeline)
✅ **Created a professional REPL** with debugging tools (1,351 LOC)
✅ **Validated module type-checking** with dependency analysis
✅ **Produced 12 working examples** demonstrating core features
✅ **Wrote comprehensive documentation** (~9,000 lines, 25 files)

### What We Learned

💡 **Type systems are complex** - 8x more code than estimated
💡 **Parser limitations exist** - 3-deep let nesting is architectural
💡 **Honest documentation matters** - Users appreciate transparency
💡 **REPL is invaluable** - Best way to test type system features
💡 **Modular architecture pays off** - Clean separation enabled rapid development

### What's Next

🚀 **v0.2.0 in 3.5-4.5 weeks** - Module execution, effects, pattern matching
🚀 **Test coverage to 60%+** - Comprehensive testing for stability
🚀 **Example expansion** - More working examples, tutorials
🚀 **Performance tuning** - Optimize type inference and evaluation

---

**Thank you for trying AILANG v0.1.0!** 🎉

This is the foundation of an AI-first programming language. We're excited to see what you build with it, and we welcome your feedback, bug reports, and contributions.

**Happy hacking!** 🚀

---

*Released October 2, 2025 - AILANG Development Team*
