package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// A shadow run (AILANG_TASK_SHADOW=true, M-AILANG-LANE-EXECUTOR-SHADOW) runs a
// real task's directive on the other lane harness so the two can be graded. It
// may write nothing outside its own run directory. Three independent gates
// enforce that, so no single mistake publishes a shadow's work:
//
//  1. the job skips its own write steps: the deploy-key write pre-flight, the
//     push and the PR (executeCloudTask returns after collecting evidence);
//  2. the agent child gets no git credential (childGitCredential);
//  3. origin's push URL is replaced with one no transport can reach, so a push
//     from anyone in the clone fails, whatever credential is lying around
//     (an SSH deploy key's alias lives in the job's HOME, not in the child's
//     credential file).
//
// The commit is still made locally: the diff against the clone point is the
// evidence the grader compares.

// shadowPushURL names a transport git does not have, so `git push origin`
// fails before any network or credential is consulted.
const shadowPushURL = "shadow-run-no-push://refused"

// disableShadowPush points origin's push URL at shadowPushURL. Fetches keep
// using the real URL.
func disableShadowPush(ctx context.Context, workDir string) error {
	cmd := gitexec.CommandContext(ctx, "-C", workDir, "remote", "set-url", "--push", "origin", shadowPushURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shadow run: could not disable pushes to origin: %w: %s", err, out)
	}
	fmt.Println("execute-job: shadow run — origin push URL disabled; no push, PR or git credential")
	return nil
}

// shadowKeyRefusal starts the error a shadow run fails with when its plane has
// no shadow key; the grader reads it to tell "not run" from "ran and failed".
const shadowKeyRefusal = "shadow run refused"

// useShadowKey points the run at the shadow key (design D7): shadows have
// their own capped OpenRouter key so they can never drain the fleet's. With no
// shadow key the run is refused, never quietly billed to the fleet key.
func useShadowKey() error {
	key := config.OpenRouterKeyShadow()
	if key == "" {
		return fmt.Errorf("%s: %s is not set, and a shadow never spends the fleet key", shadowKeyRefusal, config.EnvOpenRouterKeyShadow)
	}
	if err := os.Setenv(config.EnvOpenRouterAPIKey, key); err != nil {
		return fmt.Errorf("shadow run: %w", err)
	}
	fmt.Printf("execute-job: shadow run — OpenRouter calls bill the shadow key (%s)\n", config.EnvOpenRouterKeyShadow)
	return nil
}

// shadowRun reports whether this job is a shadow run.
func shadowRun() bool { return config.TaskShadow() }

// needsDeployKeyWriteCheck reports whether the job should prove its deploy key
// can push. A shadow run never writes, so it neither needs nor checks write
// access; a read-only key is enough for it to clone.
func needsDeployKeyWriteCheck(deployKeyRepo string) bool {
	return deployKeyRepo != "" && !shadowRun()
}
