package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// SharedIndex builtin functions for AILANG
// These provide similarity-based semantic retrieval operations
// Part of M-DX16 (Deterministic Semantic Retrieval)

func init() {
	registerSharedIndexUpsert()
	registerSharedIndexDelete()
	registerSharedIndexFindSimHash()
	registerSharedIndexEntryCount()
	registerSharedIndexNamespaces()
	registerSharedIndexUpsertWithEmbedding()
	registerSharedIndexFindByEmbedding()
}

// ============================================================================
// SharedIndex Builtins
// ============================================================================

// registerSharedIndexUpsert registers the _sharedindex_upsert builtin
func registerSharedIndexUpsert() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_upsert",
		NumArgs: 5,
		IsPure:  false, // Has SharedIndex effect
		Effect:  "SharedIndex",
		Type:    makeSharedIndexUpsertType,
		Impl:    sharedIndexUpsertImpl,

		Metadata: &BuiltinMetadata{
			Description: "Insert or update an entry in the shared index",
			LongDesc:    "Adds or updates an entry in the shared index with a namespace, key, SimHash, version, and timestamp. If the key already exists in the namespace, it is overwritten.",
			Params: []ParamDoc{
				{Name: "namespace", Description: "The namespace to store in (partitions the index)"},
				{Name: "key", Description: "The unique key within the namespace"},
				{Name: "simhash", Description: "The 64-bit SimHash of the content"},
				{Name: "version", Description: "Version number for conflict detection"},
				{Name: "timestamp", Description: "Unix timestamp of the entry"},
			},
			Returns: "unit",
			Examples: []Example{
				{Code: `_sharedindex_upsert("beliefs", "belief1", simhash, 1, timestamp)`, Description: "Stores a belief in the beliefs namespace"},
			},
			SeeAlso:   []string{"_sharedindex_delete", "_sharedindex_find_simhash"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "upsert"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_upsert: %v", err))
	}
}

// makeSharedIndexUpsertType builds the type signature for _sharedindex_upsert
// Type: (string, string, int, int, int) -> unit ! {SharedIndex}
func makeSharedIndexUpsertType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(), // namespace
		T.String(), // key
		T.Int(),    // simhash
		T.Int(),    // version
		T.Int(),    // timestamp
	).Returns(T.Unit()).Effects("SharedIndex")
}

// sharedIndexUpsertImpl is the implementation for _sharedindex_upsert
func sharedIndexUpsertImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nsVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert: expected String namespace, got %T", args[0])
	}

	keyVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert: expected String key, got %T", args[1])
	}

	simhashVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert: expected Int simhash, got %T", args[2])
	}

	versionVal, ok := args[3].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert: expected Int version, got %T", args[3])
	}

	timestampVal, ok := args[4].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert: expected Int timestamp, got %T", args[4])
	}

	// Verify SharedIndex effect is enabled
	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_upsert: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	ctx.SharedIndex.Index.Upsert(
		nsVal.Value,
		keyVal.Value,
		int64(simhashVal.Value),
		int64(versionVal.Value),
		int64(timestampVal.Value),
	)
	ctx.SharedIndex.IncrementUpsert()

	return &eval.UnitValue{}, nil
}

// registerSharedIndexDelete registers the _sharedindex_delete builtin
func registerSharedIndexDelete() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_delete",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "SharedIndex",
		Type:    makeSharedIndexDeleteType,
		Impl:    sharedIndexDeleteImpl,

		Metadata: &BuiltinMetadata{
			Description: "Delete an entry from the shared index",
			LongDesc:    "Removes an entry from the shared index by namespace and key. No-op if the key doesn't exist.",
			Params: []ParamDoc{
				{Name: "namespace", Description: "The namespace containing the entry"},
				{Name: "key", Description: "The key to delete"},
			},
			Returns: "unit",
			Examples: []Example{
				{Code: `_sharedindex_delete("beliefs", "belief1")`, Description: "Removes belief1 from the beliefs namespace"},
			},
			SeeAlso:   []string{"_sharedindex_upsert", "_sharedindex_find_simhash"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "delete"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_delete: %v", err))
	}
}

