package runtime

import "fmt"

// Log prints a message to stdout and returns unit.
// This implements the AILANG log builtin for the IO effect.
func Log(msg any) any {
	fmt.Println(msg)
	return struct{}{}
}

// Debug prints a debug message with a label.
// Useful for debugging generated code.
func Debug(label string, value any) any {
	fmt.Printf("[DEBUG] %s: %v\n", label, value)
	return value
}
