// Package site_test asserts structural and accessibility invariants of the
// GitHub Pages site in public/.
//
// The site has no build step -- pages.yml uploads public/ as it is -- and it
// was excluded from the CI path filter, so until these tests existed a broken
// index.html deployed to production on merge with nothing in between.
//
// The page is parsed into a DOM rather than searched with regular expressions.
// index.html carries ~480 lines of inline CSS and ~30 of inline JavaScript, so
// a text search for "<h1" or "nav" also matches selectors and strings and would
// report success while checking nothing.
package site_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// pagePath is the deployed page, relative to this test file.
const pagePath = "../../public/index.html"

// publicDir is the document root every relative link resolves against.
const publicDir = "../../public"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// loadPage parses index.html into a document tree.
func loadPage(t *testing.T) *html.Node {
	t.Helper()

	f, err := os.Open(pagePath)
	if err != nil {
		t.Fatalf("failed to open the page: %v", err)
	}
	defer func() { _ = f.Close() }()

	doc, err := html.Parse(f)
	if err != nil {
		t.Fatalf("the page is not parseable as HTML: %v", err)
	}
	return doc
}

// loadSource returns the raw file, for the few assertions whose subject is text
// rather than markup.
func loadSource(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("failed to read the page: %v", err)
	}
	return string(b)
}

// walk calls fn for every node in the tree, depth first.
func walk(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

// elements collects every element with the given tag name. Namespace is
// ignored, so this finds SVG elements as well as HTML ones.
func elements(root *html.Node, name string) []*html.Node {
	var out []*html.Node
	walk(root, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == name {
			out = append(out, n)
		}
	})
	return out
}

// attr returns an attribute's value, and whether it was present at all. The
// distinction matters: alt="" is a valid declaration that an image is
// decorative, while a missing alt is an omission.
func attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

// text returns the concatenated text content of a node.
func text(n *html.Node) string {
	var b strings.Builder
	walk(n, func(c *html.Node) {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	})
	return strings.TrimSpace(b.String())
}

// ids collects every id declared anywhere in the document, including inside the
// SVG sprite, which is where the icon symbols live.
func ids(doc *html.Node) map[string]bool {
	out := map[string]bool{}
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if id, ok := attr(n, "id"); ok && id != "" {
			out[id] = true
		}
	})
	return out
}

// inlineCSS returns the text of every <style> element joined together.
func inlineCSS(doc *html.Node) string {
	var b strings.Builder
	for _, s := range elements(doc, "style") {
		b.WriteString(text(s))
		b.WriteString("\n")
	}
	return b.String()
}

// isExternal reports whether a link target leaves the site.
func isExternal(ref string) bool {
	return strings.HasPrefix(ref, "http://") ||
		strings.HasPrefix(ref, "https://") ||
		strings.HasPrefix(ref, "//") ||
		strings.HasPrefix(ref, "mailto:") ||
		strings.HasPrefix(ref, "data:")
}

// ---------------------------------------------------------------------------
// Structure
// ---------------------------------------------------------------------------

// TestSite_SingleH1 pins the document outline. The page shipped for several
// releases with no <h1> at all: the product name was <text> inside an SVG, so
// the outline started at <h2> and no assistive technology could announce what
// the page was.
func TestSite_SingleH1(t *testing.T) {
	h1s := elements(loadPage(t), "h1")

	if len(h1s) != 1 {
		t.Fatalf("the page has %d <h1> elements, want exactly 1", len(h1s))
	}
	if text(h1s[0]) == "" {
		t.Error("the <h1> has no text content")
	}
}

// TestSite_HeadingOrderHasNoGaps checks the outline does not skip a level, which
// is how screen-reader users navigate a long page.
func TestSite_HeadingOrderHasNoGaps(t *testing.T) {
	doc := loadPage(t)

	var levels []int
	var texts []string
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
			levels = append(levels, int(n.Data[1]-'0'))
			texts = append(texts, text(n))
		}
	})

	if len(levels) < 2 {
		t.Fatalf("found %d headings; the page should have an outline", len(levels))
	}

	for i := 1; i < len(levels); i++ {
		if levels[i] > levels[i-1]+1 {
			t.Errorf("heading %q jumps from h%d to h%d, skipping a level",
				texts[i], levels[i-1], levels[i])
		}
	}
}

// TestSite_SingleNav pins the landmark count. The page once carried two copies
// of the same nav -- one hidden with opacity alone, so its links stayed in the
// tab order and every nav item was reachable twice.
func TestSite_SingleNav(t *testing.T) {
	navs := elements(loadPage(t), "nav")

	if len(navs) == 0 {
		t.Fatal("the page has no <nav> landmark")
	}

	// A header nav and a footer nav are two distinct landmarks, which is
	// correct. What must not happen is two navs with the same accessible name.
	seen := map[string]bool{}
	for _, n := range navs {
		label, ok := attr(n, "aria-label")
		if !ok || label == "" {
			t.Errorf("a <nav> has no aria-label, so its landmark cannot be told apart")
			continue
		}
		if seen[label] {
			t.Errorf("two <nav> landmarks are both labelled %q, which is the duplicated-nav bug", label)
		}
		seen[label] = true
	}
}

