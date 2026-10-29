package eval_harness

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// MaintenanceToolchainEvidence binds observed version output and executable
// bytes. The evidence file's hash is the native ledger's nativeToolDigest.
type MaintenanceToolchainEvidence struct {
	Schema         string   `json:"schema"`
	Command        []string `json:"command"`
	VersionCommand []string `json:"versionCommand"`
	Version        string   `json:"version"`
	BinarySHA256   string   `json:"binarySHA256"`
}

func maintenanceReadToolchainEvidence(tc MaintenanceToolchain) (*MaintenanceToolchainEvidence, error) {
	data, err := maintenanceReadBoundFile(tc.Evidence)
	if err != nil {
		return nil, err
	}
	var evidence MaintenanceToolchainEvidence
	if err = maintenanceStrictJSON(data, &evidence); err != nil {
		return nil, err
	}
	if evidence.Schema != "maintenance.toolchain/1" || len(evidence.Command) != 1 || !slices.Equal(evidence.Command, tc.Command) || !maintenanceCommand(evidence.VersionCommand) || evidence.VersionCommand[0] != evidence.Command[0] || strings.TrimSpace(evidence.Version) == "" || evidence.Version != tc.Version || !maintenanceSHA(evidence.BinarySHA256) {
		return nil, fmt.Errorf("native toolchain command/version/binary evidence differs")
	}
	return &evidence, nil
}

// Static validation does not resolve binaries or run version commands: dry run
// binds evidence without claiming that the current machine matches it.
func maintenanceValidateToolchainEvidence(tc MaintenanceToolchain) error {
	_, err := maintenanceReadToolchainEvidence(tc)
	return err
}

func verifyMaintenanceToolchain(ctx context.Context, e *MaintenanceExecution, tr MaintenanceTrajectoryPlan) (string, error) {
	for _, tc := range e.Manifest.Toolchains {
		if tc.Language != tr.Language {
			continue
		}
		if tc.Evidence.SHA256 != tr.NativeToolDigest {
			return "", fmt.Errorf("trajectory/native toolchain binding differs")
		}
		frozen, err := maintenanceReadToolchainEvidence(tc)
		if err != nil {
			return "", err
		}
		path, err := exec.LookPath(frozen.Command[0])
		if err != nil {
			return "", err
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		digest, err := maintenanceFileSHA(path)
		if err != nil {
			return "", err
		}
		if digest != frozen.BinarySHA256 {
			return "", fmt.Errorf("native runtime binary differs from frozen evidence")
		}
		seconds := e.Manifest.Limits.MaxStageSeconds
		if seconds <= 0 {
			return "", fmt.Errorf("native toolchain verification bound required")
		}
		call, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
		version, err := maintenanceRunCommand(call, "", path, frozen.VersionCommand[1:]...)
		cancel()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(string(version)) != strings.TrimSpace(frozen.Version) {
			return "", fmt.Errorf("native runtime version differs from frozen evidence")
		}
		after, err := maintenanceFileSHA(path)
		if err != nil {
			return "", err
		}
		if after != digest {
			return "", fmt.Errorf("native runtime changed during version verification")
		}
		return digest, nil
	}
	return "", fmt.Errorf("native toolchain missing for %s", tr.Language)
}
