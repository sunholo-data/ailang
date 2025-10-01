📄 AILANG Design Doc: Self-Improving Programs via Tight Feedback Cycles

Author: AILANG Working Group
Date: October 2025
Version: Draft v0.1

⸻

1. Vision

AILANG should become the default language for self-improving software, where the feedback loop between proposing changes, evaluating them deterministically, and persisting improvements is first-class in the language runtime and type system.

Tagline: Programs that evolve faster and safer than in any other language.

⸻

2. Motivation

Traditional languages (Python, Go, Rust, etc.) can orchestrate self-improvement loops, but:
	•	They lack built-in safety: resource exhaustion, uncontrolled network calls, runaway costs, and non-deterministic execution are pervasive.
	•	They lack determinism: evaluation results vary due to hidden state, random seeds, or external data drift.
	•	They lack observability & provenance: improvements aren't checkpointed, reproducible, or auditable.
	•	They lack integration with AI APIs: model calls, token budgets, and retries are bolted on via third-party libs.

AILANG’s advantage: effects, budgets, and reproducibility are part of the language core. That makes it uniquely suited for self-improving systems, from local deterministic loops to distributed AI agents.

⸻

3. Design Goals

Primary
	1.	Tight Feedback Cycles
	•	Run: propose → evaluate → checkpoint → repeat.
	•	Low overhead, predictable, hermetic.
	2.	Safety by Construction
	•	Effects (! {AI, Net, FS, Rand, Clock, DB, Trace}) must be declared.
	•	Resource budgets must be explicit.
	•	Deterministic evaluation by default.
	•	NOTE: AI effect introduced in v0.2, joins 8 existing canonical effects.
	3.	Reproducibility & Provenance
	•	Content-addressed artifacts (SHA256 digests).
	•	Seeds, datasets, configs all pinned.
	•	Ledger/trace persisted automatically.
	4.	Composable Optimizers
	•	Hill climbing, bandits, evolutionary search as library functions.
	•	Evaluators are just user-written pure functions.

Secondary
	•	Human-in-the-loop gates (PRs, approvals, rollbacks).
	•	Support both local evaluation and remote AI model calls.
	•	Progressive disclosure: beginners can run small local loops, experts can orchestrate complex multi-agent experiments.

3.5 Anti-Goals

What AILANG explicitly does NOT do:
	1.	Auto-apply improvements - Always require human review/approval before production deployment
	2.	Unbounded optimization - Every loop must have explicit iteration/budget limits
	3.	Opaque AI calls - All model interactions logged, reproducible via seed/digest
	4.	Mutable checkpoints - Artifacts are content-addressed and immutable
	5.	Implicit resource usage - No silent network calls, file writes, or token consumption

Why: Self-improving systems must be safe by default, transparent always, and controllable explicitly.

⸻

4. Core Language Features to Enable Self-Improvement

	4.1	Effects + Budgets

	Budget syntax (row-polymorphic, composable):

-- v0.2: Row-polymorphic budgets
func callModel[e](p: Prompt) -> Result[Text, Error]
  ! {AI | e}
  with budget { tokens: 50_000, requests: 5, timeout: 3.s }

-- v0.3: Budget as capability (more testable)
type AIBudget = { tokens: Int, requests: Int, timeout: Duration }

func callModel(p: Prompt, budget: AIBudget) -> Result[Text, Error]
  ! {AI with budget}

	•	Compile-time check: functions must declare when they use AI/Net/FS/etc.
	•	Runtime check: budgets enforced automatically (fail with BudgetExceeded).
	•	Design: Budgets as capabilities (passed explicitly) are more testable and composable.

	4.2	Checkpoints / Artifact Store

	Built-in module std/checkpoint with deterministic digests:

module std/checkpoint

-- Save returns content-addressed digest (SHA256)
export func save[a](data: a) -> Digest ! {FS}

-- Load is pure given a digest (immutable store)
export pure func load(digest: Digest) -> Result[Blob, NotFound] ! {FS}

-- Garbage collection (explicit, not automatic)
export func prune(before: Timestamp, keep: [Digest]) -> () ! {FS}

-- Provenance: link digest to source code SID
export func tag(digest: Digest, meta: Metadata) -> () ! {FS}

	•	Everything self-improvement loops touch can be persisted/reproduced.
	•	Hash collisions: Use SHA256, astronomically unlikely; fail loudly on collision.
	•	GC strategy: Explicit prune() calls, no automatic cleanup.

	4.3	Ledger / Trace

	Append-only log of runs with queryable schema:

