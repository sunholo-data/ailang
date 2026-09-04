package coordinator

// Resident agent idle sweep (v6.40.0 M4).
//
// WHY THIS IS AN ENDPOINT AND NOT A SCRIPT
//
// The behaviour first existed as ailang-multivac/scripts/resident-sweep.sh,
// driven by hand; that script was DELETED when this landed. A schedule cannot
// run a shell script without a job to host it, and the way back
// (/instances/start) already lives here with the identity and the narrow
// get/start/stop role the work needs. So the sweep rides the same service for
// the same reason D13 gave the start path: no new thing to deploy, and — since
// the script is gone rather than kept "just in case" — no second copy of these
// rules to drift out of step.
//
// WHAT IT REFUSES IS THE SPECIFICATION
//
// Stopping costs a user their running agent to save pennies, and a stopped
// instance does NOT come back on its own — its URL 404s until something calls
// :start. So:
//
//   - It NEVER sweeps on missing information. An instance whose health cannot
//     be read, or whose health does not report idleness, is SKIPPED. An
//     instance we cannot assess might be mid-run.
//   - It reports by default. `apply` must be asked for explicitly.
//   - It refuses to apply at all unless this service can start instances
//     again. A sweep without a way back does not save money, it strands the
//     users who were idle longest — quietly, and worst for the ones who used
//     it least recently.
//
// WHY IT ASKS THE INSTANCE RATHER THAN A DATABASE
//
// The agent knows when it last did work. `/health` reports `runs.idle_s`,
// counting WORK and not traffic — a sweep that counted health probes would
// never stop anything, because the sweep's own probe is traffic.
//
// WHY THE COORDINATOR'S OWN IDENTITY PROBES HEALTH
//
// The script impersonated each instance's service account. Doing that here
// would need the coordinator to hold serviceAccountTokenCreator on every
// PER-USER agent SA — the exact power D11 says must not exist, since it would
// let one identity act as any user's agent. Instead the coordinator calls
// /health as ITSELF and each instance lists it in RESIDENT_ALLOWED_CALLERS.
// That grants the coordinator permission to ASK, never to ACT AS.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/iterator"
)

const (
	defaultSweepIdleMinutes = 30
	// A health probe that hangs must not hold the sweep open. Short, because
	// a live instance answers /health locally in milliseconds.
	healthProbeTimeout = 20 * time.Second
)

type sweepRequest struct {
	IdleMinutes int  `json:"idle_minutes"`
	Apply       bool `json:"apply"`
}

// Actions are named for what happened, not for what was decided, so a reader of
// the response cannot mistake a dry run for a stop.
const (
	sweepActionStopped   = "stopped"
	sweepActionWouldStop = "would-stop"
	sweepActionKept      = "kept"
	sweepActionSkipped   = "skipped"
)

