package site_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// llmsPath is the agent-facing summary, served next to the page.
const llmsPath = "../../public/llms.txt"

// repoRoot is where a raw.githubusercontent.com link on main resolves to.
const repoRoot = "../.."

// rawLink matches a raw GitHub link into this repository's main branch and
// captures the path inside it.
var rawLink = regexp.MustCompile(`https://raw\.githubusercontent\.com/spencerjireh/goingenv/main/([^\s)]+)`)

// llmsURL is where the deployed file lives; the agent prompt on the page
// points at it.
const llmsURL = "https://spencerjireh.github.io/goingenv/llms.txt"

func loadLLMs(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(llmsPath)
	if err != nil {
		t.Fatalf("failed to read llms.txt: %v", err)
	}
	return string(b)
}

// TestLLMs_Shape checks the llmstxt.org outline: an H1 naming the project,
// then a blockquote summary before any section.
func TestLLMs_Shape(t *testing.T) {
	lines := strings.Split(loadLLMs(t), "\n")
	if len(lines) == 0 || lines[0] != "# goingenv" {
		t.Fatalf("llms.txt must start with %q, got %q", "# goingenv", lines[0])
	}

	summary := false
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "## ") {
			break
		}
		if strings.HasPrefix(l, "> ") && len(strings.TrimSpace(l)) > 2 {
			summary = true
			break
		}
	}
	if !summary {
		t.Error("llms.txt has no blockquote summary before its first section")
	}
}

// TestLLMs_LinkedRepoFilesExist resolves every raw GitHub link in llms.txt to
// a file in this checkout, so renaming a document breaks the build here rather
// than the agent's read.
func TestLLMs_LinkedRepoFilesExist(t *testing.T) {
	matches := rawLink.FindAllStringSubmatch(loadLLMs(t), -1)
	if len(matches) == 0 {
		t.Fatal("llms.txt links no repository files; this test is not checking anything")
	}

	seen := map[string]bool{}
	for _, m := range matches {
		rel := m[1]
		if seen[rel] {
			continue
		}
		seen[rel] = true
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
			t.Errorf("llms.txt links %q, which is not in the repository: %v", rel, err)
		}
	}
	t.Logf("checked %d linked files", len(seen))
}

// TestSite_AgentPromptPointsAtLLMs ties the page's copyable agent prompt to
// the file it tells the agent to read.
func TestSite_AgentPromptPointsAtLLMs(t *testing.T) {
	doc := loadPage(t)

	var prompt string
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if label, _ := attr(n, "aria-label"); label != "Copy agent prompt" {
			return
		}
		prompt, _ = attr(n, "data-copy")
	})
	if prompt == "" {
		t.Fatal("the page has no button labelled \"Copy agent prompt\" with a data-copy value")
	}

	if !strings.Contains(prompt, llmsURL) {
		t.Errorf("the agent prompt does not point at %s", llmsURL)
	}
	if _, err := os.Stat(llmsPath); err != nil {
		t.Errorf("the agent prompt points at llms.txt, but %s is missing: %v", llmsPath, err)
	}
}
