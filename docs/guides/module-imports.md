# Module Imports and Explicit Dependencies

AILANG requires **explicit imports** for every module you use. This document explains why and how to avoid common import errors.

## Key Principle: Imports Are NOT Transitive

When module A imports module B, A does **not** automatically inherit B's imports.

This means:
- You must explicitly import every module you directly use
- Removing an import from module B doesn't break module A
- Module dependencies are visible and explicit

### Why Not Transitive Imports?

**Hidden Dependencies Problem:**
```
module/auth imports std/fs, std/net
module/api imports module/auth
module/api gets std/fs and std/net implicitly
(but doesn't know it uses them!)
```

When module/auth changes its imports, module/api breaks without warning.

**AILANG Solution - Explicit Imports:**
```
module/auth imports std/fs, std/net
module/api imports module/auth, std/fs, std/net
(All dependencies are visible)
```

When module/auth changes, module/api is unaffected (has explicit imports).

## Import Rules

### Rule 1: Import Everything You Use

If your module calls a function from a library, import it:

```ailang
-- ❌ WRONG - Using fs.fileExists but not importing std/fs
module myapp
import std/option (Some, None)

func checkConfig() -> bool ! {FS} {
  fileExists("config.json")  -- ERROR: std/fs not imported
}

-- ✅ CORRECT - Explicit import
module myapp
import std/fs (fileExists)
import std/option (Some, None)

func checkConfig() -> bool ! {FS} {
  fileExists("config.json")  -- Works!
}
```

### Rule 2: Import Everything Your Imported Modules Use

This is the "gotcha" that surprises developers:

```ailang
-- Module: services/gcp_auth
module services/gcp_auth
import std/fs (readFile, fileExists)  -- Uses file I/O

export func readCredentials() -> Result[string, string] ! {FS} {
  readFile("creds.json")
}

-- Module: main
module main
import services/gcp_auth (readCredentials)

-- ❌ WRONG - Transitive import doesn't work
func main() -> () ! {FS} {
  match readCredentials() {
    Ok(creds) => println(creds),
    Err(e) => println(e)
  }
}
-- ERROR: failed to resolve global std/fs.fileExists

-- ✅ CORRECT - Add explicit import
module main
import std/fs (readFile, fileExists)
import services/gcp_auth (readCredentials)

func main() -> () ! {FS} {
  match readCredentials() {
    Ok(creds) => println(creds),
    Err(e) => println(e)
  }
}
```

**Why?** Even though `main` doesn't directly call `readFile`, the runtime needs the type and effect information for `std/fs` to be available.

### Rule 3: Import Patterns

AILANG supports multiple import patterns:

#### Full Module Import (Qualified Access)

```ailang
import std/list

-- Access functions with module prefix
length(std/list.map(addOne, [1,2,3]))
```

#### Selective Import (Unqualified Access)

```ailang
import std/list (map, filter, length)

-- Access functions directly
length(map(addOne, [1,2,3]))
```

#### Aliased Imports (v0.4.8+)

```ailang
-- Module alias
import std/list as List

-- Access via alias
List.length(xs)
List.map(f, xs)

-- Symbol alias
import std/list (length as listLength)
listLength(xs)

-- Combined
import std/list as List (map, filter)
-- Direct access: map(f, xs), filter(p, xs)
-- Qualified access: List.length(xs)
```

## Common Import Errors

### Error 1: "module std/X not imported"

**Error Message:**
```
failed to resolve global std/fs.fileExists: module std/fs not imported by module_name
```

**Cause:** You're using a standard library module without importing it.

**Fix:**
```ailang
-- Add the import
import std/fs (fileExists, readFile)
```

### Error 2: Forgetting Transitive Dependencies

**Problem:**
```ailang
module myapp
import services/auth (authenticate)  -- Uses std/json internally

-- ❌ Type checking passes, runtime fails
func main() -> () ! {IO, FS} {
  authenticate("user")
}
```

**Solution:** Import all dependencies transitively:
```ailang
module myapp
import std/json (decode)  -- Explicitly import transitive deps
import services/auth (authenticate)

func main() -> () ! {IO, FS} {
  authenticate("user")
}
```

### Error 3: Symbol Name Collisions

When importing from multiple modules with same function names:

**Problem:**
```ailang
import std/string (length)  -- string.length
import std/list (length)     -- list.length (ERROR: conflicting import)

-- Both modules export 'length', conflict!
```

**Solution: Use Import Aliases (v0.4.8+)**
```ailang
import std/string as Str (length)
import std/list as List (length)

-- Now disambiguated
let strLen = Str.length("hello")
let listLen = List.length([1,2,3])
```

**Alternative: Qualified Access**
```ailang
import std/string
import std/list

-- Access with module prefix
let strLen = std/string.length("hello")
let listLen = std/list.length([1,2,3])
```

