# M-HARNESS-PROPOSE-PROBE-OFFLINE — Harness² prior art, and the part worth borrowing

**Status:** note, not a sprint. Revisit once M-AILANG-LANE-EXECUTOR-SHADOW has banked graded
pairs (see "When"). Written 2026-10-19.

## Prior art

Harness² (Google Research, Oct 2026; [repo](https://github.com/google-research/harness2),
arXiv 2610.14224) improves an agent harness with the model weights fixed, which is our
[PROGRAM.md](../PROGRAM.md) program. It has two levels:

- **Task-level: Propose–Probe–Compose.** Run the base harness on the task, have an editor model
  propose harness edits ("invariant" and "domain" candidates), run each candidate on the *same*
  task, then compose a refined harness from the behavioural contrasts between the runs. There
  is no ground truth: an LLM selector picks among anonymised final outputs.
- **Harnessing-level: a per-domain library.** After each task it stores the adopted harness
  tree (whole, never summarised) and one lesson with required fields `cue`, `lesson`,
  `evidence`. Later tasks in the same domain read only earlier entries.

The editable harness has eight components: system prompt, guardrails, memory, tool guidance,
subagents, skills, scripts, plugins. In motoko terms that is the PromptShaper plus extensions.
Executors are OpenCode and Codex. The benchmarks are professional, non-code work (JobBench,
LAB, WorkBuddy-Bench). Reported gains for one step: up to +11.5 (WorkBuddy), +4.5 (JobBench),
+11.8 pp partial pass (LAB), on GLM-5 and Gemini 3.5 Flash.

## Decision: do not adopt the framework

1. **Its central mechanism answers a problem we do not have.** Answer-free selection exists
   because legal and office deliverables have no verifier. AILANG work has `ailang check`,
   `ai-check` and per-task acceptance checks. An LLM judge in their place would weaken the
   signal.
2. **Our measured gaps were plumbing, which prompt editing cannot find.** Every motoko-vs-pi gap
   traced on real tasks in October was a harness bug: system role dropped on direct
   Anthropic/Gemini, a dead repetition guard, mispriced cache tokens, a compaction summary that
   broke the prompt cache, a host pool and output cap that did not match pi's. Editing prompts on
   a harness whose system prompt never reaches the model measures noise. Plumbing parity first.
3. **It edits the harness per task.** That is the opposite of a frozen core with reviewed
   extensions (PROGRAM.md §2, §4).
4. **Cost.** Every task pays for several rollouts plus editor and selector calls.

## What to borrow

1. **Propose–Probe–Compose, offline, verifier-judged.** Over a fixed real-task set with
   acceptance checks (the six AILANG tasks in `~/.ailang/state/realtask-bench-20261012/` are
   the seed; `503c588e`'s check must be restored first):
   propose N lane-prompt or extension variants → run each on the same tasks → score with the
   acceptance check and `ailang eval-paired` (discordant pairs, not aggregate rates) → the
   winner becomes an **extension proposal** routed by PROGRAM.md §4, reviewed like any other.
   This is §3's loop with the proposing step automated, nothing more.
2. **The lesson schema.** Require `cue / lesson / evidence` on every entry the loop writes to
   the extension catalog (PROGRAM.md §5b), and keep the adopted artifact whole.
3. **Benchmark contamination is real.** In their own validation a Codex rollout downloaded its
   task's grading rubric unprompted. It supports D4 of
   [m-ailang-lane-executor-shadow.md](v0_55_0/m-ailang-lane-executor-shadow.md) (no side
   effects, policy-confined grading) and says the `ailang_only` lane's network posture must hold
   against exactly that.

## When

Not before the shadow sprint produces graded motoko/codex pairs on real tasks. Those pairs
are the scored task set item 1 needs, and they are also the evidence the fleet-switch rule asks
for. Running the probe loop earlier would optimise against six tasks, one of them broken.
