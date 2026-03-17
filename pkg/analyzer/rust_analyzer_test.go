package analyzer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

func rustFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata", "rust", name)
}

func TestParseRustTestFile_BasicTests(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	// We expect at least these test functions
	names := make(map[string]bool)
	for _, f := range funcs {
		names[f.Name] = true
	}

	expected := []string{
		"test_empty",
		"test_only_comments",
		"test_no_assertion",
		"test_log_only",
		"test_trivial_true",
		"test_trivial_eq",
		"test_good_assertion",
		"test_good_assert",
		"test_tautological",
		"test_panic_not_assert",
		"test_should_panic",
		"test_async_good",
		"test_async_no_assert",
		"test_zero_value",
		"test_meaningful_inputs",
		"test_error_swallowed",
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected test function %q not found. Found: %v", name, namesSlice(funcs))
		}
	}
}

func TestParseRustTestFile_EmptyBody(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	emptyTest := findTestFunc(funcs, "test_empty")
	if emptyTest == nil {
		t.Fatal("test_empty not found")
	}
	if emptyTest.HasBody {
		t.Error("test_empty should have HasBody=false (empty)")
	}
	if emptyTest.BodyLength != 0 {
		t.Errorf("test_empty BodyLength should be 0, got %d", emptyTest.BodyLength)
	}
}

func TestParseRustTestFile_CommentsOnly(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	commentsTest := findTestFunc(funcs, "test_only_comments")
	if commentsTest == nil {
		t.Fatal("test_only_comments not found")
	}
	if commentsTest.HasBody {
		t.Error("test_only_comments should have HasBody=false (comments only)")
	}
}

func TestParseRustTestFile_AssertionDetection(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	// test_good_assertion should have assertion calls
	goodTest := findTestFunc(funcs, "test_good_assertion")
	if goodTest == nil {
		t.Fatal("test_good_assertion not found")
	}

	hasAssert := false
	for _, ce := range goodTest.CallExprs {
		if ce.Function == "assert_eq!" {
			hasAssert = true
		}
	}
	if !hasAssert {
		t.Error("test_good_assertion should have assert_eq! call")
	}

	// test_no_assertion should NOT have assertion calls
	noAssertTest := findTestFunc(funcs, "test_no_assertion")
	if noAssertTest == nil {
		t.Fatal("test_no_assertion not found")
	}
	for _, ce := range noAssertTest.CallExprs {
		if ce.Function == "assert_eq!" || ce.Function == "assert!" || ce.Function == "assert_ne!" {
			t.Errorf("test_no_assertion should not have assertion call, found %s", ce.Function)
		}
	}
}

func TestParseRustTestFile_LogDetection(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	logTest := findTestFunc(funcs, "test_log_only")
	if logTest == nil {
		t.Fatal("test_log_only not found")
	}

	hasLog := false
	for _, ce := range logTest.CallExprs {
		if ce.Function == "println!" {
			hasLog = true
		}
	}
	if !hasLog {
		t.Error("test_log_only should have println! call")
	}
}

func TestParseRustTestFile_TrivialAssertArgs(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	// assert!(true) — single literal arg
	trivTest := findTestFunc(funcs, "test_trivial_true")
	if trivTest == nil {
		t.Fatal("test_trivial_true not found")
	}
	found := false
	for _, ce := range trivTest.CallExprs {
		if ce.Function == "assert!" && len(ce.Args) == 1 && ce.Args[0].IsLiteral && ce.Args[0].Value == "true" {
			found = true
		}
	}
	if !found {
		t.Error("test_trivial_true should have assert!(true) with literal arg")
	}

	// assert_eq!(1, 1) — two identical literal args
	eqTest := findTestFunc(funcs, "test_trivial_eq")
	if eqTest == nil {
		t.Fatal("test_trivial_eq not found")
	}
	foundEq := false
	for _, ce := range eqTest.CallExprs {
		if ce.Function == "assert_eq!" && len(ce.Args) == 2 {
			if ce.Args[0].IsLiteral && ce.Args[1].IsLiteral {
				foundEq = true
			}
		}
	}
	if !foundEq {
		t.Error("test_trivial_eq should have assert_eq!(1, 1) with two literal args")
	}
}

func TestParseRustTestFile_ShouldPanicAttribute(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	// test_should_panic should have should_panic flag set
	panicTest := findTestFunc(funcs, "test_should_panic")
	if panicTest == nil {
		t.Fatal("test_should_panic not found")
	}

	// The #[should_panic] test should be treated as having assertions
	// (the panic IS the assertion), so rules shouldn't flag it
	hasAssert := false
	for _, ce := range panicTest.CallExprs {
		if isRustAssertionCall(ce) {
			hasAssert = true
		}
	}
	// We inject a synthetic assertion call for should_panic tests
	if !hasAssert {
		t.Error("test_should_panic should have synthetic assertion (should_panic is the assertion)")
	}
}

func TestParseRustTestFile_LocalFuncCalls(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	goodTest := findTestFunc(funcs, "test_good_assertion")
	if goodTest == nil {
		t.Fatal("test_good_assertion not found")
	}

	found := false
	for _, name := range goodTest.LocalFuncCalls {
		if name == "compute" {
			found = true
		}
	}
	if !found {
		t.Error("test_good_assertion should have 'compute' in LocalFuncCalls")
	}
}

func TestParseRustTestFile_TautologicalArgs(t *testing.T) {
	path := rustFixturePath(t, "basic_test.rs")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	funcs, err := ParseRustTestFile(path, src)
	if err != nil {
		t.Fatalf("ParseRustTestFile error: %v", err)
	}

	tautTest := findTestFunc(funcs, "test_tautological")
	if tautTest == nil {
		t.Fatal("test_tautological not found")
	}

	found := false
	for _, ce := range tautTest.CallExprs {
		if ce.Function == "assert_eq!" && len(ce.Args) == 2 {
			if ce.Args[0].IsVariable && ce.Args[1].IsVariable &&
				ce.Args[0].VarName == ce.Args[1].VarName {
				found = true
			}
		}
	}
	if !found {
		t.Error("test_tautological should have assert_eq!(result, result) with matching variable args")
	}
}

func namesSlice(funcs []*rules.TestFunc) []string {
	var names []string
	for _, f := range funcs {
		names = append(names, f.Name)
	}
	return names
}
