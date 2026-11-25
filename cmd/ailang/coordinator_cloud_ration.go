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
)

// agyQuotaRun reads `agy -p /quota` under a HOME; a variable so tests replay output.
var agyQuotaRun = agy.RunQuota

// checkAgyRation is the cloud agy job's admission check (M-AGY-CLOUD D3, §4): read
// the Google AI subscription's own quota with the leased credential and refuse to
// clone when the account is over the shared `google` pace line, the same evaluator
// and AILANG_GOOGLE_DAILY_PACE the rig's mission loops use. Quota is per account,
// so rig and cloud spend one budget with no shared ledger.
//
// This is ADMISSION, not reservation: a rig fire and a cloud job checking at the
// same moment can both pass. The start margin absorbs that bounded overshoot.
//
// An executor.ErrProcessTerminationUnconfirmed from the read is returned as is, so
// the caller quarantines the credential.
func checkAgyRation(ctx context.Context) error {
	pace, err := config.GoogleDailyPace()
	if err != nil {
		return fmt.Errorf("google ration unreadable (%v); not cloning", err)
	}
	home, err := agy.SubscriptionHome()
	if err != nil {
		return fmt.Errorf("google ration unreadable (%v); not cloning", err)
	}
	out, runErr := agyQuotaRun(ctx, home)
	if errors.Is(runErr, executor.ErrProcessTerminationUnconfirmed) {
		return runErr
	}
	o := mission.EvaluateGoogleQuota(out, runErr, home, pace, time.Now())
	if o.Blocked() {
		return fmt.Errorf("google ration %s — %s; not cloning", o.State, o.Reason)
	}
	fmt.Printf("execute-job: google ration ok — %s\n", o.Reason)
	return nil
}
