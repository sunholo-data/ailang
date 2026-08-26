package browserbase

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
)

func TestLiveBrowserbaseLifecycle(t *testing.T) {
	if os.Getenv("AILANG_BROWSERBASE_LIVE") != "1" {
		t.Skip("set AILANG_BROWSERBASE_LIVE=1 with BROWSERBASE_API_KEY to run the billable live contract smoke")
	}
	provider, err := New(Config{
		APIKey:    os.Getenv("BROWSERBASE_API_KEY"),
		ProjectID: os.Getenv("BROWSERBASE_PROJECT_ID"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	session, err := provider.Create(ctx, browser.SessionSpec{
		RunID: "ailang-live-contract", MaximumDuration: 60 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		if _, stopErr := provider.Stop(stopCtx, session); stopErr != nil {
			t.Errorf("stop live session: %v", stopErr)
		}
	}()
	if _, err := provider.Connection(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Inspect(ctx, session); err != nil {
		t.Fatal(err)
	}
}
