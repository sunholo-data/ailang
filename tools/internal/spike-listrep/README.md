# THROWAWAY — DO NOT IMPORT

This package is the LC-1 representation benchmark harness. It lives below
`tools/internal` so Go permits benchmark consumers below `tools/` while mechanically
rejecting production consumers elsewhere in the module. It is retained only so the
spike measurements can be reproduced; it is not production list machinery.

The C0 implementation mirrors `internal/builtins/list.go:98-105` locally so drift is
visible next to the benchmark code:

```go
result := make([]eval.Value, 0, 1+len(tail.Elements))
result = append(result, head)
result = append(result, tail.Elements...)
return &eval.ListValue{Elements: result}, nil
```

The spike omits `EffContext` and the type-error path because its constructor accepts a
typed `List`; `slicelist.go` otherwise performs the same preallocated shallow copy.
