package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreditRecoveryUsesExactAuditedOperatorPayloads(t *testing.T) {
	for _, action := range []string{"conservative-debit", "external-debit"} {
		t.Run(action, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/admin/credits/anthropic-api-credits/"+action || r.Header.Get("Authorization") != "Bearer identity" || r.Header.Get("X-Serverless-Authorization") != "Bearer identity" {
					t.Fatalf("wrong recovery endpoint or identity: %s", r.URL)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["AccountID"] != "anthropic-api-credits" || body["GrantID"] != "grant-a" || body["Evidence"] != "retained evidence" || body["Operator"] != "" {
					t.Fatal(body)
				}
				if action == "conservative-debit" {
					if body["RequestID"] != "request-a" || body["ExpectedAmount"] != float64(1320000) {
						t.Fatal(body)
					}
				} else if body["DebitID"] != "receipt-a" || body["Amount"] != float64(1320000) || body["Kind"] != "provider-verified" {
					t.Fatal(body)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Settled":2550044,"ConservativeDebited":2550040,"CanaryGatewaySettled":4}`))}, nil
			})}
			args := []string{action, "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "grant-a", "--amount-usd", "1.32", "--evidence", "retained evidence", "--yes"}
			if action == "conservative-debit" {
				args = append(args, "--request-id", "request-a")
			} else {
				args = append(args, "--debit-id", "receipt-a", "--kind", "provider-verified")
			}
			var out strings.Builder
			if err := runCoordinatorCredits(args, strings.NewReader(""), &out, client, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || !strings.Contains(out.String(), "Booked settled: $2.550044") || !strings.Contains(out.String(), "Conservative debit subset: $2.550040") || !strings.Contains(out.String(), "verified gateway settled: $0.000004") {
				t.Fatal(calls, out.String())
			}
		})
	}
}

func TestCreditRecoveryRequiresCompleteExplicitFlagsBeforeNetwork(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: creditTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })}
	token := func(context.Context, string) (string, error) { calls++; return "identity", nil }
	for _, action := range []string{"conservative-debit", "external-debit"} {
		args := []string{action, "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "grant-a", "--amount-usd", "1.32", "--evidence", "retained evidence", "--yes"}
		if action == "conservative-debit" {
			args = append(args, "--request-id", "request-a")
		} else {
			args = append(args, "--debit-id", "receipt-a", "--kind", "conservative")
		}
		for _, missing := range []string{"--grant-id", "--amount-usd", "--evidence", "--yes"} {
			var trimmed []string
			for i := 0; i < len(args); i++ {
				if args[i] == missing {
					if missing != "--yes" {
						i++
					}
					continue
				}
				trimmed = append(trimmed, args[i])
			}
			if err := runCoordinatorCredits(trimmed, strings.NewReader(""), io.Discard, client, token); err == nil {
				t.Fatalf("%s accepted missing %s", action, missing)
			}
		}
	}
	for _, args := range [][]string{
		{"requests", "--remote", "gcp", "--gateway", "https://gateway"},
		{"requests", "--remote", "gcp", "--gateway", "https://gateway", "--task-id", "bad/escape"},
		{"external-debit", "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "g", "--debit-id", "d", "--amount-usd", "1", "--evidence", "proof", "--kind", "estimated", "--yes"},
		{"conservative-debit", "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "g", "--amount-usd", "1", "--evidence", "proof", "--yes"},
	} {
		if err := runCoordinatorCredits(args, strings.NewReader(""), io.Discard, client, token); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid flags authenticated or made %d calls", calls)
	}
}

func TestCreditRequestsUsesBoundedOperatorQueryAndJSON(t *testing.T) {
	client := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.Path != "/admin/credits/anthropic-api-credits/requests" || r.URL.RawQuery != "task_id=task-a" || r.Header.Get("Authorization") != "Bearer identity" || r.Header.Get("X-Serverless-Authorization") != "Bearer identity" {
			t.Fatal(r.URL, r.Method)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"RequestID":"request-a","Amount":1320000,"State":"unresolved","Actual":0,"UpstreamID":""}]`))}, nil
	})}
	var out strings.Builder
	if err := runCoordinatorCredits([]string{"requests", "--remote", "gcp", "--gateway", "https://gateway", "--task-id", "task-a", "--json"}, strings.NewReader(""), &out, client, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"Amount":1320000`) {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := runCoordinatorCredits([]string{"requests", "--remote", "gcp", "--gateway", "https://gateway", "--task-id", "task-a"}, strings.NewReader(""), &out, client, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "original reservation: $1.320000") || !strings.Contains(out.String(), "state: unresolved") {
		t.Fatal(out.String())
	}
}

func TestCreditRecoveryNeverFollowsOperatorRedirect(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: creditTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"https://other.test"}}, Body: io.NopCloser(strings.NewReader("redirect"))}, nil
	})}
	err := runCoordinatorCredits([]string{"external-debit", "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "g", "--debit-id", "d", "--amount-usd", "0.000004", "--kind", "provider-verified", "--evidence", "receipt", "--yes"}, strings.NewReader(""), io.Discard, client, func(context.Context, string) (string, error) { return "identity", nil })
	if err == nil || calls != 1 {
		t.Fatal(calls, err)
	}
}
