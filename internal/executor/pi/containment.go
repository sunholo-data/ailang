package pi

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sunholo-data/ailang/internal/executor"
)

const maintenanceFenceFile = "maintenance-fence.ts"
const maintenanceSandboxFile = "maintenance-sandbox.mjs"

func validateEvalContainmentTask(task *executor.Task) error {
	if !task.IsolateFromAmbientContext || !task.IsolateFromProjectExtensions || task.ResumeSessionID != "" || len(task.MCPServers) != 0 || task.AllowedTools == nil || len(task.AllowedTools) == 0 {
		return fmt.Errorf("pi: containment requires explicit isolated fresh-session native tool set without MCP")
	}
	tools := map[string]bool{"Read": true, "Write": true, "Edit": true, "Bash": true, "Grep": true, "Find": true, "Ls": true}
	for _, name := range task.AllowedTools {
		if !tools[name] {
			return fmt.Errorf("pi: containment refuses unqualified tool %q", name)
		}
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return fmt.Errorf("pi: containment unsupported on %s", runtime.GOOS)
	}
	_, err := executor.LoadEvalContainment(task.EvalContainment, task.Workspace)
	if err != nil {
		return err
	}
	return executor.ErrEvalContainmentAdmissionClosed
}
func evalContainmentArgs(task *executor.Task) ([]string, error) {
	if task == nil || task.EvalContainment == nil {
		return nil, nil
	}
	if err := validateEvalContainmentTask(task); err != nil {
		return nil, err
	}
	dir, err := materializeProfileAssets()
	if err != nil {
		return nil, err
	}
	return []string{"-e", filepath.Join(dir, maintenanceFenceFile)}, nil
}

// EvalContainmentPrefix is the trusted wrapper argv for a candidate compiler/app.
// The host grader expands {runtimeRoot} to its new owned scratch root. A returned
// prefix identifies policy; it is never sufficient qualification by itself.
func EvalContainmentPrefix(c *executor.EvalContainment, workspace string) ([]string, error) {
	if _, err := executor.LoadEvalContainment(c, workspace); err != nil {
		return nil, err
	}
	dir, err := materializeProfileAssets()
	if err != nil {
		return nil, err
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("pi containment requires node: %w", err)
	}
	node, err = filepath.EvalSymlinks(node)
	if err != nil {
		return nil, err
	}
	return []string{node, filepath.Join(dir, maintenanceSandboxFile), "--policy", c.PolicyPath, "--digest", c.PolicyDigest, "--network", "tools", "--runtime-root", "{runtimeRoot}"}, nil
}

// evalContainmentCommand returns a complete argv including executable. Nil
// policy is byte-for-byte the old command and contributes no extra environment.
func evalContainmentCommand(task *executor.Task, piPath string, args []string) ([]string, []string, error) {
	if task == nil || task.EvalContainment == nil {
		return append([]string{piPath}, args...), nil, nil
	}
	if err := validateEvalContainmentTask(task); err != nil {
		return nil, nil, err
	}
	prefix, err := EvalContainmentPrefix(task.EvalContainment, task.Workspace)
	if err != nil {
		return nil, nil, err
	}
	// The outer pi process owns only workspace writes and protected-root read
	// denial. Its network permits declared inference hosts; bash/app children use
	// the same wrapper in tools mode (no permitted domains) through M7.
	prefix = prefix[:len(prefix)-2]
	for i := range prefix {
		if prefix[i] == "tools" {
			prefix[i] = "inference"
		}
	}
	path, err := exec.LookPath(piPath)
	if err != nil {
		return nil, nil, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, nil, err
	}
	argv := append(prefix, "--", path)
	argv = append(argv, args...)
	dir, err := materializeProfileAssets()
	if err != nil {
		return nil, nil, err
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, nil, err
	}
	env := []string{executor.EvalContainmentPolicyEnv + "=" + task.EvalContainment.PolicyPath, executor.EvalContainmentDigestEnv + "=" + task.EvalContainment.PolicyDigest, executor.EvalContainmentWrapperEnv + "=" + filepath.Join(dir, maintenanceSandboxFile), "JITI_FS_CACHE=0"}
	return argv, env, nil
}
