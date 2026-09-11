//go:build !js

package effects

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-PROCESS-SUBCMD M1: `cmd:sub[:sub…]` entries narrow a binary to subcommand chains.
func TestProcessContext_ResolveAllowlist_Subcommands(t *testing.T) {
	cases := []struct {
		name    string
		list    string
		wantCmd []string              // keys expected in Allowlist
		wantSub map[string][][]string // expected Subcommands (nil entry = unrestricted)
	}{
		{"single chain", "echo:status",
			[]string{"echo"}, map[string][][]string{"echo": {{"status"}}}},
		{"two chains same binary", "echo:pr:list,echo:pr:view",
			[]string{"echo"}, map[string][][]string{"echo": {{"pr", "list"}, {"pr", "view"}}}},
		{"bare wins over narrowed", "echo,echo:status",
			[]string{"echo"}, map[string][][]string{}},
		{"narrowed then bare also bare wins", "echo:status,echo",
			[]string{"echo"}, map[string][][]string{}},
		{"star is bare", "echo:*",
			[]string{"echo"}, map[string][][]string{}},
		{"absolute path keeps path as key", "/bin/echo:status",
			[]string{"/bin/echo"}, map[string][][]string{"/bin/echo": {{"status"}}}},
		{"mixed bare and narrowed", "date,echo:hello",
			[]string{"date", "echo"}, map[string][][]string{"echo": {{"hello"}}}},
		{"legacy list unchanged", "echo,date",
			[]string{"echo", "date"}, map[string][][]string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := NewProcessContext()
			if err := pc.ResolveAllowlist(tc.list); err != nil {
				t.Fatalf("ResolveAllowlist(%q): %v", tc.list, err)
			}
			for _, k := range tc.wantCmd {
				if _, ok := pc.Allowlist[k]; !ok {
					t.Errorf("Allowlist missing %q (have %v)", k, pc.Allowlist)
				}
			}
			if len(pc.Allowlist) != len(tc.wantCmd) {
				t.Errorf("Allowlist has %d keys, want %d: %v", len(pc.Allowlist), len(tc.wantCmd), pc.Allowlist)
			}
			got := pc.Subcommands
			if got == nil {
				got = map[string][][]string{}
			}
			if !reflect.DeepEqual(got, tc.wantSub) {
				t.Errorf("Subcommands = %v, want %v", got, tc.wantSub)
			}
		})
	}
}

// An empty subcommand segment is a startup error naming the entry — never a silent allow.
func TestProcessContext_ResolveAllowlist_MalformedSubcommand(t *testing.T) {
	for _, bad := range []string{"echo:", "echo::status", ":status", "echo:status:"} {
		pc := NewProcessContext()
		err := pc.ResolveAllowlist("date," + bad)
		if err == nil {
			t.Errorf("ResolveAllowlist(%q): want error, got nil (Allowlist=%v Subcommands=%v)", bad, pc.Allowlist, pc.Subcommands)
			continue
		}
		if !strings.Contains(err.Error(), bad) {
			t.Errorf("ResolveAllowlist(%q): error %q does not name the entry", bad, err)
		}
	}
}

// ---- M2: one authorizer, three exec paths -------------------------------

func TestProcessAuthorize_Subcommands(t *testing.T) {
	pc := NewProcessContext()
	if err := pc.ResolveAllowlist("echo:status,echo:pr:list,date"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		cmd    string
		args   []string
		allow  bool
		detail string // denial detail when !allow
	}{
		{"echo", []string{"status"}, true, ""},
		{"echo", []string{"status", "--short"}, true, ""},
		{"echo", []string{"pr", "list"}, true, ""},
		{"echo", []string{"pr", "list", "--limit", "3"}, true, ""},
		{"echo", []string{"push"}, false, "echo push"},
		{"echo", []string{"push", "--force"}, false, "echo push"},
		{"echo", []string{"pr", "merge"}, false, "echo pr merge"},
		{"echo", []string{"pr"}, false, "echo pr"},
		{"echo", nil, false, "echo"},
		{"echo", []string{"-C", "/x", "status"}, false, "echo -C /x"}, // positional: fail closed
		{"date", []string{"anything", "at", "all"}, true, ""},
		{"cat", []string{"status"}, false, "cat"},
	}
	for _, tc := range cases {
		path, denial := pc.Authorize(tc.cmd, tc.args)
		if tc.allow {
			if denial != nil {
				t.Errorf("Authorize(%q,%v): denied %s(%q), want allow", tc.cmd, tc.args, denial.Ctor, denial.Detail)
			} else if path == "" {
				t.Errorf("Authorize(%q,%v): allowed with empty path", tc.cmd, tc.args)
			}
			continue
		}
		if denial == nil {
			t.Errorf("Authorize(%q,%v): allowed, want NotAllowed(%q)", tc.cmd, tc.args, tc.detail)
			continue
		}
		if denial.Ctor != "NotAllowed" || denial.Detail != tc.detail {
			t.Errorf("Authorize(%q,%v) = %s(%q), want NotAllowed(%q)", tc.cmd, tc.args, denial.Ctor, denial.Detail, tc.detail)
		}
	}
}

