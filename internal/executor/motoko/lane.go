package motoko

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The ailang_only lane on motoko (design: m-motoko-ailang-only-lane, D4).
//
// motoko runs the lane only when the properties that make it safe hold, read
// as motoko itself resolves them — never a Go re-parse of its config file:
//
//   - the task carries a program policy that resolves as restricted;
//   - motoko's own resolver (config.ail print_config_json) shows
//     extensions.order[0] == "ailang_policy", every other entry on the lane
//     allowlist, extensions.strict == true and tools.hybrid == false;
//   - the motoko build ENFORCES extensions.strict (upstream
//     arniwesth/motoko_agent#205): a probe profile naming an uninstalled
//     extension must exit 2. A strict flag on a build that ignores it means
//     nothing.
//
// After the run, the session is checked: the first loaded extension must be
// ailang_policy and no tool outside the lane may have executed. Either breach
// fails the run as a policy violation — never a pass.

const laneExtension = "ailang_policy"

// laneCompanions may load after ailang_policy: they make model calls or shape
// prompts, and run nothing outside the policy.
var laneCompanions = []string{"empty_stop_guard", "compaction_ai", "repetition_guard"}

// laneToolNames are the tool names the lane exposes to the model (pi's names).
var laneToolNames = []string{"ailang_read", "ailang_write", "ailang_edit", "ailang_check", "ailang_run", "ailang_cli", "builtins_search", "examples_search"}

// isLaneTask reports whether the task asks for exactly the ailang_only tools.
func isLaneTask(task *executor.Task) bool {
	if task == nil || task.AllowedTools == nil {
		return false
	}
	lane, err := executor.ProfileTools(executor.ToolProfileAILANGOnly)
	if err != nil {
		return false
	}
	return sameSet(task.AllowedTools, lane)
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := slices.Clone(a)
	y := slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

// resolvedProfile is the slice of motoko's resolved config the lane checks.
type resolvedProfile struct {
	Extensions struct {
		Order  []string `json:"order"`
		Strict bool     `json:"strict"`
	} `json:"extensions"`
	Tools struct {
		Hybrid bool `json:"hybrid"`
	} `json:"tools"`
}

// laneRunner runs a command in the motoko repo; swapped in tests.
type laneRunner func(ctx context.Context, dir string, env []string, name string, args ...string) (stdout string, exitCode int, err error)

func defaultLaneRunner(ctx context.Context, dir string, env []string, name string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// A non-zero exit is an answer (the strict probe expects 2), not a failure to run.
			code = ee.ExitCode()
		} else {
			return "", -1, err
		}
	}
	return string(out), code, nil
}

// checkLaneProfile reads the profile through motoko's own resolver and checks
// the lane properties.
func checkLaneProfile(ctx context.Context, run laneRunner, repo, workdir, profile string) error {
	// MOTOKO_REPO: motoko looks for the profile in the workdir first, then in
	// the repo (config.ail profile_dir_for) — the same env the real run gets.
	out, code, err := run(ctx, repo, []string{"AILANG_RELAX_MODULES=1", "MOTOKO_REPO=" + repo},
		"ailang", "run", "--quiet", "--caps", "IO,FS,Env", "--entry", "print_config_json", "src/core/config.ail",
		"--", "--workdir", workdir, "--profile", profile)
	if err != nil || code != 0 {
		return fmt.Errorf("could not resolve motoko profile %q through motoko's own resolver (exit %d): %v", profile, code, err)
	}
	var cfg resolvedProfile
	if jerr := json.Unmarshal([]byte(lastJSONObject(out)), &cfg); jerr != nil {
		return fmt.Errorf("motoko's resolver printed no config for profile %q: %v", profile, jerr)
	}
	order := cfg.Extensions.Order
	switch {
	case len(order) == 0:
		// print_config_json silently returns defaults when it finds no profile.
		return fmt.Errorf("profile %q resolves to an empty extensions.order (profile not found?)", profile)
	case order[0] != laneExtension:
		return fmt.Errorf("profile %q: extensions.order[0] is %q, the lane needs %q first", profile, order[0], laneExtension)
	case !cfg.Extensions.Strict:
		return fmt.Errorf("profile %q: extensions.strict is false — a lane session could start without %s", profile, laneExtension)
	case cfg.Tools.Hybrid:
		return fmt.Errorf("profile %q: tools.hybrid is on — delegated tools run in the env-server outside the policy", profile)
	}
	for _, ext := range order[1:] {
		if !slices.Contains(laneCompanions, ext) {
			return fmt.Errorf("profile %q loads %q, which is not on the lane allowlist %v", profile, ext, laneCompanions)
		}
	}
	return nil
}

// lastJSONObject returns the last line that looks like a JSON object (the
// resolver may print warnings before it).
func lastJSONObject(out string) string {
	out = strings.TrimSpace(out)
	if i := strings.Index(out, "{"); i >= 0 {
		return out[i:]
	}
	return out
}

var (
	strictProbeMu    sync.Mutex
	strictProbeCache = map[string]error{}
)

