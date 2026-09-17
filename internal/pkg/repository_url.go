package pkg

import (
	"net/url"
	"strings"
)

// RepoRef is a package's source location as the coordinator needs it:
// the GitHub workspace ("owner/repo"), the branch, and the subdirectory the
// package lives in — parsed from metadata.repository, which is the ONLY place
// the registry records where a package's code lives.
//
// M-PKG-QUALITY-LADDER M6 (Mark, attended 2026-09-17): a new package gets an
// agent inbox the moment it is published, derived from this — so a message
// to `pkg:<name>` dispatches instead of being accepted and silently never
// worked (12/53 packages had no inbox agent; five dogfood reports to
// pkg:sunholo/email sat unread for that reason on 2026-09-07).
type RepoRef struct {
	Workspace    string // "sunholo-data/ailang-packages"
	Branch       string // "main" ("" when the URL has no /tree/<branch>)
	Subdirectory string // "packages/gcp-auth" ("" for a repo-root package)
}

// ParseRepositoryURL understands GitHub URLs of the forms
//
//	https://github.com/<owner>/<repo>
//	https://github.com/<owner>/<repo>/tree/<branch>
//	https://github.com/<owner>/<repo>/tree/<branch>/<sub/dir>
//
// with an optional .git suffix on the repo. Anything else reports ok=false —
// a package hosted elsewhere gets no derived agent, not a wrong one.
func ParseRepositoryURL(raw string) (RepoRef, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host != "github.com" {
		return RepoRef{}, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return RepoRef{}, false
	}
	ref := RepoRef{Workspace: parts[0] + "/" + strings.TrimSuffix(parts[1], ".git")}
	if len(parts) >= 4 && parts[2] == "tree" {
		ref.Branch = parts[3]
		if len(parts) > 4 {
			ref.Subdirectory = strings.Join(parts[4:], "/")
		}
	} else if len(parts) > 2 {
		return RepoRef{}, false // /blob/, /issues/, … — not a package location
	}
	return ref, true
}

// PackageAgentID is the derived coordinator agent id for a package inbox:
// `pkg:sunholo/gcp_auth` → `pkg-sunholo-gcp-auth`, matching the hand-written
// entries the plane config already carries.
func PackageAgentID(pkgName string) string {
	return "pkg-" + strings.NewReplacer("/", "-", "_", "-").Replace(pkgName)
}
