# M-DX7: JSON AST Output & Structured Querying

**Status**: Planned
**Target**: v0.3.18
**Priority**: P1 (High)
**Estimated**: 3-4 hours
**Dependencies**: M-DX2-M3 (Debug CLI - completed)
**Related**: M-DX6 (Pipeline Visualization)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | Tooling, doesn't affect language syntax |
| Preserve Semantic Clarity | Positive | +2 | **Structured data > text parsing** |
| Increase Determinism | Positive | +1 | Programmatic queries eliminate interpretation ambiguity |
| Lower Token Cost | Positive | +2 | **AI parses JSON directly, no LLM calls for parsing** |
| **Net Score** | | **+5** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Rationale**: Current text output forces AI to parse unstructured text or read source code. JSON enables programmatic queries with zero LLM tokens.

## Problem Statement

**Discovered during**: AILANG Ease of Use Assessment (October 2025)

**Current State:**
`ailang debug ast` exists (M-DX2-M3) but outputs text only:
- ✅ Shows Core AST structure
- ✅ Shows types with `--show-types`
- ✅ Shows NodeIDs
- ❌ Text output requires parsing (grep/awk or LLM)
- ❌ No filtering by node type
- ❌ No querying by type properties
- ❌ No machine-readable format

**Pain Points:**
1. **Text parsing overhead**: AI must parse text output or use LLM to extract info
2. **No filtering**: To find all `Intrinsic` nodes, must grep text output
3. **No querying**: To find nodes with type `List[int]`, must parse types manually
4. **No programmatic access**: Can't integrate with tools (jq, scripts, etc.)

**Impact:**
- **Who**: AI agents building AILANG features
- **Significance**: AI wastes tokens/time parsing text instead of using structured data
- **Example**: "Find all Intrinsic nodes with type List[int]" requires:
  - Text approach: Run `ailang debug ast --show-types`, parse output with LLM
  - JSON approach: Run `ailang debug ast --format json | jq '.nodes[] | select(.kind == "Intrinsic" and .type.kind == "List" and .type.elem.kind == "Int")'`

**Metrics:**
- Current: Text-only output
- Token cost: ~500 tokens to parse simple AST with LLM
- Query time: ~2-3 seconds (LLM call)
- JSON approach: 0 tokens, <100ms (jq)

## Goals

**Primary Goal:** Enable programmatic AST queries without LLM parsing.

**Success Metrics:**
- AI can extract AST info via JSON (no text parsing)
- AI can filter by node type without grepping
- AI can query by type properties using jq
- Integration with standard tools (jq, scripts)

## Solution Design

### Overview

Extend `ailang debug ast` command with:
1. **JSON output mode** (`--format json`)
2. **Node filtering** (`--filter Intrinsic`)
3. **Type querying** (`--query "type.kind == List"`)
4. **Surface AST view** (`--surface` flag)

**Strategy**: Minimal code changes to existing debug command.

### Architecture

**Component**: `cmd/ailang/debug.go` (already exists from M-DX2-M3)

**New functionality**:
1. JSON serializer for Core AST nodes
2. Filtering logic (by node kind)
3. Query DSL (simple property matching)
4. Surface AST printer (for comparison)

### JSON Schema Design

**Core AST JSON structure:**

```json
{
  "version": "v0.3.18",
  "ast_type": "Core",
  "file": "example.ail",
  "nodes": [
    {
      "id": 13,
      "kind": "Let",
      "name": "xs",
      "type": {
        "kind": "List",
        "elem": {"kind": "Int"}
      },
      "value": {
        "node_id": 4,
        "kind": "List",
        "elements": [
          {"node_id": 1, "kind": "Lit", "value": 1, "type": {"kind": "Int"}},
          {"node_id": 2, "kind": "Lit", "value": 2, "type": {"kind": "Int"}},
          {"node_id": 3, "kind": "Lit", "value": 3, "type": {"kind": "Int"}}
        ]
      },
      "body": {"node_id": 12}
    },
    {
      "id": 11,
      "kind": "Intrinsic",
      "op": "OpConcat",
      "op_id": 11,
      "type": {
        "kind": "List",
        "elem": {"kind": "Int"}
      },
      "args": [
        {"node_id": 9, "kind": "Var", "name": "xs", "type": {"kind": "List", "elem": {"kind": "Int"}}},
        {"node_id": 10, "kind": "Var", "name": "ys", "type": {"kind": "List", "elem": {"kind": "Int"}}}
      ]
    }
  ]
}
```

**Type representation:**

```json
// Simple types
{"kind": "Int"}
{"kind": "Float"}
{"kind": "String"}
{"kind": "Bool"}

// Compound types
{"kind": "List", "elem": {"kind": "Int"}}
{"kind": "Tuple", "elems": [{"kind": "Int"}, {"kind": "String"}]}
{"kind": "Function", "params": [{"kind": "Int"}], "return": {"kind": "String"}}

// Polymorphic types
{"kind": "TypeVar", "name": "a", "id": 7}
{"kind": "Forall", "vars": ["a"], "body": {"kind": "Function", "params": [{"kind": "TypeVar", "name": "a", "id": 7}], "return": {"kind": "TypeVar", "name": "a", "id": 7}}}

// Effects
{"kind": "Function", "params": [{"kind": "String"}], "return": {"kind": "Unit"}, "effects": ["IO"]}
```

