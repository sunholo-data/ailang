package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/creditbudget"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/impersonate"
)

func coordinatorCredits(args []string) error {
	return runCoordinatorCredits(args, os.Stdin, os.Stdout, &http.Client{Timeout: 30 * time.Second}, func(ctx context.Context, audience string) (string, error) {
		source, err := idtoken.NewTokenSource(ctx, audience)
		if err != nil {
			return "", err
		}
		tok, err := source.Token()
		if err != nil {
			return "", err
		}
		return tok.AccessToken, nil
	})
}

var creditDollarPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,6})?$`)

func parseCreditUSD(raw string) (creditbudget.MicroUSD, error) {
	if !creditDollarPattern.MatchString(raw) {
		return 0, fmt.Errorf("amount must be positive USD with at most six decimal places")
	}
	parts := strings.SplitN(raw, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > 200 {
		return 0, fmt.Errorf("amount exceeds $200 pilot maximum")
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	frac += strings.Repeat("0", 6-len(frac))
	micro, _ := strconv.ParseInt(frac, 10, 64)
	result := creditbudget.MicroUSD(whole*1000000 + micro)
	if result <= 0 || result > creditbudget.MaximumGrant {
		return 0, fmt.Errorf("amount must be between zero and $200")
	}
	return result, nil
}
func runCoordinatorCredits(args []string, in io.Reader, out io.Writer, client *http.Client, token func(context.Context, string) (string, error)) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "help" || args[0] == "-h") {
		fmt.Fprintln(out, "Usage: ailang coordinator credits confirm|status|enable|disable|promote --remote gcp --gateway HTTPS_URL\nGateway default: AILANG_CLAUDE_CREDIT_GATEWAY_URL\nconfirm: --account --grant-id --expected-grant-id --amount-usd --available-usd --starts-at --expires-at --evidence [--yes] [--json]\nstatus: --account [--json]\nCommon: --impersonate-service-account OPERATOR_SA (ADC caller needs scoped Token Creator)\nenable|disable|promote: --evidence (required); enable/promote also require --yes\nDates are RFC3339. Grant evidence confirms fresh allocation; calendar rollover never renews credits.")
		return nil
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: coordinator credits confirm|status|enable|disable|promote --remote gcp --gateway HTTPS_URL [options]")
	}
	action := args[0]
	if action != "status" && action != "confirm" && action != "enable" && action != "disable" && action != "promote" {
		return fmt.Errorf("unknown credits action %q", action)
	}
	fs := flag.NewFlagSet("coordinator credits "+action, flag.ContinueOnError)
	fs.SetOutput(out)
	remote := fs.String("remote", "", "canonical credit authority (required: gcp)")
	gateway := fs.String("gateway", config.ClaudeCreditGatewayURL(), "authenticated HTTPS credit gateway")
	account := fs.String("account", "anthropic-api-credits", "credit account")
	grant := fs.String("grant-id", "", "provider grant or stable organization/cycle identifier")
	expected := fs.String("expected-grant-id", "", "previous grant ID (empty for first grant)")
	amount := fs.String("amount-usd", "", "confirmed grant amount")
	available := fs.String("available-usd", "", "available exclusively allocated credits")
	starts := fs.String("starts-at", "", "grant start, RFC3339")
	expires := fs.String("expires-at", "", "grant expiry, RFC3339")
	evidence := fs.String("evidence", "", "reference to operator-verified grant and allocation evidence")
	operatorSA := fs.String("impersonate-service-account", "", "mint the operator ID token through an allowlisted service account; requires scoped Token Creator")
	yes := fs.Bool("yes", false, "confirm displayed grant without final prompt")
	jsonOut := fs.Bool("json", false, "print full canonical JSON status")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected credits arguments: %v", fs.Args())
	}
	if *operatorSA != "" {
		if !regexp.MustCompile(`^[A-Za-z0-9_-]+@[A-Za-z0-9.-]+\.iam\.gserviceaccount\.com$`).MatchString(*operatorSA) {
			return fmt.Errorf("invalid operator service-account email")
		}
		token = func(ctx context.Context, audience string) (string, error) {
			source, err := impersonate.IDTokenSource(ctx, impersonate.IDTokenConfig{TargetPrincipal: *operatorSA, Audience: audience, IncludeEmail: true})
			if err != nil {
				return "", fmt.Errorf("operator impersonation requires scoped Token Creator: %w", err)
			}
			t, err := source.Token()
			if err != nil {
				return "", err
			}
			return t.AccessToken, nil
		}
	}
	if *remote != "gcp" {
		return fmt.Errorf("credits requires --remote gcp; local credits are not authoritative")
	}
	base, err := url.Parse(*gateway)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.Path != "" {
		return fmt.Errorf("--gateway must be an HTTPS origin without credentials, path, query or fragment")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(*account) {
		return fmt.Errorf("invalid credit account")
	}
	var body []byte
	endpoint := action
	if action == "enable" || action == "disable" || action == "promote" {
		if strings.TrimSpace(*evidence) == "" {
			return fmt.Errorf("--evidence is required for audited account enable/disable")
		}
		if (action == "enable" || action == "promote") && !*yes {
			return fmt.Errorf("enable/promote requires --yes after reviewing canonical credits status")
		}
		body, err = json.Marshal(struct {
			Enabled  bool
			Evidence string
		}{Enabled: action == "enable", Evidence: *evidence})
		if err != nil {
			return err
		}
		endpoint = "enabled"
		if action == "promote" {
			body, err = json.Marshal(struct{ Evidence string }{Evidence: *evidence})
			if err != nil {
				return err
			}
			endpoint = "promote"
		}
	}
	if action == "confirm" {
		reader := bufio.NewReader(in)
		ask := func(label string, value *string) error {
			if *value != "" {
				return nil
			}
			fmt.Fprintf(out, "%s: ", label)
			s, e := reader.ReadString('\n')
			if e != nil {
				return fmt.Errorf("missing %s; supply explicit flags: %w", label, e)
			}
			*value = strings.TrimSpace(s)
			if *value == "" {
				return fmt.Errorf("%s cannot be empty", label)
			}
			return nil
		}
		for _, p := range []struct {
			label string
			value *string
		}{{"Grant identifier", grant}, {"Confirmed amount USD", amount}, {"Available allocated USD", available}, {"Starts at RFC3339", starts}, {"Expires at RFC3339", expires}, {"Evidence reference", evidence}} {
			if err := ask(p.label, p.value); err != nil {
				return err
			}
		}
		a, err := parseCreditUSD(*amount)
		if err != nil {
			return err
		}
		v, err := parseCreditUSD(*available)
		if err != nil {
			return err
		}
		if v > a {
			return fmt.Errorf("available credits exceed confirmed grant")
		}
		start, err := time.Parse(time.RFC3339, *starts)
		if err != nil {
			return fmt.Errorf("invalid starts-at: %w", err)
		}
		end, err := time.Parse(time.RFC3339, *expires)
		if err != nil || !end.After(start) {
			return fmt.Errorf("expires-at must be RFC3339 and later than starts-at")
		}
		confirmation := creditbudget.Confirmation{AccountID: *account, GrantID: *grant, Amount: a, Available: v, StartsAt: start, ExpiresAt: end, Evidence: *evidence, ExpectedGrantID: *expected}
		fmt.Fprintf(out, "Authority: %s\nAccount: %s\nPrevious grant: %s\nConfirm grant: %s; amount $%s; available $%s; expiry %s\nThis records verified credits; it does not enable a disabled account.\n", *gateway, *account, *expected, *grant, *amount, *available, *expires)
		// Read the current canonical allowance before the operator confirms a change.
		previewToken, err := token(context.Background(), *gateway)
		if err != nil {
			return fmt.Errorf("authenticate credit operator (use --impersonate-service-account with ordinary user ADC): %w", err)
		}
		previewReq, err := http.NewRequest(http.MethodGet, *gateway+"/admin/credits/"+*account+"/status", nil)
		if err != nil {
			return err
		}
		previewReq.Header.Set("Authorization", "Bearer "+previewToken)
		previewReq.Header.Set("X-Serverless-Authorization", "Bearer "+previewToken)
		previewClient := *client
		previewClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		previewResponse, err := previewClient.Do(previewReq)
		if err != nil {
			return fmt.Errorf("read canonical credit status: %w", err)
		}
		previewData, readErr := io.ReadAll(io.LimitReader(previewResponse.Body, 1<<20))
		previewResponse.Body.Close()
		if readErr != nil {
			return readErr
		}
		if previewResponse.StatusCode != http.StatusOK {
			return fmt.Errorf("cannot confirm without canonical status (%d): %s", previewResponse.StatusCode, previewData)
		}
		var current creditbudget.Status
		if err = json.Unmarshal(previewData, &current); err != nil {
			return err
		}
		if current.Grant.GrantID != *expected {
			return fmt.Errorf("stale grant expectation: canonical current grant is %q; pass --expected-grant-id after reviewing status", current.Grant.GrantID)
		}
		fmt.Fprintf(out, "Canonical organization: %s; workspace: %s; current grant: %s; current available: $%.6f; reserved: $%.6f; unresolved: $%.6f; blockers: %s\n", current.Organization, current.Workspace, current.Grant.GrantID, float64(current.Available)/1e6, float64(current.Reserved)/1e6, float64(current.Unresolved)/1e6, strings.Join(current.Blockers, "; "))
		if !*yes {
			fmt.Fprint(out, "Confirm fresh credits [yes/no]: ")
			answer, err := reader.ReadString('\n')
			if err != nil || strings.TrimSpace(answer) != "yes" {
				return fmt.Errorf("credit confirmation cancelled")
			}
		}
		body, err = json.Marshal(confirmation)
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	identity, err := token(ctx, *gateway)
	if err != nil {
		return fmt.Errorf("authenticate credit operator (use --impersonate-service-account with ordinary user ADC): %w", err)
	}
	if identity == "" {
		return fmt.Errorf("empty operator identity token")
	}
	method := http.MethodGet
	if action != "status" {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, *gateway+"/admin/credits/"+*account+"/"+endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+identity)
	req.Header.Set("X-Serverless-Authorization", "Bearer "+identity)
	req.Header.Set("Content-Type", "application/json")
	// Never follow a redirect carrying the operator's bearer identity.
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := copyClient.Do(req)
	if err != nil {
		return fmt.Errorf("credit authority unavailable: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("credit authority rejected %s (%d): %s", action, response.StatusCode, strings.TrimSpace(string(data)))
	}
	var status creditbudget.Status
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("invalid credit authority status: %w", err)
	}
	if *jsonOut {
		_, err = fmt.Fprintln(out, string(data))
		return err
	}
	fmt.Fprintf(out, "Account: %s; organization: %s; workspace: %s\nGrant: %s; expires: %s\nConfirmed: $%.6f; operating ceiling: $%.6f; available: $%.6f\nSettled: $%.6f; reserved: $%.6f; unresolved: $%.6f\nEligible: %t; blockers: %s\n", status.AccountID, status.Organization, status.Workspace, status.Grant.GrantID, status.Grant.ExpiresAt.Format(time.RFC3339), float64(status.Grant.Amount)/1e6, float64(status.OperatingCeiling)/1e6, float64(status.Available)/1e6, float64(status.Settled)/1e6, float64(status.Reserved)/1e6, float64(status.Unresolved)/1e6, status.Eligible, strings.Join(status.Blockers, "; "))
	fmt.Fprintf(out, "Forwarding exposure: $%.6f\nCanary complete: %t; settled: $%.6f; reserved: $%.6f; remaining: $%.6f\n", float64(status.Forwarding)/1e6, status.CanaryComplete, float64(status.CanarySettled)/1e6, float64(status.CanaryReserved)/1e6, float64(status.CanaryAvailable)/1e6)
	return nil
}
