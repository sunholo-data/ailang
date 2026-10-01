package server

// Caller authentication for the secret-approval intake and status routes
// (POST /api/approvals, GET /api/approvals/{id}).
//
// The dashboard is public (allUsers invoker), and these two routes used to do
// no auth at all: anyone on the internet could put pending secret requests in
// the approver's queue, and poll any approval's decision by its predictable
// id. The client always sent AILANG_APPROVAL_TOKEN when it had one, but
// nothing here checked it.
//
// Two credentials are accepted, and nothing else:
//
//  1. A Google-signed OIDC ID token (what an executor job on GCP mints from the
//     metadata server), whose audience is the dashboard URL and whose verified
//     email is on AILANG_APPROVAL_ALLOWED_CALLERS. The same shape as the
//     coordinator's resident-lifecycle route (verifyLifecycleCaller).
//  2. AILANG_APPROVAL_TOKEN, a shared secret configured on BOTH sides, compared
//     in constant time — for local/CLI callers who cannot mint an ID token.
//
// Fail closed: with neither configured, every caller is refused.
// AILANG_APPROVAL_INTAKE_AUTH=off is the one way to admit anonymous callers
// (a local dashboard, or a rollback lever), and it says so in the log.
//
// The poll needs the same credential as the create because the client already
// holds it (zero cost), the ids are a nanosecond timestamp (enumerable), and
// the status is exactly the signal of when a secret is being released.
//
// Approve/reject are NOT gated here: they keep their own signed ntfy token or
// Approver session (checkSecretApprovalToken / approverSessionAuthorized).

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"

	"google.golang.org/api/idtoken"

	"github.com/sunholo-data/ailang/internal/config"
)

// idTokenValidator checks an ID token's signature and expiry. Audience and
// issuer are checked by the caller, so a list of audiences works. Tests inject
// a fake; nil means idtoken.Validate.
type idTokenValidator func(ctx context.Context, token, audience string) (*idtoken.Payload, error)

// googleIDTokenIssuers are the issuers of Google-signed service-account ID
// tokens. idtoken.Validate does not check iss, and it also accepts ES256 IAP
// assertions (iss https://cloud.google.com/iap), which are not this credential.
var googleIDTokenIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

// approvalCallerAnonymous is recorded as the caller when intake auth is off.
const approvalCallerAnonymous = "anonymous (AILANG_APPROVAL_INTAKE_AUTH=off)"

// approvalCallerSharedToken is recorded as the caller for the shared secret,
// which identifies no principal.
const approvalCallerSharedToken = "shared-token (AILANG_APPROVAL_TOKEN)"

// authorizeApprovalCaller authenticates a create/poll request. It returns the
// caller identity on success; otherwise it has written 401 (no or bad
// credential) or 403 (a valid Google identity not on the allowlist) and
// returns ok=false.
func (s *Server) authorizeApprovalCaller(w http.ResponseWriter, r *http.Request) (caller string, ok bool) {
	caller, status, reason := s.approvalCaller(r)
	if status == 0 {
		return caller, true
	}
	log.Printf("secret approval: refused %s %s from %s: %s", r.Method, r.URL.Path, r.RemoteAddr, reason)
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ailang-approvals"`)
	}
	http.Error(w, http.StatusText(status), status)
	return "", false
}

// approvalCaller is authorizeApprovalCaller without the response: status 0 on
// success, else the HTTP status and a log-only reason (never the token).
func (s *Server) approvalCaller(r *http.Request) (caller string, status int, reason string) {
	if config.ApprovalIntakeAuthOff() {
		return approvalCallerAnonymous, 0, ""
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", http.StatusUnauthorized, "missing bearer token"
	}
	tok := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if tok == "" {
		return "", http.StatusUnauthorized, "empty bearer token"
	}

	if shared := config.ApprovalToken(); shared != "" && constantTimeEqual(tok, shared) {
		return approvalCallerSharedToken, 0, ""
	}

	audiences := splitList(config.ApprovalAudience())
	allowed := splitList(config.ApprovalAllowedCallers())
	if len(audiences) == 0 || len(allowed) == 0 {
		return "", http.StatusUnauthorized, "token not accepted: no shared token matched and ID-token auth is unconfigured (AILANG_APPROVAL_ALLOWED_CALLERS / AILANG_APPROVAL_AUDIENCE)"
	}

	validate := s.idTokenValidate
	if validate == nil {
		validate = idtoken.Validate
	}
	// Signature and expiry first: a claim from an unverified token is just a
	// string the caller chose.
	payload, err := validate(r.Context(), tok, "")
	if err != nil {
		return "", http.StatusUnauthorized, "ID token rejected: " + err.Error()
	}
	if !googleIDTokenIssuers[payload.Issuer] {
		return "", http.StatusUnauthorized, "ID token issuer " + payload.Issuer + " is not Google"
	}
	if !containsFold(audiences, normalizeAudience(payload.Audience)) {
		return "", http.StatusUnauthorized, "ID token audience " + payload.Audience + " is not this dashboard"
	}
	email, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !verified {
		return "", http.StatusForbidden, "ID token carries no verified email"
	}
	if !containsFold(allowed, email) {
		return "", http.StatusForbidden, "caller " + email + " is not on AILANG_APPROVAL_ALLOWED_CALLERS"
	}
	return email, 0, ""
}

// constantTimeEqual compares two secrets without leaking where they differ or
// (by hashing first) how long the configured one is.
func constantTimeEqual(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

// splitList splits a comma-separated config value, dropping blanks and
// normalizing URL-shaped entries so "https://x/" and "https://x" agree.
func splitList(v string) []string {
	var out []string
	for _, item := range strings.Split(v, ",") {
		if item = normalizeAudience(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeAudience(v string) string { return strings.TrimRight(strings.TrimSpace(v), "/") }

func containsFold(list []string, v string) bool {
	for _, item := range list {
		if strings.EqualFold(item, v) {
			return true
		}
	}
	return false
}
