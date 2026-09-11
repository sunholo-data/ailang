package main

import (
	"strings"
	"testing"
)

// The rig ran for days on a subscription it had not been asked to pull, because
// --extra-messages-sub was accepted and ignored. The daemon looked healthy the
// whole time; that is the failure being refused here.

func TestValidateSubscriptionFlags(t *testing.T) {
	tests := []struct {
		name      string
		extraSub  string
		extraEnvs []string
		wantErr   bool
	}{
		{"the rig's actual misconfiguration", "messages-rig", nil, true},
		{"…and with an empty (not nil) env list", "messages-rig", []string{}, true},
		// Legitimate: the flag renames the subscription for real extra sources.
		{"extra sub WITH extra envs", "messages-rig", []string{"prod"}, false},
		{"neither given", "", nil, false},
		// --messages-sub is the correct flag for the primary and is unaffected.
		{"extra envs but no extra sub", "", []string{"prod"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubscriptionFlags(tt.extraSub, tt.extraEnvs)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			// A refusal that does not name the fix just moves the confusion.
			for _, want := range []string{"messages-rig", "--messages-sub", "NOTHING would subscribe"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the error must contain %q so it can be acted on:\n%s", want, err)
				}
			}
		})
	}
}
