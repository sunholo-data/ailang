package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sunholo-data/ailang/internal/accessapproval"
)

// This machine protocol must never probe unrelated databases or print startup
// warnings. Language membership stays unchanged; only this explicit operator
// command has its own quiet startup contract.
func isAccessApprovalsInvocation(args []string) bool {
	return len(args) > 0 && (args[0] == "access-approvals" || (len(args) > 1 && args[0] == groupOps && args[1] == "access-approvals"))
}
func accessApprovalsCommand(args []string) error {
	if err := runAccessApprovals(args, os.Stdin, os.Stdout); err != nil {
		var protocol *accessapproval.Error
		if !errors.As(err, &protocol) {
			protocol = &accessapproval.Error{Code: "internal_error", Message: "local approval storage or execution failed"}
		}
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"error": protocol})
		os.Exit(1)
	}
	return nil
}

// runAccessApprovals consumes exactly one bounded JSON document and emits one
// canonical JSON document. The caller supplies the authenticated actor and
// restricts access to this trusted local CLI; it is not an authorization API.
func runAccessApprovals(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("access-approvals", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("state-dir", "", "explicit absolute private local state directory")
	source := flags.String("source", "", "explicit daneel-local authority")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := fmt.Fprintln(stdout, "Usage: ailang access-approvals --state-dir ABS_PATH --source daneel-local COMMAND\n\nJSON commands: request, get, list, decide, claim, execute, receipt, cancel, reconcile.\nOnly dummy/canary is configured. See docs/internal/access-approvals.md.")
			return err
		}
		return &accessapproval.Error{Code: "invalid_input", Message: "invalid flags; use --state-dir PATH --source daneel-local COMMAND"}
	}
	if flags.NArg() != 1 {
		return &accessapproval.Error{Code: "invalid_command", Message: "exactly one access-approvals command is required"}
	}
	const maxInput = 16384
	raw, err := io.ReadAll(io.LimitReader(stdin, maxInput+1))
	if err != nil {
		return err
	}
	if len(raw) > maxInput {
		return &accessapproval.Error{Code: "invalid_input", Message: "JSON input exceeds 16384 bytes"}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var in accessapproval.Input
	if err := decoder.Decode(&in); err != nil {
		return &accessapproval.Error{Code: "invalid_input", Message: "expected one JSON command object with known fields"}
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return &accessapproval.Error{Code: "invalid_input", Message: "expected exactly one JSON command object"}
	}
	s, err := accessapproval.Open(*dir, *source)
	if err != nil {
		return err
	}
	defer s.Close()
	result, err := s.Handle(context.Background(), flags.Arg(0), in)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(result)
}
