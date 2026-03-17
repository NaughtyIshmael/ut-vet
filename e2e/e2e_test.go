package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Helpers ---

func binaryPath(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("UT_VET_BIN")
	if bin == "" {
		t.Fatal("UT_VET_BIN env var not set — run 'make build' first or set UT_VET_BIN")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("ut-vet binary not found at %s: %v", bin, err)
	}
	return bin
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "pkg", "testdata", "go")
}

func runUTVet(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath(t), args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run ut-vet: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func writeTempTest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// --- Exit Code Tests ---

func TestE2E_ExitCode1_WhenIssuesFound(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, fixtureDir(t))
	if exitCode != 1 {
		t.Errorf("expected exit code 1 (issues found), got %d\nstdout: %s", exitCode, stdout)
	}
}

func TestE2E_ExitCode0_WhenNoIssues(t *testing.T) {
	dir := writeTempTest(t, `package example
import "testing"
func TestClean(t *testing.T) {
	if 1+1 != 2 { t.Error("broken") }
}
`)
	_, _, exitCode := runUTVet(t, dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// --- Text Output Against Fixtures ---

func TestE2E_Fixtures_P0RulesDetected(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	// All P0 rules should fire on the fixture files
	for _, rule := range []string{"no-assertion", "empty-test", "log-only-test", "trivial-assertion"} {
		if !strings.Contains(stdout, "["+rule+"]") {
			t.Errorf("expected rule [%s] in output, not found.\nOutput:\n%s", rule, stdout)
		}
	}
}

func TestE2E_Fixtures_BadTestsDetected(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	// Tests from basic_test.go
	for _, name := range []string{"TestNoAssertion", "TestEmptyBody", "TestOnlyComments", "TestLogOnly"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %s in output, not found", name)
		}
	}

	// Tests from trivial_test.go
	for _, name := range []string{"TestTrivialTrue", "TestTrivialEqualLiterals", "TestTrivialEqualStrings", "TestTrivialFalse"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %s in output, not found", name)
		}
	}

	// Tests from log_only_test.go
	for _, name := range []string{"TestFmtPrintOnly", "TestLogPrintOnly", "TestTLogOnly"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %s in output, not found", name)
		}
	}
}

func TestE2E_Fixtures_CleanTestsNotFlagged(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	// Tests from basic_test.go that are clean
	for _, name := range []string{"TestWithAssertion", "TestWithFatal", "TestWithFail"} {
		if strings.Contains(stdout, name) {
			t.Errorf("clean test %s should NOT appear in findings", name)
		}
	}

	// Tests from log_only_test.go that are clean
	if strings.Contains(stdout, "TestLogWithAssertion") {
		t.Error("TestLogWithAssertion should NOT appear — it has assertion")
	}

	// Tests from clean_test.go
	for _, name := range []string{"TestAddition", "TestSubtraction"} {
		if strings.Contains(stdout, name) {
			t.Errorf("clean test %s should NOT appear in findings", name)
		}
	}
}

func TestE2E_Fixtures_P1RulesDetected(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--severity", "p1", fixtureDir(t))
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Tests from p1_test.go
	if !strings.Contains(stdout, "[error-not-checked]") {
		t.Errorf("expected [error-not-checked] from p1_test.go fixtures.\nOutput:\n%s", stdout)
	}
	if !strings.Contains(stdout, "TestSaveIgnoreError") {
		t.Errorf("expected TestSaveIgnoreError in output")
	}
	if !strings.Contains(stdout, "[zero-value-input]") {
		t.Errorf("expected [zero-value-input] from p1_test.go fixtures.\nOutput:\n%s", stdout)
	}
	if !strings.Contains(stdout, "TestCreateUserZero") {
		t.Errorf("expected TestCreateUserZero in output")
	}
	if !strings.Contains(stdout, "[no-code-under-test]") {
		t.Errorf("expected [no-code-under-test] from fixtures.\nOutput:\n%s", stdout)
	}
}

func TestE2E_Fixtures_P1CleanTestsNotFlagged(t *testing.T) {
	stdout, _, _ := runUTVet(t, "--severity", "p1", fixtureDir(t))

	// Tests from p1_test.go that are clean
	if strings.Contains(stdout, "TestSaveChecked") {
		t.Error("TestSaveChecked should NOT appear — error is checked")
	}
	if strings.Contains(stdout, "TestCreateUserReal") {
		t.Error("TestCreateUserReal should NOT appear — has meaningful inputs")
	}
}

