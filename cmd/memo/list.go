// ABOUTME: List command for displaying notes.
// ABOUTME: Supports filtering by tag and search queries.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	Long:  `List all notes, optionally filtered by tag or search query.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tagFlag, _ := cmd.Flags().GetString("tag")
		searchFlag, _ := cmd.Flags().GetString("search")
		limitFlag, _ := cmd.Flags().GetInt("limit")

		if searchFlag != "" {
			results, err := db.SearchNotes(dbConn, searchFlag, limitFlag)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No notes found.")
				return nil
			}

			for _, result := range results {
				tags, _ := db.GetNoteTags(dbConn, result.ID)
				fmt.Print(ui.FormatNoteListItem(result.Note, tags))
			}
			return nil
		}

		var tag *string
		if tagFlag != "" {
			tag = &tagFlag
		}

		notes, err := db.ListNotes(dbConn, tag, limitFlag)
		if err != nil {
			return fmt.Errorf("failed to list notes: %w", err)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		for _, note := range notes {
			tags, _ := db.GetNoteTags(dbConn, note.ID)
			fmt.Print(ui.FormatNoteListItem(note, tags))
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("tag", "t", "", "filter by tag")
	listCmd.Flags().StringP("search", "s", "", "search query")
	listCmd.Flags().IntP("limit", "n", 20, "number of results")
	rootCmd.AddCommand(listCmd)
}
