# M-CSP-SESSION-TYPES: CSP Concurrency with Session Types

**Status**: Planned
**Target**: v0.8.0
**Priority**: P2 (Medium) - Enables Axiom A6 compliance
**Estimated**: 6-8 weeks (significant feature)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | CSP message order is deterministic |
| A2: Replayability | +1 | Message traces are replayable |
| A3: Effect Legibility | +1 | Channel operations are explicit effects |
| A4: Explicit Authority | +1 | Channels require explicit capabilities |
| A5: Bounded Verification | +1 | Session types enable local protocol verification |
| A6: Safe Concurrency | +1 | **Primary goal** - session types prevent races |
| A7: Machines First | +1 | Protocols machine-verifiable |
| A8: Minimal Syntax | -1 | New syntax for channels and protocols |
| A9: Cost Visibility | +1 | Channel operations have visible cost |
| A10: Composability | +1 | Session types compose via delegation |
| A11: Structured Failure | +1 | Protocol violations are typed errors |
| A12: System Boundary | +1 | Channels are explicit boundaries |

**Net Score: +10** -> **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): CSP semantics are deterministic (message order preserved)
- [x] A3 (Effects): Channel ops declared as effects (! Chan)
- [x] A4 (Authority): Channels require capability
- [x] A7 (Machines First): Protocol verification is automated

## Problem Statement

AILANG's Axiom A6 (Safe Concurrency) states: "Concurrency must not destroy meaning. Shared mutable state is a bug factory."

**Current State:**
- Sequential execution only (deterministic by default)
- No parallel execution model
- No message passing primitives
- **Axiom A6 score: 1/2 (partial)**

**Impact:**
- Cannot express parallel algorithms
- Cannot model concurrent agent communication
- Limited to sequential AI task execution

## Goals

**Primary Goal:** Add CSP-style channels with session types for safe concurrent communication.

**Success Metrics:**
- Channel creation with typed protocols
- Send/receive operations with session type checking
- Protocol violations caught at compile time
- Effect typing for channel operations (! Chan)
- **Axiom A6 score improved to 2/2 (strong)**

## Solution Design

### Overview

Implement Communicating Sequential Processes (CSP) with session types. Session types statically verify that communication protocols are followed correctly, preventing deadlocks and protocol violations.

### Architecture

**Components:**
1. **Session Type System**: Type-level protocol descriptions
2. **Channel Runtime**: Go-backed channel implementation
3. **Protocol Checker**: Verify send/receive sequences at compile time
4. **Scheduler**: Deterministic task scheduling

### Session Type Syntax

```ailang
-- Protocol definition
protocol Counter =
  | Inc -> Counter       -- receive Inc, continue as Counter
  | Get -> !int -> end   -- receive Get, send int, terminate

-- Dual (client-side)
protocol CounterClient =
  | !Inc -> CounterClient
  | !Get -> int -> end
```

### Channel Operations

```ailang
-- Create channel pair (returns both endpoints)
let (server, client) = newChan[Counter]()

-- Server side
let serve = \ch. match recv(ch) with
  | Inc -> serve(ch)              -- loop
  | Get -> send(ch, count); close(ch)

-- Client side
let use = \ch.
  send(ch, Inc);
  send(ch, Inc);
  send(ch, Get);
  let n = recv(ch) in
  close(ch);
  n
```

### Effect Integration

Channel operations are effects:

```ailang
-- Type signature
newChan : unit -> (Chan[S], Chan[dual(S)]) ! Chan
send : Chan[!T -> S] -> T -> Chan[S] ! Chan
recv : Chan[T -> S] -> (T, Chan[S]) ! Chan
close : Chan[end] -> unit ! Chan
```

### Implementation Plan

**Phase 1: Session Type System** (~40 hours)
- [ ] Define session type AST
- [ ] Implement dual type computation
- [ ] Add session types to type checker
- [ ] Protocol subtyping rules

**Phase 2: Channel Runtime** (~30 hours)
- [ ] Create Go channel wrapper
- [ ] Implement typed send/receive
- [ ] Add channel registry for tracing
- [ ] Deterministic scheduling

**Phase 3: Protocol Checking** (~40 hours)
- [ ] Track session state through type checker
- [ ] Verify send/receive sequences
- [ ] Detect deadlock patterns
- [ ] Generate clear error messages

