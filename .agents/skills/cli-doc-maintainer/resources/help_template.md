# CLI Help Text Templates

Copy-paste templates for adding new content to `cmd/ailang/help.go`.

## Command Entry Templates

### Simple Command

```go
fmt.Printf("  %s           Brief one-line description\n", cyan("command"))
```

### Command with Arguments

```go
fmt.Printf("  %s <arg>        Description with required arg\n", cyan("command"))
fmt.Printf("  %s [opt]        Description with optional arg\n", cyan("command"))
fmt.Printf("  %s <req> [opt]  Description with both\n", cyan("command"))
```

### Command with Subcommands

```go
fmt.Printf("  %s <cmd>          Main description (N subcommands)\n", cyan("command"))
fmt.Printf("                          Use '%s' for full command list\n", yellow("ailang command --help"))
fmt.Printf("    %s       Subcommand description\n", cyan("command sub1"))
fmt.Printf("    %s       Subcommand description\n", cyan("command sub2"))
fmt.Printf("    %s       Subcommand description\n", cyan("command sub3"))
```

### Command with Flags

```go
fmt.Printf("  %s [flags] <file>       Description\n", cyan("command"))
fmt.Println("    --flag1          Flag description")
fmt.Println("    --flag2 <value>  Flag with value")
```

## Category Section Template

```go
fmt.Println("Category Name:")
fmt.Printf("  %s              Command 1 description\n", cyan("command1"))
fmt.Printf("  %s              Command 2 description\n", cyan("command2"))
fmt.Printf("  %s              Command 3 description\n", cyan("command3"))
fmt.Println()
```

## Environment Variable Templates

### Debug Flag

```go
fmt.Println("  DEBUG_COMPONENT=1            Purpose of this debug flag (use case)")
```

### Configuration Variable

```go
fmt.Println("  AILANG_FEATURE=<value>       Purpose and what values are valid")
```

### External Integration

```go
fmt.Println("  EXTERNAL_VAR=<value>         Integration purpose and where it's used")
```

## Example Section Templates

### Basic Example

```go
fmt.Printf("  %s                        # Brief comment\n", cyan("ailang command args"))
```

### Example with Flags

```go
fmt.Printf("  %s  # Longer comment explaining flags\n", cyan("ailang command --flag1 --flag2 value args"))
```

### Example with Environment Variable

```go
fmt.Printf("  %s                 # Purpose of this env var usage\n", cyan("DEBUG_STRICT=1 ailang command args"))
```

### Grouped Examples by Category

```go
fmt.Println("Category Name:")
fmt.Printf("  %s              # Example 1\n", cyan("ailang cmd1 args"))
fmt.Printf("  %s  # Example 2\n", cyan("ailang cmd2 --flag value"))
fmt.Printf("  %s                  # Example 3\n", cyan("ailang cmd3"))
fmt.Println()
```

## Full Command Addition Example

Adding a new command called "analyze" with subcommands:

```go
// In the appropriate category section (e.g., Development Tools):
fmt.Println("Development Tools:")
// ... existing commands ...
fmt.Printf("  %s <cmd>            Analyze AILANG code (3 subcommands)\n", cyan("analyze"))
fmt.Printf("    %s          Show complexity metrics\n", cyan("analyze complexity"))
fmt.Printf("    %s        Find code smells\n", cyan("analyze quality"))
fmt.Printf("    %s       Generate coverage report\n", cyan("analyze coverage"))
fmt.Println()

// In the Examples section:
fmt.Println("Code Analysis:")
fmt.Printf("  %s             # Analyze file complexity\n", cyan("ailang analyze complexity src/main.ail"))
fmt.Printf("  %s        # Quality check with JSON output\n", cyan("ailang analyze quality src/ --json"))
fmt.Println()
```

## Full Environment Variable Addition Example

Adding a new debug flag `DEBUG_ANALYZE`:

```go
// In Environment Variables > Debug Flags section:
fmt.Println("Debug Flags (for troubleshooting):")
// ... existing debug flags ...
fmt.Println("  DEBUG_ANALYZE=1              Trace code analysis decisions (complexity, smells)")
fmt.Println()

// In Examples > Debugging & Telemetry section:
fmt.Println("Debugging & Telemetry:")
// ... existing examples ...
fmt.Printf("  %s                 # Debug analysis engine\n", cyan("DEBUG_ANALYZE=1 ailang analyze complexity test.ail"))
fmt.Println()
```

## Full Category Addition Example

Adding a new category "Security & Auth":

```go
// After existing categories, before Run Command Flags:
fmt.Println("Security & Auth:")
fmt.Printf("  %s <cmd>           Manage access tokens\n", cyan("tokens"))
fmt.Printf("    %s         List active tokens\n", cyan("tokens list"))
fmt.Printf("    %s       Create new token\n", cyan("tokens create"))
fmt.Printf("    %s <id>    Revoke token\n", cyan("tokens revoke"))
fmt.Printf("  %s <cmd>           Manage API keys\n", cyan("keys"))
fmt.Printf("    %s           List API keys\n", cyan("keys list"))
fmt.Printf("    %s         Generate new key\n", cyan("keys generate"))
fmt.Println()

// In Examples section:
fmt.Println("Security & Auth:")
fmt.Printf("  %s                  # List all active tokens\n", cyan("ailang tokens list"))
fmt.Printf("  %s                # Generate new API key\n", cyan("ailang keys generate --name production"))
fmt.Println()
```

## Alignment Helper

Use this to ensure consistent spacing:

```go
// Column positions:
// 0         10        20        30        40        50        60
// |---------|---------|---------|---------|---------|---------|
fmt.Printf("  %s           Description starts here\n", cyan("cmd"))
//  ^^command^^         ^^description (starts ~col 20)^^

// For long commands, adjust spacing:
fmt.Printf("  %s <args>   Description\n", cyan("longer-command"))
```

## Testing Your Changes

After editing help.go:

```bash
# 1. Rebuild
make quick-install

# 2. View help
ailang --help

# 3. Check specific sections
ailang --help | grep -A 5 "Category Name"

# 4. Test examples (copy from help and run them)
ailang command --flag value

# 5. Verify formatting
# - Command names aligned
# - Descriptions start consistently
# - No lines over 100 chars
# - Blank lines between sections
```

## Common Patterns

### Command with Multiple Subcommand Groups

```go
fmt.Printf("  %s <cmd>          Description (10+ subcommands)\n", cyan("command"))
fmt.Printf("                          Use '%s' for full list\n", yellow("ailang command --help"))
fmt.Printf("    %s        Group 1: Item management\n", cyan("command add/remove/list"))
fmt.Printf("    %s      Group 2: Status operations\n", cyan("command status/check"))
fmt.Println()
```

### Alias Note

```go
fmt.Printf("  %s, %s               Description (aliases)\n", cyan("long"), cyan("short"))
```

### Deprecated Command Warning

```go
fmt.Printf("  %s           Description %s\n", cyan("command"), yellow("[DEPRECATED]"))
```

### Experimental Feature Notice

```go
fmt.Printf("  %s           Description %s\n", cyan("command"), blue("[EXPERIMENTAL]"))
```
