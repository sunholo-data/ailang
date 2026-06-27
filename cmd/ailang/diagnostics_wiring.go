package main

import (
	"github.com/sunholo-data/ailang/internal/stdlibindex"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-AGENT-ERGONOMICS: wire stdlib import suggestions into "undefined variable" type errors.
// Kept in the CLI (not internal/types) so the type checker stays free of a stdlib-scan dependency.
func init() {
	types.ImportSuggester = stdlibindex.Modules
}