### CLI Design

**Flags:**

```bash
# JSON output mode
--format json|text|compact  # Default: text

# Node filtering
--filter <NodeKind>         # e.g., --filter Intrinsic

# Type querying (simple syntax)
--query <expr>              # e.g., --query "type.kind == List"

# AST selection
--surface                   # Show Surface AST instead of Core

# Existing flags (from M-DX2-M3)
--show-types                # Include type annotations
--compact                   # Compact text output
```

**Examples:**

```bash
# Get JSON output
ailang debug ast example.ail --format json

# Filter to Intrinsic nodes only
ailang debug ast example.ail --filter Intrinsic --format json

# Query nodes with List type
ailang debug ast example.ail --query "type.kind == List" --format json

# Show Surface AST (pre-elaboration)
ailang debug ast example.ail --surface --format json

# Combine with jq for complex queries
ailang debug ast example.ail --format json | jq '.nodes[] | select(.kind == "Intrinsic" and .op == "OpConcat")'
```

### Implementation Plan

**Phase 1: JSON Serializer** (~1.5 hours)
- [ ] Create `cmd/ailang/debug_json.go`
- [ ] Implement `ToJSON(node core.CoreExpr, ti CoreTypeInfo) map[string]interface{}`
- [ ] Implement type serialization: `TypeToJSON(t types.Type) map[string]interface{}`
- [ ] Handle all Core node types:
  - Lit, Var, VarGlobal, Let, Lambda, App, If, List, Intrinsic, Match, etc.
- [ ] Unit test: Serialize example nodes, verify JSON structure

**Phase 2: Filtering Logic** (~0.5 hours)
- [ ] Add `--filter` flag to `cmd/ailang/debug.go`
- [ ] Implement filter by node kind (string match)
- [ ] Apply filter before printing (skip non-matching nodes)
- [ ] Test: `--filter Intrinsic` shows only Intrinsic nodes

**Phase 3: Query DSL** (~1 hour)
- [ ] Design simple query syntax: `field.subfield op value`
  - Operators: `==`, `!=`, `contains`
  - Fields: `kind`, `type.kind`, `type.elem.kind`, `name`, `op`
- [ ] Implement query parser: `ParseQuery(expr string) Query`
- [ ] Implement query evaluator: `Matches(node, query) bool`
- [ ] Test: `--query "type.kind == List"` filters correctly

**Phase 4: Surface AST Support** (~0.5 hours)
- [ ] Add `--surface` flag
- [ ] Implement Surface AST → JSON (similar to Core)
- [ ] Test: `--surface --format json` shows Surface AST

**Phase 5: Integration & Docs** (~0.5 hours)
- [ ] Update help text in `cmd/ailang/debug.go`
- [ ] Add examples to `docs/guides/debugging.md` (new file)
- [ ] Test all flag combinations
- [ ] Update CLAUDE.md with JSON examples

### Files to Modify/Create

**New files:**
- `cmd/ailang/debug_json.go` - JSON serializer (~200 LOC)
- `cmd/ailang/debug_query.go` - Query parser/evaluator (~100 LOC)
- `docs/guides/debugging.md` - Debugging guide with JSON examples (~200 LOC)

**Modified files:**
- `cmd/ailang/debug.go` - Add flags, integrate JSON/query (~50 LOC)
- `CLAUDE.md` - Add JSON examples (~20 LOC)

**Total new code:** ~500 LOC
**Total modified code:** ~70 LOC

## Examples

### Example 1: JSON Output

**Command:**
```bash
ailang debug ast concat.ail --format json
```

**Output:**
```json
{
  "version": "v0.3.18",
  "ast_type": "Core",
  "file": "concat.ail",
  "nodes": [
    {
      "id": 13,
      "kind": "Let",
      "name": "xs",
      "type": {"kind": "List", "elem": {"kind": "Int"}},
      "value": {
        "node_id": 4,
        "kind": "List",
        "elements": [
          {"node_id": 1, "kind": "Lit", "value": 1, "type": {"kind": "Int"}},
          {"node_id": 2, "kind": "Lit", "value": 2, "type": {"kind": "Int"}},
          {"node_id": 3, "kind": "Lit", "value": 3, "type": {"kind": "Int"}}
        ]
      },
      "body": {"node_id": 12}
    },
    {
      "id": 11,
      "kind": "Intrinsic",
      "op": "OpConcat",
      "op_id": 11,
      "type": {"kind": "List", "elem": {"kind": "Int"}},
      "args": [
        {"node_id": 9, "kind": "Var", "name": "xs"},
        {"node_id": 10, "kind": "Var", "name": "ys"}
      ]
    }
  ]
}
```

### Example 2: Filtering

**Command:**
```bash
ailang debug ast concat.ail --filter Intrinsic --format json
```

