---
sidebar_position: 4
title: Module System
description: Code organization, imports, and exports in AILANG
---

# Module System

AILANG uses a module system for code organization, with explicit imports and exports for clear dependencies.

## Module Declaration

Every AILANG file should declare its module path:

```typescript
module examples/math

-- Module contents follow...
```

The module path should match the file path relative to the project root:
- File: `examples/math.ail`
- Module: `module examples/math`

### Relaxed Module Matching

For quick prototyping, you can disable strict path matching:

```bash
# Flag
ailang run --relax-modules --caps IO --entry main temp.ail

# Environment variable
AILANG_RELAX_MODULES=1 ailang run --caps IO --entry main temp.ail
```

Files in temp directories (`/tmp/`, `/var/folders/`) auto-relax with a warning.

## Imports

### Import Syntax

AILANG has three import forms — each for a different scope:

```ailang
-- Standard library
import std/io (println, print)
import std/fs (readFile, writeFile)

-- External package (from dependencies in ailang.toml)
import pkg/sunholo/firestore/client (getDoc, setDoc)

-- Intra-package sibling (same package, resolved in module namespace)
import ./plan (Plan, lookupPlan)
import ./sub/helpers (validate)
```

**Rule of thumb**: `./` for local siblings, `pkg/` for external deps, `std/` for stdlib.

`./` resolves in module namespace: if current module is `a/b/c`, then `./d` means `a/b/d`. Interfaces and hashes always use canonical paths.

### Basic Imports

Import specific symbols from a module:

```typescript
import std/io (println, print)
import std/fs (readFile, writeFile)

func main() -> () ! {IO, FS} {
  println("Reading file...");
  let content = readFile("data.txt");
  println(content)
}
```

### Import Aliasing (v0.5.1+)

Rename modules on import to avoid conflicts:

```typescript
-- Module alias: qualified access
import std/list as List

func main() -> () ! {IO} {
  let xs = [1, 2, 3];
  println(show(List.length(xs)));     -- 3
  println(show(List.map(\x. x * 2, xs)))  -- [2, 4, 6]
}
```

### Symbol Aliasing (v0.5.1+)

Rename individual symbols on import:

```typescript
-- Symbol alias: direct access with new name
import std/list (length as listLength, map as listMap)

func main() -> () ! {IO} {
  let xs = [1, 2, 3];
  println(show(listLength(xs)));  -- 3
  println(show(listMap(\x. x * 2, xs)))  -- [2, 4, 6]
}
```

### Combined Aliasing

Use both module and symbol aliasing together:

```typescript
-- Module alias + specific symbols
import std/list as L (map, filter)

func main() -> () ! {IO} {
  let xs = [1, 2, 3, 4, 5];

  -- Direct access to map and filter
  let doubled = map(\x. x * 2, xs);
  let evens = filter(\x. x % 2 == 0, doubled);

  -- Qualified access to other functions
  println(show(L.length(evens)))  -- 2
}
```

### Resolving Name Conflicts

When two modules export the same name:

```typescript
-- Both modules have a 'parse' function
import json/parser as JsonParser
import xml/parser as XmlParser

func parseInput(format: string, data: string) -> Result[Data] {
  match format {
    "json" => JsonParser.parse(data),
    "xml" => XmlParser.parse(data),
    _ => Err("Unknown format")
  }
}
```

## Import Transitivity

**Imports are NOT transitive in AILANG.** When module A imports module B, A does **not** automatically get access to B's imports.

```typescript
-- module myapp/db imports std/fs internally
module myapp/db
import std/fs (readFile)
export func loadConfig() -> string ! {FS} = readFile("config.json")

-- module myapp/main - must explicitly import std/fs
module myapp/main
import std/fs (readFile)        -- Required! Not inherited from myapp/db
import myapp/db (loadConfig)

export func main() -> () ! {IO, FS} {
  let config = loadConfig();
  let extra = readFile("other.txt");
  println(config ++ extra)
}
```

**Why?** Explicit imports prevent hidden dependencies and ensure each module clearly declares what it uses. This is similar to Python's "explicit is better than implicit" philosophy.

**Common error:**
```
failed to resolve global std/fs.fileExists: module std/fs not imported
```

