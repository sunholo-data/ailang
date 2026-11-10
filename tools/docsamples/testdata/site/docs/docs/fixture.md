---
title: Doc-sample gate fixture
---

# Fixture page for tools/docsamples

Each block below is a control for one verdict; docsamples_test.go asserts it by the
marker comment on the block's first line.

```ailang
-- marker: pass-program
module fixture/hello

export func main() -> () ! {IO} = println("hello")
```

```ailang
-- marker: pass-header (no module line: the gate synthesises one)
export func double(x: int) -> int = x * 2
```

```ailang
-- marker: fail-untagged
export func broken() -> int = undefinedThing
```

```ailang parse
-- marker: fragment-parse (uses a name defined elsewhere; must only parse)
export func useHelper() -> int = helper(41)
```

```ailang error=PAR_UNEXPECTED_TOKEN
-- marker: error-fails-correctly
export func noLoops() -> int { loop { 1 } }
```

```ailang error=PAR017
-- marker: error-wrongly-passes
export func fine() -> int = 1
```

```ailang error=MOD014
-- marker: error-wrong-code
export func alsoBroken() -> int = stillUndefined
```

```ailang
-- marker: baselined-failure
export func listed() -> int = notDefinedAnywhere
```

```ailang
-- marker: stale-baseline (passes, but its key is in the baseline)
export func nowFixed() -> int = 42
```

```ailang sketch
-- marker: sketch
func signatureOnly(x: int) -> int
```

```ailang repl
λ> 1 + 2
3 :: Int
```

```ailang file=fixture/lib.ail
-- marker: file-lib
module fixture/lib

export func triple(x: int) -> int = x * 3
```

```ailang
-- marker: file-user (imports its file= sibling)
module fixture/user

import fixture/lib (triple)

export func nine() -> int = triple(3)
```

```typescript
-- marker: misfenced
export func hidden() -> int ! {IO} = 1
```

```ailang skecth
-- marker: bad-tag
export func typo() -> int = 1
```

```typescript
const notAilang: number = 1;
```

```bash
ailang check file.ail
```
