package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// KeyProtector wraps and unwraps the per-blob data encryption key (DEK). It is
// the seam that keeps the key-management backend out of this package: the
// durable backend (cloud KMS, HSM, local key file) is an explicitly deferred
// decision in the design document, and nothing here picks one.
//
// Implementations must be safe for concurrent use.
type KeyProtector interface {
	// WrapKey encrypts a freshly generated DEK and returns the wrapped form
	// together with the identifier of the key-encryption key that protected it.
	// The wrapped bytes are stored beside the ciphertext and carry no secret on
	// their own.
	WrapKey(ctx context.Context, dek []byte) (wrapped []byte, keyID string, err error)

	// UnwrapKey recovers a DEK. It must fail when keyID does not name the key
	// it holds, so that a substituted identifier cannot silently select a
	// different key.
	UnwrapKey(ctx context.Context, wrapped []byte, keyID string) ([]byte, error)
}

const (
	// envelopeMagic prefixes every serialized envelope. It is domain separation,
	// not security: it makes a non-envelope blob fail at parse rather than at
	// decrypt.
	envelopeMagic = "AILANGBAE"

	// envelopeVersion is the wire version of the serialized envelope. It is
	// authenticated, so a downgrade attempt fails closed.
	envelopeVersion = 1

	// dekSize selects AES-256.
	dekSize = 32

	// envelopeHeaderSize is magic + version + the four length prefixes.
	envelopeHeaderSize = len(envelopeMagic) + 1 + 2 + 2 + 1 + 4
)

// SealedEnvelope is canonical profile material at rest: AES-256-GCM ciphertext
// under a per-blob DEK, with that DEK wrapped by a KeyProtector.
//
// It follows the SensitiveProfileMaterial precedent — no exported fields, every
// presentation redacts, and one explicit extraction API (Bytes, for persistence).
// The ciphertext is not plaintext, but the repo treats sealed browser state as
// credential-grade: "sealed" and "ciphertext" are both redaction markers in
// browser.SanitizeDiagnostics, and a printed blob invites offline attack.
type SealedEnvelope struct {
	version    int
	keyID      string
	wrappedKey []byte
	nonce      []byte
	ciphertext []byte
}

// KeyID names the key-encryption key that protected this envelope's DEK. It is
// an operator-facing identifier and is safe to log.
func (e SealedEnvelope) KeyID() string { return e.keyID }

// Nonce returns a copy of the GCM nonce. A GCM nonce is public by construction;
// it is exposed so that callers and tests can verify per-blob freshness.
func (e SealedEnvelope) Nonce() []byte { return append([]byte(nil), e.nonce...) }

func (e SealedEnvelope) Empty() bool { return len(e.ciphertext) == 0 }

// rawCiphertext is the unexported accessor used by redaction tests.
func (e SealedEnvelope) rawCiphertext() []byte { return e.ciphertext }

func (e SealedEnvelope) String() string {
	return fmt.Sprintf("browser auth sealed envelope %s (v%d, key %s)", Redacted, e.version, e.keyID)
}

func (e SealedEnvelope) GoString() string { return e.String() }

func (e SealedEnvelope) Error() string { return e.String() }

func (e SealedEnvelope) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Version    int    `json:"version"`
		KeyID      string `json:"key_id"`
		Ciphertext string `json:"ciphertext"`
	}{Version: e.version, KeyID: e.keyID, Ciphertext: Redacted})
}

// additionalData binds the envelope header into the GCM authentication tag, so
// a swapped key identifier or a downgraded version is detected as tampering
// rather than merely failing to decrypt.
func (e SealedEnvelope) additionalData() []byte {
	out := make([]byte, 0, len(envelopeMagic)+1+len(e.keyID))
	out = append(out, envelopeMagic...)
	out = append(out, byte(e.version))
	out = append(out, e.keyID...)
	return out
}

// Bytes serializes the envelope for durable storage. The layout is fixed and
// length-prefixed so that any truncation or corruption of a length field fails
// at parse instead of producing a short read.
func (e SealedEnvelope) Bytes() []byte {
	out := make([]byte, 0, envelopeHeaderSize+len(e.keyID)+len(e.wrappedKey)+len(e.nonce)+len(e.ciphertext))
	out = append(out, envelopeMagic...)
	out = append(out, byte(e.version))
	out = binary.BigEndian.AppendUint16(out, uint16(len(e.keyID)))
	out = append(out, e.keyID...)
	out = binary.BigEndian.AppendUint16(out, uint16(len(e.wrappedKey)))
	out = append(out, e.wrappedKey...)
	out = append(out, byte(len(e.nonce)))
	out = append(out, e.nonce...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(e.ciphertext)))
	out = append(out, e.ciphertext...)
	return out
}

// envelopeParseFailure keeps every parse rejection in one category. The reason
// is an operator-safe token; it never echoes bytes.
func envelopeParseFailure(reason string) error {
	return NewFailureReason(FailureMaterializeFailed, "parse sealed envelope", reason)
}

