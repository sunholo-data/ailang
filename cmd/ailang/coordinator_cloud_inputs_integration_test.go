package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/gitexec"
	"github.com/sunholo-data/ailang/internal/messaging"
)

func TestTaskInputsNoNetworkIngressToJob(t *testing.T) {
	ctx := context.Background()
	source, workspace := t.TempDir(), t.TempDir()
	runGit(t, source, "init", "-b", "incoming/demo")
	writeInputFile(t, source, "assets/poster.png", []byte{0, 255, 3})
	runGit(t, source, "add", ".")
	runGit(t, source, "commit", "-m", "fixture")
	runGit(t, source, "tag", "poster-v1")
	commit := runGit(t, source, "rev-parse", "HEAD")
	runGit(t, workspace, "init")
	inputs := []messaging.TaskInput{{Repo: "org/data", Ref: "incoming/demo", Path: "assets"}, {Repo: "org/data", Ref: "poster-v1", Path: "assets/poster.png", Dest: "public/poster.png"}}
	store, err := messaging.OpenStore(filepath.Join(t.TempDir(), "messages.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.InsertInboxMessage(&messaging.InboxMessage{ID: "msg-input", ToInbox: "site", FromAgent: "sender", Title: "poster", Payload: "publish", Inputs: inputs}); err != nil {
		t.Fatal(err)
	}
	watcher := coordinator.NewMessageWatcher(coordinator.NewInboxMessageAdapter(store, "site"), time.Hour)
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = watcher.Start(watchCtx) }()
	var task *coordinator.Task
	select {
	case task = <-watcher.Tasks():
	case <-time.After(5 * time.Second):
		t.Fatal("watcher didn't preserve ingress")
	}
	tasks, err := coordinator.NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer tasks.Close()
	record := &coordinator.TaskRecord{ID: task.ID, Title: task.Title, Content: task.Content, Inputs: task.Inputs}
	if err := tasks.CreateTask(ctx, record); err != nil {
		t.Fatal(err)
	}
	record, err = tasks.GetTask(ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.ValidateInputsForAgent(record.Inputs, &coordinator.AgentConfig{ID: "site", InputsAllow: []string{"org/data"}}); err != nil {
		t.Fatal(err)
	}
	// Model the production env boundary with the canonical dispatch payload.
	data, _ := json.Marshal(coordinator.DispatchParams{Inputs: record.Inputs}.Inputs)
	t.Setenv(config.EnvTaskInputs, string(data))
	decoded, err := messaging.DecodeTaskInputs([]byte(config.TaskInputs()))
	if err != nil {
		t.Fatal(err)
	}
	var clones []string
	sourcePath := filepath.ToSlash(source)
	if !strings.HasPrefix(sourcePath, "/") {
		sourcePath = "/" + sourcePath // file:///C:/... on Windows
	}
	f := taskInputFetcher{limit: maxTaskInputBytes, clone: func(ctx context.Context, input messaging.TaskInput, dest string) (string, error) {
		clones = append(clones, dest)
		cmd := gitexec.CommandContext(ctx, "clone", "--depth", "1", "--single-branch", "--branch", input.Ref, "--", "file://"+sourcePath, dest)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmtInputCloneError(out, err)
		}
		return gitEvidenceOutput(ctx, dest, "rev-parse", "HEAD")
	}}
	provenance, err := f.fetch(ctx, workspace, decoded)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".incoming/1/poster.png", "public/poster.png"} {
		data, err := os.ReadFile(filepath.Join(workspace, p))
		if err != nil || string(data) != string([]byte{0, 255, 3}) {
			t.Fatalf("%s: %v %v", p, data, err)
		}
	}
	for _, p := range clones {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("clone not cleaned: %s", p)
		}
	}
	if len(provenance) != 2 || provenance[0].Commit != commit || provenance[1].Commit != commit {
		t.Fatalf("provenance: %+v", provenance)
	}
	report := appendInputReport("completed", provenance)
	for _, want := range []string{"org/data", "incoming/demo", "poster-v1", commit, "public/poster.png", "sha256"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %s", want)
		}
	}
	if status := runGit(t, workspace, "status", "--porcelain"); strings.Contains(status, ".incoming") || !strings.Contains(status, "public") {
		t.Fatal(status)
	}
	if err := coordinator.ValidateInputsForAgent(record.Inputs, &coordinator.AgentConfig{ID: "site"}); err == nil {
		t.Fatal("empty trusted grant authorized inputs")
	}
}

func fmtInputCloneError(out []byte, err error) error {
	return fmt.Errorf("fixture clone: %w: %s", err, out)
}

func TestTaskInputsJobAndCredentialBoundary(t *testing.T) {
	t.Setenv(config.EnvTaskInputs, "broken")
	_, result, _, err := executeCloudTask(context.Background(), "never-clone", "site", "", "main", "", "", "", "", "1m")
	if err == nil || !strings.Contains(err.Error(), config.EnvTaskInputs) || result != nil {
		t.Fatalf("malformed env: %v", err)
	}
	t.Setenv("GITHUB_TOKEN", "parent-only-token")
	t.Setenv("AILANG_CHILD_GIT_CREDENTIALS", "")
	args := strings.Join(taskInputCloneArgs(messaging.TaskInput{Repo: "org/input", Ref: "poster-v1"}, "/tmp/clone"), " ")
	if !strings.Contains(args, "https://github.com/org/input.git") || !strings.Contains(args, "credential.helper= -c credential.helper=!") || strings.Contains(args, "parent-only-token") {
		t.Fatalf("unsafe input command: %s", args)
	}
	for _, repo := range []string{"https://github.com/org/workspace.git", "git@gh-deploy-key:org/workspace.git"} {
		task := &executor.Task{Workspace: t.TempDir()}
		cleanup, err := childGitCredential(task, repo)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		env, err := executor.BuildEnvironment(executor.EnvironmentOptions{Task: task, Executor: "codex"})
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range env {
			if strings.HasPrefix(entry, "GITHUB_TOKEN=") || strings.Contains(entry, "org/input") {
				t.Fatalf("input credential leaked to child: key %s", strings.SplitN(entry, "=", 2)[0])
			}
		}
		for _, scope := range task.GitCredentialScopes {
			if strings.Contains(scope, "org/input") {
				t.Fatal("input repo in child credential scope")
			}
		}
		if strings.HasPrefix(repo, "git@") && task.GitCredentialFile != "" {
			t.Fatal("SSH workspace child received token credential")
		}
	}
	if appendInputReport("unchanged", nil) != "unchanged" {
		t.Fatal("zero-input completion changed")
	}
	got, err := fetchTaskInputs(context.Background(), "absent", nil)
	if err != nil || len(got) != 0 {
		t.Fatal("zero-input fetch changed")
	}
	t.Setenv("GITHUB_TOKEN", "")
	if _, err := cloneTaskInput(context.Background(), messaging.TaskInput{Repo: "org/input", Ref: "main"}, "/never"); err == nil {
		t.Fatal("missing parent credential was accepted")
	}
}
