package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	storefs "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// `ailang mission quota --allow <bucket> --for <d>` lifts a cloud plane's admission ration
// (coordinator_cloud_ration.go) for one subscription bucket until a fixed time — so an
// attended session can send a task up while the shared account is over its pace line.
//
// Like --codex-reset it is an attended decision, refused inside a mission iteration. That
// is a guard against the loop doing it by accident, not a security boundary: writing the
// record needs the operator's Firestore access, which is the actual trust line, and every
// job it admits logs who granted it, why, and until when.

// overrideBuckets are the buckets a cloud ration exists for.
var overrideBuckets = []string{"google", "codex", "anthropic"}

// maxQuotaOverride bounds --for: an override is an exception, never a setting.
const maxQuotaOverride = 24 * time.Hour

// quotaOverrideStore is what the command needs from Firestore.
type quotaOverrideStore interface {
	Get(context.Context, string) (*storefs.QuotaOverride, error)
	Put(context.Context, storefs.QuotaOverride) error
	Delete(context.Context, string) error
}

// openQuotaOverrideStore opens the plane's store; a variable so tests use a fake.
var openQuotaOverrideStore = func(ctx context.Context, project string) (quotaOverrideStore, func(), error) {
	client, err := storefs.NewClientForProject(ctx, project)
	if err != nil {
		return nil, nil, err
	}
	return storefs.NewQuotaOverrideStore(client), func() { _ = client.Close() }, nil
}

type quotaOverrideRequest struct {
	allow, revoke string
	project       string
	reason        string
	duration      time.Duration
	yes           bool
}

func (r quotaOverrideRequest) wanted() bool { return r.allow != "" || r.revoke != "" }

// missionQuotaOverride grants or revokes an override and prints what is now in force.
func missionQuotaOverride(r quotaOverrideRequest, now time.Time) error {
	if config.MissionControlActive() || config.MissionRole() != "" {
		return fmt.Errorf("--allow/--revoke is an attended decision and is refused inside a mission iteration (MISSION_CONTROL_ACTIVE/MISSION_ROLE set)")
	}
	if r.allow != "" && r.revoke != "" {
		return fmt.Errorf("use --allow or --revoke, not both")
	}
	name := r.allow + r.revoke
	buckets, err := overrideBucketsFor(name)
	if err != nil {
		return err
	}
	if r.project == "" {
		return fmt.Errorf("--project is required: the plane whose cloud jobs to override (ailang-multivac for prod, ailang-multivac-dev for dev)")
	}
	if r.allow != "" {
		if r.duration <= 0 || r.duration > maxQuotaOverride {
			return fmt.Errorf("--for must be between 1s and %s (got %s)", maxQuotaOverride, r.duration)
		}
		if strings.TrimSpace(r.reason) == "" {
			return fmt.Errorf("--reason is required: it is logged by every job the override admits")
		}
		if !r.yes {
			return fmt.Errorf("--allow lets cloud %s jobs start over their pace line until %s; re-run with --yes to confirm",
				strings.Join(buckets, "/"), now.Add(r.duration).Format(time.RFC3339))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, closeStore, err := openQuotaOverrideStore(ctx, r.project)
	if err != nil {
		return fmt.Errorf("open quota overrides in %s: %w", r.project, err)
	}
	defer closeStore()
	for _, b := range buckets {
		if r.revoke != "" {
			if err := store.Delete(ctx, b); err != nil {
				return fmt.Errorf("revoke %s: %w", b, err)
			}
			fmt.Printf("%s: override revoked in %s; the ration applies again\n", b, r.project)
			continue
		}
		o := storefs.QuotaOverride{Bucket: b, Until: now.Add(r.duration).UTC(), GrantedAt: now.UTC(), GrantedBy: overrideGrantor(), Reason: strings.TrimSpace(r.reason)}
		if err := store.Put(ctx, o); err != nil {
			return fmt.Errorf("allow %s: %w", b, err)
		}
		fmt.Printf("%s: cloud ration NOT ENFORCED in %s until %s (granted by %s: %s)\n", b, r.project, o.Until.Format(time.RFC3339), o.GrantedBy, o.Reason)
	}
	return nil
}

// overrideBucketsFor expands a bucket name; "all" is every rationed bucket.
func overrideBucketsFor(name string) ([]string, error) {
	if name == "all" {
		return overrideBuckets, nil
	}
	for _, b := range overrideBuckets {
		if name == b {
			return []string{b}, nil
		}
	}
	return nil, fmt.Errorf("unknown quota override bucket %q: want one of %s or all", name, strings.Join(overrideBuckets, ", "))
}

// overrideGrantor names who granted an override: login@host.
func overrideGrantor() string {
	name := "unknown"
	if u, err := user.Current(); err == nil && u.Username != "" {
		name = u.Username
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		name += "@" + h
	}
	return name
}
