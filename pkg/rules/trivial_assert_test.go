package rules

import (
	"testing"
)

func TestTrivialAssertRule_Detects(t *testing.T) {
	rule := &TrivialAssertRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "assert.True(t, true)",
			tf: &TestFunc{
				Name: "TestTrivial", Line: 10, HasBody: true, BodyLength: 1,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "True", FullName: "assert.True",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, Value: "true"},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "assert.Equal(t, 1, 1)",
			tf: &TestFunc{
				Name: "TestTrivialEqual", Line: 10, HasBody: true, BodyLength: 1,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "Equal", FullName: "assert.Equal",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, Value: "1"},
							{IsLiteral: true, Value: "1"},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "assert.False(t, false)",
			tf: &TestFunc{
				Name: "TestTrivialFalse", Line: 10, HasBody: true, BodyLength: 1,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "False", FullName: "assert.False",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, Value: "false", IsZeroVal: true},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "assert.Nil(t, nil)",
			tf: &TestFunc{
				Name: "TestTrivialNil", Line: 10, HasBody: true, BodyLength: 1,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "Nil", FullName: "assert.Nil",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, IsNil: true, Value: "nil"},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "assert.Equal(t, 42, x) — real assertion",
			tf: &TestFunc{
				Name: "TestReal", Line: 10, HasBody: true, BodyLength: 2,
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
		{
			name: "assert.True(t, x) — real assertion",
			tf: &TestFunc{
				Name: "TestRealTrue", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{
						Receiver: "assert", Function: "True", FullName: "assert.True",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsVariable: true, VarName: "x"},
						},
					},
				},
			},
			wantHit: false,
		},
		{
			name: "require.Equal(t, \"hello\", \"hello\") — trivial",
			tf: &TestFunc{
				Name: "TestRequireTrivial", Line: 10, HasBody: true, BodyLength: 1,
				CallExprs: []CallExpr{
					{
						Receiver: "require", Function: "Equal", FullName: "require.Equal",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsLiteral: true, Value: `"hello"`},
							{IsLiteral: true, Value: `"hello"`},
						},
					},
				},
			},
			wantHit: true,
		},
		{
			name: "t.Errorf with literal — not a trivial assertion pattern",
			tf: &TestFunc{
				Name: "TestTErrorf", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{
						Receiver: "t", Function: "Errorf", FullName: "t.Errorf", IsTestingT: true,
						Args: []Arg{
							{IsLiteral: true, Value: `"expected 3"`},
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
				if findings[0].Rule != "trivial-assertion" {
					t.Errorf("expected rule 'trivial-assertion', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestTrivialAssertRule_Metadata(t *testing.T) {
	rule := &TrivialAssertRule{}
	if rule.ID() != "trivial-assertion" {
		t.Errorf("expected ID 'trivial-assertion', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP0 {
		t.Errorf("expected SeverityP0, got %v", rule.Severity())
	}
}