## Real-World Example: BigQuery Connector

Here's how the BigQuery connector correctly handles imports:

```ailang
-- Module: services/gcp_auth
module services/gcp_auth

-- Explicitly import everything we use
import std/fs (readFile, fileExists)
import std/net (httpRequest)
import std/json (decode, encode)
import std/string (length, substring)
import std/option (getOrElse, isSome)
import std/result (Ok, Err)

export func getAccessToken() -> Result[string, string] ! {FS, Net} {
  -- Implementation uses all imported modules
  ...
}

-- Module: services/bigquery
module services/bigquery

-- Import library module
import services/gcp_auth (getAccessToken)

-- IMPORTANT: Also import what gcp_auth uses
import std/net (httpRequest)
import std/json (encode, decode)
import std/result (Ok, Err)

export func query(projectId: string, sql: string, token: string)
  -> Result[QueryResult, string] ! {Net}
{
  -- Implementation
  ...
}

-- Module: main
module main

-- Import both service modules
import services/gcp_auth (getAccessToken, getDefaultProject)
import services/bigquery (query)

-- Also import transitive dependencies
import std/result (Ok, Err)
import std/option (getOrElse, isSome)

export func main() -> () ! {IO, FS, Net} {
  -- All imports are explicit
  -- Easy to see what this module depends on
  ...
}
```

**Checking imports:** Run `ailang check --show-dependencies main.ail` to see all imports (future feature).

## Import Organization Best Practices

### 1. Organize by Layer

```ailang
module myapp

-- Standard library first
import std/fs (readFile, writeFile)
import std/json (encode, decode)
import std/option (Some, None, getOrElse)
import std/result (Ok, Err)
import std/string (split, join)

-- Internal modules
import lib/database (query, insert)
import lib/auth (authenticate)

-- This module's implementations
func main() -> () ! {IO, FS} { ... }
```

### 2. Alphabetical Within Groups

```ailang
import std/fs
import std/json
import std/list
import std/option  -- Alphabetical
import std/result
import std/string
```

### 3. Be Explicit, Not Clever

```ailang
-- ❌ AVOID: Importing more than needed
import std/fs (readFile, writeFile, appendFile, deleteFile, etc...)

-- ✅ GOOD: Import only what you use
import std/fs (readFile, writeFile)
```

### 4. Document Non-Obvious Dependencies

```ailang
module myapp

import std/fs (readFile)
import std/json (decode)  -- Used by parseConfig, called from main

export func main() -> () ! {FS} {
  match parseConfig() {
    Ok(cfg) => run(cfg),
    Err(e) => println(e)
  }
}

func parseConfig() -> Result[Config, string] ! {FS} {
  let content = readFile("config.json")
  match decode(content) {
    Ok(json) => extractConfig(json),
    Err(e) => Err("Invalid JSON: " ++ e)
  }
}
```

## Debugging Import Issues

### Step 1: Check the Error Message

```bash
$ ailang check mymodule.ail
Error: failed to resolve global std/fs.readFile: module std/fs not imported by mymodule
```

### Step 2: Identify Missing Module

The error tells you which module isn't imported. Add it:

```ailang
import std/fs (readFile)  -- Add this
```

### Step 3: Ensure All Transitive Dependencies

If a function you're calling uses other libraries, import those too:

```bash
$ ailang check myapp.ail
Error: failed to resolve global std/json.decode: module std/json not imported by myapp

# myapp imports services/auth
# services/auth uses std/json
# → myapp must also import std/json
```

### Step 4: Check for Typos

```ailang
import std/list       -- ✅ Correct
import std/lists      -- ❌ Wrong (no such module)
import std/List       -- ❌ Wrong (lowercase required)
```

## Performance Impact

**Good news:** Import overhead is minimal.

- Transitive imports: Only analyzed once at compile time
- Effect checking uses import information (no runtime cost)
- Type checking uses import information (no runtime cost)

**No runtime penalty for being explicit.**

## Related Documentation

- [Module System Guide](modules.md) - Module declarations and organization
- [Reserved Keywords](../reference/reserved-keywords.md) - `import`, `export`, `as`
- [Effect System](effects.md) - Effect requirements (`! {IO, FS, Net}`)
- [Standard Library](../../stdlib/) - Available modules to import from

## Quick Reference: Common Imports

```ailang
-- I/O and System
import std/io (println, print)
import std/fs (readFile, writeFile, fileExists)
import std/time (now, sleep)

-- Collections
import std/list (map, filter, fold, length)
import std/string (split, join, length, substring)

-- Data
import std/json (encode, decode, getString, getNumber)
import std/option (Some, None, getOrElse, isSome)
import std/result (Ok, Err)

-- Networking
import std/net (httpRequest, listen, accept)

-- Full module access
import std/prelude  -- Everything
```

Run `ailang docs std/X` to see available functions in module X.
