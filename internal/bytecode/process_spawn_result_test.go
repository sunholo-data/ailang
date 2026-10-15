package bytecode

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestProcessSpawnResultAppendAndErrorOrdinals(t *testing.T) {
	old := []string{"__io_print", "__io_println", "__io_readLine", "__io_readLineOpt", "__io_writeBytes", "__io_flush", "__io_printErr", "__io_eprintln", "__io_exit", "__terminal_info", "__terminal_withTerminal", "__terminal_readEvent", "__env_getArgs", "__process_spawn_process", "__process_write_stdin", "__process_close_stdin", "__process_cancel", "__stream_async_exec_process", "__stream_cancel_process_source"}
	for i, name := range old {
		if EffectBuiltinNames[i] != name {
			t.Fatalf("persisted index %d changed", i)
		}
	}
	if len(EffectBuiltinNames) <= len(old) || EffectBuiltinNames[len(old)] != "__process_spawn_result" {
		t.Fatal("new effect must be appended")
	}
	source, err := os.ReadFile(filepath.Join("..", "..", "std", "process.ail"))
	if err != nil {
		t.Fatal(err)
	}
	body := regexp.MustCompile(`(?s)export type ProcessError\s*=\s*(.*?)deriving`).FindSubmatch(source)
	if len(body) != 2 {
		t.Fatal("ProcessError declaration missing")
	}
	variants := regexp.MustCompile(`\|\s*(\w+)\(`).FindAllSubmatch(body[1], -1)
	if len(variants) != 7 {
		t.Fatalf("expected seven errors, got %d", len(variants))
	}
	for ordinal, v := range variants {
		name := string(v[1])
		tag, ok := StdADTTag("std/process", "ProcessError", name)
		if !ok || tag != ordinal {
			t.Fatalf("%s: tag %d want %d", name, tag, ordinal)
		}
	}
	if _, ok := StdADTTag("app", "ProcessError", "NotFound"); ok {
		t.Fatal("foreign type admitted")
	}
}
