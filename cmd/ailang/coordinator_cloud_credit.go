package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// releaseJobCreditAttempt runs at the outer job boundary, including setup
// failures. Failure retains the lease until expiry; it never refunds exposure.
func releaseJobCreditAttempt() {
	if config.ClaudeCreditAccount() == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cap := coordinator.CloudCreditCapability{AccountID: config.ClaudeCreditAccount(), Capability: config.AnthropicAPIKey(), GatewayURL: config.AnthropicBaseURL()}
	if err := coordinator.ReleaseCloudCreditTask(ctx, cap, nil, nil); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: credit lease release failed; exposure retained: %v\n", err)
	}
}
