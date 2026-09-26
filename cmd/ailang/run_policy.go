package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/policy"
)

// M-AGENT-AILANG-ONLY-EXECUTION M2 / M-EXECUTOR-POLICY-HARDENING M3 —
// `ailang run --policy <agent-policy.toml>`.
//
// The caller is an AGENT, and an agent that can pass --caps has no policy.
// So with --policy the authority comes from the file and nowhere else: caps,
// the Net allowlist, the FS sandbox, the entrypoint and the limits are
// DERIVED from it, and every flag that could widen them is refused outright
// rather than merged. Admission itself is admitProgram — the same path
// `policy-check` runs, so the two can never disagree.
//
// Process shape (M3): the invocation is SUPERVISED. The parent resolves the
// policy, starts a worker (this same binary, `--policy-worker <fd>`) with the
// wall-clock deadline armed before the worker loads a byte of source, the
// environment reduced to an allowlist in restricted mode, the output capped,
// and a dedicated control pipe for the admission decision — program stdout
// cannot spoof it. See run_policy_supervise.go.
//
// Output contract (unchanged for the pi tool): on DENIAL the decision JSON
// goes to stdout and the exit code is 2 — nothing executes. On ADMISSION one
// JSON line goes to STDERR (prefixed "policy: ") so stdout stays the
// program's. A limit (timeout, output cap) is exit 3 with a
// "policy-result: {...}" envelope on stderr.

// runPolicyWidening names the flags a caller must not combine with --policy.
// Each is refused BY NAME so the agent's transcript says which one.
type runPolicyWidening struct {
	set map[string]bool
}

// refusedWithPolicy is every `run` flag that widens or redirects authority:
// capabilities, environment, model choice, destinations, process access,
// module resolution roots, the entrypoint, and the FS cap.
var refusedWithPolicy = []struct{ name, why string }{
	{"caps", "capabilities come from the policy's allowed_caps"},
	{"no-budgets", "budgets come from the policy"},
	{"allow-env", "environment access is not a policy field"},
	{"allow-env-file", "environment access is not a policy field"},
	{"env", "the environment is not the caller's to set"},
	{"env-snapshot", "the environment is not the caller's to set"},
	{"write-env-snapshot", "the environment is not the caller's to read"},
	{"ai", "the model comes from the policy's ai_provider"},
	{"ai-stub", "set ai_provider = \"stub\" in the policy instead"},
	{"allow-routing", "dynamic provider selection is not a policy field"},
	{"routing-fallback", "dynamic provider selection is not a policy field"},
	{"routing-require", "dynamic provider selection is not a policy field"},
	{"routing-prefer", "dynamic provider selection is not a policy field"},
	{"routing-max-price", "dynamic provider selection is not a policy field"},
	{"entry", "the entrypoint is the policy's entry"},
	{"net-allow-http", "comes from the policy's net_allow_http"},
	{"net-allow-domains", "comes from the policy's net_allow"},
	{"net-allow-localhost", "restricted mode has no localhost grant; trusted_host may list it in net_allow"},
	{"net-allow-metadata", "restricted mode has no metadata grant"},
	{"stream-allow-http", "comes from the policy's net_allow_http"},
	{"stream-allow-domains", "comes from the policy's net_allow"},
	{"stream-allow-localhost", "restricted mode has no localhost grant"},
	{"process-allowlist", "comes from the policy's process_allow"},
	{"stdlib-path", "module roots are not the caller's to choose"},
	{"package-dir", "module roots are not the caller's to choose"},
	{"fs-max-bytes", "comes from the policy's max_fs_transfer_bytes"},
}

// runPolicyResolved is what the policy DECIDES for the run, in the shape the
// existing flag plumbing consumes.
type runPolicyResolved struct {
	policy       *policy.Resolved
	caps         string
	netDomains   string
	netAllowHTTP bool
	// netAllowLocalhost: trusted_host only — an operator who lists a loopback
	// name/literal in net_allow has granted it explicitly. Restricted mode
	// has no such grant (design §3).
	netAllowLocalhost bool
	processAllow      string
	sandbox           string
	entry             string
	digest            string
	// aiModel/aiStub feed setupAIHandler exactly where --ai/--ai-stub did.
	aiModel string
	aiStub  bool
}

