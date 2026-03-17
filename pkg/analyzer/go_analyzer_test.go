package analyzer

import (
	"os"
	"testing"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

func TestParseGoTestFile_BasicTests(t *testing.T) {
	src, err := os.ReadFile("../testdata/go/basic_test.go")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	funcs, err := ParseGoTestFile("basic_test.go", src)
	if err != nil {
		t.Fatalf("ParseGoTestFile failed: %v", err)
	}

	// Should find exactly 7 test functions
	if len(funcs) != 7 {
		t.Fatalf("expected 7 test functions, got %d", len(funcs))
	}

	// Verify names
	expectedNames := []string{
		"TestNoAssertion", "TestEmptyBody", "TestOnlyComments",
		"TestLogOnly", "TestWithAssertion", "TestWithFatal", "TestWithFail",
	}
	for i, name := range expectedNames {
		if funcs[i].Name != name {
			t.Errorf("func[%d]: expected name %q, got %q", i, name, funcs[i].Name)
		}
	}
}

func TestParseGoTestFile_EmptyBody(t *testing.T) {
	src, err := os.ReadFile("../testdata/go/basic_test.go")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	funcs, err := ParseGoTestFile("basic_test.go", src)
	if err != nil {
		t.Fatalf("ParseGoTestFile failed: %v", err)
	}

	// TestEmptyBody should have no body
	emptyFunc := findTestFunc(funcs, "TestEmptyBody")
	if emptyFunc == nil {
		t.Fatal("TestEmptyBody not found")
	}
	if emptyFunc.HasBody {
		t.Error("TestEmptyBody: expected HasBody=false")
	}
	if emptyFunc.BodyLength != 0 {
		t.Errorf("TestEmptyBody: expected BodyLength=0, got %d", emptyFunc.BodyLength)
	}
}

func TestParseGoTestFile_OnlyComments(t *testing.T) {
	src := []byte(`package example

import "testing"

func TestOnlyComments(t *testing.T) {
	// TODO: implement this test
	// another comment
}
`)
	funcs, err := ParseGoTestFile("comments_test.go", src)
	if err != nil {
		t.Fatalf("ParseGoTestFile failed: %v", err)
	}

	if len(funcs) != 1 {
		t.Fatalf("expected 1 test function, got %d", len(funcs))
	}

	// Go parser treats lines with only comments as having no statements in the AST
	// So the body might be empty from the AST perspective
	tf := funcs[0]
	if tf.BodyLength != 0 {
		t.Errorf("expected BodyLength=0 for comments-only body, got %d", tf.BodyLength)
	}
}

func TestParseGoTestFile_CallExprs(t *testing.T) {
	src, err := os.ReadFile("../testdata/go/basic_test.go")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	funcs, err := ParseGoTestFile("basic_test.go", src)
	if err != nil {
		t.Fatalf("ParseGoTestFile failed: %v", err)
	}

	// TestWithAssertion should have a t.Errorf call
	withAssert := findTestFunc(funcs, "TestWithAssertion")
	if withAssert == nil {
		t.Fatal("TestWithAssertion not found")
	}

	hasErrorf := false
	for _, call := range withAssert.CallExprs {
		if call.IsTestingT && call.Function == "Errorf" {
			hasErrorf = true
			break
		}
	}
	if !hasErrorf {
		t.Error("TestWithAssertion: expected to find t.Errorf call")
	}

	// TestLogOnly should have t.Log and t.Logf calls but NO assertion calls
	logOnly := findTestFunc(funcs, "TestLogOnly")
	if logOnly == nil {
		t.Fatal("TestLogOnly not found")
	}

	hasLog := false
	hasAssert := false
	for _, call := range logOnly.CallExprs {
		if call.IsTestingT && (call.Function == "Log" || call.Function == "Logf") {
			hasLog = true
		}
		if isAssertionCall(call) {
			hasAssert = true
		}
	}
	if !hasLog {
		t.Error("TestLogOnly: expected to find t.Log/t.Logf calls")
	}
	if hasAssert {
		t.Error("TestLogOnly: should NOT have assertion calls")
	}
}

func TestParseGoTestFile_NotATestFunc(t *testing.T) {
	src := []byte(`package example

import "testing"

func helperFunc() {}

func BenchmarkSomething(b *testing.B) {
	for i := 0; i < b.N; i++ {}
}

func TestReal(t *testing.T) {
	t.Error("fail")
}
`)
	funcs, err := ParseGoTestFile("mixed_test.go", src)
	if err != nil {
		t.Fatalf("ParseGoTestFile failed: %v", err)
	}

	// Should only find TestReal, not helperFunc or BenchmarkSomething
	if len(funcs) != 1 {
		t.Fatalf("expected 1 test function, got %d", len(funcs))
	}
	if funcs[0].Name != "TestReal" {
		t.Errorf("expected TestReal, got %s", funcs[0].Name)
	}
}

func TestParseGoTestFile_TestifyCallExprs(t *testing.T) {
	src := []byte(`package example

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestWithTestify(t *testing.T) {
	x := 42
	assert.Equal(t, 42, x)
}
`)
	funcs, err := ParseGoTestFile("testify_test.go", src)
	if err != nil {
		t.Fatalf("ParseGoTestFile failed: %v", err)
	}

	if len(funcs) != 1 {
		t.Fatalf("expected 1, got %d", len(funcs))
	}

	hasAssertEqual := false
	for _, call := range funcs[0].CallExprs {
		if call.Receiver == "assert" && call.Function == "Equal" {
			hasAssertEqual = true
			// Check args: first is t (variable), second is 42 (literal), third is x (variable)
			if len(call.Args) < 3 {
				t.Errorf("expected 3 args, got %d", len(call.Args))
			} else {
				if !call.Args[1].IsLiteral {
					t.Error("arg[1] should be literal")
				}
				if !call.Args[2].IsVariable {
					t.Error("arg[2] should be variable")
				}
			}
		}
	}
	if !hasAssertEqual {
		t.Error("expected assert.Equal call")
	}
}

// isAssertionCall is a helper for tests — checks if a call is an assertion.
func isAssertionCall(call rules.CallExpr) bool {
	if call.IsTestingT {
		switch call.Function {
		case "Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow":
			return true
		}
	}
	if call.Receiver == "assert" || call.Receiver == "require" {
		return true
	}
	return false
}

func findTestFunc(funcs []*rules.TestFunc, name string) *rules.TestFunc {
	for _, f := range funcs {
		if f.Name == name {
			return f
		}
	}
	return nil
}
