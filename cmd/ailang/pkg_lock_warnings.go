package main

import (
	"fmt"
	"io"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/strutil"
)

// warnShadowedPathDeps observes the already-written resolution; it never changes
// precedence, re-resolves dependencies or makes a failed lock look successful.
func warnShadowedPathDeps(w io.Writer, manifest *pkg.PackageManifest, resolved []pkg.ResolvedPackage) {
	sources := make(map[string]string, len(resolved))
	for _, entry := range resolved {
		sources[entry.Name] = entry.Source
	}
	for _, name := range strutil.SortedKeys(manifest.Dependencies) {
		source := sources[name]
		if manifest.Dependencies[name].Path == "" || source == "" || source == "path" {
			continue
		}
		fmt.Fprintf(w, "%s dependency %s is declared as a path dep but was resolved from %s (already pinned via a transitive dependency); your local %s source is NOT in this lock\n", yellow("⚠"), name, source, name)
	}
}
