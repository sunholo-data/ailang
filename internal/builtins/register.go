package builtins

import (
	"fmt"
	"unicode/utf8"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// This file contains builtin registrations using the new spec-based registry.
// Builtins are migrated here from their old scattered locations.

func init() {
	// Register string primitive builtins
	registerStringLen()

	// Register Net effect builtins
	registerNetHTTPRequest()
}

// registerStringLen registers the _str_len builtin
// Old location: internal/eval/builtins.go
func registerStringLen() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_len",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeStrLenType,
		Impl:    strLenImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_len: %v", err))
	}
}

// makeStrLenType builds the type signature for _str_len
// Type: (String) -> Int
func makeStrLenType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Int()).Build()
}

// strLenImpl is the implementation for _str_len
// UTF-8 aware string length (returns number of runes, not bytes)
func strLenImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract string argument
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_str_len: expected String, got %T", args[0])
	}

	// Count UTF-8 runes
	count := utf8.RuneCountInString(strVal.Value)

	return &eval.IntValue{Value: count}, nil
}

// registerNetHTTPRequest registers the _net_httpRequest builtin
// Old location: internal/effects/net.go
func registerNetHTTPRequest() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_httpRequest",
		NumArgs: 4,
		IsPure:  false,
		Effect:  "Net",
		Type:    makeHTTPRequestType,
		Impl:    effects.NetHTTPRequest,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_httpRequest: %v", err))
	}
}

// makeHTTPRequestType builds the type signature for _net_httpRequest
// Type: (String, String, List<{name: String, value: String}>, String)
//
//	-> Result<{status: Int, headers: List<{name: String, value: String}>, body: String, ok: Bool}, NetError>
//	! {Net}
func makeHTTPRequestType() types.Type {
	T := types.NewBuilder()

	// Header type: {name: String, value: String}
	headerType := T.Record(
		types.Field("name", T.String()),
		types.Field("value", T.String()),
	)

	// Response type: {status: Int, headers: List<Header>, body: String, ok: Bool}
	responseType := T.Record(
		types.Field("status", T.Int()),
		types.Field("headers", T.List(headerType)),
		types.Field("body", T.String()),
		types.Field("ok", T.Bool()),
	)

	// Function signature with effects
	return T.Func(
		T.String(),         // method
		T.String(),         // url
		T.List(headerType), // headers
		T.String(),         // body
	).Returns(
		T.App("Result", responseType, T.Con("NetError")),
	).Effects("Net")
}
