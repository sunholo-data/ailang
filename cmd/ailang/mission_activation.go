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
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/sunholo-data/ailang/internal/mission/activation"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

type missionActivationDeps struct {
	Home           string
	LegacyIdle     func(context.Context, string) error
	SessionStopped func(context.Context, int) error
	ProcessAlive   func(int) (bool, error)
	RunChild       func(context.Context, string, *activationProcess, io.Writer) error
}
type activationOptions struct{ verb, name, operation, work, binding string }

var activationID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

func missionActivationCommand(args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return runMissionActivation(ctx, args, os.Stdout, missionActivationDeps{})
}
func parseActivationOptions(args []string, out io.Writer) (o activationOptions, err error) {
	if len(args) < 2 {
		return o, errors.New("usage: ailang mission activation run|inspect|recover docs --operation ID [--work-item FILE --binding FILE]")
	}
	o.verb, o.name = args[0], args[1]
	if o.name != "docs" {
		return o, errors.New("activation currently supports local Docs only")
	}
	fs := flag.NewFlagSet("mission activation "+o.verb, flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&o.operation, "operation", "", "owned operation ID")
	switch o.verb {
	case "run":
		fs.StringVar(&o.work, "work-item", "", "approved work item JSON file")
		fs.StringVar(&o.binding, "binding", "", "strict canary runtime TOML file")
	case "inspect", "recover", "child":
	default:
		return o, errors.New("unknown activation operation")
	}
	seen := map[string]bool{}
	for _, arg := range args[2:] {
		if strings.HasPrefix(arg, "-") {
			key := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)[0]
			if seen[key] {
				return o, fmt.Errorf("duplicate flag %s", key)
			}
			seen[key] = true
		}
	}
	if err = fs.Parse(args[2:]); err != nil {
		return o, err
	}
	if fs.NArg() != 0 || !activationID.MatchString(o.operation) || o.verb == "run" && (o.work == "" || o.binding == "") {
		return o, errors.New("operation ID required; run also requires work-item and binding files")
	}
	return o, nil
}
func runMissionActivation(ctx context.Context, args []string, out io.Writer, d missionActivationDeps) error {
	o, err := parseActivationOptions(args, out)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return iterationExit(2, err)
	}
	if d.Home == "" {
		d.Home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	dir := filepath.Join(d.Home, ".ailang", "state", "mission-activations")
	m := activation.Manager{Dir: dir}
	if o.verb == "child" {
		return runActivationChild(ctx, dir, o.operation, out)
	}
	if o.verb == "inspect" {
		r, e := m.Inspect(o.operation)
		if e != nil {
			return e
		}
		p, processErr := readActivationProcess(dir, o.operation)
		status := struct {
			Installation activation.Record  `json:"installation"`
			Process      *activationProcess `json:"process,omitempty"`
			ProcessError string             `json:"process_error,omitempty"`
			NextAction   string             `json:"next_action"`
		}{Installation: r, NextAction: "Inspect retained runtime evidence with mission status --activation " + o.operation}
		if processErr == nil {
			status.Process = &p
		} else {
			status.ProcessError = processErr.Error()
		}
		if r.Phase != "restored" {
			status.NextAction = "After verifying owned execution stopped, run mission activation recover docs --operation " + o.operation
		}
		return json.NewEncoder(out).Encode(status)
	}
	if d.LegacyIdle == nil {
		d.LegacyIdle = activationLegacyIdle
	}
	if d.ProcessAlive == nil {
		d.ProcessAlive = activationProcessAlive
	}
	if d.SessionStopped == nil {
		d.SessionStopped = activationSessionStopped
	}
	if o.verb == "recover" {
		r, e := m.Recover(ctx, o.operation, activationStopVerifier(dir, d, false))
		if r.OperationID != "" {
			e = errors.Join(e, json.NewEncoder(out).Encode(r))
		}
		return e
	}
	f, err := os.Open(o.work)
	if err != nil {
		return err
	}
	spec, err := iteration.Decode(f)
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if spec.MissionID != "docs" {
		return errors.New("work item must target docs")
	}
	if _, err = iteration.LoadBinding(o.binding); err != nil {
		return err
	}
	binding, err := os.ReadFile(o.binding)
	if err != nil {
		return err
	}
	work := filepath.Join(dir, o.operation+".work-item.json")
	// One implementation, build-tagged in internal/mission/activation: this was an
	// un-tagged inline copy, and on Windows MkdirAll(dir, 0700) does not yield 0700, so it
	// rejected every directory it had just created.
	if err = activation.EnsurePrivateDir(dir); err != nil {
		return err
	}
	frozen, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	frozenFile, err := os.OpenFile(work, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return err
	}
	_, err = frozenFile.Write(frozen)
	err = errors.Join(err, frozenFile.Sync(), frozenFile.Close())
	if err != nil {
		return err
	}
	if err = activationSyncDir(dir); err != nil {
		return err
	}
	p := activationProcess{Version: 1, OperationID: o.operation, WorkItemID: spec.WorkItemID, WorkFile: work, SpecDigest: spec.Digest(), LauncherPID: os.Getpid(), Phase: "launching"}
	if err = createActivationProcess(dir, p); err != nil {
		return err
	}
	q := activation.Request{OperationID: o.operation, MissionID: "docs", WorkItemID: spec.WorkItemID, MarkerPath: filepath.Join(d.Home, ".ailang", "state", "mission-docs.disabled"), BindingPath: filepath.Join(d.Home, ".config", "ailang", "mission-runtime.toml"), Binding: binding}
	_, err = m.Activate(ctx, q, func(ctx context.Context, _ activation.Record) error { return d.LegacyIdle(ctx, d.Home) })
	if err == nil {
		run := d.RunChild
		if run == nil {
			run = superviseActivationChild
		}
		err = run(ctx, dir, &p, out)
	}
	// Cancellation must not prevent a read-only stop check and safe cleanup.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), activationCheckTimeout)
	defer cancel()
	r, cleanupErr := m.Recover(cleanupCtx, o.operation, activationStopVerifier(dir, d, true))
	if r.OperationID != "" {
		cleanupErr = errors.Join(cleanupErr, json.NewEncoder(out).Encode(r))
	}
	return errors.Join(err, cleanupErr)
}