-- Ledger: append-only log of runs
type LedgerEntry = {
  runId: Digest,
  timestamp: Timestamp,
  seed: Int,
  effects: EffectRow,
  budgets: BudgetMap,
  artifacts: [Digest],
  result: Result[Digest, Error],
  trace: [TraceEvent]  -- for debugging
}

export func append(entry: LedgerEntry) -> () ! {FS}
export func query(predicate: LedgerEntry -> Bool) -> [LedgerEntry] ! {FS}

	•	Every run records effects, budgets, artifacts, seeds.
	•	Deterministic runs can be diffed or replayed.
	•	Enables "what changed?" analysis between runs.

	4.4	Optimizers & Evaluators as First-Class Citizens
	•	std/opt.hillClimb, std/opt.bandit, std/opt.evolutionary.
	•	std/eval.scoreDataset, std/eval.assertGolden.

	4.5	Deterministic Seeds
	•	Every run seeded at initialization (via CLI or config), recorded in ledger with full provenance.
	•	Guarantees reproducible results unless Rand or Clock effects are allowed.

⸻

5. Example Workflow

Prompt Optimization Loop (Enhanced with Error Handling & Checkpointing)

import std/opt (hillClimb, Config)
import std/eval (scoreDataset)
import std/checkpoint (save, load)
import std/io (println)

type Candidate = { prompt: String, digest: Option[Digest] }

-- Evaluator: pure, deterministic, cacheable
pure func evaluate(c: Candidate, ds: Dataset) -> Float {
  scoreDataset(ds, c.prompt)
}

-- Proposer: uses AI, handles errors, respects budget
func propose(c: Candidate) -> Result[Candidate, AIError]
  ! {AI, Net}
  with budget { tokens: 1000, requests: 1, timeout: 5.s }
{
  match callModel("improve this prompt", c.prompt) {
    Ok(suggestion) => Ok({ prompt: suggestion, digest: None }),
    Err(e) => Err(e)
  }
}

-- Pure mutator: always succeeds
pure func mutate(c: Candidate) -> [Candidate] {
  [ { prompt: c.prompt ++ "\nBe concise.", digest: None }
  , { prompt: c.prompt ++ "\nCite sources.", digest: None }
  ]
}

-- Main loop: orchestrates, checkpoints, handles failures
func main(ds: Dataset, seed: Candidate, iters: Int)
  -> Result[Candidate, Error] ! {AI, Net, FS}
  with budget { tokens: 50_000, requests: 20, wall: 60.s }
{
  let config = Config {
    maxIters: iters,
    patience: 5,  -- early stopping
    minimize: false
  };

  hillClimb(
    seed,
    \c. evaluate(c, ds),
    \c. match propose(c) {
      Ok(new) => mutate(c) ++ [new],  -- mix AI + heuristics
      Err(_) => mutate(c)              -- fallback on AI failure
    },
    config
  ) |> save  -- checkpoint final result
}

Properties:
	•	Effects explicit: {AI, Net, FS}.
	•	Budgets enforced for tokens, requests, and wall time.
	•	Error handling: AI failures fall back to pure heuristics.
	•	Early stopping: patience parameter prevents overfitting.
	•	Checkpointing: Final result saved to immutable artifact store.
	•	Ledger records every candidate, score, and digest.
	•	Improvement loop is safe, reproducible, and resilient.

⸻

6. Phased Roadmap

v0.1 (Current MVP - October 2025)
	•	Pure deterministic evaluation loops (COMPLETE ✅)
	•	Module system for composable optimizers (COMPLETE ✅)
	•	Effect system foundation (COMPLETE ✅)
	•	Checkpoints as FS digests (TODO for v0.2)
	•	Golden test integration (TODO for v0.2)
	•	No AI/Net yet — use offline heuristics.

