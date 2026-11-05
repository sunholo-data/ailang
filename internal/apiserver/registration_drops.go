package apiserver

import (
	"fmt"
	"log"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
)

// Startup checks for modules the under-basePath filter dropped
// (M-SERVEAPI-SURFACE-DROPS), split from server.go.

// AllowDropsEnvVar is the env var that demotes a fail-fast on
// @route-bearing module drops to a strong WARN log line. Set to "1" by
// operators who need to start a partially-degraded server during a
// migration (e.g. immediately after upgrading from a pre-fix AILANG
// version where the same layout was silently degraded). NOT a CLI flag
// — env-var-only forces operators to make the bypass explicit in their
// Dockerfile / deployment manifest. See M-SERVEAPI-SURFACE-DROPS.
const AllowDropsEnvVar = config.EnvServeAPIAllowDrops

// ValidateRegistration must be called after LoadModules and before
// Start. Returns a non-nil error if any module rejected by the
// under-basePath filter carries an "@route" annotation, unless
// AILANG_SERVE_API_ALLOW_DROPS=1 is set. Non-fatal drops (no @route) are
// logged in a banner WARN line but never cause an error return.
//
// M-SERVEAPI-SURFACE-DROPS converts the previously-silent under-basePath
// rejection into a startup-time check. Before this method existed, a
// handler module under basePath could import a catalog module outside
// basePath; the loader would resolve the import (so handler code
// compiled), but registerModule would silently drop the catalog module
// — leaving downstream code free to walk an empty catalog and emit
// zero-valued records. This was the root cause of the docparse v0.14.1
// billing bug.
func (s *Server) ValidateRegistration() error {
	s.mu.RLock()
	drops := make([]DroppedModule, len(s.droppedModules))
	copy(drops, s.droppedModules)
	s.mu.RUnlock()

	if len(drops) == 0 {
		return nil
	}

	var fatal []DroppedModule
	for _, d := range drops {
		for _, ann := range d.Annotations {
			if ann == "@route" {
				fatal = append(fatal, d)
				break
			}
		}
	}

	// Banner WARN line summarizing ALL drops (fatal + non-fatal), so
	// operators see the same diagnostic regardless of whether the start
	// proceeds.
	names := make([]string, 0, len(drops))
	for _, d := range drops {
		names = append(names, d.FileBaseName)
	}
	log.Printf("⚠  Dropped %d module(s) outside basePath: %s", len(drops), strings.Join(names, ", "))

	allowDrops := config.ServeAPIAllowDrops()
	if len(fatal) > 0 && allowDrops {
		log.Printf("⚠  %s=1 — starting with %d @route-bearing module(s) dropped (NOT recommended for production)",
			AllowDropsEnvVar, len(fatal))
		return nil
	}
	if len(fatal) == 0 {
		// Non-annotation drops (stdlib resolution edges, etc.) — warn
		// only, never fail.
		return nil
	}

	var msg strings.Builder
	for _, d := range fatal {
		fmt.Fprintf(&msg, "  • %s\n      declared:    %s\n      basePath:    %s\n      resolved:    %s\n      annotations: %s\n",
			d.FileBaseName, d.DeclaredPath, s.normalizedBasePath, d.PhysicalPath, strings.Join(d.Annotations, ", "))
	}
	return fmt.Errorf("%d @route-bearing module(s) dropped by under-basePath filter:\n%s", len(fatal), msg.String())
}

// DroppedModules returns a defensive copy of the current droppedModules
// slice for use by /api/v1/health and tests. Modules that registered
// successfully are NOT in this slice.
func (s *Server) DroppedModules() []DroppedModule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DroppedModule, len(s.droppedModules))
	copy(out, s.droppedModules)
	return out
}
