package bytecode

// EffectBuiltinNames is an explicit supported set, independent of pure builtin
// adaptation. All calls use the host's capability/budget/trace dispatch.
var EffectBuiltinNames = []string{
	"__io_print", "__io_println", "__io_readLine", "__io_readLineOpt",
	"__io_writeBytes", "__io_flush", "__io_printErr", "__io_eprintln", "__io_exit",
	"__terminal_info", "__terminal_withTerminal", "__terminal_readEvent",
	// The demo selects its explicit adapter using command-line arguments.
	"__env_getArgs",
	// Append only: persisted bytecode uses these explicit numeric indices.
	"__process_spawn_process", "__process_write_stdin", "__process_close_stdin",
	"__process_cancel", "__stream_async_exec_process", "__stream_cancel_process_source",
}
