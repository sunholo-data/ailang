package iface

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/types"
)

// ValidateIFC is shared by full-cache decoders and artifact readers (including
// injected codecs). A byte hash alone cannot authorize an obsolete schema.
func (i *Iface) ValidateIFC() error {
	if i == nil || i.Schema != SchemaVersion {
		return fmt.Errorf("incompatible IFC interface schema (require %s); recompile module from source", SchemaVersion)
	}
	if i.Exports == nil {
		return fmt.Errorf("missing interface exports; recompile module from source")
	}
	names := make([]string, 0, len(i.Exports))
	for name := range i.Exports {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		item := i.Exports[name]
		if item == nil {
			return fmt.Errorf("missing IFC export %s; recompile module from source", name)
		}
		if err := item.IFC.Validate(); err != nil {
			return fmt.Errorf("export %s: %w", name, err)
		}
		arity := 0
		if item.Type != nil {
			if fn, ok := item.Type.Type.(*types.TFunc2); ok {
				arity = len(fn.Params)
			}
		}
		if len(item.IFC.Params) != arity {
			return fmt.Errorf("export %s: IFC parameter count does not match type; recompile module from source", name)
		}
	}
	return nil
}
