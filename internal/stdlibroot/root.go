// Package stdlibroot decides where the AILANG standard library lives. It is the
// ONE resolver every command uses — the module loader, `ailang docs` and the
// import-hint index (M-STDLIB-ROOT-RESOLUTION). Before it existed there were five
// resolvers with four search orders and only one of them knew about the copy
// compiled into the binary, so `ailang docs std/stream` failed outside a repo while
// `ailang run` of the same import worked.
//
// Order (the first candidate holding io.ail wins):
//
//  1. the --stdlib-path override         (Source "flag")
//  2. AILANG_STDLIB_PATH, a path-list    (Source "env")
//  3. ./std                              (Source "cwd")
//  4. <binary>/../std                    (Source "binary")
//  5. the user data dir                  (Source "user")
//  6. /usr/local/share/ailang/std, /usr/share/ailang/std (Source "system", not Windows)
//  7. the embedded std.FS                (Source "embedded", always present)
//
// An explicit override (1 or 2) that holds no stdlib is an error naming every path
// tried — a user who names a stdlib never silently gets a different one. The
// implicit tiers fall through to the embedded copy without noise.
//
// The result is memoised on its inputs (override, AILANG_STDLIB_PATH, cwd), so a
// process resolves exactly once and every module of a run comes from one root.
package stdlibroot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/std"
)

// Marker is the file whose presence makes a directory a stdlib root. It keeps an
// unrelated project `std/` directory from being mistaken for the stdlib.
const Marker = "io.ail"

// Root is a resolved stdlib root.
type Root struct {
	FS     fs.FS    // os.DirFS(Dir), or std.FS when embedded
	Dir    string   // absolute directory; "" when embedded
	Source string   // "flag" | "env" | "cwd" | "binary" | "user" | "system" | "embedded"
	Tried  []string // every candidate directory checked, in order
}

// Embedded reports whether the root is the copy compiled into the binary.
func (r Root) Embedded() bool { return r.Dir == "" }

// DisplayPath names a file of the root for messages and lexer positions:
// <Dir>/<name> on disk, "<embedded>/std/<name>" for the compiled-in copy.
func (r Root) DisplayPath(name string) string {
	if r.Embedded() {
		return "<embedded>/std/" + name
	}
	return filepath.Join(r.Dir, filepath.FromSlash(name))
}

// Options is the process-wide configuration the CLI sets once (from `run`'s
// --stdlib-path, --trace-loader and --strict flags) before any module loads.
type Options struct {
	Override      string // --stdlib-path; "" means none
	Trace         bool   // print the chosen root and every candidate to stderr
	StrictVersion bool   // a stdlib VERSION mismatch is an error, not a warning
}

type memoKey struct{ override, env, cwd string }

type memoVal struct {
	root Root
	err  error
}

var (
	mu   sync.Mutex
	opts Options
	memo = map[memoKey]memoVal{}
)

// Configure sets the process options. Call it before the first Resolve; it clears
// earlier resolutions so the new override takes effect.
func Configure(o Options) {
	mu.Lock()
	defer mu.Unlock()
	opts = o
	memo = map[memoKey]memoVal{}
}

// Current returns the process options set by Configure.
func Current() Options {
	mu.Lock()
	defer mu.Unlock()
	return opts
}

// Resolve returns the stdlib root. override "" means the configured override
// (Options.Override), which is itself usually "".
func Resolve(override string) (Root, error) {
	mu.Lock()
	defer mu.Unlock()
	if override == "" {
		override = opts.Override
	}
	cwd, _ := os.Getwd()
	key := memoKey{override: override, env: config.StdlibPath(), cwd: cwd}
	if v, ok := memo[key]; ok {
		return v.root, v.err
	}
	root, err := resolve(key)
	memo[key] = memoVal{root, err}
	if opts.Trace {
		trace(root, err)
	}
	return root, err
}

