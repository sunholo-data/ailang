package builtins

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	for _, entry := range []struct {
		name          string
		strict, check bool
	}{
		{"_jsonschema_validate", true, false},
		{"_jsonschema_validate_lenient", false, false},
		{"_jsonschema_check_schema", true, true},
	} {
		check := entry.check
		example := entry.name + "(schema, doc)"
		if check {
			example = entry.name + "(schema)"
		}
		n := 2
		if check {
			n = 1
		}
		err := RegisterEffectBuiltin(BuiltinSpec{
			Module: "std/jsonschema", Name: entry.name, NumArgs: n, IsPure: true,
			Type: func() types.Type {
				T := types.NewBuilder()
				record := T.Record(types.Field("path", T.String()), types.Field("keyword", T.String()), types.Field("message", T.String()))
				if check {
					return T.Func(T.Con("Json")).Returns(T.List(record)).Build()
				}
				return T.Func(T.Con("Json"), T.Con("Json")).Returns(T.List(record)).Build()
			},
			Impl: jsonschemaImpl(entry.strict, entry.check),
			Metadata: &BuiltinMetadata{
				Description: "Validate Draft 2020-12 JSON schemas and documents with deterministic structured errors",
				LongDesc:    "Pure, local references only. Strict validation rejects unknown keywords; lenient validation skips only unknown keywords. Patterns use RE2; format is annotation-only. Schema errors precede document validation.",
				Params:      jsonschemaParams(check), Returns: "List of {path, keyword, message} errors; empty on success",
				Examples: []Example{{Code: example, Description: "See std/jsonschema for typed public wrappers"}},
				Since:    "v0.57.1", Stability: StabilityStable, Tags: []string{"json", "schema", "validation"}, Category: "json",
			},
		})
		if err != nil {
			panic(fmt.Sprintf("register %s: %v", entry.name, err))
		}
	}
}

func jsonschemaParams(check bool) []ParamDoc {
	p := []ParamDoc{{Name: "schema", Description: "Draft 2020-12 schema as Json"}}
	if !check {
		p = append(p, ParamDoc{Name: "doc", Description: "Document as Json"})
	}
	return p
}

func jsonschemaImpl(strict, check bool) EffectImpl {
	return func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
		schema, err := jsonschemaNative(args[0])
		if err != nil {
			return nil, err
		}
		program, errors := compileSchema(schema, strict)
		if len(errors) == 0 && !check {
			doc, err := jsonschemaNative(args[1])
			if err != nil {
				return nil, err
			}
			errors = validateCompiledSchema(program, doc)
		}
		values := make([]eval.Value, len(errors))
		for i, e := range errors {
			values[i] = &eval.RecordValue{Fields: map[string]eval.Value{
				"path":    &eval.StringValue{Value: e.Path},
				"keyword": &eval.StringValue{Value: e.Keyword},
				"message": &eval.StringValue{Value: e.Message},
			}}
		}
		return &eval.ListValue{Elements: values}, nil
	}
}

// Reuse the canonical Json ADT encoder, including its malformed-value checks.
// Decoding the emitted representation preserves the ADT's float64 number model.
func jsonschemaNative(v eval.Value) (any, error) {
	var buffer strings.Builder
	if err := encodeValue(v, &buffer); err != nil {
		return nil, fmt.Errorf("jsonschema: %w", err)
	}
	var value any
	if err := json.Unmarshal([]byte(buffer.String()), &value); err != nil {
		return nil, fmt.Errorf("jsonschema: %w", err)
	}
	return value, nil
}
