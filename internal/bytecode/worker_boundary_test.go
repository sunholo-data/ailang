package bytecode

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestWorkerEffectAppendPreservesExistingIndices(t *testing.T) {
	for i, name := range []string{
		"__io_print", "__io_println", "__io_readLine", "__io_readLineOpt", "__io_writeBytes", "__io_flush", "__io_printErr", "__io_eprintln", "__io_exit", "__terminal_info", "__terminal_withTerminal", "__terminal_readEvent", "__env_getArgs",
	} {
		if len(EffectBuiltinNames) <= i || EffectBuiltinNames[i] != name {
			t.Fatalf("persisted effect index %d changed from %s", i, name)
		}
	}
}

func TestWorkerCancelOrdinalsMatchStdlibDeclaration(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "std", "process.ail"))
	if err != nil {
		t.Fatal(err)
	}
	body := regexp.MustCompile(`(?s)export type WorkerCancelError\s*=\s*(.*?)deriving`).FindSubmatch(source)
	if len(body) != 2 {
		t.Fatal("missing WorkerCancelError declaration")
	}
	variants := regexp.MustCompile(`\|\s*(\w+)\(`).FindAllSubmatch(body[1], -1)
	if len(variants) != 4 {
		t.Fatalf("expected four cancellation constructors, found %d", len(variants))
	}
	for ordinal, variant := range variants {
		name := string(variant[1])
		tag, ok := StdADTTag("std/process", "WorkerCancelError", name)
		if !ok || tag != ordinal {
			t.Fatalf("%s: bridge tag %d, declared %d", name, tag, ordinal)
		}
	}
}
