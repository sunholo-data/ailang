package builtins

// This file serves as the central registration point for all AILANG builtins.
// The actual builtin implementations are organized by category in separate files:
//
//   - string.go  : String manipulation (_str_len, _str_compare, concat_String, etc.)
//   - list.go    : List operations (concat_List)
//   - math.go    : Arithmetic, comparison, logic, conversions (add_Int, eq_Bool, etc.)
//   - io.go      : Console I/O (_io_print, _io_println, _io_readLine)
//   - net.go     : Network operations (_net_httpRequest)
//   - show.go    : Polymorphic show() function
//   - json_decode.go : JSON parsing (_json_decode)
//
// Each file has its own init() function that registers its builtins.
// This file is intentionally minimal to keep file size under 800 lines (AI-friendly).
//
// To add a new builtin:
//   1. Decide which category it belongs to (string, math, io, net, etc.)
//   2. Add the implementation to the appropriate file
//   3. Register it in that file's init() function using RegisterEffectBuiltin()
//   4. Run `ailang doctor builtins` to validate
//
// For detailed instructions, see: docs/ADDING_BUILTINS.md (coming in M-DX1.11)

// Note: JSON encoding (_json_encode) is intentionally not registered yet.
// It has complex logic for encoding Json ADT that needs migration.
// TODO: Migrate _json_encode in future iteration (M-DX1.10)