type sweepInstanceResult struct {
	Instance string `json:"instance"`
	Action   string `json:"action"`
	IdleS    int    `json:"idle_s,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type sweepResponse struct {
	Applied bool                  `json:"applied"`
	Stopped int                   `json:"stopped"`
	Kept    int                   `json:"kept"`
	Skipped int                   `json:"skipped"`
	Results []sweepInstanceResult `json:"results"`
}

// residentHealth mirrors the fields the sweep needs from GET /health.
//
// POINTERS, DELIBERATELY. A missing `idle_s` must be UNKNOWN, not zero — and
// zero means "busy right now", the most dangerous value to invent. An image
// predating the runs block reports no idleness at all, which is exactly the
// case that has to skip rather than read as freshly idle.
type residentHealth struct {
	Runs struct {
		Active *int `json:"active"`
		IdleS  *int `json:"idle_s"`
	} `json:"runs"`
}

// decideSweep turns one assessment into an action. Pure, so the rules are
// testable without a GCP project: the rules are the risky part, not the RPCs.
func decideSweep(h *residentHealth, probeErr error, thresholdS int) (action, reason string, idleS int) {
	if probeErr != nil {
		return sweepActionSkipped, fmt.Sprintf("health unreachable: %v", probeErr), 0
	}
	if h == nil || h.Runs.IdleS == nil {
		return sweepActionSkipped, "health did not report idleness", 0
	}
	if h.Runs.Active == nil {
		return sweepActionSkipped, "health did not report active runs", 0
	}
	if *h.Runs.Active > 0 {
		return sweepActionKept, "working", 0
	}
	if *h.Runs.IdleS >= thresholdS {
		return sweepActionStopped, "", *h.Runs.IdleS
	}
	return sweepActionKept, "recently active", *h.Runs.IdleS
}

// canStartAgain reports whether this service could bring a stopped instance
// back. The interlock: without it, sweeping is a one-way door.
func canStartAgain() error {
	if os.Getenv("RESIDENT_LIFECYCLE_PROJECT") == "" || os.Getenv("RESIDENT_LIFECYCLE_REGION") == "" {
		return fmt.Errorf("resident lifecycle project/region not configured")
	}
	if os.Getenv("RESIDENT_LIFECYCLE_AUDIENCE") == "" || os.Getenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS") == "" {
		return fmt.Errorf("resident lifecycle start route is not reachable by any caller")
	}
	return nil
}

// probeHealth asks one instance how idle it is, as the coordinator's own
// identity. The instance authorises us by listing our SA; we never act as it.
func probeHealth(ctx context.Context, baseURL string) (*residentHealth, error) {
	ctx, cancel := context.WithTimeout(ctx, healthProbeTimeout)
	defer cancel()
	client, err := idtoken.NewClient(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("could not mint an identity token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		// 404 here is a STOPPED instance answering from the Cloud Run frontend,
		// not a broken one. Either way it is not something to stop again.
		return nil, fmt.Errorf("health returned %d", resp.StatusCode)
	}
	var h residentHealth
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return nil, fmt.Errorf("health was not readable: %w", err)
	}
	return &h, nil
}

// handleSweepInstances stops resident instances that are doing nothing.
func (d *Daemon) handleSweepInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	if err := verifyLifecycleCaller(ctx, r); err != nil {
		d.logger.Printf("sweep: refused: %v", err)
		// Same single shape as the start route, for the same reason.
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req sweepRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.IdleMinutes <= 0 {
		req.IdleMinutes = defaultSweepIdleMinutes
	}
	if req.Apply {
		if err := canStartAgain(); err != nil {
			// 409, not 500: the request is well-formed and the refusal is a
			// state of the world the caller can fix.
			d.logger.Printf("sweep: refusing --apply: %v", err)
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "refusing to stop instances with no way to start them: " + err.Error(),
			})
			return
		}
	}

	project := os.Getenv("RESIDENT_LIFECYCLE_PROJECT")
	region := os.Getenv("RESIDENT_LIFECYCLE_REGION")
	if project == "" || region == "" {
		http.Error(w, "resident lifecycle project/region not configured", http.StatusServiceUnavailable)
		return
	}

	client, err := run.NewInstancesClient(ctx)
	if err != nil {
		d.logger.Printf("sweep: client: %v", err)
		http.Error(w, "could not reach Cloud Run", http.StatusBadGateway)
		return
	}
	defer func() { _ = client.Close() }()

	out := sweepResponse{Applied: req.Apply, Results: []sweepInstanceResult{}}
	thresholdS := req.IdleMinutes * 60

	it := client.ListInstances(ctx, &runpb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", project, region),
	})
	for {
		inst, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			d.logger.Printf("sweep: list: %v", err)
			http.Error(w, "could not list instances", http.StatusBadGateway)
			return
		}
		name := inst.GetName()
		if i := lastSegment(name); i != "" {
			name = i
		}
		if !residentInstanceName.MatchString(name) {
			continue
		}
		// Only a RUNNING instance can be idle. Anything else is already stopped
		// or mid-transition, and stopping it again is not an improvement.
		if inst.GetTerminalCondition().GetState() != runpb.Condition_CONDITION_SUCCEEDED {
			continue
		}
		urls := inst.GetUrls()
		if len(urls) == 0 {
			out.Results = append(out.Results, sweepInstanceResult{
				Instance: name, Action: sweepActionSkipped, Reason: "instance has no URL to ask",
			})
			out.Skipped++
			continue
		}

		h, probeErr := probeHealth(ctx, urls[0])
		action, reason, idleS := decideSweep(h, probeErr, thresholdS)

		switch action {
		case sweepActionSkipped:
			out.Skipped++
		case sweepActionKept:
			out.Kept++
		case sweepActionStopped:
			if !req.Apply {
				action = sweepActionWouldStop
				out.Stopped++
				break
			}
			full := fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, region, name)
			if _, err := client.StopInstance(ctx, &runpb.StopInstanceRequest{Name: full}); err != nil {
				d.logger.Printf("sweep: stop %s: %v", full, err)
				action, reason = sweepActionSkipped, fmt.Sprintf("stop failed: %v", err)
				out.Skipped++
				break
			}
			d.logger.Printf("sweep: stopped %s (idle %ds)", full, idleS)
			out.Stopped++
		}
		out.Results = append(out.Results, sweepInstanceResult{
			Instance: name, Action: action, IdleS: idleS, Reason: reason,
		})
	}

	d.logger.Printf("sweep: apply=%t %d stopped, %d kept, %d skipped",
		req.Apply, out.Stopped, out.Kept, out.Skipped)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// lastSegment returns the final "/"-separated element of a resource name.
func lastSegment(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' {
			return name[i+1:]
		}
	}
	return name
}
