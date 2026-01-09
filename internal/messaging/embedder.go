package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ollama/ollama/api"
)

// Embedder provides text embedding capabilities for semantic search
type Embedder interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
}

// EmbedConfig configures the embedding provider
type EmbedConfig struct {
	Provider string       `yaml:"provider"` // "ollama" or "none"
	Ollama   OllamaConfig `yaml:"ollama"`
}

// OllamaConfig configures the Ollama embedding provider
type OllamaConfig struct {
	Model     string        `yaml:"model"`
	Endpoint  string        `yaml:"endpoint"`
	Dimension int           `yaml:"dimension"`
	Timeout   time.Duration `yaml:"timeout"`
	BatchSize int           `yaml:"batch_size"`
}

// DefaultEmbedConfig returns the default embedding configuration
func DefaultEmbedConfig() EmbedConfig {
	return EmbedConfig{
		Provider: "ollama",
		Ollama: OllamaConfig{
			Model:     "nomic-embed-text", // Good balance of speed/quality
			Endpoint:  "http://localhost:11434",
			Dimension: 768, // nomic-embed-text dimension
			Timeout:   30 * time.Second,
			BatchSize: 10,
		},
	}
}

// LoadEmbedConfigFromEnv loads embedding config from config file and environment variables
// Priority: env vars > config file > defaults
func LoadEmbedConfigFromEnv() EmbedConfig {
	cfg := DefaultEmbedConfig()

	// Load from config file first
	yamlCfg, err := LoadEmbeddingsConfig()
	if err == nil && yamlCfg != nil {
		if yamlCfg.Provider != "" {
			cfg.Provider = yamlCfg.Provider
		}
		if yamlCfg.Ollama.Model != "" {
			cfg.Ollama.Model = yamlCfg.Ollama.Model
		}
		if yamlCfg.Ollama.Endpoint != "" {
			cfg.Ollama.Endpoint = yamlCfg.Ollama.Endpoint
		}
		if yamlCfg.Ollama.Dimension > 0 {
			cfg.Ollama.Dimension = yamlCfg.Ollama.Dimension
		}
		if yamlCfg.Ollama.Timeout != "" {
			if parsed, err := time.ParseDuration(yamlCfg.Ollama.Timeout); err == nil {
				cfg.Ollama.Timeout = parsed
			}
		}
		if yamlCfg.Ollama.BatchSize > 0 {
			cfg.Ollama.BatchSize = yamlCfg.Ollama.BatchSize
		}
	}

	// Environment variables override config file
	if provider := os.Getenv("AILANG_EMBED_PROVIDER"); provider != "" {
		cfg.Provider = provider
	}
	if model := os.Getenv("AILANG_OLLAMA_MODEL"); model != "" {
		cfg.Ollama.Model = model
	}
	if endpoint := os.Getenv("AILANG_OLLAMA_ENDPOINT"); endpoint != "" {
		cfg.Ollama.Endpoint = endpoint
	}

	return cfg
}

// OllamaEmbedder implements Embedder using local Ollama
type OllamaEmbedder struct {
	client    *api.Client
	model     string
	dimension int
	timeout   time.Duration
}

// NewOllamaEmbedder creates a new Ollama-based embedder
func NewOllamaEmbedder(cfg OllamaConfig) (*OllamaEmbedder, error) {
	// Set OLLAMA_HOST for the client
	if cfg.Endpoint != "" {
		os.Setenv("OLLAMA_HOST", cfg.Endpoint)
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Auto-detect based on model
		dimension = detectDimension(cfg.Model)
	}

	return &OllamaEmbedder{
		client:    client,
		model:     cfg.Model,
		dimension: dimension,
		timeout:   timeout,
	}, nil
}

// detectDimension returns the embedding dimension for known models
func detectDimension(model string) int {
	dimensions := map[string]int{
		"nomic-embed-text":       768,
		"mxbai-embed-large":      1024,
		"gemma2:2b":              2048,
		"all-minilm":             384,
		"snowflake-arctic-embed": 1024,
	}
	if dim, ok := dimensions[model]; ok {
		return dim
	}
	return 768 // Default
}