// checkStrictEnforced runs motoko's own probe once per repo: a profile naming
// an uninstalled extension with strict = true must exit 2.
func checkStrictEnforced(ctx context.Context, run laneRunner, repo string) error {
	strictProbeMu.Lock()
	defer strictProbeMu.Unlock()
	if err, ok := strictProbeCache[repo]; ok {
		return err
	}
	err := func() error {
		if _, serr := os.Stat(filepath.Join(repo, "scripts", "verify_strict_extensions.ail")); serr != nil {
			return fmt.Errorf("motoko build does not enforce extensions.strict: scripts/verify_strict_extensions.ail is absent (needs arniwesth/motoko_agent#205)")
		}
		dir, derr := os.MkdirTemp("", "motoko-strict-probe-")
		if derr != nil {
			return derr
		}
		defer os.RemoveAll(dir)
		pdir := filepath.Join(dir, ".motoko", "config", "probe")
		if merr := os.MkdirAll(pdir, 0o755); merr != nil {
			return merr
		}
		if werr := os.WriteFile(filepath.Join(pdir, "config.json"), []byte(`{"agent":{"model":"stub"},"extensions":{"order":["no_such_ext"],"strict":true}}`), 0o644); werr != nil {
			return werr
		}
		pctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		_, code, rerr := run(pctx, repo, []string{"AILANG_RELAX_MODULES=1"},
			"ailang", "run", "--quiet", "--caps", "Net,AI,SharedMem,IO,Env,Clock,FS,Process,Stream", "--ai-stub", "--entry", "main",
			"scripts/verify_strict_extensions.ail", "--", dir)
		if rerr != nil {
			return fmt.Errorf("strict-enforcement probe could not run: %v", rerr)
		}
		if code != 2 {
			return fmt.Errorf("motoko build does not enforce extensions.strict: the probe profile started (exit %d, want 2)", code)
		}
		return nil
	}()
	strictProbeCache[repo] = err
	return err
}

// checkLaneTask is the whole pre-spawn gate for a lane task.
func (e *MotokoExecutor) checkLaneTask(ctx context.Context, run laneRunner, task *executor.Task, profile string) error {
	if task.PolicyPath == "" {
		return fmt.Errorf("ailang_only lane on motoko needs a program policy (task.PolicyPath)")
	}
	raw, err := os.ReadFile(task.PolicyPath)
	if err != nil {
		return fmt.Errorf("ailang_only lane: reading the policy: %w", err)
	}
	if err := executor.CheckLanePolicy(executor.ToolProfileAILANGOnly, raw); err != nil {
		return err
	}
	if e.motokoRepo == "" {
		return fmt.Errorf("ailang_only lane: motoko repo unknown (HealthCheck has not located it)")
	}
	if err := checkLaneProfile(ctx, run, e.motokoRepo, task.Workspace, profile); err != nil {
		return fmt.Errorf("ailang_only lane: %w", err)
	}
	return checkStrictEnforced(ctx, run, e.motokoRepo)
}

// laneSessionViolation scans the session JSONL after a lane run: the first
// loaded extension must be ailang_policy, and every tool that executed must be
// a lane tool (a denied call is not an execution).
func laneSessionViolation(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("cannot read the session to verify the lane: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	sawStart := false
	for sc.Scan() {
		var ev struct {
			Type             string    `json:"type"`
			LoadedExtensions *[]string `json:"loaded_extensions"`
			Results          []struct {
				Tool    string         `json:"tool"`
				Payload map[string]any `json:"payload"`
			} `json:"results"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "session_start":
			// motoko writes two session_start events: the host's, which lists
			// loaded_extensions, and the run's, which does not. Judge the one
			// that carries the list; require that one exists.
			if ev.LoadedExtensions == nil {
				continue
			}
			sawStart = true
			if exts := *ev.LoadedExtensions; len(exts) == 0 || exts[0] != laneExtension {
				return fmt.Sprintf("session loaded %v; the lane needs %s first", exts, laneExtension)
			}
		case "native_tool_results":
			for _, r := range ev.Results {
				if r.Tool == "" || slices.Contains(laneToolNames, r.Tool) {
					continue
				}
				if denied, _ := r.Payload["denied_by_policy"].(bool); denied {
					continue
				}
				return fmt.Sprintf("tool %q executed outside the lane", r.Tool)
			}
		}
	}
	if !sawStart {
		return "session has no session_start event listing loaded_extensions; cannot verify the lane"
	}
	return ""
}

// laneProfile is motoko's profile for the ailang_only lane (motoko_ext_ailang_policy
// first, strict on). It is the only kind of profile checkLaneProfile accepts.
const laneProfile = "ailang_only"

// laneProfileFor is the profile a lane task runs on: the task's explicit
// motoko_profile (the eval harness sets one from models.yml), else laneProfile.
// NOT the executor's default: the cloud job pins MOTOKO_CONFIG=dogfood, which
// fails the lane gate, so a coordinator agent on the lane (it sets no
// motoko_profile) would have refused every task.
func laneProfileFor(task *executor.Task) string {
	if p := task.Metadata["motoko_profile"]; p != "" {
		return p
	}
	return laneProfile
}

// executeLane runs an ailang_only task: the pre-spawn gate, the run (the policy
// reaches motoko as AILANG_AGENT_POLICY), then the post-run session check.
func (e *MotokoExecutor) executeLane(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	profile := laneProfileFor(task)
	// executeStreaming re-derives the profile from the same metadata, so the
	// gate and the run cannot disagree about which profile they judged.
	if task.Metadata == nil {
		task.Metadata = map[string]string{}
	}
	task.Metadata["motoko_profile"] = profile
	if err := e.checkLaneTask(ctx, defaultLaneRunner, task, profile); err != nil {
		return nil, err
	}
	// AILANG_AGENT_POLICY reaches motoko through executor.BuildEnvironment,
	// which derives it from task.PolicyPath (and refuses it as task ExtraEnv).
	res, err := e.executeStreaming(ctx, task, handler)
	if res == nil {
		return res, err
	}
	res.ToolPolicy = append([]string(nil), task.AllowedTools...)
	res.PolicyDigest = executor.PolicyDigest(task.PolicyPath)
	path, _ := res.ProviderData["motoko_session_jsonl"].(string)
	if violation := laneSessionViolation(path); violation != "" {
		res.Success = false
		res.Error = "policy_violation: " + violation
	}
	return res, err
}
