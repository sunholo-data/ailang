package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiEmbedder implements Embedder using the Gemini Embedding API.
type GeminiEmbedder struct {
	apiKey    string
	model     string
	dimension int
	timeout   time.Duration
	client    *http.Client
}

// NewGeminiEmbedder creates a new Gemini-based embedder.
func NewGeminiEmbedder(cfg GeminiEmbedConfig) (*GeminiEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "text-embedding-004"
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		dimension = detectGeminiDimension(model)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &GeminiEmbedder{
		apiKey:    cfg.APIKey,
		model:     model,
		dimension: dimension,
		timeout:   timeout,
		client:    &http.Client{Timeout: timeout},
	}, nil
}

func detectGeminiDimension(_ string) int {
	return 768
}

type geminiEmbedRequest struct {
	Model   string             `json:"model"`
	Content geminiEmbedContent `json:"content"`
}

type geminiEmbedContent struct {
	Parts []geminiEmbedPart `json:"parts"`
}

type geminiEmbedPart struct {
	Text string `json:"text"`
}

type geminiEmbedResponse struct {
	Embedding *geminiEmbedValues `json:"embedding"`
	Error     *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type geminiEmbedValues struct {
	Values []float32 `json:"values"`
}

// Embed generates an embedding for a single text.
// For long texts, it chunks and averages the embeddings.
func (e *GeminiEmbedder) Embed(text string) ([]float32, error) {
	if len(text) <= MaxChunkSize {
		return e.embedSingle(text)
	}

	chunks := chunkText(text, MaxChunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks generated from text")
	}

	var embeddings [][]float32
	for _, chunk := range chunks {
		emb, err := e.embedSingle(chunk)
		if err != nil {
			continue
		}
		embeddings = append(embeddings, emb)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("all chunks failed to embed")
	}

	return averageEmbeddings(embeddings), nil
}

func (e *GeminiEmbedder) embedSingle(text string) ([]float32, error) {
	reqBody := geminiEmbedRequest{
		Model: "models/" + e.model,
		Content: geminiEmbedContent{
			Parts: []geminiEmbedPart{{Text: text}},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		e.model, e.apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini embed request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini embed failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var embedResp geminiEmbedResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embedResp.Error != nil {
		return nil, fmt.Errorf("gemini embed error: %s", embedResp.Error.Message)
	}

	if embedResp.Embedding == nil || len(embedResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding values returned")
	}

	return embedResp.Embedding.Values, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *GeminiEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = emb
	}
	return results, nil
}

// Dimension returns the embedding dimension.
func (e *GeminiEmbedder) Dimension() int {
	return e.dimension
}

// ModelName returns the model identifier.
func (e *GeminiEmbedder) ModelName() string {
	return "gemini:" + e.model
}
