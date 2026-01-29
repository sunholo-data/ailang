# Hello World Feature

**Status:** Planned
**Target:** v0.7.1
**Priority:** P2 (Low)
**Estimated:** 1 hour
**Dependencies:** None

## Problem Statement

Currently there is no simple hello world example to demonstrate basic AILANG functionality. Users need a minimal entry point to understand the language.

## Goals

**Primary Goal:** Create a minimal hello world example that outputs text

**Success Metrics:**
- Simple, clear example exists
- Works out of the box
- Documentation is updated

## Solution Design

### Overview

Create a minimal AILANG program that outputs "Hello, World!" to demonstrate:
- Module structure
- IO effects
- Entry point execution

### Architecture

Simple module with main entry point using IO capability.

### Implementation Plan

**Phase 1: Implementation** (30 minutes)
- [ ] Create `examples/hello_world.ail`
- [ ] Test with `ailang run --caps IO --entry main`
- [ ] Verify output is correct

**Phase 2: Documentation** (30 minutes)
- [ ] Add to example documentation
- [ ] Update getting started guide
- [ ] Add to README examples

### Files to Modify/Create

- `examples/hello_world.ail` (new, ~10 LOC)
- `docs/README.md` (modify, +5 LOC)

## Example Usage

### Before
No simple hello world example exists.

### After
```ailang
module hello_world

import std/io (println)

let main : ! {IO} unit =
  println("Hello, World!")
```

Run with:
```bash
ailang run --caps IO --entry main examples/hello_world.ail
```

## Related Documents

- `design_docs/implemented/v0_3_14/list_pattern_spread.md` - Shows module structure patterns
- `examples/runnable/hello.ail` - Existing hello example (if any)

## Success Criteria

- [ ] Hello world example created and works
- [ ] Example runs without errors
- [ ] Output is exactly "Hello, World!"
- [ ] Documentation updated
- [ ] All tests passing

## Timeline

**Week 1:**
- Implementation and testing
- Documentation updates

**Total time:** ~1 hour

## Notes

This is a minimal design doc for a simple hello world feature demonstration.