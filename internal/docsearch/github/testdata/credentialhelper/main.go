// A portable fake credential/repository helper. It never reads real credentials.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	f, err := os.OpenFile(os.Getenv("DOCSEARCH_HELPER_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		os.Exit(2)
	}
	fmt.Fprintln(f, name)
	f.Close()
	if name == "git" {
		fmt.Println("https://github.com/sunholo-data/ailang.git")
		return
	}
	os.Exit(1)
}
