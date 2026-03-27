package builtins

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerRSAVerifyPKCS1v15()
}

// registerRSAVerifyPKCS1v15 registers _crypto_rsa_verify_pkcs1v15:
// (bytes, bytes, string) -> Result[bool, string]
//
// Verifies an RSA PKCS#1 v1.5 signature with SHA-256.
// Args: message (bytes), signature (bytes), publicKeyPEM (string)
// The PEM can be:
//   - PKCS#8 public key ("BEGIN PUBLIC KEY")
//   - PKCS#1 public key ("BEGIN RSA PUBLIC KEY")
//   - X.509 certificate ("BEGIN CERTIFICATE") — extracts RSA public key
//
// Returns Ok(true) if valid, Ok(false) if invalid sig, Err(msg) if key parsing fails.
func registerRSAVerifyPKCS1v15() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/crypto",
		Name:    "_crypto_rsa_verify_pkcs1v15",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeRSAVerifyType,
		Impl:    rsaVerifyImpl,
		Metadata: &BuiltinMetadata{
			Description: "RSA PKCS#1 v1.5 signature verification with SHA-256",
			LongDesc:    "Verifies that the given signature is a valid RSA-SHA256 signature of the message using the provided PEM-encoded public key or X.509 certificate. Returns Ok(true) for valid signatures, Ok(false) for invalid signatures, and Err(message) for key parsing failures.",
			Params: []ParamDoc{
				{Name: "message", Description: "The original message bytes that were signed"},
				{Name: "signature", Description: "The RSA signature bytes to verify"},
				{Name: "publicKeyPEM", Description: "PEM-encoded RSA public key or X.509 certificate"},
			},
			Returns: "Result[bool, string]: Ok(true) if valid, Ok(false) if invalid, Err if key error",
			Examples: []Example{
				{Code: `_crypto_rsa_verify_pkcs1v15(msg, sig, pem)`, Description: "Verify RSA-SHA256 signature"},
			},
			Since:     "v0.9.5",
			Stability: StabilityStable,
			Tags:      []string{"crypto", "rsa", "signature", "verification", "jwt", "security"},
			Category:  "crypto",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _crypto_rsa_verify_pkcs1v15: %v", err))
	}
}

func makeRSAVerifyType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Bytes(), T.Bytes(), T.String()).
		Returns(T.App("Result", T.Bool(), T.String())).Build()
}

func rsaVerifyImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	msgVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_rsa_verify_pkcs1v15: expected Bytes for message, got %T", args[0])
	}
	sigVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_rsa_verify_pkcs1v15: expected Bytes for signature, got %T", args[1])
	}
	pemVal, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_crypto_rsa_verify_pkcs1v15: expected String for publicKeyPEM, got %T", args[2])
	}

	pubKey, err := parseRSAPublicKey(pemVal.Value)
	if err != nil {
		return rsaWrapErr(err.Error()), nil
	}

	hashed := sha256.Sum256(msgVal.Value)
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], sigVal.Value)
	if err != nil {
		// Invalid signature is a normal outcome, not an error
		return rsaWrapOk(false), nil
	}
	return rsaWrapOk(true), nil
}

// parseRSAPublicKey extracts an RSA public key from a PEM string.
// Supports CERTIFICATE, PUBLIC KEY (PKCS#8), and RSA PUBLIC KEY (PKCS#1).
func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM: no PEM block found")
	}

	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid certificate: %v", err)
		}
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("certificate does not contain RSA public key")
		}
		return rsaKey, nil

	case "PUBLIC KEY":
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid public key: %v", err)
		}
		rsaKey, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil

	case "RSA PUBLIC KEY":
		parsed, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid PKCS#1 public key: %v", err)
		}
		return parsed, nil

	default:
		return nil, fmt.Errorf("unsupported PEM type: %s", block.Type)
	}
}

// rsaWrapOk wraps a bool in Ok(bool) for Result[bool, string].
func rsaWrapOk(val bool) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{&eval.BoolValue{Value: val}},
	}
}

// rsaWrapErr wraps a string in Err(string) for Result[bool, string].
func rsaWrapErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}
