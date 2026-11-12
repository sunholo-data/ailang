package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/traverse"
)

// M-V1-TYVAR-HYGIENE (#2157, #2076). Design:
// design_docs/planned/v1_0_0/m-v1-tyvar-hygiene-and-digest-determinism.md
//
// Two defects with one missing invariant: the raw type-variable names of one
// inference context crossed the module boundary as identity.
//   - #2157: an exported scheme left variables free (the generalisation
//     collector skipped open records), so an importer's own fresh αN collided
//     with them and a valid program was rejected depending on unrelated code.
//   - #2076: fresh variables were allocated in map order and the digest hashed
//     the raw names, so identical cold runs produced different digests.

// The issue's program, verbatim. It was rejected 10/10 before the fix with
// "failed to unify field 'ver': cannot unify record with unexpandable type
// constructor int"; an unrelated declaration made it pass.
const repro2157 = `module repro_is_newer_unify
import std/sem (sem_frame, make_frame_at, with_embedding, has_embedding, is_newer, bump_version)
import std/embedding (encodeF32LE)
import std/bytes (fromString)
%s
pure func describe(label: string, f: sem_frame) -> string = "${label}: id=${f.id} ver=${show(f.ver)} embedding=${show(has_embedding(f))} dim=${show(f.embedding_dim)}"

pure func pickNewer(a: sem_frame, b: sem_frame) -> sem_frame = if is_newer(b, a) then b else a

export func main() -> () ! {IO} {
  let original = make_frame_at("faq-42", "How", fromString(""), 1700000000)
  let replicaA = with_embedding(original, encodeF32LE([0.12, 0.85, 0.03, 0.4]), 4)
  let edited = make_frame_at("faq-42", "How can", fromString(""), 1700000600)
  let replicaB = bump_version(edited)
  println("B newer than A: ${show(is_newer(replicaB, replicaA))}")
}
`

// writeAndCheck writes files into a fresh directory and type-checks entry with
// the disk cache off.
func writeAndCheck(t *testing.T, files map[string]string, entry string) (*Result, error) {
	t.Helper()
	t.Setenv("AILANG_NO_CACHE", "1")
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	res, err := Run(Config{Mode: ModeCheck}, Source{Filename: entry})
	return &res, err
}

func padDecls(k int) string {
	var b strings.Builder
	for i := 0; i < k; i++ {
		fmt.Fprintf(&b, "\npure func pad%d(n: int) -> int = n + %d\n", i, i)
	}
	return b.String()
}

// T4 (#2157): the issue repro must check, and the verdict must not depend on how
// many unrelated declarations precede it (the issue's "add one unrelated func
// and it passes" flip is exactly this property).
func TestTyvarHygiene_2157_VerdictIndependentOfUnrelatedCode(t *testing.T) {
	for _, k := range []int{0, 1, 2, 7, 50} {
		t.Run(fmt.Sprintf("pad=%d", k), func(t *testing.T) {
			src := fmt.Sprintf(repro2157, padDecls(k))
			if _, err := writeAndCheck(t, map[string]string{"repro_is_newer_unify.ail": src}, "repro_is_newer_unify.ail"); err != nil {
				t.Fatalf("valid program rejected with %d unrelated declarations: %v", k, err)
			}
		})
	}
}

// T4 (use2): an imported function over an open record is polymorphic in the
// field type — two legal uses at different field types must both check. Before
// the fix the field variable was exported free, so the uses shared it
// ("No instance for Num[string]").
func TestTyvarHygiene_ImportedOpenRecordIsPolymorphic(t *testing.T) {
	files := map[string]string{
		"lib.ail": `module lib

export pure func newer(a: t, b: t) -> bool = a.ver > b.ver
`,
		"use2.ail": `module use2
import lib (newer)

export func main() -> () ! {IO} {
  let p = {id: "x", ver: 1}
  let q = {id: "y", ver: 2}
  let s = {name: "z", ver: "v1"}
  let t = {name: "w", ver: "v2"}
  println(show(newer(p, q)));
  println(show(newer(s, t)))
}
`,
	}
	if _, err := writeAndCheck(t, files, "use2.ail"); err != nil {
		t.Fatalf("two legal uses of an imported open-record function rejected: %v", err)
	}
}

