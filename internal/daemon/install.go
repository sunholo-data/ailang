package daemon

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed plist.tmpl
var plistTmplData string

const (
	plistLabel    = "com.sunholo.ailang.daemon"
	plistFilename = plistLabel + ".plist"
)

// launchctlRun is the indirection point so tests can substitute a fake.
// Default: real `launchctl` invocation.
var launchctlRun = func(args ...string) ([]byte, error) {
	return exec.Command("launchctl", args...).CombinedOutput()
}

// InstallOpts controls plist generation.
type InstallOpts struct {
	Env        string // dev|test|prod
	BinaryPath string // absolute path to the ailang binary
	Force      bool   // overwrite existing plist
}

// Install renders and loads the launchd plist for the daemon. Returns an
// error if a plist already exists and Force is false.
func Install(opts InstallOpts) error {
	mapping, ok := EnvProject[opts.Env]
	if !ok {
		return fmt.Errorf("unknown env %q (want dev|test|prod)", opts.Env)
	}
	plistPath, err := plistInstallPath()
	if err != nil {
		return err
	}
	if !opts.Force {
		if _, err := os.Stat(plistPath); err == nil {
			return fmt.Errorf("daemon already installed at %s (use --force to overwrite)", plistPath)
		}
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	tmpl, err := template.New("plist").Parse(plistTmplData)
	if err != nil {
		return fmt.Errorf("parse plist template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"BinaryPath": opts.BinaryPath,
		"Env":        opts.Env,
		"Project":    mapping.Project,
	}); err != nil {
		return fmt.Errorf("render plist: %w", err)
	}
	if err := os.WriteFile(plistPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	// Best-effort load; failure here doesn't undo the file write.
	if _, err := launchctlRun("load", plistPath); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}
	return nil
}

// Uninstall unloads the daemon (if loaded) and removes the plist. Missing
// plist is treated as a no-op.
func Uninstall() error {
	plistPath, err := plistInstallPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return nil
	}
	// Best-effort unload — ignore error since the file removal is what matters.
	_, _ = launchctlRun("unload", plistPath)
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}

// Status returns a one-line human-readable summary of the daemon's launchd
// state. Does not return error for "not installed" — that's reported in the
// returned string.
func Status() (string, error) {
	out, err := launchctlRun("list", plistLabel)
	if err != nil {
		// launchctl returns non-zero for "not loaded" — treat as not installed.
		return "ailang daemon: not installed (launchctl error)", nil
	}
	body := strings.TrimSpace(string(out))
	if body == "" {
		return "ailang daemon: not installed (launchctl list returned empty)", nil
	}
	return fmt.Sprintf("ailang daemon: running\n%s", body), nil
}

func plistInstallPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", plistFilename), nil
}
