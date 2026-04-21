package main

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// findEntryProto resolves an entry name (e.g. "main") to a prototype in the
// image. The lower pass prefixes function names with the module package, so
// we accept several spellings.
func findEntryProto(img *bytecode.BytecodeImage, name string) *bytecode.FuncPrototype {
	for _, p := range img.Prototypes {
		if p.Name == name {
			return p
		}
		// Tolerate "<pkg>.<name>" and "<pkg>_<name>" qualifications.
		if strings.HasSuffix(p.Name, "."+name) || strings.HasSuffix(p.Name, "_"+name) {
			return p
		}
	}
	return nil
}
