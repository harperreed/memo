// ABOUTME: Terminal UI formatting for memo output.
// ABOUTME: Uses goldmark for markdown and fatih/color for styling.

package ui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/harper/memo/internal/models"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var (
	faint = color.New(color.Faint).SprintFunc()
	bold  = color.New(color.Bold).SprintFunc()
	cyan  = color.New(color.FgCyan).SprintFunc()
)

type TagCount struct {
	Name  string
	Count int
}

func FormatNoteListItem(note *models.Note, tags []*models.Tag) string {
	var sb strings.Builder

	// ID prefix and title
	idPrefix := note.ID.String()[:6]
	sb.WriteString(fmt.Sprintf("  %s  %s\n", faint(idPrefix), bold(note.Title)))

	// Tags line if present
	if len(tags) > 0 {
		tagNames := make([]string, 0, len(tags))
		for _, t := range tags {
			tagNames = append(tagNames, t.Name)
		}
		sb.WriteString(fmt.Sprintf("         %s %s\n",
			faint("Tags:"),
			cyan(strings.Join(tagNames, ", "))))
	}

	// Date
	sb.WriteString(fmt.Sprintf("         %s %s\n",
		faint("Updated:"),
		faint(note.UpdatedAt.Format("2006-01-02 15:04"))))

	return sb.String()
}

func FormatNoteContent(content string) (string, error) {
	md := goldmark.New(
		goldmark.WithRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(
					util.Prioritized(NewTerminalRenderer(), 1000),
				),
			),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		// Fallback to raw content if rendering fails
		return content, nil //nolint:nilerr // Intentional fallback
	}
	return buf.String(), nil
}

func FormatNoteHeader(note *models.Note, tags []*models.Tag) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", bold(note.Title)))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("ID:"), faint(note.ID.String())))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("Created:"), faint(note.CreatedAt.Format("2006-01-02 15:04"))))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("Updated:"), faint(note.UpdatedAt.Format("2006-01-02 15:04"))))

	if len(tags) > 0 {
		tagNames := make([]string, 0, len(tags))
		for _, t := range tags {
			tagNames = append(tagNames, t.Name)
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", faint("Tags:"), cyan(strings.Join(tagNames, ", "))))
	}

	sb.WriteString(Separator())
	return sb.String()
}

func FormatTagList(tags []TagCount) string {
	var sb strings.Builder

	for _, t := range tags {
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			cyan(t.Name),
			faint(fmt.Sprintf("(%d)", t.Count))))
	}

	return sb.String()
}

func FormatAttachmentList(attachments []AttachmentInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s\n", bold("Attachments:")))
	for _, a := range attachments {
		sb.WriteString(fmt.Sprintf("  %s  %s %s\n",
			faint(a.ID[:6]),
			a.Filename,
			faint(fmt.Sprintf("[%s]", a.MimeType))))
	}

	return sb.String()
}

type AttachmentInfo struct {
	ID       string
	Filename string
	MimeType string
}

func Separator() string {
	return faint(strings.Repeat("─", 50)) + "\n"
}

func Success(msg string) string {
	return color.New(color.FgGreen).Sprint("+ ") + msg
}

func Error(msg string) string {
	return color.New(color.FgRed).Sprint("x ") + msg
}

func FormatDirSectionHeader(dirPath string) string {
	return fmt.Sprintf("\n%s %s\n", "[D]", bold(dirPath))
}

func FormatGlobalSectionHeader() string {
	return fmt.Sprintf("\n%s %s\n", "[G]", bold("Global"))
}

func FormatShowMorePrompt(count int) string {
	return faint(fmt.Sprintf("\nShow %d more notes? (y/n) ", count))
}

// TerminalRenderer renders markdown to ANSI terminal output.
type TerminalRenderer struct{}

// NewTerminalRenderer creates a new terminal renderer.
func NewTerminalRenderer() *TerminalRenderer {
	return &TerminalRenderer{}
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *TerminalRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Block elements
	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)

	// Inline elements
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
}

func (r *TerminalRenderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n, _ := node.(*ast.Heading)
	if entering {
		// Add newline before headings (except at document start)
		if node.PreviousSibling() != nil {
			_, _ = w.WriteString("\n")
		}
		_, _ = w.WriteString(bold(strings.Repeat("#", n.Level) + " "))
	} else {
		_, _ = w.WriteString("\n")
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("\n\n")
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, _ := node.(*ast.CodeBlock)
		_, _ = w.WriteString(faint("```\n"))
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			_, _ = w.WriteString(faint(string(line.Value(source))))
		}
		_, _ = w.WriteString(faint("```\n"))
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, _ := node.(*ast.FencedCodeBlock)
		lang := string(n.Language(source))
		_, _ = w.WriteString(faint("```" + lang + "\n"))
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			_, _ = w.WriteString(faint(string(line.Value(source))))
		}
		_, _ = w.WriteString(faint("```\n"))
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(faint("> "))
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("\n")
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		parent, _ := node.Parent().(*ast.List)
		if parent.IsOrdered() {
			// Find index
			idx := 1
			for sibling := node.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
				idx++
			}
			_, _ = fmt.Fprintf(w, "  %d. ", parent.Start+idx-1)
		} else {
			_, _ = w.WriteString("  - ")
		}
	} else {
		_, _ = w.WriteString("\n")
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(faint(strings.Repeat("─", 40) + "\n"))
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, _ := node.(*ast.Text)
		_, _ = w.Write(n.Segment.Value(source))
		if n.SoftLineBreak() {
			_, _ = w.WriteString("\n")
		}
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, _ := node.(*ast.String)
		_, _ = w.Write(n.Value)
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n, _ := node.(*ast.Emphasis)
	if n.Level == 2 {
		// Bold
		if entering {
			_, _ = w.WriteString("\033[1m")
		} else {
			_, _ = w.WriteString("\033[0m")
		}
	} else {
		// Italic
		if entering {
			_, _ = w.WriteString("\033[3m")
		} else {
			_, _ = w.WriteString("\033[0m")
		}
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	_, _ = w.WriteString("`")
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n, _ := node.(*ast.Link)
	if !entering {
		_, _ = fmt.Fprintf(w, " (%s)", cyan(string(n.Destination)))
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, _ := node.(*ast.AutoLink)
		_, _ = w.WriteString(cyan(string(n.URL(source))))
	}
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n, _ := node.(*ast.Image)
	if entering {
		_, _ = fmt.Fprintf(w, "[Image: %s]", faint(string(n.Destination)))
	}
	return ast.WalkSkipChildren, nil
}

func (r *TerminalRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Skip raw HTML in terminal output
	return ast.WalkContinue, nil
}

func (r *TerminalRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Skip HTML blocks in terminal output
	return ast.WalkContinue, nil
}
