# M-R3: Pattern Matching Polish

**Milestone**: M-R3 (Pattern Matching Polish)
**Version**: v0.2.0
**Timeline**: 1 week
**Estimated LOC**: ~450–650 lines
**Priority**: MEDIUM (stretch goal, can slip to v0.3.0)

---

## Executive Summary

M-R3 **polishes** the existing pattern matching implementation (M-P3, v0.1.0) by adding:
1. **Guards** (`if cond`) for conditional matching
2. **Exhaustiveness diagnostics** that warn about missing cases
3. **Decision tree compilation** for faster pattern matching

**Current State**: Pattern matching works ✅ (constructors, tuples, lists, wildcards), but lacks guards and exhaustiveness checking
**Target State**: Guards work ✅, exhaustiveness warnings ✅, decision trees ✅

---

## Problem Statement

### What Works (v0.1.0)

Pattern matching is **implemented and functional**:

```ailang
match value {
  Some(x) => x,
  None => 0
}

match list {
  [] => "empty",
  [x] => "single",
  [x, y] => "pair",
  [x, y, z, ...rest] => "many"
}
```

**Implemented Patterns**:
- ✅ Constructor patterns: `Some(x)`, `None`
- ✅ Tuple patterns: `(x, y)`
- ✅ List patterns: `[]`, `[head, ...tail]`
- ✅ Wildcard: `_`
- ✅ Variable binding: `x`
- ✅ Nested patterns: `Some((x, y))`

### What's Missing (v0.2.0)

1. **Guards** - Conditional matching
   ```ailang
   match value {
     Some(x) if x > 0 => x,  -- ❌ Not supported yet
     Some(x) => -x,
     None => 0
   }
   ```

2. **Exhaustiveness Warnings**
   ```ailang
   match opt {
     Some(x) => x
     -- ❌ Missing case: None (no warning!)
   }
   ```

3. **Decision Tree Optimization**
   - Current: Linear scan through patterns
   - Target: Compiled decision tree (faster, O(log n) in many cases)

---

## Goals & Non-Goals

### Goals

1. **Guards**: Support `pattern if condition => body` syntax
2. **Exhaustiveness**: Warn (not error) on incomplete matches with suggested missing cases
3. **Decision Trees**: Compile matches to efficient decision trees
4. **Backward Compatible**: All existing patterns continue to work

### Non-Goals (Deferred)

- ❌ Or-patterns (`Some(x) | None => ...`) (v0.3+)
- ❌ As-patterns (`x @ Some(y)`) (v0.3+)
- ❌ View patterns (v0.3+)
- ❌ Active patterns (v0.3+)
- ❌ Pattern macros (v0.3+)

---

## Design

### 1. Guards

#### Syntax

**Grammar Addition**:
```
pattern_clause ::= pattern ("if" expr)? "=>" expr
```

**Examples**:
```ailang
match x {
  0 => "zero",
  n if n > 0 => "positive",
  n if n < 0 => "negative",
  _ => "unreachable"
}

match pair {
  (x, y) if x == y => "equal",
  (x, y) if x > y => "first larger",
  (x, y) => "second larger"
}
```

#### AST Changes

**File**: `internal/ast/ast.go` (modifications)
**Size**: ~20 LOC

```go
// MatchClause represents a single match clause
type MatchClause struct {
    Pattern Pattern
    Guard   Expr   // NEW: Optional guard expression (nil if no guard)
    Body    Expr
    Pos     Position
}
```

#### Parser Changes

**File**: `internal/parser/parser.go` (modifications)
**Size**: ~50 LOC

```go
func (p *Parser) parseMatchClause() *ast.MatchClause {
    // Parse pattern
    pattern := p.parsePattern()

    // Check for guard
    var guard ast.Expr
    if p.curTokenIs(lexer.IF) {
        p.nextToken() // consume 'if'
        guard = p.parseExpression(LOWEST)
    }

    // Expect '=>'
    if !p.expectPeek(lexer.ARROW) {
        return nil
    }

    // Parse body
    body := p.parseExpression(LOWEST)

    return &ast.MatchClause{
        Pattern: pattern,
        Guard:   guard,
        Body:    body,
    }
}
```

