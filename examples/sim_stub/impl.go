// Sim Stub Implementation
// This file implements the extern functions declared in world.ail
//
// NOTE: This file is NOT generated - it's the user's implementation.
// The generated stubs in gen/game/extern_stubs.go show the signatures
// but this file provides the actual logic.

package main

import (
	"math/rand"

	"github.com/sunholo/ailang/examples/sim_stub/gen/game"
)

// InitWorld creates a new world with the given seed
// This is a deterministic function - same seed always produces same result
func InitWorld(seed int64) *game.World {
	// Initialize RNG with seed
	rng := rand.New(rand.NewSource(seed))

	return &game.World{
		Tick:  0,
		Value: rng.Int63n(100), // Start with random value 0-99
		Seed:  seed,
	}
}

// Step advances the simulation by one tick
// Returns the new world state and frame output
func Step(world *game.World, input *game.FrameInput) (*game.World, *game.FrameOutput) {
	// Recreate RNG from seed + tick for determinism
	rng := rand.New(rand.NewSource(world.Seed + world.Tick))

	// Calculate delta (-5 to +5)
	delta := rng.Int63n(11) - 5

	// Create new world state
	newWorld := &game.World{
		Tick:  world.Tick + 1,
		Value: world.Value + delta,
		Seed:  world.Seed,
	}

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
