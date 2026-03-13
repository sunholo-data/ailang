package coordinator

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
)

// kmsEncryptPrefix marks a value as KMS-encrypted. Values with this prefix
// must be decrypted by the agent executor before use.
const kmsEncryptPrefix = "ENC:"

// KMSEncrypter encrypts API keys using Cloud KMS before they are passed
// as Cloud Run Job env var overrides. This prevents plaintext keys from
// appearing in Cloud Audit Logs.
//
// M-CLOUD-DUAL-AUTH: Coordinator SA has roles/cloudkms.cryptoKeyEncrypter
// (encrypt only — cannot read back user keys).
type KMSEncrypter struct {
	keyName string // Full KMS key resource name
}

// NewKMSEncrypter creates an encrypter using the AILANG_KMS_KEY env var.
// Returns nil if the env var is not set (local dev — plaintext passthrough).
func NewKMSEncrypter() *KMSEncrypter {
	keyName := os.Getenv("AILANG_KMS_KEY")
	if keyName == "" {
		return nil
	}
	return &KMSEncrypter{keyName: keyName}
}

// Encrypt encrypts a plaintext string with Cloud KMS.
// Returns "ENC:" + base64(ciphertext).
func (e *KMSEncrypter) Encrypt(ctx context.Context, plaintext string) (string, error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return "", fmt.Errorf("kms: failed to create client: %w", err)
	}
	defer client.Close()

	resp, err := client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      e.keyName,
		Plaintext: []byte(plaintext),
	})
	if err != nil {
		return "", fmt.Errorf("kms: encrypt failed: %w", err)
	}

	return kmsEncryptPrefix + base64.StdEncoding.EncodeToString(resp.Ciphertext), nil
}
