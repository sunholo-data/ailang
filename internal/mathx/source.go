package mathx

import "embed"

// Source holds this package's algorithm files so the Go code generator can
// emit the very same implementation into compiled AILANG programs (which may
// only import the Go standard library). One source of truth: interpreter, VM
// and compiled Go all run these bytes.
//
//go:embed exp.go log.go trig.go trig_reduce.go atan.go pow.go
var Source embed.FS

// SourceFiles lists the embedded algorithm files in emission order.
var SourceFiles = []string{"exp.go", "log.go", "trig.go", "trig_reduce.go", "atan.go", "pow.go"}
