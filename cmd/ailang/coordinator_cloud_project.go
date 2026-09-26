package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/config"
)

// `--remote gcp` should not also require you to export a project.
//
// Naming the plane (`--remote gcp`) is enough intent; the project is
// discoverable. Discovery is config.CloudProject — the one resolver, whose
// precedence is documented there — plus one extra last resort that is specific
// to this command family: AILANG_MESSAGES_PROJECT, the messaging pin every
// machine doing AILANG work exports (CLAUDE.md's session-start block), which
// is nearly always the plane a coordinator command means. The SOURCE is
// reported alongside the value: a command that mutates a plane must say which
// plane, and "which project, and how did it decide" is half of that answer.
//
// `gcloud config get-value project` is deliberately NOT a source. It was, for
// about ten minutes, until the first test with a clean environment resolved to
// `aitana-multivac-dev` — this machine's gcloud default, and not the AILANG
// fleet at all. The banner named the source, but a banner is not a safeguard:
// `--clear-orphans` against the wrong project would have been silent, wrong and
// irreversible.

// projectFromMessages is the source label for the messaging pin.
const projectFromMessages config.Source = "AILANG_MESSAGES_PROJECT"

// resolveCloudProject finds the GCP project for a cloud-plane command and
// says where it came from. It fails with errNoCloudProject's explanation.
func resolveCloudProject(ctx context.Context, mode string) (string, config.Source, error) {
	project, source, err := config.CloudProjectSource(ctx)
	if err == nil {
		return project, source, nil
	}
	if !errors.Is(err, config.ErrNoCloudProject) {
		return "", "", err
	}
	if v := config.MessagesProject(); v != "" {
		return v, projectFromMessages, nil
	}
	return "", "", errNoCloudProject(mode)
}

// errNoCloudProject explains every place the project could have come from,
// because "must be set" does not tell you what to set it to.
func errNoCloudProject(mode string) error {
	return fmt.Errorf(
		"cannot act on the %s plane: no GCP project found.\n"+
			"  Tried: AILANG_CLOUD_PROJECT, GOOGLE_CLOUD_PROJECT, pubsub.project_id in\n"+
			"  ~/.ailang/config.yaml, the GCE metadata server, AILANG_MESSAGES_PROJECT.\n"+
			"  Your gcloud default is deliberately NOT used — it points at whatever you\n"+
			"  last worked on, and acting on the wrong project would be silent.\n"+
			"  Fix with:\n"+
			"    export AILANG_CLOUD_PROJECT=<project>", mode)
}