// ParseSealedEnvelope decodes Bytes. Every malformed input fails closed as
// FailureMaterializeFailed; nothing is returned on failure.
func ParseSealedEnvelope(raw []byte) (SealedEnvelope, error) {
	if len(raw) < envelopeHeaderSize {
		return SealedEnvelope{}, envelopeParseFailure("truncated header")
	}
	if string(raw[:len(envelopeMagic)]) != envelopeMagic {
		return SealedEnvelope{}, envelopeParseFailure("bad magic")
	}
	cursor := len(envelopeMagic)

	version := int(raw[cursor])
	cursor++
	if version != envelopeVersion {
		return SealedEnvelope{}, envelopeParseFailure("unsupported version")
	}

	keyID, cursor, err := takeBytes16(raw, cursor, "key id")
	if err != nil {
		return SealedEnvelope{}, err
	}
	wrappedKey, cursor, err := takeBytes16(raw, cursor, "wrapped key")
	if err != nil {
		return SealedEnvelope{}, err
	}

	if cursor >= len(raw) {
		return SealedEnvelope{}, envelopeParseFailure("truncated nonce length")
	}
	nonceLength := int(raw[cursor])
	cursor++
	if cursor+nonceLength > len(raw) {
		return SealedEnvelope{}, envelopeParseFailure("truncated nonce")
	}
	nonce := raw[cursor : cursor+nonceLength]
	cursor += nonceLength

	if cursor+4 > len(raw) {
		return SealedEnvelope{}, envelopeParseFailure("truncated ciphertext length")
	}
	ciphertextLength := int(binary.BigEndian.Uint32(raw[cursor : cursor+4]))
	cursor += 4
	if cursor+ciphertextLength != len(raw) {
		// Strict equality, not >=: trailing bytes mean the blob is not what it
		// claims to be, and accepting them would be a partial read.
		return SealedEnvelope{}, envelopeParseFailure("ciphertext length mismatch")
	}
	if ciphertextLength == 0 {
		return SealedEnvelope{}, envelopeParseFailure("empty ciphertext")
	}

	return SealedEnvelope{
		version:    version,
		keyID:      string(keyID),
		wrappedKey: append([]byte(nil), wrappedKey...),
		nonce:      append([]byte(nil), nonce...),
		ciphertext: append([]byte(nil), raw[cursor:cursor+ciphertextLength]...),
	}, nil
}

func takeBytes16(raw []byte, cursor int, what string) ([]byte, int, error) {
	if cursor+2 > len(raw) {
		return nil, 0, envelopeParseFailure("truncated " + what + " length")
	}
	length := int(binary.BigEndian.Uint16(raw[cursor : cursor+2]))
	cursor += 2
	if cursor+length > len(raw) {
		return nil, 0, envelopeParseFailure("truncated " + what)
	}
	return raw[cursor : cursor+length], cursor + length, nil
}

// Seal encrypts plaintext under a freshly generated DEK and a fresh nonce.
//
// A new DEK and a new nonce are drawn for EVERY call. Nothing is derived from
// the plaintext, the profile, or a counter: reusing a (key, nonce) pair under
// GCM discloses the XOR of the two plaintexts and destroys the authentication
// guarantee, so freshness is not an optimization to be skipped.
func Seal(ctx context.Context, protector KeyProtector, plaintext []byte) (SealedEnvelope, error) {
	if protector == nil {
		return SealedEnvelope{}, NewFailureReason(FailureMaterializeFailed, "seal", "no key protector")
	}
	if err := ctx.Err(); err != nil {
		return SealedEnvelope{}, NewFailureReason(FailureMaterializeFailed, "seal", "context ended")
	}

	dek := make([]byte, dekSize)
	if _, err := rand.Read(dek); err != nil {
		return SealedEnvelope{}, NewFailureReason(FailureMaterializeFailed, "seal", "entropy unavailable")
	}
	wrappedKey, keyID, err := protector.WrapKey(ctx, dek)
	if err != nil {
		// The protector's error may name an internal endpoint or key path.
		return SealedEnvelope{}, NewFailureReason(FailureMaterializeFailed, "seal", "wrap key failed")
	}

	gcm, err := newGCM(dek)
	if err != nil {
		return SealedEnvelope{}, NewFailureReason(FailureMaterializeFailed, "seal", "cipher unavailable")
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return SealedEnvelope{}, NewFailureReason(FailureMaterializeFailed, "seal", "entropy unavailable")
	}

	envelope := SealedEnvelope{version: envelopeVersion, keyID: keyID, wrappedKey: wrappedKey, nonce: nonce}
	envelope.ciphertext = gcm.Seal(nil, nonce, plaintext, envelope.additionalData())
	return envelope, nil
}

