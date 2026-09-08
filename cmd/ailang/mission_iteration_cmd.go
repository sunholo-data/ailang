package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

type missionIterationExitError struct {
	code  int
	cause error
}

func (e *missionIterationExitError) Error() string { return e.cause.Error() }
func (e *missionIterationExitError) Unwrap() error { return e.cause }
func (e *missionIterationExitError) ExitCode() int { return e.code }
func missionErrorExitCode(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}
func iterationExit(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("mission iteration stopped (exit %d)", code)
	}
	return &missionIterationExitError{code, err}
}

// Dependencies are injectable by hermetic tests, never through CLI DB overrides.
type missionIterationDeps struct {
	BindingPath string
	Registry    func() (*mission.Registry, error)
	Models      *modelreg.ModelsConfig
	Factory     dispatch.Factory
	Admit       func(context.Context, dispatch.Candidate) (dispatch.Admission, error)
}

func missionIterationCommand(verb string, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return runMissionIteration(ctx, verb, args, os.Stdout, missionIterationDeps{})
}

type iterationFlags struct {
	name, work string
	activation string
	dry, json  bool
	version    int64
}

func parseIterationFlags(verb string, args []string, out io.Writer) (iterationFlags, error) {
	var o iterationFlags
	if verb != "iterate" && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.name, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("mission "+verb, flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&o.work, "work-item", "", "work item JSON file (iterate), stable ID otherwise")
	switch verb {
	case "iterate":
		fs.BoolVar(&o.dry, "dry-run", false, "validate and resolve without state writes or provider calls")
	case "status":
		fs.StringVar(&o.activation, "activation", "", "read retained binding for this operation; status only")
		fs.BoolVar(&o.json, "json", false, "machine-readable status without prompts or credentials")
	case "resume":
	case "cancel":
		fs.Int64Var(&o.version, "version", 0, "expected status version (required)")
	default:
		return o, fmt.Errorf("unknown iteration command %q", verb)
	}
	// Reject repeated flags instead of letting the last value silently win.
	seen := map[string]bool{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			name := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)[0]
			if seen[name] {
				return o, fmt.Errorf("duplicate flag %q", name)
			}
			seen[name] = true
		}
	}
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() != 0 || o.work == "" || (verb != "iterate" && o.name == "") || (verb == "cancel" && o.version < 1) {
		return o, fmt.Errorf("usage: ailang mission %s %s--work-item %s", verb, map[bool]string{true: "NAME ", false: ""}[verb != "iterate"], map[bool]string{true: "FILE [--dry-run]", false: "ID"}[verb == "iterate"])
	}
	return o, nil
}

