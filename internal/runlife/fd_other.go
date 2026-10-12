//go:build !unix

package runlife

func closeOnExec(fd int) {}
