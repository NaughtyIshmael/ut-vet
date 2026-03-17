package rules

import (
	"testing"
)

func TestNoArrangeRule_Detects(t *testing.T) {
	rule := &NoArrangeRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "calls function with only nil args, has assertion",
			tf: &TestFunc{
				Name: "TestNilSetup", Line: 10, HasBody: true, BodyLength: 3,
				LocalFuncCalls: []string{"NewHandler"},
				CallExprs: []CallExpr{
					{
						Function: "NewHandler", FullName: "NewHandler",
						Args: []Arg{
							{IsNil: true, IsLiteral: true, IsZeroVal: true, Value: "nil"},
							{IsNil: true, IsLiteral: true, IsZeroVal: true, Value: "nil"},
						},
						Line: 11,
					},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError", Line: 13},
				},
			},
			wantHit: true,
		},
		{
			name: "calls function with meaningful args, has assertion",
			tf: &TestFunc{
				Name: "TestRealSetup", Line: 10, HasBody: true, BodyLength: 4,
				LocalFuncCalls: []string{"NewHandler"},
				CallExprs: []CallExpr{
					{
						Function: "NewHandler", FullName: "NewHandler",
						Args: []Arg{
							{IsVariable: true, VarName: "db"},
							{IsVariable: true, VarName: "logger"},
						},
						Line: 12,
					},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError", Line: 14},
				},
			},
			wantHit: false,
		},
		{
			name: "no local function calls — skip",
			tf: &TestFunc{
				Name: "TestNoLocal", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "assert", Function: "True", FullName: "assert.True", Line: 12},
				},
			},
			wantHit: false,
		},
		{
			name: "local func with no args — not flagged (function takes no params)",
			tf: &TestFunc{
				Name: "TestNoArgs", Line: 10, HasBody: true, BodyLength: 2,
				LocalFuncCalls: []string{"GetStatus"},
				CallExprs: []CallExpr{
					{Function: "GetStatus", FullName: "GetStatus", Args: nil, Line: 11},
					{Receiver: "assert", Function: "NotNil", FullName: "assert.NotNil", Line: 12},
				},
			},
			wantHit: false,
		},
		{
			name: "mixed zero and non-zero args — not flagged",
			tf: &TestFunc{
				Name: "TestMixed", Line: 10, HasBody: true, BodyLength: 3,
				LocalFuncCalls: []string{"Create"},
				CallExprs: []CallExpr{
					{
						Function: "Create", FullName: "Create",
						Args: []Arg{
							{IsLiteral: true, Value: `"Alice"`},
							{IsLiteral: true, Value: "0", IsZeroVal: true},
						},
						Line: 11,
					},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError", Line: 12},
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
				if findings[0].Rule != "no-arrange" {
					t.Errorf("expected rule 'no-arrange', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestNoArrangeRule_Metadata(t *testing.T) {
	rule := &NoArrangeRule{}
	if rule.ID() != "no-arrange" {
		t.Errorf("expected ID 'no-arrange', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP2 {
		t.Errorf("expected SeverityP2, got %v", rule.Severity())
	}
}
