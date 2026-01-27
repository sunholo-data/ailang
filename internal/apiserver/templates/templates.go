// Package templates provides embedded templates for scaffolding AILANG web apps.
package templates

import "embed"

//go:embed web_app/*
var WebAppFS embed.FS
