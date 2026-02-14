package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewOutput(t *testing.T) {
	out := NewOutput("1.2.3")
	if out == nil {
		t.Fatal("NewOutput returned nil")
	}
	if out.version != "1.2.3" {
		t.Errorf("version = %q, want %q", out.version, "1.2.3")
	}
}

func TestOutputHeader(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Header()

	got := stdout.String()
	want := "[*] goingenv v1.0.0\n"
	if got != want {
		t.Errorf("Header() = %q, want %q", got, want)
	}
}

func TestOutputSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Success("Operation completed")

	got := stdout.String()
	want := "[+] Operation completed\n"
	if got != want {
		t.Errorf("Success() = %q, want %q", got, want)
	}
}

func TestOutputWarning(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Warning("Something might be wrong")

	got := stdout.String()
	want := "[!] Something might be wrong\n"
	if got != want {
		t.Errorf("Warning() = %q, want %q", got, want)
	}
}

func TestOutputError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Error("Something went wrong")

	// Error should go to stderr
	if stdout.Len() != 0 {
		t.Errorf("Error() wrote to stdout: %q", stdout.String())
	}

	got := stderr.String()
	want := "[x] Something went wrong\n"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestOutputAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Action("Processing files")

	got := stdout.String()
	want := "[>] Processing files\n"
	if got != want {
		t.Errorf("Action() = %q, want %q", got, want)
	}
}

func TestOutputHint(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Hint("Try running 'goingenv status'")

	got := stdout.String()
	want := "[?] Try running 'goingenv status'\n"
	if got != want {
		t.Errorf("Hint() = %q, want %q", got, want)
	}
}

func TestOutputListItem(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.ListItem(".env")

	got := stdout.String()
	want := "[-] .env\n"
	if got != want {
		t.Errorf("ListItem() = %q, want %q", got, want)
	}
}

func TestOutputSkipped(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Skipped("Already exists")

	got := stdout.String()
	want := "[~] Already exists\n"
	if got != want {
		t.Errorf("Skipped() = %q, want %q", got, want)
	}
}

func TestOutputSection(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Section("Environment Files")

	got := stdout.String()
	want := "Environment Files\n"
	if got != want {
		t.Errorf("Section() = %q, want %q", got, want)
	}
}

func TestOutputIndent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Indent(".env.local")

	got := stdout.String()
	want := "  .env.local\n"
	if got != want {
		t.Errorf("Indent() = %q, want %q", got, want)
	}
}

func TestOutputBlank(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	out.Blank()

	got := stdout.String()
	want := "\n"
	if got != want {
		t.Errorf("Blank() = %q, want %q", got, want)
	}
}

func TestOutputWithColors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, true, "1.0.0")

	out.Header()

	got := stdout.String()
	// With colors, output should contain ANSI escape codes
	if !strings.Contains(got, "goingenv") {
		t.Errorf("Header() with colors should contain 'goingenv', got %q", got)
	}
	if !strings.Contains(got, "1.0.0") {
		t.Errorf("Header() with colors should contain version, got %q", got)
	}
}

func TestOutputWarningList(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	items := []string{".env", ".env.local", ".env.production"}
	out.WarningList("Files would be overwritten:", items, 0)

	got := stdout.String()
	if !strings.Contains(got, "[!] Files would be overwritten:") {
		t.Errorf("WarningList() should contain warning header, got %q", got)
	}
	if !strings.Contains(got, ".env") {
		t.Errorf("WarningList() should contain items, got %q", got)
	}
}

func TestOutputWarningListWithLimit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	items := []string{".env", ".env.local", ".env.production", ".env.test"}
	out.WarningList("Files:", items, 2)

	got := stdout.String()
	if !strings.Contains(got, "... and 2 more") {
		t.Errorf("WarningList() with limit should show remaining count, got %q", got)
	}
}

func TestOutputTable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	rows := [][]string{
		{".env", "245 bytes"},
		{".env.local", "189 bytes"},
	}
	out.Table(rows)

	got := stdout.String()
	if !strings.Contains(got, ".env") {
		t.Errorf("Table() should contain file names, got %q", got)
	}
	if !strings.Contains(got, "245 bytes") {
		t.Errorf("Table() should contain sizes, got %q", got)
	}
}

func TestGlobalOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriter(&stdout, &stderr, false, "1.0.0")

	SetGlobalOutput(out)
	got := GetGlobalOutput()

	if got != out {
		t.Error("GetGlobalOutput() should return the set output")
	}
}
