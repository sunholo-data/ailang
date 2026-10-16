package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/ai/factory"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// setupRunEmbedder snapshots embedding configuration alongside AI configuration.
// A separately configured provider wins even when completions use OpenRouter.
func setupRunEmbedder(ctx *effects.EffContext, cfg aiSetup, bound string, lazy ...bool) error {
	path := cfg.EmbedCacheDB
	if path == "" {
		path = config.EmbedCacheDB()
	}
	if !ctx.HasCap("AI") {
		if path != "" {
			return fmt.Errorf("--embed-cache-db requires AI capability")
		}
		// Embedding configuration belongs to AI-authorized runs. An unrelated
		// IO-only program must not depend on CLI brain provider credentials.
		return nil
	}
	var store effects.EmbeddingCache
	if path != "" {
		var err error
		store, err = effects.OpenEmbeddingCache(path)
		if err != nil {
			return fmt.Errorf("--embed-cache-db: %w", err)
		}
	}
	var embedder effects.Embedder
	var err error
	if cfg.Stub {
		embedder = effects.NewCachingEmbedder(stubEmbedder{}, store)
	} else {
		var plan *runEmbeddingPlan
		plan, err = prepareRunEmbeddingPlan(cfg, bound)
		if err == nil && plan != nil {
			open := func() (effects.Embedder, error) {
				e, err := plan.open()
				if err != nil {
					return nil, err
				}
				return effects.NewCachingEmbedder(e, store), nil
			}
			if len(lazy) > 0 && lazy[0] {
				embedder = &lazyRunEmbedder{open: open}
			} else {
				embedder, err = open()
			}
		}
	}
	if err != nil {
		if store != nil {
			store.Close()
		}
		return err
	}
	if ctx.AI == nil && (embedder != nil || store != nil) {
		ctx.AI = effects.NewAIContext(nil)
	}
	if ctx.AI != nil {
		if embedder != nil {
			ctx.AI.WithEmbedder(embedder)
		} else {
			ctx.AI.WithEmbedder(nil)
		}
	}
	if store != nil {
		ctx.AI.WithEmbeddingCacheResource(store)
	}
	return nil
}

func resolveRunEmbedder(cfg aiSetup, bound string) (effects.Embedder, error) {
	plan, err := prepareRunEmbeddingPlan(cfg, bound)
	if err != nil || plan == nil {
		return nil, err
	}
	return plan.open()
}

type runEmbeddingPlan struct {
	config   messaging.EmbedConfig
	explicit bool
	keyFile  string
}

