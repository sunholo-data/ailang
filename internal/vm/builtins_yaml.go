package vm

// VM-native YAML encode. The emitter itself lives in internal/builtins
// (EncodeYAML) so the evaluator and the VM produce byte-identical YAML; this
// file only rebuilds the Json ADT on the evaluator side. That is possible here
// although the generic bridge cannot name VM ADTs, because the Json tag order is
// fixed by std/json.ail (see the jsonTag* constants in builtins_json.go).

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// --- yaml_encode: Json -> Result[string, string] ------------------------------

func builtinYamlEncode(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__yaml_encode: expected 1 arg, got %d", len(args))
	}
	j, err := vmJsonToEval(args[0])
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("__yaml_encode: %w", err)
	}
	out, err := builtins.EncodeYAML(j)
	if err != nil {
		return vmResultErr(err.Error()), nil
	}
	return vmResultOk(bytecode.NewString(out)), nil
}

var jsonCtorNames = [...]string{
	jsonTagJNull:   "JNull",
	jsonTagJBool:   "JBool",
	jsonTagJNumber: "JNumber",
	jsonTagJString: "JString",
	jsonTagJArray:  "JArray",
	jsonTagJObject: "JObject",
}

// vmJsonToEval converts a VM Json ADT into the evaluator's TaggedValue form.
func vmJsonToEval(v bytecode.Value) (eval.Value, error) {
	if v.Tag != bytecode.TagADT {
		return nil, fmt.Errorf("expected Json ADT, got tag %d", v.Tag)
	}
	adt := v.AsADT()
	if adt.Tag < 0 || adt.Tag >= len(jsonCtorNames) {
		return nil, fmt.Errorf("unknown Json tag %d", adt.Tag)
	}
	ctor := jsonCtorNames[adt.Tag]
	if adt.Tag == jsonTagJNull {
		return &eval.TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: ctor}, nil
	}
	if len(adt.Fields) != 1 {
		return nil, fmt.Errorf("%s: expected 1 field, got %d", ctor, len(adt.Fields))
	}
	f := adt.Fields[0]

	var field eval.Value
	switch adt.Tag {
	case jsonTagJBool:
		field = &eval.BoolValue{Value: f.Bool}
	case jsonTagJNumber:
		switch f.Tag {
		case bytecode.TagFloat:
			field = &eval.FloatValue{Value: f.Flt}
		case bytecode.TagInt:
			field = &eval.FloatValue{Value: float64(f.Int)}
		default:
			return nil, fmt.Errorf("JNumber: expected numeric, got tag %d", f.Tag)
		}
	case jsonTagJString:
		field = &eval.StringValue{Value: f.AsString()}
	case jsonTagJArray:
		elems := f.AsList()
		out := make([]eval.Value, len(elems))
		for i, e := range elems {
			ev, err := vmJsonToEval(e)
			if err != nil {
				return nil, err
			}
			out[i] = ev
		}
		field = &eval.ListValue{Elements: out}
	case jsonTagJObject:
		pairs := f.AsList()
		out := make([]eval.Value, len(pairs))
		for i, kv := range pairs {
			if kv.Tag != bytecode.TagRecord {
				return nil, fmt.Errorf("JObject entry: expected record, got tag %d", kv.Tag)
			}
			fields := kv.AsRecord()
			// Records are sorted alphabetically: key=0, value=1
			if len(fields) != 2 || fields[0].Name != "key" || fields[1].Name != "value" {
				return nil, fmt.Errorf("JObject entry: expected {key, value} record")
			}
			val, err := vmJsonToEval(fields[1].Value)
			if err != nil {
				return nil, err
			}
			out[i] = &eval.RecordValue{Fields: map[string]eval.Value{
				"key":   &eval.StringValue{Value: fields[0].Value.AsString()},
				"value": val,
			}}
		}
		field = &eval.ListValue{Elements: out}
	}
	return &eval.TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: ctor, Fields: []eval.Value{field}}, nil
}
