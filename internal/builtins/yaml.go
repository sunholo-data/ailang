package builtins

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// std/yaml is a thin bridge: it parses a YAML string with gopkg.in/yaml.v3 and
// re-emits it as a JSON string via encoding/json. Callers hand the result to
// std/json.decode to obtain the shared Json ADT. Keeping the bridge in one pure
// builtin (string -> string) means std/yaml.decode is pure AILANG composition
// with zero additional Go surface. The library is pure Go and WASM-portable.

func init() {
	registerYAMLToJSON()
}

func registerYAMLToJSON() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/yaml",
		Name:    "_yaml_to_json",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeYAMLToJSONType,
		Impl:    yamlToJSONImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert a YAML string to an equivalent JSON string",
			LongDesc: `Parses a YAML document with gopkg.in/yaml.v3 and re-emits it as a JSON string.
The output is intended to be handed to std/json.decode to obtain the Json ADT.
Only the first document of a multi-document stream is read. Any YAML that cannot be
represented as JSON (non-string map keys, NaN/Inf floats) returns Err — no silent coercion.
Returns Result[string, string] - Ok(jsonString) on success, Err(message) on failure.`,
			Params: []ParamDoc{
				{Name: "input", Description: "YAML string to convert"},
			},
			Returns: "Result[string, string] - Ok(JSON string) on success, Err(error message) on failure",
			Examples: []Example{
				{Code: `_yaml_to_json("a: 1\nb: [x, y]\n")`, Description: `Returns Ok("{\"a\":1,\"b\":[\"x\",\"y\"]}")`},
				{Code: `_yaml_to_json("")`, Description: `Returns Ok("null")`},
				{Code: `_yaml_to_json("1: a\n")`, Description: "Returns Err(...) — non-string map key cannot be JSON"},
			},
			SeeAlso:   []string{"std/yaml.decode", "std/json.decode"},
			Since:     "v0.30.0",
			Stability: StabilityStable,
			Tags:      []string{"yaml", "parsing", "data", "result"},
			Category:  "yaml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _yaml_to_json: %v", err))
	}
}

func makeYAMLToJSONType() types.Type {
	T := types.NewBuilder()
	// Type signature: string -> Result[string, string]
	resultType := T.App("Result", T.String(), T.String())
	return T.Func(T.String()).Returns(resultType).Build()
}

// GetYAMLToJSONImpl exports the implementation for legacy registry integration.
func GetYAMLToJSONImpl() EffectImpl {
	return yamlToJSONImpl
}

func yamlToJSONImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	sv, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_yaml_to_json: expected string, got %T", args[0])
	}

	// Decode YAML into a generic Go value. yaml.v3 reads only the first document
	// of a multi-document stream, which is the documented single-document behavior.
	var v interface{}
	if err := yaml.Unmarshal([]byte(sv.Value), &v); err != nil {
		return wrapErr(fmt.Sprintf("yaml: %s", err.Error())), nil
	}

	// Re-emit as JSON. Marshal fails loudly on constructs that have no JSON
	// representation (non-string map keys, NaN/Inf) — we return Err rather than
	// coercing, per the no-silent-fallback policy.
	b, err := json.Marshal(v)
	if err != nil {
		return wrapErr(fmt.Sprintf("yaml: cannot represent as JSON: %s", err.Error())), nil
	}

	return wrapOk(&eval.StringValue{Value: string(b)}), nil
}