// activeRunPolicy is the resolved policy of the current worker, read when
// the runner options are built (nil outside a policy run).
var activeRunPolicy *policy.Resolved

// refusePolicy prints a named refusal and exits 1.
func refusePolicy(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "%s: --policy: %s\n", red("Error"), fmt.Sprintf(format, a...))
	os.Exit(1)
}

// resolveRunPolicyFor loads the policy ONCE, refuses widening flags, applies
// the platform rule and the entry-file rule (M7): with an fs_sandbox, the
// program file — and so the module root the imports resolve from — must lie
// inside it (the lane's ailang_run used to accept any path, which let an
// agent execute, and thereby read, .ail sources anywhere on the host under
// the policy). Shared by the parent (supervisor) and the worker: both see
// the same argv, so both refuse the same things.
func resolveRunPolicyFor(policyPath, filename string, w runPolicyWidening) (*policy.Resolved, []byte) {
	for _, f := range refusedWithPolicy {
		if w.set[f.name] {
			refusePolicy("--%s is not allowed with --policy — %s", f.name, f.why)
		}
	}
	res, raw, err := policy.LoadResolved(policyPath)
	if err != nil {
		refusePolicy("%v", err)
	}
	if res.Restricted() && !restrictedModeSupported() {
		refusePolicy("restricted mode is not supported on %s/%s (confined filesystem roots and descendant termination are unverified here); this policy needs a Linux or macOS worker, or security_mode = %q with its weaker guarantees", runtime.GOOS, runtime.GOARCH, policy.ModeTrustedHost)
	}
	if filename != "" && res.Root != "" {
		if !entryInsideSandbox(res.Root, filename) {
			refusePolicy("program %s is outside fs_sandbox %s — under a policy the entry file and its module root must be inside the sandbox", filename, res.Root)
		}
	}
	return res, raw
}

// entryInsideSandbox reports whether filename resolves (symlinks included)
// to a path under root.
func entryInsideSandbox(root, filename string) bool {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return false
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	if real, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootAbs = real
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// restrictedModeSupported: os.Root confines through directory descriptors
// and proctree kills the process group on Linux and macOS. Windows has
// neither claim (D6) and GOOS=js has a documented TOCTOU.
func restrictedModeSupported() bool {
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return true
	}
	return false
}

// applyRunPolicy is the WORKER side: resolve, admit the program, bind the
// runtime to the policy, and report the decision on the control channel.
// It exits the process on refusal (1) or denial (2).
func applyRunPolicy(policyPath, filename string, w runPolicyWidening, control *os.File) runPolicyResolved {
	res, _ := resolveRunPolicyFor(policyPath, filename, w)

	// Freeze source reads from here on: the bytes admitted are the bytes
	// executed, under the policy's per-file and module-graph caps (AC6).
	loader.EnableSourceSnapshot(res.MaxSourceBytes, res.MaxModuleGraphBytes)

	// In-process backstop for the wall clock: the supervisor kills the group
	// at res.Timeout; if this worker was started without one (a direct
	// --policy-worker invocation), it still cannot outlive the policy.
	time.AfterFunc(res.Timeout+time.Second, func() {
		fmt.Fprintf(os.Stderr, "policy-result: %s\n", limitEnvelope(res, "execute", "timeout", fmt.Sprintf("exceeded timeout_ms (%s) — worker self-terminated", res.Timeout)))
		os.Exit(3)
	})

	out, code := admitProgram(res, policyPath, filename)
	if code != 0 {
		reportDecision(control, controlMessage{Kind: "denied", Code: code, Decision: out})
		if control == nil {
			emitJSON(out)
		}
		os.Exit(code)
	}

	resolved := runPolicyResolved{
		policy:            res,
		caps:              strings.Join(res.Effects, ","),
		netDomains:        strings.Join(res.NetAllow, ","),
		netAllowHTTP:      res.NetAllowHTTP,
		netAllowLocalhost: !res.Restricted() && listsLoopback(res.NetAllow),
		processAllow:      strings.Join(res.ProcessAllow, ","),
		sandbox:           res.Root,
		entry:             res.Entry,
		digest:            res.Digest,
		aiStub:            res.AIProvider == "stub",
	}
	if res.AIProvider != "" && res.AIProvider != "stub" {
		resolved.aiModel = res.AIProvider
	}
	if resolved.sandbox != "" {
		// The effects context reads the sandbox from the environment
		// (internal/config.FSSandbox). Setting it here — after the policy has
		// been read and before the runtime starts — is what makes the policy,
		// not the caller's environment, the authority.
		os.Setenv(config.EnvFSSandbox, resolved.sandbox)
	}
	activeRunPolicy = res

	graphDigest, graphFiles, graphBytes := loader.SourceSnapshotDigest()
	line := admissionLine{
		OK: true, Policy: policyPath, PolicyDigest: res.Digest, Mode: res.Mode, Caps: res.Effects,
		FSSandbox: res.Root, NetAllow: res.NetAllow, ProcessAllow: res.ProcessAllow, AIProvider: res.AIProvider,
		Entry: res.Entry, TimeoutMs: res.Timeout.Milliseconds(), Budgets: res.Budgets,
		Limits: map[string]int64{
			"max_source_bytes": res.MaxSourceBytes, "max_module_graph_bytes": res.MaxModuleGraphBytes,
			"max_output_bytes": res.MaxOutputBytes, "max_fs_transfer_bytes": res.MaxFSTransferBytes,
		},
		ModuleGraph: moduleGraphInfo{Digest: graphDigest, Files: graphFiles, Bytes: graphBytes},
		Decision:    out.Decision,
	}
	reportDecision(control, controlMessage{Kind: "admitted", Admission: &line})
	if control == nil {
		b, _ := json.Marshal(line)
		fmt.Fprintf(os.Stderr, "policy: %s\n", b)
	}
	return resolved
}

