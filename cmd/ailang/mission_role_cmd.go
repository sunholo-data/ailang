package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

func missionRoleRun(args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return runMissionRole(ctx, args, os.Stdout, nil, nil)
}

// runMissionRole accepts an explicit attended request, never an inbox message.
func runMissionRole(ctx context.Context, args []string, out io.Writer, models *modelreg.ModelsConfig, factory dispatch.Factory) error {
	fs := flag.NewFlagSet("mission role-run", flag.ContinueOnError)
	fs.SetOutput(out)
	requestPath := fs.String("request", "", "versioned role request JSON file (required)")
	receiptPath := fs.String("receipt", "", "new exclusive JSONL receipt path (required for execution)")
	stateDB := fs.String("state-db", "", "opt-in durable attempt fencing in an explicit coordinator SQLite DB")
	dryRun := fs.Bool("dry-run", false, "resolve models without probing or executing providers")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 || *requestPath == "" || (!*dryRun && *receiptPath == "") {
		return fmt.Errorf("usage: ailang mission role-run --request FILE --receipt NEW_FILE [--state-db FILE] [--dry-run]")
	}
	f, err := os.Open(*requestPath)
	if err != nil {
		return err
	}
	req, err := dispatch.DecodeRequest(f)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if models == nil {
		if err := modelreg.InitModelsConfig(); err != nil {
			return err
		}
		models = modelreg.GlobalModelsConfig
	}
	plan, err := dispatch.Resolve(req, models)
	if err != nil {
		return err
	}
	if *dryRun {
		return json.NewEncoder(out).Encode(plan)
	}
	if factory == nil {
		factory = executor.GlobalFactory()
	}
	if *stateDB != "" {
		report, runErr := runDurableMissionRole(ctx, req, *receiptPath, *stateDB, models, factory)
		if report != nil {
			return errors.Join(runErr, json.NewEncoder(out).Encode(report))
		}
		return runErr
	}
	journal, err := dispatch.OpenJournal(*receiptPath)
	if err != nil {
		return err
	}
	if factory == nil {
		factory = executor.GlobalFactory()
	}
	runner := dispatch.Runner{Models: models, Executors: factory, Record: journal.Record}
	report, runErr := runner.Run(ctx, req)
	closeErr = journal.Close()
	var outputErr error
	if report != nil {
		outputErr = json.NewEncoder(out).Encode(report)
	}
	return errors.Join(runErr, closeErr, outputErr)
}
