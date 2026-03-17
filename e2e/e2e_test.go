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

// binaryPath returns the path to the ut-vet binary built for E2E tests.
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

// --- Exit Code Tests ---

func TestE2E_ExitCode1_WhenIssuesFound(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, fixtureDir(t))
	if exitCode != 1 {
		t.Errorf("expected exit code 1 (issues found), got %d\nstdout: %s", exitCode, stdout)
	}
}

func TestE2E_ExitCode0_WhenNoIssues(t *testing.T) {
	dir := t.TempDir()
	cleanTest := filepath.Join(dir, "clean_test.go")
	err := os.WriteFile(cleanTest, []byte(`package example

import "testing"

func TestClean(t *testing.T) {
	result := 1 + 2
	if result != 3 {
		t.Errorf("expected 3, got %d", result)
	}
}
`), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, _, exitCode := runUTVet(t, dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (no issues), got %d", exitCode)
	}
}

// --- Text Output Tests ---

func TestE2E_TextOutput_ContainsExpectedRules(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	expectedRules := []string{"no-assertion", "empty-test", "log-only-test", "trivial-assertion"}
	for _, rule := range expectedRules {
		if !strings.Contains(stdout, "["+rule+"]") {
			t.Errorf("expected rule [%s] in text output, not found.\nOutput:\n%s", rule, stdout)
		}
	}
}

func TestE2E_TextOutput_ContainsExpectedTestNames(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	expectedTests := []string{
		"TestNoAssertion",
		"TestEmptyBody",
		"TestOnlyComments",
		"TestLogOnly",
		"TestTrivialTrue",
		"TestTrivialEqualLiterals",
		"TestFmtPrintOnly",
		"TestLogPrintOnly",
		"TestTLogOnly",
	}
	for _, name := range expectedTests {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected test %s in output, not found.\nOutput:\n%s", name, stdout)
		}
	}
}

func TestE2E_TextOutput_DoesNotFlagCleanTests(t *testing.T) {
	stdout, _, _ := runUTVet(t, fixtureDir(t))

	cleanTests := []string{
		"TestWithAssertion",
		"TestWithFatal",
		"TestWithFail",
		"TestLogWithAssertion",
		"TestRealAssertTrue",
		"TestRealAssertEqual",
	}
	for _, name := range cleanTests {
		if strings.Contains(stdout, name) {
			t.Errorf("clean test %s should NOT appear in findings.\nOutput:\n%s", name, stdout)
		}
	}
}

// --- JSON Output Tests ---

func TestE2E_JSONOutput_ValidJSON(t *testing.T) {
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
		t.Fatalf("invalid JSON output: %v\nRaw output:\n%s", err, stdout)
	}

	if result.Total == 0 {
		t.Error("expected total > 0 in JSON output")
	}
	if len(result.Findings) != result.Total {
		t.Errorf("findings count (%d) != total (%d)", len(result.Findings), result.Total)
	}

	for i, f := range result.Findings {
		if f.File == "" {
			t.Errorf("finding[%d]: missing file", i)
		}
		if f.Line == 0 {
			t.Errorf("finding[%d]: missing line", i)
		}
		if f.Rule == "" {
			t.Errorf("finding[%d]: missing rule", i)
		}
		if f.Message == "" {
			t.Errorf("finding[%d]: missing message", i)
		}
		if f.TestName == "" {
			t.Errorf("finding[%d]: missing test_name", i)
		}
	}
}

func TestE2E_JSONOutput_EmptyWhenClean(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "ok_test.go"), []byte(`package example

import "testing"

func TestOK(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("math is broken")
	}
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runUTVet(t, "--format", "json", dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	var result struct {
		Findings []interface{} `json:"findings"`
		Total    int           `json:"total"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

// --- Verbose Mode ---

func TestE2E_VerboseMode(t *testing.T) {
	stdout, stderr, _ := runUTVet(t, "-v", fixtureDir(t))

	if !strings.Contains(stderr, "analyzing") {
		t.Errorf("expected verbose info on stderr, got: %q", stderr)
	}
	if !strings.Contains(stdout, "issue(s) found") {
		t.Errorf("expected summary in verbose output, got:\n%s", stdout)
	}
}

// --- Rule Filtering ---

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

// --- Exclude Pattern ---

func TestE2E_ExcludePattern(t *testing.T) {
	stdout, _, _ := runUTVet(t, "--exclude", "trivial_*", fixtureDir(t))

	if strings.Contains(stdout, "trivial_test.go") {
		t.Error("trivial_test.go should be excluded")
	}
	if !strings.Contains(stdout, "basic_test.go") {
		t.Error("basic_test.go should still be present")
	}
}

// --- Version and List ---

func TestE2E_VersionFlag(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--version")
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "ut-vet") {
		t.Errorf("expected version output, got: %q", stdout)
	}
}

func TestE2E_ListRulesFlag(t *testing.T) {
	stdout, _, exitCode := runUTVet(t, "--list-rules")
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	expectedRules := []string{"empty-test", "no-assertion", "log-only-test", "trivial-assertion"}
	for _, rule := range expectedRules {
		if !strings.Contains(stdout, rule) {
			t.Errorf("expected rule %q in list output, not found.\nOutput:\n%s", rule, stdout)
		}
	}
}

// --- Edge Cases ---

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
	err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

func main() {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, _, exitCode := runUTVet(t, dir)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for non-test file, got %d", exitCode)
	}
}

// --- Specific Rule Detection E2E ---

func TestE2E_DetectsEmptyTest(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte(`package a

import "testing"

func TestEmpty(t *testing.T) {}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runUTVet(t, dir)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout, "[empty-test]") {
		t.Errorf("expected [empty-test] finding, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "TestEmpty") {
		t.Errorf("expected TestEmpty in output, got:\n%s", stdout)
	}
}

func TestE2E_DetectsNoAssertion(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte(`package a

import "testing"

func TestNoAssert(t *testing.T) {
	x := 1 + 2
	_ = x
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runUTVet(t, dir)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout, "[no-assertion]") {
		t.Errorf("expected [no-assertion], got:\n%s", stdout)
	}
}

func TestE2E_DetectsLogOnly(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte(`package a

import "testing"

func TestLogOnly(t *testing.T) {
	t.Log("hello")
	t.Logf("world %d", 42)
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runUTVet(t, dir)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout, "[log-only-test]") {
		t.Errorf("expected [log-only-test], got:\n%s", stdout)
	}
}

func TestE2E_DetectsTrivialAssertion(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte(`package a

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestTrivial(t *testing.T) {
	assert.True(t, true)
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runUTVet(t, dir)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout, "[trivial-assertion]") {
		t.Errorf("expected [trivial-assertion], got:\n%s", stdout)
	}
}
