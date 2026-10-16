package runner

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sunholo-data/ailang/internal/effects"
)

// SharedStorageOptions selects durable stores. Empty paths use the matching
// environment variable, then the existing in-memory backend.
type SharedStorageOptions struct {
	SharedMemDB   string
	SharedIndexDB string
}

// SetupSharedStorage opens capability-gated stores. The caller owns cleanup,
// including on subsequent setup or evaluation failures. Partial setup closes
// every connection already opened before returning an error.
func SetupSharedStorage(ctx *effects.EffContext, opts SharedStorageOptions) (func() error, error) {
	memPath := storagePath(opts.SharedMemDB, "AILANG_SHAREDMEM_DB")
	indexPath := storagePath(opts.SharedIndexDB, "AILANG_SHAREDINDEX_DB")
	for _, spec := range []struct{ path, flag, cap string }{
		{memPath, "--sharedmem-db", "SharedMem"},
		{indexPath, "--sharedindex-db", "SharedIndex"},
	} {
		if spec.path != "" && !ctx.HasCap(spec.cap) {
			return nil, fmt.Errorf("%s needs --caps %s", spec.flag, spec.cap)
		}
	}
	var closers []func() error
	var closeOnce sync.Once
	var closeErr error
	closeAll := func() error {
		closeOnce.Do(func() {
			var errs []error
			for i := len(closers) - 1; i >= 0; i-- {
				errs = append(errs, closers[i]())
			}
			closeErr = errors.Join(errs...)
		})
		return closeErr
	}
	if ctx.HasCap("SharedMem") {
		if memPath == "" {
			SetupSharedMemHandler(ctx)
		} else {
			cache, err := effects.OpenSharedCache(memPath)
			if err != nil {
				return nil, fmt.Errorf("--sharedmem-db: %w", err)
			}
			closers = append(closers, cache.Close)
			ctx.SharedMem = effects.NewSharedMemContext(cache)
		}
	}
	if ctx.HasCap("SharedIndex") {
		if indexPath == "" {
			SetupSharedIndexHandler(ctx)
		} else {
			index, err := effects.OpenSharedIndex(indexPath)
			if err != nil {
				return nil, errors.Join(fmt.Errorf("--sharedindex-db: %w", err), closeAll())
			}
			closers = append(closers, index.Close)
			ctx.SharedIndex = effects.NewSharedIndexContext(index)
		}
	}
	return closeAll, nil
}

func storagePath(flag, env string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv(env)
}
