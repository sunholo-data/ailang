package effects

import (
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M1 — permanent denial tests for the audited
// counterexamples F1 (relative traversal) and F2 (in-root symlink to an
// outside file), generalised to every registered FS operation and to every
// route out of the sandbox the audit named: `..`, a symlinked file, a
// symlinked directory, and an absolute sibling-prefix path.
//
// These tests were written BEFORE the fix and FAILED on the audited baseline
// (resolveSandboxPath joined without containment; every op then called os.*
// on the joined path, which follows symlinks). They must keep failing if
// either behaviour comes back.

// containmentTree is one disposable layout:
//
//	<tmp>/
//	  marker.txt            outside file carrying the sentinel
//	  outdir/marker.txt     outside directory carrying the sentinel
//	  sandbox2/marker.txt   sibling whose name shares the sandbox's prefix
//	  sandbox/              the FS root under test
//	    inside.txt          an ordinary in-root file
//	    link.txt      -> ../marker.txt         (relative symlink, escapes)
//	    linkdir       -> ../outdir             (relative symlink, escapes)
//	    abslink.txt   -> <tmp>/marker.txt      (absolute symlink, escapes)
//	    abslinkdir    -> <tmp>/outdir          (absolute symlink to a dir, escapes)
//	    okdir/inner.txt                        an in-root subdirectory
//	    oklink.txt    -> okdir/inner.txt       (relative symlink, stays inside)
type containmentTree struct {
	tmp, sandbox, sentinel string
}

func newContainmentTree(t *testing.T) *containmentTree {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs a privilege on Windows; restricted mode is refused there (D6)")
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	// macOS TempDir lives under a symlinked /var; resolve so the absolute
	// sibling-prefix rows compare real paths.
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	c := &containmentTree{tmp: tmp, sandbox: filepath.Join(tmp, "sandbox"), sentinel: "SENTINEL-" + hex.EncodeToString(b[:])}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(c.sandbox, "okdir"), 0o755))
	must(os.MkdirAll(filepath.Join(tmp, "outdir"), 0o755))
	must(os.MkdirAll(filepath.Join(tmp, "sandbox2"), 0o755))
	must(os.WriteFile(filepath.Join(tmp, "marker.txt"), []byte(c.sentinel), 0o644))
	must(os.WriteFile(filepath.Join(tmp, "outdir", "marker.txt"), []byte(c.sentinel), 0o644))
	must(os.WriteFile(filepath.Join(tmp, "sandbox2", "marker.txt"), []byte(c.sentinel), 0o644))
	must(os.WriteFile(filepath.Join(c.sandbox, "inside.txt"), []byte("inside"), 0o644))
	must(os.WriteFile(filepath.Join(c.sandbox, "okdir", "inner.txt"), []byte("inner"), 0o644))
	must(os.Symlink("../marker.txt", filepath.Join(c.sandbox, "link.txt")))
	must(os.Symlink("../outdir", filepath.Join(c.sandbox, "linkdir")))
	must(os.Symlink(filepath.Join(tmp, "marker.txt"), filepath.Join(c.sandbox, "abslink.txt")))
	must(os.Symlink(filepath.Join(tmp, "outdir"), filepath.Join(c.sandbox, "abslinkdir")))
	must(os.Symlink("okdir/inner.txt", filepath.Join(c.sandbox, "oklink.txt")))
	return c
}

func (c *containmentTree) ctx(t *testing.T) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = c.sandbox
	ctx.Grant(NewCapability("FS"))
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })
	return ctx
}

