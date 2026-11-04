package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// JSON Schema's engine is shared with the interpreter. Json ADTs cross through
// the existing JSON codec, which supplies constructor identities explicitly.
func builtinJSONSchema(name string, args []bytecode.Value) (bytecode.Value, error) {
	spec, ok := builtins.GetSpec(name)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("JSON Schema builtin %s is not registered", name)
	}
	if len(args) != spec.NumArgs {
		return bytecode.Value{}, fmt.Errorf("%s: expected %d arguments, got %d", name, spec.NumArgs, len(args))
	}
	decode, ok := builtins.GetSpec("_json_decode")
	if !ok {
		return bytecode.Value{}, fmt.Errorf("JSON decoder is not registered")
	}
	values := make([]eval.Value, len(args))
	for i, arg := range args {
		encoded, err := builtinJsonEncode([]bytecode.Value{arg})
		if err != nil {
			return bytecode.Value{}, err
		}
		decoded, err := decode.Impl(nil, []eval.Value{&eval.StringValue{Value: encoded.AsString()}})
		if err != nil {
			return bytecode.Value{}, err
		}
		result, ok := decoded.(*eval.TaggedValue)
		if !ok || result.CtorName != "Ok" || len(result.Fields) != 1 {
			return bytecode.Value{}, fmt.Errorf("%s: cannot decode Json argument %d: %v", name, i, decoded)
		}
		values[i] = result.Fields[0]
	}
	result, err := spec.Impl(nil, values)
	if err != nil {
		return bytecode.Value{}, err
	}
	return EvalToBytecode(result)
}
func builtinJSONSchemaValidate(args []bytecode.Value) (bytecode.Value, error) {
	return builtinJSONSchema("_jsonschema_validate", args)
}
func builtinJSONSchemaValidateLenient(args []bytecode.Value) (bytecode.Value, error) {
	return builtinJSONSchema("_jsonschema_validate_lenient", args)
}
func builtinJSONSchemaCheckSchema(args []bytecode.Value) (bytecode.Value, error) {
	return builtinJSONSchema("_jsonschema_check_schema", args)
}
