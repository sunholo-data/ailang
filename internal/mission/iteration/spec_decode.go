package iteration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

// Inspect tokens before unmarshalling: encoding/json otherwise accepts duplicate
// keys and case-insensitive struct field aliases, both ambiguous frozen inputs.
func decodeStrict(r io.Reader, limit int, out any) error {
	body, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return err
	}
	if len(body) > limit {
		return fmt.Errorf("JSON exceeds %d bytes", limit)
	}
	if !utf8.Valid(body) {
		return fmt.Errorf("JSON must be UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if err := checkValue(d, reflect.TypeOf(out).Elem()); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected exactly one JSON object")
	}
	return json.Unmarshal(body, out)
}
func checkValue(d *json.Decoder, t reflect.Type) error {
	tok, err := d.Token()
	if err != nil {
		return err
	}
	if tok == nil {
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
			return nil
		}
		return fmt.Errorf("null is not valid for %s", t)
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Map:
		if tok != json.Delim('{') {
			return fmt.Errorf("expected JSON object")
		}
		fields := map[string]reflect.Type{}
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				fields[strings.Split(f.Tag.Get("json"), ",")[0]] = f.Type
			}
		}
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("invalid or duplicate JSON field %q", name)
			}
			seen[name] = true
			var child reflect.Type
			if t.Kind() == reflect.Map {
				child = t.Elem()
			} else {
				child = fields[name]
				if child == nil {
					return fmt.Errorf("unknown JSON field %q", name)
				}
			}
			if err := checkValue(d, child); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		_, err = d.Token()
		return err
	case reflect.Slice:
		if tok != json.Delim('[') {
			return fmt.Errorf("expected JSON array")
		}
		for d.More() {
			if err := checkValue(d, t.Elem()); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	default:
		if _, ok := tok.(json.Delim); ok {
			return fmt.Errorf("expected scalar")
		}
		return nil
	}
}
