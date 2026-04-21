// Sim Stub Example - Go Driver
//
// This demonstrates the AILANG -> Go code generation workflow:
// 1. world.ail defines types and extern function signatures
// 2. `ailang compile --emit-go` generates Go types and stubs
// 3. impl.go implements the extern functions
// 4. main.go runs the simulation
//
// DEBUG EFFECT EXAMPLE:
// This driver demonstrates the Debug effect host contract:
// 1. Create DebugContext before simulation
// 2. Pass to each step (host-controlled lifecycle)
// 3. Collect and display debug output
// 4. Reset between steps
//
// Run with: go run .

package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/examples/sim_stub/gen/game"
)

func main() {
	// Create Debug context (host-controlled)
	debugCtx := game.NewDebugContext()

	// Initialize world with fixed seed for determinism
	// Same seed (42) always produces the same output
	world := InitWorld(42, debugCtx)

	fmt.Println("Sim Stub Example (with Debug Effect)")
	fmt.Println("=====================================")
	fmt.Printf("Initial world: tick=%d, value=%d, seed=%d\n\n", world.Tick, world.Value, world.Seed)

	// Show debug output from initialization
	showDebugOutput(debugCtx, "init")

	// Run 10 ticks
	for i := 0; i < 10; i++ {
		// Reset debug context for this step
		debugCtx.Reset()
		debugCtx.SetTimestamp(int64(i + 1))

		var output *game.FrameOutput
		world, output = Step(world, &game.FrameInput{}, debugCtx)
		fmt.Println(output.Message)

		// Check for assertion failures (optional - for demonstration)
		if debugCtx.HasFailedAssertions() {
			for _, a := range debugCtx.FailedAssertions() {
				fmt.Printf("  ASSERTION FAILED at %s: %s\n", a.Location, a.Message)
			}
		}
	}

	fmt.Println("\nSimulation complete!")

	// Show final debug summary
	showDebugOutput(debugCtx, "final")
}

// showDebugOutput displays collected debug data (for demonstration)
func showDebugOutput(d *game.DebugContext, phase string) {
	output := d.Collect()
	if len(output.Logs) > 0 {
		fmt.Printf("\n[Debug logs from %s]\n", phase)
		for _, log := range output.Logs {
			fmt.Printf("  [t=%d] %s (%s)\n", log.Timestamp, log.Message, log.Location)
		}
	}
	if len(output.Assertions) > 0 {
		fmt.Printf("\n[Assertions from %s]\n", phase)
		for _, a := range output.Assertions {
			status := "PASS"
			if !a.Passed {
				status = "FAIL"
			}
			fmt.Printf("  [%s] %s (%s)\n", status, a.Message, a.Location)
		}
	}
}
