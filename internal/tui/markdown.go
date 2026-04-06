package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// tuiDarkStyle is Glamour's dark theme with document margins and block padding relaxed for
// a narrow chat viewport (defaults add leading newlines and 2-column side margins).
func tuiDarkStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig
	doc := s.Document
	doc.BlockPrefix = ""
	doc.BlockSuffix = ""
	doc.Margin = nil
	s.Document = doc

	cb := s.CodeBlock
	cb.Margin = nil
	s.CodeBlock = cb

	return s
}

// newMarkdownRenderer builds a Glamour renderer for the terminal. wordWrap is the
// maximum line width for wrapped text (typically a few columns less than the viewport).
func newMarkdownRenderer(wordWrap int) (*glamour.TermRenderer, error) {
	if wordWrap < 20 {
		wordWrap = 20
	}
	// Use a fixed style, not WithAutoStyle(). Auto style calls termenv.HasDarkBackground(),
	// which queries the terminal (OSC 11); the response is written to stdin and Bubble Tea
	// would show it as input text (e.g. "]11;rgb:...").
	return glamour.NewTermRenderer(
		glamour.WithStyles(tuiDarkStyle()),
		glamour.WithWordWrap(wordWrap),
		glamour.WithEmoji(),
	)
}

func renderMarkdown(r *glamour.TermRenderer, src string) string {
	if r == nil {
		return src
	}
	out, err := r.Render(src)
	if err != nil {
		return src
	}
	out = strings.TrimRight(out, "\n")
	return strings.TrimLeft(out, "\n")
}
