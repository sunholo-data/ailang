package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The fine-tune programme's held-out set (tools/finetune/heldout.json) names
// benchmark ids in two roles: core + stretch are the before/after EVAL list
// (the pilot scripts pass them to --benchmarks, which loads
// benchmarks/<id>.yml), and core + stretch + retired_excluded is the TRAINING
// exclusion (extract_lang.load_heldout, extract_traj, collect_onpolicy.sh).
// Retiring a benchmark used to leave its id in the eval list (unloadable) and
// its replacement out of the exclusion (a near-duplicate leaking into
// training). These invariants keep both roles consistent with benchmarks/.

type heldoutSet struct {
	Core            []string `json:"core"`
	Stretch         []string `json:"stretch"`
	RetiredExcluded []string `json:"retired_excluded"`
}

func loadHeldout(t *testing.T, root string) heldoutSet {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "tools", "finetune", "heldout.json"))
	if err != nil {
		t.Fatal(err)
	}
	var h heldoutSet
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatal(err)
	}
	return h
}

var versionSuffix = regexp.MustCompile(`_v[0-9]+$`)

// checkHeldout returns every violation of the held-out invariants against the
// benchmark specs under root.
func checkHeldout(root string, h heldoutSet) []string {
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(root, rel))
		return err == nil
	}
	excluded := map[string]bool{}
	for _, id := range append(append(append([]string{}, h.Core...), h.Stretch...), h.RetiredExcluded...) {
		excluded[id] = true
	}
	var bad []string
	for _, id := range append(append([]string{}, h.Core...), h.Stretch...) {
		if !exists(filepath.Join("benchmarks", id+".yml")) {
			if exists(filepath.Join("benchmarks", "retired", id+".yml")) {
				bad = append(bad, id+": retired benchmark in the held-out EVAL list (move it to retired_excluded and evaluate its successor)")
			} else {
				bad = append(bad, id+": not a benchmark (benchmarks/"+id+".yml missing)")
			}
		}
	}
	for _, id := range h.RetiredExcluded {
		if !exists(filepath.Join("benchmarks", "retired", id+".yml")) {
			bad = append(bad, id+": in retired_excluded but not under benchmarks/retired/")
		}
	}
	// Every version of a held-out task is the same task: each must be
	// excluded from training. Look for successors (<id>_vN) and predecessors
	// (<base> of an <base>_vN id), live or retired.
	specs, _ := filepath.Glob(filepath.Join(root, "benchmarks", "*.yml"))
	retired, _ := filepath.Glob(filepath.Join(root, "benchmarks", "retired", "*.yml"))
	for _, p := range append(specs, retired...) {
		id := strings.TrimSuffix(filepath.Base(p), ".yml")
		base := versionSuffix.ReplaceAllString(id, "")
		if base == id {
			continue
		}
		if (excluded[base] || anyVersionExcluded(excluded, base)) && !excluded[id] {
			bad = append(bad, id+": a version of held-out task "+base+" is missing from the training exclusion")
		}
		if excluded[id] && !excluded[base] && (exists(filepath.Join("benchmarks", base+".yml")) || exists(filepath.Join("benchmarks", "retired", base+".yml"))) {
			bad = append(bad, base+": earlier version of held-out "+id+" is missing from the training exclusion")
		}
	}
	return bad
}

func anyVersionExcluded(excluded map[string]bool, base string) bool {
	for id := range excluded {
		if versionSuffix.ReplaceAllString(id, "") == base {
			return true
		}
	}
	return false
}

func TestFinetuneHeldoutMatchesBenchmarks(t *testing.T) {
	root := filepath.Join("..", "..")
	if bad := checkHeldout(root, loadHeldout(t, root)); len(bad) > 0 {
		t.Fatalf("tools/finetune/heldout.json is inconsistent with benchmarks/:\n  %s", strings.Join(bad, "\n  "))
	}
}

// The invariants fire on the two failure shapes PR #2229 first introduced.
func TestFinetuneHeldoutCheckCatchesRetirementMistakes(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"benchmarks/retired/rle.yml", "benchmarks/rle_v2.yml", "benchmarks/other.yml"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte("id: x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]struct {
		h    heldoutSet
		want []string
	}{
		"retired id still evaluated, successor not excluded": {heldoutSet{Stretch: []string{"rle", "other"}},
			[]string{"rle: retired benchmark in the held-out EVAL list", "rle_v2: a version of held-out task rle"}},
		"successor evaluated, retired id dropped from exclusion": {heldoutSet{Stretch: []string{"rle_v2", "other"}},
			[]string{"rle: earlier version of held-out rle_v2"}},
		"retired_excluded names a live benchmark": {heldoutSet{Stretch: []string{"rle_v2"}, RetiredExcluded: []string{"rle", "other"}},
			[]string{"other: in retired_excluded but not under benchmarks/retired/"}},
		"consistent": {heldoutSet{Stretch: []string{"rle_v2", "other"}, RetiredExcluded: []string{"rle"}}, nil},
	}
	for name, c := range cases {
		got := checkHeldout(root, c.h)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %q, want %d violation(s) %q", name, got, len(c.want), c.want)
			continue
		}
		for i, w := range c.want {
			if !strings.HasPrefix(got[i], w) {
				t.Errorf("%s: violation %d = %q, want prefix %q", name, i, got[i], w)
			}
		}
	}
}

