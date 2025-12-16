package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// SharedMem builtin functions for AILANG
// These provide shared memory cache operations for semantic caching
// Part of M-DX15 (Semantic Caching MVP)

func init() {
	registerSharedMemGet()
	registerSharedMemPut()
	registerSharedMemCAS()
	registerSharedMemDelete()
	registerSharedMemKeys()
}

// ============================================================================
// SharedMem Builtins
// ============================================================================

// registerSharedMemGet registers the _sharedmem_get builtin
func registerSharedMemGet() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedmem",
		Name:    "_sharedmem_get",
		NumArgs: 1,
		IsPure:  false, // Has SharedMem effect
		Effect:  "SharedMem",
		Type:    makeSharedMemGetType,
		Impl:    sharedMemGetImpl,

		Metadata: &BuiltinMetadata{
			Description: "Get a value from the shared memory cache",
			LongDesc:    "Retrieves a value by key from the shared memory cache. Returns Some(bytes) if the key exists, None otherwise. The returned bytes are a copy - callers can safely modify them.",
			Params: []ParamDoc{
				{Name: "key", Description: "The key to look up"},
			},
			Returns: "Option[bytes]: Some(value) if key exists, None if not found",
			Examples: []Example{
				{Code: `_sharedmem_get("my-key")`, Description: "Returns Some(bytes) if cached, None otherwise"},
			},
			SeeAlso:   []string{"_sharedmem_put", "_sharedmem_cas", "_sharedmem_delete"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedmem", "cache", "get"},
			Category:  "sharedmem",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedmem_get: %v", err))
	}
}

// makeSharedMemGetType builds the type signature for _sharedmem_get
// Type: string -> Option[bytes] ! {SharedMem}
func makeSharedMemGetType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Option", T.Bytes()),
	).Effects("SharedMem")
}

// sharedMemGetImpl is the implementation for _sharedmem_get
func sharedMemGetImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	keyVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_get: expected String key, got %T", args[0])
	}

	// Verify SharedMem effect is enabled
	if ctx.SharedMem == nil {
		return nil, fmt.Errorf("_sharedmem_get: SharedMem effect not enabled (use --caps SharedMem)")
	}

	value, found := ctx.SharedMem.Cache.Get(keyVal.Value)
	ctx.SharedMem.IncrGetCount()

	if !found {
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.BytesValue{Value: value}},
	}, nil
}

// registerSharedMemPut registers the _sharedmem_put builtin
func registerSharedMemPut() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedmem",
		Name:    "_sharedmem_put",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "SharedMem",
		Type:    makeSharedMemPutType,
		Impl:    sharedMemPutImpl,

		Metadata: &BuiltinMetadata{
			Description: "Store a value in the shared memory cache",
			LongDesc:    "Stores a value at the given key in the shared memory cache. Overwrites any existing value. The value is copied - callers can safely modify the input after Put returns.",
			Params: []ParamDoc{
				{Name: "key", Description: "The key to store under"},
				{Name: "value", Description: "The bytes to store"},
			},
			Returns: "unit",
			Examples: []Example{
				{Code: `_sharedmem_put("my-key", bytes)`, Description: "Stores bytes at my-key"},
			},
			SeeAlso:   []string{"_sharedmem_get", "_sharedmem_cas", "_sharedmem_delete"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedmem", "cache", "put"},
			Category:  "sharedmem",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedmem_put: %v", err))
	}
}

// makeSharedMemPutType builds the type signature for _sharedmem_put
// Type: (string, bytes) -> unit ! {SharedMem}
func makeSharedMemPutType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Bytes()).Returns(T.Unit()).Effects("SharedMem")
}

// sharedMemPutImpl is the implementation for _sharedmem_put
func sharedMemPutImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	keyVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_put: expected String key, got %T", args[0])
	}

	bytesVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_put: expected Bytes value, got %T", args[1])
	}

	if ctx.SharedMem == nil {
		return nil, fmt.Errorf("_sharedmem_put: SharedMem effect not enabled (use --caps SharedMem)")
	}

	ctx.SharedMem.Cache.Put(keyVal.Value, bytesVal.Value)
	ctx.SharedMem.IncrPutCount()

	return &eval.UnitValue{}, nil
}

// registerSharedMemCAS registers the _sharedmem_cas builtin
func registerSharedMemCAS() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedmem",
		Name:    "_sharedmem_cas",
		NumArgs: 3,
		IsPure:  false,
		Effect:  "SharedMem",
		Type:    makeSharedMemCASType,
		Impl:    sharedMemCASImpl,

		Metadata: &BuiltinMetadata{
			Description: "Compare-and-swap a value in the shared memory cache",
			LongDesc:    "Atomically compares the current value at key with oldValue, and if they match, replaces it with newValue. Returns true if the swap succeeded. If oldValue is None, creates the key only if it doesn't exist (create-if-absent semantics).",
			Params: []ParamDoc{
				{Name: "key", Description: "The key to update"},
				{Name: "old", Description: "Expected current value (Option[bytes]): None for create-if-absent"},
				{Name: "new", Description: "New value to store if old matches"},
			},
			Returns: "bool: true if swap succeeded, false if current value didn't match",
			Examples: []Example{
				{Code: `_sharedmem_cas("key", None, new_bytes)`, Description: "Create if absent"},
				{Code: `_sharedmem_cas("key", Some(old_bytes), new_bytes)`, Description: "Update if matches"},
			},
			SeeAlso:   []string{"_sharedmem_get", "_sharedmem_put"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedmem", "cache", "cas", "atomic"},
			Category:  "sharedmem",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedmem_cas: %v", err))
	}
}

