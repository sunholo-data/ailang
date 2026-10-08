package builtins

import (
	"bytes"
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
			Description: "Convert YAML to JSON preserving mapping document order",
			LongDesc: `Parses a YAML document with gopkg.in/yaml.v3 and re-emits it as a JSON string.
Mapping keys retain document order. Merges insert inherited keys at the merge position;
explicit keys override merged keys, and earlier merge sources win conflicts.
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

	// Parse only the first document, retaining mapping pairs in source order.
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(sv.Value), &node); err != nil {
		return wrapErr(fmt.Sprintf("yaml: %s", err)), nil
	}
	// Keep yaml.v3's validation, including duplicate keys, cyclic aliases and its
	// excessive-alias expansion guard. This value is never used for emission.
	var validated interface{}
	if err := node.Decode(&validated); err != nil {
		return wrapErr(fmt.Sprintf("yaml: %s", err)), nil
	}
	if _, err := json.Marshal(validated); err != nil {
		return wrapErr(fmt.Sprintf("yaml: cannot represent as JSON: %s", err)), nil
	}
	b, err := orderedYAMLJSON(&node)
	if err != nil {
		return wrapErr(fmt.Sprintf("yaml: %s", err)), nil
	}
	return wrapOk(&eval.StringValue{Value: string(b)}), nil
}

// orderedYAMLJSON uses maps only for membership; output follows Node.Content.
// yaml.v3 validation above bounds alias expansion before this traversal.
func orderedYAMLJSON(n *yaml.Node) ([]byte, error) {
	switch n.Kind {
	case 0:
		return []byte("null"), nil
	case yaml.DocumentNode:
		return orderedYAMLJSON(n.Content[0])
	case yaml.AliasNode:
		return orderedYAMLJSON(n.Alias)
	case yaml.MappingNode:
		pairs, err := orderedYAMLPairs(n)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		out.WriteByte('{')
		for i, pair := range pairs {
			if i > 0 {
				out.WriteByte(',')
			}
			key, _ := json.Marshal(pair.key)
			out.Write(key)
			out.WriteByte(':')
			value, err := orderedYAMLJSON(pair.value)
			if err != nil {
				return nil, err
			}
			out.Write(value)
		}
		out.WriteByte('}')
		return out.Bytes(), nil
	case yaml.SequenceNode:
		var out bytes.Buffer
		out.WriteByte('[')
		for i, child := range n.Content {
			if i > 0 {
				out.WriteByte(',')
			}
			value, err := orderedYAMLJSON(child)
			if err != nil {
				return nil, err
			}
			out.Write(value)
		}
		out.WriteByte(']')
		return out.Bytes(), nil
	default:
		var value interface{}
		if err := n.Decode(&value); err != nil {
			return nil, err
		}
		return json.Marshal(value)
	}
}

type yamlJSONPair struct {
	key   string
	value *yaml.Node
}

func orderedYAMLPairs(n *yaml.Node) ([]yamlJSONPair, error) {
	for n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("map merge requires map or sequence of maps as the value")
	}
	explicit := make(map[string]bool)
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i]
		if k.Tag == "!!merge" {
			continue
		}
		var key interface{}
		if err := k.Decode(&key); err != nil {
			return nil, err
		}
		text, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("cannot represent as JSON: non-string mapping key")
		}
		explicit[text] = true
	}
	var pairs []yamlJSONPair
	seen := make(map[string]bool)
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if k.Tag != "!!merge" {
			var key string
			if err := k.Decode(&key); err != nil {
				return nil, err
			}
			pairs = append(pairs, yamlJSONPair{key, v})
			continue
		}
		for v.Kind == yaml.AliasNode {
			v = v.Alias
		}
		sources := []*yaml.Node{v}
		if v.Kind == yaml.SequenceNode {
			sources = v.Content
		}
		for _, source := range sources {
			merged, err := orderedYAMLPairs(source)
			if err != nil {
				return nil, err
			}
			for _, pair := range merged {
				if !explicit[pair.key] && !seen[pair.key] {
					pairs = append(pairs, pair)
					seen[pair.key] = true
				}
			}
		}
	}
	return pairs, nil
}
