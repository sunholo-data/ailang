package main

import (
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/proctree"
	"github.com/sunholo-data/ailang/internal/runlife"
	"os"
)

func bootstrapRunLifetime() error {
	session, err := runlife.Bootstrap(config.Raw(runlife.MetadataEnv), os.Exit, func() { _ = proctree.KillOwnGroup() })
	if session != nil {
		// The owner contains this run by killing its process group, so a
		// program's exec subprocesses must stay inside that group.
		proctree.StayInOwnerGroup()
	}
	return err
}
func completeRunLifetime() { runlife.Complete(nil) }

// Early CLI exits have no execution owner, but must still close admission and
// acknowledge cleanup to an inherited owner. os.Exit does not run defers.
func exitRunLifetime(code int) {
	completeRunLifetime()
	os.Exit(code)
}