// Two fresh-variable supplies must not share a spelling. Open record patterns
// mint inference-context record rows `ρN`; the row unifier minted its own `ρN`
// (effect rows, separate counter), and the two met in one substitution as one
// variable. Rejected before the fix: "kind mismatch: row variable ρ7 has kind
// Row Effect, but {manager: Option[string], ...ρ_empty_1} has kind Row Record".
func TestTyvarHygiene_FreshRowSuppliesDoNotCollide(t *testing.T) {
	src := `module coll
import std/option (flatMap)
import std/list as L

pure func findPerson(dir, wanted: string) = L.head(L.filter(\p. match p { {id, ...} => id == wanted }, dir))

export pure func managerEmail(dir, wanted: string) {
  let person = findPerson(dir, wanted)
  let manager = flatMap(\p. match p { {manager, ...} => flatMap(\mid. findPerson(dir, mid), manager) }, person)
  flatMap(\m. match m { {email, ...} => email }, manager)
}
`
	if _, err := writeAndCheck(t, map[string]string{"coll.ail": src}, "coll.ail"); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// The iface builder recomputes the quantifiers of an export with a declared
// effect-row variable (`! {e}`) after label restoration. That recompute must
// keep the scheme's quantified record rows, or the field-access row of
// `r.x` is exported free.
func TestTyvarHygiene_RowEffectExportKeepsRecordRowQuantified(t *testing.T) {
	src := `module rowe

export func applyX(r: t, f: (int) -> int ! {e}) -> int ! {e} = f(r.x)
`
	res, err := writeAndCheck(t, map[string]string{"rowe.ail": src}, "rowe.ail")
	if err != nil {
		t.Fatal(err)
	}
	item := res.Interface.Exports["applyX"]
	if item == nil {
		t.Fatal("applyX not exported")
	}
	if free := unclosedVars(item.Type); len(free) > 0 {
		t.Fatalf("applyX exported with free variables %v: %s", free, item.Type)
	}
}

// schemeVarNames returns every variable name occurring in a scheme's type.
func schemeVarNames(s *types.Scheme) map[string]bool {
	return traverse.CollectFreeVars(unlabel(s.Type))
}

// unlabel strips TLabelled wrappers, which traverse does not descend into.
func unlabel(t types.Type) types.Type {
	switch typ := t.(type) {
	case *types.TLabelled:
		return unlabel(typ.Inner)
	case *types.TFunc2:
		cp := *typ
		cp.Params = make([]types.Type, len(typ.Params))
		for i, p := range typ.Params {
			cp.Params[i] = unlabel(p)
		}
		cp.Return = unlabel(typ.Return)
		return &cp
	}
	return t
}

// unclosedVars lists the variables of an exported scheme that are not quantified.
func unclosedVars(s *types.Scheme) []string {
	bound := map[string]bool{}
	for _, v := range s.TypeVars {
		bound[v] = true
	}
	for _, v := range s.RowVars {
		bound[v] = true
	}
	var free []string
	for v := range schemeVarNames(s) {
		if !bound[v] {
			free = append(free, v)
		}
	}
	sort.Strings(free)
	return free
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository go.mod not found")
		}
		root = parent
	}
}

