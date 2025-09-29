package link

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// ImportedSym represents an imported symbol with its type information
type ImportedSym struct {
	Ref    core.GlobalRef
	Type   *types.Scheme
	Purity bool
}

// GlobalEnv maps imported names to their symbol information
type GlobalEnv map[string]*ImportedSym

// LinkDiagnostics contains diagnostic information from linking process
// (separate from LinkReport which is for structured error output)
type LinkDiagnostics struct {
	ResolutionTrace []string // Paths tried during resolution
	Suggestions     []string // Suggestions for fixes
}