func prepareRunEmbeddingPlan(cfg aiSetup, bound string) (*runEmbeddingPlan, error) {
	yamlCfg, err := messaging.LoadEmbeddingsConfig()
	if err != nil {
		return nil, fmt.Errorf("embedding configuration: %w", err)
	}
	explicit := config.EmbedProvider() != "" || yamlCfg != nil
	ecfg := messaging.LoadEmbedConfigFromEnv()
	if yamlCfg != nil && yamlCfg.Provider == "" && config.EmbedProvider() == "" {
		return nil, fmt.Errorf("embeddings.provider must be set")
	}
	if yamlCfg != nil {
		for _, timeout := range []string{yamlCfg.OpenAI.Timeout, yamlCfg.Gemini.Timeout, yamlCfg.Ollama.Timeout} {
			if timeout != "" {
				if duration, err := time.ParseDuration(timeout); err != nil || duration <= 0 {
					return nil, fmt.Errorf("embedding timeout must be a positive duration: %q", timeout)
				}
			}
		}
	}
	if !explicit {
		switch strings.ToLower(bound) {
		case "google", "gemini":
			ecfg.Provider = "gemini"
		case "openai":
			ecfg.Provider = "openai"
		case "ollama":
			ecfg.Provider = "ollama"
		default:
			return nil, nil
		}
		// These are embedding model defaults, never the completion model.
		if ecfg.Provider == "openai" && ecfg.OpenAI.Model == "" {
			ecfg.OpenAI.Model = "text-embedding-3-small"
		}
		if ecfg.Provider == "gemini" && ecfg.Gemini.Model == "" {
			ecfg.Gemini.Model = "text-embedding-004"
		}
	}
	if explicit && ecfg.Provider == "" {
		return nil, fmt.Errorf("embeddings.provider must be set")
	}
	switch ecfg.Provider {
	case "none":
		return nil, nil
	case "openai", "gemini", "ollama":
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", ecfg.Provider)
	}
	keyFile := ""
	if !explicit && (ecfg.Provider == "openai" || ecfg.Provider == "gemini") {
		keyFile = cfg.KeyFile
		if keyFile == "" {
			keyFile = config.AIKeyFile()
		}
	}
	return &runEmbeddingPlan{config: ecfg, explicit: explicit, keyFile: keyFile}, nil
}
func (p *runEmbeddingPlan) open() (effects.Embedder, error) {
	ecfg := p.config
	if p.keyFile != "" {
		key, err := factory.ReadKeyFile(p.keyFile)
		if err != nil {
			return nil, err
		}
		if ecfg.Provider == "openai" {
			ecfg.OpenAI.APIKey = key
		} else {
			ecfg.Gemini.APIKey = key
		}
	}
	// Freeze missing model defaults too: deferred setup must not re-read model env.
	if ecfg.Provider == "openai" && ecfg.OpenAI.Model == "" {
		ecfg.OpenAI.Model = "text-embedding-3-small"
	}
	if ecfg.Provider == "gemini" && ecfg.Gemini.Model == "" {
		ecfg.Gemini.Model = "text-embedding-004"
	}
	e, err := messaging.NewEmbedderFromConfig(ecfg)
	if err != nil {
		return nil, fmt.Errorf("embedding provider: %w", err)
	}
	if e == nil {
		return nil, nil
	}
	if !p.explicit {
		fmt.Fprintf(os.Stderr, "AI embedding: derived %s/%s (configure embeddings to pin provider/model)\n", ecfg.Provider, e.ModelName())
	}
	return e, nil
}

// Stub embeddings are keyless and stable across processes and versions.
type stubEmbedder struct{}

func (stubEmbedder) Dimension() int    { return 32 }
func (stubEmbedder) ModelName() string { return "ailang/stub-sha256-v1" }
func (stubEmbedder) Embed(text string) ([]float32, error) {
	h := sha256.Sum256([]byte(text))
	v := make([]float32, len(h))
	for i, b := range h {
		v[i] = (float32(b) - 127.5) / 127.5
	}
	return v, nil
}
func (e stubEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i], _ = e.Embed(t)
	}
	return out, nil
}

// lazyRunEmbedder publishes a frozen provider/cache once, including its error.
// Completion credentials are independent: embed never initializes a chat client.
type lazyRunEmbedder struct {
	once     sync.Once
	open     func() (effects.Embedder, error)
	provider effects.Embedder
	err      error
}

func (e *lazyRunEmbedder) ready() (effects.Embedder, error) {
	e.once.Do(func() {
		e.provider, e.err = e.open()
		if e.err != nil {
			e.err = lazyAuthError(e.err)
		}
		e.open = nil
	})
	return e.provider, e.err
}
func (e *lazyRunEmbedder) Embed(text string) ([]float32, error) {
	p, err := e.ready()
	if err != nil {
		return nil, err
	}
	return p.Embed(text)
}
func (e *lazyRunEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	p, err := e.ready()
	if err != nil {
		return nil, err
	}
	return p.EmbedBatch(texts)
}
func (e *lazyRunEmbedder) Dimension() int {
	p, err := e.ready()
	if err != nil {
		return 0
	}
	return p.Dimension()
}
func (e *lazyRunEmbedder) ModelName() string {
	p, err := e.ready()
	if err != nil {
		return ""
	}
	return p.ModelName()
}
