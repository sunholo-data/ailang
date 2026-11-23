package agy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// Auth routes. agy authenticates one of two ways, and they need different
// HOMEs (design_docs/planned/m-agy-executor.md, Architecture §3):
//
//   - subscription: a Google sign-in (Google AI Pro/Ultra on a personal
//     account) cached by agy itself. Flat rate; cost is list-price-equivalent.
//   - apikey: settings `{"modelProvider":"gemini"}` plus GEMINI_API_KEY in the
//     child only, taken from GEMINI_AGENTS_API_KEY. Metered.
//
// The route is chosen by the model string: "api/<agy-model>" is the API-key
// route; a bare "<agy-model>" is the subscription route. The registry rows
// spell it out in agent_model_name, so a row cannot change route silently.
const (
	routeSubscription = "subscription"
	routeAPIKey       = "apikey"
	apiKeyPrefix      = "api/"
)

// splitRoute returns the auth route and the model id agy itself accepts.
func splitRoute(model string) (route, cliModel string) {
	if rest, ok := strings.CutPrefix(model, apiKeyPrefix); ok {
		return routeAPIKey, rest
	}
	return routeSubscription, model
}

// homeRoot is AILANG_AGY_HOME_ROOT, else ~/.ailang/state/agy.
func homeRoot() (string, error) {
	if r := config.AgyHomeRoot(); r != "" {
		return r, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("agy: cannot resolve the operator HOME for the agy HOME root: %w", err)
	}
	return filepath.Join(h, ".ailang", "state", "agy"), nil
}

// routeHome is the isolated HOME agy runs under for a route. Never the
// operator's real HOME: that one's ~/.gemini settings can point agy at a GCP
// project, which silently turns a "subscription" run into metered Vertex
// (measured on this machine 2026-11-23: gcp.project sunholo-gemini-enterprise).
func routeHome(route string) (string, error) {
	root, err := homeRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, route), nil
}

// settingsPath is where agy reads its user-level settings under a HOME.
func settingsPath(home string) string {
	return filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
}

// Permission posture (D3, PROPOSED — awaiting Mark's ratification).
//
// agy runs in --mode accept-edits: workspace file edits proceed, everything else
// needs a rule. In -p mode an action without a rule is SOFT-DENIED: the run
// exits 0 with status SUCCESS and lists it under denied_actions, which this
// executor banks as a failure. Rules are word-by-word command prefixes
// (antigravity.google/docs/permissions?tab=cli); deny beats allow.
//
// --dangerously-skip-permissions is deliberately NOT used.
var (
	defaultAllowCommands = []string{
		"ailang", "go", "git", "make", "gofmt",
		"ls", "cat", "head", "tail", "grep", "rg", "find", "wc", "diff", "sort", "uniq",
		"sed", "awk", "jq", "echo", "printf", "pwd", "mkdir", "touch", "cp", "mv", "test",
	}
	defaultDeny = []string{
		"command(git push)", "command(git reset)", "command(git clean)", "command(git checkout)",
		"command(git stash)", "command(git rebase)", "command(rm -rf)", "command(sudo)",
		"command(curl)", "command(wget)", "write_file(.git/)",
	}
)

// permissionRules builds the allow/deny lists for a task. A nil AllowedTools
// gets the default command allowlist; an explicit list grants shell commands
// only when it names Bash (the canonical shell tool name).
func permissionRules(task *executor.Task) (allow, deny []string) {
	shell := task == nil || task.AllowedTools == nil
	if task != nil {
		for _, t := range task.AllowedTools {
			if strings.EqualFold(t, "Bash") {
				shell = true
			}
		}
	}
	allow = []string{}
	if shell {
		for _, c := range defaultAllowCommands {
			allow = append(allow, "command("+c+")")
		}
	}
	return allow, append([]string(nil), defaultDeny...)
}

