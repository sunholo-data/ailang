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
if [ "$1" = "lock" ]; then
	# Enough of 'ailang lock' for provisioning: path deps must exist, registry deps
	# must be in AILANG_PACKAGE_ROOT, an unreadable lock is refused, never repaired.
	env > "$AILANG_STATE_DIR/lock-env.txt"
	if [ -f ailang.lock ] && ! grep -q 'ailang.lock/v1' ailang.lock; then echo "Error: cannot update existing ailang.lock" >&2; exit 1; fi
	if grep -q HANGLOCK ailang.toml; then sleep 30; fi
	if [ "$ARM" = cand ] && grep -q CANDLOCKFAIL ailang.toml; then echo "Error: dependency resolution failed: the candidate's resolver broke" >&2; exit 1; fi
	entries=""
	if [ "$ARM" = cand ] && grep -q CANDLOCKDIFF ailang.toml; then entries='{"name":"acme/extra","version":"9.0.0","source":"registry"}'; fi
	while IFS= read -r line; do
		case "$line" in \"*=*) ;; *) continue;; esac
		name=${line#\"}; name=${name%%\"*}
		case "$line" in
		*path*)
			p=$(printf '%s\n' "$line" | sed 's/.*path *= *"\([^"]*\)".*/\1/')
			[ -f "$p/ailang.toml" ] || { echo "Error: dependency resolution failed: path dependency $p not found" >&2; exit 1; }
			e="{\"name\":\"$name\",\"version\":\"0.1.0\",\"source\":\"path\",\"path\":\"$p\"}";;
		*)
			v=$(printf '%s\n' "$line" | sed 's/.*= *"\([^"]*\)".*/\1/')
			[ -d "$AILANG_PACKAGE_ROOT/$name/$v" ] || { echo "Error: dependency resolution failed: registry package $name@$v is not provisioned in AILANG_PACKAGE_ROOT" >&2; exit 1; }
			e="{\"name\":\"$name\",\"version\":\"$v\",\"source\":\"registry\"}";;
		esac
		entries="${entries:+$entries,}$e"
	done < ailang.toml
	printf '{"schema":"ailang.lock/v1","packages":[%s]}\n' "$entries" > ailang.lock
	echo "Generated ailang.lock"
	exit 0
fi
if [ "$1" = "install" ]; then
	# 'ailang install --no-bin name@version' from DOWNSTREAM_FAKE_REGISTRY (a dir);
	# unset = the network is unreachable.
	[ "$2" = "--no-bin" ] || { echo "Error: want --no-bin" >&2; exit 64; }
	name=${3%@*}; ver=${3##*@}
	echo "Fetching $3 from ${DOWNSTREAM_FAKE_REGISTRY:-https://registry.invalid}..."
	[ -n "$DOWNSTREAM_FAKE_REGISTRY" ] || { echo "Error: failed to fetch package metadata: dial tcp: connection refused" >&2; exit 1; }
	[ -d "$DOWNSTREAM_FAKE_REGISTRY/$name/$ver" ] || { echo "Error: failed to fetch package metadata: registry returned HTTP 404 for metadata of $3" >&2; exit 1; }
	[ -f "$DOWNSTREAM_FAKE_REGISTRY/$name/$ver/FAIL503" ] && { echo "Error: failed to fetch package metadata: registry returned HTTP 503 for metadata of $3" >&2; exit 1; }
	# Like the real install: the named package only, not its dependencies.
	mkdir -p "$HOME/.ailang/cache/registry/$name" && cp -R "$DOWNSTREAM_FAKE_REGISTRY/$name/$ver" "$HOME/.ailang/cache/registry/$name/$ver"
	exit 0
fi
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
case "$content" in *OWNBADMANIFEST*) failj "$f" "package resolution setup failed: cannot load package manifest $PWD/ailang.toml: invalid manifest $PWD/ailang.toml: exported module main must start with package name";; esac
case "$content" in *DEPBADMANIFEST*) failj "$f" "package resolution setup failed: cannot load package manifest $AILANG_PACKAGE_ROOT/acme/bad/1.0.0/ailang.toml: invalid manifest $AILANG_PACKAGE_ROOT/acme/bad/1.0.0/ailang.toml: bad";; esac
case "$content" in *UNEXPORTED*) failj "$f" "module loading error: failed to load pkg/acme/p/hidden: failed to resolve package import pkg/acme/p/hidden: module acme/p/hidden is not exported by package acme/p";; esac
case "$content" in *NEEDDEPS*)
	# Resolve through the lock beside the file, as the loader does.
	[ -f ailang.lock ] || failj "$f" "module loading error: package acme/x not found in ailang.lock; run 'ailang lock'"
	grep -q 'ailang.lock/v1' ailang.lock || failj "$f" "package resolution setup failed: invalid lock file ailang.lock: unsupported lock file schema"
	for nv in $(grep -o '"name":"[^"]*","version":"[^"]*","source":"registry"' ailang.lock | sed 's/"name":"\([^"]*\)","version":"\([^"]*\)".*/\1@\2/'); do
		[ -d "$AILANG_PACKAGE_ROOT/${nv%@*}/${nv##*@}" ] || failj "$f" "package resolution setup failed: registry package $nv is not provisioned in AILANG_PACKAGE_ROOT"
	done;;
esac
case "$content" in *MSGDRIFT*)
	if [ "$ARM" = cand ]; then failj "$f" "type error in m (decl 0): cannot unify string vs int at m.ail:9:2"; fi
	failj "$f" "type error in m (decl 0): cannot unify string vs int at m.ail:3:1";;
esac
case "$content" in *TOUCH_FILE=*)
	# a bystander: something OUTSIDE the run's own trees changes while the gate runs
	touch_path=${content#*TOUCH_FILE=}; touch_path=${touch_path%%[[:space:]]*}
	echo x > "$touch_path";;
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
