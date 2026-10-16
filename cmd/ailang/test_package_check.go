package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/check"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// checkUntestedSources type-checks every non-test source module of a package,
// the way `go test ./...` refuses a package that does not build. Before #1845
// `ailang test --package` compiled only *_test.ail files and what they import,
// so a type error in a module no test touched still printed "All tests passed"
// and exited 0. Modules a test already compiled are compile-cache hits here.
//
// Only failures that stop a module compiling count (compile, inter-function
// reference, read, timeout). The publish-boundary STRICT_FALLBACK rule stays a
// `check --package` concern; it is a policy gate, not a build failure.
// Messages go to stderr so `--format json` stdout stays one JSON document.
func checkUntestedSources(absDir string, manifest *pkg.PackageManifest) bool {
	sources, _, err := check.DiscoverPackageSources(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to discover package sources: %v\n", red("Error"), err)
		return false
	}
	var nonTest []string
	for _, f := range sources {
		if !strings.HasSuffix(f, "_test.ail") {
			nonTest = append(nonTest, f)
		}
	}
	if len(nonTest) == 0 {
		return true
	}
	report := check.CheckPackageFiles(absDir, manifest, nonTest, check.PackageOptions{})
	ok := true
	for _, fr := range report.Files {
		switch fr.Status {
		case check.FilePassed, check.FileStrictFallback:
			continue
		}
		if ok {
			fmt.Fprintf(os.Stderr, "%s package source modules failed to compile:\n", red("✗"))
			ok = false
		}
		for _, e := range fr.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
	}
	return ok
}
