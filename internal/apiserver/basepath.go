package apiserver

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ResolveBasePath decides serve-api's project root from the path arguments
// (M-SERVEAPI-OPERATOR-SURFACE D5).
//
// The directory the operator named is the floor; a module header may only
// move the base path OUTWARD. For every .ail file named or found under a
// named directory, the file path minus "<declared module>.ail" is an implied
// root. Only implied roots that contain every argument count, and of those
// the outermost wins (it contains all the others, so every local module lands
// under it). With none: cwd when it contains every argument, else the
// arguments' common directory.
//
// Before this, serve-api trusted the FIRST .ail file in lexical order, so a
// client/wsclient.ail declaring `module wsclient` made client/ the base path
// of `serve-api .` and dropped the real route module outside it.
func ResolveBasePath(paths []string, cwd string) string {
	cwd = resolvePath(cwd)
	args := make([]string, 0, len(paths)) // directories each argument lives in
	var files []string
	for _, p := range paths {
		p = resolvePath(p)
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			args = append(args, p)
			files = append(files, walkAIL(p)...)
			continue
		}
		args = append(args, filepath.Dir(p))
		if strings.HasSuffix(p, ".ail") {
			files = append(files, p)
		}
	}
	if len(args) == 0 {
		return cwd
	}

	best := ""
	for _, f := range files {
		root, ok := impliedRoot(f)
		if !ok || !containsAll(root, args) {
			continue
		}
		if best == "" || len(root) < len(best) {
			best = root
		}
	}
	if best != "" {
		return best
	}
	if containsAll(cwd, args) {
		return cwd
	}
	return commonDir(args)
}

// walkAIL lists .ail files under dir, skipping nested pkg/ directories the
// same way LoadProject does.
func walkAIL(dir string) []string {
	var out []string
	_ = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			if fi.Name() == "pkg" && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".ail") {
			out = append(out, path)
		}
		return nil
	})
	return out
}

// impliedRoot is the directory a file's `module` header places it under:
// /p/api/handlers.ail declaring `module api/handlers` implies /p.
func impliedRoot(file string) (string, bool) {
	mod := readModuleDecl(file)
	if mod == "" {
		return "", false
	}
	suffix := string(filepath.Separator) + filepath.FromSlash(mod) + ".ail"
	if !strings.HasSuffix(file, suffix) {
		return "", false
	}
	return strings.TrimSuffix(file, suffix), true
}

// readModuleDecl returns the `module X` header of an .ail file: the first
// line that is neither blank nor a -- comment. "" when there is none.
func readModuleDecl(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
		return ""
	}
	return ""
}

func resolvePath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Clean(p)
}

// containsAll reports whether every dir is root or lies under it.
func containsAll(root string, dirs []string) bool {
	for _, d := range dirs {
		rel, err := filepath.Rel(root, d)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
	}
	return true
}

func commonDir(dirs []string) string {
	c := dirs[0]
	for !containsAll(c, dirs) {
		parent := filepath.Dir(c)
		if parent == c {
			break
		}
		c = parent
	}
	return c
}