// makeSharedMemCASType builds the type signature for _sharedmem_cas
// Type: (string, Option[bytes], bytes) -> bool ! {SharedMem}
func makeSharedMemCASType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),
		T.App("Option", T.Bytes()),
		T.Bytes(),
	).Returns(T.Bool()).Effects("SharedMem")
}

// sharedMemCASImpl is the implementation for _sharedmem_cas
func sharedMemCASImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	keyVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_cas: expected String key, got %T", args[0])
	}

	// Parse oldValue: Option[bytes]
	var oldBytes []byte
	oldTagged, ok := args[1].(*eval.TaggedValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_cas: expected Option[bytes] for old, got %T", args[1])
	}
	if oldTagged.CtorName == "Some" {
		if len(oldTagged.Fields) != 1 {
			return nil, fmt.Errorf("_sharedmem_cas: Some should have 1 field, got %d", len(oldTagged.Fields))
		}
		bytesVal, ok := oldTagged.Fields[0].(*eval.BytesValue)
		if !ok {
			return nil, fmt.Errorf("_sharedmem_cas: Some field should be bytes, got %T", oldTagged.Fields[0])
		}
		oldBytes = bytesVal.Value
	} else if oldTagged.CtorName != "None" {
		return nil, fmt.Errorf("_sharedmem_cas: expected Some or None, got %s", oldTagged.CtorName)
	}
	// oldBytes is nil for None (create-if-absent semantics)

	newBytesVal, ok := args[2].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_cas: expected Bytes for new, got %T", args[2])
	}

	if ctx.SharedMem == nil {
		return nil, fmt.Errorf("_sharedmem_cas: SharedMem effect not enabled (use --caps SharedMem)")
	}

	success := ctx.SharedMem.Cache.CAS(keyVal.Value, oldBytes, newBytesVal.Value)
	ctx.SharedMem.IncrCASCount(success)

	return &eval.BoolValue{Value: success}, nil
}

// registerSharedMemDelete registers the _sharedmem_delete builtin
func registerSharedMemDelete() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedmem",
		Name:    "_sharedmem_delete",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "SharedMem",
		Type:    makeSharedMemDeleteType,
		Impl:    sharedMemDeleteImpl,

		Metadata: &BuiltinMetadata{
			Description: "Delete a key from the shared memory cache",
			LongDesc:    "Removes the value at the given key from the shared memory cache. No-op if the key doesn't exist.",
			Params: []ParamDoc{
				{Name: "key", Description: "The key to delete"},
			},
			Returns: "unit",
			Examples: []Example{
				{Code: `_sharedmem_delete("my-key")`, Description: "Removes my-key from cache"},
			},
			SeeAlso:   []string{"_sharedmem_get", "_sharedmem_put"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedmem", "cache", "delete"},
			Category:  "sharedmem",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedmem_delete: %v", err))
	}
}

// makeSharedMemDeleteType builds the type signature for _sharedmem_delete
// Type: string -> unit ! {SharedMem}
func makeSharedMemDeleteType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Unit()).Effects("SharedMem")
}

// sharedMemDeleteImpl is the implementation for _sharedmem_delete
func sharedMemDeleteImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	keyVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_sharedmem_delete: expected String key, got %T", args[0])
	}

	if ctx.SharedMem == nil {
		return nil, fmt.Errorf("_sharedmem_delete: SharedMem effect not enabled (use --caps SharedMem)")
	}

	ctx.SharedMem.Cache.Delete(keyVal.Value)

	return &eval.UnitValue{}, nil
}

// registerSharedMemKeys registers the _sharedmem_keys builtin
func registerSharedMemKeys() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/sharedmem",
		Name:    "_sharedmem_keys",
		NumArgs: 1, // Takes unit to work around M-DX10 nullary function bug
		IsPure:  false,
		Effect:  "SharedMem",
		Type:    makeSharedMemKeysType,
		Impl:    sharedMemKeysImpl,

		Metadata: &BuiltinMetadata{
			Description: "Get all keys in the shared memory cache",
			LongDesc:    "Returns a list of all keys currently stored in the shared memory cache. The returned list is a snapshot - modifications to the cache don't affect it. Takes unit parameter for M-DX10 compatibility.",
			Params: []ParamDoc{
				{Name: "_", Description: "Unit parameter (ignored, required for M-DX10 compatibility)"},
			},
			Returns: "list[string]: All keys in the cache",
			Examples: []Example{
				{Code: `_sharedmem_keys(())`, Description: "Returns all cached keys"},
			},
			SeeAlso:   []string{"_sharedmem_get"},
			Since:     "v0.5.11",
			Stability: StabilityStable,
			Tags:      []string{"sharedmem", "cache", "keys"},
			Category:  "sharedmem",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _sharedmem_keys: %v", err))
	}
}

// makeSharedMemKeysType builds the type signature for _sharedmem_keys
// Type: unit -> list[string] ! {SharedMem}
func makeSharedMemKeysType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.List(T.String())).Effects("SharedMem")
}

// sharedMemKeysImpl is the implementation for _sharedmem_keys
func sharedMemKeysImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// args[0] is unit, ignored

	if ctx.SharedMem == nil {
		return nil, fmt.Errorf("_sharedmem_keys: SharedMem effect not enabled (use --caps SharedMem)")
	}

	keys := ctx.SharedMem.Cache.Keys()
	elements := make([]eval.Value, len(keys))
	for i, k := range keys {
		elements[i] = &eval.StringValue{Value: k}
	}

	return &eval.ListValue{Elements: elements}, nil
}