// effectiveToolPolicy is what Result.ToolPolicy banks: the rules agy ran under.
func effectiveToolPolicy(task *executor.Task) []string {
	allow, deny := permissionRules(task)
	out := make([]string, 0, len(allow)+len(deny)+1)
	out = append(out, "mode:accept-edits")
	out = append(out, allow...)
	for _, d := range deny {
		out = append(out, "deny:"+d)
	}
	return out
}

// ensureSettings merges this executor's keys into the route HOME's
// settings.json, keeping every key agy wrote itself (agy rewrites the file into
// its own format at startup and keeps unknown keys). The file is only rewritten
// when a key we own differs, by atomic rename, so concurrent runs with the same
// policy never race on its content.
func ensureSettings(home, route string, allow, deny []string) error {
	path := settingsPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("agy: create settings dir: %w", err)
	}
	cur := map[string]any{}
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &cur); err != nil {
			return fmt.Errorf("agy: %s is not valid JSON (agy would reset it, rules included): %w", path, err)
		}
	}
	want := map[string]any{}
	for k, v := range cur {
		want[k] = v
	}
	if route == routeAPIKey {
		want["modelProvider"] = "gemini"
	} else {
		// The subscription route must not inherit a provider or a GCP project:
		// either one moves billing off the plan.
		delete(want, "modelProvider")
		delete(want, "gcp")
	}
	want["permissions"] = map[string]any{"allow": toAny(allow), "deny": toAny(deny)}

	if reflect.DeepEqual(normalize(cur), normalize(want)) {
		return nil
	}
	b, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		return err
	}
	// A unique temp per write: parallel runs in ONE process (eval-suite --parallel)
	// share a PID, so a PID-named temp let one run rename away the other's file
	// (measured 2026-11-23: "rename settings.json.ailang.94067: no such file").
	f, err := os.CreateTemp(filepath.Dir(path), "settings.json.ailang.*")
	if err != nil {
		return fmt.Errorf("agy: write settings: %w", err)
	}
	tmp := f.Name()
	_, werr := f.Write(append(b, '\n'))
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("agy: write settings: %v %v", werr, cerr)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("agy: install settings: %w", err)
	}
	return nil
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// normalize round-trips through JSON so []string and []any compare equal.
func normalize(m map[string]any) any {
	b, _ := json.Marshal(m)
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

// routeEnv is the child's executor-owned environment for a route.
//
// HOME moves to the route HOME, so the operator's toolchain locations are
// passed explicitly: without them every run would re-download Go modules into
// the agy HOME, and git would lose the operator's global config.
func routeEnv(route, home string) ([]string, error) {
	env := []string{
		"HOME=" + home,
		// Any agy invocation may start the background self-updater, and an
		// unpinned harness under a longitudinal series is the pi-pin lesson
		// (D4). Measured 2026-11-23: plain runs moved 1.3.5 -> 1.5.15.
		"AGY_CLI_DISABLE_AUTO_UPDATE=true",
	}
	realHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("agy: cannot resolve the operator HOME: %w", err)
	}
	gopath := config.Raw("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(realHome, "go")
	}
	env = append(env, "GOPATH="+gopath)
	if v := config.Raw("GOMODCACHE"); v != "" {
		env = append(env, "GOMODCACHE="+v)
	}
	gocache := config.Raw("GOCACHE")
	if gocache == "" {
		if cd, err := os.UserCacheDir(); err == nil {
			gocache = filepath.Join(cd, "go-build")
		}
	}
	if gocache != "" {
		env = append(env, "GOCACHE="+gocache)
	}
	if gc := filepath.Join(realHome, ".gitconfig"); fileExists(gc) {
		env = append(env, "GIT_CONFIG_GLOBAL="+gc)
	}

	if route == routeAPIKey {
		key := config.GeminiAgentsAPIKey()
		if key == "" {
			return nil, fmt.Errorf("agy: the API-key route needs %s (the locked, SA-bound key; ~/.config/ailang/secrets.env)", config.EnvGeminiAgentsAPIKey)
		}
		// Child only. The parent never gains GEMINI_API_KEY.
		env = append(env, config.EnvGeminiAPIKey+"="+key)
	}
	return env, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