// makeSharedIndexDeleteType builds the type signature for _sharedindex_delete
// Type: (string, string) -> unit ! {SharedIndex}
func makeSharedIndexDeleteType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Unit()).Effects("SharedIndex")
}

// sharedIndexDeleteImpl is the implementation for _sharedindex_delete
func sharedIndexDeleteImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nsVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_delete: expected String namespace, got %T", args[0])
	}

	keyVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_delete: expected String key, got %T", args[1])
	}

	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_delete: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	ctx.SharedIndex.Index.Delete(nsVal.Value, keyVal.Value)
	ctx.SharedIndex.IncrementDelete()

	return &eval.UnitValue{}, nil
}

// registerSharedIndexFindSimHash registers the _sharedindex_find_simhash builtin
func registerSharedIndexFindSimHash() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_find_simhash",
		NumArgs: 5,
		IsPure:  false, // Has SharedIndex effect (reads index state)
		Effect:  "SharedIndex",
		Type:    makeSharedIndexFindSimHashType,
		Impl:    sharedIndexFindSimHashImpl,

		Metadata: &BuiltinMetadata{
			Description: "Find entries similar to a query SimHash",
			LongDesc:    "Searches the shared index for entries similar to the query SimHash using hamming distance. Returns top-K results sorted by similarity score (1.0 - hamming_distance/64). In deterministic mode, ties are broken by key ASC.",
			Params: []ParamDoc{
				{Name: "namespace", Description: "The namespace to search in"},
				{Name: "query_simhash", Description: "The 64-bit SimHash to compare against"},
				{Name: "top_k", Description: "Maximum number of results to return"},
				{Name: "max_scan", Description: "Maximum entries to scan (0 = unlimited)"},
				{Name: "deterministic", Description: "true for Strict mode (deterministic ordering), false for BestEffort"},
			},
			Returns: "list[{key: string, score: float, version: int, timestamp: int}]",
			Examples: []Example{
				{Code: `_sharedindex_find_simhash("beliefs", query_hash, 5, 100, true)`, Description: "Find top 5 similar beliefs, scanning at most 100 entries"},
			},
			SeeAlso:   []string{"_sharedindex_upsert", "_simhash"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "search", "similarity"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_find_simhash: %v", err))
	}
}

// makeSharedIndexFindSimHashType builds the type signature for _sharedindex_find_simhash
// Type: (string, int, int, int, bool) -> list[{key: string, score: float, version: int, timestamp: int}] ! {SharedIndex}
func makeSharedIndexFindSimHashType() types.Type {
	T := types.NewBuilder()
	resultRecord := T.Record(
		types.Field("key", T.String()),
		types.Field("score", T.Float()),
		types.Field("version", T.Int()),
		types.Field("timestamp", T.Int()),
	)
	return T.Func(
		T.String(), // namespace
		T.Int(),    // query_simhash
		T.Int(),    // top_k
		T.Int(),    // max_scan
		T.Bool(),   // deterministic
	).Returns(T.List(resultRecord)).Effects("SharedIndex")
}

// sharedIndexFindSimHashImpl is the implementation for _sharedindex_find_simhash
func sharedIndexFindSimHashImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nsVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_simhash: expected String namespace, got %T", args[0])
	}

	queryVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_simhash: expected Int query_simhash, got %T", args[1])
	}

	topKVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_simhash: expected Int top_k, got %T", args[2])
	}

	maxScanVal, ok := args[3].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_simhash: expected Int max_scan, got %T", args[3])
	}

	deterministicVal, ok := args[4].(*eval.BoolValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_simhash: expected Bool deterministic, got %T", args[4])
	}

	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_find_simhash: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	// Convert bool to DeterminismMode
	mode := effects.DeterminismBestEffort
	if deterministicVal.Value {
		mode = effects.DeterminismStrict
	}

	results := ctx.SharedIndex.Index.FindSimilarSimHash(
		nsVal.Value,
		int64(queryVal.Value),
		topKVal.Value,
		maxScanVal.Value,
		mode,
	)
	ctx.SharedIndex.IncrementSearch(int64(len(results)))

	// Convert results to AILANG list of records
	elements := make([]eval.Value, len(results))
	for i, r := range results {
		elements[i] = &eval.RecordValue{
			Fields: map[string]eval.Value{
				"key":       &eval.StringValue{Value: r.Key},
				"score":     &eval.FloatValue{Value: r.Score},
				"version":   &eval.IntValue{Value: int(r.Version)},
				"timestamp": &eval.IntValue{Value: int(r.Timestamp)},
			},
		}
	}

	return &eval.ListValue{Elements: elements}, nil
}