#### Elaboration

**File**: `internal/elaborate/match.go` (modifications)
**Size**: ~50 LOC

```go
// Elaborate match clause with guard
func (e *Elaborator) elaborateMatchClause(clause *ast.MatchClause) (*core.MatchClause, error) {
    // Elaborate pattern
    pattern, bindings := e.elaboratePattern(clause.Pattern)

    // Elaborate guard (if present)
    var guard core.CoreExpr
    if clause.Guard != nil {
        // Add pattern bindings to environment for guard
        for name, typ := range bindings {
            e.env.Set(name, typ)
        }

        guardCore, err := e.elaborate(clause.Guard)
        if err != nil {
            return nil, err
        }

        // Guard must be Bool type
        if !e.typeChecker.IsCompatible(guardCore.Type(), types.BoolType) {
            return nil, fmt.Errorf("guard must be Bool, got %s", guardCore.Type())
        }

        guard = guardCore
    }

    // Elaborate body
    body, err := e.elaborate(clause.Body)
    if err != nil {
        return nil, err
    }

    return &core.MatchClause{
        Pattern: pattern,
        Guard:   guard,
        Body:    body,
    }, nil
}
```

#### Evaluation

**File**: `internal/eval/eval_core.go` (modifications)
**Size**: ~30 LOC

```go
func (e *CoreEvaluator) evalMatch(match *core.Match) (Value, error) {
    // Evaluate scrutinee
    scrutinee, err := e.evalCore(match.Scrutinee)
    if err != nil {
        return nil, err
    }

    // Try each clause
    for _, clause := range match.Clauses {
        // Try to match pattern
        bindings, ok := e.matchPattern(clause.Pattern, scrutinee)
        if !ok {
            continue // Pattern doesn't match, try next
        }

        // Check guard (if present)
        if clause.Guard != nil {
            // Push bindings for guard evaluation
            e.env.Push(bindings)

            guardVal, err := e.evalCore(clause.Guard)
            if err != nil {
                e.env.Pop()
                return nil, err
            }

            // Guard must evaluate to Bool
            boolVal, ok := guardVal.(*BoolValue)
            if !ok {
                e.env.Pop()
                return nil, fmt.Errorf("guard must be Bool, got %T", guardVal)
            }

            // Pop bindings
            e.env.Pop()

            // If guard is false, continue to next clause
            if !boolVal.Value {
                continue
            }
        }

        // Pattern matched and guard passed (if present)
        // Push bindings and evaluate body
        e.env.Push(bindings)
        result, err := e.evalCore(clause.Body)
        e.env.Pop()

        return result, err
    }

    // No clause matched
    return nil, fmt.Errorf("non-exhaustive match")
}
```

---

### 2. Exhaustiveness Checking

#### Algorithm

**File**: `internal/elaborate/exhaustiveness.go` (NEW)
**Size**: ~200 LOC

