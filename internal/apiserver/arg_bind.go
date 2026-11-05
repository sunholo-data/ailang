package apiserver

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"mime/multipart"
	"reflect"
	"strconv"

	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

// bindCallArgs is the conversion boundary shared by all JSON call surfaces.
// Resolution (presence, defaults, files and headers) must happen before binding.
func bindCallArgs(exp ExportInfo, args []any) ([]any, error) {
	out := make([]any, len(args))
	for i, v := range args {
		s := ParamSchema{}
		if i < len(exp.ParamSchemas) {
			s = exp.ParamSchemas[i]
		}
		name := fmt.Sprintf("argument %d", i+1)
		if i < len(exp.ParamNames) {
			name = exp.ParamNames[i]
		}
		value, err := bindArg(s, v, name)
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}

func bindArg(s ParamSchema, v any, path string) (eval.Value, error) {
	if v == nil {
		return &eval.UnitValue{}, nil
	}
	if ev, ok := v.(eval.Value); ok {
		return ev, nil
	}
	invalid := func() (eval.Value, error) { return nil, argumentTypeError(s, v, path) }
	switch s.Kind {
	case "int":
		n, ok := integerArg(v)
		if !ok {
			return invalid()
		}
		return &eval.IntValue{Value: n}, nil
	case "float":
		n, ok := numericArg(v)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return invalid()
		}
		return &eval.FloatValue{Value: n}, nil
	case "bool":
		b, ok := v.(bool)
		if !ok {
			return invalid()
		}
		return &eval.BoolValue{Value: b}, nil
	case "string":
		str, ok := v.(string)
		if !ok {
			return invalid()
		}
		return &eval.StringValue{Value: str}, nil
	case "record":
		return bindRecord(s, v, path)
	case "list", "array":
		return bindSequence(s, v, path)
	}

	b, err := embed.FromGo(v)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return b, nil
}

func bindRecord(s ParamSchema, v any, path string) (eval.Value, error) {
	rec, ok := v.(map[string]any)
	if !ok {
		return nil, argumentTypeError(s, v, path)
	}
	fields := make(map[string]eval.Value, len(rec))
	// Declared fields are visited in declaration order, including nested errors.
	for _, f := range s.Fields {
		if value, present := rec[f.Name]; present {
			b, err := bindArg(f.Schema, value, path+"."+f.Name)
			if err != nil {
				return nil, err
			}
			fields[f.Name] = b
		}
	}
	for k, value := range rec {
		if _, done := fields[k]; !done {
			b, err := embed.FromGo(value)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", path, k, err)
			}
			fields[k] = b
		}
	}
	return &eval.RecordValue{Fields: fields}, nil
}

func bindSequence(s ParamSchema, v any, path string) (eval.Value, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, argumentTypeError(s, v, path)
	}
	values := make([]eval.Value, rv.Len())
	elem := ParamSchema{}
	if s.Element != nil {
		elem = *s.Element
	}
	for i := range values {
		b, err := bindArg(elem, rv.Index(i).Interface(), fmt.Sprintf("%s[%d]", path, i))
		if err != nil {
			return nil, err
		}
		values[i] = b
	}
	if s.Kind == "array" {
		return eval.NewArray(values), nil
	}
	return &eval.ListValue{Elements: values}, nil
}

func argumentTypeError(s ParamSchema, v any, path string) error {
	return fmt.Errorf("%s: expected %s, got %s (%v)", path, s.Kind, argKind(v), v)
}

// argKind names a decoded argument the way the caller wrote it (JSON terms),
// not the Go type it decoded to: "got number (2.5)", not "got float64".
func argKind(v any) string {
	switch v.(type) {
	case json.Number:
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	}
	return fmt.Sprintf("%T", v)
}

func numericArg(v any) (float64, bool) {
	if n, ok := v.(json.Number); ok {
		f, err := n.Float64()
		return f, err == nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	}
	return 0, false
}

func integerArg(v any) (int, bool) {
	if n, ok := v.(json.Number); ok {
		r, ok := new(big.Rat).SetString(string(n))
		if !ok || !r.IsInt() || !r.Num().IsInt64() {
			return 0, false
		}
		i := r.Num().Int64()
		return int(i), strconv.IntSize == 64 || (i >= math.MinInt32 && i <= math.MaxInt32)
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := rv.Int()
		return int(i), strconv.IntSize == 64 || (i >= math.MinInt32 && i <= math.MaxInt32)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		limit := uint64(^uint(0) >> 1)
		return int(u), u <= limit
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		limit := math.Ldexp(1, strconv.IntSize-1)
		if math.IsNaN(f) || math.IsInf(f, 0) || math.Trunc(f) != f || f < -limit || f >= limit {
			return 0, false
		}
		return int(f), true
	}
	return 0, false
}

// typeFormTextArgs turns multipart text fields into the JSON shape their
// declared type takes, so the shared binder sees the same values a JSON body
// would give it. A form field is always text: without this an int param sent
// as -F n=21 reached the function as the string "21". Only a declared int,
// float, bool, record, list or array field is parsed; string and unknown
// leaves stay text. Text that does not parse is left as it is, so bindCallArgs
// rejects it naming the parameter (never a silent zero). File fields, padded
// zeros and injected values are untouched: only an arg that is exactly the
// field's own text, under the param's own name, is converted.
func typeFormTextArgs(form *multipart.Form, names []string, schemas []ParamSchema, args []any) {
	if form == nil {
		return
	}
	for i, name := range names {
		if i >= len(args) || i >= len(schemas) {
			return
		}
		if _, isFile := form.File[name]; isFile {
			continue
		}
		values, ok := form.Value[name]
		if !ok || len(values) == 0 {
			continue
		}
		if text, ok := args[i].(string); ok && text == values[0] {
			args[i] = formTextValue(schemas[i].Kind, text)
		}
	}
}

func formTextValue(kind, text string) any {
	switch kind {
	case "int", "float":
		if looksNumeric(text) {
			return json.Number(text) // integerArg/numericArg validate it exactly
		}
	case "bool":
		if b, err := strconv.ParseBool(text); err == nil {
			return b
		}
	case "record", "list", "array":
		var v any
		if err := json.Unmarshal([]byte(text), &v); err == nil {
			return v
		}
	}
	return text
}
