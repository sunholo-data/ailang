package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
)

// Writing the config is not deploying it.
//
// `config set` writes gs://<project>-ailang-config/config.yaml, and Cloud Run
// mounts that bucket into the coordinator at /etc/ailang-config via gcsfuse — so
// the FILE the container sees changes within seconds. The AgentRegistry does
// not: it is built once at startup from AILANG_CONFIG. Routing therefore keeps
// following the old config until a revision rolls.
//
// Measured 2026-09-10: `controlplane` was declared triage_only, the write
// succeeded, and a probe to `controlplane` still bounced. Only a forced revision
// made it take effect. The gap is invisible from the writing end — the write
// reports success, the file is genuinely updated, and nothing says the running
// service disagrees.
//
// So a successful write now rolls the service (Mark, attended 2026-09-10).
//
// TWO PROPERTIES THIS MUST KEEP:
//
//  1. A FAILED ROLL MUST NOT LOOK LIKE A FAILED WRITE. The write already
//     happened and is durable; losing that distinction would send the next
//     operator to re-write config that is already correct. The write is reported
//     first, on its own line, and the roll's outcome separately.
//  2. A FAILED ROLL MUST NOT LOOK LIKE SUCCESS. A machine without Cloud Run
//     permissions (or offline) is a normal case, not an error worth refusing the
//     write over — but it leaves the plane on the OLD config, which is exactly
//     the state this exists to prevent. It exits non-zero and names the manual
//     command.

// rollTimeout bounds the roll. A revision that has not begun serving in this
// long is a problem to report, not to keep waiting on — the write is safe
// either way.
const rollTimeout = 3 * time.Minute

// configReloadLabel is stamped with the write time. Cloud Run creates a new
// revision only when the template CHANGES, so rolling requires mutating
// something; a label is the least invasive thing that is not a runtime input.
// It doubles as a record of when the config was last pushed.
const configReloadLabel = "ailang-config-reload"

// coordinatorService names the service to roll.
//
// Only ailang-coordinator mounts the config bucket (verified against every
// service in the region, 2026-09-10). Overridable so a differently-named
// deployment does not need a code change.
func coordinatorService() (project, region, service string) {
	project = os.Getenv("AILANG_CLOUD_PROJECT")
	if project == "" {
		project = "ailang-multivac"
	}
	region = os.Getenv("AILANG_CLOUD_REGION")
	if region == "" {
		region = "europe-west1"
	}
	service = os.Getenv("AILANG_COORDINATOR_SERVICE")
	if service == "" {
		service = "ailang-coordinator"
	}
	return project, region, service
}

// rollCoordinatorForConfig forces a new revision so the coordinator re-reads the
// config it is mounting.
//
// Returns the new revision name on success.
func rollCoordinatorForConfig(ctx context.Context) (string, error) {
	project, region, service := coordinatorService()
	full := fmt.Sprintf("projects/%s/locations/%s/services/%s", project, region, service)

	ctx, cancel := context.WithTimeout(ctx, rollTimeout)
	defer cancel()

	client, err := run.NewServicesClient(ctx)
	if err != nil {
		return "", fmt.Errorf("could not reach Cloud Run: %w", err)
	}
	defer func() { _ = client.Close() }()

	svc, err := client.GetService(ctx, &runpb.GetServiceRequest{Name: full})
	if err != nil {
		return "", fmt.Errorf("get %s: %w", service, err)
	}

	// Mutate a label on the TEMPLATE, not the service: a service-level label
	// does not change the revision template and would not roll anything, which
	// would reproduce the silent no-op this function exists to remove.
	if svc.Template == nil {
		return "", fmt.Errorf("%s has no revision template to roll", service)
	}
	if svc.Template.Labels == nil {
		svc.Template.Labels = map[string]string{}
	}
	svc.Template.Labels[configReloadLabel] = time.Now().UTC().Format("2006-01-02t15-04-05z")
	// Let Cloud Run name the revision; a stale pinned name is rejected.
	svc.Template.Revision = ""

	op, err := client.UpdateService(ctx, &runpb.UpdateServiceRequest{Service: svc})
	if err != nil {
		return "", fmt.Errorf("update %s: %w", service, err)
	}
	rolled, err := op.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("waiting for %s to roll: %w", service, err)
	}
	return rolled.GetLatestReadyRevision(), nil
}

// reportConfigRoll prints the outcome and reports whether the plane is live.
//
// Callers must let a false result reach the exit code. The whole point is that
// "written" and "live" stop being the same statement.
func reportConfigRoll(ctx context.Context, skip bool) bool {
	_, _, service := coordinatorService()

	if skip {
		fmt.Println()
		fmt.Println(yellow("!"), "NOT rolled (--no-roll):", service, "is still running the PREVIOUS config.")
		fmt.Println("  Routing follows the registry loaded at startup, not the file on the mount.")
		fmt.Printf("  Roll it with: %s\n", cyan("ailang coordinator config roll"))
		return false
	}

	fmt.Printf("\nrolling %s so it re-reads the config…\n", service)
	revision, err := rollCoordinatorForConfig(ctx)
	if err != nil {
		project, region, svc := coordinatorService()
		fmt.Println()
		fmt.Println(red("✗"), "the config WAS written, but", service, "was NOT rolled:", err)
		fmt.Println("  The file is updated and durable — do not re-write it. The running")
		fmt.Println("  service is still using the config it loaded at startup.")
		fmt.Printf("  Roll it with: %s\n", cyan("ailang coordinator config roll"))
		fmt.Printf("  or:           gcloud run services update %s --project %s --region %s \\\n", svc, project, region)
		fmt.Printf("                  --update-labels=%s=$(date +%%s)\n", configReloadLabel)
		return false
	}

	fmt.Println(green("✓"), "rolled:", revision)
	fmt.Println("  Verify with a TWO-probe test — an absent bounce is ambiguous on its own:")
	fmt.Printf("    %s\n", cyan("ailang messages send <a-declared-inbox> … # must NOT bounce"))
	fmt.Printf("    %s\n", cyan("ailang messages send <unregistered-name> … # MUST bounce"))
	return true
}

// coordinatorConfigRoll implements `ailang coordinator config roll` — the manual
// path, for recovering a write whose roll failed.
func coordinatorConfigRoll(ctx context.Context) error {
	_, _, service := coordinatorService()
	fmt.Printf("rolling %s…\n", service)
	revision, err := rollCoordinatorForConfig(ctx)
	if err != nil {
		return err
	}
	fmt.Println(green("✓"), "rolled:", revision)
	return nil
}

// noRollRequested reports whether --no-roll was passed, and returns the args
// with it removed so the existing flag parsing is unchanged.
func noRollRequested(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	skip := false
	for _, a := range args {
		if strings.EqualFold(a, "--no-roll") {
			skip = true
			continue
		}
		out = append(out, a)
	}
	return out, skip
}
