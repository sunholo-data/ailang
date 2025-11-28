AILANG Game-Support Enablement

(Design Doc for AILANG Repo Only)

Purpose:
Define the language/compiler/runtime interfaces, effects, and evaluation hooks required so external projects (e.g., “PlanetWorld”) can use AILANG as a deterministic simulation + AI logic language, while a Go engine handles rendering, input, and native performance kernels.

This doc does not describe the game implementation itself.
This is purely what AILANG must provide.

⸻

1. High-Level Objective

Enable AILANG to function as:
	•	A state-orchestrating language for large simulations.
	•	A deterministic-ish engine with controlled randomness.
	•	An AI-controllable logic layer where agents use AI effects.
	•	A pure functional core interoperating cleanly with Go.
	•	A testable, benchmarkable system that emits machine-readable results.

Achieve this without expanding AILANG into graphics, windowing, or OS tasks.
Its only external dependencies are via:
	•	Interop → Go (generated code)
	•	Effects (implemented by Go)
	•	Extern functions (pure Go helpers)
	•	Eval harness (run outside AILANG)

⸻

2. Required AILANG Additions

2.1. Stable Interop Types

AILANG must be able to define ADTs that map deterministically to Go:

Requirements
	•	Algebraic data types (sum + product types).
	•	Lists, arrays, maps → Go slices/maps.
	•	Tagged union codegen for sum types:
	•	Either interface + structs, or a single struct with a discriminator + nested variant.
	•	Recursive types allowed (e.g., entities, UI trees).
	•	Generated Go code must be stable across compiles.

Deliverables
	•	Compiler module: ailang/gen/go that handles:
	•	Type lowering
	•	Naming scheme
	•	Go package output directory selection
	•	Update documentation for ADT ↔ Go mapping rules.

⸻

2.2. Exported Function ABI

AILANG must support defining top-level exported functions that Go can call:

Required semantics

export func init_world(seed: int) -> World
export func step(world: World, input: FrameInput)
    -> (World, FrameOutput) ! {RNG, AI, Debug}

Compiler must produce:

func InitWorld(seed int64) World
func Step(world World, input FrameInput) (World, FrameOutput, error)

Requirements
	•	Effects must propagate into an explicit error return.
	•	Pure functions must compile to pure Go functions.
	•	World and FrameOutput must be fully serialisable Go structs.

⸻

2.3. Effects Required

AILANG must implement 3 built-in effects with Go runtimes:

RNG Effect

effect RNG {
    rand_float() -> float
    rand_int(max: int) -> int
}

Go side:
	•	Provide a rand.Rand seeded per InitWorld.
	•	Allow deterministic or non-deterministic mode in the future.

⸻

AI Effect

Hook for decision-making:

type NPCContext = { ... }
type NPCAction = { ... }

effect AI {
    choose_action(ctx: NPCContext) -> NPCAction
}

Go runtime:
	•	Stub initially (deterministic placeholder).
	•	Must support eventually:
	•	Local model call
	•	Remote model call
	•	JSONL logging
	•	But interface stays stable.

⸻

Debug / Assert Effect

effect Debug {
    assert(cond: bool, msg: string) -> unit
    log(msg: string) -> unit
}

Go runtime:
	•	Collect logs and failed asserts.
	•	Returned as part of the error or appended to FrameOutput.debug.

⸻

3. Extern Function Support

Extern allows Go to implement pure high-performance helpers.

Example:

extern func find_path(world: World, from: Coord, to: Coord) -> Path

AILANG must support:
	•	Signature checking.
	•	Type compatibility between AILANG types and Go types.
	•	Generated stubs on Go side where the engine developer implements the logic.

This is critical for simulation-heavy workloads (pathfinding, influence maps).

⸻

4. Debug Build Mode

AILANG compiler should support a flag:

ailc --debug ...

Debug mode:
	•	Enables Debug.assert and Debug.log.
	•	Tracks allocations.
	•	Enables trace output.
	•	Allows invariant checks inside step.

Release mode:
	•	Removes/inlines debug overhead.
	•	Faster execution.

⸻

5. Evaluation Harness Integration Hooks

The eval harness (benchmarks + scenarios) lives in the game repo, not here.
But AILANG must expose:

5.1. Stable public API
	•	Must guarantee that:
	•	InitWorld
	•	Step
	•	ADTs in World, FrameInput, FrameOutput

remain stable across compiler/runtime refactors.

5.2. Deterministic ADT hashes (optional, future)

Provide a built-in hash(value) for ADTs → uint64, used by scenario trace comparison.

(This is optional v0.1, but good future-proofing.)

5.3. Compiler flags
	•	--emit-go
	•	--package-name <name>
	•	--out <dir>

For clean integration with Go modules.

⸻

6. Minimal AILANG Standard Library Additions

To support game-style sims, standard library should include:

6.1. Small math / geometry prims
	•	clamp, lerp, min, max
	•	vec2 type (optional)
	•	Fixed-point numeric type (future optional)

6.2. Utility modules
	•	std/random (thin wrapper around RNG effect)
	•	std/assert (wrapper around Debug.assert)
	•	std/time (not OS time, logical time helpers)

⸻

7. Testing AILANG for Game Use

AILANG repo must provide:

7.1. A “dummy sim” example

In examples/sim_stub/:
	•	Defines World, FrameInput, FrameOutput.
	•	Implements trivial init_world + step.
	•	Provides a tiny Go binder that prints a few ticks.

This is essential for ensuring stability across refactors.

7.2. CI job: Integration test

A CI test that:
	1.	Builds AILANG compiler.
	2.	Compiles the example sim into Go.
	3.	Runs a tiny evaluation loop.
	4.	Ensures ADT compatibility hasn’t broken.

⸻

8. Deliverables & Milestones

Milestone A — Interop Layer v0.1
	•	ADT → Go codegen stabilised.
	•	Exported functions callable.
	•	Minimal extern function support.

Milestone B — Effects v0.1
	•	RNG implemented.
	•	Debug implemented.
	•	AI stub implemented.

Milestone C — Compiler UX
	•	--emit-go, --out, --package-name implemented.
	•	Error messages improved for interop types.

Milestone D — Sim Example + Integration Test
	•	examples/sim_stub added.
	•	CI green end-to-end.

Milestone E — Evaluation-Ready
	•	ADTs stable.
	•	Step ABI frozen.
	•	Ready for real game repo to begin development.

This is the milestone at which you start the game repo.

⸻

9. Future Enhancements (Post-Game v0.1)
	•	Extern function auto-binding for fast matrix/array operations.
	•	Parallel execution of pure functions.
	•	Deterministic float or fixed-point type.
	•	Hot-reload of AILANG code inside the game.
	•	Built-in time-travel debugging for World.

⸻

Summary

This design document defines only what AILANG must guarantee for external game repos to build on it:
	•	Stable ADTs
	•	Exported function ABI
	•	Effects (RNG/AI/Debug)
	•	Extern hooks
	•	Compiler flags
	•	Integration tests

Everything else (rendering, scenarios, benchmarking, world design) lives in the game repo.
