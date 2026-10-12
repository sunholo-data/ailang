package iface

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON makes absent IFC metadata an error for published v2 interfaces.
// Legacy compact interfaces remain readable for inspection, never as v2 data.
func (i *InterfaceJSON) UnmarshalJSON(data []byte) error {
	type plain InterfaceJSON
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*i = InterfaceJSON(decoded)
	if i.Schema == SchemaVersion {
		if i.Funcs == nil {
			return fmt.Errorf("missing v2 interface exports; recompile module from source")
		}
		for _, fn := range i.Funcs {
			if err := fn.IFC.Validate(); err != nil {
				return fmt.Errorf("export %s: %w", fn.Name, err)
			}
		}
	}
	return nil
}
