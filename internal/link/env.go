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

// LinkReport contains diagnostic information from linking
type LinkReport struct {
	ResolutionTrace []string // Paths tried during resolution
	Suggestions     []string // Suggestions for fixes
}
