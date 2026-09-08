package coordinator

// Resident agent lifecycle (v6.40.0 M4).
//
// WHY THIS LIVES IN THE COORDINATOR
//
// A stopped Cloud Run instance does NOT wake on a request: its URL returns 404
// from the frontend until something calls :start, and it then takes ~30s to
// serve (both measured 2026-09-03). So an idle sweep without a way back does
// not save money — it permanently strands the users who were idle longest.
//
// The platform decides that a user needs their agent; this estate performs the
// act. That split is deliberate (design P2): handing a customer-facing web
// backend the power to operate the estate that runs the agent fleet is a much
// larger blast radius than the feature. The coordinator already runs here with
// its own identity, so it is the host — no new service to deploy or watch.
//
// WHY NOT requireAPIKey
//
// That middleware passes every request when COORDINATOR_API_KEY is unset —
// correct for read-only status in local mode, wrong for a route that starts and
// stops infrastructure, where a missing env var would silently open it. This
// verifies Google-signed OIDC against an explicit caller allowlist and FAILS
// CLOSED when unconfigured, matching the resident's own auth (docker/resident/
// lib/auth.mjs) rather than inventing a third rule.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/idtoken"
)

// Only instances whose name matches this may be operated through this route.
// The coordinator's identity may well be able to touch more; this endpoint may
// not. A capability is defined by what it refuses.
var residentInstanceName = regexp.MustCompile(`^resident-[a-z0-9][a-z0-9-]{0,40}$`)

type startInstanceRequest struct {
	// Either the instance name, or the URL the platform was trying to reach —
	// the platform holds URLs, not names, so making it derive one would put a
	// naming rule in two places.
	Instance string `json:"instance"`
	URL      string `json:"url"`
}

type startInstanceResponse struct {
	Instance string `json:"instance"`
	Started  bool   `json:"started"`
	State    string `json:"state"`
	Message  string `json:"message,omitempty"`
}

// instanceNameFromURL extracts "resident-x" from
// https://resident-x-1234567890.europe-west4.run.app
func instanceNameFromURL(rawURL string) string {
	host := rawURL
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	label := host
	if i := strings.Index(label, "."); i >= 0 {
		label = label[:i]
	}
	// Cloud Run appends "-<projectNumber>"; strip the last numeric segment.
	if i := strings.LastIndex(label, "-"); i > 0 {
		if suffix := label[i+1:]; suffix != "" && strings.Trim(suffix, "0123456789") == "" {
			label = label[:i]
		}
	}
	return label
}

// verifyLifecycleCaller checks a Google-signed ID token against an explicit
// allowlist. Fails closed: unconfigured means nobody, not everybody.
func verifyLifecycleCaller(ctx context.Context, r *http.Request) error {
	audience := os.Getenv("RESIDENT_LIFECYCLE_AUDIENCE")
	allowed := os.Getenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS")
	if audience == "" || allowed == "" {
		return fmt.Errorf("resident lifecycle is not configured (audience/allowlist unset)")
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return fmt.Errorf("missing bearer token")
	}
	// Signature and audience are validated before any claim is read — a claim
	// from an unverified token is just a string the caller chose.
	payload, err := idtoken.Validate(ctx, strings.TrimPrefix(auth, "Bearer "), audience)
	if err != nil {
		return fmt.Errorf("token rejected: %w", err)
	}
	email, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !verified {
		return fmt.Errorf("token carries no verified email")
	}
	for _, allowedEmail := range strings.Split(allowed, ",") {
		if strings.EqualFold(strings.TrimSpace(allowedEmail), email) {
			return nil
		}
	}
	return fmt.Errorf("caller %s is not on the allowlist", email)
}

// handleStartInstance starts a stopped resident instance.
//
// Idempotent: starting a running instance is a success, because the caller's
// intent ("make this reachable") is already satisfied and an error would push
// them into retry logic for a state they wanted.
func (d *Daemon) handleStartInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := verifyLifecycleCaller(ctx, r); err != nil {
		d.logger.Printf("lifecycle: refused: %v", err)
		// One shape for every refusal, so a probe cannot distinguish "not
		// configured" from "not allowed" and map the allowlist.
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req startInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	name := req.Instance
	if name == "" && req.URL != "" {
		name = instanceNameFromURL(req.URL)
	}
	if !residentInstanceName.MatchString(name) {
		http.Error(w, fmt.Sprintf("not a resident instance name: %q", name), http.StatusBadRequest)
		return
	}

	project := os.Getenv("RESIDENT_LIFECYCLE_PROJECT")
	region := os.Getenv("RESIDENT_LIFECYCLE_REGION")
	if project == "" || region == "" {
		http.Error(w, "resident lifecycle project/region not configured", http.StatusServiceUnavailable)
		return
	}

	client, err := run.NewInstancesClient(ctx)
	if err != nil {
		d.logger.Printf("lifecycle: client: %v", err)
		http.Error(w, "could not reach Cloud Run", http.StatusBadGateway)
		return
	}
	defer func() { _ = client.Close() }()

	full := fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, region, name)
	op, err := client.StartInstance(ctx, &runpb.StartInstanceRequest{Name: full})
	if err != nil {
		// Report the reason rather than a bare failure: "no such instance" and
		// "denied" send an operator to completely different places.
		d.logger.Printf("lifecycle: start %s: %v", full, err)
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(startInstanceResponse{
			Instance: name, Started: false, State: "error", Message: err.Error(),
		})
		return
	}

	// Return as soon as the start is ACCEPTED. Waiting out the ~30s boot would
	// hold a user's chat turn open for the whole of it; the caller is told to
	// ask again shortly instead.
	//
	// But WATCH the operation, because "accepted" is not "started". Observed
	// live 2026-09-04: a start was accepted at 12:11:24 and the instance was
	// still stopped five minutes later, with nothing anywhere saying so — this
	// function had discarded `op`. The user had already been told "starting,
	// ask again shortly", so the one outcome that permanently strands somebody
	// was the one outcome nothing watched. Not fatal to the request, so it does
	// not change the response; it changes whether a human can ever find out.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, err := op.Wait(ctx); err != nil {
			d.logger.Printf("lifecycle: start of %s was ACCEPTED but FAILED: %v", full, err)
			return
		}
		d.logger.Printf("lifecycle: start of %s completed", full)
	}()
	d.logger.Printf("lifecycle: start requested for %s", full)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(startInstanceResponse{
		Instance: name, Started: true, State: "starting",
		Message: "start accepted; the instance takes about 30s to serve",
	})
}
