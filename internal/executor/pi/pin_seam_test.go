package pi

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// The pi pin lives in TWO places: ExpectedPackage/ExpectedVersion here (what
// HealthCheck asserts at run time) and the PI_PACKAGE/PI_VERSION ARGs in the
// Dockerfiles (what the images install and assert at build time). One concept,
// two implementations — the seam where drift lives. This test holds them equal
// so a bump of one without the other fails here, not on the first eval after
// the images roll. M-PI-HARNESS-UPGRADE M4.
func TestDockerfilesPinExpectedPiVersion(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	pkgRe := regexp.MustCompile(`(?m)^ARG PI_PACKAGE=(\S+)$`)
	verRe := regexp.MustCompile(`(?m)^ARG PI_VERSION=(\S+)$`)
	for _, rel := range []string{
		"docker/Dockerfile.agent-pi",
		"docker/Dockerfile.agent-eval",
		"docker/resident/Dockerfile",
	} {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		// A Windows checkout with autocrlf carries \r before \n; `$` in the
		// multiline regex then never matches (test-windows, 2026-09-16).
		b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
		pkg := pkgRe.FindSubmatch(b)
		ver := verRe.FindSubmatch(b)
		if pkg == nil || ver == nil {
			t.Fatalf("%s: no ARG PI_PACKAGE / ARG PI_VERSION line", rel)
		}
		if string(pkg[1]) != ExpectedPackage {
			t.Errorf("%s: PI_PACKAGE=%s, executor pins %s", rel, pkg[1], ExpectedPackage)
		}
		if string(ver[1]) != ExpectedVersion {
			t.Errorf("%s: PI_VERSION=%s, executor pins %s — bump both together", rel, ver[1], ExpectedVersion)
		}
	}
}
