# Getting Started with AILANG

## Installation

### From GitHub Releases

Download pre-built binaries for your platform from the [latest release](https://github.com/sunholo-data/ailang/releases/latest):

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
```

### From Source

```bash
# Clone the repository
git clone https://github.com/sunholo/ailang.git
cd ailang

# Build and install
make install

# Verify installation
ailang --version
```

### Making ailang Accessible System-Wide

#### First-Time Setup
1. Install ailang to your Go bin directory:
   ```bash
   make install
   ```

2. Add Go bin to your PATH (if not already done):
   ```bash
   # For zsh (macOS default)
   echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   
   # For bash
   echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```

3. Test it works:
   ```bash
   ailang --version
   ```

#### Keeping ailang Up to Date

**Option 1: Manual Update**
```bash
make quick-install  # Fast reinstall
# OR
make install        # Full reinstall with version info
```

**Option 2: Auto-Update on File Changes**
```bash
make watch-install  # Automatically rebuilds and installs on file changes
```

**Option 3: Alias for Quick Updates**
```bash
# Add to ~/.zshrc or ~/.bashrc
alias ailang-update='cd /path/to/ailang && make quick-install && cd -'
```

## Quick Start

### Hello World

```ailang
-- hello.ail (✅ WORKS with current implementation)
print("Hello, AILANG!")
```

Run it:
```bash
ailang run hello.ail
```

### Working with Values

```ailang
-- values.ail
let name = "AILANG" in
let version = 0.0 in
print("Welcome to " ++ name ++ " v" ++ show(version))
```

### Lambda Expressions

```ailang
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
```ailang
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

## Working Examples

The following examples are confirmed to work with the current implementation:
- `examples/hello.ail` - Simple print statement
- `examples/simple.ail` - Basic arithmetic operations
- `examples/arithmetic.ail` - Arithmetic with show function
- `examples/lambda_expressions.ail` - Full lambda functionality
- `examples/test_basic.ail` - Basic test cases
- `examples/type_inference_basic.ail` - Type inference examples

## Next Steps

- Learn the [language syntax](../reference/language-syntax.md)
- Explore [REPL commands](../reference/repl-commands.md)
- Check [implementation status](../reference/implementation-status.md)
- Read the [development guide](./development.md)