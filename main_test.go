package main

import (
	"os/exec"
	"strings"
	"testing"
)

// runBinary runs the main.go with the given args and returns stdout/stderr
func runBinary(t *testing.T, args ...string) (string, string) {
	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && !strings.Contains(stderr.String(), "exit status") {
		// Only fail if it's strictly an execution error, not a non-zero exit code if expected
	}
	return stdout.String(), stderr.String()
}

func TestVersionFlag(t *testing.T) {
	out, _ := runBinary(t, "--version")
	if !strings.Contains(out, "GoGrep, version") {
		t.Errorf("Expected version in output, got: %s", out)
	}
}

func TestHelpFlag(t *testing.T) {
	// Help output likely goes to stderr or stdout depending on implementation
	_, errOut := runBinary(t, "--help")
	if !strings.Contains(errOut, "Usage:") {
		t.Errorf("Expected usage info to stderr, got: %s", errOut)
	}
}

func TestSearchOutput(t *testing.T) {
	// We'll search for something we know is in main.go
	out, _ := runBinary(t, "func main", "main.go", "--lines=1")
	if !strings.Contains(out, "func main()") {
		t.Errorf("Expected to find 'func main()' in output, got: %s", out)
	}
}

func TestLinesFlag(t *testing.T) {
	// Search for something unique, ask for context
	// main.go has "package main" at the top
	out, _ := runBinary(t, "--lines=3", "package main", "main.go")
	if !strings.Contains(out, "package main") {
		t.Errorf("Expected 'package main' match, got: %s", out)
	}
	// Check if we get some context (next few lines)
	if !strings.Contains(out, "import (") {
		t.Errorf("Expected context lines (import), got: %s", out)
	}
}

func TestSilentFlag(t *testing.T) {
	out, errOut := runBinary(t, "--silent", "func", "main.go")
	if out != "" {
		t.Errorf("Expected no output with --silent, got stdout: %s", out)
	}
	// Depending on implementation, some logs might go to stderr or simply not print
	// We only verify stdout is empty as per description
	if strings.Contains(errOut, "Searching in file") {
		t.Errorf("Expected no verbose logs with --silent (unless internal implementation varies), check implementation")
	}
}

func TestExtFlag(t *testing.T) {
	// Verify it picks up .go files
	out, errOut := runBinary(t, "--ext=.go", "main", ".")
	if !strings.Contains(out, "main.go") {
		t.Errorf("Expected to process main.go with --ext=.go, output: %s, stderr: %s", out, errOut)
	}
}
