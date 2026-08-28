package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/storage"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M4b (decision D5, ratified by Mark 2026-08-27).
//
// `ailang models publish` pushes the repo's models.yml to the config bucket as
// registry.yml, using the same compare-and-swap store the coordinator config
// uses (M-MESSAGE-PLANE-FAIL-LOUD M4).
//
// WHY A SEPARATE OBJECT, AND WHY TERRAFORM MUST NOT DECLARE IT (D5).
// config.cloud.yaml is repo-canonical in ailang-multivac and is uploaded by
// `google_storage_bucket_object.coordinator_config`. Because terraform OWNS that
// object, a direct write to it is reverted on the next config push — that is
// exactly what clobbered two writes on 2026-08-26. models.yml is canonical in
// THIS repo, so it publishes to an object terraform never declares: not in
// terraform's state, therefore never reverted, and never a second copy that can
// drift from the file it came from.
//
// The consequence, accepted at ratification: this is the one artifact in the
// config bucket `terraform apply` cannot recreate. If it is deleted, the
// embedded floor keeps every binary running (that is what the floor is for) and
// the registry must be re-published by hand.

const registryObjectDefault = "registry.yml"

// registryLocation mirrors configLocation, but for the registry object.
func registryLocation() (bucket, object string) {
	bucket, _ = configLocation()
	object = os.Getenv("AILANG_REGISTRY_OBJECT")
	if object == "" {
		object = registryObjectDefault
	}
	return bucket, object
}

func newGCSRegistryStore(ctx context.Context) (*gcsConfigStore, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	bucket, object := registryLocation()
	return &gcsConfigStore{ctx: ctx, client: client, bucket: bucket, object: object}, nil
}

// writeRegistryCAS validates a registry then writes it only if the object is
// still at ifGeneration.
//
// It deliberately does NOT reuse writeConfigCAS: that one validates
// COORDINATOR-config invariants (agent ids, inbox uniqueness), which a registry
// does not have and would fail. The shape is the same, the schema is not, and a
// validator that passed the wrong document type would be worse than none.
func writeRegistryCAS(store configStore, data []byte, ifGeneration int64) error {
	cfg, err := modelreg.LoadModelsConfigBytes(data)
	if err != nil {
		return fmt.Errorf("refusing to publish: registry does not parse: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("refusing to publish: %w", err)
	}
	return store.Write(data, ifGeneration)
}

// modelsPublish implements `ailang models publish [--dry-run]`.
func modelsPublish(args []string) error {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" || a == "-dry-run" {
			dryRun = true
		}
	}

	path := os.Getenv(modelreg.ModelsPathEnv)
	if path == "" {
		path = "internal/modelreg/models.yml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read registry %s: %w", path, err)
	}

	// Validate before touching the network, so a bad registry costs nothing and
	// prints the same message it would have printed at the bucket.
	cfg, err := modelreg.LoadModelsConfigBytes(data)
	if err != nil {
		return fmt.Errorf("refusing to publish: registry does not parse: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("refusing to publish: %w", err)
	}

	bucket, object := registryLocation()
	fmt.Printf("registry:  %s (%d models, %d roles)\n", path, len(cfg.Models), len(cfg.Roles))
	fmt.Printf("target:    gs://%s/%s\n", bucket, object)

	if dryRun {
		fmt.Println("dry-run:   validated, nothing written")
		return nil
	}

	ctx := context.Background()
	store, err := newGCSRegistryStore(ctx)
	if err != nil {
		return err
	}
	_, gen, err := store.Read()
	if err != nil {
		// A registry that does not exist yet publishes at generation 0, which is
		// GCS's "create only if absent" precondition.
		gen = 0
	}
	if err := writeRegistryCAS(store, data, gen); err != nil {
		return err
	}
	fmt.Printf("published: generation %d -> new\n", gen)
	return nil
}
