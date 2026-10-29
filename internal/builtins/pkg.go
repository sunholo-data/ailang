package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Package-asset builtin for AILANG (M-EXT-PORTABILITY-GATE, v0.19.0).
// Lets packages bundle helper files (scripts, schemas, templates) under their
// assets/ subdirectory and resolve them at runtime via std/package.assetPath.

func init() {
	registerPkg()
}

func registerPkg() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "FS", "pkgAssetPath", args)
	}
	typeFn := func() types.Type {
		T := types.NewBuilder()
		// (string, string) -> Result[string, string] ! {FS}
		return T.Func(T.String(), T.String()).
			Returns(T.App("Result", T.String(), T.String())).
			Effects("FS")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/package",
		Name:    "_pkg_asset_path",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "FS",
		Type:    typeFn,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: "Resolve an asset shipped inside an installed package",
			Params: []ParamDoc{
				{Name: "pkg_name", Description: "Package name in vendor/name form (e.g., \"sunholo/motoko_ext_mcp\")"},
				{Name: "rel_path", Description: "Asset path relative to the package's assets/ subdirectory"},
			},
			Returns: "Result[string, string] - Ok(absolute path) when the asset exists, Err(message) otherwise",
			Examples: []Example{
				{
					Code:        `match _pkg_asset_path("sunholo/motoko_ext_mcp", "mcp-call.mjs") { Ok(p) => p, Err(e) => "missing: " ++ e }`,
					Description: "Resolve a bundled helper script for execution",
				},
			},
			LongDesc: `Resolves the absolute path of a file shipped under the package's
assets/ subdirectory. A dependency in the running program's ailang.lock uses its
resolved directory (registry, path, or git). Missing locked targets return Err
with the locked version and install guidance, never another cached version.
Without a lock entry, selects the highest semantic version installed under
~/.ailang/cache/registry/ or AILANG_PACKAGE_ROOT (0.10.0 beats 0.9.0).
Invalid versions are skipped, and prereleases sort before their release.
The REPL has no lock context. The root package resolves its own source assets;
engine targets stay fixed across calls and context replacements.

Use this builtin via the std/package.assetPath wrapper rather than calling it
directly. Returns Err when the package is not installed, the path is unsafe
(absolute or contains ..), or the asset does not exist.

Bundling: declare assets in ailang.toml:

  [assets]
  files = ["mcp-call.mjs", "schemas/tool-call.json"]

At publish time, declared assets must exist under assets/ or publish is rejected.`,
			SeeAlso:   []string{},
			Since:     "v0.19.0",
			Stability: StabilityExperimental,
			Tags:      []string{"package", "asset", "extension"},
			Category:  "package",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _pkg_asset_path: %v", err))
	}
}
