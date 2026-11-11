package eval_harness

import (
	"encoding/json"
	"os"
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

// Every script that reads heldout.json to EXCLUDE training data must include
// retired_excluded; only the eval runners (which pass the list to
// --benchmarks) read core + stretch alone.
func TestFinetuneExclusionReadersIncludeRetired(t *testing.T) {
	dir := filepath.Join("..", "..", "tools", "finetune")
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	readers := 0
	for _, f := range files {
		if !strings.HasSuffix(f, ".py") && !strings.HasSuffix(f, ".sh") {
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		if !strings.Contains(src, "heldout.json") || !strings.Contains(src, `"stretch"]`) && !strings.Contains(src, `'stretch']`) {
			continue
		}
		readers++
		evalRunner := strings.Contains(src, "--benchmarks") && strings.Contains(src, "HELD=")
		if !evalRunner && !strings.Contains(src, "retired_excluded") {
			t.Errorf("%s reads the held-out set for exclusion but ignores retired_excluded", filepath.Base(f))
		}
	}
	if readers < 3 {
		t.Fatalf("found only %d heldout.json readers; the instrument is not seeing the scripts", readers)
	}
}
