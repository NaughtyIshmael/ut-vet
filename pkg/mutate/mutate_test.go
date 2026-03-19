package mutate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTool_GoProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	tool := DetectTool(dir)
	if tool != "gremlins" {
		t.Errorf("expected gremlins, got %s", tool)
	}
}

func TestDetectTool_RustProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)

	tool := DetectTool(dir)
	if tool != "cargo-mutants" {
		t.Errorf("expected cargo-mutants, got %s", tool)
	}
}

func TestDetectTool_DefaultsToGremlins(t *testing.T) {
	dir := t.TempDir()
	tool := DetectTool(dir)
	if tool != "gremlins" {
		t.Errorf("expected gremlins default, got %s", tool)
	}
}

func TestParseGremlinsJSON(t *testing.T) {
	jsonData := `{
		"go_module": "github.com/example/project",
		"test_efficacy": 80.00,
		"mutations_coverage": 75.00,
		"mutants_total": 20,
		"mutants_killed": 16,
		"mutants_lived": 4,
		"mutants_not_viable": 0,
		"mutants_not_covered": 0,
		"files": [
			{
				"file_name": "handler.go",
				"mutations": [
					{"line": 10, "column": 5, "type": "CONDITIONALS_NEGATION", "status": "KILLED"},
					{"line": 20, "column": 8, "type": "ARITHMETIC_BASE", "status": "LIVED"}
				]
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-gremlins-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte(jsonData))
	tmpFile.Close()

	result, err := ParseGremlinsJSON(tmpFile.Name(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tool != "gremlins" {
		t.Errorf("tool: got %s, want gremlins", result.Tool)
	}
	if result.Score != 0.8 {
		t.Errorf("score: got %f, want 0.8", result.Score)
	}
	if result.Total != 20 {
		t.Errorf("total: got %d, want 20", result.Total)
	}
	if result.Killed != 16 {
		t.Errorf("killed: got %d, want 16", result.Killed)
	}
	if result.Survived != 4 {
		t.Errorf("survived: got %d, want 4", result.Survived)
	}
	if len(result.Mutants) != 2 {
		t.Fatalf("mutants: got %d, want 2", len(result.Mutants))
	}
	if result.Mutants[0].Status != "killed" {
		t.Errorf("mutant[0] status: got %s, want killed", result.Mutants[0].Status)
	}
	if result.Mutants[1].Status != "survived" {
		t.Errorf("mutant[1] status: got %s, want survived", result.Mutants[1].Status)
	}
}

func TestParseMutantLine(t *testing.T) {
	tests := []struct {
		line     string
		wantFile string
		wantLine int
		wantFunc string
	}{
		{
			line:     "src/lib.rs:42: replace parse_config with Default::default()",
			wantFile: "src/lib.rs",
			wantLine: 42,
			wantFunc: "parse_config",
		},
		{
			line:     "src/main.rs:10: replace add -> i32 with 0",
			wantFile: "src/main.rs",
			wantLine: 10,
			wantFunc: "add",
		},
		{
			line:     "unparseable line",
			wantFile: "",
			wantLine: 0,
			wantFunc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			m := ParseMutantLine(tt.line)
			if m.File != tt.wantFile {
				t.Errorf("file: got %q, want %q", m.File, tt.wantFile)
			}
			if m.Line != tt.wantLine {
				t.Errorf("line: got %d, want %d", m.Line, tt.wantLine)
			}
			if m.Function != tt.wantFunc {
				t.Errorf("function: got %q, want %q", m.Function, tt.wantFunc)
			}
		})
	}
}

func TestParseCargoMutantsOutput(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "caught.txt"), []byte(
		"src/lib.rs:10: replace add with Default::default()\n"+
			"src/lib.rs:20: replace sub with Default::default()\n",
	), 0644)

	os.WriteFile(filepath.Join(dir, "missed.txt"), []byte(
		"src/lib.rs:30: replace mul with Default::default()\n",
	), 0644)

	os.WriteFile(filepath.Join(dir, "timeout.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "unviable.txt"), []byte(""), 0644)

	result, err := ParseCargoMutantsOutput(dir, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("total: got %d, want 3", result.Total)
	}
	if result.Killed != 2 {
		t.Errorf("killed: got %d, want 2", result.Killed)
	}
	if result.Survived != 1 {
		t.Errorf("survived: got %d, want 1", result.Survived)
	}
	expectedScore := 2.0 / 3.0
	if result.Score < expectedScore-0.01 || result.Score > expectedScore+0.01 {
		t.Errorf("score: got %f, want ~%f", result.Score, expectedScore)
	}
}

func TestFormatText(t *testing.T) {
	r := &MutationResult{
		Tool:      "gremlins",
		Language:  "go",
		Directory: "/test",
		Score:     0.75,
		Total:     4,
		Killed:    3,
		Survived:  1,
		Mutants: []Mutant{
			{File: "handler.go", Line: 10, Mutation: "CONDITIONALS_NEGATION", Status: "killed"},
			{File: "handler.go", Line: 20, Mutation: "ARITHMETIC_BASE", Status: "survived"},
		},
	}

	output := FormatText(r, false)

	if !contains(output, "75.0%") {
		t.Errorf("expected score in output:\n%s", output)
	}
	if !contains(output, "Survived mutants") {
		t.Errorf("expected survived section:\n%s", output)
	}
	if !contains(output, "handler.go:20") {
		t.Errorf("expected survived mutant file:line:\n%s", output)
	}
	// Non-verbose: should NOT show killed
	if contains(output, "Killed mutants") {
		t.Errorf("non-verbose should not show killed mutants:\n%s", output)
	}
}

func TestFormatText_AllKilled(t *testing.T) {
	r := &MutationResult{
		Tool:     "gremlins",
		Language: "go",
		Score:    1.0,
		Total:    5,
		Killed:   5,
		Survived: 0,
	}

	output := FormatText(r, false)

	if !contains(output, "All mutants were caught") {
		t.Errorf("expected success message:\n%s", output)
	}
}

func TestFormatJSON(t *testing.T) {
	r := &MutationResult{
		Tool:     "gremlins",
		Language: "go",
		Score:    0.8,
		Total:    10,
		Killed:   8,
		Survived: 2,
	}

	output, err := FormatJSON(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(output, `"score": 0.8`) {
		t.Errorf("expected score in JSON:\n%s", output)
	}
	if !contains(output, `"killed": 8`) {
		t.Errorf("expected killed count in JSON:\n%s", output)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