**Fix:** Add the missing import to your module. See the [Imports section](#imports) above for syntax examples.

## Exports

### Exporting Functions

Use `export` to make functions available to other modules:

```typescript
module math/utils

-- Exported: available to importers
export func add(x: int, y: int) -> int {
  x + y
}

-- Not exported: private to this module
func helper(x: int) -> int {
  x * 2
}

-- Can use private functions internally
export func double(x: int) -> int {
  helper(x)
}
```

### Exporting Types

Export type definitions:

```typescript
module data/option

-- Export type and constructors
export type Option[a] = Some(a) | None

-- Export functions that work with the type
export func map[a, b](f: a -> b, opt: Option[a]) -> Option[b] {
  match opt {
    Some(x) => Some(f(x)),
    None => None
  }
}
```

### Re-Exporting

Import and immediately re-export:

```typescript
module prelude

-- Re-export common utilities
export import std/io (println, print)
export import std/list (map, filter, fold)
export import std/option (Option, Some, None)
```

## Standard Library

AILANG includes a standard library with common utilities:

### std/io

Console I/O operations.

```typescript
import std/io (println, print, readLine)

func main() -> () ! {IO} {
  print("Enter name: ");
  let name = readLine();
  println("Hello, ${name}!")
}
```

### std/fs

File system operations.

```typescript
import std/fs (readFile, writeFile, exists)

func main() -> () ! {FS} {
  if exists("config.json") then
    let config = readFile("config.json");
    writeFile("config.backup.json", config)
  else
    writeFile("config.json", "{}")
}
```

### std/list

List operations.

```typescript
import std/list (map, filter, fold, length, head, tail)

func main() -> () ! {IO} {
  let numbers = [1, 2, 3, 4, 5];

  let doubled = map(\x. x * 2, numbers);
  let evens = filter(\x. x % 2 == 0, doubled);
  let sum = fold(\acc x. acc + x, 0, evens);

  println("Sum of doubled evens: ${show(sum)}")
}
```

### std/zip

ZIP archive reading operations (requires FS effect).

```typescript
import std/result (Result, Ok, Err)

-- List all entries in a ZIP archive
func listArchive(path: string) -> Result[List[string], string] ! {FS} {
  _zip_listEntries(path)
}

-- Read a text entry from a ZIP
func readText(path: string, entry: string) -> Result[string, string] ! {FS} {
  _zip_readEntry(path, entry)
}

-- Read a binary entry as base64
func readBinary(path: string, entry: string) -> Result[string, string] ! {FS} {
  _zip_readEntryBytes(path, entry)
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `_zip_listEntries` | `string -> Result[List[string], string] ! {FS}` | List all entry paths in a ZIP archive |
| `_zip_readEntry` | `(string, string) -> Result[string, string] ! {FS}` | Read a text entry (UTF-8) from a ZIP |
| `_zip_readEntryBytes` | `(string, string) -> Result[string, string] ! {FS}` | Read a binary entry as base64 string |

**Security:** Path traversal rejected, 10K entry limit, 100MB decompressed size limit.

### std/xml

XML parsing operations (pure functions, no effect required).

```typescript
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)

-- Parse XML string into XmlNode tree
let result = _xml_parse("<root><item>Hello</item></root>");

-- Query: find all elements by tag name
let items = _xml_findAll(node, "item");

-- Query: find first element by tag name
let first = _xml_findFirst(node, "item");

-- Extract text content, attributes, children, tag
let text = _xml_getText(node);
let attr = _xml_getAttr(node, "id");
let children = _xml_getChildren(node);
let tag = _xml_getTag(node);
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `_xml_parse` | `string -> Result[XmlNode, string]` | Parse XML string into XmlNode tree |
| `_xml_findAll` | `(XmlNode, string) -> List[XmlNode]` | Find all descendant elements by tag name |
| `_xml_findFirst` | `(XmlNode, string) -> Option[XmlNode]` | Find first descendant element by tag name |
| `_xml_getText` | `XmlNode -> string` | Extract text content from a node |
| `_xml_getAttr` | `(XmlNode, string) -> Option[string]` | Get attribute value by name |
| `_xml_getChildren` | `XmlNode -> List[XmlNode]` | Get child nodes of an element |
| `_xml_getTag` | `XmlNode -> string` | Get tag name (empty string for text/comment nodes) |

**XmlNode ADT:** `Element(tag, attrs, children) | Text(content) | CData(content) | Comment(content)`

### std/prelude

Common utilities automatically available:

- Type classes: `Num`, `Eq`, `Ord`, `Show`
- Operators: `+`, `-`, `*`, `/`, `==`, `<`, `>`, `++`
- Functions: `show`, `compare`

## Entry Points

### Main Function

Modules can define entry points:

```typescript
module examples/hello

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, World!")
}
```

Run with:
```bash
ailang run --caps IO --entry main examples/hello.ail
```

### Custom Entry Points

Any exported function can be an entry point:

```typescript
module examples/calc

export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

export func main() -> () ! {IO} {
  println(show(factorial(10)))
}
```

```bash
# Run main
ailang run --caps IO --entry main examples/calc.ail

# Run factorial directly (pure function, no caps needed)
ailang run --entry factorial examples/calc.ail
```

## Module Organization

### Recommended Structure

```
project/
├── src/
│   ├── main.ail           -- module src/main
│   ├── utils/
│   │   ├── math.ail       -- module src/utils/math
│   │   └── strings.ail    -- module src/utils/strings
│   └── data/
│       ├── types.ail      -- module src/data/types
│       └── json.ail       -- module src/data/json
├── tests/
│   └── test_math.ail      -- module tests/test_math
└── examples/
    └── demo.ail           -- module examples/demo
```

### Circular Dependencies

AILANG does not allow circular imports:

```typescript
-- module a
import b (funcB)  -- OK
export func funcA() = funcB()

-- module b
import a (funcA)  -- ERROR: Circular dependency!
export func funcB() = funcA()
```

**Solution:** Extract shared code to a third module.

## REPL vs File Modules

### REPL Mode

In the REPL, you can use imports interactively:

```
ailang> import std/list (map)
ailang> map(\x. x * 2, [1, 2, 3])
[2, 4, 6]
```

### File Mode

Files require module declarations:

```typescript
module scratch/test

import std/list (map)

export func main() -> () ! {IO} {
  println(show(map(\x. x * 2, [1, 2, 3])))
}
```

## Related Resources

- [Language Syntax](/docs/reference/language-syntax) - Complete syntax reference
- [Effect System](/docs/reference/effects) - Capabilities and effects
- [Getting Started](/docs/guides/getting-started) - Installation and first program
- [Module Execution](/docs/guides/module_execution) - Running modules
