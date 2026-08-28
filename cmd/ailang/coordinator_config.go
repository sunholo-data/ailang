package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"gopkg.in/yaml.v3"
)

// M-MESSAGE-PLANE-FAIL-LOUD M4: compare-and-swap for the shared coordinator
// config.
//
// The cloud config is read by every coordinator on cold start and is edited from
// more than one machine. It was written with a plain `gsutil cp`, which is
// last-writer-wins: there is no generation precondition anywhere in the repo
// (verified 2026-08-26, zero hits for if-generation-match).
//
// This is not theoretical. On 2026-08-26 a correct edit uploaded at 14:24:33Z was
// clobbered at 14:37:10Z by a copy fetched before it, restoring a byte-identical
// earlier version. Neither side saw an error, and the change was simply gone.
//
// Per CLAUDE.md Critical Principle 2, silently discarding another machine's write
// is a data-integrity fallback, not a convenience one. Refuse instead.

// configStore is the minimal surface the CAS logic needs, so the policy is
// testable without touching a bucket.
type configStore interface {
	Read() ([]byte, int64, error)
	Write(data []byte, ifGeneration int64) error
}

// staleGenerationError reports that the object changed since it was read.
type staleGenerationError struct {
	Expected int64
	Actual   int64
}

func (e *staleGenerationError) Error() string {
	return fmt.Sprintf("config changed since you fetched it (you have generation %d, live is %d): "+
		"someone else wrote in between. Re-fetch, re-apply your change, and try again — "+
		"overwriting would discard their edit", e.Expected, e.Actual)
}

