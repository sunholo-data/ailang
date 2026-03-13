package claude

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
)

// kmsEncryptPrefix marks a value as KMS-encrypted.
const kmsEncryptPrefix = "ENC:"

// decryptAPIKeyIfNeeded checks if ANTHROPIC_API_KEY has the "ENC:" prefix
// and decrypts it using Cloud KMS. If the value is not encrypted (no prefix),
// it's a no-op (supports local dev and backwards compatibility).
//
// M-CLOUD-DUAL-AUTH: Agent SA has roles/cloudkms.cryptoKeyDecrypter
// (decrypt only — cannot forge encrypted keys).
func decryptAPIKeyIfNeeded(ctx context.Context) error {
	encrypted := os.Getenv("ANTHROPIC_API_KEY")
	if encrypted == "" || !strings.HasPrefix(encrypted, kmsEncryptPrefix) {
		return nil // Not encrypted or not set — passthrough
	}

	kmsKeyName := os.Getenv("AILANG_KMS_KEY")
	if kmsKeyName == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is KMS-encrypted but AILANG_KMS_KEY not set")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, kmsEncryptPrefix))
	if err != nil {
		return fmt.Errorf("kms: failed to decode encrypted API key: %w", err)
	}

	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("kms: failed to create client: %w", err)
	}
	defer client.Close()

	resp, err := client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       kmsKeyName,
		Ciphertext: ciphertext,
	})
	if err != nil {
		return fmt.Errorf("kms: decrypt failed: %w", err)
	}

	os.Setenv("ANTHROPIC_API_KEY", string(resp.Plaintext))
	fmt.Fprintf(os.Stderr, "claude-auth: decrypted ANTHROPIC_API_KEY via KMS\n")
	return nil
}
