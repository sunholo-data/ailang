# AILANG: The AI-First Programming Language

## Executive Summary

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It eliminates ambiguity through static typing with effect tracking, provides typed quasiquotes for safe string handling, uses CSP-based concurrency with session types, and automatically generates training data for AI model improvement. The complete implementation is approximately 5,000 lines of Go.

**Core Philosophy**: Make programs maximally machine-decidable and debuggable through explicit effects, typed staged metaprogramming, and deterministic execution traces.

**File Extension**: `.ail`

## Design Principles

### 1. **Explicit Effects via Algebraic Effect System**
- Pure functions by default
- Effects tracked in types using row polymorphism
- Capability-based effect permissions

### 2. **Everything is a Typed Expression**
- No statements or void returns
- Complete execution traces with typed values
- Errors as values via Result type

### 3. **Type-Safe Metaprogramming**
- Typed quasiquotes for SQL, HTML, regex, etc.
- Compile-time validation against schemas
- AST generation, not string concatenation

### 4. **Deterministic Execution**
- Explicit random seeds and time virtualization
- Reproducible traces for AI training
- Structured error context with typed bindings

### 5. **Single Concurrency Model (CSP)**
- Channels with session types
- No shared mutable state
- Message passing only

## Type System

### Core Types

```ailang
// Primitives (inferred, never declared explicitly)
int, float, string, bool

// Type variables
a, b, c

// Composite types
[a]                      // List
(a, b)                   // Tuple
a -> b                   // Function
a -> b ! {e}            // Function with effects

// Records with row polymorphism
{ label: type, ...r }    // Open record with row variable r

// Algebraic data types
type Result[a, e] = Ok(a) | Err(e)
type Option[a] = Some(a) | None
```

### Type Classes

```ailang
// Minimal set of type classes for ad-hoc polymorphism
class Eq[a] {
  func eq(x: a, y: a) -> bool
  func neq(x: a, y: a) -> bool = (x, y) => not(eq(x, y))
}

class Ord[a] : Eq {
  func lt(x: a, y: a) -> bool
  func lte(x: a, y: a) -> bool
  func gt(x: a, y: a) -> bool = (x, y) => lt(y, x)
  func gte(x: a, y: a) -> bool = (x, y) => lte(y, x)
}

class Num[a] {
  func add(x: a, y: a) -> a
  func sub(x: a, y: a) -> a
  func mul(x: a, y: a) -> a
  func div(x: a, y: a) -> Result[a, string]
  func zero() -> a
  func one() -> a
}

class Show[a] {
  func show(x: a) -> string
}

class Encode[a] {
  func encode(x: a) -> bytes
}

class Decode[a] {
  func decode(b: bytes) -> Result[a, string]
}
```

### Effect System with Row Polymorphism

```ailang
// Effect types
type Effect = 
  | IO           // Console I/O
  | FS           // File system
  | Net          // Network
  | DB           // Database
  | Rand         // Random generation
  | Clock        // Time operations
  | Trace        // Execution tracing

// Effect rows (polymorphic sets of effects)
type ! = { Effect* }

// Function types include effect annotations
pure func add(x: int, y: int) -> int                    // No effects
func readFile(path: string) -> string ! {FS}            // File system effect
func fetchUrl(url: string) -> string ! {Net}            // Network effect
func process(data: Data) -> Result[Output] ! {FS, Net}  // Multiple effects

// Effect inference and propagation
func main() -> Unit ! {IO, FS, Net} {
  let config = readFile("config.ail")?    // Propagates {FS}
  let data = fetchUrl(config.url)?        // Propagates {Net}
  print(process(data))                    // Propagates {IO}
}
```

### Records and Row Polymorphism