// registerSharedIndexEntryCount registers the _sharedindex_entry_count builtin
func registerSharedIndexEntryCount() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_entry_count",
		NumArgs: 1,
		IsPure:  false, // Has SharedIndex effect
		Effect:  "SharedIndex",
		Type:    makeSharedIndexEntryCountType,
		Impl:    sharedIndexEntryCountImpl,

		Metadata: &BuiltinMetadata{
			Description: "Get the number of entries in a namespace",
			LongDesc:    "Returns the count of entries in the specified namespace. Returns 0 if the namespace doesn't exist.",
			Params: []ParamDoc{
				{Name: "namespace", Description: "The namespace to count entries in"},
			},
			Returns: "int: Number of entries in the namespace",
			Examples: []Example{
				{Code: `_sharedindex_entry_count("beliefs")`, Description: "Returns number of beliefs indexed"},
			},
			SeeAlso:   []string{"_sharedindex_namespaces"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "count"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_entry_count: %v", err))
	}
}

// makeSharedIndexEntryCountType builds the type signature for _sharedindex_entry_count
// Type: string -> int ! {SharedIndex}
func makeSharedIndexEntryCountType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Int()).Effects("SharedIndex")
}

// sharedIndexEntryCountImpl is the implementation for _sharedindex_entry_count
func sharedIndexEntryCountImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nsVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_entry_count: expected String namespace, got %T", args[0])
	}

	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_entry_count: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	count := ctx.SharedIndex.Index.EntryCount(nsVal.Value)

	return &eval.IntValue{Value: count}, nil
}

// registerSharedIndexNamespaces registers the _sharedindex_namespaces builtin
func registerSharedIndexNamespaces() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_namespaces",
		NumArgs: 1, // Takes unit for M-DX10 compatibility
		IsPure:  false,
		Effect:  "SharedIndex",
		Type:    makeSharedIndexNamespacesType,
		Impl:    sharedIndexNamespacesImpl,

		Metadata: &BuiltinMetadata{
			Description: "Get all namespace names in the index",
			LongDesc:    "Returns a sorted list of all namespace names that have entries in the shared index. Takes unit parameter for M-DX10 compatibility.",
			Params: []ParamDoc{
				{Name: "_", Description: "Unit parameter (ignored, required for M-DX10 compatibility)"},
			},
			Returns: "list[string]: All namespace names, sorted alphabetically",
			Examples: []Example{
				{Code: `_sharedindex_namespaces(())`, Description: "Returns all namespace names"},
			},
			SeeAlso:   []string{"_sharedindex_entry_count"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "namespaces"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_namespaces: %v", err))
	}
}

// makeSharedIndexNamespacesType builds the type signature for _sharedindex_namespaces
// Type: unit -> list[string] ! {SharedIndex}
func makeSharedIndexNamespacesType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.List(T.String())).Effects("SharedIndex")
}

// sharedIndexNamespacesImpl is the implementation for _sharedindex_namespaces
func sharedIndexNamespacesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// args[0] is unit, ignored

	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_namespaces: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	namespaces := ctx.SharedIndex.Index.Namespaces()
	elements := make([]eval.Value, len(namespaces))
	for i, ns := range namespaces {
		elements[i] = &eval.StringValue{Value: ns}
	}

	return &eval.ListValue{Elements: elements}, nil
}

// ============================================================================
// DX-17: Embedding-based Similarity Search Builtins
// ============================================================================

