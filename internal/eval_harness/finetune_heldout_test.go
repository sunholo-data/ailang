package eval_harness

import (
	"encoding/json"
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

// Every training-exclusion reader, run for real against a fixture heldout.json
// (review 2 of #2229: a text grep passed with the key removed from the code).
// heldout.load_heldout is the one function; extract_lang re-exports it to
// onpolicy_pairs, stale_fix, probe_after_read and synth_episodes; extract_traj
// imports it; collect_onpolicy.sh's guard snippet calls it. No collection,
// training or eval script is run — only the readers and the guard snippet.
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
	tmp := t.TempDir()
	fixture := filepath.Join(tmp, "tools", "finetune", "heldout.json")
	if err := os.MkdirAll(filepath.Dir(fixture), 0o755); err != nil {
		t.Fatal(err)
	}
	const fixtureJSON = `{"core": ["held_core"], "stretch": ["task_v2"], "retired_excluded": ["task"]}`
	if err := os.WriteFile(fixture, []byte(fixtureJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	// extract_lang imports PyYAML; the readers never call it, so a stub keeps
	// the test independent of the environment.
	stub := filepath.Join(tmp, "stub")
	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stub, "yaml.py"), []byte("def safe_load(*a, **k):\n    return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	script := `import sys
sys.path[:0] = [sys.argv[1], sys.argv[2]]
import heldout, extract_lang, extract_traj
fx = sys.argv[3]
assert extract_lang.load_heldout is heldout.load_heldout, "extract_lang has its own load_heldout"
assert extract_traj.load_heldout is heldout.load_heldout, "extract_traj has its own load_heldout"
print(",".join(sorted(heldout.load_heldout(fx))))
print(",".join(heldout.eval_ids(fx)))
`
	cmd := exec.Command(python, "-I", "-c", script, stub, finetune, fixture)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("readers failed: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || lines[0] != "held_core,task,task_v2" || lines[1] != "held_core,task_v2" {
		t.Fatalf("training exclusion / eval list:\n%s\nwant held_core,task,task_v2 / held_core,task_v2", out)
	}

	// collect_onpolicy.sh's guard snippet, run on its own (cwd = a fixture
	// repo root): it must refuse the retired id and its successor alike.
	src, err := os.ReadFile(filepath.Join(finetune, "collect_onpolicy.sh"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	start := strings.Index(body, "<<'EOF'\n")
	end := strings.Index(body[start+1:], "\nEOF\n")
	if start < 0 || end < 0 {
		t.Fatal("collect_onpolicy.sh: held-out guard heredoc not found")
	}
	snippet := body[start+len("<<'EOF'\n") : start+1+end+1]
	if err := os.WriteFile(filepath.Join(tmp, "tools", "finetune", "heldout.py"), mustRead(t, filepath.Join(finetune, "heldout.py")), 0o644); err != nil {
		t.Fatal(err)
	}
	for list, wantRefused := range map[string]bool{"task\nother\n": true, "task_v2\n": true, "held_core\n": true, "other\nmore\n": false} {
		bench := filepath.Join(tmp, "bench.txt")
		if err := os.WriteFile(bench, []byte(list), 0o644); err != nil {
			t.Fatal(err)
		}
		guard := exec.Command(python, "-I", "-", bench)
		guard.Dir = tmp
		guard.Stdin = strings.NewReader(snippet)
		out, err := guard.CombinedOutput()
		refused := err != nil && strings.Contains(string(out), "held-out benchmarks in the collection list")
		if refused != wantRefused {
			t.Errorf("collection list %q: refused=%v (err %v), want %v\n%s", list, refused, err, wantRefused, out)
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