// outsideSnapshot is every path outside the sandbox with its contents — the
// "no outside side effect" oracle. Symlink targets are not followed.
func (c *containmentTree) outsideSnapshot(t *testing.T) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(c.tmp, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == c.sandbox {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(c.tmp, p)
		line := rel
		if d.Type().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			line += "=" + string(data)
		}
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// containmentShape says how to build an op's arguments from a target path and
// how to recognise a denial in its result.
type containmentShape struct {
	// args builds the argument list; second is the rename destination.
	args func(target string) []eval.Value
	// dirTarget: the op wants a directory (listDir) rather than a file.
	dirTarget bool
	// entry: the op acts on the directory ENTRY named by the final path
	// component, not through it (remove/rename a symlink, mkdir beside one).
	// On a final-symlink target such an op is legitimately allowed; the
	// outside-tree oracle is the only assertion there.
	entry bool
	// denied reports whether (result, err) is a denial with no leaked data.
	denied func(t *testing.T, sentinel string, result eval.Value, err error) bool
}

func sv(s string) eval.Value { return &eval.StringValue{Value: s} }

func deniedByError(t *testing.T, sentinel string, result eval.Value, err error) bool {
	t.Helper()
	if err == nil {
		t.Errorf("expected an error, got result %v", result)
		return false
	}
	return true
}

func deniedByFalse(t *testing.T, sentinel string, result eval.Value, err error) bool {
	t.Helper()
	if err != nil {
		t.Errorf("probe must not error, got %v", err)
		return false
	}
	b, ok := result.(*eval.BoolValue)
	if !ok || b.Value {
		t.Errorf("probe must return false, got %v", result)
		return false
	}
	return true
}

func deniedByErr(t *testing.T, sentinel string, result eval.Value, err error) bool {
	t.Helper()
	if err != nil {
		// A Go error is also a denial (the sandbox reject path), never a leak.
		return true
	}
	tv, ok := result.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Err" {
		t.Errorf("expected Err, got %v", result)
		return false
	}
	return true
}

// fsContainmentShapes covers EVERY op registered under FS. The registry
// completeness test below fails when an op is added without a row here.
var fsContainmentShapes = map[string]containmentShape{
	"readFile":         {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, denied: deniedByError},
	"readFileBytes":    {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, denied: deniedByErr},
	"readFileResult":   {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, denied: deniedByErr},
	"writeFile":        {args: func(p string) []eval.Value { return []eval.Value{sv(p), sv("clobbered")} }, denied: deniedByError},
	"writeFileBytes":   {args: func(p string) []eval.Value { return []eval.Value{sv(p), &eval.BytesValue{Value: []byte("clobbered")}} }, denied: deniedByError},
	"writeFileResult":  {args: func(p string) []eval.Value { return []eval.Value{sv(p), sv("clobbered")} }, denied: deniedByErr},
	"appendFile":       {args: func(p string) []eval.Value { return []eval.Value{sv(p), sv("appended")} }, denied: deniedByError},
	"appendFileBytes":  {args: func(p string) []eval.Value { return []eval.Value{sv(p), &eval.BytesValue{Value: []byte("appended")}} }, denied: deniedByError},
	"appendFileResult": {args: func(p string) []eval.Value { return []eval.Value{sv(p), sv("appended")} }, denied: deniedByErr},
	"exists":           {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, denied: deniedByFalse},
	"isDir":            {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, dirTarget: true, denied: deniedByFalse},
	"isFile":           {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, denied: deniedByFalse},
	"listDir":          {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, dirTarget: true, denied: deniedByError},
	"mkdir":            {args: func(p string) []eval.Value { return []eval.Value{sv(p + ".newdir")} }, entry: true, denied: deniedByError},
	"mkdirAll":         {args: func(p string) []eval.Value { return []eval.Value{sv(p + ".newdir/deeper")} }, entry: true, denied: deniedByError},
	"mkdirResult":      {args: func(p string) []eval.Value { return []eval.Value{sv(p + ".newdir")} }, entry: true, denied: deniedByErr},
	"mkdirAllResult":   {args: func(p string) []eval.Value { return []eval.Value{sv(p + ".newdir/deeper")} }, entry: true, denied: deniedByErr},
	"removeFile":       {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, entry: true, denied: deniedByError},
	"removeFileResult": {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, entry: true, denied: deniedByErr},
	"removeDirResult":  {args: func(p string) []eval.Value { return []eval.Value{sv(p)} }, dirTarget: true, denied: deniedByErr},
	// Rename: the outside target is the SOURCE here; the destination row is
	// exercised separately in TestFSContainment_RenameDestination.
	"renameFile":       {args: func(p string) []eval.Value { return []eval.Value{sv(p), sv("stolen.txt")} }, entry: true, denied: deniedByError},
	"renameFileResult": {args: func(p string) []eval.Value { return []eval.Value{sv(p), sv("stolen.txt")} }, entry: true, denied: deniedByErr},
	// pkgAssetPath resolves into ~/.ailang/cache, which is outside every
	// sandbox by construction: under a sandbox it is Err before resolving.
	"pkgAssetPath": {args: func(p string) []eval.Value { return []eval.Value{sv("vendor/name"), sv("asset.txt")} }, denied: deniedByErr},
}

