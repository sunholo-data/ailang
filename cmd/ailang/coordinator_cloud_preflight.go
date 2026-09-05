package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The container verifies what the dispatcher chose (2026-09-05).
//
// Which CLI runs was decided twice, independently: the dispatcher derives it
// from the executor_variant (which is the image), and then this process read
// AILANG_PROVIDER and, when that was empty, assumed "claude". Two decisions with
// nothing forcing them to agree is the same shape of bug as the one the
// dispatcher fix removed — so it recurred there after being fixed here.
//
// It cost three prod runs on 2026-08-27/28: the codex images were told to run
// opencode, which they do not install. Each cloned the repo (24,032 files),
// created a branch and resolved pi extensions before exec failing on a missing
// binary — roughly a minute of work to discover something knowable at startup.
//
// Two guards, because they fail for different reasons:
//
//   - resolveContainerProvider compares the request against what the image says
//     it installs. Catches a wrong DECLARATION, and names both sides.
//   - preflightExecutor asks the executor's own HealthCheck whether its binary is
//     there. Catches a wrong IMAGE BUILD, and needs no declaration to work — it
//     would have caught all three of those failures with no other change.
//
// HealthCheck is not new: eval_harness, doctor, smoke-motoko and motoko's canary
// all call it. This path simply never did.

// multiCLIImage is what an image sets for AILANG_IMAGE_PROVIDER when it carries
// several executor CLIs and so cannot answer the question itself. Mirrors
// coordinator.multiCLIVariants on the dispatcher side.
const multiCLIImage = "multi"

// preflightHealthCheckTimeout bounds the binary probe. HealthCheck shells out to
// `<cli> --version`; a hang here would stall the task before it starts, which is
// the failure mode this whole function exists to prevent.
const preflightHealthCheckTimeout = 30 * time.Second

// resolveContainerProvider decides which executor this container should run.
//
// requested is AILANG_PROVIDER (set by the dispatcher from the agent's variant);
// image is AILANG_IMAGE_PROVIDER, baked into the Dockerfile beside the line that
// installs the CLI. There is deliberately no default: a provider this process
// guessed is exactly what put opencode into a codex container.
func resolveContainerProvider(requested, image string) (string, error) {
	requested = strings.TrimSpace(requested)
	image = strings.TrimSpace(image)

	switch {
	case image == "":
		// Built before images declared themselves, or agent-base, which carries
		// no executor CLI at all. Trust the dispatcher — it derives from the
		// variant — but still refuse to invent one.
		if requested == "" {
			return "", fmt.Errorf("no executor provider: AILANG_PROVIDER is unset and this image does not declare AILANG_IMAGE_PROVIDER")
		}
		return requested, nil

	case image == multiCLIImage:
		if requested == "" {
			return "", fmt.Errorf("this image carries several executor CLIs, so AILANG_PROVIDER must name one — it is unset")
		}
		return requested, nil

	case requested == "":
		return image, nil

	case requested != image:
		return "", fmt.Errorf(
			"provider mismatch: this image installs %q, but AILANG_PROVIDER asks for %q — the %s image cannot run %s",
			image, requested, image, requested)

	default:
		return requested, nil
	}
}

// preflightExecutor proves the chosen executor can actually run here, before the
// caller spends a repo clone finding out.
//
// The error names what is installed, because "opencode: executable file not
// found in $PATH" says what is missing and not what the container is.
func preflightExecutor(ctx context.Context, provider string) error {
	exec, err := executor.GlobalFactory().GetExecutor(provider)
	if err != nil {
		return fmt.Errorf("executor %q is not available in this image: %w", provider, err)
	}
	// Not Close()d: GetExecutor caches, and this same instance runs the task.

	ctx, cancel := context.WithTimeout(ctx, preflightHealthCheckTimeout)
	defer cancel()

	if err := exec.HealthCheck(ctx); err != nil {
		return fmt.Errorf("executor %q failed its health check in this image (%s): %w",
			provider, describeImage(), err)
	}
	return nil
}

// describeImage reports what the image says it is, for an error message read by
// someone who cannot see which container ran.
func describeImage() string {
	if declared := strings.TrimSpace(os.Getenv("AILANG_IMAGE_PROVIDER")); declared != "" {
		return "AILANG_IMAGE_PROVIDER=" + declared
	}
	// ListAvailable is what this BINARY can build, not what the image installs —
	// the two differing is the whole bug. Offered as a hint, labelled as such.
	if compiled := executor.GlobalFactory().ListAvailable(); len(compiled) > 0 {
		sort.Strings(compiled)
		return "image declares no provider; executors compiled in: " + strings.Join(compiled, ", ")
	}
	return "image declares no provider"
}
