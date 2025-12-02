// Sim Stub Implementation
// This file implements the extern functions declared in world.ail
//
// NOTE: This file is NOT generated - it's the user's implementation.
// The generated stubs in gen/game/extern_stubs.go show the signatures
// but this file provides the actual logic.
//
// DEBUG EFFECT EXAMPLE:
// This implementation demonstrates how to use the Debug effect:
// 1. Accept DebugContext as parameter (host-controlled)
// 2. Call Log() and Assert() during execution
// 3. Host collects debug output after each step

package main

import (
	"fmt"
	"math/rand"

	"github.com/sunholo/ailang/examples/sim_stub/gen/game"
)

// InitWorld creates a new world with the given seed
// This is a deterministic function - same seed always produces same result
func InitWorld(seed int64, debug *game.DebugContext) *game.World {
	// Initialize RNG with seed
	rng := rand.New(rand.NewSource(seed))

	value := rng.Int63n(100)

	// Debug: Log initialization
	debug.Log(fmt.Sprintf("Initializing world with seed=%d, initial_value=%d", seed, value), "impl.go:InitWorld")

	return &game.World{
		Tick:  0,
		Value: value, // Start with random value 0-99
		Seed:  seed,
	}
}

// Step advances the simulation by one tick
// Returns the new world state and frame output
// Debug context is used for logging and assertions during execution
func Step(world *game.World, input *game.FrameInput, debug *game.DebugContext) (*game.World, *game.FrameOutput) {
	// Recreate RNG from seed + tick for determinism
	rng := rand.New(rand.NewSource(world.Seed + world.Tick))

	// Calculate delta (-5 to +5)
	delta := rng.Int63n(11) - 5

	// Debug: Log the computation
	debug.Log(fmt.Sprintf("tick=%d: delta=%d", world.Tick+1, delta), "impl.go:Step")

	// Create new world state
	newWorld := &game.World{
		Tick:  world.Tick + 1,
		Value: world.Value + delta,
		Seed:  world.Seed,
	}

	// Debug: Assert invariants
	debug.Assert(newWorld.Tick > world.Tick, "tick should increase", "impl.go:Step")
	debug.Assert(newWorld.Seed == world.Seed, "seed should be preserved", "impl.go:Step")

	// Create output
	output := &game.FrameOutput{
		Message: formatTick(newWorld.Tick, newWorld.Value),
		Tick:    newWorld.Tick,
	}

	return newWorld, output
}

func formatTick(tick, value int64) string {
	return "Tick " + itoa(tick) + ": value=" + itoa(value)
}

// Simple int64 to string conversion (avoiding fmt dependency in hot path)
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return sign + string(digits)
}
