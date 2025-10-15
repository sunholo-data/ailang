# Getting Started with AILANG

## 🤖 For AI Agents: Quick Integration

**AILANG is designed for AI-assisted development.** To integrate AILANG into your AI coding agent:

### Step 1: Install AILANG

```bash
# macOS (Apple Silicon)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-arm64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Linux
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-linux-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Verify
ailang --version
```

### Step 2: Load the AILANG Teaching Prompt

**Provide your AI agent with the current AILANG syntax guide:**

📖 **[AILANG v0.3.8 Teaching Prompt](/docs/prompts/v0.3.8)**

This prompt teaches AI models:
- ✅ Correct AILANG syntax (not Python/Rust/JavaScript)
- ✅ Pure functional programming with recursion
- ✅ Module system with effects (IO, FS, Clock, Net)
- ✅ Record updates, pattern matching, ADTs
- ✅ Auto-imported std/prelude (no manual imports for comparisons)

**Copy the full prompt** from [prompts/v0.3.8.md](/docs/prompts/v0.3.8) and include it in your AI agent's system prompt or context.

### Step 3: Test AI Code Generation

Ask your AI agent to write AILANG code:

```
Using AILANG v0.3.8, write a program that reads a file and counts the number of lines.

[Include full v0.3.8 prompt here]
```

**Expected output:**
```typescript
module benchmark/solution

import std/io (println)
import std/fs (readFile)

export func countLines(content: string) -> int {
  -- Implementation using recursion
  ...
}

export func main() -> () ! {IO, FS} {
  let content = readFile("data.txt");
  println("Lines: " ++ show(countLines(content)))
}
```

### Step 4: Run AI-Generated Code

```bash
ailang run --caps IO,FS --entry main solution.ail
```

### AI Success Metrics (v0.3.8)

**Current benchmark results:**
- **AILANG**: 49.1% success rate (28/57 benchmarks)
- **Improvement**: +10.5% from v0.3.7 (38.6% → 49.1%)
- **Best model**: Claude Sonnet 4.5 (68.4% across all tasks)

See [AI Prompt Guide](/docs/guides/ai-prompt-guide) for detailed instructions.

---

## 👤 For Human Developers: Manual Installation

### Installation Options

#### From GitHub Releases (Recommended)

Download pre-built binaries from the [latest release](https://github.com/sunholo-data/ailang/releases/latest).

#### From Source (For Development)

```bash
git clone https://github.com/sunholo/ailang.git
cd ailang
make install
ailang --version
```

**Add Go bin to PATH:**
```bash
# For zsh (macOS default)
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**Development workflow:**
```bash
make quick-install    # Fast rebuild after changes
make test            # Run tests
make verify-examples # Test example files
```

## Quick Start

### Hello World (v0.3.8 Module Syntax)

```typescript
-- hello.ail
module examples/hello

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, AILANG!")
}
```

Run it:
```bash
ailang run --caps IO --entry main hello.ail
```

**Note**: Flags must come BEFORE the filename when using `ailang run`.

### Working with Values

```typescript
-- values.ail
let name = "AILANG" in
let version = 0.0 in
print("Welcome to " ++ name ++ " v" ++ show(version))
```

### Lambda Expressions

```typescript
-- Lambda syntax with closures
let add = \x y. x + y in
let add5 = add(5) in  -- Partial application
print("Result: " ++ show(add5(3)))  -- Result: 8

-- Higher-order functions
let compose = \f g x. f(g(x)) in
let double = \x. x * 2 in
let inc = \x. x + 1 in
let doubleThenInc = compose(inc)(double) in
print("Composed: " ++ show(doubleThenInc(5)))  -- Composed: 11
```

### Using the REPL

Start the interactive REPL:
```bash
ailang repl
```

Try some expressions:
```typescript
λ> 1 + 2
3 :: Int

λ> "Hello " ++ "World"
Hello World :: String

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> :quit
```

## Working Examples (v0.3.8)

The following examples are confirmed to work with the current implementation:

**Recursion**:
- `examples/recursion_factorial.ail` - Recursive factorial function
- `examples/recursion_fibonacci.ail` - Fibonacci sequence
- `examples/recursion_quicksort.ail` - Quicksort implementation
- `examples/recursion_mutual.ail` - Mutual recursion (isEven/isOdd)

**Records**:
- `examples/micro_record_person.ail` - Record literals and field access
- `examples/test_record_subsumption.ail` - Record subsumption

**Effects**:
- `examples/test_effect_io.ail` - IO effect examples
- `examples/test_effect_fs.ail` - File system operations
- `examples/micro_clock_measure.ail` - Clock effect (time, sleep)
- `examples/micro_net_fetch.ail` - Net effect (HTTP GET)

**Pattern Matching & ADTs**:
- `examples/adt_simple.ail` - Algebraic data types
- `examples/adt_option.ail` - Option type with pattern matching
- `examples/guards_basic.ail` - Pattern guards

**Blocks**:
- `examples/micro_block_if.ail` - Block expressions with if
- `examples/micro_block_seq.ail` - Sequential blocks
- `examples/block_recursion.ail` - Recursion in blocks

See [examples/STATUS.md](https://github.com/sunholo-data/ailang/blob/main/examples/STATUS.md) for the complete list of 48+ working examples.

## Next Steps

- Learn the [language syntax](../reference/language-syntax.md)
- Explore [REPL commands](../reference/repl-commands.md)
- Check [implementation status](../reference/implementation-status.md)
- Read the [development guide](./development.md)