func stdModules(t *testing.T, root string) []string {
	t.Helper()
	var mods []string
	err := filepath.WalkDir(filepath.Join(root, "std"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".ail") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			mods = append(mods, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(mods)
	return mods
}

// T5a: every exported scheme of every std module is closed, and its stored
// names are canonical (no raw αN / ρN / literal "r" crosses the boundary).
// Before the fix 10 std/sem exports had free variables.
func TestTyvarHygiene_StdExportSchemesAreClosed(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	root := repoRoot(t)
	t.Chdir(root)
	var unclosed, raw []string
	exports := 0
	for _, rel := range stdModules(t, root) {
		res, err := Run(Config{Mode: ModeCheck}, Source{Filename: rel})
		if err != nil {
			t.Fatalf("compile %s: %v", rel, err)
		}
		for name, item := range res.Interface.Exports {
			if item.Type == nil {
				continue
			}
			exports++
			key := res.Interface.Module + "." + name
			if free := unclosedVars(item.Type); len(free) > 0 {
				unclosed = append(unclosed, key+" free="+strings.Join(free, ","))
			}
			for v := range schemeVarNames(item.Type) {
				if !strings.HasPrefix(v, "'") {
					raw = append(raw, key+" var="+v)
				}
			}
		}
	}
	sort.Strings(unclosed)
	sort.Strings(raw)
	if exports < 400 {
		t.Fatalf("census saw only %d std exports; the walk is broken", exports)
	}
	if len(unclosed) > 0 {
		t.Errorf("%d std exports have free type/row variables:\n  %s", len(unclosed), strings.Join(unclosed, "\n  "))
	}
	if len(raw) > 0 {
		t.Errorf("%d std exports store non-canonical variable names:\n  %s", len(raw), strings.Join(raw[:min(len(raw), 20)], "\n  "))
	}
}

// T4 hygiene property: instantiating an imported scheme twice gives disjoint
// variable sets that share no name with the stored scheme.
func TestTyvarHygiene_ImportedInstantiationIsFresh(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	root := repoRoot(t)
	t.Chdir(root)
	res, err := Run(Config{Mode: ModeCheck}, Source{Filename: "std/sem.ail"})
	if err != nil {
		t.Fatal(err)
	}
	item := res.Interface.Exports["is_newer"]
	if item == nil {
		t.Fatal("std/sem.is_newer not exported")
	}
	stored := schemeVarNames(item.Type)
	if len(item.Type.TypeVars) == 0 {
		t.Fatalf("is_newer exported with no quantified type variables: %v", item.Type)
	}
	counter := 0
	fresh := func(k types.Kind) types.Type {
		counter++
		if k.Equals(types.Star) {
			return &types.TVar2{Name: fmt.Sprintf("α%d", counter), Kind: k}
		}
		return &types.RowVar{Name: fmt.Sprintf("ρ%d", counter), Kind: k}
	}
	first := traverse.CollectFreeVars(unlabel(item.Type.Instantiate(fresh)))
	second := traverse.CollectFreeVars(unlabel(item.Type.Instantiate(fresh)))
	for v := range first {
		if stored[v] {
			t.Errorf("instantiation kept stored variable %q", v)
		}
		if second[v] {
			t.Errorf("two instantiations share variable %q", v)
		}
	}
}

const ndFixture = `module nd
import std/option (Option, Some, None)

export pure func mk(n: int) -> rec_t = {p: None, q: None, s: None, n: n}

export pure func mkAgain(n: int) -> rec_t = mk(n)
`

// T1: building the same interface repeatedly in one process gives one digest and
// one serialised iface.json. Go randomises every map range, so one process is
// enough to expose map-order naming (before the fix: up to 6 digests).
func TestIfaceDeterminism_InProcess(t *testing.T) {
	cases := []struct {
		name, entry string
		files       map[string]string
		std         bool
	}{
		{name: "nd", entry: "nd.ail", files: map[string]string{"nd.ail": ndFixture}},
		{name: "std/sem", entry: "std/sem.ail", std: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AILANG_NO_CACHE", "1")
			if tc.std {
				t.Chdir(repoRoot(t))
			} else {
				dir := t.TempDir()
				for rel, content := range tc.files {
					if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				t.Chdir(dir)
			}
			digests := map[string]int{}
			var firstJSON []byte
			jsonVariants := 0
			builds := 32
			if tc.std {
				builds = 16 // ~0.3 s each; the unfixed rate was 9 digests in 32
			}
			for i := 0; i < builds; i++ {
				res, err := Run(Config{Mode: ModeCheck}, Source{Filename: tc.entry})
				if err != nil {
					t.Fatal(err)
				}
				digests[res.Interface.Digest]++
				data, err := marshalIfaceFull(res.Interface)
				if err != nil {
					t.Fatal(err)
				}
				if firstJSON == nil {
					firstJSON = data
				} else if !bytes.Equal(firstJSON, data) {
					jsonVariants++
				}
			}
			if len(digests) != 1 || jsonVariants != 0 {
				t.Fatalf("%s: %d distinct digests and %d differing iface.json serialisations in %d builds: %v",
					tc.name, len(digests), jsonVariants, builds, digests)
			}
		})
	}
}

// coreTIFingerprint renders a module's Core type info (node id → type) in node
// order: the raw, pre-canonicalisation names the checker allocated.
func coreTIFingerprint(cti types.CoreTypeInfo) string {
	ids := make([]uint64, 0, len(cti))
	for id := range cti {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var b strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&b, "%d:%s\n", id, cti[id])
	}
	return b.String()
}

// B3: below the module boundary, the checker allocates fresh variables in a
// fixed order, so the names inside one module (error messages, Core type info)
// are deterministic too — not only the canonicalised interface. The nd fixture's
// record literal is the site: its fields were inferred in map order.
func TestTyvarHygiene_InModuleFreshNamesAreDeterministic(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nd.ail"), []byte(ndFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	seen := map[string]int{}
	for i := 0; i < 32; i++ {
		res, err := Run(Config{Mode: ModeCheck}, Source{Filename: "nd.ail"})
		if err != nil {
			t.Fatal(err)
		}
		seen[coreTIFingerprint(res.Artifacts.CoreTI)]++
	}
	if len(seen) != 1 {
		t.Fatalf("Core type info of one module differs across 32 builds (%d variants): fresh names depend on map order", len(seen))
	}
}

// T3: adding a private declaration does not change the digest of an otherwise
// unchanged interface (before the fix `mk`'s variable moved α1 → α3).
func TestIfaceDeterminism_PrivateDeclarationDoesNotMoveDigest(t *testing.T) {
	const tmpl = `module offm
import std/option (Option, Some, None)
%s
export pure func mk(n: int) -> rec_t = {p: None, n: n}
`
	digest := func(extra string) string {
		res, err := writeAndCheck(t, map[string]string{"offm.ail": fmt.Sprintf(tmpl, extra)}, "offm.ail")
		if err != nil {
			t.Fatal(err)
		}
		return res.Interface.Digest
	}
	plain := digest("")
	padded := digest("\npure func helper(n: int) -> int = n + 1\n")
	if plain != padded {
		t.Fatalf("a private declaration changed the interface digest: %s vs %s", plain, padded)
	}
}

// ---- T2: cross-process determinism with an isolated cold cache ----

const ifaceDetHelperEnv = "AILANG_TEST_IFACE_DET_HELPER"

// TestIfaceDeterminism_Helper is not a test: the parent re-executes the test
// binary with ifaceDetHelperEnv set, and this compiles the listed modules into
// the AILANG_CACHE_DIR the parent chose.
func TestIfaceDeterminism_Helper(t *testing.T) {
	spec := os.Getenv(ifaceDetHelperEnv)
	if spec == "" {
		t.Skip("subprocess helper for TestIfaceDeterminism_FreshProcesses")
	}
	dir, list, _ := strings.Cut(spec, "|")
	t.Chdir(dir)
	for _, rel := range strings.Split(list, ",") {
		if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: rel}); err != nil {
			t.Fatalf("compile %s: %v", rel, err)
		}
	}
}

