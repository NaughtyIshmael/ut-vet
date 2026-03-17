package rules

import "testing"

func TestOnlyNilCheckRule_Detects(t *testing.T) {
	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "only assert.NoError — detected",
			tf: &TestFunc{
				Name:       "TestGetUser",
				Line:       10,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "service", Function: "GetUser", FullName: "service.GetUser"},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError",
						Args: []Arg{{IsVariable: true, VarName: "err"}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"_", "err"}, RHSCall: &CallExpr{Receiver: "service", Function: "GetUser"}, ErrorVarName: "err"},
				},
			},
			wantHit: true,
		},
		{
			name: "only require.NoError — detected",
			tf: &TestFunc{
				Name:       "TestSaveItem",
				Line:       20,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "repo", Function: "Save", FullName: "repo.Save"},
					{Receiver: "require", Function: "NoError", FullName: "require.NoError",
						Args: []Arg{{IsVariable: true, VarName: "err"}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"err"}, RHSCall: &CallExpr{Receiver: "repo", Function: "Save"}, ErrorVarName: "err"},
				},
			},
			wantHit: true,
		},
		{
			name: "NoError + Equal on result — NOT detected",
			tf: &TestFunc{
				Name:       "TestGetUser",
				Line:       30,
				HasBody:    true,
				BodyLength: 4,
				CallExprs: []CallExpr{
					{Receiver: "service", Function: "GetUser", FullName: "service.GetUser"},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError",
						Args: []Arg{{IsVariable: true, VarName: "err"}}},
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal",
						Args: []Arg{{Value: "t"}, {Value: `"Alice"`}, {IsVariable: true, VarName: "user.Name"}}},
				},
			},
			wantHit: false,
		},
		{
			name: "no assertions at all — skip (handled by no-assertion)",
			tf: &TestFunc{
				Name:       "TestNoAssertions",
				Line:       40,
				HasBody:    true,
				BodyLength: 1,
				CallExprs: []CallExpr{
					{Receiver: "fmt", Function: "Println", FullName: "fmt.Println"},
				},
			},
			wantHit: false,
		},
		{
			name: "empty test — skip",
			tf: &TestFunc{
				Name:       "TestEmpty",
				Line:       50,
				HasBody:    false,
				BodyLength: 0,
			},
			wantHit: false,
		},
		{
			name: "assert.Nil on err variable — detected",
			tf: &TestFunc{
				Name:       "TestWithNilCheck",
				Line:       60,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "", Function: "DoWork", FullName: "DoWork"},
					{Receiver: "assert", Function: "Nil", FullName: "assert.Nil",
						Args: []Arg{{Value: "t"}, {IsVariable: true, VarName: "err"}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"_", "err"}, RHSCall: &CallExpr{Function: "DoWork"}, ErrorVarName: "err"},
				},
			},
			wantHit: true,
		},
		{
			name: "multiple NoError calls — still detected",
			tf: &TestFunc{
				Name:       "TestMultipleErrors",
				Line:       70,
				HasBody:    true,
				BodyLength: 4,
				CallExprs: []CallExpr{
					{Receiver: "svc", Function: "Step1", FullName: "svc.Step1"},
					{Receiver: "require", Function: "NoError", FullName: "require.NoError",
						Args: []Arg{{IsVariable: true, VarName: "err"}}},
					{Receiver: "svc", Function: "Step2", FullName: "svc.Step2"},
					{Receiver: "require", Function: "NoError", FullName: "require.NoError",
						Args: []Arg{{IsVariable: true, VarName: "err"}}},
				},
			},
			wantHit: true,
		},
		{
			name: "assert.NotNil on result — NOT detected (checks result)",
			tf: &TestFunc{
				Name:       "TestChecksResult",
				Line:       80,
				HasBody:    true,
				BodyLength: 4,
				CallExprs: []CallExpr{
					{Receiver: "svc", Function: "Create", FullName: "svc.Create"},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError",
						Args: []Arg{{IsVariable: true, VarName: "err"}}},
					{Receiver: "assert", Function: "NotNil", FullName: "assert.NotNil",
						Args: []Arg{{Value: "t"}, {IsVariable: true, VarName: "result"}}},
				},
			},
			wantHit: false,
		},
	}

	rule := &OnlyNilCheckRule{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AnalysisContext{File: "svc_test.go", TestFunc: tt.tf}
			findings := rule.Analyze(ctx)
			if tt.wantHit && len(findings) == 0 {
				t.Errorf("expected finding but got none")
			}
			if !tt.wantHit && len(findings) > 0 {
				t.Errorf("expected no finding but got: %v", findings)
			}
		})
	}
}

func TestOnlyNilCheckRule_Metadata(t *testing.T) {
	rule := &OnlyNilCheckRule{}
	if rule.ID() != "only-nil-check" {
		t.Errorf("expected ID 'only-nil-check', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP1 {
		t.Errorf("expected severity P1, got %v", rule.Severity())
	}
}
