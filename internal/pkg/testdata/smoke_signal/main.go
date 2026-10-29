// Process-only smoke fixture: ignores CLI arguments and terminates by signal.
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	process, err := os.FindProcess(os.Getpid())
	if err == nil {
		err = process.Kill()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(99)
	}
	time.Sleep(time.Minute)
}
