package main

import _ "embed"

// packageAuthoringGuide is served by `ailang docs package-authoring`. It is
// embedded so the guide works offline, outside a source checkout.
//
//go:embed guides/package-authoring.md
var packageAuthoringGuide string
