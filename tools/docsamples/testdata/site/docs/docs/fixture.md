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
-- marker: pass-header (a library fragment: the gate synthesises its module line)
export func double(x: int) -> int = x * 2
```

```ailang
-- marker: main-needs-module (defines main, no module line: checked as written)
export func main() -> () ! {IO} = println("no module")
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

```ailang error=PAR_UNEXPECTED
-- marker: error-prefix-only (a prefix of the real code is not the code)
export func noLoops2() -> int { loop { 2 } }
```

```ailang error=PAR_UNEXPECTED_TOKEN
-- marker: error-second-code (EXT002 comes first; PAR_UNEXPECTED_TOKEN only second)
extern func identity[T](x: T) -> T
```

```ailang error=OTHER
-- marker: error-pseudo-code (OTHER is the gate's own class, not a code the binary prints)
export func pseudo() -> int = nope
```

```ailang
-- marker: baselined-failure
export func listed() -> int = notDefinedAnywhere
```

```ailang
-- marker: baselined-code-changed (listed as PAR015, now fails TC_UNDEFINED_VAR)
export func moved() -> int = goneAway
```

```ailang
-- marker: stale-baseline (passes, but its key is in the baseline)
export func nowFixed() -> int = 42
```

```ailang sketch
-- marker: sketch
func signatureOnly(x: int) -> int
```

```ailang sketch
-- marker: sketch-unlisted (tagged sketch, not in the unchecked list)
func anotherSignature(x: int) -> int
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

~~~ailang
-- marker: tilde-fence (a ~~~ fence is a fence)
export func tilde() -> int = undefinedUnderTilde
~~~

```typescript
-- marker: misfenced (does not parse: only the keyword hint can catch it)
export func hidden() -> int ! {IO} = 1)
```

```
-- marker: misfenced-untagged (does not parse: only the keyword hint can catch it)
export func alsoHidden() -> () ! {IO} = println("x"))
```

```text
-- marker: misfenced-parse-probe (no keyword hint, but it parses as AILANG)
let total = 1 + 2 in total
```

The next three carry no `--` marker on purpose (a `--` comment is itself a hint); the
test finds them by content.

```
let answer = 6 * 7 in answer
```

```typescript
match parse(input) {
  x => x,,
}
```

```typescript
let r = run(job)
  Ok(v) => v,,
```

```text
The output looks like this, which is not AILANG:
(15, 5)
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

```
-- marker: stray-fence (this unlabelled fence swallows the next ailang block)
```ailang
export func swallowed() -> int = neverChecked
```

```ailang
-- marker: unterminated (the last fence on the page never closes)
export func runsToEof() -> int = 1
