package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// encoding/json otherwise accepts duplicate keys (including in nested objects),
// which would let an audit tool and the runner bind different values.
func maintenanceStrictJSON(raw []byte, target any) error {
	scan := json.NewDecoder(bytes.NewReader(raw))
	scan.UseNumber()
	if err := maintenanceJSONValue(scan); err != nil {
		return err
	}
	if _, err := scan.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
func maintenanceJSONValue(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || keys[name] {
				return fmt.Errorf("duplicate or invalid JSON key %q", key)
			}
			keys[name] = true
			if err = maintenanceJSONValue(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err = maintenanceJSONValue(d); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	_, err = d.Token()
	return err
}
