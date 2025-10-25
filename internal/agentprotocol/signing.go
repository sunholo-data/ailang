package agentprotocol

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SigningKey represents a key for HMAC message signing.
type SigningKey struct {
	// KID is the key ID (for key rotation)
	KID string `json:"kid"`

	// Algorithm is the signing algorithm (e.g., "hmac-sha256")
	Algorithm string `json:"algorithm"`

	// Secret is the raw key bytes (base64 encoded in JSON)
	Secret []byte `json:"secret"`

	// CreatedAt is when the key was generated
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the key expires (optional)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// MessageSigner handles HMAC signing and verification of messages.
type MessageSigner struct {
	stateDir string
	key      *SigningKey
}

// NewMessageSigner creates a new message signer.
//
// It loads the signing key from the state directory, or generates a new one
// if none exists.
func NewMessageSigner(stateDir string) (*MessageSigner, error) {
	signer := &MessageSigner{
		stateDir: stateDir,
	}

	// Try to load existing key
	key, err := signer.loadKey()
	if err != nil {
		// Generate new key if none exists
		key, err = signer.generateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate signing key: %w", err)
		}

		// Save the new key
		if err := signer.saveKey(key); err != nil {
			return nil, fmt.Errorf("failed to save signing key: %w", err)
		}
	}

	signer.key = key
	return signer, nil
}

// SignMessage adds an HMAC signature to a message envelope.
//
// The signature is computed over the canonical JSON representation of the
// envelope (excluding the signature fields themselves).
func (s *MessageSigner) SignMessage(env *Envelope) error {
	// Build canonical representation (without signature fields)
	canonical, err := s.canonicalRepresentation(env)
	if err != nil {
		return fmt.Errorf("failed to build canonical representation: %w", err)
	}

	// Compute HMAC
	signature := s.computeHMAC(canonical)

	// Add signature fields to envelope
	env.SignatureAlg = s.key.Algorithm
	env.KID = s.key.KID
	env.Signature = signature

	return nil
}

// VerifyMessage verifies the HMAC signature of a message envelope.
//
// Returns nil if the signature is valid, an error otherwise.
func (s *MessageSigner) VerifyMessage(env *Envelope) error {
	// Check signature fields are present
	if env.Signature == "" {
		return fmt.Errorf("message has no signature")
	}
	if env.KID == "" {
		return fmt.Errorf("message has no key ID")
	}
	if env.SignatureAlg == "" {
		return fmt.Errorf("message has no signature algorithm")
	}

	// Check key ID matches
	if env.KID != s.key.KID {
		return fmt.Errorf("key ID mismatch: expected %s, got %s", s.key.KID, env.KID)
	}

	// Check algorithm matches
	if env.SignatureAlg != s.key.Algorithm {
		return fmt.Errorf("algorithm mismatch: expected %s, got %s", s.key.Algorithm, env.SignatureAlg)
	}

	// Build canonical representation (without signature fields)
	canonical, err := s.canonicalRepresentation(env)
	if err != nil {
		return fmt.Errorf("failed to build canonical representation: %w", err)
	}

	// Compute expected HMAC
	expectedSignature := s.computeHMAC(canonical)

	// Compare signatures (constant time comparison)
	if !hmac.Equal([]byte(env.Signature), []byte(expectedSignature)) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// canonicalRepresentation builds a deterministic JSON representation of the
// envelope for signing/verification.
//
// It excludes the signature fields (SignatureAlg, KID, Signature) to avoid
// circular dependencies.
func (s *MessageSigner) canonicalRepresentation(env *Envelope) ([]byte, error) {
	// Create a copy of the envelope without signature fields
	canonical := map[string]interface{}{
		"protocol_version": env.ProtocolVersion,
		"schema_version":   env.SchemaVersion,
		"message_id":       env.MessageID,
		"correlation_id":   env.CorrelationID,
		"trace_id":         env.TraceID,
		"timestamp":        env.Timestamp,
		"ttl_seconds":      env.TTLSeconds,
		"from_agent":       env.FromAgent,
		"to_agent":         env.ToAgent,
		"message_type":     env.MessageType,
		"retries":          env.Retries,
		"payload_schema":   env.PayloadSchema,
		"payload":          env.Payload,
	}

	// Add optional fields if present
	if env.ParentMessageID != nil {
		canonical["parent_message_id"] = *env.ParentMessageID
	}
	if env.Deadline != "" {
		canonical["deadline"] = env.Deadline
	}
	if len(env.DeclaredEffects) > 0 {
		canonical["declared_effects"] = env.DeclaredEffects
	}

	// Marshal to JSON (deterministic order via sorted keys)
	return json.Marshal(canonical)
}

// computeHMAC computes the HMAC-SHA256 signature of data.
func (s *MessageSigner) computeHMAC(data []byte) string {
	h := hmac.New(sha256.New, s.key.Secret)
	h.Write(data)
	signature := h.Sum(nil)
	return hex.EncodeToString(signature)
}

// generateKey generates a new signing key.
func (s *MessageSigner) generateKey() (*SigningKey, error) {
	// Generate random key ID
	kidBytes := make([]byte, 8)
	if _, err := rand.Read(kidBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key ID: %w", err)
	}
	kid := hex.EncodeToString(kidBytes)

	// Generate random secret (32 bytes = 256 bits)
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	return &SigningKey{
		KID:       "key-" + kid,
		Algorithm: "hmac-sha256",
		Secret:    secret,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// loadKey loads the signing key from disk.
func (s *MessageSigner) loadKey() (*SigningKey, error) {
	// Try environment variable first
	if keyEnv := os.Getenv("AILANG_SIGNING_KEY"); keyEnv != "" {
		return s.parseKeyFromEnv(keyEnv)
	}

	// Try loading from file
	keyPath := filepath.Join(s.stateDir, "signing_key.json")
	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("signing key not found")
		}
		return nil, fmt.Errorf("failed to read signing key: %w", err)
	}

	var key SigningKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signing key: %w", err)
	}

	return &key, nil
}

// saveKey saves the signing key to disk.
func (s *MessageSigner) saveKey(key *SigningKey) error {
	keyPath := filepath.Join(s.stateDir, "signing_key.json")

	// Ensure state directory exists
	if err := os.MkdirAll(s.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal signing key: %w", err)
	}

	// Write to file (with restricted permissions)
	if err := os.WriteFile(keyPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write signing key: %w", err)
	}

	return nil
}

// parseKeyFromEnv parses a signing key from the AILANG_SIGNING_KEY env var.
//
// Expected format: base64-encoded JSON
func (s *MessageSigner) parseKeyFromEnv(keyEnv string) (*SigningKey, error) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(keyEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AILANG_SIGNING_KEY: %w", err)
	}

	// Unmarshal JSON
	var key SigningKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AILANG_SIGNING_KEY: %w", err)
	}

	return &key, nil
}

// GetKeyID returns the current key ID.
func (s *MessageSigner) GetKeyID() string {
	return s.key.KID
}

// RotateKey generates a new signing key and saves it.
//
// This invalidates all messages signed with the old key.
func (s *MessageSigner) RotateKey() error {
	newKey, err := s.generateKey()
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	if err := s.saveKey(newKey); err != nil {
		return fmt.Errorf("failed to save new key: %w", err)
	}

	s.key = newKey
	return nil
}
