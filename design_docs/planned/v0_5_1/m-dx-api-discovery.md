# M-DX-API-DISCOVERY: Expose Existing Builtin Metadata

**Version**: v0.5.1
**Priority**: High (major DX pain point)
**Estimated Effort**: 1-2 hours
**Status**: Planned

## Problem Statement

Users (including AI agents) have to guess-and-check to discover function signatures. From stapledons_voyage:

> "I tried `rand_int(4)` first, got 'arity mismatch: 2 vs 1'. Had to experiment to find `rand_int(0, 4)`."

Key issues:
1. `ailang builtins list` exists but **doesn't show signatures or descriptions**
2. `ailang iface std/rand` doesn't work for stdlib modules
3. `ailang prompt` doesn't document effect APIs (rand, debug, clock, etc.)

## Current State: Metadata Already Exists!

The builtin metadata system is **already complete** (M-DX1.11):

```go
// internal/builtins/metadata.go
type BuiltinMetadata struct {
    Description string      // ✅ "Generate random integer in range [min, max]"
    Params      []ParamDoc  // ✅ [{Name: "min", Description: "..."}]
    Returns     string      // ✅ "Random integer in [min, max]"
    Examples    []Example   // ✅ [{Code: "_rand_int(1, 6)", Description: "Roll d6"}]
    Since       string      // ✅ "v0.5.1"
    Stability   Stability   // ✅ StabilityStable
    Tags        []string    // ✅ ["random", "rand", "int", "game"]
    Category    string      // ✅ "rand"
}
```

Example from `internal/builtins/rand.go`:
```go
Metadata: &BuiltinMetadata{
    Description: "Generate random integer in range [min, max] inclusive",
    Params: []ParamDoc{
        {Name: "min", Description: "Minimum value (inclusive)"},
        {Name: "max", Description: "Maximum value (inclusive)"},
    },
    Returns: "Random integer in [min, max]",
    Examples: []Example{
        {Code: "_rand_int(1, 6)", Description: "Roll a d6 (returns 1-6)"},
    },
}
```

**The metadata exists - we just need to expose it!**

## Proposed Solution

### 1. Add `--verbose` flag to `ailang builtins list`

```bash
# Current (shows only names)
$ ailang builtins list --by-module
# std/rand (4)
  _rand_int                      [rand]
  _rand_float                    [rand]

# Proposed (shows full docs)
$ ailang builtins list --by-module --verbose
# std/rand (4)

  _rand_int(min: int, max: int) -> int ! {Rand}
    Generate random integer in range [min, max] inclusive
    Examples:
      _rand_int(1, 6)       -- Roll a d6 (returns 1-6)
      _rand_int(0, 100)     -- Random percentage

  _rand_float(min: float, max: float) -> float ! {Rand}
    Generate random float in range [min, max)
    ...
```

### 2. Add `ailang builtins show <name>` subcommand

```bash
$ ailang builtins show _rand_int
_rand_int(min: int, max: int) -> int ! {Rand}

Description:
  Generate random integer in range [min, max] inclusive

Parameters:
  min: int    Minimum value (inclusive)
  max: int    Maximum value (inclusive)

Returns:
  Random integer in [min, max]

Examples:
  _rand_int(1, 6)        Roll a d6 (returns 1-6)
  _rand_int(0, 100)      Random percentage

Since: v0.5.1
Stability: stable
Tags: random, rand, int, game
Module: std/rand
```

### 3. Add JSON output for tooling

```bash
$ ailang builtins list --format json > builtins.json
```

Output includes all metadata for IDE/AI tool integration.

## Implementation

### Files to Modify

1. **cmd/ailang/doctor.go** - Add flags to `runBuiltinsList()`
   - Add `--verbose` flag
   - Add `--format` flag (text/json)
   - Add `show` subcommand

2. **internal/builtins/spec.go** - Add helper to format signature
   ```go
   func (s *BuiltinSpec) FormatSignature() string {
       t := s.Type()
       return fmt.Sprintf("%s: %s", s.Name, t.String())
   }
   ```

### Code Changes

```go
// cmd/ailang/doctor.go - runBuiltinsList()

func runBuiltinsList() {
    listFlags := flag.NewFlagSet("list", flag.ExitOnError)
    byEffect := listFlags.Bool("by-effect", false, "Group by effect type")
    byModule := listFlags.Bool("by-module", false, "Group by module")
    verbose := listFlags.Bool("verbose", false, "Show full documentation")  // NEW
    format := listFlags.String("format", "text", "Output format (text/json)")  // NEW
    _ = listFlags.Parse(flag.Args()[2:])

    specs := builtins.AllSpecs()

    if *format == "json" {
        outputBuiltinsJSON(specs)
        return
    }

    if *verbose {
        listBuiltinsVerbose(specs, *byModule, *byEffect)
    } else if *byEffect {
        listBuiltinsByEffect(specs)
    } else if *byModule {
        listBuiltinsByModule(specs)
    } else {
        listAllBuiltins(specs)
    }
}

func listBuiltinsVerbose(specs map[string]*BuiltinSpec, byModule, byEffect bool) {
    // Group as requested
    // For each builtin, print:
    //   - Formatted signature from Type()
    //   - Description from Metadata
    //   - Examples from Metadata
}
```

## Success Criteria

1. `ailang builtins list --verbose` shows signatures and descriptions
2. `ailang builtins show _rand_int` shows full documentation
3. `ailang builtins list --format json` outputs machine-readable metadata
4. Users can discover function arity without trial-and-error

## Future Enhancements

Once this works:
- Auto-generate effect API sections for `ailang prompt`
- Make `ailang iface` work for virtual stdlib modules
- Add `:doc <func>` command to REPL

## Notes

- The hard work (metadata system) is already done
- This is ~100 lines of code in doctor.go
- High impact: Solves the "module gotcha" frustration
