Amazing spec. v4.0 already nails most of what makes an AI-first language actually usable by… well, AIs. Below are focused upgrades to make one-shot programs simpler and security tighter, plus a few “AI ergonomics” niceties that would make this my daily driver.

What I’d Add (Priority-ordered)

1) One-Shot Mode (Hermetic, declarative, auto-signed)

Goal: run a single .ail file as a safe, reproducible “function app” with zero scaffolding.
	•	@oneshot entrypoint: single function + declarative I/O contract.
	•	Inputs/outputs declared as data contracts, with schema & refinements.
	•	Hermetic bundle: compiler vendors dependencies, locks versions, embeds effect budgets & capability manifest, and signs the artifact.
	•	Attestation record: SBOM + hash of compiler, stdlib, decision ledger excerpt.
	•	Deterministic I/O harness: auto-provision mockable world (clock, rand, fs, net) unless explicitly bound.

@oneshot
func main(args: { file: Path, webhook: Url, ... }) 
  -> Result[{ summary: NonEmptyString, report: Bytes }, OneShotError]
  ! {FS with budget(reads: 10, bytes: 5.MB)
    , Net with timeout(5.s) with ratelimit(10.PerMinute)}
  @intent "Summarize a local file and POST result"
{
  let text = fs.readText(args.file)?
  let summary = summarize(text)?                 // may call LLM via std/ai
  net.post(args.webhook, json{ summary })?
  Ok({ summary, report: toPDF(summary) })
}

CLI:

$ ailang build --oneshot main.ail
# emits main.airun (signed), main.sbom.json, main.ledger.min.json
$ ailang run main.airun --file notes.txt --webhook https://...

2) Linear/Affine Types for Resources (No leaks by construction)

Goal: prevent “forgot to close” bugs and enforce one-time semantics.
	•	linear T: values must be consumed exactly once.
	•	Typestates for lifecycles: Socket[Closed] -> Socket[Open] -> Socket[Closed].
	•	Borrow scopes for read-only views without transfer.

type File : linear
func open(path: Path) -> Result[File[Open], IOError] ! {FS}
func readAll(f: &File[Open]) -> Result[Bytes, IOError] ! {FS}
func close(f: File[Open]) -> Result[Unit, IOError] ! {FS}

-- Compiler enforces close() is called exactly once.

3) Session Types for Protocols (Net safety)

Goal: encode request/response choreography and timeouts as types.

protocol Http1 = Send(Request) ; Recv(Response) ; End
func talk(conn: Channel[Http1, Net]) -> Result[Response] ! {Net}

Compose with effect operators:

! {Net with timeout(2.s) with retry(2, Exponential)}

4) Information-Flow & Taint Types (Privacy by default)

Goal: stop accidental data exfiltration and unsafe prompt stuffing.
	•	Labels: Public, Internal, Sensitive[Pii], Secret[Key].
	•	No implicit flows: compile-time checks; explicit downgrades require policy.
	•	LLM guards: callLLM refuses Sensitive unless redaction policy given.

type EmailBody : Sensitive[Pii]
func redactPii(x: Sensitive[Pii]) -> Public

func summarize(x: EmailBody) -> Result[Public, PolicyError] ! {LLM}
{
  callLLM(prompt"...", policy: { redactor: redactPii })
}

5) Policy DSL + Effect Handlers (Org guardrails)

Goal: enforce org rules (domains, endpoints, token ceilings) centrally.
	•	policy {} blocks compiled to effect handlers.
	•	Scoped by module, package, or oneshot artifact.

policy NetPolicy {
  allow domains ["api.mycorp.com", "stripe.com"]
  deny ipRanges ["10.0.0.0/8"]
  maxRequests 100 per Minute
}
use NetPolicy for Net

6) Proof-Carrying Refinements (SMT-assisted, bounded)

Goal: stronger correctness for tricky refinements without full theorem proving.
	•	prove where clauses call a bounded SMT solver (time/memory budgeted).
	•	Proof artifact pinned in the decision ledger; failures degrade to runtime checks if @prototype.

func bucketCount(n: PositiveInt) -> Int where prove (n % 2 == 0 => result > 0) {
  ...
}

7) First-Class Plans & Contracts (AI planning as values)