```ailang
// Basic records
type User = { 
  id: int, 
  name: string, 
  email: string 
}

// Row polymorphism for extensibility
func getName[r](user: { name: string | r }) -> string {
  user.name  // Works with any record containing 'name' field
}

// Record extension
let user = { id: 1, name: "Alice", email: "alice@example.com" }
let admin = { ...user, role: "admin", permissions: ["all"] }

// Pattern matching on records
match user {
  { name: "Alice", email: e, ...rest } => sendTo(e),
  { name: n, id: i } if i > 100 => handleSpecial(n),
  _ => defaultCase()
}
```

## Language Constructs

### 1. Function Definition

```ailang
// Pure function (no effects)
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (5, 120),
    (10, 3628800)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}

// Effectful function with explicit effects
func readConfig(path: string) -> Config ! {FS, Trace} {
  let content = readFile(path)?
  parseConfig(content)
}

// Generic function with type classes
func sum[a: Num](list: [a]) -> a {
  fold(list, zero[a](), add)
}
```

### 2. Pattern Matching

```ailang
// Exhaustive pattern matching enforced
match value {
  Ok(x) => process(x),
  Err(e) => handleError(e)
}

// Guard clauses
match list {
  [] => "empty",
  [x] if x > 0 => "single positive",
  [x, y] => "pair",
  [head, ...tail] => "multiple"
}

// Pattern extraction in strings
match input {
  pattern"${user}@${domain}.com" => Email{user, domain},
  _ => InvalidEmail
}
```

### 3. Typed Quasiquotes

```ailang
// SQL - produces typed AST, validates against schema
let query = sql"""
  SELECT id, name, email 
  FROM users 
  WHERE age > ${min_age: int} 
    AND country = ${country: string}
""" : SQL[Query[User]]

// HTML - typed DOM nodes with sanitization
let page = html"""
  <div class=${style: Style}>
    <h1>${title: SafeText}</h1>
    ${content: SafeHtml}
  </div>
""" : Html[Div]

// Regex - compile-time validation
let pattern = regex/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

// JSON - validated structure
let config = json{
  "name": ${app_name},
  "version": "1.0.0",
  "settings": {
    "debug": false,
    "port": ${port: int}
  }
} : Json[AppConfig]

// Shell - safe command construction
let cmd = shell"""
  grep ${pattern: string} ${file: Path} 
  | wc -l
""" : Shell[Command]

// URL - automatic encoding
let api = url"https://api.example.com/users?name=${userName}&age=${age}"
```

### 4. Capability-Based Effects

```ailang
// Import capabilities from modules
import std/io (FS, Net)
import std/random (Rand)
import std/time (Clock)

// Capabilities are required for effects
func processData(data: Data) -> Result[Output] ! {FS, Net} {
  with FS, Net {
    let config = readFile(FS, "config.ail")?
    let response = httpGet(Net, config.url)?
    writeFile(FS, "output.ail", response)?
  }
}

// Pure functions cannot perform effects
pure func calculate(x: int) -> int {
  // readFile(...) would be compile error - no FS capability
  x * 2
}

// Capability passing
func withLogging[a, e](cap: FS, f: () -> a ! e) -> a ! {FS | e} {
  let result = f()
  writeFile(cap, "log.txt", show(result))
  result
}
```

### 5. CSP Concurrency with Session Types

```ailang
// Session types define channel protocols
type Protocol = 
  | Send[Request, Recv[Response, End]]
  | Choice[BranchA, BranchB]

// Channels with session types
func client(ch: Channel[Protocol]) ! {Async} {
  send(ch, Request{data: "hello"})    // Type ensures correct order
  let response = recv(ch)?             // Must receive after send
  close(ch)                            // Must close after protocol
}

// Structured concurrency
func parallelMap[a, b](f: a -> b, items: [a]) -> [b] ! {Async} {
  let results = channel[b](len(items))
  
  parallel {
    for item in items {
      spawn { 
        results <- f(item) 
      }
    }
  }  // Waits for all spawned tasks
  
  [recv(results)? for _ in items]
}

// Select for multiple channels
func multiplex(ch1: Channel[a], ch2: Channel[b]) ! {Async} {
  select {
    x <- ch1 => handleA(x),
    y <- ch2 => handleB(y),
    timeout(5.seconds) => handleTimeout()
  }
}
```

