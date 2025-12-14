// ABOUTME: Terminal UI formatting for memo output.
// ABOUTME: Uses glamour for markdown and fatih/color for styling.

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/harper/memo/internal/models"
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
		var tagNames []string
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
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback to raw content if renderer fails
		return content, nil //nolint:nilerr // Intentional fallback
	}

	out, err := renderer.Render(content)
	if err != nil {
		// Fallback to raw content if rendering fails
		return content, nil //nolint:nilerr // Intentional fallback
	}
	return out, nil
}

func FormatNoteHeader(note *models.Note, tags []*models.Tag) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", bold(note.Title)))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("ID:"), faint(note.ID.String())))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("Created:"), faint(note.CreatedAt.Format("2006-01-02 15:04"))))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("Updated:"), faint(note.UpdatedAt.Format("2006-01-02 15:04"))))

	if len(tags) > 0 {
		var tagNames []string
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
	return color.New(color.FgGreen).Sprint("✓ ") + msg
}

func Error(msg string) string {
	return color.New(color.FgRed).Sprint("✗ ") + msg
}

func FormatDirSectionHeader(dirPath string) string {
	return fmt.Sprintf("\n%s %s\n", "📁", bold(dirPath))
}

func FormatGlobalSectionHeader() string {
	return fmt.Sprintf("\n%s %s\n", "🌐", bold("Global"))
}

func FormatShowMorePrompt(count int) string {
	return faint(fmt.Sprintf("\nShow %d more notes? (y/n) ", count))
}
