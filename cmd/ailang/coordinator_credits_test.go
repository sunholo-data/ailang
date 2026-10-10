package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type creditTransport func(*http.Request) (*http.Response, error)

func (f creditTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCreditsRejectInvalidInputsBeforeNetwork(t *testing.T) {
	calls := 0
	c := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) { calls++; return nil, nil })}
	for _, args := range [][]string{{"status"}, {"status", "--remote", "local"}, {"status", "--remote", "gcp", "--gateway", "http://bad"}, {"confirm", "--remote", "gcp", "--gateway", "https://gateway", "--amount-usd", "NaN", "--yes"}} {
		if err := runCoordinatorCredits(args, strings.NewReader(""), io.Discard, c, func(context.Context, string) (string, error) { return "id-token", nil }); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid input made %d network calls", calls)
	}
}
func TestCreditsStatusUsesAuthenticatedCanonicalEndpoint(t *testing.T) {
	c := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://gateway/admin/credits/anthropic-api-credits/status" || (r.Header.Get("Authorization") != "Bearer identity" || r.Header.Get("X-Serverless-Authorization") != "Bearer identity") {
			t.Fatalf("wrong authenticated request: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Eligible":false,"Blockers":["grant expired"]}`)), Header: http.Header{}}, nil
	})}
	var out strings.Builder
	if err := runCoordinatorCredits([]string{"status", "--remote", "gcp", "--gateway", "https://gateway", "--json"}, strings.NewReader(""), &out, c, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "grant expired") {
		t.Fatal(out.String())
	}
}
func TestCreditDollarParsingExact(t *testing.T) {
	for _, s := range []string{"NaN", "Inf", "-1", "0", "1.0000001", "1junk", "1e2"} {
		if _, err := parseCreditUSD(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	if got, err := parseCreditUSD("1.000001"); err != nil || got != 1000001 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestCreditsConfirmShowsCurrentAuthorityAndUsesExactAmounts(t *testing.T) {
	calls := 0
	c := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer identity" || r.Header.Get("X-Serverless-Authorization") != "Bearer identity" {
			t.Fatal("missing operator identity")
		}
		if r.Method == http.MethodPost {
			var body map[string]any
			data, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(data, &body); err != nil {
				t.Fatal(err)
			}
			if body["Amount"] != float64(200000000) || body["Available"] != float64(190000000) || body["Operator"] != "" {
				t.Fatal(body)
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Organization":"org","Workspace":"workspace","Grant":{"GrantID":"old"},"Reserved":123,"Blockers":["disabled"]}`)), Header: http.Header{}}, nil
	})}
	var out strings.Builder
	args := []string{"confirm", "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "new", "--expected-grant-id", "old", "--amount-usd", "200", "--available-usd", "190", "--starts-at", "2026-10-01T00:00:00Z", "--expires-at", "2026-11-01T00:00:00Z", "--evidence", "console receipt and allocation", "--yes"}
	if err := runCoordinatorCredits(args, strings.NewReader(""), &out, c, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !strings.Contains(out.String(), "Canonical organization: org") {
		t.Fatal(calls, out.String())
	}
}
func TestCreditsConfirmationRejectsStaleGrantBeforeWrite(t *testing.T) {
	writes := 0
	c := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			writes++
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Grant":{"GrantID":"other"}}`))}, nil
	})}
	err := runCoordinatorCredits([]string{"confirm", "--remote", "gcp", "--gateway", "https://gateway", "--grant-id", "new", "--amount-usd", "200", "--available-usd", "190", "--starts-at", "2026-10-01T00:00:00Z", "--expires-at", "2026-11-01T00:00:00Z", "--evidence", "allocation", "--yes"}, strings.NewReader(""), io.Discard, c, func(context.Context, string) (string, error) { return "identity", nil })
	if err == nil || writes != 0 {
		t.Fatalf("stale grant wrote %d times: %v", writes, err)
	}
}

func TestCreditsDisableUsesAuditedOperatorEndpoint(t *testing.T) {
	c := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/admin/credits/anthropic-api-credits/enabled" || r.Method != http.MethodPost {
			t.Fatal(r.URL, r.Method)
		}
		var body struct {
			Enabled  bool
			Evidence string
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Enabled || body.Evidence != "incident" {
			t.Fatal(body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"Blockers":["disabled"]}`))}, nil
	})}
	if err := runCoordinatorCredits([]string{"disable", "--remote", "gcp", "--gateway", "https://gateway", "--evidence", "incident"}, strings.NewReader(""), io.Discard, c, func(context.Context, string) (string, error) { return "identity", nil }); err != nil {
		t.Fatal(err)
	}
}

func TestCreditsPromoteRequiresAuditedReconciliation(t *testing.T) {
	writes := 0
	c := &http.Client{Transport: creditTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/admin/credits/anthropic-api-credits/promote" || r.Method != http.MethodPost || r.Header.Get("X-Serverless-Authorization") != "Bearer identity" {
			t.Fatal("wrong promotion authority")
		}
		var body struct{ Evidence string }
		json.NewDecoder(r.Body).Decode(&body)
		if body.Evidence != "billing reconciled" {
			t.Fatal(body)
		}
		writes++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"CanaryComplete":true,"CanarySettled":1000,"Forwarding":0}`))}, nil
	})}
	args := []string{"promote", "--remote", "gcp", "--gateway", "https://gateway", "--evidence", "billing reconciled"}
	token := func(context.Context, string) (string, error) { return "identity", nil }
	if err := runCoordinatorCredits(args, strings.NewReader(""), io.Discard, c, token); err == nil || writes != 0 {
		t.Fatal("unauthorized promotion")
	}
	var out strings.Builder
	if err := runCoordinatorCredits(append(args, "--yes"), strings.NewReader(""), &out, c, token); err != nil {
		t.Fatal(err)
	}
	if writes != 1 || !strings.Contains(out.String(), "Canary complete: true") {
		t.Fatal(out.String())
	}
}
