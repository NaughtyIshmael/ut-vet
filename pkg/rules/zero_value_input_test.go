package rules

import (
	"testing"
)

func TestZeroValueInputRule_Detects(t *testing.T) {
	rule := &ZeroValueInputRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "all args are zero values — nil, empty string, 0",
			tf: &TestFunc{
				Name: "TestZero", Line: 10, HasBody: true, BodyLength: 2,
				LocalFuncCalls: []string{"CreateUser"},
				CallExprs: []CallExpr{
					{
						Function: "CreateUser", FullName: "CreateUser",
						Args: []Arg{
							{IsLiteral: true, Value: `""`, IsZeroVal: true},
							{IsLiteral: true, Value: "0", IsZeroVal: true},
							{IsLiteral: true, Value: "false", IsZeroVal: true},
						},
					},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError"},
				},
			},
			wantHit: true,
		},
		{
			name: "all args are nil",
			tf: &TestFunc{
				Name: "TestAllNil", Line: 10, HasBody: true, BodyLength: 2,
				LocalFuncCalls: []string{"NewHandler"},
				CallExprs: []CallExpr{
					{
						Function: "NewHandler", FullName: "NewHandler",
						Args: []Arg{
							{IsNil: true, IsLiteral: true, IsZeroVal: true, Value: "nil"},
							{IsNil: true, IsLiteral: true, IsZeroVal: true, Value: "nil"},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "mixed zero and non-zero — not detected",
			tf: &TestFunc{
				Name: "TestMixed", Line: 10, HasBody: true, BodyLength: 2,
				LocalFuncCalls: []string{"CreateUser"},
				CallExprs: []CallExpr{
					{
						Function: "CreateUser", FullName: "CreateUser",
						Args: []Arg{
							{IsLiteral: true, Value: `"Alice"`},
							{IsLiteral: true, Value: "30"},
							{IsLiteral: true, Value: "true"},
						},
					},
				},
			},
			wantHit: false,
		},
		{
			name: "variable args — not detected",
			tf: &TestFunc{
				Name: "TestVars", Line: 10, HasBody: true, BodyLength: 3,
				LocalFuncCalls: []string{"Process"},
				CallExprs: []CallExpr{
					{
						Function: "Process", FullName: "Process",
						Args: []Arg{
							{IsVariable: true, VarName: "input"},
						},
					},
				},
			},
			wantHit: false,
		},
		{
			name: "no args — not detected (function takes no params)",
			tf: &TestFunc{
				Name: "TestNoArgs", Line: 10, HasBody: true, BodyLength: 2,
				LocalFuncCalls: []string{"GetStatus"},
				CallExprs: []CallExpr{
					{Function: "GetStatus", FullName: "GetStatus", Args: nil},
				},
			},
			wantHit: false,
		},
		{
			name: "assertion call with zero args — skip assertion calls",
			tf: &TestFunc{
				Name: "TestAssertOnly", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "assert", Function: "True", FullName: "assert.True",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, Value: "false", IsZeroVal: true},
						}},
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
				if findings[0].Rule != "zero-value-input" {
					t.Errorf("expected rule 'zero-value-input', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestZeroValueInputRule_Metadata(t *testing.T) {
	rule := &ZeroValueInputRule{}
	if rule.ID() != "zero-value-input" {
		t.Errorf("expected ID 'zero-value-input', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP1 {
		t.Errorf("expected SeverityP1, got %v", rule.Severity())
	}
}
