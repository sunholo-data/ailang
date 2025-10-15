# AILANG v0.3.4+ Strategic Direction - Implementation Audit

**Date**: 2025-10-13
**Current Version**: v0.3.4 (REPL Stabilization + Browser Playground)
**Purpose**: Audit original vision vs current state, identify high-value next steps

---

## Executive Summary

**Current State**: v0.3.4 is a **solid functional foundation** with 72.7% example success rate, complete effect system, and working REPL.

**Gap Analysis**: Original vision (typed quasiquotes, CSP concurrency, deterministic execution) is **deferred to v0.4.0+**. Current bottleneck is **language completeness** for real-world programs.

**Recommendation**: Focus on **high-impact language features** that unblock M-EVAL benchmarks and improve AI codegen success rates before tackling advanced features.

---

## Original Vision vs Current Implementation

### ✅ What's Working (v0.3.4)

| Feature | Status | Notes |
|---------|--------|-------|
| Hindley-Milner Type Inference | ✅ Complete | Full HM with let-polymorphism |
| Type Classes | ✅ Complete | Num, Eq, Ord, Show with dictionary passing |
| Algebraic Effects | ✅ Complete | IO, FS, Clock, Net with capability security |
| Effect System | ✅ Complete | Row polymorphism, effect propagation |
| Pattern Matching | ✅ Complete | Exhaustiveness checking, guards |
| Recursion | ✅ Complete | Self-recursion, mutual recursion (func only) |
| Module System | ✅ Complete | Cross-module imports, manifest-based loading |
| Records | ✅ Complete | Field access, subsumption, row polymorphism |
| REPL | ✅ Complete | Full type checking, persistent bindings |
| Block Expressions | ✅ Complete | Sequencing with semicolons |
| Lambda Calculus | ✅ Complete | First-class functions, closures |

### ❌ What's Missing (Original Vision)

| Feature | Status | Priority | Estimated Effort |
|---------|--------|----------|------------------|
| **Typed Quasiquotes** | ❌ Not started | P4 (Low - v0.5.0+) | 2-3 weeks |
| **CSP Concurrency** | ❌ Not started | P4 (Low - v0.5.0+) | 3-4 weeks |
| **Session Types** | ❌ Not started | P4 (Low - v0.5.0+) | 2-3 weeks |
| **Deterministic Execution** | ❌ Not started | P4 (Low - v0.5.0+) | 1-2 weeks |
| **Training Data Export** | ❌ Not started | P4 (Low - v0.5.0+) | 1 week |

### ⚠️ What's Incomplete (Blocking Real Usage)

| Feature | Status | Impact | Priority |
|---------|--------|--------|----------|
| **Anonymous Function Syntax** | ⚠️ Partial | HIGH - blocks higher-order functions | **P0** |
| **Recursive Lambdas (letrec)** | ❌ Missing | Medium - REPL UX | P2 |
| **Numeric Coercion** | ❌ Missing | Medium - convenience | P3 |
| **Record Update Syntax** | ❌ Missing | Medium - ergonomics | P2 |
| **List Literals** | ⚠️ ADT only | Medium - stdlib limitations | P2 |
| **Error Propagation (?)** | ❌ Missing | Low - convenience | P3 |

---

## M-EVAL Benchmark Analysis (v0.3.4)

**Current Success Rate**: 38.9% (35/90 benchmarks passing)

### Failure Categories

| Category | Count | Root Cause | Fix Priority |
|----------|-------|------------|--------------|
| **Parse Errors** | 15 | Missing `func` syntax in expressions | **P0 CRITICAL** |
| **Logic Errors** | 35 | AI mistakes (modulo, fold, etc.) | P4 (prompt/repair) |
| **Compile Errors** | 5 | Missing language features | P1-P2 |
| **Not Yet Analyzed** | 0 | - | - |

### Top Blockers from Benchmarks

1. **higher_order_functions (5/5 fail)** - Parse error: `func(x: int) -> int { ... }` not allowed in let-binding
   - Root cause: Anonymous func expressions not parsed
   - Fix: Add lambda syntax sugar for `func` keyword
   - Impact: **HIGH** - Blocks all higher-order programming

2. **list_comprehension (5/5 fail)** - Logic error: outputs 0 instead of 220
   - Root cause: Modulo operator or fold accumulator bug
   - Fix: Debug generated code, likely AI mistake not language bug
   - Impact: Medium - Eval repair should catch this

3. **json_parse, float_eq, cli_args, numeric_modulo, pipeline (all fail)** - Not implemented features
   - Root cause: Missing stdlib (JSON, args parsing, etc.)
   - Fix: Expand stdlib in future versions
   - Impact: Medium - Nice to have, not core language

---

## Original Design Philosophy Check

### Core Principles (from initial_design.md)

