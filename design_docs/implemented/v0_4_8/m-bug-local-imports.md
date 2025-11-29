# M-BUG-LOCAL-IMPORTS: Fix Local Module Import Resolution

**Status**: Planned
**Target**: v0.4.9
**Priority**: P0 (High) - Blocking external game development
**Estimated**: 1-2 days
**Dependencies**: None
**Reported by**: stapledons_voyage (agent inbox)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Users can organize code across files without duplication |
| Preserve Semantic Clarity | 0 | 0 | No change to semantics |
| Increase Determinism | + | +1 | Imports resolve consistently (currently broken) |
| Lower Token Cost | + | +1 | Eliminates duplicated type definitions across files |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

When trying to import from local files within a multi-file project, users get error `LDR001: module not found`.

**Reproduction:**
```
project/
├── sim/
│   ├── world.ail      # imports sim/protocol
│   └── protocol.ail   # defines Coord type
└── main.ail           # entry point
```

```ailang
-- sim/world.ail
module sim/world
import sim/protocol (Coord)  -- ERROR: LDR001: module not found: sim/protocol

-- ...rest of file uses Coord
```

**Current Workaround:** Duplicate type definitions in each file (defeats module system purpose).

**Current State:**
- `ailang run` from project root should resolve `sim/protocol` relative to project
- Error: `LDR001: module not found: sim/protocol`
- Both files exist and type-check individually
- The ailang prompt shows multi-module projects but imports don't work in practice

**Impact:**
- Blocks all multi-file projects
- Forces code duplication
- Particularly affects game development (PlanetWorld/stapledons_voyage)

## Goals

**Primary Goal:** Fix local module import resolution so `import sim/protocol` works from any file in the project.

**Success Metrics:**
- `import local/path (Type)` resolves correctly from any file
- Same module can be imported from multiple files
- Works with `ailang run --entry main`
- Clear error message if module truly doesn't exist

## Solution Design

### Overview

The loader needs to resolve module paths relative to the project root (where `ailang run` was invoked), not relative to the importing file.

### Architecture

**Root Cause Investigation:**

The LDR001 error is generated in multiple places:
1. `internal/loader/loader.go:137` - Main loader
2. `internal/link/topo.go:120` - Topological sort
3. `internal/link/module_linker.go:63` - Module linker

The issue is likely one of:
1. **Search path not including project root** - Loader only searches stdlib, not CWD
2. **Relative path calculation wrong** - Paths calculated relative to importing file, not project
3. **Module discovery not recursive** - Only top-level .ail files found

**Proposed Fix:**

1. When `ailang run --entry main file.ail` is invoked:
   - Record project root as directory containing entry file
   - Add project root to module search path
2. When resolving `import foo/bar`:
   - First check stdlib (`std/foo/bar`)
   - Then check project root (`projectRoot/foo/bar.ail`)
3. Search recursively for `.ail` files in project

### Implementation Plan

**Phase 1: Diagnosis** (~2 hours)
- [ ] Add debug logging to loader.go to trace search paths
- [ ] Create minimal reproduction case
- [ ] Identify exact failure point

**Phase 2: Fix** (~4 hours)
- [ ] Add project root to loader search paths
- [ ] Ensure recursive file discovery
- [ ] Handle relative imports correctly

**Phase 3: Testing** (~2 hours)
- [ ] Add integration test with multi-file project
- [ ] Test nested directories (sim/sub/module)
- [ ] Verify existing tests still pass

### Files to Modify

**Modified files:**
- `internal/loader/loader.go` - Add project root search path (~20 LOC)
- `internal/module/loader.go` - Fix path resolution (~20 LOC)
- `cmd/ailang/main.go` - Pass project root to loader (~5 LOC)

**New files:**
- `tests/integration/multi_module_test.go` - Integration test (~50 LOC)
- `examples/multi_module/` - Example project structure

## Examples

### Example 1: Two-File Project

**Project structure:**
```
game/
├── types.ail       # type definitions
└── main.ail        # uses types
```

**types.ail:**
```ailang
module game/types

type Coord = { x: int, y: int }
type Entity = { pos: Coord, health: int }
```

**main.ail:**
```ailang
module game/main

import game/types (Coord, Entity)

func spawn(x: int, y: int) -> Entity {
  { pos: { x: x, y: y }, health: 100 }
}

entry func main() ! {IO} {
  let player = spawn(0, 0)
  print("Spawned at: " ++ show(player.pos.x))
}
```

**Expected:** `ailang run --entry main --caps IO game/main.ail` works

### Example 2: Nested Directories

```
sim/
├── core/
│   ├── types.ail
│   └── physics.ail
├── entities/
│   └── npc.ail      # imports sim/core/types
└── main.ail
```

**Expected:** `import sim/core/types (Vec2)` works from any file

## Success Criteria

- [ ] `import local/path (Type)` resolves from project root
- [ ] Works with `ailang run --entry main`
- [ ] Works with nested directories
- [ ] Same module can be imported from multiple files
- [ ] Clear error if module file doesn't exist (with search paths shown)
- [ ] All existing tests pass
- [ ] Integration test added

## Testing Strategy

**Unit tests:**
- Path resolution logic
- Search path ordering (stdlib before project)

**Integration tests:**
- Multi-file project imports
- Nested directory imports
- Circular import detection

**Manual testing:**
- Run stapledons_voyage project after fix

## Non-Goals

- **Not in this fix:**
  - Relative imports (`import ./sibling`) - use absolute from project root
  - Package management - just local file resolution
  - Import caching/optimization

## Timeline

**Day 1** (4 hours):
- Diagnosis and minimal reproduction
- Implement fix

**Day 2** (4 hours):
- Testing and edge cases
- Documentation update

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Multiple loader implementations | Medium | Fix all three LDR001 sources |
| Breaking existing imports | High | Run full test suite before merge |
| Path separator issues (Windows) | Low | Use filepath.Join throughout |

## References

- stapledons_voyage bug report (agent inbox, 2025-11-28)
- `internal/loader/loader.go` - Main module loader
- `internal/link/topo.go` - Topological sort for imports
- `internal/errors/codes.go` - LDR001 error code definition

## Future Work

- Relative imports (`import ./sibling`)
- Import path aliases
- Package manifest (ailang.toml)

---

**Document created**: 2025-11-28
**Last updated**: 2025-11-28
