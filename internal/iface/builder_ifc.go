package iface

import "github.com/sunholo-data/ailang/internal/types"

// WithIFCImports supplies summaries keyed by resolved surface names. The
// caller owns import identity resolution; the builder only analyzes bodies.
func (b *Builder) WithIFCImports(imports map[string]*types.ImportedIFCSig) *Builder {
	b.imports = imports
	return b
}