```go
// ExhaustivenessChecker checks if a match is exhaustive
type ExhaustivenessChecker struct {
    typeEnv *types.TypeEnv
}

// Check returns missing patterns (nil if exhaustive)
func (ec *ExhaustivenessChecker) Check(scrutineeType types.Type, patterns []Pattern) ([]Pattern, error) {
    // Build universe of all possible patterns for scrutineeType
    universe := ec.buildUniverse(scrutineeType)

    // Subtract covered patterns
    uncovered := universe
    for _, pattern := range patterns {
        uncovered = ec.subtract(uncovered, pattern)
    }

    return uncovered, nil
}

// buildUniverse creates all possible patterns for a type
func (ec *ExhaustivenessChecker) buildUniverse(typ types.Type) PatternSet {
    switch t := typ.(type) {
    case *types.TBool:
        return PatternSet{
            &LiteralPattern{Value: true},
            &LiteralPattern{Value: false},
        }

    case *types.TConstructor:
        // ADT: all constructors
        var patterns []Pattern
        for _, ctor := range t.Constructors {
            patterns = append(patterns, &ConstructorPattern{
                Name: ctor.Name,
                Args: ec.buildUniverses(ctor.Args),
            })
        }
        return patterns

    case *types.TList:
        return PatternSet{
            &ListPattern{Elements: []}, // []
            &ListPattern{Elements: []Pattern{WildcardPattern{}}}, // [_]
            &ListPattern{Elements: []Pattern{WildcardPattern{}, WildcardPattern{}}, Rest: &WildcardPattern{}}, // [_, _, ...]
        }

    default:
        // Infinite universe (Int, String, etc.) - use wildcard
        return PatternSet{&WildcardPattern{}}
    }
}

// subtract removes covered patterns from universe
func (ec *ExhaustivenessChecker) subtract(universe PatternSet, pattern Pattern) PatternSet {
    var remaining PatternSet

    for _, universePattern := range universe {
        if !ec.covers(pattern, universePattern) {
            remaining = append(remaining, universePattern)
        }
    }

    return remaining
}

// covers checks if pattern1 covers pattern2
func (ec *ExhaustivenessChecker) covers(p1, p2 Pattern) bool {
    // Wildcard covers everything
    if _, ok := p1.(*WildcardPattern); ok {
        return true
    }

    // Variable covers everything
    if _, ok := p1.(*VariablePattern); ok {
        return true
    }

    // Constructor: must match constructor and all args
    if c1, ok := p1.(*ConstructorPattern); ok {
        c2, ok := p2.(*ConstructorPattern)
        if !ok || c1.Name != c2.Name {
            return false
        }

        // Check all arguments
        for i := range c1.Args {
            if !ec.covers(c1.Args[i], c2.Args[i]) {
                return false
            }
        }

        return true
    }

    // Literal: must match exactly
    if l1, ok := p1.(*LiteralPattern); ok {
        l2, ok := p2.(*LiteralPattern)
        return ok && l1.Value == l2.Value
    }

    return false
}
```

#### Warning Generation

**File**: `internal/elaborate/warnings.go` (NEW)
**Size**: ~100 LOC

```go
// GenerateExhaustivenessWarning creates a warning for non-exhaustive match
func GenerateExhaustivenessWarning(pos Position, missing []Pattern) *Warning {
    var examples []string
    for _, pattern := range missing {
        examples = append(examples, formatPattern(pattern))
    }

    return &Warning{
        Code:     "PAT_NONEXHAUSTIVE",
        Message:  "non-exhaustive pattern match",
        Position: pos,
        Suggestion: fmt.Sprintf("Missing cases:\n  %s", strings.Join(examples, "\n  ")),
    }
}

// formatPattern pretty-prints a pattern
func formatPattern(p Pattern) string {
    switch pt := p.(type) {
    case *WildcardPattern:
        return "_"
    case *VariablePattern:
        return pt.Name
    case *ConstructorPattern:
        if len(pt.Args) == 0 {
            return pt.Name
        }
        args := make([]string, len(pt.Args))
        for i, arg := range pt.Args {
            args[i] = formatPattern(arg)
        }
        return fmt.Sprintf("%s(%s)", pt.Name, strings.Join(args, ", "))
    case *LiteralPattern:
        return fmt.Sprintf("%v", pt.Value)
    case *ListPattern:
        // Format list pattern
        if len(pt.Elements) == 0 {
            return "[]"
        }
        // ...
    default:
        return "<?>"
    }
}
```

#### CLI Output

```bash
$ ailang run incomplete_match.ail

Warning: PAT_NONEXHAUSTIVE (incomplete_match.ail:5:3)
  Non-exhaustive pattern match

  Missing cases:
    None

  Example:
    match opt {
      Some(x) => x,
      None => 0  -- Add this case
    }
```

---

### 3. Decision Tree Compilation

#### Decision Tree Structure

