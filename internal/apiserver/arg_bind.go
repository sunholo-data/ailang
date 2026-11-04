package apiserver

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
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
	return fmt.Errorf("%s: expected %s, got %T (%v)", path, s.Kind, v, v)
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
