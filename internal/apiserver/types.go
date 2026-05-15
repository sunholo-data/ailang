package apiserver

// DroppedModule records a module that was loaded by the loader but rejected
// by registerModule's under-basePath filter. Before M-SERVEAPI-SURFACE-DROPS,
// these rejections were silent — the catalog-parsing module of a published
// package whose handlers lived under basePath could be dropped without any
// operator-visible signal, leaving handler code free to construct
// structurally-valid but semantically-empty responses (the docparse v0.14.1
// billing bug — see inbox e1814c9f).
//
// registerModule now appends a DroppedModule for every rejection.
// ValidateRegistration partitions them by Annotations: a drop carrying
// "@route" is fatal (author declared exposed intent → fail-fast). Everything
// else is a warning logged in the startup banner.
type DroppedModule struct {
	// PhysicalPath is the symlink-resolved absolute path of the dropped file.
	PhysicalPath string

	// DeclaredPath is the `module X` header as written in source. Empty for
	// files without a parsable module declaration.
	DeclaredPath string

	// FileBaseName is filepath.Base(PhysicalPath) — used in short
	// error / warning messages where the full path is too noisy.
	FileBaseName string

	// Annotations lists declarative annotations found on any function in
	// the module. Currently the set we surface is {"@route"} — that's the
	// only annotation that fail-fasts. Other annotations may be added
	// later; they're listed here for diagnostic purposes only.
	Annotations []string

	// Reason is a short tag for the rejection cause. Currently always
	// "outside-basePath" — it's the only rejection path. Kept as a string
	// to allow future rejection causes without churning the type.
	Reason string
}
