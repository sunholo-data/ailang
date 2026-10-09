package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/gitexec"
	"github.com/sunholo-data/ailang/internal/messaging"
)

const maxTaskInputBytes int64 = 256 * 1024 * 1024
const maxTaskInputReportBytes = 64 * 1024

type taskInputProvenance struct {
	Repo     string            `json:"repo"`
	Ref      string            `json:"ref"`
	Commit   string            `json:"commit"`
	Dest     string            `json:"dest"`
	Digests  map[string]string `json:"sha256"`
	Verified []string          `json:"verified,omitempty"`
}

type taskInputFetcher struct {
	clone func(context.Context, messaging.TaskInput, string) (string, error)
	limit int64
}

// fetchTaskInputs runs only in the Cloud Run parent. No clone or credential
// material survives into the executor workspace. Dispatch already authorized
// the exact repos against the trusted registry; transport is validated again.
func fetchTaskInputs(ctx context.Context, workspace string, inputs []messaging.TaskInput) ([]taskInputProvenance, error) {
	return (taskInputFetcher{clone: cloneTaskInput, limit: maxTaskInputBytes}).fetch(ctx, workspace, inputs)
}

func cloneTaskInput(ctx context.Context, input messaging.TaskInput, dest string) (string, error) {
	if config.GitHubToken() == "" {
		return "", fmt.Errorf("GITHUB_TOKEN required for parent input reads")
	}
	// Reset inherited helpers, including SSH workspace credentials. This helper
	// is command-scoped and reads the parent token without recording its value.
	cmd := gitexec.CommandContext(ctx, taskInputCloneArgs(input, dest)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("clone %s@%s: %w", input.Repo, input.Ref, err)
	}
	return gitEvidenceOutput(ctx, dest, "rev-parse", "HEAD")
}

func taskInputCloneArgs(input messaging.TaskInput, dest string) []string {
	return []string{"-c", "credential.helper=", "-c", "credential.helper=" + envTokenCredentialHelper,
		"clone", "--depth", "1", "--single-branch", "--branch", input.Ref, "--", "https://github.com/" + input.Repo + ".git", dest}
}

func taskInputReport(provenance []taskInputProvenance) string {
	if len(provenance) == 0 {
		return ""
	}
	// All fields have concrete JSON types; marshaling cannot fail.
	data, _ := json.Marshal(provenance)
	return "Task input provenance: " + string(data)
}

func appendInputReport(summary string, inputs []taskInputProvenance) string {
	report := taskInputReport(inputs)
	if report == "" {
		return summary
	}
	if summary == "" {
		return report
	}
	return summary + "\n\n" + report
}

func (f taskInputFetcher) fetch(ctx context.Context, workspace string, inputs []messaging.TaskInput) ([]taskInputProvenance, error) {
	if err := messaging.ValidateTaskInputs(inputs); err != nil {
		return nil, err
	}
	if len(inputs) == 0 {
		return nil, nil
	}
	temp, err := os.MkdirTemp("", "ailang-task-inputs-")
	if err != nil {
		return nil, fmt.Errorf("stage inputs: %w", err)
	}
	defer os.RemoveAll(temp)
	var files []stagedInputFile
	var reports []taskInputProvenance
	remaining := f.limit
	for i, input := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("inputs[%d]: %w", i, err)
		}
		clone := filepath.Join(temp, fmt.Sprintf("clone-%d", i))
		commit, err := f.clone(ctx, input, clone)
		if err != nil {
			return nil, fmt.Errorf("inputs[%d]: %w", i, err)
		}
		staged, report, err := stageTaskInput(ctx, temp, clone, input, i, &remaining)
		if err != nil {
			return nil, fmt.Errorf("inputs[%d]: %w", i, err)
		}
		files = append(files, staged...)
		reports = append(reports, report)
		reports[i].Commit = commit
	}
	if len(taskInputReport(reports)) > maxTaskInputReportBytes {
		return nil, fmt.Errorf("inputs: provenance exceeds %d bytes", maxTaskInputReportBytes)
	}
	// Preflight the entire batch before creating any agent-visible files.
	seen := map[string]bool{}
	for _, file := range files {
		if seen[file.dest] {
			return nil, fmt.Errorf("inputs[%d]: duplicate destination %q", file.index, file.dest)
		}
		seen[file.dest] = true
		if err := checkInputDestination(workspace, file.dest, file.explicit); err != nil {
			return nil, fmt.Errorf("inputs[%d]: %w", file.index, err)
		}
	}
	// Cross-input parent/file collisions are invisible to filesystem preflight.
	for _, file := range files {
		for parent := filepath.Dir(file.dest); parent != "."; parent = filepath.Dir(parent) {
			if seen[parent] {
				return nil, fmt.Errorf("inputs[%d]: destination parent %q is an input file", file.index, parent)
			}
		}
	}
	for _, input := range inputs {
		if input.Dest == "" {
			if err := excludeFromGit(workspace, ".incoming/"); err != nil {
				return nil, fmt.Errorf("exclude inputs: %w", err)
			}
			break
		}
	}
	// A tracked .gitignore can override info/exclude. Prove each default
	// file is ignored before delivery rather than trust pattern precedence.
	for _, file := range files {
		if !file.explicit {
			if err := gitexec.CommandContext(ctx, "-C", workspace, "check-ignore", "-q", "--", file.dest).Run(); err != nil {
				return nil, fmt.Errorf("inputs[%d]: default destination %q is not ignored by git: %w", file.index, file.dest, err)
			}
		}
	}
	if err := deliverTaskInputFiles(ctx, workspace, files); err != nil {
		return nil, err
	}
	fmt.Printf("execute-job: %s\n", taskInputReport(reports))
	return reports, nil
}

// Instructions and harness metadata can never be introduced as task data,
// even under a nested directory. Matching is case-insensitive across platforms.
func validateInputDataPath(rel string) error {
	if err := messaging.ValidateInputPath(filepath.ToSlash(rel)); err != nil {
		return err
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		switch strings.ToLower(part) {
		case ".git", ".claude", ".pi", ".agents", ".codex", ".gemini", ".cursor", ".mcp.json", "agents.md", "claude.md", "gemini.md", ".cursorrules", ".windsurfrules", "copilot-instructions.md":
			return fmt.Errorf("harness instruction/metadata path refused: %q", rel)
		}
	}
	return nil
}

// Lstat every existing component: Stat on the final file alone follows ancestor
// symlinks and would let a syntactically relative source/destination escape.
func checkInputAncestors(root, rel string) error {
	current := root
	parts := append([]string{""}, strings.Split(filepath.FromSlash(strings.TrimSuffix(rel, "/")), string(filepath.Separator))...)
	for _, part := range parts {
		if part != "" {
			current = filepath.Join(current, part)
		}
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink refused: %s", current)
		}
	}
	return nil
}

func checkInputDestination(workspace, rel string, explicit bool) error {
	if err := validateInputDestination(rel, explicit); err != nil {
		return err
	}
	if err := checkInputAncestors(workspace, rel); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(workspace, rel)); err == nil {
		return fmt.Errorf("existing destination refused: %q", rel)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Explicit files participate in git operations. Default excluded copies may
// contain repository control files as data, but explicit copies cannot change
// ignore rules, filters or submodule configuration.
func validateInputDestination(rel string, explicit bool) error {
	if err := validateInputDataPath(rel); err != nil {
		return err
	}
	if explicit {
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			switch strings.ToLower(part) {
			case ".gitignore", ".gitattributes", ".gitmodules", ".gitconfig":
				return fmt.Errorf("git control destination refused: %q", rel)
			}
		}
	}
	return nil
}
