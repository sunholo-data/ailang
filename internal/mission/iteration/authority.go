package iteration

import (
	"context"
	"fmt"
	"strings"
)

// VerifyPrerequisites checks the operator-frozen snapshot without changing Git
// state. These references bind approved content; they are not human signatures.
func VerifyPrerequisites(ctx context.Context, repo string, spec Spec) error {
	if err := ValidateRepository(ctx, repo, spec); err != nil {
		return err
	}
	for _, p := range spec.Prerequisites {
		if _, err := Git(ctx, repo, "merge-base", "--is-ancestor", p.Artifact.Commit, spec.BaseRevision); err != nil {
			return fmt.Errorf("prerequisite is outside approved base history: %w", err)
		}
		artifact, err := inspectArtifact(ctx, repo, p.Artifact.Commit, p.Artifact.Path)
		if err != nil {
			return err
		}
		if artifact.SHA256 != p.Artifact.SHA256 {
			return fmt.Errorf("prerequisite %s artifact hash changed", p.Role)
		}
		for _, a := range p.AuthorityRefs {
			if a.ArtifactDigest != artifact.SHA256 {
				return fmt.Errorf("authority does not bind prerequisite artifact")
			}
		}
		if err := verifyAuthorities(ctx, repo, spec.BaseRevision, p.AuthorityRefs); err != nil {
			return err
		}
	}
	for _, stage := range spec.Stages {
		if err := verifyAuthorities(ctx, repo, spec.BaseRevision, stage.AuthorityRefs); err != nil {
			return err
		}
	}
	return nil
}
func verifyAuthorities(ctx context.Context, repo, base string, refs []AuthorityRef) error {
	if err := validateAuthorities(refs); err != nil {
		return err
	}
	for _, a := range refs {
		if _, err := Git(ctx, repo, "merge-base", "--is-ancestor", a.Revision, base); err != nil {
			return fmt.Errorf("authority is outside approved input history: %w", err)
		}
		artifact, err := inspectArtifact(ctx, repo, a.Revision, a.Path)
		if err != nil {
			return err
		}
		if artifact.SHA256 != a.SHA256 {
			return fmt.Errorf("authority hash mismatch for %s", a.Path)
		}
		frozen, err := inspectArtifact(ctx, repo, base, a.Path)
		if err != nil || frozen.SHA256 != a.SHA256 {
			return fmt.Errorf("authority differs from approved base snapshot: %s", a.Path)
		}
		body, err := Git(ctx, repo, "cat-file", "blob", artifact.Blob)
		if err != nil {
			return err
		}
		if !strings.Contains(body, a.Locator) {
			return fmt.Errorf("authority locator %q missing from %s", a.Locator, a.Path)
		}
	}
	return nil
}
