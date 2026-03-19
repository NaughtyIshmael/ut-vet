package mutate

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MutationResult represents the outcome of mutation testing.
type MutationResult struct {
	Tool      string   `json:"tool"`
	Language  string   `json:"language"`
	Directory string   `json:"directory"`
	Score     float64  `json:"score"`
	Total     int      `json:"total"`
	Killed    int      `json:"killed"`
	Survived  int      `json:"survived"`
	Timeout   int      `json:"timeout"`
	NotViable int      `json:"not_viable"`
	Mutants   []Mutant `json:"mutants,omitempty"`
}

// Mutant represents a single mutation and its outcome.
type Mutant struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function,omitempty"`
	Mutation string `json:"mutation"`
	Status   string `json:"status"` // killed, survived, timeout, not_viable
}

// Options configures a mutation testing run.
type Options struct {
	Tool      string  // gremlins, cargo-mutants, or "" for auto-detect
	Threshold float64 // minimum mutation score (0.0-1.0)
	Timeout   int     // per-mutant timeout in seconds (0 = tool default)
	Verbose   bool
	JSON      bool
}

// Run executes mutation testing on the given path.
func Run(path string, opts Options) (*MutationResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	tool := opts.Tool
	if tool == "" {
		tool = DetectTool(absPath)
	}

	switch tool {
	case "gremlins":
		return runGremlins(absPath, opts)
	case "cargo-mutants":
		return runCargoMutants(absPath, opts)
	default:
		return nil, fmt.Errorf("unknown mutation tool: %q (use 'gremlins' or 'cargo-mutants')", tool)
	}
}

// DetectTool determines which mutation tool to use based on project files.
func DetectTool(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		return "cargo-mutants"
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return "gremlins"
	}
	parent := filepath.Dir(dir)
	if parent != dir {
		return DetectTool(parent)
	}
	return "gremlins"
}

// CheckToolInstalled returns an error if the tool binary is not found.
func CheckToolInstalled(name string) error {
	binName := name
	if name == "cargo-mutants" {
		binName = "cargo"
	}
	_, err := exec.LookPath(binName)
	if err != nil {
		installHint := ""
		switch name {
		case "gremlins":
			installHint = "\n  Install: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest"
		case "cargo-mutants":
			installHint = "\n  Install: cargo install --locked cargo-mutants"
		}
		return fmt.Errorf("%s not found in PATH.%s", name, installHint)
	}
	return nil
}

// runGremlins runs gremlins and parses results.
func runGremlins(dir string, opts Options) (*MutationResult, error) {
	if err := CheckToolInstalled("gremlins"); err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp("", "ut-vet-gremlins-*.json")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	args := []string{"unleash", "--output", tmpPath}
	if opts.Threshold > 0 {
		args = append(args, fmt.Sprintf("--threshold-efficacy=%.0f", opts.Threshold*100))
	}

	cmd := exec.Command("gremlins", args...)
	cmd.Dir = dir
	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	_ = cmd.Run()

	return ParseGremlinsJSON(tmpPath, dir)
}

// gremlinsOutput represents gremlins JSON output structure.
type gremlinsOutput struct {
	GoModule          string         `json:"go_module"`
	TestEfficacy      float64        `json:"test_efficacy"`
	MutationsCoverage float64        `json:"mutations_coverage"`
	MutantsTotal      int            `json:"mutants_total"`
	MutantsKilled     int            `json:"mutants_killed"`
	MutantsLived      int            `json:"mutants_lived"`
	MutantsNotViable  int            `json:"mutants_not_viable"`
	MutantsNotCovered int            `json:"mutants_not_covered"`
	Files             []gremlinsFile `json:"files"`
}

type gremlinsFile struct {
	FileName  string             `json:"file_name"`
	Mutations []gremlinsMutation `json:"mutations"`
}

type gremlinsMutation struct {
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// ParseGremlinsJSON parses gremlins JSON output into MutationResult.
func ParseGremlinsJSON(path string, dir string) (*MutationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading gremlins output: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("gremlins produced empty output — check that tests build and pass")
	}

	var out gremlinsOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing gremlins JSON: %w", err)
	}

	result := &MutationResult{
		Tool:      "gremlins",
		Language:  "go",
		Directory: dir,
		Score:     out.TestEfficacy / 100.0,
		Total:     out.MutantsTotal,
		Killed:    out.MutantsKilled,
		Survived:  out.MutantsLived,
		NotViable: out.MutantsNotViable,
	}

	for _, f := range out.Files {
		for _, m := range f.Mutations {
			status := normalizeStatus(m.Status)
			result.Mutants = append(result.Mutants, Mutant{
				File:     f.FileName,
				Line:     m.Line,
				Mutation: m.Type,
				Status:   status,
			})
		}
	}

	return result, nil
}

