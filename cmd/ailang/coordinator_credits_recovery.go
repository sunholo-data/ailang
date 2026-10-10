package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

func creditIdentifier(id string) bool {
	return id != "" && len(id) <= 256 && id != "." && id != ".." && !strings.Contains(id, "/") && strings.TrimSpace(id) == id && strings.IndexFunc(id, unicode.IsControl) < 0
}

func creditRecoveryPayload(action, account, grant, request, debit, amount, kind, evidence string, yes bool) ([]byte, error) {
	if !yes || !creditIdentifier(grant) || strings.TrimSpace(evidence) == "" {
		return nil, fmt.Errorf("recovery requires explicit --grant-id, --evidence and --yes; review disabled canonical status first")
	}
	value, err := parseCreditUSD(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid --amount-usd: %w", err)
	}
	if action == "conservative-debit" {
		if !creditIdentifier(request) {
			return nil, fmt.Errorf("--request-id is required for the original unresolved reservation")
		}
		return json.Marshal(creditbudget.ConservativeDebit{AccountID: account, GrantID: grant, RequestID: request, ExpectedAmount: value, Evidence: evidence})
	}
	if !creditIdentifier(debit) || (kind != "conservative" && kind != "provider-verified") {
		return nil, fmt.Errorf("external recovery requires --debit-id and --kind conservative|provider-verified")
	}
	return json.Marshal(creditbudget.ExternalDebit{AccountID: account, GrantID: grant, DebitID: debit, Amount: value, Kind: kind, Evidence: evidence})
}

func printCreditRequests(out io.Writer, data []byte, asJSON bool) error {
	var requests []creditbudget.Request
	if err := json.Unmarshal(data, &requests); err != nil || len(requests) > 100 {
		return fmt.Errorf("invalid or truncated credit authority request listing")
	}
	if asJSON {
		_, err := fmt.Fprintln(out, string(data))
		return err
	}
	for _, r := range requests {
		fmt.Fprintf(out, "Request: %s; task: %s; state: %s; original reservation: $%.6f; recorded actual: $%.6f; upstream ID: %s\n", r.RequestID, r.TaskID, r.State, float64(r.Amount)/1e6, float64(r.Actual)/1e6, r.UpstreamID)
	}
	return nil
}
