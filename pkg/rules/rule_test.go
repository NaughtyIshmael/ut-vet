package rules

import (
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityP0, "P0"},
		{SeverityP1, "P1"},
		{SeverityP2, "P2"},
	}
	for _, tt := range tests {
		if got := tt.sev.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", int(tt.sev), got, tt.want)
		}
	}
}

func TestFindingString(t *testing.T) {
	f := Finding{
		File:     "handler_test.go",
		Line:     42,
		Rule:     "no-assertion",
		Message:  "TestCreateUser has no assertion calls",
		Severity: SeverityP0,
		TestName: "TestCreateUser",
	}
	want := "handler_test.go:42: [no-assertion] TestCreateUser has no assertion calls"
	if got := f.String(); got != want {
		t.Errorf("Finding.String() = %q, want %q", got, want)
	}
}
