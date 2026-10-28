package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

const sampleMarkdown = "# Title\n\nSome *emphasis*, **strong** and `code` with a [link](https://example.com).\n\n## List\n\n- one\n- two\n  - nested\n1. first\n- [x] done\n- [ ] open\n\n> A quote\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n```go\nfmt.Println(1)\n```\n\n---\n\n![alt text](image.png)\n"

func TestMarkdownRendersAsciiStructureWithoutBoldOrItalic(t *testing.T) {
	for _, dark := range []bool{true, false} {
		m := model{dark: dark}
		out := m.renderDocument("DESIGN.md", sampleMarkdown, 60)
		plain := ansi.Strip(out)
		requireText(t, plain, "# Title", "## List", "- one", "[x] done", "[ ] open", "| A quote", "fmt.Println(1)", "--------", "Image: alt text ->")
		for _, r := range plain {
			if r > 127 {
				t.Fatalf("renderer added a non-ASCII glyph %q:\n%s", r, plain)
			}
		}
		for _, sgr := range []string{"\x1b[1m", ";1m", "\x1b[3m", ";3m"} {
			if strings.Contains(out, sgr) {
				t.Fatalf("bold or italic in output (%q):\n%q", sgr, out)
			}
		}
		for _, line := range strings.Split(out, "\n") {
			if ansi.StringWidth(line) > 60 {
				t.Fatalf("line wider than the wrap width: %q", ansi.Strip(line))
			}
		}
	}
}

func TestDocumentControlsCannotReachTheTerminal(t *testing.T) {
	out := model{dark: true}.renderDocument("notes.md", "# Hi\x1b]0;title\a\x1b[1mBold\x1b[0m\x07", 40)
	if strings.Contains(out, "title") || strings.Contains(out, "\x07") || strings.Contains(out, "\x1b[1m") {
		t.Fatalf("control sequences leaked: %q", out)
	}
	plain := model{dark: true}.renderDocument("go.mod", "module x\n\x1b[2Jgo 1.26", 40)
	if strings.Contains(plain, "\x1b[2J") {
		t.Fatal("plain text kept a control sequence")
	}
}

func TestFrontMatterRendersAsDataAndEditorGutterStaysAligned(t *testing.T) {
	out := ansi.Strip(model{dark: true}.renderDocument("DESIGN.md", "---\nversion: alpha\nname: Calm\n---\n\n# Calm\n", 60))
	if strings.Contains(out, "## version") || strings.Contains(out, "--------") {
		t.Fatalf("front matter was parsed as Markdown:\n%s", out)
	}
	requireText(t, out, "version: alpha", "name: Calm", "# Calm")
	lines := strings.Repeat("x\n", 30)
	e, err := newEditor("a.md", lines, "", false, 40, 40, model{dark: true}.editorStyles())
	if err != nil {
		t.Fatal(err)
	}
	view := strings.Split(ansi.Strip(e.area.View()), "\n")
	if strings.Index(view[0], "x") != strings.Index(view[12], "x") {
		t.Fatalf("text shifts after line 9:\n%q\n%q", view[0], view[12])
	}
}