func runMissionIteration(ctx context.Context, verb string, args []string, out io.Writer, deps missionIterationDeps) error {
	opts, err := parseIterationFlags(verb, args, out)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return iterationExit(2, err)
	}
	binding, err := resolveMissionReadBinding(deps.BindingPath, opts.activation, opts.name, opts.work)
	if err != nil {
		return iterationExit(2, err)
	}
	key := coordinator.MissionWorkItemKey{MissionID: opts.name, WorkItemID: opts.work}
	var spec iteration.Spec
	if verb == "iterate" {
		f, e := os.Open(opts.work)
		if e != nil {
			return iterationExit(2, e)
		}
		spec, err = iteration.Decode(f)
		err = errors.Join(err, f.Close())
		if err != nil {
			return iterationExit(2, err)
		}
		key = coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
	}
	var saved *coordinator.MissionWorkItem
	// All inspection uses read-only open. Missing state is allowed only for iterate.
	ro, err := coordinator.OpenMissionReadOnlyStore(binding.StateDB)
	if err == nil {
		saved, err = ro.GetMissionWorkItem(ctx, key)
		if verb == "status" && err == nil {
			err = writeIterationStatus(ctx, out, ro, saved, opts.json)
			return errors.Join(err, ro.Close())
		}
		closeErr := ro.Close()
		if closeErr != nil {
			return iterationExit(2, closeErr)
		}
	}
	if err != nil && !(verb == "iterate" && (errors.Is(err, os.ErrNotExist) || errors.Is(err, sql.ErrNoRows))) {
		return iterationExit(2, err)
	}
	if verb == "status" {
		return iterationExit(2, fmt.Errorf("work item not found"))
	}
	if runtime.GOOS == "windows" && !opts.dry {
		return iterationExit(2, fmt.Errorf("mission iteration execution requires macOS/Linux; Windows descendant cleanup and receipt sync are unsupported"))
	}
	var snapshot iteration.Snapshot
	if saved != nil {
		if err = json.Unmarshal([]byte(saved.SpecJSON), &snapshot); err != nil {
			return iterationExit(2, err)
		}
		if verb == "iterate" && spec.Digest() != snapshot.Spec.Digest() {
			return iterationExit(2, fmt.Errorf("work item input changed; supply a reviewed successor"))
		}
		spec = snapshot.Spec
	}
	if verb == "cancel" {
		store, e := coordinator.NewSQLiteStore(binding.StateDB)
		if e != nil {
			return iterationExit(2, e)
		}
		defer store.Close()
		if e = store.CancelMissionWorkItem(ctx, key, opts.version); e != nil {
			return iterationExit(3, e)
		}
		item, e := store.GetMissionWorkItem(ctx, key)
		if e != nil {
			return iterationExit(5, e)
		}
		if e = writeIterationStatus(ctx, out, store, item, true); e != nil {
			return iterationExit(5, e)
		}
		return iterationOutcome(item, nil)
	}
	service := &iteration.Service{Models: deps.Models, Factory: deps.Factory, Admit: deps.Admit, WorkspaceRoot: binding.WorkspaceRoot}
	if saved != nil {
		service.Models = snapshot.Models
		service.RepositoryPath = snapshot.RepositoryPath
		root, e := filepath.EvalSymlinks(binding.WorkspaceRoot)
		if e != nil || filepath.Clean(root) != snapshot.WorkspaceRoot {
			return iterationExit(2, fmt.Errorf("saved workspace placement is unavailable or changed; restore the admitted binding: %v", e))
		}
	} else {
		load := deps.Registry
		if load == nil {
			load = loadMissionRegistry
		}
		reg, e := load()
		if e != nil {
			return iterationExit(2, e)
		}
		m, ok := reg.Get(spec.MissionID)
		if !ok {
			return iterationExit(2, fmt.Errorf("unregistered mission %q", spec.MissionID))
		}
		service.RepositoryPath = m.Workdir
		if service.Models == nil {
			if e = modelreg.InitModelsConfig(); e != nil {
				return iterationExit(2, e)
			}
			service.Models = modelreg.GlobalModelsConfig
		}
	}
	if opts.dry || saved == nil {
		plan, e := service.Plan(ctx, spec)
		if e != nil {
			return iterationExit(2, e)
		}
		if opts.dry {
			return json.NewEncoder(out).Encode(plan)
		}
	}
	if service.Factory == nil {
		service.Factory = executor.GlobalFactory()
	}
	if service.Admit == nil {
		service.Admit = mission.NewAdmissionPolicy(mission.DefaultPaths()).Check
	}
	store, err := coordinator.NewSQLiteStore(binding.StateDB)
	if err != nil {
		return iterationExit(2, err)
	}
	defer store.Close()
	service.Store = store
	item, runErr := service.Run(ctx, spec)
	if item != nil {
		if err = writeIterationStatus(ctx, out, store, item, true); err != nil {
			return iterationExit(5, errors.Join(runErr, err))
		}
	}
	return iterationOutcome(item, runErr)
}

func iterationOutcome(item *coordinator.MissionWorkItem, err error) error {
	if item != nil {
		switch item.State {
		case "completed":
			if err == nil {
				return nil
			}
		case "needs_reconciliation":
			return iterationExit(4, err)
		case "cancelled":
			return iterationExit(130, err)
		case "failed":
			return iterationExit(5, err)
		case "waiting", "ready", "running", "validating":
			return iterationExit(3, err)
		}
	}
	if errors.Is(err, context.Canceled) {
		return iterationExit(130, err)
	}
	if errors.Is(err, coordinator.ErrMissionAttemptConflict) {
		return iterationExit(3, err)
	}
	return iterationExit(5, err)
}
