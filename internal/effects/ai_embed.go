package effects

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"time"

	ollama "github.com/sunholo-data/ailang/internal/ai/ollama/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

// EmbeddingError marks provider, fixture and input failures that Result wrappers
// may recover from. Authorization errors are raised before the operation runs.
type EmbeddingError struct{ Err error }

func (e *EmbeddingError) Error() string { return e.Err.Error() }
func (e *EmbeddingError) Unwrap() error { return e.Err }

// embeddingStub is deliberately private: real embedding calls always go to
// Ollama, independently of the configured chat provider.
type embeddingStub interface {
	embedOllama(model, text string, media bool, image, audio []byte) ([]float64, error)
}

// StubEmbed produces eight input-dependent unit coordinates. These vectors prove
// wiring and format only; they have no semantic meaning. Use fixtures for realism.
func StubEmbed(model, text string, media bool, image, audio []byte) []float64 {
	if len(image) == 0 {
		image = nil
	}
	if len(audio) == 0 {
		audio = nil
	}
	canonical, _ := json.Marshal([]any{model, text, media, image, audio})
	sum := sha256.Sum256(canonical)
	vector := make([]float64, 8)
	norm := 0.0
	for i := range vector {
		vector[i] = float64(binary.LittleEndian.Uint32(sum[i*4:]))/float64(math.MaxUint32)*2 - 1
		norm += vector[i] * vector[i]
	}
	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] /= norm
	}
	return vector
}
func (h *StubAIHandler) embedOllama(model, text string, media bool, image, audio []byte) ([]float64, error) {
	return StubEmbed(model, text, media, image, audio), nil
}

func init() {
	RegisterOp("AI", "embedOllama", func(ctx *EffContext, args []eval.Value) (eval.Value, error) { return aiEmbed(ctx, args, false) })
	RegisterOp("AI", "embedOllamaMedia", func(ctx *EffContext, args []eval.Value) (eval.Value, error) { return aiEmbed(ctx, args, true) })
}

func aiEmbed(ctx *EffContext, args []eval.Value, media bool) (eval.Value, error) {
	count := 2
	name, op := "_ollama_embed", "embedOllama"
	timeout := 30 * time.Second
	if media {
		count = 4
		name, op = "_ollama_embed_media", "embedOllamaMedia"
		timeout = 120 * time.Second
	}
	if len(args) != count {
		return nil, fmt.Errorf("%s: expected %d arguments", name, count)
	}
	model, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected String model", name)
	}
	text, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("%s: expected String text", name)
	}
	var image, audio []byte
	if media {
		im, ok := args[2].(*eval.BytesValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected Bytes image", name)
		}
		au, ok := args[3].(*eval.BytesValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected Bytes audio", name)
		}
		image, audio = im.Value, au.Value
	}
	traceArgs := []string{truncateForTrace(model.Value), truncateForTrace(text.Value)}
	if media {
		traceArgs = append(traceArgs, fmt.Sprintf("image:%d bytes", len(image)), fmt.Sprintf("audio:%d bytes", len(audio)))
	}
	vector, err := dispatchEmbed(ctx, name, model.Value, text.Value, media, image, audio, timeout)
	if err != nil {
		ctx.RecordAIEffect(op, traceArgs, "err:"+truncateForTrace(err.Error()), nil)
		return nil, &EmbeddingError{Err: err}
	}
	ctx.RecordAIEffect(op, traceArgs, fmt.Sprintf("vector:%d dimensions", len(vector)), nil)
	values := make([]eval.Value, len(vector))
	for i, v := range vector {
		values[i] = &eval.FloatValue{Value: v}
	}
	return &eval.ListValue{Elements: values}, nil
}

func dispatchEmbed(ctx *EffContext, name, model, text string, media bool, image, audio []byte, timeout time.Duration) ([]float64, error) {
	if media && text == "" && len(image) == 0 && len(audio) == 0 {
		return nil, fmt.Errorf("%s: text, image and audio are all empty", name)
	}
	// Inspect an already-resolved handler only. Starting a lazy chat provider here
	// would demand unrelated credentials or reroute a local embedding request.
	c := ctx.AI
	if c != nil && c.lazy != nil {
		c = c.lazy.metadataContext()
	}
	if c != nil {
		if stub, ok := c.handler.(embeddingStub); ok {
			return stub.embedOllama(model, text, media, image, audio)
		}
	}
	var input any = text
	if media {
		input = ollama.MediaInput(text, image, audio)
	}
	return ollama.EmbedRequest(name, model, input, timeout)
}
