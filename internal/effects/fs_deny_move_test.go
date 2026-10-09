package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func moveTestContext(t *testing.T, pattern string) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = t.TempDir()
	ctx.Env.DenyWrite = []string{pattern}
	ctx.Grant(NewCapability("FS"))
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })
	return ctx
}

func moveTestCall(ctx *EffContext, op string, paths ...string) (eval.Value, error) {
	args := make([]eval.Value, len(paths))
	for i, p := range paths {
		args[i] = &eval.StringValue{Value: p}
	}
	return Call(ctx, "FS", op, args)
}

func assertMoveDenied(t *testing.T, v eval.Value, err error, result bool) {
	t.Helper()
	if result {
		tag, ok := v.(*eval.TaggedValue)
		if err != nil || !ok || tag.CtorName != "Err" || len(tag.Fields) != 1 {
			t.Fatalf("expected Err, got %v %v", v, err)
		}
		msg, ok := tag.Fields[0].(*eval.StringValue)
		if !ok || !strings.Contains(msg.Value, "E_FS_PROTECTED") {
			t.Fatalf("expected security Err, got %v", v)
		}
	} else if err == nil || !strings.Contains(err.Error(), "E_FS_PROTECTED") {
		t.Fatalf("expected security denial, got %v %v", v, err)
	}
}

func TestFSDenyMove_RenameOperands(t *testing.T) {
	cases := []struct{ pattern, dir, file string }{
		{".claude/settings.json", ".claude", "settings.json"},
		{"a/b/**", "a", "b/x.txt"},
		{"a/*/x.txt", "a/b", "x.txt"},
	}
	for _, tc := range cases {
		for _, op := range []string{"renameFile", "renameFileResult"} {
			for _, destination := range []bool{false, true} {
				name := tc.pattern + op
				if destination {
					name += "destination"
				}
				t.Run(name, func(t *testing.T) {
					ctx := moveTestContext(t, tc.pattern)
					from, to := tc.dir, "tmp"
					if destination {
						from, to = "tmp", tc.dir
					}
					file := filepath.Join(ctx.Env.Sandbox, from, tc.file)
					if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(file, []byte("orig"), 0644); err != nil {
						t.Fatal(err)
					}
					// Make the destination's parent available so this would succeed without the gate.
					if err := os.MkdirAll(filepath.Dir(filepath.Join(ctx.Env.Sandbox, to)), 0755); err != nil {
						t.Fatal(err)
					}
					v, err := moveTestCall(ctx, op, from, to)
					assertMoveDenied(t, v, err, strings.HasSuffix(op, "Result"))
					if b, err := os.ReadFile(file); err != nil || string(b) != "orig" {
						t.Fatalf("source changed: %q %v", b, err)
					}
					if _, err := os.Stat(filepath.Join(ctx.Env.Sandbox, to)); !os.IsNotExist(err) {
						t.Fatalf("destination changed: %v", err)
					}
				})
			}
		}
	}
}

func TestFSDenyMove_Removals(t *testing.T) {
	for _, pattern := range []string{"a/x.txt", "a/b/**", "a/*/x.txt"} {
		for _, op := range []string{"removeFile", "removeFileResult", "removeDirResult", "FSRemove"} {
			t.Run(pattern+op, func(t *testing.T) {
				ctx := moveTestContext(t, pattern)
				dir := filepath.Join(ctx.Env.Sandbox, "a")
				if err := os.Mkdir(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if op == "FSRemove" {
					assertMoveDenied(t, nil, ctx.FSRemove("a"), false)
				} else {
					v, err := moveTestCall(ctx, op, "a")
					assertMoveDenied(t, v, err, strings.HasSuffix(op, "Result"))
				}
				if info, err := os.Stat(dir); err != nil || !info.IsDir() {
					t.Fatalf("empty ancestor removed: %v", err)
				}
			})
		}
	}
}

func TestFSDenyMove_AllowedOperations(t *testing.T) {
	for _, pattern := range []string{"*.yml", "Makefile", "protected/x.txt", ""} {
		t.Run(pattern, func(t *testing.T) {
			ctx := moveTestContext(t, pattern)
			if pattern == "" {
				ctx.Env.DenyWrite = nil
			}
			for _, op := range []string{"mkdir", "mkdirAll", "mkdirResult", "mkdirAllResult"} {
				v, err := moveTestCall(ctx, op, "protected")
				if err != nil {
					t.Fatal(err)
				}
				if tag, ok := v.(*eval.TaggedValue); ok && tag.CtorName != "Ok" {
					t.Fatalf("mkdir: %v", v)
				}
				if err := os.Remove(filepath.Join(ctx.Env.Sandbox, "protected")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := moveTestCall(ctx, "writeFile", "protected", "ancestor file"); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(ctx.Env.Sandbox, "protected")); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(ctx.Env.Sandbox, "src"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(ctx.Env.Sandbox, "src", "x.yml"), []byte("orig"), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := moveTestCall(ctx, "renameFile", "src", "moved"); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(filepath.Join(ctx.Env.Sandbox, "moved", "x.yml"))
			if err != nil || string(b) != "orig" {
				t.Fatalf("move: %q %v", b, err)
			}
			if pattern == "*.yml" {
				v, err := moveTestCall(ctx, "writeFile", "moved/x.yml", "evil")
				assertMoveDenied(t, v, err, false)
			}
			if _, err := moveTestCall(ctx, "writeFile", "run.json.tmp", "publish"); err != nil {
				t.Fatal(err)
			}
			if v, err := moveTestCall(ctx, "renameFileResult", "run.json.tmp", "run.json"); err != nil || v.(*eval.TaggedValue).CtorName != "Ok" {
				t.Fatalf("publish: %v %v", v, err)
			}
			if b, err := os.ReadFile(filepath.Join(ctx.Env.Sandbox, "run.json")); err != nil || string(b) != "publish" {
				t.Fatalf("publish content: %q %v", b, err)
			}
		})
	}
}
