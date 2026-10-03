package policytool

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/policy"
)

// fakeOutput makes every CLI op print stdout (and nothing on stderr).
func (f *fixture) fakeOutput(stdout string) {
	f.host.run = func(dir string, argv []string) (string, string, int) {
		f.runs = append(f.runs, append([]string{dir}, argv...))
		return stdout, "", 0
	}
}

// bigJSON is a valid JSON document of at least n bytes.
func bigJSON(n int) string {
	b, _ := json.Marshal(map[string]string{"pad": strings.Repeat("x", n)})
	return string(b)
}

// #1551: the cap is the policy's max_output_bytes, not a hard-coded 64 KiB —
// an ~83 KB builtins listing fits under the restricted default.
func TestPolicyTool_CLIOutputCapIsThePolicys(t *testing.T) {
	f := newFixture(t, "")
	doc := bigJSON(83_000)
	f.fakeOutput(doc)
	r := f.host.Dispatch(Request{Op: "builtins_list", Flags: map[string]string{"json": ""}})
	if !r.OK || r.Stdout != doc {
		t.Fatalf("83 KB JSON must come back whole under the %d-byte restricted default; got ok=%v len=%d refused=%q",
			policy.DefaultRestrictedMaxOutputBytes, r.OK, len(r.Stdout), r.Refused)
	}
	var v map[string]string
	if err := json.Unmarshal([]byte(r.Stdout), &v); err != nil {
		t.Fatalf("stdout is not parseable JSON: %v", err)
	}

	// A smaller policy cap applies to text output: truncated, and says so
	// with the byte counts.
	f = newFixture(t, "max_output_bytes = 1000\n")
	f.fakeOutput(strings.Repeat("y", 5000))
	r = f.host.Dispatch(Request{Op: "builtins_list"})
	if !r.OK || len(r.Stdout) > 1200 || !strings.Contains(r.Stdout, "truncated") || !strings.Contains(r.Stdout, "max_output_bytes") {
		t.Fatalf("text output over a 1000-byte cap: ok=%v len=%d tail=%q", r.OK, len(r.Stdout), tail(r.Stdout))
	}
}

// #1551: a --json op whose output exceeds the cap is a structured refusal,
// never a cut (unparseable) document.
func TestPolicyTool_JSONOverCapFailsLoudly(t *testing.T) {
	f := newFixture(t, "max_output_bytes = 1000\n")
	f.fakeOutput(bigJSON(5000))
	for _, req := range []Request{
		{Op: "builtins_list", Flags: map[string]string{"json": ""}},
		{Op: "docs_search", Query: "x", Flags: map[string]string{"json": ""}},
	} {
		r := f.host.Dispatch(req)
		if r.OK || r.Stdout != "" || !strings.Contains(r.Refused, "max_output_bytes") || !strings.Contains(r.Refused, "1000") {
			t.Errorf("%s --json over cap must refuse with no stdout: ok=%v stdout=%d refused=%q", req.Op, r.OK, len(r.Stdout), r.Refused)
		}
	}
}

// #1551: builtins_list admits the narrowing flags `ailang builtins list`
// itself takes, validated by kind.
func TestPolicyTool_BuiltinsListNarrowingFlags(t *testing.T) {
	f := newFixture(t, "")
	r := f.host.Dispatch(Request{Op: "builtins_list", Flags: map[string]string{"json": "", "module": "std/fs", "query": "read file"}})
	if !r.OK {
		t.Fatalf("%+v", r)
	}
	got := strings.Join(f.runs[len(f.runs)-1][1:], " ")
	if got != "builtins list --json --module std/fs --query read file" {
		t.Fatalf("argv = %q", got)
	}
	for _, bad := range []map[string]string{
		{"module": "../etc"},
		{"module": "/abs"},
		{"module": "--json"},
		{"query": "-x"},
		{"query": "a\nb"},
	} {
		if r := f.host.Dispatch(Request{Op: "builtins_list", Flags: bad}); r.OK {
			t.Errorf("flags %v admitted: %+v", bad, r)
		}
	}
}

// #1552: every CLI child carries the policy path, so the child knows it is
// confined (examples resolution then ignores cwd-relative corpora).
func TestPolicyTool_ChildEnvCarriesPolicy(t *testing.T) {
	f := newFixture(t, "")
	t.Setenv(config.EnvAgentPolicy, "/somewhere/else.toml")
	env := f.host.childEnv()
	want := config.EnvAgentPolicy + "=" + f.policyPath
	n := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, config.EnvAgentPolicy+"=") {
			n++
			if kv != want {
				t.Errorf("child env has %q, want %q", kv, want)
			}
		}
	}
	if n != 1 {
		t.Fatalf("child env must carry exactly one %s, has %d", config.EnvAgentPolicy, n)
	}
}

// examples_list forwards the flag `ailang examples list` defines (--tags);
// the schema used to admit --tag, which the child rejected.
func TestPolicyTool_ExamplesListTagsFlag(t *testing.T) {
	f := newFixture(t, "")
	if r := f.host.Dispatch(Request{Op: "examples_list", Flags: map[string]string{"tags": "recursion"}}); !r.OK {
		t.Fatalf("%+v", r)
	}
	if got := strings.Join(f.runs[len(f.runs)-1][1:], " "); got != "examples list --tags recursion" {
		t.Fatalf("argv = %q", got)
	}
}

func tail(s string) string {
	if len(s) > 80 {
		return s[len(s)-80:]
	}
	return s
}