1. **Explicit Effects via Algebraic Effect System** ✅ **IMPLEMENTED**
   - Pure functions by default ✅
   - Effects tracked in types ✅
   - Capability-based permissions ✅

2. **Everything is a Typed Expression** ✅ **IMPLEMENTED**
   - No statements ✅ (blocks are expressions)
   - Complete execution traces ✅
   - Errors as values (Result type) ✅

3. **Type-Safe Metaprogramming** ❌ **DEFERRED**
   - Typed quasiquotes ❌ (v0.5.0+)
   - Compile-time validation ❌
   - AST generation ❌

4. **Deterministic Execution** ⚠️ **PARTIAL**
   - Explicit random seeds ❌
   - Time virtualization ⚠️ (Clock effect exists, virtual time partial)
   - Reproducible traces ❌

5. **Single Concurrency Model (CSP)** ❌ **DEFERRED**
   - Channels with session types ❌ (v0.5.0+)
   - No shared mutable state ✅ (enforced)
   - Message passing ❌

**Assessment**: v0.3.4 achieves principles #1 and #2 (effect system, expression-based). Principles #3-5 are deferred to v0.4.0+.

---

## Prioritized Next Steps

### 🚨 P0: CRITICAL - Unblock Higher-Order Functions

**Issue**: `func(x: int) -> int { x * 2 }` syntax doesn't parse in expression position.

**Impact**:
- Blocks 5+ benchmarks (higher_order_functions, etc.)
- Makes functional programming painful
- Users expect this to work

**Solution Options**:

#### Option A: Lambda Expression Syntax (Easy - RECOMMENDED)
```ailang
-- Allow 'func' keyword in expression position (desugar to \x. ...)
let double = func(x: int) -> int { x * 2 };

-- Parser change: parseFunc() can return FuncLit node
-- Elaborate: FuncLit -> core.Lambda
```
**Estimated**: 2-3 hours, ~150 LOC

#### Option B: Keep Lambda Only (Status Quo)
```ailang
-- Current working syntax
let double = \x. x * 2;

-- With type annotations (if needed)
let double: int -> int = \x. x * 2;
```
**Estimated**: 0 hours, update docs to teach this pattern

**Recommendation**: **Option A** - Small effort, huge UX improvement, matches user expectations.

---

### 🔥 P1: HIGH VALUE - Quick Wins

#### 1. Add `letrec` Syntax (2 hours, ~200 LOC)
- **Why**: Enable recursive lambdas in REPL
- **Design**: [20251013_letrec_surface_syntax.md](planned/20251013_letrec_surface_syntax.md)
- **Impact**: Better REPL UX, aligns with Haskell/ML conventions
- **Effort**: Lexer + Parser + Elaborate (core.LetRec already exists)

#### 2. Add Numeric Conversion Functions (1 hour, ~100 LOC)
- **Why**: Unblock `1 + 2.5` with `intToFloat(1) + 2.5`
- **Design**: [20251013_numeric_coercion.md](planned/20251013_numeric_coercion.md) Option 1
- **Impact**: Immediate workaround for mixed numeric types
- **Effort**: Add builtins, update stdlib/std/prelude

#### 3. Fix Modulo Operator Bug (if exists) (1-2 hours)
- **Why**: list_comprehension outputs 0 instead of 220
- **Investigation**: Test `n % 2 == 0` for even numbers
- **Impact**: Fix logic errors in benchmarks
- **Effort**: Debug + test + fix

---

### 📈 P2: MEDIUM VALUE - Language Completeness

#### 4. Record Update Syntax `{r | field: val}` (1 day, ~300 LOC)
- **Design**: [M-R5b_record_extension.md](planned/M-R5b_record_extension.md)
- **Impact**: Ergonomics for record manipulation
- **Effort**: Parser + Type checker + Evaluator

#### 5. Capability Inference (AUTO_CAPS) (2-3 days, ~500 LOC)
- **Design**: [20251013_auto_caps_capability_inference.md](planned/20251013_auto_caps_capability_inference.md)
- **Impact**: UX - no need to pass `--caps` flag
- **Effort**: Static analysis of effect usage

#### 6. Better List Syntax (1-2 days, ~400 LOC)
- **Current**: `[1, 2, 3]` desugars to ADT constructors
- **Issue**: Verbose stdlib implementation
- **Fix**: Native list type or better sugar
- **Impact**: Cleaner list operations

---

### 🎯 P3: NICE TO HAVE - Polish

#### 7. Error Propagation Operator `?` (2-3 days)
- **Syntax**: `let x = readFile(path)?`
- **Impact**: Convenience, matches Rust
- **Defer**: Can use explicit match for now

#### 8. List Comprehensions (3-4 days)
- **Syntax**: `[x * 2 | x <- xs, x > 0]`
- **Impact**: Syntactic sugar only
- **Defer**: Can use map/filter for now

---

### ❌ P4: DEFERRED - Advanced Features