**Phase 4: Effect Integration** (~20 hours)
- [ ] Add `Chan` effect type
- [ ] Require `--caps Chan` at runtime
- [ ] Trace channel operations
- [ ] Documentation and examples

### Files to Modify/Create

**New files:**
- `internal/session/types.go` - Session type definitions (~300 LOC)
- `internal/session/dual.go` - Dual type computation (~150 LOC)
- `internal/session/check.go` - Protocol verification (~400 LOC)
- `internal/channels/channel.go` - Runtime implementation (~300 LOC)
- `internal/channels/scheduler.go` - Deterministic scheduler (~200 LOC)

**Modified files:**
- `internal/types/types.go` - Add SessionType (~50 LOC)
- `internal/types/typecheck.go` - Session type checking (~200 LOC)
- `internal/effects/effects.go` - Add Chan effect (~30 LOC)
- `internal/eval/eval.go` - Evaluate channel operations (~100 LOC)
- `internal/builtins/channels.go` - Channel builtins (~150 LOC)

## Examples

### Example 1: Simple Request-Response

```ailang
module examples/channels

protocol Echo =
  | string -> !string -> end

let echoServer = \ch. ! Chan
  let msg = recv(ch) in
  send(ch, "Echo: " ++ msg);
  close(ch)

let echoClient = \ch. ! Chan
  send(ch, "Hello");
  let response = recv(ch) in
  close(ch);
  response

let main = \(). ! IO, Chan
  let (server, client) = newChan[Echo]() in
  spawn(echoServer(server));
  let result = echoClient(client) in
  println(result)  -- "Echo: Hello"
```

### Example 2: Counter Protocol

```ailang
protocol Counter =
  | Inc -> Counter
  | Dec -> Counter
  | Get -> !int -> end

let counterServer = \ch. \count. ! Chan
  match recv(ch) with
  | Inc -> counterServer(ch, count + 1)
  | Dec -> counterServer(ch, count - 1)
  | Get -> send(ch, count); close(ch)

let counterClient = \ch. ! Chan
  send(ch, Inc);
  send(ch, Inc);
  send(ch, Inc);
  send(ch, Get);
  recv(ch)  -- Returns 3
```

### Example 3: Protocol Violation (Compile Error)

```ailang
-- This FAILS at compile time
let badClient = \ch. ! Chan
  send(ch, Inc);
  let n = recv(ch) in  -- ERROR: expected Inc/Dec/Get, got recv
  n
```

## Success Criteria

- [ ] Session types parse and type-check
- [ ] Dual types computed correctly
- [ ] Protocol violations caught at compile time
- [ ] Channel operations work at runtime
- [ ] `! Chan` effect enforced
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- Session type parsing
- Dual type computation
- Protocol subtyping

**Integration tests:**
- End-to-end channel communication
- Protocol verification
- Effect checking

**Manual testing:**
- Deadlock detection
- Performance under load
- Error message clarity

## Non-Goals

**Not in this feature:**
- Distributed channels (network) - Deferred
- Multiparty session types - Future work
- Channel priority/fairness - Out of scope
- Linear types (affine OK) - Separate feature

## Timeline

**Week 1-2** (40 hours):
- Phase 1: Session type system

**Week 3-4** (30 hours):
- Phase 2: Channel runtime

**Week 5-6** (40 hours):
- Phase 3: Protocol checking

**Week 7-8** (20 hours):
- Phase 4: Effect integration
- Documentation
- Examples

**Total: ~130 hours across 8 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Type system complexity | High | Start with binary sessions, defer multiparty |
| Deadlock detection | Medium | Conservative analysis, runtime detection as fallback |
| Performance overhead | Medium | Benchmark, optimize hot paths |
| Syntax complexity | Medium | Keep close to existing pattern match syntax |

## Related Documents

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A6: Safe Concurrency
- [Axiom Scorecard](docs/static/benchmarks/axiom_scorecard.json) - KPI tracking

**Academic References:**
- Kohei Honda et al. - "Session Types for Object-Oriented Languages"
- Simon Gay & Vasco Vasconcelos - "Linear Type Theory for Asynchronous Session Types"
- C.A.R. Hoare - "Communicating Sequential Processes"

## Future Work

- Multiparty session types (more than 2 participants)
- Distributed channels over network
- Channel delegation (passing channels through channels)
- Session type inference
- Visual protocol diagrams

---

**Document created**: 2025-12-19
**Last updated**: 2025-12-19
