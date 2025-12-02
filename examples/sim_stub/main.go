// Sim Stub Example - Go Driver
//
// This demonstrates the AILANG -> Go code generation workflow:
// 1. world.ail defines types and extern function signatures
// 2. `ailang compile --emit-go` generates Go types and stubs
// 3. impl.go implements the extern functions
// 4. main.go runs the simulation
//
// Run with: go run .

package main

import (
	"fmt"

	"github.com/sunholo/ailang/examples/sim_stub/gen/game"
)

func main() {
	// Initialize world with fixed seed for determinism
	// Same seed (42) always produces the same output
	world := InitWorld(42)

	fmt.Println("Sim Stub Example")
	fmt.Println("================")
	fmt.Printf("Initial world: tick=%d, value=%d, seed=%d\n\n", world.Tick, world.Value, world.Seed)

	// Run 10 ticks
	for i := 0; i < 10; i++ {
		var output *game.FrameOutput
		world, output = Step(world, &game.FrameInput{})
		fmt.Println(output.Message)
	}

	fmt.Println("\nSimulation complete!")
}
