package executor

import "errors"

// ErrProcessTerminationUnconfirmed means an executor could not prove its CLI process
// tree has stopped. Cloud jobs that hold a ROTATING credential (codex auth.json, the
// agy token file) must not read, write back or release that credential then: a live
// process could still refresh it after another holder takes over. The cloud entry
// point quarantines the credential instead (cmd/ailang finalizeRotatingCredential).
//
// Shared by every executor that runs on a rotating credential, so one errors.Is
// check covers them all (design_docs/planned/m-agy-cloud-executor.md).
var ErrProcessTerminationUnconfirmed = errors.New("executor refresh owner termination unconfirmed")
