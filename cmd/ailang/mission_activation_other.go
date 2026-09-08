//go:build !darwin && !linux

package main

import (
	"context"
	"errors"
	"io"
)

func activationProcessAlive(int) (bool, error) {
	return false, errors.New("activation unsupported on this host")
}
func activationLegacyIdle(context.Context, string) error {
	return errors.New("activation unsupported on this host")
}
func activationSessionStopped(context.Context, int) error {
	return errors.New("activation unsupported on this host")
}
func superviseActivationChild(context.Context, string, *activationProcess, io.Writer) error {
	return errors.New("activation unsupported on this host")
}
