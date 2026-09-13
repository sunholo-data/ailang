package server

import (
	"errors"
	"net/http"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
)

// SetSecretApprovalAuth enables signed single-use token auth on the
// approve/reject endpoints (M-SECRET-EFFECT). Pass the same signer used to mint
// the iPhone action-button tokens. When never called, token auth stays disabled
// and the endpoints behave exactly as before.
func (s *Server) SetSecretApprovalAuth(signer *approvaltoken.Signer) {
	s.secretTokenSigner = signer
	s.secretTokenGuard = approvaltoken.NewSingleUseGuard()
}

// checkSecretApprovalToken authorizes an approve/reject request via a signed
// ?token= query parameter. It returns:
//   - handled=false when token auth is disabled or no token is present (the
//     caller falls back to the existing dashboard/IAM auth path);
//   - handled=true with ok reporting whether the token is valid for this exact
//     approvalID + action and has not been used before.
//
// A token that is present but invalid is handled=true, ok=false — the caller
// must reject it rather than fall through to another auth path.
func (s *Server) checkSecretApprovalToken(r *http.Request, approvalID, action string) (handled, ok bool) {
	if s.secretTokenSigner == nil {
		return false, false
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		return false, false
	}

	claims, err := s.secretTokenSigner.Verify(token)
	if err != nil {
		return true, false
	}
	if claims.ApprovalID != approvalID || claims.Action != action {
		return true, false
	}
	// Enforce single-use: a replayed token (same nonce) is rejected.
	if err := s.secretTokenGuard.Use(claims.Nonce); err != nil {
		if errors.Is(err, approvaltoken.ErrReused) {
			return true, false
		}
		return true, false
	}
	return true, true
}

// approverSessionAuthorized enforces the same bar as the requireApprover route
// wrapper (server.go) for the in-handler case, where the secret-approval token
// is an alternative authorization (#920). When Firebase auth is not configured
// it passes through — every route on an auth-less dashboard is public by
// design; otherwise the request must carry a bearer-token session whose role
// is Approver. Writes the error response and returns false when unauthorized.
func (s *Server) approverSessionAuthorized(w http.ResponseWriter, r *http.Request) bool {
	if s.tokenVerifier == nil {
		return true
	}
	authorized := false
	s.requireApprover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		authorized = true
	})).ServeHTTP(w, r)
	return authorized
}
