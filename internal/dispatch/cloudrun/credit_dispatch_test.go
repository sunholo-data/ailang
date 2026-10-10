package cloudrun

import (
	"context"
	"errors"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func guardedParams() coordinator.DispatchParams {
	return coordinator.DispatchParams{TaskID: "task-credit", Provider: "claude", AuthMode: "apikey", Model: "claude-haiku-5-5", MaxCostUSD: 2, CreditAccount: "anthropic-api-credits", CreditGatewayURL: "https://gateway", CreditJobIdentity: "job@example.iam.gserviceaccount.com"}
}
func TestGuardedDispatchNeverLaunchesBlockedTask(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "prod", "region", "ailang")
	d.creditAdmission = func(context.Context, coordinator.DispatchParams) (coordinator.CloudCreditCapability, error) {
		return coordinator.CloudCreditCapability{}, errors.New("grant expired")
	}
	err := d.Dispatch(context.Background(), guardedParams())
	if !errors.Is(err, coordinator.ErrDispatchPermanent) || mock.lastReq != nil {
		t.Fatalf("blocked task launched or retriable: %v", err)
	}
}
func TestGuardedDispatchCarriesOnlyScopedGatewayCredential(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "prod", "region", "ailang")
	d.creditAdmission = func(context.Context, coordinator.DispatchParams) (coordinator.CloudCreditCapability, error) {
		return coordinator.CloudCreditCapability{Capability: "scoped", GatewayURL: "https://gateway", AccountID: "anthropic-api-credits"}, nil
	}
	if err := d.Dispatch(context.Background(), guardedParams()); err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{}
	for _, v := range mock.lastReq.Overrides.ContainerOverrides[0].Env {
		vars[v.Name] = v.GetValue()
	}
	for k, want := range map[string]string{"ANTHROPIC_API_KEY": "scoped", "ANTHROPIC_BASE_URL": "https://gateway", "AILANG_CLAUDE_CREDIT_ACCOUNT": "anthropic-api-credits", "AILANG_AUTH_MODE": "apikey", "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-haiku-5-5"} {
		if vars[k] != want {
			t.Fatalf("%s=%q", k, vars[k])
		}
	}
}

func TestGuardedDispatchReleasesOnlyKnownRejectedLaunch(t *testing.T) {
	for _, code := range []codes.Code{codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unavailable, codes.DeadlineExceeded} {
		t.Run(code.String(), func(t *testing.T) {
			mock := &mockJobRunner{err: status.Error(code, "launch failed")}
			d := newDispatcherWithClient(mock, "prod", "region", "ailang")
			releases := 0
			d.creditRelease = func(context.Context, coordinator.CloudCreditCapability) error { releases++; return nil }
			d.creditAdmission = func(context.Context, coordinator.DispatchParams) (coordinator.CloudCreditCapability, error) {
				return coordinator.CloudCreditCapability{AccountID: "anthropic-api-credits", GatewayURL: "https://gateway", Capability: "scoped"}, nil
			}
			if err := d.Dispatch(context.Background(), guardedParams()); err == nil {
				t.Fatal("launch failure lost")
			}
			known := code == codes.InvalidArgument || code == codes.NotFound || code == codes.PermissionDenied
			if (releases == 1) != known {
				t.Fatalf("%s released %d leases", code, releases)
			}
		})
	}
}
