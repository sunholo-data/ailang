package claudegateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

type faultAuthority struct {
	creditbudget.Authority
	reserveFail, forwardFail, settleFail, unknownFail bool
}

func (a faultAuthority) Reserve(ctx context.Context, r creditbudget.Reservation) (creditbudget.Request, error) {
	if a.reserveFail {
		return creditbudget.Request{}, errors.New("ledger unavailable")
	}
	return a.Authority.Reserve(ctx, r)
}
func (a faultAuthority) MarkForwarded(ctx context.Context, account, request string) error {
	if a.forwardFail {
		return errors.New("durable send marker unavailable")
	}
	return a.Authority.MarkForwarded(ctx, account, request)
}
func (a faultAuthority) Settle(ctx context.Context, account, request string, cost creditbudget.MicroUSD, id string) error {
	if a.settleFail {
		return errors.New("settlement commit unavailable")
	}
	return a.Authority.Settle(ctx, account, request, cost, id)
}
func (a faultAuthority) MarkUnresolved(ctx context.Context, account, request string) error {
	if a.unknownFail {
		return errors.New("unknown marker commit unavailable")
	}
	return a.Authority.MarkUnresolved(ctx, account, request)
}
func TestClientCancellationRetainsExposureWithDetachedAccounting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	g, a, cap := fixture(t, func(r *http.Request) (*http.Response, error) { calls++; cancel(); return nil, r.Context().Err() })
	r := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(plainRequest)).WithContext(ctx)
	r.Header.Set("x-api-key", cap)
	r.Header.Set("Authorization", "Bearer job-token")
	r.Header.Set("anthropic-version", "2023-06-01")
	g.ServeHTTP(httptest.NewRecorder(), r)
	s, err := a.Status(context.Background(), AccountID)
	if err != nil || calls != 1 || s.Reserved <= 0 || s.Unresolved != s.Reserved || !s.ReconciliationRequired {
		t.Fatalf("cancellation lost exposure %+v calls%d %v", s, calls, err)
	}
}
func TestStorageFailuresCannotAuthorizeOrRefundSend(t *testing.T) {
	for _, phase := range []string{"reserve", "forward", "settle", "settle-and-unknown"} {
		t.Run(phase, func(t *testing.T) {
			calls := 0
			g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(goodResponse))}, nil
			})
			g.Authority = faultAuthority{Authority: a, reserveFail: phase == "reserve", forwardFail: phase == "forward", settleFail: strings.HasPrefix(phase, "settle"), unknownFail: phase == "settle-and-unknown"}
			invoke(g, cap, plainRequest)
			s, err := a.Status(context.Background(), AccountID)
			if err != nil {
				t.Fatal(err)
			}
			if phase == "reserve" {
				if calls != 0 || s.Reserved != 0 {
					t.Fatal("failed reservation sent")
				}
				return
			}
			if s.Reserved <= 0 || s.Settled != 0 {
				t.Fatalf("storage failure refunded exposure %+v", s)
			}
			if phase == "forward" && calls != 0 {
				t.Fatal("failed durable send marker forwarded")
			}
			if strings.HasPrefix(phase, "settle") && calls != 1 {
				t.Fatal("invalid send count", calls)
			}
			if phase == "settle-and-unknown" && (s.Forwarding != s.Reserved || s.Unresolved != 0) {
				t.Fatal("lost crash/send intent exposure")
			}
		})
	}
}
