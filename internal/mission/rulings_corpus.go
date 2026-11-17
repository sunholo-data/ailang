package mission

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// M-FLEET-RULINGS-LEDGER M2: the corpus helpers. They decide WHICH files are checked
// (root derivation, charter location, skills walk) and WHICH lines match (literal
// matcher with the decision-ledger exclusion). Pure helpers: no finding is produced here.

// rulingsRoot derives the checkout whose ledger is checked (design §2). Exactly one of the results is set.
func rulingsRoot(missions []*Mission) (root string, skipped *Finding) {
	skip := func(detail string) (string, *Finding) {
		return "", &Finding{Mission: rulingsMission, Kind: "rulings-skipped", Severity: Note, Detail: detail}
	}
	if len(missions) == 0 {
		return skip("registry has no missions")
	}
	seen := map[string]bool{}
	for _, m := range missions {
		if r := m.Root(); r != "" {
			seen[r] = true
		}
	}
	if len(seen) == 0 {
		return skip("no mission carries a registry root")
	}
	if len(seen) > 1 {
		roots := make([]string, 0, len(seen))
		for r := range seen {
			roots = append(roots, r)
		}
		sort.Strings(roots)
		return skip("missions disagree on registry root: " + strings.Join(roots, ", "))
	}
	for r := range seen {
		return r, nil
	}
	return "", nil
}

// charterPath: absolute m.Doc as-is; m.Repo == sharedDriverRepo → filepath.Join(root, m.Doc); else filepath.Join(m.Workdir, m.Doc).
func charterPath(root string, m *Mission) string {
	if filepath.IsAbs(m.Doc) {
		return m.Doc
	}
	if m.Repo == sharedDriverRepo {
		return filepath.Join(root, m.Doc)
	}
	return filepath.Join(m.Workdir, m.Doc)
}

// rulingSkillFiles: for prefix in {".claude/skills", ".agents/skills"}: os.ReadDir(root/prefix) (IsNotExist → skip that prefix;
// other error → return it); for each DIR entry whose name has prefix "mission-": filepath.WalkDir collecting regular files
// ending ".md"; finally sort.Strings. Never filepath.Glob ("**" is unsupported).
func rulingSkillFiles(root string) ([]string, error) {
	var files []string
	for _, prefix := range []string{".claude/skills", ".agents/skills"} {
		dir := filepath.Join(root, prefix)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || !strings.HasPrefix(e.Name(), "mission-") {
				continue
			}
			walkErr := filepath.WalkDir(filepath.Join(dir, e.Name()), func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Type().IsRegular() && strings.HasSuffix(p, ".md") {
					files = append(files, p)
				}
				return nil
			})
			if walkErr != nil {
				return nil, walkErr
			}
		}
	}
	sort.Strings(files)
	return files, nil
}

// outsideLedger[i] reports whether lines[i] is OUTSIDE every decision-ledger block; marker lines count as inside.
func outsideLedger(lines []string) []bool {
	out := make([]bool, len(lines))
	inside := false
	for i, ln := range lines {
		switch {
		case strings.Contains(ln, ledgerStartMarker):
			inside = true
		case strings.Contains(ln, ledgerEndMarker):
			inside = false
		default:
			out[i] = !inside
		}
	}
	return out
}

// literalHits returns 1-based numbers of lines containing lit (byte-exact, case-sensitive; one hit per line);
// when charter is true, lines inside decision-ledger blocks are skipped.
func literalHits(lines []string, lit string, charter bool) []int {
	var outside []bool
	if charter {
		outside = outsideLedger(lines)
	}
	var hits []int
	for i, ln := range lines {
		if charter && !outside[i] {
			continue
		}
		if strings.Contains(ln, lit) {
			hits = append(hits, i+1)
		}
	}
	return hits
}

// displayPath: absolute p under root → filepath.Rel(root, p), slash-separated; a relative p is returned slash-separated;
// an absolute p outside root is returned unchanged (native separators, it is not repo-relative).
// Slash form keeps finding details identical on every OS (Windows Rel yields backslashes).
func displayPath(root, p string) string {
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(p)
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return p
	}
	return filepath.ToSlash(rel)
}
