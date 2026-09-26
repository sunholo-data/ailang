//go:build windows

package main

// Descendant termination is not claimed on Windows (D6); the test that uses
// these skips there, but the file must still compile.
func processGone(int) bool { return true }
func killProcess(int)      {}
