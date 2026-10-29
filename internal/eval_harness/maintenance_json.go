package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
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

	var value any
	fields := json.NewDecoder(bytes.NewReader(raw))
	fields.UseNumber()
	if err := fields.Decode(&value); err != nil {
		return err
	}
	if err := maintenanceExactJSONFields(value, reflect.TypeOf(target)); err != nil {
		return err
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

// DisallowUnknownFields alone accepts case-insensitive aliases of JSON tags.
// Exact schema names are required so JSON Schema, native analysis and Go bind
// the same contract (including nested trajectory and stage objects).
func maintenanceExactJSONFields(value any, typ reflect.Type) error {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch actual := value.(type) {
	case map[string]any:
		if typ.Kind() != reflect.Struct {
			return nil
		}
		fields := map[string]reflect.Type{}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			fields[name] = field.Type
		}
		for name, child := range actual {
			childType, ok := fields[name]
			if !ok {
				return fmt.Errorf("unknown exact JSON field %q", name)
			}
			if err := maintenanceExactJSONFields(child, childType); err != nil {
				return err
			}
		}
	case []any:
		if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
			return nil
		}
		for _, child := range actual {
			if err := maintenanceExactJSONFields(child, typ.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}