// Every reader that excludes held-out tasks from TRAINING data, run through
// its real entry point against a fixture heldout.json (a held-out id
// held_core, a retired id task, its successor task_v2, and other):
// extract_lang (main), stale_fix (collect), onpolicy_pairs (main),
// probe_after_read (main), synth_episodes (main), extract_traj (main) and the
// guard in collect_onpolicy.sh. Each may admit only `other`. Network, model
// and compiler calls are stubbed in testdata/finetune_readers.py; the held-out
// filter is not, so a reader that re-reads heldout.json without
// retired_excluded admits `task` and fails here. The EVAL list (core +
// stretch, read by summarize_pilot.py, pilot*_run.sh, weekend_pilot.sh) is
// checked by TestFinetuneHeldoutMatchesBenchmarks.
func TestFinetuneExclusionReadersBehaviour(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not on PATH")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	finetune := filepath.Join(root, "tools", "finetune")
	fx := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(fx, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ids := []string{"held_core", "task", "task_v2", "other"}
	write("heldout.json", `{"core": ["held_core"], "stretch": ["task_v2"], "retired_excluded": ["task"]}`)
	write("empty/.keep", "")
	write("stub/yaml.py", "class YAMLError(Exception):\n    pass\n\ndef safe_load(*a, **k):\n    return {\"id\": \"x\"}\n")
	var lang strings.Builder
	for i, id := range ids {
		row := fmt.Sprintf(`{"id": %q, "lang": "ailang", "code": "module m%d\nexport func main() -> int = %d", "compile_ok": true, "runtime_ok": true, "stdout_ok": true}`, id, i, i)
		write(filepath.Join("repo", "eval_results", "run", id+".json"), row)
		write(filepath.Join("repo", "benchmarks", id+".yml"), "id: "+id+"\n")
		fmt.Fprintf(&lang, `{"benchmark": %q, "code": "c%d"}`+"\n", id, i)
		ws := "ailang_eval/" + id + "_pi-gemma4-12b-ft1_20260101_000000"
		write(filepath.Join("traces", "onpolicy_"+id+".jsonl"),
			fmt.Sprintf(`{"request": {"messages": [{"role": "system", "content": "Current working directory: /tmp/%s"}, {"role": "user", "content": "go"}]}, "response": {}}`+"\n", ws))
		write(filepath.Join("traces", "probe_"+id+".jsonl"),
			fmt.Sprintf(`{"request": {"messages": [{"role": "system", "content": "cwd /tmp/%s"}, {"role": "user", "content": "go"}, {"role": "assistant", "tool_calls": [{"function": {"name": "read", "arguments": "{}"}}]}, {"role": "tool", "content": "x"}]}}`+"\n", ws))
	}
	write("lang/eval.jsonl", lang.String())

	harness := filepath.Join(root, "internal", "eval_harness", "testdata", "finetune_readers.py")
	cmd := exec.Command(python, "-I", harness, finetune, fx, filepath.Join(fx, "stub"))
	cmd.Dir = fx
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("reader harness failed: %v\n%s", err, out)
	}
	got := map[string][]string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		var r struct {
			Reader   string   `json:"reader"`
			Admitted []string `json:"admitted"`
		}
		if json.Unmarshal([]byte(line), &r) == nil && r.Reader != "" {
			got[r.Reader] = r.Admitted
		}
	}
	for _, reader := range []string{"extract_lang", "stale_fix", "onpolicy_pairs", "probe_after_read", "synth_episodes", "extract_traj"} {
		admitted, ok := got[reader]
		if !ok {
			t.Errorf("%s: no result from the harness\n%s", reader, out)
			continue
		}
		if len(admitted) != 1 || admitted[0] != "other" {
			t.Errorf("%s admits %v to training; want only [other] (held_core, task and task_v2 are held out)", reader, admitted)
		}
	}

	// collect_onpolicy.sh's inline guard, run on its own (cwd = a fixture
	// repo root): it must refuse the held-out, retired and successor ids.
	src, err := os.ReadFile(filepath.Join(finetune, "collect_onpolicy.sh"))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(src), "\r\n", "\n") // a Windows checkout may be CRLF
	start := strings.Index(body, "<<'EOF'\n")
	end := strings.Index(body[start+1:], "\nEOF\n")
	if start < 0 || end < 0 {
		t.Fatal("collect_onpolicy.sh: held-out guard heredoc not found")
	}
	snippet := body[start+len("<<'EOF'\n") : start+1+end+1]
	write(filepath.Join("guardroot", "tools", "finetune", "heldout.json"), string(mustRead(t, filepath.Join(fx, "heldout.json"))))
	for _, id := range ids {
		bench := filepath.Join(fx, "bench_"+id+".txt")
		write("bench_"+id+".txt", id+"\n")
		guard := exec.Command(python, "-I", "-", bench)
		guard.Dir = filepath.Join(fx, "guardroot")
		guard.Stdin = strings.NewReader(snippet)
		out, err := guard.CombinedOutput()
		refused := err != nil && strings.Contains(string(out), "held-out benchmarks in the collection list")
		if refused != (id != "other") {
			t.Errorf("collect_onpolicy.sh guard on %q: refused=%v (err %v)\n%s", id, refused, err, out)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