**Output:**
```json
{
  "version": "v0.3.18",
  "ast_type": "Core",
  "file": "concat.ail",
  "nodes": [
    {
      "id": 11,
      "kind": "Intrinsic",
      "op": "OpConcat",
      "op_id": 11,
      "type": {"kind": "List", "elem": {"kind": "Int"}},
      "args": [...]
    }
  ]
}
```

### Example 3: Querying with jq

**Command:**
```bash
ailang debug ast concat.ail --format json | jq '.nodes[] | select(.kind == "Intrinsic" and .op == "OpConcat")'
```

**Output:**
```json
{
  "id": 11,
  "kind": "Intrinsic",
  "op": "OpConcat",
  "op_id": 11,
  "type": {"kind": "List", "elem": {"kind": "Int"}},
  "args": [...]
}
```

### Example 4: Built-in Query DSL

**Command:**
```bash
ailang debug ast concat.ail --query "type.kind == List" --format json
```

**Output:**
```json
{
  "version": "v0.3.18",
  "ast_type": "Core",
  "file": "concat.ail",
  "nodes": [
    {"id": 4, "kind": "List", "type": {"kind": "List", "elem": {"kind": "Int"}}, ...},
    {"id": 8, "kind": "List", "type": {"kind": "List", "elem": {"kind": "Int"}}, ...},
    {"id": 11, "kind": "Intrinsic", "type": {"kind": "List", "elem": {"kind": "Int"}}, ...}
  ]
}
```

## Success Criteria

- [ ] `--format json` produces valid JSON
- [ ] JSON schema includes all Core node types
- [ ] `--filter <kind>` filters correctly
- [ ] `--query` DSL works for common queries
- [ ] `--surface` shows Surface AST
- [ ] Integration with jq works
- [ ] Documentation includes examples
- [ ] All existing tests pass (no regressions)

## Testing Strategy

**Unit tests:**
- `cmd/ailang/debug_json_test.go`: JSON serialization
  - Serialize each Core node type
  - Verify JSON structure
  - Round-trip test (JSON → parse → compare)
- `cmd/ailang/debug_query_test.go`: Query parsing/evaluation
  - Parse valid queries
  - Reject invalid queries
  - Evaluate queries on sample nodes

**Integration tests:**
- Run on example files:
  - `ailang debug ast examples/factorial.ail --format json`
  - `ailang debug ast examples/list_ops.ail --filter Intrinsic --format json`
  - `ailang debug ast examples/hello.ail --query "type.kind == String" --format json`
- Verify jq integration:
  - `ailang debug ast ... --format json | jq '.nodes[] | select(...)'`

**Manual testing:**
- Test all flag combinations
- Verify JSON is valid (use jq)
- Check performance on large files

## Non-Goals

**Not in this feature:**
- Interactive query builder (CLI-only, no GUI)
- Streaming JSON (for large files) - deferred to v0.4.0+
- JSON schema validation (docs only, no validation)
- Pretty-printing JSON (use `jq` instead)

**Why deferred:**
- Interactive builder requires GUI (not AI-first)
- Streaming adds complexity for minimal benefit (AILANG files are small)
- Schema validation is nice-to-have, not critical

## Timeline

**Week 1** (3-4 hours):
- Day 1: Phase 1 (JSON serializer) - 1.5h
- Day 1: Phase 2 (Filtering) - 0.5h
- Day 2: Phase 3 (Query DSL) - 1h
- Day 2: Phase 4 (Surface AST) - 0.5h
- Day 2: Phase 5 (Integration & Docs) - 0.5h

**Total: ~3-4 hours across 2 days**

**Buffer:** Already applied (raw estimate was 2.5h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| JSON serialization bugs | High | Comprehensive unit tests for all node types |
| Query DSL complexity | Medium | Start simple, iterate based on feedback |
| Performance on large files | Low | AILANG files are small; defer streaming if needed |
| Breaking changes | Low | Additive only, existing text mode unchanged |

## References

- **Prior art**: M-DX2-M3 Debug CLI (v0.3.16)
  - [Completion report](../v0_3_16/M-DX2-M3-COMPLETE.md)
- **Related**: AILANG Ease of Use Assessment
  - [Assessment](./AILANG_EASE_OF_USE_ASSESSMENT.md)
- **Existing implementation**:
  - `cmd/ailang/debug.go` - Text-based debug command

## Future Work

**Potential extensions (not in v0.3.18):**

1. **Streaming JSON** (v0.4.0+)
   - For very large files, stream nodes one at a time
   - Useful for scalability

2. **Advanced query DSL** (v0.4.0+)
   - Boolean operators: `--query "kind == Intrinsic AND type.kind == List"`
   - Negation: `--query "kind != Lit"`
   - Regex: `--query "name matches 'temp.*'"`

3. **JSON schema file** (v0.3.19+)
   - Formal schema for validation
   - Useful for tooling

4. **Diff mode** (v0.4.0+)
   - Compare two ASTs: `ailang debug ast file1.ail file2.ail --diff`
   - Show structural differences

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-22