### 6. Error Handling

```ailang
// Result type for fallible operations
func divide(a: float, b: float) -> Result[float, string] {
  if b == 0.0 then 
    Err("Division by zero")
  else 
    Ok(a / b)
}

// ? operator for error propagation
func calculate(x: float) -> Result[float, string] ! {Trace} {
  let a = divide(100.0, x)?    // Returns Err if divide fails
  let b = sqrt(a)?              // Returns Err if sqrt fails
  Ok(b * 2.0)
}

// Error type with structured context
type Error = {
  message: string,
  code: ErrorCode,
  context: [(string, exists a. (a, Show[a]))],  // Typed, showable bindings
  suggestions: [string],
  trace: [StackFrame],
  effects: [Effect]
}
```

### 7. Testing

```ailang
// Inline property tests
pure func reverse[a](list: [a]) -> [a]
  tests [
    ([], []),
    ([1], [1]),
    ([1,2,3], [3,2,1])
  ]
  properties [
    forall(xs: [a]) => reverse(reverse(xs)) == xs,
    forall(xs: [a]) => length(reverse(xs)) == length(xs)
  ]
{
  match list {
    [] => [],
    [x, ...xs] => reverse(xs) ++ [x]
  }
}

// Test blocks with assertions
test "user serialization" {
  let user = { id: 1, name: "Alice", email: "alice@example.com" }
  let json = encode(user)
  let decoded = decode[User](json)?
  assert decoded == user
}

// Property-based testing with shrinking
property "sort preserves length" {
  forall(list: [int]) =>
    length(sort(list)) == length(list)
}
```

## Standard Library

### Core Modules

```ailang
// std/prelude (auto-imported)
module prelude {
  export pure func id[a](x: a) -> a
  export pure func const[a, b](x: a) -> b -> a
  export pure func compose[a, b, c](f: b -> c, g: a -> b) -> a -> c
  export pure func flip[a, b, c](f: a -> b -> c) -> b -> a -> c
  
  // Re-export core type classes
  export class Eq[a] { ... }
  export class Ord[a] : Eq { ... }
  export class Num[a] { ... }
  export class Show[a] { ... }
}

// std/io - Capability-based I/O
module io {
  export capability FS {
    func readFile(path: Path) -> Result[string, IOError] ! {FS}
    func writeFile(path: Path, content: string) -> Result[(), IOError] ! {FS}
    func deleteFile(path: Path) -> Result[(), IOError] ! {FS}
  }
  
  export capability Net {
    func httpGet(url: Url) -> Result[Response, NetError] ! {Net}
    func httpPost(url: Url, body: Body) -> Result[Response, NetError] ! {Net}
  }
  
  export capability Console {
    func print(s: string) -> () ! {IO}
    func readLine() -> string ! {IO}
  }
}

// std/collections - Functional data structures
module collections {
  export pure func map[a, b](f: a -> b, list: [a]) -> [b]
  export pure func filter[a](pred: a -> bool, list: [a]) -> [a]
  export pure func fold[a, b](f: b -> a -> b, init: b, list: [a]) -> b
  export pure func zip[a, b](xs: [a], ys: [b]) -> [(a, b)]
  
  export type Tree[a] = Leaf(a) | Node(Tree[a], a, Tree[a])
  export type Queue[a]  // Efficient functional queue
  export type Set[a: Ord]
  export type Map[k: Ord, v]
}

// std/concurrent - CSP primitives
module concurrent {
  export type Channel[p]  // p is session type
  export func channel[p](size?: int) -> Channel[p] ! {Async}
  export func send[p](ch: Channel[Send[a, p]], value: a) -> Channel[p] ! {Async}
  export func recv[p](ch: Channel[Recv[a, p]]) -> (a, Channel[p]) ! {Async}
  export func spawn[a](f: () -> a ! e) -> Task[a] ! {Async}
  export func await[a](task: Task[a]) -> a ! {Async}
}
```

