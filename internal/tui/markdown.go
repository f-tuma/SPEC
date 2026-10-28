package tui

import (
	"path/filepath"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
)

// markdownStyle renders Markdown in the dashboard palette with regular-weight
// text and ASCII structure: headings keep their # markers, emphasis is color,
// lists use "-" and quotes use "|". Code blocks are not syntax highlighted.
func (m model) markdownStyle() ansi.StyleConfig {
	pick := func(light, dark string) *string {
		if m.dark {
			return &dark
		}
		return &light
	}
	text, muted := pick("#1F2937", "#E2E8F0"), pick("#526172", "#94A3B8")
	accent, warning := pick("#006D60", "#86E6C2"), pick("#875600", "#F4CF85")
	yes, zero, one, two := true, uint(0), uint(1), uint(2)
	quote, table, column, row := "| ", "+", "|", "-"
	heading := func(prefix string) ansi.StyleBlock {
		return ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: prefix, Color: accent}}
	}
	return ansi.StyleConfig{
		Document:              ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: text}, Margin: &zero},
		BlockQuote:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: muted}, Indent: &one, IndentToken: &quote},
		List:                  ansi.StyleList{LevelIndent: 2},
		Heading:               ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n", Color: accent}},
		H1:                    heading("# "),
		H2:                    heading("## "),
		H3:                    heading("### "),
		H4:                    heading("#### "),
		H5:                    heading("##### "),
		H6:                    heading("###### "),
		Emph:                  ansi.StylePrimitive{Color: warning},
		Strong:                ansi.StylePrimitive{Color: accent},
		Strikethrough:         ansi.StylePrimitive{CrossedOut: &yes},
		HorizontalRule:        ansi.StylePrimitive{Color: muted, Format: "\n--------\n"},
		Item:                  ansi.StylePrimitive{BlockPrefix: "- "},
		Enumeration:           ansi.StylePrimitive{BlockPrefix: ". "},
		Task:                  ansi.StyleTask{Ticked: "[x] ", Unticked: "[ ] "},
		Link:                  ansi.StylePrimitive{Color: accent, Underline: &yes},
		LinkText:              ansi.StylePrimitive{Color: accent},
		Image:                 ansi.StylePrimitive{Color: accent},
		ImageText:             ansi.StylePrimitive{Color: muted, Format: "Image: {{.text}} ->"},
		Code:                  ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: warning}},
		CodeBlock:             ansi.StyleCodeBlock{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: warning}, Margin: &two}},
		Table:                 ansi.StyleTable{CenterSeparator: &table, ColumnSeparator: &column, RowSeparator: &row},
		DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n  "},
	}
}

func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".mdx", ".markdown":
		return true
	}
	return false
}

// renderDocument renders Markdown, or shows other text files as plain text.
// Content is sanitized first so a document cannot inject terminal controls.
func (m model) renderDocument(path, content string, width int) string {
	content = strings.ReplaceAll(clean(content), "\r", "")
	if isMarkdown(path) {
		// YAML front matter (as in DESIGN.md) is data, not Markdown: show it as
		// a code block instead of letting "---" become rules and headings.
		if rest, ok := strings.CutPrefix(content, "---\n"); ok {
			if front, body, ok := strings.Cut(rest, "\n---\n"); ok {
				content = "```yaml\n" + front + "\n```\n\n" + body
			}
		}
		renderer, err := glamour.NewTermRenderer(glamour.WithStyles(m.markdownStyle()), glamour.WithWordWrap(width))
		if err == nil {
			if out, err := renderer.Render(content); err == nil {
				return strings.Trim(out, "\n")
			}
		}
	}
	return wrap(content, width)
}
