package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/agy"
	"github.com/sunholo-data/ailang/internal/mission"
	storefs "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// Cloud admission rations (M-AGY-CLOUD §4, widened 2026-11-25 to every subscription lane).
//
// A cloud job on a subscription reads that account's own quota with the credential it
// just leased, and refuses to clone when the account is over the shared pace line —
// the same evaluators the rig's mission loops use (`ailang mission quota`). Quota is per
// account, so the rig, attended use and the cloud spend one budget with no shared ledger.
// This is ADMISSION, not reservation: two starts checking at once can both pass.
//
//	agy    -> google     `agy -p /quota`, AILANG_GOOGLE_DAILY_PACE %/day; unreadable BLOCKS
//	codex  -> codex      `codex app-server` account/rateLimits/read, weekday pace; unreadable BLOCKS
//	claude -> anthropic  /api/oauth/usage with the job's token, weekday pace; unreadable is
//	                     ADMITTED and logged NOT ENFORCED (Mark, 2026-11-25): the cloud
//	                     setup-token is refused by that endpoint (HTTP 403, measured
//	                     2026-11-25), so failing closed would stop all cloud claude work.
//	                     The gate starts refusing by itself once a token can read usage.
//
// An operator overrides a refusal with `ailang mission quota --allow <bucket> --for <d>`:
// a time-boxed Firestore record in the plane's project (storefs.QuotaOverride). It is
// read only when the ration would refuse, and every admission it causes is logged with
// who granted it, why, and until when.

// cloudRationBucket maps a cloud provider to the subscription bucket it spends; "" means
// the provider has no cloud ration (metered and flat-rate lanes are paced elsewhere).
func cloudRationBucket(provider string) string {
	switch provider {
	case "agy":
		return "google"
	case "codex":
		return "codex"
	case "claude":
		return "anthropic"
	}
	return ""
}

// rationVerdict is one bucket's admission reading.
type rationVerdict struct {
	blocked bool
	state   string
	reason  string
}

// Seams so tests replay provider output without credentials or network.
var (
	// agyQuotaRun reads `agy -p /quota` under a HOME.
	agyQuotaRun = agy.RunQuota
	// codexQuotaObserve asks the codex app-server for account rate limits.
	codexQuotaObserve = mission.ObserveCodexQuota
	// anthropicQuotaObserve reads /api/oauth/usage with one token.
	anthropicQuotaObserve = mission.ObserveAnthropicQuotaWithToken
	// quotaOverrideLookup reads the bucket's operator override in the plane's project.
	quotaOverrideLookup = lookupQuotaOverride
)

// checkCloudRation is the admission check every subscription cloud job runs after its
// credential is installed and before it clones. It returns nil to proceed.
//
// An executor.ErrProcessTerminationUnconfirmed from a quota read is returned as is, so
// the caller quarantines the credential.
func checkCloudRation(ctx context.Context, provider, project string) error {
	bucket := cloudRationBucket(provider)
	if bucket == "" {
		return nil
	}
	// The Claude credit lane bills metered API credits, not the subscription; it has
	// its own guard (internal/executor/claude/credit_guard.go).
	if bucket == "anthropic" && config.AuthMode() == "apikey" {
		return nil
	}
	v, err := observeCloudRation(ctx, bucket)
	if err != nil {
		return err
	}
	if !v.blocked {
		fmt.Printf("execute-job: %s ration ok — %s\n", bucket, v.reason)
		return nil
	}
	ov, ovErr := quotaOverrideLookup(ctx, project, bucket)
	if ovErr == nil && ov.Active(time.Now()) {
		fmt.Printf("execute-job: %s ration NOT ENFORCED — %s; operator override by %s until %s (%s)\n",
			bucket, v.reason, ov.GrantedBy, ov.Until.UTC().Format(time.RFC3339), ov.Reason)
		return nil
	}
	msg := fmt.Sprintf("%s ration %s — %s; not cloning", bucket, v.state, v.reason)
	if ovErr != nil {
		msg += fmt.Sprintf(" (operator override unreadable: %v)", ovErr)
	}
	msg += fmt.Sprintf(". Override from an attended session: ailang mission quota --allow %s --for 1h --reason <why> --project %s --yes", bucket, project)
	return errors.New(msg)
}

// observeCloudRation reads one bucket. Only a process-termination doubt is an error;
// every other failure is a verdict, so it is logged and decided like any other.
func observeCloudRation(ctx context.Context, bucket string) (rationVerdict, error) {
	now := time.Now()
	switch bucket {
	case "google":
		return observeGoogleRation(ctx, now)
	case "codex":
		o := codexQuotaObserve(config.CodexHome(), now)
		return rationVerdict{blocked: o.Blocked(), state: o.State, reason: o.Reason}, nil
	case "anthropic":
		token, _ := config.ClaudeCodeOAuthToken()
		o := anthropicQuotaObserve(token, now)
		if o.State == "unknown" {
			return rationVerdict{state: o.State,
				reason: "Anthropic usage unmeasurable from the cloud (" + o.Reason + "); NOT ENFORCED"}, nil
		}
		return rationVerdict{blocked: o.Blocked(), state: o.State, reason: o.Reason}, nil
	}
	return rationVerdict{blocked: true, state: "unknown", reason: "no ration reader for bucket " + bucket}, nil
}

// observeGoogleRation reads the Google AI subscription with the leased agy credential.
// The /quota read is itself a refresh-capable agy process, so it runs under the lease.
func observeGoogleRation(ctx context.Context, now time.Time) (rationVerdict, error) {
	pace, err := config.GoogleDailyPace()
	if err != nil {
		return rationVerdict{blocked: true, state: "unreadable", reason: err.Error()}, nil
	}
	home, err := agy.SubscriptionHome()
	if err != nil {
		return rationVerdict{blocked: true, state: "unreadable", reason: err.Error()}, nil
	}
	out, runErr := agyQuotaRun(ctx, home)
	if errors.Is(runErr, executor.ErrProcessTerminationUnconfirmed) {
		return rationVerdict{}, runErr
	}
	o := mission.EvaluateGoogleQuota(out, runErr, home, pace, now)
	return rationVerdict{blocked: o.Blocked(), state: o.State, reason: o.Reason}, nil
}

// lookupQuotaOverride reads the bucket's override from the plane's Firestore.
func lookupQuotaOverride(ctx context.Context, project, bucket string) (*storefs.QuotaOverride, error) {
	if project == "" {
		return nil, fmt.Errorf("no GCP project to read the override from")
	}
	client, err := storefs.NewClientForProject(ctx, project)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()
	return storefs.NewQuotaOverrideStore(client).Get(ctx, bucket)
}
