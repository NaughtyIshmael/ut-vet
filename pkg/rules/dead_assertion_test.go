package rules

import (
	"testing"
)

func TestDeadAssertionRule_Detects(t *testing.T) {
	rule := &DeadAssertionRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "assertion after t.Fatal in body order",
			tf: &TestFunc{
				Name: "TestDeadCode", Line: 10, HasBody: true, BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Fatal", FullName: "t.Fatal", IsTestingT: true, Line: 12},
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal", Line: 14},
				},
				TerminatingStatements: []TerminatingStatement{
					{Line: 12, Kind: "t.Fatal"},
				},
			},
			wantHit: true,
		},
		{
			name: "assertion after t.FailNow",
			tf: &TestFunc{
				Name: "TestDeadFailNow", Line: 10, HasBody: true, BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "FailNow", FullName: "t.FailNow", IsTestingT: true, Line: 12},
					{Receiver: "assert", Function: "True", FullName: "assert.True", Line: 14},
				},
				TerminatingStatements: []TerminatingStatement{
					{Line: 12, Kind: "t.FailNow"},
				},
			},
			wantHit: true,
		},
		{
			name: "assertion before t.Fatal — not dead",
			tf: &TestFunc{
				Name: "TestNotDead", Line: 10, HasBody: true, BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal", Line: 12},
					{Receiver: "t", Function: "Fatal", FullName: "t.Fatal", IsTestingT: true, Line: 14},
				},
				TerminatingStatements: []TerminatingStatement{
					{Line: 14, Kind: "t.Fatal"},
				},
			},
			wantHit: false,
		},
		{
			name: "no terminating statements",
			tf: &TestFunc{
				Name: "TestNormal", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal", Line: 12},
				},
			},
			wantHit: false,
		},
		{
			name: "assertion after return statement",
			tf: &TestFunc{
				Name: "TestAfterReturn", Line: 10, HasBody: true, BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal", Line: 15},
				},
				TerminatingStatements: []TerminatingStatement{
					{Line: 13, Kind: "return"},
				},
			},
			wantHit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AnalysisContext{File: "test.go", TestFunc: tt.tf}
			findings := rule.Analyze(ctx)
			if tt.wantHit && len(findings) == 0 {
				t.Error("expected finding but got none")
			}
			if !tt.wantHit && len(findings) > 0 {
				t.Errorf("expected no findings but got: %v", findings)
			}
			if tt.wantHit && len(findings) > 0 {
				if findings[0].Rule != "dead-assertion" {
					t.Errorf("expected rule 'dead-assertion', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestDeadAssertionRule_Metadata(t *testing.T) {
	rule := &DeadAssertionRule{}
	if rule.ID() != "dead-assertion" {
		t.Errorf("expected ID 'dead-assertion', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP2 {
		t.Errorf("expected SeverityP2, got %v", rule.Severity())
	}
}
