// ABOUTME: Remove command for deleting notes.
// ABOUTME: Includes confirmation prompt before deletion.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id-prefix>",
	Short: "Remove a note",
	Long:  `Delete a note and all its attachments.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		force, _ := cmd.Flags().GetBool("force")

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if !force {
			fmt.Printf("Delete note %q (%s)? [y/N] ", note.Title, note.ID.String()[:6])
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := db.DeleteNote(dbConn, note.ID); err != nil {
			return fmt.Errorf("failed to delete note: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Deleted note %s", note.ID.String()[:6])))
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "skip confirmation")
	rootCmd.AddCommand(rmCmd)
}
