package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// fakeAilang is a stand-in compiler with the `check --json` contract. Both arms
// fail files containing BROKEN (and the canary's "not an int"); the candidate arm
// additionally rejects PLANT (as #2184 rejected `pure … ! {IO}`), times out on
// SLOW, and writes into its cwd on WRITE_CWD — each a control the gate must see.
const fakeAilang = `#!/bin/bash
ARM=__ARM__
if [ "$1" = "--version" ]; then echo "fake-ailang v0.58.1 ($ARM)"; exit 0; fi
env > "$AILANG_STATE_DIR/env.txt"
[ "$1" = "check" ] || { echo "usage" >&2; exit 64; }
shift
pass() { printf '{"file":"%s","passed":true,"error_count":0,"errors":[]}\n' "$1"; exit 0; }
failj() { printf '{"file":"%s","passed":false,"error_count":1,"errors":[{"code":"ERROR","message":"%s"}]}\n' "$1" "$2"; exit 1; }
if [ "$1" = "--package" ]; then
	if grep -rq BROKEN . ; then
		# The candidate adds a warning to a package that fails in both arms: warnings
		# never fail a check, so this must stay fail_same, not fail_changed.
		w=""; [ "$ARM" = cand ] && w=',{"code":"WARNING","message":"unused import std/io"}'
		printf '{"file":".","passed":false,"error_count":1,"errors":[{"code":"ERROR","message":"type error in pkg (decl 0): cannot unify int vs string"}%s]}\n' "$w"
		exit 1
	fi
	pass .
fi
f="$2"
content=$(cat "$f") || { echo "cannot read $f" >&2; exit 2; }
case "$content" in *BROKEN*|*"not an int"*) failj "$f" "type error in m (decl 0): type unification failed: cannot unify string vs int";; esac
case "$content" in *DEPMISS*) failj "$f" "failed to resolve package import sunholo/missing: not in the package root";; esac
case "$content" in *MSGDRIFT*)
	if [ "$ARM" = cand ]; then failj "$f" "type error in m (decl 0): cannot unify string vs int at m.ail:9:2"; fi
	failj "$f" "type error in m (decl 0): cannot unify string vs int at m.ail:3:1";;
esac
case "$content" in *ABSPATH*) failj "$f" "type error in m (decl 0): cannot unify string vs int in $AILANG_CACHE_DIR/m.cache";; esac
case "$content" in *BASESLOW*)
	if [ "$ARM" = cand ]; then failj "$f" "type error in m (decl 0): cannot unify string vs int"; fi
	sleep 20;;
esac
if [ "$ARM" = cand ]; then
	case "$content" in *WRITE_PKGROOT*) echo x > "$AILANG_PACKAGE_ROOT/leaked.txt";; esac
	case "$content" in *PLANT*) failj "$f" "effect checking failed in a/b: Effect checking failed for function 'f'\\n  The pure keyword contradicts the declared effect row ! {IO}.";; esac
	case "$content" in *SLOW*) sleep 20;; esac
	case "$content" in *WRITE_CWD*) echo x > ./leaked.txt;; esac
fi
pass "$f"
`

func needUnix(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake compiler is a bash script")
	}
}

func writeFake(t *testing.T, dir, arm string) string {
	t.Helper()
	p := filepath.Join(dir, "fake-"+arm, "ailang")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(fakeAilang, "__ARM__", arm)), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// stubRemote serves in-memory repos as GitHub-style tarballs (one top-level dir).
type stubRemote struct {
	repos map[string]map[string]string // repo -> path -> content
	fail  map[string]bool
	calls []string
}

func (s *stubRemote) ResolveRef(repo, ref string) (string, error) {
	s.calls = append(s.calls, "resolve "+repo+"@"+ref)
	if s.fail[repo] {
		return "", fmt.Errorf("HTTP 404: Not Found (repos/%s)", repo)
	}
	return strings.Repeat("a", 40), nil
}

func (s *stubRemote) Tarball(repo, sha string, w io.Writer) error {
	s.calls = append(s.calls, "tarball "+repo)
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	files := s.repos[repo]
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		body := files[n]
		if err := tw.WriteHeader(&tar.Header{Name: "owner-repo-" + sha[:7] + "/" + n, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// fixture is a scratch run: manifest, expected-breaks file, fake binaries.
type fixture struct {
	t                *testing.T
	dir              string
	base, cand       string
	remote           *stubRemote
	manifest, expect string
}

func newFixture(t *testing.T, manifest, expected string) *fixture {
	t.Helper()
	needUnix(t)
	dir := t.TempDir()
	f := &fixture{t: t, dir: dir, base: writeFake(t, dir, "base"), cand: writeFake(t, dir, "cand"),
		remote:   &stubRemote{repos: map[string]map[string]string{}, fail: map[string]bool{}},
		manifest: filepath.Join(dir, "corpora.yaml"), expect: filepath.Join(dir, "expected-breaks.yaml")}
	for p, body := range map[string]string{f.manifest: manifest, f.expect: expected} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "pkgroot"), 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) options(base, cand string) Options {
	return Options{Candidate: cand, Baseline: base, Tier: "release", Ref: refDefault,
		Manifest: f.manifest, Expected: f.expect, Out: filepath.Join(f.dir, "out"), State: filepath.Join(f.dir, "state"),
		PackageRoot: filepath.Join(f.dir, "pkgroot"), SelfRoot: f.dir, Jobs: 3, Timeout: 30, NoNice: true}
}

func (f *fixture) run(o Options) (int, *Report) {
	f.t.Helper()
	var log bytes.Buffer
	code, err := Run(o, f.remote, &log)
	if err != nil {
		return code, nil
	}
	data, err := os.ReadFile(filepath.Join(o.Out, "report.json"))
	if err != nil {
		f.t.Fatalf("no report.json (exit %d): %v; log: %s", code, err, log.String())
	}
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		f.t.Fatal(err)
	}
	return code, &rep
}

func stateCounts(rep *Report) map[string]int {
	out := map[string]int{}
	for _, c := range rep.Corpora {
		for k, v := range c.States {
			out[k] += v
		}
	}
	return out
}

func rowByRel(rep *Report, rel string) *Row {
	for i := range rep.Rows {
		if rep.Rows[i].Rel == rel {
			return &rep.Rows[i]
		}
	}
	return nil
}