// containmentTargets are the routes out. file is a file target, dir a
// directory target, both as the PROGRAM would spell them.
func containmentTargets(c *containmentTree) []struct{ name, file, dir string } {
	return []struct{ name, file, dir string }{
		{"traversal", "../marker.txt", ".."},
		{"traversal-deep", "okdir/../../outdir/marker.txt", "okdir/../../outdir"},
		{"symlink-file", "link.txt", "link.txt"},
		{"symlink-dir", "linkdir/marker.txt", "linkdir"},
		{"absolute-symlink", "abslink.txt", "abslink.txt"},
		{"absolute-symlink-dir", "abslinkdir/marker.txt", "abslinkdir"},
		{"absolute-outside", filepath.Join(c.tmp, "marker.txt"), c.tmp},
		{"absolute-sibling-prefix", filepath.Join(c.tmp, "sandbox2", "marker.txt"), filepath.Join(c.tmp, "sandbox2")},
	}
}

// TestFSContainment_RegistryComplete: every FS op has a containment row.
func TestFSContainment_RegistryComplete(t *testing.T) {
	for op := range Registry["FS"] {
		if _, ok := fsContainmentShapes[op]; !ok {
			t.Errorf("FS op %q is registered but has no containment row in fsContainmentShapes — add one before shipping it", op)
		}
	}
	for op := range fsContainmentShapes {
		if _, ok := Registry["FS"][op]; !ok {
			t.Errorf("containment row %q names an op that is not registered", op)
		}
	}
}

