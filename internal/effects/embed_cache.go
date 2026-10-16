package effects

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sync"
)

// CachingEmbedder serializes misses, avoiding duplicate paid calls. Vectors are
// copied at both boundaries; neither providers nor callers own stored slices.
type CachingEmbedder struct {
	mu       sync.Mutex
	provider Embedder
	store    EmbeddingCache
	values   map[string][]float32
}

func NewCachingEmbedder(provider Embedder, store EmbeddingCache) *CachingEmbedder {
	return &CachingEmbedder{provider: provider, store: store, values: make(map[string][]float32)}
}
func (c *CachingEmbedder) Dimension() int    { return c.provider.Dimension() }
func (c *CachingEmbedder) ModelName() string { return c.provider.ModelName() }
func ValidateEmbedding(v []float32, dim int) error {
	if len(v) == 0 || (dim > 0 && len(v) != dim) {
		return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(v), dim)
	}
	for _, x := range v {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return fmt.Errorf("embedding contains non-finite value")
		}
	}
	return nil
}
func (c *CachingEmbedder) Embed(text string) ([]float32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	model := c.ModelName()
	key := model + "\x00" + hash
	if v, ok := c.values[key]; ok {
		return append([]float32(nil), v...), nil
	}
	if c.store != nil {
		v, found, err := c.store.Get(model, hash, c.Dimension())
		if err != nil {
			return nil, err
		}
		if found {
			if err := ValidateEmbedding(v, c.Dimension()); err != nil {
				return nil, err
			}
			c.values[key] = append([]float32(nil), v...)
			return append([]float32(nil), v...), nil
		}
	}
	v, err := c.provider.Embed(text)
	if err != nil {
		return nil, err
	}
	if err := ValidateEmbedding(v, c.Dimension()); err != nil {
		return nil, err
	}
	if c.store != nil {
		if err := c.store.Put(model, hash, v); err != nil {
			return nil, err
		}
	}
	c.values[key] = append([]float32(nil), v...)
	return append([]float32(nil), v...), nil
}
func (c *CachingEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		var err error
		vectors[i], err = c.Embed(text)
		if err != nil {
			return nil, err
		}
	}
	return vectors, nil
}