// registerSharedIndexUpsertWithEmbedding registers the _sharedindex_upsert_emb builtin
func registerSharedIndexUpsertWithEmbedding() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_upsert_emb",
		NumArgs: 6,
		IsPure:  false,
		Effect:  "SharedIndex",
		Type:    makeSharedIndexUpsertEmbType,
		Impl:    sharedIndexUpsertEmbImpl,

		Metadata: &BuiltinMetadata{
			Description: "Insert or update an entry with a neural embedding",
			LongDesc:    "Adds or updates an entry in the shared index with both SimHash and neural embedding. The embedding enables cosine similarity search for true semantic matching.",
			Params: []ParamDoc{
				{Name: "namespace", Description: "The namespace to store in (partitions the index)"},
				{Name: "key", Description: "The unique key within the namespace"},
				{Name: "simhash", Description: "The 64-bit SimHash of the content (for fast pre-filtering)"},
				{Name: "embedding", Description: "Neural embedding vector (e.g., 768 floats from EmbeddingGemma)"},
				{Name: "version", Description: "Version number for conflict detection"},
				{Name: "timestamp", Description: "Unix timestamp of the entry"},
			},
			Returns: "unit",
			Examples: []Example{
				{Code: `let emb = _ollama_embed("embeddinggemma", text) in _sharedindex_upsert_emb("beliefs", "b1", hash, emb, 1, ts)`, Description: "Store belief with embedding"},
			},
			SeeAlso:   []string{"_sharedindex_find_by_embedding", "_ollama_embed"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "embedding", "upsert"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_upsert_emb: %v", err))
	}
}

// makeSharedIndexUpsertEmbType builds the type signature for _sharedindex_upsert_emb
// Type: (string, string, int, list[float], int, int) -> unit ! {SharedIndex}
func makeSharedIndexUpsertEmbType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),        // namespace
		T.String(),        // key
		T.Int(),           // simhash
		T.List(T.Float()), // embedding
		T.Int(),           // version
		T.Int(),           // timestamp
	).Returns(T.Unit()).Effects("SharedIndex")
}

// sharedIndexUpsertEmbImpl is the implementation for _sharedindex_upsert_emb
func sharedIndexUpsertEmbImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nsVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: expected String namespace, got %T", args[0])
	}

	keyVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: expected String key, got %T", args[1])
	}

	simhashVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: expected Int simhash, got %T", args[2])
	}

	embListVal, ok := args[3].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: expected List embedding, got %T", args[3])
	}

	versionVal, ok := args[4].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: expected Int version, got %T", args[4])
	}

	timestampVal, ok := args[5].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: expected Int timestamp, got %T", args[5])
	}

	// Verify SharedIndex effect is enabled
	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_upsert_emb: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	// Convert embedding list to []float64
	embedding := make([]float64, len(embListVal.Elements))
	for i, elem := range embListVal.Elements {
		floatVal, ok := elem.(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("_sharedindex_upsert_emb: embedding element %d is not float, got %T", i, elem)
		}
		embedding[i] = floatVal.Value
	}

	ctx.SharedIndex.Index.UpsertWithEmbedding(
		nsVal.Value,
		keyVal.Value,
		int64(simhashVal.Value),
		embedding,
		int64(versionVal.Value),
		int64(timestampVal.Value),
	)
	ctx.SharedIndex.IncrementUpsert()
	ctx.SharedIndex.TraceUpsertWithEmbedding(nsVal.Value, keyVal.Value, len(embedding))

	return &eval.UnitValue{}, nil
}

