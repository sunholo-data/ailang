package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/claudegateway"
	"github.com/sunholo-data/ailang/internal/creditbudget"
	"github.com/sunholo-data/ailang/internal/modelreg"
	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
	"google.golang.org/api/idtoken"
)

func creditGatewayCommand(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		fmt.Println("Usage: ailang credit-gateway serve --ledger-project <isolated-project> --gateway-url <https-url> --provider-key-file <mount> --signing-key-file <mount> --organization <org> --workspace <workspace> --operators <emails> --coordinators <coordinator=job,...> [--listen :8080]")
		return nil
	}
	if args[0] != "serve" {
		return errors.New("credit-gateway supports serve only")
	}
	flags := flag.NewFlagSet("credit-gateway serve", flag.ContinueOnError)
	project := flags.String("ledger-project", "", "Private credit authority Firestore project; ordinary executors must have no access")
	origin := flags.String("gateway-url", "", "HTTPS gateway URL and Google ID-token audience")
	providerFile := flags.String("provider-key-file", "", "Gateway-only Secret Manager key mount")
	signingFile := flags.String("signing-key-file", "", "Gateway-only 32-byte signing key mount")
	organization := flags.String("organization", "", "Verified Anthropic credit organization")
	workspace := flags.String("workspace", "", "Verified API-key workspace")
	operators := flags.String("operators", "", "Comma-separated operator identity emails")
	coordinators := flags.String("coordinators", "", "Comma-separated coordinator-email=job-email pairs")
	listen := flags.String("listen", ":8080", "HTTP listen address behind authenticated Cloud Run")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected credit-gateway arguments")
	}
	u, err := url.Parse(*origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("gateway-url must be an HTTPS origin")
	}
	if *project == "" || *organization == "" || *workspace == "" || *operators == "" || *coordinators == "" {
		return errors.New("ledger project, organization/workspace and identity allowlists required")
	}
	key, err := os.ReadFile(*providerFile)
	if err != nil {
		return errors.New("cannot read provider key mount")
	}
	if len(strings.TrimSpace(string(key))) < 16 {
		return errors.New("invalid provider key mount")
	}
	signing, err := os.ReadFile(*signingFile)
	if err != nil || len(signing) < 32 || len(signing) > 4096 {
		return errors.New("invalid signing key mount (32+ raw bytes required)")
	}
	if err = modelreg.InitModelsConfig(); err != nil {
		return err
	}
	ctx := context.Background()
	client, err := fsstore.NewClientForProject(ctx, *project)
	if err != nil {
		return err
	}
	defer client.Close()
	authority := fsstore.NewCreditAuthority(client)
	if err = authority.Configure(ctx, creditbudget.Policy{AccountID: claudegateway.AccountID, Organization: *organization, Workspace: *workspace, OperatingCeiling: 190 * creditbudget.USD, DailyCeiling: 6 * creditbudget.USD, TaskCeiling: 2 * creditbudget.USD, MaxTasks: 2}); err != nil {
		return err
	}
	h := &claudegateway.Gateway{Authority: authority, ProviderKey: strings.TrimSpace(string(key)), SigningKey: signing, PublicURL: strings.TrimRight(*origin, "/"), Operators: map[string]bool{}, Coordinators: map[string]string{}}
	for _, identity := range strings.Split(*operators, ",") {
		if identity = strings.TrimSpace(identity); identity != "" {
			h.Operators[identity] = true
		}
	}
	for _, pair := range strings.Split(*coordinators, ",") {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return errors.New("invalid coordinator=job identity binding")
		}
		h.Coordinators[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	h.Authenticate = func(ctx context.Context, token string) (string, error) {
		payload, err := idtoken.Validate(ctx, token, strings.TrimRight(*origin, "/"))
		if err != nil {
			return "", errors.New("invalid Google ID token")
		}
		email, ok := payload.Claims["email"].(string)
		verified, _ := payload.Claims["email_verified"].(bool)
		if !ok || email == "" || !verified {
			return "", errors.New("verified identity email required")
		}
		return email, nil
	}
	server := &http.Server{Addr: *listen, Handler: h, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 11 * time.Minute, IdleTimeout: time.Minute, MaxHeaderBytes: 16 << 10}
	return server.ListenAndServe()
}
