package pipeline

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/types"
)

// registerDerivedEq registers the Eq instance of every `deriving (Eq)` type the
// elaborator saw (M-DX19), then requires Eq of each of their fields
// (M-EQ-DERIVE-CONTAINERS R-D5). All types register before any field is checked,
// so derived types may refer to each other and to themselves.
//
// Before R-D5 no field was checked at all: `type H = H(int -> int) deriving (Eq)`
// was accepted and its == silently answered false for equal handlers.
//
// importedAliases carries the importing module's alias definitions so a field
// naming an imported record alias is checked by that record's fields.
func registerDerivedEq(cfg Config, elaborator *elaborate.Elaborator, importedAliases map[string]types.Type) error {
	derivedEqTypes := elaborator.GetDerivedEqTypes()
	sort.Strings(derivedEqTypes)
	for _, typeName := range derivedEqTypes {
		inst := &types.ClassInstance{
			ClassName: "Eq",
			TypeHead:  &types.TCon{Name: typeName},
			Dict: types.Dict{
				"eq":  fmt.Sprintf("derived_eq_%s", typeName),
				"neq": fmt.Sprintf("derived_neq_%s", typeName),
			},
		}
		if err := cfg.InstEnv.Add(inst); err != nil {
			// Duplicate registrations are expected when several files declare
			// into one environment; only surface them in debug mode.
			if cfg.DebugCompile {
				fmt.Fprintf(os.Stderr, "[DEBUG] Could not add derived Eq instance for %s: %v\n", typeName, err)
			}
		}
		// Runtime: the evaluator compares these TaggedValues structurally
		cfg.DictReg.RegisterDerivedEq(typeName)
	}

	aliases := make(map[string]types.Type, len(importedAliases))
	for name, target := range importedAliases {
		aliases[name] = target
	}
	for name, target := range elaborator.GetTypeAliases() {
		aliases[name] = target
	}
	fields := elaborator.GetDerivedEqFields()
	for _, typeName := range derivedEqTypes {
		for _, f := range fields[typeName] {
			if err := cfg.InstEnv.CheckDerivedEqField(f.Type, aliases); err != nil {
				detail := err.Error()
				var missing *types.MissingInstanceError
				if errors.As(err, &missing) {
					detail = missing.Hint
				}
				return fmt.Errorf("cannot derive Eq for %s: %s has type %s, which has no ==. "+
					"Every field of a `deriving (Eq)` type needs ==. %s", typeName, f.Where, f.Type, detail)
			}
		}
	}
	return nil
}
