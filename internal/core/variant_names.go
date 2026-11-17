package core

import "regexp"

// numVariantSuffix matches the suffix a numeric variant adds to a binding's
// name (#2378, types.NumVariantName): `inc$numF`, `qi$numFI`, also inside a
// derived name (`f$numF$lambda3`).
var numVariantSuffix = regexp.MustCompile(`\$num[IF]+`)

// DisplayFuncName is a compiled function's name as the author wrote it: a
// numeric variant (`inc$numF`) is shown as its binding (`inc`). A message that
// names a compiled function to a user goes through it.
func DisplayFuncName(name string) string {
	return numVariantSuffix.ReplaceAllString(name, "")
}