// collectIfaceJSON reads every compile/modules/*/iface.json under a cache dir.
func collectIfaceJSON(t *testing.T, cacheDir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	matches, err := filepath.Glob(filepath.Join(cacheDir, "compile", "modules", "*", artifactIfaceName))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(filepath.Dir(m))] = data
	}
	return out
}

// T2: the same check run in N fresh processes, each with its own cold
// AILANG_CACHE_DIR, writes byte-identical iface.json for every module — the
// fixture, std/sem, and every std module they import. Before the fix std/sem
// produced several distinct files across runs (#2076).
func TestIfaceDeterminism_FreshProcesses(t *testing.T) {
	root := repoRoot(t)
	fixtureDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixtureDir, "nd.ail"), []byte(ndFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	const runs = 6
	type job struct{ dir, list string }
	jobs := []job{{fixtureDir, "nd.ail"}, {root, "std/sem.ail"}}
	variants := map[string]map[string]bool{} // module -> distinct contents
	for run := 0; run < runs; run++ {
		cacheDir := t.TempDir()
		for _, j := range jobs {
			cmd := exec.Command(os.Args[0], "-test.run=^TestIfaceDeterminism_Helper$", "-test.count=1")
			cmd.Env = append(os.Environ(),
				ifaceDetHelperEnv+"="+j.dir+"|"+j.list,
				"AILANG_CACHE_DIR="+cacheDir,
				"AILANG_NO_CACHE=",
			)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("helper process failed: %v\n%s", err, out)
			}
		}
		files := collectIfaceJSON(t, cacheDir)
		if len(files) < 2 {
			t.Fatalf("run %d wrote only %d iface.json files; the cache is not being used", run, len(files))
		}
		for mod, data := range files {
			if variants[mod] == nil {
				variants[mod] = map[string]bool{}
			}
			variants[mod][string(data)] = true
		}
	}
	var bad []string
	for mod, v := range variants {
		if len(v) > 1 {
			bad = append(bad, fmt.Sprintf("%s: %d distinct iface.json in %d runs", mod, len(v), runs))
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("iface.json differs across fresh processes:\n  %s", strings.Join(bad, "\n  "))
	}
}