**File**: `internal/elaborate/decision_tree.go` (NEW)
**Size**: ~150 LOC

```go
// DecisionTree represents a compiled pattern match
type DecisionTree interface {
    isDecisionTree()
}

// Leaf: pattern matched, execute body
type Leaf struct {
    Bindings map[string]Value
    Body     core.CoreExpr
}

// Switch: test a value and branch
type Switch struct {
    Path     AccessPath // How to access the value to test
    Branches map[any]DecisionTree // Constructor/literal → subtree
    Default  DecisionTree // Wildcard case
}

// Guard: check a condition
type Guard struct {
    Condition core.CoreExpr
    Then      DecisionTree
    Else      DecisionTree
}

// Fail: no pattern matched
type Fail struct{}

// AccessPath describes how to reach a sub-value
type AccessPath struct {
    Root  string // Variable name (usually scrutinee)
    Steps []AccessStep
}

type AccessStep interface {
    isAccessStep()
}

type FieldAccess struct{ Field string }
type IndexAccess struct{ Index int }
type DeconstructArg struct{ ArgIndex int }
```

#### Compilation

```go
// CompileMatch compiles patterns to a decision tree
func CompileMatch(scrutinee string, clauses []*core.MatchClause) DecisionTree {
    // Initial state: all clauses, empty path
    return compile(AccessPath{Root: scrutinee}, clauses)
}

func compile(path AccessPath, clauses []*core.MatchClause) DecisionTree {
    if len(clauses) == 0 {
        return &Fail{}
    }

    // Check if first clause is unconditional leaf
    if isUnconditional(clauses[0]) {
        return &Leaf{
            Bindings: extractBindings(clauses[0].Pattern),
            Body:     clauses[0].Body,
        }
    }

    // Build switch on first discriminator
    discriminator := findDiscriminator(clauses)
    return buildSwitch(path, discriminator, clauses)
}

func buildSwitch(path AccessPath, disc Discriminator, clauses []*core.MatchClause) DecisionTree {
    // Group clauses by constructor/literal
    groups := groupBy(clauses, disc)

    branches := make(map[any]DecisionTree)
    for key, group := range groups {
        // Recurse on subpatterns
        subPath := path.Extend(disc)
        branches[key] = compile(subPath, group)
    }

    return &Switch{
        Path:     path,
        Branches: branches,
        Default:  compile(path, groups[wildcard]),
    }
}
```

#### Evaluation

**File**: `internal/eval/decision_tree.go` (NEW)
**Size**: ~100 LOC

```go
func (e *CoreEvaluator) evalDecisionTree(tree DecisionTree, scrutinee Value) (Value, error) {
    switch t := tree.(type) {
    case *Leaf:
        // Push bindings and evaluate body
        e.env.Push(t.Bindings)
        result, err := e.evalCore(t.Body)
        e.env.Pop()
        return result, err

    case *Switch:
        // Access the value at path
        val := e.accessPath(scrutinee, t.Path)

        // Match against branches
        for key, subtree := range t.Branches {
            if e.matches(val, key) {
                return e.evalDecisionTree(subtree, scrutinee)
            }
        }

        // Fall back to default
        return e.evalDecisionTree(t.Default, scrutinee)

    case *Guard:
        // Evaluate condition
        condVal, err := e.evalCore(t.Condition)
        if err != nil {
            return nil, err
        }

        boolVal := condVal.(*BoolValue)
        if boolVal.Value {
            return e.evalDecisionTree(t.Then, scrutinee)
        } else {
            return e.evalDecisionTree(t.Else, scrutinee)
        }

    case *Fail:
        return nil, fmt.Errorf("non-exhaustive match")

    default:
        return nil, fmt.Errorf("unknown decision tree node: %T", t)
    }
}
```

---

## Implementation Plan

### Phase 1: Guards (Days 1-2)

**Goal**: Add guard syntax and evaluation

**Tasks**:
1. Update AST for guards (~20 LOC)
2. Parser changes (~50 LOC)
3. Elaboration changes (~50 LOC)
4. Evaluation changes (~30 LOC)
5. Tests (~150 LOC)

