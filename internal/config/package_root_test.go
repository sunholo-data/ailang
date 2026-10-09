package config

import "testing"

func TestPackageRoot(t *testing.T) {
	t.Setenv(EnvPackageRoot, "/operator/packages")
	if PackageRoot() != "/operator/packages" {
		t.Fatal(PackageRoot())
	}
	t.Setenv(EnvPackageRoot, "")
	if PackageRoot() != "" {
		t.Fatal(PackageRoot())
	}
}
