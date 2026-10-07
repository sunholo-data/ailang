package builtins

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/yaml", Name: "_yaml_encode", NumArgs: 1, IsPure: true,
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Con("Json")).Returns(T.App("Result", T.String(), T.String())).Build()
		},
		Impl: yamlEncodeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Encode Json as deterministic block YAML, preserving object pair order",
			LongDesc:    "Quotes all keys and strings, uses two-space indentation and one trailing newline. Duplicate keys are emitted unchanged; decode rejects them. Non-finite numbers return Err.",
			Params:      []ParamDoc{{Name: "value", Description: "Json value to encode"}},
			Returns:     "Result[string, string] - Ok(YAML string) or Err(error message)",
			Examples:    []Example{{Code: `_yaml_encode(JNull)`, Description: `Returns Ok("null\n")`}},
			SeeAlso:     []string{"std/yaml.decode", "std/json.encode"},
			Since:       "v0.53.0", Stability: StabilityStable, Tags: []string{"yaml", "encoding", "data", "result"}, Category: "yaml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _yaml_encode: %v", err))
	}
}

func yamlEncodeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	var buf strings.Builder
	if err := emitYAML(args[0], &buf, 0); err != nil {
		return wrapErr("yaml encode: " + err.Error()), nil
	}
	buf.WriteByte('\n')
	return wrapOk(&eval.StringValue{Value: buf.String()}), nil
}

// emitYAML starts at the caller's current column. indent is the column for
// continuation lines, allowing the first mapping key to follow a sequence dash.
func emitYAML(v eval.Value, buf *strings.Builder, indent int) error {
	tagged, ok := v.(*eval.TaggedValue)
	if !ok || tagged.TypeName != "Json" {
		return fmt.Errorf("expected Json ADT, got %T", v)
	}
	if tagged.CtorName == "JArray" || tagged.CtorName == "JObject" {
		if len(tagged.Fields) != 1 {
			return fmt.Errorf("%s expected one field", tagged.CtorName)
		}
		list, ok := tagged.Fields[0].(*eval.ListValue)
		if !ok {
			return fmt.Errorf("%s expected list", tagged.CtorName)
		}
		if len(list.Elements) == 0 {
			if tagged.CtorName == "JArray" {
				buf.WriteString("[]")
			} else {
				buf.WriteString("{}")
			}
			return nil
		}
		for i, item := range list.Elements {
			if i > 0 {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat(" ", indent))
			}
			if tagged.CtorName == "JArray" {
				buf.WriteString("- ")
				if err := emitYAML(item, buf, indent+2); err != nil {
					return err
				}
			} else {
				pair, ok := item.(*eval.RecordValue)
				if !ok {
					return fmt.Errorf("JObject expected key-value record")
				}
				key, ok := pair.Fields["key"].(*eval.StringValue)
				if !ok {
					return fmt.Errorf("JObject expected string key")
				}
				value, ok := pair.Fields["value"]
				if !ok {
					return fmt.Errorf("JObject missing value")
				}
				var keyBuf strings.Builder
				writeYAMLString(key.Value, &keyBuf)
				// YAML simple keys must have fewer than 1024 characters
				// including quotes. Explicit keys have no such limit.
				explicit := utf8.RuneCountInString(keyBuf.String()) >= 1024
				if explicit {
					buf.WriteString("? ")
				}
				buf.WriteString(keyBuf.String())
				if explicit {
					buf.WriteByte('\n')
					buf.WriteString(strings.Repeat(" ", indent))
				}
				buf.WriteByte(':')
				if yamlBlock(value) {
					buf.WriteByte('\n')
					buf.WriteString(strings.Repeat(" ", indent+2))
				} else {
					buf.WriteByte(' ')
				}
				if err := emitYAML(value, buf, indent+2); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if tagged.CtorName == "JNumber" && len(tagged.Fields) == 1 {
		if f, ok := tagged.Fields[0].(*eval.FloatValue); ok && (math.IsNaN(f.Value) || math.IsInf(f.Value, 0)) {
			return fmt.Errorf("non-finite number cannot be represented as JSON-compatible YAML")
		}
	}
	if tagged.CtorName == "JString" && len(tagged.Fields) == 1 {
		if s, ok := tagged.Fields[0].(*eval.StringValue); ok {
			writeYAMLString(s.Value, buf)
			return nil
		}
	}
	// Shared scalar validation, escaping and canonical number formatting.
	return encodeValue(v, buf)
}

func yamlBlock(v eval.Value) bool {
	tagged, ok := v.(*eval.TaggedValue)
	if !ok || (tagged.CtorName != "JArray" && tagged.CtorName != "JObject") || len(tagged.Fields) != 1 {
		return false
	}
	list, ok := tagged.Fields[0].(*eval.ListValue)
	return ok && len(list.Elements) > 0
}

func writeYAMLString(s string, buf *strings.Builder) {
	buf.WriteByte('"')
	// YAML rejects C1 controls and normalizes literal NEL. Escape these
	// with JSON-compatible Unicode escapes; retain the shared JSON routine
	// for all other text (including the ordinary ASCII escape set).
	start := 0
	for i, r := range s {
		if (r >= 0x7f && r <= 0x9f) || r == 0x2028 || r == 0x2029 || r == 0xfffe || r == 0xffff {
			escapeString(s[start:i], buf)
			fmt.Fprintf(buf, `\u%04x`, r)
			start = i + len(string(r))
		}
	}
	escapeString(s[start:], buf)
	buf.WriteByte('"')
}