Goal: make high-level plans typed, checkable, and executable.

type Plan[a, e] = { intent: string, steps: [Step], effects: {e}, contract: Contract[a] }
func execute[a, e](p: Plan[a, e]) -> Result[a] ! {e}

let plan = plan"""
  intent: "ETL daily"
  steps:
    - read gs://bucket/input.csv
    - transform drop_nulls | normalize
    - write bq://dataset.table
""" : Plan[Unit, {FS, Net, DB}]
execute(plan)?

8) Deterministic Numerics & Units

Goal: reproducible math + safer physical code.
	•	Decimal and IEEE-deterministic modes.
	•	Units of measure: Length[m], Mass[kg], compile-time dimensional analysis.

func kineticEnergy(m: Mass[kg], v: Velocity[m/s]) -> Energy[J] { 0.5 * m * v * v }

9) Structured Concurrency (Nurseries)

Goal: simpler, safer async with budget propagation.

async nursery {   -- cancels children on failure; shares budgets
  let a = spawn fetch(urlA)
  let b = spawn fetch(urlB)
  await all [a, b]
}

Budgets auto-split and enforced per child; cancellations produce typed Cancelled errors.

10) Supply-Chain Security & Model Call Receipts

Goal: verify what actually ran and which model answered.
	•	SBOM embedded in artifact; Sigstore signing.
	•	LLM Receipts: model name/version, temperature, token usage, safety filters, hash of prompt/response (with label redactions).
	•	Replay guard: receipts bound to decision ledger & attested environment.

⸻

Smaller Ergonomic Wins
	•	with infer … default at function scope: func f() ! {infer with timeout(3.s)} {...}
	•	freeze blocks to lock known-good fragments (hash in ledger); compiler warns on drift.
	•	assume/guarantee for module boundaries to cut false positives.
	•	Improved FFI: foreign func requires declared effects & refinements; sandbox defaults.
	•	Schema evolution: shape types with migration rules (derive migrate v1→v2).
	•	Data provenance baked into Trace: every value can carry optional lineage.

⸻

Concrete Syntax Proposals

A) Capability Manifests (per module or oneshot)

manifest {
  effects: { FS(readOnly), Net, Clock }
  budgets: { Net.requests: 500/Hour, Net.bandwidth: 50.MB, Clock.wall: 2.Minutes }
  policies: [NetPolicy, LLMSafetyPolicy]
}

B) Linear/Typestate

type Tx : linear
func begin() -> Tx[Open] ! {DB}
func commit(tx: Tx[Open]) -> Unit ! {DB}
func rollback(tx: Tx[Open]) -> Unit ! {DB}

C) Session Types

protocol SMTP = Send(Helo) ; Recv(Ok) ; Send(MailFrom) ; Recv(Ok) ; ... ; End

D) Info-Flow Labels

let pwd : Secret[Key] = readSecret("db_password")?
-- compile-time error:
net.post(url, json{ pwd })  -- Secret → Net requires explicit downgrade

E) Ones-hot CLI embed

@oneshot
@cli "--file Path --webhook Url --since Date?"
func main(args: {...}) -> Result[...]


⸻

Tooling to Seal the Deal
	•	:lint-sec — static pass for info-flow leaks, missing redactions, unsafe FFI.
	•	:prove — runs bounded SMT for prove where clauses, emits proof blobs.
	•	:attest — creates signed attestation + SBOM + policy snapshot.
	•	LSP “AI Hints” — show missing coverage, suggested refinements, policy violations inline.
	•	Package manifests declare effects/policies; resolver refuses packages exceeding project caps.

⸻

How These Map to Your Goals
	•	Easier one-shot: @oneshot + manifests + hermetic bundles mean zero boilerplate and predictable runs.
	•	More secure: linear/affine + info-flow + policy DSL + receipts + session types keep code safe by construction.
	•	AI-friendly: first-class plans, receipts, annotations, examples, coverage hints, incremental checking—all accelerate my ability to propose, write, and verify code in tight loops.

⸻

Suggested Roadmap Deltas

Phase 1.5 (Q4 2024–Q1 2025)
	•	Minimal @oneshot runner (hermetic + budgets + attestation).
	•	Linear resources for FS/Net handles + basic typestates.
	•	Structured concurrency (nursery) atop existing Async.

