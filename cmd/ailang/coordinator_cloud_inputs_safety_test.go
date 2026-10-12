package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func TestTaskInputsBatchCollisionsAndCancellation(t *testing.T) {
	wsOverride, _, fetchOverride := inputFixture(t)
	writeInputFile(t, wsOverride, ".gitignore", []byte("!.incoming/\n"))
	if _, err := fetchOverride.fetch(context.Background(), wsOverride, []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload"}}); err == nil || !strings.Contains(err.Error(), "not ignored") {
		t.Fatalf("ignore override did not fail loudly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wsOverride, ".incoming")); !os.IsNotExist(err) {
		t.Fatal("unexcluded default input delivered")
	}
	for _, name := range []string{".gitignore", "nested/.gitattributes", ".gitmodules", ".gitconfig"} {
		ws, src, f := inputFixture(t)
		writeInputFile(t, src, name, []byte("!.incoming/\n"))
		inputs := []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: ""}, {Repo: "org/data", Ref: "main", Path: name, Dest: name}}
		if _, err := f.fetch(context.Background(), ws, inputs); err == nil {
			t.Fatalf("explicit git control file accepted: %s", name)
		}
		if _, err := os.Stat(filepath.Join(ws, ".incoming")); !os.IsNotExist(err) {
			t.Fatal("mixed batch placed default data before metadata refusal")
		}
		if _, err := f.fetch(context.Background(), ws, inputs[:1]); err != nil {
			t.Fatalf("default git control data refused: %v", err)
		}
	}
	for _, dest := range []string{"images/", "images/photo.png"} {
		ws, _, f := inputFixture(t)
		inputs := []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload/photo.png", Dest: dest}, {Repo: "org/data", Ref: "main", Path: "payload/photo.png", Dest: dest}}
		if _, err := f.fetch(context.Background(), ws, inputs); err == nil {
			t.Fatal("duplicate destination accepted")
		}
		if _, err := os.Stat(filepath.Join(ws, "images")); !os.IsNotExist(err) {
			t.Fatal("collision delivered bytes")
		}
	}
	ws, _, f := inputFixture(t)
	inputs := []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload/photo.png", Dest: "images"}, {Repo: "org/data", Ref: "main", Path: "payload/photo.png", Dest: "images/photo.png"}}
	if _, err := f.fetch(context.Background(), ws, inputs); err == nil {
		t.Fatal("file/parent collision accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.fetch(ctx, ws, inputs[:1]); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := f.fetch(context.Background(), ws, []messaging.TaskInput{{Repo: "../bad", Ref: "main"}}); err == nil {
		t.Fatal("malformed input accepted")
	}
	writeInputFile(t, ws, "images", []byte("keep"))
	if _, err := f.fetch(context.Background(), ws, []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload", Dest: "images/"}}); err == nil {
		t.Fatal("non-directory ancestor accepted")
	}
	if _, err := f.fetch(context.Background(), t.TempDir(), []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload"}}); err == nil || !strings.Contains(err.Error(), "exclude") {
		t.Fatalf("default exclusion must fail loudly: %v", err)
	}
}

func TestTaskInputsDeliveryRollback(t *testing.T) {
	ws, src := t.TempDir(), t.TempDir()
	writeInputFile(t, src, "first", []byte("a"))
	files := []stagedInputFile{{source: filepath.Join(src, "first"), dest: "new/first"}, {source: filepath.Join(src, "missing"), dest: "new/second", index: 1}}
	if err := deliverTaskInputFiles(context.Background(), ws, files); err == nil {
		t.Fatal("missing stage file succeeded")
	}
	if _, err := os.Stat(filepath.Join(ws, "new")); !os.IsNotExist(err) {
		t.Fatal("rollback retained input files/directory")
	}
	writeInputFile(t, ws, "existing/keep", []byte("keep"))
	files[0].dest = "existing/first"
	files[1].dest = "existing/second"
	if err := deliverTaskInputFiles(context.Background(), ws, files); err == nil {
		t.Fatal("missing stage file succeeded")
	}
	if data, err := os.ReadFile(filepath.Join(ws, "existing/keep")); err != nil || string(data) != "keep" {
		t.Fatal("rollback touched existing workspace data")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := deliverTaskInputFiles(ctx, ws, files); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	files[0].dest = "existing/keep"
	if err := deliverTaskInputFiles(context.Background(), ws, files[:1]); err == nil {
		t.Fatal("delivery overwrote existing file")
	}
	writeInputFile(t, ws, "not-dir", []byte("keep"))
	files[0].dest = "not-dir/child"
	if err := deliverTaskInputFiles(context.Background(), ws, files[:1]); err == nil {
		t.Fatal("non-directory ancestor accepted")
	}
}

func TestTaskInputsStageReadFailuresAndFilePin(t *testing.T) {
	ws, _, f := inputFixture(t)
	input := messaging.TaskInput{Repo: "org/data", Ref: "main", Path: "payload/photo.png", Dest: "public/", SHA256: "A5F56A2CE564CF7B5E181BE1FD959CE349965E011B9166E4F804D793B797885A"}
	// Use the actual fixture digest rather than duplicating a checksum algorithm.
	_, src, _ := inputFixture(t)
	remaining := int64(100)
	digest, err := stageInputFile(context.Background(), filepath.Join(src, "payload/photo.png"), filepath.Join(t.TempDir(), "file"), &remaining)
	if err != nil {
		t.Fatal(err)
	}
	input.SHA256 = strings.ToUpper(digest)
	got, err := f.fetch(context.Background(), ws, []messaging.TaskInput{input})
	if err != nil || len(got[0].Verified) != 1 {
		t.Fatalf("file pin: %v %v", got, err)
	}
	if appendInputReport("", got) == "" {
		t.Fatal("empty executor summary lost provenance")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	remaining = 100
	if _, err := stageInputFile(ctx, filepath.Join(src, "payload/photo.png"), filepath.Join(t.TempDir(), "file"), &remaining); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := stageInputFile(context.Background(), "/nonexistent-input", filepath.Join(t.TempDir(), "file"), &remaining); err == nil {
		t.Fatal("missing source accepted")
	}
	dest := filepath.Join(t.TempDir(), "file")
	writeInputFile(t, filepath.Dir(dest), "file", []byte("keep"))
	if _, err := stageInputFile(context.Background(), filepath.Join(src, "payload/photo.png"), dest, &remaining); err == nil {
		t.Fatal("stage overwrite accepted")
	}
	if _, err := stageInputFile(context.Background(), filepath.Join(src, "payload/photo.png"), filepath.Join(dest, "nested"), &remaining); err == nil {
		t.Fatal("invalid stage parent accepted")
	}
	for _, line := range []string{"", strings.Repeat("z", 64) + "  file", strings.Repeat("0", 64) + "  dir/", strings.Repeat("0", 64) + "  "} {
		stage := t.TempDir()
		writeInputFile(t, stage, "manifest.sha256", []byte(line))
		if _, err := verifyInputManifest(stage, map[string]string{"manifest.sha256": "digest"}); err == nil {
			t.Fatalf("malformed manifest accepted: %q", line)
		}
	}
}
