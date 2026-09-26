package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/sunholo-data/ailang/internal/fsyncdir"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
	"gopkg.in/yaml.v3"
)

func missionRetryReviewCommand(args []string) error {
	return runMissionRetryReview(context.Background(), args, os.Stdout, missionIterationDeps{})
}
func runMissionRetryReview(ctx context.Context, args []string, out io.Writer, deps missionIterationDeps) error {
	if len(args) == 0 || args[0] == "" || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: mission retry-review NAME --work-item ID --new-id ID --output FILE --max-tokens N --timeout-seconds N --max-cost-usd N [--authority-file JSON --base-revision COMMIT] [--evaluator MODEL]")
	}
	name := args[0]
	fs := flag.NewFlagSet("mission retry-review", flag.ContinueOnError)
	fs.SetOutput(out)
	activationID := fs.String("activation", "", "read retained binding from an owned activation record")
	work := fs.String("work-item", "", "terminal source work item")
	newid := fs.String("new-id", "", "distinct successor ID")
	output := fs.String("output", "", "new JSON file; also creates .manifest.json and .models.yml")
	auth := fs.String("authority-file", "", "JSON authority-reference array; omit for non-executable draft")
	base := fs.String("base-revision", "", "full candidate-descendant commit containing approved references")
	model := fs.String("evaluator", "", "explicit new evaluator model; otherwise frozen original routes")
	tokens := fs.Int("max-tokens", 0, "cumulative fresh input+output tokens")
	timeout := fs.Int("timeout-seconds", 0, "stage/iteration deadline (1..1800 seconds)")
	cost := fs.Float64("max-cost-usd", 0, "positive metered cost guard")
	seen := map[string]bool{}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			key := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)[0]
			if seen[key] {
				return iterationExit(2, fmt.Errorf("duplicate flag %q", key))
			}
			seen[key] = true
		}
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return iterationExit(2, err)
	}
	if fs.NArg() != 0 || *work == "" || *output == "" {
		return fmt.Errorf("source work item and output are required")
	}
	binding, err := resolveMissionReadBinding(deps.BindingPath, *activationID, name, *work)
	if err != nil {
		return err
	}
	store, err := coordinator.OpenMissionReadOnlyStore(binding.StateDB)
	if err != nil {
		return err
	}
	defer store.Close()
	opts := iteration.ReviewOptions{NewID: *newid, Limits: iteration.Limits{TimeoutSeconds: *timeout, MaxTokens: *tokens, MaxCostUSD: *cost}, BaseRevision: *base, Evaluator: *model}
	if *auth != "" {
		f, e := os.Open(*auth)
		if e != nil {
			return e
		}
		opts.AuthorityRefs, e = iteration.DecodeReviewAuthorities(f)
		closeErr := f.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
	}
	draft, err := iteration.PrepareReview(ctx, store, coordinator.MissionWorkItemKey{MissionID: name, WorkItemID: *work}, opts)
	if err != nil {
		return err
	}
	spec, err := json.MarshalIndent(draft.Spec, "", "  ")
	if err != nil {
		return err
	}
	manifest, err := json.MarshalIndent(struct {
		Ready    bool                     `json:"ready"`
		Missing  []iteration.ArtifactRef  `json:"missing_approval"`
		Evidence iteration.ReviewManifest `json:"evidence"`
	}{draft.Ready, draft.MissingApproval, draft.Manifest}, "", "  ")
	if err != nil {
		return err
	}
	models, err := yaml.Marshal(draft.Models)
	if err != nil {
		return err
	}
	if err = writeReviewFiles(*output, [][]byte{append(spec, '\n'), append(manifest, '\n'), models}); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(map[string]any{"ready": draft.Ready, "work_item": *output, "manifest": *output + ".manifest.json", "models": *output + ".models.yml", "missing_approval": draft.MissingApproval, "next_action": map[bool]string{true: "review generated input; set AILANG_MODELS_PATH to the absolute .models.yml sidecar for dry-run and execution", false: "supply committed authority binding missing artifact digests; regenerate into a new output path"}[draft.Ready]})
}

// Publish the runnable input last, after both durable evidence sidecars. Exclusive
// hard links expose only complete files and refuse existing destinations/symlinks.
func writeReviewFiles(path string, bodies [][]byte) error {
	if len(bodies) != 3 {
		return fmt.Errorf("three review bundle files required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return err
	}
	if parent != filepath.Dir(abs) {
		return fmt.Errorf("output directory must not redirect through symlinks")
	}
	paths := []string{abs, abs + ".manifest.json", abs + ".models.yml"}
	for _, p := range paths {
		if _, err = os.Lstat(p); !os.IsNotExist(err) {
			return fmt.Errorf("output exists or cannot be checked: %s", p)
		}
	}
	for _, i := range []int{2, 1, 0} {
		if err = publishReviewFile(parent, paths[i], bodies[i]); err != nil {
			return fmt.Errorf("incomplete bundle retained; inspect before retry: %w", err)
		}
	}
	return nil
}
func publishReviewFile(parent, path string, body []byte) error {
	f, err := os.CreateTemp(parent, ".review-bundle-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(body)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = os.Link(f.Name(), path); err != nil {
		return err
	}
	// Directory fsync is unix-only; see internal/fsyncdir. This was the fourth hand-rolled
	// copy and the one the first sweep missed, which is the argument for the helper.
	return fsyncdir.Sync(parent)
}
