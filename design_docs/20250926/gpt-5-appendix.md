## Appendix A — Type & Effect System (Formal Core)

A.1 Surface→Core Elaboration (overview)

Surface AIL desugars to a Core calculus with:
	•	ANF terms
	•	Explicit dictionaries for type classes
	•	Explicit effect handlers
	•	Linear channel endpoints
	•	? desugars to a match that returns from the current function

Surface AIL
  → desugar (? , pipelines, pattern sugar, tests)
  → Core AIL (ANF, explicit dict args, effect handlers, linear endpoints)
  → typed IR
  → interpreter / VM

A.2 Types, Effects, Rows
	•	Types: τ ::= α | int | float | bool | string | τ→τ ! ρ | [τ] | (τ,τ) | {l:τ | r} | ∀α. τ
	•	Effect rows: ρ ::= · | {E | r} where E is a finite set of effect labels (IO, FS, Net, Rand, Clock, Trace, DB, Async…), r a row variable.

Row Unification (principal)

Unify (ρ1, ρ2) returns substitution S or fail.
	1.	U(·, ·) = id.
	2.	U({E ∪ R1 | r1}, {E ∪ R2 | r2}) = U({R1 | r1}, {R2 | r2}). (Cancel common labels.)
	3.	U(r, ρ) where r not in fv(ρ) and r not occurs in ρ: bind r ↦ ρ.
	4.	Symmetric cases by commutation.
	5.	Otherwise fail.

Row Subsumption (optional—but recommended):
If Γ ⊢ e : τ ! ρ and ρ ⊆ ρ' then Γ ⊢ e : τ ! ρ'. (Monotone effect use.)

Implementation note: normalize rows to (sorted-set, residual-var) for canonical forms → principal types and cheap equality.

? Desugaring

Surface:

let x = e? in k

Core:

match e with
  Ok v  -> let x = v in k
  Err e -> return Err e

Typing rule:

Γ ⊢ e : Result τ ε ! ρ
Γ, x:τ ⊢ k : σ ! ρ
———————————————
Γ ⊢ (let x = e? in k) : σ ! ρ

A.3 Effect Handlers (elimination)

Handlers discharge rows:

handle  e  with H : τ ! (ρ \ δ)

where H provides implementations for each effect in δ.

Soundness: Effect Soundness Theorem
If ⊢ e : τ ! ρ and ρ = ·, evaluation performs no side effects.

⸻

Appendix B — Type Classes (Dictionary Elaboration)

B.1 Coherence & Instances
	•	Global coherence: at most one visible instance per (Class, Type) per build.
	•	No orphan instances: instances must be defined in the module of either the class or the type (or via a newtype wrapper).
	•	Overlaps: disallowed in v1.

B.2 Elaboration

Class method calls elaborate to dictionary application.

Surface:

sum : ∀a. Num a ⇒ [a] → a
sum xs = fold add (zero()) xs

Core (implicit args made explicit):

sum (Num_a : Dict Num a) (xs : [a]) =
  fold (Num_a.add) (Num_a.zero) xs

B.3 Defaulting Rules (REPL & literals)
	•	Unsatisfied Num a constraints at top level default to a := int.
	•	No defaulting for Show a, Decode a, etc. (fail if unsolved).
	•	Numeric literals: fromInteger : ∀a. Num a ⇒ Integer → a.

These rules are deterministic and must be documented by the LSP on hover.

⸻

Appendix C — Records & Row Polymorphism (Typing Rules)

Selection

Γ ⊢ e : { l : τ | r }     ————
———————————————  T-SEL
Γ ⊢ e.l : τ

Extension

Γ ⊢ e : { r }   Γ ⊢ v : τ
l ∉ labels(r)
———————————————  T-EXT
Γ ⊢ { e with l = v } : { l:τ | r }

Row polymorphic abstraction

getName : ∀r. { name:string | r } → string

Principal types guaranteed via row-unification (Appendix A.2).

⸻

Appendix D — Linear Resources & Capabilities

D.1 Qualifiers
	•	lin τ : linear (must be used exactly once along each path)
	•	Capabilities and channel endpoints are lin.

Linear send/recv residual typing
Session types p:

p ::= End | Send τ p | Recv τ p | Choice p p | SelectL p p | SelectR p p

Typing (sketch):