**Deliverable**: Guards working

### Phase 2: Exhaustiveness (Days 3-4)

**Goal**: Warn on incomplete matches

**Tasks**:
1. Implement exhaustiveness checker (~200 LOC)
2. Warning generation (~100 LOC)
3. CLI integration (~20 LOC)
4. Tests (~200 LOC)

**Deliverable**: Warnings working

### Phase 3: Decision Trees (Days 5-7)

**Goal**: Compile matches to decision trees

**Tasks**:
1. Decision tree structure (~100 LOC)
2. Compilation (~150 LOC)
3. Evaluation (~100 LOC)
4. Tests (~200 LOC)

**Deliverable**: Decision trees working, faster matching

---

## Testing Strategy

### Unit Tests (~700 LOC)

**Test Files**:
- `internal/parser/parser_test.go` (guards, +50 LOC)
- `internal/elaborate/exhaustiveness_test.go` (NEW, ~200 LOC)
- `internal/elaborate/decision_tree_test.go` (NEW, ~200 LOC)
- `internal/eval/eval_core_test.go` (guards, +100 LOC)
- `internal/eval/decision_tree_test.go` (NEW, ~150 LOC)

**Test Cases**:

1. **Guards**
   - Simple guards (`n if n > 0`)
   - Multiple guards
   - Guard with complex condition
   - Guard evaluation order

2. **Exhaustiveness**
   - Complete matches (no warning)
   - Missing constructor (warning)
   - Missing list cases (warning)
   - Wildcard makes exhaustive

3. **Decision Trees**
   - Linear patterns → linear tree
   - Nested patterns → nested tree
   - Shared prefixes → optimized tree
   - Performance vs naive (benchmark)

### Integration Tests (~100 LOC)

**Test File**: `tests/integration/pattern_matching_test.go`

**Test Cases**:

1. **Guards with ADTs**
   ```ailang
   match opt {
     Some(x) if x > 0 => x,
     Some(x) => -x,
     None => 0
   }
   ```

2. **Exhaustiveness Warning**
   ```ailang
   match opt {
     Some(x) => x
     -- Expect warning: Missing case: None
   }
   ```

3. **Decision Tree Performance**
   - Benchmark: 1000 matches
   - Compare naive vs decision tree

---

## Acceptance Criteria

### Minimum Success

- ✅ Guards parse and evaluate correctly
- ✅ Exhaustiveness warnings work for ADTs
- ✅ All existing PM tests still pass
- ✅ 5+ new tests for guards/exhaustiveness

### Stretch Goals

- ✅ Decision trees implemented
- ✅ Decision trees faster than naive (benchmark proof)
- ✅ Exhaustiveness for lists and tuples
- ✅ 15+ new tests

---

## Risks & Mitigation

| Risk | Impact | Mitigation | Fallback |
|------|--------|-----------|----------|
| Exhaustiveness false positives | Med | Conservative checking; warn only | Add `--no-exhaustive-check` flag |
| Decision tree bugs | Med | Extensive testing; keep naive path | Use naive if decision tree fails |
| Performance regression | Low | Benchmark; optimize hot paths | Defer decision trees to v0.3.0 |

---

## Future Extensions (Post-v0.2.0)

### v0.3.0: Advanced Patterns
- Or-patterns: `Some(x) \| None => ...`
- As-patterns: `x @ Some(y)`
- View patterns: `view f -> pattern`

### v0.3.0: Refinement Types
- Patterns with refinements: `x where x > 0`

### v0.4.0: Pattern Macros
- User-defined pattern syntax

---

## Status

**Status**: Design complete, ready for implementation
**Depends On**: Nothing (can start immediately, but better after M-R1 stabilizes)
**Blocks**: Nothing (stretch goal)

---

**Document Version**: v1.0
**Created**: 2025-10-02
**Last Updated**: 2025-10-02
**Author**: AILANG Development Team
