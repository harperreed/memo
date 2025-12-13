// ABOUTME: Show command for displaying a single note.
// ABOUTME: Renders markdown content with glamour.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id-prefix>",
	Short: "Show a note",
	Long:  `Display a note's full content with rendered markdown.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		tags, _ := db.GetNoteTags(dbConn, note.ID)
		attachments, _ := db.ListNoteAttachments(dbConn, note.ID)

		// Print header
		fmt.Print(ui.FormatNoteHeader(note, tags))

		// Print content
		content, _ := ui.FormatNoteContent(note.Content)
		fmt.Print(content)

		// Print attachments if any
		if len(attachments) > 0 {
			var attInfos []ui.AttachmentInfo
			for _, a := range attachments {
				attInfos = append(attInfos, ui.AttachmentInfo{
					ID:       a.ID.String(),
					Filename: a.Filename,
					MimeType: a.MimeType,
				})
			}
			fmt.Print(ui.FormatAttachmentList(attInfos))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