## Training Data Generation

### Automatic Execution Tracing

```ailang
// Every execution produces structured training data
type ExecutionTrace = {
  id: UUID,
  timestamp: ISO8601,
  code: string,
  ast: AST,
  typeEnvironment: TypeEnv,
  
  result: Result[Value, Error],
  effects: [Effect],
  
  trace: [{
    function: string,
    inputs: [(string, exists a. (a, Show[a]))],
    output: exists b. (b, Show[b]),
    duration: Duration,
    memory: Bytes
  }],
  
  // For training
  patterns: [Pattern],
  quality_score: float,  // 0.0 to 1.0
  corrections: Option[{
    original: AST,
    corrected: AST,
    explanation: string
  }]
}

// Export for model training
func exportTraining(traces: [ExecutionTrace]) -> TrainingDataset ! {FS} {
  traces
    |> filter(t => t.quality_score > 0.7)
    |> groupBy(t => t.patterns)
    |> map(formatForFineTuning)
    |> writeJsonLines("training_data.jsonl")
}
```

### Learning from Corrections

```ailang
// REPL tracks corrections for training
$ ailang repl --learn

>>> let f = (x) => x / 2
Error: Missing type annotation
Suggestion: let f = (x: float) -> float => x / 2.0

>>> let f = (x: float) -> float => x / 2.0
✓ Correction recorded for training

// Generates training example:
{
  "prompt": "Create a function that halves a number",
  "incorrect": "let f = (x) => x / 2",
  "error": "Missing type annotation",
  "correct": "let f = (x: float) -> float => x / 2.0",
  "pattern": "type_annotation_missing",
  "weight": 2.0  // User corrections weighted higher
}
```

## Deterministic Execution

```ailang
// All non-determinism is explicit and controlled
module random {
  export capability Rand {
    func random() -> float ! {Rand}
    func randomInt(min: int, max: int) -> int ! {Rand}
  }
  
  // Deterministic seeding for reproducibility
  export func withSeed[a](seed: int, f: Rand -> a ! e) -> a ! e {
    let rng = createRNG(seed)
    f(rng)
  }
}

module time {
  export capability Clock {
    func now() -> Timestamp ! {Clock}
    func sleep(d: Duration) -> () ! {Clock, Async}
  }
  
  // Virtual time for testing
  export func withVirtualTime[a](start: Timestamp, f: Clock -> a ! e) -> a ! e {
    let clock = VirtualClock(start)
    f(clock)
  }
}

// Example: Deterministic simulation
func simulate(seed: int) -> Result[SimData] ! {Trace} {
  withSeed(seed) { rng =>
    withVirtualTime(epoch) { clock =>
      runSimulation(rng, clock)
    }
  }
}
```

## Foreign Function Interface

```ailang
// FFI for Go interop
foreign "go" {
  func sqlite3_open(path: string) -> pointer
  func sqlite3_exec(db: pointer, sql: string) -> int
  func sqlite3_close(db: pointer) -> int
}

// Safe wrapper with capabilities
module database {
  export capability DB {
    func execute(query: SQL[a]) -> Result[[a], DBError] ! {DB}
  }
  
  export func withDB[a](path: Path, f: DB -> a ! e) -> a ! {FS | e} {
    let ptr = sqlite3_open(path.toString())
    try {
      let db = DB(ptr)
      f(db)
    } finally {
      sqlite3_close(ptr)
    }
  }
}
```

## Implementation Architecture

### Realistic Component Sizes

