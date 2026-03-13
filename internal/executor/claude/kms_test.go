package claude

import (
	"context"
	"os"
	"testing"
)

func TestDecryptAPIKeyIfNeeded_NotSet(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")

	err := decryptAPIKeyIfNeeded(context.Background())
	if err != nil {
		t.Fatalf("expected no error when key not set, got: %v", err)
	}
}

func TestDecryptAPIKeyIfNeeded_PlaintextPassthrough(t *testing.T) {
	// Non-ENC: prefixed values should pass through unchanged
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key-123")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	err := decryptAPIKeyIfNeeded(context.Background())
	if err != nil {
		t.Fatalf("expected no error for plaintext key, got: %v", err)
	}

	// Key should be unchanged
	if got := os.Getenv("ANTHROPIC_API_KEY"); got != "sk-ant-test-key-123" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want unchanged plaintext", got)
	}
}

func TestDecryptAPIKeyIfNeeded_EncryptedNoKMSKey(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "ENC:c29tZWVuY3J5cHRlZGRhdGE=")
	os.Unsetenv("AILANG_KMS_KEY")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	err := decryptAPIKeyIfNeeded(context.Background())
	if err == nil {
		t.Fatal("expected error when AILANG_KMS_KEY not set for encrypted key")
	}
	if got := err.Error(); got != "ANTHROPIC_API_KEY is KMS-encrypted but AILANG_KMS_KEY not set" {
		t.Errorf("error = %q, want missing KMS key error", got)
	}
}

func TestKMSEncryptPrefixConstant(t *testing.T) {
	if kmsEncryptPrefix != "ENC:" {
		t.Errorf("kmsEncryptPrefix = %q, want %q", kmsEncryptPrefix, "ENC:")
	}
}
