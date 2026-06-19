package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
)

func TestCheckSecretApprovalToken(t *testing.T) {
	signer, err := approvaltoken.NewSigner([]byte("server-test-key-0123456789"))
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	s := &Server{}
	s.SetSecretApprovalAuth(signer)

	tok, err := signer.Mint("appr-1", "approve", time.Hour)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	// Valid token for matching id+action.
	r := httptest.NewRequest("POST", "/api/approvals/appr-1/approve?token="+tok, nil)
	handled, ok := s.checkSecretApprovalToken(r, "appr-1", "approve")
	if !handled || !ok {
		t.Fatalf("valid token: handled=%v ok=%v, want true,true", handled, ok)
	}

	// Replay of the same token is rejected (single-use).
	r2 := httptest.NewRequest("POST", "/api/approvals/appr-1/approve?token="+tok, nil)
	handled, ok = s.checkSecretApprovalToken(r2, "appr-1", "approve")
	if !handled || ok {
		t.Fatalf("replay: handled=%v ok=%v, want true,false", handled, ok)
	}

	// Token bound to approve must not authorize reject.
	tok2, _ := signer.Mint("appr-2", "approve", time.Hour)
	r3 := httptest.NewRequest("POST", "/api/approvals/appr-2/reject?token="+tok2, nil)
	handled, ok = s.checkSecretApprovalToken(r3, "appr-2", "reject")
	if !handled || ok {
		t.Fatalf("action mismatch: handled=%v ok=%v, want true,false", handled, ok)
	}
}

func TestCheckSecretApprovalToken_DisabledWhenNoSigner(t *testing.T) {
	s := &Server{} // no SetSecretApprovalAuth
	r := httptest.NewRequest("POST", "/api/approvals/x/approve?token=whatever", nil)
	handled, _ := s.checkSecretApprovalToken(r, "x", "approve")
	if handled {
		t.Fatal("token auth must be inert when no signer configured")
	}
}

func TestCheckSecretApprovalToken_NoTokenFallsThrough(t *testing.T) {
	signer, _ := approvaltoken.NewSigner([]byte("k0123456789abcdef"))
	s := &Server{}
	s.SetSecretApprovalAuth(signer)
	r := httptest.NewRequest("POST", "/api/approvals/x/approve", nil)
	handled, _ := s.checkSecretApprovalToken(r, "x", "approve")
	if handled {
		t.Fatal("absent token must fall through to existing auth (handled=false)")
	}
}
