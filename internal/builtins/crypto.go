package builtins

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerSha256Hex()
	registerSha256Bytes()
	registerHmacSha256()
	registerConstantTimeEqual()
}

// registerSha256Hex registers _crypto_sha256hex: string -> string
func registerSha256Hex() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/crypto",
		Name:    "_crypto_sha256hex",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeSha256HexType,
		Impl:    sha256HexImpl,
		Metadata: &BuiltinMetadata{
			Description: "SHA-256 hash of a string, returned as lowercase hex",
			Params: []ParamDoc{
				{Name: "input", Description: "The string to hash"},
			},
			Returns: "64-character lowercase hex string",
			Examples: []Example{
				{Code: `_crypto_sha256hex("hello")`, Description: "Returns SHA-256 hex digest"},
				{Code: `_crypto_sha256hex("")`, Description: "Returns hash of empty string"},
			},
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"crypto", "hash", "sha256", "security"},
			Category:  "crypto",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _crypto_sha256hex: %v", err))
	}
}

func makeSha256HexType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

func sha256HexImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_sha256hex: expected String, got %T", args[0])
	}
	hash := sha256.Sum256([]byte(strVal.Value))
	return &eval.StringValue{Value: hex.EncodeToString(hash[:])}, nil
}

// registerSha256Bytes registers _crypto_sha256bytes: bytes -> string
func registerSha256Bytes() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/crypto",
		Name:    "_crypto_sha256bytes",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeSha256BytesType,
		Impl:    sha256BytesImpl,
		Metadata: &BuiltinMetadata{
			Description: "SHA-256 hash of raw bytes, returned as lowercase hex",
			Params: []ParamDoc{
				{Name: "input", Description: "The bytes to hash"},
			},
			Returns: "64-character lowercase hex string",
			Examples: []Example{
				{Code: `_crypto_sha256bytes(b)`, Description: "Returns SHA-256 hex digest of bytes"},
			},
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"crypto", "hash", "sha256", "bytes", "security"},
			Category:  "crypto",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _crypto_sha256bytes: %v", err))
	}
}

func makeSha256BytesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes()).Returns(T.String()).Build()
}

func sha256BytesImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_sha256bytes: expected Bytes, got %T", args[0])
	}
	hash := sha256.Sum256(bytesVal.Value)
	return &eval.StringValue{Value: hex.EncodeToString(hash[:])}, nil
}

// registerHmacSha256 registers _crypto_hmacsha256: (string, string) -> string
func registerHmacSha256() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/crypto",
		Name:    "_crypto_hmacsha256",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeHmacSha256Type,
		Impl:    hmacSha256Impl,
		Metadata: &BuiltinMetadata{
			Description: "HMAC-SHA256 keyed hash for message authentication",
			Params: []ParamDoc{
				{Name: "message", Description: "The message to authenticate"},
				{Name: "key", Description: "The secret key"},
			},
			Returns: "64-character lowercase hex string",
			Examples: []Example{
				{Code: `_crypto_hmacsha256("message", "secret")`, Description: "Returns HMAC-SHA256 hex digest"},
			},
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"crypto", "hmac", "sha256", "authentication", "security"},
			Category:  "crypto",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _crypto_hmacsha256: %v", err))
	}
}

func makeHmacSha256Type() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.String()).Build()
}

func hmacSha256Impl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	msgVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_hmacsha256: expected String for message, got %T", args[0])
	}
	keyVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_hmacsha256: expected String for key, got %T", args[1])
	}
	mac := hmac.New(sha256.New, []byte(keyVal.Value))
	mac.Write([]byte(msgVal.Value))
	return &eval.StringValue{Value: hex.EncodeToString(mac.Sum(nil))}, nil
}

// registerConstantTimeEqual registers _crypto_constanttimeequal: (string, string) -> bool
func registerConstantTimeEqual() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/crypto",
		Name:    "_crypto_constanttimeequal",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeConstantTimeEqualType,
		Impl:    constantTimeEqualImpl,
		Metadata: &BuiltinMetadata{
			Description: "Constant-time string comparison to prevent timing attacks",
			Params: []ParamDoc{
				{Name: "a", Description: "First string"},
				{Name: "b", Description: "Second string"},
			},
			Returns: "true if strings are equal, false otherwise",
			Examples: []Example{
				{Code: `_crypto_constanttimeequal("abc", "abc")`, Description: "Returns true"},
				{Code: `_crypto_constanttimeequal("abc", "def")`, Description: "Returns false"},
			},
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"crypto", "comparison", "timing-safe", "security"},
			Category:  "crypto",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _crypto_constanttimeequal: %v", err))
	}
}

func makeConstantTimeEqualType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
}

func constantTimeEqualImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	aVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_constanttimeequal: expected String for a, got %T", args[0])
	}
	bVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_constanttimeequal: expected String for b, got %T", args[1])
	}
	equal := subtle.ConstantTimeCompare([]byte(aVal.Value), []byte(bVal.Value)) == 1
	return &eval.BoolValue{Value: equal}, nil
}
