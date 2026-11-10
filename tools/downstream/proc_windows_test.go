//go:build windows

package main

// The process-group tests skip on Windows (needUnix); these keep them compiling.
func alive(pid int) bool { return false }

func killPid(pid int) {}
