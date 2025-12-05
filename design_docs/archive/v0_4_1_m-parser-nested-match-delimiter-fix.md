# M-PARSER: Fix Nested Match Delimiter Tracking Bug

**Status:** Planned (v0.4.1)
**Type:** Bug Fix
**Priority:** Medium
**Effort:** ~4-8 hours

---

## Problem

Match expressions inside statement blocks have parsing issues due to incorrect nested delimiter tracking. This is a **parser bug**, not a language design limitation.

**Example of failure:**
```ailang
export func process(x: int) -> () ! {IO} {
  {
    let result = match x {
      0 => "zero",
      n => "other"
    };
    println(result)
  }
}
```

**Current behavior:** Parser loses track of delimiters when `match` appears inside a block (`{ ... }`), causing:
- "Unexpected token" errors
- Incorrect nesting level tracking
- Match arms parsed as block statements

**Root cause:** The parser's delimiter stack doesn't properly handle 3-level nesting:
1. Function body block `{ ... }`
2. Statement block `{ ... }`
3. Match expression `match x { ... }`

---

## Current Workaround

Until fixed, users can:

1. **Extract match to helper function:**
   ```ailang
   export func classify(x: int) -> string =
     match x {
       0 => "zero",
       n => "other"
     }

   export func process(x: int) -> () ! {IO} {
     let result = classify(x);
     println(result)
   }
   ```

2. **Use match as final expression in block:**
   ```ailang
   export func process(x: int) -> () ! {IO} {
     let result = {
       match x {
         0 => "zero",
         n => "other"
       }
     };
     println(result)
   }
   ```

---

## Proposed Fix

### Phase 1: Delimiter Stack Improvements (~2-4h)

**File:** `internal/parser/parser.go`

1. **Track delimiter context:**
   ```go
   type delimiterContext struct {
       kind  string  // "block", "match", "paren", "bracket"
       depth int
       start token.Position
   }

   type Parser struct {
       // ... existing fields ...
       delimStack []delimiterContext
   }
   ```

2. **Push/pop on delimiter boundaries:**
   ```go
   func (p *Parser) pushDelim(kind string) {
       p.delimStack = append(p.delimStack, delimiterContext{
           kind:  kind,
           depth: len(p.delimStack),
           start: p.curPos(),
       })
   }

   func (p *Parser) popDelim(kind string) error {
       if len(p.delimStack) == 0 {
           return fmt.Errorf("unmatched closing %s", kind)
       }
       ctx := p.delimStack[len(p.delimStack)-1]
       if ctx.kind != kind {
           return fmt.Errorf("expected closing %s, got %s", ctx.kind, kind)
       }
       p.delimStack = p.delimStack[:len(p.delimStack)-1]
       return nil
   }
   ```

3. **Update parser functions:**
   - `parseBlock()` - push/pop "block"
   - `parseMatchExpression()` - push/pop "match"
   - `parseCallArguments()` - push/pop "paren"
   - `parseListExpression()` - push/pop "bracket"

### Phase 2: Match-Specific Fixes (~2-4h)

**File:** `internal/parser/parser_expr.go` (or wherever match parsing lives)

1. **Track match nesting level:**
   ```go
   func (p *Parser) parseMatchExpression() ast.Expr {
       p.pushDelim("match")
       defer p.popDelim("match")

       match := &ast.Match{Pos: p.curPos()}

       // ... parse scrutinee ...

       p.expectPeek(lexer.LBRACE)  // match body
       p.pushDelim("match-body")

       // Parse arms with awareness of nesting
       for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
           arm := p.parseMatchArm()
           match.Arms = append(match.Arms, arm)

           // Commas are REQUIRED between arms
           if p.peekTokenIs(lexer.COMMA) {
               p.nextToken()
           }
       }

       p.expectPeek(lexer.RBRACE)
       p.popDelim("match-body")

       return match
   }
   ```

2. **Improve error messages:**
   ```go
   func (p *Parser) popDelim(kind string) error {
       // ... validation ...
       if err != nil {
           ctx := p.delimStack[len(p.delimStack)-1]
           return fmt.Errorf("%s: opened at %s but not closed properly",
               kind, ctx.start)
       }
       // ...
   }
   ```

### Phase 3: Testing (~1-2h)

**Test cases:**
```go
// internal/parser/parser_match_nesting_test.go

func TestMatchInBlock(t *testing.T) {
    input := `
    export func test() -> int {
        {
            let x = match 5 {
                0 => 10,
                n => n * 2
            };
            x + 1
        }
    }
    `
    // Should parse without errors
}

func TestMatchInFunctionBody(t *testing.T) {
    input := `
    export func classify(x: int) -> string ! {IO} {
        println("classifying");
        match x {
            0 => "zero",
            n => "other"
        }
    }
    `
    // Should parse without errors
}

func TestNestedMatchInMatch(t *testing.T) {
    input := `
    export func nested(x: Option[Option[int]]) -> int {
        match x {
            None => 0,
            Some(inner) => match inner {
                None => 1,
                Some(n) => n
            }
        }
    }
    `
    // Should parse without errors
}

func TestMatchWithBlockInArm(t *testing.T) {
    input := `
    export func complex(x: int) -> int ! {IO} {
        match x {
            0 => { println("zero"); 0 },
            n => { println("other"); n * 2 }
        }
    }
    `
    // Should parse without errors
}
```

---

## Success Criteria

- ✅ Match expressions parse correctly inside statement blocks
- ✅ 3+ level nesting works (function → block → match)
- ✅ Delimiter mismatch errors show clear messages with positions
- ✅ All existing tests still pass
- ✅ New test cases cover match-in-block scenarios

---

## Non-Goals

- **NOT fixing** double-brace ergonomics (`{{ ... }}`) - that's a separate issue
- **NOT adding** match-as-statement syntax - match is always an expression
- **NOT changing** match syntax itself - only fixing delimiter tracking

---

## Migration / Breaking Changes

**None.** This is a bug fix that makes currently-broken code work. No valid code breaks.

Users can remove workarounds (helper functions) once the fix lands.

---

## Related Issues

- User reported: "match inside blocks fails to parse"
- Root cause: Nested delimiter tracking in parser
- Workaround documented in: [prompts/v0.3.24.md](../../../prompts/v0.3.24.md) line 166

---

## Implementation Checklist

- [ ] **Phase 1:** Delimiter stack infrastructure (~2-4h)
  - [ ] Add `delimiterContext` struct
  - [ ] Implement `pushDelim()` / `popDelim()`
  - [ ] Update `parseBlock()` to track delimiters
  - [ ] Update `parseMatchExpression()` to track delimiters
- [ ] **Phase 2:** Match-specific fixes (~2-4h)
  - [ ] Fix match arm parsing with nesting awareness
  - [ ] Improve error messages for delimiter mismatches
  - [ ] Test manually with example code
- [ ] **Phase 3:** Tests and validation (~1-2h)
  - [ ] Write `parser_match_nesting_test.go`
  - [ ] Run `make test` - ensure all pass
  - [ ] Update v0.3.24 prompt to remove "known bug" warning
- [ ] **Phase 4:** Documentation (~0.5h)
  - [ ] Update CHANGELOG.md with fix
  - [ ] Remove workaround notes from teaching prompt
  - [ ] Move design doc to `implemented/v0_4_1/`

---

## Notes

This is a **bug fix**, not a feature. The language already supports match-in-blocks semantically; the parser just fails to handle it correctly.

Once fixed, users can write natural code without extracting helper functions or restructuring blocks.
