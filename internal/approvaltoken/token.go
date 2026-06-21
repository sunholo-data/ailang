// Package approvaltoken mints and verifies short-lived, single-use,
// HMAC-signed tokens that authorize a single approval action (approve|reject)
// for a single approval ID — without Google IAM.
//
// This is what makes the iPhone notification's Approve/Deny action buttons safe:
// each button URL carries a token bound to {approvalID, action, expiry, nonce}.
// Anyone who later sees the URL cannot reuse it (single-use) or use it past its
// TTL (expiry), and cannot forge one without the signing key.
package approvaltoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Errors returned by Verify. Callers should test with errors.Is.
var (
	ErrMalformed = errors.New("approval token malformed")
	ErrBadSig    = errors.New("approval token signature invalid")
	ErrExpired   = errors.New("approval token expired")
	ErrReused    = errors.New("approval token already used")
)

// Claims are the verified contents of a token.
type Claims struct {
	ApprovalID string
	Action     string // "approve" or "reject"
	Nonce      string
	ExpiresAt  time.Time
}

// nowFunc is overridable in tests.
var nowFunc = time.Now

// Signer mints and verifies tokens with a fixed HMAC key.
type Signer struct {
	key []byte
}

// NewSigner returns a Signer using key (must be non-empty).
func NewSigner(key []byte) (*Signer, error) {
	if len(key) == 0 {
		return nil, errors.New("approvaltoken: empty signing key")
	}
	return &Signer{key: append([]byte(nil), key...)}, nil
}

// Mint creates a token authorizing action on approvalID, valid for ttl.
func (s *Signer) Mint(approvalID, action string, ttl time.Duration) (string, error) {
	if approvalID == "" || (action != "approve" && action != "reject") {
		return "", fmt.Errorf("approvaltoken: invalid mint args (id=%q action=%q)", approvalID, action)
	}
	nonceBytes := make([]byte, 12)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("approvaltoken: nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)
	exp := nowFunc().Add(ttl).Unix()
	payload := strings.Join([]string{approvalID, action, nonce, strconv.FormatInt(exp, 10)}, "|")
	sig := s.sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig)), nil
}

// Verify checks the token's signature and expiry and returns its claims.
// It does NOT enforce single-use; pair it with a SingleUseGuard for that.
func (s *Signer) Verify(token string) (*Claims, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("%w: base64: %v", ErrMalformed, err)
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 5 {
		return nil, fmt.Errorf("%w: expected 5 fields, got %d", ErrMalformed, len(parts))
	}
	approvalID, action, nonce, expStr, sig := parts[0], parts[1], parts[2], parts[3], parts[4]

	payload := strings.Join([]string{approvalID, action, nonce, expStr}, "|")
	if !hmac.Equal([]byte(sig), []byte(s.sign(payload))) {
		return nil, ErrBadSig
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: expiry: %v", ErrMalformed, err)
	}
	expiresAt := time.Unix(exp, 0)
	if nowFunc().After(expiresAt) {
		return nil, ErrExpired
	}
	return &Claims{ApprovalID: approvalID, Action: action, Nonce: nonce, ExpiresAt: expiresAt}, nil
}

func (s *Signer) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
