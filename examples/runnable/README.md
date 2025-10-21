# Runnable Examples

Full AILANG programs that can be executed with `ailang run`. All files in this directory have proper module structure with `export func main()` entry points.

## How to Run

```bash
# Basic example
ailang run --caps IO examples/runnable/simple.ail --entry main

# Example with effects
ailang run --caps IO,FS examples/runnable/effects_basic.ail --entry main

# Example with Net capability
ailang run --caps IO,Net examples/runnable/micro_net_fetch.ail --entry main
```

## Categories

### ADTs and Pattern Matching
- `adt_simple.ail` - Basic algebraic data types
- `adt_option.ail` - Option type with Some/None
- `patterns.ail` - Pattern matching examples
- `guards_basic.ail` - Pattern guards

### Recursion
- `recursion_factorial.ail` - Factorial (direct and tail recursive)
- `recursion_fibonacci.ail` - Fibonacci sequence
- `recursion_mutual.ail` - Mutually recursive functions
- `recursion_quicksort.ail` - Quicksort implementation
- `letrec_recursion.ail` - Let-rec bindings

### Effects
- `effects_basic.ail` - Basic effect usage
- `effects_pure.ail` - Pure vs effectful functions
- `micro_io_echo.ail` - Simple I/O example

### Block Expressions
- `micro_block_if.ail` - If expressions in blocks
- `micro_block_seq.ail` - Sequential expressions
- `block_recursion.ail` - Blocks with recursion

### Data Structures
- `micro_record_person.ail` - Record examples
- `micro_option_map.ail` - Option mapping
- `json_basic_decode.ail` - JSON parsing

### Complete Programs
- `simple.ail` - Minimal working program
- `demos/hello_io.ail` - Hello world with I/O
- `demos/adt_pipeline.ail` - ADT pipeline example

## Testing

All examples in this directory are verified by CI:

```bash
make verify-examples  # Runs all runnable examples
```

Expected: 100% pass rate (all examples should run without errors)