// MaxChunkSize is the maximum characters per chunk for embedding
// embeddinggemma has 2K context (~8000 chars), we use 6000 to be safe
const MaxChunkSize = 6000

// Embed generates an embedding for a single text
// For long texts, it chunks and averages the embeddings
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	// If text fits in one chunk, embed directly
	if len(text) <= MaxChunkSize {
		return e.embedSingle(text)
	}

	// Chunk the text and embed each chunk
	chunks := chunkText(text, MaxChunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks generated from text")
	}

	// Embed each chunk
	var embeddings [][]float32
	for _, chunk := range chunks {
		emb, err := e.embedSingle(chunk)
		if err != nil {
			// Skip failed chunks but continue
			continue
		}
		embeddings = append(embeddings, emb)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("all chunks failed to embed")
	}

	// Average the embeddings
	return averageEmbeddings(embeddings), nil
}

// embedSingle embeds a single chunk of text
func (e *OllamaEmbedder) embedSingle(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	resp, err := e.client.Embed(ctx, &api.EmbedRequest{
		Model: e.model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("ollama embed failed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Embeddings[0], nil
}

// chunkText splits text into chunks of maxSize characters
// Uses markdown-aware boundaries: headers, code blocks, paragraphs, sentences
func chunkText(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxSize {
			chunks = append(chunks, remaining)
			break
		}

		// Try to find a good split point in priority order
		chunk := remaining[:maxSize]
		splitPoint := maxSize

		// Priority 1: Markdown header (## or #)
		if idx := findLastIndex(chunk, "\n## "); idx > maxSize/3 {
			splitPoint = idx + 1 // Keep newline with previous chunk
		} else if idx := findLastIndex(chunk, "\n# "); idx > maxSize/3 {
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, "\n### "); idx > maxSize/3 {
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, "\n```"); idx > maxSize/3 {
			// Priority 2: Code block boundary
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, "\n\n"); idx > maxSize/3 {
			// Priority 3: Paragraph boundary
			splitPoint = idx + 2
		} else if idx := findLastIndex(chunk, "\n- "); idx > maxSize/3 {
			// Priority 4: List item
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, ". "); idx > maxSize/2 {
			// Priority 5: Sentence boundary
			splitPoint = idx + 2
		} else if idx := findLastIndex(chunk, "\n"); idx > maxSize/2 {
			// Priority 6: Any line break
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, " "); idx > maxSize/2 {
			// Priority 7: Word boundary
			splitPoint = idx + 1
		}

		chunks = append(chunks, remaining[:splitPoint])
		remaining = remaining[splitPoint:]
	}

	return chunks
}

// findLastIndex finds the last occurrence of substr in s
func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// averageEmbeddings computes the mean of multiple embeddings
func averageEmbeddings(embeddings [][]float32) []float32 {
	if len(embeddings) == 0 {
		return nil
	}

	dim := len(embeddings[0])
	result := make([]float32, dim)

	for _, emb := range embeddings {
		for i, v := range emb {
			result[i] += v
		}
	}

	n := float32(len(embeddings))
	for i := range result {
		result[i] /= n
	}

	return result
}

// EmbedBatch generates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = embedding
	}

	return results, nil
}

// Dimension returns the embedding dimension
func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}

// ModelName returns the model identifier
func (e *OllamaEmbedder) ModelName() string {
	return "ollama:" + e.model
}

// CosineSimilarity computes cosine similarity between two embeddings
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EmbeddingToJSON converts an embedding to JSON string for storage
func EmbeddingToJSON(embedding []float32) string {
	data, _ := json.Marshal(embedding)
	return string(data)
}

// EmbeddingFromJSON parses an embedding from JSON string
func EmbeddingFromJSON(data string) ([]float32, error) {
	var embedding []float32
	if err := json.Unmarshal([]byte(data), &embedding); err != nil {
		return nil, err
	}
	return embedding, nil
}
