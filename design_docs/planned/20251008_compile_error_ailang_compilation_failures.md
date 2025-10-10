# AILANG: Compilation Failures

**Discovered**: AI Eval Analysis - 2025-10-08
**Frequency**: 36 failures across 5 benchmark(s)
**Priority**: P0 (Critical - Must Ship)
**Estimated**:  LOC, 
**Category**: compile_error
**Impact**: critical

## Problem Statement





**Last Updated**: 2025-10-25 (merged 12 new failures)


**Last Updated**: 2025-10-25 (merged 12 new failures)
## Evidence from AI Eval

**Affected Benchmarks**: adt_option, fizzbuzz, json_parse, pipeline, records_person

**Models Affected**: claude-sonnet-4-5, gpt-4o-mini

**Failure Rate**: 12/34 (35.3%)

### Example Failures


**Error 1:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:1:13: expected =, got { PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:1: unexpected token in expression: } PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:7:13: expected next token to be then, got { instead expected ; or }, got IDENT PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:8:12:...
```

**Generated Code:**
```ailang
type Option {
  Some(value)
  None
}

fn divide(a, b) {
  if b == 0 {
    return Option::None
  } else {
    return Option::Some(a / b)
  }
}

fn print_result(opt) {
  match opt {
    Option::Some(value) => print("Result: " + str(value))
    Option::None => print("Error: Division by zero")
  }
}

let result1 = divide(10, 2)
print_result(result1)

let result2 = divide(10, 0)
print_result(result2)
```

---

**Error 2:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [expected ; or }, got , PAR_NO_PREFIX_PARSE at benchmark/solution.ail:2:16: unexpected token in expression: , PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:1: unexpected token in expression: } PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:7:15: expected next token to be then, got { instead expected ; or }, got IDENT PAR_UNEXPECTED...
```

**Generated Code:**
```ailang
enum Option {
    Some(value),
    None
}

fn divide(a, b) {
    if b == 0 {
        return Option::None
    } else {
        return Option::Some(a / b)
    }
}

fn print_result(opt) {
    match opt {
        Option::Some(value) => print("Result: " + value),
        Option::None => print("Error: Division by zero")
    }
}

let result1 = divide(10, 2)
print_result(result1)

let result2 = divide(10, 0)
print_result(result2)
```

---

**Error 3:**
```
Error: type error in benchmark/solution (decl 0): defaulting failed for let binding $tmp1: ambiguous type variable α2 with classes [Ord]: ambiguous type requires annotation

```

**Generated Code:**
```ailang
module benchmark/solution

import std/io (println)

export func fizzbuzz(n: int, max: int) -> () ! {IO} {
  if n > max
  then ()
  else {
    let output = if n % 15 == 0
                 then "FizzBuzz"
                 else if n % 3 == 0
                 then "Fizz"
                 else if n % 5 == 0
                 then "Buzz"
                 else show(n);
    println(output);
    fizzbuzz(n + 1, max)
  }
}

export func main() -> () ! {IO} {
  fizzbuzz(1, 100)
}
```


### Additional Examples (Latest Analysis)

**Error 1:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:1:13: expected =, got { PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:1: unexpected token in expression: } PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:7:13: expected next token to be then, got { instead expected ; or }, got IDENT PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:8:12:...
```

**Generated Code:**
```ailang
type Option {
  Some(value)
  None
}

fn divide(a, b) {
  if b == 0 {
    return Option::None
  } else {
    return Option::Some(a / b)
  }
}

fn print_result(opt) {
  match opt {
    Option::Some(value) => print("Result: " + str(value))
    Option::None => print("Error: Division by zero")
  }
}

let result1 = divide(10, 2)
print_result(result1)

let result2 = divide(10, 0)
print_result(result2)
```

---

**Error 2:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [expected ; or }, got , PAR_NO_PREFIX_PARSE at benchmark/solution.ail:2:16: unexpected token in expression: , PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:1: unexpected token in expression: } PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:7:15: expected next token to be then, got { instead expected ; or }, got IDENT PAR_UNEXPECTED...
```

**Generated Code:**
```ailang
enum Option {
    Some(value),
    None
}

fn divide(a, b) {
    if b == 0 {
        return Option::None
    } else {
        return Option::Some(a / b)
    }
}

fn print_result(opt) {
    match opt {
        Option::Some(value) => print("Result: " + value),
        Option::None => print("Error: Division by zero")
    }
}

let result1 = divide(10, 2)
print_result(result1)

let result2 = divide(10, 0)
print_result(result2)
```

---


### Additional Examples (Latest Analysis)

**Error 1:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:1:13: expected =, got { PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:1: unexpected token in expression: } PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:7:13: expected next token to be then, got { instead expected ; or }, got IDENT PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:8:12:...
```

**Generated Code:**
```ailang
type Option {
  Some(value)
  None
}

fn divide(a, b) {
  if b == 0 {
    return Option::None
  } else {
    return Option::Some(a / b)
  }
}

fn print_result(opt) {
  match opt {
    Option::Some(value) => print("Result: " + str(value))
    Option::None => print("Error: Division by zero")
  }
}

let result1 = divide(10, 2)
print_result(result1)

let result2 = divide(10, 0)
print_result(result2)
```

---

**Error 2:**
```
Error: module loading error: failed to load benchmark/solution.ail (search trace: [Loading module: benchmark/solution.ail]): parse errors in benchmark/solution.ail: [expected ; or }, got , PAR_NO_PREFIX_PARSE at benchmark/solution.ail:2:16: unexpected token in expression: , PAR_NO_PREFIX_PARSE at benchmark/solution.ail:4:1: unexpected token in expression: } PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:7:15: expected next token to be then, got { instead expected ; or }, got IDENT PAR_UNEXPECTED...
```

**Generated Code:**
```ailang
enum Option {
    Some(value),
    None
}

fn divide(a, b) {
    if b == 0 {
        return Option::None
    } else {
        return Option::Some(a / b)
    }
}

fn print_result(opt) {
    match opt {
        Option::Some(value) => print("Result: " + value),
        Option::None => print("Error: Division by zero")
    }
}

let result1 = divide(10, 2)
print_result(result1)

let result2 = divide(10, 0)
print_result(result2)
```

---

---


## Root Cause Analysis



## Proposed Solution



### Implementation Approach



## Technical Design

### API Changes



### Type System Changes



### Runtime Changes



## Implementation Plan



## Testing Strategy

### Unit Tests



### Integration Tests



### New Benchmarks



## Success Criteria



## References

- **Similar Features**: See design_docs/implemented/ for reference implementations
- **Design Docs**: CLAUDE.md, README.md, design_docs/planned/v0_4_0_net_enhancements.md
- **AILANG Architecture**: See CLAUDE.md, README.md

## Estimated Impact

**Before Fix**:
- AI success rate: %
- Token efficiency: 

**After Fix** (projected):
- AI success rate: %
- Token efficiency: 

---

*Generated by ailang eval-analyze on 2025-10-08 14:59:16*
*Model: gpt5*