func resolve(k memoKey) (Root, error) {
	var tried []string

	// Explicit tiers: an override that resolves nowhere is an error, never a
	// silent fall-through to some other stdlib.
	if k.override != "" {
		return explicit("flag", "--stdlib-path", []string{k.override}, tried)
	}
	if strings.TrimSpace(k.env) != "" {
		return explicit("env", config.EnvStdlibPath, splitList(k.env), tried)
	}

	for _, c := range implicitCandidates(k.cwd) {
		abs := absPath(c.dir)
		tried = append(tried, abs)
		if isStdlibDir(abs) {
			return Root{FS: os.DirFS(abs), Dir: abs, Source: c.source, Tried: tried}, nil
		}
	}
	return Root{FS: std.FS, Source: "embedded", Tried: tried}, nil
}

func explicit(source, what string, dirs, tried []string) (Root, error) {
	for _, d := range dirs {
		abs := absPath(d)
		tried = append(tried, abs)
		if isStdlibDir(abs) {
			return Root{FS: os.DirFS(abs), Dir: abs, Source: source, Tried: tried}, nil
		}
	}
	return Root{Source: source, Tried: tried}, &NotStdlibError{What: what, Tried: tried}
}

// NotStdlibError reports an explicit override that holds no stdlib.
type NotStdlibError struct {
	What  string   // "--stdlib-path" or "AILANG_STDLIB_PATH"
	Tried []string // absolute directories checked
}

func (e *NotStdlibError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s does not name a stdlib directory (none of these contains %s):\n", e.What, Marker)
	for _, t := range e.Tried {
		fmt.Fprintf(&sb, "  - %s\n", t)
	}
	if strings.HasPrefix(e.What, "--") {
		fmt.Fprintf(&sb, "point %s at a std/ directory, or drop the flag to use the stdlib built into this binary", e.What)
	} else {
		fmt.Fprintf(&sb, "point %s at a std/ directory, or unset it to use the stdlib built into this binary", e.What)
	}
	return sb.String()
}

type candidate struct{ source, dir string }

func implicitCandidates(cwd string) []candidate {
	var cs []candidate
	if cwd != "" {
		cs = append(cs, candidate{"cwd", filepath.Join(cwd, "std")})
	}
	if bin, err := os.Executable(); err == nil {
		cs = append(cs, candidate{"binary", filepath.Join(filepath.Dir(bin), "..", "std")})
	}
	if u := UserDataDir(); u != "" {
		cs = append(cs, candidate{"user", u})
	}
	if runtime.GOOS != "windows" {
		cs = append(cs,
			candidate{"system", "/usr/local/share/ailang/std"},
			candidate{"system", "/usr/share/ailang/std"})
	}
	return cs
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, string(os.PathListSeparator)) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func absPath(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

func isStdlibDir(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, Marker))
	return err == nil && !st.IsDir()
}

// IsStdlibDir reports whether dir holds a stdlib (contains the Marker file).
// Callers that CHOOSE a root for a child process (the executor, the eval
// harness) use it so they only pass a root that will resolve.
func IsStdlibDir(dir string) bool { return isStdlibDir(dir) }

func trace(r Root, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[trace-loader] stdlib root: unresolved (%s)\n", r.Source)
	} else if r.Embedded() {
		fmt.Fprintf(os.Stderr, "[trace-loader] stdlib root: embedded (built into the binary)\n")
	} else {
		fmt.Fprintf(os.Stderr, "[trace-loader] stdlib root: %s (%s)\n", r.Dir, r.Source)
	}
	for i, t := range r.Tried {
		fmt.Fprintf(os.Stderr, "[trace-loader]   tried %d. %s\n", i+1, t)
	}
}

// UserDataDir returns the platform user data directory for an installed stdlib,
// "" when it cannot be determined.
func UserDataDir() string {
	var base string
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		if xdg := config.XDGDataHome(); xdg != "" {
			base = xdg
		} else if home := config.Home(); home != "" {
			base = filepath.Join(home, ".local", "share")
		}
	case "darwin":
		if home := config.Home(); home != "" {
			base = filepath.Join(home, "Library", "Application Support")
		}
	case "windows":
		base = config.AppData()
	}
	if base == "" {
		return ""
	}
	return filepath.Join(base, "ailang", "std")
}
