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
	return config.AgyHomeRootFor(h), nil
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

// Permission posture (D3, option B — Mark, attended 2026-11-23: "yep go ahead").
//
// agy runs with --sandbox (terminal sandbox) in --mode accept-edits, and the rules
// allow EVERY command — the sandbox, not a command allowlist, is the boundary. The
// narrow allowlist this replaced voided 2 of 23 smoke runs on the model's own
// self-checks (an inline python3 assertion; `diff <(...) && rm -f`).
//
// Measured on agy 1.5.15 (2026-11-23) with this posture:
//   - writes outside the workspace: "operation not permitted"; ~/.config reads: blocked
//   - the terminal starts in the task workspace; Go builds and `ailang run` work
//   - writable outside the workspace ONLY where an absolute write_file() rule says:
//     the Go build and module caches and a fixed per-route TMPDIR
//
// Two traps the rules must avoid:
//   - A RELATIVE path rule (write_file(.git/)) makes the sandbox config invalid, so
//     EVERY command fails to start while agy still reports status SUCCESS with no
//     denials. Only absolute paths go in; TestPermissionRules_AllPathsAbsolute pins it,
//     and stream.go fails a run whose tool steps report a sandbox configuration error.
//   - --add-dir moves the shell's starting directory to the first added directory (the
//     Go cache), not the workspace, so it is NOT used to grant the caches.
//
// Deny beats allow, so the git-write and network commands below stay refused. In -p mode
// a refused action is SOFT-DENIED (exit 0, status SUCCESS, denied_actions), which this
// executor banks as a failure. --dangerously-skip-permissions is never used.
var defaultDenyCommands = []string{
	"git push", "git reset", "git clean", "git checkout", "git stash", "git rebase",
	"rm -rf", "sudo", "curl", "wget",
}

// permissionRules builds the allow/deny lists. A nil AllowedTools allows all commands
// (inside the sandbox); an explicit list grants commands only when it names Bash. The
// writable paths are granted either way (file tools may need them).
func permissionRules(task *executor.Task, writable []string) (allow, deny []string) {
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
		allow = append(allow, "command(*)")
	}
	for _, w := range writable {
		if filepath.IsAbs(w) {
			allow = append(allow, "write_file("+w+")")
		}
	}
	for _, c := range defaultDenyCommands {
		deny = append(deny, "command("+c+")")
	}
	return allow, deny
}

// effectiveToolPolicy is what Result.ToolPolicy banks: the rules agy ran under.
func effectiveToolPolicy(task *executor.Task, writable []string) []string {
	allow, deny := permissionRules(task, writable)
	out := make([]string, 0, len(allow)+len(deny)+1)
	out = append(out, "mode:accept-edits+sandbox")
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

// toolchain is where the child's Go toolchain and temp files live. HOME moves to the
// route HOME, so the operator's locations are passed explicitly (otherwise every run
// re-downloads modules into the agy HOME and git loses the operator's global config),
// and the sandbox must be told they are writable.
type toolchain struct {
	gopath, gomodcache, gocache, tmp, gitconfig string
}

// writable is what the sandbox must let the child write outside the workspace.
func (t toolchain) writable() []string {
	var out []string
	for _, p := range []string{t.gocache, t.gomodcache, t.tmp} {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toolchainFor(home string) (toolchain, error) {
	realHome, err := os.UserHomeDir()
	if err != nil {
		return toolchain{}, fmt.Errorf("agy: cannot resolve the operator HOME: %w", err)
	}
	t := toolchain{gopath: config.Raw("GOPATH"), gomodcache: config.Raw("GOMODCACHE"), gocache: config.Raw("GOCACHE")}
	if t.gopath == "" {
		t.gopath = filepath.Join(realHome, "go")
	}
	if t.gomodcache == "" {
		t.gomodcache = filepath.Join(strings.Split(t.gopath, string(os.PathListSeparator))[0], "pkg", "mod")
	}
	if t.gocache == "" {
		if cd, err := os.UserCacheDir(); err == nil {
			t.gocache = filepath.Join(cd, "go-build")
		}
	}
	// A FIXED per-route TMPDIR, so the settings file (shared by concurrent runs) does
	// not change per run.
	t.tmp = filepath.Join(home, "tmp")
	if err := os.MkdirAll(t.tmp, 0o700); err != nil {
		return toolchain{}, fmt.Errorf("agy: create TMPDIR: %w", err)
	}
	if gc := filepath.Join(realHome, ".gitconfig"); fileExists(gc) {
		t.gitconfig = gc
	}
	// The sandbox compares resolved paths (/var -> /private/var on macOS).
	for _, p := range []*string{&t.gomodcache, &t.gocache, &t.tmp} {
		if *p == "" {
			continue
		}
		if r, err := filepath.EvalSymlinks(*p); err == nil {
			*p = r
		}
	}
	return t, nil
}

// routeEnv is the child's executor-owned environment for a route.
func routeEnv(route, home string, t toolchain) ([]string, error) {
	env := []string{
		"HOME=" + home,
		// Any agy invocation may start the background self-updater, and an
		// unpinned harness under a longitudinal series is the pi-pin lesson
		// (D4). Measured 2026-11-23: plain runs moved 1.3.5 -> 1.5.15.
		"AGY_CLI_DISABLE_AUTO_UPDATE=true",
		"GOPATH=" + t.gopath,
		"GOMODCACHE=" + t.gomodcache,
		"TMPDIR=" + t.tmp + "/",
	}
	if t.gocache != "" {
		env = append(env, "GOCACHE="+t.gocache)
	}
	if t.gitconfig != "" {
		env = append(env, "GIT_CONFIG_GLOBAL="+t.gitconfig)
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
