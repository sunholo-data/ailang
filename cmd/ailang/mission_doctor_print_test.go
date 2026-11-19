package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mission"
)

func TestMissionDoctorPrint_ZeroFindingsByteIdentical(t *testing.T) {
	rep := &mission.Report{Rows: []mission.Row{{Name: "a"}, {Name: "b"}}}
	var buf bytes.Buffer
	printDoctorReport(&buf, rep)
	if got, want := buf.String(), "2 mission(s): no drift\n"; got != want {
		t.Fatalf("zero-finding output = %q, want %q", got, want)
	}
}

func TestMissionDoctorPrint_NotesThenSummary(t *testing.T) {
	rep := &mission.Report{
		Rows:     []mission.Row{{Name: "a"}},
		Findings: []mission.Finding{{Mission: "a", Kind: "ruling", Severity: mission.Note, Detail: "x"}},
	}
	var buf bytes.Buffer
	printDoctorReport(&buf, rep)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), buf.String())
	}
	if last := lines[len(lines)-1]; last != "1 mission(s): no drift" {
		t.Fatalf("last line = %q, want summary", last)
	}
}

func TestMissionDoctorPrint_DriftHasNoSummary(t *testing.T) {
	rep := &mission.Report{
		Rows: []mission.Row{{Name: "a"}},
		Findings: []mission.Finding{
			{Mission: "a", Kind: "ruling", Severity: mission.Note, Detail: "n"},
			{Mission: "a", Kind: "plist", Severity: mission.Drift, Detail: "d"},
		},
	}
	var buf bytes.Buffer
	printDoctorReport(&buf, rep)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "[DRIFT]") {
		t.Fatalf("last line = %q, want the drift finding", last)
	}
	if strings.Contains(buf.String(), "no drift") {
		t.Fatalf("summary printed despite drift: %q", buf.String())
	}
}
