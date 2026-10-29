package embed

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// enginePackageAssets snapshots the engine's dependency targets once, before
// evaluation. New cannot return errors, so Load/Call surface discovery errors.
type enginePackageAssets struct {
	targets map[string]effects.PkgAssetTarget
	err     error
}

func newPackageAssets(basePath string) enginePackageAssets {
	targets, err := pipeline.AssetTargetsFor(basePath)
	return enginePackageAssets{targets: targets, err: err}
}

// applyPackageAssets is called after cloning the caller's context, so replacing
// an engine context cannot lose its lock or mutate a context borrowed by others.
func (e *Engine) applyPackageAssets(ctx interface{}) {
	if effCtx, ok := ctx.(*effects.EffContext); ok && effCtx != nil {
		effCtx.PkgAssets = e.pkgAssets.targets
	}
}