// Bare entries keep today's contract exactly: unlisted → NotAllowed(cmd),
// listed-but-unresolved → NotFound(cmd), no allowlist → LookPath.
func TestProcessAuthorize_BareEntriesUnchanged(t *testing.T) {
	pc := NewProcessContext()
	if err := pc.ResolveAllowlist("echo,nonexistent_xyz"); err != nil {
		t.Fatal(err)
	}
	if _, d := pc.Authorize("echo", []string{"x"}); d != nil {
		t.Errorf("echo: %+v", d)
	}
	if _, d := pc.Authorize("ls", nil); d == nil || d.Ctor != "NotAllowed" || d.Detail != "ls" {
		t.Errorf("ls: %+v, want NotAllowed(ls)", d)
	}
	if _, d := pc.Authorize("nonexistent_xyz", nil); d == nil || d.Ctor != "NotFound" || d.Detail != "nonexistent_xyz" {
		t.Errorf("nonexistent_xyz: %+v, want NotFound", d)
	}
	var nilPC *ProcessContext
	if p, d := nilPC.Authorize("echo", nil); d != nil || p == "" {
		t.Errorf("nil context should LookPath: path=%q denial=%+v", p, d)
	}
}

// The bypass-closed proof: a subcommand allowlist is enforced by exec,
// spawnProcess AND asyncExecProcess. Patching only exec leaves
// spawnProcess("git", ["push"]) open, which is the failure mode #1137 is about.
func TestProcessAuthorize_AllThreePathsAgree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test requires unix")
	}
	newCtx := func() *EffContext {
		ctx := newProcessCtx()
		if err := ctx.Process.ResolveAllowlist("echo:hello"); err != nil {
			t.Fatal(err)
		}
		ctx.Stream = NewStreamContext()
		return ctx
	}
	argv := func(args ...string) *eval.ListValue {
		els := make([]eval.Value, len(args))
		for i, a := range args {
			els[i] = &eval.StringValue{Value: a}
		}
		return &eval.ListValue{Elements: els}
	}

	t.Run("exec", func(t *testing.T) {
		ctx := newCtx()
		ok, err := Call(ctx, "Process", "exec", []eval.Value{&eval.StringValue{Value: "echo"}, argv("hello")})
		if err != nil {
			t.Fatal(err)
		}
		unwrapOk(t, ok)
		res, err := Call(ctx, "Process", "exec", []eval.Value{&eval.StringValue{Value: "echo"}, argv("bye")})
		if err != nil {
			t.Fatal(err)
		}
		ctor, fields := unwrapErr(t, res)
		if ctor != "NotAllowed" || fields[0].(*eval.StringValue).Value != "echo bye" {
			t.Errorf("exec: got %s(%v), want NotAllowed(echo bye)", ctor, fields)
		}
	})

	t.Run("spawnProcess", func(t *testing.T) {
		ctx := newCtx()
		defer ctx.Process.CloseAllManaged()
		if _, err := ProcessSpawn(ctx, []eval.Value{&eval.StringValue{Value: "echo"}, argv("hello")}); err != nil {
			t.Fatalf("spawn echo hello: %v", err)
		}
		_, err := ProcessSpawn(ctx, []eval.Value{&eval.StringValue{Value: "echo"}, argv("bye")})
		if err == nil || !strings.Contains(err.Error(), "not allowed: echo bye") {
			t.Errorf("spawn echo bye: err=%v, want 'not allowed: echo bye'", err)
		}
	})

	t.Run("asyncExecProcess", func(t *testing.T) {
		ctx := newCtx()
		defer ctx.Stream.CloseAll()
		mk := func(a *eval.ListValue) []eval.Value {
			return []eval.Value{&eval.StringValue{Value: "echo"}, a, &eval.StringValue{Value: "src"}, &eval.IntValue{Value: 0}, &eval.IntValue{Value: 1024}}
		}
		if _, err := StreamAsyncExecProcess(ctx, mk(argv("hello"))); err != nil {
			t.Fatalf("async echo hello: %v", err)
		}
		_, err := StreamAsyncExecProcess(ctx, mk(argv("bye")))
		if err == nil || !strings.Contains(err.Error(), "not allowed: echo bye") {
			t.Errorf("async echo bye: err=%v, want 'not allowed: echo bye'", err)
		}
	})
}