// Open decrypts a sealed envelope. Every failure — a wrong key, a tampered
// ciphertext, a substituted key identifier, an unavailable protector — returns
// FailureMaterializeFailed and a nil plaintext. GCM verifies the tag before
// releasing any plaintext, so a failed Open is never a partial read.
func Open(ctx context.Context, protector KeyProtector, sealed SealedEnvelope) ([]byte, error) {
	if protector == nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "no key protector")
	}
	if err := ctx.Err(); err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "context ended")
	}
	if sealed.Empty() {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "empty envelope")
	}

	dek, err := protector.UnwrapKey(ctx, sealed.wrappedKey, sealed.keyID)
	if err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "unwrap key failed")
	}
	gcm, err := newGCM(dek)
	if err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "cipher unavailable")
	}
	if len(sealed.nonce) != gcm.NonceSize() {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "nonce size mismatch")
	}
	plaintext, err := gcm.Open(nil, sealed.nonce, sealed.ciphertext, sealed.additionalData())
	if err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "open", "authentication failed")
	}
	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// StaticKeyProtector holds a key-encryption key in process memory and wraps
// DEKs with AES-256-GCM.
//
// It is the reference implementation: it is what tests use, and what
// NewLocalFileKeyProtector returns. It is NOT a key-management system — the KEK
// lives in this process's memory (and, for the local file variant, on this
// host's disk), so it provides no envelope-key rotation, no access audit, and
// no protection against an attacker who can read the process or the file. A
// KMS-backed KeyProtector implements the same interface; choosing that backend
// is a deferred decision in the design document.
type StaticKeyProtector struct {
	id  string
	kek []byte
}

func NewStaticKeyProtector(id string, kek []byte) (*StaticKeyProtector, error) {
	if id == "" {
		return nil, fmt.Errorf("key protector needs a key id")
	}
	if len(kek) != dekSize {
		return nil, fmt.Errorf("key-encryption key must be %d bytes, got %d", dekSize, len(kek))
	}
	return &StaticKeyProtector{id: id, kek: append([]byte(nil), kek...)}, nil
}

// NewRandomStaticKeyProtector generates an ephemeral KEK. Anything sealed with
// it is unrecoverable once the process exits, which is what tests want and what
// production does not.
func NewRandomStaticKeyProtector(id string) (*StaticKeyProtector, error) {
	kek := make([]byte, dekSize)
	if _, err := rand.Read(kek); err != nil {
		return nil, fmt.Errorf("generate key-encryption key: %w", err)
	}
	return NewStaticKeyProtector(id, kek)
}

// NewLocalFileKeyProtector loads a KEK from path, creating it on first use with
// owner-only permissions.
//
// LOCAL DEVELOPMENT AND SINGLE-HOST OPERATION ONLY. The key sits in a file next
// to the data it protects, so anyone who can read the host can read the
// profiles. Use a KMS-backed KeyProtector for anything shared or durable.
func NewLocalFileKeyProtector(path string) (*StaticKeyProtector, error) {
	if path == "" {
		return nil, fmt.Errorf("local key protector needs a path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}

	kek, err := os.ReadFile(path) //nolint:gosec // operator-supplied key path
	switch {
	case err == nil:
		if len(kek) != dekSize {
			return nil, fmt.Errorf("key file %s holds %d bytes, want %d", filepath.Base(path), len(kek), dekSize)
		}
	case os.IsNotExist(err):
		kek = make([]byte, dekSize)
		if _, err := rand.Read(kek); err != nil {
			return nil, fmt.Errorf("generate key-encryption key: %w", err)
		}
		// O_EXCL so two processes racing to bootstrap cannot both win and leave
		// half the profiles sealed under a discarded key.
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, fmt.Errorf("create key file: %w", err)
		}
		if _, err := file.Write(kek); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("write key file: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close key file: %w", err)
		}
	default:
		return nil, fmt.Errorf("read key file: %w", err)
	}

	return NewStaticKeyProtector("local-file", kek)
}

func (p *StaticKeyProtector) WrapKey(_ context.Context, dek []byte) ([]byte, string, error) {
	gcm, err := newGCM(p.kek)
	if err != nil {
		return nil, "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	// The wrapped form is nonce || ciphertext; the key id is authenticated as
	// additional data so it cannot be swapped for another protector's id.
	return gcm.Seal(nonce, nonce, dek, []byte(p.id)), p.id, nil
}

func (p *StaticKeyProtector) UnwrapKey(_ context.Context, wrapped []byte, keyID string) ([]byte, error) {
	if keyID != p.id {
		return nil, fmt.Errorf("key id %q is not held by this protector", keyID)
	}
	gcm, err := newGCM(p.kek)
	if err != nil {
		return nil, err
	}
	if len(wrapped) < gcm.NonceSize() {
		return nil, fmt.Errorf("wrapped key is truncated")
	}
	nonce, ciphertext := wrapped[:gcm.NonceSize()], wrapped[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, []byte(p.id))
}