// ---------------------------------------------------------------------------
// Links and assets
// ---------------------------------------------------------------------------

// TestSite_FragmentLinksResolve catches the commonest way this page breaks:
// renaming a section and leaving the nav pointing at the old id. Nothing fails
// visibly -- the link just does not move the page.
//
// It covers SVG <use href="#..."> as well, where a stale reference renders
// nothing at all.
func TestSite_FragmentLinksResolve(t *testing.T) {
	doc := loadPage(t)
	declared := ids(doc)

	checked := 0
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		for _, name := range []string{"href", "xlink:href"} {
			ref, ok := attr(n, name)
			if !ok || !strings.HasPrefix(ref, "#") {
				continue
			}
			checked++
			target := strings.TrimPrefix(ref, "#")
			if !declared[target] {
				t.Errorf("<%s %s=%q> points at an id that does not exist", n.Data, name, ref)
			}
		}
	})

	if checked == 0 {
		t.Fatal("no fragment links were examined; this test is not checking anything")
	}
	t.Logf("checked %d fragment links against %d declared ids", checked, len(declared))
}

// TestSite_LocalAssetsExist checks every relative reference resolves to a file
// that is actually deployed. The icons and the social preview are referenced by
// name; deleting one leaves a broken tab icon and a link that unfurls blank.
func TestSite_LocalAssetsExist(t *testing.T) {
	doc := loadPage(t)

	checked := 0
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		for _, name := range []string{"href", "src"} {
			ref, ok := attr(n, name)
			if !ok || ref == "" || strings.HasPrefix(ref, "#") || isExternal(ref) {
				continue
			}
			checked++
			if _, err := os.Stat(filepath.Join(publicDir, ref)); err != nil {
				t.Errorf("<%s %s=%q> refers to a file that is not in public/", n.Data, name, ref)
			}
		}
	})

	if checked == 0 {
		t.Fatal("no local assets were examined; this test is not checking anything")
	}
	t.Logf("checked %d local asset references", checked)
}

// TestSite_NoRenderBlockingCDN keeps third-party JavaScript off a static page.
//
// The page used to load the Tailwind play CDN, which is a JIT compiler that
// Tailwind's own documentation says not to ship. Every script is now inline, so
// any <script src> pointing off-origin is a regression.
func TestSite_NoRenderBlockingCDN(t *testing.T) {
	doc := loadPage(t)

	for _, s := range elements(doc, "script") {
		src, ok := attr(s, "src")
		if !ok {
			continue
		}
		t.Errorf("the page loads an external script from %q; scripts must stay inline", src)
	}

	if strings.Contains(loadSource(t), "cdn.tailwindcss.com") {
		t.Error("the Tailwind play CDN is back on the page")
	}
}

// ---------------------------------------------------------------------------
// Accessibility
// ---------------------------------------------------------------------------

// TestSite_CopyControlsAreButtons pins the defect that made the page unusable
// without a mouse: the copy-to-clipboard affordances were <div>s with click
// handlers and cursor:pointer, so they advertised themselves as interactive
// while being unreachable by keyboard.
func TestSite_CopyControlsAreButtons(t *testing.T) {
	doc := loadPage(t)

	found := 0
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if _, ok := attr(n, "data-copy"); !ok {
			return
		}
		found++

		if n.Data != "button" {
			t.Errorf("a copy control is a <%s>; it must be a <button> to be focusable and operable by keyboard", n.Data)
		}
		if typ, ok := attr(n, "type"); !ok || typ != "button" {
			t.Errorf("a copy control has type=%q; without type=\"button\" it submits any enclosing form", typ)
		}
		if label, ok := attr(n, "aria-label"); !ok || strings.TrimSpace(label) == "" {
			t.Error("a copy control has no aria-label, so it is announced as an unlabelled button")
		}
	})

	if found == 0 {
		t.Fatal("no copy controls were found; this test is not checking anything")
	}
	t.Logf("checked %d copy controls", found)
}

// TestSite_GraphicsAreLabelledOrHidden checks every graphic either carries an
// accessible name or is explicitly marked decorative. An unlabelled inline SVG
// is announced as "graphic" with no further information.
func TestSite_GraphicsAreLabelledOrHidden(t *testing.T) {
	doc := loadPage(t)

	svgs := elements(doc, "svg")
	if len(svgs) == 0 {
		t.Fatal("no <svg> elements were found; this test is not checking anything")
	}

	for _, s := range svgs {
		if hidden, _ := attr(s, "aria-hidden"); hidden == "true" {
			continue
		}
		if label, ok := attr(s, "aria-label"); ok && strings.TrimSpace(label) != "" {
			continue
		}
		if len(elements(s, "title")) > 0 {
			continue
		}
		t.Errorf("an <svg> is neither aria-hidden nor given an accessible name")
	}

	// No <img> is on the page today. The rule is stated so that adding one
	// without alt text fails here rather than in review.
	for _, img := range elements(doc, "img") {
		src, _ := attr(img, "src")
		if _, ok := attr(img, "alt"); !ok {
			t.Errorf("<img src=%q> has no alt attribute", src)
		}
	}

	t.Logf("checked %d graphics", len(svgs))
}

