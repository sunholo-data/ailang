# Sim Stub Example

A minimal simulation example demonstrating the AILANG -> Go code generation workflow.

## Overview

This example shows:
1. **Type definitions** in AILANG (`world.ail`)
2. **Extern function declarations** for Go-implemented functions
3. **Generated Go types** from AILANG ADTs
4. **Deterministic simulation** with fixed seed

## Files

```
sim_stub/
├── world.ail          # AILANG types and extern function declarations
├── impl.go            # Go implementation of extern functions
├── main.go            # Go driver that runs the simulation
├── go.mod             # Go module file
├── Makefile           # Build automation
├── expected_output.txt # Golden file for CI testing
├── README.md          # This file
└── gen/               # Generated code (gitignored)
    └── game/
        ├── types.go       # Generated type definitions
        └── extern_stubs.go # Generated function stubs
```

## Usage

### Prerequisites

- Go 1.21 or later
- AILANG compiler (`ailang` in PATH)

### Build and Run

```bash
# Generate Go code, build, and run
make run

# Or step by step:
make generate  # Generate Go from AILANG
make build     # Build the Go binary
./sim_stub     # Run the simulation
```

### Test Determinism

```bash
make test
```

This runs the simulation and compares output to `expected_output.txt`.

## Workflow

1. **Define types in AILANG** (`world.ail`):
   ```ailang
   type World = { tick: int, value: int, seed: int }
   type FrameInput = { dummy: int }
   type FrameOutput = { message: string, tick: int }

   extern func init_world(seed: int) -> World
   extern func step(world: World, input: FrameInput) -> FrameOutput
   ```

2. **Generate Go code**:
   ```bash
   ailang compile --emit-go --package-name game world.ail
   ```

3. **Implement extern functions** (`impl.go`):
   - Uses generated types from `gen/game/types.go`
   - Implements `InitWorld()` and `Step()` functions

4. **Write the driver** (`main.go`):
   - Imports generated types
   - Calls the implemented functions

5. **Run and verify**:
   ```bash
   go run .
   ```

## Determinism

The simulation is fully deterministic:
- Same seed (42) always produces identical output
- RNG state is derived from seed + tick number
- No external dependencies that could vary

This makes it suitable for:
- Reproducible testing
- Replay systems
- Lockstep multiplayer

## Type Mapping

| AILANG Type | Go Type |
|-------------|---------|
| `int` | `int64` |
| `float` | `float64` |
| `string` | `string` |
| `bool` | `bool` |
| `{ field: T }` | `*StructName` |
| `[T]` | `[]T` |

See [docs/guides/go-interop.md](../../docs/docs/guides/go-interop.md) for complete documentation.