```
Core Language (~3,500 lines)
├── lexer.go         // 200 lines - Tokenization with position tracking
├── parser.go        // 500 lines - Recursive descent with Pratt parsing
├── types.go         // 800 lines - HM inference with effect rows
├── typeclass.go     // 400 lines - Type class resolution
├── effects.go       // 400 lines - Effect system and capabilities
├── eval.go          // 500 lines - Tree-walking interpreter
├── channels.go      // 400 lines - CSP implementation
├── session.go       // 300 lines - Session type checking

Quasiquotes (~800 lines)
├── quasiquote.go    // 200 lines - Base framework
├── sql.go           // 200 lines - SQL AST and validation
├── html.go          // 200 lines - HTML AST and sanitization
├── regex.go         // 100 lines - Regex compilation
└── json.go          // 100 lines - JSON validation

Standard Library (~1,000 lines)
├── prelude.go       // 200 lines - Core functions
├── collections.go   // 300 lines - Data structures
├── io.go            // 300 lines - I/O capabilities
└── concurrent.go    // 200 lines - CSP primitives

Tooling (~1,500 lines)
├── repl.go          // 300 lines - Interactive mode
├── test.go          // 300 lines - Test runner with property testing
├── module.go        // 400 lines - Module system
├── training.go      // 300 lines - Training data export
└── ffi.go           // 200 lines - Foreign function interface

Total: ~6,800 lines of Go for complete implementation
```

### Build and Execution

```bash
# Build the interpreter
$ go build -o ailang cmd/ailang/main.go

# Run a program
$ ailang run program.ail

# Interactive REPL with learning
$ ailang repl --learn
AILANG v1.0.0 - AI-First Functional Language
>>> let x = 42
x : int = 42
>>> 

# Watch mode with hot reload
$ ailang watch src/main.ail --trace
✓ Type checking... OK
✓ Effect checking... OK  
✓ Tests... 10/10 passed
⚡ Running with trace...

# Export training data
$ ailang export-training --since=2024-01-01 --min-quality=0.8
Exported 5,234 high-quality execution traces

# Run with deterministic seed (for debugging)
$ ailang run simulation.ail --seed=42 --virtual-time
Result is deterministic and reproducible
```

## Example Programs

### Hello World with Effects

```ailang
// hello.ail
import std/io (Console)

func main() -> () ! {IO} {
  with Console {
    print("Hello, AILANG!")
  }
}
```

### Type-Safe Web API

```ailang
// api.ail
import std/io (Net, Console)
import std/concurrent (spawn, channel)
import std/json

type Request = {
  method: Method,
  path: string,
  headers: Map[string, string],
  body: Option[Json]
}

type Response = {
  status: int,
  body: Json
}

func handleRequest(req: Request) -> Response ! {Net, DB} {
  match (req.method, req.path) {
    (GET, pattern"/users/${id: int}") => 
      getUser(id),
      
    (POST, "/users") => 
      match req.body {
        Some(json) => createUser(decode[User](json)?),
        None => Response{status: 400, body: json{"error": "Missing body"}}
      },
      
    _ => Response{status: 404, body: json{"error": "Not found"}}
  }
}

test "API routing" {
  let req = Request{
    method: GET,
    path: "/users/123",
    headers: empty,
    body: None
  }
  let resp = handleRequest(req)
  assert resp.status == 200
}
```

### Concurrent Data Processing with Session Types

```ailang
// pipeline.ail
import std/concurrent
import std/collections

// Define protocol for worker communication
type WorkerProtocol = 
  | Send[Task, Recv[Result, WorkerProtocol]]
  | End

func worker(id: int, ch: Channel[WorkerProtocol]) ! {Async, Trace} {
  match ch {
    Send[Task, rest] => {
      let (task, ch') = recv(ch)
      let result = process(task)
      let ch'' = send(ch', result)
      worker(id, ch'')  // Continue protocol
    },
    End => close(ch)
  }
}

func distributeWork(tasks: [Task]) -> [Result] ! {Async} {
  let numWorkers = 4
  let channels = [channel[WorkerProtocol]() for _ in range(numWorkers)]
  
  // Spawn workers
  parallel {
    for (i, ch) in enumerate(channels) {
      spawn { worker(i, ch) }
    }
  }
  
  // Distribute tasks round-robin
  for (i, task) in enumerate(tasks) {
    let ch = channels[i % numWorkers]
    send(ch, task)
  }
  
  // Collect results
  let results = []
  for ch in channels {
    while match ch {
      Recv[Result, rest] => {
        let (result, ch') = recv(ch)
        results.append(result)
        true
      },
      _ => false
    }
  }
  
  results
}
```

