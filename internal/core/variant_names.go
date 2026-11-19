package core

import "regexp"

// compiledNameSuffix matches the suffixes compilation adds to a binding's
// name: a numeric variant's (#2378, types.NumVariantName: `inc$numF`,
// `qi$numFI`) and the local uniquifier's (gen/lower uniquify: `h$s2`), also
// inside a derived name (`f$numF$lambda3`).
var compiledNameSuffix = regexp.MustCompile(`\$(num[IF]+|s[0-9]+)`)

// DisplayFuncName is a compiled function's name as the author wrote it: a
// numeric variant (`inc$numF`) is shown as its binding (`inc`), a uniquified
// local (`h$s2`) as the local (`h`). A message or trace that names a compiled
// function to a user goes through it.
func DisplayFuncName(name string) string {
	return compiledNameSuffix.ReplaceAllString(name, "")
}
