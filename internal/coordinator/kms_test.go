package coordinator

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func TestNewKMSEncrypterNoEnvVar(t *testing.T) {
	// Unset to ensure clean test
	os.Unsetenv("AILANG_KMS_KEY")

	enc := NewKMSEncrypter()
	if enc != nil {
		t.Fatal("expected nil encrypter when AILANG_KMS_KEY is not set")
	}
}

func TestNewKMSEncrypterWithEnvVar(t *testing.T) {
	os.Setenv("AILANG_KMS_KEY", "projects/test/locations/global/keyRings/test/cryptoKeys/test")
	defer os.Unsetenv("AILANG_KMS_KEY")

	enc := NewKMSEncrypter()
	if enc == nil {
		t.Fatal("expected non-nil encrypter when AILANG_KMS_KEY is set")
	}
	if enc.keyName != "projects/test/locations/global/keyRings/test/cryptoKeys/test" {
		t.Errorf("keyName = %q, want full resource name", enc.keyName)
	}
}

func TestKMSEncryptPrefix(t *testing.T) {
	// Verify the prefix constant is correct
	if kmsEncryptPrefix != "ENC:" {
		t.Errorf("kmsEncryptPrefix = %q, want %q", kmsEncryptPrefix, "ENC:")
	}
}

func TestKMSEncryptedValueFormat(t *testing.T) {
	// Verify that a properly encrypted value can be decoded
	fakePayload := []byte("encrypted-data-here")
	encoded := kmsEncryptPrefix + base64.StdEncoding.EncodeToString(fakePayload)

	if !strings.HasPrefix(encoded, "ENC:") {
		t.Errorf("encoded value should start with ENC:")
	}

	// Strip prefix and decode
	b64 := strings.TrimPrefix(encoded, "ENC:")
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if string(decoded) != "encrypted-data-here" {
		t.Errorf("decoded = %q, want %q", string(decoded), "encrypted-data-here")
	}
}
