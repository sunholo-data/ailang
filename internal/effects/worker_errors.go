package effects

import (
	"runtime"

	"github.com/sunholo-data/ailang/internal/eval"
)

func WorkerCancellationSupported() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "linux" }
func WorkerCancelOK() eval.Value {
	return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{&eval.UnitValue{}}}
}
func WorkerCancelFailure(ctor, detail string) eval.Value {
	return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Err", Fields: []eval.Value{&eval.TaggedValue{ModulePath: "std/process", TypeName: "WorkerCancelError", CtorName: ctor, Fields: []eval.Value{&eval.StringValue{Value: detail}}}}}
}
func WorkerCancelTimeout() eval.Value {
	return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Err", Fields: []eval.Value{&eval.TaggedValue{ModulePath: "std/process", TypeName: "WorkerCancelError", CtorName: "WorkerCancelTimedOut", Fields: []eval.Value{&eval.IntValue{Value: int(WorkerShutdownTimeout.Milliseconds())}}}}}
}