### Property-Based Testing Example

```ailang
// sort.ail
import std/test (property, forall, Gen)

pure func quicksort[a: Ord](list: [a]) -> [a] {
  match list {
    [] => [],
    [pivot, ...rest] => {
      let less = filter(x => x < pivot, rest)
      let greater = filter(x => x >= pivot, rest)
      quicksort(less) ++ [pivot] ++ quicksort(greater)
    }
  }
}

property "sort is idempotent" {
  forall(list: Gen.list(Gen.int)) =>
    quicksort(quicksort(list)) == quicksort(list)
}

property "sort preserves elements" {
  forall(list: Gen.list(Gen.int)) =>
    sort(count(list)) == count(quicksort(list))
}

property "sort orders elements" {
  forall(list: Gen.list(Gen.int)) => {
    let sorted = quicksort(list)
    all(i => sorted[i] <= sorted[i+1], range(0, length(sorted)-1))
  }
}
```

## Tooling Ecosystem

### Language Server Protocol (LSP)

```bash
# Start LSP server for IDE integration
$ ailang lsp
AILANG Language Server v1.0.0
Listening on stdio...

# Features:
- Type information on hover
- Effect inference display
- Go-to-definition
- Find references
- Rename refactoring
- Quick fixes for common errors
- Inline test execution
```

### Package Manager

```yaml
# ailang.yaml
name: my-project
version: 0.1.0
dependencies:
  http: "github.com/ailang/http@1.0.0"
  postgres: "github.com/ailang/postgres@2.1.0"
  
dev-dependencies:
  quickcheck: "github.com/ailang/quickcheck@1.0.0"

capabilities:
  - Net     # Required for http
  - DB      # Required for postgres
```

## Key Design Decisions

### Why Algebraic Effects over Monads?
- **Composability**: Effects compose automatically via row polymorphism
- **Inference**: Effect inference is more straightforward than monad transformer stacks
- **Readability**: Code looks like normal code, not lifted computations

### Why CSP over Actors/Async-Await?
- **Formal semantics**: CSP has decades of formal verification research
- **Session types**: Can statically verify protocol correctness
- **Go integration**: Natural fit with Go's channel implementation

### Why Typed Quasiquotes over Template Strings?
- **Injection prevention**: Impossible to construct injection attacks
- **Compile-time validation**: Catch errors before runtime
- **IDE support**: Full autocomplete and type checking in templates

### Why Capability-Based Security?
- **Explicit permissions**: Clear what code can do from type signature
- **Principle of least privilege**: Functions only get capabilities they need
- **Testability**: Easy to mock capabilities for testing

## Conclusion

AILANG provides a rigorous foundation for AI-assisted programming through:

1. **Algebraic effects with row polymorphism** - Making all effects explicit and composable
2. **Type classes** - Enabling ad-hoc polymorphism while maintaining type safety
3. **Typed quasiquotes** - Eliminating injection vulnerabilities at compile time
4. **CSP with session types** - Providing race-free concurrency by construction
5. **Deterministic execution** - Ensuring reproducible traces for AI training
6. **Capability-based security** - Making permissions explicit in types

The result is a language where AI can generate correct code with high confidence, humans can understand programs through types alone, and the system continuously improves through structured execution traces.

**AILANG: Where correctness is not optional, it's guaranteed.**

---

*Version 2.0 - Incorporating rigorous PL theory feedback*  
*Ready for implementation with realistic scope*