// TestFSContainment_EveryOpEveryRoute is F1/F2 generalised: no op, on any
// route, reads the sentinel or changes anything outside the sandbox.
func TestFSContainment_EveryOpEveryRoute(t *testing.T) {
	ops := make([]string, 0, len(fsContainmentShapes))
	for op := range fsContainmentShapes {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	for _, op := range ops {
		shape := fsContainmentShapes[op]
		for _, tgt := range containmentTargets(&containmentTree{tmp: "/T"}) {
			t.Run(op+"/"+tgt.name, func(t *testing.T) {
				c := newContainmentTree(t)
				// Re-derive the target for THIS tree (absolute rows embed tmp).
				var target string
				for _, real := range containmentTargets(c) {
					if real.name == tgt.name {
						target = real.file
						if shape.dirTarget {
							target = real.dir
						}
					}
				}
				before := c.outsideSnapshot(t)
				ctx := c.ctx(t)
				result, err := Call(ctx, "FS", op, shape.args(target))
				finalSymlink := tgt.name == "symlink-file" || tgt.name == "absolute-symlink"
				if !(shape.entry && finalSymlink) && !shape.denied(t, c.sentinel, result, err) {
					t.Logf("CONTAINMENT FAILURE: %s(%q) was not denied", op, target)
				}
				if s, ok := result.(*eval.StringValue); ok && strings.Contains(s.Value, c.sentinel) {
					t.Errorf("CONTAINMENT FAILURE: %s(%q) returned the outside sentinel", op, target)
				}
				if tv, ok := result.(*eval.TaggedValue); ok {
					for _, f := range tv.Fields {
						if s, ok := f.(*eval.StringValue); ok && strings.Contains(s.Value, c.sentinel) {
							t.Errorf("CONTAINMENT FAILURE: %s(%q) returned the outside sentinel inside a Result", op, target)
						}
					}
				}
				if after := c.outsideSnapshot(t); after != before {
					t.Errorf("CONTAINMENT FAILURE: %s(%q) changed the outside tree:\nbefore:\n%s\nafter:\n%s", op, target, before, after)
				}
			})
		}
	}
}

// TestFSContainment_RenameDestination: the DESTINATION operand escapes.
func TestFSContainment_RenameDestination(t *testing.T) {
	for _, op := range []string{"renameFile", "renameFileResult"} {
		// Each destination goes THROUGH a directory that leaves the root. (A
		// destination naming an existing symlink entry would replace the entry,
		// which is ordinary rename semantics and touches nothing outside.)
		for _, dst := range []string{"../leak.txt", "linkdir/leak.txt", "abslinkdir/leak.txt"} {
			t.Run(op+"/"+dst, func(t *testing.T) {
				c := newContainmentTree(t)
				before := c.outsideSnapshot(t)
				ctx := c.ctx(t)
				result, err := Call(ctx, "FS", op, []eval.Value{sv("inside.txt"), sv(dst)})
				if op == "renameFile" {
					deniedByError(t, c.sentinel, result, err)
				} else {
					deniedByErr(t, c.sentinel, result, err)
				}
				if _, err := os.Lstat(filepath.Join(c.sandbox, "inside.txt")); err != nil {
					t.Errorf("inside.txt must survive a denied rename: %v", err)
				}
				if after := c.outsideSnapshot(t); after != before {
					t.Errorf("denied rename changed the outside tree:\nbefore:\n%s\nafter:\n%s", before, after)
				}
			})
		}
	}
}

// TestFSContainment_PositiveControls: what MUST keep working inside the root,
// so the denial tests cannot pass by denying everything.
func TestFSContainment_PositiveControls(t *testing.T) {
	c := newContainmentTree(t)
	ctx := c.ctx(t)
	read := func(p string) string {
		t.Helper()
		v, err := Call(ctx, "FS", "readFile", []eval.Value{sv(p)})
		if err != nil {
			t.Fatalf("readFile(%q): %v", p, err)
		}
		return v.(*eval.StringValue).Value
	}
	if got := read("inside.txt"); got != "inside" {
		t.Errorf("relative read: %q", got)
	}
	if got := read(filepath.Join(c.sandbox, "inside.txt")); got != "inside" {
		t.Errorf("absolute in-root read: %q", got)
	}
	if got := read("oklink.txt"); got != "inner" {
		t.Errorf("relative in-root symlink must still resolve: %q", got)
	}
	if got := read("okdir/../inside.txt"); got != "inside" {
		t.Errorf("dot-dot that stays inside must work: %q", got)
	}
	if got := read("./okdir/inner.txt"); got != "inner" {
		t.Errorf("dot-slash: %q", got)
	}
	if _, err := Call(ctx, "FS", "writeFile", []eval.Value{sv("okdir/new.txt"), sv("new")}); err != nil {
		t.Fatalf("in-root write: %v", err)
	}
	if got := read("okdir/new.txt"); got != "new" {
		t.Errorf("write then read: %q", got)
	}
	if _, err := Call(ctx, "FS", "mkdirAll", []eval.Value{sv("a/b/c")}); err != nil {
		t.Fatalf("mkdirAll: %v", err)
	}
	if _, err := Call(ctx, "FS", "renameFile", []eval.Value{sv("okdir/new.txt"), sv("a/b/c/moved.txt")}); err != nil {
		t.Fatalf("in-root rename: %v", err)
	}
	if got := read("a/b/c/moved.txt"); got != "new" {
		t.Errorf("rename then read: %q", got)
	}
	v, err := Call(ctx, "FS", "listDir", []eval.Value{sv(".")})
	if err != nil {
		t.Fatalf("listDir(.): %v", err)
	}
	names := make([]string, 0)
	for _, e := range v.(*eval.ListValue).Elements {
		names = append(names, e.(*eval.StringValue).Value)
	}
	if !sort.StringsAreSorted(names) || !strings.Contains(strings.Join(names, ","), "inside.txt") {
		t.Errorf("listDir must be sorted and contain inside.txt: %v", names)
	}
	// Removing a symlink ENTRY is an in-root operation on the entry, not on
	// its target: the outside marker survives.
	if _, err := Call(ctx, "FS", "removeFile", []eval.Value{sv("link.txt")}); err != nil {
		t.Fatalf("removing the link entry itself must be allowed: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(c.tmp, "marker.txt")); err != nil || string(data) != c.sentinel {
		t.Errorf("removing link.txt must not touch its target: %v %q", err, data)
	}
	// removeDirResult on a symlink to a directory: not a directory entry → Err,
	// and the target directory survives.
	r, err := Call(ctx, "FS", "removeDirResult", []eval.Value{sv("linkdir")})
	if err != nil || r.(*eval.TaggedValue).CtorName != "Err" {
		t.Errorf("removeDirResult(symlink-to-dir) must be Err, got %v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(c.tmp, "outdir")); err != nil {
		t.Errorf("outdir must survive: %v", err)
	}
}

// TestFSContainment_SymlinkSwapRace: a concurrent writer flips `swap` between
// an inside file and the outside marker while readers hammer it. The
// sentinel must never come back — the root handle validates every component
// at open time, so there is no check/use window a lexical check would leave.
func TestFSContainment_SymlinkSwapRace(t *testing.T) {
	c := newContainmentTree(t)
	ctx := c.ctx(t)
	swap := filepath.Join(c.sandbox, "swap")
	inside := "inside.txt"
	outside := "../marker.txt"
	if err := os.Symlink(inside, swap); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			target := inside
			if i%2 == 1 {
				target = outside
			}
			tmp := swap + ".tmp"
			_ = os.Symlink(target, tmp)
			_ = os.Rename(tmp, swap)
			i++
		}
	}()
	for i := 0; i < 2000; i++ {
		v, err := Call(ctx, "FS", "readFile", []eval.Value{sv("swap")})
		if err != nil {
			continue // denied or mid-rename: both fine
		}
		if strings.Contains(v.(*eval.StringValue).Value, c.sentinel) {
			close(stop)
			wg.Wait()
			t.Fatalf("CONTAINMENT FAILURE: iteration %d read the outside sentinel through a swapped symlink", i)
		}
	}
	close(stop)
	wg.Wait()
}

// TestFSContainment_RootLifecycle: the root is opened once per sandbox,
// shared by derived contexts, closed by the owner, and Close is idempotent.
func TestFSContainment_RootLifecycle(t *testing.T) {
	c := newContainmentTree(t)
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = c.sandbox
	ctx.Grant(NewCapability("FS"))

	fdsBefore := openFDCount(t)
	if _, err := Call(ctx, "FS", "readFile", []eval.Value{sv("inside.txt")}); err != nil {
		t.Fatal(err)
	}
	child := ctx.WithBudget(nil)
	if _, err := Call(child, "FS", "readFile", []eval.Value{sv("inside.txt")}); err != nil {
		t.Fatal(err)
	}
	if ctx.fsRoot != child.fsRoot {
		t.Fatal("WithBudget must share the root holder, not copy it")
	}
	clone := ctx.Clone().(*EffContext)
	if clone.fsRoot != ctx.fsRoot {
		t.Fatal("Clone must share the root holder")
	}
	if err := ctx.CloseFSRoot(); err != nil {
		t.Fatal(err)
	}
	if err := ctx.CloseFSRoot(); err != nil {
		t.Fatalf("second close must be a no-op: %v", err)
	}
	if fdsAfter := openFDCount(t); fdsAfter > fdsBefore {
		t.Errorf("descriptor leak: %d open before, %d after CloseFSRoot", fdsBefore, fdsAfter)
	}
	// After close, the next operation reopens (the sandbox path is unchanged)
	// rather than failing — the holder is a cache with an owner, not a
	// one-shot.
	if _, err := Call(ctx, "FS", "readFile", []eval.Value{sv("inside.txt")}); err != nil {
		t.Fatalf("read after close must reopen: %v", err)
	}
	_ = ctx.CloseFSRoot()
}

// openFDCount counts this process's open descriptors via /dev/fd (darwin,
// linux). Returns 0 where that is unavailable, which makes the leak check
// vacuous there rather than failing.
func openFDCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/dev/fd")
	if err != nil {
		return 0
	}
	return len(entries)
}
