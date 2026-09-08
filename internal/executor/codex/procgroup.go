package codex

import (
	"github.com/sunholo-data/ailang/internal/executor/proctree"
	"os/exec"
)

func configureProcessTree(cmd *exec.Cmd) { proctree.Configure(cmd) }
func killProcessTree(cmd *exec.Cmd)      { proctree.Kill(cmd) }