Phase 2 (Q1 2025)
	•	Policy DSL + enforcement hooks (Net, LLM).
	•	Info-flow labels (compile-time only to start; redaction helpers in stdlib).
	•	Session types MVP for simple request/reply protocols.

Phase 3 (Q2 2025)
	•	Proof-carrying refinements with bounded SMT.
	•	Deterministic numerics & units.
	•	First-class Plans + :propose validator integration.

Phase 4 (Q3 2025)
	•	Receipts & supply-chain attestation (Sigstore).
	•	Full LSP hints and :lint-sec, :prove, :attest.

⸻

Love it. Here’s a tight, v0.1.0 MVP you can actually ship. It’s the smallest set that still proves AILANG’s “one-shot + secure by construction” thesis.

v0.1.0 MVP — Scope

1) Primary Goal

Run a single .ail file hermetically with explicit effects and resource budgets, producing a signed, reproducible artifact.

2) Non-Goals (defer)
	•	Gradual typing, session types, linear/affine types, proof-carrying refinements
	•	Example blocks as tests, full policy DSL, LSP “AI hints”, package manager

⸻

What’s In (Must-Have)

A. Language Core
	•	Expressions only (no statements), first-class functions, tuples, lists, records ({..., ...}), ADTs (type Result[a,e] = Ok(a)|Err(e)), pattern matching with basic exhaustiveness.
	•	Type inference with monomorphic let + parametric polymorphism on functions.
	•	Row-polymorphic records with the ... syntax: { name: string, ... }.

B. Effects (Minimal but explicit)
	•	Effects: { IO, FS, Net, Clock, Rand }.
	•	Functions annotate effects: a -> b ! {FS, Net}.
	•	Effect combinators (MVP): timeout(dur), retry(n, Exponential|Constant), trace(debug|info).
	•	Budgets (MVP): counts + simple bandwidth/time ceilings:
	•	Net with budget(requests: Int, bandwidth: Bytes)
	•	Clock with budget(wall_time: Duration)
	•	FS with budget(reads: Int, writes: Int, bytes: Bytes)
	•	Effect inference inside body (! {infer}) with required explicit export signature.

C. Refinements (Starter set)
	•	Built-ins: PositiveInt, NonZero, NonEmptyString, Percentage.
	•	Compile-time where obvious; else guarded at runtime with compiler-inserted checks.
	•	Ergonomic helpers in std/refinement: nonzero, positive, nonEmpty.

D. Determinism & Reproducibility
	•	Time virtualization (now() comes from Clock effect), seeded RNG via seed().
	•	Decision ledger (MVP): records compiler version, flags, inferred effects per function, budgets, and seed. JSON output.

E. One-Shot Runner (centerpiece)
	•	@oneshot on a single main(args: { ... }) -> Result[Out, Err] ! {…}.
	•	Hermetic bundle: build --oneshot emits:
	•	*.airun (artifact), *.sbom.json (deps + stdlib versions), *.ledger.json (decisions + budgets)
	•	Signed artifact using a built-in dev key (configurable later).
	•	CLI parsing from @cli spec (types: Path, Url, int, string, bool, optional with ?).
	•	Budgets enforced at runtime with clear, typed errors (Err(BudgetExceeded { kind, limit, used })).

F. Minimal Stdlib
	•	std/io (print, readText, writeText)
	•	std/net (httpGet, httpPost; JSON helpers)
	•	std/time (Duration, sleep)
	•	std/rand (seed, nextInt/Float)
	•	std/json (encode/decode with schema)
	•	std/refinement (helpers above)
	•	std/effects (retry, timeout, trace)

G. Tooling
	•	Compiler: ailang build file.ail (typecheck → emit IR → bundle)
	•	Runner: ailang run file.airun --flag ...
	•	Formatter: ailang fmt
	•	REPL (tiny): :type <expr>, :effects <fn>, :run main … (for @oneshot only)
	•	Test harness (MVP): test "name" { assert expr == value } + ailang test
	•	Security lint (MVP): ailang lint-sec flags unbounded Net/FS usage in exported funcs and @oneshot.

⸻

