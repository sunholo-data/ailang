package approvaltoken

import (
	"errors"
	"testing"
	"time"
)

func newTestSigner(t *testing.T) *Signer {
	t.Helper()
	s, err := NewSigner([]byte("test-signing-key-0123456789"))
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return s
}

func TestMintVerify_RoundTrip(t *testing.T) {
	s := newTestSigner(t)
	tok, err := s.Mint("approval-1", "approve", time.Hour)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	claims, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.ApprovalID != "approval-1" || claims.Action != "approve" {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestVerify_TamperedSignatureRejected(t *testing.T) {
	s := newTestSigner(t)
	tok, _ := s.Mint("approval-1", "approve", time.Hour)
	// Flip a character.
	bad := tok[:len(tok)-1] + flip(tok[len(tok)-1])
	_, err := s.Verify(bad)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
	if !errors.Is(err, ErrBadSig) && !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrBadSig/ErrMalformed", err)
	}
}

func TestVerify_WrongKeyRejected(t *testing.T) {
	s := newTestSigner(t)
	tok, _ := s.Mint("approval-1", "approve", time.Hour)
	other, _ := NewSigner([]byte("a-totally-different-key-987654"))
	if _, err := other.Verify(tok); !errors.Is(err, ErrBadSig) {
		t.Fatalf("err = %v, want ErrBadSig", err)
	}
}

func TestVerify_ExpiredRejected(t *testing.T) {
	s := newTestSigner(t)
	tok, _ := s.Mint("approval-1", "approve", time.Hour)
	// Jump now past expiry.
	orig := nowFunc
	nowFunc = func() time.Time { return orig().Add(2 * time.Hour) }
	defer func() { nowFunc = orig }()
	if _, err := s.Verify(tok); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
}

func TestSingleUseGuard_RejectsReplay(t *testing.T) {
	s := newTestSigner(t)
	guard := NewSingleUseGuard()
	tok, _ := s.Mint("approval-1", "approve", time.Hour)
	claims, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := guard.Use(claims.Nonce); err != nil {
		t.Fatalf("first use should succeed: %v", err)
	}
	if err := guard.Use(claims.Nonce); !errors.Is(err, ErrReused) {
		t.Fatalf("second use err = %v, want ErrReused", err)
	}
}

func TestMint_RejectsBadAction(t *testing.T) {
	s := newTestSigner(t)
	if _, err := s.Mint("approval-1", "delete", time.Hour); err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func flip(b byte) string {
	if b == 'A' {
		return "B"
	}
	return "A"
}
