package runtime

import "fmt"

// ConcatString concatenates two values as strings.
// This implements the AILANG ++ operator for strings.
func ConcatString(a, b any) string {
	return fmt.Sprintf("%v%v", a, b)
}