// TestSite_FocusAndMotionRules asserts over the inline stylesheet.
//
// A string search is the right tool here: the subject is CSS, not markup. What
// it guards is real -- the page shipped once with 327 lines of CSS containing no
// :focus rule of any kind, no media query, and four infinite animations that
// ran regardless of the visitor's motion preference.
func TestSite_FocusAndMotionRules(t *testing.T) {
	css := inlineCSS(loadPage(t))
	if strings.TrimSpace(css) == "" {
		t.Fatal("the page has no inline CSS; this test is not checking anything")
	}

	required := []struct {
		needle string
		why    string
	}{
		{":focus-visible", "keyboard users cannot see what is focused"},
		{"@media", "the page has no responsive rules and will overflow on a phone"},
		{"prefers-reduced-motion", "animations run regardless of the visitor's motion preference"},
	}

	for _, r := range required {
		if !strings.Contains(css, r.needle) {
			t.Errorf("the stylesheet has no %s rule: %s", r.needle, r.why)
		}
	}

	// A fixed pixel width on the hero wordmark is what made the page overflow
	// horizontally below 408px, which includes every iPhone SE.
	if !strings.Contains(css, "max-width") {
		t.Error("the stylesheet sets no max-width anywhere, which is how the hero used to overflow narrow screens")
	}
}

// ---------------------------------------------------------------------------
// Metadata and claims
// ---------------------------------------------------------------------------

// TestSite_HeadMetadata checks the tags that decide how the page behaves when it
// is shared or saved. og-image.png was deployed but never referenced, so every
// shared link unfurled with no preview.
func TestSite_HeadMetadata(t *testing.T) {
	doc := loadPage(t)

	props := map[string]string{}
	for _, m := range elements(doc, "meta") {
		val, _ := attr(m, "content")
		if p, ok := attr(m, "property"); ok {
			props[p] = val
		}
		if n, ok := attr(m, "name"); ok {
			props[n] = val
		}
	}

	for _, key := range []string{
		"viewport", "description", "theme-color", "color-scheme",
		"og:title", "og:description", "og:image", "og:url", "twitter:card",
	} {
		if strings.TrimSpace(props[key]) == "" {
			t.Errorf("the head declares no %q", key)
		}
	}

	if _, ok := props["keywords"]; ok {
		t.Error("meta keywords is back; no search engine has used it for over a decade")
	}

	var canonical bool
	for _, l := range elements(doc, "link") {
		if rel, _ := attr(l, "rel"); rel == "canonical" {
			canonical = true
		}
	}
	if !canonical {
		t.Error("the page declares no canonical URL")
	}

	if title := elements(doc, "title"); len(title) != 1 || text(title[0]) == "" {
		t.Error("the page has no single non-empty <title>")
	}
}

// TestSite_NoStaleClaims pins the three false statements removed when the page
// was rebuilt. Each was wrong in a way a reader could not detect.
func TestSite_NoStaleClaims(t *testing.T) {
	src := loadSource(t)

	claims := []struct {
		needle string
		why    string
	}{
		{"envs.goingenv", "the tool has never produced a file by that name; archives are .goingenv/archive-<timestamp>.enc"},
		{"0.003s", "an unsourced benchmark presented as a measurement"},
		{"coming soon", "the only instance promised native Windows support, which .goreleaser.yaml does not build"},
	}

	for _, c := range claims {
		if strings.Contains(strings.ToLower(src), strings.ToLower(c.needle)) {
			t.Errorf("the page claims %q again: %s", c.needle, c.why)
		}
	}
}

// TestSite_CommandsMatchTheCLI checks the commands the page tells people to run
// are commands the binary actually has. A rename in cmd/ that misses the website
// leaves copy-paste instructions that fail.
func TestSite_CommandsMatchTheCLI(t *testing.T) {
	doc := loadPage(t)

	// The subcommands wired up in internal/cli/root.go.
	known := map[string]bool{
		"init": true, "pack": true, "unpack": true, "list": true, "status": true,
	}

	checked := 0
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		cmd, ok := attr(n, "data-copy")
		if !ok {
			return
		}
		fields := strings.Fields(cmd)
		for i, f := range fields {
			if f != "goingenv" {
				continue
			}
			checked++
			// `goingenv` alone launches the TUI, which is valid.
			if i+1 >= len(fields) {
				continue
			}
			sub := fields[i+1]
			if strings.HasPrefix(sub, "-") {
				continue
			}
			if !known[sub] {
				t.Errorf("the page offers %q, but %q is not a goingenv subcommand", cmd, sub)
			}
		}
	})

	if checked == 0 {
		t.Fatal("no goingenv commands were examined; this test is not checking anything")
	}
	t.Logf("checked %d goingenv invocations", checked)
}