// admissionLine is the banked, machine-readable record of what the run is
// bound by: effective mode, policy and module-graph digests, caps, limits.
type admissionLine struct {
	OK           bool             `json:"ok"`
	Policy       string           `json:"policy"`
	PolicyDigest string           `json:"policy_digest"`
	Mode         string           `json:"security_mode"`
	Caps         []string         `json:"caps"`
	FSSandbox    string           `json:"fs_sandbox"`
	NetAllow     []string         `json:"net_allow"`
	ProcessAllow []string         `json:"process_allow"`
	AIProvider   string           `json:"ai_provider"`
	Entry        string           `json:"entry"`
	TimeoutMs    int64            `json:"timeout_ms"`
	Budgets      map[string]int   `json:"budgets,omitempty"`
	Limits       map[string]int64 `json:"limits"`
	ModuleGraph  moduleGraphInfo  `json:"module_graph"`
	Decision     policy.Decision  `json:"decision"`
}

type moduleGraphInfo struct {
	Digest string `json:"digest"`
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
}

// controlMessage is one line on the worker→supervisor control pipe.
type controlMessage struct {
	Kind      string            `json:"kind"` // admitted | denied
	Code      int               `json:"code,omitempty"`
	Decision  policyCheckOutput `json:"decision,omitempty"`
	Admission *admissionLine    `json:"admission,omitempty"`
}

// reportDecision writes the decision on the control pipe when there is one.
func reportDecision(control *os.File, msg controlMessage) {
	if control == nil {
		return
	}
	b, _ := json.Marshal(msg)
	_, _ = control.Write(append(b, '\n'))
	_ = control.Close()
}

// limitEnvelope is the versioned result envelope for a runtime denial or
// limit: stage, reason, policy digest, safe message. Never program output.
func limitEnvelope(res *policy.Resolved, stage, reason, message string) string {
	b, _ := json.Marshal(map[string]any{
		"version":       1,
		"stage":         stage,
		"reason":        reason,
		"message":       message,
		"policy_digest": res.Digest,
		"security_mode": res.Mode,
	})
	return string(b)
}

// listsLoopback reports whether an allowlist names the loopback host by
// name or literal — the explicit grant trusted_host honours.
func listsLoopback(allow []string) bool {
	for _, h := range allow {
		switch strings.ToLower(strings.TrimSuffix(h, ".")) {
		case "localhost", "127.0.0.1", "::1", "[::1]":
			return true
		}
	}
	return false
}
