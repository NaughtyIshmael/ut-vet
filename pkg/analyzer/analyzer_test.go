package analyzer

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

func TestAnalyzer_BasicFixtures(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	fixtureDir := filepath.Join(filepath.Dir(thisFile), "..", "testdata", "go")

	a := NewAnalyzer(DefaultRules())
	findings, err := a.AnalyzePaths([]string{fixtureDir})
	if err != nil {
		t.Fatalf("AnalyzePaths failed: %v", err)
	}

	// Verify specific expected findings
	ruleCount := make(map[string]int)
	for _, f := range findings {
		ruleCount[f.Rule]++
	}

	// basic_test.go: TestEmptyBody, TestOnlyComments = 2 empty-test
	if ruleCount["empty-test"] < 2 {
		t.Errorf("expected at least 2 empty-test findings, got %d", ruleCount["empty-test"])
	}

	// basic_test.go: TestNoAssertion + log_only_test.go tests + trivial_test.go: TestImportButNoAssert
	if ruleCount["no-assertion"] < 2 {
		t.Errorf("expected at least 2 no-assertion findings, got %d", ruleCount["no-assertion"])
	}

	// trivial_test.go: TestTrivialTrue, TestTrivialEqualLiterals, TestTrivialEqualStrings, TestTrivialFalse
	if ruleCount["trivial-assertion"] < 4 {
		t.Errorf("expected at least 4 trivial-assertion findings, got %d", ruleCount["trivial-assertion"])
	}

	// basic_test.go: TestLogOnly + log_only_test.go: 3 tests
	if ruleCount["log-only-test"] < 3 {
		t.Errorf("expected at least 3 log-only-test findings, got %d", ruleCount["log-only-test"])
	}
}

func TestAnalyzer_NoFindingsOnCleanCode(t *testing.T) {
	src := `package example

import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2)
	if result != 3 {
		t.Errorf("Add(1, 2) = %d, want 3", result)
	}
}

func TestSubtract(t *testing.T) {
	result := Subtract(5, 3)
	if result != 2 {
		t.Fatalf("Subtract(5, 3) = %d, want 2", result)
	}
}
`
	a := NewAnalyzer(DefaultRules())
	testFuncs, err := ParseGoTestFile("clean_test.go", []byte(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	for _, tf := range testFuncs {
		for _, rule := range a.Rules {
			ctx := &rules.AnalysisContext{File: "clean_test.go", TestFunc: tf}
			findings := rule.Analyze(ctx)
			if len(findings) > 0 {
				t.Errorf("clean test %s triggered rule %s: %s", tf.Name, findings[0].Rule, findings[0].Message)
			}
		}
	}
}

func TestAnalyzer_RuleFiltering(t *testing.T) {
	allRules := DefaultRules()
	if len(allRules) != 4 {
		t.Fatalf("expected 4 default rules, got %d", len(allRules))
	}

	// Test with only one rule
	a := NewAnalyzer(allRules[:1])
	if len(a.Rules) != 1 {
		t.Errorf("expected 1 rule after filtering, got %d", len(a.Rules))
	}
}

func TestAnalyzer_ExcludePattern(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	fixtureDir := filepath.Join(filepath.Dir(thisFile), "..", "testdata", "go")

	a := NewAnalyzer(DefaultRules())
	a.Exclude = []string{"trivial_*"}

	findings, err := a.AnalyzePaths([]string{fixtureDir})
	if err != nil {
		t.Fatalf("AnalyzePaths failed: %v", err)
	}

	// Should have no trivial-assertion findings from trivial_test.go
	for _, f := range findings {
		if filepath.Base(f.File) == "trivial_test.go" {
			t.Errorf("expected trivial_test.go to be excluded, but found: %s", f)
		}
	}
}
