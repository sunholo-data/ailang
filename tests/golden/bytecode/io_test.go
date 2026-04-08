package bytecode_golden_test

import "os"

func readFileImpl(path string) ([]byte, error) { return os.ReadFile(path) }
