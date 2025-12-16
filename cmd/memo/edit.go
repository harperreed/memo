// ABOUTME: Edit command for modifying existing notes.
// ABOUTME: Opens note content in $EDITOR for modification.

package main

import (
	"context"
	"fmt"
	"os"

	"suitesync/vault"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/sync"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id-prefix>",
	Short: "Edit a note",
	Long:  `Open a note in $EDITOR for editing.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		newContent, err := openEditor(note.Content)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		if newContent == note.Content {
			fmt.Println("No changes made.")
			return nil
		}

		note.Content = newContent
		note.Touch()

		if err := db.UpdateNote(dbConn, note); err != nil {
			return fmt.Errorf("failed to update note: %w", err)
		}

		// Get current tags for sync
		tags, err := db.GetNoteTags(dbConn, note.ID)
		if err != nil {
			return fmt.Errorf("failed to get tags: %w", err)
		}
		var tagNames []string
		for _, t := range tags {
			tagNames = append(tagNames, t.Name)
		}

		// Queue sync change
		if err := sync.TryQueueNoteChange(
			context.Background(),
			dbConn,
			note.ID,
			note.Title,
			note.Content,
			tagNames,
			note.CreatedAt,
			note.UpdatedAt,
			vault.OpUpsert,
		); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to queue sync: %v\n", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Updated note %s", note.ID.String()[:6])))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