// runCargoMutants runs cargo-mutants and parses results.
func runCargoMutants(dir string, opts Options) (*MutationResult, error) {
	if err := CheckToolInstalled("cargo-mutants"); err != nil {
		return nil, err
	}

	args := []string{"mutants", "--no-shuffle"}
	if opts.Timeout > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", opts.Timeout))
	}

	cmd := exec.Command("cargo", args...)
	cmd.Dir = dir
	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	_ = cmd.Run()

	outDir := filepath.Join(dir, "mutants.out")
	return ParseCargoMutantsOutput(outDir, dir)
}

// ParseCargoMutantsOutput parses cargo-mutants output directory.
func ParseCargoMutantsOutput(outDir string, projectDir string) (*MutationResult, error) {
	result := &MutationResult{
		Tool:      "cargo-mutants",
		Language:  "rust",
		Directory: projectDir,
	}

	statusMap := map[string]string{
		"caught":   "killed",
		"missed":   "survived",
		"timeout":  "timeout",
		"unviable": "not_viable",
	}

	for file, status := range statusMap {
		path := filepath.Join(outDir, file+".txt")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			m := ParseMutantLine(line)
			m.Status = status
			result.Mutants = append(result.Mutants, m)
			result.Total++
			switch status {
			case "killed":
				result.Killed++
			case "survived":
				result.Survived++
			case "timeout":
				result.Timeout++
			case "not_viable":
				result.NotViable++
			}
		}
	}

	if result.Total > 0 {
		viable := result.Killed + result.Survived
		if viable > 0 {
			result.Score = float64(result.Killed) / float64(viable)
		}
	}

	return result, nil
}

// ParseMutantLine parses a line from cargo-mutants output files.
// Format: "src/file.rs:42: replace function_name with Default::default()"
func ParseMutantLine(line string) Mutant {
	m := Mutant{Mutation: line}

	parts := strings.SplitN(line, ":", 3)
	if len(parts) >= 3 {
		m.File = strings.TrimSpace(parts[0])
		var lineNum int
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &lineNum); err == nil {
			m.Line = lineNum
		}
		m.Mutation = strings.TrimSpace(parts[2])

		if strings.HasPrefix(m.Mutation, "replace ") {
			rest := strings.TrimPrefix(m.Mutation, "replace ")
			if idx := strings.Index(rest, " "); idx > 0 {
				m.Function = rest[:idx]
			}
		}
	}

	return m
}

func normalizeStatus(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "killed":
		return "killed"
	case "lived", "survived":
		return "survived"
	case "not_viable", "not viable", "not_covered", "not covered":
		return "not_viable"
	case "timed_out", "timed out", "timeout":
		return "timeout"
	default:
		return s
	}
}

// FormatText formats mutation results as human-readable text.
func FormatText(r *MutationResult, verbose bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "🧬 Mutation Testing Results (%s via %s)\n", r.Language, r.Tool)
	fmt.Fprintf(&b, "   Directory: %s\n", r.Directory)

	viable := r.Killed + r.Survived
	fmt.Fprintf(&b, "   Score: %.1f%% (%d killed / %d viable)\n",
		r.Score*100, r.Killed, viable)
	fmt.Fprintf(&b, "   Total: %d  Killed: %d  Survived: %d",
		r.Total, r.Killed, r.Survived)
	if r.Timeout > 0 {
		fmt.Fprintf(&b, "  Timeout: %d", r.Timeout)
	}
	if r.NotViable > 0 {
		fmt.Fprintf(&b, "  Not Viable: %d", r.NotViable)
	}
	fmt.Fprintln(&b)

	if r.Survived > 0 {
		fmt.Fprintln(&b, "\n⚠️  Survived mutants (tests didn't catch these bugs):")
		for _, m := range r.Mutants {
			if m.Status == "survived" {
				if m.File != "" && m.Line > 0 {
					fmt.Fprintf(&b, "   %s:%d: %s\n", m.File, m.Line, m.Mutation)
				} else {
					fmt.Fprintf(&b, "   %s\n", m.Mutation)
				}
			}
		}
	}

	if verbose && r.Killed > 0 {
		fmt.Fprintln(&b, "\n✅ Killed mutants:")
		for _, m := range r.Mutants {
			if m.Status == "killed" {
				if m.File != "" && m.Line > 0 {
					fmt.Fprintf(&b, "   %s:%d: %s\n", m.File, m.Line, m.Mutation)
				} else {
					fmt.Fprintf(&b, "   %s\n", m.Mutation)
				}
			}
		}
	}

	if r.Survived == 0 && r.Total > 0 {
		fmt.Fprintln(&b, "\n✅ All mutants were caught by your tests!")
	}

	return b.String()
}

// FormatJSON formats mutation results as JSON.
func FormatJSON(r *MutationResult) (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