// validateCoordinatorConfigBytes parses and sanity-checks a candidate config.
//
// Runs BEFORE any write: a malformed or invariant-violating config is read by
// every coordinator on its next cold start, so publishing one is strictly worse
// than refusing the edit.
func validateCoordinatorConfigBytes(data []byte) error {
	var file struct {
		Coordinator coordinator.CoordinatorConfig `yaml:"coordinator"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("config is not valid YAML: %w", err)
	}

	agents := file.Coordinator.Agents
	if len(agents) == 0 {
		return errors.New("config declares no agents: refusing to publish a config that would leave every inbox unserved")
	}

	seenID := make(map[string]bool, len(agents))
	seenInbox := make(map[string]string, len(agents))
	for _, a := range agents {
		if a.ID == "" {
			return errors.New("an agent has no id")
		}
		if seenID[a.ID] {
			return fmt.Errorf("duplicate agent id %q", a.ID)
		}
		seenID[a.ID] = true

		if a.Workspace == "" {
			return fmt.Errorf("agent %q has no workspace: a dispatched job would have nothing to clone or chdir into", a.ID)
		}
		if a.Inbox != "" {
			if prev, dup := seenInbox[a.Inbox]; dup {
				return fmt.Errorf("agents %q and %q both claim inbox %q", prev, a.ID, a.Inbox)
			}
			seenInbox[a.Inbox] = a.ID
		}
	}
	return nil
}

// writeConfigCAS validates then writes, but only if the object is still at
// ifGeneration.
func writeConfigCAS(store configStore, data []byte, ifGeneration int64) error {
	if err := validateCoordinatorConfigBytes(data); err != nil {
		return fmt.Errorf("refusing to write: %w", err)
	}
	return store.Write(data, ifGeneration)
}

// gcsConfigStore is the real store, backed by a GCS object with generation
// preconditions.
type gcsConfigStore struct {
	ctx    context.Context
	client *storage.Client
	bucket string
	object string
}

func (g *gcsConfigStore) Read() ([]byte, int64, error) {
	obj := g.client.Bucket(g.bucket).Object(g.object)
	attrs, err := obj.Attrs(g.ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("stat gs://%s/%s: %w", g.bucket, g.object, err)
	}
	r, err := obj.Generation(attrs.Generation).NewReader(g.ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("read gs://%s/%s: %w", g.bucket, g.object, err)
	}
	defer func() { _ = r.Close() }()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, fmt.Errorf("read gs://%s/%s: %w", g.bucket, g.object, err)
	}
	return data, attrs.Generation, nil
}

func (g *gcsConfigStore) Write(data []byte, ifGeneration int64) error {
	obj := g.client.Bucket(g.bucket).Object(g.object).
		If(storage.Conditions{GenerationMatch: ifGeneration})
	w := obj.NewWriter(g.ctx)
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("write gs://%s/%s: %w", g.bucket, g.object, err)
	}
	if err := w.Close(); err != nil {
		// A precondition failure is the whole point of this command: translate it
		// into the typed error so the caller can print the live generation.
		if isPreconditionFailure(err) {
			_, live, rerr := g.Read()
			if rerr != nil {
				live = -1
			}
			return &staleGenerationError{Expected: ifGeneration, Actual: live}
		}
		return fmt.Errorf("write gs://%s/%s: %w", g.bucket, g.object, err)
	}
	return nil
}

// isPreconditionFailure reports a GCS 412 (generation precondition not met).
//
// Matched structurally via the transport error's status code where available,
// with a string fallback for wrapped errors that lose the type.
func isPreconditionFailure(err error) bool {
	if err == nil {
		return false
	}
	type coder interface{ HTTPStatusCode() int }
	var c coder
	if errors.As(err, &c) {
		return c.HTTPStatusCode() == 412
	}
	msg := err.Error()
	return strings.Contains(msg, "conditionNotMet") ||
		strings.Contains(msg, "Precondition Failed") ||
		strings.Contains(msg, "412")
}

// defaultConfigBucket/Object locate the shared coordinator config. Overridable
// so a staging bucket can be targeted without a code change.
func configLocation() (bucket, object string) {
	bucket = os.Getenv("AILANG_CONFIG_BUCKET")
	if bucket == "" {
		project := os.Getenv("AILANG_CLOUD_PROJECT")
		if project == "" {
			project = "ailang-multivac"
		}
		bucket = project + "-ailang-config"
	}
	object = os.Getenv("AILANG_CONFIG_OBJECT")
	if object == "" {
		object = "config.yaml"
	}
	return bucket, object
}

func newGCSConfigStore(ctx context.Context) (*gcsConfigStore, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	bucket, object := configLocation()
	return &gcsConfigStore{ctx: ctx, client: client, bucket: bucket, object: object}, nil
}

// coordinatorConfig implements `ailang coordinator config <get|set|diff>`.
func coordinatorConfig(args []string) error {
	if len(args) == 0 {
		printCoordinatorConfigHelp()
		return nil
	}

	ctx := context.Background()
	sub, rest := args[0], args[1:]

	switch sub {
	case "get":
		return coordinatorConfigGet(ctx, rest)
	case "set":
		return coordinatorConfigSet(ctx, rest)
	case "diff":
		return coordinatorConfigDiff(ctx, rest)
	case "help", "-h", "--help":
		printCoordinatorConfigHelp()
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (want: get, set, diff)", sub)
	}
}

// coordinatorConfigGet writes the live config to stdout or a file, and reports
// the generation on stderr so it can be piped without corrupting the payload.
func coordinatorConfigGet(ctx context.Context, args []string) error {
	store, err := newGCSConfigStore(ctx)
	if err != nil {
		return err
	}
	data, gen, err := store.Read()
	if err != nil {
		return err
	}

	out := ""
	if len(args) > 0 {
		out = args[0]
	}
	if out == "" || out == "-" {
		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
	} else if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}

	fmt.Fprintf(os.Stderr, "gs://%s/%s generation %d\n", store.bucket, store.object, gen)
	fmt.Fprintf(os.Stderr, "To write it back safely:\n  ailang coordinator config set <file> --if-generation %d\n", gen)
	return nil
}

// coordinatorConfigSet validates and writes, refusing a stale generation.
func coordinatorConfigSet(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ailang coordinator config set <file> --if-generation N [--force]")
	}
	path := args[0]

	var ifGen int64 = -1
	force := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--if-generation":
			if i+1 >= len(args) {
				return errors.New("--if-generation needs a value")
			}
			i++
			if _, err := fmt.Sscanf(args[i], "%d", &ifGen); err != nil {
				return fmt.Errorf("--if-generation %q is not a number", args[i])
			}
		case "--force":
			force = true
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	store, err := newGCSConfigStore(ctx)
	if err != nil {
		return err
	}

	if force {
		// --force still NAMES what it overwrites. An override that hides which
		// version it discarded is the same silent clobber with extra steps.
		_, live, rerr := store.Read()
		if rerr != nil {
			return rerr
		}
		fmt.Fprintf(os.Stderr, "--force: overwriting generation %d (its contents will be discarded)\n", live)
		ifGen = live
	}
	if ifGen < 0 {
		return errors.New("--if-generation is required (get it from `ailang coordinator config get`); pass --force only to deliberately discard another machine's write")
	}

	if err := writeConfigCAS(store, data, ifGen); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote gs://%s/%s\n", store.bucket, store.object)
	return nil
}

// coordinatorConfigDiff reports whether a local copy differs from live, without
// writing anything.
func coordinatorConfigDiff(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ailang coordinator config diff <file>")
	}
	local, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read %s: %w", args[0], err)
	}
	store, err := newGCSConfigStore(ctx)
	if err != nil {
		return err
	}
	live, gen, err := store.Read()
	if err != nil {
		return err
	}
	if string(local) == string(live) {
		fmt.Printf("identical to gs://%s/%s generation %d\n", store.bucket, store.object, gen)
		return nil
	}
	fmt.Printf("DIFFERS from gs://%s/%s generation %d (local %d bytes, live %d bytes)\n",
		store.bucket, store.object, gen, len(local), len(live))
	if verr := validateCoordinatorConfigBytes(local); verr != nil {
		fmt.Printf("and your local copy would be REFUSED: %v\n", verr)
	}
	return nil
}

func printCoordinatorConfigHelp() {
	fmt.Fprintln(os.Stdout, `ailang coordinator config — read/write the shared coordinator config safely

  get [file]                              Fetch config; reports its generation
  set <file> --if-generation N [--force]  Validate and write iff unchanged
  diff <file>                             Compare a local copy against live

Writes use a generation precondition. A write built on a stale read is REFUSED
rather than applied, because overwriting silently discards the other machine's
edit (measured 2026-08-26: a correct change was clobbered 13 minutes later, with
no error on either side).

Location: $AILANG_CONFIG_BUCKET / $AILANG_CONFIG_OBJECT, defaulting to
<AILANG_CLOUD_PROJECT>-ailang-config/config.yaml.`)
}

// coordinatorRouting implements `ailang coordinator routing [role]`.
//
// RETIRED by M-MODEL-REGISTRY-SINGLE-SOURCE M7. It read the `model_routing`
// table in the coordinator config (M-PIPELINE-RECONCILIATION M5, D3); that table
// is deleted, because the registry answers the same question in one place that
// does not need its own deploy.
//
// It forwards rather than 404s: the mission driver and any operator muscle
// memory should land on the replacement with the answer in hand, not an error.
// The output shape differs — `models role` prints friendly name, wire string and
// harness — so anything parsing this must move to field 2.
func coordinatorRouting(args []string) error {
	fmt.Fprintln(os.Stderr,
		"note: `coordinator routing` is retired — the model_routing table it read no longer exists.\n"+
			"      The registry answers this now; forwarding to `ailang models role`.")
	return modelsRole(args)
}