#### 9. Typed Quasiquotes (2-3 weeks)
- Original vision feature
- Requires: Schema system, compile-time validation
- **Defer to v0.5.0+**

#### 10. CSP Concurrency (3-4 weeks)
- Original vision feature
- Requires: Channel runtime, session types, scheduler
- **Defer to v0.5.0+**

#### 11. Deterministic Execution (1-2 weeks)
- Training data generation
- Requires: Virtual time, seed management
- **Defer to v0.5.0+**

---

## Recommended Sprint Plan (v0.3.5)

**Theme**: Functional Programming Completeness

**Duration**: 1 week

**Goals**:
1. Unblock higher-order functions (P0)
2. Quick wins for REPL and numeric handling (P1)
3. Improve M-EVAL success rate to 50%+

### Day-by-Day Plan

**Day 1: Anonymous Function Syntax**
- Add `func(...) -> T { body }` expression parsing
- Elaborate to lambda
- Test with higher_order_functions benchmark
- **Expected**: 5+ benchmarks now pass

**Day 2: Numeric Conversion + letrec**
- Morning: Add `intToFloat`, `floatToInt` builtins
- Afternoon: Add `letrec` keyword and parsing
- Test in REPL
- **Expected**: Better UX, unblock mixed numeric code

**Day 3: Debug Modulo/Logic Bugs**
- Investigate list_comprehension failure
- Test modulo operator: `n % 2 == 0`
- Fix any evaluator bugs
- **Expected**: 3-5 more benchmarks pass

**Day 4: Documentation + Examples**
- Update prompts/v0.3.0.md with func expression syntax
- Update playground.mdx
- Add examples/higher_order_functions.ail
- Add examples/numeric_conversion.ail
- **Expected**: Better AI codegen

**Day 5: M-EVAL Validation**
- Run full eval suite
- Compare v0.3.4 → v0.3.5
- Generate report
- **Expected**: 50%+ success rate (45/90 benchmarks)

---

## Metrics & Success Criteria

### Current (v0.3.4)
- Example success: 72.7% (48/66)
- M-EVAL success: 38.9% (35/90)
- Benchmarks blocked by syntax: 15
- REPL limitations: Recursive lambdas don't work

### Target (v0.3.5)
- Example success: 75%+ (50/66)
- M-EVAL success: 50%+ (45/90)
- Benchmarks blocked by syntax: 0 (P0 fixed)
- REPL limitations: Only advanced features missing

### Long-term (v0.4.0+)
- Example success: 90%+ (60/66)
- M-EVAL success: 70%+ (63/90)
- Advanced features: Quasiquotes, CSP, deterministic execution

---

## Strategic Alignment

### What We're Building Now (v0.3.x)
**"A practical functional language with effect tracking"**
- Target: Real programs with I/O, file systems, HTTP
- Users: Developers learning FP, AI-assisted coding
- Philosophy: Safety + Ergonomics + Explicitness

### What We'll Build Later (v0.4.0+)
**"An AI-first metaprogramming language"**
- Target: Code generation, training data, deterministic execution
- Users: AI researchers, code synthesis systems
- Philosophy: Machine-decidability + Reproducibility

### Current Focus: Get Basics Right
- Higher-order functions should just work ✅
- REPL should be great for experimentation ✅
- Common patterns should be ergonomic ⚠️ (improving)
- M-EVAL success rate indicates AI-friendliness ⚠️ (improving)

---

## Risks & Mitigation

### Risk 1: Feature Creep
- **Risk**: Keep adding features without stabilization
- **Mitigation**: Strict P0/P1/P2 prioritization, defer P3/P4

### Risk 2: Complexity Debt
- **Risk**: Type system or effect system becomes unmaintainable
- **Mitigation**: Keep test coverage >70%, document design decisions

### Risk 3: Original Vision Divergence
- **Risk**: v0.4.0 becomes too different from v0.1.0 plans
- **Mitigation**: This is OK! Validate MVP first, then add advanced features

### Risk 4: M-EVAL Not Representative
- **Risk**: Optimizing for benchmarks, not real users
- **Mitigation**: Track both examples/ (real code) and benchmarks (AI code)

---

## Conclusion

**v0.3.4 is a strong foundation**. The effect system, type inference, and module system are solid.

**Next focus: Language completeness**, not advanced features. Fix the P0 blocker (func expressions), add quick wins (letrec, numeric conversion), and improve M-EVAL success rates.

**Original vision (quasiquotes, CSP, etc.) is still valid** but premature. Get the basics right first, then layer on metaprogramming and concurrency.

**Recommendation**: Execute v0.3.5 sprint (1 week) focusing on P0+P1 items. Re-evaluate after hitting 50% M-EVAL success rate.

---

**Next Action**: Review this audit, discuss priorities, and decide whether to proceed with v0.3.5 sprint or adjust course.