v0.2 (~2-3 weeks after v0.1.0 ships)
	- [ ] Add `AI` to CanonicalEffects (1 day)
	- [ ] Implement `std/checkpoint` with SHA256 digests (3 days)
	- [ ] Add budget syntax to effect parser (2 days)
	- [ ] Runtime budget enforcement (tokens, requests, wall time) (4 days)
	- [ ] `std/opt.hillClimb` with early stopping (3 days)
	- [ ] Ledger append/query API (2 days)
	- [ ] Example: prompt optimizer with GPT-4 (1 day)
	- [ ] Integration tests: budget exhaustion, checkpoint recovery (2 days)

	Success Criteria:
	- ✅ Can run 100-iteration hillClimb loop with AI effect
	- ✅ Budget exceeded → graceful failure with partial results
	- ✅ Checkpoints are byte-for-byte reproducible
	- ✅ Ledger queryable for "which run had best score?"

v0.3
	•	Bandits & evolutionary optimizers.
	•	Auto-PR + rollback modules.
	•	Canary runs.
	•	Basic human-in-loop review hooks.
	•	Effect handlers (intercept AI calls for caching, mocking, rate limiting).

v0.4
	•	Service mode (ailang serve) → schedule improvement jobs.
	•	Signed artifacts, provenance receipts.
	•	Multi-agent negotiation of interfaces.
	•	Row polymorphism for "this optimizer works with ANY effects".

⸻

7. Applications
	•	Prompt tuning & fine-tuning search (LLM ops).
	•	Auto-ETL optimization: improving data cleaning scripts deterministically.
	•	Config optimizers: hyperparams, feature flags.
	•	Agent skill learning: chaining AI API calls with resource guards.
	•	Self-upgrading libraries: stdlib functions propose & verify their own improvements.

⸻

7.5 Security & Safety

Self-improving systems pose unique risks. AILANG mitigates them:

1. Prompt Injection Attacks
   •	Mitigation: Sandboxed evaluators, input validation at type level
   •	Example: `Prompt` newtype that sanitizes on construction

2. Resource Exhaustion
   •	Mitigation: Budgets enforced at runtime, not advisory
   •	Example: Loop that hits token limit fails fast, doesn't retry

3. Data Poisoning
   •	Mitigation: Dataset digests pinned, immutable checkpoint store
   •	Example: Training on tampered data → digest mismatch → rejection

4. Unintended Optimization
   •	Mitigation: Pure evaluators, human review before production
   •	Example: Optimizer finds adversarial prompt → caught in review

5. Effect Leakage
   •	Mitigation: Type system prevents undeclared effects
   •	Example: "Pure" evaluator can't make network calls

Design Principle: Fail closed, not open. Budget exhaustion, missing artifacts, or type errors should halt execution, not continue with degraded behavior.

⸻

8. Open Questions
	•	Should artifact store be local-only (FS digests) in v0.1, or integrate with cloud (S3/GCS) early?
	•	How much of budget enforcement should be compile-time vs runtime?
	•	Should optimizers themselves declare effects (Rand, Clock)?
	•	What governance do we enforce around auto-PRs (thresholds, approvals)?

⸻

9. Success Criteria
	•	v0.1: run local optimization loops deterministically, reproducible by digest.
	•	v0.2: safe AI/Net calls with enforced budgets.
	•	v0.3: non-trivial programs (prompt optimizer, config tuner, ETL improver) run stably.
	•	v0.4: AILANG programs evolve themselves faster than any human-written workflow.

⸻

10. Competitive Positioning

Why AILANG is uniquely suited for self-improving software:

vs Python
	•	Type safety, determinism, budget enforcement at language level
	•	No silent failures, no hidden state, no runaway costs

vs Rust
	•	Higher-level abstractions for AI workflows
	•	Effect system makes resource control ergonomic, not burdensome

vs Haskell
	•	Simpler effect system (rows, not free monads)
	•	Better AI library ecosystem (future)
	•	Pragmatic defaults (determinism without ceremony)

vs LangChain/AutoGPT/CrewAI
	•	Provable safety properties via type system
	•	Reproducibility via content-addressed artifacts
	•	Budget enforcement prevents runaway costs
	•	Pure evaluators prevent non-deterministic drift

Tagline: "The language for self-improving software that's safe, reproducible, and provably correct."

No other language can credibly make this claim.

⸻

👉 This design doc deliberately keeps self-improvement as the north star. Every new feature (effects, budgets, checkpoints, optimizers) should be judged by: does it make the feedback cycle safer, faster, or more reproducible?

Implementation Status (Oct 2025): v0.1.0 complete with effect system, imports, stdlib foundation. v0.2 roadmap above provides concrete path to AI effect + budgets + checkpointing.