Γ ⊢ ch : lin Channel (Send τ p)
Γ ⊢ v  : τ
———————————————  T-SEND
Γ ⊢ send ch v : lin Channel p

Γ ⊢ ch : lin Channel (Recv τ p)
———————————————  T-RECV
Γ ⊢ recv ch : (τ, lin Channel p)

Close

Γ ⊢ ch : lin Channel End
———————————————  T-CLOSE
Γ ⊢ close ch : Unit

Aliasing a lin value is a type error; passing through branches requires both branches consume it to compatible residuals.

D.2 Borrowing (Read-only)

Introduce &cap FS as a non-linear read-only borrow; linear ownership returns at scope exit (compiler-enforced).

D.3 Finally

try {...} finally {...} is guaranteed to run on both Ok and Err paths; used to close linear FFI handles.

⸻

Appendix E — CSP Operational Semantics (Small-Step)

Configurations ⟨Threads; Mailboxes; Heap⟩.

Key rules (informal):
	•	Spawn: adds a thread; returns a Task a (non-linear) handle.
	•	Send/Recv: synchronized transfer on matching endpoints; step produces residual endpoints per session type.
	•	Select: if multiple guards ready, pick leftmost ready (documented fairness policy). Timeouts produce a typed branch (Timeout).
	•	Progress: well-typed programs with open tasks and non-terminated sessions can always step unless blocked on external effect.

Session Preservation: if ⊢ ch : Channel p and ch ↦ ch' then ⊢ ch' : Channel p' where p' is the residual from the rule.

⸻

Appendix F — Determinism Contract

Domain	Rule
RNG	All randomness via Rand capability; seeding required in tests.
Time	All time via Clock; virtual clocks available; sleeping is effect.
Maps	Deterministic iteration order (key-ordered) or explicitly undefined; choose one (recommend key-ordered).
Floats	IEEE-754 strict; disable FMA differences; document NaN propagation.
Net/DB	Nondeterministic by nature; tests use capability substitution/mocks.

Traces include seeds and virtual time anchor.

⸻

Appendix G — Typed Quasiquotes (Provenance & Validation)

G.1 Provenance

Every quasiquote carries (provider, version) captured at build:

sql"..." : SQL[Query T] @ (pg-schema, v=sha256:ABCD...)

Build fails if the provider version does not match the lockfile.

G.2 Expansion

Quasiquotes expand to builder ASTs; parameters become typed placeholders. Example:

Surface:

sql"""
  select id, name from users where age > ${min:int}
"""

Elaborates to:

Sql.query
  (Select
     (Cols [Id:int, Name:string])
     (From "users")
     (Where (Gt (Col "age") (Param min))))
  : SQL[Query {id:int, name:string}]

HTML quasiquotes expand to typed DOM constructors and sanitizer marks (SafeText, SafeHtml[Policy]).

Regex uses RE2-like engine; compile-time check rejects catastrophic patterns.

⸻

Appendix H — FFI Safety
	•	Foreign values wrapped in opaque linear types; no raw pointers leak to surface.
	•	Each FFI function declares its effect row explicitly.
	•	Only stdlib modules may mint capabilities; user code can only receive them via with Cap { ... } or arguments.

Soundness: no capability forging; purity is preserved by effect typing.

⸻

Appendix I — LSP & Tooling Contracts
	•	Effect Lens: show inferred effect row and required capabilities on hover for any symbol.
	•	Row/Dict explorer: view solved row variables and instance dictionaries at call sites.
	•	Quick-fixes:
	•	Add missing capability parameter and with block.
	•	Insert handler to discharge an effect.
	•	Add missing instance or wrap type with newtype for instance locality.
	•	Trace Viewer: stable JSON of spans (fn, inputs via Show, output via Show, effects, duration, mem) with source ranges.

⸻

Appendix J — Conformance Suite (must-pass cases)
	1.	Row principal types: polymorphic function uses effects only via arguments; check generalization and specialization.
	2.	Defaulting: 1 + 2 at REPL yields int without annotations.
	3.	Session misuse: swapped send/recv is a type error; missing close is a linearity error.
	4.	Quasiquote schema drift: change schema → compile fails with pinpointed column/codec mismatch.
	5.	Determinism: same seed + virtual time → identical trace hashes.
	6.	FFI: attempt to call foreign without declared effect → type error.

⸻
