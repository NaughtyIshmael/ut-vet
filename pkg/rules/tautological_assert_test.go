package rules

import (
	"testing"
)

func TestTautologicalAssertRule_Detects(t *testing.T) {
	rule := &TautologicalAssertRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "assert.Equal(t, x, x) — same variable",
			tf: &TestFunc{
				Name: "TestSelfCompare", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "Equal", FullName: "assert.Equal",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsVariable: true, VarName: "x"},
							{IsVariable: true, VarName: "x"},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "require.Equal(t, result, result)",
			tf: &TestFunc{
				Name: "TestSelfRequire", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{
						Receiver: "require", Function: "Equal", FullName: "require.Equal",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsVariable: true, VarName: "result"},
							{IsVariable: true, VarName: "result"},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "assert.Equal(t, expected, actual) — different variables",
			tf: &TestFunc{
				Name: "TestDifferentVars", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "Equal", FullName: "assert.Equal",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsVariable: true, VarName: "expected"},
							{IsVariable: true, VarName: "actual"},
						},
					},
				},
			},
			wantHit: false,
		},
		{
			name: "assert.Equal(t, 42, x) — literal vs variable",
			tf: &TestFunc{
				Name: "TestLitVsVar", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "Equal", FullName: "assert.Equal",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, Value: "42"},
							{IsVariable: true, VarName: "x"},
						},
					},
				},
			},
			wantHit: false,
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
				if findings[0].Rule != "tautological-assert" {
					t.Errorf("expected rule 'tautological-assert', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestTautologicalAssertRule_Metadata(t *testing.T) {
	rule := &TautologicalAssertRule{}
	if rule.ID() != "tautological-assert" {
		t.Errorf("expected ID 'tautological-assert', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP2 {
		t.Errorf("expected SeverityP2, got %v", rule.Severity())
	}
}