// registerSharedIndexFindByEmbedding registers the _sharedindex_find_by_embedding builtin
func registerSharedIndexFindByEmbedding() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedindex",
		Name:    "_sharedindex_find_by_embedding",
		NumArgs: 5,
		IsPure:  false,
		Effect:  "SharedIndex",
		Type:    makeSharedIndexFindByEmbeddingType,
		Impl:    sharedIndexFindByEmbeddingImpl,

		Metadata: &BuiltinMetadata{
			Description: "Find entries similar to a query embedding using cosine similarity",
			LongDesc:    "Searches the shared index for entries with embeddings similar to the query embedding. Uses cosine similarity normalized to [0,1]. Only entries with embeddings are considered.",
			Params: []ParamDoc{
				{Name: "namespace", Description: "The namespace to search in"},
				{Name: "query_embedding", Description: "The embedding vector to find similar entries for"},
				{Name: "top_k", Description: "Maximum number of results to return"},
				{Name: "max_scan", Description: "Maximum entries to scan (0 = unlimited)"},
				{Name: "deterministic", Description: "true for Strict mode (deterministic ordering), false for BestEffort"},
			},
			Returns: "list[{key: string, score: float, version: int, timestamp: int}]",
			Examples: []Example{
				{Code: `let qemb = _ollama_embed("embeddinggemma", query) in _sharedindex_find_by_embedding("beliefs", qemb, 5, 100, true)`, Description: "Find top 5 semantically similar beliefs"},
			},
			SeeAlso:   []string{"_sharedindex_upsert_emb", "_ollama_embed", "_sharedindex_find_simhash"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedindex", "semantic", "embedding", "search", "cosine"},
			Category:  "sharedindex",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedindex_find_by_embedding: %v", err))
	}
}

// makeSharedIndexFindByEmbeddingType builds the type signature for _sharedindex_find_by_embedding
// Type: (string, list[float], int, int, bool) -> list[{key: string, score: float, version: int, timestamp: int}] ! {SharedIndex}
func makeSharedIndexFindByEmbeddingType() types.Type {
	T := types.NewBuilder()
	resultRecord := T.Record(
		types.Field("key", T.String()),
		types.Field("score", T.Float()),
		types.Field("version", T.Int()),
		types.Field("timestamp", T.Int()),
	)
	return T.Func(
		T.String(),        // namespace
		T.List(T.Float()), // query_embedding
		T.Int(),           // top_k
		T.Int(),           // max_scan
		T.Bool(),          // deterministic
	).Returns(T.List(resultRecord)).Effects("SharedIndex")
}

// sharedIndexFindByEmbeddingImpl is the implementation for _sharedindex_find_by_embedding
func sharedIndexFindByEmbeddingImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nsVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_by_embedding: expected String namespace, got %T", args[0])
	}

	embListVal, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_by_embedding: expected List query_embedding, got %T", args[1])
	}

	topKVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_by_embedding: expected Int top_k, got %T", args[2])
	}

	maxScanVal, ok := args[3].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_by_embedding: expected Int max_scan, got %T", args[3])
	}

	deterministicVal, ok := args[4].(*eval.BoolValue)
	if !ok {
		return nil, fmt.Errorf("_sharedindex_find_by_embedding: expected Bool deterministic, got %T", args[4])
	}

	if ctx.SharedIndex == nil {
		return nil, fmt.Errorf("_sharedindex_find_by_embedding: SharedIndex effect not enabled (use --caps SharedIndex)")
	}

	// Convert embedding list to []float64
	queryEmbedding := make([]float64, len(embListVal.Elements))
	for i, elem := range embListVal.Elements {
		floatVal, ok := elem.(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("_sharedindex_find_by_embedding: query_embedding element %d is not float, got %T", i, elem)
		}
		queryEmbedding[i] = floatVal.Value
	}

	// Convert bool to DeterminismMode
	mode := effects.DeterminismBestEffort
	if deterministicVal.Value {
		mode = effects.DeterminismStrict
	}

	results := ctx.SharedIndex.Index.FindSimilarByEmbedding(
		nsVal.Value,
		queryEmbedding,
		topKVal.Value,
		maxScanVal.Value,
		mode,
	)
	ctx.SharedIndex.IncrementSearch(int64(len(results)))
	ctx.SharedIndex.TraceFindByEmbedding(nsVal.Value, len(queryEmbedding), topKVal.Value, maxScanVal.Value, mode, len(results))

	// Convert results to AILANG list of records
	elements := make([]eval.Value, len(results))
	for i, r := range results {
		elements[i] = &eval.RecordValue{
			Fields: map[string]eval.Value{
				"key":       &eval.StringValue{Value: r.Key},
				"score":     &eval.FloatValue{Value: r.Score},
				"version":   &eval.IntValue{Value: int(r.Version)},
				"timestamp": &eval.IntValue{Value: int(r.Timestamp)},
			},
		}
	}

	return &eval.ListValue{Elements: elements}, nil
}
