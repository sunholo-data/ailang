package types

import (
	"encoding/json"
	"fmt"
)

// requireIFCFields prevents omitted security fields from becoming Go zero
// values. In particular, an absent forbidden label must never become clean.
func requireIFCFields(data []byte, names ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, name := range names {
		value, ok := fields[name]
		if !ok || string(value) == "null" {
			return fmt.Errorf("missing IFC field %s; recompile module from source", name)
		}
	}
	return nil
}

func (s *ImportedIFCSig) UnmarshalJSON(data []byte) error {
	if err := requireIFCFields(data, "params", "return", "has_return_label", "intrinsic", "declassify"); err != nil {
		return err
	}
	type plain ImportedIFCSig
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = ImportedIFCSig(decoded)
	return s.Validate()
}

func (p *IFCParam) UnmarshalJSON(data []byte) error {
	if err := requireIFCFields(data, "name", "label", "forbidden", "applied"); err != nil {
		return err
	}
	type plain IFCParam
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = IFCParam(decoded)
	return nil
}
