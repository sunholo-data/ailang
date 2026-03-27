package builtins

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// testRSAKeyPair generates a 2048-bit RSA key pair for testing.
func testRSAKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	return key, string(pemBlock)
}

// testSelfSignedCert creates a self-signed X.509 certificate for testing.
func testSelfSignedCert(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	return string(pemBlock)
}

// signMessage signs a message with RSA-SHA256 PKCS#1 v1.5.
func signMessage(t *testing.T, key *rsa.PrivateKey, msg []byte) []byte {
	t.Helper()
	hashed := sha256.Sum256(msg)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	return sig
}

func TestRSAVerify_ValidSignature(t *testing.T) {
	key, pubPEM := testRSAKeyPair(t)
	msg := []byte("hello world")
	sig := signMessage(t, key, msg)

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: msg},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: pubPEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s: %v", tv.CtorName, tv.Fields)
	}
	bv, ok := tv.Fields[0].(*eval.BoolValue)
	if !ok || !bv.Value {
		t.Fatalf("expected Ok(true), got Ok(%v)", tv.Fields[0])
	}
}

func TestRSAVerify_InvalidSignature(t *testing.T) {
	key, pubPEM := testRSAKeyPair(t)
	msg := []byte("hello world")
	sig := signMessage(t, key, msg)

	// Tamper with the signature
	sig[0] ^= 0xff

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: msg},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: pubPEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok(false), got %s", tv.CtorName)
	}
	bv := tv.Fields[0].(*eval.BoolValue)
	if bv.Value {
		t.Fatal("expected Ok(false) for tampered signature, got Ok(true)")
	}
}

func TestRSAVerify_WrongMessage(t *testing.T) {
	key, pubPEM := testRSAKeyPair(t)
	msg := []byte("hello world")
	sig := signMessage(t, key, msg)

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: []byte("different message")},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: pubPEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok(false), got %s", tv.CtorName)
	}
	bv := tv.Fields[0].(*eval.BoolValue)
	if bv.Value {
		t.Fatal("expected Ok(false) for wrong message, got Ok(true)")
	}
}

func TestRSAVerify_X509Certificate(t *testing.T) {
	key, _ := testRSAKeyPair(t)
	certPEM := testSelfSignedCert(t, key)
	msg := []byte("verify with certificate")
	sig := signMessage(t, key, msg)

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: msg},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: certPEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s: %v", tv.CtorName, tv.Fields)
	}
	bv := tv.Fields[0].(*eval.BoolValue)
	if !bv.Value {
		t.Fatal("expected Ok(true) for valid cert signature")
	}
}

func TestRSAVerify_PKCS1PublicKey(t *testing.T) {
	key, _ := testRSAKeyPair(t)
	// Encode as PKCS#1
	pkcs1Bytes := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	pkcs1PEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pkcs1Bytes,
	}))
	msg := []byte("pkcs1 test")
	sig := signMessage(t, key, msg)

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: msg},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: pkcs1PEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s: %v", tv.CtorName, tv.Fields)
	}
	bv := tv.Fields[0].(*eval.BoolValue)
	if !bv.Value {
		t.Fatal("expected Ok(true) for valid PKCS#1 key signature")
	}
}

func TestRSAVerify_InvalidPEM(t *testing.T) {
	msg := []byte("test")
	sig := []byte("fake-sig")

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: msg},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: "not a pem"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for invalid PEM, got %s", tv.CtorName)
	}
	errMsg := tv.Fields[0].(*eval.StringValue).Value
	if errMsg != "invalid PEM: no PEM block found" {
		t.Fatalf("unexpected error message: %s", errMsg)
	}
}

func TestRSAVerify_UnsupportedPEMType(t *testing.T) {
	badPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: []byte("fake"),
	}))

	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: []byte("test")},
		&eval.BytesValue{Value: []byte("sig")},
		&eval.StringValue{Value: badPEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for unsupported PEM type, got %s", tv.CtorName)
	}
}

func TestRSAVerify_WrongKeyForSignature(t *testing.T) {
	key1, _ := testRSAKeyPair(t)
	_, pub2PEM := testRSAKeyPair(t)

	msg := []byte("signed with key1")
	sig := signMessage(t, key1, msg)

	// Verify with key2 — should be Ok(false)
	result, err := rsaVerifyImpl(nil, []eval.Value{
		&eval.BytesValue{Value: msg},
		&eval.BytesValue{Value: sig},
		&eval.StringValue{Value: pub2PEM},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Ok" {
		t.Fatalf("expected Ok(false), got %s", tv.CtorName)
	}
	bv := tv.Fields[0].(*eval.BoolValue)
	if bv.Value {
		t.Fatal("expected Ok(false) when verifying with wrong key")
	}
}

func TestRSAVerify_Deterministic(t *testing.T) {
	key, pubPEM := testRSAKeyPair(t)
	msg := []byte("determinism test")
	sig := signMessage(t, key, msg)

	// Run 20 times to check determinism (per sprint checklist)
	for i := 0; i < 20; i++ {
		result, err := rsaVerifyImpl(nil, []eval.Value{
			&eval.BytesValue{Value: msg},
			&eval.BytesValue{Value: sig},
			&eval.StringValue{Value: pubPEM},
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		tv := result.(*eval.TaggedValue)
		if tv.CtorName != "Ok" {
			t.Fatalf("run %d: expected Ok, got %s", i, tv.CtorName)
		}
		bv := tv.Fields[0].(*eval.BoolValue)
		if !bv.Value {
			t.Fatalf("run %d: expected Ok(true)", i)
		}
	}
}
