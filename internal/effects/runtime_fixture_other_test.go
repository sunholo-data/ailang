//go:build !unix && !js

package effects

func runtimeFixtureGone(pid int) bool { return false }