func TestE2E_Fixtures_P1RulesNotShownAtP0Severity(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	// Default severity is P0, so P1 rules should NOT appear
	for _, rule := range []string{"[error-not-checked]", "[zero-value-input]"} {
		if strings.Contains(stdout, rule) {
			t.Errorf("%s should NOT appear at default P0 severity.\nOutput:\n%s", rule, stdout)
		}
	}
}

// --- JSON Output ---

func TestE2E_Fixtures_JSONOutput(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--format", "json", fixtureDir(t))
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	var result struct {
		Findings []struct {
			File     string `json:"file"`
			Line     int    `json:"line"`
			Rule     string `json:"rule"`
			Message  string `json:"message"`
			Severity int    `json:"severity"`
			TestName string `json:"test_name"`
		} `json:"findings"`
		Total int `json:"total"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nRaw:\n%s", err, stdout)
	}

	if result.Total == 0 {
		t.Error("expected total > 0")
	}
	if len(result.Findings) != result.Total {
		t.Errorf("findings count (%d) != total (%d)", len(result.Findings), result.Total)
	}

	// Every finding must have all fields populated
	for i, f := range result.Findings {
		if f.File == "" || f.Line == 0 || f.Rule == "" || f.Message == "" || f.TestName == "" {
			t.Errorf("finding[%d] has empty fields: %+v", i, f)
		}
	}
}

func TestE2E_JSONOutput_EmptyWhenClean(t *testing.T) {
	dir := writeTempTest(t, `package a
import "testing"
func TestOK(t *testing.T) {
	if 1+1 != 2 { t.Fatal("broken") }
}
`)
	stdout, _, exitCode := runUTVet(t, "--format", "json", dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

// --- CLI Flag Tests (use fixtures) ---

func TestE2E_VerboseMode(t *testing.T) {
	stdout, stderr, _ := runUTVet(t, "-v", fixtureDir(t))

	if !strings.Contains(stderr, "analyzing") {
		t.Errorf("expected verbose info on stderr, got: %q", stderr)
	}
	if !strings.Contains(stdout, "issue(s) found") {
		t.Errorf("expected summary in verbose output")
	}
}

func TestE2E_RuleFilter_SingleRule(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--rules", "empty-test", fixtureDir(t))
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if !strings.Contains(stdout, "[empty-test]") {
		t.Error("expected [empty-test] findings")
	}
	for _, rule := range []string{"[no-assertion]", "[log-only-test]", "[trivial-assertion]"} {
		if strings.Contains(stdout, rule) {
			t.Errorf("should NOT contain %s when filtered to empty-test only", rule)
		}
	}
}

func TestE2E_RuleFilter_MultipleRules(t *testing.T) {
	stdout, _, _ := runUTVet(t, "--rules", "empty-test,no-assertion", fixtureDir(t))

	if !strings.Contains(stdout, "[empty-test]") {
		t.Error("expected [empty-test]")
	}
	if !strings.Contains(stdout, "[no-assertion]") {
		t.Error("expected [no-assertion]")
	}
	if strings.Contains(stdout, "[trivial-assertion]") {
		t.Error("should NOT contain [trivial-assertion]")
	}
}

func TestE2E_ExcludePattern(t *testing.T) {
	stdout, _, _ := runUTVet(t, "--exclude", "trivial_*", fixtureDir(t))

	if strings.Contains(stdout, "trivial_test.go") {
		t.Error("trivial_test.go should be excluded")
	}
	if !strings.Contains(stdout, "basic_test.go") {
		t.Error("basic_test.go should still be present")
	}
}

func TestE2E_VersionFlag(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--version")
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "ut-vet") {
		t.Errorf("expected version string, got: %q", stdout)
	}
}

func TestE2E_ListRulesFlag(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--list-rules")
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	for _, rule := range []string{"empty-test", "no-assertion", "log-only-test", "trivial-assertion", "error-not-checked", "no-code-under-test", "zero-value-input"} {
		if !strings.Contains(stdout, rule) {
			t.Errorf("expected rule %q in list, not found", rule)
		}
	}
}

// --- Edge Cases (require temp dirs) ---

func TestE2E_NonExistentPath(t *testing.T) {
	_, _, exitCode := runUTVet(t, "/nonexistent/path")
	if exitCode != 2 {
		t.Errorf("expected exit code 2 (tool error), got %d", exitCode)
	}
}

func TestE2E_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, _, exitCode := runUTVet(t, dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for empty directory, got %d", exitCode)
	}
}

func TestE2E_NonTestGoFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)

	_, _, exitCode := runUTVet(t, dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for non-test file, got %d", exitCode)
	}
}
