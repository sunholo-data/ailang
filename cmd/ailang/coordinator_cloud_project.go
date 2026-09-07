package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// `--remote gcp` should not also require you to export a project.
//
// The cloud backends read AILANG_CLOUD_PROJECT from the environment, so every
// coordinator command against gcp failed with "AILANG_CLOUD_PROJECT must be set"
// unless the caller prefixed it — which is exactly the kind of friction that
// gets worked around in shell history and never fixed. Naming the plane
// (`--remote gcp`) is enough intent; the project is discoverable.
//
// Discovery is ordered most-explicit first, and the SOURCE is reported alongside
// the value. A command that mutates a plane must say which plane, and "which
// project, and how did it decide" is half of that answer.
//
// `gcloud config get-value project` is deliberately NOT a source. It was, for
// about ten minutes, until the first test with a clean environment resolved to
// `aitana-multivac-dev` — this machine's gcloud default, and not the AILANG
// fleet at all. The banner named the source, but a banner is not a safeguard:
// `--clear-orphans` against the wrong project would have been silent, wrong and
// irreversible. The three variables below are all pinned deliberately by someone
// doing AILANG work (CLAUDE.md's session-start block exports the messaging pair),
// so an ambient developer default has no business outranking "no answer".

// cloudProjectSource names where a project id came from, for the plane banner.
type cloudProjectSource string

const (
	projectFromAilangEnv  cloudProjectSource = "AILANG_CLOUD_PROJECT"
	projectFromGoogleEnv  cloudProjectSource = "GOOGLE_CLOUD_PROJECT"
	projectFromMessages   cloudProjectSource = "AILANG_MESSAGES_PROJECT"
	projectSourceNotFound cloudProjectSource = ""
)

// resolveCloudProject finds the GCP project for a cloud-plane command.
func resolveCloudProject(_ context.Context) (string, cloudProjectSource) {
	for _, c := range []struct {
		env    string
		source cloudProjectSource
	}{
		{"AILANG_CLOUD_PROJECT", projectFromAilangEnv},
		{"GOOGLE_CLOUD_PROJECT", projectFromGoogleEnv},
		// The messaging plane is pinned per CLAUDE.md's session-start block, so
		// on a machine doing AILANG work this is nearly always already correct.
		{"AILANG_MESSAGES_PROJECT", projectFromMessages},
	} {
		if v := strings.TrimSpace(os.Getenv(c.env)); v != "" {
			return v, c.source
		}
	}

	return "", projectSourceNotFound
}

// errNoCloudProject explains every place the project could have come from,
// because "must be set" does not tell you what to set it to.
func errNoCloudProject(mode string) error {
	return fmt.Errorf(
		"cannot act on the %s plane: no GCP project found.\n"+
			"  Tried: AILANG_CLOUD_PROJECT, GOOGLE_CLOUD_PROJECT, AILANG_MESSAGES_PROJECT.\n"+
			"  Your gcloud default is deliberately NOT used — it points at whatever you\n"+
			"  last worked on, and acting on the wrong project would be silent.\n"+
			"  Fix with:\n"+
			"    export AILANG_CLOUD_PROJECT=ailang-multivac", mode)
}