Acceptance Criteria
	1.	Hello One-Shot

	•	Program reads a file, computes simple stat, posts JSON to a webhook, writes a report; runs within Net.requests ≤ 3, FS.reads ≤ 2, Clock.wall ≤ 2s.
	•	Build produces signed .airun, SBOM, ledger; reruns deterministically with fixed seed.

	2.	Effect Discipline

	•	A function using readText without FS in export signature fails compile with a clear message + suggestion (add ! {FS}).

	3.	Budgets Enforced

	•	A loop that would exceed Net.requests: 5 is cut off with BudgetExceeded(Net.requests); exit code non-zero; ledger records actual usage.

	4.	Refinement Safety

	•	divide(_, 0) rejected at compile time if NonZero typed; or returns Err("Division by zero") when guards are inserted by compiler.

	5.	Timeout + Retry Work

	•	A flaky endpoint succeeds with retry(2, Exponential); a hanging endpoint yields Timeout within declared timeout(1.s).

	6.	Repro Trace

	•	Running the same .airun twice with same seed + inputs produces byte-identical outputs and identical decision ledger hashes.

⸻

Minimal Syntax Reference (v0.1.0)

-- Types
int, float, string, bool
[a], (a, b), { f: t, ... }, a -> b, a -> b ! {FS, Net}
type Option[a] = Some(a) | None

-- Functions
func name(x: T, y: U) -> R ! {Effects} { ... }

-- Pattern matching
match xs {
  [] => 0,
  [h, ...t] => 1 + length(t)
}

-- Refinements (starter set)
type PositiveInt = int where (x > 0)
type NonZero = int where (x != 0)
type NonEmptyString = string where length(x) > 0

-- One-shot
@oneshot
@cli "--in Path --out Path --webhook Url?"
func main(args: { in: Path, out: Path, webhook: Option[Url] })
  -> Result[{ summary: NonEmptyString }, string]
  ! {FS with budget(reads: 5, writes: 3, bytes: 5.MB),
     Net with timeout(3.s) with retry(2, Exponential),
     Clock with budget(wall_time: 3.s)}
{
  let txt = readText(args.in)?
  let summary = summarize(txt)?         -- user func, pure
  writeText(args.out, summary)?
  match args.webhook {
    Some(u) => httpPost(u, json{ summary })?,
    None => ()
  }
  Ok({ summary })
}


⸻

Cut List (if you must trim further)
	•	Drop trace(level) and keep a single trace() toggle.
	•	Defer Percentage refinement; keep NonZero, PositiveInt, NonEmptyString only.
	•	Drop std/json decode schemas; keep encode + simple dynamic decode.
	•	REPL can be build/run only; move :effects to compiler flag --effects.

⸻

Implementation Plan (Order of Work)
	1.	Core typechecker + effects (infer internally, require explicit on export)
	2.	Runtime budgets + errors (counters for Net/FS/Clock)
	3.	Net/FS minimal backends (httpGet/Post, readText/writeText)
	4.	Effect combinators timeout, retry, plus simple trace
	5.	Refinements (starter) with compile-time check + fallback runtime guards
	6.	One-shot bundler & runner (embed seed, SBOM, ledger; signing key support)
	7.	CLI parser from @cli (types: Path/Url/int/string/bool/Option)
	8.	Formatter + tiny test harness
	9.	Security lint (MVP) for unbounded effects in exports and @oneshot

⸻

Demo Programs to Ship with v0.1.0
	•	file_to_webhook.ail (shown above)
	•	safe_divide.ail (refinements + tests)
	•	budget_guard.ail (exceed requests → BudgetExceeded)
	•	retry_timeout.ail (flaky vs. hanging endpoints)
	•	pure_etl.ail (FS-only pipeline with strict byte/write budgets)

⸻

Go/No-Go Checklist
	•	All demo programs pass build, run, and produce stable ledger hashes
	•	Budgets fail fast with typed errors and human-readable messages
	•	Export effect mismatch yields actionable compiler diagnostics
	•	Formatter idempotent; test runner reports failures with line/col
	•	One-shot artifacts verify signature at run time
	•	Docs: “Why effects”, “Why budgets”, “Write your first oneshot”

⸻

If you want, I can turn this into a tracked issue list (labels: compiler, runtime, stdlib, tooling, docs) and draft the five demo programs + compiler error message copy so the first experience feels great.