package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerGetHeader()
	registerHasHeader()
}

// registerGetHeader registers the _get_header builtin.
// Type: ({string: string}, string) -> Option[string]
func registerGetHeader() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/http",
		Name:    "_get_header",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			a := T.Var("a")
			return T.Func(a, T.String()).Returns(T.App("Option", T.String())).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			rec, ok := args[0].(*eval.RecordValue)
			if !ok {
				return nil, fmt.Errorf("_get_header: expected record, got %T", args[0])
			}
			nameVal, ok := args[1].(*eval.StringValue)
			if !ok {
				return nil, fmt.Errorf("_get_header: expected string, got %T", args[1])
			}

			// Headers are stored with lowercase keys
			name := strings.ToLower(nameVal.Value)
			if val, exists := rec.Fields[name]; exists {
				if sv, ok := val.(*eval.StringValue); ok {
					return &eval.TaggedValue{
						ModulePath: "std/option",
						TypeName:   "Option",
						CtorName:   "Some",
						Fields:     []eval.Value{&eval.StringValue{Value: sv.Value}},
					}, nil
				}
			}

			return &eval.TaggedValue{
				ModulePath: "std/option",
				TypeName:   "Option",
				CtorName:   "None",
				Fields:     []eval.Value{},
			}, nil
		},
		Metadata: &BuiltinMetadata{
			Description: "Get an HTTP header value from a headers record",
			LongDesc:    "Looks up a header by name (case-insensitive) in a headers record. Returns Some(value) if found, None otherwise.",
			Params: []ParamDoc{
				{Name: "headers", Description: "Headers record from serve-api"},
				{Name: "name", Description: "Header name (case-insensitive)"},
			},
			Returns:   "Option[string]: Some(value) if header exists, None otherwise",
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"http", "headers", "serve-api"},
			Category:  "http",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _get_header: %v", err))
	}
}

// registerHasHeader registers the _has_header builtin.
// Type: ({string: string}, string) -> bool
func registerHasHeader() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/http",
		Name:    "_has_header",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type: func() types.Type {
			T := types.NewBuilder()
			a := T.Var("a")
			return T.Func(a, T.String()).Returns(T.Bool()).Build()
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			rec, ok := args[0].(*eval.RecordValue)
			if !ok {
				return nil, fmt.Errorf("_has_header: expected record, got %T", args[0])
			}
			nameVal, ok := args[1].(*eval.StringValue)
			if !ok {
				return nil, fmt.Errorf("_has_header: expected string, got %T", args[1])
			}

			name := strings.ToLower(nameVal.Value)
			_, exists := rec.Fields[name]
			return &eval.BoolValue{Value: exists}, nil
		},
		Metadata: &BuiltinMetadata{
			Description: "Check if an HTTP header exists in a headers record",
			LongDesc:    "Returns true if the header name exists in the headers record (case-insensitive).",
			Params: []ParamDoc{
				{Name: "headers", Description: "Headers record from serve-api"},
				{Name: "name", Description: "Header name (case-insensitive)"},
			},
			Returns:   "true if header exists, false otherwise",
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"http", "headers", "serve-api"},
			Category:  "http",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _has_header: %v", err))
	}
}
